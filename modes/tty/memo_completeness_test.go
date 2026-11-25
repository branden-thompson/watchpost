package tty

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty/console"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// memo_completeness_test.go — F-30, mechanised.
//
// THE FAILURE IT GUARDS. The modal frame is memoised on a key built by
// modalKeyFor, a hand-maintained switch where each window adds what IT shows
// that moves. A window whose moving state is absent renders ONCE and the memo
// replays that frame for as long as it is open — while the model underneath
// works perfectly: the arrows move the cursor, the countdown counts, the
// fall-through fires on time, and the display never changes. A cursor, a clock
// or a Pause/Play left out of the key is enough.
//
// A HAND-MAINTAINED LIST CANNOT GUARD IT — a table of "windows with a cursor"
// cannot catch a window whose moving state is not a cursor, and still has to
// be remembered. The one below cannot be forgotten: it DERIVES the fields from
// the struct.
//
// THE PROPERTY, stated as an implication:
//
//	renderModal(a) != renderModal(b)  =>  modalKeyFor(a) != modalKeyFor(b)
//
// "If the frame would look different, the key must differ." The converse is not
// required — a key that changes without the frame changing is a wasted render,
// not a lie. This is exactly the direction that matters: a memo may miss, it
// must never wrongly HIT.
func TestTheMemoKeyCoversEverythingTheFrameShows(t *testing.T) {
	// EVERY MODAL IN THE ENUM, derived rather than listed. A hand-written set of
	// windows is the same shape as the hand-written key this test exists to
	// check: a window added later would be absent from both.
	for m := modalHelp; m < numModals; m++ {
		t.Run(modalName(m), func(t *testing.T) {
			base := fixtureFor(t, m)
			o := base.opts()
			// A WINDOW WITH NO FIXTURE IS A FAILURE, not a skip (FR-3.3).
			//
			// A skip, however loud, fires zero times while every window has a
			// fixture — so a green number would describe today's windows rather
			// than the guard, and the next window added with a cursor and no
			// fixture would be silently uncovered, which is the defect F-30 was
			// filed for. Adding a window means adding a fixture, and this is
			// where that is enforced.
			if base.renderModal(o) == "" {
				t.Fatalf("%s draws nothing at fixtureFor: give it a fixture, or this window is "+
					"NOT covered by the memo guard and the next thing that freezes it will be found "+
					"in UAT (F-30)", modalName(m))
			}
			for _, p := range perturbations(t, base) {
				// EACH MODEL RENDERS THROUGH ITS OWN OPTS, as View does. Holding
				// one Opts across both is not what production does — Opts is
				// DERIVED from the model (units, clock) — and it manufactures a
				// divergence the app cannot have.
				po := p.model.opts()
				before, after := base.renderModal(o), p.model.renderModal(po)
				if before == after {
					continue // this field does not reach the frame; nothing is owed
				}
				if base.modalKeyFor(o) == p.model.modalKeyFor(po) {
					t.Errorf("changing %s changes the frame and NOT the key: the memo will replay a "+
						"stale frame while the model moves underneath it (F-30)", p.field)
				}
			}
		})
	}
}

// perturbed is one single-field change and the name of the field changed.
type perturbed struct {
	field string
	model Dashboard
}

// perturbations is one model per mutable field of the Dashboard, each differing
// from base in exactly that field.
//
// DERIVED BY REFLECTION, which is the whole point: a field added later is
// perturbed without anyone remembering to add it here. Unexported fields are
// reachable because this is an in-package test and the struct is addressable.
//
// THE MAP'S MODEL IS WALKED AS A MODEL OF ITS OWN (0.19.6 A-31, D-81): a named
// field, it is one struct the walk would descend into once, leaving its pane
// and its choices unperturbed while the guard reported them covered. So the
// Dashboard is walked with it excused, and then the model, each of its
// perturbations set back into the Dashboard, the Dashboard's writer modelled
// on it as on the Dashboard's own.
func perturbations(t *testing.T, base Dashboard) []perturbed {
	t.Helper()
	var out []perturbed
	ttytest.PerturbEach(t, base, nestedExcuse, dashboardWriter, func(path string, next Dashboard) {
		out = append(out, perturbed{field: path, model: next})
	})
	ttytest.PerturbEach(t, base.mapModel, nil, nil, func(path string, next mapModel) {
		d := base
		d.mapModel = next
		dashboardWriter(path, &d)
		out = append(out, perturbed{field: "mapModel." + path, model: d})
	})
	return out
}

