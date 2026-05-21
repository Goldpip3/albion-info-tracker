// Prevents an extra console window from opening alongside the GUI on
// Windows release builds.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    gda_desktop_lib::run()
}
