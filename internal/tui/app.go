// Package tui is the primary front-end: a keyboard-driven cockpit over
// ports.Service. It depends on the service, the store and the sync provider —
// never on internal/platform, and never on a concrete remote client.
package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/store"
	opsync "github.com/SinghAman21/openport/internal/sync"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/components"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
	"github.com/SinghAman21/openport/internal/tui/views"
)

// Model is the root of the Elm-architecture tree. Every state mutation in the
// application happens inside its Update — commands only dispatch and report.
type Model struct {
	svc      ports.Service
	store    store.Store
	provider opsync.SyncProvider

	route  msg.Route
	views  map[msg.Route]views.View
	width  int
	height int

	// Typed handles onto the same view instances held in the map, for the
	// places the root needs more than the View interface.
	portsView     *views.PortsView
	actionsView   *views.ActionsView
	aliasesView   *views.AliasesView
	configView    *views.ConfigView
	historyView   *views.HistoryView
	workspaceView *views.WorkspaceView

	palette   *components.Palette
	modal     *components.Modal
	statusbar components.StatusBar
	toasts    []components.Toast
	nextToast int

	// Work in flight. Several operations can overlap — a group health check
	// fans out one probe per port — so this is a count, not a flag.
	spinner   spinner.Model
	busy      int
	busyLabel string

	// Watch state. watchCh is what the re-arming subscription reads from.
	watching    bool
	watchCh     <-chan []ports.Port
	watchCancel func()

	// lastPorts is the most recent listing, used to answer "is this port
	// protected?" when building a confirmation without a round-trip.
	lastPorts []ports.Port
	// groups and blueprints back the pickers.
	groups     []model.Group
	blueprints []model.Blueprint
	project    model.ProjectConfig
	hasProject bool

	workDir string
	keys    KeyMap
	theme   styles.Theme
	quit    bool
}

// New builds the root model with its dependencies injected.
func New(svc ports.Service, st store.Store, provider opsync.SyncProvider, initial msg.Route) Model {
	theme := styles.New()
	isPro := provider.IsPro()

	portsView := views.NewPortsView(theme)
	actionsView := views.NewActionsView(theme, isPro)
	aliasesView := views.NewAliasesView(theme)
	blueprintsView := views.NewBlueprintsView(theme)
	configView := views.NewConfigView(theme)
	historyView := views.NewHistoryView(theme)
	workspaceView := views.NewWorkspaceView(theme, isPro)

	wd, _ := os.Getwd()

	spin := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.Accent),
	)

	return Model{
		svc:      svc,
		store:    st,
		provider: provider,
		route:    initial,
		views: map[msg.Route]views.View{
			msg.RoutePorts:      portsView,
			msg.RouteActions:    actionsView,
			msg.RouteAliases:    aliasesView,
			msg.RouteBlueprints: blueprintsView,
			msg.RouteConfig:     configView,
			msg.RouteHistory:    historyView,
			msg.RouteWorkspace:  workspaceView,
		},
		portsView:     portsView,
		actionsView:   actionsView,
		aliasesView:   aliasesView,
		configView:    configView,
		historyView:   historyView,
		workspaceView: workspaceView,
		palette:       components.NewPalette(theme, views.ActionItems(isPro)),
		workDir:       wd,
		keys:          DefaultKeyMap(),
		theme:         theme,
		spinner:       spin,
		// The first listing is already on its way from Init.
		busy:      1,
		busyLabel: "reading ports",
	}
}

// Init loads the first listing and the stored collections, and starts the
// ambient poll that keeps them current.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listPortsCmd(m.svc),
		loadDataCmd(m.store, m.workDir),
		autoRefreshCmd(),
		m.spinner.Tick,
		m.windowTitle(),
	)
}

// windowTitle names the terminal tab. With several terminals open, "which one
// is watching my ports" is otherwise guesswork.
func (m Model) windowTitle() tea.Cmd {
	title := "openport — " + m.route.String()
	if m.watching {
		title = "● " + title
	}
	return tea.SetWindowTitle(title)
}

// startBusy records an operation in flight and returns the command that keeps
// the spinner animating. Only the transition from idle starts a new tick loop;
// overlapping work shares it.
func (m *Model) startBusy(label string) tea.Cmd {
	m.busy++
	m.busyLabel = label
	if m.busy == 1 {
		return m.spinner.Tick
	}
	return nil
}

// endBusy marks one operation finished.
func (m *Model) endBusy() {
	if m.busy > 0 {
		m.busy--
	}
	if m.busy == 0 {
		m.busyLabel = ""
	}
}

// --- Update --------------------------------------------------------------

// Update is the single place model state changes.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch t := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(t)

	case tea.MouseMsg:
		return m.handleMouse(t)

	case spinner.TickMsg:
		// Only keep the animation running while there is something to
		// animate; forwarding the tick is what schedules the next one.
		if m.busy == 0 {
			return m, nil
		}
		s, cmd := m.spinner.Update(t)
		m.spinner = s
		return m, cmd

	case tea.ResumeMsg:
		// Time passed while suspended, so whatever is on screen is stale.
		return m.refresh()
	}

	return m.handleMessage(message)
}

// handleMouse translates wheel scrolling into cursor motion. The program asks
// for mouse events, so it owes the user something in return for taking them —
// an enabled mouse that does nothing just costs them their terminal's own
// selection behaviour.
func (m Model) handleMouse(mouse tea.MouseMsg) (tea.Model, tea.Cmd) {
	var key tea.KeyMsg
	switch mouse.Button {
	case tea.MouseButtonWheelUp:
		key = tea.KeyMsg{Type: tea.KeyUp}
	case tea.MouseButtonWheelDown:
		key = tea.KeyMsg{Type: tea.KeyDown}
	default:
		return m, nil
	}

	// Scroll whatever is on top, matching where the keyboard would go.
	if m.modal != nil {
		modal, cmd := m.modal.Update(key)
		m.modal = modal
		return m, cmd
	}
	if m.palette.IsOpen() {
		palette, cmd := m.palette.Update(key)
		m.palette = palette
		return m, cmd
	}
	return m.forwardToActive(key)
}

