package rendering

import (
	"os"
	"strings"
)

// TerminalCapabilities represents terminal feature support
type TerminalCapabilities struct {
	SupportsColor     bool
	Supports256Color  bool
	SupportsTrueColor bool
	SupportsUTF8      bool
	Width             int
	Height            int
}

// TextFormatter provides comprehensive text formatting utilities
type TextFormatter struct {
	ansi         *ANSIFormatter
	capabilities *TerminalCapabilities
}

// NewTextFormatter creates a new text formatter with terminal capability detection
func NewTextFormatter() *TextFormatter {
	formatter := &TextFormatter{
		ansi:         NewANSIFormatter(),
		capabilities: DetectTerminalCapabilities(),
	}
	return formatter
}

// DetectTerminalCapabilities automatically detects terminal capabilities
func DetectTerminalCapabilities() *TerminalCapabilities {
	caps := &TerminalCapabilities{
		SupportsColor:     true,  // Default to supporting color
		Supports256Color:  true,  // Most modern terminals support 256 colors
		SupportsTrueColor: false, // More conservative default for true color
		SupportsUTF8:      true,  // Most terminals support UTF-8
	}

	// Detect color support from environment variables
	term := os.Getenv("TERM")
	colorterm := os.Getenv("COLORTERM")

	if term == "" || term == "dumb" {
		caps.SupportsColor = false
		caps.Supports256Color = false
		caps.SupportsTrueColor = false
	}

	if strings.Contains(term, "256color") || strings.Contains(term, "256") {
		caps.Supports256Color = true
	}

	if colorterm == "truecolor" || colorterm == "24bit" {
		caps.SupportsTrueColor = true
	}

	// Detect terminal size
	caps.Width, caps.Height = GetTerminalSize()

	return caps
}

// GetTerminalSize returns the current terminal dimensions
// GetTerminalSize returns the current terminal dimensions with robust fallback logic.
// This is the canonical STUDS library implementation for terminal size detection.
//
// Detection order:
// 1. Open /dev/tty directly and query size (works even when stdout is piped/redirected)
// 2. Try stdin as fallback (when /dev/tty is not available)
// 3. Check COLUMNS and LINES environment variables
// 4. Fall back to reasonable defaults (80x24)
// GetTerminalSize is implemented in platform-specific files:
// - formatter_unix.go for Unix-like systems (macOS, Linux)
// - formatter_windows.go for Windows systems

// Style applies semantic token styling with capability detection
func (tf *TextFormatter) Style(text, semanticToken string) string {
	if !tf.capabilities.SupportsColor {
		return text // Return unstyled text for terminals without color support
	}

	return tf.ansi.Style(text, semanticToken)
}

// StyleWithFallback applies styling with a fallback for unsupported terminals
func (tf *TextFormatter) StyleWithFallback(text, semanticToken, fallback string) string {
	if !tf.capabilities.SupportsColor {
		if fallback != "" {
			return fallback
		}
		return text
	}

	return tf.ansi.Style(text, semanticToken)
}

// WrapText wraps text to fit within specified width, preserving ANSI codes.
// Delegates to WrapTextANSI — width math is runewidth-correct (previously
// byte-count via GetVisualWidth, which over-counted multi-byte glyphs).
func (tf *TextFormatter) WrapText(text string, width int) []string {
	return WrapTextANSI(text, width)
}

// TruncateText truncates text to fit within specified width with ellipsis
func (tf *TextFormatter) TruncateText(text string, width int, ellipsis string) string {
	if ellipsis == "" {
		ellipsis = "..."
	}

	visualWidth := tf.ansi.GetVisualWidth(text)
	ellipsisWidth := tf.ansi.GetVisualWidth(ellipsis)

	if visualWidth <= width {
		return text
	}

	if width <= ellipsisWidth {
		// If width is too small even for ellipsis, return truncated ellipsis
		return ellipsis[:width]
	}

	// Calculate target width for truncation
	targetWidth := width - ellipsisWidth

	// Simple truncation - could be enhanced to handle ANSI codes more precisely
	stripped := tf.ansi.StripANSI(text)
	if len(stripped) <= targetWidth {
		return stripped + ellipsis
	}

	return stripped[:targetWidth] + ellipsis
}

// AlignText aligns text within a given width
type TextAlignment int

