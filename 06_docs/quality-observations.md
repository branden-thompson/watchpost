# Running observations — what is working, what keeps happening

**Purpose (HUM LEAD, 2026-09-03):** *"Keep making notes on what's working well, what we're
encountering, because I think this all should be codified as skills or a skill family so future
projects on a canonical A2DH install can benefit."*

This is the EVIDENCE LEDGER that feeds that codification. A rule earns its place here by having
caught something, with the occurrence named. A rule that has caught nothing in two releases is
retired rather than kept for completeness — the list is meant to shrink.

**Status:** `defect-classes` (D-1, D-2, D-5, D-11) is a skill already. The rest of this document is
what has not yet been extracted.

---

## What is working, with the catch that earned it

| Practice | Caught | Cost |
|---|---|---|
| **D-11 — validate the pin against its own defect** | **Three false-green pins in two tasks.** A probe that never ran before the operation finished; the same probe passing on ONE processor; and a tone test structurally incapable of failing on the cross-lane case. All three were green | seconds — revert, run, restore |
| **The corpus guard** (`mutant-check`) | Mutants going stale or INVALID from my own edits, every task. m23/m24 **retired** (their rules were deleted), m49/m22 **re-anchored**. m22 had silently become INVALID — it deleted a rule AND broke the build, so its verdict was no evidence either way | ~2 min per run |
| **Escalate on survival** | mF1's `SURVIVED` was a SCOPE artifact — run against `./app` while its pin lived in `domains/radio/player`. Escalating caught it | one re-run |
| **Fresh review, tightly scoped to one commit** | F-D5 (a lock-order twin nobody was looking for) and the MVS-D-12 tone regression. **Both were found in minutes** | ~10-20 min |
| **Test first, watched to fail** (Phase 3) | The T3.1 guarantee — written, watched failing on the real defect, then fixed | minutes |
| **D-8 — derive, never recall** | Two wrong numbers in a gate table, self-caught | seconds |

**The headline: minutes, not hours.** The 2026-09-02/03 session spent hours per defect because the
instruments were themselves defective. The same class of defect now surfaces in minutes because the
instrument is checked before it is believed.

---

## What keeps happening — the recurring shapes

Candidates for codification. Each is stated as the SHAPE, not the incident.

### 1. A pin is weakened by the very change it would have caught

**Twice now.** Routing a test through a new code path left it with the one input where the old and
new orderings coincide — green, named for the rule, structurally unable to fail on it. The mB5
finding was the same shape.

**The tell:** you edited a test in the same commit as the behaviour it guards, and the edit made it
*pass*. That is the moment to run it against the defect.

### 2. An instrument measures a PROXY that silently stops being faithful

`cueVoice` logged `render` as "the words". DR-18 is about when words are HEARD; render and play were
adjacent, so the proxy held — until a change rendered ahead of time and the proxy broke while the
contract did not. **A pin failing is not proof of a defect: the instrument is a suspect too.**

**The discipline that made this safe:** correcting the instrument, then re-validating it against the
defect it exists for (m49 still CAUGHT). Otherwise "fix the test" is indistinguishable from
"weaken the test".

### 3. A refactor turns a mutant from a rule-deleter into a build-breaker

m22 deleted the seen-mark; a change elsewhere left that mark as the only reader of a variable, so
deleting it stopped compiling. **The mutant still "existed" and still "failed" — it just stopped
being evidence.** Only the corpus guard's applies-and-compiles check separates those.

### 4. Safety machinery that defends the clean exit and not the interrupted one

`run.sh` restores through a shell trap. A foreground command timeout KILLED it mid-mutation and left
the mutation in the tree. Same family as the trap-armed-above-the-clean-check bug. **Ask of any
cleanup: what happens if this is killed rather than returning?**

### 4b. The safety harness destroys uncommitted work WHILE it runs

Distinct from the killed-run hazard above, and sharper. `run.sh` restores the tree by checking out
from git — **the whole tree, not the file it patched**. So any edit made while a sweep is in flight is
silently deleted, on the harness's HEALTHY path. An implementation, its test and a new exported
method vanished mid-task; the only symptom was a test that had passed a minute earlier reporting
"no tests to run".

**Three rules, and the third is the one a near-miss added.** Commit before starting a sweep; never
edit into a running one; and **never `git add`/`commit` while one runs** — the tree may hold a LIVE
MUTATION, and `git status` showing a modified file looks identical whether it is your work or the
harness's. A commit taken at that moment would have committed the defect a mutant exists to detect,
under a message describing the fix. What caught it was reading `git diff` on the modified file rather
than assuming a file recently touched was modified BY ME.

**The durable fix is not a discipline.** `run.sh` should restore only the paths it PATCHED. Then work
elsewhere in the tree is never at risk, and exactly one file is ever ambiguous — which is a property
rather than a rule to remember. Recorded as the standing recommendation; the rules above are what
holds until it exists.

### 4c. A guard cannot be pinned by a mutant, and the attempt looks like a finding

`mH1` mutated a TEST — weakening a guard's loop from walking the type back to a
hand-written list — and reported SURVIVED. It always would: nothing in a suite fails when a test gets
weaker, because tests do not test tests. The verdict looked like "this behaviour is unpinned", which
sends a person hunting for a missing pin that cannot exist.

**This is the withdrawn D-4 in concrete form** — *mutate a RULE, never an ASSERTION* — and it is worth
keeping precisely because the rule was withdrawn as unmechanisable and then broken by its own author
within the hour. The triage that catches it is already written: **a SURVIVED mutant whose edit lands
only in a test is not evidence, it is an invalid mutant.** Retire it rather than chase it.

**The real remedy for a guard that can drift is to move the assumption into the PRODUCT** — a sentinel
the type carries, an invariant the code checks — so a mutant has a rule to delete. A guard that lives
only in a test is protected by review and by D-11, not by mutation.

### 4d. A hot function collects mutant anchors, and refactoring it costs a re-anchor each time

One read loop needed **six re-anchors in a single task** — m22, m49, m50, mG0, mG2, mG3 — every one
caused by a restructure, none by a behaviour change. The cost is real but it is bounded and it is
CHEAP, because the corpus guard names the file in about two minutes. Without it the same six would
have surfaced as a sweep quietly reporting fewer verdicts than it launched.

**The lesson is in the anchoring, not the refactoring.** A mutant anchored on a multi-line block
breaks whenever anything is inserted into that block — mG0 anchored on `for … {` plus the line after
it and went stale the moment an air check landed between them. **Anchor on the smallest line that is
distinctive to the rule**, so an unrelated insertion nearby does not invalidate it. A mutant that goes
stale from a change it has nothing to do with is a maintenance tax with no coverage in return.

**AND AN ANCHOR THAT QUOTES AN EXPRESSION BREAKS WHEN THE EXPRESSION IS NAMED.** Consolidating five
copies of `j.ctx.Err() == nil` behind a `live(j)` helper — the exact refactor D-1 asks for — went
stale in eight mutants, none of whose rules had changed. The refactor you most want to be safe is the
one that breaks the most anchors, because every anchor quoted the thing you just gave a name to.
Anchor on the surrounding STRUCTURE (the loop, the branch it guards) rather than on the condition,
and a rename leaves it standing.

### 4e. Work moved onto the UI's own goroutine, without asking what it does there

Two defects in one day from the same seam, arriving from opposite directions. Bubbletea's `Update`
goroutine is the one that drains the message channel and paints the frame, so anything running there
must NEITHER BLOCK NOR SEND:

| What was put on it | What the listener got |
|---|---|
| a WAIT for a goroutine to finish | half a second of visible lag on `[esc]` |
| a SEND into the program's channel | a hard softlock — no key worked, terminal killed |

**Neither was a wrong decision about the behaviour.** Both were the right operation on the wrong
goroutine: cancelling a read on close is correct, and clearing the row's mark when it is cancelled is
correct. The defect in each case was WHERE it ran.

**The tell is cheap and I missed it twice:** before adding a call to a UI handler, ask what goroutine
the handler runs on, and whether the call can block or send. In this file every other send already
ran on a background goroutine — the pattern was there and I broke it without noticing.

**And the pin for a deadlock is a HANG, not a count.** `TestTheClosePathNeverSendsAndSoCannotSoftlock`
gives the reader a send that never returns, which is what a full channel looks like from the caller,
and requires the close path to return anyway. Counting sends would have passed while the deadlock
remained: the count right and the goroutine wrong.

