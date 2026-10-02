package agememo

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// clock is a test's clock, moved by hand.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newClock() *clock { return &clock{now: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)} }

// A VALUE ANSWERS WHILE IT IS FRESH, and is asked again once it is not.
func TestAValueAnswersWhileFresh(t *testing.T) {
	c := newClock()
	m := New[string, int](Options{Fresh: time.Minute, Now: c.Now})
	asked := 0
	fetch := func() (int, error) { asked++; return asked, nil }
	for range 3 {
		if v, err := m.Do(context.Background(), "k", fetch); v != 1 || err != nil {
			t.Fatalf("a fresh value: %d %v", v, err)
		}
	}
	if v, ok := m.Get("k"); !ok || v != 1 {
		t.Errorf("Get while fresh: %d %v", v, ok)
	}
	c.add(time.Minute)
	if _, ok := m.Get("k"); ok {
		t.Error("Get answered a value past its freshness")
	}
	if v, _ := m.Do(context.Background(), "k", fetch); v != 2 || asked != 2 {
		t.Errorf("past its freshness: %d after %d asks; want it asked again", v, asked)
	}
}

// ONE FETCH A KEY AT A TIME: callers that ask together share the one fetch.
func TestCallersAskingTogetherShareOneFetch(t *testing.T) {
	m := New[string, int](Options{Fresh: time.Minute})
	var asked atomic.Int32
	release := make(chan struct{})
	fetch := func() (int, error) {
		asked.Add(1)
		<-release
		return 7, nil
	}
	var wg sync.WaitGroup
	var got [8]int
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i], _ = m.Do(context.Background(), "k", fetch)
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if asked.Load() != 1 {
		t.Errorf("%d fetches for eight callers asking together; want one", asked.Load())
	}
	for _, v := range got {
		if v != 7 {
			t.Fatalf("a caller got %d; want the one fetch's 7", v)
		}
	}
}

// A FAILED REFRESH IS ANSWERED BY THE LAST VALUE while that value is younger
// than StandIn, and by the failure past it; a failure is never kept.
func TestAFailedRefreshStandsOnTheLastValue(t *testing.T) {
	c := newClock()
	m := New[string, int](Options{Fresh: time.Minute, StandIn: 10 * time.Minute, Now: c.Now})
	if _, err := m.Do(context.Background(), "k", func() (int, error) { return 5, nil }); err != nil {
		t.Fatal(err)
	}
	fail := errors.New("down")
	asked := 0
	failing := func() (int, error) { asked++; return 0, fail }
	c.add(5 * time.Minute)
	if v, err := m.Do(context.Background(), "k", failing); v != 5 || err != nil {
		t.Errorf("a failed refresh within StandIn: %d %v; want the last value", v, err)
	}
	if v, err := m.Do(context.Background(), "k", failing); v != 5 || err != nil || asked != 2 {
		t.Errorf("the failure was kept: %d %v after %d asks; want it asked again", v, err, asked)
	}
	c.add(10 * time.Minute)
	if _, err := m.Do(context.Background(), "k", failing); !errors.Is(err, fail) {
		t.Errorf("a failed refresh past StandIn: %v; want the failure", err)
	}
	none := New[string, int](Options{Fresh: time.Minute})
	if _, err := none.Do(context.Background(), "k", failing); !errors.Is(err, fail) {
		t.Errorf("a first fetch that fails: %v; want the failure", err)
	}
}

// IT HOLDS AT MOST MAX KEYS, the least recently used out; one when unset.
func TestItHoldsAtMostMaxKeys(t *testing.T) {
	c := newClock()
	m := New[string, int](Options{Fresh: time.Hour, Max: 2, Now: c.Now})
	put := func(k string, v int) {
		c.add(time.Second)
		_, _ = m.Do(context.Background(), k, func() (int, error) { return v, nil })
	}
	put("a", 1)
	put("b", 2)
	c.add(time.Second)
	_, _ = m.Get("a") // a used: b is the least recently used
	put("c", 3)
	if _, ok := m.Get("b"); ok {
		t.Error("the least recently used key stayed past Max")
	}
	if _, ok := m.Get("a"); !ok {
		t.Error("a key used since was dropped")
	}
	one := New[string, int](Options{Fresh: time.Hour})
	_, _ = one.Do(context.Background(), "x", func() (int, error) { return 1, nil })
	_, _ = one.Do(context.Background(), "y", func() (int, error) { return 2, nil })
	if _, ok := one.Get("x"); ok {
		t.Error("Max unset held two keys; want one")
	}
}

// LAST IS A KEY'S VALUE WHATEVER ITS AGE, with when it was stored; Forget
// drops it; Put stores without fetching.
func TestLastPutAndForget(t *testing.T) {
	c := newClock()
	m := New[string, int](Options{Fresh: time.Minute, Max: 4, Now: c.Now})
	if _, _, ok := m.Last("k"); ok {
		t.Fatal("a key never stored has a last value")
	}
	m.Put("k", 9)
	c.add(time.Hour)
	if v, at, ok := m.Last("k"); !ok || v != 9 || !at.Equal(c.Now().Add(-time.Hour)) {
		t.Errorf("Last an hour on: %d at %v (%v)", v, at, ok)
	}
	m.Forget("k")
	if _, _, ok := m.Last("k"); ok {
		t.Error("a forgotten key has a last value")
	}
}

// A FETCH THAT PANICS LETS ITS WAITING CALLERS GO with an error, never a zero
// value passed off as an answer.
func TestAPanickingFetchLetsItsCallersGo(t *testing.T) {
	m := New[string, int](Options{Fresh: time.Minute})
	started, release := make(chan struct{}), make(chan struct{})
	go func() {
		defer func() { _ = recover() }()
		_, _ = m.Do(context.Background(), "k", func() (int, error) { close(started); <-release; panic("boom") })
	}()
	<-started
	done := make(chan error, 1)
	go func() { _, err := m.Do(context.Background(), "k", func() (int, error) { return 1, nil }); done <- err }()
	time.Sleep(20 * time.Millisecond)
	close(release)
	select {
	case err := <-done:
		if err == nil {
			t.Error("a caller waiting on a panicked fetch got no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a caller waiting on a panicked fetch was never let go")
	}
}

// A CALLER WAITING ON ANOTHER'S FETCH GIVES UP WHEN ITS OWN CONTEXT ENDS: a
// slow source never holds a caller that has stopped wanting the answer.
func TestAWaitingCallerGivesUpWithItsContext(t *testing.T) {
	m := New[string, int](Options{Fresh: time.Minute})
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	go func() {
		_, _ = m.Do(context.Background(), "k", func() (int, error) { close(started); <-release; return 1, nil })
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := m.Do(ctx, "k", func() (int, error) { return 2, nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a waiting caller whose context ended: %v; want its context's end", err)
	}
}

// DO REFUSES WHAT IT CANNOT HONOUR, with an error rather than a panic or a
// rule that never applies: no fetch to ask, or a stand-in shorter than the
// freshness - a value is only asked again once it is older than Fresh, so a
// StandIn under it could never stand in.
func TestDoRefusesWhatItCannotHonour(t *testing.T) {
	m := New[string, int](Options{Fresh: time.Minute})
	if _, err := m.Do(context.Background(), "k", nil); err == nil {
		t.Error("a nil fetch was taken")
	}
	short := New[string, int](Options{Fresh: time.Hour, StandIn: time.Minute})
	if _, err := short.Do(context.Background(), "k", func() (int, error) { return 1, nil }); err == nil {
		t.Error("a StandIn shorter than Fresh was taken")
	}
}
