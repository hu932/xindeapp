package com.app.helper.ui.compose

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.app.helper.vpn.VpnController

/**
 * 手动选择代理应用。
 *
 * 自动检测在部分 ROM 上会被包可见性限制挡住，这时不能靠猜包名，
 * 直接把已装应用列出来让用户点。选中后写进 vpn_state 的 preferred_pkg，
 * 之后所有启停、推节点操作都用这个包名。
 */
@Composable
fun VpnPickerDialog(
    apps: List<VpnController.AppEntry>,
    currentPkg: String,
    onPick: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("选择代理应用") },
        text = {
            Column {
                Text(
                    "自动检测可能被系统限制挡住，请手动点选你的代理应用" +
                            "（v2rayNG、FlClash 等）。选错会导致无法开关代理。",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (apps.isEmpty()) {
                    Text(
                        "读取不到已装应用列表，可能是系统限制了应用可见性。",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                } else {
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(max = 380.dp)
                            .padding(top = 8.dp),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        items(apps, key = { it.pkg }) { app ->
                            val selected = app.pkg == currentPkg
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .background(
                                        if (selected) MaterialTheme.colorScheme.primaryContainer
                                        else MaterialTheme.colorScheme.surface,
                                        RoundedCornerShape(8.dp),
                                    )
                                    .clickable { onPick(app.pkg) }
                                    .padding(horizontal = 10.dp, vertical = 9.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column(Modifier.weight(1f)) {
                                    Text(
                                        app.label.ifBlank { app.pkg },
                                        style = MaterialTheme.typography.bodyMedium,
                                        fontWeight = if (selected) FontWeight.SemiBold
                                        else FontWeight.Normal,
                                        color = MaterialTheme.colorScheme.onSurface,
                                    )
                                    Text(
                                        app.pkg,
                                        fontSize = 10.sp,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                                if (selected) {
                                    Text(
                                        "当前",
                                        fontSize = 11.sp,
                                        color = MaterialTheme.colorScheme.primary,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) { Text("关闭") }
        },
    )
}
