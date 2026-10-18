package main

// score.go - the score command: the replay run through go-ionomaps once
// for each column, each drawn scenario answered three ways and read by
// D-220 (C1), and the agreement tallied.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/propanswers"
)

// column is one way a scenario is answered.
type column uint8

// The columns (`m1-m4-instrument.md`, "The columns").
const (
	climatology column = iota // no GloTEC, no stations: PyIRI's climatology as go-ionomaps carries it
	background                // GloTEC and the climatology, no stations (B-F7, D-214)
	shipped                   // GloTEC, the climatology and GIRO's stations, as watchpost answers
	columns
)

// String is the column's name.
func (c column) String() string {
	return [...]string{"climatology", "background", "shipped", "?"}[min(c, columns)]
}

// The scored hours: every third UTC hour, when PLAN's week has a grid, the
// clock 30 minutes after the grid's valid time - about when NOAA publishes
// it - so the grid is the newest and under go-ionomaps' hour.
const (
	scoredEvery  = 3 * time.Hour
	clockPastDay = 35 * time.Minute
)

// clocks are the scored clocks of the days, in order.
func clocks(ds []time.Time) []time.Time {
	if len(ds) == 0 {
		return nil
	}
	if len(ds) > maxDays {
		return nil
	}
	var out []time.Time
	for _, d := range ds {
		for h := time.Duration(0); h < 24*time.Hour; h += scoredEvery {
			out = append(out, d.Add(h+clockPastDay))
		}
	}
	return out
}

// hourFacts are what fed a column's snapshot at a scored clock.
type hourFacts struct {
	Grid     bool
	Stations int
	F107     float32
}

// errUnknownAsks is a run in which go-ionomaps asked an address the replay
// does not answer.
var errUnknownAsks = errors.New("go-ionomaps asked addresses the replay does not answer")

// errReplayed is a column whose snapshots are not what the column is: a
// grid in the climatology's, or a station's reading where none is asked.
var errReplayed = errors.New("a column's snapshot is fed other than the column says")

// snapshots runs the days through one library for a column, an update an
// hour as a host left running would ask - the stations' burst after
// NOAA's when it is due - and keeps the snapshot and its facts at each
// scored clock.
func snapshots(in *recorded, col column, at []time.Time) (map[time.Time]ionomaps.Snapshot, map[time.Time]hourFacts, error) {
	if in == nil || len(at) == 0 {
		return nil, nil, errUsage
	}
	bounded := len(at) <= maxDays*8 // the hourly walk below is at most maxDays days long
	if err := invariant.Check(bounded, "more scored clocks than maxDays days hold"); err != nil {
		return nil, nil, err
	}
	rp := replayFor(in, col)
	lib, err := ionomaps.New(rp, ionomaps.Options{Clock: rp.clock})
	if err != nil {
		return nil, nil, err
	}
	opts := ionomaps.UpdateOptions{Noise: ionomaps.NoiseResidential, NoStations: col != shipped} // the Settings' defaults, GloTEC as is (D-105)
	scored := map[time.Time]bool{}
	for _, t := range at {
		scored[t] = true
	}
	snaps, facts := map[time.Time]ionomaps.Snapshot{}, map[time.Time]hourFacts{}
	ctx := context.Background()
	for t := at[0]; !t.After(at[len(at)-1]); t = t.Add(time.Hour) {
		rp.now = t
		s, err := lib.Update(ctx, opts)
		if err == nil && s.Reason == ionomaps.StationsNext {
			s, err = lib.Update(ctx, opts)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("%s at %s: %w", col, t.Format(time.RFC3339), err)
		}
		if scored[t] {
			snaps[t] = s
			facts[t] = hourFacts{Grid: !s.Inputs.GloTECValid.IsZero(), Stations: s.Inputs.Stations, F107: s.Inputs.F107Mean}
		}
	}
	if len(rp.unknown) > 0 {
		return nil, nil, fmt.Errorf("%w: %v", errUnknownAsks, rp.unknown)
	}
	return snaps, facts, fedAsSaid(col, facts)
}

// replayFor is a column's replay: the climatology's withholds GloTEC.
func replayFor(in *recorded, col column) *replay {
	return &replay{in: in, grids: col != climatology}
}

// fedAsSaid checks a column's snapshots were fed as the column says: the
// climatology's never by a grid, no column but the shipped build's by a
// station.
func fedAsSaid(col column, facts map[time.Time]hourFacts) error {
	if col >= columns {
		return errReplayed
	}
	for t, f := range facts {
		gridded := col == climatology && f.Grid
		stationed := col != shipped && f.Stations > 0
		if gridded || stationed {
			return fmt.Errorf("%w: %s at %s", errReplayed, col, t.Format(time.RFC3339))
		}
	}
	return nil
}

