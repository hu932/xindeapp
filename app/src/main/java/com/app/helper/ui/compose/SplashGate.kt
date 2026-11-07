package com.app.helper.ui.compose

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
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
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.app.helper.R
import kotlinx.coroutines.delay

/**
 * 启动动效（纯黑 + 真实 App 图标 + 文案收拢）。
 *
 * 时序（总计约 2.6s，可按需调 BRAND_MS / HOLD_MS / OUT_MS）：
 *   0    - 500ms  图标从 0.90 放大到 1.0 并淡入
 *   300  - 900ms  标题字距从 18dp 收拢到 0，同时淡入
 *   900  - 1500ms 副标题淡入
 *   1500 - 2100ms 静止
 *   2100 - 2600ms 整层淡出，露出已就绪的登录页
 *
 * 关键：登录页在底下同时组合，动画只做覆盖层，不做任何 IO，冷启动不会卡。
 */
private const val BRAND_MS = 500
private const val HOLD_MS = 1500
private const val OUT_MS = 500

@Composable
fun SplashGate(content: @Composable () -> Unit) {
    var phase by remember { mutableStateOf(0) }   // 0 入场 1 静止 2 淡出 3 消失
    var entered by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        entered = true
        delay((BRAND_MS + HOLD_MS).toLong())
        phase = 2
        delay(OUT_MS.toLong())
        phase = 3
    }

    val brandScale by animateFloatAsState(
        targetValue = if (entered) 1f else 0.90f,
        animationSpec = tween(durationMillis = BRAND_MS),
        label = "brandScale"
    )
    val brandAlpha by animateFloatAsState(
        targetValue = if (entered) 1f else 0f,
        animationSpec = tween(durationMillis = BRAND_MS),
        label = "brandAlpha"
    )
    val outAlpha by animateFloatAsState(
        targetValue = if (phase >= 2) 0f else 1f,
        animationSpec = tween(durationMillis = if (phase >= 2) OUT_MS else 1),
        label = "outAlpha"
    )

    Box(Modifier.fillMaxSize().background(Color.Black)) {
        content()

        if (phase < 3) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(Color.Black)
                    .alpha(outAlpha),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(18.dp)
                ) {
                    Image(
                        painter = painterResource(R.mipmap.ic_launcher),
                        contentDescription = null,
                        modifier = Modifier
                            .size(112.dp)
                            .scale(brandScale)
                            .alpha(brandAlpha)
                    )
                    Text(
                        text = stringResource(R.string.app_name),
                        color = Color.White,
                        fontSize = 30.sp,
                        fontWeight = FontWeight.Light,
                        letterSpacing = (18 * (1f - brandAlpha)).sp,
                        modifier = Modifier.alpha(brandAlpha)
                    )
                    Text(
                        text = "正在加载环境",
                        color = Color(0xFF8A8F98),
                        fontSize = 13.sp,
                        letterSpacing = 4.sp,
                        modifier = Modifier.alpha(brandAlpha)
                    )
                }
            }
        }
    }
}
