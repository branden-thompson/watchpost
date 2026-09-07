# Severe Weather / Disaster Events Modals — DEBRIEF (After Action Report)

**Feature:** `severe-alerts-modals` → **shipped 0.13.0** (2026-08-29, `v0.13.0`, PR #2 squash-merged, release
green first time). **SEV-0 · HUMAN LEAD.**
**Flow:** DISCOVER → PLAN → BUILD (P1–P4 + a long UAT arc) → REVIEW → VALIDATE → SHIP → REFLECT.
**Post-ship (same day):** PR #3 (README: install first), PR #4 (badge row + the publisher-test timing fix).

## 1. The goal, and what was delivered

**Locked problem (DISCOVER, ratified 2026-08-28):** a user who has just been told a severe weather or disaster
event is active cannot see which events are active, or read the full details of any one of them, without
leaving the app or recalling and searching a location by hand.

**Delivered against it:**

| Metric | Target | At ship |
|---|---|---|
| M1 `K2D` keystrokes to detail | ≤ 4, no typing (was ≈ 5 + typing; ∞ for a quake or storm) | **4** — `w` `→` `↓` `enter`; journey 22/22 |
| M2 `COV` render-list coverage | 100 % | **100 %** — quake 11/11 · storm 13/13 · warning 11/11 |
| M3 `NAR` narration re-point | 2/2 | **2/2** |
| M4 `R6·PERF` radio untouched, frame budget | smokes green both platforms; allocations unchanged | macOS smoke green; allocations within pins; soak flat; **Linux half owed** (accepted risk) |

