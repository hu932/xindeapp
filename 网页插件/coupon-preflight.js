const 预检状态键 = '虾皮任务助手_优惠券预检状态';
const 预检ProfileUrl = 'https://shopee.tw/api/v2/user/profile/get/';
let 预检当前Profile = null;
let 预检上报中 = false;

function 预检读取JSON(key, fallback) {
  try {
    const value = JSON.parse(localStorage.getItem(key));
    return value === null || value === undefined ? fallback : value;
  } catch (_) {
    return fallback;
  }
}

function 预检保存JSON(key, value) {
  localStorage.setItem(key, JSON.stringify(value));
}

function 预检上海日期() {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).formatToParts(new Date());
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${values.year}-${values.month}-${values.day}`;
}

function 获取优惠券预检配置() {
  return typeof 优惠券预检配置 !== 'undefined' && 优惠券预检配置 ? 优惠券预检配置 : null;
}

function 预检接口地址(path) {
  const config = 获取优惠券预检配置();
  if (!config || !config.serverBaseUrl) return '';
  return `${String(config.serverBaseUrl).replace(/\/+$/, '')}${path}`;
}

async function 预检请求(path, body) {
  const config = 获取优惠券预检配置();
  if (!config || !config.token) throw new Error('优惠券预检配置缺失');
  const response = await XxHttp.Post(预检接口地址(path), body, {
    Authorization: `Bearer ${config.token}`,
    Accept: 'application/json',
    'Content-Type': 'application/json'
  });
  if (!response || !response.success) {
    throw new Error(response && response.error || '优惠券预检请求失败');
  }
  const payload = response.data;
  if (payload && payload.success === false) {
    throw new Error(payload.error || payload.message || '优惠券预检接口失败');
  }
  return payload;
}

async function 预检读取Shopee账户() {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10000);
  try {
    const response = await fetch(预检ProfileUrl, {
      credentials: 'include',
      cache: 'no-store',
      signal: controller.signal
    });
    clearTimeout(timeoutId);
    if (!response.ok) throw new Error(`Shopee 登录态检查失败: HTTP ${response.status}`);
    const payload = await response.json();
    const userId = payload && payload.data && payload.data.userid;
    if (payload.error !== 0 || userId === null || userId === undefined) {
      throw new Error('Shopee 登录态已失效，请重新登录');
    }
    return {
      userId: String(userId),
      username: String(payload.data.username || '')
    };
  } catch (error) {
    clearTimeout(timeoutId);
    if (error && error.name === 'AbortError') {
      throw new Error('Shopee 登录态检查超时');
    }
    throw error;
  }
}

async function 预检执行(profile) {
  const response = await 预检请求('/api/coupon-preflight/start/', {
    shopeeUserId: profile.userId,
    shopeeUsername: profile.username
  });
  if (!任务状态管理器.是否运行中()) return false;
  const data = response && response.data || {};
  if (data.allowed || !data.required) {
    预检保存JSON(预检状态键, {
      userId: profile.userId,
      dayKey: data.dayKey || 预检上海日期()
    });
    return true;
  }
  const probe = data.probe;
  if (!probe || !probe.id || !probe.url) throw new Error('服务器未返回优惠券检测商品');
  localStorage.setItem('储存_获取任务的数据', JSON.stringify({
    taskId: `coupon-probe-${probe.id}`,
    taskUrl: probe.url,
    shopId: probe.shopId,
    itemId: probe.itemId,
    couponProbe: true,
    probeId: probe.id,
    shopeeProfile: profile
  }));
  submitTaskUrl = '';
  slog.mlog('开始优惠券数据检测');
  启动任务看门狗();
  Xx项目_跳转.处理跳转(probe.url);
  return false;
}

async function 预检确保() {
  const config = 获取优惠券预检配置();
  if (!config) return true;
  const dayKey = 预检上海日期();
  const state = 预检读取JSON(预检状态键, null);
  // 今天已通过检测：直接放行，不再请求 Shopee profile 接口，避免二次请求卡住任务
  if (state && state.userId && state.dayKey === dayKey) return true;
  const profile = await 预检读取Shopee账户();
  if (!任务状态管理器.是否运行中()) return false;
  const previousUserId = 预检当前Profile && 预检当前Profile.userId;
  预检当前Profile = profile;
  if (previousUserId && previousUserId !== profile.userId) {
    localStorage.removeItem(预检状态键);
    slog.mlog('检测到 Shopee 账户切换，重新检测');
  }
  const freshState = 预检读取JSON(预检状态键, null);
  if (freshState && freshState.userId === profile.userId && freshState.dayKey === dayKey) return true;
  return 预检执行(profile);
}

async function 预检提交(task, result) {
  const profile = task.shopeeProfile || await 预检读取Shopee账户();
  const response = await 预检请求('/api/coupon-preflight/report/', {
    shopeeUserId: profile.userId,
    shopeeUsername: profile.username,
    probeId: task.probeId,
    status: 'captured',
    rawResponseText: JSON.stringify(result)
  });
  const data = response && response.data || {};
  if (!data.passed) throw new Error(data.reason || '优惠券数据检测未通过');
  预检保存JSON(预检状态键, {
    userId: profile.userId,
    dayKey: data.dayKey || 预检上海日期()
  });
  预检当前Profile = profile;
}

function 预检读取当前任务() {
  const raw = localStorage.getItem('储存_获取任务的数据');
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch (_) {
    return null;
  }
}

async function 预检处理上传(result) {
  if (预检上报中) return false;
  const task = 预检读取当前任务();
  if (!task || !task.couponProbe) return false;
  预检上报中 = true;
  try {
    if (typeof result === 'string') {
      try {
        result = JSON.parse(result);
      } catch (_) {
        return false;
      }
    }
    await 预检提交(task, result);
    if (!任务状态管理器.是否运行中()) return true;
    XxQianDuan.removeItem_info('储存_获取任务的数据');
    submitTaskUrl = '';
    slog.mlog('优惠券数据检测通过');
    setTimeout(申请任务, 300);
    return true;
  } catch (error) {
    slog.mlog(`优惠券数据检测失败: ${error.message || error}`);
    XxQianDuan.亮牌子('yellow');
    任务状态管理器.停止();
    Xx框架.更新任务状态显示();
    return true;
  } finally {
    预检上报中 = false;
  }
}

const 预检原始申请任务 = 申请任务;
申请任务 = async function () {
  const config = 获取优惠券预检配置();
  if (config) {
    try {
      if (!await 预检确保()) return;
    } catch (error) {
      slog.mlog(`优惠券预检失败: ${error.message || error}`);
      XxQianDuan.亮牌子('yellow');
      任务状态管理器.停止();
      Xx框架.更新任务状态显示();
      return;
    }
  }
  return 预检原始申请任务();
};

const 预检原始上传商品 = 上传商品;
上传商品 = async function (result) {
  if (获取优惠券预检配置() && 预检读取当前任务() && 预检读取当前任务().couponProbe) {
    return 预检处理上传(result);
  }
  return 预检原始上传商品(result);
};
