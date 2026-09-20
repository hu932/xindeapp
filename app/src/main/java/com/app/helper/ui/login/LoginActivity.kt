package com.app.helper.ui.login

import android.content.Intent
import android.os.Bundle
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.app.helper.databinding.ActivityLoginBinding
import com.app.helper.network.ApiService
import com.app.helper.ui.main.MainActivity
import com.app.helper.util.TokenManager
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

        binding.btnLogin.setOnClickListener {
            val accountId = binding.etUsername.text.toString().trim()
            val token = binding.etPassword.text.toString().trim()

            if (accountId.isEmpty() || token.isEmpty()) {
                Toast.makeText(this, "请输入下发ID和Token", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            performLogin(accountId, token)
        }
    }

    private fun performLogin(accountId: String, token: String) {
        binding.btnLogin.isEnabled = false
        binding.btnLogin.text = "登录中..."

        lifecycleScope.launch {
            try {
                val response = ApiService.login(accountId, token)
                if (response.isSuccess && response.data != null) {
                    TokenManager.saveToken(response.data.token)
                    TokenManager.saveUsername(response.data.accountId.ifBlank { accountId })
                    TokenManager.saveGroupId(response.data.groupId)
                    TokenManager.saveApiType(TokenManager.API_TYPE_1)
                    TokenManager.saveAccountCount(
                        if (response.data.api1TaskAccountCount > 0) response.data.api1TaskAccountCount else response.data.taskAccountCount
                    )
                    getSharedPreferences("task_config", MODE_PRIVATE)
                        .edit().putString("saved_token", token).apply()
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
            binding.btnLogin.text = "登录"
        }
    }

    private fun startMain() {
        startActivity(Intent(this, MainActivity::class.java))
        finish()
    }
}
