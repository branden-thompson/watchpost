package tty

// router.go — the model the program holds (FR-1.1, D-2).
//
// ONE MODEL, TWO SURFACES. `Dashboard` was handed to tea.NewProgram directly,
// so there was no seam a second surface could arrive at. The Router is that
// seam and nothing more: it owns which surface is active and the fan-out of
// the messages that belong to the PROGRAM rather than to either surface.
//
// P0'S CLAIM IS THAT NOTHING CHANGES. Observer renders and behaves exactly as
// it did; the Router is a delegation. The second surface arrives in P1, and
// the swap gate it needs arrives in P2 — until then `canSwap` fails closed,
// because a gate whose state source is a stub must refuse rather than permit
// (the P1->P2 window, PLAN red team).

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/term"
)

// Surface names which UI is on screen.
type Surface int

const (
	// SurfaceObserver is the zero value because Observer is what starts.
	SurfaceObserver Surface = iota
	SurfaceBroadcaster

	// numSurfaces bounds the set; it is not itself a surface. A guard walks
	// the type rather than a hand-written list, so adding a surface cannot
	// leave a check silently green (INST-1).
	numSurfaces
)

// The console's own actions. They are ACTIONS rather than literal keys so a
// user can rebind them (FR-1.5) through the same table Observer's use.
const (
	actSwapObserver    term.Action = "swap-observer"
	actSwapBroadcaster term.Action = "swap-broadcaster"

	// actDiagnostics reaches the ctrl+d window FROM THE CONSOLE (D-58). It is
	// the same window the Observer draws — see Router.View.
	actDiagnostics   term.Action = "diagnostics"
	actStationToggle term.Action = "station-toggle"

	// THE BED'S OWN CONTROLS (D-78). `[B]` cuts the programme between the
	// station's line-up and its bed; the arrows move through the relays the
	// station's fence reaches. The reference mock has drawn all three since the
	// first wave and none of them was bound to anything.
	actBedCut    term.Action = "bed-cut"
	actBedPrev   term.Action = "bed-prev"
	actBedNext   term.Action = "bed-next"
	actQueuePrev term.Action = "queue-prev"
	actQueueNext term.Action = "queue-next"

	// actGainUp and actGainDown are the station's output level — Observer's VOL
	// under the station's own word (HUM LEAD, 2026-09-10). They are FORWARDED
	// rather than reimplemented: the Dashboard owns the level, the step, the
	// chip flash and the engine call, and a second owner would be a second
	// answer to how loud the station is.
	actGainUp   term.Action = "gain-up"
	actGainDown term.Action = "gain-down"

	// THE MASTHEAD ADVERTISES THESE, so they must reach something (D-65). They
	// were printed as TEXT and bound to nothing: the console named five controls
	// and answered to none of them, which is a UI lying about what it can do.
	//
	// FORWARDED, NOT REBUILT. Settings, About, Status and Help are Observer's
	// own windows; the Router composites whichever one opens over the console,
	// exactly as it does for ctrl+d.
	actSettings term.Action = "settings"
	actAbout    term.Action = "about"
	actStatus   term.Action = "status"
	actHelp     term.Action = "help"
	actQuit     term.Action = "quit"
)

// StationControlMsg hands the console the control it asks ON AIR / STANDBY
// through (FR-5.4).
//
// A MESSAGE, BECAUSE OF THE ORDER THINGS ARE BUILT IN. MasterControl is
// constructed with the program's own `Send`, so it cannot exist when the
// program's model does. Passing it through a constructor would mean building
// the router twice or the effector early, and both are worse than the console
// learning this the way it learns everything else: it is TOLD.
type StationControlMsg struct{ Control Station }

// Station is what the console may ask of the station's STATE (FR-5.4).
//
// TWO METHODS, NOT ONE WITH AN ARGUMENT. The console toggles between on the air
// and dead air; `Stopped` is Observer's control and not the operator's, so a
// setter taking a Power would let this surface ask for a state it has no
// business naming.
//
// MASTERCONTROL DECLARES, EVERYONE COMPLIES (MVS-D-78). The console does not
// change the power itself and does not hold a flag saying what it is: it ASKS,
// and learns the answer the same way it learns everything else — from the
// Director, through Publish (FR-5.1).
type Station interface {
	GoOnAir()
	GoToStandby()

	// CutBed moves the programme between the station's line-up and its bed
	// (D-78) — `[B]`, and the first production caller `lineup.CutOver` has ever
	// had. The Director has modelled this since 0.14.0 and nothing emitted it.
	//
	// TO THE BED IS THE ARGUMENT, not a toggle, for the reason the power's two
	// controls are two methods: a toggle asks the caller to know the current
	// state, and the console's whole discipline is that it holds no opinion of
	// its own — it draws what it is told.
	CutBed(toBed bool)
}

