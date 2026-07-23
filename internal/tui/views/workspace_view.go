package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	opsync "github.com/SinghAman21/openport/internal/sync"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// WorkspaceView is the Pro screen: sign-in, workspace selection, the team
// registry of who runs what, and drift against the shared environment.
//
// Off Pro it renders an explanation of what the tier adds rather than a broken
// or hidden screen — the feature stays discoverable, it just is not usable.
type WorkspaceView struct {
	theme styles.Theme
	isPro bool

	session    opsync.Session
	workspaces []opsync.Workspace
	members    []opsync.Member
	drifts     []opsync.Drift

	tbl  table.Model
	cols []table.Column
	err  error
}

// NewWorkspaceView builds the Pro screen.
func NewWorkspaceView(theme styles.Theme, isPro bool) *WorkspaceView {
	cols := []table.Column{
		{Title: "MEMBER", Width: 20},
		{Title: "DEVICE", Width: 16},
		{Title: "SERVICE", Width: 20},
		{Title: "PORT", Width: 7},
		{Title: "STATUS", Width: 10},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	t.SetStyles(tableStyles(theme))
	return &WorkspaceView{theme: theme, isPro: isPro, tbl: t, cols: cols}
}

// Title implements View.
func (v *WorkspaceView) Title() string { return "Workspace" }

// SetPro updates the gate.
func (v *WorkspaceView) SetPro(isPro bool) { v.isPro = isPro }

var workspaceKeys = struct {
	login, workspaces, selectWS, refresh, push, pull, drift key.Binding
}{
	login:      key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "sign in")),
	workspaces: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "load workspaces")),
	selectWS:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "select workspace")),
	refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh registry")),
	push:       key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "push")),
	pull:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "pull")),
	drift:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "detect drift")),
}

// Help implements View.
func (v *WorkspaceView) Help() []key.Binding {
	return []key.Binding{
		workspaceKeys.login, workspaceKeys.workspaces, workspaceKeys.selectWS,
		workspaceKeys.refresh, workspaceKeys.push, workspaceKeys.pull, workspaceKeys.drift,
	}
}

// SelectionSummary feeds the status bar.
func (v *WorkspaceView) SelectionSummary() string {
	if !v.isPro {
		return "Pro required"
	}
	if v.session.Email != "" {
		return v.session.Email
	}
	return "not signed in"
}

// Update implements View.
func (v *WorkspaceView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.RegistryMsg:
		v.err = m.Err
		if m.Err == nil {
			v.members = m.Members
			v.rebuild()
		}
		return v, nil

	case msg.WorkspacesMsg:
		v.err = m.Err
		if m.Err == nil {
			v.workspaces = m.Workspaces
		}
		return v, nil

	case msg.LoggedInMsg:
		v.err = m.Err
		if m.Err == nil {
			v.session = m.Session
		}
		return v, nil

	case msg.DriftMsg:
		v.err = m.Err
		if m.Err == nil {
			v.drifts = m.Drifts
		}
		return v, nil

	case tea.KeyMsg:
		if !v.isPro {
			return v, nil
		}
		switch {
		case key.Matches(m, workspaceKeys.login):
			return v, emit(msg.RunActionMsg{ID: actions.ProLogin})
		case key.Matches(m, workspaceKeys.workspaces):
			return v, emit(msg.RunActionMsg{ID: actions.ProWorkspaces})
		case key.Matches(m, workspaceKeys.selectWS):
			return v, emit(msg.RunActionMsg{ID: actions.ProSelectWorkspace})
		case key.Matches(m, workspaceKeys.refresh):
			return v, emit(msg.LoadRegistryMsg{})
		case key.Matches(m, workspaceKeys.push):
			return v, emit(msg.SyncPushMsg{})
		case key.Matches(m, workspaceKeys.pull):
			return v, emit(msg.SyncPullMsg{})
		case key.Matches(m, workspaceKeys.drift):
			return v, emit(msg.DetectDriftMsg{})
		}
		t, cmd := v.tbl.Update(m)
		v.tbl = t
		return v, cmd
	}
	return v, nil
}

