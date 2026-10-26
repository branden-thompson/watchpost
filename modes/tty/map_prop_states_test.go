package tty

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
	"github.com/branden-thompson/go-ionomaps/ionomapstest"
)

// statesDay is the replayed days' first day.
var statesDay = time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

// statesStations are four of the library's seeded live stations, at their
// published places, so an update asks GIRO for each.
var statesStations = []ionomapstest.Station{
	{Code: "AL945", Lat: 45.1, Lon: -83.6, FoF2: 6, M3000: 3},
	{Code: "EG931", Lat: 30.5, Lon: -86.5, FoF2: 6, M3000: 3},
	{Code: "IF843", Lat: 43.8, Lon: -112.7, FoF2: 6, M3000: 3},
	{Code: "MHJ45", Lat: 42.6, Lon: -71.5, FoF2: 6, M3000: 3},
}

// statesRecorded writes two grids and the stations reading every 15 minutes
// through the first two hours - synthetic days, never a service's own data.
func statesRecorded(t *testing.T) *ionomapstest.Recorded {
	t.Helper()
	dir := t.TempDir()
	var readings []time.Time
	for m := 0; m <= 120; m += 15 {
		readings = append(readings, statesDay.Add(time.Duration(m)*time.Minute))
	}
	solar, err := ionomapstest.Write(dir, ionomapstest.Synthetic{
		Grids:    []time.Time{statesDay.Add(3*time.Hour + 5*time.Minute), statesDay.Add(5 * time.Minute)},
		GridFoF2: 6, Stations: statesStations, Readings: readings, F107: 100})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ionomapstest.Open(dir, solar)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// statesUpdate is one update as the window asks it: NOAA's files, then at
// once GIRO's stations.
func statesUpdate(lib *ionomaps.Library) (ionomaps.Snapshot, error) {
	s, err := lib.Update(context.Background(), ionomaps.UpdateOptions{})
	if err == nil && s.Reason == ionomaps.StationsNext {
		s, err = lib.Update(context.Background(), ionomaps.UpdateOptions{})
	}
	return s, err
}

// TestTheLibrarysRealStatesAreSaid is the REVIEW red team's RL-12: the real
// library, driven by a replay into each state a listener must be told, and
// the words for each - never a snapshot built by hand. GIRO's refusal of the
// host is said all hour (D-247), a station's own failure counted, the kept
// F10.7 mean said with its age when the solar file is not read (D-249), and a
// cold start with no solar file says what is lacking and the key that asks
// again.
func TestTheLibrarysRealStatesAreSaid(t *testing.T) {
	rec := statesRecorded(t)
	at := statesDay.Add(35 * time.Minute)
	served := func() map[string]ionomapstest.Answer {
		return map[string]ionomapstest.Answer{
			"D-RAP":  {Status: 200, Body: ionomapstest.DRAPTable(at.Add(-3*time.Minute), 3)},
			"scales": {Status: 200, Body: ionomapstest.ScalesFile(at.Add(-2 * time.Minute))},
		}
	}
	every := func(a ionomapstest.Answer) map[string]ionomapstest.Answer {
		out := served()
		for _, st := range statesStations {
			out[st.Code] = a
		}
		return out
	}
	oneFails := served()
	oneFails[statesStations[0].Code] = ionomapstest.Answer{Status: 503}
	glotecRefused := served()
	glotecRefused["GloTEC"] = ionomapstest.Answer{Status: 429}
	open := func(answers map[string]ionomapstest.Answer) (*ionomapstest.Replay, *ionomaps.Library) {
		r := rec.Replay(ionomapstest.Options{Answers: answers})
		lib, err := ionomaps.New(r, ionomaps.Options{Clock: r.Now})
		if err != nil {
			t.Fatal(err)
		}
		r.SetNow(at)
		return r, lib
	}
	for _, c := range []struct {
		name        string
		answers     map[string]ionomapstest.Answer
		want, never []string
	}{
		{"every file served", served(), []string{"4 stations, oldest"}, []string{"GIRO", "not read", "did not answer"}},
		{"GloTEC refused", glotecRefused, []string{"NOAA's GloTEC", "not read this update", "foF2 from the climatology"}, []string{"GIRO"}}, // a 429 ends NOAA's round: D-RAP and the scales unread too
		{"GIRO refusing", every(ionomapstest.Answer{Status: 429}), []string{"GIRO unavailable this hour"}, []string{"refused this host"}},
		{"GIRO denying the host", every(ionomapstest.Answer{Status: 403}), []string{"GIRO refused this host: not asked again for a day"}, []string{"unavailable"}},
		{"one station failing", oneFails, []string{"3 stations, oldest", "1 station did not answer"}, []string{"GIRO"}},
	} {
		r, lib := open(c.answers)
		s, err := statesUpdate(lib)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		w := snapshotWords(s, 0)
		for _, want := range c.want {
			if !strings.Contains(w, want) {
				t.Errorf("%s: %q does not say %q", c.name, w, want)
			}
		}
		for _, never := range c.never {
			if strings.Contains(w, never) {
				t.Errorf("%s: %q says %q", c.name, w, never)
			}
		}
		if c.name != "GIRO denying the host" {
			continue
		}
		r.SetNow(at.Add(20 * time.Minute)) // within the hour: the denial is said still (D-247)
		later, err := statesUpdate(lib)
		if w := snapshotWords(later, 0); err != nil || !strings.Contains(w, "GIRO refused this host") {
			t.Errorf("twenty minutes after the denial: %q (%v); want it said all hour", w, err)
		}
		r.SetNow(at.Add(25 * time.Minute)) // asked too soon: another reason, the denial said still
		soon, err := statesUpdate(lib)
		if w := snapshotWords(soon, 0); err != nil || soon.Reason == ionomaps.GIROPaused || !strings.Contains(w, "GIRO refused this host") {
			t.Errorf("five minutes on, said %v: %q (%v); want another reason and the denial said", soon.Reason, w, err)
		}
		var noted []string // the diagnostics name the denial once, as it begins
		d := goldenDash(t, false)
		d.cfg.MapProblem = func(p string) { noted = append(noted, p) }
		for _, snap := range []ionomaps.Snapshot{s, later} {
			m, _ := d.applyPropUpdated(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: snap}})
			d = m.(Dashboard)
		}
		if n := strings.Count(strings.Join(noted, "\n"), "GIRO refused this host (a 401 or 403)"); n != 1 {
			t.Errorf("the diagnostics name the denial %d times over two snapshots: %q; want once", n, noted)
		}
	}

	// The solar file unread two days on: the kept mean is used, its age said.
	_, first := open(served())
	if _, err := statesUpdate(first); err != nil {
		t.Fatal(err)
	}
	r, kept := open(map[string]ionomapstest.Answer{"solar": {Status: 503}})
	if err := kept.Restore(first.State()); err != nil {
		t.Fatal(err)
	}
	r.SetNow(at.Add(48 * time.Hour))
	s, err := statesUpdate(kept)
	if w := snapshotWords(s, 0); err != nil || !strings.Contains(w, "NOAA's solar") || !strings.Contains(w, "the solar flux held from 2 days ago") {
		t.Errorf("the solar file unread two days on: %q (%v); want the kept mean's age said", w, err)
	}

	// A cold start with no solar file: no field, what is lacking said, and the
	// key that asks again (D-124).
	_, cold := open(map[string]ionomapstest.Answer{"solar": {Status: 503}})
	_, err = statesUpdate(cold)
	if !errors.Is(err, ionomaps.ErrNoField) {
		t.Fatalf("a cold start without the solar file: %v; want no field", err)
	}
	d := goldenDash(t, false)
	d.mapPane.propErr = err
	if w := d.propStatusWords(); !strings.Contains(w, "solar data could not be read") || !strings.Contains(w, "refreshes now") {
		t.Errorf("a cold start without the solar file says %q; want what is lacking and the key that asks again", w)
	}
}

