---
title: "REVIEW report — multi-voice-support (0.14.0)"
date: 2026-09-06
phase: REVIEW
sev: SEV-0
authority: HUM LEAD
status: "REVIEW exit — recommended, pending the HUM LEAD items in §7"
---

# REVIEW report — multi-voice-support (0.14.0)

**Scope:** `86444ce~1..HEAD` — 18 commits, 21 files, +888 / −92. Entered on the HUM LEAD's
**"BUILD-EXIT APPROVED; GO 4 REVIEW"** (2026-09-06).

**In one line:** REVIEW found no new defect in the *product* that UAT had not already surfaced — and
five in the things that measure it.

---

## 1. Defects closed

| Ref | Defect | How it was found |
|---|---|---|
| **F-41** | Previewing a voice in Settings gave no sign it was working — `p` sent the deck's progress, and its reason on failure, to a field nothing draws | red team, then confirmed against UAT 119 |
| **F-42** | Details opened from a lookup was titled `Location` for a frame or more after enter | the PTY journey, ~1 run in 3 |
| — | The `[w]` Watchlist tab could not say why it was empty **when locations were set** — the one case where it looks like a national view that has missed something | **live UAT**: an Alabama Special Weather Statement, seen in another app, absent here |
| **F-44** | The journey's whole severe-window block could not fail | chasing a failing step in REVIEW |

**F-41 and F-42 are the same shape as C-3, three times in one release:** *receiver built, wire
missing.* In F-41 `castNote` already drew the note and `VoiceNoteMsg` landed in a field the retired
`[V]` chooser used to read. In F-42 `lookupRef` was already documented as "the location a lookup
opened Details for" and the title never consulted it. Round 3's proposed general answer — a
producer/consumer completeness check over the closed sets — would have caught all three, and remains
unbuilt.

## 2. Instrumented, not fixed

**F-43 — a tone sounded three times with no words and no marquee callout** (HUM LEAD, UAT, on a real
burst). No diagnosis was reached and none is claimed. What landed (`c7ace52`) is the missing
invariant and a way for the next sighting to explain itself: `app/tone_promise_test.go` pins that a
sounded tone is followed by words — **nothing pinned that before**, since `tone_latency_test.go`
measures the tone's *latency* — and `openTone` now traces `read:gaveup:after-tone` with the class,
duration, part count and the context's error.

Two things are recorded honestly against it. **The pin's first version passed while proving
nothing** — `classVoice.tone()` returns a zero duration, which sends `holdRest` down its `awaitAir`
branch and never poses the window; only its control caught that. And **I made this failure quieter
myself**: before I-2 a cut-short read raised the relay-fault window on an empty schedule; making the
cut-short `Routed` was right, and it also silenced this. Not reproduced since, through a later
burst — **a trap armed and unsprung, not a fix.**

## 3. The documents caught up with the release

- **CHANGELOG** was missing its three most recent fixes; entered.
- **README still documented a window this release renamed.** Two places said *Setup* four screens
  below a CHANGELOG entry announcing *Settings*; the report was introduced as *maritime* after
  MARITIME became MARINE; and the Radio section's broadcast order named Fire and Seismic and
  **skipped the marine report entirely**, though `compose.go` puts it immediately before fire.
- **`docs/extending.md` Walkthrough 3 was verified by resolving every symbol it names** against the
  tree — all seven live. A walkthrough naming a dead symbol is worse than none.

**None of this is reachable by any gate we have: every one of them reads Go.** The README is the
artefact written entirely for people who are not us and the first thing a reader of this release
sees, and nothing measures it. A candidate check is recorded in `quality-observations.md` and
deliberately **not built** — REVIEW is not where new gates get added.

## 4. The instruments — this phase's dominant finding

**F-44: four consecutive journey assertions could not fail.** 0.14.0 made the ticker band name its
lane in words, and those words *are* the category labels — `BandLabel` is `"Emergency Orders"`,
`"Warnings"`, `"Watches"`, `"Disasters"` (`platform/category/category.go:83–92`) — and the marquee is
always on screen. A fifth, `Showing 1-`, was written to prove the Settings modal had **closed** and is
drawn by the dashboard's recent table beside the open modal.

