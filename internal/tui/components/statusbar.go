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
//
// Every segment bakes the bar surface colour into its own style. Lip Gloss
// does not repaint a background across the reset codes that nested, pre-styled
// segments emit, so a single outer wrapper would leave the bar full of holes;
// painting each piece (and the connective spacing) is what keeps it solid.
func (s StatusBar) View(theme styles.Theme, width int) string {
	// The active view name is a filled "mode" segment, the way an editor names
	// its mode in the corner — an immediate, always-present anchor for "where am
	// I" that the eye finds without reading.
	left := theme.StatusMode.Render(s.Title)
	if s.Watching {
		left += theme.BarSeg.Render("  ") +
			theme.BarSeg.Foreground(styles.Success).Render(styles.GlyphWatching+" watching")
	}
	if s.Pro {
		left += theme.BarSeg.Render("  ") + theme.BadgePro.Render("PRO")
	}
	if s.Selection != "" {
		left += theme.BarHint.Render("  " + styles.GlyphArrow + " " + s.Selection)
	}
	if s.Busy != "" {
		left += theme.BarSeg.Render("  " + s.Busy)
	}

	right := s.Message
	if right == "" {
		right = renderHints(s.Hints, width-lipgloss.Width(left)-3)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		// Too tight for the fill: keep the mode segment legible and let the bar
		// surface run to the edge.
		content := truncate(left, width-1)
		return content + theme.BarFill.Render(strings.Repeat(" ", max0(width-lipgloss.Width(content))))
	}
	return theme.BarSeg.Render(" ") + left +
		theme.BarFill.Render(strings.Repeat(" ", gap)) +
		right + theme.BarSeg.Render(" ")
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// renderHints defers to bubbles/help for the "key desc · key desc" line, which
// already knows how to fit bindings into a budget and drop the overflow.
func renderHints(hints []key.Binding, budget int) string {
	if budget < 6 || len(hints) == 0 {
		return ""
	}
	h := help.New()
	h.Width = budget
	// Paint the bar surface behind the hints too, so the right end of the strip
	// stays a continuous fill instead of hints on the bare terminal background.
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(styles.Accent).Background(styles.BarBg).Bold(true)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(styles.BarText).Background(styles.BarBg)
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Foreground(styles.Subtle).Background(styles.BarBg)
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
