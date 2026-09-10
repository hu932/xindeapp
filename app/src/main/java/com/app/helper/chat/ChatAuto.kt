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

    /**
     * 搜图页/结果页的**专属**标题。故意不含「相似商品 / 相關商品 / 找相似」——
     * 这些词在商品页上也有（相似商品栏目），放进来自动关窗会误点商品页。
     */
    val searchSheetKeys = listOf(
        "以圖搜圖結果", "以图搜图结果", "圖片搜尋結果", "图片搜索结果", "搜圖結果", "搜图结果",
        "以圖搜圖", "以图搜图", "圖片搜尋", "图片搜索", "圖片搜索", "搜圖", "搜图",
        "image search", "shop similar"
    )
    /** 弱信号：商品页也有，只有在「标题位置 + 附近有关闭按钮」时才算搜图页 */
    val searchSheetWeakKeys = listOf(
        "相似商品", "相似产品", "相似宝贝", "相关商品", "相關商品", "找相似", "similar product"
    )
    val sendKeys = listOf("发送", "發送", "送出", "傳送", "Send", "send")
    val inputHintKeys = listOf(
        "私訊", "私信", "输入", "輸入", "留言", "說點什麼", "说点什么",
        "輸入訊息", "输入消息", "Message", "message"
    )
    val dismissKeys = listOf("取消", "关闭", "關閉", "完成", "Done", "Cancel", "Close", "close")
    val backKeys = listOf("返回", "上一頁", "上一页", "Back", "back")

    /** 聊天输入栏旁边的控件，用来确认「这确实是聊天页」 */
    val chatBarKeys = listOf(
        "相冊", "相册", "相簿", "相機", "相机", "Camera", "camera",
        "表情", "emoji", "Emoji", "貼圖", "贴图", "禮物", "礼物", "更多", "More"
    )
    /**
     * 商品页特征文案，用来判断「还停在商品页、没退回聊天」。
     * 只放商品页独有的词；像「領取」这种聊天里的优惠券卡片也有，不能放。
     */
    val productPageKeys = listOf(
        "加入購物車", "加入购物车", "直接購買", "直接购买", "立即購買", "立即购买",
        "查看配送資訊", "查看配送信息", "領取優惠券"
    )

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

        // 1) 找输入框。找不到多半是上一个任务结束后没回到聊天页（还停在商品页/搜图页）
        var input = findChatInput(root(), w, h)
        if (input == null && isProductPageVisible()) {
            log(logger, "聊一聊：当前还停在商品页，先返回聊天页再发送")
            returnToChat(logger)
            delay(600)
            input = findChatInput(root(), w, h)
        }
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
        val inputRect = Rect(ChatNodes.bounds(input))

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
        val sent = tapSend(logger, inputRect)
        if (!sent) {
            log(logger, "聊一聊：找不到发送按钮（OCR/几何也未命中），链接已填入，回退普通流程")
            return false
        }
        log(logger, "聊一聊：已发送 $link")
        delay(1200)

        // 4) 点击刚发送的那一条消息。
        //    Shopee 常把链接渲染成卡片（节点里没有原文），若只按「文本包含链接」找节点，
        //    会命中历史里的旧链接消息 → 点开的是上一个商品。所以以「输入栏正上方最新一条消息」为准，
        //    文本命中只用于「同在最新位置」时的精确优先。
        var tapped = false
        var attempts = 0
        var lastRect: Rect? = null
        repeat(8) {
            if (tapped) return@repeat
            val r = root()
            val input = findChatInput(r, w, h)
            if (input == null) {
                // 已经不在聊天页（可能上一次点击已生效）→ 不再乱点，交给下方校验
                if (attempts > 0) return@repeat
                delay(500)
                return@repeat
            }
            val inputTop = ChatNodes.bounds(input).top
            val textNode = findLinkMessage(r, w, h, link)
            val newest = newestMessageNode(r, h, inputTop)
            val target = when {
                textNode == null -> newest
                newest == null -> textNode
                // 取更靠下（=更新）的那个；文本节点不新于最新消息时才用文本节点
                ChatNodes.bounds(textNode).bottom >= ChatNodes.bounds(newest).bottom -> textNode
                else -> newest
            }
            if (target == null) {
                delay(400)
                return@repeat
            }
            val rect = ChatNodes.bounds(target)
            // 重试时必须还是同一条消息，避免页面已切换后误点到别的控件
            val prev = lastRect
            if (prev != null && Math.abs(rect.centerY() - prev.centerY()) > 60) return@repeat
            lastRect = Rect(rect)
            attempts++
            log(logger, "聊一聊：点击最新一条消息 ${rect.toShortString()}")
            tapped = performTapOnNode(target)
            if (!tapped) {
                delay(400)
                return@repeat
            }
            // 校验是否真的离开聊天页；仍停留在聊天页说明点了个“死”节点 → 再点一次
            delay(1400)
            val stillChat = root()?.let { isChatPage(it, w, h) } ?: false
            if (stillChat && attempts < 2) {
                log(logger, "聊一聊：点击后仍在聊天页，重试一次")
                tapped = false
                delay(300)
            }
        }
        if (!tapped) {
            log(logger, "聊一聊：节点未命中，尝试OCR点击最新链接")
            tapped = tapOcrLink(logger, link)
        }
        if (!tapped) {
            log(logger, "聊一聊：未能点击聊天中的链接，回退普通流程")
            return false
        }
        log(logger, "聊一聊：已点击聊天中的链接，等待环境抓取数据")
        return true
    }

    /** 发送按钮：节点文本优先，OCR兜底，最后几何点输入栏右侧(纯图标) */
    private suspend fun tapSend(logger: ((String) -> Unit)?, inputRect: Rect?): Boolean {
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
        // OCR 兜底（文字发送按钮）
        val rect = ocrFindText(sendKeys, minY = h * 45 / 100, preferBottom = true)
        if (rect != null) {
            log(logger, "聊一聊：OCR识别发送按钮并点击")
            svc()?.tap(rect.centerX(), rect.centerY())
            return true
        }
        // 几何兜底：Shopee 发送按钮常是纯图标，点输入栏最右侧
        if (inputRect != null && !inputRect.isEmpty) {
            val screenW = screen().x
            val x = screenW - 42
            if (x > 0) {
                svc()?.tap(x, inputRect.centerY())
                log(logger, "聊一聊：几何点击输入栏右侧发送图标")
                return true
            }
        }
        return false
    }

    /**
     * 找输入栏正上方、最靠下（=最新）的一条可点击消息气泡。
     * 刚发送的消息一定是最新一条；Shopee 把它渲染成卡片时节点里没有链接原文，只能靠位置判定。
     * 同一底部区域内有多个嵌套可点节点时取最内层（面积最小），避免点到整块列表容器。
     */
    private fun newestMessageNode(root: AccessibilityNodeInfo?, h: Int, inputTop: Int): AccessibilityNodeInfo? {
        val nodes = ChatNodes.collect(root)
        val screenW = screen().x
        var best: AccessibilityNodeInfo? = null
        var bestBottom = Int.MIN_VALUE
        var bestArea = Long.MAX_VALUE
        for (n in nodes) {
            if (!runCatching { n.isClickable }.getOrDefault(false)) continue
            val r = ChatNodes.bounds(n)
            if (r.isEmpty) continue
            if (r.top < 0 || r.bottom > inputTop) continue          // 必须在输入栏上方
            if (r.height() < 40 || r.height() > h * 50 / 100) continue // 排除整块消息列表容器
            if (r.width() < 40 || r.width() > screenW) continue
            val area = r.width().toLong() * r.height().toLong()
            if (best == null ||
                r.bottom > bestBottom + 24 ||
                (r.bottom >= bestBottom - 24 && area < bestArea)
            ) {
                best = n
                bestBottom = maxOf(bestBottom, r.bottom)
                bestArea = area
            }
        }
        return best
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
        val m = Regex("/product/(\\d+)/(\\d+)").find(link)
        val shop = Regex("shop_id=(\\d+)").find(link)?.groupValues?.get(1)
            ?: m?.groupValues?.get(1)
        val item = Regex("item_id=(\\d+)").find(link)?.groupValues?.get(1)
            ?: m?.groupValues?.get(2)
        // 解析不出 id 时不再退化成「包含 shopee」——否则会命中历史旧链接消息，
        // 导致点开的是上一个商品。此时返回 false，让调用方改用「最新一条消息」定位。
        if (shop.isNullOrEmpty() || item.isNullOrEmpty()) return false
        val t = text.replace(",", "").replace(" ", "")
        return t.contains(shop) && t.contains(item)
    }

    private fun performTapOnNode(node: AccessibilityNodeInfo): Boolean {
        // 优先注入真实手势点击（更接近用户操作，链接/卡片才会被打开）；
        // ACTION_CLICK 对某些包装节点会返回 true 但不产生实际效果。
        val c = ChatNodes.center(node)
        val sc = screen()
        if (c != null && c[0] > 0 && c[0] < sc.x && c[1] > 0 && c[1] < sc.y) {
            svc()?.tap(c[0], c[1])
            return true
        }
        return ChatNodes.performClick(node)
    }

    // ---------- 返回聊天（含搜图页处理） ----------
    /**
     * 上传完数据后回到聊天页。按实测流程写成确定性步骤：
     *   1) 有搜图页/中间浮层 → 先关掉（可能连着弹，最多关 3 轮）
     *   2) 返回一次
     *   3) 若返回后仍不是聊天页（例如又弹了搜图页）→ 关掉再返回，最多补 2 轮
     *
     * 注意：这里**不能**在第一次迭代就判定「已在聊天页」直接返回 ——
     * 上传完数据那一刻界面上是商品页，那个判定会误判成已回聊天，导致一次都不返回。
     */
    suspend fun returnToChat(logger: ((String) -> Unit)?) {
        if (svc() == null) return
        val (w, h) = screen().x to screen().y

        // 0) 确实已经在聊天页就不用动（严格判定，避免商品页被误判）
        if (looksLikeChatPage(w, h)) {
            log(logger, "聊一聊：已在聊天界面")
            return
        }

        // 1) 关掉搜图页 / 中间浮层（没有就立刻跳出，别白做 OCR）
        var closedAny = false
        for (i in 0 until 3) {
            if (!closeSearchSheetIfPresent(logger)) break
            closedAny = true
            delay(800)
        }
        if (closedAny) log(logger, "聊一聊：搜图页已关闭")

        // 2) 搜图页关掉后可能已经回到聊天页了
        if (looksLikeChatPage(w, h)) {
            log(logger, "聊一聊：已返回聊天界面")
            return
        }

        // 3) 返回一次
        backOnce(logger)
        delay(900)

        // 4) 校验 + 补救
        repeat(2) { i ->
            if (closeSearchSheetIfPresent(logger)) {
                log(logger, "聊一聊：又出现搜图页，关闭后再返回一次")
                delay(800)
                backOnce(logger)
                delay(900)
                return@repeat
            }
            if (looksLikeChatPage(w, h)) {
                log(logger, "聊一聊：已返回聊天界面")
                return
            }
            if (i == 0) {
                log(logger, "聊一聊：还不是聊天页，再返回一次")
                backOnce(logger)
                delay(900)
            }
        }
        log(logger, "聊一聊：返回聊天页结束")
    }

    /** 返回一步：优先点页面左上角返回键，找不到就按系统返回 */
    private suspend fun backOnce(logger: ((String) -> Unit)?) {
        val w = screen().x
        val node = findBackControl(root(), w)
        if (node != null) {
            ChatNodes.performClick(node)
            log(logger, "聊一聊：点击返回")
        } else {
            svc()?.globalBack()
            log(logger, "聊一聊：系统返回")
        }
    }

    /**
     * 严格判定「就在聊天页」。
     * 判据：屏幕上没有商品页特征、也没有搜图页标题，且底部有可编辑输入框。
     * 这样商品页（有「加入購物車/直接購買」）和搜图页都不会被误判成聊天页。
     */
    private fun looksLikeChatPage(w: Int, h: Int): Boolean {
        if (isProductPageVisible()) return false   // 商品页特征 → 不是聊天页
        if (findSearchSheetNode(w, h) != null) return false // 有搜图页 → 不是聊天页
        val r = root() ?: return false
        val input = findChatInput(r, w, h) ?: return false
        val ib = ChatNodes.bounds(input)
        if (ib.height() <= 0) return false
        if (ChatNodes.containsAny(ChatNodes.textOf(input), inputHintKeys) != null) return true
        // 输入框上下 3% 屏高范围内有发送/相册/表情等控件 → 就是聊天输入栏
        for (n in ChatNodes.collect(r)) {
            val b = ChatNodes.bounds(n)
            if (b.height() <= 0) continue
            if (b.centerY() < ib.top - h * 3 / 100 || b.centerY() > ib.bottom + h * 3 / 100) continue
            val t = ChatNodes.textOf(n)
            if (ChatNodes.containsAny(t, sendKeys) != null) return true
            if (ChatNodes.containsAny(t, chatBarKeys) != null) return true
        }
        // 没商品页特征、没搜图页，底部又有输入框 → 认定为聊天页
        return true
    }

    /** 是否还停在商品页（用于「没回到聊天页」时自救） */
    fun isProductPageVisible(): Boolean {
        for (n in ChatNodes.collect(root())) {
            if (ChatNodes.containsAny(ChatNodes.textOf(n), productPageKeys) != null) return true
        }
        return false
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
            log(logger, "聊一聊：识别到搜图页「${ChatNodes.textOf(sheetNode).trim()}」，点 × 关闭后再返回")
            dismissSheet(logger, ChatNodes.bounds(sheetNode))
            tapCloseGlyph(logger, ChatNodes.bounds(sheetNode))
            return true
        }
        // OCR 找标题
        val ocrRect = ocrFindText(searchSheetKeys, minY = (h * 12 / 100), maxY = (h * 88) / 100)
        if (ocrRect != null) {
            log(logger, "聊一聊：OCR识别到搜图页标题，点 × 关闭后再返回")
            dismissSheet(logger, ocrRect)
            tapCloseGlyph(logger, ocrRect)
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
            val strong = ChatNodes.containsAny(t, searchSheetKeys) != null
            val weak = !strong && ChatNodes.containsAny(t, searchSheetWeakKeys) != null
            if (!strong && !weak) continue
            val lower = t.lowercase()
            val resultMarker = lower.contains("結果") || lower.contains("结果") || lower.contains("result")
            val minGate = if (resultMarker) h * 15 / 1000 else h * 12 / 100
            val r = ChatNodes.bounds(n)
            if (r.top < minGate || r.bottom > h * 88 / 100) continue
            if (r.height() > h * 40 / 100) continue
            // 弱信号（「相似商品」这种）必须附近真的有关闭/取消按钮，否则很可能只是商品页栏目
            if (weak && findDismissNear(r) == null) continue
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
        val w = screen().x
        val h = screen().y
        // 1) 标题行附近的 取消/完成/關閉
        val dismissNode = findDismissNear(titleRect)
        if (dismissNode != null) {
            ChatNodes.performClick(dismissNode)
            log(logger, "聊一聊：点击搜图页关闭/取消按钮")
            return
        }
        // 2) 标题行最右侧 ×：不论靠顶整页还是半屏层都点（标题行中线、右缘内 36）
        val rowY = titleRect.centerY()
        if (rowY > 0) {
            svc()?.tap(w - 36, rowY)
            log(logger, "聊一聊：点击搜图页标题行右侧 ×(x=${w - 36},y=$rowY)")
            return
        }
        // 3) 半屏层兜底：点 sheet 上方暗区
        val titleNearTop = titleRect.top < h * 22 / 100
        if (!titleNearTop) {
            val sheetTop = (titleRect.top - 30).coerceAtLeast(0)
            val tapY = (sheetTop / 2).coerceAtLeast(60).coerceAtMost(h * 40 / 100)
            svc()?.tap(w / 2, tapY)
            log(logger, "聊一聊：点击搜图页上方暗区关闭")
        }
    }

    // OCR 全屏找关闭符（×/✕/X/关闭/取消）并点击（标题行附近优先）
    private suspend fun tapCloseGlyph(logger: ((String) -> Unit)?, titleRect: Rect?) {
        val items = ocrItems()
        val tokens = listOf("×", "✕", "✖", "X", "x", "close", "Close", "取消", "关闭", "關閉", "cancel", "done", "Done")
        val h = screen().y
        var best: OcrItem? = null
        var bestDelta = Int.MAX_VALUE
        for (it in items) {
            val t = it.text.trim()
            if (t.isEmpty() || t.length > 12) continue
            val match = tokens.any { tok -> t == tok || t.equals(tok, ignoreCase = true) }
            if (!match) continue
            val r = it.bounds
            val near: Boolean
            val delta: Int
            if (titleRect != null && !titleRect.isEmpty) {
                val d = r.centerY() - titleRect.centerY()
                delta = if (d < 0) -d else d
                near = delta <= 260
            } else {
                delta = r.top
                near = r.top <= h * 40 / 100
            }
            if (!near) continue
            if (delta < bestDelta) {
                bestDelta = delta
                best = it
            }
        }
        best?.let {
            svc()?.tap(it.bounds.centerX(), it.bounds.centerY())
            log(logger, "聊一聊：OCR点击关闭符「${it.text}」")
        }
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
}
