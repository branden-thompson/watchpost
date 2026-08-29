# P4 — Verification, the reused `[A]` path, docs, gates

Depends on P1–P3. This batch closes NFR-6 on the path the window shares with `[A]`, machine-verifies the
journeys in a real PTY, and lands the user-facing docs.

---

## Task 4.1 — `render.Plain` on every `[A]` field (NFR-6, red-team S1)

**File:** `modes/tty/alerts.go` (MODIFY), `modes/tty/alerts_test.go` (MODIFY)

**Test first (RED):** `modes/tty/alerts_test.go` (append)
```go
// Every provider-supplied field the [A] record renders must be stripped of
// terminal escapes — before 0.13.0 only Event and Description were (red-team
// S1). The OSC-52 payload is the clipboard-write class the S-F6 rule exists for.
func TestAlertRecordStripsEscapesFromEveryField(t *testing.T) {
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[31mz\x1b]0;title\x07"
	a := snapshot.Alert{Event: evil, Severity: evil, Urgency: evil, Certainty: evil, AreaDesc: evil, Headline: evil, Description: evil, Instruction: evil,
		Effective: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC), Expires: time.Date(2026, 8, 28, 13, 0, 0, 0, time.UTC)}
	d := dash(t).(Dashboard)
	o := d.opts()
	for _, line := range alertRecordLines(o, a, 60, time.UTC) {
		plain := stripANSITest(line) // the renderer's OWN tints are allowed; provider bytes are not
		if strings.ContainsAny(plain, "\x07") || strings.Contains(plain, "]52;") || strings.Contains(plain, "]0;") {
			t.Fatalf("provider escape survived: %q", line)
		}
	}
	// The compact line and the module title too.
	loc := &snapshot.Location{Label: "L", Alerts: []snapshot.Alert{a}}
	if line := d.alertCompactLine(o, loc, a, 100); strings.Contains(stripANSITest(line), "]52;") {
		t.Fatalf("compact line leaks: %q", line)
	}
}
```

**Code:** `alertRecordLines` in full (`alerts.go:198-236` today):
```go
// alertRecordLines formats one alert's full record for the modal. Every
// provider field passes through render.Plain (S-F6): before 0.13.0 only the
// event and the description did (red-team S1).
func alertRecordLines(o render.Opts, a snapshot.Alert, wrapW int, in *time.Location) []string {
	tone := modalAlertTone(a)                      // UAT 28.3/28.4 modal text tones
	head := strings.ToUpper(render.Plain(a.Event)) // provider text never addresses the terminal (S-F6)
	meta := fmt.Sprintf("[%s · %s · %s]", render.Plain(a.Severity), render.Plain(a.Urgency), render.Plain(a.Certainty))
	out := []string{"  " + render.TintRaw(head, "1;"+tone) + "  " + meta} // bold title (UAT 28.5)
	start, end := a.Effective, a.Expires
	if a.Onset != nil {
		start = *a.Onset
	}
	if a.Ends != nil {
		end = *a.Ends
	}
	timing := "  Starts " + start.In(in).Format("Mon 01/02 3:04 PM") // the location's clock (F17)
	if end.After(start) {
		timing += "   Ends " + end.In(in).Format("Mon 01/02 3:04 PM") +
			fmt.Sprintf("   (~%s)", end.Sub(start).Round(time.Hour))
	}
	out = append(out, timing)
	if a.AreaDesc != "" {
		out = append(out, wrapPrefixed(o, "Area: "+render.Plain(a.AreaDesc), wrapW)...)
	}
	if a.Description != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, render.Plain(a.Description), wrapW)...)
	}
	if a.Instruction != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, "Instructions: "+render.Plain(a.Instruction), wrapW)...)
	}
	// UAT 55: body text (everything below the toned title) reads white for
	// contrast - advisories and alerts earn it.
	for i := 1; i < len(out); i++ {
		if out[i] != "" {
			out[i] = render.Tint(out[i], render.Tok(render.AlertModalText))
		}
	}
	return append(out, "")
}
```
(the vestigial `i` parameter and its "paging lives in the modal title now" history comment are dropped —
AP-DEAD-01/AP-HIST-01; update the one caller at `alerts.go:161`) and at the two
`fmt.Sprintf("[%s] %s", a.Severity, a.Headline)` sites (`alertArea` `:283`,
`alertCompactLine` `:306`) use `render.Plain(a.Severity)`, `render.Plain(a.Headline)`. `alertBlocks` (`:32-33`) already strips `Event`/`Description`.

