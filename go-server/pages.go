package main

import (
	"net/http"
)

func renderAdminPage(w http.ResponseWriter, r *http.Request, h5 bool) {
	writeHTML(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Ajie 控制台</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#0a0f1c;--panel:#0d1526;--card:#121b31;--card2:#182238;--line:#22304f;--line2:#2e3d60;--text:#e9eef8;--muted:#94a3c2;--muted2:#64748b;--accent:#3b82f6;--accent2:#60a5fa;--accent-soft:rgba(59,130,246,.16);--green:#34d399;--red:#f87171;--amber:#fbbf24;--blue:#60a5fa;--cyan:#22d3ee;--shadow:0 20px 50px -22px rgba(2,6,18,.75)}
html[data-theme=purple]{--bg:#0c0917;--panel:#14102b;--card:#1b153a;--card2:#231b4a;--line:#2d2458;--line2:#3b3070;--accent:#8b5cf6;--accent2:#a78bfa;--accent-soft:rgba(139,92,246,.18)}
html[data-theme=teal]{--bg:#061310;--panel:#0b1e1a;--card:#10261f;--card2:#153028;--line:#1e4038;--line2:#285548;--accent:#10b981;--accent2:#34d399;--accent-soft:rgba(16,185,129,.16)}
html[data-theme=light]{--bg:#eef2f8;--panel:#ffffff;--card:#ffffff;--card2:#f4f7fb;--line:#e2e8f1;--line2:#d2dae7;--text:#172033;--muted:#5c6b84;--muted2:#8a97ad;--accent:#2563eb;--accent2:#3b82f6;--accent-soft:rgba(37,99,235,.1);--shadow:0 16px 40px -22px rgba(15,23,42,.28)}
html,body{height:100%}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;font-size:14px;line-height:1.55;-webkit-font-smoothing:antialiased;background-image:radial-gradient(900px 460px at 12% -8%,var(--accent-soft),transparent 60%),radial-gradient(780px 420px at 100% -4%,rgba(34,211,238,.06),transparent 55%)}
::-webkit-scrollbar{width:10px;height:10px}
::-webkit-scrollbar-thumb{background:var(--line2);border-radius:8px;border:2px solid transparent;background-clip:padding-box}
::-webkit-scrollbar-thumb:hover{background:var(--muted2);background-clip:padding-box}
.hidden{display:none!important}
.login{position:fixed;inset:0;display:flex;align-items:center;justify-content:center;background:radial-gradient(1100px 560px at 50% -10%,var(--accent-soft) 0%,transparent 55%),var(--bg)}
.login-card{width:384px;max-width:92vw;background:var(--card);border:1px solid var(--line2);border-radius:20px;padding:40px 34px;text-align:center;box-shadow:var(--shadow)}
.logo-big{display:flex;justify-content:center;color:var(--accent2);filter:drop-shadow(0 6px 18px var(--accent-soft))}
.login-card h1{font-size:24px;letter-spacing:-.3px;margin:14px 0 4px}
.login-card .sub{color:var(--muted);margin-bottom:26px}
.login-card input{width:100%;height:46px;background:var(--bg);border:1px solid var(--line);border-radius:11px;color:var(--text);padding:0 15px;font-size:15px;margin-bottom:14px}
.login-card input:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 3px var(--accent-soft)}
.app{display:flex;min-height:100vh}
.sidebar{width:228px;flex-shrink:0;background:var(--panel);border-right:1px solid var(--line);display:flex;flex-direction:column;position:sticky;top:0;height:100vh}
.logo{display:flex;align-items:center;gap:11px;font-size:16px;font-weight:800;letter-spacing:-.2px;padding:20px;border-bottom:1px solid var(--line)}
.logo .dot{width:27px;height:27px;border-radius:8px;background:linear-gradient(135deg,var(--accent),var(--accent2));display:flex;align-items:center;justify-content:center;flex-shrink:0;box-shadow:0 6px 16px var(--accent-soft)}
nav{padding:14px 12px;display:flex;flex-direction:column;gap:3px;flex:1;overflow-y:auto}
.nav{display:flex;align-items:center;gap:11px;width:100%;text-align:left;background:transparent;border:1px solid transparent;border-radius:10px;color:var(--muted);padding:11px 13px;font-size:14px;font-weight:500;cursor:pointer;font-family:inherit;transition:background .18s ease,color .18s ease,box-shadow .18s ease}
.nav:hover{background:var(--card2);color:var(--text)}
.nav.active{background:var(--accent-soft);color:var(--text);box-shadow:inset 2px 0 0 var(--accent)}
.nav .ic{display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;flex-shrink:0}
.nav .ic svg{width:18px;height:18px}
.nav.active .ic{color:var(--accent2)}
.side-foot{padding:12px;border-top:1px solid var(--line);display:flex;flex-direction:column;gap:4px}
.main{flex:1;min-width:0;display:flex;flex-direction:column}
.topbar{display:flex;align-items:center;justify-content:space-between;padding:16px 30px;border-bottom:1px solid var(--line);background:var(--panel);position:sticky;top:0;z-index:5}
.top-title{font-size:19px;font-weight:800;letter-spacing:-.3px}
.top-actions{display:flex;gap:8px}
.content{padding:26px 30px;width:100%}
.card{background:var(--card);border:1px solid var(--line);border-radius:16px;padding:22px;margin-bottom:20px;box-shadow:var(--shadow)}
.card-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;gap:12px;flex-wrap:wrap}
.card-head h2{font-size:16px;font-weight:700;letter-spacing:-.2px}
.hint{color:var(--muted);font-size:12.5px;margin:-6px 0 12px}
.hint code{background:var(--bg);border:1px solid var(--line);border-radius:5px;padding:1px 6px;color:var(--cyan);font-size:12px}
.btn{background:var(--card2);border:1px solid var(--line2);border-radius:10px;color:var(--text);padding:9px 15px;cursor:pointer;font:inherit;font-size:13px;font-weight:500;transition:background .18s ease,border-color .18s ease,transform .18s ease,box-shadow .18s ease;white-space:nowrap}
.btn:hover{background:var(--line2);border-color:var(--muted2)}
.btn:active{transform:translateY(1px) scale(.99)}
button:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible{outline:2px solid var(--accent);outline-offset:1px}
.btn.primary{background:linear-gradient(135deg,var(--accent),var(--accent2));border-color:transparent;color:#fff;font-weight:600;box-shadow:0 8px 20px -10px var(--accent-soft)}
.btn.primary:hover{filter:brightness(1.08)}
.btn.danger{background:rgba(248,113,113,.12);border-color:rgba(248,113,113,.3);color:var(--red)}
.btn.ghost{background:transparent;border-color:var(--line)}
.btn.sm{padding:6px 12px;font-size:12px;border-radius:8px}
.btn.block{width:100%;height:46px;font-size:15px;border-radius:11px}
.tbl{width:100%;border-collapse:collapse}
.tbl th{text-align:left;color:var(--muted2);font-size:11.5px;font-weight:600;text-transform:uppercase;letter-spacing:.5px;padding:11px 14px;border-bottom:1px solid var(--line)}
.tbl td{padding:13px 14px;border-bottom:1px solid var(--line);font-size:13px;font-variant-numeric:tabular-nums}
.tbl tbody tr{transition:background .15s ease}
.tbl tbody tr:hover{background:var(--card2)}
.tbl tbody tr:nth-child(even){background:rgba(255,255,255,.02)}
.tbl thead th{position:sticky;top:0;background:var(--card);z-index:2}
.mono{font-family:ui-monospace,"SF Mono",Consolas,monospace;font-size:12px;color:var(--muted)}
.badge{display:inline-block;padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600;letter-spacing:.2px}
.badge.ok{background:rgba(52,211,153,.15);color:var(--green)}
.badge.off{background:rgba(248,113,113,.15);color:var(--red)}
.badge.blue{background:rgba(96,165,250,.15);color:var(--blue)}
.badge.amber{background:rgba(251,191,36,.15);color:var(--amber)}
.stats-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:16px;margin-bottom:22px}
.stat{background:var(--card);border:1px solid var(--line);border-top:3px solid var(--accent);border-radius:14px;padding:18px;transition:transform .2s ease,box-shadow .2s ease}
.stat:hover{transform:translateY(-3px);box-shadow:var(--shadow)}
.s-l{color:var(--muted);font-size:12.5px;font-weight:500}
.s-v{font-size:26px;font-weight:800;margin-top:8px;letter-spacing:-.5px;font-variant-numeric:tabular-nums}
.item{background:var(--card2);border:1px solid var(--line);border-radius:12px;padding:16px 18px;margin-bottom:10px;transition:border-color .2s ease}
.item:hover{border-color:var(--line2)}
.item-head{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.item-sub{color:var(--muted);font-size:12.5px;margin-top:6px}
.dev-group{margin-top:10px;border-top:1px dashed var(--line);padding-top:8px}
.dev-sub{display:flex;align-items:center;gap:9px;flex-wrap:wrap;font-size:12.5px;padding:7px 2px;border-bottom:1px solid var(--line)}
.dev-sub:last-child{border-bottom:none}
.dev-sub .muted{font-size:12px}
.dev-sub .btn{margin-left:auto}
.item-actions{display:flex;gap:8px;margin-top:10px;flex-wrap:wrap}
.dot{display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px}
.dot.green{background:var(--green);box-shadow:0 0 6px var(--green)}
.dot.off{background:var(--muted2)}
.dot.red{background:var(--red)}
input[type=text],input[type=number],input[type=password],select,textarea{background:var(--bg);border:1px solid var(--line);border-radius:10px;color:var(--text);padding:10px 13px;font:inherit;font-size:13px;width:100%;transition:border-color .18s ease,box-shadow .18s ease}
input[type=file]{background:var(--bg);border:1px solid var(--line);border-radius:10px;color:var(--text);padding:9px 12px;font:inherit;font-size:13px;width:100%}
textarea{font-family:ui-monospace,"SF Mono",Consolas,monospace;font-size:12.5px;line-height:1.6;resize:vertical}
input:focus,select:focus,textarea:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 3px var(--accent-soft)}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}
.form-grid label{display:flex;flex-direction:column;gap:7px;color:var(--muted);font-size:12.5px}
.form-grid .full{grid-column:1/-1}
.check{flex-direction:row!important;align-items:center;gap:8px!important;color:var(--text)!important;font-size:14px!important}
.check input{width:auto}
.inv-layout{display:grid;grid-template-columns:1fr 1fr;gap:20px;align-items:start}
.inv-acc{border:1px solid var(--line);border-radius:10px;padding:12px;margin-bottom:12px;background:var(--card2)}
.inv-acc-head{margin-bottom:8px}
#invFetchLog,#invQueueDetail{max-height:420px;overflow-y:auto}
.inv-right{position:sticky;top:70px}
@media(max-width:1100px){.inv-layout{grid-template-columns:1fr}}
.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
.empty{padding:36px;text-align:center;color:var(--muted2)}
.muted{color:var(--muted)}
.modal{position:fixed;inset:0;background:rgba(2,6,18,.62);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;z-index:50;padding:16px;animation:fadein .2s ease}
.modal-box{width:580px;max-width:94vw;max-height:90vh;overflow-y:auto;background:var(--card);border:1px solid var(--line2);border-radius:18px;padding:24px;box-shadow:var(--shadow)}
@keyframes fadein{from{opacity:0}to{opacity:1}}
pre.log-json{background:var(--bg);border:1px solid var(--line);border-radius:10px;padding:13px;font-family:ui-monospace,Consolas,monospace;font-size:12px;white-space:pre-wrap;word-break:break-all;max-height:50vh;overflow:auto;color:var(--text)}
#toasts{position:fixed;top:18px;right:18px;z-index:100;display:flex;flex-direction:column;gap:10px}
.toast{background:var(--card);border:1px solid var(--line2);border-left:3px solid var(--accent);border-radius:12px;padding:13px 17px;min-width:230px;box-shadow:var(--shadow);font-size:13px;animation:tin .28s cubic-bezier(.2,.8,.2,1)}
.toast.success{border-left-color:var(--green)}
.toast.error{border-left-color:var(--red)}
.toast.out{opacity:0;transform:translateX(20px);transition:.3s}
@keyframes tin{from{opacity:0;transform:translateX(20px)}to{opacity:1;transform:none}}
@media(max-width:860px){.sidebar{width:62px}.sidebar .nav>span:not(.ic),.sidebar .logo>span:not(.dot){display:none}.nav{justify-content:center;padding:11px 0}.content{padding:18px}.topbar{padding:13px 16px}.form-grid{grid-template-columns:1fr}}
</style></head><body>
<div id="loginView" class="login"><div class="login-card">
<div class="logo-big"><svg viewBox="0 0 24 24" width="38" height="38" fill="currentColor"><path d="M13 2 4 14h6l-1 8 9-12h-6z"/></svg></div><h1>Ajie 控制台</h1><div class="sub">管理员登录</div>
<input type="password" id="keyInput" placeholder="后台密码" onkeydown="if(event.key==='Enter')doLogin()">
<button class="btn primary block" onclick="doLogin()">登录</button>
</div></div>
<div id="adminView" class="app hidden">
<aside class="sidebar">
<div class="logo"><span class="dot"><svg viewBox="0 0 24 24" width="15" height="15" fill="#fff"><path d="M13 2 4 14h6l-1 8 9-12h-6z"/></svg></span><span>Ajie 控制台</span></div>
<nav>
<button class="nav active" data-t="ov" onclick="switchTab('ov')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="3.5" y="3.5" width="7" height="7" rx="1.5"/><rect x="13.5" y="3.5" width="7" height="7" rx="1.5"/><rect x="3.5" y="13.5" width="7" height="7" rx="1.5"/><rect x="13.5" y="13.5" width="7" height="7" rx="1.5"/></svg></span><span>总览</span></button>
<button class="nav" data-t="logs" onclick="switchTab('logs')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M8.5 6h11M8.5 12h11M8.5 18h11"/><path d="M4.5 6h.01M4.5 12h.01M4.5 18h.01"/></svg></span><span>日志</span></button>
<button class="nav" data-t="accounts" onclick="switchTab('accounts')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3"/></svg></span><span>插件账号</span></button>
<button class="nav" data-t="devices" onclick="switchTab('devices')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="7" y="2.5" width="10" height="19" rx="2.5"/><path d="M11 18h2"/></svg></span><span>设备</span></button>
<button class="nav" data-t="auth" onclick="switchTab('auth')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="3.5"/><path d="M10.2 12.8 20.5 2.5M15.5 7.5l3 3"/></svg></span><span>授权账号</span></button>
<button class="nav" data-t="sub" onclick="switchTab('sub')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="3.5"/><path d="M3 20.5c0-3.3 2.7-6 6-6s6 2.7 6 6"/><path d="M16 4.6a3.5 3.5 0 0 1 0 6.8M18.5 20.5c0-2.6-1.4-4.8-3.5-5.9"/></svg></span><span>子账号</span></button>
<button class="nav" data-t="update" onclick="switchTab('update')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3.5v11M7.5 10.5l4.5 4 4.5-4"/><path d="M4.5 20.5h15"/></svg></span><span>更新配置</span></button>
<button class="nav" data-t="cache" onclick="switchTab('cache')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/></svg></span><span>缓存</span></button>
<button class="nav" data-t="inv" onclick="switchTab('inv')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M21 8.2 12 3.5 3 8.2v7.6L12 20.5l9-4.7z"/><path d="M3.3 8.3 12 12.8l8.7-4.5M12 12.8v7.7"/></svg></span><span>库存模式</span></button>
</nav>
<div class="side-foot">
<button class="nav" onclick="openPwd()"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="4.5" y="10.5" width="15" height="9.5" rx="2"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/></svg></span><span>修改密码</span></button>
<button class="nav" onclick="logout()"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3"/><path d="M16 17l5-5-5-5M21 12H9"/></svg></span><span>退出登录</span></button>
</div>
</aside>
<main class="main">
<header class="topbar"><div class="top-title" id="topTitle">总览</div>
<div class="top-actions"><button class="btn ghost sm" onclick="cycleTheme()"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:-2px;margin-right:5px"><path d="M12 3a9 9 0 1 0 0 18c1.6 0 2.3-1 2.3-2 0-.6-.2-1.2-.7-1.6-.5-.4-.8-.9-.8-1.5 0-1 .8-1.8 1.9-1.8h1.6A2.7 2.7 0 0 0 19 11.3C19 6.7 16 3 12 3z"/><circle cx="7.5" cy="10.5" r="1"/><circle cx="10.5" cy="6.5" r="1"/><circle cx="14.5" cy="6.5" r="1"/></svg>主题</button><button class="btn ghost sm" onclick="location.href='../load_monitor.php'">负载监控</button><button class="btn ghost sm" onclick="location.href='../cx_links.php'">备注链接</button></div>
</header>
<div class="content">

<div id="tab-ov" class="tabpane">
<div class="stats-grid" id="ovCards"></div>
<div class="card"><div class="card-head"><h2>最近成功转发</h2><button class="btn ghost sm" onclick="loadOverview()">刷新</button></div><div id="ovRecent"></div></div>
</div>

<div id="tab-logs" class="tabpane hidden">
<div class="card"><div class="card-head"><h2>转发日志（成功提交）</h2><div class="row"><button class="btn danger sm" onclick="clearLogs()">清空日志</button><button class="btn ghost sm" onclick="loadLogs()">刷新</button></div></div><div id="logTable"></div></div>
</div>

<div id="tab-accounts" class="tabpane hidden">
<div class="card"><div class="card-head"><h2>插件账号</h2><div class="row"><button class="btn primary sm" onclick="openAccModal()">添加账号</button><button class="btn ghost sm" onclick="openAccBatch()">批量导入</button><button class="btn ghost sm" onclick="loadAccounts()">刷新</button></div></div>
<div class="hint">插件账号 = 给「解码插件」客户端登录用的账号（accountId + token），每个账号下面挂一批接口1/接口2任务账号去抢任务</div>
<div id="accountList"></div></div>
</div>

<div id="tab-devices" class="tabpane hidden">
<div class="card"><div class="card-head"><h2>设备</h2><button class="btn ghost sm" onclick="loadDevices()">刷新</button></div><div id="deviceList"></div></div>
</div>

<div id="tab-auth" class="tabpane hidden">
<div class="hint">授权账号 = 接口1/接口2 上游平台允许使用的任务账号白名单（用户名清单，用于账号备注展示与校验）<button class="btn ghost sm" onclick="syncAuthFromPlugins()">从插件账号同步</button></div>
<div class="card"><div class="card-head"><h2>接口1 授权账号</h2><button class="btn primary sm" onclick="saveAuth(1)">保存</button></div>
<div class="hint">每行一条：<code>用户名|备注</code>（备注可省略，也支持空格/tab分隔）</div><textarea id="api1Text" rows="10"></textarea></div>
<div class="card"><div class="card-head"><h2>接口2 授权账号</h2><button class="btn primary sm" onclick="saveAuth(2)">保存</button></div>
<div class="hint">每行一条：<code>用户名|备注</code>（备注可省略）</div><textarea id="api2Text" rows="10"></textarea></div>
</div>

<div id="tab-sub" class="tabpane hidden">
<div class="hint">子账号 = 给「旧版 App」客户端登录<b>本后台</b>用的账号（用户名+密码）。App 用它登录这里后，通过本后台转发去抢接口1任务（take_r）和回传（submit）。每个子账号可单独设权限、每天最多抢几条、绑定哪个设备。如果你不用旧版 App，这一页可以不管。</div>
<div class="card"><div class="card-head"><h2>子账号</h2><button class="btn primary sm" onclick="saveSubs()">保存</button></div>
<div class="hint">每行一条：<code>用户名|密码|备注|启用(1/0)|权限|每日限额|绑定设备</code>，权限逗号分隔（take_r,submit,fetch_submit,view_logs）；也支持空格/tab分隔</div>
<textarea id="subText" rows="16"></textarea></div>
</div>

<div id="tab-update" class="tabpane hidden">
<div class="card"><div class="card-head"><h2>App 更新配置</h2><div class="row"><button class="btn primary sm" onclick="uploadApk()">上传APK</button><button class="btn ghost sm" onclick="saveUpdate()">仅保存</button></div></div>
<div class="hint">选择新版本 APK 文件，填好「版本号（数字）」后点「上传APK」；App 登录后会检测到更高版本号并提示在线更新。</div>
<div class="form-grid">
<label class="full">APK 文件<input id="upApk" type="file" accept=".apk"></label>
<label>版本号（数字）<input id="upVerCode" type="number"></label>
<label>版本名<input id="upVerName" type="text"></label>
<label class="full">APK 下载地址<input id="upUrl" type="text"></label>
<label>更新标题<input id="upTitle" type="text"></label>
<label class="check"><input id="upForce" type="checkbox"> 强制更新</label>
<label class="full">更新说明<textarea id="upMsg" rows="3"></textarea></label>
</div></div>
<div class="card"><div class="card-head"><h2>版本控制</h2><button class="btn primary sm" onclick="saveVer()">保存</button></div>
<div class="form-grid">
<label>最低版本号（数字）<input id="vrMinCode" type="number"></label>
<label>最低版本提示<input id="vrMinMsg" type="text"></label>
</div></div>
<div class="card"><div class="card-head"><h2>插件更新配置</h2><button class="btn primary sm" onclick="savePluginUpdate()">保存</button></div>
<div class="form-grid">
<label class="check"><input id="plEnabled" type="checkbox"> 启用插件更新</label>
<label>版本号（数字）<input id="plVerCode" type="number"></label>
<label>版本名<input id="plVerName" type="text"></label>
<label class="full">插件包下载地址<input id="plUrl" type="text"></label>
<label>更新标题<input id="plTitle" type="text"></label>
<label class="check"><input id="plForce" type="checkbox"> 强制更新</label>
<label class="full">更新说明<textarea id="plMsg" rows="3"></textarea></label>
</div></div>
</div>

<div id="tab-cache" class="tabpane hidden">
<div class="card"><div class="card-head"><h2>缓存概览</h2><div class="row"><button class="btn ghost sm" onclick="loadCache()">刷新</button><button class="btn danger sm" onclick="clearCache()">清空查询缓存</button></div></div><div id="cacheView"></div></div>
</div>

<div id="tab-inv" class="tabpane hidden">
<div class="inv-layout"><div class="inv-left">
<div class="card"><div class="card-head"><h2>任务库存模式</h2><div class="row"><label class="check"><input type="checkbox" id="invEnabled"> 总开关</label></div></div>
<div class="hint">拉取 / 回传都走接口1，与旧功能隔离。开启后 App/插件直接走任务模式接口（GET /inventory/task）从全局库存取任务</div>
<div class="card-head" style="margin-top:4px"><h2>全局抢任务配置</h2><div class="row"><button class="btn primary sm" onclick="saveInv()">保存</button><button class="btn ghost sm" onclick="loadInv()">刷新</button></div></div>
<div class="form-grid">
<label>库存（保持多少条在库）<input id="gStock" type="number"></label>
<label>并发（每轮用几个子账号同时抢）<input id="gCon" type="number"></label>
<label>间隔秒（每几秒补一次库存）<input id="gInt" type="number"></label>
<label>超时秒（多久没回传就丢弃）<input id="gTimeout" type="number"></label>
<label>任务类型<select id="gType"><option value="">普通任务</option><option value="r">R任务</option></select></label>
</div>
<div id="invStatus"></div>
</div>
<div class="card"><div class="card-head"><h2>库存单价（成功一条的金额）</h2><button class="btn primary sm" onclick="saveInvPrices()">保存单价</button></div>
<div class="hint">每行一条：<code>下发id|单价</code>（例如 <code>C|0.05</code>），金额 = 今日成功 × 单价</div>
<textarea id="invPricesText" rows="6"></textarea>
</div>
<div class="card"><div class="card-head"><h2>审核失败分配（实时）</h2><button class="btn ghost sm" onclick="loadInvAudit()">刷新</button></div>
<div class="hint">失败总数 − 已分配 = 剩余，剩余平均分配到 cx.php 下每个账号</div>
<div id="invAudit"></div>
</div>
</div>
<div class="inv-right">
<div class="card"><div class="card-head"><h2>抢任务明细</h2><button class="btn ghost sm" onclick="loadInvDetail()">刷新</button></div><div id="invFetchLog"></div></div>
<div class="card"><div class="card-head"><h2>库存明细</h2></div><div id="invQueueDetail"></div></div>
<div class="card"><div class="card-head"><h2>回传明细（上游提交结果）</h2><button class="btn ghost sm" onclick="loadInvSubmitLogs()">刷新</button></div><div id="invSubmitLog"></div></div>
</div>
</div>
<div class="card"><div class="card-head"><h2>接口1 子账号表（抢任务用，可上千个）</h2><div class="row"><button class="btn primary sm" onclick="saveInvSub()">保存</button><button class="btn ghost sm" onclick="loadInvSub()">刷新</button><span class="muted" id="invSubCount"></span></div></div>
<div class="hint">每行一条：<code>用户名 密码</code>；支持 <code>|</code>、空格、Tab 分隔，可直接从 Excel 复制粘贴（所有账号全局轮询，不分下发id）</div>
<textarea id="invSubTable" rows="14"></textarea>
</div>
</div>
</div>

</div>
</main>
</div>
<div id="accModal" class="modal hidden">
<div class="modal-box">
<div class="card-head"><h2 id="accModalTitle">添加插件账号</h2><button class="btn ghost sm" onclick="closeAccModal()">关闭</button></div>
<div class="form-grid">
<label>账号ID（留空自动生成）<input id="accId" type="text"></label>
<label>备注<input id="accRemark" type="text"></label>
<label class="full">接口1 任务账号（每行：用户名|密码|备注）<textarea id="accApi1" rows="4"></textarea></label>
<label class="full">接口2 任务账号（可选，每行：用户名|密码|备注）<textarea id="accApi2" rows="3"></textarea></label>
<label>任务模式<select id="accMode"><option value="poll">轮询</option><option value="concurrent">并发</option></select></label>
<label>并发数<input id="accPara" type="number" value="1"></label>
<label class="check"><input id="accEnabled" type="checkbox" checked> 启用</label>
</div>
<div class="row" style="margin-top:14px"><button class="btn primary" onclick="saveAcc()">保存</button><button class="btn ghost" onclick="closeAccModal()">取消</button></div>
</div>
</div>
<div id="logDetailModal" class="modal hidden">
<div class="modal-box">
<div class="card-head"><h2>提交明细</h2><button class="btn ghost sm" onclick="closeLogDetail()">关闭</button></div>
<div id="logDetailBody"></div>
</div>
</div>
<div id="toasts"></div>
<script>
const BASE=location.pathname;let KEY=localStorage.getItem('admin_key')||'';
function esc(s){return String(s==null?'':s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
function toast(msg,type){type=type||'info';var box=document.getElementById('toasts');var t=document.createElement('div');t.className='toast '+type;t.textContent=msg;box.appendChild(t);setTimeout(function(){t.classList.add('out');setTimeout(function(){t.remove();},350);},2600);}
function api(act,opts){opts=opts||{};if(!opts.headers)opts.headers={};opts.headers['X-Auth-Key']=KEY;if(opts.body&&typeof opts.body==='object'&&!(opts.body instanceof FormData)){opts.headers['Content-Type']='application/json';opts.body=JSON.stringify(opts.body);}return fetch(BASE+'?act='+act,opts).then(function(r){return r.json();});}
function showApp(){document.getElementById('loginView').classList.add('hidden');document.getElementById('adminView').classList.remove('hidden');}
var THEMES=['blue','purple','teal','light'];var currentTab='ov';
function applyTheme(t){document.documentElement.setAttribute('data-theme',t==='blue'?'':t);localStorage.setItem('admin_theme',t);}
function cycleTheme(){var t=localStorage.getItem('admin_theme')||'blue';var i=THEMES.indexOf(t);applyTheme(THEMES[(i+1)%THEMES.length]);}
applyTheme(localStorage.getItem('admin_theme')||'blue');
setInterval(function(){if(!KEY)return;if(currentTab==='ov')loadOverview();else if(currentTab==='logs')loadLogs();else if(currentTab==='accounts')loadAccounts();else if(currentTab==='devices')loadDevices();else if(currentTab==='cache')loadCache();},10000);
function logout(){localStorage.removeItem('admin_key');location.reload();}
function doLogin(){KEY=document.getElementById('keyInput').value.trim();if(!KEY){toast('请输入后台密码','error');return;}api('logs&limit=1').then(function(d){if(d.ok===false){toast(d.msg||'密码错误','error');return;}localStorage.setItem('admin_key',KEY);showApp();switchTab('ov');});}
function switchTab(t){currentTab=t;var ps=document.querySelectorAll('.tabpane');for(var i=0;i<ps.length;i++)ps[i].classList.add('hidden');var pane=document.getElementById('tab-'+t);if(pane)pane.classList.remove('hidden');var ns=document.querySelectorAll('.sidebar .nav');for(var j=0;j<ns.length;j++)ns[j].classList.toggle('active',ns[j].getAttribute('data-t')===t);var titles={ov:'总览',logs:'日志',accounts:'插件账号',devices:'设备',auth:'授权账号',sub:'子账号',update:'更新配置',cache:'缓存',inv:'库存模式'};document.getElementById('topTitle').textContent=titles[t]||'';if(t==='ov')loadOverview();if(t==='logs')loadLogs();if(t==='accounts')loadAccounts();if(t==='devices')loadDevices();if(t==='auth'){loadAuth(1);loadAuth(2);}if(t==='sub')loadSubs();if(t==='update'){loadUpdate();loadVer();loadPluginUpdate();}if(t==='cache')loadCache();if(t==='inv'){loadInv();loadInvSub();loadInvDetail();loadInvAudit();loadInvSubmitLogs();if(invTimer)clearInterval(invTimer);invTimer=setInterval(function(){loadInvDetail();loadInvAudit();loadInvSubmitLogs();},3000);}else{if(invTimer){clearInterval(invTimer);invTimer=null;}}}
function statCard(l,v,c){return '<div class="stat" style="border-top-color:'+c+'"><div class="s-l">'+esc(l)+'</div><div class="s-v" style="color:'+c+'">'+v+'</div></div>';}
function loadOverview(){Promise.all([api('devices'),api('decoded_ext_accounts'),api('logs&limit=200'),api('sub_accounts'),api('inventory_status'),api('cache_overview')]).then(function(rs){var dev=rs[0],acc=rs[1],logs=rs[2],subs=rs[3],inv=rs[4],cache=rs[5];var accRows=acc.data||[],logRows=logs.data||[],subRows=subs.data||[],invRows=inv.data||[];var accOn=accRows.filter(function(a){return a.enabled;}).length;var invQ=0,invOk=0,invFail=0;for(var i=0;i<invRows.length;i++){invQ+=invRows[i].queue||0;invOk+=invRows[i].success||0;invFail+=invRows[i].fail||0;}var items=(cache.data&&cache.data.items)||[];var cFiles=0,cBytes=0;for(var j=0;j<items.length;j++){cFiles+=items[j].files||0;cBytes+=items[j].bytes||0;}document.getElementById('ovCards').innerHTML=statCard('在线设备',(dev.online_count||0)+' 台','#60a5fa')+statCard('插件账号',accOn+'/'+accRows.length+' 启用','#818cf8')+statCard('子账号',subRows.length+' 个','#22d3ee')+statCard('成功转发',logRows.length+' 条','#34d399')+statCard('库存·在库',invQ+' 条','#a78bfa')+statCard('库存·今日成功',invOk+' 条','#34d399')+statCard('库存·今日失败',invFail+' 条','#f87171')+statCard('缓存占用',(cBytes/1048576).toFixed(1)+' MB','#fbbf24');var recent=logRows.slice(-8).reverse();document.getElementById('ovRecent').innerHTML=recent.length?('<table class="tbl"><thead><tr><th>时间</th><th>账号</th><th>备注</th></tr></thead><tbody>'+recent.map(function(l){return '<tr><td class="mono">'+esc(l.time)+'</td><td>'+esc(l.username)+'</td><td>'+esc(l.account_remark||'-')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无日志</div>';});}
var LOG_ROWS=[];
function loadLogs(){api('logs&limit=200').then(function(d){var rows=(d.data||[]).slice().reverse();LOG_ROWS=rows;document.getElementById('logTable').innerHTML=rows.length?('<table class="tbl"><thead><tr><th>时间</th><th>状态</th><th>下发id</th><th>子账号</th><th>来源</th><th>响应</th></tr></thead><tbody>'+rows.map(function(l,i){var sub=l.task_account_username||l.sub_username||'';return '<tr><td class="mono">'+esc(l.time)+'</td><td><span class="badge ok">'+esc(l.status||'成功')+'</span></td><td>'+esc(l.username||'-')+'</td><td>'+esc(sub||'-')+'</td><td>'+esc(l.source||'-')+'</td><td>'+(l.response?'<button class="btn sm" onclick="showLogDetail('+i+')">查看</button>':'-')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无日志</div>';});}
function showLogDetail(i){var l=LOG_ROWS[i];if(!l)return;var resp='';var r=l.response;if(typeof r==='string'){try{resp=JSON.stringify(JSON.parse(r),null,2);}catch(e){resp=r;}}else{try{resp=JSON.stringify(r,null,2);}catch(e){resp=String(r);}}var html='<div class="item-sub">下发id：<b>'+esc(l.username||'-')+'</b></div><div class="item-sub">子账号：<b>'+esc(l.task_account_username||l.sub_username||'-')+'</b></div><div class="item-sub">task_id：'+esc(l.task_id||'-')+'</div><div class="item-sub">task_url：'+esc(l.task_url||'-')+'</div><div class="item-sub" style="margin-top:10px">上游响应：</div><pre class="log-json">'+esc(resp||'-')+'</pre>';document.getElementById('logDetailBody').innerHTML=html;document.getElementById('logDetailModal').classList.remove('hidden');}
function closeLogDetail(){document.getElementById('logDetailModal').classList.add('hidden');}
function clearLogs(){if(!confirm('确定清空日志和数据？'))return;api('clear').then(function(d){toast(d.msg||'已清空','success');loadLogs();});}
function loadAccounts(){api('decoded_ext_accounts').then(function(d){var rows=d.data||[];ACCS=rows;document.getElementById('accountList').innerHTML=rows.length?rows.map(function(a){return '<div class="item"><div class="item-head"><b>'+esc(a.accountId)+'</b><span class="badge '+(a.enabled?'ok':'off')+'">'+(a.enabled?'启用':'停用')+'</span><span class="muted">'+esc(a.remark||'')+'</span></div><div class="item-sub">任务账号 '+esc(a.task_usernames||'-')+' · 模式 '+esc(a.task_mode||'-')+' · 并发 '+esc(a.task_parallelism||'-')+'</div><div class="item-actions"><button class="btn sm" onclick="editAcc(\''+esc(a.accountId)+'\')">编辑</button><button class="btn sm" onclick="toggleAcc(\''+esc(a.accountId)+'\','+(a.enabled?0:1)+')">'+(a.enabled?'停用':'启用')+'</button><button class="btn sm" onclick="resetToken(\''+esc(a.accountId)+'\')">重置Token</button><button class="btn sm danger" onclick="delAcc(\''+esc(a.accountId)+'\')">删除</button></div></div>';}).join(''):'<div class="empty">暂无插件账号</div>';});}
function toggleAcc(id,en){api('decoded_ext_account_toggle&accountId='+encodeURIComponent(id)+'&enabled='+en).then(function(){loadAccounts();});}
function resetToken(id){if(!confirm('重置 '+id+' 的 Token？'))return;api('decoded_ext_account_token&accountId='+encodeURIComponent(id)).then(function(d){toast(d.ok?'已重置':'失败',d.ok?'success':'error');loadAccounts();});}
function delAcc(id){if(!confirm('删除 '+id+'？'))return;api('decoded_ext_account_delete&accountId='+encodeURIComponent(id)).then(function(){toast('已删除','success');loadAccounts();});}
var DEV_ROWS=[];
function loadDevices(){api('devices').then(function(d){var rows=d.data||[];DEV_ROWS=rows;var groups={},order=[];rows.forEach(function(x){var g=x.account_remark||x.username||'-';if(!groups[g]){groups[g]={name:g,online:0,total:0,list:[]};order.push(g);}groups[g].total++;if(x.online)groups[g].online++;groups[g].list.push(x);});document.getElementById('deviceList').innerHTML=order.length?order.map(function(g){var gr=groups[g];var tag=gr.online>0?'ok':'blue';var last='';for(var i=0;i<gr.list.length;i++){if(gr.list[i].last_seen>last)last=gr.list[i].last_seen;}return '<div class="item"><div class="item-head"><b>'+esc(gr.name)+'</b><span class="badge '+tag+'">'+(tag==='ok'?'在线':'离线')+'</span><span class="muted">在线 '+gr.online+' 台 / 共 '+gr.total+' 台</span></div><div class="item-sub">最近心跳 '+esc(last||'-')+'</div><div class="item-actions"><button class="btn sm" onclick="toggleDevGroup(\''+esc(gr.name)+'\',1)">禁用该下发id</button><button class="btn sm" onclick="toggleDevGroupExpand(this)">设备明细</button></div><div class="dev-group hidden">'+gr.list.map(function(x){var dk=x.device_key||x.key||'';var dot=x.disabled?'red':(x.online?'green':'off');var st=x.disabled?'禁用':(x.online?'在线':'离线');return '<div class="dev-sub"><span class="dot '+dot+'"></span><b>'+esc(x.device_label||x.device_id||'-')+'</b><span class="badge '+(x.disabled?'off':(x.online?'ok':'blue'))+'">'+st+'</span><span class="muted">'+esc(x.username||'-')+' · '+esc(x.last_seen||'')+'</span><button class="btn sm" onclick="toggleDev(\''+esc(dk)+'\','+(x.disabled?0:1)+')">'+(x.disabled?'启用':'禁用')+'</button></div>';}).join('')+'</div></div>';}).join(''):'<div class="empty">暂无设备</div>';});}
function toggleDevGroup(g,dis){var us={};DEV_ROWS.forEach(function(x){var n=x.account_remark||x.username||'-';if(n===g&&x.username)us[x.username]=1;});var ks=Object.keys(us);if(!ks.length){loadDevices();return;}var done=0;ks.forEach(function(u){api('device_toggle_user&username='+encodeURIComponent(u)+'&disabled='+dis).then(function(){done++;if(done===ks.length){toast('已处理','success');loadDevices();}});});}
function toggleDevGroupExpand(el){var gr=el.closest('.item').querySelector('.dev-group');if(gr)gr.classList.toggle('hidden');}
function toggleDev(dk,dis){api('device_toggle&device_key='+encodeURIComponent(dk)+'&disabled='+dis+'&reason=').then(function(){loadDevices();});}
function toggleDevUser(u,dis){api('device_toggle_user&username='+encodeURIComponent(u)+'&disabled='+dis).then(function(d){toast(d.msg||'已处理','success');loadDevices();});}
function loadAuth(t){api(t===1?'api1_accounts':'api2_accounts').then(function(d){var rows=d.data||[];document.getElementById('api'+(t===1?1:2)+'Text').value=rows.map(function(a){return a.username+(a.remark?'|'+a.remark:'');}).join('\n');});}
function syncAuthFromPlugins(){api('sync_auth_from_plugins').then(function(d){var r=d.data||{};toast('已同步：新增 '+r.added+'，更新 '+r.updated,'success');loadAuth(1);loadAuth(2);});}
function saveAuth(t){var text=document.getElementById('api'+(t===1?1:2)+'Text').value;api(t===1?'save_api1_accounts':'save_api2_accounts',{method:'POST',body:{accounts:text}}).then(function(d){toast(d.ok?'已保存':'保存失败',d.ok?'success':'error');});}
function loadSubs(){api('sub_accounts').then(function(d){var rows=d.data||[];document.getElementById('subText').value=rows.map(function(a){return a.username+'|'+(a.password_set?'(已设密码)':'')+'|'+a.remark+'|'+(a.enabled?'1':'0')+'|'+(Object.keys(a.permissions).filter(function(k){return a.permissions[k];}).join(','))+'|'+a.daily_take_limit+'|'+a.bind_device;}).join('\n');});}
function saveSubs(){var text=document.getElementById('subText').value;api('save_sub_accounts',{method:'POST',body:{accounts:text}}).then(function(d){toast(d.ok?('已保存 '+d.count+' 个'):(d.msg||'保存失败'),d.ok?'success':'error');});}
function loadUpdate(){api('update_config').then(function(d){var c=d.data||{};document.getElementById('upVerCode').value=c.version_code||'';document.getElementById('upVerName').value=c.version_name||'';document.getElementById('upUrl').value=c.apk_url||'';document.getElementById('upTitle').value=c.title||'';document.getElementById('upMsg').value=c.message||'';document.getElementById('upForce').checked=!!c.force;});}
function saveUpdate(){var q=[];q.push('version_code='+encodeURIComponent(document.getElementById('upVerCode').value));q.push('version_name='+encodeURIComponent(document.getElementById('upVerName').value));q.push('apk_url='+encodeURIComponent(document.getElementById('upUrl').value));q.push('title='+encodeURIComponent(document.getElementById('upTitle').value));q.push('message='+encodeURIComponent(document.getElementById('upMsg').value));q.push('force='+(document.getElementById('upForce').checked?'1':'0'));api('save_update&'+q.join('&')).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');loadUpdate();});}
function uploadApk(){var f=document.getElementById('upApk').files[0];if(!f){toast('请先选择 .apk 文件','error');return;}var fd=new FormData();fd.append('apk',f);fd.append('version_code',document.getElementById('upVerCode').value);fd.append('version_name',document.getElementById('upVerName').value);fd.append('title',document.getElementById('upTitle').value||'发现新版本');fd.append('message',document.getElementById('upMsg').value||'');fd.append('force',document.getElementById('upForce').checked?'1':'0');api('upload_update',{method:'POST',body:fd}).then(function(d){toast(d.msg||'已上传',d.ok?'success':'error');loadUpdate();});}
function loadVer(){api('version_controls').then(function(d){var c=d.data||{};document.getElementById('vrMinCode').value=c.min_version_code||'';document.getElementById('vrMinMsg').value=c.min_version_msg||'';});}
function saveVer(){var q='save_version_controls&min_version_code='+encodeURIComponent(document.getElementById('vrMinCode').value)+'&min_version_msg='+encodeURIComponent(document.getElementById('vrMinMsg').value);api(q).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');});}
function loadPluginUpdate(){api('plugin_update_config').then(function(d){var c=d.data||{};document.getElementById('plEnabled').checked=!!c.enabled;document.getElementById('plVerCode').value=c.version_code||'';document.getElementById('plVerName').value=c.version_name||'';document.getElementById('plUrl').value=c.package_url||'';document.getElementById('plTitle').value=c.title||'';document.getElementById('plMsg').value=c.message||'';document.getElementById('plForce').checked=!!c.force;});}
function savePluginUpdate(){var q=[];q.push('enabled='+(document.getElementById('plEnabled').checked?'1':'0'));q.push('version_code='+encodeURIComponent(document.getElementById('plVerCode').value));q.push('version_name='+encodeURIComponent(document.getElementById('plVerName').value));q.push('package_url='+encodeURIComponent(document.getElementById('plUrl').value));q.push('title='+encodeURIComponent(document.getElementById('plTitle').value));q.push('message='+encodeURIComponent(document.getElementById('plMsg').value));q.push('force='+(document.getElementById('plForce').checked?'1':'0'));api('save_plugin_update&'+q.join('&')).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');loadPluginUpdate();});}
function parseTaskLines(text){var out=[];text.split('\n').forEach(function(line){line=line.trim();if(!line||line[0]==='#')return;var p;if(line.indexOf('|')>=0){p=line.split('|');}else if(line.indexOf('\t')>=0){p=line.split('\t');}else{p=line.split(/\s+/);}var u=(p[0]||'').trim(),pw=(p[1]||'').trim(),rm=(p.slice(2).join(' ')).trim();if(u&&pw)out.push({username:u,password:pw,remark:rm});});return out;}
var ACCS=[];var accEditing=false;
function openAccModal(){accEditing=false;document.getElementById('accModalTitle').textContent='添加插件账号';document.getElementById('accId').value='';document.getElementById('accRemark').value='';document.getElementById('accApi1').value='';document.getElementById('accApi2').value='';document.getElementById('accMode').value='poll';document.getElementById('accPara').value='1';document.getElementById('accEnabled').checked=true;document.getElementById('accModal').classList.remove('hidden');}
function editAcc(id){api('decoded_ext_account_detail&accountId='+encodeURIComponent(id)).then(function(d){var a=d.data;if(!a){toast('未找到账号','error');return;}accEditing=true;document.getElementById('accModalTitle').textContent='编辑插件账号：'+a.accountId;document.getElementById('accId').value=a.accountId;document.getElementById('accRemark').value=a.remark||'';document.getElementById('accApi1').value=(a.api1_accounts||[]).map(function(x){return x.username+'|'+(x.password||'')+'|'+(x.remark||'');}).join('\n');document.getElementById('accApi2').value=(a.api2_accounts||[]).map(function(x){return x.username+'|'+(x.password||'')+'|'+(x.remark||'');}).join('\n');document.getElementById('accMode').value=a.task_mode==='concurrent'?'concurrent':'poll';document.getElementById('accPara').value=a.task_parallelism||1;document.getElementById('accEnabled').checked=a.enabled!==false;document.getElementById('accModal').classList.remove('hidden');});}
function closeAccModal(){document.getElementById('accModal').classList.add('hidden');}
function saveAcc(){var api1=parseTaskLines(document.getElementById('accApi1').value);var api2=parseTaskLines(document.getElementById('accApi2').value);if(!accEditing&&!api1.length&&!api2.length){toast('请至少配置一个任务账号','error');return;}var body={accountId:document.getElementById('accId').value.trim(),remark:document.getElementById('accRemark').value.trim(),task_mode:document.getElementById('accMode').value,task_parallelism:parseInt(document.getElementById('accPara').value)||1,enabled:document.getElementById('accEnabled').checked};if(api1.length)body.api1_accounts=api1;if(api2.length)body.api2_accounts=api2;api('decoded_ext_account_save',{method:'POST',body:body}).then(function(d){toast(d.ok?'已保存':'失败：'+(d.msg||''),d.ok?'success':'error');if(d.ok){closeAccModal();loadAccounts();}});}
function openAccBatch(){var t=prompt('批量导入插件账号：每行一个「用户名|密码|备注」，系统会自动生成账号ID和Token');if(!t)return;api('decoded_ext_account_batch',{method:'POST',body:{batch:t}}).then(function(d){toast('已导入 '+d.count+' 个账号',d.ok?'success':'error');loadAccounts();});}
function loadCache(){api('cache_overview').then(function(d){var items=(d.data&&d.data.items)||[];document.getElementById('cacheView').innerHTML='<table class="tbl"><thead><tr><th>路径</th><th>文件数</th><th>大小</th></tr></thead><tbody>'+items.map(function(i){return '<tr><td>'+esc(i.path)+'</td><td>'+i.files+'</td><td>'+(i.bytes/1024).toFixed(1)+' KB</td></tr>';}).join('')+'</tbody></table>';});}
function clearCache(){fetch('../cx.php?act=clear_cache&json=1').then(function(r){return r.json();}).then(function(d){toast('已清理 '+d.deleted+' 个文件','success');loadCache();});}
function loadInv(){api('inventory_config').then(function(d){var c=d.data||{};document.getElementById('invEnabled').checked=!!c.enabled;document.getElementById('gStock').value=c.stock||100;document.getElementById('gCon').value=c.concurrency||50;document.getElementById('gInt').value=c.interval_seconds||5;document.getElementById('gTimeout').value=c.timeout_seconds||300;document.getElementById('gType').value=c.take_type||'';});api('inventory_status').then(function(d){var accs=d.accounts||[];var prices=d.prices||{};var accRows=accs.map(function(a){var p=prices[a.id]||0;var amt=a.amount||0;return '<tr><td>'+esc(a.id)+(a.unassigned?' <span class="badge amber">未归属</span>':'')+'</td><td style="color:var(--green)">'+a.success+'</td><td style="color:var(--red)">'+a.fail+'</td><td>'+p+'</td><td style="color:var(--cyan)">¥'+amt+'</td></tr>';}).join('');document.getElementById('invStatus').innerHTML='<div class="hint" style="margin-top:12px">总开关：'+(d.enabled?'<span class="badge ok">已开启</span>':'<span class="badge off">已关闭</span>')+' · 日期：'+esc(d.date)+' · 今日金额 ¥'+(d.total_amount||0)+'</div><table class="tbl" style="margin-top:8px"><thead><tr><th>在库</th><th>未回传</th><th>今日拉取</th></tr></thead><tbody><tr><td>'+d.queue+'</td><td>'+d.outstanding+'</td><td>'+d.fetched+'</td></tr></tbody></table>'+(accRows?'<div class="card-head" style="margin-top:12px"><h2>各下发id今日统计</h2></div><table class="tbl"><thead><tr><th>下发id</th><th>今日成功</th><th>今日失败</th><th>单价</th><th>金额</th></tr></thead><tbody>'+accRows+'</tbody></table>':'');});loadInvPrices();}
function loadInvPrices(){api('inventory_prices').then(function(d){var p=d.prices||{};document.getElementById('invPricesText').value=Object.keys(p).sort().map(function(k){return k+'|'+p[k];}).join('\n');});}
function saveInvPrices(){var text=document.getElementById('invPricesText').value;var prices={};text.split('\n').forEach(function(line){line=line.trim();if(!line)return;var parts=line.split('|');var k=(parts[0]||'').trim();var v=parseFloat(parts[1]);if(k&&!isNaN(v))prices[k]=v;});api('inventory_prices_save',{method:'POST',body:{prices:prices}}).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');loadInv();});}
function loadInvAudit(){api('inventory_audit').then(function(d){var html='';[['cx','cx.php（Ajie）']].forEach(function(g){var a=d[g[0]]||{};var accs=a.accounts||[];if(!accs.length){return;}var rows=accs.map(function(x){return '<tr><td>'+esc(x.username)+'</td><td>'+esc(x.remark||'-')+'</td><td style="color:var(--red)">'+x.own+'</td><td style="color:var(--amber)">'+x.audit+'</td><td style="color:var(--red)">'+x.total+'</td></tr>';}).join('');var cnt=a.count||0;var rmd=a.remainingFail||0;var remExtra=cnt>0?(rmd%cnt):0;html+='<div class="card-head" style="margin-top:10px"><h2>'+g[1]+'</h2><span class="muted">'+esc(a.date||d.date)+'</span></div><div class="hint" style="margin-top:0">失败总数 '+a.totalFail+' − 已分配 '+a.allocatedFail+' = 剩余 <b style="color:var(--amber)">'+rmd+'</b>，均摊到 '+cnt+' 个账号（每人 '+a.perAccount+'，前 '+remExtra+' 个多 1）</div><table class="tbl"><thead><tr><th>账号</th><th>备注</th><th>已分配失败</th><th>审核失败(均摊)</th><th>合计</th></tr></thead><tbody>'+rows+'</tbody></table>';});document.getElementById('invAudit').innerHTML=html||'<div class="empty">暂无数据（等待 cx 查询写入数据）</div>';});}
function loadInvSub(){api('inventory_sub_table').then(function(d){document.getElementById('invSubTable').value=d.data||'';document.getElementById('invSubCount').textContent='共 '+d.count+' 个子账号';});}
function saveInvSub(){var text=document.getElementById('invSubTable').value;api('inventory_sub_table_save',{method:'POST',body:{accounts:text}}).then(function(d){toast(d.msg||(d.ok?'已保存':'失败'),d.ok?'success':'error');loadInvSub();});}
var invTimer=null;
function loadInvDetail(){api('inventory_detail').then(function(d){var q=d.queue||[],o=d.outstanding||[],evs=d.events||[];var evHtml=evs.length?('<table class="tbl"><thead><tr><th>时间</th><th>子账号</th><th>结果</th></tr></thead><tbody>'+evs.map(function(e){return '<tr><td class="mono">'+esc(e.time)+'</td><td>'+esc(e.sub)+'</td><td>'+(e.ok?'<span class="badge ok">抢到 '+esc(e.task_id||'')+'</span>':'<span class="badge off">'+esc(e.note||'失败')+'</span>')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无抢任务记录（开启总开关后才会出现）</div>';document.getElementById('invFetchLog').innerHTML=evHtml;var rows='';q.forEach(function(t){rows+='<tr><td class="mono" style="max-width:110px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(t.id)+'</td><td><span class="badge blue">在库</span></td><td>'+esc(t.sub)+'</td><td class="mono">'+esc(t.fetched_at)+'</td></tr>';});o.forEach(function(t){rows+='<tr><td class="mono" style="max-width:110px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(t.id)+'</td><td><span class="badge amber">已取·'+esc(t.account_id||'')+'</span></td><td>'+esc(t.sub)+'</td><td class="mono">'+esc(t.fetched_at)+'</td></tr>';});document.getElementById('invQueueDetail').innerHTML=rows?('<table class="tbl"><thead><tr><th>任务ID</th><th>状态</th><th>子账号</th><th>时间</th></tr></thead><tbody>'+rows+'</tbody></table>'):'<div class="empty">暂无库存</div>';});}
function loadInvSubmitLogs(){api('inventory_submit_logs&limit=100').then(function(d){var rows=d.data||[];var html=rows.length?('<table class="tbl"><thead><tr><th>时间</th><th>下发id</th><th>任务ID</th><th>结果</th></tr></thead><tbody>'+rows.map(function(l){var st=l.status||'';var badge=st==='success'?'<span class="badge ok">成功</span>':(st==='item_id_mismatch'?'<span class="badge amber">item_id不匹配</span>':(st==='accepted'?'<span class="badge blue">已接收</span>':(st==='fail'?'<span class="badge off">失败</span>':'<span class="badge off">'+esc(st)+'</span>')));return '<tr><td class="mono">'+esc(l.time)+'</td><td>'+esc(l.account_id||l.username||'-')+'</td><td class="mono" style="max-width:130px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(l.task_id||'-')+'</td><td>'+badge+(l.error?'<div class="muted" style="font-size:11px">'+esc(l.error)+'</div>':'')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无回传记录</div>';document.getElementById('invSubmitLog').innerHTML=html;});}
function saveInv(){var cfg={enabled:document.getElementById('invEnabled').checked,stock:parseInt(document.getElementById('gStock').value)||100,concurrency:parseInt(document.getElementById('gCon').value)||50,interval_seconds:parseInt(document.getElementById('gInt').value)||5,timeout_seconds:parseInt(document.getElementById('gTimeout').value)||300,take_type:document.getElementById('gType').value};api('inventory_save',{method:'POST',body:{config:cfg}}).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');if(d.ok)loadInv();});}
function openPwd(){var cur=prompt('当前密码'),nw=prompt('新密码（至少6位）'),cf=prompt('确认新密码');if(!cur||!nw)return;var fd=new FormData();fd.append('act','change_password');fd.append('current_password',cur);fd.append('new_password',nw);fd.append('confirm_password',cf);fetch(BASE,{method:'POST',headers:{'X-Auth-Key':KEY},body:fd}).then(function(r){return r.json();}).then(function(d){toast(d.msg||'修改失败',d.ok?'success':'error');});}
if(KEY){api('logs&limit=1').then(function(d){if(d.ok===false){localStorage.removeItem('admin_key');}else{showApp();switchTab('ov');}});}
</script></body></html>`)
}

func renderPasswordPage(w http.ResponseWriter) {
	writeHTML(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>修改密码</title></head>
<body style="font-family:sans-serif;background:#0f172a;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh">
<form onsubmit="return save()" style="background:#1e293b;padding:32px;border-radius:14px;width:340px">
<h2 style="margin:0 0 16px">修改后台密码</h2>
<input type="password" id="cur" placeholder="当前密码" style="width:100%;height:40px;margin-bottom:10px;border-radius:8px;border:1px solid #334155;background:#0f172a;color:#e2e8f0;padding:0 12px;box-sizing:border-box">
<input type="password" id="nw" placeholder="新密码（至少6位）" style="width:100%;height:40px;margin-bottom:10px;border-radius:8px;border:1px solid #334155;background:#0f172a;color:#e2e8f0;padding:0 12px;box-sizing:border-box">
<input type="password" id="cf" placeholder="确认新密码" style="width:100%;height:40px;margin-bottom:14px;border-radius:8px;border:1px solid #334155;background:#0f172a;color:#e2e8f0;padding:0 12px;box-sizing:border-box">
<button type="submit" style="width:100%;height:40px;border:none;border-radius:8px;background:#6366f1;color:#fff;font-weight:700;cursor:pointer">保存</button>
<div id="msg" style="margin-top:12px;font-size:13px;color:#94a3b8"></div>
</form>
<script>
function save(){const cur=document.getElementById('cur').value,nw=document.getElementById('nw').value,cf=document.getElementById('cf').value;
const fd=new FormData();fd.append('act','change_password');fd.append('current_password',cur);fd.append('new_password',nw);fd.append('confirm_password',cf);
fetch(location.pathname,{method:'POST',headers:{'X-Auth-Key':cur},body:fd}).then(r=>r.json()).then(d=>{document.getElementById('msg').textContent=d.msg||'修改失败';});return false;}
</script></body></html>`)
}

func renderParallelControlPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	tokenJSON := jsonEncode(token)
	writeHTML(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>运行控制</title>
<style>:root{--bg:#f6f8fb;--card:#fff;--text:#172033;--muted:#6b778c;--border:#dfe6f0;--primary:#2563eb;--ok:#10b981;--bad:#e11d48}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:-apple-system,"Segoe UI","PingFang SC",sans-serif;font-size:15px}
.page{min-height:100vh;padding:40px 20px;display:flex;justify-content:center}.shell{width:min(100%,1120px)}
h1{font-size:30px;margin:0 0 8px}.sub{color:var(--muted);margin:0 0 24px}
.layout{display:grid;grid-template-columns:minmax(0,520px) 1fr;gap:24px;align-items:start}
.panel{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:22px;box-shadow:0 18px 48px rgba(15,23,42,.09)}
.badge{display:inline-block;padding:5px 10px;border-radius:999px;font-size:12px;font-weight:700;background:#ecfdf5;color:#047857}
.badge.off{background:#fef2f2;color:#b91c1c}
.metrics{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin:16px 0}.metric{border:1px solid var(--border);border-radius:10px;padding:12px;background:#f8fafc}.metric b{display:block;font-size:22px}.metric span{color:var(--muted);font-size:11px}
input[type=range]{width:100%}input[type=number]{width:100%;height:44px;border:1px solid var(--border);border-radius:10px;padding:0 12px;font-size:16px}
select{width:100%;height:44px;border:1px solid var(--border);border-radius:10px;padding:0 12px}
.btn{width:100%;height:48px;border:none;border-radius:10px;background:linear-gradient(135deg,#2563eb,#14b8a6);color:#fff;font-size:16px;font-weight:800;cursor:pointer;margin-top:16px}
.msg{margin-top:12px;font-size:13px;color:var(--muted)}.msg.ok{color:#047857}.msg.err{color:#b91c1c}
table{width:100%;border-collapse:collapse;font-size:13px}th,td{padding:8px 6px;border-bottom:1px solid #edf2f7;text-align:center}th{color:var(--muted);font-size:11px}
.daily{max-height:420px;overflow:auto}.tag{display:inline-block;padding:2px 8px;border-radius:999px;font-size:11px}.tag.ok{background:#ecfdf5;color:#047857}.tag.bad{background:#fef2f2;color:#b91c1c}
.names{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}.chip{border:1px solid #dbeafe;background:#eff6ff;color:#1e40af;border-radius:999px;padding:5px 9px;font-size:12px}
.loading{padding:40px;text-align:center;color:var(--muted)}.error{color:#b91c1c;padding:20px}
@media(max-width:900px){.layout{grid-template-columns:1fr}}
</style></head><body><div class="page"><div class="shell">
<div><h1>运行控制</h1><p class="sub">切换轮询/并发模式，调整任务并发数。</p></div>
<div class="layout">
<div class="panel" id="panel"><div class="loading">正在读取配置...</div></div>
<div class="panel"><h2 style="margin:0 0 12px;font-size:16px">当日任务数量</h2><div class="daily" id="daily"><div class="loading">加载中...</div></div></div>
</div></div></div>
<script>
const TOKEN=`+tokenJSON+`;
function apiUrl(act){return '?act='+act+'&t='+encodeURIComponent(TOKEN);}
function esc(s){return String(s??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));}
function num(v){return Number(v||0).toLocaleString();}
let current=null;
function render(data){current=data;const taskMode=data.task_mode==='concurrent'?'concurrent':'poll';
const max=Math.max(1,data.max_parallelism||1),val=Math.max(1,Math.min(data.task_parallelism||1,max));
document.getElementById('panel').innerHTML=
'<div class="badge '+(data.enabled?'':'off')+'">'+(data.enabled?'可使用':'已停用')+'</div>'+
'<div class="metrics"><div class="metric"><b>'+(taskMode==='concurrent'?val:'轮询')+'</b><span>当前并发</span></div><div class="metric"><b>'+max+'</b><span>最高可设</span></div><div class="metric"><b>'+data.enabled_task_count+'</b><span>可用账号</span></div></div>'+
'<div><label style="font-size:13px;color:var(--muted)">运行模式</label><select id="taskMode"><option value="poll" '+(taskMode==='poll'?'selected':'')+'>轮询（当前页面跳转）</option><option value="concurrent" '+(taskMode==='concurrent'?'selected':'')+'>并发（新标签页）</option></select></div>'+
(taskMode==='concurrent'?'<div style="margin-top:14px"><label style="font-size:13px;color:var(--muted)">并发数（1 - '+max+'）</label><input type="range" id="range" min="1" max="'+max+'" value="'+val+'" oninput="document.getElementById(\'num\').value=this.value"><input type="number" id="num" min="1" max="'+max+'" value="'+val+'"></div>':'')+
'<div class="names">'+(data.task_usernames||[]).map(n=>'<span class="chip">'+esc(n)+'</span>').join('')+'</div>'+
'<button class="btn" id="saveBtn" '+(data.enabled?'':'disabled')+' onclick="save()">保存设置</button><div class="msg" id="msg"></div>';
}
async function save(){const mode=document.getElementById('taskMode').value;const numEl=document.getElementById('num');const body={task_mode:mode};if(numEl)body.task_parallelism=parseInt(numEl.value,10);
const r=await fetch(apiUrl('parallel_control_save'),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const d=await r.json();
const m=document.getElementById('msg');m.textContent=d.msg||'保存失败';m.className='msg '+(d.ok?'ok':'err');if(d.ok)load();}
function renderDaily(data){const s=data.summary||{},rows=data.rows||[];
let html='<div class="metrics"><div class="metric"><b>'+num(s.take)+'</b><span>总领取</span></div><div class="metric"><b>'+num(s.complete)+'</b><span>总完成</span></div><div class="metric"><b style="color:#047857">'+num(s.success)+'</b><span>总通过</span></div><div class="metric"><b style="color:#e11d48">'+num(s.fail)+'</b><span>总失败</span></div></div>';
if(data.warning)html+='<div class="error" style="font-size:12px">'+esc(data.warning)+'</div>';
html+='<table><thead><tr><th>账号</th><th>备注</th><th>领取</th><th>完成</th><th>通过</th><th>失败</th></tr></thead><tbody>';
html+=rows.map(r=>'<tr><td>'+esc(r.username)+'</td><td>'+esc(r.remark||'')+'</td><td>'+num(r.take)+'</td><td>'+num(r.complete)+'</td><td style="color:#047857">'+num(r.success)+'</td><td style="color:#e11d48">'+num(r.fail)+'</td></tr>').join('');
html+='</tbody></table>';document.getElementById('daily').innerHTML=html;}
async function load(){const r=await fetch(apiUrl('parallel_control_info'));const d=await r.json();if(!d.ok){document.getElementById('panel').innerHTML='<div class="error">'+esc(d.msg||'链接无效')+'</div>';return;}render(d.data);}
async function loadDaily(){const r=await fetch(apiUrl('parallel_control_daily_stats')+'&date='+new Date().toISOString().slice(0,10));const d=await r.json();if(d.ok)renderDaily(d.data);}
load();loadDaily();setInterval(loadDaily,15000);
</script></body></html>`)
}
