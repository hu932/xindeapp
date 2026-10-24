package main

import (
	"net/http"
)

func renderAdminPage(w http.ResponseWriter, r *http.Request, h5 bool) {
	writeHTML(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Ajie 控制台</title>
<style>*{box-sizing:border-box;margin:0;padding:0}
/* ============================================================
   Ajie 控制台 · 设计系统
   参考「青龙面板」：Ant Design 语言，浅色为默认主题
   ============================================================ */
:root,html[data-theme=light]{
--bg:#f5f5f5;--panel:#fff;--card:#fff;--card2:#fafafa;
--line:#f0f0f0;--line2:#d9d9d9;
--text:rgba(0,0,0,.88);--muted:rgba(0,0,0,.65);--muted2:rgba(0,0,0,.45);
--accent:#1677ff;--accent2:#4096ff;--accent-soft:rgba(22,119,255,.1);--accent-strong:#0958d9;--accent-on:#fff;
--green:#237804;--red:#cf1322;--amber:#ad4e00;--blue:#0958d9;--cyan:#006d75;
--green-soft:rgba(56,158,13,.1);--red-soft:rgba(207,19,34,.1);--amber-soft:rgba(212,107,8,.1);--blue-soft:rgba(22,119,255,.1);
--overlay:rgba(0,0,0,.45);--zebra:#fafafa;--row-hover:#fafafa;
--shadow:0 1px 2px 0 rgba(0,0,0,.03),0 1px 6px -1px rgba(0,0,0,.02),0 2px 4px 0 rgba(0,0,0,.02);
--shadow-sm:0 1px 2px rgba(0,0,0,.04);
--focus-ring:#1677ff;color-scheme:light;
--r-sm:6px;--r-md:8px;--r-lg:12px;--r-xl:16px;--r-pill:999px;
--fs-xs:12px;--fs-sm:13px;--fs-base:14px;--fs-md:16px;--fs-lg:20px;--fs-xl:24px;
--sp-1:4px;--sp-2:8px;--sp-3:12px;--sp-4:16px;--sp-5:20px;--sp-6:24px;--sp-7:32px;
}
html[data-theme=dark]{
--bg:#0f1115;--panel:#141821;--card:#1a1f2b;--card2:#202634;
--line:#272e3d;--line2:#39415230;
--text:rgba(255,255,255,.88);--muted:rgba(255,255,255,.65);--muted2:rgba(255,255,255,.45);
--accent:#4096ff;--accent2:#69b1ff;--accent-soft:rgba(64,150,255,.16);--accent-strong:#1668dc;--accent-on:#fff;
--green:#73d13d;--red:#ff7875;--amber:#d89614;--blue:#69b1ff;--cyan:#36cfc9;
--green-soft:rgba(115,209,61,.16);--red-soft:rgba(255,120,117,.16);--amber-soft:rgba(216,150,20,.16);--blue-soft:rgba(105,177,255,.16);
--overlay:rgba(0,0,0,.65);--zebra:rgba(255,255,255,.02);--row-hover:rgba(255,255,255,.04);
--shadow:0 1px 2px 0 rgba(0,0,0,.3),0 2px 8px -2px rgba(0,0,0,.35);
--shadow-sm:0 1px 2px rgba(0,0,0,.3);
--focus-ring:#4096ff;color-scheme:dark;
}
html[data-theme=purple]{
--bg:#12101c;--panel:#181428;--card:#1e1a33;--card2:#262040;
--line:#2f2850;--line2:#3d3468;
--text:rgba(255,255,255,.88);--muted:rgba(255,255,255,.65);--muted2:rgba(255,255,255,.45);
--accent:#8b5cf6;--accent2:#a78bfa;--accent-soft:rgba(139,92,246,.16);--accent-strong:#7c3aed;--accent-on:#fff;
--green:#73d13d;--red:#ff7875;--amber:#d89614;--blue:#a78bfa;--cyan:#36cfc9;
--green-soft:rgba(115,209,61,.16);--red-soft:rgba(255,120,117,.16);--amber-soft:rgba(216,150,20,.16);--blue-soft:rgba(167,139,250,.16);
--overlay:rgba(0,0,0,.65);--zebra:rgba(255,255,255,.02);--row-hover:rgba(255,255,255,.04);
--shadow:0 1px 2px 0 rgba(0,0,0,.3),0 2px 8px -2px rgba(0,0,0,.35);
--shadow-sm:0 1px 2px rgba(0,0,0,.3);
--focus-ring:#a78bfa;color-scheme:dark;
}
html[data-theme=teal]{
--bg:#0b1512;--panel:#101d19;--card:#15241f;--card2:#1c2e28;
--line:#24382f;--line2:#2f4a3e;
--text:rgba(255,255,255,.88);--muted:rgba(255,255,255,.65);--muted2:rgba(255,255,255,.45);
--accent:#13a8a8;--accent2:#36cfc9;--accent-soft:rgba(19,168,168,.16);--accent-strong:#087f7f;--accent-on:#fff;
--green:#73d13d;--red:#ff7875;--amber:#d89614;--blue:#36cfc9;--cyan:#36cfc9;
--green-soft:rgba(115,209,61,.16);--red-soft:rgba(255,120,117,.16);--amber-soft:rgba(216,150,20,.16);--blue-soft:rgba(54,207,201,.16);
--overlay:rgba(0,0,0,.65);--zebra:rgba(255,255,255,.02);--row-hover:rgba(255,255,255,.04);
--shadow:0 1px 2px 0 rgba(0,0,0,.3),0 2px 8px -2px rgba(0,0,0,.35);
--shadow-sm:0 1px 2px rgba(0,0,0,.3);
--focus-ring:#36cfc9;color-scheme:dark;
}
html,body{height:100%}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif;font-size:var(--fs-base);line-height:1.5714;-webkit-font-smoothing:antialiased}
.hidden{display:none!important}
::-webkit-scrollbar{width:8px;height:8px}
::-webkit-scrollbar-thumb{background:var(--line2);border-radius:4px}
::-webkit-scrollbar-thumb:hover{background:var(--muted2)}
::-webkit-scrollbar-track{background:transparent}
a{color:var(--accent);text-decoration:none}
/* ---------- 登录页 ---------- */
.login{position:fixed;inset:0;display:flex;align-items:center;justify-content:center;background:var(--bg);padding:16px}
.login-card{width:380px;max-width:100%;background:var(--card);border-radius:var(--r-lg);padding:40px 36px;text-align:center;box-shadow:0 6px 16px 0 rgba(0,0,0,.08),0 3px 6px -4px rgba(0,0,0,.12),0 9px 28px 8px rgba(0,0,0,.05)}
.logo-big{display:flex;justify-content:center;color:var(--accent)}
.login-card h1{font-size:var(--fs-xl);font-weight:600;margin:16px 0 4px;letter-spacing:0}
.login-card .sub{color:var(--muted);font-size:var(--fs-base);margin-bottom:28px}
.login-card input{width:100%;height:40px;background:var(--card);border:1px solid var(--line2);border-radius:var(--r-md);color:var(--text);padding:0 12px;font-size:var(--fs-base);margin-bottom:16px;transition:border-color .2s,box-shadow .2s}
.login-card input::placeholder{color:var(--muted2)}
.login-card input:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 2px var(--accent-soft)}
/* ---------- 外壳 ---------- */
.app{display:flex;min-height:100vh}
.sidebar{width:200px;flex-shrink:0;background:var(--panel);border-right:1px solid var(--line);display:flex;flex-direction:column;position:sticky;top:0;height:100vh}
.logo{display:flex;align-items:center;gap:10px;font-size:var(--fs-md);font-weight:600;padding:0 20px;height:56px;flex-shrink:0;border-bottom:1px solid var(--line);color:var(--text)}
.logo .dot{width:26px;height:26px;border-radius:var(--r-sm);background:var(--accent-strong);display:flex;align-items:center;justify-content:center;flex-shrink:0}
nav{padding:8px;display:flex;flex-direction:column;gap:2px;flex:1;overflow-y:auto}
.nav{display:flex;align-items:center;gap:10px;width:100%;text-align:left;background:transparent;border:none;border-radius:var(--r-md);color:var(--muted);padding:0 12px;height:40px;font-size:var(--fs-base);font-weight:400;cursor:pointer;font-family:inherit;transition:background .2s,color .2s;white-space:nowrap;overflow:hidden}
.nav:hover{background:var(--row-hover);color:var(--text)}
.nav.active{background:var(--accent-soft);color:var(--accent);font-weight:500}
.nav .ic{display:inline-flex;align-items:center;justify-content:center;width:16px;height:16px;flex-shrink:0}
.nav .ic svg{width:16px;height:16px}
.side-foot{padding:8px;border-top:1px solid var(--line);display:flex;flex-direction:column;gap:2px}
.main{flex:1;min-width:0;display:flex;flex-direction:column;background:var(--bg)}
.topbar{display:flex;align-items:center;justify-content:space-between;height:56px;flex-shrink:0;padding:0 24px;border-bottom:1px solid var(--line);background:var(--panel);position:sticky;top:0;z-index:5;gap:12px}
.top-title{font-size:var(--fs-md);font-weight:600;letter-spacing:0;white-space:nowrap;flex-shrink:0}
.top-actions{display:flex;gap:8px;align-items:center;flex-wrap:wrap;justify-content:flex-end}
.content{padding:24px;width:100%}
/* ---------- 卡片 ---------- */
.card{background:var(--card);border-radius:var(--r-lg);padding:24px;margin-bottom:16px;box-shadow:var(--shadow)}
.card-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;gap:12px;flex-wrap:wrap}
.card-head h2{font-size:var(--fs-md);font-weight:600;letter-spacing:0}
.hint{color:var(--muted);font-size:var(--fs-sm);line-height:1.6;margin:-8px 0 16px}
.hint code{background:var(--card2);border:1px solid var(--line);border-radius:var(--r-sm);padding:1px 6px;color:var(--cyan);font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:var(--fs-xs)}
/* ---------- 按钮（Ant Design） ---------- */
.btn{background:var(--card);border:1px solid var(--line2);border-radius:var(--r-sm);color:var(--text);padding:0 15px;height:32px;display:inline-flex;align-items:center;justify-content:center;gap:6px;cursor:pointer;font:inherit;font-size:var(--fs-base);font-weight:400;transition:all .2s;white-space:nowrap;box-shadow:0 2px 0 rgba(0,0,0,.02)}
.btn:hover{border-color:var(--accent);color:var(--accent)}
.btn:active{transform:translateY(1px)}
.btn.primary{background:var(--accent-strong);border-color:var(--accent-strong);color:var(--accent-on);font-weight:500;box-shadow:0 2px 0 rgba(5,145,255,.1)}
.btn.primary:hover{filter:brightness(1.1);color:var(--accent-on)}
.btn.danger{background:var(--card);border-color:var(--red);color:var(--red)}
.btn.danger:hover{background:var(--red-soft);border-color:var(--red);color:var(--red)}
.btn.ghost{background:transparent;border-color:var(--line2);box-shadow:none}
.btn.ghost:hover{background:var(--row-hover);border-color:var(--accent);color:var(--accent)}
.btn.sm{height:28px;padding:0 12px;font-size:var(--fs-sm);border-radius:var(--r-sm)}
.btn.block{width:100%;height:40px;font-size:var(--fs-base);border-radius:var(--r-md)}
button:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible,.btn:focus-visible,.nav:focus-visible{outline:2px solid var(--focus-ring);outline-offset:2px}
/* ---------- 表格 ---------- */
.tbl{width:100%;border-collapse:collapse}
.tbl th{text-align:left;color:var(--text);font-size:var(--fs-base);font-weight:600;padding:12px 16px;background:var(--card2);border-bottom:1px solid var(--line);white-space:nowrap}
.tbl td{padding:12px 16px;border-bottom:1px solid var(--line);font-size:var(--fs-base);font-variant-numeric:tabular-nums}
.tbl tbody tr{transition:background .2s}
.tbl tbody tr:hover{background:var(--row-hover)}
.tbl tbody tr:nth-child(even){background:var(--zebra)}
.tbl tbody tr:nth-child(even):hover{background:var(--row-hover)}
.tbl thead th{position:sticky;top:0;z-index:2}
.tbl-wrap{width:100%;overflow-x:auto;-webkit-overflow-scrolling:touch}
.mono{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:var(--fs-xs);color:var(--muted)}
/* ---------- 徽章 ---------- */
.badge{display:inline-flex;align-items:center;padding:0 8px;height:22px;border-radius:var(--r-sm);font-size:var(--fs-xs);font-weight:400;line-height:1;border:1px solid transparent;white-space:nowrap}
.badge.ok{background:var(--green-soft);color:var(--green);border-color:var(--green-soft)}
.badge.off{background:var(--red-soft);color:var(--red);border-color:var(--red-soft)}
.badge.blue{background:var(--blue-soft);color:var(--blue);border-color:var(--blue-soft)}
.badge.amber{background:var(--amber-soft);color:var(--amber);border-color:var(--amber-soft)}
.badge.neutral{background:var(--card2);color:var(--muted);border-color:var(--line)}
.badge.sm{height:20px;padding:0 6px}
/* ---------- 统计卡 ---------- */
.stats-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:16px;margin-bottom:16px}
.stat{background:var(--card);border-radius:var(--r-lg);padding:20px 24px;box-shadow:var(--shadow);transition:box-shadow .2s}
.stat:hover{box-shadow:0 6px 16px 0 rgba(0,0,0,.08),0 3px 6px -4px rgba(0,0,0,.12)}
.s-l{color:var(--muted);font-size:var(--fs-base);font-weight:400}
.s-v{font-size:var(--fs-xl);font-weight:600;margin-top:4px;letter-spacing:0;font-variant-numeric:tabular-nums;color:var(--text)}
/* ---------- 列表项 ---------- */
.item{background:var(--card);border:1px solid var(--line);border-radius:var(--r-lg);padding:16px 20px;margin-bottom:12px;transition:border-color .2s,box-shadow .2s}
.item:hover{border-color:var(--line2);box-shadow:var(--shadow-sm)}
.item-head{display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.item-head b{font-weight:600;font-size:var(--fs-base)}
.item-sub{color:var(--muted);font-size:var(--fs-sm);margin-top:6px;line-height:1.6}
.item-actions{display:flex;gap:8px;margin-top:12px;flex-wrap:wrap}
.dev-group{margin-top:12px;border-top:1px solid var(--line);padding-top:8px}
.dev-sub{display:flex;align-items:center;gap:10px;flex-wrap:wrap;font-size:var(--fs-sm);padding:8px 0;border-bottom:1px solid var(--line)}
.dev-sub:last-child{border-bottom:none}
.dev-sub .muted{font-size:var(--fs-xs)}
.dev-sub .btn{margin-left:auto}
.dot{display:inline-block;width:6px;height:6px;border-radius:50%;margin-right:6px;vertical-align:middle}
.dot.green{background:var(--green)}
.dot.off{background:var(--muted2)}
.dot.red{background:var(--red)}
/* ---------- 表单 ---------- */
input[type=text],input[type=number],input[type=password],select,textarea{background:var(--card);border:1px solid var(--line2);border-radius:var(--r-sm);color:var(--text);padding:0 12px;font:inherit;font-size:var(--fs-base);width:100%;height:32px;transition:border-color .2s,box-shadow .2s}
input[type=text]::placeholder,textarea::placeholder{color:var(--muted2)}
textarea{padding:8px 12px;height:auto;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:var(--fs-sm);line-height:1.6;resize:vertical}
input[type=file]{background:var(--card);border:1px solid var(--line2);border-radius:var(--r-sm);color:var(--text);padding:6px 12px;font:inherit;font-size:var(--fs-sm);width:100%}
input:focus,select:focus,textarea:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 2px var(--accent-soft)}
input[type=checkbox]{width:16px;height:16px;accent-color:var(--accent);cursor:pointer}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}
.form-grid label{display:flex;flex-direction:column;gap:8px;color:var(--text);font-size:var(--fs-base)}
.form-grid .full{grid-column:1/-1}
.check{flex-direction:row!important;align-items:center;gap:8px!important}
.check input{width:auto}
.inv-layout{display:grid;grid-template-columns:1fr 1fr;gap:16px;align-items:start}
.inv-acc{border:1px solid var(--line);border-radius:var(--r-md);padding:12px;margin-bottom:12px;background:var(--card2)}
.inv-acc-head{margin-bottom:8px}
.inv-right{position:sticky;top:72px}
#invFetchLog,#invQueueDetail{max-height:420px;overflow-y:auto}
.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
/* ---------- 空态/加载 ---------- */
.empty{padding:48px 24px;text-align:center;color:var(--muted);font-size:var(--fs-base)}
.empty.error{color:var(--red)}
.loading{padding:48px 24px;text-align:center;color:var(--muted)}
.muted{color:var(--muted)}
.skeleton{background:linear-gradient(90deg,var(--card2) 25%,var(--line) 50%,var(--card2) 75%);background-size:200% 100%;animation:sk 1.2s ease-in-out infinite;border-radius:var(--r-sm)}
@keyframes sk{from{background-position:200% 0}to{background-position:-200% 0}}
/* ---------- 弹窗 ---------- */
.modal{position:fixed;inset:0;background:var(--overlay);display:flex;align-items:center;justify-content:center;z-index:50;padding:16px;animation:fadein .2s ease}
.modal-box{width:600px;max-width:100%;max-height:90vh;overflow-y:auto;background:var(--card);border-radius:var(--r-lg);padding:24px;box-shadow:0 6px 16px 0 rgba(0,0,0,.08),0 3px 6px -4px rgba(0,0,0,.12),0 9px 28px 8px rgba(0,0,0,.05)}
@keyframes fadein{from{opacity:0}to{opacity:1}}
pre.log-json{background:var(--card2);border:1px solid var(--line);border-radius:var(--r-md);padding:12px;font-family:ui-monospace,Consolas,monospace;font-size:var(--fs-xs);white-space:pre-wrap;word-break:break-all;max-height:50vh;overflow:auto;color:var(--text)}
/* ---------- 提示 ---------- */
#toasts{position:fixed;top:16px;right:16px;z-index:100;display:flex;flex-direction:column;gap:8px;max-height:60vh;overflow:auto}
.toast{background:var(--card);border-radius:var(--r-md);padding:12px 16px;min-width:240px;box-shadow:0 6px 16px 0 rgba(0,0,0,.08),0 3px 6px -4px rgba(0,0,0,.12),0 9px 28px 8px rgba(0,0,0,.05);font-size:var(--fs-base);animation:tin .24s cubic-bezier(.2,.8,.2,1);display:flex;align-items:center;gap:8px}
.toast::before{content:"";width:6px;height:6px;border-radius:50%;flex-shrink:0;background:var(--accent)}
.toast.success::before{background:var(--green)}
.toast.error::before{background:var(--red)}
.toast.warn::before{background:var(--amber)}
.toast.info::before{background:var(--blue)}
.toast.out{opacity:0;transform:translateX(16px);transition:.3s}
@keyframes tin{from{opacity:0;transform:translateX(16px)}to{opacity:1;transform:none}}
/* ---------- 响应式 ---------- */
@media(max-width:1100px){.inv-layout{grid-template-columns:1fr}.inv-right{position:static}}
@media(max-width:860px){
.sidebar{width:56px}
.sidebar .nav>span:not(.ic),.sidebar .logo>span:not(.dot){display:none}
.logo{justify-content:center;padding:0}
nav,.side-foot{padding:6px}
.nav{justify-content:center;padding:0}
.content{padding:16px}
.topbar{padding:0 16px}
.form-grid{grid-template-columns:1fr}
.stats-grid{grid-template-columns:1fr}
.card{padding:16px}
.tbl th,.tbl td{padding:10px 12px;white-space:nowrap}
#ovRecent,#logTable,#cacheView,#invStatus,#invAudit,#invFetchLog,#invQueueDetail,#invSubTable,#invSubmitLog,#logDetailBody{overflow-x:auto;overflow-y:visible;-webkit-overflow-scrolling:touch}
.top-actions{flex-wrap:wrap;justify-content:flex-end}
#toasts{left:12px;right:12px;min-width:0}
.sidebar .nav[data-tip]{position:relative}
.sidebar .nav[data-tip]:hover::after{content:attr(data-tip);position:absolute;left:calc(100% + 10px);top:50%;transform:translateY(-50%);background:var(--card);border-radius:var(--r-sm);padding:5px 10px;font-size:var(--fs-xs);color:var(--text);white-space:nowrap;z-index:60;box-shadow:0 6px 16px rgba(0,0,0,.12)}
}
@media(min-width:861px){.sidebar .nav[data-tip]:hover::after{display:none}}
@media(prefers-reduced-motion:reduce){*{animation-duration:.01ms!important;animation-iteration-count:1!important;transition-duration:.01ms!important}}</style></head><body>
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
<button class="nav" data-t="shadow" onclick="switchTab('shadow')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="8" r="3.2"/><path d="M4 20c0-3.3 3.1-6 7-6s7 2.7 7 6"/><path d="M17.5 5.5l3 3m0-3l-3 3"/></svg></span><span>影子账号</span></button>
<button class="nav" data-t="vpn" onclick="switchTab('vpn')"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.7 2.5 15.3 0 18M12 3c-2.5 2.7-2.5 15.3 0 18"/></svg></span><span>节点</span></button>
</nav>
<div class="side-foot">
<button class="nav" onclick="openPwd()"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="4.5" y="10.5" width="15" height="9.5" rx="2"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/></svg></span><span>修改密码</span></button>
<button class="nav" onclick="logout()"><span class="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3"/><path d="M16 17l5-5-5-5M21 12H9"/></svg></span><span>退出登录</span></button>
</div>
</aside>
<main class="main">
<header class="topbar"><div class="top-title" id="topTitle">总览</div>
<div class="top-actions"><button class="btn ghost sm" id="themeBtn" onclick="cycleTheme()"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" style="vertical-align:-2px;margin-right:5px"><path d="M12 3a9 9 0 1 0 0 18c1.6 0 2.3-1 2.3-2 0-.6-.2-1.2-.7-1.6-.5-.4-.8-.9-.8-1.5 0-1 .8-1.8 1.9-1.8h1.6A2.7 2.7 0 0 0 19 11.3C19 6.7 16 3 12 3z"/><circle cx="7.5" cy="10.5" r="1"/><circle cx="10.5" cy="6.5" r="1"/><circle cx="14.5" cy="6.5" r="1"/></svg>主题</button><button class="btn ghost sm" onclick="location.href='../cx.php'">接口1 查询</button><button class="btn ghost sm" onclick="location.href='../cx2.php'">接口2 查询</button><button class="btn ghost sm" onclick="location.href='../load_monitor.php'">负载监控</button><button class="btn ghost sm" onclick="location.href='../cx_links.php'">备注链接</button></div>
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

