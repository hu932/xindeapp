package com.app.helper.util

import android.content.Context
import android.os.Build
import java.io.File
import java.io.PrintWriter
import java.io.StringWriter
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * 全局崩溃捕获。
 *
 * 之前闪退后什么都没留下，只能靠“跑任务时偶尔闪”这种描述猜。
 * 这里把未捕获异常连同运行状态一起写到磁盘：
 *
 *   /sdcard/Android/data/<pkg>/files/crash/crash_<时间>.txt
 *
 * 内容包含：异常堆栈、线程名、内存水位、任务状态快照。
 * 只保留最近 [MAX_FILES] 个，避免占用空间。
 *
 * 注意：这里任何代码都可能再次抛异常，全部包在 runCatching 里，
 * 写日志失败绝不能影响原异常的传递。
 */
object CrashHandler {

    private const val MAX_FILES = 5
    private const val DIR_NAME = "crash"

    /** 任务状态快照的来源，由 TaskService 注册。崩溃时调用。 */
    @Volatile
    var stateProvider: (() -> String)? = null

    @Volatile
    private var installed = false

    fun install(context: Context) {
        if (installed) return
        installed = true
        val app = context.applicationContext
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            runCatching { write(app, thread, throwable) }
            // 交回系统，保持原有的崩溃行为（别吞掉，否则 ANR 更难查）
            runCatching { previous?.uncaughtException(thread, throwable) }
        }
    }

    private fun write(context: Context, thread: Thread, throwable: Throwable) {
        val dir = File(context.filesDir, DIR_NAME)
        if (!dir.exists()) dir.mkdirs()

        val ts = SimpleDateFormat("yyyyMMdd_HHmmss", Locale.US).format(Date())
        val file = File(dir, "crash_$ts.txt")

        val sw = StringWriter()
        PrintWriter(sw).use { throwable.printStackTrace(it) }

        val sb = StringBuilder()
        sb.appendLine("=== 崩溃报告 ===")
        sb.appendLine("时间: ${SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US).format(Date())}")
        sb.appendLine("线程: ${thread.name}  (id=${thread.id})")
        sb.appendLine("异常: ${throwable.javaClass.name}")
        sb.appendLine("信息: ${throwable.message}")
        sb.appendLine()
        sb.appendLine("--- 设备 ---")
        sb.appendLine("型号: ${Build.MANUFACTURER} ${Build.MODEL}")
        sb.appendLine("系统: Android ${Build.VERSION.RELEASE} (SDK ${Build.VERSION.SDK_INT})")
        sb.appendLine("ABI: ${Build.SUPPORTED_ABIS.joinToString()}")
        sb.appendLine()
        sb.appendLine("--- 内存 ---")
        runCatching {
            val rt = Runtime.getRuntime()
            val max = rt.maxMemory() / 1024 / 1024
            val total = rt.totalMemory() / 1024 / 1024
            val free = rt.freeMemory() / 1024 / 1024
            sb.appendLine("堆上限: ${max}MB  已分配: ${total}MB  空闲: ${free}MB  已用: ${total - free}MB")
        }
        runCatching {
            val am = context.getSystemService(Context.ACTIVITY_SERVICE) as android.app.ActivityManager
            val mi = android.app.ActivityManager.MemoryInfo()
            am.getMemoryInfo(mi)
            sb.appendLine("系统可用: ${mi.availMem / 1024 / 1024}MB  低内存: ${mi.lowMemory}")
        }
        sb.appendLine()
        sb.appendLine("--- 任务状态 ---")
        runCatching { sb.appendLine(stateProvider?.invoke() ?: "（未注册状态回调）") }
        sb.appendLine()
        sb.appendLine("--- 堆栈 ---")
        sb.append(sw.toString())

        file.writeText(sb.toString())

        // 只保留最近几个
        runCatching {
            dir.listFiles()
                ?.filter { it.isFile && it.name.startsWith("crash_") }
                ?.sortedByDescending { it.lastModified() }
                ?.drop(MAX_FILES)
                ?.forEach { it.delete() }
        }
    }

    /** 供界面读取最近的崩溃报告。 */
    fun latest(context: Context): String? {
        val dir = File(context.filesDir, DIR_NAME)
        val f = dir.listFiles()
            ?.filter { it.isFile && it.name.startsWith("crash_") }
            ?.maxByOrNull { it.lastModified() }
            ?: return null
        return runCatching { f.readText() }.getOrNull()
    }

    fun clear(context: Context) {
        runCatching {
            File(context.filesDir, DIR_NAME).listFiles()?.forEach { it.delete() }
        }
    }
}
