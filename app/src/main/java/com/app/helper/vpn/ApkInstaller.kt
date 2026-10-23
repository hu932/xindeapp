package com.app.helper.vpn

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings
import android.util.Log
import androidx.core.content.FileProvider
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.net.HttpURLConnection
import java.net.URL

/**
 * 应用内下载并安装 v2rayNG。
 *
 * Android 不允许真正的静默安装（除非系统签名或 root），所以流程是：
 *   1. 从 GitHub Releases 下载对应 ABI 的 APK 到 cache
 *   2. 用 FileProvider 生成 content:// URI
 *   3. 拉起系统安装器，用户在弹窗里点「安装」
 *
 * 关键前置：需要在设置里允许本应用「安装未知应用」。
 * 没有这个权限时系统会弹一次授权页，之后才继续。
 */
object ApkInstaller {

    private const val TAG = "ApkInstaller"

    /** 固定版本，避免每次最新版变更导致文件名对不上。 */
    private const val VERSION = "2.3.11"

    /** 按 ABI 选择下载文件。 */
    private fun assetName(): String = when {
        Build.SUPPORTED_ABIS.any { it.contains("arm64") } ->
            "v2rayNG_${VERSION}-fdroid_arm64-v8a.apk"
        Build.SUPPORTED_ABIS.any { it.contains("armeabi") } ->
            "v2rayNG_${VERSION}-fdroid_armeabi-v7a.apk"
        else ->
            "v2rayNG_${VERSION}-fdroid_universal.apk"
    }

    private fun downloadUrl(): String =
        "https://github.com/2dust/v2rayNG/releases/download/${VERSION}/" + assetName()

    /** 下载好的 APK 路径。 */
    private fun apkFile(ctx: Context): File =
        File(File(ctx.cacheDir, "apk"), assetName())

    /** 已下载且大小合理。 */
    fun isDownloaded(ctx: Context): Boolean {
        val f = apkFile(ctx)
        return f.exists() && f.length() > 5 * 1024 * 1024
    }

    /**
     * 下载 APK。
     * @param onProgress 0..100，下载过程中回调（已切到主线程由调用方处理）
     */
    suspend fun download(ctx: Context, onProgress: (Int) -> Unit = {}): File? =
        withContext(Dispatchers.IO) {
            val out = apkFile(ctx)
            if (isDownloaded(ctx)) return@withContext out
            out.parentFile?.mkdirs()
            runCatching {
                val conn = URL(downloadUrl()).openConnection() as HttpURLConnection
                conn.connectTimeout = 20000
                conn.readTimeout = 60000
                conn.instanceFollowRedirects = true
                conn.setRequestProperty("User-Agent", "Mozilla/5.0")
                conn.connect()
                val total = conn.contentLengthLong
                if (conn.responseCode != 200) {
                    Log.w(TAG, "download http ${conn.responseCode}")
                    return@withContext null
                }
                val tmp = File(out.parentFile, out.name + ".part")
                conn.inputStream.use { input ->
                    tmp.outputStream().use { output ->
                        val buf = ByteArray(64 * 1024)
                        var read: Int
                        var done = 0L
                        var lastPct = -1
                        while (true) {
                            read = input.read(buf)
                            if (read <= 0) break
                            output.write(buf, 0, read)
                            done += read
                            if (total > 0) {
                                val pct = (done * 100 / total).toInt()
                                if (pct != lastPct) {
                                    lastPct = pct
                                    onProgress(pct)
                                }
                            }
                        }
                    }
                }
                conn.disconnect()
                if (tmp.length() < 5 * 1024 * 1024) {
                    Log.w(TAG, "downloaded file too small: ${tmp.length()}")
                    tmp.delete()
                    return@withContext null
                }
                if (out.exists()) out.delete()
                tmp.renameTo(out)
                Log.i(TAG, "downloaded ${out.length()} bytes")
                out
            }.onFailure { Log.w(TAG, "download failed: $it") }.getOrNull()
        }

    /** 是否已允许安装未知来源应用。 */
    fun canInstall(ctx: Context): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            ctx.packageManager.canRequestPackageInstalls()
        } else {
            true
        }
    }

    /** 跳到「安装未知应用」授权页。 */
    fun requestInstallPermission(ctx: Context) {
        runCatching {
            val i = Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES).apply {
                data = Uri.parse("package:${ctx.packageName}")
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            ctx.startActivity(i)
        }.onFailure { Log.w(TAG, "open install permission failed: $it") }
    }

    /**
     * 拉起系统安装器。
     * 返回 false 表示文件缺失或安装器拉不起来。
     */
    fun install(ctx: Context): Boolean {
        val f = apkFile(ctx)
        if (!f.exists() || f.length() < 5 * 1024 * 1024) {
            Log.w(TAG, "apk not ready")
            return false
        }
        return runCatching {
            val uri: Uri = FileProvider.getUriForFile(
                ctx, "${ctx.packageName}.fileprovider", f
            )
            val i = Intent(Intent.ACTION_VIEW).apply {
                setDataAndType(uri, "application/vnd.android.package-archive")
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            ctx.startActivity(i)
            Log.i(TAG, "installer launched")
            true
        }.onFailure { Log.w(TAG, "install failed: $it") }.getOrDefault(false)
    }

    /** 从 GitHub 拉最新版本号（可选，用于提示更新）。 */
    suspend fun latestVersion(): String? = withContext(Dispatchers.IO) {
        runCatching {
            val conn = URL("https://api.github.com/repos/2dust/v2rayNG/releases/latest")
                .openConnection() as HttpURLConnection
            conn.connectTimeout = 15000
            conn.readTimeout = 15000
            conn.setRequestProperty("User-Agent", "Mozilla/5.0")
            val body = conn.inputStream.bufferedReader().use { it.readText() }
            conn.disconnect()
            Regex("\"tag_name\"\\s*:\\s*\"([^\"]+)\"").find(body)?.groupValues?.get(1)
        }.getOrNull()
    }
}
