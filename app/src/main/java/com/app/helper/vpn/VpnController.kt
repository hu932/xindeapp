package com.app.helper.vpn

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.Log
import com.app.helper.network.ApiConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.net.HttpURLConnection
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URL

/**
 * VPN 节点管理与自动切换。
 *
 * 设计取舍：工具本身不内嵌 VPN 引擎。
 * 内嵌需要 sing-box 的 libbox.aar，而它必须用 gomobile 自己交叉编译，
 * 没有官方预编译产物。所以这里改为控制已安装的 v2rayNG：
 *
 *   1. 从后台 /api/vpn/nodes 拉节点（已按 TCP 延迟排序）
 *   2. 生成 base64 订阅，通过 v2rayng://install-sub 导入
 *   3. 通过 Tasker FIRE_SETTING 协议启动 / 切换节点
 *   4. 定时探测连通性，掉线就换下一个节点重连
 *
 * 用户需要先装 v2rayNG 并授予 VPN 权限。
 */
object VpnController {

    private const val TAG = "VpnCtl"

    /** 主目标：v2rayNG。 */
    const val TARGET_PKG = "com.v2ray.ang"

    /**
     * 可识别的代理 App。
     * Android 11+ 的包可见性要求 manifest 的 queries 里逐个声明，
     * 这里列出的都必须同步出现在 manifest 中，否则查不到。
     */
    /**
     * 可识别的代理 App，顺序即优先级。
     *
     * v2rayNG 必须排第一：只有它能接收推送的节点订阅，
     * 其他 App 只能启停，掉线后无法自动换节点。
     *
     * 这些包名都要同步出现在 manifest 的 queries 里，
     * 否则 Android 11+ 上 getPackageInfo 会抛 NameNotFound。
     */
    val CANDIDATE_PKGS = listOf(
        "com.v2ray.ang",                    // v2rayNG —— 支持推节点
        "moe.nb4a",                         // NekoBox
        "io.nekohasekai.sagernet",           // SagerNet
        "com.github.kr328.clash",           // Clash Meta for Android
        "com.follow.clash",                 // FlClash
        "com.hiddify.hiddify",              // Hiddify
        "com.nebula.v2ray",                 // 各类改版
    )

    /**
     * 扫描已安装包时的关键词。
     * 覆盖常见命名的拼写变体，避免漏掉改版或分支版本。
     */
    private val PKG_KEYWORDS = listOf(
        "v2ray", "v2rayng", "clash", "mihomo", "nekobox", "sagernet",
        "sing-box", "singbox", "xray", "hysteria", "matsuri",
        "hiddify", "karing", "husi", "sfa", "npv",
    )

    /** 最近一次检测的说明，供界面显示。 */
    @Volatile
    var lastDetectLog: String = ""
        private set

    /** v2rayNG 的 Tasker 协议常量，取自其 AppConfig.kt。 */
    private const val ACTION_FIRE_SETTING = "com.twofortyfouram.locale.intent.action.FIRE_SETTING"
    private const val EXTRA_BUNDLE = "com.twofortyfouram.locale.intent.extra.BUNDLE"
    private const val EXTRA_SWITCH = "tasker_extra_bundle_switch"
    private const val EXTRA_GUID = "tasker_extra_bundle_guid"
    private const val DEFAULT_GUID = "Default"

    /** FlClash：靠 TempActivity 收 intent，没有推配置的接口。 */
    const val FLCLASH_PKG = "com.follow.clash"
    private const val FLCLASH_ACTIVITY = "com.follow.clash.TempActivity"
    private const val FLCLASH_ACTION_START = "com.follow.clash.action.START"
    private const val FLCLASH_ACTION_STOP = "com.follow.clash.action.STOP"

    /** 探测用的目标，能被墙外正常访问即可。 */
    private const val PROBE_HOST = "www.gstatic.com"
    private const val PROBE_PORT = 80

