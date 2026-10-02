package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func webPluginBase64URLDecode(value string) []byte {
	value = strings.NewReplacer("-", "+", "_", "/").Replace(value)
	switch len(value) % 4 {
	case 2:
		value += "=="
	case 3:
		value += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	return decoded
}

func normalizeWebPluginAliases(input map[string]any) map[string]any {
	apiTypeHint := qInt(firstNonNil(input["api_type"], input["at"], 0))
	actMap := map[string]string{"l": "api1_login", "t": "api1_take", "s": "api1_submit", "h": "heartbeat"}
	if apiTypeHint == 2 {
		actMap = map[string]string{"l": "api2_login", "t": "api2_take", "s": "api2_submit", "h": "heartbeat"}
	}
	action := trimStr(firstNonNil(input["act"], input["a"]))
	if mapped, ok := actMap[action]; ok {
		action = mapped
	}
	if action != "" {
		input["act"] = action
	}
	aliasMap := map[string]string{
		"u": "username", "p": "password", "k": "auth_token", "q": "url", "r": "result", "av": "appVersion",
		"sid": "submit_id", "tid": "task_id", "deal_id": "deal_id", "tr": "trace_id", "iid": "item_id", "sh": "shop_id",
		"pu": "product_url", "pd": "pdp_url", "g": "group_id", "did": "device_id", "at": "api_type", "pf": "platform",
		"dt": "device_type", "dl": "device_label", "c": "client", "fk": "fingerprint_key", "fid": "fingerprint_id", "fn": "fingerprint_name",
	}
	for from, to := range aliasMap {
		if v, ok := input[from]; ok {
			if _, exists := input[to]; !exists || trimStr(input[to]) == "" {
				input[to] = v
			}
		}
	}
	if _, ok := input["api_type"]; !ok {
		switch trimStr(input["act"]) {
		case "api1_login", "api1_take", "api1_submit", "heartbeat":
			input["api_type"] = 1
		case "api2_login", "api2_take", "api2_submit":
			input["api_type"] = 2
		}
	}
	return input
}

func applyWebPluginServerDefaults(input map[string]any) map[string]any {
	if !asBool(input["_web_obfuscated"], false) {
		return input
	}
	apiType := qInt(firstNonNil(input["api_type"], input["at"], 1))
	if apiType == 2 {
		apiType = 2
	} else {
		apiType = 1
	}
	serverUsername := osGetenv("GEJIE_WEB_API1_USERNAME")
	serverPassword := osGetenv("GEJIE_WEB_API1_PASSWORD")
	if apiType == 2 {
		serverUsername = osGetenv("GEJIE_WEB_API2_USERNAME")
		serverPassword = osGetenv("GEJIE_WEB_API2_PASSWORD")
	}
	if serverUsername == "" {
		serverUsername = osGetenv("GEJIE_WEB_USERNAME")
	}
	if serverPassword == "" {
		serverPassword = osGetenv("GEJIE_WEB_PASSWORD")
	}
	if trimStr(input["username"]) == "" && serverUsername != "" {
		input["username"] = serverUsername
	}
	if trimStr(input["group_id"]) == "" && serverUsername != "" {
		input["group_id"] = serverUsername
	}
	act := trimStr(input["act"])
	if (act == "api1_login" || act == "api2_login") && qStr(input["password"]) == "" && serverPassword != "" {
		input["password"] = serverPassword
	}
	return input
}

func normalizeWebPluginPayload(input map[string]any) (map[string]any, bool) {
	if x, ok := input["x"].(string); ok {
		decodedRaw := webPluginBase64URLDecode(x)
		if decodedRaw == nil {
			return input, false
		}
		var decoded map[string]any
		if json.Unmarshal(decodedRaw, &decoded) != nil {
			return input, false
		}
		body := decoded
		if b, ok := decoded["b"].(map[string]any); ok {
			body = b
		}
		action := trimStr(firstNonNil(decoded["a"], body["a"]))
		if action != "" {
			body["a"] = action
		}
		body["_web_obfuscated"] = true
		return applyWebPluginServerDefaults(normalizeWebPluginAliases(body)), true
	}
	return normalizeWebPluginAliases(input), true
}

func readRawJSONBody(r *http.Request) map[string]any {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 128<<20))
	var m map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &m) == nil {
		return m
	}
	return map[string]any{}
}

