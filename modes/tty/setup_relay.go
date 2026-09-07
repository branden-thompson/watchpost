package tty

// setup_relay.go — the WATCHPOST RADIO - RELAY REPLAY group.
//
// A GROUP OF ITS OWN, not a sixth row under CORRESPONDENTS (HUM LEAD, UAT
// 2026-09-04). The rows above it are about WHO READS; this is about PACING, and
// giving it a heading leaves somewhere for the next pacing setting to land
// rather than growing the correspondents list with something that is not a
// correspondent.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// relayDwell is one rotation interval the picker offers.
type relayDwell struct {
	d     time.Duration
	label string
}

// relayDwells are the choices, shortest first (HUM LEAD: 30s / 45s / 1m /
// 1m 30s / 2m / 3m / 5m).
//
// THIRTY SECONDS IS THE FLOOR BECAUSE IT IS THE SHORTEST USEFUL TURN, not
// because anything breaks below it: a relay cycle runs minutes, so a shorter
// rotation would cut every station mid-sentence and never let one finish.
// Five minutes is the top because it is one NWR cycle — the length the rotation
// was built around (UAT 93) — and it stays the default.
func relayDwells() []relayDwell {
	return []relayDwell{
		{30 * time.Second, "30s"},
		{45 * time.Second, "45s"},
		{time.Minute, "1m"},
		{90 * time.Second, "1m 30s"},
		{2 * time.Minute, "2m"},
		{3 * time.Minute, "3m"},
		{5 * time.Minute, "5m"},
	}
}

// defaultRelayDwell is the entry a listener who has set nothing lands on.
// relayLabelW is the group's own label column, and relayCellW its own picker
// cell. Both rows use both, so the chips line up down the group the way they do
// in every other group — the shared rowControlW is 21 and the rotation's label
// is 32, so padding to the shared column aligns nothing and the two pickers sat
// three cells apart.
func relayLabelW() int {
	w := 0
	for _, l := range []string{relayDwellLabelText, relayLangLabelText} { // bounded by the labels (P10-02)
		w = max(w, len(l))
	}
	return w + 1
}

const (
	relayDwellLabelText = "Repeat: Watchlist - Rotate every"
	relayLangLabelText  = "Preferred language"
	relayCellW          = 7 // "English", the longest value either picker holds
)

// relayRowH is the lines one relay row occupies (its control and its support
// line), and relayLineOf where a row begins within the group.
const relayRowH = 2

func relayLineOf(focus setupRowID) int {
	if focus == rowRelayLang {
		return relayRowH + 1 // past the rotation row and the blank under it
	}
	return 0
}

// relayLangs are the languages NWR broadcasts in. Two, because the table
// carries two — this is a list of what exists, not a list of what is wanted.
func relayLangs() []relayLang {
	return []relayLang{{"en", "English"}, {"es", "Spanish"}}
}

type relayLang struct {
	code  string
	label string
}

func defaultRelayLang() string { return "en" }

func relayLangLabel(code string) string {
	for _, l := range relayLangs() { // bounded by the list (P10-02)
		if l.code == code {
			return l.label
		}
	}
	return relayLangs()[0].label
}

// cycleRelayLang moves the picker one entry, wrapping at both ends.
func cycleRelayLang(cur string, forward bool) string {
	list := relayLangs()
	at := 0
	for i, l := range list { // bounded by the list (P10-02)
		if l.code == cur {
			at = i
			break
		}
	}
	step := 1
	if !forward {
		step = -1
	}
	return list[((at+step)%len(list)+len(list))%len(list)].code
}

func defaultRelayDwell() time.Duration { return 5 * time.Minute }

// relayDwellLabel names a duration, falling back to the default's label for
// anything the list does not carry — a config written by hand, or a value from
// a later build with more choices.
func relayDwellLabel(d time.Duration) string {
	// ONE PASS, TWO ANSWERS. This used to fall back by calling itself with the
	// default — which reads fine and is a stack overflow the day the default
	// leaves the list, because nothing in the types says it is in there. The
	// fallback label is found in the same walk instead, and an empty one is
	// impossible rather than merely unlikely.
	fallback, def := "", defaultRelayDwell()
	for _, c := range relayDwells() { // bounded by the list (P10-02)
		if c.d == d {
			return c.label
		}
		if c.d == def {
			fallback = c.label
		}
	}
	if fallback == "" { // INVARIANT: the default is one of the offered entries
		return def.String()
	}
	return fallback
}

// cycleRelayDwell moves the picker one entry, wrapping at both ends like every
// other picker in the window.
func cycleRelayDwell(cur time.Duration, forward bool) time.Duration {
	list := relayDwells()
	at := 0
	for i, c := range list { // bounded by the list (P10-02)
		if c.d == cur {
			at = i
			break
		}
	}
	step := 1
	if !forward {
		step = -1
	}
	return list[((at+step)%len(list)+len(list))%len(list)].d
}

// relayLines draws the group: the picker row and its support line.
//
// The support line names the ENDS of the range rather than listing every
// choice — the picker already shows the choices as you move through it, and a
// support line that repeats them is a second copy of the list that can disagree
// with the first.
func (d Dashboard) relayLines(o render.Opts) []string {
	focused := d.setup.focus == rowRelayDwell
	chips := newArrowChips(o)
	row := castRowIndent + setupMark(o, focused) +
		settingLabel(render.PadTo(relayDwellLabelText, relayLabelW()), focused) +
		pickerCellW(relayDwellLabel(d.setup.relayDwell), chips, d.pickerFlashFor(rowRelayDwell), relayCellW)
	langFocused := d.setup.focus == rowRelayLang
	langRow := castRowIndent + setupMark(o, langFocused) +
		settingLabel(render.PadTo(relayLangLabelText, relayLabelW()), langFocused) +
		pickerCellW(relayLangLabel(d.setup.relayLang), chips, d.pickerFlashFor(rowRelayLang), relayCellW)
	return []string{
		row,
		castNoteIndent + "Shortest: 30 seconds / Longest: 5 minutes",
		"",
		langRow,
		castNoteIndent + "Used when two relays share a transmitter site.",
	}
}
