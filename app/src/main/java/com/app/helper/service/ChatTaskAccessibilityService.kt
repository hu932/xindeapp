package com.app.helper.service

import android.accessibilityservice.AccessibilityService
import android.accessibilityservice.AccessibilityServiceInfo
import android.content.BroadcastReceiver
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
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
        const val EXTRA_URL = "url"

        @Volatile
        var isEnabled = false
            private set
    }

    private val handler = Handler(Looper.getMainLooper())
    private var receiver: BroadcastReceiver? = null
    private var pendingUrl: String? = null

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
            handler.removeCallbacksAndMessages(null)
            handler.postDelayed({ runPendingUrl() }, 250L)
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
                        pendingUrl = url
                        handler.post { runPendingUrl() }
                    }
                    ACTION_RETURN_TO_CHAT -> {
                        handler.post { performGlobalAction(GLOBAL_ACTION_BACK) }
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
        val root = rootInActiveWindow ?: return
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
        handler.postDelayed({ clickVisibleTaskLink(url) }, 1100L)
        handler.postDelayed({ clickVisibleTaskLink(url) }, 2200L)
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
        val root = rootInActiveWindow ?: return
        val sendNode = findSendButton(root)
        if (sendNode != null && clickNode(sendNode)) {
            handler.postDelayed({ clickVisibleTaskLink(url) }, 700L)
            return
        }
        // Some chat pages turn pasted links into tappable chips before sending.
        clickVisibleTaskLink(url)
    }

    private fun clickVisibleTaskLink(url: String): Boolean {
        val root = rootInActiveWindow ?: return false
        val node = findUrlNode(root, url) ?: findUrlNode(root, "shopee.tw")
        return node != null && clickNode(node)
    }

    private fun findEditable(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
        if (node.isEditable && node.isEnabled) return node
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            findEditable(child)?.let { return it }
        }
        return null
    }

    private fun findSendButton(node: AccessibilityNodeInfo): AccessibilityNodeInfo? {
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

    private val sendLabels = setOf(
        "发送", "送出", "傳送", "传送", "Send", "send", "Enter", "enter", "Go", "go", "进入", "前往"
    )

    private val sendIdHints = listOf("send", "submit", "enter", "go")
}
