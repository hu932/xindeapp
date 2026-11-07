package com.app.helper.ui.splash

import android.content.Intent
import android.graphics.Color
import android.media.MediaPlayer
import android.net.Uri
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.View
import android.view.WindowManager
import android.widget.VideoView
import androidx.appcompat.app.AppCompatActivity
import com.app.helper.ui.compose.ComposeLoginActivity
import java.io.File
import java.io.FileOutputStream
import java.net.HttpURLConnection
import java.net.URL

/**
 * 启动视频页：全屏播放后台下发的视频，播完自动进入 App。
 *
 * 设计原则（任何一步出错都不能卡住用户）：
 *   1. 先尝试下载到 cache（避免边下边播的卡顿），下载超过 6s 直接放弃并进入 App
 *   2. 播放失败 / 文件异常 → 立刻进入 App
 *   3. 播放器 15s 兜底超时 → 进入 App
 *   4. 只有「播放完成」才走正常跳转，保证"播完才进 App"
 *
 * 视频地址可通过 Intent extra [EXTRA_URL] 覆盖，默认用内置地址。
 */
class SplashVideoActivity : AppCompatActivity() {

    companion object {
        // 后台接口，直接返回 video/mp4
        const val DEFAULT_URL =
            "https://api.suyanw.cn/api/jhsp.php?msg=%E5%AF%B9%E8%84%B8%E8%87%AA%E6%8B%8D"
        const val EXTRA_URL = "splash_video_url"

        private const val DL_TIMEOUT_MS = 6000
        private const val PLAY_TIMEOUT_MS = 15000L
    }

    private val ui = Handler(Looper.getMainLooper())
    private var finished = false
    private var videoView: VideoView? = null

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

        // 兜底：无论如何 15s 后一定进 App
        ui.postDelayed({ goNext() }, PLAY_TIMEOUT_MS)

        // 后台下载到 cache，再本地播放（比边下边播稳）
        Thread {
            val local = runCatching { download(url) }.getOrNull()
            ui.post {
                if (finished) return@post
                if (local != null && local.length() > 0) {
                    playLocal(local)
                } else {
                    // 下载失败就直接播远程，再不行就跳过
                    playRemote(url)
                }
            }
        }.start()
    }

    private fun download(url: String): File? {
        val f = File(cacheDir, "splash.mp4")
        if (f.exists() && f.length() > 1024) return f
        var conn: HttpURLConnection? = null
        return try {
            conn = (URL(url).openConnection() as HttpURLConnection).apply {
                connectTimeout = DL_TIMEOUT_MS
                readTimeout = DL_TIMEOUT_MS
                instanceFollowRedirects = true
                setRequestProperty("User-Agent", "okhttp/4.9.3")
            }
            if (conn.responseCode !in 200..299) return null
            val tmp = File(cacheDir, "splash.tmp")
            conn.inputStream.use { input ->
                FileOutputStream(tmp).use { out -> input.copyTo(out, 64 * 1024) }
            }
            if (tmp.length() < 1024) {
                tmp.delete()
                return null
            }
            if (f.exists()) f.delete()
            tmp.renameTo(f)
            f
        } catch (e: Exception) {
            null
        } finally {
            conn?.disconnect()
        }
    }

    private fun playLocal(file: File) {
        val vv = videoView ?: return goNext()
        vv.setOnPreparedListener { mp: MediaPlayer ->
            mp.isLooping = false
            mp.setVolume(0f, 0f)   // 启动动画不打扰用户
            vv.start()
        }
        vv.setOnCompletionListener { goNext() }
        vv.setOnErrorListener { _, _, _ ->
            goNext()
            true
        }
        runCatching { vv.setVideoURI(Uri.fromFile(file)) }
            .onFailure { goNext() }
    }

    private fun playRemote(url: String) {
        val vv = videoView ?: return goNext()
        vv.visibility = View.VISIBLE
        vv.setOnPreparedListener { mp: MediaPlayer ->
            mp.setVolume(0f, 0f)
            vv.start()
        }
        vv.setOnCompletionListener { goNext() }
        vv.setOnErrorListener { _, _, _ ->
            goNext()
            true
        }
        runCatching { vv.setVideoURI(Uri.parse(url)) }
            .onFailure { goNext() }
    }

    /** 只跳一次，避免超时和播完同时触发重复启动 */
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
