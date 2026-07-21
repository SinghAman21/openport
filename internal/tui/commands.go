package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/blueprint"
	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/store"
	opsync "github.com/SinghAman21/openport/internal/sync"
	"github.com/SinghAman21/openport/internal/tui/msg"
)

// Timeouts for service calls made from the UI. They exist so a wedged OS call
// can never leave the interface waiting forever — the command returns an error
// message and the UI stays responsive.
const (
	listTimeout   = 5 * time.Second
	actionTimeout = 10 * time.Second
	probeTimeout  = 5 * time.Second
)

// Everything in this file is a tea.Cmd factory: it calls the service on Bubble
// Tea's command goroutine and returns a result message. Commands never touch
// model state — that is Update's job, and only Update's.

func listPortsCmd(svc ports.Service) tea.Cmd { return listPortsWith(svc, false) }

// backgroundListCmd is the ambient variant: its result is flagged so the root
// does not treat it as work the user is waiting on, which would flash the
// spinner every few seconds.
func backgroundListCmd(svc ports.Service) tea.Cmd { return listPortsWith(svc, true) }

func listPortsWith(svc ports.Service, background bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
		defer cancel()
		list, err := svc.List(ctx)
		return msg.PortsLoadedMsg{Ports: list, Err: err, Background: background}
	}
}

// AutoRefreshInterval is how often the ambient poll runs. It is slower than
// watch mode: watch is the explicit, indicator-lit "I am staring at this"
// mode, while this just keeps the interface from ever being meaningfully out
// of date.
const AutoRefreshInterval = 3 * time.Second

// autoRefreshCmd schedules the next ambient poll.
func autoRefreshCmd() tea.Cmd {
	return tea.Tick(AutoRefreshInterval, func(time.Time) tea.Msg {
		return msg.AutoRefreshMsg{}
	})
}

func checkPortCmd(svc ports.Service, port int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
		defer cancel()
		p, found, err := svc.Check(ctx, port)
		return msg.PortCheckedMsg{Port: p, Found: found, Err: err}
	}
}

func killPortCmd(svc ports.Service, port int, opts ports.KillOptions) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		res, err := svc.KillWith(ctx, port, opts)
		return msg.KilledMsg{Results: []ports.KillResult{res}, Err: err}
	}
}

func killPIDCmd(svc ports.Service, pid int, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		res, err := svc.KillPID(ctx, pid, force)
		return msg.KilledMsg{Results: []ports.KillResult{res}, Err: err}
	}
}

func killRangeCmd(svc ports.Service, lo, hi int, opts ports.KillOptions) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		res, err := svc.KillRangeWith(ctx, lo, hi, opts)
		return msg.KilledMsg{Results: res, Err: err}
	}
}

func suggestCmd(svc ports.Service, near, count int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		list, err := svc.SuggestFree(ctx, near, count)
		return msg.SuggestedMsg{Near: near, Ports: list, Err: err}
	}
}

func reserveCmd(svc ports.Service, port int, service string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		err := svc.Reserve(ctx, port, service)
		return msg.ReservedMsg{Port: port, Service: service, Err: err}
	}
}

func healthCmd(svc ports.Service, port int, kind ports.HealthKind) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		defer cancel()
		res, err := svc.Health(ctx, port, kind)
		return msg.HealthResultMsg{Port: port, Result: res, Err: err}
	}
}

// watchCmd reads exactly one listing from the watch channel. On receiving the
// resulting WatchTickMsg the root model issues this command again, which is
// what keeps watch mode ticking without ever blocking Update: the channel read
// happens on a command goroutine, so the UI stays interactive throughout.
func watchCmd(ch <-chan []ports.Port) tea.Cmd {
	return func() tea.Msg {
		list, ok := <-ch
		if !ok {
			return msg.WatchStoppedMsg{}
		}
		return msg.WatchTickMsg{Ports: list}
	}
}

