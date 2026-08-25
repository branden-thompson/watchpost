// Package locations resolves user queries (city names, US zips) to
// LocationRefs — embedded-first (geodata index: offline, ~7µs) with the
// Open-Meteo geocoder as online fallback for misses (AI-8 hybrid; a fallback
// resolve carries the geocode_fallback warning so honesty survives — §10.2).
package locations

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/branden-thompson/watchpost/domains/locations/coverage"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/locations/openmeteo"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution lines for the About view (OQ-15).
const Attribution = "GeoNames.org cities & postal codes (CC BY 4.0)" // attribution REQUIRED

var zipRe = regexp.MustCompile(`^\d{5}$`)

// Fallback is the online resolver interface (openmeteo.Geocoder in prod).
type Fallback interface {
	Resolve(ctx context.Context, query string) (snapshot.LocationRef, error)
}

// Resolver combines the embedded index with the online fallback.
type Resolver struct {
	idx      *geodata.Index
	fallback Fallback
}

// New builds a Resolver. fallback may be nil (offline-only mode).
func New(idx *geodata.Index, fallback Fallback) (*Resolver, error) {
	if err := invariant.Check(idx != nil, "resolver requires the embedded index"); err != nil {
		return nil, err
	}
	return &Resolver{idx: idx, fallback: fallback}, nil
}

// Suggestion is one type-ahead hint, label always zip-adorned (R-2′).
type Suggestion struct {
	Display string // "Oceanside, CA (92057)"
	Ref     snapshot.LocationRef
}

// TypeAhead returns ranked hints for a partial query (embedded only — the
// online fallback is never hit per keystroke; AI-8 ToS discipline).
func (r *Resolver) TypeAhead(query string, limit int) []Suggestion {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	if zipRe.MatchString(q) {
		if z, ok := r.idx.Zip(q); ok {
			ref := r.zipToRef(z, q)
			return []Suggestion{{Display: fmt.Sprintf("%s (%s)", ref.Label, q), Ref: ref}}
		}
		return nil
	}
	// A ", ST" qualifier filters hints — "Portland, ME" must never surface
	// Portland, OR first (B2 red-team #1, probe-confirmed).
	name, state, hasState := strings.Cut(q, ",")
	state = strings.ToUpper(strings.TrimSpace(state))
	var out []Suggestion
	for _, c := range r.idx.PrefixSearch(strings.TrimSpace(name), limit*4) {
		if hasState && state != "" && !strings.EqualFold(c.State, state) {
			continue
		}
		if len(out) == limit {
			break
		}
		ref := cityToRefFast(r.idx, c)
		display := ref.Label
		if ref.Zip != "" {
			display = fmt.Sprintf("%s (%s)", ref.Label, ref.Zip)
		}
		out = append(out, Suggestion{Display: display, Ref: ref})
	}
	return out
}

// Resolve returns the best LocationRef for a full query. fellBack reports
// whether the online fallback produced the answer (caller emits the
// geocode_fallback warning).
func (r *Resolver) Resolve(ctx context.Context, query string) (ref snapshot.LocationRef, fellBack bool, err error) {
	q := strings.TrimSpace(query)
	if err := invariant.Check(q != "", "cannot resolve an empty location query"); err != nil {
		return snapshot.LocationRef{}, false, err
	}
	if zipRe.MatchString(q) {
		if z, ok := r.idx.Zip(q); ok {
			return r.zipToRef(z, q), false, nil
		}
	} else if hits := r.idx.PrefixSearch(cityPart(q), 8); len(hits) > 0 {
		if c, ok := pickCity(hits, q); ok {
			if !coverage.NWS(c.Country) { // red-team 0.9.0 F5: NWS has nothing for it — say so, instead of a row that never loads
				return snapshot.LocationRef{}, false, errors.New(coverage.Outside(c.Name + ", " + c.Country))
			}
			return cityToRef(r.idx, c), false, nil
		}
	}
	if r.fallback == nil {
		return snapshot.LocationRef{}, false, fmt.Errorf("no match for %q in the offline index and no network fallback — try 'City, ST' or a 5-digit zip", q)
	}
	ref, err = r.fallback.Resolve(ctx, q)
	if err != nil {
		return snapshot.LocationRef{}, false, err
	}
	return ref, true, nil
}

// cityPart strips a trailing ", ST" qualifier.
func cityPart(q string) string {
	name, _, _ := strings.Cut(q, ",")
	return strings.TrimSpace(name)
}

// pickCity honors an explicit ", ST" qualifier; otherwise takes the top hit
// only when the name matches exactly (prefix-only matches go to fallback —
// "San F" should not silently resolve to San Francisco on a full Resolve).
func pickCity(hits []geodata.City, q string) (geodata.City, bool) {
	name, state, hasState := strings.Cut(q, ",")
	name = strings.TrimSpace(name)
	state = strings.ToUpper(strings.TrimSpace(state))
	for _, c := range hits {
		nameMatch := strings.EqualFold(c.Name, name) || strings.EqualFold(c.ASCII, name)
		if !nameMatch {
			continue
		}
		if hasState && !strings.EqualFold(c.State, state) {
			continue
		}
		return c, true
	}
	return geodata.City{}, false
}

func cityToRef(idx *geodata.Index, c geodata.City) snapshot.LocationRef {
	return snapshot.LocationRef{
		Label: c.Label(),
		Zip:   idx.RepresentativeZip(c),
		Lat:   c.Lat, Lon: c.Lon, TZ: c.TZ,
	}
}

// cityToRefFast is the per-keystroke variant: zip adornment is best-effort
// (place-name lookup only, no O(41k) centroid scan — B2 red-team #3); the
// full Resolve path still computes the definitive representative zip.
func cityToRefFast(idx *geodata.Index, c geodata.City) snapshot.LocationRef {
	return snapshot.LocationRef{
		Label: c.Label(),
		Zip:   idx.RepresentativeZipFast(c),
		Lat:   c.Lat, Lon: c.Lon, TZ: c.TZ,
	}
}

// Seeds returns refs for the n most-populous US cities — the default
// RECENT/SEARCHED table content until real search history exists (B3 UAT
// session 2). Fully offline: embedded geodata + representative zips.
func Seeds(idx *geodata.Index, n int) []snapshot.LocationRef {
	cities := idx.TopUS(n)
	refs := make([]snapshot.LocationRef, 0, len(cities))
	for _, c := range cities {
		ref := cityToRefFast(idx, c)
		if ref.Zip == "" {
			// Startup-only path: the centroid scan (~6ms/miss) is fine here —
			// the per-keystroke budget that bans it does not apply. Catches
			// place-name drift like "New York City" vs zip rows' "New York".
			ref = cityToRef(idx, c)
		}
		refs = append(refs, ref)
	}
	return refs
}

// zipToRef builds a ref from a zip row, backfilling the timezone from the
// city index (zip rows carry none — caught by the B2 PTY verification, which
// saved a default location with tz=”).
func (r *Resolver) zipToRef(z geodata.ZipRow, zip string) snapshot.LocationRef {
	label := z.Place
	if z.State != "" {
		label = z.Place + ", " + z.State
	}
	tz := ""
	for _, c := range r.idx.PrefixSearch(z.Place, 8) {
		if strings.EqualFold(c.State, z.State) && (strings.EqualFold(c.Name, z.Place) || strings.EqualFold(c.ASCII, z.Place)) {
			tz = c.TZ
			break
		}
	}
	return snapshot.LocationRef{Label: label, Zip: zip, Lat: z.Lat, Lon: z.Lon, TZ: tz}
}

var _ Fallback = (*openmeteo.Geocoder)(nil) // compile-time interface pin
