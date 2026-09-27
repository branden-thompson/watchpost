package app

// mapradar.go — the map's radar (0.18.0 W8, FR-5): the loop for the boxes the
// view takes, from MRMS by default or IEM where the listener chose it for the
// lower 48 (D-83), fetched frame by frame through the radar client - memory
// only, capped, public addresses only (W8.5, W8.14) - and handed to the
// window as one image loop per box (FR-5.2).

import (
	"bytes"
	"context"
	pngpkg "image/png"
	"sort"
	"strings"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
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
	hrrr      *radar.HRRR // the hours ahead (D-113): the lower 48's
}

// radarSourcesOver is both sources over one radar client (overClient).
func radarSourcesOver(c *httpx.Client) *radarSources {
	return &radarSources{iem: radar.NewIEM(c, ""), mrms: radar.NewMRMS(c, ""), hrrr: radar.NewHRRR(c, "")}
}

// overClient builds a layer's sources over one hardened client of their own:
// the radar's and the temperature's (W8.5, W10.2).
func overClient[T any](newClient func(string) (*httpx.Client, error), userAgent string, build func(*httpx.Client) T) (T, error) {
	c, err := newClient(userAgent)
	if err != nil {
		var none T
		return none, err
	}
	return build(c), nil
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

// mapRadar is the window's radar: the whole loop, which the window shows once
// it is in (D-85).
func (lp *livePipelines) mapRadar(ctx context.Context, ask tty.MapAsk) tty.MapRadar {
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
	boxes := radar.BoxesFor(ask.Region, ask.View)
	allEmpty := true
	for _, b := range boxes {
		o, painted, ok := radarLoop(ctx, src, ask.Region, times, slots, b)
		if ok {
			out.Overlays = append(out.Overlays, o)
		}
		allEmpty = allEmpty && painted == 0
	}
	out.Overlays = trimToBudget(out.Overlays)
	if ask.RadarAhead > 0 && len(out.Overlays) > 0 {
		newest, until := slots[len(slots)-1].at, time.Now().Add(time.Duration(ask.RadarAhead)*time.Hour)
		switch {
		case lp.radar.hrrr != nil && lp.radar.hrrr.Covers(ask.Region):
			out = withForecast(ctx, out, lp.radar.hrrr, boxes, newest, until)
		case lp.temp != nil && lp.temp.rain != nil: // where HRRR is not, a model's rain (D-115)
			out = withModelRain(ctx, out, lp.temp.rain, ask.Region, ask.View, newest, until)
		}
	}
	if allEmpty && len(out.Overlays) > 0 {
		if other := lp.radar.other(src, ask.Region); other != nil && echoes(ctx, other, ask.Region, boxes) {
			out.Note = src.Name() + " shows no echo where " + other.Name() + " does: its data may be missing." // D-84's check
		}
	}
	return out
}

// withForecast adds the loop's hours ahead (D-113): a forecast loop a box of
// HRRR's quarter-hours after the newest observed frame and up to the horizon,
// every frame marked forecast, joined beside the observed (joinAhead).
func withForecast(ctx context.Context, out tty.MapRadar, h *radar.HRRR, boxes []radar.Box, newest, until time.Time) tty.MapRadar {
	run, err := h.Run(ctx)
	if err != nil {
		out.Note = strings.TrimPrefix(out.Note+" The radar's hours ahead are unavailable: HRRR did not answer.", " ")
		return out
	}
	minutes := radar.Minutes(run, newest, until)
	if len(minutes) == 0 {
		return out
	}
	var loops []tuimaps.Overlay
	for _, b := range boxes {
		frames := make([]tuimaps.LoopFrame, len(minutes))
		held := 0
		for i, m := range minutes {
			at := run.Add(time.Duration(m) * time.Minute)
			frames[i] = tuimaps.LoopFrame{Valid: at, Gap: true, Forecast: true}
			if ctx.Err() != nil {
				continue
			}
			png, err := h.Frame(ctx, m, b)
			if err != nil {
				continue
			}
			if _, err := radar.Check(png); err != nil {
				continue // refused undecoded (W8.5)
			}
			frames[i] = tuimaps.LoopFrame{Valid: at, PNG: png, Forecast: true}
			held++
		}
		if held == 0 {
			continue
		}
		o := tuimaps.RadarImage(tty.RadarLayer+"/fc-"+b.Name, tuimaps.Image{Frames: frames, Provider: tuimaps.ProviderIEM,
			West: b.W, South: b.S, East: b.E, North: b.N, Projection: tuimaps.PlateCarree}, run)
		o.Keeps = until.Sub(run) + time.Hour // a run's frames are current until the horizon has passed
		o.During = tuimaps.Span{From: frames[0].Valid}
		loops = append(loops, o)
	}
	if len(loops) == 0 {
		return out
	}
	return joinAhead(out, loops, newest, h.Name())
}

// chargeOf is what loops cost the image budget as fetched: each frame's PNG
// and a byte a pixel.
func chargeOf(loops []tuimaps.Overlay) int64 {
	total := int64(0)
	for _, o := range loops {
		for _, f := range o.Image.Frames {
			if !f.Gap {
				total += int64(len(f.PNG)) + pixelsOf(f.PNG)
			}
		}
	}
	return total
}

// trimForecast drops the farthest forecast frames, the same number from every
// loop, until the loops fit the room left: the nearest hours matter most.
func trimForecast(loops []tuimaps.Overlay, room int64) []tuimaps.Overlay {
	for chargeOf(loops) > room {
		longest := 0
		for _, o := range loops {
			longest = max(longest, len(o.Image.Frames))
		}
		if longest <= 1 {
			return nil
		}
		for i, o := range loops {
			img := *o.Image
			img.Frames = img.Frames[:max(len(img.Frames)-1, 1)]
			o.Image = &img
			loops[i] = o
		}
	}
	return loops
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

// radarBudgetShare is how much of the map's image budget the loops may take:
// four fifths, the rest the library's shared readings and a replaced loop's
// spare frames.
const radarBudgetShare = radarImageBudget * 4 / 5

// trimToBudget drops the oldest frames, the same number from every loop,
// until the loops as fetched fit the budget share: the estimate is only an
// estimate, and a loop over the budget is refused whole (UAT-2 U2-5).
func trimToBudget(loops []tuimaps.Overlay) []tuimaps.Overlay {
	charge := func(drop int) int64 {
		total := int64(0)
		for _, o := range loops {
			for _, f := range o.Image.Frames[min(drop, len(o.Image.Frames)):] {
				if !f.Gap {
					total += int64(len(f.PNG)) + pixelsOf(f.PNG)
				}
			}
		}
		return total
	}
	longest := 0
	for _, o := range loops {
		longest = max(longest, len(o.Image.Frames))
	}
	drop := 0
	for drop < longest-1 && charge(drop) > radarBudgetShare {
		drop++
	}
	if drop == 0 {
		return loops
	}
	out := make([]tuimaps.Overlay, 0, len(loops))
	for _, o := range loops {
		img := *o.Image
		img.Frames = img.Frames[min(drop, len(img.Frames)-1):]
		o.Image = &img
		out = append(out, o)
	}
	return out
}

// pixelsOf is a PNG's pixels from its header, nothing decoded.
func pixelsOf(png []byte) int64 {
	cfg, err := pngpkg.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		return 0
	}
	return int64(cfg.Width) * int64(cfg.Height)
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
	bytes, requests := int64(boxes*frames)*radarFrameBytes, boxes*frames+1
	switch {
	case in.ahead == 0:
	case in.region == geo.RegionContiguous: // HRRR's quarter-hours, a quarter of a radar frame's size (D-114)
		ahead := in.ahead * int(time.Hour/radar.HRRRStep)
		bytes, requests = bytes+int64(boxes*ahead)*radarFrameBytes/4, requests+boxes*ahead
	default: // Open-Meteo's hours, a request a field box (D-115)
		fields := len(fieldBoxes(in.region, in.view))
		bytes, requests = bytes+int64(fields)*rainHoursBytes, requests+fields
	}
	return bytes, requests
}

// radarHosts are the radar's entries for the Status window's MAP block.
func radarHosts() []tty.MapSource {
	return hostsFor(radar.Hosts(), "radar frames of fixed boxes around the region shown (never the view itself)")
}

// hostsFor is a layer's hosts as the Status window's entries, by name, each
// with what it is sent.
func hostsFor(hosts map[string]string, use string) []tty.MapSource {
	var out []tty.MapSource
	for name, base := range hosts {
		out = append(out, tty.MapSource{Name: name, Host: hostOf(base), Use: use})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
