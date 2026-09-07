# P1 — foundation (multi-voice-support, 0.14.0)

> **PRIOR ART — not the plan.** A verbatim snapshot of this batch document *before* the PLAN artefacts were
> stripped to task shape (`AP-PLANCODE-01`, `../../06-key_learnings/retro-notes.md` RN-1). The artefact of
> record is `../p1-foundation.md`. Read `README.md` in this folder first: it says which of these blocks were actually
> compiled and run, and lists the known defects (D-1 … D-9) not to copy forward.



```
Goal:         The seams every later batch builds on: a bounded Say (FR-12), Compose(Reports{}) (RS-12), the
              cast package (registry · inheritance · resolution · validation · tone classifier), the config
              pairs/tones/knob with load-time validation and a save that preserves unknown keys (FR-8, NFR-5).
Architecture: plan.md §2.1–2.2, §2.5; data-shape.md §1–4. Nothing in P1 changes what the radio says.
Tech Stack:   Go 1.27 · go-toml v2 · testing (table-driven) · the recVoice fake (synth_test.go)
Branch:       feature/multi-voice-support
Gate:         go test ./domains/radio/... ./platform/config ./app; make verify; make p10 0/0
```

Every task: exact file · RED test first · complete code · verify command. Order is dependency order.
Tasks 1.1–1.2 touch `synth` only; 1.3–1.7 build `cast`; 1.8–1.11 the config; 1.12–1.13 wire `app`.

## File map

```
MODIFY: scripts/quality/p10-unmatched.sh        — Task 1.0: dormancy for a Go-free diff; rename-aware symbol scope (POSIX)
CREATE: scripts/quality/p10-unmatched_test.sh   — the three fixture cases (docs-only, rename unchanged, rename changed)
CREATE: domains/radio/synth/limit.go            — Limiter, Limited(Voice), WithPriority, the per-Say bound
CREATE: domains/radio/synth/limit_test.go
MODIFY: domains/radio/synth/compose.go          — Reports struct; Compose(loc, products, now, imperial, station, Reports)
MODIFY: domains/radio/synth/synth_test.go, seismic_test.go, soak_test.go — the Compose call sites
MODIFY: app/radio.go                            — segments() builds Reports
CREATE: domains/radio/cast/cast.go              — Role, Parent, Class, Pair, Tones, Config
CREATE: domains/radio/cast/resolve.go           — Host, Link, Resolution, Resolve
CREATE: domains/radio/cast/validate.go          — Problem, Validate
CREATE: domains/radio/cast/tone.go              — Classify, ToneName
CREATE: domains/radio/cast/cast_test.go         — registry (incl. out-of-range), the fallback matrix, WantsInstall, validate, classify
MODIFY: platform/config/config.go               — RoleVoice, Voices, Tones, Radio fields, Radio.Validate, Save preserves unknown keys
CREATE: platform/config/keep.go                 — unknownKeys (go-toml strict decode) + keepUnknown (flat copy)
MODIFY: platform/config/config_test.go          — round-trip, validate, unknown-key, cross-version fixtures
CREATE: platform/config/testdata/{0.13.0-only,0.14.0-both-os,0.14.0-macos-only,0.14.0-piper-only,hostile-name,quoted-escape-key,unknown-table,scalar-vs-table,unknown-key,wrong-type,bad-tone}.toml
CREATE: app/cast.go                             — castConfig(cfg) cast.Config; radioDeck as cast.Host
CREATE: app/cast_test.go
```

---

### Task 1.0 — the P10 gate's scope helper: dormancy for a Go-free diff, and rename awareness

**Why first:** every later batch's gate runs `make p10`, whose second step is `scripts/quality/p10-unmatched.sh`.
Two defects in that helper make the gate fail for reasons unrelated to the code under review, so it is fixed
before the first gate runs, not by the batch that trips it.

**Defect 1 — a Go-free diff is treated as "no base".** The helper scopes ledger rows to the Go files in the diff
(`-- '*.go'`, line 31) — P10 governs compiled code only, and the tool agrees (a docs-only diff yields
`"findings": null`). But its `in_scope` fallback conflates *no base resolved* with *base resolved, no Go file
changed*: both leave `changed` empty, and the empty case returns "in scope". On a docs-only commit every ledger
row is then in scope, matches nothing, and the helper exits 1 (reproduced on this branch: three unmatched rows,
exit 1, against a run with zero findings). **Contract:** a resolved base with no Go files changed makes every row
**dormant** (exit 0, counted); only an unresolvable base keeps the pre-FULL-GIT "everything in scope" behaviour.
The shape is one flag — `base_known` set where the base resolves — and two guards at the top of `in_scope`.

**Defect 2 — a renamed file's rows read as unmatched.** Git reports `app/narrate.go → app/director.go` as `R099`
and shows only the changed hunks, so P2 Task 2.6's re-keyed `admit`/`awaitAir` rows match no finding while their
new path *is* in the diff → unmatched → exit 1 (machine-confirmed by two lenses on a fixture rename).
**Contract:** for a path git reports as a rename target, the symbol test runs against the diff of **both** paths
(`-M`, `-U0`, old and new in the pathspec); a symbol whose hunks are unchanged is dormant, not unmatched. The
rename map comes from `git diff -M --name-status "$base"`. **Constraint:** the script's shebang is `#!/bin/sh`
and it runs on macOS (bash 3.2) and Linux (dash) — POSIX only: no associative arrays, no `[[`, no `local`, no
process substitution. An `awk` lookup over a two-column `new⇥old` list is the intended shape.

**Tests (the fixture, not the tree):** a temporary git repo under the test's dir with a base commit, then three
cases — (a) a docs-only commit → every row dormant, exit 0; (b) a rename with the symbol's body unchanged and
the row re-keyed to the new path → dormant, exit 0; (c) a rename where the symbol's body *did* change and no
finding matched → unmatched, exit 1. Shell-level, driven from `scripts/quality/` or a small `_test.sh`; name the
runner in the build log.

**Verify:** the three fixture cases; then `A2DH=<framework build> make p10` on this (docs-only) branch → exit 0,
"every in-scope ledger entry matched a finding (N dormant …)".

**Ledger:** no rows change here — the helper's scoping is what changes.

---

### Task 1.1 — `synth.Limiter`: a bounded, prioritised `Say` (FR-12)

**File:** `domains/radio/synth/limit_test.go` (RED)

```go
package synth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowVoice blocks in Say until released; it signals every entry on
// `entered` (so tests wait on a channel, never a timing poll) and records
// the most concurrent callers it saw.
type slowVoice struct {
	inside  atomic.Int32
	maxSeen atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func newSlowVoice() *slowVoice {
	return &slowVoice{entered: make(chan struct{}, 16), release: make(chan struct{})}
}

func (v *slowVoice) Name() string { return "slow" }
func (v *slowVoice) Rate() int    { return 22050 }
func (v *slowVoice) Say(ctx context.Context, _ string) ([]byte, error) {
	n := v.inside.Add(1)
	for {
		m := v.maxSeen.Load()
		if n <= m || v.maxSeen.CompareAndSwap(m, n) {
			break
		}
	}
	v.entered <- struct{}{}
	defer v.inside.Add(-1)
	select {
	case <-v.release:
		return []byte{0, 0}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// awaitEntries waits for n Say calls to be inside the voice.
func awaitEntries(t *testing.T, v *slowVoice, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-v.entered:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d renders entered Say", i, n)
		}
	}
}

func TestLimitedSayQueuesBeyondTheOrdinarySlots(t *testing.T) {
	v := newSlowVoice()
	lim := NewLimiter(2, 1)
	lv := Limited(v, lim)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = lv.Say(context.Background(), "x") }()
	}
	awaitEntries(t, v, 2)
	select { // a third entry would be an over-admission: none may arrive
	case <-v.entered:
		t.Fatal("a third ordinary render entered Say past the two slots")
	case <-time.After(50 * time.Millisecond):
	}
	if got := v.maxSeen.Load(); got != 2 {
		t.Fatalf("ordinary renders inside Say = %d, want the 2 ordinary slots", got)
	}
	close(v.release)
	wg.Wait()
}

func TestPrioritySayTakesTheReservedSlotWhileOrdinaryIsFull(t *testing.T) {
	v := newSlowVoice()
	lim := NewLimiter(1, 1)
	lv := Limited(v, lim)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _, _ = lv.Say(context.Background(), "ordinary") }()
	awaitEntries(t, v, 1)
	done := make(chan struct{})
	go func() { _, _ = lv.Say(WithPriority(context.Background()), "takeover"); close(done) }()
	awaitEntries(t, v, 1) // the takeover entered on the reserved slot while the ordinary pool is full
	if v.inside.Load() != 2 {
		t.Fatal("a priority Say must not queue behind a full ordinary pool")
	}
	close(v.release)
	<-done
	wg.Wait()
}

func TestLimitedSayIsBoundedInTime(t *testing.T) {
	v := newSlowVoice() // never released
	lim := NewLimiter(1, 1)
	lim.Bound = 30 * time.Millisecond
	lv := Limited(v, lim)
	start := time.Now()
	if _, err := lv.Say(context.Background(), "x"); err == nil {
		t.Fatal("a wedged Say must return an error at the bound")
	}
	if time.Since(start) > time.Second {
		t.Fatal("the bound did not fire")
	}
}

func TestLimitedKeepsNameAndRate(t *testing.T) {
	lv := Limited(newSlowVoice(), NewLimiter(1, 1))
	if lv.Name() != "slow" || lv.Rate() != 22050 {
		t.Fatal("Limited must be transparent for Name and Rate")
	}
}

func TestLimitedFailsClosedOnBadWiring(t *testing.T) {
	if _, err := Limited(newSlowVoice(), nil).Say(context.Background(), "x"); err == nil {
		t.Fatal("a nil limiter must not yield an uncapped voice")
	}
}

func TestLimitedBoundCoversTheWaitForASlot(t *testing.T) {
	v := newSlowVoice() // never released: the one slot stays held
	lim := NewLimiter(1, 0)
	lim.Bound = 30 * time.Millisecond
	lv := Limited(v, lim)
	go func() { _, _ = lv.Say(context.Background(), "holder") }()
	awaitEntries(t, v, 1)
	start := time.Now()
	if _, err := lv.Say(context.Background(), "queued"); err == nil || time.Since(start) > time.Second {
		t.Fatalf("a queued Say gives up at the bound: %v after %s", err, time.Since(start))
	}
}
```

