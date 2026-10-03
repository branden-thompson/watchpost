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
	"sync"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/radar"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/agememo"
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

// radarParallel is the most frames fetched at once: six, as the zones are
// (D-46, NFR-4; D-130).
const radarParallel = 6

// radarFrameBytes is a frame on the wire, measured: 12 to 30 KB at the
// boxes' sizes (wave 1, and the fixtures of 2026-09-26); the estimate's unit.
const radarFrameBytes = 25_000

// radarSources is the radar the app holds: the two sources over the radar
// client.
type radarSources struct {
	iem, mrms radar.Source
	hrrr      *radar.HRRR // the hours ahead (D-113): the lower 48's
	ahead     aheadFetch  // HRRR's hours ahead, fetched apart from any one ask (D-204)
	checks    frameChecks // each frame's check, kept by the frame (P-17)
}

// The hours ahead's timings (D-204): asked again aheadSoon after an answer
// that owes them, aheadRetry after HRRR failed; a fetch given up after
// aheadTimeout.
const (
	aheadSoon    = 3 * time.Second
	aheadRetry   = 30 * time.Second
	aheadTimeout = 2 * time.Minute
)

// aheadFetch is HRRR's hours ahead for one set of boxes and horizon, fetched
// in the background so the observed loop is never held for them (D-204): an
// ask starts it, and an ask once it has landed joins it.
type aheadFetch struct {
	mu       sync.Mutex
	key      string
	running  bool
	done     bool
	told     bool // its failure told to the diagnostics
	result   hoursAhead
	finished time.Time
	// last is the newest hours ahead that landed whole, and lastGroup the
	// boxes they are for: what a new fetch for the same boxes stands on until
	// it lands (U2-59).
	last      hoursAhead
	lastGroup string
	cancel    context.CancelFunc
	now       func() time.Time // the clock; time.Now when nil
}

// clock is the fetch's clock.
func (a *aheadFetch) clock() time.Time {
	if a.now != nil {
		return a.now()
	}
	return time.Now()
}

// take is what an ask gets of the hours ahead for key, whose boxes are
// group: the loops when they have landed; otherwise how long until it should
// ask again - aheadSoon while they are on their way, what is left of
// aheadRetry after a failure, whose problem is returned the first time - and,
// while they are on their way, the same boxes' last hours ahead to stand in
// (U2-59). A new key, or a failure past aheadRetry, starts a fetch.
func (a *aheadFetch) take(group, key string, fetch func(context.Context) hoursAhead) (fc hoursAhead, again time.Duration, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.key == key && a.done && a.result.problem == "" {
		return a.result, 0, true
	}
	if a.key == key && a.done {
		if wait := aheadRetry - a.clock().Sub(a.finished); wait > 0 {
			if a.told {
				return hoursAhead{}, wait, false
			}
			a.told = true
			return hoursAhead{problem: a.result.problem}, wait, false
		}
	}
	if a.key == key && a.running {
		return a.standIn(group), aheadSoon, false
	}
	if a.cancel != nil {
		a.cancel() // another region's or horizon's: not wanted any more
	}
	ctx, cancel := context.WithTimeout(context.Background(), aheadTimeout)
	a.key, a.running, a.done, a.told, a.cancel = key, true, false, false, cancel
	go func() {
		got := fetch(ctx)
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.key != key {
			return // superseded while it ran
		}
		a.running, a.done, a.result, a.finished = false, true, got, a.clock()
		if got.problem == "" && len(got.loops) > 0 {
			a.last, a.lastGroup = got, group
		}
	}()
	return a.standIn(group), aheadSoon, false
}

// standIn is the last hours ahead that landed for group's boxes, or nothing.
func (a *aheadFetch) standIn(group string) hoursAhead {
	if a.lastGroup != group {
		return hoursAhead{}
	}
	return a.last
}

// radarSourcesOver is both sources over one radar client (overClient).
func radarSourcesOver(c *httpx.Client) *radarSources {
	return &radarSources{iem: radar.NewIEM(c, ""), mrms: radar.NewMRMS(c, ""), hrrr: radar.NewHRRR(c, "")}
}