### 4f. Fixing the reported CASE instead of the class — the same defect three times

"Where can a read be?" has exactly three answers: on air, suspended by a takeover, waiting behind
one. I answered it three separate times — the original fix, review round 1, review round 2 — each
prompted by a reviewer showing me the next one, and each time wrote a fix and a pin for THAT case
only.

**Enumerating the state space once costs minutes. Discovering it a case at a time cost three review
rounds**, and round 1's partial fix shipped something worse than the bug: it claimed a cancelled
read's corpse and reported success, so the chip said "Paused" with nothing held, 100 times out of 100.

**The tell:** a fix that names ONE state. `if job := d.onAir; …` should have prompted "and the other
places a job can be?" before the fix was written, not after a reviewer asked. D-5 is the nearest
existing rule — it makes an ordering name its scope — and this is the same discipline applied to
STATE rather than to time.

**Two smaller shapes from the same stretch:**

- **A flaky pin was telling the truth.** One failure in three was an ordering defect, not machine
  noise: the mark was sent AFTER the goroutine that could clear it. The right response to a flake is
  to find the cause; a retry loop would have buried a real bug.
- **A pin can be racy in the FAILING direction** — green against the defect about half the time, which
  is a pin reporting clean on broken code. D-11 covers pins that measure the wrong thing; this is one
  that measures the right thing unreliably. Assert on the predicate directly rather than driving it
  through a goroutine whose unwind you are racing.

### 5. One domain mapping reused for a second concern

`globalfeed.LaneOf` is the TAPE lane, and it was reused as the READ category — which put hurricanes
at read rank 7, below every land warning. The two questions look identical and are not.

**It recurred within the same task**, which is what makes it a shape rather than an incident: the
burst's TONE needed an ordering, and three plausible ones already existed — the Setup checkbox order,
the read order, and the tone's own loudness. Only the third answers "which sound says listen now", so
`cast.Class.ToneRank` was written as a new carrier (MVS-D-73) rather than borrowed. **The tell is a
sentence with two nouns in it:** "sort the tones by read order" is two concerns wearing one list.

### 5b. A gate that resolves by NAME reports a collision as a defect

P10 resolves methods by name, not by receiver. Adding `eventReader.Cancel` as
`eventReader.Stop` made it collide with `radioDeck.Stop` — which genuinely sits in a `stopDwell`
cycle and carries its own ratified exemption — and the gate reported the new function as recursion it
has no part in.

**Three renames in one release** for this: `executors.cue` and `executors.release` at T2.3, and this
one. Each time a rename was cheaper and more honest than an exemption for something that is not
recursive — an exemption would have recorded a false claim in the ledger permanently.

**The tell:** a P10-01 recursion finding on a function with no call cycle you can trace. Check whether
a same-named method exists on another type before reaching for an exemption. **Upstream item:** the
finding should name the receiver, or the collision is indistinguishable from the real thing.

### 6. The tool that cannot say what it can do

Two `a2dh` builds both reported `v1.17.0`; only one had `p10`. The gate's failure message said
"live findings" for a missing subcommand, so a stale toolchain read as a code failure. **A gate
wrapping an external binary must distinguish "cannot run" from "found something".**

---

## The review loop: a conditional change, with its trigger stated

**RULING (HUM LEAD, 2026-09-03):** *"Let's continue with the current process for the next set of
tasks — if it turns out the pattern holds, we can update the process so that guard fixing is a
focused 'meta round' with a scoped reviewer specifically looking for those things vs. full
adversarial review of the entire surface area."*

**The pattern this is watching for.** T3.1/T3.3 took four rounds. The behaviour was correct after
round 2; rounds 3 and 4 found only problems with the GUARDS — a guard that could not catch the case
it was written for, a fix with no coverage, a stale doc, and one more door into a guard gap. Each was
real work. Each also cost a full adversarial pass over the whole surface to find something confined
to the instruments.

**So it is measured rather than sensed.** From the next task, every review round is classified by
the KIND of its findings:

| Class | Meaning |
|---|---|
| **behaviour** | the product does the wrong thing for a listener |
| **instrument** | a pin, fixture or mutant does not measure what it claims |
| **guard** | a check on an instrument, a doc, or a claim in the record |

**The trigger:** a round that finds **zero behaviour findings and one or more guard findings** is a
round that a scoped meta-review could have handled. If that describes the majority of rounds across
the next three tasks, the meta round is adopted: a reviewer scoped to the pins, fixtures, mutants and
records touched by the task, rather than to the whole surface.

**What would argue AGAINST adopting it**, and is worth watching for just as hard: a full round that
finds a *behaviour* defect late, as round 4 nearly did — its finding was a guard gap, but the defect
behind it (a callout with no words) is one a listener would have heard. A meta round scoped to
instruments would still have found that one, since the pin was the thing at fault. A round that finds
a behaviour defect NOT reachable from any instrument is the counter-example that kills the idea.

## Proposed shape for A2DH

A family, because these are different jobs, and the existing skill is already one of them.

| Skill | Covers | State |
|---|---|---|
| `implementation/defect-classes` | D-1 one carrier · D-2 no vacuous invariant · D-5 ordering names its scope · D-11 validate the pin | **Written.** Ready to upstream |
| `verification/instrument-integrity` | Shapes 1-4 above: the pin weakened by its own change, the proxy that stops being faithful, the mutant that stops being evidence, cleanup that only survives a clean exit | **Candidate** — the strongest of the three, and the one with the most catches behind it |
| `verification/mutation-testing` | Authoring rules (delete a RULE, keep the tree compiling), verdict taxonomy CAUGHT/SURVIVED/INVALID/UNAPPLIED/SKIPPED, escalate-on-survival, the corpus guard, re-run after any edit to the target or its pin | **Candidate** — largely written already, scattered across `06_docs/mutants` |

**Upstream items already identified, independent of the skills:**

1. `make p10`'s failure message conflates "the binary has no p10" with "p10 found violations".
2. A thin install on PATH drifts from the canonical build with nothing reporting it, and the version
   string does not distinguish them.
3. `a2dh` has no SPIKE command — the backlog item already recorded.

---

## What UAT is for, and what it is not evidence of

**UAT findings are specification gaps, not defects, and they do not count against the metrics**
(HUM LEAD, 2026-09-03). Every one in this session was code doing exactly what it was told: nothing
said a `[w]` read could be paused, nothing said `[esc]` should stop one, and no written rule forbade
the read a human could stack with fast keys.

**The corollary is the useful half.** No gate could have found any of them, and no review round would
have either — an adversarial reviewer checks code against its intent, and the intent was the thing
missing. **A gate protects a specification; it cannot write one.** That is a real and permanent limit
on what the proposed meta-round, or any review, can be expected to cover — and an argument for
putting a build in front of a person EARLY, since that is the only instrument that finds this class
at all.

**The metrics still apply to what follows a ruling.** Once MVS-D-74 existed, the code satisfying it
was ordinary work: the three regressions introduced while implementing it are counted like any
others.

## The unit test that tests around the feature (2026-09-04, RELAY REPLAY)

Three tests for the new Settings row passed while the row was **inert**. They asserted the picker's
seven choices, its wrap at both ends, its unknown-value fallback, and the exact text of its two drawn
lines — everything except that pressing → does anything. All three called `cycleRelayDwell` directly.
The row's `picker` field was false, so `setup.go` never dispatched an arrow key to it, and the
control rendered perfectly and ignored every press.

**The tell is the seam the test skips.** Each of those tests entered below the keyboard, and the
defect lived exactly there. The pin that found it (`TestTheChosenRotationIsSavedToTheRadio`) enters
where the listener does — open the window, walk to the group, press →, save — and it caught the
inert row, a close path that forgets the value, and a store that never re-tells the Director. Three
defects, one pin, because it crosses every seam instead of standing inside one.

This is the **same shape as the Watchlist regression** the HUM LEAD found at UAT the day before, where
`Powered{Running}` was never emitted and no fixture noticed because every fixture sent it itself. In
both cases the units were right and the WIRE was missing, and in both cases the test suite was
looking at the units.

**Candidate rule (D-12): a feature needs one pin that starts where the human starts.** Unit tests may
enter anywhere; at least one test per feature must cross every seam between the input and the effect.
The cheap heuristic: if no test in the set would fail when the feature is disconnected from its
keyboard, the set is testing around the feature.

## The gate that found a hang nobody would have hit (2026-09-04)

