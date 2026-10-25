package com.app.helper.ui.compose

import android.graphics.Bitmap
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.app.helper.util.FingerprintManager

/** 主界面所有可变状态集中在这里，compose 直接读它做重组。 */
class UiState {

    // ---- 账号 ----
    var username by mutableStateOf("")
    /** 顶栏头像，null 时显示占位图标。 */
    var avatarBitmap by mutableStateOf<Bitmap?>(null)
    var loggedIn by mutableStateOf(false)

    // ---- 页签 ----
    var tab by mutableIntStateOf(0)   // 0=任务 1=环境

    // ---- 任务状态 ----
    var status by mutableStateOf("未运行")
    var currentTask by mutableStateOf("")
    var completed by mutableIntStateOf(0)
    var failed by mutableIntStateOf(0)
    var autoRunning by mutableStateOf(false)

    // ---- 延时 ----
    var delayMinMs by mutableLongStateOf(15_000L)
    var delayMaxMs by mutableLongStateOf(35_000L)

    // ---- 暂停 ----
    var pauseEnabled by mutableStateOf(false)
    var pauseCount by mutableIntStateOf(30)
    var pauseMinutes by mutableIntStateOf(30)

    // ---- 静默 / 聊天 ----
    var silentMode by mutableStateOf(false)
    var chatMode by mutableStateOf(false)
    var chatStatus by mutableStateOf("未开启")
    var chatPermissionOk by mutableStateOf(false)

    // ---- 版本 ----
    var targetVersion by mutableStateOf("")
    var updateAvailable by mutableStateOf(false)

    // ---- 日志（最多保留 300 行）----
    val logs = mutableStateListOf<String>()

    fun addLog(line: String) {
        val stamped = line
        logs.add(stamped)
        if (logs.size > 300) {
            repeat(logs.size - 300) { logs.removeAt(0) }
        }
    }

    fun clearLogs() = logs.clear()

    // ---- VPN ----
    var vpnEnabled by mutableStateOf(false)
    var vpnStatus by mutableStateOf("未启用")
    var vpnRegion by mutableStateOf("")
    var vpnTargetInstalled by mutableStateOf(false)
    var vpnDownloadPct by mutableIntStateOf(-1)   // -1 表示未在下载
    var vpnInstalling by mutableStateOf(false)
    var vpnDetectLog by mutableStateOf("")
    /** 检测到的代理 App 包名列表，多于一个时界面给切换入口。 */
    val vpnAppChoices = mutableStateListOf<String>()
    /** 手动选择弹窗是否显示。 */
    var vpnPickerVisible by mutableStateOf(false)
    /** 当前选中的代理 App 包名，空表示未选中。 */
    var vpnCurrentPkg by mutableStateOf("")
    /** 弹窗里列出的已装应用。 */
    var vpnPickerApps by mutableStateOf<List<com.app.helper.vpn.VpnController.AppEntry>>(emptyList())

    // ---- 指纹 ----
    var fpCount by mutableIntStateOf(0)
    val fpList = mutableStateListOf<FingerprintManager.FpFileInfo>()
    /** 环境页多选删除：选中的指纹文件名。 */
    val fpSelected = mutableStateListOf<String>()

    /** 当前延时是否匹配某个预设区间，用于高亮按钮。 */
    fun isDelay(min: Long, max: Long): Boolean =
        delayMinMs == min && delayMaxMs == max

    fun formatDelay(): String = "%.1fs ~ %.1fs".format(delayMinMs / 1000f, delayMaxMs / 1000f)
}
