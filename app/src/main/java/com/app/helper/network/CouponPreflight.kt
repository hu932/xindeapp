package com.app.helper.network

import android.content.Context
import com.app.helper.App
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.time.LocalDate
import java.time.ZoneId
import java.util.concurrent.TimeUnit

/**
 * 优惠券预检（与网页插件 coupon-preflight.js 保持一致）。
 * 流程：领取任务前先调 start，若要求检测则打开检测商品页抓包，
 * 再把抓到的原始数据上报 report，通过后当日不再重复检测。
 */
object CouponPreflight {

    data class Probe(val id: String, val url: String, val shopId: String, val itemId: String)

    data class StartResult(
        val allowed: Boolean = false,
        val required: Boolean = false,
        val dayKey: String = "",
        val probe: Probe? = null
    )

    data class ReportResult(
        val passed: Boolean = false,
        val reason: String = "",
        val dayKey: String = ""
    )

    private const val PREFS_NAME = "coupon_preflight_state"
    private const val KEY_USER_ID = "user_id"
    private const val KEY_DAY_KEY = "day_key"
    private const val KEY_SHOPEE_USER_ID = "shopee_user_id"
    private const val KEY_SHOPEE_USERNAME = "shopee_username"

    private val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(20, TimeUnit.SECONDS)
            .writeTimeout(20, TimeUnit.SECONDS)
            .build()
    }

    private fun baseUrl(): String = ApiConfig.COUPON_PREFLIGHT_BASE_URL.trimEnd('/')

    private suspend fun post(path: String, body: JSONObject): JSONObject = withContext(Dispatchers.IO) {
        val jsonBody = body.toString().toRequestBody("application/json; charset=utf-8".toMediaType())
        val request = Request.Builder()
            .url(baseUrl() + path)
            .header("Authorization", "Bearer ${ApiConfig.COUPON_PREFLIGHT_TOKEN}")
            .header("Accept", "application/json")
            .post(jsonBody)
            .build()
        client.newCall(request).execute().use { response ->
            val text = response.body?.string() ?: throw Exception("优惠券预检响应为空")
            val json = JSONObject(text)
            if (!json.optBoolean("success", false)) {
                throw Exception(json.optString("error", json.optString("msg", "优惠券预检请求失败")))
            }
            json.optJSONObject("data") ?: JSONObject()
        }
    }

    suspend fun start(userId: String, username: String): StartResult {
        val body = JSONObject()
        body.put("shopeeUserId", userId)
        body.put("shopeeUsername", username)
        val data = post("/api/coupon-preflight/start/", body)
        val probeObj = data.optJSONObject("probe")
        return StartResult(
            allowed = data.optBoolean("allowed", false),
            required = data.optBoolean("required", false),
            dayKey = data.optString("dayKey", dayKey()),
            probe = probeObj?.let {
                Probe(
                    id = it.optString("id", ""),
                    url = it.optString("url", ""),
                    shopId = it.optString("shopId", ""),
                    itemId = it.optString("itemId", "")
                )
            }
        )
    }

    suspend fun report(userId: String, username: String, probeId: String, rawResponseText: String): ReportResult {
        val body = JSONObject()
        body.put("shopeeUserId", userId)
        body.put("shopeeUsername", username)
        body.put("probeId", probeId)
        body.put("status", "captured")
        body.put("rawResponseText", rawResponseText)
        val data = post("/api/coupon-preflight/report/", body)
        return ReportResult(
            passed = data.optBoolean("passed", false),
            reason = data.optString("reason", ""),
            dayKey = data.optString("dayKey", dayKey())
        )
    }

    fun dayKey(): String = LocalDate.now(ZoneId.of("Asia/Shanghai")).toString()

    fun isPassedToday(userId: String): Boolean {
        if (userId.isBlank()) return false
        val p = App.context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return p.getString(KEY_USER_ID, "") == userId && p.getString(KEY_DAY_KEY, "") == dayKey()
    }

    fun markPassed(userId: String, key: String) {
        App.context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_USER_ID, userId)
            .putString(KEY_DAY_KEY, key.ifBlank { dayKey() })
            .apply()
    }

    /**
     * 从环境 App 回传的抓包内容里提取登录账号。
     * 回传结构：{"response_body": "<Shopee API 响应 JSON>"}，其中顶层带 userid 字段。
     */
    fun extractAccount(rawContent: String): Pair<String, String>? {
        if (rawContent.isBlank()) return null
        return try {
            val outer = JSONObject(rawContent)
            val bodyStr = outer.optString("response_body", "")
                .ifBlank { outer.optString("responseBody", "") }
                .ifBlank { rawContent }
            val body = JSONObject(bodyStr)
            val data = body.optJSONObject("data")
            val userId = listOf(
                body.optString("userid"),
                body.optString("user_id"),
                body.optString("userId"),
                data?.optString("userid") ?: "",
                data?.optString("user_id") ?: "",
                data?.optString("userId") ?: ""
            ).firstOrNull { it.isNotBlank() } ?: return null
            val username = listOf(
                body.optString("username"),
                body.optString("user_name"),
                body.optString("userName"),
                data?.optString("username") ?: "",
                data?.optString("user_name") ?: ""
            ).firstOrNull { it.isNotBlank() } ?: ""
            Pair(userId, username)
        } catch (_: Exception) {
            null
        }
    }

    fun saveAccount(userId: String, username: String) {
        App.context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_SHOPEE_USER_ID, userId)
            .putString(KEY_SHOPEE_USERNAME, username)
            .apply()
    }

    fun getAccount(): Pair<String, String>? {
        val p = App.context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val userId = p.getString(KEY_SHOPEE_USER_ID, "") ?: ""
        if (userId.isBlank()) return null
        return Pair(userId, p.getString(KEY_SHOPEE_USERNAME, "") ?: "")
    }
}