// dashboardWriter models the real writer for a field the app never writes bare.
//
// SETUP IS KEYED BY GENERATION, not by its fields: the window carries two maps
// and fingerprinting it every frame would be the largest thing it costs, so
// every WRITER bumps gen instead (setupState.touch, via settled/castTouched).
// Writing a field without bumping is not something the app does, so this
// models the real writer rather than reporting the design as a defect.
//
// mapTempNDFD IS SETTINGS' TOO (D-190): its one writer, toggleTempSource, is a
// Settings row's and settles - bumps gen - and its metered note changes the
// window's size on every tab, so it reaches the frame there.
func dashboardWriter(name string, d *Dashboard) {
	if name == "setup" || name == "mapTempNDFD" || name == "mapRainFull" {
		d.setup = d.setup.touch()
	}
}

// nestedExcuse names a struct field this walk does NOT descend into, and why.
// A reason, not a silencer: it is read by TestTheNestedWalkExcusesNothingQuietly,
// which fails on an excuse for a field that no longer exists.
var nestedExcuse = map[string]string{
	"memo": "the memo itself: perturbing the cache is not a state the model can be in",
	// THE VERSION IS WRITTEN ONCE, AT CONSTRUCTION. The About window and the
	// header both draw it, so perturbing it changes the frame — but nothing in
	// the app writes cfg.Version after NewDashboard, so no reachable state
	// change can make a memoised frame stale on it. Carrying it in the key
	// would cost a comparison every frame for a value that cannot move.
	//
	// The REST of cfg is walked, and must be: Cast, Tones, Radio, Spectrum,
	// Units and Clock are all written while the app runs (dashboard.go,
	// setup_ui.go), which is why the excuse is one FIELD rather than the struct
	// it sits in.
	"cfg.Version": "written once at construction; the app has no path that changes it while running",
	"mapModel":    "the map's model, walked as a model of its own (perturbations, 0.19.6 D-81)",
}

// modalName is the window's name for the subtest, derived from the enum so a
// new member is named rather than numbered.
func modalName(m modal) string {
	switch m {
	case modalHelp:
		return "help"
	case modalDetails:
		return "details"
	case modalAdd:
		return "add"
	case modalRemove:
		return "remove"
	case modalAlerts:
		return "alerts"
	case modalStatus:
		return "status"
	case modalAbout:
		return "about"
	case modalSetup:
		return "setup"
	case modalSevere:
		return "severe"
	case modalRelayFault:
		return "relay-fault"
	case modalDebug:
		return "debug"
	case modalCard:
		return "card"
	case modalRequest:
		return "request"
	case modalMap:
		return "map"
	case modalPropAck:
		return "propagation-ack"
	}
	return "modal-" + strconv.Itoa(int(m))
}