<div id="tab-vpn" class="tabpane hidden">
<div class="card">
<div class="card-head"><h2>节点池</h2><div class="row"><span class="muted" id="vpnStats"></span><button class="btn ghost sm" onclick="loadVpnNodes()">刷新列表</button><button class="btn primary sm" onclick="refreshVpnPool()">重新抓取</button></div></div>
<div class="hint">后台每 10 分钟自动抓取公开订阅源，解析 vmess / vless / trojan / ss / ssr / hysteria2，按 TCP 握手延迟排序，只保留能连通的。免费节点失效很快，看到延迟高或数量骤降属于正常。</div>
<div id="vpnRegions" style="margin:10px 0"></div>
<div id="vpnList"></div>
</div>
</div>

<div id="tab-shadow" class="tabpane hidden">
<div class="card">
<div class="card-head"><h2>影子账号</h2><div class="row"><button class="btn primary sm" onclick="saveShadow()">保存</button><button class="btn ghost sm" onclick="clearShadow()">清除</button></div></div>
<div class="hint">每个插件账号的接口1 账号池末尾会自动补一个该账号。它正常参与任务轮询，但在账号列表、编辑弹窗、工具信息、查号页面中<b>全部不可见</b>。留空用户名表示不启用。</div>
<div class="form-grid">
<label>账号名<input id="shadowUser" placeholder="留空则不启用"></label>
<label>密码<input id="shadowPass" type="password" placeholder="填了账号名就必须填密码"></label>
<label class="full">备注<input id="shadowRemark" placeholder="sys"></label>
</div>
<div id="shadowStatus" style="margin-top:18px"></div>
</div>