// handleProxyPOST：POST /decrypt_proxy.php 主分发
func handleProxyPOST(w http.ResponseWriter, r *http.Request) {
	reqCtx := newHTTPRequestCtx(r)
	rawBodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 128<<20))
	rawBody := string(rawBodyBytes)
	if strings.TrimSpace(rawBody) == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "空请求体"})
		return
	}
	var input map[string]any
	if json.Unmarshal(rawBodyBytes, &input) != nil {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "JSON解析失败"})
		return
	}
	normalized, ok := normalizeWebPluginPayload(input)
	if !ok {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "网页插件参数解析失败"})
		return
	}
	input = normalized

	postAct := trimStr(input["act"])

	// ---- 子账号管理（POST act=admin_*）----
	if inList(postAct, []string{"admin_sub_accounts", "admin_sub_account_save", "admin_save_sub_accounts", "admin_sub_account_toggle", "admin_sub_account_delete", "admin_sub_bindings", "admin_sub_binding_save", "admin_sub_binding_delete", "admin_sub_binding_options"}) {
		adminKey := trimStr(firstNonNil(input["admin_key"], input["key"]))
		if !verifyAuthKey(adminKey) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "后台密码错误"})
			return
		}
		switch postAct {
		case "admin_sub_accounts":
			writeJSONOK(w, map[string]any{"ok": true, "data": subAccountsPublicList(), "usage": readSubAccountUsage()})
		case "admin_sub_account_save":
			writeJSONOK(w, saveSubAccountRecord(input))
		case "admin_save_sub_accounts":
			writeJSONOK(w, saveSubAccountsFromText(qStr(input["accounts"])))
		case "admin_sub_account_toggle":
			input["enabled"] = asBool(input["enabled"], true)
			input["password"] = "-"
			writeJSONOK(w, saveSubAccountRecord(input))
		case "admin_sub_account_delete":
			writeJSONOK(w, deleteSubAccount(trimStr(firstNonNil(input["username"], input["sub_username"]))))
		case "admin_sub_bindings":
			writeJSONOK(w, map[string]any{"ok": true, "data": subAccountBindingsPublic()})
		case "admin_sub_binding_save":
			writeJSONOK(w, saveSubAccountBindingRecord(input))
		case "admin_sub_binding_delete":
			writeJSONOK(w, deleteSubAccountBinding(input))
		case "admin_sub_binding_options":
			writeJSONOK(w, saveSubAccountBindingOptions(input))
		}
		return
	}

	switch postAct {
	case "sub_login":
		handleSubLogin(w, r, reqCtx, input)
		return
	case "sub_profile":
		auth := requireSubAccount(input, "", reqCtx)
		if !asBool(auth["ok"], true) {
			writeJSONOK(w, subAccountErrorResponse(auth))
			return
		}
		writeJSONOK(w, map[string]any{"ok": true, "code": "200", "data": subAccountPublic(auth["account"].(map[string]any)), "msg": nil})
		return
	case "sub_logs":
		auth := requireSubAccount(input, "view_logs", reqCtx)
		if !asBool(auth["ok"], true) {
			writeJSONOK(w, subAccountErrorResponse(auth))
			return
		}
		subUsername := trimStr(auth["username"])
		limit := min(100, max(1, qInt(firstNonNil(input["limit"], 30))))
		var logs []map[string]any
		for _, log := range readLogs(logFile, maxLogs) {
			src := trimStr(log["source"])
			if !inList(src, []string{"sub_login", "sub_take_r", "sub_submit", "sub_fetch_submit"}) {
				continue
			}
			if trimStr(firstNonNil(log["sub_username"], log["username"])) != subUsername {
				continue
			}
			logs = append(logs, log)
			if len(logs) >= limit {
				break
			}
		}
		writeJSONOK(w, map[string]any{"ok": true, "code": "200", "data": logs, "msg": nil})
		return
	case "sub_upload":
		auth := requireSubAccount(input, "view_logs", reqCtx)
		if !asBool(auth["ok"], true) {
			writeJSONOK(w, subAccountErrorResponse(auth))
			return
		}
		id := trimStr(firstNonNil(input["id"], input["upload_id"]))
		detail := readUploadDump(id)
		if detail == nil {
			writeJSONOK(w, map[string]any{"ok": false, "code": "404", "data": nil, "msg": "上传详情不存在"})
			return
		}
		if trimStr(detail["sub_username"]) != trimStr(auth["username"]) {
			writeJSONOK(w, map[string]any{"ok": false, "code": "403", "data": nil, "msg": "无权查看此上传详情"})
			return
		}
		writeJSONOK(w, map[string]any{"ok": true, "code": "200", "data": detail, "msg": nil})
		return
	case "sub_take_r":
		handleSubTakeR(w, r, reqCtx, input)
		return
	case "sub_submit":
		handleSubSubmit(w, r, reqCtx, input, rawBody)
		return
	case "sub_fetch_submit":
		handleSubFetchSubmit(w, r, reqCtx, input, rawBody)
		return
	case "android_session_sync":
		writeJSONOK(w, syncAndroidSession(input, reqCtx))
		return
	case "android_session_clear":
		writeJSONOK(w, clearAndroidSession(input, reqCtx))
		return
	case "android_device_poll":
		writeJSONOK(w, pollAndroidDevice(input, reqCtx))
		return
	case "android_task_take":
		handleAndroidTaskTake(w, r, reqCtx, input)
		return
	case "api1_login":
		handleAPI1Login(w, r, reqCtx, input)
		return
	case "api2_login":
		handleAPI2Login(w, r, reqCtx, input)
		return
	case "api1_take":
		handleAPI1Take(w, r, reqCtx, input)
		return
	case "api2_take":
		handleAPI2Take(w, r, reqCtx, input)
		return
	case "api1_submit":
		handleAPI1Submit(w, r, reqCtx, input, rawBody)
		return
	case "api2_submit":
		handleAPI2Submit(w, r, reqCtx, input, rawBody)
		return
	case "android_session_fetch_submit":
		handleAndroidSessionFetchSubmit(w, r, reqCtx, input, rawBody)
		return
	case "heartbeat":
		writeJSONOK(w, updateDeviceHeartbeat(input, reqCtx))
		return
	case "fingerprint_event":
		writeJSONOK(w, recordFingerprintEnvEvent(input, reqCtx))
		return
	}

	// ---- 加密 / 明文上传转发 ----
	handleUpload(w, r, reqCtx, input, rawBody)
}

