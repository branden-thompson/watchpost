# Red team — BUILD exit (round 3)

**Scope:** `0e15e82..HEAD` — 224 commits, 402 files, ~8,300 production lines. Everything since
round 1, per the HUM LEAD's instruction ("cover everything since Round 1 — which was already
extensive").

**Method:** the canonical A2DH structure, dispatched **blind** — no lens was told what any other
was looking at, and none saw a prior round's findings list. Four always-on axes (Code Quality,
Project Hygiene, Docs Quality, Business Quality), the BUILD phase lens (Distinguished Engineer,
nine challenge areas), and five personas (InfoSec, A11y, Perf, Junior-Dev, Safety-Critical). Ten
lenses. The orchestrator verified every finding below against the tree before it was written down;
a lens's verdict alone is not evidence.

**Cost:** ~1.93M tokens.

**Verdict: NO-GO for the tag until the four Criticals are dispositioned.** Three of the four want a
HUM LEAD ruling, not just a patch. Nothing found here contradicts UAT — the product does what UAT
exercised. What the round found is hazard paths UAT cannot reach from the keyboard.

---

## The four Criticals

### C-1 — Muting one tone class silences every spoken alert, permanently, from the next launch

**Found independently by Business Quality and Safety-Critical.** The strongest convergence of the
round: two lenses, no shared context, the same five-line chain.

```
modes/tty/setup_tones.go:165   tick ANY class → toneMode = toneModeMute
platform/config/config.go:398  cfg.TickerMuted = cfg.Radio.Tones.Mode == toneModeMute   (unconditional, in Save)
app/dashboard.go:116           muted: tickerMuteState(cfg.TickerMuted)                  (seeded once, at launch)
app/ticker.go:283              if t.muted.Load() { return }                             (above the Arrived send)
app/executors.go:266           the second gate, same flag
modes/tty/dashboard.go:109     MuteTicker — declared, called nowhere; [M] now opens Settings (MVS-D-48)
```

MVS-D-26 is stated in four places in the tree — *"[M] mutes the TONES ONLY — the words always
read"* (`app/ticker.go:36`, `app/dashboard.go:150`, `platform/config/config.go:326`,
`app/radio.go:581`). The code does the opposite.

The flag cannot change at runtime: `muteHook` is the only caller of `flag.Store`, it is handed to
`tty.Config.MuteTicker`, and `MuteTicker` has no call site — the toggle went with `[M]` when
MVS-D-48 repurposed the key, and the hook was already dead at `0e15e82`.

**Scenario.** A listener finds the EAS dual-tone startling at night and sets ALERT TONES → Mute.
This session is fine. Next launch, `ticker_muted = true` is on disk. A tornado warning is issued.
It appears on the tape and in `[w]`. Nothing sounds — no tone and no words — and turning tones back
on in Settings writes the file but does not clear the in-session flag, so the listener's own fix
appears to do nothing.

`withToneCompat` (`platform/config/config.go:324-341`) applies the same derivation to every 0.13.0
upgrader who had `ticker_muted = true`, and its own comment promises them the opposite: *"They will
hear the words again; that is the accepted, CHANGELOG'd behaviour change."*

**Ruling needed.** (a) May a tone preference gate the read at all? MVS-D-26 says no — in which case
`ticker.go:283` and `executors.go:266` must ask a separate state that no tone setting writes.
(b) Wire `MuteTicker` to a live control, or delete it. (c) Stop `config.Save` deriving a
hazard-suppression flag as a side effect of a tone preference.

### C-2 — Read rank 1 can never be occupied: Emergency Orders are never spoken

**Safety-Critical, corrected by the orchestrator.** The lens claimed three categories were
unreachable; two of those are unreachable **by design** and the claim was wrong for them.

`platform/category/category.go:74` defines `Spec.Watchlist` — *"marks a category the national feed
cannot produce, whose rows arrive only through tracked locations."* Advisories and Statements both
carry it. They are correctly unreachable in a burst. **Emergency does not carry it**, so the
registry asserts the national feed can produce Emergency Orders. It cannot:

