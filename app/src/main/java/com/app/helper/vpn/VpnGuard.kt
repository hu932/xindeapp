package com.app.helper.vpn

import android.content.Context
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * VPN 自动保活。
 *
 * 开启后按固定间隔探测连通性：
 *   - 正常 → 什么都不做
 *   - 掉线 → 从节点列表里挑一个没试过的、TCP 能通的，导入并重连
 *   - 连续失败 → 重新拉一次节点列表（可能是节点集体失效）
 *
 * 探测本身是「TCP 连 gstatic:80」，成本很低，不会明显耗电。
 */
object VpnGuard {

    private const val TAG = "VpnGuard"

    /** 探测间隔。太短会耗电，太长掉线后业务已经失败了。 */
    private const val CHECK_INTERVAL_MS = 15_000L

    /** 连续几次探测失败才认为真的掉线，避免偶发波动就换节点。 */
    private const val FAIL_THRESHOLD = 2

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var job: Job? = null

    @Volatile
    private var failures = 0

    /** 本次会话已经试过的节点，避免反复撞同一个死节点。 */
    private val tried = mutableSetOf<String>()

    @Volatile
    var lastMessage: String = ""
        private set

    fun isRunning(): Boolean = job?.isActive == true

    /** 启动守护。重复调用安全。 */
    fun start(ctx: Context) {
        if (isRunning()) return
        val app = ctx.applicationContext
        tried.clear()
        failures = 0
        lastMessage = "守护已启动"
        job = scope.launch {
            Log.i(TAG, "guard started")
            var current: VpnController.Node? = null
            while (isActive) {
                delay(CHECK_INTERVAL_MS)
                if (!VpnController.isAutoEnabled(app)) {
                    lastMessage = "自动模式已关闭"
                    continue
                }
                val online = VpnController.isOnline()
                if (online) {
                    if (failures > 0) Log.i(TAG, "back online")
                    failures = 0
                    continue
                }
                failures++
                lastMessage = "探测失败 $failures/$FAIL_THRESHOLD"
                Log.w(TAG, "probe failed, count=$failures")
                if (failures < FAIL_THRESHOLD) continue

                // 确认掉线，开始换节点
                failures = 0
                val region = VpnController.region(app)
                var nodes = VpnController.fetchNodes(app, region, 30)
                if (nodes.isEmpty()) {
                    lastMessage = "节点池为空，无法切换"
                    Log.w(TAG, "node pool empty")
                    continue
                }
                // 把试过的排到后面
                nodes = nodes.sortedBy { tried.contains(it.server + ":" + it.port) }

                val next = VpnController.pickFastest(nodes, tried)
                if (next == null) {
                    // 全都试过了还不行 → 清空重来，顺便重新拉列表
                    tried.clear()
                    lastMessage = "候选节点全部不可用，已重置重试"
                    Log.w(TAG, "all candidates failed, reset")
                    continue
                }
                val key = next.server + ":" + next.port
                tried.add(key)
                VpnController.rememberLast(app, key)

                lastMessage = "切换到 ${next.region} ${next.server}:${next.port}"
                Log.i(TAG, "switching to $key (${next.protocol})")

                // 只有 v2rayNG 能接收节点订阅；其他 App 只能重启现有配置
                val pkg = VpnController.detectPkg(app)
                if (VpnController.supportsNodePush(pkg)) {
                    val imported = VpnController.importSubscription(
                        app, nodes.filter { it.raw.isNotBlank() }
                    )
                    if (!imported) {
                        lastMessage = "导入订阅失败，检查 v2rayNG 是否已安装"
                        continue
                    }
                    delay(1500)
                } else {
                    // FlClash 等没有推配置接口：先断开再启动，
                    // 让它在已有订阅里按策略组重新选路
                    lastMessage = "应用不支持推送节点，重启代理以重选线路"
                    VpnController.setEnabled(app, false)
                    delay(1200)
                }
                VpnController.setEnabled(app, true)
                current = next
                delay(3000)
                if (VpnController.isOnline()) {
                    lastMessage = "已连接 ${next.region} ${next.server}:${next.port}"
                    Log.i(TAG, "connected via $key")
                } else {
                    lastMessage = "切换后仍不通，继续找下一个"
                }
            }
            Log.i(TAG, "guard stopped, last=${current?.server}")
        }
    }

    fun stop() {
        job?.cancel()
        job = null
        tried.clear()
        failures = 0
        lastMessage = "守护已停止"
        Log.i(TAG, "guard stopped")
    }

    /**
     * 手动执行一次「连上最快的节点」。
     * 用于界面上的「一键连接」。
     */
    fun connectNow(ctx: Context, onResult: (Boolean, String) -> Unit) {
        val app = ctx.applicationContext
        scope.launch {
            lastMessage = "正在获取节点..."
            val region = VpnController.region(app)
            val nodes = VpnController.fetchNodes(app, region, 30)
            if (nodes.isEmpty()) {
                lastMessage = "没有可用节点"
                withContext(Dispatchers.Main) { onResult(false, lastMessage) }
                return@launch
            }
            lastMessage = "共 ${nodes.size} 个候选，正在测速..."
            val picked = VpnController.pickFastest(nodes)
            if (picked == null) {
                lastMessage = "候选节点都连不上"
                withContext(Dispatchers.Main) { onResult(false, lastMessage) }
                return@launch
            }
            lastMessage = "导入订阅..."
            VpnController.importSubscription(app, nodes)
            delay(1500)
            lastMessage = "启动 ${picked.region} ${picked.server}:${picked.port}"
            VpnController.setEnabled(app, true)
            delay(4000)
            val ok = VpnController.isOnline()
            lastMessage = if (ok) {
                "已连接 ${picked.region} · ${picked.latency}ms"
            } else {
                "已启动但探测不通，可能节点失效"
            }
            Log.i(TAG, "connectNow result=$ok ${picked.server}")
            withContext(Dispatchers.Main) { onResult(ok, lastMessage) }
        }
    }
}
