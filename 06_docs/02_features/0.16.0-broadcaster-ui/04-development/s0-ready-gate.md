---
title: "0.16.0 BUILD entry — before-you-write-code gate for S0 (the spike)"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "READY"
---

# BEFORE-YOU-WRITE-CODE GATE — S0, the audio-merge spike

**Run at BUILD entry, before any line of code.**  Emitted as its own artifact because the last release
shipped the gate and never produced its own verdict — the write-only failure mode the gate exists to
prevent.

```
BEFORE-YOU-WRITE-CODE GATE — S0
  [✓] Intent & plan understood
  [✓] Directives satisfied
  [✓] Dependencies pinned & compiling
  [✓] Library safety verified
  [✓] Abstraction spike done
  VERDICT: READY
```

## 1. Intent and plan — ✓

**One sentence:** the spike answers whether the rotation's reads can travel the card path —
`Propose → BuildCard → Speak` through the narrator arbiter — instead of
`Tune → startSynth → engine.StartSource`, **without a window in which both paths can speak.**

Approved plan: `03-architecture-design/spike-the-merge.md`, chartered by **D-25**.  Timeboxed to one
session, isolated to its own worktree, three named outputs, code deleted afterwards.

## 2. Directives — ✓, with one stated exception

| Directive | For S0 |
|---|---|
| **FULL GIT** | Branch exists; **the spike works in a separate worktree and commits nothing to the feature branch** |
| **FULL TDD** | **Deliberately N/A for the spike, and this is the exception worth stating.**  A spike's output is an ANSWER, not a shipped unit; its code is deleted by charter.  Writing tests first for code that will be deleted tests nothing that survives.  **TDD applies in full from P0 onward**, and any probe the spike proves load-bearing is **re-authored** as a real test in P3 rather than promoted from spike code |
| **FULL INST** | Applies to what the spike REPORTS: any number it publishes states its blind spot (INST-5), and it answers a KNOWN case before an unknown one (INST-4) |
| **FULL DOCS / REPORTS** | The three outputs land in `04-development/` |

## 3. Dependencies pinned and compiling — ✓

| Check | Evidence |
|---|---|
| Whole tree builds | `go build ./...` — **exit 0** |
| Requires pinned | **29** pinned requires; no `latest` |
| No duplicate majors | **`charm.land/bubbletea/v2 v2.0.3` only** — the go-top failure was v1 and v2 in one `go.mod`; not present here |
| Modules verified | `go mod verify` — *all modules verified* |

## 4. Library safety — ✓, and it is the spike's own subject

**No new library is introduced.**  The spike spans two existing packages, and their concurrency shapes
are opposite — which is precisely the seam it must characterise:

| Package | Shape |
|---|---|
| `platform/lineup` | **Pure.**  No goroutines — verified by grep, and consistent with its documented "one caller, pure `Step`" design |
| `domains/radio/player` | **Mutex-guarded** — `mu sync.Mutex` and `startMu sync.Mutex` (`engine.go:66,79`) |

**So the risk is not inside either package; it is at the join** — a pure state machine handing work to
a lock-guarded audio owner.  **`-race` is confirmed working** on `platform/lineup`, and every spike run
uses it.

## 5. Abstraction spike — ✓ by definition

**S0 IS the spike**, chartered exactly for the highest-leverage anti-pattern: do not depend on assumed
behaviour of an unfamiliar path.  The assumption under test is that the arbiter can own the rotation's
audio without a double-speak window.  **The plan says it "should be mostly wiring"; the spike exists to
find out before three batches are sunk on that belief.**

## Verdict

**READY.**  No item blocked.