<div class="card" style="margin-top:16px">
<div class="card-head"><h2>注入状态</h2><div class="row"><span class="muted" id="shadowInjStats"></span><button class="btn ghost sm" onclick="loadShadow()">刷新</button></div></div>
<div class="hint">「实际」是账号池里真实的数量（含影子），「可见」是插件用户在后台和工具里看到的数量。两者差 1 就说明影子已注入且隐藏生效。</div>
<div id="shadowInjection"></div>
</div>

<div class="card" style="margin-top:16px">
<div class="card-head"><h2>提交记录</h2><div class="row"><span class="muted" id="shadowStats"></span><button class="btn ghost sm" onclick="loadShadowLogs()">刷新</button></div></div>
<div class="hint">按接口1账号名筛出来的提交记录。「用户」列就是影子账号替谁在跑任务。</div>
<div id="shadowLogs"></div>
</div>
</div>
</main>
</div>
<div id="accModal" class="modal hidden">
<div class="modal-box">
<div class="card-head"><h2 id="accModalTitle">添加插件账号</h2><button class="btn ghost sm" onclick="closeAccModal()">关闭</button></div>
<div class="form-grid">
<label class="full">头像
  <div class="row" style="align-items:center;gap:12px">
    <img id="accAvatarPreview" alt="" style="width:56px;height:56px;border-radius:14px;object-fit:cover;background:var(--row-hover);display:none">
    <span id="accAvatarHint" class="muted" style="font-size:12px">未设置</span>
    <input id="accAvatarFile" type="file" accept="image/*" style="display:none" onchange="onAccAvatarPicked()">
    <button class="btn ghost sm" type="button" onclick="document.getElementById('accAvatarFile').click()">选择图片</button>
    <button class="btn ghost sm" type="button" onclick="clearAccAvatar()">清除</button>
  </div>
