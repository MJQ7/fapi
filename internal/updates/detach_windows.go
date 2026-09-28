package updates

import (
	"os/exec"
	"syscall"
)

// detachedProcess is the Windows DETACHED_PROCESS flag: the new process gets
// no console window. The syscall package doesn't name it.
const detachedProcess = 0x00000008

// detach makes command run without a console and in its own process group,
// so it keeps running after the fapi that started it exits, and closing
// that fapi's terminal doesn't stop it.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
