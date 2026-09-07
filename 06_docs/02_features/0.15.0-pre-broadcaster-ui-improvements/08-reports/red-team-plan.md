---
title: "0.15.0 — Pre-Broadcaster UI Improvements — RED TEAM, PLAN EXIT"
date: 2026-09-07
phase: PLAN (exit)
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
lenses: "4 axes + 5 personas + PLAN phase lens (Principal Architect) + Principal Engineer — 11, dispatched blind"
status: "NO-GO. The plan drew more Criticals than the discovery report it was built from."
---

# Red team — PLAN exit

## Bottom Line Up Front / BLUF

**NO-GO.**  Eleven lenses, blind to one another, scoped to PLAN exit.  The plan drew **more Criticals
than the discovery report did**, and the severe ones are not judgement calls — they are verifiable
facts about the codebase that the plan asserted wrongly.

**The plan's three load-bearing constructs are each broken in a different way, and all three break
for the same reason: a claim was made about the code without running a command against it.**

1. `closedset.Spec` **cannot express the check it was derived from** — no `Producer` field.
2. B1's falsification test **cannot falsify** — `tickerCategory` is the identity function.
3. B4's prescribed instrument **cannot see B4's change** — every `AllocBudget` test is a frame pin.

The third is the one to sit with.  FR-3.4 says a frame pin *"would have returned a green false pass
for a change the frame cannot see"* — and then names `make alloc-budget`, whose four tests are all
frame pins in `modes/tty`.  **The error was diagnosed and then committed, inside the correction.**

That is the **fifth** instance in one day of a single mistake: taking a green signal without asking
what produces it.  The lane guard's pass · F-30's 11/11 · #17's 52 ms · the 10-of-10 coverage
check · `make alloc-budget`.

## Convergence

| # | Finding | Lenses | Verified |
|---|---|---|---|
| **C-1** | **FR-1.1, FR-1.3, FR-1.4 have no batch**, and the plan's own validation certified "10 of 10" | **4** | ✔ `grep` — FR-1.2 is the only FR-1 in the implementation plan |
| **C-2** | **`closedset.Spec` has no `Producer`**, so two of its three promised failure modes collapse into one | **4** | ✔ the lane guard needs `LaneOf` *and* `tickerCategory`; `Spec` has one function |
| **C-3** | **No gate runs debug-tagged tests.**  `vet-tags` runs `go vet`, not `go test`.  FR-4's whole surface lands in a build nothing executes | **2** | ✔ `Makefile:37`; no `go test -tags watchpost_debug` anywhere |
| **C-4** | **B1's falsification test cannot falsify.**  `tickerCategory` is `return globalfeed.LaneOf(e)` — the two assertions are one expression | **2** | ✔ `app/ticker.go:630` |
| **C-5** | **B4's instrument cannot see B4's change.**  All four `AllocBudget` tests are in `modes/tty`; the memos are in the provider packages | 1 | ✔ zero pins in `firms`/`usgs` |
| **C-6** | **PL-D-5 sites the bound where a spoken read never goes.**  Reads go `Preview` → `playClip`, never `watch`; `playClip` already has a 10-min P10-02 bound | **3** | ✔ `watch` has one caller, inside `playPCM` |
| **C-7** | **FR-9.3 reverses a ratified ruling.**  `fault.go`: *"A DELIBERATE NON-DELIVERY IS NOT A FAULT (I-2)"* | 1 | ✔ |
| **C-8** | **The family taxonomy's membership is wrong.**  The honest Family-A call-site count is **two**, not five | **3** | ✔ one instance exists; `declset_test.go` and `contrast_test.go` appear zero times in the plan |
| **I-9** | `make verify` never runs `alloc-budget`, so "Gates green" — the exit condition of six batches — is vacuous for every allocation regression | 1 | ✔ `Makefile:172` |
| **I-10** | **No architecture diagram**, and the plan certified FULL DIAGRAMS conformance | 1 | ✔ |
| **I-11** | **B4's opening task is a no-op.**  The `t.Skipf` fires zero times today | 1 | ✔ 11/11, 0 skips |
| **I-12** | **"FR-10 contends with nothing" is false.**  FR-10.3 is UI and read-script text — the files B3 and B5 rewrite | 2 | ✔ |
| **I-13** | **FR-6.4 — a WCAG 2.2.1 Level A failure — sits in the last batch**, behind the only scope lever, in the slot a re-cut removes first | 1 | ✔ |

## Where the lenses disagree, and why that matters

Three lenses attacked the two-family split and reached **three different corrections**:

