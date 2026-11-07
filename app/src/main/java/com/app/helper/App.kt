package com.app.helper

import android.app.Application
import android.content.Context
import com.app.helper.util.CrashHandler

class App : Application() {
    companion object {
        lateinit var context: Context
            private set

        /** debug 包才输出网络日志，release 一律静默，避免打日志拖慢任务队列 */
        val isDebug: Boolean by lazy {
            (context.applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0
        }
    }

    override fun onCreate() {
        super.onCreate()
        context = applicationContext
        // 装得越早越好：闪退之后再装就抓不到了
        CrashHandler.install(this)
    }
}
