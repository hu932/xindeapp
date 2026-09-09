package com.app.helper.chat

import android.graphics.Bitmap
import android.graphics.Rect
import com.google.mlkit.vision.common.InputImage
import com.google.mlkit.vision.text.Text
import com.google.mlkit.vision.text.TextRecognition
import com.google.mlkit.vision.text.chinese.ChineseTextRecognizerOptions
import kotlin.coroutines.resume
import kotlinx.coroutines.suspendCancellableCoroutine

data class OcrItem(val text: String, val bounds: Rect)

/**
 * 屏幕像素 OCR（ML Kit 中文识别）。坐标 = 截图位图像素坐标（与屏幕 px 同坐标空间）。
 * 无 Google Play 服务的设备上会失败 → 返回空列表，自动降级到“仅无障碍节点”。
 */
object OcrUtil {

    private val recognizer by lazy {
        runCatching { TextRecognition.getClient(ChineseTextRecognizerOptions.Builder().build()) }
            .getOrNull()
    }

    fun available(): Boolean = recognizer != null

    suspend fun recognize(bitmap: Bitmap?): List<OcrItem> {
        if (bitmap == null || recognizer == null) return emptyList()
        return suspendCancellableCoroutine { cont ->
            val image = InputImage.fromBitmap(bitmap, 0)
            recognizer!!.process(image)
                .addOnSuccessListener { result ->
                    if (cont.isCancelled) return@addOnSuccessListener
                    val list = ArrayList<OcrItem>()
                    for (block: Text.TextBlock in result.textBlocks) {
                        for (line in block.lines) {
                            val b = runCatching { line.boundingBox }.getOrNull() ?: Rect()
                            if (b.isEmpty) continue
                            val txt = line.text ?: continue
                            if (txt.isBlank()) continue
                            list.add(OcrItem(txt, b))
                        }
                    }
                    cont.resume(list)
                }
                .addOnFailureListener {
                    if (!cont.isCancelled) cont.resume(emptyList())
                }
        }
    }
}
