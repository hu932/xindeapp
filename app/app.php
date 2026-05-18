<?php
if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
}

$apiUrl = 'upload_count_api.php';
$today = date('Y-m-d');
const APP_AUTH_USERNAME = '格界';
const APP_AUTH_PASSWORD = '2026888';
const APP_AUTH_SESSION_KEY = 'app_panel_auth';
const APP_ICON_FILE = '1777990327149137925.png';
const APP_LOGIN_BG_FILE = '1777990939581269311.png';
const APP_MIRROR_TOKEN = 'shp-mirror-20260506-7ffb8429c9b54c3e';
const APP_MIRROR_LOG_FILE = __DIR__ . DIRECTORY_SEPARATOR . 'mirror_submit_logs.jsonl';
const APP_MIRROR_MAX_LOGS = 200;

function appJsonResponse(array $payload): void {
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function appRequestHeaders(): array {
    if (function_exists('getallheaders')) {
        $headers = getallheaders();
        if (is_array($headers)) {
            $normalized = [];
            foreach ($headers as $key => $value) {
                $normalized[strtolower((string)$key)] = is_scalar($value) ? (string)$value : '';
            }
            return $normalized;
        }
    }

    $normalized = [];
    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'HTTP_') !== 0) {
            continue;
        }
        $headerName = strtolower(str_replace('_', '-', substr($key, 5)));
        $normalized[$headerName] = is_scalar($value) ? (string)$value : '';
    }

    if (!isset($normalized['content-type']) && isset($_SERVER['CONTENT_TYPE'])) {
        $normalized['content-type'] = (string)$_SERVER['CONTENT_TYPE'];
    }

    return $normalized;
}

function appClientIp(): string {
    $forwarded = trim((string)($_SERVER['HTTP_X_FORWARDED_FOR'] ?? ''));
    if ($forwarded !== '') {
        $parts = explode(',', $forwarded);
        $candidate = trim((string)($parts[0] ?? ''));
        if ($candidate !== '') {
            return $candidate;
        }
    }

    $remoteAddr = trim((string)($_SERVER['REMOTE_ADDR'] ?? ''));
    return $remoteAddr !== '' ? $remoteAddr : 'unknown';
}

function appMirrorDecodeResult($rawResult) {
    if (!is_string($rawResult) || $rawResult === '') {
        return null;
    }

    $decoded = json_decode($rawResult, true);
    if (json_last_error() !== JSON_ERROR_NONE) {
        return null;
    }

    return $decoded;
}