**Verify:** `go test ./modes/tty -run 'TestAlertRecordStrips' -v`

---

## Task 4.2 — Escape-injection fixture on the window path (end to end)

**File:** `modes/tty/severe_test.go` (append)
```go
func TestSevereWindowNeverForwardsProviderEscapes(t *testing.T) {
	// The domain strips at RecordOf; this pins the TTY never re-introduces a raw
	// field: Product/Location/Declared come from the app already Plain'd, but a
	// hostile SevereRow (a future mapping bug) must still not reach the terminal.
	evil := "x\x1b]52;c;aGVsbG8=\x07y"
	m := dash(t)
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: []SevereRow{{Key: "k", Tab: SevereWarnings, Product: evil, Location: evil, Declared: evil, Expires: evil, Record: SevereRecord{Title: evil, Meta: evil, Timing: evil, Area: evil, Paras: []string{evil}}}}, Totals: [severeNumTabs]int{1}})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	for _, view := range []string{m.View().Content, func() string { mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); return mm.View().Content }()} {
		if strings.Contains(view, "]52;") || strings.Contains(view, "\x07") {
			t.Fatal("a provider escape reached the frame")
		}
	}
}
```
**Code:** already in place — `severeBrowseLines` Plains the four cells when it builds `render.SevereCell`s
(P3 Task 3.5) and `severeDetailLines` Plains every record field at the point of use (P3 Task 3.6); this task
only adds the end-to-end fixture above.

**Verify:** `go test ./modes/tty -run 'TestSevereWindowNever' -v`

---

## Task 4.3 — PTY journeys (NFR-9)

**File:** `scripts/quality/severe-modal.expect` (CREATE, `chmod +x`)
```tcl
#!/usr/bin/expect -f
# severe-modal.expect — the severe-events window's keyboard journeys on a REAL pty (0.13.0, NFR-9):
#   1. w  → the window opens (its title appears) → enter → the record → esc → the table → esc → closed
#   2. ctrl+s (0x13) → the window opens (the alias is delivered in raw mode; an outer tmux/`stty ixon`
#      layer is a documented limitation this script cannot exercise — it owns its own pty)
# Usage: scripts/quality/severe-modal.expect [./dist/watchpost]
# Exit 0 on success; a failed expect prints which step and exits 1.
set timeout 10
set bin [lindex $argv 0]
if {$bin eq ""} { set bin "./dist/watchpost" }
log_user 0
proc step {name script} {
    if {[catch {uplevel 1 $script} err]} { puts "FAIL $name: $err"; exit 1 }
    puts "ok   $name"
}
spawn env WATCHPOST_ASCII= $bin
catch {exec stty rows 44 cols 133 < $spawn_out(slave,name)}
step "launch" { expect -timeout 30 "W A T C H P O S T" {} timeout { error "no header" } }
expect -timeout 60 -re "\\d+ºF|Setup" {}
step "w opens the window"        { send "w";     expect "SEVERE WEATHER / DISASTER EVENTS" {} timeout { error "window did not open" } }
step "enter opens a record or the empty state stays" {
    send "\r"; expect -re "\\[esc\\] Back|no active events" {} timeout { error "neither a record nor the empty state" }
}
step "esc backs out"             { send "\x1b";  expect -re "Total Category Events" {} timeout { error "table not restored" } }
step "esc esc closes"            { send "\x1b";  expect -timeout 5 -re "W A T C H P O S T" {} ; sleep 0.5 }
step "ctrl+s alias opens"        { send "\x13";  expect "SEVERE WEATHER / DISASTER EVENTS" {} timeout { error "ctrl+s not delivered" } }
step "esc closes"                { send "\x1b"; sleep 0.3 }
step "quit"                      { send "q"; expect -timeout 15 eof }
```
Wire into `Makefile` as `make pty-severe` (documented beside the other quality scripts) and into the release
checklist (`07-readiness/`). The Linux run is HUM LEAD's half (same command on Arch).

