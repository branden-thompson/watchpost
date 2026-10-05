package app

// maprain.go — the map's rain and snow ahead (0.18.0 W12.2, W12.3; D-115 to
// D-118), all Open-Meteo's: outside the lower 48 the radar loop's hours ahead,
// its hourly rain painted in radar's scale; and Forecast mode's days, each
// day's heaviest hour in radar's colours with its totals marked on it.

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/units"
)

// Forecast mode's rain and snow, registered: ON BY DEFAULT (D-117), so
// Forecast mode mirrors Radar mode - the rain in radar's colours, the
// temperature and the rest the tints under it.
func init() {
	registerMapLayer(mapLayer{key: tty.RainLayer, label: "Rain & snow", on: true, cost: rainLayerCost})
}

// modelRainNote is what the loop says of hours ahead that are Open-Meteo's
// (D-115).
const modelRainNote = "The hours ahead are Open-Meteo's model rain, not radar."

// withModelRain adds the loop's hours ahead where HRRR is not (D-115): a
// forecast loop a field box of Open-Meteo's hourly rain, a frame an hour
// after the newest observed frame and up to the horizon, painted in radar's
// scale. Each hour's value is the rain of the hour before it, so a frame on
// the hour shows the hour it closes.
func withModelRain(ctx context.Context, out tty.MapRadar, om *temperature.OpenMeteo, region string, view geo.Box, newest, until time.Time) tty.MapRadar {
	hours := int(math.Ceil(until.Sub(newest).Hours()))
	var loops []tuimaps.Overlay
	failed := false
	for _, b := range fieldBoxes(region, view) {
		lat := temperature.LatticeFor(b.Name, b.Box)
		r, err := om.Rain(ctx, lat, newest, hours)
		if err != nil {
			failed = true
			continue
		}
		var frames []tuimaps.LoopFrame
		for i, h := range r.Hours {
			if !h.After(newest) || h.After(until) {
				continue
			}
			frames = append(frames, tuimaps.LoopFrame{Valid: h, PNG: ratePNG(lat.Interpolate(r.Hourly[i])), Forecast: true})
		}
		if len(frames) == 0 {
			continue
		}
		o := tuimaps.RadarImage(tty.RadarLayer+"/fc-"+b.Name, tuimaps.Image{Frames: frames, Table: rateTable, Exact: true,
			West: b.W, South: b.S, East: b.E, North: b.N, Projection: tuimaps.PlateCarree}, frames[0].Valid)
		o.Keeps = until.Sub(frames[0].Valid) + time.Hour // current until the horizon has passed
		o.During = tuimaps.Span{From: frames[0].Valid}
		loops = append(loops, o)
	}
	if failed && len(loops) == 0 {
		out.Problems = append(out.Problems, "Radar ahead: Open-Meteo did not answer") // D-124
		return out
	}
	if len(loops) == 0 {
		return out
	}
	out = joinAhead(out, loops, newest, "Open-Meteo")
	out.Note = strings.TrimPrefix(out.Note+" "+modelRainNote, " ")
	return out
}

// joinAhead puts forecast loops beside the observed ones (D-113): the
// observed drawn until their newest frame and the forecast from their first
// (go-tuiMaps L-15.1), so no moment shows both; the forecast fitting what the
// observed leave of the image budget, the farthest dropped first.
func joinAhead(out tty.MapRadar, loops []tuimaps.Overlay, newest time.Time, source string) tty.MapRadar {
	for i := range out.Overlays {
		out.Overlays[i].During = tuimaps.Span{Until: newest}
	}
	out.Overlays = append(out.Overlays, trimForecast(loops, radarImageBudget-chargeOf(out.Overlays))...)
	out.Ahead = source
	return out
}

// rateTable is the table the model's frames are painted with: a colour an
// index, each half a dBZ from 0; the library reads the classes from it and
// draws them in its own radar colours (go-tuiMaps D-45).
var rateTable = func() []tuimaps.TableEntry {
	out := make([]tuimaps.TableEntry, 0, 255)
	for k := 1; k <= 255; k++ {
		out = append(out, tuimaps.TableEntry{Colour: rateColour(k), Value: float64(k-1) / 2})
	}
	return out
}()

// rateColour is index k's colour: distinct for every k, opaque.
func rateColour(k int) tuimaps.RGB {
	return tuimaps.RGB{R: uint8(k), G: uint8(255 - k), B: 128}
}

