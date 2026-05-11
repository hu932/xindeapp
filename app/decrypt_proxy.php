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
$DEVICE_FILE = __DIR__ . '/device_controls.json';
$UPDATE_FILE = __DIR__ . '/app_update.json';
$APK_DIR = __DIR__ . '/app_updates';
$STATS_DIR = __DIR__ . '/daily_stats';
$API1_TOKEN_FILE = __DIR__ . '/api1_token.json';
$API1_ACCOUNTS_FILE = __DIR__ . '/api1_allowed_accounts.json';
$API2_ACCOUNTS_FILE = __DIR__ . '/api2_allowed_accounts.json';
$MAX_LOGS = 500;

$API1_AUTH = [
    'login_url' => 'https://eqwofaygdsjko.uk:443/api/user/login',
    'username'  => 'youngdenise',
    'password'  => '@dzaM3mT9I',
];

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

@ini_set('upload_max_filesize', '200M');
@ini_set('post_max_size', '220M');
@ini_set('max_execution_time', '180');

// ===== 工具函数 =====
function aesDecrypt(string $encrypted, string $key): ?string {
    $decoded = base64_decode($encrypted, true);
    if ($decoded === false) return null;
    $plain = openssl_decrypt($decoded, 'aes-256-ecb', $key, OPENSSL_RAW_DATA);
    return $plain === false ? null : $plain;
}

function forwardToServer(string $url, string $body, bool $gzip, array $extraHeaders = []): array {
    $headers = ['Content-Type: application/json; charset=utf-8', 'Accept: application/json'];
    foreach ($extraHeaders as $header) {
        if (is_string($header) && $header !== '') $headers[] = $header;
    }
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
    $httpCode = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);
    return ['response' => $response, 'error' => $error, 'http_code' => $httpCode, 'headers' => $headers];
}

function jwtExp(string $token): int {
    $parts = explode('.', $token);
    if (count($parts) < 2) return 0;
    $payload = strtr($parts[1], '-_', '+/');
    $payload .= str_repeat('=', (4 - strlen($payload) % 4) % 4);
    $decoded = base64_decode($payload, true);
    if ($decoded === false) return 0;
    $data = json_decode($decoded, true);
    return is_array($data) ? (int)($data['exp'] ?? 0) : 0;
}

