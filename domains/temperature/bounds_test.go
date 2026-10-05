package temperature

// bounds_test.go — every measure a source answers is held to its physical
// bound where it is read: a value outside it stays missing, at both sources,
// so a temperature, gust or UV of 1e308 is never drawn or recorded.

import (
	"context"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/units"
)

// TestEachBoundIsItsMeasuresRange pins each measure's physical range: both
// ends kept, a step past either end and NaN and Inf missing.
func TestEachBoundIsItsMeasuresRange(t *testing.T) {
	for _, c := range []struct {
		name   string
		b      bound
		lo, hi float64
	}{
		{"temperature, °C", celsiusBound, -90, 60},
		{"wind and gust, km/h", windBound, 0, 500},
		{"direction, degrees", directionBound, 0, 360},
		{"UV index", uvBound, 0, 20},
		{"rain and snow's water, mm", precipBound, 0, 1000},
		{"snowfall, cm", snowBound, 0, 100},
		{"wave height, m", waveBound, 0, 30},
	} {
		for _, v := range []float64{c.lo, c.hi, (c.lo + c.hi) / 2} {
			if !c.b.holds(v) {
				t.Errorf("%s: %v is in range and is not kept", c.name, v)
			}
		}
		for _, v := range []float64{c.lo - 0.01, c.hi + 0.01, math.NaN(), math.Inf(1), math.Inf(-1), 1e308, -1e308} {
			if c.b.holds(v) {
				t.Errorf("%s: %v is out of range and is kept", c.name, v)
			}
		}
	}
}

// boundCase is a value written into an answer, in the source's units, and
// whether it is kept.
type boundCase struct {
	v    float64
	kept bool
}

// edgeCases are a bound's ends kept and a step past each missing, plus 1e308,
// in the source's units: to is the measure's units into the source's.
func edgeCases(b bound, to func(float64) float64) []boundCase {
	step := (b.hi - b.lo) / 100
	return []boundCase{{to(b.lo + step), true}, {to(b.hi - step), true},
		{to(b.lo - step), false}, {to(b.hi + step), false}, {1e308, false}, {-1e308, false}}
}

func same(v float64) float64 { return v }

// checkRows fails unless the rows hold a value, when it is kept, or hold
// none, when it is not.
func checkRows(t *testing.T, what string, c boundCase, rows [][]float64) {
	t.Helper()
	found := 0
	for _, row := range rows {
		for _, v := range row {
			if !math.IsNaN(v) {
				found++
			}
		}
	}
	if c.kept && found == 0 {
		t.Errorf("%s of %v: nothing kept; it is in range", what, c.v)
	}
	if !c.kept && found > 0 {
		t.Errorf("%s of %v: %d values kept; it is out of range", what, c.v, found)
	}
}

// days is a day-by-day array's rows.
func days(a [Days][]float64) [][]float64 { return a[:] }

