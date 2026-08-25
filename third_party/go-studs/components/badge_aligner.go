package components

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// BadgeAligner provides consistent right-alignment of badges/labels to terminal edge
type BadgeAligner struct {
	width int
}

// NewBadgeAligner creates a new badge aligner for the specified terminal width
func NewBadgeAligner(width int) *BadgeAligner {
	if width < 20 {
		width = 20 // Minimum width
	}
	return &BadgeAligner{width: width}
}

// AlignRight takes a content string and a badge string, and returns the content
// with the badge right-aligned to the terminal edge with proper spacing
func (b *BadgeAligner) AlignRight(content, badge string, minGutter int) string {
	if badge == "" {
		return content
	}

	if minGutter < 1 {
		minGutter = 1 // Minimum 1 space gutter
	}

	// Calculate display width without ANSI codes
	contentDisplayWidth := b.getDisplayWidth(content)
	badgeDisplayWidth := b.getDisplayWidth(badge) // Badge may contain ANSI codes

	// Calculate padding needed: total_width - content_width - badge_width
	paddingNeeded := b.width - contentDisplayWidth - badgeDisplayWidth

	// Ensure minimum gutter
	if paddingNeeded < minGutter {
		paddingNeeded = minGutter
	}

	return content + strings.Repeat(" ", paddingNeeded) + badge
}

// getDisplayWidth returns the display width of a string — ANSI-stripped AND
// runewidth-correct (multi-byte glyphs like '⌄' count as their terminal
// column width, not their byte length).
func (b *BadgeAligner) getDisplayWidth(text string) int {
	return rendering.DisplayWidth(text)
}

// GetWidth returns the configured terminal width
func (b *BadgeAligner) GetWidth() int {
	return b.width
}
