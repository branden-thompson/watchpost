package tty

// map_work.go — the map's commands and its clock (0.18.0 W2.2, W2.6).
//
// THE LIBRARY IS CALLED FROM UPDATE, and from exactly two commands: Work,
// which may block on a tile, and the feed, which asks the app (C-7, FR-3.3).
// Both are joined when the map closes (FR-8.2): the close cancels what is
// running, waits for it, and a command that starts after the close touches
// nothing. The map's clock is one tick kept outstanding at the library's
// NextCall - a marker's phase, a tile's retry, an overlay going stale - armed
// after every Update, so no path that draws has to remember it.

import (
	"bytes"
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

// mapJoinLimit bounds how long a close waits for the map's commands. They
// are cancelled first, so this is only reached by a call that ignores it.
const mapJoinLimit = 2 * time.Second

// mapTickFloor is the least delay a tick is armed with, so a time the
// library still wants after its draw cannot spin the loop.
const mapTickFloor = 50 * time.Millisecond

// mapWorkers are the commands running for one map: begun under the lock, so
// a close either sees a command and waits for it, or the command sees the
// close and does nothing.
type mapWorkers struct {
	mu     sync.Mutex
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newMapWorkers() *mapWorkers {
	ctx, cancel := context.WithCancel(context.Background())
	return &mapWorkers{ctx: ctx, cancel: cancel}
}

// begin admits one command, with the context it runs under, or refuses it
// once the map is closed. A command admitted calls done.
func (w *mapWorkers) begin() (context.Context, context.CancelFunc, bool) {
	if w == nil {
		return nil, nil, false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil, nil, false
	}
	w.wg.Add(1)
	ctx, cancel := context.WithTimeout(w.ctx, mapWorkLimit)
	return ctx, func() { cancel(); w.wg.Done() }, true
}

// close cancels every command and waits for them, up to mapJoinLimit.
func (w *mapWorkers) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.closed = true
	w.cancel()
	w.mu.Unlock()
	joined := make(chan struct{})
	go func() { w.wg.Wait(); close(joined) }()
	select {
	case <-joined:
	case <-time.After(mapJoinLimit):
	}
}

// mapTickMsg is the map's clock: a draw the library asked for at a time.
type mapTickMsg struct{ at time.Time }

// armMapTick keeps one tick outstanding at the library's NextCall while the
// window is open, and none while it is closed.
func (d Dashboard) armMapTick() (Dashboard, tea.Cmd) {
	m := d.mapPane.m
	if m == nil || d.modal != modalMap {
		d.mapPane.tickAt = time.Time{}
		return d, nil
	}
	now := d.now()
	var at time.Time
	var ok bool
	d.mapPane.call("NextCall", func() { at, ok = m.NextCall(now) })
	if !ok || at.Equal(d.mapPane.tickAt) {
		return d, nil
	}
	d.mapPane.tickAt = at
	return d, tea.Tick(max(at.Sub(now), mapTickFloor), func(time.Time) tea.Msg { return mapTickMsg{at: at} })
}

// applyMapTick draws for the tick outstanding; one for a time no longer
// wanted is dropped, the newer one being already armed.
func (d Dashboard) applyMapTick(v mapTickMsg) Dashboard {
	if d.modal != modalMap || !v.at.Equal(d.mapPane.tickAt) {
		return d
	}
	d.mapPane.tickAt = time.Time{}
	return d.renderMap()
}

// callSite is one library call and the goroutine it ran on, for W2.2's test.
type callSite struct {
	name      string
	goroutine uint64
}

// goroutineID is the calling goroutine's number, read from its stack's first
// line. For the tests' record only: nothing decides by it.
func goroutineID() uint64 {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	buf = bytes.TrimPrefix(buf, []byte("goroutine "))
	if i := bytes.IndexByte(buf, ' '); i > 0 {
		buf = buf[:i]
	}
	n, _ := strconv.ParseUint(string(buf), 10, 64)
	return n
}

// CloseMap lets Observer's map go when the program ends: the app calls it
// with the model the program ended with.
func (r Router) CloseMap() { r.observer.closeMap() }
