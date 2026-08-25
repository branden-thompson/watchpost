package tokens

import "strings"

// RawColors defines the base color definitions supporting ANSI, HEX, and RGB formats
// This forms the foundation layer of the 3-layer design token architecture
var RawColors = map[string]string{
	// ==========================================================================
	// BASIC ANSI - Standard 16-color codes (0-15)
	// ==========================================================================
	"ansi-0": "0", // Reset/default

	// ==========================================================================
	// ANSI DARK THEME - Bright variants for dark backgrounds (90-97)
	// ==========================================================================
	"ansi-90": "90", // Dark gray (secondary text, dividers)
	"ansi-91": "91", // Bright red (errors)
	"ansi-92": "92", // Bright green (success)
	"ansi-93": "93", // Bright yellow (warnings)
	"ansi-94": "94", // Bright blue (links, info)
	"ansi-95": "95", // Bright magenta (headers)
	"ansi-96": "96", // Bright cyan (highlights)
	"ansi-97": "97", // Bright white (primary text)

	// ==========================================================================
	// ANSI LIGHT THEME - Darker variants for light backgrounds (30-37)
	// ==========================================================================
	"ansi-30": "30", // Black/dark gray (primary text on light bg)
	"ansi-31": "31", // Regular red (darker than 91)
	"ansi-32": "32", // Regular green (darker than 92)
	"ansi-33": "33", // Regular yellow/orange (darker than 93)
	"ansi-34": "34", // Regular blue (darker than 94)
	"ansi-35": "35", // Regular magenta (darker than 95)
	"ansi-36": "36", // Regular cyan (darker than 96)
	"ansi-37": "37", // Light gray (dividers on light bg)

	// ==========================================================================
	// 256-COLOR PALETTE - Dark theme accent colors
	// ==========================================================================
	"ansi-48":  "48",  // Spring green (flag green #0FD)
	"ansi-51":  "51",  // Bright cyan (canary indicator)
	"ansi-73":  "73",  // Teal (manager on-call)
	"ansi-135": "135", // Light purple (table headers)
	"ansi-177": "177", // Light violet (virtual crew)
	"ansi-196": "196", // Deep red (primary on-call)
	"ansi-198": "198", // Hot pink (brand pink)
	"ansi-202": "202", // Orange-red (secondary on-call)
	"ansi-208": "208", // Orange (commands)
	"ansi-245": "245", // Medium gray (muted/secondary text — replaces theme-dependent ansi-90 for readability, ui-improvements)
	"ansi-84":  "84",  // Fixed green (multi-port GOOD — 12.81:1 on dark; 84 avoids the basic bg range 40-47 that broke 46, UAT 2026-07-22)
	"ansi-151": "151", // Pale green (stale-GOOD — 10.45:1 on dark)
	"ansi-204": "204", // Light red/pink (SICK roll-up — 5.75:1 on dark)
	"ansi-220": "220", // Gold (DEGRADED roll-up — 11.89:1 on dark)
	"ansi-229": "229", // Straw (stale-DEGRADED — 16.02:1 on dark)

	// ==========================================================================
	// 256-COLOR PALETTE - Light theme accent colors (darker variants)
	// ==========================================================================
	"ansi-22":  "22",  // Dark green (flag green for light bg)
	"ansi-23":  "23",  // Dark cyan (manager on-call for light bg)
	"ansi-54":  "54",  // Dark purple (table headers for light bg)
	"ansi-90x": "90",  // Dark gray (virtual crew for light bg)
	"ansi-124": "124", // Dark red (primary on-call for light bg)
	"ansi-125": "125", // Dark magenta (brand for light bg)
	"ansi-130": "130", // Dark orange (commands for light bg)
	"ansi-166": "166", // Dark orange-red (secondary on-call for light bg)

	// HEX colors - Dark theme (web-compatible, brighter for dark backgrounds)
	"hex-06b6d4": "#06B6D4", // Cyan-500
	"hex-8b5cf6": "#8B5CF6", // Violet-500
	"hex-10b981": "#10B981", // Emerald-500
	"hex-f59e0b": "#F59E0B", // Amber-500
	"hex-ef4444": "#EF4444", // Red-500
	"hex-3b82f6": "#3B82F6", // Blue-500

	// HEX colors - Light theme (darker variants for light backgrounds)
	"hex-0891b2": "#0891B2", // Cyan-600 (darker primary)
	"hex-7c3aed": "#7C3AED", // Violet-600 (darker secondary)
	"hex-d97706": "#D97706", // Amber-600 (darker highlight)
	"hex-2563eb": "#2563EB", // Blue-600 (darker info)

	// RGB colors (programmatic support)
	"rgb-6-182-212":  "6:182:212",  // Cyan-500 RGB
	"rgb-139-92-246": "139:92:246", // Violet-500 RGB
	"rgb-16-185-129": "16:185:129", // Emerald-500 RGB
	"rgb-245-158-11": "245:158:11", // Amber-500 RGB
	"rgb-239-68-68":  "239:68:68",  // Red-500 RGB
	"rgb-59-130-246": "59:130:246", // Blue-500 RGB
}

// ColorFormat represents the supported color format types
type ColorFormat int

const (
	FormatANSI ColorFormat = iota
	FormatHEX
	FormatRGB
)

// DetectColorFormat determines the format of a raw color definition
func DetectColorFormat(colorValue string) ColorFormat {
	if len(colorValue) > 0 && colorValue[0] == '#' {
		return FormatHEX
	}
	if strings.Contains(colorValue, ":") {
		return FormatRGB
	}
	return FormatANSI
}

// IsValidRawColor checks if a color key exists in the raw colors registry
func IsValidRawColor(colorKey string) bool {
	_, exists := RawColors[colorKey]
	return exists
}

// GetRawColor retrieves a raw color value by key
func GetRawColor(colorKey string) (string, bool) {
	color, exists := RawColors[colorKey]
	return color, exists
}
