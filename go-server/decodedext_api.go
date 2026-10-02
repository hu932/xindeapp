package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func decodedExtAPIPath(r *http.Request) string {
	q := r.URL.Query()
	for _, v := range []string{q.Get("ajie_api"), q.Get("api")} {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "login":
			return "/api/login"
		case "take", "task":
			return "/api/task"
		case "submit":
			return "/api/submit"
		case "heartbeat", "heart_beat":
			return "/api/heartbeat"
		}
	}
	path := r.URL.Path
	switch path {
	case "/api/login", "/decrypt_proxy.php/api/login", "/g.php/api/login":
		return "/api/login"
	case "/api/task", "/decrypt_proxy.php/api/task", "/g.php/api/task":
		return "/api/task"
	case "/api/submit", "/decrypt_proxy.php/api/submit", "/g.php/api/submit":
		return "/api/submit"
	case "/api/heartbeat", "/decrypt_proxy.php/api/heartbeat", "/g.php/api/heartbeat":
		return "/api/heartbeat"
	}
	return ""
}

func decodedExtCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

func decodedExtJSON(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, v)
}

func decodedExtReadJSONBody(r *http.Request) map[string]any {
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 256<<20))
	var m map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &m) == nil {
		return m
	}
	return map[string]any{}
}

// handleDecodedExtensionAPI 处理插件 /api/* 路由；非该路由返回 false
func handleDecodedExtensionAPI(w http.ResponseWriter, r *http.Request) bool {
	path := decodedExtAPIPath(r)
	if path == "" {
		return false
	}
	decodedExtCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return true
	}
	if r.Method != http.MethodPost {
		decodedExtJSON(w, 405, map[string]any{"success": false, "error": "method not allowed"})
		return true
	}
	reqCtx := newHTTPRequestCtx(r)
	input := decodedExtReadJSONBody(r)
	ajieRequest := decodedExtIsAjieRequest(input, nil)
	bearer := bearerToken(r)
	pluginUsername := decodedExtInputAccountID(input)
	pluginToken := trimStr(firstNonNil(input["token"], input["auth_token"], input["plugin_token"]))
	if pluginToken == "" {
		pluginToken = bearer
	}
	if pluginUsername == "" && pluginToken != "" {
		pluginUsername = resolveAPIUsernameFromToken(pluginToken)
	}
	pluginAccount := decodedExtFindAccount(pluginUsername, pluginToken)
	if pluginAccount == nil && pluginToken != "" {
		pluginAccount = decodedExtFindAccountByToken(pluginToken)
		if pluginAccount != nil {
			pluginUsername = trimStr(pluginAccount["accountId"])
		}
	}
	if pluginAccount == nil && pluginUsername != "" {
		pluginAccount = decodedExtFindAccountByID(pluginUsername)
		if pluginAccount != nil {
			pluginUsername = trimStr(pluginAccount["accountId"])
			if pluginToken == "" {
				pluginToken = trimStr(pluginAccount["token"])
			}
		}
	}
	if pluginAccount == nil {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "account not allowed"})
		return true
	}
	deviceContext := decodedExtDeviceContext(input, pluginUsername)
	deviceCheck := checkDeviceAllowed(deviceContext, reqCtx)
	if !asBool(deviceCheck["allowed"], true) {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": firstNonNilStr(deviceCheck["msg"], "device disabled")})
		return true
	}
	runtimeAccount := decodedExtRuntimeAccount(pluginAccount, deviceContext, "")
	taskAccountPool := decodedExtEnabledTaskAccounts(pluginAccount)
	if ajieRequest {
		taskAccountPool = decodedExtEnabledTaskAccountsByType(pluginAccount, 1)
	}

	switch path {
	case "/api/login":
		decodedExtAPI1Login(w, r, reqCtx, input, pluginAccount, pluginUsername, pluginToken, ajieRequest, taskAccountPool)
	case "/api/task":
		decodedExtAPITask(w, r, reqCtx, input, pluginAccount, pluginUsername, ajieRequest, deviceContext, runtimeAccount, taskAccountPool)
	case "/api/submit":
		decodedExtAPISubmit(w, r, reqCtx, input, pluginAccount, pluginUsername, ajieRequest, deviceContext, runtimeAccount, taskAccountPool)
	case "/api/heartbeat":
		ver := trimStr(firstNonNil(input["extension_version"], input["plugin_version"], input["version"]))
		code := qInt(firstNonNil(input["extension_version_code"], input["plugin_version_code"], input["version_code"], 0))
		decodedExtJSON(w, 200, map[string]any{
			"success": true, "msg": "ok",
			"data": map[string]any{
				"allowed": true, "device_key": deviceCheck["device_key"],
				"last_seen": tsDateTime(), "plugin_update": buildPluginUpdateResponse(ver, code),
			},
		})
	}
	return true
}

