package com.app.helper.chat

import android.graphics.Point
import android.graphics.Rect
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import com.app.helper.service.ChatAccessibilityService
import kotlinx.coroutines.delay

/**
 * 聊一聊编排（Android 版，对应 iOS Tweak 的 startChatFlow/goBackToChat）：
 *  - openTaskLinkInChat：把商品链接填进聊天输入框→发送→点击聊天里的链接（打开商品页，环境App照常抓包）
 *  - returnToChat：识别「搜图页/中间浮层」→先关闭→再逐级返回直到回到聊天页
 * 节点树(无障碍)优先，像素 OCR(ML Kit) 作 WebView/无节点场景兜底。
 */
object ChatAuto {

    private const val TAG = "ChatAuto"

    val searchSheetKeys = listOf(
        "以圖搜圖", "以图搜图", "搜圖", "搜图", "圖片搜尋", "图片搜索", "圖片搜索",
        "相似商品", "相似产品", "相似宝贝", "相关商品", "相關商品", "找相似",
        "image search", "similar product", "shop similar"
    )
    val sendKeys = listOf("发送", "發送", "送出", "傳送", "Send", "send")
    val inputHintKeys = listOf(
        "私訊", "私信", "输入", "輸入", "留言", "說點什麼", "说点什么",
        "輸入訊息", "输入消息", "Message", "message"
    )
    val dismissKeys = listOf("取消", "关闭", "關閉", "完成", "Done", "Cancel", "Close", "close")
    val backKeys = listOf("返回", "上一頁", "上一页", "Back", "back")

    // ---------- 基础 ----------
    private fun svc(): ChatAccessibilityService? = ChatAccessibilityService.current()

    private fun screen(): Point = svc()?.screenSize() ?: Point(1080, 2400)

    private fun root(): AccessibilityNodeInfo? = svc()?.activeShopeeRoot()

    private fun canOcr(): Boolean {
        if (!OcrUtil.available()) return false
        return svc()?.screenshotSupported() == true
    }

    private fun log(logger: ((String) -> Unit)?, m: String) {
        android.util.Log.i(TAG, m)
        logger?.invoke(m)
    }