func (v *WorkspaceView) rebuild() {
	var rows []table.Row
	for _, m := range v.members {
		if len(m.Services) == 0 {
			rows = append(rows, table.Row{m.Name, m.Device, "—", "—", "idle"})
			continue
		}
		for i, s := range m.Services {
			name, device := m.Name, m.Device
			if i > 0 {
				name, device = "", "" // group a member's services visually
			}
			// Plain text only — see the note on PortsView.row.
			status := s.Status
			switch s.Status {
			case "up":
				status = styles.GlyphOK + " up"
			case "down":
				status = styles.GlyphDown + " down"
			}
			rows = append(rows, table.Row{
				name, device, s.Name, strconv.Itoa(s.Port), status,
			})
		}
	}
	setRows(&v.tbl, rows)
}

// View implements View.
func (v *WorkspaceView) View(width, height int) string {
	if !v.isPro {
		return v.upsell(width)
	}

	var b strings.Builder

	if v.session.Email == "" {
		b.WriteString(v.theme.Warning.Render("Not signed in.") + "  ")
		b.WriteString(v.theme.Faint.Render("Press L to connect with GitHub or Google.") + "\n\n")
	} else {
		b.WriteString(v.theme.Success.Render(styles.GlyphOK+" "+v.session.Email) + "\n")
		if len(v.workspaces) > 0 {
			names := make([]string, 0, len(v.workspaces))
			for _, w := range v.workspaces {
				names = append(names, w.Name)
			}
			b.WriteString(v.theme.Faint.Render("Workspaces: "+strings.Join(names, ", ")) + "\n")
		}
		b.WriteString("\n")
	}

	if v.err != nil {
		b.WriteString(v.theme.Danger.Render(v.err.Error()) + "\n\n")
	}

	if len(v.drifts) > 0 {
		b.WriteString(v.theme.Bold.Render("Drift") + "\n")
		for _, d := range v.drifts {
			b.WriteString(v.theme.Warning.Render(fmt.Sprintf("  %s %s: %s",
				styles.GlyphDrift, d.Service, driftDetail(d))) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(v.theme.Bold.Render("Team registry") + "\n")
	if len(v.members) == 0 {
		b.WriteString(v.theme.Faint.Render("  No members loaded. Press r to refresh."))
		return b.String()
	}

	bodyHeight := height - strings.Count(b.String(), "\n") - 2
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	fitTable(&v.tbl, v.cols, width, bodyHeight)
	b.WriteString(v.tbl.View())
	return b.String()
}

// driftDetail renders a drift record as a sentence.
func driftDetail(d opsync.Drift) string {
	switch d.Kind {
	case opsync.DriftMissing:
		return fmt.Sprintf("expected on %d, not running", d.Expected)
	case opsync.DriftUnexpected:
		return fmt.Sprintf("running on %d, not in the blueprint", d.Actual)
	case opsync.DriftPortMismatch:
		return fmt.Sprintf("expected %d, found %d", d.Expected, d.Actual)
	case opsync.DriftConflict:
		return fmt.Sprintf("port %d also claimed by %s", d.Actual, d.Owner)
	default:
		return d.Detail
	}
}

// upsell explains what Pro adds. Never a broken screen.
func (v *WorkspaceView) upsell(width int) string {
	var b strings.Builder
	b.WriteString(v.theme.BadgePro.Render("PRO") + "  " +
		v.theme.Title.Render("Team workspaces") + "\n\n")
	b.WriteString(v.theme.Body.Render(wrapText(
		"Everything OpenPort does locally works offline, forever, with no account. "+
			"Pro adds the parts that only make sense with other people:", min(width, 72))) + "\n\n")

	features := [][2]string{
		{"Cloud sync", "Your aliases, groups and blueprints on every machine you work from."},
		{"Team registry", "See which teammate is running which service, on which port."},
		{"Drift detection", "Get told when your environment diverges from the team blueprint."},
		{"Shared blueprints", "One environment definition the whole team applies."},
	}
	for _, f := range features {
		b.WriteString("  " + v.theme.Accent.Render(styles.GlyphArrow+" "+f[0]) + "\n")
		b.WriteString("    " + v.theme.Faint.Render(wrapText(f[1], min(width-4, 68))) + "\n\n")
	}

	b.WriteString(v.theme.Faint.Render(
		"Set OPENPORT_LICENSE to enable. No Free feature makes a network call."))
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
