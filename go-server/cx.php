<?php

/* - cx.php?act=clear_cache
*/




$CX_OWNS_SESSION = (session_status() !== PHP_SESSION_ACTIVE);
session_start();
if (!defined('CX_DATA_ONLY') || CX_DATA_ONLY !== true) {
    header('Content-Type: text/html; charset=utf-8');
}

$API1_BASE = 'https://zb1.eqwofaygdsjko.uk:443/api';
$API2_BASE = 'https://zb2.eqwofaygdsjko.uk:443/api';
$API_BASE = $API1_BASE;
$USERNAME = 'poisu906';
$PASSWORD = 'dfd$d7641HETD2af';
$CACHE_TTL = 120;
$RETRY_BACKOFF = 120;    // 上游返空/异常后的退避秒数（空数据也必须退避）
$MIN_REFRESH_DELAY = 30; // 前端轮询间隔硬下限，避免退化成 5 秒重试风暴
$API1_ACCOUNTS_FILE = __DIR__ . '/api1_allowed_accounts.json';
$API2_ACCOUNTS_FILE = __DIR__ . '/api2_allowed_accounts.json';
$LOGO_PATH = '../../1777990327149137925.png';
// 数据命名空间键：更换管理员账号后保持不变，历史查询数据（cx_date_lastgood_* 等）继续命中。
// 值为旧账号 tpadsf532 的 md5 前 16 位，与磁盘上已有的历史缓存文件名保持一致。
$CX_DATA_KEY = '3bd0f92c13ed0c94';
// 账号切换日：此日期之前的历史数据只有旧管理员账号能查到，
// 因此旧日期仍用旧管理员账号 + 旧白名单查询，新日期用新账号 + 新白名单。
$CX_ACCOUNT_SWITCH_DATE = '2026-09-23';
$CX_LEGACY_USERNAME = 'poisu906';
$CX_LEGACY_PASSWORD = 'dfd$d7641HETD2af';
$CX_LEGACY_ACCOUNTS_FILE = __DIR__ . '/api1_allowed_accounts_legacy.json';

function cxNormalizeApiType($value): int {
    $value = strtolower(trim((string)$value));
    if (in_array($value, ['2', 'api2', 'interface2', 'zb2', 'nonlogin', 'feideng'], true)) return 2;
    return 1;
}

$apiType = cxNormalizeApiType($_GET['api_type'] ?? $_GET['api'] ?? $_GET['interface'] ?? $_GET['source'] ?? 1);

$CX_API_CONFIGS = [
    1 => [
        'api_type' => 1,
        'label' => '接口1',
        'base' => $API1_BASE,
        'username' => 'tpadsf532',
        'password' => 'a1qrga$JHFl52aP',
        'accounts_file' => $API1_ACCOUNTS_FILE,
    ],
    2 => [
        'api_type' => 2,
        'label' => '接口2',
        // 接口2 账号（tpadsf531）同样登录 zb1，上游 base 与接口1 一致
        'base' => $API1_BASE,
        'username' => 'tpadsf531',
        'password' => 'a1q$r6gw9Fl52aP',
        'accounts_file' => $API2_ACCOUNTS_FILE,
        'remark_whitelist' => ['阳1', '传媒1', '杰1', '果子1'],
    ],
];
$cxApiConfig = $CX_API_CONFIGS[$apiType] ?? $CX_API_CONFIGS[1];
$API_BASE = (string)$cxApiConfig['base'];
$USERNAME = (string)$cxApiConfig['username'];
$PASSWORD = (string)$cxApiConfig['password'];
$AUTHORIZED_ACCOUNTS_FILE = (string)$cxApiConfig['accounts_file'];
$apiLabel = (string)$cxApiConfig['label'];

$date = isset($_GET['date']) ? trim((string)$_GET['date']) : date('Y-m-d');
if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $date)) {
    $date = date('Y-m-d');
}

// 旧日期（切换日之前）用旧管理员账号 + 旧白名单查询，保证历史数据可查。
if ($apiType === 1 && $date < $CX_ACCOUNT_SWITCH_DATE) {
    $USERNAME = $CX_LEGACY_USERNAME;
    $PASSWORD = $CX_LEGACY_PASSWORD;
    $AUTHORIZED_ACCOUNTS_FILE = $CX_LEGACY_ACCOUNTS_FILE;
}

// 切换日：当天新旧两批子账号都可能有数据（子账号变了、备注没变），需要合并显示。
$isSwitchDay = ($apiType === 1 && $date === $CX_ACCOUNT_SWITCH_DATE);

$isAutoRefresh = (string)($_GET['auto'] ?? '') === '1';
$forceRefresh = (string)($_GET['force'] ?? '') === '1' || (string)($_GET['refresh'] ?? '') === '1';
$startTime = $date . ' 00:00:00';
$endTime = $date . ' 23:59:59';

function e($value): string {
    return htmlspecialchars((string)$value, ENT_QUOTES, 'UTF-8');
}

function num($value): string {
    return number_format((int)$value);
}

function rateText(int $part, int $total): string {
    if ($total <= 0) return '0%';
    return rtrim(rtrim(number_format($part / $total * 100, 1), '0'), '.') . '%';
}

function cxGetPricingTable(): array {
    return [
        1 => [
            '阳'   => 0.04,
            '胡逸'   => 0.04,
            '老妈'   => 0.048,
            '传媒' => 0.04,
            '杰' => 0.05,
            '阳1' => 0.04,
            '传媒1'  => 0.04,
            'C2'  => 0.05,
            'C3'  => 0.05,
            'C4'  => 0.05,
            'C5'  => 0.05,
            'C'   => 0.05,
        ],
        2 => [
            '阳1'   => 0.04,
            '传媒1' => 0.04,
            '杰1' => 0.05,
            '果子1' => 0.048,
            '自己' => 0.048,
            '毛毛' => 0.04,
            'C-C5' => 0.04,
            'C1'  => 0.04,
            'C2'  => 0.04,
            'C3'  => 0.04,
            'C4'  => 0.04,
            'C5'  => 0.04,
            'C'   => 0.04,
        ],
    ];
}

function cxGetPricePerSuccess(int $apiType, string $remark): ?float {
    $table = cxGetPricingTable();
    $pricing = $table[$apiType] ?? [];
    $remark = trim($remark);
    if ($remark === '') return null;
    if (isset($pricing[$remark])) return $pricing[$remark];
    // 按 key 长度降序排列，让 "传媒"、"C-C5" 等长 key 优先于 "C" 等短 key，避免误匹配
    $keys = array_keys($pricing);
    usort($keys, fn($a, $b) => strlen($b) - strlen($a));
    foreach ($keys as $key) {
        if (stripos($remark, $key) !== false) return $pricing[$key];
    }
    return null;
}

function cxFormatMoney(float $amount, int $decimals = 2): string {
    return '¥' . number_format($amount, $decimals);
}

function cxPageUrl(array $params = []): string {
    $query = array_merge($_GET, $params);
    foreach ($query as $key => $value) {
        if ($value === null || $value === '') unset($query[$key]);
    }
    unset($query['_ts'], $query['auto']);
    return '?' . http_build_query($query);
}

function cxCacheDir(): string {
    $dir = __DIR__ . '/cx_cache';
    if (!is_dir($dir)) {
        @mkdir($dir, 0755, true);
    }
    return $dir;
}

function cxCacheAccountKey(): string {
    global $USERNAME;
    return substr(md5(trim((string)$USERNAME)), 0, 16);
}

// 数据命名空间键：与登录账号解耦。更换管理员账号后保持数据缓存键不变，
// 这样历史数据（cx_date_lastgood_* / cx_cache_* / cx_lastgood_*）不会丢失。
function cxCacheDataKey(): string {
    global $CX_DATA_KEY;
    return (string)$CX_DATA_KEY;
}

function cxClearCache(): array {
    $dir = cxCacheDir();
    $deleted = 0;
    // 注意：不删 cx_date_lastgood_* / cx_switchold_*，它们是按日期保存的历史归档，清缓存不应丢历史数据
    $patterns = [
        'cx_cache_*.json',
        'cx_lastgood_*.json',
        'cx_cooldown_*.json',
        'cx_api_token_cache*.json',
        'cx_authorized_accounts_lastgood_*.json',
        'cx_authorized_accounts_lastgood.json',
    ];
    foreach ($patterns as $pattern) {
        foreach (glob($dir . '/' . $pattern) ?: [] as $file) {
            if (is_file($file) && @unlink($file)) $deleted++;
        }
    }
    if (session_status() === PHP_SESSION_ACTIVE) {
        unset($_SESSION['cx_token'], $_SESSION['cx_token_ts']);
        unset($_SESSION['cx_token_' . cxCacheAccountKey()], $_SESSION['cx_token_ts_' . cxCacheAccountKey()]);
        foreach (array_keys($_SESSION) as $sessionKey) {
            if (strpos((string)$sessionKey, 'cx_token_') === 0 || strpos((string)$sessionKey, 'cx_token_ts_') === 0) {
                unset($_SESSION[$sessionKey]);
            }
        }
    }
    return ['ok' => true, 'deleted' => $deleted];
}

if ((string)($_GET['act'] ?? '') === 'clear_cache') {
    $result = cxClearCache();
    if ((string)($_GET['json'] ?? '') === '1' || (strtolower((string)($_SERVER['HTTP_X_REQUESTED_WITH'] ?? '')) === 'fetch')) {
        header('Content-Type: application/json; charset=utf-8');
        echo json_encode($result, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        exit;
    }
    header('Content-Type: text/html; charset=utf-8');
    echo '<!doctype html><meta charset="utf-8"><title>Ajie 缓存清理</title><body style="font-family:Arial,Microsoft YaHei,sans-serif;padding:32px"><h2>缓存已清理</h2><p>删除文件：' . (int)$result['deleted'] . ' 个</p><p><a href="' . e(basename((string)($_SERVER['SCRIPT_NAME'] ?? 'cx.php'))) . '?force=1">立即用新账号刷新数据</a></p></body>';
    exit;
}

function httpPost(string $url, array $data, array $headers = []): ?array {
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_TIMEOUT => 15,
        CURLOPT_CONNECTTIMEOUT => 8,
        CURLOPT_HTTPHEADER => array_merge(['Content-Type: application/json', 'Accept: application/json'], $headers),
    ]);
    $res = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);
    if ($err || $res === false) return null;
    $json = json_decode((string)$res, true);
    return is_array($json) ? $json : null;
}

function httpGetJson(string $url, array $headers = [], int $timeout = 15, int $connectTimeout = 8): ?array {
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_TIMEOUT => $timeout,
        CURLOPT_CONNECTTIMEOUT => $connectTimeout,
        CURLOPT_HTTPHEADER => array_merge(['Accept: application/json'], $headers),
    ]);
    $res = curl_exec($ch);
    $err = curl_error($ch);
    curl_close($ch);
    if ($err || $res === false) return null;
    $json = json_decode((string)$res, true);
    return is_array($json) ? $json : null;
}

// 原子写：先写临时文件再 rename，读者永远读不到“写了一半”的 JSON（手动刷新瞬间回退旧缓存的元凶）
function cxAtomicWrite(string $file, string $content): void {
    $tmp = $file . '.tmp.' . getmypid() . '.' . str_replace('.', '', uniqid('', true));
    if (@file_put_contents($tmp, $content, LOCK_EX) === false) {
        @unlink($tmp);
        return;
    }
    if (!@rename($tmp, $file)) {
        @unlink($tmp);
        @file_put_contents($file, $content, LOCK_EX);
    }
}

