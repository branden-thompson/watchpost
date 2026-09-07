package render

// clock.go — how a time of day is written, for the whole app.
//
// Before this every site picked its own layout: the alert list wrote "3:04 PM",
// the tide table "15:04", the forecast's sunrise "1504" and the Updated stamps
// "15:04:05". Three conventions on one screen, none of them anything a listener
// had chosen. This is the one owner, and the choice behind it lives in
// Settings → WATCHPOST UI → Radio Convention.
//
// The methods are named for what a SITE needs — a bare time, a time with
// seconds, a date and time, a full stamp — rather than for a layout string, so
// a site cannot half-adopt the setting by keeping its own date and borrowing
// the clock.

import (
	"strings"
	"time"
)

// Clock is how times of day are written.
type Clock int

// The three forms, as the sketch draws them.
const (
	Clock12  Clock = iota // 8:00 PM — the default, and what the mocks were drawn with
	Clock24               // 20:00
	ClockMil              // 2000
)

// The config words. A closed set: an unknown one reads as the default rather
// than failing a load, because a clock format is not worth refusing to start
// over.
const (
	clockKey12  = "12h"
	clockKey24  = "24h"
	clockKeyMil = "mil"
)

// ClockByKey is the config word's clock, Clock12 for anything unrecognised.
func ClockByKey(key string) Clock {
	switch key {
	case clockKey24:
		return Clock24
	case clockKeyMil:
		return ClockMil
	}
	return Clock12
}

// Key is the config word for c.
func (c Clock) Key() string {
	switch c {
	case Clock24:
		return clockKey24
	case ClockMil:
		return clockKeyMil
	}
	return clockKey12
}

// Label is how the Settings row names c — the form AND an example of it, so the
// choice can be made without applying it to find out.
func (c Clock) Label() string {
	switch c {
	case Clock24:
		return "24hr (20:00)"
	case ClockMil:
		return "MIL  (2000)"
	}
	return "12hr (8:00 PM)"
}

// ClockOrder is the three forms in the order the Settings group draws them.
func ClockOrder() []Clock { return []Clock{Clock12, Clock24, ClockMil} }

// Time is a bare time of day: "8:00 PM" · "20:00" · "2000".
func (c Clock) Time(t time.Time) string { return t.Format(c.layout()) }

// TimeSec is a time of day carrying seconds — the freshness stamps, where the
// seconds are the point: "8:00:05 PM" · "20:00:05" · "2000:05".
//
// Military time has no conventional seconds form; "2000:05" is the one used
// where they are needed, and it keeps the four-digit clock intact rather than
// running six digits together into something unreadable.
func (c Clock) TimeSec(t time.Time) string {
	switch c {
	case Clock24:
		return t.Format("15:04:05")
	case ClockMil:
		return t.Format("1504:05")
	}
	return t.Format("3:04:05 PM")
}

// DateTime is a date and a time: "01/02 8:00 PM".
func (c Clock) DateTime(t time.Time) string { return t.Format("01/02 " + c.layout()) }

// WeekdayDateTime leads with the day, for a span a listener reads as "when":
// "Mon 01/02 8:00 PM".
func (c Clock) WeekdayDateTime(t time.Time) string {
	return t.Format("Mon 01/02 " + c.layout())
}

// DateTimeZone is a date, a time and the zone it is in: "01/02 8:00 PM MST".
func (c Clock) DateTimeZone(t time.Time) string {
	return t.Format("01/02 " + c.layout() + " MST")
}

// Stamp is the full freshness stamp — date, time to the second, zone:
// "01/02/2006 8:00:05 PM MST".
func (c Clock) Stamp(t time.Time) string {
	return t.Format("01/02/2006 ") + c.TimeSec(t) + t.Format(" MST")
}

// Since is a time written for a reader who may be looking at something that did
// not happen today: the bare time when it did, the date and the time when it did
// not.
//
// The rule is the timestamp, not the caller's idea of how long its events live.
// The ticker holds significant quakes for seven days and a storm for as long as
// the feed lists it, so most of what is on it is not from today and a bare clock
// time silently claims it is.
func (c Clock) Since(t, now time.Time) string {
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return c.Time(t)
	}
	return c.DateTime(t)
}

