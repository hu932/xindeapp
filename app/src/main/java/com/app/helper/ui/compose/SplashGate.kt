package com.app.helper.ui.compose

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
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

/**
 * 仅做内容渐入（约 260ms）。
 *
 * 品牌动画已交给启动视频页（SplashVideoActivity），这里不再放图标/黑底/延时，
 * 否则会出现「视频 → 图标动画 → 页面」三段跳。
 */
@Composable
fun SplashGate(content: @Composable () -> Unit) {
    var entered by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) { entered = true }
    val alpha by animateFloatAsState(
        targetValue = if (entered) 1f else 0f,
        animationSpec = tween(durationMillis = 260),
        label = "contentFade"
    )
    Box(Modifier.fillMaxSize().alpha(alpha)) {
        content()
    }
}
