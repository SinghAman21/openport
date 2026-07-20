# OpenPort — standing guardrails

> Mirrored from `openport.md` §2 and §14 so these survive context compaction.
> Re-read before starting each phase.

## Architecture rules (invariants — violating any is a bug)

1. `internal/tui` **NEVER** imports `internal/platform`. The UI depends only on
   `ports.Service` and `store.Store` (and, for Pro, `sync.SyncProvider`).
2. All OS-specific code lives in `internal/platform` behind the `Provider`
   interface, guarded by `//go:build` tags. No `lsof`/`ss`//proc calls anywhere
   else.
3. On Linux, prefer std-lib `/proc` parsing; shell out to `ss` only as a
   fallback. On macOS, use `lsof`. The service and UI must not know which ran.
4. **State mutations happen ONLY in `Update`.** `tea.Cmd` functions are pure
   dispatch — they call the service and return a message; they never write
   shared state.
5. Every **mutating action** (kill, kill-range, apply blueprint, reserve port)
   appends a `HistoryEntry` via the store.
6. Every **destructive action** (kill, kill-range, group shutdown) requires a
   confirmation modal, and must respect the **protected ports** list.
7. **Pro/cloud code depends on `sync.SyncProvider`; the default is
   `NoopProvider`.** The entire offline path must work with zero network. No
   core feature may import a concrete remote client.
8. Pro features are **additive** — extra views + extra palette entries — never a
   different interaction paradigm. Gate their UI on `provider.IsPro()`.
9. No global mutable state. Dependencies (`Service`, `Store`, `SyncProvider`)
   are constructed in `main` and injected.

## Definition of done

**Per change / phase:**
- `go build ./...` clean
- `go vet ./...` clean
- `gofmt -l .` prints nothing
- `go test ./...` green; new logic has table-driven tests using a fake
  `Provider`/`Store` (**tests never touch the real OS or network**)

**Whole project:**
- `openport` launches the TUI; every feature in §8 of the spec is reachable via
  menu **and** palette; destructive actions confirm and respect protected ports.
- `internal/tui` imports neither `internal/platform` nor any concrete remote
  client (enforced by `make arch-check`).
- Offline path works with `OPENPORT_LICENSE` unset (NoopProvider); no network
  calls on any Free path.
- Watch mode refreshes live while remaining fully interactive; selection
  persists.
- Builds for `GOOS=darwin` and `GOOS=linux` (`make cross`).
- README documents install, key bindings, palette, config format, and the Free
  vs Pro split.

## Two failure modes to actively avoid

1. Leaking `lsof`/`ss`//proc calls out of `internal/platform` because it's the
   fast path. Keep them behind `Provider`.
2. Coupling core features to a concrete remote client. Depend on
   `sync.SyncProvider`; default to `NoopProvider`.