// handleKey implements the input-routing order: a modal eats keys first, then
// the palette, then the active view, and only then the global shortcuts. Two
// overlays are never open at once, so nothing can capture input ambiguously.
func (m Model) handleKey(keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ctrl+c is the one binding nothing may swallow.
	if keyMsg.Type == tea.KeyCtrlC {
		return m.shutdown()
	}

	if m.modal != nil {
		// Esc closes the top overlay, whatever it contains, before the content
		// gets a look. Content cannot opt out of this: a popup with no exit
		// strands the user with nothing to try but killing the process, and
		// relying on each widget to implement its own escape is exactly how
		// that happens — huh, for one, binds abort to ctrl+c and nothing else.
		if keyMsg.Type == tea.KeyEsc {
			m.modal = nil
			return m, nil
		}
		modal, cmd := m.modal.Update(keyMsg)
		m.modal = modal
		return m, cmd
	}

	if m.palette.IsOpen() {
		switch keyMsg.String() {
		case "esc":
			m.palette.Close()
			return m, nil
		case "enter":
			item, selectable := m.palette.Current()
			m.palette.Close()
			if !selectable {
				if item.Badge == "PRO" {
					return m.withToast(msg.Warn(item.Name + " requires OpenPort Pro"))
				}
				return m, nil
			}
			return m.runAction(item.ID)
		}
		palette, cmd := m.palette.Update(keyMsg)
		m.palette = palette
		return m, cmd
	}

	// While a view is capturing text (the ports filter, say), it gets every
	// key — otherwise typing "q" in a filter would quit the app.
	if capturer, ok := m.views[m.route].(interface{ CapturingInput() bool }); ok && capturer.CapturingInput() {
		return m.forwardToActive(keyMsg)
	}

	// Global bindings that no view claims.
	switch {
	case key.Matches(keyMsg, m.keys.Quit):
		return m.shutdown()

	case key.Matches(keyMsg, m.keys.Help):
		return m.openHelp()

	case key.Matches(keyMsg, m.keys.Palette):
		return m, m.palette.Open()

	case key.Matches(keyMsg, m.keys.Menu):
		return m.navigate(msg.RouteActions)

	case key.Matches(keyMsg, m.keys.Refresh):
		return m.refresh()

	case key.Matches(keyMsg, m.keys.Debug):
		return m, dumpStateCmd(m.debugDump())

	case key.Matches(keyMsg, m.keys.Suspend):
		// Hand the terminal back to the shell; ResumeMsg refreshes on return.
		return m, tea.Suspend

	case key.Matches(keyMsg, m.keys.GotoPorts):
		return m.navigate(msg.RoutePorts)
	case key.Matches(keyMsg, m.keys.GotoActions):
		return m.navigate(msg.RouteActions)
	case key.Matches(keyMsg, m.keys.GotoAliases):
		return m.navigate(msg.RouteAliases)
	case key.Matches(keyMsg, m.keys.GotoBlueprints):
		return m.navigate(msg.RouteBlueprints)
	case key.Matches(keyMsg, m.keys.GotoConfig):
		return m.navigate(msg.RouteConfig)
	case key.Matches(keyMsg, m.keys.GotoHistory):
		return m.navigate(msg.RouteHistory)
	case key.Matches(keyMsg, m.keys.GotoWorkspace):
		return m.navigate(msg.RouteWorkspace)

	case key.Matches(keyMsg, m.keys.Back):
		// Esc dismisses a sticky toast before it means anything else.
		if len(m.toasts) > 0 {
			m.toasts = nil
			return m, nil
		}
		if m.route != msg.RoutePorts {
			return m.navigate(msg.RoutePorts)
		}
	}

	return m.forwardToActive(keyMsg)
}

