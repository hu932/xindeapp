package com.app.helper.ui.compose

import com.app.helper.R

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay

/**
 * 启动动效：
 *   - 内容层（登录页）立刻组合，用户不会看到空白
 *   - 上面盖一层渐变遮罩 + 缩放淡入的标识，600ms 后淡出移除
 *
 * 关键点：动画只是"贴"在最上层，不做任何 IO / 阻塞，
 * 所以不会像传统 SplashActivity 那样拖慢冷启动。
 */
@Composable
fun SplashGate(content: @Composable () -> Unit) {
    var visible by remember { mutableStateOf(true) }
    var entered by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        entered = true
        delay(650)
        visible = false
    }

    val scale by animateFloatAsState(
        targetValue = if (entered) 1f else 0.82f,
        animationSpec = tween(durationMillis = 480),
        label = "splashScale"
    )
    val alpha by animateFloatAsState(
        targetValue = if (entered) 1f else 0f,
        animationSpec = tween(durationMillis = 420),
        label = "splashAlpha"
    )

    Box(Modifier.fillMaxSize()) {
        content()

        AnimatedVisibility(
            visible = visible,
            enter = fadeIn(tween(200)) + scaleIn(initialScale = 0.96f, animationSpec = tween(240)),
            exit = fadeOut(tween(260))
        ) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(
                        Brush.verticalGradient(
                            listOf(Color(0xFF0B1220), Color(0xFF131C2F), Color(0xFF0B1220))
                        )
                    ),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(14.dp)
                ) {
                    Box(
                        Modifier
                            .size(96.dp)
                            .scale(scale)
                            .alpha(alpha)
                            .clip(RoundedCornerShape(28.dp))
                            .background(
                                Brush.linearGradient(
                                    listOf(Color(0xFF3B82F6), Color(0xFF8B5CF6))
                                )
                            ),
                        contentAlignment = Alignment.Center
                    ) {
                        Text("A", color = Color.White, fontSize = 44.sp, fontWeight = FontWeight.Bold)
                    }
                    // 标题取 App 自己的名字（strings.xml: app_name = Ajie），不再写死文案
                    Text(
                        stringResource(R.string.app_name),
                        color = Color.White,
                        fontSize = 22.sp,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.alpha(alpha)
                    )
                    Text(
                        "正在准备运行环境…",
                        color = Color(0xFF94A3B8),
                        fontSize = 13.sp,
                        modifier = Modifier.padding(top = 2.dp).alpha(alpha)
                    )
                }
            }
        }
    }
}
