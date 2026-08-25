package components

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// SeparatorStyle represents different visual styles for separators
type SeparatorStyle int

const (
	SeparatorLine   SeparatorStyle = iota // ────────────────────────
	SeparatorDots                         // ........................
	SeparatorDash                         // ------------------------
	SeparatorEqual                        // ========================
	SeparatorWave                         // ~~~~~~~~~~~~~~~~~~~~~~~~
	SeparatorDouble                       // ════════════════════════
	SeparatorCustom                       // User-defined character
)

// SeparatorPosition represents label positioning within the separator
type SeparatorPosition int

const (
	PositionLeft   SeparatorPosition = iota // ─── Label ─────────────────
	PositionCenter                          // ────────── Label ──────────
	PositionRight                           // ─────────────── Label ─────
)

// Separator represents a terminal-width aware separator component with optional labeling
type Separator struct {
	label        string
	labelPadding int
	position     SeparatorPosition
	style        SeparatorStyle
	customChar   string
	width        int // Terminal width (auto-detected if 0)
	minWidth     int // Minimum separator width
	fillChar     string
	showLabel    bool
	formatter    *rendering.TextFormatter
}

// NewSeparator creates a new separator with default configuration
func NewSeparator() *Separator {
	return &Separator{
		labelPadding: 1,
		position:     PositionCenter,
		style:        SeparatorLine,
		minWidth:     10,
		fillChar:     "─",
		showLabel:    false,
		formatter:    rendering.NewTextFormatter(),
	}
}

// WithLabel sets the separator label text
func (s *Separator) WithLabel(label string) *Separator {
	s.label = label
	s.showLabel = len(label) > 0
	return s
}

// WithPosition sets the label position within the separator
func (s *Separator) WithPosition(position SeparatorPosition) *Separator {
	s.position = position
	return s
}

// WithStyle sets the visual style of the separator
func (s *Separator) WithStyle(style SeparatorStyle) *Separator {
	s.style = style
	s.updateFillChar()
	return s
}

// WithCustomChar sets a custom character for the separator (requires SeparatorCustom style)
func (s *Separator) WithCustomChar(char string) *Separator {
	s.customChar = char
	s.style = SeparatorCustom
	s.fillChar = char
	return s
}

// WithTerminalWidth sets the terminal width for responsive design
func (s *Separator) WithTerminalWidth(width int) *Separator {
	s.width = width
	return s
}

// WithMinWidth sets the minimum separator width
func (s *Separator) WithMinWidth(minWidth int) *Separator {
	s.minWidth = minWidth
	return s
}

// WithLabelPadding sets the spacing around the label
func (s *Separator) WithLabelPadding(padding int) *Separator {
	s.labelPadding = padding
	return s
}

// Render generates the complete separator display
func (s *Separator) Render() string {
	terminalWidth := s.getTerminalWidth()

	if terminalWidth < s.minWidth {
		terminalWidth = s.minWidth
	}

	if !s.showLabel || s.label == "" {
		return s.renderPlainSeparator(terminalWidth)
	}

	return s.renderLabeledSeparator(terminalWidth)
}

// RenderPlain generates the separator without styling (for width calculations)
func (s *Separator) RenderPlain() string {
	terminalWidth := s.getTerminalWidth()

	if !s.showLabel || s.label == "" {
		return strings.Repeat(s.getPlainChar(), terminalWidth)
	}

	// Calculate plain separator with label
	labelText := s.label
	padding := strings.Repeat(" ", s.labelPadding)
	labelSection := padding + labelText + padding

	totalLabelWidth := len(labelSection)
	if totalLabelWidth >= terminalWidth {
		return labelText // Just the label if no room for separator
	}

	remainingWidth := terminalWidth - totalLabelWidth
	leftWidth, rightWidth := s.calculateSideWidths(remainingWidth)

	leftSide := strings.Repeat(s.getPlainChar(), leftWidth)
	rightSide := strings.Repeat(s.getPlainChar(), rightWidth)

	return leftSide + labelSection + rightSide
}

// renderPlainSeparator creates a separator without a label
func (s *Separator) renderPlainSeparator(width int) string {
	fillChar := s.formatter.Style(s.fillChar, s.getSeparatorToken())
	return strings.Repeat(fillChar, width)
}

