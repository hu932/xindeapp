package main

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---- cx.php / cx1.php 查询引擎（内存缓存 + 磁盘 lastgood，替代 PHP session/文件锁）----

type cxConfig struct {
	File            string
	DataKey         string
	APITypeDefault  int
	Admin1Username  string
	Admin1Password  string
	Admin2Username  string
	Admin2Password  string
	LegacyUsername  string
	LegacyPassword  string
	SwitchDate      string
	TeamMode        bool
	TeamRemarks     []string
	Pricing1        map[string]float64
	Pricing2        map[string]float64
}

var cxConfigs = map[string]*cxConfig{
	"cx.php": {
		File: "cx.php", DataKey: "3bd0f92c13ed0c94",
		Admin1Username: "tpadsf532", Admin1Password: "a1qrga$JHFl52aP",
		Admin2Username: "tpadsf644", Admin2Password: "d78QEv6Y7K4Hl2af",
		LegacyUsername: "poisu906", LegacyPassword: "dfd$d7641HETD2af",
		SwitchDate: "2026-09-23",
		Pricing1: map[string]float64{"阳": 0.04, "胡逸": 0.04, "老妈": 0.048, "果子": 0.048, "传媒": 0.04, "杰": 0.05, "yee": 0.045, "C1": 0.05, "C2": 0.05, "C3": 0.05, "C4": 0.05, "C5": 0.05, "CN": 0.04, "C": 0.05},
		Pricing2: map[string]float64{"阳": 0.043, "传媒": 0.043, "自己": 0.048, "毛毛": 0.04, "C-C5": 0.04, "C1": 0.04, "C2": 0.04, "C3": 0.04, "C4": 0.04, "C5": 0.04, "C": 0.04},
	},
	// 接口2（zb2）专用查询页：固定 apiType=2，独立 DataKey 避免与 cx.php 的
	// cx_date_lastgood_<DataKey>_* 互相覆盖。
	"cx2.php": {
		File: "cx2.php", DataKey: "8f21ab04d7c93e15", APITypeDefault: 2,
		Admin1Username: "tpadsf532", Admin1Password: "a1qrga$JHFl52aP",
		Admin2Username: "tpadsf644", Admin2Password: "d78QEv6Y7K4Hl2af",
		SwitchDate: "",
		Pricing1: map[string]float64{},
		Pricing2: map[string]float64{"阳": 0.043, "传媒": 0.043, "自己": 0.048, "毛毛": 0.04, "C-C5": 0.04, "C1": 0.04, "C2": 0.04, "C3": 0.04, "C4": 0.04, "C5": 0.04, "C": 0.04},
	},
}

const (
	cxCacheTTL           = 120
	cxRetryBackoff       = 120
	cxMinRefresh         = 30
	cxRateLimitCap       = 300
	cxEmptyRetryInterval = 15
	cxSwitchDate         = "2026-09-23"
)

// 内存缓存
var (
	cxCacheMu    sync.Mutex
	cxMemory     = map[string]cxCacheEntry{}
	cxFetchLocks = map[string]*sync.Mutex{}
	cxCooldown   = map[string]cxCooldownEntry{}
)

type cxCacheEntry struct {
	payload   map[string]any
	fetchedAt time.Time
}

type cxCooldownEntry struct {
	until       time.Time
	reason      string
	attempts    int
	isRateLimit bool
}

func cxGetFetchLock(key string) *sync.Mutex {
	cxCacheMu.Lock()
	defer cxCacheMu.Unlock()
	if l, ok := cxFetchLocks[key]; ok {
		return l
	}
	l := &sync.Mutex{}
	cxFetchLocks[key] = l
	return l
}

func cxTokenCacheFile(username string) string {
	return cxCacheDirName + "/cx_api_token_cache_" + md5short(username) + ".json"
}

func cxCacheDir() string {
	return cxCacheDirName
}

func cxGetPricing(cfg *cxConfig, apiType int) map[string]float64 {
	if apiType == 2 {
		return cfg.Pricing2
	}
	return cfg.Pricing1
}

