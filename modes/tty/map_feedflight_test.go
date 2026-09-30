package tty

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// feedAsks counts the feed answers a command's messages carry: each is one
// ask the command made.
func feedAsks(t *testing.T, cmd tea.Cmd) []mapFeedMsg {
	t.Helper()
	var out []mapFeedMsg
	for _, m := range msgsOf(t, cmd) {
		if f, ok := m.(mapFeedMsg); ok {
			out = append(out, f)
		}
	}
	return out
}

// TestAFeedAskIsOneAtATime is D-157: with an ask in flight, new data asks
// nothing more - it marks the feed wanted again - and when the answer lands it
// is drawn, and one fresh ask follows; then no more. Before, every snapshot
// asked anew and dropped the answer it was waiting for, so under a feed slower
// than the snapshots nothing was ever drawn (W14's C-1).
func TestAFeedAskIsOneAtATime(t *testing.T) {
	asked := 0
	d := mapDash(t, Config{MapFeed: squareFeed(&asked, "Wind Warning")})
	d, open := pressKey(d, "g")
	for range 3 { // three publishes while the opening's ask is in flight
		m, cmd := d.Update(SnapshotMsg{Snap: placedSnap()})
		d = m.(Dashboard)
		if n := len(feedAsks(t, cmd)); n != 0 {
			t.Fatalf("new data asked the feed %d times with an ask in flight", n)
		}
	}
	answers := feedAsks(t, open)
	if len(answers) != 1 {
		t.Fatalf("the opening asked %d times, want once", len(answers))
	}
	m, cmd := d.Update(answers[0])
	d = m.(Dashboard)
	if _, drawn := d.mapPane.given["alert/Wind Warning"]; !drawn {
		t.Error("the answer for the place and view in view was not drawn")
	}
	again := feedAsks(t, cmd)
	if len(again) != 1 {
		t.Fatalf("after the answer, %d asks followed; want the one the new data wanted", len(again))
	}
	m, cmd = d.Update(again[0])
	d = m.(Dashboard)
	if n := len(feedAsks(t, cmd)); n != 0 {
		t.Errorf("with nothing new wanted, the answer asked again %d times", n)
	}
}

// TestAStaleFeedAskIsCancelled is D-157's other half: a move makes the ask in
// flight stale, so its settle tick cancels it and asks for the new view; the
// stale ask's answer, landing late, is never drawn.
func TestAStaleFeedAskIsCancelled(t *testing.T) {
	cancelled := make(chan struct{}, 4)
	first := true
	feed := func(ctx context.Context, _ MapAsk) MapFeed {
		if first { // the opening's ask: hangs until cancelled
			first = false
			<-ctx.Done()
			cancelled <- struct{}{}
			return MapFeed{Notes: []string{"stale"}}
		}
		return MapFeed{Notes: []string{"fresh"}}
	}
	d := mapDash(t, Config{MapFeed: feed})
	d, open := pressKey(d, "g")
	stale := make(chan tea.Msg, 1)
	go runEach(open, stale) // the opening's commands, each on its own goroutine as Bubble Tea runs them
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	m, cmd := d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	d = m.(Dashboard)
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("the move's settle did not cancel the stale ask")
	}
	m, _ = d.Update(<-stale)
	d = m.(Dashboard)
	for _, n := range d.mapPane.notes {
		if n == "stale" {
			t.Error("the cancelled ask's answer was drawn")
		}
	}
	fresh := feedAsks(t, cmd)
	if len(fresh) != 1 {
		t.Fatalf("the settled view asked %d times, want once", len(fresh))
	}
	m, _ = d.Update(fresh[0])
	d = m.(Dashboard)
	if len(d.mapPane.notes) != 1 || d.mapPane.notes[0] != "fresh" {
		t.Errorf("the new view's answer was not drawn: %v", d.mapPane.notes)
	}
}

// runEach runs a command as Bubble Tea does - a batch's commands each on its
// own goroutine - and sends any feed answer on out. A timer's command just
// sleeps on its goroutine.
func runEach(cmd tea.Cmd, out chan<- tea.Msg) {
	if cmd == nil {
		return
	}
	switch m := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range m {
			go runEach(c, out)
		}
	case mapFeedMsg:
		out <- m
	}
}

