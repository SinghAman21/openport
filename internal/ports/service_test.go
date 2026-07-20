package ports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// --- fakes ---------------------------------------------------------------
//
// These stand in for the OS and the disk. No test in this package touches
// either.

type fakeProvider struct {
	sockets []platform.Socket
	listErr error
	killErr error

	killed []killCall
}

type killCall struct {
	pid int
	sig platform.Signal
}

func (f *fakeProvider) ListListening(context.Context) ([]platform.Socket, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]platform.Socket(nil), f.sockets...), nil
}

func (f *fakeProvider) ProcessOnPort(_ context.Context, port int) (platform.Socket, error) {
	if f.listErr != nil {
		return platform.Socket{}, f.listErr
	}
	for _, s := range f.sockets {
		if s.Port == port {
			return s, nil
		}
	}
	return platform.Socket{}, platform.ErrNotFound
}

func (f *fakeProvider) Kill(_ context.Context, pid int, sig platform.Signal) error {
	f.killed = append(f.killed, killCall{pid, sig})
	return f.killErr
}

type fakeStore struct {
	aliases   []model.Alias
	groups    []model.Group
	protected model.ProtectedPorts

	history []model.HistoryEntry
	logs    []model.LogEntry

	aliasErr error
}

func (f *fakeStore) Aliases() ([]model.Alias, error) { return f.aliases, f.aliasErr }
func (f *fakeStore) SaveAlias(a model.Alias) error {
	f.aliases = append(f.aliases, a)
	return nil
}
func (f *fakeStore) DeleteAlias(string) error               { return nil }
func (f *fakeStore) Groups() ([]model.Group, error)         { return f.groups, nil }
func (f *fakeStore) SaveGroup(model.Group) error            { return nil }
func (f *fakeStore) DeleteGroup(string) error               { return nil }
func (f *fakeStore) Blueprints() ([]model.Blueprint, error) { return nil, nil }
func (f *fakeStore) SaveBlueprint(model.Blueprint) error    { return nil }
func (f *fakeStore) DeleteBlueprint(string) error           { return nil }
func (f *fakeStore) ProtectedPorts() (model.ProtectedPorts, error) {
	return f.protected, nil
}
func (f *fakeStore) SaveProtectedPorts(p model.ProtectedPorts) error {
	f.protected = p
	return nil
}
func (f *fakeStore) ProjectConfig(string) (model.ProjectConfig, bool, error) {
	return model.ProjectConfig{}, false, nil
}
func (f *fakeStore) SaveProjectConfig(string, model.ProjectConfig) error { return nil }
func (f *fakeStore) AppendHistory(e model.HistoryEntry) error {
	f.history = append(f.history, e)
	return nil
}
func (f *fakeStore) History(int) ([]model.HistoryEntry, error) { return f.history, nil }
func (f *fakeStore) AppendLog(e model.LogEntry) error {
	f.logs = append(f.logs, e)
	return nil
}
func (f *fakeStore) Logs(int) ([]model.LogEntry, error)  { return f.logs, nil }
func (f *fakeStore) Snapshot() (opsync.Snapshot, error)  { return opsync.Snapshot{}, nil }
func (f *fakeStore) ApplySnapshot(opsync.Snapshot) error { return nil }

type fakeProber struct {
	tcpErr   error
	httpCode int
	httpErr  error
}

func (f fakeProber) TCP(context.Context, string, int, time.Duration) error { return f.tcpErr }
func (f fakeProber) HTTP(context.Context, string, time.Duration) (int, error) {
	return f.httpCode, f.httpErr
}

// newTestService builds a service with every I/O seam replaced by a fake and a
// frozen clock.
func newTestService(prov *fakeProvider, st *fakeStore) *service {
	return &service{
		prov:    prov,
		store:   st,
		now:     func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) },
		canBind: func(int) bool { return true },
		probe:   fakeProber{},
	}
}

