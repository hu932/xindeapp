package com.app.helper.ui.compose

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Fingerprint
import androidx.compose.material.icons.filled.PlaylistAdd
import androidx.compose.material3.Button
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun EnvPage(
    state: UiState,
    onGenerateFp: () -> Unit,
    onBatchGenerateFp: () -> Unit,
    onFpClick: (FingerprintManagerFp) -> Unit,
    onFpDelete: (List<String>) -> Unit,
) {
    var confirming by remember { mutableStateOf(false) }
    Column(
        Modifier
            .fillMaxSize()
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        // ---- 顶部：数量 + 生成 ----
        Card(
            Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(14.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        ) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(
                        Icons.Filled.Fingerprint,
                        null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(22.dp),
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        "指纹环境",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.weight(1f),
                    )
                    Text(
                        "${state.fpCount} 个",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(
                        onClick = onGenerateFp,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                    ) {
                        Icon(Icons.Filled.AutoAwesome, null, Modifier.size(17.dp))
                        Spacer(Modifier.width(4.dp))
                        Text("生成")
                    }
                    OutlinedButton(
                        onClick = onBatchGenerateFp,
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                    ) {
                        Icon(Icons.Filled.PlaylistAdd, null, Modifier.size(17.dp))
                        Spacer(Modifier.width(4.dp))
                        Text("批量")
                    }
                }
                if (state.fpSelected.isNotEmpty()) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Text(
                            "已选 ${state.fpSelected.size} 个",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.weight(1f),
                        )
                        TextButton(onClick = {
                            if (state.fpSelected.size == state.fpList.size) {
                                state.fpSelected.clear()
                            } else {
                                state.fpSelected.clear()
                                state.fpList.forEach { state.fpSelected.add(it.name) }
                            }
                        }) { Text("全选/取消", fontSize = 12.sp) }
                        Button(
                            onClick = { confirming = true },
                            shape = RoundedCornerShape(10.dp),
                            colors = ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.error,
                            ),
                        ) { Text("删除", fontSize = 12.sp) }
                    }
                }
            }
        }

        // ---- 列表 ----
        Card(
            Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(14.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        ) {
            Column(Modifier.padding(14.dp)) {
                Text(
                    "指纹列表",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.height(8.dp))
                if (state.fpList.isEmpty()) {
                    Text(
                        "还没有指纹，点上面的「生成」创建一个",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(vertical = 20.dp),
                    )
                } else {
                    LazyColumn(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        items(state.fpList) { fp ->
                            val selected = state.fpSelected.contains(fp.name)
                            FpRow(
                                fp = fp,
                                selected = selected,
                                onClick = {
                                    if (state.fpSelected.isNotEmpty()) {
                                        if (selected) state.fpSelected.remove(fp.name)
                                        else state.fpSelected.add(fp.name)
                                    } else {
                                        onFpClick(fp)
                                    }
                                },
                                onLongClick = {
                                    if (selected) state.fpSelected.remove(fp.name)
                                    else state.fpSelected.add(fp.name)
                                },
                            )
                        }
                    }
                }
            }
        }
    }

    if (confirming) {
        AlertDialog(
            onDismissRequest = { confirming = false },
            title = { Text("删除环境") },
            text = {
                Text(
                    "确定删除选中的 ${state.fpSelected.size} 个环境？" +
                            "删除后不可恢复，正在使用的那条也会一起删掉。",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    onFpDelete(state.fpSelected.toList())
                    state.fpSelected.clear()
                    confirming = false
                }) {
                    Text("删除", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { confirming = false }) { Text("取消") }
            },
        )
    }
}

@Composable
private fun FpRow(
    fp: FingerprintManagerFp,
    selected: Boolean = false,
    onClick: () -> Unit,
    onLongClick: () -> Unit = {},
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(
                when {
                    selected -> MaterialTheme.colorScheme.secondaryContainer
                    fp.isActive -> MaterialTheme.colorScheme.primaryContainer
                    else -> MaterialTheme.colorScheme.surfaceVariant
                }
            )
            .combinedClickable(onClick = onClick, onLongClick = onLongClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (selected) {
            Icon(
                Icons.Filled.CheckCircle,
                null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
        } else if (fp.isActive) {
            Icon(
                Icons.Filled.CheckCircle,
                null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(
                fp.name,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = if (fp.isActive) FontWeight.SemiBold else FontWeight.Normal,
                color = if (fp.isActive) MaterialTheme.colorScheme.onPrimaryContainer
                else MaterialTheme.colorScheme.onSurface,
            )
            Text(
                "${fp.size / 1024} KB · ${formatTime(fp.lastModified)}",
                style = MaterialTheme.typography.labelSmall,
                color = if (fp.isActive) MaterialTheme.colorScheme.onPrimaryContainer
                else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun formatTime(ms: Long): String =
    runCatching {
        SimpleDateFormat("MM-dd HH:mm", Locale.getDefault()).format(Date(ms))
    }.getOrDefault("-")
