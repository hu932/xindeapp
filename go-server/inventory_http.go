package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// ---- 新入口：库存取任务 ----
// GET /inventory/task
func handleInventoryTask(w http.ResponseWriter, r *http.Request) {
	cfg := invLoadConfig()
	if !cfg.Enabled {
		writeJSONOK(w, map[string]any{"code": 404, "success": false, "msg": "库存模式未开启"})
		return
	}
	accountID := trimStr(firstNonNil(r.URL.Query().Get("account_id"), r.URL.Query().Get("phone"), r.URL.Query().Get("下发id")))
	if accountID == "" {
		accountID = "unknown"
	}
	timeout := time.Duration(max(30, cfg.TimeoutSeconds)) * time.Second
	task := invMgr.popTask(accountID, timeout)
	if task == nil {
		writeJSONOK(w, map[string]any{"code": 404, "success": false, "msg": "暂无可用任务"})
		return
	}
	writeJSONOK(w, map[string]any{"code": 200, "success": true, "task": task.Task})
}

// ---- 新入口：回传（解密后异步提交上游） ----
// POST /inventory/submit
func handleInventorySubmit(w http.ResponseWriter, r *http.Request) {
	reqCtx := newHTTPRequestCtx(r)
	cfg := invLoadConfig()
	if !cfg.Enabled {
		writeJSONOK(w, map[string]any{"code": 404, "success": false, "msg": "库存模式未开启"})
		return
	}
	input := readRawJSONBody(r)
	accountID := trimStr(firstNonNil(input["account_id"], input["phone"], input["下发id"]))
	traceID := trimStr(firstNonNil(input["trace_id"], input["traceId"], input["task_id"]))
	if traceID == "" {
		appendLog(logFile, map[string]any{
			"ip": reqCtx.ip, "source": "inventory_submit", "status": "missing_trace_id",
			"username": accountID, "account_id": accountID, "msg": "缺少 trace_id",
		}, maxLogs)
		writeJSONOK(w, map[string]any{"code": 400, "success": false, "msg": "缺少 trace_id"})
		return
	}
	if accountID == "" {
		accountID = "unknown"
	}
	timeout := time.Duration(max(30, cfg.TimeoutSeconds)) * time.Second
	task, tookAccountID := invMgr.takeOutstanding(accountID, traceID, timeout)
	if task == nil {
		appendLog(logFile, map[string]any{
			"ip": reqCtx.ip, "source": "inventory_submit", "status": "task_missing",
			"username": accountID, "account_id": accountID, "trace_id": traceID, "msg": "任务已超时或不存在",
		}, maxLogs)
		writeJSONOK(w, map[string]any{"code": 409, "success": false, "msg": "任务已超时或不存在，请重新获取"})
		return
	}
	// 计数归属：优先用领取任务时记录的下发id（提交请求没带 account_id 时也能正确归属）
	if trimStr(accountID) == "" || trimStr(accountID) == "unknown" {
		if tookAccountID != "" {
			accountID = tookAccountID
		}
	}
	if accountID == "" {
		accountID = "unknown"
	}

	// 解密 content
	content := anyToStr(input["content"])
	if content == "" {
		if d, ok := input["data"]; ok {
			switch v := d.(type) {
			case map[string]any:
				content = jsonEncode(v)
			case string:
				content = v
			default:
				content = anyToStr(v)
			}
		}
	}
	// 与普通提交对齐：明文 JSON 提取商品数据 / AES 解密后再提取（body→dumps→response_body 等包装）
	if strings.TrimSpace(content) != "" {
		if decrypted, okDec := decryptMarsSubmitData(content); okDec {
			content = decrypted
		}
	}
	if content == "" {
		appendLog(logFile, map[string]any{
			"ip": reqCtx.ip, "source": "inventory_submit", "status": "empty_content",
			"username": accountID, "account_id": accountID, "trace_id": traceID, "msg": "回传内容为空",
		}, maxLogs)
		writeJSONOK(w, map[string]any{"code": 400, "success": false, "msg": "回传内容为空"})
		return
	}

	// 与普通提交对齐：url 优先带 item_id=，必要时从 result 重建 get_pc 链接；再校验 item_id 一致
	submitURL := refineSubmitURL(task.TaskURL, input, content)
	if urlItemID := submitItemIDFromURL(submitURL); urlItemID != "" {
		if resultItemID := submitItemIDFromResult(content); resultItemID != "" && urlItemID != resultItemID {
			appendLog(logFile, map[string]any{
				"ip": reqCtx.ip, "source": "inventory_submit", "status": "item_id_mismatch",
				"username": accountID, "account_id": accountID, "trace_id": traceID,
				"task_id": task.TaskID, "sub_username": task.SubUsername,
				"url_item_id": urlItemID, "result_item_id": resultItemID,
			}, maxLogs)
			invRecordFail(accountID)
			writeJSONOK(w, map[string]any{"code": 400, "success": false, "msg": "item_id 不匹配"})
			return
		}
	}
	// 解密后按接口1回传格式组装：{appVersion, url, result}
	forwardBody := jsonEncode(map[string]any{
		"appVersion": firstNonNilStr(input["appVersion"], "vv2"),
		"url":        submitURL,
		"result":     content,
	})

	// 记录已接收（accepted）
	appendLog(logFile, map[string]any{
		"ip": reqCtx.ip, "source": "inventory_submit", "status": "accepted",
		"username": accountID, "account_id": accountID, "trace_id": traceID,
		"task_id": task.TaskID, "sub_username": task.SubUsername, "task_url": task.TaskURL,
	}, maxLogs)

	// 异步提交接口1
	go func() {
		res := forwardToServer(api1SubmitURL, forwardBody, submitGzip, []string{"Authorization: Bearer " + task.Token})
		var serverResp map[string]any
		_ = jsonDecode(res.Response, &serverResp)
		responseStr := compactLogResponseStr(res.Response, maxLogRespBytes)
		if res.Error == "" && api1ForwardSuccess(serverResp) {
			invRecordSuccess(accountID)
			appendLog(logFile, map[string]any{
				"ip": reqCtx.ip, "source": "inventory_submit", "status": "success",
				"username": accountID, "account_id": accountID, "trace_id": traceID,
				"task_id": task.TaskID, "sub_username": task.SubUsername, "task_url": task.TaskURL,
				"http_code": res.HTTPCode, "response": responseStr,
			}, maxLogs)
		} else {
			invRecordFail(accountID)
			appendLog(logFile, map[string]any{
				"ip": reqCtx.ip, "source": "inventory_submit", "status": "fail",
				"username": accountID, "account_id": accountID, "trace_id": traceID,
				"task_id": task.TaskID, "sub_username": task.SubUsername, "task_url": task.TaskURL,
				"error": res.Error, "http_code": res.HTTPCode,
				"upstream_msg": api1SubmitFailureMessage(serverResp, res.Error),
				"forward_response_len": len(res.Response), "response": responseStr,
			}, maxLogs)
		}
	}()

	writeJSONOK(w, map[string]any{"code": 200, "success": true, "accepted": true, "trace_id": traceID})
}

