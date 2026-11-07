package com.app.helper.ui.splash

import android.app.Activity
import android.content.Intent
import android.graphics.Color
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.Gravity
import android.view.View
import android.view.WindowManager
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.AspectRatioFrameLayout
import androidx.media3.ui.PlayerView
import com.app.helper.R
import com.app.helper.ui.compose.ComposeLoginActivity

/**
 * 启动页：播放自有服务器上的视频。
 *
 * 用 ExoPlayer 而不是系统 VideoView —— 后者在部分机型/封装上会出现
 * 「有声音、画面全黑」（MediaPlayer 视频解码后无法上屏）。
 * 这里还加了失败自动换片（最多 3 条）与三重超时兜底。
 */
class SplashVideoActivity : Activity() {

    companion object {
        const val SERVER_BASE = "http://38.175.194.43"
        const val LIST_URL = SERVER_BASE + "/splash/list.json"
        const val EXTRA_URL = "splash_video_url"

        val FALLBACK_VIDEOS = listOf(
            "splash_01.mp4", "splash_02.mp4", "splash_03.mp4",
            "splash_04.mp4", "splash_05.mp4", "splash_06.mp4",
        )

        private const val PREPARE_TIMEOUT_MS = 4000L
        private const val STALL_TIMEOUT_MS = 4000L
        private const val HARD_TIMEOUT_MS = 30000L
        private const val MAX_TRIES = 3
    }

    private val ui = Handler(Looper.getMainLooper())
    private var finished = false
    private var player: ExoPlayer? = null
    private var playerView: PlayerView? = null
    private var loading: FrameLayout? = null
    private var candidates: List<String> = emptyList()
    private var tryIndex = 0
    private var renderingStarted = false
    private var lastPos = -1L

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        window.statusBarColor = Color.BLACK
        window.navigationBarColor = Color.BLACK

        val density = resources.displayMetrics.density
        val root = FrameLayout(this)
        root.setBackgroundColor(Color.BLACK)

        val pv = PlayerView(this)
        pv.useController = false
        pv.resizeMode = AspectRatioFrameLayout.RESIZE_MODE_ZOOM
        pv.setShutterBackgroundColor(Color.BLACK)
        root.addView(pv, FrameLayout.LayoutParams(-1, -1))
        playerView = pv

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
        root.addView(box)
        loading = box

        setContentView(root)

        val explicit = intent.getStringExtra(EXTRA_URL)?.takeIf { it.isNotBlank() }
        if (explicit != null) {
            candidates = listOf(explicit)
            startPlay()
        } else {
            Thread {
                val list = fetchList()
                ui.post {
                    candidates = list
                    startPlay()
                }
            }.start()
        }

        ui.postDelayed({ goNext() }, HARD_TIMEOUT_MS)
    }

    /** 从 /splash/list.json 取清单并打乱；失败用内置兜底清单 */
    private fun fetchList(): List<String> {
        val fromServer = runCatching {
            val conn = (java.net.URL(LIST_URL).openConnection() as java.net.HttpURLConnection).apply {
                connectTimeout = 1500
                readTimeout = 1500
                setRequestProperty("User-Agent", "okhttp/4.9.3")
            }
            val code = conn.responseCode
            val body = if (code in 200..299) conn.inputStream.bufferedReader().use { it.readText() } else ""
            conn.disconnect()
            val arr = if (body.isBlank()) null else org.json.JSONObject(body).optJSONArray("videos")
            if (arr == null || arr.length() == 0) emptyList()
            else (0 until arr.length()).map { i -> SERVER_BASE + "/splash/" + arr.getString(i) }
        }.getOrDefault(emptyList())

        return if (fromServer.isNotEmpty()) fromServer.shuffled()
        else FALLBACK_VIDEOS.map { SERVER_BASE + "/splash/" + it }.shuffled()
    }

    private fun startPlay() {
        if (finished) return
        if (candidates.isEmpty() || tryIndex >= candidates.size || tryIndex >= MAX_TRIES) {
            goNext()
            return
        }
        renderingStarted = false
        lastPos = -1L

        val p = ExoPlayer.Builder(this).build()
        player = p
        playerView?.player = p
        p.setMediaItem(MediaItem.fromUri(candidates[tryIndex]))
        p.prepare()
        p.playWhenReady = true
        p.volume = 1f

        p.addListener(object : Player.Listener {
            override fun onRenderedFirstFrame() {
                renderingStarted = true
                hideLoading()
            }

            override fun onPlaybackStateChanged(state: Int) {
                when (state) {
                    Player.STATE_READY -> {
                        hideLoading()
                        startStallWatchdog()
                    }
                    Player.STATE_ENDED -> goNext()
                }
            }

            override fun onPlayerError(error: PlaybackException) {
                tryNext()
            }
        })

        ui.postDelayed({ if (!finished && !renderingStarted) tryNext() }, PREPARE_TIMEOUT_MS)
    }

    private fun tryNext() {
        if (finished) return
        tryIndex++
        runCatching {
            player?.stop()
            player?.release()
        }
        player = null
        if (tryIndex >= MAX_TRIES || tryIndex >= candidates.size) goNext() else startPlay()
    }

    private fun startStallWatchdog() {
        val check = object : Runnable {
            override fun run() {
                if (finished) return
                val p = player ?: return
                val pos = p.currentPosition
                if (pos <= lastPos) {
                    tryNext()
                    return
                }
                lastPos = pos
                ui.postDelayed(this, STALL_TIMEOUT_MS)
            }
        }
        ui.postDelayed(check, STALL_TIMEOUT_MS)
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

    private fun goNext() {
        if (finished) return
        finished = true
        ui.removeCallbacksAndMessages(null)
        runCatching {
            player?.stop()
            player?.release()
        }
        player = null
        startActivity(Intent(this, ComposeLoginActivity::class.java))
        runCatching { overridePendingTransition(android.R.anim.fade_in, android.R.anim.fade_out) }
        finish()
    }

    override fun onDestroy() {
        super.onDestroy()
        ui.removeCallbacksAndMessages(null)
        runCatching {
            player?.stop()
            player?.release()
        }
        player = null
        playerView = null
        loading = null
    }
}
