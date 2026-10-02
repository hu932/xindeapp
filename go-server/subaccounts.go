package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var subAccountMu sync.Mutex

func subAccountDefaultPermissions() map[string]bool {
	return map[string]bool{"take_r": true, "submit": true, "fetch_submit": true, "view_logs": false}
}

func hashEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func normalizeSubAccountPermissions(raw any, base map[string]bool) map[string]bool {
	defaults := subAccountDefaultPermissions()
	perms := map[string]bool{}
	if base != nil {
		for k, v := range base {
			perms[k] = v
		}
	} else {
		for k, v := range defaults {
			perms[k] = v
		}
	}
	for k, d := range defaults {
		if v, ok := perms[k]; ok {
			perms[k] = asBool(v, d)
		}
	}
	if raw == nil {
		return perms
	}
	switch v := raw.(type) {
	case string:
		t := strings.TrimSpace(v)
		if t == "" {
			return perms
		}
		if strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[") {
			var decoded any
			if jsonDecode(t, &decoded) {
				return normalizeSubAccountPermissions(decoded, perms)
			}
		}
		list := splitAnyLower(t)
		all := false
		for _, n := range list {
			if n == "*" || n == "all" {
				all = true
			}
		}
		if all {
			for k := range defaults {
				perms[k] = true
			}
			return perms
		}
		for k := range defaults {
			perms[k] = false
		}
		for _, n := range list {
			if _, ok := defaults[n]; ok {
				perms[n] = true
			}
		}
		return perms
	case []any:
		isList := true
		for i := range v {
			if _, ok := v[i].(string); !ok {
				isList = false
				break
			}
		}
		if isList {
			for k := range defaults {
				perms[k] = false
			}
			for _, item := range v {
				n := strings.ToLower(strings.TrimSpace(qStr(item)))
				if _, ok := defaults[n]; ok {
					perms[n] = true
				}
			}
			return perms
		}
		m := map[string]any{}
		for i, item := range v {
			if i%2 == 0 && i+1 < len(v) {
				m[qStr(item)] = v[i+1]
			}
		}
		// 处理成 map 形式（来自 JSON object 的情况由 map 分支处理）
		return normalizeSubAccountPermissions(m, perms)
	case map[string]any:
		for k, d := range defaults {
			if val, ok := v[k]; ok {
				perms[k] = asBool(val, d)
			}
		}
	}
	return perms
}

func splitAnyLower(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ',' || r == ' ' || r == '|' || r == ';' || r == '，' || r == '；'
	})
}