// inventorySubmitLogs 返回库存模式回传记录（提交上游成功/失败结果），从主日志过滤 source=inventory_submit
func inventorySubmitLogs(limit int) map[string]any {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	logs := readLogs(logFile, 5000)
	var rows []map[string]any
	for _, l := range logs {
		if trimStr(l["source"]) == "inventory_submit" {
			rows = append(rows, l)
		}
	}
	if len(rows) > limit {
		rows = rows[len(rows)-limit:]
	}
	// 新的在上
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return map[string]any{"ok": true, "data": rows}
}

func decryptInventoryContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 明文 JSON 直接返回
	var v any
	if jsonUnmarshalOK(s, &v) {
		return s
	}
	// AES 解密
	if plain, ok := aesDecryptString(s, aesKey); ok {
		return plain
	}
	return s
}

func jsonUnmarshalOK(s string, out any) bool {
	return jsonDecode(s, out)
}

// ---- 管理接口（后台） ----
func handleInventoryConfigGet(w http.ResponseWriter) {
	cfg := invLoadConfig()
	writeJSONOK(w, map[string]any{"ok": true, "data": cfg})
}

// ---- 库存模式各下发id单价 ----
func handleInventoryPricesGet(w http.ResponseWriter) {
	writeJSONOK(w, map[string]any{"ok": true, "prices": invLoadPrices()})
}

func handleInventoryPricesSave(w http.ResponseWriter, r *http.Request) {
	input := readRawJSONBody(r)
	var prices map[string]float64
	if raw, ok := input["prices"]; ok {
		b, _ := json.Marshal(raw)
		if json.Unmarshal(b, &prices) == nil {
			if invSavePrices(prices) {
				writeJSONOK(w, map[string]any{"ok": true, "msg": "单价已保存", "prices": prices})
				return
			}
		}
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "单价格式错误"})
}

