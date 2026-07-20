package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// composite draws over on top of base at (top, left), preserving whatever base
// had to the left and right of the overlay on those rows.
//
// Lip Gloss has no z-buffer, so overlays are done by splicing rendered lines.
// The splice is ANSI-aware: cutting a styled string at a display column with
// plain slicing would sever escape sequences and bleed colour down the screen.
func composite(base, over string, top, left int) string {
	if over == "" {
		return base
	}
	baseLines := strings.Split(base, "\n")
	overLines := strings.Split(over, "\n")

	for i, overLine := range overLines {
		row := top + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = spliceLine(baseLines[row], overLine, left)
	}
	return strings.Join(baseLines, "\n")
}

// spliceLine replaces the run of cells [left, left+width(over)) in base.
func spliceLine(base, over string, left int) string {
	if left < 0 {
		left = 0
	}
	overWidth := lipgloss.Width(over)

	head := ansi.Truncate(base, left, "")
	if gap := left - lipgloss.Width(head); gap > 0 {
		// The base line was shorter than the overlay offset; pad it out.
		head += strings.Repeat(" ", gap)
	}

	tail := ansi.TruncateLeft(base, left+overWidth, "")
	return head + over + tail
}

// place computes the top-left corner that centres a block of the given size
// inside width x height, clamped so nothing lands off-screen.
func place(width, height, blockWidth, blockHeight int) (top, left int) {
	left = (width - blockWidth) / 2
	top = (height - blockHeight) / 2
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	return top, left
}

// clampFrame trims a fully composited frame to the terminal box.
//
// This is a backstop, not the layout: views size themselves to the width they
// are handed. But a single over-wide line wraps in the terminal, which shunts
// every following line down and corrupts the whole frame — so the cost of one
// stray cell is the entire screen. Clipping here makes that failure mode
// impossible regardless of what any individual view does.
func clampFrame(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = ansi.Truncate(line, width, "")
		}
	}
	return strings.Join(lines, "\n")
}

// padCanvas ensures s has exactly height lines, so overlays always have rows
// to draw onto even when the active view rendered short.
func padCanvas(s string, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
