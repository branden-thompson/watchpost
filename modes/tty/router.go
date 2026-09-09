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
	actStationToggle   term.Action = "station-toggle"
)

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
		actStationToggle:   {Keys: []string{"shift+enter"}, Help: "ON AIR / STANDBY"},
	}
}

// Router holds the surfaces and delegates to the active one.
type Router struct {
	observer    Dashboard
	broadcaster Broadcaster
	active      Surface

	// keys are the console's own bindings, merged once and shared by value.
	keys term.KeyMap

	// refusal is why the last swap was refused, shown to the operator. A
	// refusal they cannot read is indistinguishable from a broken control.
	refusal string
}

// NewRouter wraps Observer. The second surface arrives in P1.
func NewRouter(o Dashboard) Router {
	// The console inherits the SAME --ascii decision Observer was built with.
	// Two surfaces disagreeing about whether the terminal can draw a glyph
	// would be one setting with two carriers, which is the shape this codebase
	// has removed twice.
	b := NewBroadcaster()
	b.ascii = o.cfg.ASCII
	return Router{observer: o, broadcaster: b, active: SurfaceObserver}
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
	case LineupMsg, StationMsg:
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
				return r.swapTo(SurfaceObserver), nil
			case actSwapBroadcaster:
				return r.swapTo(SurfaceBroadcaster), nil
			}
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
func (r Router) View() tea.View { return r.surface().View() }

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
func (r Router) swapTo(to Surface) Router {
	ok, why := r.canSwap(to)
	if !ok {
		r.refusal = why
		return r
	}
	r.refusal = ""
	r.active = to
	return r
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
	if r.broadcaster.power == lineup.Running {
		return false, "the station is ON AIR — go to STANDBY before leaving the console"
	}
	return true, ""
}