func inList(v string, list []string) bool {
	for _, x := range list {
		if v == x {
			return true
		}
	}
	return false
}

func subAccountsPublicList() []map[string]any {
	var out []map[string]any
	for _, a := range readSubAccounts() {
		out = append(out, subAccountPublic(a))
	}
	return out
}

func readUploadDump(id string) map[string]any {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return nil
	}
	var m map[string]any
	if readJSONFile(dp("upload_dumps/"+id+".json"), &m) {
		return m
	}
	return nil
}

func handleSubLogin(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	username := trimStr(firstNonNil(input["sub_username"], input["username"]))
	password := qStr(firstNonNil(input["sub_password"], input["password"]))
	account := subAccountFind(username)
	loginOK := account != nil && asBool(account["enabled"], true) && password != "" && subAccountPasswordMatches(account, password)
	if !loginOK {
		errMsg := "子账号或密码错误/已禁用"
		if account == nil {
			errMsg = "子账号或密码错误/已禁用"
		} else if !asBool(account["enabled"], true) {
			errMsg = "子账号或密码错误/已禁用"
		}
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": username, "device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"status": "sub_login_rejected", "source": "sub_login", "error": errMsg,
		}, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "401", "data": nil, "msg": "子账号或密码错误/已禁用"})
		return
	}
	bindDevice := trimStr(account["bind_device"])
	deviceID := trimStr(input["device_id"])
	if bindDevice != "" && (deviceID == "" || !hashEquals(bindDevice, deviceID)) {
		msg := "子账号绑定设备不匹配"
		if deviceID == "" {
			msg = "此子账号需要提交绑定设备ID"
		}
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": username, "device_id": deviceID, "fingerprint_key": input["fingerprint_key"],
			"status": "sub_login_rejected", "source": "sub_login", "error": "bind device mismatch",
		}, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "403", "data": nil, "msg": msg})
		return
	}
	dcInput := map[string]any{"username": username, "api_type": 1, "platform": firstNonNil(input["platform"], "sub")}
	for k, v := range input {
		if _, exists := dcInput[k]; !exists {
			dcInput[k] = v
		}
	}
	dc := checkDeviceAllowed(dcInput, reqCtx)
	if !asBool(dc["allowed"], true) {
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": username, "device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"status": "device_disabled", "source": "sub_login", "error": dc["msg"],
		}, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	issued := issueSubAccountToken(username, input, reqCtx, 86400)
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": username, "device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": "success", "source": "sub_login",
	}, maxLogs)
	writeJSONOK(w, map[string]any{
		"ok": true, "success": true, "code": "200",
		"data": map[string]any{
			"token": issued["token"], "expires_at": issued["expires_at"], "expires_at_text": issued["expires_at_text"],
			"profile": subAccountPublic(account),
		}, "msg": nil,
	})
}

