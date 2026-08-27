package app

// Diagnostics the app assembles for the [S] modal and the dump (quality
// pass Q0, plan §2.1). Every bounded structure in the process is listed
// here as a Gauge with its current length and bytes, so a soak can prove
// "flat" per structure instead of reading one footprint number (red-team
// BQ-1, RT-5). Adding a structure = adding one line to gauges(); its bound
// lives with its owner (§0.8).

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/marine/coops"
	"github.com/branden-thompson/watchpost/domains/seismic/usgs"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// Gauge is one structure's size at a point in time. Len counts items;
// Bytes is the payload size when the owner can say it cheaply, else 0.
type Gauge struct {
	Name  string `json:"name"`
	Len   int    `json:"len"`
	Bytes int64  `json:"bytes"`
}

// diagSources is what the gauges read: the clients, the providers that
// memoise, the assemblers and the radio deck. Nil members are skipped.
type diagSources struct {
	clients     []*httpx.Client
	weather     *nws.Provider
	tides       *coops.Provider
	hms         *hms.Provider
	wfigs       *wfigs.Provider
	firms       *firms.Provider
	usgs        *usgs.Provider
	priority    *snapshot.Assembler
	recent      *snapshot.Assembler
	priorityPub *publisher
	recentPub   *publisher
	deck        *radioDeck
	// The three directories the disk gauges size ("" = skipped): the HTTP
	// cache (flat), the profiles dir and the voices dir (nested).
	cacheDir, profilesDir, voicesDir string
}

// gauges lists every bounded structure with its current size.
func (d diagSources) gauges() []Gauge {
	var mem, neg Gauge
	for _, c := range d.clients {
		if c == nil {
			continue
		}
		cs := c.CacheStats()
		mem.Len, mem.Bytes = mem.Len+cs.Entries, mem.Bytes+int64(cs.Bytes)
		neg.Len += cs.Negative
	}
	mem.Name, neg.Name = "httpx.mem.entries", "httpx.neg.entries"
	var writes Gauge
	for _, c := range d.clients {
		if c != nil {
			writes.Len += int(c.CacheStats().DiskWrites)
		}
	}
	writes.Name = "httpx.disk.writes" // files written since launch (Q1 gate: the derived rate)
	out := []Gauge{mem, neg, writes}
	if d.weather != nil {
		out = append(out, Gauge{Name: "nws.gridinfo", Len: d.weather.CachedGrids()}, Gauge{Name: "nws.grid.decodes", Len: d.weather.GridDecodes()}) // decodes since launch: one per grid body change (Q5)
	}
	if d.tides != nil {
		out = append(out, Gauge{Name: "coops.stations", Len: d.tides.CachedStations()})
	}
	if d.hms != nil {
		pts, parses := d.hms.MemoStats()
		out = append(out, Gauge{Name: "hms.memo.points", Len: pts}, Gauge{Name: "hms.memo.parses", Len: parses}) // parses since launch: the parse-spike counter (plan §1, Q3)
	}
	if d.firms != nil {
		tiles, parses := d.firms.MemoStats()
		out = append(out, Gauge{Name: "firms.memo.tiles", Len: tiles}, Gauge{Name: "firms.memo.parses", Len: parses}) // bound: maxTiles (Q5)
	}
	if d.wfigs != nil {
		ins, parses := d.wfigs.MemoStats()
		out = append(out, Gauge{Name: "wfigs.memo.incidents", Len: ins}, Gauge{Name: "wfigs.memo.parses", Len: parses}) // bound: the last layer (Q3)
	}
	if d.usgs != nil {
		boxes, parses := d.usgs.MemoStats()
		out = append(out, Gauge{Name: "usgs.memo.boxes", Len: boxes}, Gauge{Name: "usgs.memo.parses", Len: parses}) // bound: maxBoxes; shared regional box ⇒ parses ≪ locations (seismic P2)
	}
	out = append(out, assemblerGauges("priority", d.priority)...)
	out = append(out, assemblerGauges("recent", d.recent)...)
	if src := d.deck.synthSource(); src != nil {
		n, b := src.Cached()
		out = append(out, Gauge{Name: "synth.pcm.cache", Len: n, Bytes: int64(b)})
	}
	out = append(out, Gauge{Name: "tz.cache", Len: tz.Cached()})
	out = append(out,
		dirGauge("disk.cache", d.cacheDir, false),
		dirGauge("disk.profiles", d.profilesDir, true),
		dirGauge("disk.voices", d.voicesDir, true))
	return out
}

