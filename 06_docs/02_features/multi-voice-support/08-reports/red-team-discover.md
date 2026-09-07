# Red-team round 1 — DISCOVER exit (multi-voice-support, 0.14.0)

| Field | Value |
|---|---|
| Date | 2026-08-29 |
| Reviewers | **A** requirements & problem-fit · **B** adversarial technical (feasibility, concurrency, correctness — two probes) · **C** hostile input, accessibility, Setup UX, configuration (two probes) · **D** documentation truth, consistency, harness completeness, performance lens (~120 citations verified) |
| Inputs | brief v1.1.0 · objectives v1.0.0 · risk register (R-n) · the four analyses · tones.md |
| Totals | **57 findings** — A 21 · B 10 · C 11 · D 15. Verdicts: four × SHIP-WITH-CONDITIONS. **Dispositions: 44 Fixed in DISCOVER · 9 Fix in PLAN (owned) · 4 Accepted with reason · 0 Declined**; 5 HUM LEAD rulings requested at exit (OQ-12, OQ-19, OQ-20, OQ-21, and the downgrade wording under B-1) |
| Truth | every load-bearing citation verified by A and D held; drift was ≤ 10 lines and is corrected; B measured two claims wrong (worst-case concurrency 4–5 → **6**; one Source reaches **3**, not 2) — corrected |

Disposition codes: **F-D** fixed in DISCOVER (the document named) · **F-P** fix in PLAN (batch named) · **ACC** accepted with reason · **HUM** ruling requested.

## A — requirements & problem-fit (21)

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| A-1 | High | Two incompatible config models on the record | **F-D** — `02-analysis/data-shape.md` is the shape of record (typed pair per role); the old block in `voice-architecture.md` marked superseded |
| A-2 | High | FR-2 "byte-identical" contradicted by the tones, the tail wording and the tail key | **F-D** — FR-2 rescoped to order/text/pauses/one voice with the three deliberate deltas named |
| A-3 | High | Removing the `voice` action breaks a rebound `V` in `[keys]` | **F-D** — FR-14: the action stays, opens Setup at Correspondents; removed actions ignored with an `[S]` note (OQ-19 to ratify) |
| A-4 | Medium | Station role, maritime order and burst tone baked in unratified | **F-D / HUM** — Station and burst are MVS-D-13/12; maritime order restored as OQ-12 for ratification at exit |
| A-5 | Medium | The severe read's new tone vs `[M]` mute | **F-D** — FR-11: tone suppressed under mute, the words not |
| A-6 | Medium | The root has no stated fallback | **F-D** — FR-7 terminal link (platform default, stated in `[S]`); matrix row added |
| A-7 | Medium | M2 counts steps, not keypresses | **F-D** — M2 = keypresses, ≤ 12 (AM-8) |
| A-8 | Medium | The locked problem's default-on answer is the tones; M3 has no fresh-install arm | **F-D** — M3 fresh-install arm (AM-9); objectives §0 says it |
| A-9 | Medium | `TestNoColorOnlyCarriers` does not exist | **F-D** — FR-13 cites the real gates |
| A-10 | Medium | FR-6 absence semantics contradict the analysis; no screen-vs-voice check | **F-D** — one absence line; the equality test forces the `platform/render` lift |
| A-11 | Medium | Validation window before `listVoices` lands | **F-D** — "not yet discovered ≠ unknown"; the sentinel always resolves (matrix rows) |
| A-12 | Medium | FR-4 "survives downgrade" overstated | **F-D** — reads as 0.13.0, drops the tables on its next save (NFR-5) |
| A-13 | Medium | No requirement for the no-audio host | **F-D** — FR-4 clause + the nil-deck PTY case |
| A-14 | Medium | A preview downloads 63 MB with no confirmation | **F-D** — FR-4/NFR-4: picker marks installed; asks once |
| A-15 | Medium | "Nothing installed" row missing from M5 | **F-D** — the one legitimately silent row, stated |
| A-16 | Low | Tone table pre-empts OQ-16; no default class; a no-op switch | **F-D** — OQ-16 ruled (storm wins); unmatched → warning; Blizzard marked for PLAN; the `warning` switch dropped |
| A-17 | Low | FR-12's numbers untestable and inconsistent | **F-D** — one rule, one owner; N bracketed for PLAN |
| A-18 | Low | `[V]` references outside FR-4 (help, hint, README, docs) | **F-D** — NFR-8 |
| A-19 | Low | FR-5 unsatisfiable for the sentinel | **F-D** — "your correspondent" clause |
| A-20 | Low | Stale rows in the ratified brief | **ACC** (frozen text) — the v1.2.0 Amendment Log points at the current wording |
| A-21 | Low | FR-10 serves the on-screen user | **ACC** — the M1 human-visible proxy; PLAN batch 4, never earlier |

