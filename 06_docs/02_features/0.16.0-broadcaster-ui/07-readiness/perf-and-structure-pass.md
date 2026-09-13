---
title: "0.16.0 — the pre-BUILD-exit performance and structure pass"
date: 2026-09-13
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "MEASURED.  Two items await a HUM LEAD ruling."
---

# The performance and structure pass

**Ordered by the HUM LEAD before BUILD exit**: *"I do want to do a full performance and code structure
pass on all of our work before we get to Build Exit."*

## Performance — every frame path, measured today

Apple M3 Pro, `-benchtime 200x -count 3`, medians.  **The frame redraws every 300 ms for the life of
the process** while the marquee has anything to say (`tickNeeded`, accepted-costs §2), so every
per-frame number here is a 24/7 number.

| Path | time | bytes | allocs | Standing |
|---|---|---|---|---|
| Observer frame 133x44 (memo hit) | 98 µs | 89 KB | 376 | within its pinned budget |
| Observer 133x70 / 200x60 | 123 / 154 µs | 115 / 147 KB | 394 / 397 | scales with area, as expected |
| Observer frame (memo MISS) | 387 µs | 395 KB | 5671 | within its pinned budget |
| **Console frame (memo hit)** | **214 µs** | **107 KB** | **545** | **D-123 — was 537 µs / 421 KB / 9298** |
| Console frame (memo MISS) | 559 µs | 421 KB | 9298 | unchanged by the memo, which is the point |
| Observer + a window open | 1124 µs | 888 KB | 2294 | **accepted cost §1** |
| Console + its card window | 1419 µs | 1619 KB | 1644 | **the same accepted cost, scaled by area** |

### The largest cost in the app was already ruled on, and my proposed fix was already rejected

`render.Overlay` is **1017 µs · 763 KB · 1434 allocations** — measured today on
`BenchmarkOverlayOnly`, the instrument `docs/accepted-costs.md` §1 names.  That is ~84 % of a frame
with a window open, and it is the single most expensive operation in the application.

**It is a HUM LEAD ruling from 2026-08-30 and it stands.**  Re-derived here, and the pass nearly went
further than that: the obvious fix — memoise the composite on (base, window, width) — **is written
into §1 as already considered and rejected ON MEASUREMENT**, because `advanceTicker` steps the marquee
one cell per tick, so the base changes every frame and such a memo would miss every time.  That
rejection still holds.

**Drift since the acceptance was recorded:** 1042 → 1017 µs, 765 → 763 KB, **1346 → 1434 allocations
(+6.5 %)**.  Small, and in one direction worth watching.  The acceptance's re-open triggers are
unchanged and none is met: no cheaper upstream API, and nothing user-visible.

**The console pays the same cost about 1.9× over**, because it draws at 150x74 against the
acceptance's 133x44 fixture and §1's own explanation is *"the cost is the area the window does not
cover"*.  1.6 MB a frame while a card window is open.  Still bounded — a window is open for seconds,
not for the life of the process — and still not a re-open trigger at 1.4 ms against a 300 ms tick.

### A measurement error worth recording rather than hiding

An ad-hoc probe of the same compositor read **1883 allocations** and was very nearly written up as a
40 % regression.  It was a **different fixture** — the help window on `benchDash` rather than the
severe window on `severeBench`.  §1 names its instrument precisely so that a comparison is possible;
substituting a probe for it manufactures findings.  **The acceptance's own benchmark is the only
number that may be compared to the acceptance's own number.**

## Structure

| Check | Result |
|---|---|
| `modes/` → `domains/` | none.  Gated by `lint-imports` |
| `platform/` → `domains/` or `modes/` | none.  Gated, including the one-hop hole |
| `modes/` → `app/` | none |
| `ForTest` exports | **five**, and two are not where the rule says |

### Two comments asserted things that were not true

**1. `modes/tty/setup.go` said "IN `platform/` BY THE ARCHITECTURE'S OWN RULE"** — in a file that is
in `modes/tty`.  It stated the rule and asserted compliance in the same breath from the package the
rule points away from.  Corrected to say where it actually is and why.

**2. `app/livesource.go` said "all four `ForTest` exports in the tree are in `platform/`."**  There
are five and two are in `modes/tty` (D-115).  The rule the sentence exists to state is unchanged; the
census beside it went stale without anyone touching the file, which is what a hand-kept count in a
comment always does.

**Neither was a code defect.  Both were the record lying about the code**, which is the same class of
problem as the roster that stopped at P3 — and it is the reason this pass exists.

### FOR THE HUM LEAD — the service radius' bounds have two carriers

`serviceRadiusMin/Max = 2/100` in `modes/tty/setup.go`, and `config.MinServiceRadiusMi` /
`MaxServiceRadiusMi` in `platform/config`.  What stops them drifting is
`TestSetupServiceBoundsMatchTheConfig` in `app`, the only package importing both.

**Importing `platform/config` from `modes/tty` is possible** — it compiles and `lint-imports` passes —
**but no package under `modes/` does it.**  The UI is handed its configuration through `tty.Config`
rather than reading storage, and **D-91's air boundary is a classification of exactly those seams**.
This would be the first breach of that convention, to save a constant.

| Option | Cost |
|---|---|
| **A — leave it tied** | Two carriers, drift prevented by a test that already exists and already fires.  No change, no risk |
| **B — a neutral owner in `platform/`** both `config` and `tty` import | One carrier.  A new package for two integers |
| **C — pass them through `tty.Config`** like every other configured value | One carrier, and consistent with the convention.  Two new fields, each owing a D-91 airBoundary classification |

**Recommended: A for 0.16.0, C as a follow-on.**  The duplication is real but it is *pinned*, and C
touches the seam D-91 spent a batch classifying — which is not a thing to do in the last hour before
BUILD exit.  **The ruling is the HUM LEAD's.**
