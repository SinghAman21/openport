package components

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/styles"
)

// huhTheme is OpenPort's form theme. huh ships ThemeCharm, whose fuchsia
// selectors and buttons fight the violet the rest of the app is built on — a
// modal that looks like it came from a different program is exactly the kind of
// seam that reads as unfinished. This restyles huh's fields onto the shared
// palette so a form or confirmation is unmistakably part of the same product.
func huhTheme() *huh.Theme {
	t := huh.ThemeCharm()

	accent := styles.Accent
	on := styles.AccentOn

	t.Focused.Base = t.Focused.Base.BorderForeground(styles.Border)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(accent).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(accent).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(styles.Muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(styles.Danger)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(styles.Danger)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.Option = t.Focused.Option.Foreground(styles.Text)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(accent).Bold(true)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(styles.Muted)

	// The active button is a filled violet pill; the other one a quiet chip.
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(on).Background(accent).Bold(true)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(styles.Muted).Background(styles.BarBg)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(accent)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(accent)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(styles.Subtle)

	// huh's Charm theme copies Focused into Blurred, so mirror the overrides and
	// keep the hidden-border trick that distinguishes an unfocused field.
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}