func cxPricePerSuccess(pricing map[string]float64, remark string) *float64 {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return nil
	}
	if v, ok := pricing[remark]; ok {
		return &v
	}
	// 长 key 优先
	keys := make([]string, 0, len(pricing))
	for k := range pricing {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len([]rune(keys[i])) > len([]rune(keys[j])) })
	for _, k := range keys {
		if strings.Contains(remark, k) {
			v := pricing[k]
			return &v
		}
	}
	return nil
}

func cxReadJSONFile(file string) map[string]any {
	var m map[string]any
	if readJSONFile(dp(file), &m) {
		return m
	}
	return nil
}

func cxAtomicWrite(file string, v any) {
	_ = atomicWrite(dp(file), jsonEncode(v))
}

// cxCacheKey：与 PHP 完全一致（PHP 对 ksort 后的关联数组 json_encode 得到对象）
func cxCacheKey(dataKey, date string, authorizedAccounts map[string]string) string {
	accJSON, _ := json.Marshal(authorizedAccounts)
	return "cx_v2_" + dataKey + "_" + date + "_" + md5hex(string(accJSON))
}

func cxRowsHaveData(rows []map[string]any) bool {
	for _, row := range rows {
		if cxActivityTotal(row) > 0 {
			return true
		}
	}
	return false
}

func cxActivityTotal(row map[string]any) int {
	return qInt(row["takeCount"]) + qInt(row["completeCount"]) + qInt(row["checkSuccessCount"]) + qInt(row["checkFailCount"])
}

// cxDedupeBatchRows：与 PHP fetchAuthorizedSettleRows 一致，按 username 去重（后出现的覆盖先出现的，顺序保持首次出现）
func cxDedupeBatchRows(rows []map[string]any) []map[string]any {
	if len(rows) == 0 {
		return rows
	}
	byUser := map[string]map[string]any{}
	var order []string
	for _, r := range rows {
		u := trimStr(r["username"])
		if u == "" {
			continue
		}
		if _, ok := byUser[u]; !ok {
			order = append(order, u)
		}
		byUser[u] = r
	}
	out := make([]map[string]any, 0, len(order))
	for _, u := range order {
		out = append(out, byUser[u])
	}
	return out
}

func cxSectionTotal(payload map[string]any, key string) int {
	rows := rowsOf(payload, key)
	total := 0
	for _, r := range rows {
		total += cxActivityTotal(r)
	}
	return total
}

func cxPayloadHasData(payload map[string]any) bool {
	return cxSectionTotal(payload, "items") > 0 || cxSectionTotal(payload, "batchItems") > 0
}

func cxPayloadSectionHasData(payload map[string]any, key string) bool {
	return cxRowsHaveData(rowsOf(payload, key))
}

func cxSameUsernameSet(a, b []map[string]any) bool {
	setA, setB := map[string]bool{}, map[string]bool{}
	for _, r := range a {
		if n := trimStr(r["username"]); n != "" {
			setA[n] = true
		}
	}
	for _, r := range b {
		if n := trimStr(r["username"]); n != "" {
			setB[n] = true
		}
	}
	if len(setA) == 0 {
		return false
	}
	if len(setA) != len(setB) {
		return false
	}
	for k := range setA {
		if !setB[k] {
			return false
		}
	}
	return true
}

func cxMergeMonotonic(fresh, best map[string]any) map[string]any {
	if best != nil {
		if cxSectionTotal(fresh, "items") < cxSectionTotal(best, "items") {
			fresh["items"] = best["items"]
		}
		freshBatch := rowsOf(fresh, "batchItems")
		bestBatch := rowsOf(best, "batchItems")
		if cxSameUsernameSet(freshBatch, bestBatch) && cxSectionTotal(fresh, "batchItems") < cxSectionTotal(best, "batchItems") {
			fresh["batchItems"] = bestBatch
		}
	}
	return fresh
}

// cx lastgood 文件（与 PHP 同名）
func cxLastGoodFile(key string) string {
	return cxCacheDirName + "/cx_lastgood_" + md5hex(key) + ".json"
}

func cxDateLastGoodFile(dataKey, date string) string {
	return cxCacheDirName + "/cx_date_lastgood_" + dataKey + "_" + md5hex(date) + ".json"
}

func cxCacheFile(key string) string {
	return cxCacheDirName + "/cx_cache_" + md5hex(key) + ".json"
}