// overClient builds a layer's sources over one hardened client of their own:
// the radar's and the temperature's (W8.5, W10.2).
func overClient[T any](newClient func(string) (*httpx.Client, error), userAgent string, build func(*httpx.Client) T) (T, *httpx.Client, error) {
	c, err := newClient(userAgent)
	if err != nil {
		var none T
		return none, nil, err
	}
	return build(c), c, nil // the client too: its counters are the Status window's MAP STATUS (D-150)
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
	times, err := src.Times(ctx, ask.Region)
	if err != nil || len(times) == 0 {
		if other := lp.radar.other(src, ask.Region); other != nil { // a Setting draws another (D-124)
			out.Note = "Radar is unavailable: " + src.Name() + " did not answer. Settings → Maps → Radar: " + other.Name() + " draws it instead."
		} else {
			out.Note = "" // nothing the listener can do: the diagnostics' (D-124)
			out.Problems = append(out.Problems, "Radar: "+src.Name()+" did not answer for "+ask.Region)
		}
		return out
	}
	slots := loopSlots(times, radarStep, radar.Window)
	boxes := radar.BoxesFor(ask.Region, ask.View)
	newest := slots[len(slots)-1].at
	until := time.Now().Add(time.Duration(ask.RadarAhead) * time.Hour).Truncate(15 * time.Minute) // HRRR's quarter-hours: one horizon a quarter-hour
	var fc hoursAhead
	var again time.Duration
	hrrr := ask.RadarAhead > 0 && lp.radar.hrrr != nil && lp.radar.hrrr.Covers(ask.Region)
	if hrrr { // asked alongside the observed loop (D-130), never waited for (D-204)
		var names []string
		for _, b := range boxes { // a region's boxes (P10-02)
			names = append(names, b.Name)
		}
		key := strings.Join(names, ",") + "|" + newest.UTC().Format(time.RFC3339) + "|" + until.UTC().Format(time.RFC3339)
		fc, again, _ = lp.radar.ahead.take(strings.Join(names, ","), key, func(ctx context.Context) hoursAhead {
			return fetchForecast(ctx, &lp.radar.checks, lp.radar.hrrr, boxes, newest, until)
		})
	}
	allEmpty := true
	for _, b := range boxes {
		o, painted, ok := radarLoop(ctx, &lp.radar.checks, src, ask.Region, times, slots, b)
		if ok {
			out.Overlays = append(out.Overlays, o)
		}
		allEmpty = allEmpty && painted == 0
	}
	out.Overlays = trimToBudget(out.Overlays)
	if ask.RadarAhead > 0 && len(out.Overlays) > 0 {
		switch {
		case hrrr:
			out = joinForecast(out, fc, newest)
			out.AheadIn = again
		case lp.temp != nil && lp.temp.rain != nil: // where HRRR is not, a model's rain (D-115)
			out = withModelRain(ctx, out, lp.temp.rain, ask.Region, ask.View, newest, until)
		}
	}
	if allEmpty && len(out.Overlays) > 0 {
		if other := lp.radar.other(src, ask.Region); other != nil && echoes(ctx, &lp.radar.checks, other, ask.Region, boxes) {
			out.Note = src.Name() + " shows no echo where " + other.Name() + " does: its data may be missing." // D-84's check
		}
	}
	return out
}

// hoursAhead is HRRR's loops as fetched, or why there are none, and whose.
type hoursAhead struct {
	loops   []tuimaps.Overlay
	problem string
	name    string
}

// joinForecast joins the hours ahead to the observed loops, or says why not.
func joinForecast(out tty.MapRadar, fc hoursAhead, newest time.Time) tty.MapRadar {
	if fc.problem != "" {
		out.Problems = append(out.Problems, fc.problem) // D-124
		return out
	}
	var loops []tuimaps.Overlay
	for _, o := range fc.loops { // a frame at or before the newest is not ahead of it (U2-59)
		img := *o.Image
		img.Frames = nil
		for _, f := range o.Image.Frames {
			if f.Valid.After(newest) {
				img.Frames = append(img.Frames, f)
			}
		}
		if len(img.Frames) > 0 {
			o.Image = &img
			loops = append(loops, o)
		}
	}
	if len(loops) == 0 {
		return out
	}
	return joinAhead(out, loops, newest, fc.name)
}

