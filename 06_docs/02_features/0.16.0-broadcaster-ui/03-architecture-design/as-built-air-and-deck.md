---
title: "0.16.0 BUILD — the air, the deck and the guard boundary, AS BUILT"
date: 2026-09-12
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "Durable reference.  Redrawn after D-91 stage A at the HUM LEAD's request; supersedes diagrams.md for the audio path only."
---

# How this actually works now

**`diagrams.md` is the PLAN's picture (2026-09-09) and stays as the record of what was intended.**
This is the audio path as BUILT, after D-74, D-78, D-90 and D-91.

**Why it exists:** the shared-deck decision has been re-litigated across sessions because the model
was rebuilt from `app/radio.go` instead of read from the rulings.  A picture is harder to
misremember than a call graph.

---

## 1. One deck, one air

**Both surfaces run through the SAME deck, on purpose** (HUM LEAD, 2026-09-12): both do synth reads
and relay streams, so two decks would be duplicated code and two things that could play at once.
The Director arbitrates the rotation in BOTH modes; MasterControl gates the deck from the air.

```mermaid
flowchart TB
  OBS["OBSERVER<br/>the operator's own listening"]
  BC["BROADCASTER<br/>the station's line-up"]
  RTR["Router<br/>routes keys, mirrors gain,<br/>owns the surface swap"]

  OBS --> RTR
  BC --> RTR

  RTR -->|"OnSurface → takeTheAir"| MC
  MC["MasterControl<br/><b>declares the air</b><br/>silences the bed<br/>silences the programme"]

  MC -->|"Aired{To}"| DIR
  DIR["Director<br/><b>arbitrates BOTH rotations</b><br/>holds the schedule + the dwell"]

  DIR -->|effects| EXEC["executors<br/>the Reader"]
  EXEC --> DECK
  RTR -.->|"tty.Radio + tty.Config<br/>(the guarded boundary)"| DECK

  DECK["radioDeck<br/><b>ONE deck</b><br/>synth reads · relay streams"]
  DECK --> ENG["player.Engine<br/><b>ONE audio out</b>"]
  ENG --> OUT(["a human patches this<br/>into a transmitter"])

  style MC fill:#7a4242,color:#fff
  style DECK fill:#42607a,color:#fff
  style ENG fill:#42607a,color:#fff
```

**"ON AIR" never means the antenna is radiating** (FR-5.5).  Watchpost produces audio; a person
patches it.  That is why anything mixed into the output on the console *goes out*.

---

## 2. Who may advance, and it is mutually exclusive by construction

```mermaid
flowchart LR
  subgraph state["what MasterControl holds"]
    P["power<br/>Stopped | Running | OffAir<br/><i>the STATION's</i>"]
    M["monitor<br/>running or not<br/><i>the OPERATOR's</i>"]
    A["air<br/>AirMonitor | AirProgramme<br/><i>declared by the SURFACE</i>"]
  end

  P --> SG
  A --> SG
  A --> MG
  M --> MG

  SG["advances(MainTrack)<br/>power == Running<br/>AND air == AirProgramme<br/>AND !bed.carries"]
  MG["advancesMonitor()<br/>monitor<br/>AND air == AirMonitor<br/>AND !bed.carries"]

  SG -->|true| PROG["the station's line-up advances"]
  MG -->|true| MON["the watchlist rotation advances"]

  RAIL["the ALERT RAIL<br/><b>exempt from both</b><br/>hazards read in either mode<br/>(its FENCE follows the surface, D-73)"]

  style RAIL fill:#7a6b42,color:#fff
```

`TestOnlyOneProgrammeCanEverAdvance` states the exclusivity.  **This is why Observer's watchlist dwell
cannot re-tune the deck while the console holds the air** — F-99 was filed claiming it could, and
retracted when measured.

---

## 3. The surface swap

