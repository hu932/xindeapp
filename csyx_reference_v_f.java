package v;

import android.app.ActivityManager;
import android.content.Context;
import android.os.Environment;
import android.os.StatFs;
import android.provider.MediaStore;
import android.util.Log;
import androidx.compose.animation.core.AnimationConstants;
import androidx.compose.material3.internal.CalendarModelKt;
import androidx.lifecycle.CoroutineLiveDataKt;
import androidx.vectordrawable.graphics.drawable.PathInterpolatorCompat;
import com.app.helper.fp.Utils;
import com.blankj.utilcode.constant.CacheConstants;
import com.blankj.utilcode.constant.TimeConstants;
import java.security.MessageDigest;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Date;
import java.util.HashMap;
import java.util.Locale;
import java.util.Random;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;
import kotlin.UByte;
import kotlin.time.DurationKt;
import okhttp3.HttpUrl;
import org.json.JSONException;
import org.json.JSONObject;

/* loaded from: classes2.dex */
public abstract class f {

    /* renamed from: a, reason: collision with root package name */
    public static final HashMap f3709a = new HashMap();

    /* renamed from: b, reason: collision with root package name */
    public static final Random f3710b = new Random();

    /* renamed from: c, reason: collision with root package name */
    public static final char[] f3711c = "0123456789abcdef".toCharArray();

