package components

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// ColorResolver supplies the default colors for header/footer/separator
// lines at render time: the dash/fill color and the default label color.
// It is a function (not static bytes) because applications may resolve
// colors through a theme-aware token system whose answer can change
// between renders.
type ColorResolver func() (dashColor, labelColor string)

// defaultColorResolver preserves the library's historical defaults:
// dark-gray dashes (90) and bright-white labels (97).
func defaultColorResolver() (string, string) {
	return "90", "97"
}

// HeaderFooterComponent provides consistent terminal-width-aware headers, footers, and separators
// with built-in default colors for consistent styling across all plugins
type HeaderFooterComponent struct {
	width         int
	resolveColors ColorResolver
}

// NewHeaderFooterComponent creates a new header/footer component
func NewHeaderFooterComponent(width int) *HeaderFooterComponent {
	if width < 40 {
		width = 40 // Minimum width
	}
	return &HeaderFooterComponent{width: width, resolveColors: defaultColorResolver}
}

// SetColorResolver injects a per-render color resolver (e.g. a theme-aware
// design-token lookup). Passing nil restores the library default (90/97).
func (h *HeaderFooterComponent) SetColorResolver(r ColorResolver) {
	if r == nil {
		r = defaultColorResolver
	}
	h.resolveColors = r
}

// RenderHeader creates a header line with optional left and right labels
// Format: ----<left-label>---{...}---<right-label>----
// Uses the resolver's default colors: grey dashes, white labels by default
func (h *HeaderFooterComponent) RenderHeader(leftLabel, rightLabel string) string {
	return h.renderLine(leftLabel, rightLabel, "", "")
}

// RenderHeaderWithColors creates a header line with custom label colors
// Pass empty string for the resolver's default label color
func (h *HeaderFooterComponent) RenderHeaderWithColors(leftLabel, rightLabel, leftColor, rightColor string) string {
	return h.renderLine(leftLabel, rightLabel, leftColor, rightColor)
}

// RenderFooter creates a footer line with optional left and right labels
// Format: ----<left-label>---{...}---<right-label>----
// Uses the resolver's default colors: grey dashes, white labels by default
func (h *HeaderFooterComponent) RenderFooter(leftLabel, rightLabel string) string {
	return h.renderLine(leftLabel, rightLabel, "", "")
}

// RenderFooterWithColors creates a footer line with custom label colors
// Pass empty string for the resolver's default label color
func (h *HeaderFooterComponent) RenderFooterWithColors(leftLabel, rightLabel, leftColor, rightColor string) string {
	return h.renderLine(leftLabel, rightLabel, leftColor, rightColor)
}

// RenderSeparator creates a plain separator line
// Format: ----{...}----
// Uses the resolver's default dash color
func (h *HeaderFooterComponent) RenderSeparator() string {
	return h.renderLine("", "", "", "")
}

// renderLine is the core implementation for all line types with color support
func (h *HeaderFooterComponent) renderLine(leftLabel, rightLabel, leftColor, rightColor string) string {
	// Resolve default colors at render time (theme-aware when a resolver
	// is injected; 90/97 library default otherwise)
	dashColor, labelColor := h.resolveColors()
	if leftColor == "" {
		leftColor = labelColor // Default for left label
	}
	if rightColor == "" {
		rightColor = labelColor // Default for right label
	}

	// Build colored components
	lead := h.colorize("----", dashColor)
	tail := h.colorize("----", dashColor)

	// Prepare labels with spaces and colors if they exist
	leftLabelWithSpaces := ""
	if leftLabel != "" {
		coloredLabel := h.colorize(leftLabel, leftColor)
		leftLabelWithSpaces = fmt.Sprintf(" %s ", coloredLabel)
	}

	rightLabelWithSpaces := ""
	if rightLabel != "" {
		coloredLabel := h.colorize(rightLabel, rightColor)
		rightLabelWithSpaces = fmt.Sprintf(" %s ", coloredLabel)
	}

	// Calculate remaining width for fill dashes (using display width for calculation to handle ANSI codes)
	usedWidth := 4 + h.getDisplayWidth(leftLabel) + h.getDisplayWidth(rightLabel) + 4 // lead + labels + tail (without color codes)
	if leftLabel != "" {
		usedWidth += 2 // spaces around left label
	}
	if rightLabel != "" {
		usedWidth += 2 // spaces around right label
	}

	remainingWidth := h.width - usedWidth
	if remainingWidth < 1 {
		remainingWidth = 1 // Minimum fill — a 4-dash floor pushed narrow
		// lines past the terminal width (V-1: w60 footer rendered 61)
	}

	// Create colored fill dashes
	fillDashes := h.colorize(strings.Repeat("-", remainingWidth), dashColor)

	// Build the complete line
	return fmt.Sprintf("%s%s%s%s%s\n",
		lead,
		leftLabelWithSpaces,
		fillDashes,
		rightLabelWithSpaces,
		tail)
}

// colorize applies ANSI color codes to text
func (h *HeaderFooterComponent) colorize(text, color string) string {
	if color == "" || text == "" {
		return text
	}
	// Delegate to the range-aware applier — the raw format emitted invalid
	// escapes for 256-palette colors (e.g. a dash color resolving to 245).
	return rendering.ApplyColorSimple(text, color)
}

// GetWidth returns the configured width
func (h *HeaderFooterComponent) GetWidth() int {
	return h.width
}

// getDisplayWidth returns the display width of a string — ANSI-stripped AND
// runewidth-correct via the shared rendering primitive.
func (h *HeaderFooterComponent) getDisplayWidth(text string) int {
	return rendering.DisplayWidth(text)
}