// TestAMoveAsksTheFeedOnceWhenItSettles is W14's P-2, D-66 as ruled: "never on
// every key". A pan asks nothing; its settle tick asks once. Before, every
// pan, zoom and region key asked the whole feed at once and the settle tick
// asked it again.
func TestAMoveAsksTheFeedOnceWhenItSettles(t *testing.T) {
	asked := 0
	d := mapDash(t, Config{MapFeed: squareFeed(&asked, "Wind Warning")})
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyRight}, {Code: '+', Text: "+"}, {Code: '1', Text: "1"}} {
		m, cmd := d.Update(key)
		d = m.(Dashboard)
		if n := len(feedAsks(t, cmd)); n != 0 {
			t.Errorf("%s asked the feed %d times before the view settled", key.String(), n)
		}
		m, cmd = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
		d = m.(Dashboard)
		asks := feedAsks(t, cmd)
		if len(asks) != 1 {
			t.Errorf("%s's settle asked %d times, want once", key.String(), len(asks))
			continue
		}
		m, _ = d.Update(asks[0])
		d = m.(Dashboard)
	}
}

// TestAnAnswerForAViewLeftIsNotDrawn: a pan moves the view while the ask
// runs; its answer, landing before the pan's settle tick, is for a view no
// longer shown and is not drawn - the settle tick asks for the new one.
func TestAnAnswerForAViewLeftIsNotDrawn(t *testing.T) {
	calls := 0
	feed := func(_ context.Context, _ MapAsk) MapFeed {
		calls++
		return MapFeed{Notes: []string{fmt.Sprint("answer ", calls)}}
	}
	d := mapDash(t, Config{MapFeed: feed})
	d, open := pressKey(d, "g")
	answers := feedAsks(t, open) // asked for the view as it opened
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	m, _ = d.Update(answers[0]) // lands after the pan, before its settle tick
	d = m.(Dashboard)
	for _, n := range d.mapPane.notes {
		if n == "answer 1" {
			t.Error("the answer for the view the pan left was drawn")
		}
	}
	m, cmd := d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	d = m.(Dashboard)
	if n := len(feedAsks(t, cmd)); n != 1 {
		t.Errorf("the settled view asked %d times, want once", n)
	}
}

// TestAnAnswerIsDrawnWhenOnlyTheMapsSizeChanged is the regression the first
// re-measure found (W14, 2 of 5 cold opens drew no alerts in 45 s): the view
// an answer is for was keyed on the view's box, and the box follows the map's
// size, which shrinks as notes and the description fill in - no move at all.
// The answer was dropped and nothing asked again. A view is the listener's:
// a move or a resize, which is viewGen.
func TestAnAnswerIsDrawnWhenOnlyTheMapsSizeChanged(t *testing.T) {
	asked := 0
	d := mapDash(t, Config{MapFeed: squareFeed(&asked, "Wind Warning")})
	d, open := pressKey(d, "g")
	answers := feedAsks(t, open)
	before := d.mapAsk().View
	d.mapPane.notes = []string{"a note", "another", "and a third, as the station's data fills in"}
	if d.mapAsk().View == before {
		t.Fatal("control: the notes did not change the map's size, so this proves nothing")
	}
	m, _ := d.Update(answers[0])
	d = m.(Dashboard)
	if _, drawn := d.mapPane.given["alert/Wind Warning"]; !drawn {
		t.Error("an answer was dropped because the map's size changed, with no move")
	}
}

// TestTheAskSaysWhatIsSwitchedOff is the dashboard's half of W14's P-3 and
// D-160: the feed's ask carries the Fire and Quakes rows and the alert
// switches as the listener set them, so what is off is never fetched.
func TestTheAskSaysWhatIsSwitchedOff(t *testing.T) {
	d := mapDash(t, Config{MapLayers: []MapLayer{{Key: AlertLayer, On: true}, {Key: FireLayer, On: true}, {Key: QuakeLayer, On: true}}})
	if a := d.mapAsk(); !a.Fire || !a.Quakes || a.AlertsOff || len(a.AlertCategoriesOff) != 0 {
		t.Fatalf("with everything on the ask says %+v", a)
	}
	off := AlertCategories()[0].Key
	d.mapLayerChoice = layerChoiceKey(map[string]bool{FireLayer: false, QuakeLayer: false, categoryChoice(off): false})
	a := d.mapAsk()
	if a.Fire || a.Quakes || len(a.AlertCategoriesOff) != 1 || a.AlertCategoriesOff[0] != off {
		t.Errorf("with Fire, Quakes and %s off the ask says Fire %v Quakes %v categories off %v", off, a.Fire, a.Quakes, a.AlertCategoriesOff)
	}
	d.mapLayerChoice = layerChoiceKey(map[string]bool{AlertLayer: false})
	if !d.mapAsk().AlertsOff {
		t.Error("with the Alert areas layer off the ask does not say so")
	}
}