func cxCooldownFile(key string) string {
	return cxCacheDirName + "/cx_cooldown_" + md5hex(key) + ".json"
}

func cxReadPayloadFile(file string) map[string]any {
	var wrapper map[string]any
	if !readJSONFile(dp(file), &wrapper) {
		return nil
	}
	if p, ok := wrapper["payload"].(map[string]any); ok {
		return p
	}
	// 兼容旧格式：文件本身就是 payload
	return wrapper
}

func cxWritePayloadFile(file string, payload map[string]any) {
	cxAtomicWrite(file, map[string]any{"_ts": timeNowUnix(), "payload": payload})
}

// 冷却
func cxCooldownRemaining(key string) int {
	cxCacheMu.Lock()
	defer cxCacheMu.Unlock()
	if c, ok := cxCooldown[key]; ok && time.Now().Before(c.until) {
		return int(time.Until(c.until).Seconds()) + 1
	}
	// 从磁盘恢复
	var d map[string]any
	if readJSONFile(dp(cxCooldownFile(key)), &d) {
		until := int64(qInt(d["until"]))
		if until > timeNowUnix() {
			cxCooldown[key] = cxCooldownEntry{until: time.Unix(until, 0), reason: qStr(d["reason"]), attempts: qInt(d["attempts"]), isRateLimit: asBool(d["is_rate_limit"], false)}
			return int(until - timeNowUnix())
		}
	}
	return 0
}

func cxSetCooldown(key string, seconds int, reason string, isRateLimit bool) int {
	entry := cxCooldownEntry{until: time.Now().Add(time.Duration(seconds) * time.Second), reason: reason, attempts: 1, isRateLimit: isRateLimit}
	cxCacheMu.Lock()
	cxCooldown[key] = entry
	cxCacheMu.Unlock()
	cxAtomicWrite(cxCooldownFile(key), map[string]any{"until": entry.until.Unix(), "reason": reason, "attempts": 1, "is_rate_limit": isRateLimit})
	return seconds
}

func cxClearCooldown(key string) {
	cxCacheMu.Lock()
	delete(cxCooldown, key)
	cxCacheMu.Unlock()
	_ = os.Remove(dp(cxCooldownFile(key)))
}

// ---- 授权账号 ----
func cxLoadAuthorizedAccounts(cfg *cxConfig, apiType int, date string, reqCtx *httpRequestCtx) map[string]string {
	accountsFile := api1AccountsFile
	if apiType == 2 {
		accountsFile = api2AccountsFile
	}
	username := cfg.Admin1Username
	if apiType == 2 {
		username = cfg.Admin2Username
	}
	if apiType == 1 && date < cfg.SwitchDate {
		username = cfg.LegacyUsername
		accountsFile = "api1_allowed_accounts_legacy.json"
	}
	result := map[string]string{}
	for _, acc := range readAllowedAccountsForFile(accountsFile) {
		result[trimStr(acc["username"])] = trimStr(acc["remark"])
	}
	if len(result) == 0 {
		// 从 lastgood 恢复
		lastGoodFile := cxCacheDirName + "/cx_authorized_accounts_lastgood_" + md5short(username) + ".json"
		var d map[string]any
		if readJSONFile(dp(lastGoodFile), &d) {
			if accounts, ok := d["accounts"].(map[string]any); ok {
				for u, r := range accounts {
					if rm, ok := r.(map[string]any); ok {
						result[u] = trimStr(firstNonNil(rm["remark"], rm["nickname"], rm["label"]))
					} else {
						result[u] = qStr(r)
					}
				}
			}
		}
	}
	if len(result) > 0 {
		cxAtomicWrite(cxCacheDirName+"/cx_authorized_accounts_lastgood_"+md5short(username)+".json", map[string]any{"_ts": timeNowUnix(), "accounts": result})
	}
	// 团队C 过滤
	if cfg.TeamMode && apiType == 1 {
		filtered := map[string]string{}
		for u, r := range result {
			if containsStr(cfg.TeamRemarks, strings.TrimSpace(r)) {
				filtered[u] = r
			}
		}
		result = filtered
	}
	return result
}

