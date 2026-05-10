<?php
header('Content-Type: application/json; charset=utf-8');

$AES_KEY = '25d9a1907ab38e273c4f57c476e64377';

$TARGETS = [
    1 => [
        'url' => 'https://eqwofaygdsjko.uk:443/api/task/submit/v2',
        'gzip' => false,
    ],
    2 => [
        'url' => 'http://103.143.80.158:2000/up',
        'gzip' => true,
    ],
];

function aesDecrypt(string $encrypted, string $key): ?string {
    $decoded = base64_decode($encrypted, true);
    if ($decoded === false) return null;
    $plain = openssl_decrypt($decoded, 'aes-256-ecb', $key, OPENSSL_RAW_DATA);
    return $plain === false ? null : $plain;
}

function forwardToServer(string $url, string $body, bool $gzip): array {
    $headers = [
        'Content-Type: application/json; charset=utf-8',
        'Accept: application/json',
    ];
    $postData = $body;
    if ($gzip) {
        $postData = gzencode($body);
        $headers[] = 'Content-Encoding: gzip';
    }

    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $postData,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT => 15,
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_SSL_VERIFYPEER => false,
    ]);
    $response = curl_exec($ch);
    $error = curl_error($ch);
    curl_close($ch);
    return ['response' => $response, 'error' => $error];
}

function jsonResp(array $data): void {
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

// ===== 读取请求 =====
$rawBody = file_get_contents('php://input');
if (!$rawBody) jsonResp(['ok' => false, 'msg' => '空请求体']);

$input = json_decode($rawBody, true);
if (!$input) jsonResp(['ok' => false, 'msg' => 'JSON解析失败']);

$apiType = (int)($input['api_type'] ?? 0);
$encryptedData = $input['data'] ?? '';

if (!$apiType || !isset($TARGETS[$apiType])) {
    jsonResp(['ok' => false, 'msg' => '无效的api_type，需要1或2']);
}

if (!$encryptedData || !is_string($encryptedData)) {
    jsonResp(['ok' => false, 'msg' => '缺少data字段']);
}

// ===== 解密 =====
$plain = aesDecrypt($encryptedData, $AES_KEY);
if ($plain === null) {
    jsonResp(['ok' => false, 'msg' => 'AES解密失败']);
}

$decrypted = json_decode($plain, true);
if ($decrypted === null && json_last_error() !== JSON_ERROR_NONE) {
    jsonResp(['ok' => false, 'msg' => '解密后非JSON: ' . json_last_error_msg(), 'preview' => mb_substr($plain, 0, 200)]);
}

// ===== 按接口类型构建转发请求 =====
$target = $TARGETS[$apiType];

if ($apiType === 2) {
    $username = $input['用户名'] ?? '';
    $groupId = $input['组ID'] ?? '';
    $taskInfo = $input['任务数据'] ?? null;

    if (!$username || !$groupId || !$taskInfo) {
        jsonResp(['ok' => false, 'msg' => '接口2缺少 用户名/组ID/任务数据']);
    }

    $forwardBody = json_encode([
        '用户名' => $username,
        '组ID' => $groupId,
        '任务数据' => $taskInfo,
        'data' => $decrypted,
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);

} elseif ($apiType === 1) {
    $taskUrl = $input['url'] ?? '';
    if (!$taskUrl) {
        jsonResp(['ok' => false, 'msg' => '接口1缺少url']);
    }

    $forwardBody = json_encode([
        'appVersion' => 'vv2',
        'url' => $taskUrl,
        'result' => $plain,
    ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

// ===== 转发到任务服务器 =====
$result = forwardToServer($target['url'], $forwardBody, $target['gzip']);

if ($result['error']) {
    jsonResp(['ok' => false, 'msg' => '转发失败: ' . $result['error']]);
}

$serverResp = json_decode($result['response'], true);
if ($serverResp) {
    jsonResp($serverResp);
} else {
    jsonResp(['ok' => false, 'msg' => '任务服务器响应异常', 'raw' => $result['response']]);
}