// broadcasterKeyMap is the console's bindings.
//
// EVERY ACTION CARRIES A NON-CHORD KEY (FR-1.6). The chord is the mnemonic
// one and it is NOT the only door: tmux takes ctrl+b by default and screen
// takes ctrl+a, and an operator whose multiplexer eats the chord would
// otherwise have no route back at all. D-3 left the exact chords to be
// finalised, so these are placeholders — but the non-chord route is a
// REQUIREMENT, not a placeholder, and it stays whatever the chords become.
func broadcasterKeyMap() term.KeyMap {
	return term.KeyMap{
		actSwapObserver:    {Keys: []string{"ctrl+o", "O"}, Help: "Observer"},
		actSwapBroadcaster: {Keys: []string{"ctrl+b", "B"}, Help: "Broadcaster"},
		// THE SAME BINDING THE DASHBOARD USES, deliberately: one key for one
		// thing, on every surface (D-56). An operator who learned ctrl+d in
		// Observer does not learn a second key here.
		actDiagnostics: {Keys: []string{"ctrl+d"}, Help: "Diagnostics"},
		// THE SAME KEYS OBSERVER USES, for the same reason ctrl+d is the same
		// key: one control, one binding, on every surface.
		actGainUp:        {Keys: []string{"+", "="}, Help: "Gain Up"},
		actGainDown:      {Keys: []string{"-"}, Help: "Gain Down"},
		actSettings:      {Keys: []string{"s"}, Help: "Settings"},
		actAbout:         {Keys: []string{"a"}, Help: "About"},
		actStatus:        {Keys: []string{"S"}, Help: "Status"},
		actHelp:          {Keys: []string{"?"}, Help: "Help"},
		actQuit:          {Keys: []string{"q"}, Help: "Quit"},
		actStationToggle: {Keys: []string{"shift+enter"}, Help: "ON AIR / STANDBY"},
		// THE BED'S OWN CONTROLS (D-78). `[B]` cuts the programme to the bed and
		// back; the arrows move through the relays the station's fence reaches.
		// LOWERCASE, BECAUSE `B` IS ALREADY A DOOR. FR-1.6 requires every action
		// to carry a non-chord key, and `B` is the Broadcaster swap's — an
		// operator whose multiplexer eats `ctrl+b` would otherwise have no route
		// to the console at all. One key means one thing (D-56), so the bed
		// takes `b` and the reference's `[ B ]` chip becomes `[ b ]`. Stated
		// rather than quietly re-bound: the mock is the record.
		actBedCut:  {Keys: []string{"b"}, Help: "Bed"},
		actBedPrev: {Keys: []string{"left"}, Help: "Previous Relay"},
		actBedNext: {Keys: []string{"right"}, Help: "Next Relay"},
		// THE QUEUE SCROLLS (D-87, HUM LEAD 2026-09-11): "that's why we have the
		// vertical scroll bar so that works like Observer — that section just
		// needs to be able to scroll up and down."
		//
		// THE ARROWS, LIKE EVERYTHING ELSE THAT MOVES THROUGH A LIST. Up and down
		// are unbound on the console and walk the table on Observer, so they
		// fall through exactly as the bed's do — one key, one meaning per
		// surface (D-56).
		actQueuePrev: {Keys: []string{"up"}, Help: "Scroll Up"},
		actQueueNext: {Keys: []string{"down"}, Help: "Scroll Down"},
	}
}

// consoleOwnsTheKeys reports whether the console's own controls may act.
//
// A WINDOW ON TOP OWNS THE KEYBOARD (D-58), and this switch runs BEFORE the
// check that enforces it — so a console control handled here reaches past an
// open window and acts on input meant for it.
//
// THE BED'S ARROWS HAD THIS BUG SINCE D-78 and nothing saw it: with the
// diagnostics or status window up, `←`/`→` stepped the relay instead of moving
// through the window. Adding the queue's `↑`/`↓` is what surfaced it, because
// those are the keys the modal-reachability gate drives — "a line the keyboard
// cannot bring on screen is not in the window". The fix is one predicate for
// both, which is what the bug was for: two controls asking the same question in
// two places, and only one of them asking it at all.
func (r Router) consoleOwnsTheKeys() bool {
	return r.active == SurfaceBroadcaster && !r.observer.ModalOpen()
}

