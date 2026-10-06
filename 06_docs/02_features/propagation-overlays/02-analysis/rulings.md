---
title: "0.19.0 Propagation overlays — HUM LEAD rulings"
date: 2026-10-05
phase: DISCOVER (intake)
sev: SEV-0
authority: HUM LEAD
status: "LIVE — every ruling is written here the moment it is made."
---

# 0.19.0 — rulings

Recorded verbatim. A ruling's disposition says what it binds; a correction to a ruling is a new row,
never an edit of an old one. Rulings of 0.18.0 are cited as "0.18.0 D-n"
(`../../observer-maps/02-analysis/rulings.md`).

This log covers the paired train: watchpost 0.19.0, go-giro-data (new) and go-tuiMaps v0.3.0. Each of
the other two keeps its own log for its own decisions; a decision that binds more than one is written
here and cited from theirs.

| # | Date | Question | HUM LEAD verbatim | Ruling |
|---|---|---|---|---|
| D-1 | 2026-10-05 | INIT | "INIT NEW MAJOR DUAL PROJECTS \| 'watchpost-0.19.0' + 'go-giro-data' / LEVEL-1; SEV-0; FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST / Summary / Intent: - Address 0.19.0 items for watchpost + address open gitHub bugs - Implement MUF data as an overlay in watchpost (may require updates to go-tuimaps as well) - create a GO port of prop.kc2g.com to create the tuimaps compatible overlays / Requirements: - derived from discover work already done as part of watchpost 0.18.0 - need to be able to display MUF and potentially fOF2 as overlays in watchpost's MAP modal / Technical constraints: - Needs to be in GO to integrate with watchpost and go-tuimaps - go-giro-data needs to be a github repo under branden-thompson (personal account) / refine problem statement, collect-brief, and engage in prediscovery and appoprirate git initialization protocols when needed" | LEVEL-1, SEV-0, the full directive set. Seeded by the 0.18.0 research (`../00-research/rcc-muf-overlays.md`, 0.18.0 D-238). The reference implementation, `arodland/prop`, carries no licence: "port" is read as a reimplementation from the published science, never a copy of its code (to be ruled in DISCOVER). |
| D-2 | 2026-10-05 | go-giro-data: visibility and licence | "Public, MIT" | `branden-thompson/go-giro-data`, public, MIT for the code (as go-tuiMaps). Against the recommendation (Apache-2.0). GIRO's data terms (CC BY-NC-SA 4.0) and the IRI / PyIRI material's licences are DISCOVER findings; no GIRO data is committed until they are ruled. |
| D-3 | 2026-10-05 | How go-tuiMaps' changes run | "Third paired release v0.3.0 (Recommended)" | go-tuiMaps v0.3.0 is a paired release, run as v0.2.0 was with 0.18.0 (0.18.0 D-11): its own branch, rulings and gates; watchpost pins its release candidates in BUILD and ships on its final tag. |
| D-4 | 2026-10-05 | 0.19.0's scope and order | "MUF first, then accessibility" | **Amends 0.18.0 D-256** (accessibility before any new layer) and REFLECT L4's control: in 0.19.0 MUF leads and the accessibility carry-over (F-191 to F-194, F-196 to F-198, F-180, F-205) follows. Against the recommendation. The issues' order (#18, #12; verifying #24, #23, #9) and whether MUF's own words-path ships with the layer are DISCOVER questions. |
| D-5 | 2026-10-05 | Branch names | "As recommended" | watchpost `feature/propagation-overlays`, cut from `main` (31038ebf) so the release's base is its fork point, the 0.18.0 REFLECT record carried from `feature/map-drawing`; squash-merged `release/0.19.0` at SHIP. go-giro-data: `main` holds the scaffold only; DISCOVER on `feature/discover`. |
