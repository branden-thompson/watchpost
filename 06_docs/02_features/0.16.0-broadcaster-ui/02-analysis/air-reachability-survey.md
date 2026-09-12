---
title: "0.16.0 — Air-reachability survey (FR-2.1a)"
date: 2026-09-12
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "Complete. 3 live defects, 4 latent, 12 owe nothing."
---

# Air-reachability survey

**FR-2.1a: size the population before generalising a mechanism over it.**  The 0.15.0 survey turned
"49 default arms" into "the population is FOUR"; the same discipline applies here before a gate is
built over "everything that can reach the air".

**The ruling this serves** — HUM LEAD, 2026-09-12:

> "masterControl is the one who determines who gets the air … In Broadcaster Mode, Broadcaster ALWAYS
> gets the air, so nothing from Observer should ever be able to re-tune, take over, or 'sneak under'
> to get on the air."

---

## The exclusion rule, stated before counting

**An air-MUTATING call is one that can change what is audible.**  Non-test, `app/` only — the deck is
the only thing in the tree that holds the engine.

**Excluded, and why:**

- **Queries** — `Status`, `Samples`, `Rate`, `Cached`, `Err`.  They report; they do not change.
- **Registration** — `OnSilence`, `OnClipSpent`, `Trace`, `SetResolver`, `SetHandoffLine`.  Applied at
  the next render by whatever is already running.
- **`pipelines.go`'s `s.Start(ctx)`** — three hits, all DATA pipelines, not the engine.  Caught by
  reading them; a keyword survey that kept them would have reported 22.

Without the rule the count is **22**.  With it, **19**.

## Population

**19 air-mutating functions**, over the engine's 14 mutators and `synth.Source`'s 3.

## Classification

| Bucket | Count | Meaning |
|---|---|---|
| **1 — ungated, monitor-initiated, REACHABLE FROM THE CONSOLE** | **3** | Live defects.  Observer can disturb the station's air today |
| **2 — ungated, monitor-initiated, not reachable from the console today** | **4** | Latent.  Unreachable by ROUTING ACCIDENT, not by rule — and the ruling is a rule |
| **3 — programme- or arbiter-initiated** | 9 | Must reach the air.  No gate owed |
| **4 — already gated** | 1 | `needsRead` asks `monitorHasTheAir()` |
| **5 — device-level, deliberately shared** | 1 | `SetVolume`: the Router mirrors gain to both surfaces by design |
| **6 — host fact, not an operator action** | 1 | `softChanged`: discovery/install landed; "nobody is waiting for it" |

**The headline: the population that needs a DECISION is SEVEN, not nineteen — and only THREE are
live.**

---

## Bucket 1 — the three live defects

Every one arrives the same way: **`[s]` Settings is forwarded to Observer from the console**
(`router.go`, `actSettings`), and `setup*.go` contains **no reference to Surface at all** — the window
has zero surface awareness, so every Observer row is editable while looking at the console.

| Function | Op | Route | Consequence |
|---|---|---|---|
| `radio_queue.go` `SetRepeat` | `src.Loop` | Settings → relay dwell → `setRelayDwell` → `SetRepeat` | **MEASURED**: with `RepeatOne`, the source loops.  Per BD-9 `d.source` during a main-track read IS the Broadcaster's card — so the card on the air reads for ever and the line-up never advances |
| `cast.go` `setCast` | `src.Recast` | Settings → cast save → `SetCast` → `reloadCast` | A HARD recast of the card on the air.  Its own comment: *"the listener is waiting to hear it"* — but on the console the listener is the AUDIENCE |
| `voices.go` `PreviewVoice` | `engine.Audition` | Settings → voice preview | A sample mixed over the station's output.  Its comment says *"never the narration's line in flight"*; it says nothing about the programme |

**`SetTones` is the counter-example already in the tree**, and it is the standard to write the fix
against: *"deliberately NOT a recast: no config reload, no re-resolution, no install pass — `[M]` must
be instant and must not disturb a broadcast in flight."*  The decision exists; it was made once, for
one setting.

## Bucket 2 — the four latent

| Function | Op | Monitor route | Why it does not reach the console today |
|---|---|---|---|
| `radio.go` `tune` | `engine.Start` | `SetMode` (`[m]`) | `[m]` is not forwarded |
| `radio.go` `tuneCallsign` | `engine.Start` | relay-fault window → `TuneRelay` | The window IS console-reachable (MVS-D-76).  **Needs a ruling**: it picks from the MONITOR's silent-relay candidates, and on the console the station's bed is `stepBedRelay`'s |
| `voices.go` `SetVoice` | `src.Recast` | the voice chooser | Not forwarded |
| `radio.go` `Stop` | `engine.Halt` | Observer's stop key | Not forwarded (and `silenceMonitor`'s own call is correct) |

## Bucket 3 — the nine that must reach the air

`readCard`, `play`, `pause`, `resume`, `discard`, `restore`, `duck`, `tone`, `stop`.

The Reader and the arbiter's mechanics, driven by the Director's effects and gated upstream by
`advances(MainTrack)` / the rail's exemption.  **No UI reaches them directly.**

## Bucket 4 — the one already gated

`startSynth`, via `needsRead`, which asks `monitorHasTheAir()`.

**And its comment is now stale**: `radio.go:540` says the guard inside `startSynth` stays because *"it
has its own callers"*.  It has exactly one — `needsRead`, line 589.  Recorded, not fixed here.

---

## The structural finding, and it decides where the gate goes

**Two of the nineteen serve BOTH initiators, so the guard cannot live on the method:**

- **`tune`** is called by `SetMode` (the monitor's) **and** by `tuneTo` (the Director's `Tune` effect,
  already gated upstream by `advancesMonitor()` in `advanceBed`/`onEnded`).
- **`tuneCallsign`** is called by Observer's `tuneRelay` **and** by the console's own `stepBedRelay`.

A guard inside `tune` would break the Director's bed rotation.  A guard inside `tuneCallsign` would
break the console's own relay selector — the control fixed yesterday in D-90.

**So the gate belongs at the CALLER**, which is exactly where `needsRead` already puts it.  That is not
a new pattern to invent; it is the existing one, applied to the six other callers that owe it.

**And the closed set to enumerate is the CALLERS, not the methods** — concretely, the `tty.Config`
hook set, which is the whole boundary between a surface and the app.  Every field classified
*guarded / cannot reach the air / console-owned / device-level*, with a written reason, and a new hook
failing the gate until it is classified.

---

## What this survey does NOT settle

- **`tuneCallsign` from the relay-fault window while the console holds the air.**  The window is the
  console's own fault path, but its candidate list is the monitor's.  HUM LEAD ruling.
- **`SetVolume`.**  Shared deliberately today (the Router mirrors gain).  Recorded as bucket 5 rather
  than assumed correct.
- **Whether Settings should be surface-aware at all**, which is D-18's build state and is wider than
  this survey — see F-87 and the M4 "settings bleed" metric, target 0.

## Reproducing this

```
# population
grep -n '(engine|src|source)\.(Start|Fail|Halt|Volume|Preview|Audition|PreviewAside|PausePreview|
  ResumePreview|DropHeld|Suppress|Restore|StartSource|StopPreview|Invalidate|Recast|Loop)\('  app/*.go
# minus _test.go, minus pipelines.go's data-pipeline Start
```

The `SetRepeat` consequence was **measured**, not read: a probe set a bare `synth.Source` as the
deck's, called `SetRepeat(RepeatOne, nil)`, and reported `repeating = true`.
