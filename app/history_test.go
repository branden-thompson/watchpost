package app

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
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
		if ok && rec.Shape != shapeOf(temperature.NDFDLatticeFor(b.Name, b.Box)) {
			t.Errorf("%s was recorded %v; want NDFD's lattice, the one the map draws (D-201)", b.Name, rec.Shape)
		}
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

// nowOnly is NDFD as it is: the current hour alone, no hour before it.
type nowOnly struct{ f fakeHour }

func (n *nowOnly) Name() string       { return "NDFD" }
func (n *nowOnly) Covers(string) bool { return true }
func (n *nowOnly) Fetch(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Series, error) {
	return n.f.hour(ctx, l, now)
}

// OPEN-METEO REFUSED, A LOOP'S PAST HOURS ARE DRAWN FROM THE HISTORY (W18.3b,
// D-166, D-173): the hours before the current one that were recorded are
// drawn in their own hours - not the current hour stretched under them - and
// the layers say so with a RECORDED chip after their source's. With nothing
// recorded, the current hour is stretched under the loop (the cold start).
func TestAPastHourIsReplayedFromTheHistory(t *testing.T) {
	now := tempNow
	anchor := now.Truncate(time.Hour)
	store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box) // the map draws NDFD on its own (D-201)
	f := &fakeHour{}
	for _, back := range []time.Duration{2 * time.Hour, time.Hour} {
		s, _ := f.hour(context.Background(), lat, anchor.Add(-back))
		s.Hourly[0][0] = 10 + float64(back/time.Hour) // each hour its own value
		rec, ok := ndfdRecord(s, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-back))
		if !ok || !store.Put(ndfdHourly.Name, rec) {
			t.Fatal("could not seed the history")
		}
	}
	fb := &fallback{src: &nowOnly{}, past: recordedHour(store)}
	got := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: now, failed: true}, nil, tempAsk(false), now, fb)
	// EACH RECORDED HOUR IS DRAWN AS ITSELF: a grid of its own covers it and
	// ends with it - the current hour is never stretched over a recorded one
	// (the earliest may reach back under the loop's start, U2-55).
	own := func(hour time.Time) bool {
		for _, o := range got.Overlays {
			if strings.Contains(o.ID, "/"+box.Name+"/") && !o.During.From.After(hour) && o.During.Until.Before(hour.Add(time.Hour)) && !o.During.Until.Before(hour.Add(time.Hour-time.Second)) {
				return true
			}
		}
		return false
	}
	for _, back := range []time.Duration{2 * time.Hour, time.Hour} {
		if !own(anchor.Add(-back)) {
			t.Errorf("the hour %v before was not replayed as itself", back)
		}
	}
	if chips := got.Chips[tty.TemperatureLayer]; !slices.Equal(chips, []string{"NDFD", "RECORDED"}) {
		t.Errorf("the temperature's chips are %v; want NDFD then RECORDED (D-173)", chips)
	}
	// A RECORD OF ANOTHER BOX IS NOT DRAWN: its points are another place's.
	// (A record on another lattice of this box is put on its points: onLattice.)
	odd := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	moved := lat.Box
	moved.W -= 1
	other := temperature.Lattice{Name: box.Name, Box: moved, Cols: lat.Cols, Rows: lat.Rows}
	oddSeries, _ := f.hour(context.Background(), other, anchor.Add(-time.Hour))
	if rec, ok := ndfdRecord(oddSeries, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-time.Hour)); !ok || !odd.Put(ndfdHourly.Name, rec) {
		t.Fatal("could not seed the odd record")
	}
	if _, n := withRecorded(temperature.Series{Lattice: lat}, recordedHour(odd), box.Name, anchor); n != 0 {
		t.Errorf("a record of another box was replayed (%d)", n)
	}
	cold := buildTemperature(context.Background(), &fakeTemp{name: "Open-Meteo", now: now, failed: true}, nil, tempAsk(false), now, &fallback{src: &nowOnly{}, past: recordedHour(history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly))})
	if slices.Contains(cold.Chips[tty.TemperatureLayer], "RECORDED") {
		t.Error("nothing recorded, yet the chips say RECORDED")
	}
	stretched := false
	for _, o := range cold.Overlays {
		stretched = stretched || !o.During.From.After(anchor.Add(-pastHours*time.Hour)) && o.During.Until.After(anchor)
	}
	if !stretched {
		t.Error("with nothing recorded the current hour is not drawn under all of the loop's past (the cold start, U2-55)")
	}
}

