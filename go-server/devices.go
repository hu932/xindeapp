package main

import (
	"strings"
	"sync"
)

const heartbeatWriteInterval = 20

var deviceMu sync.Mutex

func readDeviceControls() map[string]map[string]any {
	m := map[string]map[string]any{}
	_ = readJSONFile(dp(deviceFile), &m)
	return m
}

func saveDeviceControls(devices map[string]map[string]any) {
	_ = atomicWrite(dp(deviceFile), jsonEncodePretty(devices))
}

func deviceKey(username, deviceID string) string {
	return md5hex(username + "|" + deviceID)
}

func displayDecodedExtensionAccountRemark(accountID string) string {
	accounts := decodedExtReadAccounts()
	for _, acc := range accounts {
		if trimStr(acc["accountId"]) == accountID {
			return trimStr(acc["remark"])
		}
	}
	return ""
}

func updateDeviceHeartbeat(input map[string]any, r *httpRequestCtx) map[string]any {
	username := trimStr(input["username"])
	if username == "" {
		username = "unknown"
	}
	deviceID := trimStr(input["device_id"])
	if deviceID == "" {
		deviceID = "unknown-" + md5short(r.ip)
	}
	deviceMu.Lock()
	defer deviceMu.Unlock()
	devices := readDeviceControls()
	key := deviceKey(username, deviceID)
	old := devices[key]
	if old == nil {
		old = map[string]any{}
	}
	disabled := asBool(old["disabled"], false)
	apiType := qInt(firstNonNil(input["api_type"], old["api_type"]))
	fpKey := trimStr(firstNonNil(input["fingerprint_key"], old["fingerprint_key"]))
	fpID := trimStr(firstNonNil(input["fingerprint_id"], old["fingerprint_id"]))
	fpName := trimStr(firstNonNil(input["fingerprint_name"], old["fingerprint_name"]))
	platform := stringsToLower(firstNonNilStr(input["platform"], input["device_type"], old["platform"]))
	client := trimStr(firstNonNil(input["client"], old["client"]))
	oldSeen := parseTimeToUnix(trimStr(old["last_seen"]))
	deviceLabel := "设备"
	if platform == "ios" {
		deviceLabel = "iOS设备"
	} else if v := trimStr(firstNonNil(input["device_label"], old["device_label"])); v != "" {
		deviceLabel = v
	} else if platform != "" {
		deviceLabel = stringsToUpper(platform) + "设备"
	}
	sameFP := trimStr(old["fingerprint_key"]) == fpKey && trimStr(old["fingerprint_id"]) == fpID && trimStr(old["fingerprint_name"]) == fpName
	if len(old) > 0 && !disabled && sameFP && int(timeNowUnix()-oldSeen) < heartbeatWriteInterval {
		return map[string]any{"ok": true, "allowed": true, "msg": "ok", "device_key": key, "throttled": true}
	}
	extRemark := ""
	if client == "gejie-extension" || strings.Contains(platform, "extension") {
		extRemark = displayDecodedExtensionAccountRemark(username)
	}
	accRemark := extRemark
	if accRemark == "" {
		accRemark = displayAccountRemark(apiType, username, trimStr(input["group_id"]))
	}
	if accRemark == "" {
		accRemark = trimStr(old["account_remark"])
	}
	newDev := map[string]any{
		"key": key, "username": username, "account_remark": accRemark,
		"group_id": trimStr(firstNonNil(input["group_id"], old["group_id"])), "device_id": deviceID,
		"fingerprint_key": fpKey, "fingerprint_id": fpID, "fingerprint_name": fpName,
		"api_type": apiType, "platform": platform, "device_type": platform,
		"device_label": deviceLabel, "client": client,
		"ip": r.ip, "last_seen": tsDateTime(),
		"first_seen": firstNonNilStr(old["first_seen"], tsDateTime()),
		"disabled": disabled, "disabled_reason": trimStr(old["disabled_reason"]),
	}
	for k, v := range old {
		if _, exists := newDev[k]; !exists {
			newDev[k] = v
		}
	}
	devices[key] = newDev
	saveDeviceControls(devices)
	msg := "ok"
	if disabled {
		msg = trimStr(newDev["disabled_reason"])
		if msg == "" {
			msg = "设备已被禁用"
		}
	}
	return map[string]any{"ok": true, "allowed": !disabled, "msg": msg, "device_key": key}
}

func checkDeviceAllowed(input map[string]any, r *httpRequestCtx) map[string]any {
	username := trimStr(input["username"])
	deviceID := trimStr(input["device_id"])
	if username == "" || deviceID == "" {
		return map[string]any{"allowed": true}
	}
	devices := readDeviceControls()
	key := deviceKey(username, deviceID)
	if d, ok := devices[key]; ok && asBool(d["disabled"], false) {
		msg := trimStr(d["disabled_reason"])
		if msg == "" {
			msg = "设备已被禁用"
		}
		return map[string]any{"allowed": false, "msg": msg, "device_key": key}
	}
	return updateDeviceHeartbeat(input, r)
}

