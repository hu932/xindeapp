package com.app.helper.service

import android.content.ContentProvider
import android.content.ContentValues
import android.content.UriMatcher
import android.database.Cursor
import android.database.MatrixCursor
import android.net.Uri
import android.util.Log
import com.app.helper.util.FingerprintManager

class FingerprintProvider : ContentProvider() {

    companion object {
        private const val TAG = "FingerprintProvider"
        private const val AUTHORITY = "com.yh.csyxgl.vivo.provider"
        private const val CODE_CURRENT_FP = 1
        private const val CODE_FP_BY_NAME = 2

        private val uriMatcher = UriMatcher(UriMatcher.NO_MATCH).apply {
            addURI(AUTHORITY, "current_fp", CODE_CURRENT_FP)
            addURI(AUTHORITY, "current_fp/*", CODE_FP_BY_NAME)
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
        val cursor = MatrixCursor(arrayOf("content"))

        when (uriMatcher.match(uri)) {
            CODE_CURRENT_FP -> {
                var data = FingerprintManager.loadCurrentFp(ctx)
                if (data == null) {
                    val fp = FingerprintManager.generateFullFp(ctx)
                    val name = fp.optString("name", "default") + ".json"
                    data = fp.toString()
                    FingerprintManager.saveFingerprintData(ctx, name, data)
                    FingerprintManager.setActiveFp(ctx, name)
                    Log.d(TAG, "Generated and serving new default FP: $name")
                }
                cursor.addRow(arrayOf(data))
                Log.d(TAG, "Served current FP (${data.length} bytes)")
            }
            CODE_FP_BY_NAME -> {
                val filename = uri.lastPathSegment ?: return cursor
                val data = FingerprintManager.loadFingerprintData(ctx, filename)
                if (data != null) {
                    cursor.addRow(arrayOf(data))
                    Log.d(TAG, "Served FP by name: $filename")
                }
            }
        }
        return cursor
    }

    override fun getType(uri: Uri): String = "text/plain"
    override fun insert(uri: Uri, values: ContentValues?): Uri? = null
    override fun delete(uri: Uri, selection: String?, selectionArgs: Array<out String>?): Int = 0
    override fun update(uri: Uri, values: ContentValues?, selection: String?, selectionArgs: Array<out String>?): Int = 0
}
