package views

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
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

// historyTabCount is how many sub-tabs the view cycles through, so Tab wraps
// from the last back to the first instead of dead-ending.
const historyTabCount = 2

// timeFormat is the timestamp format used across both panes.
const timeFormat = "15:04:05"

// HistoryView is the audit trail: every mutating action OpenPort has taken,
// plus the internal log of OS interactions that did not go to plan.
type HistoryView struct {
	theme styles.Theme
	tab   historyTab

	// Full data as loaded; the filtered projections are what the panes render.
	history  []model.HistoryEntry
	logs     []model.LogEntry
	fHistory []model.HistoryEntry
	fLogs    []model.LogEntry

	// One filter box serves both panes: the query persists as you switch
	// between History and Logs, so "everything about port 3000" stays narrowed
	// whichever pane you look at it in.
	filterInput textinput.Model
	filtering   bool

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

	filter := textinput.New()
	filter.Prompt = "/"
	filter.Placeholder = "filter by action, summary, source or message"
	filter.CharLimit = 48

	return &HistoryView{
		theme: theme, historyTable: t, historyCols: cols,
		tbl:         viewport.New(0, 0),
		filterInput: filter,
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
	nextTab, prevTab, filter key.Binding
}{
	nextTab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "logs")),
	prevTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "history")),
	filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
}

// Help implements View.
func (v *HistoryView) Help() []key.Binding {
	return []key.Binding{historyKeys.filter, historyKeys.nextTab, historyKeys.prevTab}
}

// CapturingInput tells the root model the filter box has the keyboard, so
// global single-letter bindings and section switching do not fire while typing.
func (v *HistoryView) CapturingInput() bool { return v.filtering }

// SelectionSummary feeds the status bar.
func (v *HistoryView) SelectionSummary() string {
	if v.tab == tabLogs {
		return countLabel(len(v.fLogs), len(v.logs), "line", "lines")
	}
	return countLabel(len(v.fHistory), len(v.history), "entry", "entries")
}

// countLabel renders "N entries", or "N of M entries" when a filter is hiding
// some of them.
func countLabel(shown, total int, one, many string) string {
	noun := many
	if shown == 1 {
		noun = one
	}
	if shown != total {
		return strconv.Itoa(shown) + " of " + strconv.Itoa(total) + " " + many
	}
	return strconv.Itoa(shown) + " " + noun
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
			v.applyFilter()
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(m)
	}
	return v, nil
}

// handleKey routes a keypress, with the filter box taking precedence while it
// is focused.
func (v *HistoryView) handleKey(keyMsg tea.KeyMsg) (View, tea.Cmd) {
	if v.filtering {
		switch keyMsg.String() {
		case "enter":
			v.filtering = false
			v.filterInput.Blur()
			return v, nil
		case "esc":
			v.filtering = false
			v.filterInput.Blur()
			v.filterInput.SetValue("")
			v.applyFilter()
			return v, nil
		case "up", "down", "pgup", "pgdown", "home", "end":
			// Scroll the results without leaving the search box.
			return v.scrollActive(keyMsg)
		}
		input, cmd := v.filterInput.Update(keyMsg)
		v.filterInput = input
		v.applyFilter()
		return v, cmd
	}

	switch {
	case key.Matches(keyMsg, historyKeys.filter):
		v.filtering = true
		return v, v.filterInput.Focus()
	case key.Matches(keyMsg, historyKeys.nextTab):
		v.tab = (v.tab + 1) % historyTabCount
		v.render()
		return v, nil
	case key.Matches(keyMsg, historyKeys.prevTab):
		v.tab = (v.tab + historyTabCount - 1) % historyTabCount
		v.render()
		return v, nil
	}

	return v.scrollActive(keyMsg)
}

// scrollActive forwards a motion key to whichever pane is currently showing.
func (v *HistoryView) scrollActive(keyMsg tea.KeyMsg) (View, tea.Cmd) {
	if v.tab == tabLogs {
		vp, cmd := v.tbl.Update(keyMsg)
		v.tbl = vp
		return v, cmd
	}
	t, cmd := v.historyTable.Update(keyMsg)
	v.historyTable = t
	return v, cmd
}

// applyFilter recomputes both filtered projections from the current query and
// refreshes whichever pane is showing.
func (v *HistoryView) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(v.filterInput.Value()))

	v.fHistory = v.fHistory[:0]
	for _, e := range v.history {
		if matchesHistory(e, q) {
			v.fHistory = append(v.fHistory, e)
		}
	}
	v.fLogs = v.fLogs[:0]
	for _, e := range v.logs {
		if matchesLog(e, q) {
			v.fLogs = append(v.fLogs, e)
		}
	}

	v.rebuild()
	v.render()
}

// matchesHistory reports whether an audit entry matches the filter query.
func matchesHistory(e model.HistoryEntry, q string) bool {
	if q == "" {
		return true
	}
	for _, h := range []string{
		strings.ToLower(string(e.Kind)),
		strings.ToLower(e.Summary),
		strings.ToLower(e.Detail),
		strings.ToLower(e.Err),
		e.At.Format(timeFormat),
	} {
		if h != "" && strings.Contains(h, q) {
			return true
		}
	}
	return false
}

// matchesLog reports whether a log line matches the filter query.
func matchesLog(e model.LogEntry, q string) bool {
	if q == "" {
		return true
	}
	for _, h := range []string{
		strings.ToLower(string(e.Level)),
		strings.ToLower(e.Source),
		strings.ToLower(e.Message),
		e.At.Format(timeFormat),
	} {
		if h != "" && strings.Contains(h, q) {
			return true
		}
	}
	return false
}

func (v *HistoryView) rebuild() {
	rows := make([]table.Row, 0, len(v.fHistory))
	for _, e := range v.fHistory {
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
	setRows(&v.historyTable, rows)
}

// render rebuilds the scrollable log body.
func (v *HistoryView) render() {
	if v.tab != tabLogs {
		return
	}
	if len(v.fLogs) == 0 {
		if v.filterInput.Value() != "" {
			v.tbl.SetContent(v.theme.Faint.Render(
				"No log lines match " + strconv.Quote(v.filterInput.Value()) + "."))
			return
		}
		v.tbl.SetContent(v.theme.Faint.Render(
			"No log entries. Anything OpenPort could not do — a permission error, a\n" +
				"procfs fallback, a probe that timed out — shows up here."))
		return
	}

	var b strings.Builder
	for _, e := range v.fLogs {
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
	head := v.headerLine(tabs)

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
		return head + "\n\n" + v.tbl.View()
	}

	if len(v.history) == 0 {
		return head + "\n\n" + v.theme.Faint.Render(
			"Nothing yet. Every kill, suggestion, reservation and blueprint apply is\n"+
				"recorded here with a timestamp, so you can always answer \"what did I "+
				"just do?\".")
	}
	if len(v.fHistory) == 0 {
		return head + "\n\n" + v.theme.Faint.Render(
			"No entries match "+strconv.Quote(v.filterInput.Value())+". Press esc to clear the filter.")
	}

	fitTable(&v.historyTable, v.historyCols, width, bodyHeight)
	return head + "\n\n" + v.historyTable.View()
}

// headerLine puts the sub-tab strip and the filter side by side, so searching
// never costs a row of the pane below.
func (v *HistoryView) headerLine(tabs string) string {
	if v.filtering {
		return tabs + "  " + v.filterInput.View()
	}
	if q := v.filterInput.Value(); q != "" {
		return tabs + "  " + v.theme.Accent.Render("/"+q)
	}
	return tabs + "  " + v.theme.Faint.Render("press / to search")
}
