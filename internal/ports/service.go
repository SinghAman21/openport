package ports

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	"github.com/SinghAman21/openport/internal/store"
)

// service is the one implementation of Service. Both front-ends go through it,
// so protection, history and enrichment can never be bypassed by using a
// different entry point.
type service struct {
	prov  platform.Provider
	store store.Store

	// Injected seams so tests never touch the real OS or network.
	now     func() time.Time
	canBind func(port int) bool
	probe   prober
}

// NewService wires the core service to a platform provider and a store.
func NewService(prov platform.Provider, st store.Store) Service {
	return &service{
		prov:    prov,
		store:   st,
		now:     time.Now,
		canBind: canBindLocal,
		probe:   netProber{},
	}
}

// enrichment is the store-side context needed to decorate raw sockets. It is
// loaded once per operation rather than per row.
type enrichment struct {
	aliases   map[int]string
	groups    map[int][]string
	protected model.ProtectedPorts
}

// loadEnrichment reads aliases, groups and the protected list. Store failures
// degrade to empty enrichment rather than failing the whole listing — a
// corrupt aliases file should not stop you seeing your ports.
func (s *service) loadEnrichment() enrichment {
	e := enrichment{
		aliases: map[int]string{},
		groups:  map[int][]string{},
	}
	if aliases, err := s.store.Aliases(); err == nil {
		for _, a := range aliases {
			e.aliases[a.Port] = a.Name
		}
	} else {
		s.logf(model.LogWarn, "aliases unavailable: %v", err)
	}
	if groups, err := s.store.Groups(); err == nil {
		for _, g := range groups {
			for _, p := range g.Ports {
				e.groups[p] = append(e.groups[p], g.Name)
			}
		}
	} else {
		s.logf(model.LogWarn, "groups unavailable: %v", err)
	}
	if prot, err := s.store.ProtectedPorts(); err == nil {
		e.protected = prot
	} else {
		s.logf(model.LogWarn, "protected ports unavailable, assuming defaults: %v", err)
		e.protected = model.DefaultProtectedPorts()
	}
	return e
}

// decorate turns a raw socket into a Port using preloaded enrichment.
func (e enrichment) decorate(sock platform.Socket) Port {
	p := Port{Socket: sock}
	p.Alias = e.aliases[sock.Port]
	if g, ok := e.groups[sock.Port]; ok {
		p.Groups = append([]string(nil), g...)
		sort.Strings(p.Groups)
	}
	p.Protected, p.Reason = e.protected.IsProtected(sock.Port, sock.Process)
	return p
}

// List returns every listening port, enriched with aliases, groups and
// protection state.
func (s *service) List(ctx context.Context) ([]Port, error) {
	socks, err := s.prov.ListListening(ctx)
	if err != nil {
		return nil, err
	}
	e := s.loadEnrichment()
	out := make([]Port, 0, len(socks))
	for _, sock := range socks {
		out = append(out, e.decorate(sock))
	}
	return out, nil
}

// Check inspects a single port. The bool reports whether anything is listening.
func (s *service) Check(ctx context.Context, port int) (Port, bool, error) {
	if err := model.ValidatePort(port); err != nil {
		return Port{}, false, err
	}
	sock, err := s.prov.ProcessOnPort(ctx, port)
	if errors.Is(err, platform.ErrNotFound) {
		// Nothing listening is a valid answer, not a failure. Still report
		// protection so the UI can warn before the user tries to claim it.
		e := s.loadEnrichment()
		p := Port{Socket: platform.Socket{Port: port}}
		p.Alias = e.aliases[port]
		p.Groups = e.groups[port]
		p.Protected, p.Reason = e.protected.IsProtected(port, "")
		return p, false, nil
	}
	if err != nil {
		return Port{}, false, err
	}
	return s.loadEnrichment().decorate(sock), true, nil
}

// Kill terminates whatever holds a port. It can never take down a protected
// port — that requires KillWith and an explicit override.
func (s *service) Kill(ctx context.Context, port int, force bool) (KillResult, error) {
	return s.KillWith(ctx, port, KillOptions{Force: force})
}

// KillWith terminates whatever holds a port, honouring the override flag.
func (s *service) KillWith(ctx context.Context, port int, opts KillOptions) (KillResult, error) {
	if err := model.ValidatePort(port); err != nil {
		return KillResult{Port: port}, err
	}
	e := s.loadEnrichment()

	sock, err := s.prov.ProcessOnPort(ctx, port)
	if errors.Is(err, platform.ErrNotFound) {
		res := KillResult{Port: port, Skipped: true, Reason: "nothing listening"}
		s.recordKill(model.ActionKill, res)
		return res, nil
	}
	if err != nil {
		return KillResult{Port: port}, err
	}

	res := s.killSocket(ctx, sock, e, opts)
	s.recordKill(model.ActionKill, res)
	return res, nil
}

// KillPID terminates a process by PID, still honouring process-name protection
// when the PID corresponds to a known listener.
func (s *service) KillPID(ctx context.Context, pid int, force bool) (KillResult, error) {
	if pid <= 0 {
		return KillResult{PID: pid}, fmt.Errorf("invalid pid %d", pid)
	}
	e := s.loadEnrichment()

	// Look the PID up among listeners so we can report the port it held and
	// apply protection rules.
	res := KillResult{PID: pid, Forced: force}
	if socks, err := s.prov.ListListening(ctx); err == nil {
		for _, sock := range socks {
			if sock.PID == pid {
				res.Port, res.Process = sock.Port, sock.Process
				if protected, reason := e.protected.IsProtected(sock.Port, sock.Process); protected {
					res.Skipped, res.Reason = true, reason
					s.recordKill(model.ActionKillPID, res)
					return res, nil
				}
				break
			}
		}
	}

	sig := platform.Terminate
	if force {
		sig = platform.Force
	}
	res.Err = s.prov.Kill(ctx, pid, sig)
	s.recordKill(model.ActionKillPID, res)
	return res, nil
}

