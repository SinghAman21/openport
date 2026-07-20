//go:build linux

package platform

import (
	"strings"
	"testing"
)

func TestParseHexAddrPort(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantAddr string
		wantPort int
		wantErr  bool
	}{
		{
			name:     "ipv4 loopback",
			in:       "0100007F:1F90",
			wantAddr: "127.0.0.1",
			wantPort: 8080,
		},
		{
			name:     "ipv4 wildcard",
			in:       "00000000:0BB8",
			wantAddr: "0.0.0.0",
			wantPort: 3000,
		},
		{
			name:     "ipv6 wildcard",
			in:       "00000000000000000000000000000000:1538",
			wantAddr: "::",
			wantPort: 5432,
		},
		{
			name:     "ipv6 loopback",
			in:       "00000000000000000000000001000000:1F90",
			wantAddr: "::1",
			wantPort: 8080,
		},
		{name: "no colon", in: "0100007F", wantErr: true},
		{name: "bad hex", in: "ZZZZZZZZ:1F90", wantErr: true},
		{name: "wrong length", in: "0100:1F90", wantErr: true},
		{name: "bad port", in: "0100007F:ZZZZ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port, err := parseHexAddrPort(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseHexAddrPort(%q) = %q, %d; want error", tt.in, addr, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHexAddrPort(%q) returned error: %v", tt.in, err)
			}
			if addr != tt.wantAddr || port != tt.wantPort {
				t.Errorf("parseHexAddrPort(%q) = %q, %d; want %q, %d",
					tt.in, addr, port, tt.wantAddr, tt.wantPort)
			}
		})
	}
}

// procNetTCPFixture is a verbatim /proc/net/tcp sample: a listener on 8080, an
// established connection (must be filtered out), and a listener on 3000.
const procNetTCPFixture = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 41253 1 0000000000000000 100 0 0 10 0
   1: 0100007F:1F90 0100007F:C1B2 01 00000000:00000000 00:00000000 00000000  1000        0 41999 1 0000000000000000 20 4 30 10 -1
   2: 00000000:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 41260 1 0000000000000000 100 0 0 10 0
`

func TestParseProcNet(t *testing.T) {
	rows := parseProcNet(strings.NewReader(procNetTCPFixture))
	if len(rows) != 3 {
		t.Fatalf("parsed %d rows, want 3", len(rows))
	}

	want := []procNetRow{
		{Addr: "127.0.0.1", Port: 8080, State: "0A", UID: "1000", Inode: "41253"},
		{Addr: "127.0.0.1", Port: 8080, State: "01", UID: "1000", Inode: "41999"},
		{Addr: "0.0.0.0", Port: 3000, State: "0A", UID: "0", Inode: "41260"},
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row %d = %+v; want %+v", i, rows[i], w)
		}
	}
}

func TestParseProcNetSkipsMalformed(t *testing.T) {
	in := `  sl  local_address rem_address   st
   0: garbage
   1: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 41253 1
`
	rows := parseProcNet(strings.NewReader(in))
	if len(rows) != 1 {
		t.Fatalf("parsed %d rows, want 1 (malformed lines skipped)", len(rows))
	}
	if rows[0].Port != 8080 {
		t.Errorf("port = %d, want 8080", rows[0].Port)
	}
}

func TestParseProcNetEmpty(t *testing.T) {
	// Header only — a machine with nothing listening.
	rows := parseProcNet(strings.NewReader("  sl  local_address rem_address   st\n"))
	if len(rows) != 0 {
		t.Fatalf("parsed %d rows, want 0", len(rows))
	}
}

func TestSocketInode(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"socket:[41253]", "41253", true},
		{"socket:[1]", "1", true},
		{"/dev/pts/3", "", false},
		{"pipe:[8899]", "", false},
		{"anon_inode:[eventpoll]", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := socketInode(tt.in)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("socketInode(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestParseSSOutput(t *testing.T) {
	// Verbatim `ss -tulpn` output, including the header we must skip.
	const fixture = `Netid State  Recv-Q Send-Q Local Address:Port  Peer Address:Port Process