    data class Node(
        val region: String,
        val protocol: String,
        val name: String,
        val server: String,
        val port: Int,
        val raw: String,
        val latency: Int,
    )

    /** 已安装可用代理 App。 */
    fun isTargetInstalled(ctx: Context): Boolean = detectPkg(ctx) != null

    /**
     * 返回命中的包名，没找到返回 null。
     *
     * Android 11+ 上 getPackageInfo 对未在 queries 里声明的包会抛
     * NameNotFoundException。这里把每个候选的失败原因都记下来，
     * 便于区分「真没装」和「装了但看不见」。
     */
    /** 用户手动指定的包名，空表示自动挑。 */
    fun preferred(ctx: Context): String =
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .getString("preferred_pkg", "").orEmpty()

    fun setPreferred(ctx: Context, pkg: String) {
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .edit().putString("preferred_pkg", pkg).apply()
    }

    /** 读取 App 显示名，取不到时退回包名。 */
    fun appLabel(ctx: Context, pkg: String): String = runCatching {
        val pm = ctx.packageManager
        val ai = pm.getApplicationInfo(pkg, 0)
        pm.getApplicationLabel(ai).toString()
    }.getOrDefault(pkg)

    /**
     * 列出所有可识别的代理 App。
     *
     * 排序规则：先按 [CANDIDATE_PKGS] 的顺序（v2rayNG 最前），
     * 再按包名字母序排那些只在关键词里命中的未知改版。
     *
     * 这里必须用 getInstalledPackages 而不是逐个 getPackageInfo：
     * 部分 ROM 对 queries 声明处理不一致，逐个查会全部抛
     * NameNotFoundException，而声明 QUERY_ALL_PACKAGES 后
     * getInstalledPackages 仍能拿到全量列表。
     */
    fun detectAll(ctx: Context): List<String> {
        val installed = runCatching {
            ctx.packageManager.getInstalledPackages(0)
        }.getOrElse { e ->
            Log.w(TAG, "getInstalledPackages failed: " + e)
            emptyList()
        }
        if (installed.isEmpty()) return emptyList()

        val names = installed.map { it.packageName }
        val lowered = names.associateWith { it.lowercase() }

        // 关键词命中
        val hit = names.filter { pkg ->
            val low = lowered[pkg] ?: return@filter false
            PKG_KEYWORDS.any { low.contains(it) }
        }.toMutableSet()

        // 已知包名即使不含关键词也要算进来（比如 com.v2ray.ang 含 v2ray，
        // 但 com.follow.clash 也可能出现在这里作双保险）
        for (pkg in CANDIDATE_PKGS) {
            if (names.contains(pkg)) hit.add(pkg)
        }
        if (hit.isEmpty()) return emptyList()

        // 排序：已知包名按 CANDIDATE_PKGS 下标，未知的排后面按字母序
        return hit.sortedWith(compareBy(
            { pkg ->
                val idx = CANDIDATE_PKGS.indexOf(pkg)
                if (idx >= 0) idx else Int.MAX_VALUE
            },
            { it }
        ))
    }

    /**
     * 选出实际使用的代理 App。
     *
     * 优先级：用户手动指定 > [CANDIDATE_PKGS] 顺序（v2rayNG 优先）。
     */
    fun detectPkg(ctx: Context): String? {
        val all = detectAll(ctx)
        if (all.isEmpty()) {
            lastDetectLog = "未找到代理 App"
            Log.w(TAG, lastDetectLog)
            return null
        }

        // 手动指定优先
        val pref = preferred(ctx)
        if (pref.isNotBlank()) {
            if (all.contains(pref)) {
                lastDetectLog = "使用指定应用 " + appLabel(ctx, pref) + "（" + pref + "）"
                Log.i(TAG, lastDetectLog)
                return pref
            }
            // 指定过但已卸载，清掉
            setPreferred(ctx, "")
        }

        val picked = all.first()
        val labels = all.joinToString("、") { appLabel(ctx, it) + "(" + it + ")" }
        lastDetectLog = if (all.size > 1) {
            "使用 " + appLabel(ctx, picked) + "，共检测到 " +
                    all.size + " 个代理应用：" + labels
        } else {
            "检测到 " + appLabel(ctx, picked) + "（" + picked + "）"
        }
        Log.i(TAG, lastDetectLog)
        return picked
    }