// ---- 上游查询 ----
func cxGetToken(cfg *cxConfig, apiType int, date string) string {
	username := cfg.Admin1Username
	password := cfg.Admin1Password
	base := api1Base
	if apiType == 2 {
		username = cfg.Admin2Username
		password = cfg.Admin2Password
		base = api2Base
	}
	if apiType == 1 && date < cfg.SwitchDate {
		username = cfg.LegacyUsername
		password = cfg.LegacyPassword
	}
	tokenFile := cxTokenCacheFile(username)
	// 读缓存
	var cached map[string]any
	if readJSONFile(dp(tokenFile), &cached) {
		if tok := trimStr(cached["token"]); tok != "" {
			if ts := int64(qInt(cached["ts"])); timeNowUnix()-ts < 3500 {
				return tok
			}
		}
	}
	res := forwardToServer(base+"/user/login", jsonEncode(map[string]string{"username": username, "password": password}), false, nil)
	var j map[string]any
	token := ""
	if jsonDecode(res.Response, &j) {
		if trimStr(j["code"]) == "200" || qInt(j["code"]) == 200 {
			if data, ok := j["data"].(map[string]any); ok {
				token = trimStr(data["token"])
				if token == "" {
					token = trimStr(data["accessToken"])
				}
			}
			if token == "" {
				token = trimStr(j["token"])
			}
		}
	}
	if token != "" {
		cxAtomicWrite(tokenFile, map[string]any{"token": token, "ts": timeNowUnix()})
		return token
	}
	return firstNonNilStr(trimStr(j["msg"]), "登录失败")
}

func cxFetchSettleView(base, token string, userList []string, startTime, endTime string) (bool, string, []map[string]any, map[string]any) {
	if userList == nil {
		userList = []string{}
	}
	authHeaders := []string{"Authorization: Bearer " + token, "token: " + token}
	body := map[string]any{"pageNum": 1, "pageSize": max(100, len(userList)+20), "userList": userList, "startTime": startTime, "endTime": endTime}
	res := forwardToServer(base+"/user/settle/view", jsonEncode(body), false, authHeaders)
	if res.HTTPCode == 429 {
		return false, "rate limit (HTTP 429)", nil, nil
	}
	var j map[string]any
	if res.Error != "" {
		return false, "查询接口无响应", nil, nil
	}
	if !jsonDecode(res.Response, &j) {
		return false, "查询接口无响应", nil, nil
	}
	if trimStr(j["code"]) == "200" || qInt(j["code"]) == 200 {
		var items []map[string]any
		if data, ok := j["data"].(map[string]any); ok {
			if arr, ok := data["items"].([]any); ok {
				for _, it := range arr {
					if m, ok := it.(map[string]any); ok {
						items = append(items, m)
					}
				}
			}
		}
		return true, "", items, j
	}
	return false, firstNonNilStr(j["msg"], "查询失败"), nil, j
}

