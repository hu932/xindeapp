package com.app.helper.ui.compose

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color

/**
 * 启动渐入 —— 不再有独立的「闪屏页」。
 *
 * 之前白屏的原因不是动画，而是 Activity 窗口在 Compose 画第一帧之前露的是主题白底。
 * 现在：
 *   1. ComposeLoginActivity.onCreate 里先把 window 背景设成同一个深色（第一帧就是深色，不闪白）
 *   2. 登录页内容自身 alpha 0→1、scale 0.985→1 渐入（420~460ms）
 *   3. 背后铺一层竖直渐变，避免纯色显得空旷
 *
 * 没有任何额外页面和延时，所以不会出现「白屏 → 动画 → 页面」三段跳。
 */
@Composable
fun SplashGate(content: @Composable () -> Unit) {
    var entered by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) { entered = true }

    val alpha by animateFloatAsState(
        targetValue = if (entered) 1f else 0f,
        animationSpec = tween(durationMillis = 420),
        label = "contentAlpha"
    )
    val scale by animateFloatAsState(
        targetValue = if (entered) 1f else 0.985f,
        animationSpec = tween(durationMillis = 460),
        label = "contentScale"
    )

    Box(
        Modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    listOf(Color(0xFF0B1220), Color(0xFF101A2B), Color(0xFF0B1220))
                )
            )
    ) {
        Box(
            Modifier
                .fillMaxSize()
                .alpha(alpha)
                .scale(scale)
        ) {
            content()
        }
    }
}