P10-01 flagged `relayDwellLabel` for recursion. It was not a style finding: the function fell back by
calling itself with the default duration, which terminates only because the default happens to be one
of the offered entries — a fact nothing in the types states and one list edit away from a stack
overflow. Rewriting it to find the fallback in the same walk removed the recursion and made the empty
case impossible. **A structural gate can catch a latent hang that no test would**, because no test
would ever supply the input that triggers it.

The same run flagged a package-level `atomic.Int64` override. The honest answer was not an exemption:
its own pin had needed a save/restore `t.Cleanup` to keep from leaking into the next test, which is
the diagnosis rather than the workaround. Moving it onto the deck deleted the global, the cleanup and
the exemption together. **Two P10 findings, two restructures, zero exemptions** — worth recording
against the standing worry that P10 mostly generates paperwork.

## The state table found the defects before the code existed (2026-09-04, PD-3)

Seven rows written before any implementation. Two of them failed on the first run, and both were real:

- **The notice was queued at PROPOSED**, and the lineup holds admitted cards only — so `Queue` refused
  it silently. A listener would have got the gap AND no explanation: strictly worse than the stale
  read the notice replaces.
- **Row 5 was posed on the wrong track.** `advances(AlertRail)` is unconditionally true, because the
  rail drains whatever the listener has done to the programme. A stopped-programme case posed on the
  rail tests the rail's own rule and passes while the main-track case goes unmeasured.

The row worth writing the table for was **row 7**: the notice is a card on the same schedule, so if it
could go stale it would raise a second notice, which could go stale in turn. Writing the row forced
the question "what stops this looping?" before there was code to inspect — and the answer (its words
are fixed at proposal, so `BuiltAt` is zero) closed the loop through the model instead of a special
case.

**This is the second feature where the table paid for itself in minutes.** The pattern holds: a table
of cases, written first, catches the ones a reviewer reads past because the code looks reasonable.

## An anchor that navigates goes stale; an anchor that names survives (2026-09-04)

Removing a mutual recursion changed one function's return arity, and **four mutants stopped matching
the tree** — caught by the corpus guard, not by any test. Three were mechanical re-anchors. The
fourth is the lesson: `mS2` located the block it wanted by searching for a *call inside it* and doing
index arithmetic from there, so restructuring that call left the mutant unable to find anything. It
now names the two blocks it swaps, as literal text.

**The rule: a mutant should name what it changes, never navigate to it.** Navigation couples the
mutant to code it does not test. And the broader point, now on its third sighting: **the corpus guard
finds things no test does** — a rule lost in a move, a mutant silently gone INVALID, and now an anchor
made obsolete by a refactor two functions away.

## The outage that passed every check (2026-09-04, weatherusa.net)

The HUM LEAD reported live relay silent at two locations and working at a third. Every signal the app
inspects was green on the failing ones: HTTP 200, `Content-Type: audio/mpeg`, correct ICY headers,
~19 KB/s of well-formed MP3 that decoded without a single error. The relay was serving **digital
silence** — RMS 0.0–0.1 against 1775 on a working mount.

**Four hypotheses were wrong before the right instrument existed**, and each was cheap to kill only
because a measurement existed or could be built in minutes: a dead mount (probe said 200 and 19 KB/s);
a lock re-entry deadlock in the code just changed (the lock was released before the call); a blocked
event pump (a third location worked, so nothing global was stuck); the language tie-break (the mount
it replaced was equally silent).

**The tell was the third data point.** One failing location is a bug hunt; two failing and one working
is a partition, and the partition was by RELAY, not by location. Asking "what do the failures share
that the success does not" turned a code search into a two-row table.

**The rule: when every signal about a thing is healthy and the thing is not, measure the thing
itself.** Reachability, content type, byte rate and decoder health are all proxies for "is there
audio", and all four can be green while the answer is no. `TestMountDiag` decodes the samples and
reports RMS, and it is kept for that reason — the same shape as the perf protocol's "one owner of the
instrument".

**And a defect of our own came out of the hunt** that no listener had hit yet: the engine starts at
`urls[0]` while the deck is labelled with the station the chooser picked. Those agreed only while
"chosen" meant "the first candidate with a mount" — a language preference added hours earlier broke
the coupling, so a Spanish-preferring listener would have SEEN one transmitter and HEARD another.
**An invariant held by a coincidence between two functions is one a later change silently removes**,
and this one had no test because nothing had ever made the two disagree.

## The gate that reported its own failure mode (2026-09-04)

A `mutant-check` run failed nine mutants at once with "makes the tree uncompilable". One of the
compile errors named **`net/netip`** — a standard library package. Nothing in this repository can
break the standard library, so every one of those verdicts was about the machine.

**The guard exists to say when a verdict is no evidence, and it was producing verdicts that were no
evidence.** Each subtest compiles the whole tree, at one per CPU by default; with the app under UAT
and an editor running, a dozen concurrent full builds started failing. Diagnosis was three
measurements, not a guess: each mutant passed in isolation, the copy was only 24 MB (4 GB per run, not
the bottleneck), and a serial run went green on all 159.

**The ceiling now lives in the test**, not in whatever command invokes it — four lanes, 286 s against
429 s serial. A gate whose reliability falls as the corpus grows is least trustworthy exactly when it
is worth the most, and this corpus only grows.

**The transferable rule: a failing gate is a claim that needs its own evidence.** "The gate went red"
and "the code is wrong" are different statements, and the distance between them is where a green-only
habit quietly starts editing code to satisfy a broken instrument. The tell here was free and took one
look: an error naming a package the change could not possibly touch.

## A mutant can be CAUGHT for a reason other than its rule (2026-09-04, T3.8)

Restructuring the reading path moved **eight** mutant anchors at once — the corpus guard's largest
catch this release, and no test flagged any of them. Seven re-anchored mechanically. The eighth,
`m50`, claimed to guard "the band has one owner" (D-1) and **survived** at its new home: sending the
message directly instead of through `mastercontrol` produces the identical observable message, so no
test can see the difference.

**Its old verdicts were real, and they were about something else.** The old form also cut the read
short, and that is what failed the test. It had been green for years against a rule it never
measured.

**Two rules come out of this.** First: **a rule about WHICH CODE may do a thing is not observable in
behaviour** — it belongs in a gate, not a test, which is the shape `lint-imports` and `lint-watermark`
already have (recorded as F-22). Second, and more general: **re-validating a mutant after its anchor
moves is not bookkeeping — it is the only moment the mutant's own premise gets checked.** A mutant
that never moves is never re-examined, and one passing for the wrong reason looks exactly like one
passing for the right one.

**The cost lesson attached to it.** Nine verdict runs at ~90-120 s each is ~18 minutes, and the HUM
LEAD noticed the wall-clock before the work was priced — the standing rule is to multiply unit by
count BEFORE launching. The cheaper policy that would have found this anyway: `mutant-check` already
proves every anchor applies and compiles, so a full CAUGHT re-verification only earns its cost for
mutants whose RULE relocated. Here that was three of nine, and the survivor was among them.

## Death Valley's temperature, shown as Lone Pine's (2026-09-05)

The HUM LEAD reported 86 °F at half past six in the morning, against ~58 °F everywhere else.
Diagnosed in one pass by asking the API what the app asks it:

	KO26   Lone Pine Airport      2 km   — no observation at all
	KBIH   Bishop Airport        90 km   — no temperature
	DEVC1  FURNACE CREEK, DEATH VALLEY  110 km  — 29.78 °C = 85.6 °F

`fetchObs` walks the station chain and takes the first station returning a temperature. **Nothing
bounded the distance.** `Source.DistanceKm` was recorded and never checked.

**It was not stale and not cached** — the reading was 42 minutes old and perfectly real, so the
2-hour staleness warning correctly stayed quiet. Every component did what it was told. The missing
rule was *"a measurement from 110 km away in a different climate is not this location's weather."*

**The right answer was already present and being suppressed by the wrong one.**
`rehydrateFromForecast` fills a missing temperature from the location's OWN hourly forecast — which
read 59-60 °F — but only when the observation has none. So the fix converts a wrong number into a
right one rather than into a blank, and that is what made it safe to ship quickly.

**And the fix had a defect of its own, found by the HUM LEAD within minutes.** The claim above —
"the forecast fills it" — was ASSERTED, not tested. `rehydrateFromForecast` fills a **sparse**
observation and returns early on `Source.Provider == ""`; its own comment says "a location with no
observation at all stays a loading state". By rejecting EVERY station for Lone Pine I removed the
observation entirely, so there was nothing to rehydrate and the row loaded forever.

