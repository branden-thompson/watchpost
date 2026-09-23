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
| Brief | `01-objectives/project-brief.md` — APPROVED (D-17); it is the body of issue #22 (D-19) |
| Problem & metrics | `01-objectives/problem-statement.md` — M1–M6, anti-solution hardened |
| Rulings | `02-analysis/rulings.md` — **every ruling lands here the moment it is made** |
| Paired release | go-tuiMaps v0.2.0: loops, the MRMS table, a host view bound (HR-1..HR-5) |
| Inherited | `06_docs/handoff-0.18.0.md` — its §6 step 3 is **wrong** about the reachability gate (C-1) |
| Follow-ups | `06_docs/follow-ups.md` only. F-174 is the Broadcaster's map (0.19.0) |

## The rules that cost something

1. **Rulings one at a time**, with evidence, options, a recommendation and the strongest
   counter-argument, recorded verbatim. Silence is not consent.
2. **Never wider than the region the selected location sits in** (D-8, **D-28**) — the contiguous US,
   Alaska, Hawaii, a territory, a marine area. No global or continental frame, ever, and no single
   US-centred rectangle either: that version excluded Anchorage and San Juan.
3. **Radar is a loop** (D-10), from IEM and MRMS both (D-9), from a **closed list** of sources
   (D-31): no free-form address.
4. **Nothing is fetched before the listener asks for a map** (D-21, **D-25**) — basemap, radar **and
   NWS zone geometry**. The code still seeds zones at start-up; gating it is a BUILD task.
5. **A text description ships in phase 1** (D-26): where the picture cannot be drawn or read, the
   window says in words what covers this place, how far and which way. It is the only path a screen
   reader, an `--ascii` listener and a font without braille have.
6. **Every requirement carries its phase** (D-33). Phase 1 is drawing and access; phase 2 is pan,
   zoom, the map from Details, layers and size tiers.
7. **No code in PLAN** (D-13) — signatures and API shape only.
8. **A test of a thing is not a test of its wiring** (handoff §4.1). Wire from the composition
   root, and make the reachability gate see it.
9. **Read what reads your output** (handoff §4.2) — open the consumer's source.
10. **A gate is obeyed or ruled on**, never argued around. Both CI platforms green before merge.
11. **Blind red team**, dispatched from `06_docs/red-team-brief.md`, never from memory.
12. **No AI attribution** in commits, PRs or tracked files.
13. **`make` must be GNU Make ≥ 3.82.** On macOS, `/opt/homebrew/opt/make/libexec/gnubin` first on
    PATH, or the gate oracle reports COULD NOT RUN.
14. **The listener chooses; the builders make choice cheap (D-23).** Default to a Settings option unless
    a hard constraint (a source's request limit) forbids it; warn when a choice costs a lot of network,
    never silently cap. Optimise and structure the code so choice is cheap to add. **Never architect
    into a corner** — extensibility is a first-class requirement. Interactive toggles live in Settings;
    flags only on one-shot CLI commands (D-22).
