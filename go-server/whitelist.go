package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var whitelistMu sync.Mutex

// readAllowedAccounts：读取 api1/api2 授权账号
func readAllowedAccounts(apiType int) []map[string]any {
	return readAllowedAccountsFromPath(apiAccountsFileForType(apiType))
}

var accountLineSplitRe = regexp.MustCompile(`\r\n|\r|\n`)

func parseAccountLines(raw string) []map[string]any {
	clean := map[string]map[string]any{}
	lines := accountLineSplitRe.Split(raw, -1)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized := strings.ReplaceAll(line, "｜", "|")
		var parts []string
		joiner := " "
		switch {
		case strings.Contains(normalized, "|"):
			parts = strings.Split(normalized, "|")
			joiner = "|"
		case strings.ContainsAny(line, "\t"):
			parts = strings.FieldsFunc(line, func(r rune) bool { return r == '\t' })
		case strings.ContainsAny(line, "，,"):
			parts = strings.FieldsFunc(line, func(r rune) bool { return r == '，' || r == ',' })
			joiner = ","
		case strings.ContainsAny(line, "；;"):
			parts = strings.FieldsFunc(line, func(r rune) bool { return r == '；' || r == ';' })
			joiner = ";"
		default:
			parts = strings.Fields(line)
		}
		if len(parts) == 0 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		remark := strings.Join(parts[1:], joiner)
		clean[name] = map[string]any{"username": name, "remark": strings.TrimSpace(remark)}
	}
	names := make([]string, 0, len(clean))
	for n := range clean {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		out = append(out, clean[n])
	}
	return out
}

func saveAPIAccounts(file string, accounts []map[string]any) map[string]any {
	_ = atomicWrite(dp(file), jsonEncodePretty(map[string]any{
		"accounts": accounts, "updated_at": tsDateTime(),
	}))
	return map[string]any{"ok": true, "data": accounts, "count": len(accounts)}
}

