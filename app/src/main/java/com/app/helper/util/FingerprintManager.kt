package com.app.helper.util

import android.annotation.SuppressLint
import android.app.ActivityManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Environment
import android.os.StatFs
import android.provider.MediaStore
import android.system.Os
import android.util.Log
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.security.MessageDigest
import java.security.SecureRandom
import java.text.SimpleDateFormat
import java.util.Base64
import java.util.Date
import java.util.Locale
import java.util.Random
import java.util.UUID
import java.util.concurrent.ThreadLocalRandom

object FingerprintManager {
    private const val TAG = "FingerprintManager"
    private const val FP_DIR = "fps"
    private const val ACTIVE_FP_KEY = "active_fp"
    private val HEX_CHARS = "0123456789abcdef".toCharArray()
    private val rand = Random()

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
        catch (e: Exception) { null }
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
        val dir = getFpDir(context)
        return dir.listFiles()?.sortedByDescending { it.lastModified() }?.firstOrNull()?.let {
            setActiveFp(context, it.name); it.readText()
        }
    }

    fun listFpFiles(context: Context): List<FpFileInfo> {
        val dir = getFpDir(context); val activeName = getActiveFpName(context)
        return dir.listFiles()?.sortedByDescending { it.lastModified() }
            ?.map { FpFileInfo(it.name, it.length(), it.lastModified(), it.name == activeName) } ?: emptyList()
    }

    fun deleteFpFile(context: Context, filename: String): Boolean {
        val result = try { File(getFpDir(context), filename).delete() } catch (e: Exception) { false }
        if (getActiveFpName(context) == filename) {
            context.getSharedPreferences("FP", Context.MODE_PRIVATE).edit().remove("useFp").apply()
            context.getSharedPreferences("fp_prefs", Context.MODE_PRIVATE).edit().remove(ACTIVE_FP_KEY).apply()
        }
        return result
    }

    // ============ Exact csyx v/f.c() replica ============
    @SuppressLint("HardwareIds", "PackageManagerGetSignatures")
    fun generateFullFp(context: Context): JSONObject {
        // === Variable generation (matching csyx v/f.c line by line) ===
        val b5 = rHex(16)                                          // androidId
        val str7 = "3" + rHex(15)                                  // gsfAndroidId
        val uuid = UUID.randomUUID().toString()                    // advertiserId
        val encodeToString = b64(rHex(32))                         // vendingSourceDir (Base64!)
        val encodeToString2 = b64(rHex(32))                        // gmsSourceDir (Base64!)
        val encodeToString3 = b64(rHex(32))                        // shopeeSourceDir (Base64!)
        val encodeToString4 = b64(rHex(32))                        // apkSourceDir (Base64!)
        val uuid2 = UUID.randomUUID().toString()                   // 0x4
        val b6 = rHex(32)                                          // 0x5
        val b7 = rHex(32)                                          // 0x6
        val b8 = rHex(32)                                          // 0x7
        val b9 = generateMacAddress()                              // 0x9c (MAC address from v/e.b())
        val b10 = rHex(32)                                         // 0xe4
        val b11 = rHex(32)                                         // 0xe5
        val b12 = rHex(64)                                         // 0x39
        val b13 = rHex(64)                                         // 0x102
        val b14 = rHex(32)                                         // 0x10a
        val b15 = rHex(32)                                         // 0x43
        val b16 = rHex(40)                                         // 0xa4
        val b17 = rHex(16)                                         // keyECHashKey2 = b17 (NOT Base64!)
        val b18 = rHex(16)                                         // part of keyECNewId
        val q = "$b17-$b18"                                        // keyECNewId = b17-b18
        val b19 = rHex(32)                                         // 0x126
        val b20 = rHex(16)                                         // 0x3e
        val b21 = rHex(32)                                         // 0x12a
        val b22 = rHex(32)                                         // 0x165
        val b23 = rHex(32)                                         // 0x166

        // sb2: 10-digit random decimal string for 0xb3
        val sb2 = buildString { repeat(10) { append(rand.nextInt(10)) } }

        // MediaStore version for 0x119
        val str8 = try {
            MediaStore.getVersion(context).split(":")[0] + ":" + UUID.randomUUID()
        } catch (_: Exception) { "1:" + UUID.randomUUID() }

        // l4: decimal representation of unsigned hex for 0xaf
        val l4 = try { java.lang.Long.parseUnsignedLong(b17, 16).toString() } catch (_: Exception) { "0" }

        // Timestamps
        val now = System.currentTimeMillis()
        val d4 = randomLong(now - 604800000L, now - 172800000L)    // random time in last 7d-2d
        val valueOf = d4.toString()                                 // randomInstallShopeeTime + Update
        val valueOf2 = randomLong(parseDateMs("2022-01-01 00:00:00"), parseDateMs("2024-01-01 00:00:00")).toString()
        val valueOf3 = randomLong(parseDateMs("2024-01-01 00:00:00"), parseDateMs("2025-09-01 00:00:00")).toString()

        // Memory
        val memInfo = ActivityManager.MemoryInfo()
        try { (context.getSystemService(Context.ACTIVITY_SERVICE) as? ActivityManager)?.getMemoryInfo(memInfo) } catch (_: Exception) {}
        val nextInt5 = (if (memInfo.totalMem > 0) memInfo.totalMem else 4L * 1024 * 1024 * 1024) - rand.nextInt(100) * 1048576L

        // Storage
        val statFs = try { StatFs(Environment.getExternalStorageDirectory().absolutePath) } catch (_: Exception) { null }
        val blockCountLong = ((statFs?.blockCountLong ?: 0L) * (statFs?.blockSizeLong ?: 4096L)) - rand.nextInt(100) * 1048576L
        val nextDouble = (blockCountLong * (rand.nextDouble() * 0.5 + 0.1)).toLong()

        val nextInt6 = rand.nextInt(100) + 100                    // fix7cTime
        val i6 = intArrayOf(15000, 30000, 60000)[rand.nextInt(3)]  // screenOffTimeout
        val nextInt7 = rand.nextInt(100) + 50                      // bootCount
        val nextInt8 = rand.nextInt(259200) + 1                    // fixAEAccessTime
        val nextInt9 = rand.nextInt(7200) + 300                    // fixAEModifyTime
        val nextInt10 = rand.nextInt(300000000) + 300000000        // screen width base
        val nextInt11 = rand.nextInt(1000000) + 1000000            // screen height base

        // USB config
        val str12 = arrayOf("mtp", "sec_charging", "mtp,sec_charging")[rand.nextInt(3)]

        // 0x91: elapsedRealtime
        val d5Boot = randomLong(parseDateMs(formatDateMs(now - 86400000L)), parseDateMs(formatDateMs(now - 7200000L)))
        val j6 = d5Boot / 1000                                     // 0xdc
        val currentTimeMillis2 = now - d5Boot                       // elapsed
        val l5 = "{\n    \"elapsedRealtime\": \"$currentTimeMillis2\",\n    \"uptime\": \"$currentTimeMillis2\"\n}"
        val valueOf4 = (currentTimeMillis2 + randomInt(100, 300)).toString()  // 0x137, 0x138

        // Network IPs (matching v/e)
        val ipSub = rand.nextInt(256).toString()
        val str_ip = "192.168.$ipSub.${rand.nextInt(256)}"         // 0x40
        val str2_gw = "192.168.$ipSub.1"                           // 0x42
        val format_ipv6 = String.format("fe80::%x:%x:%x:%x",
            randomInt(0, 65535), randomInt(0, 65535), randomInt(0, 65535), randomInt(0, 65535))

        // m1INodeRandom
        val eVal = randomInt(600000, 900000)

        // 0x127
        val str11 = "0,${rand.nextInt(200) + 100}"

        // vbmetaDigest: derived from androidId (hash*31 seed)
        val str5 = deriveVbmetaDigest(b5)

        // f7_params: derived from androidId (FNV-1a seed)
        val f7Params = deriveF7Params(b5, str5)

        // 0x105: device names
        val str13 = generateDeviceNames()

        // 0x106: 7-item device stats
        val str18 = generate0x106()

        // Network interface info for 0x146
        val str3_net = try {
            val parts = str_ip.split(".")
            if (parts.size == 4) "${parts[0]}.${parts[1]}.${parts[2]}.0" else "0.0.0.0"
        } catch (_: Exception) { "0.0.0.0" }
        val t3 = "[\"" + "{InterfaceName: wlan0 LinkAddresses: [ $format_ipv6/64,$str_ip/24 ] DnsAddresses: [ /8.8.8.8,/1.1.1.1 ] Domains: null MTU: 0 ServerAddress: /$str2_gw TcpBufferSizes: 524288,1048576,8388608,524288,1048576,4194304 Routes: [ fe80::/64 -> :: wlan0 mtu 0,$str3_net/24 -> 0.0.0.0 wlan0 mtu 0,0.0.0.0/0 -> $str2_gw wlan0 mtu 0 ]}" + "\"]"

        // 0x140/0x141: stat info + MD5
        val d92 = randomLong(parseDateMs("2024-01-01 00:00:00"), parseDateMs(formatDateMs(now))) / 1000
        val nextInt142 = ThreadLocalRandom.current().nextInt(0, 10) + d92
        val str202 = "${ThreadLocalRandom.current().nextInt(100, 2000)},33188,0,0,${ThreadLocalRandom.current().nextInt(300, 8000)},${ThreadLocalRandom.current().nextInt(0, 86400) + nextInt142},$nextInt142,$d92"
        val str212 = try {
            val digest = MessageDigest.getInstance("MD5").digest((str202 + b7).toByteArray(Charsets.UTF_8))
            digest.joinToString("") { "%02x".format(it.toInt() and 0xFF) }
        } catch (_: Exception) { rHex(32) }

        // 0x75: fake APK path (NOT f7_params!)
        val path0x75 = "/data/app/~~$encodeToString4/com.shopee.tw-$encodeToString3/split_config.arm64_v8a.apk!libshpssdk.so"

        // d6/d7/d8 for 0x13b/0x13c/0x13d
        val d6 = d4 - randomLong(5000, 12000)
        val d7 = randomLong(0, 10000) + d6
        val d8 = randomLong(0, 10000) + d7

        // Package info for 0x152/0x153/0x154
        val a4 = getPackageInfoString(context, "com.android.systemui")
        val a5 = getPackageInfoString(context, "com.android.settings")
        val a6 = getPackageInfoString(context, "android")

        // gsm fields
        val networkTypes = arrayOf("NR", "LTE", "LTE_CA")
        val str22 = "${networkTypes[rand.nextInt(3)]},UNKNOWN"
        val opNames = arrayOf("O1", "O2", "O3")
        val concat = "${opNames[rand.nextInt(3)]},"
        val ip = generateRandomPublicIp()

        // ============ Build HashMap (exact csyx field mapping) ============
        val map = HashMap<String, String>()

        // Defaults from a()
        arrayOf("0x16","0x19","0x1e","0x1f","0x20","0x28","0x46","0x49","0x4a","0x4b",
            "0x6f","0x70","0x71","0x8e","0x96","0x97","0x98","0x99","0xb2","0xe1").forEach { map[it] = "false" }
        arrayOf("0x1a","0x73","0x9b","0xa6","0xc0","0xc1","0xc2","0xbd","0x121","0x14c").forEach { map[it] = "0" }
        map["0x3d"] = "0efde111a560f04fb73e18fd00cc8afa"
        map["0x78"] = "[]"; map["0x79"] = "[]"
        map["0xa5"] = "com.shopee.app.asm.anr.launch.ShopeeAppComponentFactory"
        map["0xd1"] = "{\"wifi_on\": \"1\", \"cell_on\": \"1\", \"bluetooth_on\": \"0\", \"development_settings_enabled\": \"0\", \"auto_time_zone\": \"1\", \"adb_wifi_enabled\": \"0\", \"adb_enabled\": \"0\"}"
        map["0x122"] = "[\"/apex/com.android.runtime/bin/linker64\",\"/apex/com.android.runtime/lib64/bionic/libdl.so\",\"[anon:bytehook-plt-trampolines]\",\"[anon:shadowhook-enter]\",\"[anon:shadowhook-exit]\",\"[anon:shadowhook-hub-trampo]\"]"
        map["0xab"] = "1"; map["0xb5"] = "2"
        map["0xb6"] = "0efde111a560f04fb73e18fd00cc8afa"
        map["0xb7"] = "{\"cf\": 0, \"or\": 0}"
        map["0xf6"] = "2,0"
        map["0x11d"] = "000000000"; map["0x11e"] = "000000000"
        map["0x123"] = "{}"; map["0x13e"] = "71919287"; map["0x13f"] = "7"

        // Dynamic values from c()
        map["androidId"] = b5
        map["gsfAndroidId"] = str7
        map["advertiserId"] = uuid
        map["vendingSourceDir"] = encodeToString
        map["gmsSourceDir"] = encodeToString2
        map["keyECHashKey2"] = b18
        map["keyECNewId"] = q
        map["randomInstallShopeeTime"] = valueOf
        map["randomUpdateShopeeTime"] = valueOf
        map["randomInstallOthersTime"] = valueOf2
        map["randomUpdateOthersTime"] = valueOf3
        map["totalMem"] = nextInt5.toString()
        map["totalSize"] = blockCountLong.toString()
        map["remainSize"] = nextDouble.toString()
        map["shopeeSourceDir"] = encodeToString3.toString()
        map["apkSourceDir"] = encodeToString4.toString()
        map["fix7cTime"] = nextInt6.toString()
        map["screenOffTimeout"] = i6.toString()
        map["bootCount"] = nextInt7.toString()
        map["fixAEAccessTime"] = nextInt8.toString()
        map["fixAEModifyTime"] = nextInt9.toString()
        map["m1INodeRandom"] = eVal.toString()
        map["vbmetaDigest"] = str5
        map["hotapps"] = getInstalledHotApps(context).toString()
        map["f7_params"] = f7Params

        map["0x4"] = uuid2
        map["0x5"] = b6
        map["0x6"] = b7
        map["0x7"] = b8
        map["0x8"] = d5Boot.toString()
        map["0x1c"] = str12
        map["0x3e"] = b20
        map["0x39"] = b12
        map["0x40"] = str_ip
        map["0x42"] = str2_gw
        map["0x43"] = b15
        map["0x44"] = "zh-Hant-TW"
        map["0x59"] = "Asia/Taipei"
        map["0x50"] = "1"
        map["0x5d"] = "1"
        map["0x62"] = "14"
        map["0x63"] = ((randomInt(1, 4) * nextInt10) / 10).toString()
        map["0x64"] = ((randomInt(1, 4) * nextInt11) / 10).toString()
        map["0x65"] = nextInt10.toString()
        map["0x66"] = nextInt11.toString()
        map["0x67"] = "[]"; map["0x68"] = "[]"; map["0x6a"] = "0"; map["0x6d"] = "[]"; map["0x6c"] = "0"
        map["0x75"] = path0x75
        map["0x91"] = l5
        map["0x9c"] = b9
        map["0xa4"] = b16
        map["0xa9"] = d5Boot.toString()
        map["0xaa"] = "-1"
        map["0xac"] = "1"
        map["0xaf"] = l4
        map["0xb3"] = sb2
        map["0xc3"] = nextInt7.toString()
        map["0xdc"] = j6.toString()
        map["0xe4"] = b10
        map["0xe5"] = b11
        map["0x102"] = b13
        map["0x105"] = str13
        map["0x106"] = str18
        map["0x10a"] = b14
        map["0x119"] = str8
        map["0x126"] = b19
        map["0x127"] = str11
        map["0x12a"] = b21
        map["0x128"] = format_ipv6
        map["0x133"] = UUID.randomUUID().toString()
        map["0x137"] = valueOf4
        map["0x138"] = valueOf4
        map["0x139"] = encodeToString3
        map["0x13a"] = eVal.toString()
        map["0x13b"] = d6.toString()
        map["0x13c"] = d8.toString()
        map["0x13d"] = d7.toString()

        val secBytes = ByteArray(48)
        SecureRandom().nextBytes(secBytes)
        val sb14a = CharArray(96)
        val hexC = "0123456789abcdef".toCharArray()
        for (i in 0 until 48) {
            val b = secBytes[i].toInt() and 0xFF
            sb14a[i * 2] = hexC[b ushr 4]
            sb14a[i * 2 + 1] = hexC[b and 0xF]
        }
        map["0x14a"] = String(sb14a)

        map["0x140"] = str202
        map["0x141"] = str212
        map["0x146"] = t3
        map["0x152"] = a4
        map["0x153"] = a5
        map["0x154"] = a6
        map["0x165"] = b22
        map["0x166"] = b23
        map["0x2015"] = str5
        map["0xd3"] = "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}"

        map["gsm.sim.state"] = "LOADED,ABSENT"
        map["gsm.network.type"] = str22
        map["gsm.operator.alpha"] = concat
        map["gsm.operator.iso-country"] = "TW,"
        map["0x1b"] = "LOADED,ABSENT"
        map["0x2e"] = str22
        map["0x3a"] = concat
        map["0x41"] = ip

        // Convert to JSONObject
        val json = JSONObject()
        json.put("name", "fp_${System.currentTimeMillis()}")
        for ((k, v) in map) { json.put(k, v) }
        return json
    }

    // ============ Helper methods matching csyx exactly ============

    private fun rHex(len: Int): String {
        val sb = StringBuilder(len)
        repeat(len) { sb.append(Integer.toHexString(rand.nextInt(16))) }
        return sb.toString()
    }

    fun randomHex(len: Int) = rHex(len)

    private fun b64(hex: String): String {
        val bytes = ByteArray(hex.length / 2)
        for (i in bytes.indices) bytes[i] = ((Character.digit(hex[i * 2], 16) shl 4) + Character.digit(hex[i * 2 + 1], 16)).toByte()
        return Base64.getUrlEncoder().encodeToString(bytes)
    }

    private fun generateMacAddress(): String {
        val b = ByteArray(6); Random().nextBytes(b)
        b[0] = ((b[0].toInt() and 0xFE) or 0x02).toByte()
        return b.joinToString(":") { "%02x".format(it) }
    }

    private fun randomLong(min: Long, max: Long): Long {
        if (min >= max) return min
        return min + (rand.nextDouble() * (max - min + 1)).toLong()
    }

    private fun randomInt(min: Int, max: Int): Int = rand.nextInt(max - min + 1) + min

    private fun parseDateMs(s: String): Long {
        return try { SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).parse(s)?.time ?: 0L }
        catch (_: Exception) {
            try { SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.ROOT).parse(s)?.time ?: 0L }
            catch (_: Exception) { 0L }
        }
    }

    private fun formatDateMs(ms: Long): String = SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).format(Date(ms))

    private fun deriveVbmetaDigest(androidId: String): String {
        var hash = 0L
        for (c in androidId) hash = hash * 31 + (c.code and 0xFF)
        val r = Random(hash and 0xFFFFFFFFL)
        val sb = StringBuilder(64)
        repeat(64) { sb.append(HEX_CHARS[r.nextInt(16)]) }
        return sb.toString()
    }

    private fun deriveF7Params(androidId: String, vbmetaDigest: String): String {
        var hash = 1469598103934665603L
        for (c in androidId) hash = (hash xor c.code.toLong()) * 1099511628211L
        val r = Random(hash)
        val serials = Array(6) { seededHex(r, 40) }
        val snHex = Array(6) { seededHex(r, 40) }
        val snDec = Array(6) {
            val c = CharArray(22); c[0] = (r.nextInt(9) + 49).toChar()
            for (i in 1 until 22) c[i] = (r.nextInt(10) + 48).toChar(); String(c)
        }
        val c1 = r.nextInt(61) - 30; val c2 = r.nextInt(61) - 30
        val leaf = -(r.nextInt(30) + 1); val gap = r.nextInt(50000) + 10000
        val f7 = JSONObject()
        f7.put("serialHexPool", JSONArray(serials))
        f7.put("snHexPool", JSONArray(snHex))
        f7.put("snDecPool", JSONArray(snDec))
        f7.put("c1TimeOffsetDays", c1); f7.put("c2TimeOffsetDays", c2)
        f7.put("leafOffsetDays", leaf); f7.put("leafGapMs", gap)
        f7.put("vbh2", vbmetaDigest); f7.put("dl", "1"); f7.put("vbs", "0")
        return f7.toString()
    }

    private fun seededHex(r: Random, len: Int): String {
        val c = CharArray(len); for (i in 0 until len) c[i] = HEX_CHARS[r.nextInt(16)]; return String(c)
    }

    private fun generateDeviceNames(): String {
        val pool = listOf("Fresh Start","Eureka","Your New Adventure","Next Chapter","New Beginnings",
            "Hidden Treasures","Midnight Thoughts","Bright Horizons","Silent Echoes","Breaking Ground","New Horizons","Second Chance")
        val shuffled = pool.shuffled()
        return "{" + (0 until 4).joinToString(", ") { "\"$it\": \"${shuffled[it]}\"" } + "}"
    }

    private fun generate0x106(): String {
        fun statLine(a: Int, b: Int): String { val v1 = randomInt(a, b); return "$v1,7,${randomInt(v1, 7)},${randomInt(v1, 7)},-1,-1,-1,-1,-1" }
        val items = listOf(
            run { val e = randomInt(0,1); val e5 = randomInt(e,7); "$e,7,$e5,$e5,-1,-1,-1,-1,-1" },
            run { val a = randomInt(0,7); val b = randomInt(0,7); "0,7,$a,$b" },
            run { val e = randomInt(0,7); "0,7,$e,$e,-1,-1,-1,-1,-1,-1" },
            run { val e = randomInt(0,25); "0,25,$e,$e,-1,${ randomInt(0,25)},-1,-1,-1,-1" },
            run { val e = randomInt(0,1); val e9 = randomInt(e,7); "$e,7,$e9,$e9,-1" },
            run { val e = randomInt(0,7); "0,7,$e,$e,-1,-1,-1,-1" },
            "${randomInt(0,7)},-1,-1,-1,-1,-1"
        )
        return "{" + items.mapIndexed { i, v -> "\"$i\": \"$v\"" }.joinToString(", ") + "}"
    }

    private fun generateRandomPublicIp(): String {
        val r = Random()
        while (true) {
            val a = r.nextInt(223) + 1; val b = r.nextInt(256); val c = r.nextInt(256); val d = r.nextInt(256)
            if (a == 10 || a == 127 || (a == 169 && b == 254) || (a == 172 && b in 16..31) ||
                (a == 192 && b == 168) || (a == 100 && b in 64..127) || (a == 192 && b == 0 && c == 0) ||
                (a == 192 && b == 0 && c == 2) || (a == 198 && (b == 18 || b == 19)) ||
                (a == 198 && b == 51 && c == 100) || (a == 203 && b == 0 && c == 113) ||
                a >= 224 || (a == 255 && b == 255 && c == 255 && d == 255) || a == 0) continue
            return "$a.$b.$c.$d"
        }
    }

    @SuppressLint("PackageManagerGetSignatures")
    private fun getPackageInfoString(context: Context, packageName: String): String {
        return try {
            val pm = context.packageManager
            val pi = pm.getPackageInfo(packageName, 0)
            val srcDir = pi.applicationInfo.sourceDir ?: return "d087e72912fba064cafa78dc34aea839"
            val stat = Os.lstat(srcDir)
            val size = stat.st_size
            val inode = stat.st_ino + rand.nextInt(9) + 2
            val ver = pi.versionName ?: ""
            val verCode = if (Build.VERSION.SDK_INT >= 28) pi.longVersionCode else pi.versionCode.toLong()
            val install = pi.firstInstallTime
            val delta = pi.lastUpdateTime - install
            val sig = try {
                if (Build.VERSION.SDK_INT >= 28) {
                    val si = pm.getPackageInfo(packageName, PackageManager.GET_SIGNING_CERTIFICATES).signingInfo
                    val sigs = if (si.hasMultipleSigners()) si.apkContentsSigners else si.signingCertificateHistory
                    sigs?.firstOrNull()?.let { md5(it.toByteArray()) }
                } else {
                    @Suppress("DEPRECATION")
                    pm.getPackageInfo(packageName, PackageManager.GET_SIGNATURES).signatures?.firstOrNull()?.let { md5(it.toByteArray()) }
                }
            } catch (_: Exception) { null } ?: "d087e72912fba064cafa78dc34aea839"
            "$sig$size,$ver,$verCode,$install,$delta,$inode"
        } catch (_: Exception) { "d087e72912fba064cafa78dc34aea839" }
    }

    private fun md5(data: ByteArray): String {
        return try {
            MessageDigest.getInstance("MD5").digest(data).joinToString("") { "%02x".format(it.toInt() and 0xFF) }
        } catch (_: Exception) { "d087e72912fba064cafa78dc34aea839" }
    }

    private fun getInstalledHotApps(context: Context): JSONArray {
        val apps = listOf("com.facebook.katana","com.instagram.android","com.whatsapp","com.google.android.youtube",
            "com.google.android.gm","jp.naver.line.android","com.facebook.orca","com.twitter.android",
            "com.google.android.apps.maps","com.spotify.music","com.netflix.mediaclient","com.tiktok.musically",
            "com.zhiliaoapp.musically","com.google.android.apps.photos","com.google.android.apps.docs",
            "com.google.android.calendar","com.microsoft.office.outlook")
        val r = JSONArray()
        for (p in apps) { try { context.packageManager.getApplicationInfo(p, 0); r.put(p) } catch (_: PackageManager.NameNotFoundException) {} }
        return r
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
