package app

import "testing"

// KEPTCOPY'S COPY IS ITS OWN: an append to a copy's slice never writes into
// the original's spare room, and a key set in a copy's map never reaches the
// original's - whatever room the original's slices have to grow.
func TestAKeptCopyIsItsOwn(t *testing.T) {
	type answer struct {
		Grids []int
		Chips map[string]string
	}
	grids := make([]int, 2, 8) // room to grow
	kept := answer{Grids: grids, Chips: map[string]string{"temp": "NDFD"}}
	a, b := keptCopy(kept), keptCopy(kept)
	a.Grids = append(a.Grids, 1)
	b.Grids = append(b.Grids, 2)
	a.Chips["uv"] = "EPA"
	if a.Grids[2] != 1 || kept.Grids[:3][2] != 0 {
		t.Errorf("a copy's append wrote into the original's room: a %v, original %v", a.Grids, kept.Grids[:3])
	}
	if _, ok := kept.Chips["uv"]; ok {
		t.Error("a key set in a copy's map reached the original's")
	}
}
