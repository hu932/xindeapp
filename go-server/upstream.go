package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---- 共享上游客户端：连接池 + keep-alive + 静态 DNS 解析 ----

var (
	upstreamOnce    sync.Once
	upstreamClient  *http.Client
	resolvePrefMu   sync.Mutex
	resolvePref     = map[string]map[string]any{} // key(host:port) -> {ip, ts}
)

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		d := net.Dialer{Timeout: time.Duration(forwardConnTimout) * time.Second}
		return d.DialContext(ctx, network, addr)
	}
	if os.Getenv("GEJIE_USE_SYSTEM_DNS") == "1" {
		d := net.Dialer{Timeout: time.Duration(forwardConnTimout) * time.Second}
		return d.DialContext(ctx, network, addr)
	}
	ip := resolveIP(host, port)
	if ip == "" {
		d := net.Dialer{Timeout: time.Duration(forwardConnTimout) * time.Second}
		return d.DialContext(ctx, network, addr)
	}
	d := net.Dialer{Timeout: time.Duration(forwardConnTimout) * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(ip, port))
}

func resolveIP(host, port string) string {
	key := strings.ToLower(host) + ":" + port
	var ips []string
	for _, entry := range staticDNSResolve {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		i := strings.LastIndex(entry, ":")
		if i < 0 {
			continue
		}
		if strings.EqualFold(entry[:i], key) {
			ips = append(ips, entry[i+1:])
		}
	}
	ips = dedupe(ips)
	if len(ips) == 0 {
		// 未知域名走系统 DNS
		return ""
	}
	resolvePrefMu.Lock()
	defer resolvePrefMu.Unlock()
	if good, ok := resolvePref[key]["ip"].(string); ok && good != "" {
		ordered := []string{}
		for _, ip := range ips {
			if ip == good {
				ordered = append([]string{ip}, ordered...)
			} else {
				ordered = append(ordered, ip)
			}
		}
		ips = ordered
	}
	return ips[0]
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func noteResolveSuccess(urlStr, ip string) {
	if ip == "" {
		return
	}
	u, err := parseURLSimple(urlStr)
	if err != nil || u.Host == "" {
		return
	}
	host := strings.ToLower(u.Host)
	port := u.Port
	if port == "" {
		port = "443"
	}
	key := host + ":" + port
	resolvePrefMu.Lock()
	if p, ok := resolvePref[key]; ok && p["ip"] == ip {
		resolvePrefMu.Unlock()
		return
	}
	resolvePref[key] = map[string]any{"ip": ip, "ts": time.Now().Unix()}
	snapshot := map[string]any{}
	for k, v := range resolvePref {
		snapshot[k] = v
	}
	resolvePrefMu.Unlock()
	_ = atomicWrite(dp(resolvePrefFile), jsonEncode(snapshot))
}

type simpleURL struct{ Host, Port, Scheme string }

func parseURLSimple(s string) (*simpleURL, error) {
	u, err := parseURL(s)
	return u, err
}

func parseURL(s string) (*simpleURL, error) {
	// 简化：仅解析 scheme://host:port
	rest := s
	scheme := ""
	if i := strings.Index(rest, "://"); i >= 0 {
		scheme = rest[:i]
		rest = rest[i+3:]
	}
	hostport := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		hostport = rest[:i]
	}
	host := hostport
	port := ""
	if i := strings.LastIndex(hostport, ":"); i >= 0 && !strings.Contains(hostport[i:], "]") {
		host = hostport[:i]
		port = hostport[i+1:]
	}
	return &simpleURL{Host: host, Port: port, Scheme: scheme}, nil
}

func initUpstreamClient() {
	upstreamOnce.Do(func() {
		// 载入 resolve pref
		_ = readJSONFile(dp(resolvePrefFile), &resolvePref)
		tr := &http.Transport{
			DialContext:           dialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ForceAttemptHTTP2:     false,
			DisableCompression:    true,
			ResponseHeaderTimeout: time.Duration(forwardTimeout+4) * time.Second,
		}
		upstreamClient = &http.Client{Transport: tr}
	})
}

