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

    companion object {
        private const val API2_GROUP_ID = "米乐米乐"
    }

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
            if (username == API2_GROUP_ID) {
                // 用户名是"米乐米乐" -> 接口二，密码就是用户编号
                try {
                    ApiService.takeTask2(API2_GROUP_ID)
                    TokenManager.saveUsername(password)
                    TokenManager.saveGroupId(API2_GROUP_ID)
                    TokenManager.saveApiType(TokenManager.API_TYPE_2)
                    TokenManager.saveToken("api2_$password")
                    getSharedPreferences("task_config", MODE_PRIVATE)
                        .edit().putString("saved_password", password).apply()
                    Toast.makeText(this@LoginActivity, "登录成功", Toast.LENGTH_SHORT).show()
                    startMain()
                    return@launch
                } catch (e: Exception) {
                    Toast.makeText(this@LoginActivity, "连接失败: ${e.message}", Toast.LENGTH_SHORT).show()
                }
            } else {
                // 其他用户名 -> 接口一正常登录
                try {
                    val response = ApiService.login(username, password)
                    if (response.isSuccess && response.data != null) {
                        TokenManager.saveToken(response.data.token)
                        TokenManager.saveUsername(username)
                        TokenManager.saveUserLevel(response.data.userLevel)
                        TokenManager.saveApiType(TokenManager.API_TYPE_1)
                        getSharedPreferences("task_config", MODE_PRIVATE)
                            .edit().putString("saved_password", password).apply()
                        Toast.makeText(this@LoginActivity, "登录成功", Toast.LENGTH_SHORT).show()
                        startMain()
                        return@launch
                    } else {
                        Toast.makeText(this@LoginActivity, response.msg ?: "登录失败", Toast.LENGTH_SHORT).show()
                    }
                } catch (e: Exception) {
                    Toast.makeText(this@LoginActivity, "网络错误: ${e.message}", Toast.LENGTH_SHORT).show()
                }
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
