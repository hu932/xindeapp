package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// handleUpload：加密/明文数据上传转发（旧 App 上传主路径，也是高峰期“卡住”的核心）
func handleUpload(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, rawBody string) {
	apiType := qInt(input["api_type"])
	encryptedData := input["data"]
	dataStr, isStr := encryptedData.(string)
	if apiType == 0 || apiType != 1 {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "无效api_type"})
		return
	}
	if !isStr || dataStr == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "缺少data"})
		return
	}

	dc := checkDeviceAllowed(input, reqCtx)
	if !asBool(dc["allowed"], true) {
		logEntry := uploadBaseLog(reqCtx, input, apiType, dc)
		logEntry["encrypted_len"] = len(dataStr)
		logEntry["raw_body_len"] = len(rawBody)
		logEntry["status"] = "device_disabled"
		logEntry["error"] = firstNonNilStr(dc["msg"], "设备已被禁用")
		appendLog(logFile, logEntry, maxLogs)
		recordStats(statsDir, input, apiType, "device_disabled", reqCtx)
		writeJSONOK(w, map[string]any{"ok": false, "msg": firstNonNilStr(dc["msg"], "设备已被禁用")})
		return
	}

	logEntry := uploadBaseLog(reqCtx, input, apiType, dc)
	logEntry["encrypted_len"] = len(dataStr)
	logEntry["raw_body_len"] = len(rawBody)
	logEntry["account_remark"] = displayAccountRemark(apiType, trimStr(logEntry["username"]), trimStr(logEntry["group_id"]))

	targetURL := api1SubmitURL

	// 明文 JSON：直接转发
	var plainInput map[string]any
	if json.Unmarshal([]byte(dataStr), &plainInput) == nil {
		taskURL := trimStr(input["url"])
		if taskURL == "" {
			itemID := trimStr(input["item_id"])
			shopID := trimStr(input["shop_id"])
			if itemID != "" && shopID != "" {
				taskURL = "https://shopee.tw/api/v4/pdp/get_pc?display_model_id=0&item_id=" + urlQueryEscape(itemID) + "&model_selection_logic=3&shop_id=" + urlQueryEscape(shopID) + "&tz_offset_in_minutes=480&detail_level=0"
			}
		}
		if taskURL == "" {
			logEntry["status"] = "plain_missing_url"
			appendLog(logFile, logEntry, maxLogs)
			writeJSONOK(w, map[string]any{"ok": false, "msg": "plain api1 missing url"})
			return
		}
		payload := map[string]any{"appVersion": firstNonNilStr(input["appVersion"], "vv2"), "url": taskURL, "result": dataStr}
		forwardHeaders := []string{}
		apiInputToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
		if apiInputToken != "" {
			forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+apiInputToken)
			logEntry["api_token_source"] = "input"
		}
		logEntry["task_url"] = taskURL
		logEntry["source"] = "plain_data"
		forwardBody := jsonEncode(payload)
		res := forwardToServer(targetURL, forwardBody, submitGzip, forwardHeaders)
		var serverResp map[string]any
		_ = jsonDecode(res.Response, &serverResp)
		isForwardSuccess := api1ForwardSuccess(serverResp)
		logEntry["forward_url"] = targetURL
		logEntry["forward_attempts"] = []map[string]any{forwardAttemptSummary(res, serverResp)}
		logEntry["status"] = condStr(res.Error != "", "forward_fail", condStr(isForwardSuccess, "success", "task_server_error"))
		logEntry["response"] = compactLogResponse(serverResp, maxLogRespBytes)
		if res.Error != "" {
			logEntry["error"] = res.Error
		}
		uploadID := dateTimeStamp() + "_" + md5short(rawBody+"|plain_upload")
		logEntry["upload_id"] = uploadID
		appendLog(logFile, logEntry, maxLogs)
		recordStats(statsDir, input, apiType, trimStr(logEntry["status"]), reqCtx)
		if trimStr(logEntry["status"]) != "success" {
			saveUploadDump(uploadID, map[string]any{
				"api_type": apiType, "source": "plain_data", "target_url": targetURL, "method": "POST",
				"content_encoding": condStr(submitGzip, "gzip", "identity"), "request_body": payload,
				"http_code": res.HTTPCode, "response_json": serverResp, "response_raw": res.Response, "error": res.Error,
			})
		}
		if res.Error != "" {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "forward failed: " + res.Error})
			return
		}
		if serverResp != nil {
			writeJSONOK(w, serverResp)
			return
		}
		writeJSONOK(w, map[string]any{"ok": false, "msg": "task server response invalid", "raw": res.Response})
		return
	}

	// AES 解密
	plain, ok := aesDecryptString(dataStr, aesKey)
	if !ok {
		logEntry["status"] = "decrypt_fail"
		appendLog(logFile, logEntry, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "msg": "AES解密失败"})
		return
	}
	var decrypted map[string]any
	if json.Unmarshal([]byte(plain), &decrypted) != nil {
		logEntry["status"] = "json_fail"
		dumpID := saveDecryptDump(dataStr, plain, apiType, nil, nil, nil)
		logEntry["dump_id"] = dumpID
		logEntry["preview"] = truncate(plain, 200)
		appendLog(logFile, logEntry, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "msg": "解密后非JSON"})
		return
	}
	logEntry["decrypted_len"] = len(plain)

	var responseBodyRaw string
	var productResponse map[string]any
	extractFrom := "missing_responseContent.response_body"
	if rc, ok := decrypted["responseContent"].(map[string]any); ok {
		if rb, ok2 := rc["response_body"].(string); ok2 && rb != "" {
			responseBodyRaw = rb
			var parsed map[string]any
			if json.Unmarshal([]byte(rb), &parsed) == nil {
				productResponse = parsed
				extractFrom = "responseContent.response_body(object)"
			} else {
				extractFrom = "raw_wrapper(response_body_not_json)"
			}
		}
	}
	logEntry["extract_from"] = extractFrom

	taskURL := trimStr(input["url"])
	if taskURL == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "接口1缺少url"})
		return
	}
	logEntry["task_url"] = taskURL
	if responseBodyRaw == "" {
		logEntry["status"] = "extract_fail"
		dumpID := saveDecryptDump(dataStr, plain, apiType, decrypted, productResponse, nil)
		logEntry["dump_id"] = dumpID
		appendLog(logFile, logEntry, maxLogs)
		writeJSONOK(w, map[string]any{"ok": false, "msg": "未提取到responseContent.response_body，接口1不会上传完整wrapper"})
		return
	}
	payload := map[string]any{"appVersion": "vv2", "url": taskURL, "result": responseBodyRaw}
	forwardHeaders := []string{}
	apiUsingInputToken := false
	apiInputToken := trimStr(firstNonNil(input["auth_token"], input["token"]))
	if apiInputToken != "" {
		forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+apiInputToken)
		apiUsingInputToken = true
		logEntry["api_token_source"] = "app"
		if e := jwtExp(apiInputToken); e > 0 {
			logEntry["api_token_expires_at"] = timeNow().Add(time.Duration(int64(e)-timeNowUnix()) * time.Second).Format("2006-01-02 15:04:05")
		}
	} else {
		tokenResult := getAPIToken(api1Auth, api1TokenFile, false)
		if !tokenResult.OK {
			logEntry["status"] = "api1_login_fail"
			logEntry["error"] = firstNonNilStr(tokenResult.Msg, "接口1登录失败")
			appendLog(logFile, logEntry, maxLogs)
			writeJSONOK(w, map[string]any{"ok": false, "msg": logEntry["error"]})
			return
		}
		forwardHeaders = append(forwardHeaders, "Authorization: Bearer "+tokenResult.Token)
		logEntry["api_token_source"] = "proxy"
		logEntry["api_token_cached"] = tokenResult.Cached
		logEntry["api_token_expires_at"] = timeNow().Add(time.Duration(tokenResult.ExpiresAt-timeNowUnix()) * time.Second).Format("2006-01-02 15:04:05")
	}
	logEntry["forward_result_from"] = "responseContent.response_body"
	logEntry["forward_result_len"] = len(responseBodyRaw)
	forwardBody := jsonEncode(payload)
	logEntry["forward_url"] = targetURL

	var forwardAttempts []map[string]any
	var res forwardResult
	var serverResp map[string]any
	maxAttempts := api1ParseMaxRetry
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		res = forwardToServer(targetURL, forwardBody, submitGzip, forwardHeaders)
		_ = jsonDecode(res.Response, &serverResp)
		summary := forwardAttemptSummary(res, serverResp)
		summary["attempt"] = attempt
		forwardAttempts = append(forwardAttempts, summary)
		if !api1ParseTimeoutResponse(serverResp, res.Response) || api1ForwardSuccess(serverResp) {
			break
		}
	}
	logEntry["forward_attempts"] = forwardAttempts

	// token 过期重试
	if !apiUsingInputToken && serverResp != nil && strings.Contains(toLower(trimStr(serverResp["msg"])), "login") {
		tokenResult := getAPIToken(api1Auth, api1TokenFile, true)
		if tokenResult.OK {
			forwardHeaders = []string{"Authorization: Bearer " + tokenResult.Token}
			res = forwardToServer(targetURL, forwardBody, submitGzip, forwardHeaders)
			_ = jsonDecode(res.Response, &serverResp)
			logEntry["api_token_refreshed"] = true
			summary := forwardAttemptSummary(res, serverResp)
			summary["attempt"] = "token_refresh"
			forwardAttempts = append(forwardAttempts, summary)
		}
	}

	if res.Error != "" {
		logEntry["status"] = "forward_fail"
		logEntry["error"] = res.Error
		uploadID := trimStr(logEntry["dump_id"])
		if uploadID == "" {
			uploadID = dateTimeStamp() + "_" + md5short(dataStr+"|upload")
		}
		logEntry["upload_id"] = uploadID
		appendLog(logFile, logEntry, maxLogs)
		recordStats(statsDir, input, apiType, "forward_fail", reqCtx)
		saveUploadDump(uploadID, uploadDetail(targetURL, forwardBody, res, serverResp, submitGzip))
		writeJSONOK(w, map[string]any{"ok": false, "msg": "转发失败: " + res.Error})
		return
	}
	isForwardSuccess := api1ForwardSuccess(serverResp)
	if !isForwardSuccess && api1ParseTimeoutResponse(serverResp, res.Response) {
		logEntry["error"] = "api1 parse timeout, auto retry attempts: " + strconvItoa(len(forwardAttempts))
	}
	logEntry["status"] = condStr(isForwardSuccess, "success", "task_server_error")
	logEntry["response"] = compactLogResponse(serverResp, maxLogRespBytes)
	uploadID := trimStr(logEntry["dump_id"])
	if trimStr(logEntry["status"]) != "success" || uploadID == "" {
		if uploadID == "" {
			uploadID = dateTimeStamp() + "_" + md5short(dataStr+"|upload")
		}
		logEntry["upload_id"] = uploadID
	}
	appendLog(logFile, logEntry, maxLogs)
	recordStats(statsDir, input, apiType, trimStr(logEntry["status"]), reqCtx)
	if trimStr(logEntry["status"]) != "success" {
		saveUploadDump(uploadID, uploadDetail(targetURL, forwardBody, res, serverResp, submitGzip))
	}
	if serverResp != nil {
		serverResp["_proxy"] = map[string]any{
			"submit_accepted": isForwardSuccess, "counted": isForwardSuccess, "status": logEntry["status"], "api_type": apiType,
		}
		writeJSONOK(w, serverResp)
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "任务服务器响应异常", "raw": res.Response})
}