// forwardResult 与 PHP forwardToServer 返回结构对齐
type forwardResult struct {
	Response     string
	Error        string
	HTTPCode     int
	CurlTotalMs  int
	TCPMs        int
	TLSMs        int
	ServerMs     int
	PrimaryIP    string
	GateWaitMs   int
	Busy         bool
	ContentType  string
	FinalURL     string
}

type traceTimes struct {
	start     time.Time
	dnsDone   time.Time
	gotConn   time.Time
	tlsDone   time.Time
	firstByte time.Time
	ip        string
}

func newTrace(tt *traceTimes) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSDone: func(info httptrace.DNSDoneInfo) { tt.dnsDone = time.Now() },
		GotConn: func(info httptrace.GotConnInfo) {
			tt.gotConn = time.Now()
			tt.ip = info.Conn.RemoteAddr().String()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) { tt.tlsDone = time.Now() },
		GotFirstResponseByte: func() { tt.firstByte = time.Now() },
	}
}

func ms(t time.Time) int {
	if t.IsZero() {
		return 0
	}
	return int(t.Sub(time.Time{}).Milliseconds())
}

// doForward 发送一次上游请求（带全局闸门）
func doForward(method, url, body string, gzipBody bool, extraHeaders []string, timeoutSec int) forwardResult {
	initUpstreamClient()
	tt := &traceTimes{start: time.Now()}
	gateWait := 0
	ok := true
	if fg.enabled() {
		ok, gateWait = fg.acquire()
	}
	if !ok {
		return forwardResult{
			Error:      "系统繁忙，请稍后重试（转发并发已满）",
			GateWaitMs: gateWait,
			Busy:       true,
		}
	}
	defer fg.release()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	ctx = httptrace.WithClientTrace(ctx, newTrace(tt))

	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return forwardResult{Error: err.Error(), GateWaitMs: gateWait}
	}
	req.Header.Set("Accept", "application/json")
	if body != "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	for _, h := range extraHeaders {
		if h == "" {
			continue
		}
		if i := strings.Index(h, ":"); i >= 0 {
			k := strings.TrimSpace(h[:i])
			v := strings.TrimSpace(h[i+1:])
			if k != "" {
				req.Header.Set(k, v)
			}
		}
	}
	if gzipBody && body != "" {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write([]byte(body))
		_ = gz.Close()
		req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		req.Header.Set("Content-Encoding", "gzip")
		req.ContentLength = int64(buf.Len())
	}

	resp, err := upstreamClient.Do(req)
	if err != nil {
		return forwardResult{Error: err.Error(), GateWaitMs: gateWait}
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	_ = resp.Body.Close()

	hostIP := tt.ip
	if i := strings.LastIndex(hostIP, ":"); i >= 0 {
		hostIP = hostIP[:i]
	}
	if resp.StatusCode > 0 && hostIP != "" {
		noteResolveSuccess(url, hostIP)
	}

	res := forwardResult{
		Response:    string(raw),
		HTTPCode:    resp.StatusCode,
		CurlTotalMs: int(time.Since(tt.start).Milliseconds()),
		TCPMs:       int(tt.gotConn.Sub(tt.dnsDone).Milliseconds()),
		TLSMs:       int(tt.tlsDone.Sub(tt.gotConn).Milliseconds()),
		ServerMs:    int(tt.firstByte.Sub(tt.tlsDone).Milliseconds()),
		PrimaryIP:   hostIP,
		GateWaitMs:  gateWait,
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    resp.Request.URL.String(),
	}
	if res.TCPMs < 0 {
		res.TCPMs = 0
	}
	if res.TLSMs < 0 {
		res.TLSMs = 0
	}
	if res.ServerMs < 0 {
		res.ServerMs = 0
	}
	return res
}

func forwardToServer(url, body string, gzipBody bool, extraHeaders []string) forwardResult {
	return doForward(http.MethodPost, url, body, gzipBody, extraHeaders, forwardTimeout)
}

func forwardGetToServer(url string, extraHeaders []string) forwardResult {
	return doForward(http.MethodGet, url, "", false, extraHeaders, forwardTimeout)
}

