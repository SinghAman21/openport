# OpenPort — One-Shot Build Spec for Claude Code (Opus)

> **How to use this file:** Place it at the repo root and open Claude Code in this
> directory. This is both the build brief *and* the standing guardrail document
> (Claude Code reads it every session). Build the entire project from this spec.
> Work through the phases in order (§13), keep every phase compiling and tested
> before starting the next, and run your own `go build` / `go vet` / `go test`
> loop to self-correct. Do not report a phase "done" until its acceptance
> criteria (§14) pass.
>
> Inspired by the ergonomics of opencode: the interface *is* the product, the CLI
> is a thin launcher, and every capability is reachable without memorizing flags.

---

## 1. Mission

Build **OpenPort**: a modern, fully TUI-driven port & process cockpit for local
development, written idiomatically in Go on the Charm stack (Bubble Tea / Lip
Gloss / Bubbles / Huh).

The user types one command — `openport` — and a rich, keyboard-navigable TUI
launches. Users **pick actions from menus, dropdowns, and a command palette**,
then fill placeholder values (port, alias, project name) via inline forms and
modal dialogs. Results render in structured tables and detail panes with clear
feedback. Users never drop back to a raw shell and never memorize flags.

**Interaction-first, commands-second.** There is exactly one code path per
capability, invoked from two front-ends: the TUI (primary) and a thin headless
CLI (for scripting). The TUI and CLI both call the same `ports.Service`.

**Not a thin wrapper around `lsof`/`kill`.** It's a cockpit: watch mode, aliases,
groups, blueprints, health checks, drift detection, history/audit, protected
ports, and a Pro cloud seam.

Cross-platform: **macOS and Linux at minimum** (Windows may be a build-tagged
stub). Idiomatic Go, clean architecture, testable without a terminal.

---

## 2. Non-negotiable architecture rules

These are invariants. Violating any of them is a bug.

1. `internal/tui` **NEVER** imports `internal/platform`. The UI depends only on
   `ports.Service` and `store.Store` (and, for Pro, `sync.SyncProvider`).
2. All OS-specific code lives in `internal/platform` behind the `Provider`
   interface, guarded by `//go:build` tags. No `lsof`/`ss`/`/proc` calls anywhere
   else.
3. On Linux, **prefer std-lib `/proc` parsing**; shell out to `ss` only as a
   fallback. On macOS, use `lsof`. The service and UI must not know which ran.
4. **State mutations happen ONLY in `Update`.** `tea.Cmd` functions are pure
   dispatch — they call the service and return a message; they never write shared
   state.
5. Every **mutating action** (kill, kill-range, apply blueprint, reserve port)
   appends a `HistoryEntry` via the store.
6. Every **destructive action** (kill, kill-range, group shutdown) requires a
   confirmation modal, and must respect the **protected ports** list.
7. **Pro/cloud code depends on `sync.SyncProvider`; the default is
   `NoopProvider`.** The entire offline path must work with zero network. No core
   feature may import a concrete remote client.
8. Pro features are **additive** — extra views + extra palette entries — never a
   different interaction paradigm. Gate their UI on `provider.IsPro()`.
9. No global mutable state. Dependencies (`Service`, `Store`, `SyncProvider`) are
   constructed in `main` and injected.

---

## 3. Tech stack (pin these)

```
github.com/charmbracelet/bubbletea   // runtime / event loop (The Elm Architecture)
github.com/charmbracelet/lipgloss    // styling / layout / compositing
github.com/charmbracelet/bubbles     // table, list, textinput, viewport, help, key, spinner
github.com/charmbracelet/huh         // forms & modal inputs (embeds as a Bubble Tea model)
github.com/sahilm/fuzzy              // command palette fuzzy matching
gopkg.in/yaml.v3                     // .openport.yml project config
```

Standard library for everything else (JSON store, `/proc` parsing, HTTP/TCP
health checks, signals). Do not add heavier deps until Pro remote work needs
them. Target the latest stable Go; commit `go.mod`/`go.sum`.

---

## 4. Folder structure (create exactly this)

