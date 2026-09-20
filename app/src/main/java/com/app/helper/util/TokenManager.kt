package com.app.helper.util

import android.content.Context
import android.content.SharedPreferences
import com.app.helper.App

object TokenManager {
    private const val PREF_NAME = "task_helper_prefs"
    private const val KEY_TOKEN = "access_token"
    private const val KEY_USERNAME = "username"
    private const val KEY_USER_LEVEL = "user_level"
    private const val KEY_API_TYPE = "api_type"
    private const val KEY_GROUP_ID = "group_id"
    private const val KEY_ACCOUNT_COUNT = "account_count"

    const val API_TYPE_1 = 1

    private val prefs: SharedPreferences by lazy {
        App.context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
    }

    fun saveToken(token: String) { prefs.edit().putString(KEY_TOKEN, token).apply() }
    fun getToken(): String = prefs.getString(KEY_TOKEN, "") ?: ""

    fun saveUsername(username: String) { prefs.edit().putString(KEY_USERNAME, username).apply() }
    fun getUsername(): String = prefs.getString(KEY_USERNAME, "") ?: ""

    fun saveUserLevel(level: Int) { prefs.edit().putInt(KEY_USER_LEVEL, level).apply() }

    fun saveApiType(type: Int) { prefs.edit().putInt(KEY_API_TYPE, type).apply() }
    fun getApiType(): Int = prefs.getInt(KEY_API_TYPE, API_TYPE_1)

    fun saveGroupId(groupId: String) { prefs.edit().putString(KEY_GROUP_ID, groupId).apply() }
    fun getGroupId(): String = prefs.getString(KEY_GROUP_ID, "") ?: ""

    fun saveAccountCount(count: Int) { prefs.edit().putInt(KEY_ACCOUNT_COUNT, count).apply() }
    fun getAccountCount(): Int = prefs.getInt(KEY_ACCOUNT_COUNT, 0)

    fun isLoggedIn(): Boolean = getToken().isNotEmpty()

    fun logout() { prefs.edit().clear().apply() }
}
