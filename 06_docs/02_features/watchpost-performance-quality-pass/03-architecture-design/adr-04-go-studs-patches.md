# ADR-04 — go-studs changes are patches beside the kit, upstream first (C4)

**Status:** accepted (PLAN v2) · **Owner:** HUM LEAD (copyright holder of go-studs)

## Context
`scripts/sync-go-studs.sh` does `rm -rf` + copy: any local edit vanishes on the next sync (JD-1). The kit
carries real defects on the app's path (the per-frame `/dev/tty` probe, byte-width truncation, the `$TERM`
palette gate that violates NO_COLOR).

## Options
1. Patches under `06_docs/…` re-applied by the script (PLAN v1).
2. Patches under `third_party/go-studs/patches/` + `LOCAL_CHANGES.md` with a pinned upstream commit;
   the script copies to a temp dir, `git apply --check`s every patch, applies, swaps atomically,
   regenerates licences, re-runs P10 on the kit, fails loud naming the patch.
3. Land every change upstream first and sync from that commit (PA-5).

## Decision
2 for the correctness patches (Q4a: NoAutoStyle/NO_COLOR, the lazy probe, composite SGR, bounded
loops) and 3 for the performance patches (Q4b: width authority, geometry, no-wrap, chroma, spinner,
theme lock) — landed locally only if Q3 + Q4a miss the frame gate.

## Consequences
The local patch stack stays ≤ 5; every patch is an upstream candidate with its test; the kit's exemption
entries are itemised per file and retired per patch (no directory-level collapse — SC-1).
