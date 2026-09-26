package app

// mapradar.go — the map's radar (0.18.0 W8, FR-5): the loop for the boxes the
// view takes, from MRMS by default or IEM where the listener chose it for the
// lower 48 (D-83), fetched frame by frame through the radar client - memory
// only, capped, public addresses only (W8.5, W8.14) - and handed to the
// window as one image loop per box (FR-5.2).

import (
	"context"
	"sort"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// The radar, registered: on by default - the map shows what is real (D-76).
func init() {
	registerMapLayer(mapLayer{key: tty.RadarLayer, label: "Radar", on: true, cost: radarLayerCost})
}

// radarStep is the loop's step: five minutes (W8.3b's default), 24 frames
// over the two hours.
const radarStep = 5 * time.Minute

// radarFrameBytes is a frame on the wire, measured: 12 to 30 KB at the
// boxes' sizes (wave 1, and the fixtures of 2026-09-26); the estimate's unit.
const radarFrameBytes = 25_000

// radarSources is the radar the app holds: the two sources over the radar
// client.
type radarSources struct {
	iem, mrms radar.Source
}

// newRadarSources builds both over one radar client.
func newRadarSources(userAgent string) (*radarSources, error) {
	c, err := radar.NewClient(userAgent)
	if err != nil {
		return nil, err
	}
	return &radarSources{iem: radar.NewIEM(c, ""), mrms: radar.NewMRMS(c, "")}, nil
}

// sourceFor is the source for a region: IEM where the listener chose it and
// it covers, MRMS otherwise (D-83); nil where none covers.
func (rs *radarSources) sourceFor(region string, iem bool) radar.Source {
	if iem && rs.iem.Covers(region) {
		return rs.iem
	}
	if rs.mrms.Covers(region) {
		return rs.mrms
	}
	return nil
}

// radarNotes are the words beside the loop: MRMS's approximate colours
// (W8.15a), and a region no radar covers (D-83).
const (
	mrmsNote    = "MRMS radar's colours are approximate: its scale is read from its legend."
	noRadarNote = "No radar covers "
)

// mapRadar is the window's radar: the newest frame alone, or the loop.
func (lp *livePipelines) mapRadar(ctx context.Context, ask tty.MapAsk, newestOnly bool) tty.MapRadar {
	if lp.radar == nil || ask.Region == "" {
		return tty.MapRadar{}
	}
	src := lp.radar.sourceFor(ask.Region, ask.RadarIEM)
	if src == nil {
		return tty.MapRadar{Note: noRadarNote + ask.Region + "."}
	}
	out := tty.MapRadar{Source: src.Name()}
	if src.Name() == "MRMS" {
		out.Note = mrmsNote
	}
	times, err := src.Times(ctx, ask.Region)
	if err != nil || len(times) == 0 {
		out.Note = "Radar is unavailable: " + src.Name() + " did not answer."
		return out
	}
	slots := loopSlots(times, radarStep, radar.Window)
	if newestOnly {
		slots = slots[len(slots)-1:]
	}
	allEmpty := true
	boxes := radar.BoxesFor(ask.Region, ask.View)
	for _, b := range boxes {
		o, painted, ok := radarLoop(ctx, src, ask.Region, times, slots, b)
		if ok {
			out.Overlays = append(out.Overlays, o)
		}
		allEmpty = allEmpty && painted == 0
	}
	if allEmpty && !newestOnly && len(out.Overlays) > 0 {
		if other := lp.radar.other(src, ask.Region); other != nil && echoes(ctx, other, ask.Region, boxes) {
			out.Note = src.Name() + " shows no echo where " + other.Name() + " does: its data may be missing." // D-84's check
		}
	}
	return out
}

// other is the region's other source, when it has one: the check on a loop
// that shows nothing (D-84).
func (rs *radarSources) other(src radar.Source, region string) radar.Source {
	for _, s := range []radar.Source{rs.iem, rs.mrms} {
		if s.Name() != src.Name() && s.Covers(region) {
			return s
		}
	}
	return nil
}

// echoes reports whether a source's newest frame paints anything in any of
// the boxes: one request a box, made only when a whole loop showed nothing.
func echoes(ctx context.Context, src radar.Source, region string, boxes []radar.Box) bool {
	times, err := src.Times(ctx, region)
	if err != nil || len(times) == 0 {
		return false
	}
	for _, b := range boxes {
		png, err := src.Frame(ctx, region, times[len(times)-1], times, b)
		if err != nil {
			continue
		}
		if empty, err := radar.Check(png); err == nil && !empty {
			return true
		}
	}
	return false
}

// slot is one moment of the loop and the advertised time that fills it, zero
// where the source holds none (a gap, stated, never filled from a neighbour).
type slot struct {
	at, time time.Time
}

// loopSlots is the loop's moments, oldest first: one every step back from the
// newest advertised time, over the window, each filled by the newest
// advertised time at or before it and within a step of it. IEM's grid fills
// every slot exactly; MRMS's two-minute cadence fills each with its nearest.
func loopSlots(times []time.Time, step, window time.Duration) []slot {
	sorted := append([]time.Time(nil), times...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Before(sorted[j]) })
	newest := sorted[len(sorted)-1]
	n := int(window / step)
	out := make([]slot, 0, n)
	for k := n - 1; k >= 0; k-- {
		at := newest.Add(-time.Duration(k) * step)
		s := slot{at: at}
		for i := len(sorted) - 1; i >= 0; i-- {
			if !sorted[i].After(at) {
				if at.Sub(sorted[i]) < step {
					s.time = sorted[i]
				}
				break
			}
		}
		out = append(out, s)
	}
	return out
}

