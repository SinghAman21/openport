package views

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// aliasTab selects which pane of the view is showing.
type aliasTab int

const (
	tabAliases aliasTab = iota
	tabGroups
)

// AliasesView manages the two things that give ports names: aliases (one
// name, one port) and groups (one name, many ports).
type AliasesView struct {
	theme styles.Theme
	tab   aliasTab

	aliases []model.Alias
	groups  []model.Group

	aliasTable table.Model
	groupTable table.Model
	aliasCols  []table.Column
	groupCols  []table.Column
}

// NewAliasesView builds the aliases/groups manager.
func NewAliasesView(theme styles.Theme) *AliasesView {
	aliasCols := []table.Column{
		{Title: "NAME", Width: 22},
		{Title: "PORT", Width: 8},
		{Title: "NOTE", Width: 32},
	}
	groupCols := []table.Column{
		{Title: "NAME", Width: 22},
		{Title: "PORTS", Width: 34},
		{Title: "COUNT", Width: 7},
	}

	at := table.New(table.WithColumns(aliasCols), table.WithFocused(true))
	at.SetStyles(tableStyles(theme))
	gt := table.New(table.WithColumns(groupCols))
	gt.SetStyles(tableStyles(theme))

	return &AliasesView{
		theme: theme, aliasTable: at, groupTable: gt,
		aliasCols: aliasCols, groupCols: groupCols,
	}
}

// Title implements View.
func (v *AliasesView) Title() string {
	if v.tab == tabGroups {
		return "Groups"
	}
	return "Aliases"
}

var aliasKeys = struct {
	newItem, edit, del, shutdown, health, nextTab, prevTab key.Binding
}{
	newItem:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	edit:     key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e", "edit")),
	del:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	shutdown: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "shut down group")),
	health:   key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "health check group")),
	nextTab:  key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "next tab")),
	prevTab:  key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab", "prev tab")),
}

// Help implements View.
func (v *AliasesView) Help() []key.Binding {
	base := []key.Binding{aliasKeys.newItem, aliasKeys.edit, aliasKeys.del, aliasKeys.nextTab}
	if v.tab == tabGroups {
		return append(base, aliasKeys.shutdown, aliasKeys.health)
	}
	return base
}

// SelectionSummary feeds the status bar.
func (v *AliasesView) SelectionSummary() string {
	if v.tab == tabGroups {
		if g, ok := v.selectedGroup(); ok {
			return g.Name + " (" + strconv.Itoa(len(g.Ports)) + " ports)"
		}
		return ""
	}
	if a, ok := v.selectedAlias(); ok {
		return a.Name + " → " + strconv.Itoa(a.Port)
	}
	return ""
}

func (v *AliasesView) selectedAlias() (model.Alias, bool) {
	i := v.aliasTable.Cursor()
	if i < 0 || i >= len(v.aliases) {
		return model.Alias{}, false
	}
	return v.aliases[i], true
}

func (v *AliasesView) selectedGroup() (model.Group, bool) {
	i := v.groupTable.Cursor()
	if i < 0 || i >= len(v.groups) {
		return model.Group{}, false
	}
	return v.groups[i], true
}

// SetTab switches panes, used when an action jumps straight to groups.
func (v *AliasesView) SetTab(t aliasTab) { v.tab = t; v.focusActive() }

// ShowGroups switches to the groups pane.
func (v *AliasesView) ShowGroups() { v.SetTab(tabGroups) }

// ShowAliases switches to the aliases pane.
func (v *AliasesView) ShowAliases() { v.SetTab(tabAliases) }

func (v *AliasesView) focusActive() {
	if v.tab == tabGroups {
		v.aliasTable.Blur()
		v.groupTable.Focus()
		return
	}
	v.groupTable.Blur()
	v.aliasTable.Focus()
}

// Update implements View.
func (v *AliasesView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.DataLoadedMsg:
		if m.Err == nil {
			v.setData(m.Aliases, m.Groups)
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(m)
	}
	return v, nil
}

func (v *AliasesView) handleKey(keyMsg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, aliasKeys.nextTab):
		v.SetTab(tabGroups)
		return v, nil
	case key.Matches(keyMsg, aliasKeys.prevTab):
		v.SetTab(tabAliases)
		return v, nil
	}

	if v.tab == tabGroups {
		return v.handleGroupKey(keyMsg)
	}
	return v.handleAliasKey(keyMsg)
}

