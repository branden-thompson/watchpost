package tty

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func faultDash(t *testing.T) Dashboard {
	t.Helper()
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	return d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
		{Label: "<Next Best Relay>", Key: "a"},
		{Label: "<2nd best Relay>", Key: "b"},
	}})
}

// THE WINDOW IS THE MOCK.
//
// Compared against the HUM LEAD's mock stored as a fixture, not against a
// golden this code produced: a golden records what was built, and the question
// here is whether what was built is what was asked for. The only departure is
// the esc chip, which uses the app-wide KeyCap so this window is not the one
// place the control looks different (HUM LEAD: "use the standard chip controls
// for the [esc] - I just cant effectively mock that via text-based mocks").
func TestTheRelayFaultWindowIsTheMock(t *testing.T) {
	want, err := os.ReadFile("testdata/relay-fault.mock")
	if err != nil {
		t.Fatal(err)
	}
	got := stripANSITest(faultDash(t).renderModal(faultDash(t).opts()))
	wantLines := strings.Split(strings.TrimRight(string(want), "\n"), "\n")
	gotLines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(wantLines) != len(gotLines) {
		t.Fatalf("the window is %d lines, the mock is %d:\n%s", len(gotLines), len(wantLines), got)
	}
	for i := range wantLines {
		if wantLines[i] != gotLines[i] {
			t.Errorf("line %d\n want |%s|\n  got |%s|", i, wantLines[i], gotLines[i])
		}
	}
}

// THE FALL-THROUGH IS ALWAYS OFFERED, even when no relay is left. It is what
// the countdown takes, so a window that could omit it would have a default it
// never showed.
func TestTheFallThroughIsAlwaysOffered(t *testing.T) {
	// EVERY candidate count, not just the empty one. The first version of this
	// checked only "no relays left" and a mutant that dropped the fall-through
	// whenever a relay remained SURVIVED it — the case the window actually
	// meets, every time, went unmeasured (D-11).
	for n := 0; n <= 3; n++ {
		var cands []RelayCandidate
		for i := 0; i < n; i++ {
			cands = append(cands, RelayCandidate{Label: "relay", Key: "k"})
		}
		d := dash(t).(Dashboard)
		d.width, d.height = 133, 44
		d = d.openRelayFault(RelaySilentMsg{Candidates: cands})
		rows := d.relayFaultRows()
		if len(rows) == 0 || rows[len(rows)-1].label != "Fall-Thru" {
			t.Errorf("with %d candidates the fall-through must be the last row, got %v", n, rows)
			continue
		}
		if rows[len(rows)-1].key != "" {
			t.Errorf("the fall-through tunes to nothing; it reads the report")
		}
		// At most two relays are offered, per the mock's two rows.
		if want := min(n, 2) + 1; len(rows) != want {
			t.Errorf("with %d candidates the window offers %d rows, want %d", n, len(rows), want)
		}
	}
}

