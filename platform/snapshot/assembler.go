package snapshot

import (
	"github.com/branden-thompson/watchpost/platform/astro"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// Assembler is the single merge point (§2 concurrency contract): providers
// hand it Fragments; it maintains internal state under a mutex and publishes
// fresh, immutable Snapshot values. No caller ever mutates a published
// Snapshot; Snapshot() deep-copies everything reachable.
type Assembler struct {
	mu        sync.Mutex
	refs      []LocationRef
	order     []LocationKey
	providers []string
	sections  map[LocationKey]map[string]*Section // location -> provider -> data
	alerts    map[LocationKey][]Alert
	fire      map[LocationKey]map[string]*FireState // location -> provider -> its contribution (B5: HMS, WFIGS and FIRMS each add a part)
	status    map[string]*ProviderStatus
	warnings  []Warning
}

// NewAssembler builds an Assembler for the configured locations and providers.
// Duplicate location keys or provider IDs are a wiring bug and are refused by
// invariant (config validation happens upstream; this is the last line).
func NewAssembler(refs []LocationRef, providerIDs []string) *Assembler {
	return newAssembler(refs, providerIDs)
}

// maxWarnings is how many warnings a snapshot carries: the newest, enough
// for the [S] modal and the report, never a session's whole history.
const maxWarnings = 256

func newAssembler(refs []LocationRef, providerIDs []string) *Assembler {
	a := &Assembler{
		refs:     refs,
		sections: map[LocationKey]map[string]*Section{},
		alerts:   map[LocationKey][]Alert{},
		fire:     map[LocationKey]map[string]*FireState{},
		status:   map[string]*ProviderStatus{},
	}
	kept := make([]LocationRef, 0, len(refs))
	for _, r := range refs {
		k := Key(r)
		if err := invariant.Check(a.sections[k] == nil, "duplicate location key "+string(k)); err != nil {
			continue // keep the first; a duplicate ref is a config bug, not fatal
		}
		kept = append(kept, r)
		a.order = append(a.order, k)
		a.sections[k] = map[string]*Section{}
	}
	a.refs = kept // order and refs stay aligned (red-team 0.9.0 F4: a duplicate used to publish an EMPTY snapshot forever)
	for _, id := range providerIDs {
		if err := invariant.Check(id != "" && a.status[id] == nil, "provider id must be unique and non-empty"); err != nil {
			continue
		}
		a.providers = append(a.providers, id)
		a.status[id] = &ProviderStatus{ID: id, Status: ProviderOK}
	}
	return a
}

// published is a provider's status row as the snapshot carries it: "off"
// while inactive, and the role from the closed set (REVIEW M3) — nws is
// the reference harmonize defers to; every other feed is a secondary.
func published(st *ProviderStatus) ProviderStatus {
	out := *st
	if st.Inactive {
		out.Status = ProviderOff
	}
	if out.Role == "" {
		out.Role = "secondary"
		if out.ID == "nws" {
			out.Role = "reference"
		}
	}
	return out
}

// FireFor returns a tracked location's merged fire state and position
// without cloning the whole snapshot (REVIEW C2: the radio deck asks once
// per broadcast cycle and needs only this). ok is false for an untracked ref.
func (a *Assembler) FireFor(ref LocationRef) (fs FireState, lat, lon float64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := Key(ref)
	if a.sections[k] == nil {
		return FireState{}, 0, 0, false
	}
	for _, r := range a.refs {
		if Key(r) == k {
			lat, lon = r.Lat, r.Lon
		}
	}
	return mergeFire(a.fire[k]), lat, lon, true
}

// ProviderStatus reports a registered provider's current status ("" when
// unknown) — the radio deck credits FIRMS only when it answered ok.
func (a *Assembler) ProviderStatus(id string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := a.status[id]
	if st == nil {
		return ""
	}
	if st.Inactive {
		return ProviderOff
	}
	return st.Status
}

// SetInactive marks a registered provider as not a source right now (or
// back on): its status reads ProviderOff regardless of fragments, so the
// API status never says "ok" for a feed that contributes nothing (B5 /
// UAT 100: FIRMS until a key is stored). Unknown ids are a caller bug.
func (a *Assembler) SetInactive(id string, off bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := a.status[id]
	if err := invariant.Check(st != nil, "SetInactive on unregistered provider "+id); err != nil {
		return err
	}
	st.Inactive = off
	return nil
}

// SetLocations reconciles the tracked set to refs — order included: kept
// locations keep every section, alert and fire state; removed ones are
// dropped; new ones start empty (they show as loading until their first
// fragment lands). Returns the added and removed refs so the caller can
// fetch exactly the newcomers (B3 UAT 69: a lookup is one location's
// requests, never a pipeline rebuild). Duplicate refs keep the first.
func (a *Assembler) SetLocations(refs []LocationRef) (added, removed []LocationRef) {
	a.mu.Lock()
	defer a.mu.Unlock()
	keep := map[LocationKey]bool{}
	order := make([]LocationKey, 0, len(refs))
	kept := make([]LocationRef, 0, len(refs))
	for _, r := range refs {
		k := Key(r)
		if keep[k] {
			continue
		}
		keep[k] = true
		if a.sections[k] == nil {
			a.sections[k] = map[string]*Section{}
			added = append(added, r)
		}
		order = append(order, k)
		kept = append(kept, r)
	}
	for i, k := range a.order {
		if !keep[k] {
			removed = append(removed, a.refs[i])
			delete(a.sections, k)
			delete(a.alerts, k)
			delete(a.fire, k)
		}
	}
	a.order, a.refs = order, kept
	return added, removed
}

// Apply merges one Fragment: last-write-wins per (provider, location,
// domain-section). A failed Fragment (Err != nil) degrades the provider and
// appends a provider_error Warning, but whatever it DID fetch still lands
// (B3 UAT 59: one bad location must not blank the rest of the batch);
// locations it could not serve keep their prior data (§10.1; obs_stale
// never degrades status — see Warn).
func (a *Assembler) Apply(f Fragment) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, known := a.status[f.Provider]
	if err := invariant.Check(known, "fragment from unregistered provider "+f.Provider); err != nil {
		a.warnings = append(a.warnings, Warning{Code: WarnProviderError, Message: err.Error(), Provider: f.Provider})
		return
	}
	if f.Err != nil {
		st.Status = ProviderDegraded
		a.warnings = append(a.warnings, Warning{
			Code: WarnProviderError, Message: f.Err.Error(), Provider: f.Provider,
		})
	} else {
		st.Status = ProviderOK
		st.FetchedAt = f.FetchedAt
	}
	for k, pd := range f.PerLocation {
		secs, ok := a.sections[k]
		if !ok {
			continue // unknown location: fragment for a place we no longer watch
		}
		sec := secs[f.Provider]
		if sec == nil {
			sec = &Section{}
			secs[f.Provider] = sec
		}
		if pd.Current != nil {
			c := *pd.Current
			sec.Current = &c
		}
		if pd.Hourly != nil {
			sec.Hourly = append([]Hourly(nil), pd.Hourly...)
		}
		if pd.Daily != nil {
			sec.Daily = append([]Daily(nil), pd.Daily...)
		}
		if pd.Marine != nil {
			sec.Marine = pd.Marine.Clone()
		}
		if pd.Alerts != nil {
			a.alerts[k] = append([]Alert(nil), pd.Alerts...)
		}
		if pd.Fire != nil {
			fs := *pd.Fire
			if a.fire[k] == nil {
				a.fire[k] = map[string]*FireState{}
			}
			a.fire[k][f.Provider] = &fs // this provider's part; the others keep theirs
		}
	}
}

