package severe

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ProductCode is a product's short form for a narrow line — the NWS product
// identifier the offices use where one is customary (AWIPS PILs such as
// "Special Weather Statement" → SPS, "Tornado Warning" → TOR, VTEC-style
// codes such as "Severe Thunderstorm Watch" → SVA), else the product's
// initials ("Beach Hazards Statement" → BHS). The radio panel's narrow head
// reads "EVENT · SPS · Palomar Mountain, CA" (HUM LEAD UAT 2026-08-28). A
// function, not a global (P10-06).
func ProductCode(product string) string {
	p := strings.TrimSpace(product)
	if code, ok := productCodes()[p]; ok {
		return code
	}
	var initials []rune
	for _, w := range strings.Fields(p) {
		if r, _ := utf8.DecodeRuneInString(w); unicode.IsUpper(r) { // capitalised words only — "of", "and" stay out
			initials = append(initials, r)
		}
	}
	return strings.ToUpper(string(initials))
}

// productCodes are the standard codes for the products the window lists.
func productCodes() map[string]string {
	return map[string]string{
		"Tornado Warning": "TOR", "Tornado Watch": "TOA",
		"Severe Thunderstorm Warning": "SVR", "Severe Thunderstorm Watch": "SVA",
		"Flash Flood Warning": "FFW", "Flash Flood Watch": "FFA", "Flood Warning": "FLW", "Flood Watch": "FLA", "Flood Advisory": "FLY",
		"Extreme Wind Warning": "EWW", "High Wind Warning": "HWW", "High Wind Watch": "HWA", "Wind Advisory": "WIY",
		"Hurricane Warning": "HUW", "Hurricane Watch": "HUA", "Tropical Storm Warning": "TRW", "Tropical Storm Watch": "TRA",
		"Special Weather Statement": "SPS", "Special Marine Warning": "SMW", "Gale Warning": "GLW", "Small Craft Advisory": "SCY",
		"Red Flag Warning": "RFW", "Fire Weather Watch": "FWA",
		"Excessive Heat Warning": "EHW", "Extreme Heat Warning": "EHW", "Heat Advisory": "HTY", "Excessive Heat Watch": "EHA",
		"Winter Storm Warning": "WSW", "Winter Storm Watch": "WSA", "Winter Weather Advisory": "WWY", "Blizzard Warning": "BZW",
		"Dense Fog Advisory": "FGY", "Frost Advisory": "FRY", "Freeze Warning": "FZW", "Wind Chill Warning": "WCW",
		"Coastal Flood Warning": "CFW", "Coastal Flood Advisory": "CFY", "Storm Warning": "SRW",
	}
}
