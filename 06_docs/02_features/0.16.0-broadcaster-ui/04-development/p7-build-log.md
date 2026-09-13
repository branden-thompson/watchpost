> **Reconstructed 2026-09-13**, from the commits, the tests and a re-run of the corpus.  See
> `p4-build-log.md` for why the record ran a month behind the code.

# P7 — instruments and gates

**Plan:** *"FR-11 — runs THROUGHOUT, closed here."*  It did run throughout; this records what it
closed with.

## The performance work, and the number that made it honest

**D-120 — the budget was measuring the path that does not cost.**  `bcFrameAllocs` had spent its whole
history measuring a console with **no pool and no snapshot**: no row ever joined its weather, no pool
table drew a location, and the expensive path was never in the number at all.

HUM LEAD, on being shown that: **re-base it.**  The figure went UP because the measurement got honest,
not because anything got slower — and the budget now **validates its own fixture** (`loadedJoins`)
before it reports, because a budget that cannot tell whether it measured the expensive path is a
budget reporting on nothing.

**Only half of D-120's optimisation paid, and the other half was RETRACTED.**  The day-cell skip saved
351 allocations a frame — two tables building five `DayCell`s per row that neither draws.  The
map-once change measured **alloc-neutral (9296 → 9296) and time-neutral (503 µs vs 519 µs,
overlapping)**, and the claim was withdrawn in the code comment and to the HUM LEAD.

**AND ONE MEASUREMENT IN THAT ROUND WAS VOID.**  A `git stash` round-trip mid-measurement reverted the
optimised files, so the same tree was benchmarked twice and 5% of noise was read as signal.  Caught
only by validating the fixture.  It is recorded because the lesson is not "be careful with stash" — it
is that **a measurement whose instrument was not checked is not evidence**, and this one looked
exactly like a result.

## D-123 — the console memoises its two tables

Measured on the loaded fixture before anything was changed, which is the whole method:

| | before | after (hit) | after (miss) |
|---|---|---|---|
| time | 537 µs | **197 µs** | 508 µs |
| bytes | 421 KB | **107 KB** | 420 KB |
| allocations | 9298 | **530** | 9298 |

**The prize was sized before it was built for**: a throwaway probe put the two spans at 320 µs and
8796 allocs — 60% of the frame's time and 95% of its allocations.  **Nothing came off the miss path**,
which is the claim that matters when a cache is added.

**What makes a memo safe is not the key, it is the guard.**  Every one of the twelve key fields was
deleted and the completeness guard watched to go red.  **Six were not caught the first time**, and
every one was a real hole — `used` and `theme` cannot be reached by perturbing the model at all;
`lineupGen`/`areaGen` stand for values the walk cannot perturb; `recent` is a pointer; `fireBoldMW`
reached nothing because the fixture had no fire; `shimmer` needs something LOADING before the frame is
an input.

**The engine is generic now rather than copied**, and it perturbs FLOATS, which it never did — it
listed the kinds it handled and fell through the rest in silence, the same shape as the hand-written
key it exists to check.  Observer's own guard passes with them included: 88 perturbations became 91,
no new findings, one blind spot fewer.

## D-122 — the fence bypass, and the direction of a wrong fix

A zone-only alert has no point, so no fence can measure it; the only thing that admits one is the app
already following that hazard at a watched location.  `arrivalsOf` stamped `Tracked: true` on every
arrival — true of the scope that PLANNED the card, false of every scope after it.  **It is the one
bypass a narrower radius could not close, because narrowing is not what it consulted.**

**The over-correction is the worse failure and is pinned separately.**  If the arrival's key and the
tie set's key normalise differently, NOTHING zone-only is admitted and a flood warning at the
listener's own location goes unread.

## `make mutant-anchors` — the gate this release added

**RATIFIED BY THE HUM LEAD 2026-09-13** after four anchors drifted in one session.  306 mutants, 0.18
seconds; `mutant-check` answers the same question in ~400 because it also compiles every mutation.

**It runs each mutant's OWN assertion** — executed with `write_text` neutered — rather than grepping
`old = "…"`.  A second parser for the corpus can disagree with the harness, and the first draft did:
it decoded anchors with `unicode_escape` and reported a false drift on the one anchor containing a
`◆`.  Four mutants also COMPUTE their anchor, which no regex sees.

Four controls, all watched firing; unconditional in CI, where `mutant-check` is policy-scheduled and
therefore absent from an ordinary PR.
