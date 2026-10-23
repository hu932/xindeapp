package main

import (
	"net/http"
	"strings"
)

// api1LoginWithPassword：直接转发登录（返回 raw response）
func api1LoginWithPassword(url, username, password string) forwardResult {
	body := jsonEncode(map[string]string{"username": username, "password": password})
	return doForward(http.MethodPost, url, body, false, []string{"Accept: */*"}, loginTimeout)
}

func handleAPI1Login(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	username := trimStr(firstNonNil(input["username"], input["accountId"], input["account_id"], input["id"]))
	password := qStr(input["password"])
	if account := decodedExtAccountFromInput(map[string]any{"username": username, "token": firstNonNil(input["auth_token"], input["token"])}, r); account != nil {
		writeJSONOK(w, decodedExtLoginResponseForAccount(account))
		return
	}
	if password == "" && username != "" {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "下发ID不存在或未启用"})
		return
	}
	if username == "" || password == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "账号或密码为空"})
		return
	}
	decodedExtSyncAllTaskAccountsToWhitelists()
	allowed := readAllowedAccounts(1)
	if !containsStr(accountNames(allowed), username) {
		appendLog(logFile, map[string]any{"api_type": 1, "ip": reqCtx.ip, "username": username, "status": "login_rejected", "error": "账号未在后台聚星白名单"}, maxLogs)
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
		return
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 1}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	loginResult := api1LoginWithPassword(api1Auth.LoginURL, username, password)
	if loginResult.Error != "" {
		writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "登录转发失败: " + loginResult.Error})
		return
	}
	var loginJSON map[string]any
	_ = jsonDecode(loginResult.Response, &loginJSON)
	if loginJSON != nil {
		if data, ok := loginJSON["data"].(map[string]any); ok {
			if token := trimStr(data["token"]); token != "" {
				cacheAPITokenUsername(token, username, int64(jwtExp(token)))
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(loginResult.Response))
}

// handleAPI2Login：接口2（zb2）登录。
// 两种用法：
//  1. 下发id（插件账号）登录——只认下发id，不要求接口2账号密码，直接发下发id token；
//  2. 老客户端透传登录——把接口2账号密码原样转发给上游 zb2，并缓存 token。
func handleAPI2Login(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	username := trimStr(firstNonNil(input["username"], input["accountId"], input["account_id"], input["id"]))
	password := qStr(input["password"])
	if account := decodedExtAccountFromInput(map[string]any{"username": username, "token": firstNonNil(input["auth_token"], input["token"])}, r); account != nil {
		writeJSONOK(w, decodedExtLoginResponseForAccount(account))
		return
	}
	if password == "" && username != "" {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "下发ID不存在或未启用"})
		return
	}
	if username == "" || password == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "账号或密码为空"})
		return
	}
	decodedExtSyncAllTaskAccountsToWhitelists()
	allowed := readAllowedAccounts(2)
	if !containsStr(accountNames(allowed), username) {
		appendLog(logFile, map[string]any{"api_type": 2, "ip": reqCtx.ip, "username": username, "status": "login_rejected", "error": "账号未在后台接口2白名单"}, maxLogs)
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
		return
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 2}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	loginResult := api1LoginWithPassword(api2Auth.LoginURL, username, password)
	if loginResult.Error != "" {
		writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "登录转发失败: " + loginResult.Error})
		return
	}
	var loginJSON map[string]any
	_ = jsonDecode(loginResult.Response, &loginJSON)
	if loginJSON != nil {
		if data, ok := loginJSON["data"].(map[string]any); ok {
			if token := trimStr(data["token"]); token != "" {
				cacheAPITokenUsername(token, username, int64(jwtExp(token)))
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(loginResult.Response))
}

func decodedExtTakeByAccountID(pluginAccount, input map[string]any, apiType int, reqCtx *httpRequestCtx) map[string]any {
	pluginUsername := trimStr(pluginAccount["accountId"])
	takeType := strings.ToLower(strings.TrimSpace(firstNonNilStr(input["type"], input["task_type"])))
	deviceContext := decodedExtDeviceContext(input, pluginUsername)
	dcInput := mergeMaps(deviceContext, input)
	dcInput["username"] = pluginUsername
	dcInput["api_type"] = apiType
	if apiType == 0 {
		dcInput["api_type"] = 1
	}
	dc := checkDeviceAllowed(dcInput, reqCtx)
	if !asBool(dc["allowed"], true) {
		appendLog(logFile, map[string]any{
			"api_type": apiType, "ip": reqCtx.ip, "username": pluginUsername, "device_id": deviceContext["device_id"],
			"fingerprint_key": deviceContext["fingerprint_key"], "status": "device_disabled", "source": "account_id_take",
			"error": firstNonNilStr(dc["msg"], "设备已被禁用"),
		}, maxLogs)
		return map[string]any{"code": "403", "success": false, "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")}
	}
	runtimeAccount := decodedExtRuntimeAccount(pluginAccount, deviceContext, "")
	requestClientType := decodedExtRequestClientType(input, deviceContext)
	isIOSSingleTake := requestClientType == "ios" || asBool(input["ios_single_take"], false)
	var taskAccountPool []map[string]any
	if apiType > 0 {
		taskAccountPool = decodedExtEnabledTaskAccountsByType(pluginAccount, apiType)
	} else {
		taskAccountPool = decodedExtEnabledTaskAccounts(pluginAccount)
	}
	if len(taskAccountPool) == 0 {
		if apiType > 0 {
			return map[string]any{"code": "500", "success": false, "data": nil, "msg": "接口" + strconvItoa(apiType) + "账号池为空"}
		}
		return map[string]any{"code": "500", "success": false, "data": nil, "msg": "账号池为空"}
	}
	takeLimit := max(1, len(taskAccountPool))
	if isIOSSingleTake {
		takeLimit = 1
	}
	takeResult := decodedExtTakeTaskFromPluginPool(pluginAccount, takeLimit, takeType, apiType, isIOSSingleTake, false)
	task, _ := takeResult["task"].(map[string]any)
	taskAccount, _ := takeResult["task_account"].(map[string]any)
	takeToken := qStr(takeResult["api1_token"])
	if task == nil || taskAccount == nil {
		appendLog(logFile, map[string]any{
			"api_type": apiType, "ip": reqCtx.ip, "username": pluginUsername, "device_id": deviceContext["device_id"],
			"fingerprint_key": deviceContext["fingerprint_key"], "status": "no_task", "source": "account_id_take",
			"forward_attempts": takeResult["attempts"], "response": compactLogResponse(takeResult["last_msg"], maxLogRespBytes),
		}, maxLogs)
		return map[string]any{"code": "404", "success": false, "data": nil, "msg": firstNonNilStr(takeResult["last_msg"], "no task")}
	}
	decodedExtRecordTaskLease(runtimeAccount, task, taskAccount, takeToken)
	task = decodedExtTaskWithAccountMeta(task, taskAccount)
	task = decodedExtTaskWithTakeTokenMeta(task, takeToken)
	task["task_runtime_scope"] = trimStr(runtimeAccount["__storage_scope"])
	task["taskRuntimeScope"] = trimStr(runtimeAccount["__storage_scope"])
	taskAPIType := decodedExtAPITypeForTaskAccount(taskAccount)
	taskAccountUsername := trimStr(taskAccount["username"])
	appendLog(logFile, map[string]any{
		"api_type": taskAPIType, "ip": reqCtx.ip, "username": pluginUsername,
		"api1_username": condStr(taskAPIType == 1, taskAccountUsername), "api2_username": condStr(taskAPIType == 2, taskAccountUsername),
		"task_account_username": taskAccountUsername, "task_api_type": taskAPIType,
		"device_id": deviceContext["device_id"], "fingerprint_key": deviceContext["fingerprint_key"],
		"runtime_scope": trimStr(runtimeAccount["__storage_scope"]), "status": "success", "source": "account_id_take",
		"forward_url": apiTakeURLForType(taskAPIType, takeType), "task_type": takeType,
		"task_id": decodedExtTaskDealID(task), "task_url": decodedExtTaskURL(task), "forward_attempts": takeResult["attempts"],
	}, maxLogs)
	// 兼容网页插件：插件从 data.taskUrl / data.taskId 取任务，这里显式补齐
	taskURL := decodedExtTaskURL(task)
	if taskURL != "" {
		task["taskUrl"] = taskURL
		task["url"] = firstNonNilStr(task["url"], taskURL)
	}
	if taskID := decodedExtTaskDealID(task); taskID != "" {
		task["taskId"] = taskID
	}
	return map[string]any{
		"code": "200", "success": true, "data": task, "task": task, "msg": nil,
		"_proxy": map[string]any{"accountId": pluginUsername, "api_type": taskAPIType, "task_account_username": taskAccountUsername, "status": "success"},
	}
}

