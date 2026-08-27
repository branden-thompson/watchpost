// Package hms reads NOAA-NESDIS Hazard Mapping System fire detections (B5,
// live-probed 2026-08-25; keyless): one KMZ for the whole continent,
// refreshed every 10 minutes, holding a merged `hms_fire<date>.kml` with
// analyst-curated points from GOES-East/West, NOAA-20/21, Suomi NPP and
// MODIS — ~25k placemarks, each with Lon/Lat/YearDay/Time/Satellite/Method/
// Ecosystem/FRP in its description. The provider fetches the archive once
// per cycle (the client cache holds it for 10 minutes) and answers every
// location from it, so the watchlist costs one download, not sixty.
package hms

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Attribution is the credit line (public domain).
const Attribution = "NOAA HMS wildfire detections (ospo.noaa.gov)"

// archiveTTL matches HMS's own refresh interval.
const archiveTTL = 10 * time.Minute

// maxPlacemarks bounds the parse (P10-02); the archive holds tens of
// thousands (27.5k live 2026-08-25; peak days have exceeded 100k). Hitting
// it is reported, never silent (red-team B5 F2).
const maxPlacemarks = 200000

// Archive budgets (red-team B5 F5): the KMZ inflates under one shared
// byte budget across at most maxFiles entries — a hostile archive cannot
// buy gigabytes of CPU with a 32 MB body.
const (
	maxFiles       = 16
	maxInflateByte = 96 << 20
)

// ErrTruncated reports a parse that hit maxPlacemarks: the tail of the
// archive is missing, and the fragment says so.
var ErrTruncated = errors.New("hms: archive holds more than 200000 placemarks — the tail was not read")

// Provider is the HMS snapshot provider. The parsed archive is memoized by
// content hash (red-team B5 P1): every RECENT location runs its own
// scheduler, so without the memo one 15-minute tick parsed the same 1.4 MB
// KMZ fifty times (measured 4.5 GB allocated, 616 MB heap peak); with it
// the archive is parsed once per change, ~120 ms and ~90 MB, whoever asks.
type Provider struct {
	client *httpx.Client
	url    string
	rules  fire.Rules
	memo   fire.Memo[[]Point] // the last archive's parse (Q3: the shared fire.Memo)
}

// MemoPoints reports how many parsed points the archive memo holds (the
// diagnostic dump's view of the memo; one archive at a time by design).
func (p *Provider) MemoPoints() int {
	pts, _ := p.memo.Peek()
	return len(pts)
}

// MemoStats is the memo's size and its parse count since launch — the
// diagnostic dump's parse-spike counter (plan §1: parse spikes reported
// per event).
func (p *Provider) MemoStats() (points, parses int) { return p.MemoPoints(), p.memo.Parses() }

// New builds the provider; url "" means the production archive.
func New(client *httpx.Client, url string, rules fire.Rules) *Provider {
	if url == "" {
		url = "https://www.ospo.noaa.gov/data/spl/kmlfiles/fire/fireAllSats.kmz"
	}
	return &Provider{client: client, url: url, rules: rules}
}

// ID implements snapshot.Provider.
func (p *Provider) ID() string { return "hms" }

// Domains implements snapshot.Provider.
func (p *Provider) Domains() []string { return []string{"fire"} }

// Fetch implements snapshot.Provider for KindFire: every requested location
// gets a FireState — empty when nothing burns nearby (that is data, not a
// failure, so the scheduler never retries it).
func (p *Provider) Fetch(ctx context.Context, req snapshot.FetchReq) (snapshot.Fragment, error) {
	frag := snapshot.Fragment{Provider: p.ID(), Kind: req.Kind, FetchedAt: time.Now().UTC(), PerLocation: map[snapshot.LocationKey]snapshot.PartialData{}}
	if err := invariant.Check(req.Kind == snapshot.KindFire, "hms serves only KindFire"); err != nil {
		return frag, err
	}
	if err := p.rules.Valid(); err != nil {
		return frag, err
	}
	raw, err := p.client.GetText(ctx, p.url, httpx.TTL(archiveTTL))
	if err != nil {
		frag.Err = fmt.Errorf("hms: %w", err)
		return frag, nil
	}
	points, err := p.parsed(raw)
	if err != nil && !errors.Is(err, ErrTruncated) {
		p.client.Forget(p.url) // a cached body that does not parse must not be served for the rest of its TTL (P6)
		frag.Err = fmt.Errorf("hms: %w", err)
		return frag, nil
	}
	if err != nil {
		frag.Err = err // truncated: what was read is published, the warning names the loss
	}
	for _, ref := range req.Locations {
		var hs []snapshot.Hotspot
		for _, pt := range points {
			km, ok := fire.Near(ref, pt.Lat, pt.Lon, p.rules.RadiusKm)
			if !ok || !p.rules.Keep("analyst", pt.FRPMW) {
				continue
			}
			d := km
			hs = append(hs, snapshot.Hotspot{Lat: pt.Lat, Lon: pt.Lon, DetectedAt: pt.At, Confidence: "analyst", FRPMW: pt.FRPMW, DistanceKm: &d,
				Source: snapshot.SourceInfo{Provider: p.ID(), ModelOrStation: pt.Satellite, IssuedAt: pt.At}})
		}
		frag.PerLocation[snapshot.Key(ref)] = snapshot.PartialData{Fire: &snapshot.FireState{AsOf: frag.FetchedAt, Hotspots: fire.Cluster(hs)}}
	}
	return frag, nil
}

