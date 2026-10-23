package com.app.helper.ui.login

import android.content.Intent
import android.graphics.Color
import android.os.Bundle
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.app.helper.databinding.ActivityLoginBinding
import com.app.helper.network.ApiService
import com.app.helper.ui.compose.ComposeMainActivity
import com.app.helper.util.TokenManager
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class LoginActivity : AppCompatActivity() {

    private lateinit var binding: ActivityLoginBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (TokenManager.isLoggedIn()) {
            startMain()
            return
        }

        binding = ActivityLoginBinding.inflate(layoutInflater)
        setContentView(binding.root)

        refreshModeBadge()

        binding.btnLogin.setOnClickListener {
            val accountId = binding.etUsername.text.toString().trim()

            if (accountId.isEmpty()) {
                Toast.makeText(this, "请输入账号 / 下发id", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            performLogin(accountId)
        }
    }

    // 登录页顶部模式徽标：每 3 秒实时检测后台库存模式开关
    private fun refreshModeBadge() {
        setModeBadge("detecting")
        lifecycleScope.launch {
            while (isActive) {
                val inventory = try {
                    ApiService.refreshInventoryMode()
                } catch (_: Exception) {
                    false
                }
                setModeBadge(if (inventory) "inventory" else "normal")
                delay(3000)
            }
        }
    }

    private fun setModeBadge(state: String) {
        val tv = binding.tvMode
        when (state) {
            "inventory" -> {
                tv.text = "⚡ 库存模式"
                tv.setTextColor(Color.parseColor("#FFD54F"))
            }
            "normal" -> {
                tv.text = "✓ 普通模式"
                tv.setTextColor(Color.parseColor("#A5D6A7"))
            }
            else -> {
                tv.text = "正在检测后台模式..."
                tv.setTextColor(Color.parseColor("#C5CFFF"))
            }
        }
    }

    private fun performLogin(accountId: String) {
        binding.btnLogin.isEnabled = false
        binding.btnLogin.text = "登录中..."

        lifecycleScope.launch {
            try {
                val response = ApiService.login(accountId)
                if (response.isSuccess && response.data != null) {
                    TokenManager.saveToken(response.data.token)
                    TokenManager.saveUsername(response.data.accountId.ifBlank { accountId })
                    TokenManager.saveGroupId(response.data.groupId)
                    TokenManager.saveApiType(TokenManager.API_TYPE_1)
                    TokenManager.saveAccountCount(
                        if (response.data.api1TaskAccountCount > 0) response.data.api1TaskAccountCount else response.data.taskAccountCount
                    )
                    Toast.makeText(this@LoginActivity, "登录成功", Toast.LENGTH_SHORT).show()
                    startMain()
                    return@launch
                } else {
                    Toast.makeText(this@LoginActivity, response.msg ?: "登录失败", Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(this@LoginActivity, "网络错误：${e.message}", Toast.LENGTH_SHORT).show()
            }

            binding.btnLogin.isEnabled = true
            binding.btnLogin.text = "登 录"
        }
    }

    private fun startMain() {
        startActivity(Intent(this, ComposeMainActivity::class.java))
        finish()
    }
}