func handleAPI1Take(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	username := trimStr(input["username"])
	takeType := strings.ToLower(strings.TrimSpace(firstNonNilStr(input["type"], input["task_type"])))
	takeURL := api1TakeURL
	if takeType == "r" {
		takeURL = api1RTakeURL
	}
	takeToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if takeToken == "" {
		takeToken = bearerToken(r)
	}
	if username == "" && takeToken != "" {
		username = resolveAPIUsernameFromToken(takeToken)
		if username != "" {
			input["username"] = username
		}
	}
	if account := decodedExtAccountFromInput(map[string]any{"username": username, "token": takeToken}, r); account != nil {
		writeJSONOK(w, decodedExtTakeByAccountID(account, map[string]any{"username": trimStr(account["accountId"])}, 1, reqCtx))
		return
	}
	if username == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "账号为空"})
		return
	}
	decodedExtSyncAllTaskAccountsToWhitelists()
	allowed := readAllowedAccounts(1)
	if !containsStr(accountNames(allowed), username) {
		appendLog(logFile, map[string]any{"api_type": 1, "ip": reqCtx.ip, "username": username, "status": "take_rejected", "source": resolveLogSource(input, "api1_take", "web_take", "ctrl_take"), "error": "账号未在后台聚星白名单"}, maxLogs)
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
		return
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 1}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	if takeToken == "" {
		writeJSONOK(w, map[string]any{"code": "401", "data": nil, "msg": "登录已失效，请重新登录"})
		return
	}
	res := forwardGetToServer(takeURL, []string{"Authorization: Bearer " + takeToken})
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	takeOK := false
	if serverResp != nil {
		data, _ := serverResp["data"].(map[string]any)
		takeOK = trimStr(serverResp["code"]) == "200" && data != nil && trimStr(data["taskUrl"]) != ""
	}
	appendLog(logFile, map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": username, "account_remark": displayAccountRemark(1, username, ""),
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": condStr(res.Error != "", "forward_fail", condStr(takeOK, "success", "task_server_error")),
		"source": resolveLogSource(input, "api1_take", "web_take", "ctrl_take"), "forward_url": takeURL, "task_type": takeType,
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes),
	}, maxLogs)
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "取任务转发失败: " + res.Error})
		return
	}
	if serverResp != nil {
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "任务服务器响应异常", "raw": res.Response})
}