func readSubAccounts() []map[string]any {
	var data map[string]any
	if !readJSONFile(dp(subAccountsFile), &data) {
		return []map[string]any{}
	}
	var raw []any
	raw, _ = data["accounts"].([]any)
	var out []map[string]any
	for _, a := range raw {
		acc, ok := a.(map[string]any)
		if !ok {
			continue
		}
		username := trimStr(firstNonNil(acc["username"], acc["name"]))
		if username == "" {
			continue
		}
		out = append(out, map[string]any{
			"username":         username,
			"password_hash":    trimStr(acc["password_hash"]),
			"password":         qStr(acc["password"]),
			"enabled":          asBool(acc["enabled"], true),
			"remark":           trimStr(acc["remark"]),
			"permissions":      normalizeSubAccountPermissions(acc["permissions"], nil),
			"daily_take_limit": max(0, qInt(firstNonNil(acc["daily_take_limit"], acc["take_limit"], 0))),
			"bind_device":      trimStr(firstNonNil(acc["bind_device"], acc["device_id"])),
			"created_at":       qStr(acc["created_at"]),
			"updated_at":       qStr(acc["updated_at"]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return trimStr(out[i]["username"]) < trimStr(out[j]["username"]) })
	return out
}

func saveSubAccounts(accounts []map[string]any) map[string]any {
	clean := map[string]map[string]any{}
	for _, acc := range accounts {
		username := trimStr(acc["username"])
		if username == "" {
			continue
		}
		row := map[string]any{
			"username": username,
			"enabled":  asBool(acc["enabled"], true),
			"remark":   trimStr(acc["remark"]),
			"permissions": normalizeSubAccountPermissions(acc["permissions"], nil),
			"daily_take_limit": max(0, qInt(acc["daily_take_limit"])),
			"bind_device":      trimStr(acc["bind_device"]),
			"created_at":       firstNonNilStr(acc["created_at"], tsDateTime()),
			"updated_at":       firstNonNilStr(acc["updated_at"], tsDateTime()),
		}
		if ph := trimStr(acc["password_hash"]); ph != "" {
			row["password_hash"] = ph
		} else if pw := qStr(acc["password"]); pw != "" {
			row["password"] = pw
		}
		clean[username] = row
	}
	names := make([]string, 0, len(clean))
	for n := range clean {
		names = append(names, n)
	}
	sort.Strings(names)
	arr := make([]map[string]any, 0, len(names))
	for _, n := range names {
		arr = append(arr, clean[n])
	}
	ok := atomicWrite(dp(subAccountsFile), jsonEncodePretty(map[string]any{"accounts": arr, "updated_at": tsDateTime()}))
	msg := "子账号已保存"
	if !ok {
		msg = "子账号保存失败"
	}
	return map[string]any{"ok": ok, "data": arr, "count": len(arr), "msg": msg}
}

func subAccountPasswordHash(password string) string {
	if strings.HasPrefix(password, "$2y$") || strings.HasPrefix(password, "$2a$") || strings.HasPrefix(password, "$2b$") || strings.HasPrefix(password, "$argon2") {
		return password
	}
	return passwordHash(password)
}

func subAccountFind(username string) map[string]any {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	for _, a := range readSubAccounts() {
		if trimStr(a["username"]) == username {
			return a
		}
	}
	return nil
}

func subAccountPasswordMatches(account map[string]any, password string) bool {
	if ph := trimStr(account["password_hash"]); ph != "" && passwordVerify(password, ph) {
		return true
	}
	plain := qStr(account["password"])
	return plain != "" && hashEquals(plain, password)
}

func subAccountAllowed(account map[string]any, permission string) bool {
	if permission == "" {
		return true
	}
	perms := normalizeSubAccountPermissions(account["permissions"], nil)
	return asBool(perms[permission], false)
}

func readSubAccountUsage() map[string]any {
	today := tsDate()
	var data map[string]any
	if !readJSONFile(dp(subAccountUsageF), &data) {
		return map[string]any{"date": today, "accounts": map[string]any{}}
	}
	if qStr(data["date"]) != today {
		return map[string]any{"date": today, "accounts": map[string]any{}}
	}
	acc, _ := data["accounts"].(map[string]any)
	if acc == nil {
		acc = map[string]any{}
	}
	return map[string]any{"date": today, "accounts": acc}
}

func saveSubAccountUsage(usage map[string]any) {
	usage["date"] = tsDate()
	if _, ok := usage["accounts"].(map[string]any); !ok {
		usage["accounts"] = map[string]any{}
	}
	_ = atomicWrite(dp(subAccountUsageF), jsonEncodePretty(usage))
}

func subAccountUsageCount(username, metric string) int {
	usage := readSubAccountUsage()
	acc, _ := usage["accounts"].(map[string]any)
	if u, ok := acc[username].(map[string]any); ok {
		return max(0, qInt(u[metric]))
	}
	return 0
}

func incrementSubAccountUsage(username, metric string) int {
	subAccountMu.Lock()
	defer subAccountMu.Unlock()
	usage := readSubAccountUsage()
	acc, _ := usage["accounts"].(map[string]any)
	u, _ := acc[username].(map[string]any)
	if u == nil {
		u = map[string]any{}
		acc[username] = u
	}
	u[metric] = max(0, qInt(u[metric])) + 1
	u["updated_at"] = tsDateTime()
	saveSubAccountUsage(usage)
	return qInt(u[metric])
}

func subAccountUsageSnapshot(account map[string]any) map[string]any {
	username := trimStr(account["username"])
	take := subAccountUsageCount(username, "take_r")
	submit := subAccountUsageCount(username, "submit")
	limit := max(0, qInt(account["daily_take_limit"]))
	remaining := any(nil)
	if limit > 0 {
		remaining = max(0, limit-take)
	}
	return map[string]any{"date": tsDate(), "take_r": take, "submit": submit, "daily_take_limit": limit, "remaining_take_r": remaining}
}

func subAccountPublic(account map[string]any) map[string]any {
	pub := map[string]any{
		"username": trimStr(account["username"]), "enabled": asBool(account["enabled"], true),
		"remark": qStr(account["remark"]), "permissions": normalizeSubAccountPermissions(account["permissions"], nil),
		"daily_take_limit": max(0, qInt(account["daily_take_limit"])), "bind_device": qStr(account["bind_device"]),
		"password_set": trimStr(account["password_hash"]) != "" || qStr(account["password"]) != "",
		"created_at": qStr(account["created_at"]), "updated_at": qStr(account["updated_at"]),
	}
	pub["usage_today"] = subAccountUsageSnapshot(account)
	return pub
}

func subAccountIdentityKeys(input map[string]any) []map[string]any {
	var items []map[string]any
	seen := map[string]bool{}
	add := func(t, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		key := t + ":" + v
		if seen[key] {
			return
		}
		seen[key] = true
		items = append(items, map[string]any{"key": key, "type": t, "value": v})
	}
	username := trimStr(firstNonNil(input["username"], input["accountId"], input["account_id"], input["id"]))
	ul := strings.ToLower(username)
	if username != "" && ul != "android" && ul != "unknown" && ul != "unknown-user" {
		add("username", username)
	}
	for field, t := range map[string]string{
		"app_user": "app_user", "app_user_id": "app_user", "user_id": "app_user", "customer_id": "customer",
		"customer": "customer", "client_user": "client_user", "channel_user": "channel_user", "group_id": "group",
		"account_id": "account", "tenant_id": "tenant", "invite_code": "invite", "bind_code": "bind_code", "user_key": "user_key",
	} {
		if v, ok := input[field]; ok {
			add(t, qStr(v))
		}
	}
	add("device", qStr(input["device_id"]))
	add("fingerprint", qStr(input["fingerprint_key"]))
	add("fingerprint_id", qStr(input["fingerprint_id"]))
	return items
}

func readSubAccountBindings() map[string]any {
	data := map[string]any{}
	_ = readJSONFile(dp(subAccountBindingsF), &data)
	if _, ok := data["bindings"].(map[string]any); !ok {
		data["bindings"] = map[string]any{}
	}
	if _, ok := data["auto_assign"]; !ok {
		data["auto_assign"] = false
	}
	if _, ok := data["strategy"]; !ok {
		data["strategy"] = "round_robin"
	}
	if _, ok := data["last_index"]; !ok {
		data["last_index"] = -1
	}
	return data
}

func saveSubAccountBindings(data map[string]any) map[string]any {
	data["auto_assign"] = asBool(data["auto_assign"], false)
	if trimStr(data["strategy"]) == "" {
		data["strategy"] = "round_robin"
	}
	data["last_index"] = qInt(data["last_index"])
	if _, ok := data["bindings"].(map[string]any); !ok {
		data["bindings"] = map[string]any{}
	}
	data["updated_at"] = tsDateTime()
	ok := atomicWrite(dp(subAccountBindingsF), jsonEncodePretty(data))
	msg := "绑定已保存"
	if !ok {
		msg = "绑定保存失败"
	}
	return map[string]any{"ok": ok, "msg": msg, "data": data}
}

func subAccountTakeLimitReached(account map[string]any) bool {
	limit := max(0, qInt(account["daily_take_limit"]))
	if limit <= 0 {
		return false
	}
	return subAccountUsageCount(trimStr(account["username"]), "take_r") >= limit
}

func subAccountUsableForPermission(account map[string]any, permission string) bool {
	if !asBool(account["enabled"], true) {
		return false
	}
	if permission != "" && !subAccountAllowed(account, permission) {
		return false
	}
	if permission == "take_r" && subAccountTakeLimitReached(account) {
		return false
	}
	return true
}

func subAccountBindingKeyFromInput(input map[string]any) string {
	key := trimStr(firstNonNil(input["key"], input["binding_key"]))
	if key != "" && strings.Contains(key, ":") {
		return key
	}
	t := trimStr(firstNonNil(input["match_type"], input["type"], "device"))
	v := trimStr(firstNonNil(input["match_value"], input["value"]))
	if v == "" {
		return ""
	}
	return t + ":" + v
}

func subAccountBindingsPublic() map[string]any {
	data := readSubAccountBindings()
	var bindings []map[string]any
	bmap, _ := data["bindings"].(map[string]any)
	for key, e := range bmap {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		subUsername := trimStr(entry["sub_username"])
		account := subAccountFind(subUsername)
		parts := strings.SplitN(key, ":", 2)
		matchType, matchValue := "", ""
		if len(parts) >= 1 {
			matchType = parts[0]
		}
		if len(parts) >= 2 {
			matchValue = parts[1]
		}
		bindings = append(bindings, map[string]any{
			"key": key, "match_type": matchType, "match_value": matchValue, "sub_username": subUsername,
			"enabled": asBool(entry["enabled"], true), "remark": qStr(entry["remark"]),
			"account_enabled": account != nil && asBool(account["enabled"], true),
			"account_exists":  account != nil,
			"created_at": qStr(entry["created_at"]), "updated_at": qStr(entry["updated_at"]),
		})
	}
	sort.Slice(bindings, func(i, j int) bool { return trimStr(bindings[i]["key"]) < trimStr(bindings[j]["key"]) })
	return map[string]any{
		"auto_assign": asBool(data["auto_assign"], false), "strategy": qStr(data["strategy"]),
		"last_index": qInt(data["last_index"]), "bindings": bindings, "updated_at": qStr(data["updated_at"]),
	}
}

func saveSubAccountBindingRecord(input map[string]any) map[string]any {
	key := subAccountBindingKeyFromInput(input)
	if key == "" {
		return map[string]any{"ok": false, "msg": "缺少绑定 key 或 match_type/match_value"}
	}
	subUsername := trimStr(firstNonNil(input["sub_username"], input["username"]))
	if subUsername == "" {
		return map[string]any{"ok": false, "msg": "缺少 sub_username"}
	}
	if subAccountFind(subUsername) == nil {
		return map[string]any{"ok": false, "msg": "子账号不存在: " + subUsername}
	}
	data := readSubAccountBindings()
	bmap, _ := data["bindings"].(map[string]any)
	var old map[string]any
	if bmap[key] != nil {
		old, _ = bmap[key].(map[string]any)
	}
	if old == nil {
		old = map[string]any{}
	}
	bmap[key] = map[string]any{
		"sub_username": subUsername,
		"enabled":      asBool(input["enabled"], asBool(old["enabled"], true)),
		"remark":       trimStr(firstNonNil(input["remark"], old["remark"])),
		"created_at":   firstNonNilStr(old["created_at"], tsDateTime()),
		"updated_at":   tsDateTime(),
	}
	if _, ok := input["auto_assign"]; ok {
		data["auto_assign"] = asBool(input["auto_assign"], true)
	}
	saved := saveSubAccountBindings(data)
	saved["binding"] = subAccountBindingsPublic()
	return saved
}

func deleteSubAccountBinding(input map[string]any) map[string]any {
	key := subAccountBindingKeyFromInput(input)
	if key == "" {
		return map[string]any{"ok": false, "msg": "缺少绑定 key 或 match_type/match_value"}
	}
	data := readSubAccountBindings()
	bmap, _ := data["bindings"].(map[string]any)
	if bmap[key] == nil {
		return map[string]any{"ok": false, "msg": "绑定不存在"}
	}
	delete(bmap, key)
	return saveSubAccountBindings(data)
}

func saveSubAccountBindingOptions(input map[string]any) map[string]any {
	data := readSubAccountBindings()
	if _, ok := input["auto_assign"]; ok {
		data["auto_assign"] = asBool(input["auto_assign"], true)
	}
	if _, ok := input["strategy"]; ok {
		if s := trimStr(input["strategy"]); s != "" {
			data["strategy"] = s
		}
	}
	return saveSubAccountBindings(data)
}

// ---- 子账号 token ----
func readSubAccountTokens() map[string]any {
	var data map[string]any
	_ = readJSONFile(dp(subAccountTokensF), &data)
	tokens, _ := data["tokens"].(map[string]any)
	if tokens == nil {
		tokens = map[string]any{}
	}
	return tokens
}

func saveSubAccountTokens(tokens map[string]any) {
	_ = atomicWrite(dp(subAccountTokensF), jsonEncodePretty(map[string]any{"tokens": tokens, "updated_at": tsDateTime()}))
}

func subAccountTokenFromRequest(input map[string]any, r *http.Request) string {
	token := trimStr(firstNonNil(input["sub_token"], input["local_token"]))
	if token == "" {
		maybe := trimStr(input["token"])
		if strings.HasPrefix(maybe, "sub_") {
			token = maybe
		}
	}
	if token == "" {
		bearer := bearerToken(r)
		if strings.HasPrefix(bearer, "sub_") {
			token = bearer
		}
	}
	return token
}

func issueSubAccountToken(username string, input map[string]any, r *httpRequestCtx, ttl int) map[string]any {
	token := "sub_" + randHex(32)
	h := sha256hex(token)
	tokens := readSubAccountTokens()
	now := timeNowUnix()
	for k, e := range tokens {
		entry, ok := e.(map[string]any)
		if !ok || int64(qInt(entry["expires_at"])) < now {
			delete(tokens, k)
		}
	}
	tokens[h] = map[string]any{
		"username": username, "device_id": trimStr(input["device_id"]), "ip": r.ip,
		"created_at": tsDateTime(), "expires_at": now + int64(ttl),
	}
	saveSubAccountTokens(tokens)
	return map[string]any{"token": token, "expires_at": now + int64(ttl), "expires_at_text": timeUnixText(now + int64(ttl))}
}

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func timeUnixText(ts int64) string {
	return timeNow().Add(time.Duration(ts-timeNowUnix()) * time.Second).Format("2006-01-02 15:04:05")
}

func resolveSubAccountToken(token string) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return map[string]any{"ok": false, "code": "401", "msg": "缺少子账号token"}
	}
	tokens := readSubAccountTokens()
	h := sha256hex(token)
	entry, _ := tokens[h].(map[string]any)
	if entry == nil {
		return map[string]any{"ok": false, "code": "401", "msg": "子账号token无效"}
	}
	if int64(qInt(entry["expires_at"])) < timeNowUnix() {
		delete(tokens, h)
		saveSubAccountTokens(tokens)
		return map[string]any{"ok": false, "code": "401", "msg": "子账号token已过期"}
	}
	return map[string]any{"ok": true, "username": qStr(entry["username"]), "token_entry": entry}
}