- `app/ticker.go:339` — `arrivalsOf` sets `Category: globalfeed.LaneOf(e)`.
- `domains/globalfeed/stack.go:77-88` — `LaneOf` has arms for Disasters, Marine and Watches, and
  defaults to Warnings. No arm for Emergency.
- `domains/globalfeed/nws.go:27-38` — the national query is a curated filter of eight product
  names. No civil-emergency product is fetched at all.
- `domains/severe/severe.go:117-121` maps `Evacuation Immediate` → `TabEmergency`, but those rows
  come from the tracked-location snapshots and reach the **tape** and `[w]` only. `startTakeover`
  has exactly one caller (`app/ticker.go:243`), with the national feed's `fresh`.

`platform/lineup/plan.go:65` asserts *"the read order leads with the emergency orders"* and
`selectBurst` exempts rank 1 from `Max` and lets it overrun the budget. That machinery is correct,
pinned, and unreachable.

**Scenario.** An `Evacuation Immediate` is issued for the listener's tracked location — the Weather
Service's highest-urgency product, an instruction to leave now. It is painted on the marquee's
Emergency lane and listed in `[w]`. The radio says nothing. The person that product exists for is,
by its own premise, not looking at the screen.

**Ruling needed.** Add the civil-emergency products to `severeEvents()` and teach `LaneOf` the
emergency family — which is F-2's fourth classifier, so it wants the shared classifier, not a
fourth copy. If that is too large for this cut, the minimum honest ship is `Watchlist: true` on the
Emergency row plus a note in `lineup-model.md`, because today `Ladder()`'s invariant reads as a
delivered guarantee.

### C-3 — BD-6's significance reach is implemented, pinned, mutant-guarded, and connected to nothing

**Distinguished Engineer.** The ratified exception — *"an M7.5 in Los Angeles (~120 mi) reaches a
listener with a 50-mile radius"* (BD-6, HUM LEAD 2026-09-02) — never fires.

`git grep ReachMi` over production code: the only assignment in the tree is
`platform/lineup/fence_test.go:30`. `arrivalsOf` (`app/ticker.go:337-350`) sets ten fields and not
that one, so `fence.go:160`'s disaster branch always compares `km <= 0`. Upstream it is fenced out
twice more: `scopeToRadius` → `scopeEvents` applies a bare `WithinMiles` with no significance
exception, so the distant quake is dropped by the producer before the Director sees it.

Two mutants guard the scale — `m83_reach_below_the_strong_threshold.py` and
`m87_reach_compared_in_the_wrong_unit.py` — and both live below the wire, mutating a function only
a test calls. Their CAUGHT verdicts are the m50/m42 pattern already recorded in the build log:
passing for a reason other than the stated rule. And `fence_test.go:30` builds the `Arrival` with
`ReachMi: QuakeReachMi(mag)` itself — **D-12 exactly**, a pin that does not start where the human
starts.

This is the T3.10 round's own headline ("THE WIRE IS NOT PINNED") recurring one layer up, in the
same release, on the hazard path.

**Mitigation, stated honestly:** `TickerRadiusMi` defaults to 0 = All, where the fence is not in
force. Only a listener who set a radius is affected.

**Ruling needed.** Wire it (populate `Arrival.ReachMi` **and** teach `scopeEvents` the exception),
or rule BD-6 out of 0.14.0 and mark `QuakeReachMi`/`ReachMi` unwired in the build log. The release
posture argues against the second. Either way the pin moves to start at `startTakeover`.

### C-4 — Every hazard active at launch is silently marked read

**Safety-Critical.** `app/ticker.go:238-242`:

```go
if !t.warm.Swap(true) {
    t.seen.mark(events, now) // the first cycle seeds current events quietly — no launch storm
    t.seen.save()
    return
}
```

The persistent seen store (7-day window) already prevents re-announcing across a restart. The
blanket seed therefore adds suppression only for alerts that appeared **while the app was not
running**.

