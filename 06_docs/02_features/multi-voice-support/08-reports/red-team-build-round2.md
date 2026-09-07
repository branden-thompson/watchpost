# BUILD exit — red team, round 2 (convergence)

**Base:** `0e15e82` (the round-1 verdict). **Status: OPEN — the round has not concluded.**

Round 1's report is `red-team-build.md`. This is the convergence pass it called for, and it found
that roughly thirty of round 1's own remediations were defective. Every one of them had passed
`make verify`, `make p10`, the allocation pins and the goldens: **the gates cannot see any of the
defects in this report.**

> **This file exists because it nearly did not.** For most of this round the findings lived only in a
> chat log, which `follow-ups.md` and the red-team skill both say is the same as not existing. Two of
> the six safety defects below were named nowhere on disk until this file was written. If you are
> picking this up cold, this file and `06_docs/follow-ups.md` are the record.

## 1. The six safety defects

Worked one at a time through the loop in `06_docs/remediation-review-loop.md`: failing test through
the real entry point first, fix, mutate, then a fresh adversarial reviewer who must run something.

| # | Defect | Status | Commits |
|---|---|---|---|
| 1 | **A zone-only alert defeats the radius.** The tie that keeps a point-less alert when the app already tracks it was unscoped, and the RECENT table is seeded with the fifty largest US cities — so a Cook County tornado warning took the marquee and was read aloud to a listener in Oceanside with a 100-mile radius set. | **Behaviour correct.** Third reviewer: "holds under every attack I could construct." Four items outstanding, all honesty/coverage — see §2. | `400b847` (wrong), `e26661f`, `d01cfaf` |
| 2 | **An outbreak silences the quiet lanes on the air.** The tape's per-lane floor did not reach the audio path: the takeover took its eight by severity alone, and the cycle seen-marked everything on the same tick, so a live hurricane behind a tornado outbreak was never spoken and could never come back. | **Closed — partly fixed, partly superseded.** Three reviews. The double-announce this remediation introduced, the mark-on-read line, the takeover slot's release and the unpinned caps are all fixed and pinned. The starvation itself is **superseded by MVS-D-56**: the third review measured the boundary at ~8.14 s per read, and two rounds of moving it proved the ordering is a preference being treated as a constant. 0.14.0 holds for the Director-owned lineup (`01-objectives/read-order-design.md`) rather than tuning it a third time. |
| 3 | **`setLaneRows` ordering.** `publish` sorts its rows once, hands the same slice to `setLaneRows`, and then reuses it for the window's cap and totals. The lane cut orders by severity, the window by recency — so a cut that reached the caller's slice would silently relist the window by severity while the tape listed by recency. | **Verified closed, and now pinned.** Checked through `publish` rather than assumed: `laneRowsPerTab` builds its own per-tab slices with `append` to nil, so its sorts cannot reach the caller. `TestTheLaneCutDoesNotReorderTheWindow` drives `publish` with an old-but-severe advisory against newer mild ones, so the two orders genuinely disagree, and asserts the window still reads most-recent-first. A mutant that sorts the caller's slice is caught by it. | `(this commit)` |
| 4 | **A new headline waits for its lane's tape to wrap.** A lane parks its scroll offset and keeps it when its alerts change, so a warning issued while the rotation was elsewhere comes back parked deep in the old tape. | **Reverted, carried as F-16.** Two fixes were each worse than the defect (12 % then 47 % of the tape reachable). Needs its own DISCOVER with the ticker's geometry modelled first. | `07f762e` (reverted by `6ae6b24`) |
| 5 | **`--no-animation` froze the band at its first alert.** `advanceTicker` returned early under `NoAnimation`, so the offset never left 0. Measured on a full 30-alert lane: **1 of 30 alerts legible at 80 and 133 columns, 2 of 30 at 200**, permanently, while the band's counter said 30. | **Closed — the flag is withdrawn (MVS-D-55).** Paging was built and measured first: it reaches all thirty, but in 43 min of air on one lane and 3 h 39 m across six, which is the normal case. The HUM LEAD ruled that a technicality is not an accommodation. `--no-animation` had never shipped — it existed only in the unreleased 0.14.0 section — so it was removed rather than deprecated. The reduced-motion surface is `[w]`, a navigable table of the same alerts, already named by the spoken reports. | `(this commit)` |
| 6 | **An alert on the air is not the stop button's to end.** `playPCM` read only half of `giveWay`, so a rendered report opened *playing at full volume* over an alert; and `radioDeck.Stop` cleared a suppression the takeover owns, so whatever the listener started next came up over an alert still being read. | **Remediated, never reviewed.** | `eabc401` |

