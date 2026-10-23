package com.app.helper.network

import com.app.helper.App
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import com.google.gson.Gson
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.net.URLEncoder
import java.util.concurrent.TimeUnit
import java.util.zip.GZIPOutputStream

object ApiService {

    private val gson = Gson()

    private val api2Client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .writeTimeout(15, TimeUnit.SECONDS)
            .build()
    }

    private val proxyClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(20, TimeUnit.SECONDS)
            .readTimeout(75, TimeUnit.SECONDS)
            .writeTimeout(75, TimeUnit.SECONDS)
            .build()
    }

    // 登录：先检测后台库存模式开关，再决定走哪套接口
    suspend fun login(accountId: String): ApiResponse<LoginData> = withContext(Dispatchers.IO) {
        try {
            val inventory = refreshInventoryMode()
            if (inventory) {
                // 库存模式：账号即「下发id」，无需服务端鉴权
                val token = "ajie-inv-" + accountId
                ApiResponse(
                    code = "200",
                    data = LoginData(
                        token = token,
                        accountId = accountId,
                        userLevel = 1,
                        groupId = "inventory",
                        taskAccountCount = 0,
                        api1TaskAccountCount = 0
                    )
                )
            } else {
                // 普通模式：走解码插件 /api/login 获取插件 token
                val body = ajieBody()
                body["accountId"] = accountId
                val jsonBody = gson.toJson(body)
                    .toRequestBody("application/json; charset=utf-8".toMediaType())
                val request = Request.Builder()
                    .url(ApiConfig.GO_BASE_URL + ApiConfig.API_LOGIN_PATH)
                    .post(jsonBody)
                    .build()
                val response = proxyClient.newCall(request).execute()
                val json = response.body?.string() ?: throw Exception("登录失败：空响应")
                val obj = JSONObject(json)
                if (!obj.optBoolean("success", false)) {
                    throw Exception(obj.optString("error", obj.optString("msg", "登录失败")))
                }
                val data = obj.optJSONObject("data")
                val token = data?.optString("token", "") ?: ""
                if (token.isBlank()) throw Exception("登录失败：未获取到 token")
                ApiResponse(
                    code = "200",
                    data = LoginData(
                        token = token,
                        accountId = data.optString("accountId", accountId),
                        userLevel = 1,
                        groupId = data.optString("groupId", "").ifBlank { "ajie" },
                        taskAccountCount = data.optInt("task_account_count", 0),
                        api1TaskAccountCount = data.optInt("api1_task_account_count", 0)
                    )
                )
            }
        } catch (e: Exception) {
            ApiResponse(code = "0", msg = e.message ?: "网络错误")
        }
    }

    suspend fun takeTask(): ExtTakeResponse = withContext(Dispatchers.IO) {
        try {
            if (currentInventoryMode()) {
                val inv = takeInventoryTask()
                // 若后台库存模式刚被关闭（返回“未开启”），立即切回普通模式重试一次
                if (!inv.isSuccess && inv.message.contains("未开启")) {
                    TokenManager.saveInventoryMode(false)
                    return@withContext takeNormalTask()
                }
                inv
            } else {
                takeNormalTask()
            }
        } catch (e: Exception) {
            ExtTakeResponse(success = false, error = e.message ?: "网络错误")
        }
    }

    // 库存模式：GET /inventory/task?account_id=xxx，从全局库存弹一个任务
    private fun takeInventoryTask(): ExtTakeResponse {
        val accountId = TokenManager.getUsername()
        val url = ApiConfig.GO_BASE_URL + ApiConfig.INVENTORY_TASK_PATH +
            "?account_id=" + URLEncoder.encode(accountId, "UTF-8")
        val request = Request.Builder().url(url).get().build()
        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string()?.trim().orEmpty()
        if (json.isEmpty() || json == "null") return ExtTakeResponse(success = false, msg = "暂无任务")
        val obj = JSONObject(json)
        val ok = obj.optBoolean("success", false) || obj.optInt("code", 0) == 200
        val taskObj = obj.optJSONObject("task")
        if (!ok || taskObj == null) {
            return ExtTakeResponse(success = false, msg = obj.optString("msg", "暂无任务"))
        }
        return ExtTakeResponse(success = true, task = extTaskFromJson(taskObj))
    }

    // 普通模式：POST /api/task 领取解码插件任务
    private fun takeNormalTask(): ExtTakeResponse {
        val jsonBody = gson.toJson(ajieBody())
            .toRequestBody("application/json; charset=utf-8".toMediaType())
        val request = Request.Builder()
            .url(ApiConfig.GO_BASE_URL + ApiConfig.API_TASK_PATH)
            .post(jsonBody)
            .build()
        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string()?.trim().orEmpty()
        if (json.isEmpty() || json == "null") return ExtTakeResponse(success = false, msg = "暂无任务")
        val obj = JSONObject(json)
        if (!obj.optBoolean("success", false)) {
            return ExtTakeResponse(success = false, msg = obj.optString("error", obj.optString("msg", "暂无任务")))
        }
        val taskObj = obj.optJSONObject("task")
        if (taskObj == null) return ExtTakeResponse(success = false, msg = "暂无任务")
        return ExtTakeResponse(success = true, task = extTaskFromJson(taskObj))
    }

    // 把后台返回的任务 JSON 映射为 ExtTask（优先 taskUrl / url / trace_id / taskId）
    private fun extTaskFromJson(taskObj: JSONObject): ExtTask {
        val taskUrl = taskObj.optString("taskUrl", "")
            .ifBlank { taskObj.optString("url", "") }
            .ifBlank { taskObj.optString("productUrl", "") }
        val traceId = taskObj.optString("trace_id", "")
            .ifBlank { taskObj.optString("taskId", "") }
            .ifBlank { taskObj.optString("deal_id", "") }
            .ifBlank { taskObj.optString("id", "") }
        val shopId = taskObj.optString("shopId", "").ifBlank { taskObj.optString("shop_id", "") }
        val goodId = taskObj.optString("goodId", "").ifBlank { taskObj.optString("good_id", "") }
        val runtimeScope = taskObj.optString("taskRuntimeScope", "")
            .ifBlank { taskObj.optString("task_runtime_scope", "") }
        return ExtTask(
            taskId = traceId,
            dealId = taskObj.optString("deal_id", traceId),
            url = taskUrl,
            taskUrl = taskUrl,
            country = taskObj.optString("country", ""),
            taskType = taskObj.optString("type", ""),
            rawType = taskObj.optString("type", ""),
            shopId = shopId,
            goodId = goodId,
            traceId = traceId,
            taskRuntimeScope = runtimeScope
        )
    }

    suspend fun checkUpdate(versionCode: Int): UpdateResponse = withContext(Dispatchers.IO) {
        val url = ApiConfig.DECRYPT_PROXY_URL +
            "?act=check_update&version_code=" +
            URLEncoder.encode(versionCode.toString(), "UTF-8")
        val request = Request.Builder()
            .url(url)
            .get()
            .build()

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, UpdateResponse::class.java)
    }

    suspend fun heartbeat(): HeartbeatResponse {
        // 团队C：手机号本地登录，无账号白名单，心跳直接放行（不再走旧代理账号校验，
        // 否则手机号不在旧账号列表里会返回 account not allowed 导致设备被禁用）。
        return HeartbeatResponse(
            ok = true,
            allowed = true,
            msg = "",
            deviceKey = "",
            riskControl = false,
            riskMsg = ""
        )
    }

    suspend fun submitTask(
        taskId: String,
        taskPageUrl: String,
        submitUrl: String,
        rawResponse: String,
        task: ExtTask
    ): ExtSubmitResponse = withContext(Dispatchers.IO) {
        try {
            // 后台在回传时统一解密（库存 / 普通两种模式都支持 data 原文，无需本地解密）
            if (currentInventoryMode()) {
                val body = JSONObject()
                body.put("account_id", TokenManager.getUsername())
                body.put("trace_id", taskId.ifBlank { task.traceId.ifBlank { task.id() } })
                body.put("data", rawResponse)
                val jsonBody = body.toString()
                    .toRequestBody("application/json; charset=utf-8".toMediaType())
                val request = Request.Builder()
                    .url(ApiConfig.GO_BASE_URL + ApiConfig.INVENTORY_SUBMIT_PATH)
                    .post(jsonBody)
                    .build()
                val response = proxyClient.newCall(request).execute()
                val json = response.body?.string() ?: throw Exception("Empty response")
                val obj = JSONObject(json)
                if (obj.optBoolean("success", false) || obj.optInt("code", 0) == 200) {
                    ExtSubmitResponse(success = true, code = obj.optInt("code", 200).toString())
                } else {
                    ExtSubmitResponse(success = false, error = obj.optString("msg", "上传失败"))
                }
            } else {
                // 普通模式：带上与领取一致的设备上下文 + taskRuntimeScope，才能匹配到租约
                val body = ajieBody()
                body["deal_id"] = taskId.ifBlank { task.id() }
                body["task_id"] = taskId.ifBlank { task.id() }
                body["url"] = taskPageUrl.ifBlank { task.pageUrl() }
                body["taskRuntimeScope"] = task.runtimeScope()
                body["task_runtime_scope"] = task.runtimeScope()
                body["data"] = rawResponse
                val jsonBody = gson.toJson(body)
                    .toRequestBody("application/json; charset=utf-8".toMediaType())
                val request = Request.Builder()
                    .url(ApiConfig.GO_BASE_URL + ApiConfig.API_SUBMIT_PATH)
                    .post(jsonBody)
                    .build()
                val response = proxyClient.newCall(request).execute()
                val json = response.body?.string() ?: throw Exception("Empty response")
                val obj = JSONObject(json)
                if (obj.optBoolean("success", false)) {
                    ExtSubmitResponse(success = true, code = "200")
                } else {
                    ExtSubmitResponse(success = false, error = obj.optString("error", obj.optString("msg", "上传失败")))
                }
            }
        } catch (e: Exception) {
            ExtSubmitResponse(success = false, error = e.message ?: "上传异常")
        }
    }

    suspend fun reportFingerprintEvent(event: String, fpName: String, fpJson: String) = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["act"] = "fingerprint_event"
        body["event"] = event
        body["username"] = TokenManager.getUsername()
        body["group_id"] = TokenManager.getGroupId()
        body["api_type"] = TokenManager.getApiType()
        body["event_count"] = 1
        addFingerprintFields(body, fpName, fpJson)

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        proxyClient.newCall(request).execute().close()
    }

    suspend fun reportDeviceRisk(reason: String) = withContext(Dispatchers.IO) {
        try {
            val body = HashMap<String, Any>()
            body["act"] = "device_risk"
            body["username"] = TokenManager.getUsername()
            body["device_id"] = deviceId()
            body["reason"] = reason
            val jsonBody = gson.toJson(body)
                .toRequestBody("application/json; charset=utf-8".toMediaType())
            val request = Request.Builder()
                .url(ApiConfig.DECRYPT_PROXY_URL)
                .post(jsonBody)
                .build()
            proxyClient.newCall(request).execute().close()
        } catch (_: Exception) {
        }
    }

    private fun addFingerprintFields(body: MutableMap<String, Any>) {
        try {
            val activeName = FingerprintManager.getActiveFpName(App.context).orEmpty()
            val raw = FingerprintManager.loadCurrentFp(App.context)
            addFingerprintFields(body, activeName, raw)
        } catch (_: Exception) {
        }
    }

    private fun addFingerprintFields(body: MutableMap<String, Any>, sourceName: String, sourceRaw: String?) {
        try {
            var fpName = sourceName
            var fpId = ""

            if (!sourceRaw.isNullOrEmpty()) {
                val json = JSONObject(sourceRaw)
                fpId = json.optString("androidId", "")
                if (fpName.isBlank()) fpName = json.optString("name", "")
            }

            if (fpName.isNotBlank()) body["fingerprint_name"] = fpName
            if (fpId.isNotBlank()) body["fingerprint_id"] = fpId
            body["fingerprint_key"] = when {
                fpId.isNotBlank() -> fpId
                fpName.isNotBlank() -> fpName
                else -> body["device_id"]?.toString().orEmpty()
            }
        } catch (_: Exception) {
        }
    }

    private fun gzipCompress(data: ByteArray): ByteArray {
        val bos = ByteArrayOutputStream()
        GZIPOutputStream(bos).use { it.write(data) }
        return bos.toByteArray()
    }

    private fun deviceId(): String {
        val androidId = android.provider.Settings.Secure.getString(
            App.context.contentResolver, android.provider.Settings.Secure.ANDROID_ID)
        return androidId?.takeIf { it.isNotBlank() } ?: "android-${android.os.Build.MODEL}"
    }

    private fun ajieBody(): HashMap<String, Any> = hashMapOf(
        "accountId" to TokenManager.getUsername(),
        "token" to TokenManager.getToken(),
        "client" to "ajie-extension",
        "source" to "ajie",
        "platform" to "android",
        "device_type" to "android",
        "device_label" to (TokenManager.getDeviceName().ifBlank { "安卓设备" }),
        "device_id" to deviceId()
    )

    // 实时检测后台库存模式开关（不缓存，每次任务周期都拉最新状态）
    suspend fun refreshInventoryMode(): Boolean = withContext(Dispatchers.IO) {
        try {
            val request = Request.Builder()
                .url(ApiConfig.GO_BASE_URL + ApiConfig.INVENTORY_MODE_PATH)
                .get()
                .build()
            val response = api2Client.newCall(request).execute()
            val json = response.body?.string()
            val obj = if (json.isNullOrBlank()) null else JSONObject(json)
            val enabled = obj?.optBoolean("enabled", false) ?: TokenManager.getInventoryMode()
            TokenManager.saveInventoryMode(enabled)
            enabled
        } catch (_: Exception) {
            TokenManager.getInventoryMode()
        }
    }

    private suspend fun currentInventoryMode(): Boolean = refreshInventoryMode()

    /**
     * 从服务器领取一份指纹环境。
     *
     * 服务端负责生成身份字段（androidId / gsf / gaid / MAC / 机型），
     * 并按账号做每日额度控制。客户端不再本地随机生成。
     *
     * 返回 null 表示领取失败（未授权、额度用完、网络异常）。
     */
    suspend fun fetchFingerprintEnv(): JSONObject? = withContext(Dispatchers.IO) {
        try {
            val token = TokenManager.getToken()
            val url = ApiConfig.GO_BASE_URL + ApiConfig.FP_ENV_GENERATE_PATH +
                    "?token=" + java.net.URLEncoder.encode(token, "UTF-8")
            val request = Request.Builder()
                .url(url)
                .get()
                .apply { if (token.isNotBlank()) addHeader("Authorization", "Bearer $token") }
                .build()
            val response = api2Client.newCall(request).execute()
            val body = response.body?.string()
            if (body.isNullOrBlank()) return@withContext null
            val root = JSONObject(body)
            if (!root.optBoolean("success", false)) return@withContext null
            val data = root.optJSONObject("data") ?: return@withContext null
            data.optJSONObject("fp")
        } catch (_: Exception) {
            null
        }
    }

}