func updateDeviceControl(key string, disabled bool, reason string) map[string]any {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	devices := readDeviceControls()
	if key == "" || devices[key] == nil {
		return map[string]any{"ok": false, "msg": "设备不存在"}
	}
	devices[key]["disabled"] = disabled
	if disabled {
		devices[key]["disabled_reason"] = reason
	} else {
		devices[key]["disabled_reason"] = ""
	}
	devices[key]["updated_at"] = tsDateTime()
	saveDeviceControls(devices)
	msg := "设备已启用"
	if disabled {
		msg = "设备已禁用"
	}
	return map[string]any{"ok": true, "msg": msg, "data": devices[key]}
}

// ---- Android 会话 ----
func readAndroidSessions() map[string]map[string]any {
	m := map[string]map[string]any{}
	_ = readJSONFile(dp(androidSessionFile), &m)
	return m
}

func saveAndroidSessions(s map[string]map[string]any) {
	_ = atomicWrite(dp(androidSessionFile), jsonEncodePretty(s))
}

func androidManagedDeviceKey(input map[string]any, r *httpRequestCtx) string {
	username := trimStr(input["username"])
	if username == "" {
		username = "unknown"
	}
	deviceID := trimStr(input["device_id"])
	if deviceID == "" {
		deviceID = "unknown-" + md5short(r.ip)
	}
	return deviceKey(username, deviceID)
}

func syncAndroidSession(input map[string]any, r *httpRequestCtx) map[string]any {
	username := trimStr(input["username"])
	if username == "" {
		username = "unknown"
	}
	deviceID := trimStr(input["device_id"])
	if deviceID == "" {
		deviceID = "unknown-" + md5short(r.ip)
	}
	key := deviceKey(username, deviceID)
	cookie := trimStr(firstNonNil(input["cookies"], input["cookie"]))
	cookieURL := trimStr(firstNonNil(input["cookie_url"], input["url"]))
	host := trimStr(input["cookie_host"])
	if host == "" && cookieURL != "" {
		if u, err := parseURL(cookieURL); err == nil {
			host = u.Host
		}
	}
	if cookie == "" {
		return map[string]any{"ok": false, "success": false, "msg": "cookie empty"}
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "device_id": deviceID, "api_type": 1}, r)
	if !asBool(dc["allowed"], true) {
		return map[string]any{"ok": false, "success": false, "msg": trimStr(dc["msg"]), "device_key": key}
	}

	deviceMu.Lock()
	defer deviceMu.Unlock()
	sessions := readAndroidSessions()
	sessions[key] = map[string]any{
		"key": key, "username": username, "device_id": deviceID,
		"fingerprint_key": trimStr(input["fingerprint_key"]),
		"device_label":    trimStr(input["device_label"]),
		"cookie": cookie, "cookie_len": len(cookie), "cookie_url": cookieURL, "cookie_host": host,
		"user_agent": trimStr(firstNonNil(input["user_agent"], input["ua"])),
		"ip": r.ip, "updated_at": tsDateTime(),
	}
	saveAndroidSessions(sessions)

	devices := readDeviceControls()
	old := devices[key]
	if old == nil {
		old = map[string]any{}
	}
	devices[key] = mergeMaps(old, map[string]any{
		"key": key, "username": username, "device_id": deviceID,
		"fingerprint_key": trimStr(firstNonNil(input["fingerprint_key"], old["fingerprint_key"])),
		"platform": "android", "device_type": trimStr(firstNonNil(input["device_type"], "android")),
		"device_label": trimStr(firstNonNil(input["device_label"], old["device_label"], "Android")),
		"client": trimStr(firstNonNil(input["client"], "ajie-android")),
		"client_label": trimStr(input["client_label"]),
		"ip": r.ip, "last_seen": tsDateTime(),
		"first_seen": firstNonNilStr(old["first_seen"], tsDateTime()),
		"disabled": asBool(old["disabled"], false), "disabled_reason": trimStr(old["disabled_reason"]),
		"has_android_session": true, "android_session_at": tsDateTime(), "android_cookie_host": host,
		"android_command": firstNonNilStr(old["android_command"], "run"),
		"android_poll_interval_seconds": max(10, qInt(firstNonNil(old["android_poll_interval_seconds"], 30))),
	})
	saveDeviceControls(devices)
	return map[string]any{"ok": true, "success": true, "msg": "session synced", "device_key": key, "data": map[string]any{"has_session": true, "cookie_host": host}}
}

func clearAndroidSession(input map[string]any, r *httpRequestCtx) map[string]any {
	key := androidManagedDeviceKey(input, r)
	deviceMu.Lock()
	defer deviceMu.Unlock()
	sessions := readAndroidSessions()
	if _, ok := sessions[key]; ok {
		delete(sessions, key)
		saveAndroidSessions(sessions)
	}
	devices := readDeviceControls()
	if d, ok := devices[key]; ok {
		d["has_android_session"] = false
		d["android_has_cookie"] = false
		d["android_session_at"] = ""
		d["android_cookie_host"] = ""
		d["updated_at"] = tsDateTime()
		devices[key] = d
		saveDeviceControls(devices)
	}
	return map[string]any{"ok": true, "success": true, "msg": "session cleared", "device_key": key}
}