// TestEveryRefusedReadingIsNoted is FR-5.2 with go-ionomaps' counts apart
// (R2C-10, R2L-5): the burst's rows and replies refused, readings refused
// whole as implausible, and readings whose M(3000)F2 alone was refused are
// each noted in the diagnostics with their own words - once, when a count
// changes, never again for each snapshot of the hour that carries it
// (R3L-7) - and nothing is noted for a snapshot that refused none.
func TestEveryRefusedReadingIsNoted(t *testing.T) {
	var noted []string
	d := goldenDash(t, false)
	d.cfg.MapProblem = func(p string) { noted = append(noted, p) }
	apply := func(snap ionomaps.Snapshot) {
		t.Helper()
		m, _ := d.applyPropUpdated(propUpdatedMsg{gen: d.mapPane.propGen, res: PropagationResult{Snapshot: snap}})
		d = m.(Dashboard)
	}
	refusals := func() int {
		n := 0
		for _, p := range noted {
			if strings.Contains(p, "refused") {
				n++
			}
		}
		return n
	}
	snap := propSnapAt(drawnAt, 18)
	apply(snap)
	if n := refusals(); n != 0 {
		t.Fatalf("a snapshot that refused nothing noted %d refusals: %q", n, noted)
	}
	snap.Computed = drawnAt.Add(10 * time.Minute)
	snap.Inputs.Rejected, snap.Inputs.Implausible, snap.Inputs.ImplausibleM = 2, 3, 4
	apply(snap)
	text := strings.Join(noted, "\n")
	for _, want := range []string{"2 rows or replies that were refused", "3 station readings were refused as implausible", "4 station readings' M(3000)F2 was refused as implausible; their foF2 was used"} {
		if !strings.Contains(text, want) {
			t.Errorf("the diagnostics lack %q:\n%s", want, text)
		}
	}
	for i := range 5 { // the hour's other snapshots carry the same counts
		snap.Computed = drawnAt.Add(time.Duration(20+10*i) * time.Minute)
		apply(snap)
	}
	if n := refusals(); n != 3 {
		t.Errorf("one burst's counts carried through the hour were noted %d times; want 3, once each", n)
	}
	snap.Inputs.Rejected = 5
	apply(snap)
	if n := refusals(); n != 4 {
		t.Errorf("a changed count: %d refusals noted; want the change noted, 4", n)
	}
}
