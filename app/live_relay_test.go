package app

// Live relay proof (quality pass Q1 gate, plan Q1: "tunes a weatherUSA
// mount and a wxradio mount"): with WATCHPOST_LIVE=1 this reads both relay
// directories over the real network, opens one mount from each through the
// player's stream client (same-origin redirect policy included) and reads
// audio bytes from it. Skipped otherwise — it needs the internet and the
// relays' goodwill (one directory read, one short connection each).
//
//	WATCHPOST_LIVE=1 go test ./app -run LiveRelay -v
//
// HUM LEAD runs the same command on Arch for the Linux half of the gate.

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// liveMountTries bounds the proof's search for a mount that plays.
const liveMountTries = 25

func TestLiveRelayMountsPlayOnBothRelays(t *testing.T) {
	if os.Getenv("WATCHPOST_LIVE") != "1" {
		t.Skip("set WATCHPOST_LIVE=1 to read the real relay directories and open one mount per relay")
	}
	c, err := httpx.New(httpx.Config{UserAgent: UserAgent, RatePerSec: 5, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	mounts, statuses := stream.NewDirectory(c, "", "").MountsWithStatus(ctx)
	down := map[string]error{}
	for _, st := range statuses {
		if st.Err != nil {
			down[st.Relay] = st.Err
		}
	}
	byRelay := map[string][]stream.Mount{}
	for _, ms := range mounts {
		for _, m := range ms {
			byRelay[m.Relay] = append(byRelay[m.Relay], m)
		}
	}
	played := 0
	for _, relay := range []string{"wxradio.org", "weatherusa.net"} {
		if err, isDown := down[relay]; isDown {
			// A relay that is not answering from here is reported, not
			// failed: the gate needs each relay proven on a day it is up.
			t.Logf("RELAY DOWN (re-run later): %s — %v", relay, err)
			continue
		}
		if len(byRelay[relay]) == 0 {
			t.Fatalf("%s answered but offered no mounts (%d callsigns in total)", relay, len(mounts))
		}
		// A directory lists sources; one that is not connected right now
		// answers 404 (the app's tune list falls through the same way — on
		// 2026-08-26 weatherUSA advertised 116 and served a minority), so up
		// to liveMountTries mounts are tried before the relay is judged.
		var lastErr error
		for i, m := range byRelay[relay] {
			if i == liveMountTries {
				break
			}
			if err := readMount(ctx, m, t); err != nil {
				lastErr = err
				continue
			}
			lastErr = nil
			break
		}
		if lastErr != nil {
			t.Fatalf("%s: no mount played among the first %d of %d: %v", relay, liveMountTries, len(byRelay[relay]), lastErr)
		}
		played++
	}
	if played == 0 {
		t.Fatal("neither relay could be proven: both directories are down from here")
	}
	if h := c.RequestStats(); len(h.Hosts) == 0 {
		t.Fatal("the directory reads must be counted")
	}
}

// readMount opens one mount and reads 64 KB of audio through the player.
func readMount(ctx context.Context, m stream.Mount, t *testing.T) error {
	s, err := player.Open(ctx, UserAgent, m.URL)
	if err != nil {
		t.Logf("  %s %s: %v", m.Callsign, m.URL, err)
		return err
	}
	n, err := io.ReadFull(s, make([]byte, 64<<10))
	_ = s.Close()
	if err != nil {
		t.Logf("  %s %s: read %d bytes: %v", m.Callsign, m.URL, n, err)
		return err
	}
	t.Logf("PLAYS: %s — %s %s (%s, %d kbps, 64 KB read)", m.Relay, m.Callsign, m.URL, s.Type, s.Bitrate)
	return nil
}
