package com.app.helper.ui.compose

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Bolt
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.app.helper.ui.theme.HelperTheme

/** 后台模式检测状态。 */
enum class ModeBadge { DETECTING, NORMAL, INVENTORY }

/**
 * 登录界面。
 *
 * 保留原版行为：顶部模式徽标每 3 秒轮询后台库存开关，
 * 登录成功后写入 token / username / groupId / apiType / accountCount。
 */
@Composable
fun LoginScreen(
    accountId: String,
    onAccountIdChange: (String) -> Unit,
    modeBadge: ModeBadge,
    loading: Boolean,
    errorText: String,
    onLogin: () -> Unit,
) {
    HelperTheme {
        val focus = LocalFocusManager.current
        val badgeAlpha by animateFloatAsState(
            targetValue = if (modeBadge == ModeBadge.DETECTING) 0.55f else 1f,
            animationSpec = tween(400),
            label = "badge",
        )
        Surface(color = MaterialTheme.colorScheme.background, modifier = Modifier.fillMaxSize()) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .imePadding()
                    .padding(horizontal = 28.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                // ---- 图标 ----
                Box(
                    Modifier
                        .size(72.dp)
                        .clip(RoundedCornerShape(22.dp))
                        .background(
                            Brush.linearGradient(
                                listOf(
                                    MaterialTheme.colorScheme.primary,
                                    MaterialTheme.colorScheme.secondary,
                                )
                            )
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        Icons.Filled.Bolt,
                        contentDescription = null,
                        tint = Color.White,
                        modifier = Modifier.size(38.dp),
                    )
                }

                Spacer(Modifier.height(22.dp))

                Text(
                    "Ajie",
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onBackground,
                )
                Spacer(Modifier.height(6.dp))
                Text(
                    "普通 · 库存 双模式任务自动化",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )

                Spacer(Modifier.height(18.dp))

                // ---- 模式徽标 ----
                val (badgeText, badgeColor) = when (modeBadge) {
                    ModeBadge.INVENTORY -> "⚡ 库存模式" to Color(0xFFF5A524)
                    ModeBadge.NORMAL -> "✓ 普通模式" to MaterialTheme.colorScheme.primary
                    ModeBadge.DETECTING -> "正在检测后台模式..." to MaterialTheme.colorScheme.onSurfaceVariant
                }
                Row(
                    modifier = Modifier
                        .clip(CircleShape)
                        .background(badgeColor.copy(alpha = 0.12f * badgeAlpha))
                        .padding(horizontal = 14.dp, vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        Modifier
                            .size(6.dp)
                            .clip(CircleShape)
                            .background(badgeColor.copy(alpha = badgeAlpha))
                    )
                    Spacer(Modifier.width(7.dp))
                    Text(
                        badgeText,
                        style = MaterialTheme.typography.labelMedium,
                        color = badgeColor.copy(alpha = badgeAlpha),
                        fontWeight = FontWeight.Medium,
                    )
                }

                Spacer(Modifier.height(34.dp))

                // ---- 输入 ----
                Text(
                    "账号登录",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = accountId,
                    onValueChange = { onAccountIdChange(it.trim()) },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text("下发id / 手机号") },
                    leadingIcon = {
                        Icon(
                            Icons.Filled.Person,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.size(19.dp),
                        )
                    },
                    singleLine = true,
                    enabled = !loading,
                    shape = RoundedCornerShape(12.dp),
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { focus.clearFocus(); onLogin() }),
                )

                // ---- 错误 ----
                AnimatedVisibility(visible = errorText.isNotBlank()) {
                    Text(
                        errorText,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 8.dp),
                    )
                }

                Spacer(Modifier.height(18.dp))

                // ---- 登录按钮 ----
                Button(
                    onClick = { focus.clearFocus(); onLogin() },
                    enabled = !loading && accountId.isNotBlank(),
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(50.dp),
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.primary,
                    ),
                ) {
                    if (loading) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                            color = MaterialTheme.colorScheme.onPrimary,
                        )
                        Spacer(Modifier.width(10.dp))
                        Text("登录中...", fontSize = 15.sp)
                    } else {
                        Text("登 录", fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
                    }
                }

                Spacer(Modifier.height(30.dp))

                Text(
                    "后台未开启库存 = 普通模式\n开启库存 = 库存模式",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f),
                    textAlign = TextAlign.Center,
                    lineHeight = 18.sp,
                )
            }
        }
    }
}