```
openport/
├── cmd/openport/main.go              # entrypoint: detect provider, build service, launch TUI or headless
├── internal/
│   ├── cli/headless.go               # thin scripting front-end -> ports.Service
│   ├── ports/                        # CORE: OS-agnostic port/process service
│   │   ├── ports.go                  # Service interface + domain types (Port, KillResult)
│   │   ├── service.go                # concrete Service, enriches sockets w/ aliases+groups+protection
│   │   ├── suggest.go                # free-port suggestion + optional reservation
│   │   ├── health.go                 # HTTP/TCP health checks
│   │   ├── watch.go                  # Watch(): channel + cancel
│   │   └── service_test.go           # table-driven, uses fakeProvider (NEVER hits real OS)
│   ├── platform/                     # OS-SPECIFIC, behind Provider, build-tagged
│   │   ├── platform.go               # Provider interface + Detect() + Socket + Signal
│   │   ├── darwin.go                 # //go:build darwin   (lsof)
│   │   ├── linux.go                  # //go:build linux    (/proc + ss fallback)
│   │   ├── windows.go                # //go:build windows  (stub returning ErrUnsupported ok)
│   │   └── kill.go                   # signal/terminate helpers (build-tagged as needed)
│   ├── model/                        # shared domain types (UI + persistence)
│   │   ├── alias.go                  # Alias
│   │   ├── group.go                  # Group
│   │   ├── blueprint.go              # Blueprint, BlueprintService
│   │   ├── project.go                # ProjectConfig (.openport.yml)
│   │   ├── protected.go              # ProtectedPorts
│   │   └── history.go                # HistoryEntry, LogEntry
│   ├── store/                        # persistence (XDG paths)
│   │   ├── store.go                  # Store interface
│   │   ├── file_store.go             # JSON on disk + project YAML loader
│   │   └── store_test.go
│   ├── sync/                         # PRO SEAM (no-op default)
│   │   ├── provider.go               # SyncProvider interface + NoopProvider + Detect(license)
│   │   ├── types.go                  # Snapshot, Workspace, Drift, Member
│   │   └── remote.go                 # RemoteProvider (stubbed; behind auth/license gate)
│   ├── tui/                          # UI LAYER
│   │   ├── app.go                    # root Model, top-level Update/View, router, overlay compositing
│   │   ├── keys.go                   # global + per-view keymaps (arrows + Vim j/k/h/l)
│   │   ├── messages.go               # ALL tea.Msg types (intents + results)
│   │   ├── commands.go               # tea.Cmd factories calling ports.Service async
│   │   ├── styles/theme.go           # Lip Gloss theme tokens (colors, borders, spacing, badges)
│   │   ├── components/
│   │   │   ├── menu.go               # main action menu (list-based, dropdown-like)
│   │   │   ├── palette.go            # command palette (fuzzy quick-actions)
│   │   │   ├── statusbar.go          # bottom bar: view, selection, hints, watch indicator
│   │   │   ├── toast.go              # transient notifications (success/warn/error)
│   │   │   ├── modal.go              # generic modal wrapper (z-ordered overlay)
│   │   │   ├── form.go               # inline/modal input forms (huh-backed)
│   │   │   ├── confirm.go            # yes/no confirmation (destructive actions)
│   │   │   └── help.go               # help/keymap overlay
│   │   └── views/
│   │       ├── view.go               # View interface all screens implement
│   │       ├── ports_view.go         # main Ports table + watch mode + row actions
│   │       ├── actions_view.go       # discoverable action menu
│   │       ├── aliases_view.go       # aliases + groups manager
│   │       ├── blueprints_view.go    # blueprint list / apply / drift badges
│   │       ├── config_view.go        # per-project config + inline editor
│   │       ├── history_view.go       # actions history + scrollable log viewer
│   │       └── workspace_view.go     # PRO: team registry + login (hidden unless IsPro)
│   └── version/version.go
├── CLAUDE.md                         # symlink or copy of §2 + §14 (guardrails that survive compaction)
├── Makefile                          # build test lint fmt vet cross-compile
├── go.mod / go.sum
└── README.md
```

---

## 5. Core contracts — WRITE AND COMMIT THESE FIRST

Before any implementation, create every interface file below and commit them.
Fixing the contracts up front is what prevents architectural drift across the
one-shot build.

### 5.1 `internal/platform/platform.go`

