package snapshot

import (
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/plaintext"
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
	fire      map[LocationKey]map[string]*FireState   // location -> provider -> its contribution (B5: HMS, WFIGS and FIRMS each add a part)
	seismic   map[LocationKey]*SeismicState           // location -> its latest USGS state (0.11.0: one provider, no cross-merge)
	asked     map[LocationKey]map[FetchKind]time.Time // location -> per KIND, when a reference fetch covering it completed (#13)
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
		seismic:  map[LocationKey]*SeismicState{},
		asked:    map[LocationKey]map[FetchKind]time.Time{},
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

// SeismicFor is the radio deck's narrow read (P4): a location's latest seismic
// state and its coordinates, without cloning the whole snapshot per cycle. ok
// is false when the location is not tracked or no seismic feed has answered.
func (a *Assembler) SeismicFor(ref LocationRef) (ss *SeismicState, lat, lon float64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := Key(ref)
	state := a.seismic[k]
	if state == nil {
		return nil, 0, 0, false
	}
	for _, r := range a.refs {
		if Key(r) == k {
			lat, lon = r.Lat, r.Lon
		}
	}
	return cloneSeismic(state), lat, lon, true
}

// MarineFor is one location's merged coastal block, for the maritime report.
//
// The NARROW read, like SeismicFor: the radio deck asks per cycle, and cloning
// the whole snapshot to reach one location's sea state would be work for
// nothing (REVIEW C2).
//
// It merges through harmonizeMarine — the SAME body the publisher uses — rather
// than re-implementing the field-wise loop. Two implementations of "which
// provider's wave height wins" is how the Details view and the maritime report
// would start disagreeing about the same sea.
func (a *Assembler) MarineFor(ref LocationRef) (m *Marine, tz string, lat, lon float64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := Key(ref)
	if len(a.sections[k]) == 0 {
		return nil, "", 0, 0, false
	}
	scratch := &Location{ByProvider: map[string]Section{}}
	for pid, sec := range a.sections[k] {
		scratch.ByProvider[pid] = Section{Marine: sec.Marine.Clone()}
	}
	harmonizeMarine(scratch, a.providers)
	if scratch.Marine == nil {
		return nil, "", 0, 0, false // inland: not "no data", not applicable
	}
	for i, r := range a.refs {
		if a.order[i] == k {
			tz, lat, lon = r.TZ, r.Lat, r.Lon
		}
	}
	return scratch.Marine, tz, lat, lon, true
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
			delete(a.seismic, k)
			// AND THE ATTEMPT RECORD. Without this a location removed and
			// re-added in the same session inherited the stamp of its previous
			// life and read "n/a" — asserting "we asked and there is nothing
			// here" before a single fetch had been issued for it (REVIEW red
			// team, 2026-09-08). It also stopped the map growing for ever
			// across removals.
			delete(a.asked, k)
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
// asked is which locations the fetch COVERED. It is a PARAMETER, not a field on
// Fragment, because a field can be forgotten and a parameter cannot: the first
// version carried it on the Fragment, three of the four call sites set it, and
// the one that did not — platform/sched, the ONLY path the dashboard refreshes
// through — silently recorded no attempt for any location, ever. The fix ran in
// `watchpost report` and nowhere a listener could see it (red team, 2026-09-08).
//
// PerLocation cannot answer this: a provider that returned nothing for a
// location looks exactly like one nobody asked about, and telling those apart is
// the whole of issue #13.
func (a *Assembler) Apply(f Fragment, asked []LocationKey) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, known := a.status[f.Provider]
	if err := invariant.Check(known, "fragment from unregistered provider "+f.Provider); err != nil {
		a.warnings = append(a.warnings, cleanWarning(Warning{Code: WarnProviderError, Message: err.Error(), Provider: f.Provider}))
		return
	}
	if f.Err != nil {
		st.Status = ProviderDegraded
		a.warnings = append(a.warnings, cleanWarning(WithFailure(Warning{
			Code: WarnProviderError, Message: f.Err.Error(), Provider: f.Provider,
		}, f.Err)))
	} else {
		st.Status = ProviderOK
		st.FetchedAt = f.FetchedAt
	}
	// THE ATTEMPT IS RECORDED, NOT THE RESULT (#13) — but only when the fetch
	// can actually answer the question the row is asking, and only when the
	// provider was REACHABLE. Both qualifications were missing and both were
	// wrong in a way a listener would see (red team, 2026-09-08).
	//
	// WHY THE KIND MATTERS. A row reads as loading until it has conditions AND a
	// daily forecast, so only those two fetches answer it. The alerts tier is a
	// single GET and starts at the same instant as obs (three chained GETs), so
	// it lands first — and stamping on it ended the shimmer across the whole
	// board a second into every cold start, flashing "n/a" for temperatures that
	// were on their way. app/pipelines.go already refuses to stamp on the
	// supplementary hourly fetch for exactly this reason; the same hazard was
	// left standing on the path that matters.
	//
	// answersTheRow HERE IS A STORAGE FILTER, NOT THE GUARD, and saying so is the
	// point: weatherAsOf requires BOTH answering kinds, so an alerts fragment
	// could be recorded and still end nothing. Removing this line changes no
	// behaviour — a plant proved it — it only stops the map growing an entry per
	// kind that nobody reads. The protection lives in weatherAsOf; a comment
	// claiming it lives here would be the same false attribution this round has
	// been removing.
	//
	// WHY REACHABILITY MATTERS, PER LOCATION. An earlier version stamped every
	// asked location whenever the fragment served ANYBODY, on the reasoning that
	// a served location proves the provider answered. That is right for a total
	// outage and WRONG FOR A PARTIAL ONE: with A served and B refused, B was
	// stamped and its row read "n/a" — asserting an absence for a location the
	// request never reached.
	//
	// Fragment.Failed now says which locations failed and why, so the question is
	// asked per location rather than per fragment. A 404 for a point outside the
	// forecast area IS an answer — "we do not cover you" — and the row should say
	// n/a. A refused connection is not, and the row should keep waiting.
	//
	// This also closes issue #13's last hole: a single-location watchlist whose
	// only location the feed genuinely does not cover now gets a truthful n/a
	// instead of shimmering for ever, which the previous rule could not tell from
	// an outage.
	if st.Role == "reference" && answersTheRow(f.Kind) {
		at := f.FetchedAt
		if at.IsZero() {
			// A stamp cannot be zero: zero is how "never asked" is spelled, and a
			// provider may simply not set FetchedAt.
			at = time.Now().UTC()
		}
		for _, k := range asked {
			if _, tracked := a.sections[k]; !tracked {
				continue
			}
			if Unreachable(f.Failed[k]) {
				continue // we could not ask about this one; it is still waiting
			}
			if a.asked[k] == nil {
				a.asked[k] = map[FetchKind]time.Time{}
			}
			a.asked[k][f.Kind] = at
		}
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
		if pd.Seismic != nil {
			a.seismic[k] = pd.Seismic // the one seismic provider's latest state (0.11.0)
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
	a.warnings = append(a.warnings, cleanWarning(w))
}

// cleanWarning takes a warning's provider-supplied text through the plaintext
// boundary.
//
// A warning's Message is a PROVIDER'S OWN WORDS — several feeds put their error
// envelope's text straight into it — and it reaches a terminal on more than one
// surface. Escape sequences a server sends can address that terminal: an OSC 52
// writes the user's clipboard, an OSC 8 paints a hyperlink over honest text, a
// bidi override reverses a line. Cleaning at the boundary means no render site
// has to remember, which is the only way this stays true as surfaces are added.
func cleanWarning(w Warning) Warning {
	w.Message = plaintext.Line(w.Message)
	w.Endpoint = plaintext.Line(w.Endpoint)
	w.Location = plaintext.Line(w.Location)
	return w
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

// answersTheRow names the fetches whose completion can end a row's "loading"
// state. A row shows conditions and a daily forecast, so those two answer it;
// alerts, fire, marine and the rest do not, however fast they arrive.
func answersTheRow(k FetchKind) bool { return k == KindObs || k == KindForecast }

// weatherAsOf is when the LAST of the answering fetches completed for this
// location, or zero while either is outstanding. Both are required because
// either alone leaves half the row empty — obs without forecast is a row with a
// temperature and no high/low, which would read "n/a" while the forecast is
// still in flight.
func (a *Assembler) weatherAsOf(k LocationKey) time.Time {
	obs, okObs := a.asked[k][KindObs]
	fc, okFc := a.asked[k][KindForecast]
	if !okObs || !okFc {
		return time.Time{}
	}
	if fc.After(obs) {
		return fc
	}
	return obs
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
		loc := Location{ // a config file's or a resolver's text is cleaned ONCE here, for every surface that draws it (NFR-6, R5-C-05)
			Label:       plaintext.Line(ref.Label),
			Tag:         plaintext.Line(ref.Tag),
			Zip:         plaintext.Line(ref.Zip),
			Lat:         ref.Lat,
			Lon:         ref.Lon,
			TZ:          ref.TZ,
			ByProvider:  map[string]Section{},
			Alerts:      append([]Alert(nil), a.alerts[k]...),
			WeatherAsOf: a.weatherAsOf(k), // zero until BOTH answering fetches have covered it (#13)
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
		if ss := a.seismic[k]; ss != nil {
			loc.Seismic = cloneSeismic(ss) // deep copy: the published snapshot aliases no assembler state
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
