# Go 高性能重构版（Ajie 后台）

把原 PHP 后台（`decrypt_proxy.php` / `cx.php` / `cx1.php` / `cx_view.php` / `cx_links.php` / `accounts_share.php` / `load_monitor.php` / `cx_cron.php`）重写为 **Go 单二进制服务**，功能与数据格式完全兼容，可直接迁移旧数据。

## 为什么更快（解决“转发卡住 / 查询卡住”）

| 原 PHP 痛点 | Go 重构后 |
|---|---|
| PHP-FPM 进程池被上游阻塞占满 → 高峰期“卡死/超时” | Go goroutine 每请求一个协程，几乎不占资源；全局并发闸门用内存信号量，无文件锁争用 |
| 转发闸门用 `flock` + 每次读改写 JSON 文件 | 内存原子计数 + 信号量，零磁盘 IO |
| `cx.php` 用 `session_start()` 文件锁，多请求串行 | 内存缓存 + `sync.Mutex` 单飞锁，读缓存不阻塞 |
| 每个请求重复建连、DNS 解析 | 全局连接池 keep-alive + 静态 DNS 优选 IP + TLS 复用 |

## 目录结构

```
go-server/
  go.mod              模块定义（仅依赖 golang.org/x/crypto/bcrypt）
  main.go             入口 + 路由
  config.go           常量 / 上游账号 / 数据文件路径
  util.go crypto.go   工具 / AES-128-ECB / bcrypt / JWT
  gate.go             转发并发闸门（内存信号量）
  upstream.go         上游 HTTP 客户端 + token 缓存 + 静态 DNS
  logs.go devices.go subaccounts.go whitelist.go stats.go
  decodedext_*.go     插件账号 / 任务队列 / 租约 / /api/* 接口
  api_endpoints.go    api1/api2 login/take/submit + 子账号接口
  endpoints_post.go   POST 主分发
  upload.go           加密/明文数据上传转发（旧 App 主路径）
  admin_get.go pages.go  后台 GET 接口 + 页面
  cx_query.go cx_page.go cx_view.go cx_links.go   查询/备注链接
  accounts_share.go monitor.go updates.go dumps.go
```

## 构建

本机需安装 Go 1.21+。

```bash
cd go-server
go mod tidy        # 拉取 bcrypt 依赖（需要联网一次）
go build -o ajie-server .
```

跨平台给宝塔 Linux 服务器编译（Windows 上直接出 Linux 二进制）：

```powershell
cd go-server
$env:CGO_ENABLED=0
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -o ajie-server .
```

## 部署（宝塔）——数据 + 二进制一起打包

> 核心思路：**二进制和所有数据文件放同一个目录**。`go-server/deploy.sh` 会帮你在服务器上编译、把旧 PHP 目录的全部数据原样拷过来、生成 systemd 服务并启动。

### 方案 A（推荐）：整目录打包上传，服务器一键部署

**第一步（在你现在的 Windows 机器上）**：把整个 `后台111` 目录打包成一个 tar.gz 上传宝塔（包含 `go-server/` 源码 + 所有 PHP 数据文件）。

PowerShell 里执行（Windows 10+ 自带 tar）：

```powershell
cd "C:\Users\Administrator\Desktop"
tar -czf 后台111.tar.gz 后台111
```

宝塔面板 → 文件 → 上传 `后台111.tar.gz` → 解压到 `/www/wwwroot/`，得到 `/www/wwwroot/后台111/`。

**第二步（服务器上装 Go，只需一次）**：宝塔软件商店搜 `Go` 安装，或：

```bash
# 官方 tarball（amd64）
wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile
go version
```

**第三步（一键部署）**：

```bash
cd /www/wwwroot/后台111
bash go-server/deploy.sh
```

`deploy.sh` 自动完成：编译 Linux 二进制 → 拷贝到站点根目录 → 把 PHP 数据文件/子账号表等全部就位 → 写入 `/etc/systemd/system/ajie-server.service` → `systemctl enable --now` 启动。

### 方案 B：手动分步

1. 编译并上传 `ajie-server` 二进制到站点目录（如 `/www/wwwroot/后台111/`）：

```bash
cd go-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o ../ajie-server .
```

2. 数据文件与二进制**同目录**（把下面这些原样复制过去，保持相对路径）：

```
.auth_hash  .decoded_extension_accounts.json  .resolve_pref.json
.api1_removed_accounts.json  .api2_removed_accounts.json
api1_allowed_accounts.json  api2_allowed_accounts.json  api1_allowed_accounts_legacy.json
api1_token_map.json  api2_number_remarks.json
device_controls.json  cx_remark_links.json  cx_teamc_prices.json
sub_accounts.json  sub_account_usage.json  sub_account_bindings.json  sub_account_tokens.json
cx_cache/  fingerprint_env_stats/  daily_stats/  app_updates/  plugin_updates/
upload_dumps/  decrypt_dumps/  decoded_extension_runtime/
task_inventory.json  task_inventory_stats.json  task_inventory_sub_accounts.txt  task_inventory_tokens/
1777990327149137925.png  bj.jpeg
```

   密码文件 `.auth_hash` 直接沿用即可（bcrypt 兼容，无需改密码）。上面 `task_inventory_*` 是库存模式新增的，首次没有会自动生成。

3. 启动：

```bash
chmod +x ajie-server
./ajie-server -listen :2036 -data /www/wwwroot/后台111
```

   建议用 systemd 守护（`deploy.sh` 已自动写好）：

