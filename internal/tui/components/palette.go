package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// paletteRows is how many matches the palette shows at once.
const paletteRows = 9

// Palette is the fuzzy quick-action launcher opened with ':' or ctrl+k. It is
// the TUI-native equivalent of the headless subcommands: same actions, nothing
// to memorise.
type Palette struct {
	input   textinput.Model
	all     []Item
	matches []Item
	cursor  int
	theme   styles.Theme
	open    bool
}

// NewPalette builds a closed palette over the given items.
func NewPalette(theme styles.Theme, items []Item) *Palette {
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = "Type an action…"
	in.CharLimit = 64

	p := &Palette{input: in, all: items, theme: theme}
	p.filter()
	return p
}

// SetItems replaces the searchable set, e.g. when Pro gating changes.
func (p *Palette) SetItems(items []Item) {
	p.all = items
	p.filter()
}

// Open focuses the palette and clears any previous query.
func (p *Palette) Open() tea.Cmd {
	p.open = true
	p.input.SetValue("")
	p.cursor = 0
	p.filter()
	return p.input.Focus()
}

// Close hides the palette.
func (p *Palette) Close() {
	p.open = false
	p.input.Blur()
}

// IsOpen reports whether the palette is capturing input.
func (p *Palette) IsOpen() bool { return p.open }

// Current returns the highlighted match.
func (p *Palette) Current() (Item, bool) {
	if p.cursor < 0 || p.cursor >= len(p.matches) {
		return Item{}, false
	}
	it := p.matches[p.cursor]
	return it, !it.Disabled
}

var paletteKeys = struct {
	up, down key.Binding
}{
	// Ctrl+p/n as well as arrows: the palette is a text field, so j/k must
	// stay typeable.
	up:   key.NewBinding(key.WithKeys("up", "ctrl+p")),
	down: key.NewBinding(key.WithKeys("down", "ctrl+n")),
}

// Update handles typing and match navigation.
func (p *Palette) Update(message tea.Msg) (*Palette, tea.Cmd) {
	if !p.open {
		return p, nil
	}
	if keyMsg, ok := message.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, paletteKeys.up):
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil
		case key.Matches(keyMsg, paletteKeys.down):
			if p.cursor < len(p.matches)-1 {
				p.cursor++
			}
			return p, nil
		}
	}

	before := p.input.Value()
	in, cmd := p.input.Update(message)
	p.input = in
	if p.input.Value() != before {
		p.filter()
		p.cursor = 0
	}
	return p, cmd
}

// paletteSource adapts items to the fuzzy matcher, searching title, category
// and description together so "kill range", "ports" and "terminate" all land.
type paletteSource []Item

func (s paletteSource) Len() int { return len(s) }
func (s paletteSource) String(i int) string {
	return s[i].Title + " " + s[i].Group + " " + s[i].Desc
}

func (p *Palette) filter() {
	query := strings.TrimSpace(p.input.Value())
	if query == "" {
		p.matches = p.all
		return
	}
	results := fuzzy.FindFrom(query, paletteSource(p.all))
	p.matches = make([]Item, 0, len(results))
	for _, r := range results {
		p.matches = append(p.matches, p.all[r.Index])
	}
}

// View renders the palette overlay.
func (p *Palette) View(width int) string {
	inner := width - 4
	if inner < 20 {
		inner = 20
	}
	p.input.Width = inner - 2

	var b strings.Builder
	b.WriteString(p.input.View())
	b.WriteString("\n")
	b.WriteString(p.theme.Faint.Render(strings.Repeat("─", inner)))
	b.WriteString("\n")

	if len(p.matches) == 0 {
		b.WriteString(p.theme.Faint.Render("  No matching actions"))
		return p.theme.Palette.Width(inner).Render(b.String())
	}

	start := 0
	if p.cursor >= paletteRows {
		start = p.cursor - paletteRows + 1
	}
	for i := start; i < len(p.matches) && i < start+paletteRows; i++ {
		b.WriteString(p.renderRow(p.matches[i], i == p.cursor, inner))
		b.WriteString("\n")
	}
	if len(p.matches) > paletteRows {
		b.WriteString(p.theme.Faint.Render(
			"  " + itoa(len(p.matches)) + " actions · ctrl+n/ctrl+p to move"))
	}
	return p.theme.Palette.Width(inner).Render(strings.TrimRight(b.String(), "\n"))
}

func (p *Palette) renderRow(it Item, selected bool, width int) string {
	prefix := "  "
	title := it.Title
	switch {
	case it.Disabled:
		title = p.theme.Faint.Render(title)
	case selected:
		prefix = p.theme.Accent.Render(styles.GlyphArrow + " ")
		title = p.theme.Selected.Render(title)
	default:
		title = p.theme.Body.Render(title)
	}

	left := prefix + title
	if it.Badge != "" {
		left += " " + p.theme.BadgePro.Render(it.Badge)
	}
	right := p.theme.Faint.Render(it.Group)
	if it.Key != "" {
		right = p.theme.StatusKey.Render(it.Key) + "  " + right
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncate(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// itoa avoids pulling strconv into every render path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
