package geo

// countries.go - a country's name for the words (0.19.0 D-231): a place
// abroad is said "Tokyo, Japan", never its code letter by letter.

import (
	_ "embed"
	"strings"
)

// iso3166 is the tz database's table of ISO 3166 alpha-2 codes and their
// usual English names - public domain, its departures from ISO-listed names
// stated in its own header - as macOS and every tz install carries it.
//
//go:embed iso3166.tab
var iso3166 string

// CountryName is the usual English name for an ISO 3166 alpha-2 code;
// false for a code the table does not hold.
func CountryName(code string) (string, bool) {
	if len(code) != 2 {
		return "", false
	}
	for line := range strings.Lines(iso3166) {
		got, name, ok := strings.Cut(strings.TrimRight(line, "\n"), "\t")
		if ok && got == code {
			return name, true
		}
	}
	return "", false
}

// SpokenPlace is a place's label as the words say it: "Tokyo, JP" said
// "Tokyo, Japan" when the label ends in the place's known country code - a
// US place's label ends in its state's ("Oceanside, CA"), so it is said as
// written - else the label as written.
func SpokenPlace(label, country string) string {
	if country == "" || !strings.HasSuffix(label, ", "+country) {
		return label
	}
	name, known := CountryName(country)
	if !known {
		return label
	}
	return strings.TrimSuffix(label, country) + name
}
