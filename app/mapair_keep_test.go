package app

import (
	"context"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// A BOX'S AIR IS RASTERISED ONCE A FILE (W14 P-19, D-212): the same hour's
// contours over the same box are the same cells, so a box's grid is kept by
// the box and the file's hour - an ask again, or a pan within the boxes,
// rasterises nothing.
func TestABoxsAirIsRasterisedOnceAFile(t *testing.T) {
	airnow := airquality.New(&contoursFile{}, "")
	ask := tempAsk(false)
	ask.View, ask.Air = tty.MapView{W: -124.5, S: 32.5, E: -114, N: 42}, true
	now := time.Date(2026, 10, 1, 4, 30, 0, 0, time.UTC)
	grids := &airGrids{}
	first := withAir(context.Background(), tty.MapTemperature{}, airnow, grids, ask, now)
	built := grids.built
	if len(first.Air) == 0 || built != len(first.Air) {
		t.Fatalf("%d grids drawn, %d built; this test measures nothing", len(first.Air), built)
	}
	again := withAir(context.Background(), tty.MapTemperature{}, airnow, grids, ask, now)
	if grids.built != built || len(again.Air) != len(first.Air) {
		t.Errorf("an ask again built %d grids more and drew %d; want none built, the same drawn", grids.built-built, len(again.Air))
	}
}
