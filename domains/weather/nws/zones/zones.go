// Package zones holds the shapes of the forecast zones an alert names.
//
// **Four alerts in five carry no polygon of their own** and name zones
// instead, so a map that draws only the polygons draws almost nothing. This is
// what fills that gap: a zone fetched once, kept whole, and served by its id.
//
// It does no caching of its own beyond remembering the shapes it has parsed.
// The client already keeps a URL-keyed store with a disk tier, honours the
// lifetime a server declares, and revalidates a stale entry rather than
// discarding it - which is exactly what MG-11 and MG-13 asked for, already
// built. What is kept here is the *parsed* shape, because parsing the text
// again for every frame would be the waste.
package zones

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// DefaultBase is the service the alerts themselves come from.
const DefaultBase = "https://api.weather.gov"

// fetchAtOnce bounds how many zones are asked for at the same time. The
// service is a public good and this is a weather station, not a crawler.
const fetchAtOnce = 6

// Zone is one forecast zone: what it is called, and what it covers.
//
// **The name is kept because it arrives in the same answer** and because a
// description that says "the whole of Allen county" needs it. Fetching three
// thousand zones again later for a field we already had would be indefensible
// (MG-9).
type Zone struct {
	ID   string
	Name string
	Area geo.Shape
}

// maxHeld bounds how many shapes are kept. A zone asked for once is worth
// keeping - they change a few times a year - but **every zone ever asked for,
// on a program that runs for days, is not** (RT-4). Four hundred and
// seventy-seven were in play nationally at one measurement; this is room for
// well over that, and for a watchlist's own zones many times over.
const maxHeld = 2_000

// maxAtOnce bounds how many distinct zones one resolve may ask for.
//
// **Nothing bounded the fan-out**, and one alerts response names the zones:
// four thousand alerts of fifty zones each is two hundred thousand requests,
// and the client paces every request the program makes at five a second, so
// that is eleven hours in which no weather is fetched at all. Measured
// nationally, 477 zones were in play at once across every active alert, so
// this is room for that and more, and the refusal is reported rather than
// silent.
const maxAtOnce = 512

// maxAge is how long a held shape is served before it is fetched again on its
// next use (0.18.0 FR-4.6, D-43): the service redraws a zone a few times a
// year, and a map drawing last year's line would be wrong without saying so.
const maxAge = 7 * 24 * time.Hour

// Store serves zone shapes by id. It is safe for concurrent use.
type Store struct {
	client *httpx.Client
	base   string

	// limit is maxHeld, except in this package's own tests, which lower it to
	// drive the forgetting through Zone rather than calling forget by hand -
	// the call site was what nothing exercised (RT-4).
	limit int

	mu   sync.RWMutex
	held map[string]Zone
	at   map[string]time.Time // when each held shape was fetched
	now  func() time.Time     // nil is the wall clock; the tests fix it

	// What this has done, for the diagnostics window. **A new path over the
	// network with no counters is invisible** (RT-5): when a map is blank
	// there is otherwise nothing to say whether the shapes failed or were
	// never asked for.
	fetched atomic.Int64
	failed  atomic.Int64
	served  atomic.Int64
}

// Stats is what the store has done since launch.
type Stats struct {
	Fetched int64 `json:"fetched"` // asked of the service
	Failed  int64 `json:"failed"`  // asked and not got
	Served  int64 `json:"served"`  // answered from what was already held
	Held    int   `json:"held"`    // shapes in hand now
}

// clock is the store's time: the wall clock, unless a test fixed it.
func (s *Store) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// Stats reports what this store has done.
func (s *Store) Stats() Stats {
	if s == nil {
		return Stats{}
	}
	return Stats{Fetched: s.fetched.Load(), Failed: s.failed.Load(), Served: s.served.Load(), Held: s.Held()}
}

// New makes a store that reads from base.
func New(client *httpx.Client, base string) *Store {
	if base == "" {
		base = DefaultBase
	}
	return &Store{client: client, base: strings.TrimRight(base, "/"), held: map[string]Zone{}, at: map[string]time.Time{}, limit: maxHeld}
}

// Zone is one zone's shape, fetched if it is not already held.
func (s *Store) Zone(ctx context.Context, id string) (Zone, error) {
	if s == nil || id == "" {
		return Zone{}, fmt.Errorf("zones: no id")
	}
	s.mu.RLock()
	z, ok := s.held[id]
	at := s.at[id]
	s.mu.RUnlock()
	if ok && s.clock().Sub(at) < maxAge {
		s.served.Add(1)
		return z, nil
	}
	s.fetched.Add(1)
	z, err := s.fetch(ctx, id)
	if err != nil {
		s.failed.Add(1)
		return Zone{}, err
	}
	s.mu.Lock()
	s.held[id], s.at[id] = z, s.clock()
	over := len(s.held) > s.limit
	s.mu.Unlock()
	if over {
		s.forget()
	}
	return z, nil
}

