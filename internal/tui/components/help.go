package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// HelpSection is one titled block of bindings in the help overlay.
type HelpSection struct {
	Title    string
	Bindings []key.Binding
}

// Help is the '?' overlay. It is built from the global keymap plus whatever
// the active view reports from its Help() method, so the overlay is always
// accurate for wherever the user currently is.
//
// It scrolls: the full keymap runs to about forty rows, which does not fit an
// 80x24 terminal, and silently clipping the bottom would hide bindings the
// overlay exists to teach.
type Help struct {
	sections []HelpSection
	theme    styles.Theme
	offset   int
	overflow bool
}

// NewHelp builds the overlay.
func NewHelp(theme styles.Theme, sections []HelpSection) *Help {
	return &Help{sections: sections, theme: theme}
}

// Title implements ModalContent.
func (h *Help) Title() string { return "Keyboard shortcuts" }

// Danger implements ModalContent.
func (h *Help) Danger() bool { return false }

// Hint implements ModalContent, advertising the scroll keys only when there is
// something below the fold.
func (h *Help) Hint() string {
	if h.overflow {
		return "↑/↓ scroll"
	}
	return ""
}

var helpKeys = struct{ up, down, close key.Binding }{
	up:    key.NewBinding(key.WithKeys("up", "k", "pgup")),
	down:  key.NewBinding(key.WithKeys("down", "j", "pgdown", " ")),
	close: key.NewBinding(key.WithKeys("esc", "?", "q")),
}

// Update scrolls, or closes the overlay.
func (h *Help) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	keyMsg, ok := message.(tea.KeyMsg)
	if !ok {
		return h, nil
	}
	switch {
	case key.Matches(keyMsg, helpKeys.close):
		return h, func() tea.Msg { return msg.CloseOverlayMsg{} }
	case key.Matches(keyMsg, helpKeys.up):
		if h.offset > 0 {
			h.offset--
		}
	case key.Matches(keyMsg, helpKeys.down):
		h.offset++
	}
	return h, nil
}

// View renders as many bindings as fit, scrolled to the current offset.
func (h *Help) View(width, height int) string {
	lines := h.lines(width)

	if height < 1 {
		height = 1
	}
	h.overflow = len(lines) > height

	// Clamp the offset so scrolling cannot run past the end and leave the
	// overlay looking empty.
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if h.offset > maxOffset {
		h.offset = maxOffset
	}

	end := min(h.offset+height, len(lines))
	visible := lines[h.offset:end]

	// Pad so the footer stays pinned rather than jumping as you scroll.
	for len(visible) < height {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n")
}

// lines flattens the sections into individually renderable rows, which is what
// makes scrolling by row possible.
func (h *Help) lines(width int) []string {
	var out []string
	for _, s := range h.sections {
		if len(s.Bindings) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, h.theme.Subtitle.Render(strings.ToUpper(s.Title)))

		for _, bind := range s.Bindings {
			help := bind.Help()
			if help.Key == "" {
				continue
			}
			row := h.theme.StatusKey.Render(pad(help.Key, 12)) + h.theme.Body.Render(help.Desc)
			out = append(out, truncate(row, width))
		}
	}
	if len(out) == 0 {
		return []string{h.theme.Faint.Render("No bindings.")}
	}
	return out
}

// pad right-pads s to n display cells.
func pad(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s + " "
}
