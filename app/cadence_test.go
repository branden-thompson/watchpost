package app

// Quality pass Q2 (L3-F14): the cadence tables have one owner (pipelines.go)
// and this test renders them as the markdown architecture.md §11.1 carries,
// pinned by testdata/cadences.md. A cadence change fails here first, and the
// doc is copied from the golden — never re-typed.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/sched"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var updateCadences = flag.Bool("update-cadences", false, "re-capture testdata/cadences.md")

func kindName(k snapshot.FetchKind) string {
	return map[snapshot.FetchKind]string{snapshot.KindAlerts: "alerts", snapshot.KindObs: "observations", snapshot.KindMarineObs: "marine observations",
		snapshot.KindForecast: "forecast (daily)", snapshot.KindForecastHourly: "forecast (hourly)", snapshot.KindMarine: "marine forecast", snapshot.KindFire: "fire"}[k]
}

func cadenceTable() string {
	var b strings.Builder
	b.WriteString("| Pipeline | Tier | Cadence |\n|---|---|---|\n")
	row := func(pipe string, tiers []sched.Tier) {
		for _, t := range tiers {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", pipe, kindName(t.Kind), t.Every)
		}
	}
	row("Priority (≤ 10 favourites, one batched scheduler)", priorityTiers())
	row("RECENT (one scheduler per location)", recentTiers())
	fmt.Fprintf(&b, "| RECENT (one batched scheduler for the list) | alerts | %s |\n", recentAlertsEvery)
	fmt.Fprintf(&b, "| RECENT | forecast (hourly) | on demand (Details / lookup) |\n")
	fmt.Fprintf(&b, "\nRehydrate on failure: %s; publish coalescing window: %s.\n", "10 s / 20 s / 40 s (sched)", publishCoalesce)
	return b.String()
}

func TestCadenceTableIsTheDoc(t *testing.T) {
	got := cadenceTable()
	path := filepath.Join("testdata", "cadences.md")
	if *updateCadences {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden yet: run with -update-cadences (%v)", err)
	}
	if got != string(want) {
		t.Fatalf("a cadence changed — record its freshness argument (plan §0.3) and re-run with -update-cadences:\n%s", got)
	}
	_ = time.Second
}