**File:** `domains/radio/synth/limit.go` (GREEN)

```go
package synth

import (
	"context"
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Limiter bounds how many Say calls run at once (FR-12 — the code had no cap:
// one broadcast alone reached three concurrent renders and the process six;
// on Linux each is a ~63 MB model load). Ordinary renders share `ordinary`
// slots; a Say whose context carries WithPriority — the Station Director's
// jobs: a takeover or a severe read, never the broadcast's render-ahead —
// may take a reserved slot when the ordinary pool is full, so a line the
// listener is waiting for never queues behind render-ahead. There are TWO
// reserved slots: the job on the air and the job it suspended (a read whose
// render is in flight when a takeover is admitted) — one would let the
// suspended read's render block the takeover. Every Say
// is also bounded in time (a wedged process never holds a slot for the
// session). One Limiter per process, owned by the deck.
type Limiter struct {
	ordinary chan struct{}
	reserved chan struct{}
	Bound    time.Duration // per-Say ceiling; 0 = DefaultBound
}

// DefaultBound is the per-Say ceiling: generous for a ~10 s Piper model load
// under contention, far below "the session".
const DefaultBound = 90 * time.Second

// NewLimiter makes a Limiter with n ordinary slots and r reserved ones.
func NewLimiter(n, r int) *Limiter {
	if n < 1 {
		n = 1
	}
	if r < 0 {
		r = 0
	}
	return &Limiter{ordinary: make(chan struct{}, n), reserved: make(chan struct{}, r)}
}

type priorityKey struct{}

// WithPriority marks a context as a Station Director job's: its Say may use
// the reserved slot. The Director sets it for every job it runs (a takeover
// outranks a read inside the Director; both outrank render-ahead here).
func WithPriority(ctx context.Context) context.Context {
	return context.WithValue(ctx, priorityKey{}, true)
}

// Priority reports whether ctx carries WithPriority.
func Priority(ctx context.Context) bool { v, _ := ctx.Value(priorityKey{}).(bool); return v }

// acquire takes a slot: an ordinary one when free; the reserved one for a
// priority context; else waits for an ordinary slot or ctx.
func (l *Limiter) acquire(ctx context.Context) (release func(), err error) {
	select {
	case l.ordinary <- struct{}{}:
		return func() { <-l.ordinary }, nil
	default:
	}
	if Priority(ctx) && cap(l.reserved) > 0 {
		select {
		case l.reserved <- struct{}{}:
			return func() { <-l.reserved }, nil
		default:
		}
	}
	select {
	case l.ordinary <- struct{}{}:
		return func() { <-l.ordinary }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// limited is a Voice whose Say is admitted by a Limiter and bounded in time.
type limited struct {
	Voice
	l *Limiter
}

// Limited wraps v so every Say is admitted by l (FR-12). Name and Rate pass
// through untouched. A nil voice or limiter is a wiring bug: the result
// FAILS CLOSED — its Say returns the invariant error — never an uncapped
// voice (a silent loss of the cap is the failure this exists to prevent).
func Limited(v Voice, l *Limiter) Voice {
	if err := invariant.Check(v != nil && l != nil, "synth: Limited needs a voice and a limiter"); err != nil {
		return deadVoice{err: err}
	}
	return limited{Voice: v, l: l}
}

// ToneRate is the tone's constant sample rate (P2 Task 2.0 moves it to tone.go).
const ToneRate = 22050

// deadVoice is the fail-closed result of a bad Limited call (synth_test.go
// already has a deadVoice fake — a different thing, hence the name).
type deadVoice struct{ err error }

func (deadVoice) Name() string                                   { return "" }
func (deadVoice) Rate() int                                      { return ToneRate }
func (b deadVoice) Say(context.Context, string) ([]byte, error) { return nil, b.err }

// Say implements Voice: the bound covers the WAIT for a slot as well as the
// render (a Say queued behind wedged renders must not wait N × bound).
func (v limited) Say(ctx context.Context, text string) ([]byte, error) {
	bound := v.l.Bound
	if bound <= 0 {
		bound = DefaultBound
	}
	ctx, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	release, err := v.l.acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("synth: waiting for a render slot: %w", err)
	}
	defer release()
	out, err := v.Voice.Say(ctx, text)
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("synth: %s did not render within %s", v.Voice.Name(), bound)
	}
	return out, err
}
```

(P2 Task 2.0 moves `ToneRate` to `tone.go` and deletes it here. Run `gofmt -w` on every file this batch touches
before the gate — `make verify` starts with `fmt`.)
**Verify:** `go test ./domains/radio/synth -run 'Limited|Priority' -race -count=3`

---

### Task 1.2 — `Compose(…, Reports{})` (RS-12; the seam P2 and P3 both need)

**File:** `domains/radio/synth/synth_test.go` (RED — change the call sites; the assertions stay)

Replace each `std.Compose(loc, products, now, true, "Samantha", Station{...}, FIRE, SEISMIC)` with
`std.Compose(loc, products, now, true, Station{...}, Reports{Voice: "Samantha", Fire: FIRE, Seismic: SEISMIC})`:

```go
// line 31
segs := std.Compose(loc, []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear. Lows 66 to 69.\n\n$$"}}, now, true, Station{Callsign: "KEC62", Site: "San Diego", State: "CA", FreqMHz: "162.400"}, Reports{Voice: "Samantha"})
// line 569
segs := std.Compose(snapshot.Location{Label: "Oceanside, CA"}, nil, time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), true, Station{}, Reports{Voice: "Samantha"})
// line 582
segs := std.Compose(loc, products, now, true, Station{}, Reports{Voice: "Samantha", Fire: fire})
// line 599
segs = std.Compose(loc, products, now, true, Station{}, Reports{Voice: "Samantha"})
```

`seismic_test.go:49` → `std.Compose(snapshot.Location{Label: "Ridgecrest, CA"}, nil, now, true, Station{}, Reports{Voice: "rec", Seismic: sr})`
`soak_test.go:40` → `std.Compose(loc, nil, now, true, Station{}, Reports{Voice: "rec", Seismic: sr})`

Add to `synth_test.go`:

```go
func TestComposeReportsZeroValueIsTodaysBroadcast(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	loc := snapshot.Location{Label: "Oceanside, CA"}
	segs := std.Compose(loc, nil, now, true, Station{}, Reports{})
	if len(segs) != 3 || segs[0].Key != "lead:Oceanside, CA" || !strings.HasPrefix(segs[2].Key, "tail") {
		t.Fatalf("zero Reports = lead, span, tail: %+v", segs) // the tail's key/text change deliberately in P2 Task 2.1 (the {{voice}} token); this pin survives both
	}
}
```

**File:** `domains/radio/synth/compose.go` (GREEN) — replace the signature and the report block:

```go
// Reports carries everything a broadcast cycle reads besides the location's
// own weather: the correspondent's name for the sign-off, and the fire and
// seismic reports (0.14.0 adds Maritime here — P3). A struct, not positional
// parameters: Compose was at eight and two batches grow it at once (RS-12).
type Reports struct {
	Voice   string // the sign-off's correspondent; "" = "your correspondent"
	Fire    FireReport
	Seismic SeismicReport
}

// Compose builds one broadcast cycle the way NWR does (AI-13): the lead
// (UAT 79 script), current conditions, active alerts, the office's products
// in broadcast order, the reports, then the tail naming the correspondent.
// Temperatures are read in the location's display units; everything is
// plain sentences for the voice.
func (c Composer) Compose(loc snapshot.Location, products []Product, now time.Time, imperial bool, station Station, r Reports) []Segment {
	var segs []Segment
	notice, span := c.LeadParts(loc.Label, station, now)
	segs = append(segs, Segment{Key: "lead:" + loc.Label + station.Callsign, Text: notice, Pause: leadPause},
		Segment{Key: "lead-span:" + now.Format("2006-01-02"), Text: span})
	if cond := c.conditions(loc, imperial); cond != "" {
		segs = append(segs, Segment{Key: "wx:" + cond, Text: cond})
	}
	for _, a := range loc.Alerts {
		text := c.say("weather-radio", "alert", map[string]string{"Headline": strings.TrimSuffix(NormalizeLine(a.Headline), "."), "Description": strings.Join(Normalize(a.Description), " ")})
		for i, piece := range Segments([]string{ExpandStates(text)}) {
			segs = append(segs, Segment{Key: fmt.Sprintf("alert:%s:%d", a.ID, i), Text: piece})
		}
	}
	for _, p := range products {
		for i, piece := range Segments(Normalize(p.Text)) {
			segs = append(segs, Segment{Key: fmt.Sprintf("%s:%s:%d", p.Type, p.ID, i), Text: piece})
		}
	}
	// UAT 115: two seconds of air between reports (forecast → fire → …),
	// one second before the sign-off — never one report running into the next.
	for _, report := range [][]Segment{
		c.FireSegments(loc.Label, r.Fire, imperial, now),       // UAT 114: after the forecast, before the tail; skipped without fire data
		c.SeismicSegments(loc.Label, r.Seismic, imperial, now), // P4: after the fire report; skipped without seismic entries
	} {
		if len(report) > 0 {
			pauseLast(segs, reportPause)
			segs = append(segs, report...)
		}
	}
	pauseLast(segs, tailPause)
	segs = append(segs, Segment{Key: "tail:" + r.Voice, Text: c.Tail(r.Voice)})
	return segs
}
```

