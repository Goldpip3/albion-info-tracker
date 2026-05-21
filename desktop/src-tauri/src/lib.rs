use std::fs;
use std::path::PathBuf;

#[cfg(windows)]
mod elevate;

// New installs point at the shared Skirmish backend, matching the agent's
// default. A custom Worker still works because the token (room name) is
// what actually pairs the shell, agent, and any browser together.
const PUSH_URL: &str = "wss://albion-meter.goldpipe.workers.dev/ingest";

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .setup(|app| {
            debug_log(&format!("setup start; current_exe={:?}", std::env::current_exe()));

            // The token lives in agent.json. Reading (or creating) it here,
            // before the agent launches, guarantees the shell and agent
            // share one token — and pre-writing it keeps the agent's
            // first-run flow from opening a stray browser tab, since the
            // shell *is* the UI now.
            let token = ensure_token().unwrap_or_default();
            debug_log(&format!("token len={}", token.len()));

            // Launch the capture agent elevated (one UAC prompt). It runs as
            // a separate elevated process and meets this window only in the
            // Cloudflare Worker room — they never talk directly, so the
            // integrity boundary between an unelevated UI and an elevated
            // agent is not a problem. Passing our PID lets the agent shut
            // itself down when this window closes (no orphaned capture).
            #[cfg(windows)]
            match agent_path() {
                Some(agent) => {
                    // Elevate on a dedicated thread: ShellExecuteW("runas")
                    // can block (COM activation / UAC consent), and blocking
                    // setup() here would stop the window from ever building.
                    let pid = std::process::id();
                    std::thread::spawn(move || {
                        // NOTE: --verbose is temporary for diagnostics; it tees
                        // the agent's per-event log to %LocalAppData%\GDA\agent-verbose.log.
                        let rc = elevate::run_as_admin(&agent, &format!("--parent-pid {pid} --verbose"));
                        debug_log(&format!("runas agent={agent:?} rc={rc}"));
                    });
                    debug_log("runas dispatched on background thread");
                }
                None => debug_log("agent_path() = None (sidecar not found next to exe)"),
            }

            // Load the bundled UI at the pair URL so the existing
            // readPairFromURL() in the web app wires this window to the agent
            // without anyone typing a token.
            let url = if token.is_empty() {
                "index.html".to_string()
            } else {
                format!("index.html?pair={token}")
            };
            let built = tauri::WebviewWindowBuilder::new(app, "main", tauri::WebviewUrl::App(url.into()))
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

// ensure_token returns the existing pushToken from agent.json, or generates
// one and writes a minimal config the agent completes on launch.
fn ensure_token() -> Option<String> {
    let dir = gda_dir()?;
    let path = dir.join("agent.json");

    if let Ok(bytes) = fs::read(&path) {
        if let Ok(v) = serde_json::from_slice::<serde_json::Value>(&bytes) {
            if let Some(t) = v.get("pushToken").and_then(|t| t.as_str()) {
                if !t.is_empty() {
                    return Some(t.to_string());
                }
            }
        }
    }

    let token = random_token()?;
    let _ = fs::create_dir_all(&dir);
    let cfg = serde_json::json!({
        "pushUrl": PUSH_URL,
        "pushToken": token,
    });
    fs::write(&path, serde_json::to_vec_pretty(&cfg).ok()?).ok()?;
    Some(token)
}

// random_token returns 32 hex chars (16 random bytes) — the same shape the
// agent and website use for a pairing token.
fn random_token() -> Option<String> {
    let mut b = [0u8; 16];
    getrandom::getrandom(&mut b).ok()?;
    Some(b.iter().map(|x| format!("{x:02x}")).collect())
}
