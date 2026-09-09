package com.app.helper.chat

import android.graphics.Rect
import android.view.accessibility.AccessibilityNodeInfo

/**
 * 无障碍节点工具：遍历 Shopee 前台节点树、按文本/关键字找控件、算坐标。
 * （等价 iOS 端遍历 UIView 层级的逻辑）
 */
object ChatNodes {

    fun children(node: AccessibilityNodeInfo?): List<AccessibilityNodeInfo> {
        node ?: return emptyList()
        val count = runCatching { node.childCount }.getOrDefault(0)
        if (count <= 0) return emptyList()
        val list = ArrayList<AccessibilityNodeInfo>(count)
        for (i in 0 until count) {
            val c = runCatching { node.getChild(i) }.getOrNull() ?: continue
            list.add(c)
        }
        return list
    }

    /** 深度遍历，返回所有后代（含自身） */
    fun collect(root: AccessibilityNodeInfo?): List<AccessibilityNodeInfo> {
        root ?: return emptyList()
        val out = ArrayList<AccessibilityNodeInfo>()
        fun walk(n: AccessibilityNodeInfo) {
            out.add(n)
            for (c in children(n)) walk(c)
        }
        walk(root)
        return out
    }

    fun textOf(node: AccessibilityNodeInfo?): String {
        node ?: return ""
        val t = runCatching { node.text?.toString() }.getOrNull() ?: ""
        val cd = runCatching { node.contentDescription?.toString() }.getOrNull() ?: ""
        return (t + " " + cd).trim()
    }

    fun bounds(node: AccessibilityNodeInfo?): Rect {
        val r = Rect()
        node ?: return r
        runCatching { node.getBoundsInScreen(r) }
        return r
    }

    fun containsAny(text: String, keywords: List<String>): String? {
        if (text.isEmpty() || keywords.isEmpty()) return null
        val lower = text.lowercase()
        for (k in keywords) {
            if (k.isNotEmpty() && lower.contains(k.lowercase())) return k
        }
        return null
    }

    /** 找一个可点击节点（自身或祖先可点击） */
    fun clickableNode(node: AccessibilityNodeInfo?): AccessibilityNodeInfo? {
        var cur: AccessibilityNodeInfo? = node ?: return null
        while (true) {
            val c = cur ?: break
            if (runCatching { c.isClickable }.getOrDefault(false)) return c
            cur = runCatching { c.parent }.getOrNull()
        }
        return null
    }

    fun performClick(node: AccessibilityNodeInfo?): Boolean {
        val target = clickableNode(node) ?: node ?: return false
        return runCatching { target.performAction(AccessibilityNodeInfo.ACTION_CLICK) }.getOrDefault(false)
    }

    fun visibleCenter(node: AccessibilityNodeInfo?, width: Int, height: Int): IntArray? {
        val r = bounds(node)
        if (r.isEmpty) return null
        if (r.right < 0 || r.left > width || r.bottom < 0 || r.top > height) return null
        return intArrayOf(r.centerX(), r.centerY())
    }

    fun center(node: AccessibilityNodeInfo?): IntArray? {
        val r = bounds(node)
        if (r.isEmpty) return null
        return intArrayOf(r.centerX(), r.centerY())
    }

    fun isEditableCandidate(node: AccessibilityNodeInfo?): Boolean {
        node ?: return false
        if (!runCatching { node.isVisibleToUser }.getOrDefault(false)) return false
        val editable = runCatching { node.isEditable }.getOrDefault(false)
        if (editable) return true
        val cls = runCatching { node.className?.toString() }.getOrNull() ?: ""
        return cls.contains("EditText") || cls.contains("TextInputEditText") ||
            cls.contains("ChatInput") || cls.contains("Composer")
    }

    /** 在列表里找一个文本命中关键字的节点（可再按区域过滤） */
    fun findByText(root: AccessibilityNodeInfo?, keywords: List<String>): AccessibilityNodeInfo? {
        val nodes = collect(root)
        for (n in nodes) {
            if (containsAny(textOf(n), keywords) != null) return n
        }
        return null
    }
}
