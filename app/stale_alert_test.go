package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// I-8 — AN ALERT THAT EXPIRED BETWEEN ARRIVING AND AIRING IS NOT READ AS LIVE.
//
// The producer's record is snapshotted at arrival; the card is composed later.
// Nothing re-asked whether the alert was still active, so a warning that lapsed
// in between was spoken as current — with its own "until" time in the sentence,
// which is a station saying something false with confidence.
func TestAnAlertThatExpiredBeforeItAiredIsNotSpoken(t *testing.T) {
	nar := testDirector(&scriptVoice{}, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
	st := newStation(t, deck)

	// Live when it arrives, so the burst is planned and admitted...
	ev := globalfeed.Event{
		ID: "lapsing", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
		Location: "the Oklahoma City area", Severity: globalfeed.SevRed,
		At: time.Now().Add(-30 * time.Minute), Until: time.Now().Add(50 * time.Millisecond),
	}
	st.offer([]globalfeed.Event{ev})
	if st.pendingCount() == 0 {
		t.Fatal("the fixture never reached the Director while it was live; it poses nothing")
	}
	// ...and lapsed by the time the card is composed.
	time.Sleep(80 * time.Millisecond)
	st.drain(context.Background())

	if deck.seen.set()["lapsing"] {
		t.Error("an expired warning was read aloud as current, with its original expiry in the sentence")
	}
}

// THE CONTROL: an alert with no expiry — a quake's instant — must still read,
// and one still inside its window must too. Without this the check above could
// be satisfied by declining everything.
func TestALiveAlertAndAnAlertWithNoExpiryStillRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		until time.Time
	}{
		{"still inside its window", time.Now().Add(time.Hour)},
		{"no expiry at all", time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nar := testDirector(&scriptVoice{}, nil)
			nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
			deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
			ev := globalfeed.Event{
				ID: "live", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning",
				Location: "the Oklahoma City area", Severity: globalfeed.SevRed,
				At: time.Now().Add(-5 * time.Minute), Until: tc.until,
			}
			newStation(t, deck).takeover(context.Background(), []globalfeed.Event{ev})
			if !deck.seen.set()["live"] {
				t.Error("a live alert was declined; the staleness check is refusing what it should pass")
			}
		})
	}
}

// I-9 — A DIRECTOR THAT REFUSED TO BE BUILT IS NOT A SCHEDULE.
//
// lineup.New returns a zero Director on a zero clock or a negative Max, and a
// zero Director refuses every event for ever with nothing observable from
// outside: Step fails its clock invariant and returns unchanged, and Now()
// fails quiet too. The pump checked its two closures and not the thing it was
// built to drive.
func TestAPumpRefusesADirectorThatWasNeverBuilt(t *testing.T) {
	run := func(context.Context, lineup.Effect) []lineup.Event { return nil }
	onFault := func(lineup.Effect, any) {}
	if p := newPump(lineup.New(lineup.Settings{Max: 5}, time.Time{}), run, onFault); p != nil {
		t.Error("a pump was built over a Director with no clock: it would tick for ever and schedule nothing")
	}
	if p := newPump(lineup.New(lineup.Settings{Max: -1}, time.Now()), run, onFault); p != nil {
		t.Error("a pump was built over a Director refused for a negative Max")
	}
	// THE CONTROL, or the two above prove only that newPump can return nil.
	if p := newPump(lineup.New(lineup.Settings{Max: 5}, time.Now()), run, onFault); p == nil {
		t.Fatal("a well-formed Director was refused; the checks above are measuring the wrong thing")
	}
}
