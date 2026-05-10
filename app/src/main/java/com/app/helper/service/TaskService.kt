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
import com.app.helper.network.Api2TaskInfo
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
    private var countdownJob: Job? = null

    private var touchDownX = 0f
    private var touchDownY = 0f
    private var windowX = 0
    private var windowY = 0

    private var receiver: BroadcastReceiver? = null
    @Volatile private var pendingFileContent: String? = null
    private var currentApi2TaskInfo: Api2TaskInfo? = null

    private val prefs by lazy { getSharedPreferences(PREF_NAME, MODE_PRIVATE) }
    private val delayMin get() = prefs.getInt("delay_min", 3)
    private val delayMax get() = prefs.getInt("delay_max", 8)

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        isRunning = true
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
        registerDataReceiver()
        createFloatWindow()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            "SET_DELAY" -> {
                val min = intent.getIntExtra("delay_min", 3)
                val max = intent.getIntExtra("delay_max", 8)
                prefs.edit().putInt("delay_min", min).putInt("delay_max", max).apply()
                updateFloatStatus("延时已设置: ${min}-${max}秒")
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
        val apiTag = if (TokenManager.getApiType() == TokenManager.API_TYPE_2) "[接口二]" else "[接口一]"
        statusMessage = "$message\n$apiTag 账号：${TokenManager.getUsername()}\n当日上传成功：$uploadCount"
        handler.post { if (!isCollapsed) tvStatus?.text = statusMessage }
    }

    private fun toggleAutoTask() {
        if (isUploading) { updateFloatStatus("正在上传中..."); return }
        if (isAutoRunning) {
            isAutoRunning = false
            countdownJob?.cancel()
            handler.post { tvAction?.text = "一键做任务" }
            updateFloatStatus("已停止自动执行")
        } else {
            isAutoRunning = true
            handler.post { tvAction?.text = "停止自动" }
            updateFloatStatus("自动执行已开启")
            startNextTaskCycle()
        }
    }

    // ==================== Task Flow ====================

    private fun startNextTaskCycle() {
        if (!isAutoRunning) return
        val min = delayMin.coerceAtLeast(1)
        val max = delayMax.coerceAtLeast(min)
        val delaySec = (min..max).random()

        countdownJob?.cancel()
        countdownJob = serviceScope.launch {
            for (i in delaySec downTo 1) {
                if (!isAutoRunning) return@launch
                if (isUploading) { delay(1000); continue }
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
        if (TokenManager.getApiType() == TokenManager.API_TYPE_2) {
            claimAndExecuteTask2()
        } else {
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
                currentApi2TaskInfo = null
                pendingFileContent = null

                updateFloatStatus("已领取任务\nshop=$currentShopId item=$currentItemId")
                delay(800)
                launchShopeeProduct()

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

    private fun claimAndExecuteTask2() {
        serviceScope.launch {
            try {
                val groupId = TokenManager.getGroupId()
                val taskResponse = ApiService.takeTask2(groupId)
                if (!taskResponse.ok || taskResponse.taskData == null) {
                    updateFloatStatus("接口二领取任务失败")
                    if (isAutoRunning) startNextTaskCycle()
                    return@launch
                }

                val taskData = taskResponse.taskData
                currentApi2TaskInfo = taskData.task
                currentShopId = taskData.parseShopId()
                currentItemId = taskData.parseItemId()
                currentTaskId = taskData.task?.id ?: ""
                pendingFileContent = null

                updateFloatStatus("[接口二] 已领取任务\nshop=$currentShopId item=$currentItemId")
                delay(800)
                launchShopeeProduct()

                updateFloatStatus("等待数据回传...")
                if (!waitForFileContent()) return@launch

                val content = pendingFileContent!!
                pendingFileContent = null
                isUploading = true
                updateFloatStatus("正在上传数据...")
                submitTaskResult2(content)

            } catch (e: Exception) {
                Log.e(TAG, "Task2 error", e)
                updateFloatStatus("接口二任务异常: ${e.message}")
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
        delay(1000)
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
            if (pendingFileContent != null) return true
            delay(500)
        }
        updateFloatStatus("数据回传超时(${timeout/1000}s)")
        if (isAutoRunning) startNextTaskCycle()
        return false
    }

    private suspend fun submitTaskResult1(fileContent: String) {
        try {
            val responseBody = try {
                JSONObject(fileContent).optString("response_body", fileContent)
            } catch (_: Exception) { fileContent }

            val pdpUrl = "https://shopee.tw/api/v4/pdp/get_pc?" +
                "display_model_id=0&item_id=$currentItemId&model_selection_logic=3" +
                "&shop_id=$currentShopId&tz_offset_in_minutes=480&detail_level=0"

            val submitResponse = ApiService.submitTask(pdpUrl, responseBody)
            if (submitResponse.isSuccess && submitResponse.data?.isSuccess == true) {
                uploadCount++
                updateFloatStatus("上传成功！")
                Log.d(TAG, "Submit OK: task=$currentTaskId count=$uploadCount")
            } else {
                val msg = submitResponse.data?.msg ?: submitResponse.msg ?: "提交失败"
                updateFloatStatus("上传失败: $msg")
            }
        } catch (e: Exception) {
            updateFloatStatus("上传异常: ${e.message}")
        } finally {
            isUploading = false
            currentTaskId = ""
            if (isAutoRunning) startNextTaskCycle()
        }
    }

    private suspend fun submitTaskResult2(fileContent: String) {
        try {
            val responseBody = try {
                val json = JSONObject(fileContent)
                val body = json.optString("response_body", "")
                if (body.isNotEmpty()) JSONObject(body) else json
            } catch (_: Exception) {
                try { JSONObject(fileContent) } catch (_: Exception) { fileContent }
            }

            val taskInfo = currentApi2TaskInfo ?: Api2TaskInfo()
            val response = ApiService.submitTask2(
                username = TokenManager.getUsername(),
                groupId = TokenManager.getGroupId(),
                taskInfo = taskInfo,
                data = responseBody
            )
            if (response.ok) {
                uploadCount++
                updateFloatStatus("[接口二] 上传成功！")
                Log.d(TAG, "Submit2 OK: task=$currentTaskId count=$uploadCount")
            } else {
                updateFloatStatus("[接口二] 上传失败: ${response.msg ?: "未知错误"}")
            }
        } catch (e: Exception) {
            updateFloatStatus("[接口二] 上传异常: ${e.message}")
        } finally {
            isUploading = false
            currentTaskId = ""
            currentApi2TaskInfo = null
            if (isAutoRunning) startNextTaskCycle()
        }
    }

    private suspend fun autoReLogin() {
        if (TokenManager.getApiType() == TokenManager.API_TYPE_2) {
            updateFloatStatus("[接口二] 无需重新登录，继续执行...")
            if (isAutoRunning) startNextTaskCycle()
            return
        }
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
        handler.post { tvAction?.text = "一键做任务" }
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
                        Log.d(TAG, "Target version: ${intent.getStringExtra("extra_data")}")
                    }
                    "com.app.helper.LOGS_ACTION" -> {
                        Log.d(TAG, "Target log: ${intent.getStringExtra("extra_data")}")
                    }
                    "com.app.helper.SHOW_FLOAT_ACTION" -> {
                        handler.post { tvAction?.visibility = View.VISIBLE }
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