// Warn appends a snapshot-level warning. Warnings never change provider
// status (obs_stale carve-out, §10.11) — only Apply with Err degrades.
func (a *Assembler) Warn(w Warning) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := invariant.Check(w.Code != "", "warnings must carry a machine-readable code (§10.2)"); err != nil {
		w.Code = WarnProviderError
	}
	a.warnings = append(a.warnings, w)
}

// Size reports how many locations the assembler tracks and how many
// warnings it holds — the two structures a diagnostic dump watches for
// growth (quality pass Q0); no snapshot copy is made.
func (a *Assembler) Size() (locations, warnings int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.refs), len(a.warnings)
}

// SetAttribution records a provider's role and attribution line for output.
func (a *Assembler) SetAttribution(id, role, attribution string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if st, ok := a.status[id]; ok {
		st.Role, st.Attribution = role, attribution
	}
}

// Snapshot publishes a fresh immutable value: everything is copied, nothing
// aliases assembler state.
func (a *Assembler) Snapshot() *Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := invariant.Check(len(a.order) == len(a.refs), "location order and refs must stay aligned"); err != nil {
		// Misalignment would mispair labels and data — publish empty rather than wrong (RS-10).
		return &Snapshot{SchemaVersion: SchemaVersion, GeneratedAt: time.Now().UTC(),
			Warnings: []Warning{{Code: WarnProviderError, Message: err.Error()}}}
	}
	s := &Snapshot{SchemaVersion: SchemaVersion, GeneratedAt: time.Now().UTC()}
	for i, k := range a.order {
		ref := a.refs[i]
		loc := Location{
			Label:      ref.Label,
			Tag:        ref.Tag,
			Zip:        ref.Zip,
			Lat:        ref.Lat,
			Lon:        ref.Lon,
			TZ:         ref.TZ,
			ByProvider: map[string]Section{},
			Alerts:     append([]Alert(nil), a.alerts[k]...),
		}
		for pid, sec := range a.sections[k] {
			cp := Section{}
			if sec.Current != nil {
				c := *sec.Current
				cp.Current = &c
			}
			cp.Hourly = append([]Hourly(nil), sec.Hourly...)
			cp.Daily = append([]Daily(nil), sec.Daily...)
			cp.Marine = sec.Marine.Clone()
			loc.ByProvider[pid] = cp
		}
		if parts := a.fire[k]; len(parts) > 0 {
			loc.Fire = mergeFire(parts)
		}
		finalize(&loc, a.providers, s.GeneratedAt)
		s.Locations = append(s.Locations, loc)
	}
	for _, id := range a.providers {
		st := a.status[id] // single deref (P10-09)
		s.Providers = append(s.Providers, published(st))
	}
	if len(a.warnings) > maxWarnings { // bounded (red-team 0.9.0 F6): hours offline must not grow every publish
		a.warnings = append([]Warning(nil), a.warnings[len(a.warnings)-maxWarnings:]...)
	}
	s.Warnings = append([]Warning(nil), a.warnings...)
	if s.Warnings == nil {
		s.Warnings = []Warning{}
	}
	if s.Locations == nil {
		s.Locations = []Location{}
	}
	if s.Providers == nil {
		s.Providers = []ProviderStatus{}
	}
	return s
}