// requireSubAccount 与 PHP 对齐
func requireSubAccount(input map[string]any, permission string, r *httpRequestCtx) map[string]any {
	authSource := "password"
	token := subAccountTokenFromRequest(input, r.r)
	var username string
	var account map[string]any
	if token != "" {
		resolved := resolveSubAccountToken(token)
		if !asBool(resolved["ok"], false) {
			resolved["auth_source"] = "token"
			return resolved
		}
		username = trimStr(resolved["username"])
		account = subAccountFind(username)
		authSource = "token"
	} else {
		username = trimStr(firstNonNil(input["sub_username"], input["username"]))
		password := qStr(firstNonNil(input["sub_password"], input["password"]))
		if username == "" || password == "" {
			return map[string]any{"ok": false, "code": "401", "msg": "子账号或密码为空", "auth_source": "password"}
		}
		account = subAccountFind(username)
		if account == nil || !subAccountPasswordMatches(account, password) {
			return map[string]any{"ok": false, "code": "401", "msg": "子账号或密码错误", "auth_source": "password", "username": username}
		}
	}
	if account == nil {
		return map[string]any{"ok": false, "code": "401", "msg": "子账号不存在", "auth_source": authSource, "username": username}
	}
	if !asBool(account["enabled"], true) {
		return map[string]any{"ok": false, "code": "403", "msg": "子账号已禁用", "auth_source": authSource, "username": trimStr(account["username"])}
	}
	if permission != "" && !subAccountAllowed(account, permission) {
		return map[string]any{"ok": false, "code": "403", "msg": "子账号无此权限: " + permission, "auth_source": authSource, "username": trimStr(account["username"])}
	}
	bindDevice := trimStr(account["bind_device"])
	deviceID := trimStr(input["device_id"])
	if bindDevice != "" && deviceID != "" && !hashEquals(bindDevice, deviceID) {
		return map[string]any{"ok": false, "code": "403", "msg": "子账号绑定设备不匹配", "auth_source": authSource, "username": trimStr(account["username"])}
	}
	if bindDevice != "" && deviceID == "" {
		return map[string]any{"ok": false, "code": "403", "msg": "此子账号需要提交绑定设备ID", "auth_source": authSource, "username": trimStr(account["username"])}
	}
	dc := checkDeviceAllowed(map[string]any{"username": trimStr(account["username"]), "api_type": 1, "platform": firstNonNil(input["platform"], "sub")}, r)
	if !asBool(dc["allowed"], true) {
		return map[string]any{"ok": false, "code": "403", "msg": firstNonNilStr(dc["msg"], "设备已被禁用"), "auth_source": authSource, "username": trimStr(account["username"])}
	}
	return map[string]any{"ok": true, "account": account, "username": trimStr(account["username"]), "auth_source": authSource, "device_key": dc["device_key"]}
}