const (
	AlignLeft TextAlignment = iota
	AlignCenter
	AlignRight
)

// AlignText aligns text within the specified width
func (tf *TextFormatter) AlignText(text string, width int, alignment TextAlignment) string {
	switch alignment {
	case AlignLeft:
		return tf.ansi.PadRight(text, width)
	case AlignCenter:
		return tf.ansi.PadCenter(text, width)
	case AlignRight:
		return tf.ansi.PadLeft(text, width)
	default:
		return tf.ansi.PadRight(text, width)
	}
}

// CreateBox creates a simple text box around content
func (tf *TextFormatter) CreateBox(content []string, width int, title string) []string {
	if width < 4 {
		return content // Too narrow for a box
	}

	var result []string

	// Top border
	if title != "" {
		titleWidth := tf.ansi.GetVisualWidth(title)
		if titleWidth+4 <= width {
			padding := (width - titleWidth - 4) / 2
			topLine := "┌" + strings.Repeat("─", padding) + " " + title + " " +
				strings.Repeat("─", width-padding-titleWidth-4) + "┐"
			result = append(result, topLine)
		} else {
			result = append(result, "┌"+strings.Repeat("─", width-2)+"┐")
		}
	} else {
		result = append(result, "┌"+strings.Repeat("─", width-2)+"┐")
	}

	// Content lines
	for _, line := range content {
		contentWidth := width - 4 // Account for borders and padding
		if contentWidth > 0 {
			truncated := tf.TruncateText(line, contentWidth, "...")
			padded := tf.ansi.PadRight(truncated, contentWidth)
			result = append(result, "│ "+padded+" │")
		} else {
			result = append(result, "│"+strings.Repeat(" ", width-2)+"│")
		}
	}

	// Bottom border
	result = append(result, "└"+strings.Repeat("─", width-2)+"┘")

	return result
}

// CreateProgressBar creates a visual progress bar
func (tf *TextFormatter) CreateProgressBar(progress float64, width int, fillChar, emptyChar string) string {
	if width <= 2 {
		return "[]"
	}

	if fillChar == "" {
		fillChar = "█"
	}
	if emptyChar == "" {
		emptyChar = "░"
	}

	// Ensure progress is between 0 and 1
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	barWidth := width - 2 // Account for brackets
	filled := int(progress * float64(barWidth))
	empty := barWidth - filled

	bar := "[" + strings.Repeat(fillChar, filled) + strings.Repeat(emptyChar, empty) + "]"
	return bar
}

// GetCapabilities returns the detected terminal capabilities
func (tf *TextFormatter) GetCapabilities() *TerminalCapabilities {
	return tf.capabilities
}

// GetANSIFormatter returns the underlying ANSI formatter for advanced usage
func (tf *TextFormatter) GetANSIFormatter() *ANSIFormatter {
	return tf.ansi
}

// RefreshCapabilities re-detects terminal capabilities (useful if terminal changes)
func (tf *TextFormatter) RefreshCapabilities() {
	tf.capabilities = DetectTerminalCapabilities()
}

// Convenience functions for common formatting needs

// Header formats text as a header with semantic styling
func (tf *TextFormatter) Header(text string) string {
	return tf.Style(text, "tui.HeaderLabel")
}

// Subheader formats text as a subheader
func (tf *TextFormatter) Subheader(text string) string {
	return tf.Style(text, "tui.SubheaderLabel")
}

// Success formats text as a success message
func (tf *TextFormatter) Success(text string) string {
	return tf.Style(text, "tui.SuccessText")
}

// Error formats text as an error message
func (tf *TextFormatter) Error(text string) string {
	return tf.Style(text, "tui.ErrorText")
}

// Warning formats text as a warning message
func (tf *TextFormatter) Warning(text string) string {
	return tf.Style(text, "tui.WarningText")
}

// Info formats text as informational text
func (tf *TextFormatter) Info(text string) string {
	return tf.Style(text, "tui.InfoText")
}

// Muted formats text as muted/secondary text
func (tf *TextFormatter) Muted(text string) string {
	return tf.Style(text, "tui.MutedText")
}

// Command formats text as command text (backticks style)
func (tf *TextFormatter) Command(text string) string {
	return "`" + tf.Style(text, "tui.CommandText") + "`"
}
