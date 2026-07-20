package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// ModalContent is anything a modal can wrap: a form, a confirmation, the help
// overlay, or a plain info panel.
//
// View is handed both dimensions because content is responsible for fitting
// the space it is given. A popup that renders taller than the window loses its
// own footer — which is exactly where the keys that dismiss it are documented,
// so overflow strands the user rather than merely looking untidy.
type ModalContent interface {
	Update(tea.Msg) (ModalContent, tea.Cmd)
	View(width, height int) string
	Title() string
	// Danger styles the border in the danger colour, for destructive actions.
	Danger() bool
	// Hint is the content-specific key help shown in the modal footer, after
	// the universal "esc close".
	Hint() string
}

// Modal chrome costs, in rows: border top/bottom, padding top/bottom, the
// title and its margin, and the footer with its blank line.
const (
	modalChromeTall  = 8
	modalChromeShort = 5 // padding drops to zero on a short terminal
)

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

// View renders the bordered overlay, sized to fit inside width x height.
func (m *Modal) View(width, height int) string {
	if width < 4 || height < 4 {
		return ""
	}

	style := m.theme.Modal
	if m.content.Danger() {
		style = m.theme.ModalDanger
	}

	// On a short terminal, reclaim the vertical padding — losing two rows of
	// whitespace is a better trade than losing two rows of content.
	chrome := modalChromeTall
	if height < 16 {
		style = style.Padding(0, 2)
		chrome = modalChromeShort
	}

	inner := modalWidth(width)
	body := max(height-chrome, 1)

	rendered := m.theme.ModalTitle.Render(truncate(m.content.Title(), inner)) + "\n" +
		m.content.View(inner, body) + "\n\n" +
		m.footer(inner)

	out := style.Width(inner).Render(rendered)

	// Backstop: whatever the content did, the popup never exceeds the window.
	return clampBlock(out, width, height)
}

// modalWidth picks a comfortable reading width that still fits the terminal.
func modalWidth(width int) int {
	inner := width * 2 / 3
	if inner < 40 {
		inner = min(width-4, 40)
	}
	if inner > 72 {
		inner = 72
	}
	return max(inner, 16)
}

// footer always leads with the way out. It is rendered unconditionally, and
// the height budget above reserves room for it, so there is no terminal size
// at which the user is left guessing how to dismiss what they opened.
func (m *Modal) footer(width int) string {
	parts := []string{m.theme.StatusKey.Render("esc") + " " + m.theme.StatusHint.Render("close")}
	if hint := m.content.Hint(); hint != "" {
		parts = append(parts, m.theme.StatusHint.Render(hint))
	}
	return truncate(strings.Join(parts, m.theme.StatusHint.Render(" · ")), width)
}

// clampBlock trims a rendered block to fit a box.
func clampBlock(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = truncate(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
