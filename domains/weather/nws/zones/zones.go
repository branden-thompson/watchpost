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
	"strings"
	"sync"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// DefaultBase is the service the alerts themselves come from.
const DefaultBase = "https://api.weather.gov"

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

// Store serves zone shapes by id. It is safe for concurrent use.
type Store struct {
	client *httpx.Client
	base   string

	mu   sync.RWMutex
	held map[string]Zone
}

// New makes a store that reads from base.
func New(client *httpx.Client, base string) *Store {
	if base == "" {
		base = DefaultBase
	}
	return &Store{client: client, base: strings.TrimRight(base, "/"), held: map[string]Zone{}}
}

// Zone is one zone's shape, fetched if it is not already held.
func (s *Store) Zone(ctx context.Context, id string) (Zone, error) {
	if s == nil || id == "" {
		return Zone{}, fmt.Errorf("zones: no id")
	}
	s.mu.RLock()
	z, ok := s.held[id]
	s.mu.RUnlock()
	if ok {
		return z, nil
	}
	z, err := s.fetch(ctx, id)
	if err != nil {
		return Zone{}, err
	}
	s.mu.Lock()
	s.held[id] = z
	s.mu.Unlock()
	return z, nil
}

// Zones is the shapes for a set of ids, and the ids it could not get.
//
// **It answers with both rather than failing as a whole.** Whether an alert
// with a missing zone should draw at all is a question for whatever draws it,
// which knows what is on screen; this layer has no view to decide against
// (MG-10).
func (s *Store) Zones(ctx context.Context, ids []string) (map[string]Zone, []string) {
	out := make(map[string]Zone, len(ids))
	var missing []string
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue // the same zone is named by several alerts; ask once
		}
		seen[id] = true
		z, err := s.Zone(ctx, id)
		if err != nil {
			missing = append(missing, id)
			continue
		}
		out[id] = z
	}
	return out, missing
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

func (s *Store) fetch(ctx context.Context, id string) (Zone, error) {
	u := s.base + "/zones/forecast/" + url.PathEscape(id)
	var payload zonePayload
	if _, err := s.client.GetJSON(ctx, u, &payload); err != nil {
		return Zone{}, fmt.Errorf("zones: %s: %w", id, err)
	}
	area, err := geo.ReadGeometry(payload.Geometry)
	if err != nil {
		return Zone{}, fmt.Errorf("zones: %s: %w", id, err)
	}
	name := payload.Properties.Name
	return Zone{ID: id, Name: name, Area: area}, nil
}
