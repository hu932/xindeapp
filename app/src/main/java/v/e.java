package v;
import java.util.Random;
public abstract class e {
    public static String a_ip;
    public static String b_gw;
    static { a_ip = null; b_gw = null; }
    public static void a() {
        try {
            Random r = new Random();
            String sub = String.valueOf(r.nextInt(256));
            a_ip = "192.168." + sub + "." + r.nextInt(256);
            b_gw = "192.168." + sub + ".1";
        } catch (Throwable t) { t.printStackTrace(); }
    }
    public static String b() {
        byte[] b = new byte[6]; new Random().nextBytes(b);
        b[0] = (byte)((b[0] & 0xFE) | 0x02);
        StringBuilder sb = new StringBuilder(18);
        for (int i = 0; i < 6; i++) {
            if (sb.length() > 0) sb.append(":");
            sb.append(String.format("%02x", b[i]));
        }
        return sb.toString();
    }
}
