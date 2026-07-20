// Package cli is the thin scripting front-end. It parses arguments and calls
// the same ports.Service the TUI does — there is no capability here that the
// interface lacks, and no logic here that the interface duplicates.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/x/term"

	"github.com/SinghAman21/openport/internal/blueprint"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/store"
	"github.com/SinghAman21/openport/internal/version"
)

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

const cmdTimeout = 15 * time.Second

// Env bundles the streams a run reads and writes, so tests can drive the CLI
// without touching the real stdio.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Interactive reports whether prompting makes sense. When false, a
	// destructive command must be given --force rather than silently
	// proceeding or blocking a script forever.
	Interactive bool
}

// DefaultEnv wires the real process streams.
func DefaultEnv() Env {
	// A real TTY check, not a character-device check: /dev/null is a
	// character device, so the cheaper test would happily prompt a cron job
	// and then read EOF — exactly the case the guard exists to catch.
	return Env{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Interactive: term.IsTerminal(os.Stdin.Fd()),
	}
}

// IsSubcommand reports whether arg names a headless subcommand, which is how
// main decides between launching the TUI and running a one-shot.
func IsSubcommand(arg string) bool {
	switch arg {
	case "ls", "list", "check", "kill", "kill-range", "free", "apply",
		"version", "help":
		return true
	}
	return false
}

// Run executes a subcommand and returns the process exit code.
func Run(args []string, svc ports.Service, st store.Store, env Env) int {
	if len(args) == 0 {
		usage(env.Stderr)
		return ExitUsage
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "ls", "list":
		return runList(ctx, svc, env)
	case "check":
		return runCheck(ctx, svc, env, rest)
	case "kill":
		return runKill(ctx, svc, env, rest)
	case "kill-range":
		return runKillRange(ctx, svc, env, rest)
	case "free":
		return runFree(ctx, svc, env, rest)
	case "apply":
		return runApply(ctx, svc, st, env, rest)
	case "version":
		fmt.Fprintln(env.Stdout, version.String())
		return ExitOK
	case "help":
		usage(env.Stdout)
		return ExitOK
	default:
		fmt.Fprintf(env.Stderr, "unknown command %q\n\n", cmd)
		usage(env.Stderr)
		return ExitUsage
	}
}

// --- subcommands ---------------------------------------------------------

func runList(ctx context.Context, svc ports.Service, env Env) int {
	list, err := svc.List(ctx)
	if err != nil {
		return fail(env, "could not list ports: %v", err)
	}
	if len(list) == 0 {
		fmt.Fprintln(env.Stdout, "Nothing is listening.")
		return ExitOK
	}

	w := tabwriter.NewWriter(env.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PORT\tPROTO\tPID\tPROCESS\tALIAS\tSTATE\t")
	for _, p := range list {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Port, p.Protocol, dashInt(p.PID), dash(p.Process),
			dash(p.Alias), dash(p.State), protectedMark(p.Protected))
	}
	return flush(w, env)
}

