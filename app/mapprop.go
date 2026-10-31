package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/domains/propagation"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// errPropagationUnavailable is what the window says when the service could
// not be made: the Status window carries why (D-124).
var errPropagationUnavailable = errors.New("not available in this session; the Status window (S) says why")

// propagationBodyBytes caps a propagation reply: about three times the
// measured 2.5 MB GloTEC grid (W0.0, D-141), go-ionomaps' own bound.
const propagationBodyBytes = 8 << 20

// propagationClientConfig is the propagation client's: its own instance, no
// retry, no cache, no pacing shared with the data client (FR-4.4), https
// alone, no private address and a body cap (FR-4.7), each host's tightened to
// what go-ionomaps parses from its source (W0.2).
func propagationClientConfig() httpx.PlainConfig {
	caps := map[string]int64{}
	for _, s := range ionomaps.Sources() {
		for _, h := range s.Hosts {
			caps[h] = int64(s.MaxBytes)
		}
	}
	return httpx.PlainConfig{UserAgent: UserAgent, MaxBodyBytes: propagationBodyBytes, MaxBodyByHost: caps, RefusePrivate: true, HTTPSOnly: true}
}

// newPropagationService is how the service is made over its client; a test
// counts it.
var newPropagationService = func(note func(string), client *httpx.Plain) (*propagation.Service, error) {
	return propagation.NewService(client, propagation.Options{Note: note, StateFile: config.StatePath("propagation-throttle.json")}) // GIRO's throttle across restarts (D-213)
}

// propagationHolder is the process's one go-ionomaps object (FR-4.8), made
// on the first update, which the window asks only once its mode is open
// (FR-4.2).
type propagationHolder struct {
	once   sync.Once
	svc    *propagation.Service
	err    error
	client atomic.Pointer[httpx.Plain] // the Status window reads its counts from the UI goroutine (D-169)
}

// settleBound is how long the app's end waits for an update in flight to
// keep GIRO's throttle state: the app's cancel, which comes first, stops the
// update's ask at once, and the update keeps its state as it ends.
const settleBound = 2 * time.Second

// settle waits at the app's end for an update in flight to keep its state
// (D-213); a service never made has nothing to wait for. The once orders
// this read after the service was made - and spends the once, so a service
// not made by now never is: a late Ask finds no service and answers so
// (domains/propagation's Ask, its nil check). The app's end goes on whether
// or not the wait was long enough; Settle says which for its tests.
func (h *propagationHolder) settle(bound time.Duration) {
	h.once.Do(func() {})
	if h.svc != nil {
		h.svc.Settle(bound)
	}
}

// service is the one service, made once, over its own client.
func (h *propagationHolder) service(note func(string)) (*propagation.Service, error) {
	h.once.Do(func() {
		client, err := httpx.NewPlain(propagationClientConfig())
		if err != nil {
			h.err = err
			return
		}
		h.client.Store(client)
		h.svc, h.err = newPropagationService(note, client)
	})
	return h.svc, h.err
}

// requestStats are the propagation client's asks per host (D-169); none
// before the first update makes it.
func (h *propagationHolder) requestStats() httpx.RequestStats {
	if c := h.client.Load(); c != nil {
		return c.RequestStats()
	}
	return httpx.RequestStats{}
}

// wireBytes is what the propagation client has read off the wire, still
// compressed (G1, D-141); 0 before the first update makes it, or with no
// holder.
func (h *propagationHolder) wireBytes() int64 {
	if h == nil {
		return 0
	}
	if c := h.client.Load(); c != nil {
		return c.WireBytes()
	}
	return 0
}

// propagationUpdate is the window's seam (A-30): the service's update, its
// result as the window reads it.
func (lp *livePipelines) propagationUpdate(ctx context.Context, opts ionomaps.UpdateOptions) <-chan tty.PropagationResult {
	out := make(chan tty.PropagationResult, 1)
	svc, err := lp.propagation.service(lp.problems.note)
	if err != nil {
		lp.problems.note("Propagation: not started - " + err.Error()) // ours to fix, never the listener's (D-124)
		out <- tty.PropagationResult{Err: errPropagationUnavailable}
		return out
	}
	go func() {
		ask, stop := context.WithCancel(ctx)
		defer stop()
		if lp.ctx != nil {
			defer context.AfterFunc(lp.ctx, stop)() // the app's end stops the update, so it keeps GIRO's state inside settleBound (D-213, R6T-1)
		}
		r := <-svc.Ask(ask, opts)
		out <- tty.PropagationResult{Snapshot: r.Snapshot, Err: r.Err}
	}()
	return out
}

// propagationHosts are MAP STATUS's rows for the Propagation mode's hosts
// (FR-4.3): each learns the IP address; none is sent the view or the place.
func propagationHosts() []tty.MapSource {
	var out []tty.MapSource
	for _, s := range ionomaps.Sources() {
		for _, h := range s.Hosts {
			out = append(out, tty.MapSource{Name: s.Name, Host: h, Layers: "MUF, foF2",
				Notes: []string{"Propagation sources asked only while the mode is open; learn this computer's IP, sent no place"}}) // said once under the hosts (D-172)
		}
	}
	return out
}
