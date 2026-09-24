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
    @SerializedName("userLevel") val userLevel: Int = 0,
    @SerializedName("groupId") val groupId: String = "",
    @SerializedName("accountId") val accountId: String = "",
    @SerializedName("task_account_count") val taskAccountCount: Int = 0,
    @SerializedName("api1_task_account_count") val api1TaskAccountCount: Int = 0
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
    @SerializedName("device_key") val deviceKey: String = "",
    @SerializedName("risk_control") val riskControl: Boolean = false,
    @SerializedName("risk_msg") val riskMsg: String = ""
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

data class ExtTask(
    @SerializedName("taskId") val taskId: String = "",
    @SerializedName("deal_id") val dealId: String = "",
    @SerializedName("url") val url: String = "",
    @SerializedName("taskUrl") val taskUrl: String = "",
    @SerializedName("api1TakeToken") val api1TakeToken: String = "",
    @SerializedName("api1_take_token") val api1TakeTokenSnake: String = "",
    @SerializedName("taskAccountKey") val taskAccountKey: String = "",
    @SerializedName("task_account_key") val taskAccountKeySnake: String = "",
    @SerializedName("taskAccountUsername") val taskAccountUsername: String = "",
    @SerializedName("task_account_username") val taskAccountUsernameSnake: String = "",
    @SerializedName("taskApiType") val taskApiType: Int = 0,
    @SerializedName("task_api_type") val taskApiTypeSnake: Int = 0,
    @SerializedName("taskRuntimeScope") val taskRuntimeScope: String = "",
    @SerializedName("task_runtime_scope") val taskRuntimeScopeSnake: String = ""
) {
    fun id(): String = taskId.ifBlank { dealId }
    fun pageUrl(): String = url.ifBlank { taskUrl }
    fun takeToken(): String = api1TakeToken.ifBlank { api1TakeTokenSnake }
    fun accountKey(): String = taskAccountKey.ifBlank { taskAccountKeySnake }
    fun accountUsername(): String = taskAccountUsername.ifBlank { taskAccountUsernameSnake }
    fun apiType(): Int = if (taskApiType in 1..2) taskApiType else taskApiTypeSnake
    fun runtimeScope(): String = taskRuntimeScope.ifBlank { taskRuntimeScopeSnake }
    fun parseShopId(): Long {
        val parts = pageUrl().trimEnd('/').split("/")
        return if (parts.size >= 2) parts[parts.size - 2].toLongOrNull() ?: 0L else 0L
    }
    fun parseItemId(): Long = pageUrl().trimEnd('/').split("/").lastOrNull()?.toLongOrNull() ?: 0L
}

data class ExtTakeResponse(
    @SerializedName("success") val success: Boolean = false,
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("error") val error: String? = null,
    @SerializedName("retry_after_seconds") val retryAfterSeconds: Int = 0,
    @SerializedName("task") val task: ExtTask? = null
) {
    val isSuccess: Boolean get() = success && task != null
    val message: String get() = error ?: msg ?: "获取任务失败"
}

data class ExtSubmitResponse(
    @SerializedName("success") val success: Boolean = false,
    @SerializedName("ok") val ok: Boolean = false,
    @SerializedName("code") val code: String = "",
    @SerializedName("status") val status: String = "",
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("error") val error: String? = null
) {
    val isSuccess: Boolean get() = success || ok || code == "200" || status == "success"
    val message: String get() = error ?: msg ?: "提交失败"
}

data class HeartbeatExtData(
    @SerializedName("allowed") val allowed: Boolean = true,
    @SerializedName("device_key") val deviceKey: String = "",
    @SerializedName("risk_control") val riskControl: Boolean = false,
    @SerializedName("risk_msg") val riskMsg: String = ""
)

data class HeartbeatExtResponse(
    @SerializedName("success") val success: Boolean = false,
    @SerializedName("msg") val msg: String? = null,
    @SerializedName("error") val error: String? = null,
    @SerializedName("data") val data: HeartbeatExtData? = null
)
