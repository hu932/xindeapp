#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# 宝塔一键部署脚本（在站点根目录执行，例如 /www/wwwroot/后台111）
#   bash go-server/deploy.sh
# 作用：编译 Linux 二进制 -> 放到站点根目录（与数据同目录）-> 写 systemd -> 启动
# ============================================================

SITE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$SITE_DIR/ajie-server"
SVC="/etc/systemd/system/ajie-server.service"
PORT="${PORT:-2036}"

if ! command -v go >/dev/null 2>&1; then
  echo "[错误] 未检测到 Go。请先安装：宝塔软件商店搜 Go，或见 README「部署 第二步」" >&2
  exit 1
fi

echo "==> 编译 Linux 二进制（源码：$SITE_DIR/go-server）"
cd "$SITE_DIR/go-server"
go mod tidy 2>/dev/null || go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$BIN" .
chmod +x "$BIN"
echo "==> 二进制已生成：$BIN"

# 数据文件本来就在站点根目录（与二进制同目录），无需额外复制；
# 库存模式首次运行会自动生成 task_inventory.json / task_inventory_stats.json 等。

echo "==> 写入 systemd 服务：$SVC"
cat > "$SVC" <<EOF
[Unit]
Description=Ajie Server
After=network.target

[Service]
WorkingDirectory=$SITE_DIR
ExecStart=$BIN -listen :$PORT
Restart=always
RestartSec=2
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ajie-server >/dev/null 2>&1 || true
# 关键：必须 restart，否则已运行的服务不会加载新二进制（enable --now 不会重启已运行的服务）
systemctl restart ajie-server
sleep 1
systemctl status ajie-server --no-pager || true

echo ""
echo "============================================================"
echo "完成！服务监听 :$PORT，数据目录：$SITE_DIR"
echo "宝塔站点 -> 反向代理 -> http://127.0.0.1:$PORT"
echo "查看日志：journalctl -u ajie-server -f"
echo "============================================================"
