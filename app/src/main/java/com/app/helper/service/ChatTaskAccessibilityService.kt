package com.app.helper.service

import android.accessibilityservice.AccessibilityService
import android.accessibilityservice.AccessibilityServiceInfo
import android.accessibilityservice.GestureDescription
import android.content.BroadcastReceiver
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.graphics.Path
import android.graphics.Rect
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.util.Log
import android.view.accessibility.AccessibilityEvent
import android.view.accessibility.AccessibilityNodeInfo
import android.widget.Toast

class ChatTaskAccessibilityService : AccessibilityService() {

    companion object {
        private const val TAG = "ChatTaskAccessibility"
        const val ACTION_OPEN_TASK_URL = "com.app.helper.OPEN_TASK_URL_IN_CHAT"
        const val ACTION_RETURN_TO_CHAT = "com.app.helper.RETURN_TO_CHAT"
        const val ACTION_PRODUCT_BLOCKED = "com.app.helper.PRODUCT_BLOCKED"
        const val EXTRA_URL = "url"

        @Volatile
        var isEnabled = false
            private set
    }

    private val handler = Handler(Looper.getMainLooper())
    private val runPendingUrlRunnable = Runnable { runPendingUrl() }
    private val linkAttemptRunnables = mutableListOf<Runnable>()
    private var receiver: BroadcastReceiver? = null
    private var pendingUrl: String? = null
    private var returnBackUsed = false
    private var lastPopupCloseAt = 0L

    override fun onServiceConnected() {
        super.onServiceConnected()
        isEnabled = true
        serviceInfo = serviceInfo.apply {
            eventTypes = AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED or
                AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED or
                AccessibilityEvent.TYPE_VIEW_FOCUSED
            feedbackType = AccessibilityServiceInfo.FEEDBACK_GENERIC
            flags = flags or
                AccessibilityServiceInfo.FLAG_REPORT_VIEW_IDS or
                AccessibilityServiceInfo.FLAG_RETRIEVE_INTERACTIVE_WINDOWS
            notificationTimeout = 80L
        }
        registerCommandReceiver()
        Toast.makeText(this, "聊天任务辅助已开启", Toast.LENGTH_SHORT).show()
    }

    override fun onDestroy() {
        super.onDestroy()
        isEnabled = false
        receiver?.let { runCatching { unregisterReceiver(it) } }
        receiver = null
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        if (pendingUrl != null) {
            handler.removeCallbacks(runPendingUrlRunnable)
            handler.postDelayed(runPendingUrlRunnable, 250L)
            return
        }

        val root = activeContentRoot() ?: return
        if (containsAnyText(root, productUnavailableHints)) {
            closeBlockingPopup(root)
            sendBroadcast(Intent(ACTION_PRODUCT_BLOCKED))
            handler.postDelayed({ returnToChatScreen() }, 1_400L)
            return
        }

        if (hasShopeeRecommendPopup(root)) {
            closeBlockingPopup(root)
        }
    }

    override fun onInterrupt() = Unit