// scrollStep is which way a queue key moves the window.
func scrollStep(a term.Action) int {
	if a == actQueuePrev {
		return -1
	}
	return 1
}

// Router holds the surfaces and delegates to the active one.
type Router struct {
	observer    Dashboard
	broadcaster Broadcaster
	active      Surface

	// keys are the console's own bindings, merged once and shared by value.
	//
	// ASSIGNED AT CONSTRUCTION, and it was not for a release (F-72). With it
	// nil the swap branch in Update is skipped entirely, so `ctrl+o` and
	// `ctrl+b` reached nothing and the console could not be arrived at — which
	// is the only reason the one-way door below was latent rather than live.
	keys term.KeyMap

	// station is how the console asks for ON AIR or STANDBY (FR-5.4).
	//
	// IT LANDS IN THE SAME CHANGE AS THE KEYMAP, deliberately. `canSwap`
	// refuses to leave a running station — the ratified rule — and until
	// something could produce `OffAir` there was no way to satisfy it. Install
	// the keymap without this and the console becomes a surface with no
	// controls and no exit but killing the process.
	station Station

	// refusal is why the last swap was refused, shown to the operator. A
	// refusal they cannot read is indistinguishable from a broken control.
	refusal string

	// onSurface tells the app which surface owns the air, so the alert rail can
	// be scoped to it (D-73). Nil in tests that do not care.
	onSurface func(Surface) tea.Cmd

	// relays steps the bed's selection through what the station's fence reaches
	// (D-78). Nil where there is no radio.
	relays func(by int) tea.Cmd
}

// NewRouter wraps Observer. The second surface arrives in P1.
func NewRouter(o Dashboard) Router {
	// The console inherits the SAME --ascii decision Observer was built with.
	// Two surfaces disagreeing about whether the terminal can draw a glyph
	// would be one setting with two carriers, which is the shape this codebase
	// has removed twice.
	b := NewBroadcaster()
	b.ascii = o.cfg.ASCII
	// AND THE SAME BUILD. The masthead names the version on both surfaces, and
	// two surfaces disagreeing about which build this is would be the same
	// two-carriers defect one field along.
	b.version = o.cfg.Version
	// AND THE STATION IT IS A CONSOLE FOR (D-72). The area MOVES, so changes
	// arrive as a message; this is the value it opens with.
	b.area = o.cfg.StationArea
	return Router{observer: o, broadcaster: b, active: SurfaceObserver,
		keys: broadcasterKeyMap(), onSurface: o.cfg.OnSurface, relays: o.cfg.StepBedRelay}
}

// Init delegates to the active surface. Observer asks for the terminal's
// background colour here, and dropping that would leave the frame painting
// against the wrong ground.
func (r Router) Init() tea.Cmd { return r.surface().Init() }

// programScoped reports whether a message describes the PROGRAM's world
// rather than either surface's business — the terminal's size, its colours,
// whether it has focus, whether the process was suspended.
//
// AN ENUMERATED SET, AND HONESTLY SO. INST-1 says the set a gate ITERATES is
// derived, never hand-written — but this set is a property of the FRAMEWORK,
// not of our code, and there is no type here to walk. It is written down the
// way a threshold is, and the guard beside it states what it cannot see.
//
// THE LAST FIVE ARE F-67 (0.16.0 P1). They had no production handler anywhere
// before this: focus, blur, suspend, resume, colour-profile. Fanning them
// costs nothing while nobody handles them, and it means the surface that
// eventually does will RECEIVE them rather than discover they were dropped at
// a seam. That is the structural half of F-67; handling them is still open.
func programScoped(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.WindowSizeMsg, tea.BackgroundColorMsg,
		tea.FocusMsg, tea.BlurMsg, tea.SuspendMsg, tea.ResumeMsg, tea.ColorProfileMsg:
		return true
	// THE SNAPSHOT GOES TO BOTH (D-59). Both surfaces draw the same masthead,
	// and its `Updated:` stamp and API summary come from here — so a console
	// that only learned the data while on screen would show a stale masthead the
	// instant it was swapped to, which is the exact argument `consoleScoped`
	// already makes for the schedule.
	case SnapshotMsg:
		return true
	}
	return false
}