// TestTheCountdownRunsAndActs — the CLOCK AND ITS WIRE, driven through Update.
//
// RED TEAM 2026-09-05: onTick's whole body could be replaced with
// `return d.applyTick(), nil` — deleting the countdown step AND the auto-close —
// and this package stayed green. The arithmetic had a test that called
// stepRelayFault directly and the fall-through had one that called
// fallThroughRelayFault directly, so between them they pinned everything except
// that anything ever calls either. That is the same D-12 shape three UAT rounds
// already found in this window, one layer further out.
//
// Those two tests are folded in here: what they asserted is asserted below,
// through tickMsg, against the model's own clock.
func TestTheCountdownRunsAndActs(t *testing.T) {
	tuned, read := "", false
	d := faultDash(t)
	d.cfg.TuneRelay = func(k string) { tuned = k }
	d.cfg.ReadReport = func() { read = true }
	d.relayFault.focus = 0 // sitting on "Recommended", never confirmed
	base := time.Now()
	at := base
	d.now = func() time.Time { return at }

	// The command is run only once the window has CLOSED. Every other tick
	// returns the shimmer's own tea.Tick, and running that blocks for its 300 ms
	// — fifteen of them turned a millisecond test into four seconds.
	step := func() Dashboard {
		t.Helper()
		m, cmd := d.Update(tickMsg{})
		next := m.(Dashboard)
		if cmd != nil && next.modal != modalRelayFault {
			cmd()
		}
		return next
	}
	d = step() // the first tick starts the clock, it does not spend it
	if d.relayFault.left != relayFaultSeconds {
		t.Fatalf("the first tick spent %d s; it only starts the clock", relayFaultSeconds-d.relayFault.left)
	}
	// FOUR TICKS INSIDE ONE SECOND SPEND NOTHING: the tick is 300 ms and its
	// rate is not this window's business.
	for i := 1; i <= 4; i++ {
		at = base.Add(time.Duration(i) * 200 * time.Millisecond)
		d = step()
	}
	if d.relayFault.left != relayFaultSeconds {
		t.Errorf("ticks inside a second spend %d s, want none", relayFaultSeconds-d.relayFault.left)
	}
	// AND IT RUNS DOWN, one second at a time, to the ruled number.
	at = base.Add(time.Second)
	if d = step(); d.relayFault.left != relayFaultSeconds-1 {
		t.Fatalf("a second passed and the clock reads %d, want %d", d.relayFault.left, relayFaultSeconds-1)
	}
	for i := 2; i <= relayFaultSeconds; i++ {
		at = base.Add(time.Duration(i) * time.Second)
		d = step()
	}
	// THE CLOCK ACTS. Reaching zero takes the FALL-THROUGH, whatever the cursor
	// is on — a listener who walked elsewhere and did not press enter has not
	// chosen it — and the window closes.
	if d.relayFault.left != 0 {
		t.Errorf("after %d s the clock reads %d", relayFaultSeconds, d.relayFault.left)
	}
	if d.modal == modalRelayFault {
		t.Error("the clock ran out and the window is still open; its default never fired")
	}
	if tuned != "" || !read {
		t.Errorf("the clock reads the report; tuned=%q read=%v", tuned, read)
	}
}

// THE CLOCK'S ARITHMETIC, at the unit it is defined in. The wire is pinned
// above; this is the boundary behaviour the wire drives.
func TestTheCountdownRunsOnWallTime(t *testing.T) {
	d := faultDash(t)
	if d.relayFault.left != relayFaultSeconds {
		t.Fatalf("the window opens on %d, got %d", relayFaultSeconds, d.relayFault.left)
	}
	now := time.Now()
	d, done := d.stepRelayFault(now) // the first step only sets the mark
	if done || d.relayFault.left != relayFaultSeconds {
		t.Fatalf("the first tick starts the clock, it does not spend it: left=%d", d.relayFault.left)
	}
	// Four ticks inside one second spend nothing.
	for i := 1; i <= 4; i++ {
		d, done = d.stepRelayFault(now.Add(time.Duration(i) * 200 * time.Millisecond))
	}
	if d.relayFault.left != relayFaultSeconds {
		t.Errorf("ticks inside a second spend nothing, left=%d", d.relayFault.left)
	}
	// And it runs out after exactly the ruled number of seconds.
	for i := 1; i <= relayFaultSeconds; i++ {
		d, done = d.stepRelayFault(now.Add(time.Duration(i) * time.Second))
	}
	if !done || d.relayFault.left != 0 {
		t.Errorf("the clock runs out at %d s: done=%v left=%d", relayFaultSeconds, done, d.relayFault.left)
	}
}