// handleAPI2Take：接口2（zb2）取任务。
// 下发id 命中插件账号时走「接口2账号池」轮询；否则按老客户端透传方式转发给上游 zb2。
func handleAPI2Take(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any) {
	username := trimStr(firstNonNil(input["username"], input["accountId"], input["account_id"]))
	takeType := strings.ToLower(strings.TrimSpace(firstNonNilStr(input["type"], input["task_type"])))
	takeURL := api2TakeURL
	if takeType == "r" {
		takeURL = api2RTakeURL
	}
	takeToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if takeToken == "" {
		takeToken = bearerToken(r)
	}
	if username == "" && takeToken != "" {
		username = resolveAPIUsernameFromToken(takeToken)
		if username != "" {
			input["username"] = username
		}
	}
	if account := decodedExtAccountFromInput(map[string]any{"username": username, "token": takeToken}, r); account != nil {
		writeJSONOK(w, decodedExtTakeByAccountID(account, mergeMaps(input, map[string]any{"username": trimStr(account["accountId"]), "type": takeType}), 2, reqCtx))
		return
	}
	if username == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "账号为空"})
		return
	}
	decodedExtSyncAllTaskAccountsToWhitelists()
	allowed := readAllowedAccounts(2)
	if !containsStr(accountNames(allowed), username) {
		appendLog(logFile, map[string]any{"api_type": 2, "ip": reqCtx.ip, "username": username, "status": "take_rejected", "source": resolveLogSource(input, "api2_take", "web_take", "ctrl_take"), "error": "账号未在后台接口2白名单"}, maxLogs)
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
		return
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 2}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	if takeToken == "" {
		writeJSONOK(w, map[string]any{"code": "401", "data": nil, "msg": "登录已失效，请重新登录"})
		return
	}
	res := forwardGetToServer(takeURL, []string{"Authorization: Bearer " + takeToken})
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	takeOK := false
	if serverResp != nil {
		data, _ := serverResp["data"].(map[string]any)
		takeOK = trimStr(serverResp["code"]) == "200" && data != nil && trimStr(data["taskUrl"]) != ""
	}
	appendLog(logFile, map[string]any{
		"api_type": 2, "ip": reqCtx.ip, "username": username, "account_remark": displayAccountRemark(2, username, ""),
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"status": condStr(res.Error != "", "forward_fail", condStr(takeOK, "success", "task_server_error")),
		"source": resolveLogSource(input, "api2_take", "web_take", "ctrl_take"), "forward_url": takeURL, "task_type": takeType,
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes),
	}, maxLogs)
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "取任务转发失败: " + res.Error})
		return
	}
	if serverResp != nil {
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"code": "500", "data": nil, "msg": "任务服务器响应异常", "raw": res.Response})
}

