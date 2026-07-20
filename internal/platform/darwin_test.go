//go:build darwin

package platform

import "testing"

// lsofFixture is verbatim `lsof -nP -iTCP -sTCP:LISTEN -FpcLnPT` output for two
// processes, one of which holds two sockets.
const lsofFixture = `p1234
cnode
Laman
f21
nTCP
PTCP
n127.0.0.1:3000
TST=LISTEN
f22
PTCP
n[::1]:3000
TST=LISTEN
p5678
credis-server
Laman
f6
PTCP
n127.0.0.1:6379
TST=LISTEN
`

func TestParseLsofFields(t *testing.T) {
	got := parseLsofFields(lsofFixture, "LISTEN")
	if len(got) != 3 {
		t.Fatalf("parsed %d sockets, want 3: %+v", len(got), got)
	}

	tests := []struct {
		idx     int
		port    int
		proto   string
		pid     int
		process string
		addr    string
	}{
		{0, 3000, "tcp", 1234, "node", "127.0.0.1"},
		{1, 3000, "tcp6", 1234, "node", "::1"},
		{2, 6379, "tcp", 5678, "redis-server", "127.0.0.1"},
	}
	for _, tt := range tests {
		s := got[tt.idx]
		if s.Port != tt.port || s.Protocol != tt.proto || s.PID != tt.pid ||
			s.Process != tt.process || s.Addr != tt.addr {
			t.Errorf("socket %d = %+v; want port=%d proto=%s pid=%d process=%s addr=%s",
				tt.idx, s, tt.port, tt.proto, tt.pid, tt.process, tt.addr)
		}
		if s.User != "aman" {
			t.Errorf("socket %d user = %q; want %q", tt.idx, s.User, "aman")
		}
		if s.State != "LISTEN" {
			t.Errorf("socket %d state = %q; want LISTEN", tt.idx, s.State)
		}
	}
}

func TestParseLsofFieldsDefaultState(t *testing.T) {
	// UDP output carries no T field, so the default state must be applied.
	const udp = `p812
cmDNSResponder
Laman
f8
PUDP
n*:5353
`
	got := parseLsofFields(udp, "OPEN")
	if len(got) != 1 {
		t.Fatalf("parsed %d sockets, want 1: %+v", len(got), got)
	}
	if got[0].State != "OPEN" {
		t.Errorf("state = %q; want OPEN", got[0].State)
	}
	if got[0].Addr != "0.0.0.0" || got[0].Port != 5353 {
		t.Errorf("addr/port = %q/%d; want 0.0.0.0/5353", got[0].Addr, got[0].Port)
	}
	if got[0].Protocol != "udp" {
		t.Errorf("protocol = %q; want udp", got[0].Protocol)
	}
}

func TestParseLsofFieldsEmpty(t *testing.T) {
	if got := parseLsofFields("", "LISTEN"); len(got) != 0 {
		t.Fatalf("parsed %d sockets from empty input, want 0", len(got))
	}
}

func TestSplitLsofAddr(t *testing.T) {
	tests := []struct {
		in       string
		wantHost string
		wantPort int
		wantOK   bool
	}{
		{"127.0.0.1:3000", "127.0.0.1", 3000, true},
		{"*:3000", "0.0.0.0", 3000, true},
		{"[::1]:8080", "::1", 8080, true},
		{"[::]:22", "::", 22, true},
		// An established connection is not a listener.
		{"127.0.0.1:3000->127.0.0.1:51234", "", 0, false},
		// A named service rather than a numeric port.
		{"127.0.0.1:http", "", 0, false},
		{"nonsense", "", 0, false},
	}
	for _, tt := range tests {
		host, port, ok := splitLsofAddr(tt.in)
		if host != tt.wantHost || port != tt.wantPort || ok != tt.wantOK {
			t.Errorf("splitLsofAddr(%q) = %q, %d, %v; want %q, %d, %v",
				tt.in, host, port, ok, tt.wantHost, tt.wantPort, tt.wantOK)
		}
	}
}

func TestDedupeAndSortSocketsDarwin(t *testing.T) {
	in := []Socket{
		{Port: 8080, Protocol: "tcp", PID: 10, Addr: "127.0.0.1"},
		{Port: 3000, Protocol: "tcp", PID: 20, Addr: "0.0.0.0"},
		{Port: 8080, Protocol: "tcp", PID: 10, Addr: "127.0.0.1"},
	}
	got := dedupeSockets(in)
	if len(got) != 2 {
		t.Fatalf("dedupeSockets returned %d sockets, want 2: %+v", len(got), got)
	}
	sortSockets(got)
	if got[0].Port != 3000 {
		t.Errorf("after sort, first port = %d; want 3000", got[0].Port)
	}
}
