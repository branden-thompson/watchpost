# Release checklist — 0.15.0 pre-Broadcaster UI improvements

The FULL GIT shape (`project-watchpost-git-protocol`), unchanged from 0.14.0: one
`feature/0.15.0-pre-broadcaster-ui-improvements` branch for the release; local `main` is the dev trunk
(never pushed); `main-publish` mirrors `origin/main`; the release is one `commit-tree` squash of the
feature tree with parent `main-publish`, pushed as `release/v0.15.0`, PR'd to `main` with the
canonical template, squash-merged, tagged on the merged commit; the feature branch is deleted after.

**Branch name:** `release/v0.15.0`, with the `v`, matching `release/v0.14.0`. The HUM LEAD wrote
`release/0.15.0`; the `v` is carried for consistency with the one prior release that used this shape,
and the branch is deleted after the merge either way.

## Before the release commit

- [x] **`make verify` — ALL GATES GREEN**, on the tree being shipped, after the CI fix. 14 gates,
  including `race` and the **172**-mutant corpus. `make journey` is RED **by HUM LEAD ruling**
  (F-58, 26 of 28) and is named in `06_docs/required-gates.txt` as such.
- [x] **The branch's CI was RED and is fixed — and that this was found at SHIP is the release's last
  finding.** Run `34247621543` failed `race` on `ubuntu-latest` while the `macos-latest` leg of the
  SAME run passed; every local gate runs on macOS, so "ALL GATES GREEN" was true and incomplete
  through BUILD exit, REVIEW and VALIDATE. Not a platform defect:
  `TestEventReaderDucksSpeaksRestoresAndOverlaysThePanel` asserted a second read was inert while one
  was running without holding the first one open, so it passed on scheduling rather than on the
  guard. Pinned to the idiom already in the file. **Instrument validated before trusted (INST-4):**
  green 25/25 → plant the removed `busy` guard → compiles → **RED 25/25** → revert → green 25/25.
  Kept as mutant `mK8`. The older red run (`34245681417`) was the `SA1019` lint, already fixed.
  Carried as **F-63**: no phase exit reads the branch's CI state. 0.14.0 met this from the other side
  — its PR was the first Linux run in eight days and took five rounds — and the lesson was never
  converted into a check.
- [x] **`main-publish` is current** — fetched 2026-09-09 and fast-forwarded to `e66e0ef` (0.14.2).
  It was **one behind**, which matters here and not in 0.14.0: the squash takes the feature TREE
  wholesale, so a hotfix on `origin/main` that the feature branch did not contain would be silently
  reverted by the release commit. Checked rather than assumed —
  `git merge-base --is-ancestor origin/main HEAD` → **0.14.2 IS contained**.
- [x] **`gh api user` = `branden-thompson`**, under `GH_CONFIG_DIR=~/.config/gh-personal`, checked
  2026-09-09 before any outward action.
- [x] **Internal-name scrub — ZERO matches.** Run by the HUM LEAD in their own shell, 2026-09-09, so
  the two names stay out of the tree and out of this file:
  `git grep -n -i -e "$A2DH_HOME_NAME" -e "$EMPLOYER_NAME" -- ':!third_party'` → no output. The guard
  refuses to run with either name unset; an empty `-e` would match every line.
- [x] **`git worktree list` shows the main tree only.**
- [x] **README captures are current — checked, not assumed.** The Settings diff routes marks through
  the glyph set for `--ascii` correctness, so the DEFAULT rendering is character-for-character
  unchanged (`…`, `—`, `•`, `⚠`, `✔` are the same glyphs, now sourced from the set); the fire line
  removed on the HUM LEAD's UAT ruling was a spoken phrasebook entry
  (`domains/radio/script/scripts/fire-report/outside.txt`, deleted with no orphaned key), never on
  screen. No capture in `docs/img/` is stale, so 0.14.0's capture round has no counterpart here.
