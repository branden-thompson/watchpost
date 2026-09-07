---
title: "Discovery Report: Multi-Voice Support (correspondent roles)"
subtitle: "watchpost · multi-voice-support · 0.14.0"
date: "2026-08-29"
toc: true
toc-depth: 2
---

# Discovery Report — Multi-Voice Support (correspondent roles)

| Field | Value |
|---|---|
| Report | discover-report v1.0.0 · detail level FULL (FULL RCC · FULL REPORTS) |
| Feature | `multi-voice-support` → **0.14.0** |
| Phase | DISCOVER — exit |
| Classification | LEVEL-1 · SEV-0 · HUMAN LEAD (confirmed: the feature lives on the radio path, changes the broadcast stream, the narration seam and the config) |
| Directives | FULL GIT · DOCS · REPORTS · RCC · PLAN · DIAGRAMS · TDD · standing performance lens · R6 |
| Branch | `feature/multi-voice-support` off `main @ 186d97c`; DISCOVER commits `7250321` → this exit |
| Related | `08-reports/project-brief.md` v1.2.0 · `01-objectives/objectives.md` v1.1.0 · `02-analysis/{data-shape, voice-architecture, maritime-report, piper-and-platform, tones, risk-register}.md` · `07-readiness/perf-protocol.md` · `08-reports/red-team-discover.md` |
| Status | **APPROVED — HUM LEAD, 2026-08-29** ("OQ-12/19/20/21 Approved; B-1 Approved"); rulings MVS-D-18…22 |

## Executive Summary