**The lesson is precise: I read the fallback's code, quoted its comment, and still asserted the wrong
half of it.** The distinction between "sparse" and "absent" was written down in the function I was
relying on. What would have caught it is the thing the ledger already says — a test that crosses
every seam between the change and the effect (D-12). The bound was pinned; the CONSEQUENCE of the
bound was not, and the consequence is what the listener sees.

The correction: "no station near enough" is a stable FACT and returns a provenance-only observation
the forecast can fill; "the fetch failed" is TRANSIENT and still errors, so the row stays loading and
the retry owns it. Settling for the forecast on a network blip would be the original defect in a
quieter costume — a plausible number standing in for the true one with nothing to say it had.

**The shape: a fallback with no bound is a silent substitution.** Every fallback chain answers "what
if the first choice is unavailable" and almost none answer "how far is too far". The chain was
correct, ordered nearest-first, well-tested — and had no stopping rule. Worth asking of the others:
the relay tune list, the voice fallback, the geocoder.

## "The race gate failed" was never a race (2026-09-05)

`make verify`'s race step is `go test -race ./...`, so ANY failing test in the tree reports as
"race failed". Twice this session that was a stale `declset` golden, and twice it could not be
reproduced — because recapturing declset in between fixed it. The label named a mechanism the
failure did not have.

**Two habits, both already recorded and both violated here.** Tailing gate output instead of
capturing it threw away the failing test's name — the same mistake as the earlier mutant-check
diagnosis, where the rule "a failing gate is a claim that needs its own evidence" was written down
that morning. And a gate whose name describes a mechanism rather than a scope will be misread every
time it fires for another reason.

## Two fixes that passed their tests and failed in the listener's ears (2026-09-05)

A `[w]` read kept playing after `[esc]`, under the location report that resumed beneath it. Two
shipped attempts were green and wrong, and the third worked. The interesting part is WHY the first
two were green.

**Every test reached the arbiter's DECISION, and the decision was correct all along.** The narration
arbiter suspended, released and settled exactly as designed; pins covered all of it and passed. The
failure was one layer down, in the audio, where **no test in the repository reached**. So the tests
were not weak — they were aimed at the half that worked.

**The trace eventually reported the truth, and the truth read like success:**

	release:onair cancelled=true voice=true audible=true
	mc:stopLine
	player:stopPreview found=true held=0

A player *was* found and *was* closed — and the report played on. `Close` releases a player; `Pause`
is what stops the device emitting what is already BUFFERED, and for a minutes-long read the buffered
remainder is not a tail, it is the rest of the report.

**Three rules come out of it.**

**A layer with no test is where the defect will be, and "all pins pass" says nothing about it.** The
gap is now closed for this rule: `TestStoppingALinePausesBeforeClosing` uses the recording player to
assert pause-before-close as an ORDER, and the mutant reproducing the shipped defect is CAUGHT.

**An instrument that logs an attempt must not be read as logging the outcome.** `narrate:admit` sits
above the wait loop, so forty "admits" stood against one release — and I read the ratio as a finding
rather than as a flaw in my own line.

**An append-only debug log across binary versions is a trap.** A five-hour file mixed three builds; I
read the ABSENCE of a line as evidence about the current one, when that code had not existed when
those entries were written, and nearly reported a bug that was not there. A fresh file per run makes
the timeline unable to lie about which code produced it.

## BUILD exit, round 3 — the shapes a ten-lens blind sweep found (2026-09-06)

Ten lenses over 224 commits, dispatched blind, ~1.93M tokens. Four Criticals, and the two that
matter here are SHAPES rather than instances.

**A field that exists for a FOREIGN reader is not this program's state.** `config.Save` derives
`ticker_muted` from the tone mode so a **0.13.0** binary reading the file still mutes its ticker.
0.14.0 then read it back at launch as its own runtime "do not speak to me" flag — so muting one tone
class silenced every spoken alert, permanently, with no way to clear it because the toggle had been
retired. Two lenses found it independently: the strongest convergence signal of the round. **The
rule: a back-compat mirror is written, never read back.** The fix deleted the parameter rather than
defaulting it (P10-07) — a seed nothing may vary is one more thing that can be set wrongly.

**A registry that describes its own reachability will tell you what is unreachable, if asked.** One
lens claimed three read-rank categories could never be occupied. `category.Spec` has a `Watchlist`
flag — "a category the national feed cannot produce" — and two of the three carry it, so they are
correct by design. **Emergency did not**, which is what made rank 1 a defect rather than a scope
decision: an `Evacuation Immediate` was painted on the marquee and never spoken. The registry had
the answer; nobody had asked it. **Verification is not a formality — a lens's verdict is a lead.**

**Enumerated guards with holes, for the third and fourth time.** `TestTheEffectSetIsClosed` listed
eight effects and omitted `Escalate`, which is live: the guard whose only job is to notice the set
growing a member had already missed one. Two memo registers claimed "one row per field" with 15 rows
for 22 and 27 for 34 — and the nine missing included the exact four whose absence froze three windows
in UAT this release. F-18 and F-30 are the same shape, already known. **Derivation is the answer
every time**: the effect set now comes from the `isEffect` marker in the package source, and the
flow-map check resolves the package-scoped refs its regex had been silently skipping.

**The wire is not pinned — again, three more.** BD-6's significance reach was implemented, pinned,
mutant-guarded and connected to nothing: the only assignment of `ReachMi` in the tree was in the test
that built the `Arrival` itself (D-12), and two mutants guarding the scale lived BELOW the wire, so
their CAUGHT verdicts were the m50/m42 pattern. Found by asking *who writes this field and who reads
it*, not by reading tests. **The general instrument is a producer/consumer completeness check over
the closed sets** — every effect, event, `Arrival` field and slot names a production writer and a
production reader, or carries an explicit "unwired, and why". Written once it subsumes F-18, F-30,
C-3 and the effect-set hole.

**A build tag with no gate is a tag that rots.** `app/inject_seam_test.go` — the test the whole
injector stands on — stopped compiling at T3.10b and stayed dark for the rest of the release, because
`go vet ./...` never sees a tagged file. It also had no schedule wired, so its takeover assertion was
measuring nothing. `make verify` gained `vet-tags`.

**Fixture staleness is invisible until a check gets stricter.** Adding an expiry re-check at compose
time failed four test fixtures at once — all posing alerts dated 2026-08-27, expired nine days. Their
own validity guards ("the takeover never cued the band — this proves nothing") are what named it
rather than a bare assertion failure. **The guards paid for themselves here.**

**The mutant-anchor tax, once more.** Changing two lines to add one field detached three mutants from
the code they guard, and only `mutant-check` said so. Re-anchoring is not enough on its own — all
three were re-run for verdicts, because "applies and compiles" is not "still CAUGHT". They were.

**And one lens's finding was wrong in my favour, which is worth recording too.** `Duck` → `mc.hold()`
against `Restore` → `voice.releaseBed()` was reported as two owners for one pair. It is not: acquiring
is one critical section inside mastercontrol (that IS the F-D5 fix), and releasing must go through the
arbiter's lock or the order inverts and deadlocks. It is documented now precisely so it is not
"tidied" — the finding was a comprehension gap, and the code was missing the sentence that would have
prevented it.

## F-29 — the flake was never where it was recorded (2026-09-06)

F-29 said `platform/sched.TestTierCadenceIsAFixedGrid` was flaky under `-race` — 2 failures in 25 on
the clean tip — and diagnosed it as the fake clock's `Advance` sleeping 5 ms and the wake-up
"occasionally" costing more than that.

**The recorded instrument did not reproduce.** 25 runs under `-race`: zero failures. F-29's numbers
were taken during a gate sweep, i.e. under load, and generating synthetic load is off the table here.
So the statistic could not be used, and a fix could not be justified by one.

**Proving it by construction instead.** `Advance` fired the due waiters and then slept 5 ms — "give
the scheduler goroutines a beat". That is a wall-clock GUESS against work of unbounded cost, and no
number of green runs makes it not one. The pin written for it asserts the property directly, with no
wait at all: when `Advance` returns, the tier it woke has acted.

**The control then said something better than the flake rate would have.** With the ORIGINAL 5 ms
sleep, that pin fails **10 times out of 10**. The sleep was never adequate — not "occasionally
short". What made the old test pass was `waitFor`'s 2-second polling loop absorbing the entire
shortfall, and *that* bound is what ran out under load. The recorded diagnosis had the right file and
the wrong mechanism.

