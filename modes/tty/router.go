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

import tea "charm.land/bubbletea/v2"

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

// Router holds the surfaces and delegates to the active one.
type Router struct {
	observer Dashboard
	active   Surface
}

// NewRouter wraps Observer. The second surface arrives in P1.
func NewRouter(o Dashboard) Router { return Router{observer: o, active: SurfaceObserver} }

// Init delegates to the active surface. Observer asks for the terminal's
// background colour here, and dropping that would leave the frame painting
// against the wrong ground.
func (r Router) Init() tea.Cmd { return r.surface().Init() }

// Update routes the message and keeps the Router as the program's model.
//
// EVERY MESSAGE GOES TO THE ACTIVE SURFACE while there is only one. The
// fan-out the program-scoped messages need — window size and background
// colour reaching BOTH surfaces — arrives with the second surface in P1,
// because fanning to one surface is the same as delegating to it and a
// fan-out nothing can observe is a claim rather than a mechanism.
//
// F-67 IS NOT CLOSED BY THIS BATCH, and saying so is the disposition. The
// five program-scoped messages with no production handler — focus, blur,
// suspend, resume and colour-profile — are forwarded exactly as every other
// message is, so they are neither more nor less handled than before P0. They
// become actionable when a second surface can be inactive and stale, which is
// P1. What P0 changes is that there is now ONE place to handle them.
func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch r.active {
	case SurfaceObserver:
		m, cmd := r.observer.Update(msg)
		if d, ok := m.(Dashboard); ok {
			r.observer = d
		}
		return r, cmd
	}
	return r, nil
}

// View renders the active surface, unchanged.
func (r Router) View() tea.View { return r.surface().View() }

// surface is the active model. It exists so Init and View cannot disagree
// about which surface is on screen — one accessor, not two switches.
func (r Router) surface() tea.Model { return r.observer }

// canSwap reports whether the operator may leave the given surface.
//
// FAILS CLOSED, DELIBERATELY (PLAN red team). The precondition it enforces —
// the station is in STANDBY — reads a Power that is not wired until P2, and a
// second surface first exists in P1. For that window the honest answer is no:
// a refused swap is an inconvenience, and a permitted one during a live read
// is the hazard D-1 exists to prevent.
func (r Router) canSwap(to Surface) (bool, string) {
	if to < 0 || to >= numSurfaces {
		return false, "that surface does not exist"
	}
	return false, "switching arrives with the Broadcaster surface"
}