// renderLabeledSeparator creates a separator with a positioned label
func (s *Separator) renderLabeledSeparator(width int) string {
	// Prepare label with styling
	labelText := s.formatter.Style(s.label, "tui.SeparatorLabel")
	padding := strings.Repeat(" ", s.labelPadding)
	labelSection := padding + labelText + padding

	// Calculate visual width (accounting for ANSI codes)
	plainLabelSection := padding + s.label + padding
	totalLabelWidth := len(plainLabelSection)

	// If label is too wide, just return the label
	if totalLabelWidth >= width {
		return s.formatter.Style(s.label, "tui.SeparatorLabel")
	}

	// Calculate remaining width for separator characters
	remainingWidth := width - totalLabelWidth
	leftWidth, rightWidth := s.calculateSideWidths(remainingWidth)

	// Create styled separator sides
	fillChar := s.formatter.Style(s.fillChar, s.getSeparatorToken())
	leftSide := strings.Repeat(fillChar, leftWidth)
	rightSide := strings.Repeat(fillChar, rightWidth)

	return leftSide + labelSection + rightSide
}

// calculateSideWidths determines the width distribution for left and right sides
func (s *Separator) calculateSideWidths(totalWidth int) (int, int) {
	if totalWidth <= 0 {
		return 0, 0
	}

	switch s.position {
	case PositionLeft:
		// More space on the right
		leftWidth := totalWidth / 4
		if leftWidth < 1 {
			leftWidth = 1
		}
		rightWidth := totalWidth - leftWidth
		return leftWidth, rightWidth

	case PositionRight:
		// More space on the left
		rightWidth := totalWidth / 4
		if rightWidth < 1 {
			rightWidth = 1
		}
		leftWidth := totalWidth - rightWidth
		return leftWidth, rightWidth

	case PositionCenter:
		// Even distribution (center position)
		fallthrough
	default:
		leftWidth := totalWidth / 2
		rightWidth := totalWidth - leftWidth
		return leftWidth, rightWidth
	}
}

// updateFillChar sets the appropriate fill character based on style
func (s *Separator) updateFillChar() {
	switch s.style {
	case SeparatorLine:
		s.fillChar = "─"
	case SeparatorDots:
		s.fillChar = "."
	case SeparatorDash:
		s.fillChar = "-"
	case SeparatorEqual:
		s.fillChar = "="
	case SeparatorWave:
		s.fillChar = "~"
	case SeparatorDouble:
		s.fillChar = "═"
	case SeparatorCustom:
		if s.customChar != "" {
			s.fillChar = s.customChar
		} else {
			s.fillChar = "─" // Fallback
		}
	default:
		s.fillChar = "─"
	}
}

// getPlainChar returns the fill character without styling
func (s *Separator) getPlainChar() string {
	return s.fillChar
}

// getSeparatorToken returns the appropriate semantic token for separator styling
func (s *Separator) getSeparatorToken() string {
	switch s.style {
	case SeparatorLine, SeparatorDouble:
		return "tui.SeparatorLine"
	case SeparatorDots:
		return "tui.SeparatorDots"
	case SeparatorDash:
		return "tui.SeparatorDash"
	case SeparatorEqual:
		return "tui.SeparatorEqual"
	case SeparatorWave:
		return "tui.SeparatorWave"
	case SeparatorCustom:
		return "tui.SeparatorCustom"
	default:
		return "tui.SeparatorLine"
	}
}

// getTerminalWidth returns the terminal width, detecting automatically if not set
func (s *Separator) getTerminalWidth() int {
	if s.width > 0 {
		return s.width
	}

	// Auto-detect terminal width
	caps := s.formatter.GetCapabilities()
	if caps.Width > 0 {
		return caps.Width
	}

	// Fallback to reasonable default
	return 80
}

// GetVisualWidth returns the visual width of the separator
func (s *Separator) GetVisualWidth() int {
	return s.getTerminalWidth()
}

// ResponsiveRender automatically adjusts to different terminal widths
func (s *Separator) ResponsiveRender() string {
	terminalWidth := s.getTerminalWidth()

	// For very narrow terminals, simplify the separator
	if terminalWidth < 20 {
		return s.renderNarrowMode(terminalWidth)
	}

	// For narrow terminals, may need to abbreviate label
	if terminalWidth < 40 && s.showLabel && len(s.label) > 10 {
		originalLabel := s.label
		s.label = s.abbreviateLabel(s.label, 8)
		result := s.Render()
		s.label = originalLabel // Restore original
		return result
	}

	// Full render for standard terminals
	return s.Render()
}

