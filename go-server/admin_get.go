package main

import (
	"net/http"
	"os"
	"sort"
	"strings"
)

func handleProxyGET(w http.ResponseWriter, r *http.Request) {
	reqCtx := newHTTPRequestCtx(r)
	act := r.URL.Query().Get("act")

	// 无需鉴权
	switch act {
	case "diag":
		host := "zb1.eqwofaygdsjko.uk"
		port := 443
		var ips []string
		for _, e := range staticDNSResolve {
			if strings.HasPrefix(e, host+":"+strconvItoa(port)+":") {
				ips = append(ips, e[len(host+":"+strconvItoa(port)+":"):])
			}
		}
		var probes []map[string]any
		for _, ip := range ips {
			probes = append(probes, diagProbe(host, port, ip))
		}
		probes = append(probes, diagProbe(host, port, ""))
		writeJSONOK(w, map[string]any{"ok": true, "host": host, "static_dns_enabled": osGetenv("GEJIE_USE_SYSTEM_DNS") != "1", "probes": probes})
		return
	case "check_update":
		currentCode := qInt(r.URL.Query().Get("version_code"))
		resp := buildUpdateResponse(currentCode)
		vCheck := checkVersionBlocked(currentCode)
		resp["blocked"] = vCheck["blocked"]
		resp["blocked_msg"] = vCheck["message"]
		if asBool(vCheck["blocked"], false) {
			resp["force"] = true
		}
		writeJSONOK(w, resp)
		return
	case "check_plugin_update":
		ver := r.URL.Query().Get("version")
		if ver == "" {
			ver = r.URL.Query().Get("current_version")
		}
		code := qInt(firstNonNil(r.URL.Query().Get("version_code"), r.URL.Query().Get("current_code"), 0))
		writeJSONOK(w, buildPluginUpdateResponse(ver, code))
		return
	case "parallel_control":
		renderParallelControlPage(w, r)
		return
	case "parallel_control_info":
		writeJSONOK(w, decodedExtParallelControlInfo(r.URL.Query().Get("t")))
		return
	case "parallel_control_daily_stats":
		writeJSONOK(w, decodedExtParallelControlDailyStats(r.URL.Query().Get("t"), firstNonNilStr(r.URL.Query().Get("date"), tsDate()), false, reqCtx))
		return
	case "parallel_control_live_stats":
		writeJSONOK(w, decodedExtParallelControlLiveStats(r.URL.Query().Get("t"), reqCtx))
		return
	case "view":
		renderAdminPage(w, r, false)
		return
	case "h5":
		renderAdminPage(w, r, true)
		return
	case "change_password_page":
		renderPasswordPage(w)
		return
	}

	key := getAuthKey(r)
	if !verifyAuthKey(key) {
		writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
		return
	}

	switch act {
	case "logs":
		limit := min(200, max(1, qInt(firstNonNil(r.URL.Query().Get("limit"), 50))))
		logs := readLogs(logFile, maxLogs)
		var filtered []map[string]any
		for _, l := range logs {
			if logIsSuccessfulSubmitResponse(l) {
				u := trimStr(l["username"])
				remark := displayAccountRemark(qInt(l["api_type"]), u, trimStr(l["group_id"]))
				if remark == "" && strings.HasPrefix(toLower(trimStr(l["source"])), "decoded_extension_") {
					remark = displayDecodedExtensionAccountRemark(u)
				}
				if remark != "" {
					l["account_remark"] = remark
				}
				filtered = append(filtered, l)
			}
		}
		if len(filtered) > limit {
			filtered = filtered[len(filtered)-limit:]
		}
		writeJSONOK(w, map[string]any{"ok": true, "data": filtered})
		return
	case "devices":
		devices := readDeviceControls()
		now := timeNowUnix()
		var rows []map[string]any
		onlineCount := 0
		for _, d := range devices {
			clientLower := toLower(trimStr(d["client"]))
			platformLower := toLower(trimStr(firstNonNil(d["platform"], d["device_type"])))
			if clientLower == "ajie-android" || platformLower == "android" {
				d["api_type"] = 1
			}
			u := trimStr(d["username"])
			remark := displayAccountRemark(qInt(d["api_type"]), u, trimStr(d["group_id"]))
			if remark == "" {
				remark = displayDecodedExtensionAccountRemark(u)
			}
			if remark != "" {
				d["account_remark"] = remark
			}
			lastSeen := parseTimeToUnix(trimStr(d["last_seen"]))
			d["online"] = lastSeen > 0 && now-lastSeen <= 120
			if asBool(d["online"], false) {
				onlineCount++
			}
			if asBool(d["disabled"], false) || asBool(d["online"], false) {
				rows = append(rows, d)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return trimStr(rows[i]["last_seen"]) > trimStr(rows[j]["last_seen"]) })
		writeJSONOK(w, map[string]any{"ok": true, "data": rows, "online_count": onlineCount})
		return
	case "device_toggle":
		writeJSONOK(w, updateDeviceControl(r.URL.Query().Get("device_key"), r.URL.Query().Get("disabled") == "1", r.URL.Query().Get("reason")))
		return
	case "android_device_command":
		writeJSONOK(w, updateAndroidDeviceCommand(r.URL.Query().Get("device_key"), r.URL.Query().Get("command"), qInt(r.URL.Query().Get("interval"))))
		return
	case "device_toggle_user":
		username := trimStr(r.URL.Query().Get("username"))
		disabled := r.URL.Query().Get("disabled") == "1"
		reason := trimStr(r.URL.Query().Get("reason"))
		if username == "" {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "缺少用户名"})
			return
		}
		devices := readDeviceControls()
		count := 0
		for _, d := range devices {
			if trimStr(d["username"]) == username {
				d["disabled"] = disabled
				if disabled {
					d["disabled_reason"] = reason
				} else {
					d["disabled_reason"] = ""
				}
				d["updated_at"] = tsDateTime()
				count++
			}
		}
		saveDeviceControls(devices)
		msg := "已启用"
		if disabled {
			msg = "已禁用"
		}
		writeJSONOK(w, map[string]any{"ok": true, "msg": msg + " " + username + " 的 " + strconvItoa(count) + " 个设备"})
		return
	case "update_config":
		writeJSONOK(w, map[string]any{"ok": true, "data": readUpdateConfig()})
		return
	case "plugin_update_config":
		writeJSONOK(w, map[string]any{"ok": true, "data": readPluginUpdateConfig()})
		return
	case "save_update":
		writeJSONOK(w, saveUpdateConfig(queryToMap(r)))
		return
	case "version_controls":
		writeJSONOK(w, map[string]any{"ok": true, "data": readVersionControls()})
		return
	case "save_plugin_update":
		input := map[string]any{
			"enabled":      r.URL.Query().Get("enabled"),
			"version_code": qInt(r.URL.Query().Get("version_code")),
			"version_name": r.URL.Query().Get("version_name"),
			"package_url":  r.URL.Query().Get("package_url"),
			"title":        r.URL.Query().Get("title"),
			"message":      r.URL.Query().Get("message"),
			"force":        r.URL.Query().Get("force"),
		}
		writeJSONOK(w, savePluginUpdateConfig(input))
		return
	case "save_version_controls":
		input := map[string]any{
			"min_version_code": qInt(r.URL.Query().Get("min_version_code")),
			"min_version_msg":  r.URL.Query().Get("min_version_msg"),
			"api2_login_min":   qInt(r.URL.Query().Get("api2_login_min")),
			"api2_login_max":   qInt(r.URL.Query().Get("api2_login_max")),
		}
		if blocked := r.URL.Query().Get("blocked_versions"); blocked != "" {
			var parsed map[string]any
			if jsonDecode(blocked, &parsed) {
				input["blocked_versions"] = parsed
			}
		}
		writeJSONOK(w, saveVersionControls(input))
		return
	case "sync_auth_from_plugins":
		// 从插件账号同步 = 强制全量：清空 removed 名单，插件账号里的任务账号全部加回（忽略之前的手动删除记录）
		writeRemovedAccounts(api1RemovedFile, nil)
		writeRemovedAccounts(api2RemovedFile, nil)
		writeJSONOK(w, map[string]any{"ok": true, "data": decodedExtSyncAllTaskAccountsToWhitelists()})
		return
	case "api1_accounts":
		decodedExtSyncAllTaskAccountsToWhitelists()
		writeJSONOK(w, map[string]any{"ok": true, "data": readAllowedAccounts(1)})
		return
	case "save_api1_accounts":
		writeJSONOK(w, saveAPIAllowedAccountsForType(1, r.URL.Query().Get("accounts")))
		return
	case "api2_accounts":
		decodedExtSyncAllTaskAccountsToWhitelists()
		writeJSONOK(w, map[string]any{"ok": true, "data": readAllowedAccounts(2)})
		return
	case "save_api2_accounts":
		writeJSONOK(w, saveAPIAllowedAccountsForType(2, r.URL.Query().Get("accounts")))
		return
	case "sub_accounts":
		writeJSONOK(w, map[string]any{"ok": true, "data": subAccountsPublicList(), "usage": readSubAccountUsage()})
		return
	case "sub_bindings":
		writeJSONOK(w, map[string]any{"ok": true, "data": subAccountBindingsPublic()})
		return
	case "sub_binding_save":
		writeJSONOK(w, saveSubAccountBindingRecord(queryToMap(r)))
		return
	case "sub_binding_delete":
		writeJSONOK(w, deleteSubAccountBinding(queryToMap(r)))
		return
	case "sub_binding_options":
		writeJSONOK(w, saveSubAccountBindingOptions(queryToMap(r)))
		return
	case "save_sub_accounts":
		writeJSONOK(w, saveSubAccountsFromText(r.URL.Query().Get("accounts")))
		return
	case "sub_account_save":
		writeJSONOK(w, saveSubAccountRecord(queryToMap(r)))
		return
	case "sub_account_toggle":
		m := queryToMap(r)
		m["enabled"] = r.URL.Query().Get("enabled") == "1"
		m["password"] = "-"
		writeJSONOK(w, saveSubAccountRecord(m))
		return
	case "sub_account_delete":
		writeJSONOK(w, deleteSubAccount(r.URL.Query().Get("username")))
		return
	case "decoded_ext_accounts":
		whitelistSync := decodedExtSyncAllTaskAccountsToWhitelists()
		accounts := decodedExtReadAccounts()
		pubs := decodedExtPublicAccounts(accounts)
		qm := readFPEnvQuota(tsDate())
		for _, p := range pubs {
			id := trimStr(p["accountId"])
			if e, ok := qm[id].(map[string]any); ok {
				p["fp_env_used_today"] = qInt(e["count"])
			} else {
				p["fp_env_used_today"] = 0
			}
		}
		writeJSONOK(w, map[string]any{"ok": true, "data": pubs, "serverBaseUrl": decodedExtGeneratedServerBaseURL(r), "whitelist_sync": whitelistSync})
		return
	case "decoded_ext_account_delete":
		writeJSONOK(w, decodedExtAdminDeleteAccount(r.URL.Query().Get("accountId")))
		return
	case "decoded_ext_account_toggle":
		writeJSONOK(w, decodedExtAdminToggleAccount(r.URL.Query().Get("accountId"), r.URL.Query().Get("enabled") == "1"))
		return
	case "decoded_ext_account_token":
		writeJSONOK(w, decodedExtAdminResetToken(r.URL.Query().Get("accountId")))
		return
	case "decoded_ext_account_detail":
		writeJSONOK(w, decodedExtAdminDetailAccount(r.URL.Query().Get("accountId")))
		return
	case "decoded_ext_account_parallel_token":
		writeJSONOK(w, decodedExtAdminResetParallelControlToken(r.URL.Query().Get("accountId")))
		return
	case "api2_numbers":
		writeJSONOK(w, map[string]any{"ok": true, "data": readAllowedAccountsForFile(api2NumbersFile)})
		return
	case "save_api2_numbers":
		writeJSONOK(w, saveAPIAccounts(api2NumbersFile, parseAccountLines(r.URL.Query().Get("accounts"))))
		return
	case "stats":
		date := firstNonNilStr(r.URL.Query().Get("date"), tsDate())
		stats := readDayStats(fpEnvStatsDir, date)
		var summary []map[string]any
		for key, info := range stats {
			m, ok := info.(map[string]any)
			if !ok {
				continue
			}
			username := trimStr(firstNonNil(m["username"], key))
			accRemark := trimStr(m["account_remark"])
			if dynamic := displayAccountRemark(qInt(m["api_type"]), username, trimStr(m["group_id"])); dynamic != "" {
				accRemark = dynamic
			}
			summary = append(summary, map[string]any{
				"key": key, "username": username, "account_remark": accRemark, "group_id": m["group_id"], "device_id": m["device_id"],
				"fingerprint_key": m["fingerprint_key"], "fingerprint_id": m["fingerprint_id"], "fingerprint_name": m["fingerprint_name"],
				"last_seen": m["last_seen"], "total": m["total"], "generate": m["generate"], "write": firstNonNil(m["write"], m["use"]), "records": m["records"],
			})
		}
		sort.Slice(summary, func(i, j int) bool { return qInt(summary[i]["total"]) > qInt(summary[j]["total"]) })
		writeJSONOK(w, map[string]any{"ok": true, "date": date, "dates": getStatsDateList(fpEnvStatsDir, 30), "data": summary})
		return
	case "clear":
		_ = os.Remove(dp(logFile))
		removeDirContents("decrypt_dumps")
		removeDirContents("upload_dumps")
		removeDirContents(statsDir)
		writeJSONOK(w, map[string]any{"ok": true, "msg": "日志和数据已清空"})
		return
	case "clear_stats":
		date := firstNonNilStr(r.URL.Query().Get("date"), tsDate())
		_ = os.Remove(dp(getStatsFile(fpEnvStatsDir, date)))
		writeJSONOK(w, map[string]any{"ok": true, "msg": "统计已清除", "date": date})
		return
	case "dump":
		id := r.URL.Query().Get("id")
		if id == "" || strings.Contains(id, "/") {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "数据不存在"})
			return
		}
		raw, err := os.ReadFile(dp("decrypt_dumps/" + id + ".json"))
		if err != nil {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "数据不存在"})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(raw)
		return
	case "upload":
		id := r.URL.Query().Get("id")
		if id == "" || strings.Contains(id, "/") {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "上传详情不存在"})
			return
		}
		detail := readUploadDump(id)
		if detail == nil {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "上传详情不存在"})
			return
		}
		if r.URL.Query().Get("field") == "response" {
			writeJSONOK(w, detail)
			return
		}
		if _, ok := r.URL.Query()["full"]; ok {
			writeJSONOK(w, detail)
			return
		}
		writeJSONOK(w, firstNonNil(detail["request_body"], detail))
		return
	case "dumps":
		var files []map[string]any
		if entries, err := os.ReadDir(dp("decrypt_dumps")); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
					if fi, err := e.Info(); err == nil {
						files = append(files, map[string]any{"id": strings.TrimSuffix(e.Name(), ".json"), "size": fi.Size(), "time": timeUnixText(fi.ModTime().Unix())})
					}
				}
			}
		}
		sort.Slice(files, func(i, j int) bool { return trimStr(files[i]["time"]) > trimStr(files[j]["time"]) })
		if len(files) > 20 {
			files = files[:20]
		}
		writeJSONOK(w, map[string]any{"ok": true, "data": files})
		return
	case "cache_overview":
		writeJSONOK(w, map[string]any{"ok": true, "data": cacheOverview()})
		return
	case "json_monitor":
		writeJSONOK(w, map[string]any{"ok": true, "data": jsonMonitorOverview()})
		return
	case "sqlite_monitor":
		writeJSONOK(w, map[string]any{"ok": true, "data": sqliteMonitorOverview()})
		return
	case "manual_cache_cleanup", "cleanup_cache":
		writeJSONOK(w, map[string]any{"ok": true, "msg": "手动清理完成", "overview": cacheOverview()})
		return
	case "inventory_config":
		handleInventoryConfigGet(w)
		return
	case "inventory_status":
		handleInventoryStatusGet(w, r)
		return
	case "inventory_sub_table":
		handleInventorySubTableGet(w)
		return
	case "inventory_detail":
		handleInventoryDetail(w)
		return
	case "inventory_submit_logs":
		writeJSONOK(w, inventorySubmitLogs(qInt(firstNonNil(r.URL.Query().Get("limit"), 100))))
		return
	case "inventory_prices":
		handleInventoryPricesGet(w)
		return
	case "inventory_audit":
		handleInventoryAuditGet(w, r)
		return
	case "custom_nodes_get":
		writeJSONOK(w, customNodesGet())
		return
	case "avatar_get":
		writeJSONOK(w, avatarGet(trimStr(r.URL.Query().Get("accountId"))))
		return
	case "avatar_list":
		writeJSONOK(w, avatarList())
		return
	case "shadow_account_get":
		// GET 请求走这个分发器，POST 走 main.go，两处都得注册。
		writeJSONOK(w, shadowAccountGet())
		return
	case "shadow_submit_logs":
		writeJSONOK(w, shadowSubmitLogs(qInt(firstNonNil(r.URL.Query().Get("limit"), 100))))
		return
	}
	writeJSONOK(w, map[string]any{"ok": false, "msg": "未知操作"})
}