function getApi1Token(array $auth, string $tokenFile, bool $forceRefresh = false): array {
    if (!$forceRefresh && is_file($tokenFile)) {
        $cached = json_decode((string)@file_get_contents($tokenFile), true);
        $token = is_array($cached) ? (string)($cached['token'] ?? '') : '';
        $expiresAt = is_array($cached) ? (int)($cached['expires_at'] ?? 0) : 0;
        if ($token !== '' && $expiresAt > time() + 60) {
            return ['ok' => true, 'token' => $token, 'cached' => true, 'expires_at' => $expiresAt];
        }
    }

    $body = json_encode([
        'username' => (string)($auth['username'] ?? ''),
        'password' => (string)($auth['password'] ?? ''),
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

    $ch = curl_init((string)($auth['login_url'] ?? ''));
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $body,
        CURLOPT_HTTPHEADER => ['Content-Type: application/json', 'Accept: */*'],
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 15,
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => false,
    ]);
    $response = curl_exec($ch);
    $error = curl_error($ch);
    $httpCode = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    if ($error) return ['ok' => false, 'msg' => '接口1登录失败: ' . $error, 'http_code' => $httpCode];
    $json = json_decode((string)$response, true);
    $token = is_array($json) ? (string)($json['data']['token'] ?? '') : '';
    if ($token === '') {
        return ['ok' => false, 'msg' => '接口1登录未返回token', 'http_code' => $httpCode, 'response' => $json ?? $response];
    }

    $expiresAt = jwtExp($token);
    if ($expiresAt <= 0) $expiresAt = time() + 3600;
    @file_put_contents($tokenFile, json_encode([
        'token' => $token,
        'expires_at' => $expiresAt,
        'updated_at' => date('Y-m-d H:i:s'),
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES), LOCK_EX);

    return ['ok' => true, 'token' => $token, 'cached' => false, 'expires_at' => $expiresAt];
}

function readApi1AllowedAccounts(string $file): array {
    if (!is_file($file)) return [];
    $data = json_decode((string)@file_get_contents($file), true);
    if (!is_array($data)) return [];
    $accounts = $data['accounts'] ?? $data;
    if (!is_array($accounts)) return [];
    $result = [];
    foreach ($accounts as $account) {
        if (is_array($account)) {
            $name = trim((string)($account['username'] ?? $account['name'] ?? ''));
            $remark = trim((string)($account['remark'] ?? ''));
        } else {
            $name = trim((string)$account);
            $remark = '';
        }
        if ($name !== '') $result[$name] = ['username' => $name, 'remark' => $remark];
    }
    return array_values($result);
}

function parseAccountLines(string $raw): array {
    $clean = [];
    $lines = preg_split('/\r\n|\r|\n/', $raw) ?: [];
    foreach ($lines as $line) {
        $line = trim((string)$line);
        if ($line === '') continue;
        if (strpos($line, '|') !== false) {
            [$name, $remark] = array_pad(explode('|', $line, 2), 2, '');
        } else {
            $parts = preg_split('/\s+/', $line, 2) ?: [];
            $name = $parts[0] ?? '';
            $remark = $parts[1] ?? '';
        }
        $name = trim((string)$name);
        if ($name !== '') $clean[$name] = ['username' => $name, 'remark' => trim((string)$remark)];
    }
    ksort($clean);
    return array_values($clean);
}

function saveApiAccounts(string $file, array $accounts): array {
    @file_put_contents($file, json_encode([
        'accounts' => array_values($accounts),
        'updated_at' => date('Y-m-d H:i:s'),
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES), LOCK_EX);
    return ['ok' => true, 'data' => array_values($accounts), 'count' => count($accounts)];
}

function saveApi1AllowedAccounts(string $file, string $raw): array {
    return saveApiAccounts($file, parseAccountLines($raw));
}

function accountNames(array $accounts): array {
    return array_values(array_filter(array_map(fn($a) => is_array($a) ? (string)($a['username'] ?? '') : (string)$a, $accounts)));
}

function accountRemarkMap(array $accounts): array {
    $map = [];
    foreach ($accounts as $account) {
        if (!is_array($account)) continue;
        $username = (string)($account['username'] ?? '');
        if ($username !== '') $map[$username] = (string)($account['remark'] ?? '');
    }
    return $map;
}

function api1LoginWithPassword(string $url, string $username, string $password): array {
    $body = json_encode(['username' => $username, 'password' => $password], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $body,
        CURLOPT_HTTPHEADER => ['Content-Type: application/json', 'Accept: */*'],
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 15,
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => false,
    ]);
    $response = curl_exec($ch);
    $error = curl_error($ch);
    $httpCode = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);
    return ['response' => $response, 'error' => $error, 'http_code' => $httpCode];
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

function readDeviceControls(string $file): array {
    if (!is_file($file)) return [];
    $data = json_decode((string)@file_get_contents($file), true);
    return is_array($data) ? $data : [];
}

function saveDeviceControls(string $file, array $devices): void {
    @file_put_contents($file, json_encode($devices, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES), LOCK_EX);
}

function deviceKey(string $username, string $deviceId): string {
    return md5($username . '|' . $deviceId);
}

function updateDeviceHeartbeat(string $file, array $input): array {
    global $API1_ACCOUNTS_FILE, $API2_ACCOUNTS_FILE;
    $username = trim((string)($input['username'] ?? $input['用户名'] ?? ''));
    $groupId = trim((string)($input['group_id'] ?? $input['组ID'] ?? ''));
    $deviceId = trim((string)($input['device_id'] ?? $input['设备ID'] ?? ''));
    if ($username === '') $username = 'unknown';
    if ($deviceId === '') $deviceId = 'unknown-' . substr(md5($_SERVER['REMOTE_ADDR'] ?? ''), 0, 8);

    $devices = readDeviceControls($file);
    $key = deviceKey($username, $deviceId);
    $old = $devices[$key] ?? [];
    $disabled = !empty($old['disabled']);
    $apiType = (int)($input['api_type'] ?? ($old['api_type'] ?? 0));
    $remarkMap = accountRemarkMap(readApi1AllowedAccounts($apiType === 2 ? $API2_ACCOUNTS_FILE : $API1_ACCOUNTS_FILE));
    $devices[$key] = array_merge($old, [
        'key' => $key,
        'username' => $username,
        'account_remark' => $remarkMap[$username] ?? ($old['account_remark'] ?? ''),
        'group_id' => $groupId,
        'device_id' => $deviceId,
        'fingerprint_key' => trim((string)($input['fingerprint_key'] ?? '')),
        'fingerprint_id' => trim((string)($input['fingerprint_id'] ?? '')),
        'fingerprint_name' => trim((string)($input['fingerprint_name'] ?? '')),
        'api_type' => $apiType,
        'ip' => $_SERVER['REMOTE_ADDR'] ?? '',
        'last_seen' => date('Y-m-d H:i:s'),
        'first_seen' => $old['first_seen'] ?? date('Y-m-d H:i:s'),
        'disabled' => $disabled,
        'disabled_reason' => $old['disabled_reason'] ?? '',
    ]);
    saveDeviceControls($file, $devices);

    return [
        'ok' => true,
        'allowed' => !$disabled,
        'msg' => $disabled ? ($devices[$key]['disabled_reason'] ?: '设备已被禁用') : 'ok',
        'device_key' => $key,
    ];
}

function checkDeviceAllowed(string $file, array $input): array {
    $username = trim((string)($input['username'] ?? $input['用户名'] ?? ''));
    $deviceId = trim((string)($input['device_id'] ?? $input['设备ID'] ?? ''));
    if ($username === '' || $deviceId === '') return ['allowed' => true];

    $devices = readDeviceControls($file);
    $key = deviceKey($username, $deviceId);
    if (!empty($devices[$key]['disabled'])) {
        return [
            'allowed' => false,
            'msg' => $devices[$key]['disabled_reason'] ?: '设备已被禁用',
            'device_key' => $key,
        ];
    }
    updateDeviceHeartbeat($file, $input);
    return ['allowed' => true, 'device_key' => $key];
}

function updateDeviceControl(string $file, string $key, bool $disabled, string $reason = ''): array {
    $devices = readDeviceControls($file);
    if ($key === '' || empty($devices[$key])) {
        return ['ok' => false, 'msg' => '设备不存在'];
    }

    $devices[$key]['disabled'] = $disabled;
    $devices[$key]['disabled_reason'] = $disabled ? $reason : '';
    $devices[$key]['updated_at'] = date('Y-m-d H:i:s');
    saveDeviceControls($file, $devices);

    return [
        'ok' => true,
        'msg' => $disabled ? '设备已禁用' : '设备已启用',
        'data' => $devices[$key],
    ];
}

function readUpdateConfig(string $file): array {
    $defaults = [
        'version_code' => 1,
        'version_name' => '1.0.0',
        'apk_url' => '',
        'title' => '发现新版本',
        'message' => '',
        'force' => false,
        'updated_at' => '',
    ];
    if (!is_file($file)) return $defaults;
    $data = json_decode((string)@file_get_contents($file), true);
    return is_array($data) ? array_merge($defaults, $data) : $defaults;
}

function saveUpdateConfig(string $file, array $input): array {
    $config = readUpdateConfig($file);
    $config['version_code'] = max(1, (int)($input['version_code'] ?? $config['version_code']));
    $config['version_name'] = trim((string)($input['version_name'] ?? $config['version_name']));
    $config['apk_url'] = trim((string)($input['apk_url'] ?? $config['apk_url']));
    $config['title'] = trim((string)($input['title'] ?? $config['title']));
    $config['message'] = trim((string)($input['message'] ?? $config['message']));
    $config['force'] = !empty($input['force']) && (string)$input['force'] !== '0' && $input['force'] !== 'false';
    $config['updated_at'] = date('Y-m-d H:i:s');

    @file_put_contents($file, json_encode($config, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES), LOCK_EX);
    return ['ok' => true, 'msg' => '更新配置已保存', 'data' => $config];
}

function uploadUpdateApk(string $configFile, string $apkDir, array $input, array $files): array {
    if (empty($files['apk']) || !is_uploaded_file($files['apk']['tmp_name'])) {
        return saveUpdateConfig($configFile, $input);
    }
    if (($files['apk']['error'] ?? UPLOAD_ERR_OK) !== UPLOAD_ERR_OK) {
        return ['ok' => false, 'msg' => 'APK上传失败，错误码: ' . (int)$files['apk']['error']];
    }
    $original = (string)($files['apk']['name'] ?? '');
    if (strtolower(pathinfo($original, PATHINFO_EXTENSION)) !== 'apk') {
        return ['ok' => false, 'msg' => '只允许上传 .apk 文件'];
    }
    if (!is_dir($apkDir)) @mkdir($apkDir, 0755, true);

    $versionCode = max(1, (int)($input['version_code'] ?? 1));
    $safeName = 'taskhelper_v' . $versionCode . '_' . date('Ymd_His') . '.apk';
    $target = $apkDir . '/' . $safeName;
    if (!move_uploaded_file($files['apk']['tmp_name'], $target)) {
        return ['ok' => false, 'msg' => '保存APK失败，请检查目录权限'];
    }

    $scheme = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') ? 'https' : 'http';
    $host = $_SERVER['HTTP_HOST'] ?? '';
    $basePath = rtrim(dirname($_SERVER['SCRIPT_NAME'] ?? ''), '/\\');
    $apkUrl = $scheme . '://' . $host . ($basePath ? $basePath : '') . '/app_updates/' . rawurlencode($safeName);

    $input['apk_url'] = $apkUrl;
    $saved = saveUpdateConfig($configFile, $input);
    if (!empty($saved['data'])) {
        $saved['data']['apk_file'] = $safeName;
        $saved['data']['apk_size'] = filesize($target);
    }
    return $saved;
}

function buildUpdateResponse(string $file, int $currentCode): array {
    $config = readUpdateConfig($file);
    $hasUpdate = !empty($config['apk_url']) && (int)$config['version_code'] > $currentCode;
    return [
        'ok' => true,
        'has_update' => $hasUpdate,
        'version_code' => (int)$config['version_code'],
        'version_name' => (string)$config['version_name'],
        'apk_url' => (string)$config['apk_url'],
        'title' => (string)$config['title'],
        'message' => (string)$config['message'],
        'force' => !empty($config['force']),
        'updated_at' => (string)$config['updated_at'],
    ];
}

function jsonDecodeAssoc(string $json): ?array {
    $decoded = json_decode($json, true);
    return json_last_error() === JSON_ERROR_NONE && is_array($decoded) ? $decoded : null;
}

function saveDecryptDump(string $encryptedData, string $plain, int $apiType, $decrypted, $productData = null, $forwardPayload = null): string {
    $dumpDir = __DIR__ . '/decrypt_dumps';
    if (!is_dir($dumpDir)) @mkdir($dumpDir, 0755, true);

    $dumpId = date('Ymd_His') . '_' . substr(md5($encryptedData), 0, 8);
    $dumpData = [
        'dump_time' => date('Y-m-d H:i:s'),
        'api_type' => $apiType,
        'encrypted_len' => strlen($encryptedData),
        'decrypted_len' => strlen($plain),
        'decrypted_full' => $decrypted,
        'decrypted_keys' => is_array($decrypted) ? array_keys($decrypted) : 'not_array',
        'product_data' => $productData,
        'forward_payload' => $forwardPayload,
    ];

    if (is_array($decrypted) && isset($decrypted['responseContent']) && is_array($decrypted['responseContent'])) {
        $rc = $decrypted['responseContent'];
        $dumpData['responseContent_keys'] = array_keys($rc);
        $rb = $rc['response_body'] ?? null;
        $dumpData['response_body_type'] = gettype($rb);
        $dumpData['response_body_len'] = is_string($rb) ? strlen($rb) : null;
        if (is_string($rb) && $rb !== '') {
            $parsed = jsonDecodeAssoc($rb);
            $dumpData['response_body_parsed'] = $parsed;
            $dumpData['response_body_parse_ok'] = is_array($parsed);
        }
    }

    @file_put_contents("$dumpDir/$dumpId.json", json_encode($dumpData, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));

    $dumps = glob("$dumpDir/*.json");
    if (is_array($dumps) && count($dumps) > 20) {
        usort($dumps, fn($a, $b) => filemtime($a) - filemtime($b));
        for ($i = 0; $i < count($dumps) - 20; $i++) @unlink($dumps[$i]);
    }

    return $dumpId;
}

function saveUploadDump(string $dumpId, array $detail): string {
    $dumpDir = __DIR__ . '/upload_dumps';
    if (!is_dir($dumpDir)) @mkdir($dumpDir, 0755, true);

    $uploadId = $dumpId !== '' ? $dumpId : date('Ymd_His') . '_' . substr(md5(json_encode($detail)), 0, 8);
    @file_put_contents("$dumpDir/$uploadId.json", json_encode($detail, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));

    $dumps = glob("$dumpDir/*.json");
    if (is_array($dumps) && count($dumps) > 50) {
        usort($dumps, fn($a, $b) => filemtime($a) - filemtime($b));
        for ($i = 0; $i < count($dumps) - 50; $i++) @unlink($dumps[$i]);
    }

    return $uploadId;
}

// ===== 统计函数 =====
function getStatsFile(string $dir, string $date = ''): string {
    if ($date === '') $date = date('Y-m-d');
    if (!is_dir($dir)) @mkdir($dir, 0755, true);
    return $dir . '/' . $date . '.json';
}

function readDayStats(string $dir, string $date = ''): array {
    $file = getStatsFile($dir, $date);
    if (!is_file($file)) return [];
    $data = json_decode((string)@file_get_contents($file), true);
    return is_array($data) ? $data : [];
}

function fingerprintInfoFromInput(array $input): array {
    $username = trim((string)($input['username'] ?? $input['用户名'] ?? 'unknown'));
    $groupId = trim((string)($input['group_id'] ?? $input['组ID'] ?? ''));
    $deviceId = trim((string)($input['device_id'] ?? $input['设备ID'] ?? ''));
    $fpKey = trim((string)($input['fingerprint_key'] ?? ''));
    $fpId = trim((string)($input['fingerprint_id'] ?? ''));
    $fpName = trim((string)($input['fingerprint_name'] ?? ''));

    if ($fpKey === '') $fpKey = $fpId;
    if ($fpKey === '') $fpKey = $fpName;
    if ($fpKey === '') $fpKey = $deviceId;
    if ($fpKey === '') $fpKey = 'unknown-' . substr(md5($username . '|' . ($_SERVER['REMOTE_ADDR'] ?? '')), 0, 10);

    return [
        'key' => md5($username . '|' . $fpKey),
        'fingerprint_key' => $fpKey,
        'fingerprint_id' => $fpId,
        'fingerprint_name' => $fpName,
        'username' => $username,
        'group_id' => $groupId,
        'device_id' => $deviceId,
    ];
}

function recordStats(string $dir, array $input, int $apiType, string $status, string $taskId = ''): void {
    $date = date('Y-m-d');
    $file = getStatsFile($dir, $date);
    $stats = readDayStats($dir, $date);
    $fp = fingerprintInfoFromInput($input);
    $key = $fp['key'];

    if (!isset($stats[$key])) {
        $stats[$key] = [
            'fingerprint_key' => $fp['fingerprint_key'],
            'fingerprint_id' => $fp['fingerprint_id'],
            'fingerprint_name' => $fp['fingerprint_name'],
            'username' => $fp['username'],
            'group_id' => $fp['group_id'],
            'device_id' => $fp['device_id'],
            'total' => 0,
            'success' => 0,
            'fail' => 0,
            'records' => [],
        ];
    }
    $stats[$key]['fingerprint_key'] = $fp['fingerprint_key'];
    $stats[$key]['fingerprint_id'] = $fp['fingerprint_id'];
    $stats[$key]['fingerprint_name'] = $fp['fingerprint_name'];
    $stats[$key]['username'] = $fp['username'];
    $stats[$key]['group_id'] = $fp['group_id'];
    $stats[$key]['device_id'] = $fp['device_id'];
    $stats[$key]['last_seen'] = date('Y-m-d H:i:s');
    $stats[$key]['total']++;
    if ($status === 'success') {
        $stats[$key]['success']++;
    } else {
        $stats[$key]['fail']++;
    }
    $record = ['time' => date('H:i:s'), 'api' => $apiType, 'status' => $status];
    if ($taskId !== '') $record['task'] = $taskId;
    $stats[$key]['records'][] = $record;
    if (count($stats[$key]['records']) > 200) {
        $stats[$key]['records'] = array_slice($stats[$key]['records'], -200);
    }

    @file_put_contents($file, json_encode($stats, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES), LOCK_EX);
}

function getStatsDateList(string $dir, int $limit = 30): array {
    if (!is_dir($dir)) return [];
    $files = glob("$dir/*.json");
    if (!is_array($files)) return [];
    $dates = [];
    foreach ($files as $f) {
        $dates[] = basename($f, '.json');
    }
    rsort($dates);
    return array_slice($dates, 0, $limit);
}

// ===== 前端页面 =====
if ($_SERVER['REQUEST_METHOD'] === 'GET' && isset($_GET['act'])) {
    $act = $_GET['act'];

    if ($act === 'view') {
        renderPage();
        exit;
    }

    if ($act === 'check_update') {
        $currentCode = (int)($_GET['version_code'] ?? 0);
        jsonResp(buildUpdateResponse($UPDATE_FILE, $currentCode));
    }

    $key = $_GET['key'] ?? '';
    if ($key !== $AUTH_KEY) jsonResp(['ok' => false, 'msg' => '密码错误']);

    if ($act === 'logs') {
        $limit = min(200, max(1, (int)($_GET['limit'] ?? 50)));
        jsonResp(['ok' => true, 'data' => readLogs($LOG_FILE, $limit)]);
    }

    if ($act === 'devices') {
        $devices = array_values(readDeviceControls($DEVICE_FILE));
        $api1Remarks = accountRemarkMap(readApi1AllowedAccounts($API1_ACCOUNTS_FILE));
        $api2Remarks = accountRemarkMap(readApi1AllowedAccounts($API2_ACCOUNTS_FILE));
        foreach ($devices as &$device) {
            $u = (string)($device['username'] ?? '');
            $device['account_remark'] = ((int)($device['api_type'] ?? 0) === 2 ? ($api2Remarks[$u] ?? '') : ($api1Remarks[$u] ?? '')) ?: ($device['account_remark'] ?? '');
        }
        unset($device);
        $now = time();
        $onlineThreshold = 120;
        $devices = array_filter($devices, function($d) use ($now, $onlineThreshold) {
            $lastSeen = strtotime($d['last_seen'] ?? '');
            return $lastSeen && ($now - $lastSeen) <= $onlineThreshold;
        });
        $devices = array_values($devices);
        usort($devices, fn($a, $b) => strcmp((string)($b['last_seen'] ?? ''), (string)($a['last_seen'] ?? '')));
        jsonResp(['ok' => true, 'data' => $devices, 'online_count' => count($devices)]);
    }

    if ($act === 'device_toggle') {
        $deviceKey = (string)($_GET['device_key'] ?? '');
        $disabled = (int)($_GET['disabled'] ?? 0) === 1;
        $reason = trim((string)($_GET['reason'] ?? ''));
        jsonResp(updateDeviceControl($DEVICE_FILE, $deviceKey, $disabled, $reason));
    }

    if ($act === 'update_config') {
        jsonResp(['ok' => true, 'data' => readUpdateConfig($UPDATE_FILE)]);
    }

    if ($act === 'save_update') {
        jsonResp(saveUpdateConfig($UPDATE_FILE, $_GET));
    }

    if ($act === 'api1_accounts') {
        jsonResp(['ok' => true, 'data' => readApi1AllowedAccounts($API1_ACCOUNTS_FILE)]);
    }

    if ($act === 'save_api1_accounts') {
        jsonResp(saveApi1AllowedAccounts($API1_ACCOUNTS_FILE, (string)($_GET['accounts'] ?? '')));
    }

    if ($act === 'api2_accounts') {
        jsonResp(['ok' => true, 'data' => readApi1AllowedAccounts($API2_ACCOUNTS_FILE)]);
    }

    if ($act === 'save_api2_accounts') {
        jsonResp(saveApiAccounts($API2_ACCOUNTS_FILE, parseAccountLines((string)($_GET['accounts'] ?? ''))));
    }

    if ($act === 'stats') {
        $date = trim((string)($_GET['date'] ?? date('Y-m-d')));
        $stats = readDayStats($STATS_DIR, $date);
        $summary = [];
        foreach ($stats as $key => $info) {
            $summary[] = [
                'key' => (string)$key,
                'username' => $info['username'] ?? (string)$key,
                'group_id' => $info['group_id'] ?? '',
                'device_id' => $info['device_id'] ?? '',
                'fingerprint_key' => $info['fingerprint_key'] ?? '',
                'fingerprint_id' => $info['fingerprint_id'] ?? '',
                'fingerprint_name' => $info['fingerprint_name'] ?? '',
                'last_seen' => $info['last_seen'] ?? '',
                'total' => $info['total'] ?? 0,
                'success' => $info['success'] ?? 0,
                'fail' => $info['fail'] ?? 0,
                'records' => $info['records'] ?? [],
            ];
        }
        usort($summary, fn($a, $b) => $b['total'] - $a['total']);
        jsonResp(['ok' => true, 'date' => $date, 'dates' => getStatsDateList($STATS_DIR), 'data' => $summary]);
    }

    if ($act === 'clear') {
        if (is_file($LOG_FILE)) @unlink($LOG_FILE);
        $dumpDir = __DIR__ . '/decrypt_dumps';
        if (is_dir($dumpDir)) { array_map('unlink', glob("$dumpDir/*.json")); }
        $uploadDumpDir = __DIR__ . '/upload_dumps';
        if (is_dir($uploadDumpDir)) { array_map('unlink', glob("$uploadDumpDir/*.json")); }
        jsonResp(['ok' => true, 'msg' => '日志和数据已清空']);
    }

    // 查看解密后的完整数据
    if ($act === 'dump') {
        $id = $_GET['id'] ?? '';
        $dumpFile = __DIR__ . '/decrypt_dumps/' . basename($id) . '.json';
        if (!$id || !is_file($dumpFile)) jsonResp(['ok' => false, 'msg' => '数据不存在']);
        header('Content-Type: application/json; charset=utf-8');
        readfile($dumpFile);
        exit;
    }

    // 查看上传任务接口详情
    if ($act === 'upload') {
        $id = $_GET['id'] ?? '';
        $dumpFile = __DIR__ . '/upload_dumps/' . basename($id) . '.json';
        if (!$id || !is_file($dumpFile)) jsonResp(['ok' => false, 'msg' => '上传详情不存在']);
        header('Content-Type: application/json; charset=utf-8');
        $detail = json_decode((string)@file_get_contents($dumpFile), true);
        if (isset($_GET['full'])) {
            echo json_encode($detail, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
        } else {
            echo json_encode($detail['request_body'] ?? $detail, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
        }
        exit;
    }

    // 列出所有dump文件
    if ($act === 'dumps') {
        $dumpDir = __DIR__ . '/decrypt_dumps';
        $files = [];
        if (is_dir($dumpDir)) {
            foreach (glob("$dumpDir/*.json") as $f) {
                $files[] = ['id' => basename($f, '.json'), 'size' => filesize($f), 'time' => date('Y-m-d H:i:s', filemtime($f))];
            }
            usort($files, fn($a, $b) => strcmp($b['time'], $a['time']));
        }
        jsonResp(['ok' => true, 'data' => array_slice($files, 0, 20)]);
    }

    jsonResp(['ok' => false, 'msg' => '未知操作']);
}

if ($_SERVER['REQUEST_METHOD'] === 'POST' && ($_POST['act'] ?? '') === 'upload_update') {
    $key = $_POST['key'] ?? '';
    if ($key !== $AUTH_KEY) jsonResp(['ok' => false, 'msg' => '密码错误']);
    jsonResp(uploadUpdateApk($UPDATE_FILE, $APK_DIR, $_POST, $_FILES));
}

// ===== 代理转发逻辑 =====
header('Content-Type: application/json; charset=utf-8');

$rawBody = file_get_contents('php://input');
if (!$rawBody) jsonResp(['ok' => false, 'msg' => '空请求体']);

$input = json_decode($rawBody, true);
if (!$input) jsonResp(['ok' => false, 'msg' => 'JSON解析失败']);

$postAct = (string)($input['act'] ?? '');
if ($postAct === 'api1_login') {
    $username = trim((string)($input['username'] ?? ''));
    $password = (string)($input['password'] ?? '');
    if ($username === '' || $password === '') {
        jsonResp(['code' => '400', 'data' => null, 'msg' => '账号或密码为空']);
    }

    $allowed = readApi1AllowedAccounts($API1_ACCOUNTS_FILE);
    if (!in_array($username, accountNames($allowed), true)) {
        appendLog($LOG_FILE, [
            'time' => date('Y-m-d H:i:s'),
            'api_type' => 1,
            'ip' => $_SERVER['REMOTE_ADDR'] ?? '',
            'username' => $username,
            'status' => 'login_rejected',
            'error' => '账号未在后台接口1白名单',
        ], $MAX_LOGS);
        jsonResp(['code' => '403', 'data' => null, 'msg' => '账号未授权，请联系管理员']);
    }

    $loginResult = api1LoginWithPassword($API1_AUTH['login_url'], $username, $password);
    if (!empty($loginResult['error'])) {
        jsonResp(['code' => '500', 'data' => null, 'msg' => '登录转发失败: ' . $loginResult['error']]);
    }
    header('Content-Type: application/json; charset=utf-8');
    echo (string)$loginResult['response'];
    exit;
}

if ($postAct === 'heartbeat') {
    jsonResp(updateDeviceHeartbeat($DEVICE_FILE, $input));
}

$apiType       = (int)($input['api_type'] ?? 0);
$encryptedData = $input['data'] ?? '';

if (!$apiType || !isset($TARGETS[$apiType])) jsonResp(['ok' => false, 'msg' => '无效api_type']);
if (!$encryptedData || !is_string($encryptedData)) jsonResp(['ok' => false, 'msg' => '缺少data']);

$deviceCheck = checkDeviceAllowed($DEVICE_FILE, $input);
if (empty($deviceCheck['allowed'])) {
    $logEntry = [
        'time'       => date('Y-m-d H:i:s'),
        'api_type'   => $apiType,
        'ip'         => $_SERVER['REMOTE_ADDR'] ?? '',
        'username'   => $input['username'] ?? $input['用户名'] ?? '',
        'group_id'   => $input['group_id'] ?? $input['组ID'] ?? '',
        'device_id'  => $input['device_id'] ?? $input['设备ID'] ?? '',
        'fingerprint_key' => $input['fingerprint_key'] ?? '',
        'fingerprint_id' => $input['fingerprint_id'] ?? '',
        'fingerprint_name' => $input['fingerprint_name'] ?? '',
        'device_key' => $deviceCheck['device_key'] ?? '',
        'encrypted_len' => strlen($encryptedData),
        'status'     => 'device_disabled',
        'error'      => $deviceCheck['msg'] ?? '设备已被禁用',
    ];
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    recordStats($STATS_DIR, $input, $apiType, 'device_disabled');
    jsonResp(['ok' => false, 'msg' => $deviceCheck['msg'] ?? '设备已被禁用']);
}

$logEntry = [
    'time'       => date('Y-m-d H:i:s'),
    'api_type'   => $apiType,
    'ip'         => $_SERVER['REMOTE_ADDR'] ?? '',
    'username'   => $input['username'] ?? $input['用户名'] ?? '',
    'group_id'   => $input['group_id'] ?? $input['组ID'] ?? '',
    'device_id'  => $input['device_id'] ?? $input['设备ID'] ?? '',
    'fingerprint_key' => $input['fingerprint_key'] ?? '',
    'fingerprint_id' => $input['fingerprint_id'] ?? '',
    'fingerprint_name' => $input['fingerprint_name'] ?? '',
    'device_key' => $deviceCheck['device_key'] ?? '',
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
    $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted);
    $logEntry['preview'] = mb_substr($plain, 0, 200);
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    jsonResp(['ok' => false, 'msg' => '解密后非JSON: ' . json_last_error_msg()]);
}

$logEntry['decrypted_len'] = strlen($plain);
$logEntry['decrypted_keys'] = is_array($decrypted) ? array_keys($decrypted) : 'not_array';

// Mars加密的数据是一个wrapper JSON，结构: {requestHeaders, responseHeaders, responseContent, extra}
// 商品完整数据在 responseContent.response_body；它本身是一个 JSON 字符串，内容类似：
// {"bff_meta":null,"error":null,"error_msg":null,"data":{"item":{...完整商品字段...}}}
$responseBodyRaw = null;
$productResponse = null;
$actualData = null;
if (is_array($decrypted)) {
    if (isset($decrypted['responseContent']) && is_array($decrypted['responseContent'])) {
        $rc = $decrypted['responseContent'];
        $rb = $rc['response_body'] ?? '';
        if (is_string($rb) && $rb !== '') {
            $responseBodyRaw = $rb;
            $parsed = jsonDecodeAssoc($rb);
            if (is_array($parsed)) {
                $productResponse = $parsed;
                // 接口2的 data 传 response_body 解出来的完整对象，不传转义 JSON 字符串。
                $actualData = $parsed;
                $logEntry['extract_from'] = 'responseContent.response_body(object)';
                $logEntry['response_body_keys'] = array_keys($parsed);
                $item = $parsed['data']['item'] ?? $parsed['item'] ?? null;
                $logEntry['item_keys_count'] = is_array($item) ? count($item) : 0;
            } else {
                $logEntry['extract_from'] = 'raw_wrapper(response_body_not_json)';
            }
        } else {
            $logEntry['extract_from'] = 'raw_wrapper(response_body_empty)';
        }
    } else {
        $logEntry['extract_from'] = 'missing_responseContent.response_body';
    }
    $logEntry['response_body_preview'] = mb_substr(
        is_array($decrypted['responseContent'] ?? null) ? ($decrypted['responseContent']['response_body'] ?? '(missing)') : '(missing)',
        0, 200
    );
}

// 构建转发
$target = $TARGETS[$apiType];
$forwardHeaders = [];

if ($apiType === 2) {
    $username = $input['用户名'] ?? '';
    $groupId  = $input['组ID'] ?? '';
    $taskInfo = $input['任务数据'] ?? null;
    if (!$username || !$groupId || !$taskInfo) jsonResp(['ok' => false, 'msg' => '接口2缺少字段']);

    $logEntry['username'] = $username;
    $logEntry['group_id'] = $groupId;
    $logEntry['task_info'] = $taskInfo;
    if (!is_array($actualData)) {
        $logEntry['status'] = 'extract_fail';
        $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted, $productResponse);
        appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
        jsonResp(['ok' => false, 'msg' => '未提取到response_body对象，接口2不会上传wrapper或转义字符串']);
    }
    $payload = [
        '用户名' => $username, '组ID' => $groupId,
        '任务数据' => $taskInfo, 'data' => $actualData,
    ];
    $logEntry['forward_data_type'] = gettype($payload['data']);
    $logEntry['forward_data_keys'] = is_array($payload['data']) ? array_slice(array_keys($payload['data']), 0, 30) : null;
    $logEntry['forward_has_response_body_raw'] = is_string($responseBodyRaw) && $responseBodyRaw !== '';
    $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted, $productResponse, $payload);

    $forwardBody = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

} elseif ($apiType === 1) {
    $taskUrl = $input['url'] ?? '';
    if (!$taskUrl) jsonResp(['ok' => false, 'msg' => '接口1缺少url']);

    $logEntry['task_url'] = $taskUrl;
    if (!is_string($responseBodyRaw) || $responseBodyRaw === '') {
        $logEntry['status'] = 'extract_fail';
        $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted, $productResponse);
        appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
        jsonResp(['ok' => false, 'msg' => '未提取到responseContent.response_body，接口1不会上传完整wrapper']);
    }

    $payload = [
        'appVersion' => 'vv2', 'url' => $taskUrl, 'result' => $responseBodyRaw,
    ];
    $api1InputToken = trim((string)($input['auth_token'] ?? $input['token'] ?? ''));
    $api1UsingInputToken = $api1InputToken !== '';
    if ($api1UsingInputToken) {
        $forwardHeaders[] = 'Authorization: Bearer ' . $api1InputToken;
        $logEntry['api1_token_source'] = 'app';
        $api1InputTokenExp = jwtExp($api1InputToken);
        $logEntry['api1_token_expires_at'] = $api1InputTokenExp > 0 ? date('Y-m-d H:i:s', $api1InputTokenExp) : '';
    } else {
        $tokenResult = getApi1Token($API1_AUTH, $API1_TOKEN_FILE);
        if (empty($tokenResult['ok'])) {
            $logEntry['status'] = 'api1_login_fail';
            $logEntry['error'] = $tokenResult['msg'] ?? '接口1登录失败';
            appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
            jsonResp(['ok' => false, 'msg' => $logEntry['error']]);
        }
        $forwardHeaders[] = 'Authorization: Bearer ' . $tokenResult['token'];
        $logEntry['api1_token_source'] = 'proxy';
        $logEntry['api1_token_cached'] = !empty($tokenResult['cached']);
        $logEntry['api1_token_expires_at'] = date('Y-m-d H:i:s', (int)$tokenResult['expires_at']);
    }
    $logEntry['forward_result_from'] = 'responseContent.response_body';
    $logEntry['forward_result_len'] = strlen($responseBodyRaw);
    $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted, $productResponse, $payload);

    $forwardBody = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