func assemblerGauges(name string, asm *snapshot.Assembler) []Gauge {
	if asm == nil {
		return nil
	}
	locs, warns := asm.Size()
	return []Gauge{{Name: "assembler." + name + ".locations", Len: locs}, {Name: "assembler." + name + ".warnings", Len: warns}}
}

// dirGauge sizes a directory: file count and bytes, recursing only when
// asked (the cache dir is flat and large; the voices dir is nested and
// small). A missing directory is a zero gauge, never an error.
func dirGauge(name, dir string, recursive bool) Gauge {
	g := Gauge{Name: name}
	if dir == "" {
		return g
	}
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return g
		}
		for _, e := range entries {
			if info, ierr := e.Info(); ierr == nil && info.Mode().IsRegular() {
				g.Len++
				g.Bytes += info.Size()
			}
		}
		return g
	}
	_ = filepath.WalkDir(dir, func(_ string, e os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: count what we can
		}
		if info, ierr := e.Info(); ierr == nil && info.Mode().IsRegular() {
			g.Len++
			g.Bytes += info.Size()
		}
		return nil
	})
	return g
}

// runtimeCounts is the process-level trio every soak row carries: live
// goroutines, OS threads ever created (Go never retires one, so this is
// the ratchet LR-3 describes), and open file descriptors (-1 when the OS
// offers no /dev/fd listing).
func runtimeCounts() (goroutines, threads, fds int) {
	goroutines = runtime.NumGoroutine()
	if p := pprof.Lookup("threadcreate"); p != nil {
		threads = p.Count()
	}
	fds = -1
	if entries, err := os.ReadDir("/dev/fd"); err == nil {
		fds = len(entries)
	}
	return goroutines, threads, fds
}

// ttyStats is the [S] modal's view: merged request counters, publish
// counters per pipeline, and the last dump's outcome.
func (lp *livePipelines) ttyStats() tty.Stats {
	src := lp.sources()
	st := tty.Stats{Requests: requestStats(src.clients)}
	for i, pub := range [...]*publisher{src.priorityPub, src.recentPub} {
		if pub != nil {
			st.Pipelines[i] = pub.stats()
		}
	}
	if lp.dump != nil {
		st.LastDump, st.DumpHint = lp.dump.note(), lp.dump.hint()
	}
	return st
}

// sources gathers the live diagnostic sources (the pipelines are wired
// after lp is built, so this is read on demand, never cached).
func (lp *livePipelines) sources() diagSources {
	src := diagSources{clients: lp.clients, weather: lp.weather, tides: lp.tides, deck: lp.deck,
		cacheDir: cacheDir(), profilesDir: userCacheSubdir("profiles"), voicesDir: voiceDir()}
	for _, pr := range lp.fire {
		if h, ok := pr.(*hms.Provider); ok {
			src.hms = h
		}
		if w, ok := pr.(*wfigs.Provider); ok {
			src.wfigs = w
		}
		if f, ok := pr.(*firms.Provider); ok {
			src.firms = f
		}
	}
	for _, pr := range lp.seismic {
		if u, ok := pr.(*usgs.Provider); ok {
			src.usgs = u
		}
	}
	if lp.priority != nil {
		src.priority, src.priorityPub = lp.priority.asm, lp.priority.pub
	}
	if lp.recent != nil {
		src.recent, src.recentPub = lp.recent.asm, lp.recent.pub
	}
	return src
}

func requestStats(clients []*httpx.Client) httpx.RequestStats {
	all := make([]httpx.RequestStats, 0, len(clients))
	for _, c := range clients {
		if c != nil {
			all = append(all, c.RequestStats())
		}
	}
	return httpx.MergeRequestStats(all...)
}

// synthSource is the running synthesized broadcast, nil when the deck is
// absent or a relay is playing.
func (d *radioDeck) synthSource() interface{ Cached() (int, int) } {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.source == nil {
		return nil
	}
	return d.source
}
