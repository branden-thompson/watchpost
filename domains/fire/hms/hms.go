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
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
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

	mu      sync.Mutex
	memoSum [sha256.Size]byte
	memoPts []Point
	memoErr error
}

// MemoPoints reports how many parsed points the archive memo holds (the
// diagnostic dump's view of the memo; one archive at a time by design).
func (p *Provider) MemoPoints() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.memoPts)
}

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
func (p *Provider) parsed(raw []byte) ([]Point, error) {
	sum := sha256.Sum256(raw)
	p.mu.Lock()
	defer p.mu.Unlock()
	if sum == p.memoSum && (p.memoPts != nil || p.memoErr != nil) {
		return p.memoPts, p.memoErr
	}
	pts, err := Parse(raw)
	p.memoSum, p.memoPts, p.memoErr = sum, pts, err
	return pts, err
}

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
		b, err := io.ReadAll(io.LimitReader(rc, budget+1))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		if int64(len(b)) > budget {
			return nil, errors.New("archive inflates past the 96 MB budget — refused")
		}
		budget -= int64(len(b))
		pts, err := parseKML(b)
		out = append(out, pts...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

type placemark struct {
	Description string `xml:"description"`
	Coordinates string `xml:"Point>coordinates"`
}

// parseKML streams the placemarks; a placemark that does not decode or a
// description that does not parse is skipped (one odd point must not lose
// the continent — red-team B5 F7). The document must be KML: an HTML
// outage page must not read as "no fires" (F6). The cap counts placemarks,
// not tokens (F2), and reaching it returns ErrTruncated with what was read.
func parseKML(b []byte) ([]Point, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	var out []Point
	root := ""
	placemarks := 0
	for tokens := 0; tokens <= maxTokens; tokens++ { // P10-02: bounded — a placemark is a handful of tokens, so the cap trips first
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
			continue // one malformed placemark, not the continent
		}
		if pt, ok := parseDescription(pm.Description, pm.Coordinates); ok {
			out = append(out, pt)
		}
	}
	return out, ErrTruncated
}

// maxTokens bounds the XML token walk (P10-02): placemarks are ~2–8
// tokens each, so maxPlacemarks trips long before this does.
const maxTokens = maxPlacemarks * 32

// parseDescription reads "Lon: -121.55<br>Lat: 49.89<br>YearDay: 2026237<br>Time: 0201UTC<br>Satellite: GOES-EAST<br>Method: NGFS<br>Ecosystem: 22<br>FRP: 10.980MW".
func parseDescription(desc, coords string) (Point, bool) {
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
