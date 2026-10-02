package main

import (
	"os"
	"strings"
	"sync"
	"time"
)

// ===================== 任务库存模式（与旧功能完全隔离，开启后才生效） =====================

// 说明：本模式“拉任务”与“回传”都走接口1（zb1）自己的任务接口：
//   - 拉任务：GET  apiTakeURLForType(1, take_type)   （/api/task/take 或 ?type=r）
//   - 回传：  POST apiSubmitURLForType(1)             （/api/task/submit/v2）
// 本模块只负责：抢任务进库存 + 封装任务接口给 app/插件。

const inventoryConfigFile = "task_inventory.json"
const inventoryStatsFile = "task_inventory_stats.json"
const inventorySubTableFile = "task_inventory_sub_accounts.txt" // 接口1子账号表（约 1000+，文本每行一条）
const inventoryPricesFile = "task_inventory_prices.json"       // 库存模式各下发id单价（成功一条的金额）

// InvSubAccount 接口1 子账号（用于登录接口1并取任务）
type InvSubAccount struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// invSubEntry 子账号表（task_inventory_sub_accounts.txt）解析后的一条
type invSubEntry struct {
	Group    string // 下发id（空 = 全局池，所有下发id共享轮询）
	Username string
	Password string
}

// InventoryConfig 库存模式配置（全局库存，不分下发id）
type InventoryConfig struct {
	Enabled         bool   `json:"enabled"`          // 总开关
	Stock           int    `json:"stock"`            // 全局库存目标（保持多少条在库）
	Concurrency     int    `json:"concurrency"`      // 每轮并发拉取账号数
	IntervalSeconds int    `json:"interval_seconds"` // 拉取间隔秒
	TimeoutSeconds  int    `json:"timeout_seconds"`  // 任务超时秒（默认 300 = 5 分钟）
	TakeType        string `json:"take_type"`        // "" 普通 / "r" R任务
}

type invTask struct {
	SubUsername string
	TaskID      string // trace_id（回传凭据）
	TaskURL     string // 接口1 任务 URL（回传时用）
	Token       string // 该子账号取任务时的 token（回传时用）
	Task        map[string]any
	FetchedAt   time.Time
}

type invOutstanding struct {
	Task      invTask
	HandedAt  time.Time
	AccountID string // 取走该任务的 app 下发id（用于回传匹配与计数）
}

type invFetchEvent struct {
	Time   string `json:"time"`
	Sub    string `json:"sub"`
	TaskID string `json:"task_id"`
	OK     bool   `json:"ok"`
	Note   string `json:"note"`
}

type inventoryManager struct {
	mu          sync.Mutex
	queue       []invTask
	outstanding map[string]*invOutstanding
	nextFetchAt time.Time
	poolCursor  int
	statsMu     sync.Mutex
	eventMu     sync.Mutex
	recent      []invFetchEvent
}

var invMgr = &inventoryManager{outstanding: map[string]*invOutstanding{}}

// recordEvent 记录抢任务事件（最多保留最近 200 条）
func (m *inventoryManager) recordEvent(e invFetchEvent) {
	e.Time = timeNow().Format("15:04:05")
	m.eventMu.Lock()
	defer m.eventMu.Unlock()
	m.recent = append(m.recent, e)
	if len(m.recent) > 200 {
		m.recent = m.recent[len(m.recent)-200:]
	}
}

// recentEvents 返回最近的抢任务事件（新→旧）
func (m *inventoryManager) recentEvents(limit int) []invFetchEvent {
	m.eventMu.Lock()
	defer m.eventMu.Unlock()
	n := len(m.recent)
	if limit <= 0 || limit >= n {
		limit = n
	}
	out := make([]invFetchEvent, limit)
	for i := 0; i < limit; i++ {
		out[i] = m.recent[n-limit+i]
	}
	return out
}

func invDefaults() InventoryConfig {
	return InventoryConfig{Enabled: false, Stock: 100, Concurrency: 50, IntervalSeconds: 5, TimeoutSeconds: 300, TakeType: ""}
}