// consoleScoped reports whether a message is the CONSOLE's own business,
// wherever the operator happens to be looking.
//
// A THIRD CATEGORY, and it earns its place for the same reason the fan-out
// does: a surface that only learns things while on screen is stale the
// instant it is swapped to — and here the stale things would be the running
// order and whether the station is on the air, which is the one screen an
// operator swaps to in order to trust.
func consoleScoped(msg tea.Msg) bool {
	switch msg.(type) {
	case LineupMsg, StationMsg, StationAreaMsg, BedMsg:
		return true
	}
	return false
}

// Update routes the message and keeps the Router as the program's model.
//
// PROGRAM-SCOPED MESSAGES GO TO BOTH SURFACES; everything else goes to the
// active one. An inactive surface holding a stale size renders wrong the
// instant it is swapped to, and the operator would meet a broken frame at
// exactly the moment they asked for it.
func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := r.update(msg)
	// THE LEVEL IS MIRRORED, NEVER OWNED TWICE (D-56). The Dashboard holds it —
	// it has the engine call, the step and the chip flash — and the console
	// draws the same number under its own word for it. Mirrored on EVERY update
	// rather than on a message, so the console cannot lag the control the
	// operator just pressed, and there is exactly one answer to how loud the
	// station is.
	if out, ok := m.(Router); ok {
		out.broadcaster.gain = out.observer.radioVolume
		// AND THE REFUSAL IS CARRIED TO THE SURFACE THAT WAS REFUSED. It was
		// recorded here and drawn nowhere — under a comment saying a refusal
		// the operator cannot read is indistinguishable from a broken control,
		// which is exactly how it was reported: "ctrl+o will soft lock
		// randomly" (HUM LEAD, UAT 2026-09-10).
		//
		// IT DIES WITH ITS REASON. The only refusal there is says the station
		// is ON AIR; once it is not, the sentence is false and must go, or the
		// operator is left reading about a state they have already left.
		if !out.stationIsLive() {
			out.refusal = ""
		}
		out.broadcaster.statusNote = out.refusal
		return out, cmd
	}
	return m, cmd
}

// update is the Router's own routing; Update wraps it to mirror what the two
// surfaces must agree on.
func (r Router) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// THE ROUTER KEEPS THE CONTROL, not the console, because the router is
	// where the key is looked up — one key-lookup site, which is the same
	// reason the swap is handled here (D-1).
	if m, ok := msg.(StationControlMsg); ok {
		r.station = m.Control
		return r, nil
	}
	if consoleScoped(msg) {
		var cmd tea.Cmd
		r.broadcaster, cmd = r.broadcaster.Update(msg)
		return r, cmd
	}
	if programScoped(msg) {
		var oc, bc tea.Cmd
		if m, c := r.observer.Update(msg); true {
			if d, ok := m.(Dashboard); ok {
				r.observer = d
			}
			oc = c
		}
		r.broadcaster, bc = r.broadcaster.Update(msg)
		return r, tea.Batch(oc, bc)
	}
	// A SWAP REQUEST GOES THROUGH canSwap AND NOWHERE ELSE. Handling it here,
	// before the message reaches either surface, is what keeps the D-1 rule to
	// one carrier — a binding that switched surfaces itself would be a second.
	if k, ok := msg.(tea.KeyPressMsg); ok && r.keys != nil {
		if a, bound := r.keys.Lookup(k.String()); bound {
			switch a {
			case actSwapObserver:
				return r.swapTo(SurfaceObserver)
			case actSwapBroadcaster:
				return r.swapTo(SurfaceBroadcaster)
			case actStationToggle:
				return r.toggleStation(), nil
			// THE BED'S CONTROLS ARE THE CONSOLE'S, AND THEY FALL THROUGH
			// EVERYWHERE ELSE (D-79). The Router looks a key up BEFORE either
			// surface sees it, so a case that returned unconditionally would
			// swallow the ARROWS on Observer — where they walk the table — and
			// the listener's navigation would simply stop working.
			//
			// NOT RETURNING IS THE FALL-THROUGH: execution continues past this
			// switch to the active surface, which is exactly what an unbound key
			// does.
			case actBedCut, actBedPrev, actBedNext:
				if r.consoleOwnsTheKeys() {
					return r.bedControl(a)
				}
			// THE QUEUE'S SCROLL, AND IT FALLS THROUGH FOR THE SAME REASON THE
			// BED'S ARROWS DO: on Observer these walk the table, and a case that
			// returned unconditionally would stop the listener's navigation.
			case actQueuePrev, actQueueNext:
				if r.consoleOwnsTheKeys() {
					r.broadcaster = r.broadcaster.scrollQueue(scrollStep(a))
					return r, nil
				}
			case actGainUp, actGainDown,
				actSettings, actAbout, actStatus, actHelp, actQuit:
				return r.throughToObserver(msg)
			case actDiagnostics:
				// FORWARDED TO THE SURFACE THAT OWNS THE WINDOW, and the
				// active surface does NOT change: the operator is here to
				// watch what the injection does to the console.
				return r.throughToObserver(msg)
			}
		}
	}
	// THE WINDOW ON TOP OWNS THE KEYS (D-58). While the diagnostics window is
	// composited over the console, its own navigation — arrows, enter, esc —
	// must reach IT and not the lanes beneath it. An arrow that promoted a card
	// while the operator was choosing a scenario would be the console acting on
	// input meant for the window over it.
	if r.active == SurfaceBroadcaster && r.observer.ModalOpen() {
		if _, isKey := msg.(tea.KeyPressMsg); isKey {
			return r.throughToObserver(msg)
		}
	}
	switch r.active {
	case SurfaceObserver:
		m, cmd := r.observer.Update(msg)
		if d, ok := m.(Dashboard); ok {
			r.observer = d
		}
		return r, cmd
	case SurfaceBroadcaster:
		b, cmd := r.broadcaster.Update(msg)
		r.broadcaster = b
		return r, cmd
	}
	return r, nil
}

