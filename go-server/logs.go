package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var logMu sync.Mutex

// appendLog：JSONL 追加日志，保留最近 max 条
func appendLog(file string, entry map[string]any, max int) {
	entry["time"] = tsDateTime()
	line := jsonEncode(entry)
	logMu.Lock()
	defer logMu.Unlock()

	f := dp(file)
	_ = os.MkdirAll(dirOf(f), 0o775)

	fh, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = fh.WriteString(line + "\n")
	_ = fh.Close()
}

func dirOf(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return "."
	}
	return p[:i]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// readLogs：读取日志文件末尾 limit 条
func readLogs(file string, limit int) []map[string]any {
	raw, err := os.ReadFile(dp(file))
	if err != nil {
		return []map[string]any{}
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	var out []map[string]any
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		var e map[string]any
		if json.Unmarshal([]byte(l), &e) == nil {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// logIsSuccessfulSubmitResponse：仅返回成功提交日志（用于后台 logs 展示）
func logIsSuccessfulSubmitResponse(log map[string]any) bool {
	status := trimStr(log["status"])
	source := trimStr(log["source"])
	return status == "success" && strings.Contains(source, "submit")
}

// isControllerPayload / isWebPluginPayload（与 PHP 一致）
func isControllerPayload(input map[string]any) bool {
	checks := []string{
		toLower(trimStr(input["client"])), trimStr(input["device_label"]), trimStr(input["fingerprint_name"]),
		toLower(trimStr(input["device_id"])), toLower(trimStr(input["fingerprint_key"])),
		toLower(trimStr(input["fingerprint_id"])), toLower(trimStr(input["source"])),
	}
	for _, v := range checks {
		if v == "" {
			continue
		}
		if v == "gejie-controller" || v == "中控" || v == "ajie中控" {
			return true
		}
		if strings.HasPrefix(v, "ctrl-") || strings.HasPrefix(v, "ctrl_") ||
			strings.Contains(v, "gejie-controller") || strings.Contains(v, "中控") {
			return true
		}
	}
	return false
}

func isWebPluginPayload(input map[string]any) bool {
	if isControllerPayload(input) {
		return false
	}
	source := toLower(trimStr(input["source"]))
	client := toLower(trimStr(input["client"]))
	deviceType := toLower(trimStr(input["device_type"]))
	platform := toLower(trimStr(input["platform"]))
	return strings.HasPrefix(source, "web_") || asBool(input["_web_obfuscated"], false) ||
		client == "gejie-extension" || deviceType == "web" || platform == "web"
}

// resolveClientLabel：与 PHP 一致
func resolveClientLabel(input map[string]any) string {
	if isControllerPayload(input) {
		return "中控"
	}
	if isWebPluginPayload(input) {
		return "网页插件"
	}
	return ""
}

// resolveLogSource：与 PHP 一致
func resolveLogSource(input map[string]any, defaultSource, webSource, controllerSource string) string {
	if isControllerPayload(input) {
		return controllerSource
	}
	if isWebPluginPayload(input) {
		return webSource
	}
	return defaultSource
}