```go
package platform

type Provider interface {
    ListListening(ctx context.Context) ([]Socket, error)
    ProcessOnPort(ctx context.Context, port int) (Socket, error)
    Kill(ctx context.Context, pid int, sig Signal) error
}

type Socket struct {
    Port     int
    Protocol string // tcp, tcp6, udp
    PID      int
    Process  string // executable name
    Command  string // full command line, best-effort
    User     string
    Addr     string // 127.0.0.1, 0.0.0.0, ::
    State    string // LISTEN, ESTABLISHED, ...
}

type Signal int
const ( Terminate Signal = iota; Force ) // SIGTERM, SIGKILL

func Detect() (Provider, error) // returns the build-tagged impl for this OS
var ErrNotFound = errors.New("no process on port")
var ErrUnsupported = errors.New("platform not supported")
```

### 5.2 `internal/ports/ports.go`

```go
package ports

type Service interface {
    List(ctx context.Context) ([]Port, error)
    Check(ctx context.Context, port int) (Port, bool, error)
    Kill(ctx context.Context, port int, force bool) (KillResult, error)
    KillPID(ctx context.Context, pid int, force bool) (KillResult, error)
    KillRange(ctx context.Context, lo, hi int, force bool) ([]KillResult, error)
    SuggestFree(ctx context.Context, near, count int) ([]int, error)
    Reserve(ctx context.Context, port int, service string) error
    Health(ctx context.Context, port int, kind HealthKind) (HealthResult, error)
    Watch(ctx context.Context, interval time.Duration) (<-chan []Port, func())
}

type Port struct {
    platform.Socket
    Alias     string   // resolved from store, "" if none
    Groups    []string // group memberships
    Protected bool     // in protected list
    Health    *HealthResult
}

type KillResult struct { Port, PID int; Process string; Forced, Skipped bool; Reason string; Err error }
type HealthKind int
const ( HealthTCP HealthKind = iota; HealthHTTP )
type HealthResult struct { OK bool; Status string; LatencyMS int; Code int }
```

Notes: `Kill`/`KillRange` MUST refuse protected ports (return `Skipped=true`,
`Reason="protected"`) unless an explicit override flag is threaded through — and
even then the TUI must double-confirm. `List`/`Check` enrich sockets with alias,
group, protection, and (lazily) health from the store.

### 5.3 `internal/store/store.go`

```go
package store

type Store interface {
    Aliases() ([]model.Alias, error)
    SaveAlias(model.Alias) error
    DeleteAlias(name string) error

    Groups() ([]model.Group, error)
    SaveGroup(model.Group) error
    DeleteGroup(name string) error

    Blueprints() ([]model.Blueprint, error)
    SaveBlueprint(model.Blueprint) error
    DeleteBlueprint(name string) error

    ProtectedPorts() (model.ProtectedPorts, error)
    SaveProtectedPorts(model.ProtectedPorts) error

    ProjectConfig(dir string) (model.ProjectConfig, bool, error) // reads .openport.yml
    SaveProjectConfig(dir string, cfg model.ProjectConfig) error

    AppendHistory(model.HistoryEntry) error
    History(limit int) ([]model.HistoryEntry, error)
    AppendLog(model.LogEntry) error
    Logs(limit int) ([]model.LogEntry, error)

    Snapshot() (sync.Snapshot, error)          // for Pro push
    ApplySnapshot(sync.Snapshot) error          // for Pro pull
}

func NewFileStore() (Store, error) // JSON under $XDG_CONFIG_HOME/openport (or macOS equivalent)
```

### 5.4 `internal/sync/provider.go`

```go
package sync

type SyncProvider interface {
    IsPro() bool
    Login(ctx context.Context, provider OAuthProvider) (Session, error) // github|google
    Workspaces(ctx context.Context) ([]Workspace, error)
    SelectWorkspace(ctx context.Context, id string) error
    Push(ctx context.Context, snap Snapshot) error
    Pull(ctx context.Context) (Snapshot, error)
    Registry(ctx context.Context) ([]Member, error)     // who runs what/where
    DetectDrift(ctx context.Context, local Snapshot) ([]Drift, error)
}

type NoopProvider struct{}
func (NoopProvider) IsPro() bool { return false }
// ... all other methods return ErrProRequired

func Detect(license string) SyncProvider // NoopProvider unless a valid license/token is present
var ErrProRequired = errors.New("this feature requires OpenPort Pro")
```

