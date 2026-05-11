package com.app.helper.ui.main

import android.Manifest
import android.app.DownloadManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import com.app.helper.R
import com.app.helper.databinding.ActivityMainBinding
import com.app.helper.network.ApiService
import com.app.helper.network.UpdateResponse
import com.app.helper.service.TaskService
import com.app.helper.ui.fp.FpManagerActivity
import com.app.helper.ui.login.LoginActivity
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.json.JSONObject

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private var logReceiver: android.content.BroadcastReceiver? = null
    private var updateReceiver: android.content.BroadcastReceiver? = null
    private val activityScope = CoroutineScope(Dispatchers.Main + SupervisorJob())

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        requestPermissions()
        initFingerprint()
        setupUI()
        registerLogReceiver()
        registerUpdateReceiver()
        checkAppUpdate()

        // Start service with float window
        if (checkOverlayPermission()) {
            startTaskService()
        }
    }

    private fun setupUI() {
        binding.tvUsername.text = "用户: ${TokenManager.getUsername()}"

        binding.btnChangeFp.setOnClickListener {
            startActivity(Intent(this, FpManagerActivity::class.java))
        }

        binding.btnTestApi.setOnClickListener {
            showDelayConfigDialog()
        }

        binding.btnStartAuto.setOnClickListener {
            if (checkOverlayPermission()) {
                startTaskService()
                Toast.makeText(this, "服务已启动，请在悬浮窗中操作", Toast.LENGTH_SHORT).show()
            }
        }

        binding.btnStopAuto.setOnClickListener {
            stopService(Intent(this, TaskService::class.java))
            Toast.makeText(this, "服务已停止", Toast.LENGTH_SHORT).show()
        }

        binding.btnSingleTask.setOnClickListener {
            Toast.makeText(this, "请使用悬浮窗中的【一键做任务】按钮", Toast.LENGTH_SHORT).show()
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

        binding.btnClearLog.setOnClickListener { binding.tvLogs.text = "" }
    }

    private fun appendLog(msg: String) {
        val time = java.text.SimpleDateFormat("HH:mm:ss", java.util.Locale.getDefault()).format(java.util.Date())
        binding.tvLogs.append("[$time] $msg\n")
        binding.scrollLogs.post { binding.scrollLogs.fullScroll(android.view.View.FOCUS_DOWN) }
    }

    private fun registerLogReceiver() {
        logReceiver = object : android.content.BroadcastReceiver() {
            override fun onReceive(context: android.content.Context, intent: android.content.Intent) {
                val msg = intent.getStringExtra("log_msg") ?: return
                appendLog(msg)
            }
        }
        val filter = android.content.IntentFilter("com.app.helper.LOG_ACTION")
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(logReceiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(logReceiver, filter)
        }
    }

    private fun registerUpdateReceiver() {
        updateReceiver = object : android.content.BroadcastReceiver() {
            override fun onReceive(context: android.content.Context, intent: android.content.Intent) {
                val isBlocked = intent.getBooleanExtra("blocked", false)
                val update = UpdateResponse(
                    ok = true,
                    hasUpdate = true,
                    versionName = intent.getStringExtra("version_name") ?: "",
                    apkUrl = intent.getStringExtra("apk_url") ?: "",
                    title = intent.getStringExtra("title") ?: "发现新版本",
                    message = intent.getStringExtra("message") ?: "",
                    force = intent.getBooleanExtra("force", false),
                    blocked = isBlocked,
                    blockedMsg = intent.getStringExtra("message") ?: ""
                )
                if (update.apkUrl.isNotBlank()) {
                    if (isBlocked) showBlockedDialog(update) else showUpdateDialog(update)
                }
            }
        }
        val filter = android.content.IntentFilter("com.app.helper.UPDATE_ACTION")
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(updateReceiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(updateReceiver, filter)
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        activityScope.cancel()
        logReceiver?.let { try { unregisterReceiver(it) } catch (_: Exception) {} }
        updateReceiver?.let { try { unregisterReceiver(it) } catch (_: Exception) {} }
    }

    private fun initFingerprint() {
        val fpFiles = FingerprintManager.listFpFiles(this)
        if (fpFiles.isEmpty()) {
            val fp = FingerprintManager.generateFullFp(this)
            val name = fp.optString("name", "default")
            FingerprintManager.saveFingerprintData(this, name, fp.toString())
            FingerprintManager.setActiveFp(this, name)
        }
        FingerprintManager.dumpFpToDownloads(this)
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
                binding.tvCurrentFp.text = "当前指纹: ID:${aid}..."
            } catch (e: Exception) {
                binding.tvCurrentFp.text = "当前指纹: 已加载"
            }
        }
    }

    private fun checkAppUpdate() {
        activityScope.launch {
            try {
                val currentCode = getCurrentVersionCode()
                val update = ApiService.checkUpdate(currentCode)
                if (update.ok && update.blocked) {
                    showBlockedDialog(update)
                } else if (update.ok && update.hasUpdate && update.apkUrl.isNotBlank()) {
                    showUpdateDialog(update)
                }
            } catch (_: Exception) {
            }
        }
    }

    private fun showBlockedDialog(update: UpdateResponse) {
        val msg = update.blockedMsg.ifBlank { "您的版本已停用，请更新到最新版本" }
        val dialog = AlertDialog.Builder(this)
            .setTitle("版本已停用")
            .setMessage(msg + if (update.apkUrl.isNotBlank()) "\n\n最新版本: ${update.versionName}" else "")
            .apply {
                if (update.apkUrl.isNotBlank()) {
                    setPositiveButton("立即更新", null)
                }
                setNegativeButton("退出应用") { _, _ -> finishAffinity() }
            }
            .create()
        dialog.setCanceledOnTouchOutside(false)
        dialog.setCancelable(false)
        dialog.setOnKeyListener { _, keyCode, _ ->
            keyCode == android.view.KeyEvent.KEYCODE_BACK
        }
        dialog.setOnShowListener {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE)?.setOnClickListener {
                downloadApk(update.apkUrl, update.versionName)
                dialog.setMessage("正在下载，请在通知栏查看进度。\n安装完成后重新打开App。\n\n$msg")
                dialog.getButton(AlertDialog.BUTTON_POSITIVE).text = "重新下载"
            }
        }
        dialog.show()
    }

    private fun showUpdateDialog(update: UpdateResponse) {
        val msg = buildString {
            append("最新版本: ${update.versionName}")
            if (update.message.isNotBlank()) {
                append("\n\n")
                append(update.message)
            }
        }
        val dialog = AlertDialog.Builder(this)
            .setTitle(update.title.ifBlank { "发现新版本" })
            .setMessage(msg)
            .setPositiveButton("下载更新", null)
            .apply {
                if (!update.force) {
                    setNegativeButton("稍后", null)
                }
            }
            .create()
        dialog.setCanceledOnTouchOutside(!update.force)
        dialog.setCancelable(!update.force)
        dialog.setOnKeyListener { _, keyCode, _ ->
            update.force && keyCode == android.view.KeyEvent.KEYCODE_BACK
        }
        dialog.setOnShowListener {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener {
                downloadApk(update.apkUrl, update.versionName)
                if (update.force) {
                    dialog.setMessage("正在下载，请在通知栏查看进度。\n安装完成后重新打开App。\n\n${msg}")
                    dialog.getButton(AlertDialog.BUTTON_POSITIVE).text = "重新下载"
                } else {
                    dialog.dismiss()
                }
            }
        }
        dialog.show()
    }

    private fun downloadApk(apkUrl: String, versionName: String) {
        try {
            val fileName = "TaskHelper-${versionName.ifBlank { "update" }}.apk"
            val request = DownloadManager.Request(Uri.parse(apkUrl))
                .setTitle("格界更新")
                .setDescription("正在下载新版本")
                .setMimeType("application/vnd.android.package-archive")
                .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
                .setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, fileName)
                .setAllowedOverMetered(true)
                .setAllowedOverRoaming(true)
            val manager = getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
            manager.enqueue(request)
            Toast.makeText(this, "已开始下载，请在通知栏查看", Toast.LENGTH_LONG).show()
        } catch (e: Exception) {
            Toast.makeText(this, "下载失败: ${e.message}", Toast.LENGTH_LONG).show()
            try {
                startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(apkUrl)))
            } catch (_: Exception) {
            }
        }
    }

    private fun getCurrentVersionCode(): Int {
        return try {
            val info = packageManager.getPackageInfo(packageName, 0)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) info.longVersionCode.toInt() else @Suppress("DEPRECATION") info.versionCode
        } catch (_: Exception) {
            0
        }
    }

    override fun onResume() {
        super.onResume()
        updateFpDisplay()
        binding.tvStatus.text = if (TaskService.isRunning) "状态: 服务运行中(悬浮窗)" else "状态: 服务未启动"
    }

    private fun showDelayConfigDialog() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val curMin = prefs.getInt("delay_min", 3)
        val curMax = prefs.getInt("delay_max", 8)

        val view = layoutInflater.inflate(R.layout.dialog_delay, null)
        val etMin = view.findViewById<android.widget.EditText>(R.id.et_delay_min)
        val etMax = view.findViewById<android.widget.EditText>(R.id.et_delay_max)
        etMin.setText(curMin.toString())
        etMax.setText(curMax.toString())

        AlertDialog.Builder(this)
            .setTitle("设置随机延时(秒)")
            .setView(view)
            .setPositiveButton("确定") { _, _ ->
                val min = etMin.text.toString().toIntOrNull() ?: 3
                val max = etMax.text.toString().toIntOrNull() ?: 8
                prefs.edit().putInt("delay_min", min.coerceAtLeast(1))
                    .putInt("delay_max", max.coerceAtLeast(min)).apply()
                Toast.makeText(this, "延时已设置: $min-$max 秒", Toast.LENGTH_SHORT).show()
                if (TaskService.isRunning) {
                    val intent = Intent(this, TaskService::class.java).apply {
                        action = "SET_DELAY"
                        putExtra("delay_min", min)
                        putExtra("delay_max", max)
                    }
                    startService(intent)
                }
            }
            .setNegativeButton("取消", null)
            .show()
    }

    private fun startTaskService() {
        val intent = Intent(this, TaskService::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun checkOverlayPermission(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M && !Settings.canDrawOverlays(this)) {
            AlertDialog.Builder(this)
                .setTitle("需要悬浮窗权限")
                .setMessage("请授予悬浮窗权限，用于显示任务状态")
                .setPositiveButton("去设置") { _, _ ->
                    val intent = Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                        Uri.parse("package:$packageName"))
                    startActivity(intent)
                }
                .setNegativeButton("取消", null)
                .show()
            return false
        }
        return true
    }

    private fun requestPermissions() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                try {
                    startActivity(Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                        data = Uri.parse("package:$packageName")
                    })
                } catch (e: Exception) {
                    startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
                }
            }
        } else {
            val perms = arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE, Manifest.permission.WRITE_EXTERNAL_STORAGE)
            val needed = perms.filter { ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED }
            if (needed.isNotEmpty()) ActivityCompat.requestPermissions(this, needed.toTypedArray(), 100)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 101)
            }
        }
    }
}
