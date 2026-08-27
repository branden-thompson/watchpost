package render

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestFormatTempUnits(t *testing.T) {
	if got := (Opts{ThinBands: true, Units: UnitF}).Temp(f64(22.8)); got != "73ºF" {
		t.Fatalf("F: %q", got)
	}
	if got := (Opts{ThinBands: true, Units: UnitC}).Temp(f64(22.8)); got != "23ºC" {
		t.Fatalf("C: %q", got)
	}
	if got := (Opts{ThinBands: true, Units: UnitF}).Temp(nil); got != "n/a" {
		t.Fatalf("nil: %q", got)
	}
}

func TestHealthGlyphsAreTextual(t *testing.T) {
	o := Opts{ThinBands: true, Width: 80}
	good := o.HealthGlyph("NWS", snapshot.ProviderOK)
	bad := o.HealthGlyph("NWS", snapshot.ProviderDegraded)
	if !strings.Contains(good, "✔") || !strings.Contains(good, "NWS") {
		t.Fatalf("good glyph: %q", good)
	}
	if !strings.Contains(bad, "✘") && !strings.Contains(bad, "⚠") {
		t.Fatalf("bad glyph must carry a non-color signal: %q", bad)
	}
	// ASCII mode swaps glyphs, never drops the signal (RS-14).
	ascii := (Opts{ThinBands: true, Width: 80, ASCII: true}).HealthGlyph("NWS", snapshot.ProviderOK)
	if strings.ContainsAny(ascii, "✔⚠✘") || !strings.Contains(ascii, "NWS") {
		t.Fatalf("ascii glyph: %q", ascii)
	}
}

func TestConditionVocabularyAndClamping(t *testing.T) {
	// UAT session 3.2: "PARTLY_CLOUDY" (13 cells) overflowed the 12-cell
	// CONDITIONS column and shifted the whole row. The seam maps provider
	// vocabulary to the mock's (P.CLOUDY) and hard-clamps every cell.
	r := testRow()
	r.Conditions = "partly_cloudy"
	r.TomorrowConditions = "MOSTLY_CLOUDY"
	out := stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
	lines := strings.Split(out, "\n")
	row := []rune(lines[2])
	if got := string(row[46 : 46+8]); got != "P.CLOUDY" {
		t.Fatalf("condition vocabulary: got %q want P.CLOUDY\n%s", got, out)
	}
	if got := string(row[86 : 86+8]); got != "M.CLOUDY" {
		t.Fatalf("tomorrow condition: got %q\n%s", got, out)
	}
	// A pathological over-wide value must clamp, never shift the row.
	r.Conditions = "SOMETHING_ABSURDLY_LONG_CONDITION"
	out = stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{r}, 0))
	row = []rune(strings.Split(out, "\n")[2])
	if got := string(row[60 : 60+6]); got != " 73ºF↗" {
		t.Fatalf("NOW must hold col 60 under clamped overflow, got %q\n%s", got, out)
	}
}

func TestLoadingShimmerNotNA(t *testing.T) {
	// UAT 18.2b: pending data shimmers through the 4-phase dot sweep; "n/a"
	// is reserved for truly absent data after load.
	r := LocationRow{Index: 1, Name: "Loading City", Zip: "00000", Loading: true}
	for frame, want := range []string{"...", "·..", ".·.", "..·"} {
		out := stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF, Frame: frame}).LocationTable([]LocationRow{r}, 0))
		row := strings.Split(out, "\n")[2]
		if !strings.Contains(row, want) {
			t.Fatalf("frame %d must show %q:\n%s", frame, want, row)
		}
		if strings.Contains(row, "n/a") {
			t.Fatalf("loading row must never read n/a:\n%s", row)
		}
	}
	// Post-load nil stays honest.
	loaded := stripANSI((Opts{ThinBands: true, Width: 115, Units: UnitF}).LocationTable([]LocationRow{{Index: 1, Name: "X", Zip: "1", Now: f64(20)}}, 0))
	if !strings.Contains(loaded, "n/a") {
		t.Fatalf("post-load missing values must read n/a:\n%s", loaded)
	}
}
