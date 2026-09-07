# Risk register — Station Director & the Lineup

**DISCOVER, 2026-09-01 · SEV-0 · HUM LEAD.** Companion to
`01-objectives/director-requirements.md`. Severities are about **what reaches a listener**, not about
effort.

Most of what follows is measured from this release rather than imagined. That is deliberate: the
useful risks here are the ones that already happened once.

## Likelihood × impact

**Added after the PLAN red team (PL-9):** the register rated impact alone, so "HIGH" conflated
*certain and survivable* with *unlikely and fatal*. Likelihood is judged from this release's own
record where one exists, and marked as judgement where it does not.

| # | Likelihood | Impact | Basis for the likelihood |
|---|---|---|---|
| RD-1 | **High** | High | Happened twice on this code, ~30 defects |
| RD-2 | Medium | **High** | The duck-lift bug is one occurrence; the rule is about to move again |
| RD-3 | Medium | **High** | Approach C removes the deadlock class but PL-1/PL-2 were its first two instances — found at PLAN, not at BUILD |
| RD-4 | **High** | Medium | **Recurred during this very phase** — see the note below |
| RD-5 | Medium | Medium | New behaviour on a safety path; judgement |
| RD-6 | Low | Medium | Needs an emergency count Observer rarely produces |
| RD-7 | Medium | Low–Med | Six Broadcaster concepts were live in the rulings; PD-1 caught them once |
| RD-8 | Medium | Medium | Two content-destroying bulk edits already this release |
| RD-9 | **Low** | Medium | **Downgraded** — PD-2 is now settled by measurement, not judgement |
| RD-10 | Low | **High** | Low chance, silent failure, and the registry exists to prevent exactly it |
| RD-11 | Low | Low | Deferred discovery, no safety component |
| RD-12 | Low | Medium | Only if this work touches the parking behaviour, which it is told not to |

## High

### RD-1 — This exact code has a measured remediation failure rate

**Evidence.** Two red-team rounds on the ticker/narration path closed 17 findings and opened roughly
**30 new defects**, every one of which passed `make verify`, the allocation pins and the goldens.
Four claims in commit messages were disproved by later review
(`06_docs/remediation-review-loop.md` §Corrections).

**Why it recurs.** The failure modes are all *semantic* — fixed the finding not the defect, fixed one
of N paths, test written to match the fix, asserted what the code does not do. Gates cannot see any
of them.

**Mitigation.** The remediation review loop is mandatory on this path: one finding at a time, failing
test first through the real entry point and **watched to fail**, mutate your own change with
`06_docs/mutants/run.sh`, then a fresh adversarial reviewer who must run something. Carry the
previous review's mutants forward verbatim as a regression set.

**Residual: HIGH.** The loop roughly doubles the cost per finding and does not eliminate the class.

### RD-2 — Absorbing the watchlist rotation changes audio with UAT history

**Evidence.** `radioDeck.armDwell` / `advanceQueue` / `stopDwell` carry rulings from UAT 83, 93 and
97, and one live bug fixed there was subtle in exactly the way this refactor invites: the exported
`Tune` lifted the alert duck and the unexported `tune` did not, and the Watchlist advance called the
wrong one — *"a rule that lives in the case of an identifier is a rule waiting to be missed"*
(`app/radio.go:121-138`). A listener heard the next location's report come up at full volume over a
breaking alert still being read.

**Mitigation.** Enumerate every consumer before moving anything (ratified refinement R-B). The duck
must keep exactly **one** owner across the move. Pin the UAT-93 advance behaviour with a test through
the lineup's entry point *before* the absorb, so it can fail.

**Residual: HIGH.** This is the single most defect-dense file pair in the release.

### RD-3 — Four goroutines touch this today; a single-writer Lineup serialises them

**Evidence.** The ticker cycle goroutine, the takeover goroutine (`t.breakers`), the engine's status
callback (`onStatus`, which spawns `startSynth` and `advanceQueue`), and the bubbletea update loop
all mutate or read state this design consolidates. `radioDeck` already carries three separate mutexes
(`mu`, `tuneMu`, `installMu`) plus an epoch counter to make "check then act" atomic.

**Risk.** A single-owner Lineup is the right shape and is also where deadlock and dropped-command
bugs live. A command that blocks the owner blocks the schedule; a command dropped on a full channel
silently loses a read.

**Mitigation.** Name the owner's concurrency model in PLAN explicitly, with the failure mode of each
alternative stated. Run the suite under `-race`. Assert liveness, not just correctness: a test that
the owner survives a command arriving during every phase of a takeover.

**Residual: HIGH until the PLAN option is chosen.**

### RD-4 — Measurement on this path has been unreliable

**Evidence.** The mutation sweep decided a mutant was caught by grepping `go test` output for `FAIL`
— and a mutation that breaks the build prints `FAIL … [build failed]`. **Roughly ten measurements
were false and four reached commit messages as evidence.** Separately, five fixtures passed while
proving nothing: an invented CAP id the OID grammar rejects, an alert expiring exactly at `now`, a
sweep measuring a 35-cell lane instead of 2,563, a churn model that could not distinguish the two
rules under test, and a product NWS does not issue.

**Mitigation.** `06_docs/mutants/run.sh` replaced the instrument: green baseline required,
edit-applied asserted, build and vet gated, verdicts `CAUGHT / SURVIVED / INVALID / UNAPPLIED`.
**Assert a fixture is accepted by its validator before asserting behaviour** — the most repeated
mistake in the record.

