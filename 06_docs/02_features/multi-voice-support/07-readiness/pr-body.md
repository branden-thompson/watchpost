## Summary / BLUF

Watchpost Radio gets a **cast of correspondents** and, more importantly, **a sound that says what kind
of alert is coming before a word is spoken**. A listener who is not looking at the screen can now tell
an alert from a routine report by ear alone. Alongside it: a read order that puts the most serious,
closest hazard first and *speaks* the remainder rather than dropping it; a marine report for coastal
locations; a window when the radio goes quiet; and Setup grown into Settings.

Three defects found during this work could each have cost a listener a hazard — a tone mute that
permanently silenced every spoken alert, Emergency Orders that could never be read, and every hazard
active at launch being marked as already-heard. All three are fixed here.

**Reviewer ask:** review the alert path first (`app/read_script.go`, `platform/lineup/`,
`domains/radio/cast/`) — everything else is recoverable, and a missed hazard is not.

## Problem & Solution Overview

### Problem Statement

*A Watchpost listener who is not looking at the screen cannot tell, by ear, that what has just started
speaking is an alert rather than one of the routine reports.*

(Locked at DISCOVER exit, 2026-08-29; `08-reports/project-brief.md`.)

### Proposed Solution

Give every alert class its own tone, sounded **before** the words, so the class is identifiable in the
first moment of audio without looking. Give the station a cast of correspondents, so the *voice* also
carries meaning — the alerts read in a different voice from the reports — with hand-overs announced by
name mid-broadcast. Make the read order deliberate: most serious, closest first, capped at five per
burst, with anything beyond the cap spoken rather than silently dropped.

## Intended Outcomes / Value Impact

### Business Outcomes

A listener working with Watchpost in the background hears a hazard and knows what kind it is before
the sentence starts, and never loses one to a busy feed, a mute, or an app that was closed when the
alert arrived. The failure mode this release is really about is the silent one: an alert that exists,
is displayed, and is never spoken.

### Metrics of Success

| Metric Name | Symbol | Type | Definition | Measured In |
|-------------|--------|------|------------|-------------|
| Time to tone start | TTS | Primary | **Lower is better;** milliseconds from a breaking event being admitted to its class tone sounding. Budget ≤ 250 ms absolute, and the code path pinned at ≤ the 0.13.0 median + 50 % | `WATCHPOST_DEBUG_RADIO` log (`breaking:` → `tone:`); `BenchmarkTimeToToneStart` |
| Alert class named by ear | EAR | Primary | **Higher is better;** alerts whose class a listener names correctly from the tone alone, with the words stripped. Pass ≥ 9/10 | M3 ear test, `07-readiness/validate/m3.md` |
| Keypresses to assign a correspondent | A2V | Secondary | **Lower is better;** keystrokes from the dashboard to "role X speaks in voice Y". Measured at **6**, against a pin of ≤ 11 | `TestM2TheKeypressesToAssignACorrespondent` |
| Frame allocations | ALLOC | Maintenance | **Lower is better;** allocations per rendered frame, pinned at the measurement × 1.05, with no new per-tick work | `make alloc-budget` |
| Known vulnerabilities | CVE | Maintenance | **Lower is better;** count of known CVEs in the dependency set | `govulncheck` in `make verify` |

## Scope of Change

**Touched:** the radio (`domains/radio/{cast,synth,player,script,stream}`), the alert pipeline
(`app/`, `platform/lineup/` — a new pure Director), the terminal UI (`modes/tty/` — Settings, the
`[S]` status window, the severe window), the marine domain (`domains/marine/`), and the docs
(README, CHANGELOG, `docs/where-things-happen.md`, `docs/extending.md`).

**Deliberately not in scope**, each a recorded ruling rather than an omission: the resident Piper
backend (deferred to 0.15.0 behind an RSS measurement taken on Linux after this release ships); the
on-air chip; the `transition/` script, which is written and pinned but not yet on air; and the alert
**injector**, which is compiled out of release builds by build tag rather than gated at run time — a
screenshot of a fabricated tornado warning is indistinguishable from a real one, so the capability is
absent from a shipped binary rather than switched off inside it.

## Caveats for the (Human) Reviewer

**Look hardest at the alert read path.** `app/read_script.go`'s `openTone` is where a tone can sound
with nothing following it; `platform/lineup/` decides what is read and in what order; `app/severe.go`
scopes which events reach a listener at all. A defect anywhere in that chain is silent by nature.

**Three things are shipping known and recorded, not hidden:**

1. **F-43** — a listener heard three tones with no words, once, on a live burst. Not reproduced since,
   and **not diagnosed**. What ships is the missing invariant (a sounded tone must be followed by
   words — nothing pinned that before) and a trace line, so the next sighting names its own cause
   instead of needing a reconstruction.
2. **One PTY journey step is deliberately red.** A `[space]` event read takes ~11.4 s from keypress to
   first audio on the development machine. There is no budget for that path — NFR-2's 250 ms covers
   the *breaking* tone — so the step is left failing rather than having its bound widened, which would
   turn a measured number into a green tick.
3. **F-40** — a real evacuation order and its fire were both invisible during UAT. That is a data/feed
   question rather than this release's read path, and it gets its own investigation.

