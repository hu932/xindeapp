package main

import (
	"os"
	"sort"
	"strings"
)

// decodedExtCxPayloadHasActivity
func decodedExtCxPayloadHasActivity(payload map[string]any) bool {
	for _, section := range []string{"batchItems", "items"} {
		for _, row := range rowsOf(payload, section) {
			total := qInt(row["takeCount"]) + qInt(row["completeCount"]) + qInt(row["checkSuccessCount"]) + qInt(row["checkFailCount"])
			if total > 0 {
				return true
			}
		}
	}
	return false
}

// decodedExtLoadCxPayloadForDate：直接调用 cx 查询引擎
func decodedExtLoadCxPayloadForDate(date string, force bool, apiType int, reqCtx *httpRequestCtx) map[string]any {
	if apiType == 2 {
		apiType = 2
	} else {
		apiType = 1
	}
	payload := CxQuery("cx.php", apiType, date, true, force, reqCtx)
	payload["_loaded_via"] = "local"
	return map[string]any{"ok": true, "payload": payload}
}

func decodedExtLoadLatestCxCachePayload(date string, apiType int) map[string]any {
	dir := dp(cxCacheDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return map[string]any{"ok": false, "msg": "cx cache missing"}
	}
	var bestPayload map[string]any
	bestScore := int64(0)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if !(strings.HasPrefix(name, "cx_cache_") || strings.HasPrefix(name, "cx_lastgood_") || strings.HasPrefix(name, "cx_date_lastgood_")) {
			continue
		}
		var wrapper map[string]any
		if !readJSONFile(dir+"/"+name, &wrapper) {
			continue
		}
		payload, _ := wrapper["payload"].(map[string]any)
		if payload == nil {
			payload = wrapper
		}
		if !decodedExtCxPayloadHasActivity(payload) {
			continue
		}
		if apiType > 0 {
			if t := qInt(firstNonNil(payload["apiType"], payload["api_type"], 1)); t != apiType {
				continue
			}
		}
		if pd := trimStr(payload["date"]); pd != "" && pd != date {
			continue
		}
		score := int64(qInt(payload["fetchTime"]))
		if ts := int64(qInt(wrapper["_ts"])); ts > score {
			score = ts
		}
		if fi, err := e.Info(); err == nil && fi.ModTime().Unix() > score {
			score = fi.ModTime().Unix()
		}
		if score >= bestScore {
			bestScore = score
			payload["fromCache"] = true
			payload["cacheAge"] = max(0, int(timeNowUnix()-score))
			payload["_loaded_via"] = "cx_cache"
			bestPayload = payload
		}
	}
	if bestPayload != nil {
		return map[string]any{"ok": true, "payload": bestPayload}
	}
	return map[string]any{"ok": false, "msg": "cx cache has no activity"}
}