// mergeFire folds every provider's fire contribution into one FireState
// (B5): hotspots deduped across feeds (the same fire seen by HMS and FIRMS
// — ~300 m, same UTC day — keeps the strongest reading) and sorted nearest
// first; incidents deduped by name, largest first. Providers merge in name
// order so the result is stable.
func mergeFire(parts map[string]*FireState) FireState {
	ids := make([]string, 0, len(parts))
	for id := range parts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	type spotKey struct {
		lat, lon int
		day      string
	}
	spots := map[spotKey]int{}
	names := map[string]int{}
	out := FireState{Hotspots: []Hotspot{}, Incidents: []Incident{}}
	for _, id := range ids {
		if t := parts[id].AsOf; t.After(out.AsOf) {
			out.AsOf = t // the freshest answer from any fire feed
		}
	}
	for _, id := range ids {
		for _, h := range parts[id].Hotspots {
			k := spotKey{int(math.Round(h.Lat / 0.003)), int(math.Round(h.Lon / 0.003)), h.DetectedAt.UTC().Format("2006-01-02")}
			if i, ok := spots[k]; ok {
				if frpOf(h) > frpOf(out.Hotspots[i]) {
					out.Hotspots[i] = h
				}
				continue
			}
			spots[k] = len(out.Hotspots)
			out.Hotspots = append(out.Hotspots, h)
		}
		for _, in := range parts[id].Incidents {
			if i, ok := names[in.Name]; ok {
				if in.Source.IssuedAt.After(out.Incidents[i].Source.IssuedAt) {
					out.Incidents[i] = in
				}
				continue
			}
			names[in.Name] = len(out.Incidents)
			out.Incidents = append(out.Incidents, in)
		}
	}
	sort.SliceStable(out.Hotspots, func(i, j int) bool { return kmOf(out.Hotspots[i]) < kmOf(out.Hotspots[j]) })
	sort.SliceStable(out.Incidents, func(i, j int) bool { return acresOf(out.Incidents[i]) > acresOf(out.Incidents[j]) })
	return out
}

