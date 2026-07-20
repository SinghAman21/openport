// Command openport launches the port & process cockpit.
//
// Bare invocation opens the TUI, which is the primary interface. Subcommands
// route to the same ports.Service for scripting. Dependencies are constructed
// here and injected — there is no global state anywhere below this file.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/cli"
	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/platform"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/store"
	opsync "github.com/SinghAman21/openport/internal/sync"
	"github.com/SinghAman21/openport/internal/tui"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/version"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	// Handled before anything is constructed so they work even if the store
	// or the platform provider cannot be opened.
	for _, arg := range args {
		switch arg {
		case "--version", "-v":
			fmt.Println(version.String())
			return cli.ExitOK
		case "--help", "-h":
			cli.Run([]string{"help"}, nil, nil, cli.DefaultEnv())
			return cli.ExitOK
		}
	}

	st, err := store.NewFileStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "openport: could not open config store: %v\n", err)
		return cli.ExitError
	}

	// The platform layer reports fallbacks and permission problems through
	// this sink. Routing it into the store's log is what puts OS-level
	// diagnostics in the Logs view without the service or UI ever learning
	// which implementation ran.
	logSink := func(level, message string) {
		_ = st.AppendLog(model.LogEntry{
			At:      time.Now(),
			Level:   model.LogLevel(level),
			Source:  "platform",
			Message: message,
		})
	}

	prov, err := platform.Detect(logSink)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openport: %v\n", err)
		return cli.ExitError
	}

	svc := ports.NewService(prov, st)
	// NoopProvider unless a license is present. Nothing on the Free path can
	// reach the network, because nothing on the Free path holds a client.
	provider := opsync.Detect(os.Getenv("OPENPORT_LICENSE"))

	if len(args) > 0 && cli.IsSubcommand(args[0]) {
		return cli.Run(args, svc, st, cli.DefaultEnv())
	}

	route, debug, err := parseTUIArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openport: %v\n\n", err)
		cli.Run([]string{"help"}, nil, nil, cli.DefaultEnv())
		return cli.ExitUsage
	}
	if debug {
		writeDebugBanner()
	}

	program := tea.NewProgram(
		tui.New(svc, st, provider, route),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "openport: %v\n", err)
		return cli.ExitError
	}
	return cli.ExitOK
}

// parseTUIArgs reads the flags that only apply to the interactive front-end.
func parseTUIArgs(args []string) (msg.Route, bool, error) {
	route := msg.RoutePorts
	debug := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--debug":
			debug = true

		case arg == "--view" && i+1 < len(args):
			i++
			r, err := routeByName(args[i])
			if err != nil {
				return route, debug, err
			}
			route = r

		case strings.HasPrefix(arg, "--view="):
			r, err := routeByName(strings.TrimPrefix(arg, "--view="))
			if err != nil {
				return route, debug, err
			}
			route = r

		default:
			return route, debug, fmt.Errorf("unknown option %q", arg)
		}
	}
	return route, debug, nil
}

func routeByName(name string) (msg.Route, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ports":
		return msg.RoutePorts, nil
	case "actions":
		return msg.RouteActions, nil
	case "aliases", "groups":
		return msg.RouteAliases, nil
	case "blueprints":
		return msg.RouteBlueprints, nil
	case "config":
		return msg.RouteConfig, nil
	case "history", "logs":
		return msg.RouteHistory, nil
	case "workspace":
		return msg.RouteWorkspace, nil
	default:
		return msg.RoutePorts, fmt.Errorf("unknown view %q", name)
	}
}

// writeDebugBanner opens the debug log at startup so ctrl+d dumps have context
// about which build and environment produced them.
func writeDebugBanner() {
	const path = "openport-debug.log"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openport: could not open %s: %v\n", path, err)
		return
	}
	defer f.Close()

	wd, _ := os.Getwd()
	fmt.Fprintf(f, "\n=== openport start %s ===\n%s\ncwd=%s\npro=%v\n",
		time.Now().Format(time.RFC3339), version.String(), wd,
		os.Getenv("OPENPORT_LICENSE") != "",
	)
	fmt.Fprintf(os.Stderr, "openport: debug logging to ./%s (ctrl+d dumps state)\n", path)
}