**File:** `app/radio.go` — the call site in `segments()`:

```go
	var r synth.Reports
	r.Voice = voiceName
	if d.fire != nil {
		r.Fire = d.fire(ref)
	}
	if d.seismic != nil {
		r.Seismic = d.seismic(ref)
	}
	return d.composer.Compose(snap.Locations[0], products, now, d.units == render.UnitF, d.stationFor(county, ref), r), nil
```

(delete the two `var fire … var seismic …` blocks it replaces).

**Verify:** `go build ./... && go test ./domains/radio/synth ./app -run 'Compose|Reports|Lead|Seismic' -count=1`

---

### Task 1.3 — `cast`: the role registry (FR-1; data-shape §1)

**File:** `domains/radio/cast/cast_test.go` (RED)

```go
package cast

import "testing"

func TestEveryRoleHasAParentEndingAtAll(t *testing.T) {
	for r := All; r < roles; r++ {
		seen := 0
		for x := r; x != All; x = x.Parent() {
			if seen++; seen > 3 {
				t.Fatalf("%s: the walk must reach All within three links", r)
			}
		}
	}
	if Breaking.Parent() != Alerts || SevereRead.Parent() != Alerts || Alerts.Parent() != All {
		t.Fatal("alert roles inherit Alerts, Alerts inherits All")
	}
	for _, r := range []Role{Weather, Maritime, Fire, Seismic, Station} {
		if r.Parent() != Standard {
			t.Fatalf("%s must inherit Standard", r)
		}
	}
	if All.Parent() != All {
		t.Fatal("All is its own parent (the walk's fixed point)")
	}
}

func TestOutOfRangeRolesAndClassesReadEmptyNeverPanic(t *testing.T) {
	for _, r := range []Role{-1, roles, 99} {
		if r.Key() != "" || r.String() != "" || r.Parent() != All {
			t.Fatalf("role %d out of range: %q %q", r, r.Key(), r.String())
		}
	}
	for _, c := range []Class{-1, classes} {
		if c.String() != "" || c.Label() != "" {
			t.Fatalf("class %d out of range", c)
		}
	}
}

func TestRoleKeysAreTheConfigKeys(t *testing.T) {
	want := map[Role]string{All: "voice", Alerts: "alerts", Breaking: "breaking", SevereRead: "severe_read",
		Standard: "standard", Weather: "weather", Maritime: "maritime", Fire: "fire", Seismic: "seismic", Station: "station"}
	for r, k := range want {
		if r.Key() != k || r.String() == "" {
			t.Fatalf("%v: Key() = %q, want %q", int(r), r.Key(), k)
		}
	}
	if len(Assignable()) != 9 {
		t.Fatalf("nine assignable nodes below the root, got %d", len(Assignable()))
	}
}
```

**File:** `domains/radio/cast/cast.go` (GREEN)

```go
// Package cast is the station's cast: the correspondent roles a listener can
// assign a voice to, how an unassigned role inherits one (FR-1), how a role's
// voice resolves on this host with an explicit, never-silent fallback (FR-7),
// and which tone class a product belongs to (FR-11). Pure: no synth, no
// config, no UI — the app maps its config into a Config and supplies the
// host facts. One owner of "who speaks" (data-shape.md).
package cast

// Role is a node of the assignment tree. The zero value is the root.
type Role int

// The tree: All → {Alerts → {Breaking, SevereRead}, Standard → {Weather,
// Maritime, Fire, Seismic, Station}}. Order is the Setup window's order.
const (
	All Role = iota
	Alerts
	Breaking
	SevereRead
	Standard
	Weather
	Maritime
	Fire
	Seismic
	Station
	roles
)

// Parent is the node a role inherits from; All is its own parent.
func (r Role) Parent() Role {
	switch r {
	case Breaking, SevereRead:
		return Alerts
	case Weather, Maritime, Fire, Seismic, Station:
		return Standard
	case Alerts, Standard:
		return All
	}
	return All
}

// roleKeys and roleLabels are sized by the compiler to the role count: a
// missing entry is a build error, an out-of-range value reads "" (never a
// panic — a Role is a plain int on a Segment).
// Key is the role's config key ("voice" for the root); "" out of range.
func (r Role) Key() string {
	keys := [roles]string{"voice", "alerts", "breaking", "severe_read", "standard", "weather", "maritime", "fire", "seismic", "station"}
	if r < 0 || r >= roles {
		return ""
	}
	return keys[r]
}

// String is the role's label as Setup shows it; "" out of range.
func (r Role) String() string {
	labels := [roles]string{"All reads", "Alerts & Notification reads", "Breaking takeover", "Severe-event read",
		"Standard reports", "Location weather", "Location maritime", "Location fire & hotspots", "Location seismic", "Station"}
	if r < 0 || r >= roles {
		return ""
	}
	return labels[r]
}

// Assignable lists the nine nodes below the root in Setup order.
func Assignable() []Role {
	out := make([]Role, 0, roles-1)
	for r := Alerts; r < roles; r++ {
		out = append(out, r)
	}
	return out
}

// Pair is a role's voice on each platform: the macOS `say -v ?` name and the
// Piper catalogue key. "" = inherit on that platform.
type Pair struct {
	MacOS string
	Piper string
}

// Class is a tone class (FR-11, MVS-D-28): the six the Setup window lists,
// over five ratified presets (Disaster and Warning share the dual-tone).
type Class int

const (
	Disaster  Class = iota // significant quakes, tsunami — the ticker's quake class
	Warning                // NWS warnings
	Watch                  // NWS watches
	Advisory               // NWS advisories
	Statement              // special weather statements
	Storm                  // tropical cyclones, hurricane / tropical storm / winter storm / blizzard products — the mock's "Maritime"
	classes
)

// String is the class's config key; "" out of range.
func (c Class) String() string {
	keys := [classes]string{"disaster", "warning", "watch", "advisory", "statement", "storm"}
	if c < 0 || c >= classes {
		return ""
	}
	return keys[c]
}

// Label is the class as Setup lists it; "" out of range.
func (c Class) Label() string {
	labels := [classes]string{"Significant Quakes & Disasters", "Warnings", "Watches", "Advisories", "Special Statements", "Maritime"}
	if c < 0 || c >= classes {
		return ""
	}
	return labels[c]
}

// Classes lists every class in Setup order.
func Classes() []Class {
	out := make([]Class, 0, classes)
	for c := Disaster; c < classes; c++ {
		out = append(out, c)
	}
	return out
}

// Tones is the per-class mute (MVS-D-26): Mode "" = All Tones On, "mute" =
// the classes in Muted sound no tone (the words always read); [M] flips
// Mode and the set survives the flip.
type Tones struct {
	Mode  string
	Muted []string // class keys
}

// MuteMode is the one non-empty Tones.Mode value.
const MuteMode = "mute"

// Config is the cast as configured: the root voice, the nine pairs, whether
// the cast is in force (CastMode "cast") or the root reads everything
// (MVS-D-25: Single Voice keeps the pairs, unread), and the tones.
type Config struct {
	Root     string
	Pairs    map[Role]Pair
	CastMode string
	Tones    Tones
}
```

**Verify:** `go test ./domains/radio/cast -run 'Role|Parent' -count=1`

---

### Task 1.4 — `cast.Resolve`: the walk and the fallback matrix (FR-7; data-shape §3–4)

**File:** `domains/radio/cast/cast_test.go` (RED — append)

