<?php
/*
 * 账号共享接口（配合 zh.html 使用）
 *
 * 存储：同目录 accounts_share_data.php（纯 JSON 文件，无需数据库）
 *      文件头带 PHP 退出守卫（见下方 $DATA_GUARD），直接用浏览器访问该文件不会泄露内容。
 * 鉴权：无（不需要密码）
 *
 * 接口：
 *   GET  ?act=ping                    连通性检测
 *   GET  ?act=list                    返回全部账号
 *   GET  ?act=clear                   清空全部账号
 *   POST {act:"list"}
 *   POST {act:"sync",  accounts:[...], removed:[...]}
 *                                     合并同步（按 updatedAt 较新者胜出），返回合并后的完整列表
 *   POST {act:"delete", keys:["u:123"]}
 *   POST {act:"clear"}
 *
 * 统一返回：{"ok":bool,"msg":string,"data":[...],"rev":int,"updated_at":string}
 */

declare(strict_types=1);

$DATA_FILE      = __DIR__ . '/accounts_share_data.php';
$DATA_GUARD     = "<?php exit; ?>\n";
$MAX_ACCOUNTS   = 5000;
$MAX_BODY       = 64 * 1024 * 1024;

// ---------- CORS（zh.html 从 file:// 或其他域名打开时需要） ----------
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');
header('Cache-Control: no-store');
if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') {
    http_response_code(204);
    exit;
}

