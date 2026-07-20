package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/ports"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// sortColumn identifies which column the table is ordered by.
type sortColumn int

const (
	sortByPort sortColumn = iota
	sortByProcess
	sortByPID
	sortByAlias
	sortByState
)

var sortNames = map[sortColumn]string{
	sortByPort:    "port",
	sortByProcess: "process",
	sortByPID:     "pid",
	sortByAlias:   "alias",
	sortByState:   "state",
}

// Layout thresholds for the detail pane.
const (
	// detailPaneMinWidth is the width at which the pane fits beside the table.
	detailPaneMinWidth = 104
	// stackedDetailMinHeight is the body height needed to stack it underneath
	// instead; below this the table itself would be squeezed to nothing.
	stackedDetailMinHeight = 16
	// stackedDetailHeight is how many rows the stacked pane occupies.
	stackedDetailHeight = 8
)

// PortsView is the home screen: the live table of listening ports plus a
// detail pane for the selection.
type PortsView struct {
	theme styles.Theme

	all      []ports.Port // everything the service reported
	filtered []ports.Port // what the table currently shows
	table    table.Model

	health map[int]*ports.HealthResult

	filterInput textinput.Model
	filtering   bool

	sortBy   sortColumn
	sortDesc bool

	// showDetail is toggled with tab, so the whole width can go to the table
	// when the user wants it.
	showDetail bool

	// selectedKey survives refreshes: the cursor follows the process, not the
	// row index, so a listing that reorders under watch mode does not move the
	// user's selection.
	selectedKey string

	watching bool
	loading  bool
	err      error

	width, height int
}

// NewPortsView builds the ports screen.
func NewPortsView(theme styles.Theme) *PortsView {
	filter := textinput.New()
	filter.Prompt = "/"
	filter.Placeholder = "filter by port, process, alias or group"
	filter.CharLimit = 48

	t := table.New(
		table.WithColumns(portColumns(100)),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles(theme))

	return &PortsView{
		theme:       theme,
		table:       t,
		health:      map[int]*ports.HealthResult{},
		filterInput: filter,
		showDetail:  true,
		loading:     true,
	}
}

// tableStyles applies the app theme to a bubbles table.
func tableStyles(theme styles.Theme) table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Muted)
	s.Selected = s.Selected.
		Foreground(styles.Accent).
		Bold(true)
	s.Cell = s.Cell.Foreground(styles.Text)
	return s
}

// Column identifiers. The flag column has no header, so it is keyed by a
// sentinel rather than by its (empty) title.
const (
	colPort    = "PORT"
	colProto   = "PROTO"
	colPID     = "PID"
	colProcess = "PROCESS"
	colAlias   = "ALIAS"
	colGroups  = "GROUPS"
	colState   = "STATE"
	colHealth  = "HLTH"
	colFlag    = " " // protection marker
	colCommand = "COMMAND"
)

// cellPadding is what bubbles/table adds around each cell's content.
const cellPadding = 2

// portColumnSpecs is the full column set in display order. dropOrder decides
// what goes first when the terminal is too narrow — higher drops sooner. Port
// and process have no drop order because a ports table without them is not a
// ports table.
var portColumnSpecs = []struct {
	key   string
	width int
	drop  int
}{
	{colPort, 6, 0},
	{colProto, 6, 70},
	{colPID, 7, 40},
	{colProcess, 16, 0},
	{colAlias, 14, 30},
	{colGroups, 14, 90},
	{colState, 8, 60},
	{colHealth, 5, 50},
	{colFlag, 2, 20},
}