```go
type fakeHost struct {
	platform   string
	discovered []string
	installed  map[string]bool
}

func (h fakeHost) Platform() string          { return h.platform }
func (h fakeHost) Discovered() []string      { return h.discovered }
func (h fakeHost) Installed(key string) bool { return h.installed[key] }

func mac(names ...string) fakeHost { return fakeHost{platform: "darwin", discovered: names} }
func linux(keys ...string) fakeHost {
	h := fakeHost{platform: "linux", installed: map[string]bool{}}
	for _, k := range keys {
		h.installed[k] = true
	}
	return h
}

// The M5 fallback matrix: every node × the states data-shape.md §4 lists.
func TestResolveFallbackMatrix(t *testing.T) {
	cases := []struct {
		name string
		role Role
		cfg  Config
		host Host
		want Resolution
	}{
		{"single-voice mode ignores the pairs (kept, unread)", Breaking, Config{Root: "Samantha", Pairs: map[Role]Pair{Breaking: {MacOS: "Rishi"}}}, mac("Samantha", "Rishi"),
			Resolution{Role: Breaking, Requested: "Samantha", Spoken: "Samantha", Link: LinkRoot}},
		{"assigned, discovered", Breaking, Config{Root: "Samantha", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {MacOS: "Rishi"}}}, mac("Samantha", "Rishi"),
			Resolution{Role: Breaking, Requested: "Rishi", Spoken: "Rishi", Link: LinkRole}},
		{"inherits the group", Breaking, Config{Root: "Samantha", CastMode: "cast", Pairs: map[Role]Pair{Alerts: {MacOS: "Rishi"}}}, mac("Samantha", "Rishi"),
			Resolution{Role: Breaking, Requested: "Rishi", Spoken: "Rishi", Link: LinkGroup}},
		{"inherits the root", Weather, Config{Root: "Samantha"}, mac("Samantha"),
			Resolution{Role: Weather, Requested: "Samantha", Spoken: "Samantha", Link: LinkRoot}},
		{"assigned but unknown on macOS → the group", Breaking, Config{Root: "Samantha", CastMode: "cast", Pairs: map[Role]Pair{Alerts: {MacOS: "Rishi"}, Breaking: {MacOS: "Nobody"}}}, mac("Samantha", "Rishi"),
			Resolution{Role: Breaking, Requested: "Nobody", Spoken: "Rishi", Link: LinkGroup, Reason: `"Nobody" is not a voice on this host`}},
		{"assigned in the other OS's field only → inherit", Breaking, Config{Root: "Samantha", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {Piper: "en_US-ryan-medium"}}}, mac("Samantha"),
			Resolution{Role: Breaking, Requested: "Samantha", Spoken: "Samantha", Link: LinkRoot}},
		{"the sentinel always resolves", Weather, Config{Root: SystemVoice}, mac("Samantha"),
			Resolution{Role: Weather, Requested: SystemVoice, Spoken: SystemVoice, Link: LinkRoot}},
		{"root unresolvable → platform default", Weather, Config{Root: "Nobody"}, mac("Samantha"),
			Resolution{Role: Weather, Requested: "Nobody", Spoken: SystemVoice, Link: LinkDefault, Reason: `"Nobody" is not a voice on this host`}},
		{"Piper: assigned, installed", Breaking, Config{Root: "en_US-lessac-medium", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {Piper: "en_US-ryan-medium"}}}, linux("en_US-lessac-medium", "en_US-ryan-medium"),
			Resolution{Role: Breaking, Requested: "en_US-ryan-medium", Spoken: "en_US-ryan-medium", Link: LinkRole}},
		{"Piper: assigned, not installed → root, installing", Breaking, Config{Root: "en_US-lessac-medium", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {Piper: "en_US-ryan-medium"}}}, linux("en_US-lessac-medium"),
			Resolution{Role: Breaking, Requested: "en_US-ryan-medium", Spoken: "en_US-lessac-medium", Link: LinkRoot, Reason: `"en_US-ryan-medium" is not installed`}},
		{"Piper: nothing installed → the one silent row", Breaking, Config{Root: "en_US-lessac-medium"}, linux(),
			Resolution{Role: Breaking, Requested: "en_US-lessac-medium", Spoken: "", Link: LinkDefault, Reason: `"en_US-lessac-medium" is not installed`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Resolve(c.role, c.cfg, c.host)
			if got != c.want {
				t.Fatalf("\n got %+v\nwant %+v", got, c.want)
			}
		})
	}
	// FR-9: the install a resolution wants is derived, on Piper hosts only.
	res := Resolve(Breaking, Config{Root: "en_US-lessac-medium", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {Piper: "en_US-ryan-medium"}}}, linux("en_US-lessac-medium"))
	if res.WantsInstall(linux()) != "en_US-ryan-medium" || res.WantsInstall(mac()) != "" {
		t.Fatalf("WantsInstall: %q", res.WantsInstall(linux()))
	}
	if got := Resolve(Role(99), Config{Root: "Samantha"}, mac("Samantha")); got.Spoken != "" || got.Reason == "" {
		t.Fatal("an unknown role fails closed with a reason")
	}
}
```

**File:** `domains/radio/cast/resolve.go` (GREEN)

```go
package cast

import (
	"fmt"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// SystemVoice is the macOS sentinel: `say` with no -v (UAT 88). It is always
// resolvable on darwin and has no name of its own.
const SystemVoice = "System Voice"

// Host is what the app knows about this machine's voices. Discovered is the
// macOS voice list — the curated default list until `say -v ?` has been
// read, then the intersection (there is no "trust anything while
// discovering" window: a config name that is not on the list falls back,
// and never reaches argv); Installed is a pure os.Stat on a Piper key —
// never an install (FR-9).
type Host interface {
	Platform() string
	Discovered() []string
	Installed(key string) bool
}

// Link says which node of the walk supplied the voice that speaks.
type Link int

// The links, nearest first; LinkDefault is the platform's own default.
const (
	LinkRole Link = iota
	LinkGroup
	LinkRoot
	LinkDefault
)

// String is the link as [S] prints it; "" out of range (never a panic).
func (l Link) String() string {
	if l < 0 || l > LinkDefault {
		return ""
	}
	return [LinkDefault + 1]string{"role", "group", "root", "default"}[l]
}

// Resolution is what a role resolved to and why (RS-18: Spoken is the only
// name that ever reaches a voice or the [S] cast table).
type Resolution struct {
	Role      Role
	Requested string // the nearest assigned name, resolvable or not
	Spoken    string // the voice that speaks; "" only when nothing on this host can (Linux before any install)
	Link      Link
	Reason    string // why Requested lost, when it did
}

// WantsInstall is the Piper key worth installing in the background (FR-9):
// the requested name, when it lost for not being installed on a Piper host.
func (r Resolution) WantsInstall(host Host) string {
	if host.Platform() == "darwin" || r.Reason == "" || r.Link == LinkRole {
		return ""
	}
	return r.Requested
}

// maxDepth bounds the inheritance walk: role → group → root (P10-02).
const maxDepth = 3

// Resolve walks role → parent → … → root and returns the first voice that
// exists on this host; failing all, the platform default. Only this host's
// half of each pair is read. The walk is bounded by maxDepth and must reach
// All (an invariant, not an assumption).
func Resolve(role Role, cfg Config, host Host) Resolution {
	res := Resolution{Role: role}
	if err := invariant.Check(host != nil && role >= 0 && role < roles, "cast: a host and a known role are required"); err != nil {
		res.Reason = err.Error()
		return res
	}
	r := role
	for i := 0; i <= maxDepth; i, r = i+1, r.Parent() {
		name, defaulted := cfg.name(r, host.Platform()), false
		if name == "" && r != All {
			continue
		}
		if r == All && name == "" {
			name, defaulted = platformDefault(host), true
		}
		if res.Requested == "" {
			res.Requested = name
		}
		if ok, reason := resolvable(name, host); ok {
			res.Spoken = name
			res.Link = linkFor(role, r, defaulted)
			return res
		} else if res.Reason == "" {
			res.Reason = reason
		}
		if r == All {
			break
		}
	}
	if err := invariant.Check(r == All, "cast: the inheritance walk reached the root"); err != nil {
		res.Reason = err.Error()
	}
	res.Link = LinkDefault
	res.Spoken = platformDefault(host)
	if ok, _ := resolvable(res.Spoken, host); !ok {
		res.Spoken = "" // Linux with nothing installed: the one legitimately silent row (M5)
	}
	return res
}

// name is the role's own assignment on this platform ("" = inherit). In
// Single Voice mode (MVS-D-25) only the root is read: the pairs are kept
// for the day the cast is switched back on.
func (c Config) name(r Role, platform string) string {
	if r == All {
		return c.Root
	}
	if c.CastMode != "cast" {
		return ""
	}
	p := c.Pairs[r]
	if platform == "darwin" {
		return p.MacOS
	}
	return p.Piper
}

// resolvable: on darwin the sentinel or a name on the host's list (the
// curated default list before discovery, the intersection after — either
// way a closed allowlist); elsewhere an installed key.
func resolvable(name string, host Host) (bool, string) {
	if name == "" {
		return false, "no voice assigned"
	}
	if host.Platform() == "darwin" {
		if name == SystemVoice {
			return true, ""
		}
		for _, d := range host.Discovered() {
			if d == name {
				return true, ""
			}
		}
		return false, fmt.Sprintf("%q is not a voice on this host", name)
	}
	if host.Installed(name) {
		return true, ""
	}
	return false, fmt.Sprintf("%q is not installed", name)
}

// platformDefault is the voice a host speaks with when nothing resolves.
func platformDefault(host Host) string {
	if host.Platform() == "darwin" {
		return SystemVoice
	}
	return ""
}

// linkFor names the link a resolution came through: the role itself, its
// group, the root, or the platform default the empty root stood in for.
func linkFor(asked, got Role, defaulted bool) Link {
	switch {
	case defaulted:
		return LinkDefault
	case got == asked:
		return LinkRole
	case got == All:
		return LinkRoot
	}
	return LinkGroup
}
```

*Note for GREEN:* the "Piper: nothing installed" case expects `Requested = "en_US-lessac-medium"` (the root's own
value) — the walk reaches `All` with `cfg.Root` non-empty, finds it uninstalled, and the default on Linux is `""`;
the test pins that the row is silent, not wrong-voiced. `resolve.go` imports `platform/invariant` (the one import
below `platform/` the package allows).

`Resolve` measures 15 decision points (gocyclo) — the P10-04 ceiling; anything added to the walk goes into a
helper, not the loop. **Verify:** `go test ./domains/radio/cast -run Resolve -count=1`

---

### Task 1.5 — `cast.Validate` (FR-8, RS-2)

**File:** `domains/radio/cast/cast_test.go` (RED — append)