// RADAR MODE ON NDFD DRAWS ITS PAST HOURS FROM THE HISTORY (W19.1, D-185,
// D-190): NDFD has the current hour and the hours ahead, none before; the
// hours recorded are the loop's past hours, each in its own, the chips NDFD
// then RECORDED - and Open-Meteo is not asked. Nothing recorded, the current
// hour is stretched under the loop (the cold start), the chip NDFD alone.
func TestRadarModeOnNDFDDrawsItsPastHoursFromTheHistory(t *testing.T) {
	now := tempNow
	anchor := now.Truncate(time.Hour)
	store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box) // the map draws NDFD on its own (D-201)
	f := &fakeHour{}
	for _, back := range []time.Duration{2 * time.Hour, time.Hour} {
		s, _ := f.hour(context.Background(), lat, anchor.Add(-back))
		rec, ok := ndfdRecord(s, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-back))
		if !ok || !store.Put(ndfdHourly.Name, rec) {
			t.Fatal("could not seed the history")
		}
	}
	om := &fakeTemp{name: "Open-Meteo", now: now}
	got := buildTemperature(context.Background(), &nowOnly{}, om, tempAsk(false), now, &fallback{past: recordedHour(store)})
	until := map[time.Time]bool{} // each recorded hour a grid of its own, ending with it
	for _, o := range got.Overlays {
		if strings.Contains(o.ID, "/"+box.Name+"/") {
			until[o.During.Until.Truncate(time.Hour)] = true
		}
	}
	for _, back := range []time.Duration{2 * time.Hour, time.Hour} {
		if !until[anchor.Add(-back)] {
			t.Errorf("the hour %v before was not drawn from the history: %v", back, until)
		}
	}
	if chips := got.Chips[tty.TemperatureLayer]; !slices.Equal(chips, []string{"NDFD", "RECORDED"}) {
		t.Errorf("the chips are %v; want NDFD then RECORDED", chips)
	}
	if om.asked != 0 {
		t.Errorf("Open-Meteo was asked %d times; NDFD answered", om.asked)
	}
	cold := buildTemperature(context.Background(), &nowOnly{}, om, tempAsk(false), now, &fallback{past: recordedHour(history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly))})
	if chips := cold.Chips[tty.TemperatureLayer]; !slices.Equal(chips, []string{"NDFD"}) {
		t.Errorf("nothing recorded, the chips are %v; want NDFD alone", chips)
	}
	stretched := false
	for _, o := range cold.Overlays {
		stretched = stretched || o.During.From.Before(anchor) && !o.During.Until.Before(anchor)
	}
	if !stretched {
		t.Error("nothing recorded, the current hour is not stretched under the loop (the cold start)")
	}
}

// NDFD'S NEXT-HOUR FEELS-LIKE IS KEPT AS THAT HOUR'S (W19.1, D-188): NDFD
// answers feels-like from the next hour, so an hour's own record would never
// hold one. The recorder keeps the next hour's as that hour's, and when the
// hour comes its record keeps it beside the hour's own fields.
func TestTheNextHoursFeelsLikeIsKeptAsItsOwn(t *testing.T) {
	now := tempNow
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box) // the map draws NDFD on its own (D-201)
	n := lat.Cols * lat.Rows
	all := func(v float64) []float64 {
		out := make([]float64, n)
		for i := range out {
			out[i] = v
		}
		return out
	}
	hour := func(_ context.Context, l temperature.Lattice, at time.Time) (temperature.Series, error) {
		h := at.UTC().Truncate(time.Hour)
		return temperature.Series{Lattice: l, Hours: []time.Time{h, h.Add(time.Hour)},
			Hourly: [][]float64{all(20), all(math.NaN())}, Feels: [][]float64{all(math.NaN()), all(float64(h.Hour()))},
			WindSpeed: [][]float64{all(10), all(math.NaN())}, WindFrom: [][]float64{all(270), all(math.NaN())}, WindGust: [][]float64{all(30), all(math.NaN())}}, nil
	}
	h := &historian{store: history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly), hour: hour}
	key := history.Key{Source: "ndfd", Place: box.Name}
	h.record(context.Background(), lat, now)
	anchor := now.Truncate(time.Hour)
	next, ok := h.store.Get(ndfdHourly.Name, key, anchor.Add(time.Hour))
	if !ok || next.Values["feels"][0] != float64(anchor.Hour()) {
		t.Fatalf("the next hour's feels-like was not kept as its own: %v %v", ok, next.Values)
	}
	now = now.Add(time.Hour)
	h.record(context.Background(), lat, now)
	got, ok := h.store.Get(ndfdHourly.Name, key, anchor.Add(time.Hour))
	if !ok || got.Values["temp"][0] != 20 || got.Values["feels"][0] != float64(anchor.Hour()) {
		t.Errorf("the hour's record is %v (%v); want its own temperature and the feels-like kept for it", got.Values, ok)
	}
}

