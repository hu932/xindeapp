<?php
/**
 * 优惠券预检服务器 - 公共逻辑
 *
 * 本文件实现与插件 coupon-preflight.js 的接口对接：
 *   POST /api/coupon-preflight/start
 *   POST /api/coupon-preflight/report
 *
 * 返回统一信封：{"success": bool, "error": string|null, "data": object|null}
 */

require_once __DIR__ . '/config.php';

$__coupon_config = null;

function coupon_config() {
    global $__coupon_config;
    if ($__coupon_config === null) {
        $__coupon_config = require __DIR__ . '/config.php';
    }
    return $__coupon_config;
}

/* ------------------------------ 基础工具 ------------------------------ */

function coupon_json($data, $httpCode = 200) {
    http_response_code($httpCode);
    header('Content-Type: application/json; charset=utf-8');
    header('Access-Control-Allow-Origin: *');
    header('Access-Control-Allow-Headers: Content-Type, Authorization');
    header('Access-Control-Allow-Methods: POST, OPTIONS');
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function coupon_success($data) {
    coupon_json(['success' => true, 'error' => null, 'data' => $data]);
}

function coupon_fail($message, $httpCode = 200) {
    coupon_json(['success' => false, 'error' => $message, 'data' => null], $httpCode);
}

function coupon_handle_options() {
    if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') {
        header('Access-Control-Allow-Origin: *');
        header('Access-Control-Allow-Headers: Content-Type, Authorization');
        header('Access-Control-Allow-Methods: POST, OPTIONS');
        http_response_code(204);
        exit;
    }
}

function coupon_get_bearer() {
    $auth = '';
    if (function_exists('getallheaders')) {
        foreach (getallheaders() as $k => $v) {
            if (strcasecmp((string)$k, 'Authorization') === 0) {
                $auth = (string)$v;
                break;
            }
        }
    }
    if ($auth === '' && isset($_SERVER['HTTP_AUTHORIZATION'])) {
        $auth = (string)$_SERVER['HTTP_AUTHORIZATION'];
    }
    if ($auth === '' && isset($_SERVER['REDIRECT_HTTP_AUTHORIZATION'])) {
        $auth = (string)$_SERVER['REDIRECT_HTTP_AUTHORIZATION'];
    }
    if (preg_match('/Bearer\s+(.+)/i', $auth, $m)) {
        return trim($m[1]);
    }
    return '';
}

function coupon_check_auth() {
    $token = coupon_get_bearer();
    if ($token === '' || !hash_equals((string)coupon_config()['auth_token'], $token)) {
        coupon_fail('未授权：token 无效', 200);
    }
}

function coupon_read_body() {
    $raw = file_get_contents('php://input');
    if ($raw === false || trim($raw) === '') {
        return [];
    }
    $data = json_decode($raw, true);
    if (!is_array($data)) {
        coupon_fail('请求体必须是 JSON', 200);
    }
    return $data;
}

function coupon_shanghai_date() {
    $dt = new DateTime('now', new DateTimeZone('Asia/Shanghai'));
    return $dt->format('Y-m-d');
}

function coupon_strlen($s) {
    if (function_exists('mb_strlen')) {
        return mb_strlen($s, 'UTF-8');
    }
    return strlen($s);
}

function coupon_sanitize_id($id) {
    return preg_replace('/[^A-Za-z0-9_.\-]/', '_', (string)$id);
}

/* ------------------------------ 数据解密 ------------------------------ */

function coupon_aes_decrypt($encrypted, $key) {
    $decoded = base64_decode($encrypted, true);
    if ($decoded === false) return null;
    $plain = openssl_decrypt($decoded, 'aes-256-ecb', $key, OPENSSL_RAW_DATA);
    return $plain === false ? null : $plain;
}

/**
 * 安卓 App 上报的 rawResponseText：可能是明文 JSON（直接返回），
 * 也可能是 AES 加密的 Mars 包装 JSON，需解密后取出 responseContent.response_body / response_body。
 */