Requirements: FR-1…FR-14 met; **FR-15 waived at UAT** (the severity-glyph column read the same in every row —
severity moved to the tab and the tint, with the class in text under `--ascii`). Beyond the brief, the UAT arc
delivered what the plan did not foresee: the `[space]` **event read over the radio** with a takeover that pauses
and resumes it (the narrator arbiter, the engine's held/aside/audition lines), **product codes** and
**pronunciation rule tables**, the **dashboard facelift** (boxed severe strip, painted header and column bands,
centred WX STN/ZIP, the grouped Setup), **Watchpost Light** and the **AA contrast pass over every theme**, and
**entry-by-entry feed decoding** so one malformed record no longer empties a category. 28 CHANGELOG bullets;
70 commits; 216 files, +21 556 / −1 381.

**Scope changes and why:** the plan's severity glyph column (waived, above); the spread `E V E N T` header under
`--ascii` (ruled plain `EVENT` — the mock's letter-spacing was a typographic choice, not information); the
"circle" multi-alert visualisation (deferred: needs a mock); every follow-up carried from 0.11.0 and 0.12.0
was closed here on the HUM LEAD's instruction ("some have been carried for two versions now").

## 2. What went well

- **DATA FIRST, again, paid for the UAT arc.** `domains/severe` (index, categories, detection, codes) and
  `domains/globalfeed` (entry-by-entry decoders, names, the frozen render list) were built and tested in P1–P2;
  the window, the read, the facelift and the themes were presentation and orchestration on top of a spine that
  did not move. The longest UAT stretch — seven reshapes of the strip, the header and the column bands — touched
  `modes/tty` and `platform/render` only.
- **The red-team cadence scaled with the change.** Five rounds (DISCOVER 85 findings/13 Critical · PLAN 100 ·
  BUILD 57 · BUILD-exit 47 · REVIEW 60) — 349 findings, every one dispositioned with a reason, every reproduced
  defect pinned by a test. The BUILD-exit round earned its cost with one High that UAT had *approved* against a
  fake (A-01, §3.1).
- **The HUM LEAD's eyes were the sharpest reviewer.** The missing tail after a fire report, the Lookup modal
  showing the wrong row, `[space]` Read muted on every build, the uppercase esc-fusion — four defects no
  reviewer or test had, each found by running the binary. Two of them (V-1, V-2) are now covered by the
  fresh-HOME PTY journey that runs the real binary through the real input layer.
- **One render-time fact beat thirteen hand-tuned palettes.** The AA lift (`withAA` over the `aaPairs` register)
  fixed Watchpost Light *and* visibly improved every imported Quattro theme — the HUM LEAD noticed the other
  themes "feel less dull" before being told why. The root cause of Light's dark grounds was one line
  (`frameText` composing `38;5;` in front of a truecolour token); finding it instead of patching the symptom
  is what made the fix general.
- **The ship itself was clean.** Rollback tested before the tag; the squash-onto-`main-publish` shape kept the
  private history private; CI green on both runners on the push run and the PR run; the release workflow green
  first time; the installed artifact verified by the public one-liner. The only snag was administrative (§3.4).

## 3. What to carry forward (lessons)

1. **A fix that passes against the fake is not a fix** (round 4 A-01). The takeover-pauses-a-read behaviour
   was UAT-approved on the voice fake; the real engine closed the paused line the moment the tone took the one
   preview slot. Contract tests now run on the recording output as well as the fake, and the R6 audio smoke is
   a blocking VALIDATE gate whenever the radio path changes.
2. **Decide at the press, not at wiring time** (V-2). A guard evaluated when the dashboard was configured
   (`deck != nil`) was permanently false because the deck attaches later. Hooks that depend on late-bound state
   must read that state when they fire. The same shape as 0.12.0's "the ExpandStates call was missing": a
   comment can say the wiring exists; only a test on the real path proves it.
3. **The real input layer is part of the product** (V-1, D-2). A pty delivers `esc` fused with the next key,
   and a shifted letter arrives as a modifier with no text. Two P1s in a row came from that seam; the journey
   script now drives every key through it on every VALIDATE.
4. **Don't wait for the first `done` when there can be two** (PR #4). A counter test read after one completion
   signal while a second publish was in flight — green on macOS for a month, red on the loaded ubuntu runner.
   Wait for the *condition* (the sum), bounded, never for the first event. Candidate sweep: any test that
   receives once from a channel and then reads a counter.
5. **Search and slice the same string** (round 4 A-05): `ToUpper` can change byte lengths; an index into one
   form used on the other reached the publish goroutine as a panic.
6. **Bare SGR numbers are ambiguous** (B-01, D-9, and the Light root cause): `97` is bright white, `250` is a
   256-colour index, and a truecolour token is neither. Tokens carry full parameters; a box with no tone of its
   own carries no SGR at all; foregrounds are composed by one function (`render.FgSGR`).
7. **Pins should say what the behaviour is, not what the output was.** Fixed label offsets broke on centring;
   "centred over its column" survived the facelift. Goldens carry width invariants; the header ladder pins the
   order, not the widths; the 80×24 pin says "never exceeds the terminal".
8. **Budgets catch design regressions before eyes do.** Eager chip forms (+170 allocs), the header's compositor
   cost (+740), the thin-bands double render (−350 when fixed) — all found by the allocation pins.
9. **The ledger is a document.** Four P10 rows' reasons predated the code they absorbed; rows are presented with
   their reasons for ratification, re-run at every exit. Two gate quirks to hand upstream: the P10-05 density
   row anchors at a different file — or vanishes — between identical clean-tree runs (0 live either way), and
   on the trunk with a docs-only change the ledger's rows read "unmatched" because the diff scope collapsed to
   the README.
10. **Check the token before the branch.** `gh pr create` 403'd twice at the ship because the personal
    fine-grained PAT lacked *Pull requests: write* — never needed while releases were tag-only. The
    pre-ship checklist now includes the token's permissions, not just the login.
11. **Lint is part of the PR contract, so run it before claiming it.** The template's "golangci-lint and
    staticcheck clean" box surfaced five real nits on new code at SHIP (an unused helper, an unchecked
    `Unsetenv`, three De Morgan hints) — small, but the box is a claim. Note the linters rewrite `go.sum`;
    restore it before `make verify`.
12. **Mocks are exact, colours are the HUM LEAD's pass** — and an AA floor is better as a render-time fact than
    as a value per theme; the intention of a hue survives a lift, not a hand-edit.

## 4. Process review

- **SEV-0 was the right level.** The feature touched the radio (sacred), the input layer, every theme and the
  feed parsers; the two P1s found at VALIDATE and the round-4 High would each have shipped under a lighter gate.
- **What cost time:** the UAT arc's reshapes (worth it — the HUM LEAD's eye is the product's), the P10 gate's
  loop at the ship (a stale run record after each `cp` of the report; solved by committing first, then running),
  and the token permission. **What saved time:** the journey script (22 steps, fresh HOME, real pty) — every
  post-fix re-run was one command; `waitUntil`/gated sequences instead of sleeps (no timing flakes in the
  feature's own tests); the release shape (one squash commit, rollback pre-tested).
- **Gates:** every exit at `a2dh validate` 18/18; `make verify` green on every commit; p10 0 live / 0 unmatched;
  budgets pinned in one owner. The linters were the one gate the template named that the Makefile did not run —
  candidate for `make verify` (with the `go.sum` note).

## 5. Follow-ups (open)

- **Linux R6 half** — relay + audio smoke, `make pty-severe`, the race suite on Arch (needs an audio device);
  `07-readiness/linux-validation-protocol.md`. Verification only.
- **Live takeover pause/resume** of a `[space]` read — HUM LEAD's manual check on a busier day; pinned by test.
- **Frame time +38 % at 133×44** vs v0.12.0 (facelifts) — accepted; a candidate lever is caching the painted
  bands' composed strings between identical frames.
- **Multi-alert circle viz** — needs a HUM LEAD mock.
- **Counter-test sweep** — tests that receive one `done` then read a count (lesson 4).
- **`make lint`** target (golangci-lint + staticcheck on Watchpost's packages, `go.sum` restored) folded into
  `make verify`; the two upstream nits in the vendored kit go to the go-studs patch stack.
- **P10 gate quirks** (lesson 9) — report upstream with the two reproductions.

## 6. By the numbers

- 70 commits on the feature branch; 216 files, +21 556 / −1 381; 28 CHANGELOG bullets.
- Red-team: 5 rounds, 349 findings dispositioned (DISCOVER 85 · PLAN 100 · BUILD 57 · BUILD-exit 47 · REVIEW 60);
  0 open. Debug log: 12 investigations (D-1…D-12), each with method and root cause.
- Coverage: globalfeed 92.5 % · severe 92.0 % · tty 91.6 % · render 85.9 % · app 53.0 %.
- Budgets at ship: frame 133×44 hit 655 / miss 7 102; 80×24 964 / 3 238; severe window hit 2 031 / miss 7 637 —
  all within pins. Soak: 1 hour, no trend.
- Ship: PR #2, release run 33264986559 green first time (1m22s); artifact verified 0.13.0; rollback to
  0.12.0 tested; 45 downloads on the badge by the evening of release day.

## Source documents

`08-reports/{project-brief,discover-report,por-report,build-report,red-team-*,review-report,validate-report,ship-report}.md` ·
`07-readiness/{gates,release-checklist,linux-validation-protocol}.md` · `05-debugging/debug-log.md` ·
`06-key_learnings/key-learnings.md` · `CHANGELOG.md` §0.13.0.
