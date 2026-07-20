package tui

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	"github.com/SinghAman21/openport/internal/ports"
	opsync "github.com/SinghAman21/openport/internal/sync"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/msg"
)

// --- fakes ---------------------------------------------------------------
//
// The UI is exercised against fake dependencies, so these tests never read
// /proc, never signal a process and never touch the disk.

type fakeService struct {
	list []ports.Port

	killCalls  []killCall
	watchCalls int
	watchCh    chan []ports.Port
	cancelled  bool
}

type killCall struct {
	port int
	opts ports.KillOptions
}

func (f *fakeService) List(context.Context) ([]ports.Port, error) {
	return f.list, nil
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
	f.killCalls = append(f.killCalls, killCall{port, opts})
	return ports.KillResult{Port: port, PID: 1, Process: "node", Forced: opts.Force}, nil
}

func (f *fakeService) KillPID(_ context.Context, pid int, force bool) (ports.KillResult, error) {
	f.killCalls = append(f.killCalls, killCall{-pid, ports.KillOptions{Force: force}})
	return ports.KillResult{PID: pid}, nil
}

func (f *fakeService) KillRange(ctx context.Context, lo, hi int, force bool) ([]ports.KillResult, error) {
	return f.KillRangeWith(ctx, lo, hi, ports.KillOptions{Force: force})
}

func (f *fakeService) KillRangeWith(_ context.Context, lo, hi int, opts ports.KillOptions) ([]ports.KillResult, error) {
	f.killCalls = append(f.killCalls, killCall{lo * 100000, opts})
	return []ports.KillResult{{Port: lo}}, nil
}

func (f *fakeService) SuggestFree(context.Context, int, int) ([]int, error) {
	return []int{4000, 4001}, nil
}
func (f *fakeService) Reserve(context.Context, int, string) error { return nil }

func (f *fakeService) Health(context.Context, int, ports.HealthKind) (ports.HealthResult, error) {
	return ports.HealthResult{OK: true, Status: "open", LatencyMS: 1}, nil
}

func (f *fakeService) Watch(context.Context, time.Duration) (<-chan []ports.Port, func()) {
	f.watchCalls++
	f.watchCh = make(chan []ports.Port, 1)
	return f.watchCh, func() { f.cancelled = true }
}

type fakeStore struct {
	aliases   []model.Alias
	groups    []model.Group
	blueprint []model.Blueprint
	protected model.ProtectedPorts
	saved     []string
}

func (f *fakeStore) Aliases() ([]model.Alias, error) { return f.aliases, nil }
func (f *fakeStore) SaveAlias(a model.Alias) error {
	f.saved = append(f.saved, "alias:"+a.Name)
	return nil
}
func (f *fakeStore) DeleteAlias(n string) error {
	f.saved = append(f.saved, "del-alias:"+n)
	return nil
}
func (f *fakeStore) Groups() ([]model.Group, error) { return f.groups, nil }
func (f *fakeStore) SaveGroup(g model.Group) error {
	f.saved = append(f.saved, "group:"+g.Name)
	return nil
}
func (f *fakeStore) DeleteGroup(n string) error {
	f.saved = append(f.saved, "del-group:"+n)
	return nil
}
func (f *fakeStore) Blueprints() ([]model.Blueprint, error) { return f.blueprint, nil }
func (f *fakeStore) SaveBlueprint(b model.Blueprint) error {
	f.saved = append(f.saved, "blueprint:"+b.Name)
	return nil
}
func (f *fakeStore) DeleteBlueprint(n string) error {
	f.saved = append(f.saved, "del-blueprint:"+n)
	return nil
}
func (f *fakeStore) ProtectedPorts() (model.ProtectedPorts, error) { return f.protected, nil }
func (f *fakeStore) SaveProtectedPorts(p model.ProtectedPorts) error {
	f.protected = p
	f.saved = append(f.saved, "protected")
	return nil
}
func (f *fakeStore) ProjectConfig(string) (model.ProjectConfig, bool, error) {
	return model.ProjectConfig{}, false, nil
}
func (f *fakeStore) SaveProjectConfig(string, model.ProjectConfig) error {
	f.saved = append(f.saved, "project")
	return nil
}
func (f *fakeStore) AppendHistory(model.HistoryEntry) error    { return nil }
func (f *fakeStore) History(int) ([]model.HistoryEntry, error) { return nil, nil }
func (f *fakeStore) AppendLog(model.LogEntry) error            { return nil }
func (f *fakeStore) Logs(int) ([]model.LogEntry, error)        { return nil, nil }
func (f *fakeStore) Snapshot() (opsync.Snapshot, error)        { return opsync.Snapshot{}, nil }
func (f *fakeStore) ApplySnapshot(opsync.Snapshot) error       { return nil }

