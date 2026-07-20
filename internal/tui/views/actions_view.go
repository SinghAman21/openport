package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/components"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// ActionsView is the discoverable menu of everything OpenPort can do. It and
// the command palette render the same catalog, so nothing is reachable from
// only one of them.
type ActionsView struct {
	theme styles.Theme
	menu  *components.Menu
	isPro bool
}

// NewActionsView builds the menu. isPro decides whether Pro entries are
// selectable; they are listed either way.
func NewActionsView(theme styles.Theme, isPro bool) *ActionsView {
	v := &ActionsView{theme: theme, isPro: isPro}
	v.menu = components.NewMenu(theme, ActionItems(isPro))
	return v
}

// ActionItems converts the action catalog into menu/palette items. Shared with
// the palette so both front doors stay in lockstep.
func ActionItems(isPro bool) []components.Item {
	all := actions.All()
	items := make([]components.Item, 0, len(all))
	for _, a := range all {
		item := components.Item{
			ID:    a.ID,
			Title: a.Title,
			Desc:  a.Desc,
			Group: string(a.Category),
			Key:   a.Key,
		}
		if a.Pro {
			item.Badge = "PRO"
			item.Disabled = !isPro
		}
		items = append(items, item)
	}
	return items
}

// SetPro re-gates the Pro entries.
func (v *ActionsView) SetPro(isPro bool) {
	v.isPro = isPro
	v.menu.SetItems(ActionItems(isPro))
}

// Title implements View.
func (v *ActionsView) Title() string { return "Actions" }

var actionsKeys = struct{ run key.Binding }{
	run: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run action")),
}

// Help implements View.
func (v *ActionsView) Help() []key.Binding { return []key.Binding{actionsKeys.run} }

// SelectionSummary feeds the status bar.
func (v *ActionsView) SelectionSummary() string {
	if item, ok := v.menu.Current(); ok {
		return item.Title
	}
	return ""
}

// Update implements View.
func (v *ActionsView) Update(message tea.Msg) (View, tea.Cmd) {
	keyMsg, ok := message.(tea.KeyMsg)
	if !ok {
		return v, nil
	}

	if key.Matches(keyMsg, actionsKeys.run) {
		item, selectable := v.menu.Current()
		if !selectable {
			if item.Badge == "PRO" {
				return v, toast(msg.Warn(item.Title + " requires OpenPort Pro"))
			}
			return v, nil
		}
		return v, emit(msg.RunActionMsg{ID: item.ID})
	}

	menu, cmd := v.menu.Update(keyMsg)
	v.menu = menu
	return v, cmd
}

// View implements View.
func (v *ActionsView) View(width, height int) string {
	return v.menu.View(width, height)
}
