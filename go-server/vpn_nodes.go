package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------------
// 免费节点抓取
//
// 从公开聚合源拉订阅，解析 vmess / vless / trojan / ss / ssr / hysteria2，
// 并发测 TCP 连通性与握手延迟，按延迟排序后通过 API 提供。
//
// 现实提醒：免费公开节点 90% 以上是不可用的（已失效、限速、被墙、被风控
// 拉黑）。这个模块的价值在于「自动剔除死节点 + 按延迟排序」，而不是
// 「找到好节点」——真正稳定的节点只能自建。
// ------------------------------------------------------------------

const (
	vpnNodesCacheFile = "vpn_nodes_cache.json"
	vpnRefreshEvery   = 10 * time.Minute
	vpnDialTimeout    = 4 * time.Second
	vpnMaxNodes       = 800
)

// vpnNode 一个解析后的节点。
type vpnNode struct {
	Protocol string `json:"protocol"` // vmess / vless / trojan / ss / ssr / hysteria2 / wg
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Raw      string `json:"raw"`      // 原始分享链接，客户端直接拿去用
	Region   string `json:"region"`   // 从名称里猜的地区
	Latency  int    `json:"latency"`  // 毫秒，-1 表示超时
	Alive    bool   `json:"alive"`
	Source   string `json:"source"`
}

// 公开订阅源。这些链接本身会变动，抓不到是常态，代码要容忍失败。
var vpnSubSources = []string{
	// GitHub Actions 自动维护的聚合订阅（Clash / base64）
	"https://raw.githubusercontent.com/sunmiao4458/free-proxy-airport/main/sub/base64.txt",
	"https://raw.githubusercontent.com/sunmiao4458/free-proxy-airport/main/sub/clash.yaml",
	"https://raw.githubusercontent.com/mahdibland/V2RayAggregator/master/sub/sub_merge_base64.txt",
	"https://raw.githubusercontent.com/aiboboxx/v2rayfree/main/v2",
	"https://raw.githubusercontent.com/Pawdroid/Free-servers/main/sub",
	"https://raw.githubusercontent.com/ripaojiedian/freenode/main/sub",
}

var (
	vpnMu       sync.RWMutex
	vpnCache    []vpnNode
	vpnLastSync time.Time
	vpnSyncing  bool
)

// fetchText 带超时抓取文本。
func fetchText(src string) string {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; node-aggregator/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 8192)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > 4<<20 {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return string(buf)
}

// maybeBase64 如果整体是 base64 就解开。
func maybeBase64(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 已经是明文协议列表就不动
	if strings.Contains(s, "://") && !strings.Contains(s, "proxies:") {
		return s
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if dec, err := enc.DecodeString(cleaned); err == nil {
			out := string(dec)
			if strings.Contains(out, "://") {
				return out
			}
		}
	}
	return s
}

// parseVmess 解析 vmess://base64(json)。
func parseVmess(link, src string) *vpnNode {
	b := strings.TrimPrefix(link, "vmess://")
	dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b))
	if err != nil {
		if dec2, err2 := base64.RawStdEncoding.DecodeString(strings.TrimSpace(b)); err2 == nil {
			dec = dec2
		} else {
			return nil
		}
	}
	var m map[string]any
	if json.Unmarshal(dec, &m) != nil {
		return nil
	}
	host := trimStr(firstNonNil(m["add"], m["host"]))
	port := qInt(firstNonNil(m["port"], 0))
	if host == "" || port <= 0 {
		return nil
	}
	return &vpnNode{
		Protocol: "vmess", Name: trimStr(m["ps"]), Server: host, Port: port,
		Raw: link, Region: guessRegion(trimStr(m["ps"])), Latency: -1, Source: src,
	}
}

