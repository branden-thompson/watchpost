package app

// air_boundary_test.go — the closed set, and the behaviour behind it (D-91).
//
// THE PROPERTY: every seam a surface can reach this package through is
// CLASSIFIED for what it can do to the air, and every seam classified as the
// monitor's is actually REFUSED while the console holds it.
//
// TWO HALVES, BECAUSE MEMBERSHIP IS NOT BEHAVIOUR. D-74's own retro records why:
// plant `y4` — "the deck plays while the console owns the air" — SURVIVED,
// because the tests asserted `monitorHasTheAir()`, the PREDICATE, and never that
// the audio was actually skipped. A completeness gate alone would repeat that
// exactly.

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// spySource is a liveSource that records the two things which reach a broadcast
// IN FLIGHT — which is the whole of what the air guard is about.
//
// IT LIVES HERE, NOT IN THE DOMAIN. The first draft of this test added
// `RepeatingForTest` to `synth.Source`; the HUM LEAD caught it, and he was right
// — every `ForTest` export in the tree is in `platform/`, and a domain does not
// learn that this package has tests. `app` declares what it needs from a source
// (livesource.go), so `app` can also say what a fake one does.
type spySource struct {
	loops   []bool
	recasts int
}

func (s *spySource) Loop(on bool) { s.loops = append(s.loops, on) }
func (s *spySource) Recast()      { s.recasts++ }

// The rest is inert: the guard is about the two above, and a fake that did more
// would be claiming to model a Source rather than to observe two calls.
func (s *spySource) SetResolver(func(cast.Role) (synth.Voice, error)) {}
func (s *spySource) SetHandoffLine(func(from, to string) string)      {}
func (s *spySource) Invalidate()                                      {}
func (s *spySource) Err() error                                       { return nil }
func (s *spySource) Rate() int                                        { return 22050 }
func (s *spySource) Cached() (int, int)                               { return 0, 0 }
func (s *spySource) Open(context.Context) io.Reader                   { return nil }

// airReach is what a member of the surface-to-app boundary can do to the air.
type airReach int

const (
	// airNone cannot change what is audible. Settings that are recorded, queries,
	// and anything whose effect lands at the NEXT render rather than this one.
	airNone airReach = iota

	// airMonitor is the OPERATOR'S OWN LISTENING, and it is refused while the
	// console holds the air. This is the bucket the ruling is about.
	airMonitor

	// airProgramme is the STATION's own, and it must work on the console —
	// guarding it would break the surface it belongs to.
	airProgramme

	// airShared reaches the air from either surface, deliberately.
	airShared

	// airGatedDownstream reaches the air and already asks, further in. Recorded
	// rather than re-guarded: a second check would be a second carrier.
	airGatedDownstream

	// airDeclares is the boundary member that MOVES the air. It cannot be gated
	// on the air without making the air unreachable.
	airDeclares

	// numAirReaches bounds the set; it is not itself a reach.
	numAirReaches
)

// airMember is one member of the boundary and why it is classified as it is.
type airMember struct {
	reach airReach
	why   string
}

