# Key Learnings — B3: working UX-backwards

> Captured during B3 UAT (sessions 59–72). Audience: the next person who plans a phase.

## 1. The UI is the best requirements document we have

Every infrastructure change in B3 was pulled by a screenshot, not pushed by a plan:

| Finding on screen | What it turned out to require |
|---|---|
| "Carlsbad shows UNKNOWN n/a" | station fallback chains, forecast rehydration, fail-soft batches, retry-before-cadence |
| "no maritime for San Francisco / San Diego / Seattle / Miami" | provider selection by station *type*, upper-case product ids, tide gauges as a temperature source |
| "tides missing for my favourites but present for New York" | a priority lane in the HTTP client, per-provider publishes, concurrent fan-out |
| "lookups re-request the entire list" | incremental commits (`SetLocations` / `Update`) |
| "lots of n/a in TODAY HI" | the gridpoint fill — and, once measured, the whole caching strategy |

None of these were visible from the architecture document. The lesson is not "skip planning" —
it is that a plan cannot anticipate a data source's quirks (a mesonet station without a sky
condition, a NOS gauge listed under a lower-case id, a "weak and variable" current station), and
that the fastest way to find them is to render real data for real places and look.

## 2. Measure before designing

Three times the first hypothesis was wrong and a probe fixed it: the tide "throttle" (it was a
probe bug plus starvation), the CO-OPS 403s (station-list bursts, not prediction calls), and the
request-count vs. payload question (the APIs meter count; we pay for bytes). Every design in
`caching.md` and the rehydration tiers rests on measured lifetimes and sizes, not on reading docs.

### 2a. Measure the process, not the pipeline (UAT 74)

The UAT 73 probe reproduced btop's 137 threads, the fixes took it to 15 — and the real binary
still showed 90. The probe's own first run had warmed the disk cache, which spread the launch
out; the real launch fired 300 publishes in a second. Two lessons: a probe must start from the
same state as the thing it models, and the cheapest instrument for a live process is Go's own
`/debug/pprof/` behind an environment switch — two goroutine dumps 80 ms apart said more than
an afternoon of theory.

## 3. Modularity paid for itself within days

`FetchEach` (moved at its second caller), `geo.HaversineKM`, `stationNote`, `wrapModal`, the
`marineRow` two-pass layout: each was extracted because a second caller appeared, and each was
then reused a third time within the same week. The "second caller" rule is cheap to follow and
expensive to skip.

## 4. Fewer concepts beat clever ones

`platform/memo` was a correct, tested TTL memo. It lasted eleven sessions. Once the client itself
cached by URL, a second caching concept only cost explanation. Retiring it was the right call
even though nothing was wrong with it.

## 5. Gate discipline is a human problem, not a tooling one

Three commits landed with a failing test or a live P10 finding because a shell chain was written
to keep going. The fix that stuck was procedural: run each gate as its own step, read it, then
commit — never `a && b; c`.

## 6. "Borrow the style" means read the source, then fit it to the content

The visualizer brief named CLIAmp as the reference (UAT 92). Its README says only "a spectrum
visualizer"; the look lives in the code — band edges, the block glyphs, the 0.6/0.25 smoothing,
the 0.3/0.6 gradient thresholds. Reading those made the port exact where it mattered and showed
where it must NOT be exact: CLIAmp's edges run to 20 kHz for music, and on a 32 kbps weather relay
that leaves four bars dead. The deviations (voice-weighted edges; per-band tint on a single row)
are recorded next to the fidelity, so a reviewer can tell homage from drift.

## 7. CI is a different machine — rehearse the tag before you need it

Four release runs for one tag, and none of the three failures was a product bug: a test that assumed
macOS, a lint that assumed a `main` ref on a detached tag checkout, a 10 ms budget that the race
detector turns into 14. Every one was visible in advance to anyone who asked "what is different on
the runner?" — OS, ref layout, instrumentation. Now `GOOS=linux go vet` and a dry tag on a scratch
branch are part of the exit, and timing budgets carry a race-aware constant.

## 8. Identity is infrastructure; check it before every outward step

The account, the key, the agent, the managed config: each one independently decided whether a push
went out as the right person, and two of them changed under us during a single afternoon (a managed
`~/.ssh/config` rewrite; a key dropping out of the agent). The rule that held: verify the acting
identity (`gh api user`, `ssh-add -l`) immediately before an outward action, and pin the repo to one
key with the ambient config ignored. The rule that failed: assuming a setup that worked an hour ago
still does.

## 9. Validation on the second machine finds a different class of bug

Nothing the Linux run found was a defect in logic. It found what the first machine could not show:
that the one installed voice was the only voice, that ten seconds of silence reads as "broken", that
a font may lack the glyphs. The fix-forward cadence (four small releases in a day) was cheap because
the pipeline was already green and the record already explained each change — the exit reports paid
for themselves here, not at the gate.

## 10. Content-keyed caches, or the cache lies under repeat

The fire report's audio was cached by position; under Repeat: Watchlist a changed hotspot count would
have replayed yesterday's sentence while the marquee showed today's. Anything that caches rendered
output of changing data keys on the content, not the slot — the same rule the observation segments
already followed, and the review lens caught the one place that didn't.
