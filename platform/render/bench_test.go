package render

// Render primitives (quality pass Q0, from DISCOVER lens L5): the table
// body, the width measure, the tint pass, the overlay compositor and the
// SGR builder — the pieces Q3/Q4 move, measured before. Colour forced on.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func benchRows(n int) []LocationRow {
	rows := make([]LocationRow, 0, n)
	for i := range n {
		r := LocationRow{Index: i + 1, Name: fmt.Sprintf("Benchmark City %02d", i), Zip: "92057", Station: "KCRQ", StationKM: fp(12.5),
			Conditions: "partly_cloudy", Now: fp(22.8), Hi: fp(30), Lo: fp(17), Trend: "up",
			TomorrowConditions: "clear", TomorrowHi: fp(31), TomorrowLo: fp(18), Selected: i == 2, HasAlert: i%3 == 0, WarnAlert: i%6 == 0, Fire: i % 4}
		for d := range 5 {
			r.Extended = append(r.Extended, DayCell{Date: fmt.Sprintf("08/%02d", 26+d), Hi: fp(30 + float64(d)), Lo: fp(18)})
		}
		rows = append(rows, r)
	}
	return rows
}

func fp(v float64) *float64 { return &v }

func colourOn(b *testing.B) {
	b.Helper()
	rendering.SetColorEnabledForTest(true)
	b.Cleanup(rendering.ResetColorEnabledForTest)
}

func BenchmarkLocationTable_50rows_w129(b *testing.B) {
	colourOn(b)
	o := Opts{Width: 129}
	rows := benchRows(50)
	b.ReportMetric(float64(len(o.LocationTable(rows, 0))), "bytes")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = o.LocationTable(rows, 0)
	}
}

func BenchmarkLocationTable_50rows_w196_ext(b *testing.B) {
	colourOn(b)
	o := Opts{Width: 196}
	rows := benchRows(50)
	b.ReportAllocs()
	for b.Loop() {
		_ = o.LocationTable(rows, 5)
	}
}

func benchFrameText() string {
	o := Opts{Width: 129}
	return o.LocationTable(benchRows(40), 0) + "\n" + o.KeyCap("enter") + " Details"
}

func BenchmarkTintDefault(b *testing.B) {
	colourOn(b)
	f := benchFrameText()
	b.ReportAllocs()
	for b.Loop() {
		_ = TintDefault(f)
	}
}

func BenchmarkOverlay(b *testing.B) {
	colourOn(b)
	f := benchFrameText()
	o := Opts{Width: 60}
	modal := o.Block(o.Panel("Help", strings.Repeat("line of help text\n", 20)), Tok(ModalFG), Tok(ModalBGDark))
	b.ReportAllocs()
	for b.Loop() {
		_ = Overlay(f, modal, 133)
	}
}

func BenchmarkDisplayWidth_styledRow(b *testing.B) {
	colourOn(b)
	o := Opts{Width: 129}
	row := strings.Split(o.LocationTable(benchRows(3), 0), "\n")[3]
	b.ReportAllocs()
	for b.Loop() {
		_ = displayWidth(row)
	}
}

func BenchmarkKitDisplayWidth_styledRow(b *testing.B) {
	colourOn(b)
	o := Opts{Width: 129}
	row := strings.Split(o.LocationTable(benchRows(3), 0), "\n")[3]
	b.ReportAllocs()
	for b.Loop() {
		_ = rendering.DisplayWidth(row)
	}
}

func BenchmarkWrapSGR(b *testing.B) {
	colourOn(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = rendering.WrapSGR("  72ºF", "208")
	}
}

func BenchmarkNewTextFormatter(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = rendering.NewTextFormatter()
	}
}
