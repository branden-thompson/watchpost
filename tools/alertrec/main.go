// Command alertrec records live NWS alert-feed states as replay fixtures
// (architecture §10.8): it polls /alerts/active for the given zones on a
// cadence and appends one JSONL feed-state line per poll — replayable by the
// domains/alerts harness (M2/M3).
//
// Usage: alertrec -zones CAZ554,CAC073 -every 20s -out feed.jsonl -for 10m
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "alertrec:", err)
		os.Exit(1)
	}
}

func run() error {
	zones := flag.String("zones", "", "comma-separated UGC zones (required)")
	every := flag.Duration("every", 20*time.Second, "poll cadence")
	out := flag.String("out", "feed.jsonl", "output JSONL path")
	dur := flag.Duration("for", 10*time.Minute, "recording duration")
	flag.Parse()
	if err := invariant.Check(*zones != "", "zones are required (e.g. -zones CAZ554,CAC073)"); err != nil {
		return err
	}
	if err := invariant.Check(*every >= 5*time.Second, "cadence under 5s would abuse the NWS CDN (AI-1)"); err != nil {
		return err
	}
	client, err := httpx.New(httpx.Config{UserAgent: "watchpost-alertrec/0.1 (github.com/branden-thompson/watchpost)"})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(*out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", *out, err)
	}
	defer func() { _ = f.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), *dur)
	defer cancel()
	enc := json.NewEncoder(f)
	deadline := time.Now().Add(*dur)
	// Statically bounded (P10-02): at most dur/every+1 polls, deadline-checked.
	maxPolls := int(*dur / *every) + 1
	for i := 0; i < maxPolls && time.Now().Before(deadline); i++ {
		state, err := poll(ctx, client, *zones)
		if err != nil {
			fmt.Fprintln(os.Stderr, "poll:", err) // keep recording through blips
		} else if err := enc.Encode(state); err != nil {
			return fmt.Errorf("cannot write fixture line: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(*every):
		}
	}
	return nil
}

type feedAlert struct {
	ID       string    `json:"id"`
	Event    string    `json:"event"`
	Severity string    `json:"severity"`
	Sent     time.Time `json:"sent"`
	Expires  time.Time `json:"expires"`
}

type feedState struct {
	At     time.Time   `json:"at"`
	Alerts []feedAlert `json:"alerts"`
}

func poll(ctx context.Context, client *httpx.Client, zones string) (feedState, error) {
	var payload struct {
		Features []struct {
			Properties feedAlert `json:"properties"`
		} `json:"features"`
	}
	u := "https://api.weather.gov/alerts/active?status=actual&zone=" + zones
	if _, err := client.GetJSON(ctx, u, &payload); err != nil {
		return feedState{}, err
	}
	st := feedState{At: time.Now().UTC(), Alerts: []feedAlert{}}
	for _, f := range payload.Features {
		st.Alerts = append(st.Alerts, f.Properties)
	}
	return st, nil
}