**Residual: MEDIUM impact, HIGH likelihood — and it recurred during PLAN.** The card-build instrument
(P-1) built its httpx client without setting `RatePerSec`, taking the default of 5/sec where the app
uses 30, and so reported **2.14 s** for what is actually **1.03 s**: it timed its own
misconfiguration. The figure was published in four documents and PD-2 was decided on it. Caught by
this phase's own red team (PL-17), not by a gate.

**What that says about the risk.** The failure was not the fixture this time — it was the
*instrument's configuration*, a category the anti-vacuity rules did not name. The rule they were
missing, now added: **an instrument must be built the way production is built, and the constructor
should carry the reason so the next reader cannot repeat it by omission.** The tell was present in the
data all along — three samples within 0.1 s of each other is a token bucket in lockstep, not a
network — and it was read as reassuring consistency.

## Medium

### RD-5 — The significance fence is new safety-path behaviour, landing inside a redesign

`scopeToRadius` / `scopeEvents` is today a flat fence with one exception (zone-only alerts tied to a
tracked location, open decision D-1). DR-13 adds a magnitude-scaled reach for disasters. It is
genuinely wanted and improves Observer now (R-3), but it changes **what the listener is told about at
all**, and it arrives alongside a structural rewrite, so a defect in it will be attributed to the
rewrite.

**Mitigation.** Land and verify it as its own commit with its own tests, before or after the
structural work, never inside it. Its fixtures must pass the classifier and the active-window check
before their distances are asserted.

### RD-6 — Emergency Orders are exempt from Max

The ruling is correct on safety grounds and is the HUM LEAD's (R-2). Two consequences follow.
**Technically**, it is the one read with no constant bound, in a codebase that enforces P10-02 — it
must be written as a bound derived from the feed's own size, or a later reader will "fix" it with a
cap and silently reintroduce the defect this feature exists to remove. **For the listener**, twelve
in-radius evacuation orders is a read of several minutes with no way to shorten it.

**Mitigation.** Write the bound as data-derived and say so at the site. Surface the length in the M3
listening trials rather than discovering it live.

### RD-7 — Broadcaster scope leaking into 0.14.0

The rulings describe operator commands, card editing, ON AIR / STANDBY, callsign and service radius.
0.14.0 ships Observer. PD-1 exists precisely because the boundary is not obvious, and the honest
answer may be "some of it must be stubbed".

**Mitigation.** PD-1 gets a written recommendation with reasoning at PLAN, and the HUM LEAD rules.
Nothing Broadcaster-shaped is built on the agent's own judgement.

### RD-8 — The rename touches many files, and bulk edits have destroyed content twice this release

**Evidence.** A range slice from `| F-14 |` to `| F-15 |` deleted **F-16, F-17 and F-18**, which sat
between them. A mid-line `//` silently commented out `Product`, `Location` and `Detection` in
`toSevereRow`, and only one of the three had a test. A blanket `sed` renamed `.Label` on an unrelated
type.

**Mitigation.** Assert the affected symbol set before and after any bulk rename. Compile is necessary
and not sufficient — two of the three above compiled.

### RD-9 — PD-2 unresolved can produce an audible regression either way

Standby rendering removes a cutover gap and adds discardable work on a limited render budget
(`synth.Limiter`, reserved slots). Line-by-line keeps today's behaviour and leaves the charter's
analog approximated rather than implemented. Choosing wrong is audible.

**Mitigation.** Measure the current gap before deciding. It is measurable today via
`WATCHPOST_DEBUG_RADIO` segment timings.

## Low

### RD-10 — Three per-category orderings invite a later "harmonisation"

Tab order, ticker rotation, and now read rank are all legitimately different and will all live in
`category.Spec`. A future reader who tidies them into one changes what is spoken, silently — which is
the exact class `platform/category` was built to end.

**Mitigation.** DR-10's test asserts the three are independently declared, so collapsing them fails
loudly.

### RD-11 — "Text at standby" is validated only when Broadcaster is built

DR-7 is the right call for staleness, but the Operator's queue then shows cards without text until
they near the air. Whether that is acceptable is a UX judgement nobody can make until the Broadcaster
UI exists.

**Mitigation.** Recorded as a known deferral rather than an assumption. Revisit at Broadcaster.

### RD-12 — This feature touches the ticker, and F-16 is unfixed there

F-16 (a lane's parked scroll offset hides a new headline until the tape wraps) is explicitly **not**
to be fixed here — two attempts, two reverts, and every fixture written for it was invalid. But this
work edits the same file.

**Mitigation.** Leave the parking behaviour untouched and say so in the commit. If a change makes
F-16 worse, that is a blocker; if it makes it better by accident, it still gets F-16's own DISCOVER.

## Dependencies

| On | Status |
|---|---|
| `platform/category` registry | **Done** — F-21 closed 2026-09-01. Read rank is a new `Spec` field. |
| F-18 AA register | **Open.** This feature will add render tokens; each must be registered or it is never measured. |
| D-1 (zone-only alert outside a tracked area) | **Open**, and needs a count from the live feed rather than an estimate. It interacts with DR-13's fence. |
| M3 listening trials | **Owed by the HUM LEAD.** They settle the read-length assumption that RD-6 and the Max default both rest on. |
