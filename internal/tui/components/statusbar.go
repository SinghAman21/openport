package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// StatusBar is the bottom line: where you are, what is selected, what you can
// press, and whether watch mode is live.
type StatusBar struct {
	// Title is the active view's name.
	Title string
	// Selection summarises what the cursor is on.
	Selection string
	// Hints are the contextual bindings for the active view.
	Hints []key.Binding
	// Watching lights the live indicator.
	Watching bool
	// Pro shows the Pro badge when a licensed provider is active.
	Pro bool
	// Message overrides the hints with a transient line (used for toasts on
	// short terminals).
	Message string
	// Busy is the spinner plus a label for whatever is in flight. Async work
	// with no acknowledgement reads as a hang, and some of it — scanning for a
	// free port — genuinely takes a moment.
	Busy string
}

// View renders the bar to exactly width cells.
func (s StatusBar) View(theme styles.Theme, width int) string {
	left := theme.StatusKey.Render(s.Title)
	if s.Watching {
		left += "  " + theme.Success.Render(styles.GlyphWatching+" watching")
	}
	if s.Pro {
		left += "  " + theme.BadgePro.Render("PRO")
	}
	if s.Selection != "" {
		left += theme.StatusHint.Render("  " + styles.GlyphArrow + " " + s.Selection)
	}
	if s.Busy != "" {
		left += "  " + s.Busy
	}

	// The hints get whatever room is left after the left half. A view can
	// report a dozen bindings; showing the first few that fit beats showing
	// none, which is what a plain "does it all fit?" check would do.
	budget := width - lipgloss.Width(left) - 3
	right := s.Message
	if right == "" {
		right = renderHints(theme, s.Hints, budget)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		return theme.StatusBar.Render(truncate(left, width-2))
	}
	return theme.StatusBar.Render(left + strings.Repeat(" ", gap) + right)
}

// renderHints formats bindings as "key desc" pairs separated by dots, adding
// them only while they fit inside budget.
func renderHints(theme styles.Theme, hints []key.Binding, budget int) string {
	if budget < 6 {
		return ""
	}
	separator := theme.StatusHint.Render(" · ")
	sepWidth := lipgloss.Width(separator)

	var out string
	used := 0
	for _, h := range hints {
		if !h.Enabled() {
			continue
		}
		help := h.Help()
		if help.Key == "" {
			continue
		}
		part := theme.StatusKey.Render(help.Key) + " " + theme.StatusHint.Render(help.Desc)
		partWidth := lipgloss.Width(part)

		next := used + partWidth
		if out != "" {
			next += sepWidth
		}
		if next > budget {
			break
		}
		if out != "" {
			out += separator
		}
		out += part
		used = next
	}
	return out
}
