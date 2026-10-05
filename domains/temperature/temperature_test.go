package temperature

// temperature_test.go — 0.18.0 W10.1 to W10.3 over the recorded answers
// (D-93, D-96, D-97): the parsers, the lattice, the interpolation and the
// client's hardening.

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// fakeGet answers each request with the fixture its address names, and
// records the addresses.
type fakeGet struct {
	t    *testing.T
	asks []string
}

func (f *fakeGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	f.asks = append(f.asks, rawURL)
	switch {
	case strings.Contains(rawURL, "open-meteo") || strings.Contains(rawURL, "/v1/forecast"):
		return fixture(f.t, "openmeteo.json"), nil
	case strings.Contains(rawURL, "temp=temp"):
		return fixture(f.t, "ndfd-hour.xml"), nil
	}
	return fixture(f.t, "ndfd-days.xml"), nil
}

func fixture(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// captured is when the fixtures were recorded: 18:09 in San Diego.
var captured = time.Date(2026, 9, 27, 1, 9, 0, 0, time.UTC)

// fixtureLattice is the fixtures' 3x2 lattice: San Diego's coast and the sea.
var fixtureLattice = Lattice{Name: "fixture", Box: geo.Box{W: -121, S: 31, E: -117, N: 33}, Cols: 3, Rows: 2}

// escondido is the one point NDFD answers (33N 117W, inland); the others are
// the sea or Mexico, past its grid.
const escondido = 2

func cOf(f float64) float64 { return (f - 32) * 5 / 9 }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestEveryRecordedTemperatureFixtureIsPresent(t *testing.T) {
	var m struct {
		Captured string   `json:"captured"`
		Files    []string `json:"files"`
	}
	if err := json.Unmarshal(fixture(t, "manifest.json"), &m); err != nil || m.Captured == "" || len(m.Files) != 17 { // every recorded fixture, wind, gusts, UV and history included (D-136, D-137, D-168)
		t.Fatalf("the manifest is %+v (%v)", m, err)
	}
	for _, f := range m.Files {
		if len(fixture(t, f)) == 0 {
			t.Errorf("%s is empty", f)
		}
	}
}

func TestNDFDReadsTheDaysAndTheCurrentHour(t *testing.T) {
	get := &fakeGet{t: t}
	s, err := NewNDFD(get, "").Fetch(context.Background(), fixtureLattice, captured)
	if err != nil {
		t.Fatal(err)
	}
	// After 20:00 local nothing is left of today's day: the first maximum is
	// tomorrow's, and the first minimum the night that ends tomorrow morning.
	if !math.IsNaN(s.High[0][escondido]) {
		t.Errorf("today's high is %v; it has passed, and NDFD gave none", s.High[0][escondido])
	}
	if !near(s.High[1][escondido], cOf(86)) || !near(s.High[2][escondido], cOf(80)) {
		t.Errorf("the highs are %v, %v; want 86F and 80F", s.High[1][escondido], s.High[2][escondido])
	}
	if !near(s.Low[1][escondido], cOf(68)) {
		t.Errorf("tomorrow's low is %v; want 68F, the night ending tomorrow morning", s.Low[1][escondido])
	}
	hour, at, ok := s.HourAt(captured)
	if !ok || !at.Equal(captured.Truncate(time.Hour)) || !near(hour[escondido], cOf(80)) {
		t.Errorf("the current hour is %v at %v (%v); want 80F at 01:00Z", hour, at, ok)
	}
	// OFFSHORE IS MISSING, NEVER ZERO.
	for _, p := range []int{0, 1, 3, 4, 5} {
		if !math.IsNaN(s.High[1][p]) || !math.IsNaN(hour[p]) {
			t.Errorf("point %d is %v / %v; NDFD answered nil, which is missing", p, s.High[1][p], hour[p])
		}
	}
}

func TestNDFDSendsTheHourWithItsZone(t *testing.T) {
	get := &fakeGet{t: t}
	if _, err := NewNDFD(get, "").Fetch(context.Background(), fixtureLattice, captured); err != nil {
		t.Fatal(err)
	}
	if len(get.asks) != 2 {
		t.Fatalf("NDFD was asked %d times; want 2, the days then the hours", len(get.asks))
	}
	for _, a := range get.asks {
		if !strings.HasPrefix(a, "https://graphical.weather.gov/") {
			t.Errorf("asked %s: not NDFD's host", a)
		}
		u, _ := url.Parse(a)
		if got := len(strings.Fields(u.Query().Get("listLatLon"))); got != 6 {
			t.Errorf("asked for %d points; want the lattice's 6", got)
		}
	}
	u, _ := url.Parse(get.asks[1])
	if b := u.Query().Get("begin"); b != "2026-09-27T01:00:00Z" {
		t.Errorf("begin is %q; without its Z NDFD reads it as each point's local time", b)
	}
	if q := u.Query(); q.Get("wspd") == "" || q.Get("wdir") == "" {
		t.Error("the current hour's request asks no wind: the days' answer starts at the next hour, and Now would have none (W11)")
	}
}

func TestOpenMeteoReadsThePastHoursAndTheSea(t *testing.T) {
	get := &fakeGet{t: t}
	s, err := NewOpenMeteo(get, "").Fetch(context.Background(), fixtureLattice, captured)
	if err != nil {
		t.Fatal(err)
	}
	// The sea's points answer in Etc/GMT+8 and the land's in Los Angeles time,
	// 14:00 and 15:00: both 22:00Z, read each with its own offset.
	if len(s.Hours) != 5 || !s.Hours[0].Equal(time.Date(2026, 9, 26, 22, 0, 0, 0, time.UTC)) {
		t.Fatalf("the hours are %v; want five from 22:00Z, each point's local time read with its offset", s.Hours)
	}
	if !near(s.Hourly[0][0], 18.4) || !near(s.Hourly[4][0], 17.9) {
		t.Errorf("the sea's hours are %v .. %v; want 18.4 .. 17.9", s.Hourly[0][0], s.Hourly[4][0])
	}
	// Two hours back is what Radar mode's oldest frames draw (D-96).
	if v, at, ok := s.HourAt(captured.Add(-2 * time.Hour)); !ok || !at.Equal(time.Date(2026, 9, 26, 23, 0, 0, 0, time.UTC)) || math.IsNaN(v[escondido]) {
		t.Errorf("two hours back is %v at %v (%v)", v, at, ok)
	}
	if !near(s.High[0][escondido], 34.5) || !near(s.High[1][escondido], 28.5) || !near(s.Low[0][escondido], 19.4) {
		t.Errorf("the days are high %v, %v and low %v; want 34.5, 28.5 and 19.4", s.High[0][escondido], s.High[1][escondido], s.Low[0][escondido])
	}
	if len(get.asks) != 1 || !strings.HasPrefix(get.asks[0], "https://api.open-meteo.com/v1/forecast?") {
		t.Errorf("asked %v; want one request to Open-Meteo", get.asks)
	}
}

func TestAnAnswerForOtherPointsIsRefused(t *testing.T) {
	s := newSeries(Lattice{Box: fixtureLattice.Box, Cols: 2, Rows: 2})
	if err := parseOpenMeteo(fixture(t, "openmeteo.json"), captured, &s); err == nil {
		t.Error("six points answered for four asked was read")
	}
}

func TestAnHourIsNeverShownOutsideItself(t *testing.T) {
	s := newSeries(fixtureLattice)
	at := time.Date(2026, 9, 27, 1, 0, 0, 0, time.UTC)
	s.Hourly[s.hourIndex(at)][0] = 1
	if _, _, ok := s.HourAt(at.Add(-time.Minute)); ok {
		t.Error("a moment before the first hour drew it")
	}
	if _, _, ok := s.HourAt(at.Add(time.Hour)); ok {
		t.Error("a moment an hour on drew the hour before it")
	}
	if _, got, ok := s.HourAt(at.Add(59 * time.Minute)); !ok || !got.Equal(at) {
		t.Error("a moment inside the hour did not draw it")
	}
}

func TestEveryBoxsLatticeFitsBothSources(t *testing.T) {
	for _, b := range []geo.Box{{W: -126, S: 23, E: -65, N: 51}, {W: -126, S: 23, E: -110.75, N: 37}, {W: -176, S: 50, E: -126, N: 72},
		{W: -164, S: 15, E: -151, N: 26}, {W: 140, S: 9, E: 150, N: 18}} {
		l := LatticeFor("b", b)
		pts := l.Points()
		if len(pts) > MaxPoints || l.Cols < 2 || l.Rows < 2 {
			t.Errorf("%v: %dx%d, %d points; NDFD answers 100 at most and Open-Meteo charges each", b, l.Cols, l.Rows, len(pts))
		}
		if first, last := pts[0], pts[len(pts)-1]; first != (Point{Lat: b.N, Lon: b.W}) || last != (Point{Lat: b.S, Lon: b.E}) {
			t.Errorf("%v: the corners are %v and %v; the lattice must reach the box's edges, or boxes side by side leave a seam", b, first, last)
		}
	}
}

func TestInterpolationIsExactOnAPlane(t *testing.T) {
	l := Lattice{Box: geo.Box{W: -100, S: 30, E: -90, N: 40}, Cols: 3, Rows: 3}
	plane := func(lat, lon float64) float64 { return 2*lat + lon }
	var vals []float64
	for _, p := range l.Points() {
		vals = append(vals, plane(p.Lat, p.Lon))
	}
	f := l.Interpolate(vals)
	if f.Cols != 2*Fine || f.Rows != 2*Fine || len(f.Values) != f.Cols*f.Rows {
		t.Fatalf("the field is %dx%d with %d values", f.Cols, f.Rows, len(f.Values))
	}
	for r := range f.Rows {
		for c := range f.Cols {
			lat := 40 - 10*(float64(r)+0.5)/float64(f.Rows)
			lon := -100 + 10*(float64(c)+0.5)/float64(f.Cols)
			if got := f.Values[r*f.Cols+c]; math.Abs(got-plane(lat, lon)) > 1e-9 {
				t.Fatalf("cell %d,%d is %v; want %v", r, c, got, plane(lat, lon))
			}
		}
	}
}

// TestAMissingPointIsLeftOutNotZero is D-101: a source is drawn only as far
// as its own points reach. A cell whose nearest point has no value is
// missing - no square extrapolated past NDFD's grid (UAT-2 U2-17) - and a
// cell whose nearest point has one is weighed from the points that do.
func TestAMissingPointIsLeftOutNotZero(t *testing.T) {
	l := Lattice{Box: geo.Box{W: 0, S: 0, E: 1, N: 1}, Cols: 2, Rows: 2}
	nan := math.NaN()
	f := l.Interpolate([]float64{10, nan, 10, 10}) // the north-east point missing
	for r := range f.Rows {
		for c := range f.Cols {
			v := f.Values[r*f.Cols+c]
			nearNE := r < f.Rows/2 && c >= f.Cols/2
			switch {
			case nearNE && !math.IsNaN(v):
				t.Fatalf("cell %d,%d, nearest the missing point, is %v; want missing", r, c, v)
			case !nearNE && !near(v, 10):
				t.Fatalf("cell %d,%d is %v; the three points left all say 10", r, c, v)
			}
		}
	}
	f = l.Interpolate([]float64{nan, nan, nan, nan})
	for i, v := range f.Values {
		if !math.IsNaN(v) {
			t.Fatalf("cell %d is %v with no point around it; want missing", i, v)
		}
	}
}

func TestTheClientIsHardened(t *testing.T) {
	c := ClientConfig("watchpost/test", "/cache/map-http")
	if !c.RefusePrivate || !c.HTTPSOnly || c.MaxBodyBytes != bodyCap {
		t.Errorf("the client is %+v; want public addresses, https only, capped", c)
	}
	// AND IT KEEPS WHAT IT PAID FOR (D-219): on disk at the directory given,
	// within the stated cap, and with room in memory for several regions' hours.
	if c.CacheDir != "/cache/map-http" || c.DiskBytes != CacheBytes || c.MemBytes != memBytes || c.MemBytes/4 < bodyCap {
		t.Errorf("the client keeps %q (%d on disk, %d in memory); want the directory given, %d, %d", c.CacheDir, c.DiskBytes, c.MemBytes, CacheBytes, memBytes)
	}
	if got := untilNextHour(time.Date(2026, 9, 27, 1, 50, 0, 0, time.UTC)); got != 11*time.Minute {
		t.Errorf("an answer at 01:50 is kept %v; want to 02:01", got)
	}
}

func TestTheSourcesAreTheClosedList(t *testing.T) {
	want := map[string]string{"NWS NDFD": "https://graphical.weather.gov", "Open-Meteo": "https://api.open-meteo.com", "Open-Meteo Marine": "https://marine-api.open-meteo.com"} // D-125
	got := Hosts()
	if len(got) != len(want) {
		t.Fatalf("the hosts are %v; want %v (FR-3.8, D-93)", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s is %q; want %q", k, got[k], v)
		}
	}
	if NewNDFD(nil, "").Covers(geo.RegionSamoa) || !NewOpenMeteo(nil, "").Covers(geo.RegionSamoa) {
		t.Error("NDFD has no American Samoa; Open-Meteo has everywhere")
	}
}

// windGet answers with the wind fixtures.
type windGet struct{ t *testing.T }

func (w windGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	if strings.Contains(rawURL, "/v1/forecast") {
		return fixture(w.t, "openmeteo-wind.json"), nil
	}
	return fixture(w.t, "ndfd-days-wind.xml"), nil
}

// windCaptured is when the wind fixtures were recorded: 10:39 in San Diego.
var windCaptured = time.Date(2026, 9, 27, 17, 39, 0, 0, time.UTC)

// TestOpenMeteoReadsTheWind is W11.1 (D-108): each hour's speed and where it
// blows from, and each day's peak and dominant direction, in km/h.
func TestOpenMeteoReadsTheWind(t *testing.T) {
	s, err := NewOpenMeteo(windGet{t}, "").Fetch(context.Background(), fixtureLattice, windCaptured)
	if err != nil {
		t.Fatal(err)
	}
	speed, from, at, ok := s.WindAt(windCaptured)
	if !ok || !at.Equal(time.Date(2026, 9, 27, 17, 0, 0, 0, time.UTC)) || !near(speed[escondido], 8.6) || !near(from[escondido], 182) {
		t.Errorf("the wind now is %v from %v at %v (%v); want 8.6 km/h from 182 at 17:00Z", speed, from, at, ok)
	}
	if !near(s.PeakSpeed[0][escondido], 14.8) || !near(s.PeakFrom[0][escondido], 198) || !near(s.PeakSpeed[1][escondido], 19.5) {
		t.Errorf("the peaks are %v from %v, then %v; want 14.8 from 198, then 19.5", s.PeakSpeed[0][escondido], s.PeakFrom[0][escondido], s.PeakSpeed[1][escondido])
	}
}

// TestNDFDWorksOutEachDaysPeakWind is W11.1: NDFD has no daily wind; each
// day's peak is its strongest hour, in km/h from knots, and the direction then.
func TestNDFDWorksOutEachDaysPeakWind(t *testing.T) {
	s, err := NewNDFD(windGet{t}, "").Fetch(context.Background(), fixtureLattice, windCaptured)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[int][2]float64{0: {7, 220}, 1: {10, 250}, 6: {4, 260}} {
		if !near(s.PeakSpeed[k][escondido], want[0]*1.852) || !near(s.PeakFrom[k][escondido], want[1]) {
			t.Errorf("day %d's peak is %v from %v; want %v kt (%v km/h) from %v", k, s.PeakSpeed[k][escondido], s.PeakFrom[k][escondido], want[0], want[0]*1.852, want[1])
		}
	}
	missing := 0 // NDFD answers a few of the sea's hours nil: missing, never calm
	for _, h := range s.WindSpeed {
		if math.IsNaN(h[0]) {
			missing++
		}
	}
	if missing == 0 {
		t.Error("none of the sea's nil hours is missing: a nil read as calm")
	}
}

// TestWindIsInterpolatedAsAVector is W11.1: between a wind from 350 degrees
// and one from 10, the wind is from the north - averaged as numbers, the two
// would meet at 180, the wind turned round.
func TestWindIsInterpolatedAsAVector(t *testing.T) {
	l := Lattice{Box: geo.Box{W: 0, S: 0, E: 1, N: 1}, Cols: 2, Rows: 2}
	f, dirs := l.InterpolateWind([]float64{20, 20, 20, 20}, []float64{350, 10, 350, 10})
	mid := f.Cols / 2
	if d := dirs[mid]; !(d > 355 || d < 5) || math.IsNaN(d) {
		t.Errorf("between 350 and 10 degrees the wind is from %v; want the north", d)
	}
	if !near(f.Values[mid], 20) {
		t.Errorf("the speed is %v; want 20", f.Values[mid])
	}
	_, dirs = l.InterpolateWind([]float64{20, math.NaN(), 20, 20}, []float64{350, 10, 350, 10})
	if !math.IsNaN(dirs[f.Cols-1]) {
		t.Error("a cell nearest a point with no wind has a direction")
	}
}

// gustGet answers from the gust fixtures (D-136): NDFD's days and hour,
// Open-Meteo's forecast; asked keeps each address.
type gustGet struct {
	t     *testing.T
	asked *[]string
}

func (g gustGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	if g.asked != nil {
		*g.asked = append(*g.asked, rawURL)
	}
	switch {
	case strings.Contains(rawURL, "/v1/forecast"):
		return fixture(g.t, "openmeteo-gust.json"), nil
	case strings.Contains(rawURL, "begin="):
		return fixture(g.t, "ndfd-hour-gust.xml"), nil
	}
	return fixture(g.t, "ndfd-days-gust.xml"), nil
}

// gustCaptured is when the gust fixtures were recorded: 16:09 in San Diego.
var gustCaptured = time.Date(2026, 9, 28, 23, 9, 49, 0, time.UTC)

// TestNDFDReadsTheGusts is D-136: NDFD's wgust, hourly in knots, read as
// km/h beside the sustained wind, and each day's strongest worked out.
func TestNDFDReadsTheGusts(t *testing.T) {
	var asked []string
	s, err := NewNDFD(gustGet{t, &asked}, "").Fetch(context.Background(), fixtureLattice, gustCaptured)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range asked { // the days and the current hour both ask for it
		if !strings.Contains(u, "wgust=wgust") {
			t.Errorf("an NDFD request does not ask for the gusts: %s", u)
		}
	}
	gust, ok := s.GustAt(gustCaptured)
	if !ok || !near(gust[escondido], 16*1.852) {
		t.Errorf("the gust now is %v (%v); want 16 kt, %v km/h", gust, ok, 16*1.852)
	}
	if !near(s.PeakGust[0][escondido], 16*1.852) || !near(s.PeakGust[1][escondido], 15*1.852) {
		t.Errorf("the days' strongest gusts are %v, %v; want 16 kt, then 15", s.PeakGust[0][escondido], s.PeakGust[1][escondido])
	}
}

// TestOpenMeteoReadsTheGusts is D-136: Open-Meteo's hourly gusts and each
// day's strongest, in km/h.
func TestOpenMeteoReadsTheGusts(t *testing.T) {
	var asked []string
	s, err := NewOpenMeteo(gustGet{t, &asked}, "").Fetch(context.Background(), fixtureLattice, gustCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || !strings.Contains(asked[0], "wind_gusts_10m%2C") && !strings.Contains(asked[0], "wind_gusts_10m&") || !strings.Contains(asked[0], "wind_gusts_10m_max") {
		t.Errorf("Open-Meteo is not asked for the hours' and the days' gusts: %v", asked)
	}
	if gust, ok := s.GustAt(gustCaptured); !ok || !near(gust[escondido], 16.9) {
		t.Errorf("the gust now is %v (%v); want 16.9 km/h", gust, ok)
	}
	if !near(s.PeakGust[0][escondido], 20.2) || !near(s.PeakGust[1][escondido], 34.2) {
		t.Errorf("the days' strongest gusts are %v, %v; want 20.2, then 34.2", s.PeakGust[0][escondido], s.PeakGust[1][escondido])
	}
}

// fixedGet answers every request with one fixture, and keeps each address.
type fixedGet struct {
	t     *testing.T
	name  string
	asked *[]string
}

func (g fixedGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	*g.asked = append(*g.asked, rawURL)
	return fixture(g.t, g.name), nil
}

// uvCaptured is when the UV and air-quality fixtures were recorded: 17:51 in
// San Diego.
var uvCaptured = time.Date(2026, 9, 29, 0, 51, 5, 0, time.UTC)

// TestOpenMeteoReadsTheUV is D-137: the UV index of each hour and each day's
// highest, in the request temperature already makes.
func TestOpenMeteoReadsTheUV(t *testing.T) {
	var asked []string
	s, err := NewOpenMeteo(fixedGet{t, "openmeteo-uv.json", &asked}, "").Fetch(context.Background(), fixtureLattice, uvCaptured)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Fatalf("asked %v; want one request", asked)
	}
	u, _ := url.Parse(asked[0])
	if !slices.Contains(strings.Split(u.Query().Get("hourly"), ","), "uv_index") || !slices.Contains(strings.Split(u.Query().Get("daily"), ","), "uv_index_max") {
		t.Errorf("the request does not ask for each hour's UV and each day's highest: %s", asked[0])
	}
	if uv, ok := s.UVAt(uvCaptured); !ok || !near(uv[escondido], 2.45) {
		t.Errorf("the UV now is %v (%v); want 2.45 at 17:00 local", uv, ok)
	}
	if !near(s.UVMax[0][escondido], 6.9) || !near(s.UVMax[1][escondido], 6.8) {
		t.Errorf("the days' highest UV are %v, %v; want 6.9, then 6.8", s.UVMax[0][escondido], s.UVMax[1][escondido])
	}
}

// TestOpenMeteoReadsTheAirQuality is D-138: the model's US AQI of each hour,
// from the air-quality API's host, and each day's worst hour on the point's
// THE CURRENT HOUR ALONE, FOR THE HISTORY (W18.3b, D-166): one ask - the very
// address Fetch's second ask is, so the two share the HTTP cache and an hour
// the map already fetched costs the recorder nothing - read as Fetch reads it.
func TestNDFDsCurrentHourIsFetchsOwnAsk(t *testing.T) {
	hourGet, fetchGet := &fakeGet{t: t}, &fakeGet{t: t}
	s, err := NewNDFD(hourGet, "").Hour(context.Background(), fixtureLattice, captured)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewNDFD(fetchGet, "").Fetch(context.Background(), fixtureLattice, captured); err != nil {
		t.Fatal(err)
	}
	if len(hourGet.asks) != 1 || hourGet.asks[0] != fetchGet.asks[1] {
		t.Fatalf("the hour asked %v; want Fetch's own second ask, %s", hourGet.asks, fetchGet.asks[1])
	}
	hour, at, ok := s.HourAt(captured)
	if !ok || !at.Equal(captured.Truncate(time.Hour)) || !near(hour[escondido], cOf(80)) {
		t.Errorf("the current hour is %v at %v (%v); want 80F at 01:00Z", hour, at, ok)
	}
}
