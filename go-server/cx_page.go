package main

import (
	"fmt"
	"net/http"
	"strings"
)

func handleCxPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	apiType := qInt(firstNonNil(q.Get("api_type"), q.Get("api"), q.Get("interface"), q.Get("source"), 1))
	cxPageRespond(w, r, "cx.php", apiType)
}

// handleCx2Page：接口2（zb2）专用查询页，功能与 cx.php 一致，固定走接口2。
func handleCx2Page(w http.ResponseWriter, r *http.Request) {
	cxPageRespond(w, r, "cx2.php", 2)
}

func cxPageRespond(w http.ResponseWriter, r *http.Request, fileName string, apiType int) {
	reqCtx := newHTTPRequestCtx(r)
	q := r.URL.Query()
	if q.Get("act") == "clear_cache" {
		result := cxClearCache()
		if q.Get("json") == "1" || toLower(r.Header.Get("X-Requested-With")) == "fetch" {
			writeJSONOK(w, result)
			return
		}
		writeHTML(w, `<!doctype html><meta charset="utf-8"><title>缓存清理</title><body style="font-family:sans-serif;padding:32px"><h2>缓存已清理</h2><p>删除文件：`+strconvItoa(qInt(result["deleted"]))+` 个</p></body>`)
		return
	}
	if apiType != 2 {
		apiType = 1
	}
	date := trimStr(q.Get("date"))
	autoRefresh := q.Get("auto") == "1"
	forceRefresh := q.Get("force") == "1" || q.Get("refresh") == "1"
	formatJSON := q.Get("format") == "json" || toLower(r.Header.Get("X-Requested-With")) == "fetch"

	payload := CxQuery(fileName, apiType, date, autoRefresh, forceRefresh, reqCtx)

	if formatJSON {
		writeJSONOK(w, payload)
		return
	}
	writeHTML(w, cxPageHTML(fileName, payload, apiType))
}