// proProvider is a SyncProvider that claims Pro without a network client, so
// the gating logic can be tested on both sides.
type proProvider struct{ opsync.NoopProvider }

func (proProvider) IsPro() bool { return true }

// --- harness -------------------------------------------------------------

type harness struct {
	t          *testing.T
	model      Model
	svc        *fakeService
	store      *fakeStore
	lastResult tea.Msg
	steps      int
}

func newHarness(t *testing.T, provider opsync.SyncProvider) *harness {
	t.Helper()

	svc := &fakeService{list: samplePorts()}
	st := &fakeStore{
		groups:    []model.Group{{Name: "dev_stack", Ports: []int{3000, 5432}}},
		blueprint: []model.Blueprint{{Name: "web", Services: []model.BlueprintService{{Name: "api", Port: 3000}}}},
	}
	if provider == nil {
		provider = opsync.NoopProvider{}
	}

	h := &harness{t: t, svc: svc, store: st}
	h.model = New(svc, st, provider, msg.RoutePorts)
	h.send(tea.WindowSizeMsg{Width: 140, Height: 40})
	h.send(msg.PortsLoadedMsg{Ports: svc.list})
	h.send(msg.DataLoadedMsg{Groups: st.groups, Blueprints: st.blueprint})
	return h
}

func samplePorts() []ports.Port {
	return []ports.Port{
		{
			Socket: platform.Socket{Port: 3000, Protocol: "tcp", PID: 100, Process: "node", State: "LISTEN", Addr: "0.0.0.0"},
			Alias:  "frontend",
		},
		{
			Socket: platform.Socket{Port: 5432, Protocol: "tcp", PID: 200, Process: "postgres", State: "LISTEN", Addr: "127.0.0.1"},
		},
		{
			Socket:    platform.Socket{Port: 22, Protocol: "tcp", PID: 300, Process: "sshd", State: "LISTEN", Addr: "0.0.0.0"},
			Protected: true,
			Reason:    "port 22 is protected",
		},
	}
}

// cmdBudget bounds how long a command may take before the harness gives up on
// it. Two commands legitimately never return promptly — the watch subscription
// blocks on its channel, and toast expiry is a multi-second timer — and both
// run on their own goroutine under the real runtime. Dropping them keeps the
// pump synchronous without pretending they do not exist.
const cmdBudget = 50 * time.Millisecond

// maxPumpDepth stops a message loop from hanging a test.
const maxPumpDepth = 32

// maxPumpSteps bounds total dispatches per send. A result that broadcasts to
// every view fans out into a batch, and each of those can produce more work —
// so depth alone permits exponential blowup.
const maxPumpSteps = 400

// send delivers a message and then pumps the resulting commands back into the
// model, the way the Bubble Tea runtime does. Without this, an intent emitted
// by a view would never reach the root dispatcher and the test would be
// asserting against a half-wired app.
func (h *harness) send(m tea.Msg) tea.Msg {
	h.t.Helper()
	h.lastResult = nil
	h.steps = 0
	h.dispatch(m, 0)
	return h.lastResult
}

func (h *harness) dispatch(m tea.Msg, depth int) {
	if m == nil || depth > maxPumpDepth {
		return
	}
	h.steps++
	if h.steps > maxPumpSteps {
		return
	}
	updated, cmd := h.model.Update(m)
	h.model = updated.(Model)
	h.runCmd(cmd, depth+1)
}

