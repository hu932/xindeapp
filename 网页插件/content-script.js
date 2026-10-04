typeof Object.entries === "undefined" && (Object.entries = function (param1) {
  var local1 = Object.keys(param1),
    local2 = local1.length,
    local3 = new Array(local2);
  while (local2--) {
    local3[local2] = [local1[local2], param1[local1[local2]]];
  }
  return local3;
});
typeof Object.values === "undefined" && (Object.values = function (param2) {
  var local4 = Object.keys(param2),
    local5 = [];
  for (var local6 = 0; local6 < local4.length; local6++) {
    local5.push(param2[local4[local6]]);
  }
  return local5;
});
typeof Array.from === "undefined" && (Array.from = function (param3) {
  var local7 = [],
    local8 = param3.length;
  for (var local9 = 0; local9 < local8; local9++) {
    local7.push(param3[local9]);
  }
  return local7;
});
const 服务器 = typeof 任务服务器 !== "undefined" && 任务服务器 ? String(任务服务器).replace(/\/+$/, "") : "https://zb1.eqwofaygdsjko.uk";
let IP = '',
  submitTaskUrl = '',
  用户Token = null;
function 获取任务服务器地址() {
  return String(服务器 || "https://zb1.eqwofaygdsjko.uk").replace(/\/+$/, "");
}
function 获取下发Id() {
  return typeof 下发Id !== "undefined" ? String(下发Id || '').trim() : '';
}
function 获取代理地址() {
  return typeof 代理地址 !== "undefined" ? String(代理地址 || '').replace(/\/+$/, "") : "http://103.146.231.72:2036";
}
function 代理接口(api) {
  return 获取代理地址() + "/decrypt_proxy.php?ajie_api=" + api;
}
function 提取任务商品标识(param6 = {}) {
  let local16 = param6.shop_id || param6.shopId || '',
    local17 = param6.item_id || param6.itemId || '';
  if ((!local16 || !local17) && param6.taskUrl) {
    const local18 = String(param6.taskUrl).match(/product\/(\d+)\/(\d+)/);
    local18 && (local16 = local18[1], local17 = local18[2]);
  }
  return {
    shopId: String(local16 || ''),
    itemId: String(local17 || '')
  };
}
function 提取任务ID(param7 = {}) {
  return String(param7.task_id || param7.taskId || param7.trace_id || param7.traceId || param7.id || '');
}
async function 代理Post(param11, param12 = {}) {
  const local23 = 获取任务服务器地址();
  const localHeaders = {
    "Content-Type": "application/json",
    Accept: "*/*"
  };
  if (param11 === "l") {
    用户Token = null;
    return await XxHttp.Post(代理接口("login"), {
      accountId: param12.id || 获取下发Id()
    }, localHeaders);
  }
  if (param11 === "t") {
    return await XxHttp.Post(代理接口("take"), {
      accountId: param12.id || 获取下发Id(),
      token: param12.k || 用户Token || ""
    }, localHeaders);
  }
  if (param11 === "s") {
    return await XxHttp.Post(local23 + "/api/task/submit/v2", {
      appVersion: param12.av || "vv2",
      url: param12.q,
      result: param12.r
    }, Object.assign({}, localHeaders, {
      Authorization: "Bearer " + (param12.k || 用户Token || "")
    }));
  }
  return {
    success: false,
    data: null,
    error: "未知的接口动作"
  };
}
function 提取接口消息(param13, param14 = "请求失败") {
  if (param13 && param13.error) return String(param13.error);
  const local24 = param13 && param13.data !== undefined ? param13.data : param13;
  if (local24 === null || local24 === undefined) return param14;
  if (typeof local24 !== "object") return String(local24);
  if (local24.msg) return String(local24.msg);
  if (local24.message) return String(local24.message);
  if (local24.error) return String(local24.error);
  try {
    return JSON.stringify(local24);
  } catch (local25) {
    return param14;
  }
}
class 计数器类 {
  constructor(param14, param15 = 0) {
    this.name = param14, this.defaultValue = param15;
  }
  ["add"]() {
    let local29 = this.get();
    return localStorage.setItem(this.name, local29 + 1), local29 + 1;
  }
  ["get"]() {
    const local30 = localStorage.getItem(this.name);
    return local30 ? parseInt(local30) : this.defaultValue;
  }
  ['重置']() {
    return localStorage.setItem(this.name, this.defaultValue), this.defaultValue;
  }
  ["set"](param16) {
    return localStorage.setItem(this.name, parseInt(param16)), param16;
  }
  ['减1']() {
    let local31 = this.get();
    if (local31 > 0) return localStorage.setItem(this.name, local31 - 1), local31 - 1;
    return 0;
  }
}
const 风控计数 = new 计数器类("风控计数"),
  连续风控计数 = new 计数器类('连续风控计数'),
  任务计数 = new 计数器类("任务计数"),
  台湾任务重试计数 = new 计数器类("台湾任务重试计数"),
  开始按钮图标 = "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M8 5v14l11-7z\"></path></svg>",
  停止按钮图标 = "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><rect x=\"6\" y=\"6\" width=\"12\" height=\"12\" rx=\"2\"></rect></svg>",
  任务状态管理器 = {
    正在运行: false,
    恢复状态() {
      const local32 = localStorage.getItem("储存_开始或停止");
      this.正在运行 = local32 === '开始';
      if (!隐藏元素 || typeof 隐藏元素.getElementById !== "function") return this.正在运行;
      const local33 = 隐藏元素.getElementById("toggleTaskBtn"),
        local34 = 隐藏元素.getElementById('toggleLabel'),
        local35 = 隐藏元素.getElementById("toggleIcon");
      if (this.正在运行) {
        local33 && (local33.classList.remove("stopped"), local33.classList.add("running"), local33.style.background = "#dc2626", local33.style.boxShadow = "0 6px 14px rgba(220, 38, 38, 0.22)");
        if (local34) local34.textContent = '停止';
        if (local35) local35.innerHTML = 停止按钮图标;
      } else {
        local33 && (local33.classList.remove('running'), local33.classList.add('stopped'), local33.style.background = '#16a34a', local33.style.boxShadow = '0\x206px\x2014px\x20rgba(22,\x20163,\x2074,\x200.24)');
        if (local34) local34.textContent = '开始';
        if (local35) local35.innerHTML = 开始按钮图标;
      }
      return this.正在运行;
    },
    开始() {
      this.正在运行 = true, localStorage.setItem('储存_开始或停止', '开始'), 记录任务活动();
      const local36 = 隐藏元素.getElementById("toggleTaskBtn"),
        local37 = 隐藏元素.getElementById("toggleLabel"),
        local38 = 隐藏元素.getElementById("toggleIcon");
      local36 && (local36.classList.remove("stopped"), local36.classList.add("running"), local36.style.background = "#dc2626", local36.style.boxShadow = "0 6px 14px rgba(220, 38, 38, 0.22)");
      if (local37) local37.textContent = '停止';
      if (local38) local38.innerHTML = 停止按钮图标;
    },
    停止(param16 = false) {
      this.正在运行 = false, localStorage.setItem("储存_开始或停止", '停止'), 清除任务看门狗();
      const local39 = 隐藏元素.getElementById('toggleTaskBtn'),
        local40 = 隐藏元素.getElementById('toggleLabel'),
        local41 = 隐藏元素.getElementById("toggleIcon");
      local39 && (local39.classList.remove("running"), local39.classList.add("stopped"), local39.style.background = '#16a34a', local39.style.boxShadow = "0 6px 14px rgba(22, 163, 74, 0.24)");
      if (local40) local40.textContent = '开始';
      if (local41) local41.innerHTML = 开始按钮图标;
    },
    是否运行中() {
      return this.正在运行;
    },
    切换() {
      return this.正在运行 ? this.停止() : this.开始(), this.正在运行;
    }
  },
  域名配置 = {
    台湾: {
      shopee: "shopee.tw",
      xiapibuy: "xiapi.xiapibuy.com",
      country_id: 'tw'
    }
  };
const 看门狗超时毫秒 = 25000,
  看门狗任务时间键 = '储存_看门狗_任务时间',
  看门狗重试计数键 = '储存_看门狗_重试次数';
let 任务看门狗定时器 = null,
  上传处理中 = false,
  任务监督定时器 = null,
  任务上次活动时间 = Date.now();
function 记录任务活动() {
  任务上次活动时间 = Date.now();
}
function 看门狗任务运行中() {
  return localStorage.getItem('储存_开始或停止') === '开始';
}
function 看门狗有待处理任务() {
  return !!localStorage.getItem('储存_获取任务的数据');
}
function 清除任务看门狗() {
  任务看门狗定时器 && (clearTimeout(任务看门狗定时器), 任务看门狗定时器 = null);
  localStorage.removeItem(看门狗任务时间键);
}
function 启动任务看门狗() {
  清除任务看门狗();
  if (!看门狗任务运行中() || !看门狗有待处理任务()) return;
  localStorage.setItem(看门狗任务时间键, String(Date.now()));
  任务看门狗定时器 = setTimeout(任务看门狗超时, 看门狗超时毫秒);
}
function 任务看门狗超时() {
  任务看门狗定时器 = null;
  localStorage.removeItem(看门狗任务时间键);
  if (!看门狗任务运行中() || !看门狗有待处理任务()) return;
  setTimeout(function () {
    if (!看门狗任务运行中() || !看门狗有待处理任务()) return;
    if (上传处理中) {
      启动任务看门狗();
      return;
    }
    const local = parseInt(localStorage.getItem(看门狗重试计数键) || '0', 10) || 0,
      次数 = local + 1;
    localStorage.setItem(看门狗重试计数键, String(次数));
    slog.mlog('等待商品数据超时' + 看门狗超时毫秒 / 1000 + '秒，重新获取任务(第' + 次数 + '次)');
    if (次数 > 3) {
      slog.mlog('连续多次等待商品数据超时，已停止任务');
      XxQianDuan.亮牌子('yellow');
      任务状态管理器.停止();
      Xx框架 && Xx框架.更新任务状态显示 && Xx框架.更新任务状态显示();
      return;
    }
    记录任务活动();
    申请任务();
    启动任务看门狗();
  }, 200);
}
function 任务看门狗收到数据() {
  记录任务活动();
  清除任务看门狗();
  localStorage.removeItem(看门狗重试计数键);
}
function 任务看门狗恢复() {
  if (!看门狗任务运行中() || !看门狗有待处理任务()) {
    localStorage.removeItem(看门狗任务时间键);
    return;
  }
  启动任务看门狗();
}
function 启动任务监督() {
  if (任务监督定时器) clearInterval(任务监督定时器);
  任务监督定时器 = setInterval(() => {
    if (!看门狗任务运行中()) return;
    if (上传处理中) return;
    if (申请任务锁) return;
    if (任务看门狗定时器) return;
    if (Date.now() - 任务上次活动时间 < 60000) return;
    slog.mlog('检测到任务长时间无进展，自动恢复任务循环');
    记录任务活动();
    申请任务();
  }, 10000);
}
function 停止任务监督() {
  if (任务监督定时器) clearInterval(任务监督定时器);
  任务监督定时器 = null;
}
async function 处理商品数据(param) {
  if (上传处理中) return;
  上传处理中 = true;
  try {
    await 上传商品(param);
  } catch (local) {
    slog.mlog('处理商品数据异常: ' + (local && local.message ? local.message : local));
  } finally {
    上传处理中 = false;
    if (!任务看门狗定时器 && 看门狗任务运行中() && 看门狗有待处理任务()) 启动任务看门狗();
  }
}
function 规范化间隔(param) {
  const local = Math.round((Number(param) || 0.1) * 10) / 10;
  return Math.min(3600, Math.max(0.1, local));
}
let 任务接口模式 = "old";
const 固定分标识ClientId = '台湾';
let 分标识ClientId = 固定分标识ClientId;
typeof localStorage !== "undefined" && (任务接口模式 = "old", localStorage.setItem("任务接口模式", "old"), localStorage.setItem('地区', '台湾'));
const 页面配置 = {
    任务间隔最小: typeof 默认任务间隔最小 !== 'undefined' ? 默认任务间隔最小 : 1,
    任务间隔最大: typeof 默认任务间隔最大 !== "undefined" ? 默认任务间隔最大 : 3,
    批次任务数: typeof 默认批次任务数 !== "undefined" ? 默认批次任务数 : 1000,
    版本: typeof 配置版本 !== "undefined" ? 配置版本 : "interval-1-3-1000"
  },
  任务管理器 = {
    当前间隔: null,
    任务队列: [],
    定时器: null,
    设置间隔(param17, param18) {
      页面配置.任务间隔最小 = 规范化间隔(param17 || 页面配置.任务间隔最小), 页面配置.任务间隔最大 = Math.max(规范化间隔(param17 || 页面配置.任务间隔最小), 规范化间隔(param18 || 页面配置.任务间隔最大)), localStorage.setItem("任务配置", JSON.stringify(页面配置));
    },
    获取随机间隔() {
      const local43 = 规范化间隔(页面配置.任务间隔最小),
        local44 = Math.max(local43, 规范化间隔(页面配置.任务间隔最大));
      return Math.round((Math.random() * (local44 - local43) + local43) * 10) / 10;
    },
    设置批次任务数(param19) {
      页面配置.批次任务数 = param19 || 页面配置.批次任务数, 多少次暂停 = 页面配置.批次任务数, localStorage.setItem('任务配置', JSON.stringify(页面配置));
    },
    加载配置() {
      const local45 = localStorage.getItem("任务配置");
      if (local45) try {
        const local46 = JSON.parse(local45);
        local46.版本 && local46.版本 === 页面配置.版本 ? Object.assign(页面配置, local46) : localStorage.setItem('任务配置', JSON.stringify(页面配置));
      } catch (local47) {
        localStorage.setItem("任务配置", JSON.stringify(页面配置));
      } else localStorage.setItem("任务配置", JSON.stringify(页面配置));
      return 多少次暂停 = 页面配置.批次任务数, 页面配置;
    },
    重置配置() {
      const local48 = {
        任务间隔最小: typeof 默认任务间隔最小 !== 'undefined' ? 默认任务间隔最小 : 1,
        任务间隔最大: typeof 默认任务间隔最大 !== "undefined" ? 默认任务间隔最大 : 3,
        批次任务数: typeof 默认批次任务数 !== "undefined" ? 默认批次任务数 : 1000,
        版本: 页面配置.版本
      };
      return Object.assign(页面配置, local48), 多少次暂停 = 页面配置.批次任务数, localStorage.setItem("任务配置", JSON.stringify(页面配置)), 页面配置;
    }
  };
