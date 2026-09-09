// Package snapshot is THE data contract (architecture.md §2, §10.1, §10.11).
//
// Providers fetch Fragments; the single Assembler merges them into immutable
// Snapshot values. Renderers (modes/tty, modes/report) read Snapshots and may
// never import a domain — the import-direction lint enforces it. Nil pointers
// mean "provider has no value": they marshal as JSON null and render as n/a
// (the null-parity rule). SI units internally; conversion happens at render.
package snapshot

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"
)

// SchemaVersion is the published JSON contract version (v1.0-rc until B5
// ratification — architecture §10.3).
const SchemaVersion = "1.0.0-rc"

// Snapshot is the single source every renderer consumes. Immutable after
// publication.
type Snapshot struct {
	SchemaVersion string           `json:"schema_version"`
	GeneratedAt   time.Time        `json:"generated_at"`
	Locations     []Location       `json:"locations"` // stable order = config order
	Providers     []ProviderStatus `json:"providers"`
	Warnings      []Warning        `json:"warnings"`
}

// Location is one watched place with all its data domains.
type Location struct {
	Label string  `json:"label"`
	Tag   string  `json:"tag"` // user 5-char short label (mock LABEL column)
	Zip   string  `json:"zip"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	TZ    string  `json:"tz"`

	Harmonized Conditions         `json:"harmonized"`
	ByProvider map[string]Section `json:"by_provider"`
	Alerts     []Alert            `json:"alerts"`
	Fire       FireState          `json:"fire"`
	Radio      RadioState         `json:"radio"`
	Hourly     []Hourly           `json:"hourly"`
	Daily      []Daily            `json:"daily"`
	Marine     *Marine            `json:"marine"`  // null inland (B3 UAT 29)
	Seismic    *SeismicState      `json:"seismic"` // nil until the USGS feed has answered (0.11.0)

	// WeatherAsOf is when the REFERENCE provider last completed a fetch that
	// covered this location — zero = none has yet. It is the weather half of a
	// distinction FireState.AsOf and Seismic's nil-ness already make, and the
	// only reason a never-resolving lookup could not be told from a loading one
	// (issue #13): with no attempt recorded, an empty location shimmers for
	// ever. It says the feed ANSWERED, never that it found anything.
	WeatherAsOf time.Time `json:"weather_as_of"`
}

// Section is one provider's contribution to one location.
type Section struct {
	Current *Conditions `json:"current"`
	Hourly  []Hourly    `json:"hourly,omitempty"`
	Daily   []Daily     `json:"daily,omitempty"`
	Marine  *Marine     `json:"marine,omitempty"`
}

// Marine is the coastal-waters section (B3 UAT 29). SI internal: metres,
// seconds, degrees true, °C. nil = not a coastal location / no data. The
// NWS gridpoint carries swell/wave forecasts for coastal grids; the nearest
// NDBC buoy supplies observed water temperature (and wave obs as fill).
type Marine struct {
	SwellHeight          *float64  `json:"swell_height"`           // m
	SwellDirDeg          *float64  `json:"swell_dir_deg"`          // degrees true
	WaveHeight           *float64  `json:"wave_height"`            // m (significant)
	WavePeriod           *float64  `json:"wave_period_s"`          // s (dominant)
	WindWaveHeight       *float64  `json:"wind_wave_height"`       // m
	SecondarySwellHeight *float64  `json:"secondary_swell_height"` // m (UAT 32.3)
	SecondarySwellDirDeg *float64  `json:"secondary_swell_dir_deg"`
	SecondaryPeriod      *float64  `json:"secondary_period_s"`
	WindSpeed            *float64  `json:"wind_speed"`     // m/s at the buoy
	WindGust             *float64  `json:"wind_gust"`      // m/s at the buoy
	WaterTemp            *float64  `json:"water_temp"`     // °C
	Buoy                 string    `json:"buoy,omitempty"` // NDBC station id
	BuoyDistanceKM       *float64  `json:"buoy_distance_km"`
	ObservedAt           time.Time `json:"observed_at"`
	// Tides and currents (B3 UAT 61, NOAA CO-OPS): upcoming high/low
	// predictions and the observed level at the nearest tide station;
	// max-flood / max-ebb / slack predictions at the nearest current station.
	TideLevel      *float64       `json:"tide_level"` // m above MLLW (observed)
	Tides          []TideEvent    `json:"tides"`      // upcoming highs/lows, time order
	TideStation    string         `json:"tide_station,omitempty"`
	TideStationKM  *float64       `json:"tide_station_km"`
	Currents       []CurrentEvent `json:"currents"` // upcoming max flood/ebb/slack, time order
	CurrentStation string         `json:"current_station,omitempty"`
	Source         SourceInfo     `json:"source"`
}

// TideEvent is one predicted high ("H") or low ("L") water.
type TideEvent struct {
	Time   time.Time `json:"time"`
	Height float64   `json:"height"` // m above MLLW
	Type   string    `json:"type"`   // H | L
}

// CurrentEvent is one predicted tidal-current extreme.
type CurrentEvent struct {
	Time  time.Time `json:"time"`
	Speed float64   `json:"speed"` // m/s (magnitude)
	Type  string    `json:"type"`  // flood | ebb | slack
}

// Clone deep-copies a Marine block (the event slices never alias published
// snapshots — §2 immutability). nil stays nil.
func (m *Marine) Clone() *Marine {
	if m == nil {
		return nil
	}
	c := *m
	c.Tides = append([]TideEvent(nil), m.Tides...)
	c.Currents = append([]CurrentEvent(nil), m.Currents...)
	return &c
}

// Conditions is the normalized current-weather payload (§10.1). Nil = no value.
type Conditions struct {
	ObservedAt  time.Time  `json:"observed_at"`
	Temp        *float64   `json:"temp"`         // °C
	Feels       *float64   `json:"feels"`        // °C
	Dewpoint    *float64   `json:"dewpoint"`     // °C
	HumidityPct *float64   `json:"humidity_pct"` // %
	Pressure    *float64   `json:"pressure"`     // hPa
	Wind        *float64   `json:"wind"`         // m/s
	WindDirDeg  *float64   `json:"wind_dir_deg"` // degrees
	WindGust    *float64   `json:"wind_gust"`    // m/s
	Precip1h    *float64   `json:"precip_1h"`    // mm
	PrecipProb  *float64   `json:"precip_prob_pct"`
	CloudPct    *float64   `json:"cloud_pct"`
	Visibility  *float64   `json:"visibility"` // m
	UVIndex     *float64   `json:"uv_index"`
	Condition   string     `json:"condition_code"` // WMO-mapped closed enum
	IsDay       *bool      `json:"is_day"`
	Source      SourceInfo `json:"source"`
}

// SourceInfo names where a Conditions block came from (OQ-9 provenance).
type SourceInfo struct {
	Provider       string            `json:"provider"`
	ModelOrStation string            `json:"model_or_station"`
	DistanceKm     *float64          `json:"distance_km"`
	IssuedAt       time.Time         `json:"issued_at"`
	FillFrom       map[string]string `json:"fill_from,omitempty"` // field -> provider
}

// Hourly is one forecast hour (reuses the Conditions field meanings).
type Hourly struct {
	Time       time.Time `json:"time"`
	Temp       *float64  `json:"temp"`
	PrecipProb *float64  `json:"precip_prob_pct"`
	Wind       *float64  `json:"wind"`
	WindDirDeg *float64  `json:"wind_dir_deg"`
	Condition  string    `json:"condition_code"`
}

// Daily is one forecast day.
type Daily struct {
	Date       string            `json:"date"` // YYYY-MM-DD in the location's TZ
	TempMax    *float64          `json:"temp_max"`
	TempMin    *float64          `json:"temp_min"`
	PrecipProb *float64          `json:"precip_prob_pct"`
	Condition  string            `json:"condition_code"`
	Sunrise    time.Time         `json:"sunrise"`
	Sunset     time.Time         `json:"sunset"`
	FillFrom   map[string]string `json:"fill_from,omitempty"` // field -> source, e.g. temp_max -> nws:gridpoint (UAT 71)
}

// Alert uses CAP field names verbatim (AI-10 §2; NWS-only in v0.1).
type Alert struct {
	ID            string     `json:"id"`
	Event         string     `json:"event"`
	Severity      string     `json:"severity"`
	Urgency       string     `json:"urgency"`
	Certainty     string     `json:"certainty"`
	MessageType   string     `json:"message_type"`
	Sent          time.Time  `json:"sent"`
	Effective     time.Time  `json:"effective"`
	Onset         *time.Time `json:"onset"`
	Expires       time.Time  `json:"expires"`
	Ends          *time.Time `json:"ends"`
	References    []string   `json:"references"`
	AffectedZones []string   `json:"affected_zones"`
	AreaDesc      string     `json:"area_desc"`
	Headline      string     `json:"headline"`
	Description   string     `json:"description"`
	Instruction   string     `json:"instruction"`
	SenderName    string     `json:"sender_name"` // the issuing office ("NWS Wichita KS") — the superseded guard's key (0.13.0, NFR-12)
	Source        SourceInfo `json:"source"`
}

// FireState holds hotspots and incidents (B5).
// MaxHotspots caps the hotspots a location carries (nearest first): a
// mega-fire clusters into hundreds of GOES positions per day (941 near
// Mineral Wells, TX on 2026-08-25), and three rows plus a count are all the
// modal shows. At the cap the count reads "300+".
const MaxHotspots = 300

type FireState struct {
	AsOf      time.Time  `json:"as_of"` // when a fire feed last answered for this location; zero = no feed has yet (never "no hotspots" — red-team B5 P3)
	Hotspots  []Hotspot  `json:"hotspots"`
	Incidents []Incident `json:"incidents"`

	// HotspotsAsOf and IncidentsAsOf are per-FEED, and AsOf alone was not enough
	// (REVIEW red team, 2026-09-08). AsOf is the freshest answer from ANY fire
	// feed, so with HMS up and WFIGS down it is set — and the spoken report then
	// stated "there are currently no named incidents within a 31 mile radius" as
	// a FACT, one sentence after crediting the National Interagency Fire Center
	// as a source. The mirror case says "no hotspots" while HMS and FIRMS are
	// both down.
	//
	// This is the on-screen distinction the UAT already forced — "fire feed not
	// yet available" is not "none within this radius" — applied per FEED rather
	// than per ring, and on the air rather than only on screen.
	//
	// Zero means that half was never answered for this location. A count of zero
	// is only a fact when its own stamp is set.
	HotspotsAsOf  time.Time `json:"hotspots_as_of"`
	IncidentsAsOf time.Time `json:"incidents_as_of"`
}

// Hotspot is one satellite fire detection.
type Hotspot struct {
	Lat        float64    `json:"lat"`
	Lon        float64    `json:"lon"`
	DetectedAt time.Time  `json:"detected_at"`
	Confidence string     `json:"confidence"`
	FRPMW      *float64   `json:"frp_mw"`
	DistanceKm *float64   `json:"distance_km"`
	Source     SourceInfo `json:"source"`
}

// Incident is one named wildfire incident (WFIGS).
type Incident struct {
	Name             string     `json:"name"`
	Lat              float64    `json:"lat"` // the incident point (WFIGS) — the broadcast names its direction (UAT 116)
	Lon              float64    `json:"lon"`
	Discovered       time.Time  `json:"discovered"`
	PercentContained *float64   `json:"percent_contained"`
	Acres            *float64   `json:"acres"`
	State            string     `json:"state"`
	Source           SourceInfo `json:"source"`
}

// SeismicState reports recent earthquakes near a location (0.11.0). AsOf
// is when the USGS feed last answered for this location; zero means no feed
// has yet, which is NOT "no quakes" (the FireState.AsOf precedent — a cold
// or down feed must read "unavailable", never a reassuring "none").
type SeismicState struct {
	AsOf   time.Time `json:"as_of"`
	Quakes []Quake   `json:"quakes"`
}

// Quake is one earthquake as it matters to a tracked location: the event
// plus its distance and bearing FROM that location (USGS gives neither).
type Quake struct {
	Mag        float64    `json:"mag"`
	MagType    string     `json:"mag_type"`        // ml, mww, md … (USGS magnitude scale)
	Place      string     `json:"place"`           // "52 km SSW of Progreso, B.C., MX"
	DepthKm    float64    `json:"depth_km"`        // hypocentre depth; a shallow quake is felt more
	At         time.Time  `json:"at"`              // origin time
	DistanceKm float64    `json:"distance_km"`     // from the tracked location (computed, not USGS)
	Bearing    string     `json:"bearing"`         // compass point from the location (computed)
	Tsunami    bool       `json:"tsunami"`         // a tsunami message was issued
	Alert      string     `json:"alert,omitempty"` // USGS PAGER level: green|yellow|orange|red ("" = none)
	Felt       *int       `json:"felt,omitempty"`  // Did-You-Feel-It report count (nil = none)
	Sig        int        `json:"sig"`             // USGS significance 0..1000+
	Source     SourceInfo `json:"source"`
}

// RadioState reports tuner availability (never live playback state — PD note).
type RadioState struct {
	Station   string `json:"station"`
	StreamURL string `json:"stream_url"`
	Source    string `json:"source" enum:"live,synth,none"` // documented in the schema (REVIEW M3)
	Status    string `json:"status" enum:"available,none"`
}

// ProviderStatus values.
const (
	ProviderOK       = "ok"
	ProviderDegraded = "degraded"
	ProviderOff      = "off" // registered but not a source right now (FIRMS without a key, UAT 100) — never exit 2
)

// ProviderStatus reports one provider's health in this snapshot.
type ProviderStatus struct {
	ID          string    `json:"id"`
	Role        string    `json:"role" enum:"reference,secondary"`
	Status      string    `json:"status" enum:"ok,degraded,off"` // the closed set, in the schema (REVIEW M3)
	FetchedAt   time.Time `json:"fetched_at"`
	Attribution string    `json:"attribution"`
	Inactive    bool      `json:"-"` // set by Assembler.SetInactive; published as Status "off" (never in the schema)
}

// Warning codes (closed enum v1 — §10.2).
const (
	WarnProviderError     = "provider_error"
	WarnObsStale          = "obs_stale"
	WarnAlertFeedDegraded = "alert_feed_degraded"
	WarnGeocodeFallback   = "geocode_fallback"
	WarnRadioUnavailable  = "radio_unavailable"
	WarnDeprecatedField   = "deprecated_field"
)

// Warning is one machine-readable caveat about this snapshot.
type Warning struct {
	Code     string `json:"code" enum:"provider_error,obs_stale,alert_feed_degraded,geocode_fallback,radio_unavailable,deprecated_field"` // closed enum v1 (§10.2), in the schema (REVIEW M3)
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
	Provider string `json:"provider,omitempty"`

	// The failure as VALUES, for a diagnostic that shows them in columns
	// (0.14.0). Additive and omitempty: a warning that is not an HTTP failure
	// carries none of them, and an older reader sees the file it always saw.
	Endpoint   string `json:"endpoint,omitempty"`    // the host, not the whole URL
	HTTPStatus int    `json:"http_status,omitempty"` // 0 when the failure had no status
	Blame      string `json:"blame,omitempty"`       // see Blame
}

// Who a failure belongs to, as far as the response can tell.
const (
	BlameProvider  = "provider_error"  // theirs
	BlameWatchpost = "watchpost_error" // ours
	BlameUnclear   = "unclear"         // it could be either, and saying so is the honest answer
)

// FailureDetail is what a transport error can tell a warning about itself.
// httpx's StatusError and ReachError implement it; this package does not import
// transport, so the shared vocabulary stays a leaf.
type FailureDetail interface {
	Endpoint() string
	HTTPStatus() int
}

// WithFailure fills a warning's structured half from an error that carries one,
// and decides blame from the status. An error with no detail keeps the message
// it always had and reads as unclear.
func WithFailure(w Warning, err error) Warning {
	var d FailureDetail
	if errors.As(err, &d) {
		w.Endpoint, w.HTTPStatus = d.Endpoint(), d.HTTPStatus()
	}
	w.Blame = BlameFor(w.HTTPStatus)
	return w
}

// BlameFor reads a status as whose fault it is.
//
// 5xx IS THEIRS. A server error is a server error, and 408 and 429 go with them:
// a timeout and a throttle are both the far end saying "not now".
//
// 4xx IS USUALLY OURS, because a request is something we built — a malformed
// query, a URL that no longer exists, a method they do not take. The exceptions
// are the two that turn on a credential: a 401 or a 403 is ours when we sent no
// key and theirs when the key we sent was revoked, and the response cannot tell
// the two apart. Those read UNCLEAR rather than guessing, because a diagnostic
// that blames the wrong side sends somebody to fix the wrong thing.
//
// 3xx reaching here at all means a redirect we did not follow, which is a
// question about the request as often as about the server — unclear.
//
// NO STATUS is a transport failure: DNS, a refused connection, a dead link. That
// is as likely to be the listener's own network as the provider's, so it is not
// pinned on either.
func BlameFor(status int) string {
	switch {
	case status == 0:
		return BlameUnclear
	case status >= 500, status == 408, status == 429:
		return BlameProvider
	case status == 401, status == 403:
		return BlameUnclear
	case status >= 400:
		return BlameWatchpost
	case status >= 300:
		return BlameUnclear
	}
	return BlameUnclear
}

// --- fetch plumbing (§10.1) ---

// FetchKind names one cadence class (scheduler tiers own cadence — the single
// freshness authority).
type FetchKind int

// Fetch kinds.
const (
	KindAlerts FetchKind = iota
	KindObs
	KindForecast
	KindFire
	KindProducts
	KindGeocode
	KindMarine         // coastal-waters forecast: NWS gridpoint swell/wave, CO-OPS tide/current predictions (B3 UAT 29/61)
	KindMarineObs      // coastal-waters observations: NDBC buoy files, CO-OPS water level — the fast marine tier (UAT 72)
	KindForecastHourly // NWS hourly forecast — its own tier so the RECENT list can skip its 162 KB (UAT 72)
	KindSeismic        // USGS earthquakes near the location (0.11.0)
)

// LocationRef identifies a location for fetching (Zip/TZ carried through to
// the published Location — R-2': every rendered label shows its zip).
type LocationRef struct {
	Label string
	Tag   string // user 5-char short label
	Zip   string
	Lat   float64
	Lon   float64
	TZ    string
}

// LocationKey is the normalized identity: "lat,lon" at 4 decimal places.
type LocationKey string

// Key normalizes a LocationRef to its LocationKey.
func Key(ref LocationRef) LocationKey {
	// strconv, not Sprintf: the row path asks per location per frame while
	// the radio plays (quality pass Q3); one allocation, same digits
	// (TestKeyMatchesTheSprintfForm pins the equivalence).
	var buf [48]byte
	b := strconv.AppendFloat(buf[:0], ref.Lat, 'f', 4, 64)
	b = append(b, ',')
	b = strconv.AppendFloat(b, ref.Lon, 'f', 4, 64)
	return LocationKey(b)
}

// Keys is Key over a set — the shape every Apply caller needs to say which
// locations its fetch covered (#13).
func Keys(refs []LocationRef) []LocationKey {
	out := make([]LocationKey, 0, len(refs))
	for _, r := range refs {
		out = append(out, Key(r))
	}
	return out
}

// FetchReq asks a provider for one scheduled unit of work.
type FetchReq struct {
	Kind      FetchKind
	Locations []LocationRef
	Hint      map[string]string // e.g. "key" for keyed providers
}

// PartialData is one provider's per-location contribution; nil sections are
// untouched by the merge.
type PartialData struct {
	Current *Conditions
	Hourly  []Hourly
	Daily   []Daily
	Alerts  []Alert // non-nil replaces (alert sets are authoritative per fetch)
	Fire    *FireState
	Marine  *Marine
	Seismic *SeismicState
}

// Fragment is one provider fetch result.
type Fragment struct {
	Provider    string
	Kind        FetchKind
	PerLocation map[LocationKey]PartialData
	FetchedAt   time.Time
	Err         error
}

// Provider is the only interface a data source implements (§2).
type Provider interface {
	ID() string
	Domains() []string
	Fetch(ctx context.Context, req FetchReq) (Fragment, error)
}

// The three optional fire numbers, read with the sentinel every ranking uses.
//
// EACH OF THESE WAS WRITTEN TWICE — once here beside the type and once in
// domains/fire — with identical bodies and identical sentinels. They agree
// today. The sentinels are POLICY (a missing FRP sorts last; a missing distance
// is infinitely far), so the failure mode is that one is adjusted and the other
// is not, and the merge and the ranking then disagree about the same fire while
// both look right in isolation.
//
// They live on the type because the type is what they are about. Issue #7 was
// the same shape — one operation, four hand-written copies, one of them wrong.

// FRPOrMissing is the fire radiative power, or -1 when the feed omitted it.
func (h Hotspot) FRPOrMissing() float64 {
	if h.FRPMW == nil {
		return -1
	}
	return *h.FRPMW
}

// KmOrFar is the distance in kilometres, or "infinitely far" when unknown, so
// an unplaced hotspot never wins a nearest-first comparison.
func (h Hotspot) KmOrFar() float64 {
	if h.DistanceKm == nil {
		return math.MaxFloat64
	}
	return *h.DistanceKm
}

// AcresOrMissing is the incident's size, or -1 when it has not been reported.
func (in Incident) AcresOrMissing() float64 {
	if in.Acres == nil {
		return -1
	}
	return *in.Acres
}
