package v;

import androidx.constraintlayout.core.motion.utils.TypedValues;
import java.util.Random;

/* loaded from: classes2.dex */
public abstract class e {

    /* renamed from: a, reason: collision with root package name */
    public static String f3707a;

    /* renamed from: b, reason: collision with root package name */
    public static String f3708b;

    static {
        Random random = new Random(System.currentTimeMillis());
        f3707a = null;
        f3708b = null;
        b();
        b();
        random.nextInt(TypedValues.Custom.TYPE_INT);
        int nextInt = random.nextInt(100);
        if (nextInt > 0 && nextInt < 100) {
            if (nextInt <= 10) {
                Math.round((nextInt / 10.0d) * 200.0d);
            } else if (nextInt <= 90) {
                Math.round(((nextInt - 10) / 80.0d) * 500.0d);
            } else {
                Math.round(((nextInt - 90) / 10.0d) * 400.0d);
            }
        }
        random.nextInt(50);
        random.nextInt(130);
    }

    public static void a() {
        try {
            Random random = new Random();
            String valueOf = String.valueOf(random.nextInt(256));
            f3707a = "192.168." + valueOf + "." + random.nextInt(256);
            StringBuilder sb = new StringBuilder("192.168.");
            sb.append(valueOf);
            sb.append(".1");
            f3708b = sb.toString();
        } catch (Throwable th) {
            th.printStackTrace();
        }
    }

    public static String b() {
        byte[] bArr = new byte[6];
        new Random().nextBytes(bArr);
        byte b4 = (byte) (bArr[0] & 254);
        bArr[0] = b4;
        bArr[0] = (byte) (b4 | 2);
        StringBuilder sb = new StringBuilder(18);
        for (int i = 0; i < 6; i++) {
            byte b5 = bArr[i];
            if (sb.length() > 0) {
                sb.append(":");
            }
            sb.append(String.format("%02x", Byte.valueOf(b5)));
        }
        return sb.toString();
    }
}
