# Red-team round 4 — BUILD exit, after UAT (0.13.0 severe-alerts-modals)

**Date:** 2026-08-28 · **SEV-0 · HUMAN LEAD** · **Scope:** the 25 UAT commits `0187bd7..e3b739f` on
`feature/severe-alerts-modals` (117 files, +4 227 / −904) plus the radio diagnostic `8625069` · **Dispatch:** the
round-3 shape — three sectioned reviewers, read-only, demonstrating with throwaway probes: A: narrator / radio
engine / scripts / app · B: TTY + render facelifts (+ A11y persona, budgets) · C: plan conformance + documentation
truth + hygiene. Every finding was **verified before acceptance** — the demonstrations re-run or the cited site read —
and the demonstrations that held were lifted into the repo as tests.

Disposition codes: **Fixed** · **Accepted-as-is** (with reason) · **Declined** (with reason) · **Deferred** (to a named
owner/phase) · **HUM LEAD ruling** (presented at this gate).

Totals: **47 findings** — 37 Fixed · 2 Accepted-as-is · 2 Declined · 2 Deferred (A-08, A-16 c/e) · 3 HUM LEAD rulings (a mixed row — A-13, A-16, B-12 — counts under its primary disposition; A-08 and A-16 c/e were closed at REVIEW). Reviewer
verdicts: A **HOLD** · B **SHIP-WITH-CONDITIONS** · C **SHIP-WITH-CONDITIONS** — every HOLD/condition item is closed
below.