// CxQuery 执行一次查询并返回与 PHP 一致的 payload
func CxQuery(cfgFile string, apiType int, date string, autoRefresh, forceRefresh bool, reqCtx *httpRequestCtx) map[string]any {
	cfg := cxConfigs[cfgFile]
	if cfg == nil {
		cfg = cxConfigs["cx.php"]
	}
	if date == "" || !regexpDate.MatchString(date) {
		date = tsDate()
	}
	if apiType != 2 {
		if cfg.APITypeDefault == 2 {
			apiType = 2
		} else {
			apiType = 1
		}
	}
	base := api1Base
	if apiType == 2 {
		base = api2Base
	}
	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	authorizedAccounts := cxLoadAuthorizedAccounts(cfg, apiType, date, reqCtx)
	cacheKey := cxCacheKey(cfg.DataKey, date, authorizedAccounts)

	// 冷却检查
	cooldownLeft := cxCooldownRemaining(cacheKey)
	if forceRefresh && cooldownLeft > 0 {
		forceRefresh = false
	}

	// 内存缓存
	cxCacheMu.Lock()
	mem, memOK := cxMemory[cacheKey]
	cxCacheMu.Unlock()
	cached := map[string]any(nil)
	if !forceRefresh {
		if memOK {
			if autoRefresh {
				if time.Since(mem.fetchedAt).Seconds() < cxCacheTTL {
					cached = deepCopyMap(mem.payload)
				}
			} else {
				cached = deepCopyMap(mem.payload)
			}
		}
		if cached == nil {
			// 磁盘缓存
			cached = cxReadPayloadFile(cxCacheFile(cacheKey))
			if cached != nil && !cxPayloadHasData(cached) {
				cached = nil
			}
		}
	}

	fromCache := cached != nil
	usingLastGood := false
	cacheAge := 0
	warning := ""
	errorMsg := ""
	items := []map[string]any{}
	batchItems := []map[string]any{}
	if cached != nil {
		items = rowsOf(cached, "items")
		batchItems = rowsOf(cached, "batchItems")
		cacheAge = int(timeNowUnix() - int64(qInt(cached["fetchTime"])))
	}

	shouldFetch := (forceRefresh || cached == nil || (autoRefresh && cacheAge >= cxCacheTTL)) && cooldownLeft <= 0
	if shouldFetch {
		lock := cxGetFetchLock(cacheKey)
		if lock.TryLock() {
			defer lock.Unlock()
			// 再次检查（double-check）
			cxCacheMu.Lock()
			mem2, ok2 := cxMemory[cacheKey]
			cxCacheMu.Unlock()
			if !forceRefresh && ok2 {
				cached = deepCopyMap(mem2.payload)
				items = rowsOf(cached, "items")
				batchItems = rowsOf(cached, "batchItems")
				cacheAge = int(timeNowUnix() - int64(qInt(cached["fetchTime"])))
				fromCache = true
				shouldFetch = false
			}
		} else {
			// 另一个刷新进行中，重读缓存
			cxCacheMu.Lock()
			mem3, ok3 := cxMemory[cacheKey]
			cxCacheMu.Unlock()
			if ok3 {
				cached = deepCopyMap(mem3.payload)
				items = rowsOf(cached, "items")
				batchItems = rowsOf(cached, "batchItems")
				cacheAge = int(timeNowUnix() - int64(qInt(cached["fetchTime"])))
				fromCache = true
			}
			if warning == "" {
				warning = "另一个刷新任务正在进行中，本次使用已有缓存数据。"
			}
			shouldFetch = false
		}
	}

	lastGood := cxReadPayloadFile(cxLastGoodFile(cacheKey))
	dateLastGood := cxReadPayloadFile(cxDateLastGoodFile(cfg.DataKey, date))

	if shouldFetch {
		cxSetCooldown(cacheKey, cxRetryBackoff, "正在刷新", false)
		token := cxGetToken(cfg, apiType, date)
		fetchError := ""
		if len(token) < 20 {
			fetchError = token
		} else {
			ok, msg, viewItems, viewRaw := cxFetchSettleView(base, token, nil, startTime, endTime)
			if ok {
				items = viewItems
			} else {
				code := trimStr(viewRaw["code"])
				if code == "401" {
					fetchError = "登录已过期，请等待自动更新或稍后再试"
				} else {
					fetchError = msg
				}
			}
			if fetchError == "" && len(authorizedAccounts) > 0 {
				userList := make([]string, 0, len(authorizedAccounts))
				for u := range authorizedAccounts {
					userList = append(userList, u)
				}
				sort.Strings(userList)
				ok2, msg2, authRows, _ := cxFetchSettleView(base, token, userList, startTime, endTime)
				if ok2 {
					batchItems = []map[string]any{}
					for _, r := range authRows {
						u := trimStr(r["username"])
						if u == "" || authorizedAccounts[u] == "" {
							continue
						}
						if cxActivityTotal(r) <= 0 {
							continue
						}
						r["_remark"] = authorizedAccounts[u]
						batchItems = append(batchItems, r)
					}
					batchItems = cxDedupeBatchRows(batchItems)
				} else {
					fetchError = msg2
				}
			}
		}

		if fetchError == "" {
			if cfg.TeamMode {
				// 团队C：白名单明细汇总成 items
				var teamTake, teamComplete, teamSuccess, teamFail int
				for _, r := range batchItems {
					teamTake += qInt(r["takeCount"])
					teamComplete += qInt(r["completeCount"])
					teamSuccess += qInt(r["checkSuccessCount"])
					teamFail += qInt(r["checkFailCount"])
				}
				items = []map[string]any{{
					"username": "团队C-所有", "takeCount": teamTake, "completeCount": teamComplete,
					"checkSuccessCount": teamSuccess, "checkFailCount": teamFail,
					"completeRate": rateText(teamComplete, max(1, teamTake)), "checkSuccessRate": rateText(teamSuccess, max(1, teamComplete)),
				}}
			}
			freshPayload := map[string]any{
				"items": items, "batchItems": batchItems, "fetchTime": timeNowUnix(),
				"date": date, "apiType": apiType, "api_type": apiType,
			}
			hasData := cxPayloadHasData(freshPayload)
			bestSnapshot := lastGood
			if bestSnapshot == nil {
				bestSnapshot = dateLastGood
			}
			if !hasData {
				fallback := lastGood
				if fallback == nil {
					fallback = dateLastGood
				}
				if fallback != nil && cxPayloadHasData(fallback) {
					items = rowsOf(fallback, "items")
					batchItems = rowsOf(fallback, "batchItems")
					fromCache = true
					usingLastGood = true
					cacheAge = int(timeNowUnix() - int64(qInt(fallback["fetchTime"])))
					warning = "上游本次返回空数据，已继续使用最近一次有数据的缓存（" + strconvItoa(cacheAge) + " 秒前）。"
					cxWritePayloadFile(cxCacheFile(cacheKey), freshPayload)
				cxSetCooldown(cacheKey, cxEmptyRetryInterval, "上游返回空数据", false)
				cooldownLeft = cxEmptyRetryInterval
			} else {
				errorMsg = "上游本次返回空数据，未使用本次结果；暂无可用历史缓存，请稍后刷新。"
				fromCache = false
				cxSetCooldown(cacheKey, cxEmptyRetryInterval, "上游返回空数据且无可用历史缓存", false)
				cooldownLeft = cxEmptyRetryInterval
				}
			} else {
				// 查到数据就立即写入最新缓存（单调合并），不再因“部分返回”回滚到旧数据
				persist := cxMergeMonotonic(freshPayload, bestSnapshot)
				items = rowsOf(persist, "items")
				batchItems = rowsOf(persist, "batchItems")
				persist["date"] = date
				persist["apiType"] = apiType
				persist["api_type"] = apiType
				persist["authorizedAccounts"] = mapToAnyMap(authorizedAccounts)
				cxWritePayloadFile(cxCacheFile(cacheKey), persist)
				cxWritePayloadFile(cxLastGoodFile(cacheKey), persist)
				cxWritePayloadFile(cxDateLastGoodFile(cfg.DataKey, date), persist)
				fromCache = false
				cacheAge = 0
				cxClearCooldown(cacheKey)
				cooldownLeft = 0
			}
		} else {
			cooldownLeft = cxSetCooldown(cacheKey, cxEmptyRetryInterval, fetchError, false)
			if !fromCache {
				fallback := lastGood
				if fallback == nil {
					fallback = dateLastGood
				}
				if fallback != nil && cxPayloadHasData(fallback) {
					items = rowsOf(fallback, "items")
					batchItems = rowsOf(fallback, "batchItems")
					cacheAge = int(timeNowUnix() - int64(qInt(fallback["fetchTime"])))
					fromCache = true
					usingLastGood = true
					warning = "上游接口暂时不可用：" + fetchError + "，已使用最近一次有数据的缓存（" + strconvItoa(cacheAge) + " 秒前）。"
				} else {
					errorMsg = fetchError
				}
			} else {
				warning = "上游接口暂时不可用：" + fetchError + "，" + strconvItoa(cxEmptyRetryInterval) + " 秒内将继续使用缓存数据。"
			}
		}
	}

	if cooldownLeft > 0 && warning == "" && errorMsg == "" && fromCache {
		warning = "上游接口冷却中，" + strconvItoa(cooldownLeft) + " 秒后再尝试刷新。"
	}

	// 汇总计算
	payload := cxComputePayload(cfg, apiType, date, items, batchItems, authorizedAccounts, fromCache, cacheAge, usingLastGood, warning, errorMsg, cooldownLeft)

	// 写内存缓存
	finalPayload := deepCopyMap(payload)
	cxCacheMu.Lock()
	cxMemory[cacheKey] = cxCacheEntry{payload: finalPayload, fetchedAt: time.Now()}
	cxCacheMu.Unlock()

	return payload
}

