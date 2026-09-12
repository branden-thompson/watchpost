# MVS-D-91 — nothing from Observer reaches the air on the console

**Status:** stage A BUILT, 2026-09-12.  Closes **F-100**.
**Ruled by:** HUM LEAD, 2026-09-12.

> "masterControl is the one who determines who gets the air.  In Observer mode, Observer ALWAYS gets
> the air … In Broadcaster Mode, Broadcaster ALWAYS gets the air, so nothing from Observer should ever
> be able to re-tune, take over, or 'sneak under' to get on the air."

**The model was never in question** — D-74 and `00-REQUIRED-READING`'s *ONE DECK, ONE AIR*.  What was
missing is that of nineteen paths to the engine, exactly **one** asked.

---

## Sized before it was built (FR-2.1a)

`02-analysis/air-reachability-survey.md`.  **19** air-mutating functions; **3** live defects, **4**
latent, **12** owed nothing.  A mechanism aimed at "everything that can reach the air" would have been
aimed at nineteen when the population needing a decision was seven.

## The guard is at the caller

The survey's structural finding: `tune` serves the monitor's `SetMode` **and** the Director's `Tune`
effect; `tuneCallsign` serves Observer's relay pick **and** the console's own bed selector (D-90).  A
guard inside either method would break the half that is entitled to the air.

**The exported `tty.Radio` methods are the monitor's control surface; the lower-case internals are
shared.**  That split is the seam, and it is where `needsRead` already put its guard.

| Seam | Guard |
|---|---|
| `Radio.Tune` | refused; internal `tune` stays open for the Director |
| `Radio.Stop` | refused; `stopMonitor` extracted for the swap |
| `Radio.SetRepeat` | the Director is still told; `src.Loop` is not reached |
| `Radio.SetMode` | the preference still saves; the re-tune does not fire |
| `setCast` | the cast still saves and re-resolves; `src.Recast` is not reached |
| `PreviewVoice` | refused, with a note in the chooser |
| `narrateEvent` | refused — the `[w]` read ducks the broadcast |
| `Radio.SetVolume` | **not guarded**.  HUM LEAD: "one volume setting for the app" |

**The pattern is `setTones`', already in the tree**: *"deliberately NOT a recast … `[M]` must be
instant and must not disturb a broadcast in flight."*  Settings still apply; only the disturbance
stops.

## The trap, and the test that caught it

`silenceMonitor` calls the deck's stop **after** `owner` has moved to the console — so guarding `Stop`
refused the very silencing the swap exists to perform.  `air_test.go`'s *"taking the air to the console
did not stop the monitor"* reported it immediately.  `stopMonitor` is the unguarded internal it now
uses.

## The concrete type was the reason the seam could not be tested

`radioDeck.source` was `*synth.Source`, so no test in `app` could observe whether the guard actually
stopped a `Loop` or a `Recast` — only that the predicate said it should.  **That is precisely how
D-74's plant `y4` survived**: *"my tests asserted `monitorHasTheAir()` — the PREDICATE — and never that
`needsRead` actually skips the audio."*

`liveSource` (app/livesource.go) is an interface **declared where it is consumed**, nine methods wide,
satisfied by `*synth.Source`.  Nothing is added to the domain.

**The first draft got this wrong and the HUM LEAD caught it.**  It added `RepeatingForTest` to
`synth.Source`, citing `lineup.MonitorAdvancesForTest` as precedent.  All four `ForTest` exports in the
tree are in `platform/`; there is **none** in `domains/`.  The precedent did not say what it was
claimed to say — a domain owns its own business, shared things live in `platform`, and neither learns
that a consumer has tests.  `stats.go`'s `interface{ Cached() (int, int) }` was the pattern already in
`app`, one method wide.

## And the `wires` gate moved the table out of production

Every member of `airReach` was reported **NO WRITER**: the guards check `monitorHasTheAir()` directly
and never consulted the classification.  **A table production does not read is a second carrier of
what the guards already say** — the exact shape this batch exists to remove.  What the table is FOR is
the gate, so it lives with the gate, in `air_boundary_test.go`.

## The closed set

**31 members** — 26 `tty.Config` func seams plus 5 `tty.Radio` methods — each classified with a written
reason, derived by reflection and ratcheting both ways: a new seam with no row fails, and a row naming
a seam that no longer exists fails too.

**Membership is not behaviour**, so every `airMonitor` row owes a behaviour test.  Both instruments
were proved able to fail before being trusted.

## Not in stage A

- **Settings surface-awareness** (HUM LEAD point 3): certain Observer settings do not belong in
  Broadcaster and vice-versa.  D-18 already rules the taxonomy (S/O/B/SPLIT, approved 2026-09-09) and
  metric **M4** already targets zero bleed; `setup*.go` has **no reference to Surface at all**.  Stage B.
- **The relay-fault window's data flow** (HUM LEAD point 1): transparent to the operator — the window
  still appears and the choice still works — but on the console it must be the STATION's relays.  Stage C.
- **F-102**: voice preview is now unavailable while the console holds the air, and voices are a SHARED
  setting (D-18 row 7).  A cue/PFL bus is the real answer and Watchpost has one audio out.  HUM LEAD ruling.