func handleInventoryStatusGet(w http.ResponseWriter, r *http.Request) {
	cfg := invLoadConfig()
	date := firstNonNilStr(r.URL.Query().Get("date"), tsDate())
	stats := invStatsFor(date)
	invMgr.mu.Lock()
	q := len(invMgr.queue)
	o := len(invMgr.outstanding)
	invMgr.mu.Unlock()
	fetched := 0
	if stats["global"] != nil {
		fetched = stats["global"]["fetched"]
	}
	prices := invLoadPrices()
	activeRemarks := invActiveRemarks()
	// 当前下发id = 单价表 key ∪ 统计里仍在授权的备注；新增下发id（单价表里有但今日无统计）也显示为 0
	ids := map[string]bool{}
	unassigned := map[string]bool{}
	for id := range prices {
		if id != "" {
			ids[id] = true
		}
	}
	for id := range stats {
		if id == "global" {
			continue
		}
		if activeRemarks[id] {
			ids[id] = true
		} else if s := stats[id]; s != nil && s["success"]+s["fail"] > 0 {
			// 有统计但不在授权名单/单价表里的下发id（如未带 account_id 的提交），单独显示以便对账
			ids[id] = true
			unassigned[id] = true
		}
	}
	var accs []map[string]any
	totalAmount := 0.0
	for id := range ids {
		success := 0
		fail := 0
		if s := stats[id]; s != nil {
			success = s["success"]
			fail = s["fail"]
		}
		price := prices[id]
		amount := round2(price * float64(success))
		totalAmount += amount
		accs = append(accs, map[string]any{"id": id, "success": success, "fail": fail, "price": price, "amount": amount, "unassigned": unassigned[id]})
	}
	sort.Slice(accs, func(i, j int) bool { return qStr(accs[i]["id"]) < qStr(accs[j]["id"]) })
	writeJSONOK(w, map[string]any{
		"ok": true, "enabled": cfg.Enabled, "date": date,
		"queue": q, "outstanding": o, "fetched": fetched, "accounts": accs,
		"prices": prices, "total_amount": round2(totalAmount),
	})
}

// ---- 审核失败分配：失败总数 − 已分配 = 剩余，剩余平均分配到 cx 下每个账号 ----
func cxAuditFailAlloc(dataKey, date string) map[string]any {
	if date == "" {
		date = tsDate()
	}
	payload := cxReadPayloadFile(cxDateLastGoodFile(dataKey, date))
	if payload == nil {
		return map[string]any{"date": date, "dataKey": dataKey, "count": 0}
	}
	totalFail := 0
	for _, it := range rowsOf(payload, "items") {
		totalFail += qInt(it["checkFailCount"])
	}
	// 只分配当日有数据（拉取/成功/失败任一 > 0）的下发id
	activeRemarks := map[string]bool{}
	for id, s := range invStatsFor(date) {
		if s["fetched"]+s["success"]+s["fail"] > 0 {
			activeRemarks[id] = true
		}
	}
	type accRow struct {
		Username string
		Remark   string
		Own      int
		Share    int
		Total    int
	}
	var accs []accRow
	allocated := 0
	for _, r := range rowsOf(payload, "batchItems") {
		u := trimStr(r["username"])
		if u == "" {
			continue
		}
		rm := trimStr(r["_remark"])
		if !activeRemarks[rm] {
			continue
		}
		own := qInt(r["checkFailCount"])
		allocated += own
		accs = append(accs, accRow{Username: u, Remark: rm, Own: own})
	}
	remaining := totalFail - allocated
	if remaining < 0 {
		remaining = 0
	}
	n := len(accs)
	per := 0
	rem := 0
	if n > 0 {
		per = remaining / n
		rem = remaining % n
	}
	sort.Slice(accs, func(i, j int) bool { return accs[i].Username < accs[j].Username })
	for i := range accs {
		share := per
		if i < rem {
			share++
		}
		accs[i].Share = share
		accs[i].Total = accs[i].Own + share
	}
	outAccs := make([]map[string]any, 0, len(accs))
	byRemark := map[string]int{}
	byRemarkTotal := map[string]int{}
	for _, a := range accs {
		outAccs = append(outAccs, map[string]any{"username": a.Username, "remark": a.Remark, "own": a.Own, "audit": a.Share, "total": a.Total})
		byRemark[a.Remark] += a.Share
		byRemarkTotal[a.Remark] += a.Total
	}
	var remarkRows []map[string]any
	for rk, v := range byRemark {
		remarkRows = append(remarkRows, map[string]any{"remark": rk, "audit": v, "total": byRemarkTotal[rk]})
	}
	sort.Slice(remarkRows, func(i, j int) bool { return qStr(remarkRows[i]["remark"]) < qStr(remarkRows[j]["remark"]) })
	return map[string]any{
		"date": date, "dataKey": dataKey,
		"totalFail": totalFail, "allocatedFail": allocated, "remainingFail": remaining,
		"count": n, "perAccount": per,
		"accounts": outAccs, "byRemark": remarkRows,
	}
}

