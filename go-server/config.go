package main

import (
	"os"
	"path/filepath"
)

// 数据目录：默认二进制所在目录；可用 -data 参数覆盖。
// 所有数据文件与旧 PHP 版本保持完全相同的文件名/格式，直接迁移旧数据目录即可。
var dataDir = "."

func initDataDir() {
	if d := os.Getenv("GEJIE_DATA_DIR"); d != "" {
		dataDir = d
	}
}

func dp(rel string) string { return filepath.Join(dataDir, rel) }

// ---- 密钥 / 上游目标 ----
const (
	aesKey            = "25d9a1907ab38e273c4f57c476e64377" // 32 字节 key，与 PHP openssl aes-256-ecb 一致（直接当字符串用，不 hex 解码）
	api1Base          = "https://zb1.eqwofaygdsjko.uk:443/api"
	api2Base          = "https://zb2.eqwofaygdsjko.uk:443/api"
	api1TakeURL       = "https://zb1.eqwofaygdsjko.uk:443/api/task/take"
	api1RTakeURL      = "https://zb1.eqwofaygdsjko.uk:443/api/task/take?type=r"
	api2TakeURL       = "https://zb2.eqwofaygdsjko.uk:443/api/task/take"
	api2RTakeURL      = "https://zb2.eqwofaygdsjko.uk:443/api/task/take?type=r"
	api1SubmitURL     = "https://zb1.eqwofaygdsjko.uk:443/api/task/submit/v2"
	api2SubmitURL     = "https://zb2.eqwofaygdsjko.uk:443/api/task/submit/v2"
	api1LoginURL      = "https://zb1.eqwofaygdsjko.uk:443/api/user/login"
	api2LoginURL      = "https://zb2.eqwofaygdsjko.uk:443/api/user/login"
	forwardTimeout    = 12
	forwardConnTimout  = 2
	loginTimeout      = 8
	loginConnTimeout  = 6
	maxLogs           = 500
	maxLogRespBytes   = 8192
	api1ParseMaxRetry = 2
)

// ---- 文件 ----
const (
	authHashFile        = ".auth_hash"
	logFile             = "decrypt_proxy_logs.jsonl"
	deviceFile          = "device_controls.json"
	androidSessionFile  = "android_sessions.json"
	updateFile          = "app_update.json"
	pluginUpdateFile    = "plugin_update.json"
	versionCtrlFile     = "version_controls.json"
	apkDir              = "app_updates"
	pluginUpdateDir     = "plugin_updates"
	statsDir            = "daily_stats"
	fpEnvStatsDir       = "fingerprint_env_stats"
	api1TokenFile       = "api1_token.json"
	api2TokenFile       = "api2_token.json"
	api1TokenMapFile    = "api1_token_map.json"
	api1AccountsFile    = "api1_allowed_accounts.json"
	api2AccountsFile    = "api2_allowed_accounts.json"
	api1RemovedFile     = ".api1_removed_accounts.json"
	api2RemovedFile     = ".api2_removed_accounts.json"
	subAccountsFile     = "sub_accounts.json"
	subAccountTokensF   = "sub_account_tokens.json"
	subAccountUsageF    = "sub_account_usage.json"
	subAccountBindingsF = "sub_account_bindings.json"
	api2NumbersFile     = "api2_number_remarks.json"
	resolvePrefFile     = ".resolve_pref.json"

	decodedExtRuntimeDir  = "decoded_extension_runtime"
	decodedExtTaskQueueF  = "decoded_extension_runtime/decoded_extension_tasks.json"
	decodedExtSubmitLogF  = "decoded_extension_runtime/decoded_extension_submits.jsonl"
	decodedExtAccountsF   = ".decoded_extension_accounts.json"
	decodedExtStateF      = "decoded_extension_runtime/.decoded_extension_state.json"
	decodedExtQueueTTL    = 600
	decodedExtLeaseTTL    = 1800
	parallelControlBaseF  = "parallel_control_base_url.local.php"
	webPluginAuthLocalF   = "web_plugin_auth.local.php"

	remarkLinksFile = "cx_remark_links.json"
	teamcPricesFile = "cx_teamc_prices.json"
	cxCacheDirName  = "cx_cache"
)

// ---- 上游账号（接口1 / 接口2 管理员）----
type upstreamAuth struct {
	LoginURL string
	Username string
	Password string
	Label    string
	APIType  int
}

var api1Auth = upstreamAuth{api1LoginURL, "poisu906", "dfd$d7641HETD2af", "接口1", 1}
var api2Auth = upstreamAuth{api2LoginURL, "tpadsf531", "a1q$r6gw9Fl52aP", "接口2", 2}

// ---- 静态 DNS ----
var staticDNSResolve = []string{
	"zb1.eqwofaygdsjko.uk:443:104.21.65.145",
	"zb1.eqwofaygdsjko.uk:443:172.67.164.25",
}

// ---- 转发闸门默认（可用环境变量覆盖）----
var (
	forwardMaxConcurrency = 30
	forwardRatePerSec     = 0
	forwardGateMaxWaitMs  = 500
	submitGzip            = false
)

func initGateConfig() {
	if v := envInt("GEJIE_FORWARD_MAX_CONCURRENCY", 0); v >= 0 {
		if v > 0 || os.Getenv("GEJIE_FORWARD_MAX_CONCURRENCY") != "" {
			forwardMaxConcurrency = v
		}
	}
	if v := envInt("GEJIE_FORWARD_RATE_LIMIT_PER_SEC", -1); v >= 0 {
		forwardRatePerSec = v
	}
	if v := envInt("GEJIE_FORWARD_GATE_MAX_WAIT_MS", -1); v >= 0 {
		forwardGateMaxWaitMs = v
	}
	submitGzip = os.Getenv("GEJIE_SUBMIT_GZIP") == "1"
}

func envInt(k string, def int) int {
	s := os.Getenv(k)
	if s == "" {
		return def
	}
	n := 0
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		return -n
	}
	return n
}
