package com.app.helper.ui.compose

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.lifecycle.lifecycleScope
import com.app.helper.network.ApiService
import com.app.helper.service.TaskService
import com.app.helper.util.DelayPrefs
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Compose 版主界面。
 *
 * 与旧版 MainActivity 并行存在，依赖完全一致：
 * TokenManager（登录态）、DelayPrefs（延时）、task_config（暂停/静默/聊天）、
 * TaskService（后台任务）、FingerprintManager（指纹）。
 *
 * 任务真正的开始/停止由悬浮窗控制，这里只做展示与配置下发。
 */
class ComposeMainActivity : ComponentActivity() {

    private val state = UiState()
    private var logReceiver: BroadcastReceiver? = null
    private var statsReceiver: BroadcastReceiver? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (!TokenManager.isLoggedIn()) {
            startActivity(Intent(this, com.app.helper.ui.login.LoginActivity::class.java))
            finish()
            return
        }

        loadPrefs()
        registerReceivers()
        refreshFingerprints()

        setContent {
            HelperApp(
                state = state,
                onLogout = { doLogout() },
                onStartAuto = { startTaskService() },
                onStopAuto = { stopService(Intent(this, TaskService::class.java)) },
                onSingleTask = { startTaskService() },
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
            )
        }
    }

    override fun onResume() {
        super.onResume()
        state.chatPermissionOk = isAccessibilityEnabled()
        refreshFingerprints()
    }

    override fun onDestroy() {
        super.onDestroy()
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

    /** 与旧版一致：generateFullFp → saveFingerprintData → 上报。 */
    private fun generateFp(count: Int) {
        lifecycleScope.launch {
            val names = withContext(Dispatchers.IO) {
                val out = mutableListOf<String>()
                repeat(count) { idx ->
                    runCatching {
                        val fp = FingerprintManager.generateFullFp(this@ComposeMainActivity)
                        val name = fp.optString("name", "fp_${System.currentTimeMillis()}_$idx")
                        FingerprintManager.saveFingerprintData(
                            this@ComposeMainActivity, name, fp.toString()
                        )
                        out.add(name)
                        runCatching { ApiService.reportFingerprintEvent("generate", name, fp.toString()) }
                        Thread.sleep(10)
                    }
                }
                out
            }
            state.addLog("[指纹] 生成 ${names.size} 个")
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
        runCatching { startService(Intent(this, TaskService::class.java)) }
    }

    private fun doLogout() {
        TokenManager.logout()
        startActivity(Intent(this, com.app.helper.ui.login.LoginActivity::class.java))
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
}
