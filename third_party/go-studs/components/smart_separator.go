package components

import (
	"os"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// SmartSeparator represents a terminal-width aware header/footer/separator component
// with 5-part structure: lead + leftLabel + fill-line + rightLabel + tail
// Includes configurable spacing above/below for layout control
type SmartSeparator struct {
	leftLabel     string
	rightLabel    string
	leftColor     string // Override color for left label
	rightColor    string // Override color for right label
	width         int    // Terminal width (auto-detected if 0)
	leadLength    int    // Length of lead dashes (default: 4)
	tailLength    int    // Length of tail dashes (default: 4)
	minFillLength int    // Minimum fill line length (default: 4)
	spacingBefore bool   // Add empty line before separator (default: true)
	spacingAfter  bool   // Add empty line after separator (default: true)
	formatter     *rendering.TextFormatter
}

// NewSmartSeparator creates a new smart separator with default configuration
// Default: spacing before and after (most common use case)
func NewSmartSeparator() *SmartSeparator {
	return &SmartSeparator{
		leadLength:    4,
		tailLength:    4,
		minFillLength: 4,
		spacingBefore: true, // Default: spacing before
		spacingAfter:  true, // Default: spacing after
		formatter:     rendering.NewTextFormatter(),
	}
}

// WithLeftLabel sets the left label text
func (s *SmartSeparator) WithLeftLabel(label string) *SmartSeparator {
	s.leftLabel = label
	return s
}

// WithRightLabel sets the right label text
func (s *SmartSeparator) WithRightLabel(label string) *SmartSeparator {
	s.rightLabel = label
	return s
}

// WithLeftColor sets color override for left label
func (s *SmartSeparator) WithLeftColor(color string) *SmartSeparator {
	s.leftColor = color
	return s
}

// WithRightColor sets color override for right label
func (s *SmartSeparator) WithRightColor(color string) *SmartSeparator {
	s.rightColor = color
	return s
}

// WithTerminalWidth sets the terminal width for responsive design
func (s *SmartSeparator) WithTerminalWidth(width int) *SmartSeparator {
	s.width = width
	return s
}

// WithLeadLength sets the length of lead dashes
func (s *SmartSeparator) WithLeadLength(length int) *SmartSeparator {
	s.leadLength = length
	return s
}

// WithTailLength sets the length of tail dashes
func (s *SmartSeparator) WithTailLength(length int) *SmartSeparator {
	s.tailLength = length
	return s
}

// WithMinFillLength sets minimum fill line length
func (s *SmartSeparator) WithMinFillLength(length int) *SmartSeparator {
	s.minFillLength = length
	return s
}

// WithHeaderSpacing configures separator for header use (spacing after, no spacing before)
func (s *SmartSeparator) WithHeaderSpacing() *SmartSeparator {
	s.spacingBefore = false
	s.spacingAfter = true
	return s
}

// WithFooterSpacing configures separator for footer use (spacing before, no spacing after)
func (s *SmartSeparator) WithFooterSpacing() *SmartSeparator {
	s.spacingBefore = true
	s.spacingAfter = false
	return s
}

// WithNoSpacing disables all spacing around separator
func (s *SmartSeparator) WithNoSpacing() *SmartSeparator {
	s.spacingBefore = false
	s.spacingAfter = false
	return s
}

// Render generates the complete smart separator display
// Structure: lead + leftLabel + fill-line + rightLabel + tail
func (s *SmartSeparator) Render() string {
	terminalWidth := s.getTerminalWidth()

	// Create dash character with grey color
	dashChar := s.formatter.Style("-", "tui.SeparatorDash")

	// Handle pure separator line (no labels)
	if s.leftLabel == "" && s.rightLabel == "" {
		separatorLine := strings.Repeat(dashChar, terminalWidth)

		// Add spacing based on configuration
		result := ""
		if s.spacingBefore {
			result += "\n" // Move to new line, then empty line creates the blank line
		}
		result += separatorLine
		if s.spacingAfter {
			result += "\n\n" // Move to new line, then add empty line for proper blank line spacing
		}
		return result
	}

	// Calculate component parts
	lead := strings.Repeat(dashChar, s.leadLength)
	tail := strings.Repeat(dashChar, s.tailLength)

	// Format labels with padding and colors
	leftSection := s.formatLeftLabel()
	rightSection := s.formatRightLabel()

	// Calculate plain text lengths for width calculation
	plainLeftSection := s.getPlainLeftLabel()
	plainRightSection := s.getPlainRightLabel()

	// Calculate available space for fill line
	usedSpace := s.leadLength + len(plainLeftSection) + len(plainRightSection) + s.tailLength
	availableSpace := terminalWidth - usedSpace

	// Ensure minimum fill length
	if availableSpace < s.minFillLength {
		availableSpace = s.minFillLength
	}

	fillLine := strings.Repeat(dashChar, availableSpace)

	// Build the separator line
	separatorLine := lead + leftSection + fillLine + rightSection + tail

	// Add spacing based on configuration
	result := ""
	if s.spacingBefore {
		result += "\n" // Move to new line, then empty line creates the blank line
	}
	result += separatorLine
	if s.spacingAfter {
		result += "\n\n" // Move to new line, then add empty line for proper blank line spacing
	}

	return result
}

// formatLeftLabel formats the left label with appropriate color and spacing
func (s *SmartSeparator) formatLeftLabel() string {
	if s.leftLabel == "" {
		return ""
	}

	color := s.leftColor
	if color == "" {
		color = "tui.SeparatorLabel" // Default white
	}

	styledLabel := s.formatter.Style(s.leftLabel, color)
	return " " + styledLabel + " "
}

// formatRightLabel formats the right label with appropriate color and spacing
func (s *SmartSeparator) formatRightLabel() string {
	if s.rightLabel == "" {
		return ""
	}

	color := s.rightColor
	if color == "" {
		color = "tui.SeparatorLabel" // Default white
	}

	styledLabel := s.formatter.Style(s.rightLabel, color)
	return " " + styledLabel + " "
}

// getPlainLeftLabel returns plain text version for width calculation (strips ANSI codes)
func (s *SmartSeparator) getPlainLeftLabel() string {
	if s.leftLabel == "" {
		return ""
	}
	// Strip ANSI codes from the label for accurate width calculation
	ansiFormatter := rendering.NewANSIFormatter()
	plainText := ansiFormatter.StripANSI(s.leftLabel)
	return " " + plainText + " "
}

// getPlainRightLabel returns plain text version for width calculation (strips ANSI codes)
func (s *SmartSeparator) getPlainRightLabel() string {
	if s.rightLabel == "" {
		return ""
	}
	// Strip ANSI codes from the label for accurate width calculation
	ansiFormatter := rendering.NewANSIFormatter()
	plainText := ansiFormatter.StripANSI(s.rightLabel)
	return " " + plainText + " "
}

// getTerminalWidth returns the terminal width, detecting automatically if not set
func (s *SmartSeparator) getTerminalWidth() int {
	if s.width > 0 {
		return s.width
	}

	// Auto-detect terminal width using proper GO-STUDS method
	width, _ := rendering.GetTerminalSize()
	if width > 0 {
		return width
	}

	// Try to use COLUMNS environment variable as fallback
	if colStr := os.Getenv("COLUMNS"); colStr != "" {
		if cols, err := strconv.Atoi(colStr); err == nil && cols > 0 {
			return cols
		}
	}

	// Fallback to reasonable default
	return 80
}

// ResponsiveRender automatically adjusts to different terminal widths
func (s *SmartSeparator) ResponsiveRender() string {
	terminalWidth := s.getTerminalWidth()

	// For very narrow terminals, simplify
	if terminalWidth < 20 {
		return s.renderNarrowMode(terminalWidth)
	}

	// For narrow terminals with long labels, may need to truncate
	if terminalWidth < 60 {
		return s.renderCompactMode(terminalWidth)
	}

	// Full render for standard terminals
	return s.Render()
}

// renderNarrowMode renders for very narrow terminals
func (s *SmartSeparator) renderNarrowMode(width int) string {
	dashChar := s.formatter.Style("-", "tui.SeparatorDash")

	// Pure separator if no labels or too narrow
	if s.leftLabel == "" && s.rightLabel == "" {
		return strings.Repeat(dashChar, width)
	}

	// Priority: left label only if both exist
	label := s.leftLabel
	if label == "" {
		label = s.rightLabel
	}

	if len(label)+4 >= width {
		// Just return the label if no room for dashes
		color := s.leftColor
		if s.leftLabel == "" && s.rightLabel != "" {
			color = s.rightColor
		}
		if color == "" {
			color = "tui.SeparatorLabel"
		}
		return s.formatter.Style(label, color)
	}

	// Simple format: "-- Label --"
	color := s.leftColor
	if s.leftLabel == "" && s.rightLabel != "" {
		color = s.rightColor
	}
	if color == "" {
		color = "tui.SeparatorLabel"
	}

	styledLabel := s.formatter.Style(label, color)
	remainingSpace := width - len(label) - 2
	sideWidth := remainingSpace / 2

	leftSide := strings.Repeat(dashChar, sideWidth)
	rightSide := strings.Repeat(dashChar, remainingSpace-sideWidth)

	return leftSide + " " + styledLabel + " " + rightSide
}

// renderCompactMode renders for narrow terminals with potential label truncation
func (s *SmartSeparator) renderCompactMode(width int) string {
	// Calculate if we need to truncate labels
	plainLeftSection := s.getPlainLeftLabel()
	plainRightSection := s.getPlainRightLabel()
	fixedSpace := s.leadLength + s.tailLength + s.minFillLength
	labelSpace := len(plainLeftSection) + len(plainRightSection)

	if fixedSpace+labelSpace <= width {
		return s.Render() // Fits normally
	}

	// Need to truncate - prioritize left label
	availableLabelSpace := width - fixedSpace

	if availableLabelSpace <= 0 {
		// No room for labels, just return plain separator
		dashChar := s.formatter.Style("-", "tui.SeparatorDash")
		return strings.Repeat(dashChar, width)
	}

	// Truncate labels to fit
	truncatedLeft := s.leftLabel
	truncatedRight := s.rightLabel

	if len(plainLeftSection)+len(plainRightSection) > availableLabelSpace {
		// Need to truncate
		if s.rightLabel == "" {
			// Only left label - use most of available space
			maxLeftLen := availableLabelSpace - 2 // Account for spaces
			if len(s.leftLabel) > maxLeftLen && maxLeftLen > 3 {
				truncatedLeft = s.leftLabel[:maxLeftLen-3] + "..."
			}
		} else if s.leftLabel == "" {
			// Only right label - use most of available space
			maxRightLen := availableLabelSpace - 2 // Account for spaces
			if len(s.rightLabel) > maxRightLen && maxRightLen > 3 {
				truncatedRight = s.rightLabel[:maxRightLen-3] + "..."
			}
		} else {
			// Both labels - split space equally
			halfSpace := (availableLabelSpace - 4) / 2 // Account for 4 spaces total
			if halfSpace > 3 {
				if len(s.leftLabel) > halfSpace {
					truncatedLeft = s.leftLabel[:halfSpace-3] + "..."
				}
				if len(s.rightLabel) > halfSpace {
					truncatedRight = s.rightLabel[:halfSpace-3] + "..."
				}
			}
		}
	}

	// Create temporary separator with truncated labels
	temp := &SmartSeparator{
		leftLabel:     truncatedLeft,
		rightLabel:    truncatedRight,
		leftColor:     s.leftColor,
		rightColor:    s.rightColor,
		width:         width,
		leadLength:    s.leadLength,
		tailLength:    s.tailLength,
		minFillLength: s.minFillLength,
		formatter:     s.formatter,
	}

	return temp.Render()
}

// CreateSmartSeparator creates and renders a terminal-width aware separator in one call
//
// 🎯 GO-STUDS API DESIGN PATTERN: This function exemplifies the idiomatic Go API pattern
// that ALL STUDS components should follow - single function handles all variations.
//
// Handles all separator variants based on label parameters:
//   - Both labels: CreateSmartSeparator("Left", "Right") or CreateSmartSeparator("Left", "Right", 80)
//   - Left only: CreateSmartSeparator("Left", "") or CreateSmartSeparator("Left", "", 80)
//   - Right only: CreateSmartSeparator("", "Right") or CreateSmartSeparator("", "Right", 80)
//   - Pure separator: CreateSmartSeparator("", "") or CreateSmartSeparator("", "", 80)
//
// Optional width parameter overrides terminal width detection using Go variadic pattern
func CreateSmartSeparator(leftLabel, rightLabel string, width ...int) string {
	sep := NewSmartSeparator().WithLeftLabel(leftLabel).WithRightLabel(rightLabel)

	// Apply optional width parameter if provided
	if len(width) > 0 && width[0] > 0 {
		sep = sep.WithTerminalWidth(width[0])
	}

	return sep.Render()
}
