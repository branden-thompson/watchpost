# Release checklist — 0.16.0 Broadcaster UI

The FULL GIT shape (`project-watchpost-git-protocol`), unchanged from 0.15.0, **with one ruling added
2026-09-17: the release is a squash-merge release branch — ONE commit onto `main` — and published
history is never rewritten.** One `feature/0.16.0-broadcaster-ui` branch for the release; local `main`
is the dev trunk (never pushed); `main-publish` mirrors `origin/main`; the release is one `commit-tree`
squash of the feature tree with parent `main-publish`, pushed as `release/v0.16.0`, PR'd to `main`
with the canonical template, squash-merged, tagged on the merged commit; the feature branch is deleted
on origin after. Nothing in this branch's history — including the 15 MB blob at `6b1b621` — reaches
the remote, because only the squashed tree does.

**Branch name:** `release/v0.16.0`, with the `v`, matching `release/v0.14.0` and `release/v0.15.0`.

## Before the release commit

- [x] **`make verify` — ALL GATES GREEN** on `73bce2b` (code) and `35de36a` (records), read from the
  logs (`dist/verify-review-*.log`). `p10` is a phase-exit gate under `make quality` by ruling
  (REVIEW R1), green at `f7fe5fa`; `release-matrix` and `install-test` are CI-only and were **run
  locally at VALIDATE**: `lint-injector` OK on 6 artifacts, `release-matrix` OK, the installer end to
  end against a local server OK, the tamper control fired. A stale local `dist/watchpost-diag` (an
  injector build) made the first local run refuse — correctly; it is git-ignored and was removed.
- [x] **The mutation sweep on the code commit**: 380 — 376 CAUGHT, 4 SURVIVED by design, 0 NO
  EVIDENCE (`mutant-verdicts.log`, opening with the tree hash `73bce2b`).
- [x] **`origin/main` is contained — checked as a TREE, not a commit.** `origin/main` is `fd761ab`,
  the 0.15.0 squash; it is NOT an ancestor of this branch (the squash commit never is — the branch
  descends from the feature tip `d7f2605`, whose tree the squash carries). The check that matters:
  `git diff --stat fd761ab d7f2605` → **empty**, and `d7f2605` IS an ancestor of HEAD
  (**yes**). Nothing landed on `origin/main` after `fd761ab`, so nothing can be reverted by
  the squash. **Re-run both at tag time.**
- [x] **`gh api user` = `branden-thompson`** under `GH_CONFIG_DIR=~/.config/gh-personal`, checked
  2026-09-18 before any outward action. Re-check at the push.
- [ ] **Internal-name scrub — run by the HUM LEAD in their own shell**, as in 0.15.0 (not confirmed
  to this record before the merge; run it against `origin/main` and note the result here):
  `git grep -n -i -e "$A2DH_HOME_NAME" -e "$EMPLOYER_NAME" -- ':!third_party'` → must be empty.
- [x] **`git worktree list` shows the main tree only** (2026-09-18).
- [x] **README captures — five of the Broadcaster console** from the HUM LEAD's UAT build
  (`35de36a`), placed 2026-09-18. The banner's demo transmitter position appears in them as pixels;
  **ruled fine by the HUM LEAD (2026-09-18)**, with a recapture on the 0.16.0-stamped build if wanted,
  as 0.15.0 did. The Observer captures are unchanged by this release's Observer-side diffs.
- [x] **`CHANGELOG.md` — the 0.16.0 section written and dated 2026-09-18** (`e8f98f9`); the date is
  re-checked at tag time (0.14.0 needed a second PR for exactly this).
- [x] **`THIRD_PARTY_LICENSES.md` current** — `go.mod` and the licence file were last changed in the
  same commit (`1abf27d`, 2026-09-07); no module changed in 0.16.0.
- [x] **Exposure statement re-derived at the tip** (`exposure-statement.md`, 2026-09-18): identity
  137 / 341, location 214 / 750, credential 2 distinct fixtures, path 4 / 5 placeholders; `-trimpath`
  held by `TestEveryBuildTargetTrimsThePath`.
- [x] **`07-readiness/pr-body.md` written and `a2dh pr-template check` PASSES** (2026-09-18), with
  the checker controlled before its tick was trusted: the Caveats section removed →
  `R3-section-present` CAUGHT; the metrics table reduced to the placeholder row → `R3-metrics-row`
  CAUGHT.
