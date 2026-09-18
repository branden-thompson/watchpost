---
title: "0.16.0 exposure statement"
status: "Re-derived at REVIEW exit, 2026-09-18, at 73bce2b.  The delta is location and identity; credential remains clean."
---

# Exposure statement — 0.16.0 Broadcaster UI

**The repository is PUBLIC, and 0.15.0's statement asked for exactly this.** That document closes with
a standing instruction: *"the exposure keeps GROWING at the rate the project ships … this session's
own work took `location` from 137 files to 139 while nobody intended it. **Re-derive rather than
cite.**"*  0.16.0 shipped `scripts/quality/exposure-scan.py` and then did not run it for itself —
found by red team at BUILD exit.  This is that re-derivation.

## The numbers, re-derived 2026-09-17 (third derivation)

**THE FIRST VERSION OF THIS TABLE WAS DERIVED BEFORE THE COMMIT IT LIVES IN.**  Red team round 2
caught it: the scan reads `git ls-files`, and `2d7c21e` added 49 files — including the 572 lines of
the P7 build log — so the numbers this document published were already stale when it was committed.
The document whose thesis is *"re-derive rather than cite"* had cited itself.  These are the figures
from the tree as it now stands. **Re-derived again at BUILD exit, 2026-09-17** (red team F-145: the
second table was already wrong against its own scanner): `python3 scripts/quality/exposure-scan.py` at
`59c8b38`, **and again at REVIEW exit, 2026-09-18, at `73bce2b`** — the docs-quality reviewer found this
statement's own `/Users/you/…` example inflating the identity count (the scanner harvests every
`/Users/<x>` as an account name), so the example is spelled `~/…` now and the numbers below are the
re-run after that edit. The tags column reads 21 because two local safety-net tags exist; the
published set is 19.

| Category | Tracked tree (files / occurrences) | 0.15.0 | Δ | Git history | Tags |
| --- | --- | --- | --- | --- | --- |
| **identity** | 137 / 341 | 23 / 133 | **+114 files** | 7,070 | 21/21 |
| **location** | 214 / 750 | 139 / 464 | **+75 files** | 7,836 | 21/21 |
| **host** | 12 / 28 | 11 / 23 | +1 file | 240 | 21/21 |
| **internal-url** | 0 / 0 | 0 | — | 0 | 0/21 |
| **credential** | 7 files / 11 matches, **2 distinct, both fixtures** | 2 distinct | **no change** | 69 | 21/21 |
| **path** | 4 / 5 | — | — | 138 | 21/21 |

## What changed, and what it means

**`location` grew by 75 files, and the growth is the release's own documentation.**  The Broadcaster
is a geographic product: its rulings, its mocks and its build logs name Oceanside, Vista, Bonsall,
Rainbow and the rest because those are what the fence, the pool and the hyper-local case are ABOUT.
~19 of the new occurrences are Go test files added on this branch (`app/airscope_test.go:31`,
`app/bedfence_test.go:24`, `modes/tty/broadcaster_uat_test.go:213`).

**This is not a new exposure class.**  `platform/snapshot/origin.go:23-24` already ships
`Lat: 33.2886, Lon: -117.2247` and is on `origin/main`, and the HUM LEAD ruled the demo location
*"keep as is for now"* on 2026-09-08 (0.15.0's statement).  **The digits added by 0.16.0 fall inside
that standing ruling.**  What 0.16.0 owes is the count, and here it is.

**`identity` grew by 114 files** — the release's documentation tree carries the author's name in
frontmatter and in quoted rulings throughout.  Deliberate, and the same disposition 0.15.0 recorded:
this is a personal project published under its author's own name.

**`credential` is unchanged and clean.**  Two distinct credential-shaped values across the whole
history, both literal hex ramps in test fixtures.  0.16.0 added none.  This was the category most
likely to be wrong and it is the one that did not move.

**`internal-url` is zero, in the tree, in history, and in every tag.**

## What is NOT in this scan, and was checked separately at BUILD exit

- **AI attribution**: zero occurrences across all 361 commits on this branch and in every tracked
  file — the only matches are `scripts/lint-watermark.sh` and its own test fixture.  `lint-watermark`
  is in `verify` and in `required-gates.txt`.
- **Real GPS in mocks and docs**: none.  `mock-broadcaster-v3.txt` carried `88.8888888,
  -888.8888888` — an impossible coordinate, declared a placeholder in the file, but not the ruled
  form; it now reads `<lat>, <lon>` like v1, so a grep keyed on the ruled form finds every mock.
- **Built artifacts** carry `identity` and `path` in the hundreds per binary.  That is Go embedding
  the build path, it is unchanged from 0.15.0's finding, and `dist/` is git-ignored — nothing there
  is published except through a release asset, which is the same disposition as before.
- **A 4.4 MB tool binary was in this branch's HISTORY**, added in two commits and untracked in a
  third.  None of the three is an ancestor of `origin/main`, so merging 0.16.0 would have published
  the blob permanently.  Found by red team round 2; **the HUM LEAD ruled it out — "No binaries in git
  branch history per standard best practices"** — and the branch history was rewritten to remove it.
  An earlier version of this statement asserted *"nothing there is published"* while the blob sat in
  the history this release proposes to merge.

## Standing instruction, restated for 0.17.0

**Re-derive rather than cite.**  These numbers are true on 2026-09-18 and will be wrong by the next
release.  `python3 scripts/quality/exposure-scan.py`, and write the delta down — the release that
built the scanner is the release that forgot to run it, which is exactly how a standing instruction
decays.

## `path` — the disposition the table lacked

**What it is.** A `path` hit is an absolute filesystem path naming a machine — a home directory, a
harness scratchpad, a build root. It is the category the 2026-09-16 identity leak fell into: twelve
files carried `/Users/<account>/…` and ten carried an agent-harness path, on both pushed branches.

**Disposition.** Fixed forward, never rewritten: the dead artefacts were deleted, the records scrubbed of
the account name with their findings kept, and `TestThePublishedTreeNamesNoPersonOrMachine` now asks
the WHOLE index (it had been scoped to one file while the class was live in twenty-one others). The
tracked tree stands at **4 files / 5 occurrences**, each a documentation example of a home directory
with a placeholder account name, which the gate's `reservedForDocs` pattern admits by name. **History keeps its 137**, as every
category does — `git rm` removes none of these, and published history is never rewritten. **Built
artifacts** carry `path` in the hundreds per binary because Go embeds the build path; `dist/` is
git-ignored, `release-matrix` builds with `-trimpath`, and `TestEveryBuildTargetTrimsThePath` holds it.

**Blind spot.** The scan reads the tracked tree and git objects; it does not read a contributor's
untracked files, and a path without a `/Users/`, `/home/` or harness prefix is not a `path` to it.