// radarLoop fetches a box's frames, newest first so the picture is current
// soonest, and builds its loop; a frame that fails or is refused is a gap.
// painted counts the frames with echo: an empty frame is a real, clear one
// (D-84).
func radarLoop(ctx context.Context, src radar.Source, region string, advertised []time.Time, slots []slot, b radar.Box) (o tuimaps.Overlay, painted int, ok bool) {
	frames := make([]tuimaps.LoopFrame, len(slots))
	held := 0
	for i := len(slots) - 1; i >= 0; i-- {
		s := slots[i]
		frames[i] = tuimaps.LoopFrame{Valid: s.at, Gap: true}
		if s.time.IsZero() || ctx.Err() != nil {
			continue
		}
		png, err := src.Frame(ctx, region, s.time, advertised, b)
		if err != nil {
			continue
		}
		empty, err := radar.Check(png)
		if err != nil {
			continue // too large, or no picture at all: refused undecoded (W8.5)
		}
		frames[i] = tuimaps.LoopFrame{Valid: s.at, PNG: png}
		held++
		if !empty {
			painted++
		}
	}
	if held == 0 {
		return tuimaps.Overlay{}, 0, false
	}
	provider := tuimaps.ProviderMRMS
	if src.Name() == "IEM" {
		provider = tuimaps.ProviderIEM
	}
	newest := slots[len(slots)-1].at
	return tuimaps.RadarImage(tty.RadarLayer+"/"+b.Name, tuimaps.Image{Frames: frames, Provider: provider,
		West: b.W, South: b.S, East: b.E, North: b.N, Projection: tuimaps.PlateCarree}, newest), painted, true
}

// radarLayerCost is what the radar would fetch in a refresh as if nothing
// were held: every frame of every box the view takes, and the time list.
func radarLayerCost(in mapInputs) (int64, int) {
	if in.region == "" {
		return 0, 0
	}
	boxes := len(radar.BoxesFor(in.region, in.view))
	if boxes == 0 {
		return 0, 0
	}
	frames := int(radar.Window / radarStep)
	return int64(boxes*frames) * radarFrameBytes, boxes*frames + 1
}

// radarHosts are the radar's entries for the Status window's MAP block.
func radarHosts() []tty.MapSource {
	var out []tty.MapSource
	for name, base := range radar.Hosts() {
		out = append(out, tty.MapSource{Name: name, Host: hostOf(base), Use: "radar frames of fixed boxes around the region shown (never the view itself)"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