func (h *harness) runCmd(cmd tea.Cmd, depth int) {
	if cmd == nil || depth > maxPumpDepth || h.steps > maxPumpSteps {
		return
	}

	// Run off-goroutine so a blocking command cannot wedge the test.
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	var out tea.Msg
	select {
	case out = <-done:
	case <-time.After(cmdBudget):
		return
	}

	switch t := out.(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range t {
			h.runCmd(c, depth+1)
		}
	default:
		h.lastResult = out
		h.dispatch(out, depth+1)
	}
}

// sendKey delivers a keypress.
func (h *harness) sendKey(s string) tea.Msg {
	h.t.Helper()
	switch s {
	case "enter":
		return h.send(tea.KeyMsg{Type: tea.KeyEnter})
	case "esc":
		return h.send(tea.KeyMsg{Type: tea.KeyEsc})
	default:
		return h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	}
}

func (h *harness) hasModal() bool { return h.model.modal != nil }

// selectPort moves the table cursor onto a port. The ports table sorts by
// port number, so the default selection is whatever is numerically lowest —
// tests must say which row they mean rather than assume.
func (h *harness) selectPort(port int) {
	h.t.Helper()
	for i := 0; i < 16; i++ {
		if p, ok := h.model.portsView.Selection(); ok && p.Port == port {
			return
		}
		h.sendKey("j")
	}
	h.t.Fatalf("could not move the selection to port %d", port)
}

// --- destructive actions are gated ---------------------------------------

func TestKillKeyOpensConfirmationRatherThanKilling(t *testing.T) {
	h := newHarness(t, nil)
	h.selectPort(3000)

	h.sendKey("x")

	if !h.hasModal() {
		t.Fatal("pressing kill should open a confirmation modal")
	}
	if len(h.svc.killCalls) != 0 {
		t.Fatalf("kill was dispatched before confirmation: %+v", h.svc.killCalls)
	}
}

func TestConfirmingDispatchesTheKill(t *testing.T) {
	h := newHarness(t, nil)
	h.selectPort(3000)
	h.sendKey("x")

	// Answering the confirmation resolves the whole chain.
	h.sendKey("y")

	if h.hasModal() {
		t.Error("modal should close once answered")
	}
	if len(h.svc.killCalls) != 1 {
		t.Fatalf("expected exactly one kill, got %+v", h.svc.killCalls)
	}
	if h.svc.killCalls[0].port != 3000 {
		t.Errorf("killed port %d; want the selected row, 3000", h.svc.killCalls[0].port)
	}
	if h.svc.killCalls[0].opts.OverrideProtected {
		t.Error("a plain kill must not carry the protected override")
	}
}

func TestCancellingDoesNotKill(t *testing.T) {
	h := newHarness(t, nil)
	h.selectPort(3000)
	h.sendKey("x")

	h.sendKey("n")

	if len(h.svc.killCalls) != 0 {
		t.Fatalf("cancelling still killed something: %+v", h.svc.killCalls)
	}
}

// TestProtectedPortNeedsTwoConfirmations pins the rule that overriding
// protection is never one keystroke away.
func TestProtectedPortNeedsTwoConfirmations(t *testing.T) {
	h := newHarness(t, nil)

	// Target the protected port directly.
	h.send(msg.KillPortMsg{Port: 22})
	if !h.hasModal() {
		t.Fatal("no modal opened for a protected port")
	}

	// First gate: override the protection at all?
	first := h.sendKey("y")
	if confirm, ok := first.(msg.ConfirmResultMsg); ok && confirm.ID != confirmKillProtected {
		t.Fatalf("first gate was %q, want the protection gate", confirm.ID)
	}

	if len(h.svc.killCalls) != 0 {
		t.Fatal("overriding protection killed immediately; a second confirmation is required")
	}
	if !h.hasModal() {
		t.Fatal("a second confirmation should have opened")
	}

	// Second gate: really do it.
	second := h.sendKey("y")
	if confirm, ok := second.(msg.ConfirmResultMsg); ok && confirm.ID != confirmKill {
		t.Fatalf("second gate was %q, want the kill gate", confirm.ID)
	}

	if len(h.svc.killCalls) != 1 {
		t.Fatalf("expected one kill after both gates, got %+v", h.svc.killCalls)
	}
	if !h.svc.killCalls[0].opts.OverrideProtected {
		t.Error("the override flag should be set once both gates are passed")
	}
}

