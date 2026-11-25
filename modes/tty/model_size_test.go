package tty

import (
	"testing"
	"unsafe"

	"github.com/branden-thompson/watchpost/modes/tty/console"
)

// modelSizes are the models' sizes on a 64-bit platform, the three at 0.19.6
// DISCOVER (PB-2) and the map's at B4.3, where its fields join it (D-76): each
// is copied into and out of every call on a value receiver, and a field that
// moves by value into two places shows here first.
var modelSizes = map[string]uintptr{"Router": 7_136, "Dashboard": 6_584, "Broadcaster": 472, "mapModel": 3_440}

// TestTheModelsAreNoLarger is 0.19.6 D-28's "model sizes not larger": a batch
// that moves state may shrink a model and never grow one. A new sub-model
// joins the list in the batch that creates it, at its size there.
func TestTheModelsAreNoLarger(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the sizes are a 64-bit platform's; the gates run on two")
	}
	for name, size := range map[string]uintptr{"Router": unsafe.Sizeof(Router{}), "Dashboard": unsafe.Sizeof(Dashboard{}), "Broadcaster": unsafe.Sizeof(console.Broadcaster{}), "mapModel": unsafe.Sizeof(mapModel{})} { // the models (P10-02)
		was := modelSizes[name]
		t.Logf("%s: %d bytes (at most %d)", name, size, was)
		if size > was {
			t.Errorf("%s is %d bytes, larger than its recorded %d: a model may not grow (0.19.6 D-28)", name, size, was)
		}
	}
}
