package main

import (
	"net/http"
	"os"
	"sort"
	"strings"
)

var cxLinkAccountLabels = map[string]string{"532": "Ajie"}

func cxLinksGroups() []map[string]any {
	accounts := cxViewReadAccountsMap()
	counts := map[string]int{}
	for _, remark := range accounts {
		remark = strings.TrimSpace(remark)
		if remark == "" {
			continue
		}
		counts[remark]++
	}
	var groups []map[string]any
	for accountID := range cxLinkAccountLabels {
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, remark := range keys {
			groups = append(groups, map[string]any{"account": accountID, "remark": remark, "count": counts[remark], "remarks": []string{remark}})
		}
	}
	return groups
}

func cxLinksWrite(links []map[string]any) {
	_ = atomicWrite(dp(remarkLinksFile), jsonEncodePretty(map[string]any{"links": links}))
}

func cxResolveCxViewURL(r *http.Request) string {
	host := r.Host
	if host == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil || toLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
		scheme = "https"
	}
	dir := r.URL.Path
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[:i]
	}
	dir = strings.TrimRight(dir, "/.")
	return scheme + "://" + host + dir + "/cx_view.php"
}

func cxLinkPublic(link map[string]any, base string) map[string]any {
	token := trimStr(link["token"])
	remark := trimStr(link["remark"])
	var remarks []string
	if raw, ok := link["remarks"].([]any); ok && len(raw) > 0 {
		for _, rr := range raw {
			for _, part := range strings.Split(qStr(rr), "+") {
				part = strings.TrimSpace(part)
				if part != "" && !containsStr(remarks, part) {
					remarks = append(remarks, part)
				}
			}
		}
	} else if remark != "" {
		remarks = []string{remark}
	}
	url := ""
	if base != "" {
		url = base + "?t=" + urlQueryEscape(token)
	}
	return map[string]any{
		"token": token, "account": trimStr(link["account"]), "remark": remark, "remarks": remarks,
		"enabled": asBool(link["enabled"], true), "created_at": trimStr(link["created_at"]), "url": url,
	}
}

func handleCxLinks(w http.ResponseWriter, r *http.Request) {
	reqCtx := newHTTPRequestCtx(r)
	act := r.URL.Query().Get("act")
	if act != "" {
		key := getAuthKey(r)
		if !verifyAuthKey(key) {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "密码错误"})
			return
		}
		switch act {
		case "groups":
			writeJSONOK(w, map[string]any{"ok": true, "data": cxLinksGroups()})
		case "links":
			base := cxResolveCxViewURL(r)
			var out []map[string]any
			for _, l := range cxViewReadLinks() {
				out = append(out, cxLinkPublic(l, base))
			}
			writeJSONOK(w, map[string]any{"ok": true, "data": out})
		case "create", "create_all", "toggle", "delete":
			input := readRawJSONBody(r)
			handleCxLinkMutate(w, r, reqCtx, act, input)
		default:
			writeJSONOK(w, map[string]any{"ok": false, "msg": "未知操作"})
		}
		return
	}
	// HTML 后台
	writeHTML(w, cxLinksHTML(r))
}

func handleCxLinkMutate(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, act string, input map[string]any) {
	base := cxResolveCxViewURL(r)
	links := cxViewReadLinks()
	switch act {
	case "create":
		account := trimStr(input["account"])
		remark := trimStr(input["remark"])
		if remark == "" {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "参数错误"})
			return
		}
		if _, ok := cxLinkAccountLabels[account]; !ok {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "参数错误"})
			return
		}
		remarks := []string{remark}
		for _, g := range cxLinksGroups() {
			if trimStr(g["account"]) == account && trimStr(g["remark"]) == remark {
				if raw, ok := g["remarks"].([]string); ok && len(raw) > 0 {
					remarks = raw
				}
				break
			}
		}
		for _, l := range links {
			if trimStr(l["account"]) == account && trimStr(l["remark"]) == remark {
				writeJSONOK(w, map[string]any{"ok": true, "exists": true, "data": cxLinkPublic(l, base)})
				return
			}
		}
		link := map[string]any{"token": randHex(16), "account": account, "remark": remark, "remarks": remarks, "enabled": true, "created_at": tsDateTime()}
		links = append(links, link)
		cxLinksWrite(links)
		writeJSONOK(w, map[string]any{"ok": true, "exists": false, "data": cxLinkPublic(link, base)})
	case "create_all":
		existing := map[string]bool{}
		for _, l := range links {
			existing[trimStr(l["account"])+"|"+trimStr(l["remark"])] = true
		}
		created := 0
		for _, g := range cxLinksGroups() {
			k := trimStr(g["account"]) + "|" + trimStr(g["remark"])
			if existing[k] {
				continue
			}
			remarks := []string{trimStr(g["remark"])}
			if raw, ok := g["remarks"].([]string); ok && len(raw) > 0 {
				remarks = raw
			}
			links = append(links, map[string]any{"token": randHex(16), "account": g["account"], "remark": g["remark"], "remarks": remarks, "enabled": true, "created_at": tsDateTime()})
			created++
		}
		cxLinksWrite(links)
		writeJSONOK(w, map[string]any{"ok": true, "created": created})
	case "toggle":
		token := trimStr(input["token"])
		enabled := qInt(input["enabled"]) == 1
		if token == "" {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "缺少 token"})
			return
		}
		found := false
		for _, l := range links {
			if trimStr(l["token"]) == token {
				l["enabled"] = enabled
				found = true
			}
		}
		if !found {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "链接不存在"})
			return
		}
		cxLinksWrite(links)
		writeJSONOK(w, map[string]any{"ok": true})
	case "delete":
		token := trimStr(input["token"])
		if token == "" {
			writeJSONOK(w, map[string]any{"ok": false, "msg": "缺少 token"})
			return
		}
		var keep []map[string]any
		for _, l := range links {
			if trimStr(l["token"]) != token {
				keep = append(keep, l)
			}
		}
		cxLinksWrite(keep)
		writeJSONOK(w, map[string]any{"ok": true})
	}
}

