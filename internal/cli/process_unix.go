// Starting a process that outlives this one works differently on each
// operating system, so it's split into this file (Linux and macOS) and
// process_windows.go. The line below builds this file everywhere except
// Windows.

//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// detach makes command run in a new session, so it isn't stopped when the
// terminal that started it closes.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