## 2. Outstanding reviewer items

**Defect 1 — CLOSED.** The third review's four points are answered: the redundant `key != ""` is
deleted (`NormalizeID` never returns ok with an empty key — probed over twelve inputs), the live
guard is pinned directly by `TestAlertKeysOfRefusesAnythingItCannotIdentify` and dies in isolation,
the cycle test says which of its two arms is belt-and-braces and why, and the stale `AlertsWithin`
in D-1 is corrected. One mutant deliberately still survives: `scopeEvents`' `ok` is defence in depth
and unreachable while `alertKeysOf`'s guard stands, which is now stated in the code rather than
implied to be covered.

The four points as the reviewer raised them:

1. `d01cfaf`'s message claims all three id mutants fail; only the combined one does. The two guards
   are mutually redundant, so neither is observable end-to-end alone.
2. Add a direct `alertKeysOf` assertion so the live guard dies in isolation, and say plainly that the
   other is unreachable while it stands.
3. `TestAnUnidentifiableAlertTiesToNothing`'s empty-id arm is vacuous — `Merge` and `addFeed` drop
   id-less events upstream, so it cannot fail. Keep it and say it is belt-and-braces, or move it
   where it is observable.
4. `alertKeysOf`'s `key != ""` is dead by the same argument used to delete `case e.ID == ""`. Delete
   it or stop describing it as the guard. `06_docs/follow-ups.md` D-1 still says `AlertsWithin`,
   renamed to `AlertKeysWithin` in `d01cfaf`.

**Defect 2** — three reviews run. What remains open is **superseded, not outstanding**:
- *R-1, the ~8.14 s starvation boundary* → MVS-D-56 replaces the arrangement.
- *Pin the time bound from above* (`breakingAllowance = 100h` survives) → pinning a bound that
  MVS-D-56 deletes is waste; recorded here instead so it is not mistaken for coverage.
- *`defer t.seen.save()` survives deletion* → exposure is bounded to a single cycle, because `cycle`
  saves unconditionally on every pass; not pinned, and stated rather than claimed.
**Defect 6 — CLOSED.** Two reviews. The first found the mirror of the fixed defect unguarded (a
relay opening at full volume over an alert); the second confirmed that closed and found the same
one-sided shape twice more on the same path — `StartSource`'s `setLive(false)` asserted on a fresh
engine, where `live` is already the zero value, and `alertDuck` pinned by nothing because every
expectation was written as `knob × alertDuck` and moved with it. Both are pinned now: a rendered
source started *after* a relay, and the dip depth as a literal. The three mutants the reviewer used
are checked in as `m34`–`m36`.

## 3. Round-2 findings not yet remediated

Reported in round 2 and **not re-verified since** — check each before acting on it.

| Area | Finding |
|---|---|
| a11y | **CLOSED as a documentation correction.** `--ascii` covers the furniture the app draws — marks, box rules, the legend and the spread-word headers — and not the forecast offices' own words, which reach the terminal as they wrote them. The tape's `" · "` is composed into an alert's title in the app layer and travels with it, so it is out of scope by definition rather than a leak. Ruled narrow; the README and the flag's help now say exactly that, because a wider promise would be one the flag does not keep. |
| a11y | **DOWNGRADED on re-verification.** The two tones do converge, but colour is not the channel: `body.go:178` prints a glyph (`✔`/`⚠`/`✘`) with the count and tints it, so state is legible without colour — which is R-12(a)'s rule, colour additive and never sole. Both tokens are already registered in `contrast.go`'s `onBoth` list and the AA tests pass. Left for the HUM LEAD's own colour pass rather than chased as a defect. |
| a11y | ~~`º` (U+00BA) used where `°` is meant~~ — **fixed.** Sixteen occurrences swapped to U+00B0; both are one cell wide so nothing moved, and the six goldens changed on that character alone. |
| Docs | Every README screenshot is 0.13.0, with alt text asserting 0.14.0 content |
| Docs | **VERIFIED GREEN now** — the grep over README and docs for the retired `[T]`, the min player and `radio-min` returns zero. The historical mis-record is already on the record in `p5-build-log` §4; nothing further is owed. |
| Docs | ~~README key table still teaches `M` as mute~~ — **fixed.** It now says `M` opens Settings at the alert tones, per MVS-D-48. |
| Docs | ~~`where-things-happen.md:48` — a blank line splits the table~~ — **fixed.** |
| Docs | ~~MVS-D-53 is not in the CHANGELOG~~ — **fixed.** The radius scoping the `w` window is named under Changed, as MVS-D-53 required. |
| Docs | ~~`gates.md:17` and `release-checklist.md:10` still mandate the deleted `tree_hash`~~ — **REFUTED on re-verification.** `dist/p10.json` still carries `tree_hash`, so the mandate is current, not stale. The real finding (p5-build-log §4) is narrower: the hand-copied `tree_hash` in the gates table is a *second* place to be wrong, and at P1 it does not match the JSON. The JSON is the evidence; the column should go. |
| Docs | `uat-p4.md` and `linux-validation-protocol.md` are stale |
| Hygiene | **CLOSED.** All 32 committed pprof profiles embedded `/Users/bthompso/…` and `main` — the *published* branch — carried every one; the 10 PNGs were clean. Ruled: delete the raw profiles. A profile measures one binary on one machine, so it was never reproducible evidence for anyone else; what it showed stays in `02-analysis/` and the capture method in `06-key_learnings/`. A note stands in their place. |
| Perf | The `[S]` window has no allocation pin |
| Perf | The radio-on frame is unpriced (measured ~104.5 MB/min) |
| Process | F-16 and F-17 in `follow-ups.md` were raised here; the pre-code gate and publish scrub are uncarried |

