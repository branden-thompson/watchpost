package tty

// map_hooks.go — the map window's hooks, held once (0.19.6 A-31, D-77): the
// app fills one Hooks in one keyed literal, Config.Map carries it to
// NewDashboard, and the Dashboard keeps the pointer. Each nil hook is that
// part of the map switched off, as each Config field was.

import (
	"context"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Hooks is what the map asks of the app: its data, its places, its
// instruments and its two saves.
type Hooks struct {
	NewMap  func(size tuimaps.Size) (*tuimaps.Map, error) // builds the map at its window's size (the library moves only a sized map); nil = maps off
	MapFeed func(ctx context.Context, ask MapAsk) MapFeed // the alerts the map draws, asked off the UI goroutine
	// MapRadar is the radar the map draws (W8): the whole loop, asked off the
	// UI goroutine, shown once it is in (D-85).
	MapRadar func(ctx context.Context, ask MapAsk) MapRadar
	// MapTemperature is the temperature the map draws (W10): every hour's or
	// step's grids, each with its span, asked off the UI goroutine.
	MapTemperature func(ctx context.Context, ask MapAsk) MapTemperature
	ClearMapData   func() MapCleared // 0.18.0 W3.8: the app empties what the live map cannot reach
	// MapDisk names a map's disk cache - the app's tile directory and cap.
	// Clear map data lets the live map's go while the app empties it off the
	// UI goroutine, and names it again once the app has answered (PF-4).
	MapDisk func(m *tuimaps.Map) error
	// MapCost is the app's estimate of one refresh with the layers as chosen
	// (0.18.0 W1.14, FR-9.2). Arithmetic over the snapshot: it fetches nothing.
	MapCost func(ask MapAsk, on func(key string) bool) MapCost
	// MapAreaName names what is in view from the view's centre and width,
	// for the title (D-64). Nil draws the selected place's name.
	MapAreaName func(centre tuimaps.LonLat, widthKm float64) string
	// PlaceNear names a point anywhere on the globe by the town nearest it
	// (0.19.0 AX-1), "" where none is near: the Propagation mode's centre;
	// country is the town's ISO 3166 code, which the words say a place
	// abroad by (D-231).
	PlaceNear func(lat, lon float64) (label, country string)
	// FindPlace finds a place anywhere on the globe for the Propagation mode
	// (0.19.0 FR-1.10): locally first, and only on a miss online, uncached;
	// the context is cancelled when the listener cancels the find.
	FindPlace func(ctx context.Context, query string) (snapshot.LocationRef, error)
	// PropagationUpdate runs the Propagation mode's update, with the
	// Settings' options - the reference circuit's noise (D-187), the
	// no-readings correction (D-105) and the stations (D-214) - and sends
	// its own result (0.19.0 W4.1, A-30, C-2). ctx is the mode's: leaving the mode
	// ends it. nil fetches nothing.
	PropagationUpdate func(ctx context.Context, opts ionomaps.UpdateOptions) <-chan PropagationResult
	// SavePropagationAck keeps the acknowledgement's wording seen once it is
	// closed; nil keeps nothing (0.19.0 W2.5).
	SavePropagationAck func(int) error
	// Timed is the timing instrument's ear for the map (W14, D-154): nil, and
	// nothing is measured; the app sets it only under WATCHPOST_DEBUG_TIMING=1.
	Timed func(Timing)
	// MapFrame is the map's frame recorder (D-198): nil unless
	// WATCHPOST_DEBUG_MAPFRAMES names a file.
	MapFrame func(MapFrame)
	// MapFrameText asks the frame recorder for each frame's text as well (D-198).
	MapFrameText bool
	// MapProblem takes what went wrong with the map that the listener cannot
	// act on (D-124): the diagnostics', never shown. Nil drops it.
	MapProblem func(string)
	// MapClosed is told when the map window closes - really closes, not a
	// window opened over it (D-106): the app lets the zone store's memory go,
	// the disk tier serving it back on reopen (D-162).
	MapClosed func()
}

// noHooks is the map with no hooks at all, a Config that carries none: one
// value, shared, because nothing writes a hook through a held pointer
// (TestTheHooksAreTakenOnceAndWrittenOnACopy), so it is never written.
var noHooks Hooks
