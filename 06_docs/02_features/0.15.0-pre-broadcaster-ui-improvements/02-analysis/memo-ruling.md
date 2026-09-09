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

## Metric K, with its denominator fixed FIRST

**K — memo keys with no completeness guard.**  Baseline at DISCOVER: *"1 on the
frame path (`bodyKey`); 6 data caches pending FR-3.2's ruling"*.  Target 0.

The red team's hardening note is the reason this section exists: *"K must stay
an absolute count: FR-3.2's ruling on the six data-cache memos removes them from
the numerator by definition rather than by guarding them, which is the
anti-solution the brief hardened D against, re-imported."*

**So the denominator is stated before the verdicts, and it is EIGHT** — the two
frame keys and the six named memos.  Nothing is dropped from it; the two that
cannot have a key-completeness guard are marked N/A **with what they are**,
which is a fact about their shape and not about this release's convenience.

| Key | Can it go stale? | Guard | K |
|---|---|---|---|
| `modalKey` | Yes | `TestTheMemoKeyCoversEverythingTheFrameShows` — now FAILS on an unfixtured window and derives the nested-struct set (FR-3.3) | 0 |
| `bodyKey` | Yes | `TestTheBodyKeyCoversEverythingTheTablesShow`, its theme twin, and `TestEveryBodyKeyInputIsExercised` (FR-3.1) | 0 |
| `tileMemo` | Yes | `platform/bodymemo`'s three property tests, the firms allocation pin, and its own bound/split test | 0 |
| `boxMemo` | Yes | The same three, plus the shared-regional-body test | 0 |
| `gridMemo` | Yes | The same three, plus `Retain`'s prune test | 0 |
| `sourceMemo` | Yes | `TestSourceMemoSkipsDecodeOnAnUnchangedBody`: one decode per body, a changed body re-decodes, a cache hit does not, and a parse error is not swallowed | 0 |
| `hostMemo` | **No — it holds no parsed value.**  A consecutive-5xx count and a deadline | N/A, stated | — |
| `failureMemo` | **No — the map those live in** | N/A, stated | — |

**K = 0 of 6 answerable keys, with 2 of 8 declared unanswerable and why.**

**And one thing K does not say, recorded so nobody reads the zero as more than
it is:** `bodyKey` has a guard, and that guard's own coverage is **measured** —
six of its twenty-two inputs are exercised by nothing and are declared in
`bodyKeyUnexercised` with what would close each.  A key with a guard is not the
same as a key whose every input is exercised, and the second number is the one
worth watching.
