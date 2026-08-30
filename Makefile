BINARY   := openport
PKG      := github.com/SinghAman21/openport
CMD      := ./cmd/openport
DIST     := dist

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

# Local development state. Kept inside the repo (and gitignored) so working on
# OpenPort never touches the aliases, groups and protected ports you actually
# depend on — a tool that can kill processes is not one you want experimenting
# against your real never-kill list.
DEVSTATE := $(CURDIR)/.devstate
DEVENV   := OPENPORT_CONFIG_DIR=$(DEVSTATE)

# Ports used by `make dev-listeners`. Deliberately high and obscure so they
# cannot collide with anything real you happen to be running.
DEV_PORTS := 45001 45002 45003 45004

.PHONY: all build run dev dev-cli dev-listeners dev-listeners-stop dev-reset dev-where \
        test vet fmt lint arch-check cross install clean check help

all: check build

## build: compile the binary for this platform
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## run: launch the TUI against your real config
run:
	go run -ldflags "$(LDFLAGS)" $(CMD)

# --- local development ------------------------------------------------------

## dev: launch the TUI in development mode (isolated state + debug logging)
dev:
	@mkdir -p $(DEVSTATE)
	@echo "state:  $(DEVSTATE)"
	@echo "debug:  $(CURDIR)/openport-debug.log  (ctrl+d dumps model state)"
	@echo
	$(DEVENV) go run -ldflags "$(LDFLAGS)" $(CMD) --debug $(ARGS)

## dev-cli: run a headless subcommand against the dev state, e.g. make dev-cli ARGS="ls"
dev-cli:
	@mkdir -p $(DEVSTATE)
	@$(DEVENV) go run -ldflags "$(LDFLAGS)" $(CMD) $(ARGS)

## dev-listeners: open dummy listeners so there is something to look at
dev-listeners:
	@command -v python3 >/dev/null 2>&1 || { \
		echo "python3 not found — open some listeners another way, any process will do"; exit 1; }
	@mkdir -p $(DEVSTATE)
	@for p in $(DEV_PORTS); do \
		(python3 -m http.server $$p --bind 127.0.0.1 >/dev/null 2>&1 & echo $$! >> $(DEVSTATE)/listeners.pid); \
	done
	@echo "listening on: $(DEV_PORTS)"
	@echo "stop them with: make dev-listeners-stop"

## dev-listeners-stop: close the dummy listeners
dev-listeners-stop:
	@if [ -f $(DEVSTATE)/listeners.pid ]; then \
		while read pid; do kill $$pid 2>/dev/null || true; done < $(DEVSTATE)/listeners.pid; \
		rm -f $(DEVSTATE)/listeners.pid; \
		echo "stopped"; \
	else echo "none running"; fi

## dev-where: print where dev state lives and what is in it
dev-where:
	@echo "$(DEVSTATE)"
	@ls -la $(DEVSTATE) 2>/dev/null || echo "(nothing yet — run make dev)"

## dev-reset: wipe development state (never touches your real config)
dev-reset:
	rm -rf $(DEVSTATE)
	@echo "development state cleared"

## test: run the whole suite (no test touches the real OS or network)
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format everything in place
fmt:
	gofmt -w .

## lint: fail if anything is unformatted, then vet
lint:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt'd:"; echo "$$unformatted"; exit 1; \
	fi
	@echo "gofmt: clean"
	go vet ./...

## check: the full gate — lint, architecture invariants, tests
check: lint arch-check test

## arch-check: enforce the architecture invariants from CLAUDE.md
arch-check:
	@echo "==> architecture invariants"
	@# 1. The UI layer must not reach the OS. Checked against the real import
	@#    graph rather than by grepping text, and against .Imports (which
	@#    excludes test files) because ports.Port embeds platform.Socket, so
	@#    building a fixture in a test necessarily names that type.
	@if go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/tui/... \
		| grep -F "$(PKG)/internal/platform"; then \
		echo "FAIL: internal/tui must not import internal/platform"; exit 1; \
	fi
	@echo "    ok: internal/tui does not import internal/platform"
	@# 2. Core code depends on the SyncProvider interface, never on the
	@#    concrete remote client.
	@if grep -rln "RemoteProvider" \
		--include="*.go" internal/ports internal/store internal/cli \
		internal/blueprint internal/tui internal/model 2>/dev/null; then \
		echo "FAIL: core packages must depend on sync.SyncProvider, not RemoteProvider"; exit 1; \
	fi
	@echo "    ok: no core package references RemoteProvider"
	@# 3. OS-specific mechanisms stay behind the Provider interface.
	@if grep -rlnE '"(lsof|ss)"|/proc/net|exec\.Command' \
		--include="*.go" internal/ports internal/store internal/tui \
		internal/cli internal/blueprint internal/model 2>/dev/null; then \
		echo "FAIL: lsof/ss//proc calls must live in internal/platform"; exit 1; \
	fi
	@echo "    ok: no OS-specific calls outside internal/platform"

## cross: build for the supported platforms
cross:
	@mkdir -p $(DIST)
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-arm64   $(CMD)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe $(CMD)
	@ls -lh $(DIST)

## install: install into GOBIN
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(CMD)

## clean: remove build artefacts
clean:
	rm -rf $(BINARY) $(DIST) openport-debug.log

## help: list the available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
