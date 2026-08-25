package player

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Red-team 0.9.0 C-1: two overlapping starts must never orphan an engine
// goroutine — every stream is reachable from Halt. Before the fix this
// failed 50 runs in 50.
func TestConcurrentStartsNeverOrphanAStream(t *testing.T) {
	for iter := 0; iter < 20; iter++ { // bounded: before the fix every round of the 50 tried leaked; 20 keeps the suite fast
		e, _ := New(&fakeOutput{}, "watchpost/test (t@example.com)", func(Status) { time.Sleep(2 * time.Millisecond) })
		var mu sync.Mutex
		var ctxs []context.Context
		open := func(ctx context.Context) io.Reader {
			mu.Lock()
			ctxs = append(ctxs, ctx)
			mu.Unlock()
			return blockingReader{ctx: ctx}
		}
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); e.StartSource("s", OutputRate, open) }()
		}
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		e.Halt()
		mu.Lock()
		for _, c := range ctxs {
			if c.Err() == nil {
				mu.Unlock()
				t.Fatalf("round %d: a stream outlived Halt (orphaned engine goroutine)", iter)
			}
		}
		mu.Unlock()
	}
}

// blockingReader yields silence until its context ends.
type blockingReader struct{ ctx context.Context }

func (b blockingReader) Read(p []byte) (int, error) {
	select {
	case <-b.ctx.Done():
		return 0, io.EOF
	case <-time.After(2 * time.Millisecond):
		clear(p)
		return len(p), nil
	}
}

// Relay ToS (§10.4, red-team 0.9.0): a mount answering 403/404 is honoured
// — one request, no backoff retries — and the engine moves to the next.
func TestRefusedMountIsNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	e, _ := New(&fakeOutput{}, "watchpost/test (t@example.com)", nil)
	e.Start([]string{srv.URL + "/banned", srv.URL + "/also-banned"}, "x")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && e.Status().State != Failed { // bounded by the deadline
		time.Sleep(20 * time.Millisecond)
	}
	if e.Status().State != Failed {
		t.Fatalf("two refused mounts end in Failed, got %s", e.Status().State)
	}
	if hits.Load() != 2 {
		t.Fatalf("each refused mount is asked exactly once, got %d requests", hits.Load())
	}
	// Fail carries the caller's words (F2), never "no relay carries this transmitter".
	e.Fail("no voice for linux/arm: install Piper")
	if st := e.Status(); st.State != Failed || st.Err != "no voice for linux/arm: install Piper" {
		t.Fatalf("Fail reports the given reason: %+v", st)
	}
}
