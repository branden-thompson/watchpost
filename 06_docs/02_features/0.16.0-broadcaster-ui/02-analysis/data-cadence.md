---
title: "0.16.0 DISCOVER — the data-request cadence, and why it is one mechanism with the service radius"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "Analysis — raises requirement R-8, which the brief is missing"
---

# The cadence half of the intent had no requirement, and it needs one

## Why this document exists

The intake intent named the Broadcaster difference as *"higher data request frequency/constrained
service area"*.  The brief turned the second half into requirements — R-2.3's service radius, R-4.3's
relay selection, R-7's coordinates — and **turned the first half into nothing at all.**  R-1 through
R-7 contain no cadence requirement.

That is a gap in my enumeration, not in the intent.  This document closes it, and in closing it finds
that the two halves are not two requirements.  They are one mechanism.

## What the cadence machinery already is

`platform/sched` describes itself as **"THE single freshness authority"** (`sched.go:1-3`).  A `Tier`
names a fetch kind and its cadence (`sched.go:33-34`), the scheduler fans each due tier out to the
providers that serve it, and it refuses a non-positive cadence at construction
(`sched.go:77-79`).

**The cadences are already fast, and each one carries its reason in the code:**

| Kind | Priority location | Recent locations | The rationale recorded beside it |
|---|---|---|---|
| Alerts | **20 s** | batched, 2 min | the fastest thing in the app already |
| Observations | 90 s | 10 min | |
| Marine observations | 10 min | 10 min | *"NDBC files turn over every 30 min (UAT 72)"* |
| Forecast · hourly · marine | 30 min | 1 h | *"one gridpoint download serves it and the daily fill"* |
| Fire | 10 min | 15 min | *"HMS archive refresh + FIRMS cadence"* |
| Seismic | 5 min | 15 min | *"USGS max-age=60; 'did my area shake' needs no sub-minute latency"* |

Source: `app/pipelines.go:125-149`.

**And the ticker's own comment states the governing constraint plainly** (`app/ticker.go:88-91`):

> *"The feeds' own TTLs (USGS 5 min / NHC 30 min / NWS 2 min) ride the httpx cache, so the fast tick
> only hits the network for the sources due."*

## The finding: you cannot fetch fresher than the source refreshes

**Raising the tick rate does not raise freshness.**  A tier that fires more often than the upstream
TTL re-reads the `httpx` cache and returns the same bytes.  `httpx` additionally holds a token bucket
of roughly five requests per second with jittered exponential backoff on 429 and 5xx, and clamps a
`Retry-After` (`platform/httpx/httpx.go:5`, `memo.go:17`).

So a literal reading of "higher data request frequency" — turn the numbers down — buys **more load
and no more information**, and on a 429 it buys *less* information.  Alerts are already at twenty
seconds, which leaves almost no headroom on the one kind an operator most cares about.

## What the intent actually buys, and it is the other half of the same sentence

There are two tier sets, not one.  `priorityTiers()` is fast; `recentTiers()` is three to six times
slower for the same kinds (`pipelines.go:125-149`), and the RECENT set drops alerts to a batched
two-minute sweep across the whole list.

**A location's freshness is therefore decided by which SET it is in, not by a global rate.**

That is where a constrained service area pays.  Observer watches a watchlist that can range anywhere,
so most locations must sit in the cheap tier.  **A Broadcaster operator serves one bounded area — so
everything inside the service radius can afford the priority cadence, because there is a bounded
number of them.**

**The two halves of the intent are one mechanism: the radius is what makes the frequency affordable.**
That is the requirement, and it is materially different from "make the numbers smaller".

## The arithmetic, shown — added after the DISCOVER-exit red team

**The red team was right that this document asserted affordability and showed no arithmetic.  It is
shown here.  Its own number was wrong, and that correction matters more than the finding.**