// parseURI 解析 vless / trojan / hysteria2 / ss 这类标准 URI。
func parseURI(link, src string) *vpnNode {
	scheme := ""
	for _, s := range []string{"vless", "trojan", "hysteria2", "hy2", "ss", "ssr"} {
		if strings.HasPrefix(link, s+"://") {
			scheme = s
			break
		}
	}
	if scheme == "" {
		return nil
	}
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return nil
	}
	port := 0
	if p := u.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}
	if port <= 0 {
		return nil
	}
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	proto := scheme
	if scheme == "hy2" {
		proto = "hysteria2"
	}
	return &vpnNode{
		Protocol: proto, Name: name, Server: u.Hostname(), Port: port,
		Raw: link, Region: guessRegion(name), Latency: -1, Source: src,
	}
}

// guessRegion 从节点名里猜地区，用于客户端筛选。
func guessRegion(name string) string {
	n := strings.ToUpper(name)
	table := []struct{ key, region string }{
		{"台湾", "TW"}, {"台灣", "TW"}, {"TAIWAN", "TW"}, {"TW", "TW"},
		{"香港", "HK"}, {"HONGKONG", "HK"}, {"HONG KONG", "HK"}, {"HK", "HK"},
		{"日本", "JP"}, {"JAPAN", "JP"}, {"JP", "JP"}, {"东京", "JP"},
		{"新加坡", "SG"}, {"SINGAPORE", "SG"}, {"SG", "SG"},
		{"美国", "US"}, {"UNITED STATES", "US"}, {"USA", "US"}, {"US", "US"},
		{"韩国", "KR"}, {"KOREA", "KR"}, {"KR", "KR"},
		{"德国", "DE"}, {"GERMANY", "DE"}, {"DE", "DE"},
		{"英国", "GB"}, {"UK", "GB"}, {"GB", "GB"},
		{"荷兰", "NL"}, {"NL", "NL"},
		{"法国", "FR"}, {"FR", "FR"},
	}
	for _, t := range table {
		if strings.Contains(n, t.key) {
			return t.region
		}
	}
	return "UNKNOWN"
}

// parseSub 把一份订阅内容拆成节点。
func parseSub(content, src string) []vpnNode {
	content = maybeBase64(content)
	var out []vpnNode
	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Clash YAML 里的 server/port 单独成行，这里只处理 URI 形式
		var n *vpnNode
		switch {
		case strings.HasPrefix(line, "vmess://"):
			n = parseVmess(line, src)
		case strings.Contains(line, "://"):
			n = parseURI(line, src)
		}
		if n != nil {
			out = append(out, *n)
		}
	}
	return out
}

// probe 测 TCP 握手延迟。
func probe(n *vpnNode) {
	addr := net.JoinHostPort(n.Server, strconv.Itoa(n.Port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, vpnDialTimeout)
	if err != nil {
		n.Alive = false
		n.Latency = -1
		return
	}
	n.Latency = int(time.Since(start).Milliseconds())
	n.Alive = true
	_ = conn.Close()
}

// dedupNodes 按 server:port 去重。
func dedupNodes(nodes []vpnNode) []vpnNode {
	seen := map[string]bool{}
	out := make([]vpnNode, 0, len(nodes))
	for _, n := range nodes {
		key := fmt.Sprintf("%s:%d", strings.ToLower(n.Server), n.Port)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
	}
	return out
}

// syncVPNNodes 抓取 + 测速 + 更新缓存。
func syncVPNNodes() {
	vpnMu.Lock()
	if vpnSyncing {
		vpnMu.Unlock()
		return
	}
	vpnSyncing = true
	vpnMu.Unlock()

	defer func() {
		vpnMu.Lock()
		vpnSyncing = false
		vpnMu.Unlock()
	}()

	var all []vpnNode
	for _, src := range vpnSubSources {
		body := fetchText(src)
		if body == "" {
			continue
		}
		all = append(all, parseSub(body, src)...)
		if len(all) > vpnMaxNodes*3 {
			break
		}
	}
	all = dedupNodes(all)
	if len(all) > vpnMaxNodes {
		all = all[:vpnMaxNodes]
	}

	// 并发测速
	sem := make(chan struct{}, 64)
	var wg sync.WaitGroup
	for i := range all {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			probe(&all[idx])
		}(i)
	}
	wg.Wait()

	// 只保留活的，按延迟升序
	alive := make([]vpnNode, 0, len(all))
	for _, n := range all {
		if n.Alive {
			alive = append(alive, n)
		}
	}
	sort.Slice(alive, func(i, j int) bool { return alive[i].Latency < alive[j].Latency })

	vpnMu.Lock()
	vpnCache = alive
	vpnLastSync = time.Now()
	vpnMu.Unlock()

	// 落盘，重启后先给旧数据
	runCatchingWrite(vpnNodesCacheFile, map[string]any{
		"updated_at": tsDateTime(),
		"count":      len(alive),
		"nodes":      alive,
	})
}