// EVERY WAY OUT ACTUALLY ACTS, and closes the window.
//
// The window's whole point is that the listener is not left in silence, so a
// choice that closes the window without tuning or reading is the defect it
// exists to prevent — and it would look exactly like success.
func TestEveryWayOutOfTheRelayFaultActs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		focus    int
		wantTune string
		wantRead bool
	}{
		{"the recommended relay", 0, "a", false},
		{"the alternate relay", 1, "b", false},
		{"the fall-through", 2, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tuned, read := "", false
			d := faultDash(t)
			d.cfg.TuneRelay = func(k string) { tuned = k }
			d.cfg.ReadReport = func() { read = true }
			d.relayFault.focus = tc.focus
			// THROUGH THE KEYBOARD (red team 2026-09-05). Calling
			// chooseRelayFault directly left the `enter` branch in dashboard.go
			// unpinned: replacing it with `return d, true` — the listener
			// presses enter, nothing tunes, nothing reads, the window stays
			// open — kept this package green.
			m, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			next := m.(Dashboard)
			if cmd == nil {
				t.Fatal("choosing must produce the action")
			}
			cmd()
			if next.modal != modalNone {
				t.Error("choosing closes the window")
			}
			if tuned != tc.wantTune || read != tc.wantRead {
				t.Errorf("tuned=%q read=%v, want %q/%v", tuned, read, tc.wantTune, tc.wantRead)
			}
		})
	}
}

// THE CLOCK TAKES THE FALL-THROUGH, whatever the cursor is on. A listener who
// walked to another row and did not press enter has not chosen it.
func TestTheCountdownTakesTheFallThroughNotTheCursor(t *testing.T) {
	tuned, read := "", false
	d := faultDash(t)
	d.cfg.TuneRelay = func(k string) { tuned = k }
	d.cfg.ReadReport = func() { read = true }
	d.relayFault.focus = 0 // sitting on "Recommended", never confirmed
	next, cmd := d.fallThroughRelayFault()
	if cmd == nil {
		t.Fatal("the countdown must produce the action")
	}
	cmd()
	if tuned != "" || !read {
		t.Errorf("the clock reads the report; tuned=%q read=%v", tuned, read)
	}
	if next.modal != modalNone {
		t.Error("the clock closes the window")
	}
}

// esc closes it and does nothing else: the listener said no.
func TestEscClosesTheRelayFaultWithoutActing(t *testing.T) {
	tuned, read := "", false
	d := faultDash(t)
	d.cfg.TuneRelay = func(k string) { tuned = k }
	d.cfg.ReadReport = func() { read = true }
	var m tea.Model = d
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.(Dashboard).modal != modalNone {
		t.Error("esc closes the window")
	}
	if tuned != "" || read {
		t.Errorf("esc acts on nothing; tuned=%q read=%v", tuned, read)
	}
}

// TestTheRelayFaultWindowRespondsToTheKEYS — D-12, and the reason this window
// shipped feeling dead.
//
// Every other test in this file reaches past the keyboard: the wrap test calls
// handleRelayFaultNav, the clock test calls stepRelayFault. Both were green
// while, in a listener's hands, THE ARROWS APPEARED TO DO NOTHING AND THE CLOCK
// NEVER MOVED — because the focus mark was untinted and no tick was ever armed,
// so nothing on screen changed between key presses. A pin that starts one layer
// below the key cannot see either. This one starts at the key.
func TestTheRelayFaultWindowRespondsToTheKeys(t *testing.T) {
	d := faultDash(t)

	// THE CLOCK MUST BE RUNNING. The window acts by itself when it reaches zero,
	// so a frame that does not tick is a window whose default never fires and
	// which never redraws between presses.
	if !d.tickNeeded() {
		t.Error("the window is open and no tick is wanted; its countdown can never step and its frame never redraws")
	}

	rowOf := func(x Dashboard, want string) string {
		t.Helper()
		for _, l := range strings.Split(stripANSITest(x.renderModal(x.opts())), "\n") {
			if strings.Contains(l, want) {
				return l
			}
		}
		t.Fatalf("no %q row in the window", want)
		return ""
	}
	marked := func(line string) bool { return strings.Contains(line, "\u203a") }

	if !marked(rowOf(d, "Recommended")) {
		t.Error("the window opens with nothing marked; a listener cannot tell what enter would take")
	}
	// EVERY PRESS, THROUGH Update, the way a listener makes it — INCLUDING THE
	// WRAP AT BOTH ENDS. The wrap had a test of its own that called
	// handleRelayFaultNav directly, which is the shape this file's own comments
	// name as the reason the window shipped dead; it is folded in here (red team
	// 2026-09-05).
	for _, tc := range []struct {
		name  string
		keys  []rune
		focus int
		row   string
	}{
		{"down moves to the second row", []rune{tea.KeyDown}, 1, "Alternate"},
		{"up comes back", []rune{tea.KeyDown, tea.KeyUp}, 0, "Recommended"},
		{"up from the first WRAPS to the last", []rune{tea.KeyUp}, 2, "Fall-Thru"},
		{"down from the last wraps to the first", []rune{tea.KeyUp, tea.KeyDown}, 0, "Recommended"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := faultDash(t)
			for _, k := range tc.keys {
				m, _ := x.Update(tea.KeyPressMsg{Code: k})
				x = m.(Dashboard)
			}
			if x.relayFault.focus != tc.focus {
				t.Fatalf("the focus is %d, want %d", x.relayFault.focus, tc.focus)
			}
			// AND THE SCREEN SAYS SO. Stopping dead at an end reads as a stuck
			// key (UAT 2026-08-30 #11), and a mark that does not move reads as
			// an arrow that does nothing.
			if !marked(rowOf(x, tc.row)) {
				t.Errorf("%q is not marked; the arrows do nothing a listener can see", tc.row)
			}
		})
	}
}

