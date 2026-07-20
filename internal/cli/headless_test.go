package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	"github.com/SinghAman21/openport/internal/ports"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// --- fakes ---------------------------------------------------------------

type fakeService struct {
	list      []ports.Port
	killed    []int
	forced    []bool
	listErr   error
	killSkips map[int]string
}

func (f *fakeService) List(context.Context) ([]ports.Port, error) {
	return f.list, f.listErr
}

func (f *fakeService) Check(_ context.Context, port int) (ports.Port, bool, error) {
	for _, p := range f.list {
		if p.Port == port {
			return p, true, nil
		}
	}
	return ports.Port{Socket: platform.Socket{Port: port}}, false, nil
}

func (f *fakeService) Kill(ctx context.Context, port int, force bool) (ports.KillResult, error) {
	return f.KillWith(ctx, port, ports.KillOptions{Force: force})
}

func (f *fakeService) KillWith(_ context.Context, port int, opts ports.KillOptions) (ports.KillResult, error) {
	if reason, skip := f.killSkips[port]; skip {
		return ports.KillResult{Port: port, Skipped: true, Reason: reason}, nil
	}
	f.killed = append(f.killed, port)
	f.forced = append(f.forced, opts.Force)
	return ports.KillResult{Port: port, PID: 42, Process: "node", Forced: opts.Force}, nil
}

func (f *fakeService) KillPID(context.Context, int, bool) (ports.KillResult, error) {
	return ports.KillResult{}, nil
}

func (f *fakeService) KillRange(ctx context.Context, lo, hi int, force bool) ([]ports.KillResult, error) {
	return f.KillRangeWith(ctx, lo, hi, ports.KillOptions{Force: force})
}

func (f *fakeService) KillRangeWith(ctx context.Context, lo, hi int, opts ports.KillOptions) ([]ports.KillResult, error) {
	var out []ports.KillResult
	for _, p := range f.list {
		if p.Port < lo || p.Port > hi {
			continue
		}
		res, _ := f.KillWith(ctx, p.Port, opts)
		out = append(out, res)
	}
	return out, nil
}

// SuggestFree honours count, so tests exercise the same contract the real
// service offers rather than a convenient fiction.
func (f *fakeService) SuggestFree(_ context.Context, near, count int) ([]int, error) {
	if near <= 0 {
		near = 4000
	}
	if count <= 0 {
		count = 1
	}
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, 4000+i)
	}
	return out, nil
}
func (f *fakeService) Reserve(context.Context, int, string) error { return nil }
func (f *fakeService) Health(context.Context, int, ports.HealthKind) (ports.HealthResult, error) {
	return ports.HealthResult{OK: true}, nil
}
func (f *fakeService) Watch(context.Context, time.Duration) (<-chan []ports.Port, func()) {
	ch := make(chan []ports.Port)
	return ch, func() { close(ch) }
}

type fakeStore struct{ blueprints []model.Blueprint }

func (f *fakeStore) Aliases() ([]model.Alias, error)        { return nil, nil }
func (f *fakeStore) SaveAlias(model.Alias) error            { return nil }
func (f *fakeStore) DeleteAlias(string) error               { return nil }
func (f *fakeStore) Groups() ([]model.Group, error)         { return nil, nil }
func (f *fakeStore) SaveGroup(model.Group) error            { return nil }
func (f *fakeStore) DeleteGroup(string) error               { return nil }
func (f *fakeStore) Blueprints() ([]model.Blueprint, error) { return f.blueprints, nil }
func (f *fakeStore) SaveBlueprint(model.Blueprint) error    { return nil }
func (f *fakeStore) DeleteBlueprint(string) error           { return nil }
func (f *fakeStore) ProtectedPorts() (model.ProtectedPorts, error) {
	return model.ProtectedPorts{}, nil
}
func (f *fakeStore) SaveProtectedPorts(model.ProtectedPorts) error { return nil }
func (f *fakeStore) ProjectConfig(string) (model.ProjectConfig, bool, error) {
	return model.ProjectConfig{}, false, nil
}
func (f *fakeStore) SaveProjectConfig(string, model.ProjectConfig) error { return nil }
func (f *fakeStore) AppendHistory(model.HistoryEntry) error              { return nil }
func (f *fakeStore) History(int) ([]model.HistoryEntry, error)           { return nil, nil }
func (f *fakeStore) AppendLog(model.LogEntry) error                      { return nil }
func (f *fakeStore) Logs(int) ([]model.LogEntry, error)                  { return nil, nil }
func (f *fakeStore) Snapshot() (opsync.Snapshot, error)                  { return opsync.Snapshot{}, nil }
func (f *fakeStore) ApplySnapshot(opsync.Snapshot) error                 { return nil }

