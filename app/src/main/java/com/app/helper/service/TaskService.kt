package com.app.helper.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.graphics.PixelFormat
import android.net.Uri
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.util.Log
import android.view.Gravity
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import android.widget.CheckBox
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import com.app.helper.R
import com.app.helper.chat.ChatAuto
import com.app.helper.network.ApiConfig
import com.app.helper.network.ApiService
import com.app.helper.network.CouponPreflight
import com.app.helper.network.ExtTask
import com.app.helper.util.DelayPrefs
import com.app.helper.util.CrashHandler
import com.app.helper.util.TokenManager
import kotlinx.coroutines.*
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class TaskService : android.app.Service() {

    companion object {
        private const val TAG = "TaskService"
        private const val CHANNEL_ID = "app_service_channel"
        private const val NOTIFICATION_ID = 1001
        private const val PREF_NAME = "task_config"
        var isRunning = false
            private set
    }

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val handler = Handler(Looper.getMainLooper())

    private var windowManager: WindowManager? = null
    private var floatView: View? = null
    private var floatParams: WindowManager.LayoutParams? = null
    private var tvStatus: TextView? = null
    private var tvAction: TextView? = null
    private var tvDot: TextView? = null
    private var llDotContainer: LinearLayout? = null
    private var llExpanded: LinearLayout? = null
    private var imgClose: ImageView? = null
    private var tvSmallStatus: TextView? = null
    private var tvSmallCount: TextView? = null
    private var llSmall: LinearLayout? = null
    private var llSmallBody: LinearLayout? = null
    private var tvSmallRestore: TextView? = null
    private var cbSmallWindow: CheckBox? = null
    private var isCollapsed = false

    private var isAutoRunning = false
    private var isUploading = false
    private var currentTaskId = ""
    private var currentShopId = 0L
    private var currentItemId = 0L
    private var currentTask: ExtTask? = null
    private var statusMessage = ""
    private var compactStatusMessage = ""
    private var uploadCount = 0
    private var failedCount = 0
    private var targetVersion = "未知"
    private var countdownJob: Job? = null
    private var heartbeatJob: Job? = null
    @Volatile private var deviceAllowed = true
    @Volatile private var deviceBlockedMsg = "设备已被禁用"
    @Volatile private var versionBlocked = false
    @Volatile private var versionBlockedMsg = "版本已停用，请更新"
    private var consecutiveTimeouts = 0
    @Volatile private var riskReported = false

    private var touchDownX = 0f
    private var touchDownY = 0f
    private var touchDownTime = 0L
    private var windowX = 0
    private var windowY = 0

    private var receiver: BroadcastReceiver? = null
    @Volatile private var pendingFileContent: String? = null

    private val prefs by lazy { getSharedPreferences(PREF_NAME, MODE_PRIVATE) }
    private val pauseEnabled get() = prefs.getBoolean("pause_enabled", false)
    private val pauseCount get() = prefs.getInt("pause_count", 30)
    private val pauseMinutes get() = prefs.getInt("pause_minutes", 30)
    private val silentMode get() = prefs.getBoolean("silent_mode", false)
    private val chatMode get() = prefs.getBoolean("chat_mode", false)
    private val smallWindowMode get() = prefs.getBoolean("small_window_mode", false)
    private val couponPreflightEnabled get() = prefs.getBoolean("coupon_preflight_enabled", true)
    private var sessionUploadCount = 0
    private var pauseJob: Job? = null
    @Volatile private var usedChatFlow = false

    override fun onBind(intent: Intent?): IBinder? = null

    private var updateCheckJob: Job? = null

    override fun onCreate() {
        super.onCreate()
        isRunning = true
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
        registerDataReceiver()
        createFloatWindow()
        startHeartbeat()
        startUpdateCheck()
        // 崩溃时把任务现场一起写进报告，否则只有堆栈不知道当时跑到哪一步
        CrashHandler.stateProvider = {
            buildString {
                appendLine("自动运行: $isAutoRunning")
                appendLine("上传中: $isUploading")
                appendLine("连续超时: $consecutiveTimeouts")
                appendLine("待处理数据: ${pendingFileContent?.length ?: 0} 字节")
                appendLine("任务ID: $currentTaskId")
                appendLine("shopId: $currentShopId  itemId: $currentItemId")
            }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            "PROCESS_FILE" -> {
                val content = intent.getStringExtra("file_content").orEmpty()
                if (content.isNotBlank()) {
                    pendingFileContent = content
                    Log.d(TAG, "PROCESS_FILE: ${content.length} bytes")
                }
            }
            "SET_DELAY" -> {
                val minMs = intent.getLongExtra("delay_min_ms", 15_000L)
                val maxMs = intent.getLongExtra("delay_max_ms", 35_000L)
                DelayPrefs.set(this, minMs, maxMs)
                updateFloatStatus("延时已设置: ${DelayPrefs.formatMs(minMs)}-${DelayPrefs.formatMs(maxMs)}")
            }
            "SET_PAUSE" -> {
                val enabled = intent.getBooleanExtra("pause_enabled", false)
                val count = intent.getIntExtra("pause_count", 30)
                val minutes = intent.getIntExtra("pause_minutes", 30)
                prefs.edit()
                    .putBoolean("pause_enabled", enabled)
                    .putInt("pause_count", count)
                    .putInt("pause_minutes", minutes)
                    .apply()
                if (!enabled) {
                    sessionUploadCount = 0
                    pauseJob?.cancel()
                }
                updateFloatStatus(if (enabled) "定时休息已开启: ${count}个/${minutes}分钟" else "定时休息已关闭")
            }
            "SET_SILENT_MODE" -> {
                val enabled = intent.getBooleanExtra("silent_mode", false)
                prefs.edit().putBoolean("silent_mode", enabled).apply()
                updateFloatStatus(if (enabled) "已开启静默模式（不跳转商品页）" else "已关闭静默模式（跳转商品页）")
            }
            "SET_CHAT_MODE" -> {
                val enabled = intent.getBooleanExtra("chat_mode", false)
                prefs.edit().putBoolean("chat_mode", enabled).apply()
                updateFloatStatus(if (enabled) "已开启聊一聊（需在Shopee聊天页）" else "已关闭聊一聊")
                if (enabled && !ChatAccessibilityService.isReady()) {
                    updateFloatStatus("聊一聊需先在系统设置开启无障碍【Ajie 聊一聊助手】")
                }
            }
            "SET_SMALL_WINDOW" -> {
                val enabled = intent.getBooleanExtra("small_window_mode", false)
                setSmallWindowMode(enabled)
            }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        isRunning = false
        isAutoRunning = false
        serviceScope.cancel()
        countdownJob?.cancel()
        heartbeatJob?.cancel()
        updateCheckJob?.cancel()
        pauseJob?.cancel()
        removeFloatWindow()
        unregisterDataReceiver()
    }

    // ==================== Float Window ====================

    private fun createFloatWindow(showWelcome: Boolean = true) {
        windowManager = getSystemService(WINDOW_SERVICE) as WindowManager
        val small = smallWindowMode
        floatView = LayoutInflater.from(this).inflate(
            if (small) R.layout.layout_window_float_small else R.layout.layout_window_float,
            null
        )

        val layoutType = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O)
            WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY
        else @Suppress("DEPRECATION") WindowManager.LayoutParams.TYPE_PHONE

        floatParams = WindowManager.LayoutParams(
            WindowManager.LayoutParams.WRAP_CONTENT,
            WindowManager.LayoutParams.WRAP_CONTENT,
            layoutType,
            WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL,
            PixelFormat.TRANSLUCENT
        ).apply { gravity = Gravity.TOP or Gravity.START; x = 0; y = 200 }

        tvStatus = null
        tvAction = null
        tvDot = null
        llDotContainer = null
        llExpanded = null
        imgClose = null
        tvSmallStatus = null
        tvSmallCount = null
        llSmall = null
        llSmallBody = null
        tvSmallRestore = null
        cbSmallWindow = null

        if (small) {
            tvSmallStatus = floatView?.findViewById(R.id.tv_small_status)
            tvSmallCount = floatView?.findViewById(R.id.tv_small_count)
            llSmall = floatView?.findViewById(R.id.ll_small)
            llSmallBody = floatView?.findViewById(R.id.ll_small_body)
            tvSmallRestore = floatView?.findViewById(R.id.tv_small_restore)
            llSmallBody?.setOnTouchListener { _, event -> handleSmallWindowTouch(event) }
            tvSmallRestore?.setOnClickListener { setSmallWindowMode(false) }
        } else {
            tvStatus = floatView?.findViewById(R.id.tv_status)
            tvAction = floatView?.findViewById(R.id.tv_action)
            tvDot = floatView?.findViewById(R.id.tv_dot)
            llDotContainer = floatView?.findViewById(R.id.ll_dot_container)
            llExpanded = floatView?.findViewById(R.id.ll_expanded)
            imgClose = floatView?.findViewById(R.id.img_close)
            cbSmallWindow = floatView?.findViewById(R.id.cb_small_window)

            tvStatus?.setOnTouchListener { _, event -> handleDrag(event) }
            llDotContainer?.setOnTouchListener { _, event ->
                if (event.action == MotionEvent.ACTION_DOWN) {
                    touchDownX = event.rawX; touchDownY = event.rawY
                    windowX = floatParams?.x ?: 0; windowY = floatParams?.y ?: 0
                }
                if (event.action == MotionEvent.ACTION_MOVE) {
                    val dx = Math.abs(event.rawX - touchDownX)
                    val dy = Math.abs(event.rawY - touchDownY)
                    if (dx > 5 || dy > 5) { handleDrag(event); true } else false
                } else false
            }
            imgClose?.setOnClickListener { collapseFloat() }
            llDotContainer?.setOnClickListener { expandFloat() }
            tvAction?.setOnClickListener { toggleAutoTask() }
            cbSmallWindow?.setOnCheckedChangeListener { _, checked -> setSmallWindowMode(checked) }
            cbSmallWindow?.isChecked = false
        }

        try { windowManager?.addView(floatView, floatParams) } catch (e: Exception) { Log.e(TAG, "Float window failed", e) }
        if (showWelcome) {
            updateFloatStatus(
                if (small) "小窗：点击开始/停止"
                else "点击【一键做任务】开始\n延时: ${DelayPrefs.formatMs(DelayPrefs.getMinMs(this))}-${DelayPrefs.formatMs(DelayPrefs.getMaxMs(this))}"
            )
        }
    }

    private fun handleSmallWindowTouch(event: MotionEvent): Boolean {
        when (event.action) {
            MotionEvent.ACTION_DOWN -> {
                touchDownX = event.rawX; touchDownY = event.rawY
                touchDownTime = System.currentTimeMillis()
                windowX = floatParams?.x ?: 0; windowY = floatParams?.y ?: 0
            }
            MotionEvent.ACTION_MOVE -> {
                val dx = Math.abs(event.rawX - touchDownX)
                val dy = Math.abs(event.rawY - touchDownY)
                if (dx > 5 || dy > 5) {
                    floatParams?.x = windowX + (event.rawX - touchDownX).toInt()
                    floatParams?.y = windowY + (event.rawY - touchDownY).toInt()
                    try { windowManager?.updateViewLayout(floatView, floatParams) } catch (_: Exception) {}
                }
            }
            MotionEvent.ACTION_UP -> {
                val dx = Math.abs(event.rawX - touchDownX)
                val dy = Math.abs(event.rawY - touchDownY)
                if (dx <= 5 && dy <= 5) {
                    if (System.currentTimeMillis() - touchDownTime >= 500L) setSmallWindowMode(false)
                    else toggleAutoTask()
                }
            }
        }
        return true
    }

    private fun handleDrag(event: MotionEvent): Boolean {
        when (event.action) {
            MotionEvent.ACTION_DOWN -> {
                touchDownX = event.rawX; touchDownY = event.rawY
                windowX = floatParams?.x ?: 0; windowY = floatParams?.y ?: 0
            }
            MotionEvent.ACTION_MOVE -> {
                floatParams?.x = windowX + (event.rawX - touchDownX).toInt()
                floatParams?.y = windowY + (event.rawY - touchDownY).toInt()
                try { windowManager?.updateViewLayout(floatView, floatParams) } catch (_: Exception) {}
            }
        }
        return true
    }

    private fun collapseFloat() {
        isCollapsed = true
        llExpanded?.visibility = View.GONE
        llDotContainer?.visibility = View.VISIBLE
    }

    private fun expandFloat() {
        isCollapsed = false
        llDotContainer?.visibility = View.GONE
        llExpanded?.visibility = View.VISIBLE
        tvStatus?.text = statusMessage
    }

    private fun removeFloatWindow() {
        try { floatView?.let { windowManager?.removeView(it) } } catch (_: Exception) {}
    }

    private fun setSmallWindowMode(enabled: Boolean) {
        prefs.edit().putBoolean("small_window_mode", enabled).apply()
        recreateFloatWindow()
        updateFloatStatus(if (enabled) "已切换到小窗模式" else "已切换到完整悬浮窗")
    }

    private fun recreateFloatWindow() {
        removeFloatWindow()
        createFloatWindow(showWelcome = false)
        applyStatusToViews(compactStatusMessage.ifEmpty { "点击【一键做任务】开始" })
    }

    private fun applyStatusToViews(compactMessage: String) {
        compactStatusMessage = compactMessage
        statusMessage = "$compactMessage\n下发ID：${TokenManager.getUsername()} 今日:$uploadCount"
        handler.post {
            if (!isCollapsed) tvStatus?.text = statusMessage
            tvSmallStatus?.text = compactMessage
            tvSmallCount?.text = "完成:$uploadCount 失败:$failedCount"
        }
    }

    private fun updateFloatStatus(message: String) {
        val compactMessage = message
            .replace("自动领取任务", "领任务")
            .replace("自动执行", "自动")
            .replace("当日上传成功", "今日")
        applyStatusToViews(compactMessage)
        try { sendBroadcast(Intent("com.app.helper.LOG_ACTION").putExtra("log_msg", message)) } catch (_: Exception) {}
    }

    private fun broadcastStats() {
        try {
            sendBroadcast(Intent("com.app.helper.STATS_ACTION")
                .putExtra("completed", uploadCount)
                .putExtra("failed", failedCount)
                .putExtra("target_version", targetVersion)
                .putExtra("running", isAutoRunning)
                .putExtra("current_task", if (currentTaskId.isBlank()) "无" else "shop=$currentShopId item=$currentItemId"))
        } catch (_: Exception) {}
    }

    private fun toggleAutoTask() {
        if (isUploading) { updateFloatStatus("正在上传中..."); return }
        if (versionBlocked) {
            isAutoRunning = false
            handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
            updateFloatStatus("$versionBlockedMsg\n请更新到最新版本后使用")
            sendBroadcast(Intent("com.app.helper.UPDATE_ACTION").apply {
                putExtra("title", "版本已停用")
                putExtra("message", versionBlockedMsg)
                putExtra("force", true)
                putExtra("blocked", true)
            })
            return
        }
        if (!deviceAllowed) {
            stopAutoForDeviceBlock()
            return
        }
        if (isAutoRunning) {
            isAutoRunning = false
            countdownJob?.cancel()
            pauseJob?.cancel()
            handler.post {
                tvAction?.text = "一键做任务"
                tvAction?.setBackgroundResource(R.drawable.bg_float_action)
            }
            updateFloatStatus("已停止自动执行")
            broadcastStats()
        } else {
            isAutoRunning = true
            sessionUploadCount = 0
            handler.post {
                tvAction?.text = "停止自动"
                tvAction?.setBackgroundResource(R.drawable.bg_float_action_stop)
            }
            updateFloatStatus("自动执行已开启")
            broadcastStats()
            startNextTaskCycle()
        }
    }

    // ==================== Task Flow ====================

    private fun startNextTaskCycle(forceDelaySec: Int? = null, alreadyWaitedMs: Long = 0L) {
        if (!isAutoRunning) return
        if (versionBlocked) {
            isAutoRunning = false
            handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
            updateFloatStatus("$versionBlockedMsg\n已停止自动执行")
            return
        }
        if (!deviceAllowed) {
            stopAutoForDeviceBlock()
            return
        }
        val delayMs = if (forceDelaySec != null) {
            forceDelaySec.coerceAtLeast(1) * 1000L
        } else {
            val min = DelayPrefs.getMinMs(this).coerceAtLeast(0L)
            val max = DelayPrefs.getMaxMs(this).coerceAtLeast(min)
            if (min == max) min else kotlin.random.Random.nextLong(min, max + 1L)
        }
        val remainingDelayMs = (delayMs - alreadyWaitedMs).coerceAtLeast(0L)
        val tipPrefix = if (forceDelaySec != null) "风控重试" else "自动领取任务"

        countdownJob?.cancel()
        countdownJob = serviceScope.launch {
            var remainingMs = remainingDelayMs
            while (remainingMs > 0L) {
                if (!isAutoRunning) return@launch
                if (isUploading) { delay(1000); continue }
                val displaySec = if (remainingMs < 1000L) {
                    String.format(Locale.US, "%.1f", remainingMs / 1000.0)
                } else {
                    ((remainingMs + 999L) / 1000L).toString()
                }
                updateFloatStatus("$tipPrefix，${displaySec}s后执行...")
                val stepMs = remainingMs.coerceAtMost(1000L)
                delay(stepMs)
                remainingMs -= stepMs
            }
            if (isAutoRunning && !isUploading) {
                updateFloatStatus("自动领取任务中...")
                claimAndExecuteTask()
            }
        }
    }

    // ==================== 优惠券预检 ====================

    private suspend fun awaitCouponPreflight(): Boolean {
        if (!couponPreflightEnabled) return true

        // 用缓存的真实账号（没有则用配置里的默认 accountId）向优惠券后台要检测链接
        val cached = CouponPreflight.getAccount()
        val startUserId = cached?.first ?: ApiConfig.COUPON_PREFLIGHT_ACCOUNT_ID
        if (cached != null && CouponPreflight.isPassedToday(startUserId)) return true

        try {
            val start = CouponPreflight.start(startUserId, cached?.second ?: "")
            if (start.allowed || !start.required) {
                CouponPreflight.markPassed(startUserId, start.dayKey)
                return true
            }
            val probe = start.probe
            if (probe == null || probe.url.isBlank()) {
                updateFloatStatus("优惠券检测商品缺失，跳过本次检测")
                return true
            }

            // 首次跳转到检测链接，抓包
            updateFloatStatus("开始优惠券数据检测...")
            pendingFileContent = null
            launchProductUrl(probe.url)
            val got = waitForData(30_000L)
            if (!got) {
                stopAutoForCoupon("优惠券检测超时，已停止")
                return false
            }
            val raw = pendingFileContent ?: ""
            pendingFileContent = null

            // 解密取出响应体：检查优惠券信息 + 登录信息
            val responseBody = extractEncryptedData(raw)
            val account = CouponPreflight.extractAccount(raw)
                ?: CouponPreflight.extractAccount(responseBody)
                ?: (startUserId to "")
            val (userId, username) = account
            CouponPreflight.saveAccount(userId, username)

            val report = CouponPreflight.report(userId, username, probe.id, responseBody)
            if (!report.passed) {
                stopAutoForCoupon("优惠券检测未通过: ${report.reason.ifBlank { "数据异常" }}")
                return false
            }
            CouponPreflight.markPassed(userId, report.dayKey)
            updateFloatStatus("优惠券检测通过")
            return true
        } catch (e: Exception) {
            Log.w(TAG, "Coupon preflight failed", e)
            stopAutoForCoupon("优惠券预检失败: ${e.message}")
            return false
        }
    }

    private suspend fun launchProductUrl(url: String) {
        val wakeIntent = Intent(ApiConfig.WAKE_ACTION).apply {
            setPackage(ApiConfig.TARGET_PACKAGE)
            putExtra("url", url)
        }
        sendBroadcast(wakeIntent)
        delay(1000)
        launchVisible(url)
    }

    private fun stopAutoForCoupon(reason: String) {
        isAutoRunning = false
        isUploading = false
        countdownJob?.cancel()
        pauseJob?.cancel()
        handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
        updateFloatStatus(reason)
    }

    private fun claimAndExecuteTask() {
        if (!deviceAllowed) {
            stopAutoForDeviceBlock()
            return
        }
        serviceScope.launch {
            sendHeartbeat()
            if (!deviceAllowed) return@launch
            claimAndExecuteTask1()
        }
    }

    private fun claimAndExecuteTask1() {
        serviceScope.launch {
            try {
                if (!awaitCouponPreflight()) return@launch

                val taskResponse = ApiService.takeTask()
                if (!taskResponse.isSuccess || taskResponse.task == null) {
                    val rawMsg = taskResponse.message
                    val msg = if (rawMsg.contains("fetchcool", ignoreCase = true)
                        || rawMsg.contains("cooldown", ignoreCase = true)) "暂无任务" else rawMsg
                    updateFloatStatus("领取任务失败：$msg")
                    if (msg.contains("登录") || msg.contains("token") || msg.contains("expired")) {
                        autoReLogin()
                    } else if (isAutoRunning) {
                        val retrySeconds = taskResponse.retryAfterSeconds.coerceIn(0, 300)
                        if (retrySeconds > 0) startNextTaskCycle(forceDelaySec = retrySeconds)
                        else startNextTaskCycle()
                    }
                    return@launch
                }

                val task = taskResponse.task
                currentTask = task
                currentTaskId = task.id()
                currentShopId = task.parseShopId()
                currentItemId = task.parseItemId()
                pendingFileContent = null
                usedChatFlow = false

                updateFloatStatus("已领取任务\nshop=$currentShopId item=$currentItemId")
                broadcastStats()
                delay(800)

                // 聊一聊模式：发送链接到当前聊天并点击打开商品页（环境App照常抓包）；不可用则回退普通流程
                val chatWanted = chatMode && !silentMode
                if (chatWanted) {
                    usedChatFlow = ChatAuto.openTaskLinkInChat(
                        logger = { updateFloatStatus(it) },
                        link = "https://shopee.tw/product/$currentShopId/$currentItemId"
                    )
                    if (!usedChatFlow) {
                        updateFloatStatus("聊一聊不可用，回退普通流程打开商品页")
                        launchShopeeProduct()
                    }
                } else {
                    launchShopeeProduct()
                }

                updateFloatStatus("等待数据回传...")
                val gotContent = waitForData()
                if (!gotContent) {
                    recordDataTimeout()
                    // 回传超时/失败也要先回聊天页，否则下个任务会在商品页上开始。
                    // 用 isUploading 挡住倒计时，避免还没回到聊天页就开始领下一个任务。
                    isUploading = true
                    returnToChatIfNeeded("回传失败，返回聊天页...")
                    isUploading = false
                    return@launch
                }
                consecutiveTimeouts = 0

                val content = pendingFileContent!!
                pendingFileContent = null
                isUploading = true
                updateFloatStatus("正在上传数据...")
                submitTaskResult1(content)

            } catch (e: Exception) {
                Log.e(TAG, "Task1 error", e)
                updateFloatStatus("任务异常: ${e.message}")
                isUploading = false
                if (isAutoRunning) startNextTaskCycle()
            }
        }
    }

    private suspend fun launchShopeeProduct() {
        val productUrl = "https://shopee.tw/product/$currentShopId/$currentItemId"
        val wakeIntent = Intent(ApiConfig.WAKE_ACTION).apply {
            setPackage(ApiConfig.TARGET_PACKAGE)
            putExtra("url", productUrl)
        }
        sendBroadcast(wakeIntent)
        Log.d(TAG, "Wake broadcast: $productUrl")

        if (!silentMode) {
            delay(1000)
            launchVisible(productUrl)
            return
        }

        // 静默模式：只发唤醒广播，环境App内部路由打开商品页并抓包
        updateFloatStatus("静默模式：已通知环境抓取，不跳转页面...")
        val silentWaitMs = 15_000L
        val start = System.currentTimeMillis()
        while (System.currentTimeMillis() - start < silentWaitMs) {
            if (pendingFileContent != null) {
                updateFloatStatus("静默抓取成功，数据已回传")
                return
            }
            delay(300)
        }
        // 静默等待超时未回传，回退前台跳转
        updateFloatStatus("静默模式未回传，回退前台跳转...")
        delay(1000)
        launchVisible(productUrl)
    }

    private fun launchVisible(productUrl: String) {
        try {
            val launchIntent = Intent(Intent.ACTION_VIEW).apply {
                data = Uri.parse(productUrl)
                setPackage(ApiConfig.TARGET_PACKAGE)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            }
            startActivity(launchIntent)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to launch Shopee", e)
            updateFloatStatus("启动Shopee失败: ${e.message}")
            if (isAutoRunning) startNextTaskCycle()
        }
    }

    // 纯等待数据回传（不做超时计数等副作用）
    private suspend fun waitForData(timeoutMs: Long = 20_000L): Boolean {
        val start = System.currentTimeMillis()
        while (System.currentTimeMillis() - start < timeoutMs) {
            if (pendingFileContent != null) return true
            delay(500)
        }
        return false
    }

    // 记录一次“超时未回传”，连续 2 次则暂停并上报风控
    private fun recordDataTimeout() {
        consecutiveTimeouts++
        if (consecutiveTimeouts >= 2) {
            isAutoRunning = false
            handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
            if (!riskReported) {
                riskReported = true
                serviceScope.launch {
                    try { ApiService.reportDeviceRisk("连续多次回传超时，疑似风控") } catch (_: Exception) {}
                }
            }
            updateFloatStatus("该设备已触发风控，请及时更换账号\n（连续${consecutiveTimeouts}次超时，已自动暂停）")
            consecutiveTimeouts = 0
        } else {
            updateFloatStatus("数据回传超时\n连续超时${consecutiveTimeouts}次，3秒后重试")
            if (isAutoRunning) startNextTaskCycle(forceDelaySec = 3)
        }
    }

    private suspend fun submitTaskResult1(fileContent: String) {
        try {
            if (!deviceAllowed) {
                stopAutoForDeviceBlock()
                return
            }
            val rawResponse = extractEncryptedData(fileContent)
            val task = currentTask
            if (task == null) {
                failedCount++
                updateFloatStatus("上传失败：任务信息缺失")
                returnToChatIfNeeded("返回聊天页...")
                return
            }

            val pdpUrl = "https://shopee.tw/api/v4/pdp/get_pc?" +
                "display_model_id=0&item_id=$currentItemId&model_selection_logic=3" +
                "&shop_id=$currentShopId&tz_offset_in_minutes=480&detail_level=0"

            val response = ApiService.submitTask(
                taskId = currentTaskId,
                taskPageUrl = task.pageUrl(),
                submitUrl = pdpUrl,
                rawResponse = rawResponse,
                task = task
            )
            if (response.isSuccess) {
                uploadCount++
                sessionUploadCount++
                Log.d(TAG, "Submit OK: task=$currentTaskId count=$uploadCount")
                // 聊一聊：上传成功后必须回到聊天页，否则下个任务会在商品页上开始
                if (returnToChatIfNeeded("上传成功！返回聊天页...")) {
                    updateFloatStatus("上传成功！已返回聊天")
                } else {
                    updateFloatStatus("上传成功！")
                }
            } else {
                failedCount++
                updateFloatStatus("上传失败: ${response.message}")
                // 上传失败也要回聊天页，从聊天页继续下一个任务
                returnToChatIfNeeded("返回聊天页...")
            }
        } catch (e: Exception) {
            failedCount++
            updateFloatStatus("上传异常: ${e.message}")
            returnToChatIfNeeded("异常，返回聊天页...")
        } finally {
            isUploading = false
            currentTaskId = ""
            currentTask = null
            broadcastStats()
            if (isAutoRunning && !triggerPauseRestIfNeeded()) startNextTaskCycle()
        }
    }

    /**
     * 聊一聊：回到聊天页。
     * 成功/失败/超时都要回，否则下一个任务会在商品页（或搜图页）上开始，
     * 现象就是面板打出「聊一聊：未找到输入框」然后回退成普通跳转。
     * @return true 表示这次确实执行了返回
     */
    private suspend fun returnToChatIfNeeded(reason: String): Boolean {
        if (!chatMode || !usedChatFlow || !ChatAccessibilityService.isReady()) return false
        usedChatFlow = false
        updateFloatStatus(reason)
        ChatAuto.returnToChat(logger = { updateFloatStatus(it) })
        return true
    }

    private fun triggerPauseRestIfNeeded(): Boolean {
        if (!pauseEnabled) return false
        val threshold = pauseCount.coerceAtLeast(1)
        // 休息时长严格按用户设置，不做随机浮动
        val minutes = pauseMinutes.coerceAtLeast(1)
        if (sessionUploadCount < threshold) return false

        sessionUploadCount = 0
        countdownJob?.cancel()
        pauseJob?.cancel()
        pauseJob = serviceScope.launch {
            val totalSec = minutes * 60
            for (i in totalSec downTo 1) {
                if (!isAutoRunning) return@launch
                val mm = i / 60
                val ss = i % 60
                updateFloatStatus("已完成${threshold}个，休息中... ${mm}分${ss}秒")
                delay(1000)
            }
            if (isAutoRunning) {
                updateFloatStatus("休息结束，继续任务")
                startNextTaskCycle()
            }
        }
        return true
    }

    private fun extractEncryptedData(fileContent: String): String {        return try {
            val json = JSONObject(fileContent)
            json.optString("response_body", "").ifEmpty { fileContent }
        } catch (_: Exception) {
            fileContent
        }
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = serviceScope.launch {
            while (isActive) {
                sendHeartbeat()
                delay(30_000)
            }
        }
    }

    private suspend fun sendHeartbeat() {
        try {
            val response = ApiService.heartbeat()
            if (!response.ok || !response.allowed) {
                deviceAllowed = false
                deviceBlockedMsg = response.msg ?: "设备已被禁用"
                stopAutoForDeviceBlock()
            } else {
                deviceAllowed = true
                if (response.riskControl) {
                    if (!riskReported) {
                        riskReported = true
                        updateFloatStatus(response.riskMsg.ifBlank { "该设备已触发风控，请及时更换账号" })
                    }
                } else {
                    riskReported = false
                }
            }
        } catch (e: Exception) {
            Log.w(TAG, "Heartbeat failed", e)
        }
    }

    private fun stopAutoForDeviceBlock() {
        isAutoRunning = false
        isUploading = false
        countdownJob?.cancel()
        pauseJob?.cancel()
        handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
        updateFloatStatus(deviceBlockedMsg)
    }

    private fun startUpdateCheck() {
        updateCheckJob?.cancel()
        updateCheckJob = serviceScope.launch {
            delay(5_000L)
            while (isActive) {
                try {
                    val currentCode = packageManager.getPackageInfo(packageName, 0).let {
                        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) it.longVersionCode.toInt()
                        else @Suppress("DEPRECATION") it.versionCode
                    }
                    val update = ApiService.checkUpdate(currentCode)
                    if (update.ok && update.blocked) {
                        versionBlocked = true
                        versionBlockedMsg = update.blockedMsg.ifBlank { "版本已停用，请更新" }
                        if (isAutoRunning) {
                            isAutoRunning = false
                            countdownJob?.cancel()
                            handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
                            updateFloatStatus("$versionBlockedMsg\n已停止自动执行")
                        }
                    } else {
                        versionBlocked = false
                    }
                    if (update.ok && (update.hasUpdate || update.blocked) && update.apkUrl.isNotBlank()) {
                        val intent = Intent("com.app.helper.UPDATE_ACTION").apply {
                            putExtra("version_name", update.versionName)
                            putExtra("apk_url", update.apkUrl)
                            putExtra("title", if (update.blocked) "版本已停用" else update.title)
                            putExtra("message", if (update.blocked) update.blockedMsg else update.message)
                            putExtra("force", update.force || update.blocked)
                            putExtra("blocked", update.blocked)
                        }
                        sendBroadcast(intent)
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "Update check failed", e)
                }
                delay(30 * 60_000L)
            }
        }
    }

    // ==================== Auto Task Logic ====================

    private suspend fun autoReLogin() {
        updateFloatStatus("登录过期，正在重新登录...")
        try {
            val accountId = TokenManager.getUsername()
            if (accountId.isNotEmpty()) {
                val resp = ApiService.login(accountId)
                if (resp.isSuccess && resp.data != null) {
                    TokenManager.saveToken(resp.data.token)
                    TokenManager.saveUsername(resp.data.accountId.ifBlank { accountId })
                    TokenManager.saveGroupId(resp.data.groupId)
                    TokenManager.saveAccountCount(
                        if (resp.data.api1TaskAccountCount > 0) resp.data.api1TaskAccountCount else resp.data.taskAccountCount
                    )
                    updateFloatStatus("重新登录成功")
                    if (isAutoRunning) startNextTaskCycle()
                    return
                }
            }
        } catch (_: Exception) {}
        isAutoRunning = false
        handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
        updateFloatStatus("登录失效，请重新登录")
    }

    // ==================== Broadcast Receiver ====================

    private fun registerDataReceiver() {
        receiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                when (intent.action) {
                    "com.app.helper.FILE_ACTION" -> {
                        val extraData = intent.getStringExtra("extra_data") ?: ""
                        val extraPath = intent.getStringExtra("extra_data_path") ?: ""
                        val taskId = intent.getStringExtra("task_id") ?: ""
                        Log.d(TAG, "FILE_ACTION: uri=$extraData taskId=$taskId")
                        val content = readFileContent(extraData, extraPath)
                        if (content != null) {
                            pendingFileContent = content
                            CouponPreflight.extractAccount(content)?.let {
                                CouponPreflight.saveAccount(it.first, it.second)
                            }
                            Log.d(TAG, "Data file read: ${content.length} bytes")
                        }
                    }
                    "com.app.helper.TOAST_ACTION" -> {
                        val msg = intent.getStringExtra("extra_data") ?: ""
                        if (msg.isNotEmpty()) updateFloatStatus(msg)
                    }
                    "com.app.helper.VER_ACTION" -> {
                        val ver = intent.getStringExtra("extra_data") ?: ""
                        if (ver.isNotBlank()) {
                            targetVersion = ver
                            broadcastStats()
                        }
                        Log.d(TAG, "Target version: $ver")
                    }
                    "com.app.helper.LOGS_ACTION" -> {
                        Log.d(TAG, "Target log: ${intent.getStringExtra("extra_data")}")
                    }
                    "com.app.helper.SHOW_FLOAT_ACTION" -> {
                        handler.post { tvAction?.visibility = View.VISIBLE }
                    }
                    "com.app.helper.STATS_QUERY" -> {
                        broadcastStats()
                    }
                }
            }
        }
        val filter = IntentFilter().apply {
            addAction("com.app.helper.FILE_ACTION")
            addAction("com.app.helper.TOAST_ACTION")
            addAction("com.app.helper.VER_ACTION")
            addAction("com.app.helper.LOGS_ACTION")
            addAction("com.app.helper.SHOW_FLOAT_ACTION")
            addAction("com.app.helper.STATS_QUERY")
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(receiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(receiver, filter)
        }
    }

    private fun unregisterDataReceiver() {
        receiver?.let { try { unregisterReceiver(it) } catch (_: Exception) {} }
    }

    private fun readFileContent(uriStr: String, path: String): String? {
        if (uriStr.startsWith("content://")) {
            try {
                contentResolver.openInputStream(Uri.parse(uriStr))?.use {
                    return it.bufferedReader().readText()
                }
            } catch (e: Exception) { Log.w(TAG, "URI read failed", e) }
        }
        if (path.isNotEmpty()) {
            try {
                var file = java.io.File(path)
                if (file.exists()) return file.readText()
                val downloads = android.os.Environment.getExternalStoragePublicDirectory(
                    android.os.Environment.DIRECTORY_DOWNLOADS)
                file = java.io.File(downloads, path.substringAfterLast("/"))
                if (file.exists()) return file.readText()
            } catch (e: Exception) { Log.w(TAG, "File read failed", e) }
        }
        return null
    }

    // ==================== Notification ====================

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(CHANNEL_ID, "应用后台服务", NotificationManager.IMPORTANCE_LOW)
            (getSystemService(NOTIFICATION_SERVICE) as NotificationManager).createNotificationChannel(channel)
        }
    }

    private fun createNotification(): Notification {
        val b = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) Notification.Builder(this, CHANNEL_ID)
            else @Suppress("DEPRECATION") Notification.Builder(this)
        return b.setContentTitle("任务助手运行中").setContentText("正在监听广播消息")
            .setSmallIcon(android.R.drawable.ic_menu_upload).build()
    }
}
