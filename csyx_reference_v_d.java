package v;

import java.util.Random;
import java.util.regex.Pattern;
import org.json.JSONArray;
import org.json.JSONObject;

/* loaded from: classes2.dex */
public abstract class d {

    /* renamed from: a, reason: collision with root package name */
    public static c f3706a;

    static {
        Pattern.compile("SERIALNUMBER=([0-9a-fA-F]+)");
        f3706a = null;
    }

    public static String a(Random random, int i) {
        char[] charArray = "0123456789abcdef".toCharArray();
        char[] cArr = new char[i];
        for (int i4 = 0; i4 < i; i4++) {
            cArr[i4] = charArray[random.nextInt(16)];
        }
        return new String(cArr);
    }

    public static JSONObject b(c cVar) {
        if (cVar == null) {
            return null;
        }
        JSONObject jSONObject = new JSONObject();
        JSONArray jSONArray = new JSONArray();
        for (String str : cVar.f3701a) {
            jSONArray.put(str);
        }
        JSONArray jSONArray2 = new JSONArray();
        for (String str2 : cVar.f3702b) {
            jSONArray2.put(str2);
        }
        JSONArray jSONArray3 = new JSONArray();
        for (String str3 : cVar.f3703c) {
            jSONArray3.put(str3);
        }
        jSONObject.put("serialHexPool", jSONArray);
        jSONObject.put("snHexPool", jSONArray2);
        jSONObject.put("snDecPool", jSONArray3);
        jSONObject.put("c1TimeOffsetDays", cVar.f3704d);
        jSONObject.put("c2TimeOffsetDays", cVar.e);
        jSONObject.put("leafOffsetDays", cVar.f);
        jSONObject.put("leafGapMs", cVar.g);
        jSONObject.put("vbh2", cVar.h);
        jSONObject.put("dl", cVar.i);
        jSONObject.put("vbs", cVar.f3705j);
        return jSONObject;
    }

    public static void c(String str, String str2) {
        Random random;
        if (str == null || str.isEmpty()) {
            random = new Random();
        } else {
            long j4 = 1469598103934665603L;
            for (int i = 0; i < str.length(); i++) {
                j4 = (j4 ^ str.charAt(i)) * 1099511628211L;
            }
            random = new Random(j4);
        }
        String[] strArr = new String[6];
        String[] strArr2 = new String[6];
        String[] strArr3 = new String[6];
        int i4 = 0;
        while (true) {
            if (i4 >= 6) {
                break;
            }
            strArr[i4] = a(random, 40);
            strArr2[i4] = a(random, 40);
            char[] cArr = new char[22];
            cArr[0] = (char) (random.nextInt(9) + 49);
            for (int i5 = 1; i5 < 22; i5++) {
                cArr[i5] = (char) (random.nextInt(10) + 48);
            }
            strArr3[i4] = new String(cArr);
            i4++;
        }
        int nextInt = random.nextInt(61) - 30;
        int nextInt2 = random.nextInt(61) - 30;
        int i6 = -(random.nextInt(30) + 1);
        int nextInt3 = random.nextInt(50000) + 10000;
        if (str2 == null || str2.isEmpty()) {
            str2 = a(random, 64);
        }
        f3706a = new c(strArr, strArr2, strArr3, nextInt, nextInt2, i6, nextInt3, str2, "1", "0");
    }
}