async function 用户登录() {
  try {
    const local51 = await 代理Post('l');
    const localResp = local51 && local51.data;
    const localToken = localResp && localResp.data && localResp.data.token;
    if (localToken) {
      return 用户Token = localToken, localStorage.setItem("用户Token", 用户Token), true;
    }
    try {
      slog.mlog((localResp && (localResp.error || localResp.msg)) || (local51 && local51.error) || "登录失败");
    } catch (local53) {}
    return localStorage.setItem("用户Token", ''), false;
  } catch (local55) {
    try {
      slog.mlog(local55 && local55.message ? local55.message : "登录异常");
    } catch (local56) {}
    return false;
  }
}
async function 确保用户Token() {
  if (用户Token) return true;
  const local56 = localStorage.getItem("用户Token");
  if (local56 && local56 !== 'undefined') return 用户Token = local56, true;
  return await 用户登录();
}
class CookieManager {
  static ["set"](param20, param21, param22 = {}) {
    const {
      daysToLive: local57,
      path = '/',
      domain: local58,
      secure: local59,
      sameSite: local60
    } = param22;
    let local61 = encodeURIComponent(param20) + '=' + encodeURIComponent(param21);
    if (local57) {
      const local62 = new Date();
      local62.setTime(local62.getTime() + local57 * 24 * 60 * 60 * 1000), local61 += "; expires=" + local62.toUTCString();
    }
    local61 += "; path=" + path, local58 && (local61 += "; domain=" + local58), local59 && (local61 += "; secure"), local60 && (local61 += "; sameSite=" + local60), document.cookie = local61;
  }
  static ['get'](param23) {
    const local63 = encodeURIComponent(param23) + '=',
      local64 = document.cookie.split(';');
    for (let local65 = 0; local65 < local64.length; local65++) {
      let local66 = local64[local65].trim();
      if (local66.indexOf(local63) === 0) return decodeURIComponent(local66.substring(local63.length));
    }
    return null;
  }
  static ["delete"](param24, param25 = {}) {
    const {
      path = '/',
      domain: local67
    } = param25;
    let local68 = encodeURIComponent(param24) + "=; expires=Thu, 01 Jan 1970 00:00:00 GMT";
    local68 += "; path=" + path, local67 && (local68 += "; domain=" + local67), document.cookie = local68;
  }
  static ["has"](param26) {
    return this.get(param26) !== null;
  }
  static ["getAll"]() {
    const local69 = {},
      local70 = document.cookie.split(';');
    return local70.forEach(param27 => {
      const [local71, local72] = param27.trim().split('=');
      local71 && local72 && (local69[decodeURIComponent(local71)] = decodeURIComponent(local72));
    }), local69;
  }
}
var 隐藏元素,
  XxHttp = class {
    static ['Post'](param28, param29, param30 = {}) {
      return new Promise(param31 => {
        let localDone = false;
        const local超时定时器 = setTimeout(() => {
          if (!localDone) {
            localDone = true;
            param31({ success: false, data: null, error: "请求超时" });
          }
        }, 20000);
        用户Token && !param30.Authorization && (服务器 && param28.includes(服务器.replace(':443', '')) || IP && param28.includes(IP.replace(":443", ''))) && (param30.Authorization = "Bearer " + 用户Token), param28.includes("/api/v4/pdp/get_pc") && param30["Content-Type"] == "application/json" && param28.includes("tz_offset_in_minutes") && (submitTaskUrl = param28), !param30["Content-Type"] && (param30["Content-Type"] = "application/json"), !param30.Accept && (param30.Accept = "*/*"), param29 && typeof param29 !== "string" && (param29 = JSON.stringify(param29));
        try {
          chrome.runtime.sendMessage({
            act: "发送网络请求3_Post",
            url: param28,
            data: param29,
            headers: param30
          }, param32 => {
            if (localDone) return;
            localDone = true;
            clearTimeout(local超时定时器);
            const localRuntimeError = chrome.runtime.lastError;
            localRuntimeError ? param31({
              success: false,
              data: null,
              error: localRuntimeError.message
            }) : param31(param32);
          });
        } catch (localRuntimeError) {
          if (!localDone) {
            localDone = true;
            clearTimeout(local超时定时器);
            param31({
              success: false,
              data: null,
              error: localRuntimeError && localRuntimeError.message ? localRuntimeError.message : "扩展上下文已失效，请刷新页面"
            });
          }
        }
      });
    }
    static ['get'](param33, param34 = {}) {
      return new Promise(param35 => {
        let localDone = false;
        const local超时定时器 = setTimeout(() => {
          if (!localDone) {
            localDone = true;
            param35({ success: false, data: null, error: "请求超时" });
          }
        }, 20000);
        用户Token && !param34.Authorization && (服务器 && param33.includes(服务器.replace(":443", '')) || IP && param33.includes(IP.replace(":443", ''))) && (param34.Authorization = "Bearer " + 用户Token), param33.includes('/api/task/take') ? (param34.Accept = "application/json, text/plain, */*", param34['Content-Type'] = "application/json") : !param34.Accept && (param34.Accept = '*/*');
        try {
          chrome.runtime.sendMessage({
            act: "发送网络请求3_Get",
            url: param33,
            headers: param34
          }, param36 => {
            if (localDone) return;
            localDone = true;
            clearTimeout(local超时定时器);
            const localRuntimeError = chrome.runtime.lastError;
            localRuntimeError ? param35({
              success: false,
              data: null,
              error: localRuntimeError.message
            }) : param35(param36);
          });
        } catch (localRuntimeError) {
          if (!localDone) {
            localDone = true;
            clearTimeout(local超时定时器);
            param35({
              success: false,
              data: null,
              error: localRuntimeError && localRuntimeError.message ? localRuntimeError.message : "扩展上下文已失效，请刷新页面"
            });
          }
        }
      });
    }
  },
  版本号 = 2;
localStorage.setItem("版本号", 版本号), window.addEventListener("message", async function (param37) {
  if (param37.data && param37.data.type) {
    let local73 = param37.data.type;
    if (local73 === "send") {
      let local74 = param37.data.data;
      submitTaskUrl = '', 任务看门狗收到数据(), 处理商品数据(local74);
    }
  }
}), chrome.runtime.onMessage.addListener((param38, param39, param40) => {
  param38.type === "send" && (任务看门狗收到数据(), 处理商品数据(param38.data));
}), $(document).ready(function () {
  chrome.storage.local.get({
    G: {}
  }, function (param41) {
    if (typeof G !== "undefined" && G && Array.isArray(G.log) && G.log.length) {
      G.log = (param41.G && Array.isArray(param41.G.log) ? param41.G.log : []).concat(G.log);
    } else {
      G = param41.G;
    }
    G.log === undefined && (G.log = []);
    Array.isArray(G.log) && G.log.length > 日志最大条数 && (G.log = G.log.slice(-日志最大条数), chrome.storage.local.set({
      G: G
    }, function () {}));
    const local75 = window.location.host;
    local75 === "xiapi.xiapibuy.com" && (IP = '');
    const local76 = window.location.href;
    if (local76.includes('shopee') || local76.includes('xiapibuy.com')) {
      任务管理器.加载配置(), Xx框架.创建框架窗口(), 任务状态管理器.恢复状态();
    }
  });
});
let 申请任务锁 = false;
async function 申请任务() {
  if (申请任务锁) return;
  申请任务锁 = true;
  记录任务活动();
  try {
    let local77 = await Xx项目lib.申请任务2_前置检查();
    if (local77) {
      任务状态管理器.停止();
      return;
    }
    任务接口模式 = "old", localStorage.setItem("任务接口模式", "old"), localStorage.setItem('地区', '台湾');
    const local78 = await 确保用户Token();
    if (!local78) {
      slog.mlog("登录失败"), XxQianDuan.亮牌子("red"), 任务状态管理器.停止();
      return;
    }
    const local79 = await 代理Post('t', {
      k: 用户Token
    });
    const local82 = local79 && local79.data;
    const local任务 = local82 && (local82.task || (local82.tasks && local82.tasks[0]));
    if (local任务 && (local任务.taskUrl || local任务.url)) {
      台湾任务重试计数.重置(), slog.mlog("获取任务成功"), localStorage.setItem('储存_获取任务的数据', JSON.stringify(local任务)), slog.mlog("获取任务数据成功"), 启动任务看门狗(), Xx项目_跳转.处理跳转(local任务.taskUrl || local任务.url);
      return;
    }
    if (local82) {
      slog.mlog(local82.msg || local82.error || "获取任务失败");
    } else if (local79) {
      slog.mlog(提取接口消息(local79, "获取任务失败"));
    }
    slog.mlog("获取任务失败");
    const local80 = 任务管理器.获取随机间隔() * 1000,
      local81 = local80 / 1000;
    slog.mlog('等待' + local81 + "秒重试"), setTimeout(申请任务, local80);
  } finally {
    申请任务锁 = false;
  }
}
async function 上传商品(param42) {
  if (typeof param42 === "string") try {
    param42 = JSON.parse(param42);
  } catch (local85) {
    return;
  }
  if (param42.error && param42.error !== null && param42.error !== 266900504 && param42.error !== 266900002) {
    风控计数.add(), 连续风控计数.add();
    let local86 = 连续风控计数.get();
    if (local86 > 1) {
      slog.mlog("失败过多，已停止"), XxQianDuan.亮牌子('yellow'), 任务状态管理器.停止(), Xx框架.更新任务状态显示();
      return;
    } else {
      const local87 = 任务管理器.获取随机间隔() * 1000;
      setTimeout(申请任务, local87);
      return;
    }
    return;
  }
  let local83 = await Xx项目lib.申请任务2_前置检查();
  if (local83) {
    任务状态管理器.停止();
    return;
  }
  XxQianDuan.removeItem_info("储存_几轮不活数");
  记录任务活动();
  let local84 = localStorage.getItem('储存_获取任务的数据');
  if (!local84) {
    if (看门狗任务运行中()) {
      slog.mlog('任务数据为空，重新获取任务');
      const local84Retry = 任务管理器.获取随机间隔() * 1000;
      setTimeout(申请任务, local84Retry);
    }
    return;
  }
  if (local84) {
    local84 = JSON.parse(local84);
    if (submitTaskUrl == '') {
      let local88, local89;
      if (local84.taskUrl) {
        const local90 = local84.taskUrl.match(/product\/(\d+)\/(\d+)/);
        local90 && (local88 = local90[1], local89 = local90[2]);
      }
      (!local88 || !local89) && (local88 = local84.shop_id, local89 = local84.item_id);
      if (local88 && local89) {
        let local91 = window.location.hostname,
          local92 = "shopee.tw";
        for (const [local93, local94] of Object.entries(域名配置)) {
          if (local91.includes(local94.shopee) || local91.includes(local94.xiapibuy)) {
            local92 = local94.shopee;
            break;
          }
        }
        submitTaskUrl = "https://" + local92 + "/api/v4/pdp/get_pc?display_model_id=0&item_id=" + local89 + "&model_selection_logic=3&shop_id=" + local88 + "&tz_offset_in_minutes=480&detail_level=0";
      }
    }
    try {
      const local95 = await 确保用户Token();
      if (!local95) {
        用户Token = null, slog.mlog("登录失败"), XxQianDuan.亮牌子("red");
        return;
      }
      const local96 = 提取任务商品标识(local84),
        local97 = {
          k: local84.api1_take_token || local84.api1TakeToken || 用户Token,
          av: "vv2",
          q: submitTaskUrl,
          r: JSON.stringify(param42),
          sid: Date.now().toString(36) + '_' + Math.random().toString(36).slice(2, 10),
          tid: 提取任务ID(local84),
          iid: local96.itemId,
          sh: local96.shopId,
          pu: local84.taskUrl || '',
          pd: submitTaskUrl
        };
      let local98 = await 代理Post('s', local97);
      if (!local98.success) {
        slog.mlog("网络失败");
        let local101 = localStorage.getItem("储存_获取任务的数据");
        local101 && 上传商品(param42);
        return;
      }
      let local99 = local98.data,
        local100 = false;
      typeof local99 === "object" && local99.data && local99.code === "200" && (local100 = true);
      if (local100) {
        slog.mlog('上传成功'), XxQianDuan.removeItem_info("储存_获取任务的数据"), 任务计数.add();
        let local102 = parseInt(localStorage.getItem("开始任务计数") || '0'),
          local103 = 任务计数.get(),
          local104 = local103 - local102;
        if (local104 > 0 && local104 % 多少次暂停 === 0) {
          slog.mlog("批次完成"), XxQianDuan.亮牌子('purple'), 任务状态管理器.停止(), Xx框架.更新任务状态显示();
          return;
        }
        const local105 = 任务管理器.获取随机间隔() * 1000,
          local106 = local105 / 1000;
        slog.mlog('等待' + local106 + "秒开始下一条任务"), setTimeout(申请任务, local105);
        return;
      }
      if (任务接口模式 === "old") {
        let local107 = JSON.stringify(local98);
        if (local107.includes("或者已经完成过了")) {
          slog.mlog("上传成功"), XxQianDuan.removeItem_info('储存_获取任务的数据');
          const local108 = 任务管理器.获取随机间隔() * 1000,
            local109 = local108 / 1000;
          slog.mlog('等待' + local109 + "秒开始下一条任务"), setTimeout(申请任务, local108);
          return;
        }
        if ((local107.includes('已保存') || local107.includes('success')) && (!local99 || local99.code !== "200")) {
          slog.mlog("上传成功"), XxQianDuan.removeItem_info("储存_获取任务的数据"), 任务计数.add();
          let local110 = parseInt(localStorage.getItem("开始任务计数") || '0'),
            local111 = 任务计数.get(),
            local112 = local111 - local110;
          if (local112 > 0 && local112 % 多少次暂停 === 0) {
            slog.mlog('批次完成'), XxQianDuan.亮牌子('purple'), 任务状态管理器.停止(), Xx框架.更新任务状态显示();
            return;
          }
          const local113 = 任务管理器.获取随机间隔() * 1000,
            local114 = local113 / 1000;
          slog.mlog('等待' + local114 + '秒开始下一条任务'), setTimeout(申请任务, local113);
          return;
        }
      }
      slog.mlog('提交结果未识别: ' + String(local98).slice(0, 300));
      const local118 = 任务管理器.获取随机间隔() * 1000;
      setTimeout(申请任务, local118);
    } catch (local115) {
      slog.mlog('上传商品异常: ' + (local115 && local115.message ? local115.message : local115));
      if (看门狗任务运行中()) {
        const localRetry = 任务管理器.获取随机间隔() * 1000;
        setTimeout(申请任务, localRetry);
      }
    }
  }
}
async function 开始任务() {
  let local116 = 任务计数.get();
  localStorage.setItem("开始任务计数", local116), 台湾任务重试计数.重置(), chrome.runtime.sendMessage({
    act: "CK时间"
  });
  let local117 = window.location.hostname;
  const local118 = 域名配置.台湾,
    local119 = local117.includes(local118.shopee) || local117.includes(local118.xiapibuy);
  if (!local119) {
    slog.mlog('请在台湾站点运行');
    return;
  }
  任务接口模式 = "old", localStorage.setItem('地区', '台湾'), localStorage.setItem("任务接口模式", 任务接口模式);
  const local120 = await 用户登录();
  if (!local120) {
    用户Token = null, slog.mlog('登录失败'), XxQianDuan.亮牌子("red");
    return;
  }
  任务状态管理器.开始(), Xx框架.更新任务状态显示(), 申请任务();
}
function 停止任务() {
  任务状态管理器.停止(), localStorage.setItem('储存_开始或停止', '停止'), 台湾任务重试计数.重置(), Xx框架.更新任务状态显示();
}
function 初始化语言() {
  (function () {})();
  var local121 = CookieManager.get("language");
  const local122 = window.location.host;
  local122 === 'shopee.tw' && local121 !== "zhHant" && (CookieManager.set('language', 'zhHant', {
    daysToLive: 365,
    domain: "shopee.tw"
  }), CookieManager.set("language", "zhHant", {
    daysToLive: 365,
    domain: ".shopee.tw"
  }), location.reload()), local122 === 'xiapi.xiapibuy.com' && local121 !== 'zhHant' && (CookieManager.set("language", "zhHant", {
    daysToLive: 365,
    domain: 'xiapi.xiapibuy.com'
  }), location.reload());
}
初始化语言();
let 按钮数组 = [{
    描述: "打开主页",
    字: '主页',
    函数: function () {
      window.location.href = '/';
    }
  }, {
    描述: '台湾',
    字: '台',
    函数: function () {
      跳转('https://shopee.tw/');
    }
  }, {
    描述: '日志',
    字: '日志',
    函数: function () {
      slog.显示日志(slog.简单);
    }
  }],
  数据数组 = [{
    颜色: "#00ff40",
    name: "任务计数"
  }, {
    颜色: "#4CAF50",
    name: "版本号"
  }];
