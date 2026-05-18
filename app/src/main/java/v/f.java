package v;

import android.app.ActivityManager;
import android.content.Context;
import android.os.Environment;
import android.os.StatFs;
import android.provider.MediaStore;
import android.util.Log;
import java.security.MessageDigest;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Collections;
import java.util.Date;
import java.util.HashMap;
import java.util.Locale;
import java.util.Random;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;
import org.json.JSONObject;

public abstract class f {
    public static final HashMap<String, String> a = new HashMap<>();
    public static final Random b = new Random();
    public static final char[] c = "0123456789abcdef".toCharArray();

    public static void a() {
        HashMap<String, String> m = f.a;
        m.put("0x16","false");m.put("0x19","false");m.put("0x1a","0");m.put("0x1e","false");
        m.put("0x20","false");m.put("0x28","false");m.put("0x3d","0efde111a560f04fb73e18fd00cc8afa");
        m.put("0x46","false");m.put("0x49","false");m.put("0x4a","false");m.put("0x4b","false");
        m.put("0x6f","false");m.put("0x70","false");m.put("0x71","false");m.put("0x73","0");
        m.put("0x78","[]");m.put("0x79","[]");m.put("0x8e","false");m.put("0x96","false");
        m.put("0x97","false");m.put("0x98","false");m.put("0x99","false");m.put("0x9b","0");
        m.put("0xa5","com.shopee.app.asm.anr.launch.ShopeeAppComponentFactory");m.put("0xa6","0");
        m.put("0xd1","{\"wifi_on\": \"1\", \"cell_on\": \"1\", \"bluetooth_on\": \"0\", \"development_settings_enabled\": \"0\", \"auto_time_zone\": \"1\", \"adb_wifi_enabled\": \"0\", \"adb_enabled\": \"0\"}");
        m.put("0xc0","0");m.put("0xc1","0");m.put("0xc2","0");
        m.put("0x122","[\"/apex/com.android.runtime/bin/linker64\",\"/apex/com.android.runtime/lib64/bionic/libdl.so\",\"[anon:bytehook-plt-trampolines]\",\"[anon:shadowhook-enter]\",\"[anon:shadowhook-exit]\",\"[anon:shadowhook-hub-trampo]\"]");
        m.put("0x1f","false");m.put("0xab","1");m.put("0xb2","false");m.put("0xb5","2");
        m.put("0xb6","0efde111a560f04fb73e18fd00cc8afa");m.put("0xb7","{\"cf\": 0, \"or\": 0}");
        m.put("0xbd","0");m.put("0xe1","false");m.put("0xf6","2,0");
        m.put("0x11d","000000000");m.put("0x11e","000000000");m.put("0x121","0");m.put("0x123","{}");
        m.put("0x13e","71919287");m.put("0x13f","7");m.put("0x14c","0");
    }

    public static String b(int len) {
        StringBuilder sb = new StringBuilder(len);
        for (int i = 0; i < len; i++) sb.append(Integer.toHexString(f.b.nextInt(16)));
        return sb.toString();
    }