// --- harness -------------------------------------------------------------

type result struct {
	code   int
	stdout string
	stderr string
}

// runCLI executes a command against fakes with scripted stdin.
func runCLI(args []string, svc *fakeService, st *fakeStore, stdin string, interactive bool) result {
	var out, errBuf bytes.Buffer
	env := Env{
		Stdin:       strings.NewReader(stdin),
		Stdout:      &out,
		Stderr:      &errBuf,
		Interactive: interactive,
	}
	code := Run(args, svc, st, env)
	return result{code, out.String(), errBuf.String()}
}

func sampleService() *fakeService {
	return &fakeService{
		list: []ports.Port{
			{Socket: platform.Socket{Port: 3000, Protocol: "tcp", PID: 100, Process: "node", State: "LISTEN"}, Alias: "frontend"},
			{Socket: platform.Socket{Port: 5432, Protocol: "tcp", PID: 200, Process: "postgres", State: "LISTEN"}},
		},
	}
}

// --- listing & inspection ------------------------------------------------

func TestList(t *testing.T) {
	got := runCLI([]string{"ls"}, sampleService(), &fakeStore{}, "", false)

	if got.code != ExitOK {
		t.Fatalf("exit = %d; want %d (stderr: %s)", got.code, ExitOK, got.stderr)
	}
	for _, want := range []string{"PORT", "3000", "node", "frontend", "5432", "postgres"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output missing %q:\n%s", want, got.stdout)
		}
	}
}

func TestListEmpty(t *testing.T) {
	got := runCLI([]string{"ls"}, &fakeService{}, &fakeStore{}, "", false)
	if got.code != ExitOK {
		t.Fatalf("exit = %d; want %d", got.code, ExitOK)
	}
	if !strings.Contains(got.stdout, "Nothing is listening") {
		t.Errorf("expected a plain-language empty state, got:\n%s", got.stdout)
	}
}

func TestListPropagatesFailure(t *testing.T) {
	svc := sampleService()
	svc.listErr = errors.New("procfs unreadable")

	got := runCLI([]string{"ls"}, svc, &fakeStore{}, "", false)
	if got.code != ExitError {
		t.Errorf("exit = %d; want %d", got.code, ExitError)
	}
	if !strings.Contains(got.stderr, "procfs unreadable") {
		t.Errorf("the cause should reach stderr, got: %q", got.stderr)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"occupied", []string{"check", "3000"}, ExitOK, "node"},
		{"free", []string{"check", "9999"}, ExitOK, "is free"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCLI(tt.args, sampleService(), &fakeStore{}, "", false)
			if got.code != tt.wantCode {
				t.Errorf("exit = %d; want %d", got.code, tt.wantCode)
			}
			if !strings.Contains(got.stdout, tt.wantOut) {
				t.Errorf("output missing %q:\n%s", tt.wantOut, got.stdout)
			}
		})
	}
}

func TestCheckRejectsBadPort(t *testing.T) {
	for _, arg := range []string{"abc", "0", "70000", ""} {
		got := runCLI([]string{"check", arg}, sampleService(), &fakeStore{}, "", false)
		if got.code == ExitOK {
			t.Errorf("check %q should have failed", arg)
		}
	}
}

