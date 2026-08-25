package tokens

import (
	"github.com/branden-thompson/watchpost/third_party/go-studs/theme"
)

// currentTheme tracks which theme is currently applied
var currentTheme = theme.ThemeDark

// DarkThemeTokens defines design token mappings optimized for dark terminal backgrounds
// These are the current/default values that work well with light text on dark bg
var DarkThemeTokens = map[string]string{
	// ==========================================================================
	// GRAY SCALE - Primary text and UI elements
	// ==========================================================================
	"color.gray.light": "ansi-97",  // Bright white for primary text
	"color.gray.dark":  "ansi-245", // Muted grey 245 for dividers/borders (was ansi-90)
	"color.gray.muted": "ansi-245", // Muted grey 245 for secondary text (was ansi-90)

	// ==========================================================================
	// BLUE SCALE - Links, highlights, info
	// ==========================================================================
	"color.blue.pure":   "hex-06b6d4", // Primary blue (Cyan-500)
	"color.blue.muted":  "ansi-94",    // Muted blue for secondary elements
	"color.blue.bright": "ansi-96",    // Bright cyan for highlights

	// ==========================================================================
	// STATUS COLORS - Success, error, warning indicators
	// ==========================================================================
	"color.green.success": "ansi-92",  // Bright green for success
	"color.red.error":     "ansi-91",  // Bright red for errors
	"color.red.bright":    "ansi-196", // Deep red (256-color) for critical
	"color.red.muted":     "ansi-202", // Orange-red for warnings

	// ==========================================================================
	// ACCENT COLORS - Commands, headers, syntax highlighting
	// ==========================================================================
	"color.yellow.warning": "ansi-93",  // Bright yellow for warnings
	"color.orange.cmd":     "ansi-208", // Orange (256-color) for commands
	"color.magenta.header": "ansi-95",  // Bright magenta for headers
	"color.magenta.lang":   "ansi-35",  // Regular magenta for language-specific
	"color.purple.light":   "ansi-135", // Light purple for table headers

	// ==========================================================================
	// TABLE UI ELEMENTS - Headers, rows, labels
	// ==========================================================================
	"color.table.header":    "ansi-95",  // Table headers (bright magenta)
	"color.table.row":       "ansi-245", // Row numbers (muted grey 245)
	"color.table.attribute": "ansi-245", // Attribute text (muted grey 245)
	"color.table.label":     "ansi-97",  // Label/name text (bright white)

	// ==========================================================================
	// SPECIAL PURPOSE - Brand colors and reset
	// ==========================================================================
	"color.primary":   "hex-06b6d4", // Primary brand color
	"color.secondary": "hex-8b5cf6", // Secondary brand color
	"color.highlight": "hex-f59e0b", // Highlight color
	"color.info":      "hex-3b82f6", // Information color
	"color.reset":     "ansi-0",     // Reset to default

	// ==========================================================================
	// STATUS ALIASES - Semantic naming for status indicators
	// ==========================================================================
	"color.status.success": "ansi-92", // Alias for color.green.success
	"color.status.warning": "ansi-93", // Alias for color.yellow.warning
	"color.status.error":   "ansi-91", // Alias for color.red.error

	// ==========================================================================
	// BRIGHT COLOR ALIASES - Convenient semantic names
	// ==========================================================================
	"color.yellow.bright":  "ansi-93", // Bright yellow (same as warning)
	"color.magenta.bright": "ansi-95", // Bright magenta for highlights
	"color.cyan.bright":    "ansi-96", // Bright cyan for highlights
}

// LightThemeTokens defines design token mappings optimized for light terminal backgrounds
// These use darker colors for better contrast on light/white backgrounds
var LightThemeTokens = map[string]string{
	// ==========================================================================
	// GRAY SCALE - Primary text and UI elements (inverted for light bg)
	// ==========================================================================
	"color.gray.light": "ansi-30", // Black/dark gray for primary text
	"color.gray.dark":  "ansi-37", // Light gray for dividers (visible on white)
	"color.gray.muted": "ansi-90", // Dark gray for secondary text

	// ==========================================================================
	// BLUE SCALE - Links, highlights, info (darker variants)
	// ==========================================================================
	"color.blue.pure":   "hex-0891b2", // Darker cyan for light bg
	"color.blue.muted":  "ansi-34",    // Regular blue (darker than bright)
	"color.blue.bright": "ansi-36",    // Cyan (darker variant)

	// ==========================================================================
	// STATUS COLORS - Success, error, warning (darker for contrast)
	// ==========================================================================
	"color.green.success": "ansi-32",  // Regular green (darker than 92)
	"color.red.error":     "ansi-31",  // Regular red (darker than 91)
	"color.red.bright":    "ansi-124", // Dark red (256-color)
	"color.red.muted":     "ansi-166", // Dark orange (256-color)

	// ==========================================================================
	// ACCENT COLORS - Commands, headers (darker variants)
	// ==========================================================================
	"color.yellow.warning": "ansi-33",  // Regular yellow/orange (darker)
	"color.orange.cmd":     "ansi-130", // Dark orange (256-color)
	"color.magenta.header": "ansi-35",  // Regular magenta (darker)
	"color.magenta.lang":   "ansi-35",  // Regular magenta
	"color.purple.light":   "ansi-54",  // Dark purple for table headers

	// ==========================================================================
	// TABLE UI ELEMENTS - Headers, rows, labels (adjusted for light bg)
	// ==========================================================================
	"color.table.header":    "ansi-35", // Table headers (regular magenta)
	"color.table.row":       "ansi-90", // Row numbers (dark gray)
	"color.table.attribute": "ansi-90", // Attribute text (dark gray)
	"color.table.label":     "ansi-30", // Label/name text (black)

	// ==========================================================================
	// SPECIAL PURPOSE - Brand colors and reset
	// ==========================================================================
	"color.primary":   "hex-0891b2", // Darker primary for light bg
	"color.secondary": "hex-7c3aed", // Darker secondary for light bg
	"color.highlight": "hex-d97706", // Darker highlight for light bg
	"color.info":      "hex-2563eb", // Darker info for light bg
	"color.reset":     "ansi-0",     // Reset to default

	// ==========================================================================
	// STATUS ALIASES - Semantic naming for status indicators (darker for light bg)
	// ==========================================================================
	"color.status.success": "ansi-32", // Regular green (darker)
	"color.status.warning": "ansi-33", // Regular yellow (darker)
	"color.status.error":   "ansi-31", // Regular red (darker)

	// ==========================================================================
	// BRIGHT COLOR ALIASES - Convenient semantic names (darker for light bg)
	// ==========================================================================
	"color.yellow.bright":  "ansi-33", // Regular yellow/orange (darker)
	"color.magenta.bright": "ansi-35", // Regular magenta (darker)
	"color.cyan.bright":    "ansi-36", // Regular cyan (darker)
}

// ApplyTheme applies the specified theme by updating all design tokens
// and clearing the token registry cache
func ApplyTheme(t theme.Theme) {
	var tokens map[string]string
	if t == theme.ThemeLight {
		tokens = LightThemeTokens
	} else {
		tokens = DarkThemeTokens
	}

	// Update all design tokens
	for key, value := range tokens {
		DesignTokens[key] = value
	}

	// Update current theme tracker
	currentTheme = t
}

// GetCurrentTheme returns the currently applied theme
func GetCurrentTheme() theme.Theme {
	return currentTheme
}

// ResetToDefaultTheme resets to dark theme (the default)
func ResetToDefaultTheme() {
	ApplyTheme(theme.ThemeDark)
}