// ratePNG paints a field of rain rates, mm an hour, in radar's scale: a dry
// or unknown cell transparent - no echo, never radar's lightest class.
func ratePNG(f temperature.Field) []byte {
	pal := color.Palette{color.RGBA{}}
	for k := 1; k <= 255; k++ {
		c := rateColour(k)
		pal = append(pal, color.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
	}
	pic := image.NewPaletted(image.Rect(0, 0, max(f.Cols, 1), max(f.Rows, 1)), pal)
	for i, v := range f.Values {
		dbz := radar.DBZOfRate(v)
		if math.IsNaN(dbz) || dbz < 0 {
			continue
		}
		pic.SetColorIndex(i%f.Cols, i/f.Cols, uint8(math.Round(min(dbz*2, 254))+1)) // held to the palette before the conversion: an infinite dBZ is the heaviest class
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, pic) // a paletted image in memory: it cannot fail
	return buf.Bytes()
}

// withRainDaysWhole adds Forecast mode's rain and snow (W12.3, D-116 to D-118),
// each grid during its step:
//
// KEYLESS FIRST (D-185, D-187, D-192): today and three days on are NDFD's
// daily totals in their own scale (D-184); Now - the hour's rate - and the
// days past NDFD's reach are Open-Meteo's heaviest hour in radar's scale,
// each day's total marked, asked on a quarter of a box's points unless the
// listener chose the full density - it bills every point. Each day Open-Meteo
// answered is recorded; where it does not answer, its days draw as recorded
// (D-168). The badge names what drew, a note says whose the totals are.
// It also says whether Open-Meteo answered for every box.
func withRainDaysWhole(ctx context.Context, t tty.MapTemperature, om *temperature.OpenMeteo, ask tty.MapAsk, now time.Time, rescue *rainRescue) (tty.MapTemperature, bool) {
	anchor := askAnchor(ask, now)
	nowStep, days := forecastDays(anchor)
	failed := false
	live, replayed, totals := 0, 0, 0
	for _, b := range fieldBoxes(ask.Region, ask.View) {
		drawn, got := rescue.ndfdDays(ctx, b.Name, temperature.LatticeFor(b.Name, b.Box), days, anchor, now, ask.Fahrenheit)
		t.Rain, totals = append(t.Rain, got...), totals+len(got)
		lat := rainLattice(b, ask.RainFull)
		r, err := om.Rain(ctx, lat, now, 0)
		if err != nil {
			failed = true
			past, n := rescue.recordedDays(b.Name, lat, days, drawn, anchor, ask.Fahrenheit)
			t.Rain, replayed = append(t.Rain, past...), replayed+n
			continue
		}
		live++
		rescue.record(b.Name, lat, r, anchor, now)
		for i, h := range r.Hours {
			if h.Equal(anchor) {
				if o, ok := rainGrid(tty.RainLayer+"/"+b.Name+"/now", lat, r.Hourly[i], nil, nil, ask.Fahrenheit, anchor); ok {
					o.During = nowStep.Span
					t.Rain = append(t.Rain, o)
				}
			}
		}
		for k, step := range days {
			if drawn[k] {
				continue // NDFD's (D-185)
			}
			if o, ok := rainGrid(tty.RainLayer+"/"+b.Name+"/d"+strconv.Itoa(k), lat, r.Peak[k], r.RainSum[k], r.SnowSum[k], ask.Fahrenheit, anchor); ok {
				o.During = step.Span
				t.Rain = append(t.Rain, o)
			}
		}
	}
	var chips []string // D-133, D-173: what drew, the history last
	if live > 0 {
		chips = append(chips, "O-METEO")
	}
	if totals > 0 {
		chips = append(chips, "NDFD")
		t = withNote(t, tty.RainLayer, "Rain and snow: NDFD's totals for today and three days on - amounts, not the heaviest hour.")
	}
	if replayed > 0 {
		chips = append(chips, recordedChip)
	}
	switch {
	case len(chips) > 0:
		t.Chips = withChips(t.Chips, tty.RainLayer, chips...)
	case failed:
		t.Problems = append(t.Problems, "Rain and snow: Open-Meteo did not answer") // D-124
	}
	return t, !failed
}

// coarseRain is how much of a box's lattice Open-Meteo's rain is asked on by
// default (D-192): a quarter of its points.
const coarseRain = 4

// rainLattice is the lattice Open-Meteo's rain is asked on: a quarter of the
// box's points, or all of them where the listener chose the full density.
func rainLattice(b fieldBox, full bool) temperature.Lattice {
	if full {
		return temperature.LatticeFor(b.Name, b.Box)
	}
	return temperature.LatticeOf(b.Name, b.Box, temperature.MaxPoints/coarseRain)
}

// rainRescue is Forecast mode's rain and snow beside Open-Meteo's: NDFD's
// totals first (D-185), and the days recorded while Open-Meteo answered,
// drawn where it does not (D-168).
type rainRescue struct {
	ndfd  *temperature.NDFD
	store *history.Store
}

// dayStart is the start of the anchor's day k, in its zone: a recorded
// day's key.
func dayStart(anchor time.Time, k int) time.Time {
	return time.Date(anchor.Year(), anchor.Month(), anchor.Day()+k, 0, 0, 0, 0, anchor.Location())
}

// record keeps each day Open-Meteo answered for a box: its heaviest hour,
// its rain and its snow, issued this hour - asking again rewrites nothing.
func (rr *rainRescue) record(box string, lat temperature.Lattice, r temperature.Rain, anchor, now time.Time) {
	if rr == nil || rr.store == nil {
		return
	}
	shape := history.Shape{Box: lat.Box, Cols: lat.Cols, Rows: lat.Rows}
	for k := range temperature.Days { // a week (P10-02)
		if allMissing(r.Peak[k]) {
			continue
		}
		rr.store.Put(omRainDays.Name, history.Record{Key: history.Key{Source: "openmeteo", Place: box}, At: dayStart(anchor, k), IssuedAt: now.Truncate(time.Hour),
			Shape: shape, Values: map[string][]float64{"peak": r.Peak[k], "rain": r.RainSum[k], "snow": r.SnowSum[k]}})
	}
}

// ndfdDays are NDFD's totals for a box's days it reaches - today and three
// days on - and which they are; none without NDFD (D-185).
func (rr *rainRescue) ndfdDays(ctx context.Context, box string, lat temperature.Lattice, days []tty.ForecastStep, anchor, now time.Time, imperial bool) (map[int]bool, []tuimaps.Overlay) {
	drawn := map[int]bool{}
	if rr == nil || rr.ndfd == nil {
		return drawn, nil
	}
	totals, err := rr.ndfd.Totals(ctx, lat, now)
	if err != nil {
		return drawn, nil // Open-Meteo's, then the history's, draw them
	}
	var out []tuimaps.Overlay
	for k, step := range days { // a week (P10-02)
		if o, ok := totalsGrid(tty.RainLayer+"/"+box+"/totals/d"+strconv.Itoa(k), lat, totals.QPF[k], totals.Snow[k], imperial, anchor); ok {
			o.During = step.Span
			out, drawn[k] = append(out, o), true
		}
	}
	return drawn, out
}

// recordedDays are a refused box's days NDFD did not draw, as the history
// holds them - drawn as they were - and how many.
func (rr *rainRescue) recordedDays(box string, lat temperature.Lattice, days []tty.ForecastStep, drawn map[int]bool, anchor time.Time, imperial bool) ([]tuimaps.Overlay, int) {
	if rr == nil || rr.store == nil {
		return nil, 0
	}
	shape := shapeOf(lat)
	var out []tuimaps.Overlay
	for k, step := range days { // a week (P10-02)
		if drawn[k] {
			continue
		}
		rec, ok := rr.recorded(box, dayStart(anchor, k))
		if !ok || rec.Shape != shape {
			continue
		}
		if o, ok := rainGrid(tty.RainLayer+"/"+box+"/d"+strconv.Itoa(k), lat, rec.Values["peak"], rec.Values["rain"], rec.Values["snow"], imperial, anchor); ok {
			o.During = step.Span
			out = append(out, o)
		}
	}
	return out, len(out)
}

// recorded is a box's day as the history holds it.
func (rr *rainRescue) recorded(box string, day time.Time) (history.Record, bool) {
	if rr.store == nil {
		return history.Record{}, false
	}
	return rr.store.Get(omRainDays.Name, history.Key{Source: "openmeteo", Place: box}, day)
}

// totalsGrid is a day's rain, liquid-equivalent, in mm, in the totals' own
// scale (D-184), each cell's total marked on it - snow apart - as Open-
// Meteo's days are; false where NDFD has none.
func totalsGrid(id string, l temperature.Lattice, qpfMM, snowCM []float64, imperial bool, anchor time.Time) (tuimaps.Overlay, bool) {
	f := l.Interpolate(qpfMM)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	sf := l.Interpolate(snowCM)
	g := tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N, Cols: f.Cols, Rows: f.Rows, Values: f.Values, Marks: make([]string, len(f.Values))}
	for i := range g.Marks {
		g.Marks[i] = totalMark(f.Values[i], sf.Values[i], imperial)
	}
	o := tuimaps.QPFGrid(id, g, anchor)
	o.Keeps, o.Credit = 3*time.Hour, temperature.NDFDTotalsCredit // kept as the rain's days are
	return o, true
}