// Spoken is how a time of day is READ ALOUD.
//
// Spelled out rather than handed to the synthesiser as digits, for the reason
// the marine report spells its numbers out: a layout written for a column is
// not a sentence, and what a voice makes of "16:50" varies by engine. The
// report says what it means to say.
//
// TWELVE- AND TWENTY-FOUR-HOUR READ THE SAME. They
// are two ways of writing one instant, and a person says that instant one way:
// 4:50 PM and 16:50 are both "Four Fifty PM". Only MILITARY is its own reading,
// because it is its own convention out loud as well as on the page —
// "Sixteen-Fifty Hours", and "Oh Five-Hundred Hours" for a round hour.
func (c Clock) Spoken(t time.Time) string {
	h, m := t.Hour(), t.Minute()
	if c != ClockMil {
		hour := h % 12
		if hour == 0 {
			hour = 12
		}
		meridiem := "AM"
		if h >= 12 {
			meridiem = "PM"
		}
		if m == 0 {
			return ones(hour) + " " + meridiem // "Five AM", not "Five O'Clock AM"
		}
		return ones(hour) + " " + minuteWords(m) + " " + meridiem
	}
	part := "Hundred" // a round hour is "Hundred Hours", never "Zero Hours"
	if m > 0 {
		part = minuteWords(m)
	}
	return milHourWords(h) + " " + part + " Hours"
}

// milHourWords is the hour as military time says it: "Zero" at midnight, a
// leading "Oh" below ten, the number itself from ten (HUM LEAD's readings, UAT
// 2026-08-30).
//
// Hyphens appear only INSIDE a compound number — "Twenty-Three", "Fifty-Nine" —
// and never between the hour and the minute. They are silent to a synthesiser
// either way, so this is about the string being readable to whoever changes it
// next, not about the sound.
func milHourWords(h int) string {
	switch {
	case h == 0:
		return "Zero" // 0000 is "Zero Hundred Hours"
	case h < 10:
		return "Oh " + ones(h)
	}
	return tensWords(h)
}

// minuteWords reads a minute: "Oh Five" under ten, so "Sixteen-Oh Five Hours"
// and "Four Oh Five PM" both land on the beat a listener expects.
func minuteWords(m int) string {
	if m < 10 {
		return "Oh " + ones(m)
	}
	return tensWords(m)
}

// tensWords reads 10..59 — the teens by name, the rest as tens and a unit.
func tensWords(n int) string {
	if n < 20 {
		return teens(n)
	}
	if u := n % 10; u > 0 {
		return tens(n/10) + "-" + ones(u)
	}
	return tens(n / 10)
}

// SpokenID reads an identifier ALOUD — a transmitter callsign, a station code.
//
// Under MILITARY the letters go through the NATO alphabet and the digits are
// read one at a time, with NINER for nine: "KC21F9"
// is "Kilo Charlie Two One Foxtrot Niner". That is the convention the phonetic
// alphabet exists for — a callsign misheard on a weather broadcast sends
// somebody to the wrong frequency — and it is why nine has its own word, since
// over a poor signal it is otherwise "five".
//
// Under the other two clocks the identifier is left alone: a synthesiser reads
// "KEC62" letter by letter well enough, and spelling every callsign out in full
// would make a routine lead read like a drill.
//
// TITLE CASE, not the capitals the convention is written in: a synthesiser that
// sees "KILO" may spell it back as four letters, which is the opposite of the
// point.
func (c Clock) SpokenID(s string) string {
	if c != ClockMil || s == "" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		w := phonetic(r)
		if w == "" {
			continue // punctuation and spaces: a readout has no use for them
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
	}
	if b.Len() == 0 {
		return s
	}
	return b.String()
}

// phonetic is one character's spoken word, "" for anything that is not a letter
// or a digit.
func phonetic(r rune) string {
	switch {
	case r >= 'a' && r <= 'z':
		r -= 'a' - 'A'
	case r >= '0' && r <= '9':
		// NINER for nine, the one digit the convention renames: over a poor
		// signal "nine" and "five" are the same word.
		for i, w := range []string{"Zero", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Niner"} {
			if rune('0'+i) == r {
				return w
			}
		}
		return ""
	case r < 'A' || r > 'Z':
		return ""
	}
	for i, w := range []string{"Alfa", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot", "Golf", "Hotel", "India",
		"Juliett", "Kilo", "Lima", "Mike", "November", "Oscar", "Papa", "Quebec", "Romeo", "Sierra",
		"Tango", "Uniform", "Victor", "Whiskey", "Xray", "Yankee", "Zulu"} {
		if rune('A'+i) == r {
			return w
		}
	}
	return ""
}

// ones, teens and tens are the number words. Switches rather than package-level
// slices, which would be mutable process-wide state (P10-06); they are read a
// handful of times per spoken line.
func ones(n int) string {
	for i, w := range []string{"Zero", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve"} {
		if i == n {
			return w
		}
	}
	return "Zero"
}

func teens(n int) string {
	for i, w := range []string{"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"} {
		if i == n-10 {
			return w
		}
	}
	return "Ten"
}

func tens(n int) string {
	for i, w := range []string{"Zero", "Ten", "Twenty", "Thirty", "Forty", "Fifty"} {
		if i == n {
			return w
		}
	}
	return "Zero"
}

// layout is c's bare-time layout.
func (c Clock) layout() string {
	switch c {
	case Clock24:
		return "15:04"
	case ClockMil:
		return "1504"
	}
	return "3:04 PM"
}