// KillRange terminates every listener in an inclusive port range. Protected
// ports are skipped with a per-row reason rather than aborting the batch.
func (s *service) KillRange(ctx context.Context, lo, hi int, force bool) ([]KillResult, error) {
	return s.KillRangeWith(ctx, lo, hi, KillOptions{Force: force})
}

// KillRangeWith is KillRange with the protected override threaded through.
func (s *service) KillRangeWith(ctx context.Context, lo, hi int, opts KillOptions) ([]KillResult, error) {
	if err := model.ValidatePort(lo); err != nil {
		return nil, fmt.Errorf("low bound: %w", err)
	}
	if err := model.ValidatePort(hi); err != nil {
		return nil, fmt.Errorf("high bound: %w", err)
	}
	if lo > hi {
		lo, hi = hi, lo
	}

	// One listing beats up to 65535 per-port lookups.
	socks, err := s.prov.ListListening(ctx)
	if err != nil {
		return nil, err
	}
	e := s.loadEnrichment()

	seen := map[int]bool{}
	results := make([]KillResult, 0, 8)
	for _, sock := range socks {
		if sock.Port < lo || sock.Port > hi || seen[sock.Port] {
			continue
		}
		seen[sock.Port] = true
		results = append(results, s.killSocket(ctx, sock, e, opts))
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Port < results[j].Port })
	s.recordRange(lo, hi, opts, results)
	return results, nil
}

// killSocket is the single place a signal is actually sent. Every kill path
// funnels through here, so the protection guard cannot be routed around.
func (s *service) killSocket(ctx context.Context, sock platform.Socket, e enrichment, opts KillOptions) KillResult {
	res := KillResult{
		Port:    sock.Port,
		PID:     sock.PID,
		Process: sock.Process,
		Forced:  opts.Force,
	}

	if protected, reason := e.protected.IsProtected(sock.Port, sock.Process); protected && !opts.OverrideProtected {
		res.Skipped, res.Reason = true, reason
		return res
	}
	if sock.PID <= 0 {
		res.Skipped = true
		res.Reason = "owning process not visible (try elevated privileges)"
		return res
	}

	sig := platform.Terminate
	if opts.Force {
		sig = platform.Force
	}
	if err := s.prov.Kill(ctx, sock.PID, sig); err != nil {
		res.Err = err
	}
	return res
}

// Reserve claims a port for a service by name. Reservations are stored as
// aliases, which is also what makes them visible in the ports table and
// excluded from future suggestions.
func (s *service) Reserve(ctx context.Context, port int, service string) error {
	if err := model.ValidatePort(port); err != nil {
		return err
	}
	service = strings.TrimSpace(service)
	if service == "" {
		return fmt.Errorf("a service name is required to reserve a port")
	}
	alias := model.Alias{
		Name:      service,
		Port:      port,
		Note:      "reserved",
		CreatedAt: s.now(),
	}
	if err := alias.Validate(); err != nil {
		return err
	}
	err := s.store.SaveAlias(alias)
	s.record(model.HistoryEntry{
		At:      s.now(),
		Kind:    model.ActionReserve,
		Summary: fmt.Sprintf("reserved port %d for %s", port, service),
		Port:    port,
		Success: err == nil,
		Err:     errString(err),
	})
	return err
}

// --- history helpers -----------------------------------------------------

func (s *service) record(entry model.HistoryEntry) {
	if err := s.store.AppendHistory(entry); err != nil {
		s.logf(model.LogWarn, "could not write history: %v", err)
	}
}

func (s *service) recordKill(kind model.ActionKind, res KillResult) {
	entry := model.HistoryEntry{
		At:      s.now(),
		Kind:    kind,
		Port:    res.Port,
		PID:     res.PID,
		Process: res.Process,
		Success: res.OK(),
		Err:     errString(res.Err),
	}
	signal := "SIGTERM"
	if res.Forced {
		signal = "SIGKILL"
	}
	switch {
	case res.Skipped:
		entry.Summary = fmt.Sprintf("skipped port %d", res.Port)
		entry.Detail = res.Reason
	case res.Err != nil:
		entry.Summary = fmt.Sprintf("failed to kill port %d", res.Port)
		entry.Detail = signal
	default:
		entry.Summary = fmt.Sprintf("killed %s on port %d", displayName(res.Process), res.Port)
		entry.Detail = fmt.Sprintf("%s to pid %d", signal, res.PID)
	}
	s.record(entry)
}

func (s *service) recordRange(lo, hi int, opts KillOptions, results []KillResult) {
	var killed, skipped, failed int
	for _, r := range results {
		switch {
		case r.Skipped:
			skipped++
		case r.Err != nil:
			failed++
		default:
			killed++
		}
	}
	signal := "SIGTERM"
	if opts.Force {
		signal = "SIGKILL"
	}
	s.record(model.HistoryEntry{
		At:      s.now(),
		Kind:    model.ActionKillRange,
		Summary: fmt.Sprintf("kill range %d-%d: %d killed, %d skipped, %d failed", lo, hi, killed, skipped, failed),
		Detail:  signal,
		Success: failed == 0,
	})
}

func (s *service) logf(level model.LogLevel, format string, args ...any) {
	_ = s.store.AppendLog(model.LogEntry{
		At:      s.now(),
		Level:   level,
		Source:  "service",
		Message: fmt.Sprintf(format, args...),
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func displayName(process string) string {
	if process == "" {
		return "process"
	}
	return process
}
