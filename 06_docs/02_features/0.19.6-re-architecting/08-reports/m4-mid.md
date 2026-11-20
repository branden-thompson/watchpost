---
title: "0.19.6 M4 — the blind trace's midpoint, on 6060309b"
date: 2026-11-20
phase: BUILD
sev: SEV-0
status: "Graded: 0 of 4 pass on 6060309b, as before; on the sites and the routes, 2 of 4 (T3, T4), from 0. The midpoint D-29 sets between the before and REVIEW"
---

# M4's midpoint: the blind trace on `6060309b`

**What this is.** D-29's midpoint: the four tasks of `m4-before.md`, traced again after B1's doc fixes
on the layout B0 left unchanged, so that REVIEW's delta from here is the split's and this one's from the
before is the docs'.

- **The tasks and the keys** are the before's (`m4-key/`), unchanged.
- **The reviewer.** A fresh agent, given the before's brief word for word (its title alone dropped the
  revision), traced from README's "Start here" on a `git archive` of `6060309b` with
  `06_docs/02_features/0.19.6-re-architecting/` removed. The copy names no key and no scratch path; only
  the record's index files (`06_docs/ids.md`, `06_docs/follow-ups.md`), two of `cmd/watchpost`'s tests and
  the graph tool name the removed folder, so a 0.19.6 ID resolves to a row whose file the reviewer cannot open. None of the
  four tasks met one. Its answers are in `m4-mid-trace/`; its `summary.md` was refused to it by the
  harness and is the lead agent's copy of its reply.
- **One breach.** The reviewer ran one grep over `modes/tty/*.go`, to find where four `cycle…` functions
  live, and reported it (`m4-mid-trace/T1.md`, "Where I stopped"). It found T1's S10b's neighbours, a
  site whose shape Walkthrough 6 step 9 already routes; no other site rests on it.
- **The pass, per task,** is the before's (0.19.5 D-24): every site reached from the docs or its omission
  fails a guard that names it; no sentence routes to a wrong or missing file or symbol; every ID met
  resolves through `ids.md`.

## The grade

| Task | Sites | Wrong routes | IDs | Midpoint | Before |
|---|---|---|---|---|---|
| T1, a map Setting | every code site, the Overlays menu's eight and S12h among them (the menu's by `where-things-happen.md:25` and `:29`, then reading `handleOverlaysKey` and its caller); the five goldens, `setup-133x44-maps` now by name (B1.4); the fixtures Ta to Te by step 10; Tf and Tg guessed, their guards naming themselves; CHANGELOG (D2) by `docs/extending.md:32` (B1.4). **Not reached, and no guard:** README's MAP list (D1), which names every Maps-tab Setting (`README.md:243-250`) and which no step names | none false. `docs/extending.md:191` names `livePipelines.ttyConfig`, which hands a map Setting on through `mapConfig` (`app/dashboard.go:389`, `:424`): the reviewer called it imprecise, and found the line | U2-61 not in the index; the bare map D-IDs each resolve to two to four releases | **FAIL** (D1, IDs) | FAIL |
| T2, the UV legend head | S1 by a route this time: Walkthrough 5 step 1 names `modes/tty/map_temp.go`, whose `tempLegendRow` comment says it is the row under the map; S2's guard names itself; CHANGELOG by `:32` | 1: `modes/tty/map_boxes.go:7-8`, "The legend over the upper right is map_pane.go's (D-44)", a box 0.18.0 D-103 retired. The key had it (its finding 1); the before's reviewer never met it | W10.10, W1.17 and U2-18 not in the index | **FAIL** (a route, IDs) | FAIL |
| T3, the DATA PULL line | all 4: S1 and S2 by `where-things-happen.md:67` (B1.4) and Walkthrough 8; S3 by step 5; S4 by `:32` | none | PD-3 not in the index | **FAIL** (IDs alone) | FAIL |
| T4, "yesterday at" | all 5: S1 and S2 by `where-things-happen.md:67` (B1.4); S3, the Composer's midnight note, by the Composer row (`:118`), which the key had as unrouted; S4 named; S5 by `:32` | none | S-7 and SAM-D-26 not in the index; FR-4.5, cited bare at `app/burst_words.go:87`, resolves only to 0.18.0's and 0.19.0's unrelated rows | **FAIL** (IDs alone) | FAIL |

**0 of 4 pass, as before. On the first two tests, the sites and the routes, 2 of 4 pass (T3, T4), from
0 of 4.** Every wrong route the before found is gone: the time row and the card's cell count (B1.4),
Walkthrough 6's golden and its searches (B1.4). Both CHANGELOG gaps are routed. What is left is one route
the before's reviewer never met (T2's), one document no step names (T1's D1), and IDs.

**The IDs fail every task, and nothing in the plan clears them.** B1 was not to fix them: `m4-before.md`
gives a bare D-ID its release in each batch's comment pass and leaves an ID with no row to the citation
lint (B0.10). That lint fails only a new citation, so the families met here with no index row at all,
U2- (UAT findings), W (plan tasks), PD-, S- and SAM-D-, stay unresolved at REVIEW whatever the split
does. **D-50 rules it:** `tools/ids` indexes the six families (B1.6).

## What the reviewer found that the key had not

- **T4's spoken form.** The Composer's midnight note (`app/compose_takeover.go:18-21`) writes its times
  as digits, "at 11:59 PM", where the line says them in words.
- **T4's FR-4.5.** `testLine`'s citation is bare and resolves to two unrelated requirements.
- **T3's stale test comment.** `modes/tty/broadcaster_upnext_test.go:97-98` still says "a card that has
  about sixty" cells, the figure B1.4 corrected in `cardPulledShort`'s comment.
- **T1's open.** `toggleMap` is the place to read the Setting and `resetPlayback` is not: its doc calls it
  a mode's start, so the line there would also undo a `<` flip on every `R`. The key's S11 is the same
  site; the reason is new.

## What the reviewer got wrong

- **T3's goldens.** It expected all twelve console goldens to move. They draw `DATA PULL: —`
  (`modes/tty/testdata/console-*.golden`, every one), as the key measured and `m4-before.md` lists among
  the guards missing; Walkthrough 8 step 4 told it to re-record them, and the step's claim does not hold
  for this row (the key's finding 2).

## What it means for 0.19.6

- **The doc fixes worked where they were aimed.** Each of B1.4's five fixes was met and held.
- **Three new doc errors** (T1's D1 and `:191`, T2's `map_boxes.go:7-8`, T3's test comment) are B1's
  kind of fix. Fixed after this trace, they are the docs' and not the split's at REVIEW; the build log
  names them so REVIEW's delta can leave them out.
- **The ID test** decided every task here; D-50 has `tools/ids` index the six families in B1.6, before B1 closes, so at REVIEW that test is the docs' too.
