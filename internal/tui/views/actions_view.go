package views

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/components"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// ActionsView is the discoverable menu of everything OpenPort can do. It and
// the command palette render the same catalog, so nothing is reachable from
// only one of them.
//
// It is a bubbles/list, which brings its own filtering, pagination and item
// counter — so a catalog longer than the window is paged rather than silently
// clipped, and `/` narrows it the same way it does everywhere else.
type ActionsView struct {
	theme   styles.Theme
	list    list.Model
	isPro   bool
	compact bool
}

// NewActionsView builds the menu. isPro decides whether Pro entries can be
// run; they are listed either way.
func NewActionsView(theme styles.Theme, isPro bool) *ActionsView {
	l := components.NewList(theme, ActionItems(isPro), false)
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetStatusBarItemName("action", "actions")

	return &ActionsView{theme: theme, list: l, isPro: isPro}
}

// ActionItems converts the action catalog into list items. Shared with the
// palette so both front doors stay in lockstep.
func ActionItems(isPro bool) []components.Item {
	all := actions.All()
	items := make([]components.Item, 0, len(all))
	for _, a := range all {
		item := components.Item{
			ID:    a.ID,
			Name:  a.Title,
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
	v.list.SetItems(components.ToListItems(ActionItems(isPro)))
}

// Title implements View.
func (v *ActionsView) Title() string { return "Actions" }

var actionsKeys = struct{ run, filter key.Binding }{
	run:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run action")),
	filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
}

// Help implements View.
func (v *ActionsView) Help() []key.Binding {
	return []key.Binding{actionsKeys.run, actionsKeys.filter}
}

// Current exposes the highlighted entry, for the status bar and for tests that
// walk the menu.
func (v *ActionsView) Current() (components.Item, bool) {
	return components.SelectedItem(v.list)
}

// CapturingInput reports whether the list's filter has the keyboard, so global
// single-letter bindings do not fire while the user is typing a query.
func (v *ActionsView) CapturingInput() bool { return v.list.SettingFilter() }

// SelectionSummary feeds the status bar.
func (v *ActionsView) SelectionSummary() string {
	if item, ok := v.list.SelectedItem().(components.Item); ok {
		return item.Name
	}
	return ""
}

// Update implements View.
func (v *ActionsView) Update(message tea.Msg) (View, tea.Cmd) {
	keyMsg, ok := message.(tea.KeyMsg)
	if !ok {
		return v, nil
	}

	filtering := v.list.SettingFilter()

	// Enter runs the highlighted action whether or not a search is in progress,
	// so filtering to an action and launching it is one uninterrupted move
	// rather than "commit the filter, then press enter again".
	if key.Matches(keyMsg, actionsKeys.run) {
		item, runnable := v.Current()
		if !runnable {
			if item.Badge == "PRO" {
				return v, toast(msg.Warn(item.Name + " requires OpenPort Pro"))
			}
			return v, nil
		}
		return v, emit(msg.RunActionMsg{ID: item.ID})
	}

	// bubbles/list disables its cursor keys while filtering; re-enable ↑/↓ so a
	// match can be picked without first committing the query.
	if filtering {
		switch keyMsg.String() {
		case "up":
			v.list.CursorUp()
			return v, nil
		case "down":
			v.list.CursorDown()
			return v, nil
		}
	}

	updated, cmd := v.list.Update(keyMsg)
	v.list = updated
	return v, cmd
}

// compactBelowHeight is where the stock two-line delegate costs more than the
// descriptions are worth: three rows per entry means a short window shows four
// actions, which reads as a much smaller catalog than it is.
const compactBelowHeight = 22

// View implements View.
func (v *ActionsView) View(width, height int) string {
	// A one-line header advertises the search up front, the same way the ports
	// and history screens do, so the filter is discoverable without hunting the
	// status bar for it.
	header := v.theme.Subtitle.Render("Everything OpenPort can do") +
		v.theme.Faint.Render("    press / to search")
	listHeight := height - 1
	if listHeight < 3 {
		listHeight = 3
	}

	// Swap the delegate rather than clipping: the list keeps its filtering,
	// pagination and counter either way, it just fits more per page.
	compact := listHeight < compactBelowHeight
	if compact != v.compact {
		v.compact = compact
		v.list.SetDelegate(components.NewDelegate(v.theme, compact))
	}

	v.list.SetSize(width, listHeight)
	return header + "\n" + v.list.View()
}
