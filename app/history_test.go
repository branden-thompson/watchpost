package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fakeHour is NDFD's current hour for any lattice: every point 20 °C, the
// wind 10 km/h from 270°; or its error, while fail is set.
type fakeHour struct {
	asked atomic.Int32
	fail  atomic.Bool
}

func (f *fakeHour) hour(_ context.Context, l temperature.Lattice, now time.Time) (temperature.Series, error) {
	f.asked.Add(1)
	if f.fail.Load() {
		return temperature.Series{}, errors.New("NDFD did not answer")
	}
	n := l.Cols * l.Rows
	fill := func(v float64) []float64 {
		out := make([]float64, n)
		for i := range out {
			out[i] = v
		}
		return out
	}
	return temperature.Series{Lattice: l, Hours: []time.Time{now.UTC().Truncate(time.Hour)},
		Hourly: [][]float64{fill(20)}, Feels: [][]float64{fill(19)}, WindSpeed: [][]float64{fill(10)}, WindFrom: [][]float64{fill(270)}, WindGust: [][]float64{fill(30)}}, nil
}

func historianAt(t *testing.T, now *time.Time, f *fakeHour, regions ...string) *historian {
	t.Helper()
	return &historian{store: history.Open(t.TempDir(), func() time.Time { return *now }, ndfdHourly), hour: f.hour, regions: func() []string { return regions }}
}

// THE HISTORY RECORDS EVERY BOX OF THE REGION, EACH HOUR ONCE (W18.3b, D-166,
// D-172): the recorder's passes every five minutes fetch an hour only while
// it is not yet recorded; the record holds NDFD's fields over the box.
func TestTheHistoryRecordsEachBoxOnceAnHour(t *testing.T) {
	now := time.Date(2026, 9, 30, 15, 5, 0, 0, time.UTC)
	f := &fakeHour{}
	h := historianAt(t, &now, f, geo.RegionContiguous)
	boxes := recordedBoxes(geo.RegionContiguous)
	if len(boxes) != 9 {
		t.Fatalf("the lower 48 records %d field boxes; want the whole region's and the grid's eight", len(boxes))
	}
	h.pass(context.Background(), now)
	now = now.Add(historyEvery)
	h.pass(context.Background(), now)
	if got := int(f.asked.Load()); got != len(boxes) {
		t.Errorf("NDFD was asked %d times over two passes of one hour; want once a box, %d", got, len(boxes))
	}
	for _, b := range boxes {
		rec, ok := h.store.Get(ndfdHourly.Name, history.Key{Source: "ndfd", Place: b.Name}, now)
		if !ok {
			t.Errorf("%s's hour was not recorded", b.Name)
			continue
		}
		if rec.Values["temp"][0] != 20 || rec.Values["feels"][0] != 19 || rec.Values["wind"][0] != 10 || rec.Values["wind_from"][0] != 270 || rec.Values["gust"][0] != 30 {
			t.Errorf("%s recorded %v", b.Name, rec.Values)
		}
	}
	now = now.Add(time.Hour)
	h.pass(context.Background(), now)
	if got := int(f.asked.Load()); got != 2*len(boxes) {
		t.Errorf("the next hour asked %d in all; want each box again, %d", got, 2*len(boxes))
	}
}

// AN HOUR NDFD DID NOT ANSWER IS ASKED AGAIN once its claim has gone stale -
// not left unrecorded, and not asked on every pass meanwhile.
func TestAnUnansweredHourIsAskedAgain(t *testing.T) {
	now := time.Date(2026, 9, 30, 15, 1, 0, 0, time.UTC)
	f := &fakeHour{}
	f.fail.Store(true)
	h := historianAt(t, &now, f, geo.RegionHawaii)
	h.pass(context.Background(), now)
	first := f.asked.Load()
	if first == 0 {
		t.Fatal("nothing was asked")
	}
	now = now.Add(historyEvery)
	h.pass(context.Background(), now)
	if f.asked.Load() != first {
		t.Errorf("a failed box was asked again within its claim: %d then %d", first, f.asked.Load())
	}
	f.fail.Store(false)
	now = now.Add(15 * time.Minute) // past the claim's 10 minutes, still the same hour
	h.pass(context.Background(), now)
	boxes := recordedBoxes(geo.RegionHawaii)
	if _, ok := h.store.Get(ndfdHourly.Name, history.Key{Source: "ndfd", Place: boxes[0].Name}, now); !ok {
		t.Error("the hour was not recorded once NDFD answered")
	}
}

// THE REGIONS RECORDED ARE THE STATION'S AND THE MAP'S LAST (D-172), once.
func TestTheHistoryRecordsTheStationsRegionAndTheMaps(t *testing.T) {
	lp := &livePipelines{station: stationArea{transmitter: snapshot.LocationRef{Lat: 33.29, Lon: -117.22}}}
	if got := lp.historyRegions(); len(got) != 1 || got[0] != geo.RegionContiguous {
		t.Errorf("the station alone records %v", got)
	}
	lp.lastMapRegion.Store(geo.RegionContiguous)
	if got := lp.historyRegions(); len(got) != 1 {
		t.Errorf("the map in the station's region records %v; want it once", got)
	}
	lp.lastMapRegion.Store(geo.RegionHawaii)
	if got := lp.historyRegions(); len(got) != 2 || got[1] != geo.RegionHawaii {
		t.Errorf("the map elsewhere records %v; want the station's and Hawaii", got)
	}
}