// airBoundary is EVERY func seam of `tty.Config` and EVERY method of `tty.Radio`
// — the whole boundary between a surface and this package — with what each can
// do to the air.
//
// IT IS A CLOSED SET AND THE GATE PROVES IT (air_boundary_test.go): a seam added
// to `tty.Config` with no row here FAILS, and a row naming a seam that no longer
// exists fails too. That is the one list-shaped thing in this repo that has not
// gone stale — `reachabilityBaseline`'s shape — and it is chosen because three
// hand-written lists have rotted in this package's history.
//
// MEMBERSHIP IS NOT BEHAVIOUR. A row saying `airMonitor` is a claim that the seam
// is REFUSED while the console holds the air, and the gate cannot see that by
// reflection — so every `airMonitor` row has a behaviour test beside it. D-74's
// own retro is why: plant `y4` survived because the tests asserted the PREDICATE
// and never that the audio was actually skipped.
var airBoundary = map[string]airMember{
	// --- tty.Radio: the monitor's control surface -------------------------
	"Radio.Tune":      {airMonitor, "the operator tunes their own listening; the Director's rotation uses the lower-case `tune`"},
	"Radio.Stop":      {airMonitor, "the operator stops listening; the swap's own silencing uses `stopMonitor`"},
	"Radio.SetRepeat": {airMonitor, "`src.Loop` reaches whatever is running, and per BD-9 that is the CARD during a main-track read"},
	"Radio.SetMode":   {airMonitor, "a mode change re-tunes what is playing"},
	"Radio.SetVolume": {airShared, "HUM LEAD 2026-09-12: \"one volume setting for the app\" — the Router already mirrors it to both surfaces"},

	// --- tty.Config: reaches the air --------------------------------------
	"SetVoice":      {airMonitor, "a saved root is a cast change, and a recast is applied to the LIVE source"},
	"SetCast":       {airMonitor, "`src.Recast()` — its own comment says \"the listener is waiting to hear it\", and on the console the listener is the AUDIENCE"},
	"PreviewVoice":  {airMonitor, "an audition mixed over the output; on the console that output is the station's. See the note in air_boundary_test.go — this one has a UX consequence"},
	"SetRelayDwell": {airMonitor, "re-sends the repeat mode so the Director hears the new dwell, and that reaches `src.Loop`"},
	"NarrateEvent":  {airMonitor, "the [w] window's [space]: an operator-initiated read that ducks the broadcast. NOT the hazard rail, which is exempt from air ownership by D-74"},
	"EndEventRead":  {airMonitor, "stops the read NarrateEvent started; paired with it, and a stop that outlived its start would leave the window's mark on a read nobody can end"},
	"ReadReport":    {airGatedDownstream, "goes through `needsRead`, which asks `monitorHasTheAir()` — the one entry that already did"},
	"StepBedRelay":  {airProgramme, "the console's own bed selector (D-90); guarding it would break the control it belongs to"},
	"TuneRelay":     {airProgramme, "the relay-fault window's pick. HUM LEAD 2026-09-12 ruled it must stay usable and be routed correctly — F-101, stage C"},
	"OnSurface":     {airDeclares, "`takeTheAir` — this is the thing that MOVES the air, so it cannot be gated on it"},
	"InjectAlert":   {airShared, "feeds the hazard RAIL, which D-74 exempts from air ownership deliberately: hazards read in either mode"},

	// --- tty.Config: cannot reach the air ---------------------------------
	"SetTones":       {airNone, "deliberately NOT a recast — the in-tree standard: \"[M] must be instant and must not disturb a broadcast in flight\""},
	"SetRelayLang":   {airNone, "writes a field; takes effect on the NEXT tune, deliberately, so a language change does not cut a sentence"},
	"SetAlertRadius": {airNone, "a filter bound; the rail re-scopes without touching the engine"},
	"SetTheme":       {airNone, "colour"},
	"SetUI":          {airNone, "display preferences"},
	"Setup":          {airNone, "persists the default location and the FIRMS key, and keys the live provider"},
	"Commit":         {airNone, "persists the watchlist and re-stations; publishes the area, touches no engine"},
	"Resolve":        {airNone, "a location lookup"},
	"Suggest":        {airNone, "a search"},
	"Hydrate":        {airNone, "an hourly forecast fetch for a RECENT row"},
	"Voices":         {airNone, "lists what is installed"},
	"VoiceInstalled": {airNone, "a query"},
	"Spectrum":       {airNone, "reads the visualiser tap"},
	"FIRMSKey":       {airNone, "a key hint for the Settings window"},
	"Stats":          {airNone, "the [S] counters"},
}

