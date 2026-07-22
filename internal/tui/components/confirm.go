package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// Confirm is the yes/no gate in front of every destructive action.
//
// It is a huh confirm field, which already binds exactly the semantics this
// needs — y/Y accepts, n/N rejects, ←/→ toggles, enter submits — and defaults
// to the negative, so a stray enter can never kill anything.
type Confirm struct {
	id      string
	title   string
	payload any
	danger  bool

	form  *huh.Form
	value *bool
	done  bool
}

// NewConfirm builds a confirmation. payload is round-tripped back in the
// result so the opener can carry whatever context the follow-up needs.
func NewConfirm(theme styles.Theme, id, title, body, yes, no string, payload any, danger bool) *Confirm {
	value := new(bool)

	field := huh.NewConfirm().
		Key("confirm").
		Title(body).
		Affirmative(yes).
		Negative(no).
		Value(value)

	form := huh.NewForm(huh.NewGroup(field)).
		WithTheme(huhTheme()).
		WithKeyMap(escapableKeyMap()).
		WithShowHelp(true).
		WithShowErrors(false)

	return &Confirm{
		id: id, title: title, payload: payload, danger: danger,
		form: form, value: value,
	}
}

// Init focuses the field.
func (c *Confirm) Init() tea.Cmd { return c.form.Init() }

// Title implements ModalContent.
func (c *Confirm) Title() string { return c.title }

// Danger implements ModalContent.
func (c *Confirm) Danger() bool { return c.danger }

// Hint implements ModalContent; huh renders its own key help.
func (c *Confirm) Hint() string { return "" }

// Update resolves the confirmation once huh reports a terminal state.
func (c *Confirm) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	if c.done {
		return c, nil
	}

	model, cmd := c.form.Update(message)
	if form, ok := model.(*huh.Form); ok {
		c.form = form
	}

	switch c.form.State {
	case huh.StateCompleted:
		c.done = true
		id, payload, answer := c.id, c.payload, *c.value
		return c, tea.Batch(cmd, func() tea.Msg {
			return msg.ConfirmResultMsg{ID: id, Confirmed: answer, Payload: payload}
		})
	case huh.StateAborted:
		c.done = true
		return c, tea.Batch(cmd, func() tea.Msg { return msg.CloseOverlayMsg{} })
	}
	return c, cmd
}

// View renders the prompt and the two buttons.
func (c *Confirm) View(width, height int) string {
	c.form = c.form.WithWidth(width).WithHeight(max(height, 3))
	return c.form.View()
}