func handleAPI1Submit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	username := trimStr(input["username"])
	submitURL := trimStr(input["url"])
	submitResult := anyToStr(input["result"])
	submitID := trimStr(input["submit_id"])
	taskID := trimStr(firstNonNil(input["task_id"], input["trace_id"]))
	itemID := trimStr(input["item_id"])
	shopID := trimStr(input["shop_id"])
	submitToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if submitToken == "" {
		submitToken = bearerToken(r)
	}
	if username == "" && submitToken != "" {
		username = resolveAPIUsernameFromToken(submitToken)
		if username != "" {
			input["username"] = username
		}
	}
	if username != "" {
		decodedExtSyncAllTaskAccountsToWhitelists()
		allowed := readAllowedAccounts(1)
		if !containsStr(accountNames(allowed), username) {
			appendLog(logFile, map[string]any{"api_type": 1, "ip": reqCtx.ip, "username": username, "status": "submit_rejected", "source": resolveLogSource(input, "api1_submit", "web_submit", "ctrl_submit"), "error": "账号未在后台聚星白名单"}, maxLogs)
			writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
			return
		}
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 1}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	if submitURL == "" || submitResult == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "提交数据为空"})
		return
	}
	forwardBody := jsonEncode(map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": submitURL, "result": submitResult})
	forwardHeaders := []string{}
	if submitToken != "" {
		forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+submitToken)
	}
	res := forwardToServer(api1SubmitURL, forwardBody, submitGzip, forwardHeaders)
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	isForwardSuccess := api1ForwardSuccess(serverResp)
	if !isForwardSuccess && api1ParseTimeoutResponse(serverResp, res.Response) {
		isForwardSuccess = false
	}
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
	sourceLabel := resolveLogSource(input, "ios_submit", "web_submit", "ctrl_submit")
	uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|"+sourceLabel)
	status := "task_server_error"
	if res.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	logEntry := map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": username, "account_remark": displayAccountRemark(1, username, ""),
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"], "device_type": input["device_type"], "platform": input["platform"],
		"client": input["client"], "client_label": resolveClientLabel(input), "status": status, "source": sourceLabel,
		"task_url": submitURL, "forward_url": api1SubmitURL, "forward_result_len": len(submitResult),
		"submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "upload_id": uploadID,
	}
	if res.Error != "" {
		logEntry["error"] = res.Error
	}
	appendLog(logFile, logEntry, maxLogs)
	recordStats(statsDir, input, 1, status, reqCtx)
	saveUploadDump(uploadID, map[string]any{
		"api_type": 1, "source": sourceLabel, "target_url": api1SubmitURL, "method": "POST",
		"content_type": "application/json; charset=utf-8", "content_encoding": "identity",
		"request_body": mustJSONMap(forwardBody), "http_code": res.HTTPCode, "response_json": serverResp, "response_raw": res.Response, "error": res.Error,
		"submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
	})
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "转发失败: " + res.Error})
		return
	}
	if serverResp != nil && proxyContextMatch {
		serverResp["_proxy"] = map[string]any{
			"submit_id": submitID, "task_id": taskID, "server_task_id": serverTaskID, "server_submit_id": serverSubmitID,
			"context_match": proxyContextMatch, "submit_accepted": isForwardSuccess, "counted": isForwardSuccess, "status": status,
		}
		writeJSONOK(w, serverResp)
		return
	}
	if serverResp != nil && !proxyContextMatch {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "上游未返回当前任务回执", "upstream": serverResp, "_proxy": map[string]any{
			"submit_id": submitID, "task_id": taskID, "server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "context_match": false,
		}})
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "任务服务器响应异常", "raw": res.Response})
}

