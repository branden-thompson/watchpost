package geo

import "testing"

// TestAPlaceAbroadIsSaidByItsCountry is 0.19.0 D-231 (F-218): a place whose
// country is known and not the US is said with its country's name, from the
// tz database's ISO 3166 table; a US place keeps its state's code - CA is
// California there and Canada in Toronto, which the US region's box takes
// in; a label not ending in the code, or a code the table lacks, is said as
// written.
func TestAPlaceAbroadIsSaidByItsCountry(t *testing.T) {
	for _, c := range []struct {
		label, country, want string
	}{
		{"near Tokyo, JP", "JP", "near Tokyo, Japan"},
		{"Reykjavik, IS", "IS", "Reykjavik, Iceland"},
		{"Toronto, CA", "CA", "Toronto, Canada"},
		{"Oceanside, CA", "US", "Oceanside, CA"},
		{"Oceanside, CA", "", "Oceanside, CA"},
		{"Frankfurt", "DE", "Frankfurt"},
		{"Nowhere, ZZ", "ZZ", "Nowhere, ZZ"},
	} {
		if got := SpokenPlace(c.label, c.country); got != c.want {
			t.Errorf("%q in %q: %q; want %q", c.label, c.country, got, c.want)
		}
	}
	if n, ok := CountryName("GB"); !ok || n == "" {
		t.Errorf("GB: %q, %v", n, ok)
	}
	if _, ok := CountryName("Japan"); ok {
		t.Error("a name taken for a code")
	}
}