func invLoadConfig() InventoryConfig {
	cfg := invDefaults()
	_ = readJSONFile(dp(inventoryConfigFile), &cfg)
	// 补齐全局默认值
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

func invSaveConfig(cfg InventoryConfig) bool {
	return atomicWrite(dp(inventoryConfigFile), jsonEncodePretty(cfg))
}

// invLoadPrices 库存模式各下发id单价（成功一条的金额）
func invLoadPrices() map[string]float64 {
	var data struct {
		Prices map[string]float64 `json:"prices"`
	}
	_ = readJSONFile(dp(inventoryPricesFile), &data)
	if data.Prices == nil {
		data.Prices = map[string]float64{}
	}
	return data.Prices
}

func invSavePrices(prices map[string]float64) bool {
	return atomicWrite(dp(inventoryPricesFile), jsonEncodePretty(map[string]any{"prices": prices}))
}

// tick 每 1 秒执行一次：到点补库存、清理超时任务
func (m *inventoryManager) tick() {
	cfg := invLoadConfig()
	now := time.Now()
	timeout := time.Duration(max(30, cfg.TimeoutSeconds)) * time.Second
	// 无论开关状态都先清理超时任务，否则关闭库存模式后，超时任务仍显示“在库”
	m.cleanup(timeout)
	if !cfg.Enabled {
		return
	}

	if now.Before(m.nextFetchAt) {
		return
	}
	m.nextFetchAt = now.Add(time.Duration(max(1, cfg.IntervalSeconds)) * time.Second)

	m.mu.Lock()
	queueLen := len(m.queue)
	outLen := len(m.outstanding)
	m.mu.Unlock()
	need := cfg.Stock - queueLen - outLen
	if need <= 0 {
		return
	}
	pool := invGlobalPool()
	if len(pool) == 0 {
		return
	}
	slots := need
	if cfg.Concurrency > 0 && slots > cfg.Concurrency {
		slots = cfg.Concurrency
	}
	if slots > len(pool) {
		slots = len(pool)
	}
	go m.fetchRound(slots, pool, cfg.TakeType)
}

func (m *inventoryManager) cleanup(timeout time.Duration) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.queue[:0]
	for _, t := range m.queue {
		if now.Sub(t.FetchedAt) <= timeout {
			kept = append(kept, t)
		}
	}
	m.queue = kept
	for k, o := range m.outstanding {
		if now.Sub(o.HandedAt) > timeout {
			delete(m.outstanding, k)
		}
	}
}

// loadInventorySubTable 读取子账号表（每行一条），分隔符兼容：竖线 |、制表符 tab、逗号、空格：
//   username password              -> 全局池（所有下发id共享，轮询）
//   下发id username password        -> 只给该下发id用
// 以 # 开头或空行跳过
func loadInventorySubTable() []invSubEntry {
	var out []invSubEntry
	raw, err := os.ReadFile(dp(inventorySubTableFile))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts, sep := splitInvSubLine(line)
		if len(parts) < 2 {
			continue
		}
		var e invSubEntry
		if len(parts) == 2 {
			e.Username = parts[0]
			e.Password = parts[1]
		} else {
			e.Group = parts[0]
			e.Username = parts[1]
			e.Password = strings.Join(parts[2:], sep)
		}
		if e.Username == "" || e.Password == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

// splitInvSubLine 把一行按分隔符拆开（兼容 | 、tab、逗号、空格），返回字段与拼接分隔符
func splitInvSubLine(line string) ([]string, string) {
	line = strings.ReplaceAll(line, "｜", "|")
	switch {
	case strings.Contains(line, "|"):
		return trimFields(strings.Split(line, "|")), "|"
	case strings.Contains(line, "\t"):
		return trimFields(strings.FieldsFunc(line, func(r rune) bool { return r == '\t' })), ""
	case strings.ContainsAny(line, "，,"):
		return trimFields(strings.FieldsFunc(strings.ReplaceAll(line, "，", ","), func(r rune) bool { return r == ',' })), ","
	default:
		return strings.Fields(line), " "
	}
}

// trimFields 去掉空白字段
func trimFields(parts []string) []string {
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// invGlobalPool 全局子账号池（子账号表里所有账号，不分下发id），剔除插件账号（授权名单）下的任务账号，避免与插件直接抢任务冲突
func invGlobalPool() []InvSubAccount {
	excluded := map[string]bool{}
	for _, acc := range readAllowedAccounts(1) {
		if u := trimStr(acc["username"]); u != "" {
			excluded[u] = true
		}
	}
	var pool []InvSubAccount
	for _, e := range loadInventorySubTable() {
		if excluded[e.Username] {
			continue
		}
		pool = append(pool, InvSubAccount{Username: e.Username, Password: e.Password})
	}
	return pool
}

// fetchRound 用全局子账号池里的 slots 个不同账号并发拉任务入库（轮询游标，避免重复用同一账号）
func (m *inventoryManager) fetchRound(slots int, pool []InvSubAccount, takeType string) {
	if slots <= 0 || len(pool) == 0 {
		return
	}
	takeURL := apiTakeURLForType(1, takeType)

	// 取轮询起点并推进游标，保证同一轮用不同账号
	m.mu.Lock()
	start := m.poolCursor % len(pool)
	m.poolCursor = (start + slots) % len(pool)
	m.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < slots; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sub := pool[(start+idx)%len(pool)]
			auth := upstreamAuth{LoginURL: api1LoginURL, Username: sub.Username, Password: sub.Password, Label: "接口1", APIType: 1}
			tf := "task_inventory_tokens/api1_token_" + md5short(sub.Username) + ".json"
			ti := getAPIToken(auth, tf, false)
			if !ti.OK {
				m.recordEvent(invFetchEvent{Sub: sub.Username, Note: "登录失败"})
				return
			}
			res := forwardGetToServer(takeURL, []string{"Authorization: Bearer " + ti.Token})
			if res.Error != "" {
				m.recordEvent(invFetchEvent{Sub: sub.Username, Note: "拉取失败"})
				return
			}
			var serverResp map[string]any
			if !jsonDecode(res.Response, &serverResp) {
				m.recordEvent(invFetchEvent{Sub: sub.Username, Note: "响应解析失败"})
				return
			}
			if trimStr(serverResp["code"]) != "200" {
				m.recordEvent(invFetchEvent{Sub: sub.Username, Note: "上游code=" + trimStr(serverResp["code"])})
				return
			}
			data, _ := serverResp["data"].(map[string]any)
			if data == nil || trimStr(data["taskUrl"]) == "" {
				return // 无任务，正常，不记事件
			}
			taskID := trimStr(firstNonNil(data["taskId"], data["task_id"], data["deal_id"], data["trace_id"], data["id"]))
			if taskID == "" {
				taskID = md5short(trimStr(data["taskUrl"]))
			}
			// 封装给客户端：接口1 的 data + trace_id
			wrapped := map[string]any{}
			for k, v := range data {
				wrapped[k] = v
			}
			wrapped["trace_id"] = taskID
			m.mu.Lock()
			m.queue = append(m.queue, invTask{
				SubUsername: sub.Username, TaskID: taskID,
				TaskURL: trimStr(data["taskUrl"]), Token: ti.Token, Task: wrapped, FetchedAt: time.Now(),
			})
			m.mu.Unlock()
			invRecordFetch()
			m.recordEvent(invFetchEvent{Sub: sub.Username, TaskID: taskID, OK: true, Note: "抢到任务"})
		}(i)
	}
	wg.Wait()
}