func TestAbandoningTheProtectionGateKillsNothing(t *testing.T) {
	h := newHarness(t, nil)
	h.send(msg.KillPortMsg{Port: 22})

	h.sendKey("n")

	if len(h.svc.killCalls) != 0 {
		t.Fatalf("declining the protection gate still killed: %+v", h.svc.killCalls)
	}
	if h.hasModal() {
		t.Error("declining should close the modal, not open another")
	}
}

func TestDeleteAsksFirst(t *testing.T) {
	h := newHarness(t, nil)

	h.send(msg.DeleteAliasMsg{Name: "frontend"})
	if !h.hasModal() {
		t.Fatal("deleting should open a confirmation")
	}
	for _, s := range h.store.saved {
		if strings.HasPrefix(s, "del-alias") {
			t.Fatal("delete happened before confirmation")
		}
	}
}

// --- palette -------------------------------------------------------------

func TestPaletteOpensFiltersAndRuns(t *testing.T) {
	h := newHarness(t, nil)

	h.sendKey(":")
	if !h.model.palette.IsOpen() {
		t.Fatal("':' should open the command palette")
	}

	for _, r := range "kill range" {
		h.sendKey(string(r))
	}
	item, ok := h.model.palette.Current()
	if !ok {
		t.Fatal("no palette match for \"kill range\"")
	}
	if item.ID != actions.PortsKillRange {
		t.Errorf("top match is %q; want %q", item.ID, actions.PortsKillRange)
	}

	h.sendKey("enter")
	if h.model.palette.IsOpen() {
		t.Error("palette should close after running an action")
	}
	if !h.hasModal() {
		t.Error("kill-range should open its form")
	}
}

func TestPaletteEscClosesWithoutRunning(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey(":")
	h.sendKey("esc")

	if h.model.palette.IsOpen() {
		t.Error("esc should close the palette")
	}
	if h.hasModal() {
		t.Error("esc should not have run anything")
	}
}

// TestEveryActionDispatches guards the invariant that the catalog and the
// dispatcher stay in sync: an action reachable from the menu or the palette
// must actually do something.
func TestEveryActionDispatches(t *testing.T) {
	for _, action := range actions.All() {
		t.Run(action.ID, func(t *testing.T) {
			var provider opsync.SyncProvider = opsync.NoopProvider{}
			if action.Pro {
				provider = proProvider{}
			}
			h := newHarness(t, provider)

			updated, _ := h.model.runAction(action.ID)
			m := updated.(Model)

			// The dispatcher's fallthrough raises this warning; nothing in the
			// catalog should reach it.
			for _, toast := range m.toasts {
				if strings.Contains(toast.Text, "Unknown action") {
					t.Fatalf("action %q is in the catalog but has no dispatcher case", action.ID)
				}
			}
		})
	}
}

func TestProActionsAreGatedOffPro(t *testing.T) {
	h := newHarness(t, nil) // NoopProvider

	updated, _ := h.model.runAction(actions.ProPush)
	m := updated.(Model)

	if len(m.toasts) == 0 {
		t.Fatal("a Pro action off Pro should explain itself")
	}
	if !strings.Contains(m.toasts[len(m.toasts)-1].Text, "Pro") {
		t.Errorf("toast was %q; want it to mention Pro", m.toasts[len(m.toasts)-1].Text)
	}
}

// --- watch mode ----------------------------------------------------------

func TestWatchTogglesOnAndOff(t *testing.T) {
	h := newHarness(t, nil)

	h.sendKey("w")
	if !h.model.watching {
		t.Fatal("w should start watch mode")
	}
	if h.svc.watchCalls != 1 {
		t.Fatalf("Watch called %d times, want 1", h.svc.watchCalls)
	}

	h.sendKey("w")
	if h.model.watching {
		t.Fatal("w should stop watch mode")
	}
	if !h.svc.cancelled {
		t.Error("stopping watch mode must cancel the subscription")
	}
}