func decodedExtAPI1Login(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input, pluginAccount map[string]any, pluginUsername, pluginToken string, ajieRequest bool, taskAccountPool []map[string]any) {
	if pluginUsername == "" {
		decodedExtJSON(w, 400, map[string]any{"success": false, "error": "accountId is empty"})
		return
	}
	if pluginAccount == nil {
		appendLog(logFile, map[string]any{"api_type": 1, "ip": reqCtx.ip, "username": pluginUsername, "status": "login_rejected", "source": "decoded_extension_login", "error": "decoded extension account not allowed"}, maxLogs)
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "account not allowed"})
		return
	}
	if len(taskAccountPool) == 0 {
		decodedExtJSON(w, 500, map[string]any{"success": false, "error": "task account pool is empty"})
		return
	}
	loginTaskAccountPool := decodedExtEnabledTaskAccountsByType(pluginAccount, 1)
	if len(loginTaskAccountPool) == 0 {
		decodedExtJSON(w, 500, map[string]any{"success": false, "error": "api1 task account pool is empty"})
		return
	}
	var loginAttempts []map[string]any
	var loginOK tokenInfo
	var api1Auth upstreamAuth
	firstOK := false
	for _, loginTaskAccount := range loginTaskAccountPool {
		tryAuth := decodedExtAPI1AuthForTaskAccount(pluginAccount, loginTaskAccount)
		tf := decodedExtAPI1TokenFileForAccount(trimStr(pluginAccount["accountId"]), tryAuth)
		tryTokenInfo := getAPIToken(tryAuth, tf, false)
		loginAttempts = append(loginAttempts, map[string]any{
			"api1_username": tryAuth.Username, "ok": tryTokenInfo.OK, "cached": tryTokenInfo.Cached, "msg": tryTokenInfo.Msg,
		})
		if tryTokenInfo.OK && !firstOK {
			loginOK = tryTokenInfo
			api1Auth = tryAuth
			firstOK = true
		}
	}
	if !firstOK {
		decodedExtJSON(w, 502, map[string]any{"success": false, "error": "all task accounts login failed"})
		return
	}
	if pluginToken == "" {
		pluginToken = trimStr(pluginAccount["token"])
	}
	cacheAPITokenUsername(pluginToken, pluginUsername, timeNowUnix()+86400*30)
	configuredTaskMode := "poll"
	if !ajieRequest {
		configuredTaskMode = decodedExtNormalizeTaskMode(pluginAccount["task_mode"])
	}
	mixedTaskEnabled := !ajieRequest && decodedExtMixedTaskEnabled(pluginAccount)
	mixedTaskClientType := decodedExtRequestClientType(input, deviceContextOf(pluginUsername))
	mixedTaskWebEnabled := !ajieRequest && decodedExtMixedTaskWebEnabled(pluginAccount)
	mixedTaskIOSEnabled := !ajieRequest && decodedExtMixedTaskIosEnabled(pluginAccount)
	parallelControlScope := !ajieRequest && decodedExtParallelControlAppliesToRequest(pluginAccount, input, nil)
	mixedTaskScope := !ajieRequest && decodedExtMixedTaskAppliesToRequest(pluginAccount, input, nil)
	taskMode := "poll"
	if !ajieRequest && parallelControlScope {
		taskMode = configuredTaskMode
	}
	api1PoolCount := len(decodedExtEnabledTaskAccountsByType(pluginAccount, 1))
	api2PoolCount := 0
	if !ajieRequest {
		api2PoolCount = len(decodedExtEnabledTaskAccountsByType(pluginAccount, 2))
	}
	api1Parallelism := 0
	if api1PoolCount > 0 {
		if ajieRequest {
			api1Parallelism = 1
		} else {
			api1Parallelism = decodedExtAPITaskParallelism(pluginAccount, 1, api1PoolCount)
		}
	}
	api2Parallelism := 0
	if api2PoolCount > 0 {
		api2Parallelism = decodedExtAPITaskParallelism(pluginAccount, 2, api2PoolCount)
	}
	taskParallelism := 1
	if !ajieRequest {
		if taskMode == "concurrent" {
			if mixedTaskScope {
				taskParallelism = max(1, api1Parallelism+api2Parallelism)
			} else {
				taskParallelism = decodedExtNormalizeTaskParallelism(pluginAccount["task_parallelism"], len(taskAccountPool))
			}
		}
	}
	appendLog(logFile, map[string]any{
		"api_type": api1Auth.APIType, "ip": reqCtx.ip, "username": pluginUsername,
		"api1_username": condStr(api1Auth.APIType == 1, api1Auth.Username), "api2_username": condStr(api1Auth.APIType == 2, api1Auth.Username),
		"task_account_username": api1Auth.Username, "task_api_type": api1Auth.APIType,
		"api1_pool_count": len(taskAccountPool), "task_mode": taskMode, "task_mode_configured": configuredTaskMode,
		"parallel_control_effective_enabled": parallelControlScope, "task_parallelism": taskParallelism,
		"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
		"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
		"api1_task_parallelism": api1Parallelism, "api2_task_parallelism": api2Parallelism,
		"task_parallelism_configured": decodedExtNormalizeTaskParallelism(pluginAccount["task_parallelism"], len(taskAccountPool)),
		"api1_login_attempts": loginAttempts, "api1_token_cached": loginOK.Cached, "ajie_request": ajieRequest,
		"status": "success", "source": "decoded_extension_login",
	}, maxLogs)
	decodedExtJSON(w, 200, map[string]any{
		"success": true, "msg": "ok",
		"data": map[string]any{
			"accountId": pluginUsername, "token": pluginToken, "task_mode": taskMode, "task_mode_configured": configuredTaskMode,
			"parallel_control_effective_enabled": parallelControlScope, "task_parallelism": taskParallelism,
			"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
			"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
			"api1_task_parallelism": api1Parallelism, "api2_task_parallelism": api2Parallelism,
			"task_account_count": len(taskAccountPool), "api1_task_account_count": api1PoolCount, "api2_task_account_count": api2PoolCount,
			"api_type": 1, "api1_only": ajieRequest, "disable_api2": ajieRequest, "disable_api1_fallback_api2": ajieRequest,
		},
	})
}

// condStr 三元字符串：condStr(cond, a) = cond?a:""; condStr(cond, a, b) = cond?a:b
func condStr(cond bool, args ...string) string {
	if len(args) == 0 {
		return ""
	}
	if cond {
		return args[0]
	}
	if len(args) >= 2 {
		return args[1]
	}
	return ""
}

func deviceContextOf(username string) map[string]any {
	return map[string]any{"username": username}
}