// 转发
$logEntry['forward_url'] = $target['url'];
$result = forwardToServer($target['url'], $forwardBody, $target['gzip'], $forwardHeaders);
$serverResp = json_decode((string)$result['response'], true);

if ($apiType === 1 && empty($api1UsingInputToken) && is_array($serverResp) && stripos((string)($serverResp['msg'] ?? ''), 'login') !== false) {
    $tokenResult = getApi1Token($API1_AUTH, $API1_TOKEN_FILE, true);
    if (!empty($tokenResult['ok'])) {
        $forwardHeaders = ['Authorization: Bearer ' . $tokenResult['token']];
        $result = forwardToServer($target['url'], $forwardBody, $target['gzip'], $forwardHeaders);
        $serverResp = json_decode((string)$result['response'], true);
        $logEntry['api1_token_refreshed'] = true;
    }
}

$uploadDetail = [
    'time' => date('Y-m-d H:i:s'),
    'api_type' => $apiType,
    'target_url' => $target['url'],
    'method' => 'POST',
    'content_type' => 'application/json; charset=utf-8',
    'content_encoding' => $target['gzip'] ? 'gzip' : 'identity',
    'request_body' => json_decode($forwardBody, true),
    'http_code' => $result['http_code'] ?? 0,
    'response_json' => is_array($serverResp) ? $serverResp : null,
    'response_raw' => $result['response'],
    'error' => $result['error'],
];
$logEntry['upload_id'] = saveUploadDump((string)($logEntry['dump_id'] ?? ''), $uploadDetail);

