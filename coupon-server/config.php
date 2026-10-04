<?php
/**
 * 优惠券预检服务器配置
 *
 * 部署到自己服务器后，请按需修改本文件。
 */

return [
    // 鉴权 Token，必须与插件 init.js 中 优惠券预检配置.token 保持一致
    // 如需更换，两边要同时改。
    'auth_token' => '-CN6_PNYhHpDzN-qJw-RGecT5vOM1nY8zqORenKnFnw',

    // 安卓 App 上报数据解密密钥（AES-256-ECB，与后台 decrypt_proxy / go-server 一致）
    'aes_key' => '25d9a1907ab38e273c4f57c476e64377',

    // 后台登录账号 / 密码（务必修改！）
    'admin_user'     => 'ajie99',
    'admin_password' => '962669581',

    // 探测商品池：仅在首次使用时作为“默认数据”写入 storage/probe_products.json，
    // 之后请在后台里在线增删改，这里的值不再生效。
    // 下面是两个真实通过过优惠券预检的商品（来自 1.josn / 2.josn），可直接使用。
    'probe_products' => [
        [
            'id'     => 'probe-001',
            'url'    => 'https://shopee.tw/api/v4/pdp/get_rw?item_id=43954767076&shop_id=50662979&tz_offset_in_minutes=480&detail_level=0&incoming_pdp_page_source=0&incoming_pdp_page_scenario=0',
            'shopId' => '50662979',
            'itemId' => '43954767076',
            // 标题参考：味覺糖 特濃軟糖(牛奶味/焦糖味) 70g
        ],
        [
            'id'     => 'probe-002',
            'url'    => 'https://shopee.tw/api/v4/pdp/get_rw?item_id=46760554619&shop_id=50662979&tz_offset_in_minutes=480&detail_level=0&incoming_pdp_page_source=0&incoming_pdp_page_scenario=0',
            'shopId' => '50662979',
            'itemId' => '46760554619',
            // 标题参考：DIKE 6吋自動擺頭USB充電式風扇 DUF320
        ],
    ],

    // 上报数据校验
    'report_min_length'    => 20,     // rawResponseText 最小字符数，防止空数据直接通过
    'report_require_json'  => true,   // 真实通过的数据都是 JSON，这里要求必须为合法 JSON
    'report_require_coupon'=> true,   // 必须包含优惠券数据（promotion_drawer.shop_vouchers 等），否则不通过

    // 存储目录（需要 PHP 进程可写）
    'storage_dir' => __DIR__ . '/storage',
];