func runCheck(ctx context.Context, svc ports.Service, env Env, args []string) int {
	if len(args) < 1 {
		return fail(env, "usage: openport check <port>")
	}
	port, err := parsePort(args[0])
	if err != nil {
		return fail(env, "%v", err)
	}

	p, found, err := svc.Check(ctx, port)
	if err != nil {
		return fail(env, "could not check port %d: %v", port, err)
	}
	if !found {
		fmt.Fprintf(env.Stdout, "Port %d is free.\n", port)
		return ExitOK
	}

	w := tabwriter.NewWriter(env.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Port\t%d\n", p.Port)
	fmt.Fprintf(w, "Process\t%s\n", dash(p.Process))
	fmt.Fprintf(w, "PID\t%s\n", dashInt(p.PID))
	fmt.Fprintf(w, "Protocol\t%s\n", p.Protocol)
	fmt.Fprintf(w, "Address\t%s\n", dash(p.Addr))
	fmt.Fprintf(w, "State\t%s\n", dash(p.State))
	if p.Alias != "" {
		fmt.Fprintf(w, "Alias\t%s\n", p.Alias)
	}
	if len(p.Groups) > 0 {
		fmt.Fprintf(w, "Groups\t%s\n", strings.Join(p.Groups, ", "))
	}
	if p.Protected {
		fmt.Fprintf(w, "Protected\t%s\n", p.Reason)
	}
	if p.Command != "" {
		fmt.Fprintf(w, "Command\t%s\n", p.Command)
	}
	return flush(w, env)
}

func runKill(ctx context.Context, svc ports.Service, env Env, args []string) int {
	flags, rest := parseFlags(args)
	if len(rest) < 1 {
		return fail(env, "usage: openport kill <port> [--force] [-9]")
	}
	port, err := parsePort(rest[0])
	if err != nil {
		return fail(env, "%v", err)
	}

	if !flags.force {
		ok, code := confirm(env, fmt.Sprintf("Kill whatever is listening on port %d?", port))
		if !ok {
			return code
		}
	}

	res, err := svc.Kill(ctx, port, flags.sigkill)
	if err != nil {
		return fail(env, "could not kill port %d: %v", port, err)
	}
	return reportKills(env, []ports.KillResult{res})
}

func runKillRange(ctx context.Context, svc ports.Service, env Env, args []string) int {
	flags, rest := parseFlags(args)
	if len(rest) < 2 {
		return fail(env, "usage: openport kill-range <lo> <hi> [--force] [-9]")
	}
	lo, err := parsePort(rest[0])
	if err != nil {
		return fail(env, "%v", err)
	}
	hi, err := parsePort(rest[1])
	if err != nil {
		return fail(env, "%v", err)
	}

	if !flags.force {
		ok, code := confirm(env,
			fmt.Sprintf("Kill every listener between %d and %d?", lo, hi))
		if !ok {
			return code
		}
	}

	results, err := svc.KillRange(ctx, lo, hi, flags.sigkill)
	if err != nil {
		return fail(env, "could not kill range: %v", err)
	}
	if len(results) == 0 {
		fmt.Fprintf(env.Stdout, "Nothing listening between %d and %d.\n", lo, hi)
		return ExitOK
	}
	return reportKills(env, results)
}

func runFree(ctx context.Context, svc ports.Service, env Env, args []string) int {
	flags, _ := parseFlags(args)

	near := flags.near
	if near == 0 {
		near = ports.DefaultSuggestNear
	}
	count := flags.count
	if count == 0 {
		count = 1
	}

	list, err := svc.SuggestFree(ctx, near, count)
	if err != nil {
		return fail(env, "%v", err)
	}
	for _, p := range list {
		fmt.Fprintln(env.Stdout, p)
	}
	return ExitOK
}

func runApply(ctx context.Context, svc ports.Service, st store.Store, env Env, args []string) int {
	if len(args) < 1 {
		return fail(env, "usage: openport apply <blueprint>")
	}

	results, err := blueprint.Apply(ctx, svc, st, args[0])
	if err != nil {
		return fail(env, "%v", err)
	}

	w := tabwriter.NewWriter(env.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tPORT\tSTATUS\tDETAIL\t")
	conflicts := 0
	for _, r := range results {
		detail := r.Detail
		if r.Suggested > 0 {
			detail += fmt.Sprintf(" (try %d)", r.Suggested)
		}
		if r.Status == blueprint.StatusConflict {
			conflicts++
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t\n", r.Service, r.Wanted, r.Status, detail)
	}
	if err := w.Flush(); err != nil {
		return fail(env, "%v", err)
	}
	// A conflict is a real failure for a script gating a dev-server start on
	// this command, so it earns a non-zero exit.
	if conflicts > 0 {
		return ExitError
	}
	return ExitOK
}

// --- helpers -------------------------------------------------------------

// flags is the small set of options the headless commands accept.
type flags struct {
	// force skips the confirmation prompt.
	force bool
	// sigkill sends SIGKILL instead of SIGTERM.
	sigkill bool
	near    int
	count   int
}

// parseFlags pulls recognised flags out of args and returns the positional
// remainder.
func parseFlags(args []string) (flags, []string) {
	var f flags
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--force" || arg == "-f":
			f.force = true
		case arg == "-9" || arg == "--sigkill":
			f.sigkill = true
		case arg == "--near" && i+1 < len(args):
			i++
			f.near, _ = strconv.Atoi(args[i])
		case strings.HasPrefix(arg, "--near="):
			f.near, _ = strconv.Atoi(strings.TrimPrefix(arg, "--near="))
		case arg == "--count" && i+1 < len(args):
			i++
			f.count, _ = strconv.Atoi(args[i])
		case strings.HasPrefix(arg, "--count="):
			f.count, _ = strconv.Atoi(strings.TrimPrefix(arg, "--count="))
		default:
			rest = append(rest, arg)
		}
	}
	return f, rest
}

func parsePort(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%q is not a port number", s)
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("port %d out of range (1-65535)", n)
	}
	return n, nil
}

