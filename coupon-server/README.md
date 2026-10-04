# 优惠券预检服务器（PHP 版）

这个目录是一个自托管的「优惠券预检」服务端，用来替代插件里原本指向
`http://122.51.109.235:8080` 的第三方服务器。

## 文件结构

```
coupon-server/
├── config.php                      # 配置（token、后台账号密码、探测商品默认值、校验规则）
├── lib.php                         # 公共逻辑（鉴权、存储、两个接口 + 后台辅助）
├── index.php                       # 根目录跳转到后台登录页
├── .htaccess                       # 禁止直接访问 config.php / lib.php
├── admin/
│   └── index.php                   # 管理后台（登录 + 商品管理 + 记录查看）
├── storage/                        # 数据目录（需 PHP 可写，探测商品也存这里）
│   └── .htaccess                   # 禁止浏览器访问数据文件
└── api/
    └── coupon-preflight/
        ├── start/index.php         # POST /api/coupon-preflight/start
        └── report/index.php        # POST /api/coupon-preflight/report
```

接口路径直接对应插件 `coupon-preflight.js` 里调用的地址，所以目录结构不能乱改。

## 部署步骤

1. 把整个 `coupon-server` 目录上传到你的 PHP 主机（Apache/cPanel 共享主机、
   Nginx + PHP-FPM 均可，要求 PHP 7.0+）。
2. 确保 `storage` 目录可写（权限 755 或 775，必要时 777）。
3. 修改 `config.php`：
   - `auth_token`：保持和插件 `init.js` 里 `优惠券预检配置.token` 一致（默认已一致）。
   - `admin_user` / `admin_password`：**务必改成你自己的后台账号密码**。
   - `probe_products`：只是首次使用的默认数据，之后在后台里在线管理即可。
4. 访问 `https://你的域名/coupon-server/`，会自动跳到后台登录页。

> 如果使用 Nginx，默认不会像 Apache 那样自动把
> `/api/coupon-preflight/start` 映射到 `.../start/index.php`，
> 需要自行加 `try_files $uri $uri/ /index.php?$query_string;` 之类的规则，
> 或改用带 `index.php` 的完整路径，并同步修改插件里的请求路径。

## 管理后台

- 地址：`https://你的域名/coupon-server/admin/`（或访问根目录自动跳转）。
- 功能：
  - 在线增删改「探测商品」（存到 `storage/probe_products.json`，无需再改 `config.php`）。
  - 查看「已通过检测的账号」记录，可一键清空（清空后当天会重新检测）。
  - 查看「待处理检测任务」，可一键清空。

## 接口说明

统一返回信封：

```json
{ "success": true, "error": null, "data": { ... } }
```

### POST /api/coupon-preflight/start

请求头：`Authorization: Bearer <token>`

```json
{ "shopeeUserId": "123456", "shopeeUsername": "xxx" }
```

返回 `data`：

```json
{
  "allowed": false,
  "required": true,
  "dayKey": "2026-09-17",
  "probe": { "id": "probe-001", "url": "https://shopee.tw/...", "shopId": "0", "itemId": "0" }
}
```

- `allowed=true` / `required=false`：该账号今天已通过，直接放行。
- `required=true`：需要检测，插件会跳转到 `probe.url` 采集优惠券数据。

### POST /api/coupon-preflight/report

```json
{
  "shopeeUserId": "123456",
  "shopeeUsername": "xxx",
  "probeId": "probe-001",
  "status": "captured",
  "rawResponseText": "{...采集到的数据...}"
}
```

返回 `data`：

```json
{ "passed": true, "dayKey": "2026-09-17" }
```

`passed=false` 时会带 `reason`，插件会停止任务并提示原因。

## 本地 / 服务器测试

```bash
# start
curl -X POST "http://你的域名/coupon-server/api/coupon-preflight/start" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer -CN6_PNYhHpDzN-qJw-RGecT5vOM1nY8zqORenKnFnw" \
  -d '{"shopeeUserId":"123456","shopeeUsername":"test"}'

# report（probeId 要填 start 返回的 probe.id）
curl -X POST "http://你的域名/coupon-server/api/coupon-preflight/report" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer -CN6_PNYhHpDzN-qJw-RGecT5vOM1nY8zqORenKnFnw" \
  -d '{"shopeeUserId":"123456","shopeeUsername":"test","probeId":"probe-001","status":"captured","rawResponseText":"{\"demo\":\"coupon data captured\"}"}'
```

## 如何把插件切到你自己的服务器

改插件 `init.js` 里这一行即可：

```js
const 优惠券预检配置 = Object.freeze({
  serverBaseUrl: "http://122.51.109.235:8080",   // 改成你的地址，例如 https://你的域名/coupon-server
  accountId: "5064336969",                        // 服务端目前未使用，可保留
  token: "-CN6_PNYhHpDzN-qJw-RGecT5vOM1nY8zqORenKnFnw"   // 与 config.php 的 auth_token 保持一致
});
```

同时记得改 `manifest.json` 的 `host_permissions` 里那条
`"http://122.51.109.235:8080/*"`，改成你的服务器域名，否则请求会被拦截。

## 说明

- 本实现是「每天每账号通过一次检测后当天放行」，检测通过记录存在 `storage/` 下。
- 服务器只是做「发放探测商品 + 接收上报 + 判定通过」的流程闸门，
  真正的数据采集由插件在 Shopee 页面完成。
- `report` 的通过判定默认只校验「非空 + 长度 + 可选 JSON」，
  如需更严格的校验（例如检查 coupon 字段），可在 `lib.php` 的
  `handle_coupon_report()` 里自行补充。
