package com.app.helper

import android.app.Application
import android.content.Context
import com.app.helper.util.CrashHandler

class App : Application() {
    companion object {
        lateinit var context: Context
            private set
    }

    override fun onCreate() {
        super.onCreate()
        context = applicationContext
        // 装得越早越好：闪退之后再装就抓不到了
        CrashHandler.install(this)
    }
}