// withOpenMeteo is an Open-Meteo answer with every answered value of one
// field, hourly or daily, made v.
func withOpenMeteo(t *testing.T, body []byte, field string, v float64) []byte {
	t.Helper()
	var pts []map[string]any
	if err := json.Unmarshal(body, &pts); err != nil {
		t.Fatal(err)
	}
	set := 0
	for _, p := range pts {
		for _, sect := range []string{"hourly", "daily"} {
			part, _ := p[sect].(map[string]any)
			vals, ok := part[field].([]any)
			if !ok {
				continue
			}
			for i, x := range vals {
				if x != nil {
					vals[i], set = v, set+1
				}
			}
		}
	}
	if set == 0 {
		t.Fatalf("the answer has no %s: this measures nothing", field)
	}
	out, err := json.Marshal(pts)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestOpenMeteosValuesAreHeldToTheirBounds: every measure Open-Meteo answers
// - the forecast's, the rain's and the waves' - is held to its bound.
func TestOpenMeteosValuesAreHeldToTheirBounds(t *testing.T) {
	ctx := context.Background()
	for _, f := range []struct {
		field string
		b     bound
		rows  func(Series) [][]float64
	}{
		{"temperature_2m", celsiusBound, func(s Series) [][]float64 { return s.Hourly }},
		{"apparent_temperature", celsiusBound, func(s Series) [][]float64 { return s.Feels }},
		{"wind_speed_10m", windBound, func(s Series) [][]float64 { return s.WindSpeed }},
		{"wind_direction_10m", directionBound, func(s Series) [][]float64 { return s.WindFrom }},
		{"wind_gusts_10m", windBound, func(s Series) [][]float64 { return s.WindGust }},
		{"uv_index", uvBound, func(s Series) [][]float64 { return s.UV }},
		{"temperature_2m_max", celsiusBound, func(s Series) [][]float64 { return days(s.High) }},
		{"temperature_2m_min", celsiusBound, func(s Series) [][]float64 { return days(s.Low) }},
		{"apparent_temperature_max", celsiusBound, func(s Series) [][]float64 { return days(s.FeelsHigh) }},
		{"apparent_temperature_min", celsiusBound, func(s Series) [][]float64 { return days(s.FeelsLow) }},
		{"wind_speed_10m_max", windBound, func(s Series) [][]float64 { return days(s.PeakSpeed) }},
		{"wind_direction_10m_dominant", directionBound, func(s Series) [][]float64 { return days(s.PeakFrom) }},
		{"wind_gusts_10m_max", windBound, func(s Series) [][]float64 { return days(s.PeakGust) }},
		{"uv_index_max", uvBound, func(s Series) [][]float64 { return days(s.UVMax) }},
	} {
		for _, c := range edgeCases(f.b, same) {
			s := newSeries(fixtureLattice)
			if err := parseOpenMeteo(withOpenMeteo(t, fixture(t, "openmeteo-uv.json"), f.field, c.v), uvCaptured, &s); err != nil {
				t.Fatal(err)
			}
			checkRows(t, "Open-Meteo's "+f.field, c, f.rows(s))
		}
	}
	for _, f := range []struct {
		field string
		b     bound
		rows  func(Rain) [][]float64
	}{
		{"precipitation", precipBound, func(r Rain) [][]float64 { return append(append([][]float64{}, r.Hourly...), days(r.Peak)...) }},
		{"rain_sum", precipBound, func(r Rain) [][]float64 { return days(r.RainSum) }},
		{"showers_sum", precipBound, func(r Rain) [][]float64 { return days(r.RainSum) }},
		{"snowfall_sum", snowBound, func(r Rain) [][]float64 { return days(r.SnowSum) }},
	} {
		for _, c := range edgeCases(f.b, same) {
			r := newRain(rainLattice)
			if err := parseRain(withOpenMeteo(t, fixture(t, "openmeteo-rain-days.json"), f.field, c.v), rainCaptured, &r); err != nil {
				t.Fatal(err)
			}
			checkRows(t, "Open-Meteo's "+f.field, c, f.rows(r))
		}
	}
	for _, field := range []string{"wave_height", "wave_height_max"} {
		for _, c := range edgeCases(waveBound, same) {
			w, err := NewOpenMeteo(&bodyGet{body: withOpenMeteo(t, fixture(t, "openmeteo-waves.json"), field, c.v)}, "").Waves(ctx, fixtureLattice, wavesCaptured)
			if err != nil {
				t.Fatal(err)
			}
			rows := w.Hourly
			if field == "wave_height_max" {
				rows = days(w.Max)
			}
			checkRows(t, "Open-Meteo's "+field, c, rows)
		}
	}
}

// withDWML is an NDFD answer with every value of one element, by its opening
// tag, made v.
func withDWML(t *testing.T, body []byte, open string, v float64) []byte {
	t.Helper()
	tag := strings.Fields(strings.TrimPrefix(open, "<"))[0]
	elem := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(open) + `.*?</` + tag + `>`)
	value := regexp.MustCompile(`<value>[^<]*</value>`)
	set := 0
	out := elem.ReplaceAllFunc(body, func(b []byte) []byte {
		set += len(value.FindAll(b, -1))
		return value.ReplaceAll(b, []byte("<value>"+strconv.FormatFloat(v, 'g', -1, 64)+"</value>"))
	})
	if set == 0 {
		t.Fatalf("the answer has no %s value: this measures nothing", open)
	}
	return out
}