// rainGrid is a lattice's rain rates, mm an hour, as a grid in radar's
// scale, each day's totals marked on it when given; false when no point has
// any. Its currency counts from the hour's start (U2-13).
func rainGrid(id string, l temperature.Lattice, rates, rainMM, snowCM []float64, imperial bool, anchor time.Time) (tuimaps.Overlay, bool) {
	f := l.Interpolate(rates)
	if allMissing(f.Values) {
		return tuimaps.Overlay{}, false
	}
	for i, v := range f.Values {
		f.Values[i] = radar.DBZOfRate(v)
	}
	g := tuimaps.Grid{West: f.Box.W, South: f.Box.S, East: f.Box.E, North: f.Box.N, Cols: f.Cols, Rows: f.Rows, Values: f.Values,
		Type: tuimaps.Type{Preset: "radar", Unit: "dBZ"}} // drawn as rain is (go-tuiMaps L-17.1)
	if rainMM != nil {
		rf, sf := l.Interpolate(rainMM), l.Interpolate(snowCM)
		g.Marks = make([]string, len(f.Values))
		for i := range g.Marks {
			g.Marks[i] = totalMark(rf.Values[i], sf.Values[i], imperial)
		}
	}
	return tuimaps.Overlay{ID: id, Valid: anchor, Keeps: 3 * time.Hour, Grid: &g}, true
}

