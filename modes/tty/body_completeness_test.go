package tty

// body_completeness_test.go — FR-3, the tables' half of the memo-completeness
// property (B4). Its twin is memo_completeness_test.go, which does this for the
// open window.
//
// THE PROPERTY, stated as an implication:
//
//	tables(a) != tables(b)  =>  bodyKeyFor(a) != bodyKeyFor(b)
//
// "If the two tables would look different, the key must differ." The converse
// is not required — a key that changes without the frame changing is a wasted
// render, not a lie. A memo may MISS spuriously; it must never wrongly HIT.
// That asymmetry is the ruling on OQ-9 (HUM LEAD, 2026-09-07) one layer up from
// the data caches it was asked about.
//
// WHY A GUARD AND NOT THE TABLE. bodyKey's own comment already carries the
// evidence: the enumerated table in memo_test.go said it had "one row per
// field" and had 15 rows for 22 fields. A hand-kept register that tells the
// next author it is complete is worse than no register.
//
// THE LAYOUT IS RE-DERIVED, NEVER HAND-BUILT. frameLayout has no independent
// setter — d.layout() computes it — so perturbing one directly manufactures
// states the app cannot reach, which is the error this file's twin exists to
// warn about. Perturb the Dashboard, re-derive, compare.

import (
	"reflect"
	"testing"

	"github.com/branden-thompson/watchpost/platform/closedset"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestTheBodyKeyCoversEverythingTheTablesShow(t *testing.T) {
	base := benchDash(t, 133, 44).(Dashboard)
	baseFL := base.layout()
	before := bodyFrame(base, baseFL)

	// THE FIXTURE MUST DRAW BOTH TABLES, or a difference has nowhere to appear
	// and this measures nothing.
	if p, r := base.priorityTable(baseFL), base.recentSection(baseFL); p == "" || r == "" {
		t.Fatalf("the fixture draws priority=%d recent=%d cells; it cannot show a difference",
			len(p), len(r))
	}

	reached := 0
	for _, p := range perturbations(t, base) {
		fl := p.model.layout() // RE-DERIVED, as View does
		if after := bodyFrame(p.model, fl); after == before {
			continue // this field does not reach the tables; nothing is owed
		}
		reached++
		if base.bodyKeyFor(baseFL) == p.model.bodyKeyFor(fl) {
			t.Errorf("changing %s changes the tables and NOT the key: the memo will replay a "+
				"stale frame while the model moves underneath it", p.field)
		}
	}
	// AND THE WALK REACHED THE TABLES. Every assertion above is inside a branch
	// that only runs when a perturbation changes the frame; a fixture that
	// changed nothing would pass this test having checked nothing at all, which
	// is the failure mode of every guard in this repo that has ever lied.
	t.Logf("%d of %d perturbations reach the tables", reached, len(perturbations(t, base)))
	if reached < bodyPerturbationsFloor {
		t.Errorf("only %d perturbations reached the tables, floor %d: the fixture stopped "+
			"exercising the frame and this guard is asserting nothing", reached, bodyPerturbationsFloor)
	}
}

// bodyPerturbationsFloor is how many Dashboard fields must still reach the two
// tables for the walk above to be worth anything. A RATCHET, not a target: it
// is the measurement, and it drops only when someone can say why a field stopped
// reaching the frame.
const bodyPerturbationsFloor = 1

// AND THE THEME, WHICH REFLECTION CANNOT REACH AT ALL.
//
// render.ThemeGeneration() is a package global with no Dashboard preimage, so
// the walk above cannot perturb it and would report full coverage without ever
// touching the one input that re-tints every cell in both tables. A guard that
// derives its subjects from a struct is blind to everything that is not in it.
func TestTheBodyKeyCoversTheTheme(t *testing.T) {
	base := benchDash(t, 133, 44).(Dashboard)
	fl := base.layout()
	before, key := bodyFrame(base, fl), base.bodyKeyFor(fl)

	was := render.ThemeName()
	if !render.SetTheme("Monochrome") {
		t.Fatal("the fixture theme is missing; nothing was switched")
	}
	t.Cleanup(func() { render.SetTheme(was) })

	after, next := bodyFrame(base, base.layout()), base.bodyKeyFor(base.layout())
	if after == before {
		t.Fatal("the theme switch did not change the tables; this measures nothing")
	}
	if next == key {
		t.Error("the theme re-tints every cell in both tables and does not change the key: the " +
			"memo will replay the old theme's frame for as long as nothing else moves")
	}
}

// bodyFrame is the two tables as one string — what the memo caches, and the
// only thing the property is about.
func bodyFrame(d Dashboard, fl frameLayout) string {
	return d.priorityTable(fl) + "\x00" + d.recentSection(fl)
}

// EVERY INPUT bodyKey CARRIES IS ACTUALLY EXERCISED (B4).
//
// THE COUNT ABOVE IS NOT COVERAGE. Six of sixty-six perturbations reach the
// tables, and the six are width, height, units, selected, recentOff and
// radioViz — so most of what bodyKey carries is never varied by the walk at
// all. Dropping any of those inputs from the key would leave the guard green:
// a hole shaped exactly like coverage, which is the defect this release exists
// to close, sitting inside the instrument written to close it.
//
// THE REFLECTION WALK CANNOT DO BETTER ON ITS OWN, and the reason is worth
// writing down: it BUMPS a value, and several of these inputs only matter when
// they MATCH something. Bumping radioKey to a nonsense key moves the ▶ mark
// from no row to no row; bumping it to a row's own key is what draws it. The
// walk covers fields whose effect is unconditional and is blind to every field
// whose effect is a join.
//
// So the walk is supplemented by named perturbations — each one a state the app
// can actually reach — and the two together are crossed against bodyKey's own
// fields. A field that neither reaches is FAILED here, or declared with a
// reason nobody can silently inherit.
func TestEveryBodyKeyInputIsExercised(t *testing.T) {
	base := benchDash(t, 133, 44).(Dashboard)
	baseFL := base.layout()
	before, baseKey := bodyFrame(base, baseFL), base.bodyKeyFor(baseFL)

	exercised := map[string]bool{}
	all := append(perturbations(t, base), joinPerturbations(t, base)...)
	for _, p := range all {
		fl := p.model.layout()
		if bodyFrame(p.model, fl) == before {
			continue // it changes nothing the tables draw; it exercises nothing
		}

		for _, f := range differingKeyFields(baseKey, p.model.bodyKeyFor(fl)) {
			exercised[f] = true
		}
	}

	closedset.EachMember(t, "bodyKey inputs", bodyKeyFields(), bodyKeyUnexercised, func(f string) bool {
		return exercised[f]
	})
}

// bodyKeyUnexercised is an input no perturbation can vary, and why. A reason is
// not a silencer: an input that starts being exercised while its row stands is
// itself a failure (platform/closedset).
var bodyKeyUnexercised = map[string]string{
	// GEOMETRY THAT NEVER FLIPS BETWEEN THE TWO SIZES MEASURED. compact, alertH
	// and days are layout decisions, and neither 133x44 nor 80x24 turns any of
	// them over — the walk's ±1 bump certainly does not. Closing these means a
	// fixture per decision (a row with alerts for alertH, a height that forces
	// the compact radio panel, a width that drops a forecast day), which is
	// worth doing and is not free.
	"compact": "no fixture flips it: it is the same at 133x44 and at 80x24",
	"alertH":  "no fixture flips it: the bench rows carry no alert block that changes height",
	"days":    "no fixture flips it: the forecast column count is the same at both sizes",
	// THE SHIMMER IS A PHASE, so it needs two loading frames to compare, not a
	// loading frame against a settled one — the pair differs by the snapshot as
	// well, and the snapshot is what the frame comparison then attributes the
	// change to.
	"shimmer": "needs a loading-vs-loading pair; measured against a settled base it is attributed to snap",
	// THE THRESHOLD NEEDS A HOTSPOT NEAR IT. The bench rows carry no fire, so
	// moving the bold rule from 50 MW to 1 MW re-draws nothing.
	"fireBoldMW": "the bench fixture has no fire hotspots for the threshold to reclassify",
	// THE THEME HAS NO Dashboard PREIMAGE AT ALL and is covered by its own test
	// (TestTheBodyKeyCoversTheTheme), which switches the global and was watched
	// failing with the field removed from the key.
	"theme": "covered by TestTheBodyKeyCoversTheTheme; it is a package global, not a field of anything",
}

// joinPerturbations are the states the reflection walk cannot reach because
// their effect is a JOIN rather than a value: each one is set to match
// something the fixture actually holds.
func joinPerturbations(t *testing.T, base Dashboard) []perturbed {
	t.Helper()
	if base.snap == nil || len(base.snap.Locations) == 0 {
		t.Fatal("the fixture has no locations; none of these can match anything")
	}
	l0 := base.snap.Locations[0]
	first := snapshot.Key(snapshot.LocationRef{Label: l0.Label, Lat: l0.Lat, Lon: l0.Lon})

	set := func(name string, f func(*Dashboard)) perturbed {
		next := base
		f(&next)
		return perturbed{field: name, model: next}
	}
	return []perturbed{
		// The ▶ mark is drawn on the row whose key MATCHES. A bumped key
		// matches nothing, which is why the walk sees no change.
		set("radioKey=row", func(d *Dashboard) { d.radioKey, d.radioPlaying = first, true }),
		set("radioPlaying=false", func(d *Dashboard) { d.radioKey, d.radioPlaying = first, false }),
		// AND ∞ NEEDS ▶. Setting the repeat alone changes the key and nothing on
		// screen: the mark is drawn on a row that is playing, so a perturbation
		// without radioPlaying exercises the field it names and proves nothing.
		set("radioRepeat=one", func(d *Dashboard) {
			d.radioKey, d.radioPlaying, d.radioRepeat = first, true, RepeatOne
		}),
		// The shimmer only exists while something is loading, and loading is a
		// property of the snapshot's rows rather than of the Dashboard.
		set("shimmer", func(d *Dashboard) { d.snap, d.frame = loadingSnap(base.snap), 1 }),
		// A pending lookup draws a placeholder row before any snapshot carries
		// it.
		set("lookup", func(d *Dashboard) {
			ref := snapshot.LocationRef{Label: "Pending, CA", Lat: 1.5, Lon: -2.5}
			d.lookupRef = &ref
		}),
		// The two snapshots are pointers: the walk skips them entirely.
		set("snap", func(d *Dashboard) { d.snap = trimmedSnap(base.snap) }),
		set("recent", func(d *Dashboard) { d.recent = trimmedSnap(base.recent) }),
		// GEOMETRY, at the app's floor. The walk bumps an int by one, and one
		// cell flips none of the layout's decisions: compact, the control rows,
		// the forecast days, the alert height and the thin bands all turn over
		// somewhere between 133x44 and 80x24, and nowhere in between one cell.
		set("floor", func(d *Dashboard) { d.width, d.height = 80, 24 }),
		set("short", func(d *Dashboard) { d.height = 26 }),
		// THE CONFIG IS NOT A DASHBOARD FIELD. --ascii and the fire-bold
		// threshold arrive through cfg, which the walk does not descend into,
		// and both re-draw every row that has a glyph or a hotspot.
		set("ascii", func(d *Dashboard) { d.cfg.ASCII = !d.cfg.ASCII }),
		set("fireBoldMW", func(d *Dashboard) { d.cfg.FireBoldMW = 1 }),
	}
}

// loadingSnap is the fixture's snapshot with its first row still loading.
func loadingSnap(sn *snapshot.Snapshot) *snapshot.Snapshot {
	out := *sn
	out.Locations = append([]snapshot.Location(nil), sn.Locations...)
	// rowLoading: no provider on the harmonized source, or no daily rows.
	out.Locations[0].Harmonized.Source.Provider = ""
	return &out
}

// trimmedSnap is the fixture's snapshot one row shorter — a different table
// with the same shape.
func trimmedSnap(sn *snapshot.Snapshot) *snapshot.Snapshot {
	if sn == nil || len(sn.Locations) < 2 {
		return sn
	}
	out := *sn
	out.Locations = append([]snapshot.Location(nil), sn.Locations[1:]...)
	return &out
}

// bodyKeyFields is bodyKey's own field names, derived rather than listed: a
// field added to the key is a member here without anyone remembering.
func bodyKeyFields() []string {
	t := reflect.TypeOf(bodyKey{})
	out := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		out = append(out, t.Field(i).Name)
	}
	return out
}

// differingKeyFields names the fields two keys disagree on.
func differingKeyFields(a, b bodyKey) []string {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	var out []string
	for i := range va.NumField() {
		if !va.Field(i).Equal(vb.Field(i)) {
			out = append(out, va.Type().Field(i).Name)
		}
	}
	return out
}
