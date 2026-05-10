<?php
/*
 * 解密代理服务器
 * - POST /decrypt_proxy.php          → App上传加密数据，解密后转发到任务服务器
 * - GET  /decrypt_proxy.php?act=view → 查看转发日志前端页面
 * - GET  /decrypt_proxy.php?act=logs → JSON格式返回日志
 * - GET  /decrypt_proxy.php?act=clear→ 清空日志
 */

$AES_KEY  = '25d9a1907ab38e273c4f57c476e64377';
$AUTH_KEY = 'gejie2026';  // 前端查看密码
$LOG_FILE = __DIR__ . '/decrypt_proxy_logs.jsonl';
$MAX_LOGS = 500;

$TARGETS = [
    1 => [
        'url'  => 'https://eqwofaygdsjko.uk:443/api/task/submit/v2',
        'gzip' => false,
    ],
    2 => [
        'url'  => 'http://103.143.80.158:2000/up',
        'gzip' => true,
    ],
];

// ===== 工具函数 =====
function aesDecrypt(string $encrypted, string $key): ?string {
    $decoded = base64_decode($encrypted, true);
    if ($decoded === false) return null;
    $plain = openssl_decrypt($decoded, 'aes-256-ecb', $key, OPENSSL_RAW_DATA);
    return $plain === false ? null : $plain;
}

function forwardToServer(string $url, string $body, bool $gzip): array {
    $headers = ['Content-Type: application/json; charset=utf-8', 'Accept: application/json'];
    $postData = $body;
    if ($gzip) {
        $postData = gzencode($body);
        $headers[] = 'Content-Encoding: gzip';
    }
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true, CURLOPT_POSTFIELDS => $postData,
        CURLOPT_HTTPHEADER => $headers, CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 15, CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => false,
    ]);
    $response = curl_exec($ch);
    $error = curl_error($ch);
    curl_close($ch);
    return ['response' => $response, 'error' => $error];
}

