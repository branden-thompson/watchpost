// Package openmeteo resolves location queries via the Open-Meteo Geocoding API
// (keyless; AI-8). In B1a this is the one-shot resolver behind
// `watchpost report <query>`; B2 adds the embedded GeoNames index in front of
// it (embedded-first, online fallback) plus type-ahead.
package openmeteo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/coverage"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the geocoding credit line for the About view (OQ-15).
const Attribution = "Open-Meteo.com geocoding (CC BY 4.0)" // attribution REQUIRED

// Geocoder resolves free-text queries (city names, US zips) to LocationRefs.
type Geocoder struct {
	client *httpx.Client
	base   string
}

// NewGeocoder builds the resolver. base "" = production endpoint.
func NewGeocoder(client *httpx.Client, base string) *Geocoder {
	if base == "" {
		base = "https://geocoding-api.open-meteo.com"
	}
	return &Geocoder{client: client, base: base}
}

// Geocode returns the best match for a query, with zip and timezone populated
// (R-2': the rendered label always carries a zip when one exists); a place
// the weather does not cover is refused.
func (g *Geocoder) Geocode(ctx context.Context, query string) (snapshot.LocationRef, error) {
	return g.search(ctx, query, false)
}

// ResolveWorld is Geocode for the Propagation mode, which answers anywhere
// (0.19.0 FR-1.10, D-61): any country's place, and the answer never kept -
// the request is not cached, so no query is held on disk.
func (g *Geocoder) ResolveWorld(ctx context.Context, query string) (snapshot.LocationRef, error) {
	return g.search(ctx, query, true)
}

// search asks the geocoder: refusing a place outside the weather's coverage
// unless world, and, for world, keeping no cached answer.
func (g *Geocoder) search(ctx context.Context, query string, world bool) (snapshot.LocationRef, error) {
	if err := invariant.Check(query != "", "cannot resolve an empty location query"); err != nil {
		return snapshot.LocationRef{}, err
	}
	var payload struct {
		Results []struct {
			Name        string   `json:"name"`
			Latitude    float64  `json:"latitude"`
			Longitude   float64  `json:"longitude"`
			CountryCode string   `json:"country_code"`
			Admin1      string   `json:"admin1"`
			Timezone    string   `json:"timezone"`
			Population  int      `json:"population"`
			Postcodes   []string `json:"postcodes"`
		} `json:"results"`
	}
	u := fmt.Sprintf("%s/v1/search?count=5&name=%s", g.base, url.QueryEscape(query))
	var opts []httpx.Option
	if world {
		opts = append(opts, httpx.NoCache()) // never kept (FR-1.10)
	}
	if _, err := g.client.GetJSON(ctx, u, &payload, opts...); err != nil {
		if world { // the Propagation mode's finder: what it finds without the network (AX-10)
			return snapshot.LocationRef{}, LookupFailed{Cause: err}
		}
		return snapshot.LocationRef{}, fmt.Errorf("geocoding: %w", err) // never the query (FR-4.9): the URL's name is redacted
	}
	if len(payload.Results) == 0 {
		if world {
			return snapshot.LocationRef{}, errors.New("no place found by that name - try a city and its country, coordinates or a grid locator") // anywhere on the globe, not 'City, ST' (AX-10)
		}
		return snapshot.LocationRef{}, errors.New("no location found by that name — try 'City, ST' or a zip code")
	}
	r := payload.Results[0]
	r.Name, r.Admin1, r.CountryCode = cleanField(r.Name), cleanField(r.Admin1), cleanField(r.CountryCode) // the provider's text reaches the terminal and the words
	onGlobe := r.Latitude >= -90 && r.Latitude <= 90 && r.Longitude >= -180 && r.Longitude <= 180         // false for NaN too
	if !onGlobe {
		return snapshot.LocationRef{}, errors.New("the geocoder answered with a place off the globe - try another name, coordinates or a grid locator") // RI-2: never marked as found
	}
	if _, err := time.LoadLocation(r.Timezone); err != nil || r.Timezone == "" {
		r.Timezone = "" // a zone Go does not know, or text that is no zone, is not kept (RI-2): the place's own clock is worked out without it
	}
	if !world && r.CountryCode != "" && !coverage.NWS(r.CountryCode) { // the same answer as the offline index (round 2 N-2)
		return snapshot.LocationRef{}, errors.New(coverage.Outside(r.Name + ", " + r.CountryCode))
	}
	label := r.Name
	if r.CountryCode == "US" && r.Admin1 != "" {
		label = fmt.Sprintf("%s, %s", r.Name, abbrevState(r.Admin1))
	} else if r.CountryCode != "" && r.CountryCode != "US" {
		label = fmt.Sprintf("%s, %s", r.Name, r.CountryCode)
	}
	zip := ""
	// Deterministic representative-zip rule (AI-8 §3): if the query itself is
	// one of the place's zips, show that; else the lowest-numbered zip.
	for _, p := range r.Postcodes {
		if !postcodeShape(p, r.CountryCode) {
			continue // the provider's text, drawn and kept in the config: only a postcode's own characters (RI-2)
		}
		if p == query {
			zip = p
			break
		}
		if zip == "" || p < zip {
			zip = p
		}
	}
	return snapshot.LocationRef{Label: label, Zip: zip, Lat: r.Latitude, Lon: r.Longitude, TZ: r.Timezone, Country: r.CountryCode}, nil
}

// abbrevState maps common US state names to postal codes (label compactness);
// unknown names pass through verbatim.
func abbrevState(name string) string {
	m := map[string]string{
		"Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR", "California": "CA",
		"Colorado": "CO", "Connecticut": "CT", "Delaware": "DE", "Florida": "FL", "Georgia": "GA",
		"Hawaii": "HI", "Idaho": "ID", "Illinois": "IL", "Indiana": "IN", "Iowa": "IA",
		"Kansas": "KS", "Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD",
		"Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS", "Missouri": "MO",
		"Montana": "MT", "Nebraska": "NE", "Nevada": "NV", "New Hampshire": "NH", "New Jersey": "NJ",
		"New Mexico": "NM", "New York": "NY", "North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH",
		"Oklahoma": "OK", "Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC",
		"South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT", "Vermont": "VT",
		"Virginia": "VA", "Washington": "WA", "West Virginia": "WV", "Wisconsin": "WI", "Wyoming": "WY",
		"District of Columbia": "DC",
	}
	if ab, ok := m[name]; ok {
		return ab
	}
	return name
}

// cleanField is a geocoder's field made safe to show and say: one line, no
// escape sequence or control character a server could address the terminal
// with, at most a field's length.
func cleanField(s string) string { return plaintext.ClampField(plaintext.Line(s)) }

// postcodeShape is whether a postcode is one: in the US five digits; else
// letters, digits, spaces and hyphens, at most ten (RI-2).
func postcodeShape(p, country string) bool {
	if country == "US" {
		return usZip.MatchString(p)
	}
	return anyPostcode.MatchString(p)
}

// The postcodes' shapes.
var (
	usZip       = regexp.MustCompile(`^[0-9]{5}$`)
	anyPostcode = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 -]{0,9}$`)
)

// LookupFailed is a world lookup the geocoder did not answer: its words are
// the listener's - what is found without it - and its cause, the transport's
// error, is for the diagnostics alone (errors.Unwrap), never said.
type LookupFailed struct{ Cause error }

// Error is the listener's words.
func (e LookupFailed) Error() string {
	return "the online lookup did not answer; coordinates, a grid locator and the world's larger cities are found without it"
}

// Unwrap is the cause, for the diagnostics.
func (e LookupFailed) Unwrap() error { return e.Cause }