const 日志最大条数 = 300;
let 日志刷新定时器 = null;
const slog = {
  简单: '简单',
  详细: '详细',
  log: function (param43, param44) {
    param43 = String(param43 === undefined || param43 === null ? '' : param43);
    if (!G) G = {};
    if (!Array.isArray(G.log)) G.log = [];
    const local123 = {
      内容: param43,
      级别: param44,
      时间: Xxlib.getDate("h:i:s", new Date()),
      toString() {
        return this.内容;
      }
    };
    G.log.push(local123);
    G.log.length > 日志最大条数 && (G.log = G.log.slice(-日志最大条数));
    this.刷新小日志();
    this.安排刷新大日志();
    chrome.storage.local.set({
      G: G
    }, function () {});
  },
  mlog: function (param45) {
    this.log(param45, this.简单);
  },
  获取日志: function (param46) {
    if (!G || !Array.isArray(G.log)) return [];
    return param46 === this.简单 ? G.log.filter(param47 => param47.级别 === this.简单) : G.log;
  },
  创建日志元素: function (param48) {
    const local126 = document.createElement("div");
    local126.className = "log-entry";
    const local127 = String(param48 && param48.内容 || '');
    if (local127.includes('失败') || local127.includes('错误') || local127.includes('异常')) local126.classList.add("error");else (local127.includes('警告') || local127.includes('注意')) && local126.classList.add("warning");
    Object.assign(local126.style, {
      marginBottom: "6px",
      padding: "6px 8px",
      borderLeft: local126.classList.contains("error") ? "3px solid #ef4444" : local126.classList.contains("warning") ? "3px solid #f59e0b" : "3px solid #16a34a",
      borderRadius: "7px",
      background: "#f8fafc",
      boxSizing: "border-box",
      wordBreak: "break-word"
    });
    const local128 = document.createElement("span");
    local128.className = "log-time";
    local128.textContent = "[" + (param48 && param48.时间 ? param48.时间 : Xxlib.getDate("h:i:s", new Date())) + "]";
    Object.assign(local128.style, {
      display: "block",
      marginBottom: "3px",
      color: "#94a3b8",
      fontSize: "10px"
    });
    const local129 = document.createElement("span");
    local129.className = "log-message";
    local129.textContent = local127;
    Object.assign(local129.style, {
      color: "#334155",
      fontSize: "12px",
      lineHeight: "1.45"
    });
    local126.appendChild(local128);
    local126.appendChild(local129);
    return local126;
  },
  刷新小日志: function () {
    const local130 = 隐藏元素 ? 隐藏元素.getElementById(this.小日志框ID) : null;
    if (!local130) return;
    local130.innerHTML = '';
    this.获取日志(this.简单).slice(-50).forEach(param49 => {
      local130.appendChild(this.创建日志元素(param49));
    });
    local130.scrollTop = local130.scrollHeight;
  },
  安排刷新大日志: function () {
    if (日志刷新定时器) return;
    日志刷新定时器 = setTimeout(() => {
      日志刷新定时器 = null;
      this.刷新大日志();
    }, 300);
  },
  刷新大日志: function () {
    if (!this.日志框) return;
    const local131 = document.getElementById(this.日志框);
    if (!local131 || local131.style.display === "none") return;
    local131.innerHTML = '';
    this.获取日志(this.当前日志级别 || this.简单).forEach(param50 => {
      local131.appendChild(this.创建日志元素(param50));
    });
    local131.scrollTop = local131.scrollHeight;
  },
  显示日志: function (param48) {
    this.当前日志级别 = param48;
    let local132 = null;
    if (this.日志框) local132 = $('#' + this.日志框);else {
      this.日志框 = Xxlib.随机签名();
      let local133 = " <div style='display: none; overflow-y: scroll; position: fixed; right: 0; top: 10px; color: #000000; background: white; border: 1px solid #e9e9e9; margin: 0 auto 1px; width: 350px; height: 97% !important; padding: 8px; z-index: 999998; box-sizing: border-box;' id='" + this.日志框 + '\x27></div>';
      $("body").append(local133), local132 = $('#' + this.日志框);
    }
    if (local132.is(':visible')) local132.hide(), local132.empty();else {
      local132.show();
      this.刷新大日志();
    }
  },
  清空当前日志: function (param50) {
    任务计数.重置(), 连续风控计数.重置(), 风控计数.重置(), G.log = [], this.刷新小日志(), this.刷新大日志(), chrome.storage.local.set({
      G: G
    }, function () {
      param50 && param50();
    });
  },
  打开页面初始化: function () {
    this.刷新小日志();
  }
};
var Xx项目_校验 = class {
    static ['校验账号名']() {
      let local132 = 账号名;
      if (local132 === '' || local132 === undefined || local132 === null || typeof local132 === "string" && local132.trim() === '' || local132 === 0 || local132 === false) return alert("账号名不能为空,必须填写账号名"), true;else {
        const local133 = /^[a-z0-9]{2,4}$/;
        return !local133.test(账号名) ? (alert("用户名不符合规范,只能使用小写字母和数字一共2-4位!!"), true) : false;
      }
    }
    static ["校验组名"]() {
      let local134 = 组ID;
      if (local134 === '' || local134 === undefined || local134 === null || typeof local134 === 'string' && local134.trim() === '' || local134 === 0 || local134 === false) return alert('组ID不能为空,必须填写组ID'), true;else {
        const local135 = /^[\u4e00-\u9fa5]{4}$/;
        return !local135.test(组ID) ? (alert("组ID不符合规范,只能使用4个字的汉字多一个少一个都不行"), true) : false;
      }
    }
    static ["校验页面是不是英文"]() {
      let local136 = this.getCookie('language');
      const local137 = window.location.host;
      if (local137 === "shopee.tw" || local137 === "xiapi.xiapibuy.com") return local136 !== "zhHant" ? (alert("台湾网页只允许繁体中文!"), true) : false;
      let local138 = Xx项目lib.获取当前的浏览地区();
      return local138 == 'TW' ? local136 === null || local136 === "zhHant" ? false : (alert("台湾网页只允许繁体中文!"), true) : local136 && local136.includes('en') ? false : (alert("页面只允许是英文或者你还没有登录"), true);
    }
    static ['校验地区']() {}
    static ['getCookie'](param52) {
      let local139 = param52 + '=',
        local140 = document.cookie.split(';');
      for (let local141 = 0; local141 < local140.length; local141++) {
        let local142 = local140[local141];
        while (local142.charAt(0) == '\x20') {
          local142 = local142.substring(1, local142.length);
        }
        if (local142.indexOf(local139) == 0) return local142.substring(local139.length, local142.length);
      }
      return null;
    }
  },
  Xx项目lib = class {
    static ["获取当前的浏览地区"]() {
      function fn1(param53) {
        var local143 = encodeURIComponent(param53) + '=',
          local144 = document.cookie.indexOf(local143),
          local145 = null;
        if (local144 > -1) {
          var local146 = document.cookie.indexOf(';', local144);
          local146 == -1 && (local146 = document.cookie.length), local145 = decodeURIComponent(document.cookie.substring(local144 + local143.length, local146));
        }
        return local145;
      }
      return fn1('__LOCALE__null');
    }
    static async ["申请任务2_前置检查"]() {
      if (window.location.href.includes("verify/captcha")) {
        slog.mlog('检测到验证码'), XxQianDuan.亮牌子("blue");
        var local147 = "tSm9iHO11111111",
          local148 = '检测到验证码页面，请处理！',
          local149 = document.createElement('img');
        return local149.src = "http://miaotixing.com/trigger?id=" + local147 + "&text=" + encodeURI(local148) + "&_=" + Date.now(), document.body.appendChild(local149), true;
      }
      if (!看门狗任务运行中()) {
        slog.mlog('任务未运行，已自动停止');
        return true;
      }
      return false;
    }
  },
  Xx项目_跳转 = class {
    static ['处理跳转'](param54) {
      Xx项目_跳转.处理跳转1(param54);
    }
    static ['处理跳转1'](param55) {
      window.location.href = param55;
    }
  };
