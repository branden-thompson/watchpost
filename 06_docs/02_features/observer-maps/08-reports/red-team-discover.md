---
title: "0.18.0 — DISCOVER exit red team, round 1"
date: 2026-09-22
phase: DISCOVER (RCC) exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 COMPLETE — every finding dispositioned; round 2 decision pending (multi-round rule)"
---

# DISCOVER exit — red team, round 1

**Five reviewers, none with prior context, each dispatched verbatim from `06_docs/red-team-brief.md`
in its own scratch directory.** The full axis set plus the DISCOVER lens plus the two personas the
HUM LEAD confirmed (`rulings.md` D-24 onward). Sectioned dispatch with independent verdicts, per
calibration; the code axis ran solo, as it always does.

| Reviewer | Verdict |
|---|---|
| Code Quality (solo) | **Do not ship** |
| Project Hygiene | Ship |
| Docs Quality | **Do not ship as the DISCOVER record** |
| Business Quality | **Do not proceed to PLAN** |
| DISCOVER lens (Distinguished PM) | **Do not exit** |
| InfoSec persona | **Do not proceed to PLAN** |
| Accessibility persona | **Do not proceed to PLAN** |

Six of seven verdicts were negative. Every finding below was **re-checked against the code before
remediation** (Verify-Before-Accept): two were downgraded on that check, and one was over-stated.

## Disposition ledger

**Fixed in code.**

| # | Finding | Disposition |
|---|---|---|
| Code 9a | Fail-open: an unreadable `$HOME` returned `nil, nil`, so gates ran with half the rule and **printed a pass** — the exact class the package exists to close | **Fixed.** Returns the error; positive control asserts the failure, plus a control that an *empty* home still derives nothing without error (`trees.go`, `trees_test.go`) |
| Code 9b / Hygiene H3 | `make lint-identity` ran `-run PublishedTreeNames`, which does not match the new one-definition test: the gate named as enforcing the rule never ran it | **Fixed.** Recipe widened; the selection guard now requires the pattern to select **every** test declared in `identity_test.go`, discovered from the file rather than listed (`Makefile:123`, `gates_test.go`) |
| Code 9c | The one-definition test asserted a *substring*: deleting the live call and keeping its comment left it green | **Fixed.** It now asserts an execution with comment lines stripped, plus a control proving a commented-out call fails the check (`identity_test.go`) |
| Code 4 | Four of six guards added the same day cannot fire — `if expr == ""` is provably unreachable, which P10 Rule 5 forbids ("an assertion a static checking tool can prove can never fail violates this rule") | **Fixed by deletion (D-30).** Both unreachable guards and their tests removed; one **reachable** check added (an explicitly empty `HOME` argument) and P10 re-run: 0 findings. Recorded in the code comment as what it was — checks written to satisfy a density count |
| Code, unlisted | `panic` in a package-level initializer took down every test in `cmd/watchpost` on a plain permissions error | **Fixed.** Built once on first use (`sync.OnceValues`); the test that needs it fails with the reason (`identity_test.go`) |
| Code 3 | Two adjacent filesystem failures had opposite policies (one silent, one fatal) with no explanation | **Fixed** by the 9a fix: both are errors now |
| Code 1 | `"LI_PROJECTS" + "|" + "DESIGN_FOUNDATIONS"` read as a scanner dodge and was pointless (the file is its own exemption row) | **Fixed.** Literal inlined with the reason |
| Code 5 / Hygiene H3b | The self-test printed "8 leak class(es) fired" while counting probes, and probes only the static half | **Fixed (honest half).** It now says "8 static probe(s) … the derived half is NOT probed — F-177". The fixture probe is F-177 |

**Fixed in the record.**

