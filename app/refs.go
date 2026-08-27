package app

// refs.go — config ↔ location-ref conversion, the RECENT seed and restore, the keymap override layer. Split from dashboard.go by the quality pass (Q2, pure move).

import (
	"github.com/branden-thompson/watchpost/domains/locations"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// refsFromConfig is the config → request-identity conversion (one builder;
// favourites and the recent stack both ride it, UAT 96).
func refsFromConfig(locs []config.Location) []snapshot.LocationRef {
	refs := make([]snapshot.LocationRef, 0, len(locs))
	for _, l := range locs {
		refs = append(refs, snapshot.LocationRef{Label: l.Label, Tag: l.Tag, Zip: l.Zip, Lat: l.Lat, Lon: l.Lon, TZ: l.TZ})
	}
	return refs
}

// configLocations is the reverse conversion; a missing tag is derived.
func configLocations(refs []snapshot.LocationRef) []config.Location {
	out := make([]config.Location, 0, len(refs))
	for _, r := range refs {
		if r.Tag == "" {
			r.Tag = deriveTag(r.Label)
		}
		out = append(out, config.Location{Label: r.Label, Tag: r.Tag, Zip: r.Zip, Lat: r.Lat, Lon: r.Lon, TZ: r.TZ})
	}
	return out
}

// restoreRecent rebuilds the RECENT / SEARCHED stack at launch (UAT 96):
// the saved stack first (newest first, as saved; anything now a favourite
// drops out), then the seeds fill the room below, deduped by zip, capped.
func restoreRecent(saved, watch, seeds []snapshot.LocationRef, n int) []snapshot.LocationRef {
	used := make(map[string]bool, len(watch))
	for _, r := range watch {
		used[r.Zip] = true
	}
	out := make([]snapshot.LocationRef, 0, n)
	for _, r := range append(append([]snapshot.LocationRef(nil), saved...), seeds...) {
		if used[r.Zip] || len(out) == n {
			continue
		}
		used[r.Zip] = true
		out = append(out, r)
	}
	return out
}

// seedRecent builds the top-N major-city refs for the RECENT/SEARCHED table
// (UAT session 2A: prepopulate so the table is judgeable before real search
// history exists), skipping zips already configured as priority locations.
func seedRecent(idx *geodata.Index, priority []snapshot.LocationRef, n int) []snapshot.LocationRef {
	if idx == nil {
		return nil // the seed list is a nicety; the dashboard renders without it
	}
	used := make(map[string]bool, len(priority))
	for _, r := range priority {
		used[r.Zip] = true
	}
	out := make([]snapshot.LocationRef, 0, n)
	for _, ref := range locations.Seeds(idx, n+len(priority)) {
		if used[ref.Zip] || len(out) == n {
			continue
		}
		ref.Tag = deriveTag(ref.Label)
		out = append(out, ref)
	}
	return out
}

// fullyPopulated reports whether every location has current conditions —
// the M1 "fully-populated multi-location view" definition (brief M1).
func fullyPopulated(s *snapshot.Snapshot) bool {
	if len(s.Locations) == 0 {
		return false
	}
	for _, loc := range s.Locations {
		if loc.Harmonized.Source.Provider == "" {
			return false
		}
	}
	return true
}

// toKeyMap converts the config [keys] table to a term.KeyMap override layer
// (Help text comes from the defaults — B3 ledger item: preserve Help on
// override merge lands with the full keymap config work).
func toKeyMap(keys map[string][]string) term.KeyMap {
	if len(keys) == 0 {
		return nil
	}
	out := term.KeyMap{}
	for action, bindings := range keys {
		out[term.Action(action)] = term.Binding{Keys: bindings}
	}
	return out
}
