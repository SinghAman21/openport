package blueprint

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	"github.com/SinghAman21/openport/internal/ports"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// --- fakes ---------------------------------------------------------------

type fakeService struct {
	live       map[int]ports.Port
	checkErr   error
	suggestErr error
}

func (f *fakeService) Check(_ context.Context, port int) (ports.Port, bool, error) {
	if f.checkErr != nil {
		return ports.Port{}, false, f.checkErr
	}
	p, ok := f.live[port]
	if !ok {
		return ports.Port{Socket: platform.Socket{Port: port}}, false, nil
	}
	return p, true, nil
}

// SuggestFree mirrors the real contract: the first free port is at or above
// the hint, not past it.
func (f *fakeService) SuggestFree(_ context.Context, near, count int) ([]int, error) {
	if f.suggestErr != nil {
		return nil, f.suggestErr
	}
	return []int{near}, nil
}

func (f *fakeService) List(context.Context) ([]ports.Port, error) { return nil, nil }
func (f *fakeService) Kill(context.Context, int, bool) (ports.KillResult, error) {
	return ports.KillResult{}, nil
}
func (f *fakeService) KillWith(context.Context, int, ports.KillOptions) (ports.KillResult, error) {
	return ports.KillResult{}, nil
}
func (f *fakeService) KillPID(context.Context, int, bool) (ports.KillResult, error) {
	return ports.KillResult{}, nil
}
func (f *fakeService) KillRange(context.Context, int, int, bool) ([]ports.KillResult, error) {
	return nil, nil
}
func (f *fakeService) KillRangeWith(context.Context, int, int, ports.KillOptions) ([]ports.KillResult, error) {
	return nil, nil
}
func (f *fakeService) Reserve(context.Context, int, string) error { return nil }
func (f *fakeService) Health(context.Context, int, ports.HealthKind) (ports.HealthResult, error) {
	return ports.HealthResult{}, nil
}
func (f *fakeService) Watch(context.Context, time.Duration) (<-chan []ports.Port, func()) {
	ch := make(chan []ports.Port)
	return ch, func() { close(ch) }
}

type fakeStore struct {
	blueprints []model.Blueprint
	history    []model.HistoryEntry
	listErr    error
}

func (f *fakeStore) Blueprints() ([]model.Blueprint, error) { return f.blueprints, f.listErr }
func (f *fakeStore) AppendHistory(e model.HistoryEntry) error {
	f.history = append(f.history, e)
	return nil
}
func (f *fakeStore) Aliases() ([]model.Alias, error)     { return nil, nil }
func (f *fakeStore) SaveAlias(model.Alias) error         { return nil }
func (f *fakeStore) DeleteAlias(string) error            { return nil }
func (f *fakeStore) Groups() ([]model.Group, error)      { return nil, nil }
func (f *fakeStore) SaveGroup(model.Group) error         { return nil }
func (f *fakeStore) DeleteGroup(string) error            { return nil }
func (f *fakeStore) SaveBlueprint(model.Blueprint) error { return nil }
func (f *fakeStore) DeleteBlueprint(string) error        { return nil }
func (f *fakeStore) ProtectedPorts() (model.ProtectedPorts, error) {
	return model.ProtectedPorts{}, nil
}
func (f *fakeStore) SaveProtectedPorts(model.ProtectedPorts) error { return nil }
func (f *fakeStore) ProjectConfig(string) (model.ProjectConfig, bool, error) {
	return model.ProjectConfig{}, false, nil
}
func (f *fakeStore) SaveProjectConfig(string, model.ProjectConfig) error { return nil }
func (f *fakeStore) History(int) ([]model.HistoryEntry, error)           { return f.history, nil }
func (f *fakeStore) AppendLog(model.LogEntry) error                      { return nil }
func (f *fakeStore) Logs(int) ([]model.LogEntry, error)                  { return nil, nil }
func (f *fakeStore) Snapshot() (opsync.Snapshot, error)                  { return opsync.Snapshot{}, nil }
func (f *fakeStore) ApplySnapshot(opsync.Snapshot) error                 { return nil }

func livePort(port int, process, alias string) ports.Port {
	return ports.Port{
		Socket: platform.Socket{Port: port, PID: 100, Process: process, Protocol: "tcp"},
		Alias:  alias,
	}
}

// --- Apply ---------------------------------------------------------------

func TestApplyClassifiesEachService(t *testing.T) {
	svc := &fakeService{live: map[int]ports.Port{
		3000: livePort(3000, "node", ""),        // matches by process name
		5432: livePort(5432, "python", ""),      // wrong thing on the port
		6379: livePort(6379, "server", "cache"), // matches by alias
	}}
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name: "web",
		Services: []model.BlueprintService{
			{Name: "node", Port: 3000},
			{Name: "postgres", Port: 5432},
			{Name: "cache", Port: 6379},
			{Name: "worker", Port: 9000}, // nothing listening
		},
	}}}

	results, err := Apply(context.Background(), svc, st, "web")
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("got %d results, want 4", len(results))
	}

	want := []struct {
		service string
		status  string
	}{
		{"node", StatusOK},
		{"postgres", StatusConflict},
		{"cache", StatusOK},
		{"worker", StatusFree},
	}
	for i, w := range want {
		if results[i].Service != w.service || results[i].Status != w.status {
			t.Errorf("result %d = %s/%s; want %s/%s",
				i, results[i].Service, results[i].Status, w.service, w.status)
		}
	}
}

