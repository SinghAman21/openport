package components

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// HelpSection is one titled block of bindings in the help overlay.
type HelpSection struct {
	Title    string
	Bindings []key.Binding
}

// Help is the '?' overlay: bubbles/help for the rendering, bubbles/viewport
// for the scrolling. The full keymap runs past forty rows, which does not fit
// an 80x24 terminal, and clipping it would hide the bindings the overlay
// exists to teach.
type Help struct {
	sections []HelpSection
	help     help.Model
	viewport viewport.Model
	theme    styles.Theme
	ready    bool
}

// NewHelp builds the overlay.
func NewHelp(theme styles.Theme, sections []HelpSection) *Help {
	h := help.New()
	h.ShowAll = true
	h.Styles.FullKey = h.Styles.FullKey.Foreground(styles.Accent).Bold(true)
	h.Styles.FullDesc = h.Styles.FullDesc.Foreground(styles.Text)
	h.Styles.FullSeparator = h.Styles.FullSeparator.Foreground(styles.Subtle)

	return &Help{sections: sections, help: h, viewport: viewport.New(0, 0), theme: theme}
}

// Title implements ModalContent.
func (h *Help) Title() string { return "Keyboard shortcuts" }

// Danger implements ModalContent.
func (h *Help) Danger() bool { return false }

// Hint implements ModalContent, advertising the scroll keys only when there is
// something below the fold.
func (h *Help) Hint() string {
	if h.ready && h.viewport.TotalLineCount() > h.viewport.Height {
		return "↑/↓ scroll"
	}
	return ""
}

var helpCloseKeys = key.NewBinding(key.WithKeys("esc", "?", "q"))

// Update scrolls through the viewport, or closes the overlay.
func (h *Help) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	if keyMsg, ok := message.(tea.KeyMsg); ok && key.Matches(keyMsg, helpCloseKeys) {
		return h, func() tea.Msg { return msg.CloseOverlayMsg{} }
	}
	vp, cmd := h.viewport.Update(message)
	h.viewport = vp
	return h, cmd
}

// View renders the bindings, scrolled to the current offset.
func (h *Help) View(width, height int) string {
	h.help.Width = width
	h.viewport.Width, h.viewport.Height = width, height

	// One help.Model render per section keeps each group under its own
	// heading; help.FullHelpView alone lays columns out but has no notion of
	// a titled block.
	var content string
	for i, section := range h.sections {
		if len(section.Bindings) == 0 {
			continue
		}
		if i > 0 {
			content += "\n\n"
		}
		content += h.theme.Subtitle.Render(section.Title) + "\n" +
			h.help.FullHelpView(columns(section.Bindings, width))
	}
	if content == "" {
		content = h.theme.Faint.Render("No bindings.")
	}

	h.viewport.SetContent(content)
	h.ready = true
	return h.viewport.View()
}

// minColumnWidth is the room one "key  description" pair needs before the
// column becomes unreadable.
const minColumnWidth = 26

// columns splits bindings into the column groups help.FullHelpView expects.
//
// The count is derived from the available width: FullHelpView joins its groups
// horizontally without reflowing, so asking for more columns than fit wraps
// entries mid-pair and corrupts the whole block.
func columns(bindings []key.Binding, width int) [][]key.Binding {
	count := width / minColumnWidth
	if count < 1 {
		count = 1
	}
	if count > len(bindings) {
		count = len(bindings)
	}
	if count == 0 {
		return nil
	}

	// Spread evenly so the last column is not left with a single entry.
	perColumn := (len(bindings) + count - 1) / count

	out := make([][]key.Binding, 0, count)
	for start := 0; start < len(bindings); start += perColumn {
		end := start + perColumn
		if end > len(bindings) {
			end = len(bindings)
		}
		out = append(out, bindings[start:end])
	}
	return out
}