**Three rules.**

**A success bound and a failure bound are not the same instrument.** `time.Sleep(5ms); assert` says
"I hope the work is done". `for deadline { if cond { return } }` says "proceed when it IS done, fail
if that takes absurdly long". The first is a race; the second is a bound. The suite has 65 of the
second shape and they are fine. It had one of the first, and it was the flake.

**A polling wait can HIDE an unsynchronised helper indefinitely.** The 5 ms sleep was wrong from the
day it was written and cost nothing for months, because a 2-second poll sat downstream of it. Removing
the poll is what made the defect visible — and the poll was the thing everyone blamed.

**When the recorded instrument will not reproduce, do not reach for a bigger sample — change the
question.** "How often does this fail?" needed load. "Does this helper synchronise?" needed one run
and answered deterministically.

## The instrument said CAUGHT and the rule was unpinned (2026-09-06)

A mutant's edit reached a commit, and the whole suite — `-race` included — went
green over it. Every layer that should have stopped it had a reason not to.

**What happened.** `run.sh` applies a mutant, tests, restores on exit. I committed
with `git add -A` while a run was in flight, so the commit captured the applied
mutation; the trap's `git checkout -- <file>` then restored the file FROM HEAD,
which now held it. **The guard designed to undo the edit made it permanent.** The
harness's stated rule is "a mutant needs a clean base"; the missing half is that
the base must stay clean for the whole run.

**Why nothing caught it.** `holdRest`'s `rest <= 0` branch was genuinely unpinned
— `hold(d)` loops on `d > 0`, so a non-positive d returns true without ever
reaching the air check, and no test failed when the branch was disabled. The
defect landed in the one place the suite could not see.

**And the verdict that hid it was false.** mH0 had been reported *"CAUGHT —
TestAPanickingExecutorDoesNotKillThePump"*. A panicking-executor test has nothing
to do with a remainder guard on a hold. The machine was loaded (my own parallel
work), a timing-sensitive goroutine test failed for its own reasons, and the
harness credited the mutation. **run.sh's green-baseline gate already names this
hazard in its own comment** — *"a flaky timing test makes every mutant look
caught"* — but it samples the baseline ONCE, so a 1-in-N failure passes it and
then fires on the mutated run.

**Four rules.**

**A CAUGHT verdict is only evidence if the failure is ATTRIBUTABLE.** run.sh now
restores and re-runs the named test against the unmutated tree; a genuine catch
cannot fail there. One test, only on a catch. Modelled deterministically in the
controls with a counter on disk — pass once, fail after — and verified both ways.

**Read the verdict, not just the colour.** Two tells were in the line I relayed
and I passed over both: the test's SUBJECT did not match the mutated rule, and
the failure took 70s rather than 0.00s. A deterministic assertion fails instantly;
a timing failure does not. The four verdicts I re-checked on a quiet machine all
named subject-matching tests failing in 0.00s.

**A single-sample gate is not a gate under load.** The baseline check, the 5ms
sleep in F-29's fake clock, and this are the same shape: one observation standing
in for a property. Contention is what turns them from usually-right into wrong.

**Concurrency I create is my problem, not the machine's.** I was told twice not to
run work in parallel and did it anyway; both of the day's instrument failures
trace to load I generated.

**What the sweep found elsewhere.** The absence-assertion class — "no glyph in
this output", which passes vacuously on an empty render — is already handled well:
`detail_seismic_test.go`, `severe_test.go` and `TestFrameGoldenASCII` each carry a
positive control beside the negative scan, and the golden files back the rest. The
susceptible classes that remain are **fixture COVERAGE** (a gate whose fixture
reaches only some of the states its surface can be in — `TestSetupGoldenASCII` was
blind to the suggestion list for exactly this reason, and F-37 records the same
for the windows with no scan at all) and **`-update` regeneration**, where a
golden or declset blessed from a wrong tree encodes the wrong answer silently.
run.sh's crash-is-CAUGHT path also still has no attribution, and says so.

## F-28 closed by a test, not a session — and the first version of it proved nothing (2026-09-06)

F-28 asked whether a stopped read's decoded audio and its player are RELEASED,
after 133–175 MB was seen across UAT sessions. It proposed a pprof session with
the heap sampled three times.

**It was answerable as a test.** The player is drivable headless, so the question
became three pins in `domains/radio/player` instead of a one-off measurement —
and unlike a session, they keep answering.

**The first version passed and meant nothing.** It used the existing `fakeOutput`,
which drains PCM at memory speed: an 8 MB clip finishes in microseconds and is
collected whether or not the engine holds it. **The control caught it** — "a clip
still playing must NOT be reported freed" failed, which said plainly that the
probe could not tell retention from release. Without that control the green would
have closed F-28 on nothing. A retaining player fixed it.

**The second version found a defect that was not one.** Written as "a paused read
that is stopped must not strand its audio", it failed: StopPreview closes what is
in FLIGHT and leaves heldOrder alone. That looked like a leak on the `[esc]` path
and is not — the app never leaves a held line behind, because `app/director.go`
calls `dropHeld()` whenever a suspended job is released or its context ends, and
that guarantee was ALREADY pinned twice
(`TestReadCancelledWhileSuspendedDoesNotWedgeTheDirector`,
`TestReadCancelledWhileParkedSuspendedReturnsAtOnce`). The test was posing a state
production does not reach. Rewritten to pin the SPLIT — a held line survives a
stop, DropHeld releases it — so a future StopPreview that quietly starts draining
heldOrder makes the arbiter's call dead code loudly rather than silently.

**The answer: no leak.** In-flight audio is released on stop; held audio is
released by DropHeld; the arbiter always calls it. The 133–175 MB is what a
minutes-long read costs — ~10 MB per minute of decoded PCM — which is what F-28
itself guessed and nobody had checked.

**Two rules.**

**A probe needs a control that can SEE the thing it is looking for.** "Is it
freed?" is worthless without "and would you notice if it weren't?" The control
failed first here, which is the only reason the answer is trustworthy.

**A test that fails is not yet a defect.** Both of this session's most alarming
failures — this and the `--ascii` scan — needed the reachability question asked
before the verdict: can production get here at all? One was real, one was the
test posing a state the app never produces.

## Re-deriving what the code already says (2026-09-06)

Five times in one session I built an experiment to learn something a comment
stated, within a few lines of where I was already reading.

| I probed for | The code already said it |
|---|---|
| which key saves Settings — tried enter, down, tab, ctrl+s | `setup.go`: *"esc and the enter-save are the window's only two exits… Both exits write through sequenceWrites"* |
| whether `hold(0)` reaches the air check | `holdRest`'s own comment: *"s.hold(0) returns true without reaching its own air check"* |
| whether `fakeOutput` could retain a clip | its comment: *"drains PCM on a goroutine"* — a drainer by construction |
| whether a flaky test could fake a CAUGHT | `run.sh`'s baseline gate: *"a flaky timing test makes every mutant look caught"* |
| whether `e.held` leaked | `delete(e.held, p)` sat 25 lines below where I said "no delete anywhere" |

**Why it happens.** Probing FEELS like rigour, and it is — but it is the
expensive second step. The cheap first step is reading, and it gets skipped
precisely when there is a hypothesis to test, because the hypothesis supplies
its own sense of direction.

**Three rules.**

**Read the header and the doc comment before probing the behaviour.** This
codebase puts its contracts, its rulings and its history in headers. Running an
experiment without reading them is paying for an answer that was free.

**Ask "who already knows this?" before building an instrument.** There are three
standing authorities: the file header, the single-owner registry
(`category.Spec`, the slot table, `aaPairs`) and the goldens. `Spec.Watchlist`
corrected a Critical finding; the goldens settled the label question. A probe is
for when none of the three can answer.

**A rule stated only in prose is unfindable, so index it.** The real gap was
structural, not attentional: `docs/where-things-happen.md` mapped *events to
functions* and nothing mapped *rules to where they are stated* — and you cannot
grep for "how does Settings save" unless you already know the answer is
`applyOnCloseCmds`. It now has a **Rules** section under the same machine guard,
seeded with the five above. Verified the guard covers it: a bad symbol there
fails `TestWhereThingsHappenNamesRealSymbols`.

## Four hypotheses, then a smaller reproduction (2026-09-06)

