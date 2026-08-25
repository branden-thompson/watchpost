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

	"github.com/branden-thompson/watchpost/domains/locations/coverage"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
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

// Resolve returns the best match for a query, with zip and timezone populated
// (R-2': the rendered label always carries a zip when one exists).
func (g *Geocoder) Resolve(ctx context.Context, query string) (snapshot.LocationRef, error) {
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
	if _, err := g.client.GetJSON(ctx, u, &payload); err != nil {
		return snapshot.LocationRef{}, fmt.Errorf("geocoding %q: %w", query, err)
	}
	if len(payload.Results) == 0 {
		return snapshot.LocationRef{}, fmt.Errorf("no location found for %q — try 'City, ST' or a zip code", query)
	}
	r := payload.Results[0]
	if r.CountryCode != "" && !coverage.NWS(r.CountryCode) { // the same answer as the offline index (round 2 N-2)
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
		if p == query {
			zip = p
			break
		}
		if zip == "" || p < zip {
			zip = p
		}
	}
	return snapshot.LocationRef{Label: label, Zip: zip, Lat: r.Latitude, Lon: r.Longitude, TZ: r.Timezone}, nil
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