func TestCheckWithoutArgument(t *testing.T) {
	got := runCLI([]string{"check"}, sampleService(), &fakeStore{}, "", false)
	if got.code == ExitOK {
		t.Error("check with no port should fail")
	}
	if !strings.Contains(got.stderr, "usage") {
		t.Errorf("a usage hint should be printed, got: %q", got.stderr)
	}
}

// --- destructive commands ------------------------------------------------

// TestKillRefusesWithoutATerminal pins the scripting guard: a cron job that
// forgot --force must fail loudly rather than block on a prompt nobody can
// answer.
func TestKillRefusesWithoutATerminal(t *testing.T) {
	svc := sampleService()
	got := runCLI([]string{"kill", "3000"}, svc, &fakeStore{}, "", false)

	if got.code != ExitUsage {
		t.Errorf("exit = %d; want %d", got.code, ExitUsage)
	}
	if len(svc.killed) != 0 {
		t.Fatalf("killed without confirmation: %v", svc.killed)
	}
	if !strings.Contains(got.stderr, "--force") {
		t.Errorf("the error should say how to proceed, got: %q", got.stderr)
	}
}

func TestKillPromptsWhenInteractive(t *testing.T) {
	tests := []struct {
		name       string
		answer     string
		wantKilled bool
	}{
		{"yes", "y\n", true},
		{"long yes", "yes\n", true},
		{"no", "n\n", false},
		{"empty defaults to no", "\n", false},
		{"anything else is no", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := sampleService()
			got := runCLI([]string{"kill", "3000"}, svc, &fakeStore{}, tt.answer, true)

			if killed := len(svc.killed) > 0; killed != tt.wantKilled {
				t.Fatalf("killed = %v; want %v (stdout: %s)", killed, tt.wantKilled, got.stdout)
			}
			if !tt.wantKilled && !strings.Contains(got.stdout, "Cancelled") {
				t.Errorf("declining should say so, got: %q", got.stdout)
			}
		})
	}
}

func TestKillForceSkipsThePrompt(t *testing.T) {
	svc := sampleService()
	got := runCLI([]string{"kill", "3000", "--force"}, svc, &fakeStore{}, "", false)

	if got.code != ExitOK {
		t.Fatalf("exit = %d; want %d (stderr: %s)", got.code, ExitOK, got.stderr)
	}
	if len(svc.killed) != 1 || svc.killed[0] != 3000 {
		t.Fatalf("killed %v; want [3000]", svc.killed)
	}
	if svc.forced[0] {
		t.Error("--force skips the prompt; it should not imply SIGKILL")
	}
	if !strings.Contains(got.stdout, "SIGTERM") {
		t.Errorf("the signal used should be reported, got: %q", got.stdout)
	}
}

func TestKillSigkillFlag(t *testing.T) {
	for _, flag := range []string{"-9", "--sigkill"} {
		svc := sampleService()
		got := runCLI([]string{"kill", "3000", "--force", flag}, svc, &fakeStore{}, "", false)

		if len(svc.forced) != 1 || !svc.forced[0] {
			t.Fatalf("%s did not request SIGKILL", flag)
		}
		if !strings.Contains(got.stdout, "SIGKILL") {
			t.Errorf("%s: output should say SIGKILL, got %q", flag, got.stdout)
		}
	}
}

// TestKillReportsProtectedSkip pins that the CLI honours the never-kill list
// and exits non-zero, so a script cannot mistake a refusal for success.
func TestKillReportsProtectedSkip(t *testing.T) {
	svc := sampleService()
	svc.killSkips = map[int]string{3000: "port 3000 is protected"}

	got := runCLI([]string{"kill", "3000", "--force"}, svc, &fakeStore{}, "", false)

	if len(svc.killed) != 0 {
		t.Fatal("a protected port was killed")
	}
	if got.code != ExitError {
		t.Errorf("exit = %d; want %d so a script notices", got.code, ExitError)
	}
	if !strings.Contains(got.stderr, "protected") {
		t.Errorf("the reason should reach stderr, got: %q", got.stderr)
	}
}