// NDFD'S NEXT-HOUR WAVES ARE KEPT AS THAT HOUR'S (W19.5, D-194): NDFD's
// waves start at the next hour, so the recorder keeps it as that hour's
// record - drawn when the hour comes - and a box NDFD gives no sea records
// nothing.
func TestNDFDsNextHourWavesAreKeptAsThatHours(t *testing.T) {
	now := tempNow
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box) // the map draws NDFD on its own (D-201)
	f := &fakeHour{}
	h := &historian{store: history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly, ndfdWaves), hour: f.hour,
		waves: fakeWaves{metres: 1.5, max: 2, fromNext: true}.Waves}
	h.record(context.Background(), lat, now)
	key := history.Key{Source: "ndfd", Place: box.Name}
	next := now.Truncate(time.Hour).Add(time.Hour)
	if rec, ok := h.store.Get(ndfdWaves.Name, key, next); !ok || rec.Values["waves"][0] != 1.5 {
		t.Fatalf("the next hour's waves were not kept as its own: %v %v", ok, rec.Values)
	}
	// THE HOURS ON NDFD'S LATTICE, THE WAVES ON THE BOX'S OWN (D-201): the map
	// draws NDFD's temperature on its denser lattice, and merges the waves
	// point by point with Open-Meteo Marine's on the box's.
	if rec, ok := h.store.Get(ndfdHourly.Name, key, now.Truncate(time.Hour)); !ok || rec.Shape != shapeOf(lat) {
		t.Errorf("the hour was recorded %v (%v); want NDFD's lattice %v", rec.Shape, ok, shapeOf(lat))
	}
	if rec, _ := h.store.Get(ndfdWaves.Name, key, next); rec.Shape != shapeOf(temperature.LatticeFor(box.Name, box.Box)) {
		t.Errorf("the waves were recorded %v; want the box's own lattice", rec.Shape)
	}
	dry := &historian{store: history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly, ndfdWaves), hour: f.hour,
		waves: fakeWaves{metres: math.NaN(), max: math.NaN(), fromNext: true}.Waves}
	dry.record(context.Background(), lat, now)
	if _, ok := dry.store.Get(ndfdWaves.Name, key, next); ok {
		t.Error("a box with no sea recorded waves")
	}
}

