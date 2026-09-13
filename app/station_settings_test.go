package app

// station_settings_test.go — writing the station's two settings re-derives its
// pool (D-115, F-87).
//
// HUM LEAD, 2026-09-13: the settings are exposed "so I can also UAT the
// re-derivation logic" — so the re-derivation is what these assert, not just
// the write.

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func stationPipes(t *testing.T) *livePipelines {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	idx, err := geodata.Load()
	if err != nil {
		t.Fatalf("the embedded location table is what a pool is derived from: %v", err)
	}
	lp := &livePipelines{idx: idx}
	lp.setStation(stationArea{
		transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Zip: "92003", Lat: 33.28, Lon: -117.23},
		radiusMi:    25, followsDefault: true})
	return lp
}

// A NEW TRANSMITTER MOVES THE POOL, and stops the station borrowing.
func TestSettingTheTransmitterReDerivesThePool(t *testing.T) {
	lp := stationPipes(t)
	before := lp.currentPool()
	if len(before) == 0 {
		t.Fatal("the station started with no pool, so a move proves nothing")
	}

	lp.setTransmitter(snapshot.LocationRef{Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2})

	after := lp.currentPool()
	if len(after) == 0 {
		t.Fatal("the new transmitter derived no pool at all")
	}
	if snapshot.Key(after[0]) == snapshot.Key(before[0]) {
		t.Errorf("the pool did not move: still %s", after[0].Label)
	}
	// AND IT STOPPED BORROWING (D-72): a station with a transmitter of its own
	// does not follow the listener's watchlist any more.
	if lp.currentStation().followsDefault {
		t.Error("the station still follows the listener's default after being given its own")
	}
	// AND IT PERSISTED, or the operator's choice dies with the process.
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Broadcaster.Transmitter.Zip != "83702" {
		t.Errorf("the transmitter persisted as %+v", cfg.Broadcaster.Transmitter)
	}
}

// A NEW RADIUS RESIZES IT — the other half of the same derivation.
func TestSettingTheServiceRadiusReDerivesThePool(t *testing.T) {
	lp := stationPipes(t)
	// FROM A FENCE TOO SMALL TO FILL THE CAP. `locations.PoolCap` is 25 and a
	// 25-mile radius around Bonsall already fills it — so widening from there
	// cannot change the COUNT, and a test that watched the count would have
	// reported the derivation broken when it was the cap doing its job.
	lp.setStation(stationArea{transmitter: lp.currentStation().transmitter, radiusMi: 5})
	narrow := len(lp.currentPool())
	if narrow == 0 {
		t.Fatal("a five-mile fence derived nothing, so a widening proves nothing")
	}

	lp.setServiceRadius(100)

	wide := len(lp.currentPool())
	if wide <= narrow {
		t.Errorf("a 100-mile radius derived %d candidates and 5 miles derived %d; the pool follows the reach",
			wide, narrow)
	}
	if got := lp.currentStation().radiusMi; got != 100 {
		t.Errorf("the station's radius is %v, want 100", got)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Broadcaster.ServiceRadius() != 100 {
		t.Errorf("the radius persisted as %v", cfg.Broadcaster.ServiceRadius())
	}
	// AND A RADIUS CHANGE DOES NOT MOVE THE EPICENTRE, which is the half that
	// would be easy to lose in a shared "restation" path.
	if lp.currentStation().transmitter.Zip != "92003" {
		t.Errorf("the radius change moved the transmitter to %+v", lp.currentStation().transmitter)
	}
}

// THE WINDOW'S BOUNDS ARE THE STORAGE'S BOUNDS.
//
// `modes/tty` may not import `platform/config` (make lint-imports), so the floor
// and the ceiling are stated in both places. This is the tie: a change in one
// fails here rather than letting the window offer a radius the storage would
// silently clamp — which the operator would read as "my setting was not saved".
func TestSetupServiceBoundsMatchTheConfig(t *testing.T) {
	if tty.ServiceRadiusMinForTest != config.MinServiceRadiusMi {
		t.Errorf("the window's floor is %d and the storage's is %v",
			tty.ServiceRadiusMinForTest, config.MinServiceRadiusMi)
	}
	if tty.ServiceRadiusMaxForTest != config.MaxServiceRadiusMi {
		t.Errorf("the window's ceiling is %d and the storage's is %v",
			tty.ServiceRadiusMaxForTest, config.MaxServiceRadiusMi)
	}
}
