package rendering

import (
	"strings"

	"github.com/branden-thompson/watchpost/third_party/go-studs/tokens"
)

// ANSIFormatter provides pure ANSI escape sequence formatting without external dependencies
type ANSIFormatter struct {
	registry *tokens.TokenRegistry
}

// NewANSIFormatter creates a new ANSI formatter with token registry
func NewANSIFormatter() *ANSIFormatter {
	return &ANSIFormatter{
		registry: tokens.NewTokenRegistry(),
	}
}

// Style applies a semantic token style to text with ANSI escape sequences
func (f *ANSIFormatter) Style(text, semanticToken string) string {
	ansiCode, err := f.registry.ResolveToANSI(semanticToken)
	if err != nil {
		// Fallback to unstyled text if token resolution fails
		return text
	}

	return f.applyANSI(text, ansiCode)
}

// StyleDirect applies ANSI styling with direct ANSI code (bypassing token system)
func (f *ANSIFormatter) StyleDirect(text, ansiCode string) string {
	return f.applyANSI(text, ansiCode)
}

// Bold applies bold formatting to text
func (f *ANSIFormatter) Bold(text string) string {
	return WrapSGR(text, "1")
}

// Underline applies underline formatting to text
func (f *ANSIFormatter) Underline(text string) string {
	return WrapSGR(text, "4")
}

// Italic applies italic formatting to text (not supported by all terminals)
func (f *ANSIFormatter) Italic(text string) string {
	return WrapSGR(text, "3")
}

// Strikethrough applies strikethrough formatting to text (not supported by all terminals)
func (f *ANSIFormatter) Strikethrough(text string) string {
	return WrapSGR(text, "9")
}

// Reverse applies reverse video formatting (swap foreground/background)
func (f *ANSIFormatter) Reverse(text string) string {
	return WrapSGR(text, "7")
}

// CombineStyles combines multiple ANSI formatting codes
func (f *ANSIFormatter) CombineStyles(text string, codes ...string) string {
	if len(codes) == 0 {
		return text
	}

	// Delegate to SGR so every code is classified basic-vs-256 individually —
	// the previous raw join emitted invalid escapes for 256 codes (e.g. ["208","1"]
	// produced \033[208;1m).
	return WrapSGR(text, codes...)
}

// StyledText represents text with multiple style attributes
type StyledText struct {
	Text          string
	SemanticToken string
	Bold          bool
	Underline     bool
	Italic        bool
	Strikethrough bool
	Reverse       bool
	DirectANSI    string // Optional direct ANSI code override
}

// RenderStyled renders a StyledText with all applied formatting
func (f *ANSIFormatter) RenderStyled(styled StyledText) string {
	var codes []string

	// Add color code (semantic token or direct ANSI)
	if styled.DirectANSI != "" {
		codes = append(codes, styled.DirectANSI)
	} else if styled.SemanticToken != "" {
		ansiCode, err := f.registry.ResolveToANSI(styled.SemanticToken)
		if err == nil {
			codes = append(codes, ansiCode)
		}
	}

	// Add formatting codes
	if styled.Bold {
		codes = append(codes, "1")
	}
	if styled.Underline {
		codes = append(codes, "4")
	}
	if styled.Italic {
		codes = append(codes, "3")
	}
	if styled.Strikethrough {
		codes = append(codes, "9")
	}
	if styled.Reverse {
		codes = append(codes, "7")
	}

	return f.CombineStyles(styled.Text, codes...)
}

// applyANSI applies an ANSI color code by delegating to the gated WrapSGR
// constructor — ONE classification codepath (ColorSequence) for the whole
// rendering package. Unified under D-28 (R11a): the former local range logic
// diverged from ColorSequence on {11-29} (both now 256-ify) and {100-107}
// (basic bright-background dropped — a caller census found zero reachable
// users; ColorMR3="107" always flowed through ColorSequence as 256 olive).
func (f *ANSIFormatter) applyANSI(text, ansiCode string) string {
	if ansiCode == "" {
		return text
	}
	return WrapSGR(text, ansiCode)
}

// StripANSI removes all ANSI escape sequences from text
func (f *ANSIFormatter) StripANSI(text string) string {
	// Simple ANSI escape sequence removal
	// This handles common patterns but could be enhanced for comprehensive coverage
	result := text

	// Remove standard ANSI color codes: \033[XXm
	for {
		start := strings.Index(result, "\033[")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}

		result = result[:start] + result[start+end+1:]
	}

	return result
}

// GetVisualWidth calculates the visual width of text (excluding ANSI codes)
func (f *ANSIFormatter) GetVisualWidth(text string) int {
	return len(f.StripANSI(text))
}

// PadRight pads text to a specific visual width (accounting for ANSI codes)
func (f *ANSIFormatter) PadRight(text string, width int) string {
	visualWidth := f.GetVisualWidth(text)
	if visualWidth >= width {
		return text
	}

	padding := strings.Repeat(" ", width-visualWidth)
	return text + padding
}

// PadLeft pads text to a specific visual width on the left (accounting for ANSI codes)
func (f *ANSIFormatter) PadLeft(text string, width int) string {
	visualWidth := f.GetVisualWidth(text)
	if visualWidth >= width {
		return text
	}

	padding := strings.Repeat(" ", width-visualWidth)
	return padding + text
}

// PadCenter centers text within a specific visual width (accounting for ANSI codes)
func (f *ANSIFormatter) PadCenter(text string, width int) string {
	visualWidth := f.GetVisualWidth(text)
	if visualWidth >= width {
		return text
	}

	totalPadding := width - visualWidth
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}

// GetRegistry returns the token registry for advanced usage
func (f *ANSIFormatter) GetRegistry() *tokens.TokenRegistry {
	return f.registry
}
