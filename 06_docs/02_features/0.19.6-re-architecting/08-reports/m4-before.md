---
title: "0.19.6 M4 — the blind trace's before, on e57afeaf"
date: 2026-11-15
phase: PLAN
sev: SEV-0
status: "Graded: 0 of 4 pass on e57afeaf. The yardstick M4's REVIEW gate is judged against (B-6)"
---

# M4's before: the blind trace on `e57afeaf`

**What this is.** M4 (brief, B-6) asks whether a junior can trace a change from the docs. Its REVIEW
gate is 4 of 4 tasks passing first time on the new layout, against a fresh key. This is the same four
tasks on today's tree, so the release has a before.

- **The tasks** (A-28), fixed at PLAN: `m4-key/tasks.md`.
  - T1, a map Setting that crosses the root and the map package.
  - T2, a legend key, with no walkthrough.
  - T3, a Broadcaster card change.
  - T4, a spoken phrase in the deck.
- **The keys.** The key builder did each task on a scratch copy of `e57afeaf` and left out one site at a
  time: 36 `go test` runs, one at a time. The keys are in `m4-key/`.
- **The reviewer.** A fresh agent with 0.19.5 D-24's junior persona traced the tasks from README's
  "Start here", on a copy of `e57afeaf` that holds no 0.19.6 record and no path to the keys. It read only
  what the docs and the code's comments routed it to, plus `06_docs/ids.md` and `go doc`, and ran no
  repository-wide search. Its answers are in `m4-before-trace/`.
- **The pass, per task** (0.19.5 D-24):
  - every site is reached from the docs, or its omission fails a guard that names it;
  - no sentence routes to a wrong or missing file or symbol;
  - every ID met resolves through `ids.md`.

## The grade

| Task | Key's sites | Reached by a route | Not reached, and no guard | Wrong routes | IDs | Before |
|---|---|---|---|---|---|---|
| T1, a map Setting | 44 (43 as built, plus S12h found by the reviewer) | every code site, the menu's 8 included, and S12h, which the key lacked; the 5 goldens; 6 of the test sites (Tf and Tg are guarded) | README's Maps-tab list (D1); CHANGELOG (D2) only guessed | 3: `docs/extending.md:173-174` sends the reader to search; `:190` names `setup-133x44-ui` where a map row moves `setup-133x44-maps`; `UIPrefs.MapPropRefresh`'s comment says empty is each new grid, where `platform/config/config.go:316-319` says empty is hourly | the bare map D-IDs each resolve to two to four releases | **FAIL** |
| T2, the UV legend head | 3 | none: the site was reached by a guess, because no doc names the row under the map (JN-3) | CHANGELOG (S3) | none | W10.10 and U2-18 not in the index | **FAIL** |
| T3, the DATA PULL line | 4 | all 4 | none | 2: `docs/where-things-happen.md:67` names `platform/render/clock.go` as the one owner of written times, and this stamp formats its own date; `modes/tty/broadcaster_manifest.go:121-122` says the card has "about sixty" cells, where it has 36 at the floor (`docs/extending.md:238`) | 0.16.0's D-110 and PD-3 not in the index | **FAIL** |
| T4, "yesterday at" | 5 | 4: the function, its comment, the Composer's midnight note, the new test | CHANGELOG (S5) only guessed | 1: `docs/where-things-happen.md:67` again; the spoken today-or-date choice is `app/burst_words.go:spokenWhen`, not `clock.go` | S-7 and SAM-D-26 not in the index | **FAIL** |

**0 of 4 pass.** The code routes hold better than the grade suggests: T1's 28 code sites, T3's four and
T4's four were all reached. The failures are the docs around them: a README row and a CHANGELOG with no
route, three stale or wrong routes, and IDs that resolve to several releases or to none.

## What the reviewer found that the key had not

- **T1's S12h.** The Overlays menu's change test (`modes/tty/map_pane.go:1068`) compares three fields.
  A menu move of the new Setting returns before `menuApplied`, so the menu never saves it. The key
  missed it; the reviewer found it by reading `handleOverlaysKey`'s caller. It is added to the key.
- **T2's side effect.** A longer head narrows the UV swatches (`presetRow`'s `room`), so "nothing else
  changes" does not hold at every width.
- **T4's console echo.** The Broadcaster shows a takeover's script before it airs, so the console shows
  "yesterday at …" too.

## What a pass will and will not show (D-29)

- **The midpoint.** B1 fixes the routes these four tasks failed on, on today's layout. A fresh blind
  reviewer traces the same tasks after B1 (plan B1.5); REVIEW's delta from that midpoint is the split's,
  and the midpoint's from 0 of 4 is the docs'.
- **The blind spots.** The four tasks are known in advance, and each is traced once. The grader is the
  lead agent, against keys it did not build. The keys sit on the branch the builder works on; each
  trace runs on a copy without this release's folder.

## What it means for 0.19.6

M4's gate is judged at REVIEW on the new layout, against a fresh key built by leave-one-out before the
docs are rewritten. On today's evidence the docs, not the code's shape, decide that gate. So the plan
puts the fixes where the record says they belong:

- **Docs fixed in B1** (with RC-7's and JN-5's errors), each a living document:
  - Walkthrough 6's golden for a map row, and its "Search for" sentences;
  - `where-things-happen.md:67`'s time row, naming `cardPulledShort` and `spokenWhen` as the places
    that choose a date;
  - `UIPrefs.MapPropRefresh`'s comment;
  - `broadcaster_manifest.go:121-122`'s cell count;
  - a CHANGELOG `[Unreleased]` heading, and a walkthrough line that every user-visible change gets
    an entry.
- **Routes the split itself creates** (B4 and B8): the map package's `doc.go` names its under-the-map
  rows, with one word for the legend (JN-3), and the Overlays menu's rows; Walkthrough 6 covers a map
  Setting that is also an Overlays-menu row (D-25).
- **IDs:** a cited ID that resolves to no row, or to several, is the citation lint's to catch (B0.10).
  Bare D-IDs in the code the split moves gain their release in each batch's comment pass.
- **Guards the keys found missing**, recorded for the batches that touch them:
  - the Overlays menu's layout test hand-lists its rows (`modes/tty/map_menu_test.go:54-63`);
  - `switchRow`'s fall-through saves an unknown row as a detail switch (`modes/tty/map_boxes.go:958`);
  - no golden draws a legend row;
  - all twelve console goldens draw `DATA PULL: —`.

  Each becomes a test in the batch that moves its file, or a follow-up where it is not the split's.