// handleInventoryAuditGet 后台：审核失败分配（cx.php / cx1.php）
func handleInventoryAuditGet(w http.ResponseWriter, r *http.Request) {
	date := firstNonNilStr(r.URL.Query().Get("date"), tsDate())
	writeJSONOK(w, map[string]any{
		"ok":   true,
		"date": date,
		"cx":   cxAuditFailAlloc(cxViewAccounts["532"].DataKey, date),
	})
}

func handleInventorySave(w http.ResponseWriter, r *http.Request) {
	body := readRawJSONBody(r)
	var cfg InventoryConfig
	if raw, ok := body["config"]; ok {
		b, _ := json.Marshal(raw)
		if json.Unmarshal(b, &cfg) == nil {
			cfg = normalizeInventoryConfig(cfg)
			if invSaveConfig(cfg) {
				writeJSONOK(w, map[string]any{"ok": true, "msg": "库存模式配置已保存", "data": cfg})
				return
			}
			writeJSONOK(w, map[string]any{"ok": false, "msg": "配置保存失败"})
			return
		}
	}
	// 兼容直接传配置字段
	b, _ := json.Marshal(body)
	if json.Unmarshal(b, &cfg) == nil {
		cfg = normalizeInventoryConfig(cfg)
		if invSaveConfig(cfg) {
			writeJSONOK(w, map[string]any{"ok": true, "msg": "库存模式配置已保存", "data": cfg})
			return
		}
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "配置格式错误"})
}

func normalizeInventoryConfig(cfg InventoryConfig) InventoryConfig {
	if cfg.Stock <= 0 {
		cfg.Stock = 100
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 50
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 5
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	return cfg
}

// ---- 接口1 子账号表管理（大表，约 1000+ 子账号） ----
func handleInventorySubTableGet(w http.ResponseWriter) {
	raw, err := os.ReadFile(dp(inventorySubTableFile))
	entries := loadInventorySubTable()
	if err != nil {
		writeJSONOK(w, map[string]any{"ok": true, "data": "", "count": len(entries)})
		return
	}
	writeJSONOK(w, map[string]any{"ok": true, "data": string(raw), "count": len(entries)})
}

func handleInventorySubTableSave(w http.ResponseWriter, r *http.Request) {
	input := readRawJSONBody(r)
	text := qStr(firstNonNil(input["accounts"], input["text"], input["sub_accounts"], input["table"], input["data"]))
	if text == "" {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "子账号表为空"})
		return
	}
	if !atomicWrite(dp(inventorySubTableFile), strings.TrimSpace(text)+"\n") {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "保存失败"})
		return
	}
	entries := loadInventorySubTable()
	writeJSONOK(w, map[string]any{"ok": true, "msg": "子账号表已保存", "count": len(entries)})
}

// handleInventoryDetail：实时抢任务明细 + 库存明细
func handleInventoryDetail(w http.ResponseWriter) {
	cfg := invLoadConfig()
	type taskItem struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Sub       string `json:"sub"`
		FetchedAt string `json:"fetched_at"`
		AccountID string `json:"account_id,omitempty"`
	}
	var queue, outstanding []taskItem
	invMgr.mu.Lock()
	for _, t := range invMgr.queue {
		queue = append(queue, taskItem{ID: t.TaskID, URL: t.TaskURL, Sub: t.SubUsername, FetchedAt: t.FetchedAt.Format("15:04:05")})
	}
	for _, o := range invMgr.outstanding {
		outstanding = append(outstanding, taskItem{ID: o.Task.TaskID, URL: o.Task.TaskURL, Sub: o.Task.SubUsername, FetchedAt: o.HandedAt.Format("15:04:05"), AccountID: o.AccountID})
	}
	invMgr.mu.Unlock()
	writeJSONOK(w, map[string]any{"ok": true, "enabled": cfg.Enabled, "queue": queue, "outstanding": outstanding, "events": invMgr.recentEvents(100)})
}
