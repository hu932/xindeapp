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
    $httpCode = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);
    return ['response' => $response, 'error' => $error, 'http_code' => $httpCode, 'headers' => $headers];
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
    $username = trim((string)($input['username'] ?? $input['用户名'] ?? ''));
    $groupId = trim((string)($input['group_id'] ?? $input['组ID'] ?? ''));
    $deviceId = trim((string)($input['device_id'] ?? $input['设备ID'] ?? ''));
    if ($username === '') $username = 'unknown';
    if ($deviceId === '') $deviceId = 'unknown-' . substr(md5($_SERVER['REMOTE_ADDR'] ?? ''), 0, 8);

    $devices = readDeviceControls($file);
    $key = deviceKey($username, $deviceId);
    $old = $devices[$key] ?? [];
    $disabled = !empty($old['disabled']);
    $devices[$key] = array_merge($old, [
        'key' => $key,
        'username' => $username,
        'group_id' => $groupId,
        'device_id' => $deviceId,
        'api_type' => (int)($input['api_type'] ?? ($old['api_type'] ?? 0)),
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
        usort($devices, fn($a, $b) => strcmp((string)($b['last_seen'] ?? ''), (string)($a['last_seen'] ?? '')));
        jsonResp(['ok' => true, 'data' => $devices]);
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
        'device_key' => $deviceCheck['device_key'] ?? '',
        'encrypted_len' => strlen($encryptedData),
        'status'     => 'device_disabled',
        'error'      => $deviceCheck['msg'] ?? '设备已被禁用',
    ];
    appendLog($LOG_FILE, $logEntry, $MAX_LOGS);
    jsonResp(['ok' => false, 'msg' => $deviceCheck['msg'] ?? '设备已被禁用']);
}

