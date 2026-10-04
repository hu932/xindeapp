<?php
/**
 * 优惠券预检服务器 - 管理后台
 * 访问：http://你的域名/coupon-server/admin/
 */

require_once dirname(__DIR__) . '/lib.php';

session_start();

function admin_csrf_token() {
    if (empty($_SESSION['csrf'])) {
        $_SESSION['csrf'] = bin2hex(random_bytes(16));
    }
    return $_SESSION['csrf'];
}
function admin_csrf_check($token) {
    return isset($_SESSION['csrf']) && is_string($token) && hash_equals($_SESSION['csrf'], $token);
}
function admin_e($s) {
    echo htmlspecialchars((string)$s, ENT_QUOTES, 'UTF-8');
}

$loggedIn = !empty($_SESSION['admin']);
$notice = '';
$loginError = '';

// 登出
if (isset($_GET['action']) && $_GET['action'] === 'logout') {
    session_unset();
    session_destroy();
    header('Location: index.php');
    exit;
}

// 登录
if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['login'])) {
    $u = trim((string)($_POST['username'] ?? ''));
    $p = (string)($_POST['password'] ?? '');
    $acfg = coupon_admin_config();
    if (hash_equals($acfg['user'], $u) && hash_equals($acfg['pass'], $p)) {
        session_regenerate_id(true);
        $_SESSION['admin'] = true;
        header('Location: index.php');
        exit;
    }
    $loginError = '用户名或密码错误';
    $loggedIn = false;
}

// 已登录：处理后台操作
if ($loggedIn && $_SERVER['REQUEST_METHOD'] === 'POST' && !isset($_POST['login'])) {
    if (!admin_csrf_check($_POST['csrf'] ?? '')) {
        $notice = 'CSRF 校验失败，请重试';
    } else {
        $action = $_POST['action'] ?? '';
        if ($action === 'add_product' || $action === 'save_product') {
            $id     = trim((string)($_POST['id'] ?? ''));
            $url    = trim((string)($_POST['url'] ?? ''));
            $shopId = trim((string)($_POST['shopId'] ?? ''));
            $itemId = trim((string)($_POST['itemId'] ?? ''));
            if ($id === '' || $url === '') {
                $notice = '商品 id 和 url 不能为空';
            } else {
                $list = coupon_probe_products();
                if ($action === 'add_product') {
                    $list[] = ['id' => $id, 'url' => $url, 'shopId' => $shopId, 'itemId' => $itemId];
                    $notice = '已添加探测商品';
                } else {
                    $idx = (int)($_POST['idx'] ?? -1);
                    if (isset($list[$idx])) {
                        $list[$idx] = ['id' => $id, 'url' => $url, 'shopId' => $shopId, 'itemId' => $itemId];
                        $notice = '已保存修改';
                    } else {
                        $notice = '该商品不存在';
                    }
                }
                coupon_save_probe_products($list);
            }
        } elseif ($action === 'delete_product') {
            $idx = (int)($_POST['idx'] ?? -1);
            $list = coupon_probe_products();
            if (isset($list[$idx])) {
                unset($list[$idx]);
                coupon_save_probe_products(array_values($list));
                $notice = '已删除商品';
            } else {
                $notice = '该商品不存在';
            }
        } elseif ($action === 'clear_passed') {
            coupon_clear_passed_records();
            $notice = '已清空通过记录';
        } elseif ($action === 'clear_failed') {
            coupon_clear_failed_records();
            $notice = '已清空失败记录';
        } elseif ($action === 'clear_sessions') {
            coupon_clear_session_records();
            $notice = '已清空检测任务';
        }
    }
}