- [ ] **Issue #10** (the brief is its body): the close directive goes in the RELEASE COMMIT MESSAGE,
  unadorned — `Closes #10.` — and nowhere else; every touched issue's final state is checked after
  the merge (0.15.0's #12 was closed by a sentence saying it should stay open).
- [ ] **No attribution and no internal names** in the commit or the body, per the standing rule and
  `lint-watermark`.

- [x] **The release job's cap is derived, not guessed.** `release.yml` allows 90 minutes: twice the
  Linux CI job that runs the same gates, measured at **38 min** on this release branch (CI round 2,
  2026-09-18); earlier derivations were 60 (the Makefile's 40-minute mutant bound + ~16) and, before
  VALIDATE, a 20-minute cap that would have produced a tag with no release. The trend across releases:
  13.0 → 13.6 → 13.7 → 16.1 → 38. **Record the measured release-job time here after the run.**
- [x] **F-164 ruled (a)** — a relay falling through to synth releases the operator's cut-over, as the
  code is today (HUM LEAD 2026-09-18; D-159).

## The release

- [x] `release/v0.16.0` = `57875f1`, `git commit-tree 9747ee8^{tree} -p main-publish` — verified
  before pushing: same tree hash as the feature tip, `fd761ab` (= `origin/main`) its only parent,
  empty diff against the tip, `Closes #10.` alone in the message, zero attribution strings,
  `gh api user` = `branden-thompson` (2026-09-18).
- [x] Pushed; **PR #20** opened against `main` with the checked body.
- [~] **CI round 1 (2026-09-18, `57875f1`): `policy` and both macOS legs GREEN; the pull-request
  run's Linux leg RED at `make race`** — `TestEveryLineOfEveryWindowIsReachableAtTheFloor/status`:
  one line "cannot be reached", `✔ nws NWS 2ND OK 619h 09m`. **A finding, not runner noise, and not
  Linux's:** the Status window's FETCHED column is an age read off the real clock at minute
  resolution, and a run that straddles a minute boundary between building the lines and rendering
  them holds a text no frame ever draws. Forty local runs under `-race` never hit the window; a
  clock that moves a minute per read turns the guard red every time. Fixed on the feature branch
  (`ab5d62e`: the window fixtures pin the clock) and added to the release branch as a second commit
  (`d3d7629`, same tree as the feature tip), as 0.15.0's rounds were — the PR squash-merges either
  way. Budget for further rounds: **expect Linux-only failures and treat each as a finding.**
- [x] **CI round 2 (`d3d7629`): GREEN on every leg** — `policy` 5–8 s, macOS verify 9 m 09 s and
  12 m 36 s, **Linux verify 37 m 43 s and 38 m 22 s**. That last figure moved a decision: the release
  cap had been derived from a 21.5-minute Linux verify; the job is 38 now on this tree, and 60 left
  22 minutes of headroom, under the two-times rule the VALIDATE reviewer named. **Raised to 90 (twice
  the measurement) as round 3**, one YAML line, rather than ship on a margin that had just halved.
- [x] **CI round 3 (`0d45077`, the cap): GREEN on every leg** — `policy` 6–7 s, macOS verify 10 m 47 s
  and 11 m 40 s, Linux verify 26 m 38 s and 36 m 10 s. Observed Linux range on this branch across
  two green rounds: **26–38 min**, against a 90-minute release cap. Merge state CLEAN; `origin/main`
  unchanged at `fd761ab` throughout; the CHANGELOG's date (2026-09-18) matches the merge day.
- [x] **Squash-merged as `237e2a5`** (2026-09-18, HUM LEAD: "go"), parent `fd761ab`, message the
  release commit's with `Closes #10.` alone; the CHANGELOG's date matched the merge day; the merged
  tree verified equal to the release tip's and to the feature tree at `81a7b06`.
- [x] **`v0.16.0` annotated on `237e2a5` and pushed.** Release run `35398502049` GREEN — verify,
  release matrix, installer smoke test, publish — **37 min 42 s** (21:47:23 → 22:25:05 UTC) under the
  90-minute cap; 8 assets, both Linux binaries among them. The trend across releases is now
  13.0 → 13.6 → 13.7 → 16.1 → **37.7**; the cap derivation (twice the measured job) holds with
  52 minutes of headroom.
- [x] **The PUBLISHED artifact verified, not just the build** — `watchpost-darwin-arm64` downloaded
  from the release: checksum matches `checksums.txt` (`05642efe…9d73`, OK), it reports
  `watchpost version 0.16.0`, and it carries **0** `/Users/`, `/home/` or scratch-path strings against
  110,902 strings in all, so the zero is not a vacuous scan.
- [x] Local `main` at the feature tip (the dev trunk); `main-publish` at `237e2a5`.
- [x] Deleted `origin/release/v0.16.0`, `origin/feature/0.16.0-broadcaster-ui` and both local branches
  (2026-09-18). **`origin` carries `main` alone.**
- [x] **#10 CLOSED by the merge commit** at 21:47:17 UTC, checked by hand; no other issue was
  touched by this release.

## After

- [x] **`p3-uat.md` case 10** — signed off 2026-09-18 ("30 min works fine on my testing").
- [ ] **VALIDATE on the Linux box against the installed release**, as 0.15.0 asked and did not get.
- [ ] **DEBRIEF written** — `08-reports/debrief.md`.
- [ ] **Carried items recorded in `project-watchpost-follow-ups`**: F-156, F-157, F-158, F-159,
  F-160…F-162, F-137, and the A2DH items (the dispatch brief, the remediation loop, INST-5 beside every
  number).