func decodedExtAPITask(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input, pluginAccount map[string]any, pluginUsername string, ajieRequest bool, deviceContext, runtimeAccount map[string]any, taskAccountPool []map[string]any) {
	if pluginAccount == nil {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "account not allowed"})
		return
	}
	if len(taskAccountPool) == 0 {
		decodedExtJSON(w, 500, map[string]any{"success": false, "error": "task account pool is empty"})
		return
	}
	configuredTaskMode := "poll"
	if !ajieRequest {
		configuredTaskMode = decodedExtNormalizeTaskMode(pluginAccount["task_mode"])
	}
	mixedTaskEnabled := !ajieRequest && decodedExtMixedTaskEnabled(pluginAccount)
	mixedTaskClientType := decodedExtRequestClientType(input, deviceContext)
	mixedTaskWebEnabled := !ajieRequest && decodedExtMixedTaskWebEnabled(pluginAccount)
	mixedTaskIOSEnabled := !ajieRequest && decodedExtMixedTaskIosEnabled(pluginAccount)
	parallelControlScope := !ajieRequest && decodedExtParallelControlAppliesToRequest(pluginAccount, input, deviceContext)
	mixedTaskScope := !ajieRequest && decodedExtMixedTaskAppliesToRequest(pluginAccount, input, deviceContext)
	taskMode := "poll"
	if !ajieRequest && parallelControlScope {
		taskMode = configuredTaskMode
	}
	api1TaskPool := decodedExtEnabledTaskAccountsByType(pluginAccount, 1)
	api2TaskPool := []map[string]any{}
	if !ajieRequest {
		api2TaskPool = decodedExtEnabledTaskAccountsByType(pluginAccount, 2)
	}
	api1Configured := 0
	if len(api1TaskPool) > 0 {
		if ajieRequest {
			api1Configured = 1
		} else {
			api1Configured = decodedExtAPITaskParallelism(pluginAccount, 1, len(api1TaskPool))
		}
	}
	api2Configured := 0
	if len(api2TaskPool) > 0 {
		api2Configured = decodedExtAPITaskParallelism(pluginAccount, 2, len(api2TaskPool))
	}
	configuredParallelism := 1
	if !ajieRequest {
		if taskMode == "concurrent" {
			if mixedTaskScope {
				configuredParallelism = max(1, api1Configured+api2Configured)
			} else {
				configuredParallelism = decodedExtNormalizeTaskParallelism(pluginAccount["task_parallelism"], len(taskAccountPool))
			}
		}
	}
	takeType := strings.ToLower(strings.TrimSpace(firstNonNilStr(input["type"], input["task_type"])))
	maxRequestCount := 1
	if !ajieRequest && taskMode == "concurrent" {
		maxRequestCount = min(len(taskAccountPool), configuredParallelism)
	}
	requestedCount := 1
	if ajieRequest {
		requestedCount = 1
	} else if taskMode == "concurrent" {
		requestedCount = max(1, qInt(firstNonNil(input["batch_size"], input["request_count"], maxRequestCount)))
	}
	requestedCount = min(max(1, requestedCount), maxRequestCount)

	runtimeActiveLeaseCount := decodedExtActiveTaskLeaseCount(runtimeAccount)
	activeLeaseCountsByAPI := map[int]int{0: 0, 1: 0, 2: 0}
	if mixedTaskScope {
		activeLeaseCountsByAPI = decodedExtActiveTaskLeaseCountsByType(runtimeAccount)
	}
	remainingByAPI := map[int]int{1: 1, 2: 0}
	if taskMode == "concurrent" && mixedTaskScope {
		remainingByAPI[1] = max(0, api1Configured-activeLeaseCountsByAPI[1])
		remainingByAPI[2] = max(0, api2Configured-activeLeaseCountsByAPI[2])
	}
	remainingRuntimeSlots := 1
	if taskMode == "concurrent" {
		if mixedTaskScope {
			remainingRuntimeSlots = remainingByAPI[1] + remainingByAPI[2]
		} else {
			remainingRuntimeSlots = max(0, maxRequestCount-runtimeActiveLeaseCount)
		}
	}
	blockUpstreamTake := taskMode == "concurrent" && remainingRuntimeSlots <= 0
	if taskMode == "concurrent" {
		requestedCount = min(requestedCount, max(0, remainingRuntimeSlots))
	}

	var packets []map[string]any
	var attempts []map[string]any
	lastMsg := "no task"
	seenPacketKeys := map[string]bool{}
	acceptedByAPI := map[int]int{1: 0, 2: 0}

	appendPacket := func(task, taskAccount map[string]any, api1Token string) bool {
		if api1Token == "" {
			api1Token = trimStr(firstNonNil(task["api1_take_token"], task["api1TakeToken"]))
		}
		taskID := decodedExtTaskDealID(task)
		taskURL := decodedExtTaskURL(task)
		dedupeKey := taskID + "|" + taskURL
		if dedupeKey == "|" {
			dedupeKey = md5hex(taskURL)
		}
		if seenPacketKeys[dedupeKey] {
			return false
		}
		seenPacketKeys[dedupeKey] = true
		resolvedTaskAccount := taskAccount
		if resolvedTaskAccount == nil {
			resolvedTaskAccount = decodedExtTaskAccountByKey(pluginAccount, trimStr(firstNonNil(task["task_account_key"], task["taskAccountKey"])), trimStr(firstNonNil(task["task_account_username"], task["taskAccountUsername"])))
			if resolvedTaskAccount == nil {
				resolvedTaskAccount = decodedExtTaskAccountForSubmit(runtimeAccount, taskID, taskURL, 0)
			}
		}
		if ajieRequest && resolvedTaskAccount != nil && decodedExtAPITypeForTaskAccount(resolvedTaskAccount) != 1 {
			return false
		}
		if resolvedTaskAccount != nil {
			decodedExtRecordTaskLease(runtimeAccount, task, resolvedTaskAccount, api1Token)
			task = decodedExtTaskWithAccountMeta(task, resolvedTaskAccount)
			task = decodedExtTaskWithTakeTokenMeta(task, api1Token)
			task["task_runtime_scope"] = trimStr(runtimeAccount["__storage_scope"])
			task["taskRuntimeScope"] = trimStr(runtimeAccount["__storage_scope"])
		}
		packetTaskAccount := resolvedTaskAccount
		if packetTaskAccount == nil && len(taskAccountPool) > 0 {
			packetTaskAccount = taskAccountPool[0]
		}
		packets = append(packets, map[string]any{"task": task, "task_account": packetTaskAccount})
		packetAPIType := 0
		if resolvedTaskAccount != nil {
			packetAPIType = decodedExtAPITypeForTaskAccount(resolvedTaskAccount)
		} else {
			packetAPIType = decodedExtTaskAPITypeFromMeta(pluginAccount, task)
		}
		if packetAPIType == 1 || packetAPIType == 2 {
			acceptedByAPI[packetAPIType]++
		}
		return true
	}

	var deferredAPI2LocalTasks []map[string]any
	if requestedCount > 0 {
		var deferredLocal []map[string]any
		localPopCount := requestedCount
		if taskMode == "concurrent" && mixedTaskScope {
			localPopCount = max(requestedCount, api1Configured+api2Configured)
		}
		for _, task := range decodedExtPopLocalTasks(localPopCount, runtimeAccount) {
			queuedAPIType := decodedExtTaskAPITypeFromMeta(pluginAccount, task)
			if !mixedTaskScope && queuedAPIType == 2 {
				deferredLocal = append(deferredLocal, task)
				continue
			}
			if taskMode == "concurrent" && mixedTaskScope {
				if queuedAPIType == 2 {
					deferredAPI2LocalTasks = append(deferredAPI2LocalTasks, task)
					continue
				}
				if (queuedAPIType == 1 || queuedAPIType == 2) && acceptedByAPI[queuedAPIType] >= remainingByAPI[queuedAPIType] {
					deferredLocal = append(deferredLocal, task)
					continue
				}
			}
			if len(packets) >= requestedCount {
				deferredLocal = append(deferredLocal, task)
				continue
			}
			appendPacket(task, nil, "")
		}
		if len(deferredLocal) > 0 {
			decodedExtAppendLocalTasks(deferredLocal, runtimeAccount)
		}
	}

	runTakePlan := func(plannedAPIType, takeLimit int) int {
		before := len(packets)
		takeLimit = max(1, takeLimit)
		if ajieRequest {
			plannedAPIType = 1
		}
		takeResult := decodedExtTakeTaskFromPluginPool(pluginAccount, takeLimit, takeType, plannedAPIType, false, ajieRequest)
		if arr, ok := takeResult["attempts"].([]map[string]any); ok {
			attempts = append(attempts, arr...)
		}
		lastMsg = trimStr(firstNonNil(takeResult["last_msg"], "no task"))
		var fetched []map[string]any
		if task, ok := takeResult["task"].(map[string]any); ok {
			fetched = append(fetched, map[string]any{"task": task, "task_account": takeResult["task_account"], "api1_token": takeResult["api1_token"]})
		}
		if extras, ok := takeResult["extra_tasks"].([]map[string]any); ok {
			for _, extra := range extras {
				if extra != nil && extra["task"] != nil {
					fetched = append(fetched, extra)
				}
			}
		}
		missing := max(0, requestedCount-len(packets))
		for idx, packet := range fetched {
			task, _ := packet["task"].(map[string]any)
			if task == nil {
				continue
			}
			packetAPIType := 0
			if ta, ok := packet["task_account"].(map[string]any); ok {
				packetAPIType = decodedExtAPITypeForTaskAccount(ta)
			} else {
				packetAPIType = decodedExtTaskAPITypeFromMeta(pluginAccount, task)
			}
			apiLimitReached := taskMode == "concurrent" && mixedTaskScope && (packetAPIType == 1 || packetAPIType == 2) && acceptedByAPI[packetAPIType] >= remainingByAPI[packetAPIType]
			if idx < missing && !apiLimitReached {
				ta, _ := packet["task_account"].(map[string]any)
				appendPacket(task, ta, qStr(packet["api1_token"]))
				continue
			}
			queuedTask := task
			if ta, ok := packet["task_account"].(map[string]any); ok {
				decodedExtRecordTaskLease(runtimeAccount, task, ta, qStr(packet["api1_token"]))
				queuedTask = decodedExtTaskWithAccountMeta(task, ta)
				queuedTask = decodedExtTaskWithTakeTokenMeta(queuedTask, qStr(packet["api1_token"]))
			}
			decodedExtAppendLocalTasks([]map[string]any{queuedTask}, runtimeAccount)
		}
		return max(0, len(packets)-before)
	}

	fetchRounds := 0
	for !blockUpstreamTake && len(packets) < requestedCount && fetchRounds < max(1, requestedCount) {
		fetchRounds++
		beforePacketCount := len(packets)
		missingForRound := max(1, requestedCount-len(packets))

		if taskMode == "concurrent" && mixedTaskScope {
			api1Remaining := max(0, remainingByAPI[1]-acceptedByAPI[1])
			api1Limit := min(api1Remaining, missingForRound)
			api1Accepted := 0
			if api1Limit > 0 {
				api1Accepted = runTakePlan(1, api1Limit)
			}
			loginHadNoTask := api1Configured <= 0 || (api1Limit > 0 && api1Accepted <= 0 && acceptedByAPI[1] <= 0)
			if loginHadNoTask && len(packets) < requestedCount {
				for len(deferredAPI2LocalTasks) > 0 && len(packets) < requestedCount {
					api2LocalRemaining := max(0, remainingByAPI[2]-acceptedByAPI[2])
					if api2LocalRemaining <= 0 {
						break
					}
					task := deferredAPI2LocalTasks[0]
					deferredAPI2LocalTasks = deferredAPI2LocalTasks[1:]
					appendPacket(task, nil, "")
				}
				api2Remaining := max(0, remainingByAPI[2]-acceptedByAPI[2])
				api2Limit := min(api2Remaining, max(1, requestedCount-len(packets)))
				if api2Limit > 0 {
					runTakePlan(2, api2Limit)
				}
			}
			break
		}
		takeLimit := 1
		if taskMode == "poll" {
			takeLimit = 1
		} else {
			takeLimit = min(configuredParallelism, missingForRound)
		}
		plannedType := 0
		if ajieRequest {
			plannedType = 1
		} else if mixedTaskEnabled && !mixedTaskScope {
			plannedType = 1
		}
		runTakePlan(plannedType, takeLimit)
		if len(packets) <= beforePacketCount {
			break
		}
	}
	if len(deferredAPI2LocalTasks) > 0 {
		decodedExtAppendLocalTasks(deferredAPI2LocalTasks, runtimeAccount)
	}

	if len(packets) == 0 {
		appendLog(logFile, map[string]any{
			"api_type": 0, "ip": reqCtx.ip, "username": pluginUsername, "device_id": deviceContext["device_id"],
			"runtime_scope": trimStr(runtimeAccount["__storage_scope"]),
			"status": condStr(blockUpstreamTake, "active_tasks_pending", "no_task"), "source": "decoded_extension_task",
			"task_mode": taskMode, "task_mode_configured": configuredTaskMode,
			"parallel_control_effective_enabled": parallelControlScope, "task_parallelism": configuredParallelism,
			"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
			"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
			"api1_task_parallelism": api1Configured, "api2_task_parallelism": api2Configured,
			"active_lease_counts_by_api": activeLeaseCountsByAPI, "remaining_by_api": remainingByAPI,
			"runtime_active_lease_count": runtimeActiveLeaseCount, "remaining_runtime_slots": remainingRuntimeSlots,
			"upstream_take_paused": blockUpstreamTake, "forward_url": "", "forward_attempts": attempts,
		}, maxLogs)
		decodedExtJSON(w, 200, map[string]any{
			"success": false, "msg": lastMsg,
			"data": map[string]any{
				"msg": lastMsg, "runtime_active_lease_count": runtimeActiveLeaseCount, "remaining_runtime_slots": remainingRuntimeSlots,
				"remaining_by_api": remainingByAPI, "upstream_take_paused": blockUpstreamTake,
				"pool": map[string]any{
					"accounts": len(taskAccountPool), "task_parallelism": configuredParallelism, "task_mode": taskMode,
					"task_mode_configured": configuredTaskMode, "parallel_control_effective_enabled": parallelControlScope,
					"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
					"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
					"api1_task_parallelism": api1Configured, "api2_task_parallelism": api2Configured,
				},
			},
		})
		return
	}

	firstPacket := packets[0]
	task, _ := firstPacket["task"].(map[string]any)
	taskAccount, _ := firstPacket["task_account"].(map[string]any)
	var responseTasks []map[string]any
	for _, p := range packets {
		responseTasks = append(responseTasks, p["task"].(map[string]any))
	}
	taskCount := len(responseTasks)
	taskAPIType := 1
	if !ajieRequest && taskAccount != nil {
		taskAPIType = decodedExtAPITypeForTaskAccount(taskAccount)
	}
	taskAccountUsername := ""
	if taskAccount != nil {
		taskAccountUsername = trimStr(taskAccount["username"])
	}
	appendLog(logFile, map[string]any{
		"api_type": taskAPIType, "ip": reqCtx.ip, "username": pluginUsername,
		"api1_username": condStr(taskAPIType == 1, taskAccountUsername), "api2_username": condStr(taskAPIType == 2, taskAccountUsername),
		"task_account_username": taskAccountUsername, "task_api_type": taskAPIType,
		"device_id": deviceContext["device_id"], "runtime_scope": trimStr(runtimeAccount["__storage_scope"]),
		"status": "success", "source": condStr(taskCount > 1, "decoded_extension_task_batch", "decoded_extension_task_local"),
		"task_mode": taskMode, "task_mode_configured": configuredTaskMode,
		"parallel_control_effective_enabled": parallelControlScope, "task_parallelism": configuredParallelism,
		"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
		"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
		"api1_task_parallelism": api1Configured, "api2_task_parallelism": api2Configured,
		"requested_count": requestedCount, "fetch_rounds": fetchRounds, "batch_size": taskCount,
		"runtime_active_lease_count_before_take": runtimeActiveLeaseCount, "remaining_runtime_slots_before_take": remainingRuntimeSlots,
		"active_lease_counts_by_api_before_take": activeLeaseCountsByAPI, "remaining_by_api_before_take": remainingByAPI,
		"upstream_take_paused": blockUpstreamTake, "forward_url": apiTakeURLForType(taskAPIType, takeType), "forward_attempts": attempts,
		"task_id": decodedExtTaskDealID(task), "task_url": decodedExtTaskURL(task), "ajie_request": ajieRequest,
	}, maxLogs)

	poolData := map[string]any{
		"accounts": len(taskAccountPool), "task_parallelism": configuredParallelism, "task_mode": taskMode,
		"task_mode_configured": configuredTaskMode, "parallel_control_effective_enabled": parallelControlScope,
		"mixed_task_enabled": mixedTaskEnabled, "mixed_task_effective_enabled": mixedTaskScope, "mixed_task_client_type": mixedTaskClientType,
		"mixed_task_web_enabled": mixedTaskWebEnabled, "mixed_task_ios_enabled": mixedTaskIOSEnabled,
		"api1_task_parallelism": api1Configured, "api2_task_parallelism": api2Configured,
	}
	if requestedCount > 1 || taskCount > 1 {
		decodedExtJSON(w, 200, map[string]any{
			"success": true, "tasks": responseTasks, "task": task, "msg": "ok",
			"data": map[string]any{
				"tasks": responseTasks, "task": task, "requested_count": requestedCount, "batch_size": taskCount,
				"runtime_active_lease_count": runtimeActiveLeaseCount, "remaining_runtime_slots": remainingRuntimeSlots,
				"upstream_take_paused": blockUpstreamTake, "pool": poolData, "api_type": 1, "api1_only": ajieRequest,
			},
		})
		return
	}
	decodedExtJSON(w, 200, map[string]any{
		"success": true, "task": task, "api_type": 1, "api1_only": ajieRequest, "task_mode": "poll", "task_parallelism": 1,
	})
}