udp   UNCONN 0      0      127.0.0.53%lo:53   0.0.0.0:*         users:(("systemd-resolve",pid=812,fd=13))
tcp   LISTEN 0      128    127.0.0.1:6379     0.0.0.0:*         users:(("redis-server",pid=1234,fd=6))
tcp   LISTEN 0      511    *:3000             *:*               users:(("node",pid=5678,fd=21))
tcp   LISTEN 0      128    [::]:22            [::]:*            users:(("sshd",pid=901,fd=4))
`
	got := parseSSOutput(fixture)
	if len(got) != 4 {
		t.Fatalf("parsed %d sockets, want 4:\n%+v", len(got), got)
	}

	tests := []struct {
		idx     int
		port    int
		proto   string
		process string
		pid     int
		state   string
	}{
		{0, 53, "udp", "systemd-resolve", 812, "OPEN"},
		{1, 6379, "tcp", "redis-server", 1234, "LISTEN"},
		{2, 3000, "tcp", "node", 5678, "LISTEN"},
		{3, 22, "tcp6", "sshd", 901, "LISTEN"},
	}
	for _, tt := range tests {
		s := got[tt.idx]
		if s.Port != tt.port || s.Protocol != tt.proto ||
			s.Process != tt.process || s.PID != tt.pid || s.State != tt.state {
			t.Errorf("socket %d = %+v; want port=%d proto=%s process=%s pid=%d state=%s",
				tt.idx, s, tt.port, tt.proto, tt.process, tt.pid, tt.state)
		}
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		in       string
		wantHost string
		wantPort int
		wantOK   bool
	}{
		{"0.0.0.0:8080", "0.0.0.0", 8080, true},
		{"127.0.0.1:3000", "127.0.0.1", 3000, true},
		{"*:3000", "0.0.0.0", 3000, true},
		{"[::]:22", "::", 22, true},
		{"[::1]:8080", "::1", 8080, true},
		{"nonsense", "", 0, false},
		{"host:notaport", "", 0, false},
	}
	for _, tt := range tests {
		host, port, ok := splitHostPort(tt.in)
		if host != tt.wantHost || port != tt.wantPort || ok != tt.wantOK {
			t.Errorf("splitHostPort(%q) = %q, %d, %v; want %q, %d, %v",
				tt.in, host, port, ok, tt.wantHost, tt.wantPort, tt.wantOK)
		}
	}
}

func TestDedupeCollapsesOneListenerPerProcess(t *testing.T) {
	// A single process bound across several interfaces must render as one row:
	// procfs reports a socket per bound address, but "python3 holds 3702" is
	// the fact the user needs.
	in := []Socket{
		{Port: 3702, Protocol: "udp", PID: 42, Addr: "172.18.0.1", Process: "python3"},
		{Port: 3702, Protocol: "udp", PID: 42, Addr: "239.255.255.250"},
		{Port: 3702, Protocol: "udp", PID: 42, Addr: "172.17.0.1"},
		{Port: 3000, Protocol: "tcp", PID: 20, Addr: "0.0.0.0", Process: "node"},
	}
	got := dedupeSockets(in)
	if len(got) != 2 {
		t.Fatalf("dedupeSockets returned %d sockets, want 2: %+v", len(got), got)
	}

	sortSockets(got)
	if got[0].Port != 3000 {
		t.Errorf("after sort, first port = %d; want 3000", got[0].Port)
	}
	if got[1].Process != "python3" {
		t.Errorf("merged socket lost its process name: %+v", got[1])
	}
}

func TestDedupePrefersTheMostInformativeAddress(t *testing.T) {
	tests := []struct {
		name string
		in   []Socket
		want string
	}{
		{
			name: "wildcard beats a specific interface",
			in: []Socket{
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "192.168.1.5"},
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "0.0.0.0"},
			},
			want: "0.0.0.0",
		},
		{
			name: "loopback beats a specific interface",
			in: []Socket{
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "192.168.1.5"},
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "127.0.0.1"},
			},
			want: "127.0.0.1",
		},
		{
			name: "equal rank breaks deterministically, not by input order",
			in: []Socket{
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "192.168.1.5"},
				{Port: 80, Protocol: "tcp", PID: 1, Addr: "172.17.0.1"},
			},
			want: "172.17.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeSockets(tt.in)
			if len(got) != 1 {
				t.Fatalf("got %d sockets, want 1: %+v", len(got), got)
			}
			if got[0].Addr != tt.want {
				t.Errorf("addr = %q; want %q", got[0].Addr, tt.want)
			}
		})
	}
}

func TestDedupeKeepsDistinctProcessesApart(t *testing.T) {
	// SO_REUSEPORT workers share a port legitimately; collapsing them would
	// hide processes the user may need to kill.
	in := []Socket{
		{Port: 8080, Protocol: "tcp", PID: 10, Addr: "0.0.0.0", Process: "worker"},
		{Port: 8080, Protocol: "tcp", PID: 11, Addr: "0.0.0.0", Process: "worker"},
	}
	if got := dedupeSockets(in); len(got) != 2 {
		t.Fatalf("dedupeSockets returned %d sockets, want 2: %+v", len(got), got)
	}
}
