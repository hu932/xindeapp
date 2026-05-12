package com.app.helper.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.PixelFormat
import android.hardware.display.DisplayManager
import android.hardware.display.VirtualDisplay
import android.media.ImageReader
import android.media.projection.MediaProjection
import android.media.projection.MediaProjectionManager
import android.os.Build
import android.os.IBinder
import android.provider.Settings
import android.util.Log
import com.app.helper.network.ApiService
import com.app.helper.util.TokenManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.io.ByteArrayOutputStream

class ScreenShareService : Service() {

    companion object {
        private const val TAG = "ScreenShareService"
        private const val CHANNEL_ID = "screen_share_channel"
        private const val NOTIFICATION_ID = 2001
        const val ACTION_START = "com.app.helper.screen.START"
        const val ACTION_STOP = "com.app.helper.screen.STOP"
        const val EXTRA_RESULT_CODE = "result_code"
        const val EXTRA_RESULT_DATA = "result_data"
        var isSharing = false
            private set
    }

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var projection: MediaProjection? = null
    private var reader: ImageReader? = null
    private var display: VirtualDisplay? = null
    private var uploadJob: Job? = null
    private var width = 360
    private var height = 640
    private var density = 1

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createChannel()
        startForeground(NOTIFICATION_ID, createNotification())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> stopSelf()
            ACTION_START -> {
                val resultCode = intent.getIntExtra(EXTRA_RESULT_CODE, 0)
                val data = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    intent.getParcelableExtra(EXTRA_RESULT_DATA, Intent::class.java)
                } else {
                    @Suppress("DEPRECATION")
                    intent.getParcelableExtra(EXTRA_RESULT_DATA)
                }
                if (resultCode != 0 && data != null) startProjection(resultCode, data)
            }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        isSharing = false
        uploadJob?.cancel()
        display?.release()
        reader?.close()
        projection?.stop()
        scope.cancel()
    }

    private fun startProjection(resultCode: Int, data: Intent) {
        if (isSharing) return
        val metrics = resources.displayMetrics
        density = metrics.densityDpi
        val scale = 0.45f
        width = (metrics.widthPixels * scale).toInt().coerceAtLeast(240)
        height = (metrics.heightPixels * scale).toInt().coerceAtLeast(360)
        reader = ImageReader.newInstance(width, height, PixelFormat.RGBA_8888, 2)
        val manager = getSystemService(Context.MEDIA_PROJECTION_SERVICE) as MediaProjectionManager
        projection = manager.getMediaProjection(resultCode, data)
        projection?.registerCallback(object : MediaProjection.Callback() {
            override fun onStop() {
                stopSelf()
            }
        }, null)
        display = projection?.createVirtualDisplay(
            "GejieScreenShare",
            width,
            height,
            density,
            DisplayManager.VIRTUAL_DISPLAY_FLAG_AUTO_MIRROR,
            reader?.surface,
            null,
            null
        )
        isSharing = true
        startUploadLoop()
    }

    private fun startUploadLoop() {
        uploadJob?.cancel()
        uploadJob = scope.launch {
            while (isActive) {
                try {
                    captureJpeg()?.let { jpeg ->
                        ApiService.uploadScreenFrame(
                            username = TokenManager.getUsername(),
                            groupId = TokenManager.getGroupId(),
                            apiType = TokenManager.getApiType(),
                            deviceId = getAppDeviceId(),
                            jpeg = jpeg
                        )
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "upload screen frame failed", e)
                }
                delay(260L)
            }
        }
    }

    private fun captureJpeg(): ByteArray? {
        val image = reader?.acquireLatestImage() ?: return null
        return image.use {
            val plane = it.planes.firstOrNull() ?: return null
            val buffer = plane.buffer
            val pixelStride = plane.pixelStride
            val rowStride = plane.rowStride
            val rowPadding = rowStride - pixelStride * width
            val bitmapWidth = width + rowPadding / pixelStride
            val bitmap = Bitmap.createBitmap(bitmapWidth, height, Bitmap.Config.ARGB_8888)
            bitmap.copyPixelsFromBuffer(buffer)
            val cropped = Bitmap.createBitmap(bitmap, 0, 0, width, height)
            bitmap.recycle()
            val out = ByteArrayOutputStream()
            cropped.compress(Bitmap.CompressFormat.JPEG, 55, out)
            cropped.recycle()
            out.toByteArray()
        }
    }

    private fun getAppDeviceId(): String {
        val androidId = Settings.Secure.getString(contentResolver, Settings.Secure.ANDROID_ID)
        return androidId?.takeIf { it.isNotBlank() } ?: "android-${Build.MODEL}"
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(CHANNEL_ID, "屏幕共享服务", NotificationManager.IMPORTANCE_LOW)
            (getSystemService(NOTIFICATION_SERVICE) as NotificationManager).createNotificationChannel(channel)
        }
    }

    private fun createNotification(): Notification {
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) Notification.Builder(this, CHANNEL_ID)
        else @Suppress("DEPRECATION") Notification.Builder(this)
        return builder
            .setContentTitle("格界13-14 正在共享屏幕")
            .setContentText("后台可查看实时画面，点开应用可停止。")
            .setSmallIcon(android.R.drawable.ic_menu_camera)
            .setOngoing(true)
            .build()
    }
}
