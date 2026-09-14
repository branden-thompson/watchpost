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
	tea "charm.land/bubbletea/v2"

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

// requestHelperWidth is how much room a helper line has before the window wraps
// it: the panel's content, less this window's own inset and the helper's lead.
//
// COMPUTED, NOT GUESSED. A constant here would be right at one modal width and
// wrong at every other, and the whole reason the helper lost its colour is that
// something wrapped where nobody expected it to.
func requestHelperWidth(o render.Opts) int {
	return max(12, o.Width-4-2*modalInset-4)
}

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
		// THE TONE OBSERVER ALREADY USES FOR THIS EXACT MEANING (HUM LEAD, UAT
		// 2026-09-14): "similar to the red used by the 'this is not your local
		// station' color in Observer". That line is
		// `Italic(Tint(…, NameWarning))` in detail.go, and it says the same kind
		// of thing — what you are looking at is not what you think it is.
		//
		// THE TOKEN, NOT A COLOUR. Borrowing Observer's own means the two cannot
		// drift and the theme moves both together; a new token here would be a
		// second answer to "what does a caveat look like".
		// WRAPPED HERE, AND EACH LINE TINTED SEPARATELY.
		//
		// THE WINDOW WRAPS WHAT IT IS GIVEN, and a tint applied to the whole
		// string is a pair of escape codes at its two ENDS — so the wrap put
		// "Broadcast Radius" on a second line with no colour on it at all, and
		// the caveat trailed off into plain grey mid-sentence (HUM LEAD, UAT
		// 2026-09-14, with the screenshot).
		//
		// STYLING SURVIVES A WRAP ONLY IF EVERY LINE CARRIES IT, so the text is
		// broken up first and each piece is tinted on its own.
		tone := render.Tok(render.NameWarning)
		for _, l := range render.WrapText(fact, requestHelperWidth(o)) { // bounded by the text (P10-02)
			out = append(out, "  "+o.Glyphs().Alert+" "+render.Tint(l, tone))
		}
		for _, l := range render.WrapText(aside, requestHelperWidth(o)) { // bounded by the text (P10-02)
			out = append(out, "    "+render.Italic(render.Tint(l, tone)))
		}
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
//
// EACH TRANSITION NAMES ITS DESTINATION. `field++` produced every state and
// CONSTRUCTED none of them — nothing in production ever said the words
// `requestReports` or `requestPosition`, so `wires` reported both as having no
// writer, and it was right: a state reachable only by arithmetic is one no
// reader can grep for. It also reads better, which is the usual way round.
func (st requestState) next() requestState {
	switch st.field {
	case requestLocation:
		st.field, st.at = requestReports, 0
	case requestReports:
		if st.at < len(requestRows())-1 {
			st.at++ // down walks THROUGH the rows, not over them
			return st
		}
		st.field, st.at = requestPosition, 0
	case requestPosition:
		// IT WRAPS, like every other list in this app: the operator who holds
		// down arrives back at the top rather than at a dead key.
		st.field, st.at = requestLocation, 0
	}
	return st
}

// prev is one step up.
func (st requestState) prev() requestState {
	switch st.field {
	case requestLocation:
		st.field, st.at = requestPosition, 0
	case requestReports:
		if st.at > 0 {
			st.at--
			return st
		}
		st.field, st.at = requestLocation, 0
	case requestPosition:
		st.field, st.at = requestReports, len(requestRows())-1
	}
	return st
}