func decodedExtAPISubmit(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input, pluginAccount map[string]any, pluginUsername string, ajieRequest bool, deviceContext, runtimeAccount map[string]any, taskAccountPool []map[string]any) {
	if pluginAccount == nil {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "account not allowed"})
		return
	}
	if len(taskAccountPool) == 0 {
		decodedExtJSON(w, 500, map[string]any{"success": false, "error": "task account pool is empty"})
		return
	}
	submitStart := timeNow()
	taskID := trimStr(firstNonNil(input["deal_id"], input["dealId"], input["taskId"], input["task_id"], input["tid"], input["trace_id"]))
	submitOK := true
	if hasKey(input, "success") {
		submitOK = asBool(input["success"], true)
	}
	taskAccountKey := trimStr(firstNonNil(input["taskAccountKey"], input["task_account_key"], input["__task_account_key"]))
	taskAccountUsername := trimStr(firstNonNil(input["taskAccountUsername"], input["task_account_username"], input["api1_username"]))
	inputTaskAPIType := qInt(firstNonNil(input["taskApiType"], input["task_api_type"], input["apiType"], input["api_type"], 0))
	if inputTaskAPIType != 1 && inputTaskAPIType != 2 {
		inputTaskAPIType = 0
	}
	if ajieRequest && inputTaskAPIType == 2 {
		decodedExtJSON(w, 400, map[string]any{"success": false, "error": "Ajie accepts API1 tasks only", "_proxy": map[string]any{"submit_accepted": false, "status": "api2_rejected_for_ajie"}})
		return
	}
	leaseURL := trimStr(firstNonNil(input["productUrl"], input["pageUrl"], input["taskUrl"], input["task_url"], input["url"], input["rawResponseUrl"], input["apiUrl"]))
	taskRuntimeScope := trimStr(firstNonNil(input["taskRuntimeScope"], input["task_runtime_scope"]))
	if taskRuntimeScope != "" {
		runtimeAccount = decodedExtRuntimeAccount(pluginAccount, deviceContext, taskRuntimeScope)
	}
	submitLease := decodedExtFindTaskLease(runtimeAccount, taskID, leaseURL)
	var leaseTaskAccount map[string]any
	if submitLease != nil {
		leaseTaskAccount = decodedExtTaskAccountByKey(pluginAccount, trimStr(submitLease["task_account_key"]), trimStr(submitLease["task_account_username"]))
	}
	inputTaskAccount := decodedExtTaskAccountByKey(pluginAccount, taskAccountKey, taskAccountUsername)
	submitTaskAccount := leaseTaskAccount
	if submitTaskAccount == nil {
		submitTaskAccount = inputTaskAccount
	}
	if submitTaskAccount == nil {
		submitTaskAccount = decodedExtTaskAccountForSubmit(runtimeAccount, taskID, leaseURL, inputTaskAPIType)
	}
	if submitTaskAccount == nil && len(taskAccountPool) > 0 {
		submitTaskAccount = taskAccountPool[0]
	}
	api1Auth := decodedExtAPI1AuthForTaskAccount(pluginAccount, submitTaskAccount)
	submitAPIType := decodedExtAPITypeForTaskAccount(submitTaskAccount)
	if ajieRequest && submitAPIType != 1 {
		decodedExtJSON(w, 400, map[string]any{"success": false, "error": "Ajie accepts API1 tasks only", "_proxy": map[string]any{"submit_accepted": false, "status": "api2_rejected_for_ajie"}})
		return
	}
	submitTargetURL := apiSubmitURLForType(submitAPIType)

	if !submitOK {
		purged := map[string]any{"queue_removed": 0, "lease_removed": 0, "skipped": "lease_missing"}
		if submitLease != nil {
			purged = decodedExtClearTaskRuntimeState(runtimeAccount)
		}
		entry := map[string]any{
			"api_type": submitAPIType, "ip": reqCtx.ip, "username": pluginUsername, "api1_username": api1Auth.Username,
			"task_api_type": submitAPIType, "device_id": deviceContext["device_id"], "runtime_scope": trimStr(runtimeAccount["__storage_scope"]),
			"task_runtime_scope": taskRuntimeScope, "status": "client_task_failed", "source": "decoded_extension_submit",
			"task_id": taskID, "task_url": firstNonNilStr(input["productUrl"], input["pageUrl"]), "error": qStr(input["error"]), "purged_runtime": purged,
		}
		appendLog(logFile, entry, maxLogs)
		appendLog(decodedExtSubmitLogF, entry, 1000)
		decodedExtJSON(w, 200, map[string]any{"success": true, "msg": "failure recorded", "data": map[string]any{"raw": decodedExtRawSuccess()}})
		return
	}

	submitURL := trimStr(firstNonNil(input["url"], input["rawResponseUrl"], input["apiUrl"], input["productUrl"], input["pageUrl"]))
	rawResponseValue := input["rawResponseText"]
	hasRawResponse := false
	if v, ok := input["rawResponseText"]; ok && v != nil {
		if s, isStr := v.(string); !isStr || strings.TrimSpace(s) != "" {
			hasRawResponse = true
		}
	}
	submitResultAny := rawResponseValue
	if hasRawResponse {
		submitResultAny = rawResponseValue
	} else {
		submitResultAny = input["result"]
	}
	submitResult := anyToStr(submitResultAny)
	if strings.TrimSpace(submitResult) == "" {
		if data, ok := input["data"].(string); ok && strings.TrimSpace(data) != "" {
			submitResult = data
		}
	}
	// 客户端提交的 result / rawResponseText / data 可能是 AES 加密的，先解密再转发
	if strings.TrimSpace(submitResult) != "" {
		decrypted, okDec := decryptMarsSubmitData(submitResult)
		if !okDec {
			decodedExtJSON(w, 400, map[string]any{"success": false, "error": "submit data decrypt failed"})
			return
		}
		submitResult = decrypted
	}
	submitResult = anyToStr(submitResult)

	if submitURL == "" || strings.TrimSpace(submitResult) == "" {
		decodedExtJSON(w, 400, map[string]any{"success": false, "error": "submit url or result is empty"})
		return
	}
	api2NullFields := 0
	if submitAPIType == 2 {
		submitResult, api2NullFields = api2SanitizeSubmitResult(submitResult)
	}
	submitResultDiag := decodedExtSubmitResultDiagnostics(submitResult)

	// 提交前校验：url 的 item_id 必须等于 result 的 data.item.item_id（与上游一致，避免转发后被上游拒）
	urlItemID := submitItemIDFromURL(submitURL)
	resultItemID := submitItemIDFromResult(submitResult)
	if urlItemID != "" && resultItemID != "" && urlItemID != resultItemID {
		appendLog(logFile, map[string]any{
			"api_type": submitAPIType, "ip": reqCtx.ip, "username": pluginUsername,
			"status": "item_id_mismatch", "source": "decoded_extension_submit",
			"task_id": taskID, "task_url": submitURL, "url_item_id": urlItemID, "result_item_id": resultItemID,
		}, maxLogs)
		decodedExtJSON(w, 400, map[string]any{
			"success": false, "error": "item_id 不匹配：url=" + urlItemID + " result=" + resultItemID,
			"upstreamSubmitFailed": true, "retryable": false,
			"_proxy": map[string]any{"submit_accepted": false, "status": "item_id_mismatch"},
		})
		return
	}

	inputTakeToken := trimStr(firstNonNil(input["api1_take_token"], input["api1TakeToken"]))
	leaseToken := ""
	if submitLease != nil {
		leaseToken = trimStr(submitLease["api1_token"])
	}
	leaseTokenSource := ""
	if leaseToken != "" {
		leaseTokenSource = "lease_take_token"
	}
	if leaseToken == "" && inputTakeToken != "" {
		leaseToken = inputTakeToken
		if submitLease != nil {
			leaseTokenSource = "input_take_token"
		} else {
			leaseTokenSource = "input_take_token_without_lease"
		}
	}
	if leaseToken == "" {
		purged := map[string]any{"queue_removed": 0, "lease_removed": 0, "skipped": "lease_missing"}
		if submitLease == nil {
			purged["skipped"] = "lease_missing"
		} else {
			purged["skipped"] = "token_missing"
		}
		entry := map[string]any{
			"api_type": submitAPIType, "ip": reqCtx.ip, "username": pluginUsername, "api1_username": api1Auth.Username,
			"task_api_type": submitAPIType, "task_account_key": trimStr(submitTaskAccount["key"]), "input_task_account_key": taskAccountKey,
			"lease_task_account_key": condStr(submitLease != nil, trimStr(submitLease["task_account_key"])),
			"lease_found": submitLease != nil, "token_source": "missing_take_token", "task_runtime_scope": taskRuntimeScope,
			"status": "local_lease_missing", "source": "decoded_extension_submit", "task_id": taskID, "task_url": submitURL,
			"forward_url": submitTargetURL, "error": "missing active take lease, submit blocked to avoid stale deal_id/token mismatch; please fetch a fresh task",
			"purged_runtime": purged, "timing_ms": map[string]any{"forward": 0, "total": int(timeNow().Sub(submitStart).Milliseconds())},
		}
		appendLog(logFile, entry, maxLogs)
		appendLog(decodedExtSubmitLogF, entry, 1000)
		decodedExtJSON(w, 409, map[string]any{
			"success": false, "error": "missing active take lease, submit blocked to avoid stale deal_id/token mismatch; please fetch a fresh task",
			"retryable": false, "leaseMissing": true, "_proxy": map[string]any{"submit_accepted": false, "status": "local_lease_missing"},
		})
		return
	}
	if submitLease == nil && leaseTokenSource == "input_take_token_without_lease" {
		leaseRecordURL := leaseURL
		if leaseRecordURL == "" {
			leaseRecordURL = submitURL
		}
		decodedExtRecordTaskLease(runtimeAccount, map[string]any{"deal_id": taskID, "dealId": taskID, "taskId": taskID, "url": leaseRecordURL, "taskUrl": leaseRecordURL}, submitTaskAccount, leaseToken)
	}
	tokenInfo := tokenInfo{OK: true, Token: leaseToken, Cached: true, ExpiresAt: int64(jwtExp(leaseToken))}
	if leaseToken == "" {
		tf := decodedExtAPI1TokenFileForAccount(trimStr(pluginAccount["accountId"]), api1Auth)
		tokenInfo = getAPIToken(api1Auth, tf, false)
	}
	if !tokenInfo.OK {
		decodedExtJSON(w, 502, map[string]any{"success": false, "error": firstNonNilStr(tokenInfo.Msg, "api1 login failed")})
		return
	}
	submitToken := tokenInfo.Token
	submitTokenSource := leaseTokenSource
	if submitTokenSource == "" {
		submitTokenSource = "account_cache"
	}
	submitTokenHash := sha256short(submitToken)

	// 上游提交时会校验 url 里的 item_id == result 的 data.item.item_id，
	// 短链接 https://shopee.tw/product/{shop}/{item} 没有 item_id= 参数会被判「结果有误」，
	// 这里优先选用带 item_id= 的 API 链接（rawResponseUrl/apiUrl），必要时用 result 重建 get_pc 链接。
	forwardURL := refineSubmitURL(submitURL, input, submitResult)
	forwardBody := jsonEncode(map[string]any{
		"appVersion": firstNonNilStr(input["appVersion"], "vv2"),
		"url":        forwardURL,
		"result":     submitResult,
	})
	result := forwardToServer(submitTargetURL, forwardBody, submitGzip, []string{"Authorization: Bearer " + submitToken})
	var serverResp map[string]any
	_ = jsonDecode(result.Response, &serverResp)
	forwardMs := int(timeNow().Sub(submitStart).Milliseconds()) - result.GateWaitMs
	isForwardSuccess := api1ForwardSuccess(serverResp)
	if ajieRequest {
		isForwardSuccess = api1ForwardSuccessForAjie(serverResp, result.HTTPCode, result.Response)
	}
	forwardAttempt := forwardAttemptSummary(result, serverResp)
	forwardAttempt["attempt"] = 1
	forwardAttempt["total_ms"] = forwardMs
	forwardAttempt["submit_url"] = submitURL
	forwardAttempt["api1_username"] = api1Auth.Username
	forwardAttempt["task_api_type"] = submitAPIType
	forwardAttempt["token_source"] = submitTokenSource
	forwardAttempt["token_hash"] = submitTokenHash

	postSubmitCleanup := map[string]any{"lease_removed": 0, "runtime_purged": nil}
	if isForwardSuccess {
		postSubmitCleanup["lease_removed"] = decodedExtForgetTaskLease(runtimeAccount, taskID, leaseURL)
	} else if api1SubmitDealNotFound(serverResp) {
		postSubmitCleanup["lease_removed"] = decodedExtForgetTaskLease(runtimeAccount, taskID, leaseURL)
		postSubmitCleanup["runtime_purged"] = decodedExtClearTaskRuntimeState(runtimeAccount)
	}

	status := "task_server_error"
	if result.Error != "" {
		status = "forward_fail"
	} else if isForwardSuccess {
		status = "success"
	}
	entry := map[string]any{
		"api_type": submitAPIType, "ip": reqCtx.ip, "username": pluginUsername, "api1_username": api1Auth.Username,
		"task_api_type": submitAPIType, "input_task_api_type": inputTaskAPIType,
		"device_id": deviceContext["device_id"], "platform": deviceContext["platform"], "device_type": deviceContext["device_type"],
		"device_label": deviceContext["device_label"], "client": deviceContext["client"], "client_label": resolveClientLabel(deviceContext),
		"runtime_scope": trimStr(runtimeAccount["__storage_scope"]), "task_runtime_scope": taskRuntimeScope,
		"task_account_key": trimStr(submitTaskAccount["key"]), "task_account_username": trimStr(submitTaskAccount["username"]),
		"input_task_account_key": taskAccountKey,
		"lease_task_account_key": condStr(submitLease != nil, trimStr(submitLease["task_account_key"])),
		"lease_found": submitLease != nil, "token_source": submitTokenSource, "token_hash": submitTokenHash,
		"post_submit_cleanup": postSubmitCleanup, "status": status, "source": "decoded_extension_submit",
		"task_id": taskID, "task_url": submitURL, "forwarded_task_url": forwardURL, "forward_url": submitTargetURL,
		"forward_result_len": len(submitResult), "forward_result_sha256": submitResultDiag["result_sha256"],
		"forward_result_json_valid": submitResultDiag["result_json_valid"], "forward_result_json_error": submitResultDiag["result_json_error"],
		"forward_result_first_char": submitResultDiag["result_first_char"], "forward_result_last_char": submitResultDiag["result_last_char"],
		"api2_null_fields": api2NullFields, "forward_attempts": []map[string]any{forwardAttempt},
		"response": compactLogResponse(serverResp, maxLogRespBytes), "gate_wait_ms": result.GateWaitMs,
		"local_ms": max(0, int(timeNow().Sub(submitStart).Milliseconds())-forwardMs-result.GateWaitMs),
		"timing_ms": map[string]any{
			"forward": forwardMs, "gate_wait": result.GateWaitMs,
			"local": max(0, int(timeNow().Sub(submitStart).Milliseconds())-forwardMs-result.GateWaitMs),
			"total":  int(timeNow().Sub(submitStart).Milliseconds()),
		},
	}
	if result.Error != "" {
		entry["error"] = result.Error
	}
	appendLog(logFile, entry, maxLogs)
	appendLog(decodedExtSubmitLogF, entry, 1000)
	saveUploadDump(dateTimeStamp()+"_"+md5short(submitURL+"|"+taskID), map[string]any{
		"api_type": submitAPIType, "source": "decoded_extension_submit", "target_url": submitTargetURL,
		"request_body": mustJSONMap(forwardBody), "http_code": result.HTTPCode,
		"response_json": serverResp, "response_raw": result.Response, "error": result.Error,
		"task_id": taskID, "api1_username": api1Auth.Username, "task_api_type": submitAPIType,
		"input_task_api_type": inputTaskAPIType, "task_account_key": trimStr(submitTaskAccount["key"]),
		"input_task_account_key": taskAccountKey, "lease_task_account_key": condStr(submitLease != nil, trimStr(submitLease["task_account_key"])),
		"lease_found": submitLease != nil, "token_source": submitTokenSource, "token_hash": submitTokenHash,
		"forward_result_len": len(submitResult), "forward_result_sha256": submitResultDiag["result_sha256"],
		"forward_result_json_valid": submitResultDiag["result_json_valid"], "forward_result_json_error": submitResultDiag["result_json_error"],
		"forward_result_first_char": submitResultDiag["result_first_char"], "forward_result_last_char": submitResultDiag["result_last_char"],
		"api2_null_fields": api2NullFields, "post_submit_cleanup": postSubmitCleanup,
	})

	if result.Error != "" {
		decodedExtJSON(w, 502, map[string]any{"success": false, "error": "submit forward failed: " + result.Error, "retryable": true})
		return
	}
	if isForwardSuccess {
		if !ajieRequest {
			decodedExtJSON(w, 200, map[string]any{
				"success": true, "data": map[string]any{"raw": serverRespOrRaw(serverResp)},
				"_proxy": map[string]any{"submit_accepted": true, "status": status},
			})
			return
		}
		upstreamRaw := serverRespOrRaw(serverResp)
		decodedExtJSON(w, 200, map[string]any{
			"code": "200", "success": true, "ok": true, "status": "success", "msg": "ok",
			"data": map[string]any{"raw": upstreamRaw}, "upstream": upstreamRaw,
			"_proxy": map[string]any{"submit_accepted": true, "status": "success", "upstream_status": status},
		})
		return
	}
	upstreamError := api1SubmitFailureMessage(serverResp, result.Response)
	decodedExtJSON(w, 400, map[string]any{
		"success": false, "error": upstreamError, "upstreamSubmitFailed": true, "retryable": false,
		"data": map[string]any{"raw": serverRespOrRaw(serverResp)},
		"_proxy": map[string]any{"submit_accepted": false, "status": status},
	})
}