// portColumns lays the table out for the available width, shedding the least
// important columns until it fits and giving any remainder to the command
// line. Without this the fixed column set alone needs ~97 cells, so an 80-wide
// terminal — or a split pane — would render a table wider than the screen.
func portColumns(width int) []table.Column {
	keep := make([]bool, len(portColumnSpecs))
	for i := range keep {
		keep[i] = true
	}

	total := func() int {
		sum := 0
		for i, spec := range portColumnSpecs {
			if keep[i] {
				sum += spec.width + cellPadding
			}
		}
		return sum
	}

	// Shed the highest drop-order column still standing until we fit.
	for total() > width {
		worst, worstDrop := -1, 0
		for i, spec := range portColumnSpecs {
			if keep[i] && spec.drop > worstDrop {
				worst, worstDrop = i, spec.drop
			}
		}
		if worst < 0 {
			break // only the essential columns remain
		}
		keep[worst] = false
	}

	cols := make([]table.Column, 0, len(portColumnSpecs)+1)
	for i, spec := range portColumnSpecs {
		if !keep[i] {
			continue
		}
		title := spec.key
		if spec.key == colFlag {
			title = ""
		}
		cols = append(cols, table.Column{Title: title, Width: spec.width})
	}

	// On a very narrow terminal even port+process overflow; shrink them
	// rather than render past the edge.
	if used := total(); used > width && len(cols) > 0 {
		shrinkColumns(cols, width)
		return cols
	}

	if rest := width - total() - cellPadding; rest >= 16 {
		cols = append(cols, table.Column{Title: colCommand, Width: rest})
	}
	return cols
}

// shrinkColumns scales column widths down proportionally to fit width.
func shrinkColumns(cols []table.Column, width int) {
	used := 0
	for _, c := range cols {
		used += c.Width + cellPadding
	}
	if used <= width || used == 0 {
		return
	}
	available := width - len(cols)*cellPadding
	if available < len(cols) {
		available = len(cols) // one cell each, at minimum
	}
	contentWidth := used - len(cols)*cellPadding
	for i := range cols {
		scaled := cols[i].Width * available / contentWidth
		if scaled < 1 {
			scaled = 1
		}
		cols[i].Width = scaled
	}
}

// Title implements View.
func (v *PortsView) Title() string { return "Ports" }

var portsKeys = struct {
	kill, forceKill, killPID, check, health, alias, protect,
	watch, filter, clearFilter, sort, sortRev, refresh, suggest, detail key.Binding
}{
	kill:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill")),
	forceKill:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "force kill")),
	killPID:     key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "kill by pid")),
	check:       key.NewBinding(key.WithKeys("c", "enter"), key.WithHelp("c", "check")),
	health:      key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "health check")),
	alias:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "alias")),
	protect:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "protect")),
	watch:       key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "watch")),
	filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	clearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
	sort:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
	sortRev:     key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "reverse")),
	refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	suggest:     key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "free port")),
	detail:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "detail pane")),
}

// Help implements View.
func (v *PortsView) Help() []key.Binding {
	return []key.Binding{
		portsKeys.check, portsKeys.kill, portsKeys.forceKill, portsKeys.killPID,
		portsKeys.health, portsKeys.alias, portsKeys.protect, portsKeys.watch,
		portsKeys.filter, portsKeys.sort, portsKeys.sortRev, portsKeys.suggest,
		portsKeys.detail, portsKeys.refresh,
	}
}

// Selection returns the highlighted port, if any.
func (v *PortsView) Selection() (ports.Port, bool) {
	idx := v.table.Cursor()
	if idx < 0 || idx >= len(v.filtered) {
		return ports.Port{}, false
	}
	return v.filtered[idx], true
}

// SelectionSummary feeds the status bar.
func (v *PortsView) SelectionSummary() string {
	p, ok := v.Selection()
	if !ok {
		return ""
	}
	name := p.Label()
	if name == "" {
		name = "unknown"
	}
	return fmt.Sprintf("%d %s", p.Port, name)
}

// Watching reports whether live refresh is on, for the status bar indicator.
func (v *PortsView) Watching() bool { return v.watching }

// CapturingInput tells the root model that the filter field has the keyboard,
// so global single-letter bindings must not fire while the user is typing.
func (v *PortsView) CapturingInput() bool { return v.filtering }

