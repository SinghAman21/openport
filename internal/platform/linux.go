//go:build linux

package platform

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// linuxProvider reads /proc directly — no external binaries required — and
// falls back to `ss` only when /proc is unavailable (containers with a masked
// procfs, hardened kernels).
type linuxProvider struct {
	log  LogFunc
	root string // procfs mount point; a field so tests can point elsewhere

	mu           sync.Mutex
	users        map[string]string // uid -> username
	warnedProc   bool
	warnedFDPerm bool
}

func newProvider(log LogFunc) (Provider, error) {
	return &linuxProvider{root: "/proc", log: log, users: map[string]string{}}, nil
}

// procNetRow is one parsed line of /proc/net/{tcp,tcp6,udp,udp6}.
type procNetRow struct {
	Addr  string
	Port  int
	State string
	UID   string
	Inode string
}

// TCP/UDP socket states as reported by procfs, hex-encoded.
const (
	tcpStateListen = "0A"
	udpStateUnconn = "07" // a bound-but-unconnected UDP socket is a listener
)

// tcpStateNames maps the procfs state byte to the name users expect to see.
var tcpStateNames = map[string]string{
	"01": "ESTABLISHED", "02": "SYN_SENT", "03": "SYN_RECV", "04": "FIN_WAIT1",
	"05": "FIN_WAIT2", "06": "TIME_WAIT", "07": "CLOSE", "08": "CLOSE_WAIT",
	"09": "LAST_ACK", "0A": "LISTEN", "0B": "CLOSING",
}

// parseProcNet decodes a /proc/net/* table. It is pure so it can be tested
// against fixtures without a live procfs.
func parseProcNet(r io.Reader) []procNetRow {
	var rows []procNetRow
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if first { // header
			first = false
			continue
		}
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 10 {
			continue
		}
		addr, port, err := parseHexAddrPort(f[1])
		if err != nil {
			continue
		}
		rows = append(rows, procNetRow{
			Addr:  addr,
			Port:  port,
			State: strings.ToUpper(f[3]),
			UID:   f[7],
			Inode: f[9],
		})
	}
	return rows
}

// parseHexAddrPort decodes procfs' "0100007F:1F90" into "127.0.0.1", 8080.
// Addresses are hex-encoded 32-bit words in host (little-endian) byte order,
// so each 4-byte word has to be reversed; the port is plain big-endian.
func parseHexAddrPort(s string) (string, int, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", 0, fmt.Errorf("malformed address %q", s)
	}
	rawAddr, rawPort := s[:i], s[i+1:]

	port64, err := strconv.ParseUint(rawPort, 16, 32)
	if err != nil {
		return "", 0, fmt.Errorf("malformed port %q: %w", rawPort, err)
	}

	b, err := hex.DecodeString(rawAddr)
	if err != nil || (len(b) != 4 && len(b) != 16) {
		return "", 0, fmt.Errorf("malformed address %q", rawAddr)
	}
	for w := 0; w < len(b); w += 4 { // reverse each 32-bit word
		b[w], b[w+1], b[w+2], b[w+3] = b[w+3], b[w+2], b[w+1], b[w]
	}
	return net.IP(b).String(), int(port64), nil
}

// listeners returns the listening rows for one procfs table.
func (p *linuxProvider) listeners(name, proto string) ([]procNetRow, error) {
	f, err := os.Open(filepath.Join(p.root, "net", name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	want := tcpStateListen
	if strings.HasPrefix(proto, "udp") {
		want = udpStateUnconn
	}
	var out []procNetRow
	for _, row := range parseProcNet(f) {
		if row.State == want {
			out = append(out, row)
		}
	}
	return out, nil
}

// ListListening enumerates every listening socket on the host.
func (p *linuxProvider) ListListening(ctx context.Context) ([]Socket, error) {
	tables := []struct{ file, proto string }{
		{"tcp", "tcp"}, {"tcp6", "tcp6"}, {"udp", "udp"}, {"udp6", "udp6"},
	}

	var sockets []Socket
	var readAny bool
	var firstErr error

	for _, t := range tables {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := p.listeners(t.file, t.proto)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		readAny = true
		for _, row := range rows {
			state := "LISTEN"
			if strings.HasPrefix(t.proto, "udp") {
				state = "OPEN"
			} else if n, ok := tcpStateNames[row.State]; ok {
				state = n
			}
			sockets = append(sockets, Socket{
				Port:     row.Port,
				Protocol: t.proto,
				Addr:     row.Addr,
				State:    state,
				User:     p.lookupUser(row.UID),
				PID:      -1, // resolved below via the inode map
			})
			// Stash the inode in Command temporarily; resolveOwners clears it.
			sockets[len(sockets)-1].Command = row.Inode
		}
	}

	if !readAny {
		p.warnOnce(&p.warnedProc, "warn",
			"procfs unreadable (%v) — falling back to `ss`", firstErr)
		return p.listViaSS(ctx)
	}

	p.resolveOwners(ctx, sockets)
	sockets = dedupeSockets(sockets)
	sortSockets(sockets)
	return sockets, nil
}

// resolveOwners maps socket inodes to the processes holding them by walking
// /proc/<pid>/fd. Sockets we cannot attribute (usually another user's, without
// root) keep PID 0 rather than failing the whole listing.
func (p *linuxProvider) resolveOwners(ctx context.Context, sockets []Socket) {
	want := make(map[string]struct{}, len(sockets))
	for _, s := range sockets {
		if s.Command != "" {
			want[s.Command] = struct{}{}
		}
	}
	owners := p.inodeOwners(ctx, want)

	for i := range sockets {
		inode := sockets[i].Command
		sockets[i].Command = ""
		sockets[i].PID = 0
		if inode == "" {
			continue
		}
		pid, ok := owners[inode]
		if !ok {
			continue
		}
		sockets[i].PID = pid
		sockets[i].Process = p.processName(pid)
		sockets[i].Command = p.processCmdline(pid)
	}
}

// inodeOwners builds inode -> pid for the inodes we care about.
func (p *linuxProvider) inodeOwners(ctx context.Context, want map[string]struct{}) map[string]int {
	out := make(map[string]int, len(want))
	if len(want) == 0 {
		return out
	}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		p.logf("warn", "cannot read %s: %v", p.root, err)
		return out
	}

	var denied int
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return out
		}
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process directory
		}
		fdDir := filepath.Join(p.root, e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			if os.IsPermission(err) {
				denied++
			}
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			inode, ok := socketInode(link)
			if !ok {
				continue
			}
			if _, care := want[inode]; care {
				out[inode] = pid
			}
		}
		if len(out) == len(want) {
			break // every socket attributed; no need to scan the rest
		}
	}

	if denied > 0 && len(out) < len(want) {
		p.warnOnce(&p.warnedFDPerm, "warn",
			"could not inspect %d process(es) — some ports will show no PID; "+
				"run with elevated privileges to see them all", denied)
	}
	return out
}

