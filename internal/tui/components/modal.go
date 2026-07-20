package components

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// ModalContent is anything a modal can wrap: a form, a confirmation, the help
// overlay, or a plain info panel.
type ModalContent interface {
	Update(tea.Msg) (ModalContent, tea.Cmd)
	View(width int) string
	Title() string
	// Danger styles the modal border in the danger colour, used for anything
	// destructive.
	Danger() bool
}

// Modal is the z-ordered overlay wrapper. Exactly one may be open at a time,
// and while it is open it consumes input before every other layer.
type Modal struct {
	content ModalContent
	theme   styles.Theme
}

// NewModal wraps content in a modal.
func NewModal(theme styles.Theme, content ModalContent) *Modal {
	return &Modal{content: content, theme: theme}
}

// Update forwards to the wrapped content.
func (m *Modal) Update(message tea.Msg) (*Modal, tea.Cmd) {
	content, cmd := m.content.Update(message)
	m.content = content
	return m, cmd
}

// Content exposes the wrapped content for type assertions.
func (m *Modal) Content() ModalContent { return m.content }

// View renders the bordered overlay box.
func (m *Modal) View(width, height int) string {
	inner := width * 2 / 3
	if inner < 40 {
		inner = min(width-4, 40)
	}
	if inner > 72 {
		inner = 72
	}
	if inner < 20 {
		inner = 20
	}

	body := m.theme.ModalTitle.Render(m.content.Title()) + "\n" + m.content.View(inner)

	style := m.theme.Modal
	if m.content.Danger() {
		style = m.theme.ModalDanger
	}
	return style.Width(inner).Render(body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
