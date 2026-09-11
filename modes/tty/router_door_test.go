package tty

// F-72: THE CONSOLE MUST NOT BE A ONE-WAY DOOR.
//
// The swap gate correctly refuses to LEAVE the console while the station is
// running — the ratified STANDBY-first rule (FR-1.4, D-1). What made that a
// trap rather than a rule is that nothing could reach STANDBY: `Power.OffAir`
// had no producer at all, so an operator who arrived with the radio playing had
// no way off the surface but killing the process.
//
// IT WAS LATENT ONLY BECAUSE ARRIVING WAS ALSO IMPOSSIBLE — `Router.keys` was
// never assigned, so no key reached the swap at all. That is why the keymap and
// the control land in ONE change: installing the keymap alone is what makes the
// trap live, and it would have surfaced as a different batch's bug.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/term"
)

// station records what the console asked of the station's state.
type station struct {
	asked []lineup.Power
	// bed is every cut the console asked for, in order (D-78).
	bed []bool
}

func (s *station) GoOnAir()          { s.asked = append(s.asked, lineup.Running) }
func (s *station) GoToStandby()      { s.asked = append(s.asked, lineup.OffAir) }
func (s *station) CutBed(toBed bool) { s.bed = append(s.bed, toBed) }

// withStation hands the router its control THE WAY PRODUCTION DOES — through
// the message, not by assigning the field.
//
// A PLANT MADE THIS NECESSARY. The first version of these tests set `r.station`
// directly, so a mutation that dropped the message on the floor left every one
// of them green: they exercised the toggle and never the DELIVERY of the thing
// the toggle needs. Same shape as a stubbed seam — the test drives past the
// wire it is supposed to prove.
func withStation(t *testing.T, r Router, s Station) Router {
	t.Helper()
	m, _ := r.Update(StationControlMsg{Control: s})
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	return out
}

// keyFor turns a BINDING STRING back into the key message the program would
// deliver, and asserts the round trip.
//
// THE ROUND TRIP IS THE POINT. A test that hand-wrote `tea.KeyPressMsg{Code:
// 'o', Mod: tea.ModCtrl}` would pass whatever the keymap said, so it would keep
// passing after a rebind that broke the binding it claims to exercise.
func keyFor(t *testing.T, binding string) tea.KeyPressMsg {
	t.Helper()
	var k tea.KeyPressMsg
	switch {
	case strings.HasPrefix(binding, "ctrl+"):
		k = tea.KeyPressMsg{Code: rune(binding[len("ctrl+")]), Mod: tea.ModCtrl}
	case binding == "shift+enter":
		k = tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	default:
		k = tea.KeyPressMsg{Code: rune(binding[0]), Text: binding}
	}
	if k.String() != binding {
		t.Fatalf("the test cannot press %q: it builds a key that reads as %q, so it would exercise "+
			"a binding nobody has", binding, k.String())
	}
	return k
}

// pressAction sends the FIRST key bound to an action, so the test cannot drift
// from the keymap by naming a literal.
func pressAction(t *testing.T, r Router, a term.Action) Router {
	t.Helper()
	bind, ok := broadcasterKeyMap()[a]
	if !ok || len(bind.Keys) == 0 {
		t.Fatalf("%s has no binding", a)
	}
	m, _ := r.Update(keyFor(t, bind.Keys[0]))
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	return out
}

func TestAnOperatorCanReachTheConsoleAndLeaveItAgain(t *testing.T) {
	s := &station{}
	r := withStation(t, NewRouter(Dashboard{}), s)
	r.broadcaster, _ = r.broadcaster.Update(StationMsg{Power: lineup.Running})

	r = pressAction(t, r, actSwapBroadcaster)
	if r.active != SurfaceBroadcaster {
		t.Fatal("ctrl+b must reach the console: with no keymap installed it reached nothing at all")
	}

	// The ratified rule holds: a live station is not left.
	r = pressAction(t, r, actSwapObserver)
	if r.active != SurfaceBroadcaster {
		t.Fatal("STANDBY-first is the rule, and it must still refuse")
	}
	if r.refusal == "" {
		t.Error("and the refusal is shown, not silent")
	}

	// AND THERE IS A WAY OUT. The control the refusal points at must exist.
	r = pressAction(t, r, actStationToggle)
	if len(s.asked) != 1 || s.asked[0] != lineup.OffAir {
		t.Fatalf("the toggle asks the station for STANDBY; got %v", s.asked)
	}

	// The station complies, and the console is told by the Director.
	r.broadcaster, _ = r.broadcaster.Update(StationMsg{Power: lineup.OffAir})
	r = pressAction(t, r, actSwapObserver)
	if r.active != SurfaceObserver {
		t.Error("with the station in STANDBY the operator may leave — without this the console is a " +
			"surface with no controls and no exit but killing the process")
	}
}

