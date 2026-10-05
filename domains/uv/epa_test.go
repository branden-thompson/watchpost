package uv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// fileGet answers every ask with one fixture, and records the addresses.
type fileGet struct {
	t     *testing.T
	name  string
	asked []string
	fail  bool
}

func (f *fileGet) GetText(_ context.Context, rawURL string, _ ...httpx.Option) ([]byte, error) {
	f.asked = append(f.asked, rawURL)
	if f.fail {
		return nil, errors.New("no answer")
	}
	b, err := os.ReadFile(filepath.Join("testdata", f.name))
	if err != nil {
		f.t.Fatal(err)
	}
	return b, nil
}

// EPA'S UV INDEX, HOUR BY HOUR, IN THE CITY'S OWN TIME (W18.4, D-167): the
// answer labels each hour in the city's local time and says no zone, so it
// is read in the zone the city is in. Recorded for Vista, CA, 2026-09-30.
func TestEPAsHoursAreReadInTheCitysZone(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	get := &fileGet{t: t, name: "epa-vista.json"}
	got, err := NewEPA(get, "").Hourly(context.Background(), "Vista", "CA", la)
	if err != nil {
		t.Fatal(err)
	}
	if len(get.asked) != 1 || !strings.HasPrefix(get.asked[0], "https://data.epa.gov/efservice/getEnvirofactsUVHOURLY/CITY/VISTA/STATE/CA/JSON") {
		t.Errorf("asked %v", get.asked)
	}
	if len(got) != 21 {
		t.Fatalf("%d hours; want the answer's 21", len(got))
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)
	if v, ok := At(got, noon.Add(20*time.Minute)); !ok || v != 7 {
		t.Errorf("at 12:20 PM Pacific the UV is %v (%v); want the noon hour's 7", v, ok)
	}
	if v, ok := At(got, time.Date(2026, 9, 30, 4, 30, 0, 0, la)); !ok || v != 0 {
		t.Errorf("at 4:30 AM the UV is %v (%v); want 0", v, ok)
	}
	if _, ok := At(got, time.Date(2026, 10, 1, 12, 0, 0, 0, la)); ok {
		t.Error("tomorrow was answered from today's hours")
	}
}

// A BODY THAT IS NOT EPA'S ANSWER IS REFUSED, NOT GUESSED AT.
func TestABadAnswerIsRefused(t *testing.T) {
	for _, body := range []string{"not json", `[{"DATE_TIME":"yesterday","UV_VALUE":3}]`, `[]`,
		`[{"DATE_TIME":"Sep/30/2026 12 PM","UV_VALUE":-9999}]`, `[{"DATE_TIME":"Sep/30/2026 12 PM","UV_VALUE":99}]`,
		`[{"DATE_TIME":"Sep/30/2026 01 PM","UV_VALUE":5},{"DATE_TIME":"Oct/02/2026 12 PM","UV_VALUE":7}]`,
		`[{"DATE_TIME":"Sep/30/2026 01 PM","UV_VALUE":5},{"DATE_TIME":"Sep/28/2026 12 PM","UV_VALUE":7}]`} {
		e := NewEPA(&bodyGet{body: body}, "")
		if _, err := e.Hourly(context.Background(), "Vista", "CA", time.UTC); err == nil {
			t.Errorf("%q was read as an answer", body)
		}
	}
	if _, err := NewEPA(&fileGet{t: t, fail: true}, "").Hourly(context.Background(), "Vista", "CA", time.UTC); err == nil {
		t.Error("a failed ask was read as an answer")
	}
}

type bodyGet struct{ body string }

func (b *bodyGet) GetText(context.Context, string, ...httpx.Option) ([]byte, error) {
	return []byte(b.body), nil
}

// NEW YORK CITY IS ASKED AS EPA NAMES IT (UAT-2 U2-51): GeoNames calls it
// "New York City", and EPA answers that name with an error, so the largest
// city in the lower 48 would draw no UV. EPA's name for it is "New York".
func TestNewYorkCityIsAskedAsEPANamesIt(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no zone data")
	}
	get := &fileGet{t: t, name: "epa-vista.json"}
	if _, err := NewEPA(get, "").Hourly(context.Background(), "New York City", "NY", ny); err != nil {
		t.Fatal(err)
	}
	if len(get.asked) != 1 || !strings.Contains(get.asked[0], "/CITY/NEW%20YORK/STATE/NY/") {
		t.Errorf("asked %v; want EPA's name, NEW YORK", get.asked)
	}
}

// A STATE IS TWO LETTERS (IS-M7): it goes into the address's path, so
// anything else - a slash, a dot-dot, a query - is refused before any ask.
func TestAStateThatIsNotTwoLettersIsRefused(t *testing.T) {
	for _, state := range []string{"/.", "..", "C/", "C?", "C%", "C ", "1A", "É1", "CAL", "c"} {
		get := &fileGet{t: t, name: "epa-vista.json"}
		if _, err := NewEPA(get, "").Hourly(context.Background(), "Vista", state, time.UTC); err == nil || len(get.asked) != 0 {
			t.Errorf("state %q: asked %v (%v); want refused before asking", state, get.asked, err)
		}
	}
	get := &fileGet{t: t, name: "epa-vista.json"}
	if _, err := NewEPA(get, "").Hourly(context.Background(), "Vista", "ca", time.UTC); err != nil || len(get.asked) != 1 || !strings.Contains(get.asked[0], "/STATE/CA/") {
		t.Errorf("a lower-case state was not asked as its letters: %v, %v", get.asked, err)
	}
}

// TestMoreHoursThanADayIsRefused: an answer of more hours than a day's,
// twice over, is no day's forecast.
func TestMoreHoursThanADayIsRefused(t *testing.T) {
	rows := make([]string, maxRows+1)
	for i := range rows {
		rows[i] = `{"DATE_TIME":"Sep/30/2026 04 AM","UV_VALUE":1}`
	}
	e := NewEPA(&bodyGet{body: "[" + strings.Join(rows, ",") + "]"}, "")
	if _, err := e.Hourly(context.Background(), "Austin", "TX", time.UTC); err == nil {
		t.Errorf("%d hours for one day were read", len(rows))
	}
}
