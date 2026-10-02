package main

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func readUpdateConfig() map[string]any {
	defaults := map[string]any{"version_code": 1, "version_name": "1.0.0", "apk_url": "", "title": "发现新版本", "message": "", "force": false, "updated_at": ""}
	var m map[string]any
	if readJSONFile(dp(updateFile), &m) {
		for k, v := range defaults {
			if _, ok := m[k]; !ok {
				m[k] = v
			}
		}
		return m
	}
	return defaults
}

func saveUpdateConfig(input map[string]any) map[string]any {
	c := readUpdateConfig()
	if v, ok := input["version_code"]; ok {
		c["version_code"] = max(1, qInt(v))
	}
	if v, ok := input["version_name"]; ok {
		c["version_name"] = trimStr(v)
	}
	if v, ok := input["apk_url"]; ok {
		c["apk_url"] = trimStr(v)
	}
	if v, ok := input["title"]; ok {
		c["title"] = trimStr(v)
	}
	if v, ok := input["message"]; ok {
		c["message"] = trimStr(v)
	}
	if v, ok := input["force"]; ok {
		c["force"] = asBool(v, false)
	}
	c["updated_at"] = tsDateTime()
	_ = atomicWrite(dp(updateFile), jsonEncodePretty(c))
	return map[string]any{"ok": true, "msg": "更新配置已保存", "data": c}
}

func buildUpdateResponse(currentCode int) map[string]any {
	c := readUpdateConfig()
	hasUpdate := trimStr(c["apk_url"]) != "" && qInt(c["version_code"]) > currentCode
	return map[string]any{
		"ok": true, "has_update": hasUpdate, "version_code": qInt(c["version_code"]),
		"version_name": qStr(c["version_name"]), "apk_url": qStr(c["apk_url"]),
		"title": qStr(c["title"]), "message": qStr(c["message"]), "force": asBool(c["force"], false), "updated_at": qStr(c["updated_at"]),
	}
}

var versionPartRe = regexp.MustCompile(`[^0-9]+`)

func pluginVersionCodeFromString(version string) int {
	parts := versionPartRe.Split(version, -1)
	var nums []int
	for _, p := range parts {
		if p == "" {
			continue
		}
		nums = append(nums, max(0, qInt(p)))
		if len(nums) >= 4 {
			break
		}
	}
	for len(nums) < 4 {
		nums = append(nums, 0)
	}
	return nums[0]*1000000 + nums[1]*10000 + nums[2]*100 + nums[3]
}

func readPluginUpdateConfig() map[string]any {
	defaults := map[string]any{"enabled": true, "version_code": 1000000, "version_name": "1.0.0", "package_url": "", "package_name": "", "package_size": 0, "title": "发现插件新版本", "message": "", "force": false, "updated_at": ""}
	var m map[string]any
	if readJSONFile(dp(pluginUpdateFile), &m) {
		for k, v := range defaults {
			if _, ok := m[k]; !ok {
				m[k] = v
			}
		}
		return m
	}
	return defaults
}

func savePluginUpdateConfig(input map[string]any) map[string]any {
	c := readPluginUpdateConfig()
	versionName := trimStr(input["version_name"])
	if versionName == "" {
		versionName = qStr(c["version_name"])
	}
	versionCode := qInt(input["version_code"])
	if versionCode < 1 {
		versionCode = pluginVersionCodeFromString(versionName)
	}
	if _, ok := input["enabled"]; ok {
		c["enabled"] = asBool(input["enabled"], true)
	}
	c["version_code"] = max(1, versionCode)
	c["version_name"] = versionName
	if v, ok := input["package_url"]; ok {
		c["package_url"] = trimStr(v)
	}
	if v, ok := input["title"]; ok {
		c["title"] = trimStr(v)
	}
	if v, ok := input["message"]; ok {
		c["message"] = trimStr(v)
	}
	if v, ok := input["force"]; ok {
		c["force"] = asBool(v, false)
	}
	c["updated_at"] = tsDateTime()
	_ = atomicWrite(dp(pluginUpdateFile), jsonEncodePretty(c))
	return map[string]any{"ok": true, "msg": "插件更新配置已保存", "data": c}
}

func buildPluginUpdateResponse(currentVersion string, currentCode int) map[string]any {
	c := readPluginUpdateConfig()
	if currentCode < 1 && currentVersion != "" {
		currentCode = pluginVersionCodeFromString(currentVersion)
	}
	hasUpdate := asBool(c["enabled"], true) && trimStr(c["package_url"]) != "" && qInt(c["version_code"]) > max(0, currentCode)
	return map[string]any{
		"ok": true, "has_update": hasUpdate, "current_version": currentVersion, "current_version_code": currentCode,
		"version_code": qInt(c["version_code"]), "version_name": qStr(c["version_name"]),
		"package_url": qStr(c["package_url"]), "download_url": qStr(c["package_url"]),
		"title": qStr(c["title"]), "message": qStr(c["message"]), "force": asBool(c["force"], false), "updated_at": qStr(c["updated_at"]),
	}
}

func readVersionControls() map[string]any {
	defaults := map[string]any{"min_version_code": 0, "min_version_msg": "您的版本已停用，请更新到最新版本", "blocked_versions": map[string]any{}, "api2_login_min": 0, "api2_login_max": 10000, "updated_at": ""}
	var m map[string]any
	if readJSONFile(dp(versionCtrlFile), &m) {
		for k, v := range defaults {
			if _, ok := m[k]; !ok {
				m[k] = v
			}
		}
		return m
	}
	return defaults
}

