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

// setRows installs rows and keeps the cursor on a real row.
//
// bubbles/table starts its cursor at 0, but the first time it is handed an empty
// set, SetRows clamps the cursor to len(rows)-1 = -1 — and never lifts it back
// when rows later arrive. A view that renders empty before its data loads (the
// blueprints table rebuilds on the first ports poll, say) is left with a cursor
// of -1 forever, so every selection-driven key — edit, delete, apply — silently
// does nothing. Nudging the cursor back onto row 0 is what keeps those keys live.
func setRows(t *table.Model, rows []table.Row) {
	t.SetRows(rows)
	if len(rows) > 0 && t.Cursor() < 0 {
		t.SetCursor(0)
	}
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