**Scenario.** The listener closes the app at 2pm, or the machine sleeps. A tornado warning is
issued at 2:40. They relaunch at 3:00. It is inside its active window, inside the radius, and has
never been in the seen store. The first cycle marks it seen and returns. It shows on the tape; it
is never spoken, and `unread`/`Merge` filter it for ever.

**Ruling needed** on the first-run case. The launch-storm guard is only needed when the seen store
is absent or empty; seeding conditionally on that, or seeding only events older than the newest
seen timestamp, keeps the anti-storm property and stops swallowing the downtime window.

---

## Important — correctness and safety

| # | Finding | Lens(es) |
|---|---|---|
| I-1 | **The one escalation channel nil-derefs on a no-audio station.** `app/schedule.go:82` — `escalate: func(reason string) { deck.escalate(reason) }`, no nil guard, while *the same function* guards `if deck != nil` at `:110`. `radioDeck.escalate` (`app/radio.go:868`) checks `d.p == nil`, which dereferences a nil `d`. A nil deck is first-class (`app/dashboard.go:310-315`). `newExecutors`' `invariant.Check(x.escalate != nil)` passes — the closure is non-nil while the channel behind it is dead. The pump contains the panic, `Escalate` names no card so no `Failed` is emitted, and `onFault` writes to a log that is off by default. Net: on an audio-less machine a DR-21 fault silences the station and nobody is told — verbatim the outcome DR-21 exists to remove. `station_test.go:66` stubs `escalate` with an empty body, so nothing can observe it. | DE, Junior-Dev |
| I-2 | **`Failed` is overloaded — ordinary self-healing declines raise the RELAY FAULT modal.** `stopped()` is `held() == 0`, and in 0.14.0 the schedule is always empty after a rail card leaves, so the grade collapses to "did this card fail". Three of the five producers of `Failed` are deliberate non-deliveries (mute, an evicted record, an empty script). Pressing `[M]` between an arrival and its read pops a modal saying the relay is dead, with a 10s auto fall-through. `fault.go:6-10` names this hazard by name. | DE |
| I-3 | **The debug build does not compile.** `go vet -tags watchpost_debug ./app/` fails on `deck.breakers undefined` — my own T3.10b removal, with no gate covering that tag. | Code Quality |
| I-4 | **The relay-fault window is entirely inaudible.** `app/radio.go:234` and `:871` send `tty.RelaySilentMsg` and nothing else. The reason string goes to the player's *detail line*. A non-sighted listener of a weather radio hears audio stop, ten seconds of dead air, then a synth report beginning with no explanation — never told the relay failed, never told alternatives existed, never told a choice was made for them. | A11y |
| I-5 | **`Duck`/`Restore` are emitted by nothing in production**, so `mastercontrol.held` is never true, so `hold()`, `unhold()`, `takeBack()`'s held branch and `releaseBed()` are all inert — and MVS-D-67's one-duck-per-drain is unenforced. Those functions carry the heaviest comments in the release (the F-D5 critical-section explanation, the lock-order warning), all present tense, none saying the path is not taken. **F-27's ratified design is built on this path.** Also: `Duck`'s executor calls `x.mc.hold()` (the one owner) while `Restore`'s calls `x.voice.releaseBed()` — two owners for one pair. | Code Quality, Safety-Critical, Junior-Dev, DE |
| I-6 | **The closed-effect-set guard has a hole in it.** `TestTheEffectSetIsClosed` (`director_test.go:469`) enumerates eight effects and omits `Escalate`, which is live in `fault.go:23`. The test that exists to catch the set growing a member has already failed to notice one. `CardOf` and `Holds` have no test at all, and missing either loses the cue-before-words ordering (DR-18) or races the band. | Junior-Dev, DE |
| I-7 | **Nothing stops a queued burst re-reading alerts another card is already reading.** Dedupe exists at the seen store (marked line by line, as spoken) and at `Queue` (by card ID — and a different lead is a different ID). No ref-level check anywhere. Needs a read that outlives a cycle in Observer; **structural in Broadcaster**, where `OffAir` holds the rail and every cycle queues another overlapping burst. | Safety-Critical |
| I-8 | **A card's alert data is frozen at arrival and never re-checked before it is spoken.** `StaleAfter` bounds `BuiltAt` — the composed words — not the alert's own validity, and `firstStale` skips any card with a zero `BuiltAt`, which is every card still at `Admitted`. An alert that expires or is cancelled between arrival and air is read as live, with its original "until" time. Seconds in Observer; unbounded under an emergency overrun or Broadcaster's `OffAir`. | Safety-Critical |
| I-9 | **Every invariant violation in the pure core is discarded**, and `lineup.New` can fail into a Director that never steps and cannot be observed from outside. `invariant.Check`'s own test asserts the prefix exists "so logs and `[S]` can find them" — no caller in `lineup` does. Unreachable from the single call site today; **the second caller is Broadcaster**. | DE |

