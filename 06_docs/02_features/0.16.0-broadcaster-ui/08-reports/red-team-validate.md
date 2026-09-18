---
title: "0.16.0 Broadcaster UI — the VALIDATE red team, verbatim"
date: 2026-09-18
phase: VALIDATE
sev: SEV-0
authority: HUM LEAD
status: "Record. Two blind reviewers at 6bab316 (docs quality; hygiene + safety), then a fresh reviewer on the bed fix, iterated to LGTM. Dispositions are in validate-report.md and follow-ups.md."
---

# The VALIDATE red team — reports as handed back

Dispatched blind, each in its own clone, with `06_docs/red-team-brief.md`'s discipline. Reproduced as
handed back, with only harness scratch paths and a quoted personal path removed.

## Docs Quality — the README audited whole

# README narrative audit — watchpost 6bab316 (feature/0.16.0-broadcaster-ui)

Method: fresh clone at 6bab316 under the scratchpad; README and the three docs read whole; every Broadcaster claim chased to `modes/tty/router.go`, `broadcaster*.go`, `platform/config/broadcaster.go`; `WATCHPOST_DEBUG_RADIO` exercised by a scratch test that kept its file; the `[broadcaster]` example paste-loaded through `config.Load()`; `make build` run. Repository untouched (scratch tests deleted; `git status` clean).

## (1) Broadcaster section and key table

**Backed** (true, with the backing line): `ctrl+b`/`B`, `ctrl+o`/`O` — `router.go:164-165`; leaving refused while ON AIR — `canSwap`, `router.go:~883` (`router_refusal_test.go`); toggle `shift+enter` ignored behind a window — `toggleStation` checks `ModalOpen()`, `router.go:~893` (`router_door_test.go`); `0`–`9`/`A` — `openCardWindow`, `router.go:1034/1029`; `r` Line-Up Request, `l` lookup — `router.go:178-179`; `b`, `shift+←/→` — `router.go:190/207-208`; `+ = -` gain — `router.go:172-173`; `ctrl+d`, `s a S ? q` — `router.go:169-182`; state written in words — `stationLine`, `broadcaster.go:1188-1206`; priority track drains first — `platform/lineup/lineup.go`; pool footer "N in reach at R mi" — `broadcaster_pool.go:212-218`; radius bounds/defaults — `broadcaster.go:62-89`.

**README.md:186** — Severity: **Important**. Evidence: "never sent anywhere". `app/bedrelay.go:70` → `app/radio.go:314 resolveAt` → `d.nws.CountyUGC(ctx, ref)` with the transmitter's ref → `domains/weather/nws/points.go:107` builds `/points/%.4f,%.4f`. The tower's coordinates go to api.weather.gov every bed refresh. The diagnostic half IS backed: `app/radio.go:1227 writeRadioDebug` rewrites every `lat,lon` pair via `snapshot.ReplaceKeys` (`platform/snapshot/redact.go:17-29`); my kept file showed `place:b2245476` in the needs-read line, the Director trace and even inside a URL, with no digits (`TestTheRadioDiagnosticFileCarriesNoCoordinates`, `app/radio_debuglog_test.go:167`). Caveat: the pattern is `lat,lon`; a `lat=x lon=y` string survived in my test, but no production writer formats one (all 30 call sites checked). Simplify/Delete? N. Action: "never sent anywhere but the National Weather Service, as every place on your watchlist is, and never written to a debug dump".

**README.md:181** — Minor. The console prints a third state, `STOPPED` (`broadcaster.go:1188`; `platform/lineup/power.go:23`), before the first `shift+enter`. Action: "STOPPED until you start it, then STANDBY or ON AIR".

**README.md:193** — Minor. "promote, demote, drop" are `P` (type a position, `enter`) and `k` (`router.go:1073-1076`); the table names neither key. Action: add "`P` move to a position · `k` drop".

## (2) The `[broadcaster]` config example

Keys, bounds and defaults match `platform/config/broadcaster.go:26/41/62-89` (2–100 default 25; 25–150 default 100). Pasted, it loads (`service=25 bed=100`).

