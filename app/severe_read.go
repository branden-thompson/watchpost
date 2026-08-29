package app

// severe_read.go — [space] in the Severe Weather / Disaster Events window
// reads the FOCUSED EVENT over the radio (HUM LEAD UAT 2026-08-28, option
// B): the event's own script — the alert itself, no conditions or forecast —
// spoken through the narrator as a narrateRead sequence: it ducks the
// broadcast, waits behind a breaking takeover, is PAUSED by one and resumes
// after it, and the radio panel shows the event while it plays, returning
// to whatever was on afterwards. A second [space] while one is reading is
// inert (the whole record is read; the engine's ceiling is ten minutes).

import (
	"context"
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
	mu      sync.Mutex
	busy    bool
	nar     *narrator                                                 // the voice arbiter (narrate.go)
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
func newEventReader(ctx context.Context, nar *narrator, scripts *script.Library, row func(string) (tty.SevereRow, bool), send func(tea.Msg)) *eventReader {
	if ctx == nil {
		ctx = context.Background()
	}
	return &eventReader{ctx: ctx, nar: nar, scripts: scripts, row: row, send: send}
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
	r.busy, r.cancel, r.done = true, cancel, done
	r.mu.Unlock()
	go r.run(ctx, key, done)
}

// End ends a read in progress — its line stops, its hold ends, its mark
// clears — and waits for it to finish (bounded: the narrator's release is
// immediate once the context ends). Inert with none in progress.
func (r *eventReader) End() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
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

func (r *eventReader) run(ctx context.Context, key string, done chan struct{}) {
	defer func() {
		r.mu.Lock()
		r.busy, r.cancel, r.done = false, nil, nil
		r.mu.Unlock()
		close(done)
	}()
	row, ok := r.row(key)
	if !ok {
		return
	}
	script := eventScript(r.scripts, row)
	r.nar.Run(ctx, narrateRead, true, func(ctx context.Context, s *speaker) {
		r.send(tty.SevereReadingMsg{Key: key})
		defer r.send(tty.SevereReadingMsg{})
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
	parts := []string{say("head", nil), say("opening", map[string]string{"Product": render.PlainLine(row.Product), "Location": render.PlainLine(row.Location)})}
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
