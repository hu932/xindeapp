package com.app.helper.network

import com.google.gson.annotations.SerializedName

data class ApiResponse<T>(
    @SerializedName("code") val code: String = "",
    @SerializedName("data") val data: T? = null,
    @SerializedName("msg") val msg: String? = null
) {
    val isSuccess: Boolean get() = code == "200"
}

data class LoginData(
    @SerializedName("token") val token: String = "",
    @SerializedName("userLevel") val userLevel: Int = 0
)

data class TaskData(
    @SerializedName("taskId") val taskId: String = "",
    @SerializedName("taskUrl") val taskUrl: String = "",
    @SerializedName("expireTime") val expireTime: String = "",
    @SerializedName("releaseTime") val releaseTime: String = "",
    @SerializedName("takeTime") val takeTime: String = "",
    @SerializedName("platformInfo") val platformInfo: String = ""
) {
    fun parseShopId(): Long {
        val parts = taskUrl.trimEnd('/').split("/")
        return if (parts.size >= 2) parts[parts.size - 2].toLongOrNull() ?: 0 else 0
    }

    fun parseItemId(): Long {
        val parts = taskUrl.trimEnd('/').split("/")
        return parts.lastOrNull()?.toLongOrNull() ?: 0
    }
}

data class SubmitResult(
    @SerializedName("code") val code: String = "",
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("taskId") val taskId: String = ""
) {
    val isSuccess: Boolean get() = code == "SUCCESS"
}

// ===== 接口二数据模型 =====
data class Api2TaskResponse(
    @SerializedName("ok") val ok: Boolean = false,
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("TaskData") val taskData: Api2TaskData? = null
)

data class Api2TaskData(
    @SerializedName("Url") val url: String = "",
    @SerializedName("Task") val task: Api2TaskInfo? = null
) {
    fun parseShopId(): Long {
        val parts = url.trimEnd('/').split("/")
        return if (parts.size >= 2) parts[parts.size - 2].toLongOrNull() ?: 0 else 0
    }
    fun parseItemId(): Long {
        val parts = url.trimEnd('/').split("/")
        return parts.lastOrNull()?.toLongOrNull() ?: 0
    }
}

data class Api2TaskInfo(
    @SerializedName("ID") val id: String = "",
    @SerializedName("TaskNumber") val taskNumber: String = "",
    @SerializedName("TaskType") val taskType: String = ""
)

data class Api2UpResponse(
    @SerializedName("ok") val ok: Boolean = false,
    @SerializedName("code") val code: String = "",
    @SerializedName("msg") val msg: String? = null
) {
    val isSuccess: Boolean get() = ok || code == "200" || code.equals("SUCCESS", ignoreCase = true)
}

data class HeartbeatResponse(
    @SerializedName("ok") val ok: Boolean = false,
    @SerializedName("allowed") val allowed: Boolean = true,
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("device_key") val deviceKey: String = ""
)

data class UpdateResponse(
    @SerializedName("ok") val ok: Boolean = false,
    @SerializedName("has_update") val hasUpdate: Boolean = false,
    @SerializedName("version_code") val versionCode: Int = 0,
    @SerializedName("version_name") val versionName: String = "",
    @SerializedName("apk_url") val apkUrl: String = "",
    @SerializedName("title") val title: String = "发现新版本",
    @SerializedName("message") val message: String = "",
    @SerializedName("force") val force: Boolean = false
)
