package app

// severe_read.go — [space] in the Severe Weather / Disaster Events window
// reads the FOCUSED EVENT over the radio (HUM LEAD UAT 2026-08-28, option
// B): the event's own script — the alert itself, no conditions or forecast —
// spoken through the director as a narrateRead sequence: it ducks the
// broadcast, waits behind a breaking takeover, is PAUSED by one and resumes
// after it, and the radio panel shows the event while it plays, returning
// to whatever was on afterwards. A second [space] while one is reading is
// inert (the whole record is read; the engine's ceiling is ten minutes).

import (
	"context"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"strings"
	"sync"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/render"
)

// The read speaks the WHOLE record (HUM LEAD UAT 2026-08-28: "I want the
// full report") — the parser bounds the prose at 4 000 runes, and a takeover
// pauses rather than cuts it. The no-voice hold follows the same length.
const (
	readingHoldPerRune = 60 * time.Millisecond // the overlay's hold when no voice can render (≈ reading pace)
	readingHoldMax     = 5 * time.Minute
)

// eventReader narrates one severe event on request.
type eventReader struct {
	mu sync.Mutex
	// press serialises [space]: each keypress arrives on its own tea.Cmd
	// goroutine, so one decision must not race another (T3.2b review).
	press sync.Mutex
	busy  bool
	key   string // the event being read, so a second [space] on the SAME row is a pause
	// gen is which read is CURRENT. A read winding down must not clear the
	// bookkeeping of the one that replaced it — the radioDeck.epoch pattern, for
	// the same reason (UAT 2026-09-03).
	//
	// Without it, fast input stacked reads: [space], [esc], reopen, [space] on
	// another row. The first read's cleanup ran AFTER the second had started,
	// set busy=false while it was playing and cleared its mark — so the next
	// [space] launched a third read and the window no longer knew which one to
	// pause.
	gen     uint64
	nar     *director                                                 // the voice arbiter (app/director.go)
	ctx     context.Context                                           // the app's: shutdown ends a read (A-08)
	cancel  context.CancelFunc                                        // the read in progress, under mu
	done    chan struct{}                                             // closed when the read in progress ends
	scripts *script.Library                                           // the report's phrases ("event-report.*"); nil = built-in
	row     func(key string) (tty.SevereRow, bool)                    // the deck's last publish
	send    func(tea.Msg)                                             // SevereReadingMsg to the dashboard
	status  func(station, short, detail string, spoken time.Duration) // the radio panel overlay while it plays (long and narrow heads); nil = none
	restore func()                                                    // re-sends the true radio status afterwards; nil = none
}

// newEventReader builds the reader; every read runs under ctx (the app's —
// shutdown ends a read in progress, red-team round 4 A-08).
func newEventReader(ctx context.Context, nar *director, scripts *script.Library, row func(string) (tty.SevereRow, bool), send func(tea.Msg)) *eventReader {
	if ctx == nil {
		ctx = context.Background()
	}
	return &eventReader{ctx: ctx, nar: nar, scripts: scripts, row: row, send: send}
}

// Toggle is `[space]` in the window: it starts this event's read, or pauses and
// resumes the one already running (MVS-D-74).
//
// THE KEY DECIDES WHICH. Pressing space on the row being read is a play/pause;
// pressing it on a DIFFERENT row while one reads is a request for that other
// event, so the first is ended and the new one starts. Ignoring the second
// press would leave a listener pressing a key that does nothing on a row that
// looks ready — the complaint that produced this ruling, one level down.
func (r *eventReader) Toggle(key string) {
	// ONE PRESS AT A TIME. Bubbletea runs each Cmd on its own goroutine, so two
	// fast presses are genuinely concurrent Toggles — not serialised by the
	// update loop. Each read its state, released the lock, and then acted on a
	// snapshot the other had already invalidated: the loser could pause the read
	// the winner had just started, and marked the row with the OLD key, putting
	// the ▶ on a row that was not reading. Measured at about 2 % of presses, and
	// invisible to the race detector because it is a logic race, not a data one.
	//
	// r.mu cannot serve for this: the decision calls mark and Read, which take
	// it again. A press gate is its own lock, held across the whole decision.
	r.press.Lock()
	defer r.press.Unlock()

	r.mu.Lock()
	busy, reading := r.busy, r.key
	r.mu.Unlock()
	switch {
	case busy && reading == key:
		if !r.nar.resumeRead() { // not paused, so this press is the pause
			r.nar.pauseRead()
		}
		r.mark(key)
		return
	case busy:
		// A DIFFERENT ROW: free the reader AT ONCE rather than waiting for the
		// old goroutine. Waiting here was up to two seconds of a key doing
		// nothing, and the wait was never the point — the old read is stale the
		// moment this one is asked for.
		r.Cancel()
	}
	r.Read(key)
}

// mark tells the window what the row is doing now, so the pause is visible.
func (r *eventReader) mark(key string) {
	if r.send != nil {
		r.send(tty.SevereReadingMsg{Key: key, Paused: r.nar.readPaused()})
	}
}