// handleRequestKey owns the keyboard while the request window is open.
//
// A WINDOW ON TOP OWNS THE KEYS (D-58), and a form owns them twice over: a
// digit typed into the slot field must not reach the running order, and a
// letter typed into the location must not be a console control.
//
// ESC CANCELS AND LOSES THE FORM, deliberately. A half-filled request kept
// across a close is a window that reopens saying something the operator did not
// mean to still be asking for.
func (d Dashboard) handleRequestKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		d.request = requestState{}
		return d.close(), nil
	case "enter":
		return d.requestSchedule()
	case "up":
		return d.handleRequestNav("nav-up"), nil
	case "down":
		return d.handleRequestNav("nav-down"), nil
	case "tab":
		// TAB IS THE FIELD, ↓ IS THE LINE. One walks the form, the other walks
		// whatever the focused field contains — the same split Settings uses.
		// TAB IS `next` WITHOUT THE ROWS: it moves to the next FIELD, where ↓
		// walks whatever the focused field contains. One order, not two.
		d.request.field, d.request.at = d.request.next().field, 0
		if d.request.field == requestReports && d.request.at != 0 {
			d.request.at = 0
		}
		return d, nil
	case " ":
		return d.requestToggle(), nil
	case "backspace":
		return d.requestErase(), nil
	}
	if r := key.String(); len(r) == 1 {
		return d.requestType(r), nil
	}
	return d, nil
}

// requestToggle is what [space] does, which depends on where the focus is.
func (d Dashboard) requestToggle() Dashboard {
	switch d.request.field {
	case requestReports:
		rows := requestRows()
		if d.request.at < len(rows) {
			d.request.chosen = d.request.chosen.Toggle(rows[d.request.at])
		}
	case requestPosition:
		// ONE CHOICE OF TWO, so space flips between them rather than setting
		// one — there is no state where neither is chosen.
		d.request.prioritize = !d.request.prioritize
	}
	return d
}

// requestType is a printable key, into whichever field takes text.
func (d Dashboard) requestType(r string) Dashboard {
	switch d.request.field {
	case requestLocation:
		d.request.query += r
		// THE RESOLUTION IS STALE THE MOMENT THE QUERY CHANGES. Keeping the old
		// ref would let the window show one place and schedule another.
		d.request.ref, d.request.outside = nil, false
		return d.requestResolve()
	case requestPosition:
		if r >= "0" && r <= "9" {
			d.request.slot += r
			d.request.prioritize = false // typing a slot IS choosing the slot
		}
	}
	return d
}

// requestErase is backspace, into whichever field takes text.
func (d Dashboard) requestErase() Dashboard {
	switch d.request.field {
	case requestLocation:
		if n := len(d.request.query); n > 0 {
			d.request.query = d.request.query[:n-1]
		}
		d.request.ref, d.request.outside = nil, false
		return d.requestResolve()
	case requestPosition:
		if n := len(d.request.slot); n > 0 {
			d.request.slot = d.request.slot[:n-1]
		}
	}
	return d
}

// requestResolve asks the console's pool what the operator typed means.
//
// THE POOL IS THE ANSWER, NOT THE GEOCODER (ruling 2). A location the station
// cannot broadcast about is not an error and not a lookup failure — it is a real
// place outside the service radius, and the window says so and points at
// Observer rather than refusing to understand.
func (d Dashboard) requestResolve() Dashboard {
	q := strings.TrimSpace(strings.ToLower(d.request.query))
	if q == "" || d.cfg.PoolLookup == nil {
		return d
	}
	ref, inPool, found := d.cfg.PoolLookup(q)
	switch {
	case !found:
		d.request.ref, d.request.outside = nil, false
	case !inPool:
		r := ref
		d.request.ref, d.request.outside = &r, true
	default:
		r := ref
		d.request.ref, d.request.outside = &r, false
	}
	return d
}

// requestSchedule is [enter]: the form becomes a request, or says why it cannot.
//
// IT CLOSES ONLY WHEN THE SCHEDULE WAS TOLD. FR-3.3 — "an action must never be
// shown as taken unless the schedule took it" — and a window that closed on an
// invalid form would be exactly that: the operator would believe they had
// scheduled something.
func (d Dashboard) requestSchedule() (tea.Model, tea.Cmd) {
	if !d.request.valid() || d.cfg.RequestCard == nil {
		return d, nil
	}
	ref, at, kinds := *d.request.ref, d.request.position(), d.request.chosen
	d.request = requestState{}
	d = d.close()
	return d, func() tea.Msg {
		d.cfg.RequestCard(ref, kinds, at)
		return nil
	}
}

// openRequest opens a fresh Line-Up Request window.
func (d Dashboard) openRequest() Dashboard {
	d.request = requestOpen()
	return d.open(modalRequest)
}
