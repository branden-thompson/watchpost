package temperature

import (
	"context"
	"errors"
	"net/http"
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
