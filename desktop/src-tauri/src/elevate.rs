//! Launch the capture agent with elevated privileges via ShellExecuteW's
//! "runas" verb. The UI process stays unprivileged; only the agent (which
//! needs admin for raw-socket capture) is elevated. They never communicate
//! directly — they meet in the Cloudflare Worker room — so the integrity
//! boundary that would otherwise block stdio between them does not matter.

use std::ffi::OsStr;
use std::os::windows::ffi::OsStrExt;
use std::path::Path;

use windows_sys::Win32::System::Com::{CoInitializeEx, COINIT_APARTMENTTHREADED};
use windows_sys::Win32::UI::Shell::ShellExecuteW;
use windows_sys::Win32::UI::WindowsAndMessaging::SW_HIDE;

fn wide(s: &OsStr) -> Vec<u16> {
    s.encode_wide().chain(std::iter::once(0)).collect()
}

// run_as_admin triggers a single UAC prompt and starts exe (elevated) with
// the given argument string. Returns the ShellExecuteW result as isize: a
// value > 32 means success, <= 32 is an error code. Best-effort: if the
// user declines the prompt the agent simply does not start and the meter
// shows "disconnected".
pub fn run_as_admin(exe: &Path, args: &str) -> isize {
    let verb = wide(OsStr::new("runas"));
    let file = wide(exe.as_os_str());
    let params = wide(OsStr::new(args));
    let h = unsafe {
        // ShellExecuteW can delegate to COM-activated shell handlers, so the
        // calling thread must have a COM apartment. Initialize an STA here
        // (this runs on a dedicated thread, never the UI thread). Ignoring
        // the HRESULT is fine — S_FALSE just means already-initialized.
        let _ = CoInitializeEx(std::ptr::null(), COINIT_APARTMENTTHREADED as u32);
        ShellExecuteW(
            std::ptr::null_mut(),
            verb.as_ptr(),
            file.as_ptr(),
            params.as_ptr(),
            std::ptr::null(),
            SW_HIDE,
        )
    };
    h as isize
}
