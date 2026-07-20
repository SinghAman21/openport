package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// ToastLifetime is how long a non-error toast stays on screen. Errors are
// excluded: they persist until dismissed, because a message you missed is
// worse than one you have to close.
const ToastLifetime = 4 * time.Second

// MaxToasts caps the stack so a burst of results cannot cover the UI.
const MaxToasts = 3

// Toast is one transient notification.
type Toast struct {
	ID    int
	Text  string
	Level msg.ToastLevel
}

// Sticky reports whether the toast must be dismissed manually.
func (t Toast) Sticky() bool { return t.Level == msg.ToastError }

// ExpireCmd schedules this toast's dismissal. Errors never expire.
func (t Toast) ExpireCmd() tea.Cmd {
	if t.Sticky() {
		return nil
	}
	id := t.ID
	return tea.Tick(ToastLifetime, func(time.Time) tea.Msg {
		return msg.ToastExpiredMsg{ID: id}
	})
}

// RenderToasts stacks toasts bottom-up into a right-aligned column.
func RenderToasts(theme styles.Theme, toasts []Toast, width int) string {
	if len(toasts) == 0 {
		return ""
	}
	maxWidth := width/2 - 4
	if maxWidth < 20 {
		maxWidth = width - 4
	}
	if maxWidth < 10 {
		return ""
	}

	rendered := make([]string, 0, len(toasts))
	for _, t := range toasts {
		style := theme.ToastSuccess
		icon := styles.GlyphOK
		switch t.Level {
		case msg.ToastError:
			style, icon = theme.ToastError, styles.GlyphDown
		case msg.ToastWarn:
			style, icon = theme.ToastWarn, styles.GlyphDrift
		case msg.ToastInfo:
			style, icon = theme.ToastSuccess, styles.GlyphArrow
		}

		text := icon + "  " + t.Text
		if t.Sticky() {
			text += "  " + theme.Faint.Render("(esc)")
		}
		rendered = append(rendered, style.MaxWidth(maxWidth).Render(truncate(text, maxWidth-4)))
	}
	return lipgloss.JoinVertical(lipgloss.Right, rendered...)
}

// StatusToast renders the newest toast as a single status-bar line, used when
// the terminal is too short for the floating stack.
func StatusToast(theme styles.Theme, toasts []Toast, width int) string {
	if len(toasts) == 0 {
		return ""
	}
	t := toasts[len(toasts)-1]
	style := theme.Success
	switch t.Level {
	case msg.ToastError:
		style = theme.Danger
	case msg.ToastWarn:
		style = theme.Warning
	}
	return style.Render(truncate(strings.TrimSpace(t.Text), width))
}
