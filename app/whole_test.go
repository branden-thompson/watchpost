package app

// whole_test.go — the map's builders as most tests call them: the drawing
// alone. Production calls the *Whole forms, whose flag decides whether an
// answer is kept for its hour (tempCoreFor); the flag's own tests are beside
// these.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

func buildTemperature(ctx context.Context, src, fill temperature.Source, ask tty.MapAsk, now time.Time, rescue *fallback) tty.MapTemperature {
	t, _ := buildTemperatureWhole(ctx, src, fill, ask, now, rescue)
	return t
}

func withRainDays(ctx context.Context, t tty.MapTemperature, om *temperature.OpenMeteo, ask tty.MapAsk, now time.Time, rescue *rainRescue) tty.MapTemperature {
	t, _ = withRainDaysWhole(ctx, t, om, ask, now, rescue)
	return t
}

func withWaves(ctx context.Context, t tty.MapTemperature, ndfd waveSource, om marineSource, ask tty.MapAsk, now time.Time, keep waveKeep) tty.MapTemperature {
	t, _ = withWavesWhole(ctx, t, ndfd, om, ask, now, keep)
	return t
}

// TestTheWholeFlagSaysWhetherEverySourceAnswered holds the flag tempCoreFor
// keeps an answer by: whole only where the source asked drew every box, the
// rain answered every box and the waves left no day unanswered. A partial
// answer kept for its hour would hide the missing boxes until the next one.
func TestTheWholeFlagSaysWhetherEverySourceAnswered(t *testing.T) {
	ctx := context.Background()
	if _, whole := buildTemperatureWhole(ctx, &fakeTemp{name: "Open-Meteo", now: tempNow}, nil, tempAsk(false), tempNow, nil); !whole {
		t.Error("every box answered, and the temperature is not whole")
	}
	if _, whole := buildTemperatureWhole(ctx, &fakeTemp{name: "NDFD", now: tempNow, failed: true}, &fakeTemp{name: "Open-Meteo", now: tempNow}, tempAsk(false), tempNow, nil); whole {
		t.Error("every box filled by another source, and the temperature is whole")
	}
	if _, whole := buildTemperatureWhole(ctx, &fakeTemp{name: "Open-Meteo", now: tempNow, failed: true}, nil, tempAsk(false), tempNow, nil); whole {
		t.Error("every box refused, and the temperature is whole")
	}

	ask := tempAsk(true)
	if _, whole := withRainDaysWhole(ctx, tty.MapTemperature{}, temperature.NewOpenMeteo(rainDaysGet(), ""), ask, tempNow, &rainRescue{}); !whole {
		t.Error("Open-Meteo answered the rain for every box, and the rain is not whole")
	}
	if _, whole := withRainDaysWhole(ctx, tty.MapTemperature{}, temperature.NewOpenMeteo(refusingGet{}, ""), ask, tempNow, &rainRescue{}); whole {
		t.Error("Open-Meteo refused the rain, and the rain is whole")
	}

	if _, whole := withWavesWhole(ctx, tty.MapTemperature{}, fakeWaves{metres: 1, max: 2}, fakeWaves{metres: 1, max: 2}, ask, tempNow, waveKeep{}); !whole {
		t.Error("NDFD answered every box and day, and the waves are not whole")
	}
	if _, whole := withWavesWhole(ctx, tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{failed: true}, ask, tempNow, waveKeep{}); whole {
		t.Error("no source answered the waves, and they are whole")
	}
}

// overlapTemp is a source that holds each fetch a moment and keeps the most
// fetches in flight at once.
type overlapTemp struct {
	fakeTemp
	in, most atomic.Int64
	mu       sync.Mutex
}

func (o *overlapTemp) Fetch(ctx context.Context, l temperature.Lattice, now time.Time) (temperature.Series, error) {
	n := o.in.Add(1)
	for m := o.most.Load(); n > m && !o.most.CompareAndSwap(m, n); m = o.most.Load() {
	}
	time.Sleep(50 * time.Millisecond)
	o.in.Add(-1)
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.fakeTemp.Fetch(ctx, l, now)
}

// TestTheBoxesAreFetchedTogether is REVIEW PF-5: a view across several boxes
// asks for them together, not one after another.
func TestTheBoxesAreFetchedTogether(t *testing.T) {
	ask := tempAsk(false)
	ask.View = geo.Box{W: -97, S: 34, E: -92, N: 40} // across the grid's lines: several boxes
	if n := len(fieldBoxes(ask.Region, ask.View)); n < 2 {
		t.Fatalf("the view takes %d box: this measures nothing", n)
	}
	src := &overlapTemp{fakeTemp: fakeTemp{name: "Open-Meteo", now: tempNow}}
	_, _ = buildTemperatureWhole(context.Background(), src, nil, ask, tempNow, nil)
	if src.most.Load() < 2 {
		t.Errorf("at most %d box was fetched at once; want them together", src.most.Load())
	}
}