`RemoteProvider` in `remote.go` is stubbed for this build (methods can return
`ErrProRequired` or a clearly-marked TODO), but the interface, types, and gating
must be fully wired so Pro lights up with a single provider swap.

---

## 6. TUI state model

One root `Model`; views are sub-models it routes to.

```go
type Route int
const (
    RoutePorts Route = iota
    RouteActions
    RouteAliases
    RouteBlueprints
    RouteConfig
    RouteHistory
    RouteWorkspace // PRO
)

type Model struct {
    svc      ports.Service
    store    store.Store
    provider sync.SyncProvider

    route  Route
    views  map[Route]views.View
    width  int
    height int

    palette   components.Palette // command palette overlay
    modal     *components.Modal  // active modal (form/confirm/help), nil if none
    statusbar components.StatusBar
    toasts    []components.Toast

    watching bool
    keys     KeyMap
    theme    styles.Theme
}
```

`views.View` interface (adding a screen = writing one file):

```go
type View interface {
    Update(msg tea.Msg) (View, tea.Cmd)
    View(width, height int) string
    Help() []key.Binding // feeds help overlay + status bar hints
    Title() string
}
```

**Overlay compositing & input routing.** Root `View()` renders the active screen,
then composites (in z-order) modal → palette → toasts using Lip Gloss `Place`.
Input routing is the reverse: if a modal is open it eats keys first, then the
palette, then the active view, then global shortcuts. Never use two overlays that
both capture input simultaneously.

---

## 7. Event loop & messages

Two message families in `messages.go` — **Intents** (user wants X) and
**Results** (X happened). Keep them separated.

```go
// Intents
type NavigateMsg struct{ To Route }
type OpenPaletteMsg struct{}
type ListPortsMsg struct{}
type CheckPortMsg struct{ Port int }
type KillPortMsg struct{ Port int; Force bool }
type KillRangeMsg struct{ Lo, Hi int; Force bool }
type SuggestPortMsg struct{ Near, Count int }
type HealthCheckMsg struct{ Port int; Kind ports.HealthKind }
type ApplyBlueprintMsg struct{ Name string }
type SaveAliasMsg struct{ Alias model.Alias }
// ... one per action

// Results
type PortsLoadedMsg struct{ Ports []ports.Port; Err error }
type PortCheckedMsg struct{ Port ports.Port; Found bool; Err error }
type KilledMsg struct{ Results []ports.KillResult; Err error }
type SuggestedMsg struct{ Ports []int; Err error }
type HealthResultMsg struct{ Port int; Result ports.HealthResult; Err error }
type WatchTickMsg struct{ Ports []ports.Port } // from the self-re-arming subscription
type DriftMsg struct{ Drifts []sync.Drift; Err error }
type ErrorMsg struct{ Err error }
```

`commands.go` bridges to the service **without blocking render**:

```go
func listPortsCmd(svc ports.Service) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        ps, err := svc.List(ctx)
        return PortsLoadedMsg{Ports: ps, Err: err}
    }
}
```

**Watch mode = self-re-arming subscription.** On entering watch, fire a command
that reads ONE value from the watch channel and, on receiving `WatchTickMsg`,
returns the same command again. This keeps ticking while the user moves the
selection, opens the palette, or triggers a kill — nothing blocks because the
channel read happens on Bubble Tea's command goroutines, not in `Update`.
Selection persists across ticks by tracking `PID+Port`, not row index.

```go
func watchCmd(ch <-chan []ports.Port) tea.Cmd {
    return func() tea.Msg {
        ps, ok := <-ch
        if !ok { return nil }
        return WatchTickMsg{Ports: ps}
    }
}
```

---

## 8. Feature catalog → implementation map

Every feature below must be reachable via **menu, palette, and (where sensible) a
key binding**. Free vs Pro is marked. Build all Free features fully; build Pro
UI + gating fully but leave `RemoteProvider` network calls stubbed.

### category: tui_core (all Free)

