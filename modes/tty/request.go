package tty

// request.go — the Line-Up Request window, and the state behind it (R4).
//
// HUM LEAD, 2026-09-14, specifying it: a Location from the pool, the reports
// available versus the reports chosen, and a requested position — PRIORITIZE to
// UP NEXT, or a line-up slot the rest is pushed down from.
//
// THE REPORT ROWS ARE DERIVED FROM THE REGISTRY, NOT LISTED HERE, and that is
// the ruling this window exists to honour: "we WILL have more report types in
// the future, and ensuring Broadcaster is flexible enough for that list to grow
// … WITHOUT having to completely re-architect the code and flow every time is
// critical." A fifth kind appears in this window because `report.All()` grew,
// and no line below mentions a kind by name.

import (
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// requestField is where the keyboard is inside the window.
//
// THE REPORT ROWS ARE ONE FIELD WITH AN INDEX, not one field per kind: a field
// per kind is a second list to keep in step with the registry, which is the
// coupling this whole batch is avoiding.
type requestField int

const (
	requestLocation requestField = iota
	requestReports
	requestPosition
	numRequestFields
)

// requestState is the window's own model.
type requestState struct {
	field requestField

	// query is what the operator typed, and ref is what it resolved to. A
	// resolved ref with a query that has since changed is stale by
	// construction: `resolve` clears it.
	query string
	ref   *snapshot.LocationRef
	// outside says the location was found but is NOT in the station's pool —
	// the case that gets helper text rather than a refusal (ruling 2).
	outside bool

	// at is which report row the pointer is on; chosen is the set.
	at     int
	chosen report.Set

	// prioritize is PRIORITIZE rather than a slot; slot is what was typed when
	// it is not.
	prioritize bool
	slot       string
}

// requestRows is the report rows, in registry order.
//
// ASKED OF THE REGISTRY EVERY TIME. A cached slice here would be the second
// list this design exists to avoid, and the cost is a loop over four things.
func requestRows() []report.Kind { return report.All() }

// requestOpen starts a fresh window.
//
// PRIORITIZE IS NOT THE DEFAULT. The operator came here to schedule something;
// putting it at the front unless they say otherwise would make the safe act the
// one they have to remember to choose.
func requestOpen() requestState {
	return requestState{chosen: report.Everything()}
}

// requestValid reports whether the window can be scheduled.
//
// THREE THINGS MUST BE TRUE and each has its own helper text, so a refused
// `enter` never leaves the operator guessing which one.
func (st requestState) valid() bool {
	return st.ref != nil && !st.outside && !st.chosen.Empty() && st.positionOK()
}

// positionOK reports whether the requested position is one the running order
// has. PRIORITIZE always is.
func (st requestState) positionOK() bool {
	if st.prioritize {
		return true
	}
	n, err := strconv.Atoi(strings.TrimSpace(st.slot))
	return err == nil && n >= bcScheduledFrom && n < lineup.MainTrackCap
}

// position is the running-order index the request asks for.
//
// PRIORITIZE IS ZERO — the front of the running order, which is UP NEXT and
// never LIVE (ruling 3): the card on the air is not in the running order at
// all, so there is no index that could name it.
func (st requestState) position() int {
	if st.prioritize {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(st.slot))
	if err != nil {
		return 0
	}
	return n
}

// requestNote is the helper text under the location field, "" when there is
// nothing to say.
//
// THE SHAPE OBSERVER'S LOOKUP USES (ruling 2): the fact, then what the operator
// can do about it. A location outside the service radius is NOT an error — it is
// a real place the station cannot broadcast about, and Observer can show it.
func (st requestState) note() (string, string) {
	switch {
	case strings.TrimSpace(st.query) == "":
		return "", ""
	case st.ref == nil:
		return "Location not found in Pool.", "Observer supports location lookup outside Broadcast Radius"
	case st.outside:
		return "Outside the station's service radius.", "Observer supports location lookup outside Broadcast Radius"
	}
	return "", ""
}

// requestTitle is the window's own name.
const requestTitle = "Line-Up Request"

// requestChips is the window's pinned footer.
//
// PINNED, NOT SCROLLED (OP-5): "enter Schedule / esc Cancel" is what a lost
// operator needs most, and at 80x24 a footer in the body would be the first
// thing to scroll away.
func (d Dashboard) requestChips(o render.Opts) []string {
	sched := o.KeyCap("enter") + "  Schedule"
	if !d.request.valid() {
		// THE CHIP SAYS WHAT IS MISSING rather than going quiet. A control that
		// simply stops working reads as a broken window; one that says why reads
		// as a form.
		sched = o.KeyCap("enter") + "  " + d.request.blocker()
	}
	return []string{"", "  " + sched + "    " + o.KeyCap("esc") + "  Cancel"}
}

// requestBody draws the window, and says which lines the focus is on.
//
// A FORM, SO IT SCROLLS BY ITS FOCUS — the same family as Settings, the
// diagnostics window and the relay fault (`focusBody`). The reachability gate is
// what said so: at 80x24 seven of its lines could not be brought on screen, and
// a line the keyboard cannot reach is not in the window.
func (d Dashboard) requestBody(o render.Opts) (out []string, focusAt, focusEnd int) {
	st := d.request
	// EVERY CONTENT LINE CLEARS THE INSET ON BOTH SIDES, which the margin gate
	// measures and which a hand-indented window gets wrong in one direction or
	// the other. `pad` is the one place the window's own indent is decided, so
	// there is nothing to keep in step.
	// `inset` is applied to every line at the END, in one place.
	mark := func(f requestField) int {
		if st.field == f {
			return len(out)
		}
		return -1
	}
	_ = mark
	out = []string{""}

	// THE LOCATION, AND WHAT IS WRONG WITH IT.
	out = append(out, settingLabel("Location: ", st.field == requestLocation)+"["+render.PadTo(st.query, 22)+"]")
	if fact, aside := st.note(); fact != "" {
		// THE HOUSE PATTERN, WHICH IS THE SETUP FORM'S: the alert glyph and the
		// fact, then what the operator can do instead. The HUM LEAD asked for
		// the aside in yellow italics and called the wording "not hard and
		// fast" — COLOUR IS THEIR OWN PASS, so this draws the shape and leaves
		// the tone to it rather than inventing a token.
		out = append(out, "  "+o.Glyphs().Alert+" "+fact)
		out = append(out, "    "+aside)
	} else if st.ref != nil {
		out = append(out, "    "+st.ref.Label)
	} else {
		out = append(out, "")
	}
	out = append(out, "")

	// THE REPORTS, FROM THE REGISTRY. Both forms, because the operator is
	// CHOOSING here (ruling 1) — the label alone is not enough to choose from.
	out = append(out, settingLabel("Reports:", st.field == requestReports))
	for i, k := range requestRows() { // bounded by the registry (P10-02)
		mark := " "
		if st.field == requestReports && i == st.at {
			mark = o.Glyphs().Pointer
		}
		box := "[ ]"
		if st.chosen.Has(k) {
			box = "[" + o.Glyphs().OK + "]"
		}
		spec := report.Of(k)
		// THE LABEL FIRST, THEN THE NAME. Both are shown because the operator is
		// CHOOSING here (ruling 1) — and this order is what fits: the longest
		// full name is 32 cells and the window is 56, so a name-then-label row
		// wrapped and the margin gate caught it.
		out = append(out, " "+mark+" "+box+" "+render.PadTo(spec.Label, 7)+spec.FullName)
	}
	out = append(out, "", "  Scheduled as: "+st.chosen.Describe(), "")

	// THE POSITION.
	out = append(out, settingLabel("Position:", st.field == requestPosition))
	pri, slot := "( )", "( )"
	if st.prioritize {
		pri = "(" + o.Glyphs().OK + ")"
	} else {
		slot = "(" + o.Glyphs().OK + ")"
	}
	out = append(out, " "+pri+"  PRIORITIZE "+o.Glyphs().Dash+" Move to UP NEXT")
	out = append(out, "      Everything below moves down by 1")
	out = append(out, " "+slot+"  Line-Up Slot: ["+render.PadTo(st.slot, 3)+"]")
	out = append(out, "      Cards below it move down by 1")

	// EVERY CONTENT LINE CLEARS THE INSET ON BOTH SIDES (the margin gate). Done
	// once, here, rather than at twenty string literals — a window indented by
	// hand clears one side and not the other, which is exactly what the gate
	// reported on the first draft of this one.
	for i, l := range out { // bounded by the window (P10-02)
		if strings.TrimSpace(l) == "" {
			continue // a blank spacer has no content to inset
		}
		out[i] = strings.Repeat(" ", modalInset) + l + strings.Repeat(" ", modalInset)
	}

	// THE FOCUSED RANGE, so the scroll can keep it on screen. One field at a
	// time, and the reports are a range because the pointer moves inside them.
	switch st.field {
	case requestLocation:
		return out, 1, 3
	case requestReports:
		return out, 5, 5 + len(requestRows()) + 2
	case requestPosition:
		return out, len(out) - 4, len(out) - 1
	}
	return out, 0, 0
}

// blocker names the one thing stopping a schedule, in the order the operator
// filled the form in.
//
// ONE REASON AT A TIME. Listing all three would be a wall of text on a control
// that has room for a phrase, and the first unmet condition is the one they are
// working on.
func (st requestState) blocker() string {
	switch {
	case st.ref == nil:
		return "Choose a location"
	case st.outside:
		return "Location is outside the service radius"
	case st.chosen.Empty():
		return "Choose at least one report"
	case !st.positionOK():
		return "Choose a position"
	}
	return "Schedule"
}

// handleRequestNav walks the window: down goes THROUGH the report rows rather
// than over them.
//
// ONE LIST, NOT THREE. The operator presses down and arrives at the next thing
// they can change, whatever kind of thing it is — a field, a report row, a
// position choice. The reachability gate is why it is one list: it presses down
// and requires every line to come on screen, and a walk that skipped the report
// rows left four of them unreachable at 80x24.
func (d Dashboard) handleRequestNav(act term.Action) Dashboard {
	switch act {
	case "nav-down":
		d.request = d.request.next()
	case "nav-up":
		d.request = d.request.prev()
	}
	return d
}

// next is one step down the window.
func (st requestState) next() requestState {
	switch {
	case st.field == requestReports && st.at < len(requestRows())-1:
		st.at++
	case st.field < numRequestFields-1:
		st.field++
		st.at = 0
	default:
		// IT WRAPS, like every other list in this app: the operator who holds
		// down arrives back at the top rather than at a dead key.
		st.field, st.at = requestLocation, 0
	}
	return st
}

// prev is one step up.
func (st requestState) prev() requestState {
	switch {
	case st.field == requestReports && st.at > 0:
		st.at--
	case st.field > requestLocation:
		st.field--
		if st.field == requestReports {
			st.at = len(requestRows()) - 1
		}
	default:
		st.field = numRequestFields - 1
	}
	return st
}