// fetchForecast is HRRR's loops a box, their frames fetched radarParallel
// at a time (D-130).
func fetchForecast(ctx context.Context, checks *frameChecks, h *radar.HRRR, boxes []radar.Box, newest, until time.Time) hoursAhead {
	run, err := h.Run(ctx)
	if err != nil {
		return hoursAhead{problem: "Radar ahead: HRRR did not answer - " + err.Error()}
	}
	minutes := radar.Minutes(run, newest, until)
	fc := hoursAhead{name: h.Name()}
	if len(minutes) == 0 {
		return fc
	}
	for _, b := range boxes {
		frames := make([]tuimaps.LoopFrame, len(minutes))
		atOnce(len(minutes), func(i int) {
			at := run.Add(time.Duration(minutes[i]) * time.Minute)
			frames[i] = tuimaps.LoopFrame{Valid: at, Gap: true, Forecast: true}
			if ctx.Err() != nil {
				return
			}
			png, err := h.Frame(ctx, run, minutes[i], b)
			if err != nil {
				return
			}
			if _, err := checks.check(ctx, frameKey{source: h.Name(), box: b.Name, at: at, run: run, size: len(png)}, png); err != nil {
				return // refused undecoded (W8.5)
			}
			frames[i] = tuimaps.LoopFrame{Valid: at, PNG: png, Forecast: true}
		})
		held := 0
		for _, f := range frames {
			if !f.Gap {
				held++
			}
		}
		if held == 0 {
			continue
		}
		o := tuimaps.RadarImage(tty.RadarLayer+"/fc-"+b.Name, tuimaps.Image{Frames: frames, Provider: tuimaps.ProviderIEM,
			West: b.W, South: b.S, East: b.E, North: b.N, Projection: tuimaps.PlateCarree}, run)
		o.Keeps = until.Sub(run) + time.Hour // a run's frames are current until the horizon has passed
		o.During = tuimaps.Span{From: frames[0].Valid}
		fc.loops = append(fc.loops, o)
	}
	return fc
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
	longest := 0
	for _, o := range loops {
		longest = max(longest, len(o.Image.Frames))
	}
	for range longest { // a frame off the longest each time: at most its frames (P10-02)
		if chargeOf(loops) <= room {
			return loops
		}
		for i, o := range loops {
			img := *o.Image
			img.Frames = img.Frames[:max(len(img.Frames)-1, 1)]
			o.Image = &img
			loops[i] = o
		}
	}
	if chargeOf(loops) <= room {
		return loops
	}
	return nil // a frame each, and still too big
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
func echoes(ctx context.Context, checks *frameChecks, src radar.Source, region string, boxes []radar.Box) bool {
	times, err := src.Times(ctx, region)
	if err != nil || len(times) == 0 {
		return false
	}
	for _, b := range boxes {
		png, err := src.Frame(ctx, region, times[len(times)-1], times, b)
		if err != nil {
			continue
		}
		at := times[len(times)-1]
		if empty, err := checks.check(ctx, frameKey{source: src.Name(), region: region, box: b.Name, at: at, size: len(png)}, png); err == nil && !empty {
			return true
		}
	}
	return false
}

// frameKey names a radar frame: its source, region, box and time - and for
// HRRR's hours ahead, its run - with its size, a frame at its time never
// changing.
type frameKey struct {
	source, region, box string
	at, run             time.Time
	size                int
}

// frameChecks keeps each frame's check - whether its picture paints anything
// - by the frame (W14 P-17): a loop asked again, on a pan or a refresh,
// decodes only the frames it has not seen. A refusal is not kept. A nil
// frameChecks checks every frame.
type frameChecks struct {
	m lazyMemo[frameKey, bool]
}

// frameCheckRules keep a check for the loop's two hours, for the frames of
// every box a lower-48 view can hold, observed and ahead.
var frameCheckRules = agememo.Options{Fresh: radar.Window, Max: 1024}

// check is whether png paints nothing, kept by its frame.
func (c *frameChecks) check(ctx context.Context, k frameKey, png []byte) (bool, error) {
	if c == nil {
		return radar.Check(png)
	}
	return c.m.memo(frameCheckRules).Do(ctx, k, func() (bool, error) { return radar.Check(png) })
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
	drop := max(longest-1, 0)
	for d := range drop { // the fewest frames dropped that fit, at most all but one (P10-02)
		if charge(d) <= radarBudgetShare {
			drop = d
			break
		}
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
func radarLoop(ctx context.Context, checks *frameChecks, src radar.Source, region string, advertised []time.Time, slots []slot, b radar.Box) (o tuimaps.Overlay, painted int, ok bool) {
	frames := make([]tuimaps.LoopFrame, len(slots))
	blank := make([]bool, len(slots))
	atOnce(len(slots), func(j int) {
		i := len(slots) - 1 - j // the newest first
		s := slots[i]
		frames[i] = tuimaps.LoopFrame{Valid: s.at, Gap: true}
		if s.time.IsZero() || ctx.Err() != nil {
			return
		}
		png, err := src.Frame(ctx, region, s.time, advertised, b)
		if err != nil {
			return
		}
		empty, err := checks.check(ctx, frameKey{source: src.Name(), region: region, box: b.Name, at: s.time, size: len(png)}, png)
		if err != nil {
			return // too large, or no picture at all: refused undecoded (W8.5)
		}
		frames[i], blank[i] = tuimaps.LoopFrame{Valid: s.at, PNG: png}, empty
	})
	held := 0
	for i, f := range frames {
		if !f.Gap {
			held++
			if !blank[i] {
				painted++
			}
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

// atOnce runs do for each of n, radarParallel at a time, and returns when
// every one has (D-130).
func atOnce(n int, do func(i int)) {
	room := make(chan struct{}, radarParallel)
	var wg sync.WaitGroup
	for i := range n {
		room <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-room; wg.Done() }()
			do(i)
		}()
	}
	wg.Wait()
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
	return hostsFor(radar.Hosts(), map[string]string{"NOAA / NCEP (MRMS)": "radar", "Iowa Environmental Mesonet": "radar, radar ahead"},
		map[string][]string{"NOAA / NCEP (MRMS)": {mrmsNote}}) // what never changes about it, off the map (D-132); its chip reads MRMS≈
}

// withChips is chips with a layer's set, made if there were none.
func withChips(chips map[string][]string, layer string, names ...string) map[string][]string {
	if chips == nil {
		chips = map[string][]string{}
	}
	chips[layer] = names
	return chips
}

// hostsFor is a layer's hosts as the Status window's entries, by name, each
// with the layers it serves and what never changes about its data (D-132).
func hostsFor(hosts, layers map[string]string, notes map[string][]string) []tty.MapSource {
	var out []tty.MapSource
	for name, base := range hosts {
		out = append(out, tty.MapSource{Name: name, Host: hostOf(base), Layers: layers[name], Notes: notes[name]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
