package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// ---- cx_view.php：备注专属查询 ----

type cxViewAccount struct {
	Label        string
	DataKey      string
	AccountsFile string
	Pricing      map[string]float64
}

var cxViewAccounts = map[string]cxViewAccount{
	"532": {Label: "Ajie", DataKey: "3bd0f92c13ed0c94", AccountsFile: api1AccountsFile, Pricing: map[string]float64{"阳": 0.04, "果子": 0.048, "老妈": 0.048, "传媒": 0.04, "杰": 0.05, "C-C5": 0.05, "C1": 0.05, "C2": 0.05, "C3": 0.05, "C4": 0.05, "C5": 0.05, "CN": 0.04, "C": 0.05}},
}

func cxViewReadLinks() []map[string]any {
	var data map[string]any
	if !readJSONFile(dp(remarkLinksFile), &data) {
		return []map[string]any{}
	}
	var raw []any
	raw, _ = data["links"].([]any)
	var out []map[string]any
	for _, l := range raw {
		if m, ok := l.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func cxViewFindLink(token string) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	for _, l := range cxViewReadLinks() {
		if asBool(l["enabled"], true) && hashEquals(trimStr(l["token"]), token) {
			return l
		}
	}
	return nil
}

func cxViewReadAccountsMap() map[string]string {
	out := map[string]string{}
	for _, acc := range readAllowedAccountsForFile(api1AccountsFile) {
		out[trimStr(acc["username"])] = trimStr(acc["remark"])
	}
	return out
}

func cxViewDateLastGoodFile(dataKey, date string) string {
	return cxCacheDirName + "/cx_date_lastgood_" + dataKey + "_" + md5hex(date) + ".json"
}

func cxViewCachedPayload(dataKey, date string) map[string]any {
	return cxReadPayloadFile(cxViewDateLastGoodFile(dataKey, date))
}

func cxViewDayLabel(date string) string {
	if date == tsDate() {
		return "今天"
	}
	if date == timeNow().AddDate(0, 0, -1).Format("2006-01-02") {
		return "昨天"
	}
	if date == timeNow().AddDate(0, 0, -2).Format("2006-01-02") {
		return "前天"
	}
	return ""
}

// cxViewInventoryStats 读取库存模式（任务模式）某日统计：下发id -> {success,fail,fetched}
func cxViewInventoryStats(date string) map[string]map[string]int {
	stats := map[string]map[string]map[string]int{}
	_ = readJSONFile(dp(inventoryStatsFile), &stats)
	return stats[date]
}

func handleCxView(w http.ResponseWriter, r *http.Request) {
	link := cxViewFindLink(r.URL.Query().Get("t"))
	if link == nil {
		writeHTML(w, `<body style="font-family:sans-serif;padding:40px;text-align:center"><h2>链接无效或已停用</h2></body>`)
		return
	}
	account := trimStr(link["account"])
	remark := trimStr(link["remark"])
	cfg, ok := cxViewAccounts[account]
	if !ok {
		writeHTML(w, `<body style="font-family:sans-serif;padding:40px;text-align:center"><h2>链接无效或已停用</h2></body>`)
		return
	}
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

	accountsMap := cxViewReadAccountsMap()
	var userList []string
	for username, accRemark := range accountsMap {
		if containsStr(remarks, strings.TrimSpace(accRemark)) {
			userList = append(userList, username)
		}
	}
	sort.Strings(userList)

	hasAmountCol := false
	for _, rmk := range remarks {
		if cxPricePerSuccess(cfg.Pricing, rmk) != nil {
			hasAmountCol = true
			break
		}
	}

	type dayRow struct {
		username, remark string
		take, complete, success, fail int
		amount *float64
	}
	type dayTbl struct {
		date, label string
		rows        []dayRow
		total       map[string]int
		amount      *float64
	}
	var days []dayTbl
	latestFetch := int64(0)
	grand := map[string]int{"take": 0, "complete": 0, "success": 0, "fail": 0}
	grandAmount := 0.0
	for i := 0; i < 3; i++ {
		date := timeNow().AddDate(0, 0, -i).Format("2006-01-02")
		payload := cxViewCachedPayload(cfg.DataKey, date)
		latestFetch = max64(latestFetch, int64(qInt(payload["fetchTime"])))
		byUser := map[string]map[string]any{}
		for _, row := range rowsOf(payload, "batchItems") {
			rowRemark := trimStr(row["_remark"])
			if !containsStr(remarks, rowRemark) {
				continue
			}
			username := trimStr(row["username"])
			if username != "" {
				byUser[username] = row
			}
		}
		var drs []dayRow
		total := map[string]int{"take": 0, "complete": 0, "success": 0, "fail": 0}
		dayAmount := 0.0
		for _, username := range userList {
			row := byUser[username]
			take := qInt(row["takeCount"])
			complete := qInt(row["completeCount"])
			success := qInt(row["checkSuccessCount"])
			fail := qInt(row["checkFailCount"])
			if take+complete+success+fail <= 0 {
				continue
			}
			userRemark := accountsMap[username]
			var amount *float64
			if hasAmountCol {
				if p := cxPricePerSuccess(cfg.Pricing, userRemark); p != nil {
					v := round2(*p * float64(success))
					amount = &v
					dayAmount += v
				}
			}
			drs = append(drs, dayRow{username: username, remark: userRemark, take: take, complete: complete, success: success, fail: fail, amount: amount})
			total["take"] += take
			total["complete"] += complete
			total["success"] += success
			total["fail"] += fail
		}
		if len(drs) > 0 {
			var amt *float64
			if hasAmountCol {
				v := round2(dayAmount)
				amt = &v
			}
			days = append(days, dayTbl{date: date, label: firstNonNilStr(cxViewDayLabel(date), date), rows: drs, total: total, amount: amt})
		}
		for k, v := range total {
			grand[k] += v
		}
		if hasAmountCol {
			grandAmount += dayAmount
		}
	}

	// 库存模式（任务模式）统计：按日期、按下发id
	type invRow struct {
		remark                 string
		fetched, success, fail int
		auditFail              int
		amount                 float64
	}
	type invDay struct {
		date, label string
		rows        []invRow
	}
	invPrices := invLoadPrices()
	var invDays []invDay
	for i := 0; i < 3; i++ {
		date := timeNow().AddDate(0, 0, -i).Format("2006-01-02")
		invStats := cxViewInventoryStats(date)
		audit := cxAuditFailAlloc(cfg.DataKey, date)
		auditByRemark := map[string]int{}
		for _, a := range rowsOf(audit, "accounts") {
			auditByRemark[trimStr(a["remark"])] += qInt(a["audit"])
		}
		var rows []invRow
		for _, rmk := range remarks {
			s := invStats[rmk]
			if s == nil {
				continue
			}
			if s["fetched"]+s["success"]+s["fail"] <= 0 {
				continue
			}
			amount := round2(invPrices[rmk] * float64(s["success"]))
			rows = append(rows, invRow{remark: rmk, fetched: s["fetched"], success: s["success"], fail: s["fail"], auditFail: auditByRemark[rmk], amount: amount})
		}
		if len(rows) > 0 {
			invDays = append(invDays, invDay{date: date, label: firstNonNilStr(cxViewDayLabel(date), date), rows: rows})
		}
	}

	grandAmount = round2(grandAmount)
	updatedAt := tsDateTime()
	if latestFetch > 0 {
		updatedAt = timeUnixText(latestFetch)[:16]
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	sb.WriteString("<title>" + htmlEscape(remark) + " 专属查询</title><style>")
	sb.WriteString(`*{box-sizing:border-box}body{margin:0;min-height:100vh;color:#1f2937;font-family:"Segoe UI","Microsoft YaHei UI","PingFang SC",sans-serif;display:flex;align-items:flex-start;justify-content:center;padding:28px 14px;background:#f5f7fb url('/bj.jpeg') center/cover no-repeat fixed}
.box{width:100%;max-width:1100px;background:rgba(255,255,255,.50);border:1px solid rgba(255,255,255,.28);border-radius:14px;padding:22px;box-shadow:0 10px 30px rgba(15,23,42,.12)}
h2{margin:0 0 6px;font-size:20px}.meta{color:#6b7280;font-size:13px;margin-bottom:16px}
table{width:100%;border-collapse:collapse}th,td{padding:11px 10px;text-align:center;border-bottom:1px solid #f0f0f0;font-size:14px}
th{color:#6b7280;font-size:12px;font-weight:700;background:rgba(255,255,255,.50)}td.date{text-align:left;font-weight:700;white-space:nowrap}
td.user{font-weight:700;text-align:left;white-space:nowrap}td.remark{color:#6b7280;text-align:left;white-space:nowrap}
h3.day-head{margin:18px 0 8px;font-size:14px;font-weight:700;color:#374151}
.mode-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px;margin-bottom:6px;align-items:stretch}
.mode-card{display:flex;flex-direction:column;background:rgba(255,255,255,.50);border:1px solid rgba(255,255,255,.25);border-radius:12px;padding:14px;min-width:0}
.mode-card.inv{justify-content:center}
.mode-title{font-size:12px;font-weight:800;color:#7c3aed;margin-bottom:8px;display:flex;align-items:center;gap:6px}
.mode-title::before{content:"";width:3px;height:13px;background:currentColor;border-radius:2px}
.mode-title.normal{color:#2563eb}
@media(max-width:680px){.mode-grid{grid-template-columns:1fr}}
tr.grand-row td{background:rgba(255,255,255,.50);font-weight:700}td.ok{color:#16a34a;font-weight:700}td.bad{color:#dc2626;font-weight:700}td.amt{color:#2563eb;font-weight:700}
tfoot td{font-weight:700;background:rgba(255,255,255,.50);border-bottom:0}.foot{margin-top:14px;color:#9ca3af;font-size:12px;text-align:right}.empty{padding:30px;text-align:center;color:#9ca3af}</style></head><body><div class="box">`)
	sb.WriteString("<h2>" + htmlEscape(remark) + " 专属查询</h2>")
	sb.WriteString("<div class=\"meta\">账号" + htmlEscape(cfg.Label) + " · 最近3天 · 共 " + strconvItoa(len(userList)) + " 个账号</div>")
	dayByDate := map[string]*dayTbl{}
	for i := range days {
		dayByDate[days[i].date] = &days[i]
	}
	invByDate := map[string]*invDay{}
	for i := range invDays {
		invByDate[invDays[i].date] = &invDays[i]
	}
	if len(userList) == 0 {
		sb.WriteString(`<div class="empty">暂无授权账号</div>`)
	} else {
		amtHead := ""
		if hasAmountCol {
			amtHead = `<th>金额</th>`
		}
		hasAny := false
		for i := 0; i < 3; i++ {
			date := timeNow().AddDate(0, 0, -i).Format("2006-01-02")
			label := firstNonNilStr(cxViewDayLabel(date), date)
			d := dayByDate[date]
			iv := invByDate[date]
			if d == nil && iv == nil {
				continue
			}
			hasAny = true
			sb.WriteString("<h3 class=\"day-head\">" + htmlEscape(label) + " " + htmlEscape(date) + "</h3>")
			sb.WriteString("<div class=\"mode-grid\">")
			if d != nil {
				sb.WriteString("<div class=\"mode-card\"><div class=\"mode-title normal\">普通模式</div><table><thead><tr><th style=\"text-align:left\">账号</th><th style=\"text-align:left\">备注</th><th>领取</th><th>完成</th><th>通过</th><th>失败</th>" + amtHead + "</tr></thead><tbody>")
				for _, row := range d.rows {
					amtCell := ""
					if hasAmountCol {
						if row.amount != nil {
							amtCell = `<td class="amt">¥` + formatMoney(*row.amount) + `</td>`
						} else {
							amtCell = `<td>-</td>`
						}
					}
					sb.WriteString(fmt.Sprintf(`<tr><td class="user">%s</td><td class="remark">%s</td><td>%d</td><td>%d</td><td class="ok">%d</td><td class="bad">%d</td>%s</tr>`,
						htmlEscape(row.username), htmlEscape(orDefault(row.remark, "未备注")), row.take, row.complete, row.success, row.fail, amtCell))
				}
				amtFoot := ""
				if hasAmountCol {
					if d.amount != nil {
						amtFoot = `<td class="amt">¥` + formatMoney(*d.amount) + `</td>`
					} else {
						amtFoot = `<td>-</td>`
					}
				}
				sb.WriteString(fmt.Sprintf(`</tbody><tfoot><tr><td style="text-align:left">合计</td><td style="text-align:left">%d 个账号</td><td>%d</td><td>%d</td><td class="ok">%d</td><td class="bad">%d</td>%s</tr></tfoot></table></div>`,
					len(d.rows), d.total["take"], d.total["complete"], d.total["success"], d.total["fail"], amtFoot))
			}
			if iv != nil {
				sb.WriteString(`<div class="mode-card inv"><div class="mode-title">库存模式</div><table><thead><tr><th style="text-align:left">下发id</th><th>拉取</th><th>通过</th><th>失败</th><th>审核失败</th><th>金额</th></tr></thead><tbody>`)
				var itake, isucc, ifail, iaudit int
				var iamt float64
				for _, r := range iv.rows {
					itake += r.fetched
					isucc += r.success
					ifail += r.fail
					iaudit += r.auditFail
					iamt += r.amount
					sb.WriteString(fmt.Sprintf(`<tr><td class="user">%s</td><td>%d</td><td class="ok">%d</td><td class="bad">%d</td><td style="color:#f59e0b">%d</td><td class="amt">¥%s</td></tr>`, htmlEscape(r.remark), r.fetched, r.success, r.fail, r.auditFail, formatMoney(r.amount)))
				}
				sb.WriteString(fmt.Sprintf(`</tbody><tfoot><tr><td style="text-align:left">合计</td><td>%d</td><td class="ok">%d</td><td class="bad">%d</td><td style="color:#f59e0b">%d</td><td class="amt">¥%s</td></tr></tfoot></table></div>`, itake, isucc, ifail, iaudit, formatMoney(iamt)))
			}
			sb.WriteString("</div>")
		}
		if !hasAny {
			sb.WriteString(`<div class="empty">最近3天暂无数据</div>`)
		}
		if len(days) > 1 {
			sb.WriteString("<h3 class=\"day-head\">普通模式 3天总计</h3><table><tbody><tr class=\"grand-row\"><td style=\"text-align:left\">总计</td><td style=\"text-align:left\">" + strconvItoa(len(days)) + " 天</td><td>" + strconvItoa(grand["take"]) + "</td><td>" + strconvItoa(grand["complete"]) + "</td><td class=\"ok\">" + strconvItoa(grand["success"]) + "</td><td class=\"bad\">" + strconvItoa(grand["fail"]) + "</td>")
			if hasAmountCol {
				sb.WriteString(`<td class="amt">¥` + formatMoney(grandAmount) + `</td>`)
			}
			sb.WriteString("</tr></tbody></table>")
		}
	}
	sb.WriteString("<div class=\"foot\">更新时间 " + htmlEscape(updatedAt) + "</div>")
	sb.WriteString(`<script>if(!location.search.includes('auto=1')){setTimeout(function(){location.href=location.pathname+location.search+(location.search?'&':'?')+'auto=1';},30000);}</script>`)
	sb.WriteString("</div></body></html>")
	writeHTML(w, sb.String())
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func writeHTML(w http.ResponseWriter, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
