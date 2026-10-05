package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/marine/ndbc"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// overlap counts the feed's inputs being asked at once. Each request holds
// until every input is in flight, so inputs asked together all meet; inputs
// asked one after another never do, and each is let go after holdAtMost.
type overlap struct {
	mu       sync.Mutex
	want     int
	inFlight map[string]int
	peak     int
	all      chan struct{}
}

// holdAtMost is how long a request waits for the others: it bounds the test
// only when the inputs are asked one after another.
const holdAtMost = 5 * time.Second

func newOverlap(want int) *overlap {
	return &overlap{want: want, inFlight: map[string]int{}, all: make(chan struct{})}
}

// ask is one request of an input: in flight until every input is.
func (o *overlap) ask(input string) {
	o.mu.Lock()
	o.inFlight[input]++
	busy := 0
	for _, n := range o.inFlight {
		if n > 0 {
			busy++
		}
	}
	if busy > o.peak {
		o.peak = busy
		if busy == o.want {
			close(o.all)
		}
	}
	o.mu.Unlock()
	select {
	case <-o.all:
	case <-time.After(holdAtMost):
	}
	o.mu.Lock()
	o.inFlight[input]--
	o.mu.Unlock()
}

// most is the most inputs that were in flight at once.
func (o *overlap) most() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.peak
}

// heldGet is a Getter whose every request is one of an input's.
type heldGet struct {
	o     *overlap
	input string
}

func (g heldGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	g.o.ask(g.input)
	return nil, nil
}

// TestTheFeedsInputsAreAskedTogether is W14's P-5: the view's alerts, the
// fire, the quakes, the buoys, the tide stations and AirNow do not depend on
// one another, so they are asked together - every one of the six is in flight
// at once - and the feed waits for the slowest, not the sum.
func TestTheFeedsInputsAreAskedTogether(t *testing.T) {
	const inputs = 6
	o := newOverlap(inputs)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.ask(inputOfPath(r.URL.Path))
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	c, _ := httpx.New(httpx.Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, MaxRetries: 0})
	lp := &livePipelines{idx: idx,
		areaAlerts: func(context.Context, []string) ([]snapshot.Alert, error) { o.ask("alerts"); return nil, nil },
		fire:       []snapshot.Provider{wfigs.New(c, srv.URL+"/fire/query", fire.DefaultRules())}, rules: fire.DefaultRules(),
		mapQuakes: newMapQuakes(c, srv.URL+"/quakes/"),
		marine:    []snapshot.Provider{ndbc.New(c, srv.URL+"/buoys"), coops.New(c, srv.URL+"/tides")},
		airnow:    airquality.New(heldGet{o, "airnow"}, ""),
	}
	ask := tty.MapAsk{Snap: &snapshot.Snapshot{}, View: tty.MapView{W: -118.6, S: 32.5, E: -116.4, N: 34.1},
		Fire: true, Quakes: true, Buoys: true, Tides: true, Air: true}
	_ = lp.mapInputsFetching(context.Background(), ask)
	if got := o.most(); got < inputs {
		t.Errorf("at most %d of the %d inputs were asked at once; all %d are asked together", got, inputs, inputs)
	}
}

// inputOfPath is the input a request to the test server is for: its path's
// first part.
func inputOfPath(path string) string {
	first, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return first
}