    // ---------- 找输入框 ----------
    fun findChatInput(root: AccessibilityNodeInfo?, w: Int, h: Int): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root)
        var best: AccessibilityNodeInfo? = null
        var bestBottom = Int.MIN_VALUE
        for (n in nodes) {
            if (!ChatNodes.isEditableCandidate(n)) continue
            val r = ChatNodes.bounds(n)
            if (r.height() <= 0) continue
            // 聊天输入框通常在屏幕下部
            if (r.bottom < h * 55 / 100) continue
            if (r.bottom > bestBottom) {
                bestBottom = r.bottom
                best = n
            }
        }
        return best
    }

    fun isChatPage(root: AccessibilityNodeInfo?, w: Int, h: Int): Boolean {
        return findChatInput(root, w, h) != null
    }

    // ---------- 主流程：发链接到聊天并点击 ----------
    suspend fun openTaskLinkInChat(logger: ((String) -> Unit)?, link: String): Boolean {
        if (svc() == null) {
            log(logger, "聊一聊：无障碍服务未开启，请先在设置里开启【Ajie 聊一聊助手】")
            return false
        }
        if (!svc()!!.isShopeeForeground()) {
            log(logger, "聊一聊：Shopee 不在前台，请先打开 Shopee 并进入聊天页")
            return false
        }
        if (link.isBlank()) {
            log(logger, "聊一聊：无商品链接")
            return false
        }
        val (w, h) = screen().x to screen().y

        // 1) 找输入框；找不到则 OCR 点一下输入区再重扫
        var input = findChatInput(root(), w, h)
        if (input == null) {
            log(logger, "聊一聊：未找到输入框，尝试OCR定位输入区")
            if (tapOcrInputArea(logger)) {
                delay(500)
                input = findChatInput(root(), w, h)
            }
        }
        if (input == null) {
            log(logger, "聊一聊：未找到聊天输入框，需先在Shopee打开聊天会话；回退普通流程")
            return false
        }

        // 2) 聚焦并填入链接
        log(logger, "聊一聊：填入链接")
        runCatching { input.performAction(AccessibilityNodeInfo.ACTION_FOCUS) }
        val args = Bundle().apply {
            putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, link)
        }
        val setOk = runCatching { input.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, args) }
            .getOrDefault(false)
        if (!setOk) {
            log(logger, "聊一聊：写入输入框失败，回退普通流程")
            return false
        }
        delay(350)

        // 3) 发送
        val sent = tapSend(logger)
        if (!sent) {
            log(logger, "聊一聊：找不到发送按钮（OCR也未命中），链接已填入，回退普通流程")
            return false
        }
        log(logger, "聊一聊：已发送 $link")
        delay(1200)

        // 4) 找刚发的链接消息并点击（等待消息渲染，最多 4s）
        var tapped = false
        repeat(8) {
            if (tapped) return@repeat
            val linkNode = findLinkMessage(root(), w, h, link)
            if (linkNode != null) {
                tapped = performTapOnNode(linkNode)
            } else {
                delay(500)
            }
        }
        if (!tapped) {
            log(logger, "聊一聊：未在聊天中找到链接消息，尝试OCR点击链接")
            tapped = tapOcrLink(logger, link)
        }
        if (!tapped) {
            log(logger, "聊一聊：未能点击聊天中的链接，回退普通流程")
            return false
        }
        log(logger, "聊一聊：已点击聊天中的链接，等待环境抓取数据")
        return true
    }

    /** 发送按钮：节点文本优先，OCR兜底 */
    private suspend fun tapSend(logger: ((String) -> Unit)?): Boolean {
        val (w, h) = screen().x to screen().y
        val nodes = ChatNodes.collect(root())
        var btn: AccessibilityNodeInfo? = null
        var bestBottom = Int.MIN_VALUE
        for (n in nodes) {
            if (!runCatching { n.isClickable }.getOrDefault(false)) continue
            val t = ChatNodes.textOf(n)
            if (ChatNodes.containsAny(t, sendKeys) == null) continue
            val r = ChatNodes.bounds(n)
            if (r.bottom < h / 2) continue
            if (r.bottom > bestBottom) {
                bestBottom = r.bottom
                btn = n
            }
        }
        if (btn != null) {
            log(logger, "聊一聊：点击发送按钮")
            return ChatNodes.performClick(btn)
        }
        // OCR 兜底
        val rect = ocrFindText(sendKeys, minY = h * 45 / 100, preferBottom = true) ?: return false
        log(logger, "聊一聊：OCR识别发送按钮并点击")
        svc()?.tap(rect.centerX(), rect.centerY())
        return true
    }

    /** 找聊天里刚发送的链接消息节点（取最靠下、且在输入框上方） */
    private fun findLinkMessage(root: AccessibilityNodeInfo?, w: Int, h: Int, link: String): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root)
        val input = findChatInput(root, w, h)
        val inputTop = if (input != null) ChatNodes.bounds(input).top else h
        var best: AccessibilityNodeInfo? = null
        var bestBottom = Int.MIN_VALUE
        for (n in nodes) {
            val t = ChatNodes.textOf(n)
            if (!t.contains(link) && !linkMatchesDigits(t, link)) continue
            val r = ChatNodes.bounds(n)
            if (r.bottom > inputTop) continue // 排除输入栏
            if (r.height() <= 0) continue
            if (r.bottom > bestBottom) {
                bestBottom = r.bottom
                best = n
            }
        }
        return best
    }

    private fun linkMatchesDigits(text: String, link: String): Boolean {
        val item = Regex("item_id=(\\d+)").find(link)?.groupValues?.get(1)
        val shop = Regex("shop_id=(\\d+)").find(link)?.groupValues?.get(1)
            ?: Regex("/product/(\\d+)/(\\d+)").find(link)?.groupValues?.get(1)
        if (item.isNullOrEmpty() || shop.isNullOrEmpty()) return text.contains("shopee", true)
        return text.contains(shop) && text.contains(item)
    }

    private fun performTapOnNode(node: AccessibilityNodeInfo): Boolean {
        if (ChatNodes.performClick(node)) return true
        val c = ChatNodes.center(node) ?: return false
        svc()?.tap(c[0], c[1])
        return true
    }

    // ---------- 返回聊天（含搜图页处理） ----------
    suspend fun returnToChat(logger: ((String) -> Unit)?) {
        if (svc() == null) return
        val (w, h) = screen().x to screen().y
        repeat(8) { i ->
            val r = root() ?: run { delay(500); return@repeat }
            // 1) 先识别并关闭搜图页/中间浮层
            if (closeSearchSheetIfPresent(logger)) {
                delay(700)
                return@repeat
            }
            // 2) 已在聊天页 → 若刚做过动作，OCR 确认一次无残留浮层后结束
            if (isChatPage(r, w, h)) {
                if (i == 0) {
                    log(logger, "聊一聊：已返回聊天界面")
                    return
                }
                if (ocrConfirmNoSheet(logger)) {
                    log(logger, "聊一聊：已返回聊天界面")
                    return
                }
                delay(500)
                return@repeat
            }
            // 3) 做一步返回（找返回控件 → 系统返回）
            val backNode = findBackControl(r, w)
            if (backNode != null) {
                ChatNodes.performClick(backNode)
                log(logger, "聊一聊：点击返回")
            } else {
                svc()?.globalBack()
                log(logger, "聊一聊：系统返回")
            }
            delay(650)
        }
    }

    private fun findBackControl(root: AccessibilityNodeInfo?, w: Int): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root)
        for (n in nodes) {
            if (!runCatching { n.isClickable }.getOrDefault(false)) continue
            val t = ChatNodes.textOf(n)
            if (ChatNodes.containsAny(t, backKeys) == null) continue
            val r = ChatNodes.bounds(n)
            // 返回键通常在左上角
            if (r.centerX() < w * 3 / 10) return n
        }
        return null
    }

    // ---------- 搜图页识别与关闭 ----------
    private suspend fun closeSearchSheetIfPresent(logger: ((String) -> Unit)?): Boolean {
        val (w, h) = screen().x to screen().y
        // 视图文本找标题
        val sheetNode = findSearchSheetNode(w, h)
        if (sheetNode != null) {
            log(logger, "聊一聊：识别到搜图页「${ChatNodes.textOf(sheetNode).trim()}」，先关闭再返回")
            dismissSheet(logger, ChatNodes.bounds(sheetNode))
            return true
        }
        // OCR 找标题
        val ocrRect = ocrFindText(searchSheetKeys, minY = (h * 0.12).toInt(), maxY = (h * 0.86).toInt())
        if (ocrRect != null) {
            log(logger, "聊一聊：OCR识别到搜图页标题，先关闭再返回")
            dismissSheet(logger, ocrRect)
            return true
        }
        return false
    }

    private fun findSearchSheetNode(w: Int, h: Int): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root())
        var best: AccessibilityNodeInfo? = null
        var bestScore = 0.0
        for (n in nodes) {
            val t = ChatNodes.textOf(n)
            if (t.length > 80) continue
            if (ChatNodes.containsAny(t, searchSheetKeys) == null) continue
            val r = ChatNodes.bounds(n)
            if (r.top < h * 12 / 100 || r.bottom > h * 86 / 100) continue
            if (r.height() > h * 35 / 100) continue
            val delta = Math.abs(r.centerY() - h * 0.4)
            val score = 1.0 / (1.0 + delta)
            if (score > bestScore) {
                bestScore = score
                best = n
            }
        }
        return best
    }

    private suspend fun dismissSheet(logger: ((String) -> Unit)?, titleRect: Rect) {
        val (w, h) = screen().x to screen().y
        // 1) 标题行附近的 取消/完成/關閉
        val dismissNode = findDismissNear(titleRect)
        if (dismissNode != null) {
            ChatNodes.performClick(dismissNode)
            log(logger, "聊一聊：点击搜图页关闭/取消按钮")
            return
        }
        // 2) 标题较靠顶 → 右上 X；标题在中部(bottom sheet) → 点 sheet 上方暗区
        val titleNearTop = titleRect.top < h * 22 / 100
        if (titleNearTop) {
            val x = w - 48
            val y = (titleRect.top / 2).coerceAtLeast(40)
            svc()?.tap(x, y)
            log(logger, "聊一聊：点击搜图页右上角关闭")
            return
        }
        val sheetTop = (titleRect.top - 30).coerceAtLeast(0)
        val tapY = (sheetTop / 2).coerceAtLeast(60).coerceAtMost(h * 40 / 100)
        svc()?.tap(w / 2, tapY)
        log(logger, "聊一聊：点击搜图页上方暗区关闭")
    }

    private fun findDismissNear(titleRect: Rect): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root())
        for (n in nodes) {
            if (!runCatching { n.isClickable }.getOrDefault(false)) continue
            val t = ChatNodes.textOf(n)
            if (ChatNodes.containsAny(t, dismissKeys) == null) continue
            val r = ChatNodes.bounds(n)
            val near = r.centerY() >= titleRect.top - 100 && r.centerY() <= titleRect.top + 160
            if (near) return n
        }
        return null
    }

    // ---------- OCR 工具 ----------
    /** 屏幕取像素 OCR 结果（坐标换算成屏幕 px） */
    private suspend fun ocrItems(): List<OcrItem> {
        if (!canOcr()) return emptyList()
        val bmp = svc()?.captureScreenshot() ?: return emptyList()
        val items = OcrUtil.recognize(bmp)
        val (w, h) = screen().x to screen().y
        val sw = if (bmp.width > 0) w.toFloat() / bmp.width else 1f
        val sh = if (bmp.height > 0) h.toFloat() / bmp.height else 1f
        return items.map {
            val r = it.bounds
            OcrItem(
                it.text,
                Rect((r.left * sw).toInt(), (r.top * sh).toInt(), (r.right * sw).toInt(), (r.bottom * sh).toInt())
            )
        }
    }

    private suspend fun ocrFindText(
        keys: List<String>,
        minY: Int = 0,
        maxY: Int = Int.MAX_VALUE,
        preferBottom: Boolean = false
    ): Rect? {
        val items = ocrItems()
        var best: Rect? = null
        var bestKey = if (preferBottom) Int.MIN_VALUE else Int.MAX_VALUE
        for (it in items) {
            if (ChatNodes.containsAny(it.text, keys) == null) continue
            val r = it.bounds
            if (r.bottom < minY || r.top > maxY) continue
            val key = if (preferBottom) r.bottom else r.top
            if (preferBottom) {
                if (key > bestKey) {
                    bestKey = key
                    best = r
                }
            } else {
                if (key < bestKey) {
                    bestKey = key
                    best = r
                }
            }
        }
        return best
    }

    private suspend fun tapOcrInputArea(logger: ((String) -> Unit)?): Boolean {
        val h = screen().y
        val rect = ocrFindText(inputHintKeys, minY = (h * 0.3).toInt(), preferBottom = true) ?: return false
        svc()?.tap(rect.centerX(), rect.centerY())
        log(logger, "聊一聊：OCR定位输入区并点击")
        return true
    }

    private suspend fun tapOcrLink(logger: ((String) -> Unit)?, link: String): Boolean {
        val h = screen().y
        val items = ocrItems()
        var best: Rect? = null
        var bestBottom = Int.MIN_VALUE
        for (it in items) {
            if (!(it.text.contains(link) || linkMatchesDigits(it.text, link))) continue
            val r = it.bounds
            if (r.bottom > h * 88 / 100) continue
            if (r.bottom > bestBottom) {
                bestBottom = r.bottom
                best = r
            }
        }
        if (best != null) {
            svc()?.tap(best.centerX(), best.centerY())
            log(logger, "聊一聊：OCR点击聊天中的链接")
            return true
        }
        return false
    }

    private suspend fun ocrConfirmNoSheet(logger: ((String) -> Unit)?): Boolean {
        val h = screen().y
        val sheet = ocrFindText(searchSheetKeys, minY = (h * 0.12).toInt(), maxY = (h * 0.86).toInt())
        if (sheet != null) {
            log(logger, "聊一聊：OCR仍检测到搜图页，继续关闭")
            dismissSheet(logger, sheet)
            return false
        }
        return true
    }
}