    private fun registerCommandReceiver() {
        receiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                when (intent.action) {
                    ACTION_OPEN_TASK_URL -> {
                        val url = intent.getStringExtra(EXTRA_URL)?.trim().orEmpty()
                        if (url.isBlank()) return
                        cancelLinkAttempts()
                        pendingUrl = url
                        handler.post(runPendingUrlRunnable)
                    }
                    ACTION_RETURN_TO_CHAT -> {
                        cancelLinkAttempts()
                        handler.post { returnToChatScreen() }
                    }
                }
            }
        }
        val filter = IntentFilter().apply {
            addAction(ACTION_OPEN_TASK_URL)
            addAction(ACTION_RETURN_TO_CHAT)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(receiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(receiver, filter)
        }
    }

    private fun runPendingUrl() {
        val url = pendingUrl ?: return
        val root = activeContentRoot() ?: return
        val input = findEditable(root)
        if (input == null) {
            Log.w(TAG, "No editable input found")
            return
        }

        if (!input.performAction(AccessibilityNodeInfo.ACTION_FOCUS)) {
            input.performAction(AccessibilityNodeInfo.ACTION_CLICK)
        }

        val filled = setNodeText(input, url) || pasteText(input, url)
        if (!filled) {
            Log.w(TAG, "Unable to fill chat input")
            return
        }

        pendingUrl = null
        handler.postDelayed({ clickSendOrEnter(url) }, 350L)
        scheduleLinkOpenAttempts(url)
    }

    private fun setNodeText(node: AccessibilityNodeInfo, text: String): Boolean {
        val args = Bundle().apply {
            putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, text)
        }
        return node.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, args)
    }

    private fun pasteText(node: AccessibilityNodeInfo, text: String): Boolean {
        val clipboard = getSystemService(CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("task_url", text))
        node.performAction(AccessibilityNodeInfo.ACTION_FOCUS)
        return node.performAction(AccessibilityNodeInfo.ACTION_PASTE)
    }

    private fun clickSendOrEnter(url: String) {
        val root = activeContentRoot() ?: return
        val sendNode = findSendButton(root)
        if (sendNode != null && clickNode(sendNode)) {
            return
        }
    }

    private fun scheduleLinkOpenAttempts(url: String) {
        cancelLinkAttempts()
        listOf(1800L, 2600L, 3600L, 5000L).forEach { delayMs ->
            val runnable = Runnable {
                if (clickVisibleTaskLink(url)) cancelLinkAttempts()
            }
            linkAttemptRunnables.add(runnable)
            handler.postDelayed(runnable, delayMs)
        }
    }

    private fun cancelLinkAttempts() {
        linkAttemptRunnables.forEach { handler.removeCallbacks(it) }
        linkAttemptRunnables.clear()
    }

    private fun clickVisibleTaskLink(url: String): Boolean {
        val root = activeContentRoot() ?: return false
        if (!isProbablyChatScreen(root)) return false
        val node = findProductLinkNode(root, url) ?: return false
        return clickNode(node) || clickNodeCenter(node)
    }

    private fun findEditable(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        if (node.isEditable && node.isEnabled) return node
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findEditable(child)?.let { return it }
        }
        return null
    }

    private fun findSendButton(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        val label = (node.text ?: node.contentDescription)?.toString()?.trim().orEmpty()
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        if (node.isEnabled && (node.isClickable || node.isFocusable)) {
            if (label in sendLabels || sendIdHints.any { viewId.contains(it) }) return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findSendButton(child)?.let { return it }
        }
        return null
    }

    private fun findUrlNode(node: AccessibilityNodeInfo, needle: String): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        val text = buildString {
            node.text?.let { append(it) }
            node.contentDescription?.let { append(' ').append(it) }
        }
        if (text.contains(needle, ignoreCase = true) && (node.isClickable || node.parent != null)) {
            return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findUrlNode(child, needle)?.let { return it }
        }
        return null
    }

    private fun urlNeedles(url: String): List<String> {
        val parts = url.trim().trimEnd('/').split('/').filter { it.isNotBlank() }
        val itemId = parts.lastOrNull().orEmpty()
        val shopId = parts.dropLast(1).lastOrNull().orEmpty()
        return listOf(url, "shopee.tw", "/product/", itemId, shopId)
            .filter { it.length >= 4 }
            .distinct()
    }

    private fun findProductLinkNode(root: AccessibilityNodeInfo, url: String): AccessibilityNodeInfo? {
        val needles = urlNeedles(url)
        needles.take(3).firstNotNullOfOrNull { findUrlNode(root, it) }?.let { return it }

        val idNeedles = needles.drop(3)
        if (idNeedles.isEmpty() || findUrlNode(root, "shopee") == null) return null
        return idNeedles.firstNotNullOfOrNull { findUrlNode(root, it) }
    }

    private fun clickNode(node: AccessibilityNodeInfo): Boolean {
        var current: AccessibilityNodeInfo? = node
        repeat(6) {
            val target = current ?: return false
            if (target.isClickable && target.isEnabled) {
                return target.performAction(AccessibilityNodeInfo.ACTION_CLICK)
            }
            current = target.parent
        }
        return false
    }

    private fun clickNodeCenter(node: AccessibilityNodeInfo): Boolean {
        val rect = Rect()
        node.getBoundsInScreen(rect)
        if (rect.width() <= 0 || rect.height() <= 0) return false
        return tapPoint(rect.centerX().toFloat(), rect.centerY().toFloat())
    }

    private fun tapPoint(x: Float, y: Float): Boolean {
        val path = Path().apply { moveTo(x, y) }
        val gesture = GestureDescription.Builder()
            .addStroke(GestureDescription.StrokeDescription(path, 0L, 80L))
            .build()
        return dispatchGesture(gesture, null, null)
    }

    private fun returnToChatScreen() {
        returnBackUsed = false
        val root = activeContentRoot()
        if (root != null && isProbablyChatScreen(root)) return

        if (root != null && closeBlockingPopup(root)) {
            handler.postDelayed({ singleBackToChatIfNeeded() }, 1_400L)
            return
        }

        singleBackToChatIfNeeded()
        handler.postDelayed({ closeDialogOnlyIfStillBlocking() }, 1_400L)
    }

    private fun singleBackToChatIfNeeded() {
        if (returnBackUsed) return
        val root = activeContentRoot()
        if (root != null && isProbablyChatScreen(root)) {
            return
        }
        returnBackUsed = true
        performGlobalAction(GLOBAL_ACTION_BACK)
    }

    private fun closeDialogOnlyIfStillBlocking() {
        val root = activeContentRoot() ?: return
        if (!isProbablyChatScreen(root)) closeBlockingPopup(root)
    }

    private fun activeContentRoot(): AccessibilityNodeInfo? {
        windows
            ?.asSequence()
            ?.mapNotNull { it.root }
            ?.firstOrNull { !isOwnAppNode(it) }
            ?.let { return it }
        return rootInActiveWindow?.takeUnless { isOwnAppNode(it) }
    }

    private fun isProbablyChatScreen(root: AccessibilityNodeInfo): Boolean {
        if (findEditable(root) != null && findSendButton(root) != null) return true
        return findEditable(root) != null && containsAnyText(root, chatHints)
    }

    private fun closeDialogButton(node: AccessibilityNodeInfo): Boolean {
        val label = (node.text ?: node.contentDescription)?.toString()?.trim().orEmpty()
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        if (node.isEnabled && (node.isClickable || node.isFocusable)) {
            if (label in dialogCloseLabels || dialogCloseIdHints.any { viewId.contains(it) }) {
                return clickNode(node) || clickNodeCenter(node)
            }
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            if (closeDialogButton(child)) return true
        }
        return false
    }

    private fun closeBlockingPopup(root: AccessibilityNodeInfo): Boolean {
        val now = System.currentTimeMillis()
        if (now - lastPopupCloseAt < 900L) return true

        if (closeDialogButton(root)) return true
        val hasKnownPopup = containsAnyText(root, productUnavailableHints) ||
            hasShopeeRecommendPopup(root)
        if (!hasKnownPopup) return false

        val closed = closeTopRightIconButton(root) || tapShopeePopupClosePoint()
        if (closed) lastPopupCloseAt = now
        return closed
    }

    private fun hasShopeeRecommendPopup(root: AccessibilityNodeInfo): Boolean {
        val hasMallTab = containsAnyText(root, shopeeMallHints)
        val hasPaymentTab = containsAnyText(root, shopeePaymentHints)
        return hasMallTab && hasPaymentTab
    }

    private fun closeTopRightIconButton(node: AccessibilityNodeInfo): Boolean {
        val display = resources.displayMetrics
        val screenWidth = display.widthPixels
        val screenHeight = display.heightPixels
        findTopRightIconButton(node, screenWidth, screenHeight)?.let {
            return clickNode(it) || clickNodeCenter(it)
        }
        return false
    }

    private fun tapShopeePopupClosePoint(): Boolean {
        val display = resources.displayMetrics
        val x = display.widthPixels * 0.855f
        val y = display.heightPixels * 0.16f
        return tapPoint(x, y)
    }

    private fun findTopRightIconButton(
        node: AccessibilityNodeInfo,
        screenWidth: Int,
        screenHeight: Int
    ): AccessibilityNodeInfo? {
        val rect = Rect()
        node.getBoundsInScreen(rect)
        val label = (node.text ?: node.contentDescription)?.toString()?.trim().orEmpty()
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        val smallIcon = rect.width() in 18..96 && rect.height() in 18..96
        val minX = (screenWidth * 0.55f).toInt()
        val minY = (screenHeight * 0.05f).toInt()
        val maxY = (screenHeight * 0.32f).toInt()
        val inPopupCloseArea = rect.centerX() > minX &&
            rect.centerY() in minY..maxY
        if (node.isEnabled && (node.isClickable || node.isFocusable) &&
            smallIcon && inPopupCloseArea &&
            (label.isBlank() || label in dialogCloseLabels || viewId.contains("close"))
        ) {
            return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findTopRightIconButton(child, screenWidth, screenHeight)?.let { return it }
        }
        return null
    }

    private fun containsAnyText(node: AccessibilityNodeInfo, needles: Set<String>): Boolean {
        if (isOwnAppNode(node)) return false
        val text = buildString {
            node.text?.let { append(it) }
            node.contentDescription?.let { append(' ').append(it) }
        }
        if (needles.any { text.contains(it, ignoreCase = true) }) return true
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            if (containsAnyText(child, needles)) return true
        }
        return false
    }

    private fun isOwnAppNode(node: AccessibilityNodeInfo): Boolean {
        return node.packageName?.toString() == packageName
    }

    private val sendLabels = setOf(
        "发送", "送出", "傳送", "传送", "Send", "send", "Enter", "enter", "Go", "go", "进入", "前往"
    )

    private val sendIdHints = listOf("send", "submit", "enter", "go")

    private val dialogCloseLabels = setOf(
        "关闭", "關閉", "取消", "稍后", "稍後", "知道了", "我知道了", "确定", "確認",
        "OK", "Ok", "ok", "Close", "close", "Cancel", "cancel", "Later", "later",
        "Not now", "not now", "No thanks", "no thanks", "×", "X", "x"
    )

    private val dialogCloseIdHints = listOf("close", "cancel", "dismiss")

    private val chatHints = setOf(
        "发送", "傳送", "输入", "輸入", "消息", "訊息", "message", "Message", "chat", "Chat"
    )

    private val productUnavailableHints = setOf(
        "商品不存在", "商品不存在或已下架", "商品已下架", "商品未找到", "找不到商品",
        "產品不存在", "產品已下架", "賣場不存在", "商品不存在或已被移除",
        "Product not found", "product not found", "product does not exist",
        "This product does not exist", "unavailable", "Unavailable"
    )

    private val shopeeMallHints = setOf("蝦皮商城", "虾皮商城")

    private val shopeePaymentHints = setOf("貨到付款", "货到付款")
}