// TestWatchTickRearmsSubscription pins the property the whole design rests on:
// each tick must hand back another channel read. Without it watch mode would
// refresh exactly once and then quietly stop.
func TestWatchTickRearmsSubscription(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("w")

	updated, cmd := h.model.Update(msg.WatchTickMsg{Ports: samplePorts()})
	h.model = updated.(Model)
	if cmd == nil {
		t.Fatal("a watch tick must return a command that re-arms the subscription")
	}

	// Prove the command really reads *this* channel. Closing is the
	// deterministic way to ask: every blocked reader wakes at once, so there
	// is no race with the reader the initial arm left waiting.
	close(h.svc.watchCh)

	out := runToCompletion(t, cmd)
	if _, ok := out.(msg.WatchStoppedMsg); !ok {
		t.Fatalf("re-arm command produced %T; want it to be reading the watch channel", out)
	}
}

// runToCompletion runs a command, unwrapping tea.Batch, and returns the first
// concrete message. tea.Batch collapses to the command itself when only one is
// non-nil, so callers cannot assume either shape.
func runToCompletion(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	var out tea.Msg
	select {
	case out = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("command did not return")
	}

	if batch, ok := out.(tea.BatchMsg); ok {
		for _, c := range batch {
			if m := runToCompletion(t, c); m != nil {
				return m
			}
		}
		return nil
	}
	return out
}

func TestQuitCancelsWatch(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("w")
	h.sendKey("q")

	if !h.svc.cancelled {
		t.Error("quitting must cancel the watcher so its goroutine does not outlive the program")
	}
}

func TestWatchStoppedClearsState(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("w")
	h.send(msg.WatchStoppedMsg{})

	if h.model.watching {
		t.Error("a closed watch channel should clear the watching flag")
	}
}

// --- input routing -------------------------------------------------------

func TestFilterCapturesKeysThatWouldOtherwiseQuit(t *testing.T) {
	h := newHarness(t, nil)

	h.sendKey("/") // enter filter mode
	h.sendKey("q") // must be typed, not treated as quit

	if h.model.quit {
		t.Fatal("typing q into the filter quit the application")
	}
	if !h.model.portsView.CapturingInput() {
		t.Fatal("the ports view should still be capturing input")
	}
}

func TestModalSwallowsGlobalKeys(t *testing.T) {
	h := newHarness(t, nil)
	h.send(msg.RunActionMsg{ID: actions.PortsCheck})
	if !h.hasModal() {
		t.Fatal("check should open a form")
	}

	h.sendKey("q")
	if h.model.quit {
		t.Fatal("q reached the global handler while a modal was open")
	}
}

func TestNavigationKeysSwitchViews(t *testing.T) {
	h := newHarness(t, nil)

	tests := []struct {
		key  string
		want msg.Route
	}{
		{"2", msg.RouteActions},
		{"3", msg.RouteAliases},
		{"4", msg.RouteBlueprints},
		{"5", msg.RouteConfig},
		{"6", msg.RouteHistory},
		{"7", msg.RouteWorkspace},
		{"1", msg.RoutePorts},
	}
	for _, tt := range tests {
		h.sendKey(tt.key)
		if h.model.route != tt.want {
			t.Errorf("after %q route = %v; want %v", tt.key, h.model.route, tt.want)
		}
	}
}

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	h := newHarness(t, nil)

	h.sendKey("?")
	if !h.hasModal() {
		t.Fatal("? should open the help overlay")
	}

	h.sendKey("esc")
	if h.hasModal() {
		t.Error("esc should close the help overlay")
	}
}

// --- form results --------------------------------------------------------

func TestKillFormRoutesThroughConfirmation(t *testing.T) {
	h := newHarness(t, nil)

	h.send(msg.FormResultMsg{ID: formKill, Values: map[string]string{
		"port": "3000", "force": "true",
	}})

	if !h.hasModal() {
		t.Fatal("submitting the kill form should open a confirmation, not kill")
	}
	if len(h.svc.killCalls) != 0 {
		t.Fatalf("kill form killed without confirmation: %+v", h.svc.killCalls)
	}
}