    public static void c(Context ctx) {
        String b5 = b(16); // androidId
        String str7 = "3" + b(15); // gsfAndroidId
        String uuid = UUID.randomUUID().toString();
        String enc1 = Base64.getUrlEncoder().encodeToString(hexToBytes(b(32)));
        String enc2 = Base64.getUrlEncoder().encodeToString(hexToBytes(b(32)));
        String enc3 = Base64.getUrlEncoder().encodeToString(hexToBytes(b(32)));
        String enc4 = Base64.getUrlEncoder().encodeToString(hexToBytes(b(32)));
        String uuid2 = UUID.randomUUID().toString();
        String b6 = b(32), b7 = b(32), b8 = b(32);
        String b9 = e.b(); // MAC address
        String b10 = b(32), b11 = b(32);
        String b12 = b(64), b13 = b(64), b14 = b(32), b15 = b(32), b16 = b(40);
        String b17 = b(16), b18 = b(16);
        String q = b17 + "-" + b18;
        String b19 = b(32), b20 = b(16), b21 = b(32), b22 = b(32), b23 = b(32);

        // sb2: 10-digit random decimal
        StringBuilder sb = new StringBuilder(10);
        for (int i = 0; i <= 9; i++) sb.append(f.b.nextInt(10));
        String sb2 = sb.toString();

        // MediaStore version
        String str8;
        try { str8 = MediaStore.getVersion(ctx).split(":")[0] + ":" + UUID.randomUUID(); }
        catch (Exception ex) { str8 = "1:" + UUID.randomUUID(); }

        // l4: decimal of unsigned hex
        String l4;
        try { l4 = Long.toString(Long.parseUnsignedLong(b17, 16)); } catch (Exception ex) { l4 = "0"; }

        long now = System.currentTimeMillis();
        long d4 = d(now - 604800000L, now - 172800000L);
        String valueOf = String.valueOf(d4);
        String valueOf2 = String.valueOf(d(ff("2022-01-01 00:00:00"), ff("2024-01-01 00:00:00")));
        String valueOf3 = String.valueOf(d(ff("2024-01-01 00:00:00"), ff("2025-09-01 00:00:00")));

        // Memory
        ActivityManager.MemoryInfo mi = new ActivityManager.MemoryInfo();
        ActivityManager am = (ActivityManager)ctx.getSystemService("activity");
        if (am != null) am.getMemoryInfo(mi);
        long nextInt5 = mi.totalMem - f.b.nextInt(100) * 1048576;

        // Storage
        StatFs sf = new StatFs(Environment.getExternalStorageDirectory().getAbsolutePath());
        long blk = sf.getBlockCountLong() * sf.getBlockSizeLong() - f.b.nextInt(100) * 1048576;
        long nextDbl = (long)(blk * (f.b.nextDouble() * 0.5 + 0.1));

        int nextInt6 = f.b.nextInt(100) + 100;
        int i6 = new int[]{15000, 30000, 60000}[f.b.nextInt(3)];
        int nextInt7 = f.b.nextInt(100) + 50;
        int nextInt8 = f.b.nextInt(259200) + 1;
        int nextInt9 = f.b.nextInt(7200) + 300;
        int nextInt10 = f.b.nextInt(300000000) + 300000000;
        int nextInt11 = f.b.nextInt(1000000) + 1000000;
        String str12 = new String[]{"mtp","sec_charging","mtp,sec_charging"}[f.b.nextInt(3)];

        // 0x91
        long d5 = d(ff(gg(now - 86400000L)), ff(gg(now - 7200000L)));
        long j6 = d5 / 1000;
        long elapsed = now - d5;
        String l5 = "{\n    \"elapsedRealtime\": \"" + elapsed + "\",\n    \"uptime\": \"" + elapsed + "\"\n}";
        String valueOf4 = String.valueOf(elapsed + e(100, 300));

        // Network
        e.a();
        String str = e.a_ip;
        String str2 = e.b_gw;
        int eVal = e(600000, 900000);
        long d6 = d4 - d(5000L, 12000L);
        long d7 = d(0L, 10000L) + d6;
        long d8 = d(0L, 10000L) + d7;

        String str11 = "0," + (f.b.nextInt(200) + 100);

        // vbmetaDigest
        String str5;
        try {
            long hash = 0;
            for (char ch : b5.toCharArray()) hash = hash * 31 + (ch & 0xFF);
            Random r3 = new Random(hash & 0xFFFFFFFFL);
            StringBuilder sb6 = new StringBuilder(64);
            for (int i2 = 0; i2 < 64; i2++) sb6.append(f.c[r3.nextInt(16)]);
            str5 = sb6.toString();
        } catch (Throwable t) { t.printStackTrace(); str5 = null; }

        // f7_params
        String f7str = null;
        try {
            d.c(b5, str5);
            v.c cv = d.a;
            if (cv != null) {
                JSONObject f7j = d.b(cv);
                if (f7j != null) f7str = f7j.toString();
            }
        } catch (Throwable t) { Log.e("mockfpenv", "F7 failed: " + t.getMessage()); }

        // IPv6
        String fmt = String.format("fe80::%x:%x:%x:%x", e(0,65535), e(0,65535), e(0,65535), e(0,65535));

        // Network info string
        String str3;
        if (str != null) {
            String[] parts = str.split("\\.");
            str3 = parts.length == 4 ? parts[0]+"."+parts[1]+"."+parts[2]+".0" : "0.0.0.0";
        } else str3 = "0.0.0.0";
        String t3 = "[\"" + "{InterfaceName: wlan0 LinkAddresses: [ " + fmt + "/64," + str + "/24 ] DnsAddresses: [ /8.8.8.8,/1.1.1.1 ] Domains: null MTU: 0 ServerAddress: /" + str2 + " TcpBufferSizes: 524288,1048576,8388608,524288,1048576,4194304 Routes: [ fe80::/64 -> :: wlan0 mtu 0," + str3 + "/24 -> 0.0.0.0 wlan0 mtu 0,0.0.0.0/0 -> " + str2 + " wlan0 mtu 0 ]}" + "\"]";

        // 0x105: device names
        ArrayList<String> names = new ArrayList<>();
        names.add("新的开始");names.add("灵光一现");names.add("新的旅程");names.add("下一章节");
        names.add("全新起点");names.add("隐藏宝藏");names.add("午夜思绪");names.add("明亮远方");
        names.add("静默回声");names.add("破土新生");names.add("全新视野");names.add("第二机会");
        Collections.shuffle(names);
        ArrayList<String> e105 = new ArrayList<>();
        for (int i2 = 0; i2 < 4; i2++) e105.add("\"" + i2 + "\": \"" + names.get(i2) + "\"");
        String str13 = "{" + String.join(", ", e105) + "}";

        // 0x106: 7-item stats
        ArrayList<String> e106 = new ArrayList<>();
        int ea = e(0,1), eb = e(ea,7); e106.add("\"0\": \"" + ea+",7,"+eb+","+eb+",-1,-1,-1,-1,-1\"");
        e106.add("\"1\": \"" + e(0,7)+",7,"+e(0,7)+","+e(0,7) + "\"");
        int ec = e(0,7); e106.add("\"2\": \"0,7,"+ec+","+ec+",-1,-1,-1,-1,-1,-1\"");
        int ed = e(0,25); e106.add("\"3\": \"0,25,"+ed+","+ed+",-1,"+e(0,25)+",-1,-1,-1,-1\"");
        int ee = e(0,1), ef = e(ee,7); e106.add("\"4\": \"" + ee+",7,"+ef+","+ef+",-1\"");
        int eg = e(0,7); e106.add("\"5\": \"0,7,"+eg+","+eg+",-1,-1,-1,-1\"");
        e106.add("\"6\": \"" + e(0,7)+",-1,-1,-1,-1,-1\"");
        String str18 = "{" + String.join(", ", e106) + "}";

        // 0x140/0x141
        long d92 = d(ff("2024-01-01 00:00:00"), ff(gg(now))) / 1000;
        long ni142 = ThreadLocalRandom.current().nextInt(0, 10) + d92;
        String str202 = ThreadLocalRandom.current().nextInt(100,2000)+",33188,0,0,"+ThreadLocalRandom.current().nextInt(300,8000)+","+(ThreadLocalRandom.current().nextInt(0,86400)+ni142)+","+ni142+","+d92;
        String str212;
        try {
            byte[] dig = MessageDigest.getInstance("MD5").digest((str202+b7).getBytes("UTF-8"));
            StringBuilder sb7 = new StringBuilder(dig.length*2);
            for (byte bv : dig) { String hx = Integer.toHexString(bv & 0xFF); if (hx.length()==1) sb7.append('0'); sb7.append(hx); }
            str212 = sb7.toString();
        } catch (Exception ex) { str212 = b(32); }

        // 0x75
        String path75 = "/data/app/~~" + enc4 + "/com.shopee.tw-" + enc3 + "/split_config.arm64_v8a.apk!libshpssdk.so";

        // Package info
        String a4 = v.b.a(ctx, "com.android.systemui");
        String a5 = v.b.a(ctx, "com.android.settings");
        String a6 = v.b.a(ctx, "android");
        if (a4 == null) a4 = "d087e72912fba064cafa78dc34aea839";
        if (a5 == null) a5 = "d087e72912fba064cafa78dc34aea839";
        if (a6 == null) a6 = "d087e72912fba064cafa78dc34aea839";

        // gsm
        Random hr = h.a;
        String nwType = g.b(new int[]{2}[hr.nextInt(1)]) + "," + g.b(4);
        int opIdx = g.c(3)[hr.nextInt(g.c(3).length)];
        String opAlpha = (opIdx==1?"O1":opIdx==2?"O2":opIdx==3?"O3":"null") + ",";

        // Public IP
        int na,nb,nc,nd;
        while (true) {
            na=hr.nextInt(223)+1; nb=hr.nextInt(256); nc=hr.nextInt(256); nd=hr.nextInt(256);
            if (na==0||na==10||na==127) continue;
            if (na==169&&nb==254) continue;
            if (na==172&&nb>=16&&nb<=31) continue;
            if (na==192&&nb==168) continue;
            if (na==100&&nb>=64&&nb<=127) continue;
            if (na==192&&nb==0&&nc==0) continue;
            if (na==192&&nb==0&&nc==2) continue;
            if (na==198&&(nb==18||nb==19)) continue;
            if (na==198&&nb==51&&nc==100) continue;
            if (na==203&&nb==0&&nc==113) continue;
            if (na>=224) continue;
            if (na==255&&nb==255&&nc==255&&nd==255) continue;
            break;
        }
        String ip41 = na+"."+nb+"."+nc+"."+nd;

        // ===== PUT ALL INTO HASHMAP =====
        HashMap<String, String> m = f.a;
        if (f7str != null && !f7str.isEmpty()) m.put("f7_params", f7str);
        m.put("androidId",b5); m.put("gsfAndroidId",str7); m.put("advertiserId",uuid);
        m.put("vendingSourceDir",enc1); m.put("gmsSourceDir",enc2);
        m.put("keyECHashKey2",b18); m.put("keyECNewId",q);
        m.put("randomInstallShopeeTime",valueOf); m.put("randomUpdateShopeeTime",valueOf);
        m.put("randomInstallOthersTime",valueOf2); m.put("randomUpdateOthersTime",valueOf3);
        m.put("totalMem",String.valueOf(nextInt5)); m.put("totalSize",String.valueOf(blk));
        m.put("remainSize",String.valueOf(nextDbl));
        m.put("shopeeSourceDir",String.valueOf(enc3)); m.put("apkSourceDir",String.valueOf(enc4));
        m.put("fix7cTime",String.valueOf(nextInt6)); m.put("screenOffTimeout",String.valueOf(i6));
        m.put("bootCount",String.valueOf(nextInt7));
        m.put("fixAEAccessTime",String.valueOf(nextInt8)); m.put("fixAEModifyTime",String.valueOf(nextInt9));
        m.put("m1INodeRandom",String.valueOf(eVal)); m.put("vbmetaDigest",str5);
        m.put("0x4",uuid2); m.put("0x5",b6); m.put("0x6",b7); m.put("0x7",b8);
        m.put("0x8",String.valueOf(d5)); m.put("0x1c",str12);
        m.put("0x3e",b20); m.put("0x39",b12); m.put("0x40",str); m.put("0x42",str2); m.put("0x43",b15);
        m.put("0x44","zh-Hant-TW"); m.put("0x59","Asia/Taipei"); m.put("0x50","1"); m.put("0x5d","1"); m.put("0x62","14");
        m.put("0x63",""+((e(1,4)*nextInt10)/10)); m.put("0x64",""+((e(1,4)*nextInt11)/10));
        m.put("0x65",""+nextInt10); m.put("0x66",""+nextInt11);
        m.put("0x67","[]"); m.put("0x68","[]"); m.put("0x6a","0"); m.put("0x6d","[]"); m.put("0x6c","0");
        m.put("0x75",path75); m.put("0x91",l5);
        m.put("0x9c",b9); m.put("0xa4",b16); m.put("0xa9",String.valueOf(d5));
        m.put("0xaa","-1"); m.put("0xac","1"); m.put("0xaf",l4); m.put("0xb3",sb2);
        m.put("0xc3",String.valueOf(nextInt7)); m.put("0xdc",String.valueOf(j6));
        m.put("0xe4",b10); m.put("0xe5",b11); m.put("0x102",b13);
        m.put("0x105",str13); m.put("0x106",str18); m.put("0x10a",b14);
        m.put("0x119",str8); m.put("0x126",b19); m.put("0x127",str11);
        m.put("0x12a",b21); m.put("0x128",fmt);
        m.put("0x133",String.valueOf(UUID.randomUUID()));
        m.put("0x137",valueOf4); m.put("0x138",valueOf4);
        m.put("0x139",enc3); m.put("0x13a",String.valueOf(eVal));
        m.put("0x13b",String.valueOf(d6)); m.put("0x13c",String.valueOf(d8)); m.put("0x13d",String.valueOf(d7));
        byte[] secB = new byte[48]; i.b.nextBytes(secB);
        char[] c14 = new char[96];
        for (int ix = 0; ix < 48; ix++) { int bv = secB[ix]&0xFF; c14[ix*2]=i.c[bv>>>4]; c14[ix*2+1]=i.c[bv&0xF]; }
        m.put("0x14a", new String(c14));
        m.put("0x140",str202); m.put("0x141",str212);
        m.put("0x146",t3); m.put("0x152",a4); m.put("0x153",a5); m.put("0x154",a6);
        m.put("0x165",b22); m.put("0x166",b23); m.put("0x2015",str5);
        m.put("0xd3","{\"system_locales\": \"zh-Hant-TW\", \"font_scale\": \"1.0\"}");
        m.put("gsm.sim.state","LOADED,ABSENT"); m.put("gsm.network.type",nwType);
        m.put("gsm.operator.alpha",opAlpha); m.put("gsm.operator.iso-country","TW,");
        m.put("0x1b","LOADED,ABSENT"); m.put("0x2e",nwType); m.put("0x3a",opAlpha);
        m.put("0x41",ip41);
    }

    public static long d(long min, long max) {
        if (min >= max) return min;
        return min + (long)(f.b.nextDouble() * (max - min + 1));
    }
    public static int e(int min, int max) { return f.b.nextInt(max - min + 1) + min; }
    public static long ff(String s) {
        try { Date d = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).parse(s); return d != null ? d.getTime() : 0; }
        catch (Exception e) { try { Date d = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.ROOT).parse(s); return d != null ? d.getTime() : 0; } catch (Exception e2) { return 0; } }
    }
    public static String gg(long ms) { return new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", Locale.ROOT).format(new Date(ms)); }

    private static byte[] hexToBytes(String hex) {
        int len = hex.length(); byte[] data = new byte[len/2];
        for (int i = 0; i < len; i += 2) data[i/2] = (byte)((Character.digit(hex.charAt(i),16)<<4)+Character.digit(hex.charAt(i+1),16));
        return data;
    }
}