// parsed returns the archive's points, parsing only when the bytes changed.
func (p *Provider) parsed(raw []byte) ([]Point, error) { return p.memo.Get(raw, Parse) }

// Point is one HMS detection.
type Point struct {
	Lat, Lon  float64
	At        time.Time
	Satellite string
	Method    string
	FRPMW     *float64
}

// Parse reads the KMZ (or a bare KML) and returns every detection in the
// merged hms_fire file — or, when the archive has no merged file, every
// per-satellite file. A parse that hits maxPlacemarks returns what it read
// with ErrTruncated.
func Parse(raw []byte) ([]Point, error) {
	if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("<")) {
		return parseKML(raw)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("archive is not a KMZ: %w", err)
	}
	var files []*zip.File
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "kmls/hms_fire") && strings.HasSuffix(f.Name, ".kml") {
			files = []*zip.File{f} // the merged file is the whole answer
			break
		}
		if strings.HasPrefix(f.Name, "kmls/") && strings.HasSuffix(f.Name, ".kml") && len(files) < maxFiles {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return nil, errors.New("archive holds no detection files")
	}
	var out []Point
	budget := int64(maxInflateByte)
	for _, f := range files {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		// Streamed, not inflated into memory first (Q3, PF-7): the counting
		// reader keeps the refusal exact — an entry past the budget is refused
		// whatever the parser made of its truncated tail (CQ-11).
		var read byteCount
		pts, perr := parseKMLReader(io.TeeReader(io.LimitReader(rc, budget+1), &read))
		_ = rc.Close()
		if int64(read) > budget {
			return nil, errors.New("archive inflates past the 96 MB budget — refused")
		}
		budget -= int64(read)
		out = append(out, pts...)
		if perr != nil {
			return out, perr
		}
	}
	return out, nil
}

// byteCount is a Writer that only counts (the TeeReader's side).
type byteCount int64

func (c *byteCount) Write(p []byte) (int, error) {
	*c += byteCount(len(p))
	return len(p), nil
}

// parseKML parses a KML document held in memory.
func parseKML(b []byte) ([]Point, error) { return parseKMLReader(bytes.NewReader(b)) }

// parseKMLReader streams the placemarks, hand-decoding <Placemark> from
// the raw token walk (Q3, PF-7: no per-placemark struct decode, no map):
// a placemark whose description does not parse is skipped (one odd point
// must not lose the continent — red-team B5 F7). The document must be
// KML: an HTML outage page must not read as "no fires" (F6). The cap
// counts placemarks, not tokens (F2), and reaching it returns ErrTruncated
// with what was read.
func parseKMLReader(r io.Reader) ([]Point, error) {
	w := kmlWalk{dec: xml.NewDecoder(r), in: newInterner(), open: make([]string, 0, 16)}
	for tokens := 0; tokens <= maxTokens; tokens++ { // P10-02: bounded — a placemark is a handful of tokens, so the cap trips first
		tok, err := w.dec.RawToken() // RawToken: no name-space translation (eight allocations a placemark the walk never reads)
		if err == io.EOF {
			return w.out, w.eof()
		}
		if err != nil {
			return w.out, fmt.Errorf("kml: %w", err)
		}
		if err := w.step(tok); err != nil {
			return w.out, err
		}
	}
	return w.out, ErrTruncated
}

// kmlWalk is the state of one token walk. RawToken skips nesting checks,
// so the element stack is kept here: a torn body must still error, so
// Fetch forgets the cached entry (P6).
type kmlWalk struct {
	dec        *xml.Decoder
	in         *interner
	out        []Point
	cur        placemarkText
	open       []string // the element stack
	root       string
	placemarks int
}

// step consumes one token.
func (w *kmlWalk) step(tok xml.Token) error {
	switch t := tok.(type) {
	case xml.StartElement:
		return w.start(t.Name.Local)
	case xml.CharData:
		w.cur.text(t) // the decoder's buffer: consumed now, never retained
	case xml.EndElement:
		return w.end(t.Name.Local)
	}
	return nil
}

func (w *kmlWalk) start(name string) error {
	if w.root == "" {
		w.root = name
		if name != "kml" {
			return fmt.Errorf("kml: document root is <%s>, not <kml> (an outage page in place of data?)", tagName(name))
		}
	}
	w.open = append(w.open, name)
	switch {
	case name == "Placemark":
		if w.placemarks == maxPlacemarks {
			return ErrTruncated
		}
		w.placemarks++
		w.cur = placemarkText{open: true}
	case w.cur.open && (name == "description" || name == "coordinates"):
		w.cur.field = name
	}
	return nil
}

