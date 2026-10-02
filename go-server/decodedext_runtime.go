package main

import (
	"encoding/json"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ---- 任务队列 / 租约存储（内存 + JSON 持久化，替代 PHP 的 SQLite）----

var runtimeLocks = newSyncMap()

func runtimeLock(key string) *sync.Mutex {
	mu, _ := runtimeLocks.get(key)
	if mu == nil {
		m := &sync.Mutex{}
		runtimeLocks.set(key, m)
		return m
	}
	return mu.(*sync.Mutex)
}

func decodedExtAccountStorageSuffix(pluginAccount map[string]any) string {
	accountID := trimStr(firstNonNil(pluginAccount["accountId"], pluginAccount["id"], pluginAccount["name"]))
	if accountID == "" {
		return ""
	}
	scope := trimStr(pluginAccount["__storage_scope"])
	identity := accountID
	safe := accountID
	if scope != "" {
		identity = accountID + "|" + scope
		safe = accountID + "_" + scope
	}
	safe = sanitizeStorage(safe)
	if safe == "" {
		safe = "account"
	}
	if len(safe) > 64 {
		safe = safe[:64]
	}
	return safe + "_" + md5short(identity)
}

func sanitizeStorage(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	return strings.Trim(sb.String(), "_")
}

func decodedExtRuntimeAccount(pluginAccount map[string]any, deviceContext map[string]any, forcedScope string) map[string]any {
	runtime := map[string]any{}
	for k, v := range pluginAccount {
		runtime[k] = v
	}
	if forcedScope != "" {
		runtime["__storage_scope"] = forcedScope
		runtime["__device_id"] = trimStr(deviceContext["device_id"])
		return runtime
	}
	deviceID := trimStr(deviceContext["device_id"])
	clientInstance := trimStr(deviceContext["client_instance_id"])
	fp := trimStr(firstNonNil(deviceContext["fingerprint_key"], deviceContext["fingerprint_id"]))
	var parts []string
	for _, v := range []string{deviceID, clientInstance, fp} {
		if v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) > 0 {
		runtime["__storage_scope"] = strings.Join(parts, "|")
		runtime["__device_id"] = deviceID
	}
	return runtime
}

func decodedExtQueueFileForAccount(pluginAccount map[string]any) string {
	suffix := decodedExtAccountStorageSuffix(pluginAccount)
	if suffix == "" {
		return decodedExtTaskQueueF
	}
	return decodedExtRuntimeDir + "/decoded_extension_tasks_" + suffix + ".json"
}

func decodedExtStateFileForAccount(pluginAccount map[string]any) string {
	suffix := decodedExtAccountStorageSuffix(pluginAccount)
	if suffix == "" {
		return decodedExtStateF
	}
	return decodedExtRuntimeDir + "/.decoded_extension_state_" + suffix + ".json"
}

// ---- 任务规范化 ----
func decodedExtTaskURL(task map[string]any) string {
	for _, k := range []string{"url", "taskUrl", "task_url", "productUrl", "product_url", "externalTaskUrl"} {
		if v := trimStr(task[k]); v != "" {
			return v
		}
	}
	return ""
}

func decodedExtTaskDealID(task map[string]any) string {
	for _, k := range []string{"deal_id", "dealId", "dealID", "taskId", "task_id", "tid", "trace_id", "id"} {
		if v := trimStr(task[k]); v != "" {
			return v
		}
	}
	return ""
}

func decodedExtNormalizeTask(task map[string]any) map[string]any {
	if decodedExtTaskURL(task) == "" {
		return nil
	}
	return task
}

func decodedExtIsList(v any) bool {
	switch v.(type) {
	case []any:
		return true
	}
	return false
}

func decodedExtLoadQueue(file string) map[string]any {
	out := map[string]any{"wrapper": nil, "tasks": []any{}}
	var data any
	if !readJSONFile(dp(file), &data) {
		return out
	}
	if arr, ok := data.([]any); ok {
		out["tasks"] = arr
		return out
	}
	if m, ok := data.(map[string]any); ok {
		out["wrapper"] = m
		tasks := m["tasks"]
		if tasks == nil {
			tasks = m["data"]
		}
		if tasks == nil {
			tasks = m["queue"]
		}
		if arr, ok := tasks.([]any); ok {
			out["tasks"] = arr
		}
	}
	return out
}

func decodedExtSaveQueue(file string, wrapper map[string]any, tasks []any) {
	var payload any
	if wrapper == nil {
		payload = tasks
	} else {
		wrapper["tasks"] = tasks
		payload = wrapper
	}
	_ = atomicWrite(dp(file), jsonEncodePretty(payload))
}

// appendLocalTasks
func decodedExtAppendLocalTasks(tasks []map[string]any, pluginAccount map[string]any) {
	if len(tasks) == 0 {
		return
	}
	file := decodedExtQueueFileForAccount(pluginAccount)
	mu := runtimeLock(file)
	mu.Lock()
	defer mu.Unlock()
	q := decodedExtLoadQueue(file)
	queued, _ := q["tasks"].([]any)
	seen := map[string]bool{}
	var normalizedQueued []map[string]any
	for _, t := range queued {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		n := decodedExtNormalizeTask(tm)
		if n == nil {
			continue
		}
		normalizedQueued = append(normalizedQueued, n)
		seen[queueDedupeKey(n)] = true
	}
	for _, t := range tasks {
		n := decodedExtNormalizeTask(t)
		if n == nil {
			continue
		}
		if _, ok := n["__queued_ts"]; !ok {
			n["__queued_ts"] = timeNowUnix()
		}
		if _, ok := n["__queued_at"]; !ok {
			n["__queued_at"] = tsDateTime()
		}
		k := queueDedupeKey(n)
		if seen[k] {
			continue
		}
		normalizedQueued = append(normalizedQueued, n)
		seen[k] = true
	}
	arr := make([]any, len(normalizedQueued))
	for i, t := range normalizedQueued {
		arr[i] = t
	}
	decodedExtSaveQueue(file, q["wrapper"].(map[string]any), arr)
}

func queueDedupeKey(t map[string]any) string {
	id := trimStr(firstNonNil(t["taskId"], t["task_id"], t["tid"]))
	return id + "|" + decodedExtTaskURL(t)
}

// popLocalTasks
func decodedExtPopLocalTasks(count int, pluginAccount map[string]any) []map[string]any {
	count = max(1, count)
	file := decodedExtQueueFileForAccount(pluginAccount)
	mu := runtimeLock(file)
	mu.Lock()
	defer mu.Unlock()
	q := decodedExtLoadQueue(file)
	tasks, _ := q["tasks"].([]any)
	if len(tasks) == 0 {
		return []map[string]any{}
	}
	var picked []map[string]any
	now := timeNowUnix()
	var remaining []any
	for _, t := range tasks {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		task := decodedExtNormalizeTask(tm)
		if task == nil {
			continue
		}
		queuedTs := int64(qInt(task["__queued_ts"]))
		if queuedTs > 0 && decodedExtQueueTTL > 0 && queuedTs < now-int64(decodedExtQueueTTL) {
			continue
		}
		hasBinding := trimStr(firstNonNil(task["task_account_key"], task["taskAccountKey"], task["task_account_username"], task["taskAccountUsername"])) != ""
		if hasBinding && len(picked) < count {
			picked = append(picked, task)
		} else {
			remaining = append(remaining, task)
		}
	}
	decodedExtSaveQueue(file, q["wrapper"].(map[string]any), remaining)
	return picked
}

func decodedExtClearLocalTasks(pluginAccount map[string]any) int {
	file := decodedExtQueueFileForAccount(pluginAccount)
	mu := runtimeLock(file)
	mu.Lock()
	defer mu.Unlock()
	q := decodedExtLoadQueue(file)
	tasks, _ := q["tasks"].([]any)
	n := len(tasks)
	decodedExtSaveQueue(file, q["wrapper"].(map[string]any), []any{})
	return n
}

// ---- 状态（rotation + leases）----
func decodedExtLoadState(pluginAccount map[string]any) map[string]any {
	file := decodedExtStateFileForAccount(pluginAccount)
	state := map[string]any{"rotation": map[string]any{}, "leases": map[string]any{}}
	_ = readJSONFile(dp(file), &state)
	if _, ok := state["rotation"].(map[string]any); !ok {
		state["rotation"] = map[string]any{}
	}
	if _, ok := state["leases"].(map[string]any); !ok {
		state["leases"] = map[string]any{}
	}
	return state
}

func decodedExtSaveState(state map[string]any, pluginAccount map[string]any) {
	file := decodedExtStateFileForAccount(pluginAccount)
	leases, _ := state["leases"].(map[string]any)
	now := timeNowUnix()
	for k, v := range leases {
		lease, ok := v.(map[string]any)
		if !ok {
			delete(leases, k)
			continue
		}
		created := int64(qInt(lease["created_ts"]))
		if created > 0 && decodedExtLeaseTTL > 0 && created < now-int64(decodedExtLeaseTTL) {
			delete(leases, k)
		}
	}
	_ = atomicWrite(dp(file), jsonEncodePretty(state))
}

func decodedExtMutateState(mutator func(*map[string]any) any, pluginAccount map[string]any) any {
	file := decodedExtStateFileForAccount(pluginAccount)
	mu := runtimeLock(file)
	mu.Lock()
	defer mu.Unlock()
	state := decodedExtLoadState(pluginAccount)
	res := mutator(&state)
	decodedExtSaveState(state, pluginAccount)
	return res
}

func decodedExtLeaseKey(pluginAccount map[string]any, taskID, url string) string {
	identity := taskID
	if identity == "" {
		identity = url
	}
	return md5hex(trimStr(pluginAccount["accountId"]) + "|" + trimStr(pluginAccount["__storage_scope"]) + "|" + identity)
}

func decodedExtLeaseBelongsToAccount(lease, pluginAccount map[string]any) bool {
	leaseAccountID := trimStr(lease["plugin_account_id"])
	accountID := trimStr(pluginAccount["accountId"])
	return leaseAccountID == "" || accountID == "" || leaseAccountID == accountID
}

func decodedExtLeaseMatchesTask(lease, pluginAccount map[string]any, taskID, url string) bool {
	if !decodedExtLeaseBelongsToAccount(lease, pluginAccount) {
		return false
	}
	sameTask := taskID != "" && trimStr(lease["task_id"]) == taskID
	sameURL := url != "" && trimStr(lease["url"]) == url
	return sameTask || sameURL
}

func decodedExtFindTaskLease(pluginAccount map[string]any, taskID, url string) map[string]any {
	state := decodedExtLoadState(pluginAccount)
	leases, _ := state["leases"].(map[string]any)
	key := decodedExtLeaseKey(pluginAccount, taskID, url)
	if l, ok := leases[key].(map[string]any); ok && decodedExtLeaseMatchesTask(l, pluginAccount, taskID, url) {
		return l
	}
	var best map[string]any
	for _, v := range leases {
		l, ok := v.(map[string]any)
		if !ok || !decodedExtLeaseMatchesTask(l, pluginAccount, taskID, url) {
			continue
		}
		if best == nil || int64(qInt(l["created_ts"])) >= int64(qInt(best["created_ts"])) {
			best = l
		}
	}
	return best
}

func decodedExtRecordTaskLease(pluginAccount, task, taskAccount map[string]any, api1Token string) {
	taskID := decodedExtTaskDealID(task)
	url := decodedExtTaskURL(task)
	if taskID == "" && url == "" {
		return
	}
	decodedExtMutateState(func(state *map[string]any) any {
		s := *state
		leases, _ := s["leases"].(map[string]any)
		key := decodedExtLeaseKey(pluginAccount, taskID, url)
		existing, _ := leases[key].(map[string]any)
		if api1Token == "" && existing != nil {
			api1Token = trimStr(existing["api1_token"])
		}
		lease := map[string]any{
			"plugin_account_id": trimStr(pluginAccount["accountId"]),
			"task_account_key":  trimStr(taskAccount["key"]),
			"task_account_username": trimStr(taskAccount["username"]),
			"task_id": taskID, "url": url,
			"created_ts": timeNowUnix(), "created_at": tsDateTime(),
		}
		if api1Token != "" {
			lease["api1_token"] = api1Token
			lease["api1_token_hash"] = sha256short(api1Token)
			lease["api1_token_expires_at"] = jwtExp(api1Token)
		}
		leases[key] = lease
		return true
	}, pluginAccount)
}

func decodedExtForgetTaskLease(pluginAccount map[string]any, taskID, url string) int {
	return qInt(decodedExtMutateState(func(state *map[string]any) any {
		s := *state
		leases, _ := s["leases"].(map[string]any)
		removed := 0
		for key, v := range leases {
			l, ok := v.(map[string]any)
			if !ok {
				continue
			}
			sameTask := taskID != "" && trimStr(l["task_id"]) == taskID
			sameURL := url != "" && trimStr(l["url"]) == url
			if sameTask || sameURL {
				delete(leases, key)
				removed++
			}
		}
		return removed
	}, pluginAccount))
}

func decodedExtClearTaskRuntimeState(pluginAccount map[string]any) map[string]any {
	queueRemoved := decodedExtClearLocalTasks(pluginAccount)
	leaseRemoved := qInt(decodedExtMutateState(func(state *map[string]any) any {
		s := *state
		leases, _ := s["leases"].(map[string]any)
		n := len(leases)
		s["leases"] = map[string]any{}
		return n
	}, pluginAccount))
	return map[string]any{"queue_removed": queueRemoved, "lease_removed": leaseRemoved}
}

func decodedExtActiveTaskLeaseCount(pluginAccount map[string]any) int {
	state := decodedExtLoadState(pluginAccount)
	leases, _ := state["leases"].(map[string]any)
	now := timeNowUnix()
	seen := map[string]bool{}
	for _, v := range leases {
		l, ok := v.(map[string]any)
		if !ok || !decodedExtLeaseBelongsToAccount(l, pluginAccount) {
			continue
		}
		created := int64(qInt(l["created_ts"]))
		if created > 0 && created < now-int64(decodedExtLeaseTTL) {
			continue
		}
		key := trimStr(l["task_id"]) + "|" + trimStr(l["url"])
		if key == "|" {
			key = md5hex(jsonEncode(l))
		}
		seen[key] = true
	}
	return len(seen)
}

func decodedExtTaskAccountAPITypeMap(pluginAccount map[string]any) map[string]map[string]int {
	m := map[string]map[string]int{"key": {}, "username": {}}
	for _, ta := range decodedExtNormalizeAllTaskAccounts(pluginAccount, false) {
		apiType := decodedExtAPITypeForTaskAccount(ta)
		if k := trimStr(ta["key"]); k != "" {
			m["key"][k] = apiType
		}
		if u := trimStr(ta["username"]); u != "" {
			m["username"][u] = apiType
		}
	}
	return m
}

func decodedExtActiveTaskLeaseCountsByType(pluginAccount map[string]any) map[int]int {
	counts := map[int]int{0: 0, 1: 0, 2: 0}
	apiTypeMap := decodedExtTaskAccountAPITypeMap(pluginAccount)
	state := decodedExtLoadState(pluginAccount)
	leases, _ := state["leases"].(map[string]any)
	now := timeNowUnix()
	seen := map[string]bool{}
	for _, v := range leases {
		l, ok := v.(map[string]any)
		if !ok || !decodedExtLeaseBelongsToAccount(l, pluginAccount) {
			continue
		}
		created := int64(qInt(l["created_ts"]))
		if created > 0 && created < now-int64(decodedExtLeaseTTL) {
			continue
		}
		key := trimStr(l["task_id"]) + "|" + trimStr(l["url"])
		if key == "|" {
			key = md5hex(jsonEncode(l))
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		apiType := 0
		if k := trimStr(l["task_account_key"]); k != "" {
			apiType = apiTypeMap["key"][k]
		}
		if apiType == 0 {
			if u := trimStr(l["task_account_username"]); u != "" {
				apiType = apiTypeMap["username"][u]
			}
		}
		if apiType == 1 || apiType == 2 {
			counts[apiType]++
		} else {
			counts[0]++
		}
	}
	return counts
}

func decodedExtTaskAPITypeFromMeta(pluginAccount, task map[string]any) int {
	direct := qInt(firstNonNil(task["task_api_type"], task["api_type"], task["taskApiType"], 0))
	if direct == 1 || direct == 2 {
		return direct
	}
	apiTypeMap := decodedExtTaskAccountAPITypeMap(pluginAccount)
	if k := trimStr(firstNonNil(task["task_account_key"], task["taskAccountKey"])); k != "" {
		return apiTypeMap["key"][k]
	}
	if u := trimStr(firstNonNil(task["task_account_username"], task["taskAccountUsername"])); u != "" {
		return apiTypeMap["username"][u]
	}
	return 0
}

// ---- 轮转 + 拉任务 ----
func decodedExtOrderedTaskAccountsByType(pluginAccount map[string]any, apiType int) []map[string]any {
	taskAccounts := decodedExtEnabledTaskAccountsByType(pluginAccount, apiType)
	if len(taskAccounts) == 0 {
		return []map[string]any{}
	}
	accountID := trimStr(pluginAccount["accountId"])
	rotationKey := accountID + "|api" + strconvItoa(apiType)
	res := decodedExtMutateState(func(state *map[string]any) any {
		s := *state
		rb, _ := s["rotation_by_api"].(map[string]any)
		if rb == nil {
			rb = map[string]any{}
			s["rotation_by_api"] = rb
		}
		start := qInt(rb[rotationKey])
		n := len(taskAccounts)
		var ordered []map[string]any
		for i := 0; i < n; i++ {
			ordered = append(ordered, taskAccounts[(start+i)%n])
		}
		rb[rotationKey] = (start + 1) % n
		return ordered
	}, pluginAccount)
	if arr, ok := res.([]map[string]any); ok {
		return arr
	}
	return []map[string]any{}
}

func decodedExtTakeTaskFromAPI1Response(serverResp map[string]any) map[string]any {
	candidate := findTaskInValue(serverResp, 0)
	if candidate == nil {
		return nil
	}
	return decodedExtNormalizeTask(candidate)
}

func findTaskInValue(v any, depth int) map[string]any {
	if depth > 8 {
		return nil
	}
	switch val := v.(type) {
	case string:
		trimmed := strings.TrimSpace(val)
		if trimmed != "" && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
			var decoded any
			if json.Unmarshal([]byte(trimmed), &decoded) == nil {
				return findTaskInValue(decoded, depth+1)
			}
		}
		return nil
	case map[string]any:
		if decodedExtTaskURL(val) != "" {
			return val
		}
		for _, k := range []string{"task", "data", "result", "response", "payload", "content", "body", "queue"} {
			if child, ok := val[k]; ok {
				if found := findTaskInValue(child, depth+1); found != nil {
					return found
				}
			}
		}
		return nil
	case []any:
		for _, item := range val {
			if found := findTaskInValue(item, depth+1); found != nil {
				return found
			}
		}
		return nil
	}
	return nil
}

// takeFromAPI1Pool：并发拉取多个任务账号（Go 原生并发，替代 PHP curl_multi）
func decodedExtTakeTaskFromAPI1Pool(pluginAccount map[string]any, orderedAccounts []map[string]any, takeType string, serial bool) map[string]any {
	var attempts []map[string]any
	type preparedEntry struct {
		taskAccount map[string]any
		api1Auth    upstreamAuth
		apiType     int
		takeURL     string
		token       string
	}
	var prepared []preparedEntry
	lastMsg := "no task"
	for _, ta := range orderedAccounts {
		api1Auth := decodedExtAPI1AuthForTaskAccount(pluginAccount, ta)
		tf := decodedExtAPI1TokenFileForAccount(trimStr(pluginAccount["accountId"]), api1Auth)
		tokenInfo := getAPIToken(api1Auth, tf, false)
		if !tokenInfo.OK {
			attempts = append(attempts, map[string]any{"api1_username": api1Auth.Username, "error": firstNonNilStr(tokenInfo.Msg, "api1 login failed")})
			lastMsg = firstNonNilStr(tokenInfo.Msg, "api1 login failed")
			continue
		}
		apiType := decodedExtAPITypeForTaskAccount(ta)
		prepared = append(prepared, preparedEntry{
			taskAccount: ta, api1Auth: api1Auth, apiType: apiType,
			takeURL: apiTakeURLForType(apiType, takeType), token: tokenInfo.Token,
		})
	}
	if len(prepared) == 0 {
		return map[string]any{"task": nil, "task_account": nil, "attempts": attempts, "last_msg": lastMsg}
	}

	type foundTask struct {
		task         map[string]any
		taskAccount  map[string]any
		api1Token    string
		api1Username string
		apiType      int
	}

	fetchOne := func(e preparedEntry) (foundTask, map[string]any) {
		res := forwardGetToServer(e.takeURL, []string{"Authorization: Bearer " + e.token})
		var serverResp map[string]any
		_ = jsonDecode(res.Response, &serverResp)
		task := decodedExtTakeTaskFromAPI1Response(serverResp)
		attempt := forwardAttemptSummary(res, serverResp)
		attempt["api1_username"] = e.api1Auth.Username
		attempt["api_type"] = e.apiType
		attempt["forward_url"] = e.takeURL
		attempt["token_hash"] = sha256short(e.token)
		if task != nil {
			return foundTask{task: task, taskAccount: e.taskAccount, api1Token: e.token, api1Username: e.api1Auth.Username, apiType: e.apiType}, attempt
		}
		if res.Error != "" {
			lastMsg = "take task forward failed: " + res.Error
		} else {
			if serverResp != nil {
				if data, ok := serverResp["data"].(map[string]any); ok {
					lastMsg = trimStr(firstNonNil(serverResp["msg"], data["msg"], "no task"))
				} else {
					lastMsg = trimStr(firstNonNil(serverResp["msg"], "no task"))
				}
			} else {
				lastMsg = "no task"
			}
		}
		return foundTask{}, attempt
	}

	var found []foundTask
	attemptMu := &sync.Mutex{}
	appendAttempt := func(a map[string]any, idx int) {
		a["attempt"] = idx
		attemptMu.Lock()
		attempts = append(attempts, a)
		attemptMu.Unlock()
	}

	if serial || len(prepared) == 1 {
		for i, e := range prepared {
			ft, attempt := fetchOne(e)
			appendAttempt(attempt, i+1)
			if ft.task != nil {
				found = append(found, ft)
			}
		}
	} else {
		results := make([]foundTask, len(prepared))
		attemptsArr := make([]map[string]any, len(prepared))
		var wg sync.WaitGroup
		for i, e := range prepared {
			wg.Add(1)
			go func(i int, e preparedEntry) {
				defer wg.Done()
				ft, attempt := fetchOne(e)
				results[i] = ft
				attemptsArr[i] = attempt
			}(i, e)
		}
		wg.Wait()
		for i := range results {
			if results[i].task != nil {
				found = append(found, results[i])
			}
			appendAttempt(attemptsArr[i], i+1)
		}
	}

	if len(found) > 0 {
		first := found[0]
		var extras []map[string]any
		for _, f := range found[1:] {
			extras = append(extras, map[string]any{
				"task": f.task, "task_account": f.taskAccount, "api1_token": f.api1Token,
			})
		}
		return map[string]any{
			"task": first.task, "task_account": first.taskAccount, "api1_token": first.api1Token,
			"api1_username": first.api1Username, "extra_tasks": extras, "attempts": attempts, "last_msg": "ok",
		}
	}
	return map[string]any{"task": nil, "task_account": nil, "attempts": attempts, "last_msg": lastMsg}
}

func decodedExtTakeTaskFromPluginPool(pluginAccount map[string]any, limit int, takeType string, apiType int, randomize, serial bool) map[string]any {
	limit = max(1, limit)
	if apiType == 2 {
		apiType = 2
	} else if apiType == 1 {
		apiType = 1
	} else {
		apiType = 0
	}

	if apiType > 0 {
		ordered := decodedExtOrderedTaskAccountsByType(pluginAccount, apiType)
		if len(ordered) == 0 {
			return map[string]any{"task": nil, "task_account": nil, "attempts": []any{}, "last_msg": "no task"}
		}
		if randomize {
			shuffleMaps(ordered)
		}
		if limit > len(ordered) {
			limit = len(ordered)
		}
		return decodedExtTakeTaskFromAPI1Pool(pluginAccount, ordered[:limit], takeType, serial)
	}

	// apiType=0：接口1 优先，必要时回退接口2
	if decodedExtAPIFallbackAPI2Enabled(pluginAccount) {
		api1Ordered := decodedExtOrderedTaskAccountsByType(pluginAccount, 1)
		api1Result := map[string]any{"task": nil, "task_account": nil, "attempts": []any{}, "last_msg": "api1 account pool is empty"}
		if len(api1Ordered) > 0 {
			if randomize {
				shuffleMaps(api1Ordered)
			}
			if limit > len(api1Ordered) {
				limit = len(api1Ordered)
			}
			api1Result = decodedExtTakeTaskFromAPI1Pool(pluginAccount, api1Ordered[:limit], takeType, serial)
			if api1Result["task"] != nil {
				api1Result["api1_fallback_api2"] = true
				api1Result["fallback_stage"] = "api1"
				return api1Result
			}
		}
		api2Ordered := decodedExtOrderedTaskAccountsByType(pluginAccount, 2)
		if len(api2Ordered) == 0 {
			api1Result["api1_fallback_api2"] = true
			api1Result["fallback_stage"] = "api1_only"
			return api1Result
		}
		if randomize {
			shuffleMaps(api2Ordered)
		}
		if limit > len(api2Ordered) {
			limit = len(api2Ordered)
		}
		api2Result := decodedExtTakeTaskFromAPI1Pool(pluginAccount, api2Ordered[:limit], takeType, serial)
		// 合并 attempts
		if a1, ok := api1Result["attempts"].([]map[string]any); ok {
			if a2, ok2 := api2Result["attempts"].([]map[string]any); ok2 {
				api2Result["attempts"] = append(a1, a2...)
			} else {
				api2Result["attempts"] = a1
			}
		}
		api2Result["api1_fallback_api2"] = true
		if api2Result["task"] != nil {
			api2Result["fallback_stage"] = "api2"
		} else {
			api2Result["fallback_stage"] = "api2_no_task"
		}
		if api2Result["task"] == nil && trimStr(api2Result["last_msg"]) == "" {
			api2Result["last_msg"] = firstNonNilStr(api1Result["last_msg"], "no task")
		}
		return api2Result
	}

	ordered := decodedExtOrderedTaskAccountsByType(pluginAccount, 1)
	if len(ordered) == 0 {
		return map[string]any{"task": nil, "task_account": nil, "attempts": []any{}, "last_msg": "no task"}
	}
	if randomize {
		shuffleMaps(ordered)
	}
	if limit > len(ordered) {
		limit = len(ordered)
	}
	return decodedExtTakeTaskFromAPI1Pool(pluginAccount, ordered[:limit], takeType, serial)
}

func shuffleMaps(s []map[string]any) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// ---- 任务元信息 ----
func decodedExtTaskWithAccountMeta(task, taskAccount map[string]any) map[string]any {
	key := trimStr(taskAccount["key"])
	username := trimStr(taskAccount["username"])
	apiType := decodedExtAPITypeForTaskAccount(taskAccount)
	if key != "" {
		task["task_account_key"] = key
		task["taskAccountKey"] = key
	}
	if username != "" {
		task["task_account_username"] = username
		task["taskAccountUsername"] = username
	}
	task["task_api_type"] = apiType
	task["taskApiType"] = apiType
	task["api_type"] = apiType
	task["apiType"] = apiType
	return task
}

func decodedExtTaskWithTakeTokenMeta(task map[string]any, api1Token string) map[string]any {
	api1Token = strings.TrimSpace(api1Token)
	if api1Token != "" {
		task["api1_take_token"] = api1Token
		task["api1TakeToken"] = api1Token
		task["api1_take_token_hash"] = sha256short(api1Token)
	}
	return task
}

func decodedExtTaskAccountForSubmit(pluginAccount map[string]any, taskID, url string, apiType int) map[string]any {
	var taskAccounts []map[string]any
	if apiType > 0 {
		taskAccounts = decodedExtEnabledTaskAccountsByType(pluginAccount, apiType)
	} else {
		taskAccounts = decodedExtEnabledTaskAccounts(pluginAccount)
	}
	if len(taskAccounts) == 0 {
		return nil
	}
	lease := decodedExtFindTaskLease(pluginAccount, taskID, url)
	if lease != nil {
		if key := trimStr(lease["task_account_key"]); key != "" {
			for _, ta := range taskAccounts {
				if trimStr(ta["key"]) == key {
					return ta
				}
			}
		}
	}
	return taskAccounts[0]
}

func decodedExtSubmitResultDiagnostics(result string) map[string]any {
	var jsonValid bool
	var jsonErr string
	if len(result) > 0 {
		var v any
		if err := json.Unmarshal([]byte(result), &v); err == nil {
			jsonValid = true
		} else {
			jsonErr = err.Error()
		}
	}
	first, last := "", ""
	if len(result) > 0 {
		first = result[:1]
		last = result[len(result)-1:]
	}
	return map[string]any{
		"result_sha256":     sha256hex(result),
		"result_json_valid": jsonValid,
		"result_json_error": jsonErr,
		"result_first_char": first,
		"result_last_char":  last,
	}
}

var _ = time.Now