| id | Where it lives | Behavior |
|---|---|---|
| `tui_main_menu` | `views/actions_view.go`, `components/menu.go` | Central menu launched by `openport`; list-style, dropdown-like; items: List, Check, Kill, Kill range, Watch, Suggest free, Aliases, Groups, Blueprints, Config, History/Logs. Enter runs or opens a form. |
| `tui_keyboard_navigation` | `keys.go`, all views | Arrows **and** Vim `j/k` (up/down), `h/l` (left/right / back-forward), `g/G` top/bottom, `Enter` select, `Esc` back/close. Consistent everywhere. |
| `tui_dropdown_actions` | `components/menu.go` | Reusable picker component used by the main menu and row-action popovers. |
| `tui_form_inputs` | `components/form.go` (huh) | Modal/inline forms for port number, alias name, project name, range, etc. Validation inline. |
| `tui_tables` | `components/` + `bubbles/table` | Sortable-column tables for ports, aliases, groups, blueprints, history. Column sort via `s` / header select. |
| `tui_status_bar` | `components/statusbar.go` | Bottom bar: current view title, selection summary, contextual hints, `● watching` indicator, and slot for toasts. |
| `tui_help_overlay` | `components/help.go` | `?` opens a keymap/help overlay built from each view's `Help()`. `Esc`/`?` closes. |
| `tui_command_palette` | `components/palette.go` | `:` or `Ctrl+K` opens fuzzy quick-action launcher (`sahilm/fuzzy`). Selecting runs the action or opens its form. Every action registers a palette entry. |

### category: ports_processes (all Free)

| id | Where | Behavior |
|---|---|---|
| `ports_list` | `ports.Service.List`, `views/ports_view.go` | Table: port · proto · PID · process · command · state · alias/group. Home view. |
| `ports_check` | `Service.Check` | Palette/form prompts for a port; detail pane shows process + health. |
| `ports_kill` | `Service.Kill`, `components/confirm.go` | Prompt port or PID; safe (SIGTERM) vs force (SIGKILL) modes; confirm modal; protected-port guard; writes history. |
| `ports_kill_range` | `Service.KillRange` | Prompt `lo`–`hi`; confirm with count; skip protected with a per-row reason; writes history. |
| `ports_watch` | `Service.Watch`, `watchCmd` | Live table (`w` toggles); row actions available during watch; non-blocking; selection persists. |
| `ports_health_check` | `Service.Health` | HTTP/TCP checks; status badges/icons (✓ ok, ✕ down, ⋯ checking) in table + detail. |
| `ports_auto_suggest` | `Service.SuggestFree`, `Reserve` | Find unused port near a hint; optionally reserve for a service name; result shown + copyable. |

### category: aliases_and_configs (all Free)

| id | Where | Behavior |
|---|---|---|
| `aliases_manage` | `views/aliases_view.go`, store | CRUD aliases (`frontend`→3000). Changes immediately enrich the Ports view. |
| `groups_manage` | `views/aliases_view.go` (groups tab), store | Define groups (`dev_stack`→{3000,5432,6379}); run group actions (e.g., shutdown all, health-check all) with one confirm. |
| `project_config` | `model/project.go`, `store.ProjectConfig` | Load/edit `.openport.yml` in the working dir: expected ports, aliases, ranges. Auto-detected on launch if present. |
| `config_editor` | `views/config_view.go` | Inline TUI editor to tweak config without leaving the tool; validates before save. |

### category: history_and_safety (all Free)

| id | Where | Behavior |
|---|---|---|
| `actions_history` | `views/history_view.go`, store | Scrollable list of past kills, suggestions, checks, blueprint applies with timestamp + details. |
| `logs_viewer` | `views/history_view.go` (logs tab), `bubbles/viewport` | Scrollable internal log + OS interaction errors. |
| `protected_ports` | `model/protected.go`, enforced in `Service.Kill*` | Configurable never-kill list (system/critical). Selecting one warns; killing requires explicit override + double confirm. |
| `confirm_before_kill` | `components/confirm.go` | Modal confirmation before any destructive action (kill, range, group shutdown). |

### category: pro_cloud_ready (Pro — full UI + gating, stubbed remote)

