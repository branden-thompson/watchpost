// Package temperature is the map's temperature sources (0.18.0 W10, F-182,
// D-93): the NWS's forecast grid (NDFD) and Open-Meteo, each asked for a
// lattice of points over a fixed box - never the listener's view (D-47) - and
// answering the hours and the days the map steps through (D-94, D-96, D-97).
//
// A POINT A SOURCE HAS NO VALUE FOR IS MISSING, NEVER ZERO. NDFD answers nil
// offshore, past its grid's edge; a zero there would draw a freezing band on
// the sea. Missing is NaN, which the map library draws as no data.
package temperature

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Source is one temperature source (W10.2).
type Source interface {
	Name() string
	// Covers reports whether the source has a region (geo's names).
	Covers(region string) bool
	// Fetch is the lattice's temperatures at now: its hours, and its days
	// from today.
	Fetch(ctx context.Context, l Lattice, now time.Time) (Series, error)
}

// Getter is the one call a source makes: the temperature client's.
type Getter interface {
	GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error)
}

// ErrNotCovered is a region the source does not have.
var ErrNotCovered = errors.New("temperature: the source does not cover this region")

// MaxPoints is the most points a lattice has. NDFD answers only the first 100
// points of a request, silently (measured 2026-09-26); Open-Meteo counts every
// point as a call against its free limit of 10,000 a day (D-93), and 80 keeps
// a view across four boxes, refreshed hourly all day, under it.
const MaxPoints = 80

// Lattice is the points a box is asked at: Cols by Rows, rows from the
// north, each west to east, the box's corners among them.
type Lattice struct {
	Name       string
	Box        geo.Box
	Cols, Rows int
}

// LatticeFor is a box's lattice: as square as the box, within MaxPoints.
func LatticeFor(name string, b geo.Box) Lattice {
	return LatticeOf(name, b, MaxPoints)
}

// NDFDMaxPoints is the most points NDFD's lattice has (D-201): four times
// MaxPoints, about twice as dense each way - NDFD is keyless and unmetered, so
// its temperature, feels like and wind reach nearer the coasts and borders.
const NDFDMaxPoints = 4 * MaxPoints

// NDFDLatticeFor is a box's lattice for NDFD's temperature, feels like and
// wind (D-201); asked a hundred points at a time. Open-Meteo, the history's
// waves and NDFD's rain totals keep LatticeFor's.
func NDFDLatticeFor(name string, b geo.Box) Lattice {
	return LatticeOf(name, b, NDFDMaxPoints)
}

// LatticeOf is a box's lattice of at most points points, as square as the
// box allows and never under two a side: a coarser one costs a metered source
// less, which bills every point (D-185, D-192).
func LatticeOf(name string, b geo.Box, points int) Lattice {
	points = max(points, 4)
	w, h := b.E-b.W, b.N-b.S
	cols := min(max(int(math.Sqrt(float64(points)*w/h)), 2), points/2)
	rows := max(points/cols, 2) // cols x (points / cols) is never over points
	return Lattice{Name: name, Box: b, Cols: cols, Rows: rows}
}

// Point is one lattice point.
type Point struct{ Lat, Lon float64 }

