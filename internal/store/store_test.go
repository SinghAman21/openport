package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// newTestStore builds a FileStore rooted in a temp dir with a frozen clock, so
// tests never touch the user's real config directory.
func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	s, err := NewFileStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStoreAt: %v", err)
	}
	fs := s.(*FileStore)
	fs.now = func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) }
	return fs
}

// --- aliases -------------------------------------------------------------

func TestAliasesRoundTrip(t *testing.T) {
	s := newTestStore(t)

	if got, err := s.Aliases(); err != nil || len(got) != 0 {
		t.Fatalf("fresh store: Aliases() = %v, %v; want empty, nil", got, err)
	}

	if err := s.SaveAlias(model.Alias{Name: "frontend", Port: 3000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}
	if err := s.SaveAlias(model.Alias{Name: "cache", Port: 6379}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}

	got, err := s.Aliases()
	if err != nil {
		t.Fatalf("Aliases: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Aliases returned %d entries, want 2", len(got))
	}
	// Sorted by port.
	if got[0].Port != 3000 || got[1].Port != 6379 {
		t.Errorf("aliases not sorted by port: %+v", got)
	}
	if got[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be stamped on save")
	}
}

func TestSaveAliasUpdatesInPlace(t *testing.T) {
	s := newTestStore(t)

	if err := s.SaveAlias(model.Alias{Name: "frontend", Port: 3000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}
	if err := s.SaveAlias(model.Alias{Name: "frontend", Port: 4000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}

	got, err := s.Aliases()
	if err != nil {
		t.Fatalf("Aliases: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("re-saving a name created a duplicate: %+v", got)
	}
	if got[0].Port != 4000 {
		t.Errorf("port = %d; want 4000", got[0].Port)
	}
}

func TestSaveAliasValidates(t *testing.T) {
	s := newTestStore(t)
	tests := []struct {
		name  string
		alias model.Alias
	}{
		{"empty name", model.Alias{Name: "", Port: 3000}},
		{"whitespace in name", model.Alias{Name: "my app", Port: 3000}},
		{"port zero", model.Alias{Name: "x", Port: 0}},
		{"port too high", model.Alias{Name: "x", Port: 70000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.SaveAlias(tt.alias); err == nil {
				t.Errorf("SaveAlias(%+v) should have failed", tt.alias)
			}
		})
	}
}

func TestDeleteAlias(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveAlias(model.Alias{Name: "frontend", Port: 3000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}

	if err := s.DeleteAlias("frontend"); err != nil {
		t.Fatalf("DeleteAlias: %v", err)
	}
	got, _ := s.Aliases()
	if len(got) != 0 {
		t.Fatalf("alias survived deletion: %+v", got)
	}

	if err := s.DeleteAlias("nope"); err == nil {
		t.Error("deleting a missing alias should report an error")
	}
}

// --- groups --------------------------------------------------------------

func TestGroupsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	g := model.Group{Name: "dev_stack", Ports: []int{6379, 3000, 5432}}
	if err := s.SaveGroup(g); err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}

	got, err := s.Groups()
	if err != nil {
		t.Fatalf("Groups: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Groups returned %d entries, want 1", len(got))
	}
	// Ports are normalised to sorted order on save.
	want := []int{3000, 5432, 6379}
	for i, p := range want {
		if got[0].Ports[i] != p {
			t.Fatalf("group ports = %v; want %v", got[0].Ports, want)
		}
	}

	if err := s.DeleteGroup("dev_stack"); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if got, _ := s.Groups(); len(got) != 0 {
		t.Errorf("group survived deletion: %+v", got)
	}
}

func TestSaveGroupValidates(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveGroup(model.Group{Name: "empty"}); err == nil {
		t.Error("a group with no ports should be rejected")
	}
	if err := s.SaveGroup(model.Group{Name: "", Ports: []int{80}}); err == nil {
		t.Error("a group with no name should be rejected")
	}
}

// --- blueprints ----------------------------------------------------------

func TestBlueprintsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	bp := model.Blueprint{
		Name: "web",
		Services: []model.BlueprintService{
			{Name: "api", Port: 8080, Health: model.HealthHTTP},
			{Name: "db", Port: 5432, Health: model.HealthTCP},
		},
	}
	if err := s.SaveBlueprint(bp); err != nil {
		t.Fatalf("SaveBlueprint: %v", err)
	}

	got, err := s.Blueprints()
	if err != nil {
		t.Fatalf("Blueprints: %v", err)
	}
	if len(got) != 1 || len(got[0].Services) != 2 {
		t.Fatalf("Blueprints = %+v; want one blueprint with two services", got)
	}
	if got[0].CreatedAt.IsZero() || got[0].UpdatedAt.IsZero() {
		t.Error("blueprint timestamps should be stamped on save")
	}

	if err := s.DeleteBlueprint("web"); err != nil {
		t.Fatalf("DeleteBlueprint: %v", err)
	}
	if got, _ := s.Blueprints(); len(got) != 0 {
		t.Errorf("blueprint survived deletion: %+v", got)
	}
}

func TestSaveBlueprintRejectsDuplicatePorts(t *testing.T) {
	s := newTestStore(t)
	bp := model.Blueprint{
		Name: "clash",
		Services: []model.BlueprintService{
			{Name: "api", Port: 8080},
			{Name: "admin", Port: 8080},
		},
	}
	if err := s.SaveBlueprint(bp); err == nil {
		t.Error("two services claiming the same port should be rejected")
	}
}

// --- protected ports -----------------------------------------------------

func TestProtectedPortsDefaultsOnFirstRun(t *testing.T) {
	s := newTestStore(t)

	got, err := s.ProtectedPorts()
	if err != nil {
		t.Fatalf("ProtectedPorts: %v", err)
	}
	if protected, _ := got.IsProtected(22, ""); !protected {
		t.Error("a fresh install must protect ssh by default")
	}
}

func TestProtectedPortsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	p := model.ProtectedPorts{
		Ports:     []int{5432, 22},
		Ranges:    []model.PortRange{{Name: "system", From: 1, To: 1023}},
		Processes: []string{"postgres"},
	}
	if err := s.SaveProtectedPorts(p); err != nil {
		t.Fatalf("SaveProtectedPorts: %v", err)
	}

	got, err := s.ProtectedPorts()
	if err != nil {
		t.Fatalf("ProtectedPorts: %v", err)
	}
	if got.Ports[0] != 22 {
		t.Errorf("ports should be stored sorted, got %v", got.Ports)
	}

	tests := []struct {
		port    int
		process string
		want    bool
	}{
		{5432, "", true},
		{80, "", true}, // inside the system range
		{3000, "", false},
		{3000, "postgres", true}, // protected by process name
	}
	for _, tt := range tests {
		if protected, _ := got.IsProtected(tt.port, tt.process); protected != tt.want {
			t.Errorf("IsProtected(%d, %q) = %v; want %v", tt.port, tt.process, protected, tt.want)
		}
	}
}

// --- project config ------------------------------------------------------

func TestProjectConfigRoundTrip(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()

	cfg := model.ProjectConfig{
		Name:          "shop",
		ExpectedPorts: []int{3000, 5432},
		Aliases:       map[string]int{"web": 3000},
		Ranges:        []model.PortRange{{Name: "workers", From: 9000, To: 9010}},
	}
	if err := s.SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	got, found, err := s.ProjectConfig(dir)
	if err != nil {
		t.Fatalf("ProjectConfig: %v", err)
	}
	if !found {
		t.Fatal("saved config was not found")
	}
	if got.Name != "shop" || len(got.ExpectedPorts) != 2 || got.Aliases["web"] != 3000 {
		t.Errorf("round-tripped config = %+v; want it to match what was saved", got)
	}
	if got.Dir != dir {
		t.Errorf("Dir = %q; want %q", got.Dir, dir)
	}
}

func TestProjectConfigWalksUpToParent(t *testing.T) {
	s := newTestStore(t)
	root := t.TempDir()
	nested := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := s.SaveProjectConfig(root, model.ProjectConfig{Name: "monorepo"}); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	// Running openport from a subdirectory should still find the project.
	got, found, err := s.ProjectConfig(nested)
	if err != nil {
		t.Fatalf("ProjectConfig: %v", err)
	}
	if !found {
		t.Fatal("config in a parent directory was not found")
	}
	if got.Name != "monorepo" {
		t.Errorf("name = %q; want monorepo", got.Name)
	}
}

func TestProjectConfigMissingIsNotAnError(t *testing.T) {
	s := newTestStore(t)
	_, found, err := s.ProjectConfig(t.TempDir())
	if err != nil {
		t.Fatalf("a missing project config should not error: %v", err)
	}
	if found {
		t.Error("found = true for a directory with no config")
	}
}

func TestProjectConfigReportsMalformedYAML(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	path := filepath.Join(dir, model.ProjectConfigFile)
	if err := os.WriteFile(path, []byte("name: [unclosed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, _, err := s.ProjectConfig(dir); err == nil {
		t.Error("malformed YAML should surface an error rather than be ignored")
	}
}

func TestSaveProjectConfigValidates(t *testing.T) {
	s := newTestStore(t)
	cfg := model.ProjectConfig{Name: "bad", ExpectedPorts: []int{70000}}
	if err := s.SaveProjectConfig(t.TempDir(), cfg); err == nil {
		t.Error("an out-of-range port should be rejected before writing")
	}
}

// --- history & logs ------------------------------------------------------

func TestHistoryNewestFirst(t *testing.T) {
	s := newTestStore(t)

	for i := 1; i <= 3; i++ {
		e := model.HistoryEntry{
			Kind:    model.ActionKill,
			Summary: "kill " + string(rune('0'+i)),
			Port:    3000 + i,
			Success: true,
		}
		if err := s.AppendHistory(e); err != nil {
			t.Fatalf("AppendHistory: %v", err)
		}
	}

	got, err := s.History(0)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("History returned %d entries, want 3", len(got))
	}
	if got[0].Port != 3003 {
		t.Errorf("History should be newest-first; first entry is port %d", got[0].Port)
	}
	if got[0].At.IsZero() {
		t.Error("At should be stamped on append")
	}

	limited, err := s.History(2)
	if err != nil {
		t.Fatalf("History(2): %v", err)
	}
	if len(limited) != 2 || limited[0].Port != 3003 {
		t.Errorf("History(2) = %+v; want the two newest entries", limited)
	}
}

func TestHistoryTrimsToCap(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < MaxHistory+10; i++ {
		if err := s.AppendHistory(model.HistoryEntry{Kind: model.ActionCheck, Port: i}); err != nil {
			t.Fatalf("AppendHistory: %v", err)
		}
	}
	got, err := s.History(0)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(got) != MaxHistory {
		t.Fatalf("History holds %d entries, want it capped at %d", len(got), MaxHistory)
	}
	// The newest entry survives; the oldest were dropped.
	if got[0].Port != MaxHistory+9 {
		t.Errorf("newest entry is port %d; want %d", got[0].Port, MaxHistory+9)
	}
}

func TestLogsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	if err := s.AppendLog(model.LogEntry{Level: model.LogWarn, Source: "platform", Message: "fell back to ss"}); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}
	if err := s.AppendLog(model.LogEntry{Level: model.LogError, Source: "service", Message: "permission denied"}); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}

	got, err := s.Logs(0)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Logs returned %d entries, want 2", len(got))
	}
	if got[0].Level != model.LogError {
		t.Errorf("Logs should be newest-first; got %+v", got[0])
	}
}