// ---- 取任务：从全局库存弹出 ----
func (m *inventoryManager) popTask(accountID string, timeout time.Duration) *invTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for len(m.queue) > 0 {
		t := m.queue[0]
		m.queue = m.queue[1:]
		if now.Sub(t.FetchedAt) > timeout {
			continue // 已过期
		}
		if t.TaskID != "" {
			m.outstanding[t.TaskID] = &invOutstanding{Task: t, HandedAt: now, AccountID: accountID}
		}
		return &t
	}
	return nil
}

func (m *inventoryManager) takeOutstanding(accountID, traceID string, timeout time.Duration) (*invTask, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.outstanding[traceID]
	if !ok {
		return nil, ""
	}
	if time.Since(o.HandedAt) > timeout {
		delete(m.outstanding, traceID)
		return nil, ""
	}
	if accountID != "" && o.AccountID != "" && accountID != o.AccountID {
		return nil, "" // 下发id不匹配
	}
	delete(m.outstanding, traceID)
	return &o.Task, o.AccountID
}

// ---- 统计（下发id 维度 + 全局拉取量） ----
func invRecordSuccess(accountID string) { invBumpStat(accountID, "success") }
func invRecordFetch()                   { invBumpStat("global", "fetched") }
func invRecordFail(accountID string)    { invBumpStat(accountID, "fail") }

func invBumpStat(accountID, kind string) {
	date := tsDate()
	invMgr.statsMu.Lock()
	defer invMgr.statsMu.Unlock()
	stats := map[string]map[string]map[string]int{}
	_ = readJSONFile(dp(inventoryStatsFile), &stats)
	if stats == nil {
		stats = map[string]map[string]map[string]int{}
	}
	if stats[date] == nil {
		stats[date] = map[string]map[string]int{}
	}
	if stats[date][accountID] == nil {
		stats[date][accountID] = map[string]int{"success": 0, "fail": 0, "fetched": 0}
	}
	stats[date][accountID][kind]++
	_ = atomicWrite(dp(inventoryStatsFile), jsonEncode(stats))
}

func invStatsFor(date string) map[string]map[string]int {
	stats := map[string]map[string]map[string]int{}
	_ = readJSONFile(dp(inventoryStatsFile), &stats)
	return stats[date]
}

// invActiveRemarks 当前接口1授权账号里出现过的备注（下发id），删除账号后自动隐藏其统计行
func invActiveRemarks() map[string]bool {
	set := map[string]bool{}
	for _, acc := range readAllowedAccounts(1) {
		if r := trimStr(acc["remark"]); r != "" {
			set[r] = true
		}
	}
	return set
}

// invRemoveAccountStats：删除下发id（插件账号）后清理其库存统计
func invRemoveAccountStats(keys ...string) {
	var real []string
	for _, k := range keys {
		if t := strings.TrimSpace(k); t != "" {
			real = append(real, t)
		}
	}
	if len(real) == 0 {
		return
	}
	invMgr.statsMu.Lock()
	defer invMgr.statsMu.Unlock()
	stats := map[string]map[string]map[string]int{}
	if !readJSONFile(dp(inventoryStatsFile), &stats) {
		return
	}
	changed := false
	for _, k := range real {
		for _, byID := range stats {
			if _, ok := byID[k]; ok {
				delete(byID, k)
				changed = true
			}
		}
	}
	if changed {
		_ = atomicWrite(dp(inventoryStatsFile), jsonEncode(stats))
	}
}

// startInventoryManager 启动后台拉取循环
func startInventoryManager() {
	initUpstreamClient()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			invMgr.tick()
		}
	}()
}
