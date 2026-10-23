# Plugin and v33 APK integration

This document records a local, static inspection of the supplied source and
APKs. It does not claim runtime validation on an Android device.

## Responsibilities

- The Android helper is built with namespace `com.app.helper`, but its installed
  application ID is `com.spotify.music` (`app/build.gradle.kts`).
- `FingerprintManager.generateFullFp()` calls `v.f.c(context)` and `v.f.a()`.
  These populate a string map with generated identifiers, environment values,
  numbered fields, and serialized `f7_params`.
- Fingerprint JSON files live under the helper's private `files/fps` directory.
- Selection is stored as `FP/useFp`, with `fp_prefs/active_fp` as a fallback.
- `EnvironmentProvider` exposes the selected JSON to another application.
- The modified Shopee APK contains the loading and hook implementation.
- The helper receives captured data and submits it to the Go backend. Server
  handling includes parsing wrappers and decrypting encrypted submissions.
- The browser extension and coupon server are separate components; they are
  not the Android APK's fingerprint provider or hook loader.

## Confirmed protocol correspondence

| Contract | Helper source | v33 engine |
|---|---|---|
| Installed helper package | `com.spotify.music` | Default broadcast destination is `com.spotify.music` |
| Provider authority | `com.yh.csyxgl.vivo.provider` | Same authority in the query URI |
| Selected fingerprint | `/current_fp` | Queries `content://com.yh.csyxgl.vivo.provider/current_fp` |
| Cursor payload | `key`, `content` columns | Reads the `content` column |
| Navigation command | `com.oppo.wake.launch.action`, `url` extra | Registers a receiver for the same action |
| Data callback | `com.app.helper.FILE_ACTION` | Constructs and sends that action |
| Callback payload | `extra_data`, `extra_data_path`, `task_id` | Sets those extras |
| Other callbacks | `LOGS_ACTION`, `TOAST_ACTION`, `VER_ACTION` | Matching strings and toast/version send methods |

The engine can change its broadcast destination upon receiving
`com.miui.update.action.check` with `pkg_name`; its default matches this project.
No assumption that the helper must use `com.yh.csyxgl.vivo` as its installed
package is warranted: provider authority and application ID are different contracts.

## APK implementation evidence

`v33_改机.apk`, version 3.73.27, uses
`android.app.content.core.wrapper.MarsComponentFactory` in its manifest. Its
small primary DEX contains the Mars wrapper and references `MarsMain`.
`assets/libmars.zip` contains `lib/libmars.dx`, ARM64 `libmars.so`, and a C++
runtime. The engine DEX includes Xposed/LSPosed compatibility classes and
Pine/Bien-related classes; its code calls `XposedHelpers.findAndHookMethod`.
The native library exports Dobby hook symbols. No standard Frida Gadget or
Frida Gum markers were identified. A `FRIDA` string alone is not proof that
Frida is bundled or used.

Relevant methods inside `libmars.dx`:

- `android.core.wrapper.hw.ground.maker.module.u.d.c`: queries the provider and
  reads the cursor's `content` column.
- `android.core.wrapper.hw.ground.maker.module.p.n`: parses that JSON, iterates
  all keys, and copies string values into its environment map.
- `android.core.wrapper.hw.ground.maker.module.p.w`: restores named environment
  values and `f7_params` from the map.
- `android.core.wrapper.hw.ground.a.l.c`: contains environment initialization,
  `files/cache-pf` loading/saving, and provider-read branches.
- `android.core.wrapper.hw.ground.maker.module.s.g`: creates the data callback
  with the file URI/path and task identifier, then sends the broadcast.

132 of the generator's 144 distinct map keys occur literally in the engine DEX.
The engine's generic JSON reader accepts keys beyond its literal string table;
this count does not prove that the other 12 keys are rejected or unused.

`虾皮官.apk`, version 3.76.25, uses the original Shopee component factory and
does not contain this Mars wrapper, nested engine package, provider URI, or
helper-specific callback actions in its inspected DEX files. The supplied APK
therefore does not have the same embedded integration as v33.

## Behaviors requiring attention

1. Selecting a fingerprint updates helper preferences and reports a `write`
   event to the backend. It does not notify the provider or send a reload
   command to Shopee. The UI message is not an acknowledgement from the hook
   engine. The engine's cache and in-memory state must be considered when
   diagnosing whether a changed selection has actually been read.
2. Generated names have second-level timestamps. Batch generation sleeps only
   10 milliseconds and uses that name directly, so multiple results in a second
   overwrite the same file. The reported count can exceed the file count.
3. `TaskService` reads a callback's `task_id` but does not compare it with the
   active task before setting the single `pendingFileContent` slot. Delayed or
   unrelated callbacks can therefore be consumed as the active task's result.
4. Static and service-level receivers both handle the file action. Runtime
   delivery and service lifecycle behavior have not been tested here.

No application source or APK was changed during this inspection. Local Python
inspection scripts were added to make the binary evidence reproducible.
