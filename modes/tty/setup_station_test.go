package tty

// setup_station_test.go — the station's two settings in the Settings window
// (D-115, F-87).
//
// HUM LEAD, 2026-09-13: "Currently I cannot change these settings with[out]
// direct code changes - we need [them] exposed so I can also UAT the
// re-derivation logic."

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func stationSetup(t *testing.T, at setupRowID) Dashboard {
	t.Helper()
	d := setupGolden(t, 133, 44, false, at)
	d.surface = SurfaceBroadcaster
	// THE BOUNDS ARE HANDED IN, AS THE APP HANDS THEM IN (D-124). They were
	// constants in this package and are now `platform/config`'s alone; a fixture
	// that omits them gets a window which refuses every radius, which is the
	// deliberate fail-closed behaviour and not something to paper over.
	d.cfg.ServiceRadiusMinMi, d.cfg.ServiceRadiusMaxMi = 2, 100
	return d
}

// TestAWindowWithNoBoundsRefusesEveryRadius. UNSET IS NOT A DEFAULT.
//
// The bounds arrive through `Config`, and a build that forgets to send them must
// not fall back to a guess: the window would offer a radius the storage clamps
// behind the operator's back, and the setting would silently not be what the
// screen said. Zero refuses everything, loudly.
func TestAWindowWithNoBoundsRefusesEveryRadius(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowServiceRadius)
	d.surface = SurfaceBroadcaster // and NO bounds handed in
	if _, _, ok := d.serviceBounds(); ok {
		t.Fatal("a window with no bounds must not report usable ones")
	}
	// ZERO IS IN THIS LIST BECAUSE IT IS THE ONLY VALUE THAT DISCRIMINATES, and
	// leaving it out is what let mutant mAS3 survive. With the bounds unset, a
	// check that forgets to ask whether it was TOLD compares `v >= 0 && v <= 0` —
	// which refuses 2, 25, 50 and 100 exactly as the correct code does, and
	// ADMITS zero. Zero is what `serviceRadiusChoice` returns for "not a number",
	// so the one value that slips through is the one meaning nothing was typed.
	//
	// A test named "refuses EVERY radius" that omits the only distinguishing
	// case is a test agreeing with the defect on every input it tries.
	for _, v := range []int{0, 1, 2, 25, 50, 100, 101} {
		if d.inServiceRange(v) {
			t.Errorf("%d mi was admitted by a window that was never told its bounds", v)
		}
	}
}

// THEY ARE THE CONSOLE'S AND ONLY THE CONSOLE'S (D-72, D-18's M4 metric).
//
// A listener has no transmitter and no service area; Observer's own default
// location is the row these two sit beside. A leak in either direction is the
// failure the scope table exists to count.
func TestTheStationsSettingsAreTheConsolesAlone(t *testing.T) {
	console := setupOffers(stationSetup(t, rowFIRMSKey))
	observer := setupOffers(setupGolden(t, 133, 44, false, rowFIRMSKey))

	for _, want := range []string{"Transmitter (epicenter):", "Service radius:"} {
		if !strings.Contains(console, want) {
			t.Errorf("the console's Settings window does not offer %q", want)
		}
		if strings.Contains(observer, want) {
			t.Errorf("%q leaked into OBSERVER, which has no station", want)
		}
	}
	// AND THEY SIT IN DATA, where the HUM LEAD put them: "These options should be
	// under the 'DATA' settings group in the modal."
	data := setupOffers(stationSetup(t, rowTransmitter))
	head := strings.Index(data, "DATA")
	next := strings.Index(data, "WATCHPOST UI")
	if head < 0 || next < 0 {
		t.Fatalf("the window has no DATA group to put them in")
	}
	for _, want := range []string{"Transmitter (epicenter):", "Service radius:"} {
		if at := strings.Index(data, want); at < head || at > next {
			t.Errorf("%q is at %d, outside DATA (%d..%d)", want, at, head, next)
		}
	}
}