$products = $loggedIn ? coupon_probe_products() : [];
$passed   = $loggedIn ? coupon_passed_records() : [];
$failed   = $loggedIn ? coupon_failed_records() : [];
$sessions = $loggedIn ? coupon_session_records() : [];
?>
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>优惠券预检服务器 · 管理后台</title>
<style>
  :root {
    --bg: #f4f6fb;
    --card: #ffffff;
    --border: #e3e8f0;
    --text: #1f2a37;
    --muted: #6b7280;
    --primary: #2563eb;
    --primary-dark: #1d4ed8;
    --danger: #dc2626;
    --success: #059669;
    --radius: 12px;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
    background: var(--bg);
    color: var(--text);
    font-size: 14px;
    line-height: 1.5;
  }
  a { color: var(--primary); text-decoration: none; }

  /* 登录页 */
  .login-wrap { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
  .login-card {
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 10px 40px rgba(15, 23, 42, .08);
    padding: 36px;
    width: 100%;
    max-width: 360px;
  }
  .login-card h1 { font-size: 20px; margin-bottom: 6px; }
  .login-card p.sub { color: var(--muted); margin-bottom: 24px; font-size: 13px; }
  .field { margin-bottom: 14px; }
  .field label { display: block; font-size: 13px; margin-bottom: 6px; color: var(--muted); }
  .field input {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    font-size: 14px;
    outline: none;
    transition: border-color .15s;
  }
  .field input:focus { border-color: var(--primary); }
  .btn {
    display: inline-flex; align-items: center; justify-content: center;
    padding: 10px 16px; border: none; border-radius: 8px;
    font-size: 14px; cursor: pointer; transition: background .15s;
  }
  .btn-primary { background: var(--primary); color: #fff; width: 100%; }
  .btn-primary:hover { background: var(--primary-dark); }
  .btn-danger { background: #fee2e2; color: var(--danger); }
  .btn-danger:hover { background: #fecaca; }
  .btn-sm { padding: 6px 10px; font-size: 13px; border-radius: 6px; }
  .error { background: #fee2e2; color: var(--danger); padding: 10px 12px; border-radius: 8px; margin-bottom: 16px; font-size: 13px; }

  /* 后台布局 */
  .topbar {
    background: var(--card); border-bottom: 1px solid var(--border);
    padding: 14px 24px; display: flex; align-items: center; justify-content: space-between;
    position: sticky; top: 0; z-index: 10;
  }
  .topbar .brand { font-size: 16px; font-weight: 600; }
  .topbar .brand span { color: var(--muted); font-weight: 400; font-size: 13px; margin-left: 8px; }
  .container { max-width: 1100px; margin: 0 auto; padding: 24px; }
  .notice { background: #ecfdf5; color: var(--success); 4adding: 10px 14px; border-radius: 8px; margin-bottom: 20px; font-size: 13px; }
  .cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
  .stat {
    background: var(--card); border: 1px solid var(--border); border-radius: var(--radius);
    padding: 18px 20px;
  }
  .stat .num { font-size: 28px; font-weight: 700; }
  .stat .label { color: var(--muted); font-size: 13px; margin-top: 4px; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 24px; overflow: hidden; }
  .card-head { padding: 16px 20px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; }
  .card-head h2 { font-size: 15px; }
  .card-body { padding: 20px; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); vertical-align: middle; }
  th { font-size: 12px; color: var(--muted); font-weight: 600; background: #fafbfd; }
  tr:last-child td { border-bottom: none; }
  td input[type="text"] {
    width: 100%; padding: 7px 9px; border: 1px solid var(--border); border-radius: 6px; font-size: 13px;
  }
  .row-actions { display: flex; gap: 6px; white-space: nowrap; }
  .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; color: var(--muted); word-break: break-all; }
  .empty { color: var(--muted); text-align: center; padding: 24px 0; }
  .add-form { display: grid; grid-template-columns: 1.2fr 2fr 1fr 1fr auto; gap: 8px; align-items: end; }
  @media (max-width: 760px) {
    .cards { grid-template-columns: 1fr; }
    .add-form { grid-template-columns: 1fr; }
  }
</style>
</head>
<body>

<?php if (!$loggedIn): ?>
<div class="login-wrap">
  <div class="login-card">
    <h1>优惠券预检服务器</h1>
    <p class="sub">管理后台登录</p>
    <?php if ($loginError): ?><div class="error"><?php admin_e($loginError); ?></div><?php endif; ?>
    <form method="post" autocomplete="off">
      <input type="hidden" name="login" value="1">
      <div class="field">
        <label>账号</label>
        <input type="text" name="username" required autofocus>
      </div>
      <div class="field">
        <label>密码</label>
        <input type="password" name="password" required>
      </div>
      <button class="btn btn-primary" type="submit">登录</button>
    </form>
  </div>
</div>
<?php else: ?>

<div class="topbar">
  <div class="brand">优惠券预检服务器<span>管理后台</span></div>
  <a class="btn btn-danger btn-sm" href="index.php?action=logout">退出登录</a>
</div>

<div class="container">
  <?php if ($notice): ?><div class="notice"><?php admin_e($notice); ?></div><?php endif; ?>

  <div class="cards">
    <div class="stat"><div class="num"><?php echo count($products); ?></div><div class="label">探测商品数</div></div>
    <div class="stat"><div class="num"><?php echo count($passed); ?></div><div class="label">已通过检测的账号记录</div></div>
    <div class="stat"><div class="num"><?php echo count($failed); ?></div><div class="label">检测失败账号</div></div>
    <div class="stat"><div class="num"><?php echo count($sessions); ?></div><div class="label">待处理检测任务</div></div>
  </div>

  <div class="card">
    <div class="card-head"><h2>探测商品管理</h2></div>
    <div class="card-body">
      <form class="add-form" method="post" style="margin-bottom:16px;">
        <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
        <input type="hidden" name="action" value="add_product">
        <input type="text" name="id" placeholder="ID（如 probe-003）" required>
        <input type="text" name="url" placeholder="Shopee 商品链接 https://shopee.tw/..." required>
        <input type="text" name="shopId" placeholder="shopId（可选）">
        <input type="text" name="itemId" placeholder="itemId（可选）">
        <button class="btn btn-primary" type="submit">添加</button>
      </form>

      <?php if (empty($products)): ?>
        <div class="empty">还没有探测商品，请先添加一个带优惠券的 Shopee 商品链接。</div>
      <?php else: ?>
      <table>
        <thead><tr><th style="width:110px;">ID</th><th>商品链接</th><th style="width:110px;">shopId</th><th style="width:110px;">itemId</th><th style="width:130px;">操作</th></tr></thead>
        <tbody>
        <?php foreach ($products as $idx => $prod): ?>
          <tr>
            <td><input form="editF<?php echo $idx; ?>" type="text" name="id" value="<?php admin_e($prod['id'] ?? ''); ?>"></td>
            <td><input form="editF<?php echo $idx; ?>" type="text" name="url" value="<?php admin_e($prod['url'] ?? ''); ?>"></td>
            <td><input form="editF<?php echo $idx; ?>" type="text" name="shopId" value="<?php admin_e($prod['shopId'] ?? ''); ?>"></td>
            <td><input form="editF<?php echo $idx; ?>" type="text" name="itemId" value="<?php admin_e($prod['itemId'] ?? ''); ?>"></td>
            <td>
              <div class="row-actions">
                <form id="editF<?php echo $idx; ?>" method="post" style="display:inline;">
                  <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
                  <input type="hidden" name="action" value="save_product">
                  <input type="hidden" name="idx" value="<?php echo $idx; ?>">
                  <button class="btn btn-primary btn-sm" type="submit">保存</button>
                </form>
                <form method="post" style="display:inline;">
                  <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
                  <input type="hidden" name="action" value="delete_product">
                  <input type="hidden" name="idx" value="<?php echo $idx; ?>">
                  <button class="btn btn-danger btn-sm" type="submit" onclick="return confirm('确定删除该商品？');">删除</button>
                </form>
              </div>
            </td>
          </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
      <?php endif; ?>
    </div>
  </div>

  <div class="card">
    <div class="card-head">
      <h2>已通过检测的账号</h2>
      <form method="post" onsubmit="return confirm('确定清空所有通过记录？清空后这些账号当天会重新检测。');">
        <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
        <input type="hidden" name="action" value="clear_passed">
        <button class="btn btn-danger btn-sm" type="submit">清空记录</button>
      </form>
    </div>
    <div class="card-body">
      <?php if (empty($passed)): ?>
        <div class="empty">暂无通过记录。</div>
      <?php else: ?>
      <table>
        <thead><tr><th>Shopee 账号</th><th>probeId</th><th>通过时间</th></tr></thead>
        <tbody>
        <?php foreach ($passed as $rec): ?>
          <tr>
            <td><?php admin_e($rec['shopeeUserId'] ?? '-'); ?></td>
            <td class="mono"><?php admin_e($rec['probeId'] ?? '-'); ?></td>
            <td><?php echo isset($rec['passedAt']) ? date('Y-m-d H:i:s', (int)$rec['passedAt']) : '-'; ?></td>
          </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
      <?php endif; ?>
    </div>
  </div>

  <div class="card">
    <div class="card-head">
      <h2>检测失败的账号</h2>
      <form method="post" onsubmit="return confirm('确定清空所有失败记录？');">
        <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
        <input type="hidden" name="action" value="clear_failed">
        <button class="btn btn-danger btn-sm" type="submit">清空记录</button>
      </form>
    </div>
    <div class="card-body">
      <?php if (empty($failed)): ?>
        <div class="empty">暂无失败记录。</div>
      <?php else: ?>
      <table>
        <thead><tr><th>Shopee 账号</th><th>probeId</th><th>失败原因</th><th>失败时间</th></tr></thead>
        <tbody>
        <?php foreach ($failed as $rec): ?>
          <tr>
            <td><?php admin_e($rec['shopeeUserId'] ?? '-'); ?></td>
            <td class="mono"><?php admin_e($rec['probeId'] ?? '-'); ?></td>
            <td><?php admin_e($rec['reason'] ?? '-'); ?></td>
            <td><?php echo isset($rec['failedAt']) ? date('Y-m-d H:i:s', (int)$rec['failedAt']) : '-'; ?></td>
          </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
      <?php endif; ?>
    </div>
  </div>

  <div class="card">
    <div class="card-head">
      <h2>待处理检测任务</h2>
      <form method="post" onsubmit="return confirm('确定清空所有检测任务？');">
        <input type="hidden" name="csrf" value="<?php admin_e(admin_csrf_token()); ?>">
        <input type="hidden" name="action" value="clear_sessions">
        <button class="btn btn-danger btn-sm" type="submit">清空任务</button>
      </form>
    </div>
    <div class="card-body">
      <?php if (empty($sessions)): ?>
        <div class="empty">暂无待处理检测任务。</div>
      <?php else: ?>
      <table>
        <thead><tr><th>Shopee 账号</th><th>probeId</th><th>分配时间</th></tr></thead>
        <tbody>
        <?php foreach ($sessions as $rec): ?>
          <tr>
            <td><?php admin_e($rec['shopeeUserId'] ?? '-'); ?></td>
            <td class="mono"><?php admin_e($rec['probeId'] ?? '-'); ?></td>
            <td><?php echo isset($rec['assignedAt']) ? date('Y-m-d H:i:s', (int)$rec['assignedAt']) : '-'; ?></td>
          </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
      <?php endif; ?>
    </div>
  </div>
</div>

<?php endif; ?>
</body>
</html>
