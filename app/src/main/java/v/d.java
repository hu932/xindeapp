package v;
import java.util.Random;
import org.json.JSONArray;
import org.json.JSONObject;
public abstract class d {
    public static c a;
    public static String a(Random random, int len) {
        char[] hex = "0123456789abcdef".toCharArray();
        char[] r = new char[len];
        for (int i = 0; i < len; i++) r[i] = hex[random.nextInt(16)];
        return new String(r);
    }
    public static JSONObject b(c cv) {
        if (cv == null) return null;
        JSONObject j = new JSONObject();
        JSONArray s1 = new JSONArray(); for (String s : cv.a) s1.put(s);
        JSONArray s2 = new JSONArray(); for (String s : cv.b) s2.put(s);
        JSONArray s3 = new JSONArray(); for (String s : cv.c2) s3.put(s);
        j.put("serialHexPool", s1); j.put("snHexPool", s2); j.put("snDecPool", s3);
        j.put("c1TimeOffsetDays", cv.d); j.put("c2TimeOffsetDays", cv.e);
        j.put("leafOffsetDays", cv.f); j.put("leafGapMs", cv.g);
        j.put("vbh2", cv.h); j.put("dl", cv.i); j.put("vbs", cv.j);
        return j;
    }
    public static void c(String androidId, String digest) {
        Random random;
        if (androidId == null || androidId.isEmpty()) { random = new Random(); }
        else {
            long h = 1469598103934665603L;
            for (int i = 0; i < androidId.length(); i++) h = (h ^ androidId.charAt(i)) * 1099511628211L;
            random = new Random(h);
        }
        String[] ser = new String[6], snH = new String[6], snD = new String[6];
        for (int i = 0; i < 6; i++) {
            ser[i] = a(random, 40); snH[i] = a(random, 40);
            char[] dc = new char[22]; dc[0] = (char)(random.nextInt(9) + 49);
            for (int j = 1; j < 22; j++) dc[j] = (char)(random.nextInt(10) + 48);
            snD[i] = new String(dc);
        }
        int c1 = random.nextInt(61) - 30, c2 = random.nextInt(61) - 30;
        int leaf = -(random.nextInt(30) + 1), gap = random.nextInt(50000) + 10000;
        String vbh = (digest == null || digest.isEmpty()) ? a(random, 64) : digest;
        d.a = new c(ser, snH, snD, c1, c2, leaf, gap, vbh, "1", "0");
    }
}
