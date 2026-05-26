package com.app.helper.service

import android.content.ContentProvider
import android.content.ContentValues
import android.database.Cursor
import android.database.MatrixCursor
import android.net.Uri
import java.io.File

class EnvironmentProvider : ContentProvider() {

    override fun onCreate(): Boolean = true

    override fun query(
        uri: Uri,
        projection: Array<out String>?,
        selection: String?,
        selectionArgs: Array<out String>?,
        sortOrder: String?
    ): Cursor? {
        val ctx = context ?: return null
        val fileName = when {
            uri.pathSegments.firstOrNull() == "current_fp" -> getCurrentFpName(ctx.filesDir)
            uri.pathSegments.firstOrNull() == "fp" -> uri.lastPathSegment
            else -> null
        } ?: return null

        val fpFile = File(File(ctx.filesDir, FP_DIR), fileName)
        if (!fpFile.exists() || !fpFile.isFile) return null

        return MatrixCursor(arrayOf("key", "content")).apply {
            addRow(arrayOf(fileName, fpFile.readText()))
        }
    }

    override fun getType(uri: Uri): String? = null

    override fun insert(uri: Uri, values: ContentValues?): Uri? = null

    override fun delete(uri: Uri, selection: String?, selectionArgs: Array<out String>?): Int = 0

    override fun update(
        uri: Uri,
        values: ContentValues?,
        selection: String?,
        selectionArgs: Array<out String>?
    ): Int = 0

    private fun getCurrentFpName(filesDir: File): String? {
        val fpDir = File(filesDir, FP_DIR)
        if (!fpDir.exists() || !fpDir.isDirectory) return null
        return fpDir.listFiles()
            ?.filter { it.isFile && it.length() > 0L }
            ?.maxByOrNull { it.lastModified() }
            ?.name
    }

    private companion object {
        const val FP_DIR = "fps"
    }
}
