---
title: "0.16.0 P3 entry — before-you-write-code gate: the audio merge"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "READY, with one item RE-RUN rather than inherited."
---

# BEFORE-YOU-WRITE-CODE GATE — P3, the main-track producer

**Re-run for this batch specifically.**  The BUILD-entry gate was emitted for S0, a spike that wrote no
code.  **P3 is the release's dangerous batch and inherits nothing** — the plan gives it its own red
team, a UAT shared with nothing, a timing property test, a dark path and a go/no-go, and a gate it
inherited rather than ran would be the write-only failure the gate exists to prevent.

```
BEFORE-YOU-WRITE-CODE GATE — P3
  [✓] Intent & plan understood
  [✓] Directives satisfied
  [✓] Dependencies pinned & compiling
  [✓] Library safety verified
  [✓] Abstraction spike done
  VERDICT: READY
```

## 1. Intent and plan — ✓, and it is NOT the plan's original row

**One sentence:** the rotation's reads stop driving the audio engine directly and travel the card path
through the narrator arbiter, so the two paths to speech become one — **without a window in which both
can speak.**

**S0 re-planned this batch**, and the row in `implementation-plan.md` is the re-planned one:

| Part | S0's finding |
|---|---|
| (a) the rotation's tune becomes a card | **WIRING.**  The arbiter serialises, suspends and resumes already, pinned by 8 green tests |
| (b) **the two relay-failure fallbacks** (`radio.go:789, 974`) | **THE REAL WORK.**  They bypass the schedule entirely today; a relay dying mid-broadcast must produce a CARD |
| (c) a third narration class below `narrateRead` | Safe by construction — the existing guard **walks** the class type |
| (d) `startSynth`'s direct path retires **in the same change** | **This deletion is what closes the window.**  Not a guard — a removal |
| (e) FR-2.5's no-double-speak property | **ASSERTED, not built.**  The arbiter provides it |

## 2. Directives — ✓

**FULL TDD applies in full here** (unlike S0, which was exempt by charter): the failing test comes
first, and a compile error is not a watched RED.  **FULL INST** governs every instrument added.
**FULL GIT**: this batch pushes on its first commit so CI sees both platforms while the work is warm.

## 3. Dependencies pinned & compiling — ✓

`go build ./...` exit 0 · 29 pinned requires · one `bubbletea` major · `go mod verify` all modules
verified.  **Unchanged since the BUILD-entry gate**, and re-checked rather than assumed.

## 4. Library safety — ✓, and this is the batch where it matters

**No new library.**  The risk is the JOIN, and it is the same join S0 characterised: a **pure**
`platform/lineup` handing work to a **mutex-guarded** `domains/radio/player`.

**Every P3 run uses `-race`.**  And per the plan, the merge carries a **property test over randomised
arrival timing**, because a scenario UAT targets precedence and duplication rather than races — the
safety lens's finding, not an invention here.

## 5. Abstraction spike — ✓ DONE, and its answer is the plan

**S0 answered it**: yes architecturally, and **the window is transitional, not inherent**.  What closes
it is retiring the direct path in the same change, not adding a guard.

**S0's own stated blind spot rides into this batch:** it was a STRUCTURAL answer plus a green known
case, and **it did not run the merged path, because that path did not exist.**  P3's timing property
test is still owed and is not made redundant by S0.

## Verdict

**READY.**  No item blocked.
