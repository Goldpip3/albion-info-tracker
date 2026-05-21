//go:build windows

package main

import (
	"context"
	"log"

	"golang.org/x/sys/windows"
)

// watchParentExit blocks until the process pid terminates, then calls
// cancel for a graceful shutdown. The desktop shell launches the agent
// elevated (via ShellExecuteW "runas") and passes its own PID; because the
// unelevated shell can't terminate this higher-integrity process, the
// agent watches the shell instead and stops itself when the window closes.
func watchParentExit(pid int, cancel context.CancelFunc) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		log.Printf("parent-watch: open pid %d: %v", pid, err)
		return
	}
	defer windows.CloseHandle(h)
	if _, err := windows.WaitForSingleObject(h, windows.INFINITE); err != nil {
		log.Printf("parent-watch: wait pid %d: %v", pid, err)
		return
	}
	log.Printf("parent-watch: shell (pid %d) exited — shutting down", pid)
	cancel()
}