func (v *AliasesView) handleAliasKey(keyMsg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, aliasKeys.newItem):
		return v, emit(msg.RunActionMsg{ID: actions.AliasNew})

	case key.Matches(keyMsg, aliasKeys.edit):
		if a, ok := v.selectedAlias(); ok {
			return v, emit(editAliasMsg{Alias: a})
		}
		return v, nil

	case key.Matches(keyMsg, aliasKeys.del):
		if a, ok := v.selectedAlias(); ok {
			return v, emit(msg.DeleteAliasMsg{Name: a.Name})
		}
		return v, nil
	}

	t, cmd := v.aliasTable.Update(keyMsg)
	v.aliasTable = t
	return v, cmd
}

func (v *AliasesView) handleGroupKey(keyMsg tea.KeyMsg) (View, tea.Cmd) {
	switch {
	case key.Matches(keyMsg, aliasKeys.newItem):
		return v, emit(msg.RunActionMsg{ID: actions.GroupNew})

	case key.Matches(keyMsg, aliasKeys.edit):
		if g, ok := v.selectedGroup(); ok {
			return v, emit(editGroupMsg{Group: g})
		}
		return v, nil

	case key.Matches(keyMsg, aliasKeys.del):
		if g, ok := v.selectedGroup(); ok {
			return v, emit(msg.DeleteGroupMsg{Name: g.Name})
		}
		return v, nil

	case key.Matches(keyMsg, aliasKeys.shutdown):
		if g, ok := v.selectedGroup(); ok {
			return v, emit(msg.GroupActionMsg{Group: g.Name, Kind: msg.GroupShutdown})
		}
		return v, nil

	case key.Matches(keyMsg, aliasKeys.health):
		if g, ok := v.selectedGroup(); ok {
			return v, emit(msg.GroupActionMsg{Group: g.Name, Kind: msg.GroupHealth})
		}
		return v, nil
	}

	t, cmd := v.groupTable.Update(keyMsg)
	v.groupTable = t
	return v, cmd
}

// editAliasMsg and editGroupMsg ask the root model to open a prefilled form.
// They live here because only this view produces them.
type editAliasMsg struct{ Alias model.Alias }
type editGroupMsg struct{ Group model.Group }

// EditAliasRequest exposes the payload of an edit intent to the root model.
func EditAliasRequest(m tea.Msg) (model.Alias, bool) {
	e, ok := m.(editAliasMsg)
	return e.Alias, ok
}

// EditGroupRequest exposes the payload of an edit intent to the root model.
func EditGroupRequest(m tea.Msg) (model.Group, bool) {
	e, ok := m.(editGroupMsg)
	return e.Group, ok
}

func (v *AliasesView) setData(aliases []model.Alias, groups []model.Group) {
	v.aliases, v.groups = aliases, groups

	aliasRows := make([]table.Row, 0, len(aliases))
	for _, a := range aliases {
		aliasRows = append(aliasRows, table.Row{a.Name, strconv.Itoa(a.Port), a.Note})
	}
	v.aliasTable.SetRows(aliasRows)

	groupRows := make([]table.Row, 0, len(groups))
	for _, g := range groups {
		parts := make([]string, 0, len(g.Ports))
		for _, p := range g.Ports {
			parts = append(parts, strconv.Itoa(p))
		}
		groupRows = append(groupRows, table.Row{
			g.Name, strings.Join(parts, ", "), strconv.Itoa(len(g.Ports)),
		})
	}
	v.groupTable.SetRows(groupRows)
	v.focusActive()
}

// View implements View.
func (v *AliasesView) View(width, height int) string {
	tabs := tabBar(int(v.tab), []string{"Aliases", "Groups"}, v.theme.TabActive, v.theme.Tab)
	body := ""

	bodyHeight := height - 3
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	if v.tab == tabGroups {
		if len(v.groups) == 0 {
			body = v.theme.Faint.Render(
				"No groups yet. Press n to bundle ports together, e.g. dev_stack → 3000, 5432, 6379.")
		} else {
			fitTable(&v.groupTable, v.groupCols, width, bodyHeight)
			body = v.groupTable.View()
		}
	} else {
		if len(v.aliases) == 0 {
			body = v.theme.Faint.Render(
				"No aliases yet. Press n to name a port, e.g. frontend → 3000.")
		} else {
			fitTable(&v.aliasTable, v.aliasCols, width, bodyHeight)
			body = v.aliasTable.View()
		}
	}

	return tabs + "\n\n" + body
}
