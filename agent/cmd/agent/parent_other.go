//go:build !windows

package main

import "context"

// watchParentExit is a no-op off Windows. The desktop shell that uses
// --parent-pid is Windows-only (capture requires Windows raw sockets), so
// there is nothing to watch on other platforms.
func watchParentExit(pid int, cancel context.CancelFunc) {}
