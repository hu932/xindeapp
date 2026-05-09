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
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import com.app.helper.R
import com.app.helper.network.ApiConfig
import com.app.helper.network.ApiService
import com.app.helper.util.TokenManager
import kotlinx.coroutines.*
import org.json.JSONObject
import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class TaskService : android.app.Service() {

    companion object {
        private const val TAG = "TaskService"
        private const val CHANNEL_ID = "app_service_channel"
        private const val NOTIFICATION_ID = 1001
        var isRunning = false
            private set
    }

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val handler = Handler(Looper.getMainLooper())

    // Float window
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

    // Task state
    private var isAutoRunning = false
    private var isUploading = false
    private var currentTaskId = ""
    private var currentShopId = 0L
    private var currentItemId = 0L
    private var currentTaskUrl = ""
    private var statusMessage = ""
    private var uploadCount = 0
    private var countdownJob: Job? = null

    // Touch drag state
    private var touchDownX = 0f
    private var touchDownY = 0f
    private var windowX = 0
    private var windowY = 0

    // Broadcast receiver
    private var receiver: BroadcastReceiver? = null
    // Pending file content
    @Volatile
    private var pendingFileContent: String? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        isRunning = true
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
        registerDataReceiver()
        createFloatWindow()
        log("服务启动")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        isRunning = false
        isAutoRunning = false
        serviceScope.cancel()
        countdownJob?.cancel()
        removeFloatWindow()
        unregisterDataReceiver()
        log("服务停止")
    }

    // ==================== Float Window ====================

    private fun createFloatWindow() {
        windowManager = getSystemService(WINDOW_SERVICE) as WindowManager
        floatView = LayoutInflater.from(this).inflate(R.layout.layout_window_float, null)

        val layoutType = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O)
            WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY
        else
            @Suppress("DEPRECATION") WindowManager.LayoutParams.TYPE_PHONE

        floatParams = WindowManager.LayoutParams(
            WindowManager.LayoutParams.WRAP_CONTENT,
            WindowManager.LayoutParams.WRAP_CONTENT,
            layoutType,
            WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL,
            PixelFormat.TRANSLUCENT
        ).apply {
            gravity = Gravity.TOP or Gravity.START
            x = 0
            y = 200
        }

        tvStatus = floatView?.findViewById(R.id.tv_status)
        tvAction = floatView?.findViewById(R.id.tv_action)
        tvDot = floatView?.findViewById(R.id.tv_dot)
        llDotContainer = floatView?.findViewById(R.id.ll_dot_container)
        llExpanded = floatView?.findViewById(R.id.ll_expanded)
        imgClose = floatView?.findViewById(R.id.img_close)

        // Drag support on status text
        tvStatus?.setOnTouchListener { _, event ->
            when (event.action) {
                MotionEvent.ACTION_DOWN -> {
                    touchDownX = event.rawX
                    touchDownY = event.rawY
                    windowX = floatParams?.x ?: 0
                    windowY = floatParams?.y ?: 0
                    true
                }
                MotionEvent.ACTION_MOVE -> {
                    floatParams?.x = windowX + (event.rawX - touchDownX).toInt()
                    floatParams?.y = windowY + (event.rawY - touchDownY).toInt()
                    windowManager?.updateViewLayout(floatView, floatParams)
                    true
                }
                else -> false
            }
        }

        // Drag support on dot
        llDotContainer?.setOnTouchListener { _, event ->
            when (event.action) {
                MotionEvent.ACTION_DOWN -> {
                    touchDownX = event.rawX
                    touchDownY = event.rawY
                    windowX = floatParams?.x ?: 0
                    windowY = floatParams?.y ?: 0
                    false
                }
                MotionEvent.ACTION_MOVE -> {
                    val dx = Math.abs(event.rawX - touchDownX)
                    val dy = Math.abs(event.rawY - touchDownY)
                    if (dx > 5 || dy > 5) {
                        floatParams?.x = windowX + (event.rawX - touchDownX).toInt()
                        floatParams?.y = windowY + (event.rawY - touchDownY).toInt()
                        windowManager?.updateViewLayout(floatView, floatParams)
                        true
                    } else false
                }
                else -> false
            }
        }

        // Close → collapse
        imgClose?.setOnClickListener { collapseFloat() }

        // Dot → expand
        llDotContainer?.setOnClickListener { expandFloat() }

        // Action button → start/stop
        tvAction?.setOnClickListener { toggleAutoTask() }

        try {
            windowManager?.addView(floatView, floatParams)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to add float window", e)
        }

        updateFloatStatus("点击【一键做任务】开始")
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
        try {
            floatView?.let { windowManager?.removeView(it) }
        } catch (_: Exception) {}
    }

    private fun updateFloatStatus(message: String) {
        statusMessage = message
        val account = TokenManager.getUsername()
        val dateStr = SimpleDateFormat("yyyy-MM-dd", Locale.getDefault()).format(Date())
        val displayText = "$message\n账号：$account\n当日上传成功：$uploadCount"
        handler.post {
            if (!isCollapsed) {
                tvStatus?.text = displayText
            }
        }
    }

    private fun toggleAutoTask() {
        if (isUploading) {
            updateFloatStatus("正在上传中...")
            return
        }

        if (isAutoRunning) {
            // Stop
            isAutoRunning = false
            countdownJob?.cancel()
            handler.post { tvAction?.text = "一键做任务" }
            updateFloatStatus("已停止自动执行")
        } else {
            // Start
            isAutoRunning = true
            handler.post { tvAction?.text = "停止自动" }
            updateFloatStatus("自动执行已开启")
            startNextTaskCycle()
        }
    }

    // ==================== Task Flow ====================

    private fun startNextTaskCycle() {
        if (!isAutoRunning) return
        val delayMin = 3
        val delayMax = 8
        val delaySec = (delayMin..delayMax).random()

        countdownJob?.cancel()
        countdownJob = serviceScope.launch {
            for (i in delaySec downTo 1) {
                if (!isAutoRunning) return@launch
                if (isUploading) {
                    delay(1000)
                    continue
                }
                updateFloatStatus("自动领取任务，${i}s后执行...")
                delay(1000)
            }
            if (isAutoRunning && !isUploading) {
                updateFloatStatus("自动领取任务中...")
                claimAndExecuteTask()
            }
        }
    }

    private fun claimAndExecuteTask() {
        serviceScope.launch {
            try {
                // Step 1: Claim task
                val taskResponse = ApiService.takeTask()
                if (!taskResponse.isSuccess || taskResponse.data == null) {
                    val msg = taskResponse.msg ?: "获取任务失败"
                    updateFloatStatus("领取任务失败：$msg")
                    log("领取任务失败: $msg")

                    if (msg.contains("登录") || msg.contains("token") || msg.contains("401")) {
                        autoReLogin()
                    } else if (isAutoRunning) {
                        startNextTaskCycle()
                    }
                    return@launch
                }

                val task = taskResponse.data
                currentTaskId = task.taskId
                currentShopId = task.parseShopId()
                currentItemId = task.parseItemId()
                currentTaskUrl = task.taskUrl
                pendingFileContent = null

                log("领取任务成功: taskId=${task.taskId} shop=$currentShopId item=$currentItemId")
                updateFloatStatus("已领取任务\nshop=$currentShopId\nitem=$currentItemId")

                delay(500)

                // Step 2: Send wake broadcast to target app
                sendWakeBroadcast()

                // Step 3: Wait for FILE_ACTION data
                updateFloatStatus("等待数据回传...")
                val timeout = 60_000L
                val start = System.currentTimeMillis()
                while (System.currentTimeMillis() - start < timeout) {
                    if (pendingFileContent != null) break
                    delay(500)
                }

                if (pendingFileContent == null) {
                    updateFloatStatus("数据回传超时")
                    log("等待数据超时 60s")
                    if (isAutoRunning) startNextTaskCycle()
                    return@launch
                }

                // Step 4: Upload/submit
                val content = pendingFileContent!!
                pendingFileContent = null
                isUploading = true
                updateFloatStatus("正在上传数据...")

                submitTaskResult(content)

            } catch (e: Exception) {
                log("任务异常: ${e.message}")
                updateFloatStatus("任务异常: ${e.message}")
                isUploading = false
                if (isAutoRunning) startNextTaskCycle()
            }
        }
    }

    private suspend fun submitTaskResult(fileContent: String) {
        try {
            val responseBody = extractResponseBody(fileContent)
            val pdpUrl = "https://shopee.tw/api/v4/pdp/get_pc?" +
                "display_model_id=0&item_id=$currentItemId&model_selection_logic=3" +
                "&shop_id=$currentShopId&tz_offset_in_minutes=480&detail_level=0"

            val submitResponse = ApiService.submitTask(pdpUrl, responseBody)
            if (submitResponse.isSuccess && submitResponse.data?.isSuccess == true) {
                uploadCount++
                updateFloatStatus("上传成功！")
                log("上传成功 taskId=$currentTaskId count=$uploadCount")
            } else {
                val msg = submitResponse.data?.msg ?: submitResponse.msg ?: "提交失败"
                updateFloatStatus("上传失败: $msg")
                log("上传失败: $msg")
            }
        } catch (e: Exception) {
            updateFloatStatus("上传异常: ${e.message}")
            log("上传异常: ${e.message}")
        } finally {
            isUploading = false
            currentTaskId = ""
            if (isAutoRunning) startNextTaskCycle()
        }
    }

    private fun extractResponseBody(fileContent: String): String {
        return try {
            val json = JSONObject(fileContent)
            json.optString("response_body", fileContent)
        } catch (_: Exception) {
            fileContent
        }
    }

    private fun sendWakeBroadcast() {
        val intent = Intent(ApiConfig.WAKE_ACTION).apply {
            setPackage(ApiConfig.TARGET_PACKAGE)
            putExtra("url", currentTaskUrl)
        }
        sendBroadcast(intent)
        log("唤醒广播已发送: shop=$currentShopId item=$currentItemId")
    }

    private suspend fun autoReLogin() {
        updateFloatStatus("登录过期，正在重新登录...")
        log("尝试自动重新登录")
        // TODO: Store and re-use credentials if needed
        // For now just stop auto
        if (isAutoRunning) {
            isAutoRunning = false
            handler.post { tvAction?.text = "一键做任务" }
            updateFloatStatus("登录失效，请重新登录")
        }
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
                        log("FILE_ACTION: uri=$extraData taskId=$taskId")

                        val content = readFileContent(extraData, extraPath)
                        if (content != null) {
                            pendingFileContent = content
                            log("数据文件读取成功: ${content.length}字节")
                        } else {
                            log("数据文件读取失败")
                        }
                    }
                    "com.app.helper.TOAST_ACTION" -> {
                        val msg = intent.getStringExtra("extra_data") ?: ""
                        log("TOAST: $msg")
                    }
                    "com.app.helper.VER_ACTION" -> {
                        val ver = intent.getStringExtra("extra_data") ?: ""
                        log("目标APP版本: $ver")
                    }
                    "com.app.helper.LOGS_ACTION" -> {
                        val msg = intent.getStringExtra("extra_data") ?: ""
                        Log.d(TAG, "TARGET_LOG: $msg")
                    }
                    "com.app.helper.SHOW_FLOAT_ACTION" -> {
                        handler.post {
                            tvAction?.visibility = View.VISIBLE
                        }
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
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(receiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(receiver, filter)
        }
    }

    private fun unregisterDataReceiver() {
        receiver?.let {
            try { unregisterReceiver(it) } catch (_: Exception) {}
        }
    }

    private fun readFileContent(uriStr: String, path: String): String? {
        if (uriStr.startsWith("content://")) {
            try {
                contentResolver.openInputStream(Uri.parse(uriStr))?.use {
                    return it.bufferedReader().readText()
                }
            } catch (e: Exception) {
                Log.w(TAG, "URI read failed: $uriStr", e)
            }
        }
        if (path.isNotEmpty()) {
            try {
                // Try direct path
                var file = File(path)
                if (file.exists()) return file.readText()
                // Try Downloads directory
                val downloads = android.os.Environment.getExternalStoragePublicDirectory(
                    android.os.Environment.DIRECTORY_DOWNLOADS
                )
                file = File(downloads, path.substringAfterLast("/"))
                if (file.exists()) return file.readText()
            } catch (e: Exception) {
                Log.w(TAG, "File read failed: $path", e)
            }
        }
        return null
    }

    // ==================== Notification ====================

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(CHANNEL_ID, "应用后台服务", NotificationManager.IMPORTANCE_LOW).apply {
                description = "任务助手后台服务"
            }
            (getSystemService(NOTIFICATION_SERVICE) as NotificationManager).createNotificationChannel(channel)
        }
    }

    private fun createNotification(): Notification {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION") Notification.Builder(this)
        }.setContentTitle("任务助手运行中")
            .setContentText("正在监听广播消息")
            .setSmallIcon(android.R.drawable.ic_menu_upload)
            .build()
    }

    private fun log(msg: String) {
        Log.d(TAG, msg)
    }
}
