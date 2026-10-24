package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// ------------------------------------------------------------------
// 用户头像
//
// 头像按下发 id 存储，直接放进 JSON 里（base64），不落独立文件：
// 数量少、体积小（前端会限制到 200KB 以内），省去静态目录和权限问题。
//
//	POST ?act=avatar_upload   body {accountId, dataUrl}
//	GET  /api/avatar?accountId=xxx   返回图片二进制，便于 <img src> 直接用
// ------------------------------------------------------------------

const avatarFile = "avatars.json"

// 单张头像上限，防止把 JSON 撑爆
const avatarMaxBytes = 256 * 1024

// avatars 读取全部头像。返回 accountId -> dataURL
func avatars() map[string]string {
	out := map[string]string{}
	var data map[string]any
	if !readJSONFile(dp(avatarFile), &data) {
		return out
	}
	rows, _ := data["avatars"].(map[string]any)
	for k, v := range rows {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func saveAvatars(m map[string]string) {
	rows := map[string]any{}
	for k, v := range m {
		rows[k] = v
	}
	_ = atomicWrite(dp(avatarFile), jsonEncodePretty(map[string]any{
		"avatars": rows, "updated_at": tsDateTime(),
	}))
}

// avatarGet 返回单个头像的 dataURL。
func avatarGet(accountID string) map[string]any {
	m := avatars()
	v, ok := m[accountID]
	if !ok || v == "" {
		return map[string]any{"ok": false, "msg": "未设置头像"}
	}
	return map[string]any{"ok": true, "data": map[string]any{"accountId": accountID, "dataUrl": v}}
}

// avatarUpload 保存头像。dataUrl 形如 data:image/png;base64,xxxx
func avatarUpload(input map[string]any) map[string]any {
	accountID := trimStr(firstNonNil(input["accountId"], input["account_id"], input["username"]))
	if accountID == "" {
		return map[string]any{"ok": false, "msg": "缺少 accountId"}
	}
	dataURL := qStr(firstNonNil(input["dataUrl"], input["data_url"], input["avatar"]))
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		// 传空表示删除
		m := avatars()
		delete(m, accountID)
		saveAvatars(m)
		return map[string]any{"ok": true, "msg": "已清除头像"}
	}
	if !strings.HasPrefix(dataURL, "data:image/") {
		return map[string]any{"ok": false, "msg": "格式不对，需要 data:image/...;base64,"}
	}
	idx := strings.Index(dataURL, "base64,")
	if idx < 0 {
		return map[string]any{"ok": false, "msg": "缺少 base64 段"}
	}
	raw := dataURL[idx+7:]
	dec, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return map[string]any{"ok": false, "msg": "base64 解码失败"}
	}
	if len(dec) > avatarMaxBytes {
		return map[string]any{"ok": false, "msg": "图片太大，请压到 256KB 以内"}
	}
	m := avatars()
	m[accountID] = dataURL
	saveAvatars(m)
	return map[string]any{"ok": true, "msg": "已保存", "size": len(dec)}
}

// avatarList 返回所有已设置头像的下发 id，供后台列表对照。
func avatarList() map[string]any {
	m := avatars()
	ids := make([]string, 0, len(m))
	for k := range m {
		ids = append(ids, k)
	}
	return map[string]any{"ok": true, "data": ids, "count": len(ids)}
}

// handleAvatarImage 直接输出图片二进制。
//
//	GET /api/avatar?accountId=xxx
//
// 这样 App 端和网页端都能直接 <img src="...">，不用先拉 JSON 再解码。
func handleAvatarImage(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	accountID := trimStr(r.FormValue("accountId"))
	if accountID == "" {
		accountID = trimStr(r.FormValue("id"))
	}
	if accountID == "" {
		http.NotFound(w, r)
		return
	}
	v, ok := avatars()[accountID]
	if !ok || v == "" {
		http.NotFound(w, r)
		return
	}
	idx := strings.Index(v, "base64,")
	if idx < 0 {
		http.NotFound(w, r)
		return
	}
	mime := "image/png"
	if semi := strings.Index(v, ";"); semi > 5 && semi < idx {
		mime = v[5:semi]
	}
	dec, err := base64.StdEncoding.DecodeString(v[idx+7:])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(dec)
}
