package tty

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// propAckVersion is the acknowledgement's wording, as recorded once seen
// (A-9): a changed wording raises it, and it is shown once more.
const propAckVersion = 1

// propAckWidth is the acknowledgement's width: within the 80-column floor.
const propAckWidth = 64

// propAckText is the acknowledgement's words (D-46, D-81, A-34), a
// paragraph each. Its first line is where the terminal cursor stands.
var propAckText = []string{
	"Watchpost does not transmit. The MUF and foF2 it draws are provided as reference.",
	"Anyone who transmits on HF is responsible for following all applicable laws where they are.",
	"To draw them, watchpost asks GIRO (the Lowell Global Ionospheric Radio Observatory) and NOAA's Space Weather Prediction Center, which learn this computer's IP address. MAP STATUS lists them.",
	"Enter or Esc closes this. It is shown again only if these words change.",
}

// propAckLines are the acknowledgement's lines, its paragraphs apart,
// wrapped inside the window's margins as the app's other asides are.
func (d Dashboard) propAckLines(o render.Opts) []string {
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
func (d Dashboard) propAckDue() bool { return ackDue(d.cfg.PropagationAck, propAckVersion) }

// ackDue reports whether words seen at one version are to be shown at the
// current one: when the version seen is lower, 0 being never seen.
func ackDue(seen, current int) bool { return seen < current }

// openPropAck shows the acknowledgement over the map, before anything of the
// Propagation mode's is fetched (D-81, FR-4.2).
func (d Dashboard) openPropAck() Dashboard { return d.open(modalPropAck) }

// handlePropAckKey is the acknowledgement's keyboard: Enter or Esc closes it
// and records it seen (D-46, D-81); the Observer's scroll keys read it at
// the floor; every other key does nothing.
func (d Dashboard) handlePropAckKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter", "esc":
	default:
		if act, ok := d.keys.Lookup(key.String()); ok {
			return d.handleModalNav(act), nil // scrolling only: it is no way out
		}
		return d, nil
	}
	d = d.close()
	d.cfg.PropagationAck = propAckVersion
	save := d.cfg.SavePropagationAck
	if save == nil {
		return d, nil
	}
	return d, func() tea.Msg { _ = save(propAckVersion); return nil } // a failed save shows it again next session, never an error (D-124)
}

// propAckCursor is where the terminal cursor stands on the acknowledgement,
// D-81: the start of its first line, found in the frame as drawn; false when
// it is not on screen.
func propAckCursor(frame string) (x, y int, ok bool) {
	first := propAckText[0][:12]
	for row, line := range strings.Split(frame, "\n") { // bounded by the frame (P10-02)
		if at := strings.Index(line, first); at >= 0 {
			return render.Width(line[:at]), row, true // the cells before it, its colours not counted
		}
	}
	return 0, 0, false
}