function appMirrorTrimLogs(int $maxLogs = APP_MIRROR_MAX_LOGS): void {
    if (!is_file(APP_MIRROR_LOG_FILE)) {
        return;
    }

    $lines = @file(APP_MIRROR_LOG_FILE, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
    if (!is_array($lines) || count($lines) <= $maxLogs) {
        return;
    }

    $lines = array_slice($lines, -$maxLogs);
    @file_put_contents(APP_MIRROR_LOG_FILE, implode(PHP_EOL, $lines) . PHP_EOL, LOCK_EX);
}

function appMirrorAppendLog(array $entry): void {
    $json = json_encode($entry, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    if (!is_string($json) || $json === '') {
        return;
    }

    @file_put_contents(APP_MIRROR_LOG_FILE, $json . PHP_EOL, FILE_APPEND | LOCK_EX);
    appMirrorTrimLogs();
}

function appMirrorReadLogs(int $limit = 50): array {
    if (!is_file(APP_MIRROR_LOG_FILE)) {
        return [];
    }

    $lines = @file(APP_MIRROR_LOG_FILE, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
    if (!is_array($lines) || !$lines) {
        return [];
    }

    $slice = array_reverse(array_slice($lines, -$limit));
    $list = [];
    foreach ($slice as $line) {
        $decoded = json_decode($line, true);
        if (is_array($decoded)) {
            $list[] = $decoded;
        }
    }

    return $list;
}

function appMirrorExtractToken(array $headers, array $payload): string {
    $headerToken = trim((string)($headers['x-debug-token'] ?? ''));
    if ($headerToken !== '') {
        return $headerToken;
    }

    $queryToken = trim((string)($_GET['mirror_token'] ?? ''));
    if ($queryToken !== '') {
        return $queryToken;
    }

    return trim((string)($payload['mirror_token'] ?? ''));
}

function appMirrorRenderPage(): void {
    if (!appIsAuthenticated()) {
        appRenderLoginPage();
    }

    $logs = appMirrorReadLogs(50);
    ?>
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>上传镜像调试</title>
  <style>
    :root {
      --bg:#f4efe8;
      --panel:#ffffff;
      --text:#2f2721;
      --muted:#7d7166;
      --line:#eadfd2;
      --accent:#df9552;
      --accent-strong:#c67934;
      --shadow:0 18px 42px rgba(93,71,46,.08);
      --radius:22px;
    }
    * { box-sizing:border-box; }
    body {
      margin:0;
      padding:24px;
      font-family:Inter,ui-sans-serif,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      color:var(--text);
      background:linear-gradient(180deg,#faf6f1 0%, var(--bg) 100%);
    }
    .shell { max-width:1200px; margin:0 auto; }
    .topbar {
      display:flex;
      gap:12px;
      align-items:center;
      justify-content:space-between;
      margin-bottom:18px;
      flex-wrap:wrap;
    }
    .title h1 { margin:0; font-size:30px; letter-spacing:-.03em; }
    .title p { margin:6px 0 0; color:var(--muted); font-size:14px; }
    .actions { display:flex; gap:10px; flex-wrap:wrap; }
    .btn {
      display:inline-flex;
      align-items:center;
      justify-content:center;
      height:42px;
      padding:0 16px;
      border-radius:14px;
      border:1px solid var(--line);
      background:#fff;
      color:var(--text);
      text-decoration:none;
      font-weight:700;
      cursor:pointer;
    }
    .btn.primary {
      border-color:transparent;
      color:#fff;
      background:linear-gradient(135deg, var(--accent), var(--accent-strong));
      box-shadow:0 14px 28px rgba(198,121,52,.24);
    }
    .list { display:grid; gap:16px; }
    .card {
      background:rgba(255,255,255,.92);
      border:1px solid var(--line);
      border-radius:var(--radius);
      box-shadow:var(--shadow);
      overflow:hidden;
    }
    .meta {
      padding:16px 18px;
      border-bottom:1px solid var(--line);
      display:grid;
      grid-template-columns:repeat(auto-fit,minmax(180px,1fr));
      gap:10px 16px;
    }
    .meta div { font-size:13px; color:var(--muted); }
    .meta strong { color:var(--text); display:block; margin-bottom:4px; }
    .body { padding:18px; display:grid; gap:14px; }
    .block strong { display:block; margin-bottom:8px; font-size:13px; }
    pre {
      margin:0;
      padding:16px;
      border-radius:16px;
      background:#211b17;
      color:#f5efe9;
      overflow:auto;
      white-space:pre-wrap;
      word-break:break-word;
      font-size:12px;
      line-height:1.6;
    }
    .empty {
      padding:40px 24px;
      text-align:center;
      color:var(--muted);
      background:rgba(255,255,255,.82);
      border:1px dashed var(--line);
      border-radius:var(--radius);
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="topbar">
      <div class="title">
        <h1>上传镜像调试</h1>
        <p>查看插件同步到 `app.php?mirror_action=collect` 的最近 50 条上传体。</p>
      </div>
      <div class="actions">
        <a class="btn" href="<?= htmlspecialchars($_SERVER['PHP_SELF'] ?? 'app.php', ENT_QUOTES, 'UTF-8') ?>">返回后台</a>
        <a class="btn primary" href="<?= htmlspecialchars(($_SERVER['PHP_SELF'] ?? 'app.php') . '?mirror_action=view', ENT_QUOTES, 'UTF-8') ?>">刷新</a>
      </div>
    </div>

    <?php if (!$logs): ?>
      <div class="empty">还没有收到镜像上传数据。</div>
    <?php else: ?>
      <div class="list">
        <?php foreach ($logs as $log): ?>
          <?php
            $payloadPretty = json_encode($log['payload'] ?? null, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
            $resultPretty = json_encode($log['decoded_result'] ?? null, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
            $rawPretty = (string)($log['raw_body'] ?? '');
          ?>
          <div class="card">
            <div class="meta">
              <div><strong>接收时间</strong><?= htmlspecialchars((string)($log['received_at'] ?? ''), ENT_QUOTES, 'UTF-8') ?></div>
              <div><strong>Trace ID</strong><?= htmlspecialchars((string)($log['trace_id'] ?? ''), ENT_QUOTES, 'UTF-8') ?></div>
              <div><strong>Shop / Item</strong><?= htmlspecialchars((string)($log['shop_id'] ?? ''), ENT_QUOTES, 'UTF-8') ?> / <?= htmlspecialchars((string)($log['item_id'] ?? ''), ENT_QUOTES, 'UTF-8') ?></div>
              <div><strong>来源 IP</strong><?= htmlspecialchars((string)($log['ip'] ?? ''), ENT_QUOTES, 'UTF-8') ?></div>
              <div><strong>内容类型</strong><?= htmlspecialchars((string)($log['content_type'] ?? ''), ENT_QUOTES, 'UTF-8') ?></div>
              <div><strong>Body 字节数</strong><?= (int)($log['raw_body_length'] ?? 0) ?></div>
            </div>
            <div class="body">
              <div class="block">
                <strong>上传体 JSON</strong>
                <pre><?= htmlspecialchars(is_string($payloadPretty) ? $payloadPretty : 'null', ENT_QUOTES, 'UTF-8') ?></pre>
              </div>
              <div class="block">
                <strong>`result` 解码后 JSON</strong>
                <pre><?= htmlspecialchars(is_string($resultPretty) ? $resultPretty : 'null', ENT_QUOTES, 'UTF-8') ?></pre>
              </div>
              <div class="block">
                <strong>原始请求体</strong>
                <pre><?= htmlspecialchars($rawPretty, ENT_QUOTES, 'UTF-8') ?></pre>
              </div>
            </div>
          </div>
        <?php endforeach; ?>
      </div>
    <?php endif; ?>
  </div>
</body>
</html>
    <?php
    exit;
}

function appAuthUser(): ?string {
    $user = $_SESSION[APP_AUTH_SESSION_KEY]['username'] ?? null;
    return is_string($user) && $user !== '' ? $user : null;
}

function appIsAuthenticated(): bool {
    return appAuthUser() !== null;
}

function appRequireAuthJson(): void {
    if (!appIsAuthenticated()) {
        http_response_code(401);
        appJsonResponse(['code' => 401, 'msg' => '请先登录']);
    }
}

function appHandleLogout(): void {
    unset($_SESSION[APP_AUTH_SESSION_KEY]);
}

function appRenderLoginPage(string $errorMessage = ''): void {
    $safeError = htmlspecialchars($errorMessage, ENT_QUOTES, 'UTF-8');
    ?>
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>格界管理后台 · 登录</title>
  <link rel="icon" type="image/png" href="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>">
  <link rel="apple-touch-icon" href="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>">
  <style>
    :root {
      --sand:#f4efe8;
      --cream:#fbf7f2;
      --ink:#2f2721;
      --muted:#86786a;
      --line:rgba(143,113,82,.18);
      --accent:#df9552;
      --accent-strong:#c67934;
      --green:#6fa286;
      --panel:rgba(255,255,255,.78);
      --shadow:0 30px 70px rgba(77,54,30,.12);
      --radius:28px;
    }
    *{box-sizing:border-box}
    html,body{height:100%}
    body{
      margin:0;
      font-family:Inter,ui-sans-serif,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
      color:var(--ink);
      background:
        linear-gradient(135deg, rgba(22,18,14,.28), rgba(22,18,14,.12)),
        linear-gradient(180deg, rgba(248,241,232,.18), rgba(244,239,232,.74)),
        url("<?= htmlspecialchars(APP_LOGIN_BG_FILE, ENT_QUOTES, 'UTF-8') ?>") center/cover no-repeat fixed;
      min-height:100vh;
      display:grid;
      place-items:center;
      padding:24px;
    }
    .auth-shell{
      width:min(1120px,100%);
      display:grid;
      grid-template-columns:1.15fr .85fr;
      border:1px solid rgba(255,255,255,.6);
      border-radius:36px;
      overflow:hidden;
      background:rgba(255,255,255,.18);
      box-shadow:0 38px 90px rgba(47,32,17,.22);
      backdrop-filter:blur(18px) saturate(130%);
    }
    .auth-aside{
      position:relative;
      padding:44px;
      background:
        linear-gradient(160deg, rgba(32,20,10,.18), rgba(32,20,10,.56)),
        linear-gradient(180deg, rgba(225,150,78,.26), rgba(140,84,33,.16)),
        url("<?= htmlspecialchars(APP_LOGIN_BG_FILE, ENT_QUOTES, 'UTF-8') ?>") center/cover no-repeat;
      color:#fffdf9;
      display:flex;
      flex-direction:column;
      justify-content:space-between;
      min-height:680px;
    }
    .auth-badge{
      display:inline-flex;
      align-items:center;
      gap:10px;
      padding:10px 14px;
      border-radius:999px;
      background:rgba(255,255,255,.18);
      border:1px solid rgba(255,255,255,.28);
      font-size:12px;
      font-weight:800;
      letter-spacing:.08em;
      text-transform:uppercase;
      width:max-content;
    }
    .auth-badge-dot{width:10px;height:10px;border-radius:50%;background:#fff4d9;box-shadow:0 0 0 6px rgba(255,255,255,.12)}
    .hero-copy h1{
      margin:20px 0 14px;
      font-size:56px;
      line-height:.95;
      letter-spacing:-.04em;
    }
    .hero-copy p{
      margin:0;
      max-width:420px;
      font-size:16px;
      line-height:1.7;
      color:rgba(255,252,247,.86);
    }
    .auth-tiles{
      display:grid;
      grid-template-columns:repeat(2,minmax(0,1fr));
      gap:14px;
      margin-top:26px;
      max-width:460px;
    }
    .auth-tile{
      padding:16px 18px;
      border-radius:22px;
      background:rgba(255,255,255,.14);
      border:1px solid rgba(255,255,255,.22);
      backdrop-filter:blur(8px);
    }
    .auth-tile strong{display:block;font-size:24px;margin-bottom:6px}
    .auth-tile span{font-size:13px;color:rgba(255,252,247,.8)}
    .auth-quote{
      max-width:430px;
      padding:18px 20px;
      border-radius:24px;
      background:rgba(96,56,18,.16);
      border:1px solid rgba(255,255,255,.16);
      color:rgba(255,252,247,.88);
      line-height:1.7;
      font-size:14px;
    }
    .auth-panel{
      padding:42px 38px;
      background:linear-gradient(180deg, rgba(255,255,255,.82), rgba(255,255,255,.68));
      display:flex;
      align-items:center;
      justify-content:center;
    }
    .auth-card{
      width:min(420px,100%);
      padding:34px;
      border-radius:30px;
      background:rgba(255,255,255,.76);
      border:1px solid var(--line);
      box-shadow:0 24px 54px rgba(91,64,35,.14);
      backdrop-filter:blur(20px) saturate(132%);
    }
    .brand{
      display:flex;
      align-items:center;
      gap:14px;
      margin-bottom:22px;
    }
    .brand-mark{
      width:54px;
      height:54px;
      border-radius:18px;
      overflow:hidden;
      background:rgba(255,255,255,.72);
      box-shadow:0 12px 28px rgba(179,111,40,.18);
      border:1px solid rgba(255,255,255,.52);
    }
    .brand-mark img{
      width:100%;
      height:100%;
      object-fit:cover;
      display:block;
    }
    .brand h2{
      margin:0;
      font-size:26px;
      letter-spacing:-.03em;
    }
    .brand p{
      margin:5px 0 0;
      color:var(--muted);
      font-size:13px;
    }
    .auth-card h3{
      margin:0 0 8px;
      font-size:30px;
      letter-spacing:-.03em;
    }
    .auth-card .sub{
      margin:0 0 24px;
      color:var(--muted);
      line-height:1.65;
      font-size:14px;
    }
    .field{display:grid;gap:8px;margin-bottom:16px}
    .field label{
      font-size:12px;
      font-weight:800;
      letter-spacing:.08em;
      text-transform:uppercase;
      color:#8b7b6d;
    }
    .field input{
      width:100%;
      height:52px;
      border-radius:18px;
      border:1px solid rgba(145,122,97,.18);
      background:rgba(255,255,255,.92);
      padding:0 16px;
      font-size:15px;
      color:var(--ink);
      outline:none;
      transition:.18s ease;
    }
    .field input:focus{
      border-color:rgba(223,149,82,.75);
      box-shadow:0 0 0 4px rgba(223,149,82,.12);
    }
    .error{
      margin:0 0 16px;
      padding:13px 14px;
      border-radius:16px;
      background:rgba(206,86,64,.1);
      border:1px solid rgba(206,86,64,.16);
      color:#aa503f;
      font-size:13px;
    }
    .submit{
      width:100%;
      height:54px;
      border:0;
      border-radius:18px;
      background:linear-gradient(135deg, var(--accent), var(--accent-strong));
      color:#fff;
      font-size:15px;
      font-weight:900;
      letter-spacing:.02em;
      cursor:pointer;
      box-shadow:0 18px 34px rgba(198,121,52,.24);
    }
    .hint{
      margin-top:18px;
      padding-top:18px;
      border-top:1px dashed rgba(145,122,97,.22);
      color:var(--muted);
      font-size:12px;
      line-height:1.7;
    }
    @media (max-width: 920px){
      .auth-shell{grid-template-columns:1fr}
      .auth-aside{min-height:auto;padding:30px}
      .hero-copy h1{font-size:42px}
      .auth-panel{padding:24px}
      .auth-card{padding:28px}
    }
  </style>
</head>
<body>
  <div class="auth-shell">
    <section class="auth-aside">
      <div>
        <div class="auth-badge"><span class="auth-badge-dot"></span>Secure Workspace</div>
        <div class="hero-copy">
          <h1>格界数据</h1>
          <p>统一查看账号表现、日历事件、时间规划与趋势报表。先登录，再进入你的工作台。</p>
        </div>
        <div class="auth-tiles">
          <div class="auth-tile"><strong>14D</strong><span>趋势追踪与高峰识别</span></div>
          <div class="auth-tile"><strong>Cloud</strong><span>日历事件跨端同步</span></div>
          <div class="auth-tile"><strong>Live</strong><span>账号动态与排行榜</span></div>
          <div class="auth-tile"><strong>Focus</strong><span>任务与时间块协同</span></div>
        </div>
      </div>
      <div class="auth-quote">“把每天的上传、计划和节奏收在同一张桌面上，工作会变得更轻、更稳、更清楚。”</div>
    </section>

    <section class="auth-panel">
      <form class="auth-card" method="post" action="<?= htmlspecialchars($_SERVER['PHP_SELF'] ?? 'app.php', ENT_QUOTES, 'UTF-8') ?>">
        <div class="brand">
          <div class="brand-mark">
            <img src="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>" alt="格界图标">
          </div>
          <div>
            <h2>格界</h2>
            <p>Admin Access Portal</p>
          </div>
        </div>

        <h3>欢迎回来</h3>
        <p class="sub">输入管理员账号后继续。PC 端与 H5 端会共用同一份登录状态。</p>

        <?php if ($safeError !== ''): ?>
          <div class="error"><?= $safeError ?></div>
        <?php endif; ?>

        <input type="hidden" name="auth_action" value="login_form">
        <div class="field">
          <label for="login-username">用户名</label>
          <input id="login-username" name="username" type="text" autocomplete="username" placeholder="请输入管理员用户名">
        </div>
        <div class="field">
          <label for="login-password">密码</label>
          <input id="login-password" name="password" type="password" autocomplete="current-password" placeholder="请输入登录密码">
        </div>
        <button class="submit" type="submit">进入工作台</button>
        <div class="hint"></div>
      </form>
    </section>
  </div>
</body>
</html>
    <?php
    exit;
}

/* ========== 登录 API ========== */
if (isset($_GET['auth_action']) || isset($_POST['auth_action'])) {
    $action = (string)($_GET['auth_action'] ?? $_POST['auth_action'] ?? '');

    if ($action === 'status') {
        appJsonResponse([
            'code' => 0,
            'data' => [
                'authenticated' => appIsAuthenticated(),
                'username' => appAuthUser(),
            ]
        ]);
    }

    if ($action === 'logout') {
        appHandleLogout();
        appJsonResponse(['code' => 0, 'msg' => '已退出登录']);
    }

    $payload = null;
    if (stripos($_SERVER['CONTENT_TYPE'] ?? '', 'application/json') !== false) {
        $payload = json_decode(file_get_contents('php://input'), true) ?: [];
    } else {
        $payload = $_POST;
    }

    $username = trim((string)($payload['username'] ?? ''));
    $password = (string)($payload['password'] ?? '');
    $ok = hash_equals(APP_AUTH_USERNAME, $username) && hash_equals(APP_AUTH_PASSWORD, $password);

    if ($action === 'login') {
        if (!$ok) {
            http_response_code(401);
            appJsonResponse(['code' => 401, 'msg' => '用户名或密码错误']);
        }
        $_SESSION[APP_AUTH_SESSION_KEY] = [
            'username' => APP_AUTH_USERNAME,
            'login_at' => time(),
        ];
        appJsonResponse(['code' => 0, 'msg' => '登录成功', 'data' => ['username' => APP_AUTH_USERNAME]]);
    }

    if ($action === 'login_form') {
        if ($ok) {
            $_SESSION[APP_AUTH_SESSION_KEY] = [
                'username' => APP_AUTH_USERNAME,
                'login_at' => time(),
            ];
            header('Location: ' . ($_SERVER['PHP_SELF'] ?? 'app.php'));
            exit;
        }
        appRenderLoginPage('用户名或密码错误，请重试。');
    }
}
/* ========== /登录 API ========== */

/* ========== 上传镜像调试 API ========== */
if (isset($_GET['mirror_action'])) {
    $action = (string)($_GET['mirror_action'] ?? '');

    if ($action === 'view') {
        appMirrorRenderPage();
    }

    if ($action === 'collect') {
        $headers = appRequestHeaders();
        $rawBody = file_get_contents('php://input');
        $rawBody = is_string($rawBody) ? $rawBody : '';
        $payload = json_decode($rawBody, true);
        $payload = is_array($payload) ? $payload : [];
        $token = appMirrorExtractToken($headers, $payload);

        if ($token === '' || !hash_equals(APP_MIRROR_TOKEN, $token)) {
            http_response_code(403);
            appJsonResponse(['code' => 403, 'msg' => 'mirror token invalid']);
        }

        $entry = [
            'id' => date('YmdHis') . '-' . substr(bin2hex(random_bytes(8)), 0, 10),
            'received_at' => date('Y-m-d H:i:s'),
            'ip' => appClientIp(),
            'method' => (string)($_SERVER['REQUEST_METHOD'] ?? 'POST'),
            'content_type' => (string)($_SERVER['CONTENT_TYPE'] ?? ''),
            'trace_id' => trim((string)($headers['x-shp-trace-id'] ?? '')),
            'shop_id' => trim((string)($headers['x-shp-shop-id'] ?? '')),
            'item_id' => trim((string)($headers['x-shp-item-id'] ?? '')),
            'user_agent' => trim((string)($headers['user-agent'] ?? '')),
            'payload' => $payload,
            'decoded_result' => appMirrorDecodeResult($payload['result'] ?? null),
            'raw_body' => $rawBody,
            'raw_body_length' => strlen($rawBody),
        ];

        appMirrorAppendLog($entry);
        appJsonResponse(['code' => 0, 'msg' => 'mirror saved', 'data' => ['id' => $entry['id']]]);
    }

    appRequireAuthJson();

    if ($action === 'list') {
        $limit = (int)($_GET['limit'] ?? 50);
        $limit = max(1, min(200, $limit));
        appJsonResponse(['code' => 0, 'data' => ['list' => appMirrorReadLogs($limit)]]);
    }

    if ($action === 'clear') {
        if (is_file(APP_MIRROR_LOG_FILE)) {
            @unlink(APP_MIRROR_LOG_FILE);
        }
        appJsonResponse(['code' => 0, 'msg' => '镜像日志已清空']);
    }

    appJsonResponse(['code' => 1, 'msg' => '未知镜像操作']);
}
/* ========== /上传镜像调试 API ========== */

function appCreatePdo(): PDO {
    static $pdo = null;
    if ($pdo instanceof PDO) {
        return $pdo;
    }

    $dbHost = '127.0.0.1';
    $dbPort = 3306;
    $dbName = 'app';
    $dbUser = 'app';
    $dbPass = '123456';
    $dsn = "mysql:host={$dbHost};port={$dbPort};dbname={$dbName};charset=utf8mb4";
    $pdo = new PDO($dsn, $dbUser, $dbPass, [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_TIMEOUT => 5,
    ]);
    return $pdo;
}

function appEnsureUploadAccountsRemark(PDO $pdo): void {
    try {
        $pdo->exec("ALTER TABLE `upload_accounts` ADD COLUMN `remark` VARCHAR(255) NOT NULL DEFAULT '' AFTER `account`");
    } catch (Exception $e) {}
}

function appEnsureCalendarEventsTable(PDO $pdo): void {
    $pdo->exec("
        CREATE TABLE IF NOT EXISTS `calendar_events` (
            `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
            `title` VARCHAR(255) NOT NULL,
            `event_date` DATE NOT NULL,
            `event_time` VARCHAR(5) NOT NULL DEFAULT '',
            `description` TEXT NOT NULL,
            `color` VARCHAR(16) NOT NULL DEFAULT '#e8a15f',
            `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            PRIMARY KEY (`id`),
            KEY `idx_event_date` (`event_date`)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    ");
}

function appNormalizeCalendarEvent(array $event): array {
    $title = trim((string)($event['title'] ?? ''));
    $date = trim((string)($event['date'] ?? ''));
    $time = trim((string)($event['time'] ?? ''));
    $desc = trim((string)($event['desc'] ?? ''));
    $color = trim((string)($event['color'] ?? '')) ?: '#e8a15f';
    $slice = function(string $value, int $length): string {
        if (function_exists('mb_substr')) {
            return mb_substr($value, 0, $length, 'UTF-8');
        }
        return substr($value, 0, $length);
    };

    if ($title === '') {
        throw new Exception('事件标题不能为空');
    }
    if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) {
        throw new Exception('事件日期格式错误');
    }
    if ($time !== '' && !preg_match('/^\d{2}:\d{2}$/', $time)) {
        throw new Exception('事件时间格式错误');
    }

    return [
        'title' => $slice($title, 255),
        'date' => $date,
        'time' => $time,
        'desc' => $desc,
        'color' => $slice($color, 16),
    ];
}

/* ========== 云端日历 API ========== */
if (isset($_GET['calendar_action'])) {
    appRequireAuthJson();
    $action = (string)$_GET['calendar_action'];

    try {
        $db = appCreatePdo();
        appEnsureCalendarEventsTable($db);

        if ($action === 'list') {
            $month = trim((string)($_GET['month'] ?? ''));
            if (preg_match('/^\d{4}-\d{2}$/', $month)) {
                $start = $month . '-01';
                $end = date('Y-m-t', strtotime($start));
                $stmt = $db->prepare("SELECT id, title, DATE_FORMAT(event_date, '%Y-%m-%d') AS date, event_time AS time, description AS `desc`, color, created_at, updated_at FROM calendar_events WHERE event_date BETWEEN ? AND ? ORDER BY event_date ASC, event_time ASC, id ASC");
                $stmt->execute([$start, $end]);
            } else {
                $stmt = $db->query("SELECT id, title, DATE_FORMAT(event_date, '%Y-%m-%d') AS date, event_time AS time, description AS `desc`, color, created_at, updated_at FROM calendar_events ORDER BY event_date ASC, event_time ASC, id ASC LIMIT 1000");
            }
            echo json_encode(['code' => 0, 'data' => ['list' => $stmt->fetchAll(PDO::FETCH_ASSOC)]], JSON_UNESCAPED_UNICODE);
            exit;
        }

        if ($action === 'create') {
            $payload = json_decode(file_get_contents('php://input'), true) ?: [];
            $event = appNormalizeCalendarEvent($payload);
            $stmt = $db->prepare("INSERT INTO calendar_events (title, event_date, event_time, description, color) VALUES (?,?,?,?,?)");
            $stmt->execute([$event['title'], $event['date'], $event['time'], $event['desc'], $event['color']]);
            $event['id'] = (int)$db->lastInsertId();
            echo json_encode(['code' => 0, 'data' => $event, 'msg' => '事件已保存'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        if ($action === 'delete') {
            $payload = json_decode(file_get_contents('php://input'), true) ?: [];
            $id = (int)($payload['id'] ?? 0);
            if ($id <= 0) {
                throw new Exception('事件 ID 无效');
            }
            $stmt = $db->prepare("DELETE FROM calendar_events WHERE id = ?");
            $stmt->execute([$id]);
            echo json_encode(['code' => 0, 'msg' => '事件已删除'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        if ($action === 'clear') {
            $db->exec("DELETE FROM calendar_events");
            echo json_encode(['code' => 0, 'msg' => '日历事件已清空'], JSON_UNESCAPED_UNICODE);
            exit;
        }

        if ($action === 'replace') {
            $payload = json_decode(file_get_contents('php://input'), true) ?: [];
            $events = $payload['events'] ?? null;
            if (!is_array($events)) {
                throw new Exception('事件数据格式错误');
            }

            $normalized = [];
            foreach ($events as $event) {
                if (!is_array($event)) {
                    continue;
                }
                $normalized[] = appNormalizeCalendarEvent($event);
            }

            $db->beginTransaction();
            $db->exec("DELETE FROM calendar_events");
            $stmt = $db->prepare("INSERT INTO calendar_events (title, event_date, event_time, description, color) VALUES (?,?,?,?,?)");
            foreach ($normalized as $event) {
                $stmt->execute([$event['title'], $event['date'], $event['time'], $event['desc'], $event['color']]);
            }
            $db->commit();
            echo json_encode(['code' => 0, 'data' => ['count' => count($normalized)], 'msg' => '日历事件已导入'], JSON_UNESCAPED_UNICODE);
            exit;
        }
    } catch (Exception $e) {
        if (isset($db) && $db instanceof PDO && $db->inTransaction()) {
            $db->rollBack();
        }
        echo json_encode(['code' => 1, 'msg' => $e->getMessage()], JSON_UNESCAPED_UNICODE);
        exit;
    }

    echo json_encode(['code' => 1, 'msg' => '未知日历操作'], JSON_UNESCAPED_UNICODE);
    exit;
}
/* ========== /云端日历 API ========== */

/* ========== 设置页 API ========== */
if (isset($_GET['settings_action'])) {
    appRequireAuthJson();
    $action = $_GET['settings_action'];
    $getPdo = function() {
        $pdo = appCreatePdo();
        appEnsureUploadAccountsRemark($pdo);
        return $pdo;
    };

    try {
        // 备注列表：从 upload_accounts 按 remark 分组统计
        if ($action === 'list_remarks') {
            $db = $getPdo();
            $kw = isset($_GET['kw']) ? '%'.$_GET['kw'].'%' : '%';
            $rows = $db->prepare("SELECT remark, COUNT(*) as cnt FROM upload_accounts WHERE remark != '' AND remark LIKE ? GROUP BY remark ORDER BY cnt DESC LIMIT 200");
            $rows->execute([$kw]);
            echo json_encode(['code'=>0,'data'=>$rows->fetchAll(PDO::FETCH_ASSOC)]); exit;
        }

        // 账号列表：以 upload_accounts 为主表，LEFT JOIN upload_account_auth 取 id/created_at
        if ($action === 'list_accounts') {
            $db = $getPdo();
            $kw = isset($_GET['kw']) && trim($_GET['kw']) !== '' ? trim($_GET['kw']) : null;
            $page = max(1, intval($_GET['page'] ?? 1));
            $limit = 20; $offset = ($page-1)*$limit;
            $join = 'FROM upload_accounts u LEFT JOIN upload_account_auth a ON a.account = u.account';
            if ($kw !== null) {
                $likePat = '%' . $kw . '%';
                $where = "WHERE (u.remark LIKE ? OR u.account LIKE ?)";
                $params = [$likePat, $likePat];
            } else {
                $where = '';
                $params = [];
            }
            $total = $db->prepare("SELECT COUNT(*) $join $where");
            $total->execute($params);
            $cnt = $total->fetchColumn();
            $pages = $cnt > 0 ? ceil($cnt/$limit) : 1;
            $stmt = $db->prepare("SELECT u.id, u.account, u.remark, IFNULL(a.created_at, u.created_at) as created_at $join $where ORDER BY u.id DESC LIMIT $limit OFFSET $offset");
            $stmt->execute($params);
            echo json_encode(['code'=>0,'data'=>['list'=>$stmt->fetchAll(PDO::FETCH_ASSOC),'total'=>(int)$cnt,'page'=>$page,'pages'=>(int)$pages]]); exit;
        }

        if ($action === 'add_remark') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            // 将 upload_accounts 中对应备注改名
            $db->prepare("UPDATE upload_accounts SET remark=? WHERE remark=?")
               ->execute([$data['new_remark'] ?? '', $data['old_remark'] ?? '___NOMATCH___']);
            echo json_encode(['code'=>0,'msg'=>'操作成功']); exit;
        }

        if ($action === 'update_remark') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            $db->prepare("UPDATE upload_accounts SET remark=? WHERE remark=?")
               ->execute([$data['new_remark'], $data['old_remark']]);
            echo json_encode(['code'=>0,'msg'=>'备注已更新']); exit;
        }

        if ($action === 'delete_remark') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            $db->prepare("UPDATE upload_accounts SET remark='' WHERE remark=?")->execute([$data['remark']]);
            echo json_encode(['code'=>0,'msg'=>'已清空该备注']); exit;
        }

        if ($action === 'update_account_remark') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            // 直接更新 upload_accounts 的 remark
            $db->prepare("UPDATE upload_accounts SET remark=? WHERE id=?")
               ->execute([$data['remark'] ?? '', $data['id']]);
            echo json_encode(['code'=>0,'msg'=>'备注已更新']); exit;
        }

        if ($action === 'delete_account') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            // 查出账号名，两张表都删
            $acc = $db->prepare("SELECT account FROM upload_accounts WHERE id=?");
            $acc->execute([$data['id']]);
            $row = $acc->fetch(PDO::FETCH_ASSOC);
            $db->prepare("DELETE FROM upload_accounts WHERE id=?")->execute([$data['id']]);
            if ($row) $db->prepare("DELETE FROM upload_account_auth WHERE account=?")->execute([$row['account']]);
            echo json_encode(['code'=>0,'msg'=>'已删除']); exit;
        }

        if ($action === 'add_account') {
            $data = json_decode(file_get_contents('php://input'), true) ?: [];
            $db = $getPdo();
            $db->prepare("INSERT INTO upload_account_auth (account, password, created_at) VALUES (?,?,NOW())")
               ->execute([trim($data['account']), trim($data['password'] ?? '')]);
            if (!empty($data['remark'])) {
                $db->prepare("INSERT INTO upload_accounts (account, remark) VALUES (?,?) ON DUPLICATE KEY UPDATE remark=VALUES(remark)")
                   ->execute([trim($data['account']), trim($data['remark'])]);
            }
            echo json_encode(['code'=>0,'msg'=>'添加成功']); exit;
        }

    } catch(Exception $e) {
        echo json_encode(['code'=>1,'msg'=>$e->getMessage()]); exit;
    }

    echo json_encode(['code'=>1,'msg'=>'未知操作']); exit;
}
/* ========== /设置页 API ========== */
if (!appIsAuthenticated()) {
    appRenderLoginPage();
}
?>
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>格界管理后台</title>
  <link rel="icon" type="image/png" href="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>">
  <link rel="apple-touch-icon" href="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>">
  <style>
    :root {
      --bg: #f4efe8;
      --bg-soft: #fbf8f4;
      --panel: rgba(255,255,255,.84);
      --panel-strong: #fff;
      --text: #2c2622;
      --muted: #82786f;
      --line: #eadfd2;
      --orange: #e8a15f;
      --orange-2: #f6d6b5;
      --green: #6da98d;
      --red: #de7d67;
      --shadow: 0 18px 42px rgba(93, 71, 46, .08);
      --shadow-soft: 0 12px 28px rgba(93, 71, 46, .06);
      --radius: 24px;
    }
    /* ===== 主题变量 ===== */
    [data-theme="ocean"] {
      --bg: #e8f0f7; --bg-soft: #f2f7fc; --text: #1a2b3c; --muted: #6a85a0;
      --line: #ccdcec; --orange: #3a8fd4; --orange-2: #aed0ed; --green: #4ca87a;
      --red: #e05c5c; --shadow: 0 18px 42px rgba(30,70,120,.08);
      --shadow-soft: 0 12px 28px rgba(30,70,120,.06);
    }
    [data-theme="forest"] {
      --bg: #eaf2ea; --bg-soft: #f5faf5; --text: #1c2d1e; --muted: #6a8c6e;
      --line: #cce0cc; --orange: #4aaa65; --orange-2: #b2ddb9; --green: #3d9e5a;
      --red: #d46b5a; --shadow: 0 18px 42px rgba(30,80,30,.08);
      --shadow-soft: 0 12px 28px rgba(30,80,30,.06);
    }
    [data-theme="violet"] {
      --bg: #ede8f5; --bg-soft: #f6f3fb; --text: #231c33; --muted: #7a6e96;
      --line: #d8ceee; --orange: #8b5cf6; --orange-2: #c9b8f5; --green: #6da98d;
      --red: #de7d67; --shadow: 0 18px 42px rgba(70,40,120,.08);
      --shadow-soft: 0 12px 28px rgba(70,40,120,.06);
    }
    [data-theme="rose"] {
      --bg: #f5e8ed; --bg-soft: #fcf3f6; --text: #33161e; --muted: #9a6a76;
      --line: #eeccd7; --orange: #d4607a; --orange-2: #f0b0c0; --green: #6da98d;
      --red: #c0392b; --shadow: 0 18px 42px rgba(120,40,60,.08);
      --shadow-soft: 0 12px 28px rgba(120,40,60,.06);
    }
    [data-theme="dark"] {
      --bg: #1a1a2e; --bg-soft: #16213e; --panel: rgba(30,30,50,.92); --panel-strong: #252540;
      --text: #e8e8f0; --muted: #8888aa; --line: #33335a;
      --orange: #f5a623; --orange-2: #7a4e10; --green: #4caf80; --red: #e06060;
      --shadow: 0 18px 42px rgba(0,0,0,.3); --shadow-soft: 0 12px 28px rgba(0,0,0,.25);
    }
    [data-theme="slate"] {
      --bg: #eaeef2; --bg-soft: #f4f6f9; --text: #1e2a35; --muted: #7a8a96;
      --line: #d0dae2; --orange: #4a7fa8; --orange-2: #b0cde0; --green: #5a9a7a;
      --red: #c8604a; --shadow: 0 18px 42px rgba(30,50,70,.08);
      --shadow-soft: 0 12px 28px rgba(30,50,70,.06);
    }
    [data-theme="white"] {
      --bg: #f0f0f0; --bg-soft: #ffffff; --panel: rgba(255,255,255,.95); --panel-strong: #fff;
      --text: #1a1a1a; --muted: #888888; --line: #e0e0e0;
      --orange: #e8a15f; --orange-2: #f6d6b5; --green: #4caf7a; --red: #e05c5c;
      --shadow: 0 18px 42px rgba(0,0,0,.07); --shadow-soft: 0 12px 28px rgba(0,0,0,.05);
    }
    [data-theme="night"] {
      --bg: #0d1117; --bg-soft: #0a0e14; --panel: rgba(22,28,38,.95); --panel-strong: #1c2333;
      --text: #c9d1d9; --muted: #6e7681; --line: #30363d;
      --orange: #58a6ff; --orange-2: #1f3a5c; --green: #3fb950; --red: #f85149;
      --shadow: 0 18px 42px rgba(0,0,0,.5); --shadow-soft: 0 12px 28px rgba(0,0,0,.4);
    }

    /* ===== 主题切换菜单 ===== */
    .theme-menu {
      position:absolute; right:0; top:56px; width:220px;
      background: var(--panel-strong,#fff); border:1px solid var(--line);
      border-radius:20px; box-shadow: var(--shadow); padding:14px; display:none; z-index:99;
    }
    .theme-menu.open { display:block; }
    .theme-menu-title { font-size:12px; font-weight:700; color:var(--muted); letter-spacing:.06em; margin-bottom:10px; }
    .theme-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:8px; }
    .theme-dot {
      display:flex; flex-direction:column; align-items:center; gap:5px; cursor:pointer;
      padding:6px 4px; border-radius:12px; transition:.15s;
    }
    .theme-dot:hover { background: rgba(0,0,0,.05); }
    .theme-dot.active { background: rgba(0,0,0,.08); }
    .theme-dot-circle {
      width:32px; height:32px; border-radius:50%; border:2px solid rgba(0,0,0,.1);
      box-shadow:0 2px 6px rgba(0,0,0,.15);
    }
    .theme-dot span { font-size:11px; color:var(--muted); font-weight:600; text-align:center; }
    .avatar-btn { cursor:pointer; position:relative; }

    * { box-sizing: border-box; }
    html, body { height: 100%; }
    body {
      margin: 0;
      font-family: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--text);
      background: linear-gradient(180deg, var(--bg-soft), var(--bg));
    }
    .app { display: grid; grid-template-columns: 286px minmax(0, 1fr); min-height: 100vh; }
    .sidebar {
      padding: 24px 18px; border-right: 1px solid rgba(234,223,210,.8);
      background: rgba(255,255,255,.18); backdrop-filter: blur(14px);
      overflow: hidden; transition: width .22s ease, padding .22s ease;
      height: 100vh; position: sticky; top: 0; display: flex; flex-direction: column;
    }
    .sidebar.collapsed { width: 88px; padding-left: 12px; padding-right: 12px; }
    .sidebar.collapsed .nav-text,
    .sidebar.collapsed .brand-text,
    .sidebar.collapsed .nav-group-title,
    .sidebar.collapsed .sidebar-footer,
    .sidebar.collapsed .brand-sub { display: none; }
    .sidebar-head { display:flex; align-items:center; justify-content:space-between; gap:10px; }
    .brand { display:flex; align-items:center; gap:12px; font-size:20px; font-weight:800; margin-bottom: 22px; }
    .brand-mark {
      width:46px; height:46px; border-radius:15px; overflow:hidden;
      background: rgba(255,255,255,.86); box-shadow: var(--shadow); border:1px solid rgba(234,223,210,.95);
    }
    .brand-mark img {
      width:100%; height:100%; object-fit:cover; display:block;
    }
    .brand-title { line-height: 1.05; }
    .brand-sub { display:block; color: var(--muted); font-size: 12px; font-weight: 600; margin-top: 3px; }
    .collapse-btn {
      width:38px; height:38px; border-radius:12px; border:1px solid var(--line);
      background: rgba(255,255,255,.8); display:grid; place-items:center; cursor:pointer; box-shadow: var(--shadow-soft);
    }
    .nav-group-title {
      margin: 22px 10px 10px; font-size: 11px; letter-spacing: .15em; text-transform: uppercase;
      color:#ab9c8f; font-weight:800;
    }
    .nav-item {
      display:flex; align-items:center; gap:14px; padding:14px 14px; margin-bottom:8px;
      border-radius:16px; color:var(--muted); text-decoration:none; transition:.18s ease; font-weight:700; position:relative;
    }
    .nav-item:hover, .nav-item.active {
      color:var(--text); background: rgba(255,255,255,.82); box-shadow: inset 0 0 0 1px rgba(232,161,95,.18);
    }
    .nav-item.active::before {
      content:''; position:absolute; left:-2px; top:10px; bottom:10px; width:4px; border-radius:999px;
      background: linear-gradient(180deg, var(--orange), #f3c08f);
    }
    .nav-item.disabled { opacity:.55; }
    .nav-ico { width:22px; height:22px; display:grid; place-items:center; color:currentColor; flex:0 0 auto; }
    .sidebar-footer {
      margin-top:auto; padding:14px; border-radius:18px; background: linear-gradient(180deg, rgba(255,255,255,.84), rgba(255,255,255,.66));
      border:1px solid var(--line);
    }
    .footer-row { display:flex; align-items:center; gap:10px; }
    .footer-dot { width:10px; height:10px; border-radius:50%; background: var(--green); }
    .content {
      padding: 26px 28px 26px 22px;
      min-width: 0;
      display: flex;
      flex-direction: column;
      align-items: center;
    }
    .topbar,
    .dashboard,
    .section-wide {
      width: min(100%, 1320px);
    }
    .topbar { display:grid; grid-template-columns: 1fr minmax(280px, 520px) auto; gap:16px; align-items:center; margin-bottom: 20px; }
    .title h1 { margin:0; font-size:30px; letter-spacing: -.02em; }
    .title p { margin:6px 0 0; color:var(--muted); }
    .search {
      display:flex; align-items:center; gap:12px; padding:14px 18px; border-radius:18px; background: rgba(255,255,255,.78);
      border:1px solid rgba(234,223,210,.9); box-shadow: var(--shadow);
    }
    .search input { width:100%; border:0; outline:none; background:transparent; font-size:15px; color:var(--text); }
    .top-actions { display:flex; align-items:center; gap:12px; position:relative; }
    .logout-btn{
      height:48px;
      padding:0 16px;
      border-radius:16px;
      border:1px solid rgba(234,223,210,.9);
      background:rgba(255,255,255,.82);
      box-shadow:var(--shadow);
      color:var(--text);
      font-size:13px;
      font-weight:800;
      cursor:pointer;
    }
    .icon-btn, .avatar, .notif-btn {
      width:48px; height:48px; border-radius:16px; background: rgba(255,255,255,.8); border:1px solid rgba(234,223,210,.9);
      box-shadow: var(--shadow); display:grid; place-items:center;
    }
    .notif-btn { cursor:pointer; position:relative; }
    .notif-badge { position:absolute; top:8px; right:8px; width:9px; height:9px; border-radius:50%; background: var(--red); box-shadow: 0 0 0 3px rgba(255,255,255,.95); }
    .avatar { font-weight:800; color:var(--orange); }
    .menu {
      position:absolute; right:60px; top:56px; width: 332px; background: rgba(255,255,255,.98); border:1px solid var(--line);
      border-radius:20px; box-shadow: var(--shadow); padding: 10px; display:none; z-index:20;
    }
    .menu.show { display:block; }
    .menu h3 { margin: 6px 10px 10px; font-size: 15px; }
    .menu-item { display:flex; gap:12px; padding:12px 10px; border-radius:14px; cursor:pointer; }
    .menu-item:hover { background:#faf6f1; }
    .menu-item .bullet {
      width:34px; height:34px; border-radius:12px; flex:0 0 auto; display:grid; place-items:center;
      background:#fff3e7; color:#c87a33; border:1px solid #f0d3b4;
    }
    .menu-item p { margin: 3px 0 0; color:var(--muted); font-size:13px; line-height:1.45; }
    .dashboard {
      display:grid;
      grid-template-columns: minmax(0, 1.2fr) minmax(0, .95fr);
      gap:22px;
      padding: 0 10px;
    }
    .dashboard-section { scroll-margin-top: 18px; }
    .dashboard, .panel { width: 100%; }
    .panel {
      background: var(--panel); border:1px solid rgba(234,223,210,.92); border-radius: var(--radius); box-shadow: var(--shadow); backdrop-filter: blur(12px);
      overflow:hidden; width: 100%; margin: 0 auto;
    }
    .panel-head { display:flex; align-items:flex-start; justify-content:space-between; gap:12px; padding:20px 20px 0; }
    .section-title { font-size:18px; font-weight:800; margin:0; }
    .section-sub { color:var(--muted); font-size:13px; margin-top:5px; line-height:1.45; }
    .pill { display:inline-flex; align-items:center; justify-content:center; gap:6px; padding:8px 12px; border-radius:999px; font-size:12px; font-weight:800; background:#f7efe5; color:#b37234; border:1px solid rgba(232,161,95,.18); white-space:nowrap; }
    .pill.high { background:#fde8db; color:#cb6a25; }
    .pill.ok { background:#eef7f2; color:#4f9376; }
    .sub-grid {
      display:grid;
      grid-template-columns: 1.1fr .9fr;
      gap: 22px;
      width: 100%;
      padding: 0 10px;
    }
    .projects, .tasks, .messages { padding:18px 20px 20px; }
    .project-card {
      border-radius:22px; overflow:hidden; background: var(--panel-strong); border:1px solid #efe5d8; margin-bottom:14px; cursor:pointer; transition:transform .16s ease, box-shadow .16s ease;
    }
    .project-card:hover { transform:translateY(-1px); box-shadow:0 14px 30px rgba(95,70,44,.07); }
    .project-cover { height: 94px; background: linear-gradient(135deg, #f7dfbf 0%, #f9f4ee 100%); position:relative; }
    .project-cover::after { content:''; position:absolute; inset:0; background: var(--cover); opacity:.94; }
    .project-badge {
      position:absolute; right:14px; top:14px; z-index:2; width:42px; height:42px; border-radius:14px; background:rgba(255,255,255,.92);
      border:1px solid rgba(255,255,255,.75); display:grid; place-items:center; box-shadow:0 10px 18px rgba(67,48,29,.08);
    }
    .project-body { padding:16px; }
    .project-top { display:flex; justify-content:space-between; gap:12px; align-items:flex-start; }
    .project-name { font-size:17px; font-weight:800; margin-bottom:5px; letter-spacing:-.01em; }
    .meta { color:var(--muted); font-size:13px; }
    .row { display:flex; align-items:center; justify-content:space-between; gap:10px; margin-top:12px; }
    .progress-label { font-size:12px; color:var(--muted); }
    .progress-bar { height:10px; margin-top:8px; border-radius:999px; background:#f1e7db; overflow:hidden; position:relative; }
    .progress-bar span { display:block; height:100%; border-radius:inherit; background: linear-gradient(90deg, var(--orange), #f3c28f); position:relative; }
    .progress-bar span::after { content:''; position:absolute; right:0; top:0; bottom:0; width:22px; background:linear-gradient(90deg, transparent, rgba(255,255,255,.55)); }
    .tag-wrap { display:flex; flex-wrap:wrap; gap:8px; margin-top:14px; }
    .tag { padding:7px 10px; border-radius:999px; font-size:12px; background:#faf7f3; border:1px solid #efe4d7; color:#786b60; }
    .avatars { display:flex; align-items:center; }
    .avatars span { width:32px; height:32px; border-radius:50%; display:grid; place-items:center; font-size:11px; font-weight:800; color:#9b6b3f; background:#fff; border:2px solid #efe0d1; margin-left:-8px; }
    .avatars span:first-child { margin-left:0; }
    .control-row { display:flex; align-items:center; gap:10px; flex-wrap:wrap; }
    .control { height:38px; border-radius:12px; border:1px solid #eadfd2; background: rgba(255,255,255,.88); color:var(--text); padding:0 12px; outline:none; }
    .control.small { min-width: 128px; }
    .stats { display:grid; grid-template-columns: repeat(3, 1fr); gap:12px; margin-top: 14px; }
    .stat-card { padding:16px; border-radius:18px; background:#fff; border:1px solid #efe5d8; }
    .stat-card strong { display:block; font-size:24px; margin-top:6px; }
    .chart-box { padding: 16px 18px 18px; border-radius: 20px; background: linear-gradient(180deg, #fff, #f8f2eb); border: 1px solid #efe0d0; }
    .chart-title { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom: 12px; }
    .chart { width: 100%; height: 220px; }
    .chart-mini { width:100%; height:120px; }
    .chart-axis { display:flex; justify-content:space-between; font-size:12px; color:var(--muted); margin-top: 4px; }
    .table-toolbar { display:flex; align-items:center; justify-content:space-between; gap:12px; margin: 16px 0 10px; }
    .table {
      width:100%; border-collapse: collapse; overflow:hidden; background:#fff; border:1px solid #efe5d8; border-radius:18px; display:block; max-height: 360px; overflow:auto;
    }
    .table thead, .table tbody { display:block; }
    .table tr { display:grid; grid-template-columns: minmax(160px, 1.7fr) minmax(180px, 1.1fr) 88px 120px 88px 110px; gap:0; align-items:center; }
    .table th, .table td { padding:12px 14px; border-bottom:1px solid #f0e6d8; text-align:left; font-size:13px; vertical-align:middle; }
    .table th { color:#7f7368; position: sticky; top:0; background:#fff; z-index:1; white-space: nowrap; }
    .table td:nth-child(1), .table th:nth-child(1) { min-width: 160px; }
    .table td:nth-child(2), .table th:nth-child(2) { min-width: 180px; }
    .table td:nth-child(1), .table td:nth-child(2) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .table tbody tr { cursor:pointer; }
    .table tbody tr:hover { background:#fbf7f2; }
    .pagination { display:flex; align-items:center; gap:8px; justify-content:flex-end; margin-top: 10px; }
    .page-btn { height:34px; padding:0 12px; border-radius:10px; border:1px solid #eadfd2; background:#fff; cursor:pointer; }
    .page-btn:disabled { opacity:.45; cursor:not-allowed; }
    .timeline { display:grid; gap:10px; }
    .timeline-item { display:flex; gap:12px; align-items:flex-start; padding:12px 0; border-bottom:1px solid #efe5d8; }
    .timeline-item:last-child { border-bottom:0; }
    .timeline-dot { width:12px; height:12px; border-radius:50%; background:var(--orange); margin-top:4px; box-shadow: 0 0 0 4px rgba(232,161,95,.15); }
    .timeline-item strong { display:block; margin-bottom: 3px; }
    .timeline-item p { margin:0; color:var(--muted); font-size:13px; line-height:1.5; }
    .calendar { display:grid; gap:12px; }
    .calendar-grid { display:grid; grid-template-columns: repeat(7, 1fr); gap:8px; margin-top: 12px; }
    .cal-cell {
      min-height:68px; padding:10px 10px 8px; border-radius:16px; background:#fff; border:1px solid #efe5d8; display:flex; flex-direction:column; justify-content:space-between; cursor:pointer; transition:.16s ease;
    }
    .cal-cell:hover { transform: translateY(-1px); }
    .cal-cell small { color:var(--muted); }
    .cal-cell strong { font-size:15px; }
    .cal-cell.active { background:#fce8d7; border-color:#f0c399; }
    .cal-cell.selected { box-shadow: inset 0 0 0 2px rgba(232,161,95,.7); }
    .dot { width:8px; height:8px; border-radius:50%; background:var(--orange); align-self:flex-end; }
    .message-list { display:grid; gap:0; }
    .message {
      display:flex; gap:12px; padding:14px 0; border-bottom:1px solid #efe5d8; cursor:pointer;
    }
    .message:last-child { border-bottom:0; }
    .message .avatar { width:42px; height:42px; border-radius:14px; box-shadow:none; flex:0 0 auto; }
    .message strong { display:block; margin-bottom:3px; }
    .message p { margin:0; color:var(--muted); font-size:13px; line-height:1.5; }
    .message time { color:#b08a68; font-size:12px; white-space:nowrap; }
    .empty { padding:16px; border:1px dashed #e7d9ca; border-radius:16px; color:var(--muted); background:rgba(255,255,255,.72); text-align:center; }
    .modal-backdrop { position:fixed; inset:0; background:rgba(36,28,22,.34); backdrop-filter:blur(6px); display:none; align-items:center; justify-content:center; z-index:60; padding:18px; }
    .modal-backdrop.show { display:flex; }
    .modal { width:min(920px, 100%); max-height:86vh; overflow:auto; background:rgba(255,255,255,.97); border:1px solid rgba(234,223,210,.95); border-radius:26px; box-shadow:0 30px 70px rgba(53,36,21,.18); }
    .modal-head { display:flex; justify-content:space-between; align-items:flex-start; gap:12px; padding:22px 22px 16px; border-bottom:1px solid #efe5d8; }
    .modal-body { padding:20px 22px 24px; }
    .modal-grid { display:grid; grid-template-columns: 1.05fr .95fr; gap:18px; }
    .detail-box, .mini-box { border:1px solid #efe5d8; border-radius:18px; background:#fff; padding:16px; }
    .mini-list { display:grid; gap:10px; }
    .mini-item { display:flex; justify-content:space-between; gap:10px; color:var(--muted); font-size:13px; }
    .close-btn { width:38px; height:38px; border-radius:12px; border:1px solid var(--line); background:#fff; cursor:pointer; }
    .toast { position:fixed; right:18px; bottom:18px; z-index:70; background:rgba(32,26,22,.92); color:#fff; padding:12px 14px; border-radius:14px; box-shadow:var(--shadow); opacity:0; transform:translateY(10px); transition:.2s ease; }
    .toast.show { opacity:1; transform:translateY(0); }
    @media (max-width: 1280px) {
      .dashboard, .sub-grid, .modal-grid { grid-template-columns: 1fr; }
      .topbar, .dashboard, .section-wide { width: 100%; }
      .content { align-items: stretch; }
    }
    @media (max-width: 940px) {
      .app { grid-template-columns:1fr; }
      .sidebar { border-right:0; border-bottom:1px solid rgba(234,223,210,.8); }
      .topbar { grid-template-columns:1fr; }
      .content { padding: 18px; }
      .table tr { grid-template-columns: 74px 1.3fr 1fr 82px 84px 96px; }
    }
  </style>
</head>
<body>
  <div class="app">
    <aside id="sidebar" class="sidebar">
      <div class="sidebar-head">
        <div class="brand">
          <div class="brand-mark">
            <img src="<?= htmlspecialchars(APP_ICON_FILE, ENT_QUOTES, 'UTF-8') ?>" alt="格界图标">
          </div>
          <div class="brand-title">
            <span class="brand-text">格界数据</span>
            <span class="brand-sub">格界管理后台</span>
          </div>
        </div>
        <button id="collapseBtn" class="collapse-btn" aria-label="切换侧边栏">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"></path></svg>
        </button>
      </div>
      <div class="nav-group-title">主导航</div>
      <nav>
        <a class="nav-item active" href="#section-overview" data-section="section-overview"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12l9-8 9 8"></path><path d="M5 10v10h14V10"></path></svg></span><span class="nav-text">总览</span></a>
        <a class="nav-item" href="#section-projects" data-section="section-projects"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="7" height="7" rx="2"></rect><rect x="13" y="4" width="7" height="7" rx="2"></rect><rect x="4" y="13" width="7" height="7" rx="2"></rect><rect x="13" y="13" width="7" height="7" rx="2"></rect></svg></span><span class="nav-text">项目管理</span></a>
        <a class="nav-item" href="#section-tasks" data-section="section-tasks"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M9 11l3 3L22 4"></path><path d="M21 12a9 9 0 1 1-9-9"></path></svg></span><span class="nav-text">我的任务</span></a>
        <a class="nav-item" href="#section-calendar" data-section="section-calendar"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="16" rx="3"></rect><path d="M16 3v4M8 3v4M3 11h18"></path></svg></span><span class="nav-text">日历</span></a>
        <a class="nav-item" href="#section-time" data-section="section-time"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8v4l3 3"></path><circle cx="12" cy="12" r="9"></circle></svg></span><span class="nav-text">时间管理</span></a>
        <a class="nav-item" href="#section-reports" data-section="section-reports"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"></path><path d="M7 15l4-4 3 3 6-8"></path></svg></span><span class="nav-text">报表</span></a>
        <a class="nav-item" href="#section-settings" data-section="section-settings"><span class="nav-ico"><svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06A1.65 1.65 0 0 0 15 19.4a1.65 1.65 0 0 0-1 .6 1.65 1.65 0 0 0-.33 1.82V22a2 2 0 0 1-4 0v-.08a1.65 1.65 0 0 0-.33-1.82A1.65 1.65 0 0 0 8 19.4a1.65 1.65 0 0 0-1-.6 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 2.6 15a1.65 1.65 0 0 0-.6-1 1.65 1.65 0 0 0-1.82-.33H0a2 2 0 0 1 0-4h.08a1.65 1.65 0 0 0 1.82-.33A1.65 1.65 0 0 0 2.6 8c0-.42.16-.81.33-1a1.65 1.65 0 0 0-.06-1.82l-.06-.06A2 2 0 0 1 5.64 2.3l.06.06A1.65 1.65 0 0 0 7 2.6c.42 0 .81-.16 1-.33A1.65 1.65 0 0 0 8.33.45V0a2 2 0 0 1 4 0v.08a1.65 1.65 0 0 0 .33 1.82c.19.17.58.33 1 .33s.81-.16 1-.33a1.65 1.65 0 0 0 1.82-.33l.06-.06A2 2 0 0 1 22 5.64l-.06.06A1.65 1.65 0 0 0 21.4 7c0 .42.16.81.33 1a1.65 1.65 0 0 0 1.82.33H24a2 2 0 0 1 0 4h-.08a1.65 1.65 0 0 0-1.82.33 1.65 1.65 0 0 0-.7 1.34z"></path></svg></span><span class="nav-text">设置</span></a>
      </nav>
      <div class="sidebar-footer">
        <div class="footer-row">
          <div class="footer-dot"></div>
          <div>
            <div style="font-weight:800;">系统在线</div>
            <div class="meta" id="liveStatusBadge">实时在线</div>
          </div>
        </div>
      </div>
    </aside>

    <main class="content">
      <div class="topbar">
        <div class="title">
          <h1 style="display:none;">Dashboard</h1>
          <p>正在从接口加载项目、账号、趋势和排行数据……</p>
        </div>
        <div class="search">
          <span style="color:var(--muted);">⌕</span>
          <input id="searchInput" type="text" placeholder="搜索账号、备注、消息……">
        </div>
        <div class="top-actions">
          <button class="logout-btn" onclick="logoutApp()"><?= htmlspecialchars(appAuthUser() ?? APP_AUTH_USERNAME, ENT_QUOTES, 'UTF-8') ?> · 退出</button>
          <div style="position:relative;">
            <button id="notifBtn" class="notif-btn" aria-label="通知">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M15 17h5l-1.4-1.4A2 2 0 0 1 18 14.2V11a6 6 0 0 0-12 0v3.2a2 2 0 0 1-.6 1.4L4 17h5"></path><path d="M10 17a2 2 0 0 0 4 0"></path></svg>
              <span class="notif-badge"></span>
            </button>
            <div id="notifMenu" class="menu"></div>
          </div>
          <div class="avatar-btn" id="themeBtn">
            <div class="avatar" id="themeAvatar">AM</div>
            <div class="theme-menu" id="themeMenu">
              <div class="theme-menu-title">🎨 切换主题</div>
              <div class="theme-grid">
                <div class="theme-dot active" data-theme="default" onclick="setTheme('default',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#e8a15f,#f6d6b5);"></div>
                  <span>暖橙</span>
                </div>
                <div class="theme-dot" data-theme="ocean" onclick="setTheme('ocean',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#3a8fd4,#aed0ed);"></div>
                  <span>海蓝</span>
                </div>
                <div class="theme-dot" data-theme="forest" onclick="setTheme('forest',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#4aaa65,#b2ddb9);"></div>
                  <span>森绿</span>
                </div>
                <div class="theme-dot" data-theme="violet" onclick="setTheme('violet',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#8b5cf6,#c9b8f5);"></div>
                  <span>紫调</span>
                </div>
                <div class="theme-dot" data-theme="rose" onclick="setTheme('rose',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#d4607a,#f0b0c0);"></div>
                  <span>玫瑰</span>
                </div>
                <div class="theme-dot" data-theme="dark" onclick="setTheme('dark',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#1a1a2e,#f5a623);"></div>
                  <span>暗夜</span>
                </div>
                <div class="theme-dot" data-theme="slate" onclick="setTheme('slate',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#4a7fa8,#b0cde0);"></div>
                  <span>石板</span>
                </div>
                <div class="theme-dot" data-theme="white" onclick="setTheme('white',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#ffffff,#e0e0e0);border:1px solid #ccc;"></div>
                  <span>纯白</span>
                </div>
                <div class="theme-dot" data-theme="night" onclick="setTheme('night',this)">
                  <div class="theme-dot-circle" style="background:linear-gradient(135deg,#0d1117,#58a6ff);"></div>
                  <span>夜间</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <section id="section-overview" class="dashboard dashboard-section">
        <div class="panel">
          <div class="panel-head">
            <div>
              <h2 class="section-title">总览与统计</h2>
              <div class="section-sub">包含今日总上传、活跃账号、环比变化与近 14 日趋势。</div>
            </div>
            <span id="dateBadge" class="pill"><?= htmlspecialchars($today, ENT_QUOTES, 'UTF-8') ?></span>
          </div>
          <div class="projects">
            <div class="stats">
              <div class="stat-card"><span class="meta">今日总上传</span><strong id="statTotal">0</strong><div class="meta" id="statDelta">较昨日 0</div></div>
              <div class="stat-card"><span class="meta">活跃账号</span><strong id="statActive">0</strong><div class="meta">当天有记录的账号数</div></div>
              <div class="stat-card"><span class="meta">平均单号</span><strong id="statAvg">0</strong><div class="meta">今日每账号平均上传</div></div>
            </div>
            <div class="chart-box" style="margin-top:14px;">
              <div class="chart-title">
                <div>
                  <h3 class="section-title" style="font-size:16px;">近 14 日趋势</h3>
                  <div class="section-sub">按天统计成功数，支持日期联动刷新</div>
                </div>
                <div class="control-row">
                  <input id="dateInput" class="control" type="date" value="<?= htmlspecialchars($today, ENT_QUOTES, 'UTF-8') ?>">
                  <button id="reloadBtn" class="page-btn">刷新</button>
                </div>
              </div>
              <svg id="trendChart" class="chart" viewBox="0 0 860 220" preserveAspectRatio="none"></svg>
            </div>
            <div class="sub-grid" style="margin-top: 14px;">
              <div class="chart-box">
                <div class="chart-title">
                  <div>
                    <h3 class="section-title" style="font-size:16px;">24 小时分布</h3>
                    <div class="section-sub">查看今日时间段活跃情况</div>
                  </div>
                </div>
                <svg id="hourChart" class="chart-mini" viewBox="0 0 860 120" preserveAspectRatio="none"></svg>
              </div>
              <div class="chart-box">
                <div class="chart-title">
                  <div>
                    <h3 class="section-title" style="font-size:16px;">今日 Top 账号</h3>
                    <div class="section-sub">自动取接口中的排行榜第一名</div>
                  </div>
                </div>
                <div id="topAccountBox" class="detail-box" style="margin-top:4px;">
                  <div class="meta">暂无数据</div>
                </div>
              </div>
            </div>

            <!-- 今日总量 + 备注排名 -->
            <div class="chart-box" style="margin-top:14px;">
              <div class="chart-title">
                <div>
                  <h3 class="section-title" style="font-size:16px;">今日总量 &amp; 备注排名</h3>
                  <div class="section-sub">当天活跃账号按备注分组，显示各备注上传总数排行</div>
                </div>
                <div id="todayTotalBadge" class="pill high" style="font-size:16px;padding:10px 18px;">总计 0</div>
              </div>
              <div id="remarkRankList" style="margin-top:10px;display:grid;gap:8px;"></div>
            </div>
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <div>
              <h2 class="section-title">日历联动</h2>
              <div class="section-sub">点击日期切换所有数据面板。</div>
            </div>
            <span id="calendarLabel" class="pill"><?= htmlspecialchars($today, ENT_QUOTES, 'UTF-8') ?></span>
          </div>
          <div class="projects">
            <div id="calendarGrid" class="calendar-grid"></div>
            <div style="margin-top:22px; display:flex; justify-content:space-between; align-items:center; gap:12px;">
              <div>
                <h2 class="section-title">通知</h2>
                <div class="section-sub">从接口和当前数据自动生成的提醒。</div>
              </div>
              <span class="pill ok">实时</span>
            </div>
            <div id="feedTimeline" class="timeline" style="margin-top:10px;"></div>
          </div>
        </div>
      </section>

      <section id="section-projects" style="margin-top:22px;" class="dashboard dashboard-section" style="grid-template-columns:1fr!important;">
        <style>
          #section-projects { grid-template-columns:1fr!important; gap:18px; }

          /* 账号明细工具栏 */
          #section-projects .proj-toolbar {
            display:flex; align-items:center; justify-content:space-between; flex-wrap:wrap; gap:10px;
            padding:10px 20px; border-bottom:1px solid var(--line);
          }
          #section-projects .proj-toolbar .meta { font-size:13px; color:var(--muted); }
          #section-projects .proj-btns { display:flex; gap:8px; flex-wrap:wrap; }
          #section-projects .proj-btns .page-btn { height:30px; padding:0 12px; font-size:12px; border-radius:8px; }

          /* 账号表格：紧凑风格 */
          #section-projects .proj-table { width:100%; border-collapse:collapse; font-size:13px; }
          #section-projects .proj-table thead th {
            padding:8px 12px; background:rgba(0,0,0,.025); font-size:12px; font-weight:700;
            color:var(--muted); border-bottom:1px solid var(--line); text-align:left; white-space:nowrap; position:sticky; top:0; z-index:1;
          }
          #section-projects .proj-table tbody td {
            padding:8px 12px; border-bottom:1px solid var(--line); vertical-align:middle; font-size:13px;
          }
          #section-projects .proj-table tbody tr:hover { background:rgba(232,161,95,.05); }
          #section-projects .proj-table-wrap {
            overflow-x:auto; overflow-y:auto; max-height:480px;
          }

          /* 最近动态：横向滚动卡片 */
          #section-projects .feed-wrap {
            display:flex; gap:14px; overflow-x:auto; padding:4px 2px 8px;
            scrollbar-width:thin; scrollbar-color:var(--line) transparent;
          }
          #section-projects .feed-wrap::-webkit-scrollbar { height:5px; }
          #section-projects .feed-wrap::-webkit-scrollbar-thumb { background:var(--line); border-radius:4px; }
          #section-projects .feed-card {
            flex:0 0 240px; background:var(--bg-soft); border:1px solid var(--line);
            border-radius:14px; padding:14px 16px; display:flex; flex-direction:column; gap:6px;
          }
          #section-projects .feed-card .fc-account {
            font-size:12px; font-weight:700; color:var(--text); white-space:nowrap;
            overflow:hidden; text-overflow:ellipsis;
          }
          #section-projects .feed-card .fc-meta {
            font-size:12px; color:var(--muted); line-height:1.5;
          }
          #section-projects .feed-card .fc-badge {
            display:inline-flex; align-items:center; justify-content:center;
            width:22px; height:22px; border-radius:50%;
            background:var(--orange); color:#fff; font-size:11px; font-weight:700; flex-shrink:0;
          }
          #section-projects .feed-card .fc-top {
            display:flex; align-items:center; gap:8px;
          }
          #section-projects .feed-card .fc-time {
            font-size:11px; color:var(--muted); margin-left:auto; white-space:nowrap;
          }
        </style>

        <!-- 账号明细 -->
        <div class="panel" style="overflow:hidden;">
          <div class="panel-head" style="padding-bottom:10px;">
            <div>
              <h2 class="section-title">账号明细</h2>
              <div class="section-sub">支持筛选、分页、排序与备注搜索。</div>
            </div>
            <div class="control-row">
              <select id="accountZeroFilter" class="control small">
                <option value="1">包含 0 上传</option>
                <option value="0">仅活跃账号</option>
              </select>
              <input id="keywordInput" class="control small" type="text" placeholder="搜索账号 / 备注">
            </div>
          </div>
          <div class="proj-toolbar">
            <div class="meta" id="accountSummary">共 0 条</div>
            <div class="proj-btns">
              <button id="newAccountBtn" class="page-btn">新建账号</button>
              <button id="generateLinkBtn" class="page-btn">生成查询链接</button>
              <button id="backfillBtn" class="page-btn">补齐账号档案</button>
              <button id="exportBtn" class="page-btn">导出当前页</button>
            </div>
          </div>
          <div id="accountHint" class="empty" style="margin:0 20px 8px; display:none;"></div>
          <div class="proj-table-wrap" style="padding:0 0 2px;">
            <table class="proj-table">
              <thead>
                <tr>
                  <th>账号</th><th>备注</th><th>统计日期</th><th>当天上传</th><th>最新上传</th><th>状态</th><th>操作</th>
                </tr>
              </thead>
              <tbody id="accountTbody"></tbody>
            </table>
          </div>
          <div class="pagination" style="padding:10px 20px;">
            <button id="prevPage" class="page-btn">上一页</button>
            <span id="pageInfo" class="meta">第 1 页</span>
            <button id="nextPage" class="page-btn">下一页</button>
          </div>
        </div>

        <!-- 最近动态 -->
        <div class="panel">
          <div class="panel-head" style="padding-bottom:10px;">
            <div>
              <h2 class="section-title">最近动态</h2>
              <div class="section-sub">根据账号明细自动生成卡片式消息。</div>
            </div>
            <span class="pill">消息区</span>
          </div>
          <div style="padding:0 20px 16px;">
            <div id="messagesContainer" class="feed-wrap"></div>
          </div>
        </div>
      </section>
      <!-- ===== 我的任务 ===== -->
      <section id="section-tasks" class="dashboard dashboard-section" style="display:none;margin-top:0;grid-template-columns:1fr!important;">
        <style>
          .task-board { display:grid; grid-template-columns:repeat(3,1fr); gap:18px; }
          @media(max-width:900px){ .task-board{grid-template-columns:1fr;} }
          .task-col { background:var(--bg-soft); border:1px solid var(--line); border-radius:20px; padding:16px; min-height:320px; }
          .task-col-title { font-size:13px; font-weight:800; color:var(--muted); text-transform:uppercase; letter-spacing:.1em; margin-bottom:14px; display:flex; align-items:center; justify-content:space-between; }
          .task-col-title .task-count { background:var(--orange-2); color:var(--orange); border-radius:999px; padding:2px 9px; font-size:12px; }
          .task-item { background:var(--panel-strong,#fff); border:1px solid var(--line); border-radius:14px; padding:13px 14px; margin-bottom:10px; cursor:pointer; transition:.15s; }
          .task-item:hover { box-shadow:var(--shadow-soft); transform:translateY(-1px); }
          .task-item-title { font-size:14px; font-weight:700; margin-bottom:5px; }
          .task-item-meta { font-size:12px; color:var(--muted); display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
          .task-tag { display:inline-flex; align-items:center; padding:2px 8px; border-radius:999px; font-size:11px; font-weight:700; }
          .task-tag.urgent { background:#fde8db; color:#cb6a25; }
          .task-tag.normal { background:#eef7f2; color:#4f9376; }
          .task-tag.low { background:#f3f0ff; color:#7c5cbf; }
          .task-input-row { display:flex; gap:8px; margin-top:12px; }
          .task-input-row input { flex:1; height:34px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); padding:0 12px; font-size:13px; color:var(--text); outline:none; }
          .task-input-row input:focus { border-color:var(--orange); }
          .task-input-row button { height:34px; padding:0 14px; border-radius:10px; border:0; background:var(--orange); color:#fff; font-size:13px; font-weight:700; cursor:pointer; }
          .task-progress-bar { height:5px; border-radius:999px; background:var(--line); margin-top:8px; overflow:hidden; }
          .task-progress-bar span { display:block; height:100%; border-radius:inherit; background:linear-gradient(90deg,var(--orange),#f3c28f); transition:width .3s; }
          .task-stats-row { display:grid; grid-template-columns:repeat(3,1fr); gap:12px; margin-bottom:18px; }
          .task-stat { background:var(--panel,rgba(255,255,255,.84)); border:1px solid var(--line); border-radius:16px; padding:14px 16px; display:flex; flex-direction:column; gap:4px; }
          .task-stat strong { font-size:26px; font-weight:900; }
          .task-stat span { font-size:12px; color:var(--muted); }
        </style>
        <div class="panel" style="width:100%;box-sizing:border-box;">
          <div class="panel-head" style="padding-bottom:14px;">
            <div>
              <h2 class="section-title">我的任务</h2>
              <div class="section-sub">拖动卡片管理待办、进行中和已完成任务。</div>
            </div>
            <button class="page-btn" onclick="openAddTaskModal()">＋ 新建任务</button>
          </div>
          <div style="padding:0 20px 20px;">
            <div class="task-stats-row">
              <div class="task-stat"><span>总任务</span><strong id="taskTotalCnt">0</strong></div>
              <div class="task-stat"><span>进行中</span><strong id="taskDoingCnt">0</strong></div>
              <div class="task-stat"><span>已完成</span><strong id="taskDoneCnt">0</strong></div>
            </div>
            <div class="task-board">
              <div class="task-col" id="task-col-todo">
                <div class="task-col-title">待办 <span class="task-count" id="tcnt-todo">0</span></div>
                <div id="task-list-todo"></div>
                <div class="task-input-row">
                  <input id="quickTaskInput" placeholder="快速添加任务…" onkeydown="if(event.key==='Enter')quickAddTask()">
                  <button onclick="quickAddTask()">添加</button>
                </div>
              </div>
              <div class="task-col" id="task-col-doing">
                <div class="task-col-title">进行中 <span class="task-count" id="tcnt-doing">0</span></div>
                <div id="task-list-doing"></div>
              </div>
              <div class="task-col" id="task-col-done">
                <div class="task-col-title">已完成 <span class="task-count" id="tcnt-done">0</span></div>
                <div id="task-list-done"></div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <!-- ===== /我的任务 ===== -->

      <!-- ===== 日历 ===== -->
      <section id="section-calendar" class="dashboard dashboard-section" style="display:none;margin-top:0;grid-template-columns:1fr!important;">
        <style>
          .cal-full-wrap { display:grid; grid-template-columns:1fr 340px; gap:18px; }
          @media(max-width:900px){ .cal-full-wrap{grid-template-columns:1fr;} }
          .cal-header { display:flex; align-items:center; justify-content:space-between; margin-bottom:16px; }
          .cal-nav-btn { width:36px; height:36px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); cursor:pointer; font-size:16px; display:flex; align-items:center; justify-content:center; }
          .cal-month-title { font-size:20px; font-weight:800; }
          .cal-full-grid { display:grid; grid-template-columns:repeat(7,1fr); gap:6px; }
          .cal-day-label { text-align:center; font-size:12px; color:var(--muted); font-weight:700; padding:6px 0; }
          .cal-full-cell { min-height:80px; padding:8px; border-radius:12px; background:var(--bg-soft); border:1px solid transparent; cursor:pointer; transition:.15s; display:flex; flex-direction:column; gap:3px; }
          .cal-full-cell:hover { border-color:var(--orange-2); background:rgba(255,255,255,.9); }
          .cal-full-cell.today { border-color:var(--orange); background:rgba(232,161,95,.08); }
          .cal-full-cell.selected { border-color:var(--orange); background:rgba(232,161,95,.14); box-shadow:0 0 0 2px rgba(232,161,95,.3); }
          .cal-full-cell.other-month { opacity:.38; }
          .cal-full-cell .cal-num { font-size:15px; font-weight:800; }
          .cal-full-cell.today .cal-num { color:var(--orange); }
          .cal-event-dot { height:5px; border-radius:999px; background:var(--orange); margin-top:2px; }
          .cal-event-mini { font-size:10px; padding:2px 5px; border-radius:4px; background:rgba(232,161,95,.2); color:var(--text); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
          .cal-event-list { display:grid; gap:10px; }
          .cal-event-item { display:flex; align-items:flex-start; gap:10px; padding:12px; border-radius:12px; background:var(--bg-soft); border:1px solid var(--line); cursor:pointer; transition:.15s; }
          .cal-event-item:hover { border-color:var(--orange-2); box-shadow:var(--shadow-soft); }
          .cal-event-color { width:6px; border-radius:999px; align-self:stretch; flex-shrink:0; }
          .cal-event-info { flex:1; min-width:0; }
          .cal-event-info strong { display:block; font-size:14px; margin-bottom:3px; }
          .cal-event-info span { font-size:12px; color:var(--muted); }
          .add-event-form { display:grid; gap:10px; }
          .add-event-form input, .add-event-form textarea, .add-event-form select { height:36px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); padding:0 12px; font-size:13px; color:var(--text); outline:none; width:100%; }
          .add-event-form textarea { height:68px; padding:8px 12px; resize:none; }
          .add-event-form input:focus, .add-event-form textarea:focus, .add-event-form select:focus { border-color:var(--orange); }
        </style>
        <div class="panel" style="width:100%;box-sizing:border-box;">
          <div class="panel-head" style="padding-bottom:14px;">
            <div>
              <h2 class="section-title">日历</h2>
              <div class="section-sub">查看和管理日程事件，支持添加、删除、颜色标记。</div>
            </div>
            <button class="page-btn" onclick="openAddEventModal()">＋ 添加事件</button>
          </div>
          <div style="padding:0 20px 20px;">
            <div class="cal-full-wrap">
              <div>
                <div class="cal-header">
                  <button class="cal-nav-btn" onclick="calNav(-1)">‹</button>
                  <div class="cal-month-title" id="calMonthTitle"></div>
                  <button class="cal-nav-btn" onclick="calNav(1)">›</button>
                </div>
                <div class="cal-full-grid" id="calDayLabels"></div>
                <div class="cal-full-grid" id="calFullGrid" style="margin-top:4px;"></div>
              </div>
              <div>
                <div style="font-size:15px;font-weight:800;margin-bottom:12px;" id="calSelectedLabel">今日事件</div>
                <div id="calEventList" class="cal-event-list"><div class="empty">暂无事件</div></div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <!-- ===== /日历 ===== -->

      <!-- ===== 时间管理 ===== -->
      <section id="section-time" class="dashboard dashboard-section" style="display:none;margin-top:0;grid-template-columns:1fr!important;">
        <style>
          .time-grid { display:grid; grid-template-columns:1fr 1fr; gap:18px; }
          @media(max-width:900px){ .time-grid{grid-template-columns:1fr;} }
          /* 番茄钟 */
          .pomodoro-wrap { display:flex; flex-direction:column; align-items:center; padding:28px 0 20px; gap:14px; }
          .pomodoro-ring { position:relative; width:200px; height:200px; }
          .pomodoro-ring svg { position:absolute; top:0; left:0; transform:rotate(-90deg); }
          .pomodoro-ring circle { transition:stroke-dashoffset .9s linear; }
          .pomodoro-time { position:absolute; inset:0; display:flex; flex-direction:column; align-items:center; justify-content:center; }
          .pomodoro-time span { font-size:42px; font-weight:900; line-height:1; letter-spacing:-.02em; }
          .pomodoro-time small { font-size:13px; color:var(--muted); margin-top:4px; }
          .pomodoro-btns { display:flex; gap:10px; }
          .pomodoro-btn { height:40px; padding:0 22px; border-radius:12px; border:0; font-size:14px; font-weight:700; cursor:pointer; }
          .pomodoro-btn.start { background:var(--orange); color:#fff; }
          .pomodoro-btn.reset { background:var(--bg-soft); color:var(--text); border:1px solid var(--line); }
          .pomodoro-mode-row { display:flex; gap:8px; }
          .pomodoro-mode-btn { height:30px; padding:0 14px; border-radius:999px; border:1px solid var(--line); background:transparent; font-size:12px; font-weight:700; cursor:pointer; color:var(--muted); }
          .pomodoro-mode-btn.active { background:var(--orange); color:#fff; border-color:var(--orange); }
          .pomodoro-log { max-height:160px; overflow-y:auto; display:grid; gap:6px; width:100%; margin-top:4px; }
          .pomodoro-log-item { font-size:12px; padding:7px 10px; border-radius:8px; background:var(--bg-soft); border:1px solid var(--line); color:var(--muted); display:flex; justify-content:space-between; }
          /* 时间块 */
          .timeblock-list { display:grid; gap:10px; }
          .timeblock-item { display:flex; align-items:center; gap:12px; padding:12px 14px; border-radius:14px; background:var(--bg-soft); border:1px solid var(--line); cursor:pointer; transition:.15s; }
          .timeblock-item:hover { border-color:var(--orange-2); box-shadow:var(--shadow-soft); }
          .timeblock-color { width:8px; height:42px; border-radius:999px; flex-shrink:0; }
          .timeblock-info { flex:1; min-width:0; }
          .timeblock-info strong { display:block; font-size:14px; font-weight:700; }
          .timeblock-info span { font-size:12px; color:var(--muted); }
          .timeblock-del { width:28px; height:28px; border-radius:8px; border:1px solid var(--line); background:rgba(255,255,255,.8); cursor:pointer; font-size:14px; display:flex; align-items:center; justify-content:center; color:var(--muted); }
          .timeblock-del:hover { color:var(--red); border-color:var(--red); }
          .timeblock-form { display:grid; gap:8px; margin-top:12px; }
          .timeblock-form input, .timeblock-form select { height:34px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); padding:0 12px; font-size:13px; color:var(--text); outline:none; width:100%; }
          .timeblock-form input:focus, .timeblock-form select:focus { border-color:var(--orange); }
          .timeblock-row { display:grid; grid-template-columns:1fr 1fr; gap:8px; }
          .time-stats { display:grid; grid-template-columns:repeat(4,1fr); gap:10px; margin-bottom:18px; }
          @media(max-width:700px){ .time-stats{grid-template-columns:repeat(2,1fr);} }
        </style>
        <div class="panel" style="width:100%;box-sizing:border-box;">
          <div class="panel-head" style="padding-bottom:14px;">
            <div>
              <h2 class="section-title">时间管理</h2>
              <div class="section-sub">番茄钟 · 时间块规划，提升专注效率。</div>
            </div>
            <span class="pill ok" id="pomodoroStatus">专注中</span>
          </div>
          <div style="padding:0 20px 20px;">
            <div class="time-stats">
              <div class="task-stat"><span>今日专注</span><strong id="tmTodayFocus">0</strong><small style="font-size:11px;color:var(--muted);">个番茄</small></div>
              <div class="task-stat"><span>专注分钟</span><strong id="tmTodayMins">0</strong></div>
              <div class="task-stat"><span>时间块数</span><strong id="tmBlockCnt">0</strong></div>
              <div class="task-stat"><span>今日计划</span><strong id="tmPlanHrs">0h</strong></div>
            </div>
            <div class="time-grid">
              <!-- 番茄钟 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">番茄钟</h3><div class="section-sub">25分钟专注 / 5分钟休息</div></div></div>
                <div class="pomodoro-wrap">
                  <div class="pomodoro-mode-row">
                    <button class="pomodoro-mode-btn active" id="pMode-focus" onclick="setPomodoroMode('focus')">专注 25min</button>
                    <button class="pomodoro-mode-btn" id="pMode-short" onclick="setPomodoroMode('short')">短休 5min</button>
                    <button class="pomodoro-mode-btn" id="pMode-long" onclick="setPomodoroMode('long')">长休 15min</button>
                  </div>
                  <div class="pomodoro-ring">
                    <svg width="200" height="200" viewBox="0 0 200 200">
                      <circle cx="100" cy="100" r="85" fill="none" stroke="var(--line)" stroke-width="10"/>
                      <circle id="pomodoroArc" cx="100" cy="100" r="85" fill="none" stroke="var(--orange)" stroke-width="10" stroke-linecap="round"
                        stroke-dasharray="534" stroke-dashoffset="0"/>
                    </svg>
                    <div class="pomodoro-time">
                      <span id="pomodoroDisplay">25:00</span>
                      <small id="pomodoroLabel">专注</small>
                    </div>
                  </div>
                  <div class="pomodoro-btns">
                    <button class="pomodoro-btn start" id="pomodoroStartBtn" onclick="togglePomodoro()">开始</button>
                    <button class="pomodoro-btn reset" onclick="resetPomodoro()">重置</button>
                  </div>
                  <div class="pomodoro-log" id="pomodoroLog"></div>
                </div>
              </div>
              <!-- 时间块 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">时间块规划</h3><div class="section-sub">为今天的时间段安排任务</div></div></div>
                <div class="timeblock-form">
                  <input id="tbName" placeholder="任务名称…">
                  <div class="timeblock-row">
                    <input id="tbStart" type="time" value="09:00">
                    <input id="tbEnd" type="time" value="10:00">
                  </div>
                  <select id="tbColor">
                    <option value="#e8a15f">🟠 橙色</option>
                    <option value="#6da98d">🟢 绿色</option>
                    <option value="#3a8fd4">🔵 蓝色</option>
                    <option value="#8b5cf6">🟣 紫色</option>
                    <option value="#de7d67">🔴 红色</option>
                  </select>
                  <button class="page-btn" onclick="addTimeblock()">添加时间块</button>
                </div>
                <div style="margin-top:14px;font-size:13px;font-weight:700;color:var(--muted);margin-bottom:8px;">今日时间块</div>
                <div class="timeblock-list" id="timeblockList"></div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <!-- ===== /时间管理 ===== -->

      <!-- ===== 报表 ===== -->
      <section id="section-reports" class="dashboard dashboard-section" style="display:none;margin-top:0;grid-template-columns:1fr!important;">
        <style>
          .report-grid { display:grid; grid-template-columns:1fr 1fr; gap:18px; }
          @media(max-width:900px){ .report-grid{grid-template-columns:1fr;} }
          .report-kpi { display:grid; grid-template-columns:repeat(4,1fr); gap:12px; margin-bottom:18px; }
          @media(max-width:700px){ .report-kpi{grid-template-columns:repeat(2,1fr);} }
          .kpi-card { border-radius:18px; padding:18px; background:var(--panel,rgba(255,255,255,.84)); border:1px solid var(--line); display:flex; flex-direction:column; gap:6px; }
          .kpi-card .kpi-num { font-size:30px; font-weight:900; line-height:1.1; }
          .kpi-card .kpi-label { font-size:12px; color:var(--muted); font-weight:600; }
          .kpi-card .kpi-delta { font-size:12px; font-weight:700; }
          .kpi-card .kpi-delta.up { color:var(--green); }
          .kpi-card .kpi-delta.down { color:var(--red); }
          .report-toolbar { display:flex; align-items:center; gap:10px; flex-wrap:wrap; margin-bottom:14px; }
          .report-toolbar select { height:34px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); padding:0 12px; font-size:13px; color:var(--text); outline:none; }
          .bar-chart-wrap { display:flex; flex-direction:column; gap:10px; }
          .bar-row { display:flex; align-items:center; gap:10px; }
          .bar-label { font-size:12px; font-weight:700; width:110px; text-align:right; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; color:var(--text); }
          .bar-track { flex:1; height:22px; background:var(--line); border-radius:999px; overflow:hidden; }
          .bar-fill { height:100%; border-radius:999px; background:linear-gradient(90deg,var(--orange),#f3c28f); transition:width .6s cubic-bezier(.4,0,.2,1); display:flex; align-items:center; justify-content:flex-end; padding-right:8px; }
          .bar-fill span { font-size:11px; font-weight:800; color:#fff; }
          .pie-wrap { display:flex; align-items:center; gap:18px; flex-wrap:wrap; justify-content:center; }
          .pie-legend { display:grid; gap:6px; }
          .pie-legend-item { display:flex; align-items:center; gap:8px; font-size:13px; }
          .pie-dot { width:12px; height:12px; border-radius:50%; flex-shrink:0; }
          .heat-grid { display:grid; grid-template-columns:repeat(7,1fr); gap:4px; margin-top:8px; }
          .heat-cell { height:18px; border-radius:4px; background:rgba(232,161,95,.1); transition:.2s; cursor:pointer; }
          .heat-cell:hover { opacity:.7; }
          .heat-cell.h1 { background:rgba(232,161,95,.25); }
          .heat-cell.h2 { background:rgba(232,161,95,.45); }
          .heat-cell.h3 { background:rgba(232,161,95,.65); }
          .heat-cell.h4 { background:rgba(232,161,95,.85); }
        </style>
        <div class="panel" style="width:100%;box-sizing:border-box;">
          <div class="panel-head" style="padding-bottom:14px;">
            <div>
              <h2 class="section-title">报表</h2>
              <div class="section-sub">综合数据分析：上传量、活跃趋势、备注排行、热力图。</div>
            </div>
            <div class="report-toolbar" style="margin:0;">
              <select id="reportRangeSelect" onchange="renderReportPage()">
                <option value="7">近 7 天</option>
                <option value="14" selected>近 14 天</option>
                <option value="30">近 30 天</option>
              </select>
              <button class="page-btn" onclick="renderReportPage()">刷新</button>
            </div>
          </div>
          <div style="padding:0 20px 20px;">
            <!-- KPI 卡片 -->
            <div class="report-kpi">
              <div class="kpi-card">
                <div class="kpi-label">周期总上传</div>
                <div class="kpi-num" id="rptTotal">—</div>
                <div class="kpi-delta" id="rptTotalDelta"></div>
              </div>
              <div class="kpi-card">
                <div class="kpi-label">日均上传</div>
                <div class="kpi-num" id="rptAvgDay">—</div>
              </div>
              <div class="kpi-card">
                <div class="kpi-label">峰值单日</div>
                <div class="kpi-num" id="rptPeak">—</div>
                <div class="kpi-delta" id="rptPeakDate" style="font-weight:600;color:var(--muted);"></div>
              </div>
              <div class="kpi-card">
                <div class="kpi-label">活跃账号均值</div>
                <div class="kpi-num" id="rptAvgAccount">—</div>
              </div>
            </div>
            <div class="report-grid">
              <!-- 趋势柱状图 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">上传量趋势</h3><div class="section-sub">按天统计的上传数量</div></div></div>
                <svg id="rptBarChart" class="chart" viewBox="0 0 560 200" preserveAspectRatio="none" style="height:200px;"></svg>
              </div>
              <!-- 备注排行横向条形图 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">备注分组排行</h3><div class="section-sub">各备注组今日累计上传</div></div></div>
                <div class="bar-chart-wrap" id="rptRemarkBar"></div>
              </div>
              <!-- 热力图 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">活跃热力图</h3><div class="section-sub">近期每日活跃强度</div></div></div>
                <div class="heat-grid" id="rptHeatGrid"></div>
                <div class="chart-axis" id="rptHeatAxis" style="margin-top:4px;"></div>
              </div>
              <!-- 饼图：活跃 vs 静默 -->
              <div class="chart-box">
                <div class="chart-title"><div><h3 class="section-title" style="font-size:15px;">账号活跃占比</h3><div class="section-sub">今日有/无上传账号比例</div></div></div>
                <div class="pie-wrap" style="margin-top:12px;">
                  <svg id="rptPieChart" width="150" height="150" viewBox="0 0 150 150"></svg>
                  <div class="pie-legend" id="rptPieLegend"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <!-- ===== /报表 ===== -->

      <!-- ===== 设置页 ===== -->
      <section id="section-settings" class="dashboard dashboard-section" style="display:none;margin-top:0;grid-template-columns:1fr!important;">
        <style>
          /* 设置页专用：覆盖通用 table 样式，让账号表格更紧凑 */
          #section-settings .acc-table {
            width:100%; border-collapse:collapse; font-size:13px;
          }
          #section-settings .acc-table th,
          #section-settings .acc-table td {
            padding:7px 10px; border-bottom:1px solid var(--line); text-align:left; vertical-align:middle; white-space:nowrap;
          }
          #section-settings .acc-table thead th {
            background:rgba(0,0,0,.03); font-size:12px; font-weight:700; color:var(--muted); position:sticky;top:0;
          }
          #section-settings .acc-table tbody tr:hover { background:rgba(232,161,95,.06); }
          #section-settings .acc-table td.acc-cell { max-width:260px; overflow:hidden; text-overflow:ellipsis; }
          #section-settings .acc-table-wrap {
            max-height:calc(100vh - 260px); overflow-y:auto; border:1px solid var(--line); border-radius:14px; margin-top:10px;
          }
          .s-btn { height:26px; padding:0 10px; border-radius:7px; border:1px solid var(--line); background:rgba(255,255,255,.9); cursor:pointer; font-size:12px; color:var(--text); }
          .s-btn.danger { color:var(--red); }
          .s-btn.primary { background:var(--orange); color:#fff; border-color:var(--orange); }
          .s-search-bar { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
          .s-search-bar input { height:34px; border-radius:10px; border:1px solid var(--line); background:rgba(255,255,255,.9); padding:0 12px; font-size:13px; color:var(--text); outline:none; min-width:200px; }
          .s-search-bar input:focus { border-color:var(--orange); box-shadow:0 0 0 2px rgba(232,161,95,.2); }
          .s-batch-bar { display:none; align-items:center; gap:8px; padding:8px 12px; background:var(--orange-2); border-radius:10px; margin:8px 0 4px; flex-wrap:wrap; }
          .s-batch-bar span { font-weight:600; font-size:13px; min-width:70px; }
        </style>

        <!-- 账号管理主面板 -->
        <div class="panel" style="width:100%;box-sizing:border-box;">

          <!-- 顶部标题+搜索栏 -->
          <div class="panel-head" style="flex-wrap:wrap;gap:10px;padding-bottom:12px;">
            <div>
              <h2 class="section-title">账号管理</h2>
              <div class="section-sub">按备注搜索账号，支持全选、批量删除、批量改备注</div>
            </div>
            <div class="s-search-bar">
              <input id="remarkSearchInput" type="text" placeholder="输入备注关键词…"
                onkeydown="if(event.key==='Enter')searchByRemark()">
              <button class="s-btn primary" onclick="searchByRemark()">搜索</button>
              <button class="s-btn" onclick="clearRemarkFilter()">全部账号</button>
            </div>
          </div>

          <!-- 批量操作栏 -->
          <div id="batchBar" class="s-batch-bar" style="margin:0 20px 0;">
            <span id="batchCount">已选 0 条</span>
            <button class="s-btn primary" onclick="batchChangeRemark()">批量改备注</button>
            <button class="s-btn danger" onclick="batchDelete()">批量删除</button>
            <button class="s-btn" onclick="clearSelection()">取消选择</button>
          </div>

          <!-- 账号表格 -->
          <div style="padding:0 20px 16px;">
            <div class="acc-table-wrap">
              <table class="acc-table">
                <thead>
                  <tr>
                    <th style="width:32px;text-align:center;"><input type="checkbox" id="selectAllChk" onchange="toggleSelectAll(this.checked)" title="全选/取消"></th>
                    <th>账号</th>
                    <th style="width:140px;">备注</th>
                    <th style="width:130px;">创建时间</th>
                    <th style="width:60px;text-align:center;">操作</th>
                  </tr>
                </thead>
                <tbody id="accTbody"><tr><td colspan="5" style="text-align:center;padding:20px;color:var(--muted);">加载中…</td></tr></tbody>
              </table>
            </div>
            <div style="margin-top:10px;display:flex;align-items:center;gap:8px;justify-content:flex-end;">
              <button class="s-btn" onclick="loadAccounts(settingsState.accPage-1)">上一页</button>
              <span class="meta" id="accPageInfo" style="font-size:12px;">第 1 页</span>
              <button class="s-btn" onclick="loadAccounts(settingsState.accPage+1)">下一页</button>
            </div>
          </div>
        </div>

      </section>
      <!-- ===== /设置页 ===== -->

    </main>
  </div>

  <div id="modalBackdrop" class="modal-backdrop" aria-hidden="true">
    <div class="modal" role="dialog" aria-modal="true">
      <div class="modal-head">
        <div>
          <div id="modalKicker" class="pill">详情</div>
          <h2 id="modalTitle" class="section-title" style="margin-top:10px;">详情</h2>
          <div id="modalSubtitle" class="section-sub"></div>
        </div>
        <button id="modalClose" class="close-btn" aria-label="关闭">✕</button>
      </div>
      <div id="modalBody" class="modal-body"></div>
    </div>
  </div>

  <div id="toast" class="toast"></div>

  <script>
    const apiUrl = <?= json_encode($apiUrl, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?>;
    const calendarApiUrl = <?= json_encode($_SERVER['PHP_SELF'] ?? 'app.php', JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?>;
    const authApiUrl = <?= json_encode($_SERVER['PHP_SELF'] ?? 'app.php', JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) ?>;
    const today = <?= json_encode($today, JSON_UNESCAPED_UNICODE) ?>;
    const state = {
      search: '',
      keyword: '',
      includeZero: '1',
      selectedDate: today,
      page: 1,
      pageSize: 10,
      dashboard: null,
      accounts: [],
      trend14: [],
      hour24: [],
      calendar: [],
      feed: [],
      notifications: [],
      allAccountsForLinks: [],
      refreshTimer: null,
      lastRefreshAt: null,
      refreshing: false
    };

    /* ===== 主题切换 ===== */
    function setTheme(name, el) {
      document.documentElement.setAttribute('data-theme', name === 'default' ? '' : name);
      if (name === 'default') document.documentElement.removeAttribute('data-theme');
      localStorage.setItem('theme', name);
      document.querySelectorAll('.theme-dot').forEach(d => d.classList.remove('active'));
      if (el) el.classList.add('active');
      document.getElementById('themeMenu').classList.remove('open');
    }

    // 页面加载时恢复主题
    (function() {
      const saved = localStorage.getItem('theme') || 'default';
      if (saved && saved !== 'default') document.documentElement.setAttribute('data-theme', saved);
      window.addEventListener('DOMContentLoaded', () => {
        const dot = document.querySelector(`.theme-dot[data-theme="${saved}"]`);
        if (dot) { document.querySelectorAll('.theme-dot').forEach(d=>d.classList.remove('active')); dot.classList.add('active'); }
        // 头像按钮点击切换菜单
        document.getElementById('themeBtn').addEventListener('click', function(e) {
          e.stopPropagation();
          document.getElementById('themeMenu').classList.toggle('open');
        });
        document.addEventListener('click', () => document.getElementById('themeMenu').classList.remove('open'));
        document.getElementById('themeMenu').addEventListener('click', e => e.stopPropagation());
      });
    })();

    const esc = (s) => String(s ?? '').replace(/[&<>"']/g, m => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));

    async function logoutApp() {
      try {
        await fetch(`${authApiUrl}?auth_action=logout`, { method: 'POST', credentials: 'same-origin' });
      } finally {
        window.location.reload();
      }
    }

    function toast(msg) {
      const el = document.getElementById('toast');
      el.textContent = msg;
      el.classList.add('show');
      clearTimeout(window.__toastTimer);
      window.__toastTimer = setTimeout(() => el.classList.remove('show'), 1800);
    }

    function setLiveStatus(label) {
      const badge = document.getElementById('liveStatusBadge');
      if (badge) badge.textContent = label;
    }

    function scheduleAutoRefresh() {
      clearInterval(state.refreshTimer);
      state.refreshTimer = setInterval(async () => {
        if (document.hidden || state.refreshing) return;
        try {
          state.refreshing = true;
          setLiveStatus('刷新中…');
          await loadDashboard(true);
          state.lastRefreshAt = new Date();
          setLiveStatus('实时在线');
        } catch (e) {
          console.error(e);
          setLiveStatus('同步失败');
        } finally {
          state.refreshing = false;
        }
      }, 2000);
    }

    function openModal(kind, payload = {}) {
      const backdrop = document.getElementById('modalBackdrop');
      const kicker = document.getElementById('modalKicker');
      const title = document.getElementById('modalTitle');
      const subtitle = document.getElementById('modalSubtitle');
      const body = document.getElementById('modalBody');

      if (kind === 'account') {
        kicker.textContent = '账号详情';
        title.textContent = payload.account || '未命名账号';
        subtitle.textContent = payload.remark || '未备注';
        body.innerHTML = `
          <div class="modal-grid">
            <div class="detail-box">
              <div class="mini-list">
                <div class="mini-item"><span>账号 ID</span><strong>${esc(payload.id || '-')}</strong></div>
                <div class="mini-item"><span>当天上传</span><strong>${esc(payload.day_success ?? 0)}</strong></div>
                <div class="mini-item"><span>最新上传</span><strong>${esc(formatDisplayTime(payload.latest_upload_at || payload.updated_at || '-'))}</strong></div>
                <div class="mini-item"><span>备注</span><strong>${esc(payload.remark || '未备注')}</strong></div>
              </div>
              <div style="margin-top:16px; display:flex; gap:10px; flex-wrap:wrap;">
                <button class="page-btn" data-action="edit-remark">编辑备注</button>
                <button class="page-btn" data-action="new-account">新建账号</button>
              </div>
            </div>
            <div class="detail-box">
              <div class="section-title" style="font-size:16px;margin-bottom:12px;">快速操作</div>
              <div class="mini-list">
                <div class="mini-item"><span>查看趋势</span><strong>联动日历日期</strong></div>
                <div class="mini-item"><span>当前筛选</span><strong>${state.includeZero === '1' ? '包含 0 上传' : '仅活跃'}</strong></div>
                <div class="mini-item"><span>搜索关键词</span><strong>${esc(state.keyword || '无')}</strong></div>
              </div>
            </div>
          </div>`;
      } else if (kind === 'feed') {
        kicker.textContent = '动态详情';
        title.textContent = payload.title || '消息详情';
        subtitle.textContent = payload.sub || '';
        body.innerHTML = `<div class="detail-box"><p style="margin:0;color:var(--text);line-height:1.8;font-size:15px;">${esc(payload.text || '')}</p></div>`;
      } else if (kind === 'new-account') {
        kicker.textContent = '新建账号';
        title.textContent = '新增账号';
        subtitle.textContent = '支持单个账号或批量粘贴创建，备注可统一设置';
        body.innerHTML = `
          <div class="detail-box">
            <div class="mini-list">
              <label class="mini-item" style="align-items:flex-start; gap:12px;"><span style="padding-top:10px;">账号</span><textarea id="accountFormAccounts" class="control" style="width:100%;min-height:140px;resize:vertical;" placeholder="支持每行一个账号，或粘贴批量账号\n示例：\naccount_a\naccount_b\naccount_c"></textarea></label>
              <label class="mini-item" style="align-items:center;"><span>统一备注</span><input id="accountFormRemark" class="control" style="width:100%;max-width:420px;" placeholder="输入备注，可留空"></label>
            </div>
            <div class="section-sub" style="margin-top:10px;">说明：每行一个账号，系统会自动去重、去空行。若仅填写一行，也会按单账号保存。</div>
            <div style="margin-top:16px; display:flex; gap:10px; flex-wrap:wrap;">
              <button id="saveAccountBtn" class="page-btn">保存</button>
              <button id="fillDemoBtn" class="page-btn">填入示例</button>
            </div>
          </div>`;
      } else if (kind === 'generate-links') {
        kicker.textContent = '查询链接';
        title.textContent = '生成查询链接';
        subtitle.textContent = '支持搜索账号/备注、包含当天未上传账号、全选生成';
        const current = state.allAccountsForLinks || [];
        body.innerHTML = `
          <div class="detail-box">
            <div class="control-row" style="margin-bottom:12px;">
              <input id="linkSearchInput" class="control" style="flex:1;min-width:220px;" placeholder="搜索账号 / 备注">
              <label class="page-btn" style="display:inline-flex;align-items:center;gap:8px;cursor:pointer;">
                <input id="linkSelectAll" type="checkbox"> 全选
              </label>
            </div>
            <div class="section-sub" style="margin-bottom:12px;">勾选账号后，系统会按备注分组生成查询链接。默认包含当天未上传账号。</div>
            <div class="mini-list" id="linkPickList" style="max-height: 340px; overflow:auto; padding-right:6px;">
              ${current.map(r => `
                <label class="mini-item link-row" style="align-items:flex-start; gap:12px; cursor:pointer;" data-text="${esc((r.account || '') + ' ' + (r.remark || '')).toLowerCase()}">
                  <span><input type="checkbox" class="link-cb" value="${esc(r.id)}"></span>
                  <span style="flex:1; min-width:0;">
                    <strong>${esc(r.account)}</strong>
                    <div class="meta">${esc(r.remark || '未备注')} · ${Number(r.day_success || 0) > 0 ? `今日 ${esc(r.day_success)} 次` : '今日未上传'}</div>
                  </span>
                </label>`).join('')}
            </div>
            <div style="margin-top:16px; display:flex; gap:10px; flex-wrap:wrap;">
              <button id="buildLinkBtn" class="page-btn">生成链接</button>
            </div>
            <div id="linkResult" style="margin-top:14px; display:grid; gap:10px;"></div>
          </div>`;
      }
      backdrop.classList.add('show');
      backdrop.setAttribute('aria-hidden', 'false');
      if (kind === 'new-account') {
        document.getElementById('saveAccountBtn').onclick = saveAccountFromModal;
        document.getElementById('fillDemoBtn').onclick = () => {
          document.getElementById('accountFormAccounts').value = ['demo.alpha', 'demo.beta', 'demo.gamma'].join('\n');
          document.getElementById('accountFormRemark').value = '批量导入';
        };
      }
      if (kind === 'generate-links') {
        document.getElementById('buildLinkBtn').onclick = generateLinksFromModal;
        document.getElementById('linkSearchInput').oninput = (e) => filterLinkRows(e.target.value);
        document.getElementById('linkSelectAll').onchange = (e) => {
          document.querySelectorAll('.link-cb').forEach(cb => { cb.checked = e.target.checked; });
        };
      }
      if (kind === 'account') {
        const editBtn = body.querySelector('[data-action="edit-remark"]');
        const newBtn = body.querySelector('[data-action="new-account"]');
        if (editBtn) editBtn.onclick = () => openEditRemark(payload);
        if (newBtn) newBtn.onclick = () => openModal('new-account');
      }
    }

    function closeModal() {
      document.getElementById('modalBackdrop').classList.remove('show');
      document.getElementById('modalBackdrop').setAttribute('aria-hidden', 'true');
    }

    function openEditRemark(payload) {
      const remark = prompt('输入新的备注', payload.remark || '');
      if (remark === null) return;
      saveAccount({ id: payload.id, account: payload.account, remark });
    }

    async function saveAccount(data) {
      const res = await fetch(`${apiUrl}?action=account_save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });
      const json = await res.json();
      if (json.code !== 0) throw new Error(json.msg || '保存失败');
      toast('账号已保存');
      await loadDashboard();
      closeModal();
    }

    async function saveAccountFromModal() {
      const text = document.getElementById('accountFormAccounts')?.value?.trim() || '';
      const remark = document.getElementById('accountFormRemark')?.value?.trim() || '';
      const accounts = Array.from(new Set(text.split(/\r?\n/).map(v => v.trim()).filter(Boolean)));
      if (!accounts.length) return toast('请输入至少一个账号');
      try {
        for (const account of accounts) {
          await saveAccount({ account, remark });
        }
        toast(accounts.length > 1 ? `已批量保存 ${accounts.length} 个账号` : '账号已保存');
        await loadDashboard();
        closeModal();
      } catch (e) {
        toast(e.message || '保存失败');
      }
    }

    function filterLinkRows(q) {
      const term = String(q || '').trim().toLowerCase();
      document.querySelectorAll('.link-row').forEach(row => {
        const text = row.dataset.text || '';
        row.style.display = !term || text.includes(term) ? '' : 'none';
      });
    }

    async function copyText(text) {
      const value = String(text || '');
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(value);
          return true;
        }
      } catch (e) {}
      const ta = document.createElement('textarea');
      ta.value = value;
      ta.setAttribute('readonly', 'readonly');
      ta.style.position = 'fixed';
      ta.style.left = '-9999px';
      ta.style.top = '0';
      document.body.appendChild(ta);
      ta.select();
      ta.setSelectionRange(0, ta.value.length);
      let ok = false;
      try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
      document.body.removeChild(ta);
      return ok;
    }

    async function generateLinksFromModal() {
      const ids = Array.from(document.querySelectorAll('.link-cb:checked')).map(cb => Number(cb.value)).filter(Boolean);
      if (!ids.length) return toast('请至少选择一个账号');
      const res = await fetch(`${apiUrl}?action=generate_links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ account_ids: ids })
      });
      const json = await res.json();
      if (json.code !== 0) return toast(json.msg || '生成失败');
      const links = (json.data && json.data.links) || [];
      document.getElementById('linkResult').innerHTML = links.map((link, idx) => `
        <div class="detail-box">
          <div class="mini-item"><span>${esc(link.remark)}</span><strong><button type="button" class="page-btn copy-link-btn" data-copy="${esc(link.url)}" data-idx="${idx}">复制链接</button></strong></div>
          <div class="meta" style="margin-top:8px; word-break: break-all;">${esc(link.url)}</div>
        </div>
      `).join('');
      document.querySelectorAll('.copy-link-btn').forEach(btn => btn.addEventListener('click', async () => {
        const ok = await copyText(btn.dataset.copy);
        toast(ok ? '链接已复制' : '复制失败，请手动选择链接');
      }));
    }

    async function request(action, params = {}) {
      const url = new URL(apiUrl, window.location.href);
      if (action) url.searchParams.set('action', action);
      Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
      const res = await fetch(url.toString(), { credentials: 'same-origin' });
      const json = await res.json();
      if (json.code !== 0) throw new Error(json.msg || '请求失败');
      return json.data;
    }

    function fmtDelta(n) { return n > 0 ? `+${n}` : String(n); }

    function formatDisplayTime(value) {
      if (!value) return '-';
      const str = String(value).trim();
      const m = str.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})$/);
      if (m) {
        const [, y, mo, d, h, mi, s] = m;
        return `${y}-${mo}-${d} ${h}:${mi}:${s}`;
      }
      const d = new Date(str.replace(' ', 'T'));
      if (Number.isNaN(d.getTime())) return str;
      return new Intl.DateTimeFormat('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      }).format(d).replace(/\//g, '-');
    }

    function buildCalendar(dateStr, marks = []) {
      const d = new Date(dateStr + 'T00:00:00');
      const days = ['日','一','二','三','四','五','六'];
      const start = new Date(d.getFullYear(), d.getMonth(), 1);
      const end = new Date(d.getFullYear(), d.getMonth() + 1, 0);
      const firstDow = start.getDay();
      const cells = [];
      for (let i = 0; i < firstDow; i++) cells.push(null);
      for (let n = 1; n <= end.getDate(); n++) {
        const iso = `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(n).padStart(2,'0')}`;
        cells.push({ day: days[new Date(iso + 'T00:00:00').getDay()], num: n, value: iso, mark: marks.includes(iso) });
      }
      return cells;
    }

    function makeNotifications(data) {
      return [
        { icon: '✓', title: '今日概览', text: `今日总上传 ${data?.overview?.total_success ?? 0}，活跃账号 ${data?.overview?.active_user_count ?? 0}` },
        { icon: '↗', title: '环比变化', text: `较昨日 ${fmtDelta(data?.overview?.delta_success ?? 0)}，变化率 ${data?.overview?.delta_rate_percent ?? '0'}%` },
        { icon: '⚑', title: '提示', text: '点击账号或消息卡可查看详情弹窗。' }
      ];
    }

    function drawTrendChart(rows) {
      const svg = document.getElementById('trendChart');
      const w = 860, h = 220, pad = 22;
      if (!rows.length) {
        svg.innerHTML = '<text x="430" y="110" text-anchor="middle" fill="#9b8f83" font-size="14">暂无趋势数据</text>';
        return;
      }
      const values = rows.map(r => Number(r.total_success || 0));
      const max = Math.max(10, ...values);
      const step = (w - pad * 2) / Math.max(1, rows.length - 1);
      const points = rows.map((r, i) => {
        const x = pad + i * step;
        const y = h - pad - (Number(r.total_success || 0) / max) * (h - pad * 2 - 18);
        return { x, y, label: r.date.slice(5), value: r.total_success };
      });
      const line = points.map((p, i) => `${i===0?'M':'L'} ${p.x} ${p.y}`).join(' ');
      const area = `${line} L ${points[points.length-1]?.x || pad} ${h-pad} L ${points[0]?.x || pad} ${h-pad} Z`;
      svg.innerHTML = `
        <defs>
          <linearGradient id="trendFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="rgba(232,161,95,.42)"/>
            <stop offset="100%" stop-color="rgba(232,161,95,0)"/>
          </linearGradient>
        </defs>
        <rect x="0" y="0" width="860" height="220" rx="18" fill="transparent"></rect>
        <path d="${area}" fill="url(#trendFill)"></path>
        <path d="${line}" fill="none" stroke="rgba(220,132,56,.95)" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></path>
        ${points.map(p => `<g><circle cx="${p.x}" cy="${p.y}" r="5" fill="#fff" stroke="#dc8438" stroke-width="3"></circle><text x="${p.x}" y="${h-8}" text-anchor="middle" fill="#8c8176" font-size="11">${esc(p.label)}</text></g>`).join('')}
      `;
    }

    function drawHourChart(rows) {
      const svg = document.getElementById('hourChart');
      const w = 860, h = 120, pad = 18;
      if (!rows.length) {
        svg.innerHTML = '<text x="430" y="62" text-anchor="middle" fill="#9b8f83" font-size="14">暂无小时数据</text>';
        return;
      }
      const values = rows.map(r => Number(r.total_success || 0));
      const max = Math.max(10, ...values);
      const bw = (w - pad * 2) / rows.length;
      svg.innerHTML = rows.map((r, i) => {
        const val = Number(r.total_success || 0);
        const barH = (val / max) * (h - 36);
        const x = pad + i * bw + 1;
        const y = h - pad - barH;
        const fill = val > 0 ? 'rgba(232,161,95,.92)' : 'rgba(232,161,95,.18)';
        return `<rect x="${x}" y="${y}" width="${Math.max(3, bw - 4)}" height="${barH}" rx="8" fill="${fill}"></rect>`;
      }).join('');
    }

    function renderCalendar() {
      const el = document.getElementById('calendarGrid');
      const cells = state.calendar;
      el.innerHTML = cells.map((d) => d ? `
        <div class="cal-cell ${d.mark ? 'active' : ''} ${String(d.value) === String(state.selectedDate) ? 'selected' : ''}" data-date="${d.value}">
          <small>${esc(d.day)}</small>
          <strong>${esc(d.num)}</strong>
          ${d.mark ? '<span class="dot"></span>' : ''}
        </div>` : '<div class="cal-cell" style="visibility:hidden"></div>').join('');
      el.querySelectorAll('.cal-cell[data-date]').forEach(cell => {
        cell.addEventListener('click', async () => {
          state.selectedDate = cell.dataset.date;
          document.getElementById('calendarLabel').textContent = state.selectedDate;
          document.getElementById('dateInput').value = state.selectedDate;
          await loadDashboard();
          toast(`已切换到 ${state.selectedDate}`);
        });
      });
    }

    function renderNotifications() {
      const menu = document.getElementById('notifMenu');
      menu.innerHTML = '<h3>通知</h3>' + state.notifications.map(n => `
        <div class="menu-item">
          <div class="bullet">${esc(n.icon)}</div>
          <div><strong>${esc(n.title)}</strong><p>${esc(n.text)}</p></div>
        </div>
      `).join('');
    }

    function renderFeed() {
      const el = document.getElementById('feedTimeline');
      const feed = state.feed.slice(0, 6);
      el.innerHTML = feed.length ? feed.map((f, idx) => `
        <div class="timeline-item" data-idx="${idx}">
          <div class="timeline-dot"></div>
          <div>
            <strong>${esc(f.account)} · ${esc(f.success_count)} 次上传</strong>
            <p>${esc(f.remark || '未备注')} · ${esc(formatDisplayTime(f.latest_upload_at || f.updated_at))}</p>
          </div>
        </div>
      `).join('') : '<div class="empty">暂无动态</div>';
      el.querySelectorAll('.timeline-item').forEach((item, idx) => {
        item.addEventListener('click', () => {
          const f = feed[idx];
          const t = formatDisplayTime(f.latest_upload_at || f.updated_at);
          openModal('feed', { title: `${f.account} 上传动态`, sub: t, text: `账号 ${f.account} 在 ${t} 产生了 ${f.success_count} 次成功上传，备注为：${f.remark || '未备注'}。` });
        });
      });
    }

    function renderRemarkRank(allAccounts) {
      const map = {};
      let grandTotal = 0;
      allAccounts.forEach(r => {
        const cnt = Number(r.day_success || 0);
        if (cnt <= 0) return; // 只统计当天有上传的
        const remark = (r.remark || '').trim() || '未备注';
        map[remark] = (map[remark] || 0) + cnt;
        grandTotal += cnt;
      });

      document.getElementById('todayTotalBadge').textContent = `总计 ${grandTotal}`;

      const sorted = Object.entries(map).sort((a, b) => b[1] - a[1]);
      const maxVal = sorted.length ? sorted[0][1] : 1;

      document.getElementById('remarkRankList').innerHTML = sorted.length ? sorted.map(([remark, cnt], idx) => {
        const pct = Math.round((cnt / maxVal) * 100);
        const medal = idx === 0 ? '🥇' : idx === 1 ? '🥈' : idx === 2 ? '🥉' : `${idx + 1}`;
        return `
          <div style="display:grid;grid-template-columns:48px 1fr auto;align-items:center;gap:14px;padding:14px 0;border-bottom:1px solid #f0e6d8;">
            <div style="font-weight:900;font-size:26px;text-align:center;line-height:1;">${medal}</div>
            <div>
              <div style="font-weight:700;font-size:18px;">${esc(remark)}</div>
              <div style="margin-top:7px;height:10px;border-radius:999px;background:#f1e7db;overflow:hidden;">
                <div style="width:${pct}%;height:100%;border-radius:inherit;background:linear-gradient(90deg,var(--orange),#f3c28f);"></div>
              </div>
            </div>
            <div class="pill ${idx === 0 ? 'high' : ''}" style="white-space:nowrap;font-size:16px;padding:8px 16px;">${cnt}</div>
          </div>`;
      }).join('') : '<div class="empty">暂无活跃账号数据</div>';
    }

    function renderOverview(data) {
      const ov = data.overview || {};
      const top = ov.top_account || null;
      const linkAccounts = state.allAccountsForLinks || [];
      const topRemark = top ? (
        top.remark ||
        (state.accounts.find(r => String(r.account) === String(top.account)) || {}).remark ||
        (linkAccounts.find(r => String(r.account) === String(top.account)) || {}).remark ||
        '未备注'
      ) : '';
      document.getElementById('statTotal').textContent = ov.total_success ?? 0;
      document.getElementById('statActive').textContent = ov.active_user_count ?? 0;
      document.getElementById('statAvg').textContent = Number(ov.avg_success_per_user || 0);
      document.getElementById('statDelta').textContent = `较昨日 ${fmtDelta(ov.delta_success || 0)}，变化率 ${ov.delta_rate_percent ?? 0}%`;
      document.getElementById('dateBadge').textContent = data.date || state.selectedDate;
      document.getElementById('calendarLabel').textContent = data.date || state.selectedDate;
      document.getElementById('topAccountBox').innerHTML = top ? `
        <div style="display:flex;align-items:flex-start;justify-content:space-between;gap:12px;">
          <div>
            <div class="meta">第一名账号</div>
            <div style="font-size:18px;font-weight:900;margin-top:6px;">${esc(top.account)}</div>
            <div class="meta" style="margin-top:4px;">${esc(topRemark)}</div>
          </div>
          <div class="pill high">${esc(top.success_count)} 次</div>
        </div>` : '<div class="meta">暂无数据</div>';
    }

    function renderAccountTable(data) {
      const rows = data.rows || [];
      state.accounts = rows;
      document.getElementById('accountSummary').textContent = `共 ${data.total ?? rows.length} 条，当前第 ${data.page || state.page} 页`;
      document.getElementById('pageInfo').textContent = `第 ${data.page || state.page} 页 / 共 ${(data.total && data.page_size) ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1} 页`;
      document.getElementById('prevPage').disabled = (data.page || state.page) <= 1;
      document.getElementById('nextPage').disabled = (data.page || state.page) * (data.page_size || state.pageSize) >= (data.total || 0);

      const tbody = document.getElementById('accountTbody');
      const hint = document.getElementById('accountHint');
      if (!rows.length) {
        hint.style.display = 'block';
        hint.textContent = '接口返回为空或当前筛选条件无数据，请检查日期、关键词和包含 0 上传筛选。';
      } else {
        hint.style.display = 'none';
        hint.textContent = '';
      }
      tbody.innerHTML = rows.length ? rows.map((r, idx) => `
        <tr data-idx="${idx}">
          <td><strong>${esc(r.account)}</strong></td>
          <td title="${esc(r.remark || '未备注')}">${esc(r.remark || '未备注')}</td>
          <td>${esc(data.date || state.selectedDate)}</td>
          <td><span class="pill ${Number(r.day_success) > 0 ? 'ok' : ''}">${esc(r.day_success ?? 0)}</span></td>
          <td>${esc(formatDisplayTime(r.latest_upload_at || '-'))}</td>
          <td>${Number(r.day_success) > 0 ? '活跃' : '静默'}</td>
          <td><button class="page-btn" data-open="${idx}">详情</button></td>
        </tr>
      `).join('') : '<tr><td colspan="7"><div class="empty">没有符合条件的账号数据</div></td></tr>';

      tbody.querySelectorAll('tr[data-idx]').forEach(tr => {
        const idx = Number(tr.dataset.idx);
        tr.addEventListener('click', (e) => {
          if (e.target && e.target.dataset && e.target.dataset.open !== undefined) return;
          openModal('account', rows[idx]);
        });
      });
      tbody.querySelectorAll('button[data-open]').forEach(btn => {
        btn.addEventListener('click', (e) => {
          e.stopPropagation();
          openModal('account', rows[Number(btn.dataset.open)]);
        });
      });

      const keyword = state.keyword.trim().toLowerCase();
      const msgs = rows.filter(r => !keyword || [r.account, r.remark].join(' ').toLowerCase().includes(keyword)).slice(0, 20).map(r => ({
        account: r.account,
        remark: r.remark || '未备注',
        uploads: r.day_success || 0,
        time: formatDisplayTime(r.latest_upload_at || '-'),
        raw: r
      }));
      document.getElementById('messagesContainer').innerHTML = msgs.length ? msgs.map((m, idx) => `
        <div class="feed-card" data-idx="${idx}" style="cursor:pointer;">
          <div class="fc-top">
            <span class="fc-badge">${esc(m.account.slice(-1))}</span>
            <span class="fc-time">${esc(m.time)}</span>
          </div>
          <div class="fc-account" title="${esc(m.account)}">${esc(m.account)}</div>
          <div class="fc-meta">今日上传 <strong style="color:var(--orange);">${m.uploads}</strong> 次<br>备注：${esc(m.remark)}</div>
        </div>
      `).join('') : '<div class="empty" style="padding:16px;">没有消息数据</div>';
      document.querySelectorAll('#messagesContainer .feed-card').forEach((item, idx) => {
        item.addEventListener('click', () => openModal('feed', { title: msgs[idx].account, sub: msgs[idx].remark, text: `账号 ${msgs[idx].account} 今日上传 ${msgs[idx].uploads} 次，最新上传 ${msgs[idx].time}` }));
      });
    }

    function updateCharts(data) {
      state.trend14 = data.trend14 || [];
      state.hour24 = data.hour24 || [];
      drawTrendChart(state.trend14);
      drawHourChart(state.hour24);
    }

    async function loadDashboard(silent = false) {
      const date = state.selectedDate;
      if (!silent) setLiveStatus('同步中…');
      const [proRes, accountsRes, linkAccountsRes] = await Promise.allSettled([
        request('dashboard_pro', { date, rank_page: 1, rank_page_size: 10 }),
        request('accounts', { date, include_zero: state.includeZero, keyword: state.keyword, page: state.page, page_size: state.pageSize }),
        request('accounts', { date, include_zero: 1, keyword: '', page: 1, page_size: 500 })
      ]);

      const pro = proRes.status === 'fulfilled' ? proRes.value : {
        date,
        overview: { total_success: 0, active_user_count: 0, delta_success: 0, delta_rate_percent: 0, avg_success_per_user: 0, top_account: null },
        trend14: [], hour24: [], feed50: [], rank: { rows: [] }
      };
      const accounts = accountsRes.status === 'fulfilled' ? accountsRes.value : {
        date,
        total: 0,
        page: state.page,
        page_size: state.pageSize,
        rows: []
      };

      const linkAccounts = linkAccountsRes.status === 'fulfilled' ? linkAccountsRes.value : accounts;
      state.dashboard = pro;
      state.notifications = makeNotifications(pro);
      state.calendar = buildCalendar(date, (pro.feed50 || []).map(x => x.updated_at ? String(x.updated_at).slice(0, 10) : date).filter(Boolean));
      state.feed = pro.feed50 || [];
      state.allAccountsForLinks = linkAccounts.rows || [];
      renderNotifications();
      renderOverview(pro);
      updateCharts(pro);
      renderCalendar();
      renderFeed();
      renderAccountTable(accounts);
      renderRemarkRank(state.allAccountsForLinks);
      const refreshedAt = new Date();
      state.lastRefreshAt = refreshedAt;
      document.querySelector('.title p').textContent = `今日总上传 ${pro.overview?.total_success || 0}，活跃账号 ${pro.overview?.active_user_count || 0}，实干赚大钱。`;
      document.getElementById('calendarLabel').textContent = date;
      setLiveStatus(`最新上传 ${refreshedAt.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`);
    }

    function bindUi() {
      document.getElementById('searchInput').addEventListener('input', async (e) => {
        state.keyword = e.target.value;
        await loadDashboard();
      });
      document.querySelectorAll('.nav-item[data-section]').forEach(item => {
        item.addEventListener('click', (e) => {
          e.preventDefault();
          document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
          item.classList.add('active');
          const sectionId = item.dataset.section;
          // 切换 section 显示
          document.querySelectorAll('.dashboard-section').forEach(s => {
            s.style.display = s.id === sectionId ? '' : 'none';
          });
          if (sectionId === 'section-settings') {
            initSettingsPage();
          }
        });
      });
      document.getElementById('keywordInput').addEventListener('input', async (e) => {
        state.keyword = e.target.value;
        state.page = 1;
        await loadDashboard();
      });
      document.getElementById('accountZeroFilter').addEventListener('change', async (e) => {
        state.includeZero = e.target.value;
        state.page = 1;
        await loadDashboard();
      });
      document.getElementById('dateInput').addEventListener('change', async (e) => {
        state.selectedDate = e.target.value;
        state.page = 1;
        await loadDashboard();
      });
      document.getElementById('reloadBtn').addEventListener('click', async () => {
        await loadDashboard();
        toast('已刷新数据');
      });
      document.getElementById('collapseBtn').addEventListener('click', () => document.getElementById('sidebar').classList.toggle('collapsed'));
      const notifBtn = document.getElementById('notifBtn');
      const notifMenu = document.getElementById('notifMenu');
      notifBtn.addEventListener('click', (e) => { e.stopPropagation(); notifMenu.classList.toggle('show'); });
      document.addEventListener('click', (e) => { if (!notifMenu.contains(e.target) && !notifBtn.contains(e.target)) notifMenu.classList.remove('show'); });
      document.getElementById('modalClose').addEventListener('click', closeModal);
      document.getElementById('modalBackdrop').addEventListener('click', (e) => { if (e.target.id === 'modalBackdrop') closeModal(); });
      document.addEventListener('keydown', (e) => { if (e.key === 'Escape') { closeModal(); notifMenu.classList.remove('show'); } });
      document.getElementById('prevPage').addEventListener('click', async () => { if (state.page > 1) { state.page -= 1; await loadDashboard(); } });
      document.getElementById('nextPage').addEventListener('click', async () => { state.page += 1; await loadDashboard(); });
      document.getElementById('newAccountBtn').addEventListener('click', () => openModal('new-account'));
      document.getElementById('generateLinkBtn').addEventListener('click', () => openModal('generate-links'));
      document.getElementById('backfillBtn').addEventListener('click', async () => {
        try {
          const res = await fetch(`${apiUrl}?action=backfill_accounts`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ date: state.selectedDate }) });
          const json = await res.json();
          toast(json.msg || '补齐完成');
          await loadDashboard();
        } catch (err) { toast('补齐失败'); }
      });
      document.getElementById('exportBtn').addEventListener('click', () => {
        const rows = state.accounts.map(r => [r.account, r.remark, r.day_success, r.updated_at].join('\t')).join('\n');
        navigator.clipboard.writeText(rows || '');
        toast('当前页已复制到剪贴板');
      });
    }

    bindUi();
    loadDashboard().catch(err => {
      console.error(err);
      toast('加载失败，显示为空状态');
      document.getElementById('accountTbody').innerHTML = '<tr><td colspan="6"><div class="empty">接口加载失败</div></td></tr>';
    });
    scheduleAutoRefresh();
    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) {
        loadDashboard(true).catch(console.error);
      }
    });

    /* ===========================
       设置页 JS
    =========================== */
    const settingsState = { accPage: 1, accPages: 1, remarkFilter: '', selected: new Set() };

    async function settingsApi(action, params = {}, body = null) {
      const url = new URL(window.location.href);
      url.search = '';
      url.searchParams.set('settings_action', action);
      Object.entries(params).forEach(([k,v]) => url.searchParams.set(k, v));
      const opt = { credentials: 'same-origin' };
      if (body) { opt.method = 'POST'; opt.headers = {'Content-Type':'application/json'}; opt.body = JSON.stringify(body); }
      const res = await fetch(url.toString(), opt);
      return res.json();
    }

    async function initSettingsPage() {
      settingsState.remarkFilter = '';
      settingsState.selected = new Set();
      loadAccounts(1);
    }

    async function saveDbConfig() {}
    async function testDbConn() {}
    function showAddRemarkModal() {}
    function showAddAccountModal() {}

    // 按备注关键词搜索
    function searchByRemark() {
      const kw = (document.getElementById('remarkSearchInput')?.value || '').trim();
      settingsState.remarkFilter = kw;
      settingsState.selected = new Set();
      loadAccounts(1);
    }

    // 清除备注过滤，显示全部
    function clearRemarkFilter() {
      document.getElementById('remarkSearchInput').value = '';
      settingsState.remarkFilter = '';
      settingsState.selected = new Set();
      loadAccounts(1);
    }

    // 加载账号列表
    async function loadAccounts(page) {
      page = Math.max(1, Math.min(page, settingsState.accPages || 1));
      settingsState.accPage = page;
      const kw = settingsState.remarkFilter;
      const tbody = document.getElementById('accTbody');
      tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;padding:16px;color:var(--muted);">加载中…</td></tr>';
      try {
        const params = { page };
        if (kw) params.kw = kw;
        const r = await settingsApi('list_accounts', params);
        if (r.code !== 0) {
          tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;padding:16px;color:var(--red);">错误：${esc(r.msg)}</td></tr>`;
          return;
        }
        const { list, total, pages } = r.data;
        settingsState.accPages = pages || 1;
        document.getElementById('accPageInfo').textContent = `第 ${page} / ${pages||1} 页，共 ${total} 条`;
        if (!list.length) {
          tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;padding:20px;color:var(--muted);">暂无账号数据</td></tr>';
          updateBatchBar();
          return;
        }
        tbody.innerHTML = list.map(row => `<tr id="acc-row-${row.id}">
          <td style="text-align:center;">
            <input type="checkbox" class="acc-chk" data-id="${row.id}" data-account="${esc(row.account)}"
              ${settingsState.selected.has(String(row.id)) ? 'checked' : ''}
              onchange="onRowCheck(this)">
          </td>
          <td class="acc-cell" title="${esc(row.account)}" style="font-size:13px;max-width:260px;overflow:hidden;text-overflow:ellipsis;">${esc(row.account)}</td>
          <td style="max-width:140px;overflow:hidden;text-overflow:ellipsis;">
            <span style="color:${row.remark ? 'var(--text)' : 'var(--muted)'};">${esc(row.remark || '未备注')}</span>
            <button class="s-btn" style="margin-left:4px;" data-id="${row.id}" data-remark="${esc(row.remark||'')}" onclick="editAccRemark(+this.dataset.id,this.dataset.remark)">改</button>
          </td>
          <td style="color:var(--muted);font-size:12px;">${esc((row.created_at||'').slice(0,16))}</td>
          <td style="text-align:center;">
            <button class="s-btn danger" data-id="${row.id}" data-account="${esc(row.account)}" onclick="deleteAccount(+this.dataset.id,this.dataset.account)">删除</button>
          </td>
        </tr>`).join('');
        updateSelectAllChk();
        updateBatchBar();
      } catch(e) {
        tbody.innerHTML = `<tr><td colspan="5" style="text-align:center;padding:16px;color:var(--red);">连接失败：${esc(e.message)}</td></tr>`;
      }
    }

    function onRowCheck(chk) {
      const id = String(chk.dataset.id);
      if (chk.checked) settingsState.selected.add(id);
      else settingsState.selected.delete(id);
      updateSelectAllChk();
      updateBatchBar();
    }

    function toggleSelectAll(checked) {
      document.querySelectorAll('.acc-chk').forEach(chk => {
        chk.checked = checked;
        const id = String(chk.dataset.id);
        if (checked) settingsState.selected.add(id);
        else settingsState.selected.delete(id);
      });
      updateBatchBar();
    }

    function updateSelectAllChk() {
      const all = document.querySelectorAll('.acc-chk');
      const chked = document.querySelectorAll('.acc-chk:checked');
      const el = document.getElementById('selectAllChk');
      if (!el) return;
      el.checked = all.length > 0 && chked.length === all.length;
      el.indeterminate = chked.length > 0 && chked.length < all.length;
    }

    function updateBatchBar() {
      const bar = document.getElementById('batchBar');
      const cnt = settingsState.selected.size;
      bar.style.display = cnt > 0 ? 'flex' : 'none';
      document.getElementById('batchCount').textContent = `已选 ${cnt} 条`;
    }

    function clearSelection() {
      settingsState.selected = new Set();
      document.querySelectorAll('.acc-chk').forEach(c => c.checked = false);
      const el = document.getElementById('selectAllChk');
      if (el) { el.checked = false; el.indeterminate = false; }
      updateBatchBar();
    }

    async function batchChangeRemark() {
      if (!settingsState.selected.size) return;
      const newRemark = prompt(`批量修改 ${settingsState.selected.size} 个账号的备注为：`, '');
      if (newRemark === null) return;
      const ids = [...settingsState.selected];
      let ok = 0;
      await Promise.all(ids.map(async id => {
        const r = await settingsApi('update_account_remark', {}, { id: parseInt(id), remark: newRemark.trim() });
        if (r.code === 0) ok++;
      }));
      toast(`已更新 ${ok} 个账号的备注`);
      clearSelection();
      loadAccounts(settingsState.accPage);
    }

    async function batchDelete() {
      if (!settingsState.selected.size) return;
      if (!confirm(`确认删除选中的 ${settingsState.selected.size} 个账号？此操作不可恢复！`)) return;
      const ids = [...settingsState.selected];
      let ok = 0;
      await Promise.all(ids.map(async id => {
        const r = await settingsApi('delete_account', {}, { id: parseInt(id) });
        if (r.code === 0) ok++;
      }));
      toast(`已删除 ${ok} 个账号`);
      clearSelection();
      loadAccounts(Math.min(settingsState.accPage, settingsState.accPages));
    }

    async function editAccRemark(id, oldRemark) {
      const newRemark = prompt('修改备注：', oldRemark);
      if (newRemark === null) return;
      const r = await settingsApi('update_account_remark', {}, { id, remark: newRemark.trim() });
      toast(r.code === 0 ? '备注已更新' : r.msg);
      if (r.code === 0) loadAccounts(settingsState.accPage);
    }

    async function deleteAccount(id, account) {
      if (!confirm(`确认删除账号「${account}」？此操作不可恢复！`)) return;
      const r = await settingsApi('delete_account', {}, { id });
      toast(r.code === 0 ? '已删除' : r.msg);
      if (r.code === 0) {
        settingsState.selected.delete(String(id));
        updateBatchBar();
        loadAccounts(settingsState.accPage);
      }
    }

    function loadRemarks() {} // 兼容旧调用，空实现
    /* ===========================
       /设置页 JS
    =========================== */

    /* ===========================
       我的任务 JS
    =========================== */
    const taskStore = {
      list: JSON.parse(localStorage.getItem('tasks_v1') || '[]'),
      save() { localStorage.setItem('tasks_v1', JSON.stringify(this.list)); }
    };

    function renderTasks() {
      ['todo','doing','done'].forEach(status => {
        const items = taskStore.list.filter(t => t.status === status);
        document.getElementById('tcnt-' + status).textContent = items.length;
        document.getElementById('task-list-' + status).innerHTML = items.length ? items.map((t,i) => `
          <div class="task-item" data-id="${t.id}">
            <div style="display:flex;align-items:flex-start;justify-content:space-between;gap:8px;">
              <div class="task-item-title">${esc(t.title)}</div>
              <button onclick="deleteTask(${t.id})" style="border:0;background:transparent;cursor:pointer;color:var(--muted);font-size:14px;padding:0;line-height:1;">✕</button>
            </div>
            <div class="task-item-meta">
              <span class="task-tag ${t.priority === 'urgent' ? 'urgent' : t.priority === 'low' ? 'low' : 'normal'}">${t.priority === 'urgent' ? '紧急' : t.priority === 'low' ? '低优' : '普通'}</span>
              <span>${t.date || '无截止'}</span>
            </div>
            <div class="task-progress-bar"><span style="width:${t.progress||0}%"></span></div>
            <div class="task-item-meta" style="margin-top:4px;">
              ${status !== 'todo' ? `<button class="page-btn" style="height:24px;font-size:11px;" onclick="moveTask(${t.id},'todo')">待办</button>` : ''}
              ${status !== 'doing' ? `<button class="page-btn" style="height:24px;font-size:11px;" onclick="moveTask(${t.id},'doing')">进行中</button>` : ''}
              ${status !== 'done' ? `<button class="page-btn" style="height:24px;font-size:11px;" onclick="moveTask(${t.id},'done')">完成</button>` : ''}
            </div>
          </div>`).join('') : `<div class="empty" style="text-align:center;font-size:13px;padding:12px;">暂无任务</div>`;
      });
      const total = taskStore.list.length;
      const doing = taskStore.list.filter(t=>t.status==='doing').length;
      const done = taskStore.list.filter(t=>t.status==='done').length;
      document.getElementById('taskTotalCnt').textContent = total;
      document.getElementById('taskDoingCnt').textContent = doing;
      document.getElementById('taskDoneCnt').textContent = done;
    }

    function quickAddTask() {
      const inp = document.getElementById('quickTaskInput');
      const title = inp.value.trim();
      if (!title) return;
      taskStore.list.unshift({ id: Date.now(), title, status: 'todo', priority: 'normal', date: '', progress: 0 });
      taskStore.save();
      inp.value = '';
      renderTasks();
    }

    function deleteTask(id) {
      taskStore.list = taskStore.list.filter(t => t.id !== id);
      taskStore.save();
      renderTasks();
    }

    function moveTask(id, status) {
      const t = taskStore.list.find(t => t.id === id);
      if (t) { t.status = status; if (status==='done') t.progress=100; taskStore.save(); renderTasks(); }
    }

    function openAddTaskModal() {
      const kicker = document.getElementById('modalKicker');
      const title = document.getElementById('modalTitle');
      const subtitle = document.getElementById('modalSubtitle');
      const body = document.getElementById('modalBody');
      kicker.textContent = '新建任务';
      title.textContent = '创建任务';
      subtitle.textContent = '填写任务信息后点击保存';
      body.innerHTML = `
        <div class="detail-box">
          <div class="mini-list" style="gap:12px;">
            <label class="mini-item" style="align-items:center;gap:12px;"><span style="min-width:64px;">任务名称</span><input id="newTaskTitle" class="control" style="width:100%;" placeholder="输入任务名称…"></label>
            <label class="mini-item" style="align-items:center;gap:12px;"><span style="min-width:64px;">优先级</span>
              <select id="newTaskPriority" class="control" style="width:100%;">
                <option value="normal">普通</option>
                <option value="urgent">紧急</option>
                <option value="low">低优先级</option>
              </select>
            </label>
            <label class="mini-item" style="align-items:center;gap:12px;"><span style="min-width:64px;">截止日期</span><input id="newTaskDate" class="control" type="date" style="width:100%;"></label>
            <label class="mini-item" style="align-items:center;gap:12px;"><span style="min-width:64px;">初始进度</span><input id="newTaskProgress" class="control" type="range" min="0" max="100" value="0" style="width:100%;"> <span id="taskProgressLabel">0%</span></label>
          </div>
          <div style="margin-top:16px;display:flex;gap:10px;">
            <button id="saveNewTaskBtn" class="page-btn">保存任务</button>
          </div>
        </div>`;
      document.getElementById('newTaskProgress').oninput = (e) => { document.getElementById('taskProgressLabel').textContent = e.target.value + '%'; };
      document.getElementById('saveNewTaskBtn').onclick = () => {
        const title2 = document.getElementById('newTaskTitle').value.trim();
        if (!title2) return toast('请填写任务名称');
        taskStore.list.unshift({
          id: Date.now(),
          title: title2,
          status: 'todo',
          priority: document.getElementById('newTaskPriority').value,
          date: document.getElementById('newTaskDate').value,
          progress: parseInt(document.getElementById('newTaskProgress').value) || 0
        });
        taskStore.save();
        renderTasks();
        closeModal();
        toast('任务已创建');
      };
      document.getElementById('modalBackdrop').classList.add('show');
    }

    /* ===========================
       日历 JS
    =========================== */
    const calState = {
      year: new Date().getFullYear(),
      month: new Date().getMonth(),
      selected: today,
      events: [],
      loading: false,
      loadedMonthKey: ''
    };

    const calColors = ['#e8a15f','#6da98d','#3a8fd4','#8b5cf6','#de7d67'];

    function getCalendarMonthKey() {
      return `${calState.year}-${String(calState.month + 1).padStart(2, '0')}`;
    }

    async function calendarRequest(action, options = {}) {
      const { method = 'GET', body = null, params = {} } = options;
      const url = new URL(calendarApiUrl, window.location.href);
      url.searchParams.set('calendar_action', action);
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          url.searchParams.set(key, value);
        }
      });

      const requestOptions = {
        method,
        credentials: 'same-origin'
      };
      if (body !== null) {
        requestOptions.headers = { 'Content-Type': 'application/json' };
        requestOptions.body = JSON.stringify(body);
      }

      const res = await fetch(url.toString(), requestOptions);
      const json = await res.json();
      if (json.code !== 0) {
        throw new Error(json.msg || '日历请求失败');
      }
      return json.data;
    }

    async function loadCalendarEvents(force = false) {
      const monthKey = getCalendarMonthKey();
      if (!force && calState.loadedMonthKey === monthKey && Array.isArray(calState.events)) {
        return calState.events;
      }

      calState.loading = true;
      try {
        const data = await calendarRequest('list', { params: { month: monthKey } });
        calState.events = Array.isArray(data?.list) ? data.list : [];
        calState.loadedMonthKey = monthKey;
      } finally {
        calState.loading = false;
      }
      return calState.events;
    }

    async function refreshCalendarView(force = false, silent = false) {
      try {
        await loadCalendarEvents(force);
      } catch (e) {
        console.error('calendar load failed', e);
        if (!silent) toast(e.message || '日历加载失败');
      }
      renderFullCalendar();
    }

    function renderFullCalendar() {
      const dayNames = ['日','一','二','三','四','五','六'];
      const labelsEl = document.getElementById('calDayLabels');
      if (labelsEl) labelsEl.innerHTML = dayNames.map(d=>`<div class="cal-day-label">${d}</div>`).join('');

      const title = document.getElementById('calMonthTitle');
      if (!title) return;
      title.textContent = `${calState.year} 年 ${calState.month+1} 月`;

      const firstDay = new Date(calState.year, calState.month, 1).getDay();
      const daysInMonth = new Date(calState.year, calState.month+1, 0).getDate();
      const prevDays = new Date(calState.year, calState.month, 0).getDate();

      const cells = [];
      for (let i=0; i<firstDay; i++) cells.push({day: prevDays-firstDay+1+i, cur:false});
      for (let d=1; d<=daysInMonth; d++) cells.push({day:d, cur:true});
      const remain = 42 - cells.length;
      for (let i=1; i<=remain; i++) cells.push({day:i, cur:false});

      const grid = document.getElementById('calFullGrid');
      if (!grid) return;
      grid.innerHTML = cells.map(cell => {
        if (!cell.cur) return `<div class="cal-full-cell other-month"><div class="cal-num" style="color:var(--muted);">${cell.day}</div></div>`;
        const iso = `${calState.year}-${String(calState.month+1).padStart(2,'0')}-${String(cell.day).padStart(2,'0')}`;
        const evs = calState.events.filter(e=>e.date===iso);
        const isToday = iso === today;
        const isSel = iso === calState.selected;
        return `<div class="cal-full-cell ${isToday?'today':''} ${isSel?'selected':''}" data-date="${iso}">
          <div class="cal-num">${cell.day}</div>
          ${evs.slice(0,2).map(e=>`<div class="cal-event-mini" style="background:${e.color}22;color:${e.color};">${esc(e.title)}</div>`).join('')}
          ${evs.length>2?`<div style="font-size:10px;color:var(--muted);">+${evs.length-2}</div>`:''}
        </div>`;
      }).join('');

      grid.querySelectorAll('.cal-full-cell[data-date]').forEach(el => {
        el.addEventListener('click', () => {
          calState.selected = el.dataset.date;
          renderFullCalendar();
          renderCalEventList();
        });
      });
      renderCalEventList();
    }

    function renderCalEventList() {
      const label = document.getElementById('calSelectedLabel');
      const listEl = document.getElementById('calEventList');
      if (!label || !listEl) return;
      label.textContent = `${calState.selected} 的事件`;
      const evs = calState.events.filter(e=>e.date===calState.selected);
      if (calState.loading && !evs.length) {
        listEl.innerHTML = '<div class="empty">日历事件加载中…</div>';
        return;
      }
      listEl.innerHTML = evs.length ? evs.map(e=>`
        <div class="cal-event-item">
          <div class="cal-event-color" style="background:${e.color};"></div>
          <div class="cal-event-info">
            <strong>${esc(e.title)}</strong>
            <span>${esc(e.time||'全天')} ${e.desc?'· '+esc(e.desc):''}</span>
          </div>
          <button onclick="deleteCalEvent(${e.id})" style="border:0;background:transparent;cursor:pointer;color:var(--muted);font-size:16px;padding:4px;">✕</button>
        </div>`).join('') : '<div class="empty">该日期暂无事件，点击右上角添加。</div>';
    }

    async function deleteCalEvent(id) {
      try {
        await calendarRequest('delete', { method: 'POST', body: { id } });
        await refreshCalendarView(true, true);
        toast('事件已删除');
      } catch (e) {
        toast(e.message || '删除失败');
      }
    }

    async function calNav(delta) {
      calState.month += delta;
      if (calState.month < 0) { calState.month = 11; calState.year--; }
      if (calState.month > 11) { calState.month = 0; calState.year++; }
      calState.events = [];
      calState.loadedMonthKey = '';
      renderFullCalendar();
      await refreshCalendarView(true, true);
    }

    function openAddEventModal() {
      document.getElementById('modalKicker').textContent = '添加事件';
      document.getElementById('modalTitle').textContent = '新建日历事件';
      document.getElementById('modalSubtitle').textContent = '填写事件信息后保存';
      document.getElementById('modalBody').innerHTML = `
        <div class="detail-box">
          <div class="add-event-form">
            <input id="evTitle" placeholder="事件标题…">
            <input id="evDate" type="date" value="${calState.selected}">
            <input id="evTime" type="time" placeholder="时间（可留空=全天）">
            <textarea id="evDesc" placeholder="备注/描述（可选）"></textarea>
            <select id="evColor">
              <option value="#e8a15f">🟠 橙色</option>
              <option value="#6da98d">🟢 绿色</option>
              <option value="#3a8fd4">🔵 蓝色</option>
              <option value="#8b5cf6">🟣 紫色</option>
              <option value="#de7d67">🔴 红色</option>
            </select>
          </div>
          <div style="margin-top:16px;display:flex;gap:10px;">
            <button id="saveEvBtn" class="page-btn">保存事件</button>
          </div>
        </div>`;
      document.getElementById('saveEvBtn').onclick = async () => {
        const title2 = document.getElementById('evTitle').value.trim();
        if (!title2) return toast('请填写事件标题');
        try {
          await calendarRequest('create', {
            method: 'POST',
            body: {
              title: title2,
              date: document.getElementById('evDate').value,
              time: document.getElementById('evTime').value,
              desc: document.getElementById('evDesc').value.trim(),
              color: document.getElementById('evColor').value
            }
          });
          calState.selected = document.getElementById('evDate').value || calState.selected;
          await refreshCalendarView(true, true);
          closeModal();
          toast('事件已保存');
        } catch (e) {
          toast(e.message || '保存失败');
        }
      };
      document.getElementById('modalBackdrop').classList.add('show');
    }

    /* ===========================
       时间管理 JS (番茄钟 + 时间块)
    =========================== */
    const pomo = {
      modes: { focus:25*60, short:5*60, long:15*60 },
      mode: 'focus',
      remaining: 25*60,
      total: 25*60,
      running: false,
      timer: null,
      todayFocus: parseInt(localStorage.getItem('pomo_today_focus')||'0'),
      todayMins: parseInt(localStorage.getItem('pomo_today_mins')||'0'),
      log: JSON.parse(localStorage.getItem('pomo_log')||'[]')
    };

    const tbStore = {
      list: JSON.parse(localStorage.getItem('timeblocks_v1')||'[]'),
      save() { localStorage.setItem('timeblocks_v1', JSON.stringify(this.list)); }
    };

    function setPomodoroMode(mode) {
      pomo.mode = mode;
      pomo.remaining = pomo.total = pomo.modes[mode];
      if (pomo.running) { clearInterval(pomo.timer); pomo.running = false; }
      document.getElementById('pomodoroStartBtn').textContent = '开始';
      ['focus','short','long'].forEach(m => document.getElementById('pMode-'+m)?.classList.toggle('active', m===mode));
      const labels = {focus:'专注',short:'短休',long:'长休'};
      document.getElementById('pomodoroLabel').textContent = labels[mode];
      updatePomodoroDisplay();
    }

    function updatePomodoroDisplay() {
      const m = Math.floor(pomo.remaining/60), s = pomo.remaining%60;
      document.getElementById('pomodoroDisplay').textContent = `${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
      const arc = document.getElementById('pomodoroArc');
      const perim = 534;
      arc.setAttribute('stroke-dashoffset', String(perim * (1 - pomo.remaining/pomo.total)));
      const statusEl = document.getElementById('pomodoroStatus');
      if (statusEl) statusEl.textContent = pomo.running ? (pomo.mode==='focus'?'专注中':'休息中') : '已暂停';
    }

    function togglePomodoro() {
      if (pomo.running) {
        clearInterval(pomo.timer);
        pomo.running = false;
        document.getElementById('pomodoroStartBtn').textContent = '继续';
      } else {
        pomo.running = true;
        document.getElementById('pomodoroStartBtn').textContent = '暂停';
        pomo.timer = setInterval(() => {
          pomo.remaining--;
          updatePomodoroDisplay();
          if (pomo.remaining <= 0) {
            clearInterval(pomo.timer);
            pomo.running = false;
            document.getElementById('pomodoroStartBtn').textContent = '开始';
            if (pomo.mode === 'focus') {
              pomo.todayFocus++;
              pomo.todayMins += 25;
              localStorage.setItem('pomo_today_focus', pomo.todayFocus);
              localStorage.setItem('pomo_today_mins', pomo.todayMins);
              pomo.log.unshift({ label:`专注 25min`, time: new Date().toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit'}) });
              if (pomo.log.length>10) pomo.log.length=10;
              localStorage.setItem('pomo_log', JSON.stringify(pomo.log));
              renderPomodoroLog();
              updateTimeStats();
            }
            toast(pomo.mode==='focus'?'🎉 专注完成！去休息一下吧':'休息结束，继续专注！');
            setPomodoroMode(pomo.mode==='focus'?'short':'focus');
          }
        }, 1000);
      }
      updatePomodoroDisplay();
    }

    function resetPomodoro() {
      clearInterval(pomo.timer);
      pomo.running = false;
      pomo.remaining = pomo.total = pomo.modes[pomo.mode];
      document.getElementById('pomodoroStartBtn').textContent = '开始';
      updatePomodoroDisplay();
    }

    function renderPomodoroLog() {
      const el = document.getElementById('pomodoroLog');
      if (!el) return;
      el.innerHTML = pomo.log.length ? pomo.log.map(l=>`<div class="pomodoro-log-item"><span>✓ ${esc(l.label)}</span><span>${esc(l.time)}</span></div>`).join('') : '';
    }

    function addTimeblock() {
      const name = document.getElementById('tbName').value.trim();
      const start = document.getElementById('tbStart').value;
      const end = document.getElementById('tbEnd').value;
      if (!name || !start || !end) return toast('请填写任务名称和时间段');
      tbStore.list.push({ id: Date.now(), name, start, end, color: document.getElementById('tbColor').value });
      tbStore.list.sort((a,b) => a.start.localeCompare(b.start));
      tbStore.save();
      renderTimeblocks();
      document.getElementById('tbName').value = '';
      updateTimeStats();
    }

    function deleteTimeblock(id) {
      tbStore.list = tbStore.list.filter(t=>t.id!==id);
      tbStore.save();
      renderTimeblocks();
      updateTimeStats();
    }

    function renderTimeblocks() {
      const el = document.getElementById('timeblockList');
      if (!el) return;
      el.innerHTML = tbStore.list.length ? tbStore.list.map(t=>`
        <div class="timeblock-item">
          <div class="timeblock-color" style="background:${t.color};"></div>
          <div class="timeblock-info">
            <strong>${esc(t.name)}</strong>
            <span>${esc(t.start)} – ${esc(t.end)}</span>
          </div>
          <button class="timeblock-del" onclick="deleteTimeblock(${t.id})">✕</button>
        </div>`).join('') : '<div class="empty">暂无时间块，在上方添加</div>';
    }

    function updateTimeStats() {
      const focusEl = document.getElementById('tmTodayFocus');
      const minsEl = document.getElementById('tmTodayMins');
      const blockEl = document.getElementById('tmBlockCnt');
      const planEl = document.getElementById('tmPlanHrs');
      if (focusEl) focusEl.textContent = pomo.todayFocus;
      if (minsEl) minsEl.textContent = pomo.todayMins;
      if (blockEl) blockEl.textContent = tbStore.list.length;
      if (planEl) {
        const totalMin = tbStore.list.reduce((acc,t)=>{
          const [h1,m1]=t.start.split(':').map(Number), [h2,m2]=t.end.split(':').map(Number);
          return acc + Math.max(0, (h2*60+m2)-(h1*60+m1));
        },0);
        planEl.textContent = (totalMin/60).toFixed(1)+'h';
      }
    }

    /* ===========================
       报表 JS
    =========================== */
    function renderReportPage() {
      const days = parseInt(document.getElementById('reportRangeSelect')?.value || '14');
      const trend = state.trend14 || [];

      // KPI
      const slice = trend.slice(-days);
      const total = slice.reduce((s,r)=>s+Number(r.total_success||0), 0);
      const avgDay = slice.length ? Math.round(total/slice.length) : 0;
      let peak = 0, peakDate = '';
      slice.forEach(r => { if (Number(r.total_success||0) > peak) { peak=Number(r.total_success||0); peakDate=r.date||''; } });
      const setEl = (id, v) => { const el=document.getElementById(id); if(el) el.textContent=v; };
      setEl('rptTotal', total.toLocaleString());
      setEl('rptAvgDay', avgDay);
      setEl('rptPeak', peak);
      setEl('rptPeakDate', peakDate?peakDate.slice(5):'');

      // 活跃账号均值（来自 state.dashboard）
      const avgAcc = state.dashboard?.overview?.active_user_count || 0;
      setEl('rptAvgAccount', avgAcc);

      // 趋势柱状图 SVG
      const barSvg = document.getElementById('rptBarChart');
      if (barSvg && slice.length) {
        const W=560, H=200, padX=30, padY=16;
        const vals = slice.map(r=>Number(r.total_success||0));
        const maxV = Math.max(1,...vals);
        const bw = (W-padX*2)/slice.length;
        barSvg.innerHTML = vals.map((v,i)=>{
          const barH = Math.max(3, (v/maxV)*(H-padY*2-20));
          const x = padX + i*bw + bw*0.1;
          const y = H-padY-barH-16;
          const label = (slice[i].date||'').slice(5);
          return `<rect x="${x}" y="${y}" width="${bw*0.8}" height="${barH}" rx="4" fill="rgba(232,161,95,.85)"></rect>
            <text x="${x+bw*0.4}" y="${H-padY}" text-anchor="middle" font-size="9" fill="#9b8f83">${slice.length<=14?esc(label):''}</text>`;
        }).join('');
      } else if (barSvg) {
        barSvg.innerHTML = '<text x="280" y="100" text-anchor="middle" fill="#9b8f83" font-size="14">暂无数据</text>';
      }

      // 备注排行条形图
      const remarkBarEl = document.getElementById('rptRemarkBar');
      if (remarkBarEl) {
        const map = {};
        (state.allAccountsForLinks||[]).forEach(r=>{
          const cnt=Number(r.day_success||0);
          if(!cnt) return;
          const remark=(r.remark||'').trim()||'未备注';
          map[remark]=(map[remark]||0)+cnt;
        });
        const sorted = Object.entries(map).sort((a,b)=>b[1]-a[1]).slice(0,8);
        const maxVal = sorted.length ? sorted[0][1] : 1;
        remarkBarEl.innerHTML = sorted.length ? sorted.map(([remark,cnt])=>`
          <div class="bar-row">
            <div class="bar-label" title="${esc(remark)}">${esc(remark)}</div>
            <div class="bar-track"><div class="bar-fill" style="width:${Math.round(cnt/maxVal*100)}%;"><span>${cnt}</span></div></div>
          </div>`).join('') : '<div class="empty">暂无数据</div>';
      }

      // 热力图
      const heatEl = document.getElementById('rptHeatGrid');
      const heatAxis = document.getElementById('rptHeatAxis');
      if (heatEl && trend.length) {
        const recent = trend.slice(-28);
        const maxH = Math.max(1,...recent.map(r=>Number(r.total_success||0)));
        heatEl.innerHTML = recent.map(r=>{
          const v=Number(r.total_success||0);
          const lvl = v===0?0:v<maxH*0.25?1:v<maxH*0.5?2:v<maxH*0.75?3:4;
          return `<div class="heat-cell h${lvl}" title="${r.date}: ${v}次"></div>`;
        }).join('');
        if (heatAxis) {
          const first = recent[0]?.date?.slice(5)||'';
          const last = recent[recent.length-1]?.date?.slice(5)||'';
          heatAxis.innerHTML = `<span>${esc(first)}</span><span>${esc(last)}</span>`;
        }
      }

      // 饼图：活跃 vs 静默
      const pieSvg = document.getElementById('rptPieChart');
      const pieLegend = document.getElementById('rptPieLegend');
      if (pieSvg) {
        const all = state.allAccountsForLinks||[];
        const active = all.filter(r=>Number(r.day_success||0)>0).length;
        const silent = all.length - active;
        const total2 = active + silent;
        if (total2 === 0) {
          pieSvg.innerHTML = '<circle cx="75" cy="75" r="60" fill="var(--line)"/><text x="75" y="79" text-anchor="middle" font-size="12" fill="var(--muted)">暂无</text>';
        } else {
          const pct = active/total2;
          const r=60, cx=75, cy=75;
          const angle = pct*2*Math.PI;
          const x1=cx+r*Math.sin(0), y1=cy-r*Math.cos(0);
          const x2=cx+r*Math.sin(angle), y2=cy-r*Math.cos(angle);
          const large = pct>0.5?1:0;
          if (pct>=1) {
            pieSvg.innerHTML = `<circle cx="${cx}" cy="${cy}" r="${r}" fill="#6da98d"/>`;
          } else if (pct<=0) {
            pieSvg.innerHTML = `<circle cx="${cx}" cy="${cy}" r="${r}" fill="var(--line)"/>`;
          } else {
            pieSvg.innerHTML = `
              <circle cx="${cx}" cy="${cy}" r="${r}" fill="var(--line)"/>
              <path d="M${cx},${cy} L${x1},${y1} A${r},${r} 0 ${large},1 ${x2},${y2} Z" fill="#6da98d"/>
              <circle cx="${cx}" cy="${cy}" r="36" fill="var(--panel,#fff)"/>
              <text x="${cx}" y="${cy-4}" text-anchor="middle" font-size="18" font-weight="900" fill="var(--text)">${Math.round(pct*100)}%</text>
              <text x="${cx}" y="${cy+14}" text-anchor="middle" font-size="11" fill="var(--muted)">活跃</text>`;
          }
        }
        if (pieLegend) pieLegend.innerHTML = `
          <div class="pie-legend-item"><div class="pie-dot" style="background:#6da98d;"></div><span>活跃 ${active} 个</span></div>
          <div class="pie-legend-item"><div class="pie-dot" style="background:var(--line);"></div><span>静默 ${silent} 个</span></div>`;
      }
    }

    // 初始化各 section 内容（切换时触发）
    const _origNavHandler = null;
    document.querySelectorAll('.nav-item[data-section]').forEach(item => {
      item.addEventListener('click', () => {
        const sid = item.dataset.section;
        if (sid === 'section-tasks') renderTasks();
        if (sid === 'section-calendar') refreshCalendarView(true, true);
        if (sid === 'section-time') { updatePomodoroDisplay(); renderPomodoroLog(); renderTimeblocks(); updateTimeStats(); }
        if (sid === 'section-reports') renderReportPage();
      });
    });

    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) {
        refreshCalendarView(true, true);
      }
    });

    // 页面初始化时预渲染
    renderTasks();
    refreshCalendarView(true, true);
    updatePomodoroDisplay();
    renderPomodoroLog();
    renderTimeblocks();
    updateTimeStats();

  </script>
</body>
</html>
