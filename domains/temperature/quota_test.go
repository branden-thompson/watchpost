package temperature

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

var quotaNow = time.Date(2026, 9, 30, 15, 12, 0, 0, time.UTC)

func spent(period string) error {
	return &httpx.StatusError{URL: "https://api.open-meteo.com/v1/forecast", Status: http.StatusTooManyRequests, Attempts: 2, Degraded: true,
		Reason: `{"error":true,"reason":"` + period + ` API request limit exceeded. Please try again tomorrow."}`}
}

// A SPENT QUOTA IS NAMED BY ITS PERIOD (W18.1, D-165): Open-Meteo's 429 says
// which limit was spent, and each resets when its period turns - Open-Meteo
// publishes no reset time and sends no Retry-After, so the period's end is
// the best said, and the gate's probe finds the true one.
func TestASpentQuotaIsNamedByItsPeriod(t *testing.T) {
	for _, tc := range []struct {
		period string
		resets time.Time
	}{
		{"Minutely", time.Date(2026, 9, 30, 15, 13, 0, 0, time.UTC)},
		{"Hourly", time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)},
		{"Daily", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{"Monthly", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
	} {
		q, ok := QuotaOf(spent(tc.period), quotaNow)
		if !ok || q.Period != tc.period || !q.Resets.Equal(tc.resets) || q.Host != "api.open-meteo.com" {
			t.Errorf("%s: %+v, %v; want the %s quota of api.open-meteo.com, reset %v", tc.period, q, ok, tc.period, tc.resets)
		}
	}
	for _, err := range []error{
		&httpx.StatusError{URL: "https://api.open-meteo.com/v1/forecast", Status: 500, Reason: "Daily API request limit exceeded"},
		&httpx.StatusError{URL: "https://api.open-meteo.com/v1/forecast", Status: 429, Reason: "slow down"},
		errors.New("Daily API request limit exceeded"),
		nil,
	} {
		if q, ok := QuotaOf(err, quotaNow); ok {
			t.Errorf("%v read as a spent quota: %+v", err, q)
		}
	}
}

// gateGetter is a Getter that answers from a function and counts its asks.
type gateGetter struct {
	asked  []string
	answer func(url string) ([]byte, error)
}

func (g *gateGetter) GetText(_ context.Context, url string, _ ...httpx.Option) ([]byte, error) {
	g.asked = append(g.asked, url)
	return g.answer(url)
}

// A SPENT HOST IS NOT ASKED AGAIN UNTIL THE GATE PROBES IT (W18.1): every ask
// of the map would otherwise go out and come back refused - each counted
// against the quota and each waiting out the client's retries. A refused host
// is held; about once a probe interval one ask goes through, and an answer
// frees it. Another host is never held for it.
func TestTheGateHoldsASpentHostUntilItsProbe(t *testing.T) {
	now := quotaNow
	refuse := true
	under := &gateGetter{answer: func(url string) ([]byte, error) {
		if refuse && url != "https://air-quality-api.open-meteo.com/v1/air-quality" {
			return nil, spent("Daily")
		}
		return []byte("ok"), nil
	}}
	g := NewQuotaGate(under, func() time.Time { return now })
	forecast := "https://api.open-meteo.com/v1/forecast"
	if _, err := g.GetText(context.Background(), forecast); err == nil {
		t.Fatal("the refusal was not passed on")
	}
	q, ok := g.Refused()
	if !ok || q.Period != "Daily" {
		t.Fatalf("the gate does not say the quota is spent: %+v, %v", q, ok)
	}
	_, err := g.GetText(context.Background(), forecast)
	if len(under.asked) != 1 {
		t.Errorf("a held host was asked again: %v", under.asked)
	}
	if held, ok := QuotaOf(err, now); !ok || held.Period != "Daily" {
		t.Errorf("a held ask's error is not the spent quota: %v", err)
	}
	if _, err := g.GetText(context.Background(), "https://air-quality-api.open-meteo.com/v1/air-quality"); err != nil || len(under.asked) != 2 {
		t.Errorf("another host was held for it: %v, %v", err, under.asked)
	}
	now = now.Add(quotaProbeEvery)
	refuse = false
	if body, err := g.GetText(context.Background(), forecast); err != nil || string(body) != "ok" || len(under.asked) != 3 {
		t.Fatalf("the probe did not go through and free the host: %q, %v, %v", body, err, under.asked)
	}
	if _, ok := g.Refused(); ok {
		t.Error("an answered probe left the quota said spent")
	}
}

// THE QUOTA IS THE MACHINE'S, NOT A PROCESS'S (W18.3, design 4b): gates
// sharing one state file - one per watchpost instance - hold together. A
// refusal met by one holds the other without its asking; when the probe is
// due, one of them asks; an answer frees both.
func TestInstancesShareTheQuotasHold(t *testing.T) {
	now := quotaNow
	clock := func() time.Time { return now }
	state := filepath.Join(t.TempDir(), "quota.json")
	refuse := true
	answer := func(string) ([]byte, error) {
		if refuse {
			return nil, spent("Daily")
		}
		return []byte("ok"), nil
	}
	ua, ub := &gateGetter{answer: answer}, &gateGetter{answer: answer}
	a, b := NewSharedQuotaGate(ua, clock, state), NewSharedQuotaGate(ub, clock, state)
	forecast := "https://api.open-meteo.com/v1/forecast"
	if _, err := a.GetText(context.Background(), forecast); err == nil {
		t.Fatal("the refusal was not passed on")
	}
	now = now.Add(quotaStateEvery) // past b's read of the state
	if _, err := b.GetText(context.Background(), forecast); err == nil || len(ub.asked) != 0 {
		t.Fatalf("the other instance asked a host the first found spent: %v, %v", err, ub.asked)
	}
	if q, ok := b.Refused(); !ok || q.Period != "Daily" {
		t.Errorf("the other instance does not say the quota is spent: %+v", q)
	}
	now = now.Add(quotaProbeEvery) // the probe is due - and refused again
	_, _ = a.GetText(context.Background(), forecast)
	_, _ = b.GetText(context.Background(), forecast)
	if probes := len(ua.asked) + len(ub.asked) - 1; probes != 1 {
		t.Errorf("%d instances probed the host; want one", probes)
	}
	now = now.Add(quotaProbeEvery) // due again; answered this time
	refuse = false
	if _, err := b.GetText(context.Background(), forecast); err != nil {
		t.Fatalf("the due probe was not asked: %v", err)
	}
	now = now.Add(quotaStateEvery) // not a's own probe: the other's answer frees it
	if _, ok := a.Refused(); ok {
		t.Error("after the other instance's answered probe, this one still says the quota is spent")
	}
	if _, err := a.GetText(context.Background(), forecast); err != nil {
		t.Errorf("after the other instance's answered probe, this one is still held: %v", err)
	}
}

// A PROBE ANOTHER INSTANCE HAS CLAIMED IS NOT ASKED AGAIN, even by one whose
// own reading says it is due: before claiming, a gate reads the state, and a
// probe already moved forward is another's.
func TestAClaimedProbeIsNotAskedTwice(t *testing.T) {
	now := quotaNow
	clock := func() time.Time { return now }
	state := filepath.Join(t.TempDir(), "quota.json")
	forecast := "https://api.open-meteo.com/v1/forecast"
	var b *QuotaGate
	var bErr error
	ub := &gateGetter{answer: func(string) ([]byte, error) { return nil, spent("Daily") }}
	ua := &gateGetter{}
	ua.answer = func(string) ([]byte, error) {
		if len(ua.asked) > 1 { // a's probe, in flight: b tries now
			_, bErr = b.GetText(context.Background(), forecast)
		}
		return nil, spent("Daily")
	}
	a := NewSharedQuotaGate(ua, clock, state)
	b = NewSharedQuotaGate(ub, clock, state)
	_, _ = a.GetText(context.Background(), forecast) // refused: held in the state
	now = now.Add(quotaProbeEvery - time.Second)
	_, _ = b.GetText(context.Background(), forecast) // b reads the hold, a second before its probe
	now = now.Add(time.Second)                       // due, by both gates' readings
	_, _ = a.GetText(context.Background(), forecast) // a claims the probe, and b tries during it
	if len(ub.asked) != 0 {
		t.Errorf("b asked a probe a had claimed (%v)", bErr)
	}
}