// FORECAST MODE ON NDFD ASKS OPEN-METEO FOR NOTHING THE HISTORY HOLDS
// (W19.1, D-188, D-189): in the evening NDFD has no Today high or low - the
// day's maximum has passed - and its feels-like starts at the next hour. Now's
// feels-like is the recorded hour's; Today's high and low are the recorded
// hours' highest and lowest; Open-Meteo is not asked. With nothing recorded,
// Now's feels-like is NDFD's next hour, and the empty day is Open-Meteo's.
func TestForecastModeOnNDFDDrawsTodayFromTheHistory(t *testing.T) {
	ask := tempAsk(true)
	ask.TempNDFD = true
	anchor := ask.Anchor
	box := fieldBoxes(ask.Region, ask.View)[0]
	// ON EITHER OF THE BOX'S LATTICES (U2-55): hours recorded on NDFD's and on
	// the box's coarser one fill Today alike.
	for name, lat := range map[string]temperature.Lattice{"NDFD's": temperature.NDFDLatticeFor(box.Name, box.Box), "the old": temperature.LatticeFor(box.Name, box.Box)} {
		t.Run(name, func(t *testing.T) {
			n := lat.Cols * lat.Rows
			all := func(v float64) []float64 {
				out := make([]float64, n)
				for i := range out {
					out[i] = v
				}
				return out
			}
			store := history.Open(t.TempDir(), func() time.Time { return tempNow }, ndfdHourly)
			day := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, anchor.Location())
			for h := day; !h.After(anchor); h = h.Add(time.Hour) {
				temp := 10 + float64(h.Hour())/2 // 10 °C at midnight, 20 at 20:00
				rec := history.Record{Key: history.Key{Source: "ndfd", Place: box.Name}, At: h, IssuedAt: h,
					Shape:  history.Shape{Box: lat.Box, Cols: lat.Cols, Rows: lat.Rows},
					Values: map[string][]float64{"temp": all(temp), "feels": all(temp - 5), "wind": all(10), "wind_from": all(180), "gust": all(20)}}
				if !store.Put(ndfdHourly.Name, rec) {
					t.Fatal("could not seed the history")
				}
			}
			ndfd := temperature.NewNDFD(&costGet{asked: map[string]bool{}, now: tempNow}, "")
			om := &fakeTemp{name: "Open-Meteo", now: tempNow}
			got := buildTemperature(context.Background(), ndfd, om, ask, tempNow, &fallback{past: recordedHour(store)})
			if om.asked != 0 {
				t.Errorf("Open-Meteo was asked %d times; the history held Now and Today", om.asked)
			}
			value := func(os []tuimaps.Overlay, id string) (float64, bool) {
				for _, o := range os {
					if o.ID == id {
						return o.Grid.Values[0], true
					}
				}
				return 0, false
			}
			f := func(c float64) float64 { return c*9/5 + 32 }
			if v, ok := value(got.Feels, tty.FeelsLayer+"/"+box.Name+"/now"); !ok || math.Abs(v-f(15)) > 0.01 {
				t.Errorf("Now's feels-like is %v (%v); want the recorded hour's %v (D-188)", v, ok, f(15))
			}
			if v, ok := value(got.High, tty.TemperatureLayer+"/"+box.Name+"/d0/high"); !ok || math.Abs(v-f(20)) > 0.01 {
				t.Errorf("Today's high is %v (%v); want the recorded hours' highest %v (D-189)", v, ok, f(20))
			}
			if v, ok := value(got.Low, tty.TemperatureLayer+"/"+box.Name+"/d0/low"); !ok || math.Abs(v-f(10)) > 0.01 {
				t.Errorf("Today's low is %v (%v); want the recorded hours' lowest %v (D-189)", v, ok, f(10))
			}
			if v, ok := value(got.FeelsHigh, tty.FeelsLayer+"/"+box.Name+"/d0/high"); !ok || math.Abs(v-61) > 0.01 {
				t.Errorf("Today's feels-like high is %v (%v); want NDFD's own 61 - the history fills only what NDFD left empty", v, ok)
			}
			cold := buildTemperature(context.Background(), ndfd, om, ask, tempNow, &fallback{past: recordedHour(history.Open(t.TempDir(), func() time.Time { return tempNow }, ndfdHourly))})
			if v, ok := value(cold.Feels, tty.FeelsLayer+"/"+box.Name+"/now"); !ok || math.Abs(v-61) > 0.01 {
				t.Errorf("nothing recorded, Now's feels-like is %v (%v); want NDFD's next hour, 61 (D-188)", v, ok)
			}
			if v, ok := value(cold.Low, tty.TemperatureLayer+"/"+box.Name+"/d0/low"); !ok || om.asked == 0 || math.Abs(v-41) > 0.01 {
				t.Error("nothing recorded, the empty Today was not Open-Meteo's (D-189)")
			}
		})
	}
}

