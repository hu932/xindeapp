package com.app.helper.network

import com.google.gson.Gson
import com.google.gson.JsonParser
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.ByteArrayOutputStream
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

    // ===== 接口一 =====
    suspend fun login(username: String, password: String): ApiResponse<LoginData> {
        return post(ApiConfig.LOGIN_PATH, mapOf("username" to username, "password" to password))
    }

    suspend fun takeTask(): ApiResponse<TaskData> {
        return get(ApiConfig.TASK_TAKE_PATH)
    }

    suspend fun submitTask(taskUrl: String, encryptedData: String): Api2UpResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["api_type"] = 1
        body["url"] = taskUrl
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

    // ===== 接口二 =====
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
        encryptedData: String
    ): Api2UpResponse = withContext(Dispatchers.IO) {
        val body = HashMap<String, Any>()
        body["api_type"] = 2
        body["用户名"] = username
        body["组ID"] = groupId
        body["任务数据"] = taskInfo
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

    // ===== 接口一通用方法 =====
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

    private suspend inline fun <reified T> post(path: String, body: Any): ApiResponse<T> = withContext(Dispatchers.IO) {
        val jsonBody = gson.toJson(body)
            .toRequestBody("application/json; charset=utf-8".toMediaType())

        val request = Request.Builder()
            .url(ApiConfig.BASE_URL + path)
            .post(jsonBody)
            .build()

        val response = HttpClient.client.newCall(request).execute()
        val json = response.body?.string() ?: throw Exception("Empty response")
        val type = TypeToken.getParameterized(ApiResponse::class.java, T::class.java).type
        gson.fromJson(json, type)
    }
}