// TestTheFocusMarkIsTintedByTheListsOwner. The mark moving is not enough: it was
// a bare glyph with NO COLOUR, on a window a listener meets while something is
// already wrong. render/list.go owns how every list marks focus (D-1), and this
// window is a list.
func TestTheFocusMarkIsTintedByTheListsOwner(t *testing.T) {
	// COLOUR ON, DELIBERATELY. Tests run with it off, and with it off every
	// assertion below is satisfied by the bare glyph — which is exactly the
	// state that shipped. A tint pin that cannot see tint is not a pin.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	d := faultDash(t)
	o := d.opts()
	// The owner's own output, so this cannot pass by matching a colour written
	// out a second time here.
	wantMark := o.ListMark(true)
	wantLabel := render.ListLabel(render.PadTo("Recommended", relayFaultLabelW)+":", true)
	if !strings.Contains(wantMark, "\x1b[") || !strings.Contains(wantLabel, "\x1b[") {
		t.Fatal("the list owner emitted no colour with colour on; this test is measuring nothing")
	}
	lines, _, _ := d.relayFaultLines(o)
	body := strings.Join(lines, "\n")
	if !strings.Contains(body, wantMark) {
		t.Error("the focused row does not use the list's pointer; this window marks focus its own way")
	}
	if !strings.Contains(body, wantLabel) {
		t.Error("the focused label is not the list's focus colour; the pointer moves and the row does not")
	}
	// AND AN UNFOCUSED ROW IS NOT WEARING IT. Without this, a row that tinted
	// everything would pass — and a list where every row looks focused marks
	// nothing at all.
	if strings.Count(body, wantMark) != 1 {
		t.Errorf("the focus pointer appears %d times; exactly one row has the focus", strings.Count(body, wantMark))
	}
}

