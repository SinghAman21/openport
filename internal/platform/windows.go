//go:build windows

package platform

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	afInet  = 2
	afInet6 = 23

	tcpTableOwnerPIDListener = 3
	udpTableOwnerPID         = 1
)

var (
	iphlpapi                = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtendedTCPTable = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUDPTable = iphlpapi.NewProc("GetExtendedUdpTable")
)

// windowsProvider uses the Windows IP Helper API. Unlike parsing netstat, these
// tables are stable, locale-independent, and include the owning PID directly.
type windowsProvider struct{ log LogFunc }

type mibTCPRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	PID        uint32
}

type mibTCP6RowOwnerPID struct {
	LocalAddr    [16]byte
	LocalScopeID uint32
	LocalPort    uint32
	RemoteAddr   [16]byte
	RemoteScope  uint32
	RemotePort   uint32
	State        uint32
	PID          uint32
}

type mibUDPRowOwnerPID struct {
	LocalAddr uint32
	LocalPort uint32
	PID       uint32
}

type mibUDP6RowOwnerPID struct {
	LocalAddr    [16]byte
	LocalScopeID uint32
	LocalPort    uint32
	PID          uint32
}

func newProvider(log LogFunc) (Provider, error) {
	if err := iphlpapi.Load(); err != nil {
		return nil, fmt.Errorf("%w: load iphlpapi.dll: %v", ErrUnsupported, err)
	}
	return &windowsProvider{log: log}, nil
}

// ListListening enumerates TCP listeners and bound UDP sockets for IPv4 and
// IPv6, then enriches each owning PID with its executable and account name.
func (p *windowsProvider) ListListening(ctx context.Context) ([]Socket, error) {
	queries := []struct {
		name string
		fn   func() ([]Socket, error)
	}{
		{"TCP/IPv4", func() ([]Socket, error) { return listWindowsTCP(afInet) }},
		{"TCP/IPv6", func() ([]Socket, error) { return listWindowsTCP(afInet6) }},
		{"UDP/IPv4", func() ([]Socket, error) { return listWindowsUDP(afInet) }},
		{"UDP/IPv6", func() ([]Socket, error) { return listWindowsUDP(afInet6) }},
	}

	var sockets []Socket
	var firstErr error
	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := q.fn()
		if err != nil {
			p.logf("warn", "could not read %s socket table: %v", q.name, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		sockets = append(sockets, rows...)
	}
	if len(sockets) == 0 && firstErr != nil {
		return nil, firstErr
	}

	p.enrichProcesses(ctx, sockets)
	sockets = dedupeSockets(sockets)
	sortSockets(sockets)
	return sockets, nil
}

func listWindowsTCP(family uint32) ([]Socket, error) {
	buf, err := readIPHelperTable(procGetExtendedTCPTable, family, tcpTableOwnerPIDListener)
	if err != nil {
		return nil, fmt.Errorf("GetExtendedTcpTable: %w", err)
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("GetExtendedTcpTable returned a truncated table")
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	base := unsafe.Add(unsafe.Pointer(&buf[0]), 4)
	out := make([]Socket, 0, count)

	if family == afInet {
		size := unsafe.Sizeof(mibTCPRowOwnerPID{})
		if !tableFits(buf, count, size) {
			return nil, fmt.Errorf("GetExtendedTcpTable returned %d truncated rows", count)
		}
		for i := uint32(0); i < count; i++ {
			row := *(*mibTCPRowOwnerPID)(unsafe.Add(base, uintptr(i)*size))
			out = append(out, Socket{Port: windowsPort(row.LocalPort), Protocol: "tcp", PID: int(row.PID), Addr: windowsIPv4(row.LocalAddr), State: "LISTEN"})
		}
		return out, nil
	}

	size := unsafe.Sizeof(mibTCP6RowOwnerPID{})
	if !tableFits(buf, count, size) {
		return nil, fmt.Errorf("GetExtendedTcpTable returned %d truncated rows", count)
	}
	for i := uint32(0); i < count; i++ {
		row := *(*mibTCP6RowOwnerPID)(unsafe.Add(base, uintptr(i)*size))
		out = append(out, Socket{Port: windowsPort(row.LocalPort), Protocol: "tcp6", PID: int(row.PID), Addr: windowsIPv6(row.LocalAddr, row.LocalScopeID), State: "LISTEN"})
	}
	return out, nil
}

func listWindowsUDP(family uint32) ([]Socket, error) {
	buf, err := readIPHelperTable(procGetExtendedUDPTable, family, udpTableOwnerPID)
	if err != nil {
		return nil, fmt.Errorf("GetExtendedUdpTable: %w", err)
	}
	if len(buf) < 4 {
		return nil, fmt.Errorf("GetExtendedUdpTable returned a truncated table")
	}
	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	base := unsafe.Add(unsafe.Pointer(&buf[0]), 4)
	out := make([]Socket, 0, count)

	if family == afInet {
		size := unsafe.Sizeof(mibUDPRowOwnerPID{})
		if !tableFits(buf, count, size) {
			return nil, fmt.Errorf("GetExtendedUdpTable returned %d truncated rows", count)
		}
		for i := uint32(0); i < count; i++ {
			row := *(*mibUDPRowOwnerPID)(unsafe.Add(base, uintptr(i)*size))
			out = append(out, Socket{Port: windowsPort(row.LocalPort), Protocol: "udp", PID: int(row.PID), Addr: windowsIPv4(row.LocalAddr), State: "OPEN"})
		}
		return out, nil
	}

	size := unsafe.Sizeof(mibUDP6RowOwnerPID{})
	if !tableFits(buf, count, size) {
		return nil, fmt.Errorf("GetExtendedUdpTable returned %d truncated rows", count)
	}
	for i := uint32(0); i < count; i++ {
		row := *(*mibUDP6RowOwnerPID)(unsafe.Add(base, uintptr(i)*size))
		out = append(out, Socket{Port: windowsPort(row.LocalPort), Protocol: "udp6", PID: int(row.PID), Addr: windowsIPv6(row.LocalAddr, row.LocalScopeID), State: "OPEN"})
	}
	return out, nil
}

func readIPHelperTable(proc *windows.LazyProc, family, tableClass uint32) ([]byte, error) {
	var size uint32
	r1, _, _ := proc.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(family), uintptr(tableClass), 0)
	if r1 != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) && r1 != 0 {
		return nil, syscall.Errno(r1)
	}
	if size < 4 {
		size = 4
	}
	for attempts := 0; attempts < 3; attempts++ {
		buf := make([]byte, size)
		r1, _, _ = proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(family), uintptr(tableClass), 0)
		if r1 == 0 {
			return buf[:size], nil
		}
		if r1 != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			return nil, syscall.Errno(r1)
		}
	}
	return nil, fmt.Errorf("socket table kept changing size")
}

