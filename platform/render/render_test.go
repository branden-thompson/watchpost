package render

import (
	"strconv"
	"strings"
)

// Spec: D-9 seam (ONLY package importing go-studs — table rides go-studs
// DataTable per UAT session 2 ruling), mock M-V1 anatomy at measured absolute
// offsets, UAT-2D responsive breakpoints, D-19 (f/c live units), R-12a
// (severity/health never color-only).

func f64(v float64) *float64 { return &v }

// runeIdx is strings.Index in display columns (the header's ♪ and the rows'
// º are multibyte - byte offsets lie).
func runeIdx(s, tok string) int {
	b := strings.Index(s, tok)
	if b < 0 {
		return -1
	}
	return len([]rune(s[:b]))
}

func testRow() LocationRow {
	return LocationRow{
		Index: 1, Name: "Oceanside, CA", Tag: "OSIDE", Zip: "92057",
		Conditions: "CLEAR", Now: f64(22.8), Hi: f64(23.9), Lo: f64(17.2), Trend: "up",
		TomorrowConditions: "RAIN", TomorrowHi: f64(25.0), TomorrowLo: f64(18.0),
		HasAlert: true, Selected: true, Playing: true,
	}
}

// validRawSGR reports whether a composite token value is a legal SGR
// parameter list for the raw path (sgrRaw emits it verbatim): attributes,
// basic colours, and fully-qualified 38;5;n / 48;5;n / 38;2;r;g;b /
// 48;2;r;g;b — never a bare 256-palette index, which the terminal ignores.
func validRawSGR(v string) bool {
	e := strings.Split(v, ";")
	for i := 0; i < len(e); i++ {
		if (e[i] == "38" || e[i] == "48") && i+1 < len(e) {
			switch e[i+1] {
			case "5":
				i += 2
				continue
			case "2":
				i += 4
				continue
			}
		}
		n, err := strconv.Atoi(e[i])
		if err != nil || !legalSGRAttr(n) {
			return false
		}
	}
	return true
}

// legalSGRAttr: attributes, basic and bright colours, defaults.
func legalSGRAttr(n int) bool {
	return n <= 9 || (n >= 21 && n <= 29) || (n >= 30 && n <= 37) || n == 39 || (n >= 40 && n <= 47) || n == 49 || (n >= 90 && n <= 97) || (n >= 100 && n <= 107)
}

// The mock's two header lines (test-only: the goldens pin the render against them).
const (
	mockGroupHeader = "           [   L O C A T I O N   /   S T A T I O N    ]     [           T O D A Y             ]     [    T O M O R R O W     ]"
	mockColHeader   = "  ♪        ###. NAME                 LABEL    ZIP      CONDITIONS    NOW       HI     LOW      CONDITIONS    HI      LOW"
)
