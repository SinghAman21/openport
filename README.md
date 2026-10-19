# OpenPort

A keyboard-driven port & process cockpit for local development.

You know the moment: `EADDRINUSE: address already in use :::3000`. The usual
answer is a shell incantation you re-derive every time — `lsof -i :3000`, squint
at a PID, `kill -9`, hope it was the right one. OpenPort replaces that with an
interface: see what is listening, see what it belongs to, and act on it without
memorising a single flag.

```
openport
```

That is the whole interface. Everything else is discoverable from inside it.

---

## Install

### Linux

Install the latest prebuilt binary without installing Go:

```bash
curl -fsSL https://raw.githubusercontent.com/SinghAman21/openport/main/install.sh | sh
```

The installer detects Linux `amd64` or `arm64`, installs to `~/.local/bin`, and
explains the `PATH` step if needed.

### Windows PowerShell

A `.sh` file cannot run directly in native PowerShell. Use the PowerShell
bootstrap URL:

```powershell
irm https://raw.githubusercontent.com/SinghAman21/openport/main/install.ps1 | iex
```

The installer detects Windows `amd64` or `arm64`, installs to your user-local
bin directory, and updates your user `PATH`. Open a new PowerShell window after
installation, then run:

```powershell
openport
```

### Go or source install

If Go 1.25+ is already installed:

```bash
go install github.com/SinghAman21/openport/cmd/openport@latest
```

Or build from source:

```bash
git clone https://github.com/SinghAman21/openport
cd openport
make build      # produces ./openport
make install    # or install into $GOBIN
```

macOS, Linux, and Windows are supported. Windows uses the native IP Helper and
process APIs, so no external port-listing tools are needed.

The install scripts download assets from GitHub Releases. Release assets must
use the names produced by `make cross`, such as `openport-linux-amd64` and
`openport-windows-amd64.exe`.

If you host these files at your own website, expose `/install.sh` for Linux and
Git Bash/MSYS users, and `/install.ps1` for native Windows PowerShell users.
The shell bootstrap automatically hands off to `install.ps1` when it detects
Git Bash/MSYS/Cygwin. A single URL cannot be natively executed by both POSIX
shells and PowerShell without a server-side installer endpoint.

---

## The interface

Launching with no arguments opens the Ports view: a live table of everything
listening, with a detail pane for the selection.

```
openport   1 Ports   2 Actions   3 Aliases   4 Blueprints   5 Config   6 History   7 Workspace ᴾᴿᴼ

34 ports                                                          sort: port ↑
 PORT   PROTO  PID     PROCESS       ALIAS     STATE   HLTH      ╭──────────────────────────
 3000   tcp    5678    node          frontend  LISTEN  ✓         │ 3000  frontend
 5432   tcp    1234    postgres      db        LISTEN  ✓         │
 6379   tcp    9012    redis-server            LISTEN  ·         │ Process  node
 22     tcp    901     sshd                    LISTEN  ·   🔒    │ PID      5678
                                                                 │ Protocol tcp
 Ports  › 3000 frontend        c check · x kill · H health · w watch · / filter
```

There are two ways to reach any capability, and they run the same code:

- **`2` — the actions menu.** Everything OpenPort can do, described and paged,
  with `/` to narrow it.
- **`:` or `ctrl+k` — the command palette.** Fuzzy-search the same catalog.

Anything needing input (a port number, an alias, a range) opens a form with
inline validation. Anything destructive opens a confirmation. **`esc` closes
whatever is open**, always — the modal layer handles it before the content
does, so no widget can leave you stranded in a popup.

The mouse wheel scrolls whatever is on top — table, menu, palette or help.
Mouse tracking does mean your terminal's own click-drag selection is taken
over; hold `shift` while dragging to select text as usual.

Anything that takes a moment says so. Scanning for a free port binds each
candidate to check it is genuinely available, so the status bar shows a spinner
and what it is doing rather than leaving you wondering whether it has hung.

The layout degrades rather than disappearing. Narrow the window and the ports
table sheds its least important columns, the detail pane moves underneath the
table instead of vanishing, and the tab strip falls back to bare numbers so the
navigation shortcuts stay visible. A list that cannot show everything says how
much is above and below rather than quietly ending.

### Key bindings