func TestKillRange(t *testing.T) {
	svc := sampleService()
	got := runCLI([]string{"kill-range", "3000", "6000", "--force"}, svc, &fakeStore{}, "", false)

	if got.code != ExitOK {
		t.Fatalf("exit = %d; want %d (stderr: %s)", got.code, ExitOK, got.stderr)
	}
	if len(svc.killed) != 2 {
		t.Fatalf("killed %v; want both listeners in range", svc.killed)
	}
}

func TestKillRangeNeedsTwoBounds(t *testing.T) {
	got := runCLI([]string{"kill-range", "3000"}, sampleService(), &fakeStore{}, "", false)
	if got.code == ExitOK {
		t.Error("kill-range with one bound should fail")
	}
}

func TestKillRangeRefusesWithoutATerminal(t *testing.T) {
	svc := sampleService()
	got := runCLI([]string{"kill-range", "3000", "6000"}, svc, &fakeStore{}, "", false)

	if got.code != ExitUsage {
		t.Errorf("exit = %d; want %d", got.code, ExitUsage)
	}
	if len(svc.killed) != 0 {
		t.Fatalf("killed without confirmation: %v", svc.killed)
	}
}

// --- free ----------------------------------------------------------------

func TestFree(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"defaults", []string{"free"}, []string{"4000"}},
		{"with count", []string{"free", "--count", "2"}, []string{"4000", "4001"}},
		{"equals form", []string{"free", "--near=8000", "--count=2"}, []string{"4000", "4001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runCLI(tt.args, sampleService(), &fakeStore{}, "", false)
			if got.code != ExitOK {
				t.Fatalf("exit = %d; want %d", got.code, ExitOK)
			}
			for _, want := range tt.want {
				if !strings.Contains(got.stdout, want) {
					t.Errorf("output missing %q:\n%s", want, got.stdout)
				}
			}
		})
	}
}

// TestFreeOutputIsScriptable checks the output is bare numbers, one per line,
// so `PORT=$(openport free)` does the obvious thing.
func TestFreeOutputIsScriptable(t *testing.T) {
	got := runCLI([]string{"free"}, sampleService(), &fakeStore{}, "", false)
	if strings.TrimSpace(got.stdout) != "4000" {
		t.Errorf("output = %q; want a bare port number", got.stdout)
	}
}

// --- apply ---------------------------------------------------------------

func TestApply(t *testing.T) {
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name: "web",
		Services: []model.BlueprintService{
			{Name: "node", Port: 3000},  // matches the live process
			{Name: "cache", Port: 6379}, // not running
		},
	}}}

	got := runCLI([]string{"apply", "web"}, sampleService(), st, "", false)
	if got.code != ExitOK {
		t.Fatalf("exit = %d; want %d (stderr: %s)", got.code, ExitOK, got.stderr)
	}
	for _, want := range []string{"SERVICE", "node", "ok", "cache", "free"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("output missing %q:\n%s", want, got.stdout)
		}
	}
}

// TestApplyExitsNonZeroOnConflict lets a script gate a dev-server start on the
// environment actually being clean.
func TestApplyExitsNonZeroOnConflict(t *testing.T) {
	st := &fakeStore{blueprints: []model.Blueprint{{
		Name:     "web",
		Services: []model.BlueprintService{{Name: "api", Port: 3000}}, // node holds it
	}}}

	got := runCLI([]string{"apply", "web"}, sampleService(), st, "", false)
	if got.code != ExitError {
		t.Errorf("exit = %d; want %d when a port is taken by the wrong thing", got.code, ExitError)
	}
	if !strings.Contains(got.stdout, "conflict") {
		t.Errorf("the conflict should be reported:\n%s", got.stdout)
	}
}

