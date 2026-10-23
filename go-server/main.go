package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var fg *ForwardGate

func main() {
	listen := flag.String("listen", ":2036", "监听地址，例如 :2036")
	data := flag.String("data", "", "数据目录（默认二进制所在目录）")
	flag.Parse()

	if *data != "" {
		dataDir = *data
	} else {
		initDataDir()
	}
	initGateConfig()
	fg = newForwardGate()
	initUpstreamClient()
	_ = os.MkdirAll(dp(decodedExtRuntimeDir), 0o775)
	_ = os.MkdirAll(dp(cxCacheDirName), 0o775)

	mux := http.NewServeMux()

	// 代理 / 后台入口
	mux.HandleFunc("/decrypt_proxy.php", proxyHandler)
	mux.HandleFunc("/g.php", proxyHandler)

	// 解码插件 API 路由（/api/*）
	// 同时注册旧的 /decrypt_proxy.php/api/* 与 /g.php/api/* 形式：
	// 旧 PHP 后台的 config.json 里 serverBaseUrl 带 decrypt_proxy.php 后缀，
	// 直接迁移过来的配置也能正常调用，避免 404。
	for _, p := range []string{"/api/login", "/api/task", "/api/submit", "/api/heartbeat", "/api/fp_env/generate"} {
		apiPath := p
		mux.HandleFunc(apiPath, func(w http.ResponseWriter, r *http.Request) {
			handleDecodedExtensionAPI(w, r)
		})
		for _, prefix := range []string{"/decrypt_proxy.php", "/g.php"} {
			mux.HandleFunc(prefix+apiPath, func(w http.ResponseWriter, r *http.Request) {
				handleDecodedExtensionAPI(w, r)
			})
		}
	}

	// 任务库存模式（新入口）
	mux.HandleFunc("/inventory/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleInventoryTask(w, r)
			return
		}
		http.Error(w, "method not allowed", 405)
	})
	mux.HandleFunc("/inventory/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleInventorySubmit(w, r)
			return
		}
		http.Error(w, "method not allowed", 405)
	})
	// 模式检测（App/插件登录前调用，无需鉴权）：库存模式开关
	mux.HandleFunc("/inventory/mode", func(w http.ResponseWriter, r *http.Request) {
		cfg := invLoadConfig()
		writeJSONOK(w, map[string]any{"code": 200, "success": true, "enabled": cfg.Enabled, "mode": condStr(cfg.Enabled, "inventory", "normal")})
	})

	// 查询页
	mux.HandleFunc("/cx.php", handleCxPage)
	mux.HandleFunc("/cx2.php", handleCx2Page)
	mux.HandleFunc("/cx_view.php", handleCxView)
	mux.HandleFunc("/cx_links.php", handleCxLinks)
	mux.HandleFunc("/accounts_share.php", handleAccountsShare)
	mux.HandleFunc("/load_monitor.php", handleLoadMonitor)
	mux.HandleFunc("/cx_cron.php", handleCxCron)

	// 平行控制漂亮路由
	for _, p := range []string{"/control", "/parallel_control", "/parallel-control", "/concurrency"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("t") != "" {
				renderParallelControlPage(w, r)
				return
			}
			http.NotFound(w, r)
		})
	}

	// 静态资源
	mux.HandleFunc("/app_updates/", func(w http.ResponseWriter, r *http.Request) {
		serveStatic(w, r, dp(apkDir))
	})
	mux.HandleFunc("/plugin_updates/", func(w http.ResponseWriter, r *http.Request) {
		serveStatic(w, r, dp(pluginUpdateDir))
	})

	// 免费节点聚合：抓取 / 测速 / 排行榜
	mux.HandleFunc("/api/vpn/nodes", handleVPNNodes)
	mux.HandleFunc("/api/vpn/refresh", handleVPNRefresh)
	for _, prefix := range []string{"/g.php", "/decrypt_proxy.php"} {
		p := prefix
		mux.HandleFunc(p+"/api/vpn/nodes", handleVPNNodes)
		mux.HandleFunc(p+"/api/vpn/refresh", handleVPNRefresh)
	}

	startInventoryManager()
	startCxAutoRefresh()
	startVPNRefresher()
	// 根路由：兼容 pretty route + 代理
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			proxyHandler(w, r)
			return
		}
		// 静态文件（logo 等）
		if strings.HasPrefix(path, "/1777990327149137925.png") || strings.HasPrefix(path, "/bj.jpeg") {
			serveStatic(w, r, dataDir)
			return
		}
		http.NotFound(w, r)
	})

	srv := &http.Server{Addr: *listen, Handler: mux}
	log.Printf("Ajie 后台服务已启动，监听 %s，数据目录 %s", *listen, dataDir)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleProxyGET(w, r)
	case http.MethodPost:
		// 先尝试解码插件 API
		if handleDecodedExtensionAPI(w, r) {
			return
		}
		// multipart 表单（上传 / 改密）
		if r.URL.Query().Get("act") == "upload_update" || r.URL.Query().Get("act") == "change_password" || r.URL.Query().Get("act") == "upload_plugin_update" {
			handleProxyFormPost(w, r)
			return
		}
		// 带 act 查询参数的 POST（解码插件账号 / 授权账号 / 平行控制）
		if handleProxyQueryActPOST(w, r) {
			return
		}
		handleProxyPOST(w, r)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleProxyFormPost：multipart 表单 POST（上传 APK / 插件包 / 改密）
