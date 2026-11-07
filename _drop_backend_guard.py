# -*- coding: utf-8 -*-
"""取消后端 item_id 前置拦截：改成只记录、继续转发，由上游判定。"""
import paramiko

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect("38.175.194.43", 22, "root", "85TcVQKflEJwHJDx", timeout=25, banner_timeout=30, auth_timeout=30)


def sh(cmd, t=300):
    _, o, e = c.exec_command(cmd, timeout=t)
    return (o.read().decode("utf-8", "replace") + e.read().decode("utf-8", "replace")).strip()


D = "/www/wwwroot/38.175.194.43/后台111"
SRC = D + "/go-server"
p = SRC + "/inventory_http.go"
src = sh("cat '%s'" % p)

old1 = '''			invRecordFail(accountID)
			writeJSONOK(w, map[string]any{"code": 410, "success": false,
				"msg": "抓错商品（拿到的是其它商品的数据），请丢弃这个任务"})
			return
		}
	}'''
new1 = '''			// 不拦截：只记录，继续转发由上游判定（避免误杀导致一条都不计数）
		}
	}'''

old2 = '''			invRecordFail(accountID)
			writeJSONOK(w, map[string]any{"code": 400, "success": false, "msg": "item_id 不匹配"})
			return
		}
	}'''
new2 = '''			// 不拦截：同上，交给上游判定
		}
	}'''

changed = 0
if old1 in src:
    src = src.replace(old1, new1, 1)
    changed += 1
if old2 in src:
    src = src.replace(old2, new2, 1)
    changed += 1
print("  替换处数:", changed)
if changed:
    sftp = c.open_sftp()
    with sftp.open("/tmp/inv_http_new.go", "w") as f:
        f.write(src)
    sftp.close()
    sh("cp '%s' '%s.bak_mismatch'" % (p, p))
    print("  写回:", sh("cp /tmp/inv_http_new.go '%s' && echo ok" % p)[:20])

print("\n=== 编译 ===")
out = sh("cd '%s' && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /tmp/ajie-nomis . 2>&1; echo rc=$?" % SRC)
print("  ", out[-400:])
if "rc=0" in out:
    print("  部署:", sh("cp -f /tmp/ajie-nomis %s/ajie-server && chmod +x %s/ajie-server && systemctl restart ajie-server && sleep 3 && systemctl is-active ajie-server" % (D, D)))
else:
    print("  编译失败，未部署")

c.close()
