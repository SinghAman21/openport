//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// darwinProvider drives lsof in field-output mode (-F), which is stable across
// macOS releases in a way the human-readable columns are not.
type darwinProvider struct {
	log LogFunc

	mu         sync.Mutex
	warnedPerm bool
}

func newProvider(log LogFunc) (Provider, error) {
	if _, err := exec.LookPath("lsof"); err != nil {
		return nil, fmt.Errorf("%w: lsof not found in PATH", ErrUnsupported)
	}
	return &darwinProvider{log: log}, nil
}

// ListListening enumerates listening TCP sockets plus bound UDP sockets.
func (p *darwinProvider) ListListening(ctx context.Context) ([]Socket, error) {
	var sockets []Socket
	var firstErr error

	// TCP listeners.
	if out, err := p.run(ctx, "-nP", "-iTCP", "-sTCP:LISTEN", "-FpcLnPT"); err == nil {
		sockets = append(sockets, parseLsofFields(out, "LISTEN")...)
	} else {
		firstErr = err
	}

	// UDP sockets have no listen state; a bound socket is the closest analogue.
	if out, err := p.run(ctx, "-nP", "-iUDP", "-FpcLnP"); err == nil {
		sockets = append(sockets, parseLsofFields(out, "OPEN")...)
	} else if firstErr == nil {
		firstErr = err
	}

	if len(sockets) == 0 && firstErr != nil {
		return nil, firstErr
	}

	p.enrichCommands(ctx, sockets)
	sockets = dedupeSockets(sockets)
	sortSockets(sockets)
	return sockets, nil
}

// run executes lsof. lsof exits 1 when it merely has nothing to report, so a
// non-empty stdout is treated as success regardless of exit code.
func (p *darwinProvider) run(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "lsof", args...).Output()
	if err != nil && len(out) == 0 {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("lsof %s: %w", strings.Join(args, " "), err)
	}
	if err != nil {
		p.warnOnce("warn", "lsof reported partial results (%v) — some ports may be "+
			"missing; run with elevated privileges to see them all", err)
	}
	return string(out), nil
}

// parseLsofFields decodes lsof -F output. The format is a stream of records:
// 'p' opens a process block, 'f' opens a file within it, and the remaining
// letters are attributes of whichever block is currently open. Pure, so it is
// unit-tested against fixtures.
func parseLsofFields(out, defaultState string) []Socket {
	var (
		sockets           []Socket
		pid               int
		process, username string
		cur               Socket
		open              bool
	)

	flush := func() {
		if open && cur.Port > 0 {
			cur.PID = pid
			cur.Process = process
			cur.User = username
			if cur.State == "" {
				cur.State = defaultState
			}
			sockets = append(sockets, cur)
		}
		open = false
		cur = Socket{}
	}

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		tag, val := line[0], line[1:]
		switch tag {
		case 'p':
			flush()
			pid, _ = strconv.Atoi(val)
			process, username = "", ""
		case 'c':
			process = val
		case 'L':
			username = val
		case 'f':
			flush()
			open = true
		case 'n':
			addr, port, ok := splitLsofAddr(val)
			if !ok {
				continue
			}
			cur.Addr, cur.Port = addr, port
		case 'P':
			cur.Protocol = strings.ToLower(val)
		case 'T':
			// e.g. "ST=LISTEN"
			if rest, found := strings.CutPrefix(val, "ST="); found {
				cur.State = rest
			}
		}
	}
	flush()

	// lsof reports the protocol family separately from the address family, so
	// tag v6 rows here rather than trusting the P field alone.
	for i := range sockets {
		if strings.Contains(sockets[i].Addr, ":") && !strings.HasSuffix(sockets[i].Protocol, "6") {
			sockets[i].Protocol += "6"
		}
	}
	return sockets
}

// splitLsofAddr handles "127.0.0.1:3000", "*:3000", "[::1]:3000" and the
// "1.2.3.4:80->5.6.7.8:99" peer form (which we ignore — not a listener).
func splitLsofAddr(s string) (string, int, bool) {
	if strings.Contains(s, "->") {
		return "", 0, false
	}
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", 0, false
	}
	host, rawPort := s[:i], s[i+1:]
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return "", 0, false // named service or a wildcard port
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	switch host {
	case "*":
		host = "0.0.0.0"
	case "":
		host = "0.0.0.0"
	}
	return host, port, true
}

// enrichCommands fills in full command lines, one `ps` call for all PIDs.
func (p *darwinProvider) enrichCommands(ctx context.Context, sockets []Socket) {
	pids := map[int]struct{}{}
	for _, s := range sockets {
		if s.PID > 0 {
			pids[s.PID] = struct{}{}
		}
	}
	if len(pids) == 0 {
		return
	}
	args := make([]string, 0, len(pids)+2)
	args = append(args, "-o", "pid=,command=", "-p")
	list := make([]string, 0, len(pids))
	for pid := range pids {
		list = append(list, strconv.Itoa(pid))
	}
	args = append(args, strings.Join(list, ","))

	out, err := exec.CommandContext(ctx, "ps", args...).Output()
	if err != nil {
		p.logf("warn", "could not read full command lines: %v", err)
		return
	}

	cmds := make(map[int]string, len(pids))
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pidStr := firstField(line)
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		cmds[pid] = strings.TrimSpace(line[len(pidStr):])
	}
	for i := range sockets {
		if c, ok := cmds[sockets[i].PID]; ok {
			sockets[i].Command = c
		}
	}
}

// ProcessOnPort returns the listener on a port, preferring TCP and a resolved
// PID.
func (p *darwinProvider) ProcessOnPort(ctx context.Context, port int) (Socket, error) {
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

// Kill signals a process. Defined in kill.go, shared with linux.
func (p *darwinProvider) Kill(ctx context.Context, pid int, sig Signal) error {
	return killPID(pid, sig)
}

func (p *darwinProvider) logf(level, format string, args ...any) {
	LogFunc(p.log).logf(level, format, args...)
}

func (p *darwinProvider) warnOnce(level, format string, args ...any) {
	p.mu.Lock()
	already := p.warnedPerm
	p.warnedPerm = true
	p.mu.Unlock()
	if !already {
		p.logf(level, format, args...)
	}
}
