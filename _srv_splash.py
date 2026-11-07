# -*- coding: utf-8 -*-
"""38 上给启动视频配静态访问：/splash/ 目录 + list.json。"""
import json

import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("38.175.194.43", 22, "root", "85TcVQKflEJwHJDx", timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=180):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


DIR = "/www/wwwroot/38.175.194.43/后台111/splash"
SITE = "/www/server/panel/vhost/nginx/38.175.194.43.conf"

print("  视频目录:", sh("ls %s/*.mp4 2>/dev/null | wc -l" % DIR), "个")

# list.json
files = [f.split("/")[-1] for f in sh("ls %s/*.mp4" % DIR).split()]
payload = {"videos": files, "base": "/splash/"}
sftp = c.open_sftp()
with sftp.open(DIR + "/list.json", "w") as f:
    f.write(json.dumps(payload, ensure_ascii=False, indent=2))
sftp.close()
print("  list.json:", files)

site = sh("cat '%s' 2>/dev/null" % SITE)
print("  站点配置存在:", bool(site), len(site))

if site and "/splash/" not in site:
    block = """
    location /splash/ {
        alias /www/wwwroot/38.175.194.43/后台111/splash/;
        autoindex off;
        add_header Cache-Control "public, max-age=86400";
        types { video/mp4 mp4; application/json json; }
    }
"""
    key = "include enable-php-00.conf;"
    i = site.find(key)
    if i < 0:
        key = "location ^~ /"
        i = site.find(key)
    assert i > 0, "站点配置锚点未找到"
    new_site = site[:i] + block + site[i:]
    sftp = c.open_sftp()
    with sftp.open("/tmp/site38.conf", "w") as f:
        f.write(new_site)
    sftp.close()
    sh("cp '%s' '%s.bak_splash'" % (SITE, SITE))
    print("  写回:", sh("cp /tmp/site38.conf '%s' && echo ok" % SITE))
    print("  nginx -t:", sh("nginx -t 2>&1 | tail -1"))
    print("  reload:", sh("nginx -s reload; sleep 1; echo done"))
else:
    print("  - 已有 /splash/ 或站点配置不存在")

print("\n=== 验证 ===")
for p in ("/splash/list.json", "/splash/splash_01.mp4"):
    code = sh("curl -s -o /dev/null -w '%%{http_code} %%{size_download} %%{content_type}' -m 15 http://127.0.0.1%s" % p)
    print("  %-26s %s" % (p, code))

c.close()