Watchpost Radio speaks with one voice. A listener who is not looking at the screen cannot tell, by ear, that
an alert has started rather than a routine report — the problem this feature solves (locked, MVS-D-1). The
answer has a default-on half and an opt-in half. **Default-on:** every class of alert gets its own tone from
five presets the HUM LEAD auditioned — warnings and disasters, watches, advisories, special statements,
tropical and winter storms. **Opt-in:** a cast — a voice per correspondent role in a tree with inheritance
(everything → alerts / standard → each report and the station's own lines), assigned in Setup with a preview,
with scripted hand-overs between correspondents and an on-air chip. Riding with it: a full **maritime report**
(the coastal-waters forecast, buoy observations, tides, currents) for coastal places, the retirement of the
`[V]` chooser, and a Radio UI redesign to the HUM LEAD's mock.

Discovery probed the code rather than assuming it. The findings that shape PLAN: more than two voices is
**not** limited by performance (the engine is voice-agnostic; the cost is Piper's per-utterance model load,
paid today regardless); **no concurrency cap exists** (the documented "≤ 2" is aspirational — six processes
are possible, three from one broadcast alone); macOS **speaks the default voice for an unknown name and the
app announces the wrong one**; the alert **tone itself takes the install path**; the coastal-waters forecast
is **free** through the endpoint already used for the land products; and the Setup window **already overflows
at 80×24**. A four-reviewer red-team (57 findings, 44 fixed in DISCOVER, none declined) replaced three of the
analyses' own recommendations with better ones on evidence. Recommendation: **proceed to PLAN.**

## Context

- 0.13.0 shipped the arbiter that orders reads (a breaking takeover pauses and resumes a read), the severe
  window's `[space]` read, the fire and seismic reports, six Piper voices and ten curated macOS voices — the
  station has a cast but one seat.
- Prior art searched (anti-pattern 4): the `[V]` chooser and its preview path; the 0.12.0 install
  serialisation; the mid-broadcast `Handoff`; the fire/seismic report shape and scripts; the settings-group
  Setup; the pronunciation tables; the config contract; the 0.9.x UAT record for Piper timings; upstream
  Piper 2023.11.14's CLI loop; the NWS products and zones APIs (live probes).
- Dependencies: NWS API (products incl. CWF; zone geometry), NDBC, CO-OPS, HuggingFace (the pinned Piper
  models — all six verified at 63 201 294 bytes), macOS `say`, Piper.

## Functional requirements

FR-1…FR-14 in `01-objectives/objectives.md` v1.1.0, each with its verify-by. In brief: the role tree with
inheritance incl. *Station* (FR-1) · today's behaviour as the zero value, with three deliberate deltas named
(FR-2) · every read in its resolved voice, sections of one broadcast included (FR-3) · assignment and preview
in Setup, usable at 80×24, no typing, ≤ 12 keypresses, no-audio host, confirm before a download; `[V]` removed
(FR-4) · identity lines name the speaker; Source-time scripted hand-over that never falls to silence (FR-5) ·
the full maritime report incl. the CWF, ordered products → maritime → fire → seismic (FR-6) · explicit,
never-silent fallback with a stated resolution; display validation (FR-7) · a typed pair per role, both
platforms one file, `Radio.Validate()` (FR-8) · find-only alert path incl. the tone (FR-9) · the on-air chip
(FR-10) · per-class tones, five presets, share switch, mute rule, burst rule, storm precedence (FR-11) ·
bounded concurrency with one rule and the arbiter renamed the Station Director (FR-12) · `--ascii`/colour-off
parity incl. Setup's joined marks (FR-13) · the `voice` action kept as a Setup deep-link; removed actions
tolerated (FR-14).

## Non-functional requirements

NFR-1 R6 both platforms (blocking; the 0.13.0 Linux half folds in) · NFR-2 time-to-tone-start not worse than
0.13.0 (the 0.13.0 number recorded as PLAN batch 0) · NFR-3 RSS measured against the 0.13.0 baseline (116 MB
launch / 86–115 MB plateau), a voice-scaled PCM bound, a Setup-open alloc pin · NFR-4 disk as a lower bound
with the extracted size recorded · NFR-5 no migration; unknown keys listed; 0.14.0's save preserves unknown
keys · NFR-6 hostile text on every new name (`PlainLine` + cap) · NFR-7 the standing gates · NFR-8 no trace of
`[V]`.

## Constraints & dependencies

**Technical** (`02-analysis/*`): the broadcast stream is one voice per Source with a segment-keyed cache and a
latent rate lock; narrations are per-utterance and rate-safe; Piper reloads its model per utterance (~10 s) and
a resident process is viable but unmeasured; identity is a bare string in two namespaces; the config has no
migration; the tone is constants; no maritime script exists; `Compose` is at 8 positional parameters (the
`Reports{}` struct comes first); the Setup form is flat and already overflows.

**Organisational:** one HUM LEAD approves every gate, supplies the Radio UI and Setup mocks at PLAN
(MVS-D-10), performs the ear test (M3), owns the tones' sound and the maritime wording, and runs the Linux
measurements (the Arch box) at UAT (MVS-D-17) — the resident-Piper decision waits on that measurement, not on
DISCOVER.

**Timeline:** none imposed; PLAN's DATA FIRST batching (§Recommendation) lets each batch be UAT-able alone,
and the Linux measurement lands mid-BUILD without blocking batches 1–3.

## Risk assessment

`02-analysis/risk-register.md`: RS-1…RS-23 with severity, likelihood, status and a concrete mitigation each.
The high-likelihood highs: unbounded synthesis (RS-1 → FR-12, batch 1), the wrong-voice-silently pair (RS-2,
RS-18 → FR-7/8), Setup at 80×24 (RS-19 → FR-4), the broadcast's one-voice assumptions (RS-4), the unmeasured
resident-Piper RSS (RS-6/RS-23 → the protocol), R6 (RS-16). Accepted: golden churn (RS-14), macOS identity
instability (RS-17), the 0.13.0 downgrade save (RS-22 — ended forward).

## Open questions

Closed during DISCOVER by ruling: OQ-1…9 (intake), OQ-11 (tones), OQ-13…18 (MVS-D-12…17). **For ratification
at this exit** (recommendations stand in the objectives):

| OQ | Question | Recommendation |
|---|---|---|
| OQ-12 | Maritime order in the broadcast | products → maritime → fire → seismic → tail |
| OQ-19 | The freed `V` key | keep the `voice` action as "open Setup at Correspondents" (FR-14) |
| OQ-20 | Setup's save drops hand-written comments | acceptable for 0.14.0, documented; keys are preserved; a targeted writer is a follow-up |
| OQ-21 | Maritime wording | "above the low-water mark", 12-hour spoken clock, wind in the listener's unit, currents in knots — the scripts are the HUM LEAD's words at PLAN |
| B-1 | Downgrade wording | "a 0.13.0 binary reads the file as 0.13.0 and drops the role tables on its next save" (NFR-5) |

Deferred by ruling to PLAN: OQ-10 (the Radio UI / Setup mocks — MVS-D-10).

## Recommendation

**Proceed to PLAN**, DATA FIRST, in four batches each UAT-able alone: **0** record the 0.13.0
time-to-tone-start; **1** the `Say` semaphore + the `Reports{}` struct + the config pair/registry/resolver/
validator + save-preserves-unknown-keys; **2** the broadcast seams (voice in the cache key, `renderedSeg.voice`,
Source-time hand-over, the resolution generation, the station role, the narration seam with the Station
Director rename, the tone as constants) + the find-only alert path and background pre-install; **3** the maritime
report (incl. the CWF via `FilterUGC` and a marine-zone resolver) + the five tone presets and the classifier;
**4** Setup's Correspondents group (collapsing, focus-following, previews via `Audition` with the Director's
duck) + the Radio UI redesign from the mock + the chip + goldens re-recorded once. FULL PLAN: three
approaches for the resident-vs-per-utterance Piper seam, evaluated against the protocol.

