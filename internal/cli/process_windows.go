// Starting a process that outlives this one works differently on each
// operating system, so it's split into this file (Windows) and
// process_unix.go. The _windows in the file name makes Go build it only on
// Windows.

package cli

import (
	"os/exec"
	"syscall"
)

// detachedProcess is the Windows DETACHED_PROCESS flag: the new process gets
// no console window. The syscall package doesn't name it.
const detachedProcess = 0x00000008

// detach makes command run without a console and in its own process group,
// so closing the terminal or pressing Ctrl+C in it doesn't stop fapi.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
