package radar

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The two sources' hosts, FR-3.8's closed list.
const (
	iemBase  = "https://mesonet.agron.iastate.edu"
	mrmsBase = "https://opengeo.ncep.noaa.gov"
)

// Hosts are the radar sources' addresses, FR-3.8's closed list: the app
// names them in the Status window and a test holds them to the table.
func Hosts() map[string]string {
	return map[string]string{"Iowa Environmental Mesonet": iemBase, "NOAA / NCEP (MRMS)": mrmsBase}
}

// window is how far back a source is asked for times: two hours, the loop's
// length (W8.3b) and radar's retention (FR-3.9).
const window = 2 * time.Hour

// IEM is the Iowa Environmental Mesonet's N0Q composite: the lower 48 only,
// on a five-minute grid, with its colour table published (wave 1).
type IEM struct {
	get  Getter
	base string
	now  func() time.Time
}

// NewIEM builds the source; base "" is the production host.
func NewIEM(get Getter, base string) *IEM {
	if base == "" {
		base = iemBase
	}
	return &IEM{get: get, base: base, now: time.Now}
}

func (s *IEM) Name() string { return "IEM" }

func (s *IEM) Covers(region string) bool { return region == geo.RegionContiguous }

// Times reads IEM's list of composite scans over the last two hours.
func (s *IEM) Times(ctx context.Context, region string) ([]time.Time, error) {
	if !s.Covers(region) {
		return nil, ErrNotCovered
	}
	end := s.now().UTC()
	q := url.Values{"operation": {"list"}, "product": {"N0Q"}, "radar": {"USCOMP"},
		"start": {end.Add(-window).Format("2006-01-02T15:04Z")}, "end": {end.Format("2006-01-02T15:04Z")}}
	body, err := s.get.GetText(ctx, s.base+"/json/radar.py?"+q.Encode(), httpx.TTL(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("IEM radar times: %w", err)
	}
	var list struct {
		Scans []struct {
			TS string `json:"ts"`
		} `json:"scans"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("IEM radar times: %w", err)
	}
	var out []time.Time
	for _, sc := range list.Scans {
		if t, err := time.Parse("2006-01-02T15:04Z", sc.TS); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// Frame is IEM's timed WMS picture of a box, at an advertised time only.
func (s *IEM) Frame(ctx context.Context, region string, at time.Time, advertised []time.Time, b Box) ([]byte, error) {
	if !s.Covers(region) {
		return nil, ErrNotCovered
	}
	if err := advertisedOrErr(at, advertised); err != nil {
		return nil, err
	}
	q := wmsQuery(b, "LAYERS", "nexrad-n0q-wmst")
	q.Set("TIME", at.UTC().Format("2006-01-02T15:04:00Z"))
	return s.get.GetText(ctx, s.base+"/cgi-bin/wms/nexrad/n0q-t.cgi?"+q.Encode(), httpx.TTL(window))
}

// MRMS is NOAA/NCEP's Multi-Radar Multi-Sensor reflectivity: the lower 48,
// Alaska, Hawaii, the Caribbean and Guam, every two minutes or so, kept about
// two hours; its colours are the library's approximate table (D-83).
type MRMS struct {
	get  Getter
	base string
}

// NewMRMS builds the source; base "" is the production host.
func NewMRMS(get Getter, base string) *MRMS {
	if base == "" {
		base = mrmsBase
	}
	return &MRMS{get: get, base: base}
}

func (s *MRMS) Name() string { return "MRMS" }

// mrmsWorkspaces are MRMS's services by map region; American Samoa has none.
var mrmsWorkspaces = map[string]string{
	geo.RegionContiguous: "conus", geo.RegionAlaska: "alaska", geo.RegionHawaii: "hawaii",
	geo.RegionCaribbean: "carib", geo.RegionMarianas: "guam",
}

func (s *MRMS) Covers(region string) bool { _, ok := mrmsWorkspaces[region]; return ok }

// layer is a region's workspace and reflectivity layer.
func (s *MRMS) layer(region string) (ws, name string, ok bool) {
	ws, ok = mrmsWorkspaces[region]
	return ws, ws + "_bref_qcd", ok
}

// Times reads the layer's time dimension from its capabilities.
func (s *MRMS) Times(ctx context.Context, region string) ([]time.Time, error) {
	ws, name, ok := s.layer(region)
	if !ok {
		return nil, ErrNotCovered
	}
	body, err := s.get.GetText(ctx, s.base+"/geoserver/"+ws+"/"+name+"/ows?service=wms&version=1.3.0&request=GetCapabilities", httpx.TTL(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("MRMS radar times: %w", err)
	}
	return mrmsTimes(body)
}

// mrmsTimes is the time dimension of a capabilities document, oldest first.
func mrmsTimes(body []byte) ([]time.Time, error) {
	var caps struct {
		Dims []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:",chardata"`
		} `xml:"Capability>Layer>Layer>Dimension"`
	}
	if err := xml.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("MRMS radar times: %w", err)
	}
	var out []time.Time
	for _, d := range caps.Dims {
		if d.Name != "time" {
			continue
		}
		for _, v := range strings.Split(strings.TrimSpace(d.Value), ",") {
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(v)); err == nil {
				out = append(out, t)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("MRMS radar times: the capabilities list no time")
	}
	return out, nil
}

// Frame is MRMS's WMS picture of a box, at an advertised time only.
func (s *MRMS) Frame(ctx context.Context, region string, at time.Time, advertised []time.Time, b Box) ([]byte, error) {
	ws, name, ok := s.layer(region)
	if !ok {
		return nil, ErrNotCovered
	}
	if err := advertisedOrErr(at, advertised); err != nil {
		return nil, err
	}
	q := wmsQuery(b, "layers", name)
	q.Set("time", at.UTC().Format("2006-01-02T15:04:05.000Z"))
	return s.get.GetText(ctx, s.base+"/geoserver/"+ws+"/"+name+"/ows?"+q.Encode(), httpx.TTL(window))
}

// wmsQuery is a WMS 1.1.1 GetMap of a box in plain degrees, transparent PNG.
func wmsQuery(b Box, layersKey, layer string) url.Values {
	return url.Values{"SERVICE": {"WMS"}, "REQUEST": {"GetMap"}, "VERSION": {"1.1.1"}, layersKey: {layer}, "STYLES": {""},
		"SRS": {"EPSG:4326"}, "FORMAT": {"image/png"}, "TRANSPARENT": {"true"},
		"BBOX":  {strings.Join([]string{ftoa(b.W), ftoa(b.S), ftoa(b.E), ftoa(b.N)}, ",")},
		"WIDTH": {strconv.Itoa(b.Cols)}, "HEIGHT": {strconv.Itoa(b.Rows)}}
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
