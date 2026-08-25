package nws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// MarineProvider reads the coastal-waters fields the NWS raw gridpoint
// carries for coastal grids (B3 UAT 29, live-probed: primarySwell*,
// waveHeight, wavePeriod, windWaveHeight — WMO units). A grid with none of
// them is inland: the location gets no Marine section. Registered under its
// own provider id so the [S] status view tracks it as a second API.
type MarineProvider struct {
	base *Provider

	mu     sync.Mutex
	inland map[string]time.Time // gridpoint URL -> known inland until (UAT 72)
}

// inlandTTL is how long a grid with no marine series is remembered as
// inland: the 228 KB gridpoint is not re-downloaded for it every cycle
// (40 of 60 seed locations). A day — coastlines do not move.
const inlandTTL = 24 * time.Hour

// NewMarine wraps the weather provider (shares its resolve cache + client).
func NewMarine(base *Provider) *MarineProvider {
	return &MarineProvider{base: base, inland: map[string]time.Time{}}
}

// knownInland reports a grid remembered as inland.
func (m *MarineProvider) knownInland(gridURL string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	until, ok := m.inland[gridURL]
	return ok && time.Now().Before(until)
}

func (m *MarineProvider) markInland(gridURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inland[gridURL] = time.Now().Add(inlandTTL)
}

// ID implements snapshot.Provider.
func (m *MarineProvider) ID() string { return "nws-marine" }

// Domains implements snapshot.Provider.
func (m *MarineProvider) Domains() []string { return []string{"marine"} }

// Fetch implements snapshot.Provider for KindMarine.
func (m *MarineProvider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: m.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindMarine, "nws-marine serves only KindMarine"); err != nil {
		return frag, err
	}
	// Concurrent + fail-soft per location (UAT 59/64): partial failure
	// travels in the Fragment (§10.1); successes always land.
	got, err := snapshot.FetchEach(ctx, req.Locations, fetchConcurrency, func(ctx context.Context, ref snapshot.LocationRef) (snapshot.PartialData, error) {
		mar, err := m.fetchMarine(ctx, ref)
		if err != nil {
			return snapshot.PartialData{}, err
		}
		return snapshot.PartialData{Marine: mar}, nil
	})
	for k, pd := range got {
		if pd.Marine != nil {
			frag.PerLocation[k] = pd
		}
	}
	frag.Err = err
	return frag, nil
}

// gridSeries is one gridpoint property: a unit + a list of timed values.
type gridSeries struct {
	UOM    string `json:"uom"`
	Values []struct {
		ValidTime string   `json:"validTime"`
		Value     *float64 `json:"value"`
	} `json:"values"`
}

// first returns the earliest non-null value of a series.
func (g gridSeries) first() *float64 {
	for _, v := range g.Values {
		if v.Value != nil {
			val := *v.Value
			return &val
		}
	}
	return nil
}

// positive reports a present, non-zero series value.
func positive(v *float64) bool { return v != nil && *v > 0 }

// fetchMarine pulls the raw gridpoint and lifts the marine series. Returns
// nil (no error) for inland grids.
func (m *MarineProvider) fetchMarine(ctx context.Context, ref snapshot.LocationRef) (*snapshot.Marine, error) {
	g, err := m.base.resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := invariant.Check(g.gridURL != "", "nws-marine: point resolved without a gridpoint URL for "+ref.Label); err != nil {
		return nil, err
	}
	var grid struct {
		Properties struct {
			PrimarySwellHeight      gridSeries `json:"primarySwellHeight"`
			PrimarySwellDirection   gridSeries `json:"primarySwellDirection"`
			WaveHeight              gridSeries `json:"waveHeight"`
			WavePeriod              gridSeries `json:"wavePeriod"`
			WindWaveHeight          gridSeries `json:"windWaveHeight"`
			SecondarySwellHeight    gridSeries `json:"secondarySwellHeight"`
			SecondarySwellDirection gridSeries `json:"secondarySwellDirection"`
			WavePeriod2             gridSeries `json:"wavePeriod2"`
		} `json:"properties"`
	}
	if m.knownInland(g.gridURL) {
		return nil, nil // remembered inland (UAT 72): no download
	}
	if _, err := m.base.client.GetJSON(ctx, g.gridURL, &grid); err != nil {
		return nil, fmt.Errorf("gridpoint for %s: %w", ref.Label, err)
	}
	pr := grid.Properties
	mar := &snapshot.Marine{
		SwellHeight:          pr.PrimarySwellHeight.first(),
		SwellDirDeg:          pr.PrimarySwellDirection.first(),
		WaveHeight:           pr.WaveHeight.first(),
		WavePeriod:           pr.WavePeriod.first(),
		WindWaveHeight:       pr.WindWaveHeight.first(),
		SecondarySwellHeight: pr.SecondarySwellHeight.first(),
		SecondarySwellDirDeg: pr.SecondarySwellDirection.first(),
		SecondaryPeriod:      pr.WavePeriod2.first(),
		ObservedAt:           time.Now().UTC(),
		Source:               snapshot.SourceInfo{Provider: m.ID()},
	}
	if !positive(mar.SwellHeight) && !positive(mar.WaveHeight) {
		m.markInland(g.gridURL)
		// Inland grid. Live finding (UAT 29): cells one step off the coast
		// still publish the marine series - as all zeros - so presence is
		// not coastal; a real swell or wave height is.
		return nil, nil
	}
	return mar, nil
}
