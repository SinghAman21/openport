package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// Menu is the list-style picker used by the actions view and by any row-action
// popover. It handles motion only; the owning view decides what Enter means.
type Menu struct {
	items  []Item
	cursor int
	theme  styles.Theme
	offset int
}

// NewMenu builds a menu over items, placing the cursor on the first selectable
// entry.
func NewMenu(theme styles.Theme, items []Item) *Menu {
	m := &Menu{items: items, theme: theme}
	m.cursor = m.nextSelectable(-1, 1)
	return m
}

// SetItems replaces the contents, keeping the cursor in range.
func (m *Menu) SetItems(items []Item) {
	m.items = items
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	if m.cursor < 0 || (m.cursor < len(items) && items[m.cursor].Disabled) {
		m.cursor = m.nextSelectable(-1, 1)
	}
}

// Items returns the current contents.
func (m *Menu) Items() []Item { return m.items }

// Current returns the highlighted item.
func (m *Menu) Current() (Item, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return Item{}, false
	}
	it := m.items[m.cursor]
	if it.Disabled {
		return it, false
	}
	return it, true
}

// Cursor reports the highlighted index.
func (m *Menu) Cursor() int { return m.cursor }

// menuKeys are the motions a menu understands. Arrows and Vim keys both.
var menuKeys = struct {
	up, down, top, bottom key.Binding
}{
	up:     key.NewBinding(key.WithKeys("up", "k")),
	down:   key.NewBinding(key.WithKeys("down", "j")),
	top:    key.NewBinding(key.WithKeys("g", "home")),
	bottom: key.NewBinding(key.WithKeys("G", "end")),
}

// Update handles cursor motion.
func (m *Menu) Update(message tea.Msg) (*Menu, tea.Cmd) {
	keyMsg, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, menuKeys.up):
		m.cursor = m.nextSelectable(m.cursor, -1)
	case key.Matches(keyMsg, menuKeys.down):
		m.cursor = m.nextSelectable(m.cursor, 1)
	case key.Matches(keyMsg, menuKeys.top):
		m.cursor = m.nextSelectable(-1, 1)
	case key.Matches(keyMsg, menuKeys.bottom):
		m.cursor = m.nextSelectable(len(m.items), -1)
	}
	return m, nil
}

// nextSelectable walks from idx in the given direction to the next enabled
// item, stopping at the ends rather than wrapping.
func (m *Menu) nextSelectable(idx, dir int) int {
	for i := idx + dir; i >= 0 && i < len(m.items); i += dir {
		if !m.items[i].Disabled {
			return i
		}
	}
	if idx >= 0 && idx < len(m.items) && !m.items[idx].Disabled {
		return idx // no move possible; stay put
	}
	// Fall back to the first selectable item anywhere in the list.
	for i := range m.items {
		if !m.items[i].Disabled {
			return i
		}
	}
	return -1
}

// View renders the menu, scrolling to keep the cursor visible.
func (m *Menu) View(width, height int) string {
	if len(m.items) == 0 {
		return m.theme.Faint.Render("Nothing here yet.")
	}
	m.scrollTo(height)

	var b strings.Builder
	lastGroup := ""
	rendered := 0

	for i := m.offset; i < len(m.items) && rendered < height; i++ {
		it := m.items[i]
		if it.Group != "" && it.Group != lastGroup {
			if rendered > 0 {
				b.WriteString("\n")
				rendered++
			}
			b.WriteString(m.theme.Subtitle.Render(strings.ToUpper(it.Group)) + "\n")
			lastGroup = it.Group
			rendered++
		}
		b.WriteString(m.renderRow(it, i == m.cursor, width) + "\n")
		rendered++
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Menu) renderRow(it Item, selected bool, width int) string {
	cursor := "  "
	title := it.Title

	switch {
	case it.Disabled:
		title = m.theme.Faint.Render(title)
	case selected:
		cursor = m.theme.Accent.Render(styles.GlyphArrow + " ")
		title = m.theme.Selected.Render(title)
	default:
		title = m.theme.Body.Render(title)
	}

	left := cursor + title
	if it.Badge != "" {
		left += " " + m.theme.BadgePro.Render(it.Badge)
	}
	if it.Desc != "" {
		left += "  " + m.theme.Faint.Render(it.Desc)
	}

	if it.Key == "" {
		return truncate(left, width)
	}
	hint := m.theme.StatusKey.Render(it.Key)
	gap := width - lipgloss.Width(left) - lipgloss.Width(hint)
	if gap < 1 {
		return truncate(left, width)
	}
	return left + strings.Repeat(" ", gap) + hint
}

// scrollTo keeps the cursor inside the visible window.
func (m *Menu) scrollTo(height int) {
	if height <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	// Group headers consume rows too, so leave a little slack.
	if m.cursor >= m.offset+height-2 {
		m.offset = m.cursor - height + 3
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// truncate clips a styled string to width, appending an ellipsis when it does
// not fit.
func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	// Walk back until it fits; styled strings make byte math unreliable.
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
