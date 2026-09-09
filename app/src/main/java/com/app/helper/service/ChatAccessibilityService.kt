package com.app.helper.service

import android.accessibilityservice.AccessibilityService
import android.accessibilityservice.AccessibilityServiceInfo
import android.accessibilityservice.GestureDescription
import android.graphics.Bitmap
import android.graphics.Path
import android.graphics.Point
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.util.Log
import android.view.Display
import android.view.accessibility.AccessibilityEvent
import kotlin.coroutines.resume
import kotlinx.coroutines.suspendCancellableCoroutine

/**
 * 聊一聊无障碍服务：持有 Shopee 前台窗口节点树、注入点击/滑动、截屏(API30+)。
 * 真正的编排逻辑在 ChatAuto（不在这里做状态机）。
 */
class ChatAccessibilityService : AccessibilityService() {

    companion object {
        private const val TAG = "ChatAccService"
        @Volatile
        private var instance: ChatAccessibilityService? = null
        private val mainHandler = Handler(Looper.getMainLooper())

        fun isReady(): Boolean = instance != null

        fun current(): ChatAccessibilityService? = instance
    }

    private val screenshotExecutor = java.util.concurrent.Executors.newSingleThreadExecutor()
    private var displaySize = Point()

    override fun onServiceConnected() {
        super.onServiceConnected()
        instance = this
        refreshDisplaySize()
        Log.i(TAG, "connected, screenshot=${screenshotSupported()}")
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        // 由 TaskService/ChatAuto 按需主动查询，这里不驱动
    }

    override fun onInterrupt() {}

    override fun onUnbind(intent: android.content.Intent?): Boolean {
        if (instance === this) instance = null
        return super.onUnbind(intent)
    }

    override fun onDestroy() {
        if (instance === this) instance = null
        screenshotExecutor.shutdown()
        super.onDestroy()
    }

    fun refreshDisplaySize() {
        try {
            val wm = getSystemService(android.content.Context.WINDOW_SERVICE) as android.view.WindowManager
            @Suppress("DEPRECATION")
            wm.defaultDisplay.getRealSize(displaySize)
        } catch (e: Exception) {
            displaySize = Point(resources.displayMetrics.widthPixels, resources.displayMetrics.heightPixels)
        }
    }

    fun screenSize(): Point = displaySize

    /** 前台窗口是否为 Shopee */
    fun isShopeeForeground(): Boolean {
        val root = rootInActiveWindow ?: return false
        return isShopeePackage(root.packageName?.toString())
    }

    /** 取 Shopee 前台窗口的根节点（非 Shopee 返回 null） */
    fun activeShopeeRoot(): android.view.accessibility.AccessibilityNodeInfo? {
        val root = rootInActiveWindow ?: return null
        return root.takeIf { isShopeePackage(it.packageName?.toString()) }
    }

    private fun isShopeePackage(pkg: String?): Boolean {
        if (pkg.isNullOrEmpty()) return false
        return pkg.equals("com.shopee.tw", true) ||
            pkg.contains("shopee", true)
    }

    /** API30+ 是否支持无障碍截屏（通过 AccessibilityServiceInfo 能力位判断） */
    fun screenshotSupported(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R) return false
        return runCatching {
            val caps = serviceInfo?.capabilities ?: 0
            caps and AccessibilityServiceInfo.CAPABILITY_CAN_TAKE_SCREENSHOT != 0
        }.getOrDefault(false)
    }

    /** API30+ 截屏（无障碍授权即可，无需 MediaProjection）；低于 30 返回 null */
    suspend fun captureScreenshot(): Bitmap? {
        if (!screenshotSupported()) return null
        return suspendCancellableCoroutine { cont ->
            try {
                @Suppress("DEPRECATION")
                takeScreenshot(Display.DEFAULT_DISPLAY, screenshotExecutor, object : AccessibilityService.TakeScreenshotCallback {
                    override fun onSuccess(screenshot: AccessibilityService.ScreenshotResult) {
                        val hb = screenshot.hardwareBuffer
                        var bmp: Bitmap? = null
                        try {
                            bmp = Bitmap.wrapHardwareBuffer(hb, null)?.copy(Bitmap.Config.ARGB_8888, false)
                        } catch (e: Exception) {
                            Log.w(TAG, "screenshot convert fail", e)
                        } finally {
                            hb.close()
                        }
                        if (!cont.isCancelled) cont.resume(bmp)
                    }

                    override fun onFailure(errorCode: Int) {
                        Log.w(TAG, "takeScreenshot fail code=$errorCode")
                        if (!cont.isCancelled) cont.resume(null)
                    }
                })
            } catch (e: Exception) {
                Log.w(TAG, "takeScreenshot exception", e)
                if (!cont.isCancelled) cont.resume(null)
            }
        }
    }

    // ---------- 手势注入（须主线程） ----------
    fun tap(x: Int, y: Int) {
        mainHandler.post {
            try {
                val path = Path().apply { moveTo(x.toFloat(), y.toFloat()) }
                val gesture = GestureDescription.Builder()
                    .addStroke(GestureDescription.StrokeDescription(path, 0L, 90L))
                    .build()
                dispatchGesture(gesture, null, null)
            } catch (e: Exception) {
                Log.w(TAG, "tap fail", e)
            }
        }
    }

    fun swipe(x1: Int, y1: Int, x2: Int, y2: Int, durationMs: Long = 300L) {
        mainHandler.post {
            try {
                val path = Path().apply {
                    moveTo(x1.toFloat(), y1.toFloat())
                    lineTo(x2.toFloat(), y2.toFloat())
                }
                val gesture = GestureDescription.Builder()
                    .addStroke(GestureDescription.StrokeDescription(path, 0L, durationMs))
                    .build()
                dispatchGesture(gesture, null, null)
            } catch (e: Exception) {
                Log.w(TAG, "swipe fail", e)
            }
        }
    }

    fun globalBack(): Boolean {
        return runCatching { performGlobalAction(GLOBAL_ACTION_BACK) }.getOrDefault(false)
    }
}