| # | Finding | Disposition |
|---|---|---|
| InfoSec F6 | `seedZoneShapes` fetches zone geometry at station start, contradicting D-21's "no cache warming, of any source"; FR-3.2's instrument passed it by counting only basemap requests | **D-25.** Seed gated on the maps setting and first open; FR-3.2 and its instrument widened to every source |
| A11y A-1 / A-5 | A screen-reader listener had no path in any surface; `Describe` appeared in zero requirements | **D-26.** FR-7.4: the text description ships in macro phase 1, scored by M1's own question; discharges go-tuiMaps D-122's redirect |
| A11y A-4 | R-7.1's "readable without colour" was false: the library's hatch runs only at `NoColour` depth | **D-27.** FR-7.1 amended (depth hinted from `NO_COLOR` *and* theme) + **HR-7** for a depth-independent pattern |
| A11y A-2 | Colour-vision deficiency named nowhere, though the library ships a host-runnable check | **D-27.** FR-7.5: every palette passes `VisionSafe` in watchpost's own tests |
| A11y A-7 | The library ships `ReduceMotion`; the record never mentioned it, so stopping the loop cost the radar | **D-27.** FR-5.8: Settings row, rate ceiling, still form keeping the newest frame and its age |
| A11y A-3 | No-braille listeners get a notice only if they already knew to relaunch | **Fixed.** FR-1.8: the window says it draws in braille and names the remedy |
| A11y A-6 | "Passes the floor" is not "legible": a county view at 69×12 rendered entirely blank | **Fixed.** FR-1.9: a stated legibility tier |
| A11y A-9 | Map colour tokens would sit outside the existing contrast register | **Fixed.** FR-7.6 |
| Business B1 / PM | The bound drifted from "the United States" to "contiguous US"; Puerto Rico, Anchorage and marine zones are live in the tree | **D-28.** FR-2.1 is per-region over every US region and territory the APIs cover; FR-2.5 covers a location in none |
| PM D2 / Docs D3 | M1 had no grader, N or protocol; M5's targets were the library's, never derived for this host | **D-29.** M1: HUM LEAD as grader, recorded-scenario protocol; M5: target set at PLAN, marked as such |
| InfoSec F1 | No threat model: nine risks, none security or privacy | **D-31.** "What leaves the machine" table in the brief; RK-10, RK-11, RK-12 added |
| InfoSec F5 | Whether a source address is user-settable was unstated — a free-form URL would allow a planted map | **D-31.** FR-3.8: closed list in 0.18.0 |
| InfoSec F3 | No watchpost-side radar body cap (httpx would accept 32 MB, twelve times a loop) | **D-31.** FR-5.7: ≤ 1 MiB, never cached over cap |
| InfoSec F4 / F7 / F8 | Caches are an unexpirable record of where the listener looked, with no purge path | **D-31.** FR-3.9: stated retention and a clear path |
| InfoSec F2 | The library opens its own HTTP client, so NFR-4's User-Agent is unachievable | **HR-6** on v0.2.0; NFR-4 states the exemption until it lands |
| Docs H4 | Stale "D-1..D-16" tick; a decided question still written as open; a commit citation that is not the tag | **Fixed** in the brief |
| Docs D5 | "Four alerts in five" published bare in the brief and problem statement; wave 1 measured nine in ten, marine-inflated | **Fixed** at both sites, with the caveat |
| Docs D1 | The required-reading file was linked from nowhere | **Fixed.** Linked from `handoff-0.18.0.md` and the F-174 row |
| Hygiene H5 | Wave 1's numbers rest on live requests with no committed fixtures | **FR-8.6:** the recorded responses are committed before FR-5.3's tests depend on them |
| Docs D4 (fork) | No designer-facing artifact for a release whose output is a picture | **D-32.** The look belongs to PLAN, ratified from rendered specimens; the window follows the existing modal pattern with control commands along the bottom |

**Deferred, with a row.**

| # | Finding | Disposition |
|---|---|---|
| Hygiene H3b | The derived half of the identity rule has no probe | **F-177** — needs a fixture workspace; the fail-open path it would have caught is already fixed |
| Code 8, Hygiene H2 | `P10-02` "bounded by" annotations that are inputs rather than bounds; the mirror script shells out at import time | **F-178** |
| Hygiene H1 | `feature/map-ready-data` is a merged local branch still present | **Deferred:** branch hygiene at SHIP, with the spike branch |

**Declined, with a reason.**

| # | Finding | Why declined |
|---|---|---|
| Docs D3 "dangling D-30" *(the docs axis's fix-first item)* | **Over-stated, downgraded to Minor.** The citation reads "go-tuiMaps NFR-5, D-30" and D-30 **is** a go-tuiMaps ruling (`rulings-discover.md:26`, the time-to-placed-view numbers). The real defect is ambiguity — both repositories number rulings `D-n` — so cross-repo citations are now prefixed, as the v0.2.0 log already does. The record's "every ruling lands here" promise was never broken |
| Business B4 | A bearing-and-distance sentence as a cheaper answer than a map | **Adopted in part, not instead.** FR-7.4 makes the text answer a committed part of the release and the fallback for every failure mode — but not a replacement: the locked problem is about *seeing* where an alert is, and D-3 ruled the Observer map first |
| PM D3 | Radar traces to a preference ruling (D-7), not to the locked problem, and is separable | **Declined, with the rationale recorded.** The HUM LEAD ruled radar the most common map view (D-7) and required loops (D-10); the problem statement classes radar as context for the same question — where is it, relative to me. Recorded so the trace is explicit rather than assumed |
| Code 4 (partial) | Delete `Expr`'s empty-root guard as duplicating `Derived`'s, and `resolve`'s | **Declined.** Both are public entry points reachable from outside the package; each self-checking is the P10 shape, and both guards can fire |
| Code 4 (partial) | Delete the `regexp.Compile` check as unable to fail | **Declined.** `static` is a package `var`: an edit to it can produce an invalid pattern, so the check can fire |
| Code 2 | The derived half differs per machine and is empty on CI, so the merge-blocking gate runs the weakest form | **Declined for now, stated instead.** That is the design (D-1's shape rule): CI has no workspace to derive from, and the static half is what holds there. Documented in the package header rather than changed |

## What the round changed, in one line

A consent promise contradicted by code already on the branch; a bound that excluded Alaska, Hawaii
and the territories; three classes of listener with no path; a metric with no grader; a gate that did
not run its own invariant; and four checks written to satisfy a counter rather than to catch a defect.

## Round 2

The multi-round rule (calibration: iterate on scope and find-rate) says another whole-base round is
due when the scope is foundational **or** the find-rate was material. Both hold: this round produced
Criticals, and remediation added requirements (FR-1.8/1.9, FR-3.8/3.9, FR-5.7/5.8, FR-7.4/7.5/7.6,
FR-2.1/2.5) plus HR-6 and HR-7 that have had no adversarial pass. **A round 2 with fresh reviewers is
recommended before the phase gate**, told the round-1 findings and fixes, asked to re-verify the fixes
hold, to attack what remediation introduced, and to say what they verified clean.