function coupon_decrypt_raw($data) {
    $key = (string)coupon_config()['aes_key'];
    $data = trim($data);
    if ($data === '') return null;
    $plainInput = json_decode($data, true);
    if (is_array($plainInput)) return $data;
    $plain = coupon_aes_decrypt($data, $key);
    if ($plain === null) return null;
    $decrypted = json_decode($plain, true);
    if (!is_array($decrypted)) return $plain;
    $rc = isset($decrypted['responseContent']) && is_array($decrypted['responseContent']) ? $decrypted['responseContent'] : null;
    if ($rc !== null) {
        $rb = isset($rc['response_body']) && is_string($rc['response_body']) && $rc['response_body'] !== '' ? $rc['response_body'] : '';
        if ($rb !== '') return $rb;
    }
    foreach (['response_body', 'responseBody'] as $k) {
        if (isset($decrypted[$k]) && is_string($decrypted[$k]) && $decrypted[$k] !== '') {
            return $decrypted[$k];
        }
    }
    return $plain;
}

/* ------------------------------ 存储层 ------------------------------ */

function coupon_storage_dir() {
    $dir = rtrim((string)coupon_config()['storage_dir'], '/\\');
    if (!is_dir($dir)) {
        @mkdir($dir, 0755, true);
    }
    return $dir;
}

function coupon_storage_path($name) {
    return coupon_storage_dir() . '/' . $name;
}

function coupon_storage_read($name, $default = null) {
    $path = coupon_storage_path($name);
    if (!file_exists($path)) {
        return $default;
    }
    $raw = file_get_contents($path);
    $data = json_decode($raw === false ? '' : $raw, true);
    return is_array($data) ? $data : $default;
}