func saveVersionControls(input map[string]any) map[string]any {
	c := readVersionControls()
	if v, ok := input["min_version_code"]; ok {
		c["min_version_code"] = max(0, qInt(v))
	}
	if v, ok := input["min_version_msg"]; ok {
		c["min_version_msg"] = trimStr(v)
	}
	if v, ok := input["blocked_versions"].(map[string]any); ok {
		c["blocked_versions"] = v
	}
	api2Min := max(0, qInt(firstNonNil(input["api2_login_min"], c["api2_login_min"])))
	api2Max := max(api2Min, qInt(firstNonNil(input["api2_login_max"], c["api2_login_max"])))
	c["api2_login_min"] = api2Min
	c["api2_login_max"] = api2Max
	c["updated_at"] = tsDateTime()
	_ = atomicWrite(dp(versionCtrlFile), jsonEncodePretty(c))
	return map[string]any{"ok": true, "msg": "版本控制已保存", "data": c}
}

func checkVersionBlocked(versionCode int) map[string]any {
	c := readVersionControls()
	if qInt(c["min_version_code"]) > 0 && versionCode > 0 && versionCode < qInt(c["min_version_code"]) {
		msg := trimStr(c["min_version_msg"])
		if msg == "" {
			msg = "您的版本已停用，请更新到最新版本"
		}
		return map[string]any{"blocked": true, "message": msg}
	}
	blocked, _ := c["blocked_versions"].(map[string]any)
	if v, ok := blocked[strconvItoa(versionCode)]; ok {
		msg := ""
		if m, ok := v.(map[string]any); ok {
			msg = trimStr(m["message"])
		} else {
			msg = qStr(v)
		}
		if msg == "" {
			msg = "此版本已停用，请更新"
		}
		return map[string]any{"blocked": true, "message": msg}
	}
	return map[string]any{"blocked": false}
}

// ---- APK / 插件包上传 ----
func uploadUpdateAPK(r *http.Request, input map[string]any) map[string]any {
	file, header, err := r.FormFile("apk")
	if err != nil {
		return saveUpdateConfig(input)
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".apk") {
		return map[string]any{"ok": false, "msg": "只允许上传 .apk 文件"}
	}
	_ = os.MkdirAll(dp(apkDir), 0o755)
	versionCode := max(1, qInt(firstNonNil(input["version_code"], 1)))
	safeName := "taskhelper_v" + strconvItoa(versionCode) + "_" + timeNow().Format("20060102_150405") + ".apk"
	target := dp(apkDir) + "/" + safeName
	out, err := os.Create(target)
	if err != nil {
		return map[string]any{"ok": false, "msg": "保存APK失败，请检查目录权限"}
	}
	_, _ = io.Copy(out, file)
	out.Close()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	basePath := r.URL.Path
	if i := strings.LastIndex(basePath, "/"); i >= 0 {
		basePath = basePath[:i]
	}
	apkURL := scheme + "://" + host + basePath + "/app_updates/" + urlQueryEscape(safeName)
	input["apk_url"] = apkURL
	saved := saveUpdateConfig(input)
	if data, ok := saved["data"].(map[string]any); ok {
		if fi, err := os.Stat(target); err == nil {
			data["apk_file"] = safeName
			data["apk_size"] = fi.Size()
		}
	}
	return saved
}

func uploadPluginUpdatePackage(r *http.Request, input map[string]any) map[string]any {
	file, header, err := r.FormFile("package")
	if err != nil {
		return savePluginUpdateConfig(input)
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".zip" && ext != ".crx" {
		return map[string]any{"ok": false, "msg": "只允许上传 .zip 或 .crx 插件包"}
	}
	_ = os.MkdirAll(dp(pluginUpdateDir), 0o755)
	versionName := trimStr(input["version_name"])
	if versionName == "" {
		versionName = "1.0.0"
	}
	versionCode := max(1, qInt(firstNonNil(input["version_code"], pluginVersionCodeFromString(versionName))))
	safeVersion := regexp.MustCompile(`[^0-9A-Za-z._-]+`).ReplaceAllString(versionName, "_")
	safeName := "ajie_plugin_v" + safeVersion + "_" + strconvItoa(versionCode) + "_" + timeNow().Format("20060102_150405") + ext
	target := dp(pluginUpdateDir) + "/" + safeName
	out, err := os.Create(target)
	if err != nil {
		return map[string]any{"ok": false, "msg": "保存插件包失败，请检查目录权限"}
	}
	_, _ = io.Copy(out, file)
	out.Close()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	basePath := r.URL.Path
	if i := strings.LastIndex(basePath, "/"); i >= 0 {
		basePath = basePath[:i]
	}
	packageURL := scheme + "://" + host + basePath + "/plugin_updates/" + urlQueryEscape(safeName)
	input["package_url"] = packageURL
	input["version_code"] = versionCode
	input["version_name"] = versionName
	saved := savePluginUpdateConfig(input)
	if data, ok := saved["data"].(map[string]any); ok {
		if fi, err := os.Stat(target); err == nil {
			data["package_name"] = safeName
			data["package_size"] = fi.Size()
			_ = atomicWrite(dp(pluginUpdateFile), jsonEncodePretty(data))
		}
	}
	return saved
}

var _ = multipart.ErrMessageTooLarge
var _ = time.Now
