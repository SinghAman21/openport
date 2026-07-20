// Package components wires the Charm widgets — bubbles/list, bubbles/help,
// bubbles/viewport, huh — into the shapes this app needs.
//
// It deliberately contains no hand-rolled widgets. Anything that behaves like
// a list, a form, a confirmation or a help screen is an upstream component
// configured here, so the behaviour users already know (fuzzy filtering, the
// pagination dots, y/n on a confirm) comes for free and stays consistent.
//
// Two things have no upstream equivalent and are therefore still local: the
// modal frame, which is z-ordering rather than a widget, and the toast stack.
package components

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// Item adapts an action to bubbles/list.
//
// list.DefaultItem is what the stock delegate renders, and FilterValue is what
// its fuzzy filter matches against — the same sahilm/fuzzy the palette used to
// call directly.
type Item struct {
	ID    string
	Name  string
	Desc  string
	Group string
	// Key is the direct binding, surfaced in the description so the stock
	// delegate can advertise it without a bespoke renderer.
	Key string
	// Badge marks an entry that exists but cannot be run, e.g. "PRO".
	Badge string
	// Disabled entries stay listed and filterable; the caller explains why on
	// selection rather than hiding them.
	Disabled bool
}

// Title implements list.DefaultItem.
func (i Item) Title() string {
	if i.Badge != "" {
		return i.Name + "  " + i.Badge
	}
	return i.Name
}

// Description implements list.DefaultItem.
func (i Item) Description() string {
	out := i.Group
	if i.Desc != "" {
		out += " · " + i.Desc
	}
	if i.Key != "" {
		out += "  [" + i.Key + "]"
	}
	return out
}

// FilterValue implements list.Item. Category and description are included so
// "kill", "ports" and "terminate" all find the same entry.
func (i Item) FilterValue() string { return i.Name + " " + i.Group + " " + i.Desc }

// ToListItems converts items for bubbles/list.
func ToListItems(items []Item) []list.Item {
	out := make([]list.Item, len(items))
	for i, it := range items {
		out[i] = it
	}
	return out
}

// NewDelegate builds the themed stock delegate. compact drops the description
// line and the spacing, which triples how many entries fit — the right trade
// for the palette, where the query is the context, and for short windows.
func NewDelegate(theme styles.Theme, compact bool) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = !compact
	if compact {
		delegate.SetSpacing(0)
		delegate.SetHeight(1)
	}

	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Accent).BorderForeground(styles.Accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Muted).BorderForeground(styles.Accent)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(styles.Text)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(styles.Subtle)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.Foreground(styles.Subtle)
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.Foreground(styles.Subtle)
	return delegate
}

// NewList builds a themed bubbles/list.
func NewList(theme styles.Theme, items []Item, compact bool) list.Model {
	l := list.New(ToListItems(items), NewDelegate(theme, compact), 0, 0)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // the status bar and help overlay carry the keys
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(styles.Subtle).Padding(0, 2)
	l.Styles.StatusBar = l.Styles.StatusBar.Foreground(styles.Muted)
	l.Styles.StatusEmpty = l.Styles.StatusEmpty.Foreground(styles.Subtle)
	l.FilterInput.Prompt = "/"
	l.FilterInput.PromptStyle = l.FilterInput.PromptStyle.Foreground(styles.Accent)
	return l
}

// SelectedItem pulls the concrete Item out of a list, reporting whether it can
// actually be run.
func SelectedItem(l list.Model) (Item, bool) {
	selected, ok := l.SelectedItem().(Item)
	if !ok {
		return Item{}, false
	}
	return selected, !selected.Disabled
}
