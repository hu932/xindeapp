package com.app.helper.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.util.Log
import java.io.File

class DataReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "DataReceiver"
        const val ACTION_FILE = "com.app.helper.FILE_ACTION"
        const val ACTION_LOGS = "com.app.helper.LOGS_ACTION"
        const val ACTION_VER = "com.app.helper.VER_ACTION"
        const val ACTION_TOAST = "com.app.helper.TOAST_ACTION"
        const val ACTION_FLOAT = "com.app.helper.SHOW_FLOAT_ACTION"

        /** 大内容走文件传递的 extra 键。 */
        const val EXTRA_FILE_PATH = "file_path_tmp"

        /** 超过这个长度就不再塞进 Intent。Binder 事务上限约 1MB，
         *  这里留足余量，避免 TransactionTooLargeException。 */
        private const val INLINE_LIMIT = 128 * 1024

        var onFileReceived: ((uri: String, path: String, taskId: String) -> Unit)? = null
        var onLogReceived: ((message: String) -> Unit)? = null
        var onVersionReceived: ((version: String) -> Unit)? = null
        var onToastReceived: ((message: String) -> Unit)? = null
    }

    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            ACTION_FILE -> {
                val extraData = intent.getStringExtra("extra_data") ?: ""
                val extraPath = intent.getStringExtra("extra_data_path") ?: ""
                val taskId = intent.getStringExtra("task_id") ?: ""
                Log.d(TAG, "FILE_ACTION received: uri=$extraData path=$extraPath taskId=$taskId")

                val fileContent = readFileContent(context, extraData, extraPath)
                if (fileContent != null) {
                    onFileReceived?.invoke(extraData, extraPath, taskId)
                    val serviceIntent = Intent(context, com.app.helper.service.TaskService::class.java).apply {
                        action = "PROCESS_FILE"
                        putExtra("file_uri", extraData)
                        putExtra("file_path", extraPath)
                        putExtra("task_id", taskId)
                    }
                    // 大内容不能塞 Intent：Binder 事务上限约 1MB，
                    // 抓包响应常见 1MB 以上，塞进去会抛
                    // TransactionTooLargeException，进程当场崩。
                    if (fileContent.length <= INLINE_LIMIT) {
                        serviceIntent.putExtra("file_content", fileContent)
                    } else {
                        runCatching {
                            val dir = File(context.cacheDir, "inbox")
                            dir.mkdirs()
                            val f = File(dir, "task_${System.currentTimeMillis()}.txt")
                            f.writeText(fileContent)
                            serviceIntent.putExtra(EXTRA_FILE_PATH, f.absolutePath)
                            Log.d(TAG, "large payload ${fileContent.length}B -> ${f.name}")
                        }.onFailure {
                            // 落盘失败就退回直传，总比不传好
                            serviceIntent.putExtra("file_content", fileContent)
                        }
                    }
                    try {
                        context.startService(serviceIntent)
                    } catch (e: Exception) {
                        // 任何启动服务的异常都不能让广播接收器崩掉
                        Log.e(TAG, "startService failed", e)
                    }
                }
            }
            ACTION_LOGS -> {
                val msg = intent.getStringExtra("extra_data") ?: ""
                Log.d(TAG, "LOG: $msg")
                onLogReceived?.invoke(msg)
            }
            ACTION_VER -> {
                val ver = intent.getStringExtra("extra_data") ?: ""
                Log.d(TAG, "VERSION: $ver")
                onVersionReceived?.invoke(ver)
            }
            ACTION_TOAST -> {
                val msg = intent.getStringExtra("extra_data") ?: ""
                Log.d(TAG, "TOAST: $msg")
                onToastReceived?.invoke(msg)
            }
            ACTION_FLOAT -> {
                Log.d(TAG, "SHOW_FLOAT_ACTION received")
            }
        }
    }

    private fun readFileContent(context: Context, uriStr: String, path: String): String? {
        // Try reading via content URI first
        if (uriStr.startsWith("content://")) {
            try {
                val uri = Uri.parse(uriStr)
                context.contentResolver.openInputStream(uri)?.use { stream ->
                    return stream.bufferedReader().readText()
                }
            } catch (e: Exception) {
                Log.w(TAG, "Failed to read via URI: $uriStr", e)
            }
        }
        // Fallback to file path
        if (path.isNotEmpty()) {
            try {
                val file = java.io.File(path)
                if (file.exists()) return file.readText()
            } catch (e: Exception) {
                Log.w(TAG, "Failed to read file: $path", e)
            }
        }
        return null
    }
}
