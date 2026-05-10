package com.app.helper.ui.main

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import com.app.helper.R
import com.app.helper.databinding.ActivityMainBinding
import com.app.helper.service.TaskService
import com.app.helper.ui.fp.FpManagerActivity
import com.app.helper.ui.login.LoginActivity
import com.app.helper.util.FingerprintManager
import com.app.helper.util.TokenManager
import org.json.JSONObject

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        requestPermissions()
        initFingerprint()
        setupUI()

        // Start service with float window
        if (checkOverlayPermission()) {
            startTaskService()
        }
    }

    private fun setupUI() {
        binding.tvUsername.text = "用户: ${TokenManager.getUsername()}"

        binding.btnChangeFp.setOnClickListener {
            startActivity(Intent(this, FpManagerActivity::class.java))
        }

        binding.btnTestApi.setOnClickListener {
            showDelayConfigDialog()
        }

        binding.btnStartAuto.setOnClickListener {
            if (checkOverlayPermission()) {
                startTaskService()
                Toast.makeText(this, "服务已启动，请在悬浮窗中操作", Toast.LENGTH_SHORT).show()
            }
        }

        binding.btnStopAuto.setOnClickListener {
            stopService(Intent(this, TaskService::class.java))
            Toast.makeText(this, "服务已停止", Toast.LENGTH_SHORT).show()
        }

        binding.btnSingleTask.setOnClickListener {
            Toast.makeText(this, "请使用悬浮窗中的【一键做任务】按钮", Toast.LENGTH_SHORT).show()
        }

        binding.btnLogout.setOnClickListener {
            AlertDialog.Builder(this)
                .setTitle("确认退出")
                .setMessage("确定要退出登录吗?")
                .setPositiveButton("确定") { _, _ ->
                    TokenManager.logout()
                    stopService(Intent(this, TaskService::class.java))
                    startActivity(Intent(this, LoginActivity::class.java))
                    finish()
                }
                .setNegativeButton("取消", null)
                .show()
        }

        binding.btnClearLog.setOnClickListener {}
    }

    private fun initFingerprint() {
        val fpFiles = FingerprintManager.listFpFiles(this)
        if (fpFiles.isEmpty()) {
            val fp = FingerprintManager.generateFullFp(this)
            val name = fp.optString("name", "default")
            FingerprintManager.saveFingerprintData(this, name, fp.toString())
            FingerprintManager.setActiveFp(this, name)
        }
        updateFpDisplay()
    }

    private fun updateFpDisplay() {
        val fpFiles = FingerprintManager.listFpFiles(this)
        binding.tvFpCount.text = "指纹数量: ${fpFiles.size}"
        val currentFp = FingerprintManager.loadCurrentFp(this)
        if (currentFp != null) {
            try {
                val json = JSONObject(currentFp)
                val aid = json.optString("androidId", "N/A").take(8)
                binding.tvCurrentFp.text = "当前指纹: ID:${aid}..."
            } catch (e: Exception) {
                binding.tvCurrentFp.text = "当前指纹: 已加载"
            }
        }
    }

    override fun onResume() {
        super.onResume()
        updateFpDisplay()
        binding.tvStatus.text = if (TaskService.isRunning) "状态: 服务运行中(悬浮窗)" else "状态: 服务未启动"
    }

    private fun showDelayConfigDialog() {
        val prefs = getSharedPreferences("task_config", MODE_PRIVATE)
        val curMin = prefs.getInt("delay_min", 3)
        val curMax = prefs.getInt("delay_max", 8)

        val view = layoutInflater.inflate(R.layout.dialog_delay, null)
        val etMin = view.findViewById<android.widget.EditText>(R.id.et_delay_min)
        val etMax = view.findViewById<android.widget.EditText>(R.id.et_delay_max)
        etMin.setText(curMin.toString())
        etMax.setText(curMax.toString())

        AlertDialog.Builder(this)
            .setTitle("设置随机延时(秒)")
            .setView(view)
            .setPositiveButton("确定") { _, _ ->
                val min = etMin.text.toString().toIntOrNull() ?: 3
                val max = etMax.text.toString().toIntOrNull() ?: 8
                prefs.edit().putInt("delay_min", min.coerceAtLeast(1))
                    .putInt("delay_max", max.coerceAtLeast(min)).apply()
                Toast.makeText(this, "延时已设置: $min-$max 秒", Toast.LENGTH_SHORT).show()
                if (TaskService.isRunning) {
                    val intent = Intent(this, TaskService::class.java).apply {
                        action = "SET_DELAY"
                        putExtra("delay_min", min)
                        putExtra("delay_max", max)
                    }
                    startService(intent)
                }
            }
            .setNegativeButton("取消", null)
            .show()
    }

    private fun startTaskService() {
        val intent = Intent(this, TaskService::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun checkOverlayPermission(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M && !Settings.canDrawOverlays(this)) {
            AlertDialog.Builder(this)
                .setTitle("需要悬浮窗权限")
                .setMessage("请授予悬浮窗权限，用于显示任务状态")
                .setPositiveButton("去设置") { _, _ ->
                    val intent = Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                        Uri.parse("package:$packageName"))
                    startActivity(intent)
                }
                .setNegativeButton("取消", null)
                .show()
            return false
        }
        return true
    }

    private fun requestPermissions() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                try {
                    startActivity(Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                        data = Uri.parse("package:$packageName")
                    })
                } catch (e: Exception) {
                    startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
                }
            }
        } else {
            val perms = arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE, Manifest.permission.WRITE_EXTERNAL_STORAGE)
            val needed = perms.filter { ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED }
            if (needed.isNotEmpty()) ActivityCompat.requestPermissions(this, needed.toTypedArray(), 100)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 101)
            }
        }
    }
}
