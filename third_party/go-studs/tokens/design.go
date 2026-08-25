package tokens

// DesignTokens defines the design token layer mapping semantic color names to raw colors
// This is the middle layer of the 3-layer design token architecture
var DesignTokens = map[string]string{
	// ==========================================================================
	// GRAY SCALE - Primary text and UI elements
	// ==========================================================================
	"color.gray.dark":  "ansi-245", // Medium gray (245) for secondary text, dividers — was ansi-90 (theme-dependent)
	"color.gray.light": "ansi-97",  // Bright white for primary text, headers
	"color.gray.muted": "ansi-245", // Muted text, attributes, descriptions — 256-palette 245 (was ansi-90)

	// ==========================================================================
	// BLUE SCALE - Links, highlights, info
	// ==========================================================================
	"color.blue.pure":   "hex-06b6d4", // Primary blue (Cyan-500)
	"color.blue.muted":  "ansi-94",    // Muted blue for secondary elements
	"color.blue.bright": "ansi-96",    // Bright cyan for highlights

	// ==========================================================================
	// STATUS COLORS - Success, error, warning indicators
	// ==========================================================================
	"color.green.success": "ansi-92",  // Success indicators, completed states
	"color.red.error":     "ansi-91",  // Error indicators, failed states
	"color.red.bright":    "ansi-196", // Deep red for critical states
	"color.red.muted":     "ansi-202", // Orange-red for warnings

	// ==========================================================================
	// ACCENT COLORS - Commands, headers, syntax highlighting
	// ==========================================================================
	"color.yellow.warning": "ansi-93",  // Warning states, JavaScript
	"color.orange.cmd":     "ansi-208", // Command indicators, PRODUCTION READY
	"color.magenta.header": "ansi-95",  // App names, paths (bright magenta)
	"color.magenta.lang":   "ansi-35",  // Language-specific (regular magenta)
	"color.purple.light":   "ansi-135", // Light purple for table headers

	// ==========================================================================
	// TABLE UI ELEMENTS - Headers, rows, labels
	// ==========================================================================
	"color.table.header":    "ansi-95",  // Table headers (bright magenta)
	"color.table.row":       "ansi-245", // Row numbers (muted grey 245)
	"color.table.attribute": "ansi-245", // Attribute text (muted grey 245)
	"color.table.label":     "ansi-97",  // Label/name text (bright white)

	// ==========================================================================
	// SPECIAL PURPOSE - Legacy compatibility
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

// IsValidDesignToken checks if a design token key exists
func IsValidDesignToken(tokenKey string) bool {
	_, exists := DesignTokens[tokenKey]
	return exists
}

// GetDesignToken retrieves a design token value by key
func GetDesignToken(tokenKey string) (string, bool) {
	token, exists := DesignTokens[tokenKey]
	return token, exists
}

// UpdateDesignToken allows runtime updates to design tokens for theme switching
func UpdateDesignToken(tokenKey, rawColorKey string) bool {
	if !IsValidRawColor(rawColorKey) {
		return false
	}
	DesignTokens[tokenKey] = rawColorKey
	return true
}

// GetAllDesignTokens returns a copy of all design tokens for inspection
func GetAllDesignTokens() map[string]string {
	tokens := make(map[string]string, len(DesignTokens))
	for k, v := range DesignTokens {
		tokens[k] = v
	}
	return tokens
}
