use std::path::PathBuf;

#[cfg(windows)]
mod elevate;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_updater::Builder::new().build())
        .setup(|app| {
            debug_log(&format!("setup start; current_exe={:?}", std::env::current_exe()));

            // Auto-update: on launch, check the GitHub Releases manifest for a
            // newer signed build and, if found, download + install it, then
            // relaunch into the new version. Best-effort and silent — any
            // error (offline, no update) just leaves the app on the current
            // version. Only in release builds; dev builds never self-update.
            #[cfg(not(debug_assertions))]
            {
                let handle = app.handle().clone();
                tauri::async_runtime::spawn(async move {
                    check_for_update(handle).await;
                });
            }

            // Launch the capture agent elevated (one UAC prompt) in LOCAL
            // mode: it serves the meter + its /view WebSocket on
            // 127.0.0.1:8787 and does NOT push to Cloudflare. This window
            // loads the bundled UI, which auto-detects the desktop shell
            // (tauri.localhost) and connects to that local socket — so no
            // data ever leaves the machine and no pairing token is needed.
            // Passing our PID lets the agent self-exit when this window
            // closes (no orphaned capture process).
            #[cfg(windows)]
            match agent_path() {
                Some(agent) => {
                    // Elevate on a dedicated thread: ShellExecuteW("runas")
                    // can block (COM activation / UAC consent), and blocking
                    // setup() here would stop the window from ever building.
                    let pid = std::process::id();
                    std::thread::spawn(move || {
                        let rc = elevate::run_as_admin(&agent, &format!("--local --parent-pid {pid}"));
                        debug_log(&format!("runas agent={agent:?} rc={rc}"));
                    });
                    debug_log("runas dispatched on background thread");
                }
                None => debug_log("agent_path() = None (sidecar not found next to exe)"),
            }

            // Load the bundled UI. It detects the tauri.localhost origin and
            // connects to the agent's local /view socket on its own; the
            // socket auto-reconnects, so it's fine that the agent (waiting on
            // the UAC prompt) may not be listening for a second or two yet.
            let built = tauri::WebviewWindowBuilder::new(app, "main", tauri::WebviewUrl::App("index.html".into()))
                .title("GDA Meter")
                .inner_size(1280.0, 820.0)
                .min_inner_size(900.0, 600.0)
                .build();
            debug_log(&format!("window build ok={}", built.is_ok()));
            built?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running the GDA desktop shell");
}

// check_for_update asks the GitHub Releases manifest whether a newer signed
// build exists; if so it downloads, installs, and relaunches. Silent and
// best-effort: friends never have to reinstall — opening the app pulls the
// latest fix on its own.
#[cfg(not(debug_assertions))]
async fn check_for_update(app: tauri::AppHandle) {
    use tauri_plugin_updater::UpdaterExt;
    let updater = match app.updater() {
        Ok(u) => u,
        Err(e) => {
            debug_log(&format!("updater init failed: {e}"));
            return;
        }
    };
    match updater.check().await {
        Ok(Some(update)) => {
            debug_log(&format!("update available: {}", update.version));
            match update.download_and_install(|_, _| {}, || {}).await {
                Ok(_) => {
                    debug_log("update installed; restarting");
                    app.restart();
                }
                Err(e) => debug_log(&format!("update install failed: {e}")),
            }
        }
        Ok(None) => debug_log("no update available"),
        Err(e) => debug_log(&format!("update check failed: {e}")),
    }
}

// agent_path returns the bundled capture agent. Tauri's externalBin strips
// the target-triple suffix at bundle time and places the file next to the
// app executable, so at runtime it is simply "agent.exe".
#[cfg(windows)]
fn agent_path() -> Option<PathBuf> {
    let exe = std::env::current_exe().ok()?;
    let dir = exe.parent()?;
    let bundled = dir.join("agent.exe");
    if bundled.exists() {
        return Some(bundled);
    }
    // Dev fallback: the un-renamed sidecar dropped into the build dir.
    let dev = dir.join("agent-x86_64-pc-windows-msvc.exe");
    dev.exists().then_some(dev)
}

// debug_log appends a line to %LocalAppData%\GDA\desktop-debug.log. The
// shell has no console (windows_subsystem = "windows"), so this file is the
// only way to see what setup() did. Cheap, best-effort, append-only.
fn debug_log(msg: &str) {
    if let Some(dir) = gda_dir() {
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join("desktop-debug.log");
        use std::io::Write;
        if let Ok(mut f) = std::fs::OpenOptions::new().create(true).append(true).open(path) {
            let _ = writeln!(f, "{msg}");
        }
    }
}

// gda_dir mirrors the agent's config.Dir(): %LocalAppData%\GDA on Windows.
// Same user (the runas elevation does not switch accounts) → same path, so
// the shell and the elevated agent agree on where agent.json lives.
fn gda_dir() -> Option<PathBuf> {
    std::env::var_os("LOCALAPPDATA").map(|p| PathBuf::from(p).join("GDA"))
}