// loadDataCmd reads every store-backed collection the views render, in one
// pass, so a refresh is a single message rather than seven.
func loadDataCmd(st store.Store, dir string) tea.Cmd {
	return func() tea.Msg {
		out := msg.DataLoadedMsg{}

		var err error
		if out.Aliases, err = st.Aliases(); err != nil {
			out.Err = err
		}
		if out.Groups, err = st.Groups(); err != nil && out.Err == nil {
			out.Err = err
		}
		if out.Blueprints, err = st.Blueprints(); err != nil && out.Err == nil {
			out.Err = err
		}
		if out.Protected, err = st.ProtectedPorts(); err != nil && out.Err == nil {
			out.Err = err
		}
		if out.History, err = st.History(200); err != nil && out.Err == nil {
			out.Err = err
		}
		if out.Logs, err = st.Logs(300); err != nil && out.Err == nil {
			out.Err = err
		}
		if out.Project, out.HasProject, err = st.ProjectConfig(dir); err != nil && out.Err == nil {
			out.Err = err
		}
		return out
	}
}

func saveAliasCmd(st store.Store, alias model.Alias) tea.Cmd {
	return func() tea.Msg {
		err := st.SaveAlias(alias)
		return msg.SavedMsg{What: fmt.Sprintf("alias %s → %d", alias.Name, alias.Port), Err: err}
	}
}

func deleteAliasCmd(st store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		err := st.DeleteAlias(name)
		return msg.SavedMsg{What: "deleted alias " + name, Err: err}
	}
}

func saveGroupCmd(st store.Store, group model.Group) tea.Cmd {
	return func() tea.Msg {
		err := st.SaveGroup(group)
		return msg.SavedMsg{What: "group " + group.Name, Err: err}
	}
}

func deleteGroupCmd(st store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		err := st.DeleteGroup(name)
		return msg.SavedMsg{What: "deleted group " + name, Err: err}
	}
}

func saveBlueprintCmd(st store.Store, bp model.Blueprint) tea.Cmd {
	return func() tea.Msg {
		err := st.SaveBlueprint(bp)
		return msg.SavedMsg{What: "blueprint " + bp.Name, Err: err}
	}
}

func deleteBlueprintCmd(st store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		err := st.DeleteBlueprint(name)
		return msg.SavedMsg{What: "deleted blueprint " + name, Err: err}
	}
}

// toggleProtectedCmd flips a port's place on the never-kill list and audits the
// change, since altering what OpenPort refuses to touch is itself a mutation
// worth recording.
func toggleProtectedCmd(st store.Store, port int) tea.Cmd {
	return func() tea.Msg {
		current, err := st.ProtectedPorts()
		if err != nil {
			return msg.SavedMsg{What: "protected ports", Err: err}
		}

		wasProtected, _ := current.IsProtected(port, "")
		updated := current.WithPort(port)
		verb := "protected"
		if wasProtected {
			updated = current.WithoutPort(port)
			verb = "unprotected"
		}

		if err := st.SaveProtectedPorts(updated); err != nil {
			return msg.SavedMsg{What: "protected ports", Err: err}
		}
		_ = st.AppendHistory(model.HistoryEntry{
			At:      time.Now(),
			Kind:    model.ActionProtect,
			Summary: fmt.Sprintf("%s port %d", verb, port),
			Port:    port,
			Success: true,
		})
		return msg.SavedMsg{What: fmt.Sprintf("%s port %d", verb, port)}
	}
}

func saveProjectConfigCmd(st store.Store, dir string, cfg model.ProjectConfig) tea.Cmd {
	return func() tea.Msg {
		err := st.SaveProjectConfig(dir, cfg)
		if err == nil {
			_ = st.AppendHistory(model.HistoryEntry{
				At:      time.Now(),
				Kind:    model.ActionConfig,
				Summary: "saved " + model.ProjectConfigFile,
				Detail:  dir,
				Success: true,
			})
		}
		return msg.SavedMsg{What: model.ProjectConfigFile, Err: err}
	}
}

func applyBlueprintCmd(svc ports.Service, st store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		results, err := blueprint.Apply(ctx, svc, st, name)
		return msg.BlueprintAppliedMsg{Name: name, Results: results, Err: err}
	}
}

