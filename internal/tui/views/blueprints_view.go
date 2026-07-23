package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/blueprint"
	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// serviceStatus is a blueprint service's live state, computed locally by
// comparing the blueprint against the current listing. This is the Free half
// of drift detection — no network involved.
type serviceStatus struct {
	service model.BlueprintService
	state   string // ok, free, conflict
	holder  string
	pid     int
}

// BlueprintsView lists environment templates and shows, per service, whether
// reality matches the blueprint.
type BlueprintsView struct {
	theme styles.Theme

	blueprints []model.Blueprint
	live       []ports.Port
	tbl        table.Model
	cols       []table.Column

	// applied holds the most recent apply result, shown in the detail pane.
	applied     map[string][]blueprint.Result
	lastApplied string
}

// NewBlueprintsView builds the blueprints screen.
func NewBlueprintsView(theme styles.Theme) *BlueprintsView {
	cols := []table.Column{
		{Title: "BLUEPRINT", Width: 22},
		{Title: "SERVICES", Width: 10},
		{Title: "STATUS", Width: 24},
		{Title: "DESCRIPTION", Width: 30},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	t.SetStyles(tableStyles(theme))

	return &BlueprintsView{
		theme:   theme,
		tbl:     t,
		cols:    cols,
		applied: map[string][]blueprint.Result{},
	}
}

// Title implements View.
func (v *BlueprintsView) Title() string { return "Blueprints" }

var blueprintKeys = struct {
	apply, newItem, edit, del key.Binding
}{
	apply:   key.NewBinding(key.WithKeys("a", "enter"), key.WithHelp("a", "apply")),
	newItem: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	del:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
}

// Help implements View.
func (v *BlueprintsView) Help() []key.Binding {
	return []key.Binding{blueprintKeys.apply, blueprintKeys.newItem, blueprintKeys.edit, blueprintKeys.del}
}

// editBlueprintMsg asks the root model to open a prefilled blueprint form. It
// lives here because only this view produces it, mirroring the alias and group
// edit intents.
type editBlueprintMsg struct{ Blueprint model.Blueprint }

// EditBlueprintRequest exposes the payload of an edit intent to the root model.
func EditBlueprintRequest(m tea.Msg) (model.Blueprint, bool) {
	e, ok := m.(editBlueprintMsg)
	return e.Blueprint, ok
}

// SelectionSummary feeds the status bar.
func (v *BlueprintsView) SelectionSummary() string {
	if bp, ok := v.selected(); ok {
		return bp.Name
	}
	return ""
}

func (v *BlueprintsView) selected() (model.Blueprint, bool) {
	i := v.tbl.Cursor()
	if i < 0 || i >= len(v.blueprints) {
		return model.Blueprint{}, false
	}
	return v.blueprints[i], true
}

// Update implements View.
func (v *BlueprintsView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.DataLoadedMsg:
		if m.Err == nil {
			v.blueprints = m.Blueprints
			v.rebuild()
		}
		return v, nil

	case msg.PortsLoadedMsg:
		if m.Err == nil {
			v.live = m.Ports
			v.rebuild()
		}
		return v, nil

	case msg.WatchTickMsg:
		v.live = m.Ports
		v.rebuild()
		return v, nil

	case msg.BlueprintAppliedMsg:
		if m.Err == nil {
			v.applied[m.Name] = m.Results
			v.lastApplied = m.Name
			v.rebuild()
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(m, blueprintKeys.apply):
			if bp, ok := v.selected(); ok {
				return v, emit(msg.ApplyBlueprintMsg{Name: bp.Name})
			}
			return v, nil
		case key.Matches(m, blueprintKeys.newItem):
			return v, emit(msg.RunActionMsg{ID: actions.BlueprintNew})
		case key.Matches(m, blueprintKeys.edit):
			if bp, ok := v.selected(); ok {
				return v, emit(editBlueprintMsg{Blueprint: bp})
			}
			return v, nil
		case key.Matches(m, blueprintKeys.del):
			if bp, ok := v.selected(); ok {
				return v, emit(msg.DeleteBlueprintMsg{Name: bp.Name})
			}
			return v, nil
		}
		t, cmd := v.tbl.Update(m)
		v.tbl = t
		return v, cmd
	}
	return v, nil
}