func androidFetchURLWithSession(url, cookies, userAgent string, extraHeaders []string) forwardResult {
	initUpstreamClient()
	tt := &traceTimes{start: time.Now()}
	gateWait := 0
	ok := true
	if fg.enabled() {
		ok, gateWait = fg.acquire()
	}
	if !ok {
		return forwardResult{Error: "系统繁忙，请稍后重试（转发并发已满）", GateWaitMs: gateWait, Busy: true}
	}
	defer fg.release()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxInt(20, forwardTimeout))*time.Second)
	defer cancel()
	ctx = httptrace.WithClientTrace(ctx, newTrace(tt))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return forwardResult{Error: err.Error(), GateWaitMs: gateWait}
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en;q=0.8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://shopee.tw/")
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Linux; Android 15; Mobile) AppleWebKit/537.36 Chrome/150.0.0.0 Mobile Safari/537.36"
	}
	req.Header.Set("User-Agent", userAgent)
	if strings.TrimSpace(cookies) != "" {
		req.Header.Set("Cookie", strings.TrimSpace(cookies))
	}
	for _, h := range extraHeaders {
		if h == "" {
			continue
		}
		if i := strings.Index(h, ":"); i >= 0 {
			req.Header.Set(strings.TrimSpace(h[:i]), strings.TrimSpace(h[i+1:]))
		}
	}

	resp, err := upstreamClient.Do(req)
	if err != nil {
		return forwardResult{Error: err.Error(), GateWaitMs: gateWait}
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	_ = resp.Body.Close()

	hostIP := tt.ip
	if i := strings.LastIndex(hostIP, ":"); i >= 0 {
		hostIP = hostIP[:i]
	}
	return forwardResult{
		Response:    string(raw),
		HTTPCode:    resp.StatusCode,
		CurlTotalMs: int(time.Since(tt.start).Milliseconds()),
		PrimaryIP:   hostIP,
		GateWaitMs:  gateWait,
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    resp.Request.URL.String(),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---- token 缓存 ----
type tokenInfo struct {
	OK        bool
	Token     string
	Cached    bool
	ExpiresAt int64
	Msg       string
	HTTPCode  int
}

var tokenLocks = newSyncMap()

func tokenLockKey(tokenFile string) *sync.Mutex {
	mu, _ := tokenLocks.get(tokenFile)
	if mu == nil {
		m := &sync.Mutex{}
		tokenLocks.set(tokenFile, m)
		return m
	}
	return mu.(*sync.Mutex)
}

// getAPIToken：与 PHP getApi1Token 对齐（进程内单飞 + 文件缓存）
func getAPIToken(auth upstreamAuth, tokenFile string, forceRefresh bool) tokenInfo {
	mu := tokenLockKey(tokenFile)
	mu.Lock()
	defer mu.Unlock()

	if !forceRefresh {
		var cached map[string]any
		if readJSONFile(dp(tokenFile), &cached) {
			tok := trimStr(cached["token"])
			exp := int64(qInt(cached["expires_at"]))
			if tok != "" && exp > time.Now().Unix()+60 {
				return tokenInfo{OK: true, Token: tok, Cached: true, ExpiresAt: exp}
			}
		}
	}

	body := jsonEncode(map[string]string{"username": auth.Username, "password": auth.Password})
	res := forwardToServer(auth.LoginURL, body, false, []string{"Accept: */*"})
	if res.Error != "" {
		return tokenInfo{OK: false, Msg: auth.Label + "登录失败: " + res.Error, HTTPCode: res.HTTPCode}
	}
	var j map[string]any
	if !jsonDecode(res.Response, &j) {
		return tokenInfo{OK: false, Msg: auth.Label + "登录未返回token", HTTPCode: res.HTTPCode}
	}
	data, _ := j["data"].(map[string]any)
	tok := ""
	if data != nil {
		tok = trimStr(data["token"])
	}
	if tok == "" {
		return tokenInfo{OK: false, Msg: auth.Label + "登录未返回token", HTTPCode: res.HTTPCode}
	}
	exp := int64(jwtExp(tok))
	if exp <= 0 {
		exp = time.Now().Unix() + 3600
	}
	_ = atomicWrite(dp(tokenFile), jsonEncodePretty(map[string]any{
		"token": tok, "expires_at": exp, "updated_at": tsDateTime(),
	}))
	cacheAPITokenUsername(tok, auth.Username, exp)
	return tokenInfo{OK: true, Token: tok, Cached: false, ExpiresAt: exp}
}

// ---- token -> username 映射 ----
func loadAPITokenMap() map[string]any {
	return readJSONMap(dp(api1TokenMapFile))
}

func cacheAPITokenUsername(token, username string, expiresAt int64) {
	token = strings.TrimSpace(token)
	username = strings.TrimSpace(username)
	if token == "" || username == "" {
		return
	}
	m := loadAPITokenMap()
	now := time.Now().Unix()
	for k, v := range m {
		entry, ok := v.(map[string]any)
		if !ok {
			delete(m, k)
			continue
		}
		e := int64(qInt(entry["expires_at"]))
		if e > 0 && e+300 < now {
			delete(m, k)
		}
	}
	m[md5hex(token)] = map[string]any{
		"username": username, "expires_at": expiresAt, "updated_at": tsDateTime(),
	}
	// 只保留最近 100 条（map 无序，超出后删除非当前 key 的条目）
	for len(m) > 100 {
		deleted := false
		for k := range m {
			if k != md5hex(token) {
				delete(m, k)
				deleted = true
				break
			}
		}
		if !deleted {
			break
		}
	}
	_ = atomicWrite(dp(api1TokenMapFile), jsonEncodePretty(m))
}

func resolveAPIUsernameFromToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	m := loadAPITokenMap()
	entry, ok := m[md5hex(token)].(map[string]any)
	if !ok {
		return ""
	}
	e := int64(qInt(entry["expires_at"]))
	if e > 0 && e+300 < time.Now().Unix() {
		return ""
	}
	return trimStr(entry["username"])
}

// ---- 上游接口配置 ----
func apiAuthForType(apiType int) upstreamAuth {
	if apiType == 2 {
		return api2Auth
	}
	return api1Auth
}

func apiTypeLabel(apiType int) string {
	if apiType == 2 {
		return "接口2"
	}
	return "接口1"
}

func apiTakeURLForType(apiType int, takeType string) string {
	if strings.EqualFold(strings.TrimSpace(takeType), "r") {
		if apiType == 2 {
			return api2RTakeURL
		}
		return api1RTakeURL
	}
	if apiType == 2 {
		return api2TakeURL
	}
	return api1TakeURL
}

func apiSubmitURLForType(apiType int) string {
	if apiType == 2 {
		return api2SubmitURL
	}
	return api1SubmitURL
}

func apiTokenFileForType(apiType int) string {
	if apiType == 2 {
		return api2TokenFile
	}
	return api1TokenFile
}

func apiAccountsFileForType(apiType int) string {
	if apiType == 2 {
		return api2AccountsFile
	}
	return api1AccountsFile
}

func apiRemovedFileForType(apiType int) string {
	if apiType == 2 {
		return api2RemovedFile
	}
	return api1RemovedFile
}

// ---- 上游成功判断 ----
func api1ForwardSuccess(serverResp map[string]any) bool {
	if serverResp == nil {
		return false
	}
	code := trimStr(serverResp["code"])
	data, hasData := serverResp["data"].(map[string]any)
	dCode := ""
	dMsgIsNull := false
	if hasData {
		dCode = trimStr(data["code"])
		if msg, exists := data["msg"]; exists && msg == nil {
			dMsgIsNull = true
		}
	}
	// 与 PHP api1ForwardSuccess 完全一致：code=200 且 data.code=SUCCESS 且 data.msg 为 null
	return code == "200" && dCode == "SUCCESS" && dMsgIsNull
}

func api1ForwardSuccessForAjie(serverResp map[string]any, httpCode int, raw string) bool {
	if serverResp == nil && strings.TrimSpace(raw) != "" {
		var d map[string]any
		if jsonDecode(raw, &d) {
			serverResp = d
		}
	}
	if serverResp == nil {
		return false
	}
	if asBool(serverResp["success"], false) || asBool(serverResp["ok"], false) {
		return true
	}
	topStatus := toLower(trimStr(serverResp["status"]))
	if topStatus == "ok" || topStatus == "success" || topStatus == "succeed" {
		return true
	}
	topCode := toUpper(trimStr(firstNonNil(serverResp["code"], serverResp["statusCode"], serverResp["status_code"])))
	if topCode == "200" || topCode == "0" || topCode == "SUCCESS" || topCode == "OK" {
		data, hasData := serverResp["data"].(map[string]any)
		if !hasData {
			return true
		}
		dataCode := toUpper(trimStr(firstNonNil(data["code"], data["statusCode"], data["status_code"])))
		dataStatus := toLower(trimStr(data["status"]))
		if dataCode == "" && dataStatus == "" {
			return true
		}
		if dataCode == "200" || dataCode == "0" || dataCode == "SUCCESS" || dataCode == "OK" {
			return true
		}
		if dataStatus == "ok" || dataStatus == "success" || dataStatus == "succeed" {
			return true
		}
	}
	if httpCode >= 200 && httpCode < 300 {
		if topCode != "" || topStatus != "" {
			return false
		}
		if toLower(trimStr(firstNonNil(serverResp["error"], serverResp["reason"]))) == "" {
			data, hasData := serverResp["data"].(map[string]any)
			if !hasData || trimStr(firstNonNil(data["error"], data["reason"])) == "" {
				return true
			}
		}
	}
	return false
}

func api1SubmitDealNotFound(serverResp map[string]any) bool {
	if serverResp == nil {
		return false
	}
	msg := strings.ToLower(trimStr(serverResp["msg"]))
	return strings.Contains(msg, "deal") && (strings.Contains(msg, "not found") || strings.Contains(msg, "notfound") || strings.Contains(msg, "不存在") || strings.Contains(msg, "not_found"))
}

func api1SubmitFailureMessage(serverResp map[string]any, fallback string) string {
	if serverResp == nil {
		if fallback != "" {
			return fallback
		}
		return "upstream submit failed"
	}
	if m := trimStr(serverResp["msg"]); m != "" {
		return m
	}
	if m := trimStr(serverResp["error"]); m != "" {
		return m
	}
	if fallback != "" {
		return fallback
	}
	return "upstream submit failed"
}

// api2ForwardSuccess：PHP 中用于非 1/2 的老接口，这里保留兼容
func api2ForwardSuccess(serverResp map[string]any) bool {
	if serverResp == nil {
		return false
	}
	if asBool(serverResp["ok"], false) || asBool(serverResp["success"], false) {
		return true
	}
	if trimStr(serverResp["code"]) == "200" || trimStr(serverResp["code"]) == "SUCCESS" {
		return true
	}
	return false
}

// rateLimit 判断
func isRateLimitMsg(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "限流") ||
		strings.Contains(msg, "频繁") ||
		strings.Contains(msg, "请求过多")
}