func decodedExtParallelControlDailyStats(token, date string, force bool, reqCtx *httpRequestCtx) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return map[string]any{"ok": false, "msg": "缺少 token"}
	}
	if !regexpDate.MatchString(date) {
		date = tsDate()
	}
	account := decodedExtFindAccountByParallelControlToken(token)
	if account == nil {
		return map[string]any{"ok": false, "msg": "未找到控制账号"}
	}
	apiLabels := map[int]string{1: "接口1:登陆", 2: "接口2:非登"}
	taskAccounts := decodedExtNormalizeAllTaskAccounts(account, false)
	type meta struct {
		username string
		remark   string
		apiType  int
		label    string
	}
	allowed := map[string]meta{}
	allowedByAPI := map[int]map[string]meta{1: {}, 2: {}}
	for _, ta := range taskAccounts {
		if !asBool(ta["enabled"], true) || trimStr(ta["username"]) == "" {
			continue
		}
		username := trimStr(ta["username"])
		apiType := decodedExtAPITypeForTaskAccount(ta)
		m := meta{username: username, remark: trimStr(ta["remark"]), apiType: apiType, label: apiLabels[apiType]}
		allowed[strconvItoa(apiType)+"|"+username] = m
		allowedByAPI[apiType][username] = m
	}
	var loadedByAPI = map[int]map[string]any{}
	var loadErrors []string
	for _, apiType := range []int{1, 2} {
		if len(allowedByAPI[apiType]) == 0 {
			loadedByAPI[apiType] = map[string]any{"ok": true, "payload": map[string]any{"apiType": apiType, "items": []map[string]any{}, "batchItems": []map[string]any{}, "authorizedAccounts": map[string]any{}}}
			continue
		}
		loaded := decodedExtLoadCxPayloadForDate(date, force, apiType, reqCtx)
		if !asBool(loaded["ok"], true) {
			loaded = decodedExtLoadLatestCxCachePayload(date, apiType)
		}
		if !asBool(loaded["ok"], true) {
			loadErrors = append(loadErrors, apiLabels[apiType]+" "+firstNonNilStr(loaded["msg"], "未知错误"))
			loaded = map[string]any{"ok": false, "payload": map[string]any{"apiType": apiType, "items": []map[string]any{}, "batchItems": []map[string]any{}, "authorizedAccounts": map[string]any{}}}
		}
		loadedByAPI[apiType] = loaded
	}
	if len(loadErrors) > 0 && !asBool(loadedByAPI[1]["ok"], true) && !asBool(loadedByAPI[2]["ok"], true) {
		return map[string]any{"ok": false, "msg": strings.Join(loadErrors, "；")}
	}

	rowsByAPIUsername := map[string]map[string]any{}
	var payloadWarnings []string
	cxAuthorizedCount, cxItemCount, cxBatchCount := 0, 0, 0
	fromCache := false
	cacheAge := 0
	for _, apiType := range []int{1, 2} {
		payload, _ := loadedByAPI[apiType]["payload"].(map[string]any)
		if asBool(payload["fromCache"], false) {
			fromCache = true
		}
		cacheAge = max(cacheAge, qInt(payload["cacheAge"]))
		if wt := trimStr(payload["warning"]); wt != "" {
			payloadWarnings = append(payloadWarnings, apiLabels[apiType]+" "+wt)
		}
		cxAuthorizedCount += len(mapToAnyMapOfString(payload["authorizedAccounts"]))
		cxItemCount += len(rowsOf(payload, "items"))
		cxBatchCount += len(rowsOf(payload, "batchItems"))
		for _, section := range []string{"batchItems", "items"} {
			for _, row := range rowsOf(payload, section) {
				username := trimStr(row["username"])
				key := strconvItoa(apiType) + "|" + username
				if username == "" {
					continue
				}
				if _, ok := allowed[key]; !ok {
					continue
				}
				if _, exists := rowsByAPIUsername[key]; exists {
					continue
				}
				rowsByAPIUsername[key] = row
			}
		}
	}

	summary := map[string]int{"take": 0, "complete": 0, "success": 0, "fail": 0}
	byAPI := map[int]map[string]any{}
	for _, apiType := range []int{1, 2} {
		byAPI[apiType] = map[string]any{
			"api_type": apiType, "api_label": apiLabels[apiType], "task_count": len(allowedByAPI[apiType]),
			"allowed_count": len(allowedByAPI[apiType]), "active_count": 0, "matched_count": 0,
			"summary": map[string]int{"take": 0, "complete": 0, "success": 0, "fail": 0},
		}
	}
	var rows []map[string]any
	for key, m := range allowed {
		apiType := m.apiType
		row := rowsByAPIUsername[key]
		take := qInt(row["takeCount"])
		complete := qInt(row["completeCount"])
		success := qInt(row["checkSuccessCount"])
		fail := qInt(row["checkFailCount"])
		active := take+complete+success+fail > 0
		for _, pair := range []struct{ k string; v int }{{"take", take}, {"complete", complete}, {"success", success}, {"fail", fail}} {
			summary[pair.k] += pair.v
			byAPI[apiType]["summary"].(map[string]int)[pair.k] += pair.v
		}
		if _, ok := rowsByAPIUsername[key]; ok {
			byAPI[apiType]["matched_count"] = qInt(byAPI[apiType]["matched_count"]) + 1
		}
		if active {
			byAPI[apiType]["active_count"] = qInt(byAPI[apiType]["active_count"]) + 1
		}
		rows = append(rows, map[string]any{
			"username": m.username, "remark": firstNonNilStr(row["_remark"], m.remark), "api_type": apiType, "api_label": apiLabels[apiType],
			"take": take, "complete": complete, "success": success, "fail": fail,
			"completeRate": firstNonNilStr(row["completeRate"], ""), "successRate": firstNonNilStr(row["checkSuccessRate"], ""),
			"active": active,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if qInt(rows[i]["api_type"]) != qInt(rows[j]["api_type"]) {
			return qInt(rows[i]["api_type"]) < qInt(rows[j]["api_type"])
		}
		if qInt(rows[i]["success"]) != qInt(rows[j]["success"]) {
			return qInt(rows[i]["success"]) > qInt(rows[j]["success"])
		}
		return qInt(rows[i]["take"]) > qInt(rows[j]["take"])
	})
	activeCount := 0
	for _, r := range rows {
		if asBool(r["active"], false) {
			activeCount++
		}
	}
	matchedCount := len(rowsByAPIUsername)
	warningParts := append([]string{}, loadErrors...)
	warningParts = append(warningParts, payloadWarnings...)
	if len(warningParts) == 0 && len(allowed) > 0 && cxItemCount+cxBatchCount > 0 && matchedCount == 0 {
		warningParts = append(warningParts, "查询接口已更新，但当前并发控制绑定的任务账号没有匹配到对应数据，请检查控制链接是否对应新账号")
	}
	warning := strings.Join(warningParts, "；")
	return map[string]any{
		"ok": true,
		"data": map[string]any{
			"date": date, "accountId": trimStr(account["accountId"]), "remark": trimStr(account["remark"]),
			"task_count": len(allowed), "allowed_count": len(allowed), "active_count": activeCount,
			"matched_count": matchedCount, "missing_count": max(0, len(allowed)-activeCount),
			"summary": summary, "by_api": mapIntMapToSlice(byAPI), "rows": rows,
			"fromCache": fromCache, "cacheAge": cacheAge, "warning": warning, "force_refresh": force,
			"cx_authorized_count": cxAuthorizedCount, "cx_item_count": cxItemCount, "cx_batch_count": cxBatchCount,
			"generatedAt": tsDateTime(),
		},
	}
}

