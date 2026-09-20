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
        return proxyPost(mapOf("act" to "api1_login", "username" to accountId))
    }

    suspend fun takeTask(): ExtTakeResponse {
        return ajiePost<ExtTakeResponse>("task", ajieBody())
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
        val resp = ajiePost<HeartbeatExtResponse>("heartbeat", ajieBody())
        return HeartbeatResponse(
            ok = resp.success,
            allowed = resp.data?.allowed ?: true,
            msg = resp.msg ?: resp.error ?: "",
            deviceKey = resp.data?.deviceKey ?: ""
        )
    }

    suspend fun submitTask(
        taskId: String,
        taskPageUrl: String,
        submitUrl: String,
        rawResponse: String,
        task: ExtTask
    ): ExtSubmitResponse {
        val body = ajieBody().apply {
            put("deal_id", taskId)
            put("productUrl", taskPageUrl)
            put("url", submitUrl)
            put("data", rawResponse)
            put("success", true)
            put("api1_take_token", task.takeToken())
            put("task_account_key", task.accountKey())
            put("task_account_username", task.accountUsername())
            put("task_api_type", task.apiType())
            put("task_runtime_scope", task.runtimeScope())
        }
        return ajiePost<ExtSubmitResponse>("submit", body)
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
        "device_label" to "安卓设备",
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