// fahrenheitOf, knotsOf, inchesOf and feetOf are the measure's units in
// NDFD's.
func fahrenheitOf(c float64) float64 { return units.FahrenheitOf(c) }
func knotsOf(kmh float64) float64    { return kmh / units.KmhPerKnot }
func inchesOf(mm float64) float64    { return mm / units.MmPerInch }
func feetOf(m float64) float64       { return m / units.MetresPerFoot }

// TestNDFDsValuesAreHeldToTheirBounds: every measure NDFD answers - the
// temperatures, the wind, the totals and the waves - is held to its bound in
// the measure's own units, after NDFD's are converted.
func TestNDFDsValuesAreHeldToTheirBounds(t *testing.T) {
	ctx := context.Background()
	for _, f := range []struct {
		fixture, open string
		b             bound
		to            func(float64) float64
		rows          func(Series) [][]float64
	}{
		{"ndfd-hour.xml", `<temperature type="hourly"`, celsiusBound, fahrenheitOf, func(s Series) [][]float64 { return s.Hourly }},
		{"ndfd-days.xml", `<temperature type="maximum"`, celsiusBound, fahrenheitOf, func(s Series) [][]float64 { return days(s.High) }},
		{"ndfd-days.xml", `<temperature type="minimum"`, celsiusBound, fahrenheitOf, func(s Series) [][]float64 { return days(s.Low) }},
		{"ndfd-hour-feels.xml", `<temperature type="apparent"`, celsiusBound, fahrenheitOf, func(s Series) [][]float64 { return s.Feels }},
		{"ndfd-days-wind.xml", `<wind-speed type="sustained"`, windBound, knotsOf, func(s Series) [][]float64 { return s.WindSpeed }},
		{"ndfd-days-wind.xml", `<direction type="wind"`, directionBound, same, func(s Series) [][]float64 { return s.WindFrom }},
		{"ndfd-hour-gust.xml", `<wind-speed type="gust"`, windBound, knotsOf, func(s Series) [][]float64 { return s.WindGust }},
	} {
		at := map[string]time.Time{"ndfd-hour.xml": captured, "ndfd-days.xml": captured, "ndfd-hour-feels.xml": feelsCaptured,
			"ndfd-days-wind.xml": windCaptured, "ndfd-hour-gust.xml": gustCaptured}
		for _, c := range edgeCases(f.b, f.to) {
			s := newSeries(fixtureLattice)
			if err := parseDWML(withDWML(t, fixture(t, f.fixture), f.open, c.v), at[f.fixture], &s); err != nil {
				t.Fatal(err)
			}
			checkRows(t, "NDFD's "+f.open+" in "+f.fixture, c, f.rows(s))
		}
	}
	for _, f := range []struct {
		open string
		b    bound
		to   func(float64) float64
		rows func(Totals) [][]float64
	}{
		{`<precipitation type="liquid"`, precipBound, inchesOf, func(o Totals) [][]float64 { return days(o.QPF) }},
		{`<precipitation type="snow"`, snowBound, func(cm float64) float64 { return cm / 2.54 }, func(o Totals) [][]float64 { return days(o.Snow) }},
	} {
		for _, c := range edgeCases(f.b, f.to) {
			o, err := NewNDFD(&bodyGet{body: withDWML(t, fixture(t, "ndfd-totals.xml"), f.open, c.v)}, "").Totals(ctx, totalsLattice, totalsCaptured)
			if err != nil {
				t.Fatal(err)
			}
			checkRows(t, "NDFD's "+f.open, c, f.rows(o))
		}
	}
	for _, c := range edgeCases(waveBound, feetOf) {
		w, err := NewNDFD(&bodyGet{body: withDWML(t, fixture(t, "ndfd-waves.xml"), `<waves type="significant"`, c.v)}, "").Waves(ctx, fixtureLattice, wavesCaptured)
		if err != nil {
			t.Fatal(err)
		}
		checkRows(t, "NDFD's waves", c, append(append([][]float64{}, w.Hourly...), days(w.Max)...))
	}
}