// THE TRANSMITTER FUNCTIONS LIKE THE DEFAULT LOCATION ROW, which is what was
// asked for: type, pick from the suggestions, enter.
func TestTheTransmitterRowResolvesLikeTheDefaultLocation(t *testing.T) {
	d := stationSetup(t, rowTransmitter)
	d.cfg.Suggest = func(string, int) []snapshot.LocationRef {
		return []snapshot.LocationRef{{Label: "Bonsall, CA", Zip: "92003", Lat: 33.28, Lon: -117.23}}
	}
	for _, r := range "bons" {
		m, _ := d.setupRowText(tea.KeyPressMsg{Code: r, Text: string(r)})
		d = m.(Dashboard)
	}
	if len(d.setup.hints) == 0 {
		t.Fatal("typing produced no suggestions")
	}
	m, _ := d.setupRowText(tea.KeyPressMsg{Code: tea.KeyEnter})
	d = m.(Dashboard)

	if d.setup.txRef == nil || d.setup.txRef.Label != "Bonsall, CA" {
		t.Fatalf("enter chose %v, want Bonsall, CA", d.setup.txRef)
	}
	// AND IT DID NOT WRITE THE LISTENER'S DEFAULT LOCATION, which is the leak the
	// shared type-ahead makes possible: one control, two settings, and only one
	// of them is the station's.
	if d.setup.ref != nil {
		t.Errorf("choosing the station's transmitter also set the listener's default to %v", d.setup.ref)
	}
	// AND THE FOCUS MOVES TO THE RADIUS, the next of the station's two.
	if d.setup.focus != rowServiceRadius {
		t.Errorf("enter left the focus on %v, not the service radius", d.setup.focus)
	}
}

// THE RADIUS FUNCTIONS LIKE THE ALERT RADIUS MINUS THE RADIO — digits, and the
// first one typed REPLACES the stored value rather than appending to it.
func TestTheServiceRadiusTakesDigitsAndReplacesTheStoredOne(t *testing.T) {
	d := stationSetup(t, rowServiceRadius)
	d.cfg.ServiceRadiusMi = 25
	d = d.openSetup().openSetupAt(rowServiceRadius)
	d.surface = SurfaceBroadcaster
	if d.setup.serviceMi != "25" || !d.setup.serviceSeeded {
		t.Fatalf("the window opened showing %q (seeded=%v), want the stored 25",
			d.setup.serviceMi, d.setup.serviceSeeded)
	}
	for _, r := range "50" {
		m, _ := d.setupRowText(tea.KeyPressMsg{Code: r, Text: string(r)})
		d = m.(Dashboard)
	}
	// 50, NOT 2550 — the defect the alert radius was fixed for at UAT 2026-09-08.
	if d.setup.serviceMi != "50" {
		t.Errorf("typing 50 over a stored 25 gave %q", d.setup.serviceMi)
	}
	if got := d.setup.serviceRadiusChoice(); got != 50 {
		t.Errorf("the choice reads %d, want 50", got)
	}
}

// AND THE BOUNDS ARE ENFORCED AT THE SAVE, not at the keystroke.
//
// A FIELD THAT REFUSED "1" ON THE WAY TO "100" would be fighting the operator
// over a number they had not finished writing — so the guard is on the write,
// and an out-of-range value leaves the stored one standing.
func TestTheServiceRadiusRefusesWhatIsOutOfBounds(t *testing.T) {
	for _, tc := range []struct {
		typed string
		wrote bool
	}{
		{"2", true}, {"100", true}, {"50", true},
		{"1", false}, {"101", false}, {"0", false}, {"", false},
	} {
		var got int
		d := stationSetup(t, rowServiceRadius)
		d.cfg.ServiceRadiusMi = 25
		d.cfg.SetServiceRadius = func(v int) { got = v }
		d.setup.serviceMi = tc.typed

		if cmd := d.serviceRadiusApplyCmd(); cmd != nil {
			cmd()
		}
		if wrote := got != 0; wrote != tc.wrote {
			t.Errorf("%q: wrote=%v (%d), want wrote=%v", tc.typed, wrote, got, tc.wrote)
		}
	}
}

