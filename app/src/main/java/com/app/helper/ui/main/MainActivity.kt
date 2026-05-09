package com.app.helper.ui.main

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.app.helper.databinding.ActivityMainBinding
import com.app.helper.network.ApiService
import com.app.helper.receiver.DataReceiver
import com.app.helper.service.TaskService
import com.app.helper.ui.fp.FpManagerActivity
import com.app.helper.ui.login.LoginActivity
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private val logBuilder = StringBuilder()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        requestPermissions()
        setupUI()
        setupCallbacks()
        initFingerprint()
        updateStatusDisplay()
    }

    private fun setupUI() {
        binding.tvUsername.text = "用户: ${TokenManager.getUsername()}"

        binding.btnStartAuto.setOnClickListener {
            startTaskService("START_AUTO")
            binding.btnStartAuto.isEnabled = false
            binding.btnStopAuto.isEnabled = true
        }

        binding.btnStopAuto.setOnClickListener {
            startTaskService("STOP_AUTO")
            binding.btnStartAuto.isEnabled = true
            binding.btnStopAuto.isEnabled = false
        }

        binding.btnSingleTask.setOnClickListener {
            startTaskService("TAKE_SINGLE")
        }

        binding.btnChangeFp.setOnClickListener {
            startActivity(Intent(this, FpManagerActivity::class.java))
        }

        binding.btnTestApi.setOnClickListener {
            testApiConnection()
        }

        binding.btnLogout.setOnClickListener {
            AlertDialog.Builder(this)
                .setTitle("确认退出")
                .setMessage("确定要退出登录吗?")
                .setPositiveButton("确定") { _, _ ->
                    TokenManager.logout()
                    stopService(Intent(this, TaskService::class.java))
                    startActivity(Intent(this, LoginActivity::class.java))
                    finish()
                }
                .setNegativeButton("取消", null)
                .show()
        }

        binding.btnClearLog.setOnClickListener {
            logBuilder.clear()
            binding.tvLogs.text = ""
        }

        binding.btnStopAuto.isEnabled = false
    }

    private fun setupCallbacks() {
        TaskService.onStatusChanged = { status ->
            runOnUiThread {
                binding.tvStatus.text = "状态: $status"
                updateStatusDisplay()
            }
        }

        TaskService.onLogMessage = { msg ->
            runOnUiThread { appendLog(msg) }
        }

        TaskService.onTaskCompleted = { success, info ->
            runOnUiThread { updateStatusDisplay() }
        }

        DataReceiver.onLogReceived = { msg ->
            runOnUiThread { appendLog("[目标APP] $msg") }
        }

        DataReceiver.onVersionReceived = { ver ->
            runOnUiThread {
                binding.tvTargetVersion.text = "目标APP版本: $ver"
                appendLog("目标APP版本: $ver")
            }
        }

        DataReceiver.onToastReceived = { msg ->
            runOnUiThread {
                Toast.makeText(this, msg, Toast.LENGTH_SHORT).show()
            }
        }

        DataReceiver.onFileReceived = { uri, path, taskId ->
            runOnUiThread {
                appendLog("收到数据文件: taskId=$taskId")
            }
        }
    }

    private fun initFingerprint() {
        val fpFiles = FingerprintManager.listFpFiles(this)
        if (fpFiles.isEmpty()) {
            val fp = FingerprintManager.generateFullFp(this)
            val name = fp.optString("name", "default") + ".json"
            FingerprintManager.saveFingerprintData(this, name, fp.toString())
            FingerprintManager.setActiveFp(this, name)
            appendLog("已生成完整指纹配置")
        }
        updateFpDisplay()
    }

    private fun updateFpDisplay() {
        val fpFiles = FingerprintManager.listFpFiles(this)
        binding.tvFpCount.text = "指纹数量: ${fpFiles.size}"
        val currentFp = FingerprintManager.loadCurrentFp(this)
        if (currentFp != null) {
            try {
                val json = JSONObject(currentFp)
                val aid = json.optString("androidId", "N/A").take(8)
                val activeName = FingerprintManager.getActiveFpName(this) ?: "未知"
                binding.tvCurrentFp.text = "当前指纹: $activeName (ID: $aid...)"
            } catch (e: Exception) {
                binding.tvCurrentFp.text = "当前指纹: 已加载"
            }
        }
    }

    private fun updateStatusDisplay() {
        binding.tvCompletedCount.text = "完成: ${TaskService.completedCount}"
        binding.tvFailedCount.text = "失败: ${TaskService.failedCount}"
        binding.tvStatus.text = "状态: ${TaskService.taskStatus}"
        val task = TaskService.currentTask
        if (task != null) {
            binding.tvCurrentTask.text = "当前任务: ${task.taskId.take(12)}... (${task.parseShopId()}/${task.parseItemId()})"
        } else {
            binding.tvCurrentTask.text = "当前任务: 无"
        }
    }

    private fun testApiConnection() {
        appendLog("测试API连接...")
        lifecycleScope.launch {
            try {
                val response = ApiService.takeTask()
                if (response.isSuccess && response.data != null) {
                    val task = response.data
                    appendLog("API连接正常 - 任务: ${task.taskId}")
                    appendLog("  商品URL: ${task.taskUrl}")
                    appendLog("  shopId: ${task.parseShopId()} itemId: ${task.parseItemId()}")
                    appendLog("  过期时间: ${task.expireTime}")
                } else {
                    appendLog("API返回异常: ${response.msg}")
                }
            } catch (e: Exception) {
                appendLog("API连接失败: ${e.message}")
            }
        }
    }

    private fun appendLog(msg: String) {
        val time = SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date())
        val line = if (msg.startsWith("[")) msg else "[$time] $msg"
        logBuilder.appendLine(line)
        if (logBuilder.length > 10000) {
            val text = logBuilder.toString()
            logBuilder.clear()
            logBuilder.append(text.takeLast(5000))
        }
        binding.tvLogs.text = logBuilder.toString()
        binding.scrollLogs.post {
            binding.scrollLogs.fullScroll(android.view.View.FOCUS_DOWN)
        }
    }

    private fun startTaskService(action: String) {
        val intent = Intent(this, TaskService::class.java).apply {
            this.action = action
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun requestPermissions() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                try {
                    val intent = Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                        data = Uri.parse("package:$packageName")
                    }
                    startActivity(intent)
                } catch (e: Exception) {
                    startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
                }
            }
        } else {
            val perms = arrayOf(
                Manifest.permission.READ_EXTERNAL_STORAGE,
                Manifest.permission.WRITE_EXTERNAL_STORAGE
            )
            val needed = perms.filter {
                ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
            }
            if (needed.isNotEmpty()) {
                ActivityCompat.requestPermissions(this, needed.toTypedArray(), 100)
            }
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 101)
            }
        }
    }

    override fun onResume() {
        super.onResume()
        updateFpDisplay()
        updateStatusDisplay()
    }

    override fun onDestroy() {
        super.onDestroy()
        TaskService.onStatusChanged = null
        TaskService.onLogMessage = null
        TaskService.onTaskCompleted = null
        DataReceiver.onLogReceived = null
        DataReceiver.onVersionReceived = null
        DataReceiver.onToastReceived = null
        DataReceiver.onFileReceived = null
    }
}
