//go:build unix

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// killPID sends SIGTERM (Terminate) or SIGKILL (Force) to a process. Shared by
// the linux and darwin providers.
func killPID(pid int, sig Signal) error {
	if pid <= 0 {
		return fmt.Errorf("refusing to signal pid %d", pid)
	}
	s := syscall.SIGTERM
	if sig == Force {
		s = syscall.SIGKILL
	}
	if err := syscall.Kill(pid, s); err != nil {
		switch {
		case errors.Is(err, syscall.EPERM):
			return fmt.Errorf("%w (pid %d)", ErrPermission, pid)
		case errors.Is(err, syscall.ESRCH):
			return fmt.Errorf("%w: pid %d already exited", ErrNotFound, pid)
		default:
			return fmt.Errorf("signal %s to pid %d: %w", sig, pid, err)
		}
	}
	return nil
}

// processAlive reports whether a PID still exists, used to verify a terminate
// actually took effect before reporting success.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 performs error checking without delivering a signal.
	return p.Signal(syscall.Signal(0)) == nil
}
