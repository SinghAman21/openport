package views

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// historyTab selects which pane of the view is showing.
type historyTab int

const (
	tabHistory historyTab = iota
	tabLogs
)

// timeFormat is the timestamp format used across both panes.
const timeFormat = "15:04:05"

// HistoryView is the audit trail: every mutating action OpenPort has taken,
// plus the internal log of OS interactions that did not go to plan.
type HistoryView struct {
	theme styles.Theme
	tab   historyTab

	history []model.HistoryEntry
	logs    []model.LogEntry

	tbl viewport.Model
	// historyTable renders the structured audit entries.
	historyTable table.Model
	historyCols  []table.Column
	ready        bool
}

// NewHistoryView builds the history screen.
func NewHistoryView(theme styles.Theme) *HistoryView {
	cols := []table.Column{
		{Title: "TIME", Width: 9},
		{Title: "ACTION", Width: 15},
		{Title: "", Width: 2},
		{Title: "SUMMARY", Width: 40},
		{Title: "DETAIL", Width: 28},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	t.SetStyles(tableStyles(theme))

	return &HistoryView{
		theme: theme, historyTable: t, historyCols: cols,
		tbl: viewport.New(0, 0),
	}
}

// Title implements View.
func (v *HistoryView) Title() string {
	if v.tab == tabLogs {
		return "Logs"
	}
	return "History"
}

var historyKeys = struct {
	nextTab, prevTab key.Binding
}{
	nextTab: key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "logs")),
	prevTab: key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab", "history")),
}

// Help implements View.
func (v *HistoryView) Help() []key.Binding {
	return []key.Binding{historyKeys.nextTab, historyKeys.prevTab}
}

// SelectionSummary feeds the status bar.
func (v *HistoryView) SelectionSummary() string {
	if v.tab == tabLogs {
		return strconv.Itoa(len(v.logs)) + " lines"
	}
	return strconv.Itoa(len(v.history)) + " entries"
}

// ShowLogs switches to the logs pane.
func (v *HistoryView) ShowLogs() { v.tab = tabLogs; v.render() }

// ShowHistory switches to the history pane.
func (v *HistoryView) ShowHistory() { v.tab = tabHistory }

// Update implements View.
func (v *HistoryView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.DataLoadedMsg:
		if m.Err == nil {
			v.history, v.logs = m.History, m.Logs
			v.rebuild()
			v.render()
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(m, historyKeys.nextTab):
			v.tab = tabLogs
			v.render()
			return v, nil
		case key.Matches(m, historyKeys.prevTab):
			v.tab = tabHistory
			return v, nil
		}
		if v.tab == tabLogs {
			vp, cmd := v.tbl.Update(m)
			v.tbl = vp
			return v, cmd
		}
		t, cmd := v.historyTable.Update(m)
		v.historyTable = t
		return v, cmd
	}
	return v, nil
}

func (v *HistoryView) rebuild() {
	rows := make([]table.Row, 0, len(v.history))
	for _, e := range v.history {
		// Plain text only — see the note on PortsView.row.
		mark := styles.GlyphOK
		if !e.Success {
			mark = styles.GlyphDown
		}
		detail := e.Detail
		if e.Err != "" {
			detail = e.Err
		}
		rows = append(rows, table.Row{
			e.At.Format(timeFormat),
			string(e.Kind),
			mark,
			e.Summary,
			detail,
		})
	}
	v.historyTable.SetRows(rows)
}

// render rebuilds the scrollable log body.
func (v *HistoryView) render() {
	if v.tab != tabLogs {
		return
	}
	if len(v.logs) == 0 {
		v.tbl.SetContent(v.theme.Faint.Render(
			"No log entries. Anything OpenPort could not do — a permission error, a\n" +
				"procfs fallback, a probe that timed out — shows up here."))
		return
	}

	var b strings.Builder
	for _, e := range v.logs {
		style := v.theme.Faint
		switch e.Level {
		case model.LogError:
			style = v.theme.Danger
		case model.LogWarn:
			style = v.theme.Warning
		}
		b.WriteString(v.theme.Faint.Render(e.At.Format(timeFormat)) + " ")
		b.WriteString(style.Render(padRight(strings.ToUpper(string(e.Level)), 6)))
		b.WriteString(v.theme.Faint.Render(padRight(e.Source, 10)))
		b.WriteString(v.theme.Body.Render(e.Message))
		b.WriteString("\n")
	}
	v.tbl.SetContent(strings.TrimRight(b.String(), "\n"))
	v.tbl.GotoTop()
}

// View implements View.
func (v *HistoryView) View(width, height int) string {
	tabs := tabBar(int(v.tab), []string{"History", "Logs"}, v.theme.TabActive, v.theme.Tab)

	bodyHeight := height - 3
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	if v.tab == tabLogs {
		v.tbl.Width, v.tbl.Height = width, bodyHeight
		if !v.ready {
			v.render()
			v.ready = true
		}
		return tabs + "\n\n" + v.tbl.View()
	}

	if len(v.history) == 0 {
		return tabs + "\n\n" + v.theme.Faint.Render(
			"Nothing yet. Every kill, suggestion, reservation and blueprint apply is\n"+
				"recorded here with a timestamp, so you can always answer \"what did I "+
				"just do?\".")
	}

	fitTable(&v.historyTable, v.historyCols, width, bodyHeight)
	return tabs + "\n\n" + v.historyTable.View()
}
