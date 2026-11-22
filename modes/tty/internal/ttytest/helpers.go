package ttytest

// helpers.go — the pure helpers more than one package's tests need:
// stripping ANSI, the first line two frames differ at, a card, the masthead's
// snapshot, the loaded pool's size, and the masthead guard's check (design
// CK-4).

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// StripANSI removes SGR sequences for column math in tests.
func StripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

// FirstDiffLine is the first line, counted from 1, at which two frames differ, and
// what each has there ("" past a frame's end).
func FirstDiffLine(want, got string) (int, string, string) {
	ws, gs := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range max(len(ws), len(gs)) {
		var w, g string
		if i < len(ws) {
			w = ws[i]
		}
		if i < len(gs) {
			g = gs[i]
		}
		if w != g {
			return i + 1, w, g
		}
	}
	return 0, "", ""
}

func Card(t *testing.T, id, subject string) lineup.Card {
	t.Helper()
	c, err := lineup.Propose(lineup.Card{ID: id, Slot: lineup.LocationReport, Subject: subject, Headline: subject, State: lineup.Proposed})
	if err != nil {
		t.Fatalf("proposing %s: %v", id, err)
	}
	// The lineup holds ADMITTED cards only — its own invariant, and it refuses
	// a proposed one. The fixture follows the real lifecycle rather than
	// side-stepping it.
	c, err = c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting %s: %v", id, err)
	}
	return c
}

// MastheadSnap is a snapshot with providers, so the API summary has something
// to count.
func MastheadSnap() *snapshot.Snapshot {
	at := time.Date(2026, 8, 28, 19, 15, 0, 0, time.UTC)
	return &snapshot.Snapshot{Providers: []snapshot.ProviderStatus{
		{ID: "a", Status: snapshot.ProviderOK, FetchedAt: at},
		{ID: "b", Status: snapshot.ProviderOK, FetchedAt: at},
	}}
}

// LoadedPoolSize is the pool's own cap, stated here because `modes/tty` may not
// import `domains/locations` (make lint-imports) — the same tie the service
// radius bounds already carry.
const LoadedPoolSize = 25

// MastheadMoved reports whether a change moved a surface's masthead, and fails
// the test when it did and the key did not. The keys are the kit's, compared
// with ==, so a test outside the kit need not name their type (design CK-4).
func MastheadMoved(t *testing.T, surface, field string, before, after string, k0, k1 any) bool {
	t.Helper()
	if before == after {
		return false
	}
	if k0 == k1 {
		t.Errorf("%s: changing %s changes the masthead and NOT mastheadKey: the masthead memo (internal/frame's masthead.go, MastheadSlot) will draw the old masthead; add what %s changes to MastheadKeyOf", surface, field, field)
	}
	return true
}
