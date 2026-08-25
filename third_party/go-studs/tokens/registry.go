package tokens

import (
	"fmt"
	"strings"
	"sync"
)

// TokenRegistry provides thread-safe token resolution with caching
type TokenRegistry struct {
	cache   map[string]string // Cached resolved ANSI codes
	mutex   sync.RWMutex      // Thread-safety for cache operations
	maxSize int               // Maximum cache size (0 = unlimited)
}

// NewTokenRegistry creates a new token registry with optional cache size limit
func NewTokenRegistry() *TokenRegistry {
	return &TokenRegistry{
		cache:   make(map[string]string),
		maxSize: 1000, // Default cache limit for most frequently used tokens
	}
}

// ResolveToANSI resolves a semantic token through the complete token chain to ANSI code
// Semantic Token -> Design Token -> Raw Color -> ANSI Code
func (r *TokenRegistry) ResolveToANSI(semanticToken string) (string, error) {
	// Check cache first (thread-safe read)
	r.mutex.RLock()
	if cached, exists := r.cache[semanticToken]; exists {
		r.mutex.RUnlock()
		return cached, nil
	}
	r.mutex.RUnlock()

	// Resolve semantic token to design token
	designToken, exists := GetSemanticToken(semanticToken)
	if !exists {
		return "", fmt.Errorf("semantic token '%s' not found", semanticToken)
	}

	// Resolve design token to raw color
	rawColorKey, exists := GetDesignToken(designToken)
	if !exists {
		return "", fmt.Errorf("design token '%s' not found for semantic token '%s'", designToken, semanticToken)
	}

	// Resolve raw color to actual color value
	colorValue, exists := GetRawColor(rawColorKey)
	if !exists {
		return "", fmt.Errorf("raw color '%s' not found for design token '%s'", rawColorKey, designToken)
	}

	// Convert color value to ANSI code based on format
	ansiCode, err := r.convertToANSI(colorValue)
	if err != nil {
		return "", fmt.Errorf("failed to convert color '%s' to ANSI: %w", colorValue, err)
	}

	// Cache the result (thread-safe write)
	r.mutex.Lock()
	r.cacheResult(semanticToken, ansiCode)
	r.mutex.Unlock()

	return ansiCode, nil
}

// ResolveToANSIUncached resolves without using cache (for testing/debugging)
func (r *TokenRegistry) ResolveToANSIUncached(semanticToken string) (string, error) {
	designToken, exists := GetSemanticToken(semanticToken)
	if !exists {
		return "", fmt.Errorf("semantic token '%s' not found", semanticToken)
	}

	rawColorKey, exists := GetDesignToken(designToken)
	if !exists {
		return "", fmt.Errorf("design token '%s' not found", designToken)
	}

	colorValue, exists := GetRawColor(rawColorKey)
	if !exists {
		return "", fmt.Errorf("raw color '%s' not found", rawColorKey)
	}

	return r.convertToANSI(colorValue)
}

// convertToANSI converts a color value to ANSI escape sequence code
func (r *TokenRegistry) convertToANSI(colorValue string) (string, error) {
	format := DetectColorFormat(colorValue)

	switch format {
	case FormatANSI:
		// Direct ANSI code - just return as-is
		return colorValue, nil

	case FormatHEX:
		// Convert HEX to closest ANSI 256-color approximation
		// For now, map common HEX colors to ANSI equivalents
		return r.hexToANSI(colorValue)

	case FormatRGB:
		// Convert RGB to closest ANSI 256-color approximation
		return r.rgbToANSI(colorValue)

	default:
		return "", fmt.Errorf("unsupported color format: %s", colorValue)
	}
}

// hexToANSI converts HEX colors to ANSI codes (basic implementation)
func (r *TokenRegistry) hexToANSI(hexColor string) (string, error) {
	// Basic HEX to ANSI mapping for common colors
	hexToANSI := map[string]string{
		"#06B6D4": "96", // Cyan-500 -> Bright cyan
		"#8B5CF6": "95", // Violet-500 -> Bright magenta
		"#10B981": "92", // Emerald-500 -> Bright green
		"#F59E0B": "93", // Amber-500 -> Bright yellow
		"#EF4444": "91", // Red-500 -> Bright red
		"#3B82F6": "94", // Blue-500 -> Bright blue
	}

	if ansiCode, exists := hexToANSI[hexColor]; exists {
		return ansiCode, nil
	}

	// Default to bright white for unmapped HEX colors
	return "97", nil
}

// rgbToANSI converts RGB colors to ANSI codes (basic implementation)
func (r *TokenRegistry) rgbToANSI(rgbColor string) (string, error) {
	// Parse RGB values
	parts := strings.Split(rgbColor, ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid RGB format: %s", rgbColor)
	}

	// For now, map to basic ANSI colors based on dominant RGB component
	// This is a simplified implementation - could be enhanced with full 256-color conversion
	return "97", nil // Default to bright white
}

// cacheResult stores a resolved token in the cache with size management
func (r *TokenRegistry) cacheResult(semanticToken, ansiCode string) {
	// Simple cache size management - remove oldest entries when at limit
	if r.maxSize > 0 && len(r.cache) >= r.maxSize {
		// Remove first entry (simple FIFO, could be enhanced with LRU)
		for k := range r.cache {
			delete(r.cache, k)
			break
		}
	}
	r.cache[semanticToken] = ansiCode
}

// ClearCache clears all cached token resolutions
func (r *TokenRegistry) ClearCache() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.cache = make(map[string]string)
}

// GetCacheStats returns cache statistics for monitoring
func (r *TokenRegistry) GetCacheStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"cache_size":     len(r.cache),
		"max_cache_size": r.maxSize,
		"cache_usage":    float64(len(r.cache)) / float64(r.maxSize),
	}
}

// ValidateTokenChain validates that a complete token resolution chain is valid
func (r *TokenRegistry) ValidateTokenChain(semanticToken string) error {
	designToken, exists := GetSemanticToken(semanticToken)
	if !exists {
		return fmt.Errorf("invalid semantic token: %s", semanticToken)
	}

	rawColorKey, exists := GetDesignToken(designToken)
	if !exists {
		return fmt.Errorf("invalid design token: %s (from semantic token: %s)", designToken, semanticToken)
	}

	_, exists = GetRawColor(rawColorKey)
	if !exists {
		return fmt.Errorf("invalid raw color: %s (from design token: %s)", rawColorKey, designToken)
	}

	return nil
}