func serverRespOrRaw(serverResp map[string]any) any {
	if serverResp != nil {
		return serverResp
	}
	return decodedExtRawSuccess()
}

func anyToStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		return jsonEncode(x)
	case []any:
		return jsonEncode(x)
	case nil:
		return ""
	default:
		return jsonEncode(x)
	}
}

func mustJSONMap(s string) map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(s), &m)
	return m
}

func dateTimeStamp() string {
	return timeNow().Format("20060102_150405")
}

// decryptMarsSubmitData：明文 JSON 直接返回（包装结构则取出真正商品数据）；否则 AES 解密后同样取出
func decryptMarsSubmitData(data string) (string, bool) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", false
	}
	var plainInput any
	if json.Unmarshal([]byte(data), &plainInput) == nil {
		if extracted := extractSubmitResult(plainInput); extracted != "" {
			return extracted, true
		}
		return data, true
	}
	plainStr, ok := aesDecryptString(data, aesKey)
	if !ok {
		return "", false
	}
	var decrypted map[string]any
	if json.Unmarshal([]byte(plainStr), &decrypted) != nil {
		return plainStr, true
	}
	if extracted := extractSubmitResult(decrypted); extracted != "" {
		return extracted, true
	}
	return plainStr, true
}

// extractSubmitResult 从包装结构里取出真正的商品数据（去掉 headers/元数据等多余内容）
func extractSubmitResult(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	// 1) body：PDP 商品 JSON（字符串）
	if body, ok := m["body"].(string); ok && body != "" {
		return body
	}
	// 2) dumps：搜索/相关商品结果（对象或字符串）
	if dumps, exists := m["dumps"]; exists && dumps != nil {
		if s, ok := dumps.(string); ok && s != "" {
			return s
		}
		if b, err := json.Marshal(dumps); err == nil && string(b) != "null" {
			return string(b)
		}
	}
	// 3) 旧版 responseContent.response_body
	if rc, ok := m["responseContent"].(map[string]any); ok {
		if rb, ok := rc["response_body"].(string); ok && rb != "" {
			return rb
		}
	}
	// 4) response_body / responseBody
	for _, k := range []string{"response_body", "responseBody"} {
		if rb, ok := m[k].(string); ok && rb != "" {
			return rb
		}
	}
	return ""
}

