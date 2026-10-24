package com.app.helper.ui.compose

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DeleteSweep
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

@Composable
fun TaskPage(
    state: UiState,
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
    onVpnToggle: (Boolean) -> Unit,
    onVpnConnect: () -> Unit,
    onInstallVpn: () -> Unit,
    onVpnRecheck: () -> Unit,
    onVpnSwitch: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        StatusCard(state)
        ControlCard(state, onStartAuto, onStopAuto, onSingleTask, onTestApi, onCheckUpdate)
        DelayCard(state, onSetDelay)
        PauseCard(state, onPauseEnabled, onPauseConfig)
        VpnCard(state, onVpnToggle, onVpnConnect, onInstallVpn, onVpnRecheck, onVpnSwitch)
        ModeCard(state, onSilentMode, onChatMode, onChatPermission)
        LogCard(state, onClearLogs)
    }
}

@Composable
private fun SectionCard(title: String, content: @Composable () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text(
                title,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                fontWeight = FontWeight.SemiBold,
            )
            content()
        }
    }
}

@Composable
private fun StatusCard(state: UiState) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    Modifier
                        .size(8.dp)
                        .clip(RoundedCornerShape(50))
                        .background(
                            if (state.autoRunning) MaterialTheme.colorScheme.primary
                            else MaterialTheme.colorScheme.outline
                        )
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    state.status,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
            }
            if (state.currentTask.isNotBlank()) {
                Text(
                    state.currentTask,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                MetricTile("完成", state.completed.toString(), MaterialTheme.colorScheme.primary, Modifier.weight(1f))
                MetricTile("失败", state.failed.toString(), MaterialTheme.colorScheme.error, Modifier.weight(1f))
                MetricTile("延时", state.formatDelay(), MaterialTheme.colorScheme.onSurfaceVariant, Modifier.weight(1f))
            }
        }
    }
}

@Composable
private fun MetricTile(label: String, value: String, color: Color, modifier: Modifier = Modifier) {
    Column(
        modifier
            .clip(RoundedCornerShape(10.dp))
            .background(MaterialTheme.colorScheme.surfaceVariant)
            .padding(vertical = 10.dp, horizontal = 12.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Spacer(Modifier.height(2.dp))
        Text(value, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold, color = color)
    }
}

@Composable
private fun ControlCard(
    state: UiState,
    onStartAuto: () -> Unit,
    onStopAuto: () -> Unit,
    onSingleTask: () -> Unit,
    onTestApi: () -> Unit,
    onCheckUpdate: () -> Unit,
) {
    SectionCard("任务控制") {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(
                onClick = onStartAuto,
                enabled = !state.autoRunning,
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
            ) {
                Icon(Icons.Filled.PlayArrow, null, Modifier.size(18.dp))
                Spacer(Modifier.width(4.dp))
                Text("开始")
            }
            Button(
                onClick = onStopAuto,
                enabled = state.autoRunning,
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.errorContainer,
                    contentColor = MaterialTheme.colorScheme.onErrorContainer,
                ),
            ) {
                Icon(Icons.Filled.Stop, null, Modifier.size(18.dp))
                Spacer(Modifier.width(4.dp))
                Text("停止")
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedButton(
                onClick = onSingleTask,
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
            ) { Text("单个任务") }
            OutlinedButton(
                onClick = onTestApi,
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
            ) { Text("测试接口") }
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "目标版本 ${state.targetVersion.ifBlank { "-" }}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.weight(1f),
            )
            TextButton(onClick = onCheckUpdate) {
                Icon(Icons.Filled.Refresh, null, Modifier.size(15.dp))
                Spacer(Modifier.width(4.dp))
                Text(if (state.updateAvailable) "有更新" else "检查更新")
            }
        }
    }
}

@Composable
private fun DelayCard(state: UiState, onSetDelay: (Long, Long) -> Unit) {
    SectionCard("延时设置") {
        val presets = listOf(
            Triple("极速", 100L, 1_000L),
            Triple("快速", 1_000L, 3_000L),
            Triple("常规", 3_000L, 8_000L),
            Triple("平稳", 5_000L, 15_000L),
            Triple("慢速", 30_000L, 60_000L),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            presets.forEach { (label, min, max) ->
                val active = state.isDelay(min, max)
                if (active) {
                    Button(
                        onClick = { onSetDelay(min, max) },
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(9.dp),
                        contentPadding = PaddingValues(horizontal = 6.dp, vertical = 8.dp),
                    ) { Text(label, fontSize = 12.sp, maxLines = 1) }
                } else {
                    OutlinedButton(
                        onClick = { onSetDelay(min, max) },
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(9.dp),
                        contentPadding = PaddingValues(horizontal = 6.dp, vertical = 8.dp),
                    ) { Text(label, fontSize = 12.sp, maxLines = 1) }
                }
            }
        }
    }
}

