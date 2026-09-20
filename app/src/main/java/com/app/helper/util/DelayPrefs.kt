package com.app.helper.util

import android.content.Context

/**
 * 延时档位存储。统一使用毫秒，便于支持 0.1s 这种亚秒级档位。
 * 旧版本用「delay_min / delay_max（秒，Int）」存储，首次读取时自动迁移。
 */
object DelayPrefs {

    private const val PREF_NAME = "task_config"
    private const val KEY_MIN_MS = "delay_min_ms"
    private const val KEY_MAX_MS = "delay_max_ms"
    private const val LEGACY_MIN = "delay_min"
    private const val LEGACY_MAX = "delay_max"

    private fun prefs(ctx: Context) = ctx.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)

    private fun migrate(ctx: Context) {
        val p = prefs(ctx)
        if (p.contains(KEY_MIN_MS) || p.contains(KEY_MAX_MS)) return
        val minMs = (p.getInt(LEGACY_MIN, 15) * 1000L).coerceAtLeast(0L)
        val maxMs = (p.getInt(LEGACY_MAX, 35) * 1000L).coerceAtLeast(minMs)
        p.edit()
            .putLong(KEY_MIN_MS, minMs)
            .putLong(KEY_MAX_MS, maxMs)
            .remove(LEGACY_MIN)
            .remove(LEGACY_MAX)
            .apply()
    }

    fun getMinMs(ctx: Context): Long {
        migrate(ctx)
        return prefs(ctx).getLong(KEY_MIN_MS, 15_000L)
    }

    fun getMaxMs(ctx: Context): Long {
        migrate(ctx)
        return prefs(ctx).getLong(KEY_MAX_MS, 35_000L)
    }

    fun set(ctx: Context, minMs: Long, maxMs: Long) {
        val min = minMs.coerceAtLeast(0L)
        val max = maxMs.coerceAtLeast(min)
        prefs(ctx).edit()
            .putLong(KEY_MIN_MS, min)
            .putLong(KEY_MAX_MS, max)
            .remove(LEGACY_MIN)
            .remove(LEGACY_MAX)
            .apply()
    }

    /** 显示用：整秒显示「3s」，非整秒显示「0.1s」 */
    fun formatMs(ms: Long): String {
        if (ms % 1000L == 0L) return "${ms / 1000L}s"
        val sec = ms / 1000.0
        val rounded = Math.round(sec * 100) / 100.0
        val str = if (rounded == Math.floor(rounded)) rounded.toLong().toString() else rounded.toString()
        return "${str}s"
    }
}
