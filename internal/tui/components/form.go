package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/SinghAman21/openport/internal/tui/msg"
)

// FieldKind selects the input widget for a form field.
type FieldKind int

const (
	// FieldText is a single-line text input.
	FieldText FieldKind = iota
	// FieldSelect is a one-of picker.
	FieldSelect
	// FieldBool is a yes/no toggle.
	FieldBool
)

// FormField describes one input. Validation runs inline as the user types, so
// a bad port number is caught before the form can be submitted.
type FormField struct {
	Key         string
	Title       string
	Description string
	Placeholder string
	Value       string
	Kind        FieldKind
	Options     []string
	Validate    func(string) error
}

// Form is a huh-backed modal form. On completion it emits a
// msg.FormResultMsg tagged with the form's ID; on abort, msg.CloseOverlayMsg.
type Form struct {
	id    string
	title string
	form  *huh.Form
	texts map[string]*string
	bools map[string]*bool
	done  bool
}

// NewForm builds a form. The ID is what the root dispatcher switches on to
// decide what to do with the values.
func NewForm(id, title string, fields []FormField) *Form {
	f := &Form{
		id:    id,
		title: title,
		texts: map[string]*string{},
		bools: map[string]*bool{},
	}

	huhFields := make([]huh.Field, 0, len(fields))
	for _, fd := range fields {
		switch fd.Kind {
		case FieldBool:
			b := new(bool)
			*b = fd.Value == "true"
			f.bools[fd.Key] = b
			huhFields = append(huhFields, huh.NewConfirm().
				Key(fd.Key).
				Title(fd.Title).
				Description(fd.Description).
				Value(b))

		case FieldSelect:
			v := new(string)
			*v = fd.Value
			f.texts[fd.Key] = v
			huhFields = append(huhFields, huh.NewSelect[string]().
				Key(fd.Key).
				Title(fd.Title).
				Description(fd.Description).
				Options(huh.NewOptions(fd.Options...)...).
				Value(v))

		default:
			v := new(string)
			*v = fd.Value
			f.texts[fd.Key] = v
			input := huh.NewInput().
				Key(fd.Key).
				Title(fd.Title).
				Description(fd.Description).
				Placeholder(fd.Placeholder).
				Value(v)
			if fd.Validate != nil {
				input = input.Validate(fd.Validate)
			}
			huhFields = append(huhFields, input)
		}
	}

	f.form = huh.NewForm(huh.NewGroup(huhFields...)).
		WithTheme(huh.ThemeCharm()).
		WithShowHelp(true).
		WithShowErrors(true)

	return f
}

// Init focuses the first field. The caller must run this when opening the
// modal, or the form renders unfocused.
func (f *Form) Init() tea.Cmd { return f.form.Init() }

// Title implements ModalContent.
func (f *Form) Title() string { return f.title }

// Danger implements ModalContent; forms are never destructive on their own.
func (f *Form) Danger() bool { return false }

// Values returns the collected input, booleans rendered as "true"/"false".
func (f *Form) Values() map[string]string {
	out := make(map[string]string, len(f.texts)+len(f.bools))
	for k, v := range f.texts {
		out[k] = *v
	}
	for k, v := range f.bools {
		if *v {
			out[k] = "true"
		} else {
			out[k] = "false"
		}
	}
	return out
}

// Update drives the huh form and converts its terminal states into app
// messages. The done latch stops a completed form re-emitting its result on
// every subsequent message.
func (f *Form) Update(message tea.Msg) (ModalContent, tea.Cmd) {
	if f.done {
		return f, nil
	}

	model, cmd := f.form.Update(message)
	if form, ok := model.(*huh.Form); ok {
		f.form = form
	}

	switch f.form.State {
	case huh.StateCompleted:
		f.done = true
		values := f.Values()
		id := f.id
		return f, tea.Batch(cmd, func() tea.Msg {
			return msg.FormResultMsg{ID: id, Values: values}
		})
	case huh.StateAborted:
		f.done = true
		return f, tea.Batch(cmd, func() tea.Msg { return msg.CloseOverlayMsg{} })
	}
	return f, cmd
}

// View renders the form body.
func (f *Form) View(width int) string {
	f.form = f.form.WithWidth(width)
	return f.form.View()
}