func mapToAnyMapOfString(v any) map[string]string {
	out := map[string]string{}
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]any:
		for k, vv := range m {
			out[k] = qStr(vv)
		}
	}
	return out
}

func mapIntMapToSlice(m map[int]map[string]any) []map[string]any {
	keys := []int{1, 2}
	var out []map[string]any
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func decodedExtParallelControlLiveStats(token string, reqCtx *httpRequestCtx) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return map[string]any{"ok": false, "msg": "缺少控制 token"}
	}
	account := decodedExtFindAccountByParallelControlToken(token)
	if account == nil {
		return map[string]any{"ok": false, "msg": "控制链接无效或已重置"}
	}
	accountID := trimStr(account["accountId"])
	accountRemark := trimStr(account["remark"])
	allowedUsernames := decodedExtParallelControlAllowedUsernames(account)
	now := timeNowUnix()
	onlineThreshold := int64(120)
	var deviceRows []map[string]any
	onlineCount := 0
	for _, device := range readDeviceControls() {
		if !decodedExtParallelControlMatchesAccount(device, accountID, allowedUsernames) {
			continue
		}
		lastSeen := parseTimeToUnix(trimStr(device["last_seen"]))
		online := lastSeen > 0 && now-lastSeen <= onlineThreshold
		if asBool(device["disabled"], false) {
			online = false
		}
		device["online"] = online
		remark := displayAccountRemark(qInt(device["api_type"]), trimStr(device["username"]), trimStr(device["group_id"]))
		if remark == "" {
			remark = displayDecodedExtensionAccountRemark(trimStr(device["username"]))
		}
		if remark != "" {
			device["account_remark"] = remark
		}
		deviceRows = append(deviceRows, device)
		if online {
			onlineCount++
		}
	}
	sort.Slice(deviceRows, func(i, j int) bool { return trimStr(deviceRows[i]["last_seen"]) > trimStr(deviceRows[j]["last_seen"]) })
	var filtered []map[string]any
	for _, d := range deviceRows {
		if asBool(d["disabled"], false) || asBool(d["online"], false) {
			filtered = append(filtered, d)
		}
	}
	deviceRows = filtered

	var logs []map[string]any
	for _, log := range readLogs(logFile, 200) {
		if !decodedExtParallelControlMatchesAccount(log, accountID, allowedUsernames) {
			continue
		}
		source := toLower(trimStr(log["source"]))
		status := toLower(trimStr(log["status"]))
		if inList(source, []string{"web_take", "api1_take", "ctrl_take", "decoded_extension_task", "decoded_extension_task_local", "decoded_extension_task_batch"}) {
			continue
		}
		if source == "decoded_extension_submit" && status == "client_task_failed" {
			continue
		}
		forwardURL := toLower(trimStr(log["forward_url"]))
		isSubmit := trimStr(log["submit_id"]) != "" || trimStr(log["upload_id"]) != "" ||
			strings.Contains(source, "submit") || strings.Contains(source, "forward") || strings.Contains(source, "relay") ||
			strings.Contains(status, "submit") || strings.Contains(status, "forward") || strings.Contains(forwardURL, "/submit") ||
			inList(source, []string{"decoded_extension_submit", "api1_submit", "web_submit", "ctrl_submit"})
		if !isSubmit {
			continue
		}
		remark := displayAccountRemark(qInt(log["api_type"]), trimStr(log["username"]), trimStr(log["group_id"]))
		if remark == "" {
			remark = displayDecodedExtensionAccountRemark(trimStr(log["username"]))
		}
		if cl := resolveClientLabel(log); cl != "" {
			log["client_label"] = cl
		}
		if remark != "" {
			log["account_remark"] = remark
		}
		if _, ok := log["response"]; ok {
			log["response"] = compactLogResponse(log["response"], 1200)
		}
		logs = append(logs, log)
		if len(logs) >= 18 {
			break
		}
	}
	return map[string]any{
		"ok": true,
		"data": map[string]any{
			"accountId": accountID, "remark": accountRemark,
			"device_count": len(deviceRows), "online_device_count": onlineCount,
			"devices": deviceRows, "logs": logs, "fromCache": false, "cacheAge": 0, "updated_at": tsDateTime(),
		},
	}
}