```go
func TestValidateNamesTheKey(t *testing.T) {
	cfg := Config{Root: "Samantha", CastMode: "cast", Pairs: map[Role]Pair{Breaking: {MacOS: "Nobody", Piper: "en_US-ryan-medium"}}, Tones: Tones{Mode: "loud", Muted: []string{"watch", "thunder"}}}
	got := Validate(cfg, mac("Samantha"))
	want := []Problem{
		{Key: "radio.voices.breaking.macos", Message: `"Nobody" is not a voice on this host — it will fall back to Alerts & Notification reads / All reads`},
		{Key: "radio.tones.mode", Message: `must be "" (all tones on) or "mute" (got "loud")`},
		{Key: "radio.tones.muted", Message: `"thunder" is not a tone class (disaster, warning, watch, advisory, statement, storm)`},
	}
	if len(got) != len(want) {
		t.Fatalf("problems: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("problem %d:\n got %+v\nwant %+v", i, got[i], want[i])
		}
	}
	if p := Validate(Config{Root: "Samantha"}, mac("Samantha")); len(p) != 0 {
		t.Fatalf("a clean config has no problems: %+v", p)
	}
}
```

**File:** `domains/radio/cast/validate.go` (GREEN)

```go
package cast

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Problem is one config fault, named by its key so the message is actionable
// (the framework's error rule); voice problems are warnings — the voice falls
// back (FR-7) — and are listed once in [S].
type Problem struct {
	Key     string
	Message string
}

// Validate checks the cast config against this host: assigned names that
// cannot speak here (a warning, with where they fall back to), the cast
// mode, and the closed tone set. The app calls it whenever the cast is
// built or a host fact changes (setCast / castChanged) and lists the
// problems in [S] — the class keys have no other validator.
func Validate(cfg Config, host Host) []Problem {
	var out []Problem
	if err := invariant.Check(host != nil, "cast: Validate needs the host"); err != nil {
		return []Problem{{Key: "radio", Message: err.Error()}}
	}
	for _, r := range Assignable() {
		if name := cfg.name(r, host.Platform()); name != "" {
			if ok, reason := resolvable(name, host); !ok {
				out = append(out, Problem{Key: "radio.voices." + r.Key() + "." + platformKey(host), Message: reason + " — it will fall back to " + r.Parent().String() + " / " + All.String()})
			}
		}
	}
	if cfg.CastMode != "" && cfg.CastMode != "cast" {
		out = append(out, Problem{Key: "radio.cast", Message: fmt.Sprintf(`must be "" (single voice) or "cast" (got %q)`, cfg.CastMode)})
	}
	if cfg.Tones.Mode != "" && cfg.Tones.Mode != MuteMode {
		out = append(out, Problem{Key: "radio.tones.mode", Message: fmt.Sprintf(`must be "" (all tones on) or %q (got %q)`, MuteMode, cfg.Tones.Mode)})
	}
	for _, m := range cfg.Tones.Muted {
		if _, ok := ClassByKey(m); !ok {
			out = append(out, Problem{Key: "radio.tones.muted", Message: fmt.Sprintf("%q is not a tone class (%s)", m, strings.Join(ClassKeys(), ", "))})
		}
	}
	return out
}

func platformKey(host Host) string {
	if host.Platform() == "darwin" {
		return "macos"
	}
	return "piper"
}
```

**Verify:** `go test ./domains/radio/cast -run Validate -count=1`

---

### Task 1.6 — `cast.Classify` and `ToneName` (FR-11; tones.md §2; MVS-D-15)

**File:** `domains/radio/cast/cast_test.go` (RED — append)

```go
func TestClassifyProducts(t *testing.T) {
	cases := map[string]Class{
		"Tornado Warning": Warning, "Severe Thunderstorm Warning": Warning, "Flash Flood Warning": Warning,
		"Tornado Watch": Watch, "Flood Watch": Watch,
		"Wind Advisory": Advisory, "Winter Weather Advisory": Advisory, "Heat Advisory": Advisory,
		"Special Weather Statement": Statement, "Hurricane Local Statement": Statement,
		"Hurricane Warning": Storm, "Tropical Storm Watch": Storm, "Winter Storm Warning": Storm, "Winter Storm Watch": Storm, "Blizzard Warning": Storm,
		"Tsunami Warning": Disaster, "Earthquake": Disaster, "": Warning, "Something Nobody Has Seen": Warning,
	}
	for product, want := range cases {
		if got := Classify(product); got != want {
			t.Errorf("Classify(%q) = %s, want %s", product, got, want)
		}
	}
}

func TestToneNameAndMute(t *testing.T) {
	if ToneName(Disaster) != "dual-tone" || ToneName(Warning) != "dual-tone" || ToneName(Watch) != "1050" || ToneName(Advisory) != "classic" || ToneName(Statement) != "soft-chime" || ToneName(Storm) != "low-sweep" {
		t.Fatal("each class sounds its ratified preset; Disaster and Warning share the dual-tone (MVS-D-28)")
	}
	tones := Tones{Mode: MuteMode, Muted: []string{"watch", "storm"}}
	if !Muted(Watch, tones) || Muted(Warning, tones) || Muted(Watch, Tones{Muted: []string{"watch"}}) {
		t.Fatal("muted = mode mute AND the class listed; All Tones On ignores the set")
	}
	if c, ok := ClassByKey("statement"); !ok || c != Statement {
		t.Fatal("ClassByKey")
	}
}
```

**File:** `domains/radio/cast/tone.go` (GREEN)

```go
package cast

import "strings"

// Classify maps a product string to its tone class (tones.md §2). Storm wins
// over warning/watch (MVS-D-15); a quake or a tsunami is a disaster; an
// advisory stays an advisory; anything no rule matches — a product nobody
// has seen — is a warning, the loud default. The product is the NWS event
// name for weather, the ticker's type for quakes and storms.
func Classify(product string) Class {
	p := strings.ToLower(strings.TrimSpace(product))
	switch {
	case strings.Contains(p, "earthquake"), strings.Contains(p, "tsunami"):
		return Disaster
	case strings.Contains(p, "hurricane") && !strings.Contains(p, "statement"),
		strings.Contains(p, "tropical storm"), strings.Contains(p, "tropical cyclone"),
		strings.Contains(p, "winter storm"), strings.Contains(p, "blizzard"):
		return Storm
	case strings.HasSuffix(p, "statement"):
		return Statement
	case strings.HasSuffix(p, "advisory"):
		return Advisory
	case strings.HasSuffix(p, "watch"):
		return Watch
	}
	return Warning
}

// Preset names, as synth registers them (P3 ports the audited parameters).
const (
	PresetDualTone = "dual-tone"
	PresetWatch1050 = "1050"
	PresetClassic  = "classic"
	PresetChime    = "soft-chime"
	PresetSweep    = "low-sweep"
)

// ToneName is the preset a class sounds (MVS-D-11/28): Disaster and
// Warning share the dual-tone; the rest their own. Out of range sounds the
// loudest preset — the fail-loud rule of the classifier (never a panic).
func ToneName(c Class) string {
	if c < 0 || c >= classes {
		return PresetDualTone
	}
	return [classes]string{PresetDualTone, PresetDualTone, PresetWatch1050, PresetClassic, PresetChime, PresetSweep}[c]
}

// ClassKeys is Classes() as config keys (one owner: the registry).
func ClassKeys() []string {
	out := make([]string, 0, classes)
	for _, c := range Classes() {
		out = append(out, c.String())
	}
	return out
}

// Muted reports whether a class sounds no tone: mute mode AND the class
// listed (MVS-D-26). The words always read regardless.
func Muted(c Class, t Tones) bool {
	if t.Mode != MuteMode {
		return false
	}
	if len(t.Muted) == 0 {
		return true // "Mute:" with nothing ticked, or the [M] key: every class (what the screen says is what the ear gets)
	}
	for _, k := range t.Muted {
		if k == c.String() {
			return true
		}
	}
	return false
}

// ClassByKey is the class for a config key.
func ClassByKey(key string) (Class, bool) {
	for _, c := range Classes() {
		if c.String() == key {
			return c, true
		}
	}
	return Warning, false
}
```

**Verify:** `go test ./domains/radio/cast -count=1 && go vet ./domains/radio/cast`

---

### Task 1.7 — (withdrawn) the `cast` import pin