**Verify:** `make build && scripts/quality/severe-modal.expect` — 8 `ok` lines on macOS.

---

## Task 4.4 — User-facing docs

**Files:** `README.md`, `CHANGELOG.md`

**README key table** (`README.md:72-75` region) — add after the `A` row:
```
| `w` (alias `ctrl+s`) | **Severe Weather / Disaster Events** — every active warning, watch, advisory, statement, significant quake and tropical cyclone in six tabs; `←` `→` category, `↑` `↓` row, `enter` the full report, `esc` back / `esc` `esc` close |
```
and a short paragraph under the ticker section: what the window lists (the world's active severe events
plus your watchlist's advisories and statements), that on a calm day it says so, and that the spoken alert
now says "press W". **Documented commands execute:** paste the key row only after the PTY journey passes.

**CHANGELOG** — `## 0.13.0` entry: the window (six tabs, in-place record, `w`/`ctrl+s`), storm names on
the ticker and in narration, the narration re-point, the guarded superseded rule and id normalisation
(fixes duplicate rows between the national feed and a tracked location), `render.Plain` on every `[A]`
field, the modal memo (every open window rebuilt only on change), bounded location alerts, `seen.json`
permissions. Attribution line for the feeds unchanged (all public domain).

**Verify:** `a2dh validate` 17/17; read the README prose (README Content Audit): every key/flag named
exists in `defaultKeyMap`.

---

## Task 4.5 — Readiness: gates and the Linux half

**File:** `06_docs/02_features/severe-alerts-modals/07-readiness/gates.md` (CREATE)

Checklist the BUILD exit and SHIP read:
- `make verify` **and `make p10`** green on macOS (the P10 gate is not folded into `make verify` — red-team PLAN B-6; folding it in is a 0.13.x chore); `go test -race ./...` green on macOS **and** the Linux run (the 0.12.0 lesson:
  a green macOS suite is not green); `GOOS=linux go vet ./...`.
- `make alloc-budget`: closed-frame rows unchanged; `Severe` rows pinned at measured × 1.05; Help drops
  to the hit path.
- PTY: `scripts/quality/severe-modal.expect` (`make pty-severe` — new, Task 4.3) on macOS (agent) and Arch (HUM LEAD).
- R6: `WATCHPOST_LIVE=1 go test ./app -run LiveRelay` on macOS (agent) and Arch (HUM LEAD) — the narration
  strings changed; relay/audio paths must not.
- 1-hour PPROF soak at 133×44 with the window held open (`quality-baseline.md:98` procedure): total
  allocations vs 111 M; the window at rest ≈ 0.
- COV: `go test ./domains/globalfeed -run TestRenderListCoverage -v` prints 100 % per class.
- Goldens at 80/100/120 + `--ascii` reviewed by HUM LEAD (the colour pass happens here — UAT).
- `seen.json` cap (20 000, oldest-evicted): a flood from a feed itself could evict live ids and re-sound a tone; bound = the feeds are TLS origins we trust for content, and 20 000 ≫ a day's volume — accepted (red-team PLAN S5).
- Owed from 0.12.0 and still owed: the Linux R6 half — recorded, not silently dropped.

**Batch exit:** `make verify`; commit `feat(severe): NFR-6 on the [A] path, PTY journeys, docs, gates`.
Then BUILD exit: red-team (full lens set + Perf/A11y/InfoSec/JuniorDev), build-report, PRESENT.
