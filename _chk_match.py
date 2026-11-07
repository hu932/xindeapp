# -*- coding: utf-8 -*-
"""检查 38 的 item_id 提取与最近「抓错商品」判定是否误判。"""
import json

import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("38.175.194.43", 22, "root", "85TcVQKflEJwHJDx", timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=180):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


S = "/www/wwwroot/38.175.194.43/后台111/go-server"
print("=== submitItemIDFromURL ===")
print(sh("sed -n '/func submitItemIDFromURL/,/^}/p' %s/inventory_http.go" % S)[:900])
print("\n=== submitItemIDFromResult ===")
print(sh("sed -n '/func submitItemIDFromResult/,/^}/p' %s/inventory_http.go" % S)[:1200])
c.close()
