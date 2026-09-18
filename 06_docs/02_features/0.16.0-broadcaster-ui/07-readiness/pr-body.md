## Summary / BLUF

**Watchpost can now be operated as a station, not only run as one.** 0.16.0 adds the **Broadcaster
console** — a second full-screen surface beside the Observer — for a person putting hyper-local
weather on the air over a short-range radio channel: a line-up of location reports they can see and
reorder before it goes out, a priority track for alerts that always drains first, an ON AIR / STANDBY
control written in words as well as colour, a live relay bed under the programme, and the station's
own settings (a transmitter and a service radius). The operator who used to broadcast blind can see
what airs next, change it, and confirm at a glance whether they are transmitting.

The release also fixed what four blind reviewers found against its own first exit: a release workflow
whose cap would have produced a tag with no release, a fault class that could never reach the
operator on a live station, a privacy sentence the code did not keep, and a completeness claim that
was wrong by method. Every one is fixed under a fresh reviewer, and the record says what happened.

**Reviewer ask:** read `platform/lineup/fault.go` and `bed.go` first — the escalation and the
cut-over are where "an action shown as taken that was not taken" would live, and both were rewritten
under review.

## Problem & Solution Overview

### Problem Statement

*A person broadcasting hyper-local weather over a two-way radio channel cannot see what is about to
be transmitted, cannot change its order before it goes out, and cannot confirm at a glance whether
they are currently on the air — so they operate the station blind, and their listeners hear the wrong
thing at the wrong time.*

(Locked by the HUM LEAD 2026-09-09, 5/5; `01-objectives/problem-statement.md`; the body of issue #10.)

### Proposed Solution

A console for the operator, built on the Director 0.14.0 shipped and the inspectable surfaces 0.15.0
gave every gate. The schedule gains its write side — an operator can promote, demote, drop and
request — behind the same Director that owns the read side, so nothing on screen can disagree with
what goes to air. The station's state is one value the audio engine reports, not a UI flag. The bed
is a resource the Director cuts over to, never a second queue.

## Intended Outcomes / Value Impact

### Business Outcomes

An operator can see the running order, change it, and know whether they are on the air — the three
things the problem statement says they cannot do today. Their listeners hear location reports in
rotation, alerts the moment they arrive, and the relay underneath when the bed is on, and never two
things at once or nothing at all without the console saying so.

### Metrics of Success

| Metric Name | Symbol | Type | Definition | Measured In |
|-------------|--------|------|------------|-------------|
| Next-item certainty | **M1** | Primary | **Higher is better;** the operator names what airs next, unprompted, from the console alone. Target 100 %. *Not scored this release* — it needs randomised, unrehearsed prompts from the live schedule; the HUM LEAD's UAT ran both surfaces as a defect-finding pass and passed | Operator UAT |
| Time to correct the running order | **M2** | Primary | **Lower is better;** wall clock from intent to a changed running order with audio never interrupted. The capability landed (move and drop through the Director, the swap key refused on a live station); *not clocked this release* | Operator UAT |
| Unsafe mode switches | **M3** | Primary | **Lower is better;** target 0. The scripted half holds: `make pty-severe` drives the shipped binary through eight keystroke assertions; the ten deliberate mid-utterance switches the measure asks for are the operator's, and the thirty-minute rotation was run and signed off | Scripted PTY + operator UAT |
| Settings bleed | **M4** | Secondary | **Lower is better;** target 0 — a shared value survives a swap, a split value does not leak. *Partial:* `TestEverySettingsRowIsRuledForItsSurface` rules every row; the per-field round-trip the measure asks for is FR-6.2's open half | Automated |
| Silent overflow | **M5** | Secondary | **Lower is better;** target 0. **Met:** `TestNoSizeRendersPastTheTerminal` sweeps widths derived from the breakpoints, and the below-the-floor notice is asserted to fit | Automated size sweep |

## Scope of Change

**Touched:** a new Router as the program's model and the Broadcaster console (`modes/tty/router.go`,
`broadcaster*.go`, `request.go`, `setup_form.go`), the schedule's write side and the fault, bed and
cool-off rules in the Director (`platform/lineup/`), the executors and the read seam (`app/`), the
station's settings (`platform/config/broadcaster.go`), a coordinate-pair redaction at the radio
diagnostic's one writer (`platform/snapshot/redact.go`), the gate oracle as a Go package
(`tools/gateoracle/`), the mutant corpus (`06_docs/mutants/`, 172 → 380), the release workflow's cap,
and the docs (README with five console captures, CHANGELOG, `docs/`, `06_docs/`).

**Deliberately not in scope**, each a recorded row: tier three of the pool never reaching a slot
(**F-157**, 0.16.5); the fault run's two authors and a free-goroutine race (**F-159**, 0.16.5); four
requirements OPEN as recorded scope (**F-160…F-162**, FR-3.6 / FR-8.8 / FR-9.2, and FR-8.3 via
F-157); the P3(d) ruling on the dormant direct read path (**F-158**); the shell port (**F-156**) and
the history comments the lint cannot see (**F-137**).

## Caveats for the (Human) Reviewer

- **The release workflow's cap changed from 20 to 60 minutes**, derived from the Makefile's own
  40-minute mutant bound plus the measured remainder. The previous cap was under this branch's
  measured Linux verify; the tag would have existed with no release behind it.
