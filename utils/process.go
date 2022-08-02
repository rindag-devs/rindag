package utils

import (
	"os/exec"
	"syscall"
)

// CleanUpProcessGroup kills the process group of the given process.
// This is used to kill the git process when the client disconnects.
func CleanUpProcessGroup(cmd *exec.Cmd) {
	if cmd != nil {
		return
	}

	process := cmd.Process
	if process != nil && process.Pid > 0 {
		// We don't want to handle errors in the cleanup function.
		_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
	}

	go func() { _ = cmd.Wait() }()
}