func tableFits(buf []byte, count uint32, rowSize uintptr) bool {
	return uint64(count)*uint64(rowSize) <= uint64(len(buf)-4)
}

func windowsPort(v uint32) int {
	v &= 0xffff
	return int((v&0xff)<<8 | (v >> 8 & 0xff))
}

func windowsIPv4(v uint32) string {
	return net.IPv4(byte(v), byte(v>>8), byte(v>>16), byte(v>>24)).String()
}

func windowsIPv6(addr [16]byte, scope uint32) string {
	s := net.IP(addr[:]).String()
	if scope != 0 && strings.HasPrefix(s, "fe80:") {
		s = fmt.Sprintf("%s%%%d", s, scope)
	}
	return s
}

func (p *windowsProvider) enrichProcesses(ctx context.Context, sockets []Socket) {
	type metadata struct{ process, command, user string }
	cache := make(map[int]metadata)
	for i := range sockets {
		if err := ctx.Err(); err != nil {
			return
		}
		pid := sockets[i].PID
		if pid <= 0 {
			continue
		}
		m, ok := cache[pid]
		if !ok {
			m = windowsProcessMetadata(pid)
			cache[pid] = m
		}
		sockets[i].Process, sockets[i].Command, sockets[i].User = m.process, m.command, m.user
	}
}

func windowsProcessMetadata(pid int) (m struct{ process, command, user string }) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return m
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &size) == nil {
		m.command = windows.UTF16ToString(buf[:size])
		m.process = filepath.Base(m.command)
	}

	var token windows.Token
	if windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token) == nil {
		defer token.Close()
		if tu, err := token.GetTokenUser(); err == nil {
			if account, domain, _, err := tu.User.Sid.LookupAccount(""); err == nil {
				m.user = account
				if domain != "" {
					m.user = domain + `\` + account
				}
			}
		}
	}
	return m
}

// ProcessOnPort returns the listener on a port, preferring TCP and a resolved PID.
func (p *windowsProvider) ProcessOnPort(ctx context.Context, port int) (Socket, error) {
	all, err := p.ListListening(ctx)
	if err != nil {
		return Socket{}, err
	}
	var best *Socket
	for i := range all {
		if all[i].Port != port {
			continue
		}
		s := all[i]
		if best == nil || (best.PID == 0 && s.PID != 0) ||
			(strings.HasPrefix(s.Protocol, "tcp") && !strings.HasPrefix(best.Protocol, "tcp")) {
			best = &s
		}
	}
	if best == nil {
		return Socket{}, ErrNotFound
	}
	return *best, nil
}

func (p *windowsProvider) Kill(ctx context.Context, pid int, sig Signal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return killPID(pid, sig)
}

func (p *windowsProvider) logf(level, format string, args ...any) {
	LogFunc(p.log).logf(level, format, args...)
}

func windowsProcessError(op string, pid int, err error) error {
	switch {
	case errors.Is(err, windows.ERROR_ACCESS_DENIED):
		return fmt.Errorf("%w (pid %d)", ErrPermission, pid)
	case errors.Is(err, windows.ERROR_INVALID_PARAMETER):
		return fmt.Errorf("%w: pid %d already exited", ErrNotFound, pid)
	default:
		return fmt.Errorf("%s pid %d: %w", op, pid, err)
	}
}