func frpOf(h Hotspot) float64 {
	if h.FRPMW == nil {
		return -1
	}
	return *h.FRPMW
}

func kmOf(h Hotspot) float64 {
	if h.DistanceKm == nil {
		return math.MaxFloat64
	}
	return *h.DistanceKm
}

func acresOf(in Incident) float64 {
	if in.Acres == nil {
		return -1
	}
	return *in.Acres
}

// finalize derives a location's published view from its provider sections:
// harmonize across providers, rehydrate sparse observations from the
// forecast, compute sun times, then normalize collections for machine mode.
func finalize(loc *Location, providers []string, now time.Time) {
	harmonize(loc, providers)
	rehydrateFromForecast(loc, now)
	fillSunTimes(loc)
	normalizeCollections(loc)
}

// normalizeCollections guarantees the machine-mode contract: collections are
// always arrays (possibly empty), never null — agents get stable types
// (AI-10 §1 conventions; the null-parity rule covers VALUE fields only).
func normalizeCollections(loc *Location) {
	if loc.Alerts == nil {
		loc.Alerts = []Alert{}
	}
	if loc.Hourly == nil {
		loc.Hourly = []Hourly{}
	}
	if loc.Daily == nil {
		loc.Daily = []Daily{}
	}
	if loc.Fire.Hotspots == nil {
		loc.Fire.Hotspots = []Hotspot{}
	}
	if loc.Fire.Incidents == nil {
		loc.Fire.Incidents = []Incident{}
	}
	if loc.ByProvider == nil {
		loc.ByProvider = map[string]Section{}
	}
	if loc.Radio.Source == "" { // the tuner block's closed sets never publish "" (REVIEW M3)
		loc.Radio.Source = "none"
	}
	if loc.Radio.Status == "" {
		loc.Radio.Status = "none"
	}
	normalizeMarine(loc.Marine)
	for _, sec := range loc.ByProvider { // every provider copy too: by_provider.*.marine.tides/currents are arrays, never null (REVIEW M1)
		normalizeMarine(sec.Marine)
	}
}

// normalizeMarine empties a marine block's nil collections (schema: arrays).
func normalizeMarine(m *Marine) {
	if m == nil {
		return
	}
	if m.Tides == nil {
		m.Tides = []TideEvent{}
	}
	if m.Currents == nil {
		m.Currents = []CurrentEvent{}
	}
}

// harmonize fills Location.Harmonized/Hourly/Daily per the OQ-9 rule: NWS wins
// outright; secondaries fill only nil fields (fill_from recorded); no blending.
// Provider order = configured order after "nws".
func harmonize(loc *Location, providerOrder []string) {
	order := make([]string, 0, len(providerOrder))
	for _, id := range providerOrder {
		if id == "nws" {
			order = append([]string{"nws"}, order...)
		} else {
			order = append(order, id)
		}
	}
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok || sec.Current == nil {
			continue
		}
		if loc.Harmonized.Source.Provider == "" {
			loc.Harmonized = *sec.Current
			continue
		}
		fillFrom(&loc.Harmonized, sec.Current, id)
	}
	// Hourly/Daily: first provider in order that has them (no cross-provider
	// splicing of time series).
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok {
			continue
		}
		if loc.Hourly == nil && len(sec.Hourly) > 0 {
			loc.Hourly = append([]Hourly(nil), sec.Hourly...)
		}
		if loc.Daily == nil && len(sec.Daily) > 0 {
			loc.Daily = append([]Daily(nil), sec.Daily...)
		}
	}
	harmonizeMarine(loc, order)
}

