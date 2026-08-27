package hms

// Quality pass Q3 (PF-7, CQ-11, CQ-12): the streaming hand-decoded parser
// must read exactly what the struct-decoding one read — the reference
// implementation lives here, in the test — and every refusal path must
// still refuse; the satellite/method strings are shared; and Parse never
// writes into the body it was handed (httpx.GetText's read-only contract).

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// legacyParseKML is the pre-Q3 parser (DecodeElement per placemark, a map
// per description), kept verbatim as the equivalence reference.
func legacyParseKML(b []byte) ([]Point, error) {
	type placemark struct {
		Description string `xml:"description"`
		Coordinates string `xml:"Point>coordinates"`
	}
	dec := xml.NewDecoder(bytes.NewReader(b))
	var out []Point
	root := ""
	placemarks := 0
	for tokens := 0; tokens <= maxTokens; tokens++ {
		tok, err := dec.Token()
		if err == io.EOF {
			if root == "" {
				return nil, errors.New("kml: empty document")
			}
			return out, nil
		}
		if err != nil {
			return out, fmt.Errorf("kml: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if root == "" {
			root = se.Name.Local
			if root != "kml" {
				return nil, fmt.Errorf("kml: document root is <%s>, not <kml> (an outage page in place of data?)", tagName(root))
			}
		}
		if se.Name.Local != "Placemark" {
			continue
		}
		if placemarks == maxPlacemarks {
			return out, ErrTruncated
		}
		placemarks++
		var pm placemark
		if err := dec.DecodeElement(&pm, &se); err != nil {
			continue
		}
		if pt, ok := legacyParseDescription(pm.Description, pm.Coordinates); ok {
			out = append(out, pt)
		}
	}
	return out, ErrTruncated
}

func legacyParseDescription(desc, coords string) (Point, bool) {
	fields := map[string]string{}
	for _, part := range strings.Split(desc, "<br>") {
		if k, v, ok := strings.Cut(part, ":"); ok {
			fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	var pt Point
	var err error
	lonS, latS := fields["Lon"], fields["Lat"]
	if c := strings.Split(strings.TrimSpace(coords), ","); len(c) >= 2 && (lonS == "" || latS == "") {
		lonS, latS = c[0], c[1]
	}
	if pt.Lon, err = strconv.ParseFloat(lonS, 64); err != nil {
		return pt, false
	}
	if pt.Lat, err = strconv.ParseFloat(latS, 64); err != nil {
		return pt, false
	}
	yd, hhmm := fields["YearDay"], strings.TrimSuffix(fields["Time"], "UTC")
	if len(yd) == 7 && len(hhmm) == 4 {
		year, _ := strconv.Atoi(yd[:4])
		day, _ := strconv.Atoi(yd[4:])
		hh, _ := strconv.Atoi(hhmm[:2])
		mm, _ := strconv.Atoi(hhmm[2:])
		pt.At = time.Date(year, 1, 1, hh, mm, 0, 0, time.UTC).AddDate(0, 0, day-1)
	}
	pt.Satellite, pt.Method = fields["Satellite"], fields["Method"]
	if v, err := strconv.ParseFloat(strings.TrimSuffix(fields["FRP"], "MW"), 64); err == nil && v >= 0 {
		pt.FRPMW = &v
	}
	return pt, true
}

// kmlOf wraps placemark markup in a KML document.
func kmlOf(body string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><kml xmlns="http://www.opengis.net/kml/2.2"><Document>` + body + `</Document></kml>`)
}

// mergedKML is the merged file inside a synthetic archive.
func mergedKML(t *testing.T, raw []byte) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "kmls/hms_fire") {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			return b
		}
	}
	t.Fatal("no merged file")
	return nil
}

func TestStreamingParserMatchesTheReference(t *testing.T) {
	rows := []struct {
		name string
		doc  []byte
	}{
		{"the live-probed fixture", []byte(kml)},
		{"2,000 synthetic placemarks", mergedKML(t, syntheticKMZ(2000))},
		{"escaped (not CDATA) description", kmlOf(`<Placemark><description>Lon: -117.24&lt;br&gt;Lat: 33.29&lt;br&gt;YearDay: 2026237&lt;br&gt;Time: 0201UTC&lt;br&gt;Satellite: GOES-WEST&lt;br&gt;Method: NGFS&lt;br&gt;FRP: 1.5MW</description><Point><coordinates>-117.24,33.29</coordinates></Point></Placemark>`)},
		{"coordinates fill missing Lon/Lat", kmlOf(`<Placemark><description><![CDATA[Satellite: MODIS<br>FRP: 3MW]]></description><Point><coordinates>-120.5,40.25,0</coordinates></Point></Placemark>`)},
		{"repeated key: last wins", kmlOf(`<Placemark><description><![CDATA[Lon: 1<br>Lat: 2<br>Lon: 3<br>Satellite: A<br>Satellite: B]]></description></Placemark>`)},
		{"no description at all", kmlOf(`<Placemark><name>x</name><Point><coordinates>-1,2</coordinates></Point></Placemark>`)},
		{"negative FRP dropped, bad time ignored", kmlOf(`<Placemark><description><![CDATA[Lon: -1<br>Lat: 2<br>YearDay: 20262<br>Time: 12UTC<br>FRP: -5MW]]></description></Placemark>`)},
		{"empty placemark", kmlOf(`<Placemark></Placemark><Placemark><description><![CDATA[Lon: 5<br>Lat: 6]]></description></Placemark>`)},
		{"no placemarks", kmlOf(``)},
	}
	for _, row := range rows {
		want, wantErr := legacyParseKML(row.doc)
		got, gotErr := parseKML(row.doc)
		if !reflect.DeepEqual(got, want) || (gotErr == nil) != (wantErr == nil) {
			t.Errorf("%s: streaming parse differs from the reference\n got: %+v (%v)\nwant: %+v (%v)", row.name, got, gotErr, want, wantErr)
		}
	}
}

func TestStreamingParserRefusalPathsStillRefuse(t *testing.T) {
	if _, err := parseKML([]byte(`<html><body>maintenance</body></html>`)); err == nil || !strings.Contains(err.Error(), "<html>") {
		t.Fatalf("an outage page must not read as no fires: %v", err)
	}
	if _, err := parseKML([]byte(``)); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("an empty document is an error: %v", err)
	}
	if _, err := parseKML([]byte(`   `)); err == nil {
		t.Fatal("whitespace only is an error")
	}
	pts, err := parseKML(kmlOf(`<Placemark><description><![CDATA[Lon: 1<br>Lat: 2]]></description></Placemark><Placemark><description>`)) // torn: <description> never closes
	if err == nil || len(pts) != 1 {
		t.Fatalf("a torn document returns what was read AND the syntax error (so Fetch forgets the cached body): %d points, %v", len(pts), err)
	}
	if _, err := parseKML([]byte(`<kml><Document><Placemark><description>Lon: 1<br>Lat: 2</description>`)); err == nil {
		t.Fatal("EOF mid-document is an error")
	}
	// The inflate budget: an entry whose inflated size passes 96 MB is
	// refused whatever the parser made of its tail (CQ-11). Whitespace
	// compresses to nothing, so the archive itself is small.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("kmls/hms_fire20260826.kml")
	_, _ = w.Write([]byte(`<kml><Document>`))
	pad := bytes.Repeat([]byte(" "), 1<<20)
	for range (maxInflateByte >> 20) + 2 {
		_, _ = w.Write(pad)
	}
	_, _ = w.Write([]byte(`</Document></kml>`))
	_ = zw.Close()
	if _, err := Parse(buf.Bytes()); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("over the inflate budget must be refused, got %v", err)
	}
}

func TestSatelliteAndMethodStringsAreShared(t *testing.T) {
	in := newInterner()
	a, b := in.intern("GOES-EAST"), in.intern(strings.Clone("GOES-EAST"))
	if a != b || len(in.seen) != 1 {
		t.Fatal("equal values intern to one string")
	}
	for i := range maxInterned + 10 {
		in.intern(fmt.Sprintf("v%d", i))
	}
	if len(in.seen) != maxInterned {
		t.Fatalf("the table is bounded at %d, got %d", maxInterned, len(in.seen))
	}
	if got := in.intern("past-the-cap"); got != "past-the-cap" {
		t.Fatal("past the cap values pass through")
	}
	// The parse's allocation count is the number that proves the sharing
	// and the no-map, no-DecodeElement walk. What remains is the XML
	// decoder's own element-name strings (eight per placemark), so the pin
	// is relative to the reference parser in this file, not absolute.
	doc := mergedKML(t, syntheticKMZ(1000))
	per := testing.AllocsPerRun(5, func() { _, _ = parseKML(doc) }) / 1000
	ref := testing.AllocsPerRun(5, func() { _, _ = legacyParseKML(doc) }) / 1000
	if per > 0.7*ref {
		t.Fatalf("allocations per placemark: %.1f vs the reference %.1f — the Q3 parser must stay under 70 %% of it", per, ref)
	}
	t.Logf("allocations per placemark: %.1f (reference %.1f)", per, ref)
}

func TestGetTextCallersMustNotMutate(t *testing.T) {
	for name, raw := range map[string][]byte{"kmz": syntheticKMZ(200), "kml": []byte(kml)} {
		before := sha256.Sum256(raw)
		if _, err := Parse(raw); err != nil {
			t.Fatal(err)
		}
		if sha256.Sum256(raw) != before {
			t.Fatalf("%s: Parse wrote into the body it was handed — the cache's own slice (httpx.GetText contract)", name)
		}
	}
}
