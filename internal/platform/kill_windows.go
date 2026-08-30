//go:build windows

package platform

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const stillActive = 259

// Windows has no process-wide equivalent of SIGTERM for arbitrary processes.
// TerminateProcess is therefore used for both requests; the distinction remains
// in history/UI so callers keep one cross-platform API.
func killPID(pid int, sig Signal) error {
	if pid <= 0 {
		return fmt.Errorf("refusing to terminate pid %d", pid)
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return windowsProcessError("open process", pid, err)
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return windowsProcessError("terminate process", pid, err)
	}
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