// decodedExtParallelControlSave
func decodedExtParallelControlSave(token string, parallelism, mixedTaskEnabled, api1Parallelism, api2Parallelism, mixedTaskWebEnabled, mixedTaskIOSEnabled, taskMode any) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return map[string]any{"ok": false, "msg": "缺少控制 token"}
	}
	decodedExtMu.Lock()
	defer decodedExtMu.Unlock()
	accounts := decodedExtReadAccounts()
	found := false
	for _, account := range accounts {
		if hashEquals(trimStr(account["parallel_control_token"]), token) {
			if parallelism != nil && trimStr(parallelism) != "" {
				account["task_parallelism"] = qInt(parallelism)
			}
			if mixedTaskEnabled != nil && trimStr(mixedTaskEnabled) != "" {
				account["mixed_task_enabled"] = asBool(mixedTaskEnabled, false)
				account["api1_fallback_api2"] = asBool(mixedTaskEnabled, false)
			}
			if api1Parallelism != nil && trimStr(api1Parallelism) != "" {
				account["api1_task_parallelism"] = qInt(api1Parallelism)
			}
			if api2Parallelism != nil && trimStr(api2Parallelism) != "" {
				account["api2_task_parallelism"] = qInt(api2Parallelism)
			}
			if mixedTaskWebEnabled != nil && trimStr(mixedTaskWebEnabled) != "" {
				account["mixed_task_web_enabled"] = asBool(mixedTaskWebEnabled, true)
			}
			if mixedTaskIOSEnabled != nil && trimStr(mixedTaskIOSEnabled) != "" {
				account["mixed_task_ios_enabled"] = asBool(mixedTaskIOSEnabled, false)
			}
			if taskMode != nil && trimStr(taskMode) != "" {
				account["task_mode"] = decodedExtNormalizeTaskMode(taskMode)
			}
			found = true
		}
	}
	if !found {
		return map[string]any{"ok": false, "msg": "控制链接无效或已重置"}
	}
	saved := decodedExtSaveAccounts(accounts)
	var publicData map[string]any
	for _, a := range saved {
		if hashEquals(trimStr(a["parallel_control_token"]), token) {
			publicData = decodedExtParallelControlPublicData(a)
			break
		}
	}
	return map[string]any{"ok": true, "msg": "设置已保存", "data": publicData}
}