// fixtureFor is a dashboard with m open, given whatever that window needs to
// draw. A window with nothing here still gets opened — it simply draws less.
func fixtureFor(t *testing.T, m modal) Dashboard {
	t.Helper()
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	// THE CLOCK IS PINNED. The Status window's FETCHED column is an age read
	// off d.now at minute resolution; on the real clock a run that straddles a
	// minute boundary between building the lines and rendering them holds a
	// text no frame ever draws, and the reachability guard reports it as a
	// line the keyboard cannot reach — a rare flake, seen in CI rather than
	// locally. An hour after the fixture's observation.
	d.now = func() time.Time { return time.Date(2026, 8, 24, 2, 0, 0, 0, time.UTC) }
	switch m {
	case modalMap:
		// A PLACED LOCATION AND A SETTLED MAP (0.18.0 W1.7). The shared fixture's
		// location sits at 0,0, where the map is open sea and draws nothing that
		// could show a key missing; Oceanside draws a coast, two cities, a scale
		// and the credit.
		ownHooks(&d).NewMap = embeddedMap
		next, _ := d.Update(SnapshotMsg{Snap: placedSnap()})
		d = next.(Dashboard).toggleMap()
		t.Cleanup(d.mapModel.closeMap)
		return settleMap(t, d)
	case modalRelayFault:
		return d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
			{Label: "KEC62 San Diego CA 162.400 MHz - 12 mi", Key: "a"},
			{Label: "WNG712 Coachella CA 162.525 MHz - 81 mi", Key: "b"},
		}})
	case modalDebug:
		d.cfg.InjectAlert = func(string) {}
		d.cfg.DebugScenarios = []DebugScenario{{Label: "one alert", Key: "one"}, {Label: "a burst", Key: "burst"}}
	case modalSevere:
		// A ROW, AND A READ IN PROGRESS. Both are needed for the window to draw
		// everything it draws: without a read, severeReadPause reaches no part
		// of the frame and the guard cannot see it. A fixture that does not
		// exercise the state is a hole shaped exactly like coverage.
		d = d.applySevere(SevereMsg{Gen: 1, Totals: [severeNumTabs]int{SevereWarnings: 1},
			Rows: []SevereRow{{Key: "k", Tab: SevereWarnings, Product: "Tornado Warning",
				Location: "Olathe, KS", Declared: "08/28 08:45 CDT",
				Record: SevereRecord{Title: "TORNADO WARNING"}}}})
		d.severeReading = "k"
	case modalAdd:
		// THE SEARCH WINDOW AS THE CONSOLE SEES IT (D-129), because that is the
		// only mode in which it draws the location verdict at all. Opened on
		// Observer's surface with no hook wired, the window is unscoped, the
		// note never reaches the frame, and the verdict fields are invisible to
		// this guard: a hole shaped exactly like coverage, which is what the
		// modalRequest arm below says in as many words.
		//
		// A SETTLED MATCH, NOT A MISS, and driven through the REAL path. On a
		// miss the ref is nil and `within` cannot reach the frame at all — the
		// refusal reads the same either way — so the fixture would cover
		// neither field while looking like it covered the window.
		d.surface, d.addMode, d.addQuery = SurfaceBroadcaster, "lookup", "Vista"
		d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
			return snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"}, true, true, true
		}
		d.addLocate = settledLocate(locateLookup, "Vista",
			snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"}, true, true)
	case modalRequest:
		// EVERY FIELD CARRYING SOMETHING, because a window whose fields are all
		// empty draws the same frame however the model moves — and the guard
		// below would then report a key covering a window that says nothing.
		//
		// AND IT SETS THE STATE THE WINDOW ACTUALLY READS, never a field nothing
		// in production writes (D-130): a fixture carrying a value the frame
		// cannot see exercises less than it appears to.
		d.request = requestOpen()
		d.request.query = "Oceanside, CA"
		d.request.locate = settledLocate(locateRequest, "Oceanside, CA",
			snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057"}, true, true)
		d.request.slot = "4"
	case modalCard:
		// THE CONSOLE'S OWN BODY, NOT A HAND-WRITTEN ONE. The window draws what
		// `Broadcaster.CardDetail` produces, and a fixture that typed its own
		// rows would keep passing on the day the console stopped producing any.
		bc := consoleOf(t, console.Opening{Now: func() time.Time { return ttytest.DetailAt().Add(2 * time.Minute) }},
			ttytest.Seed(t, ttytest.OneCard(t)))
		id, rows, ok := bc.CardDetail(1)
		if !ok {
			t.Fatalf("the console produced no card for slot 1; this fixture measures nothing")
		}
		d = d.showCard(id, rows, bc.CardWindowGroundFor(1), d.opts())
	}
	return d.open(m)
}