The PTY journey failed nine steps. It took four wrong hypotheses and one change
of method to fix, and the method is the transferable part.

**The wrong four**, each reasoned from a single symptom in a 25-step, 3-minute
run: `match_max` too small (raised, no change); drain-before-stty (no change);
the drain eating the key's redraw (removed, no change); and stream position, two
assertions sharing one repaint — which was REAL but minor. A fifth was avoided
only because the HUM LEAD said to stop guessing and build the instrument.

**What worked: shrink the reproduction.** A probe that spawns the app, presses
`w` and looks for the severe window runs in twenty seconds. It PASSED every
time, which proved the fault was in the PREFIX rather than the interaction —
something the full run could never have told us. Bisecting the prefix, then
diffing the probe against the real script MECHANICALLY rather than by eye,
narrowed it to state, and a retry made the failing screen dump a later frame.

**The root cause was two assertions passing for the wrong reason.**
`"Setup saved and closed"` matched `Lookup Location` — which is drawn on the
dashboard BEHIND the modal — so it reported success while Settings was still
open, and every later key typed into that window. Three blind Enters had stood
in for closing it; Enter ADVANCES from a picker row, and 0.14.0 added four more
groups to advance through. The second was `CAST` matching inside `FORECASTS` in
the label guard I had just written.

**Three rules.**

**When a long scripted run fails, SHRINK IT BEFORE THEORISING.** A hypothesis
tested against a twenty-second reproduction costs nothing; the same hypothesis
against a three-minute run costs an afternoon, and I spent one. The reduction
also answers a question no amount of staring can: is the fault in this
interaction, or in everything before it?

**Assert the PRECONDITION where it happens, not the outcome three steps later.**
Both defects here were missing preconditions — Settings still open, the window
already closed — and both surfaced as confusing failures far from their cause.
A step that checks the state it depends on says what is wrong where it is wrong.

**A green check that cannot fail is worse than no check, and today produced
four.** `Setup saved and closed` matching through a modal; `CAST` inside
`FORECASTS`; the mutant CAUGHT by an unrelated flaky test; the `--ascii` scan
whose fixture never rendered the line. The shared question that catches all
four: *what would this assertion do if the thing it checks were broken?*

## The README documented a window that had been renamed (2026-09-06, REVIEW)

Working the release checklist's README row, three statements were wrong in the
same way. Two places still called the window **Setup** after this very release
renamed it Settings — the CHANGELOG announces the rename four screens above.
The marine report was still introduced as **maritime** after MARITIME became
MARINE "in the windows and on the air". And the Radio section's enumeration of
the broadcast order named the Fire and Seismic reports and **skipped the marine
report entirely**, though `compose.go` puts it immediately before fire.

**None of these could have been caught by anything the tree runs.** Every gate
we have reads Go. The README is the one artefact written entirely for people who
are not us, it is the first thing a reader of this release sees, and nothing
measures it. That is this session's theme again in a new place: not a broken
instrument this time but a **missing** one, over the surface with the widest
audience.

**The rule.** A rename is not done when the code compiles. Every user-facing
rename has a documentation half, and it is invisible precisely because prose
does not fail to build. The cheap instrument exists and we already use it
elsewhere: `docs/extending.md`'s walkthrough was checked by resolving every
symbol it names against the tree, which is how we know it is current. The same
shape — a small check that retired user-facing names do not appear in the docs,
seeded from each release's own Changed section — would have caught all three.
**Recorded as a candidate, not built:** it is a new gate, and REVIEW is not
where new gates get added.

## A gate result is only as fresh as the tree it ran on (2026-09-06, REVIEW)

Resuming, the visible task outputs said `make race` FAILED. Acting on that would
have meant hunting a data race that no longer existed: the run was timestamped
09:25, and five commits had landed since — including the one that fixed the race
by moving a finalizer's write to an `atomic.Bool`. A journey failure in the same
batch was equally stale; it was the F-42 transient, fixed at 11:55, and the run
was 11:30.

Two minutes of `ls -lT` on the output files and `git log --date=iso-local`
settled both, before any debugging.

**The rule.** *A gate result carries a timestamp and a tree; a result older than
the tree is not a result.* Before treating any stored output as current, compare
when it ran against when the tree last changed. This is the same failure the
`tree_hash` column was deleted for on the P10 rows, arriving from the other
direction — there, a hand-copied hash was a second place to be wrong; here, an
uncopied timestamp made a stale answer look live.

## The end-to-end journey's severe-window block could not fail (2026-09-06, REVIEW)

One step failed — the `space` read — and chasing it found that the four steps
before it were incapable of failing. Every pattern they waited for is also drawn
by the DASHBOARD, because 0.14.0 made the ticker band name its lane in words and
those words are the category labels: `BandLabel` is `"Emergency Orders"`,
`"Warnings"`, `"Watches"`, `"Disasters"` (`platform/category/category.go:83-92`).
The marquee is on screen always. So "w opens the severe window" passed against
the tape whether or not the window opened, and so did the tab row and both tab
moves. A fifth, `Showing 1-`, was written to prove the Settings modal had CLOSED
and is drawn by the dashboard's own recent table (`modes/tty/body.go:302`).

**The rule this yields.** *An end-to-end assertion must name something ONLY the
surface it is testing draws.* The window-only anchor was sitting there:
`Total Category Events` (`severe.go:391`) appears on every tab, populated or
empty, and nowhere else. The old patterns were chosen because they are what a
person SEES in the window — which is exactly the wrong criterion, because a
person also sees the marquee.

**The defect under the defect.** With honest anchors the tab arithmetic broke,
and that was correct: `severeOpeningTab()` returns `lastBreakingTab` when a
breaking event landed recently (`severe.go:166`), so against live feeds the
window opens on any tab, and the arrows WRAP, so there is no end to clamp
against. Counting Rights from an assumed Warnings was never sound; the
unfalsifiable patterns had been hiding it. Navigation now presses Right until the
wanted tab is selected.

**And a third, in the guard that was supposed to catch all this.** Its count
check — "only N patterns were checked; the extractor has stopped matching" —
counted patterns AFTER exemptions were removed. So every honest exemption lowered
the number that exists to detect a blind reader, and enough of them would force
someone to lower the floor and quietly disarm it. Split into `extracted` (what
the reader found) and `checked` (what it verified); the floor now sits on
`extracted`, and blinding one arm of the regex drops it 28 → 8, which was
verified rather than assumed.

**What I got wrong on the way, because it is the same lesson.** I read a failing
step's `screen:` dump as a screenshot. It is not: `step` logs the tail of
`expect_out(buffer)`, and when an attempt reads nothing expect leaves the
previous contents in place — a dump timed 20:02 carried the first-run Setup form
that closed at 20:01. The finding survived only because its actual evidence was a
grep of the render code. **An artefact named `screen:` will be read as a screen.**
Either make it one or rename it.

## The disk was the instrument, and it was full (2026-09-06, REVIEW)

The volume had **1.2 GiB free of 926 GiB**, and 274 GB of that was the Go build
cache. A mutation run compiles the tree once per mutant, and every variant is a
cache entry that will never be reused, so the cache grows without bound and
nothing trims it.

**The reason this belongs in a quality record rather than a chore list:** every
measurement taken that afternoon was taken on that machine. The journey's
network steps failed, three runs disagreed with each other, and the 14.2 s radio
read — the number a release decision was about to be based on — was measured with
no disk left. **A full disk is a vector for bad results, not only for no space.**
I did not check free space until the HUM LEAD raised it, after hours of treating
the variance as flakiness.

**The protocol, ruled by the HUM LEAD.** Deterministic, and it consults the size
of nothing — a threshold only acts once the damage is done:

> do the thing → collect the results → put the results somewhere durable →
> verify those results are, in fact, saved somewhere durable →
> delete all my other build variants / artifacts

with one stated exception: one to three real binaries, for UAT and for comparing
versions.

**The verify step is the whole protocol.** Without it the last step is `rm` with
a comment, and the first run that dies early deletes the artefacts and the
evidence together. `make hygiene` refuses when `RESULTS` is unset, names a
missing file, or names an empty one — all three verified to delete nothing before
the target was trusted to delete anything. It removes only files in `dist/` that
are EXECUTABLE and not on the keep list, so the records that live beside them
(`journey.log`, `p10.json`, `bench.txt`, `validate/`) are never candidates.

**Recovered: 1.2 GiB → 275 GiB.**

## The first Linux run of the release was its release PR (2026-09-06, SHIP)

