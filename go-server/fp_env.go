package main

import (
	"crypto/md5"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var fpEnvMu sync.Mutex

// ==================== 指纹（环境）生成 ====================
// 与安卓 App v/f.java 的 generateFullFp 保持一致：随机字段 + 真实值格式。

func fpHexChars(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func fpUUID() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func fpBase64URLHex(hexChars int) string {
	b := make([]byte, hexChars/2)
	_, _ = crand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func fpMAC() string {
	b := make([]byte, 6)
	_, _ = crand.Read(b)
	b[0] = (b[0] & 0xFE) | 0x02
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

func fpRandInt(min, max int64) int64 {
	if min >= max {
		return min
	}
	return min + mrand.Int63n(max-min+1)
}

func fpDateMillis(s string) int64 {
	layouts := []string{"2006-01-02 15:04:05.000", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, locAsiaShanghai); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

func fpMD5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// fakePackageInfo 生成一条「看起来像真实系统包信息」的字符串：
// 签名MD5 + 大小 + "," + 版本名 + "," + 版本号 + "," + 安装时间 + "," + 更新时间差 + "," + inode
func fpFakePackageInfo() string {
	sig := fpHexChars(32)
	size := fpRandInt(20_000_000, 139_000_000)
	vers := []string{"13", "14", "14.0", "10", "11", "12"}
	ver := vers[mrand.Intn(len(vers))]
	verCode := fpRandInt(28, 34)
	install := fpRandInt(fpDateMillis("2021-01-01 00:00:00"), fpDateMillis("2024-06-01 00:00:00"))
	delta := fpRandInt(0, 90*86400000)
	inode := fpRandInt(10000, 909999)
	return fmt.Sprintf("%s%d,%s,%d,%d,%d,%d", sig, size, ver, verCode, install, delta, inode)
}

// f7_params：与 v/d.java 一致
func fpF7Params(vbmetaDigest string) string {
	m := map[string]any{
		"serialHexPool":    fpStringSlice(6, func() string { return fpHexChars(40) }),
		"snHexPool":        fpStringSlice(6, func() string { return fpHexChars(40) }),
		"snDecPool":        fpStringSlice(6, fpSnDec),
		"c1TimeOffsetDays": fpRandInt(-30, 30),
		"c2TimeOffsetDays": fpRandInt(-30, 30),
		"leafOffsetDays":   -fpRandInt(1, 30),
		"leafGapMs":        fpRandInt(10000, 59999),
		"vbh2":             vbmetaDigest,
		"dl":               "1",
		"vbs":              "0",
	}
	return jsonEncode(m)
}

func fpSnDec() string {
	b := make([]byte, 0, 22)
	b = append(b, byte(fpRandInt(49, 57))) // '1'..'9'
	for i := 1; i < 22; i++ {
		b = append(b, byte(fpRandInt(48, 57))) // '0'..'9'
	}
	return string(b)
}

func fpStringSlice(n int, gen func() string) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = gen()
	}
	return out
}

// generateFPEnv 返回完整的指纹 map（全部字符串值），不含 name（由调用方补）。
func generateFPEnv() map[string]string {
	nowMs := time.Now().UnixMilli()

	b5 := fpHexChars(16) // androidId
	str7 := "3" + fpHexChars(15)
	uuid := fpUUID()
	enc1 := fpBase64URLHex(32)
	enc2 := fpBase64URLHex(32)
	enc3 := fpBase64URLHex(32)
	enc4 := fpBase64URLHex(32)
	uuid2 := fpUUID()
	b6, b7, b8 := fpHexChars(32), fpHexChars(32), fpHexChars(32)
	b9 := fpMAC()
	b10, b11 := fpHexChars(32), fpHexChars(32)
	b12, b13, b14, b15, b16 := fpHexChars(64), fpHexChars(64), fpHexChars(32), fpHexChars(32), fpHexChars(40)
	b17, b18 := fpHexChars(16), fpHexChars(16)
	q := b17 + "-" + b18
	b19, b20, b21, b22, b23 := fpHexChars(32), fpHexChars(16), fpHexChars(32), fpHexChars(32), fpHexChars(32)

	// 10-digit random decimal
	sb2 := ""
	for i := 0; i < 10; i++ {
		sb2 += fmt.Sprintf("%d", fpRandInt(0, 9))
	}

	str8 := fmt.Sprintf("%d:%s", fpRandInt(26, 34), fpUUID()) // MediaStore version

	// l4: decimal of unsigned hex(b17)
	l4 := "0"
	if v, err := strconv.ParseUint(b17, 16, 64); err == nil {
		l4 = fmt.Sprintf("%d", v)
	}

	d4 := fpRandInt(nowMs-604800000, nowMs-172800000)
	valueOf := fmt.Sprintf("%d", d4)
	valueOf2 := fmt.Sprintf("%d", fpRandInt(fpDateMillis("2022-01-01 00:00:00"), fpDateMillis("2024-01-01 00:00:00")))
	valueOf3 := fmt.Sprintf("%d", fpRandInt(fpDateMillis("2024-01-01 00:00:00"), fpDateMillis("2025-09-01 00:00:00")))

	// totalMem
	memPool := []int64{1996505088, 2925522944, 3758096384, 5859708928, 7745568768, 12028280832}
	totalMem := memPool[mrand.Intn(len(memPool))] - fpRandInt(0, 255)*1048576

	// totalSize
	storagePool := []int64{31994675200, 63989350400, 127978700800, 255957401600, 511914803200}
	totalSize := storagePool[mrand.Intn(len(storagePool))] - fpRandInt(0, 511)*1048576
	remainSize := int64(float64(totalSize) * (mrand.Float64()*0.5 + 0.1))

	nextInt6 := fpRandInt(100, 199)
	i6 := []int64{15000, 30000, 60000}[mrand.Intn(3)]
	nextInt7 := fpRandInt(50, 149)
	nextInt8 := fpRandInt(1, 259200)
	nextInt9 := fpRandInt(300, 7499)
	nextInt10 := fpRandInt(300000000, 599999999)
	nextInt11 := fpRandInt(1000000, 1999999)
	str12 := []string{"mtp", "sec_charging", "mtp,sec_charging"}[mrand.Intn(3)]

	// 0x91
	d5 := fpRandInt(nowMs-86400000, nowMs-7200000)
	j6 := d5 / 1000
	elapsed := nowMs - d5
	l5 := fmt.Sprintf("{\n    \"elapsedRealtime\": \"%d\",\n    \"uptime\": \"%d\"\n}", elapsed, elapsed)
	valueOf4 := fmt.Sprintf("%d", elapsed+fpRandInt(100, 300))

	// Network (内网)
	sub := fmt.Sprintf("%d", fpRandInt(0, 255))
	aIP := "192.168." + sub + "." + fmt.Sprintf("%d", fpRandInt(0, 255))
	bGW := "192.168." + sub + ".1"

	eVal := fpRandInt(600000, 900000)
	d6 := d4 - fpRandInt(5000, 12000)
	d7 := fpRandInt(0, 10000) + d6
	d8 := fpRandInt(0, 10000) + d7

	str11 := fmt.Sprintf("0,%d", fpRandInt(100, 299))

	// vbmetaDigest（64 hex，随机）
	str5 := fpHexChars(64)

	f7str := fpF7Params(str5)

	// IPv6
	fmtv6 := fmt.Sprintf("fe80::%x:%x:%x:%x", fpRandInt(0, 65535), fpRandInt(0, 65535), fpRandInt(0, 65535), fpRandInt(0, 65535))

	// Network info string
	str3 := "0.0.0.0"
	if parts := strings.Split(aIP, "."); len(parts) == 4 {
		str3 = parts[0] + "." + parts[1] + "." + parts[2] + ".0"
	}
	t3 := fmt.Sprintf("[\"%s\"]",
		fmt.Sprintf("{InterfaceName: wlan0 LinkAddresses: [ %s/64,%s/24 ] DnsAddresses: [ /8.8.8.8,/1.1.1.1 ] Domains: null MTU: 0 ServerAddress: /%s TcpBufferSizes: 524288,1048576,8388608,524288,1048576,4194304 Routes: [ fe80::/64 -> :: wlan0 mtu 0,%s/24 -> 0.0.0.0 wlan0 mtu 0,0.0.0.0/0 -> %s wlan0 mtu 0 ]}", fmtv6, aIP, bGW, str3, bGW))

	// 0x105: device names
	names := []string{"新的开始", "灵光一现", "新的旅程", "下一章节", "全新起点", "隐藏宝藏", "午夜思绪", "明亮远方", "静默回声", "破土新生", "全新视野", "第二机会"}
	mrand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	e105 := make([]string, 4)
	for i := 0; i < 4; i++ {
		e105[i] = fmt.Sprintf("\"%d\": \"%s\"", i, names[i])
	}
	str13 := "{" + strings.Join(e105, ", ") + "}"

	// 0x106: 7-item stats
	ea := fpRandInt(0, 1)
	eb := fpRandInt(ea, 7)
	ec := fpRandInt(0, 7)
	ed := fpRandInt(0, 25)
	ee := fpRandInt(0, 1)
	ef := fpRandInt(ee, 7)
	eg := fpRandInt(0, 7)
	e106 := []string{
		fmt.Sprintf("\"0\": \"%d,7,%d,%d,-1,-1,-1,-1,-1\"", ea, eb, eb),
		fmt.Sprintf("\"1\": \"%d,7,%d,%d\"", fpRandInt(0, 7), fpRandInt(0, 7), fpRandInt(0, 7)),
		fmt.Sprintf("\"2\": \"0,7,%d,%d,-1,-1,-1,-1,-1,-1\"", ec, ec),
		fmt.Sprintf("\"3\": \"0,25,%d,%d,-1,%d,-1,-1,-1,-1\"", ed, ed, fpRandInt(0, 25)),
		fmt.Sprintf("\"4\": \"%d,7,%d,%d,-1\"", ee, ef, ef),
		fmt.Sprintf("\"5\": \"0,7,%d,%d,-1,-1,-1,-1\"", eg, eg),
		fmt.Sprintf("\"6\": \"%d,-1,-1,-1,-1,-1\"", fpRandInt(0, 7)),
	}
	str18 := "{" + strings.Join(e106, ", ") + "}"

	// 0x140/0x141
	d92 := fpRandInt(fpDateMillis("2024-01-01 00:00:00"), nowMs) / 1000
	ni142 := fpRandInt(0, 9) + d92
	str202 := fmt.Sprintf("%d,33188,0,0,%d,%d,%d,%d",
		fpRandInt(100, 1999), fpRandInt(300, 7999), fpRandInt(0, 86399)+ni142, ni142, d92)
	str212 := fpMD5Hex(str202 + b7)

	// 0x75
	path75 := "/data/app/~~" + enc4 + "/com.shopee.tw-" + enc3 + "/split_config.arm64_v8a.apk!libshpssdk.so"

	// Package info
	a4 := fpFakePackageInfo()
	a5 := fpFakePackageInfo()
	a6 := fpFakePackageInfo()

	// gsm
	nwType := "LTE,UNKNOWN"
	opAlpha := fmt.Sprintf("O%d,", fpRandInt(1, 3))

	// Public IP
	ip41 := fpPublicIP()

	// 0x14a: 48 random bytes -> 96 hex
	secB := make([]byte, 48)
	_, _ = crand.Read(secB)
	c14 := hex.EncodeToString(secB)

	m := map[string]string{
		// 静态字段（与 v/f.java a() 一致）
		"0x16": "false", "0x19": "false", "0x1a": "0", "0x1e": "false",
		"0x20": "false", "0x28": "false", "0x3d": "0efde111a560f04fb73e18fd00cc8afa",
		"0x46": "false", "0x49": "false", "0x4a": "false", "0x4b": "false",
		"0x6f": "false", "0x70": "false", "0x71": "false", "0x73": "0",
		"0x78": "[]", "0x79": "[]", "0x8e": "false", "0x96": "false",
		"0x97": "false", "0x98": "false", "0x99": "false", "0x9b": "0",
		"0xa5": "com.shopee.app.asm.anr.launch.ShopeeAppComponentFactory", "0xa6": "0",
		"0xd1": "{\"wifi_on\": \"1\", \"cell_on\": \"1\", \"bluetooth_on\": \"0\", \"development_settings_enabled\": \"0\", \"auto_time_zone\": \"1\", \"adb_wifi_enabled\": \"0\", \"adb_enabled\": \"0\"}",
		"0xc0": "0", "0xc1": "0", "0xc2": "0",
		"0x122": "[\"/apex/com.android.runtime/bin/linker64\",\"/apex/com.android.runtime/lib64/bionic/libdl.so\",\"[anon:bytehook-plt-trampolines]\",\"[anon:shadowhook-enter]\",\"[anon:shadowhook-exit]\",\"[anon:shadowhook-hub-trampo]\"]",
		"0x1f": "false", "0xab": "1", "0xb2": "false", "0xb5": "2",
		"0xb6": "0efde111a560f04fb73e18fd00cc8afa", "0xb7": "{\"cf\": 0, \"or\": 0}",
		"0xbd": "0", "0xe1": "false", "0xf6": "2,0",
		"0x11d": "000000000", "0x11e": "000000000", "0x121": "0", "0x123": "{}",
		"0x13e": "71919287", "0x13f": "7", "0x14c": "0",

		// 动态字段
		"androidId": b5, "gsfAndroidId": str7, "advertiserId": uuid,
		"vendingSourceDir": enc1, "gmsSourceDir": enc2,
		"keyECHashKey2": b18, "keyECNewId": q,
		"randomInstallShopeeTime": valueOf, "randomUpdateShopeeTime": valueOf,
		"randomInstallOthersTime": valueOf2, "randomUpdateOthersTime": valueOf3,
		"totalMem": fmt.Sprintf("%d", totalMem), "totalSize": fmt.Sprintf("%d", totalSize), "remainSize": fmt.Sprintf("%d", remainSize),
		"shopeeSourceDir": enc3, "apkSourceDir": enc4,
		"fix7cTime": fmt.Sprintf("%d", nextInt6), "screenOffTimeout": fmt.Sprintf("%d", i6),
		"bootCount": fmt.Sprintf("%d", nextInt7),
		"fixAEAccessTime": fmt.Sprintf("%d", nextInt8), "fixAEModifyTime": fmt.Sprintf("%d", nextInt9),
		"m1INodeRandom": fmt.Sprintf("%d", eVal), "vbmetaDigest": str5,
		"0x4": uuid2, "0x5": b6, "0x6": b7, "0x7": b8,
		"0x8": fmt.Sprintf("%d", d5), "0x1c": str12,
		"0x3e": b20, "0x39": b12, "0x40": aIP, "0x42": bGW, "0x43": b15,
		"0x44": "zh-Hant-TW", "0x59": "Asia/Taipei", "0x50": "1", "0x5d": "1", "0x62": "14",
		"0x63": fmt.Sprintf("%d", (fpRandInt(1, 4)*nextInt10)/10),
		"0x64": fmt.Sprintf("%d", (fpRandInt(1, 4)*nextInt11)/10),
		"0x65": fmt.Sprintf("%d", nextInt10), "0x66": fmt.Sprintf("%d", nextInt11),
		"0x67": "[]", "0x68": "[]", "0x6a": "0", "0x6d": "[]", "0x6c": "0",
		"0x75": path75, "0x91": l5,
		"0x9c": b9, "0xa4": b16, "0xa9": fmt.Sprintf("%d", d5),
		"0xaa": "-1", "0xac": "1", "0xaf": l4, "0xb3": sb2,
		"0xc3": fmt.Sprintf("%d", nextInt7), "0xdc": fmt.Sprintf("%d", j6),
		"0xe4": b10, "0xe5": b11, "0x102": b13,
		"0x105": str13, "0x106": str18, "0x10a": b14,
		"0x119": str8, "0x126": b19, "0x127": str11,
		"0x12a": b21, "0x128": fmtv6,
		"0x133": fpUUID(),
		"0x137": valueOf4, "0x138": valueOf4,
		"0x139": enc3, "0x13a": fmt.Sprintf("%d", eVal),
		"0x13b": fmt.Sprintf("%d", d6), "0x13c": fmt.Sprintf("%d", d8), "0x13d": fmt.Sprintf("%d", d7),
		"0x14a": c14,
		"0x140": str202, "0x141": str212,
		"0x146": t3, "0x152": a4, "0x153": a5, "0x154": a6,
		"0x165": b22, "0x166": b23, "0x2015": str5,
		"0xd3": "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}",
		"gsm.sim.state": "LOADED,ABSENT", "gsm.network.type": nwType,
		"gsm.operator.alpha": opAlpha, "gsm.operator.iso-country": "TW,",
		"0x1b": "LOADED,ABSENT", "0x2e": nwType, "0x3a": opAlpha,
		"0x41": ip41,
	}
	if f7str != "" {
		m["f7_params"] = f7str
	}
	return m
}

// fpPublicIP 生成非私有公网 IP
func fpPublicIP() string {
	for {
		na := int(fpRandInt(1, 223))
		nb := int(fpRandInt(0, 255))
		nc := int(fpRandInt(0, 255))
		nd := int(fpRandInt(0, 255))
		if na == 0 || na == 10 || na == 127 {
			continue
		}
		if na == 169 && nb == 254 {
			continue
		}
		if na == 172 && nb >= 16 && nb <= 31 {
			continue
		}
		if na == 192 && nb == 168 {
			continue
		}
		if na == 100 && nb >= 64 && nb <= 127 {
			continue
		}
		if na == 192 && nb == 0 && nc == 0 {
			continue
		}
		if na == 192 && nb == 0 && nc == 2 {
			continue
		}
		if na == 198 && (nb == 18 || nb == 19) {
			continue
		}
		if na == 198 && nb == 51 && nc == 100 {
			continue
		}
		if na == 203 && nb == 0 && nc == 113 {
			continue
		}
		if na >= 224 {
			continue
		}
		if na == 255 && nb == 255 && nc == 255 && nd == 255 {
			continue
		}
		return fmt.Sprintf("%d.%d.%d.%d", na, nb, nc, nd)
	}
}

// ==================== 额度 / 授权 ====================

func readFPEnvQuota(date string) map[string]any {
	m := map[string]any{}
	_ = readJSONFile(dp(fpEnvQuotaDir+"/"+date+".json"), &m)
	return m
}

func writeFPEnvQuota(date string, m map[string]any) {
	_ = os.MkdirAll(dp(fpEnvQuotaDir), 0o755)
	_ = atomicWrite(dp(fpEnvQuotaDir+"/"+date+".json"), jsonEncode(m))
}

// handleFPEnvGenerate 生成环境（指纹）：需账号已授权 + 未超当日额度
func handleFPEnvGenerate(w http.ResponseWriter, r *http.Request, reqCtx *httpRequestCtx, input map[string]any, pluginAccount map[string]any, pluginUsername string) {
	_ = r
	_ = input
	_ = reqCtx

	if !asBool(pluginAccount["fp_env_enabled"], false) {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "未授权生成环境，请联系管理员开通"})
		return
	}
	quota := qInt(pluginAccount["fp_env_daily_quota"])
	if quota <= 0 {
		decodedExtJSON(w, 403, map[string]any{"success": false, "error": "未设置每日环境额度，请联系管理员"})
		return
	}

	date := tsDate()
	fpEnvMu.Lock()
	defer fpEnvMu.Unlock()

	qm := readFPEnvQuota(date)
	entry, _ := qm[pluginUsername].(map[string]any)
	used := 0
	var records []any
	if entry != nil {
		used = qInt(entry["count"])
		records, _ = entry["records"].([]any)
	}
	if used >= quota {
		decodedExtJSON(w, 403, map[string]any{
			"success": false, "error": fmt.Sprintf("今日环境额度已用完（%d/%d）", used, quota),
			"data":    map[string]any{"quota": quota, "used": used, "remaining": 0},
		})
		return
	}

	fp := generateFPEnv()
	fp["name"] = "FP-" + now().Format("2006-01-02 15:04:05")

	used++
	records = append(records, map[string]any{
		"time":      tsDateTime(),
		"name":      fp["name"],
		"androidId": fp["androidId"],
	})
	if len(records) > 500 {
		records = records[len(records)-500:]
	}
	qm[pluginUsername] = map[string]any{"count": used, "quota": quota, "records": records}
	writeFPEnvQuota(date, qm)

	decodedExtJSON(w, 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"fp":        fp,
			"quota":     quota,
			"used":      used,
			"remaining": quota - used,
		},
	})
}
