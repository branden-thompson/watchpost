package tty

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// memo_completeness_test.go — F-30, mechanised.
//
// THE BUG THIS RETIRES. The modal frame is memoised on a key built by
// modalKeyFor, a hand-maintained switch where each window adds what IT shows
// that moves. A window whose moving state is absent renders ONCE and the memo
// replays that frame for as long as it is open — while the model underneath
// works perfectly. THREE windows shipped that way in one release: the
// relay-fault window's cursor and clock, the ctrl+d window's cursor, and the
// severe window's Pause/Play. The first took three UAT rounds to find, because
// the arrows moved the cursor, the countdown counted, the fall-through fired on
// time, and the display never changed.
//
// EVERY EARLIER GUARD WAS A HAND-MAINTAINED LIST TOO — a table of "windows with
// a cursor", which cannot catch a window whose moving state is not a cursor, and
// still has to be remembered. The one below cannot be forgotten: it DERIVES the
// fields from the struct.
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
			// It skipped, loudly, and 11 of 11 windows have a fixture — so the
			// branch fired zero times and the guard's green number described
			// today's windows rather than the guard. The next window added with
			// a cursor and no fixture would have been silently uncovered, which
			// is the defect F-30 was filed for. Adding a window means adding a
			// fixture, and this is where that is enforced.
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
func perturbations(t *testing.T, base Dashboard) []perturbed {
	t.Helper()
	var out []perturbed
	walk(t, reflect.TypeOf(base), "", func(path string, set func(*Dashboard)) {
		next := base
		if !apply(&next, set) {
			return // the perturbation panicked the copy; not a finding about the key
		}
		out = append(out, perturbed{field: path, model: next})
	})
	return out
}

// apply runs one perturbation, containing a panic from an index a fixture does
// not support. A field that cannot be perturbed safely is skipped rather than
// reported: this test is about the KEY, not about render robustness.
func apply(d *Dashboard, set func(*Dashboard)) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	set(d)
	return true
}

// walk visits every field this test knows how to change, one level into nested
// structs (relayFault, debug and setup all hold their state that way).
func walk(t *testing.T, typ reflect.Type, prefix string, emit func(string, func(*Dashboard))) {
	t.Helper()
	for i := range typ.NumField() {
		f := typ.Field(i)
		path := prefix + f.Name
		switch f.Type.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.String:
			idx := i
			emit(path, func(d *Dashboard) { bump(fieldAt(d, prefix, idx)) })
		case reflect.Struct:
			// ONE LEVEL DOWN, INTO EVERY STRUCT THIS FILE HAS NOT EXCUSED
			// (FR-3.3). It named three — relayFault, debug and setup — so a
			// FOURTH window's state was never perturbed and the guard reported
			// coverage it did not have. A hand-written list of the windows with
			// state is the same shape as the hand-written memo key it checks,
			// and would miss a new window in exactly the same way.
			if prefix == "" && nestedExcuse[path] == "" {
				walkNested(t, f, i, emit)
			}
		}
	}
}

// walkNested is walk for the fields of one nested struct.
func walkNested(t *testing.T, f reflect.StructField, outer int, emit func(string, func(*Dashboard))) {
	t.Helper()
	for j := range f.Type.NumField() {
		inner := f.Type.Field(j)
		switch inner.Type.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.String:
			jdx := j
			name := f.Name
			if nestedExcuse[name+"."+inner.Name] != "" {
				continue
			}
			emit(name+"."+inner.Name, func(d *Dashboard) {
				v := reflect.ValueOf(d).Elem().Field(outer).Field(jdx)
				bump(reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem())
				// SETUP IS KEYED BY GENERATION, not by its fields: the window
				// carries two maps and fingerprinting it every frame was the
				// largest thing it cost, so every WRITER bumps gen instead
				// (setupState.touch, via settled/castTouched). Writing a field
				// without bumping is not something the app does, so this models
				// the real writer rather than reporting the design as a defect.
				if name == "setup" {
					d.setup = d.setup.touch()
				}
			})
		}
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
}

// fieldAt is one addressable field of d, by index, at the top level.
func fieldAt(d *Dashboard, prefix string, i int) reflect.Value {
	_ = prefix
	v := reflect.ValueOf(d).Elem().Field(i)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

// bump changes a value to a DIFFERENT one of its own type. The particular value
// does not matter; only that it differs.
func bump(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.String:
		v.SetString(v.String() + "·changed")
	case reflect.Int, reflect.Int64:
		if v.Type() == reflect.TypeOf(time.Duration(0)) {
			v.SetInt(v.Int() + int64(time.Second))
			return
		}
		v.SetInt(v.Int() + 1)
	}
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
	}
	return "modal-" + strconv.Itoa(int(m))
}

// fixtureFor is a dashboard with m open, given whatever that window needs to
// draw. A window with nothing here still gets opened — it simply draws less.
func fixtureFor(t *testing.T, m modal) Dashboard {
	t.Helper()
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	switch m {
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
		// of the frame and the guard cannot see it — which it could not, at the
		// first attempt, on the very window whose omission the red team found.
		// A fixture that does not exercise the state is a hole shaped exactly
		// like coverage.
		d = d.applySevere(SevereMsg{Gen: 1, Totals: [severeNumTabs]int{SevereWarnings: 1},
			Rows: []SevereRow{{Key: "k", Tab: SevereWarnings, Product: "Tornado Warning",
				Location: "Olathe, KS", Declared: "08/28 08:45 CDT",
				Record: SevereRecord{Title: "TORNADO WARNING"}}}})
		d.severeReading = "k"
	}
	return d.open(m)
}

// TestEveryWindowClearsItsMargins — the three-cell inset, on BOTH sides, for
// EVERY window rather than the one that was reported.
//
// HOW THIS CAME ABOUT (2026-09-05). The HUM LEAD reported content running into
// the right border of the relay-fault window; it was fixed there. A red team
// then reported the same mismatch in the [S] window — and MEASURING IT SHOWED
// [S] WAS ALREADY CORRECT: the claim had read `" "+o.Controls(...)` as one cell
// without counting the panel's own two, which is the same arithmetic that trips
// everyone in these files. Measuring every window instead settled that AND
// found four real ones nobody had reported.
//
// So the survey is the test. A margin claim about any window is now answered by
// running this rather than by reading a source line.
func TestEveryWindowClearsItsMargins(t *testing.T) {
	// KNOWN-UNDER, LISTED RATHER THAN SKIPPED. Each is a window untouched by the
	// work that found them, with its own mock or golden; fixing one is a HUM
	// LEAD appearance call, and deleting its row here is how that lands. A
	// window absent from this map must clear both margins.
	known := map[modal]string{
		modalHelp:    "trails at 2 — hand-wrapped prose, its own two-column layout",
		modalDetails: "leads at 2 — the forecast table sets its own left edge",
		modalRemove:  "trails at 2 — a confirmation whose text is centred, not inset",
	}
	for m := modalHelp; m < numModals; m++ {
		t.Run(modalName(m), func(t *testing.T) {
			d := fixtureFor(t, m)
			out := stripANSITest(d.modalView(d.opts()))
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
