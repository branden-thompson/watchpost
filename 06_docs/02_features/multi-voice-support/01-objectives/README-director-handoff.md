# Director / ticker redesign — start here

**Written 2026-09-01 for a session that has none of the conversation.** Every symbol and path below
was verified to exist at commit `bf1f266`. If one has moved, trust the code and fix this file.

**Status as of 2026-09-01: DISCOVER APPROVED · PLAN COMPLETE, awaiting approval · NOT BUILT.**
0.14.0 held for it under MVS-D-56; **MVS-D-79 (HUM LEAD 2026-09-05) DEFERS THE SETTING to post-Broadcaster and releases that condition.** The ORDERING ships in 0.14.0 — only the listener-facing rearrangement is deferred. There is no schedule pressure; the requirement is
that it *reliably works*.

**The RCC and PLAN this file was written to brief are now done.** Start from the current artifacts:

| Read | For |
|---|---|
| `lineup-model.md` | **The rulings** — L-, S-, T-, R-, Q- series with verbatim context. The single most current document |
| `director-requirements.md` | DR-1 … DR-24, each naming how it is verified |
| `../03-architecture-design/director-architecture.md` | **Approach C, approved**, and the two rejected alternatives |
| `../03-architecture-design/plan-decisions.md` | PD-1 … PD-6, all resolved |
| `../03-architecture-design/director-build-plan.md` | The five build phases and the test strategy |
| `../08-reports/discover-report-director.md` · `red-team-plan-director.md` | The phase records and both critical analyses |

Everything below is the **original briefing**, kept because its "what you must not do" table is still
current and hard-won.

## Read these three, in this order

1. **`director-charter.md`** — what the Director is *for*. Start here even if you only care about the
   ticker. It is the HUM LEAD's own words, and the design is judged against it.
2. **`read-order-design.md`** — the read lineup, one of the Director's responsibilities: the default
   weighted order, the ALERTS - READ ORDER settings group, and gaps G-1…G-11.
3. **`../04-development/p5-build-log.md` §2** — the rulings MVS-D-54 … MVS-D-59, each naming what it
   amends.

Then `06_docs/follow-ups.md` for F-16, F-17, F-18 and the open decision D-1.

## The one-paragraph version

Watchpost reads weather aloud and shows a scrolling ticker. Today, *what gets read* and *what the
band shows* are decided by different code that agrees by coincidence. Two red-team rounds found the
same class of defect — a bound in one path silencing a hazard the other believed was being read —
and each fix moved the failure boundary instead of removing it. The HUM LEAD's conclusion: the
ordering is not a constant to be tuned, it is a **preference the listener sets**, and the **Director**
should own the schedule that both the audio and the ticker follow.

## What you must not do

These are not opinions. Each was tried and reverted, and the record is in
`08-reports/red-team-build-round2.md`.

| Don't | Why |
|---|---|
| **Tune `maxBreaking` / `breakingCap` / `capBurst`** | Three attempts, three moved boundaries. The last measured a starvation edge at ~8.14 s per read: past it a quiet lane goes unspoken *permanently* under sustained arrivals. MVS-D-56 replaces the arrangement rather than tuning it a fourth time. |
| **"Fix" the ticker's parked-offset behaviour** | F-16. Fixed twice, reverted twice — one attempt left 12 % of a busy lane's tape ever reachable, the other 47 %. The note says model the ticker's geometry and rotation **first**. |
| **Assume a passing test means covered** | Roughly thirty remediation defects this release passed every gate. Use `06_docs/mutants/run.sh`. |
| **Decide what the listener hears** | Read order, counts, pacing and precedence are HUM LEAD rulings, even mid-defect-fix. Present the fork with evidence; do not choose. |

## Two facts about the code, established by reading it

**1. There are two independent paths to audio, and neither is a schedule.**

- *Broadcast path*: `app/radio.go:radioDeck` → `engine.Start` (relay) or `engine.StartSource`
  (synthesised location cycle), driven by `app/radio_queue.go:tune`, `:advanceQueue`, `:armDwell`.
- *Narration path*: `app/director.go:director.Run` → `speaker`. **Exactly two callers**:
  `app/severe_read.go` (the `[space]` read) and `app/ticker.go` (the breaking takeover).

