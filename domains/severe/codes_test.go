package severe

import "testing"

func TestProductCodeIsTheStandardCodeOrTheInitials(t *testing.T) {
	for in, want := range map[string]string{
		"Special Weather Statement": "SPS", "Tornado Warning": "TOR", "Severe Thunderstorm Watch": "SVA", "Heat Advisory": "HTY",
		"Extreme Heat Warning": "EHW", "Dust Storm Warning": "DSW", "Tropical Storm Dolly": "TSD", "": "",
	} {
		if got := ProductCode(in); got != want {
			t.Errorf("ProductCode(%q) = %q, want %q", in, got, want)
		}
	}
}