// handleAPI2Submit：接口2（zb2）回传。
// 下发id 命中插件账号时走 decodedExtAPISubmit（租约 token + 结果清洗）；否则按老客户端透传转发。
func handleAPI2Submit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	username := trimStr(firstNonNil(input["username"], input["accountId"], input["account_id"]))
	submitToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if submitToken == "" {
		submitToken = bearerToken(r)
	}
	if username == "" && submitToken != "" {
		username = resolveAPIUsernameFromToken(submitToken)
		if username != "" {
			input["username"] = username
		}
	}
	if account := decodedExtAccountFromInput(map[string]any{"username": username, "token": submitToken}, r); account != nil {
		deviceContext := decodedExtDeviceContext(input, trimStr(account["accountId"]))
		runtimeAccount := decodedExtRuntimeAccount(account, deviceContext, "")
		decodedExtAPISubmit(w, r, reqCtx, input, account, trimStr(account["accountId"]), false, deviceContext, runtimeAccount, decodedExtEnabledTaskAccounts(account))
		return
	}
	submitURL := firstNonNilStr(input["url"], input["product_url"], input["pdp_url"])
	submitResult := anyToStr(firstNonNil(input["result"], input["rawResponseText"]))
	submitID := trimStr(input["submit_id"])
	taskID := trimStr(firstNonNil(input["task_id"], input["trace_id"], input["deal_id"]))
	itemID := trimStr(input["item_id"])
	shopID := trimStr(input["shop_id"])
	if username != "" {
		decodedExtSyncAllTaskAccountsToWhitelists()
		allowed := readAllowedAccounts(2)
		if !containsStr(accountNames(allowed), username) {
			appendLog(logFile, map[string]any{"api_type": 2, "ip": reqCtx.ip, "username": username, "status": "submit_rejected", "source": resolveLogSource(input, "api2_submit", "web_submit", "ctrl_submit"), "error": "账号未在后台接口2白名单"}, maxLogs)
			writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "账号未授权，请联系管理员"})
			return
		}
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 2}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}
	if submitURL == "" || submitResult == "" {
		writeJSONOK(w, map[string]any{"code": "400", "data": nil, "msg": "提交数据为空"})
		return
	}
	// 接口2 结果清洗：null 字段置空（与 PHP 一致）
	submitResult, api2NullFields := api2SanitizeSubmitResult(submitResult)
	forwardBody := jsonEncode(map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": submitURL, "result": submitResult})
	forwardHeaders := []string{}
	if submitToken == "" {
		if tokenInfo := getAPIToken(api2Auth, api2TokenFile, false); tokenInfo.OK && tokenInfo.Token != "" {
			submitToken = tokenInfo.Token
		}
	}
	if submitToken != "" {
		forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+submitToken)
	}
	res := forwardToServer(api2SubmitURL, forwardBody, submitGzip, forwardHeaders)
	var serverResp map[string]any
	_ = jsonDecode(res.Response, &serverResp)
	isForwardSuccess := apiSubmitAcceptedForType(2, serverResp)
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
	sourceLabel := resolveLogSource(input, "ios_submit", "web_submit", "ctrl_submit")
	uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|"+sourceLabel+"|api2")
	status := "task_server_error"
	if res.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	logEntry := map[string]any{
		"api_type": 2, "ip": reqCtx.ip, "username": username, "account_remark": displayAccountRemark(2, username, ""),
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"], "device_type": input["device_type"], "platform": input["platform"],
		"client": input["client"], "client_label": resolveClientLabel(input), "status": status, "source": sourceLabel,
		"task_url": submitURL, "forward_url": api2SubmitURL, "forward_result_len": len(submitResult),
		"submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID, "api2_null_fields": api2NullFields,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
		"forward_attempts": []map[string]any{forwardAttemptSummary(res, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "upload_id": uploadID,
	}
	if res.Error != "" {
		logEntry["error"] = res.Error
	}
	appendLog(logFile, logEntry, maxLogs)
	recordStats(statsDir, input, 2, status, reqCtx)
	saveUploadDump(uploadID, map[string]any{
		"api_type": 2, "source": sourceLabel, "target_url": api2SubmitURL, "method": "POST",
		"content_type": "application/json; charset=utf-8", "content_encoding": "identity",
		"request_body": mustJSONMap(forwardBody), "http_code": res.HTTPCode, "response_json": serverResp, "response_raw": res.Response, "error": res.Error,
		"submit_id": submitID, "task_id": taskID, "shop_id": shopID, "item_id": itemID,
		"server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "proxy_context_match": proxyContextMatch,
	})
	if res.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "转发失败: " + res.Error})
		return
	}
	if serverResp != nil && proxyContextMatch {
		serverResp["_proxy"] = map[string]any{
			"submit_id": submitID, "task_id": taskID, "server_task_id": serverTaskID, "server_submit_id": serverSubmitID,
			"context_match": proxyContextMatch, "submit_accepted": isForwardSuccess, "counted": isForwardSuccess, "status": status,
			"api_type": 2,
		}
		writeJSONOK(w, serverResp)
		return
	}
	if serverResp != nil && !proxyContextMatch {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "上游未返回当前任务回执", "upstream": serverResp, "_proxy": map[string]any{
			"submit_id": submitID, "task_id": taskID, "server_task_id": serverTaskID, "server_submit_id": serverSubmitID, "context_match": false,
		}})
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "任务服务器响应异常", "raw": res.Response})
}