$logEntry = [
    'time'       => date('Y-m-d H:i:s'),
    'api_type'   => $apiType,
    'ip'         => $_SERVER['REMOTE_ADDR'] ?? '',
    'username'   => $input['username'] ?? $input['用户名'] ?? '',
    'group_id'   => $input['group_id'] ?? $input['组ID'] ?? '',
    'device_id'  => $input['device_id'] ?? $input['设备ID'] ?? '',
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
    $taskPayload = is_array($taskInfo) ? [
        'ID' => (string)($taskInfo['ID'] ?? ''),
        'TaskNumber' => (string)($taskInfo['TaskNumber'] ?? ''),
        'TaskType' => (string)($taskInfo['TaskType'] ?? ''),
    ] : $taskInfo;

    $payload = [
        '用户名' => $username, '组ID' => $groupId,
        '任务数据' => $taskPayload, 'data' => $actualData,
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

    $payload = [
        'appVersion' => 'vv2', 'url' => $taskUrl, 'result' => $plain,
    ];
    $logEntry['dump_id'] = saveDecryptDump($encryptedData, $plain, $apiType, $decrypted, $productResponse, $payload);

    $forwardBody = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

// 转发
$logEntry['forward_url'] = $target['url'];
$result = forwardToServer($target['url'], $forwardBody, $target['gzip']);
$serverResp = json_decode((string)$result['response'], true);
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
    jsonResp(['ok' => false, 'msg' => '转发失败: ' . $result['error']]);
}

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
<title>任务代理控制台</title>
<style>
:root{--bg:#f4f6f8;--panel:#fff;--line:#e6eaf0;--text:#172033;--muted:#667085;--blue:#2563eb;--green:#16a34a;--red:#dc2626;--amber:#d97706;--cyan:#0891b2;--violet:#7c3aed}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:var(--bg);color:var(--text);padding:0}
.shell{max-width:1280px;margin:0 auto;padding:22px}
.top{background:#111827;color:#fff;padding:18px 22px;border-radius:8px;margin-bottom:16px;display:flex;justify-content:space-between;gap:18px;align-items:center}
.top h1{font-size:20px;line-height:1.2;margin-bottom:6px}
.top p{font-size:13px;color:#cbd5e1}
.keybox{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
.keybox input{height:36px;width:190px;border:1px solid #334155;border-radius:6px;background:#0f172a;color:#fff;padding:0 10px}
.bar{display:flex;gap:8px;margin-bottom:16px;flex-wrap:wrap}
button{height:36px;border:none;border-radius:6px;cursor:pointer;font-size:13px;font-weight:700;color:#fff;padding:0 14px}
.btn-load{background:var(--blue)} .btn-clear{background:var(--red)} .btn-auto{background:var(--green)} .btn-dump{background:var(--violet)} .btn-devices{background:var(--cyan)} .btn-update{background:var(--amber)} .btn-secondary{background:#475569}
.stats{display:grid;grid-template-columns:repeat(3,minmax(160px,1fr));gap:12px;margin-bottom:16px}
.stat{background:var(--panel);padding:14px 16px;border-radius:8px;border:1px solid var(--line)}
.stat b{font-size:26px;display:block;margin-top:4px}
.panel{background:var(--panel);border:1px solid var(--line);border-radius:8px;margin-bottom:16px;overflow:hidden}
.panel-head{padding:14px 16px;border-bottom:1px solid var(--line);display:flex;justify-content:space-between;gap:10px;align-items:center}
.panel-head h2{font-size:16px}
.panel-head p{font-size:12px;color:var(--muted);margin-top:3px}
.panel-body{padding:16px}
.grid{display:grid;grid-template-columns:repeat(2,minmax(220px,1fr));gap:12px}
.field label{display:block;font-size:12px;color:var(--muted);font-weight:700;margin-bottom:6px}
.field input,.field textarea{width:100%;border:1px solid var(--line);border-radius:6px;padding:9px 10px;font-size:14px;background:#fff;color:var(--text)}
.field textarea{min-height:86px;resize:vertical}
.field small{display:block;color:var(--muted);font-size:12px;margin-top:5px}
.check{display:flex;align-items:center;gap:8px;font-size:14px;color:var(--text);padding-top:24px}
.check input{width:16px;height:16px}
.form-actions{display:flex;gap:8px;align-items:center;margin-top:14px;flex-wrap:wrap}
.notice{background:#f8fafc;border:1px dashed #cbd5e1;color:#475569;border-radius:6px;padding:10px 12px;font-size:13px;margin-bottom:12px}
.device-list{display:none;flex-direction:column;gap:8px}
.log-list{display:flex;flex-direction:column;gap:8px}
.log-item,.device-item{background:#fff;border-radius:8px;padding:14px;border:1px solid var(--line);border-left:4px solid #94a3b8}
.log-item.success,.device-item{border-left-color:var(--green)}
.device-item.disabled,.log-item.decrypt_fail,.log-item.json_fail{border-left-color:var(--red)}
.log-item.forward_fail,.log-item.task_server_error,.log-item.device_disabled{border-left-color:var(--amber)}
.log-meta,.device-meta{font-size:12px;color:var(--muted);margin-bottom:6px;display:flex;gap:12px;flex-wrap:wrap}
.log-body{font-size:13px;color:#243044;line-height:1.8}
.log-body code{background:#eef2f7;padding:2px 6px;border-radius:4px;font-size:12px}
.log-actions,.device-actions{margin-top:10px;display:flex;gap:8px;flex-wrap:wrap}
.log-actions a{display:inline-flex;align-items:center;justify-content:center;height:30px;padding:0 10px;border-radius:6px;background:var(--blue);color:#fff;text-decoration:none;font-size:12px;font-weight:700}
.link-btn{display:inline-flex;height:36px;align-items:center;padding:0 12px;background:var(--blue);color:#fff;border-radius:6px;text-decoration:none;font-size:13px;font-weight:700}
.tag{display:inline-block;padding:3px 8px;border-radius:999px;font-size:11px;color:#fff;font-weight:700}
.tag-1{background:var(--blue)} .tag-2{background:var(--violet)}
.tag-ok{background:var(--green)} .tag-fail{background:var(--red)} .tag-warn{background:var(--amber)}
pre{background:#0f172a;color:#dbeafe;padding:12px;border-radius:6px;font-size:12px;overflow-x:auto;max-height:220px;margin-top:8px}
.empty{text-align:center;padding:38px;color:var(--muted);font-size:15px}
#autoLabel{font-size:13px;color:var(--muted)}
.hidden{display:none}
.btn-disable{background:var(--red)}.btn-enable{background:var(--green)}
@media(max-width:760px){.shell{padding:12px}.top{display:block}.keybox{margin-top:12px}.stats,.grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="shell">
<div class="top">
    <div>
        <h1>任务代理控制台</h1>
        <p>日志监控、设备下线、APK在线发布统一管理</p>
    </div>
    <div class="keybox">
        <input type="password" id="key" placeholder="查看密码" value="">
    </div>
</div>
<div class="bar">
    <button class="btn-load" onclick="loadLogs()">加载日志</button>
    <button class="btn-dump" onclick="openLatestDump()">最新解密数据</button>
    <button class="btn-dump" onclick="openDumpList()">解密数据列表</button>
    <button class="btn-devices" onclick="loadDevices()">设备控制</button>
    <button class="btn-update" onclick="openUpdateConfig()">更新配置</button>
    <button class="btn-auto" onclick="toggleAuto()" id="autoBtn">自动刷新</button>
    <button class="btn-clear" onclick="clearLogs()">清空日志</button>
    <span id="autoLabel"></span>
</div>
<div class="stats" id="stats"></div>
<div class="panel hidden" id="updatePanel"></div>
<div class="panel hidden" id="devicePanel"><div class="panel-head"><div><h2>设备控制</h2><p>禁用后App心跳和上传都会被拦截</p></div></div><div class="panel-body"><div class="device-list" id="deviceList"></div></div></div>
<div class="panel"><div class="panel-head"><div><h2>转发日志</h2><p>查看解密、上传和任务服务器响应</p></div></div><div class="panel-body"><div class="log-list" id="logList"><div class="empty">输入密码后点击加载</div></div></div></div>
</div>

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
        if (l.encrypted_len) body += `<b>加密:</b> <code>${(l.encrypted_len/1024).toFixed(1)}KB</code> `;
        if (l.decrypted_len) body += `<b>解密:</b> <code>${(l.decrypted_len/1024).toFixed(1)}KB</code> `;
        if (l.extract_from) body += `<b>提取:</b> <code>${escHtml(String(l.extract_from))}</code> `;
        if (l.forward_data_type) body += `<b>data类型:</b> <code>${escHtml(String(l.forward_data_type))}</code> `;

        let resp = '';
        if (l.response) {
            const rs = typeof l.response === 'string' ? l.response : JSON.stringify(l.response, null, 2);
            resp = `<pre>${escHtml(rs.substring(0, 500))}</pre>`;
        }
        if (l.error) resp += `<pre style="color:#ff6b6b">${escHtml(l.error)}</pre>`;
        const dumpHref = l.dump_id ? `${BASE}?act=dump&id=${encodeURIComponent(l.dump_id)}&key=${encodeURIComponent(getKey())}` : '';
        const uploadHref = l.upload_id ? `${BASE}?act=upload&id=${encodeURIComponent(l.upload_id)}&key=${encodeURIComponent(getKey())}` : '';
        const actions = `<div class="log-actions">${
            dumpHref
                ? `<a href="${dumpHref}" target="_blank" rel="noopener">查看解密数据</a>`
                : `<a href="javascript:void(0)" onclick="alert('这条旧日志没有保存解密数据，请重新跑一次任务后再点查看。')">查看解密数据</a>`
        } ${
            uploadHref
                ? `<a href="${uploadHref}" target="_blank" rel="noopener">查看上传详情</a>`
                : `<a href="javascript:void(0)" onclick="alert('这条旧日志没有保存上传详情，请重新跑一次任务后再点查看。')">查看上传详情</a>`
        }</div>`;

        return `<div class="log-item ${cls}">
            <div class="log-meta">${apiTag} ${statusTag} <span>${l.time}</span> <span>IP: ${l.ip || '-'}</span></div>
            <div class="log-body">${body}</div>
            ${actions}
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

function escHtml(s) { return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function escHtmlAttr(s) { return escHtml(s).replace(/"/g,'&quot;'); }

async function loadDevices() {
    const key = getKey();
    if (!key) return alert('请输入密码');
    try {
        const r = await fetch(BASE + '?act=devices&key=' + encodeURIComponent(key));
        const d = await r.json();
        if (!d.ok) return alert(d.msg || '加载设备失败');
        document.getElementById('devicePanel').classList.remove('hidden');
        renderDevices(d.data || []);
    } catch (e) { alert('请求失败: ' + e.message); }
}

function renderDevices(devices) {
    const el = document.getElementById('deviceList');
    el.style.display = 'flex';
    if (!devices.length) {
        el.innerHTML = '<div class="empty">暂无设备心跳。App启动后会自动上报。</div>';
        return;
    }
    el.innerHTML = devices.map(d => {
        const disabled = !!d.disabled;
        const status = disabled ? '<span class="tag tag-fail">已禁用</span>' : '<span class="tag tag-ok">允许使用</span>';
        const btn = disabled
            ? `<button class="btn-enable" onclick="toggleDevice('${escAttr(d.key)}', 0)">启用设备</button>`
            : `<button class="btn-disable" onclick="toggleDevice('${escAttr(d.key)}', 1)">禁用设备</button>`;
        return `<div class="device-item ${disabled ? 'disabled' : ''}">
            <div><b>${escHtml(d.username || '-')}</b> ${status}</div>
            <div class="device-meta">
                <span>组ID: ${escHtml(d.group_id || '-')}</span>
                <span>接口: ${escHtml(String(d.api_type || '-'))}</span>
                <span>设备ID: ${escHtml(d.device_id || '-')}</span>
                <span>IP: ${escHtml(d.ip || '-')}</span>
                <span>首次: ${escHtml(d.first_seen || '-')}</span>
                <span>最后心跳: ${escHtml(d.last_seen || '-')}</span>
            </div>
            ${d.disabled_reason ? `<pre>${escHtml(d.disabled_reason)}</pre>` : ''}
            <div class="device-actions">${btn}</div>
        </div>`;
    }).join('');
}

async function toggleDevice(deviceKey, disabled) {
    const key = getKey();
    if (!key) return alert('请输入密码');
    let reason = '';
    if (disabled) {
        reason = prompt('禁用原因，App会显示这段提示', '账号已被下线，禁止使用') || '';
        if (!reason) return;
    }
    const url = BASE + '?act=device_toggle'
        + '&key=' + encodeURIComponent(key)
        + '&device_key=' + encodeURIComponent(deviceKey)
        + '&disabled=' + encodeURIComponent(String(disabled))
        + '&reason=' + encodeURIComponent(reason);
    const r = await fetch(url);
    const d = await r.json();
    alert(d.msg || (d.ok ? '操作成功' : '操作失败'));
    loadDevices();
}

function escAttr(s) { return String(s || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'"); }

async function openUpdateConfig() {
    const key = getKey();
    if (!key) return alert('请输入密码');
    const r = await fetch(BASE + '?act=update_config&key=' + encodeURIComponent(key));
    const d = await r.json();
    if (!d.ok) return alert(d.msg || '读取更新配置失败');
    renderUpdatePanel(d.data || {});
}

function renderUpdatePanel(c) {
    const panel = document.getElementById('updatePanel');
    panel.classList.remove('hidden');
    panel.innerHTML = `
        <div class="panel-head">
            <div>
                <h2>在线更新发布</h2>
                <p>上传APK后会自动生成下载地址；App当前versionCode是1，后台版本号填2或更高才会弹窗</p>
            </div>
            <button class="btn-secondary" onclick="document.getElementById('updatePanel').classList.add('hidden')">收起</button>
        </div>
        <div class="panel-body">
            <div class="notice">版本控制规则：App请求时会带当前versionCode；只有后台versionCode大于App当前versionCode，并且已上传APK，App才会提示下载更新。</div>
            <form id="updateForm" onsubmit="submitUpdateForm(event)">
                <div class="grid">
                    <div class="field">
                        <label>versionCode</label>
                        <input name="version_code" type="number" min="1" value="${escHtmlAttr(String(c.version_code || 1))}">
                        <small>每次发布新包必须递增，例如当前App是1，新包填2。</small>
                    </div>
                    <div class="field">
                        <label>versionName</label>
                        <input name="version_name" value="${escHtmlAttr(c.version_name || '1.0.0')}">
                    </div>
                    <div class="field">
                        <label>APK文件</label>
                        <input name="apk" type="file" accept=".apk,application/vnd.android.package-archive">
                        <small>选择APK上传后自动生成下载地址。</small>
                    </div>
                    <div class="field">
                        <label>当前下载地址</label>
                        <input name="apk_url" value="${escHtmlAttr(c.apk_url || '')}" placeholder="也可以手动填外部APK地址">
                    </div>
                    <div class="field">
                        <label>弹窗标题</label>
                        <input name="title" value="${escHtmlAttr(c.title || '发现新版本')}">
                    </div>
                    <label class="check"><input name="force" type="checkbox" ${c.force ? 'checked' : ''}> 强制更新</label>
                    <div class="field" style="grid-column:1/-1">
                        <label>更新说明</label>
                        <textarea name="message" placeholder="给用户看的更新内容">${escHtml(c.message || '')}</textarea>
                    </div>
                </div>
                <div class="form-actions">
                    <button class="btn-update" type="submit">保存并发布</button>
                    ${c.apk_url ? `<a class="link-btn" href="${escHtmlAttr(c.apk_url)}" target="_blank" rel="noopener">测试下载</a>` : ''}
                    <span id="updateResult" style="font-size:13px;color:var(--muted)">${c.updated_at ? '上次更新: ' + escHtml(c.updated_at) : ''}</span>
                </div>
            </form>
        </div>`;
}

async function submitUpdateForm(event) {
    event.preventDefault();
    const key = getKey();
    if (!key) return alert('请输入密码');
    const form = document.getElementById('updateForm');
    const fd = new FormData(form);
    fd.append('act', 'upload_update');
    fd.append('key', key);
    const resultEl = document.getElementById('updateResult');
    resultEl.textContent = '正在上传保存...';
    try {
        const r = await fetch(BASE, { method: 'POST', body: fd });
        const d = await r.json();
        resultEl.textContent = d.msg || (d.ok ? '保存成功' : '保存失败');
        if (d.ok) renderUpdatePanel(d.data || {});
    } catch (e) {
        resultEl.textContent = '请求失败: ' + e.message;
    }
}

async function getDumpList() {
    const key = getKey();
    if (!key) { alert('请输入密码'); return []; }
    const r = await fetch(BASE + '?act=dumps&key=' + encodeURIComponent(key));
    const d = await r.json();
    if (!d.ok) { alert(d.msg || '加载解密数据失败'); return []; }
    return d.data || [];
}

async function openLatestDump() {
    const dumps = await getDumpList();
    if (!dumps.length) return alert('暂无解密数据。旧日志没有保存 dump，请重新跑一次任务。');
    window.open(BASE + '?act=dump&id=' + encodeURIComponent(dumps[0].id) + '&key=' + encodeURIComponent(getKey()), '_blank');
}

async function openDumpList() {
    const dumps = await getDumpList();
    if (!dumps.length) return alert('暂无解密数据。旧日志没有保存 dump，请重新跑一次任务。');
    const text = dumps.map((d, i) => `${i + 1}. ${d.time}  ${d.id}  ${(d.size / 1024).toFixed(1)}KB`).join('\n');
    const pick = prompt('输入序号打开解密数据：\n\n' + text, '1');
    const idx = parseInt(pick || '', 10) - 1;
    if (!Number.isInteger(idx) || idx < 0 || idx >= dumps.length) return;
    window.open(BASE + '?act=dump&id=' + encodeURIComponent(dumps[idx].id) + '&key=' + encodeURIComponent(getKey()), '_blank');
}

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