func sockets() []platform.Socket {
	return []platform.Socket{
		{Port: 22, Protocol: "tcp", PID: 901, Process: "sshd", Addr: "0.0.0.0", State: "LISTEN"},
		{Port: 3000, Protocol: "tcp", PID: 5678, Process: "node", Addr: "0.0.0.0", State: "LISTEN"},
		{Port: 6379, Protocol: "tcp", PID: 1234, Process: "redis-server", Addr: "127.0.0.1", State: "LISTEN"},
	}
}

// --- List ----------------------------------------------------------------

func TestListEnrichesPorts(t *testing.T) {
	prov := &fakeProvider{sockets: sockets()}
	st := &fakeStore{
		aliases:   []model.Alias{{Name: "frontend", Port: 3000}},
		groups:    []model.Group{{Name: "dev_stack", Ports: []int{3000, 6379}}},
		protected: model.ProtectedPorts{Ports: []int{22}},
	}
	svc := newTestService(prov, st)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d ports, want 3", len(got))
	}

	byPort := map[int]Port{}
	for _, p := range got {
		byPort[p.Port] = p
	}

	if a := byPort[3000].Alias; a != "frontend" {
		t.Errorf("port 3000 alias = %q; want %q", a, "frontend")
	}
	if g := byPort[3000].Groups; len(g) != 1 || g[0] != "dev_stack" {
		t.Errorf("port 3000 groups = %v; want [dev_stack]", g)
	}
	if !byPort[22].Protected {
		t.Error("port 22 should be protected")
	}
	if byPort[22].Reason == "" {
		t.Error("protected port should carry a reason for the UI to show")
	}
	if byPort[3000].Protected {
		t.Error("port 3000 should not be protected")
	}
	if byPort[6379].Alias != "" {
		t.Errorf("port 6379 alias = %q; want empty", byPort[6379].Alias)
	}
}

func TestListDegradesWhenStoreFails(t *testing.T) {
	// A broken aliases file must not stop the user seeing their ports.
	prov := &fakeProvider{sockets: sockets()}
	st := &fakeStore{aliasErr: errors.New("corrupt aliases.json")}
	svc := newTestService(prov, st)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d ports, want 3", len(got))
	}
	if len(st.logs) == 0 {
		t.Error("store failure should have been logged")
	}
}

func TestListPropagatesProviderError(t *testing.T) {
	prov := &fakeProvider{listErr: errors.New("procfs unreadable")}
	svc := newTestService(prov, &fakeStore{})

	if _, err := svc.List(context.Background()); err == nil {
		t.Fatal("List should propagate provider errors")
	}
}

// --- Check ---------------------------------------------------------------

