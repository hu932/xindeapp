package com.app.helper.ui.compose

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Logout
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.app.helper.ui.theme.HelperTheme

/**
 * 主界面骨架。
 *
 * 结构：顶栏（账号 + 退出）→ 页签（任务 / 环境）→ 内容。
 * 状态全部由 [UiState] 驱动，界面本身无状态。
 */
@Composable
fun HelperApp(
    state: UiState,
    onLogout: () -> Unit,
    onStartAuto: () -> Unit,
    onStopAuto: () -> Unit,
    onSingleTask: () -> Unit,
    onTestApi: () -> Unit,
    onSetDelay: (Long, Long) -> Unit,
    onPauseEnabled: (Boolean) -> Unit,
    onPauseConfig: (Int, Int) -> Unit,
    onSilentMode: (Boolean) -> Unit,
    onChatMode: (Boolean) -> Unit,
    onChatPermission: () -> Unit,
    onClearLogs: () -> Unit,
    onCheckUpdate: () -> Unit,
    onGenerateFp: () -> Unit,
    onBatchGenerateFp: () -> Unit,
    onFpClick: (FingerprintManagerFp) -> Unit,
    onVpnToggle: (Boolean) -> Unit,
    onVpnConnect: () -> Unit,
    onInstallVpn: () -> Unit,
    onVpnRecheck: () -> Unit,
) {
    HelperTheme {
        Surface(color = MaterialTheme.colorScheme.background, modifier = Modifier.fillMaxSize()) {
            Column(Modifier.fillMaxSize()) {
                TopBar(state = state, onLogout = onLogout)
                TabBar(state = state)
                Box(Modifier.fillMaxSize()) {
                    when (state.tab) {
                        0 -> TaskPage(
                            state = state,
                            onStartAuto = onStartAuto,
                            onStopAuto = onStopAuto,
                            onSingleTask = onSingleTask,
                            onTestApi = onTestApi,
                            onSetDelay = onSetDelay,
                            onPauseEnabled = onPauseEnabled,
                            onPauseConfig = onPauseConfig,
                            onSilentMode = onSilentMode,
                            onChatMode = onChatMode,
                            onChatPermission = onChatPermission,
                            onClearLogs = onClearLogs,
                            onCheckUpdate = onCheckUpdate,
                            onVpnToggle = onVpnToggle,
                            onVpnConnect = onVpnConnect,
                            onInstallVpn = onInstallVpn,
                            onVpnRecheck = onVpnRecheck,
                        )
                        else -> EnvPage(
                            state = state,
                            onGenerateFp = onGenerateFp,
                            onBatchGenerateFp = onBatchGenerateFp,
                            onFpClick = onFpClick,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun TopBar(state: UiState, onLogout: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surface)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(34.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primaryContainer),
            contentAlignment = Alignment.Center,
        ) {
            val bmp = state.avatarBitmap
            if (bmp != null) {
                Image(
                    bitmap = bmp.asImageBitmap(),
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize(),
                )
            } else {
                Icon(
                    Icons.Filled.Person,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onPrimaryContainer,
                    modifier = Modifier.size(19.dp),
                )
            }
        }
        Spacer(Modifier.width(10.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = state.username.ifBlank { "未登录" },
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = if (state.autoRunning) "运行中" else "已停止",
                style = MaterialTheme.typography.labelSmall,
                color = if (state.autoRunning) MaterialTheme.colorScheme.primary
                else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        TextButton(onClick = onLogout) {
            Icon(
                Icons.Filled.Logout,
                contentDescription = null,
                modifier = Modifier.size(16.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.width(4.dp))
            Text("退出", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun TabBar(state: UiState) {
    val tabs = listOf("任务", "环境")
    TabRow(
        selectedTabIndex = state.tab,
        containerColor = MaterialTheme.colorScheme.surface,
        contentColor = MaterialTheme.colorScheme.primary,
    ) {
        tabs.forEachIndexed { index, title ->
            Tab(
                selected = state.tab == index,
                onClick = { state.tab = index },
                text = {
                    Text(
                        title,
                        fontWeight = if (state.tab == index) FontWeight.SemiBold else FontWeight.Normal,
                    )
                },
            )
        }
    }
}

/** 兼容层：让屏幕层不直接依赖 FingerprintManager 的内部类型命名。 */
typealias FingerprintManagerFp = com.app.helper.util.FingerprintManager.FpFileInfo
