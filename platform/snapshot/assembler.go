package snapshot

import (
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
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
	seismic   map[LocationKey]*SeismicState         // location -> its latest USGS state (0.11.0: one provider, no cross-merge)
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