// reaches is D-220's reading (C1), the product's own: a band open or
// disturbed reaches (`propanswers.Usable`); absorbed, too weak and above the
// upper limit do not.
func reaches(b ionomaps.BandStatus) bool {
	if b.Band < ionomaps.Band160m || b.Band > ionomaps.Band10m {
		return false // a status for no band reaches nowhere
	}
	return len(propanswers.Usable([]ionomaps.BandStatus{b})) == 1
}

// errNoBand is an answer that does not name the scenario's band.
var errNoBand = errors.New("the answer names no status for the scenario's band")

// answerOne is a scenario's answer from a snapshot, as watchpost asks it:
// M1 from the tower, the place within its served area answered by the
// area (FR-2.3); M4 from the listener, the transmitter as a found place
// (D-204), within the near-vertical radius answered by the area. noData is
// a band the field had no value for.
func answerOne(snap ionomaps.Snapshot, s scenario, r reach) (reached, noData bool, err error) {
	radius := r.NearKm
	if s.Metric == m1 {
		radius = r.ServedKm
	}
	chart, err := propanswers.WorkChart(snap, propanswers.Ask{Origin: s.Origin, RadiusKm: radius, Targets: []propanswers.Target{{At: s.Target}}})
	if err != nil {
		return false, false, err
	}
	if len(chart.Targets) != 1 {
		return false, false, propanswers.ErrAnswers
	}
	for _, b := range chart.Targets[0].Bands {
		if b.Band == s.Band {
			return reaches(b), b.Status == ionomaps.StatusNoData, nil
		}
	}
	return false, false, errNoBand
}

// tally is one column's agreement over some scenarios, by outcome.
type tally struct {
	n, agree, noData [outcomes]int
}

// add counts one answer against its scenario's outcome.
func (t *tally) add(o outcome, reached, noData bool) {
	if t == nil {
		return
	}
	if o >= outcomes {
		return
	}
	t.n[o]++
	if reached == (o == open) {
		t.agree[o]++
	}
	if noData {
		t.noData[o]++
	}
}

// total is the tally's scenarios and agreements over every outcome.
func (t tally) total() (n, agree int) {
	for o := range outcomes {
		n, agree = n+t.n[o], agree+t.agree[o]
	}
	return n, agree
}

// scored is the scenarios answered: each metric's band's tally per
// column, and each scenario's answers, for the record kept outside the
// tree.
type scored struct {
	tallies map[base]*[columns]tally
	answers [][columns]bool
}

// scoreSet answers each scenario three ways.
func scoreSet(set []scenario, snaps [columns]map[time.Time]ionomaps.Snapshot, r reach) (scored, error) {
	out := scored{tallies: map[base]*[columns]tally{}, answers: make([][columns]bool, len(set))}
	if len(set) == 0 {
		return out, nil
	}
	for i, s := range set {
		b := base{s.Metric, s.Band}
		if out.tallies[b] == nil {
			out.tallies[b] = &[columns]tally{}
		}
		for c := range columns {
			snap, held := snaps[c][s.At]
			if !held {
				return scored{}, fmt.Errorf("%s has no snapshot at %s", c, s.At.Format(time.RFC3339))
			}
			reached, noData, err := answerOne(snap, s, r)
			if err != nil {
				return scored{}, fmt.Errorf("%s, %s at %s: %w", s.Metric, c, s.At.Format(time.RFC3339), err)
			}
			out.tallies[b][c].add(s.Outcome, reached, noData)
			out.answers[i][c] = reached
		}
	}
	return out, nil
}

// scoreFlags are the score command's settings.
type scoreFlags struct {
	inputs, solar, spots, out string
	from, to                  string
	seed                      uint64
	k                         int
	strata                    string
	r                         reach
}

// The draw's strata: by band alone, the hours as they fall (D-224) - which
// keeps the hour's signal, the one an answer constant across a served area
// within the hour can carry - or by band and UTC hour, as D-219 first
// ruled.
const (
	strataBandHour = "band-hour"
	strataBand     = "band"
)

// maxK is the most of each outcome a stratum may keep: past it the answers
// take hours, and B1 needs tens.
const maxK = 1000

// maxRadiusKm is the largest radius go-ionomaps answers an area for.
const maxRadiusKm = 2000

// errReach is a reach that cannot be drawn: a served area no wider than
// the ground wave's reach, a region or radius of nothing, or a radius past
// go-ionomaps' 2000 km.
var errReach = errors.New("the reach flags: -sky-km at least 0, -served-km past it, -region-km and -near-km above 0, the radii at most 2000 km")

