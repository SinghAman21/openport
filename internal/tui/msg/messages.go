// Package msg declares every tea.Msg in the app, split into two families:
//
//	Intents  — "the user wants X". Emitted by views, palette and menu.
//	Results  — "X happened".       Emitted by the tea.Cmds in tui/commands.go.
//
// Views never call the service directly: they emit an intent, the root Update
// turns it into a command, and the command returns a result. That is what
// keeps state mutation confined to Update.
//
// This lives in its own leaf package rather than in tui/ because the root
// model imports views while views must emit these messages — declaring them
// here is what breaks that cycle.
package msg

import (
	"time"

	"github.com/SinghAman21/openport/internal/blueprint"
	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/ports"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// Route identifies a top-level screen.
type Route int

const (
	RoutePorts Route = iota
	RouteActions
	RouteAliases
	RouteBlueprints
	RouteConfig
	RouteHistory
	RouteWorkspace // Pro
)

// String renders the route name for the header and status bar.
func (r Route) String() string {
	switch r {
	case RouteActions:
		return "Actions"
	case RouteAliases:
		return "Aliases"
	case RouteBlueprints:
		return "Blueprints"
	case RouteConfig:
		return "Config"
	case RouteHistory:
		return "History"
	case RouteWorkspace:
		return "Workspace"
	default:
		return "Ports"
	}
}

// AllRoutes is the display order used by the header tabs.
var AllRoutes = []Route{
	RoutePorts, RouteActions, RouteAliases, RouteBlueprints,
	RouteConfig, RouteHistory, RouteWorkspace,
}

// --- Intents -------------------------------------------------------------

// NavigateMsg switches the active view.
type NavigateMsg struct{ To Route }

// OpenPaletteMsg opens the fuzzy command palette.
type OpenPaletteMsg struct{}

// OpenHelpMsg opens the keymap overlay.
type OpenHelpMsg struct{}

// RunActionMsg runs a registered action by ID. Both the palette and the
// actions menu emit this, so there is exactly one dispatch path per
// capability.
type RunActionMsg struct{ ID string }

// ListPortsMsg asks for a fresh port listing.
type ListPortsMsg struct{}

// CheckPortMsg inspects a single port.
type CheckPortMsg struct{ Port int }

// KillPortMsg terminates whatever holds a port.
type KillPortMsg struct {
	Port     int
	Force    bool
	Override bool // only ever set after a second confirmation
}

// KillPIDMsg terminates a process by PID.
type KillPIDMsg struct {
	PID   int
	Force bool
}

// KillRangeMsg terminates every listener in an inclusive range.
type KillRangeMsg struct {
	Lo, Hi   int
	Force    bool
	Override bool
}

// SuggestPortMsg asks for free ports near a hint.
type SuggestPortMsg struct{ Near, Count int }

// ReservePortMsg claims a port for a named service.
type ReservePortMsg struct {
	Port    int
	Service string
}

// HealthCheckMsg probes one port.
type HealthCheckMsg struct {
	Port int
	Kind ports.HealthKind
}

// GroupActionKind selects what a group action does.
type GroupActionKind int

const (
	// GroupShutdown kills every port in the group.
	GroupShutdown GroupActionKind = iota
	// GroupHealth health-checks every port in the group.
	GroupHealth
)

// GroupActionMsg runs an action across every port in a group.
type GroupActionMsg struct {
	Group string
	Kind  GroupActionKind
	Force bool
}

// ApplyBlueprintMsg walks a blueprint, resolving port conflicts.
type ApplyBlueprintMsg struct{ Name string }

// SaveAliasMsg persists an alias.
type SaveAliasMsg struct{ Alias model.Alias }

// DeleteAliasMsg removes an alias.
type DeleteAliasMsg struct{ Name string }

// SaveGroupMsg persists a group.
type SaveGroupMsg struct{ Group model.Group }

// DeleteGroupMsg removes a group.
type DeleteGroupMsg struct{ Name string }

// SaveBlueprintMsg persists a blueprint.
type SaveBlueprintMsg struct{ Blueprint model.Blueprint }

// DeleteBlueprintMsg removes a blueprint.
type DeleteBlueprintMsg struct{ Name string }

// ToggleProtectedMsg adds or removes a port from the never-kill list.
type ToggleProtectedMsg struct{ Port int }

// SaveProjectConfigMsg writes .openport.yml.
type SaveProjectConfigMsg struct {
	Dir string
	Cfg model.ProjectConfig
}

// ToggleWatchMsg starts or stops watch mode.
type ToggleWatchMsg struct{}

// LoadDataMsg asks for the store-backed collections the views render.
type LoadDataMsg struct{}

// DumpStateMsg writes the current model state to the debug log.
type DumpStateMsg struct{}

// --- Pro intents ---------------------------------------------------------

// LoginMsg starts the Pro OAuth flow.
type LoginMsg struct{ Provider opsync.OAuthProvider }

// SelectWorkspaceMsg picks the active Pro workspace.
type SelectWorkspaceMsg struct{ ID string }

// SyncPushMsg uploads the local snapshot.
type SyncPushMsg struct{}

// SyncPullMsg downloads the workspace snapshot.
type SyncPullMsg struct{}

// LoadRegistryMsg fetches the team registry.
type LoadRegistryMsg struct{}

// DetectDriftMsg compares local state against the expected environment.
type DetectDriftMsg struct{}

// --- Results -------------------------------------------------------------

// PortsLoadedMsg carries a fresh listing. Background marks a listing the
// ambient poll produced, which the root treats silently rather than as work
// the user is waiting on.
type PortsLoadedMsg struct {
	Ports      []ports.Port
	Err        error
	Background bool
}

// PortCheckedMsg carries the result of inspecting one port.
type PortCheckedMsg struct {
	Port  ports.Port
	Found bool
	Err   error
}

// KilledMsg carries the outcome of any kill operation.
type KilledMsg struct {
	Results []ports.KillResult
	Err     error
}

// SuggestedMsg carries free-port suggestions.
type SuggestedMsg struct {
	Near  int
	Ports []int
	Err   error
}

// ReservedMsg confirms a reservation.
type ReservedMsg struct {
	Port    int
	Service string
	Err     error
}

// HealthResultMsg carries one probe result.
type HealthResultMsg struct {
	Port   int
	Result ports.HealthResult
	Err    error
}

// WatchTickMsg is one listing from the watch subscription. Receiving it
// re-arms the subscription, which is what keeps watch mode live without ever
// blocking Update.
type WatchTickMsg struct{ Ports []ports.Port }

// WatchStoppedMsg fires when the watch channel closes.
type WatchStoppedMsg struct{}

// DataLoadedMsg carries every store-backed collection the views render.
type DataLoadedMsg struct {
	Aliases    []model.Alias
	Groups     []model.Group
	Blueprints []model.Blueprint
	Protected  model.ProtectedPorts
	History    []model.HistoryEntry
	Logs       []model.LogEntry
	Project    model.ProjectConfig
	HasProject bool
	Err        error
}

// SavedMsg reports the outcome of any store mutation.
type SavedMsg struct {
	What string
	Err  error
}

// BlueprintAppliedMsg carries the per-service outcome of applying a blueprint.
type BlueprintAppliedMsg struct {
	Name    string
	Results []blueprint.Result
	Err     error
}

// DriftMsg carries detected divergence from the expected environment.
type DriftMsg struct {
	Drifts []opsync.Drift
	Err    error
}

// RegistryMsg carries the Pro team registry.
type RegistryMsg struct {
	Members []opsync.Member
	Err     error
}

// WorkspacesMsg carries the Pro workspace list.
type WorkspacesMsg struct {
	Workspaces []opsync.Workspace
	Err        error
}

// LoggedInMsg carries the Pro session.
type LoggedInMsg struct {
	Session opsync.Session
	Err     error
}

// SyncedMsg reports the outcome of a push or pull.
type SyncedMsg struct {
	Direction string
	Err       error
}

// ErrorMsg surfaces an error that has no more specific home.
type ErrorMsg struct{ Err error }

// --- UI plumbing ---------------------------------------------------------

// ToastLevel is a notification's severity.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarn
	ToastError
)