## Important — the record

| # | Finding | Lens(es) |
|---|---|---|
| R-1 | **The CHANGELOG is 205 commits stale and misdated.** `## [0.14.0] — 2026-08-31`; last touched by `ca96fd9`. Absent entirely: the alert rail and the spoken divert count, the relay-silent window, `ctrl+d` diagnostics, `[w]` pause/resume and `[esc]` stop, STANDBY, the tone tie-break. This is the document the release notes are cut from. | Docs, Hygiene |
| R-2 | **MVS-D-76 exists nowhere in `06_docs/`.** Zero hits. It is cited 18 times in code as a HUM LEAD ruling governing a whole new user-facing window — the five-second silence threshold, the three ways out, and an auto fall-through that retunes the radio without the listener. MVS-D-80 has the same shape in weaker form. `follow-ups.md:5` states the rule this breaks. | Hygiene |
| R-3 | **Four documents and three file headers say the Director is inert.** `app/schedule.go:6-10` ("no arrival reaches it yet… the executors are never called") — T3.10b wired `tick.emit = s.carry` at line 108 of the same file. `app/dashboard.go:211`, `app/executors.go:11-16` (declines by name the duck and the tune, both of which it performs), and all five "Today:" blocks in `role-model.md`. A Broadcaster author reads the header of the file that wires the Director into a SEV-0 weather radio and concludes it is dead. | Docs, Junior-Dev, DE |
| R-4 | **`docs/accepted-costs.md` is stale and self-contradictory.** Frame pins quoted ~20% below the tree (283–304 vs 365–386); *"The Director work is not implicated… no caller in the running app"* invalidated by T3.10b two days after it was measured; and the headroom correction ("1.3 tiers, not 1.6") sits one paragraph above the uncorrected "1.6 tiers". This is the register the standing rule says to read before optimising. | Perf, Docs |
| R-5 | **Three P10 ledger rows still name a gate that passed weeks ago**, and carry no ratification of any kind — all three added by 0.14.0. PL-14's corrected arithmetic ("six are pending") was itself short by three, and the build log's last line still carries PL-14 as open while `:2104` declares it closed. `release-checklist.md:10` is therefore false today. | Hygiene |
| R-6 | **An unresolved concurrency carry lives only in the build log.** `holdLine`/`resumeLine`/`dropHeld` touch the voice with no lock at all, not serialised against `dip`'s `duck()` and `takeBack`'s `restore()`, which are locked. No row in `follow-ups.md`, whose own preamble says a follow-up living anywhere else "is not carried, it is forgotten". On the duck path, with F-D5's history. | Docs, Hygiene |
| R-7 | **`narrate.go` is named in seven production comments and does not exist** — renamed to `director.go` at the P2 rename, tracked for the ledger, comments never swept. The only file of that name in the repo is a shell test fixture. `app/director.go:3` — the first line a newcomer reads in the arbiter — names it. | Junior-Dev |
| R-8 | **The employer path is back in three tracked, published docs**, reintroducing PH2-2 (fixed in `1d87831`). `release-checklist.md:20`'s guard already exists to catch it and fails on today's tree. `lint-watermark.sh:16` excludes `^06_docs/` entirely. | Hygiene |
| R-9 | **A vacuous-invariant cluster that contradicts the quality plan's measured "vacuous invariants: 0".** Seven production sites with no possible failing input (D-2), four test assertions that cannot fail, and two fixtures that do not pose what they claim — `fault_test.go:36` never gets a card to OnAir, so its "a card still on air" row is never posed; `plan.golden` claims to span both bands and the overrun boundary and every row reads `CLOSE`, rungs 1-5. The instrument that produced the 0 should be re-examined, not just these sites. | DE |
| R-10 | **`make p10` and `make pty-severe` are not in CI.** `Makefile:88` says of p10 "this gate cannot be skipped"; CI skips it. F-D3 already measured the cost — the severe-window pty smoke was red since the rename and found by hand. | DE |
| R-11 | Two more D-12 residues on safety paths: `schedule_test.go:271` still nil-checks `deck.emit` (the ticker half was remediated with a real drive; the deck half was not), and PD-3's stale drop is never driven through `Step`. **And the composition "a rail card fails → a person is told" has no app-level test** — the three pieces exist and are never joined, which is what let I-1 and I-2 through. | DE |
| R-12 | Three of ten cross-package `file.go:NNN` citations point at the wrong code — including `power.go:91`, the evidence offered for why the alert rail is exempt from Stop, which sends the reader to a fence function. | Junior-Dev |
| R-13 | `docs/where-things-happen.md:4` promises the page "cannot drift silently"; the checking regex requires `.go`, so 12 of 163 symbol refs are invisible to it — and both entries rewritten this release used the unchecked form. | Docs |
| R-14 | `memo.go:24` and `:145` claim two complete invalidation registers. `bodyKey` has 22 fields and 15 rows; `modalKey` has 34 and 27. The nine missing include `faultFocus`, `faultLeft`, `debugFocus`, `severeReadPause` — **the exact four whose absence froze three windows this release.** | Docs |
| R-15 | `platform/render/theme.go:79,94` say "COLOUR PENDING A HUM LEAD RULING" for colours MVS-D-62 ruled and four themes ship. | Docs |
| R-16 | `follow-ups.md` F-30 is carried as open; the test shipped this release (`memo_completeness_test.go`, headed "F-30, mechanised"). A 0.15.0 reader would build it twice. | Docs |
| R-17 | The M3 EAR gate has no evidence file and `07-readiness/validate/` does not exist; every box in `release-checklist.md` is unchecked; F-6, F-28 and F-29 are all due "Before SHIP" and open. | Hygiene |