- [x] **`CHANGELOG.md` dated 2026-09-09**, on the day it ships and not before. **If the merge lands on
  a later day, the date is corrected before the tag** — 0.14.0 needed a second PR (#6) for exactly
  this, and the fix is to check the date at tag time rather than to date it optimistically.
- [x] **`THIRD_PARTY_LICENSES.md` current** — `go.mod` and `THIRD_PARTY_LICENSES.md` were last changed
  in the same commit (`1abf27d`), so the file cannot be behind the module list.
- [x] **Exposure statement written** (`07-readiness/exposure-statement.md`), across tree, history,
  tags and built artifacts, with its own controls. **It is a point-in-time measurement and says so**:
  re-derive with `python3 scripts/quality/exposure-scan.py` rather than citing its table.
  `internal-url` and `credential` are **clean** — the two credential-shaped values in the entire
  history are hex-ramp test fixtures. The largest fixable finding, `-trimpath` set nowhere, is fixed
  in this release: 473 occurrences of the build path per binary → 0.
- [x] **`a2dh pr-template check` PASSES** on `07-readiness/pr-body.md`. **The checker was controlled
  before its tick was trusted:** a required section removed → `R3-section-present` **CAUGHT**; the
  metrics table reduced to the template's placeholder row → `R3-metrics-row` **CAUGHT**. The FIRST
  attempt at that second control SURVIVED and the plant was wrong, not the gate — it used the table's
  header text rather than the template's actual italic placeholder row, which is INST-3 doing its job
  on a checker rather than on a mutant.
- [x] **The body carries the repo-local template's sections as well as the canonical ones.**
  `a2dh pr-template harmonize` classifies the two as **DIFFERENT** — the local requires `## What`,
  `## How to see it`, `## Checks` and `## Notes for the reviewer`, which the canonical lacks — and
  recommends keeping both. Each local section does distinct work rather than restating a canonical
  one. **Harmonizing the repo's own template is an open item, not done here** (`--emit`); doing it
  mid-release would change the contract the release is being checked against.
- [x] **No attribution and no internal names in the body**, per the standing rule and enforced by
  `lint-watermark`.

## The release

- [ ] `git branch release/v0.15.0 $(git commit-tree HEAD^{tree} -p main-publish -m "0.15.0: …")`
  — verify before pushing that the candidate's tree is byte-identical to the feature tip, that
  `main-publish` is its only parent, and that it has an empty diff against the tip.
- [x] Pushed; **PR #19** opened against `main`.
- [~] **CI round 1 (2026-09-09, `02fb023`): a SPLIT result on one commit.** `verify (ubuntu-latest)`
  **passed** in the `pull_request` run and **failed** in the `push` run — same SHA, four minutes apart.
  Not a code defect: `mutant-check` hit `go test`'s **default 10-minute timeout**. The passing run took
  **8m06s** for 172 mutants on `ubuntu-latest`, against **250s** on the developer's machine, so the
  margin was invisible locally and adding one mutant at SHIP is what crossed it. Fixed with an explicit
  `-timeout 40m` and the measurement recorded in the Makefile. **F-64.** macOS passed in both runs, and
  `policy` passed in both.
- [x] **CI round 2 (`619e841`): GREEN on both legs of both runs** — `policy`, `verify (macos-latest)`
  and `verify (ubuntu-latest)` all pass under `push` and `pull_request`. **And the fix was load-bearing
  rather than precautionary:** the two green `mutant-check` runs took **396 s** and **602 s**, and 602 s
  is **two seconds** under the default that had just failed. Three observed Linux runs span **396–602 s**
  on one commit and one platform.
- [x] **CI green on the PR — round 3 (`7db2cd0`), all six checks.** Three rounds, and **every round's
  failure was a real finding rather than runner noise**, which is the budget line from above coming
  out true: round 1's split result was the mutant corpus crossing `go test`'s default timeout (F-64),
  and the round-3 timings — **642 s and 648 s** — are BOTH over that old 600 s default, so this round
  would have failed both ubuntu legs without the fix. Observed Linux range is now 396–648 s. Budget for rounds: this branch's last CI was red, and the Linux leg has
  not run on the 41 commits since. **Expect Linux-only failures and treat each as a finding**, not as
  runner noise.
- [x] **Squash-merged as `fd761ab`.** CHANGELOG date re-checked against the merge day (2026-09-09) before tagging — it matched, so 0.14.0's second PR had no counterpart here. The merged tree was verified byte-identical to both the release commit and the feature tip BEFORE the tag was cut.
- [x] **`v0.15.0` annotated on `fd761ab`, pushed 2026-09-09.** Release workflow `34352664330` green;
  it re-ran the full `make verify` against the tag before publishing, so the artifacts come from a tree
  that passed on the runner. **8 assets**, both Linux binaries among them.
- [x] **The PUBLISHED artifact was verified, not just the build.** `watchpost-darwin-arm64` downloaded
  from the release: checksum matches `checksums.txt`, it reports `watchpost version 0.15.0`, and it
  carries **0** build-path strings — the `-trimpath` fix confirmed on the real artifact rather than
  locally, against **109,594** total strings, so the zero is not a vacuous scan.
- [x] Local `main` carries the feature tip (`d7f2605`), tree byte-identical to `origin/main`; `main-publish` mirrors `fd761ab`.
- [x] Deleted `origin/release/v0.15.0`, `origin/feature/0.15.0-pre-broadcaster-ui-improvements` and both local branches. **`origin` now carries `main` alone.**
- [x] **Issues reconciled against what the release MEANT to do with them — and one was wrong.**
  #13 and #17 closed correctly from the plain `Closes #13. Closes #17.` in the release commit.
  #14 did NOT close from `Closes **#14**` in the PR body, and was closed by hand. **#12 was closed by
  the PR body's sentence saying it should stay open** — GitHub's parser matches `close … #12` and does
  not read the `not`. Reopened, with the state of the audit recorded on the issue. **The lesson is in
  `06_docs/quality-observations.md`: put close directives in the release commit message, unadorned,
  and nowhere else — and check every touched issue's final state, because three of the four were right
  and the check would have looked like a waste right up until it was not.**
  **#17's closure is EARNED, and was verified rather than assumed:** the release found the root cause
  — the fake player was missing a rule the real `oto` states outright
  (`if p.eof && len(p.buf) == 0 { return }`), so it was an instrument defect and not an engine stall —
  and pinned it with a test that forces the losing order instead of waiting for it.

## After

- [ ] **VALIDATE on the Arch box, against the installed release** — the Linux leg has never validated
  this release's code outside CI.
- [ ] **DEBRIEF written** — `08-reports/debrief.md`.
- [ ] **Carried items recorded in `project-watchpost-follow-ups`**: F-63, F-62, F-61, F-60, F-59,
  F-58, F-55, the demo location in 67 fixture files, and the A2DH items — P10 as a linked skill, the
  INST-1..5 rules folded into FULL TDD, and the repo/canonical PR-template harmonization.