@Composable
private fun PauseCard(
    state: UiState,
    onPauseEnabled: (Boolean) -> Unit,
    onPauseConfig: (Int, Int) -> Unit,
) {
    SectionCard("批量暂停") {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("启用", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
            Switch(checked = state.pauseEnabled, onCheckedChange = onPauseEnabled)
        }
        if (state.pauseEnabled) {
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedTextField(
                    value = state.pauseCount.toString(),
                    onValueChange = { v ->
                        v.toIntOrNull()?.let { onPauseConfig(it.coerceIn(1, 9999), state.pauseMinutes) }
                    },
                    label = { Text("每多少个") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(10.dp),
                )
                OutlinedTextField(
                    value = state.pauseMinutes.toString(),
                    onValueChange = { v ->
                        v.toIntOrNull()?.let { onPauseConfig(state.pauseCount, it.coerceIn(1, 600)) }
                    },
                    label = { Text("停几分钟") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(10.dp),
                )
            }
        }
    }
}

@Composable
private fun ModeCard(
    state: UiState,
    onSilentMode: (Boolean) -> Unit,
    onChatMode: (Boolean) -> Unit,
    onChatPermission: () -> Unit,
) {
    SectionCard("运行模式") {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("静默模式", style = MaterialTheme.typography.bodyMedium)
                Text(
                    "不跳转商品页，由环境内部抓取",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Switch(checked = state.silentMode, onCheckedChange = onSilentMode)
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("聊天模式", style = MaterialTheme.typography.bodyMedium)
                Text(
                    state.chatStatus,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (!state.chatPermissionOk) {
                TextButton(onClick = onChatPermission) { Text("授权") }
            }
            Switch(checked = state.chatMode, onCheckedChange = onChatMode)
        }
    }
}

@Composable
private fun LogCard(state: UiState, onClearLogs: () -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(Modifier.padding(14.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "运行日志",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f),
                )
                TextButton(onClick = onClearLogs) {
                    Icon(Icons.Filled.DeleteSweep, null, Modifier.size(15.dp))
                    Spacer(Modifier.width(4.dp))
                    Text("清空")
                }
            }
            Spacer(Modifier.height(6.dp))
            Box(
                Modifier
                    .fillMaxWidth()
                    .heightIn(min = 160.dp, max = 260.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(MaterialTheme.colorScheme.background)
                    .padding(10.dp),
            ) {
                if (state.logs.isEmpty()) {
                    Text(
                        "暂无日志",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    Column {
                        state.logs.forEach { line ->
                            Text(
                                line,
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 11.sp,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(vertical = 1.dp),
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun VpnCard(
    state: UiState,
    onVpnToggle: (Boolean) -> Unit,
    onVpnConnect: () -> Unit,
    onInstallVpn: () -> Unit,
    onVpnRecheck: () -> Unit,
    onVpnSwitch: () -> Unit,
) {
    SectionCard("网络代理") {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("自动代理", style = MaterialTheme.typography.bodyMedium)
                Text(
                    state.vpnStatus,
                    style = MaterialTheme.typography.labelSmall,
                    color = when {
                        !state.vpnTargetInstalled -> MaterialTheme.colorScheme.error
                        state.vpnEnabled -> MaterialTheme.colorScheme.primary
                        else -> MaterialTheme.colorScheme.onSurfaceVariant
                    },
                )
            }
            Switch(
                checked = state.vpnEnabled,
                onCheckedChange = onVpnToggle,
                enabled = state.vpnTargetInstalled,
            )
        }
        if (!state.vpnTargetInstalled) {
            if (state.vpnDownloadPct >= 0) {
                Text(
                    "正在下载 v2rayNG… " + state.vpnDownloadPct + "%",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary,
                )
            } else {
                Text(
                    "需要 v2rayNG 才能自动切换节点",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(
                        onClick = onInstallVpn,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                        enabled = !state.vpnInstalling,
                    ) {
                        Text(if (state.vpnInstalling) "处理中…" else "在线安装", fontSize = 13.sp)
                    }
                    OutlinedButton(
                        onClick = onVpnRecheck,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                    ) {
                        Text("重新检测", fontSize = 13.sp)
                    }
                }
                if (state.vpnDetectLog.isNotBlank()) {
                    Text(
                        state.vpnDetectLog,
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        fontSize = 10.sp,
                    )
                }
            }
        } else {
            if (state.vpnDetectLog.isNotBlank()) {
                Text(
                    state.vpnDetectLog,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontSize = 10.sp,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(
                    onClick = onVpnConnect,
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(10.dp),
                ) { Text("连接最快节点", fontSize = 13.sp) }
                if (state.vpnAppChoices.size > 1) {
                    OutlinedButton(
                        onClick = onVpnSwitch,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                    ) { Text("切换应用", fontSize = 13.sp) }
                } else {
                    OutlinedButton(
                        onClick = onVpnRecheck,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                    ) { Text("重新检测", fontSize = 13.sp) }
                }
            }
            if (state.vpnRegion.isNotBlank()) {
                Text(
                    "地区筛选：${state.vpnRegion}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
