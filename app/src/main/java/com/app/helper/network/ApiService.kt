package com.app.helper.network

import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

object ApiService {

    private val gson = Gson()

    suspend fun login(username: String, password: String): ApiResponse<LoginData> {
        return post(ApiConfig.LOGIN_PATH, mapOf("username" to username, "password" to password))
    }

    suspend fun takeTask(): ApiResponse<TaskData> {
        return get(ApiConfig.TASK_TAKE_PATH)
    }

    suspend fun submitTask(taskUrl: String, responseBody: String): ApiResponse<SubmitResult> {
        val body = mapOf(
            "appVersion" to "vv2",
            "url" to taskUrl,
            "result" to responseBody
        )
        return post(ApiConfig.TASK_SUBMIT_PATH, body)
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