Arrows and Vim keys work everywhere; neither is second-class.

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | move |
| `←`/`h`, `→`/`l` | previous / next tab |
| `g`, `G` | top, bottom |
| `enter` | select |
| `esc` | close any popup, dismiss toasts, or go back |
| `q`, `ctrl+c` | quit |
| `:`, `ctrl+k` | command palette |
| `?` | help overlay (shows the active view's bindings too) |
| `1`–`7` | jump to a view |
| `r` | refresh |
| `ctrl+z` | suspend to the shell (`fg` to return; it refreshes on the way back) |
| `ctrl+d` | dump model state to `./openport-debug.log` |

In the Ports view:

| Key | Action |
|---|---|
| `c` | check the selected port |
| `x` / `X` | kill (SIGTERM) / force kill (SIGKILL) |
| `P` | kill by PID |
| `H` | health check |
| `a` | alias the selected port |
| `p` | protect / unprotect |
| `w` | toggle watch mode |
| `f` | suggest a free port |
| `/` | filter (ports, and the actions menu) |
| `s` / `S` | cycle sort column / reverse |
| `tab` | show/hide the detail pane |

---

## What it does

**Ports and processes.** List every listener with its process, PID, command
line, user and bind address. The list is **live** — a server you start in
another terminal appears within a few seconds, a killed one disappears, and the
header shows a `+2 -1` delta so you can see it happening. Your selection follows
the process, not the row number, so the cursor does not jump when the table
reorders. Check a single port. Kill by port, by PID, or across a range. Watch
mode (`w`) is the same thing at a faster cadence with a `● watching` indicator,
for when you are actively staring at a restart. TCP and HTTP health checks
render as badges.

**Names.** Aliases give a port a name (`frontend` → 3000). Groups bundle ports
(`dev_stack` → 3000, 5432, 6379) so you can shut down or health-check a whole
stack behind one confirmation. Both show up immediately in the ports table.

**Free ports.** Ask for an unused port near a hint. A port only qualifies if
nothing is listening, it is not reserved, it is not protected, and it can
actually be bound right now — which catches sockets a listing alone would miss.
Reserve one for a service and it stops being suggested to anything else.

**Blueprints.** Describe the services an environment expects — names, ports,
health probes — and apply it to see what is up, what is not running, and what
has taken a port it should not have. Applying is read-only: it tells you what
it would take to be correct and suggests somewhere else to put a service that
has been displaced. It never starts or kills anything on your behalf.

**Project config.** Drop an `.openport.yml` in a project and OpenPort picks it
up automatically, including from a subdirectory. Edit it from inside the tool.

**History and safety.** Every mutating action is recorded with a timestamp, and
the History view updates as they happen — no restart needed. A separate log
captures OS-level trouble — permission errors, procfs fallbacks, probes that
timed out — so a port showing no PID has a visible explanation rather than
being a mystery.

---

## Protected ports

OpenPort ships with a never-kill list — ssh, DNS, the printing daemon, and the
system supervisors — because a port tool whose worst-case outcome is locking
you out of your own machine is not a tool you can relax with.

A protected port cannot be killed by accident:

- The plain kill path refuses it outright, whatever the force flag says.
- Overriding requires two confirmations: one to override the protection, and a
  second to actually do it.
- The command line will not override protection at all. Manage the list in the
  interface (`openport --view config`, then the Protected tab).
- Refusals are reported per port with a reason, and a kill-range skips
  protected ports rather than aborting the whole batch.

You can protect by port, by range, or by process name — the last is useful when
the thing you care about moves around.

---

## Project config

`.openport.yml`, read from the working directory or any parent:

```yaml
name: shop
expected_ports:
  - 3000
  - 5432
aliases:
  web: 3000
  db: 5432
ranges:
  - name: workers
    from: 9000
    to: 9010
protected:
  - 5432
blueprint: web
```

Everything is optional. Create one from inside the tool with the
"Create project config" action.

---

## Scripting

The interface is the product; these subcommands exist for scripts and CI. They
call exactly the same service the interface does, so behaviour cannot drift.

| Command | Behaviour |
|---|---|
| `openport` | launch the interface |
| `openport --view <name>` | launch into `ports`, `actions`, `aliases`, `blueprints`, `config`, `history` or `workspace` |
| `openport ls` | list listening ports |
| `openport check <port>` | inspect one port |
| `openport kill <port>` | kill the listener (prompts first) |
| `openport kill-range <lo> <hi>` | kill every listener in a range |
| `openport free` | suggest a free port |
| `openport apply <blueprint>` | check live state against a blueprint |
| `openport version` | print the version |

Flags: `--force`/`-f` skips the confirmation prompt, `-9`/`--sigkill` sends
SIGKILL instead of SIGTERM, `--near` and `--count` tune `free`.

Exit codes are meaningful: `0` success, `1` failure (including "nothing was
killed" and "the blueprint has conflicts"), `2` usage error.

Two details that matter in a pipeline:

```bash
PORT=$(openport free --near 4000)     # bare number on stdout
openport apply web || exit 1          # non-zero when a port is taken
```

Without a terminal, a destructive command refuses rather than prompting:

```console
$ openport kill 3000 < /dev/null
refusing to run a destructive command without a terminal; pass --force
```

A cron job that forgot `--force` fails loudly instead of hanging on a prompt
nobody will ever answer.

---

## Free vs Pro

Everything above is free, works entirely offline, and makes no network calls on
any code path. No account, no telemetry. The default sync provider is a no-op —
the concrete remote client is not reachable from any core package, which is
enforced by `make arch-check` rather than by convention.

Pro adds the parts that only make sense with other people: cloud sync of your
aliases, groups and blueprints across machines; a team registry of who is
running which service where; drift detection against a shared blueprint; and
GitHub/Google sign-in from inside the interface.

Pro is enabled by setting `OPENPORT_LICENSE`. With a license present the sync
provider is a real REST client for the OpenPort cloud: sign-in exchanges the
license for a session, then push/pull/registry/drift talk to
`/v1/workspaces/{id}/…` at the endpoint. Point it at a self-hosted or test
deployment with `OPENPORT_ENDPOINT` (defaults to `https://api.openport.dev`).
Load the workspaces (`7`), pick the one to sync against (`s` on the Workspace
screen, or the "Select workspace" action), then push, pull or detect drift. Off
Pro, those actions stay visible with a `PRO` badge and explain themselves rather
than vanishing or breaking — and no Free code path ever opens a socket.

---

## Architecture

```
cmd/openport          entrypoint: builds the dependencies and injects them
internal/
  ports/              the core service — every capability, OS-agnostic
  platform/           the only code that talks to the OS, behind Provider
  store/              persistence: JSON under the XDG config dir, plus .openport.yml
  blueprint/          applying an environment template to live state
  model/              shared domain types
  sync/               the Pro seam: SyncProvider, with a no-op default
  cli/                the thin headless front-end
  tui/                the interface
```

Four invariants hold the shape together, and the first two are enforced by
`make arch-check`:

1. `internal/tui` never imports `internal/platform`. The UI depends on the
   service, the store and the sync provider — nothing else.
2. Every `lsof`, `ss` and `/proc` access lives behind `platform.Provider`.
   Linux parses `/proc` with the standard library and only falls back to `ss`
   if procfs is unreadable; macOS drives `lsof`. Neither the service nor the UI
   can tell which ran.
3. State changes only in `Update`. Commands call the service and return a
   message; they never write shared state.
4. Both front-ends call one `ports.Service`, so protection, history and
   enrichment cannot be bypassed by entering through a different door.

---

## Local development

```bash
git clone https://github.com/SinghAman21/openport
cd openport
make dev
```

`make dev` runs from source with `--debug`, and points state at `./.devstate`
instead of your real config directory. That isolation matters here more than in
most projects: this tool can kill processes, and you do not want an experiment
mid-refactor writing to the never-kill list you actually rely on.

```bash
make dev                      # TUI, isolated state, debug logging
make dev ARGS="--view config" # start on a particular view
make dev-cli ARGS="ls"        # a headless subcommand against dev state
make dev-listeners            # open dummy listeners so there is something to see
make dev-listeners-stop       # close them again
make dev-where                # print the dev state directory and its contents
make dev-reset                # wipe dev state (never touches your real config)
make help                     # list every target
```

A fresh checkout shows an empty Aliases view and no history, which is the least
useful version of the interface to develop against. `make dev-listeners` opens
four listeners on 45001–45004 — deliberately high ports that will not collide
with anything real — so the table, filtering, health checks and kill flow all
have something to act on. Any other process works just as well; there is
nothing special about those ports.

The `OPENPORT_CONFIG_DIR` environment variable is what does the isolation, and
it works standalone too:

```bash
OPENPORT_CONFIG_DIR=/tmp/openport-scratch go run ./cmd/openport
```

It exists because `XDG_CONFIG_HOME` has no effect on macOS, where Go resolves
config to `~/Library/Application Support`.

### Debugging the interface

A TUI cannot be inspected with print statements — stdout is the screen. Two
things help:

- **`ctrl+d`** appends the current model state to `./openport-debug.log`: the
  active route, terminal size, whether watch mode and overlays are live, and
  the full port listing as the model sees it. This is the fastest way to answer
  "why is this row not rendering".
- **Render tests.** `internal/tui/app_test.go` drives the root model with
  messages and asserts on the rendered frame at seven terminal sizes. Layout
  regressions fail the build rather than the user — that suite is what caught
  ANSI codes bleeding out of table cells, a table with a hard 97-cell minimum,
  and a panic when the column set changed on resize.

### The gate

```bash
make check      # gofmt + vet + arch-check + tests — run this before pushing
make test
make arch-check # the architecture invariants, checked against the import graph
make cross      # darwin, linux and windows binaries into ./dist
```

`make arch-check` is worth understanding rather than just running. It fails the
build if `internal/tui` acquires a direct import of `internal/platform`, if a
core package reaches for the concrete remote client, or if an `lsof`/`ss`/
`/proc` call appears outside the platform layer. Those are the three ways this
design degrades quietly, so they are checked mechanically instead of being left
to review.

Tests never touch the real OS or the network: the platform parsers run against
recorded `/proc` and `lsof` fixtures, and everything above them runs against
fake `Provider`, `Store` and `Service` implementations. The interface is tested
by driving the root model with messages and asserting on rendered frames, so
layout regressions and missing confirmations fail the build rather than the
user.