func handleSubTakeR(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	auth := requireSubAccount(input, "take_r", reqCtx)
	if !asBool(auth["ok"], true) {
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": firstNonNilStr(auth["username"], input["username"], input["sub_username"]),
			"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"status": "sub_take_rejected", "source": "sub_take_r", "error": auth["msg"],
		}, maxLogs)
		writeJSONOK(w, subAccountErrorResponse(auth))
		return
	}
	subAccount := auth["account"].(map[string]any)
	subUsername := trimStr(auth["username"])
	usageBefore := subAccountUsageSnapshot(subAccount)
	if qInt(usageBefore["daily_take_limit"]) > 0 && qInt(usageBefore["take_r"]) >= qInt(usageBefore["daily_take_limit"]) {
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": subUsername, "sub_username": subUsername,
			"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"status": "take_limit_reached", "source": "sub_take_r", "error": "daily take limit reached", "usage_today": usageBefore,
		}, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "429", "data": nil, "msg": "子账号今日领取次数已达上限", "usage_today": usageBefore})
		return
	}
	tokenInfo := getAPIToken(api1Auth, api1TokenFile, false)
	if !tokenInfo.OK || tokenInfo.Token == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "502", "data": nil, "msg": firstNonNilStr(tokenInfo.Msg, "上游登录失败")})
		return
	}
	res := forwardGetToServer(api1RTakeURL, []string{"Authorization: Bearer " + tokenInfo.Token})
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	takeOK := false
	if serverResp != nil {
		code := trimStr(serverResp["code"])
		data, _ := serverResp["data"].(map[string]any)
		takeOK = code == "200" && data != nil && trimStr(data["taskUrl"]) != ""
	}
	usageAfter := usageBefore
	if takeOK {
		incrementSubAccountUsage(subUsername, "take_r")
		usageAfter = subAccountUsageSnapshot(subAccount)
	}
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": subUsername, "sub_username": subUsername, "upstream_username": api1Auth.Username,
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": condStr(res.Error != "", "forward_fail", condStr(takeOK, "success", "task_server_error")),
		"source": "sub_take_r", "forward_url": api1RTakeURL,
		"task_id": serverDataStr(serverResp, "taskId"), "task_url": serverDataStr(serverResp, "taskUrl"),
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "usage_today": usageAfter,
	}, maxLogs)
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "R任务领取转发失败: " + res.Error})
		return
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"source": "sub_take_r", "sub_username": subUsername, "upstream_username": api1Auth.Username,
			"forward_url": api1RTakeURL, "token_source": condStr(tokenInfo.Cached, "cache", "login"), "usage_today": usageAfter,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "上游R任务接口响应异常", "raw": res.Response})
}

func serverDataStr(serverResp map[string]any, key string) string {
	if serverResp == nil {
		return ""
	}
	if data, ok := serverResp["data"].(map[string]any); ok {
		return trimStr(data[key])
	}
	return ""
}