// statuses computes each service's live state for a blueprint.
func (v *BlueprintsView) statuses(bp model.Blueprint) []serviceStatus {
	byPort := make(map[int]ports.Port, len(v.live))
	for _, p := range v.live {
		// Prefer the row that actually resolved a PID.
		if existing, ok := byPort[p.Port]; ok && existing.PID > 0 {
			continue
		}
		byPort[p.Port] = p
	}

	out := make([]serviceStatus, 0, len(bp.Services))
	for _, svc := range bp.Services {
		st := serviceStatus{service: svc, state: blueprint.StatusFree}
		if live, ok := byPort[svc.Port]; ok {
			st.holder = live.Label()
			st.pid = live.PID
			// A service is "ok" when the thing on its port looks like it.
			// Shared with the apply path so the badge and the apply report
			// can never disagree.
			if blueprint.Matches(live.Alias, live.Process, svc.Name) {
				st.state = blueprint.StatusOK
			} else {
				st.state = blueprint.StatusConflict
			}
		}
		out = append(out, st)
	}
	return out
}

func (v *BlueprintsView) rebuild() {
	rows := make([]table.Row, 0, len(v.blueprints))
	for _, bp := range v.blueprints {
		var ok, conflict, free int
		for _, s := range v.statuses(bp) {
			switch s.state {
			case "ok":
				ok++
			case "conflict":
				conflict++
			default:
				free++
			}
		}

		// Plain text only — see the note on PortsView.row: styled table cells
		// get truncated mid-escape by bubbles/table.
		status := fmt.Sprintf("%s %d up", styles.GlyphOK, ok)
		if conflict > 0 {
			status += fmt.Sprintf("  %s %d clash", styles.GlyphDrift, conflict)
		}
		if free > 0 {
			status += fmt.Sprintf("  %d idle", free)
		}

		rows = append(rows, table.Row{
			bp.Name,
			strconv.Itoa(len(bp.Services)),
			status,
			bp.Description,
		})
	}
	setRows(&v.tbl, rows)
}

// View implements View.
func (v *BlueprintsView) View(width, height int) string {
	if len(v.blueprints) == 0 {
		return v.theme.Faint.Render(
			"No blueprints yet.\n\n" +
				"A blueprint describes the services an environment expects — names, ports\n" +
				"and health probes — so you can see at a glance what is up, what is down,\n" +
				"and what has taken a port it should not have.\n\n" +
				"Press n to create one.")
	}

	sideBySide := width >= detailPaneMinWidth
	tableWidth := width
	if sideBySide {
		tableWidth = width * 58 / 100
	}
	bodyHeight := height - 1
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	fitTable(&v.tbl, v.cols, tableWidth, bodyHeight)
	body := v.tbl.View()

	if !sideBySide {
		return body
	}
	detail := v.theme.Panel.
		Width(width - tableWidth - 4).
		Height(bodyHeight - 2).
		Render(v.detail(width - tableWidth - 8))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(tableWidth).Render(body),
		detail,
	)
}

// detail renders the selected blueprint's services and their live state.
func (v *BlueprintsView) detail(width int) string {
	bp, ok := v.selected()
	if !ok {
		return v.theme.Faint.Render("No selection.")
	}

	var b strings.Builder
	b.WriteString(v.theme.PanelTitle.Render(bp.Name) + "\n")
	if bp.Description != "" {
		b.WriteString(v.theme.Faint.Render(wrapText(bp.Description, width)) + "\n\n")
	}

	for _, s := range v.statuses(bp) {
		var glyph, detail string
		var style = v.theme.Faint

		switch s.state {
		case "ok":
			glyph, style = styles.GlyphOK, v.theme.Success
			detail = s.holder
			if s.pid > 0 {
				detail += " (pid " + strconv.Itoa(s.pid) + ")"
			}
		case "conflict":
			glyph, style = styles.GlyphDrift, v.theme.Warning
			detail = "held by " + s.holder
		default:
			glyph, style = styles.GlyphNone, v.theme.Faint
			detail = "not running"
		}

		line := fmt.Sprintf("%s %-14s %-6d %s",
			glyph, truncateRunes(s.service.Name, 14), s.service.Port, detail)
		b.WriteString(style.Render(truncateRunes(line, width)) + "\n")
	}

	if results, applied := v.applied[bp.Name]; applied {
		b.WriteString("\n" + v.theme.Faint.Render("LAST APPLY") + "\n")
		for _, r := range results {
			if r.Suggested > 0 {
				b.WriteString(v.theme.Accent.Render(fmt.Sprintf(
					"%s → try %d", r.Service, r.Suggested)) + "\n")
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// truncateRunes clips to n runes with an ellipsis, for fixed-width columns.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
