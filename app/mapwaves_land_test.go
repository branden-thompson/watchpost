package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// THE LAND IS KEPT ACROSS LAUNCHES (D-218): the points Open-Meteo Marine
// answered nothing for are kept in a file, so a later session's first ask
// for a box already leaves them out - one billed discovery, not one a launch.
func TestTheLandIsKeptAcrossLaunches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marine-land.json")
	now := tempNow
	clock := func() time.Time { return now }
	sea := temperature.LatticeFor("box", geo.Box{W: -80, S: 30, E: -76, N: 34})
	first := landPointsAt(path, clock)
	first.learn(sea, []int{3, 5})
	next := landPointsAt(path, clock)
	if !next.is(sea, 3) || !next.is(sea, 5) || next.is(sea, 4) {
		t.Errorf("a later session knows %v, %v, %v of points 3, 5 and 4; want the two learned, not the third", next.is(sea, 3), next.is(sea, 5), next.is(sea, 4))
	}

	// A LATTICE OF ANOTHER SHAPE IS ANOTHER LATTICE: its point 3 is another place.
	other := temperature.LatticeOf("box", sea.Box, temperature.MaxPoints/2)
	if other.Cols == sea.Cols && other.Rows == sea.Rows {
		t.Fatal("the fixture's two lattices have one shape")
	}
	if next.is(other, 3) {
		t.Error("a point learned on one lattice was read on another")
	}

	// TWO INSTANCES' LEARNING IS MERGED, not the last writer's alone.
	far := temperature.LatticeFor("far", geo.Box{W: -70, S: 40, E: -66, N: 44})
	a, b := landPointsAt(path, clock), landPointsAt(path, clock)
	a.learn(far, []int{1})
	b.learn(sea, []int{7})
	merged := landPointsAt(path, clock)
	if !merged.is(far, 1) || !merged.is(sea, 7) || !merged.is(sea, 3) {
		t.Error("one instance's learning was lost to the other's write")
	}

	// AND IT IS ASKED AGAIN AFTER A SEASON: Open-Meteo's reach can grow.
	now = now.Add(landKeptFor + time.Hour)
	if landPointsAt(path, clock).is(sea, 3) {
		t.Errorf("land learned more than %v ago is still left out", landKeptFor)
	}
}

// AN UNREADABLE FILE IS NO LAND, NOT A FAILURE: the map asks as a first session would.
func TestAnUnreadableLandFileIsNoLand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marine-land.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	sea := temperature.LatticeFor("box", geo.Box{W: -80, S: 30, E: -76, N: 34})
	lp := landPointsAt(path, func() time.Time { return tempNow })
	if lp.is(sea, 0) {
		t.Error("a broken file read as land")
	}
	lp.learn(sea, []int{2})
	if !landPointsAt(path, func() time.Time { return tempNow }).is(sea, 2) {
		t.Error("learning over a broken file did not keep it")
	}
}

// A LATER SESSION'S FIRST ASK LEAVES THE KEPT LAND OUT (D-218): the URL is the
// one the session before ended on, so it does not shift - and is not billed
// again - as the session learns.
func TestALaterSessionsFirstAskLeavesTheLandOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marine-land.json")
	clock := func() time.Time { return tempNow }
	ask := tempAsk(true)
	var before, after [][]int
	ndfd := fakeWaves{metres: 1, max: 2, gapDays: true}
	_ = withWaves(context.Background(), tty.MapTemperature{}, ndfd, fakeWaves{metres: 3, max: 4, land: 2, asked: &before}, ask, tempNow, waveKeep{land: landPointsAt(path, clock)})
	_ = withWaves(context.Background(), tty.MapTemperature{}, ndfd, fakeWaves{metres: 3, max: 4, land: 2, asked: &after}, ask, tempNow, waveKeep{land: landPointsAt(path, clock)})
	if len(before) == 0 || len(after) == 0 || !slices.Contains(before[0], 2) {
		t.Fatalf("the first session asked %v; want the land among its first ask", before)
	}
	if slices.ContainsFunc(after[0], func(i int) bool { return i >= 2 }) {
		t.Errorf("the next session's first ask is %v; want the land learned before left out", after[0])
	}
}

// CLEARING THE MAP'S DATA FORGETS THE LAND (FR-3.10): it names the boxes the
// map was looked at in.
func TestClearingForgetsTheLand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marine-land.json")
	clock := func() time.Time { return tempNow }
	sea := temperature.LatticeFor("box", geo.Box{W: -80, S: 30, E: -76, N: 34})
	land := landPointsAt(path, clock)
	land.learn(sea, []int{1})
	lp := &livePipelines{temp: &tempSources{land: land}}
	got := lp.clearMapData()
	if got.Err != nil || got.Files == 0 {
		t.Errorf("cleared %+v; want the land file counted", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the land file is still there: %v", err)
	}
	if land.is(sea, 1) || landPointsAt(path, clock).is(sea, 1) {
		t.Error("the land is still known after the clear")
	}
}

// THE LAND FILE SITS BESIDE THE QUOTA'S STATE, and nowhere without one.
func TestTheLandFileSitsBesideTheQuotasState(t *testing.T) {
	if got := landStatePath("/state/watchpost/quota.json"); got != "/state/watchpost/marine-land.json" {
		t.Errorf("the land file is at %q", got)
	}
	if landStatePath("") != "" {
		t.Error("no state directory, yet a land file")
	}
}

// LEARNING ADDS TO WHAT IS KNOWN: a lattice's second lesson keeps its first.
func TestLearningAddsToWhatIsKnown(t *testing.T) {
	sea := temperature.LatticeFor("box", geo.Box{W: -80, S: 30, E: -76, N: 34})
	lp := &landPoints{}
	lp.learn(sea, []int{1})
	lp.learn(sea, []int{2})
	if !lp.is(sea, 1) || !lp.is(sea, 2) {
		t.Errorf("after two lessons the land is %v and %v; want both points", lp.is(sea, 1), lp.is(sea, 2))
	}
}