// TestASecondSilenceReportDoesNotDisturbTheOpenWindow — the UAT defect, and the
// reason the arrows "did not work" in a listener's hands while every test here
// said they did.
//
// The silence detector fires once per STREAM, and the engine falls through to
// the next mount. A station broadcasting silence on every mount — the outage
// this window exists for — reports again about every five seconds. Each report
// rebuilt the state: the cursor snapped back to the first row and the countdown
// restarted at ten, so an arrow press was undone before the listener could act
// on it.
//
// EVERY OTHER TEST IN THIS FILE OPENS THE WINDOW ONCE, which is why none of them
// could see it.
func TestASecondSilenceReportDoesNotDisturbTheOpenWindow(t *testing.T) {
	d := faultDash(t)

	// The clock spends a second and THEN the listener walks to the second way
	// out. That order, and not the other one: moving the cursor holds the clock
	// (FR-6.4), so a fixture that pressed the key first would have a clock that
	// cannot spend anything and would prove nothing about a second report.
	now := time.Now()
	d, _ = d.stepRelayFault(now)
	d, _ = d.stepRelayFault(now.Add(time.Second))
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	d = m.(Dashboard)
	if d.relayFault.focus != 1 || d.relayFault.left != relayFaultSeconds-1 {
		t.Fatalf("the fixture must have a moved cursor and a spent second, got focus=%d left=%d",
			d.relayFault.focus, d.relayFault.left)
	}

	// THE NEXT MOUNT GOES QUIET. The window is already up and saying the right
	// thing; this must not touch it.
	m2, _ := d.Update(RelaySilentMsg{Candidates: []RelayCandidate{{Label: "<a third relay>", Key: "c"}}})
	after := m2.(Dashboard)
	if after.relayFault.focus != 1 {
		t.Errorf("a second report moved the cursor to %d; the listener's arrow press was undone", after.relayFault.focus)
	}
	if !after.relayFault.held {
		t.Errorf("a second report released the hold the listener's arrow press put on the clock: " +
			"the window is counting down again at someone who is mid-choice")
	}
	if after.relayFault.left != relayFaultSeconds-1 {
		t.Errorf("a second report restarted the clock at %d; it can never reach zero and the fall-through never fires",
			after.relayFault.left)
	}
	if after.modal != modalRelayFault {
		t.Errorf("the window closed on a second report, modal=%v", after.modal)
	}

	// AND ONCE IT HAS BEEN ANSWERED, A NEW REPORT OPENS A NEW WINDOW. Without
	// this the guard would be "never show it twice", which is a station that
	// goes silent and says nothing the second time.
	chosen, _ := after.chooseRelayFault()
	if chosen.modal == modalRelayFault {
		t.Fatal("choosing a way out must close the window")
	}
	m3, _ := chosen.Update(RelaySilentMsg{Candidates: []RelayCandidate{{Label: "<a fourth relay>", Key: "d"}}})
	reopened := m3.(Dashboard)
	if reopened.modal != modalRelayFault {
		t.Error("a report after the window was answered must raise it again")
	}
	if reopened.relayFault.focus != 0 || reopened.relayFault.left != relayFaultSeconds {
		t.Errorf("a fresh window starts at the first row with a full clock, got focus=%d left=%d",
			reopened.relayFault.focus, reopened.relayFault.left)
	}
}

// TestAWindowWithACursorRedrawsWhenItMoves — the defect three rounds of UAT
// kept reporting, and the reason none of the pins above could see it.
//
// The frame is MEMOISED on a key derived from the model (modalKeyFor). This
// window's cursor and its clock were absent from that key, so the memo rendered
// the window ONCE and replayed that frame for as long as it was open: the arrows
// moved the cursor, the countdown ran down, the fall-through fired on time — and
// the display never changed. Everything underneath was correct, which is why
// every test passed and the window was still dead in the hand.
//
// IT GOES THROUGH modalView, NOT renderModal. That distinction IS the test: the
// pins above call renderModal, which is the memo's MISS path, so they could
// never see a stale hit. A pin that starts below the cache cannot see a cache
// bug.
//
// THE TABLE IS THE POINT. This is a rule about every window that draws a cursor,
// not about this one, and the ctrl+d window had the identical hole.
func TestAWindowWithACursorRedrawsWhenItMoves(t *testing.T) {
	for _, tc := range []struct {
		name string
		open func(Dashboard) Dashboard
		move func(Dashboard) Dashboard
	}{
		{
			name: "the relay-fault window's cursor",
			open: func(d Dashboard) Dashboard {
				return d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
					{Label: "<Next Best Relay>", Key: "a"}, {Label: "<2nd best Relay>", Key: "b"},
				}})
			},
			move: func(d Dashboard) Dashboard { return d.handleRelayFaultNav("nav-down") },
		},
		{
			// THE SAME HOLE, IN THE WINDOW NEXT DOOR. Found while fixing this
			// one: modalDebug was equally absent from the key, so ctrl+d's
			// arrows were equally dead. A rule with one instance is an anecdote.
			name: "the ctrl+d window's cursor",
			open: func(d Dashboard) Dashboard {
				d.cfg.InjectAlert = func(string) {}
				d.cfg.DebugScenarios = []DebugScenario{{Label: "one alert", Key: "one"}, {Label: "a burst", Key: "burst"}}
				return d.open(modalDebug)
			},
			move: func(d Dashboard) Dashboard { return d.handleDebugNav("nav-down") },
		},
		{
			name: "the relay-fault window's clock",
			open: func(d Dashboard) Dashboard {
				return d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{{Label: "<relay>", Key: "a"}}})
			},
			move: func(d Dashboard) Dashboard {
				now := time.Now()
				d, _ = d.stepRelayFault(now)
				d, _ = d.stepRelayFault(now.Add(time.Second))
				return d
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := dash(t).(Dashboard)
			d.width, d.height = 133, 44
			d = tc.open(d)
			o := d.opts()

			before := d.modalView(o) // THE MEMOISED PATH, the one the app paints from
			if before == "" {
				t.Fatal("the window drew nothing; this measures nothing")
			}
			moved := tc.move(d)
			// The fixture must actually change something, or the assertion below
			// is satisfied by a move that did not happen.
			if raw := moved.renderModal(o); raw == d.renderModal(o) {
				t.Fatal("the move changed nothing in the window itself; the fixture is wrong")
			}
			if got := moved.modalView(o); got == before {
				t.Error("the frame did not change when the cursor did: the memo key is missing what moved, " +
					"so the model advances and the listener sees a still picture")
			}
		})
	}
}

