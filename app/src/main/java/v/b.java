package v;
import android.content.Context;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.content.pm.Signature;
import android.os.Build;
import android.system.Os;
import android.system.StructStat;
import android.util.Log;
import java.security.MessageDigest;
import java.util.Random;
public abstract class b {
    public static final Random a = new Random();
    public static String a(Context ctx, String pkg) {
        if (pkg.isEmpty()) return null;
        try {
            PackageManager pm = ctx.getPackageManager();
            PackageInfo pi = pm.getPackageInfo(pkg, 0);
            if (pi == null) return null;
            String src = pi.applicationInfo.sourceDir;
            if (src == null || src.isEmpty()) return null;
            StructStat stat = Os.lstat(src);
            long size = stat.st_size;
            long inode = stat.st_ino + a.nextInt(9) + 2;
            String ver = pi.versionName != null ? pi.versionName : "";
            long verCode = Build.VERSION.SDK_INT >= 28 ? pi.getLongVersionCode() : pi.versionCode;
            long install = pi.firstInstallTime;
            long delta = pi.lastUpdateTime - install;
            String sig = b(pm, pkg);
            if (sig == null || sig.isEmpty()) sig = "d087e72912fba064cafa78dc34aea839";
            return sig + size + "," + ver + "," + verCode + "," + install + "," + delta + "," + inode;
        } catch (Exception e) {
            Log.e("ApkInfoUtils", "Error: " + pkg, e);
            return null;
        }
    }
    @SuppressWarnings("deprecation")
    public static String b(PackageManager pm, String pkg) {
        try {
            Signature sig = null;
            if (Build.VERSION.SDK_INT >= 28) {
                PackageInfo pi = pm.getPackageInfo(pkg, PackageManager.GET_SIGNING_CERTIFICATES);
                if (pi.signingInfo != null) {
                    Signature[] sigs = pi.signingInfo.hasMultipleSigners() ?
                        pi.signingInfo.getApkContentsSigners() : pi.signingInfo.getSigningCertificateHistory();
                    if (sigs != null && sigs.length > 0) sig = sigs[0];
                }
            } else {
                PackageInfo pi = pm.getPackageInfo(pkg, PackageManager.GET_SIGNATURES);
                if (pi.signatures != null && pi.signatures.length > 0) sig = pi.signatures[0];
            }
            return sig != null ? c(sig.toByteArray()) : null;
        } catch (Exception e) { return null; }
    }
    public static String c(byte[] data) {
        try {
            byte[] digest = MessageDigest.getInstance("MD5").digest(data);
            StringBuilder sb = new StringBuilder(digest.length * 2);
            for (byte b : digest) sb.append(String.format("%02x", b & 0xFF));
            return sb.toString();
        } catch (Exception e) { return null; }
    }
}