func mapToAnyMap(m map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func deepCopyMap(m map[string]any) map[string]any {
	return deepCopyValue(m).(map[string]any)
}

var regexpDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func cxComputePayload(cfg *cxConfig, apiType int, date string, items, batchItems []map[string]any, authorizedAccounts map[string]string, fromCache bool, cacheAge int, usingLastGood bool, warning, errorMsg string, cooldownLeft int) map[string]any {
	batchItems = cxDedupeBatchRows(batchItems)
	totalTake, totalComplete, totalSuccess, totalFail := 0, 0, 0, 0
	for _, r := range items {
		totalTake += qInt(r["takeCount"])
		totalComplete += qInt(r["completeCount"])
		totalSuccess += qInt(r["checkSuccessCount"])
		totalFail += qInt(r["checkFailCount"])
	}

	// 活跃白名单
	activeBatch := []map[string]any{}
	activeNames := map[string]bool{}
	for _, r := range batchItems {
		if trimStr(r["_src"]) != "old" && cxActivityTotal(r) > 0 {
			activeBatch = append(activeBatch, r)
		}
		if trimStr(r["_src"]) != "old" {
			activeNames[trimStr(r["username"])] = true
		}
	}
	inactiveAuthorized := []map[string]any{}
	for u, remark := range authorizedAccounts {
		if !activeNames[u] {
			inactiveAuthorized = append(inactiveAuthorized, map[string]any{"username": u, "_remark": remark})
		}
	}

	pricing := cxGetPricing(cfg, apiType)
	byRemarkMap := map[string]map[string]any{}
	for _, r := range batchItems {
		remark := trimStr(r["_remark"])
		if remark == "" {
			remark = "未备注"
		}
		info := byRemarkMap[remark]
		if info == nil {
			info = map[string]any{"remark": remark, "take": 0, "complete": 0, "success": 0, "fail": 0, "accounts": 0}
			byRemarkMap[remark] = info
		}
		info["take"] = qInt(info["take"]) + qInt(r["takeCount"])
		info["complete"] = qInt(info["complete"]) + qInt(r["completeCount"])
		info["success"] = qInt(info["success"]) + qInt(r["checkSuccessCount"])
		info["fail"] = qInt(info["fail"]) + qInt(r["checkFailCount"])
		info["accounts"] = qInt(info["accounts"]) + 1
	}
	totalAuthAmount := 0.0
	selfAmount := 0.0
	selfPrice := cxPricePerSuccess(pricing, "自己")
	for remark, info := range byRemarkMap {
		if p := cxPricePerSuccess(pricing, remark); p != nil {
			info["unitPrice"] = *p
			info["totalAmount"] = round2(*p * float64(qInt(info["success"])))
			totalAuthAmount += info["totalAmount"].(float64)
			if remark == "自己" {
				selfAmount = info["totalAmount"].(float64)
			}
		} else {
			info["unitPrice"] = nil
			info["totalAmount"] = nil
		}
	}
	byRemark := make([]map[string]any, 0, len(byRemarkMap))
	for _, info := range byRemarkMap {
		byRemark = append(byRemark, info)
	}
	sort.Slice(byRemark, func(i, j int) bool { return qInt(byRemark[i]["success"]) > qInt(byRemark[j]["success"]) })
	sort.Slice(batchItems, func(i, j int) bool { return qInt(batchItems[i]["checkSuccessCount"]) > qInt(batchItems[j]["checkSuccessCount"]) })

	authTake, authComplete, authSuccess, authFail := 0, 0, 0, 0
	for _, r := range batchItems {
		authTake += qInt(r["takeCount"])
		authComplete += qInt(r["completeCount"])
		authSuccess += qInt(r["checkSuccessCount"])
		authFail += qInt(r["checkFailCount"])
	}
	// 顶部总数与「备注分组绩效」同口径：有授权账号时用 batchItems 汇总，否则用全局 items
	if len(batchItems) > 0 {
		totalTake, totalComplete, totalSuccess, totalFail = authTake, authComplete, authSuccess, authFail
	}
	totalProfit := 0.0
	if selfPrice != nil {
		totalProfit = round2(*selfPrice*float64(authSuccess) - (totalAuthAmount - selfAmount))
	}

	topRemark := "暂无"
	topRemarkSuccess := 0
	if len(byRemark) > 0 {
		topRemark = trimStr(byRemark[0]["remark"])
		topRemarkSuccess = qInt(byRemark[0]["success"])
	}
	topAccount := "暂无"
	topAccountRemark := "未备注"
	if len(batchItems) > 0 {
		topAccount = trimStr(batchItems[0]["username"])
		topAccountRemark = trimStr(batchItems[0]["_remark"])
		if topAccountRemark == "" {
			topAccountRemark = "未备注"
		}
	}

	payload := map[string]any{
		"ok": true, "date": date, "apiType": apiType, "api_type": apiType,
		"items": items, "batchItems": batchItems, "authorizedAccounts": mapToAnyMap(authorizedAccounts),
		"activeBatch": activeBatch, "inactiveAuthorized": inactiveAuthorized,
		"totalTake": totalTake, "totalComplete": totalComplete, "totalSuccess": totalSuccess, "totalFail": totalFail,
		"successRate": rateText(totalSuccess, max(1, totalComplete)), "completeRate": rateText(totalComplete, max(1, totalTake)),
		"authTake": authTake, "authComplete": authComplete, "authSuccess": authSuccess, "authFail": authFail,
		"authCompleteRate": rateText(authComplete, max(1, authTake)), "authSuccessRate": rateText(authSuccess, max(1, authComplete)),
		"byRemark": byRemark, "topRemark": topRemark, "topRemarkSuccess": topRemarkSuccess,
		"topAccount": topAccount, "topAccountRemark": topAccountRemark, "activeAuthorizedCount": len(activeBatch),
		"totalAuthAmount": round2(totalAuthAmount), "selfAmount": round2(selfAmount), "totalProfit": totalProfit,
		"fromCache": fromCache, "cacheAge": cacheAge, "usingLastGood": usingLastGood,
		"warning": warning, "error": errorMsg, "cooldownLeft": cooldownLeft,
		"fetchTime": timeNowUnix(), "generatedAt": tsDateTime(),
		"autoRefreshDelay": max(cxMinRefresh, cooldownLeft),
	}
	return payload
}

// cxClearCache：清缓存
func cxClearCache() map[string]any {
	cxCacheMu.Lock()
	cxMemory = map[string]cxCacheEntry{}
	cxCooldown = map[string]cxCooldownEntry{}
	cxFetchLocks = map[string]*sync.Mutex{}
	cxCacheMu.Unlock()
	deleted := 0
	patterns := []string{"cx_cache_*.json", "cx_lastgood_*.json", "cx_date_lastgood_*.json", "cx_switchold_*.json", "cx_cooldown_*.json", "cx_api_token_cache*.json", "cx_authorized_accounts_lastgood*.json"}
	for _, p := range patterns {
		matches, _ := filepathGlob(cxCacheDirName + "/" + p)
		for _, f := range matches {
			if removeFile(f) == nil {
				deleted++
			}
		}
	}
	return map[string]any{"ok": true, "deleted": deleted}
}

// startCxAutoRefresh 启动 cx.php 查询自动刷新循环：
// 每 2 分钟强制刷新一次；若本次没查到新数据（空数据/失败/限流），则按冷却时间快速重试，直到查到数据为止，再回到 2 分钟循环。
func startCxAutoRefresh() {
	go func() {
		const normalWait = 120 * time.Second
		wait := 15 * time.Second // 启动后先尽快刷一次
		for {
			time.Sleep(wait)
			p := CxQuery("cx.php", 1, tsDate(), true, true, &httpRequestCtx{})
			cd := qInt(p["cooldownLeft"])
			if cd <= 0 {
				wait = normalWait // 查到数据：2 分钟后再查
			} else {
				// 没查到数据：按冷却时间等待后重试（空数据 15s 快速重试；限流/失败 180~300s）
				wait = time.Duration(max(cxEmptyRetryInterval, cd)) * time.Second
			}
		}
	}()
}