// confirm prompts before a destructive action. In a non-interactive context it
// refuses rather than prompting, so a script that forgot --force fails loudly
// instead of hanging.
func confirm(env Env, question string) (bool, int) {
	if !env.Interactive {
		fmt.Fprintln(env.Stderr,
			"refusing to run a destructive command without a terminal; pass --force")
		return false, ExitUsage
	}

	fmt.Fprintf(env.Stdout, "%s [y/N] ", question)
	line, err := bufio.NewReader(env.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false, ExitError
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, ExitOK
	default:
		fmt.Fprintln(env.Stdout, "Cancelled.")
		return false, ExitOK
	}
}

// reportKills prints per-port outcomes and picks the exit code: skipped ports
// are reported but do not fail the command, since the protection guard working
// as designed is not an error.
func reportKills(env Env, results []ports.KillResult) int {
	code := ExitOK
	killed := 0

	for _, r := range results {
		switch {
		case r.Skipped:
			fmt.Fprintf(env.Stderr, "skipped %d: %s\n", r.Port, r.Reason)
		case r.Err != nil:
			fmt.Fprintf(env.Stderr, "failed %d: %v\n", r.Port, r.Err)
			code = ExitError
		default:
			killed++
			signal := "SIGTERM"
			if r.Forced {
				signal = "SIGKILL"
			}
			fmt.Fprintf(env.Stdout, "killed %s (pid %d) on port %d with %s\n",
				dash(r.Process), r.PID, r.Port, signal)
		}
	}

	// Nothing killed and nothing failed means everything was skipped.
	if killed == 0 && code == ExitOK {
		code = ExitError
	}
	return code
}

func fail(env Env, format string, args ...any) int {
	fmt.Fprintf(env.Stderr, format+"\n", args...)
	return ExitError
}

func flush(w *tabwriter.Writer, env Env) int {
	if err := w.Flush(); err != nil {
		return fail(env, "%v", err)
	}
	return ExitOK
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func dashInt(n int) string {
	if n <= 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

func protectedMark(p bool) string {
	if p {
		return "protected"
	}
	return ""
}

func usage(w io.Writer) {
	fmt.Fprint(w, `openport — port & process cockpit

Run with no arguments to launch the interactive interface, which is where
every capability lives. These subcommands exist for scripting.

USAGE
  openport                          launch the TUI
  openport --view <name>            launch into a view
                                    (ports|actions|aliases|blueprints|config|history|workspace)

  openport ls                       list listening ports
  openport check <port>             inspect one port
  openport kill <port>              kill the listener (prompts first)
  openport kill-range <lo> <hi>     kill every listener in a range
  openport free                     suggest a free port
  openport apply <blueprint>        check live state against a blueprint
  openport version                  print the version
  openport help                     print this message

FLAGS
  --force, -f        skip the confirmation prompt (required when not on a terminal)
  -9, --sigkill      send SIGKILL instead of SIGTERM
  --near <port>      where "free" starts searching (default 3000)
  --count <n>        how many free ports to suggest (default 1)

NOTES
  Protected ports are never killed from the command line. Manage the
  never-kill list in the interface: openport --view config.

  Every mutating command is recorded in the action history.
`)
}
