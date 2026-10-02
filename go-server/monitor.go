package main

import (
	"net/http"
	"strings"
)

func loadMonitorMetrics() map[string]any {
	g := fg.metrics()
	now := timeNow()
	// 最近 60 秒日志统计
	logsTotal, logsSuccess, logsFail, logsBusy := 0, 0, 0, 0
	sumMs := 0
	var recentLogs []map[string]any
	for _, line := range readLogs(logFile, 500) {
		t, err := timeParse("2006-01-02 15:04:05", trimStr(line["time"]))
		if err != nil || now.Sub(t).Seconds() > 60 {
			continue
		}
		logsTotal++
		status := toLower(trimStr(line["status"]))
		errText := trimStr(line["error"])
		msg := trimStr(line["msg"])
		isBusy := strings.Contains(errText, "系统繁忙") || strings.Contains(msg, "系统繁忙") || status == "busy"
		if isBusy {
			logsBusy++
		}
		switch status {
		case "success":
			logsSuccess++
		case "forward_fail", "fail", "task_server_error":
			logsFail++
		}
		if timing, ok := line["timing_ms"].(map[string]any); ok {
			sumMs += qInt(timing["total"])
		}
		if len(recentLogs) < 20 {
			forwardMs, gateWait, localMs := 0, 0, 0
			if timing, ok := line["timing_ms"].(map[string]any); ok {
				forwardMs = qInt(firstNonNil(timing["forward"], line["forward_ms"]))
				gateWait = qInt(firstNonNil(timing["gate_wait"], line["gate_wait_ms"]))
				localMs = qInt(firstNonNil(timing["local"], line["local_ms"]))
			}
			recentLogs = append(recentLogs, map[string]any{
				"time": trimStr(line["time"]), "source": trimStr(line["source"]), "status": trimStr(line["status"]),
				"username": trimStr(line["username"]), "total_ms": qInt(timingTotal(line)),
				"forward_ms": forwardMs, "gate_wait_ms": gateWait, "local_ms": localMs,
			})
		}
	}
	avgMs := 0
	if logsTotal > 0 {
		avgMs = sumMs / logsTotal
	}
	active := qInt(g["active"])
	maxc := qInt(g["max_concurrency"])
	occ := 0
	if maxc > 0 {
		occ = int(float64(active) / float64(maxc) * 100)
	}
	return map[string]any{
		"ok": true, "now": tsDateTime(), "now_ts": now.Unix(),
		"active": active, "max_concurrency": maxc, "occupancy": occ,
		"rate_1s": g["rate_1s"], "rate_10s": g["rate_10s"], "rate_60s": g["rate_60s"],
		"busy_10s": g["busy_10s"], "busy_60s": g["busy_60s"],
		"config": map[string]any{
			"max_concurrency": maxc, "rate_limit_per_sec": g["rate_limit_per_sec"], "wait_ms": g["wait_ms"],
			"forward_timeout": forwardTimeout, "updated_at": tsDateTime(),
		},
		"series": g["series"],
		"logs": map[string]any{
			"total_60s": logsTotal, "success_60s": logsSuccess, "fail_60s": logsFail, "busy_60s": logsBusy,
			"avg_ms": avgMs, "recent": recentLogs,
		},
		"system": map[string]any{
			"loadavg": nil, "go_version": goVersion(), "os": "linux",
		},
	}
}

func timingTotal(line map[string]any) int {
	if timing, ok := line["timing_ms"].(map[string]any); ok {
		return qInt(timing["total"])
	}
	return 0
}

func goVersion() string { return "1.21+" }

func handleLoadMonitor(w http.ResponseWriter, r *http.Request) {
	key := getAuthKey(r)
	authed := verifyAuthKey(key)
	if _, ok := r.URL.Query()["api"]; ok {
		if !authed {
			writeJSON(w, 403, map[string]any{"ok": false, "msg": "密码错误"})
			return
		}
		writeJSONOK(w, loadMonitorMetrics())
		return
	}
	if !authed {
		writeHTML(w, `<body style="background:#0f172a;color:#e2e8f0;font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh"><form method="get" style="background:#1e293b;padding:32px;border-radius:14px;width:320px"><h1 style="font-size:18px;margin:0 0 8px">负载监控</h1><p style="color:#94a3b8;font-size:13px">请输入后台密码</p><input type="password" name="key" placeholder="后台密码" style="width:100%;height:40px;border-radius:8px;border:1px solid #334155;background:#0f172a;color:#e2e8f0;padding:0 12px;box-sizing:border-box"><button type="submit" style="width:100%;height:40px;margin-top:12px;border:none;border-radius:8px;background:#6366f1;color:#fff;font-weight:600;cursor:pointer">进入监控</button></form></body>`)
		return
	}
	// 简化监控页
	writeHTML(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>转发负载监控</title>
<style>body{background:#0f172a;color:#e2e8f0;font-family:sans-serif;margin:0;padding:20px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px}.card{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:18px}.card .label{color:#94a3b8;font-size:12px}.card .value{font-size:28px;font-weight:800;margin-top:8px}table{width:100%;border-collapse:collapse;font-size:12px;margin-top:20px}th,td{text-align:left;padding:8px 10px;border-bottom:1px solid #1e293b}th{color:#94a3b8}</style></head><body>
<h1>转发负载监控</h1><div class="grid" id="grid"></div>
<div style="background:#1e293b;border-radius:12px;padding:18px;margin-top:16px"><h2 style="font-size:14px;margin:0 0 12px">最近请求日志</h2><div style="overflow-x:auto"><table><thead><tr><th>时间</th><th>来源</th><th>状态</th><th>账号</th><th>耗时</th></tr></thead><tbody id="logs"></tbody></table></div></div>
<script>
const KEY=`+"`"+key+"`"+`;
async function refresh(){const r=await fetch('?api=1&key='+encodeURIComponent(KEY));const d=await r.json();
document.getElementById('grid').innerHTML=
'<div class="card"><div class="label">当前转发并发</div><div class="value">'+d.active+' / '+d.max_concurrency+'</div></div>'+
'<div class="card"><div class="label">每秒转发</div><div class="value">'+d.rate_1s+'</div></div>'+
'<div class="card"><div class="label">近60秒转发</div><div class="value">'+d.rate_60s+'</div></div>'+
'<div class="card"><div class="label">近60秒繁忙</div><div class="value">'+d.busy_60s+'</div></div>';
document.getElementById('logs').innerHTML=(d.logs.recent||[]).map(l=>'<tr><td>'+l.time+'</td><td>'+l.source+'</td><td>'+l.status+'</td><td>'+l.username+'</td><td>'+(l.total_ms||0)+'ms</td></tr>').join('');}
refresh();setInterval(refresh,2000);
</script></body></html>`)
}
