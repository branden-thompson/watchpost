package tty

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// seam is the Propagation update in a test: how many times it was asked,
// the contexts it was given, and what it answers.
type seam struct {
	mu     sync.Mutex
	asked  int
	ctxs   []context.Context
	answer PropagationResult
	wait   bool // when set, it answers only once its context ends
}

func (s *seam) update(ctx context.Context) <-chan PropagationResult {
	s.mu.Lock()
	s.asked++
	s.ctxs = append(s.ctxs, ctx)
	wait, answer := s.wait, s.answer
	s.mu.Unlock()
	out := make(chan PropagationResult, 1)
	if wait {
		go func() { <-ctx.Done(); out <- PropagationResult{Err: ctx.Err()} }()
		return out
	}
	out <- answer
	return out
}

func (s *seam) count() int { s.mu.Lock(); defer s.mu.Unlock(); return s.asked }

// propDash is the map open on Oceanside with the Propagation update seam, the
// acknowledgement seen or not.
func propDash(t *testing.T, s *seam, seen bool) Dashboard {
	t.Helper()
	ack := 0
	if seen {
		ack = propAckVersion
	}
	return openMap(t, Config{PropagationUpdate: s.update, PropagationAck: ack, SavePropagationAck: func(int) error { return nil }}, 133, 44)
}

// keyCmd presses a key and returns the model and the command it asked for,
// unrun.
func keyCmd(t *testing.T, d Dashboard, key string) (Dashboard, tea.Cmd) {
	t.Helper()
	m, cmd := d.Update(mapKeyMsg(t, key))
	return m.(Dashboard), cmd
}

// runProp runs a command and feeds every Propagation result it carries back
// through Update; ticks and other waits are not run.
func runProp(t *testing.T, d Dashboard, cmd tea.Cmd) Dashboard {
	t.Helper()
	for _, msg := range propMsgs(cmd) {
		m, _ := d.Update(msg)
		d = m.(Dashboard)
	}
	return d
}

// propMsgs runs a command's batch, keeping only the Propagation results; a
// command that would wait (a tick) is never run.
func propMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if b, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range b {
				out = append(out, propMsgs(c)...)
			}
			return out
		}
		if _, ok := msg.(propUpdatedMsg); ok {
			return []tea.Msg{msg}
		}
	case <-time.After(200 * time.Millisecond): // a tick, or a wait on the network: not this test's
	}
	return nil
}

// TestThePropagationModeFetchesNothingUntilOpened is FR-4.2 (D-76): the map
// in its weather modes, its keys pressed, asks nothing; P asks once.
func TestThePropagationModeFetchesNothingUntilOpened(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d := propDash(t, s, true)
	for _, k := range []string{"R", "R", "O", "esc", "left", "+"} {
		var cmd tea.Cmd
		d, cmd = keyCmd(t, d, k)
		d = runProp(t, d, cmd)
	}
	if s.count() != 0 {
		t.Fatalf("the weather modes asked the Propagation update %d times", s.count())
	}
	d, cmd := keyCmd(t, d, "P")
	if runProp(t, d, cmd); s.count() != 1 {
		t.Errorf("P asked the update %d times; want once", s.count())
	}
}

// TestNothingIsFetchedBeforeTheAcknowledgement is FR-4.2 and D-81: while the
// acknowledgement is to be closed nothing is asked; closing it asks.
func TestNothingIsFetchedBeforeTheAcknowledgement(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d, cmd := keyCmd(t, propDash(t, s, false), "P")
	if d = runProp(t, d, cmd); d.modal != modalPropAck || s.count() != 0 {
		t.Fatalf("with the acknowledgement up (window %d) the update was asked %d times", d.modal, s.count())
	}
	d, cmd = keyCmd(t, d, "enter")
	if d = runProp(t, d, cmd); s.count() != 1 {
		t.Errorf("closing the acknowledgement asked the update %d times; want once", s.count())
	}
}

// TestTheUpdateNeverRunsOnTheUIGoroutine is FR-4.8: the key press that asks
// for an update returns without calling the seam; only its command does.
func TestTheUpdateNeverRunsOnTheUIGoroutine(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	if s.count() != 0 {
		t.Fatalf("Update called the seam %d times itself", s.count())
	}
	if runProp(t, d, cmd); s.count() != 1 {
		t.Errorf("the command called the seam %d times", s.count())
	}
}