// View renders the active surface, unchanged.
// View draws the active surface, and composites the diagnostics window over it
// when that window is open on another surface (D-58).
//
// ONE WINDOW, NOT TWO (D-56). The ctrl+d window is entirely Dashboard methods;
// giving the console its own would be a second injector UI, a second
// confirmation, and two places for the TEST EVENT wording to drift. The Router
// is where both surfaces are already held, so it is where they compose.
//
// THE CONSOLE STAYS DRAWN UNDERNEATH, which is the whole reason to reach the
// window from here: the operator injects an alert and WATCHES the takeover
// activate and drain.
func (r Router) View() tea.View {
	v := r.surface().View()
	if r.active == SurfaceBroadcaster && r.observer.ModalOpen() {
		v.Content = r.observer.OverlayDiagnostics(v.Content, r.broadcaster.width)
	}
	return v
}

// throughToObserver hands a message to the surface that owns the diagnostics
// window, WITHOUT changing which surface is drawn.
func (r Router) throughToObserver(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := r.observer.Update(msg)
	if d, ok := m.(Dashboard); ok {
		r.observer = d
	}
	return r, cmd
}

// surfaceView is the part of a surface the Router needs in order to start it
// and to draw it.
//
// DELIBERATELY NARROWER THAN tea.Model. Update is routed explicitly, with
// concrete types, so Broadcaster can return a Broadcaster rather than a
// tea.Model and the Router needs no type assertion to put it back. Observer
// keeps its tea.Model signature untouched — P0's claim was that nothing
// changes, and a signature change is a change.
type surfaceView interface {
	Init() tea.Cmd
	View() tea.View
}

// surface is the active surface. It exists so Init and View cannot disagree
// about which one is on screen — one accessor, not two switches.
func (r Router) surface() surfaceView {
	if r.active == SurfaceBroadcaster {
		return r.broadcaster
	}
	return r.observer
}

// swapTo moves to the surface if the gate permits, and records the refusal
// for the operator when it does not.
func (r Router) swapTo(to Surface) (Router, tea.Cmd) {
	ok, why := r.canSwap(to)
	if !ok {
		r.refusal = why
		return r, nil
	}
	r.refusal = ""
	r.active = to
	// THE AIR FOLLOWS THE SURFACE (D-73). Told HERE, in the one place a swap is
	// granted, rather than from the key handler — a second caller would be a
	// second answer to which surface owns the air, and the whole reason this
	// gate exists is that there is exactly one.
	//
	// WHAT COMES BACK IS A COMMAND, NOT A DONE DEED (D-79). Taking the air stops
	// the monitor's audio, and that reaches the player — which calls back into
	// the program. Run inline it would `Send` to a loop sitting inside this very
	// Update, and the app froze hard on `ctrl+b` until it was a command.
	if r.onSurface != nil {
		return r, r.onSurface(to)
	}
	return r, nil
}