function jsonResp(array $payload): void {
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function atomicWrite(string $file, string $contents): bool {
    $tmp = $file . '.' . bin2hex(random_bytes(4)) . '.tmp';
    if (@file_put_contents($tmp, $contents, LOCK_EX) === false) return false;
    if (!@rename($tmp, $file)) {
        @unlink($tmp);
        return false;
    }
    return true;
}

function emptyStore(): array {
    return ['accounts' => [], 'deleted' => [], 'rev' => 0, 'updated_at' => ''];
}

function readStore(string $file): array {
    if (!is_file($file)) return emptyStore();
    $raw = (string)@file_get_contents($file);
    // 跳过文件头的退出守卫（注释里不能写结束标签，否则会真的把 PHP 模式关掉）
    $pos = strpos($raw, '{');
    if ($pos === false) return emptyStore();
    $data = json_decode(substr($raw, $pos), true);
    if (!is_array($data)) return emptyStore();
    return [
        'accounts'   => is_array($data['accounts'] ?? null) ? $data['accounts'] : [],
        'deleted'    => is_array($data['deleted'] ?? null) ? $data['deleted'] : [],
        'rev'        => (int)($data['rev'] ?? 0),
        'updated_at' => (string)($data['updated_at'] ?? ''),
    ];
}

function saveStore(string $file, array &$store, string $guard): bool {
    $store['rev'] = (int)($store['rev'] ?? 0) + 1;
    $store['updated_at'] = date('Y-m-d H:i:s');
    return atomicWrite($file, $guard . json_encode($store, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
}

/** 合并键：优先用 uid（c_user），否则用客户端生成的 id */
function mergeKey(array $acc): string {
    $uid = trim((string)($acc['uid'] ?? ''));
    if ($uid !== '') return 'u:' . $uid;
    $id = trim((string)($acc['id'] ?? ''));
    return $id !== '' ? 'i:' . $id : '';
}

function cleanAccount(array $acc, int $now): ?array {
    if (mergeKey($acc) === '') return null;
    $cookies = $acc['cookies'] ?? null;
    if (!is_array($cookies) || !$cookies) return null;
    $ts = (int)($acc['updatedAt'] ?? 0);
    if ($ts <= 0) $ts = $now;
    $usedAt = (int)($acc['usedAt'] ?? 0);
    return [
        'id'        => (string)($acc['id'] ?? ''),
        'uid'       => trim((string)($acc['uid'] ?? '')),
        'cookies'   => $cookies,
        'ua'        => (string)($acc['ua'] ?? ''),
        'region'    => (string)($acc['region'] ?? ''),
        'usedAt'    => $usedAt > 0 ? $usedAt : null,
        'updatedAt' => $ts,
    ];
}

/** 读取为 uid/id => account 的映射 */
function indexAccounts(array $accounts): array {
    $out = [];
    foreach ($accounts as $acc) {
        if (!is_array($acc)) continue;
        $k = mergeKey($acc);
        if ($k !== '') $out[$k] = $acc;
    }
    return $out;
}

function respondWithStore(array $store): void {
    jsonResp([
        'ok'         => true,
        'data'       => array_values($store['accounts']),
        'rev'        => (int)$store['rev'],
        'updated_at' => (string)$store['updated_at'],
    ]);
}

// ================= GET =================
$method = (string)($_SERVER['REQUEST_METHOD'] ?? 'GET');

if ($method === 'GET') {
    $act = (string)($_GET['act'] ?? '');

    if ($act === 'ping') {
        jsonResp(['ok' => true, 'msg' => 'ok', 'time' => date('Y-m-d H:i:s')]);
    }
    if ($act === '') {
        jsonResp(['ok' => false, 'msg' => '缺少 act 参数']);
    }

    if ($act === 'list') {
        respondWithStore(readStore($DATA_FILE));
    }

    if ($act === 'clear') {
        $nowMs = (int)round(microtime(true) * 1000);
        $store = emptyStore();
        foreach (indexAccounts(readStore($DATA_FILE)['accounts']) as $k => $_) {
            $store['deleted'][$k] = $nowMs; // 留墓碑，避免其他客户端把旧数据同步回来
        }
        saveStore($DATA_FILE, $store, $DATA_GUARD);
        respondWithStore($store);
    }

    jsonResp(['ok' => false, 'msg' => '未知操作: ' . $act]);
}

// ================= POST =================
$raw = file_get_contents('php://input');
if (is_string($raw) && strlen($raw) > $MAX_BODY) {
    jsonResp(['ok' => false, 'msg' => '请求体过大']);
}
$input = json_decode((string)$raw, true);
if (!is_array($input)) $input = $_POST;

$act = (string)($input['act'] ?? '');
$nowMs = (int)round(microtime(true) * 1000); // 必须和前端 Date.now() 一样是毫秒，墓碑比较才正确

if ($act === 'list') {
    respondWithStore(readStore($DATA_FILE));
}

if ($act === 'sync') {
    $store = readStore($DATA_FILE);
    $byKey = indexAccounts($store['accounts']);
    $deleted = $store['deleted'];

    // 1) 先落地客户端的删除（记墓碑）
    foreach ((array)($input['removed'] ?? []) as $k) {
        $k = trim((string)$k);
        if ($k === '') continue;
        unset($byKey[$k]);
        $deleted[$k] = $nowMs;
    }

    // 2) 合并上传的账号，updatedAt 较新者胜出
    $incoming = (array)($input['accounts'] ?? []);
    if (count($incoming) > $MAX_ACCOUNTS) $incoming = array_slice($incoming, 0, $MAX_ACCOUNTS);
    foreach ($incoming as $acc) {
        if (!is_array($acc)) continue;
        $clean = cleanAccount($acc, $nowMs);
        if ($clean === null) continue;
        $k = mergeKey($clean);
        $ts = (int)$clean['updatedAt'];
        if (isset($deleted[$k]) && (int)$deleted[$k] >= $ts) continue; // 已被删除且没有更新
        $exist = $byKey[$k] ?? null;
        if ($exist === null || (int)($exist['updatedAt'] ?? 0) <= $ts) {
            $byKey[$k] = $clean;
        }
    }

    $store['accounts'] = array_values($byKey);
    $store['deleted'] = $deleted;
    if (!saveStore($DATA_FILE, $store, $DATA_GUARD)) {
        jsonResp(['ok' => false, 'msg' => '写入失败，请检查目录权限']);
    }
    respondWithStore($store);
}

if ($act === 'delete') {
    $store = readStore($DATA_FILE);
    $byKey = indexAccounts($store['accounts']);
    foreach ((array)($input['keys'] ?? []) as $k) {
        $k = trim((string)$k);
        if ($k === '') continue;
        unset($byKey[$k]);
        $store['deleted'][$k] = $nowMs;
    }
    $store['accounts'] = array_values($byKey);
    saveStore($DATA_FILE, $store, $DATA_GUARD);
    respondWithStore($store);
}

if ($act === 'clear') {
    $store = emptyStore();
    foreach (indexAccounts(readStore($DATA_FILE)['accounts']) as $k => $_) {
        $store['deleted'][$k] = $nowMs;
    }
    saveStore($DATA_FILE, $store, $DATA_GUARD);
    respondWithStore($store);
}

jsonResp(['ok' => false, 'msg' => '未知操作: ' . $act]);
