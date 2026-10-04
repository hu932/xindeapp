const originalFetch = window.fetch;

window.fetch = async function(...args) {
    var url = '';
    if (typeof args[0] === 'string') {
        url = args[0];
    }
    else if (args[0] && args[0].url) {
        url = args[0].url;
    }
    const 是否目标请求 = url.includes('get_pc') || url.includes('get_rw');
    if (!是否目标请求) {
        return originalFetch.apply(this, arguments);
    }
    const response = await originalFetch.apply(this, arguments);
    try {
        if (!response || !response.body || response.bodyUsed) return response;
        const [hijackStream, originalStream] = response.body.tee();
        const hijackedResponse = new Response(hijackStream, {
            status: response.status, headers: response.headers,
        });
        let data = null;
        try {
            data = await hijackedResponse.clone().json();
        } catch (e) {
            try {
                data = JSON.parse(await hijackedResponse.text());
            } catch (e2) {
                data = null;
            }
        }
        if (data) {
            try { window.postMessage({ type: 'send', data: data }, '*'); } catch (e3) {}
        }
        return new Response(originalStream, {
            status: response.status, headers: response.headers,
        });
    } catch (e) {
        return response;
    }
};

// XHR 兜底拦截（只读透传，不影响页面本身请求）
(function () {
    const 原始Open = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function (method, url) {
        this.__ajie_url = typeof url === 'string' ? url : '';
        return 原始Open.apply(this, arguments);
    };
    const 原始Send = XMLHttpRequest.prototype.send;
    XMLHttpRequest.prototype.send = function () {
        const self = this;
        const u = self.__ajie_url || '';
        if (u.includes('get_pc') || u.includes('get_rw')) {
            self.addEventListener('readystatechange', function () {
                if (self.readyState === 4 && self.status === 200) {
                    try {
                        const data = JSON.parse(self.responseText);
                        window.postMessage({ type: 'send', data: data }, '*');
                    } catch (e) {}
                }
            });
        }
        return 原始Send.apply(this, arguments);
    };
})();