| id | Where | Behavior |
|---|---|---|
| `env_blueprints` | `model/blueprint.go`, `views/blueprints_view.go` | **Free to define/apply locally.** Templates of services/ports/commands; apply walks the blueprint, suggests free ports for conflicts, records history. |
| `cloud_sync` | `sync.SyncProvider.Push/Pull` | Sync aliases/groups/configs across devices. Status-bar "Sync" entry; gated on `IsPro()`. |
| `team_registry` | `views/workspace_view.go`, `Registry()` | Dashboard of who runs which services/ports across the team. |
| `drift_detection` | `DetectDrift`, badges in ports/blueprints views | Compare live state vs blueprint; surface divergence as badges + a drift panel. |
| `cloud_login` | `views/workspace_view.go`, `Login/SelectWorkspace` | GitHub/Google OAuth flow + workspace selection, all inside the TUI. |

> **Blueprints nuance:** local define+apply is a Free feature; the *cloud-shared*
> blueprint registry and drift-against-team-blueprint are Pro. Keep the local
> `BlueprintService` independent of `sync`.

---

## 9. CLI ↔ internal action mapping (thin headless layer)

`main.go` launches the TUI when invoked bare; `internal/cli/headless.go` handles
scripting subcommands, all routing to the same `ports.Service`.

| Invocation | Behavior |
|---|---|
| `openport` | Launch TUI on Ports view |
| `openport --view aliases\|blueprints\|history\|config\|actions` | Launch TUI into a specific view |
| `openport ls` | Print listening ports as a table, exit |
| `openport check <port>` | Check one port, exit |
| `openport kill <port> [--force]` | Confirm + kill (or `--force` to skip), exit; respects protected list |
| `openport kill-range <lo> <hi> [--force]` | Kill range, exit |
| `openport free [--near <port>] [--count n]` | Suggest free ports, exit |
| `openport apply <blueprint>` | Apply a blueprint, exit |

The palette is the TUI-native equivalent of these subcommands — same actions, no
flags to memorize.

```go
func main() {
    prov, err := platform.Detect()
    if err != nil { log.Fatal(err) }
    st, err := store.NewFileStore()
    if err != nil { log.Fatal(err) }
    svc := ports.NewService(prov, st)
    provider := sync.Detect(os.Getenv("OPENPORT_LICENSE")) // NoopProvider by default

    if len(os.Args) > 1 && !isViewFlag(os.Args[1]) {
        os.Exit(cli.RunHeadless(os.Args[1:], svc))
    }
    m := tui.New(svc, st, provider, initialRouteFromArgs())
    if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
        log.Fatal(err)
    }
}
```

---

## 10. Platform implementation notes

- **darwin.go** (`//go:build darwin`): `lsof -iTCP -sTCP:LISTEN -nP` (+ UDP pass);
  parse into `Socket`. Best-effort full command via `ps -o command= -p <pid>`.
- **linux.go** (`//go:build linux`): parse `/proc/net/tcp`, `/proc/net/tcp6`,
  `/proc/net/udp`; map inode→pid via `/proc/*/fd/*`; process name from
  `/proc/<pid>/comm`, command from `/proc/<pid>/cmdline`. Fall back to `ss -tulpn`
  only if `/proc` is unavailable. **No external dep required.**
- **windows.go** (`//go:build windows`): stub that returns `ErrUnsupported` from
  `Detect()` is acceptable; keep it compiling.
- **kill.go**: `Terminate`→`SIGTERM`, `Force`→`SIGKILL` via `syscall`/`os.Process`.

Handle permission errors gracefully (some sockets need elevated rights): surface
a clear toast + log entry, never crash.

---

## 11. Theme & UX polish

- Lip Gloss theme tokens in `styles/theme.go`: accent, muted, success, warning,
  danger, border, badge styles. Light/dark-terminal friendly.
- Status badges for health (✓ / ✕ / ⋯) and protection (🔒 or `[P]`).
- Contextual hints in the status bar update per view and per selection.
- Vim-like *and* arrow navigation everywhere; `?` help; `:`/`Ctrl+K` palette;
  `Esc` universally closes the top overlay or goes back.
- Toasts auto-dismiss after a few seconds; errors persist until dismissed.
- Add a `--debug` flag that dumps the current `Model` state to
  `./openport-debug.log` on demand (bind a key, e.g. `Ctrl+D`) so the build agent
  can inspect state it can't see rendered.