// keepOr returns have when set, else a copy of the fallback (single-deref
// value helper, P10-09 pointer discipline).
func keepOr(have, fallback *float64) *float64 {
	if have != nil || fallback == nil {
		return have
	}
	v := *fallback
	return &v
}

// fillForecast is the FillFrom provenance for values rehydrated from the
// location's own hourly forecast rather than a second provider.
const fillForecast = "forecast"

// rehydrateFromForecast fills a SPARSE observation's holes from the hourly
// forecast period covering now (B3 UAT 59): mesonet stations publish no sky
// condition and an intermittent temperature, which read as "UNKNOWN n/a".
// Observed values are never replaced; provenance is recorded; a location
// with no observation at all stays a loading state (the obs retry owns it).
func rehydrateFromForecast(loc *Location, now time.Time) {
	c := &loc.Harmonized
	if c.Source.Provider == "" || (c.Temp != nil && c.Condition != "" && c.Condition != "unknown") {
		return
	}
	h, ok := hourCovering(loc.Hourly, now)
	if !ok {
		return
	}
	record := func(field string) {
		if c.Source.FillFrom == nil {
			c.Source.FillFrom = map[string]string{}
		}
		c.Source.FillFrom[field] = fillForecast
	}
	if c.Temp == nil && h.Temp != nil {
		c.Temp = keepOr(nil, h.Temp)
		record("temp")
	}
	if (c.Condition == "" || c.Condition == "unknown") && h.Condition != "" && h.Condition != "unknown" {
		c.Condition = h.Condition
		record("condition_code")
	}
}

// hourCovering returns the forecast period whose hour contains now.
func hourCovering(hours []Hourly, now time.Time) (Hourly, bool) {
	for _, h := range hours {
		if !now.Before(h.Time) && now.Before(h.Time.Add(time.Hour)) {
			return h, true
		}
	}
	return Hourly{}, false
}

// fillSunTimes computes sunrise/sunset for every Daily row that lacks them
// (no provider carries them — B3 UAT 32.4): geometry from lat/lon in the
// location's timezone.
func fillSunTimes(loc *Location) {
	tz, err := tz.Location(loc.TZ)
	if err != nil || loc.TZ == "" {
		tz = time.UTC
	}
	for i := range loc.Daily {
		d := &loc.Daily[i]
		if !d.Sunrise.IsZero() && !d.Sunset.IsZero() {
			continue
		}
		date, perr := time.ParseInLocation("2006-01-02", d.Date, tz)
		if perr != nil {
			continue
		}
		if rise, set, ok := astro.SunTimes(loc.Lat, loc.Lon, date, tz); ok {
			d.Sunrise, d.Sunset = rise, set
		}
	}
}

// harmonizeMarine merges the coastal-waters section field-wise across
// providers in order (forecast provider first, buoy fills the rest — water
// temperature only ever comes from the buoy). nil stays nil inland.
func harmonizeMarine(loc *Location, order []string) {
	for _, id := range order {
		sec, ok := loc.ByProvider[id]
		if !ok || sec.Marine == nil {
			continue
		}
		if loc.Marine == nil {
			loc.Marine = sec.Marine.Clone()
			continue
		}
		fillMarine(loc.Marine, sec.Marine)
	}
}