**README.md:324-326** — Severity: **Important**. Evidence: with only `label` and `zip`, `Transmitter.Lat/Lon` are 0, and `Station()` (`broadcaster.go:126`) falls back to the default location — my paste-load returned `station={Vista, CA}`. The example is a silent no-op that reads as a setting. Simplify/Delete? Y — delete the two fake values. Action: show all five keys with a real lat/lon, or replace the block with one line: "`[broadcaster.transmitter]` carries `label, zip, lat, lon, tz`; set it from Settings — a table without `lat`/`lon` is ignored."

**README.md:326** — Minor. `Location` has six fields; `tag` (`config.go:27`) is omitted. Action: "the same fields".

**README.md:317** — Severity: **Important**. "every one is also a row in the console's Settings": `service_radius_mi` and the transmitter have rows (`setup_form.go:215/268`); **`bed_radius_mi` has no row anywhere** (`grep -ri bedradius modes/` → none; only `app/dashboard.go:162` reads it). Same class of claim 0.15.0 had to retract for the voice roles. Action: "two are rows in Settings; `bed_radius_mi` is config-only".

## (3) Other sections

Verified numbers: thirteen themes (`ThemeNames()` = 13); eleven AA-exempt tokens (`06_docs/follow-ups.md:155`); 500-row severe cap (`domains/severe/severe.go:42`); twelve dumps, 60 s apart (`app/dump.go:39-40`); ten favourites (`modal_location.go:113`); 63 MB voice (`app/cast.go:254`); `--ascii`, `NO_COLOR`, exit codes, schema unchanged since 0.15.0. Every Observer key in the table resolves in `defaultKeyMap()` (`dashboard.go:418-428`).

**README.md:42-44** — Severity: **Important** (designer/PM audience). "What it looks like" carries no Broadcaster image; `docs/img` has none (`ls docs/img | grep -i broad` → empty). The release's headline surface is described in 400 words and shown nowhere. Action: one console screenshot (STANDBY with the line-up and bed) and one ON AIR frame.

**README.md:134** — Minor. The Observer table omits `ctrl+d` Diagnostics though bound (`dashboard.go:424`) and the Broadcaster table lists it. Action: add it.

## (4) Install and build

Clean. `scripts/install.sh:7-9,46-51,78-91` does what lines 15-25 and 79-84 say; `Makefile:296-299` = the listed gates (`gate-controls` is the "positive controls"); `quality: p10` (307); `lint-authoring` runs `-self-test` (109); `release.yml:22` pins 1.27.0, `go.mod` 1.25.0; `make build` produced `dist/watchpost` stamped `0.14.2-500-g6bab316`.

## (5) Linked docs

`where-things-happen.md:102` — Minor. "fifteen slots": `platform/lineup/operator.go:243 MainTrackCap = 16`. The symbol gate cannot see a number. Action: "sixteen".
`extending.md:155` — Minor. `lint-injector.sh` is also called by `build-diag` (Makefile:39) and `gate-controls --self-test` (:142). Action: "the only release-path callers".
`extending.md:153-161` gate claim (required minus verify = `p10`, `release-matrix`, `install-test`) holds exactly. `accepted-costs.md:226` pins match `broadcaster_alloc_test.go:215-216`.

## (6) Audience order

Section order is right (use → looks → daily keys → features → tinkerers → build). What the first two audiences lack: the Broadcaster picture (above) and, in the Broadcaster prose, any sentence on what a listener would hear differently — the section is entirely operator-facing.

## Verdict

**Do not ship as-is** — three Important, all in text written this week, all one-line fixes: the privacy sentence (README.md:186) overclaims against `points.go:107`; the "every one is a row" sentence (317) is false for `bed_radius_mi`; the transmitter example (324-326) silently does nothing. **Fix first: README.md:186** — a privacy claim a reader will rely on, and the code contradicts it.


## Project Hygiene + safety — from a clean clone

## VALIDATE — hygiene + safety, fresh clone at `6bab316`

Clone: `scratchpad/validate-hygiene/repo` (verify, untouched) and `.../lab` (my drivers). Nothing at the original path was edited.