---

## 12. Pro extension seam (build the wiring, stub the network)

- `sync.Detect(license)` returns `NoopProvider` unless a license/token is present.
- Every Pro UI element checks `provider.IsPro()` before rendering; when not Pro,
  show a subtle "Pro" affordance instead of the feature (never a broken screen).
- `RouteWorkspace` view and Pro palette entries exist but are hidden/disabled off
  Pro.
- `store.Snapshot()`/`ApplySnapshot()` are the serialization boundary for
  push/pull. `DetectDrift` compares a local `Snapshot` against a workspace
  blueprint.
- `RemoteProvider` methods may return `ErrProRequired`/TODO for now — but types,
  gating, and view wiring must be complete so enabling Pro is a one-line swap.

---

## 13. Build order (phases — one commit per phase, keep each runnable)

1. **Contracts.** Create every interface/type file in §5 + `model/*` structs.
   Commit. `go build ./...` clean (impls can be `panic("todo")` temporarily where
   needed, but prefer compiling stubs).
2. **Platform + service.** Implement `Detect` for your primary OS, then
   `ports.Service` (`List`/`Check`/`Kill`/`KillRange`/`SuggestFree`/`Health`/
   `Watch`). Table-driven tests with a `fakeProvider`. Verify via a throwaway
   main; **no TUI yet**.
3. **Store.** `FileStore` (XDG JSON) + project YAML loader + protected ports +
   history/logs. Tests.
4. **TUI shell.** Root `Model`, router, Ports view rendering a static table from
   the service, status bar, quit, help overlay. Prove the loop.
5. **Async + interactions.** Wire `tea.Cmd`s; row actions (check/kill with confirm
   modal + protected guard); toasts; suggest-free; health badges.
6. **Watch mode.** Self-re-arming subscription; selection persistence across ticks.
7. **Palette + remaining Free views.** Command palette, Actions menu, Aliases/
   Groups, Blueprints (local), Config editor, History/Logs. Forms via `huh`.
8. **Headless CLI.** All subcommands in §9 routing to the service.
9. **Pro seam.** `sync` interface + `NoopProvider` + gating + Workspace view +
   drift badges (remote stubbed).
10. **Polish.** Theme, Vim keys everywhere, contextual help, `--debug`, README,
    Makefile targets, cross-compile check for darwin+linux.

---

## 14. Definition of done (acceptance criteria — verify before reporting complete)

**Per change / phase:**
- `go build ./...` clean
- `go vet ./...` clean
- `gofmt -l .` prints nothing
- `go test ./...` green; new logic has table-driven tests using a fake
  `Provider`/`Store` (**tests never touch the real OS or network**)

**Whole project:**
- `openport` launches the TUI; every one of the 28 features in §8 is reachable via
  menu **and** palette; destructive actions confirm and respect protected ports.
- `internal/tui` imports neither `internal/platform` nor any concrete remote
  client (enforce with a grep in CI/Makefile: fail if `tui/` imports `platform`).
- Offline path works with `OPENPORT_LICENSE` unset (NoopProvider); no network
  calls on any Free path.
- Watch mode refreshes live while remaining fully interactive; selection persists.
- Builds for `GOOS=darwin` and `GOOS=linux` (`make cross`).
- README documents install, key bindings, palette, config format, and the Free vs
  Pro split.

**Makefile must provide:** `build`, `run`, `test`, `vet`, `fmt`, `lint`,
`cross`, and an `arch-check` target that greps for forbidden imports
(`tui/`→`platform`, core→concrete remote) and fails the build if found.

---

## 15. Guardrails to keep in `CLAUDE.md` (copy §2 + §14 there)

Because context gets compacted during a long one-shot build, mirror the
**architecture rules (§2)** and **definition of done (§14)** into a `CLAUDE.md` at
the repo root so they persist across sessions. Re-read them before each phase.

**Two failure modes to actively avoid:**
1. Leaking `lsof`/`ss`/`/proc` calls out of `internal/platform` because it's the
   fast path. Keep them behind `Provider`.
2. Coupling core features to a concrete remote client. Depend on
   `sync.SyncProvider`; default to `NoopProvider`.

Build it as a cohesive interactive cockpit, not a flag wrapper. Ship it.