// submitItemIDFromURL：从任务 URL 提取 item_id（优先 query item_id=，其次 /product/{shop}/{item} 短链最后一段数字）
func submitItemIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.Index(raw, "?"); i >= 0 {
		query := raw[i+1:]
		for _, kv := range strings.Split(query, "&") {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				k := toLower(strings.TrimSpace(parts[0]))
				if k == "item_id" || k == "itemid" || k == "item" {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	clean := raw
	if i := strings.IndexAny(clean, "?#"); i >= 0 {
		clean = clean[:i]
	}
	segs := strings.Split(strings.Trim(clean, "/"), "/")
	for i := len(segs) - 1; i >= 0; i-- {
		seg := strings.TrimSpace(segs[i])
		if seg != "" && isAllDigits(seg) {
			return seg
		}
	}
	return ""
}

// submitItemIDFromResult：从 result 商品 JSON 提取 data.item.item_id
func submitItemIDFromResult(result string) string {
	result = strings.TrimSpace(result)
	if result == "" {
		return ""
	}
	var v map[string]any
	if json.Unmarshal([]byte(result), &v) != nil {
		return ""
	}
	data, _ := v["data"].(map[string]any)
	if data == nil {
		return ""
	}
	item, _ := data["item"].(map[string]any)
	if item == nil {
		return ""
	}
	return trimStr(firstNonNil(item["item_id"], item["itemId"], item["itemid"]))
}

// refineSubmitURL：选择提交给上游的 url。上游会校验 url 里的 item_id 与 result 的 data.item.item_id 是否一致，
// 短链接 https://shopee.tw/product/{shop}/{item} 没有 item_id= 参数，上游会判「结果有误」。
// 因此优先返回带 item_id= 的 API 链接（rawResponseUrl/apiUrl），否则用 result 重建 get_pc 链接。
func refineSubmitURL(current string, input map[string]any, result string) string {
	current = trimStr(current)
	if strings.Contains(toLower(current), "item_id=") {
		return current
	}
	for _, k := range []string{"rawResponseUrl", "apiUrl", "productUrl", "pageUrl"} {
		c := trimStr(firstNonNil(input[k], ""))
		if c != "" && strings.Contains(toLower(c), "item_id=") {
			return c
		}
	}
	if rebuilt := rebuildSubmitGetPCURL(result); rebuilt != "" {
		return rebuilt
	}
	return current
}

// rebuildSubmitGetPCURL：用 result 里的 item_id/shop_id 重建 shopee get_pc 接口地址
func rebuildSubmitGetPCURL(result string) string {
	itemID := submitItemIDFromResult(result)
	if itemID == "" {
		return ""
	}
	var v map[string]any
	if json.Unmarshal([]byte(result), &v) != nil {
		return ""
	}
	shopID := ""
	if data, ok := v["data"].(map[string]any); ok {
		if item, ok := data["item"].(map[string]any); ok {
			shopID = trimStr(firstNonNil(item["shop_id"], item["shopId"], item["shopid"]))
		}
	}
	if shopID == "" {
		return ""
	}
	return "https://shopee.tw/api/v4/pdp/get_pc?display_model_id=0&item_id=" + itemID +
		"&model_selection_logic=3&shop_id=" + shopID + "&tz_offset_in_minutes=480&detail_level=0"
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// api2SanitizeSubmitResult：接口2 结果清洗（PHP 中把 null 字段置空）
func api2SanitizeSubmitResult(result string) (string, int) {
	var v any
	if json.Unmarshal([]byte(result), &v) != nil {
		return result, 0
	}
	nulls := sanitizeNulls(v)
	return jsonEncode(v), nulls
}

func sanitizeNulls(v any) int {
	count := 0
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if val == nil {
				x[k] = ""
				count++
			} else {
				count += sanitizeNulls(val)
			}
		}
	case []any:
		for i, val := range x {
			if val == nil {
				x[i] = ""
				count++
			} else {
				count += sanitizeNulls(val)
			}
		}
	}
	return count
}

// api1ParseTimeoutResponse
func api1ParseTimeoutResponse(serverResp map[string]any, raw string) bool {
	if serverResp != nil {
		msg := strings.ToLower(trimStr(firstNonNil(serverResp["msg"], serverResp["error"])))
		return strings.Contains(msg, "timeout") || strings.Contains(msg, "超时")
	}
	raw = strings.ToLower(raw)
	return strings.Contains(raw, "timeout") || strings.Contains(raw, "超时")
}
