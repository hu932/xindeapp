# -*- coding: utf-8 -*-
"""带 Host 头验证 38 的 /splash/ 静态访问。"""
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("38.175.194.43", 22, "root", "85TcVQKflEJwHJDx", timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=120):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


fmt = "-s -o /dev/null -w %{http_code}:%{size_download} -H 'Host: 38.175.194.43' -m 20 "
for path in ("/splash/list.json", "/splash/splash_01.mp4"):
    print("  %-28s %s" % (path, sh("curl " + fmt + "http://127.0.0.1" + path)))
c.close()
