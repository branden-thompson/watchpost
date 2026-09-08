# Exposure statement — what this public repository discloses (FR-7.5)

**The repository is PUBLIC and has been since 0.13.0.** This is a statement about an exposure that
has already happened, not a risk assessment of one that might. `git rm` unpublishes nothing, and each
of the 18 tags is its own complete copy of the tree.

**Scope, named, because revision 1's was wrong.** The earlier survey covered `06_docs` only. A scan
that never opens the git history, a published tag or a built artifact is scoped wrong — and the two
largest findings below are in the scopes it did not look at. Instrument:
`scripts/quality/exposure-scan.py`, re-runnable, with its own positive and negative controls.

## The table

| Category | Working tree | Git history | Tags | Built artifacts | Verdict |
|---|---|---|---|---|---|
| **identity** | 23 files / 133 | 288 | **18/18** | ~460–486 per binary | **Present, and largely deliberate** |
| **location** | 139 files / 464 | 5,835 | **18/18** | 4 per binary | **Present — needs a ruling** |
| **host** | 11 files / 23 | 195 | **18/18** | 2–3 per binary | **Present, minor** |
| **path** | 12 files / 81 | 97 | 13/18 | ~460–486 per binary | **Present — one-flag fix** |
| **internal-url** | 0 | 0 | 0/18 | 0 | **Clean** |
| **credential** | **2 distinct, both fixtures** | same 2 | — | — | **Clean** |

**Two honest caveats about the numbers.** In the binaries, `identity` and `path` are largely the SAME
strings — `/Users/<user>` matches both — so that column is one exposure counted under two headings,
not two. And occurrence counts in history are inflated by repetition: one fixture copied into sixty
blobs is sixty occurrences and one thing to judge, which is why `credential` is reported as DISTINCT
VALUES.

## What each one is

**credential — CLEAN, and this was the category most likely to be wrong.** Two distinct
credential-shaped values exist across the whole history, and both are literal hex ramps
(`0123456789abcdef…`, one lower-case and one upper) used as test fixtures. **The first version of this
scan reported 2,658 "credentials" and every one was a false positive**: in this codebase `token` is a
colour (`token = "tui.StatusQueued"`), so a secret scanner fires on the theme system. A scanner that
cannot tell those apart finds nothing and hides what it would have found.

**path — the largest fixable finding, and revision 1 could not have seen it.** `-trimpath` is set
nowhere — not in the `Makefile`, not in CI — so every binary embeds the absolute path it was compiled
from: 473 occurrences in `watchpost-darwin-arm64`, 485 in `watchpost-linux-amd64`. `release-matrix`
builds the published artifacts the same way, so this is in every release. **A documentation survey
never opens a binary.**

**identity — present and mostly intentional.** This is a personal project published under its
author's name; the README carries a byline and the About window reads "Made with ♥ by …". The part
that is not deliberate is the same build-path string as above.

**location — present, and the one that needs a HUM LEAD ruling.** The app's own demo and fixture data
is Oceanside / Vista / 92057, which the DISCOVER report identifies as a **home locality at ZIP
level**. It is spread across 139 tracked files and 18/18 tags. It is city-level rather than an
address, and it is inseparable from the product's example data — but it is the category where the
published copy actually says something about where the author lives.

**host — minor.** The development machine's name appears in pasted logs and profiling output.

## What is owed, and what is not mine to decide

1. **`-trimpath` on `build`, `build-diag` and `release-matrix`.** One flag, removes the build path
   from every FUTURE artifact, and improves reproducibility. It does nothing for the artifacts
   already published. *Recommended.*
2. **The published copy — tags, history, released binaries.** Rewriting history would break 18 tags,
   every published checksum and every release artifact, to remove city-level location data and a
   name the project is deliberately published under. **On the same footing as F-48: the HUM LEAD
   rules, not the agent.** My reading is that the cost is high and the yield is low, and that the
   honest action is to accept the published copy and stop adding — but that is a recommendation.
3. **Going forward.** If the demo location is to change to a neutral city, that is a product decision
   with UAT consequences (fixtures, goldens, the journey's own `Oceanside`), not a scrub.

**No gate.** This is a survey that informs a ruling. A gate that blocks future writes while every
published tag carries the same content defends a door that is open — the DISCOVER report's own words,
and they still hold.
