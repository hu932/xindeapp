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
import android.provider.Settings
import android.util.Log
import android.view.Gravity
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import com.app.helper.R
import com.app.helper.chat.ChatAuto
import com.app.helper.network.ApiConfig
import com.app.helper.network.ApiService
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
    private var isCollapsed = false

    private var isAutoRunning = false
    private var isUploading = false
    private var currentTaskId = ""
    private var currentShopId = 0L
    private var currentItemId = 0L
    private var statusMessage = ""
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

    private var touchDownX = 0f
    private var touchDownY = 0f
    private var windowX = 0
    private var windowY = 0

    private var receiver: BroadcastReceiver? = null
    @Volatile private var pendingFileContent: String? = null

    private val prefs by lazy { getSharedPreferences(PREF_NAME, MODE_PRIVATE) }
    private val delayMin get() = prefs.getInt("delay_min", 15)
    private val delayMax get() = prefs.getInt("delay_max", 35)
    private val pauseEnabled get() = prefs.getBoolean("pause_enabled", true)
    private val pauseCount get() = prefs.getInt("pause_count", 30)
    private val pauseMinutes get() = prefs.getInt("pause_minutes", 30)
    private val silentMode get() = prefs.getBoolean("silent_mode", false)
    private val chatMode get() = prefs.getBoolean("chat_mode", false)
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
                val min = intent.getIntExtra("delay_min", 15)
                val max = intent.getIntExtra("delay_max", 35)
                prefs.edit().putInt("delay_min", min).putInt("delay_max", max).apply()
                updateFloatStatus("延时已设置: ${min}-${max}秒")
            }
            "SET_PAUSE" -> {
                val enabled = intent.getBooleanExtra("pause_enabled", true)
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

    private fun createFloatWindow() {
        windowManager = getSystemService(WINDOW_SERVICE) as WindowManager
        floatView = LayoutInflater.from(this).inflate(R.layout.layout_window_float, null)

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

        tvStatus = floatView?.findViewById(R.id.tv_status)
        tvAction = floatView?.findViewById(R.id.tv_action)
        tvDot = floatView?.findViewById(R.id.tv_dot)
        llDotContainer = floatView?.findViewById(R.id.ll_dot_container)
        llExpanded = floatView?.findViewById(R.id.ll_expanded)
        imgClose = floatView?.findViewById(R.id.img_close)

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

        try { windowManager?.addView(floatView, floatParams) } catch (e: Exception) { Log.e(TAG, "Float window failed", e) }
        updateFloatStatus("点击【一键做任务】开始\n延时: ${delayMin}-${delayMax}秒")
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

    private fun updateFloatStatus(message: String) {
        val compactMessage = message
            .replace("自动领取任务", "领任务")
            .replace("自动执行", "自动")
            .replace("当日上传成功", "今日")
        statusMessage = "$compactMessage\n账号:${TokenManager.getUsername()} 今日:$uploadCount"
        handler.post { if (!isCollapsed) tvStatus?.text = statusMessage }
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
            val min = delayMin.coerceAtLeast(0)
            val max = delayMax.coerceAtLeast(min)
            var ms = (min..max).random() * 1000L
            // 拟人化：25% 概率延长停留，模拟真实浏览节奏，降低风控
            if (kotlin.random.Random.nextInt(100) < 25) {
                ms = (ms * (1.5 + kotlin.random.Random.nextDouble() * 1.5)).toLong()
            }
            ms
        }
        val remainingDelayMs = (delayMs - alreadyWaitedMs).coerceAtLeast(0L)
        val tipPrefix = if (forceDelaySec != null) "风控重试" else "自动领取任务"

        countdownJob?.cancel()
        countdownJob = serviceScope.launch {
            var remainingMs = remainingDelayMs
            while (remainingMs > 0L) {
                if (!isAutoRunning) return@launch
                if (isUploading) { delay(1000); continue }
                val displaySec = ((remainingMs + 999L) / 1000L).toInt()
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
                val taskResponse = ApiService.takeTask()
                if (!taskResponse.isSuccess || taskResponse.data == null) {
                    val msg = taskResponse.msg ?: "获取任务失败"
                    updateFloatStatus("领取任务失败：$msg")
                    if (msg.contains("登录") || msg.contains("token") || msg.contains("expired")) {
                        autoReLogin()
                    } else if (isAutoRunning) { startNextTaskCycle() }
                    return@launch
                }

                val task = taskResponse.data
                currentTaskId = task.taskId
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
                if (!waitForFileContent()) return@launch

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

    private suspend fun waitForFileContent(): Boolean {
        val timeout = 90_000L
        val start = System.currentTimeMillis()
        while (System.currentTimeMillis() - start < timeout) {
            if (pendingFileContent != null) {
                consecutiveTimeouts = 0
                return true
            }
            delay(500)
        }
        consecutiveTimeouts++
        if (consecutiveTimeouts >= 2) {
            isAutoRunning = false
            handler.post { tvAction?.text = "一键做任务"; tvAction?.setBackgroundResource(R.drawable.bg_float_action) }
            updateFloatStatus("疑似风控！连续${consecutiveTimeouts}次超时\n已自动暂停，请检查环境")
            consecutiveTimeouts = 0
            return false
        }
        updateFloatStatus("数据回传超时(${timeout/1000}s)\n连续超时${consecutiveTimeouts}次，10秒后重试")
        if (isAutoRunning) startNextTaskCycle(forceDelaySec = 10)
        return false
    }

    private suspend fun submitTaskResult1(fileContent: String) {
        try {
            if (!deviceAllowed) {
                stopAutoForDeviceBlock()
                return
            }
            val encryptedData = extractEncryptedData(fileContent)

            val pdpUrl = "https://shopee.tw/api/v4/pdp/get_pc?" +
                "display_model_id=0&item_id=$currentItemId&model_selection_logic=3" +
                "&shop_id=$currentShopId&tz_offset_in_minutes=480&detail_level=0"

            val response = ApiService.submitTask(
                taskUrl = pdpUrl,
                encryptedData = encryptedData,
                username = TokenManager.getUsername(),
                groupId = TokenManager.getGroupId(),
                deviceId = getAppDeviceId()
            )
            if (response.isApi1Success) {
                uploadCount++
                sessionUploadCount++
                Log.d(TAG, "Submit OK: task=$currentTaskId count=$uploadCount")
                // 聊一聊：上传成功后智能返回聊天页（识别并关闭搜图页后再返回）
                if (chatMode && usedChatFlow && ChatAccessibilityService.isReady()) {
                    updateFloatStatus("上传成功！返回聊天页...")
                    ChatAuto.returnToChat(logger = { updateFloatStatus(it) })
                    usedChatFlow = false
                    updateFloatStatus("上传成功！已返回聊天")
                } else {
                    updateFloatStatus("上传成功！")
                }
            } else {
                failedCount++
                updateFloatStatus("上传失败: ${response.msg ?: "未知错误"}")
            }
        } catch (e: Exception) {
            failedCount++
            updateFloatStatus("上传异常: ${e.message}")
        } finally {
            isUploading = false
            currentTaskId = ""
            broadcastStats()
            if (isAutoRunning && !triggerPauseRestIfNeeded()) startNextTaskCycle()
        }
    }

    private fun triggerPauseRestIfNeeded(): Boolean {
        if (!pauseEnabled) return false
        val threshold = pauseCount.coerceAtLeast(1)
        // 休息时长随机 ±25%，避免固定周期被风控识别
        val minutes = (pauseMinutes * (0.75 + kotlin.random.Random.nextDouble() * 0.5)).toInt().coerceAtLeast(3)
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
            val response = ApiService.heartbeat(
                username = TokenManager.getUsername(),
                groupId = TokenManager.getGroupId(),
                apiType = TokenManager.getApiType(),
                deviceId = getAppDeviceId()
            )
            if (!response.ok || !response.allowed) {
                deviceAllowed = false
                deviceBlockedMsg = response.msg ?: "设备已被禁用"
                stopAutoForDeviceBlock()
            } else {
                deviceAllowed = true
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

    private fun getAppDeviceId(): String {
        val androidId = Settings.Secure.getString(contentResolver, Settings.Secure.ANDROID_ID)
        return androidId?.takeIf { it.isNotBlank() } ?: "android-${Build.MODEL}"
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
            val username = TokenManager.getUsername()
            val password = prefs.getString("saved_password", "") ?: ""
            if (username.isNotEmpty() && password.isNotEmpty()) {
                val resp = ApiService.login(username, password)
                if (resp.isSuccess && resp.data != null) {
                    TokenManager.saveToken(resp.data.token)
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