## Important — performance

| # | Finding |
|---|---|
| P-1 | **`advanceTicker` rebuilds the entire 30-item tape every 300 ms to produce one integer** (`d.tickerScroll %= n`) — 68 allocations per tick, 45% of `advanceTicker`, ~3.2 MB/min. Invisible to the gate: `TestFrameAllocBudget` measures `View()` only, so the tick path is the one per-300ms cost in the app with no budget on it. Lever: cache the loop length beside `tickerScrolls`. |
| P-2 | **`Script.Empty()` is `Text() == ""`, and `Text()` builds with `out += p.Text` in a loop** — quadratic string building to answer a boolean, called from invariants on the `settle()` path of every event including every 1-second tick. Two lines. |

## Important — accessibility

| # | Finding |
|---|---|
| A-1 | **The `ctrl+d` window's tail is unreachable by any key at 24 rows, at every width, in a release build.** All nav routes to `handleDebugNav`, which returns unchanged when `debugScenarios()` is empty — which is every release build. The scroll rail draws ▲/▼, so the app tells the reader there is more, and ↓/PgDn/End/j all leave `scroll=0`. What is lost is the block explaining *why the window looks empty*. Same defect class the relay-fault window fixed and pinned this release, in the window next door. |
| A-2 | **`debugContent` is a fixed 84 and does not shrink**, so `floatModalFooter` re-wraps at 66-69 and the prose double-wraps into fragments at 80 columns — verbatim the hazard `relayfault.go:56-65` documents and fixed for its own window. Eight ragged fragments at 80. |
| A-3 | **The 10-second auto-close is a hard timeout on the app's only station-tuning control** — `cfg.TuneRelay` has exactly one caller in the tree. No extend, no pause, no reset on keypress, no second route. WCAG 2.2.1 (Level A) fails outright. For a slow reader, a magnification user or switch input, the outcome is always Fall-Thru and the other two rows are decorative. **HUM LEAD call** — this is the round-2 "a technicality is not an accommodation" shape. |
| A-4 | **Two spoken scripts say "press W in Watchpost"; `W` does nothing** — the binding is `"w"`. For a non-sighted listener that spoken line is the only instruction they ever receive about reaching the diverted alerts. One line either way: add `"W"` to `Keys`, or reword. |
| A-5 | Non-ASCII furniture survives `--ascii` in four windows that have no `--ascii` scan: `│` ×10 and `─` ×98 in Location Details (the most-opened modal), `⚠` in Alert Details, `—` in `ctrl+d` — which is on the golden test's own forbidden list. The glyph set already has every fallback needed. |

