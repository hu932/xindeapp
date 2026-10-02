package main

import (
	"os"
	"sort"
	"strings"
	"sync"
)

const maxStatRecordsPerDevice = 50

var statsMu sync.Mutex

func getStatsFile(dir, date string) string {
	if date == "" {
		date = tsDate()
	}
	_ = os.MkdirAll(dp(dir), 0o755)
	return dir + "/" + date + ".json"
}

func readDayStats(dir, date string) map[string]any {
	file := getStatsFile(dir, date)
	m := map[string]any{}
	_ = readJSONFile(dp(file), &m)
	return m
}

func fingerprintInfoFromInput(input map[string]any, reqCtx *httpRequestCtx) map[string]any {
	username := trimStr(firstNonNil(input["username"], input["用户名"], "unknown"))
	groupID := trimStr(firstNonNil(input["group_id"], input["组ID"]))
	deviceID := trimStr(firstNonNil(input["device_id"], input["设备ID"]))
	fpKey := trimStr(input["fingerprint_key"])
	fpID := trimStr(input["fingerprint_id"])
	fpName := trimStr(input["fingerprint_name"])
	if fpKey == "" {
		fpKey = fpID
	}
	if fpKey == "" {
		fpKey = fpName
	}
	if fpKey == "" {
		fpKey = deviceID
	}
	if fpKey == "" {
		fpKey = "unknown-" + md5hex(username+"|"+reqCtx.ip)[:10]
	}
	return map[string]any{
		"key": md5hex(username + "|" + fpKey), "fingerprint_key": fpKey, "fingerprint_id": fpID, "fingerprint_name": fpName,
		"username": username, "group_id": groupID, "device_id": deviceID,
	}
}

// recordStats：ENABLE_SUBMIT_STATS 默认关闭，daily_stats 直接跳过（与 PHP 一致）
func recordStats(dir string, input map[string]any, apiType int, status string, reqCtx *httpRequestCtx) {
	if dir == statsDir {
		return // ENABLE_SUBMIT_STATS=false
	}
	statsMu.Lock()
	defer statsMu.Unlock()
	date := tsDate()
	file := getStatsFile(dir, date)
	stats := readDayStats(dir, date)
	fp := fingerprintInfoFromInput(input, reqCtx)
	key := trimStr(fp["key"])
	accRemark := displayAccountRemark(apiType, trimStr(fp["username"]), trimStr(fp["group_id"]))
	entry, _ := stats[key].(map[string]any)
	if entry == nil {
		entry = map[string]any{
			"fingerprint_key": fp["fingerprint_key"], "fingerprint_id": fp["fingerprint_id"], "fingerprint_name": fp["fingerprint_name"],
			"username": fp["username"], "account_remark": accRemark, "group_id": fp["group_id"], "device_id": fp["device_id"],
			"total": 0, "success": 0, "fail": 0, "records": []any{},
		}
		stats[key] = entry
	}
	entry["fingerprint_key"] = fp["fingerprint_key"]
	entry["fingerprint_id"] = fp["fingerprint_id"]
	entry["fingerprint_name"] = fp["fingerprint_name"]
	entry["username"] = fp["username"]
	if accRemark != "" {
		entry["account_remark"] = accRemark
	}
	entry["group_id"] = fp["group_id"]
	entry["device_id"] = fp["device_id"]
	entry["last_seen"] = tsDateTime()
	entry["total"] = qInt(entry["total"]) + 1
	if status == "success" {
		entry["success"] = qInt(entry["success"]) + 1
	} else {
		entry["fail"] = qInt(entry["fail"]) + 1
	}
	record := map[string]any{"time": timeNow().Format("15:04:05"), "api": apiType, "status": status}
	records, _ := entry["records"].([]any)
	records = append(records, record)
	if len(records) > maxStatRecordsPerDevice {
		records = records[len(records)-maxStatRecordsPerDevice:]
	}
	entry["records"] = records
	_ = atomicWrite(dp(file), jsonEncode(stats))
}

func recordFingerprintEnvEvent(input map[string]any, reqCtx *httpRequestCtx) map[string]any {
	event := trimStr(input["event"])
	if event != "generate" && event != "write" {
		return map[string]any{"ok": false, "msg": "未知指纹事件，仅支持 generate/write"}
	}
	statsMu.Lock()
	defer statsMu.Unlock()
	date := tsDate()
	file := getStatsFile(fpEnvStatsDir, date)
	stats := readDayStats(fpEnvStatsDir, date)
	fp := fingerprintInfoFromInput(input, reqCtx)
	key := trimStr(fp["key"])
	accRemark := displayAccountRemark(qInt(input["api_type"]), trimStr(fp["username"]), trimStr(fp["group_id"]))
	entry, _ := stats[key].(map[string]any)
	if entry == nil {
		entry = map[string]any{
			"fingerprint_key": fp["fingerprint_key"], "fingerprint_id": fp["fingerprint_id"], "fingerprint_name": fp["fingerprint_name"],
			"username": fp["username"], "account_remark": accRemark, "group_id": fp["group_id"], "device_id": fp["device_id"],
			"total": 0, "generate": 0, "write": 0, "records": []any{},
		}
		stats[key] = entry
	}
	if _, ok := entry["use"]; ok {
		if _, ok2 := entry["write"]; !ok2 {
			entry["write"] = entry["use"]
			delete(entry, "use")
		}
	}
	entry["fingerprint_key"] = fp["fingerprint_key"]
	entry["fingerprint_id"] = fp["fingerprint_id"]
	entry["fingerprint_name"] = fp["fingerprint_name"]
	entry["username"] = fp["username"]
	if accRemark != "" {
		entry["account_remark"] = accRemark
	}
	entry["group_id"] = fp["group_id"]
	entry["device_id"] = fp["device_id"]
	entry["last_seen"] = tsDateTime()
	entry["total"] = qInt(entry["total"]) + 1
	entry[event] = qInt(entry[event]) + 1
	records, _ := entry["records"].([]any)
	records = append(records, map[string]any{"time": timeNow().Format("15:04:05"), "event": event})
	if len(records) > 200 {
		records = records[len(records)-200:]
	}
	entry["records"] = records
	_ = atomicWrite(dp(file), jsonEncode(stats))
	return map[string]any{"ok": true, "msg": "ok"}
}

func getStatsDateList(dir string, limit int) []string {
	entries, err := os.ReadDir(dp(dir))
	if err != nil {
		return []string{}
	}
	var dates []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			dates = append(dates, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	if limit > 0 && len(dates) > limit {
		dates = dates[:limit]
	}
	return dates
}