// TestEveryWindowClearsItsMargins — the three-cell inset, on BOTH sides, for
// EVERY window.
//
// THE SURVEY IS THE TEST. A margin is easy to misjudge by reading:
// `" "+o.Controls(...)` looks like one cell until the panel's own two are
// counted, which is the arithmetic that trips everyone in these files. A margin
// claim about any window is answered by running this rather than by reading a
// source line.
func TestEveryWindowClearsItsMargins(t *testing.T) {
	// KNOWN-UNDER, LISTED RATHER THAN SKIPPED. Each is a window with its own
	// mock or golden; fixing one is a HUM LEAD appearance call, and deleting
	// its row here is how that lands. A window absent from this map must clear
	// both margins.
	known := map[modal]string{
		modalHelp:    "trails at 2 — hand-wrapped prose, its own two-column layout",
		modalDetails: "leads at 2 — the forecast table sets its own left edge",
		modalRemove:  "trails at 2 — a confirmation whose text is centred, not inset",
	}
	for m := modalHelp; m < numModals; m++ {
		t.Run(modalName(m), func(t *testing.T) {
			d := fixtureFor(t, m)
			out := ttytest.StripANSI(d.modalView(d.opts()))
			if out == "" {
				t.Skipf("%s draws nothing without a richer fixture; NOT covered", modalName(m))
			}
			lead, trail := 99, 99
			for _, l := range strings.Split(out, "\n") {
				r := []rune(l)
				if len(r) < 3 || r[0] != '│' {
					continue // the borders themselves
				}
				in := string(r[1 : len(r)-1])
				if strings.TrimSpace(in) == "" {
					continue // a blank spacer has no content to inset
				}
				lead = min(lead, len(in)-len(strings.TrimLeft(in, " ")))
				trail = min(trail, len(in)-len(strings.TrimRight(in, " ")))
			}
			if lead == 99 {
				t.Fatalf("%s drew no content lines; this measures nothing", modalName(m))
			}
			// THE MAP WINDOW IS FLUSH, BY RULING (UAT-1 U1-27, HUM LEAD: "since
			// this is a map, not a table, we dont need that global padding for
			// this view"): its picture runs from border to border, and it is held
			// to that - an inset creeping back fails here too.
			if m == modalMap {
				if lead != 0 {
					t.Errorf("the map window is flush (U1-27) and its content now leads at %d", lead)
				}
				return
			}
			ok := lead >= modalInset && trail >= modalInset
			if why, listed := known[m]; listed {
				if ok {
					t.Errorf("%s clears both margins now (%d/%d) — delete its row from `known`, "+
						"or the next window to regress here will look expected", modalName(m), lead, trail)
				} else {
					t.Logf("KNOWN, not fixed: %s (%d/%d) — %s", modalName(m), lead, trail, why)
				}
				return
			}
			if !ok {
				t.Errorf("%s leads at %d and trails at %d, want %d on both: content runs into the border",
					modalName(m), lead, trail, modalInset)
			}
		})
	}
}

// AN EXCUSE CANNOT OUTLIVE THE FIELD IT EXCUSES (FR-3.3).
//
// nestedExcuse is the one hand-written thing left in this walk, so it is the
// one thing that can rot: a field renamed or deleted leaves a row that silences
// nothing and reads like coverage. The same failure mode as the stale exemption
// platform/closedset was written for.
func TestTheNestedWalkExcusesNothingQuietly(t *testing.T) {
	typ := reflect.TypeOf(Dashboard{})
	for path := range nestedExcuse {
		outer, inner, nested := strings.Cut(path, ".")
		f, ok := typ.FieldByName(outer)
		if !ok {
			t.Errorf("nestedExcuse names %q and Dashboard has no field %q", path, outer)
			continue
		}
		if !nested {
			continue
		}
		if _, ok := f.Type.FieldByName(inner); !ok {
			t.Errorf("nestedExcuse names %q and %s has no field %q", path, outer, inner)
		}
	}
}