// Zones is the shapes for a set of ids, and the ids it could not get.
//
// **It answers with both rather than failing as a whole.** Whether an alert
// with a missing zone should draw at all is a question for whatever draws it,
// which knows what is on screen; this layer has no view to decide against
// (MG-10).
func (s *Store) Zones(ctx context.Context, ids []string) (map[string]Zone, []string) {
	// **Asked for together, not one after another** (RT-3). The tail is
	// forty-two zones, and a sixth of a second each in turn is six seconds of
	// a map with nothing on it. The service is asked once per distinct id.
	wanted := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue // the same zone is named by several alerts
		}
		seen[id] = true
		wanted = append(wanted, id)
	}
	var (
		mu      sync.Mutex
		out     = make(map[string]Zone, len(wanted))
		missing []string
		g       errgroup.Group
	)
	// **Past the cap the rest are reported missing, not quietly dropped.** A
	// caller that asked for more than the whole country has at once gets an
	// answer saying so, and the ids it did not get.
	if len(wanted) > maxAtOnce {
		missing = append(missing, wanted[maxAtOnce:]...)
		wanted = wanted[:maxAtOnce]
	}
	g.SetLimit(fetchAtOnce)
	for _, id := range wanted {
		g.Go(func() (err error) {
			// **The guard has to be in the goroutine that panics.** A recover
			// in the caller cannot catch this one: recover only works in its
			// own goroutine, and these are children of it. The seeding path
			// had exactly that mistake, so a nil client took the program down
			// through a guard written to stop it.
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					missing = append(missing, id)
					mu.Unlock()
					s.failed.Add(1)
				}
			}()
			z, err := s.Zone(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				missing = append(missing, id)
				return nil // one zone that cannot be got is not the set failing
			}
			out[id] = z
			return nil
		})
	}
	_ = g.Wait() // nothing above returns an error; the misses are the answer
	sort.Strings(missing)
	return out, missing
}

// forget drops shapes once there are more than the cap. It keeps no order of
// use: a zone is cheap to fetch again and the cap exists to bound the store,
// not to be clever about which shape was wanted most recently.
func (s *Store) forget() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.held {
		if len(s.held) <= s.limit {
			return
		}
		delete(s.held, id)
		delete(s.at, id)
	}
}

// Seed takes the zones of the places being watched, before any alert needs
// them.
//
// **On demand alone is coldest exactly when the map matters** - severe weather
// activates many zones at once - and a watched location has about two zones, so
// this is a second's work, once (MG-7). A zone that cannot be got is simply not
// held: a cold start with no network is an ordinary morning, and the ordinary
// path will ask again.
func (s *Store) Seed(ctx context.Context, ids []string) {
	if s == nil {
		return
	}
	s.Zones(ctx, ids)
}

// Held is how many shapes are in hand, for a caller that wants to know whether
// seeding has finished.
func (s *Store) Held() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.held)
}

// zonePayload is the part of the answer this reads. The geometry stays raw
// until the bounded reader takes it: decoding it as part of this would be the
// unbounded recursion that reader exists to avoid.
type zonePayload struct {
	Properties struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"properties"`
	Geometry json.RawMessage `json:"geometry"`
}

// pathFor is where a zone of this id lives.
//
// **A UGC id says its own kind in its third character** - `INZ027` is a
// forecast zone, `INC003` a county - and the service keeps the two apart:
// `/zones/forecast/INC003` is a 404 and `/zones/county/INC003` is the shape.
// Asking for everything under one of them lost every alert that named the
// other, which on the day this was written was 47 of 332 active alerts, 45 of
// them Flood Warnings.
//
// The kind is read, never guessed: an id of neither kind is refused rather
// than sent hopefully to one of them.
func pathFor(id string) (string, error) {
	if len(id) < 4 {
		return "", fmt.Errorf("zones: %q is too short to be a zone id", id)
	}
	switch id[2] {
	case 'Z':
		return "/zones/forecast/", nil
	case 'C':
		return "/zones/county/", nil
	}
	return "", fmt.Errorf("zones: %q names neither a forecast zone nor a county", id)
}

func (s *Store) fetch(ctx context.Context, id string) (Zone, error) {
	kind, err := pathFor(id)
	if err != nil {
		return Zone{}, err
	}
	u := s.base + kind + url.PathEscape(id)
	var payload zonePayload
	if _, err := s.client.GetJSON(ctx, u, &payload); err != nil {
		return Zone{}, fmt.Errorf("zones: %s: %w", id, err)
	}
	area, err := geo.ReadGeometry(payload.Geometry)
	if err != nil {
		return Zone{}, fmt.Errorf("zones: %s: %w", id, err)
	}
	// **Clamped like every other string from outside** (R5-C-05). It was the
	// one that was not: a hostile 8 MB name was held whole, and the transport
	// ceiling times the store's cap is tens of gigabytes.
	name := plaintext.ClampField(payload.Properties.Name)
	return Zone{ID: id, Name: name, Area: area}, nil
}

// Forget drops every shape the store holds, and says how many (0.18.0 W3.8:
// "Clear map data"). The next use fetches again.
func (s *Store) Forget() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.held)
	s.held, s.at = map[string]Zone{}, map[string]time.Time{}
	return n
}

// Base is the service the store reads from, so a caller that clears the HTTP
// cache's copies of its answers can name them.
func (s *Store) Base() string {
	if s == nil {
		return ""
	}
	return s.base
}

// ForgetCached drops the HTTP cache's copies of the store's answers - every
// zone outline, in memory and on disk - and nothing else it caches (0.18.0
// W3.8, FR-3.10: the zone geometry is a record of where the map looked).
func (s *Store) ForgetCached() (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	return s.client.ForgetPrefix(s.base + "/zones/")
}
