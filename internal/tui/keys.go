package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap is every global binding. Views contribute their own bindings via
// their Help() method; these are the ones that work everywhere.
//
// Arrow keys and Vim keys are bound together on every motion, so neither
// muscle memory is second-class.
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Top    key.Binding
	Bottom key.Binding

	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding

	Help    key.Binding
	Palette key.Binding
	Menu    key.Binding
	Refresh key.Binding
	Debug   key.Binding
	Suspend key.Binding

	// Direct view jumps.
	GotoPorts      key.Binding
	GotoActions    key.Binding
	GotoAliases    key.Binding
	GotoBlueprints key.Binding
	GotoConfig     key.Binding
	GotoHistory    key.Binding
	GotoWorkspace  key.Binding
}

// DefaultKeyMap returns the global bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next tab"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/close"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Palette: key.NewBinding(
			key.WithKeys(":", "ctrl+k"),
			key.WithHelp(":", "command palette"),
		),
		Menu: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "action menu"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Debug: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "dump state"),
		),
		Suspend: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("ctrl+z", "suspend"),
		),
		GotoPorts:      key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "ports")),
		GotoActions:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "actions")),
		GotoAliases:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "aliases")),
		GotoBlueprints: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "blueprints")),
		GotoConfig:     key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "config")),
		GotoHistory:    key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "history")),
		GotoWorkspace:  key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "workspace")),
	}
}

// ShortHelp is the condensed hint set shown in the status bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Palette, k.Menu, k.Help, k.Quit}
}

// FullHelp is the global section of the help overlay.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom},
		{k.Enter, k.Back, k.Refresh, k.Quit},
		{k.Palette, k.Menu, k.Help, k.Debug, k.Suspend},
		{k.GotoPorts, k.GotoActions, k.GotoAliases, k.GotoBlueprints},
		{k.GotoConfig, k.GotoHistory, k.GotoWorkspace},
	}
}