// The least a total marks (D-116): a hundredth of an inch of rain, or a
// quarter of a mm; a tenth of an inch of snow, or a quarter of a cm. Less is
// a trace, and says nothing.
const (
	traceRainIn, traceRainMM = 0.01, 0.25
	traceSnowIn, traceSnowCM = 0.1, 0.25
)

// totalMark is a day's total as the map marks it (D-116): snow, marked
// apart with a * before it, where the day has any, else rain; in inches, or
// mm of rain and cm of snow. At most seven cells.
func totalMark(rainMM, snowCM float64, imperial bool) string {
	number := func(v float64, unit string) string {
		s := strconv.FormatFloat(v, 'f', 1, 64)
		switch {
		case v >= 9.95:
			s = strconv.FormatFloat(v, 'f', 0, 64)
		case imperial && unit == "in" && v < 0.995:
			s = strings.TrimPrefix(strconv.FormatFloat(v, 'f', 2, 64), "0")
		}
		return s + unit
	}
	if imperial {
		if in := snowCM / 2.54; in >= traceSnowIn {
			return "*" + number(in, "in")
		}
		if in := units.InchesOf(rainMM); in >= traceRainIn {
			return number(in, "in")
		}
		return ""
	}
	if snowCM >= traceSnowCM {
		return "*" + number(snowCM, "cm")
	}
	if rainMM >= traceRainMM {
		return number(rainMM, "mm")
	}
	return ""
}

// rainDaysBytes is a box's week of rain on the wire: 28 KB for 6 points
// measured, 4.7 KB a point, so about 375 KB for 80 (2026-09-27).
const rainDaysBytes = 375_000

// rainLayerCost is what Forecast mode's rain would fetch in a refresh: a
// week a field box. In Radar mode the row is not drawn and costs nothing;
// the hours ahead are the radar's own cost.
func rainLayerCost(in mapInputs) (int64, int) {
	if in.region == "" || !in.forecast {
		return 0, 0
	}
	boxes := len(fieldBoxes(in.region, in.view))
	return int64(boxes) * rainDaysBytes, boxes
}
