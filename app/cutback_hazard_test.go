package app

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// cutback_hazard_test.go is F-158's owed test (0.19.6 B1.1, design AP-7.1,
// RM-4): the operator's cut-back while a hazard is being read. The Director
// answers the cut-back with Restore, since nothing it holds is carried any
// more; the bed stays down only because the Restore executor goes through the
// arbiter, which gives nothing back while a job is on the air. Its reach is
// the deck's alone, so it moves intact into app/internal/deck.

// hazardHeldVoice is a script voice whose first line waits, on the air, until
// the test releases it, and says when it is there.
type hazardHeldVoice struct {
	scriptVoice
	once    sync.Once
	onAir   chan struct{}
	release chan struct{}
}

func (v *hazardHeldVoice) render(ctx context.Context, r cast.Role, t string) (clip, bool) {
	v.once.Do(func() {
		close(v.onAir)
		select {
		case <-v.release:
		case <-ctx.Done():
		}
	})
	return v.scriptVoice.render(ctx, r, t)
}

// TestACutBackUnderAHazardReadKeepsTheBedDown is F-158's test (0.19.6 B1.1,
// design AP-7.1). On the shipping path, the main track off, the station is on
// the air with the bed carrying the programme when a Tornado Warning reaches
// the rail; while it is read the operator cuts back, as `[b]` does, through
// MasterControl's CutBed. The bed stays down until the read ends, and then
// comes back by the ordinary path. It pins today's safe outcome, not the
// hazard. Seen failing under mF158, a Restore executor that lifts the bed
// without the arbiter.
func TestACutBackUnderAHazardReadKeepsTheBedDown(t *testing.T) {
	t.Setenv("WATCHPOST_MAINTRACK", "")
	if err := os.Unsetenv("WATCHPOST_MAINTRACK"); err != nil { // the shipping path: unset, so mainTrack() is off
		t.Fatal(err)
	}
	v := &hazardHeldVoice{onAir: make(chan struct{}), release: make(chan struct{})}
	nar := testArbiter(v, nil)
	nar.sleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }
	deck := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: nar, seen: loadSeen(t.TempDir(), time.Hour)}
	s := newStation(t, deck)
	mc := deck.mc
	mc.mu.Lock()
	mc.carry = s.queue // MasterControl declares to the station, as startSchedule wires it
	mc.mu.Unlock()
	ctx := context.Background()

	mc.GoOnAir()    // the console takes the air and the station runs
	mc.CutBed(true) // the bed carries the programme
	s.drain(ctx)

	s.offer([]globalfeed.Event{{ID: "tor", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "the Oklahoma City area", Severity: globalfeed.SevRed, At: time.Now().Add(-5 * time.Minute)}})
	read := make(chan struct{})
	go func() { // the read holds its executor, as a pump lane does, while the Director steps on
		defer close(read)
		s.drain(ctx)
	}()
	select {
	case <-v.onAir:
	case <-time.After(10 * time.Second):
		t.Fatal("the Tornado Warning never reached the air")
	}
	if !mc.givenWay() {
		t.Fatal("COULD NOT RUN: the bed was not down under the hazard's read before the cut-back")
	}

	mc.CutBed(false) // the operator's [b]: back to the programme
	s.mu.Lock()
	cut := s.pending
	s.pending = nil
	s.mu.Unlock()
	var fx []lineup.Effect
	for _, ev := range cut { // bounded by the cut-back's events (P10-02)
		var f []lineup.Effect
		s.dir, f = s.dir.Step(ev)
		fx = append(fx, f...)
	}
	said := make([]string, 0, len(fx))
	for _, f := range fx { // bounded by the step's effects (P10-02)
		said = append(said, lineup.Describe(f))
	}
	if !strings.Contains(strings.Join(said, "; "), lineup.Describe(lineup.Restore{})) {
		t.Fatalf("no longer exercises F-158: the Director answered the cut-back under a hazard's read with [%s], and no Restore; re-author this test for the step that replaced it (0.19.7, D-2)", strings.Join(said, "; "))
	}
	var evs []lineup.Event
	for _, f := range fx { // bounded by the step's effects (P10-02)
		evs = append(evs, s.x.run(ctx, f)...)
	}
	s.run(ctx, evs...)
	if !mc.givenWay() {
		t.Error("the cut-back's Restore lifted the bed while the Tornado Warning is still being read: the programme comes back up over the hazard (F-158)")
	}

	close(v.release)
	select {
	case <-read:
	case <-time.After(10 * time.Second):
		t.Fatal("the Tornado Warning's read never ended")
	}
	if mc.givenWay() {
		t.Error("the bed is still down after the hazard's read ended: it does not come back by the ordinary path")
	}
}
