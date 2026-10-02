package app

import (
	"context"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// aheadLoop is one box's hours ahead: a frame each at the times given.
func aheadLoop(id string, at ...time.Time) tuimaps.Overlay {
	var frames []tuimaps.LoopFrame
	for _, t := range at {
		frames = append(frames, tuimaps.LoopFrame{Valid: t, PNG: []byte{1}})
	}
	return tuimaps.Overlay{ID: id, Image: &tuimaps.Image{Frames: frames}}
}

// TestTheHoursAheadStandWhileTheyAreFetchedAgain is UAT-2 U2-59's second
// half: a new newest frame or horizon asks HRRR again, and until that lands
// the same boxes' last hours ahead are what an ask gets - the loop keeps its
// hours ahead rather than shrinking to the observed hours and growing back.
// Other boxes' hours ahead are not theirs to stand in for.
func TestTheHoursAheadStandWhileTheyAreFetchedAgain(t *testing.T) {
	var a aheadFetch
	t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	first := hoursAhead{name: "HRRR", loops: []tuimaps.Overlay{aheadLoop("radar/fc-us", t0.Add(15*time.Minute))}}
	a.take("us", "us|1", func(context.Context) hoursAhead { return first })
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, _, ok := a.take("us", "us|1", nil); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the first fetch never landed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	release := make(chan struct{})
	defer close(release)
	fc, again, _ := a.take("us", "us|2", func(ctx context.Context) hoursAhead {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return hoursAhead{}
	})
	if len(fc.loops) != 1 || fc.name != "HRRR" || again != aheadSoon {
		t.Errorf("while the same boxes' hours ahead are fetched again: %d loops from %q, again in %v; want the last ones, asked again in %v",
			len(fc.loops), fc.name, again, aheadSoon)
	}
	if fc, _, _ := a.take("us-b,us-c", "us-b,us-c|2", func(context.Context) hoursAhead { return hoursAhead{} }); len(fc.loops) != 0 {
		t.Errorf("other boxes got %d of these boxes' loops", len(fc.loops))
	}
}

// TestHoursAheadFromAnEarlierFetchStartAfterTheNewest is the guard on the
// last hours ahead standing in: a frame of theirs at or before the newest
// observed frame is not ahead of it any more, and is not joined.
func TestHoursAheadFromAnEarlierFetchStartAfterTheNewest(t *testing.T) {
	newest := time.Date(2026, 10, 2, 12, 5, 0, 0, time.UTC)
	fc := hoursAhead{name: "HRRR", loops: []tuimaps.Overlay{aheadLoop("radar/fc-us", newest.Add(-10*time.Minute), newest, newest.Add(15*time.Minute))}}
	out := joinForecast(tty.MapRadar{}, fc, newest)
	if len(out.Overlays) != 1 {
		t.Fatalf("joined %d loops; want the one", len(out.Overlays))
	}
	if frames := out.Overlays[0].Image.Frames; len(frames) != 1 || !frames[0].Valid.After(newest) {
		t.Errorf("joined frames %v; want only the one after the newest observed", frames)
	}
	if len(fc.loops[0].Image.Frames) != 3 {
		t.Error("joining trimmed the fetch's own frames: the next ask would start short")
	}
}