func runCatchingWrite(file string, v any) {
	defer func() { _ = recover() }()
	_ = atomicWrite(dp(file), jsonEncodePretty(v))
}

// startVPNRefresher 后台定时刷新。
func startVPNRefresher() {
	go func() {
		// 启动时先读缓存，避免第一次请求空手而归
		var cached map[string]any
		if readJSONFile(dp(vpnNodesCacheFile), &cached) {
			if rows, ok := cached["nodes"].([]any); ok {
				nodes := make([]vpnNode, 0, len(rows))
				for _, r := range rows {
					m, ok := r.(map[string]any)
					if !ok {
						continue
					}
					nodes = append(nodes, vpnNode{
						Protocol: trimStr(m["protocol"]), Name: trimStr(m["name"]),
						Server: trimStr(m["server"]), Port: qInt(m["port"]),
						Raw: trimStr(m["raw"]), Region: trimStr(m["region"]),
						Latency: qInt(m["latency"]), Alive: asBool(m["alive"], false),
						Source: trimStr(m["source"]),
					})
				}
				vpnMu.Lock()
				vpnCache = nodes
				vpnMu.Unlock()
			}
		}
		syncVPNNodes()
		t := time.NewTicker(vpnRefreshEvery)
		for range t.C {
			syncVPNNodes()
		}
	}()
}

// handleVPNNodes 返回节点列表。
//
//	GET|POST ?region=TW&protocol=vmess&limit=50&alive=1
func handleVPNNodes(w http.ResponseWriter, r *http.Request) {
	vpnMu.RLock()
	nodes := append([]vpnNode{}, vpnCache...)
	last := vpnLastSync
	vpnMu.RUnlock()

	_ = r.ParseForm()
	wantRegion := strings.ToUpper(strings.TrimSpace(r.FormValue("region")))
	wantProto := strings.ToLower(strings.TrimSpace(r.FormValue("protocol")))
	onlyAlive := r.FormValue("alive") != "0"
	limit := 50
	if v := r.FormValue("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}

	out := make([]vpnNode, 0, limit)
	for _, n := range nodes {
		if onlyAlive && !n.Alive {
			continue
		}
		if wantRegion != "" && wantRegion != "ALL" && n.Region != wantRegion {
			continue
		}
		if wantProto != "" && wantProto != "all" && n.Protocol != wantProto {
			continue
		}
		out = append(out, n)
		if len(out) >= limit {
			break
		}
	}

	// 统计各地区数量，便于客户端做选择
	regionStat := map[string]int{}
	for _, n := range nodes {
		if n.Alive {
			regionStat[n.Region]++
		}
	}

	writeJSONOK(w, map[string]any{
		"ok": true, "code": "200",
		"data":       out,
		"total":      len(out),
		"pool_total": len(nodes),
		"regions":    regionStat,
		"updated_at": last.Format("2006-01-02 15:04:05"),
	})
}

// handleVPNRefresh 手动触发刷新。
func handleVPNRefresh(w http.ResponseWriter, r *http.Request) {
	go syncVPNNodes()
	writeJSONOK(w, map[string]any{"ok": true, "msg": "已开始刷新，稍后再取列表"})
}
