---
title: "0.15.0 — Pre-Broadcaster UI Improvements — REVIEW REPORT"
date: 2026-09-08
phase: REVIEW
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "Awaiting HUM LEAD approval to exit REVIEW and enter VALIDATE"
---

# 0.15.0 — Pre-Broadcaster UI Improvements — REVIEW REPORT

## Bottom Line Up Front

**REVIEW found a listener-facing blocker, and then found that an accepted limit was not a limit.**

The blocker: the radio stated *"There are currently no named incidents within a 31 mile radius of your
area"* as a **fact**, while the only source of named incidents had not answered — one sentence after
crediting that source by name. This is the distinction the HUM LEAD's own UAT forced **on screen**
(*"fire feed not yet available"* is not *"none within this radius"*), recurring **per feed instead of
per ring**, and **on the air instead of on screen**.

The reversal: the `FetchEach` limitation carried out of BUILD — that a transiently-failed location
cannot be told from a permanently uncovered one — **was not a limitation.** `httpx` already
distinguishes "the service answered 404" from "we could not reach it", and nothing consumed it.

**Recommendation: exit REVIEW.** `make verify` ALL GATES GREEN. No outstanding correctness item.

## 1. Gate results

`make verify` — **ALL GATES GREEN** at `4a1ae6b`. 14 gates, 171 mutants across 32 target files.
`make journey` remains **RED at 26 of 28** by the HUM LEAD's F-58 ruling, documented rather than
absorbed.

## 2. Requirements verification

| NFR | Verified by |
|---|---|
| NFR-1 — nothing regresses the frame path | `make alloc-budget`, the tty pins pass |
| NFR-2 — the injector is absent from release artifacts, provably | `lint-injector` on the shipped matrix, and inversely on `build-diag` |
| NFR-3 — every gate carries a watched failure | Promoted into FR-8; landed there |
| NFR-4 — both platforms in CI while work is in progress | `os: [ubuntu-latest, macos-latest]` |
| NFR-5 — a test event is indistinguishable in path, unmistakable in presentation | UAT, injected tornado takeover |
| NFR-6 — WCAG 2.2.1, no hard timeout on a control | FR-6.4's `held` stops the countdown once a listener starts choosing |

Every FR landed; see the BUILD report §1 for the per-FR outcome.

## 3. Stress testing

**One finding.** Below the minimum terminal size the app does not degrade, it **overflows silently**:
at 20x5 the frame renders 57 cells wide and 25 lines tall, and 1x1 and 40x10 produce the *identical*
frame — nothing consults the size below the floor. It does **not** panic: every window renders at
24x6, and an empty location and a nil snapshot both render. So F-55's size contract owes a **notice**,
not crash-proofing. Recorded there, whose own measurements were about scroll reachability *at* 80x24.

**One non-finding, reported as such.** Seven hostile provider inputs — escapes, 5,000 characters,
newlines, 200 combining marks, an RTL override, NULs, wide CJK — all clamp to <= 133 in a 133-wide
frame. The first run showed the combining case at **333 cells** and I nearly reported it; isolating
showed the probe set the label directly, bypassing the assembler that cleans it (201 runes -> 3).

## 4. Documentation

The README told a listener the update check asks GitHub *"once an hour"*; FR-7.1 made it once at
startup, and the config **comment three lines below it** had been corrected while the prose was
missed. Three source comments said the same (`release.go`, `config.go`, `dashboard.go`).

**`155 distinct targets` was published in two documents and is not derivable from the tree** — the
figure is 171 mutants across 32 target files.

0.15.0 had **no CHANGELOG entry**; added as `[Unreleased]` and undated deliberately. The 0.14.0 entry
still says "once an hour" and is **left alone**: it was true when written, and rewriting history to
match the present is how a changelog stops being evidence.

## 5. Critical analysis — the REVIEW red team

Two blind agents on the A2DH axes (code-quality + safety-critical, docs-quality + project-hygiene),
scoped to listener-facing behaviour and to the written record.

| # | Finding | Disposition |
|---|---|---|
| **F1** | **BLOCKER** — a zero count spoken as fact for a feed that never answered | Fixed: each provider stamps the half it serves; a zero is spoken as fact only when its own feed answered |
| **F2** | The freshness window **rounded down** — 47 hours read "in the last day" | Fixed: rounds up, so the error runs OLD rather than fresh |
| **F3** | A partial outage stamped the location the request never reached | Fixed by §6 |
| **F4** | A re-added location inherited its old stamp and read `n/a` before any fetch | Fixed: `SetLocations` clears the attempt record; also stops a per-removal leak |
| **F5** | #13's single-location hole | **Closed** by §6, not carried |
| docs | `155 distinct targets`; three "hourly" comments; a stale corpus size | Corrected |

**Two of the three F1/F2/F4 fixes had no test that could catch them**, and planting found that rather
than reasoning. The composer's test builds a `FireReport` **by hand** and sets the flags itself, so a
plant hardcoding both to `true` passed it.

## 6. The limit that was not a limit

BUILD carried a documented limit: `FetchEach` joins per-location errors, so a location that
**transiently failed** could not be told from one **permanently uncovered**. The consequence was a
false `n/a` during a partial outage, and shimmer-forever for a single-location watchlist.

**`httpx` already carried the discriminator.** `StatusError.HTTPStatus()` returns the code;
`ReachError` returns `0`. Nothing consumed it.

| Case | Now | True? |
|---|---|---|
| 404, outside the forecast area | stamped -> `n/a` | yes — the service answered |
| Connection refused | not stamped -> shimmer | yes — we could not ask |
| Total outage | nothing stamped -> shimmer | yes |
| Partial outage | served stamped, refused keeps waiting | yes |

`snapshot.Unreachable` duck-types `interface{ HTTPStatus() int }` rather than importing `httpx`, so
the dependency is not inverted. An error carrying no status counts as unreachable: **"we do not know"
keeps the row waiting, and asserting an absence we cannot support is the one outcome this product
should never choose.**

## 7. The process finding worth carrying

**Three separate times this release, a test supplied the producer's own output and therefore could
not detect a missing or wrong producer:**

1. The assembler's `#13` test hand-set the `asked` list — while **nobody called it that way**.
2. `FireReport`'s composer test hand-set the known-flags — while the **wiring** set them wrong.
3. The same fixtures modelled feeds that had answered — which is why an unanswered feed went unnoticed.

Each was found by **planting**, never by reading. INST-1 says the subject list must be derived; this is
sharper and belongs beside it: **a test that constructs the value under test cannot test where the
value comes from.** Proposed for the A2DH fold-in with INST-1 to INST-5.

## 8. Open items carried into VALIDATE

| Item | State |
|---|---|
| **F-58** | Lookup stalls on a freshly seeded install; `journey` red at 26/28 by ruling, 40-second probe committed |
| **F-55** | Below the minimum terminal size the frame overflows silently; the contract lands with Broadcaster UI |
| **F-59** | 17 test-code duplicate groups — internal metric, never a release criterion |
| **F-57** | 11 unmeasurable colour tokens, each with a stated mechanism |
| Demo location | 67 fixture files; recorded API responses, a re-record not a replace. The shipped binary is clean |

## REVIEW exit — recommendation

**Exit REVIEW, enter VALIDATE.** Gate set green, requirements verified, stress tested, documentation
current, critical analysis complete, and the one design question carried out of BUILD closed rather
than deferred.