func TestApplyUnknownBlueprint(t *testing.T) {
	got := runCLI([]string{"apply", "nope"}, sampleService(), &fakeStore{}, "", false)
	if got.code != ExitError {
		t.Errorf("exit = %d; want %d", got.code, ExitError)
	}
	if !strings.Contains(got.stderr, "nope") {
		t.Errorf("the missing name should be echoed back, got: %q", got.stderr)
	}
}

// --- dispatch ------------------------------------------------------------

func TestUnknownCommand(t *testing.T) {
	got := runCLI([]string{"frobnicate"}, sampleService(), &fakeStore{}, "", false)
	if got.code != ExitUsage {
		t.Errorf("exit = %d; want %d", got.code, ExitUsage)
	}
	if !strings.Contains(got.stderr, "frobnicate") {
		t.Errorf("the unknown command should be named, got: %q", got.stderr)
	}
}

func TestNoArguments(t *testing.T) {
	got := runCLI(nil, sampleService(), &fakeStore{}, "", false)
	if got.code != ExitUsage {
		t.Errorf("exit = %d; want %d", got.code, ExitUsage)
	}
}

func TestHelpAndVersion(t *testing.T) {
	help := runCLI([]string{"help"}, sampleService(), &fakeStore{}, "", false)
	if help.code != ExitOK || !strings.Contains(help.stdout, "USAGE") {
		t.Errorf("help should print usage and succeed, got %d: %s", help.code, help.stdout)
	}

	version := runCLI([]string{"version"}, sampleService(), &fakeStore{}, "", false)
	if version.code != ExitOK || !strings.Contains(version.stdout, "openport") {
		t.Errorf("version should print a banner, got %d: %s", version.code, version.stdout)
	}
}

// TestIsSubcommand guards the dispatch in main: anything listed here launches
// headless, and anything else opens the TUI.
func TestIsSubcommand(t *testing.T) {
	for _, name := range []string{"ls", "list", "check", "kill", "kill-range", "free", "apply", "version", "help"} {
		if !IsSubcommand(name) {
			t.Errorf("%q should be recognised as a subcommand", name)
		}
	}
	for _, name := range []string{"--view", "--debug", "", "ports", "frobnicate"} {
		if IsSubcommand(name) {
			t.Errorf("%q should not be treated as a subcommand", name)
		}
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantForce bool
		wantKill9 bool
		wantNear  int
		wantCount int
		wantRest  []string
	}{
		{"bare", []string{"3000"}, false, false, 0, 0, []string{"3000"}},
		{"force long", []string{"3000", "--force"}, true, false, 0, 0, []string{"3000"}},
		{"force short", []string{"-f", "3000"}, true, false, 0, 0, []string{"3000"}},
		{"sigkill", []string{"3000", "-9"}, false, true, 0, 0, []string{"3000"}},
		{"near and count", []string{"--near", "8000", "--count", "5"}, false, false, 8000, 5, nil},
		{"equals form", []string{"--near=8000", "--count=5"}, false, false, 8000, 5, nil},
		{"two positionals", []string{"3000", "4000", "--force"}, true, false, 0, 0, []string{"3000", "4000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rest := parseFlags(tt.args)
			if got.force != tt.wantForce || got.sigkill != tt.wantKill9 ||
				got.near != tt.wantNear || got.count != tt.wantCount {
				t.Errorf("parseFlags(%v) = %+v; want force=%v sigkill=%v near=%d count=%d",
					tt.args, got, tt.wantForce, tt.wantKill9, tt.wantNear, tt.wantCount)
			}
			if len(rest) != len(tt.wantRest) {
				t.Fatalf("positional args = %v; want %v", rest, tt.wantRest)
			}
			for i := range rest {
				if rest[i] != tt.wantRest[i] {
					t.Errorf("positional args = %v; want %v", rest, tt.wantRest)
				}
			}
		})
	}
}