// forwardAttemptSummary 对应 PHP
func forwardAttemptSummary(res forwardResult, serverResp map[string]any) map[string]any {
	m := map[string]any{
		"http_code":    res.HTTPCode,
		"curl_total_ms": res.CurlTotalMs,
		"tcp_ms":       res.TCPMs,
		"tls_ms":       res.TLSMs,
		"server_ms":    res.ServerMs,
		"primary_ip":   res.PrimaryIP,
		"gate_wait_ms": res.GateWaitMs,
		"error":        res.Error,
	}
	if serverResp != nil {
		if msg, ok := serverResp["msg"]; ok {
			m["server_msg"] = msg
		}
		if code, ok := serverResp["code"]; ok {
			m["server_code"] = code
		}
	}
	return m
}

// compactLogResponse 截断日志中的响应
func compactLogResponse(resp any, maxBytes int) any {
	s := ""
	switch v := resp.(type) {
	case string:
		s = v
	case map[string]any:
		s = jsonEncode(v)
	default:
		s = jsonEncode(resp)
	}
	if len(s) > maxBytes {
		return s[:maxBytes] + "..."
	}
	return resp
}

func compactLogResponseStr(resp any, maxBytes int) string {
	switch v := resp.(type) {
	case string:
		if len(v) > maxBytes {
			return v[:maxBytes] + "..."
		}
		return v
	default:
		s := jsonEncode(resp)
		if len(s) > maxBytes {
			return s[:maxBytes] + "..."
		}
		return s
	}
}

func parseInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}