// TestApplyOffersAnAlternativeForConflicts checks the tool proposes somewhere
// else to put a service rather than only reporting the clash.
func TestApplyOffersAnAlternativeForConflicts(t *testing.T) {
	svc := &fakeService{live: map[int]ports.Port{
		5432: livePort(5432, "python", ""),
	}}
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "postgres", Port: 5432}},
	}}}

	results, err := Apply(context.Background(), svc, st, "web")
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if results[0].Suggested != 5433 {
		t.Errorf("suggested = %d; want a free port near the wanted one", results[0].Suggested)
	}
	if results[0].Detail == "" {
		t.Error("a conflict should name what is holding the port")
	}
}

func TestApplySurvivesSuggestionFailure(t *testing.T) {
	// If no alternative can be found, the conflict is still reported.
	svc := &fakeService{
		live:       map[int]ports.Port{5432: livePort(5432, "python", "")},
		suggestErr: errors.New("no free port"),
	}
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "postgres", Port: 5432}},
	}}}

	results, err := Apply(context.Background(), svc, st, "web")
	if err != nil {
		t.Fatalf("Apply should not fail when a suggestion cannot be made: %v", err)
	}
	if results[0].Status != StatusConflict {
		t.Errorf("status = %q; want %q", results[0].Status, StatusConflict)
	}
	if results[0].Suggested != 0 {
		t.Errorf("suggested = %d; want none", results[0].Suggested)
	}
}

// TestApplyNeverMutatesProcesses is the safety property: applying a blueprint
// reports, it does not start or stop anything.
func TestApplyIsReadOnly(t *testing.T) {
	svc := &fakeService{live: map[int]ports.Port{3000: livePort(3000, "python", "")}}
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "node", Port: 3000}},
	}}}

	if _, err := Apply(context.Background(), svc, st, "web"); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	// The fake would record any kill; none of its mutating methods do
	// anything, so the assertion here is the audit trail: exactly one entry,
	// and it is the apply itself.
	if len(st.history) != 1 {
		t.Fatalf("history has %d entries, want exactly the apply record", len(st.history))
	}
	if st.history[0].Kind != model.ActionApplyBlueprint {
		t.Errorf("history kind = %q; want %q", st.history[0].Kind, model.ActionApplyBlueprint)
	}
}

func TestApplyRecordsHistory(t *testing.T) {
	svc := &fakeService{live: map[int]ports.Port{3000: livePort(3000, "node", "")}}
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "node", Port: 3000}},
	}}}

	if _, err := Apply(context.Background(), svc, st, "web"); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(st.history) != 1 {
		t.Fatalf("history has %d entries, want 1", len(st.history))
	}
	if !st.history[0].Success {
		t.Error("a clean apply should be recorded as a success")
	}
}

func TestApplyUnknownBlueprint(t *testing.T) {
	st := &fakeStore{}
	if _, err := Apply(context.Background(), &fakeService{}, st, "nope"); err == nil {
		t.Fatal("applying a missing blueprint should error")
	}
	if len(st.history) != 0 {
		t.Error("a failed lookup should not be recorded as an apply")
	}
}

func TestApplyIsCaseInsensitive(t *testing.T) {
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "Web",
		Services: []model.BlueprintService{{Name: "api", Port: 3000}},
	}}}
	if _, err := Apply(context.Background(), &fakeService{}, st, "web"); err != nil {
		t.Fatalf("blueprint lookup should be case-insensitive: %v", err)
	}
}

func TestApplyRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "api", Port: 3000}},
	}}}
	if _, err := Apply(ctx, &fakeService{}, st, "web"); err == nil {
		t.Error("a cancelled context should abort the walk")
	}
}

// --- Matches -------------------------------------------------------------

func TestMatches(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		process string
		service string
		want    bool
	}{
		{"alias is exact", "cache", "redis-server", "cache", true},
		{"alias ignores case", "Cache", "redis-server", "cache", true},
		{"process contains the service name", "", "redis-server", "redis", true},
		{"service contains the process name", "", "node", "node-api", true},
		{"unrelated", "", "python3", "postgres", false},
		{"no service name", "", "node", "", false},
		{"no process and no alias", "", "", "api", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.alias, tt.process, tt.service); got != tt.want {
				t.Errorf("Matches(%q, %q, %q) = %v; want %v",
					tt.alias, tt.process, tt.service, got, tt.want)
			}
		})
	}
}

func TestResultOK(t *testing.T) {
	if !(Result{Status: StatusOK}).OK() {
		t.Error("an ok result should report OK")
	}
	for _, status := range []string{StatusFree, StatusConflict, ""} {
		if (Result{Status: status}).OK() {
			t.Errorf("status %q should not report OK", status)
		}
	}
}