// renderNarrowMode renders a minimal separator for very narrow terminals
func (s *Separator) renderNarrowMode(width int) string {
	if !s.showLabel || s.label == "" {
		return s.renderPlainSeparator(width)
	}

	// For narrow mode, just show label with minimal separators
	if len(s.label) >= width-4 {
		return s.formatter.Style(s.label, "tui.SeparatorLabel")
	}

	labelText := s.formatter.Style(s.label, "tui.SeparatorLabel")
	fillChar := s.formatter.Style(s.fillChar, s.getSeparatorToken())

	// Simple format: "-- Label --"
	sideWidth := (width - len(s.label) - 2) / 2
	if sideWidth < 1 {
		sideWidth = 1
	}

	leftSide := strings.Repeat(fillChar, sideWidth)
	rightSide := strings.Repeat(fillChar, width-len(s.label)-sideWidth-2)

	return leftSide + " " + labelText + " " + rightSide
}

// abbreviateLabel shortens a label to fit in limited space
func (s *Separator) abbreviateLabel(label string, maxLength int) string {
	if len(label) <= maxLength {
		return label
	}

	if maxLength < 4 {
		return label[:maxLength]
	}

	return label[:maxLength-3] + "..."
}

// Section creates a section separator with proper spacing
func (s *Separator) Section(title string) string {
	return s.WithLabel(title).WithPosition(PositionLeft).Render()
}

// Divider creates a simple divider line
func (s *Separator) Divider() string {
	return s.WithLabel("").Render()
}

// Convenience constructors for common separator configurations

// NewLineSeparator creates a line-style separator
func NewLineSeparator(label string) *Separator {
	return NewSeparator().WithLabel(label).WithStyle(SeparatorLine)
}

// NewDotSeparator creates a dot-style separator
func NewDotSeparator(label string) *Separator {
	return NewSeparator().WithLabel(label).WithStyle(SeparatorDots)
}

// NewDashSeparator creates a dash-style separator
func NewDashSeparator(label string) *Separator {
	return NewSeparator().WithLabel(label).WithStyle(SeparatorDash)
}

// NewEqualSeparator creates an equal-style separator
func NewEqualSeparator(label string) *Separator {
	return NewSeparator().WithLabel(label).WithStyle(SeparatorEqual)
}

// NewSectionSeparator creates a section separator with left-aligned label
func NewSectionSeparator(title string) *Separator {
	return NewSeparator().WithLabel(title).WithPosition(PositionLeft).WithStyle(SeparatorLine)
}

// NewPlainDivider creates a simple divider without label
func NewPlainDivider() *Separator {
	return NewSeparator().WithStyle(SeparatorLine)
}

// NewCustomSeparator creates a separator with custom character
func NewCustomSeparator(char string, label string) *Separator {
	return NewSeparator().WithLabel(label).WithCustomChar(char)
}

// SeparatorGroup manages multiple related separators
type SeparatorGroup struct {
	separators []Separator
	spacing    int
}

// NewSeparatorGroup creates a new group of separators
func NewSeparatorGroup() *SeparatorGroup {
	return &SeparatorGroup{
		separators: make([]Separator, 0),
		spacing:    1,
	}
}

// Add adds a separator to the group
func (sg *SeparatorGroup) Add(separator *Separator) *SeparatorGroup {
	sg.separators = append(sg.separators, *separator)
	return sg
}

// WithSpacing sets the line spacing between separators
func (sg *SeparatorGroup) WithSpacing(spacing int) *SeparatorGroup {
	sg.spacing = spacing
	return sg
}

// RenderAll renders all separators in the group with spacing
func (sg *SeparatorGroup) RenderAll() []string {
	results := make([]string, 0)

	for i, separator := range sg.separators {
		// Add separator
		results = append(results, separator.Render())

		// Add spacing (except after last separator)
		if i < len(sg.separators)-1 {
			for j := 0; j < sg.spacing; j++ {
				results = append(results, "")
			}
		}
	}

	return results
}

// Clear removes all separators from the group
func (sg *SeparatorGroup) Clear() *SeparatorGroup {
	sg.separators = make([]Separator, 0)
	return sg
}

// Count returns the number of separators in the group
func (sg *SeparatorGroup) Count() int {
	return len(sg.separators)
}
