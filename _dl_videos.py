# -*- coding: utf-8 -*-
"""从 suyanw 接口下载一批视频到 38 服务器，作为稳定的启动视频源。"""
import os
import ssl
import time
import urllib.request

import paramiko

API = "https://api.suyanw.cn/api/jhsp.php?msg=%E5%AF%B9%E8%84%B8%E8%87%AA%E6%8B%8D"
HOST = ("38.175.194.43", "85TcVQKflEJwHJDx")
REMOTE_DIR = "/www/wwwroot/38.175.194.43/后台111/splash"
COUNT = 6
LOCAL_DIR = r"C:\Users\Administrator\Desktop\appnx_xinde\_videos"
os.makedirs(LOCAL_DIR, exist_ok=True)

CTX = ssl.create_default_context()
CTX.check_hostname = False
CTX.verify_mode = ssl.CERT_NONE

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST[0], 22, "root", HOST[1], timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=180):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


print("  建目录:", sh("mkdir -p %s && echo ok" % REMOTE_DIR))

ok = []
for i in range(1, COUNT + 1):
    url = API + "&t=%d" % int(time.time() * 1000 + i)
    dst = os.path.join(LOCAL_DIR, "splash_%02d.mp4" % i)
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "okhttp/4.9.3"})
        with urllib.request.urlopen(req, timeout=40, context=CTX) as r:
            ctype = r.headers.get("Content-Type", "")
            data = r.read()
        if "video" not in ctype.lower() or len(data) < 50_000:
            print("  %d 跳过：type=%s size=%d" % (i, ctype, len(data)))
            continue
        open(dst, "wb").write(data)
        print("  %d 下载成功 %.1f MB" % (i, len(data) / 1048576))
        ok.append((dst, "splash_%02d.mp4" % i))
    except Exception as e:
        print("  %d 失败: %s" % (i, e))
    time.sleep(1)

print("\n  上传到服务器…")
sftp = c.open_sftp()
for local, name in ok:
    sftp.put(local, "%s/%s" % (REMOTE_DIR, name))
    print("   ↑", name)
sftp.close()

print("\n  服务器上的文件:", sh("ls -lh %s | tail -8" % REMOTE_DIR))
print("\n  建立对外访问路径…")
print(sh("cp -r %s /www/wwwroot/189.105.68.250/splash 2>/dev/null; echo done"))
c.close()
print("\n  完成，共 %d 个视频" % len(ok))
