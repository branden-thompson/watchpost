package app

import (
	"context"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
)

// arbiter_race_test.go — FR-1.4, answered by measurement.
//
// THE REQUIREMENT ASSUMED A RACE THAT IS NOT THERE, and the plan's entry
// condition said to prove one before adding a lock. F-34 reads: "holdLine,
// resumeLine and dropHeld touch the voice with no lock at all, and are not
// serialised against dip's duck() and takeBack's restore()." True as written —
// and it does not follow that they need to be.
//
// WHAT THE SOURCE SAYS, checked rather than assumed:
//
//   - The three touch NO mutable mastercontrol state. They call silent(), which
//     reads m and m.v, and m.v is never written after construction. dip and
//     takeBack write m.ducked and m.held, and both already hold m.mu.
//   - Underneath, the deck delegates one-to-one to engine methods that each take
//     e.mu, and on DISJOINT fields: Suppress/Restore touch e.suppressed;
//     PausePreview/ResumePreview/DropHeld touch e.preview, e.held, e.heldOrder.
//
// So there is no shared mutable state to serialise, and a wrapper mutex would
// buy nothing measurable while adding a lock acquisition to the path that
// speaks alerts. This test is the evidence for that, and the guard that catches
// the day someone gives those three some state of their own.
func TestTheArbiterIsRaceFreeWithoutAWrapperLock(t *testing.T) {
	v := &countingVoice{}
	mc := newMastercontrol(v, func(tea.Msg) {})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	spin := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				f()
			}
		}()
	}
	// The three under suspicion, against the pair they are said to need
	// serialising with — two goroutines each, so an interleaving is reachable
	// rather than theoretical.
	for range 2 {
		spin(mc.holdLine)
		spin(mc.resumeLine)
		spin(mc.dropHeld)
		spin(mc.giveWay)
		spin(mc.takeBack)
	}
	wg.Wait()

	if v.calls() == 0 {
		t.Fatal("the voice was never called — the stress did not reach the code under test, " +
			"so a green result here would prove nothing")
	}
}

// countingVoice is race-safe ITSELF, so the detector reports on the arbiter
// rather than on the fixture. A fake that races is an instrument that measures
// its own noise.
type countingVoice struct {
	mu sync.Mutex
	n  int
}

func (v *countingVoice) hit() { v.mu.Lock(); v.n++; v.mu.Unlock() }

// fault is inert here: the seam exists so a read can report a tone with no
// words (FR-9.2).
func (v *countingVoice) fault(string) {}

func (v *countingVoice) calls() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.n
}

func (v *countingVoice) duck()                         { v.hit() }
func (v *countingVoice) restore()                      { v.hit() }
func (v *countingVoice) pause()                        { v.hit() }
func (v *countingVoice) resume()                       { v.hit() }
func (v *countingVoice) discard()                      { v.hit() }
func (v *countingVoice) stop()                         { v.hit() }
func (v *countingVoice) play(clip)                     { v.hit() }
func (v *countingVoice) tone(cast.Class) time.Duration { v.hit(); return 0 }
func (v *countingVoice) render(context.Context, cast.Role, string) (clip, bool) {
	v.hit()
	return clip{}, false
}