## Decisions

MVS-D-1…22 (the brief, v1.2.0 incl. the exit rulings AM-13).

## Follow-ups noted at the exit (backlog, MVS-D-21)

- A **12/24-hour** preference for display and narration.
- A **time-zone** preference (default system/local).
- An **"FRS transmit" mode** — for a user relaying Watchpost on their own FRS channel: time wording and certain heads/tails change.
- A targeted config writer that preserves hand-written comments (MVS-D-20).

## Critical analysis (red-team)

`08-reports/red-team-discover.md` — four reviewers, 57 findings (A 21 · B 10 · C 11 · D 15), 44 fixed in
DISCOVER, 9 owned in PLAN, 4 accepted, 0 declined. Headline replacements: the cap wraps `Say` only with the
takeover as the sole reserved path (B-2); the hand-over is a Source-time decision (B-3); the tone never
resolves a voice (B-4); the display of a name is validated, not only the argv (C-1); the Setup form is a
requirement at 80×24 (C-2); the performance lens got a protocol with real baselines (D-2).

## Errata (post-approval rulings)

MVS-D-24 dropped the on-air chip from the panel (FR-10 → the `[S]` cast table); MVS-D-26 replaced the
"share the Warnings tone" switch with a per-class mute (`[M]` = tones only); MVS-D-28 made six tone classes
(Disaster separate from Warning). Where this report says "chip" or "share", read the current objectives.
PLAN red-team rounds (2026-08-29): the reserved slot is the Station Director's (two slots, every job — FR-12), not
`narrateBreaking`'s alone; M2 is pinned at ≤ 11 keypresses (AM-18), not "≤ 12"; the objectives are v1.2.0 (the
Source list below names v1.1.0, the version this report was written against).

## Source documents

`08-reports/project-brief.md` v1.2.0 · `01-objectives/objectives.md` v1.1.0 · `02-analysis/data-shape.md` ·
`02-analysis/voice-architecture.md` · `02-analysis/maritime-report.md` (+ §10 CWF addendum) ·
`02-analysis/piper-and-platform.md` · `02-analysis/tones.md` · `02-analysis/risk-register.md` ·
`07-readiness/perf-protocol.md` · `08-reports/red-team-discover.md`.

## Next steps

1. HUM LEAD: approve this report; rule OQ-12/19/20/21 and the B-1 wording.
2. Save the workflow state (DISCOVER → PLAN); commit; announce.
3. PLAN (FULL PLAN · FULL DIAGRAMS): approaches, architecture, the mocks (MVS-D-10), the implementation plan
   in the four batches, the PLAN red-team.

## Appendix — technical summary for the engineer

- **Where the voice attaches:** `Segment.Role` resolved at render by an injected resolver in `synth.Source`;
  cache key = voice + segment key; `renderedSeg.voice`; a resolution generation replaces the global `gen` as
  the "re-voice?" question (`data-shape.md` §5; `voice-architecture.md` A-2).
- **The narration seam:** role on `narrationJob`; the Station Director forwards it to `tone`/`render`; classes
  stay at two; the tone's rate is a constant (`voice-architecture.md` A-3).
- **The cap:** a semaphore around `Voice.Say`, N ordinary + 1 reserved for `narrateBreaking`, the hand-over as
  one `Say`, a bound per Source `Say` (FR-12).
- **Resolution:** `resolveVoice(role) (synth.Voice, Resolution)`, find-only, this host's namespace, the
  fallback matrix (`data-shape.md` §3–4).
- **Maritime:** `MarineReport` + `MarineSegments` + `scripts/maritime-report/*` + `Assembler.MarineFor`; the CWF
  through `Products` + `FilterUGC` with a marine-zone resolver from zone geometry (`maritime-report.md` §8, §10).
- **Tones:** `tone.go` generalised to five parameterised presets; a product-string classifier with storm >
  warning > watch > advisory > statement and warning as the default (`tones.md`).
