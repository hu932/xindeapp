package com.app.helper.util

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.util.Log
import com.app.helper.network.ApiConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

/**
 * 用户头像加载。
 *
 * 服务端接口：GET {base}/api/avatar?accountId=xxx 直接返回图片二进制。
 *
 * 不引 Coil / Glide：只是为了一个 34dp 的圆形头像，加图片库会带进
 * 一堆传递依赖和 APK 体积，用 HttpURLConnection + BitmapFactory 足够。
 *
 * 缓存策略：解码后的字节落盘到 cacheDir，网络失败时用缓存兜底，
 * 这样断网或服务器慢的时候头像依然在，不会每次都闪成占位图标。
 */
object AvatarLoader {

    private const val TAG = "AvatarLoader"

    /** 网络超时。头像不重要，等太久会拖慢启动。 */
    private const val CONNECT_TIMEOUT = 8000
    private const val READ_TIMEOUT = 12000

    /** 头像按账号 id 缓存，文件名要清洗掉路径分隔符。 */
    private fun cacheFile(ctx: Context, accountId: String): File {
        val safe = accountId.replace(Regex("[^A-Za-z0-9_\\-]"), "_").take(64)
        return File(File(ctx.cacheDir, "avatar"), "a_$safe.img")
    }

    /**
     * 拉取并解码头像。
     *
     * @param force true 时忽略缓存直接走网络（用户刚在后台上传完用得上）
     * @return 解码成功的 Bitmap，没有头像或全部失败时返回 null
     */
    suspend fun load(ctx: Context, accountId: String, force: Boolean = false): Bitmap? =
        withContext(Dispatchers.IO) {
            if (accountId.isBlank()) return@withContext null
            val cache = cacheFile(ctx, accountId)

            // 1. 有缓存且不强制刷新，先用缓存
            if (!force && cache.exists() && cache.length() > 0) {
                decode(cache.readBytes())?.let {
                    Log.d(TAG, "cache hit for $accountId")
                    return@withContext it
                }
                // 缓存坏了，删掉继续走网络
                cache.delete()
            }

            // 2. 走网络
            val bytes = download(accountId)
            if (bytes == null) {
                // 网络失败，退一步用缓存（可能是上次的旧头像）
                if (cache.exists() && cache.length() > 0) {
                    Log.w(TAG, "network failed, falling back to cache for $accountId")
                    return@withContext decode(cache.readBytes())
                }
                return@withContext null
            }

            // 3. 404 之类返回的是空或非图片，decode 会失败，此时清缓存
            val bmp = decode(bytes)
            if (bmp == null) {
                cache.delete()
                return@withContext null
            }

            // 4. 落盘
            runCatching {
                cache.parentFile?.mkdirs()
                cache.writeBytes(bytes)
            }
            bmp
        }

    private fun download(accountId: String): ByteArray? = runCatching {
        val url = ApiConfig.GO_BASE_URL + "/api/avatar?accountId=" +
                URLEncoder.encode(accountId, "UTF-8")
        val conn = URL(url).openConnection() as HttpURLConnection
        conn.connectTimeout = CONNECT_TIMEOUT
        conn.readTimeout = READ_TIMEOUT
        conn.requestMethod = "GET"
        conn.instanceFollowRedirects = true
        try {
            if (conn.responseCode != 200) {
                Log.d(TAG, "avatar http ${conn.responseCode} for $accountId")
                return null
            }
            conn.inputStream.use { it.readBytes() }
        } finally {
            conn.disconnect()
        }
    }.onFailure { Log.w(TAG, "download failed: $it") }.getOrNull()

    private fun decode(bytes: ByteArray): Bitmap? = runCatching {
        if (bytes.isEmpty()) return null
        BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
    }.getOrNull()

    /** 退出登录时清掉本机缓存，避免换账号后看到上一个人的头像。 */
    fun clearAll(ctx: Context) {
        runCatching {
            File(ctx.cacheDir, "avatar").listFiles()?.forEach { it.delete() }
        }
    }
}
