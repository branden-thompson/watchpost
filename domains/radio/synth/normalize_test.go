package synth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Spec: RS-17 goldens — the normalizer de-wraps fixed-width product text,
// drops headers/footers, breaks "..." into sentences and expands the
// directive abbreviations; unknown tokens pass through.

func TestNormalizeZoneForecast(t *testing.T) {
	raw, err := os.ReadFile("testdata/zfp_sgx.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ProductText string `json:"productText"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	paras := Normalize(doc.ProductText)
	joined := strings.Join(paras, "\n")
	for _, bad := range []string{"000", "FPUS56", "ZFPSGX", "$$", "CAZ552-", "\n."} {
		if strings.Contains(joined, bad) {
			t.Fatalf("header/footer/tag must not be spoken (%q):\n%s", bad, joined)
		}
	}
	for _, want := range []string{"Zone Forecast Product for Extreme Southwest California", "Tonight. Mostly clear this evening. Becoming partly cloudy.", "Lows 66 to 69.", "10 miles per hour", "Pacific Daylight Time", "Heat advisory in effect from 10 AM Tuesday to 8 PM Pacific Daylight Time Friday.", "10 miles per hour", "San Diego California"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "...") || strings.Contains(joined, "  ") || strings.Contains(joined, "..") {
		t.Fatalf("ellipses and double spaces must be gone:\n%s", joined)
	}
}

func TestNormalizeKeepsUnknownAbbreviations(t *testing.T) {
	got := Normalize("WINDS NW 15 TO 25 KT WITH GUSTS TO 30 KT. SEAS 6 TO 8 FT. XYZZY REMAINS.")
	if len(got) != 1 || got[0] != "Winds northwest 15 to 25 knots with gusts to 30 knots. Seas 6 to 8 feet. Xyzzy remains." {
		t.Fatalf("got %q", got)
	}
}
