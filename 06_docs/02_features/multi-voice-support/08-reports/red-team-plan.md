# Red-team — PLAN exit, multi-voice-support (0.14.0)

**Phase:** PLAN (SEV-0, HUMAN LEAD). **Reviewed tree:** `b4950e6` → `ebfff81` (the Docs axis ran on the docs-only
remediation that landed mid-round). **Lenses (round 1, ten, independent, none saw another's output):** the four
axes — Code Quality, Project Hygiene, Docs Quality, Business Quality — the PLAN phase lens (Principal Architect),
and the five personas — Accessibility, InfoSec, Junior Developer, Performance, Safety-critical (P10).
**Round-1 verdicts:** Code **NO-GO** · Hygiene **NO-GO** · Docs **NOFLIGHT** · Business **REWORK (scope)** ·
Architect **REVISE** · a11y **PROCEED WITH CONDITIONS** · InfoSec **CONDITIONAL PASS** · Junior **REVISE** ·
Perf **Conditional pass** · Safety **REVISE**. Total findings: **140** (15 · 19 · 22 · 15 · 14 · 12 · 6 · 15 · 9 · 13).
**Disposition vocabulary:** Fixed · Declined (reason) · Deferred (owner) · Moot (the site was removed) ·
**ESCALATE** (a HUM LEAD ruling; presented with the Plan of Record).

Round 2 (fresh lenses, told what round 1 found and what changed) is §12; round 3 is §13.

## 1. Business Quality (Distinguished PM) — BQ

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| BQ-1 | High | The locked problem rests on assertion — 0.13.0 already sounds a tone before a takeover; no 0.13.0 ear-test baseline | **Fixed (protocol)** — M3 is now an A/B against the 0.13.0 binary, randomised, both arms (`07-readiness/gates.md` §2). **ESCALATE E-3:** if 0.13.0 already scores ≥ 9/10, the HUM LEAD rules whether Candidate A stands or the feature is re-locked as delight (Candidate B) |
| BQ-2 | High | The "default-on answer" claim (per-class tones) answers "which class", not "alert vs report" | **ESCALATE E-3** (same ruling) — the recommendation is to drop the claim from objectives §0 / AM-9 and keep the tones as the fresh-install *cue*, not the answer |
| BQ-3 | High | MVS-D-1 (the away listener) vs MVS-D-8 (fresh install = one voice) | **ESCALATE E-4** — options: a default alert voice on a fresh install (macOS free; Linux +63 MB background) or accept opt-in and say so |
| BQ-4 | High | `ResidentPiper` ships disabled, unmeasured, ~250 lines | **Fixed** — cut from 0.14.0 (P2 Task 2.11 records it; `piper_mode` not added; RAT-2 withdrawn). **ESCALATE E-1** for confirmation (it amends AX-1's "build the seam for all three": the seam is `synth.Voice`) |
| BQ-5 | Medium | Station / Breaking / SevereRead pairs have no picker | **Fixed** — FR-8 states they are config-only; Setup shows them by inheritance (the mock) |
| BQ-6 | Medium | The CWF is not free; the maritime report is ~2–2.5 min | **ESCALATE E-2 (scope)** — recommendation: keep (MVS-D-4/14/18 ruled it; RAT-6's 3-period cap bounds the CWF; the report is between the forecast and the fire report, skippable with `space`) |
| BQ-7 | Medium | Per-class mute has no listener evidence | **ESCALATE E-2** — recommendation: keep (MVS-D-26/28 ruled from the mock); the cost is one group of checkboxes over a rule that already exists |
| BQ-8 | Medium | The Radio-panel redesign is in scope only because `[V]` leaves | **ESCALATE E-2** — recommendation: keep (MVS-D-23/24 ruled; the goldens are re-recorded once either way) |
| BQ-9 | Medium | FR-4 under-delivered: installed marks, the download step, the no-audio note | **Fixed** — P4 4.6: the not-installed note before Save, the no-audio note, the deck's progress note |
| BQ-10 | Medium | M2 not measured; `enter × N` unstated | **Fixed** — the journey counts its own keypresses; the path is 11 and the script pins ≤ 11 (AM-18, round 2). A direct-Save key: **OP-4** for the HUM LEAD (a mock deviation) |
| BQ-11 | Medium | M3 as written is theatre | **Fixed (protocol)** — see BQ-1 |
| BQ-12 | Medium | P0 on macOS only; different instrumentation on the two sides | **Fixed (round 2 re-opened it: the PTY fixture did not exist)** — M4's instrument is `BenchmarkTimeToToneStart` on both trees plus the `breaking:`/`tone:` debug lines; the Linux 0.13.0 run is row 1.1 of `linux-validation-protocol.md`, before any 0.14.0 run |
| BQ-13 | High | Scope bundling: 0.14.0 = tones + two-picker cast; maritime 0.15.0; UI 0.16.0 | **ESCALATE E-2** — the plan is presented whole; the HUM LEAD may cut by ruling. Recommendation: keep the ratified scope (MVS-D-4/23/26) now that the resident path is cut and the per-batch UAT is explicit |
| BQ-14 | Medium | `render.Columns` does not exist; "the Help rule" is a new layout engine | **Fixed** — P4 4.4 on `sideBySide`/`twoColumnsWidth`/`panelChromeFor` with a row table; no new composer |
| BQ-15 | Low | Goldens' manual review; Task 2.7's size; screenshots; Task 0.1 unbounded | **Fixed** (fixture-driven 0.1; three named captures on the release checklist); splitting 2.7: **Declined** — it is one seam; the task says where it splits if the BUILD needs it |

## 2. Code Quality — CQ

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| CQ-1 | Critical | `keep.go` uses `reflect` + recursion; `make p10 0/0` unreachable | **Fixed** — `DisallowUnknownFields` + `StrictMissingError` paths; `keepUnknown` is one flat loop; no `reflect` |
| CQ-2 | Important | Two validators, a mirrored class-key set, literal `"cast"`/`"mute"` in three packages | **Fixed** — `config.Radio.Validate` checks table shapes only; class keys are `cast.Validate`'s; `toneClassKeys` deleted; plan §3.2 says `platform/` imports no `domains/` (the `config --> cast` edge removed) |
| CQ-3 | Important | `castView` + three mappers; `Overrides` derivable | **Fixed** — `CastView{Mode, Names}` only (an override is on iff its name is set); the tick state lives in `setupState`; `castForSave` clears unticked names. `tty` editing `cast.Config` directly: **Declined** — `modes/tty` may not import `domains/` (`make lint-imports`) |
| CQ-4 | Important | Linux picker seeded with keys, cycles names | **Fixed** — `displayName`/`storedName` in `app`; a Linux round-trip test |
| CQ-5 | Important | `installing` nil map; no darwin skip | **Fixed** — lazy init; the mutex test uses a temp `voiceDir` |
| CQ-6 | Important | The fixture in the dropped share model | **Fixed** (`ebfff81`) |
| CQ-7 | Important | `ResidentPiper` dead code behind a knob | **Fixed** — cut (E-1) |
| CQ-8 | Important | `Source.SetVoice` survives only for the deleted chooser | **Fixed** — deleted; `Recast()` through the resolver pins the mid-segment tests |
| CQ-9 | Important | Three hand-over names, two argument orders | **Fixed** — `(from, to)` everywhere; `builtinHandoff` |
| CQ-10 | Important | Resolve ×3 per segment; `render` returns four values | **Fixed** — `render` returns `renderedSeg`; `play` resolves only on a generation change; `spokenFor(text, name)` |
| CQ-11 | Important | Height-compact player deleted | **Fixed** — `radioBreakpoint(compact)`: a compact frame takes the narrow player |
| CQ-12 | Important | Phantom APIs and undefined helpers in P4 | **Fixed** — P4 rewritten on real symbols; Task 4.0 scaffolding; `Glyphs.Down`/`Rail` added explicitly |
| CQ-13 | Minor | `Resolution.Installing` derivable | **Fixed** — `Resolution.WantsInstall(host)` |
| CQ-14 | Minor | `ToneMemo` for ~1 ms of arithmetic | **Fixed** — deleted; `AlertTone` per takeover |
| CQ-15 | Minor | `Reports.Voice` dead | **Fixed** — deleted in 2.1 |
| CQ-16 | Minor | `voicesSeen` rescans the cache | **Fixed** — a byte bound (`maxCachedBytes`, 40 MB) with eviction on insert; no scan |
| CQ-17 | Minor | `WAT1050` for Watches; `Storm.Label() == "Maritime"` collides with the role | **Fixed** (`Watch1050`); the label: **ESCALATE E-5** — the mock's word ("Maritime") vs the class's meaning (tropical/winter storms anywhere) |
| CQ-18 | Minor | Condition-less `Resolve` loop; `Limited` fails open; three stacked timeouts | **Fixed** — counter loop + invariant; fail-closed `deadVoice`; the Limiter is the one bound owner (2.7, the 2-minute wrap removed in round 2) |
| CQ-19 | Minor | Duplicate role list; the import test; `MarineFor` vs `harmonizeMarine`; `FilterUGC`'s `county` | **Fixed** — `cast.Assignable()`; the import test deleted; shared `mergeMarine`; the parameter renamed |

## 3. Docs Quality — DQ

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| DQ-1 | High | The share switch survives in six places | **Fixed** (`ebfff81`; the last comment in `voice-architecture.md` this round) |
| DQ-2 | High | P1 fixture in the share model | **Fixed** (`ebfff81`) |
| DQ-3 | High | Five classes / `ToneFor` in plan.md vs six / `ToneName` in P1 | **Fixed** (`ebfff81`; plan §4 this round) |
| DQ-4 | High | The on-air chip in nine places | **Fixed** (`ebfff81`; plan §2.1 and objectives NFR-6 this round) |
| DQ-5 | High | Non-goal contradicts FR-6 on the CWF | **Fixed** (`ebfff81`) |
| DQ-6 | Medium | `[T]` absent from objectives; the grep misses README | **Fixed** — NFR-8 names `[T]`, `radio-min`, the README key mentions; the grep extended |
| DQ-7 | Medium | plan.md vs implementation-plan on status and batches | **Fixed** — §4 synced; §2.7/§3.5 point at the mocks |
| DQ-8 | Medium | Config shape lacks the cast mode key | **Fixed** (`Radio.Cast` in §2.1) |
| DQ-9 | Medium | FR-12's N fixed nowhere in PLAN | **Fixed** — §1.3 states N = 2 Linux / 3 macOS + 1 reserved and why; FR-12 states it without the bracket |
| DQ-10 | Low | Objectives header frozen at MVS-D-22 | **Fixed** (`ebfff81`, v1.2.0) |
| DQ-11 | Medium | `maritime-report.md` §7 pre-MVS-D-21 wording | **Fixed** (`ebfff81`: "wording of record: P3 Task 3.3") |
| DQ-12 | Low | Line citations drift | **Declined** — line cites are re-checked at each batch's pre-code gate (the build log's first step); symbol cites used where the symbol is stable |
| DQ-13 | Low | Helpers named as existing that do not | **Fixed** — P4 rewritten; `Glyphs.Down/Rail` marked as additions |
| DQ-14 | Medium | `where-things-happen` rows P2/P4 break are in no task | **Fixed** — 2.6 (rename rows) and 4.9 (cast/Setup/`[S]` rows) |
| DQ-15 | Medium | User docs miss the config keys and the scripts folders | **Fixed** — 4.9 |
| DQ-16 | Medium | No plain-language front | **Fixed** — §0 + glossary in plan.md and implementation-plan.md |
| DQ-17 | Low | Cross-feature codes without a legend | **Fixed** — the Legend in implementation-plan.md |
| DQ-18 | Medium | Class keys hand-copied ≥ 8 times | **Fixed** — `cast.Classes()` is the named owner; the config mirror deleted; docs cite `tones.md` §2 |
| DQ-19 | Medium | Preset parameters in five places | **Fixed** (`ebfff81`: plan §2.5 points at `tones.md` + P3 3.6) |
| DQ-20 | Medium | Mixed amendment patterns | **Fixed** — header lines on plan.md and implementation-plan.md; `setup.md` records MVS-D-25..28 |
| DQ-21 | Low | Superseded blocks kept | **Fixed** (`ebfff81`) |
| DQ-22 | Low | The classification rule stated three ways | **Fixed** (`ebfff81`: `tones.md` §2 is the rule of record) |

## 4. Project Hygiene — PH

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| PH-1 | High | P4 breaks `make lint-imports` | **Fixed** — string-keyed `CastView`/`ToneClass`/`CastRow`; the parity test in `app` |
| PH-2 | High | Deleting `modal_chooser.go` deletes the theme chooser | **Fixed** — `modal_theme.go` (pure move) before the delete |
| PH-3 | High | Declsets not re-captured where P3 changes decls; `-update-declset` on `platform/term` | **Fixed** — the packages per batch in the file map and the gates; `platform/term` dropped |
| PH-4 | Medium | Planned code trips P10 with no disposition path | **Fixed** — no `reflect`/recursion; predicted rows listed per gate for ratification |
| PH-5 | High | The two plan reviews exist nowhere | **Fixed** — `plan-review-1.md`, `plan-review-2.md` (every ID, incl. the eleven uncited) |
| PH-6 | Low | Verify commands not runnable | **Fixed** — full paths, `-update-golden`, the framework `A2DH` path, no phantom filename |
| PH-7 | Medium | `voiceList` deleted then read | **Fixed** — kept (the UAT 85 snapshot) |
| PH-8 | Medium | Tone-class set owned twice | **Fixed** (see CQ-2) |
| PH-9 | Low | Dead parameters / duplicate helpers | **Fixed** — `checkbox(on)`; `slices.Contains`; `VoiceName` deleted in 4.2 |
| PH-10 | Medium | Review-round labels in planned code comments | **Fixed** — swept from P1–P4 code blocks; the *why* kept |
| PH-11 | Medium | No Setup golden task | **Fixed** — three Setup goldens in 4.12 |
| PH-12 | Medium | `render.Columns` guessed | **Fixed** (see BQ-14) |
| PH-13 | Medium | No `gates.md` / `release-checklist.md` / `linux-validation-protocol.md`; one p10 json overwritten | **Fixed** — the three files; `p10-p{1..4}.json` |
| PH-14 | Low | The whole-feature file map drifts | **Fixed** — regenerated from the batch maps |
| PH-15 | Low | P0 worktree path / removal | **Fixed** — under the scratchpad; `git worktree remove` ends 0.1 |

## 5. Principal Architect (phase lens) — PA

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| PA-1 | High | The resident path contradicts AX-1's ratified (C) and ships unrunnable | **Fixed** — cut (E-1) |
| PA-2 | High | `[M]` changes meaning silently for a 0.13.0 muted listener | **ESCALATE E-6** — recommendation: accept (MVS-D-26 rules the words always read; the CHANGELOG "Changed" line and the release checklist say so); the alternative (map `ticker_muted = true` to a "silence" mode) contradicts the ruling |
| PA-3 | Medium | A failed `announce` ends a working stream | **Fixed** — non-fatal: reported on the marquee, the segment plays |
| PA-4 | Medium | The registry lives in four shapes; the diagram's `config --> cast` edge is false | **Fixed (partial)** — the edge removed; `cast.Assignable()`/`Key()` drive `castConfig`, `castRows`, `CastView`; the TOML struct stays hand-written by design. A `Roles() []RoleInfo` table: **Deferred** (backlog; the cost of a role is stated in data-shape) |
| PA-5 | Medium | The Source rate is the voice's; a 16 kHz voice is unassignable | **Deferred** (backlog) — every catalogue voice is 22 050 Hz today; a new-rate voice arrives with a resampling decorator; the `Validate` rate rule and `SetVoice`'s check are gone |
| PA-6 | Medium | `render.Columns` does not exist; the breakpoint hard-coded | **Fixed** — `setupPlan` on `twoColumnsWidth` + `panelChromeFor`; `sideBySide` |
| PA-7 | Medium | `scrollSetupToFocus` has no row source; the iota as a state machine | **Fixed** — the `setupRow` table; `setupLine{text, focus}` from every builder; `setupFocusRow` |
| PA-8 | Low-Med | Save re-marshals every time; an unparsable old file drops keys silently | **Fixed** — re-marshal only when kept > 0 (byte-stable test); an unparsable old file is not merged and `[S]` notes it |
| PA-9 | Low-Med | Resident `r.mu` held across the render | **Moot** (E-1) |
| PA-10 | Medium | Measurements "owed before the design is pinned" after ratification | **Fixed** — moot with E-1; P0 kept as the "before" |
| PA-11 | Low-Med | An install landing hands over mid-sentence | **Fixed** — soft (`Invalidate`) vs hard (`Recast`) generations; a test pins each |
| PA-12 | Low | Key vs name on Linux; 48-rune names as identity | **Fixed** — the cache and the hand-over compare the full `Voice.Name()`; truncation only for speech/marquee; `displayName` for `[S]` |
| PA-13 | Low | Large zone geometries re-read per cycle | **Fixed** — centroids memoised on the Provider |
| PA-14 | Low-Med | "2–5 minutes" claim; N unstated; eleven scripts | **Fixed** — the claim dropped; N = 6 (11 keypresses); folding the scripts: **Deferred** to RAT-5 (the HUM LEAD's wording) |

## 6. Accessibility — A11Y

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| A11Y-1 | High | `←/→` undiscoverable | **Fixed** — `←→  Voice` chip beside `p  Preview` while a picker is focused (OP-1 for the HUM LEAD: the mock's row has neither) |
| A11Y-2 | High | The cast table not on the non-TUI surface | **Fixed** — `report --verbose` gains a `cast:` block (4.8) |
| A11Y-3 | High | `[M]` with an empty set mutes nothing while the label says muted | **Fixed** — `cast.Muted`: mode mute + empty set = every class; the support line under "Mute:" (OP-3) |
| A11Y-4 | Medium | "Mute Severe Alerts" label in `body.go` | **Fixed** — relabelled; `body.go` in the map; the grep names it |
| A11Y-5 | Medium | Inconsistent keyboard model; tab stops on every row | **Fixed** — one rule (tab = questions, ↑↓ = rows in a group, space/←→ = pick, enter = next / save). Re-wording the footer to "↑↓ Move · space Pick": **Declined** — the mock's chips are the HUM LEAD's words |
| A11Y-6 | Medium | No signal for no voices / no audio | **Fixed** — the no-audio note; the chip hides without `PreviewVoice`; `cycleVoice` from an empty list is a no-op with the note |
| A11Y-7 | Medium | Installed state not exposed | **Fixed (variant)** — a note line under the focused picker (a suffix inside the 16-cell picker would break the mock's column) |
| A11Y-8 | Medium | Seven pickers, not six | **Fixed** — `focusCastSingle` is a picker row |
| A11Y-9 | Medium | `--ascii` parity: `Down`, rails, `radioMark` | **Fixed** — `Glyphs.Down`/`Rail`; `radioMark` unchanged; `›` via the existing mark |
| A11Y-10 | Low | Bare keys via a non-existent token; `asciiKey` bypassed | **Fixed** — `o.KeyCap` is the chip on every surface (it routes `asciiKey`); no bare variant |
| A11Y-11 | Low | Contrast pairs on the modal ground | **Fixed** — `aaPairs` rows in 4.9 |
| A11Y-12 | Low | Dual-tone vs 1050 Hz share a rhythm | **Deferred** — an M3 note: the words carry the class |

## 7. InfoSec — SEC

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| SEC-1 | Medium | Raw config strings reach `[S]` and the dump | **Fixed** — `PlainLine` at `castRowOf`, at `unknownRadioKeys` (capped 32) and at `Merge`'s note; the hostile fixture asserts no `\x1b`/`\n` in `[S]` |
| SEC-2 | Medium | The coastal forecast is one unbounded utterance | **Fixed** — the forecast goes through `Segments` (280-char pieces) after the 3-period cap |
| SEC-3 | Low | `MarineZoneFor` fan-out unbounded | **Fixed** — ≤ 32 zones, a 20 s budget for the call |
| SEC-4 | Low | The discovery-trust window is unbounded | **Fixed** — `exec.CommandContext` 30 s; on timeout the curated list closes the window |
| SEC-5 | Low | Resident path reads any printed path | **Moot** (E-1) |
| SEC-6 | Low | `mergeUnknown` could replace a known scalar with a table | **Fixed** — `keepUnknown` copies only paths the strict decoder reported unknown; a known key is never overwritten (fixture) |

## 8. Junior Developer — JD

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| JD-1 | Critical | `modes/tty` imports `domains/` in P3/P4 | **Fixed** (PH-1, PR2-5) |
| JD-2 | Critical | `where-things-happen` pins break at the P2 gate | **Fixed** — the rows land in 2.6 (P2 gate) and 4.9 |
| JD-3 | High | Task 2.11 "as v1" | **Fixed** — cut (E-1); the task records why |
| JD-4 | High | P4 scaffolding undefined | **Fixed** — Task 4.0 |
| JD-5 | High | Compile errors in P4 GREEN code | **Fixed** — real symbols; `radioMark` reused unchanged |
| JD-6 | Medium | `voiceList` contradiction | **Fixed** (PH-7) |
| JD-7 | Medium | Wrong golden/declset flags and packages | **Fixed** (PH-3/PH-6) |
| JD-8 | Medium | P10 convention untaught | **Fixed** — the Legend; each gate names the framework path and the ledger step |
| JD-9 | Medium | No build logs; literal ellipses; `07-readiness/validate/` absent | **Fixed** — the batch-exit checklist; `validate/m3.md` named in gates.md |
| JD-10 | Low | The rename list is incomplete | **Fixed** — "grep `narrat`; keep `narration*`" in 2.6 |
| JD-11 | Low | Forward-reference drift | **Fixed** — the index rewritten |
| JD-12 | Low | Untested assumptions (`omitempty`, `ModShift`) | **Fixed** — the pre-code spike list; the test sends `'V'` without `Mod` |
| JD-13 | Low | README/extending paragraphs | **Fixed** — 4.9 |
| JD-14 | Low | The commit-trailer rule unstated | **Fixed** — every gate says it |
| JD-15 | Low | Placeholder noise in 2.9 | **Fixed** — deleted |

## 9. Performance — PERF

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| PERF-1 | Medium | `[S]` resolves nine roles per frame | **Fixed** — `Resolutions()` memoised per cast generation (`castGen`) |
| PERF-2 | Low-Med | `play` re-resolves on the writer goroutine | **Fixed** — resolved once in `render`; re-resolved only on a generation change |
| PERF-3 | Low | `voicesSeen` scan | **Fixed** (CQ-16) |
| PERF-4 | Med-Low | The cache ceiling triples with no byte bound | **Fixed** — 40 MB byte bound; the gauge reports it; the phase-B number is in `perf-protocol.md` §0 |
| PERF-5 | Medium | The resident pool unbounded | **Moot** (E-1) |
| PERF-6 | Med-Low | Zone geometries re-decoded per cycle; N sequential GETs | **Fixed** — centroids memoised; the fan-out capped and budgeted (concurrent fetch: **Declined** — ≤ 10 zones once per 24 h is not worth a worker pool) |
| PERF-7 | Med-Low | The Setup pin measures the wrong path; the "before" discarded | **Fixed** — `TestSetupAllocBudget` in the tree at P0 (hit and miss); `setupGen` replaces the `%+v` fingerprint; P4 re-pins |
| PERF-8 | Low | `[M]` reloads the config and invalidates the broadcast | **Fixed** — `setTones` only |
| PERF-9 | Low | A preview queues silently on Linux | **Fixed** — the "waiting for a free voice slot" note |

## 10. Safety-critical (P10) — SC

| # | Sev | Finding (short) | Disposition |
|---|---|---|---|
| SC-1 | Important | `reflect` on the Save path | **Fixed** (CQ-1) |
| SC-2 | Important | Recursion at three sites, invisible to the tool | **Fixed** — none remain (`keepUnknown` is flat) |
| SC-3 | Medium | Condition-less `Resolve` loop | **Fixed** — `maxDepth = 3` counter + `invariant.Check(r == All)` |
| SC-4 | Medium | Unchecked index into literal arrays | **Fixed** — `[roles]string`/`[classes]string` sized tables with range guards; a table test over the range |
| SC-5 | Important | The rename orphans three ratified ledger rows | **Fixed** — 2.6 re-keys them; 2.12/4.13 name the ledger steps |
| SC-6 | Medium | Predicted name-graph false positives unlisted | **Fixed** — `limit.go:Say` predicted in 1.14; the resident rows moot |
| SC-7 | Important | `cast` density row certain, unpredicted | **Fixed** — real invariants in `Resolve`/`Validate`; the row predicted in 1.14 |
| SC-8 | Medium | `Limited` fails open | **Fixed** — `brokenVoice` |
| SC-9 | Medium | The wait inside `acquire` is unbounded | **Fixed** — the timeout wraps the wait; a test with the slot held |
| SC-10 | Medium | The Director path drops the reason | **Fixed** — `render` records it (`setDetail`) |
| SC-11 | Low | Complexity at the ceiling | **Fixed** — `play` resolves once; `setupCastKey` split if gocyclo > 15 |
| SC-12 | Low | Allocation in loops; the column helper undecided | **Fixed** — `sideBySide` decided; `AlertTone`'s gap fill within the pre-sized cap noted in 2.0 |
| SC-13 | Low | `tree_hash` never recorded | **Fixed** — `gates.md` batch record |

## 11. Escalations for the HUM LEAD (presented with the Plan of Record; restated after round 2)

| ID | Question | Lenses | Recommendation |
|---|---|---|---|
| E-1 | Confirm the cut of `ResidentPiper` / `piper_mode` from 0.14.0 (amends AX-1's "build the seam for all three": the seam is `synth.Voice` + `buildVoice`; the measurement stays on the UAT list as 0.15.0 input). Record as MVS-D-29 | BQ-4, CQ-7, PA-1, PERF-5, SEC-5, JD-3; round 2: DQ2-1, PA2-12 | **Confirm** |
| E-2 | Scope. The options that can actually ship (round 2, BQ2-3): **(a)** whole — P1 13 · P2 13 · P3 7 · P4 14 tasks; **(b)** whole minus the maritime report (P3 Tasks 3.1–3.5: one new NWS endpoint, the eleven scripts, ~1 000 spec lines; re-rules MVS-D-4/14/18); **(c)** whole minus the per-class mute's UI (P4 Task 4.5 — the per-class *model* stays in config 1.8, `cast.Muted` 1.6, the Director 2.7 and the read/ticker rule 2.9; re-rules MVS-D-26/28: `[M]` stays all-or-nothing on screen); **(d)** whole minus the three-breakpoint Radio panel (P4 Task 4.3; `[V]`/`[T]` still retire via 4.2 and the panel keeps today's two layouts — re-rules MVS-D-23's mock). Residue the cuts leave: (b) leaves the *Maritime* role, its Setup override picker and the "Maritime" tone label with no report behind them (they can be hidden, not removed, without re-ruling the mock). A "tones + two-picker cast, UI later" split is **not** on offer: the pickers are P4's Setup work and MVS-D-3 couples `[V]`'s retirement with the Radio panel | BQ-6/7/8/13; BQ2-3; BQ3-7 | **(a)** — each batch is UAT-able alone, so (b), (c) or (d) can still be taken at the P2 or P3 gate |
| E-3 | The problem statement's standing. M3 is now two A/B trials against 0.13.0 (AM-17). Options: **(a)** keep Candidate A (the *voice* that follows is the cue) and the "default-on answer" wording; **(b)** keep Candidate A, drop the "default-on answer" wording from objectives §0 / FR-11 / AM-9 (the tones are the fresh-install *cue*); **(c)** re-lock as Candidate B — listener delight — with M3 as the class/cast trials (round 2's Business lens recommends this: it is what the release delivers, and a metric that cannot move is theatre) | BQ-1/2/11; BQ2-2 | **(b)** now; **(c)** if 0.13.0 scores ≥ 9/10 on the cast trial at UAT |
| E-4 | A default alert voice on a fresh install (macOS free; Linux +63 MB background) vs MVS-D-8's one voice | BQ-3 | **Keep MVS-D-8** for 0.14.0; revisit with the M3 result |
| E-5 | The Setup label "Maritime" for the storm class (tropical/winter storms anywhere — a Kansas blizzard) | CQ-17 | "Tropical / Winter Storms" — or keep the mock's word |
| E-6 | `[M]` = tones only: a 0.13.0 listener with `ticker_muted = true` hears the words again after upgrading. Options: **(a)** accept, with the CHANGELOG line; **(b)** map the legacy `true` to "mute tones and words" for one release, with an `[S]` note; **(c)** a one-time in-app note on first launch | PA-2; BQ2-7 | **(a)** — MVS-D-26 rules the words always read, and the affected listener is the HUM LEAD |
| E-7 | Explicit rulings on the three "not objected" items (SEV-0: silence is not a ruling — BQ2-5, AM-20): RAT-3 Blizzard Warnings in the storm class; RAT-5 the maritime wording as written in P3 Task 3.3; RAT-6 the three-period cap on the coastal forecast | BQ2-5 | ratify all three; RAT-5 "reword at will" stands |
| E-8 | AX-7: ship the zone resolver (A) — a new NWS endpoint, ≤ 32 zones, centroid memo — or (B) only: the first nearshore block after the synopsis, no geometry (MVS-D-14 said "for free"; (A) is not free) | BQ2-6 | **(B) only** for 0.14.0; (A) returns if UAT reads the wrong stretch — P3 Task 3.5 keeps (A)'s code in the plan as the fallback's sibling until ruled |
| E-9 | **Ratify the five amendments the PLAN red-team made to ratified material** (a ratified metric or architecture axis is the HUM LEAD's to change): AM-17 (M3 → two trials: the class trial absolute ≥ 9/10, the cast trial A/B vs 0.13.0 with pass = 0.14.0 ≥ 9 and 0.13.0 ≤ 7), AM-18 (M2's target "≤ 5 actions" → the measured 11 keypresses pinned ≤ 11 — a regression pin, not a target; OP-4's Save chord is the way to a lower number), AM-19 (M5 admits the one legitimately silent Linux row), **AM-21** (M4 → a code-path pin plus an absolute live budget — one instrument, two parts, round 3/4) and **AX-4's second reserved slot** (D-R3-1: the ratified cap gained a slot so a Director job never queues behind the read-ahead). Ratify as MVS-D-30..34, or send any back | BQ3-5, BQ4-7, PA3-3 | **Ratify all five** — or rule M2 a target and take OP-4's chord |
| E-10 | **Unattended downloads from a hand-edited config.** `setCast → ensureRoleVoices` installs every Piper voice the cast names and the host lacks, in the background, on launch — up to 6 × 63 MB ≈ 380 MB, pinned and size-limited but unasked (the `p`-asks-once rule covers previews only). Options: (a) as planned; (b) a per-session cap (two keys, then `[S]` "install on next Save"); (c) install only on a Setup save, never on launch | SEC3-5 | **(b)** |
| E-11 | **NFR-7's lint claim.** It promises `golangci-lint run ./...` and `staticcheck ./...` "at every exit"; **neither has ever run** in `make verify` or in CI, and `golangci-lint` is red today on one pre-existing `QF1001` outside this feature (plus `third_party/`, which every other gate in this repo excludes and this config does not). The gate row was added by round-3 remediation, not by any shipped release. Options: (a) amend NFR-7 to the gates that exist and keep "add a `make lint` target" on the carried follow-ups; (b) adopt the linter now — one style fix plus mirroring the `third_party/` exclusion | PH4-3 | **(a)** — do not adopt a new linter mid-feature |
| OP-1..5 | The `←→  Voice` chip; the DATA row wording; the "nothing ticked = every class" support line and the masthead count; `↑↓  Pick` vs `↑↓  Move · space  Pick` and an always-present Save chip; the chip row as a pinned footer | A11Y-1/3, A11Y2-1/5/8/15, mock fidelity | as written in `p4-ui.md` |

## 12. Round 2 — fresh lenses on the remediated tree (`3fcd822`)

**Verdicts:** Docs **SHIP-WITH-CONDITIONS** (0 High / 4 Important / 11 Minor) · Hygiene **SHIP-WITH-CONDITIONS**
(3 Important / 9 Minor) · a11y **SHIP-WITH-CONDITIONS** (8 Important / 7 Minor) · Perf **SHIP-WITH-CONDITIONS**
(3 Important / 6 Minor) · Safety **SHIP-WITH-CONDITIONS** (2 Important / 8 lesser) · InfoSec **SHIP-WITH-CONDITIONS**
(2 Important / 4 Minor) · Business **ESCALATE** (3 Important / 7 Minor) · Architect **NO-GO** (2 Important, both
in P2's hand-over/limiter; the rest Medium/Low) · Junior **NO-GO** (2 Critical compile breaks, 5 Important
RED/GREEN contradictions) · Code **NO-GO** (9 Important, none architectural). **Round-2 findings: 118.**
The three NO-GOs share one cause — the round-1 remediation was written faster than it was compiled — and every
lens said so: "single-task rewrites", "mechanical", "a fix pass over P2/P4 plus `go vet`". The remediation
below is that pass; round 3 re-verifies it with fresh lenses.

**Round-1 rows that did not verify (re-opened and closed in this pass):** A11Y-2 (the `report --verbose` seam
was the dump — now a real `cast:` block, 4.8); BQ-12 (the P0 baseline method — now a benchmark on both trees);
CQ-1 (`e.Key` is a method); CQ-2/CQ-4 (`cast.Validate` was never wired — now `setCast`/`castChanged` →
`[S]`); CQ-8/CQ-10/PA-3 (the Recast path rendered twice on the writer — now `voiceFor` + one Say, the boundary
line rendered ahead); CQ-19 (`mergeMarine` was a comment — now code); PA-1/DQ-21 (the resident cut half-propagated
— now plan §0/§1.1, perf-protocol §3, RS-23, the P2 header); DQ-22 (two rules of record — tones.md §2);
PH-3/JD-7 (`./platform/term` in the P4 declset gate); PH-9 (`VoiceName` undeleted); SC-4 (`Link.String`/`ToneName`
unguarded); SC-5/SC-6 (the ledger re-key under rename detection; phantom predicted rows).

### 12.1 Dispositions by lens (round-2 IDs)

| Lens | Fixed | Declined / Deferred (reason) |
|---|---|---|
| Docs DQ2-1..15 | all Fixed — per-ID rows in `red-team-plan-round2.md` (E-1's MVS-D-29 number pending) | — |
| Hygiene PH2-1..12 | all Fixed (PH2-1 by the benchmark method; PH2-2 the paths out, the checklist grep self-clean; PH2-11 the review worktree pruned) | — |
| a11y A11Y2-1..15 | 1 (pinned footer, OP-5) · 2 (notes under the focused row) · 3 (five tab stops: `groupKey`) · 4 (`[]setupFocus` per line) · 5 (EVENTS as two radios under the one rule; OP-4) · 6 (Reason printed; `Glyphs.Arrow`) · 7 (`report --verbose` real seam) · 8 (the masthead counts; `[M]` keeps the set) · 9 (untinted) · 10 (`Glyphs` for `›`/`⚠`/`✔`) · 11 (`RailGlyphs`) · 12 (no-voices note) · 13 (`p` asks once) · 14 (notes cleared on focus; the deck clears on audition end) | 15 (a Save chord): **OP-4** for the HUM LEAD — a mock deviation |
| Perf PERF2-1..9 | 1 (`voiceFor` in the chunk loop; one Say) · 2 (`renderHandoff` ahead, cached per from → to) · 3 (`TestWriterNeverStarvesAcrossHandOvers`; the 0.7 s budget in perf-protocol §1) · 4 (dup guard in `store`) · 5 (`setupBody` built once, memoised) · 6 (`[S]` rows memoised until a cast/host change) · 7 (`castChanged` on discovery and installs) · 8 (3.6 aligned; the gap fill in place) · 9 (P0 informational, P4 spike-then-pin) | PERF-4's 40 MB: kept, now justified in NFR-3 (one voice's cycle ≈ 29 MB + a second correspondent's sections; the gauge reports it; the phase-B numbers in §4 bound the whole) |
| Safety SC2-1..10 | 1 (counter-form `store`) · 2 (`p10-unmatched.sh` rename-aware, Task 2.6) · 3 (`handOver` non-fatal) · 4 (`Link.String`/`ToneName` guarded) · 5 (`setupPickerKey` split; the `Resolve` ceiling noted) · 6 (rows only from the tool; predictions corrected) · 7 (`AlertTone` guard) · 8 (`ToneClassCount` parity test) · 9 (the 2-minute wrap gone; 3.6 fixed) · 10 (`fetchCentroid` invariant) | — |
| InfoSec SEC2-1..6 | 1 (go-toml v2.4.3 + `recover`; the fixture) · 2 (`installFailed` + `installRetryAfter`) · 3 (the trust window deleted: `Discovered()` is a closed list at every moment) · 4 (`SetupNoteMsg` PlainLine'd on receipt) · 5 (paths as segments end-to-end — no alias; the array-of-tables limitation recorded in NFR-5) · 6 (`unknownRadioKeys` renders plain; the `[S]` hostile assertion in `app`) | — |
| Junior JD2-1..19 | all Fixed (1 `e.Key()`; 2 imports; 3 the RED assertion dropped; 4 wording; 5 `CastMode`; 6 `Muted: nil`; 7 3.6; 8 `slices`; 9 `press(rune)`; 10 five stops; 11/12 RED strings regenerated from the mock, `cycleVoice` from the displayed name, non-nil maps; 13/14 imports, `c.String()`, `VoiceByName`, one `Installed`; 15 the parity relation stated; 16 `Voice:` out of the grep; 17 `./platform/term` out; 18 the P2 header; 19 `ToneRate` in the block) | — |
| Business BQ2-1..10 | 1 (the benchmark method, both trees; `breaking:`/`tone:` debug lines) · 2 (AM-17: two trials, a comparative pass rule) · 3 (E-2 restated with the options that can ship and the task counts) · 4 (numbers in perf-protocol §4: app ≤ 126 MB, summed `piper` ≤ 2 × §3) · 8 (AM-18: ≤ 11, a regression pin) · 9 (AM-19) · 10 ("by ruling" rows in the trace) | 5 → **E-7**; 6 → **E-8**; 7 → E-6's options |
| Architect PA2-1..15 | 1 (the hand-over rendered ahead; `voiceFor` on the writer) · 2 (every Director job carries `WithPriority` — plan §1.3, FR-12) · 3 (`castChanged` from `listVoices`) · 4 (the `[M]` hook writes `Tones.Mode`; `Save` owns the mirror) · 5 (`[]setupFocus`; `groupKey`; the 20-row on-screen test) · 6 (`openSetupAt` scrolls) · 7 (`setupSave`) · 8 (group pickers cycle through "" = inherit) · 9 (breakpoints in outer columns, pinned ±1) · 10 (the phantom symbols reconciled) · 11 (= SEC2-2) · 12 (plan prose) · 13 (lines trimmed to the column) · 14 (`layoutRows` pre-build kept) · 15 (NFR-5 records the limitation) | — |
| Code CQ2-1..18 | all Fixed (1 `voiceFor`; 2 `CastMode`; 3 `DefaultVoice` fallback; 4 `cast.Validate` wired; 5 non-nil maps + displayed-name cycling; 6 membership assertions; 7 the compile fixes incl. `deadVoice`; 8 the phantoms; 9 the mock's slots; 10 `castGen` deleted, `castChanged`; 11 store unconditionally; 12 the wrap gone, the 1 s note dropped; 13 plan §2 regenerated; 14 `Save` owns the mirror; 15 guards; 16 `setupBody`; 17 the template exec outside `s.mu`; 18 the held-slot test) | — |

Every round-2 finding, one row per ID, is in `red-team-plan-round2.md` (the audit trail).

### 12.2 Decisions this pass made without a ruling (listed so the HUM LEAD can reverse them)

- **D-R2-1** Every Station Director job carries `WithPriority` (FR-12 amended in the objectives): the reserved
  slot is the Director's, not the takeover's alone — a read could otherwise wait its whole bound after the
  broadcast was already ducked.
- **D-R2-2** The macOS discovery-trust window is deleted: `Discovered()` answers with the curated list until
  `say -v ?` lands. A config name outside the list falls back at once instead of being trusted for ≤ 30 s.
- **D-R2-3** The hand-over line is rendered ahead of the air and cached per (from → to); the listener's own
  mid-segment Recast is the one render the writer waits for.
- **D-R2-4** A failed background install waits ten minutes before the next attempt (`installRetryAfter`); a
  Setup save resets it.
- **D-R2-5** go-toml is bumped to v2.4.3 at P1 (a dependency change, inside `make verify`'s tidy/vuln).

## 13. Round 3 — fresh lenses on the round-2 remediation (`1d87831`)

**Verdicts:** Code **NO-GO** (2 Critical / 6 Important / 14 Minor) · Junior **NO-GO** (3 Critical / 12 Important /
10 Minor) · Architect **NO-GO** (1 Critical / 3 Important / 6 lesser) · Perf **NO-GO** (1 Critical / 3 Important /
5 Minor) · Business **ESCALATE** (5 Important / 4 Minor) · Docs **SHIP-WITH-CONDITIONS** (8 Important / 7 Minor) ·
Hygiene **SHIP-WITH-CONDITIONS** (4 / 8) · a11y **SHIP-WITH-CONDITIONS** (5 / 7) · Safety **SHIP-WITH-CONDITIONS**
(3 / 5) · InfoSec **SHIP-WITH-CONDITIONS** (0 / 6, **converged**). **Round-3 findings: 128** (an earlier draft of
this line said 131; the verdicts above and the ID ranges in §13.1 both sum to 128 — 22+25+10+9+9+15+12+12+8+6).
Round 3 has no separate per-ID file as round 2 does; §13.1 carries every ID inline with its disposition, which is
the same information in one fewer file. Four lenses found the
same two defects independently — `cast.Validate` called under `d.mu` (a self-deadlock through `Discovered()`) and
`play`'s soft-path whole-segment render on the writer — and three found the dead `case " "` (bubbletea v2 names
the key `"space"`). Every Critical is in code round 2 wrote. The lenses' shared instruction for the next pass:
compile and run the blocks; make round 4 a compile-first pass on P2 Task 2.7, the Source's `play`/`renderLoop`,
and P4 Tasks 4.4–4.6.

### 13.1 Dispositions by lens (round-3 IDs; the remediation commit follows this section)

| Lens | Fixed | Declined / Deferred / Escalated |
|---|---|---|
| Code CQ3-1..22 | 1 (Validate outside `d.mu`, both tails; a timeout test) · 2 (`case "space"` everywhere, incl. today's `setup.go:171`; the probe recorded in 4.0) · 3 (`_ string`) · 4 (`Cached() == 3`) · 5 (the root seeded from `voiceList[0]`; fixtures name it) · 6 (the mock test has voices and a preview hook) · 7 (the reader paced; zero writer Says asserted) · 8 (`setupMemo` allocated in `NewDashboard`; nil-safe) · 9 (gofmt on every touched file, the claim dropped) · 10 (`from` deleted) · 11 (`cycleVoice(list, shown, forward)`) · 12 (the four comments) · 13 (`setupRoles()` from `Role.Key()`) · 14 (the nil-error branch) · 15 (imports listed; `sort` used) · 16 (plan §2.2–2.4 rewritten to the batch code) · 17 (the deck owns the note) · 19 (renamed; the engine is concrete) · 20 (`→` pinned) · 21 (`hostFacts` + `CastReport` as code) · 22 (`reportPause` reused) | 18 — `Reports.Voice` in P1 then deleted in P2: **Declined** (a transitional field with a stated reason; P1 must build alone) — the dead `Tail("")` branch and the duplicate `c.role` assignment are Fixed |
| Junior JD3-1..25 | all Fixed except as noted (1 space; 2 the lock; 3 a P1 `castChanged` stub; 4–7 the compile fixes; 8–13 the RED/GREEN alignments; 14 row 36 of `where-things-happen`; 15 the P4 order note; 16 the third `SetVoice` caller; 17 gofmt; 18 `-update-golden`; 19 `root.go`; 20 "add"; 21 the prose-only pieces written as code — the tone hook, the four app helpers, the rename-aware script, `flakyVoice` on a prefix; 22 the one-liners; 24 `setupMemo`/`resized`/`flipMute`; 25 `→`) | 23 — the go-toml bump: **made conditional** on the spike reproducing (two probes disagreed) |
| Architect PA3-1..10 | 1 (the lock discipline, stated in code) · 2 (the soft path plays as rendered — no writer render; the hard path is the exception, non-fatal) · 3 (two reserved slots; FR-12/§1.3/RS-1 restated; D-R2-1 restated as D-R3-1) · 4 (`setCast` clears `installFailed`; `[S]` wording) · 5 (every `setupGen` writer listed, incl. `handleResolved`, `applyCommitted`, the snapshot arrivals, the resize) · 6 (`cycleVoice` by displayed name with `""` at index 0) · 7 (the line text in the hand-over key) · 8 (plan §2.2/§2.3, data-shape §3/§4, the P2 comments) · 9 (`resized()`, the `modalLines` Setup case) | 10 — one composite save / `Recast` only on a resolution change: **Deferred** (BUILD-time; four saves at keypress rate are milliseconds; noted in 2.10) |
| Perf PERF3-1..9 | 1 · 2 · 3 (paced reader + writer-Say count) · 4 (`warmHandoffs` at `SetResolver`; the inequality in perf §1; Linux row 2.3 listens for the first cycle) · 5 (the numbers in NFR-3 and perf §4) · 6 (one instrument: the Director-level pin + the live absolute budget) · 8 (`hostFacts` shares the deck's discovery under the 30 s ceiling; `setupBody` nil-safe) · 9 (the launch-time `cfg`) | 7 — the zone fan-out inside `next()`: **moot under E-8 (B)**; if (A) ships, warm at `startSynth` (noted in 3.5). 5's newest-largest eviction: **Deferred** — FIFO stands until the soak gauge says otherwise |
| Business BQ3-1..9 | 1 (FR-7, plan §2.2, data-shape §3/§4, the matrix row) · 2 (one instrument; the pin + the absolute budget) · 3 (a true 0.13.0 `piper` row; §3 moved to a 0.14.0 build) · 4 (M3 trial 1 absolute, trial 2's decision rule and blinding) · 6 (the "by ruling" trace rows) · 8 (Linux row 2.13, offline) · 9 (neither paragraph claims "the answer" pending E-3) | 5 → **E-9**; 7 → E-2 restated with (d) and the residue |
| Docs DQ3-1..15 | 1 · 2 · 3 (A-1) · 4 (FR-12 the one owner; RS-1, AX-4, discover-report errata, the P1 comment) · 5 (the Linux precondition) · 6 (§3 retitled and moved; row 1.3 a true 0.13.0 measure) · 7 (perf §1 the one owner; NFR-2 and Task 0.1 cite it) · 8 (`piper/voices`) · 9 (the brief header v1.3.0; objectives Source AM-1…20; the index header) · 10 (both headers) · 11 (pending E-3 in both places) · 12 (RS-6, NFR-2's bracket, A-3) · 13 (`SpokenPeriodsCap`; RAT-3/RAT-6 marked E-7 pending) · 14 (`red-team-plan-round2.md`, one row per ID) · 15 (the errata lines) | — |
| Hygiene PH3-1..12 | 1 (Task 2.9b owns the file; P0 runs it) · 2 (the three rows in gates.md; NFR-7 says 17/17 and names the two lint commands) · 3 (the round-2 file) · 4 (the guard) · 5 (the worktree removed; the checklist line) · 6 (`-update-golden`) · 7 (the journey invocation, three sites) · 8 (the licence file + script, conditional on the bump) · 9 (maps regenerated: `root.go`, `hostfacts.go`, the test files, `harmonize.go`, the script, the benchmark; `nav.go` out) · 10 (the tag `v0.13.0`) · 11 (`-run '^$' -bench …`) · 12 (both headers) | — |
| a11y A11Y3-1..12 | 1 (`KeyChip` out of the pair; a composite self-check) · 2 (notes wrapped at 44 cells, never trimmed, never widening) · 3 (`[S]` TONES line; the count on every masthead form; `report --verbose` `tones:`) · 4 (`SetupNoteMsg{Voice, Text}` drawn under the picker showing that voice; scroll on arrival; `[S]` State carries it otherwise) · 5 (a pick turns the cast on) · 6 (the DATA rule stated as built) · 7 (a digit selects *Within*) · 8 (the header row; `PadTo`) · 9 (the chip's verb) · 10 (`Glyphs.Dash/Ellipsis/Bullet`) · 11 (`TruncateCells`) · 12 (the height budget and its test) | — |
| Safety SC3-1..8 | 1 (`pad` counter-form) · 2 (the CLI pinned in the batch record; one build per release; re-key if it changes) · 3 · 4 (nil guard) · 5 (`strictDecode`) · 6 (`rowOf` guarded; `barWidthFor`) · 7 (the gate runs after staging) · 8 (the `entered` channel) | — |
| InfoSec SEC3-1..6 | 1 (`discoverMacVoices` shared through `hostFacts`; every field plain) · 2 (`capNotes` 32 + de-dup; one count for unknown class keys) · 3 (= PA3-2) · 4 (`maxMaritimePieces = 12`) · 6 (the two fixtures; parents-first order) | 5 — a per-session cap on unattended background installs: **ESCALATE E-10** (a policy: how many megabytes may a hand-edited config pull on launch?) |

### 13.2 Decisions this pass made without a ruling

- **D-R3-1** (restates D-R2-1): the Limiter has **two** reserved slots for the Station Director's jobs — the job on
  the air and the one it suspended — so a takeover never falls through to the render-ahead pool while a read's
  render is in flight. FR-12, plan §1.3 and RS-1 say so; one owner (FR-12).
- **D-R3-2**: a background change (`Invalidate`) never renders on the writer — the rendered segment plays and the
  next re-resolves; the listener's Recast is the single exception and is non-fatal. Hand-over lines are pre-warmed
  at `SetResolver`.
- **D-R3-3**: the M4 instrument is two things with one owner (perf-protocol §1): a Director-level code-path pin on
  both trees, and a live absolute budget (`breaking:` → `tone:` ≤ 250 ms) — because the deck's engine is a
  concrete type and a 0.13.0 live number does not exist.

## 14. Round 4 — the compile-first pass (`c2d61e3`), and the strip

**Verdicts:** InfoSec **SHIP-WITH-CONDITIONS (converged)** · Docs, a11y, Perf, Safety **SHIP-WITH-CONDITIONS** ·
Business **ESCALATE** · Junior **SHIP-WITH-CONDITIONS** (0 Critical / 9 Important — "nothing architectural, no
deadlocks, no writer renders") · Code, Architect, Hygiene **NO-GO**. **Round-4 findings: 118.**

Two lenses did what round 3 asked and **compiled the plan's blocks** in throwaway worktrees. That produced the
phase's best evidence in both directions:

- **The design runs.** Assembled over the real packages under `-race`: `domains/radio/cast` 3/3 · the Limiter
  suite green at `-count=3` including the bound-covers-the-wait property · the whole `Source` seam **17/19** and
  **12/14** across the two lenses, with both hand-over paths, the writer-starvation property (zero writer Says),
  announce-failure non-fatality and every pre-existing synth test green · the config suite green on go-toml
  v2.4.3 · `make pty-severe` green · the Setup row and tone groups matching the mock character-for-character.
- **The written code was full of what a compiler catches.** A test-only `runtimeGOOS` called from production; a
  `WrapLines` arity; a `PadTo` column too narrow for the registry's own labels; `warmHandoffs` with no caller;
  four helpers named and never defined; two RED tests contradicting their own GREEN; a `#!/bin/sh` script given
  bash-4 syntax. None of it design.

**Three findings are kept as decisions, not fixes** (they are what the compile-first pass *discovered*):

- **D-R4-1 — one failure rule, two failures.** A **segment**'s render failure ends the broadcast with its reason
  (there is no audio to play). A **hand-over line**'s failure is reported and the segment plays (FR-5's "never
  silence"). Round 2's fatal-render pin and round 3's non-fatal hand-over were asserting one rule of the other.
- **D-R4-2 — `Invalidate` lands at the next segment *rendered*.** With one segment of look-ahead that is two
  segments later; the already-rendered segment plays as it is. A test asserting "the new voice speaks the next
  segment" cannot pass and should assert *no mid-segment hand-over* instead.
- **D-R4-3 — the go-toml bump is required.** The spike fired: v2.2.4 panics on an escaped quoted key, the recover
  masks it, and NFR-5 then drops unknown keys silently. v2.4.3 passes the suite. P1 Task 1.9 owns it with its
  licence regeneration.

**The strip (this commit).** Per the HUM LEAD's ruling and `AP-PLANCODE-01` (`06-key_learnings/retro-notes.md`
RN-1), the four batch documents were rewritten to **task shape** — file · symbol · contract · test intent ·
verify — and now carry **no Go blocks at all** (7,069 lines → 993). Every round-4 finding whose subject was a
line of that code is therefore **Moot in the plan**; the code itself is preserved verbatim in
`04-development/prior-art/` with a README recording what was proven green and nine named defects (D-1…D-9) not
to copy forward. What survived the strip was folded into the tasks as contracts:

| Survives as | From |
|---|---|
| The two-generation contract and the writer rule (P2 head) | PA4-4/5, PERF4-3, CQ4-2, JD4-2 |
| The failure rule (P2 head) | JD4-1, CQ4-2 |
| No launch-time pre-warm; a per-cycle warm only if Linux row 2.3 shows the gap | PERF4-1/2, PA4-3, CQ4-6, JD4-3 |
| `runtimeGOOS` as a production seam (P1 1.12) | CQ4-1, PA4-1, JD4-7 |
| One shared `discoverMacVoices` under one ceiling (P1 1.12, P4 4.8) | SEC4-1/4, CQ4-7, PA4-8 |
| The lock-discipline sentence, binding on later tasks (P1 1.12, P2 2.7) | PA4-7, PERF4-1 |
| `installState` lands with its consumer (P2 2.7 → P4 4.8) | JD4-4 |
| Overrides inherit from *All Reports*, not the root (P4 4.6) | A11Y4-1 |
| The display-only root seed (P4 4.6) | PA4-10 |
| Cycle by the held entry, with an inherit entry (P4 4.6) | PA4-9, A11Y4-12 |
| Notes wrapped, never truncated; the scroll covers their last line; empty draws nothing (P4 constraints, 4.4) | A11Y4-2/3, CQ4-4, JD4-6 |
| `[S]` columns sized from the registry's widest label (P4 4.8) | A11Y4-4, CQ4-5 |
| The preview failure's reason is not overwritten (P4 4.6) | A11Y4-6 |
| No `KeyChip` in the AA pairs; a composite self-check instead (P4 4.9) | A11Y4-1 (r3), A11Y4-13 |
| "Existing pins this changes" as a first-class part of a task (P2 2.1/2.2/2.6/2.9, P4 4.2) | JD4-9 |
| Declsets before the test line; stage before `make p10` (every gate) | SC4-3, JD4-15 |
| The P0 benchmark is a 0.13.0-shaped **twin**, not the same file (P2 2.9b) | PH4-1, CQ4-8, JD4-8 |
| The gate script fixed **before** the first gate runs (P1 Task 1.0) | SEC4-1, PH4-2, SC4-1/2, JD4-5 |
| Fixtures must be referenced by a case (P1 1.10) | JD4-16 |

**Still open and not folded** — they are readiness or ledger items, carried to the Plan of Record: the lint gate
names a command this repo has never run and which is red today on pre-existing style (PH4-3 — see NFR-7 in the
escalations); the employer-name grep cannot reach zero because of legacy hits in shipped features' docs
(PH4-4); the round-2 audit file's dispositions are a blanket stamp (PH4-5, DQ4-7); M3's trial rules and M4's
denominators (BQ4-1…4, BQ4-10); and the metric amendments awaiting **E-9**.