func cxLinksHTML(r *http.Request) string {
	authHashExists := verifyAuthKeyFileExists()
	notice := ""
	if !authHashExists {
		notice = `<div class="notice">未找到 .auth_hash，请先在 decrypt_proxy.php 后台设置/修改一次密码。</div>`
	}
	return `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>备注专属查询链接</title><style>
:root{--bg:#07111f;--surface:#0f1d33;--surface-2:#13243c;--line:rgba(148,163,184,.16);--text:#f2f5f8;--muted:#93a1b5;--accent:#22c55e;--accent-2:#38bdf8;--danger:#fb7185}
*{box-sizing:border-box}body{margin:0;min-height:100vh;color:var(--text);font-family:"Segoe UI Variable","Microsoft YaHei UI",sans-serif;background:#07111f}
.wrap{max-width:1120px;margin:0 auto;padding:32px 20px 60px}h1{font-size:24px;margin:0 0 6px}.sub{color:var(--muted);font-size:13px;margin-bottom:24px}
.card{background:var(--surface);border:1px solid var(--line);border-radius:10px;padding:20px;margin-bottom:20px}.card h2{margin:0 0 14px;font-size:17px}
.row{display:flex;gap:10px;flex-wrap:wrap;align-items:center}label{color:var(--muted);font-size:13px}
select,input[type=password],input[type=text]{height:40px;border:1px solid var(--line);border-radius:8px;background:var(--surface-2);color:var(--text);font:inherit;font-size:14px;padding:0 12px;min-width:220px}
.btn{height:40px;padding:0 16px;border:1px solid var(--line);border-radius:8px;background:var(--surface-2);color:var(--text);font:inherit;font-size:14px;cursor:pointer}
.btn.primary{background:linear-gradient(135deg,var(--accent),var(--accent-2));color:#05110f;font-weight:800;border-color:transparent}
.btn.danger{color:#fecdd3;border-color:rgba(251,113,133,.3)}.btn.sm{height:32px;padding:0 10px;font-size:13px}
table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:12px 10px;border-bottom:1px solid rgba(148,163,184,.1);font-size:14px}
th{color:#6f7e96;font-size:11px;text-transform:uppercase;letter-spacing:.12em}td .tag{display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px}
.tag.on{background:rgba(52,211,153,.14);color:#6ee7b7}.tag.off{background:rgba(251,113,133,.14);color:#fda4af}
.muted{color:var(--muted);font-size:13px}.empty{padding:32px;text-align:center;color:var(--muted)}
.notice{border:1px solid rgba(251,191,36,.24);background:rgba(251,191,36,.08);color:#f5d487;border-radius:8px;padding:12px 14px;font-size:13px;margin-bottom:16px}
.hidden{display:none}</style></head><body><div class="wrap">
<div id="loginView" class="card">
<h1>备注专属查询链接</h1><div class="sub">管理员登录（与 decrypt_proxy 后台密码相同）</div>
` + notice + `
<input type="password" id="loginPwd" placeholder="请输入后台密码" autocomplete="current-password" style="width:100%">
<div style="height:12px"></div>
<button class="btn primary" style="width:100%;justify-content:center" onclick="doLogin()">登录</button>
</div>
<div id="adminView" class="hidden">
<div style="display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap;margin-bottom:20px">
<div><h1>备注专属查询链接</h1><div class="sub">为每个“备注用户”生成专属查询链接。</div></div>
<button class="btn" onclick="doLogout()">退出登录</button></div>
<div class="card"><h2>生成专属链接</h2><div class="row">
<label>账号</label><select id="fAccount" onchange="loadGroups()"><option value="532">Ajie</option></select>
<label>备注用户</label><select id="fRemark"><option value="">加载中...</option></select>
<button class="btn primary" onclick="createLink()">生成链接</button>
<button class="btn" onclick="createAll()">一键生成全部备注链接</button></div></div>
<div class="card"><h2>已生成链接</h2><div id="linkTable"><div class="empty">加载中...</div></div></div>
</div></div>
<script>
const BASE=location.pathname;let KEY=localStorage.getItem('cx_links_key')||'';
function esc(s){return String(s??'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
function api(act,opts){opts=opts||{};if(!opts.headers)opts.headers={};opts.headers['X-Auth-Key']=KEY;
if(opts.body&&typeof opts.body==='object'){opts.headers['Content-Type']='application/json';opts.body=JSON.stringify(opts.body);}
return fetch(BASE+'?act='+act,opts).then(r=>r.json());}
function doLogin(){KEY=document.getElementById('loginPwd').value.trim();api('links').then(d=>{if(d.ok===false){alert(d.msg||'登录失败');return;}localStorage.setItem('cx_links_key',KEY);showAdmin();})}
function doLogout(){localStorage.removeItem('cx_links_key');KEY='';document.getElementById('adminView').classList.add('hidden');document.getElementById('loginView').classList.remove('hidden');}
function showAdmin(){document.getElementById('loginView').classList.add('hidden');document.getElementById('adminView').classList.remove('hidden');loadGroups();loadLinks();}
function loadGroups(account){api('groups').then(d=>{account=account||document.getElementById('fAccount').value;const seen={};const opts=(d.data||[]).filter(function(g){if(String(g.account)!==String(account))return false;if(seen[g.remark])return false;seen[g.remark]=1;return true;}).map(g=>'<option value="'+esc(g.remark)+'">'+esc(g.remark)+'（'+g.count+'）</option>').join('');document.getElementById('fRemark').innerHTML=opts||'<option value="">暂无</option>';});}
function createLink(){const account=document.getElementById('fAccount').value,remark=document.getElementById('fRemark').value;if(!remark)return alert('请选择备注');api('create',{method:'POST',body:{account:account,remark:remark}}).then(d=>{if(d.ok&&d.data&&d.data.url){copyText(d.data.url);alert('链接已生成并复制');}else if(d.exists){copyText(d.data.url);alert('已存在，已复制链接');}else alert(d.msg||'失败');loadLinks();});}
function createAll(){if(!confirm('确定生成全部备注链接？'))return;api('create_all',{method:'POST',body:{}}).then(d=>{alert('已生成 '+d.created+' 条');loadLinks();});}
function loadLinks(){api('links').then(d=>{const el=document.getElementById('linkTable');const rows=d.data||[];if(!rows.length){el.innerHTML='<div class="empty">暂无链接</div>';return;}
el.innerHTML='<table><thead><tr><th>账号</th><th>备注</th><th>状态</th><th>链接</th><th>操作</th></tr></thead><tbody>'+rows.map(l=>'<tr><td>'+esc(l.account)+'</td><td>'+esc(l.remark)+'</td><td><span class="tag '+(l.enabled?'on':'off')+'">'+(l.enabled?'启用':'停用')+'</span></td><td><input type="text" value="'+esc(l.url)+'" style="min-width:280px" readonly></td><td><button class="btn sm" onclick="copyText(\''+esc(l.url)+'\')">复制</button> <button class="btn sm" onclick="toggleLink(\''+esc(l.token)+'\','+(l.enabled?0:1)+')">'+(l.enabled?'停用':'启用')+'</button> <button class="btn sm danger" onclick="delLink(\''+esc(l.token)+'\')">删除</button></td></tr>').join('')+'</tbody></table>';});}
function toggleLink(token,enabled){api('toggle',{method:'POST',body:{token:token,enabled:enabled}}).then(()=>loadLinks());}
function delLink(token){if(!confirm('删除该链接？'))return;api('delete',{method:'POST',body:{token:token}}).then(()=>loadLinks());}
function copyText(t){if(navigator.clipboard){navigator.clipboard.writeText(t);}else{const i=document.createElement('input');i.value=t;document.body.appendChild(i);i.select();document.execCommand('copy');i.remove();}}
if(KEY){api('links').then(d=>{if(d.ok===false)doLogout();else showAdmin();}).catch(()=>{});}
</script></body></html>`
}

func verifyAuthKeyFileExists() bool {
	raw, err := os.ReadFile(dp(authHashFile))
	return err == nil && strings.TrimSpace(string(raw)) != ""
}
