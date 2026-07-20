//go:build windows

package platform

// killPID is a stub on Windows; see windows.go.
func killPID(pid int, sig Signal) error { return ErrUnsupported }

// processAlive is a stub on Windows; see windows.go.
func processAlive(pid int) bool { return false }
