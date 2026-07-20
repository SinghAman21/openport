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

// NewMenu builds a menu over items.
func NewMenu(theme styles.Theme, items []Item) *Menu {
	return &Menu{items: items, theme: theme}
}

// SetItems replaces the contents, keeping the cursor in range.
func (m *Menu) SetItems(items []Item) {
	m.items = items
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
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
	// The cursor moves over disabled entries rather than skipping them. Off
	// Pro those sit at the end of the catalog, and skipping meant the cursor
	// stopped short of them — so on a short terminal the list never scrolled
	// far enough to show they existed at all. They are still not runnable;
	// Current reports that separately, and the caller explains why.
	switch {
	case key.Matches(keyMsg, menuKeys.up):
		m.cursor--
	case key.Matches(keyMsg, menuKeys.down):
		m.cursor++
	case key.Matches(keyMsg, menuKeys.top):
		m.cursor = 0
	case key.Matches(keyMsg, menuKeys.bottom):
		m.cursor = len(m.items) - 1
	}

	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	return m, nil
}

// View renders the menu, scrolling to keep the cursor visible.
//
// Rows are flattened first — group headers included — so scrolling is exact.
// Computing the window against item indices while rendering extra header rows
// is what makes a menu drift out of step with its own cursor and appear to
// swallow entries.
func (m *Menu) View(width, height int) string {
	if len(m.items) == 0 {
		return m.theme.Faint.Render("Nothing here yet.")
	}
	if height < 1 {
		height = 1
	}

	lines, cursorLine := m.render(width)

	// When the list overflows, spend a row saying so. A menu that silently
	// shows six of forty entries reads as a menu with six entries.
	overflow := len(lines) > height
	visible := height
	if overflow {
		visible = height - 1
	}
	if visible < 1 {
		visible = 1
	}

	if cursorLine < m.offset {
		m.offset = cursorLine
	}
	if cursorLine >= m.offset+visible {
		m.offset = cursorLine - visible + 1
	}
	if maxOffset := len(lines) - visible; m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}

	end := min(m.offset+visible, len(lines))
	out := append([]string(nil), lines[m.offset:end]...)

	if overflow {
		out = append(out, m.scrollIndicator(len(lines), visible))
	}
	return strings.Join(out, "\n")
}

// render flattens items and headers into rows, reporting which row the cursor
// is on.
func (m *Menu) render(width int) (lines []string, cursorLine int) {
	lastGroup := ""
	for i, it := range m.items {
		if it.Group != "" && it.Group != lastGroup {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, m.theme.Subtitle.Render(strings.ToUpper(it.Group)))
			lastGroup = it.Group
		}
		if i == m.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.renderRow(it, i == m.cursor, width))
	}
	return lines, cursorLine
}

// scrollIndicator reports what is out of view in both directions.
func (m *Menu) scrollIndicator(total, visible int) string {
	above := m.offset
	below := total - m.offset - visible
	if below < 0 {
		below = 0
	}

	var parts []string
	if above > 0 {
		parts = append(parts, "↑ "+itoa(above)+" above")
	}
	if below > 0 {
		parts = append(parts, "↓ "+itoa(below)+" below")
	}
	return m.theme.Faint.Render(strings.Join(parts, "   "))
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