// TestEveryLineClearsBothMargins — the global three-cell inset, on BOTH sides
// (HUM LEAD, UAT 2026-09-05: "the content runs up against the right edge").
//
// THE FIXTURE USES A REAL STATION'S LABEL. The mock's placeholders are short, so
// every line happened to clear the right border and nothing measured whether it
// had to. A NOAA mount reads "KEC62 - San Diego, CA (NOAA Weather Radio All
// Hazards, 162.400 MHz)" and ran straight into it. A margin that holds only for
// short content is not a margin.
func TestEveryLineClearsBothMargins(t *testing.T) {
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d = d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
		{Label: "KEC62 - San Diego, CA (NOAA Weather Radio All Hazards, 162.400 MHz)", Key: "a"},
		{Label: "WWG21 - Mount Woodson, CA (NOAA Weather Radio All Hazards, 162.550 MHz)", Key: "b"},
	}})
	lines := strings.Split(stripANSITest(d.modalView(d.opts())), "\n")
	if len(lines) < 5 {
		t.Fatalf("the window drew %d lines; this measures nothing", len(lines))
	}
	checked := 0
	for _, l := range lines {
		r := []rune(l)
		if len(r) < 3 || r[0] != '│' {
			continue // the borders themselves
		}
		inner := string(r[1 : len(r)-1])
		if strings.TrimSpace(inner) == "" {
			continue // a blank spacer line has no content to inset
		}
		checked++
		lead := len(inner) - len(strings.TrimLeft(inner, " "))
		trail := len(inner) - len(strings.TrimRight(inner, " "))
		if lead < modalInset {
			t.Errorf("a line starts %d cells from the border, want %d: |%s|", lead, modalInset, l)
		}
		if trail < modalInset {
			t.Errorf("a line ends %d cells from the border, want %d: |%s|", trail, modalInset, l)
		}
	}
	// THE ROWS AND THE FOOTER MUST BE AMONG WHAT WAS CHECKED. A window that
	// drew only its prose would satisfy every assertion above.
	if checked < 8 {
		t.Errorf("only %d content lines were measured; the rows or the footer are missing", checked)
	}
	body := strings.Join(lines, "\n")
	if !strings.Contains(body, "Recommended") || !strings.Contains(body, "Auto Close in") {
		t.Error("the fixture must include the action rows and the footer, which are what overflowed")
	}
}