// apiSubmitAcceptedForType：接口1 沿用严格口径（code=200 且 data.code=SUCCESS 且 data.msg 为 null）；
// 接口2 上游回执口径较松，额外接受 success/ok=true 或「code=200 且没有任何错误信息」。
func apiSubmitAcceptedForType(apiType int, serverResp map[string]any) bool {
	if api1ForwardSuccess(serverResp) {
		return true
	}
	if apiType != 2 || serverResp == nil {
		return false
	}
	if asBool(serverResp["success"], false) || asBool(serverResp["ok"], false) {
		return true
	}
	if trimStr(firstNonNil(serverResp["msg"], serverResp["error"])) != "" {
		return false
	}
	if data, ok := serverResp["data"].(map[string]any); ok {
		if trimStr(firstNonNil(data["msg"], data["error"])) != "" {
			return false
		}
	}
	return trimStr(serverResp["code"]) == "200"
}

// androidSessionAllowedTaskUrl
func androidSessionAllowedTaskURL(url string) bool {
	u, err := parseURL(url)
	if err != nil || u.Host == "" {
		return false
	}
	host := toLower(u.Host)
	scheme := toLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	return strings.HasSuffix(host, ".shopee.tw") || strings.Contains(host, ".shopee.") ||
		strings.HasSuffix(host, ".shp.ee") || strings.HasSuffix(host, ".shopeepay.tw") || strings.HasSuffix(host, ".iftyvkhj888.com") ||
		host == "shopee.tw" || host == "shopee.com" || host == "shp.ee"
}

