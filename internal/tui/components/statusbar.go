package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// StatusBar is the bottom line: where you are, what is selected, what is in
// flight, and what you can press.
//
// The key hints come from bubbles/help in its short form, which already knows
// how to fit a binding list into a given width and elide the rest.
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

	right := s.Message
	if right == "" {
		right = renderHints(s.Hints, width-lipgloss.Width(left)-3)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		return theme.StatusBar.Render(truncate(left, width-2))
	}
	return theme.StatusBar.Render(left + strings.Repeat(" ", gap) + right)
}

// renderHints defers to bubbles/help for the "key desc · key desc" line, which
// already knows how to fit bindings into a budget and drop the overflow.
func renderHints(hints []key.Binding, budget int) string {
	if budget < 6 || len(hints) == 0 {
		return ""
	}
	h := help.New()
	h.Width = budget
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(styles.Accent).Bold(true)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(styles.Subtle)
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Foreground(styles.Subtle)
	return h.ShortHelpView(hints)
}

// truncate clips a styled string to width, appending an ellipsis when it does
// not fit.
func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