// parseScoreFlags reads the score command's flags.
func parseScoreFlags(args []string) (scoreFlags, error) {
	var f scoreFlags
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.StringVar(&f.inputs, "inputs", "", "folder of the days' GloTEC grids and GIRO replies (outside the tree)")
	fs.StringVar(&f.solar, "solar", "", "the days' daily-solar-indices.txt")
	fs.StringVar(&f.spots, "spots", "", "folder of the days' spots, as the spots command keeps them")
	fs.StringVar(&f.out, "out", "", "folder outside the tree for the scenarios and the inputs' checksums")
	fs.StringVar(&f.from, "from", "", "first UTC day, YYYY-MM-DD")
	fs.StringVar(&f.to, "to", "", "last UTC day, YYYY-MM-DD")
	fs.Uint64Var(&f.seed, "seed", 19, "the draw's seed")
	fs.IntVar(&f.k, "k", 80, "at most this many of each outcome a stratum")
	fs.StringVar(&f.strata, "strata", strataBand, "the draw's strata: "+strataBand+" (D-224) or "+strataBandHour+" (D-219 as first ruled)")
	fs.Float64Var(&f.r.SkyKm, "sky-km", defaultSkyKm, "pairs nearer than this are the ground wave's, not drawn")
	fs.Float64Var(&f.r.ServedKm, "served-km", defaultServedKm, "M1: the service radius, the served area's places within it")
	fs.Float64Var(&f.r.RegionKm, "region-km", defaultRegionKm, "M4: a continent's region, transmitters within this of its reference point")
	fs.Float64Var(&f.r.NearKm, "near-km", defaultNearKm, "M4: the near-vertical radius the listener is answered with")
	if err := fs.Parse(args); err != nil {
		return f, err
	}
	if f.inputs == "" || f.solar == "" || f.spots == "" || f.out == "" {
		return f, errUsage
	}
	if f.k < 1 || f.k > maxK {
		return f, errUsage
	}
	if f.strata != strataBandHour && f.strata != strataBand {
		return f, errUsage
	}
	if !(f.r.SkyKm >= 0) || !(f.r.ServedKm > f.r.SkyKm) || !(f.r.RegionKm > 0) || !(f.r.NearKm > 0) || f.r.ServedKm > maxRadiusKm || f.r.NearKm > maxRadiusKm {
		return f, errReach
	}
	return f, nil
}

// scoreCommand draws, answers and scores the days, and prints the table.
func scoreCommand(args []string, out io.Writer) error {
	f, err := parseScoreFlags(args)
	if err != nil {
		return err
	}
	ds, err := days(f.from, f.to)
	if err != nil {
		return err
	}
	in, err := loadRecorded(f.inputs, f.solar)
	if err != nil {
		return err
	}
	set, d, spotFiles, err := drawDays(ds, f)
	if err != nil {
		return err
	}
	at := clocks(ds)
	var snaps [columns]map[time.Time]ionomaps.Snapshot
	var facts [columns]map[time.Time]hourFacts
	for c := range columns {
		if snaps[c], facts[c], err = snapshots(in, c, at); err != nil {
			return err
		}
	}
	res, err := scoreSet(set, snaps, f.r)
	if err != nil {
		return err
	}
	files := append(append([]string{}, in.files...), spotFiles...)
	sum, err := checksums(files, filepath.Join(f.out, "inputs.sha256"))
	if err != nil {
		return err
	}
	if err := keepScenarios(filepath.Join(f.out, "scenarios.tsv"), set, res.answers); err != nil {
		return err
	}
	return report(out, reportOf{flags: f, days: ds, set: set, counts: d.counts, res: res, facts: facts,
		grids: len(in.grids), stations: len(in.giro), spotFiles: len(spotFiles), sum: sum, rev: revision()})
}

// drawDays reads each day's spots and draws the set from every scored
// window's pairs, returning the drawer's counts and the spot files read.
func drawDays(ds []time.Time, f scoreFlags) ([]scenario, *drawer, []string, error) {
	if len(ds) == 0 {
		return nil, nil, nil, errDays
	}
	if f.k < 1 {
		return nil, nil, nil, errUsage
	}
	d := newDrawer(f.k, f.seed, f.strata == strataBandHour)
	var files []string
	for _, day := range ds {
		path := spotsFile(f.spots, day)
		spots, err := readSpotsFile(path)
		if err != nil {
			return nil, nil, nil, err
		}
		files = append(files, path)
		byHour := map[time.Time][]spot{}
		for _, s := range spots {
			byHour[s.Hour] = append(byHour[s.Hour], s)
		}
		for _, at := range clocks([]time.Time{day}) {
			h := at.Truncate(time.Hour)
			near := append(append(append([]spot{}, byHour[h.Add(-time.Hour)]...), byHour[h]...), byHour[h.Add(time.Hour)]...)
			for _, b := range wsprBands {
				windowOf(near, at, b.band).classify(at, b.band, f.r, d.offer)
			}
		}
	}
	set, err := d.set()
	return set, d, files, err
}

// readSpotsFile reads a day's spots file.
func readSpotsFile(path string) ([]spot, error) {
	if path == "" {
		return nil, errUsage
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }() // read only: a failed close loses nothing
	spots, err := readSpots(fh)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return spots, nil
}