- *Re-cut at the assertion:* swap `Consumer func(F) M` for `Assert func(*testing.T, M, F)` and Family B
  becomes expressible.  The split is then a finding about a struct field, not about the domain.
- *Re-assign the members:* FR-8.2's AA register is set-membership over painted pairs — Family A, not
  B — and FR-1.2 is in neither family, because four classifiers return four different types.
- *Delete the generic entirely:* with FR-1.2 out and FR-4.2 stranded behind an unrun build tag, the
  helper has **two** honest call sites.  Write the second by hand, compare, extract then.

**They disagree about the fix and agree about the diagnosis:** the taxonomy's membership is wrong and
the helper is premature.  That agreement is the finding; the three fixes are the options.

## What survived

- **The hazard-path diagram** is the plan's best artifact; the junior-dev lens asked for it by name and
  it delivered — with one correction (TONE belongs downstream of the Director, not before it).
- **All three mermaid blocks parse clean** under mermaid 11.  The docs lens tried to break them and
  filed a refutation instead.
- **B5's `fakePlayer` hypothesis is correct** and well-aimed: `NewPlayer` starts the drain goroutine
  which sets `playing=false` on EOF, and `Play()` stores `true` afterwards with nothing to clear it.
  Cheap disproof before hardware time.
- **Estimate anchoring was NOT committed**, and creditably: sizes are relative with the driver named,
  and FR-10 was left unsized rather than invented.
- **PL-D-4's boundary holds.**  `Save` genuinely owns its mirror; the god-object was avoided.

## The PLAN anti-patterns, adjudicated

| # | Anti-pattern | Verdict |
|---|---|---|
| 1 | Strawman alternatives | **Not committed** — two rejected options were named with reasons |
| 2 | Architecture without diagrams | **Partially committed** — three diagrams, but no architecture diagram, and conformance was certified |
| 3 | Placeholder code | **Not committed** — contracts are signatures with `TBFI` markers |
| 4 | Missing ordering/dependencies | **Not committed** — critical path and parallelism are explicit |
| 5 | **Approval without internal validation** | **COMMITTED.**  The validation asserted 10/10 against a mapping the implementation plan contradicts, and substituted a bespoke checklist for the named `plan-document-reviewer-prompt` |
| 6 | **Deviations not captured** | **COMMITTED** — metric D's deferral, and the "size per FR" → "size per batch" substitution |
| 7 | Mitigations not tied to approach | **Not committed** |
| 8 | No decision log | **Not committed** — seven decisions with revisit conditions |

## An overstatement, corrected rather than repeated

The infosec lens reported PII **in the shipped binary** and called it a live violation of constraint 4.
The string is present in `dist/watchpost` — confirmed — but it is a **comment in a pronunciation
rules file** showing the worked example `"Oceanside, CA" → "Oceanside, California"` for
state-abbreviation expansion.  That is a plausible generic example for a California expansion rule,
not a disclosure on its own.  **Not repeated as a violation.**

What survives from that lens, and is right: FR-7.5 is scoped entirely to `06_docs`, and a PII scan
that never looks at `dist/watchpost-*` is scoped wrong regardless.

## Conditions for GO

None require re-planning.  Seven are documentation edits against measured facts; two are decisions.

| # | Condition | Kind |
|---|---|---|
| 1 | FR-1.1/1.3/1.4 each get a batch or a written descope; the validation count is derived, not asserted | Edit |
| 2 | `Spec` gains `Producer`, or the helper is deferred (see the decision below) | **Decision** |
| 3 | A `go test -tags watchpost_debug` target enters `verify`, as a **B3 entry condition** — not B6 work | Edit |
| 4 | B1's opening task changes; the lane guard's consumer is the identity and cannot falsify | Edit |
| 5 | B4's instrument becomes *a provider-package allocation pin that has been watched to fail*, created as an entry condition | Edit |
| 6 | `alloc-budget` joins `verify`; every batch names its gate set once | Edit |
| 7 | PL-D-5 is re-sited or withdrawn; FR-9's bound belongs where a read actually runs | Edit |
| 8 | FR-9.3 is reworded against I-2, or I-2 is explicitly re-opened as a HUM LEAD ruling | **Decision** |
| 9 | The family diagram is corrected, or the helper deferred; PL-D-1's DRY count is restated against two | Follows from 2 |

## Cost

Eleven lenses, ~1.19M subagent tokens, 322 tool calls, ~7 minutes wall-clock in parallel.

**Cost per defect:** eight Criticals, every one of them a factual error about the codebase that a
single command would have caught, and that no amount of re-reading by the author did catch.  Two
rounds of blind red-teaming have now found the same author making the same mistake five times in one
day — which is a stronger argument for the practice than any of the individual findings.
