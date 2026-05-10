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
            val groupId = binding.etGroupId.text.toString().trim()

            if (username.isEmpty()) {
                Toast.makeText(this, "请输入用户名", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            if (password.isEmpty() && groupId.isEmpty()) {
                Toast.makeText(this, "请输入密码或组ID", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            performLogin(username, password, groupId)
        }
    }

    private fun performLogin(username: String, password: String, groupId: String) {
        binding.btnLogin.isEnabled = false
        binding.btnLogin.text = "登录中..."

        lifecycleScope.launch {
            var api1Success = false

            if (password.isNotEmpty()) {
                try {
                    val response = ApiService.login(username, password)
                    if (response.isSuccess && response.data != null) {
                        TokenManager.saveToken(response.data.token)
                        TokenManager.saveUsername(username)
                        TokenManager.saveUserLevel(response.data.userLevel)
                        TokenManager.saveApiType(TokenManager.API_TYPE_1)
                        if (groupId.isNotEmpty()) TokenManager.saveGroupId(groupId)
                        getSharedPreferences("task_config", MODE_PRIVATE)
                            .edit().putString("saved_password", password).apply()
                        Toast.makeText(this@LoginActivity, "接口一登录成功", Toast.LENGTH_SHORT).show()
                        api1Success = true
                        startMain()
                        return@launch
                    }
                } catch (_: Exception) { }
            }

            if (!api1Success && groupId.isNotEmpty()) {
                try {
                    ApiService.takeTask2(groupId)
                    TokenManager.saveUsername(username)
                    TokenManager.saveGroupId(groupId)
                    TokenManager.saveApiType(TokenManager.API_TYPE_2)
                    TokenManager.saveToken("api2_$groupId")
                    Toast.makeText(this@LoginActivity, "接口二登录成功 (组ID: $groupId)", Toast.LENGTH_SHORT).show()
                    startMain()
                    return@launch
                } catch (e: Exception) {
                    Toast.makeText(this@LoginActivity, "接口二连接失败: ${e.message}", Toast.LENGTH_SHORT).show()
                }
            } else if (!api1Success) {
                Toast.makeText(this@LoginActivity, "接口一登录失败，请填写组ID使用接口二", Toast.LENGTH_SHORT).show()
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
