package com.app.helper.ui.splash

import android.content.Intent
import android.graphics.Color
import android.media.MediaPlayer
import android.net.Uri
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.view.WindowManager
import android.widget.VideoView
import android.app.Activity
import com.app.helper.ui.compose.ComposeLoginActivity

/**
 * 启动页：在启动界面内直接流式播放后台下发的视频，播完再进 App。
 *
 * 按需求调整：
 *   - 直接播远程流（setVideoURI(远程地址)），不下载、不做本地缓存
 *   - 有声音（不静音）
 *   - 视频播完才进入 App；播放失败 / prepare 超时则直接进 App，避免卡死
 *
 * 视频地址可通过 Intent extra [EXTRA_URL] 覆盖。
 */
class SplashVideoActivity : Activity() {

    companion object {
        // 后台接口，直接返回 video/mp4
        const val DEFAULT_URL =
            "https://api.suyanw.cn/api/jhsp.php?msg=%E5%AF%B9%E8%84%B8%E8%87%AA%E6%8B%8D"
        const val EXTRA_URL = "splash_video_url"

        // prepare 超时：网络差时不能一直黑屏等
        private const val PREPARE_TIMEOUT_MS = 12000L
        // 总兜底：无论如何不超过这个时间（含播放）
        private const val HARD_TIMEOUT_MS = 40000L
    }

    private val ui = Handler(Looper.getMainLooper())
    private var finished = false
    private var videoView: VideoView? = null
    private var preparedAt = 0L
    private var durationMs = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        window.statusBarColor = Color.BLACK
        window.navigationBarColor = Color.BLACK

        val vv = VideoView(this)
        vv.setBackgroundColor(Color.BLACK)
        vv.layoutParams = android.view.ViewGroup.LayoutParams(-1, -1)
        setContentView(vv)
        videoView = vv

        val url = intent.getStringExtra(EXTRA_URL)?.takeIf { it.isNotBlank() } ?: DEFAULT_URL

        vv.setOnPreparedListener { mp: MediaPlayer ->
            preparedAt = SystemClock.elapsedRealtime()
            durationMs = runCatching { mp.duration }.getOrDefault(0)
            // 不静音：保持原声播放
            runCatching { vv.start() }
        }
        vv.setOnCompletionListener { goNext() }
        vv.setOnErrorListener { _, _, _ ->
            goNext()
            true
        }

        // 直接流式播放
        runCatching { vv.setVideoURI(Uri.parse(url)) }
            .onFailure { goNext() }

        // prepare 超时：还没开始播就放弃
        ui.postDelayed({
            if (!finished && preparedAt == 0L) goNext()
        }, PREPARE_TIMEOUT_MS)

        // 硬兜底：防止某些机型 completion 不回调
        ui.postDelayed({ goNext() }, HARD_TIMEOUT_MS)
    }

    /** 只跳一次，避免超时与播完同时触发重复启动 */
    private fun goNext() {
        if (finished) return
        finished = true
        runCatching { videoView?.stopPlayback() }
        startActivity(Intent(this, ComposeLoginActivity::class.java))
        overridePendingTransition(android.R.anim.fade_in, android.R.anim.fade_out)
        finish()
    }

    override fun onDestroy() {
        super.onDestroy()
        ui.removeCallbacksAndMessages(null)
        runCatching { videoView?.stopPlayback() }
        videoView = null
    }
}
