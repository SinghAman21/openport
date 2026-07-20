// Package blueprint implements applying an environment template against live
// state. It sits between the store (which holds blueprints) and ports.Service
// (which knows what is running), so the TUI and the headless CLI share one
// implementation rather than each growing their own.
package blueprint

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/store"
)

// Status values a service can end up in after an apply.
const (
	// StatusOK — the expected service is listening on its port.
	StatusOK = "ok"
	// StatusFree — nothing is on the port; the service is simply not running.
	StatusFree = "free"
	// StatusConflict — something else has taken the port.
	StatusConflict = "conflict"
)

// Result is one service's outcome.
type Result struct {
	Service string
	Wanted  int
	Actual  int
	Process string
	Status  string
	Detail  string
	// Suggested is a free port offered when the wanted one is taken. Applying
	// never moves anything on its own — it tells you what it would take.
	Suggested int
}

// OK reports whether the service is where the blueprint says it should be.
func (r Result) OK() bool { return r.Status == StatusOK }

// Apply walks a blueprint against live state and reports what matches, what is
// missing, and what has taken a port it should not have. It is read-only with
// respect to processes: nothing is started or killed, because guessing at
// either is how a tool like this loses your trust.
func Apply(ctx context.Context, svc ports.Service, st store.Store, name string) ([]Result, error) {
	bp, err := find(st, name)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(bp.Services))
	for _, s := range bp.Services {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		r := Result{Service: s.Name, Wanted: s.Port}
		port, found, err := svc.Check(ctx, s.Port)
		switch {
		case err != nil:
			r.Status = StatusConflict
			r.Detail = err.Error()

		case !found:
			r.Status = StatusFree
			r.Detail = "not running"

		case Matches(port.Alias, port.Process, s.Name):
			r.Status = StatusOK
			r.Actual = port.Port
			r.Process = port.Process
			r.Detail = fmt.Sprintf("pid %d", port.PID)

		default:
			r.Status = StatusConflict
			r.Actual = port.Port
			r.Process = port.Process
			r.Detail = fmt.Sprintf("held by %s", label(port))
			// Offer somewhere else to put it rather than just complaining.
			if free, err := svc.SuggestFree(ctx, s.Port+1, 1); err == nil && len(free) > 0 {
				r.Suggested = free[0]
			}
		}
		results = append(results, r)
	}

	record(st, bp.Name, results)
	return results, nil
}

// find looks a blueprint up by name, case-insensitively.
func find(st store.Store, name string) (model.Blueprint, error) {
	all, err := st.Blueprints()
	if err != nil {
		return model.Blueprint{}, err
	}
	for _, bp := range all {
		if strings.EqualFold(bp.Name, name) {
			return bp, nil
		}
	}
	return model.Blueprint{}, fmt.Errorf("no blueprint named %q", name)
}

// Matches is the heuristic for "is the thing on this port the service the
// blueprint meant?". It is deliberately generous — a false conflict is more
// annoying than a false match, because the holder is always shown alongside.
func Matches(alias, process, service string) bool {
	service = strings.ToLower(strings.TrimSpace(service))
	if service == "" {
		return false
	}
	if strings.EqualFold(alias, service) {
		return true
	}
	process = strings.ToLower(process)
	if process == "" {
		return false
	}
	return strings.Contains(process, service) || strings.Contains(service, process)
}

func label(p ports.Port) string {
	if p.Alias != "" {
		return p.Alias
	}
	if p.Process != "" {
		return p.Process
	}
	return "an unidentified process"
}

// record appends the audit entry for the apply.
func record(st store.Store, name string, results []Result) {
	var ok, conflict, free int
	for _, r := range results {
		switch r.Status {
		case StatusOK:
			ok++
		case StatusConflict:
			conflict++
		default:
			free++
		}
	}
	_ = st.AppendHistory(model.HistoryEntry{
		At:   time.Now(),
		Kind: model.ActionApplyBlueprint,
		Summary: fmt.Sprintf("applied blueprint %s: %d up, %d conflicts, %d not running",
			name, ok, conflict, free),
		Success: conflict == 0,
	})
}