// --- snapshots -----------------------------------------------------------

func TestSnapshotRoundTrip(t *testing.T) {
	src := newTestStore(t)
	if err := src.SaveAlias(model.Alias{Name: "frontend", Port: 3000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}
	if err := src.SaveGroup(model.Group{Name: "dev", Ports: []int{3000}}); err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}
	if err := src.SaveBlueprint(model.Blueprint{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "api", Port: 8080}},
	}); err != nil {
		t.Fatalf("SaveBlueprint: %v", err)
	}

	snap, err := src.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Version != opsync.SnapshotVersion {
		t.Errorf("snapshot version = %d; want %d", snap.Version, opsync.SnapshotVersion)
	}

	// Applying it to a second store reproduces the state exactly.
	dst := newTestStore(t)
	if err := dst.ApplySnapshot(snap); err != nil {
		t.Fatalf("ApplySnapshot: %v", err)
	}

	aliases, _ := dst.Aliases()
	if len(aliases) != 1 || aliases[0].Name != "frontend" {
		t.Errorf("aliases after apply = %+v", aliases)
	}
	groups, _ := dst.Groups()
	if len(groups) != 1 || groups[0].Name != "dev" {
		t.Errorf("groups after apply = %+v", groups)
	}
	blueprints, _ := dst.Blueprints()
	if len(blueprints) != 1 || blueprints[0].Name != "web" {
		t.Errorf("blueprints after apply = %+v", blueprints)
	}
}

func TestApplySnapshotRejectsNewerVersion(t *testing.T) {
	s := newTestStore(t)
	snap := opsync.Snapshot{Version: opsync.SnapshotVersion + 1}
	if err := s.ApplySnapshot(snap); err == nil {
		t.Error("a snapshot from a newer build should be refused, not partially applied")
	}
}

// --- durability ----------------------------------------------------------

func TestReadsSurviveCorruptFile(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.path(fileAliases), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := s.Aliases(); err == nil {
		t.Error("a corrupt aliases file should surface an error the UI can show")
	}
}

func TestWritesAreAtomic(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveAlias(model.Alias{Name: "frontend", Port: 3000}); err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}

	// No staging files should be left behind after a successful write.
	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			t.Errorf("temp file %q left behind after save", e.Name())
		}
	}
}