    /**
     * 从后台拉节点，按 region 过滤（空表示不限）。
     * 返回的列表已由服务端按延迟升序排好。
     */
    suspend fun fetchNodes(ctx: Context, region: String = "", limit: Int = 30): List<Node> =
        withContext(Dispatchers.IO) {
            val out = mutableListOf<Node>()
            val url = buildString {
                append(ApiConfig.GO_BASE_URL)
                append("/api/vpn/nodes?limit=")
                append(limit)
                if (region.isNotBlank()) {
                    append("&region=")
                    append(Uri.encode(region))
                }
            }
            runCatching {
                val conn = URL(url).openConnection() as HttpURLConnection
                conn.connectTimeout = 15000
                conn.readTimeout = 20000
                conn.requestMethod = "GET"
                val body = conn.inputStream.bufferedReader().use { it.readText() }
                conn.disconnect()
                val root = JSONObject(body)
                val arr = root.optJSONArray("data") ?: JSONArray()
                for (i in 0 until arr.length()) {
                    val o = arr.optJSONObject(i) ?: continue
                    val raw = o.optString("raw", "")
                    if (raw.isBlank()) continue
                    out.add(
                        Node(
                            region = o.optString("region", "UNKNOWN"),
                            protocol = o.optString("protocol", ""),
                            name = o.optString("name", ""),
                            server = o.optString("server", ""),
                            port = o.optInt("port", 0),
                            raw = raw,
                            latency = o.optInt("latency", -1),
                        )
                    )
                }
            }.onFailure { Log.w(TAG, "fetchNodes failed: $it") }
            out
        }

    /** 把节点列表打包成 base64 订阅内容（每行一个分享链接）。 */
    private fun buildSubscription(nodes: List<Node>): String {
        val joined = nodes.joinToString("\n") { it.raw }
        return android.util.Base64.encodeToString(
            joined.toByteArray(Charsets.UTF_8),
            android.util.Base64.NO_WRAP
        )
    }