// fillMarine copies src's non-nil fields into dst's nil fields (never
// replaces); buoy identity travels with the water temperature.
func fillMarine(dst, src *Marine) {
	dst.SwellHeight = keepOr(dst.SwellHeight, src.SwellHeight)
	dst.SwellDirDeg = keepOr(dst.SwellDirDeg, src.SwellDirDeg)
	dst.WaveHeight = keepOr(dst.WaveHeight, src.WaveHeight)
	dst.WavePeriod = keepOr(dst.WavePeriod, src.WavePeriod)
	dst.WindWaveHeight = keepOr(dst.WindWaveHeight, src.WindWaveHeight)
	dst.SecondarySwellHeight = keepOr(dst.SecondarySwellHeight, src.SecondarySwellHeight)
	dst.SecondarySwellDirDeg = keepOr(dst.SecondarySwellDirDeg, src.SecondarySwellDirDeg)
	dst.SecondaryPeriod = keepOr(dst.SecondaryPeriod, src.SecondaryPeriod)
	dst.WindSpeed = keepOr(dst.WindSpeed, src.WindSpeed)
	dst.WindGust = keepOr(dst.WindGust, src.WindGust)
	dst.WaterTemp = keepOr(dst.WaterTemp, src.WaterTemp)
	dst.BuoyDistanceKM = keepOr(dst.BuoyDistanceKM, src.BuoyDistanceKM)
	if dst.Buoy == "" {
		dst.Buoy = src.Buoy
	}
	if dst.ObservedAt.IsZero() {
		dst.ObservedAt = src.ObservedAt
	}
	fillTides(dst, src)
}

// fillTides copies the tide/current block (UAT 61) into a section that has
// none — the whole block travels together with its station identity.
func fillTides(dst, src *Marine) {
	dst.TideLevel = keepOr(dst.TideLevel, src.TideLevel)
	if len(dst.Tides) == 0 && len(src.Tides) > 0 {
		dst.Tides = append([]TideEvent(nil), src.Tides...)
		dst.TideStation, dst.TideStationKM = src.TideStation, keepOr(nil, src.TideStationKM)
	}
	if len(dst.Currents) == 0 && len(src.Currents) > 0 {
		dst.Currents = append([]CurrentEvent(nil), src.Currents...)
		dst.CurrentStation = src.CurrentStation
	}
}

// fillFrom copies src's non-nil fields into nil fields of dst, recording
// provenance in dst.Source.FillFrom (never replaces existing values).
// DEFERRED (§10.11, lands with multi-provider B5): the staleness cutoff —
// fill only when src.ObservedAt is within 2x the field's tier cadence. With a
// single provider (B1) there is nothing to fill from, so the gap is inert;
// the B5 harmonization goldens must cover it (tracked in the B1 ledger).
func fillFrom(dst, src *Conditions, srcID string) {
	record := func(field string) {
		if dst.Source.FillFrom == nil {
			dst.Source.FillFrom = map[string]string{}
		}
		dst.Source.FillFrom[field] = srcID
	}
	fill := func(d, s *float64, name string) *float64 {
		if d != nil || s == nil {
			return d
		}
		v := *s
		record(name)
		return &v
	}
	dst.Temp = fill(dst.Temp, src.Temp, "temp")
	dst.Feels = fill(dst.Feels, src.Feels, "feels")
	dst.Dewpoint = fill(dst.Dewpoint, src.Dewpoint, "dewpoint")
	dst.HumidityPct = fill(dst.HumidityPct, src.HumidityPct, "humidity_pct")
	dst.Pressure = fill(dst.Pressure, src.Pressure, "pressure")
	dst.Wind = fill(dst.Wind, src.Wind, "wind")
	dst.WindDirDeg = fill(dst.WindDirDeg, src.WindDirDeg, "wind_dir_deg")
	dst.WindGust = fill(dst.WindGust, src.WindGust, "wind_gust")
	dst.Precip1h = fill(dst.Precip1h, src.Precip1h, "precip_1h")
	dst.PrecipProb = fill(dst.PrecipProb, src.PrecipProb, "precip_prob_pct")
	dst.CloudPct = fill(dst.CloudPct, src.CloudPct, "cloud_pct")
	dst.Visibility = fill(dst.Visibility, src.Visibility, "visibility")
	dst.UVIndex = fill(dst.UVIndex, src.UVIndex, "uv_index")
	if dst.IsDay == nil && src.IsDay != nil {
		v := *src.IsDay
		dst.IsDay = &v
		record("is_day")
	}
	if dst.Condition == "" && src.Condition != "" {
		dst.Condition = src.Condition
		record("condition_code")
	}
}
