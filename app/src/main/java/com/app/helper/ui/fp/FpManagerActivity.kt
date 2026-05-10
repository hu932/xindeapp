package com.app.helper.ui.fp

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.app.helper.R
import com.app.helper.databinding.ActivityFpManagerBinding
import com.app.helper.util.FingerprintManager
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

class FpManagerActivity : AppCompatActivity() {

    private lateinit var binding: ActivityFpManagerBinding
    private val adapter = FpAdapter()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityFpManagerBinding.inflate(layoutInflater)
        setContentView(binding.root)

        binding.toolbar.setNavigationOnClickListener { finish() }

        binding.rvFpList.layoutManager = LinearLayoutManager(this)
        binding.rvFpList.adapter = adapter

        binding.btnGenerate.setOnClickListener { generateNewFp() }
        binding.btnBatchGenerate.setOnClickListener { batchGenerateFp() }

        refreshList()
    }

    private fun generateNewFp() {
        val fp = FingerprintManager.generateFullFp(this)
        val name = fp.optString("name", "fp_${System.currentTimeMillis()}")
        FingerprintManager.saveFingerprintData(this, name, fp.toString())
        Toast.makeText(this, "已生成: $name", Toast.LENGTH_SHORT).show()
        refreshList()
    }

    private fun batchGenerateFp() {
        val counts = arrayOf("5", "10", "20", "50")
        AlertDialog.Builder(this)
            .setTitle("批量生成指纹")
            .setItems(counts) { _, which ->
                val count = counts[which].toInt()
                Thread {
                    repeat(count) { i ->
                        val fp = FingerprintManager.generateFullFp(this)
                        val name = fp.optString("name", "fp_${System.currentTimeMillis()}_$i")
                        FingerprintManager.saveFingerprintData(this, name, fp.toString())
                        Thread.sleep(10)
                    }
                    runOnUiThread {
                        Toast.makeText(this, "已批量生成 $count 个指纹", Toast.LENGTH_SHORT).show()
                        refreshList()
                    }
                }.start()
            }
            .show()
    }

    private fun refreshList() {
        val files = FingerprintManager.listFpFiles(this)
        adapter.setData(files)
        binding.tvFpCount.text = "共 ${files.size} 个指纹"
    }

    inner class FpAdapter : RecyclerView.Adapter<FpAdapter.VH>() {
        private var items = listOf<FingerprintManager.FpFileInfo>()

        fun setData(data: List<FingerprintManager.FpFileInfo>) {
            items = data
            notifyDataSetChanged()
        }

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): VH {
            val view = LayoutInflater.from(parent.context).inflate(R.layout.item_fp, parent, false)
            return VH(view)
        }

        override fun onBindViewHolder(holder: VH, position: Int) {
            val item = items[position]
            holder.bind(item)
        }

        override fun getItemCount() = items.size

        inner class VH(view: View) : RecyclerView.ViewHolder(view) {
            private val tvName: TextView = view.findViewById(R.id.tv_fp_name)
            private val tvInfo: TextView = view.findViewById(R.id.tv_fp_info)
            private val tvActive: TextView = view.findViewById(R.id.tv_fp_active)

            fun bind(item: FingerprintManager.FpFileInfo) {
                tvName.text = item.name
                val date = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.getDefault()).format(Date(item.lastModified))
                val sizeKb = item.size / 1024

                var androidId = ""
                try {
                    val data = FingerprintManager.loadFingerprintData(this@FpManagerActivity, item.name)
                    if (data != null) {
                        val json = JSONObject(data)
                        androidId = json.optString("androidId", "").take(8) + "..."
                    }
                } catch (_: Exception) {}

                tvInfo.text = "${sizeKb}KB | $date | ID: $androidId"
                tvActive.visibility = if (item.isActive) View.VISIBLE else View.GONE

                itemView.setOnClickListener {
                    showFpOptions(item)
                }
            }
        }
    }

    private fun showFpOptions(item: FingerprintManager.FpFileInfo) {
        val options = mutableListOf("设为当前使用", "查看详情", "删除")
        if (item.isActive) options[0] = "✓ 当前使用中"

        AlertDialog.Builder(this)
            .setTitle(item.name)
            .setItems(options.toTypedArray()) { _, which ->
                when (which) {
                    0 -> {
                        FingerprintManager.setActiveFp(this, item.name)
                        Toast.makeText(this, "已切换指纹: ${item.name}", Toast.LENGTH_SHORT).show()
                        refreshList()
                    }
                    1 -> showFpDetail(item.name)
                    2 -> {
                        AlertDialog.Builder(this)
                            .setTitle("确认删除")
                            .setMessage("删除 ${item.name}?")
                            .setPositiveButton("删除") { _, _ ->
                                FingerprintManager.deleteFpFile(this, item.name)
                                refreshList()
                            }
                            .setNegativeButton("取消", null)
                            .show()
                    }
                }
            }
            .show()
    }

    private fun showFpDetail(filename: String) {
        val data = FingerprintManager.loadFingerprintData(this, filename) ?: return
        try {
            val json = JSONObject(data)
            val info = buildString {
                appendLine("Android ID: ${json.optString("androidId")}")
                appendLine("GSF ID: ${json.optString("gsfAndroidId")}")
                appendLine("广告 ID: ${json.optString("advertiserId")}")
                appendLine("SIM状态: ${json.optString("gsm.sim.state")}")
                appendLine("运营商: ${json.optString("gsm.operator.alpha")}")
                appendLine("总内存: ${json.optString("totalMem").toLongOrNull()?.div(1024*1024*1024) ?: "?"}GB")
                appendLine("存储: ${json.optString("totalSize").toLongOrNull()?.div(1024*1024*1024) ?: "?"}GB")
                appendLine("启动次数: ${json.optString("bootCount")}")
                appendLine()
                val f7 = json.optString("f7_params")
                if (f7.isNotEmpty()) {
                    val f7Json = JSONObject(f7)
                    appendLine("--- F7 参数 ---")
                    appendLine("serialHexPool: ${f7Json.optJSONArray("serialHexPool")?.length() ?: 0}个")
                    appendLine("snHexPool: ${f7Json.optJSONArray("snHexPool")?.length() ?: 0}个")
                    appendLine("c1Offset: ${f7Json.optInt("c1TimeOffsetDays")}天")
                    appendLine("c2Offset: ${f7Json.optInt("c2TimeOffsetDays")}天")
                    appendLine("leafOffset: ${f7Json.optInt("leafOffsetDays")}天")
                    appendLine("leafGapMs: ${f7Json.optInt("leafGapMs")}ms")
                    appendLine("vbh2: ${f7Json.optString("vbh2").take(16)}...")
                }
            }
            AlertDialog.Builder(this)
                .setTitle(filename)
                .setMessage(info)
                .setPositiveButton("确定", null)
                .show()
        } catch (e: Exception) {
            Toast.makeText(this, "解析失败: ${e.message}", Toast.LENGTH_SHORT).show()
        }
    }
}
