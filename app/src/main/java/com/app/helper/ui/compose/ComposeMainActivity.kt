package com.app.helper.ui.compose

import android.Manifest
import android.app.AlertDialog
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.util.Log
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.app.helper.network.ApiService
import com.app.helper.service.TaskService
import com.app.helper.util.DelayPrefs
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import com.app.helper.vpn.ApkInstaller
import com.app.helper.vpn.VpnController
import com.app.helper.vpn.VpnGuard
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Compose 版主界面。
 *
 * 与旧版 MainActivity 并行存在，依赖完全一致：
 * TokenManager（登录态）、DelayPrefs（延时）、task_config（暂停/静默/聊天）、
 * TaskService（后台任务）、FingerprintManager（指纹）。
 *
 * 任务真正的开始/停止由悬浮窗控制；悬浮窗由 TaskService 创建，
 * 因此必须先授予悬浮窗权限并用 startForegroundService 拉起服务。
 */
class ComposeMainActivity : ComponentActivity() {

    private val state = UiState()
    private var logReceiver: BroadcastReceiver? = null
    private var statsReceiver: BroadcastReceiver? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (!TokenManager.isLoggedIn()) {
            startActivity(Intent(this, com.app.helper.ui.compose.ComposeLoginActivity::class.java))
            finish()
            return
        }

        requestPermissions()
        loadPrefs()
        refreshVpnState()
        registerReceivers()
        refreshFingerprints()

        // 与原版一致：有悬浮窗权限就直接把服务拉起来，
        // 悬浮窗是任务的唯一操作入口，服务不起用户会以为「点了没反应」
        if (checkOverlayPermission()) {
            startTaskService()
        }

