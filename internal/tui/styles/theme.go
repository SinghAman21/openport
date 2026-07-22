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

	// AccentOn is legible text placed on top of an Accent fill. It has to track
	// the Accent's own light/dark flip: on a light terminal Accent is a deep
	// violet (white text), on a dark one it is a pale violet (near-black text).
	AccentOn = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1E1B2E"}

	// BarBg is the chrome surface behind the top and bottom bars — a violet-
	// tinted panel that lifts them off the body without a hard border.
	BarBg = lipgloss.AdaptiveColor{Light: "#E7E1F5", Dark: "#241E33"}
	// BarText is default text on that surface, one step softer than body text.
	BarText = lipgloss.AdaptiveColor{Light: "#52525B", Dark: "#B4B4BD"}

	// SelBg / SelFg are the highlighted-row surface. A saturated-enough tint to
	// be unmistakable — this is a cockpit that kills processes, so which row is
	// selected must never be a guess — without the fatigue of a full accent bar.
	SelBg = lipgloss.AdaptiveColor{Light: "#DED7FB", Dark: "#332C52"}
	SelFg = lipgloss.AdaptiveColor{Light: "#1E1B2E", Dark: "#F5F3FF"}
)

// Theme bundles every prebuilt style the views compose with.
type Theme struct {
	// Chrome
	App        lipgloss.Style
	Header     lipgloss.Style
	HeaderName lipgloss.Style
	Brand      lipgloss.Style
	StatusBar  lipgloss.Style
	StatusKey  lipgloss.Style
	StatusHint lipgloss.Style
	StatusMode lipgloss.Style

	// Bar segments — text/keys/hints painted on the BarBg surface, so the top
	// and bottom bars read as continuous fills instead of floating text.
	BarFill lipgloss.Style
	BarSeg  lipgloss.Style
	BarKey  lipgloss.Style
	BarHint lipgloss.Style

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
	// Toasts lead with a heavy accent bar down their left edge and a filled
	// surface, so a result reads as a card that slid in rather than a line of
	// coloured text.
	toast := base.Padding(0, 2).
		Border(lipgloss.ThickBorder(), false, false, false, true).
		Bold(true)

	// Bar building blocks. Every segment that sits on a bar bakes BarBg into its
	// own style: Lip Gloss will not repaint a background across the reset codes
	// that nested, pre-rendered segments emit, so an outer wrapper alone leaves
	// holes. Painting each segment (and the filler) is what makes the fill solid.
	barFill := base.Background(BarBg)
	barSeg := base.Foreground(BarText).Background(BarBg)

	return Theme{
		App:        base.Foreground(Text),
		Header:     base.Foreground(Accent).Bold(true).Padding(0, 1),
		HeaderName: base.Foreground(Text).Bold(true),
		Brand:      base.Foreground(Accent).Background(BarBg).Bold(true).Padding(0, 1),
		StatusBar:  base.Foreground(BarText).Background(BarBg),
		StatusKey:  base.Foreground(Accent).Bold(true),
		StatusHint: base.Foreground(Subtle),
		StatusMode: base.Foreground(AccentOn).Background(Accent).Bold(true).Padding(0, 1),

		BarFill: barFill,
		BarSeg:  barSeg,
		BarKey:  base.Foreground(Accent).Background(BarBg).Bold(true),
		BarHint: base.Foreground(BarText).Background(BarBg),

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
		PanelTitle: base.Foreground(Accent).Bold(true),
		Modal: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).Padding(1, 2),
		ModalTitle: base.Foreground(Accent).Bold(true).MarginBottom(1),
		ModalDanger: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Danger).Padding(1, 2),
		Palette: base.Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).Padding(0, 1),
		Selected:    base.Foreground(SelFg).Background(SelBg).Bold(true),
		SelectedDim: base.Foreground(Text),
		Tab:         base.Foreground(BarText).Background(BarBg).Padding(0, 2),
		TabActive:   base.Foreground(AccentOn).Background(Accent).Bold(true).Padding(0, 2),

		BadgePro:       badge.Foreground(AccentOn).Background(Accent),
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
	GlyphLogo      = "◈"
	GlyphOK        = "✓"
	GlyphDown      = "✕"
	GlyphPending   = "⋯"
	GlyphNone      = "·"
	GlyphProtected = "🔒"
	GlyphWatching  = "●"
	GlyphDrift     = "⚠"
	GlyphArrow     = "›"
	GlyphDot       = "•"
)
