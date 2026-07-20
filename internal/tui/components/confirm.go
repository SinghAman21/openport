package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// Confirm is the yes/no gate in front of every destructive action. It defaults
// to "no", so a stray Enter can never kill anything.
type Confirm struct {
	id      string
	title   string
	body    string
	yesText string
	noText  string
	payload any
	danger  bool
	yes     bool
	done    bool
	theme   styles.Theme
}

// NewConfirm builds a confirmation. payload is round-tripped back in the
// result so the opener can carry whatever context the follow-up needs.
func NewConfirm(theme styles.Theme, id, title, body string, payload any, danger bool) *Confirm {
	return &Confirm{
		id:      id,
		title:   title,
		body:    body,
		yesText: "Yes",
		noText:  "Cancel",
		payload: payload,
		danger:  danger,
		theme:   theme,
	}
}

// WithLabels overrides the button text, e.g. "Kill" / "Keep running".
func (c *Confirm) WithLabels(yes, no string) *Confirm {
	c.yesText, c.noText = yes, no
	return c
}

// Title implements ModalContent.
func (c *Confirm) Title() string { return c.title }

// Danger implements ModalContent.
func (c *Confirm) Danger() bool { return c.danger }

var confirmKeys = struct {
	left, right, yes, no, accept, cancel key.Binding
}{
	left:   key.NewBinding(key.WithKeys("left", "h", "shift+tab")),
	right:  key.NewBinding(key.WithKeys("right", "l", "tab")),
	yes:    key.NewBinding(key.WithKeys("y", "Y")),
	no:     key.NewBinding(key.WithKeys("n", "N")),
	accept: key.NewBinding(key.WithKeys("enter")),
	cancel: key.NewBinding(key.WithKeys("esc")),
}

// Update handles selection and resolution.
func (c *Confirm) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	if c.done {
		return c, nil
	}
	keyMsg, ok := message.(tea.KeyMsg)
	if !ok {
		return c, nil
	}

	switch {
	case key.Matches(keyMsg, confirmKeys.left):
		c.yes = true
	case key.Matches(keyMsg, confirmKeys.right):
		c.yes = false
	case key.Matches(keyMsg, confirmKeys.yes):
		return c, c.resolve(true)
	case key.Matches(keyMsg, confirmKeys.no), key.Matches(keyMsg, confirmKeys.cancel):
		return c, c.resolve(false)
	case key.Matches(keyMsg, confirmKeys.accept):
		return c, c.resolve(c.yes)
	}
	return c, nil
}

func (c *Confirm) resolve(answer bool) tea.Cmd {
	c.done = true
	id, payload := c.id, c.payload
	return func() tea.Msg {
		return msg.ConfirmResultMsg{ID: id, Confirmed: answer, Payload: payload}
	}
}

// View renders the prompt and the two buttons.
func (c *Confirm) View(width int) string {
	var b strings.Builder
	if c.body != "" {
		b.WriteString(wrap(c.body, width))
		b.WriteString("\n\n")
	}

	active := c.theme.Selected
	if c.danger {
		active = c.theme.Danger.Bold(true)
	}

	yes, no := "  "+c.yesText+"  ", "  "+c.noText+"  "
	if c.yes {
		b.WriteString(active.Render("["+strings.TrimSpace(yes)+"]") + "   " +
			c.theme.Faint.Render(strings.TrimSpace(no)))
	} else {
		b.WriteString(c.theme.Faint.Render(strings.TrimSpace(yes)) + "   " +
			c.theme.Selected.Render("["+strings.TrimSpace(no)+"]"))
	}

	b.WriteString("\n\n")
	b.WriteString(c.theme.Faint.Render("y/n · ←/→ to choose · enter to accept · esc to cancel"))
	return b.String()
}

// wrap hard-wraps plain text to width, preserving explicit newlines.
func wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			switch {
			case line == "":
				line = word
			case len(line)+1+len(word) <= width:
				line += " " + word
			default:
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
