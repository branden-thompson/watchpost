package app

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// A RADAR FRAME IS CHECKED ONCE (W14 P-17, D-212): a frame at its time never
// changes, so whether its picture paints anything is kept by the frame - its
// source, region, box, time and size - and a loop asked again, on a pan or a
// refresh, decodes none of the frames it has checked. Here the second ask's
// bytes would not decode at all: kept, every frame still stands.
func TestARadarFrameIsCheckedOnce(t *testing.T) {
	good, err := os.ReadFile("../domains/radar/testdata/hrrr-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	times := grid5(24, time.Date(2026, 9, 28, 14, 0, 0, 0, time.UTC))
	src := &fakeRadar{name: "MRMS", times: times, png: good}
	box := radar.Box{Name: "us", W: -126, S: 23, E: -65, N: 51}
	slots := loopSlots(times, radarStep, radar.Window)
	checks := &frameChecks{}
	first, _, ok := radarLoop(context.Background(), checks, src, geo.RegionContiguous, times, slots, box)
	if !ok {
		t.Fatal("no loop")
	}
	src.png = make([]byte, len(good)) // the same size, and no picture: a decode refuses it
	again, _, ok := radarLoop(context.Background(), checks, src, geo.RegionContiguous, times, slots, box)
	if !ok || held(again.Image.Frames) != held(first.Image.Frames) {
		t.Errorf("the loop asked again holds %d frames of %d: its frames were checked again", held(again.Image.Frames), held(first.Image.Frames))
	}
	if _, _, ok := radarLoop(context.Background(), &frameChecks{}, src, geo.RegionContiguous, times, slots, box); ok {
		t.Error("frames no check has seen were taken undecoded")
	}
}

// held counts a loop's frames with a picture.
func held(frames []tuimaps.LoopFrame) int {
	n := 0
	for _, f := range frames {
		if !f.Gap {
			n++
		}
	}
	return n
}

// pictureRadar answers each frame with the picture its time picks.
type pictureRadar struct {
	fakeRadar
	pick func(at time.Time) []byte
}

func (p *pictureRadar) Frame(_ context.Context, _ string, at time.Time, _ []time.Time, _ radar.Box) ([]byte, error) {
	return p.pick(at), nil
}

// solid is a w x w picture, painted or clear.
func solid(t *testing.T, w int, painted bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, w))
	if painted {
		for i := range img.Pix {
			img.Pix[i] = 200
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// EACH FRAME'S CHECK IS ITS OWN: kept by its time, one frame's answer never
// stands for another's; and by its size, a frame whose picture changed is
// checked again.
func TestEachFramesCheckIsItsOwn(t *testing.T) {
	times := grid5(24, time.Date(2026, 9, 28, 14, 0, 0, 0, time.UTC))
	painted, clear, bigger := solid(t, 8, true), solid(t, 8, false), solid(t, 9, true)
	// THE SAME SIZE, so only the time tells them apart: a decoder stops at the
	// picture's end, and the shorter is padded after it.
	n := max(len(painted), len(clear))
	painted = append(painted, make([]byte, n-len(painted))...)
	clear = append(clear, make([]byte, n-len(clear))...)
	if len(bigger) == n {
		t.Fatal("the changed picture is the same size; the size half measures nothing")
	}
	src := &pictureRadar{fakeRadar: fakeRadar{name: "MRMS", times: times}, pick: func(at time.Time) []byte {
		if at.Minute()%10 == 0 {
			return painted
		}
		return clear
	}}
	box := radar.Box{Name: "us", W: -126, S: 23, E: -65, N: 51}
	slots := loopSlots(times, radarStep, radar.Window)
	checks := &frameChecks{}
	_, painting, ok := radarLoop(context.Background(), checks, src, geo.RegionContiguous, times, slots, box)
	if !ok || painting != 12 {
		t.Fatalf("%d of 24 frames paint; want the 12 painted ones, each checked by its own time", painting)
	}
	src.pick = func(time.Time) []byte { return bigger }
	if _, painting, _ := radarLoop(context.Background(), checks, src, geo.RegionContiguous, times, slots, box); painting != 24 {
		t.Errorf("%d of 24 frames paint after every picture changed; want each checked again, by its new size", painting)
	}
}
