<p align="center">
  <img src="web/public/openport-mark.svg" width="120" alt="OpenPort logo">
</p>

<h1 align="center">OpenPort</h1>

<p align="center">
  A keyboard-first cockpit for the ports and processes powering local development.
</p>

<p align="center">
  <a href="https://openport.singhaman.me">Website</a> ·
  <a href="#install">Install</a> ·
  <a href="#usage">Usage</a> ·
  <a href="https://github.com/SinghAman21/openport/releases">Releases</a> ·
  <a href="LICENSE">MIT License</a>
</p>

---

`EADDRINUSE` should not send you hunting through shell commands. OpenPort gives you a clear, interactive view of every listening port, the process behind it, and the safe actions you can take.

Launch it with one command:

```bash
openport
```

OpenPort is built in Go with the Charm stack and supports macOS, Linux, and Windows.

## Highlights

- **See what is listening** — inspect ports, protocols, PIDs, processes, addresses, and command lines in one live table.
- **Act safely** — stop processes by port, PID, range, or group; destructive actions require confirmation and protected ports are guarded.
- **Stay organized** — name ports with aliases, collect them into groups, and define expected services with blueprints.
- **Find a free port** — discover ports that are available to bind now, then reserve them for a service.
- **Keep context** — use health checks, project configuration, watch mode, action history, and diagnostic logs without leaving the terminal.
- **Work your way** — every capability is available from the TUI’s action menu and command palette; focused commands are also available for scripts and CI.

## Install

### Linux

```bash
curl -fsSL https://openport.singhaman.me/install.sh | sh
```

The installer selects the appropriate `amd64` or `arm64` release and installs it to `~/.local/bin`.

### Windows PowerShell

```powershell
irm https://openport.singhaman.me/install.ps1 | iex
```

The installer adds OpenPort to a user-local bin directory and updates your user `PATH`. Open a new PowerShell session when it completes.

### Go or source

Requires Go 1.25 or newer.

```bash
go install github.com/SinghAman21/openport/cmd/openport@latest
```

Or build a checkout:

```bash
git clone https://github.com/SinghAman21/openport.git
cd openport
make build
./openport
```

## Usage

Run `openport` with no arguments to start the interactive TUI. Use the arrows or `j`/`k` to move, `enter` to select, and `?` to see contextual help.

| Key | Action |
| --- | --- |
| `:` or `ctrl+k` | Open the command palette |
| `1`–`7` | Switch views |
| `r` | Refresh the active view |
| `w` | Toggle live watch mode in Ports |
| `x` / `X` | Stop / force-stop the selected process |
| `a` | Add an alias to the selected port |
| `p` | Protect or unprotect the selected port |
| `f` | Find a free port |
| `esc` | Close the current overlay or go back |
| `q` or `ctrl+c` | Quit |

### Scripting

The headless commands use the same service layer as the TUI:

```bash
openport ls
openport check 3000
openport free --near 4000
openport kill 3000
openport apply web
```

Run `openport help` for the complete command reference. Destructive commands ask for confirmation; pass `--force` only when running non-interactively.

## Project configuration

Place an `.openport.yml` in a project directory (or any parent directory) to describe the services you expect:

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

All fields are optional. OpenPort discovers this file automatically and can manage it from the Config view.

## Development

```bash
make dev        # run with isolated local state and debug logging
make test       # run the test suite
make check      # format, vet, architecture checks, and tests
make cross      # build macOS, Linux, and Windows binaries into dist/
```

`make dev` stores its state in `.devstate`, leaving your normal configuration untouched. To populate the interface with harmless local listeners, run `make dev-listeners` and clean them up with `make dev-listeners-stop`.

## Privacy and availability

OpenPort’s core feature set works locally and offline. It does not require an account, and the default configuration makes no network calls. Optional Pro features are additive and enabled only when `OPENPORT_LICENSE` is set.

## License

OpenPort is released under the [MIT License](LICENSE).
