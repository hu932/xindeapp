const messageHandlers = {};

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (!tab.url) {
    return;
  }

  chrome.declarativeNetRequest.updateEnabledRulesets({
    [tab.url.includes("verify") ? "disableRulesetIds" : "enableRulesetIds"]: ["rules"]
  });
});

function registerMessageHandler(act, handler) {
  messageHandlers[act] = handler;
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  const handler = msg && messageHandlers[msg.act];
  if (!handler) {
    sendResponse && sendResponse({ error: true, message: "未知的操作" });
    return false;
  }
  return handler(msg, sender, sendResponse);
});

registerMessageHandler("发送网络请求3_Post", (msg, sender, sendResponse) => {
  const { url, data, headers } = msg;
  XxBaHttp.Post(url, data, headers).then(result => sendResponse(result));
  return true;
});

registerMessageHandler("发送网络请求3_Get", (msg, sender, sendResponse) => {
  const { url, headers } = msg;
  XxBaHttp.Get(url, headers).then(result => sendResponse(result));
  return true;
});

registerMessageHandler("CK时间", (msg, sender, sendResponse) => {
  chrome.cookies.getAll({ name: "SPC_SC_OFFLINE_TOKEN" }, cookies => {
    if (!cookies || cookies.length < 1) {
      sendResponse("null");
      return;
    }

    const cookie = cookies[0];
    const now = Math.floor(Date.now() / 1000);
    const minRemainSeconds = 0x278d00;
    const remainSeconds = cookie.expirationDate ? cookie.expirationDate - now : -1;

    if (remainSeconds >= minRemainSeconds) {
      sendResponse("CK无需更新，剩余时间充足");
      return;
    }

    const nextExpiration = getEightMonthsLaterTimestamp();
    chrome.cookies.set({
      domain: ".xiapibuy.com",
      expirationDate: nextExpiration,
      httpOnly: true,
      name: "SPC_SC_OFFLINE_TOKEN",
      path: "/",
      sameSite: "unspecified",
      secure: false,
      storeId: "0",
      value: cookie.value,
      url: "https://xiapi.xiapibuy.com/"
    }, () => {
      if (chrome.runtime.lastError) {
        sendResponse("失败: " + chrome.runtime.lastError.message);
      } else {
        sendResponse("成功!!!");
      }
    });
  });
  return true;
});

registerMessageHandler("获取登录CK", (msg, sender, sendResponse) => {
  chrome.cookies.getAll({ name: "SPC_SC_OFFLINE_TOKEN" }, cookies => {
    sendResponse(cookies && cookies.length >= 1 ? cookies[0].value : "null");
  });
  return true;
});

registerMessageHandler("设置ck组", (msg, sender, sendResponse) => {
  const value = msg.cookie;
  if (typeof value !== "string") {
    return false;
  }

  chrome.cookies.set({
    domain: ".xiapibuy.com",
    expirationDate: getEightMonthsLaterTimestamp(),
    httpOnly: true,
    name: "SPC_SC_OFFLINE_TOKEN",
    path: "/",
    sameSite: "unspecified",
    secure: false,
    storeId: "0",
    value,
    url: "https://xiapi.xiapibuy.com/"
  }, () => {});
  return false;
});

function getEightMonthsLaterTimestamp() {
  const date = new Date();
  date.setMonth(date.getMonth() + 8);
  return Math.floor(date.getTime() / 1000);
}

class XxBaHttp {
  static async Get(url, headers = {}, timeout = 15000) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        method: "GET",
        headers,
        signal: controller.signal
      });
      return await XxBaHttp.parseResponse(response, timeoutId);
    } catch (error) {
      clearTimeout(timeoutId);
      return XxBaHttp.errorResponse(error);
    }
  }

  static async Post(url, data, headers = {}, timeout = 15000) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const requestHeaders = { "Content-Type": "application/json", ...headers };
      const response = await fetch(url, {
        method: "POST",
        headers: requestHeaders,
        body: data,
        signal: controller.signal
      });
      return await XxBaHttp.parseResponse(response, timeoutId);
    } catch (error) {
      clearTimeout(timeoutId);
      return XxBaHttp.errorResponse(error);
    }
  }

  static async parseResponse(response, timeoutId) {
    clearTimeout(timeoutId);

    const text = await response.text();
    let data = text;
    try {
      data = text ? JSON.parse(text) : null;
    } catch (error) {}

    if (!response.ok) {
      return {
        success: false,
        data,
        error: data && (data.msg || data.message || data.error) || ("HTTP error! Status: " + response.status)
      };
    }

    return { success: true, data, error: null };
  }

  static errorResponse(error) {
    return {
      success: false,
      data: null,
      error: error.name === "AbortError" ? "请求超时" : error.message || "出现未知错误"
    };
  }
}
