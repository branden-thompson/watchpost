// Package report renders Snapshots for stdout: --json (machine, schema
// v1.0-rc) and --report-only (line-oriented plain text — also the documented
// screen-reader surface, R-12d/G-9a). It imports ONLY platform packages —
// never a domain (M5 structural rule; lint-enforced).
//
// Exit codes (§10.2): 0 ok · 1 error · 2 partial (any provider degraded).
// obs_stale warnings stay in-band and never trigger exit 2.
package report

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// RenderJSON marshals the snapshot as the machine-mode envelope.
func RenderJSON(s *snapshot.Snapshot) ([]byte, error) {
	if err := invariant.Check(s != nil, "cannot render a nil snapshot"); err != nil {
		return nil, err
	}
	if err := invariant.Check(s.SchemaVersion != "", "snapshot must carry its schema version"); err != nil {
		return nil, err
	}
	return json.MarshalIndent(s, "", "  ")
}

// FormatNum renders a numeric leaf exactly as both surfaces show it (one
// decimal place) — shared by render and the parity tests so M5 compares the
// same formatting on both sides.
func FormatNum(v float64) string { return fmt.Sprintf("%.1f", v) }

// num renders a nullable leaf: nil -> "n/a" (the null-parity rule §10.11).
func num(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return FormatNum(*v)
}

// RenderPlain renders the line-oriented, ANSI-free report (R-12d surface):
// newest-last, alerts prefixed "ALERT [severity]:", width computed once by the
// caller via platform/term.
func RenderPlain(s *snapshot.Snapshot, width int) string {
	if s == nil {
		return "no data\n"
	}
	var b strings.Builder
	line := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
		b.WriteByte('\n')
	}
	rule := strings.Repeat("-", max(20, min(width, 100)))
	line("watchpost report — generated %s (schema %s)", s.GeneratedAt.Format("2006-01-02 15:04 MST"), s.SchemaVersion)
	for _, loc := range s.Locations {
		line("%s", rule)
		label := loc.Label
		if loc.Zip != "" {
			label = fmt.Sprintf("%s (%s)", loc.Label, loc.Zip) // R-2': zip always alongside
		}
		line("%s", label)
		c := loc.Harmonized
		if c.Source.Provider != "" {
			line("  observed %s via %s/%s", c.ObservedAt.Format("15:04 MST"), c.Source.Provider, c.Source.ModelOrStation)
			line("  temp %s C  feels %s C  dewpoint %s C  humidity %s%%", num(c.Temp), num(c.Feels), num(c.Dewpoint), num(c.HumidityPct))
			line("  wind %s m/s dir %s deg  gust %s  pressure %s hPa  visibility %s m",
				num(c.Wind), num(c.WindDirDeg), num(c.WindGust), num(c.Pressure), num(c.Visibility))
			line("  conditions: %s", c.Condition)
		} else {
			line("  no current conditions")
		}
		alertLines(line, loc.Alerts)
		fireLines(line, loc.Fire)
		if len(loc.Hourly) > 0 {
			h := loc.Hourly[0]
			line("  next hour: %s C, precip %s%%, wind %s m/s, %s", num(h.Temp), num(h.PrecipProb), num(h.Wind), h.Condition)
		}
		for _, d := range loc.Daily {
			line("  %s: high %s C low %s C precip %s%% — %s", d.Date, num(d.TempMax), num(d.TempMin), num(d.PrecipProb), d.Condition)
		}
	}
	line("%s", rule)
	for _, p := range s.Providers {
		line("provider %s: %s (%s)", p.ID, p.Status, p.Attribution)
	}
	seen := map[string]bool{}
	for _, w := range s.Warnings {
		if key := string(w.Code) + w.Message; !seen[key] { // one line per distinct warning (F18: a 404 came once per fetch kind)
			seen[key] = true
			line("warning [%s] %s", w.Code, w.Message)
		}
	}
	return b.String()
}

// alertLines is the alert block: severity ALWAYS a text label + fixed
// position (R-12a); "no active alerts" said outright.
func alertLines(line func(string, ...any), alerts []snapshot.Alert) {
	if len(alerts) == 0 {
		line("  no active alerts")
	}
	for _, a := range alerts {
		line("  ALERT [%s]: %s — %s (expires %s)", a.Severity, render.Plain(a.Event), render.Plain(a.Headline), a.Expires.Format("Jan 2 15:04 MST")) // plain output has no cell renderer to drop escapes (S-F6)
		if a.Instruction != "" {
			line("    %s", a.Instruction)
		}
	}
}

// hotspotCount words the count; at the cap it reads "300+" (snapshot.MaxHotspots).
func hotspotCount(n int) string {
	if n >= snapshot.MaxHotspots {
		return fmt.Sprintf("%d+", snapshot.MaxHotspots)
	}
	return fmt.Sprintf("%d", n)
}

// when formats a time the feed may not have given (zero → "n/a", the
// null-parity rule for dates — red-team B5 U2).
func when(t time.Time, layout string) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.Format(layout)
}

// fireLines is the fire block (B5): hotspots inside the ring and the named
// incidents — plain words at a fixed position, "no hotspots" said outright.
func fireLines(line func(string, ...any), fs snapshot.FireState) {
	if fs.AsOf.IsZero() {
		line("  fire: feed unavailable") // no fire provider answered — not the same as "no hotspots" (red-team B5 P3)
		return
	}
	if n := len(fs.Hotspots); n == 0 {
		line("  fire: no hotspots nearby")
	} else {
		h := fs.Hotspots[0]
		line("  fire: %s hotspot(s) nearby — nearest %s km, %s MW, %s at %s", hotspotCount(n), num(h.DistanceKm), num(h.FRPMW), render.Plain(h.Source.Provider+"/"+h.Source.ModelOrStation), when(h.DetectedAt, "Jan 2 15:04 MST"))
	}
	for _, in := range fs.Incidents {
		line("  incident: %s — %s acres, %s%% contained, %s km (%s)", render.Plain(in.Name), num(in.Acres), num(in.PercentContained), num(in.Source.DistanceKm), when(in.Discovered, "Jan 2"))
	}
}

// RenderRequests is the `report --verbose` trailer (quality pass Q0,
// red-team A11-9): one plain line per host with the request counters,
// so the chattiness of a run can be read without a proxy. Machine-friendly
// (raw byte counts), ANSI-free like the rest of the plain surface.
func RenderRequests(st httpx.RequestStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "requests: uptime %s\n", st.Uptime.Round(time.Millisecond))
	if len(st.Hosts) == 0 {
		b.WriteString("requests: none\n")
	}
	for _, h := range st.Hosts {
		fmt.Fprintf(&b, "requests %s: attempts %d net %d cache %d neg %d 304 %d bytes %d h2 %d tls %d\n",
			h.Host, h.Attempts, h.Net, h.Cache, h.Neg, h.NotModified, h.BytesNet, h.H2, h.TLSHandshakes)
	}
	return b.String()
}

// ExitCode maps a snapshot to the process exit code (§10.2): 2 only when a
// provider is degraded; warnings (obs_stale included) stay in-band.
func ExitCode(s *snapshot.Snapshot) int {
	if s == nil {
		return 1
	}
	for _, p := range s.Providers {
		if p.Status == snapshot.ProviderDegraded {
			return 2
		}
	}
	return 0
}