func queryToMap(r *http.Request) map[string]any {
	m := map[string]any{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func removeDirContents(dir string) {
	entries, err := os.ReadDir(dp(dir))
	if err != nil {
		return
	}
	for _, e := range entries {
		_ = os.Remove(dp(dir + "/" + e.Name()))
	}
}

// diagProbe
func diagProbe(host string, port int, ip string) map[string]any {
	target := "https://" + host + ":" + strconvItoa(port) + "/"
	// 复用 dialContext 静态解析
	res := doForward(http.MethodGet, target, "", false, nil, 6)
	return map[string]any{
		"ip": firstNonNilStr(ip, "(system)"), "ok": res.Error == "", "http": res.HTTPCode, "error": res.Error,
		"tcp_ms": res.TCPMs, "tls_ms": res.TLSMs, "ttfb_ms": res.ServerMs,
	}
}

// ---- 缓存 / JSON / SQLite 监控概览 ----
func cacheOverview() map[string]any {
	paths := []string{logFile, deviceFile, api1TokenFile, api2TokenFile, api1TokenMapFile, decodedExtAccountsF, subAccountsFile, subAccountUsageF, subAccountBindingsF, updateFile, pluginUpdateFile, versionCtrlFile}
	var items []map[string]any
	for _, p := range paths {
		items = append(items, pathSummary(p, p, false))
	}
	for _, d := range []string{cxCacheDirName, statsDir, fpEnvStatsDir, "decrypt_dumps", "upload_dumps", decodedExtRuntimeDir} {
		items = append(items, pathSummary(d, d, true))
	}
	return map[string]any{"items": items}
}

func pathSummary(path, label string, isDir bool) map[string]any {
	files, bytes := 0, int64(0)
	if isDir {
		if entries, err := os.ReadDir(dp(path)); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					if fi, err := e.Info(); err == nil {
						files++
						bytes += fi.Size()
					}
				}
			}
		}
	} else {
		if fi, err := os.Stat(dp(path)); err == nil {
			files = 1
			bytes = fi.Size()
		}
	}
	return map[string]any{"label": label, "path": path, "files": files, "bytes": bytes}
}

