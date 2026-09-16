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

	// query is what the operator typed. WHAT IT RESOLVED TO LIVES IN `locate`
	// (D-130) — this held a `ref` of its own until 2026-09-15, and the field
	// outlived the refactor that replaced it: nothing assigned it, and
	// `blocker()` read it, so the chip named one condition for ever. A field
	// nobody writes is not dead weight, it is a wrong answer with a type.
	query string

	// locate is the DEBOUNCED answer about the Location field (D-130), and it
	// replaces the ref/outside pair this window kept for itself. The console's
	// `[l]` asks the same question of the same hook; two copies of the
	// bookkeeping would be two ideas of what "valid" means.
	locate locateState

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
	// THE BOTTOM OF THE RUNNING ORDER (HUM LEAD, 2026-09-14: "default to the
	// bottom - position 15"). A request that has to go SOMEWHERE goes where it
	// disturbs nothing — PRIORITIZE pushes every card down, and the disruptive
	// act should be the one chosen rather than the one arrived at.
	return requestState{
		chosen: report.Everything(),
		slot:   strconv.Itoa(lineup.MainTrackCap - 1),
	}
}

// requestValid reports whether the window can be scheduled.
//
// THREE THINGS MUST BE TRUE and each has its own helper text, so a refused
// `enter` never leaves the operator guessing which one.
func (st requestState) valid() bool {
	return st.locate.reachable() && !st.chosen.Empty() && st.positionOK()
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
//
// THE TYPED NUMBER IS A SLOT AND THIS RETURNS AN INDEX (D-119). It returned the
// slot verbatim, and `Requested.To` is documented as "the same number `Moved.To`
// carries" — the number the card window's move path has subtracted `liveOffset`
// from since D-119. So the two operator paths into one field disagreed by one
// on STANDBY, which is the console's normal state: the requested card landed a
// place lower than the slot the operator typed, silently, with the window's own
// confirmation naming the slot they asked for.
//
// THE OFFSET IS PASSED, NOT ASKED, because the arithmetic's owner is
// `Broadcaster.indexForSlot` and this surface cannot reach the console. The
// Router carries it across (D-156).
func (st requestState) position(liveOffset int) int {
	if st.prioritize {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(st.slot))
	if err != nil {
		return 0
	}
	// THE FRONT OF THE RUNNING ORDER IS THE FLOOR, and it is a TRIPWIRE: the
	// D-42 shape, stated for the same reason as `refence`'s. Today it cannot
	// fire — `valid()` gates this behind `positionOK`, which refuses anything
	// below `bcScheduledFrom` (2), and `liveOffset` is never more than 1, so the
	// lowest reachable index is 1. That is a rule held by a DIFFERENT rule, and
	// it vanishes silently the day the table starts drawing from slot 1 or the
	// window learns to type UP NEXT — which is exactly when `Insert` would be
	// handed a negative and refuse the operator's card with nothing to see.
	if n -= liveOffset; n < 0 {
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
	return st.locate.locateNote()
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
	return modalHelperWidth(o.Width)
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
	out = []string{""}

	// THE LOCATION, AND WHAT IS WRONG WITH IT.
	out = append(out, settingLabel("Location: ", st.field == requestLocation)+"["+render.PadTo(st.query, 22)+"]")
	if fact, aside := st.note(); fact != "" {
		// THE WORDING AND THE TINT ARE SHARED WITH THE CONSOLE'S LOOKUP (D-129).
		// Two windows ask the pool the same question, so poolnote.go answers it
		// once — including the reason the wrap has to come before the colour.
		out = append(out, poolNoteLines(o, fact, aside, requestHelperWidth(o))...)
	} else if st.locate.reachable() {
		out = append(out, "    "+st.locate.ref.Label)
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
	posAt := len(out)
	out = append(out, settingLabel("Position:", st.field == requestPosition))
	pri, slot := "( )", "( )"
	if st.prioritize {
		pri = "(" + o.Glyphs().OK + ")"
	} else {
		slot = "(" + o.Glyphs().OK + ")"
	}
	// PRIORITIZE IS BOLD AND YELLOW (HUM LEAD, 2026-09-14). It is the one choice
	// in this window that moves every other card, and the advisory tone is the
	// app's own word for "this one is different" — the same family `NameWarning`
	// belongs to, a step down in urgency.
	out = append(out, " "+pri+"  "+
		render.Bold(render.Tint("PRIORITIZE", render.Tok(render.NameAdvisory)))+
		" "+o.Glyphs().Dash+" Move to UP NEXT")
	out = append(out, "      Everything below moves down by 1", "")
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
		// FROM THE HEADING TO THE END, not a count backwards from it. The range
		// was `len(out)-4` and adding one blank line between the two options
		// moved it silently — an offset measured from the bottom is an offset
		// that every later edit has to remember.
		return out, posAt, len(out) - 1
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
	// THE QUESTION COULD NOT BE PUT, AND THAT IS NOT "CHOOSE ANOTHER" (D-157).
	// It fell to the case below and the chip read "Choose a location" — about a
	// location that may well be real and was never actually checked — while the
	// helper line under the same field read "The lookup did not answer; press
	// enter to try again". Two sentences in one window disagreeing about what
	// the operator should do, and the one on the key was the wrong one.
	//
	// IT OUTRANKS `!found` BECAUSE A TIMEOUT SETS IT. There is no verdict to
	// report, so every test below this asks about an answer that does not exist.
	case st.locate.onSubmit() == submitRetry:
		return "Try again"
	// NOT SETTLED IS NOT FOUND, and both mean the same thing to an operator:
	// this window cannot act on what is in the Location field yet. The check
	// reaches `locate` because that is where the answer is — the `ref` this
	// used to read was never written.
	case !st.locate.settled() || !st.locate.found:
		return "Choose a location"
	case !st.locate.within:
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
		return d.requestErase()
	}
	if r := key.String(); len(r) == 1 {
		return d.requestType(r)
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
func (d Dashboard) requestType(r string) (Dashboard, tea.Cmd) {
	switch d.request.field {
	case requestLocation:
		d.request.query += r
		// THE ANSWER IS STALE THE MOMENT THE QUERY CHANGES. Keeping the old ref
		// would let the window show one place and schedule another — `edit`
		// discards it and starts the pause again.
		return d.afterRequestEdit()
	case requestPosition:
		if r >= "0" && r <= "9" {
			d.request.slot += r
			d.request.prioritize = false // typing a slot IS choosing the slot
		}
	}
	return d, nil
}

// requestErase is backspace, into whichever field takes text.
func (d Dashboard) requestErase() (Dashboard, tea.Cmd) {
	switch d.request.field {
	case requestLocation:
		if n := len(d.request.query); n > 0 {
			d.request.query = d.request.query[:n-1]
		}
		return d.afterRequestEdit()
	case requestPosition:
		if n := len(d.request.slot); n > 0 {
			d.request.slot = d.request.slot[:n-1]
		}
	}
	return d, nil
}

// afterRequestEdit restarts the pause after a change to the Location field.
//
// THE SAME MECHANISM THE CONSOLE'S SEARCH BOX USES (D-130). It reaches the
// network for the small places the embedded index does not hold, so it waits
// for the operator to stop typing rather than asking on every key.
func (d Dashboard) afterRequestEdit() (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	d.request.locate, cmd = d.request.locate.edit(locateRequest, d.request.query)
	return d, cmd
}

// requestSchedule is [enter]: the form becomes a request, or says why it cannot.
//
// IT CLOSES ONLY WHEN THE SCHEDULE WAS TOLD. FR-3.3 — "an action must never be
// shown as taken unless the schedule took it" — and a window that closed on an
// invalid form would be exactly that: the operator would believe they had
// scheduled something.
func (d Dashboard) requestSchedule() (tea.Model, tea.Cmd) {
	if d.cfg.RequestCard == nil {
		return d, nil
	}
	// WHAT [ENTER] MEANS IS `onSubmit`'S TO SAY, AND BOTH WINDOWS NOW ASK IT
	// (D-157). This carried TWO of the four states and `modal_location.go`
	// carried all four, which is the same defect twice over:
	//
	// D-151, red team round 2 — an enter pressed INSIDE the 300 ms pause plus a
	// geocoder round trip returned `d, nil`, and the chip read "Choose a
	// location" about a location the operator had already typed.
	//
	// D-157, red team round 3 — an enter pressed when the lookup COULD NOT BE
	// ASKED did nothing at all, while `locateNote` — shared, and therefore
	// right — printed "The lookup did not answer; press enter to try again".
	// The window instructed an action the window refused. A timeout is not a
	// verdict about a place, and treating it as one leaves the operator with a
	// real location they cannot request and no way to retry.
	if strings.TrimSpace(d.request.query) != "" {
		switch d.request.locate.onSubmit() {
		case submitAsk, submitRetry:
			d.request.locate.submitted = true
			return d, d.locateCmd(locateRequest, d.request.locate.gate.Seq(), d.request.locate.query)
		}
	}
	if !d.request.valid() {
		return d, nil
	}
	ref, at, kinds := *d.request.locate.ref, d.request.position(d.liveOffset), d.request.chosen
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
