package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func warmupDeck(t *testing.T, seen *seenStore, evs []globalfeed.Event) *tickerDeck {
	t.Helper()
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	return &tickerDeck{
		send:    func(tea.Msg) {},
		voice:   nar,
		sources: []globalfeed.Source{fakeSource{"NWS", evs}},
		watch:   func() []snapshot.LocationRef { return nil },
		seen:    seen,
		muted:   &atomic.Bool{},
		radius:  &atomic.Int64{}, // 0 = All
	}
}

// C-4 — THE LAUNCH SEED MUST NOT SWALLOW THE DOWN-TIME WINDOW.
//
// The first cycle of every launch marked every active event seen and returned,
// to avoid a launch storm. But the PERSISTENT seen store already prevents
// re-announcing across a restart, so the blanket seed only ever suppressed the
// alerts that appeared while the app was NOT RUNNING — the one set a returning
// listener has not heard.
//
// The scenario: the lid closes at 2pm, a tornado warning is issued at 2:40, the
// app is relaunched at 3:00. The warning is inside its active window, inside the
// radius, and has never been in the seen store. It appeared on the tape and was
// never spoken, and unread/Merge filtered it for ever after.
func TestAnAlertIssuedWhileTheAppWasClosedIsStillAnnounced(t *testing.T) {
	dir := t.TempDir()
	// AN EARLIER SESSION, so this is a returning listener and not a first run.
	prior := loadSeen(dir, tickerSeenWindow)
	prior.mark([]globalfeed.Event{{ID: "heard-yesterday"}}, time.Now().Add(-3*time.Hour))
	prior.save()

	seen := loadSeen(dir, tickerSeenWindow) // the relaunch reads it back
	if !seen.set()["heard-yesterday"] {
		t.Fatal("the fixture's earlier session did not persist; this test would pose a first run")
	}

	// ISSUED WHILE THE APP WAS CLOSED.
	away := globalfeed.Event{
		ID: "while-away", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "the Oklahoma City area", Severity: globalfeed.SevRed,
		At: time.Now().Add(-20 * time.Minute),
	}
	d := warmupDeck(t, seen, []globalfeed.Event{away})
	st := newStation(t, d)
	d.cycle(context.Background())

	// THE PRODUCER'S OWN DECISION, asserted before anything drains — the same
	// discipline the mute gate needed. Marking it seen and returning is
	// invisible downstream: the rail simply reads empty.
	if st.pendingCount() == 0 {
		t.Error("the first cycle after a relaunch swallowed a warning issued while the app was closed: " +
			"it is on the tape, it is never spoken, and unread filters it for ever")
	}
	if seen.set()["while-away"] {
		t.Error("the warning was marked read aloud without being read aloud")
	}
}

// THE CONTROL, and the property the seed exists for: a GENUINE first run — no
// store at all — still starts quietly rather than reading every active hazard
// in the country at a listener who just opened the app.
func TestAGenuineFirstRunStillStartsQuietly(t *testing.T) {
	seen := loadSeen(t.TempDir(), tickerSeenWindow)
	if len(seen.set()) != 0 {
		t.Fatal("the fixture is not a first run")
	}
	evs := []globalfeed.Event{
		{ID: "a", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Severity: globalfeed.SevRed, At: time.Now()},
		{ID: "b", Class: globalfeed.ClassSevereWx, Type: "Severe Thunderstorm Warning", Severity: globalfeed.SevOrange, At: time.Now()},
	}
	d := warmupDeck(t, seen, evs)
	st := newStation(t, d)
	d.cycle(context.Background())
	if st.pendingCount() != 0 {
		t.Errorf("a first run offered %d burst(s); it seeds quietly so a new listener is not met with a launch storm", st.pendingCount())
	}
	if len(seen.set()) != 2 {
		t.Errorf("a first run seeds every active event, got %d of 2", len(seen.set()))
	}
}