**The gates could not see most of what mattered.** A red-team round found roughly thirty of a previous
round's own remediations defective, and every one had passed `make verify`, the P10 gate, the
allocation pins and the goldens. Please read the code rather than the green ticks.

## Testing Done

- [x] Local code review completed
- [x] Unit tests added/updated
- [ ] Integration tests pass
- [x] Manual testing performed

**Evidence, rather than assertions:**

- `make verify` — **ALL GATES GREEN** (fmt, vet, vet-tags, tidy, govulncheck, `-race` across 44
  packages, import direction, watermark, gate controls, mutation-harness corpus), run cold on the
  final tree.
- **Mutation testing** — 189 mutants apply and compile; the harness self-test pins that a crash is
  CAUGHT rather than SURVIVED, that a flaky failure is not read as CAUGHT, and that an unattributable
  failure is INVALID rather than evidence either way. Durable record: `07-readiness/mutant-check.txt` (promoted from `dist/` at hygiene; `dist` is emptied and `*.log` is git-ignored).
- **Integration (the PTY journey) is UNCHECKED above, deliberately.** Every step passes except the
  `[space]` read described in the caveats. Leaving the box unchecked is the honest state; checking it
  would require widening a bound around a measured 11.4 s.
- **Manual/UAT** — multiple live sessions against real weather by the maintainer, which produced four
  defects no amount of reading found, three of them fixed here.

## Screenshots

**All sixteen README images are this release**, captured on this build (`v17e6a23`) and none older.
That mattered more than a refresh: every image was 0.13.0-era and taught a UI that no longer exists —
the masthead read `WATCHPOST v0.13.0`, the menu offered `s Setup` where the window is now Settings,
and the radio row still showed `M Mute Severe Alerts`, `V Voice:` and `T Size:` — the `[M]` semantics
this release *changed* and the two controls NFR-8 required to be gone. No grep could see it; by then
it is pixels.

New or rebuilt here: `themes.gif` replaces a still and cycles twelve palettes live (the caption
always said "applied live"); `severe.gif` walks all eight tabs where the old one walked seven;
`breaking.gif` finally shows what its caption promised — a takeover, then `w` opening the window on
*that* category with the event as row 001; `light.gif`, `responsive.gif`, `radio.gif`,
`event-read.png`, `relay.png`, `maritime.png` (new — the marine report had no image) and
`relay-fault.png` (new — the "radio has gone quiet" window, a headline feature with no image until
now).

One privacy fix rides with them: `status.png` had carried an absolute path containing the
maintainer's account name in the published README since 0.13.0. The app now abbreviates a
home-prefixed path to `~/…` (`app/dump.go`), pinned by a test confirmed to fail without it, so every
future capture is safe by construction.

## Additional Context

- `06_docs/02_features/multi-voice-support/08-reports/` — DISCOVER, PLAN, BUILD and REVIEW reports,
  plus four red-team rounds.
- `06_docs/quality-observations.md` — the recurring failure shapes this release found, kept because
  they outlive it.
- `06_docs/follow-ups.md` — every carried item, with what is known and what is not.

### Problem Statement Evaluation

- **Bad outcome:** yes — a hazard is spoken and not recognised as a hazard.
- **Affected humans:** yes — anyone using Watchpost without watching it, which is its stated purpose.
- **Technology agnostic:** yes — it names no mechanism; tones, voices and ordering are all solutions.
- **Non-prescriptive:** yes — it does not say "add tones".
- **Verifiable:** yes — the M3 ear test measures exactly the stated inability, with the words stripped.

### Anti-Solution Check

**Yes, and it changed the measurement.** *Time to tone start* can be satisfied without solving the
problem: a fast tone that every class shares tells a listener something is happening but not *what*,
which is the actual complaint. That is why EAR is a separate primary metric and why it is measured
**with the words stripped** — a listener who can name the class from the tone alone has had the
problem solved; one who needs the sentence has not.

A second one, found late: a sounded tone proves nothing if no words follow. That failure was
observable and unpinned until this release, and is now an invariant (F-43).

## For Agents

<details>
<summary>Structured context for LLM / agent reviewers</summary>

- **Intent:** make an alert identifiable by ear before any word is spoken, and make the order and
  completeness of what is read deliberate rather than incidental.
- **Invariants:** a sounded tone is followed by words; an alert active at launch is announced, not
  marked read; alerts past the per-burst cap are spoken, never silently dropped; Emergency Orders
  lead and are exempt from the cap; the tone mute silences tones only, never the words; the injector
  is absent from a release binary by build tag, asserted structurally.
- **Files of interest:** `app/read_script.go`, `app/ticker.go`, `platform/lineup/`,
  `domains/radio/cast/`, `app/severe.go`, `modes/tty/setup_cast.go`.
- **How to verify:** `make verify` · `make journey` (one step is expected red — see caveats) ·
  `go test -tags mutants -v ./06_docs/mutants`.
- **Risk notes:** the read path fails silently by nature; assertions that pass for the wrong reason
  were the dominant defect class in this release, so check what an assertion would do if the thing it
  checks were broken. Live-feed tests can pass against the marquee rather than the window under test.

</details>
