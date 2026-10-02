package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	locAsiaShanghai = mustLoadLocation("Asia/Shanghai")
)

func mustLoadLocation(name string) *time.Location {
	if l, err := time.LoadLocation(name); err == nil {
		return l
	}
	// fallback: fixed UTC+8
	return time.FixedZone("CST", 8*3600)
}

func now() time.Time { return time.Now().In(locAsiaShanghai) }

func timeNow() time.Time { return now() }

func timeParse(layout, s string) (time.Time, error) {
	return time.ParseInLocation(layout, s, locAsiaShanghai)
}

func toLower(s string) string { return strings.ToLower(s) }
func toUpper(s string) string { return strings.ToUpper(s) }

// httpRequestCtx 封装请求上下文（IP 等）
type httpRequestCtx struct {
	r  *http.Request
	ip string
}

func newHTTPRequestCtx(r *http.Request) *httpRequestCtx {
	return &httpRequestCtx{r: r, ip: clientIP(r)}
}

func tsDate() string    { return now().Format("2006-01-02") }
func tsDateTime() string { return now().Format("2006-01-02 15:04:05") }

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func md5short(s string) string { return md5hex(s)[:16] }

func sha256short(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// randomToken 与 PHP decodedExtensionRandomToken 对齐（base64url, 去 =）
func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	s := strings.TrimRight(base64URLEncode(b), "=")
	return s
}

func base64URLEncode(b []byte) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var sb strings.Builder
	for i := 0; i < len(b); i += 3 {
		var n uint32
		var rem int
		for j := 0; j < 3; j++ {
			if i+j < len(b) {
				n = n<<8 | uint32(b[i+j])
				rem++
			}
		}
		sb.WriteByte(tbl[(n>>18)&63])
		sb.WriteByte(tbl[(n>>12)&63])
		if rem >= 2 {
			sb.WriteByte(tbl[(n>>6)&63])
		} else {
			sb.WriteByte('=')
		}
		if rem == 3 {
			sb.WriteByte(tbl[n&63])
		} else {
			sb.WriteByte('=')
		}
	}
	return sb.String()
}

func jsonEncode(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func jsonEncodePretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

func jsonDecode(s string, out any) bool {
	return json.Unmarshal([]byte(s), out) == nil
}

// ---- 原子写（与 PHP atomicWrite 等价）----
func atomicWrite(file string, contents string) bool {
	dir := filepath.Dir(file)
	if dir != "." {
		_ = os.MkdirAll(dir, 0o775)
	}
	tmp := file + "." + randHex(4) + ".tmp"
	if err := os.WriteFile(tmp, []byte(contents), 0o644); err != nil {
		return false
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp)
		return false
	}
	return true
}

// 读 JSON 对象到 out；失败返回 false
func readJSONFile(file string, out any) bool {
	raw, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, out) == nil
}

// 读 JSON 文件为 map
func readJSONMap(file string) map[string]any {
	var m map[string]any
	if readJSONFile(file, &m) {
		return m
	}
	return map[string]any{}
}

// ---- HTTP 响应 ----
type responseWriter struct {
	http.ResponseWriter
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"ok":false,"msg":"json encode error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeJSONOK(w http.ResponseWriter, v any) { writeJSON(w, 200, v) }

// ---- 解析辅助 ----
func qInt(v any) int {
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
	case bool:
		if x {
			return 1
		}
	}
	return 0
}

func qStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

func trimStr(v any) string { return strings.TrimSpace(qStr(v)) }

// asBool 宽松布尔
func asBool(v any, def bool) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off", "":
			return false
		}
	}
	return def
}

// ---- 金额 / 比率 ----
func rateText(part, total int) string {
	if total <= 0 {
		return "0%"
	}
	v := float64(part) / float64(total) * 100
	s := strconv.FormatFloat(v, 'f', 1, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s + "%"
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ---- 同步 map 便捷 ----
type syncMap struct {
	mu sync.RWMutex
	m  map[string]any
}

func newSyncMap() *syncMap { return &syncMap{m: map[string]any{}} }

func (s *syncMap) get(k string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	return v, ok
}

func (s *syncMap) set(k string, v any) {
	s.mu.Lock()
	s.m[k] = v
	s.mu.Unlock()
}

func (s *syncMap) del(k string) {
	s.mu.Lock()
	delete(s.m, k)
	s.mu.Unlock()
}

func (s *syncMap) snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any, len(s.m))
	for k, v := range s.m {
		out[k] = v
	}
	return out
}

// ---- 请求工具 ----
func clientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		if i := strings.Index(ip, ","); i >= 0 {
			ip = ip[:i]
		}
		ip = strings.TrimSpace(ip)
		if ip != "" {
			return ip
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}

func getAuthKey(r *http.Request) string {
	if h := r.Header.Get("X-Auth-Key"); h != "" {
		return h
	}
	return r.URL.Query().Get("key")
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		auth = r.Header.Get("REDIRECT_HTTP_AUTHORIZATION")
	}
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// asRowSlice：把 []any / []map[string]any 统一转为 []map[string]any
func asRowSlice(v any) []map[string]any {
	switch arr := v.(type) {
	case []map[string]any:
		return arr
	case []any:
		out := make([]map[string]any, 0, len(arr))
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func rowsOf(p map[string]any, key string) []map[string]any {
	return asRowSlice(p[key])
}

// deepCopyValue：保留类型的深拷贝（用于内存缓存并发安全）
func deepCopyValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			out[k] = deepCopyValue(vv)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(x))
		for i, m := range x {
			out[i] = deepCopyValue(m).(map[string]any)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			out[i] = deepCopyValue(vv)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(x))
		for k, vv := range x {
			out[k] = vv
		}
		return out
	default:
		return v
	}
}

func filepathGlob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func removeFile(f string) error {
	return os.Remove(f)
}