func TestCheck(t *testing.T) {
	prov := &fakeProvider{sockets: sockets()}
	st := &fakeStore{aliases: []model.Alias{{Name: "frontend", Port: 3000}}}
	svc := newTestService(prov, st)

	t.Run("occupied", func(t *testing.T) {
		p, found, err := svc.Check(context.Background(), 3000)
		if err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if !found {
			t.Fatal("port 3000 should be found")
		}
		if p.Process != "node" || p.Alias != "frontend" {
			t.Errorf("Check = %+v; want process=node alias=frontend", p)
		}
	})

	t.Run("free", func(t *testing.T) {
		p, found, err := svc.Check(context.Background(), 9999)
		if err != nil {
			t.Fatalf("Check on a free port should not error: %v", err)
		}
		if found {
			t.Fatal("port 9999 should not be found")
		}
		if p.Port != 9999 {
			t.Errorf("Check returned port %d; want 9999", p.Port)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, _, err := svc.Check(context.Background(), 70000); err == nil {
			t.Fatal("Check should reject out-of-range ports")
		}
	})
}

// --- Kill ----------------------------------------------------------------

func TestKill(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		opts        KillOptions
		protected   model.ProtectedPorts
		wantSkipped bool
		wantReason  string
		wantSignal  platform.Signal
		wantKilled  bool
	}{
		{
			name:       "terminates unprotected port",
			port:       3000,
			wantSignal: platform.Terminate,
			wantKilled: true,
		},
		{
			name:       "force sends SIGKILL",
			port:       3000,
			opts:       KillOptions{Force: true},
			wantSignal: platform.Force,
			wantKilled: true,
		},
		{
			name:        "refuses protected port",
			port:        22,
			protected:   model.ProtectedPorts{Ports: []int{22}},
			wantSkipped: true,
			wantReason:  "port 22 is protected",
		},
		{
			name:        "refuses protected process by name",
			port:        6379,
			protected:   model.ProtectedPorts{Processes: []string{"redis-server"}},
			wantSkipped: true,
			wantReason:  `process "redis-server" is protected`,
		},
		{
			name:       "override kills protected port",
			port:       22,
			opts:       KillOptions{OverrideProtected: true},
			protected:  model.ProtectedPorts{Ports: []int{22}},
			wantSignal: platform.Terminate,
			wantKilled: true,
		},
		{
			name:        "nothing listening",
			port:        9999,
			wantSkipped: true,
			wantReason:  "nothing listening",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov := &fakeProvider{sockets: sockets()}
			st := &fakeStore{protected: tt.protected}
			svc := newTestService(prov, st)

			res, err := svc.KillWith(context.Background(), tt.port, tt.opts)
			if err != nil {
				t.Fatalf("KillWith returned error: %v", err)
			}

			if res.Skipped != tt.wantSkipped {
				t.Errorf("Skipped = %v; want %v (reason %q)", res.Skipped, tt.wantSkipped, res.Reason)
			}
			if tt.wantReason != "" && res.Reason != tt.wantReason {
				t.Errorf("Reason = %q; want %q", res.Reason, tt.wantReason)
			}
			if tt.wantKilled {
				if len(prov.killed) != 1 {
					t.Fatalf("provider received %d kills, want 1", len(prov.killed))
				}
				if prov.killed[0].sig != tt.wantSignal {
					t.Errorf("signal = %v; want %v", prov.killed[0].sig, tt.wantSignal)
				}
			} else if len(prov.killed) != 0 {
				t.Fatalf("provider received %d kills, want 0", len(prov.killed))
			}

			// Every kill attempt, skipped or not, must be audited.
			if len(st.history) != 1 {
				t.Fatalf("history has %d entries, want 1", len(st.history))
			}
			if st.history[0].Kind != model.ActionKill {
				t.Errorf("history kind = %q; want %q", st.history[0].Kind, model.ActionKill)
			}
		})
	}
}

// TestKillDefaultRefusesProtected pins the invariant that the plain Kill entry
// point can never take down a protected port, whatever the force flag says.
func TestKillDefaultRefusesProtected(t *testing.T) {
	for _, force := range []bool{false, true} {
		prov := &fakeProvider{sockets: sockets()}
		st := &fakeStore{protected: model.ProtectedPorts{Ports: []int{22}}}
		svc := newTestService(prov, st)

		res, err := svc.Kill(context.Background(), 22, force)
		if err != nil {
			t.Fatalf("Kill(force=%v) returned error: %v", force, err)
		}
		if !res.Skipped {
			t.Errorf("Kill(force=%v) on a protected port was not skipped", force)
		}
		if len(prov.killed) != 0 {
			t.Errorf("Kill(force=%v) signalled a protected process", force)
		}
	}
}

func TestKillReportsProviderError(t *testing.T) {
	prov := &fakeProvider{sockets: sockets(), killErr: platform.ErrPermission}
	st := &fakeStore{}
	svc := newTestService(prov, st)

	res, err := svc.Kill(context.Background(), 3000, false)
	if err != nil {
		t.Fatalf("Kill returned a transport error: %v", err)
	}
	if res.Err == nil {
		t.Fatal("result should carry the provider error")
	}
	if res.OK() {
		t.Error("result should not report OK when the kill failed")
	}
	if st.history[0].Success {
		t.Error("history should record the failure")
	}
}