// Update implements View.
func (v *PortsView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.PortsLoadedMsg:
		v.loading = false
		v.err = m.Err
		if m.Err == nil {
			v.setPorts(m.Ports)
		}
		return v, nil

	case msg.WatchTickMsg:
		v.loading = false
		v.setPorts(m.Ports)
		return v, nil

	case msg.WatchStoppedMsg:
		v.watching = false
		return v, nil

	case msg.HealthResultMsg:
		if m.Err == nil {
			result := m.Result
			v.health[m.Port] = &result
			v.applyHealth()
			v.rebuild()
		}
		return v, nil

	case msg.PortCheckedMsg:
		// A check refreshes one row's health/detail in place.
		if m.Err == nil && m.Found && m.Port.Health != nil {
			result := *m.Port.Health
			v.health[m.Port.Port] = &result
			v.applyHealth()
			v.rebuild()
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(m)
	}
	return v, nil
}

// handleKey routes a keypress, with the filter input taking precedence while
// it is focused.
func (v *PortsView) handleKey(keyMsg tea.KeyMsg) (View, tea.Cmd) {
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
			v.rebuild()
			return v, nil
		}
		input, cmd := v.filterInput.Update(keyMsg)
		v.filterInput = input
		v.rebuild()
		return v, cmd
	}

	selected, hasSelection := v.Selection()

	switch {
	case key.Matches(keyMsg, portsKeys.filter):
		v.filtering = true
		return v, v.filterInput.Focus()

	case key.Matches(keyMsg, portsKeys.clearFilter):
		if v.filterInput.Value() != "" {
			v.filterInput.SetValue("")
			v.rebuild()
			return v, nil
		}
		return v, nil

	case key.Matches(keyMsg, portsKeys.sort):
		v.sortBy = (v.sortBy + 1) % 5
		v.rebuild()
		return v, toast(msg.Info("Sorted by " + sortNames[v.sortBy]))

	case key.Matches(keyMsg, portsKeys.sortRev):
		v.sortDesc = !v.sortDesc
		v.rebuild()
		return v, nil

	case key.Matches(keyMsg, portsKeys.watch):
		return v, emit(msg.ToggleWatchMsg{})

	case key.Matches(keyMsg, portsKeys.refresh):
		return v, emit(msg.ListPortsMsg{})

	case key.Matches(keyMsg, portsKeys.suggest):
		return v, emit(msg.RunActionMsg{ID: "ports.suggest"})

	case key.Matches(keyMsg, portsKeys.killPID):
		return v, emit(msg.RunActionMsg{ID: "ports.kill_pid"})

	case key.Matches(keyMsg, portsKeys.detail):
		v.showDetail = !v.showDetail
		return v, nil
	}

	// The rest need a selected row.
	if !hasSelection {
		var cmd tea.Cmd
		v.table, cmd = v.table.Update(keyMsg)
		return v, cmd
	}

	switch {
	case key.Matches(keyMsg, portsKeys.kill):
		return v, emit(msg.KillPortMsg{Port: selected.Port})
	case key.Matches(keyMsg, portsKeys.forceKill):
		return v, emit(msg.KillPortMsg{Port: selected.Port, Force: true})
	case key.Matches(keyMsg, portsKeys.check):
		return v, emit(msg.CheckPortMsg{Port: selected.Port})
	case key.Matches(keyMsg, portsKeys.health):
		return v, emit(msg.HealthCheckMsg{Port: selected.Port, Kind: ports.HealthTCP})
	case key.Matches(keyMsg, portsKeys.protect):
		return v, emit(msg.ToggleProtectedMsg{Port: selected.Port})
	case key.Matches(keyMsg, portsKeys.alias):
		return v, emit(msg.RunActionMsg{ID: "alias.new"})
	}

	// Anything else is table motion; remember where the cursor landed.
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(keyMsg)
	v.rememberSelection()
	return v, cmd
}

