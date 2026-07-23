package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// paletteRows is how many matches the palette shows at once.
const paletteRows = 10

// Palette is the fuzzy quick-action launcher opened with ':' or ctrl+k.
//
// It is a bubbles/list held permanently in its filtering state: the list's own
// filter is a fuzzy matcher over FilterValue, which is exactly what a command
// palette is. Opening it starts filtering, so every keystroke goes to the
// query without the caller routing anything.
type Palette struct {
	list  list.Model
	theme styles.Theme
	open  bool
}

// NewPalette builds a closed palette over the given items.
func NewPalette(theme styles.Theme, items []Item) *Palette {
	l := NewList(theme, items, true)
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetStatusBarItemName("action", "actions")
	return &Palette{list: l, theme: theme}
}

// SetItems replaces the searchable set, e.g. when Pro gating changes.
func (p *Palette) SetItems(items []Item) {
	p.list.SetItems(ToListItems(items))
}

// Open focuses the palette and clears any previous query.
func (p *Palette) Open() tea.Cmd {
	p.open = true
	p.list.ResetFilter()
	p.list.SetFilterState(list.Filtering)
	return nil
}

// Close hides the palette.
func (p *Palette) Close() {
	p.open = false
	p.list.ResetFilter()
}

// IsOpen reports whether the palette is capturing input.
func (p *Palette) IsOpen() bool { return p.open }

// Current returns the highlighted match.
func (p *Palette) Current() (Item, bool) { return SelectedItem(p.list) }

// Update forwards to the list, which owns typing, filtering and motion.
func (p *Palette) Update(message tea.Msg) (*Palette, tea.Cmd) {
	if !p.open {
		return p, nil
	}

	// The palette lives permanently in the list's Filtering state, where
	// bubbles/list disables its own cursor keys so keystrokes narrow the query.
	// Drive the selection manually so ↑/↓ pick a match while you keep typing —
	// otherwise the arrows are dead exactly when they are most wanted.
	if keyMsg, ok := message.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			p.list.CursorUp()
			return p, nil
		case "down":
			p.list.CursorDown()
			return p, nil
		}
	}

	updated, cmd := p.list.Update(message)
	p.list = updated

	// The list leaves filtering when the query is accepted; put it straight
	// back so the palette keeps behaving like a search box.
	if p.list.FilterState() == list.FilterApplied {
		p.list.SetFilterState(list.Filtering)
	}
	return p, cmd
}

// View renders the palette overlay.
func (p *Palette) View(width int) string {
	inner := width - 4
	if inner < 24 {
		inner = 24
	}
	p.list.SetSize(inner, paletteRows)
	return p.theme.Palette.Width(inner).Render(p.list.View())
}