func subAccountErrorResponse(auth map[string]any) map[string]any {
	return map[string]any{"ok": false, "success": false, "code": firstNonNilStr(auth["code"], "403"), "data": nil, "msg": firstNonNilStr(auth["msg"], "子账号认证失败")}
}

func subAccountBuildRecord(input map[string]any, old map[string]any) map[string]any {
	username := trimStr(firstNonNil(input["username"], input["sub_username"], old["username"]))
	if username == "" {
		return map[string]any{"ok": false, "msg": "缺少子账号用户名"}
	}
	nowT := tsDateTime()
	record := map[string]any{
		"username": username, "enabled": asBool(input["enabled"], asBool(old["enabled"], true)),
		"remark": trimStr(firstNonNil(input["remark"], old["remark"])),
		"permissions": normalizeSubAccountPermissions(old["permissions"], nil),
		"daily_take_limit": max(0, qInt(firstNonNil(input["daily_take_limit"], input["take_limit"], old["daily_take_limit"], 0))),
		"bind_device": trimStr(firstNonNil(input["bind_device"], input["device_bind"], old["bind_device"])),
		"created_at": firstNonNilStr(old["created_at"], nowT), "updated_at": nowT,
	}
	if _, ok := input["permissions"]; ok {
		record["permissions"] = normalizeSubAccountPermissions(input["permissions"], normalizeSubAccountPermissions(old["permissions"], nil))
	} else {
		permInput := map[string]any{}
		for p := range subAccountDefaultPermissions() {
			if v, ok := input[p]; ok {
				permInput[p] = v
			}
		}
		if len(permInput) > 0 {
			record["permissions"] = normalizeSubAccountPermissions(permInput, normalizeSubAccountPermissions(old["permissions"], nil))
		}
	}
	if ph := trimStr(input["password_hash"]); ph != "" {
		record["password_hash"] = ph
	} else if pw := qStr(input["password"]); pw != "" && pw != "-" {
		h := subAccountPasswordHash(pw)
		if h == "" {
			return map[string]any{"ok": false, "msg": "子账号密码哈希生成失败"}
		}
		record["password_hash"] = h
	} else if trimStr(old["password_hash"]) != "" {
		record["password_hash"] = trimStr(old["password_hash"])
	} else if qStr(old["password"]) != "" {
		record["password"] = qStr(old["password"])
	} else {
		return map[string]any{"ok": false, "msg": "新子账号必须设置 password"}
	}
	return map[string]any{"ok": true, "record": record}
}

