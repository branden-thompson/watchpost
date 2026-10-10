package app

import (
	"context"
	"errors"
	"sync"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/domains/propagation"
	"github.com/branden-thompson/watchpost/modes/tty"
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
// alone, no private address and a body cap (FR-4.7).
func propagationClientConfig() httpx.PlainConfig {
	return httpx.PlainConfig{UserAgent: UserAgent, MaxBodyBytes: propagationBodyBytes, RefusePrivate: true, HTTPSOnly: true}
}

// newPropagationService is how the service is made; a test counts it.
var newPropagationService = func(note func(string)) (*propagation.Service, error) {
	client, err := httpx.NewPlain(propagationClientConfig())
	if err != nil {
		return nil, err
	}
	return propagation.New(client, propagation.Options{Note: note})
}

// propagationHolder is the process's one go-ionomaps object (FR-4.8), made
// on the first update, which the window asks only once its mode is open
// (FR-4.2).
type propagationHolder struct {
	once sync.Once
	svc  *propagation.Service
	err  error
}

// service is the one service, made once.
func (h *propagationHolder) service(note func(string)) (*propagation.Service, error) {
	h.once.Do(func() { h.svc, h.err = newPropagationService(note) })
	return h.svc, h.err
}

// propagationUpdate is the window's seam (D-131): the service's update, its
// result as the window reads it.
func (lp *livePipelines) propagationUpdate(ctx context.Context) <-chan tty.PropagationResult {
	out := make(chan tty.PropagationResult, 1)
	svc, err := lp.propagation.service(lp.problems.note)
	if err != nil {
		lp.problems.note("Propagation: not started - " + err.Error()) // ours to fix, never the listener's (D-124)
		out <- tty.PropagationResult{Err: errPropagationUnavailable}
		return out
	}
	go func() {
		r := <-svc.Ask(ctx)
		out <- tty.PropagationResult{Snapshot: r.Snapshot, Err: r.Err}
	}()
	return out
}

// propagationHostList is every host go-ionomaps asks at run time, read from
// the library's own list (D-113); a library made only to read it fetches
// nothing.
func propagationHostList() []string {
	var hosts []string
	for _, s := range propagationSources() {
		hosts = append(hosts, s.Hosts...)
	}
	return hosts
}

// propagationSources are the library's sources, its hosts and terms.
func propagationSources() []ionomaps.Source {
	lib, err := ionomaps.New(readOnly{}, ionomaps.Options{})
	if err != nil {
		return nil
	}
	return lib.Sources()
}

// readOnly is a fetcher that is never asked: the library is made to read
// its sources alone.
type readOnly struct{}

func (readOnly) Fetch(context.Context, string, ionomaps.Validators) (ionomaps.Response, error) {
	return ionomaps.Response{}, errPropagationUnavailable
}

// propagationHosts are MAP STATUS's rows for the Propagation mode's hosts
// (FR-4.3): each learns the IP address; none is sent the view or the place.
func propagationHosts() []tty.MapSource {
	var out []tty.MapSource
	for _, s := range propagationSources() {
		for _, h := range s.Hosts {
			out = append(out, tty.MapSource{Name: s.Name, Host: h, Layers: "MUF, foF2",
				Notes: []string{"Asked only while the Propagation mode is open; it learns this computer's IP address, and is sent no place."}})
		}
	}
	return out
}