// handleMessage routes every non-key message: intents become commands, results
// become state plus a toast.
func (m Model) handleMessage(message tea.Msg) (tea.Model, tea.Cmd) {
	switch t := message.(type) {

	// --- overlay plumbing ---

	case msg.CloseOverlayMsg:
		m.modal = nil
		m.palette.Close()
		return m, nil

	case msg.OpenPaletteMsg:
		return m, m.palette.Open()

	case msg.OpenHelpMsg:
		return m.openHelp()

	case msg.ToastMsg:
		return m.withToast(t)

	case msg.ToastExpiredMsg:
		filtered := m.toasts[:0]
		for _, toast := range m.toasts {
			if toast.ID != t.ID {
				filtered = append(filtered, toast)
			}
		}
		m.toasts = filtered
		return m, nil

	case msg.FormResultMsg:
		m.modal = nil
		return m.handleFormResult(t)

	case msg.ConfirmResultMsg:
		m.modal = nil
		return m.handleConfirmResult(t)

	// --- navigation & dispatch ---

	case msg.NavigateMsg:
		return m.navigate(t.To)

	case msg.RunActionMsg:
		return m.runAction(t.ID)

	// --- port intents ---

	case msg.ListPortsMsg:
		return m, tea.Batch(m.startBusy("reading ports"), listPortsCmd(m.svc))

	case msg.CheckPortMsg:
		return m, checkPortCmd(m.svc, t.Port)

	case msg.KillPortMsg:
		return m.confirmKillPort(t)

	case msg.KillPIDMsg:
		return m.confirmKillPID(t)

	case msg.KillRangeMsg:
		return m.confirmKillRange(t)

	case msg.SuggestPortMsg:
		// This one can scan hundreds of ports, binding each to check it is
		// really free, so it is the most important to acknowledge.
		return m, tea.Batch(m.startBusy("finding a free port"),
			suggestCmd(m.svc, t.Near, t.Count))

	case msg.ReservePortMsg:
		return m, reserveCmd(m.svc, t.Port, t.Service)

	case msg.HealthCheckMsg:
		return m, tea.Batch(m.startBusy("checking health"),
			healthCmd(m.svc, t.Port, t.Kind))

	case msg.ToggleProtectedMsg:
		return m, toggleProtectedCmd(m.store, t.Port)

	case msg.ToggleWatchMsg:
		return m.toggleWatch()

	case msg.GroupActionMsg:
		return m.confirmGroupAction(t)

	case msg.ApplyBlueprintMsg:
		return m, tea.Batch(m.startBusy("applying "+t.Name),
			applyBlueprintCmd(m.svc, m.store, t.Name))

	// --- store intents ---

	case msg.SaveAliasMsg:
		return m, saveAliasCmd(m.store, t.Alias)

	case msg.DeleteAliasMsg:
		return m.confirmDelete("alias", t.Name)

	case msg.SaveGroupMsg:
		return m, saveGroupCmd(m.store, t.Group)

	case msg.DeleteGroupMsg:
		return m.confirmDelete("group", t.Name)

	case msg.SaveBlueprintMsg:
		return m, saveBlueprintCmd(m.store, t.Blueprint)

	case msg.DeleteBlueprintMsg:
		return m.confirmDelete("blueprint", t.Name)

	case msg.SaveProjectConfigMsg:
		return m, saveProjectConfigCmd(m.store, t.Dir, t.Cfg)

	case msg.LoadDataMsg:
		return m, loadDataCmd(m.store, m.workDir)

	case msg.AutoRefreshMsg:
		return m.onAutoRefresh()

	// --- Pro intents ---

	case msg.LoginMsg:
		return m, loginCmd(m.provider, t.Provider)

	case msg.SelectWorkspaceMsg:
		return m, selectWorkspaceCmd(m.provider, t.ID)

	case msg.SyncPushMsg:
		return m, pushCmd(m.provider, m.store)

	case msg.SyncPullMsg:
		return m.confirmPull()

	case msg.LoadRegistryMsg:
		return m, registryCmd(m.provider)

	case msg.DetectDriftMsg:
		return m, driftCmd(m.provider, m.store)

	// --- results ---

	case msg.PortsLoadedMsg:
		return m.onPortsLoaded(t)

	case msg.WatchTickMsg:
		return m.onWatchTick(t)

	case msg.WatchStoppedMsg:
		m.watching = false
		m.watchCh = nil
		m.portsView.SetWatching(false)
		return m.broadcast(t)

	case msg.PortCheckedMsg:
		return m.onPortChecked(t)

	case msg.KilledMsg:
		return m.onKilled(t)

	case msg.SuggestedMsg:
		return m.onSuggested(t)

	case msg.ReservedMsg:
		return m.onReserved(t)

	case msg.HealthResultMsg:
		return m.onHealthResult(t)

	case msg.DataLoadedMsg:
		return m.onDataLoaded(t)

	case msg.SavedMsg:
		return m.onSaved(t)

	case msg.BlueprintAppliedMsg:
		return m.onBlueprintApplied(t)

	case msg.SyncedMsg:
		return m.onSynced(t)

	case msg.LoggedInMsg:
		return m.onLoggedIn(t)

	case msg.DriftMsg, msg.RegistryMsg, msg.WorkspacesMsg:
		return m.onProResult(message)

	case msg.ErrorMsg:
		cmd := m.broadcastCmd(message)
		model, toastCmd := m.withToast(msg.Error(t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}

	// Anything else — edit requests from views, spinner ticks — is broadcast so
	// whichever component cares can react.
	if alias, ok := views.EditAliasRequest(message); ok {
		return m.openModal(aliasForm(alias))
	}
	if group, ok := views.EditGroupRequest(message); ok {
		return m.openModal(groupForm(group))
	}
	return m.broadcastAll(message)
}

// broadcastAll fans an unrecognised message out to the overlays as well as the
// views.
//
// The Charm widgets do real work on commands, not just on keys: huh answers a
// confirm by emitting NextField, and bubbles/list computes its filter matches
// asynchronously and returns them as a message. Routing only keystrokes to the
// overlays leaves both half-wired — the confirm never resolves and the palette
// never filters — so whatever an overlay is on screen, it hears everything.
func (m Model) broadcastAll(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 3)

	if m.modal != nil {
		modal, cmd := m.modal.Update(message)
		m.modal = modal
		cmds = append(cmds, cmd)
	}
	if m.palette.IsOpen() {
		palette, cmd := m.palette.Update(message)
		m.palette = palette
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, m.broadcastCmd(message))
	return m, tea.Batch(cmds...)
}

// --- result handlers -----------------------------------------------------

func (m Model) onPortsLoaded(t msg.PortsLoadedMsg) (tea.Model, tea.Cmd) {
	// The ambient poll never touched the busy counter, so it must not clear it
	// either — otherwise a background refresh landing mid-probe would drop the
	// spinner while the probe is still running.
	if !t.Background {
		m.endBusy()
	}
	if t.Err == nil {
		m.lastPorts = t.Ports
	}
	cmd := m.broadcastCmd(t)
	// A background poll that fails should not raise a toast every few seconds;
	// the ports view surfaces the error in place. Only a listing the user
	// explicitly asked for gets the notification.
	if t.Err != nil && !t.Background {
		model, toastCmd := m.withToast(msg.Error("Could not list ports: " + t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}
	return m, cmd
}

// onAutoRefresh is the ambient background poll. It keeps the ports list and the
// history current without the user pressing anything, and re-arms itself.
//
// It is silent — no spinner — because a refresh the user did not request should
// not look like work they are waiting on. Store data is always reloaded (that
// is what closes the "history updates only after restart" gap); ports are
// reloaded only when watch mode is not already doing it faster.
func (m Model) onAutoRefresh() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{
		autoRefreshCmd(),
		loadDataCmd(m.store, m.workDir),
	}
	if !m.watching {
		cmds = append(cmds, backgroundListCmd(m.svc))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) onWatchTick(t msg.WatchTickMsg) (tea.Model, tea.Cmd) {
	m.lastPorts = t.Ports
	cmd := m.broadcastCmd(t)
	if !m.watching || m.watchCh == nil {
		return m, cmd
	}
	// Re-arm: reading the next value happens on a command goroutine, so the
	// interface stays responsive between ticks.
	return m, tea.Batch(cmd, watchCmd(m.watchCh))
}

func (m Model) onPortChecked(t msg.PortCheckedMsg) (tea.Model, tea.Cmd) {
	cmd := m.broadcastCmd(t)

	var notice msg.ToastMsg
	switch {
	case t.Err != nil:
		notice = msg.Error(t.Err.Error())
	case !t.Found:
		notice = msg.Info(fmt.Sprintf("Port %d is free", t.Port.Port))
	default:
		label := t.Port.Label()
		if label == "" {
			label = "an unidentified process"
		}
		notice = msg.Info(fmt.Sprintf("Port %d: %s (pid %d)", t.Port.Port, label, t.Port.PID))
	}

	model, toastCmd := m.withToast(notice)
	return model, tea.Batch(cmd, toastCmd)
}

// onKilled summarises a kill batch. Skipped results are reported as warnings
// rather than errors: the protected-port guard doing its job is not a failure.
func (m Model) onKilled(t msg.KilledMsg) (tea.Model, tea.Cmd) {
	if t.Err != nil {
		return m.withToast(msg.Error(t.Err.Error()))
	}

	var killed, skipped, failed int
	var firstReason, firstError string
	for _, r := range t.Results {
		switch {
		case r.Skipped:
			skipped++
			if firstReason == "" {
				firstReason = r.Reason
			}
		case r.Err != nil:
			failed++
			if firstError == "" {
				firstError = r.Err.Error()
			}
		default:
			killed++
		}
	}

	var notice msg.ToastMsg
	switch {
	case failed > 0:
		notice = msg.Error(fmt.Sprintf("%d failed: %s", failed, firstError))
	case killed == 0 && skipped > 0:
		notice = msg.Warn(pluralSkip(skipped, firstReason))
	case skipped > 0:
		notice = msg.Warn(fmt.Sprintf("Killed %d, skipped %d (%s)", killed, skipped, firstReason))
	case killed == 1:
		notice = msg.Success(fmt.Sprintf("Killed %s on port %d",
			processLabel(t.Results[0].Process), t.Results[0].Port))
	default:
		notice = msg.Success(fmt.Sprintf("Killed %d processes", killed))
	}

	model, toastCmd := m.withToast(notice)
	// The listing is stale the moment anything dies, and the kill just wrote a
	// history entry — reload both so History reflects it without waiting for
	// the ambient tick. Ports refresh only when watch mode is not already on it.
	refresh := []tea.Cmd{toastCmd, loadDataCmd(m.store, m.workDir)}
	if !m.watching {
		refresh = append(refresh, listPortsCmd(m.svc))
	}
	return model, tea.Batch(refresh...)
}

func pluralSkip(n int, reason string) string {
	if n == 1 {
		if reason == "" {
			return "Nothing killed"
		}
		return "Skipped: " + reason
	}
	return fmt.Sprintf("Skipped %d ports (%s)", n, reason)
}

func processLabel(p string) string {
	if p == "" {
		return "process"
	}
	return p
}

func (m Model) onSuggested(t msg.SuggestedMsg) (tea.Model, tea.Cmd) {
	m.endBusy()
	if t.Err != nil {
		return m.withToast(msg.Error(t.Err.Error()))
	}
	parts := make([]string, 0, len(t.Ports))
	for _, p := range t.Ports {
		parts = append(parts, strconv.Itoa(p))
	}
	return m.withToast(msg.Success("Free: " + strings.Join(parts, ", ")))
}

func (m Model) onReserved(t msg.ReservedMsg) (tea.Model, tea.Cmd) {
	if t.Err != nil {
		return m.withToast(msg.Error(t.Err.Error()))
	}
	model, cmd := m.withToast(msg.Success(
		fmt.Sprintf("Reserved %d for %s", t.Port, t.Service)))
	return model, tea.Batch(cmd, loadDataCmd(m.store, m.workDir), listPortsCmd(m.svc))
}

func (m Model) onHealthResult(t msg.HealthResultMsg) (tea.Model, tea.Cmd) {
	m.endBusy()
	cmd := m.broadcastCmd(t)
	if t.Err != nil {
		model, toastCmd := m.withToast(msg.Error(t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}

	notice := msg.Warn(fmt.Sprintf("Port %d: %s", t.Port, t.Result.Status))
	if t.Result.OK {
		notice = msg.Success(fmt.Sprintf("Port %d: %s (%dms)",
			t.Port, t.Result.Status, t.Result.LatencyMS))
	}
	model, toastCmd := m.withToast(notice)
	return model, tea.Batch(cmd, toastCmd)
}

func (m Model) onDataLoaded(t msg.DataLoadedMsg) (tea.Model, tea.Cmd) {
	m.groups = t.Groups
	m.blueprints = t.Blueprints
	m.project, m.hasProject = t.Project, t.HasProject

	cmd := m.broadcastCmd(t)
	if t.Err != nil {
		model, toastCmd := m.withToast(msg.Error(t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}
	return m, cmd
}

func (m Model) onSaved(t msg.SavedMsg) (tea.Model, tea.Cmd) {
	if t.Err != nil {
		return m.withToast(msg.Error(t.Err.Error()))
	}
	model, cmd := m.withToast(msg.Success("Saved " + t.What))
	// Aliases and protection change how ports render, so reload both.
	return model, tea.Batch(cmd, loadDataCmd(m.store, m.workDir), listPortsCmd(m.svc))
}

func (m Model) onBlueprintApplied(t msg.BlueprintAppliedMsg) (tea.Model, tea.Cmd) {
	m.endBusy()
	cmd := m.broadcastCmd(t)
	if t.Err != nil {
		model, toastCmd := m.withToast(msg.Error(t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}

	var ok, conflict int
	for _, r := range t.Results {
		switch r.Status {
		case "ok":
			ok++
		case "conflict":
			conflict++
		}
	}
	notice := msg.Success(fmt.Sprintf("%s: %d of %d services up", t.Name, ok, len(t.Results)))
	if conflict > 0 {
		notice = msg.Warn(fmt.Sprintf("%s: %d conflict(s) — see the detail pane",
			t.Name, conflict))
	}
	model, toastCmd := m.withToast(notice)
	// Applying a blueprint wrote a history entry; reload so History shows it.
	return model, tea.Batch(cmd, toastCmd, loadDataCmd(m.store, m.workDir))
}

func (m Model) onSynced(t msg.SyncedMsg) (tea.Model, tea.Cmd) {
	if t.Err != nil {
		return m.withToast(msg.Error(t.Direction + " failed: " + t.Err.Error()))
	}
	model, cmd := m.withToast(msg.Success("Sync " + t.Direction + " complete"))
	if t.Direction == "pull" {
		return model, tea.Batch(cmd, loadDataCmd(m.store, m.workDir), listPortsCmd(m.svc))
	}
	return model, cmd
}

func (m Model) onLoggedIn(t msg.LoggedInMsg) (tea.Model, tea.Cmd) {
	cmd := m.broadcastCmd(t)
	if t.Err != nil {
		model, toastCmd := m.withToast(msg.Error(t.Err.Error()))
		return model, tea.Batch(cmd, toastCmd)
	}
	model, toastCmd := m.withToast(msg.Success("Signed in as " + t.Session.Email))
	return model, tea.Batch(cmd, toastCmd, workspacesCmd(m.provider))
}

// onProResult forwards a Pro result and surfaces the "needs Pro" case as a
// hint rather than a hard error.
func (m Model) onProResult(message tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.broadcastCmd(message)

	var err error
	switch t := message.(type) {
	case msg.DriftMsg:
		err = t.Err
	case msg.RegistryMsg:
		err = t.Err
	case msg.WorkspacesMsg:
		err = t.Err
	}
	if err == nil {
		return m, cmd
	}

	notice := msg.Error(err.Error())
	if err == opsync.ErrProRequired {
		notice = msg.Warn("This feature requires OpenPort Pro")
	}
	model, toastCmd := m.withToast(notice)
	return model, tea.Batch(cmd, toastCmd)
}

// --- confirmations -------------------------------------------------------

// Confirmation IDs.
const (
	confirmKill          = "confirm.kill"
	confirmKillProtected = "confirm.kill.protected"
	confirmKillPID       = "confirm.kill_pid"
	confirmKillRange     = "confirm.kill_range"
	confirmGroupKill     = "confirm.group_kill"
	confirmDeleteItem    = "confirm.delete"
	confirmPullRemote    = "confirm.pull"
)

// Payload types round-tripped through a confirmation.
type killPayload struct {
	port     int
	force    bool
	override bool
	label    string
}

type killPIDPayload struct {
	pid   int
	force bool
}

type killRangePayload struct {
	lo, hi   int
	force    bool
	override bool
}

type groupPayload struct {
	name  string
	force bool
}

type deletePayload struct {
	kind string
	name string
}

// confirmKillPort opens the confirmation gate in front of a kill. A protected
// port gets an extra gate before that one: the first asks whether to override
// the protection at all, the second asks whether to really do it.
func (m Model) confirmKillPort(t msg.KillPortMsg) (tea.Model, tea.Cmd) {
	target, known := m.findPort(t.Port)

	label := "whatever is on this port"
	if known {
		label = describePort(target)
	}
	signal := "SIGTERM"
	if t.Force {
		signal = "SIGKILL"
	}

	if known && target.Protected && !t.Override {
		body := fmt.Sprintf(
			"Port %d is on the never-kill list.\n\n%s\n\nOverride the protection?",
			t.Port, target.Reason)
		return m.openModal(components.NewConfirm(
			m.theme, confirmKillProtected, "Protected port", body,
			"Override", "Keep it safe",
			killPayload{port: t.Port, force: t.Force, override: true, label: label},
			true,
		))
	}

	body := fmt.Sprintf("Send %s to %s on port %d?", signal, label, t.Port)
	if t.Override {
		body = fmt.Sprintf("This port is protected.\n\nReally send %s to %s on port %d?",
			signal, label, t.Port)
	}
	return m.openModal(components.NewConfirm(
		m.theme, confirmKill, "Confirm kill", body,
		"Kill", "Cancel",
		killPayload{port: t.Port, force: t.Force, override: t.Override, label: label},
		true,
	))
}

func (m Model) confirmKillPID(t msg.KillPIDMsg) (tea.Model, tea.Cmd) {
	signal := "SIGTERM"
	if t.Force {
		signal = "SIGKILL"
	}
	body := fmt.Sprintf("Send %s to process %d?", signal, t.PID)
	return m.openModal(components.NewConfirm(
		m.theme, confirmKillPID, "Confirm kill", body,
		"Kill", "Cancel",
		killPIDPayload{pid: t.PID, force: t.Force}, true,
	))
}

// confirmKillRange counts what would actually die before asking, so the user
// confirms a number rather than a range.
func (m Model) confirmKillRange(t msg.KillRangeMsg) (tea.Model, tea.Cmd) {
	lo, hi := t.Lo, t.Hi
	if lo > hi {
		lo, hi = hi, lo
	}

	var affected, protected int
	for _, p := range m.lastPorts {
		if p.Port < lo || p.Port > hi {
			continue
		}
		if p.Protected {
			protected++
			continue
		}
		affected++
	}

	signal := "SIGTERM"
	if t.Force {
		signal = "SIGKILL"
	}
	body := fmt.Sprintf("Send %s to %d listener(s) between %d and %d?",
		signal, affected, lo, hi)
	if protected > 0 {
		body += fmt.Sprintf("\n\n%d protected port(s) in this range will be skipped.", protected)
	}
	if affected == 0 {
		body = fmt.Sprintf("Nothing unprotected is listening between %d and %d.", lo, hi)
	}

	return m.openModal(components.NewConfirm(
		m.theme, confirmKillRange, "Confirm kill range", body,
		"Kill all", "Cancel",
		killRangePayload{lo: lo, hi: hi, force: t.Force, override: t.Override}, true,
	))
}

func (m Model) confirmGroupAction(t msg.GroupActionMsg) (tea.Model, tea.Cmd) {
	if t.Kind == msg.GroupHealth {
		// Probing is harmless; no gate needed.
		return m, groupHealthCmd(m.svc, m.store, t.Group)
	}

	count := 0
	for _, g := range m.groups {
		if g.Name == t.Group {
			count = len(g.Ports)
		}
	}
	body := fmt.Sprintf("Shut down every service in %q (%d ports)?", t.Group, count)
	return m.openModal(components.NewConfirm(
		m.theme, confirmGroupKill, "Confirm group shutdown", body,
		"Shut down", "Cancel",
		groupPayload{name: t.Group, force: t.Force}, true,
	))
}

func (m Model) confirmDelete(kind, name string) (tea.Model, tea.Cmd) {
	body := fmt.Sprintf("Delete %s %q?", kind, name)
	return m.openModal(components.NewConfirm(
		m.theme, confirmDeleteItem, "Confirm delete", body,
		"Delete", "Cancel",
		deletePayload{kind: kind, name: name}, true,
	))
}

func (m Model) confirmPull() (tea.Model, tea.Cmd) {
	body := "Pulling replaces your local aliases, groups, blueprints and " +
		"protected ports with the workspace snapshot.\n\nContinue?"
	return m.openModal(components.NewConfirm(
		m.theme, confirmPullRemote, "Confirm pull", body,
		"Replace local", "Cancel", nil, true,
	))
}

// handleConfirmResult runs the action a confirmation was guarding.
func (m Model) handleConfirmResult(t msg.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if !t.Confirmed {
		return m, nil
	}

	switch t.ID {
	case confirmKillProtected:
		// Overriding protection earns a second gate, never a direct kill.
		payload, _ := t.Payload.(killPayload)
		return m.confirmKillPort(msg.KillPortMsg{
			Port: payload.port, Force: payload.force, Override: true,
		})

	case confirmKill:
		payload, _ := t.Payload.(killPayload)
		return m, killPortCmd(m.svc, payload.port, ports.KillOptions{
			Force: payload.force, OverrideProtected: payload.override,
		})

	case confirmKillPID:
		payload, _ := t.Payload.(killPIDPayload)
		return m, killPIDCmd(m.svc, payload.pid, payload.force)

	case confirmKillRange:
		payload, _ := t.Payload.(killRangePayload)
		return m, killRangeCmd(m.svc, payload.lo, payload.hi, ports.KillOptions{
			Force: payload.force, OverrideProtected: payload.override,
		})

	case confirmGroupKill:
		payload, _ := t.Payload.(groupPayload)
		return m, groupShutdownCmd(m.svc, m.store, payload.name, payload.force)

	case confirmDeleteItem:
		payload, _ := t.Payload.(deletePayload)
		switch payload.kind {
		case "alias":
			return m, deleteAliasCmd(m.store, payload.name)
		case "group":
			return m, deleteGroupCmd(m.store, payload.name)
		case "blueprint":
			return m, deleteBlueprintCmd(m.store, payload.name)
		}

	case confirmPullRemote:
		return m, pullCmd(m.provider, m.store)
	}
	return m, nil
}

// --- form results --------------------------------------------------------

// handleFormResult converts collected input into an intent. Validation already
// ran inline, so parse failures here are defensive rather than expected.
func (m Model) handleFormResult(t msg.FormResultMsg) (tea.Model, tea.Cmd) {
	v := t.Values

	switch t.ID {
	case formCheck:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, checkPortCmd(m.svc, port)

	case formKill:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m.confirmKillPort(msg.KillPortMsg{Port: port, Force: v["force"] == "true"})

	case formKillPID:
		pid, err := intField(v, "pid")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m.confirmKillPID(msg.KillPIDMsg{PID: pid, Force: v["force"] == "true"})

	case formKillRange:
		lo, err := intField(v, "lo")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		hi, err := intField(v, "hi")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m.confirmKillRange(msg.KillRangeMsg{Lo: lo, Hi: hi, Force: v["force"] == "true"})

	case formSuggest:
		near, _ := intField(v, "near")
		count, _ := intField(v, "count")
		if count == 0 {
			count = 3
		}
		return m, suggestCmd(m.svc, near, count)

	case formReserve:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, reserveCmd(m.svc, port, strings.TrimSpace(v["service"]))

	case formHealthTCP, formHealthHTTP:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		kind := ports.HealthTCP
		if t.ID == formHealthHTTP {
			kind = ports.HealthHTTP
		}
		return m, healthCmd(m.svc, port, kind)

	case formProtect:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, toggleProtectedCmd(m.store, port)

	case formAlias:
		port, err := intField(v, "port")
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, saveAliasCmd(m.store, model.Alias{
			Name: strings.TrimSpace(v["name"]),
			Port: port,
			Note: strings.TrimSpace(v["note"]),
		})

	case formGroup:
		list, err := parsePortList(v["ports"])
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, saveGroupCmd(m.store, model.Group{
			Name:  strings.TrimSpace(v["name"]),
			Ports: list,
			Note:  strings.TrimSpace(v["note"]),
		})

	case formBlueprint:
		svcs, err := parseServiceList(v["services"])
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		return m, saveBlueprintCmd(m.store, model.Blueprint{
			Name:        strings.TrimSpace(v["name"]),
			Description: strings.TrimSpace(v["description"]),
			Services:    svcs,
		})

	case formBlueprintApply:
		return m, applyBlueprintCmd(m.svc, m.store, v["choice"])

	case formGroupShutdown:
		return m.confirmGroupAction(msg.GroupActionMsg{
			Group: v["choice"], Kind: msg.GroupShutdown,
		})

	case formGroupHealth:
		return m, groupHealthCmd(m.svc, m.store, v["choice"])

	case formProjectConfig:
		cfg, err := m.projectConfigFromForm(v)
		if err != nil {
			return m.withToast(msg.Error(err.Error()))
		}
		dir := m.workDir
		if m.hasProject && m.project.Dir != "" {
			dir = m.project.Dir
		}
		return m, saveProjectConfigCmd(m.store, dir, cfg)

	case formLogin:
		provider := opsync.OAuthGitHub
		if v["choice"] == "Google" {
			provider = opsync.OAuthGoogle
		}
		return m, loginCmd(m.provider, provider)

	case formWorkspace:
		return m, selectWorkspaceCmd(m.provider, v["choice"])
	}

	return m, nil
}

func (m Model) projectConfigFromForm(v map[string]string) (model.ProjectConfig, error) {
	cfg := model.ProjectConfig{
		Name:      strings.TrimSpace(v["name"]),
		Blueprint: strings.TrimSpace(v["blueprint"]),
	}
	if raw := strings.TrimSpace(v["expected"]); raw != "" {
		list, err := parsePortList(raw)
		if err != nil {
			return cfg, err
		}
		cfg.ExpectedPorts = list
	}
	if raw := strings.TrimSpace(v["aliases"]); raw != "" {
		aliases, err := parseAliasList(raw)
		if err != nil {
			return cfg, err
		}
		cfg.Aliases = aliases
	}
	return cfg, nil
}

func intField(values map[string]string, key string) (int, error) {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", raw)
	}
	return n, nil
}

// --- action dispatch -----------------------------------------------------

// runAction is the single dispatcher both the palette and the actions menu
// funnel through, so a capability behaves identically whichever door it was
// entered by.
func (m Model) runAction(id string) (tea.Model, tea.Cmd) {
	if action, ok := actions.Find(id); ok && action.Pro && !m.provider.IsPro() {
		return m.withToast(msg.Warn(action.Title + " requires OpenPort Pro"))
	}

	switch id {
	// Ports
	case actions.PortsList:
		model, cmd := m.navigate(msg.RoutePorts)
		return model, tea.Batch(cmd, listPortsCmd(m.svc))
	case actions.PortsCheck:
		return m.openModal(portForm(formCheck, "Check a port", "Port", m.selectedPortValue()))
	case actions.PortsKill:
		return m.openModal(killForm())
	case actions.PortsKillPID:
		return m.openModal(killPIDForm())
	case actions.PortsKillRange:
		return m.openModal(killRangeForm())
	case actions.PortsWatch:
		return m.toggleWatch()
	case actions.PortsSuggest:
		return m.openModal(suggestForm())
	case actions.PortsReserve:
		return m.openModal(reserveForm())
	case actions.PortsHealthTCP:
		return m.openModal(portForm(formHealthTCP, "TCP health check", "Port", m.selectedPortValue()))
	case actions.PortsHealthHTTP:
		return m.openModal(portForm(formHealthHTTP, "HTTP health check", "Port", m.selectedPortValue()))

	// Aliases & groups
	case actions.AliasManage:
		m.aliasesView.ShowAliases()
		return m.navigate(msg.RouteAliases)
	case actions.AliasNew:
		return m.openModal(aliasForm(model.Alias{Port: m.selectedPort()}))
	case actions.GroupManage:
		m.aliasesView.ShowGroups()
		return m.navigate(msg.RouteAliases)
	case actions.GroupNew:
		return m.openModal(groupForm(model.Group{}))
	case actions.GroupShutdown:
		return m.openGroupPicker(formGroupShutdown, "Shut down a group")
	case actions.GroupHealth:
		return m.openGroupPicker(formGroupHealth, "Health check a group")

	// Blueprints
	case actions.BlueprintList:
		return m.navigate(msg.RouteBlueprints)
	case actions.BlueprintNew:
		return m.openModal(blueprintForm())
	case actions.BlueprintApply:
		return m.openBlueprintPicker()

	// Config
	case actions.ConfigView:
		m.configView.ShowProject()
		if m.hasProject {
			return m.openModal(projectConfigForm(m.project, m.workDir))
		}
		return m.navigate(msg.RouteConfig)
	case actions.ConfigInit:
		return m.openModal(projectConfigForm(model.ProjectConfig{}, m.workDir))

	// History & safety
	case actions.HistoryView:
		m.historyView.ShowHistory()
		return m.navigate(msg.RouteHistory)
	case actions.LogsView:
		m.historyView.ShowLogs()
		return m.navigate(msg.RouteHistory)
	case actions.ProtectedList:
		m.configView.ShowProtected()
		return m.navigate(msg.RouteConfig)
	case actions.ProtectedToggle:
		return m.openModal(portForm(formProtect, "Protect or unprotect", "Port", m.selectedPortValue()))

	// Interface
	case actions.UIMenu:
		return m.navigate(msg.RouteActions)
	case actions.UIRefresh:
		return m.refresh()
	case actions.UIHelp:
		return m.openHelp()
	case actions.UIDebug:
		return m, dumpStateCmd(m.debugDump())
	case actions.UIQuit:
		return m.shutdown()

	// Pro
	case actions.ProLogin:
		return m.openModal(pickerForm(formLogin, "Sign in", "Provider",
			[]string{"GitHub", "Google"}))
	case actions.ProWorkspaces:
		model, cmd := m.navigate(msg.RouteWorkspace)
		return model, tea.Batch(cmd, workspacesCmd(m.provider))
	case actions.ProPush:
		return m, pushCmd(m.provider, m.store)
	case actions.ProPull:
		return m.confirmPull()
	case actions.ProRegistry:
		model, cmd := m.navigate(msg.RouteWorkspace)
		return model, tea.Batch(cmd, registryCmd(m.provider))
	case actions.ProDrift:
		return m, driftCmd(m.provider, m.store)
	}

	return m.withToast(msg.Warn("Unknown action: " + id))
}

func (m Model) openGroupPicker(formID, title string) (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 {
		return m.withToast(msg.Warn("No groups defined yet — press 3 then n to make one"))
	}
	names := make([]string, 0, len(m.groups))
	for _, g := range m.groups {
		names = append(names, g.Name)
	}
	sort.Strings(names)
	return m.openModal(pickerForm(formID, title, "Group", names))
}

func (m Model) openBlueprintPicker() (tea.Model, tea.Cmd) {
	if len(m.blueprints) == 0 {
		return m.withToast(msg.Warn("No blueprints defined yet — press 4 then n to make one"))
	}
	names := make([]string, 0, len(m.blueprints))
	for _, bp := range m.blueprints {
		names = append(names, bp.Name)
	}
	sort.Strings(names)
	return m.openModal(pickerForm(formBlueprintApply, "Apply a blueprint", "Blueprint", names))
}

// --- watch mode ----------------------------------------------------------

// toggleWatch starts or stops the live refresh subscription.
func (m Model) toggleWatch() (tea.Model, tea.Cmd) {
	if m.watching {
		if m.watchCancel != nil {
			m.watchCancel()
			m.watchCancel = nil
		}
		m.watching = false
		m.watchCh = nil
		m.portsView.SetWatching(false)
		model, cmd := m.withToast(msg.Info("Watch mode off"))
		return model, tea.Batch(cmd, m.windowTitle())
	}

	// Background context: the returned cancel func is what tears the poller
	// down, on quit or on toggling watch off.
	ch, cancel := m.svc.Watch(context.Background(), ports.DefaultWatchInterval)
	m.watchCh = ch
	m.watchCancel = cancel
	m.watching = true
	m.portsView.SetWatching(true)

	model, toastCmd := m.withToast(msg.Info("Watch mode on"))
	return model, tea.Batch(toastCmd, watchCmd(ch), m.windowTitle())
}

// --- helpers -------------------------------------------------------------

func (m Model) navigate(to msg.Route) (tea.Model, tea.Cmd) {
	if _, ok := m.views[to]; !ok {
		return m, nil
	}
	m.route = to

	// Give the destination fresh state immediately rather than making the user
	// wait for the next ambient tick. This is what the History view needed
	// most: it was only ever loaded at startup, so it showed the world as it
	// was the last time the app launched.
	cmds := []tea.Cmd{m.windowTitle()}
	if isDataBacked(to) {
		cmds = append(cmds, loadDataCmd(m.store, m.workDir))
	}
	if to == msg.RoutePorts {
		cmds = append(cmds, backgroundListCmd(m.svc))
	}
	return m, tea.Batch(cmds...)
}

// isDataBacked reports whether a route renders store-backed collections, and so
// benefits from a reload when navigated to.
func isDataBacked(r msg.Route) bool {
	switch r {
	case msg.RouteAliases, msg.RouteBlueprints, msg.RouteConfig, msg.RouteHistory:
		return true
	default:
		return false
	}
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	return m, tea.Batch(listPortsCmd(m.svc), loadDataCmd(m.store, m.workDir))
}

// shutdown stops the watcher before quitting so the poller goroutine does not
// outlive the program.
func (m Model) shutdown() (tea.Model, tea.Cmd) {
	if m.watchCancel != nil {
		m.watchCancel()
		m.watchCancel = nil
	}
	m.quit = true
	return m, tea.Quit
}

func (m Model) openModal(content components.ModalContent) (tea.Model, tea.Cmd) {
	m.palette.Close()
	m.modal = components.NewModal(m.theme, content)
	// huh-backed content needs Init to focus its first field.
	switch c := content.(type) {
	case *components.Form:
		return m, c.Init()
	case *components.Confirm:
		return m, c.Init()
	}
	return m, nil
}

func (m Model) openHelp() (tea.Model, tea.Cmd) {
	sections := []components.HelpSection{
		{Title: "Global", Bindings: flatten(m.keys.FullHelp())},
	}
	if active, ok := m.views[m.route]; ok {
		sections = append(sections, components.HelpSection{
			Title:    active.Title(),
			Bindings: active.Help(),
		})
	}
	return m.openModal(components.NewHelp(m.theme, sections))
}

func flatten(groups [][]key.Binding) []key.Binding {
	var out []key.Binding
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// withToast pushes a notification and schedules its expiry.
func (m Model) withToast(t msg.ToastMsg) (tea.Model, tea.Cmd) {
	m.nextToast++
	toast := components.Toast{ID: m.nextToast, Text: t.Text, Level: t.Level}

	m.toasts = append(m.toasts, toast)
	if len(m.toasts) > components.MaxToasts {
		m.toasts = m.toasts[len(m.toasts)-components.MaxToasts:]
	}
	return m, toast.ExpireCmd()
}

// broadcast sends a message to every view, so a screen the user is not looking
// at is still up to date when they switch to it.
func (m Model) broadcast(message tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.broadcastCmd(message)
}

func (m Model) broadcastCmd(message tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.views))
	for route, v := range m.views {
		updated, cmd := v.Update(message)
		m.views[route] = updated
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m Model) forwardToActive(message tea.Msg) (tea.Model, tea.Cmd) {
	active, ok := m.views[m.route]
	if !ok {
		return m, nil
	}
	updated, cmd := active.Update(message)
	m.views[m.route] = updated
	return m, cmd
}

// findPort looks a port up in the last listing, preferring a row with a
// resolved PID.
func (m Model) findPort(port int) (ports.Port, bool) {
	var best ports.Port
	found := false
	for _, p := range m.lastPorts {
		if p.Port != port {
			continue
		}
		if !found || (best.PID == 0 && p.PID != 0) {
			best, found = p, true
		}
	}
	return best, found
}

func describePort(p ports.Port) string {
	label := p.Label()
	if label == "" {
		label = "an unidentified process"
	}
	if p.PID > 0 {
		return fmt.Sprintf("%s (pid %d)", label, p.PID)
	}
	return label
}

// selectedPort prefills forms from the ports table selection, so acting on the
// highlighted row does not mean retyping its number.
func (m Model) selectedPort() int {
	if p, ok := m.portsView.Selection(); ok {
		return p.Port
	}
	return 0
}

func (m Model) selectedPortValue() string {
	if port := m.selectedPort(); port > 0 {
		return strconv.Itoa(port)
	}
	return ""
}

// debugDump renders the model state for ctrl+d, covering the things that are
// not visible on screen.
func (m Model) debugDump() string {
	var b strings.Builder
	fmt.Fprintf(&b, "route=%s size=%dx%d\n", m.route, m.width, m.height)
	fmt.Fprintf(&b, "watching=%v watchCh=%v modal=%v palette=%v toasts=%d\n",
		m.watching, m.watchCh != nil, m.modal != nil, m.palette.IsOpen(), len(m.toasts))
	fmt.Fprintf(&b, "pro=%v workDir=%s hasProject=%v\n",
		m.provider.IsPro(), m.workDir, m.hasProject)
	fmt.Fprintf(&b, "ports=%d groups=%d blueprints=%d\n",
		len(m.lastPorts), len(m.groups), len(m.blueprints))

	if p, ok := m.portsView.Selection(); ok {
		fmt.Fprintf(&b, "selection=%s protected=%v\n", p.Key(), p.Protected)
	}
	for _, p := range m.lastPorts {
		fmt.Fprintf(&b, "  %5d %-6s pid=%-7d %-18s alias=%-12s protected=%v\n",
			p.Port, p.Protocol, p.PID, p.Process, p.Alias, p.Protected)
	}
	return b.String()
}

// --- View ----------------------------------------------------------------

// View renders the screen, then composites the overlays on top in z-order:
// modal above palette above toasts.
func (m Model) View() string {
	if m.quit {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return m.theme.Accent.Render(styles.GlyphLogo+" openport") +
			m.theme.Faint.Render("  starting…")
	}

	header := m.renderHeader()
	status := m.renderStatus()

	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(status)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := ""
	if active, ok := m.views[m.route]; ok {
		body = active.View(m.width, bodyHeight)
	}
	body = padCanvas(body, bodyHeight)

	screen := header + "\n" + body + "\n" + status

	// Toasts sit lowest in the overlay stack, above the status bar.
	if len(m.toasts) > 0 && m.height > 8 {
		block := components.RenderToasts(m.theme, m.toasts, m.width)
		if block != "" {
			blockWidth := lipgloss.Width(block)
			top := m.height - lipgloss.Height(block) - 2
			left := m.width - blockWidth - 2
			screen = composite(screen, block, top, left)
		}
	}

	if m.palette.IsOpen() {
		block := m.palette.View(m.width * 3 / 5)
		top, left := place(m.width, m.height, lipgloss.Width(block), lipgloss.Height(block))
		screen = composite(screen, block, top/3, left)
	}

	if m.modal != nil {
		block := m.modal.View(m.width, m.height)
		top, left := place(m.width, m.height, lipgloss.Width(block), lipgloss.Height(block))
		screen = composite(screen, block, top, left)
	}

	return clampFrame(screen, m.width, m.height)
}

// renderHeader draws the brand plus the route tabs, which double as the
// numeric shortcuts.
//
// It degrades in tiers rather than collapsing straight to the active view.
// The strip is the only place the other screens and their number keys are
// advertised, so dropping it entirely on a narrow terminal does not just look
// worse — it hides the navigation from anyone who has not memorised it.
func (m Model) renderHeader() string {
	for _, candidate := range []string{
		m.headerTabs(true, true),   // brand + full labels
		m.headerTabs(false, true),  // full labels
		m.headerTabs(false, false), // numbers, with the active one named
	} {
		if lipgloss.Width(candidate) <= m.width {
			return m.fillBar(candidate)
		}
	}
	// Nothing fits: at least say where we are.
	return m.fillBar(truncateLine(m.theme.TabActive.Render(m.route.String()), m.width))
}

// fillBar pads bar content out to the full terminal width with the chrome
// surface colour, so the top and bottom strips read as solid bars rather than
// styled text floating on the terminal's own background. Each segment already
// bakes that colour in (Lip Gloss will not repaint a background across nested
// reset codes), and this closes the gap on the right.
func (m Model) fillBar(content string) string {
	gap := m.width - lipgloss.Width(content)
	if gap <= 0 {
		return truncateLine(content, m.width)
	}
	return content + m.theme.BarFill.Render(strings.Repeat(" ", gap))
}

// headerTabs builds one tier of the strip.
func (m Model) headerTabs(brand, labels bool) string {
	tab := m.theme.Tab
	active := m.theme.TabActive
	if !labels {
		// Reclaim the per-tab padding when only numbers are shown.
		tab = tab.Padding(0, 1)
		active = active.Padding(0, 1)
	}

	parts := make([]string, 0, len(msg.AllRoutes)+1)
	if brand {
		parts = append(parts, m.theme.Brand.Render(styles.GlyphLogo+" openport"))
	}

	for i, route := range msg.AllRoutes {
		number := strconv.Itoa(i + 1)

		// The active tab keeps its name at every tier, so the strip always
		// answers "where am I" as well as "where can I go".
		label := number
		if labels || route == m.route {
			label = number + " " + route.String()
		}
		if labels && route == msg.RouteWorkspace && !m.provider.IsPro() {
			label += " ᴾᴿᴼ"
		}

		if route == m.route {
			parts = append(parts, active.Render(label))
			continue
		}
		parts = append(parts, tab.Render(label))
	}
	return strings.Join(parts, "")
}

// truncateLine clips a styled line to width without severing escape sequences.
func truncateLine(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

func (m Model) renderStatus() string {
	bar := components.StatusBar{
		Title:    m.route.String(),
		Watching: m.watching,
		Pro:      m.provider.IsPro(),
	}
	if m.busy > 0 {
		bar.Busy = m.spinner.View() + " " + m.busyLabel
	}
	if active, ok := m.views[m.route]; ok {
		bar.Title = active.Title()
		bar.Hints = active.Help()
		if s, ok := active.(interface{ SelectionSummary() string }); ok {
			bar.Selection = s.SelectionSummary()
		}
	}
	// On a short terminal there is no room for the floating toast stack, so
	// the newest notification takes over the hint half of the bar instead.
	if m.height <= 8 && len(m.toasts) > 0 {
		bar.Message = components.StatusToast(m.theme, m.toasts, m.width/2)
	}
	return bar.View(m.theme, m.width)
}
