package tty

import (
	"fmt"
	"testing"
)

// TestTheRadiosPlannedHeightIsItsRows is 0.19.5 Q3.3 (PF-3): the layout reads
// both players' heights from the plan and renders only the one it draws, so
// the plan's count has to be the rows the player draws. Across the three
// breakpoints and their edges, from 40 to 220 columns, playing a voice with a
// long station name and a long detail or stopped, the visualiser on and off,
// compact and full, the two are equal, and the frame's player is the rows the
// mode in force renders.
func TestTheRadiosPlannedHeightIsItsRows(t *testing.T) {
	for _, w := range []int{40, 60, 78, 83, 84, 85, 100, 120, 133, 145, 146, 147, 160, 200, 220} {
		for _, playing := range []bool{false, true} {
			for _, viz := range []bool{false, true} {
				d := playingDash(t, w, 44)
				if !playing {
					d.radioPlaying, d.radioState, d.radioStation, d.radioDetail = false, "", "", ""
				}
				d.radioViz = viz
				o := d.opts()
				p := d.planRadio(o)
				for _, compact := range []bool{false, true} {
					name := fmt.Sprintf("%d columns, playing %v, visualiser %v, compact %v", w, playing, viz, compact)
					if got, want := p.rows(compact, viz), len(d.radioLines(o, compact)); got != want {
						t.Errorf("%s: the plan counts %d rows; the player draws %d", name, got, want)
					}
				}
				for _, h := range []int{24, 44, 70} {
					d.height = h
					fl := d.layout()
					if want := d.radioLines(o, fl.compact); fmt.Sprint(fl.radioRows) != fmt.Sprint(want) {
						t.Errorf("%dx%d, playing %v, visualiser %v: the frame's player is not the rows its mode renders", w, h, playing, viz)
					}
				}
			}
		}
	}
}