## Important — security

| # | Finding |
|---|---|
| S-1 | **The injector is confirmed absent at the artifact level** — built the default binary, `nm` shows zero `Inject` symbols, `strings` zero scenario labels. The P10-08 exemption holds. **But no gate checks this.** `inject_test.go` asserts on the *text* of two source files; nothing inspects the produced binary. A `GOFLAGS=-tags=watchpost_debug` in the runner env passes today's guard. One step in `release-matrix`. |
| S-2 | **`WATCHPOST_DEBUG_PPROF_ADDR` defeats the documented "loopback only" property.** `debugAddr()` returns any env-supplied address verbatim, so `0.0.0.0:6060` binds every interface — three unauthenticated routes, one of which writes profile sets to disk on a GET. This is a whole HTTP server gated only at runtime in a release binary. Clamp to a port, or move `debug.go` behind the debug tag. |
| S-3 | The on-disk response cache is a fabrication path of identical fidelity in every release build, with no build tag — a hand-written alerts document at `sha256(URL).cache` crosses the same stages the injector crosses. It takes the user's own privileges, so it is not a hole; the exemption's claim is true as "ships no fabrication *feature*" and false as "a release build cannot fabricate an alert". Record the distinction so the next reviewer does not re-derive it. |
| S-4 | `config.Path()` accepts a relative `XDG_CONFIG_HOME` — the exact failure S-F5 already fixed for the Piper binary, on the tree that decides what the radio says. One `IsAbs` check. |

## Minor — recorded, not blocking

Junior-Dev's naming cluster (`settle` is a method on both directors, one call apart; `hold` means
five things in `package app`; `report` is a callback, a card kind and a filename parameter);
`director.go:150` marks `Tune` "Not emitted yet" and it is emitted twice; `card.go:255` still lists
a retired `BurstHead`; `executors.go:98`'s emphatic `cutTo`/`tune` warning protects a distinction
T2.3 removed; lock discipline stated on four of eight helpers that require it in `app/director.go`;
`kmPerMi` declared twice behind a comment that over-argues a D-1 exception it does not need;
`worstOf` panics on an empty slice; `alertStore.capOldest` panics if its two maps ever desync and
nothing asserts they cannot; `modalVoice` is dead and makes the F-30 guard report a false coverage
hole (12 pass, 1 skips — an enum member that should not exist); the fault window steals whatever
window is open, including `[w]`; `Escalate.ID` **and** `Escalate.Reason` are both confirmed
write-only to the listener; 20 git worktrees and 18 stale branches against a checklist requiring
the main tree only; two orphan published images and one checklist deliverable never produced; F-21
names two unrelated things across six documents.

---

## Convergence map

Convergence is the strongest signal the method produces — independent lenses, no shared context,
the same defect.