// ToastMsg raises a transient notification.
type ToastMsg struct {
	Text  string
	Level ToastLevel
}

// ToastExpiredMsg dismisses a toast once its lifetime elapses.
type ToastExpiredMsg struct{ ID int }

// CloseOverlayMsg closes whichever overlay is on top.
type CloseOverlayMsg struct{}

// FormResultMsg carries a completed modal form's values, keyed by field.
type FormResultMsg struct {
	ID     string
	Values map[string]string
}

// ConfirmResultMsg carries a confirmation modal's answer.
type ConfirmResultMsg struct {
	ID        string
	Confirmed bool
	// Payload lets the opener round-trip whatever context the follow-up
	// action needs, so confirmation stays a generic component.
	Payload any
}

// TickMsg drives time-based UI updates.
type TickMsg time.Time

// AutoRefreshMsg fires the ambient background poll that keeps the ports list
// and the history current — a server started in another terminal appears, a
// killed one disappears — without the user asking. It re-arms itself.
type AutoRefreshMsg struct{}

// Toast helpers keep call sites terse at the emit site.

// Info raises an informational toast.
func Info(text string) ToastMsg { return ToastMsg{Text: text, Level: ToastInfo} }

// Success raises a success toast.
func Success(text string) ToastMsg { return ToastMsg{Text: text, Level: ToastSuccess} }

// Warn raises a warning toast.
func Warn(text string) ToastMsg { return ToastMsg{Text: text, Level: ToastWarn} }

// Error raises an error toast; these persist until dismissed.
func Error(text string) ToastMsg { return ToastMsg{Text: text, Level: ToastError} }
