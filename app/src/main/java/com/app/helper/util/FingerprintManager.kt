package com.app.helper.util

import android.annotation.SuppressLint
import android.content.Context
import android.os.Environment
import android.util.Log
import org.json.JSONObject
import java.io.File

object FingerprintManager {
    private const val TAG = "FingerprintManager"
    private const val FP_DIR = "fps"
    private const val ACTIVE_FP_KEY = "active_fp"

    private fun getFpDir(context: Context): File {
        val dir = File(context.filesDir, FP_DIR)
        if (!dir.exists()) dir.mkdirs()
        return dir
    }

    fun saveFingerprintData(context: Context, filename: String, data: String) {
        try { File(getFpDir(context), filename).writeText(data) }
        catch (e: Exception) { Log.e(TAG, "Save FP failed: $filename", e) }
    }

    fun loadFingerprintData(context: Context, filename: String): String? {
        return try { val f = File(getFpDir(context), filename); if (f.exists()) f.readText() else null }
        catch (_: Exception) { null }
    }

    fun setActiveFp(context: Context, filename: String) {
        context.getSharedPreferences("FP", Context.MODE_PRIVATE).edit().putString("useFp", filename).apply()
        context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE).edit().putString(ACTIVE_FP_KEY, filename).apply()
    }

    fun getActiveFpName(context: Context): String? {
        val n = context.getSharedPreferences("FP", Context.MODE_PRIVATE).getString("useFp", null)
        if (!n.isNullOrEmpty()) return n
        return context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE).getString(ACTIVE_FP_KEY, null)
    }

    fun loadCurrentFp(context: Context): String? {
        val activeName = getActiveFpName(context)
        if (activeName != null) { val d = loadFingerprintData(context, activeName); if (d != null) return d }
        return getFpDir(context).listFiles()?.sortedByDescending { it.lastModified() }?.firstOrNull()?.let {
            setActiveFp(context, it.name); it.readText()
        }
    }

    fun listFpFiles(context: Context): List<FpFileInfo> {
        val dir = getFpDir(context); val activeName = getActiveFpName(context)
        return dir.listFiles()?.sortedByDescending { it.lastModified() }
            ?.map { FpFileInfo(it.name, it.length(), it.lastModified(), it.name == activeName) } ?: emptyList()
    }

    fun deleteFpFile(context: Context, filename: String): Boolean {
        val result = try { File(getFpDir(context), filename).delete() } catch (_: Exception) { false }
        if (getActiveFpName(context) == filename) {
            context.getSharedPreferences("FP", Context.MODE_PRIVATE).edit().remove("useFp").apply()
            context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE).edit().remove(ACTIVE_FP_KEY).apply()
        }
        return result
    }

    @SuppressLint("HardwareIds", "PackageManagerGetSignatures")
    fun generateFullFp(context: Context): JSONObject {
        v.f.c(context)
        v.f.a()

        val json = JSONObject()
        for ((k, value) in v.f.a) {
            json.put(k, value)
        }
        json.put("name", "FP-" + java.time.LocalDateTime.now().format(
            java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")))
        return json
    }

    fun randomHex(len: Int): String {
        val chars = "0123456789abcdef"
        return (1..len).map { chars.random() }.joinToString("")
    }

    fun dumpFpToDownloads(context: Context) {
        try {
            val fp = loadCurrentFp(context) ?: return
            val file = File(Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS), "fp_debug_${System.currentTimeMillis()}.json")
            file.writeText(fp); Log.i(TAG, "FP dumped: ${file.absolutePath}")
        } catch (e: Exception) { Log.e(TAG, "Dump failed", e) }
    }

    data class FpFileInfo(val name: String, val size: Long, val lastModified: Long, val isActive: Boolean)
}
