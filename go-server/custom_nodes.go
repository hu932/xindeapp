package main

import (
	"strings"
)

// ------------------------------------------------------------------
// 自定义节点
//
// 公开源抓来的节点质量不可控，真正稳定的是自己搭的。
// 这里允许在后台粘贴分享链接，刷新时和抓取结果合并，
// 并且在节点列表里标出来源，便于优先选它们。
//
// 支持的链接格式与抓取逻辑一致：
//   vless://  vmess://  trojan://  ss://  ssr://  hysteria2://
// ------------------------------------------------------------------

const customNodesFile = "custom_nodes.json"

// customNodeLinks 读取用户填写的分享链接。
func customNodeLinks() []string {
	var data map[string]any
	if !readJSONFile(dp(customNodesFile), &data) {
		return nil
	}
	rows, _ := data["links"].([]any)
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		s := trimStr(r)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// customNodes 把链接解析成 vpnNode，来源标记为「自定义」。
func customNodes() []vpnNode {
	links := customNodeLinks()
	if len(links) == 0 {
		return nil
	}
	// 用换行拼起来复用 parseSub 的解析逻辑
	content := strings.Join(links, "\n")
	nodes := parseSub(content, "自定义")
	for i := range nodes {
		nodes[i].Source = "自定义"
		// 名字里没地区就靠 IP 反查补
		if nodes[i].Region == "" || nodes[i].Region == "未知" {
			nodes[i].Region = guessRegion(nodes[i].Name)
		}
	}
	return nodes
}

// customNodesGet 后台读取。
func customNodesGet() map[string]any {
	links := customNodeLinks()
	return map[string]any{
		"ok":    true,
		"data":  strings.Join(links, "\n"),
		"count": len(links),
	}
}

// customNodesSave 后台保存。逐行一个链接，空行忽略。
func customNodesSave(input map[string]any) map[string]any {
	raw := qStr(firstNonNil(input["links"], input["text"]))
	var kept []string
	var bad []string
	for _, line := range strings.Split(raw, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
			continue
		}
		// 只收认得出来的协议
		ok := false
		for _, p := range []string{"vless://", "vmess://", "trojan://", "ss://", "ssr://", "hysteria2://", "hy2://"} {
			if strings.HasPrefix(s, p) {
				ok = true
				break
			}
		}
		if !ok {
			bad = append(bad, s)
			continue
		}
		kept = append(kept, s)
	}

	rows := make([]any, 0, len(kept))
	for _, s := range kept {
		rows = append(rows, s)
	}
	_ = atomicWrite(dp(customNodesFile), jsonEncodePretty(map[string]any{
		"links": rows, "updated_at": tsDateTime(),
	}))

	msg := "已保存 " + qStr(len(kept)) + " 个自定义节点"
	if len(bad) > 0 {
		msg += "，" + qStr(len(bad)) + " 行格式无法识别已忽略"
	}
	return map[string]any{
		"ok": true, "msg": msg, "count": len(kept), "rejected": len(bad),
	}
}
