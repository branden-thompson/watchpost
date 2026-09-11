# D-72 — THE STATION'S WORLD IS NOT THE LISTENER'S

**Ruled by the HUM LEAD, 2026-09-10**, as the first of two blockers standing between the console and a
UAT that can judge ON AIR:

> We need to "split" the data state between Observer and Broadcaster.  They can and should share the
> same infra, but the Observer watchlist is its lineup, and a different rolling window / stack / list
> needs to serve as Broadcaster "Location Pool" for producers to create the lineup.

With three rulings inside it: the transmitter is **not** the default location, the service radius is a
**setting** bounded 2–50 miles, and the pool is **capped** well below the listener's world.

---

## The answer to F-81 was already in the tree

I was one message away from putting a fork to the HUM LEAD — NWS points API, a bundled place list, or
Observer's known locations — with a licensing question attached, because the repository is public.

**None of it needed asking.**  `domains/locations/geodata` already embeds **34,106 US cities** with
population and **41,490 zip centroids**, and `geodata.Load()` already runs at startup because the
resolver and the RECENT seed list share it.  The data to answer "what is near the transmitter" has
been in the binary since 0.9.0.

**That is the second time this release a question was about to be escalated that the tree had already
answered** — the first was the schedule-versus-line-up distinction, which had been ruled a week
earlier and was in the documents I had not re-read.  The lesson is the same one `00-REQUIRED-READING`
was written for, one layer out: **read what is there before proposing what to add.**

## "Major locations first" needs no ranking rule

The HUM LEAD's example was an ORDER — Fallbrook, Vista, Oceanside, Temecula — and I expected to need a
population-versus-distance blend to reproduce it.  The city table is **already population-filtered**,
so ordering by distance IS ordering by major-first:

```
  6.2 mi  Vista, CA          pop 100890
  6.3 mi  Fallbrook, CA      pop  30534
 10.6 mi  San Marcos, CA     pop  92931
 10.9 mi  Oceanside, CA      pop 175691
 14.9 mi  Temecula, CA       pop 110003
```

**Rainbow, CA is absent, and that is the tell**: at ~1,800 people it is below the table's floor, which
is exactly what "major" means here.  It is in the ZIP table — which is why there are three tiers.

## Three tiers, and the third is the point

| tier | what | why |
|---|---|---|
| **home** | the transmitter's own place | a station reads where it transmits from first; it is the most local report it has |
| **cities** | the region, nearest first | already population-filtered, so distance order is major-first |
| **zip places** | everything else in the fence | *"the big value here is the hyper local station reports"* — and the only tier that knows Bonsall's own 92003, San Luis Rey, Pala or Palomar Mountain exists |

The pool from Bonsall at 25 miles is 25 entries, home first, cities next, hyper-local behind — and
`TestTheStationsPoolFillsTheConsolesWindow` drives `startSchedule` through the same `producer()` the
wiring uses and reaches **all ten slots**.  F-82 is closed by measurement, not by argument.

## What the measurement changed

**The two-mile floor is legal and nearly empty.**  Measured around Bonsall: a 2-mile fence holds no
city and one zip place; 10 miles holds seven; 20 holds eighteen; 25 fills the cap.  The HUM LEAD kept
the floor knowing that (ruling 1, approved), and the default is **25** — the narrowest radius that
reliably gives the Director more to choose from than it must schedule, which is what makes the cadence
rule (D-47, D-48) a CHOICE rather than an inventory.

**The hyper-local tier had no timezone.**  Every place the city table does not hold came back with
`TZ: ""` — which is the whole tier, by construction — and a ref saved without a zone was a real defect
once.  It falls back to the TRANSMITTER's zone, with the assumption stated: a place inside the
station's own fence keeps the station's clock, which at fifty miles is true everywhere but astride a
zone boundary.

## The fallback is what makes the split free

Every install that exists has a default location and no transmitter.  `config.Station()` reads the
listener's default when the station has none, so **the split costs an existing station nothing** — and
`stationArea.followsDefault` remembers that the epicentre was BORROWED, so it moves when the listener's
default moves and a station with its own transmitter does not.  Without that flag, nothing downstream
could tell a borrowed epicentre from a chosen one.

## One named function instead of a call site

`startSchedule` takes three seams that all read one list — what may be proposed, what a ref resolves
against, and what the bed cuts to.  **All three had to move together**: feeding one from the pool and
another from the watchlist would let the Director schedule a location its own Composer cannot resolve,
and D-67's cool-off would then bench it for five minutes apiece.

The choice lives in `livePipelines.producer()` rather than at the call site, because **a call site
cannot be asserted and this can** — which is the defect shape this release keeps producing, and the
plant `w1` (the producer offers the watchlist again) is CAUGHT because of it.

## Deferred, and named

**F-87**: nothing in Settings writes either setting yet, so changing the epicentre or the radius means
hand-editing `config.toml`.  The radius is clamped at the reader, not at a widget, and the
fence-constrained lookup the HUM LEAD floated as a *might* is unbuilt.

## The defect the gate caught, and what it cost to find

The first version published the station's area from `setStation`, which runs inside `startPipelines`
— **before `p.Run()`**.  `tea.Program.Send` on a program whose loop has not started **blocks for
ever**, so the whole app deadlocked at launch.

**It did not fail; it HUNG.**  `TestRunWithoutArgsStartsTheDashboard` sat on it for ten minutes under
`-race` and then panicked on the test binary's alarm — a stack with no assertion in it, from a test
whose name says nothing about the station.  Ten minutes of gate time to learn "something is stuck",
and the reason was legible only by reading the goroutine dump.

**The fix is a rule worth keeping: a value that exists at launch travels on `tty.Config`; only
CHANGES travel as messages.**  The console now opens with the area and is told when it moves, which
is also why the pool is derived before the console is built rather than after.

And the second half: `Send` is now never called while `lp.mu` is held.  The commit path reads the
program under the lock and publishes after it, because the program's own handlers reach back into
these pipelines and a blocked Send under that lock is a deadlock waiting for a busy moment.

**Eight plants, eight caught.**
