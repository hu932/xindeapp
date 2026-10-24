package main

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------------
// IP 归属地反查
//
// 很多免费节点的名字里没有地区信息，例如 RELAY-104.21.48.161-1156。
// 这类节点分两种情况，必须区别对待：
//
//   1. 普通 VPS：按 IP 反查国家是准的
//   2. Cloudflare anycast（104.16.0.0/13、172.64.0.0/13 等）：
//      出口是离客户端最近的 CF 边缘节点，跟注册地无关，
//      给它标 US 之类是误导，这里统一标成 CF。
//
// 查询顺序：ip-api.com 批量 → geojs.io 批量 → ipwho.is 逐个兜底。
// 全部带缓存，重复 IP 不再查，缓存 7 天。
// ------------------------------------------------------------------

const (
	geoCacheFile    = "geoip_cache.json"
	geoCacheTTL     = 7 * 24 * time.Hour
	geoBatchSize    = 100
	geoPerSyncLimit = 260 // 单次刷新最多查多少个 IP，防止免费额度被刷光
	regionCF        = "CF"
)

type geoEntry struct {
	Country string `json:"country"`
	At      int64  `json:"at"`
}

var (
	geoMu    sync.Mutex
	geoCache = map[string]geoEntry{}
	geoDirty bool
)

// cloudflareNets Cloudflare 公布的主要网段，命中即视为 anycast 中继。
var cloudflareNets = func() []*net.IPNet {
	cidrs := []string{
		"103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"104.16.0.0/13", "104.24.0.0/14",
		"108.162.192.0/18",
		"131.0.72.0/22",
		"141.101.64.0/18",
		"162.158.0.0/15",
		"172.64.0.0/13",
		"173.245.48.0/20",
		"188.114.96.0/20",
		"190.93.240.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			out = append(out, n)
		}
	}
	return out
}()

func isCloudflare(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range cloudflareNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func loadGeoCache() {
	var data map[string]any
	if !readJSONFile(dp(geoCacheFile), &data) {
		return
	}
	rows, _ := data["items"].(map[string]any)
	geoMu.Lock()
	defer geoMu.Unlock()
	for ip, v := range rows {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		geoCache[ip] = geoEntry{
			Country: trimStr(m["country"]),
			At:      int64(qInt(m["at"])),
		}
	}
}

func saveGeoCache() {
	geoMu.Lock()
	if !geoDirty {
		geoMu.Unlock()
		return
	}
	rows := map[string]any{}
	for k, v := range geoCache {
		rows[k] = map[string]any{"country": v.Country, "at": v.At}
	}
	geoDirty = false
	geoMu.Unlock()

	_ = atomicWrite(dp(geoCacheFile), jsonEncodePretty(map[string]any{
		"items": rows, "updated_at": tsDateTime(),
	}))
}

func geoLookupCached(ip string) (string, bool) {
	geoMu.Lock()
	defer geoMu.Unlock()
	e, ok := geoCache[ip]
	if !ok || e.Country == "" {
		return "", false
	}
	if time.Since(time.Unix(e.At, 0)) > geoCacheTTL {
		return "", false
	}
	return e.Country, true
}

func geoStore(ip, country string) {
	if country == "" || ip == "" {
		return
	}
	geoMu.Lock()
	geoCache[ip] = geoEntry{Country: strings.ToUpper(country), At: time.Now().Unix()}
	geoDirty = true
	geoMu.Unlock()
}

// resolveHost 把节点地址解析成 IP。
func resolveHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if net.ParseIP(host) != nil {
		return host
	}
	type res struct{ ip string }
	ch := make(chan res, 1)
	go func() {
		addrs, err := net.LookupHost(host)
		if err != nil || len(addrs) == 0 {
			ch <- res{}
			return
		}
		for _, a := range addrs {
			if p := net.ParseIP(a); p != nil && p.To4() != nil {
				ch <- res{a}
				return
			}
		}
		ch <- res{addrs[0]}
	}()
	select {
	case r := <-ch:
		return r.ip
	case <-time.After(3 * time.Second):
		return ""
	}
}

func httpGetBody(url string, timeout time.Duration) []byte {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	buf := make([]byte, 0, 16*1024)
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
	return buf
}

