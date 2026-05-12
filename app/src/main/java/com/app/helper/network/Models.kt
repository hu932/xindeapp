package com.app.helper.network

import com.google.gson.JsonElement
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
    @SerializedName("ok") val ok: JsonElement? = null,
    @SerializedName("code") val code: String = "",
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("data") val data: JsonElement? = null
) {
    val isApi1Success: Boolean
        get() {
            val submit = data?.takeIf { it.isJsonObject }?.asJsonObject ?: return false
            val submitCode = submit.get("code")?.takeIf { !it.isJsonNull }?.asString ?: ""
            val submitMsg = submit.get("msg")
            return code == "200" && submitCode == "SUCCESS" && (submitMsg == null || submitMsg.isJsonNull)
        }
    val isApi2Success: Boolean
        get() {
            val okValue = ok
            if (okValue != null && !okValue.isJsonNull) {
                if (okValue.isJsonPrimitive) {
                    val primitive = okValue.asJsonPrimitive
                    if (primitive.isBoolean && primitive.asBoolean) return true
                    if (primitive.isNumber && primitive.asInt == 1) return true
                    if (primitive.isString) {
                        val text = primitive.asString.trim()
                        if (text.equals("true", ignoreCase = true) || text == "1" || text.equals("SUCCESS", ignoreCase = true)) return true
                    }
                }
            }
            return code.equals("SUCCESS", ignoreCase = true) || code == "200"
        }
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
    @SerializedName("force") val force: Boolean = false,
    @SerializedName("blocked") val blocked: Boolean = false,
    @SerializedName("blocked_msg") val blockedMsg: String = ""
)