func TestAliasFormSaves(t *testing.T) {
	h := newHarness(t, nil)

	h.send(msg.FormResultMsg{ID: formAlias, Values: map[string]string{
		"name": "frontend", "port": "3000", "note": "",
	}})

	found := false
	for _, s := range h.store.saved {
		if s == "alias:frontend" {
			found = true
		}
	}
	if !found {
		t.Fatalf("alias was not saved; store recorded %v", h.store.saved)
	}
}

func TestFormRejectsBadInputWithoutSaving(t *testing.T) {
	h := newHarness(t, nil)

	h.send(msg.FormResultMsg{ID: formGroup, Values: map[string]string{
		"name": "bad", "ports": "not-a-port",
	}})

	if len(h.store.saved) != 0 {
		t.Fatalf("an invalid group was saved: %v", h.store.saved)
	}
	if len(h.model.toasts) == 0 {
		t.Error("the user should be told why nothing was saved")
	}
}

// --- rendering -----------------------------------------------------------

// TestViewRendersAtManySizes guards against panics and runaway lines in the
// layout maths — a TUI that crashes on a narrow terminal is a TUI that crashes
// in a split pane.
func TestViewRendersAtManySizes(t *testing.T) {
	sizes := []struct{ w, h int }{
		{200, 60}, {140, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}, {20, 6},
	}
	routes := []msg.Route{
		msg.RoutePorts, msg.RouteActions, msg.RouteAliases,
		msg.RouteBlueprints, msg.RouteConfig, msg.RouteHistory, msg.RouteWorkspace,
	}

	for _, size := range sizes {
		for _, route := range routes {
			h := newHarness(t, nil)
			h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			h.model.route = route

			out := h.model.View()
			if out == "" {
				t.Fatalf("%v at %dx%d rendered nothing", route, size.w, size.h)
			}
			assertFits(t, out, size.w, size.h,
				func() string { return route.String() })
		}
	}
}

func TestOverlaysRenderWithoutBreakingLayout(t *testing.T) {
	overlays := map[string]func(*harness){
		"palette":           func(h *harness) { h.sendKey(":") },
		"help":              func(h *harness) { h.sendKey("?") },
		"confirm":           func(h *harness) { h.selectPort(3000); h.sendKey("x") },
		"confirm-protected": func(h *harness) { h.send(msg.KillPortMsg{Port: 22}) },
		"form":              func(h *harness) { h.send(msg.RunActionMsg{ID: actions.PortsKillRange}) },
		"toast":             func(h *harness) { h.send(msg.Success("saved something")) },
	}

	for name, open := range overlays {
		t.Run(name, func(t *testing.T) {
			for _, size := range []struct{ w, h int }{{140, 40}, {80, 24}, {50, 14}} {
				h := newHarness(t, nil)
				h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
				open(h)

				out := h.model.View()
				assertFits(t, out, size.w, size.h, func() string { return name })
			}
		})
	}
}

// assertFits checks a rendered frame stays inside the terminal box.
func assertFits(t *testing.T, out string, width, height int, label func() string) {
	t.Helper()

	lines := strings.Split(out, "\n")
	if len(lines) > height {
		t.Errorf("%s at %dx%d rendered %d lines; want at most %d",
			label(), width, height, len(lines), height)
	}
	for i, line := range lines {
		if w := lineWidth(line); w > width {
			t.Errorf("%s at %dx%d: line %d is %d cells wide, overflowing by %d",
				label(), width, height, i, w, w-width)
			return // one report per frame is enough
		}
	}
}

// lineWidth measures a rendered line in terminal cells, ignoring ANSI.
func lineWidth(s string) int { return lipgloss.Width(s) }

// --- every popup must be escapable ---------------------------------------

