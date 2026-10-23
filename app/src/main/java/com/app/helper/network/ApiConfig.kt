package com.app.helper.network

object ApiConfig {
    // ===== Go 重构后台（38.175.194.43，Nginx 反代到 :2036） =====
    const val GO_BASE_URL = "http://38.175.194.43"

    // 库存模式（后台开启后走这里）
    const val INVENTORY_MODE_PATH = "/inventory/mode"     // GET  检测库存模式开关
    const val INVENTORY_TASK_PATH = "/inventory/task"     // GET  ?account_id=下发id
    const val INVENTORY_SUBMIT_PATH = "/inventory/submit" // POST {account_id,trace_id,data}

    // 普通模式（解码插件 API）
    const val API_LOGIN_PATH = "/api/login"       // POST {accountId,...}
    const val API_TASK_PATH = "/api/task"         // POST 领取任务
    const val API_SUBMIT_PATH = "/api/submit"     // POST 回传
    const val API_HEARTBEAT_PATH = "/api/heartbeat"

    // 指纹环境：从服务器领取（服务端生成 + 每日额度控制）
    const val FP_ENV_GENERATE_PATH = "/api/fp_env/generate"

    // 代理入口（更新检查 / 指纹上报 / 风控上报等）
    const val DECRYPT_PROXY_URL = "http://38.175.194.43/decrypt_proxy.php"

    // 优惠券预检（自建服务器，与网页插件一致）
    const val COUPON_PREFLIGHT_BASE_URL = "http://103.146.231.72:1998"
    const val COUPON_PREFLIGHT_TOKEN = "-CN6_PNYhHpDzN-qJw-RGecT5vOM1nY8zqORenKnFnw"
    const val COUPON_PREFLIGHT_ACCOUNT_ID = "5064336969"

    const val TARGET_PACKAGE = "com.shopee.tw"
    const val WAKE_ACTION = "com.oppo.wake.launch.action"

    const val CONTENT_PROVIDER_AUTHORITY = "com.yh.csyxgl.vivo.provider"
    const val CONTENT_PROVIDER_PATH_CURRENT = "current_fp"
    const val CONTENT_PROVIDER_PATH_FINGERPRINT_CURRENT = "fingerprint/current"
}
