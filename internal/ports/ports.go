// Package ports is the OS-agnostic core of OpenPort. Both front-ends (the TUI
// and the headless CLI) call this one Service, so there is exactly one code
// path per capability.
package ports

import (
	"context"
	"fmt"
	"time"

	"github.com/SinghAman21/openport/internal/platform"
)

// Service is every capability OpenPort exposes. Implementations enrich raw
// sockets with aliases, groups and protection state from the store, and append
// a history entry for every mutating call.
type Service interface {
	List(ctx context.Context) ([]Port, error)
	Check(ctx context.Context, port int) (Port, bool, error)

	Kill(ctx context.Context, port int, force bool) (KillResult, error)
	KillPID(ctx context.Context, pid int, force bool) (KillResult, error)
	KillRange(ctx context.Context, lo, hi int, force bool) ([]KillResult, error)

	// KillWith and KillRangeWith are the same operations with the
	// protected-port override threaded through explicitly. Kill/KillRange
	// delegate to these with OverrideProtected=false, which is why a plain
	// Kill can never take down a protected port by accident.
	KillWith(ctx context.Context, port int, opts KillOptions) (KillResult, error)
	KillRangeWith(ctx context.Context, lo, hi int, opts KillOptions) ([]KillResult, error)

	SuggestFree(ctx context.Context, near, count int) ([]int, error)
	Reserve(ctx context.Context, port int, service string) error

	Health(ctx context.Context, port int, kind HealthKind) (HealthResult, error)
	Watch(ctx context.Context, interval time.Duration) (<-chan []Port, func())
}

// Port is a listening socket enriched with everything the UI needs to render a
// row without asking any other component.
type Port struct {
	platform.Socket
	Alias     string        `json:"alias,omitempty"`
	Groups    []string      `json:"groups,omitempty"`
	Protected bool          `json:"protected"`
	Reason    string        `json:"reason,omitempty"` // why it is protected
	Health    *HealthResult `json:"health,omitempty"`
}

// Key identifies a row across refreshes. Watch mode tracks selection by this
// rather than by row index, so the cursor does not jump when the table
// reorders underneath it.
func (p Port) Key() string { return fmt.Sprintf("%d/%s/%d", p.Port, p.Protocol, p.PID) }

// Label prefers the alias when one is set, falling back to the process name.
func (p Port) Label() string {
	if p.Alias != "" {
		return p.Alias
	}
	return p.Process
}

// KillOptions carries the flags that modulate a kill.
type KillOptions struct {
	// Force sends SIGKILL instead of SIGTERM.
	Force bool
	// OverrideProtected permits killing a port on the never-kill list. The TUI
	// must double-confirm before ever setting this.
	OverrideProtected bool
}

// KillResult reports the outcome for a single port. A skipped result is not an
// error: it is the protected-port guard doing its job.
type KillResult struct {
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	Process string `json:"process"`
	Forced  bool   `json:"forced"`
	Skipped bool   `json:"skipped"`
	Reason  string `json:"reason,omitempty"`
	Err     error  `json:"-"`
}

// OK reports whether the process was actually signalled.
func (r KillResult) OK() bool { return !r.Skipped && r.Err == nil }

// HealthKind selects the probe used against a port.
type HealthKind int

const (
	// HealthTCP dials the port and reports whether the connection succeeds.
	HealthTCP HealthKind = iota
	// HealthHTTP issues a GET and reports the status code.
	HealthHTTP
)

// String renders the probe name for history entries and the UI.
func (k HealthKind) String() string {
	if k == HealthHTTP {
		return "http"
	}
	return "tcp"
}

// HealthResult is the outcome of one probe.
type HealthResult struct {
	OK        bool      `json:"ok"`
	Status    string    `json:"status"`
	LatencyMS int       `json:"latency_ms"`
	Code      int       `json:"code,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Badge renders the compact health indicator used in tables.
func (r *HealthResult) Badge() string {
	switch {
	case r == nil:
		return "·"
	case r.OK:
		return "✓"
	default:
		return "✕"
	}
}
