package app

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheSevereRowsTimesReadInThePlacesZone is 0.19.5 Q5.2's pin (DR-5), written
// before the row's times and the record's share one formatter: a row tied to a
// place gives its declared and expiry times in that place's zone, month, day,
// time and zone name, and a row with no clock reads blank, never 01/01 00:00.
func TestTheSevereRowsTimesReadInThePlacesZone(t *testing.T) {
	at := time.Date(2026, 8, 28, 13, 45, 0, 0, time.UTC)
	tied := &snapshot.LocationRef{Label: "Wichita, KS", TZ: "America/Chicago"}
	row := toSevereRow(severe.Row{Product: "Tornado Warning", Tied: tied, At: at, Until: at.Add(45 * time.Minute)})
	if row.Declared != "08/28 08:45 CDT" || row.Expires != "08/28 09:30 CDT" {
		t.Errorf("a Wichita warning reads declared %q, expires %q; want 08/28 08:45 CDT and 08/28 09:30 CDT", row.Declared, row.Expires)
	}
	if blank := toSevereRow(severe.Row{Product: "Tornado Warning", Tied: tied}); blank.Declared != "" || blank.Expires != "" {
		t.Errorf("a row with no clock reads declared %q, expires %q; want both blank", blank.Declared, blank.Expires)
	}
}

// TestTheAlertWindowIsSpokenAsTheRecordShowsIt is 0.19.5 Q5.3's pin (DR-6, A-4,
// RR-7), written before the window travels as a value: a domain row, through
// toSevereRow and the spoken read, says the window the record's timing line
// shows. Rounded to the minute, days whole (a day and a half is "1 day"),
// measured from the onset when that follows the issue time, and not said at
// all when the alert expires as it starts.
func TestTheAlertWindowIsSpokenAsTheRecordShowsIt(t *testing.T) {
	at := time.Date(2026, 8, 28, 13, 45, 0, 0, time.UTC)
	for _, c := range []struct {
		name         string
		onset, until time.Duration // after the issue time; a negative onset is none
		want         string
	}{
		{"no window", -1, 0, ""},
		{"29 seconds", -1, 29 * time.Second, "In effect for about 0 minutes."},
		{"30 seconds", -1, 30 * time.Second, "In effect for about 1 minute."},
		{"59 minutes 30 seconds", -1, 59*time.Minute + 30*time.Second, "In effect for about 1 hour."},
		{"1 hour 30 minutes", -1, 90 * time.Minute, "In effect for about 1 hour 30 minutes."},
		{"1 day 12 hours", -1, 36 * time.Hour, "In effect for about 1 day."},
		{"an onset after the issue", 2 * time.Hour, 2*time.Hour + 45*time.Minute, "In effect for about 45 minutes."},
	} {
		d := &globalfeed.SevereDetail{Severity: "Extreme", Urgency: "Immediate", Certainty: "Observed"}
		if c.onset >= 0 {
			d.Onset = at.Add(c.onset)
		}
		r := severe.Row{Product: "Tornado Warning", Location: "Olathe, KS", At: at, Until: at.Add(c.until), Detail: severe.Detail{Severe: d}}
		if c.until == 0 {
			r.Until = time.Time{}
		}
		s := eventScript(nil, toSevereRow(r))
		switch {
		case c.want == "" && strings.Contains(s, "In effect for"):
			t.Errorf("%s: the read says a window:\n%s", c.name, s)
		case c.want != "" && !strings.Contains(s, c.want):
			t.Errorf("%s: the read does not say %q:\n%s", c.name, c.want, s)
		}
	}
}