func TestKillSkipsInvisibleProcess(t *testing.T) {
	// A socket whose owning PID could not be resolved (another user's process
	// without elevated rights) must be skipped with an actionable reason,
	// never signalled as PID 0.
	prov := &fakeProvider{sockets: []platform.Socket{
		{Port: 3000, Protocol: "tcp", PID: 0, Addr: "0.0.0.0", State: "LISTEN"},
	}}
	svc := newTestService(prov, &fakeStore{})

	res, err := svc.Kill(context.Background(), 3000, false)
	if err != nil {
		t.Fatalf("Kill returned error: %v", err)
	}
	if !res.Skipped {
		t.Fatal("a port with no visible owner must be skipped")
	}
	if len(prov.killed) != 0 {
		t.Fatal("must not signal pid 0")
	}
}

// --- KillRange -----------------------------------------------------------

func TestKillRange(t *testing.T) {
	prov := &fakeProvider{sockets: []platform.Socket{
		{Port: 3000, PID: 1, Process: "node", Protocol: "tcp"},
		{Port: 3001, PID: 2, Process: "vite", Protocol: "tcp"},
		{Port: 3002, PID: 3, Process: "postgres", Protocol: "tcp"},
		{Port: 9000, PID: 4, Process: "other", Protocol: "tcp"}, // out of range
	}}
	st := &fakeStore{protected: model.ProtectedPorts{Ports: []int{3001}}}
	svc := newTestService(prov, st)

	results, err := svc.KillRange(context.Background(), 3000, 3002, false)
	if err != nil {
		t.Fatalf("KillRange returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("KillRange returned %d results, want 3", len(results))
	}

	// Results are ordered by port so the UI can render them predictably.
	for i, wantPort := range []int{3000, 3001, 3002} {
		if results[i].Port != wantPort {
			t.Errorf("result %d is port %d; want %d", i, results[i].Port, wantPort)
		}
	}
	if !results[1].Skipped {
		t.Error("protected port 3001 should have been skipped")
	}
	if results[0].Skipped || results[2].Skipped {
		t.Error("unprotected ports should not be skipped")
	}
	if len(prov.killed) != 2 {
		t.Fatalf("provider received %d kills, want 2", len(prov.killed))
	}
	for _, k := range prov.killed {
		if k.pid == 2 {
			t.Error("protected port's process was signalled")
		}
	}
}

func TestKillRangeInvertedBounds(t *testing.T) {
	prov := &fakeProvider{sockets: []platform.Socket{{Port: 3000, PID: 1, Protocol: "tcp"}}}
	svc := newTestService(prov, &fakeStore{})

	// hi < lo should be normalised, not rejected.
	results, err := svc.KillRangeWith(context.Background(), 3005, 2999, KillOptions{})
	if err != nil {
		t.Fatalf("KillRangeWith returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestKillRangeRejectsBadBounds(t *testing.T) {
	svc := newTestService(&fakeProvider{}, &fakeStore{})
	if _, err := svc.KillRange(context.Background(), 0, 100, false); err == nil {
		t.Error("KillRange should reject port 0")
	}
	if _, err := svc.KillRange(context.Background(), 100, 70000, false); err == nil {
		t.Error("KillRange should reject ports above 65535")
	}
}

// --- KillPID -------------------------------------------------------------

func TestKillPIDRespectsProtection(t *testing.T) {
	prov := &fakeProvider{sockets: sockets()}
	st := &fakeStore{protected: model.ProtectedPorts{Ports: []int{22}}}
	svc := newTestService(prov, st)

	res, err := svc.KillPID(context.Background(), 901, false) // sshd
	if err != nil {
		t.Fatalf("KillPID returned error: %v", err)
	}
	if !res.Skipped {
		t.Fatal("killing the PID behind a protected port must be refused")
	}
	if len(prov.killed) != 0 {
		t.Fatal("protected process was signalled via KillPID")
	}
}

func TestKillPIDRejectsInvalid(t *testing.T) {
	svc := newTestService(&fakeProvider{}, &fakeStore{})
	if _, err := svc.KillPID(context.Background(), 0, false); err == nil {
		t.Error("KillPID should reject pid 0")
	}
}

// --- SuggestFree ---------------------------------------------------------

func TestSuggestFree(t *testing.T) {
	prov := &fakeProvider{sockets: []platform.Socket{
		{Port: 3000, PID: 1, Protocol: "tcp"},
		{Port: 3001, PID: 2, Protocol: "tcp"},
	}}
	st := &fakeStore{
		aliases:   []model.Alias{{Name: "reserved-api", Port: 3002}},
		protected: model.ProtectedPorts{Ports: []int{3003}},
	}
	svc := newTestService(prov, st)

	got, err := svc.SuggestFree(context.Background(), 3000, 2)
	if err != nil {
		t.Fatalf("SuggestFree returned error: %v", err)
	}
	// 3000/3001 occupied, 3002 reserved by alias, 3003 protected.
	want := []int{3004, 3005}
	if len(got) != len(want) {
		t.Fatalf("SuggestFree returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SuggestFree returned %v, want %v", got, want)
		}
	}
	if len(st.history) != 1 || st.history[0].Kind != model.ActionSuggest {
		t.Error("suggestion should be recorded in history")
	}
}

func TestSuggestFreeSkipsUnbindable(t *testing.T) {
	// A port nothing reports as listening but which cannot actually be bound
	// (another user's socket, or TIME_WAIT) must not be suggested.
	prov := &fakeProvider{}
	svc := newTestService(prov, &fakeStore{})
	svc.canBind = func(port int) bool { return port != 4000 }

	got, err := svc.SuggestFree(context.Background(), 4000, 1)
	if err != nil {
		t.Fatalf("SuggestFree returned error: %v", err)
	}
	if len(got) != 1 || got[0] != 4001 {
		t.Errorf("SuggestFree = %v; want [4001]", got)
	}
}

func TestSuggestFreeDefaults(t *testing.T) {
	svc := newTestService(&fakeProvider{}, &fakeStore{})

	got, err := svc.SuggestFree(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("SuggestFree returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("SuggestFree returned %d ports, want 1", len(got))
	}
	if got[0] != DefaultSuggestNear {
		t.Errorf("SuggestFree = %v; want it to start at %d", got, DefaultSuggestNear)
	}
}

func TestSuggestFreeExhausted(t *testing.T) {
	svc := newTestService(&fakeProvider{}, &fakeStore{})
	svc.canBind = func(int) bool { return false }

	if _, err := svc.SuggestFree(context.Background(), 3000, 1); err == nil {
		t.Fatal("SuggestFree should error when no port can be found")
	}
}

// --- Reserve -------------------------------------------------------------

func TestReserve(t *testing.T) {
	st := &fakeStore{}
	svc := newTestService(&fakeProvider{}, st)

	if err := svc.Reserve(context.Background(), 4000, "api"); err != nil {
		t.Fatalf("Reserve returned error: %v", err)
	}
	if len(st.aliases) != 1 || st.aliases[0].Port != 4000 || st.aliases[0].Name != "api" {
		t.Fatalf("Reserve stored %+v; want an alias api->4000", st.aliases)
	}
	if len(st.history) != 1 || st.history[0].Kind != model.ActionReserve {
		t.Error("reservation should be recorded in history")
	}
}

func TestReserveRejectsBadInput(t *testing.T) {
	svc := newTestService(&fakeProvider{}, &fakeStore{})
	if err := svc.Reserve(context.Background(), 4000, "  "); err == nil {
		t.Error("Reserve should require a service name")
	}
	if err := svc.Reserve(context.Background(), 70000, "api"); err == nil {
		t.Error("Reserve should reject out-of-range ports")
	}
}

// --- Health --------------------------------------------------------------

func TestHealth(t *testing.T) {
	tests := []struct {
		name     string
		kind     HealthKind
		probe    fakeProber
		wantOK   bool
		wantCode int
	}{
		{name: "tcp open", kind: HealthTCP, probe: fakeProber{}, wantOK: true},
		{
			name:  "tcp refused",
			kind:  HealthTCP,
			probe: fakeProber{tcpErr: errors.New("connection refused")},
		},
		{
			name:     "http 200",
			kind:     HealthHTTP,
			probe:    fakeProber{httpCode: 200},
			wantOK:   true,
			wantCode: 200,
		},
		{
			name:     "http 302 counts as up",
			kind:     HealthHTTP,
			probe:    fakeProber{httpCode: 302},
			wantOK:   true,
			wantCode: 302,
		},
		{
			name:     "http 500 counts as down",
			kind:     HealthHTTP,
			probe:    fakeProber{httpCode: 500},
			wantCode: 500,
		},
		{
			name:  "http unreachable",
			kind:  HealthHTTP,
			probe: fakeProber{httpErr: errors.New("connection refused")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(&fakeProvider{}, &fakeStore{})
			svc.probe = tt.probe

			res, err := svc.Health(context.Background(), 3000, tt.kind)
			if err != nil {
				t.Fatalf("Health returned error: %v", err)
			}
			if res.OK != tt.wantOK {
				t.Errorf("OK = %v; want %v (status %q)", res.OK, tt.wantOK, res.Status)
			}
			if res.Code != tt.wantCode {
				t.Errorf("Code = %d; want %d", res.Code, tt.wantCode)
			}
			if res.Status == "" {
				t.Error("a health result should always carry a status string")
			}
		})
	}
}

func TestHealthBadge(t *testing.T) {
	var nilResult *HealthResult
	if got := nilResult.Badge(); got != "·" {
		t.Errorf("nil badge = %q; want ·", got)
	}
	if got := (&HealthResult{OK: true}).Badge(); got != "✓" {
		t.Errorf("ok badge = %q; want ✓", got)
	}
	if got := (&HealthResult{}).Badge(); got != "✕" {
		t.Errorf("down badge = %q; want ✕", got)
	}
}

// --- Watch ---------------------------------------------------------------

func TestWatchEmitsAndStops(t *testing.T) {
	prov := &fakeProvider{sockets: sockets()}
	svc := newTestService(prov, &fakeStore{})

	ch, cancel := svc.Watch(context.Background(), 10*time.Millisecond)

	// The first listing arrives immediately rather than after a full interval.
	select {
	case got := <-ch:
		if len(got) != 3 {
			t.Fatalf("first tick carried %d ports, want 3", len(got))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first watch tick")
	}

	// And it keeps ticking.
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed while still watching")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the second watch tick")
	}

	cancel()

	// Cancelling closes the channel, which is what tells the UI's re-arming
	// command to stand down.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // closed, as expected
			}
		case <-deadline:
			t.Fatal("channel was not closed after cancel")
		}
	}
}

