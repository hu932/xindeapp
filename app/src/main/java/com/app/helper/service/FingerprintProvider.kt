package com.app.helper.service

import android.content.ContentProvider
import android.content.ContentValues
import android.content.UriMatcher
import android.database.Cursor
import android.database.MatrixCursor
import android.net.Uri
import android.util.Log
import com.app.helper.util.FingerprintManager
import java.io.BufferedReader
import java.io.File
import java.io.FileInputStream
import java.io.InputStreamReader

class FingerprintProvider : ContentProvider() {

    companion object {
        private const val TAG = "FingerprintProvider"
        private const val AUTHORITY = "com.yh.csyxgl.vivo.provider"
        private const val CODE_CURRENT_FP = 1

        private val uriMatcher = UriMatcher(UriMatcher.NO_MATCH).apply {
            addURI(AUTHORITY, "current_fp", CODE_CURRENT_FP)
        }
    }

    override fun onCreate(): Boolean = true

    override fun query(
        uri: Uri,
        projection: Array<out String>?,
        selection: String?,
        selectionArgs: Array<out String>?,
        sortOrder: String?
    ): Cursor? {
        val ctx = context ?: return null

        when (uriMatcher.match(uri)) {
            CODE_CURRENT_FP -> {
                val activeName = FingerprintManager.getActiveFpName(ctx)
                if (activeName.isNullOrEmpty()) {
                    Log.w(TAG, "No active FP set")
                    return null
                }

                val fpFile = File(ctx.filesDir, "fps/$activeName")
                if (!fpFile.exists()) {
                    Log.w(TAG, "FP file not found: $activeName")
                    return null
                }

                val data = readFileContent(fpFile.path)
                val cursor = MatrixCursor(arrayOf("key", "content"))
                cursor.addRow(arrayOf(activeName, data))
                Log.d(TAG, "Served FP: $activeName (${data.length} bytes)")
                return cursor
            }
        }
        return null
    }

    private fun readFileContent(filePath: String): String {
        return FileInputStream(File(filePath)).use { fis ->
            BufferedReader(InputStreamReader(fis, Charsets.UTF_8), 8192).use { reader ->
                reader.readText()
            }
        }
    }

    override fun getType(uri: Uri): String? = null
    override fun insert(uri: Uri, values: ContentValues?): Uri? = null
    override fun delete(uri: Uri, selection: String?, selectionArgs: Array<out String>?): Int = 0
    override fun update(uri: Uri, values: ContentValues?, selection: String?, selectionArgs: Array<out String>?): Int = 0
}