func pollAndroidDevice(input map[string]any, r *httpRequestCtx) map[string]any {
	hb := updateDeviceHeartbeat(map[string]any{
		"api_type": 1, "platform": "android", "client": firstNonNil(input["client"], "ajie-android"),
		"username": trimStr(input["username"]), "device_id": trimStr(input["device_id"]),
		"group_id": trimStr(input["group_id"]),
	}, r)
	key := trimStr(hb["device_key"])
	if key == "" {
		key = androidManagedDeviceKey(input, r)
	}
	deviceMu.Lock()
	defer deviceMu.Unlock()
	devices := readDeviceControls()
	device := devices[key]
	if device == nil {
		device = map[string]any{}
	}
	if _, ok := devices[key]; ok {
		devices[key]["android_has_task_token"] = asBool(input["has_task_token"], false)
		devices[key]["android_has_cookie"] = asBool(input["has_cookie"], false)
		devices[key]["updated_at"] = tsDateTime()
		saveDeviceControls(devices)
		device = devices[key]
	}
	sessions := readAndroidSessions()
	hasSession := sessions[key] != nil && trimStr(sessions[key]["cookie"]) != ""
	disabled := asBool(device["disabled"], false)
	command := "run"
	if disabled {
		command = "stop"
	} else {
		command = trimStr(device["android_command"])
		if command == "" {
			command = "run"
		}
	}
	if !hasSession && command == "run" {
		command = "sync_session"
	}
	interval := max(10, qInt(firstNonNil(device["android_poll_interval_seconds"], 30)))
	if _, ok := devices[key]; ok {
		lastDelivered := trimStr(devices[key]["last_android_command_delivered"])
		pending := !asBool(devices[key]["android_command_delivered"], false)
		if pending || lastDelivered != command {
			devices[key]["last_android_command_delivered"] = command
			devices[key]["last_android_command_delivered_at"] = tsDateTime()
			devices[key]["android_command_delivered"] = true
			saveDeviceControls(devices)
			appendLog(logFile, map[string]any{
				"api_type": 1, "status": "android_command_delivered", "source": "android_device_poll",
				"msg": "Android app polled and received command", "command": command,
				"device_key": key, "username": trimStr(device["username"]), "device_id": trimStr(device["device_id"]),
				"device_label": trimStr(device["device_label"]), "client": firstNonNil(device["client"], "ajie-android"), "platform": "android",
			}, maxLogs)
		}
	}
	return map[string]any{
		"ok": true, "success": true, "allowed": !disabled, "device_key": key, "command": command,
		"data": map[string]any{
			"command": command, "has_session": hasSession, "poll_interval_seconds": interval,
			"disabled_reason": trimStr(device["disabled_reason"]), "server_time": tsDateTime(),
		},
	}
}

func updateAndroidDeviceCommand(key, command string, pollInterval int) map[string]any {
	allowed := map[string]bool{"run": true, "pause": true, "stop": true, "sync_session": true, "clear_session": true}
	if !allowed[command] {
		return map[string]any{"ok": false, "msg": "bad command"}
	}
	deviceMu.Lock()
	defer deviceMu.Unlock()
	devices := readDeviceControls()
	if key == "" || devices[key] == nil {
		return map[string]any{"ok": false, "msg": "device not found"}
	}
	devices[key]["android_command"] = command
	devices[key]["android_command_at"] = tsDateTime()
	devices[key]["android_command_seq"] = qInt(devices[key]["android_command_seq"]) + 1
	devices[key]["android_command_delivered"] = false
	devices[key]["api_type"] = 1
	if pollInterval > 0 {
		devices[key]["android_poll_interval_seconds"] = max(10, min(300, pollInterval))
	}
	devices[key]["updated_at"] = tsDateTime()
	saveDeviceControls(devices)
	appendLog(logFile, map[string]any{
		"api_type": 1, "status": "android_command_saved", "source": "android_admin_control",
		"msg": "Android command saved, waiting for app poll", "command": command, "device_key": key,
		"username": trimStr(devices[key]["username"]), "device_id": trimStr(devices[key]["device_id"]),
		"device_label": trimStr(devices[key]["device_label"]), "client": firstNonNil(devices[key]["client"], "ajie-android"), "platform": "android",
	}, maxLogs)
	return map[string]any{"ok": true, "msg": "command saved, waiting for app poll", "data": devices[key]}
}

// ---- 工具 ----
func timeNowUnix() int64 { return timeNow().Unix() }

var _ = timeNowUnix

func parseTimeToUnix(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := timeParse("2006-01-02 15:04:05", s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func firstNonNil(vals ...any) any {
	for _, v := range vals {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		return v
	}
	return nil
}

func firstNonNilStr(vals ...any) string {
	for _, v := range vals {
		if s := trimStr(v); s != "" {
			return s
		}
	}
	return ""
}

func stringsToLower(s string) string { return toLower(s) }

func stringsToUpper(s string) string { return toUpper(s) }

func mergeMaps(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