// 读 JSON 文件并带短重试：万一读到中间态（半截 JSON），稍等重读
function cxReadJsonFile(string $file): ?array {
    for ($i = 0; $i < 3; $i++) {
        if (!is_file($file)) return null;
        $raw = @file_get_contents($file);
        if ($raw === false) return null;
        $data = json_decode($raw, true);
        if (is_array($data)) return $data;
        usleep(50000);
    }
    return null;
}

function cacheFile(string $key): string {
    return cxCacheDir() . '/cx_cache_' . md5($key) . '.json';
}

function cacheGet(string $key, int $ttl): ?array {
    $file = cacheFile($key);
    if (!is_file($file)) return null;
    $data = cxReadJsonFile($file);
    if (!is_array($data) || (time() - (int)($data['_ts'] ?? 0)) > $ttl) return null;
    return is_array($data['payload'] ?? null) ? $data['payload'] : null;
}

function cachePeek(string $key): ?array {
    $file = cacheFile($key);
    if (!is_file($file)) return null;
    $data = cxReadJsonFile($file);
    if (!is_array($data)) return null;
    return is_array($data['payload'] ?? null) ? $data['payload'] : null;
}

function cacheSet(string $key, array $payload): void {
    cxAtomicWrite(
        cacheFile($key),
        json_encode(['_ts' => time(), 'payload' => $payload], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

function lastGoodFile(string $key): string {
    return cxCacheDir() . '/cx_lastgood_' . md5($key) . '.json';
}

function lastGoodGet(string $key): ?array {
    $file = lastGoodFile($key);
    if (!is_file($file)) return null;
    $data = cxReadJsonFile($file);
    if (!is_array($data)) return null;
    return is_array($data['payload'] ?? null) ? $data['payload'] : null;
}

function lastGoodSet(string $key, array $payload): void {
    cxAtomicWrite(
        lastGoodFile($key),
        json_encode(['_ts' => time(), 'payload' => $payload], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

function dateLastGoodFile(string $date): string {
    global $apiType;
    $apiSuffix = $apiType === 2 ? '_api2' : '';
    return cxCacheDir() . '/cx_date_lastgood_' . cxCacheDataKey() . $apiSuffix . '_' . md5($date) . '.json';
}

function dateLastGoodGet(string $date): ?array {
    $file = dateLastGoodFile($date);
    if (!is_file($file)) return null;
    $data = cxReadJsonFile($file);
    if (!is_array($data)) return null;
    return is_array($data['payload'] ?? null) ? $data['payload'] : null;
}

function dateLastGoodSet(string $date, array $payload): void {
    cxAtomicWrite(
        dateLastGoodFile($date),
        json_encode(['_ts' => time(), 'payload' => $payload], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

// ===== 切换日：冻结“切换前旧账号”的今日快照 =====
function cxSwitchOldFile(string $date): string {
    global $apiType;
    $apiSuffix = $apiType === 2 ? '_api2' : '';
    return cxCacheDir() . '/cx_switchold_' . cxCacheDataKey() . $apiSuffix . '_' . md5($date) . '.json';
}

function cxSwitchOldGet(string $date): ?array {
    $file = cxSwitchOldFile($date);
    if (!is_file($file)) return null;
    $data = cxReadJsonFile($file);
    if (!is_array($data)) return null;
    return is_array($data['payload'] ?? null) ? $data['payload'] : null;
}

function cxSwitchOldSet(string $date, array $payload): void {
    cxAtomicWrite(
        cxSwitchOldFile($date),
        json_encode(['_ts' => time(), 'payload' => $payload], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

// 全量聚合求和：同一 username（如 0000-所有）的计数相加并重算比率。
// 切换日两批子账号不同、无重叠，求和才是正确口径。
function cxSumItems(array $a, array $b): array {
    if (empty($a)) return $b;
    if (empty($b)) return $a;
    $map = [];
    foreach (array_merge($a, $b) as $row) {
        if (!is_array($row)) continue;
        $u = (string)($row['username'] ?? '');
        if ($u === '') continue;
        if (!isset($map[$u])) { $map[$u] = $row; continue; }
        foreach (['takeCount', 'completeCount', 'checkSuccessCount', 'checkFailCount'] as $k) {
            $map[$u][$k] = (int)($map[$u][$k] ?? 0) + (int)($row[$k] ?? 0);
        }
        $t = (int)($map[$u]['takeCount'] ?? 0);
        $c = (int)($map[$u]['completeCount'] ?? 0);
        $s = (int)($map[$u]['checkSuccessCount'] ?? 0);
        $map[$u]['completeRate'] = rateText($c, max(1, $t));
        $map[$u]['checkSuccessRate'] = rateText($s, max(1, $c));
    }
    return array_values($map);
}

// 明细按 username 并集：旧账号行打上 _src=old，同名冲突以新账号为准。
function cxUnionBatchItems(array $fresh, array $extra): array {
    if (empty($extra)) return $fresh;
    $map = [];
    foreach ($extra as $row) {
        if (!is_array($row)) continue;
        $u = (string)($row['username'] ?? '');
        if ($u === '') continue;
        $row['_src'] = 'old';
        $map[$u] = $row;
    }
    foreach ($fresh as $row) {
        if (!is_array($row)) continue;
        $u = (string)($row['username'] ?? '');
        if ($u === '') continue;
        $map[$u] = $row;
    }
    return array_values($map);
}

function rowsHaveData(array $rows): bool {
    foreach ($rows as $row) {
        if (!is_array($row)) continue;
        if (activityTotal($row) > 0) return true;
    }
    return false;
}

function payloadSectionHasData(array $payload, string $key): bool {
    $rows = $payload[$key] ?? [];
    return is_array($rows) && rowsHaveData($rows);
}

function payloadHasData(array $payload): bool {
    foreach (['items', 'batchItems'] as $key) {
        if (payloadSectionHasData($payload, $key)) return true;
    }
    return false;
}

function firstPayloadWithSectionData(string $key, ?array ...$payloads): ?array {
    foreach ($payloads as $payload) {
        if (is_array($payload) && payloadSectionHasData($payload, $key)) {
            return $payload;
        }
    }
    return null;
}

// 把本次拉取里“为空的那一侧”用历史缓存补齐，保证持久化的 lastgood/date_lastgood
// 始终是完整的（items + batchItems），避免“全量已更新但白名单为空 → 不写 lastgood
// → 下次刷新回退到更旧的完整快照”。
function cxMergePayloadWithFallback(array $fresh, ?array $fallback): array {
    if (is_array($fallback)) {
        if (!rowsHaveData(is_array($fresh['items'] ?? null) ? $fresh['items'] : [])) {
            $fresh['items'] = is_array($fallback['items'] ?? null) ? $fallback['items'] : [];
        }
        if (!rowsHaveData(is_array($fresh['batchItems'] ?? null) ? $fresh['batchItems'] : [])) {
            $fresh['batchItems'] = is_array($fallback['batchItems'] ?? null) ? $fallback['batchItems'] : [];
        }
    }
    return $fresh;
}

// 计算 payload 里某个 section 的总活动量（take+complete+success+fail），用于单调性判断。
function cxSectionTotalActivity(array $payload, string $key): int {
    $rows = is_array($payload[$key] ?? null) ? $payload[$key] : [];
    $total = 0;
    foreach ($rows as $row) {
        if (is_array($row)) $total += activityTotal($row);
    }
    return $total;
}

// 判断两份 batchItems 是否属于同一组账号（username 集合一致）。
// 用于在“更换授权账号”后避免把旧账号的数据拼回新白名单。
function cxSameUsernameSet(array $a, array $b): bool {
    $setA = [];
    foreach ($a as $row) {
        if (is_array($row)) {
            $name = trim((string)($row['username'] ?? ''));
            if ($name !== '') $setA[$name] = true;
        }
    }
    $setB = [];
    foreach ($b as $row) {
        if (is_array($row)) {
            $name = trim((string)($row['username'] ?? ''));
            if ($name !== '') $setB[$name] = true;
        }
    }
    return !empty($setA) && $setA == $setB;
}

// 按 section 单调合并：全量(items)与白名单(batchItems)各自“只涨不跌”。
// 若本次某侧比历史少，说明该侧是限流下的部分返回，保留历史更完整的那一侧。
// 白名单部分仅在“账号集合相同”时合并：更换授权账号后账号集合不同，必须用本次的新结果。
function cxMergePayloadMonotonic(array $fresh, ?array $best): array {
    if (is_array($best)) {
        if (cxSectionTotalActivity($fresh, 'items') < cxSectionTotalActivity($best, 'items')) {
            $fresh['items'] = is_array($best['items'] ?? null) ? $best['items'] : [];
        }
        $freshBatch = is_array($fresh['batchItems'] ?? null) ? $fresh['batchItems'] : [];
        $bestBatch = is_array($best['batchItems'] ?? null) ? $best['batchItems'] : [];
        if (cxSameUsernameSet($freshBatch, $bestBatch)
            && cxSectionTotalActivity($fresh, 'batchItems') < cxSectionTotalActivity($best, 'batchItems')) {
            $fresh['batchItems'] = $bestBatch;
        }
    }
    return $fresh;
}

function fetchCooldownFile(string $key): string {
    return cxCacheDir() . '/cx_cooldown_' . md5($key) . '.json';
}

function fetchCooldownRemaining(string $key): int {
    $f = fetchCooldownFile($key);
    if (!is_file($f)) return 0;
    $d = cxReadJsonFile($f);
    if (!is_array($d)) return 0;
    return max(0, (int)($d['until'] ?? 0) - time());
}

function fetchCooldownState(string $key): array {
    $f = fetchCooldownFile($key);
    if (!is_file($f)) return [];
    $d = cxReadJsonFile($f);
    return is_array($d) ? $d : [];
}

function setFetchCooldown(string $key, int $seconds, string $reason = '', bool $isRateLimit = false): int {
    $attempts = 1;
    if ($isRateLimit) {
        $prev = fetchCooldownState($key);
        if (!empty($prev['is_rate_limit']) && (int)($prev['until'] ?? 0) > time()) {
            $attempts = (int)($prev['attempts'] ?? 1) + 1;
        }
        // 指数退避：180s * 2^(n-1)，封顶 3600s（1小时），避免每小时额度刚恢复就被立刻打爆
        $seconds = (int)min(3600, 180 * pow(2, $attempts - 1));
    }
    cxAtomicWrite(
        fetchCooldownFile($key),
        json_encode([
            'until' => time() + $seconds,
            'reason' => $reason,
            'attempts' => $attempts,
            'is_rate_limit' => $isRateLimit,
        ], JSON_UNESCAPED_UNICODE)
    );
    return $seconds;
}

// 冷却期间也把「上次为什么失败」显示出来，方便定位是被限流还是别的问题
function fetchCooldownReason(string $key): string {
    $f = fetchCooldownFile($key);
    if (!is_file($f)) return '';
    $d = cxReadJsonFile($f);
    return is_array($d) ? trim((string)($d['reason'] ?? '')) : '';
}

function clearFetchCooldown(string $key): void {
    $f = fetchCooldownFile($key);
    if (is_file($f)) @unlink($f);
}

function cxIsRateLimit(string $msg): bool {
    $msg = strtolower(trim($msg));
    if ($msg === '') return false;
    return strpos($msg, '429') !== false
        || strpos($msg, 'rate limit') !== false
        || strpos($msg, '限流') !== false
        || strpos($msg, '频繁') !== false
        || strpos($msg, '请求过多') !== false;
}

// 单飞锁：同一账号同一时刻只允许一个进程真正拉上游，其余进程读缓存，
// 避免 cron + 页面自动刷新 + 设备触发同时打上游形成限流风暴。
function cxFetchLockFile(): string {
    return cxCacheDir() . '/cx_fetch_' . cxCacheAccountKey() . '.lock';
}

function cxFetchLockAcquire() {
    $file = cxFetchLockFile();
    $handle = @fopen($file, 'c');
    if ($handle === false) return null;
    if (!@flock($handle, LOCK_EX | LOCK_NB)) {
        @fclose($handle);
        return null;
    }
    return $handle;
}

function cxFetchLockRelease($handle): void {
    if (is_resource($handle)) {
        @flock($handle, LOCK_UN);
        @fclose($handle);
    }
}

function cxTokenCacheFile(): string {
    return cxCacheDir() . '/cx_api_token_cache_' . cxCacheAccountKey() . '.json';
}

function cxSessionTokenKey(): string {
    return 'cx_token_' . cxCacheAccountKey();
}

function cxSessionTokenTsKey(): string {
    return 'cx_token_ts_' . cxCacheAccountKey();
}

function cxCachedTokenGet(): string {
    $file = cxTokenCacheFile();
    if (!is_file($file)) return '';
    $data = cxReadJsonFile($file);
    if (!is_array($data)) return '';
    $token = (string)($data['token'] ?? '');
    $ts = (int)($data['ts'] ?? 0);
    if ($token === '' || (time() - $ts) >= 3500) return '';
    return $token;
}

function cxCachedTokenSet(string $token): void {
    if ($token === '') return;
    cxAtomicWrite(
        cxTokenCacheFile(),
        json_encode(['token' => $token, 'ts' => time()], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

function cxCachedTokenClear(): void {
    $file = cxTokenCacheFile();
    if (is_file($file)) @unlink($file);
    if (session_status() === PHP_SESSION_ACTIVE) {
        $_SESSION[cxSessionTokenKey()] = '';
        $_SESSION[cxSessionTokenTsKey()] = 0;
        $_SESSION['cx_token'] = '';
        $_SESSION['cx_token_ts'] = 0;
    }
}

function getToken(): string {
    global $API_BASE, $USERNAME, $PASSWORD;
    $sessionTokenKey = cxSessionTokenKey();
    $sessionTokenTsKey = cxSessionTokenTsKey();
    if (!empty($_SESSION[$sessionTokenKey]) && (time() - (int)($_SESSION[$sessionTokenTsKey] ?? 0)) < 3500) {
        return (string)$_SESSION[$sessionTokenKey];
    }
    $cachedToken = cxCachedTokenGet();
    if ($cachedToken !== '') {
        if (session_status() === PHP_SESSION_ACTIVE) {
            $_SESSION[$sessionTokenKey] = $cachedToken;
            $_SESSION[$sessionTokenTsKey] = time();
        }
        return $cachedToken;
    }
    $loginRes = httpPost($API_BASE . '/user/login', [
        'username' => $USERNAME,
        'password' => $PASSWORD,
    ]);
    $token = '';
    if ($loginRes && (($loginRes['code'] ?? '') === '200' || (int)($loginRes['code'] ?? 0) === 200)) {
        $token = (string)($loginRes['token'] ?? $loginRes['data']['token'] ?? $loginRes['data']['accessToken'] ?? '');
    }
    if ($token !== '') {
        $_SESSION[$sessionTokenKey] = $token;
        $_SESSION[$sessionTokenTsKey] = time();
        cxCachedTokenSet($token);
    }
    return $token !== '' ? $token : (string)($loginRes['msg'] ?? '登录失败');
}

function readAuthorizedJuxingAccounts(string $file): array {
    if (!is_file($file)) return [];
    $data = json_decode((string)@file_get_contents($file), true);
    if (!is_array($data)) return [];
    $accounts = $data['accounts'] ?? $data;
    if (!is_array($accounts)) return [];

    $result = [];
    foreach ($accounts as $account) {
        if (is_array($account)) {
            $username = trim((string)($account['username'] ?? $account['name'] ?? ''));
            $remark = trim((string)($account['remark'] ?? ''));
        } else {
            $username = trim((string)$account);
            $remark = '';
        }
        if ($username !== '') $result[$username] = $remark;
    }
    ksort($result);
    return $result;
}

function authorizedAccountsLastGoodFile(): string {
    return cxCacheDir() . '/cx_authorized_accounts_lastgood_' . cxCacheAccountKey() . '.json';
}

function authorizedAccountsLastGoodGet(): array {
    $file = authorizedAccountsLastGoodFile();
    if (!is_file($file)) return [];
    $data = json_decode((string)@file_get_contents($file), true);
    if (!is_array($data)) return [];
    $accounts = $data['accounts'] ?? [];
    if (!is_array($accounts)) return [];

    $result = [];
    foreach ($accounts as $username => $remark) {
        if (is_array($remark)) {
            $name = trim((string)($remark['username'] ?? $remark['name'] ?? $username));
            $label = trim((string)($remark['remark'] ?? $remark['nickname'] ?? $remark['label'] ?? ''));
        } else {
            $name = trim((string)$username);
            $label = trim((string)$remark);
        }
        if ($name !== '') $result[$name] = $label;
    }
    ksort($result);
    return $result;
}

function authorizedAccountsLastGoodSet(array $accounts): void {
    if (empty($accounts)) return;
    cxAtomicWrite(
        authorizedAccountsLastGoodFile(),
        json_encode([
            '_ts' => time(),
            'accounts' => $accounts,
        ], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)
    );
}

function fetchSettleView(array $userList, string $startTime, string $endTime, array $authHeaders): array {
    global $API_BASE;
    $res = httpPost($API_BASE . '/user/settle/view', [
        'pageNum' => 1,
        'pageSize' => max(100, count($userList) + 20),
        'userList' => $userList,
        'startTime' => $startTime,
        'endTime' => $endTime,
    ], $authHeaders);
    if (!$res) return ['ok' => false, 'msg' => '查询接口无响应', 'items' => []];
    if (($res['code'] ?? '') === '200' || (int)($res['code'] ?? 0) === 200) {
        return ['ok' => true, 'msg' => '', 'items' => $res['data']['items'] ?? []];
    }
    return ['ok' => false, 'msg' => (string)($res['msg'] ?? '查询失败'), 'items' => [], 'code' => $res['code'] ?? null];
}

function filterAuthorizedRowsFromItems(array $items, array $authorizedAccounts): array {
    if (empty($items) || empty($authorizedAccounts)) return [];

    $filtered = [];
    foreach ($items as $row) {
        if (!is_array($row)) continue;
        $username = trim((string)($row['username'] ?? ''));
        if ($username === '' || !array_key_exists($username, $authorizedAccounts)) {
            continue;
        }

        $take = (int)($row['takeCount'] ?? 0);
        $complete = (int)($row['completeCount'] ?? 0);
        $success = (int)($row['checkSuccessCount'] ?? 0);
        $fail = (int)($row['checkFailCount'] ?? 0);

        if (($take + $complete + $success + $fail) <= 0) {
            continue;
        }

        $row['_remark'] = $authorizedAccounts[$username] ?? '';
        $filtered[$username] = $row;
    }

    return array_values($filtered);
}

function resolveDecryptProxyUrl(): ?string {
    $host = trim((string)($_SERVER['HTTP_HOST'] ?? ''));
    if ($host === '') return null;

    $https = (string)($_SERVER['HTTPS'] ?? '');
    $forwardedProto = strtolower((string)($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? ''));
    $scheme = ($forwardedProto === 'https' || ($https !== '' && $https !== 'off') || (string)($_SERVER['SERVER_PORT'] ?? '') === '443')
        ? 'https'
        : 'http';

    $scriptDir = str_replace('\\', '/', dirname((string)($_SERVER['SCRIPT_NAME'] ?? '')));
    $scriptDir = rtrim($scriptDir, '/.');
    return $scheme . '://' . $host . ($scriptDir !== '' ? $scriptDir : '') . '/decrypt_proxy.php';
}

function fetchAuthorizedAccountsFromProxy(): array {
    global $apiType;
    $proxyUrl = resolveDecryptProxyUrl();
    if ($proxyUrl === null) return [];

    $act = $apiType === 2 ? 'api2_accounts' : 'api1_accounts';
    $json = httpGetJson($proxyUrl . '?act=' . rawurlencode($act) . '&_ts=' . rawurlencode((string)time()), [], 4, 2);
    if (!is_array($json) || empty($json['ok']) || !is_array($json['data'] ?? null)) {
        return [];
    }

    $result = [];
    foreach ($json['data'] as $account) {
        if (!is_array($account)) continue;
        $username = trim((string)($account['username'] ?? $account['name'] ?? ''));
        $remark = trim((string)($account['remark'] ?? ''));
        if ($username !== '') $result[$username] = $remark;
    }
    ksort($result);
    return $result;
}

function loadAuthorizedAccounts(string $file): array {
    $accounts = readAuthorizedJuxingAccounts($file);
    if (!empty($accounts)) {
        authorizedAccountsLastGoodSet($accounts);
        return $accounts;
    }

    $accounts = fetchAuthorizedAccountsFromProxy();
    if (!empty($accounts)) {
        authorizedAccountsLastGoodSet($accounts);
        return $accounts;
    }

    return authorizedAccountsLastGoodGet();
}

function activityTotal(array $row): int {
    return
        (int)($row['takeCount'] ?? 0) +
        (int)($row['completeCount'] ?? 0) +
        (int)($row['checkSuccessCount'] ?? 0) +
        (int)($row['checkFailCount'] ?? 0);
}

function normalizeAuthorizedRow(array $row, array $authorizedAccounts): ?array {
    $username = trim((string)($row['username'] ?? ''));
    if ($username === '' || !array_key_exists($username, $authorizedAccounts)) {
        return null;
    }
    if (activityTotal($row) <= 0) {
        return null;
    }

    $row['_remark'] = $authorizedAccounts[$username] ?? '';
    return $row;
}

function fetchAuthorizedSettleRows(array $authorizedAccounts, string $startTime, string $endTime, array $authHeaders, array $fallbackItems = []): array {
    if (empty($authorizedAccounts)) {
        return ['ok' => true, 'msg' => '', 'items' => []];
    }

    $rowsByUsername = [];
    $userList = array_keys($authorizedAccounts);

    $view = fetchSettleView($userList, $startTime, $endTime, $authHeaders);
    // 白名单明细查询是每轮刷新的第 2 个上游请求，最容易撞限流；
    // 限流时短退避重试一次，提高“备注分组绩效 / 授权账号明细”的返回率。
    if (!$view['ok'] && cxIsRateLimit((string)($view['msg'] ?? ''))) {
        usleep(2500000); // 2.5 秒
        $view = fetchSettleView($userList, $startTime, $endTime, $authHeaders);
    }
    if (!$view['ok']) {
        return $view;
    }
    foreach (($view['items'] ?? []) as $row) {
        if (!is_array($row)) continue;
        $normalized = normalizeAuthorizedRow($row, $authorizedAccounts);
        if ($normalized === null) continue;
        $rowsByUsername[(string)$normalized['username']] = $normalized;
    }

    foreach ($fallbackItems as $row) {
        if (!is_array($row)) continue;
        $normalized = normalizeAuthorizedRow($row, $authorizedAccounts);
        if ($normalized === null) continue;
        $rowsByUsername[(string)$normalized['username']] = $normalized;
    }

    return ['ok' => true, 'msg' => '', 'items' => array_values($rowsByUsername)];
}

// 尽早释放 session 锁：仅当本文件自己开启 session（独立访问）时关闭，
// 被 decrypt_proxy.php include 时不关闭，避免影响宿主页面的 session。
if ($CX_OWNS_SESSION && session_status() === PHP_SESSION_ACTIVE) {
    session_write_close();
}

$authorizedAccounts = loadAuthorizedAccounts($AUTHORIZED_ACCOUNTS_FILE);
// 只查询指定备注的账号（remark_whitelist 白名单过滤）
if (!empty($cxApiConfig['remark_whitelist'])) {
    $authorizedAccounts = array_filter($authorizedAccounts, function ($remark) use ($cxApiConfig) {
        return in_array(trim((string)$remark), $cxApiConfig['remark_whitelist'], true);
    });
}
$authorizedError = empty($authorizedAccounts) ? '未读取到聚星授权账号，请先在后台维护聚星账号白名单。' : '';

$items = [];
$batchItems = [];
$error = '';
$warning = '';
$fromCache = false;
$cacheAge = 0;
$cacheKey = 'cx_v2_' . cxCacheDataKey() . '_' . ($apiType === 2 ? '2' : '1') . '_' . $date . '_' . md5(json_encode($authorizedAccounts, JSON_UNESCAPED_UNICODE));
$cooldownLeft = fetchCooldownRemaining($cacheKey);
// 强制刷新也不能越过冷却：防止 force=1 被高频调用（例如插件控制页）把上游打到限流
if ($forceRefresh && $cooldownLeft > 0) {
    $forceRefresh = false;
}
$lastGood = lastGoodGet($cacheKey);
$dateLastGood = dateLastGoodGet($date);
$usingLastGood = false;
$staleCached = cachePeek($cacheKey);

// 切换日：冻结“切换前旧账号”的今日快照（只存一次，之后不再重复叠加）
if ($isSwitchDay && !is_array(cxSwitchOldGet($date))) {
    $oldCandidate = (is_array($dateLastGood) && empty($dateLastGood['switchCombined']) && payloadHasData($dateLastGood))
        ? $dateLastGood
        : (is_array($lastGood) && payloadHasData($lastGood) ? $lastGood : null);
    if (is_array($oldCandidate) && payloadHasData($oldCandidate)) {
        cxSwitchOldSet($date, $oldCandidate);
    }
}

$cached = $forceRefresh ? null : ($isAutoRefresh ? cacheGet($cacheKey, $CACHE_TTL) : cachePeek($cacheKey));
if ($cached && !payloadHasData($cached)) {
    $cached = null;
}

// 切换日：只要还没合并过，就不把旧账号缓存当成品数据直接展示，强制走“拉新 + 合并旧快照”
$switchCombined = $isSwitchDay && (
    (is_array($cached) && !empty($cached['switchCombined']))
    || (is_array($dateLastGood) && !empty($dateLastGood['switchCombined']))
);
if ($isSwitchDay && !$switchCombined) {
    $cached = null;
    $lastGood = null;
    $dateLastGood = null;
    $staleCached = null;
    $usingLastGood = false;
}
if (!$forceRefresh && !$cached) {
    if ($lastGood && payloadHasData($lastGood)) {
        $cached = $lastGood;
        $usingLastGood = true;
        cacheSet($cacheKey, [
            'items' => is_array($lastGood['items'] ?? null) ? $lastGood['items'] : [],
            'batchItems' => is_array($lastGood['batchItems'] ?? null) ? $lastGood['batchItems'] : [],
            'fetchTime' => (int)($lastGood['fetchTime'] ?? time()),
        ]);
    } elseif ($dateLastGood && payloadHasData($dateLastGood)) {
        $cached = $dateLastGood;
        $usingLastGood = true;
        cacheSet($cacheKey, [
            'items' => is_array($dateLastGood['items'] ?? null) ? $dateLastGood['items'] : [],
            'batchItems' => is_array($dateLastGood['batchItems'] ?? null) ? $dateLastGood['batchItems'] : [],
            'fetchTime' => (int)($dateLastGood['fetchTime'] ?? time()),
        ]);
        lastGoodSet($cacheKey, [
            'items' => is_array($dateLastGood['items'] ?? null) ? $dateLastGood['items'] : [],
            'batchItems' => is_array($dateLastGood['batchItems'] ?? null) ? $dateLastGood['batchItems'] : [],
            'fetchTime' => (int)($dateLastGood['fetchTime'] ?? time()),
        ]);
    } elseif ($staleCached && payloadHasData($staleCached)) {
        $cached = $staleCached;
        $usingLastGood = true;
    }
}
if (!$forceRefresh && $cached && !empty($authorizedAccounts) && !payloadSectionHasData($cached, 'batchItems')) {
    $batchFallbackPayload = firstPayloadWithSectionData('batchItems', $lastGood, $dateLastGood, $staleCached);
    if ($batchFallbackPayload) {
        $cached['batchItems'] = is_array($batchFallbackPayload['batchItems'] ?? null) ? $batchFallbackPayload['batchItems'] : [];
        $cached['fetchTime'] = (int)($batchFallbackPayload['fetchTime'] ?? ($cached['fetchTime'] ?? time()));
        $usingLastGood = true;
        cacheSet($cacheKey, [
            'items' => is_array($cached['items'] ?? null) ? $cached['items'] : [],
            'batchItems' => $cached['batchItems'],
            'fetchTime' => (int)($cached['fetchTime'] ?? time()),
        ]);
    }
}
if ($cached) {
    $items = $cached['items'] ?? [];
    $batchItems = $cached['batchItems'] ?? [];
    $fromCache = true;
    $cacheAge = time() - (int)($cached['fetchTime'] ?? time());
    if ($usingLastGood) {
        $warning = '缓存为空数据，已使用最近一次有数据的缓存（' . max(0, (int)$cacheAge) . ' 秒前）。';
    }
}

$shouldFetch = ($forceRefresh || !$cached || ($isAutoRefresh && $cacheAge >= $CACHE_TTL)) && $cooldownLeft <= 0;
$fetchLockHandle = null;
if ($shouldFetch) {
    $fetchLockHandle = cxFetchLockAcquire();
    if ($fetchLockHandle === null) {
        // 已有进程在拉上游：重读一遍缓存（可能刚被更新），本次不再重复打上游
        $freshCached = $isAutoRefresh ? cacheGet($cacheKey, $CACHE_TTL) : cachePeek($cacheKey);
        if (is_array($freshCached) && payloadHasData($freshCached)) {
            $cached = $freshCached;
            $items = is_array($cached['items'] ?? null) ? $cached['items'] : [];
            $batchItems = is_array($cached['batchItems'] ?? null) ? $cached['batchItems'] : [];
            $fromCache = true;
            $cacheAge = time() - (int)($cached['fetchTime'] ?? time());
        }
        if ($warning === '') $warning = '另一个刷新任务正在进行中，本次使用已有缓存数据。';
        $shouldFetch = false;
    }
}
if ($shouldFetch) {
    // 先占位写冷却：即使本请求被 nginx/fastcgi 掐断，冷却标记也已在盘上，
    // 避免下一个请求立刻重试形成风暴；成功拉取后会被 clearFetchCooldown 清除。
    setFetchCooldown($cacheKey, $RETRY_BACKOFF, '正在刷新');
    $token = getToken();
    if (session_status() === PHP_SESSION_ACTIVE) {
        session_write_close();
    }
    $fetchError = '';
    if (strlen($token) < 20) {
        $fetchError = $token;
    } else {
        $authHeaders = ['Authorization: Bearer ' . $token, 'token: ' . $token];

        $view = fetchSettleView([], $startTime, $endTime, $authHeaders);
        if ($view['ok']) {
            $items = is_array($view['items']) ? $view['items'] : [];
        } else {
            $viewCode = (string)($view['code'] ?? '');
            if ($viewCode === '401') {
                cxCachedTokenClear();
                $fetchError = '登录已过期，请等待自动更新或稍后再试';
            } else {
                $fetchError = (string)($view['msg'] ?? '查询失败');
                if ($viewCode !== '' && $viewCode !== '200') {
                    $fetchError = '[' . $viewCode . '] ' . $fetchError;
                }
            }
        }

        if ($fetchError === '' && !empty($authorizedAccounts)) {
            $authRows = fetchAuthorizedSettleRows($authorizedAccounts, $startTime, $endTime, $authHeaders, $items);
            if ($authRows['ok']) {
                $batchItems = is_array($authRows['items'] ?? null) ? $authRows['items'] : [];
            } else {
                $fetchError = (string)($authRows['msg'] ?? '授权账号查询失败');
            }
        }
        if (!payloadHasData(['items' => $items, 'batchItems' => []]) && payloadHasData(['items' => [], 'batchItems' => $batchItems])) {
            $items = $batchItems;
        }

        if ($fetchError === '') {
            $batchFallbackPayload = null;
            if (!empty($authorizedAccounts) && !rowsHaveData($batchItems)) {
                $batchFallbackPayload = firstPayloadWithSectionData('batchItems', $lastGood, $dateLastGood, $staleCached);
                if ($batchFallbackPayload) {
                    $batchItems = is_array($batchFallbackPayload['batchItems'] ?? null) ? $batchFallbackPayload['batchItems'] : [];
                    $cacheAge = time() - (int)($batchFallbackPayload['fetchTime'] ?? time());
                    $fromCache = true;
                    $usingLastGood = true;
                    $warning = '授权账号本次返回空数据，已继续使用最近一次有效的子账户缓存（' . max(0, (int)$cacheAge) . ' 秒前）。';
                }
            }

            $freshPayload = [
                'items' => $items,
                'batchItems' => $batchItems,
                'fetchTime' => time(),
                'date' => $date,
                'apiType' => $apiType,
                'api_type' => $apiType,
            ];
            // 切换日：只展示新账号数据，不再把旧账号缓存合并进来（避免与当前白名单串数据、刷新回滚）
            if ($isSwitchDay) {
                $freshPayload['switchCombined'] = true;
            }
            $hasData = payloadHasData($freshPayload);
            $bestSnapshot = $isSwitchDay
                ? ((is_array($dateLastGood) && !empty($dateLastGood['switchCombined'])) ? $dateLastGood : null)
                : ($lastGood ?: ($dateLastGood ?: $staleCached));

            $fallbackPayload = null;
            if (!$hasData) {
                if ($lastGood && payloadHasData($lastGood)) {
                    $fallbackPayload = $lastGood;
                } elseif ($dateLastGood && payloadHasData($dateLastGood)) {
                    $fallbackPayload = $dateLastGood;
                } elseif ($staleCached && payloadHasData($staleCached)) {
                    $fallbackPayload = $staleCached;
                }
            }

            if (!$hasData && $fallbackPayload) {
                // 本次没查到数据才回滚到最近一次有数据的缓存
                $items = is_array($fallbackPayload['items'] ?? null) ? $fallbackPayload['items'] : [];
                $batchItems = is_array($fallbackPayload['batchItems'] ?? null) ? $fallbackPayload['batchItems'] : [];
                $cacheAge = time() - (int)($fallbackPayload['fetchTime'] ?? time());
                $fromCache = true;
                $usingLastGood = true;
                $warning = '上游本次返回空数据，已继续使用最近一次有数据的缓存（' . max(0, (int)$cacheAge) . ' 秒前）。';
                cacheSet($cacheKey, [
                    'items' => $items,
                    'batchItems' => $batchItems,
                    'fetchTime' => (int)($fallbackPayload['fetchTime'] ?? time()),
                    'lastTryTime' => time(),
                ]);
                setFetchCooldown($cacheKey, $RETRY_BACKOFF, '上游返回空数据');
                $cooldownLeft = $RETRY_BACKOFF;
            } elseif (!$hasData) {
                $error = '上游本次返回空数据，未使用本次结果；暂无可用历史缓存，请稍后刷新。';
                $fromCache = false;
                $cacheAge = 0;
                setFetchCooldown($cacheKey, $CACHE_TTL, '上游返回空数据且无可用历史缓存');
                $cooldownLeft = $CACHE_TTL;
            } else {
                // 查到数据就立即写入最新缓存（单调合并），不再因“部分/稀疏返回”回滚到旧数据
                $persistPayload = cxMergePayloadMonotonic($freshPayload, is_array($bestSnapshot) ? $bestSnapshot : null);
                $items = $persistPayload['items'];
                $batchItems = $persistPayload['batchItems'];
                cacheSet($cacheKey, $persistPayload);
                lastGoodSet($cacheKey, $persistPayload);
                dateLastGoodSet($date, $persistPayload);
                if (!$batchFallbackPayload) {
                    $fromCache = false;
                    $cacheAge = 0;
                }
                clearFetchCooldown($cacheKey);
                $cooldownLeft = 0;
            }
        }
    }

    if ($fetchError !== '') {
        $isRateLimit = cxIsRateLimit($fetchError);
        $cooldownSeconds = $isRateLimit ? 180 : $CACHE_TTL;
        $cooldownLeft = setFetchCooldown($cacheKey, $cooldownSeconds, $fetchError, $isRateLimit);
        // 上游异常：优先用 last-good 顶上，再退到过期但有效的普通缓存
        $errorFallbackPayload = null;
        if (!$fromCache) {
            if ($lastGood && payloadHasData($lastGood)) {
                $errorFallbackPayload = $lastGood;
            } elseif ($dateLastGood && payloadHasData($dateLastGood)) {
                $errorFallbackPayload = $dateLastGood;
            } elseif ($staleCached && payloadHasData($staleCached)) {
                $errorFallbackPayload = $staleCached;
            }
        }
        if ($errorFallbackPayload) {
            $items = is_array($errorFallbackPayload['items'] ?? null) ? $errorFallbackPayload['items'] : [];
            $batchItems = is_array($errorFallbackPayload['batchItems'] ?? null) ? $errorFallbackPayload['batchItems'] : [];
            $cacheAge = time() - (int)($errorFallbackPayload['fetchTime'] ?? time());
            $fromCache = true;
            $usingLastGood = true;
            $warning = '上游接口暂时不可用：' . $fetchError . '，已使用最近一次有数据的缓存（' . max(0, (int)$cacheAge) . ' 秒前）。';
        } elseif ($fromCache) {
            $warning = '上游接口暂时不可用：' . $fetchError . '，' . $CACHE_TTL . ' 秒内将继续使用缓存数据。';
        } else {
            $error = $fetchError;
        }
    }
}
if ($fetchLockHandle !== null) {
    cxFetchLockRelease($fetchLockHandle);
}

if ($cooldownLeft > 0 && $warning === '' && $error === '' && $fromCache) {
    $cooldownReason = fetchCooldownReason($cacheKey);
    $warning = '上游接口冷却中，' . $cooldownLeft . ' 秒后再尝试刷新'
        . ($cooldownReason !== '' ? '（上次失败：' . $cooldownReason . '）' : '') . '。';
}

$totalTake = $totalComplete = $totalSuccess = $totalFail = 0;
foreach ($items as $row) {
    $totalTake += (int)($row['takeCount'] ?? 0);
    $totalComplete += (int)($row['completeCount'] ?? 0);
    $totalSuccess += (int)($row['checkSuccessCount'] ?? 0);
    $totalFail += (int)($row['checkFailCount'] ?? 0);
}

$activeBatch = array_values(array_filter($batchItems, fn($row) => is_array($row) && ($row['_src'] ?? '') !== 'old' && activityTotal($row) > 0));
$inactiveAuthorized = [];
$activeNames = [];
foreach ($batchItems as $row) {
    if (($row['_src'] ?? '') !== 'old') $activeNames[(string)($row['username'] ?? '')] = true;
}
foreach ($authorizedAccounts as $username => $remark) {
    if (empty($activeNames[$username])) $inactiveAuthorized[] = ['username' => $username, '_remark' => $remark];
}

$byRemark = [];
foreach ($batchItems as $row) {
    $remark = trim((string)($row['_remark'] ?? '')) ?: '未备注';
    if (!isset($byRemark[$remark])) {
        $byRemark[$remark] = ['take' => 0, 'complete' => 0, 'success' => 0, 'fail' => 0, 'accounts' => 0];
    }
    $byRemark[$remark]['take'] += (int)($row['takeCount'] ?? 0);
    $byRemark[$remark]['complete'] += (int)($row['completeCount'] ?? 0);
    $byRemark[$remark]['success'] += (int)($row['checkSuccessCount'] ?? 0);
    $byRemark[$remark]['fail'] += (int)($row['checkFailCount'] ?? 0);
    $byRemark[$remark]['accounts']++;
}
foreach ($byRemark as $remark => &$info) {
    $unitPrice = cxGetPricePerSuccess($apiType, $remark);
    $info['unitPrice'] = $unitPrice;
    $info['totalAmount'] = ($unitPrice !== null) ? round($unitPrice * $info['success'], 2) : null;
}
unset($info);
$totalAuthAmount = 0;
foreach ($byRemark as $info) {
    if (isset($info['totalAmount']) && $info['totalAmount'] !== null) {
        $totalAuthAmount += $info['totalAmount'];
    }
}
$selfAmount = 0;
$selfPrice = cxGetPricePerSuccess($apiType, '自己');
foreach ($byRemark as $remark => $info) {
    if ($remark === '自己' && isset($info['totalAmount']) && $info['totalAmount'] !== null) {
        $selfAmount = $info['totalAmount'];
        break;
    }
}
uasort($byRemark, fn($a, $b) => $b['success'] <=> $a['success']);
usort($batchItems, fn($a, $b) => ((int)($b['checkSuccessCount'] ?? 0)) <=> ((int)($a['checkSuccessCount'] ?? 0)));

$authTake = $authComplete = $authSuccess = $authFail = 0;
foreach ($batchItems as $row) {
    $authTake     += (int)($row['takeCount'] ?? 0);
    $authComplete += (int)($row['completeCount'] ?? 0);
    $authSuccess  += (int)($row['checkSuccessCount'] ?? 0);
    $authFail     += (int)($row['checkFailCount'] ?? 0);
}
// 利润 = (总审核成功数 × 自己的单价) − 其他人的金额
$totalProfit = ($selfPrice !== null) ? round($selfPrice * $authSuccess, 2) - ($totalAuthAmount - $selfAmount) : 0;
$authCompleteRate = rateText($authComplete, max(1, $authTake));
$authSuccessRate  = rateText($authSuccess, max(1, $authComplete));

$today = date('Y-m-d');
$yesterday = date('Y-m-d', strtotime('-1 day'));
$successRate = rateText($totalSuccess, max(1, $totalComplete));
$completeRate = rateText($totalComplete, max(1, $totalTake));
$topRemark = !empty($byRemark) ? (string)array_key_first($byRemark) : '暂无';
$topRemarkSuccess = !empty($byRemark) ? (int)($byRemark[$topRemark]['success'] ?? 0) : 0;
$topAccount = !empty($batchItems) ? (string)($batchItems[0]['username'] ?? '暂无') : '暂无';
$topAccountRemark = !empty($batchItems) ? (string)(($batchItems[0]['_remark'] ?? '') ?: '未备注') : '未备注';
$activeAuthorizedCount = count($activeBatch);
$autoRefreshDelay = $cooldownLeft > 0
    ? $cooldownLeft
    : max($MIN_REFRESH_DELAY, $CACHE_TTL - min($CACHE_TTL, max(0, (int)$cacheAge)));
$brandDateText = $date . ' / ' . $apiLabel . ' / ' . '授权账号 ' . count($authorizedAccounts) . ' / ' . '自动刷新 ' . $CACHE_TTL . ' ' . '秒';
$authorizedTotal = count($authorizedAccounts);
$authorizedReturned = count($batchItems);
$authorizedInactive = count($inactiveAuthorized);
$activeRate = rateText($activeAuthorizedCount, max(1, $authorizedTotal));
$returnedRate = rateText($authorizedReturned, max(1, $authorizedTotal));
$topRemarkShare = rateText($topRemarkSuccess, max(1, $totalSuccess));
$inactivePreview = array_slice($inactiveAuthorized, 0, 4);

$cxPayload = [
    'ok' => $error === '',
    'apiType' => $apiType,
    'apiLabel' => $apiLabel,
    'date' => $date,
    'items' => is_array($items) ? $items : [],
    'batchItems' => is_array($batchItems) ? $batchItems : [],
    'authorizedAccounts' => is_array($authorizedAccounts) ? $authorizedAccounts : [],
    'authorizedError' => $authorizedError,
    'error' => $error,
    'warning' => $warning,
    'fromCache' => $fromCache,
    'cacheAge' => $cacheAge,
    'usingLastGood' => $usingLastGood,
];

if (defined('CX_DATA_ONLY') && CX_DATA_ONLY === true) {
    return $cxPayload;
}

if ((string)($_GET['format'] ?? '') === 'json') {
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode($cxPayload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}
?>
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ajie906数据</title>
  <style>
    :root{
      --bg:#07111f;
      --bg-2:#0a1628;
      --surface:#0f1d33;
      --surface-2:#13243c;
      --surface-3:#0c182b;
      --line:rgba(148, 163, 184, .16);
      --line-strong:rgba(148, 163, 184, .26);
      --text:#f2f5f8;
      --muted:#93a1b5;
      --soft:#6f7e96;
      --accent:#22c55e;
      --accent-2:#38bdf8;
      --ok:#34d399;
      --bad:#fb7185;
      --warn:#fbbf24;
      --radius:8px;
      --shadow:0 24px 60px rgba(0, 0, 0, .28);
      --font:"Avenir Next","Segoe UI Variable","Microsoft YaHei UI","PingFang SC","Helvetica Neue",sans-serif;
    }
    *{box-sizing:border-box}
    html{scroll-behavior:smooth}
    body{
      margin:0;
      min-height:100vh;
      color:var(--text);
      font-family:var(--font);
      background:
        radial-gradient(1200px 540px at 14% -8%, rgba(56,189,248,.16), transparent 58%),
        radial-gradient(920px 520px at 92% 2%, rgba(34,197,94,.11), transparent 54%),
        linear-gradient(180deg, var(--bg) 0%, #08101c 38%, #050b13 100%);
    }
    body::before{
      content:"";
      position:fixed;
      inset:0;
      pointer-events:none;
      background:
        linear-gradient(rgba(255,255,255,.018) 1px, transparent 1px),
        linear-gradient(90deg, rgba(255,255,255,.018) 1px, transparent 1px);
      background-size:36px 36px;
      mask-image:linear-gradient(180deg, rgba(0,0,0,.45), transparent 90%);
    }
    body[data-theme="light"]{
      --bg:#f5f7fb;
      --bg-2:#eef3f8;
      --surface:#ffffff;
      --surface-2:#fbfdff;
      --surface-3:#f2f5f8;
      --line:rgba(15, 23, 42, .08);
      --line-strong:rgba(15, 23, 42, .16);
      --text:#0f172a;
      --muted:#475569;
      --soft:#64748b;
      --shadow:0 24px 54px rgba(15, 23, 42, .08);
      background:
        radial-gradient(1100px 540px at 8% -8%, rgba(56,189,248,.11), transparent 58%),
        radial-gradient(900px 500px at 92% 0%, rgba(34,197,94,.08), transparent 56%),
        linear-gradient(180deg, #f8fafc 0%, #eef3f8 58%, #e8eef5 100%);
    }
    body[data-theme="graphite"]{
      --bg:#111317;
      --bg-2:#1a1d22;
      --surface:#1a1f27;
      --surface-2:#202631;
      --surface-3:#151a22;
      --line:rgba(203,213,225,.12);
      --line-strong:rgba(203,213,225,.22);
      --text:#f1f5f9;
      --muted:#b1bac8;
      --soft:#8792a4;
      --accent:#f59e0b;
      --accent-2:#60a5fa;
      background:
        radial-gradient(1100px 540px at 14% -8%, rgba(96,165,250,.14), transparent 58%),
        radial-gradient(920px 520px at 92% 2%, rgba(245,158,11,.10), transparent 54%),
        linear-gradient(180deg, #111317 0%, #171b21 38%, #0d1117 100%);
    }
    body[data-theme="ocean"]{
      --bg:#071422;
      --bg-2:#0d2034;
      --surface:#0e2238;
      --surface-2:#16304b;
      --surface-3:#0a1a2d;
      --line:rgba(125,211,252,.14);
      --line-strong:rgba(125,211,252,.22);
      --text:#edf6ff;
      --muted:#b8cde3;
      --soft:#7f9bb9;
      --accent:#38bdf8;
      --accent-2:#818cf8;
      background:
        radial-gradient(1100px 540px at 14% -8%, rgba(56,189,248,.18), transparent 58%),
        radial-gradient(920px 520px at 92% 2%, rgba(129,140,248,.10), transparent 54%),
        linear-gradient(180deg, #06111d 0%, #091a2d 38%, #040c15 100%);
    }
    a{color:inherit;text-decoration:none}
    .page{max-width:1380px;margin:0 auto;padding:24px 22px 56px}
    .topbar{
      position:sticky;top:0;z-index:12;
      display:grid;grid-template-columns:minmax(280px, 1fr) auto;gap:18px;align-items:center;
      padding:14px 18px;margin-bottom:18px;
      background:color-mix(in srgb, var(--surface) 88%, transparent);
      backdrop-filter:blur(20px);
      border:1px solid var(--line);
      border-radius:var(--radius);
      box-shadow:var(--shadow);
    }
    .brand{
      display:grid;grid-template-columns:52px minmax(0,1fr);gap:14px;align-items:center;
    }
    .logo{
      width:52px;height:52px;border-radius:var(--radius);overflow:hidden;
      border:1px solid var(--line-strong);background:var(--surface-3);
      box-shadow:0 12px 32px rgba(0,0,0,.14);
    }
    .logo img{width:100%;height:100%;object-fit:cover}
    .brand-name{font-size:18px;font-weight:900;letter-spacing:.02em}
    .brand-meta{margin-top:4px;color:var(--soft);font-size:12px}
    .toolbar{display:flex;align-items:center;justify-content:flex-end;gap:10px;flex-wrap:wrap}
    .chip{
      display:inline-flex;align-items:center;gap:8px;height:34px;padding:0 12px;
      border-radius:999px;border:1px solid var(--line);background:rgba(255,255,255,.02);
      color:var(--muted);font-size:12px;font-weight:750;
    }
    .chip b{color:var(--text)}
    .chip-row{display:inline-flex;flex-wrap:wrap;gap:8px;justify-content:flex-end}
    .control,.btn{
      height:40px;padding:0 14px;border:1px solid var(--line);
      border-radius:var(--radius);background:var(--surface-3);color:var(--text);
      font:inherit;font-size:14px;
    }
    .btn{
      display:inline-flex;align-items:center;justify-content:center;cursor:pointer;
      transition:.18s ease;
    }
    .btn:hover{border-color:var(--line-strong);transform:translateY(-1px)}
    .btn.primary{
      border-color:transparent;
      background:linear-gradient(135deg, var(--accent), var(--accent-2));
      color:#05110f;font-weight:900;
    }
    .btn.ghost{background:transparent}
    .theme-switch{
      display:inline-flex;align-items:center;gap:4px;padding:4px;
      border:1px solid var(--line);border-radius:999px;background:rgba(255,255,255,.02);
    }
    .theme-btn{
      min-width:58px;height:30px;padding:0 10px;border:0;border-radius:999px;
      background:transparent;color:var(--muted);font:inherit;font-size:12px;font-weight:800;cursor:pointer;
    }
    .theme-btn.active{background:rgba(56,189,248,.14);color:var(--text)}
    .hero{
      display:grid;grid-template-columns:minmax(0, 1.4fr) minmax(300px, .9fr);gap:18px;margin-bottom:18px;
    }
    .band{
      position:relative;overflow:hidden;
      border:1px solid var(--line);border-radius:var(--radius);
      background:
        linear-gradient(180deg, color-mix(in srgb, var(--surface-2) 92%, transparent), color-mix(in srgb, var(--surface) 98%, transparent));
      box-shadow:var(--shadow);
    }
    .hero-main{padding:28px 28px 24px;min-height:320px}
    .hero-main::after{
      content:"";
      position:absolute;right:-12px;top:-18px;width:340px;height:340px;pointer-events:none;
      background:
        linear-gradient(180deg, rgba(255,255,255,.04), rgba(255,255,255,0)),
        url("<?= e($LOGO_PATH) ?>") center/contain no-repeat;
      opacity:.09;filter:saturate(0) contrast(1.1);
    }
    .eyebrow{
      display:inline-flex;align-items:center;gap:8px;height:28px;padding:0 10px;
      border-radius:999px;border:1px solid var(--line);
      color:var(--soft);font-size:11px;font-weight:900;letter-spacing:.16em;text-transform:uppercase;
    }
    .hero-title{
      margin:18px 0 10px;font-size:42px;line-height:1.04;font-weight:980;letter-spacing:.01em;
    }
    .hero-sub{
      max-width:760px;color:var(--muted);font-size:14px;line-height:1.75;
    }
    .summary{
      display:grid;grid-template-columns:repeat(4, minmax(0,1fr));gap:12px;margin-top:24px;
    }
    .summary-item{
      padding:16px 14px;border:1px solid var(--line);border-radius:var(--radius);
      background:linear-gradient(180deg, rgba(255,255,255,.03), rgba(255,255,255,.01));
      min-height:106px;
    }
    .summary-item .label{
      color:var(--soft);font-size:11px;font-weight:900;letter-spacing:.18em;text-transform:uppercase;
    }
    .summary-item .value{
      margin-top:12px;font-size:32px;font-weight:980;letter-spacing:.01em;
    }
    .summary-item .hint{
      margin-top:8px;color:var(--muted);font-size:12px;
    }
    .summary-item.success{border-top:3px solid rgba(52,211,153,.62)}
    .summary-item.progress{border-top:3px solid rgba(56,189,248,.62)}
    .summary-item.complete{border-top:3px solid rgba(59,130,246,.62)}
    .summary-item.fail{border-top:3px solid rgba(251,113,133,.62)}
    .hero-insights{
      margin-top:14px;
      display:grid;
      grid-template-columns:minmax(0,1.2fr) minmax(280px,.8fr);
      gap:12px;
    }
    .insight-panel{
      border:1px solid var(--line);
      border-radius:var(--radius);
      background:linear-gradient(180deg, rgba(255,255,255,.025), rgba(255,255,255,.01));
      padding:16px;
    }
    .insight-title{
      margin:0 0 12px;
      font-size:11px;
      font-weight:900;
      letter-spacing:.18em;
      text-transform:uppercase;
      color:var(--soft);
    }
    .insight-grid{
      display:grid;
      grid-template-columns:repeat(3,minmax(0,1fr));
      gap:10px;
    }
    .insight-box{
      padding:12px 12px 10px;
      border:1px solid rgba(148,163,184,.10);
      border-radius:var(--radius);
      background:rgba(255,255,255,.015);
      min-height:88px;
    }
    .insight-box .k{font-size:11px;color:var(--soft);font-weight:850;letter-spacing:.12em;text-transform:uppercase}
    .insight-box .v{margin-top:10px;font-size:24px;font-weight:960}
    .insight-box .d{margin-top:8px;font-size:12px;color:var(--muted);line-height:1.5}
    .signal-list{display:grid;gap:10px}
    .signal-row{
      display:flex;justify-content:space-between;gap:12px;align-items:center;
      padding:12px 0;border-top:1px solid rgba(148,163,184,.10);
      font-size:13px;color:var(--muted);
    }
    .signal-row:first-child{padding-top:0;border-top:0}
    .signal-row b{color:var(--text);font-weight:900}
    .tiny-users{
      display:flex;flex-wrap:wrap;gap:8px;margin-top:6px;
    }
    .tiny-pill{
      display:inline-flex;align-items:center;height:28px;padding:0 10px;
      border-radius:999px;border:1px solid rgba(148,163,184,.14);
      background:rgba(255,255,255,.02);color:var(--muted);font-size:12px;font-weight:700;
    }
    .hero-side{
      display:grid;grid-template-rows:1fr 1fr;
    }
    .hero-side .pane{
      padding:20px 20px 18px;border-top:1px solid var(--line);
    }
    .hero-side .pane:first-child{border-top:0}
    .pane-title{
      margin:0 0 14px;font-size:12px;font-weight:900;letter-spacing:.18em;text-transform:uppercase;color:var(--soft);
    }
    .metric-list{display:grid;gap:12px}
    .metric-row{
      display:flex;justify-content:space-between;gap:14px;padding-top:12px;border-top:1px solid rgba(148,163,184,.12);
      font-size:13px;color:var(--muted);
    }
    .metric-row:first-child{padding-top:0;border-top:0}
    .metric-row b{color:var(--text);font-weight:900}
    .ok{color:var(--ok)!important}
    .bad{color:var(--bad)!important}
    .warn{color:var(--warn)!important}
    .section{margin-top:18px}
    .section-header{
      display:flex;align-items:end;justify-content:space-between;gap:18px;margin-bottom:12px;flex-wrap:wrap;
    }
    .section-title{margin:0;font-size:20px;font-weight:960;letter-spacing:.01em}
    .section-desc{margin-top:6px;color:var(--muted);font-size:13px}
    .group-grid{
      display:grid;grid-template-columns:repeat(auto-fill, minmax(230px,1fr));gap:12px;
    }
    .group-card{
      padding:16px;border:1px solid var(--line);border-radius:var(--radius);
      background:linear-gradient(180deg, rgba(255,255,255,.03), rgba(255,255,255,.01));
      box-shadow:var(--shadow);
    }
    .group-card:hover{border-color:var(--line-strong)}
    .group-top{
      display:flex;justify-content:space-between;gap:12px;align-items:flex-start;margin-bottom:14px;
    }
    .group-name{font-size:20px;font-weight:950;line-height:1.1}
    .group-count{font-size:11px;color:var(--soft);font-weight:800;letter-spacing:.14em;text-transform:uppercase}
    .group-stat{
      display:flex;justify-content:space-between;gap:12px;padding-top:10px;margin-top:10px;
      border-top:1px solid rgba(148,163,184,.12);font-size:13px;color:var(--muted);
    }
    .group-stat b{color:var(--text);font-weight:900}
    .group-amount-stat{border-top-color:rgba(52,211,153,.22)}
    .group-amount-stat .amount-unit{font-size:11px;color:var(--soft)}
    .group-amount-stat.pulse{background:rgba(52,211,153,.12);transition:background .2s ease}
    .table-band{
      border:1px solid var(--line);border-radius:var(--radius);overflow:auto;background:var(--surface);
      box-shadow:var(--shadow);
    }
    table{width:100%;border-collapse:collapse;min-width:920px}
    th{
      position:sticky;top:0;z-index:2;text-align:left;padding:14px 16px;
      background:color-mix(in srgb, var(--surface-3) 92%, transparent);
      border-bottom:1px solid var(--line);
      color:var(--soft);font-size:11px;font-weight:900;letter-spacing:.16em;text-transform:uppercase;
    }
    td{
      padding:14px 16px;border-bottom:1px solid rgba(148,163,184,.10);font-size:14px;white-space:nowrap;
    }
    tbody tr:hover td{background:rgba(56,189,248,.035)}
    tbody tr:last-child td{border-bottom:0}
    tfoot.sum-row td,tfoot tr.sum-row td{
      position:sticky;bottom:0;
      background:color-mix(in srgb, var(--surface-3) 94%, transparent);
      border-top:1px solid var(--line-strong);
      font-weight:900;font-size:14px;
    }
    .user{font-weight:900}
    .remark{color:var(--accent);font-weight:850}
    .src-tag{display:inline-block;margin-left:6px;padding:1px 7px;border-radius:999px;background:rgba(251,191,36,.16);color:var(--warn);font-size:11px;font-weight:800;vertical-align:2px}
    .empty{
      padding:44px 18px;text-align:center;color:var(--muted);font-size:14px;
      border:1px dashed var(--line);border-radius:var(--radius);background:rgba(255,255,255,.02);
    }
    .notice,.alert{
      margin-bottom:16px;padding:14px 16px;border:1px solid var(--line);border-radius:var(--radius);
      background:rgba(255,255,255,.03);color:var(--muted)
    }
    .notice{border-color:rgba(251,191,36,.24);background:rgba(251,191,36,.08);color:#f5d487}
    .alert{border-color:rgba(251,113,133,.28);background:rgba(251,113,133,.09);color:#fecdd3}
    @media(max-width:1180px){
      .hero{grid-template-columns:1fr}
      .summary{grid-template-columns:repeat(2, minmax(0,1fr))}
      .hero-insights{grid-template-columns:1fr}
    }
    @media(max-width:720px){
      .page{padding:16px 14px 46px}
      .topbar{grid-template-columns:1fr}
      .toolbar{justify-content:flex-start}
      .hero-title{font-size:32px}
      .summary{grid-template-columns:1fr}
      .insight-grid{grid-template-columns:1fr}
      .control,.btn{width:100%}
      .toolbar form{width:100%;display:grid!important;grid-template-columns:1fr;gap:10px}
      .toolbar .btn.ghost{flex:1 1 auto}
    }
  </style>
</head>
<body>
  <div
    class="page"
    id="cxApp"
    data-date="<?= e($date) ?>"
    data-refresh-delay="<?= (int)$autoRefreshDelay ?>"
    data-cache-ttl="<?= (int)$CACHE_TTL ?>"
    data-api-type="<?= (int)$apiType ?>"
  >
    <div class="topbar">
      <div class="brand">
        <div class="logo"><img src="<?= e($LOGO_PATH) ?>" alt="Logo"></div>
        <div>
          <div class="brand-name">ajie906数据</div>
          <div class="brand-meta"><?= e($brandDateText) ?></div>
        </div>
      </div>

      <div class="toolbar">
        <span class="chip">数据源 <b><?= $fromCache ? '缓存' : '实时' ?></b></span>
        <span class="chip">缓存年龄 <b><?= (int)$cacheAge ?>s</b></span>
        <span class="chip">下次更新 <b id="autoRefreshCountdown"><?= (int)$autoRefreshDelay ?>s</b></span>
        <a class="btn <?= $apiType === 1 ? 'primary' : 'ghost' ?>" href="<?= e(cxPageUrl(['api_type' => 1, 'date' => $date, 'force' => null, 'refresh' => null])) ?>">&#25509;&#21475;1</a>
        <a class="btn <?= $apiType === 2 ? 'primary' : 'ghost' ?>" href="<?= e(cxPageUrl(['api_type' => 2, 'date' => $date, 'force' => null, 'refresh' => null])) ?>">&#25509;&#21475;2</a>
        <div class="theme-switch" aria-label="皮肤切换">
          <button class="theme-btn" type="button" data-theme-value="dark">暗夜</button>
          <button class="theme-btn" type="button" data-theme-value="light">云白</button>
          <button class="theme-btn" type="button" data-theme-value="graphite">钛灰</button>
          <button class="theme-btn" type="button" data-theme-value="ocean">深海</button>
        </div>
        <a class="btn ghost" href="<?= e(cxPageUrl(['date' => $today, 'api_type' => $apiType, 'force' => null, 'refresh' => null])) ?>">&#20170;&#22825;</a>
        <a class="btn ghost" href="<?= e(cxPageUrl(['date' => $yesterday, 'api_type' => $apiType, 'force' => null, 'refresh' => null])) ?>">&#26152;&#22825;</a>
        <form method="get" style="display:flex;gap:10px;align-items:center">
          <input class="control" type="date" name="date" value="<?= e($date) ?>">
          <input type="hidden" name="api_type" value="<?= (int)$apiType ?>">
          <button class="btn primary" type="submit">查询</button>
        </form>
        <a class="btn" href="<?= e(cxPageUrl(['date' => $date, 'api_type' => $apiType, 'force' => '1', 'refresh' => '1'])) ?>">&#21047;&#26032;&#39029;&#38754;</a>
      </div>
    </div>

    <?php if ($authorizedError): ?>
      <div class="notice"><?= e($authorizedError) ?></div>
    <?php endif; ?>
    <?php if ($error): ?>
      <div class="alert"><?= e($error) ?></div>
    <?php endif; ?>
    <?php if (!$error && $warning): ?>
      <div class="notice"><?= e($warning) ?></div>
    <?php endif; ?>

    <div class="hero">
      <section class="band hero-main">
        <div class="eyebrow">Operations Overview</div>
        <h1 class="hero-title"><?= e($date) ?> <?= e($apiLabel) ?> &#25968;&#25454;&#30475;&#26495;</h1>
        <div class="hero-sub">

        </div>

        <?php if (!$error): ?>
        <div class="summary">
          <div class="summary-item progress">
            <div class="label">Take</div>
            <div class="value"><?= num($totalTake) ?></div>
            <div class="hint">全部账号当日领取总数</div>
          </div>
          <div class="summary-item complete">
            <div class="label">Complete</div>
            <div class="value"><?= num($totalComplete) ?></div>
            <div class="hint">完成率 <?= e($completeRate) ?></div>
          </div>
          <div class="summary-item success">
            <div class="label">Success</div>
            <div class="value"><?= num($totalSuccess) ?></div>
            <div class="hint">通过率 <?= e($successRate) ?></div>
          </div>
          <div class="summary-item fail">
            <div class="label">Fail</div>
            <div class="value"><?= num($totalFail) ?></div>
            <div class="hint">失败数 <?= num($totalFail) ?></div>
          </div>
        </div>

        <div class="hero-insights">
          <div class="insight-panel">
            <h3 class="insight-title">运营摘要</h3>
            <div class="insight-grid">
              <div class="insight-box">
                <div class="k">活跃率</div>
                <div class="v ok"><?= e($activeRate) ?></div>
                <div class="d">活跃账号 <?= $activeAuthorizedCount ?> / 授权账号 <?= $authorizedTotal ?></div>
              </div>
              <div class="insight-box">
                <div class="k">返回率</div>
                <div class="v"><?= e($returnedRate) ?></div>
                <div class="d">接口返回 <?= $authorizedReturned ?> 个授权账号结果</div>
              </div>
              <div class="insight-box">
                <div class="k">冠军占比</div>
                <div class="v"><?= e($topRemarkShare) ?></div>
                <div class="d"><?= e($topRemark) ?> 贡献了当前通过量的主要份额</div>
              </div>
            </div>
          </div>

          <div class="insight-panel">
            <h3 class="insight-title">自动洞察</h3>
            <div class="signal-list">
              <div class="signal-row"><span>最高产出分组</span><b class="ok"><?= e($topRemark) ?></b></div>
              <div class="signal-row"><span>最高产出账号</span><b><?= e($topAccount) ?></b></div>
              <div class="signal-row"><span>无返回账号</span><b class="warn"><?= $authorizedInactive ?></b></div>
              <?php if (!empty($inactivePreview)): ?>
              <div class="tiny-users">
                <?php foreach ($inactivePreview as $row): ?>
                <span class="tiny-pill"><?= e($row['username']) ?></span>
                <?php endforeach; ?>
                <?php if ($authorizedInactive > count($inactivePreview)): ?>
                <span class="tiny-pill">+<?= $authorizedInactive - count($inactivePreview) ?></span>
                <?php endif; ?>
              </div>
              <?php endif; ?>
            </div>
          </div>
        </div>
        <?php endif; ?>
      </section>

      <aside class="band hero-side">
        <div class="pane">
          <h3 class="pane-title">执行摘要</h3>
          <div class="metric-list">
            <div class="metric-row"><span>Top 备注</span><b class="ok"><?= e($topRemark) ?></b></div>
            <div class="metric-row"><span>Top 备注通过</span><b class="ok"><?= num($topRemarkSuccess) ?></b></div>
            <div class="metric-row"><span>Top 账号</span><b><?= e($topAccount) ?></b></div>
            <div class="metric-row"><span>Top 账号备注</span><b class="ok"><?= e($topAccountRemark) ?></b></div>
          </div>
        </div>
        <div class="pane">
          <h3 class="pane-title">授权状态</h3>
          <div class="metric-list">
            <div class="metric-row"><span>授权账号</span><b><?= (int)count($authorizedAccounts) ?></b></div>
            <div class="metric-row"><span>接口返回</span><b><?= (int)count($batchItems) ?></b></div>
            <div class="metric-row"><span>活跃账号</span><b class="ok"><?= (int)$activeAuthorizedCount ?></b></div>
            <div class="metric-row"><span>无返回账号</span><b class="warn"><?= (int)count($inactiveAuthorized) ?></b></div>
          </div>
        </div>
      </aside>
    </div>
    <?php if (!$error): ?>
    <section class="section">
      <div class="section-header">
        <div>
          <h2 class="section-title">备注分组绩效</h2>
          <div class="section-desc">按备注聚合“后台已授权且当天有数据”的账号结果，适合快速看团队或业务组表现。</div>
        </div>
        <div class="chip-row">
          <span class="chip">分组 <b><?= (int)count($byRemark) ?></b></span>
          <span class="chip">领取 <b><?= num($authTake) ?></b></span>
          <span class="chip">完成 <b><?= num($authComplete) ?></b></span>
          <span class="chip">通过 <b class="ok"><?= num($authSuccess) ?></b></span>
          <span class="chip">失败 <b class="bad"><?= num($authFail) ?></b></span>
          <?php $grossAmount = ($selfPrice !== null) ? round($selfPrice * $authSuccess, 2) : 0; ?>
          <?php if ($grossAmount > 0): ?>
          <span class="chip">预估金额 <b class="ok"><?= e(cxFormatMoney($grossAmount)) ?></b></span>
          <span class="chip">当前利润 <b class="warn"><?= e(cxFormatMoney($totalProfit)) ?></b></span>
          <?php endif; ?>
        </div>
      </div>

      <?php if (!empty($byRemark)): ?>
      <div class="group-grid">
        <?php foreach ($byRemark as $remark => $tot): ?>
        <a class="group-card" href="#group-<?= e(md5($remark)) ?>">
          <div class="group-top">
            <div class="group-name"><?= e($remark) ?></div>
            <div class="group-count"><?= (int)$tot['accounts'] ?> 个账号</div>
          </div>
          <div class="group-stat"><span>领取</span><b><?= num($tot['take']) ?></b></div>
          <div class="group-stat"><span>完成</span><b><?= num($tot['complete']) ?></b></div>
          <div class="group-stat"><span>通过</span><b class="ok"><?= num($tot['success']) ?></b></div>
          <div class="group-stat"><span>失败</span><b class="bad"><?= num($tot['fail']) ?></b></div>
          <?php if (isset($tot['unitPrice']) && $tot['unitPrice'] !== null): ?>
          <div class="group-stat group-amount-stat"><span>金额</span><b class="ok"><?= e(cxFormatMoney($tot['totalAmount'])) ?></b><span class="amount-unit">（<?= e(cxFormatMoney($tot['unitPrice'], 3)) ?>/条）</span></div>
          <?php endif; ?>
        </a>
        <?php endforeach; ?>
      </div>
      <?php else: ?>
      <div class="empty">暂无备注分组数据</div>
      <?php endif; ?>
    </section>

    <section class="section">
      <div class="section-header">
        <div>
          <h2 class="section-title">授权账号明细</h2>
          <div class="section-desc">后台已授权且当天有数据的账号明细，不再显示 0 数据账号。</div>
        </div>
        <div class="chip-row">
          <span class="chip">账号 <b><?= (int)count($batchItems) ?></b></span>
          <span class="chip">领取 <b><?= num($authTake) ?></b></span>
          <span class="chip">完成 <b><?= num($authComplete) ?></b></span>
          <span class="chip">通过 <b class="ok"><?= num($authSuccess) ?></b></span>
          <span class="chip">失败 <b class="bad"><?= num($authFail) ?></b></span>
        </div>
      </div>
      <div class="table-band">
        <table>
          <thead>
            <tr>
              <th>账号</th>
              <th>备注</th>
              <th>领取</th>
              <th>完成</th>
              <th>通过</th>
              <th>失败</th>
              <th>完成率</th>
              <th>通过率</th>
            </tr>
          </thead>
          <tbody>
            <?php if (!empty($batchItems)): foreach ($batchItems as $row): $rm = (string)(($row['_remark'] ?? '') ?: '未备注'); ?>
            <tr id="group-<?= e(md5($rm)) ?>">
              <td class="user"><?= e($row['username'] ?? '-') ?><?php if (($row['_src'] ?? '') === 'old'): ?> <span class="src-tag" title="切换前旧账号">旧</span><?php endif; ?></td>
              <td class="remark"><?= e($rm) ?></td>
              <td><?= num($row['takeCount'] ?? 0) ?></td>
              <td><?= num($row['completeCount'] ?? 0) ?></td>
              <td class="ok"><?= num($row['checkSuccessCount'] ?? 0) ?></td>
              <td class="bad"><?= num($row['checkFailCount'] ?? 0) ?></td>
              <td class="muted"><?= e($row['completeRate'] ?? rateText((int)($row['completeCount'] ?? 0), (int)($row['takeCount'] ?? 0))) ?></td>
              <td class="muted"><?= e($row['checkSuccessRate'] ?? rateText((int)($row['checkSuccessCount'] ?? 0), (int)($row['completeCount'] ?? 0))) ?></td>
            </tr>
            <?php endforeach; else: ?>
            <tr><td colspan="8"><div class="empty">授权账号暂无接口返回数据</div></td></tr>
            <?php endif; ?>
          </tbody>
          <?php if (!empty($batchItems)): ?>
          <tfoot>
            <tr class="sum-row">
              <td class="user">合计</td>
              <td class="remark"><?= (int)count($batchItems) ?> 个账号</td>
              <td><?= num($authTake) ?></td>
              <td><?= num($authComplete) ?></td>
              <td class="ok"><?= num($authSuccess) ?></td>
              <td class="bad"><?= num($authFail) ?></td>
              <td class="muted"><?= e($authCompleteRate) ?></td>
              <td class="muted"><?= e($authSuccessRate) ?></td>
            </tr>
          </tfoot>
          <?php endif; ?>
        </table>
      </div>
    </section>

    <section class="section">
      <div class="section-header">
        <div>
          <h2 class="section-title">全量账号汇总</h2>
          <div class="section-desc">By：ajie906数据。</div>
        </div>
        <span class="chip">记录 <b><?= (int)count($items) ?></b></span>
      </div>
      <div class="table-band">
        <table>
          <thead>
            <tr>
              <th>账号</th>
              <th>领取</th>
              <th>完成</th>
              <th>通过</th>
              <th>失败</th>
              <th>完成率</th>
              <th>通过率</th>
            </tr>
          </thead>
          <tbody>
            <?php if (!empty($items)): foreach ($items as $row): ?>
            <tr>
              <td class="user"><?= e($row['username'] ?? '-') ?></td>
              <td><?= num($row['takeCount'] ?? 0) ?></td>
              <td><?= num($row['completeCount'] ?? 0) ?></td>
              <td class="ok"><?= num($row['checkSuccessCount'] ?? 0) ?></td>
              <td class="bad"><?= num($row['checkFailCount'] ?? 0) ?></td>
              <td class="muted"><?= e($row['completeRate'] ?? '-') ?></td>
              <td class="muted"><?= e($row['checkSuccessRate'] ?? '-') ?></td>
            </tr>
            <?php endforeach; else: ?>
            <tr><td colspan="7"><div class="empty">当前日期暂无全量数据</div></td></tr>
            <?php endif; ?>
          </tbody>
        </table>
      </div>
    </section>
    <?php endif; ?>
  </div>

  <script>
    (function () {
      const storageKey = 'cx-theme';
      const body = document.body;
      const allowedThemes = new Set(['dark', 'light', 'graphite', 'ocean']);
      let timer = null;
      let deadline = Date.now() + getRefreshDelay() * 1000;

      try {
        const cleanUrl = new URL(window.location.href);
        cleanUrl.searchParams.delete('auto');
        cleanUrl.searchParams.delete('_ts');
        window.history.replaceState(null, '', cleanUrl.toString());
      } catch (err) {}

      function getApp() {
        return document.getElementById('cxApp');
      }

      function getCurrentDate() {
        const app = getApp();
        return (app && app.dataset.date) || '<?= e($date) ?>';
      }

      function getRefreshDelay() {
        const app = getApp();
        const fallback = <?= (int)$CACHE_TTL ?>;
        const floor = <?= (int)$MIN_REFRESH_DELAY ?>;
        const raw = app ? parseInt(app.dataset.refreshDelay || '', 10) : fallback;
        return Number.isFinite(raw) ? Math.max(floor, raw) : fallback;
      }

      function applyTheme(theme) {
        const value = allowedThemes.has(theme) ? theme : 'dark';
        if (value === 'dark') body.removeAttribute('data-theme');
        else body.setAttribute('data-theme', value);
        localStorage.setItem(storageKey, value);
        document.querySelectorAll('[data-theme-value]').forEach((button) => {
          button.classList.toggle('active', button.dataset.themeValue === value);
        });
      }

      function bindThemeButtons() {
        document.querySelectorAll('[data-theme-value]').forEach((button) => {
          button.addEventListener('click', function () {
            applyTheme(button.dataset.themeValue || 'dark');
          });
        });
        applyTheme(localStorage.getItem(storageKey) || 'dark');
      }

      function renderCountdown() {
        const countdownEl = document.getElementById('autoRefreshCountdown');
        const remain = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
        if (countdownEl) countdownEl.textContent = remain + 's';
      }

      async function refreshSilently() {
        if (timer) {
          clearInterval(timer);
          timer = null;
        }

        const currentApp = getApp();
        const countdownEl = document.getElementById('autoRefreshCountdown');
        if (countdownEl) countdownEl.textContent = '更新中';

        try {
          const nextUrl = new URL(window.location.href);
          nextUrl.searchParams.set('date', getCurrentDate());
          nextUrl.searchParams.set('auto', '1');
          nextUrl.searchParams.delete('force');
          nextUrl.searchParams.set('_ts', String(Date.now()));
          nextUrl.searchParams.delete('format');

          const response = await fetch(nextUrl.toString(), {
            cache: 'no-store',
            headers: {'X-Requested-With': 'fetch'},
          });
          if (!response.ok) throw new Error('HTTP ' + response.status);

          const html = await response.text();
          const nextDoc = new DOMParser().parseFromString(html, 'text/html');
          const nextApp = nextDoc.getElementById('cxApp');
          if (!currentApp || !nextApp) throw new Error('Missing cxApp');

          currentApp.replaceWith(document.importNode(nextApp, true));
          bindThemeButtons();
          bindGroupCardClicks();
        } catch (err) {
          console.warn('CX silent refresh failed:', err);
        } finally {
          deadline = Date.now() + getRefreshDelay() * 1000;
          startTimer();
        }
      }

      function startTimer() {
        renderCountdown();
        timer = setInterval(function () {
          if (Date.now() >= deadline) {
            renderCountdown();
            refreshSilently();
            return;
          }
          renderCountdown();
        }, 1000);
      }

      function bindGroupCardClicks() {
        document.querySelectorAll('.group-card').forEach(function (card) {
          card.addEventListener('click', function () {
            var amountEl = card.querySelector('.group-amount-stat');
            if (!amountEl) return;
            amountEl.classList.add('pulse');
            setTimeout(function () { amountEl.classList.remove('pulse'); }, 700);
          });
        });
      }

      bindThemeButtons();
      bindGroupCardClicks();
      startTimer();

      document.addEventListener('visibilitychange', function () {
        if (!document.hidden && Date.now() >= deadline) {
          refreshSilently();
        }
      });
    })();
  </script>
</body>
</html>
