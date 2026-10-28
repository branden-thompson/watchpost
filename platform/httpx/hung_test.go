package httpx

// A host that hangs (watchpost 0.19.0 D-324): it is avoided once it has not
// answered in the client's time, and while it hangs it holds at most half of
// a lane, so the other hosts' requests keep moving.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// hungServer holds every request until the test ends, counting those in
// flight and the most at once.
type hungServer struct {
	*httptest.Server
	inflight, most atomic.Int32
	release        chan struct{}
}

func newHungServer(t *testing.T) *hungServer {
	t.Helper()
	hs := &hungServer{release: make(chan struct{})}
	hs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hs.inflight.Add(1)
		defer hs.inflight.Add(-1)
		for m := hs.most.Load(); n > m && !hs.most.CompareAndSwap(m, n); m = hs.most.Load() {
		}
		select {
		case <-hs.release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(hs.release); hs.Close() })
	return hs
}

func TestAHostThatNeverAnswersIsAvoided(t *testing.T) {
	hs := newHungServer(t)
	c := mustNew(t, Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, RetryBase: time.Millisecond, MaxRetries: 1, Timeout: 100 * time.Millisecond})
	if _, err := c.GetJSON(context.Background(), hs.URL+"/a", nil, NoCache()); err == nil {
		t.Fatal("a host that never answers must fail the call")
	}
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); !avoided {
		t.Fatal("a host that does not answer in the client's time must be avoided, as one that refuses the connection is")
	}
	if _, err := c.GetJSON(context.Background(), hs.URL+"/b", nil, NoCache()); err == nil || !strings.Contains(err.Error(), "being avoided") {
		t.Fatalf("the next request to it must fail fast, got %v", err)
	}
}

func TestTheCallersOwnDeadlineNeverMarksTheHost(t *testing.T) {
	hs := newHungServer(t)
	c := mustNew(t, Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, RetryBase: time.Millisecond, MaxRetries: 1, Timeout: 5 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.GetJSON(ctx, hs.URL+"/a", nil, NoCache()); err == nil {
		t.Fatal("the call must end with its caller's deadline")
	}
	if _, avoided := c.memo.avoiding("127.0.0.1", time.Now()); avoided {
		t.Fatal("the caller's own deadline is the caller's decision, never the host's fault")
	}
}

func TestAHungHostHoldsAtMostHalfALane(t *testing.T) {
	for _, lane := range []struct {
		name string
		ctx  context.Context
		size int
	}{
		{"normal", context.Background(), maxInflight},
		{"priority", WithPriority(context.Background()), maxInflightPriority},
		{"interactive", WithInteractive(context.Background()), maxInflightInteract},
	} {
		t.Run(lane.name, func(t *testing.T) {
			hs := newHungServer(t)
			fs := newFaultServer(t)
			other := strings.Replace(fs.URL, "127.0.0.1", "localhost", 1) // another host, as the cap counts hosts
			c := mustNew(t, Config{UserAgent: "t (t@example.com)", RatePerSec: 1000, RetryBase: time.Millisecond, Timeout: 10 * time.Second})
			for i := range lane.size {
				go func() { _, _ = c.GetJSON(lane.ctx, fmt.Sprintf("%s/hang/%d", hs.URL, i), nil, NoCache()) }()
			}
			for wait := time.Now().Add(5 * time.Second); hs.inflight.Load() < int32(lane.size/2) && time.Now().Before(wait); {
				time.Sleep(5 * time.Millisecond)
			}
			start := time.Now()
			if _, err := c.GetJSON(lane.ctx, other+"/ok", nil, NoCache()); err != nil {
				t.Fatalf("another host's request must be served beside a hung one: %v", err)
			}
			if waited := time.Since(start); waited > 2*time.Second {
				t.Fatalf("another host's request waited %v behind a hung one", waited)
			}
			if most := hs.most.Load(); most > int32(lane.size/2) {
				t.Fatalf("the hung host held %d of the %s lane's %d", most, lane.name, lane.size)
			}
		})
	}
}

func TestAHostsShareComesBackWithEachAnswer(t *testing.T) {
	fs := newFaultServer(t)
	c := dashClient(t)
	for _, ctx := range []context.Context{context.Background(), WithPriority(context.Background()), WithInteractive(context.Background())} {
		for i := range maxInflight + 1 { // past any lane's share, one after another
			ask, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, err := c.GetJSON(ask, fmt.Sprintf("%s/seq/%d", fs.URL, i), nil, NoCache())
			cancel()
			if err != nil {
				t.Fatalf("request %d of one host, each after the last answered, found no share: %v", i+1, err)
			}
		}
	}
}

func TestAWaitThatEndsGivesTheShareBack(t *testing.T) {
	c := dashClient(t)
	for range maxInflight {
		c.inflight[laneNormal] <- struct{}{}
	}
	for range maxInflight { // more waits than the host's share, each ended by its caller
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		if _, err := c.acquire(ctx, "example.com"); err == nil {
			t.Fatal("control: a full lane gave a slot")
		}
		cancel()
	}
	for range maxInflight {
		<-c.inflight[laneNormal]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	release, err := c.acquire(ctx, "example.com")
	if err != nil {
		t.Fatalf("the host's share was kept by waits that ended: %v", err)
	}
	release()
}

func TestTheSharesAreBounded(t *testing.T) {
	c := dashClient(t)
	for i := range 3 * maxShareHosts {
		c.share(fmt.Sprintf("host%d.example.com", i))
	}
	if n := len(c.shares); n > maxShareHosts+1 {
		t.Fatalf("%d hosts hold a share; at most %d of their own and the one they share", n, maxShareHosts)
	}
}
