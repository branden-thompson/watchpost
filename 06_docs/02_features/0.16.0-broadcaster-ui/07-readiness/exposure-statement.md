---
title: "0.16.0 exposure statement"
status: "Re-derived at BUILD exit, 2026-09-15.  The delta is location and identity; credential remains clean."
---

# Exposure statement — 0.16.0 Broadcaster UI

**The repository is PUBLIC, and 0.15.0's statement asked for exactly this.** That document closes with
a standing instruction: *"the exposure keeps GROWING at the rate the project ships … this session's
own work took `location` from 137 files to 139 while nobody intended it. **Re-derive rather than
cite.**"*  0.16.0 shipped `scripts/quality/exposure-scan.py` and then did not run it for itself —
found by red team at BUILD exit.  This is that re-derivation.

## The numbers, re-derived 2026-09-15

| Category | Tracked tree (files / occurrences) | 0.15.0 | Δ | Git history | Tags |
| --- | --- | --- | --- | --- | --- |
| **identity** | 115 / 274 | 23 / 133 | **+92 files** | 1,354 | 19/19 |
| **location** | 192 / 690 | 139 / 464 | **+53 files** | 7,059 | 19/19 |
| **host** | 12 / 28 | 11 / 23 | +1 file | 211 | 19/19 |
| **internal-url** | 0 / 0 | 0 | — | 0 | 0/19 |
| **credential** | 7 files / 11 matches, **2 distinct, both fixtures** | 2 distinct | **no change** | 68 | 19/19 |
| **path** | 14 / 83 | — | — | 119 | 19/19 |

## What changed, and what it means

**`location` grew by 53 files, and the growth is the release's own documentation.**  The Broadcaster
is a geographic product: its rulings, its mocks and its build logs name Oceanside, Vista, Bonsall,
Rainbow and the rest because those are what the fence, the pool and the hyper-local case are ABOUT.
~19 of the new occurrences are Go test files added on this branch (`app/airscope_test.go:31`,
`app/bedfence_test.go:24`, `modes/tty/broadcaster_uat_test.go:213`).

**This is not a new exposure class.**  `platform/snapshot/origin.go:23-24` already ships
`Lat: 33.2886, Lon: -117.2247` and is on `origin/main`, and the HUM LEAD ruled the demo location
*"keep as is for now"* on 2026-09-08 (0.15.0's statement).  **The digits added by 0.16.0 fall inside
that standing ruling.**  What 0.16.0 owes is the count, and here it is.

**`identity` grew by 92 files** — the release's documentation tree carries the author's name in
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

## Standing instruction, restated for 0.17.0

**Re-derive rather than cite.**  These numbers are true on 2026-09-15 and will be wrong by the next
release.  `python3 scripts/quality/exposure-scan.py`, and write the delta down — the release that
built the scanner is the release that forgot to run it, which is exactly how a standing instruction
decays.
