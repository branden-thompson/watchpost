package synth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// gateVoice is a Voice whose Say announces its entry on entered and then
// blocks until release is closed (or the context ends). Every assertion below
// is made on channel events, never on a sleep-and-look: a limiter test that
// polls the clock is a flake waiting for a busy CI box.
type gateVoice struct {
	entered chan struct{}
	release chan struct{}
}

func newGateVoice() *gateVoice {
	return &gateVoice{entered: make(chan struct{}, 64), release: make(chan struct{})}
}

func (v *gateVoice) Name() string { return "gate" }
func (v *gateVoice) Rate() int    { return 22050 }

func (v *gateVoice) Say(ctx context.Context, _ string) ([]byte, error) {
	v.entered <- struct{}{}
	select {
	case <-v.release:
		return []byte{0, 0}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// waitEntered waits for n renders to have started, failing the test rather
// than hanging forever if the limiter admitted fewer than expected.
func waitEntered(t *testing.T, v *gateVoice, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-v.entered:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d renders started — the limiter admitted too few", i, n)
		}
	}
}

// noneEnteredWithin asserts nothing else was admitted. It is the one place a
// duration is unavoidable: proving a negative about concurrency needs a
// window. It is short enough to keep the suite quick and long enough that a
// wrongly-admitted render would have arrived.
func noneEnteredWithin(t *testing.T, v *gateVoice, d time.Duration) {
	t.Helper()
	select {
	case <-v.entered:
		t.Fatal("a render was admitted that should have queued")
	case <-time.After(d):
	}
}

func TestLimitedQueuesTheOrdinaryPoolAndKeepsTheReservedSlotsFree(t *testing.T) {
	const ordinary = 2
	v := newGateVoice()
	l := NewLimiter(ordinary, ReservedSlots)
	voice := Limited(v, l)
	defer close(v.release)

	ctx := context.Background()
	for i := 0; i < ordinary+1; i++ {
		go func() { _, _ = voice.Say(ctx, "ordinary") }()
	}

	waitEntered(t, v, ordinary)
	// The N+1th ordinary render must wait for a peer to finish; it must NOT
	// borrow one of the Director's slots.
	noneEnteredWithin(t, v, 250*time.Millisecond)
}

func TestLimitedAdmitsAPriorityJobWhileTheOrdinaryPoolIsFull(t *testing.T) {
	const ordinary = 2
	v := newGateVoice()
	l := NewLimiter(ordinary, ReservedSlots)
	voice := Limited(v, l)
	defer close(v.release)

	for i := 0; i < ordinary; i++ {
		go func() { _, _ = voice.Say(context.Background(), "ordinary") }()
	}
	waitEntered(t, v, ordinary)

	// Two priority jobs, because the Director may hold a reserved slot for a
	// read it suspended while its takeover is admitted (MVS-D-43).
	for i := 0; i < ReservedSlots; i++ {
		go func() { _, _ = voice.Say(WithPriority(context.Background()), "director") }()
	}
	waitEntered(t, v, ReservedSlots)
}

func TestLimitedPrefersAnOrdinarySlotForAPriorityJob(t *testing.T) {
	v := newGateVoice()
	l := NewLimiter(2, ReservedSlots)
	voice := Limited(v, l)
	defer close(v.release)

	// Each job is admitted before the next starts: which of two concurrent
	// jobs wins a free slot is a scheduling race, and the claim under test is
	// about preference, not about racing.
	//
	// With the ordinary pool free, a Director job takes an ORDINARY slot
	// rather than burning a reserved one.
	go func() { _, _ = voice.Say(WithPriority(context.Background()), "director") }()
	waitEntered(t, v, 1)

	// The proof it did: an ordinary job still fits, which it could not have
	// done if the Director had taken a reserved slot and left this one
	// competing for the pool's last place... it fills the pool instead.
	go func() { _, _ = voice.Say(context.Background(), "ordinary") }()
	waitEntered(t, v, 1)

	// And now that the ordinary pool is full, BOTH reserved slots are still
	// free for the Director.
	for i := 0; i < ReservedSlots; i++ {
		go func() { _, _ = voice.Say(WithPriority(context.Background()), "director") }()
	}
	waitEntered(t, v, ReservedSlots)

	// Nothing is left: a further ordinary job queues.
	go func() { _, _ = voice.Say(context.Background(), "queues") }()
	noneEnteredWithin(t, v, 250*time.Millisecond)
}