## 3a. Round 3 — the triaged pass (2026-09-01, 30-minute budget)

**Not a BUILD-exit round.** 0.14.0 holds for the read-lineup redesign (MVS-D-56), so anything the
Director rework will replace is deferred rather than fixed twice, and anything needing the HUM LEAD
to produce something for SHIP waits for them. What remained was triaged: user-facing functionality
first, data integrity next, everything else filtered down.

**Deferred — the redesign owns it.** F-5 (three cast roles unreachable), F-16 (parked offsets), F-17
(read order, now the design itself), the unpriced radio-on frame. All are ticker, reads or the audio
layer, and the Director rework replaces the surfaces they sit on.

**Deferred — HUM LEAD owed for SHIP.** The README screenshots (all 0.13.0, alt text asserting 0.14.0
content), `linux-validation-protocol.md`, F-6's M2 journey re-run, and M3's two listening trials —
which also settle the read-length assumption the redesign needs.

**Deferred — needs a live-feed measurement.** D-1, the zone-only alert outside a tracked area. The
record already says a count, not an estimate, decides it.

### A — user-facing functionality: **1 found; it opened a category-level gap**

| Finding | Verdict |
|---|---|
| F-2's "divergence is already live": `Classify` sent `Coastal Flood Statement` to no tab at all | **CONFIRMED and fixed (MVS-D-57).** Probed the classifier directly: `Coastal Flood Statement`, `Marine Weather Statement`, `Hydrologic Outlook` and `Air Quality Alert` all returned "not shown". The first two are products a forecast office issued about the listener's own location, and the app decided not to pass them on. Ruled: warning/watch/advisory decide first; any other statement goes to Spec. Statements; marine products go to Marine. `Special Marine Warning` stays a warning. Air Quality Alert and Hydrologic Outlook remain outside the taxonomy and still say so rather than being filed somewhere wrong. |

**Pulling that thread found more.** Probing the classifier product by product showed the gap was not
one product but a class of them, and three HUM LEAD rulings followed:

- **MVS-D-57** — warning/watch/advisory decide first; any *other* statement goes to Spec. Statements;
  marine products go to Marine. `Special Marine Warning` stays a warning.
- **MVS-D-58** — an **Air Quality Alert is an Advisory**, overturning v1's "not shown", which a test
  had been holding in place.
- **MVS-D-59** — a seventh category, **Forecasts and Outlooks**, for the products that are notable
  without being warnings. No ticker lane: the marquee is for what is happening. The window is renamed
  **NOTABLE EVENTS AND FORECASTS**, since it carries non-weather hazards and now non-hazards too.

Two things worth carrying forward from how that went. The gap was **invisible from the finding list** —
F-2 was filed as an architecture risk about classifier *count*, and the live defect underneath it was
that a forecast office had issued products about the listener's own area which the app decided not to
pass on. And the new tint would have **passed the AA gate by omission**: `contrast.go` lists the tints
it checks, and a token absent from that list is not checked at all. It is registered now.

F-2's *other* half — three independent classifiers over the same strings — is an architecture risk
with no remaining user-visible symptom, and drops to C.

### B — data integrity and hostile input: **4 found, 4 fixed**

