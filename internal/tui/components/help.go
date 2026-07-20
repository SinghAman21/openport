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
type Help struct {
	sections []HelpSection
	theme    styles.Theme
}

// NewHelp builds the overlay.
func NewHelp(theme styles.Theme, sections []HelpSection) *Help {
	return &Help{sections: sections, theme: theme}
}

// Title implements ModalContent.
func (h *Help) Title() string { return "Keyboard shortcuts" }

// Danger implements ModalContent.
func (h *Help) Danger() bool { return false }

var helpKeys = key.NewBinding(key.WithKeys("esc", "?", "q"))

// Update closes the overlay on esc, '?' or q.
func (h *Help) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	if keyMsg, ok := message.(tea.KeyMsg); ok && key.Matches(keyMsg, helpKeys) {
		return h, func() tea.Msg { return msg.CloseOverlayMsg{} }
	}
	return h, nil
}

// View renders the sections in two columns where there is room.
func (h *Help) View(width int) string {
	var blocks []string
	for _, s := range h.sections {
		if len(s.Bindings) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteString(h.theme.Subtitle.Render(strings.ToUpper(s.Title)))
		b.WriteString("\n")
		for _, bind := range s.Bindings {
			help := bind.Help()
			if help.Key == "" {
				continue
			}
			b.WriteString(h.theme.StatusKey.Render(pad(help.Key, 10)))
			b.WriteString(h.theme.Body.Render(help.Desc))
			b.WriteString("\n")
		}
		blocks = append(blocks, strings.TrimRight(b.String(), "\n"))
	}

	if len(blocks) == 0 {
		return h.theme.Faint.Render("No bindings.")
	}

	// Two columns when the modal is wide enough, otherwise stack.
	if width >= 56 && len(blocks) > 1 {
		mid := (len(blocks) + 1) / 2
		left := lipgloss.JoinVertical(lipgloss.Left, blocks[:mid]...)
		right := lipgloss.JoinVertical(lipgloss.Left, blocks[mid:]...)
		colWidth := width/2 - 1
		return lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(colWidth).Render(left),
			lipgloss.NewStyle().Width(colWidth).Render(right),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// pad right-pads s to n display cells.
func pad(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s + " "
}