// Read narrates the event with this key; it returns at once (the sequence
// runs on its own goroutine) and is inert while a read is in progress.
func (r *eventReader) Read(key string) {
	r.mu.Lock()
	if r.busy {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(r.ctx)
	done := make(chan struct{})
	r.gen++
	gen := r.gen
	r.busy, r.cancel, r.done, r.key = true, cancel, done, key
	r.mu.Unlock()
	// THE MARK IS SENT BY WHOEVER STARTS THE READ, not by the read's goroutine.
	// Sent there it raced: a read starting slightly later could have its mark
	// land AFTER a newer read's, leaving the window pointing at a row that was
	// no longer reading. Here it is ordered with the state it describes, and
	// Toggle holds the press gate across this call so two presses cannot
	// interleave. Never on the update goroutine — every caller is a tea.Cmd.
	//
	// BEFORE THE GOROUTINE, NOT AFTER. Sent after, a read that finished quickly
	// — a row that has gone from the feed returns almost at once — could clear
	// the mark before this set was ever sent, and the row kept a play mark for a
	// read that had already ended. It showed up as a pin that failed one run in
	// three, which is the shape of an ordering bug rather than a slow machine.
	r.mark(key)
	go r.run(ctx, key, done, gen)
}

// Cancel ends a read in progress WITHOUT waiting for it, and returns at once.
//
// NAMED Cancel, NOT Stop, and the reason is the tool rather than the code: P10
// resolves methods by NAME, so an eventReader.Stop collides with radioDeck.Stop
// — which genuinely participates in a stopDwell cycle and carries its own
// ratified exemption — and the collision reports this function as recursion it
// has no part in. The same false positive cost two renames earlier in this
// release (executors.cue/release, T2.3). A rename is cheaper and more honest
// than an exemption for something that is not recursive.
//
// IT IS THE ONE A KEYPRESS CALLS. `End` waits for the read's goroutine, which is
// right at shutdown and wrong on the render path: closing the window ran that
// wait on Bubbletea's update goroutine and cost about half a second before the
// frame redrew. A TUI that hesitates on a key has given up the only thing it has
// over a browser (HUM LEAD, UAT 2026-09-03).
//
// Cancelling is instant and complete on its own — the read's own context ends,
// its line stops, its hold ends and its mark clears. The WAIT only answers "has
// the goroutine finished", which nobody closing a window is asking.
func (r *eventReader) Cancel() {
	r.mu.Lock()
	cancel, had := r.cancel, r.busy
	r.free()
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	_ = had
}

// clearMarkUnlessReplaced takes the row's mark down, unless a NEWER read has
// taken over and now owns it.
//
// IT RUNS ON THE READ'S OWN GOROUTINE, NEVER THE CALLER'S, and that is the whole
// point. `send` is tea.Program.Send, and `close` — which is what cancels a read
// — runs on Bubbletea's UPDATE goroutine. Sending into the program's message
// channel from inside Update means the loop that drains it is the loop that is
// blocked: nothing drains, no key works, and the only way out is killing the
// terminal. Softlocked at UAT 2026-09-03 by [space], [space], [esc].
// DEFENCE IN DEPTH, AND ITS MUTANT IS EXPECTED TO SURVIVE (D-1's exception).
// A review removed this guard and the whole package stayed green, then tried
// twice to build a case that fails without it and could not: in every path that
// can be constructed, a replacement cannot take the air until the old job
// releases, and the old job's clear runs before that release. The ordering that
// protects it lives in the arbiter, not here. It is kept because the ordering is
// not this function's to rely on — and it is documented so nobody spends another
// hour hunting a pin for it.
func (r *eventReader) clearMarkUnlessReplaced(gen uint64) {
	r.mu.Lock()
	replaced := !r.current(gen) && r.busy // a newer read owns the mark now
	r.mu.Unlock()
	if !replaced && r.send != nil {
		r.send(tty.SevereReadingMsg{})
	}
}

// free makes the reader available at once and makes whatever is winding down
// STALE. Callers hold r.mu.
//
// It is what stops fast input from stacking reads: the reader is free the
// instant a read is cancelled, rather than when its goroutine gets around to
// noticing, and the goroutine that was running can no longer clear the
// bookkeeping of whatever replaced it.
func (r *eventReader) free() {
	r.gen++
	r.busy, r.cancel, r.done, r.key = false, nil, nil, ""
}

// current reports whether gen is still the read in progress. Callers hold r.mu.
func (r *eventReader) current(gen uint64) bool { return r.gen == gen }

// End stops a read in progress and WAITS for it to finish (bounded: the
// director's release is immediate once the context ends). Shutdown wants this;
// a keypress wants Cancel. Inert with none in progress.
func (r *eventReader) End() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.free()
	r.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second): // never wedge a shutdown on a read
	}
}