function coupon_storage_write($name, $data) {
    $path = coupon_storage_path($name);
    $fp = @fopen($path, 'c');
    if (!$fp) {
        coupon_fail('存储目录不可写，请检查 storage 目录权限', 200);
    }
    if (!flock($fp, LOCK_EX)) {
        fclose($fp);
        coupon_fail('文件锁定失败', 200);
    }
    ftruncate($fp, 0);
    rewind($fp);
    fwrite($fp, json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
    fflush($fp);
    flock($fp, LOCK_UN);
    fclose($fp);
}

/* ------------------------------ 业务接口 ------------------------------ */

/**
 * 判断采集到的 PDP 数据里是否包含优惠券信息。
 * 依据两个通过样本（1.josn / 2.josn）比对，两者共同包含
 * data.promotion_drawer.shop_vouchers（卖场优惠券）等字段。
 */
function coupon_has_coupon_data($data) {
    if (!is_array($data)) {
        return false;
    }
    $root = isset($data['data']) && is_array($data['data']) ? $data['data'] : $data;

    // 1) 顶层直接挂的优惠券数组（新版 get_rw 结构，如 data.shop_vouchers）
    foreach (['shop_vouchers', 'platform_vouchers', 'ads_vouchers', 'vouchers'] as $key) {
        if (!empty($root[$key]) && is_array($root[$key])) {
            return true;
        }
    }

    // 2) 促销抽屉里的优惠券：卖场券 / 平台券 / 广告券（旧版结构）
    $pd = isset($root['promotion_drawer']) && is_array($root['promotion_drawer']) ? $root['promotion_drawer'] : [];
    foreach (['shop_vouchers', 'platform_vouchers', 'ads_vouchers', 'vouchers'] as $key) {
        if (!empty($pd[$key]) && is_array($pd[$key])) {
            return true;
        }
    }

    // 3) 券后价优惠券 final_price_vouchers
    $pp = isset($root['product_price']) && is_array($root['product_price']) ? $root['product_price'] : [];
    if (!empty($pp['final_price_vouchers']) && is_array($pp['final_price_vouchers'])) {
        return true;
    }

    // 4) labels 里带 voucher 标记
    if (!empty($pp['labels']) && is_array($pp['labels'])) {
        foreach ($pp['labels'] as $label) {
            if (is_array($label) && (($label['display'] ?? '') === 'voucher')) {
                return true;
            }
        }
    }

    // 5) 价格拆解里的优惠券折扣项
    $pb = isset($root['price_breakdown']) && is_array($root['price_breakdown']) ? $root['price_breakdown'] : [];
    if (!empty($pb['discount_breakdown']) && is_array($pb['discount_breakdown'])) {
        foreach ($pb['discount_breakdown'] as $item) {
            if (is_array($item) && (!empty($item['shop_voucher']) || !empty($item['platform_voucher']) || !empty($item['ads_voucher']))) {
                return true;
            }
        }
    }

    return false;
}

/**
 * POST /api/coupon-preflight/start
 * 请求：{"shopeeUserId": "...", "shopeeUsername": "..."}
 * 返回 data：{"allowed": bool, "required": bool, "dayKey": "...", "probe": {...}}
 */
function handle_coupon_start() {
    coupon_check_auth();
    $body = coupon_read_body();

    $userId = isset($body['shopeeUserId']) ? trim((string)$body['shopeeUserId']) : '';
    if ($userId === '') {
        coupon_fail('缺少参数 shopeeUserId', 200);
    }
    $username = isset($body['shopeeUsername']) ? (string)$body['shopeeUsername'] : '';

    $dayKey = coupon_shanghai_date();

    // 当天已通过检测的账号直接放行
    $passedFile = 'passed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json';
    if (file_exists(coupon_storage_path($passedFile))) {
        return coupon_success([
            'allowed'  => true,
            'required' => false,
            'dayKey'   => $dayKey,
        ]);
    }

    // 未通过：需要检测，下发一个探测商品
    $products = coupon_probe_products();
    if (empty($products)) {
        coupon_fail('服务器未配置探测商品（probe_products）', 200);
    }
    $probe = $products[array_rand($products)];

    // 记录本次分配的探测任务，report 时校验 probeId
    coupon_storage_write('session_' . coupon_sanitize_id($userId) . '.json', [
        'probeId'        => (string)$probe['id'],
        'dayKey'         => $dayKey,
        'assignedAt'     => time(),
        'shopeeUserId'   => $userId,
        'shopeeUsername' => $username,
    ]);

    return coupon_success([
        'allowed'  => false,
        'required' => true,
        'dayKey'   => $dayKey,
        'probe'    => [
            'id'     => (string)$probe['id'],
            'url'    => (string)$probe['url'],
            'shopId' => isset($probe['shopId']) ? (string)$probe['shopId'] : '0',
            'itemId' => isset($probe['itemId']) ? (string)$probe['itemId'] : '0',
        ],
    ]);
}

/**
 * POST /api/coupon-preflight/report
 * 请求：{"shopeeUserId": "...", "shopeeUsername": "...", "probeId": "...", "status": "captured", "rawResponseText": "..."}
 * 返回 data：{"passed": bool, "reason": "...", "dayKey": "..."}
 */
function handle_coupon_report() {
    coupon_check_auth();
    $body = coupon_read_body();

    $userId = isset($body['shopeeUserId']) ? trim((string)$body['shopeeUserId']) : '';
    if ($userId === '') {
        coupon_fail('缺少参数 shopeeUserId', 200);
    }
    $probeId = isset($body['probeId']) ? (string)$body['probeId'] : '';
    $status  = isset($body['status']) ? (string)$body['status'] : '';
    $raw     = isset($body['rawResponseText']) ? (string)$body['rawResponseText'] : '';

    $cfg = coupon_config();
    $dayKey = coupon_shanghai_date();

    $sessionFile = 'session_' . coupon_sanitize_id($userId) . '.json';
    $session = coupon_storage_read($sessionFile, null);
    if (!$session) {
        // 幂等处理：页面可能会多次上报，session 已被第一次 report 处理掉
        $passedFile = 'passed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json';
        if (file_exists(coupon_storage_path($passedFile))) {
            return coupon_success(['passed' => true, 'dayKey' => $dayKey]);
        }
        $failed = coupon_storage_read('failed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json', null);
        if ($failed) {
            return coupon_success(['passed' => false, 'reason' => $failed['reason'] ?? '检测未通过', 'dayKey' => $dayKey]);
        }
        return coupon_success(['passed' => false, 'reason' => '未找到检测任务，请先调用 start 接口', 'dayKey' => $dayKey]);
    }

    // 校验失败时：删掉 session，并写一条失败记录，方便后台查看
    $fail = function ($reason) use ($sessionFile, $dayKey, $userId, $probeId, $raw) {
        coupon_storage_delete($sessionFile);
        coupon_storage_write('failed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json', [
            'probeId'      => $probeId,
            'failedAt'     => time(),
            'shopeeUserId' => $userId,
            'reason'       => $reason,
        ]);
        // 诊断用：把原始上报数据存下来，排查「页面有券但判定无券」的结构差异
        @file_put_contents(
            coupon_storage_path('debug_raw_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json'),
            $raw
        );
        return coupon_success(['passed' => false, 'reason' => $reason, 'dayKey' => $dayKey]);
    };

    if ($session['probeId'] !== $probeId) {
        return $fail('probeId 不匹配');
    }
    if ($status !== 'captured') {
        return $fail('status 必须为 captured');
    }

    // 安卓 App 上报的数据是 AES 加密的 Mars 包装，先解密取出真正的响应
    $decryptedRaw = coupon_decrypt_raw($raw);
    if ($decryptedRaw === null) {
        return $fail('采集数据解密失败');
    }
    $raw = $decryptedRaw;

    if (coupon_strlen($raw) < (int)$cfg['report_min_length']) {
        return $fail('采集数据为空或过短');
    }

    $decoded = null;
    if (!empty($cfg['report_require_json']) || !empty($cfg['report_require_coupon'])) {
        $decoded = json_decode($raw, true);
        if (json_last_error() !== JSON_ERROR_NONE || !is_array($decoded)) {
            return $fail('采集数据不是合法 JSON');
        }
    }

    // 校验采集数据里确实包含优惠券信息（对比两个通过样本后得到的关键字段）
    if (!empty($cfg['report_require_coupon']) && !coupon_has_coupon_data($decoded)) {
        return $fail('采集数据中未检测到优惠券信息');
    }

    // 通过：先删掉检测任务，再记录当天已通过，账号只进「已通过」
    coupon_storage_delete($sessionFile);
    // 当天如果之前失败过，通过后把失败记录一并清掉，避免后台同时显示两个状态
    coupon_storage_delete('failed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json');
    coupon_storage_write('passed_' . coupon_sanitize_id($userId) . '_' . $dayKey . '.json', [
        'probeId'      => $probeId,
        'passedAt'     => time(),
        'shopeeUserId' => $userId,
    ]);

    return coupon_success([
        'passed' => true,
        'dayKey' => $dayKey,
    ]);
}

/* ------------------------------ 后台辅助 ------------------------------ */

function coupon_admin_config() {
    $cfg = coupon_config();
    return [
        'user' => isset($cfg['admin_user']) ? (string)$cfg['admin_user'] : 'admin',
        'pass' => isset($cfg['admin_password']) ? (string)$cfg['admin_password'] : 'admin123',
    ];
}

function coupon_probe_products() {
    $list = coupon_storage_read('probe_products.json', null);
    if ($list === null) {
        $list = isset(coupon_config()['probe_products']) ? coupon_config()['probe_products'] : [];
        $list = array_values($list);
        coupon_storage_write('probe_products.json', $list);
    }
    return array_values($list);
}

function coupon_save_probe_products($list) {
    coupon_storage_write('probe_products.json', array_values($list));
}

function coupon_storage_delete($name) {
    $path = coupon_storage_path(basename($name));
    if (file_exists($path)) {
        @unlink($path);
    }
}

function coupon_passed_records() {
    $dir = coupon_storage_dir();
    $records = [];
    foreach (glob($dir . '/passed_*.json') ?: [] as $file) {
        $data = json_decode(@file_get_contents($file), true);
        if (is_array($data)) {
            $data['_file'] = basename($file);
            $records[] = $data;
        }
    }
    usort($records, function ($a, $b) {
        return ((int)($b['passedAt'] ?? 0)) <=> ((int)($a['passedAt'] ?? 0));
    });
    return $records;
}

function coupon_session_records() {
    $dir = coupon_storage_dir();
    $records = [];
    foreach (glob($dir . '/session_*.json') ?: [] as $file) {
        $data = json_decode(@file_get_contents($file), true);
        if (is_array($data)) {
            $data['_file'] = basename($file);
            $records[] = $data;
        }
    }
    usort($records, function ($a, $b) {
        return ((int)($b['assignedAt'] ?? 0)) <=> ((int)($a['assignedAt'] ?? 0));
    });
    return $records;
}

function coupon_clear_passed_records() {
    foreach (glob(coupon_storage_dir() . '/passed_*.json') ?: [] as $file) {
        @unlink($file);
    }
}

function coupon_failed_records() {
    $dir = coupon_storage_dir();
    $records = [];
    foreach (glob($dir . '/failed_*.json') ?: [] as $file) {
        $data = json_decode(@file_get_contents($file), true);
        if (is_array($data)) {
            $data['_file'] = basename($file);
            $records[] = $data;
        }
    }
    usort($records, function ($a, $b) {
        return ((int)($b['failedAt'] ?? 0)) <=> ((int)($a['failedAt'] ?? 0));
    });
    return $records;
}

function coupon_clear_failed_records() {
    foreach (glob(coupon_storage_dir() . '/failed_*.json') ?: [] as $file) {
        @unlink($file);
    }
}

function coupon_clear_session_records() {
    foreach (glob(coupon_storage_dir() . '/session_*.json') ?: [] as $file) {
        @unlink($file);
    }
}
