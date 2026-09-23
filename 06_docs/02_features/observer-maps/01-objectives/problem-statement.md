---
title: "0.18.0 — Observer maps — PROBLEM STATEMENT"
date: 2026-09-22
phase: DISCOVER (RCC)
sev: SEV-0
authority: HUM LEAD
status: "LOCKED 2026-09-22 (D-6).  Metrics adopted (D-18) and amended after the DISCOVER-exit red team (D-29): M1 has a named grader and a recorded protocol; M5's target is set at PLAN."
---

# Problem statement — LOCKED

## 1. The input, recorded exactly as given

The release was framed by its handoff and by the HUM LEAD at intake:

> *"0.17.0 made hazard data map-ready and deliberately drew nothing. 0.18.0 draws."*
> — `06_docs/handoff-0.18.0.md`

> *"We'll get the maps integrated and 'drawing' -> Then we'll build out how the maps actually
> 'work' from a user experience perspective in watchpost (Modals, when a user can hit a map specific
> to the location they're looking at in details, etc)"* — HUM LEAD, 2026-09-22 (D-3)

## 2. Evaluation of the input as written

```
PROBLEM STATEMENT EVALUATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  "0.18.0 draws."  /  "get the maps integrated and drawing"

  [✗] Bad Outcome       — Describes work to do, not a deficiency suffered
  [✗] Affected Humans   — Nobody is named; "a user" appears only in the UX half
  [✗] Tech Agnostic     — Names maps, modals, a details window
  [✗] Non-prescriptive  — The map is the solution
  [✗] Verifiable        — Nothing a third party could observe as true or false

  Score: 0 / 5

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**The expected shape for a planned release, not a criticism.** A handoff names the next work. The
refinement exists so scope is argued against a bad outcome rather than a feature list.

## 3. Diagnosis

| Criterion | What is missing | The question it raises |
|---|---|---|
| Bad Outcome | What goes wrong today without a map | What does a listener fail to judge now? |
| Affected Humans | Who suffers | The person watching the weather in the Observer. The Broadcaster's operator is F-174 |
| Tech Agnostic | Strip "map" and "modal" | What remains is *seeing where a hazard is relative to a place* |
| Non-prescriptive | The map is prescribed | What bad thing persists if no map is built? Judging distance from text |
| Verifiable | No observable condition | Can a stranger tell, from what is shown, whether an alert reaches a place? |

## 4. The locked problem statement

> **"A person watching the weather in Watchpost is told that an alert applies to a place, but cannot
> see where that alert actually is: whether its area covers them, stops short of them, or lies to one
> side, so they judge how close the danger is from zone names and distances they have to picture for
> themselves."**

**LOCKED by the HUM LEAD, 2026-09-22** — *"Problem Statement A approved"* (D-6).

```
PROBLEM STATEMENT EVALUATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [✓] Bad Outcome       — Danger judged from names and imagined distances
  [✓] Affected Humans   — The person watching the weather
  [~] Tech Agnostic     — Names the product; names no technology
  [✓] Non-prescriptive  — "cannot see where" — a map is one answer, not the stated one
  [✓] Verifiable        — covers / stops short / lies to one side: three observable answers

  Score: 4.5 / 5

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**The half point is kept, not polished away.** "In Watchpost" scopes the sufferer to this
product's listener; it names no technology. Recorded so a later reader does not mistake the score
for an oversight.

**Radar sits under this statement as context, not as its subject.** The HUM LEAD names radar the
most common view (D-7): it answers the same question — where is it, relative to me — for the weather
itself rather than for the warning.

## 5. Measurements of value, anti-solution hardened

Each was tested with: *"Can I imagine a way to satisfy this metric that would NOT solve the stated
problem?"* Where the answer was yes, the metric carries the bound that closes it. **Numeric targets
marked PLAN are set there from measurement**, not guessed here.

| # | Name | Symbol | Type | Definition (with direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Where is it | `WII` | Primary | Share of UAT prompts where the listener correctly says whether an active alert **covers** the selected location, **stops short** of it, or **lies to one side**, **from the picture alone** — the alert's text and zone list hidden. Higher is better; target 100%. **Grader: the HUM LEAD** (D-29). | Listener UAT over the **recorded** scenario set below — repeatable, unlike live weather |
| M2 | Never global | `NG` | Primary | Count of frames drawn wider than the region holding the selected location (D-8, **D-28**). Lower is better; target 0. | Instrument over the test suite and the scripted-PTY journeys |
| M3 | Radar honesty | `RH` | Primary | Count of frames where radar is drawn without its newest frame's age, or older than its source's cadence without a stale mark. Lower is better; target 0. | Automated, over a clock-driven fixture |
| M4 | Partial honesty | `PH` | Primary | Count of alerts that did not fully resolve and are shown — or withheld — **without that fact stated**. Lower is better; target 0. | Automated, over `Area.Missing` fixtures |
| M5 | Time to picture | `TTP` | Primary | Seconds from `g` to the first **complete** frame. **Target PLAN (D-29):** the 1 s warm / 3 s cold figures are go-tuiMaps' own (its v0.1.0 D-30), never measured in this host, and wave 1's parts already sum against them — a 41-zone cold resolve ~2 s, TileJSON ~260 ms plus 50–100 ms a tile, twelve radar requests — with D-21 forbidding any warming. Re-derived at PLAN from those numbers. Lower is better. | Instrument in the host; cold = empty disk cache, network reachable |
| M1b | Where is it, **in words** *(adopted D-36)* | `WIW` | Primary | The same recorded scenarios as M1, **picture hidden and the description shown**, scored the same three ways. M1 hides the text to prove the picture works; M1b hides the picture to prove the description does — the same question asked of the other artifact, because FR-7.4 is the only path three classes of listener have | Listener UAT over the same recorded set (R2 A11y F-1) |
| M6 | Loop smoothness | `LS` | Secondary | The radar loop advances at its configured rate, and Observer input stays responsive while it runs. Frame-interval jitter and key-to-response latency; targets PLAN. Lower is better. | Instrument in the host, Observer fully live |

**The anti-solutions each bound closes:**

| Metric | The anti-solution the bound exists to forbid |
|---|---|
| M1 | A window that lists the alert's zones in text beside the map scores well by being read, not seen — so the text is hidden and the answer must come from the picture. Prompts include **alerts that stop short of the place within a few cells** and **partially resolved alerts**, since those are the cases the problem is about. |
| M2 | Draw nothing, and zero frames are too wide. So M2 never passes alone: M5 requires a complete frame, and a blank frame with data on hand counts against M5. |
| M3 | Hide radar whenever it is old, and no frame is ever drawn stale. So a stale frame must be **shown and marked**, not withheld; radar absent while its source is reachable is itself a failure. |
| M4 | Draw only alerts that fully resolved, and no partial alert is ever drawn without comment. So **silent omission counts as a failure** exactly like silent partial drawing. This keeps M4 neutral on R-4.3: the ruling decides *how* a partial alert is shown, never *whether* it is mentioned. |
| M5 | Show a coarse frame instantly and stop the clock. So the clock stops at the library's `Complete` status, not at the first frame. |
| M6 | Slow the loop to a frame every ten seconds and it is perfectly smooth. So the loop runs at its **configured** rate, and the Observer's own work (publishes, the ticker, key handling) is running during the measurement. |

**M1's protocol (D-29, extended at round-2 remediation — the extensions are marked).** The grader is
the HUM LEAD. The scenarios are **recorded**, not live, so the same run can be repeated by anyone
later: at minimum one alert whose area covers the location, one that stops short of it within a few
cells, one partially resolved (`Area.Missing` non-empty), one in an adjacent county, and *(added at
remediation)* one marine and one **failure** scenario — offline, a failed tile, or stale radar —
because FR-3.4 and FR-7.4 promise honesty there and an all-happy-path protocol never tests it. All
are drawn from captured responses (FR-8.6).

Each is presented with the alert text and zone list hidden, at both 69×12 and a comfortable size, and
scored covers / stops short / lies to one side.

**The subject and the count** *(added at remediation; round 2 found the protocol named a grader but no
subject)*: **at least eight scenarios**, scored in one sitting, by the HUM LEAD. The honest limit is
recorded with it — the grader authored the scenarios and locked the problem, so this is a
self-assessment against a fixed script, not a blind study; its value is repeatability and a stated
bar, not independence.

**FR-7.4's description is scored by M1b**, not by M1: M1's own hardening hides the text and demands
the answer come from the picture, so scoring a text artefact under it would contradict the rule that
makes it worth anything.

**M1b is scored FIRST, or on a disjoint half of the set** — and the choice is recorded with the
result. The grader is the same person, so a scenario whose picture has already been read is one whose
answer is known: scoring the description second would measure memory, not the description. Taking the
description pass first costs nothing, because M1's own answer comes from a picture the grader has not
yet seen in that sitting.

**M1 is the metric this release is most likely to fail**, because most alerts have no polygon of their
own (C-9 says four in five; measured nine in ten on 2026-09-22, marine-inflated and weather-dependent
— W1-C) and are drawn only as far as their zones resolve. M4 is M1's machine-checkable half.

## 6. Status

**LOCKED, 2026-09-22.** Section 4 is the anchor for every DISCOVER output, PLAN decision and scope
argument in 0.18.0. The seven measures in section 5 carry forward to the release PR's Metrics of
Success table and are subject to the DISCOVER-exit red team, which may tighten them.
