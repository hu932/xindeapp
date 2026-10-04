<?php
/**
 * 优惠券预检 - start 接口
 * 对应插件 coupon-preflight.js 的 POST /api/coupon-preflight/start
 */
require_once dirname(__DIR__, 3) . '/lib.php';

coupon_handle_options();

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    coupon_fail('只支持 POST 请求', 200);
}

handle_coupon_start();