**The rule:** *an end-to-end assertion must name something only the surface it is testing draws.* The
old patterns were chosen because they are what a person **sees** in the window — the wrong criterion,
because a person sees the marquee too. The anchor was already there: `Total Category Events`
(`severe.go:391`), on every tab, empty ones included, and nowhere else.

Honest anchors then exposed a defect beneath: `severeOpeningTab()` returns `lastBreakingTab` when a
breaking event landed recently, and the arrows wrap — so **counting Rights from an assumed Warnings
was never sound**, and the unfalsifiable patterns had been hiding it.

And a third, in the guard meant to catch exactly this: its count check counted patterns **after**
exemptions, so every honest exemption eroded the tripwire that detects a blind reader. Split into
`extracted` and `checked`; blinding one regex arm now drops it 28 → 8, **verified rather than
assumed**.

## 5. The disk was an instrument too

The volume reached **1.2 GiB free of 926 GiB**, 274 GB of it Go build cache from mutation runs.
This is in the quality record and not a chore list because **every measurement taken that afternoon
was taken on that machine**: journey runs disagreed with each other, three network steps failed, and
a 14.2 s radio-read latency that a release question was about to rest on was measured with no disk
left. The variance was called flakiness for hours and free space was never checked.

**HUM LEAD ruling:** a deterministic step, not a threshold — *do the thing → collect the results →
put them somewhere durable → verify they are there → delete all other build variants*, keeping one to
three real binaries for UAT and version comparison. Implemented as `make hygiene`, wired into
`mutant-check`, and **confirmed to refuse** (no `RESULTS`, missing file, empty file) before it was
trusted to delete anything. Recovered 1.2 GiB → 275 GiB.

It also caught a thin record of mine: `mutant-check`'s durable log was one line — `ok … 241s` — which
proves the gate ran and nothing else. With `-v` it carries every harness property that held and all
189 mutants that apply and compile.

**Re-measured on the clean disk, per ruling:** the read is **11.4 s** against 14.2 s, and that run's
*only* failure was the read itself. So ≈2.8 s was contamination and **11.4 s is the app**.

## 6. Gates at REVIEW exit

| Gate | Result |
|---|---|
| `make verify` | **ALL GATES GREEN** — re-run cold after the cache clean, with `hygiene` wired in |
| `make race` | green, 44 packages, no data race |
| `make journey` | **one failing step**, and it is a real signal, not a harness fault (§5) |
| `mutant-check` | green; durable record now 730 lines |
| Label guard | green, with its own control verified |

**The journey's failing step is being left red on purpose.** Widening its bound would convert a
measured 11.4 s into a green tick.

## 7. Carried out of REVIEW

**To the release, by HUM LEAD ruling (2026-09-06):** the README captures (*"Screenshots will happen
once we're ready to cut the PR"*) and the `gh api user` identity check (*"we'll verify we're using
the right github login through the PR process"*).

**To VALIDATE, post-release, by HUM LEAD ruling:** `linux-validation-protocol.md` and
`perf-protocol.md` §3–5 — *"Arch linux box always happens POST release, because I need the cut
release to install and test on the box."* **A sequencing ruling, not a waiver**: 0.15.0's
resident-Piper decision (MVS-D-17 / OQ-18) has no input until §3 is measured.

**Still owed before the release commit:** ratification of every P10 ledger row, including P10-08 for
`app/inject_release.go`, which is presented and not taken; and the three Setup goldens plus the
re-recorded Radio goldens reviewed against the mocks.

**Open follow-ups, none blocking:** **F-40** (Brengel Fire — its own DISCOVER against the fire APIs),
**F-43** (§2), **F-44** residue — the read step still depends on its tab having a focused event and
nothing establishes that, because this harness has no honest absence check; and one golden on a
non-default severe tab would retire both new `journeyUncovered` exemptions.

## 8. Recommendation

**REVIEW exit is recommended**, conditional on the two HUM LEAD items still owed before the release
commit (§7). No defect known to this phase is being shipped unfixed except F-43, which is recorded,
instrumented, and unreproduced — and the deferred 11.4 s read, deferred by explicit ruling until a
real user reports the wait.
