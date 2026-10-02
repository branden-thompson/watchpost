package lineup

import "testing"

// onAirAnywhere asks whether ANY lane is reading — the question the tests that
// use it ask, where which lane does not matter to what they assert.
//
// A TEST HELPER, NOT A METHOD ON THE TYPE. In production the question is always
// about a LANE — the rail speaks over the programme — and a caller that does not
// say which one it means is the hazard D-82 rules out: asking "is anything
// reading anywhere" would keep a tornado warning off the air while a weather
// report held it. Naming a lane in tests about something else would claim the
// lane mattered to what they assert, and it does not.
func onAirAnywhere(l Lineup) (Card, bool) {
	for t := Track(0); t < numTracks; t++ { // bounded by the registry (P10-02)
		if c, on := l.OnAir(t); on {
			return c, true
		}
	}
	return Card{}, false
}

// reading drives one card of `slot` all the way onto the air and LEAVES IT
// THERE, which is what `seedRead` deliberately does not do: it finishes the
// read. Every test below is about what happens while a card is still speaking.
func reading(t *testing.T, d Director, id string, slot Slot) Director {
	t.Helper()
	d = seedCard(t, d, id, id, slot)
	if slot.textAtStandby() {
		d, _ = d.Step(Built{ID: id, Script: Say("the words for " + id + ".")})
	}
	d, _ = d.settle()
	return d
}

// THE RAIL INTERRUPTS THE PROGRAMME (D-82, HUM LEAD 2026-09-11: "breaking alerts
// need to interrupt reports").
//
// With "at most one card holds the air" held ACROSS BOTH TRACKS and asked
// BEFORE choosing a card, a rail card could not take the air while a report was
// reading, and a hazard would wait out the weather — as long as a location
// report.
func TestAHazardTakesTheAirOverAReportThatIsReading(t *testing.T) {
	d := New(Settings{Max: 5, Depth: 10}, cutoverBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d = reading(t, d, "report", LocationReport)

	if c, on := d.lineup.OnAir(MainTrack); !on || c.ID != "report" {
		t.Fatalf("the report must be reading before the hazard arrives; the main track holds %+v", c)
	}

	d = reading(t, d, "alert", BreakingAlert)

	if c, live := d.lineup.OnAir(AlertRail); !live || c.ID != "alert" {
		t.Fatalf("the hazard must take the air over the report; the rail holds %+v", c)
	}
	// AND THE REPORT IS STILL ON THE AIR, which is the half that makes this
	// pause-and-resume rather than drop-and-restart (D-24, and the HUM LEAD's
	// ruling of 2026-09-11). Its Speak is still running on a worker; the engine
	// HOLDS its player — a rendered report waits rather than dipping — and lets
	// go again when the rail is dry. Taking it off the air here would be the
	// cheaper design D-24 declines.
	if c, on := d.lineup.OnAir(MainTrack); !on || c.ID != "report" {
		t.Errorf("the report left the air for the hazard; it must HOLD and resume mid-sentence, not restart")
	}
}

// ONE VOICE PER LANE. Two cards on the SAME lane would be two voices, which is
// what the invariant is about.
func TestASecondCardOnOneLaneStillCannotTakeTheAir(t *testing.T) {
	d := New(Settings{Max: 5, Depth: 10}, cutoverBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d = reading(t, d, "first", LocationReport)
	d = reading(t, d, "second", LocationReport)

	if c, on := d.lineup.OnAir(MainTrack); !on || c.ID != "first" {
		t.Fatalf("the lane's air is held by %+v; the second report must wait its turn", c)
	}
}

// THE MAIN TRACK STILL WAITS FOR THE RAIL (DR-3), and the precedence must NOT
// become symmetrical: the rail interrupts the programme, and normal programming
// resumes only when the rail is dry.
func TestAReportDoesNotTakeTheAirWhileTheRailIsReading(t *testing.T) {
	d := New(Settings{Max: 5, Depth: 10}, cutoverBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d = reading(t, d, "alert", BreakingAlert)
	d = reading(t, d, "report", LocationReport)

	if c, on := d.lineup.OnAir(MainTrack); on {
		t.Errorf("a report took the air under a hazard still reading: %+v", c)
	}
}

// AND STANDBY STILL TAKES THE REPORT OFF, NOT THE HAZARD (power.go). Asking the
// main track's own air is what makes that read as the rule it is, rather than as
// "whatever is reading, unless it is the rail's".
func TestStandbyTakesTheReportOffTheAirAndLeavesTheHazardReading(t *testing.T) {
	d := New(Settings{Max: 5, Depth: 10}, cutoverBase())
	d, _ = d.Step(Powered{To: Running})
	d, _ = d.Step(Aired{To: AirProgramme})
	d = reading(t, d, "report", LocationReport)
	d = reading(t, d, "alert", BreakingAlert)

	d, _ = d.Step(Powered{To: OffAir})

	if c, on := d.lineup.OnAir(MainTrack); on {
		t.Errorf("standby left the report reading: %+v", c)
	}
	if c, live := d.lineup.OnAir(AlertRail); !live || c.ID != "alert" {
		t.Errorf("standby took the HAZARD off the air; the rail keeps draining while the programme is stopped (DR-3)")
	}
}