func readRemovedAccounts(file string) []string {
	var arr []string
	_ = readJSONFile(dp(file), &arr)
	var out []string
	for _, x := range arr {
		if s := strings.TrimSpace(x); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func writeRemovedAccounts(file string, names []string) {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		_ = os.Remove(dp(file))
		return
	}
	_ = atomicWrite(dp(file), jsonEncodePretty(out))
}

func saveAPIAllowedAccountsForType(apiType int, raw string) map[string]any {
	file := apiAccountsFileForType(apiType)
	removedFile := apiRemovedFileForType(apiType)
	newAccounts := parseAccountLines(raw)
	newNames := map[string]bool{}
	for _, a := range newAccounts {
		newNames[trimStr(a["username"])] = true
	}
	oldAccounts := readAllowedAccounts(apiType)
	removed := readRemovedAccounts(removedFile)
	// 重新加回
	var kept []string
	for _, r := range removed {
		if !newNames[r] {
			kept = append(kept, r)
		}
	}
	removed = kept
	for _, old := range oldAccounts {
		on := trimStr(old["username"])
		if !newNames[on] && !containsStr(removed, on) {
			removed = append(removed, on)
		}
	}
	writeRemovedAccounts(removedFile, removed)
	return saveAPIAccounts(file, newAccounts)
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func accountNames(accounts []map[string]any) []string {
	var out []string
	for _, a := range accounts {
		out = append(out, trimStr(a["username"]))
	}
	return out
}

func accountRemarkMap(accounts []map[string]any) map[string]string {
	m := map[string]string{}
	for _, a := range accounts {
		m[trimStr(a["username"])] = trimStr(a["remark"])
	}
	return m
}

// ---- 插件任务账号同步到白名单 ----
func decodedExtSyncTaskAccountsToWhitelist(taskAccounts []map[string]any, fallbackRemark, file string) map[string]any {
	allowed := readAllowedAccountsForFile(file)
	removedNames := map[string]bool{}
	for _, r := range readRemovedAccounts(removedAccountsFileForAccountsFile(file)) {
		removedNames[r] = true
	}
	m := map[string]map[string]any{}
	for _, acc := range allowed {
		u := trimStr(acc["username"])
		if u == "" {
			continue
		}
		m[u] = map[string]any{"username": u, "remark": trimStr(acc["remark"])}
	}
	added, updated := 0, 0
	for _, ta := range taskAccounts {
		// 影子账号不进白名单：白名单文件是查号页面的账号来源，
		// 一旦写进去就会在用户界面上暴露。
		if decodedExtIsHiddenAccount(ta) {
			continue
		}
		u := trimStr(ta["username"])
		if u == "" {
			continue
		}
		if removedNames[u] {
			continue
		}
		remark := trimStr(ta["remark"])
		if remark == "" {
			remark = fallbackRemark
		}
		if _, ok := m[u]; !ok {
			m[u] = map[string]any{"username": u, "remark": remark}
			added++
		} else if trimStr(m[u]["remark"]) == "" && remark != "" {
			m[u]["remark"] = remark
			updated++
		}
	}
	if added > 0 || updated > 0 {
		names := make([]string, 0, len(m))
		for n := range m {
			names = append(names, n)
		}
		sort.Strings(names)
		arr := make([]map[string]any, 0, len(names))
		for _, n := range names {
			arr = append(arr, m[n])
		}
		saveAPIAccounts(file, arr)
	}
	return map[string]any{"added": added, "updated": updated, "count": len(m)}
}

// readAllowedAccountsForFile：读取「指定文件」里的授权账号。
// 以前用 mapAPIAccountsFileToType 反查接口类型再读 api1/api2 白名单，导致
// api2_number_remarks.json（接口2 号码备注）和 api1_allowed_accounts_legacy.json
// 都被错读成接口1白名单；这里改成直接读传入的文件。
func readAllowedAccountsForFile(file string) []map[string]any {
	file = trimStr(file)
	if file == "" {
		return readAllowedAccounts(1)
	}
	return readAllowedAccountsFromPath(file)
}

func readAllowedAccountsFromPath(file string) []map[string]any {
	var data map[string]any
	raw := []any{}
	if readJSONFile(dp(file), &data) {
		if arr, ok := data["accounts"].([]any); ok {
			raw = arr
		}
	} else {
		// 兼容旧格式：整个文件直接就是 accounts 数组
		var arr []any
		if readJSONFile(dp(file), &arr) {
			raw = arr
		}
	}
	var out []map[string]any
	for _, acc := range raw {
		switch v := acc.(type) {
		case map[string]any:
			name := trimStr(firstNonNil(v["username"], v["name"]))
			if name == "" {
				continue
			}
			out = append(out, map[string]any{"username": name, "remark": trimStr(v["remark"])})
		case string:
			if s := trimStr(acc); s != "" {
				out = append(out, map[string]any{"username": s, "remark": ""})
			}
		}
	}
	return out
}

func removedAccountsFileForAccountsFile(file string) string {
	if file == api2AccountsFile {
		return api2RemovedFile
	}
	return api1RemovedFile
}

func mapAPIAccountsFileToType(file string) int {
	if file == api2AccountsFile {
		return 2
	}
	return 1
}

func decodedExtSyncAllTaskAccountsToWhitelists() map[string]any {
	total := map[string]any{"added": 0, "updated": 0, "count": 0,
		"api1": map[string]any{"added": 0, "updated": 0, "count": 0},
		"api2": map[string]any{"added": 0, "updated": 0, "count": 0}}
	a1 := map[string]any{}
	a2 := map[string]any{}
	for _, acc := range decodedExtReadAccounts() {
		remark := trimStr(acc["remark"])
		r1 := decodedExtSyncTaskAccountsToWhitelist(decodedExtNormalizeTaskAccounts(acc, false, 1), remark, api1AccountsFile)
		r2 := decodedExtSyncTaskAccountsToWhitelist(decodedExtNormalizeTaskAccounts(acc, false, 2), remark, api2AccountsFile)
		for k := range r1 {
			a1[k] = qInt(a1[k]) + qInt(r1[k])
		}
		for k := range r2 {
			a2[k] = qInt(a2[k]) + qInt(r2[k])
		}
		total["added"] = qInt(total["added"]) + qInt(r1["added"]) + qInt(r2["added"])
		total["updated"] = qInt(total["updated"]) + qInt(r1["updated"]) + qInt(r2["updated"])
	}
	total["api1"] = a1
	total["api2"] = a2
	total["count"] = qInt(a1["count"]) + qInt(a2["count"])
	return total
}

// displayAccountRemark：与 PHP 一致——apiType=1 查接口1备注；apiType=2 组合号码备注+账号备注；其它先 1 后 2
func displayAccountRemark(apiType int, username, groupID string) string {
	if apiType == 2 {
		return api2DisplayRemark(username, groupID)
	}
	if apiType == 1 {
		return accountRemarkMap(readAllowedAccounts(1))[username]
	}
	// apiType == 0 / 其它：先查接口1，没有再走接口2
	if r := accountRemarkMap(readAllowedAccounts(1))[username]; r != "" {
		return r
	}
	return api2DisplayRemark(username, groupID)
}

func api2DisplayRemark(username, groupID string) string {
	parts := []string{}
	numberRemarks := map[string]string{}
	for _, a := range readAllowedAccountsForFile(api2NumbersFile) {
		numberRemarks[trimStr(a["username"])] = trimStr(a["remark"])
	}
	if nr := numberRemarks[username]; nr != "" {
		parts = append(parts, nr)
	}
	accRemarks := accountRemarkMap(readAllowedAccounts(2))
	accountKey := groupID
	if accountKey == "" {
		accountKey = username
	}
	ar := accRemarks[accountKey]
	if ar == "" && accountKey != username {
		ar = accRemarks[username]
	}
	if ar != "" && !containsStr(parts, ar) {
		parts = append(parts, ar)
	}
	return strings.Join(parts, " / ")
}