```mermaid
sequenceDiagram
  participant OP as operator
  participant R as Router
  participant LP as livePipelines
  participant MC as MasterControl
  participant D as Director
  participant DK as radioDeck

  OP->>R: ctrl+b
  R->>LP: OnSurface(SurfaceBroadcaster)
  LP->>LP: owner.set(Broadcaster)
  Note over LP: deck.air now reports false<br/>for the monitor
  LP->>MC: HandAir(AirProgramme)
  MC->>D: Aired{To: AirProgramme, Fence}
  LP->>MC: StopMonitor()
  MC->>D: Monitored{Running: false}
  LP->>LP: ticker.nudgeRescope()
  LP-->>DK: stopMonitor()
  Note over DK: NOT Stop() — that is guarded,<br/>and runs after the air moved (D-91)
```

**Going back does not resume the monitor.**  The operator presses play, "like if Observer was first
opened" — which is also what stops a rapid `ctrl+b`/`ctrl+o` flip from re-resolving relays and
re-fetching products (D-74).

---

## 4. The guard boundary — D-91

**31 seams** between a surface and `app`: 26 `tty.Config` func fields and 5 `tty.Radio` methods.
The classification is a closed set, derived by reflection, ratcheting both ways
(`app/air_boundary_test.go`).

```mermaid
flowchart TB
  SURF["a surface calls in"] --> CLS{"what can it do<br/>to the air?"}

  CLS -->|airNone · 14| NONE["settings, queries, lookups<br/><i>no guard owed</i>"]
  CLS -->|airMonitor · 7| MON{"monitorHasTheAir()"}
  CLS -->|airProgramme · 2| PROG["StepBedRelay · TuneRelay<br/><i>the console's own</i>"]
  CLS -->|airShared · 2| SHR["SetVolume — one volume for the app<br/>InjectAlert — feeds the exempt RAIL"]
  CLS -->|airGatedDownstream · 1| GD["ReadReport → needsRead<br/><i>already asks</i>"]
  CLS -->|airDeclares · 1| DEC["OnSurface — it MOVES the air"]

  MON -->|"true — Observer has it"| DO["reaches the engine"]
  MON -->|"false — the console has it"| REF["<b>refused</b><br/>the SETTING still applies;<br/>only the disturbance stops"]

  style MON fill:#7a4242,color:#fff
  style REF fill:#7a4242,color:#fff
```

**The seven refused:** `Radio.Tune`, `Radio.Stop`, `Radio.SetRepeat`, `Radio.SetMode`, `SetVoice`,
`SetCast`, `PreviewVoice`, `NarrateEvent`/`EndEventRead`.

**The guard is at the CALLER, never the method** — because `tune` serves the monitor AND the
Director's `Tune` effect, and `tuneCallsign` serves Observer's relay pick AND the console's own bed
selector.  The exported `tty.Radio` methods are the monitor's surface; the lower-case internals are
shared.

---

## 5. What a settings change does now

**The pattern is `setTones`', already in the tree:** *"deliberately NOT a recast — `[M]` must be
instant and must not disturb a broadcast in flight."*

```mermaid
flowchart LR
  S["operator saves a setting<br/>from the console"] --> SAVE["the setting is recorded,<br/>validated, re-resolved"]
  SAVE --> Q{"does it disturb<br/>what is playing?"}
  Q -->|no| DONE["applies at once"]
  Q -->|"yes — Loop / Recast"| HELD["applies to the monitor's<br/>NEXT source, not this one"]
  style HELD fill:#42607a,color:#fff
```

---

## 6. The five roles, unchanged

| Role | Owns | Does not own |
|---|---|---|
| **Card Producer** | that a card should exist | what it says |
| **Card Composer** | what it says, at standby | arrangement, the UI |
| **Director** | the order, the operator's will, BOTH rotations | the air |
| **Reader** | performing the LIVE card, with its pacing | the running order |
| **MasterControl** | **the air**; declares ON AIR / STANDBY; silences the bed and the programme | planning; owning the bed |

**MasterControl declares and everyone complies, the Director included** (MVS-D-77).

---

## What this does not yet show

- **Settings surface-awareness** (D-18 / M4) — stage B.  `setup*.go` has no reference to `Surface`
  today, so Observer's rows are visible from the console.
- **The relay-fault window's data flow on the console** — stage C.
- **F-102** — voice preview is refused on the console; a cue/PFL bus is the real answer and there is
  one audio out.