func handleProxyFormPost(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	key := getAuthKey(r)
	if !verifyAuthKey(key) {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
		return
	}
	switch act {
	case "change_password":
		writeJSONOK(w, changeAuthPassword(r.FormValue("current_password"), r.FormValue("new_password"), r.FormValue("confirm_password")))
	case "upload_update":
		_ = r.ParseMultipartForm(256 << 20)
		input := map[string]any{
			"version_code": qInt(r.FormValue("version_code")), "version_name": r.FormValue("version_name"),
			"apk_url": r.FormValue("apk_url"), "title": r.FormValue("title"), "message": r.FormValue("message"),
			"force": r.FormValue("force"),
		}
		writeJSONOK(w, uploadUpdateAPK(r, input))
	case "upload_plugin_update":
		_ = r.ParseMultipartForm(256 << 20)
		input := map[string]any{
			"version_code": qInt(r.FormValue("version_code")), "version_name": r.FormValue("version_name"),
			"package_url": r.FormValue("package_url"), "title": r.FormValue("title"), "message": r.FormValue("message"),
			"force": r.FormValue("force"), "enabled": r.FormValue("enabled"),
		}
		writeJSONOK(w, uploadPluginUpdatePackage(r, input))
	default:
		writeJSONOK(w, map[string]any{"ok": false, "msg": "未知操作"})
	}
}

func serveStatic(w http.ResponseWriter, r *http.Request, root string) {
	name := filepath.Base(r.URL.Path)
	http.ServeFile(w, r, filepath.Join(root, name))
}

// handleProxyQueryActPOST：带 ?act= 的 POST（解码插件账号 / 授权账号 / 平行控制）
func handleProxyQueryActPOST(w http.ResponseWriter, r *http.Request) bool {
	act := r.URL.Query().Get("act")
	switch {
	case strings.HasPrefix(act, "decoded_ext_account_"):
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		input := readRawJSONBody(r)
		if act == "decoded_ext_account_save" {
			writeJSONOK(w, decodedExtAdminSaveAccount(input))
			return true
		}
		if act == "decoded_ext_account_batch" {
			writeJSONOK(w, decodedExtAdminBatchAccounts(qStr(input["batch"])))
			return true
		}
		writeJSONOK(w, map[string]any{"ok": false, "msg": "unknown decoded extension account action"})
		return true
	case act == "shadow_account_get":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		writeJSONOK(w, shadowAccountGet())
		return true
	case act == "shadow_account_save":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		writeJSONOK(w, shadowAccountSave(readRawJSONBody(r)))
		return true
	case act == "save_api1_accounts" || act == "save_api2_accounts":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		input := readRawJSONBody(r)
		apiType := 1
		if act == "save_api2_accounts" {
			apiType = 2
		}
		writeJSONOK(w, saveAPIAllowedAccountsForType(apiType, qStr(firstNonNil(input["accounts"], r.FormValue("accounts")))))
		return true
	case act == "parallel_control_save":
		input := readRawJSONBody(r)
		token := trimStr(firstNonNil(input["t"], r.URL.Query().Get("t")))
		writeJSONOK(w, decodedExtParallelControlSave(token,
			firstNonNil(input["task_parallelism"], input["parallelism"]),
			firstNonNil(input["mixed_task_enabled"], input["mixedTaskEnabled"], input["api1_fallback_api2"], input["api1FallbackApi2"]),
			firstNonNil(input["api1_task_parallelism"], input["api1Parallelism"], input["login_parallelism"]),
			firstNonNil(input["api2_task_parallelism"], input["api2Parallelism"], input["nonlogin_parallelism"]),
			firstNonNil(input["mixed_task_web_enabled"], input["mixedTaskWebEnabled"], input["web_plugin_mixed_task_enabled"]),
			firstNonNil(input["mixed_task_ios_enabled"], input["mixedTaskIosEnabled"], input["ios_plugin_mixed_task_enabled"]),
			firstNonNil(input["task_mode"], input["taskMode"])))
		return true
	case act == "inventory_save":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		handleInventorySave(w, r)
		return true
	case act == "inventory_sub_table_save":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		handleInventorySubTableSave(w, r)
		return true
	case act == "inventory_prices_save":
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return true
		}
		handleInventoryPricesSave(w, r)
		return true
	}
	return false
}