`make verify` had been green all day. CI's ubuntu leg failed in two minutes with
three failures and a **nil-pointer panic** that killed the test binary — so the
failure list was not even complete, because whatever ran after it never ran.

**The cause was one line in the wrong dialect.** `app/voices.go:rawVoice`
branched on `runtime.GOOS` instead of the `runtimeGOOS` seam, in four places.
`asPlatform(t, "darwin")` set the seam; `rawVoice` ignored it; on Linux two tests
that had explicitly pinned darwin walked into the Piper install path anyway — a
real network download, a missing cache directory, and a panic in a background
install's progress callback.

**Three things about this are worth keeping.**

**The P10 ledger row for that seam says exactly what it was for**, and it was
ratified: the seam exists *"so the M5 fallback matrix can walk BOTH platform
namespaces on one machine; without it half the matrix would never execute on the
developer's box or in CI."* The file that most needed it did not use it. A
documented, ratified, exempted seam with a caller that goes around it is the
fourth instance this release of *receiver built, wire missing* — after C-3, F-41
and F-42.

**A test for it already existed and could not fire.** `TestHostPlatformFollowsTheSeam`
asserts the platform follows the seam, and it passes on a Mac because there the
seam and the real OS agree. The check was written; the ability to run it in the
other configuration was not. That is a different failure from a missing test and
needs a different fix — not another assertion, but a way to run the ones we have
somewhere they can disagree.

**So the fix is a way to disagree, not another assertion.** `WATCHPOST_TEST_GOOS`
(a test-only `TestMain`) flips the seam for the whole package, and `make
test-platforms` runs the suite both ways. Reverting one call site makes the
simulated-Linux run fail three tests — verified, not assumed. Running it found
one more instance immediately: a test that branched on the real OS while the code
under test used the seam.

**Stated because it limits the claim:** this simulates the SEAM, not the
operating system. Real syscalls, paths and audio are still the host's. Green here
is evidence about platform *branching* and never a substitute for running on
Linux — which is precisely why the Linux validation protocol still has to happen
on real hardware.

**The uncomfortable part.** An eight-thousand-line SEV-0 release, ten red-team
lenses, four rounds, a mutation corpus and a 127-row exemption ledger — and the
thing that stopped the release was that nobody had run it on the other platform
until the PR. The branch was local-only by design; the last CI run of any kind
was the previous release, eight days earlier.

## Five CI rounds, and I fixed the wrong thing in four of them (2026-09-07, SHIP)

The release PR went red on Linux. It took five rounds to go green, and the
interesting part is not the defects — it is the shape of my own debugging.

**Round 1** — `app/voices.go` branched on `runtime.GOOS` instead of the
`runtimeGOOS` seam, so tests that had explicitly pinned darwin walked the Piper
install path on Linux. Real fix.

**Round 2** — I guarded `startBackgroundInstall`. Wrong path entirely; the panic
came back.

**Round 3** — the guard's predicate was half a predicate (engine, not engine *and*
program). The panic moved one line, from `Engine.Status` to `Program.Send`.

**Round 4** — the actual cause: `tune → startSynth → rawVoice` installs
**synchronously**, and a helper named `offlineDeck` was downloading 63 MB from the
internet.

**Round 5** — my own fix for a flaky test was still wrong, because I had replaced
the fake underneath an assertion without asking what the assertion was for.

**The pattern in one line: I kept fixing where the pointer was nil instead of
asking why a unit test was downloading 63 MB.** Each round I took the stack trace
as the statement of the problem. The stack trace says where the program noticed;
it does not say what is wrong. Four rounds of nil-guards were four rounds of
treating a symptom that moved.

**The same mistake in a different register, round 5.**
`TestSourceStopsFastWhileMidSegment` asserted that `io.ReadAll` returns a non-nil
error after cancellation. I built a blocking fake so the cancellation had
something to interrupt — correct — and kept the assertion. CI failed it again:
after a cancellation the stream may equally end *cleanly*, and both outcomes
appeared on macOS within one commit. The property UAT 81 protects is
**promptness**. **Preserving an assertion is not the same as preserving what it
was for**, and an assertion nobody can restate is a liability whatever it does.

**What the rounds were really telling us.** They were not five defects. They were
one: **this release had never been run on Linux.** The branch was local-only by
design and the last CI of any kind was the previous release, eight days earlier.
Every round was the same platform saying the same thing in a different place.

**The instrument that existed and could not fire.**
`TestHostPlatformFollowsTheSeam` asserts the platform follows the seam. It passed
throughout, on a Mac, where the seam and the real OS agree. **The check was
written; the ability to run it in a configuration where it could fail was not.**
That is a different failure from a missing test, and it needs a different fix —
not another assertion, but a way to disagree. `WATCHPOST_TEST_GOOS` and
`make test-platforms` are that.

**And the new instrument was hollow on its second use.** `make test-platforms`
answered `(cached)`. A cached "ok" is not a run, and the entire point was to
execute the suite in a configuration it had not been executed in. Found by
noticing the word "cached" in output I had already decided was green. **`-count=1`
is load-bearing in any target whose purpose is to run something differently.**

**Cost.** Five CI rounds, each ~2–4 minutes of runner time and a re-cut of the
release commit; perhaps two hours. A single push of the branch to CI on the day
the Director landed would have surfaced all of it while the code was warm.

**The rule, and it is cheap:** *push early enough that CI runs on every target
platform while the work is still being done.* Not before the release PR. The
branch being local-only is a git-hygiene choice; it silently became a testing
choice.

## One operation, four hand-written copies (2026-09-07, 0.14.1)

Issue #7 — the first Linux bug after release — was a name-to-install lookup
written out four times. `FindPiperVoice` locates a model by KEY; four callers
passed `synth.VoiceSpec{Name: name}`, whose Key is empty. They looked for
`voices/.onnx` and answered no for every voice however plainly installed. On
Linux the tone sounded, the ticker took over, and nothing was ever read.

**The fourth copy was written by copying the third.** `hostFacts.Installed`
carries the comment *"find-only, exactly as the deck's is"*. It was, faithfully,
defect and all. And the file's own header says why that is dangerous: *"two
implementations of which voices does this host have is how a report and a screen
start disagreeing about the same machine, so the deck and the report share this
one."* **It did not share it.** Three methods of one interface, written twice.

**The correct form was already in the tree.** The voice-preview path resolves
through `VoiceByName` first. Nothing else used it. So this was not missing
knowledge — it was knowledge that existed in one place and was re-derived,
badly, in four others.

**Then the maintainer asked one question** — *"can we turn that into a shared
helper so there aren't two call paths for the same functionality?"* — and the
grep that answered it found the two call sites the patch had missed. **The
question was a better instrument than the fix.**

**So we went looking for the class rather than the instance.** Every production
function body normalised and hashed: **nine groups of identical bodies**. Six
were one policy with two owners and are now one — the closed allowlist, the
last-resort voice, the twelve-hour clock default (the radio and the tape could
have disagreed about the time on one screen), the Producer's lock discipline,
the fire sentinels, a merge rule. Three were left, with reasons written down.

**Then for near-duplicates — the same thing done DIFFERENTLY**, which exact
matching cannot see. The sharpest find: `maxListLen = 50` and
`maxFieldRunes = 120` declared in **two** domain packages, with four differently
named helpers applying them. That is the bound on untrusted provider prose. Both
agreed, which is precisely the state issue #7 was in before it did not.

**THE RULE.** *Two implementations of one operation is a defect that has not
happened yet.* Not a style preference — a defect with a delay on it, because the
day one is corrected and the other is not is the day they disagree, and both
will look right in isolation. The cost of finding them is a hundred-line script.

**WHERE THIS SHOULD HAVE SURFACED, and this is the maintainer's point:** the
**Code Quality red-team lens, at BUILD and at REVIEW.** Ten lenses ran over this
release across four rounds and none of them asked "is this operation implemented
more than once, and do the copies agree?" — a question that is mechanical, cheap,
and would have caught issue #7 before a listener on Linux heard a tone and then
silence. It is going into the lens's brief. A red team that only reads for
correctness of what is written will not see the risk in what is written twice.

## A record that is filed is not a record that is kept (2026-09-07, inter-release hygiene)

