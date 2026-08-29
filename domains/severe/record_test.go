package severe

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var chicago, _ = time.LoadLocation("America/Chicago")

func TestRecordOfWarningIsTheAlertShape(t *testing.T) {
	sent := time.Date(2026, 8, 28, 13, 45, 0, 0, time.UTC)
	ends := sent.Add(15 * time.Minute)
	r := Row{Key: "k", Tab: TabWarnings, Product: "Tornado Warning", At: sent, Until: ends, Detail: Detail{Alert: &snapshot.Alert{
		Event: "Tornado Warning", Severity: "extreme", Urgency: "immediate", Certainty: "observed", AreaDesc: "Johnson County, KS",
		SenderName: "NWS Kansas City", Description: "At 845 AM CDT, a severe thunderstorm…\n\n* HAZARD...Damaging tornado.", Instruction: "TAKE COVER NOW!",
	}}}
	rec := RecordOf(r, chicago)
	if rec.Title != "TORNADO WARNING" || rec.Meta != "[Extreme · Immediate · Observed]" {
		t.Fatalf("title/meta: %q %q", rec.Title, rec.Meta)
	}
	if rec.Timing != "Declared 08/28 08:45 CDT   Expires 08/28 09:00 CDT   (~15m)" {
		t.Fatalf("timing: %q", rec.Timing)
	}
	if rec.Area != "Area: Johnson County, KS · NWS Kansas City" || len(rec.Paras) != 3 || !strings.HasPrefix(rec.Paras[2], "Instructions: ") {
		t.Fatalf("area/paras: %q %v", rec.Area, rec.Paras)
	}
	// The national record's CAP parameters ride along when both paths saw the alert.
	r.Detail.Severe = &globalfeed.SevereDetail{MaxWindGust: "60 mph", MaxHailSize: "1.00 in"}
	if rec := RecordOf(r, chicago); rec.Paras[0] != "Wind gusts to 60 mph · Hail to 1.00 in" {
		t.Fatalf("cap extras: %v", rec.Paras)
	}
}

func TestShortDur(t *testing.T) {
	for d, want := range map[time.Duration]string{15 * time.Minute: "15m", 2 * time.Hour: "2h", 90 * time.Minute: "1h30m", 3 * 24 * time.Hour: "3d"} {
		if got := shortDur(d); got != want {
			t.Errorf("%v → %q, want %q", d, got, want)
		}
	}
}

func TestRecordOfQuakeAndStorm(t *testing.T) {
	mag := 5.8
	cdi, mmi := 5.7, 4.624
	q := Row{Tab: TabQuakes, Product: "Earthquake", At: time.Date(2026, 8, 28, 10, 12, 0, 0, time.UTC), Location: "Kodāri, Nepal", Lat: 27.94, Lon: 85.62, HasPoint: true,
		Detail: Detail{Quake: &globalfeed.QuakeDetail{Mag: &mag, MagType: "mww", DepthKm: 61, Alert: "green", Felt: 153, CDI: &cdi, MMI: &mmi, Sig: 651, Status: "reviewed"}}}
	rec := RecordOf(q, time.UTC)
	if rec.Title != "M 5.8 EARTHQUAKE" || !strings.Contains(rec.Meta, "Depth 61 km") || !strings.Contains(rec.Area, "27.94 N, 85.62 E") {
		t.Fatalf("quake record: %+v", rec)
	}
	s := Row{Tab: TabTropical, Product: "Tropical Storm", Name: "Dolly", At: time.Date(2026, 8, 28, 16, 0, 0, 0, time.UTC),
		Detail: Detail{Tropical: &globalfeed.TropicalDetail{Name: "Dolly", Basin: "the Atlantic", BinNumber: "AT4", WindKt: 45, PressureMb: 999, MoveDirDeg: 280, MoveSpeedKt: 25, LatText: "15.0N", LonText: "46.9W", AdvisoryNum: "5"}}}
	rec = RecordOf(s, time.UTC)
	if rec.Title != "TROPICAL STORM DOLLY (AT4)" || !strings.Contains(rec.Meta, "Winds 45 kt") || !strings.Contains(rec.Meta, "Moving W at 25 kt") {
		t.Fatalf("storm record: %+v", rec)
	}
}

func TestRecordOfStripsTerminalEscapesFromEveryField(t *testing.T) {
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[31mz"
	r := Row{Tab: TabWarnings, Product: evil, Location: evil, Detail: Detail{Alert: &snapshot.Alert{Event: evil, Severity: evil, Urgency: evil, Certainty: evil, AreaDesc: evil, SenderName: evil, Headline: evil, Description: evil, Instruction: evil}}}
	for _, rec := range []Record{RecordOf(r, time.UTC), RecordOf(Row{Tab: TabWarnings, Product: evil, Location: evil, Detail: Detail{Severe: &globalfeed.SevereDetail{Severity: evil, SenderName: evil, Description: evil, MaxWindGust: evil}}}, time.UTC)} {
		for _, s := range append([]string{rec.Title, rec.Meta, rec.Timing, rec.Area}, rec.Paras...) {
			if strings.ContainsAny(s, "\x1b\x07") {
				t.Fatalf("escape survived: %q", s)
			}
		}
	}
}