// TestEveryPopupClosesWithEsc is the property that matters most for trusting
// the interface: whatever you opened, one key gets you out. A popup you cannot
// leave is worse than a missing feature, because it strands the user with no
// obvious recovery short of killing the process.
func TestEveryPopupClosesWithEsc(t *testing.T) {
	for _, action := range actions.All() {
		t.Run(action.ID, func(t *testing.T) {
			var provider opsync.SyncProvider = opsync.NoopProvider{}
			if action.Pro {
				provider = proProvider{}
			}
			h := newHarness(t, provider)
			h.send(msg.RunActionMsg{ID: action.ID})

			if !h.hasModal() {
				return // this action does not open a popup
			}

			h.sendKey("esc")
			if h.hasModal() {
				t.Fatalf("%s opened a popup that esc will not close", action.ID)
			}
		})
	}
}

// TestPopupsCloseWithEscFromEveryView checks the escape hatch is a property of
// the modal layer, not of whichever view happened to open it.
func TestPopupsCloseWithEscFromEveryView(t *testing.T) {
	routes := []msg.Route{
		msg.RoutePorts, msg.RouteActions, msg.RouteAliases,
		msg.RouteBlueprints, msg.RouteConfig, msg.RouteHistory, msg.RouteWorkspace,
	}
	for _, route := range routes {
		h := newHarness(t, nil)
		h.model.route = route
		h.send(msg.RunActionMsg{ID: actions.PortsCheck})

		if !h.hasModal() {
			t.Fatalf("%v: no popup opened", route)
		}
		h.sendKey("esc")
		if h.hasModal() {
			t.Errorf("%v: esc did not close the popup", route)
		}
	}
}

// TestActionsViewEnterOpensAndEscapes walks the path the user described:
// browse the actions menu, run an entry, then leave again.
func TestActionsViewEnterOpensAndEscapes(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("2")
	if h.model.route != msg.RouteActions {
		t.Fatal("2 should open the actions menu")
	}

	// Walk down the menu, running whatever is highlighted, and make sure each
	// one either does something visible or opens an escapable popup.
	for i := 0; i < 12; i++ {
		item, ok := h.model.actionsView.Current()
		if !ok {
			h.sendKey("j")
			continue
		}

		h.sendKey("enter")
		if h.hasModal() {
			h.sendKey("esc")
			if h.hasModal() {
				t.Fatalf("%q opened a popup that esc will not close", item.ID)
			}
		}

		// Return to the menu for the next iteration.
		h.model.route = msg.RouteActions
		h.sendKey("j")
	}
}

// --- popups must fit on screen -------------------------------------------

// TestPopupsFitTheTerminal guards the other half of the complaint: a popup
// taller than the window loses its own footer, which is where the keys that
// dismiss it are documented.
func TestPopupsFitTheTerminal(t *testing.T) {
	sizes := []struct{ w, h int }{{140, 40}, {100, 30}, {80, 24}, {60, 18}, {50, 12}}
	openers := map[string]string{
		"check form":     actions.PortsCheck,
		"kill form":      actions.PortsKill,
		"range form":     actions.PortsKillRange,
		"alias form":     actions.AliasNew,
		"blueprint form": actions.BlueprintNew,
		"config form":    actions.ConfigInit,
		"help overlay":   actions.UIHelp,
	}

	for name, id := range openers {
		for _, size := range sizes {
			h := newHarness(t, nil)
			h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			h.send(msg.RunActionMsg{ID: id})
			if !h.hasModal() {
				continue
			}

			block := h.model.modal.View(size.w, size.h)
			if got := lipgloss.Height(block); got > size.h {
				t.Errorf("%s at %dx%d: popup is %d rows tall, taller than the window",
					name, size.w, size.h, got)
			}
			if got := lipgloss.Width(block); got > size.w {
				t.Errorf("%s at %dx%d: popup is %d cells wide, wider than the window",
					name, size.w, size.h, got)
			}
		}
	}
}

// --- responsiveness: degrade, do not disappear ---------------------------