func (r *eventReader) run(ctx context.Context, key string, done chan struct{}, gen uint64) {
	defer func() {
		r.mu.Lock()
		if r.current(gen) { // a stale read clears nothing: the current one owns this state
			r.free()
		}
		r.mu.Unlock()
		// THE CLEAR IS HERE, NOT INSIDE THE SEQUENCE, so EVERY exit takes the
		// mark down. It lived in the seq closure, which `run` never reaches when
		// the row has gone from the feed between the render and the press — and
		// the set-mark now happens in `Read`, before that check. The window was
		// left showing a play mark for a read that never started, with nothing
		// able to clear it.
		r.clearMarkUnlessReplaced(gen)
		close(done)
	}()
	row, ok := r.row(key)
	if !ok {
		return
	}
	script := eventScript(r.scripts, row)
	// NO TONE ON A [space] READ (MVS-D-69). 0.14.0 opened this read with the
	// row's class tone, the way a takeover does — but a tone is an ATTENTION
	// SIGNAL, and the listener who pressed [space] on a row they are looking at
	// has already given theirs. Every read began with a sound whose only job was
	// to fetch someone who was already here.
	//
	// It also cost four seconds before the first word. Every tone carries a
	// two-second trailing silence (synth.alertTailDur) so a takeover has a beat
	// between the signal and "…has been declared", and the read held for the
	// whole buffer: 2.8-4.0 s depending on class, of which only 0.8-1.4 s is
	// audible. Measured at UAT: tone, then about four seconds, then words.
	r.nar.Run(ctx, narrateRead, cast.SevereRead, true, func(ctx context.Context, s *speaker) {
		dur := s.line(script)
		if dur == 0 { // no voice: hold the overlay long enough to read the script
			dur = min(readingHoldMax, time.Duration(len([]rune(script)))*readingHoldPerRune)
		}
		if r.status != nil {
			place := render.PlainLine(row.Location)
			r.status("EVENT · "+render.PlainLine(row.Product)+" · "+place, "EVENT · "+severe.ProductCode(row.Product)+" · "+place, script, dur)
		}
		s.hold(dur)
	})
	if r.restore != nil {
		r.restore()
	}
}

// eventScript composes the spoken record of one row from the
// "event-report" script (domains/radio/script): head · opening (the product
// for the place) · meta (the record's meta line as a sentence) · window (how
// long it is in effect) · the whole description · the whole instructions ·
// tail. No dates or clock times (they read badly
// aloud and the window shows them); no "Press W" (the reader is already
// there). A phrase whose script is missing is simply not spoken.
func eventScript(lib *script.Library, row tty.SevereRow) string {
	say := func(part string, data any) string { return scriptText(lib, "event-report", part, data) }
	// THE MARKING LEADS THE REPORT (FR-4.4). This read is one script rather than
	// a line per alert, so there is no per-line place to put it — and a listener
	// who asked for the full report on a fabricated event must hear what it is
	// before they hear what it says. In Go rather than in the script file, like
	// the burst's: the read scripts are user-editable, and a marking an edit can
	// delete is not a marking.
	var parts []string
	if row.Test {
		parts = append(parts, testHead(lib))
	}
	parts = append(parts, say("head", nil), say("opening", map[string]string{"Product": render.PlainLine(row.Product), "Location": render.PlainLine(row.Location)}))
	if meta := strings.Trim(render.PlainLine(row.Record.Meta), "[]"); meta != "" {
		parts = append(parts, say("meta", map[string]string{"Items": strings.Join(strings.Split(meta, " · "), ", ")}))
	}
	if window := spokenWindow(row.Record.Timing); window != "" {
		parts = append(parts, say("window", map[string]string{"Window": window}))
	}
	for _, p := range row.Record.Paras {
		p = render.PlainLine(p)
		switch {
		case p == "":
		case strings.HasPrefix(p, "Instructions: "):
			parts = append(parts, say("instructions", map[string]string{"Text": strings.TrimPrefix(p, "Instructions: ")}))
		case strings.HasPrefix(p, "Wind gusts") || strings.HasPrefix(p, "Hail") || strings.HasPrefix(p, "Advisories:"):
			parts = append(parts, p+".")
		default:
			parts = append(parts, p)
		}
	}
	parts = append(parts, say("tail", nil))
	kept := parts[:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
}

// spokenWindow reads the "(~15m)" window the record's timing line carries
// and words it: "15 minutes", "2 hours", "1 hour 30 minutes", "3 days".
func spokenWindow(timing string) string {
	i := strings.LastIndex(timing, "(~")
	j := strings.LastIndex(timing, ")")
	if i < 0 || j < i {
		return ""
	}
	var out []string
	num := ""
	for _, r := range timing[i+2 : j] {
		switch {
		case unicode.IsDigit(r):
			num += string(r)
		case num == "":
			continue
		default:
			unit := durationUnits()[r]
			if unit == "" {
				return ""
			}
			if num != "1" {
				unit += "s"
			}
			out = append(out, num+" "+unit)
			num = ""
		}
	}
	return strings.Join(out, " ")
}

// durationUnits spell the shorthand a Starts/Ends span uses (spokenWindow).
// A function, not a global (P10-06).
func durationUnits() map[rune]string { return map[rune]string{'d': "day", 'h': "hour", 'm': "minute"} }