func uploadBaseLog(reqCtx *httpRequestCtx, input map[string]any, apiType int, dc map[string]any) map[string]any {
	return map[string]any{
		"api_type": apiType, "ip": reqCtx.ip,
		"username":   firstNonNilStr(input["username"], input["用户名"]),
		"group_id":   firstNonNilStr(input["group_id"], input["组ID"]),
		"device_id":  firstNonNilStr(input["device_id"], input["设备ID"]),
		"platform":   trimStr(input["platform"]), "device_type": trimStr(input["device_type"]),
		"device_label": trimStr(input["device_label"]), "client": trimStr(input["client"]),
		"client_label": resolveClientLabel(input),
		"fingerprint_key": trimStr(input["fingerprint_key"]), "fingerprint_id": trimStr(input["fingerprint_id"]), "fingerprint_name": trimStr(input["fingerprint_name"]),
		"device_key": trimStr(dc["device_key"]),
	}
}

func uploadDetail(targetURL, forwardBody string, res forwardResult, serverResp map[string]any, gzipBody bool) map[string]any {
	enc := "identity"
	if gzipBody {
		enc = "gzip"
	}
	return map[string]any{
		"time": tsDateTime(), "api_type": 1, "target_url": targetURL, "method": "POST",
		"content_type": "application/json; charset=utf-8", "content_encoding": enc,
		"request_body": mustJSONMap(forwardBody), "http_code": res.HTTPCode,
		"response_json": serverResp, "response_raw": res.Response, "error": res.Error,
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// saveDecryptDump：解密 dump（仅失败/抽样时保存）
func saveDecryptDump(encryptedData, plain string, apiType int, decrypted, productData, forwardPayload any) string {
	dir := dp("decrypt_dumps")
	_ = mkdirAll(dir)
	id := dateTimeStamp() + "_" + md5short(encryptedData)
	detail := map[string]any{
		"dump_time": tsDateTime(), "api_type": apiType, "encrypted_len": len(encryptedData), "decrypted_len": len(plain),
		"decrypted_full": decrypted, "product_data": productData, "forward_payload": forwardPayload,
	}
	_ = atomicWrite(dir+"/"+id+".json", jsonEncode(detail))
	return id
}

func mkdirAll(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
