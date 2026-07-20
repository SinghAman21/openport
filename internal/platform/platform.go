// Package platform is the only place in OpenPort that talks to the operating
// system. Every OS-specific mechanism (/proc parsing, lsof, ss, signals) lives
// behind the Provider interface and is selected by build tag. Nothing outside
// this package may shell out to lsof/ss or read /proc.
package platform

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Provider enumerates and acts on listening sockets for the host OS.
type Provider interface {
	ListListening(ctx context.Context) ([]Socket, error)
	ProcessOnPort(ctx context.Context, port int) (Socket, error)
	Kill(ctx context.Context, pid int, sig Signal) error
}

// Socket is one listening endpoint plus the process behind it.
type Socket struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp, tcp6, udp, udp6
	PID      int    `json:"pid"`
	Process  string `json:"process"`           // executable name
	Command  string `json:"command,omitempty"` // full command line, best-effort
	User     string `json:"user,omitempty"`
	Addr     string `json:"addr"`  // 127.0.0.1, 0.0.0.0, ::
	State    string `json:"state"` // LISTEN, ESTABLISHED, ...
}

// Signal is the OS-agnostic termination request.
type Signal int

const (
	// Terminate asks the process to shut down (SIGTERM).
	Terminate Signal = iota
	// Force kills the process outright (SIGKILL).
	Force
)

// String renders the signal for logs and history entries.
func (s Signal) String() string {
	if s == Force {
		return "SIGKILL"
	}
	return "SIGTERM"
}

// LogFunc receives diagnostics from the platform layer — fallbacks taken,
// permission errors, unparseable rows. It may be nil. Injecting it here keeps
// the service and UI unaware of which implementation actually ran.
type LogFunc func(level, message string)

func (f LogFunc) logf(level, format string, args ...any) {
	if f == nil {
		return
	}
	f(level, fmt.Sprintf(format, args...))
}

// Detect returns the Provider compiled in for this OS. log may be nil.
func Detect(log LogFunc) (Provider, error) { return newProvider(log) }

// Sentinel errors callers may compare against.
var (
	ErrNotFound    = errors.New("no process on port")
	ErrUnsupported = errors.New("platform not supported")
	ErrPermission  = errors.New("permission denied: try running with elevated privileges")
)

// sortSockets orders sockets by port, then protocol, then PID so listings are
// stable across refreshes — important for watch mode not to jitter.
func sortSockets(s []Socket) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Port != s[j].Port {
			return s[i].Port < s[j].Port
		}
		if s[i].Protocol != s[j].Protocol {
			return s[i].Protocol < s[j].Protocol
		}
		return s[i].PID < s[j].PID
	})
}

// dedupeSockets collapses every row describing the same listener into one.
//
// Both /proc and lsof report a socket per file descriptor and per bound
// address, so a single process listening on one port across several
// interfaces shows up many times — a service bound to each Docker bridge plus
// a multicast group can easily produce a dozen identical-looking rows. For a
// port cockpit that is noise: what the user needs to know is "this process
// holds this port". Rows are keyed on port/protocol/PID, which still keeps
// genuinely distinct processes apart (SO_REUSEPORT workers, say).
func dedupeSockets(in []Socket) []Socket {
	type key struct {
		port  int
		proto string
		pid   int
	}
	index := make(map[key]int, len(in))
	out := make([]Socket, 0, len(in))

	for _, s := range in {
		k := key{s.Port, s.Protocol, s.PID}
		i, seen := index[k]
		if !seen {
			index[k] = len(out)
			out = append(out, s)
			continue
		}
		// Merge: keep the most informative address, and fill in any process
		// metadata this row resolved and the kept one did not.
		if betterAddr(s.Addr, out[i].Addr) {
			out[i].Addr = s.Addr
		}
		if out[i].Process == "" {
			out[i].Process = s.Process
		}
		if out[i].Command == "" {
			out[i].Command = s.Command
		}
		if out[i].User == "" {
			out[i].User = s.User
		}
	}
	return out
}

// addrRank orders bind addresses by how much they tell the user: a wildcard
// bind is the headline fact, loopback next, a specific interface last.
func addrRank(addr string) int {
	switch addr {
	case "0.0.0.0", "::":
		return 0
	case "127.0.0.1", "::1":
		return 1
	default:
		return 2
	}
}

// betterAddr reports whether candidate should replace current. Ties break
// lexicographically so the choice is deterministic — otherwise the address
// column would flicker between watch-mode ticks.
func betterAddr(candidate, current string) bool {
	cr, rr := addrRank(candidate), addrRank(current)
	if cr != rr {
		return cr < rr
	}
	return candidate < current
}

// firstField returns the first whitespace-separated token of s.
func firstField(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
