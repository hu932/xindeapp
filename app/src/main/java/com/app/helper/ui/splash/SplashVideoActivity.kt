package com.app.helper.ui.splash

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.media.MediaPlayer
import android.net.Uri
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.view.Gravity
import android.view.View
import android.view.WindowManager
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import android.widget.VideoView
import com.app.helper.R
import com.app.helper.ui.compose.ComposeLoginActivity

/**
 * 启动页：在启动界面内流式播放后台下发的视频，播完再进 App。
 *
 * 解决「打开 App 黑屏、视频出不来」：
 *   1. 缓冲期间显示 图标 + 进度圈 + 「加载中」，不是纯黑
 *   2. prepare 超时 12s → 3.5s：网络不行立刻跳过
 *   3. 卡顿看门狗：播放后 4s 播放位置没有推进，视为卡死直接跳过
 *   4. 首帧渲染（MEDIA_INFO_VIDEO_RENDERING_START）后才撤掉加载层
 */
class SplashVideoActivity : Activity() {

    companion object {
        /** 自己的服务器（稳定、可控）；list.json 列出可用视频 */
        const val SERVER_BASE = "http://38.175.194.43"
        const val LIST_URL = SERVER_BASE + "/splash/list.json"

        /** list.json 拿不到时的兜底清单（与服务器一致） */
        val FALLBACK_VIDEOS = listOf(
            "splash_01.mp4", "splash_02.mp4", "splash_03.mp4",
            "splash_04.mp4", "splash_05.mp4", "splash_06.mp4",
        )
        const val EXTRA_URL = "splash_video_url"

        private const val PREPARE_TIMEOUT_MS = 3500L
        private const val STALL_TIMEOUT_MS = 4000L
        private const val HARD_TIMEOUT_MS = 30000L
    }

    private val ui = Handler(Looper.getMainLooper())
    private var finished = false
    private var videoView: VideoView? = null
    private var loading: FrameLayout? = null
    private var preparedAt = 0L
    private var lastPos = -1

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        window.statusBarColor = Color.BLACK
        window.navigationBarColor = Color.BLACK

        val density = resources.displayMetrics.density
        val rootLayout = FrameLayout(this)
        rootLayout.setBackgroundColor(Color.BLACK)

        val vv = VideoView(this)
        vv.setBackgroundColor(Color.BLACK)
        vv.layoutParams = FrameLayout.LayoutParams(-1, -1)
        rootLayout.addView(vv)
        videoView = vv

        // 加载层（缓冲期间可见，避免黑屏）
        val box = FrameLayout(this)
        box.layoutParams = FrameLayout.LayoutParams(-1, -1)

        val col = LinearLayout(this)
        col.orientation = LinearLayout.VERTICAL
        col.gravity = Gravity.CENTER_HORIZONTAL
        col.layoutParams = FrameLayout.LayoutParams(-2, -2, Gravity.CENTER)

        val icon = ImageView(this)
        icon.setImageResource(R.mipmap.ic_launcher)
        val s = (96 * density).toInt()
        col.addView(icon, LinearLayout.LayoutParams(s, s))

        val bar = ProgressBar(this)
        val b = (26 * density).toInt()
        val barLp = LinearLayout.LayoutParams(b, b)
        barLp.topMargin = (26 * density).toInt()
        col.addView(bar, barLp)

        val txt = TextView(this)
        txt.text = "加载中…"
        txt.setTextColor(Color.parseColor("#8A8F98"))
        txt.textSize = 12f
        val txtLp = LinearLayout.LayoutParams(-2, -2)
        txtLp.topMargin = (14 * density).toInt()
        col.addView(txt, txtLp)

        box.addView(col)
        rootLayout.addView(box)
        loading = box

        setContentView(rootLayout)

        val explicit = intent.getStringExtra(EXTRA_URL)?.takeIf { it.isNotBlank() }
        val url = explicit ?: runCatching { pickVideoUrl() }.getOrNull() ?: (SERVER_BASE + "/splash/" + FALLBACK_VIDEOS[0])

        vv.setOnPreparedListener { mp: MediaPlayer ->
            preparedAt = SystemClock.elapsedRealtime()
            runCatching { vv.start() }   // 不静音，原声播放
            ui.postDelayed({ hideLoading() }, 150)
            startStallWatchdog()
        }
        vv.setOnInfoListener { _, what, _ ->
            if (what == MediaPlayer.MEDIA_INFO_VIDEO_RENDERING_START ||
                what == MediaPlayer.MEDIA_INFO_BUFFERING_END
            ) {
                hideLoading()
            }
            false
        }
        vv.setOnCompletionListener { goNext() }
        vv.setOnErrorListener { _, _, _ ->
            goNext()
            true
        }

        runCatching { vv.setVideoURI(Uri.parse(url)) }.onFailure { goNext() }

        ui.postDelayed({ if (!finished && preparedAt == 0L) goNext() }, PREPARE_TIMEOUT_MS)
        ui.postDelayed({ goNext() }, HARD_TIMEOUT_MS)
    }

    /**
     * 从自己服务器取视频清单，随机挑一个；失败返回 null（调用方用兜底地址）。
     */
    private fun pickVideoUrl(): String? {
        return try {
            val conn = (java.net.URL(LIST_URL).openConnection() as java.net.HttpURLConnection).apply {
                connectTimeout = 1200
                readTimeout = 1200
                setRequestProperty("User-Agent", "okhttp/4.9.3")
            }
            if (conn.responseCode !in 200..299) {
                conn.disconnect()
                return null
            }
            val body = conn.inputStream.bufferedReader().use { it.readText() }
            conn.disconnect()
            val arr = org.json.JSONObject(body).optJSONArray("videos") ?: return null
            if (arr.length() == 0) return null
            val name = arr.getString((0 until arr.length()).random())
            SERVER_BASE + "/splash/" + name
        } catch (e: Exception) {
            null
        }
    }
    private fun hideLoading() {
        val box = loading ?: return
        if (box.visibility != View.VISIBLE) return
        runCatching {
            box.animate().alpha(0f).setDuration(180).withEndAction {
                runCatching { box.visibility = View.GONE }
            }.start()
        }
    }

    private fun startStallWatchdog() {
        val check = object : Runnable {
            override fun run() {
                if (finished) return
                val vv = videoView ?: return
                val pos = runCatching { vv.currentPosition }.getOrDefault(0)
                if (pos <= lastPos) {
                    goNext()
                    return
                }
                lastPos = pos
                ui.postDelayed(this, STALL_TIMEOUT_MS)
            }
        }
        ui.postDelayed(check, STALL_TIMEOUT_MS)
    }

    private fun goNext() {
        if (finished) return
        finished = true
        ui.removeCallbacksAndMessages(null)
        runCatching { videoView?.stopPlayback() }
        startActivity(Intent(this, ComposeLoginActivity::class.java))
        runCatching { overridePendingTransition(android.R.anim.fade_in, android.R.anim.fade_out) }
        finish()
    }

    override fun onDestroy() {
        super.onDestroy()
        ui.removeCallbacksAndMessages(null)
        runCatching { videoView?.stopPlayback() }
        videoView = null
        loading = null
    }
}
