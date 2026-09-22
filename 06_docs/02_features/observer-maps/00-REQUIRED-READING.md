---
title: "0.18.0 Observer maps — REQUIRED READING, every session and after every compaction"
date: 2026-09-22
phase: ALL
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY.  Re-read at session start and after any context compaction, for the life of 0.18.0."
---

# Read this before touching 0.18.0

**Why this file exists.** 0.18.0 runs in parallel with go-tuiMaps v0.2.0 (D-11), and the work
started on a new machine whose agent memories did not survive the move. The HUM LEAD's condition for
running two releases at once: **the record is kept current at every step, and every lesson that can
be a failing test becomes one** — *"the rule an agent has to remember is a rule that will eventually
be passed over in a new session."*

## Where things are

| | |
|---|---|
| Branch | `feature/map-drawing` → squash-merged `release/v0.18.0` at SHIP (D-2) |
| Phase | DISCOVER (RCC). Update this row at every phase transition |
| Brief | `01-objectives/project-brief.md` |
| Rulings | `02-analysis/rulings.md` — **every ruling lands here the moment it is made** |
| Paired release | go-tuiMaps v0.2.0: loops, the MRMS table, a host view bound (HR-1..HR-5) |
| Inherited | `06_docs/handoff-0.18.0.md` — its §6 step 3 is **wrong** about the reachability gate (C-1) |
| Follow-ups | `06_docs/follow-ups.md` only. F-174 is the Broadcaster's map (0.19.0) |

## The rules that cost something

1. **Rulings one at a time**, with evidence, options, a recommendation and the strongest
   counter-argument, recorded verbatim. Silence is not consent.
2. **Never wider than the United States** (D-8). No global or continental frame, ever.
3. **Radar is a loop** (D-10), from IEM and MRMS both (D-9).
4. **No code in PLAN** (D-13) — signatures and API shape only.
5. **A test of a thing is not a test of its wiring** (handoff §4.1). Wire from the composition
   root, and make the reachability gate see it.
6. **Read what reads your output** (handoff §4.2) — open the consumer's source.
7. **A gate is obeyed or ruled on**, never argued around. Both CI platforms green before merge.
8. **Blind red team**, dispatched from `06_docs/red-team-brief.md`, never from memory.
9. **No AI attribution** in commits, PRs or tracked files.
10. **`make` must be GNU Make ≥ 3.82.** On macOS, `/opt/homebrew/opt/make/libexec/gnubin` first on
    PATH, or the gate oracle reports COULD NOT RUN.
