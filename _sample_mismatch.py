# -*- coding: utf-8 -*-
"""抽样最近的 item_id_mismatch 记录，核对任务 URL 与回传数据到底哪个商品。"""
import json

import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("38.175.194.43", 22, "root", "85TcVQKflEJwHJDx", timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=240):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


SCRIPT = r'''
import json, re
LOG = "/www/wwwroot/38.175.194.43/后台111/decrypt_proxy_logs.jsonl"
rows = []
with open(LOG, encoding="utf-8", errors="replace") as f:
    for line in f:
        if "item_id_mismatch" not in line:
            continue
        try: d = json.loads(line)
        except Exception: continue
        rows.append(d)
print("  最近 mismatch 总数(文件内):", len(rows))
for d in rows[-5:]:
    print("=" * 60)
    print("  时间:", d.get("time"), " 账号:", d.get("account_id") or d.get("username"))
    print("  task_url:", (d.get("task_url") or "")[:100])
    print("  task_item_id:", d.get("task_item_id"), " result_item_id:", d.get("result_item_id"))
    rb = d.get("request_body") or {}
    print("  回传 url:", (rb.get("url") or "")[:100])
    res = rb.get("result") or ""
    if res:
        try:
            j = json.loads(res)
            data = j.get("data") or {}
            item = data.get("item") or {}
            print("  数据内 name:", str(item.get("name"))[:40], " item_id:", item.get("item_id"), " shop_id:", item.get("shop_id"))
            ids = set(re.findall(r'"itemid"\s*:\s*(\d{6,})', res))
            print("  数据里出现的 itemid 数:", len(ids), list(ids)[:5])
        except Exception as e:
            print("  解析失败:", e)
'''
sftp = c.open_sftp()
with sftp.open("/tmp/chk_mm.py", "w") as f:
    f.write(SCRIPT)
sftp.close()
print(sh("python3 /tmp/chk_mm.py")[:3000])
c.close()