// TestAnUpdatesResultIsSaid is FR-5.1 for UAT-1: the snapshot's backgrounds
// and time are said; an update with no field says the library's words, which
// name when it asks again (D-124).
func TestAnUpdatesResultIsSaid(t *testing.T) {
	at := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	snap := ionomaps.Snapshot{Computed: at, Hours: []ionomaps.Hour{{At: at}},
		Background: ionomaps.Backgrounds{FoF2: ionomaps.GloTEC, M3000: ionomaps.Climatology},
		Inputs:     ionomaps.Inputs{GloTECValid: at.Add(-30 * time.Minute)}}
	s := &seam{answer: PropagationResult{Snapshot: snap}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	status := stripANSITest(d.mapStatusLine())
	for _, want := range []string{"GloTEC", "11:30 UTC", "climatology", "12:00 UTC"} {
		if !strings.Contains(status, want) {
			t.Errorf("the status %q does not say %q", status, want)
		}
	}
	if !strings.Contains(strings.Join(d.describeLinesAll(), " "), "GloTEC") {
		t.Errorf("the words do not say the snapshot: %q", d.describeLinesAll())
	}
	failed := &seam{answer: PropagationResult{Err: errors.New("ionomaps: no field yet: it is asked again at the first update after 12:01:00 UTC")}}
	d, cmd = keyCmd(t, propDash(t, failed, true), "P")
	if d = runProp(t, d, cmd); !strings.Contains(stripANSITest(d.mapStatusLine()), "asked again") {
		t.Errorf("an update with no field says %q", stripANSITest(d.mapStatusLine()))
	}
}

// TestTheModeRefreshesWhileOpen is FR-4.10 (D-94): the refresh tick asks
// again while the mode is open, and a tick from before the mode was left
// asks nothing.
func TestTheModeRefreshesWhileOpen(t *testing.T) {
	s := &seam{answer: PropagationResult{Err: errors.New("offline")}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	m, cmd := d.Update(propTickMsg{gen: d.mapPane.propGen})
	if d = runProp(t, m.(Dashboard), cmd); s.count() != 2 {
		t.Fatalf("the refresh tick asked %d times in all; want twice", s.count())
	}
	stale := d.mapPane.propGen
	d, _ = keyCmd(t, d, "P") // left
	m, cmd = d.Update(propTickMsg{gen: stale})
	if runProp(t, m.(Dashboard), cmd); s.count() != 2 {
		t.Errorf("a tick from before the mode was left asked again (%d)", s.count())
	}
	d, cmd = keyCmd(t, d, "P") // a new session, its update answered: none in flight
	if d = runProp(t, d, cmd); s.count() != 3 || d.mapPane.propBusy {
		t.Fatalf("re-entering asked %d in all, busy %v", s.count(), d.mapPane.propBusy)
	}
	m, cmd = d.Update(propTickMsg{gen: stale})
	if runProp(t, m.(Dashboard), cmd); s.count() != 3 {
		t.Errorf("an old session's tick asked again in the new one (%d)", s.count())
	}
}

// TestLeavingTheModeCancelsItsUpdate is FR-4.2: leaving the Propagation mode
// ends the update it started; a key press inside the mode does not.
func TestLeavingTheModeCancelsItsUpdate(t *testing.T) {
	s := &seam{wait: true}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	go runEvery(cmd)
	deadline := time.Now().Add(2 * time.Second)
	for s.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.count() != 1 {
		t.Fatal("the update was never asked")
	}
	ctx := s.ctxs[0]
	d, _ = keyCmd(t, d, "left")
	if ctx.Err() != nil {
		t.Fatal("a pan cancelled the update")
	}
	keyCmd(t, d, "P")
	if ctx.Err() == nil {
		t.Error("leaving the mode left its update running")
	}
}

// runEvery runs a command and every command of a batch it returns, each in
// its own goroutine, dropping their messages.
func runEvery(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if b, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range b {
			go runEvery(c)
		}
	}
}
