package stream

import (
	"context"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// Station is a playable choice for a location: the transmitter, its
// distance, whether it actually covers the location's county, and the
// relay URLs to try in order.
type Station struct {
	*Transmitter
	KM       float64
	Covering bool
	Mounts   []Mount
}

// Resolver joins the table and the directories.
type Resolver struct {
	table *Table
	dir   *Directory
}

// NewResolver builds a resolver; the table loads from the embedded CSV.
func NewResolver(dir *Directory) (*Resolver, error) {
	t, err := LoadTable()
	if err != nil {
		return nil, err
	}
	return &Resolver{table: t, dir: dir}, nil
}

// Resolve orders the playable stations for a location: the transmitter
// covering its county (SAME) first when relayed, then the nearest relayed
// transmitters by distance (up to directoryCandidateCap). Empty when no
// relay carries anything within reach — the caller degrades (text/Synth).
func (r *Resolver) Resolve(ctx context.Context, lat, lon float64, same string) []Station {
	mounts := r.dir.Mounts(ctx)
	var out []Station
	seen := map[string]bool{}
	add := func(tx *Transmitter, covering bool) {
		if seen[tx.Callsign] || len(mounts[tx.Callsign]) == 0 || tx.Status == "OUT OF SERVICE" {
			return
		}
		seen[tx.Callsign] = true
		out = append(out, Station{Transmitter: tx, KM: geo.HaversineKM(lat, lon, tx.Lat, tx.Lon), Covering: covering, Mounts: mounts[tx.Callsign]})
	}
	if same != "" {
		for _, tx := range r.table.Covering(same) {
			add(tx, true)
		}
	}
	for _, n := range r.table.Nearest(lat, lon, r.table.Len()) {
		if len(out) >= directoryCandidateCap {
			break
		}
		add(n.Transmitter, false)
	}
	return out
}

// CoveringTransmitters names the transmitter(s) covering a county even when
// unrelayed — so the UI can say "KEC62 San Diego is not relayed".
func (r *Resolver) CoveringTransmitters(same string) []*Transmitter { return r.table.Covering(same) }

// NearestTransmitter is the closest transmitter of any status, nil when
// the table is empty — the broadcast lead points listeners at it when no
// transmitter covers the county (UAT 112).
func (r *Resolver) NearestTransmitter(lat, lon float64) *Transmitter {
	for _, n := range r.table.Nearest(lat, lon, 1) {
		return n.Transmitter
	}
	return nil
}