// geoBatchIPAPI ip-api.com 批量。海外服务器通常可用，被墙时返回空。
func geoBatchIPAPI(ips []string) map[string]string {
	out := map[string]string{}
	payload := make([]map[string]string, 0, len(ips))
	for _, ip := range ips {
		payload = append(payload, map[string]string{"query": ip})
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return out
	}
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("POST",
		"http://ip-api.com/batch?fields=query,countryCode,status",
		strings.NewReader(string(body)))
	if err != nil {
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return out
	}
	var rows []map[string]any
	if json.NewDecoder(resp.Body).Decode(&rows) != nil {
		return out
	}
	for _, r := range rows {
		q := trimStr(r["query"])
		cc := trimStr(r["countryCode"])
		st := trimStr(r["status"])
		if q != "" && cc != "" && (st == "" || st == "success") {
			out[q] = strings.ToUpper(cc)
		}
	}
	return out
}

// geoBatchGeojs geojs.io 批量。对 Cloudflare 段返回空，只用于普通 IP。
func geoBatchGeojs(ips []string) map[string]string {
	out := map[string]string{}
	if len(ips) == 0 {
		return out
	}
	u := "https://get.geojs.io/v1/ip/country.json?ip=" + strings.Join(ips, ",")
	body := httpGetBody(u, 20*time.Second)
	if len(body) == 0 {
		return out
	}
	var rows []map[string]any
	if json.Unmarshal(body, &rows) != nil {
		return out
	}
	for _, r := range rows {
		ip := trimStr(r["ip"])
		cc := trimStr(r["country"])
		if ip != "" && cc != "" {
			out[ip] = strings.ToUpper(cc)
		}
	}
	return out
}

// geoLookupOne ipwho.is 单个兜底。
func geoLookupOne(ip string) string {
	body := httpGetBody("https://ipwho.is/"+ip+"?fields=country_code,success", 8*time.Second)
	if len(body) == 0 {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	if ok, _ := m["success"].(bool); !ok {
		return ""
	}
	return strings.ToUpper(trimStr(m["country_code"]))
}

// fillUnknownRegions 给地区未知的节点补上国家码。
//
//   1. Cloudflare 段直接标 CF（反查没有意义）
//   2. 缓存命中直接用
//   3. 批量查（ip-api → geojs）
//   4. 仍未命中逐个查 ipwho.is，受 geoPerSyncLimit 约束
func fillUnknownRegions(nodes []vpnNode) int {
	pending := map[string][]int{}
	filled := 0

	for i := range nodes {
		if nodes[i].Region != "" && nodes[i].Region != "未知" {
			continue
		}
		ip := resolveHost(nodes[i].Server)
		if ip == "" || net.ParseIP(ip) == nil {
			continue
		}
		if isCloudflare(ip) {
			nodes[i].Region = regionCF
			filled++
			continue
		}
		if cc, ok := geoLookupCached(ip); ok {
			nodes[i].Region = cc
			filled++
			continue
		}
		pending[ip] = append(pending[ip], i)
	}

	if len(pending) > 0 {
		ips := make([]string, 0, len(pending))
		for ip := range pending {
			ips = append(ips, ip)
		}
		if len(ips) > geoPerSyncLimit {
			ips = ips[:geoPerSyncLimit]
		}

		for start := 0; start < len(ips); start += geoBatchSize {
			end := start + geoBatchSize
			if end > len(ips) {
				end = len(ips)
			}
			chunk := ips[start:end]

			result := geoBatchIPAPI(chunk)
			if len(result) == 0 {
				result = geoBatchGeojs(chunk)
			}

			missed := make([]string, 0, len(chunk))
			for _, ip := range chunk {
				if _, ok := result[ip]; !ok {
					missed = append(missed, ip)
				}
			}
			if len(missed) > 15 {
				missed = missed[:15]
			}
			for _, ip := range missed {
				if cc := geoLookupOne(ip); cc != "" {
					result[ip] = cc
				}
			}

			for ip, cc := range result {
				geoStore(ip, cc)
				for _, idx := range pending[ip] {
					nodes[idx].Region = cc
					filled++
				}
			}
		}
	}

	saveGeoCache()
	return filled
}