function 检测是否为移动设备() {
  return false;
}
var Xx框架 = class {
    static ["创建框架窗口"]() {
      const local162 = 检测是否为移动设备(),
        local163 = document.createElement("div");
      隐藏元素 = local163.attachShadow({
        mode: "closed"
      }), document.body.appendChild(local163);
      let local164 = localStorage.getItem('地区') || '台湾',
        local165 = window.location.hostname;
      for (const [local192, local193] of Object.entries(域名配置)) {
        if (local165.includes(local193.shopee) || local165.includes(local193.xiapibuy)) {
          local164 = local192;
          break;
        }
      }
      const local166 = {
          台湾: '台'
        },
        local167 = local166[local164] || local164,
        local168 = 任务管理器.加载配置(),
        local169 = {
          开始: 开始按钮图标,
          停止: 停止按钮图标,
          设置: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M12 15.5A3.5 3.5 0 1 0 12 8a3.5 3.5 0 0 0 0 7.5Z\"></path><path d=\"M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06A1.7 1.7 0 0 0 15 19.36a1.7 1.7 0 0 0-1 .64l-.08.1a2 2 0 0 1-3.84-1.02l-.01-.09A1.7 1.7 0 0 0 9 17.7a1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.64 13a1.7 1.7 0 0 0-.64-1l-.1-.08a2 2 0 0 1 1.02-3.84l.09-.01A1.7 1.7 0 0 0 6.3 7a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 11 2.64a1.7 1.7 0 0 0 1-.64l.08-.1a2 2 0 0 1 3.84 1.02l.01.09A1.7 1.7 0 0 0 17 4.3a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.36 9c.35.36.56.72.64 1.08\"></path></svg>",
          主页: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M3 10.5 12 3l9 7.5\"></path><path d=\"M5 9.5V21h14V9.5\"></path><path d=\"M9 21v-7h6v7\"></path></svg>",
          台湾: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M12 21s7-4.7 7-11a7 7 0 1 0-14 0c0 6.3 7 11 7 11Z\"></path><circle cx=\"12\" cy=\"10\" r=\"2.5\"></circle></svg>",
          清除: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M4 7h16\"></path><path d=\"M10 11v6\"></path><path d=\"M14 11v6\"></path><path d=\"M6 7l1 14h10l1-14\"></path><path d=\"M9 7V4h6v3\"></path></svg>",
          日志: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M8 6h12\"></path><path d=\"M8 12h12\"></path><path d=\"M8 18h12\"></path><path d=\"M4 6h.01\"></path><path d=\"M4 12h.01\"></path><path d=\"M4 18h.01\"></path></svg>",
          最小化: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M6 12h12\"></path></svg>",
          还原: '<svg\x20viewBox=\x220\x200\x2024\x2024\x22\x20aria-hidden=\x22true\x22><rect\x20x=\x227\x22\x20y=\x227\x22\x20width=\x2210\x22\x20height=\x2210\x22\x20rx=\x222\x22></rect></svg>',
          关闭: "<svg viewBox=\"0 0 24 24\" aria-hidden=\"true\"><path d=\"M6 6l12 12\"></path><path d=\"M18 6 6 18\"></path></svg>"
        };
      Object.keys(local169).forEach(param57 => {
        local169[param57] = local169[param57].replace("<svg ", "<svg width=\"18\" height=\"18\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\" style=\"display:block;flex:none\" ").replace(/<circle /g, "<circle fill=\"none\" ").replace(/<rect /g, '<rect\x20fill=\x22none\x22\x20');
      });
      const local170 = param58 => ({
          打开主页: local169.主页,
          台湾: local169.台湾,
          清除日志: local169.清除,
          日志: local169.日志
        })[param58.描述] || local169.日志,
        local171 = {
          描述: '配置',
          字: '设置',
          函数: function () {
            Xx框架.显示配置面板();
          }
        };
      local171.ID = Xxlib.随机签名();
      let local172 = "\n      <button class=\"func-btn config-btn\" title=\"配置\" id=\"" + local171.ID + "\">\n        <span class=\"func-icon\">" + local169.设置 + "</span>\n        <span class=\"func-label\">设置</span>\n      </button>\n    ";
      for (let local194 of 按钮数组) {
        local194.ID = Xxlib.随机签名(), local172 += "\n        <button class=\"func-btn\" title=\"" + local194.描述 + '\x22\x20id=\x22' + local194.ID + "\">\n          <span class=\"func-icon\">" + local170(local194) + "</span>\n          <span class=\"func-label\">" + local194.字 + "</span>\n        </button>\n      ";
      }
      let local173 = '';
      for (let local195 of 数据数组) {
        if (local195.name !== "任务计数" && local195.name !== '版本号') continue;
        local195.ID = Xxlib.随机签名(), local173 += "\n        <div class=\"stat-card\">\n          <div class=\"stat-content\">\n            <div class=\"stat-name\">" + local195.name + '</div>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22stat-value\x22\x20id=\x22' + local195.ID + '\x22>0</div>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20</div>\x0a\x20\x20\x20\x20\x20\x20\x20\x20</div>\x0a\x20\x20\x20\x20\x20\x20';
      }
      const local174 = Xxlib.随机签名();
      slog.小日志框ID = local174;
      const local175 = '\x0a\x20\x20\x20\x20\x20\x20<div\x20class=\x22dashboard-container\x22\x20id=\x22dashboardContainer\x22>\x0a\x20\x20\x20\x20\x20\x20\x20\x20<!--\x20头部区域\x20-->\x0a\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22dashboard-header\x22\x20id=\x22dashboardHeader\x22>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22header-left\x22>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22brand-mark\x22>A</div>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22header-copy\x22>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22header-title\x22>Ajie</div>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<div\x20class=\x22header-region\x22>' + local167 + "</div>\n            </div>\n          </div>\n          <div class=\"header-controls\">\n            <button class=\"window-btn minimize-btn\" title=\"最小化\" id=\"minimizeBtn\">" + local169.最小化 + "</button>\n            <button class=\"window-btn close-btn\" title=\"关闭\" id=\"closeBtn\">" + local169.关闭 + "</button>\n          </div>\n        </div>\n\n        <!-- 缩放控制 -->\n        <div class=\"resize-controls\">\n          <div class=\"resize-handle\" id=\"resizeHandle\"></div>\n        </div>\n\n        <!-- 主控制区域（按钮和计数一行显示） -->\n        <div class=\"control-panel\">\n          <div class=\"control-row\">\n            <button class=\"toggle-btn\" title=\"开始/停止任务\" id=\"toggleTaskBtn\">\n              <span class=\"btn-icon\" id=\"toggleIcon\">" + local169.开始 + "</span>\n              <span class=\"btn-text\" id=\"toggleLabel\">开始</span>\n            </button>\n            <div class=\"task-count-display\">\n              <div class=\"task-count-caption\">任务</div>\n              <div class=\"task-count-value\" id=\"topTaskCount\">0</div>\n            </div>\n          </div>\n        </div>\n\n        <!-- 功能按钮 -->\n        <div class=\"functions-panel\">\n          <div class=\"panel-title\">操作</div>\n          <div class=\"func-buttons\">\n            " + local172 + "\n          </div>\n        </div>\n\n        <!-- 日志区域 -->\n        <div class=\"logs-panel\">\n          <div class=\"logs-header\">\n            <div class=\"panel-title\">运行日志</div>\n            <button class=\"clear-log-btn\" title=\"清空日志\" id=\"clearLogBtn\">\n              <span class=\"func-icon\">" + local169.清除 + "</span>\n            </button>\n          </div>\n          <div class=\"logs-container\" id=\"" + local174 + "\"></div>\n        </div>\n\n        <!-- 配置面板（隐藏） -->\n        <div class=\"config-panel\" id=\"configPanel\" style=\"display: none;\">\n          <div class=\"config-header\">\n            <div class=\"panel-title\">参数配置</div>\n            <button class=\"close-config-btn\" id=\"closeConfigBtn\">×</button>\n          </div>\n          <div class=\"config-content\">\n            <div class=\"config-group\">\n              <label class=\"config-label\">任务间隔 (秒)</label>\n              <div class=\"range-inputs\">\n                <input type=\"number\" id=\"minInterval\" min=\"0.1\" max=\"3600\" step=\"0.1\" value=\"" + local168.任务间隔最小 + '\x22\x20class=\x22config-input\x22>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<span\x20class=\x22range-separator\x22>至</span>\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20<input\x20type=\x22number\x22\x20id=\x22maxInterval\x22\x20min=\x220.1\x22\x20max=\x223600\x22\x20step=\x220.1\x22\x20value=\x22' + local168.任务间隔最大 + "\" class=\"config-input\">\n              </div>\n              <div class=\"config-hint\">每次任务之间的随机延迟时间</div>\n            </div>\n\n            <div class=\"config-group\">\n              <label class=\"config-label\">批次任务数</label>\n              <input type=\"number\" id=\"batchCount\" min=\"1\" max=\"1000\" value=\"" + local168.批次任务数 + "\" class=\"config-input\">\n              <div class=\"config-hint\">每批次执行的任务数量</div>\n            </div>\n\n            <div class=\"config-actions\">\n              <button class=\"config-btn save-btn\" id=\"saveConfigBtn\">保存配置</button>\n              <button class=\"config-btn reset-btn\" id=\"resetConfigBtn\">恢复默认</button>\n            </div>\n          </div>\n        </div>\n\n        <!-- 底部状态栏 -->\n        <div class=\"status-bar\">\n          <span class=\"status-text\" id=\"statusText\">就绪</span>\n          <span class=\"status-time\" id=\"currentTime\"></span>\n        </div>\n      </div>\n\n      <style>\n        /* 全局样式 */\n        :host {\n          all: initial;\n        }\n\n        .dashboard-container {\n          position: fixed;\n          left: 10px;\n          top: 10px;\n          width: " + (local162 ? "calc(100vw - 20px)" : "112px") + ";\n          height: " + (local162 ? 'calc(100vh\x20-\x2020px)' : '388px') + ';\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-height:\x20' + (local162 ? 'calc(100vh\x20-\x2020px)' : "388px") + ";\n          background: #ffffff;\n          border: 1px solid #e0e0e0;\n          border-radius: 4px;\n          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);\n          z-index: 10000;\n          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Microsoft YaHei', sans-serif;\n          overflow: hidden;\n          color: #333333;\n          transition: box-shadow 0.2s ease;\n          resize: " + (local162 ? "none" : "none") + ';\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x20' + (local162 ? "calc(100vw - 20px)" : "112px") + ';\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-height:\x20' + (local162 ? "300px" : "376px") + ";\n          max-width: " + (local162 ? "calc(100vw - 20px)" : "112px") + ";\n          max-height: " + (local162 ? "calc(100vh - 20px)" : "376px") + ";\n        }\n\n        .dashboard-container.minimized {\n          width: 240px !important;\n          height: 50px !important;\n          min-width: 240px !important;\n          min-height: 50px !important;\n          max-width: 240px !important;\n          max-height: 50px !important;\n          overflow: hidden;\n          padding: 0 !important;\n        }\n\n        .dashboard-container.minimized .control-panel,\n        .dashboard-container.minimized .functions-panel,\n        .dashboard-container.minimized .logs-panel,\n        .dashboard-container.minimized .config-panel,\n        .dashboard-container.minimized .status-bar,\n        .dashboard-container.minimized .resize-controls {\n          display: none !important;\n        }\n\n        .dashboard-container.minimized .dashboard-header {\n          padding: 8px 12px !important;\n          height: 40px !important;\n          cursor: pointer;\n          background: #f5f5f5 !important;\n          border-bottom: 1px solid #e0e0e0 !important;\n        }\n\n        .dashboard-container.minimized .header-left {\n          display: flex;\n          align-items: center;\n          gap: 10px;\n          width: 100%;\n          justify-content: space-between;\n        }\n\n        .dashboard-container.minimized .header-title {\n          font-size: 13px;\n          font-weight: 500;\n          flex-shrink: 0;\n        }\n\n        .dashboard-container.minimized .header-region {\n          padding: 2px 6px;\n          font-size: 11px;\n          background: #e8e8e8;\n          border: 1px solid #d0d0d0;\n          flex-shrink: 0;\n        }\n\n        .dashboard-container.minimized .task-count {\n          display: flex !important;\n          align-items: center;\n          gap: 4px;\n          background: #f0f0f0;\n          border: 1px solid #d0d0d0;\n          padding: 2px 6px;\n          font-size: 11px;\n          flex-shrink: 0;\n          margin-left: 4px;\n        }\n\n        .dashboard-container.minimized .task-count-label {\n          color: #666;\n        }\n\n        .dashboard-container.minimized .task-count-value {\n          font-weight: 600;\n          color: #333;\n        }\n\n        .dashboard-container.minimized .toggle-btn {\n          display: flex !important;\n          align-items: center;\n          gap: 4px;\n          background: #f0f0f0;\n          border: 1px solid #d0d0d0;\n          color: #333;\n          padding: 3px 8px;\n          font-size: 11px;\n          cursor: pointer;\n          transition: background 0.2s ease;\n          flex-shrink: 0;\n          margin-left: 4px;\n        }\n\n        .dashboard-container.minimized .toggle-btn:hover {\n          background: #e0e0e0;\n        }\n\n        .dashboard-container.minimized .toggle-btn.running {\n          background: #4caf50 !important;\n          color: white !important;\n          border-color: #4caf50 !important;\n        }\n\n        .dashboard-container.minimized .toggle-btn.stopped {\n          background: #f44336 !important;\n          color: white !important;\n          border-color: #f44336 !important;\n        }\n\n        .dashboard-container.minimized .header-controls {\n          display: flex !important;\n          gap: 4px;\n          flex-shrink: 0;\n          margin-left: 4px;\n        }\n\n        .dashboard-container.minimized .minimize-btn,\n        .dashboard-container.minimized .close-btn {\n          width: 18px !important;\n          height: 18px !important;\n          font-size: 10px !important;\n          padding: 0 !important;\n        }\n\n        .dashboard-container:hover {\n          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);\n        }\n\n        /* 头部样式 */\n        .dashboard-header {\n          display: flex;\n          justify-content: space-between;\n          align-items: center;\n          padding: " + (local162 ? '12px\x2010px' : "4px 4px") + ";\n          background: #f5f5f5;\n          border-bottom: 1px solid #e0e0e0;\n          cursor: move;\n          user-select: none;\n        }\n\n        .header-left {\n          display: flex;\n          align-items: center;\n          gap: " + (local162 ? '6px' : '8px') + ";\n          flex: 1;\n          overflow: hidden;\n        }\n\n        .header-title {\n          font-size: " + (local162 ? "16px" : "10px") + ";\n          font-weight: 500;\n          color: #333;\n        }\n\n        .header-region {\n          padding: " + (local162 ? "4px 8px" : "1px 3px") + ";\n          background: #e8e8e8;\n          border: 1px solid #d0d0d0;\n          font-size: " + (local162 ? "12px" : "8px") + ";\n          color: #666;\n        }\n\n        .header-controls {\n          display: flex;\n          gap: 4px;\n        }\n\n        .minimize-btn, .close-btn {\n          background: transparent;\n          border: 1px solid #d0d0d0;\n          color: #666;\n          width: 16px;\n          height: 16px;\n          border-radius: 0;\n          cursor: pointer;\n          display: flex;\n          align-items: center;\n          justify-content: center;\n          font-size: 9px;\n          transition: background 0.2s ease;\n        }\n\n        .minimize-btn:hover, .close-btn:hover {\n          background: #e0e0e0;\n        }\n\n        .close-btn:hover {\n          background: #f44336;\n          color: white;\n          border-color: #f44336;\n        }\n\n        /* 缩放控制 */\n        .resize-controls {\n          position: absolute;\n          bottom: 0;\n          right: 0;\n          width: 12px;\n          height: 12px;\n          cursor: se-resize;\n          z-index: 10001;\n        }\n\n        .resize-handle {\n          position: absolute;\n          bottom: 0;\n          right: 0;\n          width: 12px;\n          height: 12px;\n          background: #e0e0e0;\n          border-top: 1px solid #d0d0d0;\n          border-left: 1px solid #d0d0d0;\n        }\n\n        .resize-handle:hover {\n          background: #d0d0d0;\n        }\n\n        /* 面板通用样式 */\n        .control-panel,\n        .functions-panel,\n        .logs-panel {\n          padding: " + (local162 ? "12px 10px" : "4px 2px") + ";\n          background: #ffffff;\n          margin: 0;\n          border-bottom: 1px solid #e0e0e0;\n        }\n\n        .panel-title {\n          margin-bottom: " + (local162 ? "10px" : '4px') + ";\n          font-size: " + (local162 ? '13px' : "10px") + ";\n          font-weight: bold;\n          color: #666;\n          text-transform: uppercase;\n          letter-spacing: 0.3px;\n        }\n\n        /* 控制按钮 */\n        .control-buttons {\n          display: flex;\n          justify-content: center;\n        }\n\n        .control-panel {\n          padding: " + (local162 ? "12px 10px" : "6px 2px") + " !important;\n        }\n\n        .control-row {\n          display: flex;\n          align-items: center;\n          gap: 4px;\n          height: " + (local162 ? '60px' : "36px") + ";\n        }\n\n        .toggle-btn {\n          flex: 1;\n          height: 100%;\n          border: 2px solid #d0d0d0;\n          background: #4caf50;\n          color: white;\n          cursor: pointer;\n          display: flex;\n          align-items: center;\n          justify-content: center;\n          padding: 0;\n          transition: all 0.2s ease;\n          font-size: " + (local162 ? "40px" : "24px") + ";\n          font-weight: 700;\n          border-radius: 4px;\n        }\n\n        .divider {\n          width: 1px;\n          height: 70%;\n          background: #e0e0e0;\n          flex-shrink: 0;\n        }\n\n        .task-count-display {\n          flex: 1;\n          display: flex;\n          align-items: center;\n          justify-content: center;\n          height: 100%;\n          background: #f9f9f9;\n          border-radius: 4px;\n        }\n\n        .task-count-value {\n          font-size: " + (local162 ? "32px" : "18px") + ";\n          font-weight: 700;\n          color: #333;\n          line-height: 1;\n        }\n\n        .toggle-btn:hover {\n          background: #45a049;\n          transform: scale(1.02);\n        }\n\n        .toggle-btn.running {\n          background: #f44336 !important;\n          border-color: #f44336 !important;\n        }\n\n        .toggle-btn.running:hover {\n          background: #e53935 !important;\n        }\n\n        .btn-text {\n          font-size: inherit;\n          font-weight: inherit;\n          line-height: 1;\n        }\n\n        .btn-icon {\n          display: none;\n        }\n\n        .btn-label {\n          display: none;\n        }\n\n        /* 统计数据 */\n        .stats-grid {\n          display: grid;\n          grid-template-columns: " + (local162 ? "1fr 1fr" : "1fr 1fr") + ";\n          gap: " + (local162 ? "10px" : "4px") + ';\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.stat-card\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f9f9f9;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x201px\x20solid\x20#e0e0e0;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x20' + (local162 ? '12px' : "4px 2px") + ";\n          transition: background 0.2s ease;\n        }\n\n        .stat-card:hover {\n          background: #f0f0f0;\n        }\n\n        .stat-content {\n          flex: 1;\n        }\n\n        .stat-name {\n          font-size: " + (local162 ? "11px" : "7px") + ";\n          color: #999;\n          margin-bottom: " + (local162 ? '6px' : '1px') + ";\n          text-transform: uppercase;\n        }\n\n        .stat-value {\n          font-size: " + (local162 ? '24px' : '14px') + ";\n          font-weight: 600;\n          color: #333;\n        }\n\n        /* 任务计数（最小化时显示） */\n        .task-count {\n          display: none;\n        }\n\n        /* 功能按钮 */\n        .func-buttons {\n          display: grid;\n          grid-template-columns: " + (local162 ? "repeat(2, 1fr)" : "repeat(3, 1fr)") + ";\n          gap: " + (local162 ? "8px" : '3px') + ";\n          width: 100%;\n        }\n\n        .func-btn {\n          background: #f9f9f9;\n          border: 1px solid #e0e0e0;\n          color: #333;\n          padding: 0;\n          cursor: pointer;\n          display: flex;\n          align-items: center;\n          justify-content: center;\n          transition: background 0.2s ease;\n          font-size: " + (local162 ? "13px" : '9px') + ";\n          height: " + (local162 ? "40px" : "28px") + ';\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x20100%;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-width:\x20100%;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x200;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20box-sizing:\x20border-box;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20bold;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20white-space:\x20nowrap;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20overflow:\x20hidden;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f0f0f0;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-label\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x20' + (local162 ? "13px" : "9px") + ";\n          white-space: nowrap;\n          overflow: hidden;\n          text-overflow: ellipsis;\n          width: 100%;\n          max-width: 100%;\n          text-align: center;\n          line-height: 1.2;\n          font-weight: bold;\n          padding: 0 1px;\n        }\n\n        /* 日志区域 */\n        .logs-header {\n          display: flex;\n          justify-content: space-between;\n          align-items: center;\n          margin-bottom: 4px;\n        }\n\n        .clear-log-btn {\n          background: #f9f9f9;\n          border: 1px solid #e0e0e0;\n          color: #666;\n          padding: 2px 6px;\n          cursor: pointer;\n          font-size: 8px;\n          transition: background 0.2s ease;\n        }\n\n        .clear-log-btn:hover {\n          background: #f0f0f0;\n        }\n\n        .logs-container {\n          height: " + (local162 ? "180px" : "100px") + ";\n          overflow-y: auto;\n          background: #f9f9f9;\n          border: 1px solid #e0e0e0;\n          padding: " + (local162 ? "10px" : "2px") + ";\n          font-family: 'Consolas', 'Monaco', monospace;\n          font-size: " + (local162 ? "12px" : "7px") + ";\n          line-height: 1.2;\n        }\n\n        .logs-container::-webkit-scrollbar {\n          width: 6px;\n        }\n\n        .logs-container::-webkit-scrollbar-track {\n          background: #f0f0f0;\n        }\n\n        .logs-container::-webkit-scrollbar-thumb {\n          background: #d0d0d0;\n        }\n\n        .logs-container::-webkit-scrollbar-thumb:hover {\n          background: #b0b0b0;\n        }\n\n        .log-entry {\n          margin-bottom: 4px;\n          padding: 4px 6px;\n          background: #ffffff;\n          border-left: 2px solid #4caf50;\n          animation: fadeIn 0.2s ease;\n        }\n\n        .log-entry.error {\n          border-left-color: #f44336;\n        }\n\n        .log-entry.warning {\n          border-left-color: #ff9800;\n        }\n\n        .log-time {\n          color: #999;\n          font-size: 9px;\n          margin-right: 6px;\n        }\n\n        .log-message {\n          color: #333;\n          font-size: 10px;\n        }\n\n        .small-text {\n          color: #333;\n          font-size: 8px;\n          margin: 1px 0;\n          font-family: 'Consolas', 'Monaco', monospace;\n        }\n\n        .small-text2 {\n          color: #e0e0e0;\n          font-size: 9px;\n          margin: 2px 0;\n        }\n\n        /* 配置面板 */\n        .config-panel {\n          position: absolute;\n          top: 0;\n          left: 0;\n          right: 0;\n          bottom: 0;\n          background: #ffffff;\n          z-index: 100;\n          animation: slideIn 0.2s ease;\n        }\n\n        @keyframes slideIn {\n          from { transform: translateY(-10px); opacity: 0; }\n          to { transform: translateY(0); opacity: 1; }\n        }\n\n        .config-header {\n          display: flex;\n          justify-content: space-between;\n          align-items: center;\n          padding: 10px 12px;\n          border-bottom: 1px solid #e0e0e0;\n          background: #f5f5f5;\n        }\n\n        .close-config-btn {\n          background: transparent;\n          border: 1px solid #d0d0d0;\n          color: #666;\n          width: 22px;\n          height: 22px;\n          cursor: pointer;\n          display: flex;\n          align-items: center;\n          justify-content: center;\n          font-size: 14px;\n          transition: background 0.2s ease;\n        }\n\n        .close-config-btn:hover {\n          background: #e0e0e0;\n        }\n\n        .config-content {\n          padding: 12px;\n          max-height: calc(100% - 44px);\n          overflow-y: auto;\n        }\n\n        .config-group {\n          margin-bottom: 16px;\n        }\n\n        .config-group label {\n          display: block;\n          margin-bottom: 6px;\n          font-size: 12px;\n          font-weight: 500;\n          color: #333;\n        }\n\n        .range-inputs {\n          display: flex;\n          flex-wrap: wrap;\n          align-items: center;\n          gap: 6px;\n        }\n\n        .range-separator {\n          color: #999;\n          font-size: 12px;\n          width: 100%;\n          text-align: center;\n        }\n\n        .config-input {\n          width: 100%;\n          padding: 6px 8px;\n          background: #ffffff;\n          border: 1px solid #d0d0d0;\n          color: #333;\n          font-size: 13px;\n          text-align: center;\n          transition: border-color 0.2s ease;\n        }\n\n        .range-inputs .config-input {\n          flex: 1 1 calc(50% - 6px);\n          min-width: 0;\n        }\n\n        .config-input:focus {\n          outline: none;\n          border-color: #4caf50;\n        }\n\n        .config-hint {\n          font-size: 10px;\n          color: #999;\n          margin-top: 4px;\n        }\n\n        .config-actions {\n          display: flex;\n          gap: 8px;\n          margin-top: 16px;\n        }\n\n        .config-btn {\n          flex: 1;\n          padding: 8px;\n          border: 1px solid #d0d0d0;\n          font-size: 12px;\n          font-weight: 500;\n          cursor: pointer;\n          transition: background 0.2s ease;\n        }\n\n        .save-btn {\n          background: #4caf50;\n          color: white;\n          border-color: #4caf50;\n        }\n\n        .save-btn:hover {\n          background: #45a049;\n        }\n\n        .reset-btn {\n          background: #f9f9f9;\n          color: #333;\n        }\n\n        .reset-btn:hover {\n          background: #f0f0f0;\n        }\n\n        /* 底部状态栏 */\n        .status-bar {\n          display: flex;\n          justify-content: space-between;\n          align-items: center;\n          padding: " + (local162 ? "8px 10px" : '2px\x204px') + ";\n          background: #f5f5f5;\n          border-top: 1px solid #e0e0e0;\n          font-size: " + (local162 ? "12px" : "7px") + ";\n        }\n\n        #statusText {\n          color: #666;\n          max-width: " + (local162 ? "calc(100vw - 100px)" : "80px") + ";\n          white-space: nowrap;\n          overflow: hidden;\n          text-overflow: ellipsis;\n        }\n\n        .status-time {\n          color: #999;\n          font-size: 8px;\n        }\n\n        /* 动画 */\n        @keyframes fadeIn {\n          from { opacity: 0; }\n          to { opacity: 1; }\n        }\n\n        @keyframes pulse {\n          0% { transform: scale(1); }\n          50% { transform: scale(1.02); }\n          100% { transform: scale(1); }\n        }\n\n        .pulse-animation {\n          animation: pulse 0.3s ease;\n        }\n\n        /* 移动设备特殊样式 */\n        " + (local162 ? '\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20left:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20top:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x20100vw\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x20100vh\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-width:\x20100vw\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-height:\x20100vh\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x200;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.resize-controls\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.minimize-btn,\x20.close-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2028px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2028px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2016px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x20\x20\x20\x20\x20\x20\x20\x20' : '') + "\n\n        /* 响应式设计 */\n        @media (max-width: 400px) {\n          .dashboard-container {\n            width: 260px;\n          }\n\n          .func-buttons {\n            grid-template-columns: repeat(3, 1fr);\n          }\n        }\n\n        /* Ajie界面优化 */\n        .dashboard-container {\n          width: " + (local162 ? '100vw' : '138px') + '\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x20' + (local162 ? "100vh" : "426px") + " !important;\n          min-width: " + (local162 ? "100vw" : "138px") + " !important;\n          min-height: " + (local162 ? "100vh" : '426px') + " !important;\n          max-width: " + (local162 ? "100vw" : "138px") + " !important;\n          max-height: " + (local162 ? "100vh" : "426px") + " !important;\n          background: #f8fafc !important;\n          border: 1px solid #d9e1ea !important;\n          border-radius: " + (local162 ? '0' : '8px') + '\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20box-shadow:\x200\x2014px\x2034px\x20rgba(15,\x2023,\x2042,\x200.18)\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#172033\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-family:\x20-apple-system,\x20BlinkMacSystemFont,\x20\x27Segoe\x20UI\x27,\x20\x27Microsoft\x20YaHei\x27,\x20sans-serif\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20letter-spacing:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-header\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2046px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x207px\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#111827\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-bottom:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.header-left\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x200;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.brand-mark\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2027px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2027px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x207px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20flex;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20align-items:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20justify-content:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20flex:\x200\x200\x20auto;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#22c55e;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2015px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20800;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.header-copy\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20flex;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20flex-direction:\x20column;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x201px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x200;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.header-title\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2013px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201.1;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.header-region\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20transparent\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#a7f3d0\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201.1;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.header-controls\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x203px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.window-btn,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.minimize-btn,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.close-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2022px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2022px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20rgba(255,\x20255,\x20255,\x200.09)\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#dbeafe\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.window-btn:hover,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.minimize-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20rgba(255,\x20255,\x20255,\x200.16)\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.close-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ef4444\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.window-btn\x20svg,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-icon\x20svg,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.btn-icon\x20svg\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2015px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2015px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20fill:\x20none;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20stroke:\x20currentColor;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20stroke-width:\x202;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20stroke-linecap:\x20round;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20stroke-linejoin:\x20round;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20block;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.control-panel,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.functions-panel,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.logs-panel\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x209px\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f8fafc\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-bottom:\x201px\x20solid\x20#e5ebf2\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.control-row\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2048px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20grid\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20grid-template-columns:\x201fr\x2045px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.toggle-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2048px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x200;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#16a34a\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20box-shadow:\x200\x206px\x2014px\x20rgba(22,\x20163,\x2074,\x200.24);\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20flex-direction:\x20column;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x203px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20transform:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.toggle-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#15803d\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20transform:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.toggle-btn.running\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#dc2626\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-color:\x20transparent\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20box-shadow:\x200\x206px\x2014px\x20rgba(220,\x2038,\x2038,\x200.22);\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.toggle-btn.running:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#b91c1c\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.btn-icon\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20flex\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20align-items:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20justify-content:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.btn-icon\x20svg\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2018px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2018px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.btn-text\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20block\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2011px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.divider\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.task-count-display\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2048px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x201px\x20solid\x20#dfe7ef\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20flex-direction:\x20column;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x202px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.task-count-caption\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#64748b;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20600;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.task-count-value\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#0f172a\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2019px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20800\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.panel-title\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20margin-bottom:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#475569\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2011px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20text-transform:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20letter-spacing:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-buttons\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20grid-template-columns:\x20repeat(2,\x20minmax(0,\x201fr))\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2034px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x201px\x20solid\x20#dfe7ef\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#334155\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x204px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20flex-direction:\x20column;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#eef6ff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-color:\x20#b8cde5\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#0f172a\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-icon\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20flex;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20align-items:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20justify-content:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-icon\x20svg\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2014px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2014px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.func-label\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20block\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x20auto\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20inherit\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x209px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.logs-header\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20margin-bottom:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.clear-log-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2024px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x200\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x201px\x20solid\x20#dfe7ef\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#475569\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20inline-flex;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20align-items:\x20center;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20gap:\x204px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-weight:\x20700;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.clear-log-btn\x20.func-icon\x20svg\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x2012px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2012px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.clear-log-btn:hover\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f1f5f9\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.logs-container\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x20133px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x201px\x20solid\x20#dfe7ef\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201.35\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.log-entry\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20margin-bottom:\x205px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x205px\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-left:\x202px\x20solid\x20#16a34a\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x204px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f8fafc\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.log-time\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20block;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20margin:\x200\x200\x202px\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x209px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#94a3b8\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.log-message,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.small-text\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#334155\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20line-height:\x201.35\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20word-break:\x20break-word;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.status-bar\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2026px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x200\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-top:\x201px\x20solid\x20#e5ebf2\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#64748b\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x2010px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20#statusText\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-width:\x2078px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#64748b\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.status-time\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#94a3b8\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20font-size:\x209px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.resize-controls\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20display:\x20none\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.config-panel\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#f8fafc\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.config-header\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#111827\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-bottom:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.close-config-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20rgba(255,\x20255,\x20255,\x200.09)\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.config-input\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-color:\x20#dfe7ef\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.config-btn\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container.minimized\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20width:\x20172px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2042px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-width:\x20172px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20min-height:\x2042px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-width:\x20172px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20max-height:\x2042px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x208px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container.minimized\x20.dashboard-header\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20height:\x2042px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x206px\x207px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20#111827\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-bottom:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container.minimized\x20.task-count\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20background:\x20rgba(255,\x20255,\x20255,\x200.09)\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border:\x200\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20border-radius:\x206px;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20padding:\x204px\x206px\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container.minimized\x20.task-count-label,\x0a\x20\x20\x20\x20\x20\x20\x20\x20.dashboard-container.minimized\x20.task-count-value\x20{\x0a\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20color:\x20#ffffff\x20!important;\x0a\x20\x20\x20\x20\x20\x20\x20\x20}\x0a\x20\x20\x20\x20\x20\x20</style>\x0a\x20\x20\x20\x20';
      隐藏元素.innerHTML = local175;
      const localBrandIcon = typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.getURL ? chrome.runtime.getURL("icons/icon-32.png") : "icons/icon-32.png",
        localBrandMark = 隐藏元素.querySelector(".brand-mark");
      localBrandMark && (localBrandMark.innerHTML = "<img src=\"" + localBrandIcon + "\" alt=\"\">");
      const localHeaderTitle = 隐藏元素.querySelector(".header-title");
      localHeaderTitle && (localHeaderTitle.textContent = "Ajie");
      const local176 = (param59, param60, param61 = false) => {
          const local196 = param61 ? 隐藏元素.querySelectorAll(param59) : [隐藏元素.querySelector(param59)];
          local196.forEach(param62 => {
            if (param62) Object.assign(param62.style, param60);
          });
        },
        local177 = {
          width: "100%",
          height: '38px',
          border: "1px solid #d9e4ef",
          borderRadius: '8px',
          background: "#ffffff",
          color: "#1f2a37",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: "3px",
          padding: '0',
          boxSizing: 'border-box',
          cursor: "pointer",
          boxShadow: "0 1px 2px rgba(15, 23, 42, 0.05)"
        };
      local176(".dashboard-container", {
        width: local162 ? "100vw" : "156px",
        height: local162 ? "100vh" : "352px",
        minWidth: local162 ? "100vw" : "156px",
        minHeight: local162 ? "100vh" : "352px",
        maxWidth: local162 ? "100vw" : "156px",
        maxHeight: local162 ? "100vh" : "352px",
        background: "#f6f8fb",
        border: "1px solid #d6e0ea",
        borderRadius: local162 ? '0' : "8px",
        boxShadow: "0 14px 34px rgba(15, 23, 42, 0.22)",
        overflow: "hidden",
        color: "#172033",
        backdropFilter: "blur(8px)"
      }), local176(".dashboard-header", {
        height: '44px',
        padding: "7px 8px",
        background: "#172033",
        borderBottom: '0',
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        boxSizing: "border-box"
      }), local176(".header-left", {
        display: "flex",
        alignItems: "center",
        gap: "7px",
        minWidth: '0'
      }), local176(".brand-mark", {
        width: "26px",
        height: "26px",
        borderRadius: "8px",
        background: "#14b86a",
        color: "#ffffff",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: "15px",
        fontWeight: "800",
        flex: "0 0 auto",
        boxShadow: "inset 0 0 0 1px rgba(255,255,255,0.2)"
      }), local176(".header-copy", {
        display: "flex",
        flexDirection: "column",
        gap: '1px',
        minWidth: '0'
      }), local176(".header-title", {
        color: "#ffffff",
        fontSize: '13px',
        fontWeight: "800",
        lineHeight: "1.05"
      }), local176(".header-region", {
        color: "#aeeed0",
        fontSize: "9px",
        background: "transparent",
        border: '0',
        padding: '0',
        lineHeight: '1'
      }), local176(".header-controls", {
        display: "flex",
        gap: "4px",
        alignItems: "center"
      }), local176(".window-btn", {
        width: "22px",
        height: "22px",
        border: '0',
        borderRadius: "6px",
        background: "rgba(255,255,255,0.11)",
        color: "#e5e7eb",
        display: "flex",
        alignItems: "center",
        justifyContent: 'center',
        padding: '0',
        cursor: "pointer"
      }, true), local176(".control-panel", {
        padding: "9px 8px",
        background: '#f6f8fb',
        borderBottom: "1px solid #e5ebf2"
      }), local176(".control-row", {
        height: "48px",
        display: "grid",
        gridTemplateColumns: "1fr 46px",
        gap: "7px",
        alignItems: "stretch"
      }), local176(".toggle-btn", {
        height: "48px",
        border: '0',
        borderRadius: '8px',
        background: "#159a52",
        color: "#ffffff",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: "2px",
        padding: '0',
        boxShadow: "0 7px 16px rgba(21, 154, 82, 0.26)",
        cursor: "pointer"
      }), local176(".btn-icon", {
        display: "flex",
        alignItems: "center",
        justifyContent: "center"
      }), local176(".btn-text", {
        display: "block",
        fontSize: "11px",
        fontWeight: "800",
        lineHeight: '1'
      }), local176(".task-count-display", {
        height: "48px",
        borderRadius: '8px',
        background: "#ffffff",
        border: "1px solid #dfe7ef",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: "1px"
      }), local176('.task-count-caption', {
        color: "#667085",
        fontSize: "9px",
        fontWeight: "700",
        lineHeight: '1'
      }), local176(".task-count-value", {
        color: '#0f172a',
        fontSize: "18px",
        fontWeight: '800',
        lineHeight: '1'
      }), local176('.functions-panel', {
        padding: "8px 8px",
        background: "#f6f8fb",
        borderBottom: "1px solid #e5ebf2"
      }), local176(".panel-title", {
        color: "#506070",
        fontSize: "10px",
        fontWeight: '800',
        marginBottom: '6px',
        letterSpacing: '0',
        textTransform: 'none'
      }, true), local176('.func-buttons', {
        display: "grid",
        gridTemplateColumns: "repeat(2, 1fr)",
        gap: "6px"
      }), local176(".func-btn", local177, true), local176(".func-label", {
        display: "block",
        fontSize: "9px",
        fontWeight: "700",
        lineHeight: "1",
        color: "inherit",
        whiteSpace: "nowrap"
      }, true), local176(".func-icon", {
        display: "flex",
        alignItems: "center",
        justifyContent: "center"
      }, true), local176(".logs-panel", {
        padding: "8px 8px",
        background: "#f6f8fb",
        borderBottom: '1px\x20solid\x20#e5ebf2'
      }), local176(".logs-header", {
        display: "flex",
        alignItems: "center",
        justifyContent: 'space-between',
        marginBottom: "6px"
      }), local176(".clear-log-btn", {
        width: "24px",
        height: "24px",
        border: "1px solid #dbe5ef",
        borderRadius: '7px',
        background: "#ffffff",
        color: "#334155",
        display: 'flex',
        alignItems: "center",
        justifyContent: "center",
        gap: '0',
        padding: '0',
        fontSize: '10px',
        fontWeight: "800",
        cursor: "pointer"
      }), local176(".logs-container", {
        height: '96px',
        padding: "6px",
        border: "1px solid #dfe7ef",
        borderRadius: "8px",
        background: "#ffffff",
        boxSizing: "border-box",
        fontSize: "9px",
        lineHeight: "1.35"
      }), local176(".status-bar", {
        height: "22px",
        padding: "0 8px",
        background: "#ffffff",
        borderTop: '1px\x20solid\x20#e5ebf2',
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        boxSizing: "border-box",
        fontSize: "9px",
        color: "#64748b"
      }), local176(".config-panel", {
        position: "absolute",
        left: "8px",
        right: "8px",
        top: "52px",
        background: "#ffffff",
        border: "1px solid #d9e4ef",
        borderRadius: "8px",
        boxShadow: "0 12px 26px rgba(15, 23, 42, 0.18)",
        overflow: "hidden",
        zIndex: "10002"
      }), local176(".config-header", {
        height: "36px",
        padding: "0 8px",
        background: "#172033",
        borderBottom: "0",
        color: "#ffffff",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        boxSizing: "border-box"
      }), local176(".config-content", {
        padding: "9px",
        background: "#ffffff"
      }), local176(".config-group", {
        marginBottom: "9px"
      }, true), local176(".config-label", {
        display: "block",
        marginBottom: "5px",
        color: "#344054",
        fontSize: "10px",
        fontWeight: "800"
      }, true), local176(".range-inputs", {
        display: "grid",
        gridTemplateColumns: "1fr 18px 1fr",
        gap: "5px",
        alignItems: "center"
      }), local176(".config-input", {
        width: "100%",
        height: "28px",
        border: "1px solid #d9e4ef",
        borderRadius: "7px",
        padding: "0 6px",
        boxSizing: "border-box",
        color: "#172033",
        background: "#f8fafc",
        fontSize: "11px",
        outline: "none"
      }, true), local176(".range-separator", {
        color: "#667085",
        fontSize: "10px",
        textAlign: "center"
      }), local176(".config-hint", {
        marginTop: "4px",
        color: "#8792a2",
        fontSize: "9px",
        lineHeight: "1.25"
      }, true), local176(".config-actions", {
        display: "grid",
        gridTemplateColumns: "1fr 1fr",
        gap: "6px",
        marginTop: "10px"
      }), local176(".config-actions .config-btn", {
        height: "30px",
        border: "0",
        borderRadius: "7px",
        color: "#ffffff",
        fontSize: "10px",
        fontWeight: "800",
        cursor: "pointer"
      }, true), local176(".save-btn", {
        background: "#159a52"
      }), local176(".reset-btn", {
        background: "#475569"
      }), local176(".close-config-btn", {
        width: "22px",
        height: "22px",
        border: "0",
        borderRadius: "6px",
        background: "rgba(255,255,255,0.11)",
        color: "#ffffff",
        cursor: "pointer",
        padding: "0",
        lineHeight: "1"
      }), local176('.resize-controls', {
        display: "none"
      }), 隐藏元素.querySelectorAll("svg").forEach(param63 => {
        param63.setAttribute('fill', 'none'), param63.setAttribute("stroke", 'currentColor'), param63.setAttribute('stroke-width', '2'), param63.setAttribute("stroke-linecap", 'round'), param63.setAttribute('stroke-linejoin', "round"), param63.style.width = "16px", param63.style.height = '16px', param63.style.display = "block";
      });
      const localUiStyle = document.createElement("style");
      localUiStyle.textContent = `
        #dashboardContainer.dashboard-container {
          width: 156px !important;
          height: 346px !important;
          min-width: 156px !important;
          min-height: 346px !important;
          max-width: 156px !important;
          max-height: 346px !important;
          display: flex !important;
          flex-direction: column !important;
          background: #f3f7f6 !important;
          border: 1px solid #cddbd8 !important;
          border-radius: 12px !important;
          box-shadow: 0 14px 34px rgba(17, 24, 39, 0.22) !important;
          overflow: hidden !important;
          color: #172033 !important;
        }

        #dashboardHeader.dashboard-header {
          height: 42px !important;
          padding: 7px !important;
          background: #ffffff !important;
          border-bottom: 1px solid #dce7e4 !important;
          cursor: move !important;
        }

        #dashboardContainer .header-left {
          gap: 6px !important;
          min-width: 0 !important;
        }

        #dashboardContainer .brand-mark {
          width: 24px !important;
          height: 24px !important;
          border-radius: 7px !important;
          background: #111827 !important;
          color: #ffffff !important;
          box-shadow: 0 5px 12px rgba(17, 24, 39, 0.22) !important;
          font-size: 14px !important;
          overflow: hidden !important;
          display: flex !important;
          align-items: center !important;
          justify-content: center !important;
          padding: 0 !important;
        }

        #dashboardContainer .brand-mark img {
          width: 42px !important;
          height: 42px !important;
          display: block !important;
          object-fit: cover !important;
          transform: translateY(-3px) !important;
          pointer-events: none !important;
        }

        #dashboardContainer .header-title {
          color: #15202b !important;
          font-size: 11px !important;
          line-height: 1.1 !important;
          font-weight: 800 !important;
        }

        #dashboardContainer .header-region {
          color: #667085 !important;
          font-size: 9px !important;
          line-height: 1.2 !important;
          background: transparent !important;
          border: 0 !important;
          padding: 0 !important;
        }

        #dashboardContainer .header-controls {
          gap: 4px !important;
        }

        #dashboardContainer .window-btn {
          width: 20px !important;
          height: 20px !important;
          border-radius: 6px !important;
          background: #eef3f2 !important;
          color: #344054 !important;
          border: 1px solid #dce7e4 !important;
        }

        #dashboardContainer .window-btn:hover {
          background: #dfe9e6 !important;
        }

        #dashboardContainer .close-btn:hover {
          background: #ef4444 !important;
          border-color: #ef4444 !important;
          color: #ffffff !important;
        }

        #dashboardContainer .control-panel {
          padding: 7px 7px 6px !important;
          background: #f3f7f6 !important;
          border-bottom: 0 !important;
        }

        #dashboardContainer .control-row {
          height: 50px !important;
          display: grid !important;
          grid-template-columns: 1fr 44px !important;
          gap: 6px !important;
          align-items: stretch !important;
        }

        #toggleTaskBtn.toggle-btn {
          height: 50px !important;
          border: 0 !important;
          border-radius: 9px !important;
          background: #0f9f6e !important;
          color: #ffffff !important;
          box-shadow: 0 9px 18px rgba(15, 159, 110, 0.24) !important;
          display: flex !important;
          flex-direction: column !important;
          gap: 3px !important;
          align-items: center !important;
          justify-content: center !important;
        }

        #toggleTaskBtn.toggle-btn.running {
          background: #dc2626 !important;
          box-shadow: 0 9px 18px rgba(220, 38, 38, 0.22) !important;
        }

        #dashboardContainer .btn-icon svg {
          width: 16px !important;
          height: 16px !important;
        }

        #dashboardContainer .btn-text {
          display: block !important;
          font-size: 10px !important;
          font-weight: 800 !important;
        }

        #dashboardContainer .task-count-display {
          height: 50px !important;
          border-radius: 9px !important;
          border: 1px solid #dbe8e4 !important;
          background: #ffffff !important;
          box-shadow: 0 8px 18px rgba(17, 24, 39, 0.06) !important;
        }

        #dashboardContainer .task-count-caption {
          color: #667085 !important;
          font-size: 9px !important;
          font-weight: 700 !important;
        }

        #dashboardContainer .task-count-value {
          color: #101828 !important;
          font-size: 18px !important;
          font-weight: 900 !important;
          line-height: 1 !important;
        }

        #dashboardContainer .functions-panel {
          padding: 0 7px 7px !important;
          background: #f3f7f6 !important;
          border-bottom: 0 !important;
        }

        #dashboardContainer .panel-title {
          color: #475467 !important;
          font-size: 10px !important;
          font-weight: 800 !important;
          margin-bottom: 5px !important;
          letter-spacing: 0 !important;
          text-transform: none !important;
        }

        #dashboardContainer .func-buttons {
          display: grid !important;
          grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
          gap: 4px !important;
        }

        #dashboardContainer .func-btn {
          height: 31px !important;
          border-radius: 7px !important;
          border: 1px solid #dbe8e4 !important;
          background: #ffffff !important;
          color: #27374a !important;
          box-shadow: 0 7px 14px rgba(17, 24, 39, 0.05) !important;
          display: flex !important;
          flex-direction: column !important;
          gap: 3px !important;
          flex-direction: row !important;
        }

        #dashboardContainer .func-btn:hover {
          background: #ecfdf5 !important;
          border-color: #a7dbc8 !important;
          color: #0f7e59 !important;
        }

        #dashboardContainer .func-icon svg {
          width: 13px !important;
          height: 13px !important;
        }

        #dashboardContainer .func-label {
          display: block !important;
          color: inherit !important;
          font-size: 9px !important;
          line-height: 1 !important;
          font-weight: 800 !important;
          white-space: nowrap !important;
        }

        #dashboardContainer .logs-panel {
          flex: 1 1 auto !important;
          min-height: 0 !important;
          padding: 0 7px 7px !important;
          background: #f3f7f6 !important;
          border-bottom: 0 !important;
          display: flex !important;
          flex-direction: column !important;
        }

        #dashboardContainer .logs-header {
          margin-bottom: 5px !important;
        }

        #dashboardContainer .clear-log-btn {
          width: 20px !important;
          height: 20px !important;
          border-radius: 7px !important;
          background: #ffffff !important;
          border: 1px solid #dbe8e4 !important;
          color: #475467 !important;
        }

        #dashboardContainer .logs-container {
          flex: 1 1 auto !important;
          height: auto !important;
          min-height: 76px !important;
          padding: 5px !important;
          border-radius: 9px !important;
          border: 1px solid #dbe8e4 !important;
          background: #ffffff !important;
          font-size: 10px !important;
          line-height: 1.45 !important;
          box-shadow: inset 0 1px 0 rgba(17, 24, 39, 0.03) !important;
        }

        #dashboardContainer .log-entry {
          margin-bottom: 5px !important;
          padding: 5px !important;
          border-left: 3px solid #0f9f6e !important;
          border-radius: 7px !important;
          background: #f8fbfa !important;
        }

        #dashboardContainer .log-time {
          display: block !important;
          margin: 0 0 3px 0 !important;
          color: #98a2b3 !important;
          font-size: 9px !important;
        }

        #dashboardContainer .log-message,
        #dashboardContainer .small-text {
          color: #27374a !important;
          font-size: 10px !important;
          line-height: 1.45 !important;
        }

        #dashboardContainer .status-bar {
          flex: 0 0 30px !important;
          height: 30px !important;
          min-height: 30px !important;
          padding: 0 7px !important;
          background: #ffffff !important;
          border-top: 1px solid #dce7e4 !important;
          display: flex !important;
          align-items: center !important;
          justify-content: space-between !important;
          color: #475467 !important;
          font-size: 10px !important;
          box-sizing: border-box !important;
        }

        #statusText.status-text {
          display: block !important;
          max-width: 68px !important;
          min-width: 0 !important;
          overflow: hidden !important;
          text-overflow: ellipsis !important;
          white-space: nowrap !important;
          color: #344054 !important;
          font-weight: 700 !important;
        }

        #currentTime.status-time {
          color: #667085 !important;
          font-size: 9px !important;
          flex: 0 0 auto !important;
        }

        #dashboardContainer .config-panel {
          left: 7px !important;
          right: 7px !important;
          top: 50px !important;
          border-radius: 10px !important;
          border: 1px solid #dbe8e4 !important;
          box-shadow: 0 18px 38px rgba(17, 24, 39, 0.24) !important;
        }

        #dashboardContainer .config-header {
          height: 38px !important;
          padding: 0 10px !important;
          background: #172033 !important;
        }

        #dashboardContainer .config-content {
          padding: 10px !important;
        }

        #dashboardContainer.dashboard-container.minimized {
          width: 154px !important;
          height: 44px !important;
          min-width: 154px !important;
          min-height: 44px !important;
          max-width: 154px !important;
          max-height: 44px !important;
          border-radius: 12px !important;
        }

        #dashboardContainer.dashboard-container.minimized #dashboardHeader {
          height: 44px !important;
          background: #ffffff !important;
          border-bottom: 0 !important;
        }
      `;
      隐藏元素.appendChild(localUiStyle);
      const local178 = param64 => {
        for (let local197 of param64) {
          local197.ID && 隐藏元素.getElementById(local197.ID) && 隐藏元素.getElementById(local197.ID).addEventListener("click", local197.函数);
        }
      };
      local178(按钮数组), 隐藏元素.getElementById(local171.ID).addEventListener("click", local171.函数), 隐藏元素.getElementById("clearLogBtn").addEventListener('click', param64 => {
        param64.preventDefault();
        param64.stopPropagation();
        slog.清空当前日志();
      }), 隐藏元素.getElementById("closeConfigBtn").addEventListener('click', () => {
        隐藏元素.getElementById("configPanel").style.display = 'none';
      }), 隐藏元素.getElementById('saveConfigBtn').addEventListener("click", () => {
        Xx框架.保存配置();
      }), 隐藏元素.getElementById("resetConfigBtn").addEventListener("click", () => {
        Xx框架.重置配置();
      });
      const local179 = 隐藏元素.querySelector(".header-left"),
        local180 = document.createElement("div");
      local180.className = 'task-count', local180.innerHTML = "<span class=\"task-count-label\">任务:</span> <span class=\"task-count-value\" id=\"headerTaskCount\">0</span>", local179.appendChild(local180);
      const local181 = 隐藏元素.getElementById("minimizeBtn"),
        local182 = 隐藏元素.getElementById('closeBtn'),
        local183 = 隐藏元素.getElementById("dashboardContainer"),
        local184 = 隐藏元素.getElementById('dashboardHeader');
      let local185 = false;
      function fn2() {
        任务状态管理器.是否运行中() ? 停止任务() : 开始任务(), Xx框架.更新任务状态显示();
      }
      const local186 = 隐藏元素.getElementById("toggleTaskBtn");
      local186 && local186.addEventListener("click", fn2);
      local181.addEventListener("click", param65 => {
        param65.stopPropagation(), local185 = !local185, local183.classList.toggle("minimized");
        if (local185) {
          local181.innerHTML = local169.还原;
          const local198 = 任务计数 ? 任务计数.get() : 0;
          隐藏元素.getElementById("headerTaskCount").textContent = local198;
        } else local181.innerHTML = local169.最小化, local183.style.width = '', local183.style.height = '', local183.style.minWidth = '', local183.style.minHeight = '', local183.style.maxWidth = '', local183.style.maxHeight = '';
      }), local184.addEventListener("click", param66 => {
        local185 && param66.target.closest(".minimize-btn") === null && (local185 = false, local183.classList.remove('minimized'), local181.innerHTML = local169.最小化, local183.style.width = '', local183.style.height = '', local183.style.minWidth = '', local183.style.minHeight = '', local183.style.maxWidth = '', local183.style.maxHeight = '');
      }), local182.addEventListener("click", () => {
        local183.style.display = 'none';
      });
      let local187 = false,
        local188 = {
          x: 0,
          y: 0
        };
      local184.addEventListener("mousedown", param67 => {
        if (param67.target.closest(".minimize-btn") || param67.target.closest(".close-btn")) return;
        local187 = true;
        const local199 = local183.getBoundingClientRect();
        local188.x = param67.clientX - local199.left, local188.y = param67.clientY - local199.top, param67.preventDefault();
      }), document.addEventListener("mousemove", param68 => {
        local187 && (local183.style.left = param68.clientX - local188.x + 'px', local183.style.top = param68.clientY - local188.y + 'px', local183.style.right = "auto");
      }), document.addEventListener("mouseup", () => {
        local187 = false;
      });
      let local189 = false;
      const local190 = 隐藏元素.getElementById("resizeHandle");
      local190.addEventListener("mousedown", param69 => {
        local189 = true, param69.preventDefault();
      }), document.addEventListener('mousemove', param70 => {
        if (local189) {
          const local200 = local183.getBoundingClientRect(),
            local201 = param70.clientX - local200.left,
            local202 = param70.clientY - local200.top;
          local183.style.width = Math.max(250, Math.min(600, local201)) + 'px', local183.style.height = Math.max(400, Math.min(700, local202)) + 'px';
        }
      }), document.addEventListener("mouseup", () => {
        local189 = false;
      });
      for (let local203 of 数据数组) {
        if (local203.name === "任务计数" || local203.name === "版本号") {
          let local204 = this.get计数器_框架版(local203.name, local203.ID);
          local204.get(), window[local203.name] = local204;
          if (local203.name === '任务计数') {
            const local205 = 隐藏元素.getElementById("topTaskCount");
            if (local205) {
              const local206 = local204.get();
              local205.textContent = local206;
            }
          }
        }
      }
      this.更新任务状态显示();
      function fn3() {
        const local207 = 隐藏元素.getElementById("currentTime");
        if (local207) {
          const local208 = new Date();
          local207.textContent = local208.toLocaleTimeString("zh-CN", {
            hour12: false,
            hour: '2-digit',
            minute: "2-digit",
            second: '2-digit'
          });
        }
      }
      fn3(), setInterval(fn3, 1000);
      const local191 = slog.log;
      slog.log = function (param71, param72) {
        local191.call(this, param71, param72);
        param71 = String(param71 === undefined || param71 === null ? '' : param71);
        if (param71.includes('间隔')) {
          const local213 = 隐藏元素.getElementById("statusText");
          local213 && (local213.textContent = param71);
        }
        if (param71.includes('任务')) {
          const local214 = 隐藏元素.getElementById(数据数组.find(param73 => param73.name === "任务计数")?.['ID']);
          local214 && (local214.classList.add("pulse-animation"), setTimeout(() => local214.classList.remove("pulse-animation"), 500));
          const local215 = 隐藏元素.getElementById("topTaskCount");
          if (local215 && 任务计数) {
            const local216 = 任务计数.get();
            local215.textContent = local216, local215.classList.add("pulse-animation"), setTimeout(() => local215.classList.remove("pulse-animation"), 500);
          }
          if (local185) {
            const local217 = 任务计数 ? 任务计数.get() : 0;
            隐藏元素.getElementById("headerTaskCount").textContent = local217;
          }
        }
      }, slog.打开页面初始化();
    }
    static ["更新任务状态显示"]() {
      if (!隐藏元素) return;
      const local218 = 任务状态管理器.是否运行中(),
        local219 = 隐藏元素.getElementById('toggleTaskBtn'),
        local220 = 隐藏元素.getElementById("toggleLabel"),
        local221 = 隐藏元素.getElementById("toggleIcon");
      if (local218) {
        local219 && (local219.classList.remove('stopped'), local219.classList.add('running'), local219.style.background = "#dc2626", local219.style.boxShadow = "0 6px 14px rgba(220, 38, 38, 0.22)");
        if (local220) local220.textContent = '停止';
        if (local221) local221.innerHTML = 停止按钮图标;
      } else {
        local219 && (local219.classList.remove("running"), local219.classList.add("stopped"), local219.style.background = "#16a34a", local219.style.boxShadow = '0\x206px\x2014px\x20rgba(22,\x20163,\x2074,\x200.24)');
        if (local220) local220.textContent = '开始';
        if (local221) local221.innerHTML = 开始按钮图标;
      }
    }
    static ['显示配置面板']() {
      const local222 = 隐藏元素.getElementById('configPanel');
      if (local222) {
        const local223 = 任务管理器.加载配置(),
          local224 = 隐藏元素.getElementById("minInterval"),
          local225 = 隐藏元素.getElementById("maxInterval"),
          local226 = 隐藏元素.getElementById("batchCount");
        if (local224) local224.value = local223.任务间隔最小;
        if (local225) local225.value = local223.任务间隔最大;
        if (local226) local226.value = local223.批次任务数;
        local222.style.display = "block";
      }
    }
    static ["保存配置"]() {
      const local227 = 规范化间隔(隐藏元素.getElementById("minInterval").value),
        local228 = 规范化间隔(隐藏元素.getElementById('maxInterval').value),
        local229 = parseInt(隐藏元素.getElementById("batchCount").value) || 100;
      if (!(local227 > 0) || !(local228 > 0)) {
        slog.mlog('错误:\x20间隔时间必须大于0');
        return;
      }
      if (local227 > local228) {
        slog.mlog('错误:\x20最小间隔不能大于最大间隔');
        return;
      }
      if (local229 < 1) {
        slog.mlog("错误: 批次任务数必须大于0");
        return;
      }
      任务管理器.设置间隔(local227, local228), 任务管理器.设置批次任务数(local229), 隐藏元素.getElementById("configPanel").style.display = "none", slog.mlog('配置已保存:\x20间隔' + local227 + '-' + local228 + '秒,\x20批次' + local229 + "个任务");
    }
    static ['重置配置']() {
      const local230 = 任务管理器.重置配置();
      隐藏元素.getElementById("minInterval").value = local230.任务间隔最小, 隐藏元素.getElementById("maxInterval").value = local230.任务间隔最大, 隐藏元素.getElementById("batchCount").value = local230.批次任务数, 任务接口模式 = "old", localStorage.setItem("任务接口模式", "old"), localStorage.setItem('地区', '台湾'), 分标识ClientId = 固定分标识ClientId, slog.mlog('配置已恢复为默认值');
    }
    static ["get计数器_框架版"](param74, param75) {
      return {
        show: function (param76) {
          !param76 && (param76 = 0);
          const local231 = 隐藏元素.getElementById(param75);
          local231 && (local231.innerText = param76);
        },
        add: function () {
          var local232 = localStorage.getItem(param74);
          local232 = local232 ? parseInt(local232) : 0, localStorage.setItem(param74, local232 + 1), this.show(local232 + 1);
        },
        删除: function () {
          localStorage.removeItem(param74), this.show(0);
        },
        重置: function () {
          localStorage.setItem(param74, 0), this.show(0);
        },
        减1: function () {
          var local233 = localStorage.getItem(param74);
          if (local233 && local233 > 0) {
            localStorage.setItem(param74, local233 - 1), this.show(local233 - 1);
            return;
          }
          this.show(local233);
        },
        get: function () {
          var local234 = localStorage.getItem(param74);
          return local234 = local234 ? parseInt(local234) : 0, this.show(local234), local234;
        }
      };
    }
  },
  Xxlib = class {
    static ['getDate'](param77 = "y-m-d h:i:s", param78) {
      const local235 = param78 ? new Date(param78) : new Date(),
        local236 = [local235.getFullYear(), local235.getMonth() + 1, local235.getDate(), local235.getHours(), local235.getMinutes(), local235.getSeconds()],
        local237 = "ymdhis";
      return local236.forEach((param79, param80) => {
        param77 = param77.replace(local237.charAt(param80), param79.toString());
      }), param77;
    }
    static ["随机签名"](param81 = 32) {
      const local238 = "abcdef0123456789",
        local239 = local238.length;
      let local240 = '';
      for (let local241 = 0; local241 < param81; local241++) {
        local240 += local238.charAt(Math.floor(Math.random() * local239));
      }
      return local240;
    }
    static ["随机2位后缀"]() {
      return Math.floor(Math.random() * 90) + 10;
    }
  },
  XxQianDuan = class {
    static ["创建牌子"](param82, param83) {
      const local242 = document.createElement("div");
      local242.style.position = 'fixed', local242.style.bottom = '0', local242.style.left = '0', local242.style.width = '50px', local242.style.height = param83 + 'px', local242.style.zIndex = '9999', local242.style.backgroundColor = param82, document.body.appendChild(local242);
    }
    static ["亮牌子"](param84) {
      this.创建牌子(param84, 70);
    }
    static ["设置背景颜色"](param85, param86) {
      const local243 = document.getElementById(param85);
      local243 && (local243.style.backgroundColor = param86);
    }
    static ["removeItem_info"](param87) {
      return localStorage.removeItem(param87);
    }
    static ["为页面注入代码"](param88) {
      try {
        const local244 = document.createElement("script"),
          local245 = chrome.runtime.getURL("injected.js");
        local244.src = local245, local244.onload = function () {
          this.parentNode && this.parentNode.removeChild(this), console.log('代码注入成功');
        }, local244.onerror = function () {
          console.error('代码注入失败:\x20无法加载\x20injected.js\x20文件'), this.parentNode && this.parentNode.removeChild(this);
        };
        const local246 = document.head || document.documentElement;
        if (local246) local246.appendChild(local244);else {
          const local247 = setInterval(() => {
            const local248 = document.head || document.documentElement;
            local248 && (clearInterval(local247), local248.appendChild(local244));
          }, 50);
          setTimeout(() => clearInterval(local247), 5000);
        }
      } catch (local249) {
        console.error("代码注入异常:", local249);
      }
    }
  };
function 跳转(param89) {
  location.href = param89;
}
function 注入Fetch拦截代码() {
  document.head || document.documentElement ? XxQianDuan.为页面注入代码('') : setTimeout(注入Fetch拦截代码, 50);
}
注入Fetch拦截代码();
任务状态管理器.恢复状态();
任务看门狗恢复();
启动任务监督();