| Finding | Verdict |
|---|---|
| F-7 `plaintext.Text` unbounded | **CONFIRMED, fixed.** The boundary now caps at `MaxTextRunes` (16384), counting runes so a multi-byte character is never cut in half. Every field reaching it is already capped upstream — but the boundary must hold on its own, because the next provider is written by someone who has not read that rule. |
| F-8 same-origin redirect compared `Hostname()` | **CONFIRMED, fixed.** The port is part of an origin: a redirect to a different port on the same host reached a different service carrying a keyed URL's path. `sameOrigin` compares host and effective port, so an implicit default and an explicit `:443` are still one origin. |
| F-9 debug server address unvalidated | **CONFIRMED, fixed.** `WATCHPOST_DEBUG_PPROF_ADDR=0.0.0.0:6060` bound pprof and the dump writer to the network. The override may now move the port, not the audience — anything but loopback falls back to `127.0.0.1`. |
| F-10 seen-store cap enforced only on load | **CONFIRMED, fixed.** `capOldest` ran once, when the file was read. A bound that is only checked at startup is not a bound on a process meant to run for weeks. It runs on every mark now. |

### C — light pass, carried

Checked for whether each was quietly fixed already, which was the point of the pass. **None was** —
every one is still exactly as reported, and one is worse.

| # | Status today |
|---|---|
| F-1 | **Still 6** `Save` owners over `config.toml`. Unchanged. |
| F-2 | **Still 3** classifiers over the same product strings. Its user-visible half is fixed (MVS-D-57); the architecture risk stands. |
| F-3 | `app/release.go` still declares no `snapshot.Provider`. Unchanged. |
| F-4 | Allocation pins exist for the frame, the severe window and Setup — **none for `[S]`**, the most expensive window. Unchanged. |
| F-11 | Unchanged; deliberate, and the `failureMemo` is still the only brake. |
| F-12 | **648 attribution stamps in non-test Go, not the ~200 reported.** Worse than recorded; still no live symptom. |
| F-13 | Unchanged — both go-studs workarounds still live in `status_table.go`. |
| F-14 | `TestRecastHandsOverMidSegmentAtTheSameSpot` still present (in `domains/radio/synth`, not `app` as the row implied). Unchanged. |
| F-15 | No `lint` target in the Makefile. Unchanged. |
| `uat-p4.md` | Still stale. |

None has a live user-visible symptom. Each is re-evaluated at the real BUILD-exit gate, after the
redesign — which is when the sweep for F-12 is worth doing, since the redesign will rewrite many of
the files carrying those stamps.

**Still uncarried and now noted:** the pre-code gate and the publish scrub have no follow-up row.

## 4. What this round cost, and why

**The instrument was wrong three times, in the same family each time.** A mutation sweep decided a
mutant was "caught" by grepping `go test` output for `FAIL`:

1. A mutation that breaks the build prints `FAIL … [build failed]` — read as caught.
2. A mutation that only fails `go vet` — read as caught.
3. A **stale baseline** (an un-updated declset) fails on every run — so every mutant read as caught.

About ten measurements were false and four reached commit messages as evidence. The corrections are
in `06_docs/remediation-review-loop.md`. The replacement is `06_docs/mutants/run.sh`, which requires
a green baseline, asserts the edit applied, builds and vets before running, and reports
`CAUGHT / SURVIVED / INVALID / UNAPPLIED`. The mutants are checked in beside it so a reviewer can run
them rather than believe a claim about them.

**Test fixtures were invalid five times** — invented CAP ids that fail the OID grammar, an alert
expiring exactly at `now`, a sweep measuring a 35-cell lane instead of the 2,563-cell one, a churn
model that could not distinguish the two rules it existed to compare, a severity fixture whose severe
event sat at index 0 where insertion order kept it regardless. Each passed while proving nothing.
This is why `remediation-review-loop.md` now requires fixture validity to be asserted before
behaviour.

**Work was layered.** Remediations were committed on top of unreviewed remediations, so reviews
arrived against a base that had already moved. The agreed discipline from here is serial: one defect,
one review, closed before the next opens.

## 5. Picking this up cold

1. Read `06_docs/remediation-review-loop.md` — the process, its failure modes, and the corrections.
2. `06_docs/follow-ups.md` — F-16 (parked offsets) and F-17 (listener-set read order) are the two
   items this round created; D-1 is the open decision on zone-only alerts outside a tracked area.
3. Defects **3** and **5** above are unstarted and were nearly lost; **5** is verified live in code,
   **3** needs verifying before anyone assumes it is closed.
4. Reviews owed: defect 2 (send it), defect 6 (never reviewed), defect 1 (four items in §2).
5. **The round has not concluded.** Round 1's verdict called for a convergence pass; this is it, and
   its own find rate was high enough that a round 3 is warranted after §1 and §3 are closed.
6. HUM LEAD-owned before SHIP regardless: M3's two listening trials (which also confirm the read-length
   assumption behind MVS-D-54) and the Linux validation protocol.