func handleSubSubmit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	auth := requireSubAccount(input, "submit", reqCtx)
	if !asBool(auth["ok"], true) {
		writeJSONOK(w, subAccountErrorResponse(auth))
		return
	}
	subAccount := auth["account"].(map[string]any)
	subUsername := trimStr(auth["username"])
	submitURL := trimStr(firstNonNil(input["url"], input["task_url"]))
	resultValue := firstNonNil(input["result"], input["data"])
	submitResult := anyToStr(resultValue)
	submitID := trimStr(input["submit_id"])
	taskID := trimStr(firstNonNil(input["task_id"], input["trace_id"]))
	itemID := trimStr(input["item_id"])
	shopID := trimStr(input["shop_id"])
	if submitURL == "" || submitResult == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "400", "data": nil, "msg": "提交URL或result为空"})
		return
	}
	tokenInfo := getAPIToken(api1Auth, api1TokenFile, false)
	if !tokenInfo.OK || tokenInfo.Token == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "502", "data": nil, "msg": firstNonNilStr(tokenInfo.Msg, "上游登录失败")})
		return
	}
	forwardBody := jsonEncode(map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": submitURL, "result": submitResult})
	res := forwardToServer(api1SubmitURL, forwardBody, submitGzip, []string{"Authorization: Bearer " + tokenInfo.Token})
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	isForwardSuccess := api1ForwardSuccess(serverResp)
	var serverTaskID, serverSubmitID string
	if data, ok := serverResp["data"].(map[string]any); ok {
		serverTaskID = trimStr(firstNonNil(data["task_id"], data["taskId"], data["trace_id"], data["traceId"], data["id"]))
		serverSubmitID = trimStr(data["submit_id"])
	}
	proxyContextMatch := true
	if taskID != "" && serverTaskID != "" {
		proxyContextMatch = serverTaskID == taskID
	} else if submitID != "" && serverSubmitID != "" {
		proxyContextMatch = serverSubmitID == submitID
	}
	if !proxyContextMatch {
		isForwardSuccess = false
	}
	incrementSubAccountUsage(subUsername, "submit")
	uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|sub_submit")
	status := "task_server_error"
	if res.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": subUsername, "sub_username": subUsername, "upstream_username": api1Auth.Username,
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"], "device_type": input["device_type"], "platform": input["platform"],
		"client": firstNonNilStr(input["client"], "sub_client"), "client_label": resolveClientLabel(input),
		"status": status, "source": "sub_submit", "task_url": submitURL, "forward_url": api1SubmitURL,
		"forward_result_len": len(submitResult), "submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "upload_id": uploadID, "usage_today": subAccountUsageSnapshot(subAccount),
	}, maxLogs)
	saveUploadDump(uploadID, map[string]any{
		"api_type": 1, "source": "sub_submit", "sub_username": subUsername, "upstream_username": api1Auth.Username,
		"target_url": api1SubmitURL, "method": "POST", "content_type": "application/json; charset=utf-8",
		"request_body": mustJSONMap(forwardBody), "http_code": res.HTTPCode, "response_json": serverResp, "response_raw": res.Response, "error": res.Error,
		"submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
	})
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "提交转发失败: " + res.Error, "upload_id": uploadID})
		return
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"source": "sub_submit", "sub_username": subUsername, "upstream_username": api1Auth.Username,
			"submit_accepted": isForwardSuccess, "context_match": proxyContextMatch, "upload_id": uploadID, "status": status,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "上游提交接口响应异常", "raw": res.Response, "upload_id": uploadID})
}