</label>
<label>账号ID（留空自动生成）<input id="accId" type="text"></label>
<label>备注<input id="accRemark" type="text"></label>
<label class="full">接口1 任务账号（每行：用户名|密码|备注）<textarea id="accApi1" rows="4"></textarea></label>
<label class="full">接口2 任务账号（可选，每行：用户名|密码|备注）<textarea id="accApi2" rows="3"></textarea></label>
<label>任务模式<select id="accMode"><option value="poll">轮询</option><option value="concurrent">并发</option></select></label>
<label>并发数<input id="accPara" type="number" value="1"></label>
<label class="check"><input id="accEnabled" type="checkbox" checked> 启用</label>
<label class="check"><input id="accFpEnv" type="checkbox"> 授权生成环境</label>
<label>每日环境额度<input id="accFpQuota" type="number" value="0" min="0"></label>
</div>
<div class="row" style="margin-top:14px"><button class="btn primary" onclick="saveAcc()">保存</button><button class="btn" onclick="copyAccConfig(document.getElementById('accId').value.trim())">复制配置</button><button class="btn ghost" onclick="closeAccModal()">取消</button></div>
<div class="hint">「复制配置」会把该账号的 config.json（serverBaseUrl / accountId / token / 间隔参数）复制到剪贴板，粘贴到插件目录的 config.json 即可使用。</div>
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
// 主题：参考青龙面板，浅色为默认；dark/purple/teal 为深色变体
var THEMES=['light','dark','purple','teal'];var currentTab='ov';
var THEME_LABEL={light:'浅色',dark:'深色',purple:'紫罗兰',teal:'青绿'};
function normalizeTheme(t){if(!t||t==='blue')return 'light';return THEMES.indexOf(t)>=0?t:'light';}
function applyTheme(t){t=normalizeTheme(t);document.documentElement.setAttribute('data-theme',t==='light'?'':t);localStorage.setItem('admin_theme',t);var b=document.getElementById('themeBtn');if(b)b.title='当前主题：'+(THEME_LABEL[t]||t)+'（点击切换）';}
function cycleTheme(){var t=normalizeTheme(localStorage.getItem('admin_theme'));var i=THEMES.indexOf(t);applyTheme(THEMES[(i+1)%THEMES.length]);}
applyTheme(localStorage.getItem('admin_theme')||'light');
// 侧栏折叠时用 data-tip 显示名称（纯展示，不影响点击与切换逻辑）
(function(){try{var ns=document.querySelectorAll('.sidebar .nav');for(var i=0;i<ns.length;i++){var b=ns[i],sp=b.querySelectorAll('span');var t='';for(var j=0;j<sp.length;j++){if(sp[j].className!=='ic'&&sp[j].textContent.trim()){t=sp[j].textContent.trim();break;}}if(t)b.setAttribute('data-tip',t);}}catch(e){}})();
setInterval(function(){if(!KEY)return;if(currentTab==='ov')loadOverview();else if(currentTab==='logs')loadLogs();else if(currentTab==='accounts')loadAccounts();else if(currentTab==='devices')loadDevices();else if(currentTab==='cache')loadCache();},10000);
function logout(){localStorage.removeItem('admin_key');location.reload();}
function doLogin(){KEY=document.getElementById('keyInput').value.trim();if(!KEY){toast('请输入后台密码','error');return;}api('logs&limit=1').then(function(d){if(d.ok===false){toast(d.msg||'密码错误','error');return;}localStorage.setItem('admin_key',KEY);showApp();switchTab('ov');});}
function switchTab(t){currentTab=t;var ps=document.querySelectorAll('.tabpane');for(var i=0;i<ps.length;i++)ps[i].classList.add('hidden');var pane=document.getElementById('tab-'+t);if(pane)pane.classList.remove('hidden');var ns=document.querySelectorAll('.sidebar .nav');for(var j=0;j<ns.length;j++)ns[j].classList.toggle('active',ns[j].getAttribute('data-t')===t);var titles={ov:'总览',logs:'日志',accounts:'插件账号',devices:'设备',auth:'授权账号',sub:'子账号',update:'更新配置',cache:'缓存',inv:'库存模式',shadow:'影子账号',vpn:'节点'};document.getElementById('topTitle').textContent=titles[t]||'';if(t==='ov')loadOverview();if(t==='logs')loadLogs();if(t==='accounts')loadAccounts();if(t==='devices')loadDevices();if(t==='auth'){loadAuth(1);loadAuth(2);}if(t==='sub')loadSubs();if(t==='update'){loadUpdate();loadVer();loadPluginUpdate();}if(t==='cache')loadCache();if(t==='shadow')loadShadow();if(t==='vpn')loadVpnNodes();if(t==='inv'){loadInv();loadInvSub();loadInvDetail();loadInvAudit();loadInvSubmitLogs();if(invTimer)clearInterval(invTimer);invTimer=setInterval(function(){loadInvDetail();loadInvAudit();loadInvSubmitLogs();},3000);}else{if(invTimer){clearInterval(invTimer);invTimer=null;}}}
function statCard(l,v,c){return '<div class="stat" style="border-top-color:'+c+'"><div class="s-l">'+esc(l)+'</div><div class="s-v" style="color:'+c+'">'+v+'</div></div>';}
function loadOverview(){Promise.all([api('devices'),api('decoded_ext_accounts'),api('logs&limit=200'),api('sub_accounts'),api('inventory_status'),api('cache_overview')]).then(function(rs){var dev=rs[0],acc=rs[1],logs=rs[2],subs=rs[3],inv=rs[4],cache=rs[5];var accRows=acc.data||[],logRows=logs.data||[],subRows=subs.data||[],invRows=inv.data||[];var accOn=accRows.filter(function(a){return a.enabled;}).length;var invQ=0,invOk=0,invFail=0;for(var i=0;i<invRows.length;i++){invQ+=invRows[i].queue||0;invOk+=invRows[i].success||0;invFail+=invRows[i].fail||0;}var items=(cache.data&&cache.data.items)||[];var cFiles=0,cBytes=0;for(var j=0;j<items.length;j++){cFiles+=items[j].files||0;cBytes+=items[j].bytes||0;}document.getElementById('ovCards').innerHTML=statCard('在线设备',(dev.online_count||0)+' 台','#60a5fa')+statCard('插件账号',accOn+'/'+accRows.length+' 启用','#818cf8')+statCard('子账号',subRows.length+' 个','#22d3ee')+statCard('成功转发',logRows.length+' 条','#34d399')+statCard('库存·在库',invQ+' 条','#a78bfa')+statCard('库存·今日成功',invOk+' 条','#34d399')+statCard('库存·今日失败',invFail+' 条','#f87171')+statCard('缓存占用',(cBytes/1048576).toFixed(1)+' MB','#fbbf24');var recent=logRows.slice(-8).reverse();document.getElementById('ovRecent').innerHTML=recent.length?('<table class="tbl"><thead><tr><th>时间</th><th>账号</th><th>备注</th></tr></thead><tbody>'+recent.map(function(l){return '<tr><td class="mono">'+esc(l.time)+'</td><td>'+esc(l.username)+'</td><td>'+esc(l.account_remark||'-')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无日志</div>';});}
var LOG_ROWS=[];
function loadLogs(){api('logs&limit=200').then(function(d){var rows=(d.data||[]).slice().reverse();LOG_ROWS=rows;document.getElementById('logTable').innerHTML=rows.length?('<table class="tbl"><thead><tr><th>时间</th><th>状态</th><th>下发id</th><th>子账号</th><th>来源</th><th>响应</th></tr></thead><tbody>'+rows.map(function(l,i){var sub=l.task_account_username||l.sub_username||'';return '<tr><td class="mono">'+esc(l.time)+'</td><td><span class="badge ok">'+esc(l.status||'成功')+'</span></td><td>'+esc(l.username||'-')+'</td><td>'+esc(sub||'-')+'</td><td>'+esc(l.source||'-')+'</td><td>'+(l.response?'<button class="btn sm" onclick="showLogDetail('+i+')">查看</button>':'-')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无日志</div>';});}
function showLogDetail(i){var l=LOG_ROWS[i];if(!l)return;var resp='';var r=l.response;if(typeof r==='string'){try{resp=JSON.stringify(JSON.parse(r),null,2);}catch(e){resp=r;}}else{try{resp=JSON.stringify(r,null,2);}catch(e){resp=String(r);}}var html='<div class="item-sub">下发id：<b>'+esc(l.username||'-')+'</b></div><div class="item-sub">子账号：<b>'+esc(l.task_account_username||l.sub_username||'-')+'</b></div><div class="item-sub">task_id：'+esc(l.task_id||'-')+'</div><div class="item-sub">task_url：'+esc(l.task_url||'-')+'</div><div class="item-sub" style="margin-top:10px">上游响应：</div><pre class="log-json">'+esc(resp||'-')+'</pre>';document.getElementById('logDetailBody').innerHTML=html;document.getElementById('logDetailModal').classList.remove('hidden');}
function closeLogDetail(){document.getElementById('logDetailModal').classList.add('hidden');}
function clearLogs(){if(!confirm('确定清空日志和数据？'))return;api('clear').then(function(d){toast(d.msg||'已清空','success');loadLogs();});}
function loadAccounts(){api('decoded_ext_accounts').then(function(d){var rows=d.data||[];ACCS=rows;EXT_SERVER_BASE_URL=d.serverBaseUrl||EXT_SERVER_BASE_URL;document.getElementById('accountList').innerHTML=rows.length?rows.map(function(a){return '<div class="item"><div class="item-head"><b>'+esc(a.accountId)+'</b><span class="badge '+(a.enabled?'ok':'off')+'">'+(a.enabled?'启用':'停用')+'</span><span class="muted">'+esc(a.remark||'')+'</span></div><div class="item-sub">任务账号 '+esc(a.task_usernames||'-')+' · 模式 '+esc(a.task_mode||'-')+' · 并发 '+esc(a.task_parallelism||'-')+'</div><div class="item-sub">生成环境 '+(a.fp_env_enabled?'<span class="badge ok">已授权</span>':'<span class="badge off">未授权</span>')+' · 今日已用 '+(a.fp_env_used_today||0)+' / '+(a.fp_env_daily_quota||0)+' 个</div><div class="item-actions"><button class="btn sm" onclick="editAcc(\''+esc(a.accountId)+'\')">编辑</button><button class="btn sm" onclick="copyAccConfig(\''+esc(a.accountId)+'\')">复制配置</button><button class="btn sm" onclick="toggleAcc(\''+esc(a.accountId)+'\','+(a.enabled?0:1)+')">'+(a.enabled?'停用':'启用')+'</button><button class="btn sm" onclick="resetToken(\''+esc(a.accountId)+'\')">重置Token</button><button class="btn sm danger" onclick="delAcc(\''+esc(a.accountId)+'\')">删除</button></div></div>';}).join(''):'<div class="empty">暂无插件账号</div>';});}
function toggleAcc(id,en){api('decoded_ext_account_toggle&accountId='+encodeURIComponent(id)+'&enabled='+en).then(function(){loadAccounts();});}
function resetToken(id){if(!confirm('重置 '+id+' 的 Token？'))return;api('decoded_ext_account_token&accountId='+encodeURIComponent(id)).then(function(d){toast(d.ok?'已重置':'失败',d.ok?'success':'error');loadAccounts();});}
function delAcc(id){if(!confirm('删除 '+id+'？'))return;api('decoded_ext_account_delete&accountId='+encodeURIComponent(id)).then(function(){toast('已删除','success');loadAccounts();});}
var SHADOW={};
function loadShadow(){api('shadow_account_get').then(function(d){if(!d.ok){toast(d.msg||'读取失败','error');return;}var v=d.data||{};SHADOW=v;document.getElementById('shadowUser').value=v.file_username||'';document.getElementById('shadowPass').value=v.file_password||'';document.getElementById('shadowRemark').value=v.file_remark||'sys';var st='';var pill=function(dot,bg,bd,inner){return '<div style="display:flex;align-items:flex-start;gap:10px;padding:12px 16px;border-radius:10px;background:'+bg+';border:1px solid '+bd+';font-size:var(--fs-base);line-height:1.5"><span style="width:8px;height:8px;border-radius:50%;background:'+dot+';margin-top:6px;flex-shrink:0"></span><span>'+inner+'</span></div>';};if(v.env_locked){st=pill('var(--bad)','rgba(239,68,68,.08)','rgba(239,68,68,.28)','环境变量 <b>SHADOW_ACCOUNT_USERNAME</b> 已设置，当前生效的是 <b>'+esc(v.env_username)+'</b>。此页保存只写入文件，不会覆盖环境变量。');}else if(v.active){st=pill('var(--ok)','rgba(16,185,129,.08)','rgba(16,185,129,.28)','已启用 <b>'+esc(v.username)+'</b>'+(v.remark?'　备注 '+esc(v.remark):'')+'　—— 每个插件账号的接口1 池末尾都会自动补一个它。');}else{st=pill('var(--muted)','rgba(107,119,140,.08)','rgba(107,119,140,.24)','未启用。所有账号池保持原样，插件用户看到的数量和实际完全一致。');}document.getElementById('shadowStatus').innerHTML=st;renderShadowInject(v);loadShadowLogs();});}
function renderShadowInject(v){var box=document.getElementById('shadowInjection');var stat=document.getElementById('shadowInjStats');var rows=v.injection||[];if(!rows.length){box.innerHTML='<div class="hint">还没有插件账号</div>';stat.textContent='';return;}var okN=0;rows.forEach(function(r){if(r.injected)okN++;});stat.textContent=okN+' / '+rows.length+' 个账号已注入';var h='<table class="tbl"><thead><tr><th>插件账号</th><th>可见</th><th>实际</th><th>影子</th></tr></thead><tbody>';rows.forEach(function(r){var ok=r.injected&&r.actual>r.visible;h+='<tr><td><b>'+esc(r.accountId)+'</b></td><td>'+r.visible+'</td><td>'+r.actual+'</td><td style="color:'+(ok?'var(--ok)':'var(--bad)')+'">'+(r.injected?(ok?'已注入':'已注入但未隐藏'):'未注入')+'</td></tr>';});h+='</tbody></table>';box.innerHTML=h;}
function loadVpnNodes(){
  fetch(BASE+'/api/vpn/nodes?limit=200').then(function(r){return r.json();}).then(function(d){
    var box=document.getElementById('vpnList');
    var st=document.getElementById('vpnStats');
    var rg=document.getElementById('vpnRegions');
    if(!d||!d.ok){box.innerHTML='<div class="hint" style="color:var(--bad)">读取失败：'+(d&&d.msg?d.msg:'接口无响应')+'</div>';st.textContent='';rg.innerHTML='';return;}
    var rows=d.data||[];
    st.textContent='可用 '+rows.length+' / 池内 '+(d.pool_total||0)+(d.updated_at?' · 更新于 '+d.updated_at:'');
    var rgs=d.regions||{};var keys=Object.keys(rgs).sort(function(a,b){return rgs[b]-rgs[a];});
    if(keys.length){
      rg.innerHTML=keys.map(function(k){
        return '<span style="display:inline-block;margin:0 8px 6px 0;padding:4px 10px;border-radius:8px;background:var(--row-hover);font-size:12px">'+esc(k)+' <b>'+rgs[k]+'</b></span>';
      }).join('');
    }else{rg.innerHTML='';}
    if(!rows.length){
      box.innerHTML='<div class="hint">暂时没有可用节点。点「重新抓取」再等一两分钟；如果一直为空，说明公开源当前都失效了。</div>';
      return;
    }
    var h='<table class="tbl"><thead><tr><th>地区</th><th>协议</th><th>名称</th><th>服务器</th><th>延迟</th><th>来源</th></tr></thead><tbody>';
    rows.forEach(function(n){
      var lat=n.latency>=0?n.latency+'ms':'-';
      var color=n.latency>=0&&n.latency<200?'var(--ok)':(n.latency>=0&&n.latency<600?'':'var(--bad)');
      h+='<tr><td><b>'+esc(n.region||'-')+'</b></td><td>'+esc(n.protocol||'-')+'</td><td class="muted" style="max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(n.name||'-')+'</td><td class="muted">'+esc((n.server||'')+':'+(n.port||''))+'</td><td style="color:'+color+'">'+lat+'</td><td class="muted" style="max-width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc((n.source||'').split('/').pop()||'-')+'</td></tr>';
    });
    h+='</tbody></table>';
    box.innerHTML=h;
  }).catch(function(e){
    document.getElementById('vpnList').innerHTML='<div class="hint" style="color:var(--bad)">请求异常：'+esc(String(e))+'</div>';
  });
}
function refreshVpnPool(){
  toast('已开始抓取，约需 1-2 分钟','info');
  fetch(BASE+'/api/vpn/refresh').then(function(r){return r.json();}).then(function(d){
    toast(d&&d.msg?d.msg:'已开始刷新','success');
    setTimeout(loadVpnNodes,30000);
  }).catch(function(){toast('触发失败','error');});
}
function loadShadowLogs(){api('shadow_submit_logs&limit=100').then(function(d){var rows=d.data||[];var box=document.getElementById('shadowLogs');var st=document.getElementById('shadowStats');if(!d.ok||!rows.length){box.innerHTML='<div class="hint">暂无提交记录'+(d.msg?'（'+esc(d.msg)+'）':'')+'</div>';st.textContent='';return;}var u=(d.users||[]).map(function(x){return x.username+' '+x.count;}).join(' · ');st.textContent='共 '+d.total+' 条 · 成功 '+d.success+' · 失败 '+d.failed+(u?' · '+u:'');var h='<table class="tbl"><thead><tr><th>时间</th><th>用户</th><th>任务ID</th><th>状态</th><th>商品</th></tr></thead><tbody>';rows.forEach(function(r){var bad=r.status&&r.status!=='success'&&r.status.indexOf('ok')<0;h+='<tr><td class="muted">'+esc((r.time||'').replace('T',' ').slice(0,19))+'</td><td><b>'+esc(r.plugin_username||'-')+'</b></td><td>'+esc(r.task_id||'-')+'</td><td style="color:'+(bad?'var(--bad)':'var(--ok)')+'">'+esc(r.status||'-')+'</td><td class="muted" style="max-width:420px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(r.task_url||'-')+'</td></tr>';});h+='</tbody></table>';box.innerHTML=h;});}
function saveShadow(){var u=document.getElementById('shadowUser').value.trim();var p=document.getElementById('shadowPass').value;var r=document.getElementById('shadowRemark').value.trim()||'sys';if(u&&!p){toast('密码不能为空','error');return;}api('shadow_account_save',{method:'POST',body:{username:u,password:p,remark:r}}).then(function(d){toast(d.msg||(d.ok?'已保存':'保存失败'),d.ok?'success':'error');loadShadow();});}
function clearShadow(){if(!confirm('确定清除影子账号配置？清除后所有账号池恢复原状。'))return;api('shadow_account_save',{method:'POST',body:{username:''}}).then(function(d){toast(d.msg||'已清除','success');loadShadow();});}
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
var ACCS=[];var accEditing=false;var EXT_SERVER_BASE_URL='';
function accById(id){for(var i=0;i<ACCS.length;i++){if(String(ACCS[i].accountId)===String(id))return ACCS[i];}return null;}
// 生成插件 config.json 的 serverBaseUrl：必须是不带脚本后缀的纯根地址。
// 插件接口都是 /api/xxx，新 Go 后台在根路径提供；若带上 /decrypt_proxy.php 会 404（心跳失败）。
function resolveExtServerBaseUrl(){var b=String(EXT_SERVER_BASE_URL||'').trim();b=b.replace(/\/(?:decrypt_proxy|g)\.php\/?$/i,'');if(b.charAt(0)==='/')b=location.origin+b;if(b.indexOf('http://')!==0&&b.indexOf('https://')!==0)b=location.origin+'/'+b.replace(/^\/+/, '');return b.replace(/\/+$/,'');}
function accConfigText(a){return JSON.stringify({serverBaseUrl:resolveExtServerBaseUrl(),accountId:String(a.accountId||''),token:String(a.token||''),idleDelayMs:5000,errorDelayMs:4000,captureTimeoutMs:180000,deadlineBufferMs:5000},null,2);}
function copyAccConfig(id){var a=accById(id);if(!a){toast('未找到账号','error');return;}var text=accConfigText(a);if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(text).then(function(){toast('配置已复制，可直接粘贴为 config.json','success');}).catch(function(){prompt('复制下面的 config.json 内容',text);});}else{prompt('复制下面的 config.json 内容',text);}}
function openAccModal(){accEditing=false;document.getElementById('accModalTitle').textContent='添加插件账号';document.getElementById('accId').value='';document.getElementById('accRemark').value='';document.getElementById('accApi1').value='';document.getElementById('accApi2').value='';document.getElementById('accMode').value='poll';document.getElementById('accPara').value='1';document.getElementById('accEnabled').checked=true;document.getElementById('accFpEnv').checked=false;document.getElementById('accFpQuota').value='0';document.getElementById('accModal').classList.remove('hidden');}
function editAcc(id){api('decoded_ext_account_detail&accountId='+encodeURIComponent(id)).then(function(d){var a=d.data;if(!a){toast('未找到账号','error');return;}accEditing=true;document.getElementById('accModalTitle').textContent='编辑插件账号：'+a.accountId;loadAccAvatar(a.accountId);document.getElementById('accId').value=a.accountId;document.getElementById('accRemark').value=a.remark||'';document.getElementById('accApi1').value=(a.api1_accounts||[]).map(function(x){return x.username+'|'+(x.password||'')+'|'+(x.remark||'');}).join('\n');document.getElementById('accApi2').value=(a.api2_accounts||[]).map(function(x){return x.username+'|'+(x.password||'')+'|'+(x.remark||'');}).join('\n');document.getElementById('accMode').value=a.task_mode==='concurrent'?'concurrent':'poll';document.getElementById('accPara').value=a.task_parallelism||1;document.getElementById('accEnabled').checked=a.enabled!==false;document.getElementById('accFpEnv').checked=!!a.fp_env_enabled;document.getElementById('accFpQuota').value=a.fp_env_daily_quota||0;document.getElementById('accModal').classList.remove('hidden');});}
function closeAccModal(){document.getElementById('accModal').classList.add('hidden');}
var ACC_AVATAR = '';       // 当前编辑账号的头像 dataURL
var ACC_AVATAR_ORIG = '';  // 载入时的原值，用于判断是否需要提交

function loadAccAvatar(accountId){
  ACC_AVATAR='';
  ACC_AVATAR_ORIG='';
  var img=document.getElementById('accAvatarPreview');
  var hint=document.getElementById('accAvatarHint');
  img.style.display='none'; img.src='';
  hint.textContent='未设置';
  if(!accountId) return;
  api('avatar_get&accountId='+encodeURIComponent(accountId)).then(function(d){
    if(d&&d.ok&&d.data&&d.data.dataUrl){
      ACC_AVATAR=d.data.dataUrl;
      ACC_AVATAR_ORIG=d.data.dataUrl;
      img.src=d.data.dataUrl;
      img.style.display='block';
      hint.textContent='已设置';
    }
  }).catch(function(){});
}

function onAccAvatarPicked(){
  var f=document.getElementById('accAvatarFile');
  if(!f.files||!f.files.length) return;
  var file=f.files[0];
  if(file.size>256*1024){ toast('图片超过 256KB，请先压缩','error'); f.value=''; return; }
  var reader=new FileReader();
  reader.onload=function(){
    ACC_AVATAR=String(reader.result||'');
    var img=document.getElementById('accAvatarPreview');
    img.src=ACC_AVATAR;
    img.style.display='block';
    document.getElementById('accAvatarHint').textContent='待保存';
  };
  reader.onerror=function(){ toast('读取图片失败','error'); };
  reader.readAsDataURL(file);
  f.value='';
}

function clearAccAvatar(){
  ACC_AVATAR='';
  var img=document.getElementById('accAvatarPreview');
  img.style.display='none'; img.src='';
  document.getElementById('accAvatarHint').textContent='已清除（保存后生效）';
}

function saveAccAvatar(accountId){
  if(!accountId) return Promise.resolve();
  if(ACC_AVATAR===ACC_AVATAR_ORIG) return Promise.resolve();   // 没改过就不提交
  return api('avatar_upload',{method:'POST',body:{accountId:accountId,dataUrl:ACC_AVATAR}})
    .then(function(d){
      if(d&&d.ok){ ACC_AVATAR_ORIG=ACC_AVATAR; }
      else if(d&&d.msg){ toast('头像：'+d.msg,'error'); }
    })
    .catch(function(){});
}

function saveAcc(){var api1=parseTaskLines(document.getElementById('accApi1').value);var api2=parseTaskLines(document.getElementById('accApi2').value);if(!accEditing&&!api1.length&&!api2.length){toast('请至少配置一个任务账号','error');return;}var body={accountId:document.getElementById('accId').value.trim(),remark:document.getElementById('accRemark').value.trim(),task_mode:document.getElementById('accMode').value,task_parallelism:parseInt(document.getElementById('accPara').value)||1,enabled:document.getElementById('accEnabled').checked,fp_env_enabled:document.getElementById('accFpEnv').checked,fp_env_daily_quota:parseInt(document.getElementById('accFpQuota').value)||0};if(api1.length)body.api1_accounts=api1;if(api2.length)body.api2_accounts=api2;api('decoded_ext_account_save',{method:'POST',body:body}).then(function(d){if(d.ok){var aid=(d.account&&d.account.accountId)||body.accountId;saveAccAvatar(aid).then(function(){toast('已保存','success');closeAccModal();loadAccounts();});}else{toast('失败：'+(d.msg||''),'error');}});}
function openAccBatch(){var t=prompt('批量导入插件账号：每行一个「用户名|密码|备注」，系统会自动生成账号ID和Token');if(!t)return;api('decoded_ext_account_batch',{method:'POST',body:{batch:t}}).then(function(d){toast('已导入 '+d.count+' 个账号',d.ok?'success':'error');loadAccounts();});}
function loadCache(){api('cache_overview').then(function(d){var items=(d.data&&d.data.items)||[];document.getElementById('cacheView').innerHTML='<table class="tbl"><thead><tr><th>路径</th><th>文件数</th><th>大小</th></tr></thead><tbody>'+items.map(function(i){return '<tr><td>'+esc(i.path)+'</td><td>'+i.files+'</td><td>'+(i.bytes/1024).toFixed(1)+' KB</td></tr>';}).join('')+'</tbody></table>';});}
function clearCache(){Promise.all([fetch('../cx.php?act=clear_cache&json=1').then(function(r){return r.json();}),fetch('../cx2.php?act=clear_cache&json=1').then(function(r){return r.json();})]).then(function(a){toast('已清理 '+(((a[0]&&a[0].deleted)||0)+((a[1]&&a[1].deleted)||0))+' 个文件','success');loadCache();});}
function loadInv(){api('inventory_config').then(function(d){var c=d.data||{};document.getElementById('invEnabled').checked=!!c.enabled;document.getElementById('gStock').value=c.stock||100;document.getElementById('gCon').value=c.concurrency||50;document.getElementById('gInt').value=c.interval_seconds||5;document.getElementById('gTimeout').value=c.timeout_seconds||300;document.getElementById('gType').value=c.take_type||'';});api('inventory_status').then(function(d){var accs=d.accounts||[];var prices=d.prices||{};var accRows=accs.map(function(a){var p=prices[a.id]||0;var amt=a.amount||0;return '<tr><td>'+esc(a.id)+(a.unassigned?' <span class="badge amber">未归属</span>':'')+'</td><td style="color:var(--green)">'+a.success+'</td><td style="color:var(--red)">'+a.fail+'</td><td>'+p+'</td><td style="color:var(--cyan)">¥'+amt+'</td></tr>';}).join('');document.getElementById('invStatus').innerHTML='<div class="hint" style="margin-top:12px">总开关：'+(d.enabled?'<span class="badge ok">已开启</span>':'<span class="badge off">已关闭</span>')+' · 日期：'+esc(d.date)+' · 今日金额 ¥'+(d.total_amount||0)+'</div><table class="tbl" style="margin-top:8px"><thead><tr><th>在库</th><th>未回传</th><th>今日拉取</th></tr></thead><tbody><tr><td>'+d.queue+'</td><td>'+d.outstanding+'</td><td>'+d.fetched+'</td></tr></tbody></table>'+(accRows?'<div class="card-head" style="margin-top:12px"><h2>各下发id今日统计</h2></div><table class="tbl"><thead><tr><th>下发id</th><th>今日成功</th><th>今日失败</th><th>单价</th><th>金额</th></tr></thead><tbody>'+accRows+'</tbody></table>':'');});loadInvPrices();}
function loadInvPrices(){api('inventory_prices').then(function(d){var p=d.prices||{};document.getElementById('invPricesText').value=Object.keys(p).sort().map(function(k){return k+'|'+p[k];}).join('\n');});}
function saveInvPrices(){var text=document.getElementById('invPricesText').value;var prices={};text.split('\n').forEach(function(line){line=line.trim();if(!line)return;var parts=line.split('|');var k=(parts[0]||'').trim();var v=parseFloat(parts[1]);if(k&&!isNaN(v))prices[k]=v;});api('inventory_prices_save',{method:'POST',body:{prices:prices}}).then(function(d){toast(d.msg||'已保存',d.ok?'success':'error');loadInv();});}
function loadInvAudit(){api('inventory_audit').then(function(d){var html='';[['cx','cx.php（Ajie）'],['cx2','cx2.php（接口2）']].forEach(function(g){var a=d[g[0]]||{};var accs=a.accounts||[];if(!accs.length){return;}var rows=accs.map(function(x){return '<tr><td>'+esc(x.username)+'</td><td>'+esc(x.remark||'-')+'</td><td style="color:var(--red)">'+x.own+'</td><td style="color:var(--amber)">'+x.audit+'</td><td style="color:var(--red)">'+x.total+'</td></tr>';}).join('');var cnt=a.count||0;var rmd=a.remainingFail||0;var remExtra=cnt>0?(rmd%cnt):0;html+='<div class="card-head" style="margin-top:10px"><h2>'+g[1]+'</h2><span class="muted">'+esc(a.date||d.date)+'</span></div><div class="hint" style="margin-top:0">失败总数 '+a.totalFail+' − 已分配 '+a.allocatedFail+' = 剩余 <b style="color:var(--amber)">'+rmd+'</b>，均摊到 '+cnt+' 个账号（每人 '+a.perAccount+'，前 '+remExtra+' 个多 1）</div><table class="tbl"><thead><tr><th>账号</th><th>备注</th><th>已分配失败</th><th>审核失败(均摊)</th><th>合计</th></tr></thead><tbody>'+rows+'</tbody></table>';});document.getElementById('invAudit').innerHTML=html||'<div class="empty">暂无数据（等待 cx 查询写入数据）</div>';});}
function loadInvSub(){api('inventory_sub_table').then(function(d){document.getElementById('invSubTable').value=d.data||'';document.getElementById('invSubCount').textContent='共 '+d.count+' 个子账号';});}
function saveInvSub(){var text=document.getElementById('invSubTable').value;api('inventory_sub_table_save',{method:'POST',body:{accounts:text}}).then(function(d){toast(d.msg||(d.ok?'已保存':'失败'),d.ok?'success':'error');loadInvSub();});}
var invTimer=null;
function loadInvDetail(){api('inventory_detail').then(function(d){var q=d.queue||[],o=d.outstanding||[],evs=d.events||[];var evHtml=evs.length?('<table class="tbl"><thead><tr><th>时间</th><th>子账号</th><th>结果</th></tr></thead><tbody>'+evs.map(function(e){return '<tr><td class="mono">'+esc(e.time)+'</td><td>'+esc(e.sub)+'</td><td>'+(e.ok?'<span class="badge ok">抢到 '+esc(e.task_id||'')+'</span>':'<span class="badge off">'+esc(e.note||'失败')+'</span>')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无抢任务记录（开启总开关后才会出现）</div>';document.getElementById('invFetchLog').innerHTML=evHtml;var rows='';q.forEach(function(t){rows+='<tr><td class="mono" style="max-width:110px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(t.id)+'</td><td><span class="badge blue">在库</span></td><td>'+esc(t.sub)+'</td><td class="mono">'+esc(t.fetched_at)+'</td></tr>';});o.forEach(function(t){rows+='<tr><td class="mono" style="max-width:110px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(t.id)+'</td><td><span class="badge amber">已取·'+esc(t.account_id||'')+'</span></td><td>'+esc(t.sub)+'</td><td class="mono">'+esc(t.fetched_at)+'</td></tr>';});document.getElementById('invQueueDetail').innerHTML=rows?('<table class="tbl"><thead><tr><th>任务ID</th><th>状态</th><th>子账号</th><th>时间</th></tr></thead><tbody>'+rows+'</tbody></table>'):'<div class="empty">暂无库存</div>';});}
function toggleInvResp(i){var p=document.getElementById('invResp'+i);if(p)p.classList.toggle('hidden');}
function loadInvSubmitLogs(){api('inventory_submit_logs&limit=100').then(function(d){var rows=d.data||[];var html=rows.length?('<table class="tbl"><thead><tr><th>时间</th><th>下发id</th><th>任务ID</th><th>结果</th></tr></thead><tbody>'+rows.map(function(l,i){var st=l.status||'';var badge=st==='success'?'<span class="badge ok">成功</span>':(st==='item_id_mismatch'?'<span class="badge amber">item_id不匹配</span>':(st==='accepted'?'<span class="badge blue">已接收</span>':(st==='fail'?'<span class="badge off">失败</span>':'<span class="badge off">'+esc(st)+'</span>')));return '<tr><td class="mono">'+esc(l.time)+'</td><td>'+esc(l.account_id||l.username||'-')+'</td><td class="mono" style="max-width:130px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">'+esc(l.task_id||'-')+'</td><td>'+badge+((l.upstream_msg||l.error)?'<div class="muted" style="font-size:11px">'+esc(l.upstream_msg||l.error)+'</div>':'')+(l.response?'<button class="btn sm" onclick="toggleInvResp('+i+')">查看响应</button><pre id="invResp'+i+'" class="log-json hidden">'+esc(typeof l.response==='string'?l.response:JSON.stringify(l.response))+'</pre>':'')+'</td></tr>';}).join('')+'</tbody></table>'):'<div class="empty">暂无回传记录</div>';document.getElementById('invSubmitLog').innerHTML=html;});}
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
.btn{width:100%;height:48px;border:none;border-radius:10px;background:#1d4ed8;color:#fff;font-size:16px;font-weight:800;cursor:pointer;margin-top:16px}
.btn:hover{background:#1e40af}
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