// TestALineTooLongForTheWindowIsWrappedNotCut gives the body's bound a FAILING
// INPUT, and pins WHICH bound it is.
//
// The prose above the rows is fixed text that happens to fit, so the margin
// property test cannot exercise this — remove the bound and nothing fails, which
// is a guard nobody can falsify (D-2).
//
// WRAPPED, NOT CUT. An earlier pass truncated, which is the class UAT 25 ruled
// out in as many words on WrapLines. This asserts the difference directly: every
// word of the input is still present afterwards.
func TestALineTooLongForTheWindowIsWrappedNotCut(t *testing.T) {
	words := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
		"india", "juliett", "kilo", "lima", "mike", "november", "oscar", "papa", "quebec"}
	long := strings.Join(words, " ")
	content := relayFaultContentFor(dash(t).(Dashboard).opts())
	got := relayFaultInsetLines(long, content)
	if len(got) < 2 {
		t.Fatalf("a line %d cells long came back as %d line(s); it was not wrapped", len(long), len(got))
	}
	for _, l := range got {
		if w := render.Width(l); w > relayFaultPad+content {
			t.Errorf("a wrapped line is %d cells wide, want at most %d: %q", w, relayFaultPad+content, l)
		}
	}
	// NOTHING WAS LOST. This is the whole point in an error window: what gets
	// cut is the address of the station the listener is being told to tune to.
	joined := strings.Join(got, " ")
	for _, w := range words {
		if !strings.Contains(joined, w) {
			t.Errorf("%q is missing from the wrapped output; the line was cut, not wrapped", w)
		}
	}
}

// TestARowWrapsUnderItsOwnValue — the HUM LEAD's shape (UAT 2026-09-05):
//
//	›  Recommended:  Tune to WNG712 Coachella / Spanish CA
//	                 162.525 MHz (81 mi)
//
// The continuation sits under the VALUE, not back at the margin, where it would
// read as another way out — and there is air between the rows.
func TestARowWrapsUnderItsOwnValue(t *testing.T) {
	d := dash(t).(Dashboard)
	d.width, d.height = 133, 44
	d = d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
		{Label: "KEC62 San Diego CA 162.400 MHz - 12 mi (nearest relayed, and then some more words)", Key: "a"},
		{Label: "WNG712 Coachella / Spanish CA 162.525 MHz - 81 mi", Key: "b"},
	}})
	lines := strings.Split(stripANSITest(d.modalView(d.opts())), "\n")

	first, cont := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Recommended") {
			first = i
			continue
		}
		if first >= 0 && cont < 0 && strings.Contains(l, "relayed") {
			cont = i
		}
	}
	if first < 0 || cont < 0 {
		t.Fatalf("the row did not wrap; this measures nothing:\n%s", strings.Join(lines, "\n"))
	}
	if cont != first+1 {
		t.Errorf("the continuation is %d lines below its row, want the next line", cont-first)
	}
	// THE HANGING INDENT: the continuation begins in the same column the value
	// does, which is what makes the two lines read as one row.
	//
	// MEASURED IN RUNES, INSIDE THE BORDER. The box drawing is multi-byte, so a
	// byte index counts the border as three columns and the two lines come out
	// two apart when they are aligned — which cost a cycle.
	inner := func(l string) []rune { r := []rune(l); return r[1 : len(r)-1] }
	valueAt := runeIndex(inner(lines[first]), "Tune to")
	contRunes := inner(lines[cont])
	contAt := len(contRunes) - len([]rune(strings.TrimLeft(string(contRunes), " ")))
	if valueAt < 0 {
		t.Fatal("the row does not read \"Tune to\"; the fixture has moved")
	}
	if contAt != valueAt {
		t.Errorf("the continuation starts at column %d and the value at %d; it falls back to the margin and reads as another way out",
			contAt, valueAt)
	}
	// AND THE ROWS ARE SEPARATED. Without the air they read as one block.
	var alt int
	for i, l := range lines {
		if strings.Contains(l, "Alternate") {
			alt = i
		}
	}
	if alt == 0 {
		t.Fatal("no Alternate row")
	}
	if strings.TrimSpace(string(inner(lines[alt-1]))) != "" {
		t.Errorf("there is no air between the ways out; line above Alternate is %q", lines[alt-1])
	}
}