func (w *kmlWalk) end(name string) error {
	if n := len(w.open); n == 0 || w.open[n-1] != name {
		return fmt.Errorf("kml: unexpected </%s>", tagName(name))
	}
	w.open = w.open[:len(w.open)-1]
	switch name {
	case "description", "coordinates":
		w.cur.field = ""
	case "Placemark":
		if pt, ok := parseDescription(w.cur.desc, w.cur.coords, w.in); w.cur.open && ok {
			w.out = append(w.out, pt)
		}
		w.cur = placemarkText{}
	}
	return nil
}

// eof is the end of the document: an error when nothing was read or an
// element is still open.
func (w *kmlWalk) eof() error {
	if w.root == "" {
		return errors.New("kml: empty document")
	}
	if n := len(w.open); n > 0 {
		return fmt.Errorf("kml: unexpected EOF inside <%s>", tagName(w.open[n-1]))
	}
	return nil
}

// placemarkText is the two text fields of the placemark being walked.
type placemarkText struct {
	open         bool
	field        string // "description" | "coordinates" | ""
	desc, coords string
}

func (p *placemarkText) text(b xml.CharData) {
	switch p.field {
	case "description":
		p.desc += string(b)
	case "coordinates":
		p.coords += string(b)
	}
}

// maxTokens bounds the XML token walk (P10-02): placemarks are ~2–8
// tokens each, so maxPlacemarks trips long before this does.
const maxTokens = maxPlacemarks * 32

// maxFields bounds one description's field walk (P10-02): the feed writes eight.
const maxFields = 32

// parseDescription reads "Lon: -121.55<br>Lat: 49.89<br>YearDay: 2026237<br>Time: 0201UTC<br>Satellite: GOES-EAST<br>Method: NGFS<br>Ecosystem: 22<br>FRP: 10.980MW"
// with strings.Cut, field by field, no map (Q3, PF-7); the last value of a
// repeated key wins, as before.
func parseDescription(desc, coords string, in *interner) (Point, bool) {
	f := descFields(desc, in)
	if f.lon == "" || f.lat == "" {
		if c := strings.Split(strings.TrimSpace(coords), ","); len(c) >= 2 {
			f.lon, f.lat = c[0], c[1]
		}
	}
	pt := Point{Satellite: f.satellite, Method: f.method}
	var err error
	if pt.Lon, err = strconv.ParseFloat(f.lon, 64); err != nil {
		return pt, false
	}
	if pt.Lat, err = strconv.ParseFloat(f.lat, 64); err != nil {
		return pt, false
	}
	if len(f.yearDay) == 7 && len(f.hhmm) == 4 {
		year, _ := strconv.Atoi(f.yearDay[:4])
		day, _ := strconv.Atoi(f.yearDay[4:])
		hh, _ := strconv.Atoi(f.hhmm[:2])
		mm, _ := strconv.Atoi(f.hhmm[2:])
		pt.At = time.Date(year, 1, 1, hh, mm, 0, 0, time.UTC).AddDate(0, 0, day-1)
	}
	if v, err := strconv.ParseFloat(f.frp, 64); err == nil && v >= 0 {
		pt.FRPMW = &v
	}
	return pt, true
}

// fields are one description's values as written (trimmed; the unit
// suffixes already stripped).
type fields struct {
	lon, lat, yearDay, hhmm, frp, satellite, method string
}

// descFields walks "Key: value<br>Key: value…" with strings.Cut, field by
// field, no map; the last value of a repeated key wins.
func descFields(desc string, in *interner) fields {
	var f fields
	rest := desc
	for i := 0; i < maxFields && rest != ""; i++ {
		part, tail, _ := strings.Cut(rest, "<br>")
		rest = tail
		k, v, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "Lon":
			f.lon = v
		case "Lat":
			f.lat = v
		case "YearDay":
			f.yearDay = v
		case "Time":
			f.hhmm = strings.TrimSuffix(v, "UTC")
		case "Satellite":
			f.satellite = in.intern(v)
		case "Method":
			f.method = in.intern(v)
		case "FRP":
			f.frp = strings.TrimSuffix(v, "MW")
		}
	}
	return f
}

// interner shares the satellite and method strings across a parse: the
// feed carries six satellites and a handful of methods, so 27k points
// share a dozen strings instead of holding 54k (Q3). Bound: maxInterned
// distinct values; past it, values pass through un-shared.
type interner struct{ seen map[string]string }

const maxInterned = 64

func newInterner() *interner { return &interner{seen: make(map[string]string, 16)} }

func (in *interner) intern(v string) string {
	if s, ok := in.seen[v]; ok {
		return s
	}
	if len(in.seen) < maxInterned {
		in.seen[v] = v
	}
	return v
}

// tagName keeps an element name printable in an error (letters only, short).
func tagName(name string) string {
	out := make([]rune, 0, 16)
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			out = append(out, r)
		}
		if len(out) == 16 {
			break
		}
	}
	return string(out)
}
