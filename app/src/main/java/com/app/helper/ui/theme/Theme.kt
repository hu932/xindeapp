package com.app.helper.ui.theme

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

/**
 * 配色：深色为默认。
 *
 * 这个工具长期挂在后台，深色更省电也不刺眼。
 * 主色用青绿（#22C1A4）——和「运行中/成功」的语义一致，
 * 危险操作用暖红，避免用系统红（太扎眼）。
 */
private val AccentGreen = Color(0xFF22C1A4)
private val AccentAmber = Color(0xFFF5A524)
private val AccentRed = Color(0xFFF04E4E)
private val AccentBlue = Color(0xFF4C8DFF)

private val DarkScheme = darkColorScheme(
    primary = AccentGreen,
    onPrimary = Color(0xFF00201B),
    primaryContainer = Color(0xFF0F3D36),
    onPrimaryContainer = Color(0xFF8CF0DC),
    secondary = AccentBlue,
    onSecondary = Color(0xFF001B3D),
    tertiary = AccentAmber,
    onTertiary = Color(0xFF2B1700),
    error = AccentRed,
    onError = Color(0xFF40000A),
    errorContainer = Color(0xFF4A0D12),
    onErrorContainer = Color(0xFFFFD9DB),
    background = Color(0xFF0E1116),
    onBackground = Color(0xFFE6E9EF),
    surface = Color(0xFF151A21),
    onSurface = Color(0xFFE6E9EF),
    surfaceVariant = Color(0xFF1E252F),
    onSurfaceVariant = Color(0xFF9BA6B4),
    outline = Color(0xFF2C3542),
    outlineVariant = Color(0xFF222A34),
)

private val LightScheme = lightColorScheme(
    primary = Color(0xFF0F8F79),
    onPrimary = Color.White,
    primaryContainer = Color(0xFFC9F2E9),
    onPrimaryContainer = Color(0xFF00201B),
    secondary = Color(0xFF2F6FE0),
    onSecondary = Color.White,
    tertiary = Color(0xFFB97500),
    onTertiary = Color.White,
    error = Color(0xFFC62828),
    onError = Color.White,
    background = Color(0xFFF6F8FB),
    onBackground = Color(0xFF16202C),
    surface = Color.White,
    onSurface = Color(0xFF16202C),
    surfaceVariant = Color(0xFFEDF1F6),
    onSurfaceVariant = Color(0xFF5C6875),
    outline = Color(0xFFD3DBE5),
    outlineVariant = Color(0xFFE5EAF1),
)

@Composable
fun HelperTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val scheme = if (darkTheme) DarkScheme else LightScheme
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as? Activity)?.window ?: return@SideEffect
            window.statusBarColor = scheme.background.toArgb()
            window.navigationBarColor = scheme.background.toArgb()
            WindowCompat.getInsetsController(window, view).apply {
                isAppearanceLightStatusBars = !darkTheme
                isAppearanceLightNavigationBars = !darkTheme
            }
        }
    }
    MaterialTheme(colorScheme = scheme, content = content)
}