func cxPageHTML(fileName string, p map[string]any, apiType int) string {
	title := "Ajie 查询"
	if fileName == "cx1.php" {
		title = "团队C 查询"
	}
	if fileName == "cx2.php" {
		title = "接口2 查询"
	}
	date := qStr(p["date"])
	byRemark, _ := p["byRemark"].([]map[string]any)
	batchItems := rowsOf(p, "batchItems")
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + title + `</title>
<style>:root{--bg:#0f172a;--card:#1e293b;--line:#334155;--text:#e2e8f0;--muted:#94a3b8;--green:#34d399;--red:#f87171;--blue:#818cf8;--amber:#fbbf24}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:-apple-system,"Segoe UI","PingFang SC",sans-serif;font-size:14px}
.wrap{max-width:1200px;margin:0 auto;padding:20px 14px}.head{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px;margin-bottom:16px}h1{font-size:22px;margin:0}
.row{display:flex;gap:10px;flex-wrap:wrap;align-items:center}select,input{background:#0f172a;border:1px solid var(--line);border-radius:8px;color:var(--text);padding:8px 12px;font:inherit}
.btn{background:#334155;border:1px solid var(--line);border-radius:8px;color:var(--text);padding:8px 14px;cursor:pointer;font:inherit}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:12px;margin-bottom:16px}
.stat{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px}.stat .l{color:var(--muted);font-size:12px}.stat .v{font-size:26px;font-weight:800;margin-top:6px}.stat .v.green{color:var(--green)}.stat .v.red{color:var(--red)}.stat .v.blue{color:var(--blue)}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px;margin-bottom:16px}.card h2{margin:0 0 12px;font-size:16px}
table{width:100%;border-collapse:collapse;font-size:13px}th,td{padding:9px 8px;border-bottom:1px solid var(--line);text-align:center}th{color:var(--muted);font-size:11px;text-transform:uppercase}
td.l{text-align:left}.tag{display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px}.tag.ok{background:rgba(52,211,153,.15);color:var(--green)}.tag.bad{background:rgba(248,113,113,.1);color:var(--red)}
.warn{border:1px solid rgba(251,191,36,.3);background:rgba(251,191,36,.08);color:var(--amber);border-radius:8px;padding:10px 12px;font-size:13px;margin-bottom:16px}
.muted{color:var(--muted);font-size:12px}.amt{color:var(--blue);font-weight:700}
</style></head><body><div class="wrap">
<div class="head"><h1>` + title + `</h1>
<div class="row"><input type="date" id="date" value="` + date + `">
` + cxAPITypeControl(fileName, apiType) + `
<button class="btn" onclick="go()">查询</button><button class="btn" onclick="force()">强制刷新</button></div></div>`)

	if warn := qStr(p["warning"]); warn != "" {
		sb.WriteString(`<div class="warn">` + htmlEscape(warn) + `</div>`)
	}
	if errMsg := qStr(p["error"]); errMsg != "" {
		sb.WriteString(`<div class="warn">` + htmlEscape(errMsg) + `</div>`)
	}

	sb.WriteString(`<div class="grid">
<div class="stat"><div class="l">总领取</div><div class="v">` + numFmt(qInt(p["totalTake"])) + `</div></div>
<div class="stat"><div class="l">总完成</div><div class="v">` + numFmt(qInt(p["totalComplete"])) + `</div></div>
<div class="stat"><div class="l">总通过</div><div class="v green">` + numFmt(qInt(p["totalSuccess"])) + `</div></div>
<div class="stat"><div class="l">总失败</div><div class="v red">` + numFmt(qInt(p["totalFail"])) + `</div></div>
<div class="stat"><div class="l">完成率</div><div class="v">` + qStr(p["completeRate"]) + `</div></div>
<div class="stat"><div class="l">通过率</div><div class="v blue">` + qStr(p["successRate"]) + `</div></div>
</div>`)

	// 备注汇总表
	sb.WriteString(`<div class="card"><h2>备注分组绩效</h2><table><thead><tr><th class="l">备注</th><th>账号数</th><th>领取</th><th>完成</th><th>通过</th><th>失败</th><th>通过率</th><th>单价</th><th>金额</th></tr></thead><tbody>`)
	for _, r := range byRemark {
		unitPrice := "—"
		amount := "—"
		if r["unitPrice"] != nil {
			unitPrice = "¥" + formatMoney(r["unitPrice"].(float64))
		}
		if r["totalAmount"] != nil {
			amount = "¥" + formatMoney(r["totalAmount"].(float64))
		}
		sb.WriteString(fmt.Sprintf(`<tr><td class="l">%s</td><td>%d</td><td>%s</td><td>%s</td><td style="color:#34d399">%s</td><td style="color:#f87171">%s</td><td>%s</td><td>%s</td><td class="amt">%s</td></tr>`,
			htmlEscape(qStr(r["remark"])), qInt(r["accounts"]), numFmt(qInt(r["take"])), numFmt(qInt(r["complete"])), numFmt(qInt(r["success"])), numFmt(qInt(r["fail"])),
			rateText(qInt(r["success"]), max(1, qInt(r["complete"]))), unitPrice, amount))
	}
	sb.WriteString(`</tbody></table></div>`)

	// 账号明细
	sb.WriteString(`<div class="card"><h2>授权账号明细（` + strconvItoa(len(batchItems)) + `）</h2><table><thead><tr><th class="l">账号</th><th class="l">备注</th><th>领取</th><th>完成</th><th>通过</th><th>失败</th><th>通过率</th></tr></thead><tbody>`)
	for _, r := range batchItems {
		sb.WriteString(fmt.Sprintf(`<tr><td class="l">%s</td><td class="l">%s</td><td>%s</td><td>%s</td><td style="color:#34d399">%s</td><td style="color:#f87171">%s</td><td>%s</td></tr>`,
			htmlEscape(qStr(r["username"])), htmlEscape(orDefault(qStr(r["_remark"]), "未备注")), numFmt(qInt(r["takeCount"])), numFmt(qInt(r["completeCount"])), numFmt(qInt(r["checkSuccessCount"])), numFmt(qInt(r["checkFailCount"])),
			rateText(qInt(r["checkSuccessCount"]), max(1, qInt(r["completeCount"])))))
	}
	sb.WriteString(`</tbody></table></div>`)

	sb.WriteString(`<div class="muted">更新时间 ` + qStr(p["generatedAt"]) + ` · ` + condStr(asBool(p["fromCache"], false), "缓存", "实时") + ` · 缓存年龄 ` + strconvItoa(qInt(p["cacheAge"])) + `s</div>
<script>function go(){location.href='?date='+document.getElementById('date').value+'&api_type='+document.getElementById('apiType').value;}
function force(){location.href='?date='+document.getElementById('date').value+'&api_type='+document.getElementById('apiType').value+'&force=1';}
if(!location.search.includes('auto=1')){setTimeout(()=>{location.href='?date='+document.getElementById('date').value+'&api_type='+document.getElementById('apiType').value+'&auto=1';},30000);}
</script></div></body></html>`)
	return sb.String()
}

func selAttr(cond bool) string {
	if cond {
		return " selected"
	}
	return ""
}

// cxAPITypeControl：cx.php 显示接口1/接口2 下拉；cx2.php 固定接口2，只用隐藏域。
func cxAPITypeControl(fileName string, apiType int) string {
	if fileName == "cx2.php" {
		return `<input type="hidden" id="apiType" value="2"><strong style="color:#818cf8">接口2</strong>`
	}
	return `<select id="apiType"><option value="1"` + selAttr(apiType == 1) + `>接口1</option><option value="2"` + selAttr(apiType == 2) + `>接口2</option></select>`
}

func numFmt(v int) string {
	s := strconvItoa(v)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

func handleCxCron(w http.ResponseWriter, r *http.Request) {
	reqCtx := newHTTPRequestCtx(r)
	today := tsDate()
	report := map[string]any{}
	for _, job := range []struct {
		File    string
		APIType int
	}{{"cx.php", 1}, {"cx2.php", 2}} {
		p := CxQuery(job.File, job.APIType, today, true, false, reqCtx)
		status := firstNonNilStr(p["warning"], p["error"])
		if status == "" {
			status = "ok"
		}
		report[job.File] = map[string]any{"status": status, "fromCache": p["fromCache"], "totalSuccess": p["totalSuccess"]}
	}
	writeJSONOK(w, map[string]any{"ok": true, "date": today, "report": report})
}
