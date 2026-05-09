package com.app.helper.util

import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Environment
import android.os.StatFs
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
            File(getFpDir(context), filename).writeText(data)
        } catch (e: Exception) {
            Log.e(TAG, "Save FP failed: $filename", e)
        }
    }

    fun loadFingerprintData(context: Context, filename: String): String? {
        return try {
            val f = File(getFpDir(context), filename)
            if (f.exists()) f.readText() else null
        } catch (e: Exception) { null }
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
        val result = try { File(getFpDir(context), filename).delete() } catch (e: Exception) { false }
        if (getActiveFpName(context) == filename) {
            context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
                .edit().remove(ACTIVE_FP_KEY).apply()
        }
        return result
    }

    @SuppressLint("HardwareIds")
    fun generateFullFp(context: Context): JSONObject {
        val now = System.currentTimeMillis()

        // Read REAL device info
        val realAndroidId = try {
            Settings.Secure.getString(context.contentResolver, Settings.Secure.ANDROID_ID) ?: randomHex(16)
        } catch (e: Exception) { randomHex(16) }

        val realGsfId = try {
            readGsfId(context) ?: randomHex(16)
        } catch (e: Exception) { randomHex(16) }

        val advertiserId = UUID.randomUUID().toString()

        // Real device storage info
        val statFs = try { StatFs(Environment.getDataDirectory().path) } catch (e: Exception) { null }
        val realTotalMem = try {
            val memInfo = File("/proc/meminfo").readText()
            val match = Regex("MemTotal:\\s+(\\d+)").find(memInfo)
            (match?.groupValues?.get(1)?.toLongOrNull() ?: 4000000L) * 1024L
        } catch (e: Exception) { 4L * 1024 * 1024 * 1024 }

        val totalStorage = statFs?.totalBytes ?: (64L * 1024 * 1024 * 1024)
        val freeStorage = statFs?.availableBytes ?: (totalStorage / 3)

        // Real app paths
        val shopeeSourceDir = try {
            context.packageManager.getApplicationInfo("com.shopee.tw", 0).sourceDir
        } catch (e: Exception) { "/data/app/com.shopee.tw/base.apk" }

        val gmsSourceDir = try {
            context.packageManager.getApplicationInfo("com.google.android.gms", 0).sourceDir
        } catch (e: Exception) { "/system/priv-app/PrebuiltGmsCore/PrebuiltGmsCoreSplit.apk" }

        val vendingSourceDir = try {
            context.packageManager.getApplicationInfo("com.android.vending", 0).sourceDir
        } catch (e: Exception) { "/system/priv-app/Phonesky/PhoneSkySplit.apk" }

        // Real installed apps for hotapps
        val hotApps = getInstalledHotApps(context)

        // Real boot count
        val bootCount = try {
            Settings.Global.getInt(context.contentResolver, Settings.Global.BOOT_COUNT, 50)
        } catch (e: Exception) { 50 }

        // Real screen timeout
        val screenTimeout = try {
            Settings.System.getInt(context.contentResolver, Settings.System.SCREEN_OFF_TIMEOUT, 60000)
        } catch (e: Exception) { 60000 }

        // Generate F7 params (only these need to be random to hide Xposed)
        val f7Params = generateF7Params()

        // Realistic timestamps based on real times
        val installOthers = now - (30L..120L).random() * 86400000L
        val updateOthers = installOthers + (1L..20L).random() * 86400000L
        val shopeeInfo = try {
            context.packageManager.getPackageInfo("com.shopee.tw", 0)
        } catch (e: Exception) { null }
        val installShopee = shopeeInfo?.firstInstallTime ?: (now - (7L..60L).random() * 86400000L)
        val updateShopee = shopeeInfo?.lastUpdateTime ?: (installShopee + (1L..10L).random() * 86400000L)

        return JSONObject().apply {
            put("name", "fp_${System.currentTimeMillis()}")
            put("androidId", realAndroidId)
            put("gsfAndroidId", realGsfId)
            put("advertiserId", advertiserId)

            put("vendingSourceDir", vendingSourceDir)
            put("gmsSourceDir", gmsSourceDir)
            put("shopeeSourceDir", shopeeSourceDir)
            put("apkSourceDir", shopeeSourceDir)

            put("keyECHashKey2", randomHex(64))
            put("keyECNewId", randomHex(32))

            put("randomInstallOthersTime", installOthers.toString())
            put("randomUpdateOthersTime", updateOthers.toString())
            put("randomInstallShopeeTime", installShopee.toString())
            put("randomUpdateShopeeTime", updateShopee.toString())

            put("totalMem", realTotalMem.toString())
            put("totalSize", totalStorage.toString())
            put("remainSize", freeStorage.toString())

            put("fix7cTime", ((now / 1000) - (0..3600).random()).toString())
            put("screenOffTimeout", screenTimeout.toString())
            put("bootCount", bootCount.toString())
            put("fixAEAccessTime", ((now / 1000) - (0..86400).random()).toString())
            put("fixAEModifyTime", ((now / 1000) - (0..86400).random()).toString())
            put("m1INodeRandom", ((100000..999999).random()).toString())
            put("vbmetaDigest", randomHex(64))

            put("hotapps", hotApps.toString())
            put("f7_params", f7Params.toString())

            val simStates = listOf("LOADED,ABSENT", "LOADED,LOADED")
            put("gsm.sim.state", simStates.random())
            put("gsm.network.type", "LTE,UNKNOWN")
            val operators = listOf("Chunghwa Telecom", "FarEasTone", "Taiwan Mobile", "T Star", "Asia Pacific Telecom")
            val op = operators.random()
            put("gsm.operator.alpha", "$op,")
            put("gsm.operator.iso-country", "tw,")

            // Fixed Taiwan locale/timezone
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

            // Mirror fields with REAL values
            put("0xaf", realAndroidId)
            put("0x9c", advertiserId)
            put("0x139", realGsfId)
            put("0xc3", bootCount.toString())
            put("0x2015", optString("vbmetaDigest"))
            put("0x146", optString("keyECHashKey2"))
            put("0x13a", optString("m1INodeRandom"))
            put("0x133", shopeeSourceDir)
            put("0x105", hotApps.toString())
            put("0x75", f7Params.toString())

            put("0x1b", optString("gsm.sim.state"))
            put("0x2e", "LTE,UNKNOWN")
            put("0x3a", optString("gsm.operator.alpha"))

            put("0xd3", JSONObject().apply {
                put("system_locales", "zh-Hant-TW")
                put("font_scale", "1.0")
            }.toString())

            put("0x8", (now - (0..86400000).random()).toString())
            put("0x91", JSONObject().apply {
                put("elapsedRealtime", ((1..72).random() * 3600000L).toString())
                put("uptimeMillis", ((1..72).random() * 3600000L).toString())
            }.toString())
            put("0xa9", (now - (0..3600000).random()).toString())
            put("0xdc", (now - (0..86400000).random()).toString())
            put("0x137", installShopee.toString())
            put("0x138", updateShopee.toString())

            // SIM operator codes for Taiwan
            val mccMnc = listOf("46692", "46601", "46689", "46697", "46605")
            put("0x41", mccMnc.random())
            put("0x40", randomHex(20))
            put("0x42", "")
            put("0x43", "")

            // Screen dimensions - use real if possible
            val dm = context.resources.displayMetrics
            put("0x63", dm.widthPixels.toString())
            put("0x64", dm.heightPixels.toString())
            put("0x65", dm.densityDpi.toString())
            put("0x66", "2")

            put("0x4", randomHex(16))
            put("0x5", randomHex(16))
            put("0x39", "")
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

    private fun generateF7Params(): JSONObject {
        return JSONObject().apply {
            put("serialHexPool", JSONArray().apply { repeat(6) { put(randomHex(40)) } })
            put("snHexPool", JSONArray().apply { repeat(6) { put(randomHex(40)) } })
            put("snDecPool", JSONArray().apply { repeat(6) { put(randomDecimal(22)) } })
            put("c1TimeOffsetDays", (-30..30).random())
            put("c2TimeOffsetDays", (-30..30).random())
            put("leafOffsetDays", (1..30).random())
            put("leafGapMs", (10000..60000).random())
            put("vbh2", randomHex(64))
            put("dl", "1")
            put("vbs", "0")
        }
    }

    @SuppressLint("HardwareIds")
    private fun readGsfId(context: Context): String? {
        return try {
            val uri = android.net.Uri.parse("content://com.google.android.gsf.gservices")
            val cursor = context.contentResolver.query(uri, null, null, arrayOf("android_id"), null)
            cursor?.use {
                if (it.moveToFirst()) {
                    val id = it.getString(1)
                    if (!id.isNullOrEmpty()) java.lang.Long.toHexString(id.toLong()) else null
                } else null
            }
        } catch (e: Exception) { null }
    }

    private fun getInstalledHotApps(context: Context): JSONArray {
        val checkApps = listOf(
            "com.facebook.katana", "com.instagram.android", "com.whatsapp",
            "com.google.android.youtube", "com.google.android.gm",
            "jp.naver.line.android", "com.facebook.orca",
            "com.twitter.android", "com.google.android.apps.maps",
            "com.spotify.music", "com.netflix.mediaclient",
            "com.tiktok.musically", "com.zhiliaoapp.musically",
            "com.google.android.apps.photos", "com.google.android.apps.docs",
            "com.google.android.calendar", "com.microsoft.office.outlook"
        )
        val installed = JSONArray()
        val pm = context.packageManager
        for (pkg in checkApps) {
            try {
                pm.getApplicationInfo(pkg, 0)
                installed.put(pkg)
            } catch (e: PackageManager.NameNotFoundException) { }
        }
        return installed
    }

    fun randomHex(length: Int): String {
        val chars = "0123456789abcdef"
        return (1..length).map { chars.random() }.joinToString("")
    }

    private fun randomDecimal(length: Int): String {
        return (1..length).map { "0123456789".random() }.joinToString("")
    }

    data class FpFileInfo(
        val name: String,
        val size: Long,
        val lastModified: Long,
        val isActive: Boolean
    )
}
