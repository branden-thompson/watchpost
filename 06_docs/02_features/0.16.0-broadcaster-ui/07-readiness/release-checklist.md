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
- [ ] **Internal-name scrub — run by the HUM LEAD in their own shell** at SHIP, as in 0.15.0:
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

- [x] **The release job's cap is derived, not guessed.** `release.yml` allows 60 minutes: the
  Makefile grants `mutant-check` 40, and the rest of the job (verify's other gates, `release-matrix`,
  `install-test`, publish) measured ~16 on the 0.15.0 run (16.1 total; the trend across four
  releases 13.0 → 13.6 → 13.7 → 16.1; this branch's last ubuntu verify 21.5). A 20-minute cap would
  have produced a tag with no release (VALIDATE, hygiene reviewer). **Record the measured job time
  here after the release run.**
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
- [ ] **CI round 2 (`d3d7629`): green on every leg.**
- [ ] Squash-merged; CHANGELOG date re-checked against the merge day before tagging; the merged tree
  verified byte-identical to the feature tip.
- [ ] `v0.16.0` annotated on the merged commit, pushed; the release workflow re-runs `make verify`
  against the tag before publishing.
- [ ] **The PUBLISHED artifact verified, not just the build**: download one binary from the release,
  checksum against `checksums.txt`, run `--version`, count build-path strings (must be 0).
- [ ] Local `main` fast-forwarded to the feature tip; `main-publish` to the merged commit.
- [ ] Deleted `origin/release/v0.16.0`, `origin/feature/0.16.0-broadcaster-ui` and both local branches.
  **`origin` then carries `main` alone.**
- [ ] Issues reconciled against what the release MEANT to do with them (#10 closed; any other touched
  issue's state checked by hand).

## After

- [x] **`p3-uat.md` case 10** — signed off 2026-09-18 ("30 min works fine on my testing").
- [ ] **VALIDATE on the Linux box against the installed release**, as 0.15.0 asked and did not get.
- [ ] **DEBRIEF written** — `08-reports/debrief.md`.
- [ ] **Carried items recorded in `project-watchpost-follow-ups`**: F-156, F-157, F-158, F-159,
  F-160…F-162, F-137, and the A2DH items (the dispatch brief, the remediation loop, INST-5 beside every
  number).