// groupPortsCmd resolves a group name to its ports, then fans out to the right
// per-port operation.
func groupShutdownCmd(svc ports.Service, st store.Store, name string, force bool) tea.Cmd {
	return func() tea.Msg {
		list, err := groupPorts(st, name)
		if err != nil {
			return msg.KilledMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()

		results := make([]ports.KillResult, 0, len(list))
		for _, port := range list {
			res, err := svc.KillWith(ctx, port, ports.KillOptions{Force: force})
			if err != nil {
				return msg.KilledMsg{Results: results, Err: err}
			}
			results = append(results, res)
		}
		return msg.KilledMsg{Results: results}
	}
}

// groupHealthCmd probes every port in a group, emitting one message per port
// so badges appear as results arrive rather than all at the end.
func groupHealthCmd(svc ports.Service, st store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		list, err := groupPorts(st, name)
		if err != nil {
			return msg.ErrorMsg{Err: err}
		}
		cmds := make([]tea.Cmd, 0, len(list))
		for _, port := range list {
			cmds = append(cmds, healthCmd(svc, port, ports.HealthTCP))
		}
		return tea.Batch(cmds...)()
	}
}

func groupPorts(st store.Store, name string) ([]int, error) {
	groups, err := st.Groups()
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.Name == name {
			out := append([]int(nil), g.Ports...)
			sort.Ints(out)
			return out, nil
		}
	}
	return nil, fmt.Errorf("no group named %q", name)
}

// --- Pro commands --------------------------------------------------------
//
// Every one of these goes through the SyncProvider interface. With the default
// NoopProvider they fail fast with ErrProRequired and never open a socket,
// which is what makes the Free path provably offline.

func loginCmd(provider opsync.SyncProvider, oauth opsync.OAuthProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		session, err := provider.Login(ctx, oauth)
		return msg.LoggedInMsg{Session: session, Err: err}
	}
}

func workspacesCmd(provider opsync.SyncProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		list, err := provider.Workspaces(ctx)
		return msg.WorkspacesMsg{Workspaces: list, Err: err}
	}
}

func selectWorkspaceCmd(provider opsync.SyncProvider, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		err := provider.SelectWorkspace(ctx, id)
		return msg.SavedMsg{What: "workspace " + id, Err: err}
	}
}

func pushCmd(provider opsync.SyncProvider, st store.Store) tea.Cmd {
	return func() tea.Msg {
		snap, err := st.Snapshot()
		if err != nil {
			return msg.SyncedMsg{Direction: "push", Err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		return msg.SyncedMsg{Direction: "push", Err: provider.Push(ctx, snap)}
	}
}

func pullCmd(provider opsync.SyncProvider, st store.Store) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		snap, err := provider.Pull(ctx)
		if err != nil {
			return msg.SyncedMsg{Direction: "pull", Err: err}
		}
		return msg.SyncedMsg{Direction: "pull", Err: st.ApplySnapshot(snap)}
	}
}

func registryCmd(provider opsync.SyncProvider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		members, err := provider.Registry(ctx)
		return msg.RegistryMsg{Members: members, Err: err}
	}
}

func driftCmd(provider opsync.SyncProvider, st store.Store) tea.Cmd {
	return func() tea.Msg {
		snap, err := st.Snapshot()
		if err != nil {
			return msg.DriftMsg{Err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		drifts, err := provider.DetectDrift(ctx, snap)
		return msg.DriftMsg{Drifts: drifts, Err: err}
	}
}

// dumpStateCmd writes a state snapshot to ./openport-debug.log. Bound to
// ctrl+d so the current model can be inspected when a rendering problem is not
// reproducible from the screen alone.
func dumpStateCmd(dump string) tea.Cmd {
	return func() tea.Msg {
		const path = "openport-debug.log"
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return msg.Error("could not write " + path + ": " + err.Error())
		}
		defer f.Close()

		header := fmt.Sprintf("\n=== %s ===\n", time.Now().Format(time.RFC3339))
		if _, err := f.WriteString(header + dump + "\n"); err != nil {
			return msg.Error("could not write " + path + ": " + err.Error())
		}
		return msg.Success("State written to " + path)
	}
}