        setContent {
            HelperApp(
                state = state,
                onLogout = { doLogout() },
                onStartAuto = { if (checkOverlayPermission()) startTaskService() },
                onStopAuto = { stopService(Intent(this, TaskService::class.java)) },
                onSingleTask = { if (checkOverlayPermission()) startTaskService() },
                onTestApi = { applyDelay(state.delayMinMs, state.delayMaxMs) },
                onSetDelay = { min, max -> applyDelay(min, max) },
                onPauseEnabled = { on -> savePause(on, state.pauseCount, state.pauseMinutes) },
                onPauseConfig = { c, m -> savePause(state.pauseEnabled, c, m) },
                onSilentMode = { on -> saveFlag("silent_mode", on) },
                onChatMode = { on -> saveFlag("chat_mode", on) },
                onChatPermission = { openAccessibilitySettings() },
                onClearLogs = { state.clearLogs() },
                onCheckUpdate = { state.addLog("[版本] 目标版本 ${state.targetVersion.ifBlank { "未获取" }}") },
                onGenerateFp = { generateFp(1) },
                onBatchGenerateFp = { generateFp(5) },
                onFpClick = { fp -> activateFp(fp.name) },
                onVpnToggle = { on -> toggleVpn(on) },
                onVpnConnect = { connectFastestVpn() },
                onInstallVpn = { installVpnApp() },
            )
        }
    }

    override fun onResume() {
        super.onResume()
        state.chatPermissionOk = isAccessibilityEnabled()
        refreshVpnState()
        refreshFingerprints()
    }

    override fun onDestroy() {
        super.onDestroy()
        VpnGuard.stop()
        logReceiver?.let { runCatching { unregisterReceiver(it) } }
        statsReceiver?.let { runCatching { unregisterReceiver(it) } }
    }

    // ---------------- 配置 ----------------

    private fun loadPrefs() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        state.pauseEnabled = prefs.getBoolean("pause_enabled", false)
        state.pauseCount = prefs.getInt("pause_count", 30)
        state.pauseMinutes = prefs.getInt("pause_minutes", 30)
        state.silentMode = prefs.getBoolean("silent_mode", false)
        state.chatMode = prefs.getBoolean("chat_mode", false)
        state.delayMinMs = DelayPrefs.getMinMs(this)
        state.delayMaxMs = DelayPrefs.getMaxMs(this)
        state.chatPermissionOk = isAccessibilityEnabled()
        state.username = TokenManager.getUsername().ifBlank {
            if (TokenManager.getInventoryMode()) "库存模式" else "普通模式"
        }
        state.chatStatus = if (state.chatPermissionOk) "无障碍已授权" else "无障碍未授权"
        state.loggedIn = true
    }

    private fun applyDelay(min: Long, max: Long) {
        DelayPrefs.set(this, min, max)
        state.delayMinMs = min
        state.delayMaxMs = max
        state.addLog("[配置] 延时 ${DelayPrefs.formatMs(min)}-${DelayPrefs.formatMs(max)}")
        runCatching {
            startService(Intent(this, TaskService::class.java).apply {
                action = "SET_DELAY"
                putExtra("delay_min_ms", min)
                putExtra("delay_max_ms", max)
            })
        }
    }

    private fun savePause(enabled: Boolean, count: Int, minutes: Int) {
        state.pauseEnabled = enabled
        state.pauseCount = count
        state.pauseMinutes = minutes
        getSharedPreferences("task_config", MODE_PRIVATE).edit()
            .putBoolean("pause_enabled", enabled)
            .putInt("pause_count", count)
            .putInt("pause_minutes", minutes)
            .apply()
        runCatching {
            startService(Intent(this, TaskService::class.java).apply {
                action = "SET_PAUSE"
                putExtra("pause_enabled", enabled)
                putExtra("pause_count", count)
                putExtra("pause_minutes", minutes)
            })
        }
    }

    private fun saveFlag(key: String, value: Boolean) {
        getSharedPreferences("task_config", MODE_PRIVATE).edit().putBoolean(key, value).apply()
        when (key) {
            "silent_mode" -> {
                state.silentMode = value
                runCatching {
                    startService(Intent(this, TaskService::class.java).apply {
                        action = "SET_SILENT_MODE"
                        putExtra("silent_mode", value)
                    })
                }
            }
            "chat_mode" -> {
                state.chatMode = value
                runCatching {
                    startService(Intent(this, TaskService::class.java).apply {
                        action = "SET_CHAT_MODE"
                        putExtra("chat_mode", value)
                    })
                }
            }
        }
    }

    // ---------------- 指纹 ----------------

    private fun refreshFingerprints() {
        lifecycleScope.launch {
            val list = withContext(Dispatchers.IO) {
                runCatching { FingerprintManager.listFpFiles(this@ComposeMainActivity) }
                    .getOrDefault(emptyList())
            }
            state.fpList.clear()
            state.fpList.addAll(list)
            state.fpCount = list.size
        }
    }

    /**
     * 从服务器领取指纹环境并保存。
     *
     * 服务端 /api/fp_env/generate 负责生成身份字段（androidId / gsf / gaid /
     * MAC / 机型）并对每个账号做每日额度控制。客户端不再本地随机生成，
     * 这样发号可以统一管理和限流。
     */
    private fun generateFp(count: Int) {
        lifecycleScope.launch {
            var ok = 0
            var failReason = ""
            for (idx in 0 until count) {
            val fp = withContext(Dispatchers.IO) {
                try { ApiService.fetchFingerprintEnv() } catch (e: Exception) { null }
            }
            if (fp == null) {
                failReason = if (failReason.isBlank()) "服务器未返回" else failReason
                continue
            }
            val name = fp.optString("name", "").ifBlank {
                "FP-" + java.text.SimpleDateFormat(
                    "yyyy-MM-dd HH:mm:ss", java.util.Locale.US
                ).format(java.util.Date())
            }
            withContext(Dispatchers.IO) {
                runCatching {
                    FingerprintManager.saveFingerprintData(
                        this@ComposeMainActivity, name, fp.toString()
                    )
                }
            }
            try {
                ApiService.reportFingerprintEvent("generate", name, fp.toString())
            } catch (_: Exception) {
            }
            ok++
            }
            if (ok > 0) {
                state.addLog("[指纹] 已从服务器领取 $ok 个")
            } else {
                state.addLog("[指纹] 领取失败（$failReason）——请确认后台已授权生成环境且今日额度未用完")
            }
            refreshFingerprints()
        }
    }

    private fun activateFp(name: String) {
        lifecycleScope.launch {
            val ok = withContext(Dispatchers.IO) {
                runCatching {
                    FingerprintManager.setActiveFp(this@ComposeMainActivity, name)
                    FingerprintManager.getActiveFpName(this@ComposeMainActivity) == name
                }.getOrDefault(false)
            }
            state.addLog(if (ok) "[指纹] 已切换到 $name" else "[指纹] 切换失败")
            refreshFingerprints()
        }
    }

    // ---------------- 其他 ----------------

    private fun startTaskService() {
        // Android 8+ 必须用 startForegroundService：
        // TaskService 声明了 foregroundServiceType，用 startService 会被系统拒绝，
        // 表现就是「点了没反应、没有悬浮窗」。
        val intent = Intent(this, TaskService::class.java)
        runCatching {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent)
            } else {
                startService(intent)
            }
        }
    }

    /** 悬浮窗权限：TaskService 靠它显示操作窗口，没有就什么都看不到。 */
    private fun checkOverlayPermission(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M && !Settings.canDrawOverlays(this)) {
            AlertDialog.Builder(this)
                .setTitle("需要悬浮窗权限")
                .setMessage("请授予悬浮窗权限，用于显示任务操作窗口。")
                .setPositiveButton("去设置") { _, _ ->
                    runCatching {
                        startActivity(
                            Intent(
                                Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                                Uri.parse("package:$packageName")
                            )
                        )
                    }
                }
                .setNegativeButton("取消", null)
                .show()
            return false
        }
        return true
    }

    /** 存储（读取商品数据）与通知（前台服务常驻）权限。 */
    private fun requestPermissions() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                runCatching {
                    startActivity(
                        Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                            data = Uri.parse("package:$packageName")
                        }
                    )
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
            if (ContextCompat.checkSelfPermission(
                    this, Manifest.permission.POST_NOTIFICATIONS
                ) != PackageManager.PERMISSION_GRANTED
            ) {
                ActivityCompat.requestPermissions(
                    this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 101
                )
            }
        }
    }

    private fun doLogout() {
        TokenManager.logout()
        startActivity(Intent(this, com.app.helper.ui.compose.ComposeLoginActivity::class.java))
        finish()
    }

    private fun openAccessibilitySettings() {
        runCatching { startActivity(Intent(android.provider.Settings.ACTION_ACCESSIBILITY_SETTINGS)) }
    }

    private fun isAccessibilityEnabled(): Boolean = runCatching {
        val svc = "${packageName}/${com.app.helper.service.ChatAccessibilityService::class.java.name}"
        val enabled = android.provider.Settings.Secure.getString(
            contentResolver,
            android.provider.Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES
        ) ?: return false
        enabled.contains(svc)
    }.getOrDefault(false)

    private fun registerReceivers() {
        logReceiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context?, intent: Intent?) {
                val msg = intent?.getStringExtra("log_msg")?.takeIf { it.isNotBlank() } ?: return
                state.addLog(msg)
            }
        }
        statsReceiver = object : BroadcastReceiver() {
            override fun onReceive(context: Context?, intent: Intent?) {
                intent ?: return
                state.completed = intent.getIntExtra("completed", state.completed)
                state.failed = intent.getIntExtra("failed", state.failed)
                intent.getStringExtra("status")?.let { state.status = it }
                intent.getStringExtra("current_task")?.let { state.currentTask = it }
            }
        }
        val flag = if (Build.VERSION.SDK_INT >= 33) Context.RECEIVER_NOT_EXPORTED else 0
        runCatching {
            registerReceiver(logReceiver, IntentFilter("com.app.helper.LOG_ACTION"), flag)
            registerReceiver(statsReceiver, IntentFilter("com.app.helper.STATS_ACTION"), flag)
        }
    }

    // ---------------- VPN ----------------

    private fun refreshVpnState() {
        state.vpnTargetInstalled = VpnController.isTargetInstalled(this)
        state.vpnEnabled = VpnController.isAutoEnabled(this)
        state.vpnRegion = VpnController.region(this)
        state.vpnStatus = when {
            !state.vpnTargetInstalled -> "未检测到 v2rayNG"
            VpnGuard.isRunning() -> VpnGuard.lastMessage.ifBlank { "守护运行中" }
            else -> "已停止"
        }
    }

    private fun toggleVpn(on: Boolean) {
        if (!state.vpnTargetInstalled) {
            state.addLog("[代理] 请先安装 v2rayNG")
            VpnController.openInstallPage(this)
            return
        }
        VpnController.setAutoEnabled(this, on)
        state.vpnEnabled = on
        if (on) {
            state.vpnStatus = "正在连接..."
            state.addLog("[代理] 启动自动切换")
            VpnGuard.start(this)
            connectFastestVpn()
        } else {
            VpnGuard.stop()
            VpnController.setEnabled(this, false)
            state.vpnStatus = "已停止"
            state.addLog("[代理] 已停止")
        }
    }

    private fun connectFastestVpn() {
        if (!state.vpnTargetInstalled) {
            state.addLog("[代理] 请先安装 v2rayNG")
            return
        }
        state.vpnStatus = "正在测速..."
        VpnGuard.connectNow(this) { ok, msg ->
            state.vpnStatus = msg
            state.addLog("[代理] " + msg)
            if (!ok && msg.contains("v2rayNG")) {
                VpnController.openInstallPage(this)
            }
        }
    }


    /** 在线安装 v2rayNG：要权限 → 下载 → 拉起安装器。 */
    private fun installVpnApp() {
        if (state.vpnInstalling) return
        if (VpnController.isTargetInstalled(this)) {
            state.vpnTargetInstalled = true
            state.addLog("[代理] v2rayNG 已安装")
            return
        }
        // 没有「安装未知应用」权限时先去授权页
        if (!ApkInstaller.canInstall(this)) {
            state.addLog("[代理] 请先允许本应用安装未知应用")
            Toast.makeText(this, "请允许安装未知应用后返回重试", Toast.LENGTH_LONG).show()
            ApkInstaller.requestInstallPermission(this)
            return
        }
        state.vpnInstalling = true
        state.vpnDownloadPct = 0
        state.addLog("[代理] 开始下载 v2rayNG")
        lifecycleScope.launch {
            val file = ApkInstaller.download(this@ComposeMainActivity) { pct ->
                lifecycleScope.launch(Dispatchers.Main) { state.vpnDownloadPct = pct }
            }
            state.vpnInstalling = false
            state.vpnDownloadPct = -1
            if (file == null) {
                state.addLog("[代理] 下载失败，检查网络")
                Toast.makeText(
                    this@ComposeMainActivity, "下载失败，检查网络", Toast.LENGTH_LONG
                ).show()
                return@launch
            }
            state.addLog("[代理] 下载完成，正在拉起安装器")
            val ok = ApkInstaller.install(this@ComposeMainActivity)
            if (!ok) {
                state.addLog("[代理] 拉起安装器失败")
                Toast.makeText(
                    this@ComposeMainActivity, "无法启动安装器", Toast.LENGTH_LONG
                ).show()
            }
        }
    }

}