func handleAndroidSessionFetchSubmit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	username := trimStr(input["username"])
	taskURL := trimStr(firstNonNil(input["url"], input["task_url"]))
	taskID := trimStr(firstNonNil(input["task_id"], input["trace_id"]))
	cookies := trimStr(firstNonNil(input["cookies"], input["cookie"]))
	userAgent := trimStr(firstNonNil(input["user_agent"], input["ua"]))
	submitToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if submitToken == "" {
		submitToken = bearerToken(r)
	}
	if username == "" && submitToken != "" {
		username = resolveAPIUsernameFromToken(submitToken)
		if username != "" {
			input["username"] = username
		}
	}
	if username == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "username empty"})
		return
	}
	if taskURL == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "task url empty"})
		return
	}
	if cookies == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "webview cookie empty"})
		return
	}
	if !androidSessionAllowedTaskURL(taskURL) {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "task url host not allowed"})
		return
	}
	isManagedAndroid := toLower(trimStr(input["client"])) == "ajie-android"
	if !isManagedAndroid {
		decodedExtSyncAllTaskAccountsToWhitelists()
		allowed := readAllowedAccounts(1)
		if !containsStr(accountNames(allowed), username) {
			writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": "account not authorized"})
			return
		}
	}
	dc := checkDeviceAllowed(map[string]any{"username": username, "api_type": 1}, reqCtx)
	if !asBool(dc["allowed"], true) {
		writeJSONOK(w, map[string]any{"code": "403", "data": nil, "msg": firstNonNilStr(dc["msg"], "device disabled")})
		return
	}
	fetchResult := androidFetchURLWithSession(taskURL, cookies, userAgent, nil)
	responseBodyRaw := fetchResult.Response
	var responseJSON map[string]any
	_ = jsonDecode(responseBodyRaw, &responseJSON)
	fetchOK := fetchResult.Error == "" && fetchResult.HTTPCode >= 200 && fetchResult.HTTPCode < 300 && responseBodyRaw != ""
	if !fetchOK {
		uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|android_session_fetch_fail")
		appendLog(logFile, map[string]any{
			"api_type": 1, "ip": reqCtx.ip, "username": username, "device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
			"platform": firstNonNil(input["platform"], "android"), "client": firstNonNil(input["client"], "android_plugin"), "client_label": resolveClientLabel(input),
			"status": "android_fetch_fail", "source": "android_session_fetch_submit", "task_url": taskURL, "task_id": taskID, "forward_url": taskURL,
			"http_code": fetchResult.HTTPCode, "error": firstNonNilStr(fetchResult.Error, "http "+strconvItoa(fetchResult.HTTPCode)),
			"response": compactLogResponse(responseJSON, maxLogRespBytes), "upload_id": uploadID,
		}, maxLogs)
		saveUploadDump(uploadID, map[string]any{
			"api_type": 1, "source": "android_session_fetch_submit", "target_url": taskURL, "method": "GET",
			"http_code": fetchResult.HTTPCode, "content_type": fetchResult.ContentType, "response_json": responseJSON, "response_raw": responseBodyRaw, "error": fetchResult.Error,
		})
		writeJSONOK(w, map[string]any{"ok": false, "msg": "android session fetch failed", "fetch": map[string]any{
			"http_code": fetchResult.HTTPCode, "error": fetchResult.Error, "content_type": fetchResult.ContentType,
		}})
		return
	}
	forwardBody := jsonEncode(map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": taskURL, "result": responseBodyRaw})
	if submitToken == "" {
		tokenInfo := getAPIToken(api1Auth, api1TokenFile, false)
		if tokenInfo.OK && tokenInfo.Token != "" {
			submitToken = tokenInfo.Token
		}
	}
	forwardHeaders := []string{}
	if submitToken != "" {
		forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+submitToken)
	}
	submitResult := forwardToServer(api1SubmitURL, forwardBody, submitGzip, forwardHeaders)
	var serverResp map[string]any
	_ = jsonDecode(submitResult.Response, &serverResp)
	isForwardSuccess := api1ForwardSuccess(serverResp)
	uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|android_session_submit")
	status := "task_server_error"
	if submitResult.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	logEntry := map[string]any{
		"api_type": 1, "ip": reqCtx.ip, "username": username, "account_remark": displayAccountRemark(1, username, ""),
		"device_id": input["device_id"], "fingerprint_key": input["fingerprint_key"],
		"platform": firstNonNil(input["platform"], "android"), "device_type": input["device_type"],
		"client": firstNonNil(input["client"], "android_plugin"), "client_label": resolveClientLabel(input),
		"status": status, "source": "android_session_fetch_submit", "task_url": taskURL, "task_id": taskID, "forward_url": api1SubmitURL,
		"fetch_http_code": fetchResult.HTTPCode, "fetch_result_len": len(responseBodyRaw), "forward_result_len": len(responseBodyRaw),
		"forward_attempts": []map[string]any{forwardAttemptSummary(submitResult, serverResp)},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "upload_id": uploadID,
	}
	if submitResult.Error != "" {
		logEntry["error"] = submitResult.Error
	}
	appendLog(logFile, logEntry, maxLogs)
	recordStats(statsDir, input, 1, status, reqCtx)
	saveUploadDump(uploadID, map[string]any{
		"api_type": 1, "source": "android_session_fetch_submit", "target_url": api1SubmitURL, "method": "POST",
		"request_body": mustJSONMap(forwardBody),
		"fetch": map[string]any{"url": taskURL, "http_code": fetchResult.HTTPCode, "content_type": fetchResult.ContentType, "response_json": responseJSON},
		"http_code": submitResult.HTTPCode, "response_json": serverResp, "response_raw": submitResult.Response, "error": submitResult.Error,
	})
	if submitResult.Error != "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "submit forward failed: " + submitResult.Error})
		return
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"source": "android_session_fetch_submit", "task_id": taskID, "fetch_http_code": fetchResult.HTTPCode,
			"submit_accepted": isForwardSuccess, "status": status,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "submit server abnormal", "raw": submitResult.Response})
}
