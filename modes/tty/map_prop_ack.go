package tty

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// propAckVersion is the acknowledgement's wording, as recorded once seen
// (A-9): a changed wording raises it, and it is shown once more.
const propAckVersion = 3

// propAckWidth is the acknowledgement's width: within the 80-column floor.
const propAckWidth = 64

// propAckText is the acknowledgement's words (D-46, D-81, A-34), a
// paragraph each. Its first line is where the terminal cursor stands.
var propAckText = []string{
	"Watchpost does not transmit. The MUF and foF2 it draws are provided as reference.",
	"Anyone who transmits on HF is responsible for following all applicable laws where they are.",
	"To draw them, watchpost asks GIRO (the Global Ionospheric Radio Observatory) and NOAA's Space Weather Prediction Center, which learn this computer's IP address. The Status window (S) lists them.",
	"Enter or Esc closes this. It is shown again only if these words change.",
}

// propAckLines are the acknowledgement's lines, its paragraphs apart,
// wrapped inside the window's margins as the app's other asides are.
func (mm mapModel) propAckLines(o render.Opts) []string {
	var out []string
	for i, p := range propAckText {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, p)
	}
	return insetModalLines(out, debugProseWidth(o, propAckWidth))
}

// propAckDue reports whether the acknowledgement is to be shown: never seen,
// or seen at a lower version than its words now carry (A-9).
func (mm mapModel) propAckDue() bool { return ackDue(mm.ackSeen, propAckVersion) }

// ackDue reports whether words seen at one version are to be shown at the
// current one: when the version seen is lower, 0 being never seen.
func ackDue(seen, current int) bool { return seen < current }

// openPropAck shows the acknowledgement over the map, before anything of the
// Propagation mode's is fetched (D-81, FR-4.2).
func (mm mapModel) openPropAck() mapWant { return mapWant{Open: mapModalPropAck} }

// onPropAckKey is the acknowledgement's keyboard: Enter or Esc closes it
// and records it seen (D-46, D-81); the Observer's scroll keys read it at
// the floor; every other key does nothing.
func (mm mapModel) onPropAckKey(key tea.KeyPressMsg, e mapEnv) (mapModel, tea.Cmd, mapWant) {
	switch key.String() {
	case "enter", "esc":
	default:
		if act, ok := e.Keys.Lookup(key.String()); ok {
			return mm, nil, mapWant{ModalNav: act} // scrolling only: it is no way out
		}
		return mm, nil, mapWant{}
	}
	mm.ackSeen = propAckVersion
	return mm, nil, mapWant{Close: true, Then: mapStepAckAsk} // closed, then the mode's first update (D-81)
}

// ackAsked is the acknowledgement seen and closed: the mode's first update
// asked, and the wording kept seen (0.19.6 design MS-1's S4).
func (mm mapModel) ackAsked(e mapEnv) (mapModel, tea.Cmd) {
	mm, update := mm.askPropagation(e) // seen: the mode's first update (D-81)
	save := mm.mapHooks.SavePropagationAck
	if save == nil {
		return mm, update
	}
	return mm, tea.Batch(update, func() tea.Msg { _ = save(propAckVersion); return nil }) // a failed save shows it again next session, never an error (D-124)
}
