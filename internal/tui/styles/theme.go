// Package styles holds the Lip Gloss design tokens for the whole UI. Every
// colour and border in OpenPort comes from here, so restyling the app is a
// one-file change.
package styles

import "github.com/charmbracelet/lipgloss"

// Palette tokens. AdaptiveColor picks the right shade for light and dark
// terminals, so the UI stays legible on both without a config flag.
var (
	Accent  = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	Success = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	Warning = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	Danger  = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	Info    = lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#38BDF8"}

	Text   = lipgloss.AdaptiveColor{Light: "#18181B", Dark: "#E4E4E7"}
	Muted  = lipgloss.AdaptiveColor{Light: "#71717A", Dark: "#A1A1AA"}
	Subtle = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#52525B"}
	Border = lipgloss.AdaptiveColor{Light: "#D4D4D8", Dark: "#3F3F46"}
)

// Theme bundles every prebuilt style the views compose with.
type Theme struct {
	// Chrome
	App        lipgloss.Style
	Header     lipgloss.Style
	HeaderName lipgloss.Style
	StatusBar  lipgloss.Style
	StatusKey  lipgloss.Style
	StatusHint lipgloss.Style

	// Text
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Faint    lipgloss.Style
	Bold     lipgloss.Style

	// Semantic
	Success lipgloss.Style
	Warning lipgloss.Style
	Danger  lipgloss.Style
	Accent  lipgloss.Style

	// Surfaces
	Panel       lipgloss.Style
	PanelTitle  lipgloss.Style
	Modal       lipgloss.Style
	ModalTitle  lipgloss.Style
	ModalDanger lipgloss.Style
	Palette     lipgloss.Style
	Selected    lipgloss.Style
	SelectedDim lipgloss.Style
	Tab         lipgloss.Style
	TabActive   lipgloss.Style

	// Badges
	BadgePro       lipgloss.Style
	BadgeProtected lipgloss.Style
	BadgeOK        lipgloss.Style
	BadgeDown      lipgloss.Style
	BadgePending   lipgloss.Style
	BadgeGroup     lipgloss.Style
	BadgeDrift     lipgloss.Style

	// Toasts
	ToastSuccess lipgloss.Style
	ToastWarn    lipgloss.Style
	ToastError   lipgloss.Style
}

// New builds the default theme.
func New() Theme {
	base := lipgloss.NewStyle()

	badge := base.Padding(0, 1).Bold(true)
	toast := base.Padding(0, 2).Border(lipgloss.RoundedBorder()).Bold(true)

	return Theme{
		App:        base.Foreground(Text),
		Header:     base.Foreground(Accent).Bold(true).Padding(0, 1),
		HeaderName: base.Foreground(Text).Bold(true),
		StatusBar:  base.Foreground(Muted).Padding(0, 1),
		StatusKey:  base.Foreground(Accent).Bold(true),
		StatusHint: base.Foreground(Subtle),

		Title:    base.Foreground(Accent).Bold(true),
		Subtitle: base.Foreground(Muted),
		Body:     base.Foreground(Text),
		Faint:    base.Foreground(Subtle),
		Bold:     base.Foreground(Text).Bold(true),

		Success: base.Foreground(Success),
		Warning: base.Foreground(Warning),
		Danger:  base.Foreground(Danger),
		Accent:  base.Foreground(Accent),

		Panel: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).Padding(0, 1),
		PanelTitle: base.Foreground(Accent).Bold(true).MarginBottom(1),
		Modal: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).Padding(1, 2),
		ModalTitle: base.Foreground(Accent).Bold(true).MarginBottom(1),
		ModalDanger: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Danger).Padding(1, 2),
		Palette: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).Padding(0, 1),
		Selected:    base.Foreground(Accent).Bold(true),
		SelectedDim: base.Foreground(Text),
		Tab:         base.Foreground(Muted).Padding(0, 2),
		TabActive:   base.Foreground(Accent).Bold(true).Padding(0, 2).Underline(true),

		BadgePro:       badge.Foreground(lipgloss.Color("#FFFFFF")).Background(Accent),
		BadgeProtected: badge.Foreground(Warning),
		BadgeOK:        base.Foreground(Success).Bold(true),
		BadgeDown:      base.Foreground(Danger).Bold(true),
		BadgePending:   base.Foreground(Muted),
		BadgeGroup:     base.Foreground(Info),
		BadgeDrift:     badge.Foreground(Warning).Bold(true),

		ToastSuccess: toast.Foreground(Success).BorderForeground(Success),
		ToastWarn:    toast.Foreground(Warning).BorderForeground(Warning),
		ToastError:   toast.Foreground(Danger).BorderForeground(Danger),
	}
}

// Glyphs used across views. Kept here so the visual language stays consistent.
const (
	GlyphOK        = "✓"
	GlyphDown      = "✕"
	GlyphPending   = "⋯"
	GlyphNone      = "·"
	GlyphProtected = "🔒"
	GlyphWatching  = "●"
	GlyphDrift     = "⚠"
	GlyphArrow     = "›"
)
