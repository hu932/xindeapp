package com.app.helper.network

object ApiConfig {
    // ===== 团队C 任务平台接口（登录后取任务 / 上传任务） =====
    const val TEAM_BASE_URL = "http://43.128.97.191:3008"
    const val TEAM_TASK_TAKE_PATH = "/api/task"          // GET ?phone=xxx
    const val TEAM_TASK_UPLOAD_PATH = "/task/upload/json" // POST

    // ===== 解密接口（后台 Go 服务，?act=decrypt_test） =====
    const val DECRYPT_URL = "http://103.146.231.72:2238/?act=decrypt_test"

    // ===== 旧代理地址（更新检查 / 指纹上报 / 风控上报等仍走这里） =====
    const val DECRYPT_PROXY_URL = "http://103.146.231.72:2238/decrypt_proxy.php"

    const val TARGET_PACKAGE = "com.shopee.tw"
    const val WAKE_ACTION = "com.oppo.wake.launch.action"

    const val CONTENT_PROVIDER_AUTHORITY = "com.yh.csyxgl.vivo.provider"
    const val CONTENT_PROVIDER_PATH_CURRENT = "current_fp"
    const val CONTENT_PROVIDER_PATH_FINGERPRINT_CURRENT = "fingerprint/current"
}
