package com.app.helper.util

import android.annotation.SuppressLint
import android.app.ActivityManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Environment
import android.os.StatFs
import android.provider.Settings
import android.util.Log
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.security.SecureRandom
import java.util.Base64
import java.util.Random
import java.util.UUID

object FingerprintManager {
    private const val TAG = "FingerprintManager"
    private const val FP_DIR = "fps"
    private const val ACTIVE_FP_KEY = "active_fp"
    private val HEX_CHARS = "0123456789abcdef".toCharArray()

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
        context.getSharedPreferences("FP", Context.MODE_PRIVATE)
            .edit().putString("useFp", filename).apply()
        context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
            .edit().putString(ACTIVE_FP_KEY, filename).apply()
    }

    fun getActiveFpName(context: Context): String? {
        val spUtilsName = context.getSharedPreferences("FP", Context.MODE_PRIVATE)
            .getString("useFp", null)
        if (!spUtilsName.isNullOrEmpty()) return spUtilsName
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
            ?.sortedByDescending { it.lastModified() }
            ?.map { FpFileInfo(it.name, it.length(), it.lastModified(), it.name == activeName) }
            ?: emptyList()
    }

    fun deleteFpFile(context: Context, filename: String): Boolean {
        val result = try { File(getFpDir(context), filename).delete() } catch (e: Exception) { false }
        if (getActiveFpName(context) == filename) {
            context.getSharedPreferences("FP", Context.MODE_PRIVATE)
                .edit().remove("useFp").apply()
            context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE)
                .edit().remove(ACTIVE_FP_KEY).apply()
        }
        return result
    }

    // ============ Derive vbmetaDigest deterministically from androidId (matching csyx v/f) ============
    private fun deriveVbmetaDigest(androidId: String): String {
        var hash = 0L
        for (c in androidId) {
            hash = hash * 31 + (c.code and 0xFF)
        }
        val seed = hash and 0xFFFFFFFFL
        val r = Random(seed)
        val sb = StringBuilder(64)
        repeat(64) { sb.append(HEX_CHARS[r.nextInt(16)]) }
        return sb.toString()
    }

    // ============ Derive f7_params deterministically from androidId (matching csyx v/d FNV-1a) ============
    private fun deriveF7Params(androidId: String, vbmetaDigest: String): String {
        var hash = 0x14650fb0739d0383L
        val prime = 0x100000001b3L
        for (c in androidId) {
            hash = hash xor (c.code.toLong() and 0xFF)
            hash *= prime
        }
        val r = Random(hash)

        val serialHexPool = JSONArray()
        repeat(6) { serialHexPool.put(seededRandomHex(r, 40)) }
        val snHexPool = JSONArray()
        repeat(6) { snHexPool.put(seededRandomHex(r, 40)) }
        val snDecPool = JSONArray()
        repeat(6) {
            val sb = StringBuilder(22)
            sb.append((r.nextInt(9) + 1).toString())
            repeat(21) { sb.append(r.nextInt(10).toString()) }
            snDecPool.put(sb.toString())
        }

        val f7 = JSONObject()
        f7.put("serialHexPool", serialHexPool)
        f7.put("snHexPool", snHexPool)
        f7.put("snDecPool", snDecPool)
        f7.put("c1TimeOffsetDays", r.nextInt(61) - 30)
        f7.put("c2TimeOffsetDays", r.nextInt(61) - 30)
        f7.put("leafOffsetDays", r.nextInt(30) + 1)
        f7.put("leafGapMs", r.nextInt(50001) + 10000)
        f7.put("vbh2", vbmetaDigest)
        f7.put("dl", "1")
        f7.put("vbs", "0")
        return f7.toString()
    }

    private fun seededRandomHex(r: Random, length: Int): String {
        val sb = StringBuilder(length)
        repeat(length) { sb.append(HEX_CHARS[r.nextInt(16)]) }
        return sb.toString()
    }

    // ============ Generate device names for 0x105 (matching csyx) ============
    private fun generateDeviceNames(): String {
        val pool = listOf(
            "Fresh Start", "Eureka", "Your New Adventure", "Next Chapter",
            "New Beginnings", "Hidden Treasures", "Midnight Thoughts", "Bright Horizons",
            "Silent Echoes", "Breaking Ground", "New Horizons", "Second Chance"
        )
        val shuffled = pool.shuffled()
        val entries = (0 until 4).joinToString(", ") { i ->
            "\"$i\": \"${shuffled[i]}\""
        }
        return "{$entries}"
    }

    // ============ Generate 0x91 with exact csyx format (newlines + indentation) ============
    private fun generate0x91(): String {
        val now = System.currentTimeMillis()
        val minBoot = now - 86400000L
        val maxBoot = now - 7200000L
        val bootTime = minBoot + (Random().nextDouble() * (maxBoot - minBoot)).toLong()
        val elapsed = now - bootTime
        return "{\n    \"elapsedRealtime\": \"$elapsed\",\n    \"uptime\": \"$elapsed\"\n}"
    }

    @SuppressLint("HardwareIds")
    fun generateFullFp(context: Context): JSONObject {
        val now = System.currentTimeMillis()
        val rand = Random()

        val androidId = randomHex(16)
        val gsfId = "3" + randomHex(15)
        val advertiserId = UUID.randomUUID().toString()

        val keyECHashKey2 = Base64.getUrlEncoder().encodeToString(hexToBytes(randomHex(32)))
        val keyECNewId = Base64.getUrlEncoder().encodeToString(hexToBytes(randomHex(32)))
        val keyECExtra1 = Base64.getUrlEncoder().encodeToString(hexToBytes(randomHex(32)))
        val keyECExtra2 = Base64.getUrlEncoder().encodeToString(hexToBytes(randomHex(32)))

        // Memory with random offset (matching csyx)
        val memInfo = ActivityManager.MemoryInfo()
        try {
            (context.getSystemService(Context.ACTIVITY_SERVICE) as? ActivityManager)?.getMemoryInfo(memInfo)
        } catch (_: Exception) {}
        val totalMem = if (memInfo.totalMem > 0) memInfo.totalMem - rand.nextInt(100) * 1048576L else 4L * 1024 * 1024 * 1024

        val statFs = try { StatFs(Environment.getExternalStorageDirectory().absolutePath) } catch (_: Exception) { null }
        val blockSize = statFs?.blockSizeLong ?: 4096L
        val totalStorage = (statFs?.blockCountLong ?: (64L * 1024 * 1024 * 1024 / 4096)) * blockSize
        val totalStorageOffset = totalStorage - rand.nextInt(100) * 1048576L
        val freeRatio = 0.1 + rand.nextDouble() * 0.5
        val freeStorage = (totalStorageOffset * freeRatio).toLong()

        val shopeeSourceDir = try {
            context.packageManager.getApplicationInfo("com.shopee.tw", 0).sourceDir
        } catch (_: Exception) { "/data/app/com.shopee.tw/base.apk" }

        val gmsSourceDir = try {
            context.packageManager.getApplicationInfo("com.google.android.gms", 0).sourceDir
        } catch (_: Exception) { "/system/priv-app/PrebuiltGmsCore/PrebuiltGmsCoreSplit.apk" }

        val vendingSourceDir = try {
            context.packageManager.getApplicationInfo("com.android.vending", 0).sourceDir
        } catch (_: Exception) { "/system/priv-app/Phonesky/PhoneSkySplit.apk" }

        val bootCount = rand.nextInt(100) + 50
        val screenTimeout = listOf(15000, 30000, 60000)[rand.nextInt(3)]

        // Derive vbmetaDigest from androidId (CRITICAL - must be deterministic)
        val vbmetaDigest = deriveVbmetaDigest(androidId)
        val f7Params = deriveF7Params(androidId, vbmetaDigest)
        val deviceNames = generateDeviceNames()

        // Timestamps
        val installOthers = now - (30L + rand.nextInt(90)) * 86400000L
        val updateOthers = installOthers + (1L + rand.nextInt(19)) * 86400000L
        val installShopee = now - (7L + rand.nextInt(53)) * 86400000L
        val fixAEAccess = rand.nextInt(260000) + 1
        val fixAEModify = rand.nextInt(7200) + 300
        val m1Inode = rand.nextInt(600000) + 1

        val usbConfigs = listOf("mtp", "sec_charging", "mtp,sec_charging")
        val simState = "LOADED,ABSENT"
        val networkTypes = listOf("LTE", "NR", "LTE_CA")
        val networkType = "${networkTypes[rand.nextInt(3)]},UNKNOWN"
        val operatorAlpha = "O${rand.nextInt(3) + 1},"
        val operatorCountry = "TW,"

        // Screen dimensions - randomized (matching csyx)
        val dm = context.resources.displayMetrics
        val screenFactor = rand.nextInt(4) + 1
        val screenW = (screenFactor * dm.widthPixels) / 10
        val screenH = (screenFactor * dm.heightPixels) / 10

        // Random public IP
        val ip = generateRandomPublicIp()

        return JSONObject().apply {
            put("name", "fp_${System.currentTimeMillis()}")
            put("androidId", androidId)
            put("gsfAndroidId", gsfId)
            put("advertiserId", advertiserId)
            put("vendingSourceDir", vendingSourceDir)
            put("gmsSourceDir", gmsSourceDir)
            put("shopeeSourceDir", shopeeSourceDir)
            put("apkSourceDir", shopeeSourceDir)
            put("keyECHashKey2", keyECHashKey2)
            put("keyECNewId", keyECNewId)
            put("randomInstallOthersTime", installOthers.toString())
            put("randomUpdateOthersTime", updateOthers.toString())
            put("randomInstallShopeeTime", installShopee.toString())
            put("randomUpdateShopeeTime", installShopee.toString())
            put("totalMem", totalMem.toString())
            put("totalSize", totalStorageOffset.toString())
            put("remainSize", freeStorage.toString())
            put("fix7cTime", (bootCount + rand.nextInt(100)).toString())
            put("screenOffTimeout", screenTimeout.toString())
            put("bootCount", bootCount.toString())
            put("fixAEAccessTime", fixAEAccess.toString())
            put("fixAEModifyTime", fixAEModify.toString())
            put("m1INodeRandom", m1Inode.toString())
            put("vbmetaDigest", vbmetaDigest)
            put("hotapps", getInstalledHotApps(context).toString())
            put("f7_params", f7Params)
            put("gsm.sim.state", simState)
            put("gsm.network.type", networkType)
            put("gsm.operator.alpha", operatorAlpha)
            put("gsm.operator.iso-country", operatorCountry)

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

            // Security/detection flags
            put("0x16", "false"); put("0x19", "false"); put("0x1a", "0")
            put("0x1c", usbConfigs[rand.nextInt(3)])
            put("0x1e", "false"); put("0x1f", "false"); put("0x20", "false"); put("0x28", "false")
            put("0x3d", "0efde111a560f04fb73e18fd00cc8afa")
            put("0x46", "false"); put("0x49", "false"); put("0x4a", "false"); put("0x4b", "false")
            put("0x6f", "false"); put("0x70", "false"); put("0x71", "false"); put("0x73", "0")
            put("0x78", "[]"); put("0x79", "[]")
            put("0x8e", "false"); put("0x96", "false"); put("0x97", "false")
            put("0x98", "false"); put("0x99", "false"); put("0x9b", "0")
            put("0xa5", "com.shopee.app.asm.anr.launch.ShopeeAppComponentFactory")
            put("0xa6", "0")
            put("0xd1", "{\"wifi_on\": \"1\", \"cell_on\": \"1\", \"bluetooth_on\": \"0\", \"development_settings_enabled\": \"0\", \"auto_time_zone\": \"1\", \"adb_wifi_enabled\": \"0\", \"adb_enabled\": \"0\"}")
            put("0xc0", "0"); put("0xc1", "0"); put("0xc2", "0")
            put("0x122", "[\"/apex/com.android.runtime/bin/linker64\",\"/apex/com.android.runtime/lib64/bionic/libdl.so\",\"[anon:bytehook-plt-trampolines]\",\"[anon:shadowhook-enter]\",\"[anon:shadowhook-exit]\",\"[anon:shadowhook-hub-trampo]\"]")
            put("0xab", "1"); put("0xb2", "false"); put("0xb5", "2")
            put("0xb6", "0efde111a560f04fb73e18fd00cc8afa")
            put("0xb7", "{\"cf\": 0, \"or\": 0}")
            put("0xbd", "0"); put("0xe1", "false"); put("0xf6", "2,0")
            put("0x11d", "000000000"); put("0x11e", "000000000")
            put("0x121", "0"); put("0x123", "{}"); put("0x13e", "71919287"); put("0x13f", "7")
            put("0x14c", "0")

            // Mirror fields
            put("0x4", UUID.randomUUID().toString())
            put("0x5", randomHex(32))
            put("0x6", randomHex(32))
            put("0x7", randomHex(32))
            put("0x8", (now - rand.nextInt(86400000)).toString())
            put("0x1b", simState)
            put("0x2e", networkType)
            put("0x39", randomHex(32))
            put("0x3a", operatorAlpha)
            put("0x3e", randomHex(32))
            put("0x40", randomHex(20))
            put("0x41", ip)
            put("0x42", ip)
            put("0x43", randomHex(32))
            put("0x63", screenW.toString())
            put("0x64", screenH.toString())
            put("0x65", dm.densityDpi.toString())
            put("0x66", dm.heightPixels.toString())
            put("0x75", f7Params)
            put("0x91", generate0x91())
            put("0x9c", advertiserId)
            put("0xa4", randomHex(40))
            put("0xa9", (now - rand.nextInt(3600000)).toString())
            put("0xac", "1")
            put("0xaf", androidId)
            put("0xb3", randomHex(32))
            put("0xc3", bootCount.toString())
            put("0xd3", "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}")
            put("0xda", randomHex(8))
            put("0xdb", randomHex(8))
            put("0xdc", (now / 1000 - rand.nextInt(86400)).toString())
            put("0xe4", randomHex(32))
            put("0xe5", randomHex(32))
            put("0x102", randomHex(32))
            put("0x105", deviceNames)
            put("0x106", generateDeviceNames())
            put("0x10a", randomHex(32))
            put("0x119", randomHex(40))
            put("0x126", randomHex(16))
            put("0x127", "0,${rand.nextInt(200) + 100}")
            put("0x128", randomHex(32))
            put("0x12a", randomHex(16))
            put("0x133", shopeeSourceDir)
            put("0x137", installShopee.toString())
            put("0x138", installShopee.toString())
            put("0x139", gsfId)
            put("0x13a", m1Inode.toString())
            put("0x13b", (now / 1000 - rand.nextInt(86400)).toString())
            put("0x13c", (now / 1000 - rand.nextInt(86400)).toString())
            put("0x13d", (now / 1000 - rand.nextInt(86400)).toString())

            val secBytes = ByteArray(48)
            SecureRandom().nextBytes(secBytes)
            put("0x14a", secBytes.joinToString("") { "%02x".format(it) })
            put("0x140", generateStatInfo())
            put("0x141", generateStatInfo())
            put("0x146", keyECHashKey2)
            put("0x147", "0")
            put("0x152", keyECExtra1)
            put("0x153", keyECExtra2)
            put("0x154", randomHex(16))
            put("0x165", randomHex(32))
            put("0x166", randomHex(32))
            put("0x2015", vbmetaDigest)
        }
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
            try { pm.getApplicationInfo(pkg, 0); installed.put(pkg) }
            catch (_: PackageManager.NameNotFoundException) {}
        }
        return installed
    }

    fun randomHex(length: Int): String {
        return (1..length).map { HEX_CHARS.random() }.joinToString("")
    }

    private fun hexToBytes(hex: String): ByteArray {
        val data = ByteArray(hex.length / 2)
        for (i in data.indices) {
            data[i] = ((Character.digit(hex[i * 2], 16) shl 4) + Character.digit(hex[i * 2 + 1], 16)).toByte()
        }
        return data
    }

    private fun generateRandomPublicIp(): String {
        val r = Random()
        while (true) {
            val a = r.nextInt(223) + 1; val b = r.nextInt(256); val c = r.nextInt(256); val d = r.nextInt(256)
            if (a == 10 || a == 127) continue
            if (a == 169 && b == 254) continue
            if (a == 172 && b in 16..31) continue
            if (a == 192 && b == 168) continue
            if (a == 100 && b in 64..127) continue
            if (a >= 224) continue
            return "$a.$b.$c.$d"
        }
    }

    private fun generateStatInfo(): String {
        val r = Random()
        val now = System.currentTimeMillis() / 1000
        val ctime = now - r.nextInt(86400)
        val mtime = ctime + r.nextInt(3600)
        val atime = mtime + r.nextInt(10)
        return "${r.nextInt(2000) + 100},33188,0,0,${r.nextInt(8000) + 3000},$atime,$mtime,$ctime"
    }

    data class FpFileInfo(
        val name: String,
        val size: Long,
        val lastModified: Long,
        val isActive: Boolean
    )
}