- **The privacy sentence was corrected twice.** First the diagnostic was redacted at its one writer
  (every coordinate pair, whatever wrote it, gated by a test that reads the file back); then a blind
  reader found "never sent anywhere" overclaimed — the tower's pair goes to the National Weather
  Service in the same request every watched place makes. The README and the settings form say so.
- **The bed fix was iterated under review.** The first narrowing of a whole-struct write dropped the
  only place a landed tune cleared its ask, and every rotation raised a false stall; the second pass
  fixed it, watched red, three plants caught.
- **The exposure statement is a point-in-time measurement and says so.** Re-derive with
  `python3 scripts/quality/exposure-scan.py`. The five console captures carry the demo transmitter's
  position as pixels, ruled fine by the HUM LEAD.
- **This branch's history carries a 15 MB blob** from a bad edit, fixed forward. The release is a
  squash of the tree, so it does not reach `main`; the feature branch is deleted on origin after.

## Testing Done

- [x] Local code review completed
- [x] Unit tests added/updated
- [x] Integration tests pass
- [x] Manual testing performed

- **`make verify` — ALL GATES GREEN** on the VALIDATE-exit commit, read from the log; **`make quality`
  PHASE-EXIT GATES GREEN** (P10 0 live, 0 unratified, 153 ratified rows); **`a2dh validate` 100 %**.
- **The mutation sweep on the exit commit: 380 mutants — 375 CAUGHT, 5 SURVIVED by design, 0 NO
  EVIDENCE**, 4 h 28 min, its blind spot beside it (a race-only detector reads as SURVIVED).
- **Requirements traced by roster test name scoped to this release**: 31 by ID, 13 by a named test,
  4 OPEN with rows — after two reviewers found the ID-grep method colliding across releases.
- **Independent red teams at every exit, blind**: BUILD three rounds over the gate layer; REVIEW eight
  reviewers on four axes (5 convergent Criticals, all fixed) and three remediation reviewers across six
  passes to LGTM; VALIDATE two reviewers plus one on the bed fix (1 Critical, 4 Important, all fixed).
- **The install path exercised end to end on this machine**: five targets, the injector lint, the
  installer against a local server, the tamper control fired; repeated by a reviewer from a clean clone.
- **HUM LEAD UAT, 2026-09-18**: the Observer pass and the Broadcaster pass on a local build, both good;
  the thirty-minute rotation run and signed off.

## Screenshots

Five captures of the Broadcaster console are in `docs/img/` and the README: STANDBY with the line-up
and the pool, ON AIR with a read in progress, the Line-Up Request, managing a slot, the console's
Settings. Taken on the UAT build; a recapture on the 0.16.0-stamped build is the HUM LEAD's call, as
0.15.0 did.

## Additional Context

### Problem Statement Evaluation

The statement was locked at 5/5 and held through four phases: every scope argument in the release
traces to one of its three clauses (see, change, confirm), and the two safety-class fixes — the fault
band and the cut-over — are the "confirm" clause enforced against the code rather than the mock.

### Anti-Solution Check

The statement names no solution: it does not say "a second surface", "a Router" or "a console band".
Those are how it was answered, and one of them (the fault band reusing NFR-7's band rather than a
window) was ruled by the HUM LEAD against the author's first draft.

## For Agents

Read `06_docs/02_features/0.16.0-broadcaster-ui/00-REQUIRED-READING.md` first, and again after any
compaction — it is short and it is mandatory. Then `06_docs/red-team-brief.md` (every dispatch is
blind and uses the template), `06_docs/code-standards.md` (the tools state their ceilings), and the
two standing rules this release mechanised: everything in Go unless absolutely necessary, and test
code is a product. A fix is new code and gets a fresh reviewer; a plant's verdict is read from the
test's own line, never from a pipeline's exit.

## What

The Broadcaster console: a second surface for operating the station — the line-up, the alert track,
ON AIR / STANDBY, the relay bed, the station's settings — behind the Director that already owned the
schedule's read side. Answers issue #10.

## How to see it

`watchpost`, then `ctrl+b`. `shift+enter` toggles ON AIR / STANDBY; `↑` `↓` and `enter` open a slot;
`r` requests a place; `b` cuts the main track over to the bed. The README's Broadcaster section
carries the key table and five captures.

## Checks

- [x] `make verify` is green (fmt, vet, race, import direction, watermark gates) — on the VALIDATE-exit
      commit, read from the log
- [x] `golangci-lint run ./...` and `staticcheck ./...` are clean — baseline + ratchet, no new findings
- [x] Tests added or updated for the behaviour that changed
- [x] Docs touched where the behaviour is described (README, CHANGELOG, `docs/`, `06_docs/`)
- [x] No secrets, personal addresses, or machine-local paths in the diff — the identity gate covers the
      whole index, and every coordinate pair reaching the diagnostic is rewritten at its one writer

## Notes for the reviewer

**Carried, each with the reason it is carried:** F-157 (tier three never reaches a slot), F-158 (the
P3(d) ruling owed), F-159 (the fault run's two authors), F-160…F-162 (the OPEN requirements as
recorded scope), F-156 and F-137 (the shell port and the undetected history comments). All are
0.16.5's, the dedicated quality pass.

**Rulings taken in the record:** D-159 — a relay that falls through to synth under the operator's
cut-over releases the cut-over. A decline does not break a fault run. A `Run: 0` escalation is shown
by its reason alone.