// SetWatching reflects the root model's watch state.
func (v *PortsView) SetWatching(on bool) { v.watching = on }

// setPorts installs a fresh listing, preserving health results and selection.
func (v *PortsView) setPorts(list []ports.Port) {
	v.all = list
	v.applyHealth()
	v.rebuild()
}

// applyHealth re-attaches cached probe results to the current listing, so a
// health badge survives the next refresh.
func (v *PortsView) applyHealth() {
	for i := range v.all {
		if h, ok := v.health[v.all[i].Port]; ok {
			v.all[i].Health = h
		}
	}
}

// rebuild re-filters, re-sorts and re-renders the table.
func (v *PortsView) rebuild() {
	query := strings.ToLower(strings.TrimSpace(v.filterInput.Value()))
	v.filtered = v.filtered[:0]
	for _, p := range v.all {
		if query == "" || matchesFilter(p, query) {
			v.filtered = append(v.filtered, p)
		}
	}
	v.sortPorts()

	rows := make([]table.Row, 0, len(v.filtered))
	for _, p := range v.filtered {
		rows = append(rows, v.row(p))
	}
	v.table.SetRows(rows)
	v.restoreSelection()
}

// matchesFilter reports whether a port matches the filter query.
func matchesFilter(p ports.Port, query string) bool {
	haystacks := []string{
		strconv.Itoa(p.Port),
		strings.ToLower(p.Process),
		strings.ToLower(p.Alias),
		strings.ToLower(p.Command),
		strings.ToLower(p.Protocol),
		strings.ToLower(strings.Join(p.Groups, " ")),
	}
	for _, h := range haystacks {
		if h != "" && strings.Contains(h, query) {
			return true
		}
	}
	return false
}

func (v *PortsView) sortPorts() {
	less := func(i, j int) bool {
		a, b := v.filtered[i], v.filtered[j]
		switch v.sortBy {
		case sortByProcess:
			if a.Process != b.Process {
				return a.Process < b.Process
			}
		case sortByPID:
			if a.PID != b.PID {
				return a.PID < b.PID
			}
		case sortByAlias:
			if a.Alias != b.Alias {
				// Aliased rows sort ahead of unnamed ones.
				if a.Alias == "" {
					return false
				}
				if b.Alias == "" {
					return true
				}
				return a.Alias < b.Alias
			}
		case sortByState:
			if a.State != b.State {
				return a.State < b.State
			}
		}
		// Port is the tiebreaker for every ordering, so the table is stable.
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		return a.Protocol < b.Protocol
	}

	sort.SliceStable(v.filtered, func(i, j int) bool {
		if v.sortDesc {
			return less(j, i)
		}
		return less(i, j)
	})
}

// row renders one port as table cells.
//
// Cells must be plain text: bubbles/table truncates each one with
// runewidth.Truncate, which counts ANSI escape bytes as runes — a styled cell
// gets cut mid-escape and bleeds colour codes across the row. Colour in this
// table therefore comes from the row-level selected style and from the glyphs
// themselves, not from per-cell styling.
func (v *PortsView) row(p ports.Port) table.Row {
	pid := "—"
	if p.PID > 0 {
		pid = strconv.Itoa(p.PID)
	}
	process := p.Process
	if process == "" {
		process = "—"
	}
	flag := ""
	if p.Protected {
		flag = styles.GlyphProtected
	}

	byKey := map[string]string{
		colPort:    strconv.Itoa(p.Port),
		colProto:   p.Protocol,
		colPID:     pid,
		colProcess: process,
		colAlias:   p.Alias,
		colGroups:  strings.Join(p.Groups, ","),
		colState:   p.State,
		colHealth:  p.Health.Badge(),
		colFlag:    flag,
		colCommand: p.Command,
	}

	// Emit cells in whatever order the current (width-dependent) column set
	// says, so dropping a column can never shift the remaining values.
	cols := v.table.Columns()
	cells := make(table.Row, 0, len(cols))
	for _, c := range cols {
		key := c.Title
		if key == "" {
			key = colFlag
		}
		cells = append(cells, byKey[key])
	}
	return cells
}

