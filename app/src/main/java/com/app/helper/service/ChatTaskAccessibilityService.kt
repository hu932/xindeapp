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
        private const val MAX_RETURN_ATTEMPTS = 8
        private const val RETURN_STEP_DELAY_MS = 900L
        private const val POPUP_SETTLE_DELAY_MS = 1_100L
        private const val MAX_CHAT_INPUT_FOCUS_ATTEMPTS = 4
        private const val CHAT_INPUT_FOCUS_RETRY_MS = 450L
        const val ACTION_OPEN_TASK_URL = "com.app.helper.OPEN_TASK_URL_IN_CHAT"
        const val ACTION_RETURN_TO_CHAT = "com.app.helper.RETURN_TO_CHAT"
        const val ACTION_CANCEL_CHAT_TASK = "com.app.helper.CANCEL_CHAT_TASK"
        const val ACTION_CHAT_READY = "com.app.helper.CHAT_READY"
        const val ACTION_PRODUCT_BLOCKED = "com.app.helper.PRODUCT_BLOCKED"
        const val EXTRA_URL = "url"

        @Volatile
        var isEnabled = false
            private set
    }

    private val handler = Handler(Looper.getMainLooper())
    private val runPendingUrlRunnable = Runnable {
        runSafely("runPendingUrl") { runPendingUrl() }
    }
    private val linkAttemptRunnables = mutableListOf<Runnable>()
    private var receiver: BroadcastReceiver? = null
    private var pendingUrl: String? = null
    private var returnAttempts = 0
    private var lastPopupCloseAt = 0L
    private var chatInputFocusAttempts = 0

    override fun onServiceConnected() {
        super.onServiceConnected()
        runSafely("onServiceConnected") {
            isEnabled = true
            val info = serviceInfo
            if (info != null) {
                serviceInfo = info.apply {
                    eventTypes = AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED or
                        AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED or
                        AccessibilityEvent.TYPE_VIEW_FOCUSED
                    feedbackType = AccessibilityServiceInfo.FEEDBACK_GENERIC
                    flags = flags or
                        AccessibilityServiceInfo.FLAG_REPORT_VIEW_IDS or
                        AccessibilityServiceInfo.FLAG_RETRIEVE_INTERACTIVE_WINDOWS
                    notificationTimeout = 80L
                }
            } else {
                Log.w(TAG, "Service info is null, keeping XML accessibility config")
            }
            registerCommandReceiver()
            Toast.makeText(this, "聊天任务辅助已开启", Toast.LENGTH_SHORT).show()
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        isEnabled = false
        receiver?.let { runCatching { unregisterReceiver(it) } }
        receiver = null
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        runSafely("onAccessibilityEvent") {
            if (pendingUrl != null) {
                handler.removeCallbacks(runPendingUrlRunnable)
                handler.postDelayed(runPendingUrlRunnable, 250L)
                return@runSafely
            }

            val root = activeContentRoot() ?: return@runSafely
            if (containsAnyText(root, productUnavailableHints)) {
                closeBlockingPopup(root)
                sendBroadcast(Intent(ACTION_PRODUCT_BLOCKED))
                handler.postDelayed({ runSafely("returnAfterBlocked") { returnToChatScreen() } }, 1_400L)
                return@runSafely
            }

            if (hasShopeeRecommendPopup(root)) {
                closeBlockingPopup(root)
            }
        }
    }

    override fun onInterrupt() = Unit

    private fun runSafely(section: String, block: () -> Unit) {
        try {
            block()
        } catch (e: Throwable) {
            Log.e(TAG, "Accessibility section failed: $section", e)
        }
    }

    private fun registerCommandReceiver() {
        receiver?.let { runCatching { unregisterReceiver(it) } }
        receiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                runSafely("commandReceiver") {
                    when (intent.action) {
                        ACTION_OPEN_TASK_URL -> {
                            val url = intent.getStringExtra(EXTRA_URL)?.trim().orEmpty()
                            if (url.isBlank()) return@runSafely
                            cancelLinkAttempts()
                            pendingUrl = url
                            chatInputFocusAttempts = 0
                            handler.post(runPendingUrlRunnable)
                        }
                        ACTION_RETURN_TO_CHAT -> {
                            cancelLinkAttempts()
                            handler.post { runSafely("returnToChatCommand") { returnToChatScreen() } }
                        }
                        ACTION_CANCEL_CHAT_TASK -> {
                            cancelAutomation()
                        }
                    }
                }
            }
        }
        val filter = IntentFilter().apply {
            addAction(ACTION_OPEN_TASK_URL)
            addAction(ACTION_RETURN_TO_CHAT)
            addAction(ACTION_CANCEL_CHAT_TASK)
        }
        val currentReceiver = receiver ?: return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(currentReceiver, filter, RECEIVER_NOT_EXPORTED)
        } else {
            registerReceiver(currentReceiver, filter)
        }
    }

    private fun runPendingUrl() {
        val url = pendingUrl ?: return
        val root = activeContentRoot() ?: return
        val input = findEditable(root)
        if (input == null) {
            val candidate = findChatInputCandidate(root)
            if (candidate != null && chatInputFocusAttempts < MAX_CHAT_INPUT_FOCUS_ATTEMPTS) {
                chatInputFocusAttempts++
                val opened = clickNode(candidate) ||
                    candidate.performAction(AccessibilityNodeInfo.ACTION_CLICK) ||
                    candidate.performAction(AccessibilityNodeInfo.ACTION_FOCUS) ||
                    clickNodeCenter(candidate)
                if (opened) {
                    handler.postDelayed(runPendingUrlRunnable, CHAT_INPUT_FOCUS_RETRY_MS)
                    return
                }
            }
            Log.w(TAG, "No editable input found")
            return
        }
        chatInputFocusAttempts = 0

        if (!input.performAction(AccessibilityNodeInfo.ACTION_FOCUS)) {
            input.performAction(AccessibilityNodeInfo.ACTION_CLICK)
        }

        val filled = setNodeText(input, url) || pasteText(input, url)
        if (!filled) {
            Log.w(TAG, "Unable to fill chat input")
            return
        }

        pendingUrl = null
        handler.postDelayed({ runSafely("clickSendOrEnter") { clickSendOrEnter(url) } }, 350L)
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
                runSafely("clickVisibleTaskLink") {
                    if (clickVisibleTaskLink(url)) cancelLinkAttempts()
                }
            }
            linkAttemptRunnables.add(runnable)
            handler.postDelayed(runnable, delayMs)
        }
    }

    private fun cancelLinkAttempts() {
        linkAttemptRunnables.forEach { handler.removeCallbacks(it) }
        linkAttemptRunnables.clear()
    }

    private fun cancelAutomation() {
        pendingUrl = null
        chatInputFocusAttempts = 0
        handler.removeCallbacksAndMessages(null)
        cancelLinkAttempts()
    }

    private fun clickVisibleTaskLink(url: String): Boolean {
        val root = activeContentRoot() ?: return false
        if (!isProbablyChatScreen(root)) return false
        val node = findProductLinkNode(root, url) ?: return false
        return clickNode(node) || clickNodeCenter(node)
    }

    private fun findEditable(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        findEditableNode(node, ::hasStrongChatInputHint)?.let { return it }
        val hasChatContext = containsAnyText(node, chatHints) || findChatSendButton(node) != null
        if (hasChatContext) {
            findEditableNode(node, ::hasAnyChatInputHint)?.let { return it }
            findEditableNode(node) { true }?.let { return it }
        }
        return null
    }

    private fun findChatInputCandidate(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        if (isVisibleNodeWithBounds(node) &&
            !isSearchEditableNode(node) &&
            hasChatInputPlaceholderHint(node) &&
            isLikelyChatInputArea(node)
        ) {
            return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findChatInputCandidate(child)?.let { return it }
        }
        return null
    }

    private fun findEditableNode(
        node: AccessibilityNodeInfo,
        accept: (AccessibilityNodeInfo) -> Boolean
    ): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        if (node.isEnabled && isRealEditableNode(node) && !isSearchEditableNode(node) && accept(node)) return node
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findEditableNode(child, accept)?.let { return it }
        }
        return null
    }

    private fun findSendButton(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        val label = nodeLabel(node)
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        if (node.isEnabled && (node.isClickable || node.isFocusable)) {
            if (sendLabels.any { label.equals(it, ignoreCase = true) } ||
                sendIdHints.any { viewId.contains(it) }
            ) return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findSendButton(child)?.let { return it }
        }
        return null
    }

    private fun findUrlNode(node: AccessibilityNodeInfo, needle: String): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        val text = nodeLabel(node)
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
        runSafely("returnToChatScreen") {
            returnAttempts = 0
            stepReturnToChat()
        }
    }

    private fun stepReturnToChat() {
        val root = activeContentRoot()
        if (root != null && closeBlockingPopup(root)) {
            handler.postDelayed({ runSafely("stepReturnToChat") { stepReturnToChat() } }, POPUP_SETTLE_DELAY_MS)
            return
        }

        val hasChatContext = root != null && hasChatScreenContext(root)
        if (root != null && isReadyChatScreen(root)) {
            sendBroadcast(Intent(ACTION_CHAT_READY))
            return
        }

        if (hasChatContext) {
            Log.w(TAG, "Chat screen found but input is not ready")
            return
        }

        if (returnAttempts >= MAX_RETURN_ATTEMPTS) {
            Log.w(TAG, "Return to chat stopped after $returnAttempts attempts")
            return
        }

        val shouldBack = root == null ||
            isShopeeBrowsingPage(root) ||
            isLikelyProductPage(root) ||
            containsAnyText(root, productUnavailableHints)
        if (!shouldBack) return

        returnAttempts++
        performGlobalAction(GLOBAL_ACTION_BACK)
        handler.postDelayed({ runSafely("stepReturnToChat") { stepReturnToChat() } }, RETURN_STEP_DELAY_MS)
    }

    private fun activeContentRoot(): AccessibilityNodeInfo? {
        val visibleWindows = windows.orEmpty()
            .filter { it.root != null && !isOwnAppNode(it.root) }

        visibleWindows.firstOrNull { it.isActive && isShopeeNode(it.root) }?.root?.let { return it }
        visibleWindows.firstOrNull { it.isFocused && isShopeeNode(it.root) }?.root?.let { return it }
        visibleWindows.firstOrNull { isShopeeNode(it.root) }?.root?.let { return it }
        visibleWindows.firstOrNull()?.root?.let { return it }
        return rootInActiveWindow?.takeUnless { isOwnAppNode(it) }
    }

    private fun isProbablyChatScreen(root: AccessibilityNodeInfo): Boolean {
        val input = findEditable(root)
        if (input != null) return isNodeUsable(input) && isChatInputContext(root, input)
        val candidate = findChatInputCandidate(root) ?: return false
        return isChatInputContext(root, candidate)
    }

    private fun hasChatScreenContext(root: AccessibilityNodeInfo): Boolean {
        val input = findEditable(root)
        if (input != null) return isChatInputContext(root, input)
        val candidate = findChatInputCandidate(root) ?: return false
        return isChatInputContext(root, candidate)
    }

    private fun hasChatPageSignal(root: AccessibilityNodeInfo): Boolean {
        return hasChatScreenContext(root) || containsAnyText(root, chatScreenHints)
    }

    private fun isChatInputContext(
        root: AccessibilityNodeInfo,
        input: AccessibilityNodeInfo
    ): Boolean {
        val strongInputId = hasStrongChatInputHint(input)
        val hintedInput = hasAnyChatInputHint(input)
        val hasChatText = containsAnyText(root, chatHints)
        val hasSendButton = findChatSendButton(root) != null
        return strongInputId || (hintedInput && hasChatText && hasSendButton)
    }

    private fun hasStrongChatInputHint(node: AccessibilityNodeInfo): Boolean {
        val inputLabel = nodeLabel(node).lowercase()
        val inputId = node.viewIdResourceName.orEmpty().lowercase()
        return chatStrongInputIdHints.any { inputId.contains(it) } ||
            chatStrongInputTextHints.any { inputLabel.contains(it) }
    }

    private fun hasAnyChatInputHint(node: AccessibilityNodeInfo): Boolean {
        val inputLabel = nodeLabel(node).lowercase()
        val inputId = node.viewIdResourceName.orEmpty().lowercase()
        return chatInputIdHints.any { inputId.contains(it) } ||
            chatInputTextHints.any { inputLabel.contains(it) }
    }

    private fun hasChatInputPlaceholderHint(node: AccessibilityNodeInfo): Boolean {
        val inputLabel = nodeLabel(node).lowercase()
        return chatInputPlaceholderHints.any { inputLabel.contains(it) }
    }

    private fun isLikelyChatInputArea(node: AccessibilityNodeInfo): Boolean {
        val rect = Rect()
        node.getBoundsInScreen(rect)
        val displayHeight = resources.displayMetrics.heightPixels.coerceAtLeast(1)
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        return rect.centerY() >= (displayHeight * 0.55f).toInt() ||
            chatInputIdHints.any { viewId.contains(it) } ||
            node.isClickable ||
            node.isFocusable
    }

    private fun isReadyChatScreen(root: AccessibilityNodeInfo): Boolean {
        return isProbablyChatScreen(root) && !hasBlockingPopup(root)
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

        val hasUnavailablePopup = containsAnyText(root, productUnavailableHints)
        val hasRecommendPopup = hasShopeeRecommendPopup(root)
        if (!hasUnavailablePopup && !hasRecommendPopup) return false

        if (hasUnavailablePopup && clickActionButtonByLabels(root, productUnavailableActionLabels)) {
            lastPopupCloseAt = now
            return true
        }

        if (hasUnavailablePopup && closeDialogButton(root)) {
            lastPopupCloseAt = now
            return true
        }

        if (hasUnavailablePopup && performGlobalAction(GLOBAL_ACTION_BACK)) {
            lastPopupCloseAt = now
            return true
        }

        val closed = closeTopRightIconButton(root) || tapShopeePopupClosePoint()
        if (closed) lastPopupCloseAt = now
        return closed
    }

    private fun clickActionButtonByLabels(
        node: AccessibilityNodeInfo,
        labels: Set<String>
    ): Boolean {
        if (isOwnAppNode(node)) return false
        val label = nodeLabel(node)
        if (isNodeUsable(node) && labels.any { label.contains(it, ignoreCase = true) }) {
            return clickNode(node) || clickNodeCenter(node)
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            if (clickActionButtonByLabels(child, labels)) return true
        }
        return false
    }

    private fun hasBlockingPopup(root: AccessibilityNodeInfo): Boolean {
        return containsAnyText(root, productUnavailableHints) ||
            hasShopeeRecommendPopup(root)
    }

    private fun hasShopeeRecommendPopup(root: AccessibilityNodeInfo): Boolean {
        val display = resources.displayMetrics
        val hasCloseButton = findTopRightIconButton(root, display.widthPixels, display.heightPixels) != null
        if (!hasCloseButton) return false

        val hasMallTab = containsAnyText(root, shopeeMallHints)
        val hasPaymentTab = containsAnyText(root, shopeePaymentHints)
        if (hasMallTab && hasPaymentTab) return true
        if (!containsAnyTextNearTop(root, shopeeRecommendPopupHints)) return false
        return true
    }

    private fun isLikelyProductPage(root: AccessibilityNodeInfo): Boolean {
        if (!isShopeeNode(root) || hasChatPageSignal(root)) return false
        return containsAnyText(root, productPageHints) ||
            hasShopeeRecommendPopup(root) ||
            containsAnyText(root, productUnavailableHints)
    }

    private fun isShopeeBrowsingPage(root: AccessibilityNodeInfo?): Boolean {
        if (root == null || !isShopeeNode(root)) return false
        if (hasChatPageSignal(root)) return false
        return true
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
        val label = nodeLabel(node)
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
        val text = nodeLabel(node)
        if (needles.any { text.contains(it, ignoreCase = true) }) return true
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            if (containsAnyText(child, needles)) return true
        }
        return false
    }

    private fun containsAnyTextNearTop(node: AccessibilityNodeInfo, needles: Set<String>): Boolean {
        if (isOwnAppNode(node)) return false
        val rect = Rect()
        node.getBoundsInScreen(rect)
        val maxY = (resources.displayMetrics.heightPixels * 0.45f).toInt()
        val text = nodeLabel(node)
        if (rect.top in 0..maxY && needles.any { text.contains(it, ignoreCase = true) }) return true
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            if (containsAnyTextNearTop(child, needles)) return true
        }
        return false
    }

    private fun isOwnAppNode(node: AccessibilityNodeInfo): Boolean {
        return node.packageName?.toString() == packageName
    }

    private fun isRealEditableNode(node: AccessibilityNodeInfo): Boolean {
        return node.isEditable || canSetText(node)
    }

    private fun canSetText(node: AccessibilityNodeInfo): Boolean {
        return node.actionList.any { it.id == AccessibilityNodeInfo.ACTION_SET_TEXT }
    }

    private fun isSearchEditableNode(node: AccessibilityNodeInfo): Boolean {
        val label = nodeLabel(node).lowercase()
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        return searchInputHints.any { label.contains(it) || viewId.contains(it) }
    }

    private fun findChatSendButton(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (isOwnAppNode(node)) return null
        val label = nodeLabel(node)
        val viewId = node.viewIdResourceName.orEmpty().lowercase()
        if (node.isEnabled && (node.isClickable || node.isFocusable)) {
            if (chatSendLabels.any { label.equals(it, ignoreCase = true) } ||
                chatSendIdHints.any { viewId.contains(it) }
            ) return node
        }
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findChatSendButton(child)?.let { return it }
        }
        return null
    }

    private fun isNodeUsable(node: AccessibilityNodeInfo): Boolean {
        if (!node.isEnabled || !isVisibleNodeWithBounds(node)) return false
        return true
    }

    private fun isVisibleNodeWithBounds(node: AccessibilityNodeInfo): Boolean {
        if (!node.isVisibleToUser) return false
        val rect = Rect()
        node.getBoundsInScreen(rect)
        return rect.width() > 0 && rect.height() > 0
    }

    private fun isShopeeNode(node: AccessibilityNodeInfo): Boolean {
        val pkg = node.packageName?.toString().orEmpty()
        return pkg.contains("shopee", ignoreCase = true)
    }

    private fun nodeLabel(node: AccessibilityNodeInfo): String {
        return buildString {
            node.text?.let { append(it) }
            node.contentDescription?.let { append(' ').append(it) }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                node.hintText?.let { append(' ').append(it) }
            }
        }.trim()
    }

    private val sendLabels = setOf(
        "发送", "送出", "傳送", "传送", "Send", "send", "Enter", "enter", "Go", "go", "进入", "前往"
    )

    private val sendIdHints = listOf("send", "submit", "enter", "go")

    private val chatSendLabels = setOf(
        "发送", "送出", "傳送", "传送", "Send", "send"
    )

    private val chatSendIdHints = listOf("send")

    private val chatInputIdHints = listOf(
        "input", "edit", "message", "chat", "compose", "edt"
    )

    private val chatStrongInputIdHints = listOf(
        "message", "chat", "compose"
    )

    private val chatStrongInputTextHints = listOf(
        "訊息", "消息", "私訊", "私信", "message", "chat"
    )

    private val chatInputTextHints = listOf(
        "輸入", "输入", "訊息", "消息", "私訊", "私信", "message", "chat"
    )

    private val chatInputPlaceholderHints = listOf(
        "輸入私訊", "输入私信", "輸入訊息", "输入讯息", "輸入消息", "输入消息",
        "輸入訊息內容", "输入消息内容", "傳送訊息", "发送消息",
        "type a message", "write a message", "enter message"
    )

    private val searchInputHints = listOf(
        "搜尋", "搜寻", "搜索", "search", "keyword", "關鍵字", "关键字", "找商品"
    )

    private val dialogCloseLabels = setOf(
        "关闭", "關閉", "取消", "稍后", "稍後", "知道了", "我知道了", "确定", "確認",
        "OK", "Ok", "ok", "Close", "close", "Cancel", "cancel", "Later", "later",
        "Not now", "not now", "No thanks", "no thanks", "×", "X", "x"
    )

    private val dialogCloseIdHints = listOf("close", "cancel", "dismiss")

    private val chatHints = setOf(
        "发送", "傳送", "消息", "訊息", "message", "Message", "chat", "Chat"
    )

    private val chatScreenHints = setOf(
        "輸入私訊", "输入私信", "評價客服", "评价客服",
        "在開始聊天前", "在开始聊天前", "請先驗證您的電話號碼", "请先验证您的电话号码",
        "點此驗證", "点此验证"
    )

    private val productUnavailableHints = setOf(
        "你尋找的商品已不存在", "你寻找的商品已不存在", "商品不存在", "商品不存在或已下架",
        "商品已下架", "商品未找到", "找不到商品",
        "產品不存在", "產品已下架", "賣場不存在", "商品不存在或已被移除",
        "Product not found", "product not found", "product does not exist",
        "This product does not exist", "unavailable", "Unavailable"
    )

    private val productUnavailableActionLabels = setOf(
        "回到上一步", "返回上一步", "回上一頁", "回上一页", "返回上一頁", "返回上一页",
        "回去", "Go back", "go back", "Return to previous page", "return to previous page"
    )

    private val productPageHints = setOf(
        "立即購買", "立即购买", "加入購物車", "加入购物车", "購物車", "购物车",
        "商品詳情", "商品详情", "商品評價", "商品评价", "評價", "评价",
        "已售出", "已售", "規格", "规格", "運送", "运送", "賣場", "卖场",
        "Buy Now", "Add to Cart", "Product Details", "Product Ratings"
    )

    private val shopeeMallHints = setOf("蝦皮商城", "虾皮商城")

    private val shopeePaymentHints = setOf("貨到付款", "货到付款")

    private val shopeeRecommendPopupHints = setOf(
        "為你推薦", "为你推荐", "推薦商品", "推荐商品", "相關商品", "相关商品",
        "相似商品", "你可能也喜歡", "你可能也喜欢", "猜你喜歡", "猜你喜欢",
        "Recommended", "recommend", "Similar", "You may also like"
    )
}
