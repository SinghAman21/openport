// Package actions is the single registry of everything OpenPort can do.
//
// The command palette and the actions menu both render this catalog, and the
// root model dispatches on Action.ID — so a capability is reachable from both
// front doors by construction, and adding one means adding a single entry
// here plus a case in the dispatcher.
package actions

// IDs for every action. Referenced by the root model's dispatcher.
const (
	// Ports & processes
	PortsList       = "ports.list"
	PortsCheck      = "ports.check"
	PortsKill       = "ports.kill"
	PortsKillPID    = "ports.kill_pid"
	PortsKillRange  = "ports.kill_range"
	PortsWatch      = "ports.watch"
	PortsSuggest    = "ports.suggest"
	PortsReserve    = "ports.reserve"
	PortsHealthTCP  = "ports.health.tcp"
	PortsHealthHTTP = "ports.health.http"

	// Aliases & groups
	AliasManage   = "alias.manage"
	AliasNew      = "alias.new"
	GroupManage   = "group.manage"
	GroupNew      = "group.new"
	GroupShutdown = "group.shutdown"
	GroupHealth   = "group.health"

	// Blueprints
	BlueprintList  = "blueprint.list"
	BlueprintNew   = "blueprint.new"
	BlueprintApply = "blueprint.apply"

	// Project config
	ConfigView = "config.view"
	ConfigInit = "config.init"

	// History & safety
	HistoryView     = "history.view"
	LogsView        = "logs.view"
	ProtectedList   = "protected.list"
	ProtectedToggle = "protected.toggle"

	// UI
	UIHelp    = "ui.help"
	UIMenu    = "ui.menu"
	UIRefresh = "ui.refresh"
	UIDebug   = "ui.debug"
	UIQuit    = "ui.quit"

	// Pro
	ProLogin      = "pro.login"
	ProWorkspaces = "pro.workspaces"
	ProPush       = "pro.push"
	ProPull       = "pro.pull"
	ProRegistry   = "pro.registry"
	ProDrift      = "pro.drift"
)

// Category groups actions in the menu and palette.
type Category string

const (
	CatPorts   Category = "Ports & processes"
	CatAliases Category = "Aliases & groups"
	CatBlue    Category = "Blueprints"
	CatConfig  Category = "Project config"
	CatSafety  Category = "History & safety"
	CatUI      Category = "Interface"
	CatPro     Category = "Pro"
)

// Action is one invocable capability.
type Action struct {
	ID       string
	Title    string
	Desc     string
	Category Category
	// Key is the direct binding shown as a hint, empty when there is none.
	Key string
	// Pro marks actions that require a Pro provider. They stay visible but
	// render a Pro badge and are disabled off Pro, so the feature is
	// discoverable without being a broken screen.
	Pro bool
}