## Section A — narrator, radio engine, scripts, app (reviewer verdict: HOLD → fix-then-ship)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R4-A-01 | High | The engine had ONE preview slot: a read PAUSED for a takeover was closed by its own watcher the moment the takeover's tone took the slot (`paused := previewPaused && preview == p` → false; a paused player is `!IsPlaying`). The round's headline fix ("a takeover pauses a read, then resumes it") held only against the fake voice; on the real engine the read's audio was gone and its hold ran on in silence. | probe on the recording output: Preview → Pause → PreviewAside → the read's `stop` closed | **Fixed**: held lines are tracked by identity (`held map[Player]bool` + order, under `e.mu`); a later clip never displaces one; `ResumePreview` takes the last held; `DropHeld` closes them unplayed. `TestHeldLineSurvivesTheTakeoversClips` |
| R4-A-02 | Medium | A read whose context ended while SUSPENDED popped itself from the stack in `release`, overwrote `onAir` (the takeover) with the dead job and called `resume` mid-takeover; the takeover's release then left the ghost on air — every later read waited forever. Latent (the read used `context.Background()`), but the first cancellable lower-class run would wedge the voice. | probe: cancel while suspended → the next read never admitted (2 s) | **Fixed**: `release` unlinks a suspended job and `discard`s its held line (a new `narrationVoice` method → `Engine.DropHeld`); a cancelled *waiting* job also settles whoever it outranked (`settle` is the one owner of "who has the air when it frees"). `TestReadCancelledWhileSuspendedDoesNotWedgeTheNarrator` |
| R4-A-03 | Medium | `e.previewPaused = false` written outside `e.mu` — a data race against `PausePreview` and the watcher. | `-race` probe: two reports | **Fixed**: the flag is gone; the held set lives under the lock |
| R4-A-04 | Medium | `awaitAir` released the lock before `play`: a takeover admitted in the gap paused nothing and the read's line started UNDER it. | probe: 8 collisions in 2 s | **Fixed**: the line starts under the arbiter's lock (`awaitAir(then)`). `TestALineNeverStartsUnderATakeover` |
| R4-A-05 | Medium | `sourceKind` searched `ToUpper(d)` and sliced `d` — a rune that lengthens on upper-casing (ȿ → Ȿ) pushes the index off the end: a panic from feed prose on the publish goroutine. | probe: `panic: slice bounds out of range [31:23]` | **Fixed**: search and slice the same string. `TestDetectionSurvivesCaseLengtheningRunes` |
| R4-A-06 | Medium | NFR-6: provider prose reached the marquee unescaped (`Normalize` keeps ESC/OSC; `Composer.say` returned raw; only `stationText` was Plain'd on the panel). | probe: `"H\x1b[31mX. D."` composed | **Fixed** at one owner each side: `script.Library.Say` (Plain'd; used by the composer and the ticker) and `applyRadioStatus` (`PlainLine` on Station/Short/Detail) |
| R4-A-07 | Low | A preview's volume was set once; `[+]`/`[-]` during a minutes-long read did nothing. | read | **Fixed**: the watcher re-asserts the knob every tick (pinned in the A-01 test) |
| R4-A-08 | Low | A read cannot be stopped (`context.Background()`; esc/Stop/quit leave the goroutine and player). | read | **Deferred** to REVIEW (follow-ups): derive the read's context from the app's and let the window cancel it — safe now that A-02 is closed |
| R4-A-09 | Low | The "{{" invariant was asserted on RENDERED output — data is never re-parsed, so it only silenced a phrase carrying "{{". | probe: `""` + invariant violated | **Fixed**: the check is gone (the built-in tree's parse is pinned by the package test) |
| R4-A-10 | Low (InfoSec) | The override tree followed symlinks into any readable file. | probe: symlink → read | **Fixed**: `Lstat`, regular files only |
| R4-A-11 | Low (perf) | Every pronunciation table re-read the embed FS per call (µs per cycle). | bench | **Declined**: a memo is a package global, which P10-06 forbids (the attempt tripped the gate); the cost is microseconds per cycle, off the frame path |
| R4-A-12 | Low | Four stale comments ("cut by", "~45 s", "interrupt", "newNarrator", "warn hook"). | read | **Fixed** |
| R4-A-13 | Low | "Missing script = silence" written twice, disagreeing on Plain; `vizFor` claims one ownership while the tone and the voice preview decide alone. | read | **Fixed** for the scripts (`Library.Say`); **Accepted-as-is** for the bars: the tone is a takeover's by definition and the voice preview plays tapped — both agree with `vizFor`'s rule; the comment now says every narration except a takeover |
| R4-A-14 | Nit | `ProductCode` doc mixed VTEC and AWIPS PIL terms; the initials fallback compared bytes. | read | **Fixed**: reworded; `unicode.IsUpper` on the first rune |
| R4-A-15 | Nit | `Interrupt` was dead code with a test. | probe | **Fixed**: deleted with its test |
| R4-A-16 | Low (tests) | (a) pause/aside/resume untested; (b) narrator vs real engine untested; (c) two fixed sleeps; (d) the aside-tap test passed vacuously; (e) unlock/relock around a defer; (f) no non-ASCII detection input. | read | (a)(b)(d)(f) **Fixed** by the lifted tests; (c)(e) **Deferred** to REVIEW (test hygiene, not behaviour) |
| R4-A-17 | Nit | A map literal built per rune in `spokenWindow`. | read | **Fixed**: `durationUnits()` (a function, P10-06) |

## Section B — TTY + render (reviewer verdict: SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R4-B-01 | Medium | The header box was painted `Block(…, "250", "49")` — a bare 256 index is not an SGR, so every header line opened `\x1b[250;49m` and every inner reset re-armed as `\x1b[0;250;49m`, dropping the base grey after each chip on 256-index themes; the colour golden pinned it. | `grep -c '\[250;49m'` = 3 | **Fixed**: the header carries no tone of its own — `Block` with no tone pads without SGR and the frame's base grey paints it (a normaliser was tried and withdrawn: a bare `"97"` is basic bright white, not index 97 — it turned the severe window purple and cost the compositor +740 allocs). Colour golden re-recorded; `Block`'s doc names the contract |
| R4-B-02 | Medium | `marqueeTrack` padded a `w-1` line to `w+1` — the player's right rule shifted a cell (hit at 150×44). | probe | **Fixed**: air only when both cells fit |
| R4-B-03 | Medium | `[space] Read` re-broke the 80-col `--ascii` chip row (79 cells in 69); the golden was recorded wrapped. | golden read | **Fixed**: a fourth form with the arrows unlabelled; `TestSevereChipsHoldOneLineAtTheASCIIFloor`; goldens re-recorded |
| R4-B-04 | Medium (A11y) | Colour off / `--ascii`: a warning and an advisory rendered byte-identical — the class was tint only, and the `[severe]` R-12a pin had been deleted. | probe | **Fixed**: the severity token in text when colour is off or under `--ascii`; the R-12a pins restored (anatomy + a two-class check) |
| R4-B-05 | Medium (A11y) | The module title `1;AlertModalWarnFG` on `AlertModalWarnBG` is below AA in 10/12 themes (default 4.20:1 … Nord 2.57); pre-existing inside `[A]`, now permanent on the dashboard. | probe over the themes | **HUM LEAD ruling** (below) — the title's dress was your ask; the numbers are attached |
| R4-B-06 | Medium (perf) | The thin-bands path resolved the layout twice, building the player rows 4× and the control row 2× per frame; the pin comment credited the header. | AllocsPerRun | **Fixed**: the rows are built once per frame and handed to both resolutions; 80×24 memo-hit 1 312 → 962, pinned honestly |
| R4-B-07 | Low-Med | At the 80×24 floor the frame ran to 25 lines when the control row wrapped (the 3-row window floor did not yield). | probe | **Fixed**: the inset gives first, then the window shrinks to what fits — the frame never exceeds the terminal. `TestTheFloorFrameAt80x24` |
| R4-B-08 | Low-Med | `splitEscFusion` mapped every 0x01–0x1a to ctrl+letter: ESC CR read as ctrl+m — the Enter after an esc was lost. | probe | **Fixed**: Enter/Tab/Escape stay themselves; pinned in `TestLoneEscThenKeyIsNotLost` |
| R4-B-09 | Low | The narrow head's third rung shortened the long station after the second chose the short form; the failed state lost its reason. | probe | **Fixed**: `narrowHead` shortens whichever form was chosen; a failure stays a failure. Pinned at 84 cols (`♪ EVENT · SPS…`) and failed |
| R4-B-10 | Low | `BoxTitled` never clamped a row; below ~62 cols the head overflowed the box. | sweep | **Fixed**: rows are truncated to the inner width (one defensive owner); a head with no room is the mark alone |
| R4-B-11 | Low | Dead code kept alive by its tests (`AlertBanner`, `Module`, `ModuleInnerWidth`, `ModuleHeight`, `AlertBlockTone`, `CentreBetween`, four `Alert*` tokens) and three stale comments. | grep | **Fixed**: deleted (tokens from every theme and the Quattro mapper); comments rewritten |
| R4-B-12 | Low | Six hand-rolled "first form that fits" ladders; the tint pick written twice. | read | **Fixed** where the forms are cheap strings (`render.FirstFit`; `alertModalBG` beside `modalAlertTone`); **Accepted-as-is** for the chip/head ladders, which build each form only when the wider one fails — the frame budgets caught the eager version (+170 allocs) |
| R4-B-13 | Nit | The "deadline outranks the issue stamp" rung was unreachable. | probe | **Fixed**: dropped |
| R4-B-14 | Nit | The row stripped seven labels before the 5-cell `API: ` label. | probe | **Fixed**: the summary loses its label before the chips lose theirs; pinned in order (`TestHeaderRowLaddersInOrder`) |
| R4-B-15 | Nit | Stamps carry no zone label. | read | **Declined**: the ratified mock has none; the stamps are in the location's own clock, as the record's are |
| R4-B-16 | Low (tests) | Four pins that restate the output. | read | **Fixed**: the narrow loop compares with the frame width; `SPS` asserted at 84; the chip row's line count; the 80×24 line count |
| R4-B-17 | Nit | `SevereHeaderTone` on a non-truecolor hue fell through `mixBG` to a white slab. | probe | **Fixed**: the hue is its own band |

## Section C — plan conformance + documentation truth (reviewer verdict: SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R4-C-01 | Medium | CHANGELOG said a takeover "cuts" a read; eight shipped changes had no line. | read | **Fixed**: pauses/resumes; bullets for the facelifts, DETECTION, Declared, the tints, the pronunciation rules, the two-column windows, the diagnostic |
| R4-C-02 | Medium | The gate ledger was not re-run after 25 commits (budgets, finding count, the radio diff, golden count). | re-run | **Fixed**: rewritten below from this exit's runs; `bench_test.go` is the single owner of the budget numbers (the plan §0 figure is superseded) |
| R4-C-03 | Medium | NFR-7 ("radio untouched") still asserted the opposite of the amendments. | read | **Fixed**: superseded in the objectives; R6 is a blocking VALIDATE item on both platforms |
| R4-C-04 | Low | The flow map and three comments carried the "cuts/interrupt" model. | read | **Fixed** |
| R4-C-05 | Low | "the only file that imports go-studs" — seven do. | grep | **Fixed**: the only *package* |
| R4-C-06 | Low | Four ledger rows' reasons predate the code they now absorb; the ratify list omitted seven rows. | script | **HUM LEAD ruling**: the full list of rows to ratify is below; the reasons live in the local ledger and are refreshed at ratification |
| R4-C-07 | Low | The P10 gate cannot run from a clone (`.a2dh.yml` and the ledger are gitignored). | clone | **HUM LEAD ruling** (below) |
| R4-C-08 | Low | Objectives: a literal `\n`; the Help width formula; the stamp ladder; superseded window numbers; an unverified "~96 cols". | read | **Fixed** (the ladder now has no expires-return step — B-13; the head pinned at 84) |
| R4-C-09 | Low | The framework's name in two tracked files outside the allowed set. | grep | **Fixed** |
| R4-C-10 | Low | The feed-path Declared rule had no fixture with sent ≠ onset. | read | **Fixed**: `TestNWSDeclaredIsTheSentTimeOnTheFeedPath` |
| R4-C-11 | Low | The header's inner ladder and the no-voice hold had no test. | read | **Fixed**: `TestHeaderRowLaddersInOrder`, `TestEventReadWithoutAVoiceHoldsTheOverlay` |
| R4-C-12 | Nit | Help shows `space` as "Play/Pause Radio" only. | probe | **Fixed**: "Play/Pause · Read Event" (the two-column fit kept) |
| R4-C-13 | Nit | Two misleading comments (`codes.go` example; `severePlaying`). | read | **Fixed** |

Hygiene (C, verified on the clean tree): no attribution in the 25 commit messages; no probe or scratch files tracked;
`gofmt`/`go vet`/`go mod tidy` clean; P10 reproduces from a clean clone once the two local inputs are copied in.

## Found by me during the round

- The UAT report "no tail after Vista's fire report" is **not reproduced**: the composition is right (head → 2 s →
  report → {2 s → fire} → {2 s → seismic} → 1 s → tail), the tail and the seismic link render through the real voice,
  and a real-audio run of the whole path (fake voice with `say`-like latency → engine → oto) plays every segment to
  `broadcast complete`. The diagnostic now names the segment a cycle reached (`8625069`); the three questions are
  in the UAT thread. Open until reproduced with the log.

## HUM LEAD rulings requested at this gate

1. **AA of the module's warning title (B-05).** `1;AlertModalWarnFG` on `AlertModalWarnBG` — the dress you asked for —
   is 4.20:1 in the default theme and below 4.5 in 10 of 12 themes (Nord 2.57, Kanagawa 2.64). Options: lift
   `AlertModalWarnFG` (it is also the record title inside `[A]`), or paint the event in `AlertModalText` bold and keep
   the hue on the glyph. The advisory pair passes everywhere.
2. **P10 inputs (C-07).** `.a2dh.yml` and the exemption ledger are gitignored, so `make p10` fails from a clone. Track
   them, or keep the gate local and say so in the ledger (current state).
3. **P10 ledger rows to ratify (C-06)** — every finding in `p10-build.json` (14, all exempted): the package rows
   `domains/globalfeed`, `domains/severe`, `domains/radio/script`, `domains/radio/pronounce`, `domains/radio/player`,
   `domains/radio/synth`, `domains/weather/nws`, `modes/tty`, `platform/render`, `app` (P10-05 density), the two
   `app/pipelines.go` vacuous-invariant rows, and `app/narrate.go` `admit`/`awaitAir` (P10-02). Four reasons predate
   the code they absorb (severe, player, synth, nws) — refresh at ratification.
4. Still open from round 3: `--ascii` header spread vs plain; the `>` pointer/mark collision.

## Round verdict

**SHIP-WITH-CONDITIONS → conditions met.** Section A's HOLD was earned: the pause/resume that UAT approved did not
survive the real engine (A-01), the narrator had two protocol holes (A-02, A-04) and a race (A-03), and feed prose
could panic the publisher (A-05) — all reproduced, all closed with the reproductions kept as tests, the narrator now
pinned against a collision loop and a cancel-while-suspended. Section B's three golden-pinned rendering defects are
fixed and re-recorded, the 80×24 floor no longer overflows, and the layout builds its rows once. Section C's
paperwork is true again. After the round: `make verify` ALL GATES GREEN · `make p10` 0 live, 0 unmatched ·
`make pty-severe` ok · budgets within pins (80×24 962 / 3 002; severe 2 004 / 7 574) · five goldens re-recorded
deliberately (header untoned, the class token, the chip floor). What remains is the three rulings above and the
follow-ups (A-08, A-16c/e), none of which block REVIEW.