// Points are the lattice's points in its order, rounded to hundredths of a
// degree - what is sent, and what the answers are matched back to.
func (l Lattice) Points() []Point {
	out := make([]Point, 0, l.Cols*l.Rows)
	for r := range l.Rows {
		lat := l.Box.N - (l.Box.N-l.Box.S)*float64(r)/float64(l.Rows-1)
		for c := range l.Cols {
			lon := l.Box.W + (l.Box.E-l.Box.W)*float64(c)/float64(l.Cols-1)
			out = append(out, Point{Lat: round2(lat), Lon: round2(lon)})
		}
	}
	return out
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// hoursAhead is how many hours from the current one Open-Meteo is asked
// for: the radar loop's longest horizon, twelve hours (D-114), and the hour
// it ends in - Radar mode's fields are drawn through the loop's hours ahead
// (UAT-2 U2-32).
const hoursAhead = 13

// Days is how many days a series holds: today and the six after it, the map's
// Today, Tomorrow and Day 3 to Day 7 (D-94).
const Days = 7

// Series is what a source answered for a lattice: temperatures in Celsius,
// wind in km/h and the degrees it blows from (W11, D-108). Hourly, WindSpeed
// and WindFrom are by hour then point; High, Low, PeakSpeed and PeakFrom by
// day from today, then point - a day's peak sustained wind and its dominant
// direction. Feels, FeelsHigh and FeelsLow are the apparent temperature
// (D-119), as Hourly, High and Low are the air's. WindGust and PeakGust are
// the gusts, in km/h: each hour's, each day's strongest (D-136). Missing is
// NaN.
type Series struct {
	Lattice             Lattice
	Hours               []time.Time // on the hour, UTC, oldest first
	Hourly              [][]float64
	WindSpeed, WindFrom [][]float64
	WindGust            [][]float64
	PeakGust            [Days][]float64
	// UV and UVMax are the UV index, each hour's and each day's highest
	// (D-137): Open-Meteo's alone; NDFD has none.
	UV                  [][]float64
	UVMax               [Days][]float64
	Feels               [][]float64
	High                [Days][]float64
	Low                 [Days][]float64
	PeakSpeed, PeakFrom [Days][]float64
	FeelsHigh, FeelsLow [Days][]float64
}

// newSeries is a series with every value missing.
func newSeries(l Lattice) Series {
	n := l.Cols * l.Rows
	s := Series{Lattice: l}
	for k := range Days {
		s.High[k], s.Low[k], s.PeakSpeed[k], s.PeakFrom[k] = missing(n), missing(n), missing(n), missing(n)
		s.FeelsHigh[k], s.FeelsLow[k], s.PeakGust[k], s.UVMax[k] = missing(n), missing(n), missing(n), missing(n)
	}
	return s
}

func missing(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	return out
}

// HourIndex is the index of an hour in the series, added in order when it is
// not there: where a filled hour goes (D-119).
func (s *Series) HourIndex(t time.Time) int { return s.hourIndex(t) }

// hourIndex is the index of an hour in the series, adding it in order.
func (s *Series) hourIndex(t time.Time) int {
	t = t.UTC().Truncate(time.Hour)
	for i, h := range s.Hours {
		if h.Equal(t) {
			return i
		}
	}
	at := 0
	for at < len(s.Hours) && s.Hours[at].Before(t) {
		at++
	}
	n := s.Lattice.Cols * s.Lattice.Rows
	insert := func(rows [][]float64) [][]float64 {
		return append(rows[:at], append([][]float64{missing(n)}, rows[at:]...)...)
	}
	s.Hours = append(s.Hours[:at], append([]time.Time{t}, s.Hours[at:]...)...)
	s.Hourly, s.WindSpeed, s.WindFrom, s.Feels = insert(s.Hourly), insert(s.WindSpeed), insert(s.WindFrom), insert(s.Feels)
	s.WindGust, s.UV = insert(s.WindGust), insert(s.UV)
	return at
}

// HourAt is the values of the newest hour at or before t, and whether there
// is one within an hour of it: a frame never shows an hour it is not in.
func (s Series) HourAt(t time.Time) ([]float64, time.Time, bool) {
	for i := len(s.Hours) - 1; i >= 0; i-- {
		if !s.Hours[i].After(t) {
			if t.Sub(s.Hours[i]) >= time.Hour {
				return nil, time.Time{}, false
			}
			return s.Hourly[i], s.Hours[i], true
		}
	}
	return nil, time.Time{}, false
}

// FeelsAt is the apparent temperature of the newest hour at or before t,
// under HourAt's rule (D-119).
func (s Series) FeelsAt(t time.Time) ([]float64, bool) {
	if _, at, ok := s.HourAt(t); ok {
		for i, h := range s.Hours {
			if h.Equal(at) {
				return s.Feels[i], true
			}
		}
	}
	return nil, false
}

// WindAt is the wind of the newest hour at or before t - speeds and the
// directions they blow from - under HourAt's rule: never an hour t is not in.
func (s Series) WindAt(t time.Time) (speed, from []float64, hour time.Time, ok bool) {
	if _, at, ok := s.HourAt(t); ok {
		for i, h := range s.Hours {
			if h.Equal(at) {
				return s.WindSpeed[i], s.WindFrom[i], at, true
			}
		}
	}
	return nil, nil, time.Time{}, false
}

// GustAt is the gusts of the hour WindAt reads (D-136).
func (s Series) GustAt(t time.Time) ([]float64, bool) { return s.rowAt(s.WindGust, t) }

// UVAt is the UV index of the hour HourAt reads (D-137).
func (s Series) UVAt(t time.Time) ([]float64, bool) { return s.rowAt(s.UV, t) }

// rowAt is one hourly measure's row for the hour HourAt reads.
func (s Series) rowAt(rows [][]float64, t time.Time) ([]float64, bool) {
	if _, at, ok := s.HourAt(t); ok {
		for i, h := range s.Hours {
			if h.Equal(at) && i < len(rows) {
				return rows[i], true
			}
		}
	}
	return nil, false
}

// dayOffset is a local date's day from the local today, where both are read
// in the same zone.
func dayOffset(t, now time.Time) int {
	local := now.In(t.Location())
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	return int(d.Sub(today).Hours() / 24)
}

// bodyCap is the most an answer may be: 2 MiB, against measured answers of
// 110 KB for 100 points' seven days.
const bodyCap = 2 << 20

// NewClient is the temperature client: memory only, the body capped as it
// reads, https only, and no dial to a private address - the radar client's
// hardening (W8.5, RK-11).
func NewClient(userAgent string) (*httpx.Client, error) {
	return httpx.New(ClientConfig(userAgent))
}

// ClientConfig is the client's configuration, named so a test holds it.
func ClientConfig(userAgent string) httpx.Config {
	return httpx.Config{UserAgent: userAgent, MaxRetries: 1, CacheDir: "",
		MaxBodyBytes: bodyCap, RefusePrivate: true, HTTPSOnly: true}
}

// untilNextHour is how long an answer is kept: to the next hour, when both
// sources move on - so a lattice is asked once an hour, never more (W10.5).
func untilNextHour(now time.Time) time.Duration {
	return now.Truncate(time.Hour).Add(time.Hour).Sub(now) + time.Minute
}