// socketInode extracts "12345" from a "socket:[12345]" fd symlink.
func socketInode(link string) (string, bool) {
	const prefix, suffix = "socket:[", "]"
	if !strings.HasPrefix(link, prefix) || !strings.HasSuffix(link, suffix) {
		return "", false
	}
	return link[len(prefix) : len(link)-len(suffix)], true
}

func (p *linuxProvider) processName(pid int) string {
	b, err := os.ReadFile(filepath.Join(p.root, strconv.Itoa(pid), "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (p *linuxProvider) processCmdline(pid int) string {
	b, err := os.ReadFile(filepath.Join(p.root, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(b) == 0 {
		return ""
	}
	// cmdline is NUL-separated and usually NUL-terminated.
	parts := strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
	return strings.Join(parts, " ")
}

func (p *linuxProvider) lookupUser(uid string) string {
	if uid == "" {
		return ""
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if name, ok := p.users[uid]; ok {
		return name
	}
	name := uid
	if u, err := user.LookupId(uid); err == nil {
		name = u.Username
	}
	p.users[uid] = name
	return name
}

// ProcessOnPort returns the listener on a port, preferring TCP over UDP and a
// resolved PID over an unattributable one.
func (p *linuxProvider) ProcessOnPort(ctx context.Context, port int) (Socket, error) {
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

// Kill signals a process. Defined in kill.go, shared with darwin.
func (p *linuxProvider) Kill(ctx context.Context, pid int, sig Signal) error {
	return killPID(pid, sig)
}

// --- `ss` fallback -------------------------------------------------------
//
// Only reached when procfs is unreadable. Everything above is std-lib.

var ssLineRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+\S+\s+\S+\s+(\S+)\s+\S+(?:\s+(.*))?$`)
var ssProcRe = regexp.MustCompile(`\("([^"]+)",pid=(\d+)`)

func (p *linuxProvider) listViaSS(ctx context.Context) ([]Socket, error) {
	out, err := exec.CommandContext(ctx, "ss", "-tulpnH").Output()
	if err != nil {
		// Older ss lacks -H; retry and skip the header ourselves.
		out, err = exec.CommandContext(ctx, "ss", "-tulpn").Output()
		if err != nil {
			return nil, fmt.Errorf("%w: procfs unreadable and `ss` failed: %v", ErrUnsupported, err)
		}
	}
	sockets := parseSSOutput(string(out))
	sockets = dedupeSockets(sockets)
	sortSockets(sockets)
	return sockets, nil
}

// parseSSOutput decodes `ss -tulpn` rows. Pure, so it is unit-tested directly.
func parseSSOutput(out string) []Socket {
	var sockets []Socket
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Netid") {
			continue
		}
		m := ssLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		netid, state, local, procs := m[1], m[2], m[3], m[4]

		addr, port, ok := splitHostPort(local)
		if !ok {
			continue
		}
		proto := netid
		if strings.Contains(addr, ":") && !strings.HasSuffix(proto, "6") {
			proto += "6" // ss reports v6 rows with the same netid
		}
		if strings.HasPrefix(netid, "udp") {
			state = "OPEN"
		}

		s := Socket{Port: port, Protocol: proto, Addr: addr, State: strings.ToUpper(state)}
		if pm := ssProcRe.FindStringSubmatch(procs); pm != nil {
			s.Process = pm[1]
			s.PID, _ = strconv.Atoi(pm[2])
		}
		sockets = append(sockets, s)
	}
	return sockets
}

// splitHostPort handles ss' "0.0.0.0:8080", "[::]:8080" and "*:8080" forms.
func splitHostPort(s string) (string, int, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", 0, false
	}
	host, rawPort := s[:i], s[i+1:]
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return "", 0, false
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if host == "*" {
		host = "0.0.0.0"
	}
	return host, port, true
}

func (p *linuxProvider) logf(level, format string, args ...any) {
	LogFunc(p.log).logf(level, format, args...)
}

// warnOnce emits a diagnostic the first time a condition is hit, so a
// permission problem does not spam the log on every watch tick.
func (p *linuxProvider) warnOnce(flag *bool, level, format string, args ...any) {
	p.mu.Lock()
	already := *flag
	*flag = true
	p.mu.Unlock()
	if !already {
		p.logf(level, format, args...)
	}
}
