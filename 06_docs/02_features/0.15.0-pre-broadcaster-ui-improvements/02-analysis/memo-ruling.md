---
title: "0.15.0 — What a memo owes (OQ-9), and the six that were asked"
date: 2026-09-07
phase: BUILD · B4
sev: SEV-0
authority: "HUM LEAD ratified 2026-09-07"
status: "Ruled. Three of six consolidated; three declined, with reasons."
---

# What a memo owes

## The ruling

**Ratified by the HUM LEAD, 2026-09-07.**  A data-cache memo owes three
properties, and they are all observable:

1. **A hit returns what a parse would.**  Revalidation by the body's own hash is
   how it earns that: a changed body **misses**.
2. **A hit does not parse.**  The parse count is the observable, and the
   provider allocation pins hold a hit at **zero allocations**.
3. **It is bounded** — by count with least-recently-used eviction, or by the
   caller's own liveness rule.  Both are real bounds; neither is optional.

**And the non-property that shapes all three: a memo may MISS spuriously — that
costs time, not correctness — but it must NEVER wrongly HIT.**  It is the same
asymmetry the frame memo's key guard is built on, one layer down: *"a memo may
miss, it must never wrongly HIT"* (`memo_completeness_test.go`).

## The six, and what each is

| Memo | Verdict |
|---|---|
| `tileMemo` (`domains/fire/firms`) | **Consolidated** into `platform/bodymemo`.  Bounded by count |
| `boxMemo` (`domains/seismic/usgs`) | **Consolidated.**  Was byte-identical to `tileMemo` (F-53) |
| `gridMemo` (`domains/weather/nws`) | **Consolidated.**  Bounded by LIVENESS — `Retain` prunes the grids of locations that left the watchlist — which is why `bodymemo` grew `Prune` |
| `sourceMemo` (`domains/globalfeed`) | **Declined, and the reason is a contradiction of property 2.**  Its hit MUST allocate: it hands every caller a cloned slice because `Locate` writes into the elements.  It also skips hashing entirely when httpx reports the body came from cache untouched — a 1 MB national pull against a two-minute cadence — which `Parsed(key, body, parse)` cannot express, because the whole point is not to touch the body |
| `hostMemo` / `failureMemo` (`platform/httpx`) | **Declined: not a parse cache at all.**  A per-host consecutive-5xx count and a deadline to avoid the host until.  There is no body, no parse and no value to return, so none of the three properties has anything to say about it.  The name is the only thing it shares |

**Two of the three declines are worth more than the consolidation.**  A shared
type that had absorbed `sourceMemo` would have carried a clone-on-hit that the
other three must not have, and one that had absorbed `failureMemo` would have
been a parse cache with no parse in it.  *A common name is not a common shape.*

## What this changes

- `platform/bodymemo` is the single implementation of the shape, with the three
  properties as its own tests plus one neither provider had: **a failed parse is
  not stored**, so one bad body cannot become permanent until it changes.
- The two declines are recorded **here** rather than re-argued at each site.
  `domains/globalfeed/memo.go` already carried its own version of this reasoning
  against `domains/fire`'s `Memo[T]`; that comment stands and this row agrees
  with it.

## What the consolidation actually cost and bought

**Measured, not assumed** (HUM LEAD asked, 2026-09-07).  The comparison is the
same benchmark and the same line count at `27e616e` — the commit before
`platform/bodymemo` existed — against the tree with all three providers moved.

### Size: −25 implementation lines, and one implementation instead of three

Blank lines and comments excluded, so the count is code rather than prose.

| File | Before | After | Δ |
|---|---:|---:|---:|
| `domains/fire/firms/tiles.go` | 108 | 70 | **−38** |
| `domains/seismic/usgs/boxmemo.go` | 54 | 11 | **−43** |
| `domains/weather/nws/forecast.go` | 206 | 196 | −10 |
| `domains/weather/nws/provider.go` | 112 | 113 | +1 |
| `domains/weather/nws/points.go` | 158 | 154 | −4 |
| `platform/bodymemo/bodymemo.go` | 0 | 69 | +69 |
| **Total** | **638** | **613** | **−25** |

**The −25 is the least interesting number here.**  A shared implementation that
replaces three copies of ~40 lines with one of 69 will roughly break even on
lines, and it did.  What changed is that the tick counter, the hash
revalidation, the eviction and the "a failed parse is not stored" rule are
written **once** — and the third caller cost 11 lines rather than 43.

### Performance: unchanged, and that is the right answer

`BenchmarkTileMemoHit` / `BenchmarkBoxMemoHit`, 200k iterations × 6 runs, the
same benchmark file in both trees:

| Hit path | Before | After |
|---|---|---|
| `tileMemo` | ~179–184 ns/op, 0 allocs | ~176–184 ns/op, 0 allocs |
| `boxMemo` | ~134–280 ns/op (noisy), 0 allocs | ~128–131 ns/op, 0 allocs |

**Within noise on both, at zero allocations either way.**  The box numbers before
are noisier than after; that is a cold worktree, not a finding, and it is not
claimed as an improvement.  A generic taking a `parse` function had one obvious
way to cost something — a closure allocation per call — and both call sites pass
a top-level function, which is why they do not.

### The part that was never going to show up in a number

Three memos can now be walked the same way, asked the same three questions, and
pinned by the same shape of test.  The fourth and fifth were **declined on the
strength of those same questions** rather than on a reading of their code — and
that is the thing that was not possible before: there was no contract to hold
them against.
