package com.app.helper.util

import android.content.Context
import android.content.SharedPreferences
import com.app.helper.App

object TokenManager {
    private const val PREF_NAME = "task_helper_prefs"
    private const val KEY_TOKEN = "access_token"
    private const val KEY_USERNAME = "username"
    private const val KEY_USER_LEVEL = "user_level"

    private val prefs: SharedPreferences by lazy {
        App.context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
    }

    fun saveToken(token: String) {
        prefs.edit().putString(KEY_TOKEN, token).apply()
    }

    fun getToken(): String = prefs.getString(KEY_TOKEN, "") ?: ""

    fun saveUsername(username: String) {
        prefs.edit().putString(KEY_USERNAME, username).apply()
    }

    fun getUsername(): String = prefs.getString(KEY_USERNAME, "") ?: ""

    fun saveUserLevel(level: Int) {
        prefs.edit().putInt(KEY_USER_LEVEL, level).apply()
    }

    fun isLoggedIn(): Boolean = getToken().isNotEmpty()

    fun logout() {
        prefs.edit().clear().apply()
    }
}