func saveSubAccountRecord(input map[string]any) map[string]any {
	subAccountMu.Lock()
	defer subAccountMu.Unlock()
	accounts := readSubAccounts()
	m := map[string]map[string]any{}
	for _, a := range accounts {
		m[trimStr(a["username"])] = a
	}
	username := trimStr(firstNonNil(input["username"], input["sub_username"]))
	old := m[username]
	if old == nil {
		old = map[string]any{}
	}
	built := subAccountBuildRecord(input, old)
	if !asBool(built["ok"], true) {
		return built
	}
	record, _ := built["record"].(map[string]any)
	m[trimStr(record["username"])] = record
	arr := make([]map[string]any, 0, len(m))
	for _, a := range m {
		arr = append(arr, a)
	}
	saved := saveSubAccounts(arr)
	saved["account"] = subAccountPublic(record)
	return saved
}

func saveSubAccountsFromText(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{"ok": false, "msg": "子账号配置为空"}
	}
	oldMap := map[string]map[string]any{}
	for _, a := range readSubAccounts() {
		oldMap[trimStr(a["username"])] = a
	}
	var rows []map[string]any
	var j map[string]any
	if jsonDecode(raw, &j) {
		if arr, ok := j["accounts"].([]any); ok {
			for _, r := range arr {
				if m, ok := r.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		} else if arr, ok2 := j["accounts"]; ok2 {
			_ = arr
		}
	} else {
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.ReplaceAll(line, "｜", "|")
			var parts []string
			if strings.Contains(line, "|") {
				parts = strings.Split(line, "|")
			} else if strings.Contains(line, "\t") {
				parts = strings.FieldsFunc(line, func(r rune) bool { return r == '\t' })
			} else {
				parts = strings.Split(line, "|")
			}
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			get := func(i int) string {
				if i < len(parts) {
					return parts[i]
				}
				return ""
			}
			rows = append(rows, map[string]any{
				"username": get(0), "password": get(1), "remark": get(2), "enabled": get(3),
				"permissions": get(4), "daily_take_limit": get(5), "bind_device": get(6),
			})
		}
	}
	newMap := map[string]map[string]any{}
	for _, row := range rows {
		username := trimStr(firstNonNil(row["username"], row["sub_username"]))
		if username == "" {
			continue
		}
		built := subAccountBuildRecord(row, oldMap[username])
		if !asBool(built["ok"], true) {
			built["username"] = username
			return built
		}
		newMap[username] = built["record"].(map[string]any)
	}
	arr := make([]map[string]any, 0, len(newMap))
	for _, a := range newMap {
		arr = append(arr, a)
	}
	return saveSubAccounts(arr)
}

func deleteSubAccount(username string) map[string]any {
	subAccountMu.Lock()
	defer subAccountMu.Unlock()
	username = strings.TrimSpace(username)
	if username == "" {
		return map[string]any{"ok": false, "msg": "缺少子账号用户名"}
	}
	var keep []map[string]any
	deleted := false
	for _, a := range readSubAccounts() {
		if trimStr(a["username"]) == username {
			deleted = true
			continue
		}
		keep = append(keep, a)
	}
	if !deleted {
		return map[string]any{"ok": false, "msg": "子账号不存在"}
	}
	return saveSubAccounts(keep)
}
