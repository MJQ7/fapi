//go:build !windows

package updates

import (
	"os/exec"
	"syscall"
)

// detach makes command run in a new session, so it keeps running after the
// fapi that started it exits, and isn't stopped when its terminal closes.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