if ($result['error']) {
    $logEntry['status'] = 'forward_fail';
    $logEntry['error']  = $result['error'];
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    recordStats($STATS_DIR, $input, $apiType, 'forward_fail');
    jsonResp(['ok' => false, 'msg' => '转发失败: ' . $result['error']]);
}

$logEntry['status']   = $serverResp['ok'] ?? false ? 'success' : 'task_server_error';
$logEntry['response'] = $serverResp ?? $result['response'];
appendLog($LOG_FILE, $logEntry, $MAX_LOGS);

$statsTaskId = '';
if (is_array($taskInfo ?? null)) $statsTaskId = $taskInfo['ID'] ?? '';
recordStats($STATS_DIR, $input, $apiType, $logEntry['status'], $statsTaskId);

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
<title>任务代理控制台</title>
<style>
:root{--primary:#1890ff;--success:#52c41a;--danger:#f5222d;--warning:#faad14;--info:#722ed1;--sidebar:#001529;--sidebar-w:220px;--header-h:50px;--bg:#f0f2f5;--card:#fff;--border:#e8e8e8;--text:#333;--muted:#999;--font:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB",sans-serif}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:var(--font);background:var(--bg);color:var(--text);font-size:14px}

/* Login */
.login-wrap{display:flex;align-items:center;justify-content:center;min-height:100vh;background:linear-gradient(135deg,#1a1a2e 0%,#16213e 50%,#0f3460 100%)}
.login-box{background:#fff;border-radius:12px;padding:40px;width:380px;box-shadow:0 8px 40px rgba(0,0,0,.3)}
.login-box h1{font-size:22px;text-align:center;margin-bottom:8px;color:#001529}
.login-box p{text-align:center;color:var(--muted);margin-bottom:28px;font-size:13px}
.login-box input{width:100%;height:42px;border:1px solid #d9d9d9;border-radius:6px;padding:0 14px;font-size:14px;margin-bottom:16px;transition:border .2s}
.login-box input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 2px rgba(24,144,255,.1)}
.login-box button{width:100%;height:42px;background:var(--primary);color:#fff;border:none;border-radius:6px;font-size:15px;font-weight:600;cursor:pointer;transition:opacity .2s}
.login-box button:hover{opacity:.85}

/* Layout */
.layout{display:flex;min-height:100vh}
.sidebar{width:var(--sidebar-w);background:var(--sidebar);position:fixed;top:0;left:0;bottom:0;z-index:100;overflow-y:auto;transition:width .2s}
.sidebar-logo{height:50px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:16px;font-weight:700;border-bottom:1px solid rgba(255,255,255,.06)}
.sidebar-menu{padding:8px 0}
.menu-item{display:flex;align-items:center;height:44px;padding:0 20px;color:rgba(255,255,255,.7);cursor:pointer;font-size:14px;transition:all .15s;gap:10px;border-left:3px solid transparent}
.menu-item:hover{color:#fff;background:rgba(255,255,255,.06)}
.menu-item.active{color:#fff;background:rgba(24,144,255,.25);border-left-color:var(--primary)}
.menu-icon{width:18px;text-align:center;font-style:normal}
.main{margin-left:var(--sidebar-w);flex:1;min-width:0}
.header{height:var(--header-h);background:#fff;border-bottom:1px solid var(--border);display:flex;align-items:center;justify-content:space-between;padding:0 24px;position:sticky;top:0;z-index:50}
.header-left{font-size:16px;font-weight:600}
.header-right{display:flex;align-items:center;gap:16px;font-size:13px;color:var(--muted)}
.header-right a{color:var(--danger);text-decoration:none;cursor:pointer;font-weight:600}
.content{padding:20px 24px}

/* Cards */
.stat-row{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:20px}
.stat-card{background:var(--card);border-radius:8px;padding:20px;border:1px solid var(--border)}
.stat-card .label{font-size:13px;color:var(--muted);margin-bottom:8px}
.stat-card .value{font-size:28px;font-weight:700}
.stat-card .value.green{color:var(--success)}.stat-card .value.red{color:var(--danger)}.stat-card .value.blue{color:var(--primary)}.stat-card .value.purple{color:var(--info)}
.card{background:var(--card);border-radius:8px;border:1px solid var(--border);margin-bottom:20px}
.card-head{padding:16px 20px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center}
.card-head h3{font-size:15px;font-weight:600}
.card-body{padding:20px}
.card-toolbar{display:flex;gap:8px;align-items:center;flex-wrap:wrap}

/* Table */
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:10px 12px;border-bottom:1px solid #f0f0f0;font-size:13px}
th{background:#fafafa;font-weight:600;color:#666;white-space:nowrap}
tr:hover td{background:#fafafa}

/* Buttons */
.btn{height:32px;border:none;border-radius:4px;cursor:pointer;font-size:13px;font-weight:500;padding:0 14px;display:inline-flex;align-items:center;gap:4px;transition:opacity .2s;color:#fff}
.btn:hover{opacity:.85}
.btn-sm{height:26px;font-size:12px;padding:0 10px}
.btn-primary{background:var(--primary)}.btn-success{background:var(--success)}.btn-danger{background:var(--danger)}.btn-warning{background:var(--warning);color:#333}.btn-info{background:var(--info)}.btn-default{background:#fff;color:var(--text);border:1px solid #d9d9d9}

/* Tags */
.tag{display:inline-block;padding:2px 8px;border-radius:3px;font-size:12px;font-weight:500;line-height:20px}
.tag-success{background:#f6ffed;color:#52c41a;border:1px solid #b7eb8f}.tag-danger{background:#fff2f0;color:#f5222d;border:1px solid #ffccc7}.tag-warning{background:#fffbe6;color:#faad14;border:1px solid #ffe58f}.tag-info{background:#f9f0ff;color:#722ed1;border:1px solid #d3adf7}.tag-blue{background:#e6f7ff;color:#1890ff;border:1px solid #91d5ff}

/* Form */
.form-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}
.form-item{display:flex;flex-direction:column;gap:6px}
.form-item label{font-size:13px;color:#666;font-weight:500}
.form-item input,.form-item textarea,.form-item select{height:36px;border:1px solid #d9d9d9;border-radius:4px;padding:0 10px;font-size:14px}
.form-item textarea{height:auto;min-height:80px;padding:8px 10px;resize:vertical}
.form-check{display:flex;align-items:center;gap:8px;padding-top:20px}

/* Log items */
.log-entry{padding:12px 16px;border-bottom:1px solid #f5f5f5;font-size:13px}
.log-entry:last-child{border-bottom:none}
.log-entry:hover{background:#fafafa}
.log-meta{display:flex;gap:10px;align-items:center;margin-bottom:4px;flex-wrap:wrap}
.log-detail{color:#666;line-height:1.8}
.log-detail code{background:#f5f5f5;padding:1px 6px;border-radius:3px;font-size:12px}
.log-actions{margin-top:6px;display:flex;gap:6px}
.log-actions a{font-size:12px;color:var(--primary);text-decoration:none}
.log-actions a:hover{text-decoration:underline}
pre.resp{background:#f5f5f5;padding:10px;border-radius:4px;font-size:12px;overflow-x:auto;max-height:160px;margin-top:6px;color:#333}

/* Device */
.device-card{padding:14px 16px;border:1px solid var(--border);border-radius:6px;margin-bottom:10px;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:10px}
.device-info{flex:1;min-width:200px}
.device-info b{font-size:15px}
.device-meta{font-size:12px;color:var(--muted);margin-top:4px;display:flex;gap:12px;flex-wrap:wrap}

/* Stats user row */
.stats-user{padding:14px 16px;border:1px solid var(--border);border-radius:6px;margin-bottom:10px}
.stats-user-head{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px}
.stats-nums{display:flex;gap:20px;font-size:14px}
.stats-records{margin-top:8px;font-size:12px;color:var(--muted)}

.empty{text-align:center;padding:40px;color:var(--muted)}
.hidden{display:none !important}
@media(max-width:768px){.sidebar{width:0;overflow:hidden}.main{margin-left:0}.stat-row{grid-template-columns:repeat(2,1fr)}.form-grid{grid-template-columns:1fr}}
</style>
</head>
<body>

<!-- Login Page -->
<div class="login-wrap" id="loginPage">
<div class="login-box">
    <h1>任务代理控制台</h1>
    <p>日志监控 / 设备管理 / 数据统计</p>
    <input type="password" id="loginPwd" placeholder="请输入管理密码" onkeydown="if(event.key==='Enter')doLogin()">
    <button onclick="doLogin()">登 录</button>
</div>
</div>

<!-- Admin Layout -->
<div class="layout hidden" id="adminLayout">
<div class="sidebar">
    <div class="sidebar-logo">任务代理控制台</div>
    <div class="sidebar-menu">
        <div class="menu-item active" onclick="switchPage('dashboard',this)"><i class="menu-icon">&#9632;</i>仪表盘</div>
        <div class="menu-item" onclick="switchPage('logs',this)"><i class="menu-icon">&#9776;</i>转发日志</div>
        <div class="menu-item" onclick="switchPage('devices',this)"><i class="menu-icon">&#9679;</i>在线设备</div>
        <div class="menu-item" onclick="switchPage('stats',this)"><i class="menu-icon">&#9733;</i>环境统计</div>
        <div class="menu-item" onclick="switchPage('accounts',this)"><i class="menu-icon">&#9737;</i>接口1账号</div>
        <div class="menu-item" onclick="switchPage('update',this)"><i class="menu-icon">&#8679;</i>更新发布</div>
        <div class="menu-item" onclick="switchPage('data',this)"><i class="menu-icon">&#128196;</i>解密数据</div>
    </div>
</div>
<div class="main">
    <div class="header">
        <div class="header-left" id="pageTitle">仪表盘</div>
        <div class="header-right">
            <span id="headerTime"></span>
            <a onclick="doLogout()">退出登录</a>
        </div>
    </div>
    <div class="content">

    <!-- Dashboard -->
    <div class="page" id="page-dashboard">
        <div class="stat-row" id="dashStats"></div>
        <div class="card"><div class="card-head"><h3>最近转发</h3><div class="card-toolbar"><button class="btn btn-primary btn-sm" onclick="switchPage('logs',document.querySelectorAll('.menu-item')[1])">查看全部</button></div></div><div class="card-body"><div id="dashLogs" style="max-height:400px;overflow-y:auto"><div class="empty">加载中...</div></div></div></div>
    </div>

    <!-- Logs -->
    <div class="page hidden" id="page-logs">
        <div class="card"><div class="card-head"><h3>转发日志</h3><div class="card-toolbar"><button class="btn btn-primary btn-sm" onclick="loadLogs()">刷新</button><button class="btn btn-success btn-sm" onclick="toggleAutoLog()" id="autoLogBtn">自动刷新</button><button class="btn btn-danger btn-sm" onclick="clearLogs()">清空</button></div></div><div class="card-body" style="padding:0"><div id="logList"><div class="empty">点击刷新加载日志</div></div></div></div>
    </div>

    <!-- Devices -->
    <div class="page hidden" id="page-devices">
        <div class="card"><div class="card-head"><h3>在线设备 <span id="onlineCount" style="color:var(--success);font-size:14px"></span></h3><div class="card-toolbar"><span style="font-size:12px;color:var(--muted)">每10秒自动刷新，仅显示最近2分钟有心跳的设备</span></div></div><div class="card-body"><div id="deviceList"><div class="empty">加载中...</div></div></div></div>
    </div>

    <!-- Stats -->
    <div class="page hidden" id="page-stats">
        <div class="card"><div class="card-head"><h3>环境统计</h3><div class="card-toolbar"><select id="statsDate" onchange="loadStats()" style="height:32px;border:1px solid #d9d9d9;border-radius:4px;padding:0 10px;font-size:13px"></select></div></div><div class="card-body"><div id="statsBody"><div class="empty">加载中...</div></div></div></div>
    </div>

    <!-- Accounts -->
    <div class="page hidden" id="page-accounts">
        <div class="card"><div class="card-head"><h3>接口账号备注</h3></div><div class="card-body"><div class="form-grid"><div class="form-item"><label>接口1允许账号。每行：账号 备注，未设置账号会拒绝登录</label><textarea id="api1Accounts" style="min-height:260px;font-family:monospace"></textarea></div><div class="form-item"><label>接口2账号备注。每行：账号 备注，用于在线设备和统计显示</label><textarea id="api2Accounts" style="min-height:260px;font-family:monospace"></textarea></div></div><div style="margin-top:14px;display:flex;gap:10px;align-items:center"><button class="btn btn-primary" onclick="saveApiAccounts()">保存账号备注</button><span id="apiAccountsResult" style="font-size:13px;color:var(--muted)"></span></div></div></div>
    </div>

    <!-- Update -->
    <div class="page hidden" id="page-update">
        <div class="card"><div class="card-head"><h3>在线更新发布</h3></div><div class="card-body"><div id="updateBody"><div class="empty">加载中...</div></div></div></div>
    </div>

    <!-- Data -->
    <div class="page hidden" id="page-data">
        <div class="card"><div class="card-head"><h3>解密数据列表</h3><div class="card-toolbar"><button class="btn btn-info btn-sm" onclick="openLatestDump()">打开最新</button></div></div><div class="card-body"><div id="dumpList"><div class="empty">加载中...</div></div></div></div>
    </div>

    </div>
</div>
</div>

<script>
const BASE = location.pathname;
const AUTH_KEY = '';
let KEY = localStorage.getItem('admin_key') || '';
let autoLogTimer = null, deviceTimer = null;

function doLogin() {
    const pwd = document.getElementById('loginPwd').value.trim();
    if (!pwd) return alert('请输入密码');
    KEY = pwd;
    localStorage.setItem('admin_key', pwd);
    fetch(BASE+'?act=logs&limit=1&key='+encodeURIComponent(pwd)).then(r=>r.json()).then(d=>{
        if (d.ok === false && d.msg === '密码错误') { alert('密码错误'); KEY=''; localStorage.removeItem('admin_key'); return; }
        showAdmin();
    }).catch(()=>alert('网络错误'));
}
function doLogout() { KEY=''; localStorage.removeItem('admin_key'); location.reload(); }
function showAdmin() {
    document.getElementById('loginPage').classList.add('hidden');
    document.getElementById('adminLayout').classList.remove('hidden');
    loadDashboard();
    setInterval(()=>{ document.getElementById('headerTime').textContent = new Date().toLocaleTimeString(); }, 1000);
}

if (KEY) { fetch(BASE+'?act=logs&limit=1&key='+encodeURIComponent(KEY)).then(r=>r.json()).then(d=>{ if(d.ok!==false||d.msg!=='密码错误') showAdmin(); else { KEY=''; localStorage.removeItem('admin_key'); }}).catch(()=>{}); }

const pageTitles = {dashboard:'仪表盘',logs:'转发日志',devices:'在线设备',stats:'环境统计',accounts:'接口账号备注',update:'更新发布',data:'解密数据'};
let currentPage = 'dashboard';
function switchPage(name, el) {
    if (deviceTimer && name !== 'devices') { clearInterval(deviceTimer); deviceTimer = null; }
    currentPage = name;
    document.querySelectorAll('.page').forEach(p=>p.classList.add('hidden'));
    document.getElementById('page-'+name).classList.remove('hidden');
    document.querySelectorAll('.menu-item').forEach(m=>m.classList.remove('active'));
    if (el) el.classList.add('active');
    document.getElementById('pageTitle').textContent = pageTitles[name]||name;
    if (name==='dashboard') loadDashboard();
    else if (name==='logs') loadLogs();
    else if (name==='devices') { loadDevices(); if(!deviceTimer) deviceTimer=setInterval(loadDevices,10000); }
    else if (name==='stats') loadStats();
    else if (name==='accounts') loadApi1Accounts();
    else if (name==='update') loadUpdateConfig();
    else if (name==='data') loadDumpList();
}

function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');}
function escAttr(s){return esc(s).replace(/"/g,'&quot;');}

// ===== Dashboard =====
async function loadDashboard() {
    try {
        const [logsR, devR, statsR] = await Promise.all([
            fetch(BASE+'?act=logs&limit=50&key='+encodeURIComponent(KEY)).then(r=>r.json()),
            fetch(BASE+'?act=devices&key='+encodeURIComponent(KEY)).then(r=>r.json()),
            fetch(BASE+'?act=stats&key='+encodeURIComponent(KEY)).then(r=>r.json())
        ]);
        const logs = logsR.data||[];
        const online = devR.online_count||0;
        const users = statsR.data||[];
        const todayTotal = users.reduce((s,u)=>s+u.total,0);
        const todayOk = users.reduce((s,u)=>s+u.success,0);
        const ok = logs.filter(l=>l.status==='success').length;
        document.getElementById('dashStats').innerHTML = `
            <div class="stat-card"><div class="label">在线设备</div><div class="value blue">${online}</div></div>
            <div class="stat-card"><div class="label">今日请求</div><div class="value">${todayTotal}</div></div>
            <div class="stat-card"><div class="label">今日成功</div><div class="value green">${todayOk}</div></div>
            <div class="stat-card"><div class="label">Active FP</div><div class="value purple">${users.length}</div></div>`;
        renderLogEntries(logs.slice(0,10), 'dashLogs');
    } catch(e) { console.error(e); }
}

// ===== Logs =====
async function loadLogs() {
    const r = await fetch(BASE+'?act=logs&limit=100&key='+encodeURIComponent(KEY));
    const d = await r.json();
    if (!d.ok) return;
    renderLogEntries(d.data||[], 'logList');
}
function renderLogEntries(logs, target) {
    const el = document.getElementById(target);
    if (!logs.length) { el.innerHTML='<div class="empty">暂无日志</div>'; return; }
    el.innerHTML = logs.map(l => {
        const st = l.status||'';
        const apiTag = `<span class="tag tag-${l.api_type===1?'blue':'info'}">接口${l.api_type}</span>`;
        let stTag = '';
        if(st==='success') stTag='<span class="tag tag-success">成功</span>';
        else if(st==='decrypt_fail'||st==='json_fail') stTag=`<span class="tag tag-danger">${st}</span>`;
        else if(st) stTag=`<span class="tag tag-warning">${st}</span>`;
        let body='';
        if(l.username) body+=`<b>用户:</b><code>${esc(l.username)}</code> `;
        if(l.group_id) body+=`<b>组:</b><code>${esc(l.group_id)}</code> `;
        if(l.task_info) body+=`<b>任务:</b><code>${esc(l.task_info.ID||JSON.stringify(l.task_info))}</code> `;
        if(l.encrypted_len) body+=`<b>加密:</b><code>${(l.encrypted_len/1024).toFixed(1)}KB</code> `;
        if(l.decrypted_len) body+=`<b>解密:</b><code>${(l.decrypted_len/1024).toFixed(1)}KB</code> `;
        if(l.extract_from) body+=`<b>提取:</b><code>${esc(l.extract_from)}</code> `;
        let resp='';
        if(l.response){ const rs=typeof l.response==='string'?l.response:JSON.stringify(l.response,null,2); resp=`<pre class="resp">${esc(rs.substring(0,500))}</pre>`; }
        if(l.error) resp+=`<pre class="resp" style="color:var(--danger)">${esc(l.error)}</pre>`;
        const dH=l.dump_id?`<a href="${BASE}?act=dump&id=${encodeURIComponent(l.dump_id)}&key=${encodeURIComponent(KEY)}" target="_blank">解密数据</a>`:'';
        const uH=l.upload_id?`<a href="${BASE}?act=upload&id=${encodeURIComponent(l.upload_id)}&key=${encodeURIComponent(KEY)}" target="_blank">上传详情</a>`:'';
        return `<div class="log-entry"><div class="log-meta">${apiTag} ${stTag} <span style="color:var(--muted);font-size:12px">${l.time||''} &nbsp; IP:${l.ip||'-'}</span></div><div class="log-detail">${body}</div>${(dH||uH)?`<div class="log-actions">${dH} ${uH}</div>`:''} ${resp}</div>`;
    }).join('');
}
function toggleAutoLog() {
    const btn=document.getElementById('autoLogBtn');
    if(autoLogTimer){clearInterval(autoLogTimer);autoLogTimer=null;btn.textContent='自动刷新';btn.className='btn btn-success btn-sm';}
    else{loadLogs();autoLogTimer=setInterval(loadLogs,5000);btn.textContent='停止刷新';btn.className='btn btn-default btn-sm';}
}
async function clearLogs(){
    if(!confirm('确定清空所有日志和数据?'))return;
    await fetch(BASE+'?act=clear&key='+encodeURIComponent(KEY));
    alert('已清空');loadLogs();
}

// ===== Devices =====
async function loadDevices() {
    const r = await fetch(BASE+'?act=devices&key='+encodeURIComponent(KEY));
    const d = await r.json();
    if(!d.ok) return;
    document.getElementById('onlineCount').textContent='('+d.online_count+'台在线)';
    const el=document.getElementById('deviceList');
    const devs=d.data||[];
    if(!devs.length){el.innerHTML='<div class="empty">当前没有在线设备</div>';return;}
    el.innerHTML=devs.map(dv=>{
        const dis=!!dv.disabled;
        const stTag=dis?'<span class="tag tag-danger">已禁用</span>':'<span class="tag tag-success">在线</span>';
        const btn=dis
            ?`<button class="btn btn-success btn-sm" onclick="toggleDevice('${esc(dv.key)}',0)">启用</button>`
            :`<button class="btn btn-danger btn-sm" onclick="toggleDevice('${esc(dv.key)}',1)">禁用</button>`;
        return `<div class="device-card"><div class="device-info"><b>${esc(dv.username||'-')}</b> ${stTag}${dv.account_remark?`<div style="margin-top:4px;font-size:12px;color:var(--danger);font-weight:600">${esc(dv.account_remark)}</div>`:''}<div class="device-meta"><span>组:${esc(dv.group_id||'-')}</span><span>接口:${dv.api_type||'-'}</span><span>设备:${esc((dv.device_id||'').substring(0,12))}</span><span>指纹:${esc(dv.fingerprint_name||((dv.fingerprint_id||dv.fingerprint_key||'').substring(0,12))||'-')}</span><span>IP:${esc(dv.ip||'-')}</span><span>心跳:${esc(dv.last_seen||'-')}</span></div>${dv.disabled_reason?`<div style="margin-top:4px;font-size:12px;color:var(--danger)">${esc(dv.disabled_reason)}</div>`:''}</div><div>${btn}</div></div>`;
    }).join('');
}
async function toggleDevice(dk,dis){
    let reason='';
    if(dis){reason=prompt('禁用原因（App会显示）','账号已被下线，禁止使用');if(!reason)return;}
    await fetch(BASE+'?act=device_toggle&key='+encodeURIComponent(KEY)+'&device_key='+encodeURIComponent(dk)+'&disabled='+dis+'&reason='+encodeURIComponent(reason));
    loadDevices();
}

// ===== Stats =====
async function loadStats(){
    const sel=document.getElementById('statsDate');
    const date=sel.value||'';
    const r=await fetch(BASE+'?act=stats&key='+encodeURIComponent(KEY)+(date?'&date='+encodeURIComponent(date):''));
    const d=await r.json();
    if(!d.ok)return;
    if(d.dates&&d.dates.length){const cv=sel.value||d.date;sel.innerHTML=d.dates.map(dt=>`<option value="${dt}" ${dt===cv?'selected':''}>${dt}</option>`).join('');}
    const el=document.getElementById('statsBody');
    const fps=d.data||[];
    if(!fps.length){el.innerHTML='<div class="empty">No fingerprint call records today</div>';return;}
    const tAll=fps.reduce((s,u)=>s+u.total,0);
    let html=`<div class="stat-row" style="margin-bottom:16px"><div class="stat-card"><div class="label">Date</div><div class="value" style="font-size:18px">${esc(d.date)}</div></div><div class="stat-card"><div class="label">Active FP</div><div class="value blue">${fps.length}</div></div><div class="stat-card"><div class="label">Calls</div><div class="value">${tAll}</div></div><div class="stat-card"><div class="label">Success</div><div class="value green">${fps.reduce((s,u)=>s+u.success,0)}</div></div></div>`;
    html+=fps.map(u=>{
        const rate=u.total>0?((u.success/u.total)*100).toFixed(1):'0';
        const last5=(u.records||[]).slice(-8).reverse().map(r=>`<span class="tag ${r.status==='success'?'tag-success':'tag-warning'}" style="margin-right:4px">${r.time} API${r.api} ${r.status}${r.task?' '+esc(r.task):''}</span>`).join('');
        const fpTitle=u.fingerprint_name||u.fingerprint_id||u.fingerprint_key||u.device_id||'unknown';
        const fpMeta=`user:${esc(u.username||'-')} group:${esc(u.group_id||'-')} device:${esc((u.device_id||'-').substring(0,16))} fp:${esc((u.fingerprint_id||u.fingerprint_key||'-').substring(0,16))} last:${esc(u.last_seen||'-')}`;
        return `<div class="stats-user"><div class="stats-user-head"><div><b style="font-size:15px">${esc(fpTitle)}</b><div class="stats-records">${fpMeta}</div></div><div class="stats-nums"><span>Calls <b>${u.total}</b></span><span style="color:var(--success)">OK <b>${u.success}</b></span><span style="color:var(--danger)">Fail <b>${u.fail}</b></span><span style="color:var(--primary)">Rate <b>${rate}%</b></span></div></div>${last5?`<div class="stats-records">${last5}</div>`:''}</div>`;
    }).join('');
    el.innerHTML=html;
}

// ===== API1 Accounts =====
async function loadApi1Accounts(){
    const [r1,r2]=await Promise.all([
        fetch(BASE+'?act=api1_accounts&key='+encodeURIComponent(KEY)).then(r=>r.json()),
        fetch(BASE+'?act=api2_accounts&key='+encodeURIComponent(KEY)).then(r=>r.json())
    ]);
    if(r1.ok) document.getElementById('api1Accounts').value=(r1.data||[]).map(a=>`${a.username||a}${a.remark?' '+a.remark:''}`).join('\n');
    if(r2.ok) document.getElementById('api2Accounts').value=(r2.data||[]).map(a=>`${a.username||a}${a.remark?' '+a.remark:''}`).join('\n');
    document.getElementById('apiAccountsResult').textContent='已加载 接口1 '+((r1.data||[]).length)+' 个，接口2 '+((r2.data||[]).length)+' 个';
}
async function saveApiAccounts(){
    const api1=document.getElementById('api1Accounts').value;
    const api2=document.getElementById('api2Accounts').value;
    document.getElementById('apiAccountsResult').textContent='保存中...';
    const [d1,d2]=await Promise.all([
        fetch(BASE+'?act=save_api1_accounts&key='+encodeURIComponent(KEY)+'&accounts='+encodeURIComponent(api1)).then(r=>r.json()),
        fetch(BASE+'?act=save_api2_accounts&key='+encodeURIComponent(KEY)+'&accounts='+encodeURIComponent(api2)).then(r=>r.json())
    ]);
    document.getElementById('apiAccountsResult').textContent=(d1.ok&&d2.ok)?`已保存 接口1 ${d1.count||0} 个，接口2 ${d2.count||0} 个`:(d1.msg||d2.msg||'保存失败');
}

// ===== Update =====
async function loadUpdateConfig(){
    const r=await fetch(BASE+'?act=update_config&key='+encodeURIComponent(KEY));
    const d=await r.json();
    if(!d.ok)return;
    const c=d.data||{};
    document.getElementById('updateBody').innerHTML=`
        <div style="background:#fffbe6;border:1px dashed #ffe58f;padding:10px 14px;border-radius:4px;margin-bottom:16px;font-size:13px;color:#666">App请求时带versionCode，后台版本号大于App才会弹窗更新。</div>
        <form id="updateForm" onsubmit="submitUpdate(event)">
        <div class="form-grid">
            <div class="form-item"><label>versionCode</label><input name="version_code" type="number" min="1" value="${escAttr(String(c.version_code||1))}"></div>
            <div class="form-item"><label>versionName</label><input name="version_name" value="${escAttr(c.version_name||'1.0.0')}"></div>
            <div class="form-item"><label>APK文件</label><input name="apk" type="file" accept=".apk"></div>
            <div class="form-item"><label>下载地址</label><input name="apk_url" value="${escAttr(c.apk_url||'')}" placeholder="上传APK后自动生成"></div>
            <div class="form-item"><label>弹窗标题</label><input name="title" value="${escAttr(c.title||'发现新版本')}"></div>
            <div class="form-check"><input name="force" type="checkbox" ${c.force?'checked':''}><label>强制更新</label></div>
            <div class="form-item" style="grid-column:1/-1"><label>更新说明</label><textarea name="message" placeholder="给用户看的更新内容">${esc(c.message||'')}</textarea></div>
        </div>
        <div style="margin-top:16px;display:flex;gap:10px;align-items:center">
            <button class="btn btn-primary" type="submit">保存并发布</button>
            ${c.apk_url?`<a href="${escAttr(c.apk_url)}" target="_blank" class="btn btn-default">测试下载</a>`:''}
            <span id="updateResult" style="font-size:13px;color:var(--muted)">${c.updated_at?'上次: '+esc(c.updated_at):''}</span>
        </div></form>`;
}
async function submitUpdate(e){
    e.preventDefault();
    const fd=new FormData(document.getElementById('updateForm'));
    fd.append('act','upload_update');fd.append('key',KEY);
    document.getElementById('updateResult').textContent='保存中...';
    const r=await fetch(BASE,{method:'POST',body:fd});
    const d=await r.json();
    document.getElementById('updateResult').textContent=d.msg||(d.ok?'已保存':'失败');
    if(d.ok) loadUpdateConfig();
}

// ===== Data =====
async function loadDumpList(){
    const r=await fetch(BASE+'?act=dumps&key='+encodeURIComponent(KEY));
    const d=await r.json();
    if(!d.ok)return;
    const el=document.getElementById('dumpList');
    const files=d.data||[];
    if(!files.length){el.innerHTML='<div class="empty">暂无解密数据</div>';return;}
    el.innerHTML=`<table><thead><tr><th>#</th><th>时间</th><th>ID</th><th>大小</th><th>操作</th></tr></thead><tbody>${files.map((f,i)=>`<tr><td>${i+1}</td><td>${esc(f.time)}</td><td style="font-size:12px;font-family:monospace">${esc(f.id)}</td><td>${(f.size/1024).toFixed(1)}KB</td><td><a href="${BASE}?act=dump&id=${encodeURIComponent(f.id)}&key=${encodeURIComponent(KEY)}" target="_blank" style="color:var(--primary)">查看</a></td></tr>`).join('')}</tbody></table>`;
}
async function openLatestDump(){
    const r=await fetch(BASE+'?act=dumps&key='+encodeURIComponent(KEY));
    const d=await r.json();
    const files=d.data||[];
    if(!files.length) return alert('暂无数据');
    window.open(BASE+'?act=dump&id='+encodeURIComponent(files[0].id)+'&key='+encodeURIComponent(KEY),'_blank');
}
</script>
</body>
</html>
<?php
}