```ini
[Unit]
Description=Ajie Server
After=network.target

[Service]
WorkingDirectory=/www/wwwroot/后台111
ExecStart=/www/wwwroot/后台111/ajie-server -listen :2036
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now ajie-server
systemctl status ajie-server    # 看是否 running
journalctl -u ajie-server -f   # 看日志
```

4. 宝塔 nginx 反代（站点 → 反向代理 → 目标 `http://127.0.0.1:2036`），或直接把域名 80/443 反向代理到 2036。旧 PHP 入口全部指向该端口即可（`decrypt_proxy.php`、`cx.php`、`cx1.php`、`cx_view.php`、`cx_links.php`、`accounts_share.php`、`load_monitor.php`、`cx_cron.php` 等路径都走同一反代，Go 内部已按路径路由）。

5. 定时刷新：保留原 `cx_cron.php` 计划任务，改访问 `http://127.0.0.1:2036/cx_cron.php`。

6. 库存模式数据迁移：把子账号大表传到后台「库存模式 → 接口1子账号表」粘贴保存，或直接把 `task_inventory_sub_accounts.txt` 放到站点目录。

## 环境变量（可选）

```
GEJIE_FORWARD_MAX_CONCURRENCY=30       # 转发并发上限（0=不限制）
GEJIE_FORWARD_RATE_LIMIT_PER_SEC=0     # 每秒转发上限
GEJIE_FORWARD_GATE_MAX_WAIT_MS=500     # 并发满时最大排队毫秒
GEJIE_USE_SYSTEM_DNS=1                 # 关掉静态 DNS，走系统 DNS
GEJIE_DATA_DIR=/path                   # 数据目录
```

## 任务库存模式（新增，可选开启）

独立于旧功能的一套「拉任务 → 入库存 → 下发 → 解密回传 → 计数」流程。**默认关闭**，开启后旧功能不受影响。

### 新入口（给新一批客户端用）
- `GET /inventory/task?account_id=<下发id>` —— 从库存取一条任务（返回 `{code:200, success:true, task:{...}}`，与上游任务格式一致）
- `POST /inventory/submit` —— 回传结果（`data` 为 AES 加密内容，服务端解密后异步提交上游）

回传请求体示例：

```json
{
  "account_id": "下发id",
  "trace_id": "trace_001",
  "data": "<AES-128-ECB 加密后的商品JSON（也支持明文JSON）>"
}
```

（`data` 解密后作为接口1回传的 `result` 提交上游。）

### 配置（后台「库存模式」标签页，或直接编辑文件）

下发id 配置写在 `task_inventory.json`（子账号**不**写在这里）：

```json
{
  "enabled": false,
  "accounts": [
    {
      "id": "下发id",
      "remark": "备注",
      "enabled": true,
      "stock": 100,
      "concurrency": 50,
      "interval_seconds": 5,
      "timeout_seconds": 300,
      "take_type": ""
    }
  ]
}
```

**接口1 子账号**单独存在大表 `task_inventory_sub_accounts.txt`（后台「库存模式」标签页第二块 textarea，可上千个）。每行一条：

```
username|password              # 全局池：所有下发id共享，轮询
下发id|username|password        # 只给该下发id用
```

字段说明：
- `enabled` 总开关；`accounts[].enabled` 单个下发id开关
- `id` = 下发id（客户端 account_id + 计数归属）
- `stock` = 库存目标（保持多少条在库）；`concurrency` = 每轮并发拉取账号上限；`interval_seconds` = 拉取间隔秒
- `timeout_seconds` = 任务超时秒（默认 300 = 5 分钟，超时未回传即丢弃）
- `take_type` = 空则用普通任务接口 `/api/task/take`；填 `"r"` 用 R 任务接口 `type=r`
- `concurrency` 建议调大（如 50~200），服务端会按「库存缺口」调用相应数量的不同账号并发抢任务，缺口大时一次上很多账号

### 逻辑（拉取/回传都走接口1）
1. 后台 goroutine 每 `interval_seconds` 秒补库存：缺口 = `stock − 在库 − 已取未回传`，用 `min(缺口, concurrency, 池内账号数)` 个**不同**接口1子账号并发 `GET /api/task/take`（轮询游标，同一轮不重复用同一账号），任务入库存。
2. 客户端 `GET /inventory/task` 取走任务（服务端封装：接口1 返回的 data + `trace_id`），进入「已取未回传」表，5 分钟未回传自动丢弃。
3. 客户端 `POST /inventory/submit` 回传，服务端 AES 解密 → 组装 `{appVersion, url, result}` → **异步** `POST /api/task/submit/v2`（带该任务对应子账号的 token）。
4. 接口1 返回成功 → 该下发id **当日成功任务量 +1**（持久化到 `task_inventory_stats.json`）。

后台管理接口：`?act=inventory_config`（读）、`?act=inventory_save`（写，POST）、`?act=inventory_sub_table`（读子账号表）、`?act=inventory_sub_table_save`（写，POST）、`?act=inventory_status`（运行状态 + 当日统计）。

## 兼容性说明

- 所有 JSON 数据文件、日志 `decrypt_proxy_logs.jsonl`、`accounts_share_data.php` 格式与 PHP 完全一致，可直接迁移。
- 解密方式（AES-128-ECB + PKCS7）、上游接口地址、token 缓存、设备控制、子账号、插件账号、备注链接、平行控制页全部保留。
- 任务队列/租约（原 PHP 存 SQLite，TTL 10~30 分钟）改为内存 + JSON 持久化，属瞬时状态，不影响任何用户数据。
- 接口2（zb2）按原样保持“已停用”返回。