// rememberSelection records what the cursor is on, keyed by port+proto+pid.
func (v *PortsView) rememberSelection() {
	if p, ok := v.Selection(); ok {
		v.selectedKey = p.Key()
	}
}

// restoreSelection puts the cursor back on the same process after a refresh,
// falling back to the nearest valid row when it has gone away.
func (v *PortsView) restoreSelection() {
	if len(v.filtered) == 0 {
		v.selectedKey = ""
		return
	}
	if v.selectedKey != "" {
		for i, p := range v.filtered {
			if p.Key() == v.selectedKey {
				v.table.SetCursor(i)
				return
			}
		}
	}
	// The tracked row is gone (killed, or filtered out): clamp the cursor.
	if v.table.Cursor() >= len(v.filtered) {
		v.table.SetCursor(len(v.filtered) - 1)
	}
	v.rememberSelection()
}

// View implements View.
func (v *PortsView) View(width, height int) string {
	v.width, v.height = width, height

	if v.err != nil {
		return v.theme.Danger.Render("Could not read ports: "+v.err.Error()) + "\n\n" +
			v.theme.Faint.Render("Press r to retry.")
	}
	if v.loading && len(v.all) == 0 {
		return v.theme.Faint.Render("Reading listening ports…")
	}

	header := v.headerLine(width)
	bodyHeight := height - lipgloss.Height(header) - 1
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	// The detail pane carries things the table cannot: the full command line,
	// the owning user, why a port is protected. Dropping it outright on a
	// narrow terminal loses that information entirely, so it stacks under the
	// table instead — and only disappears when there is genuinely no room,
	// which the status bar hint then explains.
	sideBySide := width >= detailPaneMinWidth && v.showDetail
	stacked := !sideBySide && v.showDetail && bodyHeight >= stackedDetailMinHeight

	tableWidth := width
	tableHeight := bodyHeight
	if sideBySide {
		tableWidth = width * 62 / 100
	}
	if stacked {
		tableHeight = bodyHeight - stackedDetailHeight
	}

	v.applyColumns(portColumns(tableWidth))
	v.table.SetWidth(tableWidth)
	v.table.SetHeight(tableHeight)

	body := v.table.View()
	if len(v.filtered) == 0 {
		body = v.emptyState()
	}

	switch {
	case sideBySide:
		detail := v.theme.Panel.
			Width(width - tableWidth - 4).
			Height(bodyHeight - 2).
			Render(v.detailPane(width-tableWidth-8, bodyHeight-2))

		return header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(tableWidth).Render(body),
			detail,
		)

	case stacked:
		detail := v.theme.Panel.
			Width(width - 2).
			Height(stackedDetailHeight - 2).
			Render(v.detailPane(width-6, stackedDetailHeight-2))

		return header + "\n" + body + "\n" + detail

	default:
		return header + "\n" + body
	}
}

// applyColumns installs a new column set, re-rendering the rows to match.
//
// Row cells are addressed positionally and bubbles/table re-renders inside
// SetColumns, so leaving the old rows in place while widening the column set
// makes it index past the end of every row and panic. Clearing first is what
// lets the table reflow safely as the terminal resizes.
func (v *PortsView) applyColumns(cols []table.Column) {
	if sameColumns(v.table.Columns(), cols) {
		return
	}
	v.table.SetRows(nil)
	v.table.SetColumns(cols)

	rows := make([]table.Row, 0, len(v.filtered))
	for _, p := range v.filtered {
		rows = append(rows, v.row(p))
	}
	v.table.SetRows(rows)
	v.restoreSelection()
}

func sameColumns(a, b []table.Column) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Width != b[i].Width {
			return false
		}
	}
	return true
}