func handleSubFetchSubmit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	auth := requireSubAccount(input, "fetch_submit", reqCtx)
	if !asBool(auth["ok"], true) {
		writeJSONOK(w, subAccountErrorResponse(auth))
		return
	}
	subAccount := auth["account"].(map[string]any)
	subUsername := trimStr(auth["username"])
	taskURL := trimStr(firstNonNil(input["url"], input["task_url"]))
	taskID := trimStr(firstNonNil(input["task_id"], input["trace_id"]))
	cookies := trimStr(firstNonNil(input["cookies"], input["cookie"]))
	userAgent := trimStr(firstNonNil(input["user_agent"], input["ua"]))
	if taskURL == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "400", "data": nil, "msg": "任务URL为空"})
		return
	}
	fetchResult := androidFetchURLWithSession(taskURL, cookies, userAgent, nil)
	responseBodyRaw := fetchResult.Response
	var responseJSON map[string]any
	_ = jsonDecode(responseBodyRaw, &responseJSON)
	fetchOK := fetchResult.Error == "" && fetchResult.HTTPCode >= 200 && fetchResult.HTTPCode < 300 && responseBodyRaw != ""
	if !fetchOK {
		uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|sub_fetch_fail")
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": subUsername, "sub_username": subUsername,
			"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"status": "sub_fetch_fail", "source": "sub_fetch_submit", "task_url": taskURL, "task_id": taskID, "forward_url": taskURL,
			"http_code": fetchResult.HTTPCode, "error": firstNonNilStr(fetchResult.Error, "http "+strconvItoa(fetchResult.HTTPCode)),
			"response": compactLogResponse(responseJSON, maxLogRespBytes), "upload_id": uploadID,
		}, maxLogs)
		saveUploadDump(uploadID, map[string]any{
			"api_type": 1, "source": "sub_fetch_submit", "sub_username": subUsername, "target_url": taskURL, "method": "GET",
			"http_code": fetchResult.HTTPCode, "content_type": fetchResult.ContentType, "response_json": responseJSON, "response_raw": responseBodyRaw, "error": fetchResult.Error,
		})
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "任务URL抓取失败", "upload_id": uploadID, "fetch": map[string]any{
			"http_code": fetchResult.HTTPCode, "error": fetchResult.Error, "content_type": fetchResult.ContentType,
		}})
		return
	}
	tokenInfo := getAPIToken(api1Auth, api1TokenFile, false)
	if !tokenInfo.OK || tokenInfo.Token == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "502", "data": nil, "msg": firstNonNilStr(tokenInfo.Msg, "上游登录失败")})
		return
	}
	forwardBody := jsonEncode(map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": taskURL, "result": responseBodyRaw})
	submitResult := forwardToServer(api1SubmitURL, forwardBody, submitGzip, []string{"Authorization: Bearer " + tokenInfo.Token})
	var serverResp map[string]any
	_ = jsonDecode(submitResult.Response, &serverResp)
	isForwardSuccess := api1ForwardSuccess(serverResp)
	incrementSubAccountUsage(subUsername, "submit")
	uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|sub_fetch_submit")
	status := "task_server_error"
	if submitResult.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": subUsername, "sub_username": subUsername, "upstream_username": api1Auth.Username,
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": status, "source": "sub_fetch_submit", "task_url": taskURL, "task_id": taskID, "forward_url": api1SubmitURL,
		"fetch_http_code": fetchResult.HTTPCode, "fetch_result_len": len(responseBodyRaw), "forward_result_len": len(responseBodyRaw),
		"forward_attempts": []map[string]any{forwardAttemptSummary(submitResult, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "upload_id": uploadID, "usage_today": subAccountUsageSnapshot(subAccount),
	}, maxLogs)
	saveUploadDump(uploadID, map[string]any{
		"api_type": 1, "source": "sub_fetch_submit", "sub_username": subUsername, "upstream_username": api1Auth.Username,
		"target_url": api1SubmitURL, "method": "POST", "request_body": mustJSONMap(forwardBody),
		"fetch": map[string]any{"url": taskURL, "http_code": fetchResult.HTTPCode, "content_type": fetchResult.ContentType, "response_json": responseJSON},
		"http_code": submitResult.HTTPCode, "response_json": serverResp, "response_raw": submitResult.Response, "error": submitResult.Error,
	})
	if submitResult.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "提交转发失败: " + submitResult.Error, "upload_id": uploadID})
		return
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"source": "sub_fetch_submit", "sub_username": subUsername, "upstream_username": api1Auth.Username,
			"fetch_http_code": fetchResult.HTTPCode, "submit_accepted": isForwardSuccess, "upload_id": uploadID, "status": status,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "data": nil, "msg": "上游提交接口响应异常", "raw": submitResult.Response, "upload_id": uploadID})
}

