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
import android.graphics.Color
import android.provider.Settings
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.app.helper.R
import com.app.helper.databinding.ActivityMainBinding
import com.app.helper.network.ApiService
import com.app.helper.network.UpdateResponse
import com.app.helper.service.ChatAccessibilityService
import com.app.helper.service.TaskService
import com.app.helper.ui.login.LoginActivity
import com.app.helper.util.DelayPrefs
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private var logReceiver: android.content.BroadcastReceiver? = null
    private var updateReceiver: android.content.BroadcastReceiver? = null
    private var statsReceiver: android.content.BroadcastReceiver? = null
    private var downloadReceiver: android.content.BroadcastReceiver? = null
    private var pendingDownloadId = -1L
    private val activityScope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private val fpAdapter = FpAdapter()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        requestPermissions()
        setupUI()
        registerLogReceiver()
        registerUpdateReceiver()
        registerStatsReceiver()
        registerDownloadReceiver()
        checkAppUpdate()

        if (checkOverlayPermission()) {
            startTaskService()
        }
    }

    private fun setupUI() {
        val accountCount = TokenManager.getAccountCount()
        binding.tvUsername.text = if (accountCount > 0) {
            "下发ID：${TokenManager.getUsername()}（任务账号 $accountCount 个）"
        } else {
            "下发ID：${TokenManager.getUsername()}"
        }

        binding.btnTestApi.setOnClickListener {
            showDelayConfigDialog()
        }

        binding.etDeviceName.setText(TokenManager.getDeviceName())
        binding.btnSaveDeviceName.setOnClickListener {
            val name = binding.etDeviceName.text.toString().trim()
            TokenManager.saveDeviceName(name)
            Toast.makeText(
                this,
                "机器备注已保存：" + name.ifBlank { "默认（安卓设备）" },
                Toast.LENGTH_SHORT
            ).show()
        }

        binding.btnDelay01s.setOnClickListener { applyDelayPreset(100L, 100L) }
        binding.btnDelay01.setOnClickListener { applyDelayPreset(0L, 1000L) }
        binding.btnDelay38.setOnClickListener { applyDelayPreset(3000L, 8000L) }
        binding.btnDelay515.setOnClickListener { applyDelayPreset(5000L, 15000L) }
        binding.btnDelay3060.setOnClickListener { applyDelayPreset(30000L, 60000L) }
        refreshDelayUI()
        setupPauseConfig()
        setupChatMode()
        setupSilentMode()
        setupTabs()
        setupFingerprintPage()

        binding.btnStartAuto.setOnClickListener {
            if (checkOverlayPermission()) {
                startTaskService()
                Toast.makeText(this, "服务已启动，请使用悬浮窗开始任务。", Toast.LENGTH_SHORT).show()
            }
        }

        binding.btnStopAuto.setOnClickListener {
            stopService(Intent(this, TaskService::class.java))
            Toast.makeText(this, "服务已停止", Toast.LENGTH_SHORT).show()
        }

        binding.btnSingleTask.setOnClickListener {
            Toast.makeText(this, "请使用悬浮窗的单任务按钮。", Toast.LENGTH_SHORT).show()
        }

        binding.btnLogout.setOnClickListener {
            AlertDialog.Builder(this)
                .setTitle("退出登录")
                .setMessage("确定现在退出登录？")
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

    private fun setupTabs() {
        binding.tabGroup.addOnButtonCheckedListener { _, checkedId, isChecked ->
            if (!isChecked) return@addOnButtonCheckedListener
            val showEnv = checkedId == R.id.tab_env
            binding.pageTask.visibility = if (showEnv) android.view.View.GONE else android.view.View.VISIBLE
            binding.pageEnv.visibility = if (showEnv) android.view.View.VISIBLE else android.view.View.GONE
            if (showEnv) refreshFpList()
        }
    }

    private fun setupFingerprintPage() {
        binding.rvFpList.layoutManager = LinearLayoutManager(this)
        binding.rvFpList.adapter = fpAdapter
        binding.btnGenerateFp.setOnClickListener { generateNewFp() }
        binding.btnBatchGenerateFp.setOnClickListener { batchGenerateFp() }
        refreshFpList()
    }

    private fun generateNewFp() {
        val fp = FingerprintManager.generateFullFp(this)
        val name = fp.optString("name", "fp_${System.currentTimeMillis()}")
        FingerprintManager.saveFingerprintData(this, name, fp.toString())
        reportFpEvent("generate", name, fp.toString())
        Toast.makeText(this, "已生成：$name", Toast.LENGTH_SHORT).show()
        refreshFpList()
    }

    private fun batchGenerateFp() {
        val counts = arrayOf("5", "10", "20", "50")
        AlertDialog.Builder(this)
            .setTitle("批量生成")
            .setItems(counts) { _, which ->
                val count = counts[which].toInt()
                Thread {
                    val generated = mutableListOf<Pair<String, String>>()
                    repeat(count) { index ->
                        val fp = FingerprintManager.generateFullFp(this)
                        val name = fp.optString("name", "fp_${System.currentTimeMillis()}_$index")
                        FingerprintManager.saveFingerprintData(this, name, fp.toString())
                        generated.add(name to fp.toString())
                        Thread.sleep(10)
                    }
                    runOnUiThread {
                        generated.forEach { (name, data) -> reportFpEvent("generate", name, data) }
                        Toast.makeText(this, "已生成 $count 个指纹", Toast.LENGTH_SHORT).show()
                        refreshFpList()
                    }
                }.start()
            }
            .show()
    }

    private fun refreshFpList() {
        val files = FingerprintManager.listFpFiles(this)
        fpAdapter.setData(files)
        binding.tvFpCount.text = "总数：${files.size}"
    }

    private fun showFpOptions(item: FingerprintManager.FpFileInfo) {
        val options = mutableListOf("使用此指纹", "查看详情", "删除")
        if (item.isActive) options[0] = "当前指纹"

        AlertDialog.Builder(this)
            .setTitle(item.name)
            .setItems(options.toTypedArray()) { _, which ->
                when (which) {
                    0 -> {
                        FingerprintManager.setActiveFp(this, item.name)
                        FingerprintManager.loadFingerprintData(this, item.name)?.let { data ->
                            reportFpEvent("write", item.name, data)
                        }
                        Toast.makeText(this, "当前指纹：${item.name}", Toast.LENGTH_SHORT).show()
                        refreshFpList()
                    }
                    1 -> showFpDetail(item.name)
                    2 -> AlertDialog.Builder(this)
                        .setTitle("删除指纹")
                        .setMessage("确定删除 ${item.name}？")
                        .setPositiveButton("删除") { _, _ ->
                            FingerprintManager.deleteFpFile(this, item.name)
                            refreshFpList()
                        }
                        .setNegativeButton("取消", null)
                        .show()
                }
            }
            .show()
    }

    private fun showFpDetail(filename: String) {
        val data = FingerprintManager.loadFingerprintData(this, filename) ?: return
        try {
            val json = JSONObject(data)
            val info = buildString {
                appendLine("Android 标识：${json.optString("androidId")}")
                appendLine("GSF 标识：${json.optString("gsfAndroidId")}")
                appendLine("广告标识：${json.optString("advertiserId")}")
                appendLine("SIM 状态：${json.optString("gsm.sim.state")}")
                appendLine("运营商：${json.optString("gsm.operator.alpha")}")
                appendLine("总内存：${json.optString("totalMem").toLongOrNull()?.div(1024 * 1024 * 1024) ?: "?"}GB")
                appendLine("存储空间：${json.optString("totalSize").toLongOrNull()?.div(1024 * 1024 * 1024) ?: "?"}GB")
                appendLine("启动次数：${json.optString("bootCount")}")
                appendLine()
                val f7 = json.optString("f7_params")
                if (f7.isNotEmpty()) {
                    val f7Json = JSONObject(f7)
                    appendLine("--- F7 参数 ---")
                    appendLine("序列号池：${f7Json.optJSONArray("serialHexPool")?.length() ?: 0}")
                    appendLine("SN 池：${f7Json.optJSONArray("snHexPool")?.length() ?: 0}")
                    appendLine("C1 偏移：${f7Json.optInt("c1TimeOffsetDays")} 天")
                    appendLine("C2 偏移：${f7Json.optInt("c2TimeOffsetDays")} 天")
                    appendLine("叶子偏移：${f7Json.optInt("leafOffsetDays")} 天")
                    appendLine("叶子间隔：${f7Json.optInt("leafGapMs")}ms")
                    appendLine("vbh2: ${f7Json.optString("vbh2").take(16)}...")
                }
            }
            AlertDialog.Builder(this)
                .setTitle(filename)
                .setMessage(info)
                .setPositiveButton("确定", null)
                .show()
        } catch (e: Exception) {
            Toast.makeText(this, "解析失败：${e.message}", Toast.LENGTH_SHORT).show()
        }
    }

    private fun reportFpEvent(event: String, name: String, data: String) {
        activityScope.launch {
            try {
                ApiService.reportFingerprintEvent(event, name, data)
            } catch (_: Exception) {
            }
        }
    }

    inner class FpAdapter : RecyclerView.Adapter<FpAdapter.VH>() {
        private var items = listOf<FingerprintManager.FpFileInfo>()

        fun setData(data: List<FingerprintManager.FpFileInfo>) {
            items = data
            notifyDataSetChanged()
        }

        override fun onCreateViewHolder(parent: android.view.ViewGroup, viewType: Int): VH {
            val view = layoutInflater.inflate(R.layout.item_fp, parent, false)
            return VH(view)
        }

        override fun onBindViewHolder(holder: VH, position: Int) {
            holder.bind(items[position])
        }

        override fun getItemCount() = items.size

        inner class VH(view: android.view.View) : RecyclerView.ViewHolder(view) {
            private val tvName: android.widget.TextView = view.findViewById(R.id.tv_fp_name)
            private val tvInfo: android.widget.TextView = view.findViewById(R.id.tv_fp_info)
            private val tvActive: android.widget.TextView = view.findViewById(R.id.tv_fp_active)

            fun bind(item: FingerprintManager.FpFileInfo) {
                tvName.text = item.name
                val date = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.getDefault()).format(Date(item.lastModified))
                val sizeKb = item.size / 1024
                val androidId = try {
                    val data = FingerprintManager.loadFingerprintData(this@MainActivity, item.name)
                    if (data != null) JSONObject(data).optString("androidId", "").take(8) + "..." else ""
                } catch (_: Exception) {
                    ""
                }
                tvInfo.text = "${sizeKb}KB | $date | 标识：$androidId"
                tvActive.visibility = if (item.isActive) android.view.View.VISIBLE else android.view.View.GONE
                itemView.setOnClickListener { showFpOptions(item) }
            }
        }
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
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(logReceiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(logReceiver, filter)
        }
    }

    private fun registerStatsReceiver() {
        statsReceiver = object : android.content.BroadcastReceiver() {
            override fun onReceive(context: android.content.Context, intent: android.content.Intent) {
                val completed = intent.getIntExtra("completed", 0)
                val failed = intent.getIntExtra("failed", 0)
                val targetVersion = intent.getStringExtra("target_version") ?: "未知"
                val currentTask = intent.getStringExtra("current_task") ?: "无"
                val running = intent.getBooleanExtra("running", false)
                binding.tvCompletedCount.text = "完成：$completed"
                binding.tvFailedCount.text = "失败：$failed"
                binding.tvTargetVersion.text = "目标应用版本：$targetVersion"
                binding.tvCurrentTask.text = "当前任务：$currentTask"
                binding.tvStatus.text = if (running) "状态：服务运行中" else "状态：服务已停止"
            }
        }
        val filter = android.content.IntentFilter("com.app.helper.STATS_ACTION")
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(statsReceiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(statsReceiver, filter)
        }
    }

    private fun applyDelayPreset(minMs: Long, maxMs: Long) {
        DelayPrefs.set(this, minMs, maxMs)
        if (TaskService.isRunning) {
            val intent = Intent(this, TaskService::class.java).apply {
                action = "SET_DELAY"
                putExtra("delay_min_ms", minMs)
                putExtra("delay_max_ms", maxMs)
            }
            startService(intent)
        }
        refreshDelayUI()
        Toast.makeText(this, "延时已设置：${DelayPrefs.formatMs(minMs)}-${DelayPrefs.formatMs(maxMs)}", Toast.LENGTH_SHORT).show()
    }

    private fun refreshDelayUI() {
        val minMs = DelayPrefs.getMinMs(this)
        val maxMs = DelayPrefs.getMaxMs(this)
        binding.tvCurrentDelay.text = "当前：${DelayPrefs.formatMs(minMs)}-${DelayPrefs.formatMs(maxMs)}"

        val active = android.content.res.ColorStateList.valueOf(0xFF1976D2.toInt())
        val activeText = 0xFFFFFFFF.toInt()
        val normal = android.content.res.ColorStateList.valueOf(0x00000000)
        val normalText = 0xFF1976D2.toInt()

        fun apply(btn: com.google.android.material.button.MaterialButton, isActive: Boolean) {
            btn.backgroundTintList = if (isActive) active else normal
            btn.setTextColor(if (isActive) activeText else normalText)
        }
        apply(binding.btnDelay01s, minMs == 100L && maxMs == 100L)
        apply(binding.btnDelay01, minMs == 0L && maxMs == 1000L)
        apply(binding.btnDelay38, minMs == 3000L && maxMs == 8000L)
        apply(binding.btnDelay515, minMs == 5000L && maxMs == 15000L)
        apply(binding.btnDelay3060, minMs == 30000L && maxMs == 60000L)
    }

    private fun setupPauseConfig() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        binding.switchPauseEnabled.isChecked = prefs.getBoolean("pause_enabled", true)
        binding.etPauseCount.setText(prefs.getInt("pause_count", 30).toString())
        binding.etPauseMinutes.setText(prefs.getInt("pause_minutes", 30).toString())

        binding.switchPauseEnabled.setOnCheckedChangeListener { _, _ -> pushPauseConfig() }
        val watcher = object : android.text.TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: android.text.Editable?) { pushPauseConfig() }
        }
        binding.etPauseCount.addTextChangedListener(watcher)
        binding.etPauseMinutes.addTextChangedListener(watcher)
    }

    private fun pushPauseConfig() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val enabled = binding.switchPauseEnabled.isChecked
        val count = binding.etPauseCount.text.toString().toIntOrNull()?.coerceAtLeast(1) ?: 50
        val minutes = binding.etPauseMinutes.text.toString().toIntOrNull()?.coerceAtLeast(1) ?: 30
        prefs.edit()
            .putBoolean("pause_enabled", enabled)
            .putInt("pause_count", count)
            .putInt("pause_minutes", minutes)
            .apply()
        if (TaskService.isRunning) {
            val intent = Intent(this, TaskService::class.java).apply {
                action = "SET_PAUSE"
                putExtra("pause_enabled", enabled)
                putExtra("pause_count", count)
                putExtra("pause_minutes", minutes)
            }
            startService(intent)
        }
    }

    private fun setupSilentMode() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        binding.switchSilentMode.isChecked = prefs.getBoolean("silent_mode", false)
        binding.switchSilentMode.setOnCheckedChangeListener { _, _ -> pushSilentMode() }
    }

    private fun pushSilentMode() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val enabled = binding.switchSilentMode.isChecked
        prefs.edit().putBoolean("silent_mode", enabled).apply()
        if (TaskService.isRunning) {
            val intent = Intent(this, TaskService::class.java).apply {
                action = "SET_SILENT_MODE"
                putExtra("silent_mode", enabled)
            }
            startService(intent)
        }
    }

    // ==================== 聊一聊（OCR识别） ====================

    private fun setupChatMode() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        binding.switchChatMode.isChecked = prefs.getBoolean("chat_mode", false)
        binding.switchChatMode.setOnCheckedChangeListener { _, _ -> pushChatMode() }
        binding.btnChatPermission.setOnClickListener {
            startActivity(Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS))
        }
        refreshChatStatus()
    }

    private fun pushChatMode() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val enabled = binding.switchChatMode.isChecked
        prefs.edit().putBoolean("chat_mode", enabled).apply()
        if (TaskService.isRunning) {
            val intent = Intent(this, TaskService::class.java).apply {
                action = "SET_CHAT_MODE"
                putExtra("chat_mode", enabled)
            }
            startService(intent)
        }
        refreshChatStatus()
    }

    private fun refreshChatStatus() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val on = prefs.getBoolean("chat_mode", false)
        val ready = ChatAccessibilityService.isReady()
        val text = when {
            !on -> "未开启（开启后需保持 Shopee 停留在聊天页）"
            ready -> "无障碍已连接：可识别输入框/发送/链接/搜图页；Android 11+ 支持像素OCR"
            else -> "已开启，但无障碍未连接——请点下方按钮到系统设置开启【Ajie 聊一聊助手】"
        }
        binding.tvChatStatus.text = text
        binding.tvChatStatus.setTextColor(
            if (on && !ready) Color.parseColor("#E74C3C")
            else if (on) Color.parseColor("#27AE60")
            else Color.parseColor("#E67E22")
        )
    }

    private fun registerUpdateReceiver() {        updateReceiver = object : android.content.BroadcastReceiver() {
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
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
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
        statsReceiver?.let { try { unregisterReceiver(it) } catch (_: Exception) {} }
        downloadReceiver?.let { try { unregisterReceiver(it) } catch (_: Exception) {} }
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
        val msg = update.blockedMsg.ifBlank { "当前版本已停用，请更新后继续使用。" }
        val dialog = AlertDialog.Builder(this)
            .setTitle("版本已停用")
            .setMessage(msg + if (update.apkUrl.isNotBlank()) "\n\n最新版本：${update.versionName}" else "")
            .apply {
                if (update.apkUrl.isNotBlank()) {
                    setPositiveButton("更新", null)
                }
                setNegativeButton("退出") { _, _ -> finishAffinity() }
            }
            .create()
        dialog.setCanceledOnTouchOutside(false)
        dialog.setCancelable(false)
        dialog.setOnKeyListener { _, keyCode, _ -> keyCode == android.view.KeyEvent.KEYCODE_BACK }
        dialog.setOnShowListener {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE)?.setOnClickListener {
                downloadApk(update.apkUrl, update.versionName)
                dialog.setMessage("正在下载，请查看通知栏进度。\n\n$msg")
                dialog.getButton(AlertDialog.BUTTON_POSITIVE).text = "重新下载"
            }
        }
        dialog.show()
    }

    private fun showUpdateDialog(update: UpdateResponse) {
        val msg = buildString {
            append("最新版本：${update.versionName}")
            if (update.message.isNotBlank()) {
                append("\n\n")
                append(update.message)
            }
        }
        val dialog = AlertDialog.Builder(this)
            .setTitle(update.title.ifBlank { "发现新版本" })
            .setMessage(msg)
            .setPositiveButton("下载", null)
            .apply {
                if (!update.force) {
                    setNegativeButton("稍后", null)
                }
            }
            .create()
        dialog.setCanceledOnTouchOutside(!update.force)
        dialog.setCancelable(!update.force)
        dialog.setOnKeyListener { _, keyCode, _ -> update.force && keyCode == android.view.KeyEvent.KEYCODE_BACK }
        dialog.setOnShowListener {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener {
                downloadApk(update.apkUrl, update.versionName)
                if (update.force) {
                    dialog.setMessage("正在下载，请查看通知栏进度。\n\n$msg")
                    dialog.getButton(AlertDialog.BUTTON_POSITIVE).text = "重新下载"
                } else {
                    dialog.dismiss()
                }
            }
        }
        dialog.show()
    }

    private fun registerDownloadReceiver() {
        downloadReceiver = object : android.content.BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                val id = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1L)
                if (id > 0 && id == pendingDownloadId) {
                    installApk(id)
                }
            }
        }
        val filter = android.content.IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(downloadReceiver, filter, RECEIVER_EXPORTED)
        } else {
            registerReceiver(downloadReceiver, filter)
        }
    }

    private fun downloadApk(apkUrl: String, versionName: String) {
        try {
            val fileName = "Ajie-${versionName.ifBlank { "更新" }}.apk"
            val request = DownloadManager.Request(Uri.parse(apkUrl))
                .setTitle("Ajie 更新")
                .setDescription("下载完成后自动弹出安装")
                .setMimeType("application/vnd.android.package-archive")
                .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
                .setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, fileName)
                .setAllowedOverMetered(true)
                .setAllowedOverRoaming(true)
            val manager = getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
            pendingDownloadId = manager.enqueue(request)
            Toast.makeText(this, "开始下载，完成后自动弹出安装", Toast.LENGTH_LONG).show()
        } catch (e: Exception) {
            Toast.makeText(this, "下载失败：${e.message}", Toast.LENGTH_LONG).show()
            try {
                startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(apkUrl)))
            } catch (_: Exception) {
            }
        }
    }

    private fun installApk(downloadId: Long) {
        try {
            val manager = getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
            val uri = manager.getUriForDownloadedFile(downloadId) ?: return
            // Android 8+ 需允许「安装未知来源应用」
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && !packageManager.canRequestPackageInstalls()) {
                Toast.makeText(this, "请允许安装未知来源应用后，再点通知栏里的下载完成项安装", Toast.LENGTH_LONG).show()
                startActivity(Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:$packageName")))
                return
            }
            val intent = Intent(Intent.ACTION_VIEW).apply {
                setDataAndType(uri, "application/vnd.android.package-archive")
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            startActivity(intent)
        } catch (e: Exception) {
            Toast.makeText(this, "安装失败：${e.message}", Toast.LENGTH_LONG).show()
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
        binding.tvStatus.text = if (TaskService.isRunning) {
            "状态：服务运行中"
        } else {
            "状态：服务已停止"
        }
        refreshDelayUI()
        if (TaskService.isRunning) {
            try { sendBroadcast(Intent("com.app.helper.STATS_QUERY")) } catch (_: Exception) {}
        }
        try { refreshChatStatus() } catch (_: Exception) {}
    }

    private fun showDelayConfigDialog() {
        val curMin = DelayPrefs.getMinMs(this) / 1000L
        val curMax = DelayPrefs.getMaxMs(this) / 1000L

        val view = layoutInflater.inflate(R.layout.dialog_delay, null)
        val etMin = view.findViewById<android.widget.EditText>(R.id.et_delay_min)
        val etMax = view.findViewById<android.widget.EditText>(R.id.et_delay_max)
        etMin.setText(curMin.toString())
        etMax.setText(curMax.toString())

        AlertDialog.Builder(this)
            .setTitle("随机延迟（秒）")
            .setView(view)
            .setPositiveButton("确定") { _, _ ->
                val min = etMin.text.toString().toIntOrNull() ?: 3
                val max = etMax.text.toString().toIntOrNull() ?: 8
                val minSec = min.coerceAtLeast(0)
                val maxSec = max.coerceAtLeast(minSec)
                DelayPrefs.set(this, minSec * 1000L, maxSec * 1000L)
                Toast.makeText(this, "延迟已设置：${DelayPrefs.formatMs(minSec * 1000L)}-${DelayPrefs.formatMs(maxSec * 1000L)}", Toast.LENGTH_SHORT).show()
                if (TaskService.isRunning) {
                    val intent = Intent(this, TaskService::class.java).apply {
                        action = "SET_DELAY"
                        putExtra("delay_min_ms", minSec * 1000L)
                        putExtra("delay_max_ms", maxSec * 1000L)
                    }
                    startService(intent)
                }
                refreshDelayUI()
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
                .setMessage("请授予悬浮窗权限，用于显示任务操作窗口。")
                .setPositiveButton("去设置") { _, _ ->
                    val intent = Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION, Uri.parse("package:$packageName"))
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