## B — adversarial technical (10)

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| B-1 | Critical | A downgrade's first save drops every role table (measured) | **HUM** (wording ruling at exit) + **F-D** NFR-5 reworded; **F-P batch 1**: 0.14.0's save preserves unknown keys (the last release with the hole); RS-22 |
| B-2 | Critical | FR-12 self-contradictory; one Source reaches 3 renders; resident mode contends on the process; a read's render can starve a takeover; Source `Say` has no bound | **F-D** — FR-12 rewritten (Say-only semaphore, takeover-only reserved slot, single-utterance hand-over, bounded Say, resident-mode rule on the OQ-18 checklist) |
| B-3 | High | Compose-time hand-over double-fires and goes stale | **F-D** — Source-time hand-over from an injected script-tree line (`data-shape.md` §5; FR-5; RS-5) |
| B-4 | High | The alert tone takes the install path | **F-D** — FR-9: the tone's rate is a constant, never resolves a voice; RS-3 |
| B-5 | High | Resident Piper: no completion signal with `output_file`, no cancel, no shutdown owner, umask, previews on `context.Background()` | **F-P** (gated by OQ-18) — the Arch checklist in `perf-protocol.md` §3; FR-4 previews on the app ctx; RS-23 |
| B-6 | Medium | Two config shapes; a map mints typo'd roles | **F-D** — a struct of `RoleVoice{MacOS, Piper}` per role (`data-shape.md` §2) |
| B-7 | Medium | Resolver under `s.mu` and the 100 ms poll | **F-P batch 2** — a resolution generation on the Source (`data-shape.md` §5) |
| B-8 | Low | Concurrency counts wrong in two analyses | **F-D** — corrected (3 per Source, 6 process-wide) |
| B-9 | Low | Find-only race — verified safe; a nit on re-download | **ACC**; PLAN may skip a present checksummed model |
| B-10 | Low | Narration seam (b) — verified | **ACC** |

## C — hostile input, accessibility, UX, configuration (11)

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| C-1 | High | A config-supplied voice name renders raw (SGR) in the chip; late-bound after `Plain` (probed) | **F-D** — FR-7 "only the resolved voice's name is displayed or spoken"; NFR-6 `PlainLine` + cap; RS-18 |
| C-2 | High | Setup already overflows at 80×24 with no scroll; the focus mark leaves the screen (probed) | **F-D** — FR-4 "usable at 80×24"; the 80×24 golden; RS-19; **F-P batch 4** the collapsing, focus-following form |
| C-3 | High | The hand-over inherits "broken script = silence" | **F-D** — FR-5: built-in line fallback, `.Voice` alias; RS-20 |
| C-4 | Medium | Hand-edited config semantics half-specified; comment loss on save; no `Radio.Validate()` FR | **F-D** — NFR-5 (unknown keys listed in `[S]`; comment loss stated) + FR-8 (`Radio.Validate()`); **HUM** OQ-20 |
| C-5 | Medium | Removing the `voice` action can refuse to start | **F-D** — FR-14 (with A-3); OQ-19 |
| C-6 | Medium | Setup's `›`/`⚠` bypass `Glyphs()`; chip states unspecified | **F-D** — FR-13 (states by words; the joined marks fixed) |
| C-7 | Medium | Maritime wording for the metric / non-mariner listener; two wind units in one broadcast | **F-P batch 3** (the scripts are the HUM LEAD's words) — defaults recorded; **HUM** OQ-21 |
| C-8 | Low | Station names: `PlainLine`, a cap, `ExpandStates` and the abbreviation table | **F-D** — NFR-6 |
| C-9 | Low | Privacy — nothing new persisted or disclosed | **ACC** |
| C-10 | Low | No non-TUI twin for the cast (`report --verbose`) | **F-P** — the resolution table joins the diagnostic dump |
| C-11 | Info | argv construction sound; FR-7 is the real guard | **ACC** |

## D — documentation truth, completeness, performance (15)

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| D-1 | Critical | No data shape of record | **F-D** — `data-shape.md` |
| D-2 | Critical | M4/NFR-2/3 have no baselines or procedures; the Arch measurement has no protocol | **F-D** — `07-readiness/perf-protocol.md` (0.13.0 baselines cited; time-to-tone-start method; the Arch RSS protocol with the OQ-18 decision rule); NFR-2/3 reworded; PLAN batch 0 records the 0.13.0 number |
| D-3 | Important | FR-12 contradicts itself | **F-D** (with B-2) |
| D-4 | Important | The ratified brief not amended | **F-D** — brief v1.2.0 Amendment Log (AM-1…12) |
| D-5 | Important | Register lacks likelihood/status; `R-n` collides | **F-D** — `RS-1…23` with Lik + Status; intake RS map |
| D-6 | Important | Open questions recommended, not ruled; OQ-12 silently closed | **HUM** — OQ-12/19/20/21 at exit; OQ-13…18 ruled MVS-D-12…17 |
| D-7 | Important | Missing: discover-report, red-team ledger, data-shape, organisational/timeline constraints | **F-D** — this ledger, `data-shape.md`, `discover-report.md` (constraints incl. organisational/timeline; mocks deferred by MVS-D-10, stated) |
| D-8 | Important | Tones pacing claim half-true | **F-D** — `tones.md` corrected; NFR-2 = time-to-tone-start |
| D-9 | Important | NFR-4 uses download sizes as disk | **F-D** — lower bound + `perf-protocol.md` §5 |
| D-10 | Important | No Setup-open alloc pin; chip miss semantics | **F-D** — NFR-3 |
| D-11 | Important | Audience rule: no plain-language §0 | **F-D** — objectives §0 |
| D-12 | Minor | Citation drift | **F-D** |
| D-13 | Minor | Small inconsistencies (counts, A-3 status, Blizzard, WAT naming) | **F-D** |
| D-14 | Minor | M2 counted for a chooser | **F-D** (with A-7) |
| D-15 | Minor | A design inside NFR-2 without brackets | **F-D** |

## What the round changed, in one paragraph

The requirement set gained a plain-language front, five requirements (FR-14, NFR-8, the 80×24 Setup rule, the display-validation rule, the bounded-`Say` rule), one shape of record for the data, a performance protocol with real 0.13.0 numbers, and six new risks. Three designs the analyses recommended were replaced by better ones on evidence: the cap wraps `Say` only with the takeover as the sole reserved path; the hand-over is decided at Source time; the tone never resolves a voice. Nothing was declined.
