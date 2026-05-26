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

    suspend fun login(username: String, password: String): ApiResponse<LoginData> {
        return proxyPost(mapOf("act" to "api1_login", "username" to username, "password" to password))
    }

    suspend fun loginApi2(username: String, password: String): ApiResponse<LoginData> {
        return proxyPost(mapOf("act" to "api2_login", "username" to username, "password" to password))
    }

    suspend fun takeTask(): ApiResponse<TaskData> {
        return get(ApiConfig.TASK_TAKE_PATH)
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

    suspend fun heartbeat(
        username: String,
        groupId: String,
        apiType: Int,
        deviceId: String
    ): HeartbeatResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["act"] = "heartbeat"
        body["username"] = username
        body["group_id"] = groupId
        body["api_type"] = apiType
        body["device_id"] = deviceId
        addFingerprintFields(body)

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, HeartbeatResponse::class.java)
    }

    suspend fun submitTask(
        taskUrl: String,
        encryptedData: String,
        username: String,
        groupId: String,
        deviceId: String
    ): Api2UpResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["api_type"] = 1
        body["url"] = taskUrl
        body["data"] = encryptedData
        body["username"] = username
        body["group_id"] = groupId
        body["device_id"] = deviceId
        body["auth_token"] = TokenManager.getToken()

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2UpResponse::class.java)
    }

    suspend fun takeTask2(groupId: String): Api2TaskResponse = withContext(Dispatchers.IO) {
        val body = mapOf("info" to "台湾", "use" to groupId)
        val jsonBytes = gson.toJson(body).toByteArray(Charsets.UTF_8)
        val gzipped = gzipCompress(jsonBytes)

        val request = Request.Builder()
            .url(ApiConfig.BASE_URL_2 + ApiConfig.TASK_GET_PATH_2)
            .post(gzipped.toRequestBody("application/json; charset=utf-8".toMediaType()))
            .header("Content-Encoding", "gzip")
            .build()

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2TaskResponse::class.java)
    }

    suspend fun submitTask2(
        username: String,
        groupId: String,
        taskInfo: Api2TaskInfo,
        encryptedData: String,
        deviceId: String
    ): Api2UpResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["api_type"] = 2
        body["username"] = username
        body["group_id"] = groupId
        body["device_id"] = deviceId
        body["用户名"] = username
        body["组ID"] = groupId
        body["设备ID"] = deviceId
        body["任务数据"] = taskInfo
        body["data"] = encryptedData

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = proxyClient.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2UpResponse::class.java)
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

    private suspend inline fun <reified T> get(path: String): ApiResponse<T> = withContext(Dispatchers.IO) {
        val request = Request.Builder()
            .url(ApiConfig.BASE_URL + path)
            .get()
            .build()

        val response = HttpClient.client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        val type = TypeToken.getParameterized(ApiResponse::class.java, T::class.java).type
        gson.fromJson(json, type)
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