**The catch.** `make hygiene` is the deterministic cleanup the HUM LEAD specified: *do the thing,
collect the results, put them somewhere durable, verify they are durable, delete every other
variant.* Four records of the release that just shipped were sitting in `dist/` — the directory that
target empties. One of them was the mutation run, and the **merged PR body cites it by that path**:
"Durable record: `dist/mutant-check.log`". It was one invocation of our own hygiene target away from
not existing, and the sentence claiming its durability would have survived it.

**Then the same question one level out, and this is the part worth keeping.** The obvious fix is
"move it into `06_docs/`". So: are the records already in `06_docs/` actually kept? `.gitignore`
line 2 is `*.log`. **Eight run records across four features** — the 24-hour baseline sampler the
whole perf pass is diffed against, the seismic counters run, two soaks, a journey — were filed in
the durable tree and **not in git**. They exist on exactly one disk. Nobody was wrong about where to
put them; the directory was right and the record still was not kept.

**THE SHAPE, and it is the release's shape again:** *filed is not committed; a durable-looking
location is not a durable record.* This is the same failure as the assertions that matched the
marquee instead of the window, and the "durable record" that was one line. The question that catches
it is the one already written down — **what would this check do if the thing it checks were
broken?** — asked of the storage rather than the assertion: *if this file vanished tonight, what
would tell me?*

**The fix is a guard, not a habit.** `hygiene` now refuses a `RESULTS` that lives in `dist/` or that
`git check-ignore` matches, and names any record still sitting in `dist/` so it gets promoted rather
than lost. All three refusals were run and observed to fail before the change was committed —
per *validate the instrument*, a guard nobody has watched fail is not a guard. `.gitignore` now
re-includes `06_docs/**/*.log`, which was the right fix over renaming eight historical records and
rewriting the red-team reports that cite them by name.

**Cost:** about twenty minutes, no code touched. **What it bought:** the evidence base of four
completed features moved from one disk to the repository.

## The rules a gate must obey (2026-09-07, 0.15.0 B1–B3)

**Eleven gates or instruments were written or repaired in one working day.  Every single one was
wrong on the first attempt, and every one was caught by asking what its green meant.**  The failures
were never in the rules being enforced — they were in the *fixtures*.  These are the rules that
follow, each with the catch that earned it.

### 1. A gate's self-test must use a fixture shaped exactly like the thing it will police

**Catch:** `lint-injector` asserted no release artifact carries the injector, and its self-test built
**without `-ldflags "-s -w"`**.  Release artifacts are stripped; the test fixture was not.  The gate
was blind — `go tool nm` reports *"no symbol section"* on a stripped linux binary — and the self-test
certified it anyway, because an unstripped binary *does* have symbols.  Shipped blind for ten
minutes.

*"Validate the instrument" is not enough on its own.  An instrument validated against the wrong
shape is validated into a false positive.*

### 2. Prove the matcher is alive before it judges

**Catch:** the config one-writer gate looked for a `config.Save` **selector**.  The moment the last
production bypass was migrated, `Mutate` called `Save` unqualified, zero selectors remained, and the
gate would have been green forever.  Its own empty-corpus guard caught it.  **Count the corpus over
everything, including tests; police only what the rule is about.**

### 3. A member is never skipped — carried, or declared with a written reason

**Catch:** `TestEveryFeedLaneSurvivesTheMarqueeMap` walked seven lanes and asserted **four**,
`continue`-ing past three whose fixture disagreed — including Advisories, the exact hazard it was
written to guard.  A `continue` reads as deliberate.  F-30's guard does the same with `t.Skipf`.
**Skipping is how a set gate lies.**

### 4. Catch the stale exemption, or the exemption becomes the defect

**Catch:** the same lane guard could not see a lane that *started* being carried while still declared
unreachable.  The two guards written after it could.  One rule, three hand-written instances, and the
oldest had already drifted — which is the argument for one owner, found by looking.

### 5. Match both spellings

**Catch:** twice.  A type or function is bare inside its own package and qualified outside it.  A
matcher that knows one spelling reports green while blind to half the tree.

### 6. Find the root; do not count it

**Catch:** a hand-written `"../.."` pointed one directory short of the module root, so the walk
matched nothing.  **A gate that has to know its own depth breaks when it moves.**  Walk up for
`go.mod`.

### 7. A number with no stated boundary is not reproducible

**Catch:** the closed-set population was "50 default arms" without an exclusion rule and **49** with
one — and ~30% different again if the vendored patch stack counts.  Write the boundary down *before*
counting, or the number cannot be checked by anyone else.

### 8. Do not edit the tree while the mutant gate runs

**Catch:** three mutants reported UNMEASURED that were fine.  The harness patches source files; my
edits moved them underneath it.  `run.sh` guards this with a dirty-tree check; `make verify` does
not, so going through `verify` bypasses the guard.

### 9. "Does this need a lock?" is answered by experiment, and the experiment needs its own control

**Catch:** FR-1.4 asked for a mutex on three methods.  They touch no mutable shared state, and the
engine state underneath is disjoint.  Ten goroutines under `-race` found nothing — **and a
deliberately planted unsynchronised field produced three DATA RACE warnings**, which is the only
reason the clean run means anything.  Outcome: no lock, with evidence.  *A race test is worth keeping
precisely when it passes; its job is to notice when that stops being true.*

### 10. Extract from working implementations, never from imagined ones

**Catch:** `closedset` was designed at PLAN with five fields and five call sites.  There was **one**
call site, and the shape was wrong — the thing that repeated was not a mapping with fixtures but a
cross-product of *carried* against *declared absent*.  The true skeleton only became visible at the
**third** hand-written instance.  The standing rule says extract at the second caller; it assumes you
have two working callers in front of you, not two imagined ones.

### 11. When you correct an instrument, check the correction the same way

**Catch:** the red team found half of NFR-2's proposed predicate vacuous.  I corrected it — and my
correction could not work at all, because release binaries are stripped.  **A correction is a new
claim and carries the same burden as the original.**

### 12. A gate that measures a rendered surface must count in the coordinates the renderer draws in

**Catch:** FR-5's first probe compared an 80x24 render against an 80x200 one and reported **every
line of every window unreachable**.  A modal that overflows wraps three columns narrower than one
that does not, so the two renders share almost no line.  The defect it was hunting was the *same
mistake*: the scroll offset was computed from the body before the panel re-wrapped it, so it pointed
at line 11 of a body whose focused row had moved to 14.  **The instrument and the bug were one error
in two places** — and the probe's version was found first only because it was measured.

### 13. A cursor is not text

**Catch:** the same probe reported `a burst` unreachable while `› a burst` was on screen.  A focused
row carries a pointer glyph and an unfocused one does not, so a comparison that keeps the glyph
counts one line as two and reports the form that is not currently focused as missing.  **Normalise
away everything the frame draws that is not content** — box, rail, padding, cursor — or the
measurement is of the instrument.

### 14. Fix the member, then measure the set

**Catch:** the relay-fault window's unreachable ways out were found by a human at 80x24, fixed
correctly, and **the fix did not generalise** — the window next door had the same defect five weeks
later, and the shipped ctrl+d window had a worse version of it that no key could work around.  Three
separate discoveries, one mechanism.  A defect found by opening one window is a question about the
set of windows; the set-level property is what turns three sightings into one number that can only
go down.

### 15. The gate that pays for itself is the one that fires on YOUR change

**Catch:** FR-6.4 added one boolean to the relay-fault window's state, and F-30's memo guard failed
on the next run: the field changes the frame and was not in the memo key, which is exactly the defect
that froze that window through three UAT rounds in 0.14.0.  Nothing else would have caught it —
every test in the file renders through the memo's MISS path.  In the same run, mutant mV3's anchor
no longer matched the line it patches, and the harness reported **UNAPPLIED** rather than passing
over a mutation that no longer applies.

**Two different instruments, one property:** a gate is only worth its cost if it can fail on work
nobody wrote it for.  Both of these were written for defects that had already happened, and both
earned their place again on a change made months later by someone who had forgotten they existed.

### The meta-rule

**A red-team finding is a hypothesis, not a fix.**  Twice in one day, measuring a lens's
recommendation changed it: `debugScenarios` discriminates nothing, and the symbol approach it implied
cannot work on a stripped artifact.  Both lenses were right that something was wrong and wrong about
what to do — which is the correct division of labour, and only holds if the measurement actually
happens.

## The metric this is all judged against

Tasks completed per session. It has not moved yet (1). Every other number has. The programme
continues on the HUM LEAD's 2026-09-03 ruling, and the reason given was **cost per defect** — minutes
rather than hours — rather than defect count.