They never meet as a queue. They meet at the **volume**, inside the engine, via `Suppress`/`Restore`.
That is the structural cause of the defect class.

**2. Today's `director` is a voice ARBITER, not a schedule owner.** It holds `onAir`, `suspended`,
`waiting`, `ducked` and decides who gets the air when two things want it. Nothing in it plans.

## The map — where the behaviour lives now

| Concern | Where | D-C-7 disposition |
|---|---|---|
| Watchlist rotation and dwell | `app/radio_queue.go:advanceQueue`, `:armDwell` | **Absorb** — this is the schedule, for one item kind |
| Which alerts are announced, and how many | `app/ticker.go:startTakeover`, `:capBurst` | **Absorb** — lineup selection |
| Who gets the air; preemption, duck | `app/director.go:director.Run` | **Absorb** — the air-control half |
| PLAY / PAUSE / STOP | `app/radio.go:Stop`, `:Tune`, `radio_queue.go:SetRepeat`, the panel keys | **Relay through** the Director |
| Relay vs synth, and fallback | `app/radio.go:tune`, `:SetMode`, `:followMount`, `:startSynth` | **Instruct** — but a dead relay **reports UP**; that is a schedule change |
| How the broadcast yields to an alert | `domains/radio/player/engine.go:Suppress`, `:Restore`, `:giveWayLocked` | **Instruct** — already the right shape |
| The band's contents and rotation | `modes/tty/ticker.go:setTicker`, `:advanceTicker`, `:advanceTickerCategory` | **Instruct** — a cue with lead time replaces today's derivation |
| The window index and the tape's lane rows | `app/severe.go:publish`, `:laneRowsPerTab` | **Instruct** — data assembly |
| Fetch, scope, what is new | `app/ticker.go:cycle`, `:seenStore` | **Instruct** |
| Scripts, pronunciation, voices | `domains/radio/script`, `:pronounce`, `:synth`, `app/cast.go:resolveVoice` | **Leave alone** — outside the charter |
| Classification and bounds | `domains/severe:Classify`, `:Sort`, `:Cap`; `domains/globalfeed:Merge`, `:capPerLane` | **Leave alone** |

## The open questions — answer by design, not at the keyboard

- **G-1 … G-11** in `read-order-design.md` — the lineup's own gaps. G-11 (what "the ticker is synced
  with the reads" precisely means) is a design question, not an implementation detail.
- **D-C-1 … D-C-7** in `director-charter.md` — the charter's. D-C-7 is answered above; D-C-1 (does
  the Director own the location-report rotation too, making one queue of reports *and* alerts) is the
  one that most changes the shape.
- **D-1** in `follow-ups.md` — a zone-only alert outside a tracked area. Needs a **count from the
  live feed**, not an estimate.

## Constraints that survive the redesign

- **No ticker lane for Forecasts** (MVS-D-59). The marquee is for what is happening.
- **The lineup default** is "most serious, closest first"; app default location **Bonsall, CA**;
  burst of **5**, settable to ALL; overflow **diverted** with a spoken pointer to `[w]`.
- **Discard must be cheap and safe** — anything prepared ahead may be thrown away, so preparation
  must have no side effects that outlive it.
- **Voices are station-local**; schedule against what this machine has.
- **The AA register is hand-maintained** (F-18) — a new colour token absent from
  `platform/render/contrast.go:aaPairs` is never measured. Register anything you add.

## How to work here

`06_docs/remediation-review-loop.md` is the process, and it was derived the hard way. In short:
failing test first through the real entry point, watch it fail, fix, **mutate your own change** with
`06_docs/mutants/run.sh`, then a fresh adversarial reviewer who must run something. Assert a fixture
is *valid* before asserting behaviour — five vacuous fixtures this release passed while proving
nothing.

Gates: `make verify`, `make alloc-budget`, `make pty-severe`, and
`make p10 A2DH=<framework build>` (the PATH `a2dh` is a
stale thin install without `p10`).

## Still owed by the HUM LEAD before SHIP

M3's two listening trials — which also settle the read-length assumption this design depends on — the
Linux validation protocol, F-6's M2 journey re-run, and README screenshots (all currently 0.13.0 with
alt text asserting 0.14.0 content).