`make lint-imports` guards `modes/`; `cast`'s own purity (no `synth`/`config`/`tty`/`app` imports) is stated in
its package doc and is visible in `go list -deps`; a hand-rolled grep test duplicated a gate (Code Quality #19).

---

### Task 1.8 — config: `RoleVoice`, `Voices`, `Tones`, the `[radio]` fields (FR-8; data-shape §2)

**File:** `platform/config/config_test.go` (RED — append)

```go
func TestRadioVoicesRoundTripAndOmitEmptyPairs(t *testing.T) {
	testDir(t)
	cfg := Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Voices.Alerts = RoleVoice{MacOS: "Rishi", Piper: "en_US-ryan-medium"}
	cfg.Radio.Cast = "cast"
	cfg.Radio.Tones = Tones{Mode: "mute", Muted: []string{"watch"}}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	p, _ := Path()
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), "[radio.voices.alerts]") || strings.Contains(string(raw), "[radio.voices.breaking]") {
		t.Fatalf("an assigned pair is a table; an empty pair is omitted:\n%s", raw)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Radio.Voices.Alerts != cfg.Radio.Voices.Alerts || got.Radio.Cast != "cast" || got.Radio.Tones.Mode != "mute" || len(got.Radio.Tones.Muted) != 1 || got.Voice != "Samantha" {
		t.Fatalf("round trip: %+v", got.Radio)
	}
}

func TestRadioValidateNamesTheKey(t *testing.T) {
	cfg := Default()
	cfg.Radio.Tones.Mode = "loud"
	if err := cfg.Radio.Validate(); err == nil || !strings.Contains(err.Error(), "[radio.tones] mode") {
		t.Fatalf("validate: %v", err)
	}
	cfg = Default()
	cfg.Radio.Cast = "ensemble"
	if err := cfg.Radio.Validate(); err == nil || !strings.Contains(err.Error(), "[radio] cast") {
		t.Fatalf("validate: %v", err)
	}
}

func TestLoadValidatesRadio(t *testing.T) {
	d := testDir(t)
	dir := filepath.Join(d, "watchpost")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[radio.tones]\nmode = \"loud\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "[radio.tones] mode") {
		t.Fatalf("Load must report the bad tone by key: %v", err)
	}
}
```

**File:** `platform/config/config.go` (GREEN) — replace the `Radio` type and add the validation call:

```go
// RoleVoice is a correspondent role's voice on each platform (0.14.0): the
// macOS `say -v ?` name and the Piper catalogue key. "" = inherit from the
// level above (the cast's tree); each OS reads and writes only its own
// field, so one file serves a Mac and a Linux box (FR-8).
type RoleVoice struct {
	MacOS string `toml:"macos,omitempty"`
	Piper string `toml:"piper,omitempty"`
}

// Voices are the nine assignable nodes of the cast (data-shape.md §1). A
// struct of structs, not a map: a typo'd role in a hand-edited file is
// ignored, never minted.
type Voices struct {
	Alerts     RoleVoice `toml:"alerts,omitempty"`
	Breaking   RoleVoice `toml:"breaking,omitempty"`
	SevereRead RoleVoice `toml:"severe_read,omitempty"`
	Standard   RoleVoice `toml:"standard,omitempty"`
	Weather    RoleVoice `toml:"weather,omitempty"`
	Maritime   RoleVoice `toml:"maritime,omitempty"`
	Fire       RoleVoice `toml:"fire,omitempty"`
	Seismic    RoleVoice `toml:"seismic,omitempty"`
	Station    RoleVoice `toml:"station,omitempty"`
}

// Tones is the per-class tone mute (FR-11, MVS-D-26): Mode "" = All Tones On,
// "mute" = the classes in Muted sound no tone (the words always read); [M]
// flips Mode and the set survives. Class keys: disaster · warning · watch ·
// advisory · statement · storm.
type Tones struct {
	Mode  string   `toml:"mode,omitempty"`
	Muted []string `toml:"muted,omitempty"`
}

// Radio holds tuner settings: the [m] source pick (UAT 97), and from 0.14.0
// whether the cast is in force (Cast "" = Single Voice, "cast" — MVS-D-25),
// the cast's voices and the tone mute. (A Piper process policy — resident
// vs per-utterance — is 0.15.0's, after MVS-D-17's measurement; no key here.)
type Radio struct {
	Mode   string `toml:"mode,omitempty"` // "synth" (default) | "relay"
	Cast   string `toml:"cast,omitempty"`
	Voices Voices `toml:"voices,omitempty"`
	Tones  Tones  `toml:"tones,omitempty"`
}

// Validate checks the [radio] tables the way Load reports every other config
// fault — at load, naming the key. Voice names are not checked here: whether
// a name speaks is a host fact the cast resolves with a fallback (FR-7).
func (r Radio) Validate() error {
	if r.Cast != "" && r.Cast != "cast" {
		return fmt.Errorf(`[radio] cast must be "" (single voice) or "cast" (got %q)`, r.Cast)
	}
	if r.Tones.Mode != "" && r.Tones.Mode != "mute" {
		return fmt.Errorf(`[radio.tones] mode must be "" (all tones on) or "mute" (got %q)`, r.Tones.Mode)
	}
	return nil // the class keys are the radio domain's (cast.Validate reports an unknown one at castConfig time — one owner)
}
```

and in `Load`, after `cfg.Fire.Validate()`:

```go
	if err := cfg.Radio.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", p, err)
	}
	cfg.Radio.Tones = cfg.Radio.Tones.fromTickerMuted(cfg.TickerMuted) // 0.13.0's [M] read as "mute every class" (MVS-D-26)
```

```go
// fromTickerMuted maps a 0.13.0 `ticker_muted = true` with no [radio.tones]
// onto the 0.14.0 model: mute mode over every class. A file that already
// carries a mode keeps it.
func (t Tones) fromTickerMuted(muted bool) Tones {
	if !muted || t.Mode != "" {
		return t
	}
	return Tones{Mode: "mute"} // an empty set under "mute" is every class (cast.Muted)
}
```

and in `Save`, before marshalling: `cfg.TickerMuted = cfg.Radio.Tones.Mode == "mute"` — a 0.13.0 binary reading
the file still mutes the ticker (NFR-5). Add to `TestRadioVoicesRoundTripAndOmitEmptyPairs`: a file with
`ticker_muted = true` and no `[radio.tones]` loads as `Mode "mute"` with an empty set (= every class,
`cast.Muted`); saving a config in mute mode writes `ticker_muted = true` — `Save` is the ONE owner of that mirror
(the app's `[M]` hook writes `Radio.Tones.Mode` and nothing else, P2 Task 2.10).

**Verify:** `go test ./platform/config -run 'Radio|Validates' -count=1`

---

### Task 1.9 — config: `Save` preserves keys it does not know (NFR-5) — no reflection, no recursion

**File:** `platform/config/config_test.go` (RED — as before)

```go
func TestSavePreservesUnknownKeysAndDropsClearedKnownOnes(t *testing.T) {
	d := testDir(t)
	dir := filepath.Join(d, "watchpost")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	future := "voice = \"Samantha\"\ntheme = \"Monochrome\"\n\n[future_table]\nknob = 3\n\n[radio]\nmode = \"relay\"\nfuture_key = \"kept\"\n\n[radio.voices.alerts]\nmacos = \"Rishi\"\nfuture_field = true\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(future), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Theme = "" // a known key, cleared: must NOT survive from the old file
	cfg.Radio.Voices.Alerts.MacOS = "Daniel"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	s := string(raw)
	for _, want := range []string{"[future_table]", "knob = 3", "future_key = 'kept'", "future_field = true", "macos = 'Daniel'", "mode = 'relay'"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q after save:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Monochrome") {
		t.Fatalf("a cleared known key must not resurrect from the old file:\n%s", s)
	}
	if _, err := Load(); err != nil {
		t.Fatalf("the merged file must load: %v", err)
	}
}

func TestSaveWithoutUnknownKeysIsByteStableAndAnUnparsableOldFileIsNotMerged(t *testing.T) {
	testDir(t)
	cfg := Default()
	cfg.Voice = "Samantha"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	p, _ := Path()
	first, _ := os.ReadFile(p)
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Fatal("a save with nothing unknown to keep re-marshals nothing (byte-stable)")
	}
	if err := os.WriteFile(p, []byte("not = toml ="), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err != nil {
		t.Fatalf("an unparsable old file is replaced by the struct's marshal, never merged: %v", err)
	}
}
```

**File:** `platform/config/keep.go` (GREEN) — go-toml's strict decoder names every key the schema does not
declare (`toml.StrictMissingError`); those key paths — kept as **segments**, never joined and re-split, so a
quoted top-level key `"radio.mode"` stays one segment and can never alias the known nested key — are the only
thing copied from the old file. No reflection, no recursion. **Pre-code spike (P1 build log):** run `TestConfigFixtures` on the pinned go-toml v2.2.4 with
`quoted-escape-key.toml`. Two independent probes disagreed on whether v2.2.4 panics while wording a
`StrictMissingError` for a quoted key with an escape — bump to v2.4.3 (`go get`, `scripts/third-party-licenses.sh`,
then `make verify`'s tidy/vuln; `go.mod`, `go.sum` and `THIRD_PARTY_LICENSES.md` join this task's files) **only
if the panic reproduces**; the scoped `recover` in `strictDecode` stays either way.

```go
package config

import (
	"errors"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// unknownKeys lists the key paths (as segments) in raw that the Config
// schema does not declare, sorted by their dotted form — go-toml's strict
// decoder reports them (StrictMissingError). A file the strict decode
// cannot parse, or one the decoder chokes on, yields nil: this is a
// diagnostic and never fails a Load.
func unknownKeys(raw []byte) (paths [][]string) {
	strict := strictDecode(raw)
	if strict == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, e := range strict.Errors {
		segs := []string(e.Key()) // a method in go-toml v2; one segment per (possibly quoted) key
		if dotted := strings.Join(segs, "\x00"); !seen[dotted] {
			seen[dotted] = true
			paths = append(paths, segs)
		}
	}
	sort.Slice(paths, func(i, j int) bool { // parents before children (a whole unknown table is copied once, its sub-tables then read as taken), then by name
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) < len(paths[j])
		}
		return strings.Join(paths[i], ".") < strings.Join(paths[j], ".")
	})
	return paths
}

// strictDecode runs the strict decoder and returns its unknown-key report,
// nil when there is none or the decoder itself fails. The recover is scoped
// to the decoder only (a bug in the caller's loop must not read as "no
// unknown keys" — NFR-5 would then fail silently): some go-toml versions
// panic while wording a StrictMissingError for a quoted key with an escape.
func strictDecode(raw []byte) (strict *toml.StrictMissingError) {
	defer func() {
		if recover() != nil {
			strict = nil
		}
	}()
	dec := toml.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var probe Config
	if err := dec.Decode(&probe); !errors.As(err, &strict) {
		return nil
	}
	return strict
}

// keepUnknown copies each unknown path's value from old into fresh, creating
// intermediate tables as needed; a path whose parent is a known scalar in
// fresh is skipped (never turn a known key into a table), and a path through
// an array of tables ([[locations]], [[recent]]) is skipped — a limitation
// NFR-5 records: unknown keys survive inside tables, not inside array
// elements. Bounded by the paths and their depth; no recursion.
func keepUnknown(fresh, old map[string]any, paths [][]string) (kept int) {
	if fresh == nil || old == nil { // an empty document decodes to a nil map
		return 0
	}
	for _, parts := range paths {
		if len(parts) == 0 {
			continue
		}
		src, dst, ok := old, fresh, true
		for _, part := range parts[:len(parts)-1] {
			next, isTable := src[part].(map[string]any)
			if !isTable {
				ok = false
				break
			}
			src = next
			child, exists := dst[part]
			if !exists {
				child = map[string]any{}
				dst[part] = child
			}
			table, isTable := child.(map[string]any)
			if !isTable {
				ok = false // a known scalar sits where the old file had a table
				break
			}
			dst = table
		}
		if !ok {
			continue
		}
		leaf := parts[len(parts)-1]
		if v, exists := src[leaf]; exists {
			if _, taken := dst[leaf]; !taken {
				dst[leaf] = v
				kept++
			}
		}
	}
	return kept
}
```

Fixtures (Task 1.10) pin: a quoted top-level `"radio.mode" = "x"` and one under `[radio]` survive as
themselves and never touch `[radio] mode` (the top-level one is kept by `keepUnknown` but not listed by
`unknownRadioKeys` — it is not a `[radio.*]` key); an unknown key inside `[[locations]]` is dropped (the
recorded limitation); an unknown table with a sub-table is copied once (`unknown-table.toml`:
`[future_table]` + `[future_table.deeper]`, then `Save`, then `Load` shows both, one copy); a known scalar where
the old file had a table wins (`scalar-vs-table.toml`: `voice = { x = 1 }` alongside a fresh `voice = "…"` —
the save keeps the string); the escape-carrying quoted key loads without a panic.

**File:** `platform/config/config.go` — in `Save`, replace `raw, err := toml.Marshal(cfg)` … with:

```go
	raw, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}
	if old, readErr := os.ReadFile(p); readErr == nil { // NFR-5: a future build's keys survive this build's save
		if paths := unknownKeys(old); len(paths) > 0 {
			var oldMap, freshMap map[string]any
			if toml.Unmarshal(old, &oldMap) == nil && toml.Unmarshal(raw, &freshMap) == nil && keepUnknown(freshMap, oldMap, paths) > 0 {
				if merged, mErr := toml.Marshal(freshMap); mErr == nil {
					raw = merged // re-marshalled only when something was kept: an ordinary save stays byte-stable
				}
			}
		}
	}
```

The per-preference cost is one strict decode of a ≤ 4 KB file; the re-marshal happens only when a future key
exists. The unknown-key list under `[radio]` for `[S]` (Task 1.11) reuses `unknownKeys` filtered by the
`radio.` prefix. **Verify:** `go test ./platform/config -count=1 -race`

---

### Task 1.10 — cross-version fixtures (NFR-5, FR-8)

**Files:** `platform/config/testdata/0.13.0-only.toml`

```toml
voice = "Samantha"
theme = "Monochrome"

[radio]
mode = "synth"
```

`platform/config/testdata/0.14.0-both-os.toml`

```toml
voice = "Samantha"

[radio]
mode = "synth"
cast = "cast"

[radio.voices.alerts]
macos = "Rishi"
piper = "en_US-ryan-medium"

[radio.voices.station]
macos = "Daniel"

[radio.tones]
mode = "mute"
muted = ["watch"]
```

`platform/config/testdata/0.14.0-macos-only.toml`

```toml
voice = "Samantha"

[radio.voices.breaking]
macos = "Rishi"
```

`platform/config/testdata/0.14.0-piper-only.toml`

```toml
voice = "en_US-lessac-medium"

[radio.voices.breaking]
piper = "en_US-ryan-medium"
```

`platform/config/testdata/hostile-name.toml`

```toml
voice = "\u001b[31mRishi"

[radio.voices.breaking]
macos = "Rishi\nfoo"
```

`platform/config/testdata/quoted-escape-key.toml` (the strict decoder's own failure mode; SEC round 2)

```toml
voice = "Samantha"
"radio.mode" = "top-level alias"
[radio]
"\u001b[31mx" = 1
"radio.mode" = "alias"
```

`platform/config/testdata/unknown-table.toml` and `scalar-vs-table.toml` (SEC round 3) — the two merge-edge fixtures
named under Task 1.9.

`platform/config/testdata/unknown-key.toml`

```toml
voice = "Samantha"

[radio.voices.wether]
macos = "Amy"
```

`platform/config/testdata/wrong-type.toml`

```toml
voice = "Samantha"

[radio.voices]
alerts = 3
```

`platform/config/testdata/bad-tone.toml`

```toml
[radio.tones]
mode = "loud"
```

**File:** `platform/config/config_test.go` (append)

```go
func loadFixture(t *testing.T, name string) (Config, error) {
	t.Helper()
	d := testDir(t)
	dir := filepath.Join(d, "watchpost")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return Load()
}

func TestCrossVersionFixtures(t *testing.T) {
	if cfg, err := loadFixture(t, "0.13.0-only.toml"); err != nil || cfg.Radio.Voices != (Voices{}) || cfg.Voice != "Samantha" {
		t.Fatalf("a 0.13.0 file loads with empty tables (everything inherits the root): %+v %v", cfg.Radio, err)
	}
	if cfg, err := loadFixture(t, "0.14.0-both-os.toml"); err != nil || cfg.Radio.Voices.Alerts != (RoleVoice{MacOS: "Rishi", Piper: "en_US-ryan-medium"}) || cfg.Radio.Voices.Station.MacOS != "Daniel" || cfg.Radio.Cast != "cast" || cfg.Radio.Tones.Mode != "mute" {
		t.Fatalf("both-OS pairs load: %+v %v", cfg.Radio, err)
	}
	if cfg, err := loadFixture(t, "quoted-escape-key.toml"); err != nil || cfg.Radio.Mode != "" || len(cfg.Unknown) != 2 || strings.ContainsAny(strings.Join(cfg.Unknown, ""), "\x1b\n") {
		t.Fatalf("a hostile quoted key never panics Load, never aliases [radio] mode, and reaches [S] plain: %+v %v", cfg.Unknown, err)
	}
	if cfg, err := loadFixture(t, "unknown-key.toml"); err != nil || cfg.Radio.Voices != (Voices{}) {
		t.Fatalf("an unknown role is ignored, never minted: %+v %v", cfg.Radio, err)
	}
	if cfg, err := loadFixture(t, "0.14.0-macos-only.toml"); err != nil || cfg.Radio.Voices.Breaking != (RoleVoice{MacOS: "Rishi"}) {
		t.Fatalf("a Mac-only pair loads with an empty piper half (Linux inherits): %+v %v", cfg.Radio.Voices.Breaking, err)
	}
	if cfg, err := loadFixture(t, "0.14.0-piper-only.toml"); err != nil || cfg.Radio.Voices.Breaking != (RoleVoice{Piper: "en_US-ryan-medium"}) {
		t.Fatalf("a Piper-only pair loads with an empty macos half (macOS inherits): %+v %v", cfg.Radio.Voices.Breaking, err)
	}
	if cfg, err := loadFixture(t, "hostile-name.toml"); err != nil || cfg.Voice != "\x1b[31mRishi" || cfg.Radio.Voices.Breaking.MacOS != "Rishi\nfoo" {
		t.Fatalf("hostile names load verbatim here — the cast validates them against the discovered list and the frame never shows the config string (FR-7, P2): %+v %v", cfg, err)
	}
	if _, err := loadFixture(t, "wrong-type.toml"); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("a wrong type is a loud corrupt-file error: %v", err)
	}
	if _, err := loadFixture(t, "bad-tone.toml"); err == nil || !strings.Contains(err.Error(), "[radio.tones] mode") {
		t.Fatalf("a bad tone names its key: %v", err)
	}
}
```

**Verify:** `go test ./platform/config -run Fixtures -count=1`

---

### Task 1.11 — `[S]` lists unknown `[radio.*]` keys once (NFR-5)

**File:** `platform/config/config_test.go` (RED — append)

```go
func TestUnknownRadioKeysAreReported(t *testing.T) {
	cfg, err := loadFixture(t, "unknown-key.toml")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Unknown; len(got) != 1 || got[0] != "radio.voices.wether" {
		t.Fatalf("unknown keys under [radio.*] are listed: %v", got)
	}
}
```

**File:** `platform/config/config.go` (GREEN) — add to `Config`:

```go
	Unknown []string `toml:"-"` // keys under [radio.*] this build does not know, for [S] (NFR-5); never persisted; already plain one-line text (unknownRadioKeys).
```

and in `Load`, after the `toml.Unmarshal(raw, &cfg)` succeeds:

```go
	cfg.Unknown = unknownRadioKeys(raw)
```

**File:** `platform/config/keep.go` (append)

```go
// unknownRadioKeys is the [radio.*] subset of unknownKeys as display
// strings — dotted, and through plaintext.Line because a quoted TOML key may
// carry escapes — capped at 32 entries for the diagnostics page: hand-edited
// typos are ignored by the contract, and this is where a user learns that
// (NFR-5). The only owner of that rendering.
func unknownRadioKeys(raw []byte) []string {
	var out []string
	for _, segs := range unknownKeys(raw) {
		if len(segs) < 2 || segs[0] != "radio" {
			continue
		}
		out = append(out, plaintext.Line(strings.Join(segs, ".")))
		if len(out) == 32 {
			break
		}
	}
	return out
}
```

(`platform/plaintext` imported in `keep.go`; `config` may import `platform/*`.)

```go
```
**Verify:** `go test ./platform/config -count=1`

---

### Task 1.12 — `app`: config → `cast.Config`, and the deck as `cast.Host`

**File:** `app/cast_test.go` (RED)

```go
package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/config"
)

func TestCastConfigMapsEveryPair(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Voices.Alerts = config.RoleVoice{MacOS: "Rishi", Piper: "en_US-ryan-medium"}
	cfg.Radio.Voices.Station = config.RoleVoice{MacOS: "Daniel"}
	cfg.Radio.Cast = "cast"
	cfg.Radio.Tones = config.Tones{Mode: "mute", Muted: []string{"storm"}}
	c := castConfig(cfg)
	if c.Root != "Samantha" || c.CastMode != "cast" || c.Pairs[cast.Alerts] != (cast.Pair{MacOS: "Rishi", Piper: "en_US-ryan-medium"}) || c.Pairs[cast.Station].MacOS != "Daniel" || c.Tones.Mode != cast.MuteMode || c.Tones.Muted[0] != "storm" {
		t.Fatalf("castConfig: %+v", c)
	}
	if len(c.Pairs) != 9 {
		t.Fatalf("nine pairs, got %d", len(c.Pairs))
	}
}

func TestDeckIsAHost(t *testing.T) {
	var _ cast.Host = (*radioDeck)(nil)
}
```

**File:** `app/cast.go` (GREEN)

```go
package app

// cast.go — the app's side of the cast (0.14.0): the config mapped into the
// cast's Config, and the deck as the cast's Host (the machine's voice facts).

import (
	"runtime"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/config"
)

// castConfig maps the config's typed tables into the cast's Config — the
// struct fields ARE the registry (data-shape.md §1), so the mapping is the
// one place the two are tied.
func castConfig(cfg config.Config) cast.Config {
	v := cfg.Radio.Voices
	pair := func(rv config.RoleVoice) cast.Pair { return cast.Pair{MacOS: rv.MacOS, Piper: rv.Piper} }
	return cast.Config{
		Root: cfg.Voice,
		Pairs: map[cast.Role]cast.Pair{
			cast.Alerts: pair(v.Alerts), cast.Breaking: pair(v.Breaking), cast.SevereRead: pair(v.SevereRead),
			cast.Standard: pair(v.Standard), cast.Weather: pair(v.Weather), cast.Maritime: pair(v.Maritime),
			cast.Fire: pair(v.Fire), cast.Seismic: pair(v.Seismic), cast.Station: pair(v.Station),
		},
		CastMode:  cfg.Radio.Cast,
		Tones:     cast.Tones{Mode: cfg.Radio.Tones.Mode, Muted: append([]string(nil), cfg.Radio.Tones.Muted...)},
	}
}

// Platform implements cast.Host.
func (d *radioDeck) Platform() string { return runtime.GOOS }

// Discovered implements cast.Host: the curated macOS list (macVoices) until
// `say -v ?` has been read, then the intersection listVoices stored — a
// closed allowlist either way, so no config name reaches argv unchecked.
func (d *radioDeck) Discovered() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.discovered {
		return macVoices()
	}
	return append([]string(nil), d.voices...)
}

// Installed implements cast.Host: a Piper key whose model and json are on
// disk — a pure os.Stat, never an install (FR-9).
func (d *radioDeck) Installed(key string) bool {
	spec, ok := synth.VoiceByName(key)
	if !ok {
		return false
	}
	_, ok = synth.FindPiperVoice(d.voiceDir, spec)
	return ok
}
```

**File:** `app/radio.go` — add to `radioDeck` beside `voices`:

```go
	discovered bool // listVoices has run (Discovered answers from the curated list before)
```

and in `app/voices.go` `listVoices`:

```go
func (d *radioDeck) listVoices() {
	list := d.discoverVoices()
	d.mu.Lock()
	d.voices, d.discovered = list, true
	d.mu.Unlock()
	d.castChanged() // a host fact changed: the [S] rows rebuild, a running broadcast re-resolves at its next segment
}

// castChanged is the tail every host-fact change runs. P1 has nothing to
// rebuild yet — P2 Task 2.7 replaces this body (the [S] rows, the problems,
// the Source's Invalidate); it exists here so P1 builds on its own.
func (d *radioDeck) castChanged() {}
```

**Verify:** `go test ./app -run 'Cast|Host' -count=1 && go vet ./app`

---

### Task 1.13 — the deck owns one `Limiter`; every voice it hands out is `Limited` (FR-12)

**File:** `app/radio_test.go` (RED — append)

```go
func TestDeckVoicesAreLimited(t *testing.T) {
	// One ordinary slot, no reserved one, a short bound: a second Say while
	// the first holds the slot must come back with the limiter's error — the
	// deck's voices are admitted by its limiter, not merely wrapped.
	d := &radioDeck{limiter: &synth.Limiter{Bound: 200 * time.Millisecond}}
	*d.limiter = *synth.NewLimiter(1, 0)
	d.limiter.Bound = 200 * time.Millisecond
	stuck := &stuckVoice{entered: make(chan struct{})}
	v := d.limited(stuck)
	go func() { _, _ = v.Say(context.Background(), "one") }()
	<-stuck.entered // the first Say holds the slot (no timing poll)
	_, err := v.Say(context.Background(), "two")
	if err == nil || !strings.Contains(err.Error(), "render slot") {
		t.Fatalf("the second Say must wait on the deck's limiter and hit its bound: %v", err)
	}
}

// stuckVoice holds its slot for a second and signals when it has it.
type stuckVoice struct{ entered chan struct{} }

func (*stuckVoice) Name() string { return "stuck" }
func (*stuckVoice) Rate() int    { return 22050 }
func (v *stuckVoice) Say(ctx context.Context, _ string) ([]byte, error) {
	close(v.entered)
	select {
	case <-time.After(time.Second):
	case <-ctx.Done():
	}
	return nil, ctx.Err()
}
```

**File:** `app/radio.go` (GREEN) — field + constructor + helper:

```go
	limiter *synth.Limiter // FR-12: one owner of "how many renders at once" (N ordinary + 2 reserved for the Director's jobs)
```

in `newRadioDeck` (beside `engine:`): `limiter: synth.NewLimiter(renderSlots(), 2),` (two reserved slots — the
Director's job on the air and the one it suspended) and:

```go
// renderSlots is the ordinary render pool per platform (FR-12, plan AX-4):
// `say` is ~1 s a line, Piper ~10 s and a model load each — fewer at once.
func renderSlots() int {
	if runtime.GOOS == "darwin" {
		return 3
	}
	return 2
}

// limited admits a voice's renders through the deck's limiter.
func (d *radioDeck) limited(v synth.Voice) synth.Voice { return synth.Limited(v, d.limiter) }
```

and in `app/voices.go` `voice()`, wrap every voice it returns — **four sites**: `:40` (`SayVoice`), `:44`, `:56`
and `:62` (`PiperVoice`) — as `return d.limited(synth.SayVoice{Voice: name}), nil` /
`return d.limited(synth.PiperVoice{Install: inst}), nil`. `app/radio.go` already imports `runtime`? — check; add
`"runtime"` if `renderSlots` is its first use. **Verify:**
`go test ./app -run 'Limited|Voice' -count=1 && go test ./app -race -count=1`

---

### Task 1.14 — P1 gate (the batch-exit checklist every batch follows)

```
go test ./domains/radio/... ./platform/config ./app -race -count=2 -timeout 120s   # a hang fails, never stalls the gate
make verify                                   # fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls
a2dh validate && make alloc-budget && golangci-lint run ./... && staticcheck ./...   # the standing NFR-7 gates (gates.md §1 owns the list)
A2DH=<framework build> make p10             # the framework's dist/a2dh (the installed CLI has no p10); the path lives in your shell, never in the tree
cp dist/p10.json 06_docs/02_features/multi-voice-support/07-readiness/p10-p1.json  # one record per batch, never overwritten
go test ./app -run DeclarationSet -update-declset                                   # app gained decls (cast.go); synth has no pin
git commit -m "multi-voice-support P1: Say limiter, Compose(Reports{}), the cast package, config pairs/tones, save preserves unknown keys"
```

- **P10 ledger (local, untracked `.a2dh-p10-exemptions.yml`; rows are `file · symbol · rule_id · reason`, ratified by
  the HUM LEAD at the gate):** rows are added **only from `dist/p10.json`**, never ahead of the tool (a row the
  tool does not raise is reported *unmatched* and fails the gate). Expected: a `domains/radio/cast` package P10-05
  density row (a pure package: `Resolve`/`Validate` carry real invariants, the rest return values) and reason
  refreshes on the existing package rows the diff re-activates (`platform/config` absorbs `keep.go`'s three
  invariant-free functions; `synth`, `app`). The `limited.Say → Voice.Say` name-graph twin anchors at the
  unchanged `voice.go` decl and raises nothing. Present the run's rows in `04-development/p1-build-log.md` under
  "Decisions for HUM LEAD" with the run's `tree_hash`.
- **The gate CLI is pinned:** record `a2dh version` (version + commit) of the framework build in the batch record
  (`07-readiness/gates.md`) at P1 and use that build for every batch — the ledger's symbol keys depend on the
  CLI's key scheme (a newer build keys methods `Recv.Name`); if the build changes, re-key the method rows in the
  same commit.
- **Build log:** `04-development/p1-build-log.md` — the pre-code spike list (go-toml `omitempty` on zero structs;
  `StrictMissingError` key paths), what landed, deviations from this plan, the gate table, the ledger decisions.
- **Commits carry no attribution trailers** (`make verify`'s watermark lint scans `main..HEAD` messages).