// runeIndex is strings.Index in RUNE columns, which is what a terminal column
// is: the box drawing here is multi-byte and a byte offset is not a column.
func runeIndex(r []rune, want string) int {
	i := strings.Index(string(r), want)
	if i < 0 {
		return -1
	}
	return len([]rune(string(r)[:i]))
}

// TestTheFocusedWayOutIsAlwaysOnSCREEN, at the app's documented floor.
//
// RED TEAM 2026-09-05: floatModalFooter took its scroll from `setupScroll`,
// which read SETUP's body whatever window was open — so this window's scroll was
// a permanent zero, nothing else writes d.modalScroll, and at 80x24 EVERY WAY
// OUT WAS BELOW THE FOLD. The rail drew its arrows, the cursor moved, the screen
// did not change: the reported "arrows do not work", reached by geometry rather
// than by the memo, and invisible at the 133x44 the UAT ran at.
//
// A focused row the listener cannot see is a dead keyboard, and this window is
// one they meet while something is already wrong.
func TestTheFocusedWayOutIsAlwaysOnScreen(t *testing.T) {
	m, err := NewDashboard(Config{Version: "0.1.0-test"})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) // the documented floor
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	d := model.(Dashboard).openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{
		{Label: "KEC62 San Diego CA 162.400 MHz - 12 mi", Key: "a"},
		{Label: "WNG712 Coachella / Spanish CA 162.525 MHz - 81 mi", Key: "b"},
	}})

	// THE FIXTURE MUST NOT FIT. If the whole window fits on screen the scroll is
	// legitimately zero and this proves nothing.
	all, _, _ := d.relayFaultLines(d.opts())
	if len(all) <= d.modalMax() {
		t.Fatalf("the window is %d lines and the budget is %d; it fits, so this measures nothing",
			len(all), d.modalMax())
	}

	want := []string{"Recommended", "Alternate", "Fall-Thru"}
	for i, label := range want {
		got := stripANSITest(d.modalView(d.opts()))
		if !strings.Contains(got, label) {
			t.Errorf("with the cursor on row %d, %q is off screen — the listener cannot see what enter would take:\n%s",
				i, label, got)
		}
		if i < len(want)-1 {
			next, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			d = next.(Dashboard)
		}
	}
}

// THE CLOCK STOPS WHEN THE LISTENER STARTS CHOOSING (FR-6.4).
//
// The window acts for you after ten seconds, and what it acts on is the only
// station-tuning control in the app. Ten seconds is enough to read four lines
// and not enough to read them and decide — so the listener who IS deciding is
// exactly the one it takes the choice away from, mid-keystroke.
//
// MVS-D-76 IS UNTOUCHED. Doing nothing still falls through at ten seconds, and
// still through the one door that enter uses. What changes is that pressing a
// key is no longer doing nothing.
func TestMovingTheCursorHoldsTheRelayFaultClock(t *testing.T) {
	d := dash(t).(Dashboard)
	d = d.openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{{Label: "KEC62", Key: "a"}}})
	d = d.handleRelayFaultNav("nav-down")

	now := time.Now()
	for i := range relayFaultSeconds + 5 {
		var out bool
		d, out = d.stepRelayFault(now.Add(time.Duration(i+1) * time.Second))
		if out {
			t.Fatalf("the window took the fall-through %d seconds after the listener moved the "+
				"cursor: it acted for someone who was in the middle of choosing", i+1)
		}
	}

	// AND AN UNTOUCHED WINDOW STILL FALLS THROUGH, or this removed the ruling
	// instead of narrowing it.
	idle := dash(t).(Dashboard).openRelayFault(RelaySilentMsg{Candidates: []RelayCandidate{{Label: "KEC62", Key: "a"}}})
	var ran bool
	// relayFaultSeconds+1 steps: the first sets the clock's mark and spends
	// nothing, which is what makes the countdown wall-time rather than ticks.
	for i := range relayFaultSeconds + 1 {
		idle, ran = idle.stepRelayFault(now.Add(time.Duration(i+1) * time.Second))
	}
	if !ran {
		t.Error("an untouched window no longer falls through: MVS-D-76 was removed, not narrowed")
	}
}
