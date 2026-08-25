// Package theme provides terminal theme detection and management for STUDS
// components. It supports automatic detection of light/dark terminal
// backgrounds and allows manual override.
//
// Theme Resolution Priority (Detect):
//  1. Injected cache store (optional — see CacheStore; the library ships NO
//     file persistence of its own, applications inject their own store)
//  2. $COLORFGBG environment variable
//  3. $TERM_PROGRAM heuristics
//  4. Default (dark)
package theme

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Theme represents a color theme for terminal output
type Theme string

const (
	// ThemeDark is the dark terminal theme (light text on dark background)
	ThemeDark Theme = "dark"
	// ThemeLight is the light terminal theme (dark text on light background)
	ThemeLight Theme = "light"
)

// CacheStore is the injectable persistence seam for theme preferences.
// The library itself never touches the filesystem — applications that want
// a cached preference (e.g. a config-dir JSON file) implement this interface
// and inject it via SetCacheStore.
type CacheStore interface {
	// Load returns the cached theme and true when a valid cached
	// preference exists; false when there is none.
	Load() (Theme, bool)
	// Save persists the theme preference with the method that detected it.
	Save(theme Theme, detectionMethod string) error
}

// ThemeManager handles theme detection and application
type ThemeManager struct {
	currentTheme Theme
	cache        CacheStore // nil = no persistence (library default)
	autoDetect   bool
}

// NewThemeManager creates a new ThemeManager with default configuration
// (dark theme, auto-detect enabled, no persistence).
func NewThemeManager() *ThemeManager {
	return &ThemeManager{
		currentTheme: ThemeDark, // Default to dark theme
		autoDetect:   true,
	}
}

// SetCacheStore injects a persistence implementation. Passing nil disables
// persistence (the default).
func (tm *ThemeManager) SetCacheStore(cs CacheStore) {
	tm.cache = cs
}

// GetTheme returns the current theme
func (tm *ThemeManager) GetTheme() Theme {
	return tm.currentTheme
}

// SetTheme sets the current theme
func (tm *ThemeManager) SetTheme(theme Theme) error {
	if theme != ThemeDark && theme != ThemeLight {
		return fmt.Errorf("invalid theme: %s (must be 'dark' or 'light')", theme)
	}
	tm.currentTheme = theme
	return nil
}

// Detect attempts to detect the terminal's theme preference
// using the priority order: injected cache -> $COLORFGBG -> $TERM_PROGRAM -> default
func (tm *ThemeManager) Detect() Theme {
	// 1. Try the injected cache store (if any)
	if tm.cache != nil {
		if theme, ok := tm.cache.Load(); ok {
			tm.currentTheme = theme
			return theme
		}
	}

	// 2. Try $COLORFGBG environment variable
	if theme, detected := tm.detectFromEnv(); detected {
		tm.currentTheme = theme
		return theme
	}

	// 3. Try $TERM_PROGRAM heuristics
	if theme, detected := tm.detectFromTermProgram(); detected {
		tm.currentTheme = theme
		return theme
	}

	// 4. Default to dark theme
	return ThemeDark
}

// detectFromEnv attempts to detect theme from $COLORFGBG environment variable
// Format: "foreground;background" (e.g., "15;0" = white on black = dark mode)
func (tm *ThemeManager) detectFromEnv() (Theme, bool) {
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg == "" {
		return "", false
	}

	// Parse the background color (second value)
	parts := strings.Split(colorfgbg, ";")
	if len(parts) < 2 {
		return "", false
	}

	bgColor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", false
	}

	// Background color >= 8 typically indicates light mode
	// Colors 0-7 are dark colors, 8-15 are light colors in standard ANSI
	if bgColor >= 8 {
		return ThemeLight, true
	}
	return ThemeDark, true
}

// detectFromTermProgram attempts to detect theme based on known terminal defaults
func (tm *ThemeManager) detectFromTermProgram() (Theme, bool) {
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "" {
		return "", false
	}

	// Known terminal defaults
	switch termProgram {
	case "iTerm.app":
		// iTerm2 users typically use dark themes
		return ThemeDark, true
	case "Apple_Terminal":
		// macOS Terminal.app defaults to light theme
		return ThemeLight, true
	case "Hyper":
		// Hyper typically defaults to dark
		return ThemeDark, true
	case "vscode":
		// VS Code terminal inherits from editor theme - check additional vars
		if vsTheme := os.Getenv("VSCODE_TERMINAL_THEME"); vsTheme != "" {
			if strings.Contains(strings.ToLower(vsTheme), "light") {
				return ThemeLight, true
			}
			return ThemeDark, true
		}
		// Default to dark for VS Code
		return ThemeDark, true
	case "WarpTerminal":
		// Warp typically defaults to dark
		return ThemeDark, true
	case "Alacritty":
		// Alacritty typically defaults to dark
		return ThemeDark, true
	default:
		return "", false
	}
}

// IsValid checks if a theme string is valid
func IsValid(theme Theme) bool {
	return theme == ThemeDark || theme == ThemeLight
}