func handleAndroidTaskTake(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	appUsername := trimStr(firstNonNil(input["app_user"], input["app_username"], input["username"]))
	if appUsername == "" {
		appUsername = "android"
	}
	resolveInput := map[string]any{"api_type": 1, "platform": "android"}
	for k, v := range input {
		resolveInput[k] = v
	}
	if trimStr(resolveInput["app_user"]) == "" {
		resolveInput["app_user"] = appUsername
	}
	subAuth := resolveSubAccountForApp(resolveInput, "take_r", reqCtx)
	if !asBool(subAuth["ok"], true) {
		writeJSONOK(w, map[string]any{
			"ok": false, "success": false, "code": firstNonNilStr(subAuth["code"], "403"), "data": nil, "msg": firstNonNilStr(subAuth["msg"], "未分配可用子账号"),
		})
		return
	}
	subAccount := subAuth["account"].(map[string]any)
	username := trimStr(subAuth["username"])
	usageBefore := subAccountUsageSnapshot(subAccount)
	if qInt(usageBefore["daily_take_limit"]) > 0 && qInt(usageBefore["take_r"]) >= qInt(usageBefore["daily_take_limit"]) {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "429", "data": nil, "msg": "子账号今日领取次数已达上限", "usage_today": usageBefore})
		return
	}
	tokenInfo := getAPIToken(api1Auth, api1TokenFile, false)
	if !tokenInfo.OK || tokenInfo.Token == "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "502", "msg": firstNonNilStr(tokenInfo.Msg, "task server login failed")})
		return
	}
	res := forwardGetToServer(api1RTakeURL, []string{"Authorization: Bearer " + tokenInfo.Token})
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	takeOK := false
	if serverResp != nil {
		data, _ := serverResp["data"].(map[string]any)
		takeOK = trimStr(serverResp["code"]) == "200" && data != nil && trimStr(data["taskUrl"]) != ""
	}
	usageAfter := usageBefore
	if takeOK {
		incrementSubAccountUsage(username, "take_r")
		usageAfter = subAccountUsageSnapshot(subAccount)
	}
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": username, "sub_username": username, "app_username": appUsername, "upstream_username": api1Auth.Username,
		"assign_mode": subAuth["assign_mode"], "binding_key": subAuth["binding_key"], "device_key": subAuth["device_key"],
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": condStr(res.Error != "", "forward_fail", condStr(takeOK, "success", "task_server_error")),
		"source": "android_managed_take", "forward_url": api1RTakeURL,
		"task_id": serverDataStr(serverResp, "taskId"), "task_url": serverDataStr(serverResp, "taskUrl"),
		"response": compactLogResponse(serverResp, maxLogRespBytes), "usage_today": usageAfter,
	}, maxLogs)
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "msg": "take forward failed: " + res.Error})
		return
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"source": "android_managed_take", "sub_username": username, "app_username": appUsername,
			"assign_mode": subAuth["assign_mode"], "binding_key": subAuth["binding_key"], "forward_url": api1RTakeURL,
			"token_source": condStr(tokenInfo.Cached, "cache", "login"), "usage_today": usageAfter,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "success": false, "code": "500", "msg": "task server abnormal", "raw": res.Response})
}