// TestEverySurfaceSeamIsClassifiedForTheAir is the completeness half, and it
// ratchets BOTH ways: a seam with no row fails, and a row naming a seam that no
// longer exists fails too.
//
// DERIVED, NEVER LISTED. Three hand-written lists have rotted in this package's
// history — `handleNav`'s scrolling windows, `modalLines`' default arm, and the
// air itself, where one entry of nineteen asked. A list of seams maintained by
// hand would rot in exactly the same way, and this is the shape that has not:
// `reachabilityBaseline`'s.
func TestEverySurfaceSeamIsClassifiedForTheAir(t *testing.T) {
	seen := map[string]bool{}

	// Every FUNC field of tty.Config. Data cannot reach the air: a `string` or a
	// `bool` is read by whoever asks for it and does nothing on its own.
	cfg := reflect.TypeOf(tty.Config{})
	for i := range cfg.NumField() {
		f := cfg.Field(i)
		if f.Type.Kind() != reflect.Func || f.PkgPath != "" {
			continue
		}
		seen[f.Name] = true
		if _, ok := airBoundary[f.Name]; !ok {
			t.Errorf("tty.Config.%s is a seam into this package and nothing says what it can do to "+
				"the air: classify it in airBoundary, with a reason.  A seam nobody classified is how "+
				"nineteen paths came to have one guard between them (D-91)", f.Name)
		}
	}

	// And every method of tty.Radio — the monitor's control surface, and the
	// other half of the boundary.
	radio := reflect.TypeOf((*tty.Radio)(nil)).Elem()
	for i := range radio.NumMethod() {
		name := "Radio." + radio.Method(i).Name
		seen[name] = true
		if _, ok := airBoundary[name]; !ok {
			t.Errorf("tty.Radio.%s is a seam into this package and has no airBoundary row",
				radio.Method(i).Name)
		}
	}

	for name, m := range airBoundary {
		if !seen[name] {
			t.Errorf("airBoundary has a row for %q, which is no longer a seam — delete it, or the "+
				"table describes a boundary that has moved (%s)", name, m.why)
		}
		if m.reach < 0 || m.reach >= numAirReaches {
			t.Errorf("%q carries reach %d, which is not one of the declared kinds", name, m.reach)
		}
		if m.why == "" {
			t.Errorf("%q is classified with no reason: a classification nobody can check is a "+
				"silencer, not a decision", name)
		}
	}
}

// --- the behaviour half -------------------------------------------------

// deckOnTheConsole is a deck whose air has been taken by the console, with a
// live source in it — the state every guard below is about.
func deckOnTheConsole(t *testing.T, console bool) (*radioDeck, *spySource) {
	t.Helper()
	src := &spySource{}
	d := &radioDeck{source: src, air: func() bool { return !console }}
	if d.monitorHasTheAir() == console {
		t.Fatalf("the fixture's air is the wrong way round; this test would measure nothing")
	}
	return d, src
}

// THE ONE THE HUM LEAD'S RULING WAS MEASURED ON. Observer's repeat mode reached
// `src.Loop`, and `d.source` during a main-track read IS the Broadcaster's card
// (BD-9) — so the card on the air looped and the line-up never advanced.
func TestObserversRepeatModeCannotLoopTheCardOnTheAir(t *testing.T) {
	d, src := deckOnTheConsole(t, true)
	d.SetRepeat(tty.RepeatOne, nil)
	if len(src.loops) != 0 {
		t.Errorf("Observer's repeat mode reached the programme's own source (%v): the card on the air "+
			"would read for ever and the line-up would never advance", src.loops)
	}
	// AND IT STILL APPLIES ON OBSERVER, or the guard has broken the setting
	// rather than scoped it.
	d, src = deckOnTheConsole(t, false)
	d.SetRepeat(tty.RepeatOne, nil)
	if len(src.loops) != 1 || !src.loops[0] {
		t.Errorf("the guard refused the monitor's own repeat mode on the monitor's own air: %v", src.loops)
	}
}

