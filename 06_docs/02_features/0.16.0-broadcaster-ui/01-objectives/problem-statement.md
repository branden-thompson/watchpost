---
title: "0.16.0 Broadcaster UI — problem statement and measures of value"
date: 2026-09-09
phase: pre-DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "LOCKED — HUM LEAD ratified 2026-09-09 (\"Lock; Approved\")."
skill: refine-problem-statement v1.0.0
---

# Problem statement — LOCKED

## 1. The input, recorded exactly as given

> *"With 0.15.0 released, we should have all the foundational pieces in place to implement the
> Broadcaster UI experience.  This is a parallel UI/Dashboard that runs alongside Observer, where the
> two modes can switch back and forth with [ctrl+b] and [ctrl+o], and where Broadcaster has a
> different set of settings and preferences that align it better for FRS/GMRS/CBRS broadcasting
> (higher data request frequency/constrained service area)."*

This is the SUMMARY/INTENT.  It is not a problem statement, and it does not have to be — but the
framework requires one before DISCOVER, because every requirement below should trace back to a bad
outcome rather than to a feature list.

## 2. Evaluation of the input as written

```
PROBLEM STATEMENT EVALUATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  "We should have all the foundational pieces in place to implement
   the Broadcaster UI experience."

  [✗] Bad Outcome       — Describes work to start, not a deficiency suffered
  [~] Affected Humans   — "we"; the operator and the listener appear later, not here
  [✗] Tech Agnostic     — Names the UI, the modes, and two key bindings
  [✗] Non-prescriptive  — Prescribes the entire solution
  [✗] Verifiable        — Nothing here a third party could observe as true or false

  Score: 0.5 / 5

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**This is the expected shape for a planned release and is not a criticism of the input.**  Issue #10
was written as a work item, and the intent section of a brief is allowed to name the solution.  The
refinement below exists so that DISCOVER can test scope against a bad outcome instead of against a
feature list, which is the only way a descope decision can be argued later.

## 3. Diagnosis

| Criterion | What is missing | The question it raises |
|---|---|---|
| Bad Outcome | The statement says what will be built, not what goes wrong today | What does a person running a station over the air actually fail at right now? |
| Affected Humans | "We" is the builder, not the sufferer | Issue #10 already names two: the **human operator** and the **human radio listener**.  Which one is primary? |
| Tech Agnostic | Names Observer, Broadcaster, ctrl+b, ctrl+o | Strip those: what remains? |
| Non-prescriptive | The card stack, the two settings sets and the mode switch are all solutions | What bad thing persists if none of them is built? |
| Verifiable | No observable condition | What would a stranger watch an operator do, and see fail? |

## 4. The locked problem statement

**Ratified by the HUM LEAD, 2026-09-09, verbatim: *"Lock; Approved"*.**

> **"A person broadcasting hyper-local weather over a two-way radio channel cannot see what is about
> to be transmitted, cannot change its order before it goes out, and cannot confirm at a glance
> whether they are currently on the air — so they operate the station blind, and their listeners
> hear the wrong thing at the wrong time."**

```
REFINED PROBLEM STATEMENT — LOCKED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [✓] Bad Outcome       — Three named inabilities, each a present deficiency
  [✓] Affected Humans   — The station operator (primary); the radio listener (downstream)
  [✓] Tech Agnostic     — No UI, no key binding, no package named; "two-way radio
                          channel" is a domain concept, as "build" is in the reference examples
  [✓] Non-prescriptive  — A card stack is ONE answer; a printed running order, a
                          spoken confirmation or an audible cue would also address it
  [✓] Verifiable        — Sit a stranger beside an operator mid-broadcast and ask
                          them to name what airs next and to stop it.  Observe the failure.

  Score: 5 / 5

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Why "cannot confirm at a glance whether they are on the air" belongs in the statement.**  It is the
one inability with a safety edge: a station believed silent that is transmitting, or believed live
that is dead, is the failure mode that reaches a listener as absence during severe weather.  It also
covers the mode-switch requirement without prescribing how switching works.

**What this statement deliberately does NOT cover**, so scope arguments stay honest:

- It does not carry the settings split.  That is a **constraint on not making things worse**, not a
  bad outcome a listener suffers.  It is a requirement, and it is not in the problem statement.
- It does not carry the higher data-request frequency or the constrained service area.  Those are
  *how* a broadcast-shaped configuration differs, and they belong to DISCOVER.

## 5. Measurements of value, anti-solution hardened

Every measurement below was tested with the question the skill requires: *"Can I imagine a way to
satisfy this metric that would NOT solve the stated problem?"*  Where the answer was yes, the
measurement carries the bound that closes it.

| # | Name | Symbol | Type | Definition (with direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Next-item certainty | `NIC` | Primary | Share of UAT prompts where the operator names the next item to air, and the one after it, **without waiting for either to air**.  Higher is better; target 100%. | Operator UAT, scripted prompts |
| M2 | Time to correct the running order | `TCO` | Primary | Seconds from the operator deciding an item is wrong to that item being reordered or dropped, **with audio output never interrupted**.  Lower is better. | Operator UAT, wall clock |
| M3 | Unsafe mode switches | `UMS` | Primary | Count of Observer/Broadcaster switches that produce a truncated word, dead air, or an audio path left running with no owner, **across a run that includes at least ten switches taken deliberately mid-utterance**.  Lower is better; target 0. | Scripted PTY plus operator UAT |
| M4 | Settings bleed | `SB` | Secondary | Count of shared settings that fail to survive a switch, plus per-UI settings observable in the other UI.  Lower is better; target 0. | Automated round-trip test |
| M5 | Silent overflow | `SO` | Secondary | Count of terminal sizes at which Broadcaster renders past the terminal bounds with no notice.  Lower is better; target 0.  Directly inherits F-55. | Automated size sweep |

**The anti-solutions each bound closes:**

| Metric | The anti-solution the bound exists to forbid |
|---|---|
| M1 | A scrolling text log that technically shows what is next while the operator can do nothing about it — so M1 is paired with M2 and neither passes alone. |
| M2 | Quit and relaunch the application.  Fast, and it takes the station off the air, so the bound is "audio never interrupted". |
| M3 | Forbid switching, or force a long confirmation on every switch.  Zero unsafe switches and a UI nobody can leave — so the bound mandates ten deliberate mid-utterance switches. |
| M4 | Give each UI a wholly separate config, which trivially prevents bleed and breaks the shared-theme requirement — so the metric counts failures in BOTH directions. |
| M5 | Refuse to run below some size, reporting zero overflows because it renders nothing.  A clear refusal notice IS an acceptable answer here, which is why the metric counts silent overflow rather than non-render. |

**M3 is the metric this release is most likely to fail**, and it is stated first among the safety
items on purpose: the mode switch is the only requirement in the brief that can leave a physical
transmitter keyed with no software owner.

## 6. Status

**LOCKED, 2026-09-09.**  Section 4 is now the anchor for every DISCOVER output, PLAN decision and
scope argument in 0.16.0.  The five measures in section 5 carry forward to the release PR's Metrics
of Success table.

**Changing it requires a HUM LEAD amendment**, recorded here with its date and reason.  A scope
argument that cannot be traced to this sentence is out of scope by definition.
