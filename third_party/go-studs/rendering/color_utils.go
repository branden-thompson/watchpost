package rendering

import (
	"strings"
)

// colorReset is the SGR reset parameter ("0" → \033[0m). Local constant —
// byte-identical to the original colorReset it replaces; kept here so the
// library has no application imports.
const colorReset = "0"

// ColorUtils provides shared color utilities for terminal rendering
// This centralizes color logic to ensure consistency across all packages
//
// This is the authoritative implementation that consolidates color handling
// from multiple locations (crews renderers, app-status, TUI components, etc.)
type ColorUtils struct{}

// NewColorUtils creates a new color utilities instance
func NewColorUtils() *ColorUtils {
	return &ColorUtils{}
}

// ApplyColor applies ANSI color codes with proper support for basic ANSI, 256-color, and RGB codes
//
// This is the authoritative implementation that handles:
// - Basic ANSI codes (30-37, 40-47, 90-97): Use \033[Xm format
// - 256-color palette codes (73, 196, 202, etc.): Use \033[38;5;Xm format
// - RGB true color codes (R:G:B format): Use \033[38;2;R;G;Bm format (24-bit)
//
// CRITICAL: This implementation includes ANSI range detection to prevent color display bugs
// OPTIMIZED: Uses strings.Builder to reduce memory allocations by ~50%
//
// Example:
//
//	colorUtils := NewColorUtils()
//
//	// Basic ANSI color (bright white)
//	text := colorUtils.ApplyColor("Hello", "97")
//	// Output: "\033[97mHello\033[0m"
//
//	// 256-color palette (deep red for on-call)
//	onCallText := colorUtils.ApplyColor("⏺ On-Call", "196")
//	// Output: "\033[38;5;196m⏺ On-Call\033[0m"
//
//	// RGB true color (24-bit)
//	canaryText := colorUtils.ApplyColor("◆ Canary", "0:255:255")
//	// Output: "\033[38;2;0;255;255m◆ Canary\033[0m"
func (c *ColorUtils) ApplyColor(text, colorCode string) string {
	if !colorEnabled() {
		return text
	}
	if colorCode == "" || text == "" {
		return text
	}

	var sb strings.Builder

	// Check for RGB format (R:G:B with colons)
	if strings.Contains(colorCode, ":") {
		// RGB true color format: \033[38;2;R;G;Bm
		// Convert colon-separated to semicolon-separated for ANSI
		rgbParts := strings.Replace(colorCode, ":", ";", -1)
		sb.Grow(15 + len(rgbParts) + len(text) + len(colorReset))
		sb.WriteString("\033[38;2;")
		sb.WriteString(rgbParts)
		sb.WriteString("m")
		sb.WriteString(text)
		sb.WriteString("\033[")
		sb.WriteString(colorReset)
		sb.WriteString("m")
		return sb.String()
	}

	// Pre-calculate exact size for builder to prevent reallocation
	// ANSI codes: "\033[" (2) + code (1-3) + "m" (1) + text + "\033[" (2) + reset (1) + "m" (1)
	builderSize := 7 + len(colorCode) + len(text) + len(colorReset)
	sb.Grow(builderSize)

	// Classification is delegated to ColorSequence — the single source of truth
	// for basic-vs-256 detection (basic codes pass through; 3-digit and 2-digit
	// palette-band codes become "38;5;N"). Byte-for-byte identical output to the
	// previous inline range logic.
	sb.WriteString("\033[")
	sb.WriteString(ColorSequence(colorCode))
	sb.WriteString("m")
	sb.WriteString(text)
	sb.WriteString("\033[")
	sb.WriteString(colorReset)
	sb.WriteString("m")

	return sb.String()
}

// ApplyColorSimple is a simple wrapper for non-optimized use cases
// Use ApplyColor() for performance-critical code
func ApplyColorSimple(text, colorCode string) string {
	utils := NewColorUtils()
	return utils.ApplyColor(text, colorCode)
}

// ColorSequence converts ONE design-system color code to its SGR parameter for
// use inside an \033[...m sequence. This is the single source of truth for
// basic-vs-256 classification, extracted from ApplyColor (which delegates here):
// 1-2 digit basic ANSI codes (0-10, 30-37, 40-47, 90-97) pass through
// unchanged; all 3-digit codes and the 2-digit 256-palette bands (11-29,
// 38-39, 48-89, 98-99) become "38;5;N". The 11-29 band was unified with
// applyANSI's classification under D-28 (R11a) after a caller census proved
// zero attribute-intent users of 21-29 and color-intent raw tokens
// (ansi-22/ansi-23) declared in the palette.
//
// Single-code contract: never pass multi-code strings ("1;90") — use SGR, which
// splits and classifies each element. RGB colon codes ("0:255:255") and the
// empty string are outside this contract: ApplyColor handles RGB and empty in
// its own branches; ColorSequence passes them through unmodified.
func ColorSequence(code string) string {
	if len(code) >= 3 {
		// Colon-format RGB is not a palette code — out of contract, pass through
		if strings.Contains(code, ":") || strings.Contains(code, ";") {
			return code
		}
		return "38;5;" + code
	}
	if len(code) == 2 {
		if (code >= "11" && code <= "29") ||
			(code >= "38" && code <= "39") ||
			(code >= "48" && code <= "89") ||
			(code >= "98" && code <= "99") {
			return "38;5;" + code
		}
	}
	return code
}