// AND A SAVE THAT CHOSE NOTHING DOES NOT END THE FALLBACK (D-72).
//
// A station with no transmitter of its own BORROWS the listener's default
// location and moves with it. Writing that borrowed value back as the station's
// OWN would silently end the split — the station would stop following the
// watchlist the first time anyone opened Settings and pressed enter, and nothing
// on screen would say so.
func TestASaveWithNoChoiceLeavesTheStationBorrowing(t *testing.T) {
	wrote := false
	d := stationSetup(t, rowTransmitter)
	d.cfg.SetTransmitter = func(snapshot.LocationRef) { wrote = true }
	d.setup.txRef = nil // the operator opened Settings and changed nothing

	if cmd := d.transmitterApplyCmd(); cmd != nil {
		cmd()
	}
	if wrote {
		t.Error("a save with no choice wrote a transmitter and ended the fallback")
	}
}

// A BORROWED EPICENTRE SAYS SO, ON THE VALUE (D-115, HUM LEAD 2026-09-13).
//
// "Borrowing Observer's location when user hasn't set the Broadcaster Location is
// fine - as long as we inform the user in some way."
//
// SO THE HINT IS ONE SENTENCE FOR BOTH STATES and the fact moves to the VALUE,
// which is the thing it is about: this place is not a choice the operator made,
// and it WILL move under them the next time they change their watchlist. A
// borrowed epicentre shown as a plain answer is the setting lying about its own
// provenance.
func TestABorrowedEpicentreSaysSo(t *testing.T) {
	o := render.Opts{ASCII: true}

	borrowing := stationSetup(t, rowTransmitter)
	borrowing.cfg.Transmitter = nil
	got := stripANSITest(strings.Join(borrowing.setupTransmitterLines(o, " "), "\n"))
	if !strings.Contains(got, "following your default location") {
		t.Errorf("a borrowed epicentre is shown as a choice the operator made:\n%s", got)
	}

	own := stationSetup(t, rowTransmitter)
	own.cfg.Transmitter = &snapshot.LocationRef{Label: "Bonsall, CA", Zip: "92003", Lat: 33.28, Lon: -117.23}
	got = stripANSITest(strings.Join(own.setupTransmitterLines(o, " "), "\n"))
	if strings.Contains(got, "following your default location") {
		t.Errorf("a station with its own transmitter is told it is borrowing:\n%s", got)
	}
	if !strings.Contains(got, "Bonsall, CA") {
		t.Errorf("the station's own transmitter is not shown:\n%s", got)
	}

	// AND THE HINT IS THE SAME SENTENCE EITHER WAY, which is what was asked for.
	for _, d := range []Dashboard{borrowing, own} {
		if !strings.Contains(stripANSITest(strings.Join(d.setupTransmitterLines(o, " "), "\n")),
			"Broadcasting location - Enter City, ST or Zip") {
			t.Error("the row's hint is not the HUM LEAD's wording")
		}
	}
}

// FR-9.4 — THE STORAGE BOUNDARY IS STATED WHERE THE OPERATOR SETS THE TOWER:
// the transmitter is a real antenna at metre precision, and the question that
// asks for it says what the application does with it, in the support text the
// operator reads while answering — including the one place the pair does go,
// the National Weather Service's forecast request every watched place makes.
func TestTheTransmitterQuestionStatesTheStorageBoundary(t *testing.T) {
	o := render.Opts{ASCII: true}
	got := stripANSITest(strings.Join(stationSetup(t, rowTransmitter).setupTransmitterLines(o, " "), "\n"))
	for _, want := range []string{"stays on this machine", "National Weather Service", "debug dump"} { // bounded by the phrase list (P10-02)
		if !strings.Contains(got, want) {
			t.Errorf("the transmitter question does not say %q; the operator is owed the storage boundary (FR-9.4):\n%s", want, got)
		}
	}
}
