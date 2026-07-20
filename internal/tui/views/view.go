// Package views holds the top-level screens. Adding a screen means writing one
// file here that satisfies View, plus a Route entry and a line in the root
// model's view map.
//
// Views never import internal/platform and never call the OS. They render the
// data they are handed and emit intents from the msg package; the root model
// turns those intents into service calls.
package views

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// View is the contract every screen implements.
type View interface {
	// Update handles a message and may emit intents. It returns itself so
	// views can be value types if they prefer.
	Update(msg tea.Msg) (View, tea.Cmd)
	// View renders the screen body — the area between the header and the
	// status bar.
	View(width, height int) string
	// Help feeds both the help overlay and the status-bar hints, so a view's
	// bindings are documented in exactly one place.
	Help() []key.Binding
	// Title names the screen in the header and status bar.
	Title() string
}

// fitTable sizes a table to the available width, scaling its columns down
// proportionally when they do not fit. bubbles/table does not reflow on its
// own, so without this a fixed column set renders wider than the terminal in a
// split pane.
func fitTable(t *table.Model, base []table.Column, width, height int) {
	cols := make([]table.Column, len(base))
	copy(cols, base)

	used := 0
	for _, c := range cols {
		used += c.Width + cellPadding
	}
	if used > width {
		content := used - len(cols)*cellPadding
		available := width - len(cols)*cellPadding
		if available < len(cols) {
			available = len(cols)
		}
		if content > 0 {
			for i := range cols {
				scaled := cols[i].Width * available / content
				if scaled < 1 {
					scaled = 1
				}
				cols[i].Width = scaled
			}
		}
	}

	t.SetColumns(cols)
	t.SetWidth(width)
	t.SetHeight(height)
}

// tabBar renders a simple left-aligned tab strip, used by the views that have
// more than one pane.
func tabBar(active int, labels []string, activeStyle, inactiveStyle interface {
	Render(...string) string
}) string {
	out := ""
	for i, label := range labels {
		if i == active {
			out += activeStyle.Render(label)
		} else {
			out += inactiveStyle.Render(label)
		}
	}
	return out
}
