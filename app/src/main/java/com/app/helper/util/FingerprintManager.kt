package com.app.helper.util

import android.content.Context
import android.os.Build
import android.provider.Settings
import android.util.Log
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.util.UUID

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
        try {
            val file = File(getFpDir(context), filename)
            file.writeText(data)
            Log.d(TAG, "Saved FP: $filename (${data.length} bytes)")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to save FP: $filename", e)
        }
    }

    fun loadFingerprintData(context: Context, filename: String): String? {
        return try {
            val file = File(getFpDir(context), filename)
            if (file.exists()) file.readText() else null
        } catch (e: Exception) {
            Log.e(TAG, "Failed to load FP: $filename", e)
            null
        }
    }

    fun setActiveFp(context: Context, filename: String) {
        context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .edit().putString(ACTIVE_FP_KEY, filename).apply()
    }

    fun getActiveFpName(context: Context): String? {
        return context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .getString(ACTIVE_FP_KEY, null)
    }

    fun loadCurrentFp(context: Context): String? {
        val activeName = getActiveFpName(context)
        if (activeName != null) {
            val data = loadFingerprintData(context, activeName)
            if (data != null) return data
        }
        val dir = getFpDir(context)
        val files = dir.listFiles()?.sortedByDescending { it.lastModified() }
        return files?.firstOrNull()?.let {
            setActiveFp(context, it.name)
            it.readText()
        }
    }

    fun listFpFiles(context: Context): List<FpFileInfo> {
        val dir = getFpDir(context)
        val activeName = getActiveFpName(context)
        return dir.listFiles()
            ?.filter { it.extension == "json" }
            ?.sortedByDescending { it.lastModified() }
            ?.map { FpFileInfo(it.name, it.length(), it.lastModified(), it.name == activeName) }
            ?: emptyList()
    }

    fun deleteFpFile(context: Context, filename: String): Boolean {
        return try {
            val result = File(getFpDir(context), filename).delete()
            if (getActiveFpName(context) == filename) {
                context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
                    .edit().remove(ACTIVE_FP_KEY).apply()
            }
            result
        } catch (e: Exception) { false }
    }

    fun generateFullFp(context: Context): JSONObject {
        val androidId = randomHex(16)
        val gsfAndroidId = randomHex(16)
        val advertiserId = UUID.randomUUID().toString()
        val now = System.currentTimeMillis()

        val f7Params = generateF7Params(androidId)

        return JSONObject().apply {
            put("name", "fp_${System.currentTimeMillis()}")
            put("androidId", androidId)
            put("gsfAndroidId", gsfAndroidId)
            put("advertiserId", advertiserId)

            put("vendingSourceDir", "/system/priv-app/Phonesky/PhoneSkySplit.apk")
            put("gmsSourceDir", "/system/priv-app/PrebuiltGmsCore/PrebuiltGmsCoreSplit.apk")
            put("shopeeSourceDir", "/data/app/~~${randomBase64(12)}==/com.shopee.tw-${randomBase64(12)}==/base.apk")
            put("apkSourceDir", "/data/app/~~${randomBase64(12)}==/com.shopee.tw-${randomBase64(12)}==/base.apk")

            put("keyECHashKey2", randomHex(64))
            put("keyECNewId", randomHex(32))

            val installOthers = now - (30L..180L).random() * 86400000L
            val updateOthers = installOthers + (1L..30L).random() * 86400000L
            val installShopee = now - (7L..90L).random() * 86400000L
            val updateShopee = installShopee + (1L..14L).random() * 86400000L
            put("randomInstallOthersTime", installOthers.toString())
            put("randomUpdateOthersTime", updateOthers.toString())
            put("randomInstallShopeeTime", installShopee.toString())
            put("randomUpdateShopeeTime", updateShopee.toString())

            val totalMem = ((4L..8L).random() * 1024L * 1024L * 1024L).toString()
            val totalSize = ((64L..256L).random() * 1024L * 1024L * 1024L).toString()
            val remainSize = (totalSize.toLong() / (2L..5L).random()).toString()
            put("totalMem", totalMem)
            put("totalSize", totalSize)
            put("remainSize", remainSize)

            put("fix7cTime", ((now / 1000) - (0..3600).random()).toString())
            put("screenOffTimeout", ((30000..300000).random()).toString())
            put("bootCount", ((10..500).random()).toString())
            put("fixAEAccessTime", ((now / 1000) - (0..86400).random()).toString())
            put("fixAEModifyTime", ((now / 1000) - (0..86400).random()).toString())
            put("m1INodeRandom", ((100000..999999).random()).toString())
            put("vbmetaDigest", randomHex(64))

            put("hotapps", generateHotApps().toString())
            put("f7_params", f7Params.toString())

            // SIM / network
            val simStates = listOf("LOADED,ABSENT", "LOADED,LOADED", "ABSENT,ABSENT")
            put("gsm.sim.state", simStates.random())
            put("gsm.network.type", "LTE,UNKNOWN")
            val operators = listOf("中華電信", "遠傳電信", "台灣大哥大", "台灣之星", "亞太電信")
            val op = operators.random()
            put("gsm.operator.alpha", "$op,")
            put("gsm.operator.iso-country", "tw,")

            // Fixed locale/timezone fields
            put("0x44", "zh-Hant-TW")
            put("0x50", "1")
            put("0x59", "Asia/Taipei")
            put("0x5d", "1")
            put("0x62", "14")
            put("0x67", "[]")
            put("0x68", "[]")
            put("0x6a", "0")
            put("0x6c", "0")
            put("0x6d", "[]")
            put("0xaa", "-1")

            // Mirror fields
            put("0xaf", androidId)
            put("0x9c", advertiserId)
            put("0x139", gsfAndroidId)
            put("0xc3", optString("bootCount"))
            put("0x2015", optString("vbmetaDigest"))
            put("0x146", optString("keyECHashKey2"))
            put("0x13a", optString("m1INodeRandom"))
            put("0x133", optString("shopeeSourceDir"))
            put("0x105", optString("hotapps"))
            put("0x75", f7Params.toString())

            put("0x1b", optString("gsm.sim.state"))
            put("0x2e", optString("gsm.network.type"))
            put("0x3a", optString("gsm.operator.alpha"))

            put("0xd3", JSONObject().apply {
                put("system_locales", "zh-Hant-TW")
                put("font_scale", "1.0")
            }.toString())

            // Timestamps for various hex keys
            put("0x8", (now - (0..86400000).random()).toString())
            put("0x91", JSONObject().apply {
                put("elapsedRealtime", ((1..72) * 3600000).toString())
                put("uptimeMillis", ((1..72) * 3600000).toString())
            }.toString())
            put("0xa9", (now - (0..3600000).random()).toString())
            put("0xdc", (now - (0..86400000).random()).toString())
            put("0x137", optString("randomInstallShopeeTime"))
            put("0x138", optString("randomUpdateShopeeTime"))

            // SIM operator
            put("0x41", "46692")
            put("0x40", randomHex(20))
            put("0x42", randomHex(10))
            put("0x43", "")

            // Other computed fields
            put("0x4", randomHex(16))
            put("0x5", randomHex(16))
            put("0x39", "")
            put("0x63", ((1080..1440).random()).toString())
            put("0x64", ((2200..2960).random()).toString())
            put("0x65", ((320..480).random()).toString())
            put("0x66", ((2..4).random()).toString())
            put("0xa4", randomHex(32))
            put("0xac", randomHex(16))
            put("0xb3", randomHex(32))
            put("0xda", randomHex(8))
            put("0xdb", randomHex(8))
            put("0x102", randomHex(16))
            put("0x106", randomHex(32))
            put("0x10a", randomHex(16))
            put("0x119", "")
            put("0x126", "wifi")
            put("0x128", "")
            put("0x13b", ((100000000L..999999999L).random()).toString())
            put("0x13c", ((100000000L..999999999L).random()).toString())
            put("0x13d", ((100000000L..999999999L).random()).toString())
            put("0x152", randomHex(32))
            put("0x153", randomHex(16))
            put("0x154", randomHex(16))
        }
    }

    private fun generateF7Params(androidId: String): JSONObject {
        return JSONObject().apply {
            put("serialHexPool", JSONArray().apply {
                repeat(6) { put(randomHex(40)) }
            })
            put("snHexPool", JSONArray().apply {
                repeat(6) { put(randomHex(40)) }
            })
            put("snDecPool", JSONArray().apply {
                repeat(6) { put(randomDecimal(22)) }
            })
            put("c1TimeOffsetDays", (-30..30).random())
            put("c2TimeOffsetDays", (-30..30).random())
            put("leafOffsetDays", (1..30).random())
            put("leafGapMs", (10000..60000).random())
            put("vbh2", randomHex(64))
            put("dl", "1")
            put("vbs", "0")
        }
    }

    private fun generateHotApps(): JSONArray {
        val apps = listOf(
            "com.facebook.katana", "com.instagram.android", "com.whatsapp",
            "com.google.android.youtube", "com.google.android.gm",
            "com.google.android.apps.maps", "jp.naver.line.android",
            "com.facebook.orca", "com.twitter.android"
        )
        return JSONArray().apply {
            apps.shuffled().take((3..6).random()).forEach { put(it) }
        }
    }

    fun randomHex(length: Int): String {
        val chars = "0123456789abcdef"
        return (1..length).map { chars.random() }.joinToString("")
    }

    private fun randomDecimal(length: Int): String {
        val chars = "0123456789"
        return (1..length).map { chars.random() }.joinToString("")
    }

    private fun randomBase64(length: Int): String {
        val chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
        return (1..length).map { chars.random() }.joinToString("")
    }

    data class FpFileInfo(
        val name: String,
        val size: Long,
        val lastModified: Long,
        val isActive: Boolean
    )
}