**The budget is not 5 requests per second.**  Five is the library DEFAULT (`httpx.go:232-233`), and the
doc comment at `httpx.go:5` describes that default — which is what the reviewing lens read.  **The
application overrides it: the main client is built at `RatePerSec: 30`** (`app/app.go:123`).  Tides run
on their **own** bucket at 5 (`app/dashboard.go:330,336`), so these are per-client budgets, not one
shared pool.

**Cost of one priority location, from `pipelines.go:125-135`:**

| Kind | Cadence | Requests per minute |
|---|---|---|
| Alerts | 20 s | 3.00 |
| Observations | 90 s | 0.67 |
| Seismic | 5 min | 0.20 |
| Marine obs · fire | 10 min | 0.20 |
| Forecast · hourly · marine | 30 min | 0.10 |
| **Total** | | **≈ 4.2 req/min** |

**Against the real budget:** 30 req/s = 1,800 req/min.  1,800 ÷ 4.2 ≈ **430 locations** at the priority
cadence if Broadcaster had the whole budget.  Against the lens's assumed 5 req/s it would have been ~71.

**So the headroom is roughly six times what the review calculated, and the conclusion changes:** the
constraint on the priority tier is **not** the request budget.  Two things bound it instead, and they
are the ones the cap must be argued against:

1. **The cache and the sources' own TTLs.**  Most of that 4.2 is alerts, and alerts already sit at the
   fastest cadence the app runs anywhere.  Extra locations mostly re-read cache.
2. **Everything else sharing the client** — Observer's own watchlist and its recent tier draw on the
   same bucket, and this document has no measurement of what fraction is already spoken for.  **That,
   not the ceiling, is the open number.**

**Which is why R-8.2 stands unchanged**: the bound is each source's refresh interval and politeness,
not arithmetic headroom.  **And why the cap is still not given a number here** — the number wants a
measurement of current utilisation that nobody has taken, and inventing one would be the first cadence
figure in this project without an argument beside it.

**A measurement is therefore owed at PLAN**, and it is small: current requests per minute at idle and
during a burst, on the main client, with the watchlist at a realistic size.

## Proposed requirement — R-8, for HUM LEAD confirmation

### R-8 — Freshness follows the service area

- **R-8.1** Locations within the operator's service radius are fetched at the **priority** cadence,
  not the recent cadence.  The radius is the admission rule for the fast tier.
- **R-8.2** The tier cadences remain **bounded by each source's own refresh interval and by the
  client's politeness limits.**  Broadcaster may not request faster than a source refreshes, because
  doing so returns cached bytes and risks a 429 that makes the station *less* current.
- **R-8.3** The cadence set is **one owner**.  `priorityTiers()` and `recentTiers()` are hardcoded
  functions today; if Broadcaster needs a third set, it is defined beside them and not duplicated
  into the new surface.
- **R-8.4** The operator can **see** the effective freshness — when each kind was last fetched for
  the area — because a station operator's core question is *"is what I am about to say still true?"*
  The masthead already carries an `Updated:` stamp and an API health count, which is the seam.

### What R-8 deliberately does not say

It does not name a number.  Choosing cadences is a PLAN decision that must be argued against the
sources' documented TTLs, and every existing cadence already carries such an argument in a comment
beside it.  **A new number without that argument would be the first one in the file without a reason.**

## Open question for the HUM LEAD

**OQ-8 — how many locations may the service radius admit to the priority tier?**  The radius bounds
an *area*, not a *count*: a hundred-mile radius over a dense region could admit far more locations
than a sparse one.  Something must bound the count, or a dense service area silently multiplies fetch
load until the token bucket throttles it — which degrades the station precisely where it is busiest.

Candidate answers, none recommended without the HUM LEAD: a hard cap on priority locations, a cap
derived from the bucket's budget, or a rule that the radius admits *transmitter coverage* rather than
arbitrary points.  **This interacts with R-4.3's relay selection**, which is also radius-bounded, so
the two should be ruled together.
