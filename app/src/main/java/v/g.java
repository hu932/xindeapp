package v;
public abstract class g {
    public static final int[] a = {1, 2, 3, 4, 5, 6, 7};
    public static String b(int i) {
        if (i == 1) return "NR";
        if (i == 2) return "LTE";
        if (i == 3) return "LTE_CA";
        if (i == 4) return "UNKNOWN";
        return "null";
    }
    public static int[] c(int i) {
        int[] r = new int[i]; System.arraycopy(a, 0, r, 0, i); return r;
    }
}