    public static void a() {
        HashMap hashMap = f3709a;
        hashMap.put("0x16", "false");
        hashMap.put("0x19", "false");
        hashMap.put("0x1a", "0");
        hashMap.put("0x1e", "false");
        hashMap.put("0x20", "false");
        hashMap.put("0x28", "false");
        hashMap.put("0x3d", "0efde111a560f04fb73e18fd00cc8afa");
        hashMap.put("0x46", "false");
        hashMap.put("0x49", "false");
        hashMap.put("0x4a", "false");
        hashMap.put("0x4b", "false");
        hashMap.put("0x6f", "false");
        hashMap.put("0x70", "false");
        hashMap.put("0x71", "false");
        hashMap.put("0x73", "0");
        hashMap.put("0x78", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
        hashMap.put("0x79", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
        hashMap.put("0x8e", "false");
        hashMap.put("0x96", "false");
        hashMap.put("0x97", "false");
        hashMap.put("0x98", "false");
        hashMap.put("0x99", "false");
        hashMap.put("0x9b", "0");
        hashMap.put("0xa5", "com.shopee.app.asm.anr.launch.ShopeeAppComponentFactory");
        hashMap.put("0xa6", "0");
        hashMap.put("0xd1", "{\"wifi_on\": \"1\", \"cell_on\": \"1\", \"bluetooth_on\": \"0\", \"development_settings_enabled\": \"0\", \"auto_time_zone\": \"1\", \"adb_wifi_enabled\": \"0\", \"adb_enabled\": \"0\"}");
        hashMap.put("0xc0", "0");
        hashMap.put("0xc1", "0");
        hashMap.put("0xc2", "0");
        hashMap.put("0x122", "[\"/apex/com.android.runtime/bin/linker64\",\"/apex/com.android.runtime/lib64/bionic/libdl.so\",\"[anon:bytehook-plt-trampolines]\",\"[anon:shadowhook-enter]\",\"[anon:shadowhook-exit]\",\"[anon:shadowhook-hub-trampo]\"]");
        hashMap.put("0x1f", "false");
        hashMap.put("0xab", "1");
        hashMap.put("0xb2", "false");
        hashMap.put("0xb5", "2");
        hashMap.put("0xb6", "0efde111a560f04fb73e18fd00cc8afa");
        hashMap.put("0xb7", "{\"cf\": 0, \"or\": 0}");
        hashMap.put("0xbd", "0");
        hashMap.put("0xe1", "false");
        hashMap.put("0xf6", "2,0");
        hashMap.put("0x11d", "000000000");
        hashMap.put("0x11e", "000000000");
        hashMap.put("0x121", "0");
        hashMap.put("0x123", "{}");
        hashMap.put("0x13e", "71919287");
        hashMap.put("0x13f", "7");
        hashMap.put("0x14c", "0");
    }

    public static String b(int i) {
        StringBuilder sb = new StringBuilder(i);
        for (int i4 = 0; i4 < i; i4++) {
            sb.append(Integer.toHexString(f3710b.nextInt(16)));
        }
        return sb.toString();
    }

    /* JADX WARN: Can't wrap try/catch for region: R(41:0|1|(2:4|2)|5|6|7|8|9|(1:11)|12|2cb|20|2dc|28|(1:30)|31|(1:33)(1:214)|34|(10:35|36|(1:38)|39|40|41|(2:43|44)|207|208|209)|48|49|50|(3:198|199|(17:201|(1:56)|58|59|60|(3:62|(2:64|65)(1:67)|66)|68|69|70|(1:72)|73|74|(2:76|(1:(1:79)(1:191))(1:192))(1:193)|80|(9:81|(1:190)(2:83|(6:183|102|103|104|105|106)(1:(1:180)(8:92|(2:107|(6:119|(3:(1:177)(2:132|(1:176))|134|(6:(1:172)(2:144|(4:171|152|153|106))|146|(3:156|157|(1:1)(2:166|106))|152|153|106))(1:124)|125|104|105|106)(5:113|103|104|105|106))|101|102|103|104|105|106)))|181|101|102|103|104|105|106)|163|164))|52|(2:54|56)|58|59|60|(0)|68|69|70|(0)|73|74|(0)(0)|80|(9:81|(0)(0)|181|101|102|103|104|105|106)|163|164|(1:(0))) */
    /* JADX WARN: Code restructure failed: missing block: B:195:0x0816, code lost:
    
        r0 = move-exception;
     */
    /* JADX WARN: Code restructure failed: missing block: B:197:0x0823, code lost:
    
        r0.printStackTrace();
        r0 = "";
     */
    /* JADX WARN: Removed duplicated region for block: B:190:0x0c9f A[SYNTHETIC] */
    /* JADX WARN: Removed duplicated region for block: B:193:0x0b8d  */
    /* JADX WARN: Removed duplicated region for block: B:198:0x0722 A[EXC_TOP_SPLITTER, SYNTHETIC] */
    /* JADX WARN: Removed duplicated region for block: B:54:0x0738 A[Catch: all -> 0x0746, TryCatch #2 {all -> 0x0746, blocks: (B:50:0x0719, B:54:0x0738, B:56:0x073e, B:199:0x0722, B:201:0x0728, B:204:0x072e), top: B:49:0x0719, inners: #7 }] */
    /* JADX WARN: Removed duplicated region for block: B:62:0x0802 A[Catch: Exception -> 0x0816, TryCatch #4 {Exception -> 0x0816, blocks: (B:60:0x07e6, B:62:0x0802, B:64:0x0812, B:66:0x0818, B:69:0x081e), top: B:59:0x07e6 }] */
    /* JADX WARN: Removed duplicated region for block: B:72:0x0ad6 A[LOOP:4: B:71:0x0ad4->B:72:0x0ad6, LOOP_END] */
    /* JADX WARN: Removed duplicated region for block: B:76:0x0b7e  */
    /* JADX WARN: Removed duplicated region for block: B:83:0x0bda  */
    /*
        Code decompiled incorrectly, please refer to instructions dump.
    */
    public static void c(Context context) {
        long j4;
        String str;
        String str2;
        int i;
        String str3;
        String str4;
        String str5;
        int i4;
        Random random;
        int nextInt;
        int nextInt2;
        int nextInt3;
        int nextInt4;
        c cVar;
        JSONObject b4;
        String jSONObject;
        String b5 = b(16);
        String str6 = e.f3707a;
        String str7 = "3" + b(15);
        String uuid = UUID.randomUUID().toString();
        String encodeToString = Base64.getUrlEncoder().encodeToString(Utils.a(b(32)));
        String encodeToString2 = Base64.getUrlEncoder().encodeToString(Utils.a(b(32)));
        String encodeToString3 = Base64.getUrlEncoder().encodeToString(Utils.a(b(32)));
        String encodeToString4 = Base64.getUrlEncoder().encodeToString(Utils.a(b(32)));
        String uuid2 = UUID.randomUUID().toString();
        String b6 = b(32);
        String b7 = b(32);
        String b8 = b(32);
        String b9 = e.b();
        String b10 = b(32);
        String b11 = b(32);
        StringBuilder sb = new StringBuilder(10);
        for (int i5 = 0; i5 <= 9; i5++) {
            sb.append(f3710b.nextInt(10));
        }
        String sb2 = sb.toString();
        String b12 = b(64);
        String b13 = b(64);
        String b14 = b(32);
        String str8 = MediaStore.getVersion(context).split(":")[0] + ":" + UUID.randomUUID();
        String b15 = b(32);
        String b16 = b(40);
        String b17 = b(16);
        String b18 = b(16);
        String q = androidx.compose.foundation.c.q(b17, "-", b18);
        try {
            j4 = Long.parseUnsignedLong(b17, 16);
        } catch (Exception unused) {
            j4 = 0;
        }
        String l4 = Long.toString(j4);
        long currentTimeMillis = System.currentTimeMillis();
        long d4 = d(currentTimeMillis - 604800000, currentTimeMillis - 172800000);
        String valueOf = String.valueOf(d4);
        String valueOf2 = String.valueOf(d(f("2022-01-01 00:00:00"), f("2024-01-01 00:00:00")));
        String valueOf3 = String.valueOf(d(f("2024-01-01 00:00:00"), f("2025-09-01 00:00:00")));
        ActivityManager.MemoryInfo memoryInfo = new ActivityManager.MemoryInfo();
        ActivityManager activityManager = (ActivityManager) context.getSystemService("activity");
        if (activityManager != null) {
            activityManager.getMemoryInfo(memoryInfo);
        }
        long j5 = memoryInfo.totalMem;
        Random random2 = f3710b;
        long nextInt5 = j5 - (random2.nextInt(100) * 1048576);
        StatFs statFs = new StatFs(Environment.getExternalStorageDirectory().getAbsolutePath());
        long blockCountLong = (statFs.getBlockCountLong() * statFs.getBlockSizeLong()) - (random2.nextInt(100) * 1048576);
        long nextDouble = (long) (blockCountLong * ((random2.nextDouble() * 0.5d) + 0.1d));
        int nextInt6 = random2.nextInt(100) + 100;
        int i6 = new int[]{15000, 30000, TimeConstants.MIN}[random2.nextInt(3)];
        int nextInt7 = random2.nextInt(100) + 50;
        String str9 = e.f3707a;
        int nextInt8 = random2.nextInt(259200) + 1;
        int nextInt9 = random2.nextInt(7200) + AnimationConstants.DefaultDurationMillis;
        String str10 = new String[]{"LOADED,LOADED", "LOADED,ABSENT", "ABSENT,LOADED"}[random2.nextInt(3)];
        int nextInt10 = random2.nextInt(300000000) + 300000000;
        int nextInt11 = random2.nextInt(DurationKt.NANOS_IN_MILLIS) + DurationKt.NANOS_IN_MILLIS;
        String b19 = b(32);
        String str11 = "0," + (random2.nextInt(200) + 100);
        String str12 = new String[]{"mtp", "sec_charging", "mtp,sec_charging"}[random2.nextInt(3)];
        long d5 = d(f(g(System.currentTimeMillis() - CalendarModelKt.MillisecondsIn24Hours)), f(g(System.currentTimeMillis() - 7200000)));
        long j6 = d5 / 1000;
        long currentTimeMillis2 = System.currentTimeMillis() - d5;
        StringBuilder sb3 = new StringBuilder("{\n    \"elapsedRealtime\": \"");
        sb3.append(currentTimeMillis2);
        sb3.append("\",\n    \"uptime\": \"");
        String l5 = androidx.constraintlayout.core.a.l(currentTimeMillis2, "\"\n}", sb3);
        String valueOf4 = String.valueOf(currentTimeMillis2 + e(100, AnimationConstants.DefaultDurationMillis));
        e.a();
        synchronized (e.class) {
            try {
                if (e.f3707a == null) {
                    e.a();
                }
                str = e.f3707a;
            } finally {
            }
        }
        synchronized (e.class) {
            try {
                if (e.f3708b == null) {
                    e.a();
                }
                str2 = e.f3708b;
            } finally {
            }
        }
        int e = e(600000, 900000);
        long d6 = d4 - d(CoroutineLiveDataKt.DEFAULT_TIMEOUT, 12000L);
        long d7 = d(0L, 10000L) + d6;
        long d8 = d(0L, 10000L) + d7;
        e(100, 1000);
        e(10110, 12330);
        String l6 = Long.toString(f(new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).format(new Date(d(f("2025-01-01 08:00:00"), f("2025-09-01 08:00:00")))).substring(0, 10) + " 08:00:00"));
        if (l6.length() > 3) {
            l6.substring(0, l6.length() - 3);
        }
        UUID randomUUID = UUID.randomUUID();
        String format = String.format("fe80::%x:%x:%x:%x", Integer.valueOf(e(0, 65535)), Integer.valueOf(e(0, 65535)), Integer.valueOf(e(0, 65535)), Integer.valueOf(e(0, 65535)));
        String[] split = str.split("\\.");
        if (split.length == 4) {
            StringBuilder sb4 = new StringBuilder();
            i = 2;
            sb4.append(split[0]);
            sb4.append(".");
            sb4.append(split[1]);
            sb4.append(".");
            str3 = androidx.compose.foundation.c.t(sb4, split[2], ".0");
        } else {
            i = 2;
            str3 = "0.0.0.0";
        }
        StringBuilder sb5 = new StringBuilder("[\"{InterfaceName: wlan0 LinkAddresses: [ ");
        sb5.append(format);
        sb5.append("/64,");
        sb5.append(str);
        sb5.append("/24 ] DnsAddresses: [ /8.8.8.8,/1.1.1.1 ] Domains: null MTU: 0 ServerAddress: /");
        sb5.append(str2);
        sb5.append(" TcpBufferSizes: 524288,1048576,8388608,524288,1048576,4194304 Routes: [ fe80::/64 -> :: wlan0 mtu 0,");
        sb5.append(str3);
        sb5.append("/24 -> 0.0.0.0 wlan0 mtu 0,0.0.0.0/0 -> ");
        String t3 = androidx.compose.foundation.c.t(sb5, str2, " wlan0 mtu 0 ]}\"]");
        ArrayList arrayList = new ArrayList();
        arrayList.add("Fresh Start");
        arrayList.add("Eureka");
        arrayList.add("Your New Adventure");
        arrayList.add("Next Chapter");
        arrayList.add("New Beginnings");
        arrayList.add("Hidden Treasures");
        arrayList.add("Midnight Thoughts");
        arrayList.add("Bright Horizons");
        arrayList.add("Silent Echoes");
        arrayList.add("Breaking Ground");
        arrayList.add("New Horizons");
        arrayList.add("Second Chance");
        kotlin.streams.jdk8.a aVar = new kotlin.streams.jdk8.a(arrayList, 1);
        ArrayList arrayList2 = new ArrayList();
        arrayList2.add("\"0\": \"" + ((String) aVar.get()) + "\"");
        arrayList2.add("\"1\": \"" + ((String) aVar.get()) + "\"");
        arrayList2.add("\"2\": \"" + ((String) aVar.get()) + "\"");
        arrayList2.add("\"3\": \"" + ((String) aVar.get()) + "\"");
        String str13 = "{" + String.join(", ", arrayList2) + "}";
        int e4 = e(0, 1);
        int e5 = e(e4, 7);
        String str14 = e4 + ",7," + e5 + "," + e5 + ",-1,-1,-1,-1,-1";
        String m = androidx.compose.foundation.c.m(e(0, 7), "0,7,", ",", e(0, 7));
        int e6 = e(0, 7);
        int e7 = e(0, 25);
        String s3 = androidx.compose.foundation.c.s(androidx.activity.result.d.t("0,25,", e7, e7, ",", ",-1,"), ",-1,-1,-1,-1", e(0, 25));
        int e8 = e(0, 1);
        int e9 = e(e8, 7);
        String str15 = e8 + ",7," + e9 + "," + e9 + ",-1";
        int e10 = e(0, 7);
        String str16 = "0,7," + e10 + "," + e10 + ",-1,-1,-1,-1";
        String str17 = e(0, 7) + ",-1,-1,-1,-1,-1";
        ArrayList arrayList3 = new ArrayList();
        arrayList3.add("\"0\": \"" + str14 + "\"");
        arrayList3.add("\"1\": \"" + m + "\"");
        arrayList3.add("\"2\": \"" + ("0,7," + e6 + "," + e6 + ",-1,-1,-1,-1,-1,-1") + "\"");
        arrayList3.add("\"3\": \"" + s3 + "\"");
        arrayList3.add("\"4\": \"" + str15 + "\"");
        arrayList3.add("\"5\": \"" + str16 + "\"");
        arrayList3.add("\"6\": \"" + str17 + "\"");
        String str18 = "{" + String.join(", ", arrayList3) + "}";
        String b20 = b(16);
        String b21 = b(32);
        String a4 = b.a(context, "com.android.systemui");
        String a5 = b.a(context, "com.android.settings");
        String a6 = b.a(context, "android");
        String b22 = b(32);
        String b23 = b(32);
        try {
            char[] charArray = b5.toCharArray();
            int i7 = 0;
            long j7 = 0;
            for (int length = charArray.length; i7 < length; length = length) {
                j7 = (j7 * 31) + (r87[i7] & 255);
                i7++;
                charArray = charArray;
            }
            Random random3 = new Random(j7 & 4294967295L);
            StringBuilder sb6 = new StringBuilder(64);
            str4 = b23;
            int i8 = 0;
            for (int i9 = 64; i8 < i9; i9 = 64) {
                try {
                    sb6.append(f3711c[random3.nextInt(16)]);
                    i8++;
                } catch (Throwable th) {
                    th = th;
                    th.printStackTrace();
                    str5 = null;
                    String str19 = e.f3707a;
                    d.c(b5, str5);
                    cVar = d.f3706a;
                    if (cVar != null) {
                    }
                    jSONObject = null;
                    if (jSONObject != null) {
                    }
                    int nextInt12 = ThreadLocalRandom.current().nextInt(100, 2000);
                    int nextInt13 = ThreadLocalRandom.current().nextInt(PathInterpolatorCompat.MAX_NUM_POINTS, 8000);
                    long d9 = d(f("2024-01-01 00:00:00"), f(g(System.currentTimeMillis()))) / 1000;
                    long nextInt14 = ThreadLocalRandom.current().nextInt(0, 10) + d9;
                    String str20 = nextInt12 + ",33188,0,0," + nextInt13 + "," + (ThreadLocalRandom.current().nextInt(0, CacheConstants.DAY) + nextInt14) + "," + nextInt14 + "," + d9;
                    byte[] digest = MessageDigest.getInstance("MD5").digest(androidx.compose.foundation.c.p(str20, b7).getBytes("UTF-8"));
                    StringBuilder sb7 = new StringBuilder(digest.length * 2);
                    while (r8 < r7) {
                    }
                    String str21 = sb7.toString();
                    HashMap hashMap = f3709a;
                    hashMap.put("androidId", b5);
                    hashMap.put("gsfAndroidId", str7);
                    hashMap.put("advertiserId", uuid);
                    hashMap.put("vendingSourceDir", encodeToString);
                    hashMap.put("gmsSourceDir", encodeToString2);
                    hashMap.put("keyECHashKey2", b18);
                    hashMap.put("keyECNewId", q);
                    hashMap.put("randomInstallShopeeTime", valueOf);
                    hashMap.put("randomUpdateShopeeTime", valueOf);
                    hashMap.put("randomInstallOthersTime", valueOf2);
                    hashMap.put("randomUpdateOthersTime", valueOf3);
                    hashMap.put("totalMem", String.valueOf(nextInt5));
                    hashMap.put("totalSize", String.valueOf(blockCountLong));
                    hashMap.put("remainSize", String.valueOf(nextDouble));
                    hashMap.put("shopeeSourceDir", String.valueOf(encodeToString3));
                    hashMap.put("apkSourceDir", String.valueOf(encodeToString4));
                    hashMap.put("fix7cTime", String.valueOf(nextInt6));
                    hashMap.put("screenOffTimeout", String.valueOf(i6));
                    hashMap.put("bootCount", String.valueOf(nextInt7));
                    hashMap.put("fixAEAccessTime", String.valueOf(nextInt8));
                    hashMap.put("fixAEModifyTime", String.valueOf(nextInt9));
                    hashMap.put("m1INodeRandom", String.valueOf(e));
                    hashMap.put("vbmetaDigest", str5);
                    hashMap.put("0x4", uuid2);
                    hashMap.put("0x5", b6);
                    hashMap.put("0x6", b7);
                    hashMap.put("0x7", b8);
                    hashMap.put("0x8", String.valueOf(d5));
                    hashMap.put("0x1c", str12);
                    hashMap.put("0x3e", b20);
                    hashMap.put("0x39", b12);
                    hashMap.put("0x40", str);
                    hashMap.put("0x42", str2);
                    hashMap.put("0x43", b15);
                    hashMap.put("0x44", "zh-Hant-TW");
                    hashMap.put("0x59", "Asia/Taipei");
                    hashMap.put("0x50", "1");
                    hashMap.put("0x5d", "1");
                    hashMap.put("0x62", "14");
                    hashMap.put("0x63", "" + ((e(1, 4) * nextInt10) / 10));
                    hashMap.put("0x64", "" + ((e(1, 4) * nextInt11) / 10));
                    hashMap.put("0x65", "" + nextInt10);
                    hashMap.put("0x66", "" + nextInt11);
                    hashMap.put("0x67", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                    hashMap.put("0x68", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                    hashMap.put("0x6a", "0");
                    hashMap.put("0x6d", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                    hashMap.put("0x6c", "0");
                    hashMap.put("0x75", "/data/app/~~" + encodeToString4 + "/com.shopee.tw-" + encodeToString3 + "/split_config.arm64_v8a.apk!libshpssdk.so");
                    hashMap.put("0x91", l5);
                    hashMap.put("0x9c", b9);
                    hashMap.put("0xa4", b16);
                    hashMap.put("0xa9", String.valueOf(d5));
                    hashMap.put("0xaa", "-1");
                    hashMap.put("0xac", "1");
                    hashMap.put("0xaf", l4);
                    hashMap.put("0xb3", sb2);
                    hashMap.put("0xc3", String.valueOf(nextInt7));
                    hashMap.put("0xdc", String.valueOf(j6));
                    hashMap.put("0xe4", b10);
                    hashMap.put("0xe5", b11);
                    hashMap.put("0x102", b13);
                    hashMap.put("0x105", str13);
                    hashMap.put("0x106", str18);
                    hashMap.put("0x10a", b14);
                    hashMap.put("0x119", str8);
                    hashMap.put("0x126", b19);
                    hashMap.put("0x127", str11);
                    hashMap.put("0x12a", b21);
                    hashMap.put("0x128", format);
                    hashMap.put("0x133", String.valueOf(randomUUID));
                    hashMap.put("0x137", valueOf4);
                    hashMap.put("0x138", valueOf4);
                    hashMap.put("0x139", encodeToString3);
                    hashMap.put("0x13a", String.valueOf(e));
                    hashMap.put("0x13b", String.valueOf(d6));
                    hashMap.put("0x13c", String.valueOf(d8));
                    hashMap.put("0x13d", String.valueOf(d7));
                    byte[] bArr = new byte[48];
                    i.f3715b.nextBytes(bArr);
                    char[] cArr = new char[96];
                    while (i4 < 48) {
                    }
                    hashMap.put("0x14a", new String(cArr));
                    HashMap hashMap2 = f3709a;
                    hashMap2.put("0x140", str20);
                    hashMap2.put("0x141", str21);
                    hashMap2.put("0x146", t3);
                    hashMap2.put("0x152", a4);
                    hashMap2.put("0x153", a5);
                    hashMap2.put("0x154", a6);
                    hashMap2.put("0x165", b22);
                    hashMap2.put("0x166", str4);
                    hashMap2.put("0x2015", str5);
                    hashMap2.put("0xd3", "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}");
                    random = h.f3713a;
                    String str22 = g.b(new int[]{i}[random.nextInt(1)]) + "," + g.b(4);
                    int i10 = g.c(3)[random.nextInt(g.c(3).length)];
                    String concat = (i10 != 1 ? i10 != i ? i10 != 3 ? "null" : "O3" : "O2" : "O1").concat(",");
                    String concat2 = "TW".concat(",");
                    hashMap2.put("gsm.sim.state", "LOADED,ABSENT");
                    hashMap2.put("gsm.network.type", str22);
                    hashMap2.put("gsm.operator.alpha", concat);
                    hashMap2.put("gsm.operator.iso-country", concat2);
                    hashMap2.put("0x1b", "LOADED,ABSENT");
                    hashMap2.put("0x2e", str22);
                    hashMap2.put("0x3a", concat);
                    while (true) {
                        nextInt = random.nextInt(223) + 1;
                        nextInt2 = random.nextInt(256);
                        nextInt3 = random.nextInt(256);
                        nextInt4 = random.nextInt(256);
                        if (nextInt >= 0) {
                        }
                    }
                    hashMap2.put("0x41", nextInt + "." + nextInt2 + "." + nextInt3 + "." + nextInt4);
                }
            }
            str5 = sb6.toString();
        } catch (Throwable th2) {
            th = th2;
            str4 = b23;
        }
        String str192 = e.f3707a;
        try {
            d.c(b5, str5);
            cVar = d.f3706a;
        } catch (Throwable th3) {
            Log.e("mockfpenv", "F7Mocker.prepareParams failed: " + th3.getMessage());
        }
        if (cVar != null) {
            try {
                b4 = d.b(cVar);
            } catch (JSONException e11) {
                Log.e("F7Modifier", "dumpParams failed", e11);
            }
            if (b4 != null) {
                jSONObject = b4.toString();
                if (jSONObject != null && !jSONObject.isEmpty()) {
                    f3709a.put("f7_params", jSONObject);
                }
                int nextInt122 = ThreadLocalRandom.current().nextInt(100, 2000);
                int nextInt132 = ThreadLocalRandom.current().nextInt(PathInterpolatorCompat.MAX_NUM_POINTS, 8000);
                long d92 = d(f("2024-01-01 00:00:00"), f(g(System.currentTimeMillis()))) / 1000;
                long nextInt142 = ThreadLocalRandom.current().nextInt(0, 10) + d92;
                String str202 = nextInt122 + ",33188,0,0," + nextInt132 + "," + (ThreadLocalRandom.current().nextInt(0, CacheConstants.DAY) + nextInt142) + "," + nextInt142 + "," + d92;
                byte[] digest2 = MessageDigest.getInstance("MD5").digest(androidx.compose.foundation.c.p(str202, b7).getBytes("UTF-8"));
                StringBuilder sb72 = new StringBuilder(digest2.length * 2);
                for (byte b24 : digest2) {
                    String hexString = Integer.toHexString(b24 & UByte.MAX_VALUE);
                    if (hexString.length() == 1) {
                        sb72.append('0');
                    }
                    sb72.append(hexString);
                }
                String str212 = sb72.toString();
                HashMap hashMap3 = f3709a;
                hashMap3.put("androidId", b5);
                hashMap3.put("gsfAndroidId", str7);
                hashMap3.put("advertiserId", uuid);
                hashMap3.put("vendingSourceDir", encodeToString);
                hashMap3.put("gmsSourceDir", encodeToString2);
                hashMap3.put("keyECHashKey2", b18);
                hashMap3.put("keyECNewId", q);
                hashMap3.put("randomInstallShopeeTime", valueOf);
                hashMap3.put("randomUpdateShopeeTime", valueOf);
                hashMap3.put("randomInstallOthersTime", valueOf2);
                hashMap3.put("randomUpdateOthersTime", valueOf3);
                hashMap3.put("totalMem", String.valueOf(nextInt5));
                hashMap3.put("totalSize", String.valueOf(blockCountLong));
                hashMap3.put("remainSize", String.valueOf(nextDouble));
                hashMap3.put("shopeeSourceDir", String.valueOf(encodeToString3));
                hashMap3.put("apkSourceDir", String.valueOf(encodeToString4));
                hashMap3.put("fix7cTime", String.valueOf(nextInt6));
                hashMap3.put("screenOffTimeout", String.valueOf(i6));
                hashMap3.put("bootCount", String.valueOf(nextInt7));
                hashMap3.put("fixAEAccessTime", String.valueOf(nextInt8));
                hashMap3.put("fixAEModifyTime", String.valueOf(nextInt9));
                hashMap3.put("m1INodeRandom", String.valueOf(e));
                hashMap3.put("vbmetaDigest", str5);
                hashMap3.put("0x4", uuid2);
                hashMap3.put("0x5", b6);
                hashMap3.put("0x6", b7);
                hashMap3.put("0x7", b8);
                hashMap3.put("0x8", String.valueOf(d5));
                hashMap3.put("0x1c", str12);
                hashMap3.put("0x3e", b20);
                hashMap3.put("0x39", b12);
                hashMap3.put("0x40", str);
                hashMap3.put("0x42", str2);
                hashMap3.put("0x43", b15);
                hashMap3.put("0x44", "zh-Hant-TW");
                hashMap3.put("0x59", "Asia/Taipei");
                hashMap3.put("0x50", "1");
                hashMap3.put("0x5d", "1");
                hashMap3.put("0x62", "14");
                hashMap3.put("0x63", "" + ((e(1, 4) * nextInt10) / 10));
                hashMap3.put("0x64", "" + ((e(1, 4) * nextInt11) / 10));
                hashMap3.put("0x65", "" + nextInt10);
                hashMap3.put("0x66", "" + nextInt11);
                hashMap3.put("0x67", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                hashMap3.put("0x68", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                hashMap3.put("0x6a", "0");
                hashMap3.put("0x6d", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
                hashMap3.put("0x6c", "0");
                hashMap3.put("0x75", "/data/app/~~" + encodeToString4 + "/com.shopee.tw-" + encodeToString3 + "/split_config.arm64_v8a.apk!libshpssdk.so");
                hashMap3.put("0x91", l5);
                hashMap3.put("0x9c", b9);
                hashMap3.put("0xa4", b16);
                hashMap3.put("0xa9", String.valueOf(d5));
                hashMap3.put("0xaa", "-1");
                hashMap3.put("0xac", "1");
                hashMap3.put("0xaf", l4);
                hashMap3.put("0xb3", sb2);
                hashMap3.put("0xc3", String.valueOf(nextInt7));
                hashMap3.put("0xdc", String.valueOf(j6));
                hashMap3.put("0xe4", b10);
                hashMap3.put("0xe5", b11);
                hashMap3.put("0x102", b13);
                hashMap3.put("0x105", str13);
                hashMap3.put("0x106", str18);
                hashMap3.put("0x10a", b14);
                hashMap3.put("0x119", str8);
                hashMap3.put("0x126", b19);
                hashMap3.put("0x127", str11);
                hashMap3.put("0x12a", b21);
                hashMap3.put("0x128", format);
                hashMap3.put("0x133", String.valueOf(randomUUID));
                hashMap3.put("0x137", valueOf4);
                hashMap3.put("0x138", valueOf4);
                hashMap3.put("0x139", encodeToString3);
                hashMap3.put("0x13a", String.valueOf(e));
                hashMap3.put("0x13b", String.valueOf(d6));
                hashMap3.put("0x13c", String.valueOf(d8));
                hashMap3.put("0x13d", String.valueOf(d7));
                byte[] bArr2 = new byte[48];
                i.f3715b.nextBytes(bArr2);
                char[] cArr2 = new char[96];
                for (i4 = 0; i4 < 48; i4++) {
                    byte b25 = bArr2[i4];
                    int i11 = b25 & UByte.MAX_VALUE;
                    int i12 = i4 * 2;
                    char[] cArr3 = i.f3716c;
                    cArr2[i12] = cArr3[i11 >>> 4];
                    cArr2[i12 + 1] = cArr3[b25 & 15];
                }
                hashMap3.put("0x14a", new String(cArr2));
                HashMap hashMap22 = f3709a;
                hashMap22.put("0x140", str202);
                hashMap22.put("0x141", str212);
                hashMap22.put("0x146", t3);
                hashMap22.put("0x152", a4);
                hashMap22.put("0x153", a5);
                hashMap22.put("0x154", a6);
                hashMap22.put("0x165", b22);
                hashMap22.put("0x166", str4);
                hashMap22.put("0x2015", str5);
                hashMap22.put("0xd3", "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}");
                random = h.f3713a;
                String str222 = g.b(new int[]{i}[random.nextInt(1)]) + "," + g.b(4);
                int i102 = g.c(3)[random.nextInt(g.c(3).length)];
                String concat3 = (i102 != 1 ? i102 != i ? i102 != 3 ? "null" : "O3" : "O2" : "O1").concat(",");
                String concat22 = "TW".concat(",");
                hashMap22.put("gsm.sim.state", "LOADED,ABSENT");
                hashMap22.put("gsm.network.type", str222);
                hashMap22.put("gsm.operator.alpha", concat3);
                hashMap22.put("gsm.operator.iso-country", concat22);
                hashMap22.put("0x1b", "LOADED,ABSENT");
                hashMap22.put("0x2e", str222);
                hashMap22.put("0x3a", concat3);
                while (true) {
                    nextInt = random.nextInt(223) + 1;
                    nextInt2 = random.nextInt(256);
                    nextInt3 = random.nextInt(256);
                    nextInt4 = random.nextInt(256);
                    if (nextInt >= 0) {
                        if (nextInt <= 255 && nextInt2 >= 0 && nextInt2 <= 255 && nextInt3 >= 0 && nextInt3 <= 255 && nextInt4 >= 0 && nextInt4 <= 255) {
                            if (nextInt != 0) {
                                if (nextInt != 10 && nextInt != 127 && (nextInt != 169 || nextInt2 != 254)) {
                                    if ((nextInt != 172 || nextInt2 < 16 || nextInt2 > 31) && (nextInt != 192 || nextInt2 != 168)) {
                                        if ((nextInt != 100 || nextInt2 < 64 || nextInt2 > 127) && (nextInt != 192 || nextInt2 != 0 || nextInt3 != 0)) {
                                            if (nextInt == 192 && nextInt2 == 0) {
                                                if (nextInt3 == 2) {
                                                }
                                            }
                                            if (nextInt != 198 || (nextInt2 != 18 && nextInt2 != 19)) {
                                                if (nextInt == 198 && nextInt2 == 51) {
                                                    if (nextInt3 == 100) {
                                                    }
                                                }
                                                if ((nextInt != 203 || nextInt2 != 0 || nextInt3 != 113) && nextInt < 224) {
                                                    if (nextInt != 255 || nextInt2 != 255 || nextInt3 != 255 || nextInt4 != 255) {
                                                        break;
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                hashMap22.put("0x41", nextInt + "." + nextInt2 + "." + nextInt3 + "." + nextInt4);
            }
        }
        jSONObject = null;
        if (jSONObject != null) {
            f3709a.put("f7_params", jSONObject);
        }
        int nextInt1222 = ThreadLocalRandom.current().nextInt(100, 2000);
        int nextInt1322 = ThreadLocalRandom.current().nextInt(PathInterpolatorCompat.MAX_NUM_POINTS, 8000);
        long d922 = d(f("2024-01-01 00:00:00"), f(g(System.currentTimeMillis()))) / 1000;
        long nextInt1422 = ThreadLocalRandom.current().nextInt(0, 10) + d922;
        String str2022 = nextInt1222 + ",33188,0,0," + nextInt1322 + "," + (ThreadLocalRandom.current().nextInt(0, CacheConstants.DAY) + nextInt1422) + "," + nextInt1422 + "," + d922;
        byte[] digest22 = MessageDigest.getInstance("MD5").digest(androidx.compose.foundation.c.p(str2022, b7).getBytes("UTF-8"));
        StringBuilder sb722 = new StringBuilder(digest22.length * 2);
        while (r8 < r7) {
        }
        String str2122 = sb722.toString();
        HashMap hashMap32 = f3709a;
        hashMap32.put("androidId", b5);
        hashMap32.put("gsfAndroidId", str7);
        hashMap32.put("advertiserId", uuid);
        hashMap32.put("vendingSourceDir", encodeToString);
        hashMap32.put("gmsSourceDir", encodeToString2);
        hashMap32.put("keyECHashKey2", b18);
        hashMap32.put("keyECNewId", q);
        hashMap32.put("randomInstallShopeeTime", valueOf);
        hashMap32.put("randomUpdateShopeeTime", valueOf);
        hashMap32.put("randomInstallOthersTime", valueOf2);
        hashMap32.put("randomUpdateOthersTime", valueOf3);
        hashMap32.put("totalMem", String.valueOf(nextInt5));
        hashMap32.put("totalSize", String.valueOf(blockCountLong));
        hashMap32.put("remainSize", String.valueOf(nextDouble));
        hashMap32.put("shopeeSourceDir", String.valueOf(encodeToString3));
        hashMap32.put("apkSourceDir", String.valueOf(encodeToString4));
        hashMap32.put("fix7cTime", String.valueOf(nextInt6));
        hashMap32.put("screenOffTimeout", String.valueOf(i6));
        hashMap32.put("bootCount", String.valueOf(nextInt7));
        hashMap32.put("fixAEAccessTime", String.valueOf(nextInt8));
        hashMap32.put("fixAEModifyTime", String.valueOf(nextInt9));
        hashMap32.put("m1INodeRandom", String.valueOf(e));
        hashMap32.put("vbmetaDigest", str5);
        hashMap32.put("0x4", uuid2);
        hashMap32.put("0x5", b6);
        hashMap32.put("0x6", b7);
        hashMap32.put("0x7", b8);
        hashMap32.put("0x8", String.valueOf(d5));
        hashMap32.put("0x1c", str12);
        hashMap32.put("0x3e", b20);
        hashMap32.put("0x39", b12);
        hashMap32.put("0x40", str);
        hashMap32.put("0x42", str2);
        hashMap32.put("0x43", b15);
        hashMap32.put("0x44", "zh-Hant-TW");
        hashMap32.put("0x59", "Asia/Taipei");
        hashMap32.put("0x50", "1");
        hashMap32.put("0x5d", "1");
        hashMap32.put("0x62", "14");
        hashMap32.put("0x63", "" + ((e(1, 4) * nextInt10) / 10));
        hashMap32.put("0x64", "" + ((e(1, 4) * nextInt11) / 10));
        hashMap32.put("0x65", "" + nextInt10);
        hashMap32.put("0x66", "" + nextInt11);
        hashMap32.put("0x67", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
        hashMap32.put("0x68", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
        hashMap32.put("0x6a", "0");
        hashMap32.put("0x6d", HttpUrl.PATH_SEGMENT_ENCODE_SET_URI);
        hashMap32.put("0x6c", "0");
        hashMap32.put("0x75", "/data/app/~~" + encodeToString4 + "/com.shopee.tw-" + encodeToString3 + "/split_config.arm64_v8a.apk!libshpssdk.so");
        hashMap32.put("0x91", l5);
        hashMap32.put("0x9c", b9);
        hashMap32.put("0xa4", b16);
        hashMap32.put("0xa9", String.valueOf(d5));
        hashMap32.put("0xaa", "-1");
        hashMap32.put("0xac", "1");
        hashMap32.put("0xaf", l4);
        hashMap32.put("0xb3", sb2);
        hashMap32.put("0xc3", String.valueOf(nextInt7));
        hashMap32.put("0xdc", String.valueOf(j6));
        hashMap32.put("0xe4", b10);
        hashMap32.put("0xe5", b11);
        hashMap32.put("0x102", b13);
        hashMap32.put("0x105", str13);
        hashMap32.put("0x106", str18);
        hashMap32.put("0x10a", b14);
        hashMap32.put("0x119", str8);
        hashMap32.put("0x126", b19);
        hashMap32.put("0x127", str11);
        hashMap32.put("0x12a", b21);
        hashMap32.put("0x128", format);
        hashMap32.put("0x133", String.valueOf(randomUUID));
        hashMap32.put("0x137", valueOf4);
        hashMap32.put("0x138", valueOf4);
        hashMap32.put("0x139", encodeToString3);
        hashMap32.put("0x13a", String.valueOf(e));
        hashMap32.put("0x13b", String.valueOf(d6));
        hashMap32.put("0x13c", String.valueOf(d8));
        hashMap32.put("0x13d", String.valueOf(d7));
        byte[] bArr22 = new byte[48];
        i.f3715b.nextBytes(bArr22);
        char[] cArr22 = new char[96];
        while (i4 < 48) {
        }
        hashMap32.put("0x14a", new String(cArr22));
        HashMap hashMap222 = f3709a;
        hashMap222.put("0x140", str2022);
        hashMap222.put("0x141", str2122);
        hashMap222.put("0x146", t3);
        hashMap222.put("0x152", a4);
        hashMap222.put("0x153", a5);
        hashMap222.put("0x154", a6);
        hashMap222.put("0x165", b22);
        hashMap222.put("0x166", str4);
        hashMap222.put("0x2015", str5);
        hashMap222.put("0xd3", "{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}");
        random = h.f3713a;
        String str2222 = g.b(new int[]{i}[random.nextInt(1)]) + "," + g.b(4);
        int i1022 = g.c(3)[random.nextInt(g.c(3).length)];
        String concat32 = (i1022 != 1 ? i1022 != i ? i1022 != 3 ? "null" : "O3" : "O2" : "O1").concat(",");
        String concat222 = "TW".concat(",");
        hashMap222.put("gsm.sim.state", "LOADED,ABSENT");
        hashMap222.put("gsm.network.type", str2222);
        hashMap222.put("gsm.operator.alpha", concat32);
        hashMap222.put("gsm.operator.iso-country", concat222);
        hashMap222.put("0x1b", "LOADED,ABSENT");
        hashMap222.put("0x2e", str2222);
        hashMap222.put("0x3a", concat32);
        while (true) {
            nextInt = random.nextInt(223) + 1;
            nextInt2 = random.nextInt(256);
            nextInt3 = random.nextInt(256);
            nextInt4 = random.nextInt(256);
            if (nextInt >= 0) {
            }
        }
        hashMap222.put("0x41", nextInt + "." + nextInt2 + "." + nextInt3 + "." + nextInt4);
    }

    public static long d(long j4, long j5) {
        if (j4 < j5) {
            return j4 + ((long) (f3710b.nextDouble() * ((j5 - j4) + 1)));
        }
        throw new IllegalArgumentException("max must be greater than min");
    }

    public static int e(int i, int i4) {
        return f3710b.nextInt((i4 - i) + 1) + i;
    }

    public static long f(String str) {
        try {
            try {
                Date parse = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).parse(str);
                if (parse != null) {
                    return parse.getTime();
                }
                return 0L;
            } catch (Exception unused) {
                return 0L;
            }
        } catch (Exception unused2) {
            Date parse2 = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.ROOT).parse(str);
            if (parse2 != null) {
                return parse2.getTime();
            }
            return 0L;
        }
    }

    public static String g(long j4) {
        return new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).format(new Date(j4));
    }
}
