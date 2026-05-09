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
            val username = binding.etUsername.text.toString().trim()
            val password = binding.etPassword.text.toString().trim()

            if (username.isEmpty() || password.isEmpty()) {
                Toast.makeText(this, "请输入用户名和密码", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            performLogin(username, password)
        }
    }

    private fun performLogin(username: String, password: String) {
        binding.btnLogin.isEnabled = false
        binding.btnLogin.text = "登录中..."

        lifecycleScope.launch {
            try {
                val response = ApiService.login(username, password)
                if (response.isSuccess && response.data != null) {
                    TokenManager.saveToken(response.data.token)
                    TokenManager.saveUsername(username)
                    TokenManager.saveUserLevel(response.data.userLevel)
                    getSharedPreferences("task_config", MODE_PRIVATE)
                        .edit().putString("saved_password", password).apply()
                    Toast.makeText(this@LoginActivity, "登录成功", Toast.LENGTH_SHORT).show()
                    startMain()
                } else {
                    val msg = response.msg ?: "登录失败"
                    Toast.makeText(this@LoginActivity, msg, Toast.LENGTH_SHORT).show()
                }
            } catch (e: Exception) {
                Toast.makeText(this@LoginActivity, "网络错误: ${e.message}", Toast.LENGTH_SHORT).show()
            } finally {
                binding.btnLogin.isEnabled = true
                binding.btnLogin.text = "登录"
            }
        }
    }

    private fun startMain() {
        startActivity(Intent(this, MainActivity::class.java))
        finish()
    }
}