    /**
     * 通过 v2rayng://install-sub 导入订阅。
     * 用 data: URI 直接内嵌内容，避免还要自建一个可供 v2rayNG 访问的 URL。
     */
    fun importSubscription(ctx: Context, nodes: List<Node>): Boolean {
        if (nodes.isEmpty()) return false
        val b64 = buildSubscription(nodes)
        val payload = "data:application/octet-stream;base64," + Uri.encode(b64)
        return runCatching {
            val intent = Intent(Intent.ACTION_VIEW).apply {
                setPackage(TARGET_PKG)
                data = Uri.parse("v2rayng://install-sub?url=" + payload)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            ctx.startActivity(intent)
            true
        }.onFailure { Log.w(TAG, "importSubscription failed: $it") }.getOrDefault(false)
    }

    /** 启动 / 停止代理。v2rayNG 用 Tasker 广播，FlClash 用 TempActivity。 */
    fun setEnabled(ctx: Context, enabled: Boolean, guid: String = DEFAULT_GUID): Boolean {
        val pkg = detectPkg(ctx)
        if (pkg == FLCLASH_PKG) {
            return setFlClash(ctx, enabled)
        }
        return runCatching {
            val bundle = Bundle().apply {
                putBoolean(EXTRA_SWITCH, enabled)
                putString(EXTRA_GUID, guid)
            }
            val intent = Intent(ACTION_FIRE_SETTING).apply {
                setPackage(TARGET_PKG)
                putExtra(EXTRA_BUNDLE, bundle)
            }
            ctx.sendBroadcast(intent)
            Log.i(TAG, "setEnabled enabled=$enabled guid=$guid")
            true
        }.onFailure { Log.w(TAG, "setEnabled failed: $it") }.getOrDefault(false)
    }

    /**
     * 探测当前是否已连上墙外。
     * 只做 TCP 握手，不做 HTTP，速度够快也不依赖响应内容。
     */
    fun isOnline(timeoutMs: Int = 4000): Boolean = runCatching {
        Socket().use { s ->
            s.connect(InetSocketAddress(PROBE_HOST, PROBE_PORT), timeoutMs)
            true
        }
    }.getOrDefault(false)

    /**
     * 从候选节点里挑第一个能连通的。
     *
     * 只测 TCP 连通性——具体协议是否可用要等 v2rayNG 连上才知道，
     * 所以调用方需要配合 [isOnline] 做二次确认。
     */
    suspend fun pickFastest(nodes: List<Node>, skip: Set<String> = emptySet()): Node? =
        withContext(Dispatchers.IO) {
            for (n in nodes) {
                if (skip.contains(n.server + ":" + n.port)) continue
                val ok = runCatching {
                    Socket().use { s ->
                        s.connect(InetSocketAddress(n.server, n.port), 3500)
                        true
                    }
                }.getOrDefault(false)
                if (ok) return@withContext n
            }
            null
        }

    /** 该 App 是否支持推送节点（决定能否做自动切换）。 */
    fun supportsNodePush(pkg: String?): Boolean = pkg == TARGET_PKG

    /** 记录上次使用的节点，便于掉线时跳过。 */
    fun rememberLast(ctx: Context, key: String) {
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .edit().putString("last_node", key).apply()
    }

    fun lastNode(ctx: Context): String =
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .getString("last_node", "").orEmpty()

    fun setAutoEnabled(ctx: Context, on: Boolean) {
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .edit().putBoolean("auto_enabled", on).apply()
    }

    fun isAutoEnabled(ctx: Context): Boolean =
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .getBoolean("auto_enabled", false)

    fun setRegion(ctx: Context, region: String) {
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .edit().putString("region", region).apply()
    }

    fun region(ctx: Context): String =
        ctx.getSharedPreferences("vpn_state", Context.MODE_PRIVATE)
            .getString("region", "").orEmpty()

    /** 打开 v2rayNG 主界面，用于用户首次授予 VPN 权限。 */
    fun openTargetApp(ctx: Context) {
        runCatching {
            val i = ctx.packageManager.getLaunchIntentForPackage(TARGET_PKG)
            if (i != null) {
                i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                ctx.startActivity(i)
            }
        }
    }

    /** 引导去应用商店 / 浏览器安装 v2rayNG。 */
    fun openInstallPage(ctx: Context) {
        runCatching {
            val i = Intent(Intent.ACTION_VIEW).apply {
                data = Uri.parse("https://github.com/2dust/v2rayNG/releases")
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            ctx.startActivity(i)
        }
    }

    /**
     * 控制 FlClash。
     *
     * 它通过一个无界面的 TempActivity 接收 START / STOP intent，
     * 收到后立即 finish，不弹界面。
     *
     * 注意：FlClash 没有「导入配置」的对外接口，所以这里只能启停，
     * 无法像 v2rayNG 那样把节点列表推过去。掉线后要换节点，
     * 需要用户在 FlClash 里先配好包含多个节点的订阅，
     * 再靠它自己的策略组切换。
     */
    private fun setFlClash(ctx: Context, enabled: Boolean): Boolean {
        return runCatching {
            val action = if (enabled) FLCLASH_ACTION_START else FLCLASH_ACTION_STOP
            val intent = Intent().apply {
                setClassName(FLCLASH_PKG, FLCLASH_ACTIVITY)
                this.action = action
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            ctx.startActivity(intent)
            Log.i(TAG, "flclash " + if (enabled) "START" else "STOP")
            true
        }.onFailure { Log.w(TAG, "flclash control failed: " + it) }.getOrDefault(false)
    }

}