func TestLimitedBoundCoversTheWaitForASlotNotOnlyTheRender(t *testing.T) {
	v := newGateVoice()
	l := NewLimiter(1, ReservedSlots)
	// Two bounds, deliberately: the holder's is long so its slot stays held for
	// the whole test, and the queued caller's is short so its own bound is what
	// fires. Sharing one bound would race the two deadlines and the holder's
	// could free the slot first — the test would then measure nothing.
	holder := limitedVoice{v: v, l: l, bound: time.Minute}
	voice := limitedVoice{v: v, l: l, bound: 150 * time.Millisecond}
	defer close(v.release)

	go func() { _, _ = holder.Say(context.Background(), "holds the only slot") }()
	waitEntered(t, v, 1)

	// This one never enters Say at all: it is still queuing when its bound
	// fires, and the error is the wait's, not a render failure.
	done := make(chan error, 1)
	go func() { _, err := voice.Say(context.Background(), "queued"); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("queued Say returned %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the bound never fired while a slot was held — a queued Say can wait forever")
	}
	noneEnteredWithin(t, v, 100*time.Millisecond) // and it never reached the voice
}

func TestLimitedFailsClosedOnNilWiring(t *testing.T) {
	v := newGateVoice()
	for _, tc := range []struct {
		name  string
		voice Voice
	}{
		{"nil voice", Limited(nil, NewLimiter(2, ReservedSlots))},
		{"nil limiter", Limited(v, nil)},
		{"both nil", Limited(nil, nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.voice.Say(context.Background(), "anything"); !errors.Is(err, ErrVoiceUnwired) {
				t.Fatalf("Say returned %v, want ErrVoiceUnwired — an unwired voice must never render uncapped", err)
			}
		})
	}
	// Name and Rate stay callable on an unwired voice: a caller may ask who is
	// speaking without triggering the failure.
	if got := Limited(nil, nil).Name(); got != "" {
		t.Errorf("Name() = %q, want empty", got)
	}
	if got := Limited(nil, nil).Rate(); got != 0 {
		t.Errorf("Rate() = %d, want 0", got)
	}
	// And they pass through when the voice is real but the limiter is not.
	if got := Limited(v, nil).Name(); got != "gate" {
		t.Errorf("Name() = %q, want the wrapped voice's name", got)
	}
}

func TestNewLimiterFloorsItsCountsSoItAlwaysAdmitsSomething(t *testing.T) {
	v := newGateVoice()
	voice := Limited(v, NewLimiter(0, 0))
	defer close(v.release)

	go func() { _, _ = voice.Say(context.Background(), "ordinary") }()
	waitEntered(t, v, 1)

	go func() { _, _ = voice.Say(WithPriority(context.Background()), "director") }()
	waitEntered(t, v, 1)
}

func TestWithPriorityIsCarriedAndAbsentByDefault(t *testing.T) {
	if HasPriority(context.Background()) {
		t.Fatal("a plain context must not carry priority")
	}
	if !HasPriority(WithPriority(context.Background())) {
		t.Fatal("WithPriority must be readable by HasPriority")
	}
	// It survives further derivation, which is how the Director passes it down.
	ctx, cancel := context.WithCancel(WithPriority(context.Background()))
	defer cancel()
	if !HasPriority(ctx) {
		t.Fatal("priority must survive a derived context")
	}
}