// catalog is the master list, in menu order.
var catalog = []Action{
	// Ports & processes
	{ID: PortsList, Title: "List listening ports", Desc: "Show every port in use right now", Category: CatPorts, Key: "1"},
	{ID: PortsCheck, Title: "Check a port", Desc: "Inspect what holds a specific port", Category: CatPorts, Key: "c"},
	{ID: PortsKill, Title: "Kill a port", Desc: "Terminate whatever is listening on a port", Category: CatPorts, Key: "x"},
	{ID: PortsKillPID, Title: "Kill by PID", Desc: "Terminate a process by its process ID", Category: CatPorts},
	{ID: PortsKillRange, Title: "Kill a port range", Desc: "Terminate every listener between two ports", Category: CatPorts, Key: "X"},
	{ID: PortsWatch, Title: "Toggle watch mode", Desc: "Refresh the port table live", Category: CatPorts, Key: "w"},
	{ID: PortsSuggest, Title: "Suggest free ports", Desc: "Find unused ports near a hint", Category: CatPorts, Key: "f"},
	{ID: PortsReserve, Title: "Reserve a port", Desc: "Claim a port for a named service", Category: CatPorts},
	{ID: PortsHealthTCP, Title: "Health check (TCP)", Desc: "Dial a port and report whether it answers", Category: CatPorts, Key: "H"},
	{ID: PortsHealthHTTP, Title: "Health check (HTTP)", Desc: "GET a port and report the status code", Category: CatPorts},

	// Aliases & groups
	{ID: AliasManage, Title: "Manage aliases", Desc: "Name your ports: frontend → 3000", Category: CatAliases, Key: "3"},
	{ID: AliasNew, Title: "New alias", Desc: "Bind a name to a port", Category: CatAliases},
	{ID: GroupManage, Title: "Manage groups", Desc: "Bundle ports: dev_stack → 3000, 5432, 6379", Category: CatAliases},
	{ID: GroupNew, Title: "New group", Desc: "Define a named set of ports", Category: CatAliases},
	{ID: GroupShutdown, Title: "Shut down a group", Desc: "Kill every port in a group, with one confirm", Category: CatAliases},
	{ID: GroupHealth, Title: "Health check a group", Desc: "Probe every port in a group", Category: CatAliases},

	// Blueprints
	{ID: BlueprintList, Title: "Blueprints", Desc: "Environment templates and drift", Category: CatBlue, Key: "4"},
	{ID: BlueprintNew, Title: "New blueprint", Desc: "Describe the services an environment expects", Category: CatBlue},
	{ID: BlueprintApply, Title: "Apply a blueprint", Desc: "Compare live state to a blueprint and resolve conflicts", Category: CatBlue},

	// Project config
	{ID: ConfigView, Title: "Project config", Desc: "View and edit .openport.yml", Category: CatConfig, Key: "5"},
	{ID: ConfigInit, Title: "Create project config", Desc: "Write a starter .openport.yml here", Category: CatConfig},

	// History & safety
	{ID: HistoryView, Title: "Action history", Desc: "Everything OpenPort has done, with timestamps", Category: CatSafety, Key: "6"},
	{ID: LogsView, Title: "Logs", Desc: "Internal diagnostics and OS errors", Category: CatSafety},
	{ID: ProtectedList, Title: "Protected ports", Desc: "The never-kill list", Category: CatSafety},
	{ID: ProtectedToggle, Title: "Protect / unprotect a port", Desc: "Add or remove a port from the never-kill list", Category: CatSafety, Key: "p"},

	// Interface
	{ID: UIMenu, Title: "Action menu", Desc: "Browse every action", Category: CatUI, Key: "2"},
	{ID: UIRefresh, Title: "Refresh", Desc: "Reload ports and stored data", Category: CatUI, Key: "r"},
	{ID: UIHelp, Title: "Help", Desc: "Show the keymap overlay", Category: CatUI, Key: "?"},
	{ID: UIDebug, Title: "Dump debug state", Desc: "Write the model state to openport-debug.log", Category: CatUI, Key: "ctrl+d"},
	{ID: UIQuit, Title: "Quit", Desc: "Leave OpenPort", Category: CatUI, Key: "q"},

	// Pro
	{ID: ProLogin, Title: "Sign in", Desc: "Connect a workspace with GitHub or Google", Category: CatPro, Pro: true},
	{ID: ProWorkspaces, Title: "Workspaces", Desc: "Pick the team workspace to sync with", Category: CatPro, Key: "7", Pro: true},
	{ID: ProPush, Title: "Push to cloud", Desc: "Upload aliases, groups and blueprints", Category: CatPro, Pro: true},
	{ID: ProPull, Title: "Pull from cloud", Desc: "Replace local state with the workspace snapshot", Category: CatPro, Pro: true},
	{ID: ProRegistry, Title: "Team registry", Desc: "Who runs which service, and where", Category: CatPro, Pro: true},
	{ID: ProDrift, Title: "Detect drift", Desc: "Compare live state against the expected environment", Category: CatPro, Pro: true},
}

// All returns the full catalog. The Pro entries are always included so the
// feature set stays discoverable; callers use IsPro to decide whether to
// enable them.
func All() []Action {
	out := make([]Action, len(catalog))
	copy(out, catalog)
	return out
}

// Find looks an action up by ID.
func Find(id string) (Action, bool) {
	for _, a := range catalog {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}

// Categories returns the categories in menu order.
func Categories() []Category {
	seen := map[Category]bool{}
	var out []Category
	for _, a := range catalog {
		if !seen[a.Category] {
			seen[a.Category] = true
			out = append(out, a.Category)
		}
	}
	return out
}
