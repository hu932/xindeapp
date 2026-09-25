package com.app.helper.network

import com.app.helper.App
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
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

    suspend fun login(accountId: String): ApiResponse<LoginData> {
        // 团队C：手机号直接登录，无需服务器验证
        return ApiResponse(
            code = "200",
            data = LoginData(
                token = "team-c-" + accountId,
                accountId = accountId,
                userLevel = 1,
                groupId = "team-c",
                taskAccountCount = 0,
                api1TaskAccountCount = 0
            )
        )
    }

    suspend fun takeTask(): ExtTakeResponse = withContext(Dispatchers.IO) {
        try {
            val phone = TokenManager.getUsername()
            val url = ApiConfig.TEAM_BASE_URL + ApiConfig.TEAM_TASK_TAKE_PATH +
                "?phone=" + URLEncoder.encode(phone, "UTF-8")
            val request = Request.Builder()
                .url(url)
                .get()
                .build()

            val response = proxyClient.newCall(request).execute()
            val json = response.body?.string()?.trim().orEmpty()
            if (json.isEmpty() || json == "null") {
                return@withContext ExtTakeResponse(success = false, msg = "暂无任务")
            }

            // 手动解析，字段类型/缺失都容错，避免 Gson 因类型不匹配返回 null 导致 NPE
            val obj = JSONObject(json)
            val success = obj.optBoolean("success", false)
            val taskObj = obj.optJSONObject("task")
            val dataObj = taskObj?.optJSONObject("data")
            if (success && dataObj != null) {
                val shopId = dataObj.optString("shop_id", "")
                val goodId = dataObj.optString("good_id", "")
                val traceId = dataObj.optString("trace_id", "").ifBlank { dataObj.optString("id", "") }
                val country = dataObj.optString("country", "")
                val type = taskObj.optString("type", "").ifBlank { dataObj.optString("type", "") }
                val taskUrl = "https://shopee.tw/product/$shopId/$goodId"
                ExtTakeResponse(
                    success = true,
                    task = ExtTask(
                        taskId = traceId,
                        url = taskUrl,
                        taskUrl = taskUrl,
                        country = country,
                        taskType = type,
                        rawType = type,
                        shopId = shopId,
                        goodId = goodId,
                        traceId = traceId
                    )
                )
            } else {
                ExtTakeResponse(success = false, msg = "暂无任务")
            }
        } catch (e: Exception) {
            ExtTakeResponse(success = false, error = e.message ?: "网络错误")
        }
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
            // 1) 先解密：把加密数据发到后台解密接口，拿到商品 JSON
            val decrypted = decryptSubmitData(rawResponse)
            if (decrypted.isBlank()) {
                return@withContext ExtSubmitResponse(success = false, error = "解密失败：未拿到商品数据")
            }

            // 2) 组装上传 body：content 为商品 JSON 转对象
            val shopId = task.shopId.ifBlank { task.parseShopId().toString() }
            val goodId = task.goodId.ifBlank { task.parseItemId().toString() }
            val traceId = task.traceId.ifBlank { task.id() }
            val country = task.country.ifBlank { "tw" }
            val type = task.taskType.ifBlank { task.rawType.ifBlank { "pdp" } }

            val body = JSONObject()
            body.put("type", type)
            val taskObj = JSONObject()
            taskObj.put("shop_id", shopId)
            taskObj.put("good_id", goodId)
            taskObj.put("country", country)
            taskObj.put("trace_id", traceId)
            taskObj.put("content", JSONObject(decrypted))
            taskObj.put("phone", TokenManager.getUsername())
            body.put("task", taskObj)

            val jsonBody = body.toString()
                .toRequestBody("application/json; charset=utf-8".toMediaType())
            val request = Request.Builder()
                .url(ApiConfig.TEAM_BASE_URL + ApiConfig.TEAM_TASK_UPLOAD_PATH)
                .post(jsonBody)
                .build()

            val response = proxyClient.newCall(request).execute()
            val json = response.body?.string() ?: throw Exception("Empty response")
            val resp = gson.fromJson(json, TeamUploadResponse::class.java)
            if (resp != null && (resp.success || resp.code == 200)) {
                ExtSubmitResponse(success = true, code = resp.code.toString())
            } else {
                ExtSubmitResponse(success = false, error = resp?.msg ?: "上传失败")
            }
        } catch (e: Exception) {
            ExtSubmitResponse(success = false, error = e.message ?: "上传异常")
        }
    }

    /**
     * 调用后台解密接口（?act=decrypt_test），返回解密并提取后的商品 JSON。
     * 优先 submit_result（与回传链路一致的最终结果），其次 aes_extracted / plain_extracted。
     */
    suspend fun decryptSubmitData(encrypted: String): String = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject()
            body.put("data", encrypted)
            val jsonBody = body.toString()
                .toRequestBody("application/json; charset=utf-8".toMediaType())
            val request = Request.Builder()
                .url(ApiConfig.DECRYPT_URL)
                .post(jsonBody)
                .build()

            val response = proxyClient.newCall(request).execute()
            val json = response.body?.string() ?: return@withContext ""
            val obj = JSONObject(json)
            obj.optString("submit_result")
                .ifBlank { obj.optString("aes_extracted") }
                .ifBlank { obj.optString("plain_extracted") }
        } catch (_: Exception) {
            ""
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

    private suspend inline fun <reified T> ajiePost(path: String, body: Map<String, Any>): T = withContext(Dispatchers.IO) {
        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL + "?ajie_api=" + path)
            .post(jsonBody)
            .build()

        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, T::class.java)
    }

    private suspend inline fun <reified T> proxyPost(body: Any): ApiResponse<T> = withContext(Dispatchers.IO) {
        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        val type = TypeToken.getParameterized(ApiResponse::class.java, T::class.java).type
        gson.fromJson(json, type)
    }
}