// THE TOGGLE IS A TOGGLE. From standby it asks to go back on the air.
func TestTheToggleGoesBothWays(t *testing.T) {
	s := &station{}
	r := withStation(t, NewRouter(Dashboard{}), s)
	r.active = SurfaceBroadcaster
	r.broadcaster, _ = r.broadcaster.Update(StationMsg{Power: lineup.OffAir})

	r = pressAction(t, r, actStationToggle)

	if len(s.asked) != 1 || s.asked[0] != lineup.Running {
		t.Errorf("from dead air the control puts the station back on; got %v", s.asked)
	}
}

// THE CONSOLE'S KEYS ARE THE CONSOLE'S. A key pressed on Observer must not
// silence the station.
func TestTheStationToggleDoesNothingFromObserver(t *testing.T) {
	s := &station{}
	r := withStation(t, NewRouter(Dashboard{}), s)
	r.broadcaster, _ = r.broadcaster.Update(StationMsg{Power: lineup.Running})

	r = pressAction(t, r, actStationToggle)

	if len(s.asked) != 0 {
		t.Errorf("Observer's surface does not run the station; got %v", s.asked)
	}
}

// THE BED'S CONTROLS REACH THE STATION (D-78).
//
// `[B]` IS THE FIRST PRODUCTION CALLER `lineup.CutOver` HAS EVER HAD. The
// Director has modelled the cut-over since 0.14.0 — FR-4.2, D-11, D-32 — and
// nothing emitted it, so `bed.carries` was false for the life of every process
// and the pause it governs had never once happened.
//
// DRIVEN THROUGH THE KEYS, for the reason this file's own helper exists: a test
// that called `cutBed` directly would pass over an unbound control, which is
// exactly the state these three were in.
func TestTheBedsControlsReachTheStation(t *testing.T) {
	s := &station{}
	var stepped []int
	d, err := NewDashboard(Config{StepBedRelay: func(by int) { stepped = append(stepped, by) }})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = withStation(t, NewRouter(d), s)
	m, _ = m.Update(keyPress(t, "ctrl+b")) // the controls are the console's

	m, _ = m.Update(keyPress(t, "b"))
	if len(s.bed) != 1 || !s.bed[0] {
		t.Fatalf("[b] cuts the programme TO the bed; asked %v", s.bed)
	}
	// AND IT IS A DIRECTION, NOT A TOGGLE THE CONSOLE DECIDES. Told the bed is
	// carrying, the next press asks for the other way.
	m, _ = m.Update(BedMsg{Carrying: true})
	m, _ = m.Update(keyPress(t, "b"))
	if len(s.bed) != 2 || s.bed[1] {
		t.Errorf("[b] on a carrying bed cuts BACK; asked %v", s.bed)
	}

	m, _ = m.Update(keyPress(t, "right"))
	m, _ = m.Update(keyPress(t, "left"))
	if len(stepped) != 2 || stepped[0] != 1 || stepped[1] != -1 {
		t.Errorf("the arrows step the relay selection; got %v", stepped)
	}

	// AND NONE OF THEM ACTS FROM OBSERVER, where the bed is not drawn.
	r := m.(Router)
	r.active = SurfaceObserver
	before, steps := len(s.bed), len(stepped)
	after, _ := r.Update(keyPress(t, "b"))
	_, _ = after.Update(keyPress(t, "right"))
	if len(s.bed) != before || len(stepped) != steps {
		t.Error("a bed control acted from a surface where the operator cannot see what they did")
	}
}
