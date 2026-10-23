package com.app.helper.ui.compose

import android.content.Intent
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.lifecycleScope
import com.app.helper.network.ApiService
import com.app.helper.util.TokenManager
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/**
 * Compose 版登录页。
 *
 * 行为与旧版一致：
 *   - 已登录直接进主界面
 *   - 顶部模式徽标每 3 秒轮询后台库存开关
 *   - 登录成功写 token / username / groupId / apiType / accountCount
 */
class ComposeLoginActivity : ComponentActivity() {

    private var accountId by mutableStateOf("")
    private var modeBadge by mutableStateOf(ModeBadge.DETECTING)
    private var loading by mutableStateOf(false)
    private var errorText by mutableStateOf("")

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (TokenManager.isLoggedIn()) {
            startMain()
            return
        }

        startModePolling()

        setContent {
            LoginScreen(
                accountId = accountId,
                onAccountIdChange = {
                    accountId = it
                    if (errorText.isNotBlank()) errorText = ""
                },
                modeBadge = modeBadge,
                loading = loading,
                errorText = errorText,
                onLogin = { performLogin() },
            )
        }
    }

    private fun startModePolling() {
        lifecycleScope.launch {
            while (isActive) {
                val inventory = runCatching { ApiService.refreshInventoryMode() }.getOrDefault(false)
                modeBadge = if (inventory) ModeBadge.INVENTORY else ModeBadge.NORMAL
                delay(3000)
            }
        }
    }

    private fun performLogin() {
        if (loading) return
        val id = accountId.trim()
        if (id.isEmpty()) {
            errorText = "请输入账号 / 下发id"
            return
        }
        loading = true
        errorText = ""
        lifecycleScope.launch {
            runCatching { ApiService.login(id) }
                .onSuccess { response ->
                    val data = response.data
                    if (response.isSuccess && data != null) {
                        TokenManager.saveToken(data.token)
                        TokenManager.saveUsername(data.accountId.ifBlank { id })
                        TokenManager.saveGroupId(data.groupId)
                        TokenManager.saveApiType(TokenManager.API_TYPE_1)
                        TokenManager.saveAccountCount(
                            if (data.api1TaskAccountCount > 0) data.api1TaskAccountCount
                            else data.taskAccountCount
                        )
                        loading = false
                        Toast.makeText(this@ComposeLoginActivity, "登录成功", Toast.LENGTH_SHORT).show()
                        startMain()
                        return@launch
                    }
                    loading = false
                    errorText = response.msg ?: "登录失败"
                }
                .onFailure { e ->
                    loading = false
                    errorText = "网络错误：${e.message ?: "未知"}"
                }
        }
    }

    private fun startMain() {
        startActivity(Intent(this, ComposeMainActivity::class.java))
        finish()
    }
}
