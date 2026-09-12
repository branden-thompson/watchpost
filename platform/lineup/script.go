package lineup

// script.go — a card's words, in parts (MVS-D-77, T3.8).
//
// A CARD'S SCRIPT IS NOT A FLAT STRING, and the reason is the pacing. MVS-D-72
// gives a takeover a shape the listener hears — tone, 2 s, header, 1 s, line,
// 1 s, … 2 s, tail — and a Reader handed one blob has to infer where the breaks
// go. Inferring them from newlines works until an alert contains one.
//
// The parts also ARE the display: the Broadcaster shows a takeover's header,
// its alert lines and its tail as separate rows (the mock's `[T]` panel). One
// representation serves both, so what a listener hears and what an operator
// reads cannot drift apart.
//
// WHAT IS SAID TRAVELS; HOW LONG TO WAIT DOES NOT. The gaps are the Reader's —
// they are one ruling, they apply to every takeover, and a card carrying its own
// durations would be a card that could contradict the ruling. The part's KIND is
// what the Reader needs, and it is the least it can be given.

import "strings"

// PartKind is what a part of a script is FOR, which is how the Reader knows what
// pause follows it.
type PartKind int

const (
	// PartLine is one alert's line — the body of a takeover, and the whole of a
	// location report.
	PartLine PartKind = iota
	// PartHead opens a burst: who declared what is about to be read.
	PartHead
	// PartTail closes it.
	PartTail
)

// String names the kind for the timeline (DR-23).
func (k PartKind) String() string {
	switch k {
	case PartLine:
		return "line"
	case PartHead:
		return "head"
	case PartTail:
		return "tail"
	}
	return ""
}

// Part is one thing the Reader says.
//
// Ref names the producer's record this part is about, so the Reader can put the
// right callout on the band while it speaks — and it is EMPTY for a structural
// part, because a header is about the burst rather than about any one alert.
type Part struct {
	Kind PartKind
	Text string
	Ref  string
}

// Content is one line of a card's MANIFEST: what a read draws on, and what that
// source has to say about how much of it there is (D-87).
//
// Detail is deliberately loose — a forecast covers a span of dates, a fire
// report counts hotspots — because the console's heading is "RANGE / INCIDENTS"
// and a typed field would have to pick one of them.
//
// IT LIVES BESIDE Script BECAUSE THEY ARE ONE ANSWER. The words and the summary
// of the words come from the same compose; putting them in different files
// would invite a second producer for one of them.
type Content struct{ Name, Detail string }

// Script is a card's words in the order they are said.
//
// Tone is the attention tone that opens the card, as the cast class's key, and
// empty when the card opens with none. It is a string rather than the domain's
// own type because this package may not know about voices — the executor
// resolves it, the same way it resolves Ref.
type Script struct {
	Tone  string
	Parts []Part
}

// Text is the whole script as one string, for anything that wants the words
// without the shape: a log line, a summary, an invariant.
//
// ONE TRUTH. The parts are the script; this is a view of them, computed rather
// than stored, so there is no second copy to fall out of step.
func (s Script) Text() string {
	// A BUILDER, NOT `out +=` (P-2): the old form reallocated and copied the
	// whole string once per part.
	var b strings.Builder
	for _, p := range s.Parts { // bounded by the script (P10-02)
		if p.Text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p.Text)
	}
	return b.String()
}

// Empty reports whether the script says nothing at all.
// IT DOES NOT BUILD THE TEXT TO ANSWER A BOOLEAN (red team 2026-09-05, P-2).
// This was `s.Text() == ""`, and Text concatenates with `out += p.Text` in a
// loop — quadratic string building, allocating the whole script, to decide
// whether any part has a character in it. It is called from invariants on the
// settle path of every event, which includes every one-second tick.
func (s Script) Empty() bool {
	for _, p := range s.Parts { // bounded by the script (P10-02)
		if p.Text != "" {
			return false
		}
	}
	return true
}

// Lines is the parts of one kind, in order — what the Reader walks and what the
// Broadcaster lists.
func (s Script) Lines(k PartKind) []Part {
	var out []Part
	for _, p := range s.Parts { // bounded by the script (P10-02)
		if p.Kind == k {
			out = append(out, p)
		}
	}
	return out
}

// Say is a one-part script: the shape of a card that says a single thing — a
// transition, a notice, a location report. Most cards are this, so the common
// case reads as one call rather than a struct literal.
func Say(text string) Script {
	if text == "" {
		return Script{}
	}
	return Script{Parts: []Part{{Kind: PartLine, Text: text}}}
}