// canSwap reports whether the operator may move to the given surface, and
// why not when the answer is no.
//
// THE ONE PLACE THE D-1 PRECONDITION IS CHECKED (FR-1.4). It lives here and
// not in the console's key handler for a reason this codebase has already
// paid for once: if the surface decided, every future path that could request
// a swap — a menu, a programmatic message, a second binding — would have to
// re-implement the same guard, and "two carriers of one rule" is the shape
// that produced the duck-lift bug.
//
// FAILS CLOSED on anything it does not understand. A corrupt surface value is
// refused rather than permitted, because the cost of a wrong refusal is an
// inconvenience and the cost of a wrong permission is audio left running with
// no owner — and there is no graceful stop anywhere in the tree to catch it.
func (r Router) canSwap(to Surface) (bool, string) {
	if to < 0 || to >= numSurfaces {
		return false, "that surface does not exist"
	}
	if to == r.active {
		return true, "" // already there; not a transition at all
	}
	// ARRIVING at the console is not the hazard the ruling bounds. Leaving a
	// LIVE station is.
	if to == SurfaceBroadcaster {
		return true, ""
	}
	if r.stationIsLive() {
		return false, "the station is ON AIR — go to STANDBY before leaving the console"
	}
	return true, ""
}

// toggleStation asks for the other station state, and asks NOTHING when the
// operator is not looking at the console.
//
// THE CONSOLE RUNS THE STATION; OBSERVER DOES NOT. A key that silenced the
// broadcast from the other surface would be a control acting where it is not
// drawn, and the operator would have no way to see what they had done.
//
// IT READS THE DIRECTOR'S POWER, NEVER A UI FLAG (FR-5.1). The console's own
// `power` came from `Publish`, so the toggle's direction is decided from what
// the schedule says is true rather than from what this surface last drew.
func (r Router) toggleStation() Router {
	if r.active != SurfaceBroadcaster || r.station == nil {
		return r
	}
	if r.stationIsLive() {
		r.station.GoToStandby()
		return r
	}
	r.station.GoOnAir()
	return r
}

// bedControl routes one of the console's bed keys.
//
// ONE DOOR FOR THE THREE OF THEM, so the "only on the console" rule is stated
// once rather than three times — and the day a fourth bed control arrives it
// cannot be added without passing the same gate.
func (r Router) bedControl(a term.Action) (Router, tea.Cmd) {
	switch a {
	case actBedCut:
		return r.cutBed(), nil
	case actBedPrev:
		return r.stepBed(-1)
	case actBedNext:
		return r.stepBed(1)
	}
	return r, nil
}

// cutBed moves the programme between the line-up and the bed (D-78).
//
// THE CONSOLE ASKS AND IS TOLD, which is `toggleStation`'s rule and the reason
// this takes a DIRECTION rather than toggling: the state it draws comes back
// from the Director through Publish, so a console that decided for itself could
// draw a bed the schedule does not have.
//
// AND IT ASKS NOTHING FROM OBSERVER. The bed is the STATION's, and a key that
// moved the programme from a surface where it is not drawn would be a control
// acting where the operator cannot see what they did.
func (r Router) cutBed() Router {
	if r.active != SurfaceBroadcaster || r.station == nil {
		return r
	}
	r.station.CutBed(!r.broadcaster.bed.Carrying)
	return r
}

// stepBed moves the selection through the relays the station's fence reaches.
//
// IT IS THE APP'S LIST, NOT THE CONSOLE'S. Which relays are in reach is a fact
// about the transmitter and the bed's fence (D-77), and a console that held its
// own copy would be a second answer to what the operator may choose.
// IT HANDS BACK A COMMAND (D-79). Landing on a relay TUNES it and tells the
// console what it landed on, and both reach the program — so run inline, from
// inside Update, they send to a loop that cannot receive and the app freezes.
func (r Router) stepBed(by int) (Router, tea.Cmd) {
	if r.active != SurfaceBroadcaster || r.relays == nil {
		return r, nil
	}
	return r, r.relays(by)
}

// stationIsLive is the ONE reader of the console's power, and it stayed one
// because a gate insisted.
//
// The swap precondition and the operator's toggle both need to know whether the
// station is on the air, and the second reader tripped the D-1 guard the moment
// it was written — "two carriers of one rule is the shape that produced the
// duck-lift bug". The guard was right and was not narrowed: the question got a
// name instead, and both askers go through it.
//
// IT READS THE DIRECTOR'S POWER, NEVER A UI FLAG (FR-5.1). And `Stopped` is not
// live, so from a stopped station the operator's control puts it ON the air —
// which is what a person pressing ON AIR means.
func (r Router) stationIsLive() bool { return r.broadcaster.power == lineup.Running }