// THE DATA TAB'S CHOICES REACH THE RUNNING STORE (D-175, D-177): each preset
// is its duration - the default where the file holds none - applied to the
// store at once; Clear empties it; the usage says its size and place.
func TestTheDataTabsChoicesReachTheStore(t *testing.T) {
	for _, tc := range []struct {
		r            tty.HistoryRetention
		hours, trend time.Duration
	}{
		{tty.HistoryRetention{}, 72 * time.Hour, 30 * 24 * time.Hour},
		{tty.HistoryRetention{Hours: "7d", Trends: "90d"}, 7 * 24 * time.Hour, 90 * 24 * time.Hour},
		{tty.HistoryRetention{Hours: "30d", Trends: "1y"}, 30 * 24 * time.Hour, 365 * 24 * time.Hour},
		{tty.HistoryRetention{Hours: "1y", Trends: "5y"}, 365 * 24 * time.Hour, 5 * 365 * 24 * time.Hour},
		{tty.HistoryRetention{Hours: "junk", Trends: "junk"}, 72 * time.Hour, 30 * 24 * time.Hour},
	} {
		if h, d := historyDurations(tc.r); h != tc.hours || d != tc.trend {
			t.Errorf("%+v keeps %v and %v; want %v and %v", tc.r, h, d, tc.hours, tc.trend)
		}
	}
	now := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	root := t.TempDir()
	lp := &livePipelines{history: history.Open(root, func() time.Time { return now }, historyDatasets...)}
	lp.applyHistory(tty.HistoryRetention{Hours: "7d", Trends: "1y"})
	kept := 0
	for _, d := range lp.history.Catalog() {
		if d.Hours != 7*24*time.Hour || d.Days != 365*24*time.Hour {
			t.Errorf("%s keeps %v and %v; want the chosen 7 days and 1 year - every dataset alike", d.Name, d.Hours, d.Days)
		}
		kept++
	}
	if kept != len(historyDatasets) {
		t.Errorf("%d datasets in the catalog; want %d", kept, len(historyDatasets))
	}
	f := &fakeHour{}
	s, _ := f.hour(context.Background(), temperature.LatticeFor("us-a", geo.Box{W: -120, S: 32, E: -115, N: 36}), now)
	rec, _ := ndfdRecord(s, history.Key{Source: "ndfd", Place: "us-a"}, now)
	lp.history.Put(ndfdHourly.Name, rec)
	if usage := lp.historyUsage(); !strings.Contains(usage, "KB") && !strings.Contains(usage, "MB") {
		t.Errorf("the usage says %q; want its size", usage)
	}
	if err := lp.clearHistory(); err != nil {
		t.Fatal(err)
	}
	if _, ok := lp.history.Get(ndfdHourly.Name, history.Key{Source: "ndfd", Place: "us-a"}, now); ok {
		t.Error("a record survived Clear history")
	}
	if err := (&livePipelines{}).clearHistory(); err != nil {
		t.Errorf("with no store, Clear history failed: %v", err)
	}
}

// THE HISTORY'S SIZE IS READ ONCE, NOT ON EVERY FRAME (U2-48): the Data tab
// says what the history holds, and Settings draws every tab for the window's
// size - a walk of the store each frame, two or three a key press, growing by
// the hour. The size is kept a while; Clear history reads it again.
func TestTheHistorysSizeIsNotReadEveryFrame(t *testing.T) {
	root := t.TempDir()
	lp := &livePipelines{history: history.Open(root, time.Now, historyDatasets...)}
	if err := os.WriteFile(filepath.Join(root, "big.bin"), make([]byte, 3<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	first := lp.historyUsage()
	if err := os.WriteFile(filepath.Join(root, "bigger.bin"), make([]byte, 6<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	if again := lp.historyUsage(); again != first || !strings.Contains(first, "MB") {
		t.Errorf("the size was read again a frame later: %q, then %q", first, again)
	}
	if err := lp.clearHistory(); err != nil {
		t.Fatal(err)
	}
	if cleared := lp.historyUsage(); strings.Contains(cleared, "MB") {
		t.Errorf("after Clear history the size is %q; want it read again, the store emptied", cleared)
	}
}

// EVERY DATASET HAS A BYTE BOUND (D-143, FR-6.5): beside its retention, the
// most it holds on disk.
func TestEveryDatasetHasAByteBound(t *testing.T) {
	for _, d := range historyDatasets {
		if d.MaxBytes <= 0 {
			t.Errorf("%s has no byte bound", d.Name)
		}
	}
}

// THE DATA TAB SAYS WHICH DATASET ITS BOUND CUT SHORT (D-143): the retention
// chosen is not what it keeps, so the tab names it, by its title.
func TestTheDataTabSaysWhichDatasetItsBoundCutShort(t *testing.T) {
	now := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	tight := ndfdHourly
	tight.MaxBytes = 1
	store := history.Open(t.TempDir(), func() time.Time { return now }, tight)
	if words := boundWords(store); words != "" {
		t.Errorf("with nothing cut short the tab says %q", words)
	}
	yesterday := now.AddDate(0, 0, -1)
	if !store.Put(tight.Name, history.Record{Key: history.Key{Source: "ndfd", Place: "us-a"}, At: yesterday, IssuedAt: yesterday,
		Shape: history.Shape{Cols: 1, Rows: 1}, Values: map[string][]float64{"temp": {20}}}) {
		t.Fatal("yesterday's record was not written")
	}
	store.RollUpAndPrune()
	if words := boundWords(store); !strings.Contains(words, ndfdHourly.Title) || !strings.Contains(words, "size limit") {
		t.Errorf("the bound cut %s short and the tab says %q", ndfdHourly.Title, words)
	}
}
