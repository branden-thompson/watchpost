package hms

// HMS parse benchmarks (quality pass Q0, from DISCOVER lens L1): the
// archive parse is the largest single allocation event in the process
// (~85 MB every 10 minutes at the live size). Q3 streams the zip entry
// into the decoder and interns Satellite/Method; these numbers are the
// before, `make quality-bench` records the after.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// syntheticKMZ builds an archive the size of the live one (27.5k placemarks
// across CONUS with the six satellites the feed carries).
func syntheticKMZ(n int) []byte {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><kml xmlns="http://www.opengis.net/kml/2.2"><Document>`)
	sats := []string{"GOES-WEST", "GOES-EAST", "NOAA-20", "NOAA-21", "SUOMI NPP", "MODIS"}
	for i := range n {
		lat := 25 + float64(i%2500)*0.01
		lon := -125 + float64(i%5000)*0.01
		fmt.Fprintf(&sb, `<Placemark><description><![CDATA[Lon: %.6f<br>Lat: %.6f<br>YearDay: 2026237<br>Time: %02d%02dUTC<br>Satellite: %s<br>Method: NGFS<br>Ecosystem: 22<br>FRP: %.3fMW]]></description><Point><coordinates>%.6f,%.6f</coordinates></Point></Placemark>`+"\n",
			lon, lat, i%24, i%60, sats[i%len(sats)], float64(i%400), lon, lat)
	}
	sb.WriteString(`</Document></kml>`)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("kmls/hms_fire20260825.kml")
	_, _ = w.Write([]byte(sb.String()))
	_ = zw.Close()
	return buf.Bytes()
}

func BenchmarkParse27k(b *testing.B) {
	raw := syntheticKMZ(27_500)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pts, err := Parse(raw)
		if err != nil || len(pts) == 0 {
			b.Fatal(err)
		}
	}
}