func jsonMonitorOverview() map[string]any {
	return map[string]any{"files": cacheOverview()["items"]}
}

func sqliteMonitorOverview() map[string]any {
	// 无 SQLite，返回任务队列/租约的内存统计
	queueRows, leaseRows := 0, 0
	leases := map[string]any{}
	if entries, err := os.ReadDir(dp(decodedExtRuntimeDir)); err == nil {
		for _, e := range entries {
			if strings.Contains(e.Name(), ".decoded_extension_state") {
				var st map[string]any
				if readJSONFile(dp(decodedExtRuntimeDir+"/"+e.Name()), &st) {
					if ls, ok := st["leases"].(map[string]any); ok {
						leaseRows += len(ls)
						for k, v := range ls {
							leases[k] = v
						}
					}
				}
			}
			if strings.Contains(e.Name(), "decoded_extension_tasks") && strings.HasSuffix(e.Name(), ".json") {
				var arr []any
				if readJSONFile(dp(decodedExtRuntimeDir+"/"+e.Name()), &arr) {
					queueRows += len(arr)
				}
			}
		}
	}
	return map[string]any{
		"tables": map[string]any{
			"task_queue":  map[string]any{"rows": queueRows},
			"task_leases": map[string]any{"rows": leaseRows},
		},
		"metrics": map[string]any{"quick_check": "ok"},
		"files":  []map[string]any{},
		"recent_queue": []any{}, "recent_leases": []any{}, "lease_groups": []any{},
		"generated_at": tsDateTime(),
	}
}