// SGR is the ONE sanctioned constructor for \033[...m escape sequences.
// Each param is split on ";" and every element is classified via ColorSequence
// (attribute codes like "1", "4", "7", "3" pass through; color codes classify
// to basic or 256-palette form). This makes multi-code joins safe:
//
//	SGR("91", "1")   // "\033[91;1m"        (basic color + bold)
//	SGR("245", "1")  // "\033[38;5;245;1m"  (256 color + bold)
//	SGR("93;3")      // "\033[93;3m"        (pre-joined param split + classified)
//
// All rendering outside the range-aware appliers MUST build escapes through
// SGR — the archtest ANSI-format guard enforces this.
//
// Qualified composites are consumed atomically and passed through unchanged:
// "38;5;N", "48;5;N", "38;2;R;G;B", "48;2;R;G;B" — so a fully-qualified
// parameter list ("1;38;5;220") renders as written instead of being
// re-classified element by element ("38" is a 256-band code on its own).
// One allocation: the escape is built in place.
func SGR(params ...string) string {
	n := 3
	for _, p := range params {
		n += len(p) + 1
	}
	var b strings.Builder
	b.Grow(n + 8)
	b.WriteString("\033[")
	first := true
	for _, p := range params {
		for i := len(p); i > 0 && len(p) > 0; i-- { // every element consumes at least one byte: bounded by the parameter's length
			elem, rest := cutElem(p)
			out := ColorSequence(elem)
			if n := compositeLen(elem, rest); n > 0 {
				out, rest = p[:n], strings.TrimPrefix(p[n:], ";") // the composite as written, no copy
			}
			if !first {
				b.WriteByte(';')
			}
			first = false
			b.WriteString(out)
			p = rest
		}
	}
	b.WriteByte('m') // ansi-constructor: sole sanctioned emitter
	return b.String()
}

// cutElem splits the first ";"-separated element off p.
func cutElem(p string) (elem, rest string) {
	if i := strings.IndexByte(p, ';'); i >= 0 {
		return p[:i], p[i+1:]
	}
	return p, ""
}

// compositeLen recognises a qualified colour that begins with elem — "38"
// or "48" followed by "5;N" or "2;R;G;B" in rest — and returns the length
// of the whole composite within elem+";"+rest (0 when elem starts none).
func compositeLen(elem, rest string) int {
	if elem != "38" && elem != "48" {
		return 0
	}
	mode, after := cutElem(rest)
	count := qualifierParts(mode)
	if count == 0 {
		return 0
	}
	end := 0
	for i := 0; i < count; i++ {
		part, next := cutElem(after[end:])
		if part == "" || !allDigits(part) {
			return 0
		}
		end += len(part)
		if i < count-1 {
			if next == "" {
				return 0
			}
			end++ // the ';'
		}
	}
	return len(elem) + 1 + len(mode) + 1 + end
}

// qualifierParts is how many numbers follow a colour qualifier: one for
// the 256-palette ("5;N"), three for truecolor ("2;R;G;B"), none otherwise.
func qualifierParts(mode string) int {
	switch mode {
	case "5":
		return 1
	case "2":
		return 3
	}
	return 0
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// GetDisplayWidth calculates the display width of text containing ANSI color codes
// This is critical for proper alignment when text contains ANSI escape sequences
//
// Example:
//
//	colorUtils := NewColorUtils()
//	colored := colorUtils.ApplyColor("Hello", "91")  // "\033[91mHello\033[0m"
//	width := colorUtils.GetDisplayWidth(colored)      // Returns 5 (not 18)
func (c *ColorUtils) GetDisplayWidth(text string) int {
	// Strip all ANSI escape sequences to get actual display width
	// ANSI format: \033[...m where ... can be any sequence of digits/semicolons
	stripped := text
	inEscape := false
	displayWidth := 0

	for i := 0; i < len(stripped); i++ {
		if stripped[i] == '\033' && i+1 < len(stripped) && stripped[i+1] == '[' {
			inEscape = true
			i++ // Skip the '['
			continue
		}

		if inEscape {
			if stripped[i] == 'm' {
				inEscape = false
			}
			continue
		}

		displayWidth++
	}

	return displayWidth
}
