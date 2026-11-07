package com.app.helper.util

import android.annotation.SuppressLint
import android.content.Context
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
        // 原子写：先写 .tmp 再 rename。
        // 原来直接 writeText，进程被杀/闪退时会留下半截文件，
        // 之后 loadCurrentFp 读到坏 JSON → 指纹"获取不到"或用了坏数据。
        try {
            val dir = getFpDir(context)
            val target = File(dir, filename)
            val tmp = File(dir, "$filename.tmp")
            tmp.writeText(data)
            if (!tmp.renameTo(target)) {
                target.delete()
                tmp.copyTo(target, overwrite = true)
                tmp.delete()
            }
        } catch (e: Exception) {
            Log.e(TAG, "Save FP failed: $filename", e)
        }
    }

    fun loadFingerprintData(context: Context, filename: String): String? {
        return try {
            val file = File(getFpDir(context), filename)
            if (file.exists() && file.length() > 0) file.readText() else null
        } catch (_: Exception) {
            null
        }
    }

    fun setActiveFp(context: Context, filename: String) {
        // 只保留一个权威 key（fp_prefs/active_fp），legacy 一起写是为了向下兼容旧版本，
        // 但读取永远以 fp_prefs 为准，避免两个 key 不同步导致"界面显示 A、实际用 B"。
        context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .edit()
            .putString(ACTIVE_FP_KEY, filename)
            .apply()
        context.getSharedPreferences("FP", Context.MODE_PRIVATE)
            .edit()
            .putString("useFp", filename)
            .apply()
    }

    /**
     * 清理失效指针：active_fp 指向的文件不存在时，把这个指针删掉，
     * 否则会一直回落到"目录里最新的那个文件"，出现"用的不是我选的那份"。
     */
    private fun healStalePointer(context: Context, activeName: String?): String? {
        if (activeName.isNullOrEmpty()) return null
        val f = File(getFpDir(context), activeName)
        if (f.exists() && f.length() > 0) return activeName
        Log.w(TAG, "active fp missing, clearing pointer: $activeName")
        context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .edit()
            .remove(ACTIVE_FP_KEY)
            .apply()
        context.getSharedPreferences("FP", Context.MODE_PRIVATE)
            .edit()
            .remove("useFp")
            .apply()
        return null
    }

    fun getActiveFpName(context: Context): String? {
    fun getActiveFpName(context: Context): String? {
        // 以 fp_prefs 为权威，legacy 只做兜底，避免两个 key 打架
        val primary = context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .getString(ACTIVE_FP_KEY, null)
        if (!primary.isNullOrEmpty()) return primary
        return context.getSharedPreferences("FP", Context.MODE_PRIVATE)
            .getString("useFp", null)
    }

    fun loadCurrentFp(context: Context): String? {
        // 1) 先按指针读；指针失效就清掉（healStalePointer），不再静默用错文件
        val activeName = healStalePointer(context, getActiveFpName(context))
        if (activeName != null) {
            val data = loadFingerprintData(context, activeName)
            if (data != null) return data
        }
        // 2) 回落到目录里最新的有效文件，并回写指针
        return getFpDir(context).listFiles()
            ?.filter { it.isFile && it.length() > 0 && !it.name.endsWith(".tmp") }
            ?.sortedByDescending { it.lastModified() }
            ?.firstOrNull()
            ?.let {
                setActiveFp(context, it.name)
                loadFingerprintData(context, it.name)
            }
    }

    fun listFpFiles(context: Context): List<FpFileInfo> {
        val activeName = getActiveFpName(context)
        return getFpDir(context).listFiles()
            ?.filter { it.isFile }
            ?.sortedByDescending { it.lastModified() }
            ?.map { FpFileInfo(it.name, it.length(), it.lastModified(), it.name == activeName) }
            ?: emptyList()
    }

    fun deleteFpFile(context: Context, filename: String): Boolean {
        val result = try {
            File(getFpDir(context), filename).delete()
        } catch (_: Exception) {
            false
        }
        if (getActiveFpName(context) == filename) {
            context.getSharedPreferences("FP", Context.MODE_PRIVATE)
                .edit()
                .remove("useFp")
                .apply()
            context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
                .edit()
                .remove(ACTIVE_FP_KEY)
                .apply()
        }
        return result
    }

    @SuppressLint("HardwareIds", "PackageManagerGetSignatures")
    fun generateFullFp(context: Context): JSONObject {
        v.f.c(context)
        v.f.a()

        val json = JSONObject()
        for ((key, value) in v.f.a) {
            json.put(key, value)
        }
        json.put(
            "name",
            "FP-" + java.time.LocalDateTime.now().format(
                java.time.format.DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
            )
        )
        return json
    }

    data class FpFileInfo(
        val name: String,
        val size: Long,
        val lastModified: Long,
        val isActive: Boolean
    )
}
