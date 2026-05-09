package com.app.helper.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import com.app.helper.network.ApiConfig
import com.app.helper.network.ApiService
import com.app.helper.network.TaskData
import kotlinx.coroutines.*
import org.json.JSONObject

class TaskService : Service() {

    companion object {
        private const val TAG = "TaskService"
        private const val CHANNEL_ID = "task_service_channel"
        private const val NOTIFICATION_ID = 1

        var isRunning = false
            private set
        var currentTask: TaskData? = null
            private set
        var taskStatus: String = "空闲"
            private set
        var completedCount: Int = 0
            private set
        var failedCount: Int = 0
            private set
        var isAutoMode = false

        var onStatusChanged: ((String) -> Unit)? = null
        var onTaskCompleted: ((Boolean, String) -> Unit)? = null
        var onLogMessage: ((String) -> Unit)? = null
    }

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var autoTaskJob: Job? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        isRunning = true
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification("任务助手运行中"))
        log("服务已启动")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            "START_AUTO" -> startAutoTask()
            "STOP_AUTO" -> stopAutoTask()
            "TAKE_SINGLE" -> takeSingleTask()
            "PROCESS_FILE" -> {
                val content = intent.getStringExtra("file_content") ?: ""
                val taskId = intent.getStringExtra("task_id") ?: ""
                processReceivedFile(content, taskId)
            }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        isRunning = false
        serviceScope.cancel()
        log("服务已停止")
    }

    private fun startAutoTask() {
        if (autoTaskJob?.isActive == true) return
        isAutoMode = true
        autoTaskJob = serviceScope.launch {
            log("自动任务模式已启动")
            while (isActive && isAutoMode) {
                try {
                    executeTaskCycle()
                } catch (e: CancellationException) {
                    break
                } catch (e: Exception) {
                    log("任务循环异常: ${e.message}")
                    delay(5000)
                }
            }
            log("自动任务模式已停止")
        }
    }

    private fun stopAutoTask() {
        isAutoMode = false
        autoTaskJob?.cancel()
        autoTaskJob = null
        updateStatus("已停止")
        log("自动任务已停止")
    }

    private fun takeSingleTask() {
        serviceScope.launch {
            try {
                executeTaskCycle()
            } catch (e: Exception) {
                log("单次任务失败: ${e.message}")
                updateStatus("任务失败")
            }
        }
    }

    private suspend fun executeTaskCycle() {
        // Step 1: Take task
        updateStatus("正在获取任务...")
        val taskResponse = ApiService.takeTask()
        if (!taskResponse.isSuccess || taskResponse.data == null) {
            val msg = taskResponse.msg ?: "获取任务失败"
            log("获取任务失败: $msg")
            updateStatus("获取任务失败")
            if (isAutoMode) delay(10000)
            return
        }

        val task = taskResponse.data
        currentTask = task
        val shopId = task.parseShopId()
        val itemId = task.parseItemId()
        log("获取任务成功: taskId=${task.taskId} shopId=$shopId itemId=$itemId")
        updateStatus("已获取任务, 正在唤醒目标APP...")

        // Step 2: Send wake broadcast to target app
        sendWakeBroadcast(task.taskId, shopId, itemId)

        // Step 3: Wait for FILE_ACTION callback (with timeout)
        updateStatus("等待数据回传...")
        val startTime = System.currentTimeMillis()
        val timeout = 60_000L
        var fileProcessed = false

        while (System.currentTimeMillis() - startTime < timeout && !fileProcessed) {
            if (pendingFileContent != null) {
                val content = pendingFileContent!!
                pendingFileContent = null
                fileProcessed = true

                // Step 4: Process and submit
                updateStatus("正在提交数据...")
                submitTaskData(task, shopId, itemId, content)
            }
            delay(500)
        }

        if (!fileProcessed) {
            log("等待数据超时 (${timeout / 1000}秒)")
            updateStatus("数据回传超时")
            failedCount++
            onTaskCompleted?.invoke(false, "超时")
        }

        currentTask = null
        if (isAutoMode) {
            updateStatus("等待下一个任务...")
            delay(3000)
        }
    }

    @Volatile
    private var pendingFileContent: String? = null

    private fun processReceivedFile(content: String, taskId: String) {
        log("收到数据文件, taskId=$taskId, 大小=${content.length}字节")
        pendingFileContent = content
    }

    private suspend fun submitTaskData(task: TaskData, shopId: Long, itemId: Long, fileContent: String) {
        try {
            val pdpUrl = "https://shopee.tw/api/v4/pdp/get_pc?" +
                "display_model_id=0&item_id=$itemId&model_selection_logic=3" +
                "&shop_id=$shopId&tz_offset_in_minutes=480&detail_level=0"

            // Extract the response_body from the captured data
            val resultBody = extractResponseBody(fileContent)

            val submitResponse = ApiService.submitTask(pdpUrl, resultBody)
            if (submitResponse.isSuccess && submitResponse.data?.isSuccess == true) {
                completedCount++
                log("任务提交成功: ${task.taskId}")
                updateStatus("提交成功")
                onTaskCompleted?.invoke(true, task.taskId)
            } else {
                failedCount++
                val msg = submitResponse.data?.msg ?: submitResponse.msg ?: "提交失败"
                log("任务提交失败: $msg")
                updateStatus("提交失败: $msg")
                onTaskCompleted?.invoke(false, msg)
            }
        } catch (e: Exception) {
            failedCount++
            log("提交异常: ${e.message}")
            updateStatus("提交异常")
            onTaskCompleted?.invoke(false, e.message ?: "异常")
        }
    }

    private fun extractResponseBody(fileContent: String): String {
        return try {
            val json = JSONObject(fileContent)
            json.optString("response_body", fileContent)
        } catch (e: Exception) {
            fileContent
        }
    }

    private fun sendWakeBroadcast(taskId: String, shopId: Long, itemId: Long) {
        val wakeUrl = "https://shopee.tw/---i.${taskId}.${shopId}.${itemId}"
        val intent = Intent(ApiConfig.WAKE_ACTION).apply {
            setPackage(ApiConfig.TARGET_PACKAGE)
            putExtra("url", wakeUrl)
        }
        sendBroadcast(intent)
        log("已发送唤醒广播: shopId=$shopId itemId=$itemId")
    }

    private fun updateStatus(status: String) {
        taskStatus = status
        onStatusChanged?.invoke(status)
    }

    private fun log(message: String) {
        Log.d(TAG, message)
        onLogMessage?.invoke("[${java.text.SimpleDateFormat("HH:mm:ss", java.util.Locale.getDefault()).format(java.util.Date())}] $message")
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "任务服务",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "任务助手后台服务"
            }
            val nm = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            nm.createNotificationChannel(channel)
        }
    }

    private fun createNotification(text: String): Notification {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
                .setContentTitle("任务助手")
                .setContentText(text)
                .setSmallIcon(android.R.drawable.ic_menu_upload)
                .build()
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
                .setContentTitle("任务助手")
                .setContentText(text)
                .setSmallIcon(android.R.drawable.ic_menu_upload)
                .build()
        }
    }
}
