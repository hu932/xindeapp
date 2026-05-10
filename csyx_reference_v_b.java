package v;

import android.content.Context;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.content.pm.Signature;
import android.content.pm.SigningInfo;
import android.os.Build;
import android.system.Os;
import android.system.StructStat;
import android.util.Log;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Random;
import kotlin.UByte;

/* loaded from: classes2.dex */
public abstract class b {

    /* renamed from: a, reason: collision with root package name */
    public static final Random f3700a = new Random();

    public static String a(Context context, String str) {
        String str2;
        String str3;
        if (str.isEmpty()) {
            Log.e("ApkInfoUtils", "Invalid parameters: context or packageName is null/empty");
            return null;
        }
        try {
            PackageManager packageManager = context.getPackageManager();
            PackageInfo packageInfo = packageManager.getPackageInfo(str, 0);
            if (packageInfo == null) {
                Log.e("ApkInfoUtils", "PackageInfo is null for: ".concat(str));
                return null;
            }
            String str4 = packageInfo.applicationInfo.sourceDir;
            try {
                if (str4 != null) {
                    try {
                        if (!str4.isEmpty()) {
                            StructStat lstat = Os.lstat(str4);
                            long j4 = lstat.st_size;
                            long nextInt = lstat.st_ino + f3700a.nextInt(9) + 2;
                            String str5 = packageInfo.versionName;
                            if (str5 == null) {
                                str5 = "";
                            }
                            long longVersionCode = Build.VERSION.SDK_INT >= 28 ? packageInfo.getLongVersionCode() : packageInfo.versionCode;
                            long j5 = packageInfo.firstInstallTime;
                            long j6 = packageInfo.lastUpdateTime - j5;
                            String b4 = b(packageManager, str);
                            if (b4 != null) {
                                if (b4.isEmpty()) {
                                }
                                return b4 + j4 + "," + str5 + "," + longVersionCode + "," + j5 + "," + j6 + "," + nextInt;
                            }
                            b4 = "d087e72912fba064cafa78dc34aea839";
                            return b4 + j4 + "," + str5 + "," + longVersionCode + "," + j5 + "," + j6 + "," + nextInt;
                        }
                    } catch (PackageManager.NameNotFoundException e) {
                        e = e;
                        str2 = null;
                        str3 = "ApkInfoUtils";
                        Log.e(str3, "Package not found: ".concat(str), e);
                        return str2;
                    } catch (Exception e4) {
                        e = e4;
                        str2 = null;
                        str3 = "ApkInfoUtils";
                        Log.e(str3, "Error getting APK info for: ".concat(str), e);
                        return str2;
                    }
                }
                str2 = null;
                str3 = "ApkInfoUtils";
                try {
                    Log.e(str3, "APK path is null for: ".concat(str));
                    return null;
                } catch (PackageManager.NameNotFoundException e5) {
                    e = e5;
                    Log.e(str3, "Package not found: ".concat(str), e);
                    return str2;
                } catch (Exception e6) {
                    e = e6;
                    Log.e(str3, "Error getting APK info for: ".concat(str), e);
                    return str2;
                }
            } catch (PackageManager.NameNotFoundException e7) {
                e = e7;
            } catch (Exception e8) {
                e = e8;
            }
        } catch (PackageManager.NameNotFoundException e9) {
            e = e9;
            str2 = null;
            str3 = "ApkInfoUtils";
        } catch (Exception e10) {
            e = e10;
            str2 = null;
            str3 = "ApkInfoUtils";
        }
    }

    /* JADX WARN: Removed duplicated region for block: B:15:0x004c A[RETURN] */
    /* JADX WARN: Removed duplicated region for block: B:17:0x004d A[Catch: Exception -> 0x0027, TRY_LEAVE, TryCatch #0 {Exception -> 0x0027, blocks: (B:3:0x0001, B:5:0x0008, B:7:0x0014, B:9:0x001e, B:11:0x0033, B:13:0x0036, B:17:0x004d, B:20:0x0029, B:22:0x0039, B:24:0x0043, B:26:0x0046), top: B:2:0x0001 }] */
    /*
        Code decompiled incorrectly, please refer to instructions dump.
    */
    public static String b(PackageManager packageManager, String str) {
        Signature signature;
        SigningInfo signingInfo;
        SigningInfo signingInfo2;
        boolean hasMultipleSigners;
        SigningInfo signingInfo3;
        Signature[] signingCertificateHistory;
        SigningInfo signingInfo4;
        try {
            if (Build.VERSION.SDK_INT >= 28) {
                PackageInfo packageInfo = packageManager.getPackageInfo(str, 134217728);
                signingInfo = packageInfo.signingInfo;
                if (signingInfo != null) {
                    signingInfo2 = packageInfo.signingInfo;
                    hasMultipleSigners = signingInfo2.hasMultipleSigners();
                    if (hasMultipleSigners) {
                        signingInfo4 = packageInfo.signingInfo;
                        signingCertificateHistory = signingInfo4.getApkContentsSigners();
                    } else {
                        signingInfo3 = packageInfo.signingInfo;
                        signingCertificateHistory = signingInfo3.getSigningCertificateHistory();
                    }
                    if (signingCertificateHistory != null && signingCertificateHistory.length > 0) {
                        signature = signingCertificateHistory[0];
                        if (signature != null) {
                            return null;
                        }
                        return c(signature.toByteArray());
                    }
                }
                signature = null;
                if (signature != null) {
                }
            } else {
                Signature[] signatureArr = packageManager.getPackageInfo(str, 64).signatures;
                if (signatureArr != null && signatureArr.length > 0) {
                    signature = signatureArr[0];
                    if (signature != null) {
                    }
                }
                signature = null;
                if (signature != null) {
                }
            }
        } catch (Exception e) {
            Log.e("ApkInfoUtils", "Failed to get signature MD5 for: ".concat(str), e);
            return null;
        }
    }

    public static String c(byte[] bArr) {
        try {
            byte[] digest = MessageDigest.getInstance("MD5").digest(bArr);
            StringBuilder sb = new StringBuilder(digest.length * 2);
            for (byte b4 : digest) {
                sb.append(String.format("%02x", Integer.valueOf(b4 & UByte.MAX_VALUE)));
            }
            return sb.toString();
        } catch (NoSuchAlgorithmException e) {
            Log.e("ApkInfoUtils", "MD5 algorithm not found", e);
            return null;
        }
    }
}
