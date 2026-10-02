package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	shareDataFile  = "accounts_share_data.php"
	shareDataGuard = "<?php exit; ?>\n"
	shareMaxAcc     = 5000
)

var shareMu sync.Mutex

func shareEmptyStore() map[string]any {
	return map[string]any{"accounts": map[string]any{}, "deleted": map[string]any{}, "rev": 0, "updated_at": ""}
}

func shareReadStore() map[string]any {
	raw, err := os.ReadFile(dp(shareDataFile))
	if err != nil {
		return shareEmptyStore()
	}
	s := string(raw)
	if i := strings.Index(s, "{"); i >= 0 {
		s = s[i:]
	}
	var data map[string]any
	if jsonDecode(s, &data) {
		accounts, _ := data["accounts"].(map[string]any)
		deleted, _ := data["deleted"].(map[string]any)
		if accounts == nil {
			accounts = map[string]any{}
		}
		if deleted == nil {
			deleted = map[string]any{}
		}
		return map[string]any{"accounts": accounts, "deleted": deleted, "rev": qInt(data["rev"]), "updated_at": qStr(data["updated_at"])}
	}
	return shareEmptyStore()
}

func shareSaveStore(store map[string]any) bool {
	store["rev"] = qInt(store["rev"]) + 1
	store["updated_at"] = tsDateTime()
	return atomicWrite(dp(shareDataFile), shareDataGuard+jsonEncode(store))
}

func shareMergeKey(acc map[string]any) string {
	if uid := trimStr(acc["uid"]); uid != "" {
		return "u:" + uid
	}
	if id := trimStr(acc["id"]); id != "" {
		return "i:" + id
	}
	return ""
}

func shareCleanAccount(acc map[string]any, nowMs int64) map[string]any {
	if shareMergeKey(acc) == "" {
		return nil
	}
	cookies, ok := acc["cookies"].([]any)
	if !ok || len(cookies) == 0 {
		return nil
	}
	ts := int64(qInt(acc["updatedAt"]))
	if ts <= 0 {
		ts = nowMs
	}
	usedAt := int64(qInt(acc["usedAt"]))
	return map[string]any{
		"id": qStr(acc["id"]), "uid": trimStr(acc["uid"]), "cookies": cookies, "ua": qStr(acc["ua"]),
		"region": qStr(acc["region"]), "usedAt": condAny(usedAt > 0, usedAt), "updatedAt": ts,
	}
}

func condAny(cond bool, v any) any {
	if cond {
		return v
	}
	return nil
}

func shareIndexAccounts(accounts map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, a := range accounts {
		if m, ok := a.(map[string]any); ok {
			if k := shareMergeKey(m); k != "" {
				out[k] = m
			}
		}
	}
	return out
}

func shareRespond(w http.ResponseWriter, store map[string]any) {
	accounts, _ := store["accounts"].(map[string]any)
	writeJSONOK(w, map[string]any{
		"ok": true, "data": mapValues(accounts), "rev": qInt(store["rev"]), "updated_at": qStr(store["updated_at"]),
	})
}

func mapValues(m map[string]any) []any {
	out := make([]any, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func handleAccountsShare(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	if r.Method == http.MethodGet {
		act := r.URL.Query().Get("act")
		switch act {
		case "ping":
			writeJSONOK(w, map[string]any{"ok": true, "msg": "ok", "time": tsDateTime()})
			return
		case "":
			writeJSONOK(w, map[string]any{"ok": false, "msg": "缺少 act 参数"})
			return
		case "list":
			shareMu.Lock()
			defer shareMu.Unlock()
			shareRespond(w, shareReadStore())
			return
		case "clear":
			shareMu.Lock()
			defer shareMu.Unlock()
			nowMs := time.Now().UnixMilli()
			store := shareEmptyStore()
			for k := range shareIndexAccounts(shareReadStore()["accounts"].(map[string]any)) {
				store["deleted"].(map[string]any)[k] = nowMs
			}
			shareSaveStore(store)
			shareRespond(w, store)
			return
		default:
			writeJSONOK(w, map[string]any{"ok": false, "msg": "未知操作: " + act})
			return
		}
	}

	// POST
	body := readRawJSONBody(r)
	act := trimStr(body["act"])
	nowMs := time.Now().UnixMilli()

	shareMu.Lock()
	defer shareMu.Unlock()

	switch act {
	case "list":
		shareRespond(w, shareReadStore())
		return
	case "clear":
		store := shareEmptyStore()
		for k := range shareIndexAccounts(shareReadStore()["accounts"].(map[string]any)) {
			store["deleted"].(map[string]any)[k] = nowMs
		}
		shareSaveStore(store)
		shareRespond(w, store)
		return
	case "sync":
		store := shareReadStore()
		byKey := shareIndexAccounts(store["accounts"].(map[string]any))
		deleted := store["deleted"].(map[string]any)
		if removed, ok := body["removed"].([]any); ok {
			for _, k := range removed {
				key := trimStr(k)
				if key == "" {
					continue
				}
				delete(byKey, key)
				deleted[key] = nowMs
			}
		}
		incoming, _ := body["accounts"].([]any)
		if len(incoming) > shareMaxAcc {
			incoming = incoming[:shareMaxAcc]
		}
		for _, item := range incoming {
			acc, ok := item.(map[string]any)
			if !ok {
				continue
			}
			clean := shareCleanAccount(acc, nowMs)
			if clean == nil {
				continue
			}
			k := shareMergeKey(clean)
			ts := int64(qInt(clean["updatedAt"]))
			if dv, ok := deleted[k]; ok && int64(qInt(dv)) >= ts {
				continue
			}
			exist, ok := byKey[k]
			if !ok || int64(qInt(exist["updatedAt"])) < ts {
				byKey[k] = clean
			}
		}
		store["accounts"] = mapToAnyMapOfMap(byKey)
		shareSaveStore(store)
		shareRespond(w, store)
		return
	case "delete":
		store := shareReadStore()
		byKey := shareIndexAccounts(store["accounts"].(map[string]any))
		deleted := store["deleted"].(map[string]any)
		if keys, ok := body["keys"].([]any); ok {
			for _, k := range keys {
				key := trimStr(k)
				if key == "" {
					continue
				}
				delete(byKey, key)
				deleted[key] = nowMs
			}
		}
		store["accounts"] = mapToAnyMapOfMap(byKey)
		shareSaveStore(store)
		shareRespond(w, store)
		return
	default:
		writeJSONOK(w, map[string]any{"ok": false, "msg": "未知操作: " + act})
		return
	}
}

func mapToAnyMapOfMap(m map[string]map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