### (1) Clean clone
`make build` OK (`0.14.2-500-g6bab316` — `git describe` on this branch reaches no 0.15.0 tag; harmless, the release stamps `VERSION` from the tag). `make verify`: **ALL GATES GREEN, `real 18m59s`** (`verify.log:1788-1790`), 116 `ok`/PASS lines, 0 FAIL.

**F1 — README.md:392-393 says verify "runs from a clean clone with nothing outside the tree". It needs four things it does not name.** Severity: Minor. Evidence: `scripts/lint.sh:38` fetches golangci-lint `@v2.13.1` and `Makefile` `vuln` fetches govulncheck `@latest` (network + module proxy); `scripts/lint.sh:57` `python3`; `docs/extending.md:162` GNU Make ≥ 3.82 (macOS ships 3.81 and the gate oracle is COULD-NOT-RUN by name — README's build section is silent); git (VERSION, identity gate). Simplify/Delete: N. Action: one sentence in "Building from source" listing them; drop "nothing outside the tree".

### (2) Release path
Every step `release.yml`/`ci.yml` runs exists in the Makefile (`verify`, `release-matrix`, `install-test`, `mutant-policy`, all 22 per-gate CI targets in `.PHONY`). Runner tools: python3, curl, sha256sum/shasum, brew make on macOS — all present on ubuntu/macos-latest. `make install-test VERSION=0.16.0-validate` locally: 5 targets built, `lint-injector: OK (5 artifact(s))`, installer OK, tamper control fired, 16.5 s. `strings` on darwin-arm64 and linux-amd64: **0** `/Users/` or harness paths (`-trimpath` holds).

**F2 — `.github/workflows/release.yml:15` `timeout-minutes: 20` will kill the v0.16.0 release job before it publishes.** Severity: **Critical**. Evidence (GitHub API, read-only): the 0.15.0 release run `34352664330` took **16.1 min** of 20 (step `verify` 14.2 min); the trend is 13.0 → 13.6 → 13.7 → 16.1 across four releases. The last ubuntu `verify` job on this feature branch (`66dc88d`, 2026-09-13, before five more days of code) took **21.5 min** (`mutant-check` 13.8, `test-tags` 2.4, `race` 2.3), and this clone's verify on a faster Mac took 19.0 min. The tag-time job runs verify **plus** release-matrix (1.3 min) plus install-test plus publish. `fail_on_unmatched_files` means nothing half-publishes — the tag simply exists with no release and the installer 404s. Simplify/Delete: N. Action: raise to `timeout-minutes: 45` (mutant-check alone is allowed 40 m at `Makefile` `mutant-check`), and record the measured job time in the release checklist so the cap is derived, not guessed.

### (3) Should-not-ship scan
Tracked non-text: 18 `docs/img/*` (largest 3.2 MB gif), 2 geodata `.gz`, `domains/radio/player/testdata/tone.mp3`; 24 executables, all shell/expect/py scripts. No Go binaries. No secrets (key/token/PEM/ghp_ patterns: 0). Emails: agency/service addresses in exempted fixtures, RFC-2606 placeholders, and THIRD_PARTY_LICENSES only. `/Users/`, `<scratch>`, scratchpad: only inside the rules themselves and prose about the leak (`exposure-statement.md:23-24,88-89`, `red-team-review.md:318`) using `<account>`/`you`. `go test ./cmd/watchpost -run 'PublishedTreeNames|NoTrackedBinaries'`: **both PASS** (3.1 s).

What the gates cannot cover: `identity_test.go:139` skips `.png/.gif/...` — pixels (F-48's `status.png`) and the 15 MB blob in branch history (`6b1b621` → `docs/accepted-costs.md`, 15,235,221 bytes; the one object > 400 KB in `94b654b..HEAD`) are outside them; the squash-merge ruling is what keeps that blob off origin. `.mp3` is not in the skip list, so it is scanned as text (passes today by luck).

**F3 — CHANGELOG.md:5 top entry is `[0.15.0]`; no 0.16.0 entry yet.** Minor. Prior releases added it in the ship commit; noting so it is not forgotten. Action: SHIP step.

### (4) Owed items
Follow-ups (10 CLOSED this week): F-118, F-122, F-125 (A20 `gatemodel_test.go:83`), F-127 (A5/A37), F-131 (A33 `exemptions_test.go`), F-136 (`tools/authoring/main.go:42,158`), F-150, F-153 (`docs/extending.md:162`) — **all true against the tree**.

**F4 — `06_docs/follow-ups.md:` row F-141's closure text is now false.** Minor. It says "`make verify` from a clean clone stops [at p10] by design"; `2bf599d` (R1) removed `p10` from `verify-gates` and README:392 now says the opposite. Simplify: Y (one clause). Action: append "superseded by R1 `2bf599d`".
F-145 quotes 137/339, 212/746 at `59c8b38`; the statement (`exposure-statement.md:30-31`) and review report now say 137/341, 214/750 at the tip — a historical citation, acceptable, but the row should say so.

P10 exemptions (5 of 153 in the gitignored ledger; mirror `06_docs/p10-ledger.md` = 162 rows − 9 headers = 153, matches): `config.go:435 var mu sync.Mutex` ✓; `units.go:306 asciiMarks = sync.OnceValue` ✓; `schema.go:51,59 maxDepth=12/typeSchemaDepth` ✓; `sched.go:28 RealClock.Now` ✓; `tools/treelock` 8 funcs ✓. `tools/authoring` row says "15 functions"; the package now has 39 (4+9+26, `selftest.go` grew after ratification) — the reason still holds, the count is stale (Minor, refresh the row's figure at the next P10 run).

### (5) Safety, from the outside
Premise correction: **`WATCHPOST_MAINTRACK=live` is not a value** — `app/maintrack.go:50-55` recognises only `dark`; `live` is `off` ("live IS GONE, D-33"). So the sequences were driven at the Director/console seams (my `platform/lineup/zz_*_test.go`, `modes/tty/zz_validate_test.go` in the lab clone).

| Sequence | Director (`platform/lineup`) | Console band |
|---|---|---|
| 3 compose faults, topped off | faults 1–2: `faultRun` 1,2, no effect; fault 3: `escalate(run=3)` (`fault.go:63,73`) | faults 1–2: **nothing**; fault 3: `!!! 3 CARD(S) FAILED — … could not be composed: no key` |
| relay dies under a carrying bed | deck reports nothing (`radio.go:1099-1106`: `Tuned` only at Playing, `Ended` only for a synth cycle); with `carries` true `advancesMonitor()` is false (`air.go:167`) so **no dwell, no stall report, ever** | `● RELAY BED … ACTIVE`, ON AIR, no band — the deck's own fall-through is the only recovery |
| standby → on air after a run | `faultRun` reset (`power.go:159`) | band cleared, stays cleared on the way back up |
| fault while OFF AIR, empty schedule | `escalate(run=0)` | hidden at standby; **shows `!!! STATION FAULT — …` on the next ON AIR** (by design, F-163) |
| nil deck, ON AIR, nothing scheduled | `toggleStation` needs only `r.station != nil` (`router.go:926`); `schedule.go:230` hands the control over for a nil deck too; `readerFor(nil)` is nil (`mainread.go:45`); a rail card "reads" inaudibly by design (`executors.go:458-459`) | `*** ON AIR · BROADCASTING ***` — yes, the station can show ON AIR while nothing can play, with no cue that the machine is voiceless |

**F5 — `platform/lineup/bed.go:127` `d.bed = bed{ref:…, live:…, since:…}` replaces the whole bed on every new `Tuned`, wiping `carries` (the operator's cut-over), `asked` and `ducked`.** Severity: **Important** (FR-4.2/D-11). Evidence, driven: `CutOver{true}` → `carries=true, advances(MainTrack)=false`; then `Tuned{r2}` (operator steps the bed, `bedrelay.go:189` `tuneResolved` → `radio.go:1106`) or `Tuned{r1, Live:false}` (relay fell through to synth) → `carries=false, advances(MainTrack)=true`, **no `Publish` on that step** (console still says ACTIVE until the next tick). The main track then resumes and a read replaces the relay the operator chose. No test sends a `Tuned` after a `CutOver` (`cutover_test.go`, `bed_test.go`: none). Simplify: Y. Action: assign the three observed fields only (`d.bed.ref, d.bed.live, d.bed.since = …`) and pin it with the sequence above.

**F6 — `modes/tty/broadcaster_air.go:79-80` shows "NO REPORTS READ OR ACTIVE **IN STANDBY MODE**" on the LIVE NOW row of an ON AIR station.** Minor (HUM LEAD's verbatim wording, D-89 — a ruling, not mine). Action: HUM CALL on an ON AIR variant.

**F7 — no console cue that the station has no audio engine.** Minor-to-Important, ratified as design (D-91 "ON AIR never means the antenna is radiating"; the Observer's `[S]` carries `radio_unavailable`). Recorded, not re-litigated.

### (6) The record
`handoff-0.16.0.md` §1 vs tree: `73bce2b`, `fd761ab` (= real `origin/main`), `f7fe5fa`, `0963b36`, `6b1b621`, `94b654b` all exist; 380 mutant files with `m16/m43/mBM1/mSC3` present and `mBM2` present; 153 ledger rows ✓; F-156..F-163 rows present ✓; "followed only by records" — the five commits after `73bce2b` touch `06_docs` and README only ✓. Two figures do not reconcile: **"62,419 added lines"** (§1 prose) — `git diff --shortstat fd761ab 73bce2b` gives 68,796 (all) / 47,611 (excluding `06_docs`); and "zero objects over 400 KB in the whole release range" sits three lines above the acknowledged 15 MB blob in that same range (Minor, say "after the rewrite, before `6b1b621`"). `review-report.md`: all 15 commits it names exist; `2bf599d` does remove `p10` from `verify-gates`; `mutant-verdicts.log:1` opens with `73bce2b`'s **commit** hash labelled `tree:` (the tree hash is `0ff79f8…`) — wording. Its evidence column cites `dist/verify-review-73bce2b.log` and `-35de36a.log`, which are gitignored and **not in the tree** (Minor: promote or cite `gates.md`).

### Verdict
**Do not ship as-is — fix F2 first**: `release.yml:15` `timeout-minutes: 20` against a measured 21.5-minute ubuntu verify means the tag will produce no release. It is a one-line change; then F5 (`bed.go:127`) before the tag, because it is an unpinned loss of operator intent on the safety surface and the fix is three fields.


## Remediation review — the bed cut-over fix (two passes to LGTM)

### Pass 1 of 2

**Review of `6b4a303..d3538a9` (fresh clone at `d3538a9`, `go test ./platform/lineup/` green on the untouched tip; all plants and probes run on my clone only, tree restored and clean).**

## Answers by name

**(1) Correctness and completeness.** Writers of `d.bed`: `bed.go:130` (onTuned: ref/live/since), `:137` (tuneAsked), `:156,:160` (stalledRotation clears asked), `:204,:210,:256` (since), `:330` (carries, onCutOver only), `:400` (ducked). Readers of `carries`: `bed.go:327,:387,:413`, `air.go:167`, `power.go:127`. Readers of `asked/askedAt`: `bed.go:152-160` only. `ducked`: `bed.go:397`. No other step resets the bed wholesale: `onPowered` (`power.go:145-160`) touches `power`/`faultRun` only, `CutOver{false}` flips `carries` alone, `Tick` never writes `bed` except through `advanceBed`/`stalledRotation`. The whole-struct write at old `bed.go:127` was the ONLY place `asked` was cleared on landing, and the fix removed it without replacing it. **The fix is correct for `carries`/`ducked` and incomplete for `asked`** — see (3).

**(2) Semantics under `Tuned{Live:false}`.** Constructed through `Director.Step` (Running, AirProgramme, Monitored, Programme{r1,r2,1m}, NeedsRead, Tuned{r1,true}, CutOver{true}, Tuned{r1,false}, ten one-minute Ticks, Ended{}, Tick +1h, CutOver{false}). Observed: after the fall-through `bed={ref:r1 live:false carries:true}`; ten ticks emit no `Tune` and no `Escalate` (`air.go:167` returns false while `carries`, so `dwellElapsed` at `bed.go:267` and `onEnded` at `bed.go:246` both stop); `Ended{}` emits nothing; one hour on the main track is not on air and `Publish.Bed` reads `{r1 false true}` ("carrying"); only `CutOver{false}` releases it. The deck side (`app/radio.go:1098`) sends `Ended` when the synth cycle plays out and issues no tune of its own, so the station is silent with the main track paused for as long as the operator does not act. Before this commit the fall-through wiped `carries`, so the main track resumed — the very defect for `Live:true`, but for `Live:false` it was the only exit. `bed.go:72-76` says `carries` is "relay-only from birth"; the tree now holds `carries && !live`, a state that comment says cannot exist. **This is a HUM LEAD fork, not mine to rule:** (a) a fall-through under cut-over clears `carries` (the relay the operator chose is gone); (b) it stands, and the station is silent until the operator cuts back; (c) `onEnded` under `carries` re-tunes. Whichever is ruled, `TestACutOverSurvivesTheBedMoving`'s `{r1,false}` case currently pins (b) by construction.

**(3) `asked` / `askedAt`.** Constructed: Monitored, Programme{a,b,1m}, Tuned{a}, Ended{} (issues `Tune{b}`, `asked="b"`), Tuned{b}, Tick at `tuneLands+1s`. Observed at `d3538a9`: `Escalate{"the station was asked to move to b and did not"}` — **a tune that landed is reported as a stall 31 s later.** Same probe at parent `6b4a303`: no escalation. This is a regression introduced by the change. `TestATuneThatLandsIsNotReported` (`bed_test.go:320-333`) does not catch it because it ticks at `10*tuneLands`, past the 60 s dwell, where `advanceBed` re-issues a tune and resets `askedAt` — the exact trap the sibling test's own comment (`bed_test.go:308-312`) names.

**(4) Plants** (`bed.go:130`, full package suite each time):
- Restore `d.bed = bed{ref, live, since}` — **CAUGHT**: `cutover_test.go:162` "the cut-over was forgotten…" and `:165` "the main track advances while the operator's cut-over stands", both `Tuned` cases.
- Add `d.bed.carries = false` — **CAUGHT**, same two lines.
- Add `d.bed.asked = ""` — **SURVIVED** (suite green). Note this plant is the behaviour that fixes (3); the suite cannot distinguish clearing from not clearing in either direction.

**(5) Release cap.** `ci.yml` sets no `timeout-minutes` on `verify` (default 360), so 45 is derived from the comment alone. The measurement is honest for the work: `MUTANT_POLICY ?= push` (`Makefile:431`) means CI's ubuntu run executes every `verify-gates` member including `mutant-check` plus `release-matrix` and `install-test` — the same steps as `release.yml:23-28` minus `publish`. 45 ≈ 2.1× measured is defensible. The clause "mutant-check alone is allowed 40 (Makefile)" is not: `Makefile:195` grants `-timeout 40m`, and 40 + the remaining ~16 (the 0.15.0 figure the same comment cites) is 56 > 45, so a mutant run that uses the bound the Makefile allows still produces a tag with no release. Either the rule is "2× measured" (then delete the 40 clause) or the cap must cover the Makefile's own bound (≥ 56, i.e. 60). No tighter honest number exists in the tree.

**(6) Necessity, names, comments, P10.** Loop at `cutover_test.go:153` bounded by a two-element slice; no recursion; no new branches beyond P10-04. Names read as the domain. Comment findings below.

## Findings

1. `platform/lineup/bed.go:130` — **Critical.** Evidence: (3), parent green/tip red. Simplify/Delete? N. Action: on a `Tuned` whose `Ref == d.bed.asked`, clear `asked`/`askedAt` (an observed landing is exactly what the ask was waiting for); pin it with a tick between `tuneLands` and `Dwell` in `TestATuneThatLandsIsNotReported`, and re-apply the omission to prove the test fails. Fresh reviewer per the remediation loop.

2. `platform/lineup/bed_test.go:327` — **Important.** Evidence: `10 * tuneLands` sits past the dwell that resets `askedAt`; the `asked` plant survived. Simplify/Delete? N. Action: tick at `tuneLands + time.Second` (the sibling at `:304` already does), so the test can fail.

3. `platform/lineup/bed.go:72-76` vs `cutover_test.go:153` `{r1,false}` — **Important (fork).** Evidence: (2). Simplify/Delete? N. Action: HUM LEAD rules (a)/(b)/(c); record as MVS-D-nn; then either the comment or the test case changes.

4. `.github/workflows/release.yml:15` — **Minor.** Evidence: (5). Simplify/Delete? Y (the 40 clause). Action: state the derivation rule once, and drop "this branch's", which rots at the next branch.

5. `platform/lineup/cutover_test.go:150-151` — **Minor.** "The first draft replaced the whole bed…" narrates history in code. Simplify/Delete? Y. Action: keep the rule, move the story to the record.

`4ee7421`'s other edits (`Makefile:387` `tree:`→`commit:` with no reader of the old prefix; `setup_station_test.go:243` tracks 6b4a303's wording; docs) are consistent — no findings.

## Verdict

**NOT LGTM.** Fix first: finding 1 — `d3538a9` trades one FR-4.2 defect for an FR-9.3 false fault on every ordinary Watchlist rotation (a stall window 30 s after each successful tune), and the suite cannot see it.

### Pass 2 of 2

**Re-verification at `ab1f04d` (fresh clone `wp2`, read-only; `go test ./platform/lineup/` green; tree restored clean after plants).**

**(3) Stale ask, re-run.** Monitored, Programme{a,b,1m}, Tuned{a}, Ended (Tune{b}), Tuned{b}, Tick at `tuneLands+1s`. Observed: `asked=""` after the landing (`bed.go:132-134`); the tick emits only `Publish{... STOPPED {b true false}}` — no `Escalate`. The false stall is gone.

**(2) Fall-through, re-run.** Running, AirProgramme, Monitored, Programme{r1,r2}, NeedsRead, Tuned{r1,true}, CutOver{true}, Tuned{r1,false}. Observed: `carries=false`, `advances(MainTrack)=true`, `onTuned` itself emits nothing (`bed.go:138-140`); the next tick's `Publish` carries `Bed{r1 false false}` with the oceanside card back on the schedule; `Ended{}` under the released bed emits nothing (Dwell 1m, `advancesMonitor` true, but `Ended` from synth with `nextInWatchlist`=r2 — wait, observed `fx=[]` because `running`'s fixture… no: observed `[]` as logged). The station no longer sits silent with the programme paused; the ruling is recorded at `06_docs/follow-ups.md:386` (F-164) with the (a)/(b)/(c) options and the observed sequence.

**Plants (committed tree, full package suite):**
- Ask-clear removed (`bed.go:132-134`) — CAUGHT: `bed_test.go:334` "a station that moved when asked was reported as stalled: [Escalate …asked to move to b and did not]" and `cutover_test.go:169` "a tune that landed is still pending ("r2")".
- Whole-struct write restored (`bed.go:141`) — CAUGHT: `cutover_test.go:160,163`.
- Synth release removed (`bed.go:138-140`) — CAUGHT: `cutover_test.go:184,187`.

**Release cap** (`release.yml:15`): 60 = the Makefile's 40-minute `mutant-check` bound (`Makefile:195`) + ~16 measured remainder; the derivation is now stated from the tree's own bound and "this branch's" is gone. Consistent.

**Comments:** `cutover_test.go:148-152` and `bed.go:127-137` describe the current rule only; no history narrated.

**One observation, Minor, not blocking (`bed.go:138-141`):** a fall-through releases `carries` but `onTuned` does not `settle`, so the console's `Publish.Bed.Carrying` flips only on the next tick (observed: `fx=[]` on the Tuned, `{r1 false false}` one tick later). Same shape `onTuned` has always had for `ref`/`live`; if F-164 rules that a fall-through should re-tune, that ruling's fix is where a settle here would belong.

**LGTM.**