func TestWatchStopsOnContextCancel(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	svc := newTestService(&fakeProvider{sockets: sockets()}, &fakeStore{})

	ch, cancel := svc.Watch(ctx, 10*time.Millisecond)
	defer cancel()

	<-ch // drain the immediate tick
	cancelCtx()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("cancelling the context did not stop the watcher")
		}
	}
}

// --- domain helpers ------------------------------------------------------

func TestPortKeyIsStableAcrossRefreshes(t *testing.T) {
	a := Port{Socket: platform.Socket{Port: 3000, Protocol: "tcp", PID: 42}}
	b := Port{Socket: platform.Socket{Port: 3000, Protocol: "tcp", PID: 42}, Alias: "frontend"}
	if a.Key() != b.Key() {
		t.Errorf("key changed when enrichment changed: %q vs %q", a.Key(), b.Key())
	}

	c := Port{Socket: platform.Socket{Port: 3000, Protocol: "tcp", PID: 43}}
	if a.Key() == c.Key() {
		t.Error("different processes on the same port should have different keys")
	}
}

func TestPortLabel(t *testing.T) {
	p := Port{Socket: platform.Socket{Process: "node"}}
	if got := p.Label(); got != "node" {
		t.Errorf("Label = %q; want node", got)
	}
	p.Alias = "frontend"
	if got := p.Label(); got != "frontend" {
		t.Errorf("Label = %q; want frontend", got)
	}
}