func (v *PortsView) emptyState() string {
	if v.filterInput.Value() != "" {
		return v.theme.Faint.Render("No ports match " +
			strconv.Quote(v.filterInput.Value()) + ". Press esc to clear the filter.")
	}
	return v.theme.Faint.Render("Nothing is listening right now.")
}

// headerLine renders the count, sort indicator and filter input.
func (v *PortsView) headerLine(width int) string {
	if v.filtering {
		return v.filterInput.View()
	}

	left := v.theme.Bold.Render(strconv.Itoa(len(v.filtered)))
	if len(v.filtered) != len(v.all) {
		left += v.theme.Faint.Render(" of " + strconv.Itoa(len(v.all)))
	}
	left += v.theme.Faint.Render(" ports")

	if q := v.filterInput.Value(); q != "" {
		left += v.theme.Accent.Render("  /" + q)
	}

	arrow := "↑"
	if v.sortDesc {
		arrow = "↓"
	}
	right := v.theme.Faint.Render("sort: " + sortNames[v.sortBy] + " " + arrow)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// detailPane renders everything known about the selected port, trimmed to the
// rows available.
func (v *PortsView) detailPane(width, height int) string {
	p, ok := v.Selection()
	if !ok {
		return v.theme.Faint.Render("No selection.")
	}

	var b strings.Builder
	title := strconv.Itoa(p.Port)
	if p.Alias != "" {
		title += "  " + p.Alias
	}
	b.WriteString(v.theme.PanelTitle.Render(title) + "\n")

	rows := [][2]string{
		{"Process", orDash(p.Process)},
		{"PID", orDashInt(p.PID)},
		{"Protocol", p.Protocol},
		{"Address", orDash(p.Addr)},
		{"State", orDash(p.State)},
		{"User", orDash(p.User)},
	}
	if len(p.Groups) > 0 {
		rows = append(rows, [2]string{"Groups", strings.Join(p.Groups, ", ")})
	}
	for _, r := range rows {
		b.WriteString(v.theme.Faint.Render(padRight(r[0], 9)))
		b.WriteString(v.theme.Body.Render(r[1]))
		b.WriteString("\n")
	}

	if p.Protected {
		b.WriteString("\n" + v.theme.Warning.Render(styles.GlyphProtected+" "+p.Reason) + "\n")
	}

	if p.Health != nil {
		b.WriteString("\n")
		style := v.theme.Success
		glyph := styles.GlyphOK
		if !p.Health.OK {
			style, glyph = v.theme.Danger, styles.GlyphDown
		}
		line := fmt.Sprintf("%s %s (%dms)", glyph, p.Health.Status, p.Health.LatencyMS)
		b.WriteString(style.Render(line) + "\n")
	}

	if p.Command != "" {
		b.WriteString("\n" + v.theme.Faint.Render("Command") + "\n")
		b.WriteString(v.theme.Body.Render(wrapText(p.Command, width)) + "\n")
	}

	out := strings.TrimRight(b.String(), "\n")
	if height > 0 {
		out = clipToHeight(out, height)
	}
	return out
}

// clipToHeight trims a block to n rows, marking the cut.
func clipToHeight(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return strings.Join(append(lines[:n-1], "…"), "\n")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func orDashInt(n int) string {
	if n <= 0 {
		return "—"
	}
	return strconv.Itoa(n)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s + " "
	}
	return s + strings.Repeat(" ", n-len(s))
}

// wrapText hard-wraps to width, which keeps long command lines inside the
// detail pane instead of blowing out the layout.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > 6 {
		lines = append(lines[:6], "…")
	}
	return strings.Join(lines, "\n")
}

// emit wraps a message as a command. Views use this rather than mutating
// anything themselves.
func emit(m tea.Msg) tea.Cmd { return func() tea.Msg { return m } }

// toast is emit for notifications, kept separate for readability at call
// sites.
func toast(m msg.ToastMsg) tea.Cmd { return func() tea.Msg { return m } }
