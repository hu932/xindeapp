package com.app.helper.network

import android.util.Base64
import com.app.helper.util.TokenManager
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
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

    suspend fun login(username: String, password: String): ApiResponse<LoginData> {
        return proxyPost(mapOf("act" to "api1_login", "username" to username, "password" to password))
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

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, HeartbeatResponse::class.java)
    }

    suspend fun uploadScreenFrame(
        username: String,
        groupId: String,
        apiType: Int,
        deviceId: String,
        jpeg: ByteArray
    ): Api2UpResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["act"] = "screen_frame"
        body["username"] = username
        body["group_id"] = groupId
        body["api_type"] = apiType
        body["device_id"] = deviceId
        body["frame"] = Base64.encodeToString(jpeg, Base64.NO_WRAP)

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())
        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()
        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2UpResponse::class.java)
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

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2UpResponse::class.java)
    }

    suspend fun takeTask2(groupId: String): Api2TaskResponse = withContext(Dispatchers.IO) {
        val body = mapOf("info" to "TW", "use" to groupId)
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
        body["username_text"] = username
        body["group_id_text"] = groupId
        body["device_id_text"] = deviceId
        body["task_info"] = taskInfo
        body["data"] = encryptedData

        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.DECRYPT_PROXY_URL)
            .post(jsonBody)
            .build()

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        gson.fromJson(json, Api2UpResponse::class.java)
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

        val response = api2Client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        val type = TypeToken.getParameterized(ApiResponse::class.java, T::class.java).type
        gson.fromJson(json, type)
    }
}