// A HARD RECAST OF THE CARD ON THE AIR is Observer reaching through the Settings
// window to change what the station is saying mid-sentence.
func TestObserversCastChangeCannotRecastTheCardOnTheAir(t *testing.T) {
	d, src := deckOnTheConsole(t, true)
	d.setCast(cast.Config{Root: "Samantha"})
	if src.recasts != 0 {
		t.Errorf("a cast save hard-recast the card on the air %d time(s): Observer reaching through "+
			"the Settings window to change what the station is saying mid-sentence", src.recasts)
	}
	d, src = deckOnTheConsole(t, false)
	d.setCast(cast.Config{Root: "Samantha"})
	if src.recasts != 1 {
		t.Errorf("the guard refused the monitor's own recast on the monitor's own air: %d", src.recasts)
	}
}

// THE SWAP'S SILENCING IS NOT RE-TESTED HERE, DELIBERATELY.
//
// `air_test.go`'s "taking the air to the console did not stop the monitor" already
// owns that property, drives it through `takeTheAir`, and REPORTED IT the moment
// the guard went on `Stop` — which is how the trap was found. A second assertion
// here would be a weaker copy of a test that has already proved it can fail.

// AND THE MONITOR'S OWN TUNE STOPS AT THE CONSOLE (mZ4).
//
// `tune`'s FIRST act is to take the location and bump the generation, before any
// audio is reached — so a deck whose `ref` and `gen` have not moved is a deck that
// did not tune. Asserting the guard's PREDICATE instead would be D-74's `y4` all
// over again, which is what a surviving mutant reported here.
func TestTheMonitorsTuneStopsAtTheConsole(t *testing.T) {
	d, _ := deckOnTheConsole(t, true)
	before := d.gen

	d.Tune(snapshot.LocationRef{Label: "Oceanside, CA"})

	if d.ref.Label != "" || d.gen != before {
		t.Errorf("Observer's tune reached the engine while the console held the air: ref=%q gen %d->%d",
			d.ref.Label, before, d.gen)
	}
}

// THE OBSERVER HALF IS NOT ASSERTED HERE, and that is deliberate: `tune` resolves
// against a live NWS provider, so a bare deck cannot run it — the unguarded path
// panics on the nil provider before it reaches any audio. The existing tune tests
// own "the monitor can still tune on its own air"; what THIS owns is that it
// cannot on the console's.
//
// THE MUTANT IS STILL CAUGHT EITHER WAY (mZ4): removing the guard sends this
// fixture straight into that nil provider, and the harness calls a crash CAUGHT.

// AND SO DOES THE VOICE AUDITION (mZ6).
//
// THE GUARD RETURNS BEFORE ANYTHING ELSE CAN HAPPEN, and a nil composer is how
// that is measured: unguarded, `PreviewVoice` reaches `d.composer.SamplePCM` and
// panics on it. Guarded, it returns cleanly. The panic IS the observation — the
// harness calls a crash CAUGHT — and it needs no engine, no voice and no install.
func TestTheVoiceAuditionStopsAtTheConsole(t *testing.T) {
	d, _ := deckOnTheConsole(t, true)
	var notes []string
	d.note = func(s string) { notes = append(notes, s) }

	d.PreviewVoice("Samantha")

	// THE GUARD RETURNS BEFORE ANY WORK. Unguarded, the very next thing is a real
	// synthesis — `SayVoice.Say` shells out — and then `engine.Audition` on a nil
	// engine. Either would be seconds of audio or a crash in a unit test, so the
	// note is the observation: it is the ONLY thing a refused preview does.
	if len(notes) != 1 || !strings.Contains(notes[0], "unavailable") {
		t.Errorf("a voice preview was not refused while the console held the air; notes=%v\n"+
			"an audition is mixed over the output, and on the console that output is the station's",
			notes)
	}
}
