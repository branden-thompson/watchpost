---
title: "0.19.6 re-architecting — REQUIRED READING"
date: 2026-11-14
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY for the life of 0.19.6, at session start and after any compaction."
---

# Read this before touching 0.19.6

## What this release is

The re-architecting that 0.19.5 found and ruled out of its own scope (0.19.5 D-40). `modes/tty` and
`app` are split into smaller packages along their seams (D-1). The arbiter's dissolving into the
Director's step may come with it, where it holds to the principles (D-2). Charm is not re-implemented
(D-3).

## Where things are

| | |
|---|---|
| Branch | `feature/re-architecting`, cut from `main` at `e57afeaf` (A-1) |
| Phase | **DISCOVER**, opened 2026-11-14; LEVEL-1, SEV-0, HUMAN LEAD, the full directive set (D-4). The baseline sweep ran on `e57afeaf` before any code moved: 1,051 of 1,058 caught, the 7 survivors the kept equivalents (A-3) |
| Rulings | `02-analysis/rulings.md`, every ruling the moment it is made; minor ones as A-n rows for veto |
| Survey | nine lens agents over one computed dependency graph (D-6, A-4, A-5): `02-analysis/lens-*.md`, reconciled by the agent into one package map (D-7): `02-analysis/package-map.md`, the settled document every later finding is judged against |
| Previous release | 0.19.5, shipped; its REFLECT is `../0.19.5-quality-pass/08-reports/reflect-report.md` (0.19.5 D-60) |
| Follow-ups | `06_docs/follow-ups.md` only |

## The rules that cost something

1. **D-1's four principles bind every change:** performance stays the same or improves; the
   architecture becomes more maintainable, traceable and understandable; best practice holds, P10
   included; no behaviour or function regresses.
2. **Charm, Bubble Tea and Lip Gloss are used through their interfaces, never re-implemented**, so
   either can be upgraded as released (D-3). Their composite's cost stands (0.19.0 D-327) unless an
   improvement by orders of magnitude is shown to the HUM LEAD.
3. **The shared deck is not split.** One deck, the Director arbitrating in both modes, MasterControl
   gating the air (`../0.16.0-broadcaster-ui/00-REQUIRED-READING.md`). A package split moves the deck
   whole; D-2's work follows the role model and the disposition sheet, item by item.
4. **Every move keeps its mutants.** Before code moves, `06_docs/mutants/` is searched for its anchors;
   a moved anchor is re-pointed and re-vetted, and the closing sweep is judged against A-3's.
5. **The live app is not run for measurement** (the Open-Meteo quota). Performance is measured from
   benchmarks, allocation pins and profiles of tests.
6. **Rulings one at a time**, with evidence (`file:line`), options, a recommendation and the strongest
   counter-argument, recorded verbatim.