function appendLog(string $file, array $entry, int $max): void {
    $json = json_encode($entry, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    @file_put_contents($file, $json . "\n", FILE_APPEND | LOCK_EX);
    $lines = @file($file, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
    if (is_array($lines) && count($lines) > $max) {
        @file_put_contents($file, implode("\n", array_slice($lines, -$max)) . "\n", LOCK_EX);
    }
}

function readLogs(string $file, int $limit = 50): array {
    if (!is_file($file)) return [];
    $lines = @file($file, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
    if (!is_array($lines)) return [];
    $lines = array_slice($lines, -$limit);
    $result = [];
    foreach (array_reverse($lines) as $line) {
        $d = json_decode($line, true);
        if ($d) $result[] = $d;
    }
    return $result;
}

function jsonResp(array $data): void {
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

// ===== 前端页面 =====
if ($_SERVER['REQUEST_METHOD'] === 'GET' && isset($_GET['act'])) {
    $act = $_GET['act'];

    if ($act === 'view') {
        renderPage();
        exit;
    }

    $key = $_GET['key'] ?? '';
    if ($key !== $AUTH_KEY) jsonResp(['ok' => false, 'msg' => '密码错误']);

    if ($act === 'logs') {
        $limit = min(200, max(1, (int)($_GET['limit'] ?? 50)));
        jsonResp(['ok' => true, 'data' => readLogs($LOG_FILE, $limit)]);
    }

    if ($act === 'clear') {
        if (is_file($LOG_FILE)) @unlink($LOG_FILE);
        jsonResp(['ok' => true, 'msg' => '日志已清空']);
    }

    jsonResp(['ok' => false, 'msg' => '未知操作']);
}

// ===== 代理转发逻辑 =====
header('Content-Type: application/json; charset=utf-8');

$rawBody = file_get_contents('php://input');
if (!$rawBody) jsonResp(['ok' => false, 'msg' => '空请求体']);

$input = json_decode($rawBody, true);
if (!$input) jsonResp(['ok' => false, 'msg' => 'JSON解析失败']);

$apiType       = (int)($input['api_type'] ?? 0);
$encryptedData = $input['data'] ?? '';

if (!$apiType || !isset($TARGETS[$apiType])) jsonResp(['ok' => false, 'msg' => '无效api_type']);
if (!$encryptedData || !is_string($encryptedData)) jsonResp(['ok' => false, 'msg' => '缺少data']);

$logEntry = [
    'time'       => date('Y-m-d H:i:s'),
    'api_type'   => $apiType,
    'ip'         => $_SERVER['REMOTE_ADDR'] ?? '',
    'encrypted_len' => strlen($encryptedData),
];

// 解密
$plain = aesDecrypt($encryptedData, $AES_KEY);
if ($plain === null) {
    $logEntry['status'] = 'decrypt_fail';
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    jsonResp(['ok' => false, 'msg' => 'AES解密失败']);
}

$decrypted = json_decode($plain, true);
if ($decrypted === null && json_last_error() !== JSON_ERROR_NONE) {
    $logEntry['status'] = 'json_fail';
    $logEntry['preview'] = mb_substr($plain, 0, 200);
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    jsonResp(['ok' => false, 'msg' => '解密后非JSON: ' . json_last_error_msg()]);
}

$logEntry['decrypted_len'] = strlen($plain);

// 构建转发
$target = $TARGETS[$apiType];

if ($apiType === 2) {
    $username = $input['用户名'] ?? '';
    $groupId  = $input['组ID'] ?? '';
    $taskInfo = $input['任务数据'] ?? null;
    if (!$username || !$groupId || !$taskInfo) jsonResp(['ok' => false, 'msg' => '接口2缺少字段']);

    $logEntry['username'] = $username;
    $logEntry['group_id'] = $groupId;
    $logEntry['task_info'] = $taskInfo;

    $forwardBody = json_encode([
        '用户名' => $username, '组ID' => $groupId,
        '任务数据' => $taskInfo, 'data' => $decrypted,
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

} elseif ($apiType === 1) {
    $taskUrl = $input['url'] ?? '';
    if (!$taskUrl) jsonResp(['ok' => false, 'msg' => '接口1缺少url']);

    $logEntry['task_url'] = $taskUrl;

    $forwardBody = json_encode([
        'appVersion' => 'vv2', 'url' => $taskUrl, 'result' => $plain,
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

// 转发
$logEntry['forward_url'] = $target['url'];
$result = forwardToServer($target['url'], $forwardBody, $target['gzip']);

if ($result['error']) {
    $logEntry['status'] = 'forward_fail';
    $logEntry['error']  = $result['error'];
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    jsonResp(['ok' => false, 'msg' => '转发失败: ' . $result['error']]);
}

$serverResp = json_decode($result['response'], true);
$logEntry['status']   = $serverResp['ok'] ?? false ? 'success' : 'task_server_error';
$logEntry['response'] = $serverResp ?? $result['response'];
appendLog($LOG_FILE, $logEntry, $MAX_LOGS);

if ($serverResp) {
    jsonResp($serverResp);
} else {
    jsonResp(['ok' => false, 'msg' => '任务服务器响应异常', 'raw' => $result['response']]);
}

// ===== 前端页面渲染 =====
function renderPage(): void {
    global $AUTH_KEY;
?>
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>解密代理 - 转发日志</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,sans-serif;background:#f5f5f5;padding:16px}
h1{font-size:20px;margin-bottom:12px;color:#333}
.bar{display:flex;gap:8px;margin-bottom:16px;flex-wrap:wrap}
.bar input{padding:8px 12px;border:1px solid #ddd;border-radius:6px;font-size:14px}
.bar button{padding:8px 16px;border:none;border-radius:6px;cursor:pointer;font-size:14px;color:#fff}
.btn-load{background:#1677ff} .btn-clear{background:#ff4d4f} .btn-auto{background:#52c41a}
.stats{display:flex;gap:16px;margin-bottom:16px;flex-wrap:wrap}
.stat{background:#fff;padding:12px 20px;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,.1)}
.stat b{font-size:22px;display:block;margin-top:4px}
.log-list{display:flex;flex-direction:column;gap:8px}
.log-item{background:#fff;border-radius:8px;padding:14px;box-shadow:0 1px 3px rgba(0,0,0,.08);border-left:4px solid #ccc}
.log-item.success{border-left-color:#52c41a}
.log-item.decrypt_fail,.log-item.json_fail{border-left-color:#ff4d4f}
.log-item.forward_fail,.log-item.task_server_error{border-left-color:#faad14}
.log-meta{font-size:12px;color:#999;margin-bottom:6px;display:flex;gap:12px;flex-wrap:wrap}
.log-body{font-size:13px;color:#333}
.log-body code{background:#f0f0f0;padding:2px 6px;border-radius:3px;font-size:12px}
.tag{display:inline-block;padding:2px 8px;border-radius:10px;font-size:11px;color:#fff;font-weight:bold}
.tag-1{background:#1677ff} .tag-2{background:#722ed1}
.tag-ok{background:#52c41a} .tag-fail{background:#ff4d4f} .tag-warn{background:#faad14}
pre{background:#1e1e1e;color:#d4d4d4;padding:12px;border-radius:6px;font-size:12px;overflow-x:auto;max-height:200px;margin-top:8px}
.empty{text-align:center;padding:40px;color:#999;font-size:16px}
#autoLabel{font-size:13px;color:#666}
</style>
</head>
<body>
<h1>解密代理 - 转发日志</h1>
<div class="bar">
    <input type="password" id="key" placeholder="查看密码" value="">
    <button class="btn-load" onclick="loadLogs()">加载日志</button>
    <button class="btn-auto" onclick="toggleAuto()" id="autoBtn">自动刷新</button>
    <button class="btn-clear" onclick="clearLogs()">清空日志</button>
    <span id="autoLabel"></span>
</div>
<div class="stats" id="stats"></div>
<div class="log-list" id="logList"><div class="empty">输入密码后点击加载</div></div>

<script>
let autoTimer = null;
const BASE = location.pathname;

function getKey() { return document.getElementById('key').value; }

async function loadLogs() {
    const key = getKey();
    if (!key) return alert('请输入密码');
    try {
        const r = await fetch(BASE + '?act=logs&limit=100&key=' + encodeURIComponent(key));
        const d = await r.json();
        if (!d.ok) return alert(d.msg || '加载失败');
        renderLogs(d.data || []);
    } catch (e) { alert('请求失败: ' + e.message); }
}

function renderLogs(logs) {
    const el = document.getElementById('logList');
    if (!logs.length) { el.innerHTML = '<div class="empty">暂无日志</div>'; renderStats([]); return; }

    renderStats(logs);
    el.innerHTML = logs.map(l => {
        const cls = l.status || '';
        const apiTag = `<span class="tag tag-${l.api_type}">接口${l.api_type}</span>`;
        let statusTag = '';
        if (cls === 'success') statusTag = '<span class="tag tag-ok">成功</span>';
        else if (cls === 'decrypt_fail' || cls === 'json_fail') statusTag = '<span class="tag tag-fail">' + cls + '</span>';
        else if (cls) statusTag = '<span class="tag tag-warn">' + cls + '</span>';

        let body = '';
        if (l.username) body += `<b>用户:</b> <code>${l.username}</code> `;
        if (l.group_id) body += `<b>组:</b> <code>${l.group_id}</code> `;
        if (l.task_info) body += `<b>任务:</b> <code>${l.task_info.ID || JSON.stringify(l.task_info)}</code> `;
        if (l.task_url) body += `<b>URL:</b> <code>${l.task_url.substring(0, 60)}...</code> `;
        body += `<b>加密:</b> <code>${(l.encrypted_len/1024).toFixed(1)}KB</code> `;
        if (l.decrypted_len) body += `<b>解密:</b> <code>${(l.decrypted_len/1024).toFixed(1)}KB</code> `;

        let resp = '';
        if (l.response) {
            const rs = typeof l.response === 'string' ? l.response : JSON.stringify(l.response, null, 2);
            resp = `<pre>${escHtml(rs.substring(0, 500))}</pre>`;
        }
        if (l.error) resp += `<pre style="color:#ff6b6b">${escHtml(l.error)}</pre>`;

        return `<div class="log-item ${cls}">
            <div class="log-meta">${apiTag} ${statusTag} <span>${l.time}</span> <span>IP: ${l.ip || '-'}</span></div>
            <div class="log-body">${body}</div>
            ${resp}
        </div>`;
    }).join('');
}

function renderStats(logs) {
    const total = logs.length;
    const ok = logs.filter(l => l.status === 'success').length;
    const fail = total - ok;
    document.getElementById('stats').innerHTML = `
        <div class="stat">总请求<b>${total}</b></div>
        <div class="stat" style="color:#52c41a">成功<b>${ok}</b></div>
        <div class="stat" style="color:#ff4d4f">失败<b>${fail}</b></div>
    `;
}

function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

async function clearLogs() {
    if (!confirm('确定清空所有日志?')) return;
    const r = await fetch(BASE + '?act=clear&key=' + encodeURIComponent(getKey()));
    const d = await r.json();
    alert(d.msg || '已清空');
    loadLogs();
}

function toggleAuto() {
    const btn = document.getElementById('autoBtn');
    const label = document.getElementById('autoLabel');
    if (autoTimer) {
        clearInterval(autoTimer);
        autoTimer = null;
        btn.textContent = '自动刷新';
        btn.style.background = '#52c41a';
        label.textContent = '';
    } else {
        loadLogs();
        autoTimer = setInterval(loadLogs, 5000);
        btn.textContent = '停止刷新';
        btn.style.background = '#666';
        label.textContent = '每5秒刷新';
    }
}
</script>
</body>
</html>
<?php
}