// TestHeaderKeepsNavigationDiscoverable checks the tab strip degrades to
// numbers rather than collapsing to the active view. The strip is the only
// place the other screens and their shortcuts are advertised, so dropping it
// hides the navigation from anyone who has not memorised it.
func TestHeaderKeepsNavigationDiscoverable(t *testing.T) {
	for _, width := range []int{200, 140, 120, 100, 90, 80, 70, 60, 50, 40} {
		h := newHarness(t, nil)
		h.send(tea.WindowSizeMsg{Width: width, Height: 30})

		header := h.model.renderHeader()
		if got := lipgloss.Width(header); got > width {
			t.Errorf("width %d: header is %d cells wide", width, got)
		}

		// Every route's number should still be reachable knowledge.
		plain := ansi.Strip(header)
		for i := range msg.AllRoutes {
			if !strings.Contains(plain, strconv.Itoa(i+1)) {
				t.Errorf("width %d: header %q hides the shortcut for route %d",
					width, plain, i+1)
				break
			}
		}
		// And it should always say where we are.
		if !strings.Contains(plain, h.model.route.String()) {
			t.Errorf("width %d: header %q does not name the active view", width, plain)
		}
	}
}

// TestOverflowingMenuSaysSo guards against the failure the user hit: a menu
// showing a handful of its entries with no sign the rest exist.
func TestOverflowingMenuSaysSo(t *testing.T) {
	for _, height := range []int{30, 24, 20, 16, 12} {
		h := newHarness(t, nil)
		h.send(tea.WindowSizeMsg{Width: 100, Height: height})
		h.sendKey("2")

		frame := ansi.Strip(h.model.View())
		total := len(actions.All())
		shown := 0
		for _, a := range actions.All() {
			if strings.Contains(frame, a.Title) {
				shown++
			}
		}
		if shown >= total {
			continue // everything fits; no indicator needed
		}
		if !strings.Contains(frame, "below") && !strings.Contains(frame, "above") {
			t.Errorf("height %d: menu shows %d of %d actions with no scroll indicator:\n%s",
				height, shown, total, frame)
		}
	}
}

// TestMenuScrollsToEveryEntry checks the whole catalog stays reachable on a
// short terminal — an indicator is only honest if the content it points at can
// actually be scrolled to.
func TestMenuScrollsToEveryEntry(t *testing.T) {
	h := newHarness(t, nil)
	h.send(tea.WindowSizeMsg{Width: 100, Height: 14})
	h.sendKey("2")

	seen := map[string]bool{}
	for i := 0; i < len(actions.All())+10; i++ {
		frame := ansi.Strip(h.model.View())
		for _, a := range actions.All() {
			if strings.Contains(frame, a.Title) {
				seen[a.ID] = true
			}
		}
		h.sendKey("j")
	}

	for _, a := range actions.All() {
		if !seen[a.ID] {
			t.Errorf("%q never became visible while scrolling", a.Title)
		}
	}
}

// TestDetailPaneStacksRatherThanVanishing pins that narrowing the terminal
// moves the detail pane instead of discarding the only place the full command
// line and protection reason are shown.
func TestDetailPaneStacksRatherThanVanishing(t *testing.T) {
	// Wide: beside the table. Narrow but tall: underneath. Both must show it.
	for _, size := range []struct{ w, h int }{{140, 40}, {80, 30}, {70, 24}} {
		h := newHarness(t, nil)
		h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		h.selectPort(22) // the protected row, which has a reason to show

		frame := ansi.Strip(h.model.View())
		if !strings.Contains(frame, "sshd") {
			t.Errorf("%dx%d: detail pane is missing the selection:\n%s", size.w, size.h, frame)
		}
	}
}

// TestDetailPaneToggles checks the user can reclaim the space when they want
// the table instead.
func TestDetailPaneToggles(t *testing.T) {
	h := newHarness(t, nil)
	h.send(tea.WindowSizeMsg{Width: 140, Height: 40})
	h.selectPort(22)

	if !strings.Contains(ansi.Strip(h.model.View()), "Protocol") {
		t.Fatal("detail pane should be shown by default")
	}
	h.sendKey("tab")
	if strings.Contains(ansi.Strip(h.model.View()), "Protocol") {
		t.Error("tab should hide the detail pane")
	}
	h.sendKey("tab")
	if !strings.Contains(ansi.Strip(h.model.View()), "Protocol") {
		t.Error("tab should bring the detail pane back")
	}
}