| Defect | Lenses | Weight |
|---|---|---|
| Tone-mute → permanent hazard silence | Business Quality, Safety-Critical | **2 — and it is the round's Critical** |
| `Duck`/`Restore` unreachable, MVS-D-67 unenforced | Code Quality, Safety-Critical, Junior-Dev, DE | **4** |
| The Director is documented as inert | Docs, Junior-Dev, DE | 3 |
| CHANGELOG stale and misdated | Docs, Hygiene | 2 |
| `accepted-costs.md` stale | Perf, Docs | 2 |
| `holdLine`/`resumeLine`/`dropHeld` never reached the ledger | Docs, Hygiene | 2 |
| Escalation channel nil-derefs on a no-audio station | DE, Junior-Dev | 2 |
| Closed-effect-set guard has a hole | Junior-Dev, DE | 2 |

## The shape under it

**Enumerated guards with holes.** `TestTheEffectSetIsClosed` lists eight effects and omits
`Escalate`, which is live. `bodyKey`'s invalidation table has 15 rows for 22 fields; `modalKey`'s
has 27 for 34 — and the nine missing include the exact four whose absence froze three windows in
UAT this release. F-18's token register and F-30's memo key are the same class, both already known,
both answered by derivation. That is three more instances, one of them on the architecture's own
stated interface.

**The wire is not pinned — again.** C-3, R-11 and the `Escalate.Reason` half of R-3 are one defect:
*a rule implemented and pinned in one layer, not carried by the layer that would deliver it.* The
T3.10 round named this and fixed five instances. These are three more, in the same release, found
by asking "who writes this field, who reads this value" — not by reading tests.

Both shapes have the same general answer, and it is mechanisable: **a producer/consumer
completeness check over the closed sets.** Every `Effect`, every `Event`, every `Arrival` field and
every `Slot` must name a production writer and a production reader, or carry an explicit
"unwired, and why" entry. Written once, it subsumes F-18, F-30, C-3 and I-6.

**Three carried items became due at this cut.** F-22 (nothing enforces the band's one-owner rule —
*"before the Broadcaster adds a second writer"*), F-30 (answered, but its row is stale), F-18
(`aaPairs` shrinks silently). All three name Broadcaster as their trigger, and this release is
being cut **to start Broadcaster**. They are 0.14.0 exit conditions now, not 0.15.0 backlog.

## What the round did not reach

Stated plainly, because a coverage claim is worth less than a coverage gap.

- **No lens ran the test suite** (CPU discipline). Every finding is a reading claim with a citation;
  none is a measured failure. InfoSec built one binary and inspected it; A11y rendered modals;
  Hygiene ran two `-run`-scoped tests. That is the extent of execution this round.
- **The mutation corpus was not run.** 174 files, read only where a finding pointed. So no lens can
  say whether an existing mutant should have caught I-1 through I-9. For C-3 the answer is no, and
  that is the finding.
- **`platform/lineup/` — the largest new package (~2,600 non-test lines) — is only partly audited.**
  `director.go` (830 lines), `lineup.go`, `fence.go`, `stale.go`, `script.go` and most of `card.go`
  and `plan.go` were read in outline or at the seam.
- **`app/mastercontrol.go` was read by no lens in full** — the one owner of the band and the duck,
  and the whole F-D5 / MVS-D-67 history. `app/director.go`'s arbiter beyond `speaker` likewise.
- **`domains/radio/`** — synth, cast, player, pronounce, the relay path and the PCM cache. Round 2's
  defect 6 was "remediated, never reviewed", and it still is.
- **The 08-reports bundle was not diffed for cross-document consistency**, and the disposition of
  every finding in the five prior reports was not re-walked. Two of this round's findings (R-8 and
  the published `/Users/bthompso` paths) are re-openings hit by accident while grepping, which
  suggests a full disposition re-walk would find more.
- **No image was opened**, so the stale-screenshot finding rests on an empty `docs/img/` diff.
- **Windows and Linux paths** were reviewed by reading only.
