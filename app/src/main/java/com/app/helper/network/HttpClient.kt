package com.app.helper.network

import com.app.helper.util.TokenManager
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import java.security.cert.X509Certificate
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManager
import javax.net.ssl.X509TrustManager

object HttpClient {

    val client: OkHttpClient by lazy {
        val trustManager = object : X509TrustManager {
            override fun checkClientTrusted(chain: Array<out X509Certificate>?, authType: String?) {}
            override fun checkServerTrusted(chain: Array<out X509Certificate>?, authType: String?) {}
            override fun getAcceptedIssuers(): Array<X509Certificate> = arrayOf()
        }

        val sslContext = SSLContext.getInstance("TLS").apply {
            init(null, arrayOf<TrustManager>(trustManager), java.security.SecureRandom())
        }

        // 日志：BODY 会把每个响应的完整报文格式化输出，
        // 抓任务时每秒几十个请求、单条几十~几百 KB，直接把队列拖慢（界面卡顿的主因）。
        // 这里改成 BASIC；只有 debug 包才打 BODY。
        val logging = HttpLoggingInterceptor().apply {
            level = if (com.app.helper.App.isDebug) {
                HttpLoggingInterceptor.Level.BASIC
            } else {
                HttpLoggingInterceptor.Level.NONE
            }
        }

        OkHttpClient.Builder()
            .sslSocketFactory(sslContext.socketFactory, trustManager)
            .hostnameVerifier { _, _ -> true }
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(20, TimeUnit.SECONDS)
            .writeTimeout(15, TimeUnit.SECONDS)
            .retryOnConnectionFailure(true)
            .connectionPool(okhttp3.ConnectionPool(8, 5, TimeUnit.MINUTES))
            .addInterceptor { chain ->
                val original = chain.request()
                val builder = original.newBuilder()
                    .header("Accept", "application/json, text/plain, */*")
                    .header("Content-Type", "application/json")

                val token = TokenManager.getToken()
                if (token.isNotEmpty()) {
                    builder.header("Authorization", "Bearer $token")
                }

                chain.proceed(builder.build())
            }
            .addInterceptor(logging)
            .build()
    }
}