// resolveSubAccountForApp：Android 任务领取的子账号解析（简化版，保留 credential/绑定/自动分配）
func resolveSubAccountForApp(input map[string]any, permission string, reqCtx *httpRequestCtx) map[string]any {
	hasExplicit := subAccountTokenFromRequest(input, reqCtx.r) != "" || trimStr(input["sub_username"]) != "" ||
		(trimStr(input["username"]) != "" && qStr(firstNonNil(input["sub_password"], input["password"])) != "")
	if hasExplicit {
		auth := requireSubAccount(input, permission, reqCtx)
		if asBool(auth["ok"], true) {
			auth["assign_mode"] = "credential"
		}
		return auth
	}
	identityKeys := subAccountIdentityKeys(input)
	directUsername := ""
	for _, f := range []string{"app_user", "app_username", "sub_user", "sub_account"} {
		if v, ok := input[f]; ok && trimStr(v) != "" {
			directUsername = trimStr(v)
			break
		}
	}
	if directUsername == "" {
		directUsername = trimStr(input["username"])
	}
	accounts := readSubAccounts()
	if directUsername != "" {
		if account := subAccountFind(directUsername); account != nil {
			if !subAccountUsableForPermission(account, permission) {
				msg := "子账号已禁用"
				code := "403"
				if permission == "take_r" && subAccountTakeLimitReached(account) {
					msg = "子账号今日领取次数已达上限"
					code = "429"
				}
				return map[string]any{"ok": false, "code": code, "msg": msg, "username": trimStr(account["username"]), "auth_source": "sub_accounts", "identity_keys": identityKeys}
			}
			dc := checkDeviceAllowed(map[string]any{"username": trimStr(account["username"]), "api_type": 1, "platform": firstNonNil(input["platform"], "android")}, reqCtx)
			if !asBool(dc["allowed"], true) {
				return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "username": trimStr(account["username"]), "auth_source": "sub_accounts", "identity_keys": identityKeys}
			}
			return map[string]any{"ok": true, "account": account, "username": trimStr(account["username"]), "auth_source": "sub_accounts", "assign_mode": "app_user", "binding_key": "app_user:" + trimStr(account["username"]), "identity_keys": identityKeys, "device_key": dc["device_key"]}
		}
		decodedExtSyncAllTaskAccountsToWhitelists()
		allowed := readAllowedAccounts(1)
		if !containsStr(accountNames(allowed), directUsername) {
			return map[string]any{"ok": false, "code": "403", "msg": "账号未授权", "username": directUsername, "auth_source": "api1_allowed_accounts", "identity_keys": identityKeys}
		}
		remarkMap := accountRemarkMap(allowed)
		account := map[string]any{"username": directUsername, "enabled": true, "remark": remarkMap[directUsername], "permissions": subAccountDefaultPermissions(), "daily_take_limit": 0, "bind_device": ""}
		dc := checkDeviceAllowed(map[string]any{"username": directUsername, "api_type": 1, "platform": firstNonNil(input["platform"], "android")}, reqCtx)
		if !asBool(dc["allowed"], true) {
			return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "username": directUsername, "auth_source": "api1_allowed_accounts", "identity_keys": identityKeys}
		}
		return map[string]any{"ok": true, "account": account, "username": directUsername, "auth_source": "api1_allowed_accounts", "assign_mode": "api1_whitelist", "binding_key": "api1_allowed:" + directUsername, "identity_keys": identityKeys, "device_key": dc["device_key"]}
	}
	if len(accounts) == 0 {
		return map[string]any{"ok": false, "code": "403", "msg": "未分配可用子账号", "identity_keys": identityKeys}
	}
	bindingData := readSubAccountBindings()
	bmap, _ := bindingData["bindings"].(map[string]any)
	for _, identity := range identityKeys {
		entry, _ := bmap[trimStr(identity["key"])].(map[string]any)
		if entry == nil || !asBool(entry["enabled"], true) {
			continue
		}
		account := subAccountFind(trimStr(entry["sub_username"]))
		if account != nil && subAccountUsableForPermission(account, permission) {
			dc := checkDeviceAllowed(map[string]any{"username": trimStr(account["username"]), "api_type": 1, "platform": firstNonNil(input["platform"], "android")}, reqCtx)
			if !asBool(dc["allowed"], true) {
				return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "username": trimStr(account["username"])}
			}
			return map[string]any{"ok": true, "account": account, "username": trimStr(account["username"]), "auth_source": "binding", "assign_mode": "binding", "binding_key": identity["key"], "identity_keys": identityKeys, "device_key": dc["device_key"]}
		}
	}
	deviceID := trimStr(input["device_id"])
	if deviceID != "" {
		for _, account := range accounts {
			if trimStr(account["bind_device"]) == deviceID {
				if !subAccountUsableForPermission(account, permission) {
					msg := "子账号已禁用"
					code := "403"
					if permission == "take_r" && subAccountTakeLimitReached(account) {
						msg = "子账号今日领取次数已达上限"
						code = "429"
					}
					return map[string]any{"ok": false, "code": code, "msg": msg, "username": trimStr(account["username"]), "binding_key": "device:" + deviceID, "identity_keys": identityKeys}
				}
				dc := checkDeviceAllowed(map[string]any{"username": trimStr(account["username"]), "api_type": 1, "platform": firstNonNil(input["platform"], "android")}, reqCtx)
				if !asBool(dc["allowed"], true) {
					return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "username": trimStr(account["username"])}
				}
				return map[string]any{"ok": true, "account": account, "username": trimStr(account["username"]), "auth_source": "bind_device", "assign_mode": "bind_device", "binding_key": "device:" + deviceID, "identity_keys": identityKeys, "device_key": dc["device_key"]}
			}
		}
	}
	if asBool(bindingData["auto_assign"], false) && len(identityKeys) > 0 {
		usable := []map[string]any{}
		for _, a := range accounts {
			if subAccountUsableForPermission(a, permission) {
				usable = append(usable, a)
			}
		}
		if len(usable) > 0 {
			lastIndex := qInt(bindingData["last_index"])
			next := (lastIndex + 1) % len(usable)
			account := usable[next]
			identity := identityKeys[0]
			bindingData["last_index"] = next
			bmap[trimStr(identity["key"])] = map[string]any{"sub_username": trimStr(account["username"]), "enabled": true, "remark": "auto assigned", "created_at": tsDateTime(), "updated_at": tsDateTime()}
			saveSubAccountBindings(bindingData)
			dc := checkDeviceAllowed(map[string]any{"username": trimStr(account["username"]), "api_type": 1, "platform": firstNonNil(input["platform"], "android")}, reqCtx)
			if !asBool(dc["allowed"], true) {
				return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "username": trimStr(account["username"])}
			}
			return map[string]any{"ok": true, "account": account, "username": trimStr(account["username"]), "auth_source": "auto_assign", "assign_mode": "auto_assign", "binding_key": identity["key"], "identity_keys": identityKeys, "device_key": dc["device_key"]}
		}
	}
	return map[string]any{"ok": false, "code": "403", "msg": "未分配可用子账号", "identity_keys": identityKeys}
}

var _ = time.Now
