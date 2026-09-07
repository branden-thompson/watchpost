# Key learnings — multi-voice-support (0.14.0) — retro notes, collected as they happen

Items for the DEBRIEF and for the A2DH calibrations (the harness's own retro). Dated as raised; the DEBRIEF turns
them into lessons with evidence.

## RN-1 — 2026-08-29 (PLAN, red-team round 4 running) — no implementation code at PLAN for compiled languages

**Raised by:** the HUM LEAD, after ~2 h of plan churn across three red-team rounds.

**The item:** for compiled languages, PLAN carries **no implementation code** — only what vets the architecture
and illustrates a contract (a signature, a seam, a ten-line sketch). Implementation code — full function and
test bodies — is reserved for BUILD, where it becomes real feature work that can be compiled, linted, tested and
iterated on in reality.

**What happened here:** the four batch files (`04-development/p1..p4`) carried ~4 000 lines of Go in markdown.
Red-team rounds 1–2 found real design defects (the resident backend, `reflect` on the save path, the
`modes/ → domains/` import, the unbounded fan-out); rounds 3–4 spent most of their findings on what a compiler
catches in seconds — mixed parameter lists, undefined helpers, RED strings that did not match their builders, a
lock order, a key name bubbletea never produces — and each remediation pass, written in prose, introduced the next
round's Criticals.

**Proposed anti-pattern (A2DH catalogue, alongside `AP-ASSUME-01` / `AP-DEAD-01` / `AP-HIST-01`):**

> **`AP-PLANCODE-01` — Deep implementation code in a PLAN artefact.** Writing production-shaped code (full
> function bodies, RED/GREEN test bodies, complete type definitions with their methods) into planning documents
> for a compiled language, in place of describing the intended end state and the architecture to vet it against.

*Why it is an anti-pattern, not a style choice.* Plan code carries the **appearance** of rigour — it looks
reviewable — but it cannot be compiled, linted, run or refactored, so its defects are invisible to every tool
and visible only to a human reader. Review effort then goes to the defects a compiler would have caught for
free (undefined symbols, wrong arities, imports, stale identifiers, tests that contradict the builder beside
them) instead of to the questions only a human can answer (is this the right seam? does the failure propagate
safely? is the contract honest?). Worse, each remediation pass rewrites prose code and introduces the next
round's defects, so review rounds stop converging while the *design* is never re-examined.

*Symptoms.* Red-team findings dominated by compile-ability rather than architecture. Remediation rounds whose
new findings are mostly in the previous round's fixes. Helpers named in prose that exist in no task. Tests whose
expected strings cannot be produced by the builder in the same document. A plan a reader must mentally
type-check.

*The correct shape at PLAN.* Per task: **file · symbol · contract · test intent · verify command**, plus the
risk it addresses. Illustrative fragments are allowed **only** to settle an architecture question — a signature,
an interface, a ten-line sketch of a seam — and are marked as illustrative. Everything else (bodies, fixtures,
expected strings, imports) belongs to BUILD, where the compiler, the linter and the tests are the reviewers.

*Detection (cheap).* At PLAN exit, apply the four B-vs-C tests below; as a rough proxy, fenced blocks beyond
roughly a page per batch, or `func` bodies longer than a signature plus a few lines, mean the anti-pattern is
present. The inverse check matters too: **zero** executed spikes on a compiled-language SEV-0/1 plan is its own
smell (`AP-ASSUME-01`).

**The worked example — scale is a smell, not a verdict (the HUM LEAD, 2026-08-30).** The strip of this feature's
four batch files took them from **7,069 lines to 993**, removing **106 fenced Go blocks** and leaving zero. The
observation on seeing that number: *"7,000 lines of code tells me we were building the entire feature in plan."*

Say precisely what that licenses, because it is easy to overclaim (the first draft of this note did, and was
corrected): **size does not convict.** A genuinely large feature — many subsystems, several platforms, a long
ruling history — can need a long plan, and a rule that punished length would push planning toward being too
thin, which fails RN-2's comprehension questions instead. What size does is **trip an inspection**: a plan at
that volume is *unlikely to need to be that large*, so it must be read critically rather than accepted, and the
two things the reading looks for are:

1. **Implementation artefacts of the thing, hiding inside the plan itself.** The feature being quietly built in
   the planning document — finished bodies, fixtures, expected strings — under headings that look like planning.
   This is `AP-PLANCODE-01` proper, and it is what 7,069 lines turned out to be here.
2. **Elaboration that simply wants simplifying.** Prose restating a decision in three places, a contract argued
   at length instead of stated once, tasks that duplicate their neighbours. No code involved, no anti-pattern —
   just a plan that would serve its readers better shorter.

*So the number is a trigger, and the verdict still comes from reading.* Apply the four B-vs-C tests to the
blocks and ask, of the prose, what the document would lose if a section were cut to its decision. The failure
mode to avoid is treating the count as the finding: a plan is not wrong for being long, it is wrong for
containing the implementation, or for saying a small thing at length.

*Cheap instruments worth recording at PLAN exit* — one `wc -l`, one `grep -c '```go'`, before and after any
strip. They cost nothing and turn "this feels bloated" into a number a reviewer can act on. Two rough triggers:
Go volume within an order of magnitude of the feature's expected diff, and a fenced block that cannot be read
without scrolling. A third is about the *review* rather than the plan: if a round grows the plan's line count
faster than its decision count, the round is churning text rather than vetting design.

*Related.* `AP-ASSUME-01` (the un-spiked mental model) is its cause when the plan code is written against a
guessed API; `AP-HIST-01` (history-narrating comments) travels with it.

**The line between useful and churn — the delineation the rule needs.** The round-4 lenses that *compiled* the
plan's blocks in a throwaway worktree produced the phase's best findings: the `cast` package, the Limiter and
the Source were shown to **run** (17/19 tests green under `-race`, both hand-over paths, the writer-starvation
property), a real dependency defect was proven (go-toml v2.2.4 panics on an escaped quoted key → the bump is
required, not optional), and a contract was *discovered* rather than argued (with one segment of look-ahead a
background `Invalidate` takes effect two segments later, not one). That is code at PLAN, and it was worth more
than the three prose rounds before it. So the rule is not "no code at PLAN". It is:

> **Code at PLAN is for *executing*, never for *shipping in the document*.** A spike is compiled, run, and
> thrown away; what survives is its **result** — a number, a verdict, a contract, a decision. Implementation
> code written into the artefact is read, never run; it survives as text that every later round must re-review
> and every remediation re-writes.

**Three tiers, and where the line falls:**

| Tier | What it is | Lives where | Kept | Verdict |
|---|---|---|---|---|
| **A — Spike** | Throwaway code compiled and run to answer one question: does this seam hold, does this library behave, does this design meet its budget | a scratch worktree, deleted after | the **finding** (one line: question · method · answer · decision) | **Encouraged** — at SEV-0/1 for a compiled language, expected wherever an assumption is expensive to discover wrong at BUILD |
| **B — Illustrative fragment** | Exported signatures, an interface, a type's fields, a representative flow-control skeleton whose bodies are `// TBFI in BUILD` — the shape a principal hands the engineers who will do the work | the plan document | the shape, not the text | **Welcome** wherever it materially aids understanding — this is the *useful* half of the rule, not a grudging allowance |
| **C — Implementation artefact** | Function bodies, RED/GREEN test bodies, fixtures, expected strings, imports | the plan document | everything, forever | **`AP-PLANCODE-01`** — the PLAN target is **zero**; see the one narrow exception below |

**Four tests that tell B from C** (any "no" means it is C):
1. **Executed?** Was it compiled and run, or only read?
2. **Is the code the deliverable, or the finding?** If deleting the snippet loses knowledge, it was a spike badly recorded; if deleting it loses only typing, it was C.
3. **Does it churn?** Would an unrelated rename elsewhere in the tree oblige an edit here? C churns; A cannot (it is gone); B should not.
4. **Can a reviewer assess it without mentally compiling?** If a lens must act as a type-checker, it is C.

**What good Tier B looks like — the positive form of the rule (the HUM LEAD, 2026-08-30).** The rule must not be
read as "no code in plans". Code blocks **belong** in a PLAN artefact wherever they materially improve
understanding, *because that is exactly what a human principal engineer or architect hands to the junior and
senior engineers who will do the work.* A signature settles an argument that three paragraphs cannot; a
flow-control skeleton shows where the error path forks without asserting how it is written. Concretely, all of
these are welcome:

- **Exported signatures and interfaces** — the contract, in the language the reader will implement in.
- **Type definitions with their fields** — the data shape, where the shape *is* the decision.
- **Representative control flow with placeholder bodies** — the branches, the lock scope, the order of
  operations, with each body left as a comment.
- **Call-site sketches** — one line showing how a seam is consumed, when the consumption pattern is the point.

The convention that keeps these honest is a **visible placeholder**, so no reader mistakes a sketch for a
deliverable and no remediation round tries to "finish" it:

```go
// Illustrative (Tier B) — shape only; bodies are BUILD's.
func (s *Source) play(ctx context.Context, seg Segment) error {
    // TBFI in BUILD — resolve only on a generation change
    if s.gen() != seg.gen {
        // TBFI in BUILD — soft change plays as rendered; hard change re-renders (D-R4-2)
    }
    // TBFI in BUILD — a segment's failure is fatal to the segment; a hand-over line's is not (D-R4-1)
    return nil
}
```

**TBFI** — *to be filled in* — marks the boundary the anti-pattern is about. Above the marker is architecture a
reviewer can assess by reading; below it is implementation only a compiler can assess, which is why it waits for
BUILD. A block that has no TBFI markers and no elisions has almost certainly crossed into Tier C.

*How this refines the tiers.* Nothing above is a relaxation — it is the standard the rule was always defending.
The four B-vs-C tests still decide: a skeleton whose bodies are markers cannot churn on a rename, needs no
mental type-checking, and loses nothing but typing if deleted. The failure this feature actually suffered was
not "a signature appeared in a plan"; it was 106 blocks of finished bodies, fixtures and expected strings.

**The standing posture: we build during BUILD.** The **zero** target below is Tier C's alone — Tier B is not
rationed toward zero, and a plan with too little of it fails RN-2's comprehension questions instead. The target
for Tier C in a PLAN artefact is **zero**, not
"kept small". Every line of C is code that will be re-written, re-reviewed and re-evaluated against a compiler
in BUILD anyway — so writing it at PLAN buys nothing and costs a review surface that churns (this feature:
three red-team rounds whose findings were mostly C's own typos, and each remediation seeded the next round's).
If a task needs prose to be executable, the prose is missing, not the code.

**The one narrow exception — a spike-preserved implementation detail.** Admit a Tier-C fragment only when a
spike actually ran *and* the finding is about **how**, not merely **what**: an implementation choice that
materially changed an outcome, where prose alone would lose the distinction BUILD needs. The shape of such a
finding is always comparative — *"implementing it as X cost 10 s per boundary on Linux; the same seam as Y did
not"*, *"the obvious loop re-entered the lock; hoisting the snapshot above it did not"*. Four conditions, all
required:

1. A spike ran and produced a **measurement or a failure**, not an opinion.
2. The discriminator is the *implementation*, not the interface — the contract alone cannot carry it.
3. A BUILD engineer starting from the contract would plausibly re-derive the **bad** variant.
4. The fragment is the **minimum** that shows the discriminating difference — the guard, the ordering, the
   two-line shape — never a whole function or test.

Record it as evidence, not as a deliverable: mark it **spike-preserved**, attach the measurement, name the
variant it rules out, and say plainly that BUILD re-implements and re-evaluates it. Everything else the spike
touched is deleted with the worktree. In practice this exception should fire once or twice in a feature, if at
all; a plan with several of them has slid back into `AP-PLANCODE-01` under a new name.

**The spike protocol** (so Tier A does not drift into Tier C): state the question before writing anything;
time-box it; run it; record **question · method · result · decision** in the batch's build log and, when it
changes the design, as a contract or a risk in the plan; delete the code. The framework already sanctions this
shape — it is the remedy `AP-ASSUME-01` prescribes ("mental-model spike"). This feature's failure was
substituting *written* code for *executed* code: the spike-shaped work was typed into the artefact instead of
being run, so it bought the appearance of verification and none of the fact.

**Calibration candidate (A2DH):** a PLAN-phase rule — *"For compiled languages, plan code is illustrative only;
the red-team's PLAN lenses review contracts and architecture, not compile-ability; RED/GREEN bodies are written
at BUILD against the real tree."* — a corresponding `writing-plans` calibration (task = file · symbol · contract · test intent · verify command,
never a pasted body); an explicit **PLAN spike** step in the phase sequence for compiled languages (the Tier-A
protocol above, with its results recorded as decisions — the counterpart the `AP-ASSUME-01` remedy already
implies but the PLAN phase does not schedule); and `AP-PLANCODE-01` added to the anti-pattern catalogue the red-team and plan-review
skills already cite, so a PLAN-phase lens can name it rather than re-deriving it feature by feature.

**Carry to:** the DEBRIEF's lessons; `project-watchpost-follow-ups` (A2DH calibration candidates); the PLAN skill's
guidance in the framework repo.

---

## RN-2 — 2026-08-30 (PLAN, after round 4) — the Junior Developer persona is a *comprehension* lens at RCC and PLAN

**Raised by:** the HUM LEAD, reading the round-4 Junior Developer report. **Scope:** A2DH calibration and future
projects; this feature keeps what it has ("the money is already spent").

**The item.** The Junior Developer persona has one role at BUILD/REVIEW — *can I execute this literally, does it
compile, does the gate run* — and a **different** role at RCC (DISCOVER) and PLAN. In those phases the persona
exists to answer seven questions in the affirmative, and its findings are the answers that came back "no":

1. Can I understand this plan as a human engineer with **1–2 years** of experience in this language?
2. Is the architecture presented so that a junior engineer can follow it — and follow *why*, not only *what*?
3. Could a junior engineer **explain this plan to a non-technical stakeholder** in their own words?
4. Could a junior engineer derive **useful follow-up questions** from it (rather than only "what does this mean")?
5. Is the documentation **cohesive enough that I do not have to do extreme deep research to do my job** — RCC and
   PLAN having already done that research?
6. Where another axis or role (Distinguished Engineer, Performance, Safety) has **prescribed a spike**, are its
   goal and instructions clear enough that a *human* junior assigned the task instead of an agent would have a
   high likelihood of **affirming or refuting the hypothesis**?
7. Is the plan **free of jargon and esoteric technical prose** that only a Senior+ reader would parse? Where such
   content is essential — and it usually is — it is **not removed**: it is reworded and grounded so a junior can
   understand it. *"Don't write to sound smart, write to be clearly understood."*

**Additive.** These questions extend the persona's existing configuration; they override it only where they
directly conflict. Nothing here relaxes the persona's other duties (gate literalism, scaffolding, artefact
conventions) — it re-points them at the phase's real artefact, which in RCC/PLAN is *prose a person must act on*.

**Why this matters, from this feature.** Our PLAN-phase Junior lens spent its rounds acting as a compiler —
extracting blocks, running `go vet`, finding undefined symbols. That was valuable **only because the plan
contained Tier-C code it should not have had** (`AP-PLANCODE-01`, RN-1). With the code stripped to task shape
there is nothing to compile at PLAN, and a persona still pointed at compile-ability would report "nothing to
find" on a plan that might be unreadable. The comprehension questions are what the phase actually needs
answered, and they are the questions no other lens asks: the Architect asks whether the design is right, the
axes ask whether it is complete, consistent, safe and justified — **none of them asks whether a person of
ordinary experience can pick it up and act on it.**

**Detection the persona can apply** (each maps to a question above): a term used before it is defined; a
sentence a reader must already know the answer to in order to parse; a task that cannot be started without
reading three other documents; a spike with a method but no stated hypothesis or success criterion; an
architecture section that states a mechanism without stating the problem it solves; a paragraph a junior could
not paraphrase to a PM without inventing content.

**Calibration candidate (A2DH):** make the Junior Developer persona **phase-conditioned** in the red-team skill
— the seven questions above at RCC/PLAN, the literal-execution checks at BUILD/REVIEW — and give its PLAN-phase
report a fixed shape: per question, *answered / not answered*, with the passage that fails and a rewrite
suggestion. Pair it with `AP-PLANCODE-01`: once plans carry no implementation code, this is the only sensible
form of the persona at PLAN.


---

## RN-3 — the `SPIKE` skillset is undefined, and PLAN is where it is most needed

**The gap.** A2DH names spikes constantly — the PLAN phase directive asks for them, `AP-PLANCODE-01` (RN-1)
carves its single Tier-C exception out of them, and this feature ran four — but there is **no spike skill**. No
charter format, no timebox rule, no isolation rule, no output contract, no disposal rule. Each spike was
improvised, and the quality varied accordingly.

**What this feature's four spikes actually looked like:**

| Spike | Question | How it went |
|---|---|---|
| **go-toml round-trip** (D-R2-5 → D-R4-3) | does the pinned encoder preserve keys it does not understand? | The model spike. A yes/no hypothesis, a fixture that answered it, a version bump as the finding. Ran in a worktree, left no residue, and its result is now a **required** P1 task. |
| **Compile-first plan assembly** (round 4) | does the designed architecture actually build and pass under `-race`? | High value — proved `cast`, the Limiter and the whole broadcast seam green, and found three Criticals no prose round caught. But it had no charter and no timebox; it ran because a lens chose to, and its throwaway worktrees needed manual cleanup (one was left behind). |
| **Resident Piper** (AX-1 → E-1) | is a warm process worth the complexity? | **Never ran.** It could not run on the developer's machine, which is a legitimate result — but it was never *chartered as unrunnable*, so it sat as an unanswered assumption through three rounds until it became an escalation. |
| **The marine zone** (AX-7 → E-8) | is the nearest-zone geometry "free", as the ruling assumed? | Answered by reading an API, not by running one. Correct method, but nothing in the artefacts distinguishes "we checked" from "we assumed", which is why it needed a HUM LEAD ruling at PLAN exit. |

**What a spike skill should require.** A spike is not "go try something"; it is a bounded experiment with a
written contract, and every field below exists because one of the four above suffered for its absence:

1. **Hypothesis** — one falsifiable sentence. *"The pinned encoder drops unknown keys on save."* Not "look into
   config handling."
2. **Why it must run now** — what PLAN decision is blocked. A spike with no blocked decision is BUILD work in
   disguise; that is the churn `AP-PLANCODE-01` names.
3. **Method and the smallest instrument** — the exact command or fixture. If the instrument is bigger than the
   thing it measures, the spike is mis-scoped.
4. **Success criterion, stated before running** — including what a *negative* result looks like. This is the
   field that would have converted "resident Piper" from a dangling assumption into a clean recorded finding.
5. **Timebox**, and what happens when it expires: the spike **reports what it learned and escalates**, it does
   not extend itself. An untimeboxed spike becomes an implementation, which is exactly how a PLAN acquires
   Tier-C code.
6. **Isolation** — a throwaway worktree, never the feature tree, with cleanup named in the charter (`git
   worktree remove --force` + `prune`). This feature lost a `go.sum` to a spike three times and left a worktree
   behind once.
7. **Output contract** — the finding as a **decision or an escalation**, never as a diff into the plan. The one
   thing a spike may carry into PLAN is RN-1's narrow exception: *an implementation detail whose loss would cost
   BUILD the finding* ("approach X was slow for reason R; Y is not"). That belongs in the spike record, cited
   from the task — not pasted into the batch file.
8. **Disposal** — code deleted or filed as prior art with a "not the plan" banner; artefacts left clean;
   dependency and lockfile state restored and verified.

**Two failure modes worth naming as anti-patterns in their own right:**

- **The unbounded spike** — no timebox, no criterion, so it keeps going until it *is* the feature. Round 4's
  compile-first pass was a hair's breadth from this; it was saved by the fact that the code was thrown away.
- **The unrun spike** — chartered, never executed, never closed, and silently re-read as an assumption by every
  later round. Worse than no spike, because the artefacts imply evidence that does not exist. A spike skill must
  make **"could not run, here is why"** a first-class, recordable outcome.

**Calibration candidate (A2DH backlog).** Author a `SPIKE` skill with: the eight-field charter above; a
`06_docs/.../02-analysis/spikes/<slug>.md` record format (charter · what ran · result · decision or escalation ·
disposal); a phase rule that PLAN-phase spikes are chartered **before** the red-team round that would depend on
them; and a red-team check — probably on the Principal Architect lens — that every assumption a plan rests on is
either cited to a spike record or explicitly listed as an unverified assumption. Pair it with `AP-PLANCODE-01`:
the spike is the *only* sanctioned route by which implementation detail reaches a PLAN artefact, so the skill
that governs spikes is what keeps that exception narrow.


---

## RN-4 — 2026-08-30 (BUILD, P1 gate) — "UAT-able alone" is not the same as "a user can judge it"

**Raised by:** the HUM LEAD, on being handed a P1 UAT sheet that asked him to hand-edit a TOML file.

**The item:** every batch in this plan is marked *UAT-able alone*, and P1's was written as end-user UAT. It was
not. Its steps were: hand-write a cast into `config.toml`, corrupt a value to see the error, and grep the file
after a save. **No listener does any of that** — they open Setup and pick a voice, and Setup does not exist
until P4. The sheet was a developer regression guard wearing a UAT label.

**The distinction to hold, and the rule it yields:**

| | Agent / developer verification | End-user UAT |
|---|---|---|
| Asks | does the contract hold? | is the thing good? |
| Drives | files, flags, fixtures, seams | the interface a person actually has |
| Can run | at every batch | only once a listener-facing surface exists |
| Judges | pass/fail against a spec | *taste* — wording, timing, whether it feels right |

> **A batch is "UAT-able alone" only when it puts something in front of the user that a user can operate.**
> Otherwise it is *agent-verifiable alone*, which is a real and useful property — it is what makes a batch
> shippable and reviewable — but it must be labelled as what it is, and the HUM LEAD's time must not be spent on
> it. Name the batch where the listener-facing judgement actually lands, and defer to it.

**Why it matters beyond a filename.** Mislabelling it costs three things: the HUM LEAD's attention on work only
an agent needed to do; a false sense that the feature has been *judged* when only its plumbing has been
*checked*; and — worst — it invites the batch plan to be written around what is easy to demonstrate rather than
around what is architecturally right to build first. P1 is correctly ordered *because* it is invisible.

**A second, sharper lesson — get the threat model right before justifying a guard.** The HUM LEAD challenged
whether the config-file guards had any user-facing entry vector at all once a binary is compiled and shipped.
Half the challenge was right (hand-editing is not the normal path) and half rested on a wrong premise (the file
*is* exposed — `~/.config/watchpost/config.toml`, documented by path in the README, with a "For tinkerers"
section that hands the reader TOML to paste). But the useful correction was to the **justification** I had
written, not to the code: NFR-5's real vector is not "a user mistypes a key", it is **one config, two binaries**
— a synced or shared home where one machine runs 0.14.0 and another still runs 0.13.0, or a downgrade after a
bad release. The older binary need not *offer* the cast to destroy it; it only has to save once, for any reason.

*Rule:* when a guard's stated justification is "a user might do X", check whether a user can actually do X, and
whether X is really the dominant vector. Here the honest vector was more compelling than the one I had written
down — but I would not have found it without being asked.

**Calibration candidate (A2DH):** in the batch plan template, replace the single *UAT-able alone* field with two
— **agent-verifiable alone** (always required) and **user-testable alone** (only where a surface exists, and
naming the batch it defers to). Add a REVIEW-phase check that no batch claims end-user UAT for a batch with no
user-facing surface. Pair it with [[feedback-junior-persona-comprehension]]: the same instinct — *who is the
reader, and what can they actually do?* — is what both are enforcing.

---

## RN-5 — I read the wrong component and told the HUM LEAD the kit could not do it (0.14.0, UAT 2026-08-30)

**Raised by the HUM LEAD:** *"I hope it's a go-studs table, because if so we can make
'endpoint' a fill."* I answered that it could not: no fill, header-only sizing, byte-based
truncation. **That answer was wrong**, and the HUM LEAD corrected it — *"go-studs tables can
also specify fixed width for columns, if we want 0 there's a lot of settings that are useful:
'fill', 'fit', 'truncatable'."*

**The mistake.** go-studs vendors **two** table components. I read `components/table.go`
(`AdvancedDataTable`) — the one whose `Width` and `Truncate` fields really are never read —
concluded the kit was inadequate, and wrote the finding up as a table of upstream gaps. The
component the HUM LEAD meant is `components/data_table_row.go`, and `ColumnDefinition` there
declares exactly what was asked for: `Width` (0 = take the slack), `Fill`, `Truncatable`,
`TruncatedMinWidth`, `TruncationTail`, `MinWidth`, `MaxWidth`, `Alignment`, `Color`,
`HeaderColor`, `NoLeadingGutter`, and a table-level `GutterWidth`.

**The lesson is about the shape of the claim, not the file I opened.** "The kit cannot do X"
is a claim about a whole package, and I made it from one file. A negative about a dependency
needs a search across it before it is said out loud, because it is the kind of answer that
ends the conversation: the HUM LEAD had no reason to doubt it, and a rule they had just
restated — *"whenever we need a table, we use a go-studs table, that's why we vendored it"* —
would have been quietly broken on my say-so. **Before answering "the library can't", grep the
package for the capability, not the file you happen to have open.**

**What the migration actually found.** Two real gaps, both worked around in
`platform/render/status_table.go` so that modes/tty never sees them:

1. **The gutter starts at column three.** `gutterBefore(i)` is `i >= 3` — the dashboard
   tables the component was written for lead with a prefix, a number and a name that butt
   together on purpose. Ours do not, and `PROVIDERS` ran straight into `STATUS`
   (`COOPS · COOPS-OBSREF OK`). `statusGutters` folds the missing air into the columns that
   need it rather than forking the kit for a layout preference.
2. **The kit and `render.Width` disagree about ambiguous-width runes.** The kit's width table
   reads `✔` as two cells where ours reads one, so a fixed column sized for the health mark
   came out a cell adrift on every row. The fix was not to argue about the rune: the mark
   moved **into** the fill column, where the kit can only under-pad (which `PadTo` corrects)
   and can never over-run (which costs content). It reads better too — a failing host is a
   red NAME rather than a red mark beside a grey one.

A third thing worth remembering: **squaring a block off by clamping is not a fix.** The first
migration produced lines that were all exactly the inner width and had silently eaten
`1m 30s` down to `1m 30`, `1.5M` to `1.5`, and the S off `ROWS`.
`TestStatusTablesAreRectanglesWithNothingClipped` now pins both halves together, because
satisfying either one alone is what went wrong.

**Upstream candidates (M6).** A configurable gutter start on `DataTableRow`, and a width
function that agrees with `go-runewidth` on ambiguous-width glyphs.

## Recurring error classes, and whether a deterministic guard could remove them

**HUM LEAD, 2026-09-03:** *"if you notice you're making the same kind of errors over and over, we
need to ensure there are deterministic guards (if possible) to help remove those errors."* Raised
after reading back the T2.2 history, where the same shapes recur across commits.

Counted over 0.14.0 T2.2 and its four review rounds. The point of the table is the last column:
**three of these five are mechanically detectable, cheaply, and are not detected today.**

| Error class | Count | Deterministic guard possible? |
|---|---|---|
| **Mutant removes a USE, not a rule** → tree will not compile, verdict INVALID | 6 | **YES, cheap.** The anchor-check already applies every mutant to a throwaway copy of the tip; make it run `go build` there too and fail the ones that do not compile. Catches all six BEFORE a sweep, in seconds |
| **Stale mutant anchors** — a fix moves the line a mutant patches, and the mutant silently stops applying | 6 across the release | **YES, already built but untracked.** The apply-to-throwaway check exists only as a scratch script. It belongs beside `run.sh` with a `make` target, so it runs as a gate rather than when I remember |
| **Instrument bugs in the sweep wrapper** — read the wrong line; scoped too narrowly; reported a clean run over mutants that never ran | 3 | **YES.** The repo already gates its linters with positive controls (`make gate-controls`). The mutation wrapper has none. Feed it synthetic mutants whose verdicts are known — one CAUGHT, one SURVIVED, one UNAPPLIED, one INVALID — and assert it reports each correctly |
| **Vacuous invariants** — a check that cannot fail | 3 | **PARTLY.** The narrow shape (`invariant.Check(X)` directly inside `if X`) is a mechanical AST match and worth a lint. The general case is not decidable, and two of the three were caught only by reading |
| **False claims in commit bodies and comments** — numbers quoted from a stale measurement, a guarantee stated wider than the code enforces | 6 | **NO, not mechanically.** These are the residue that review exists for. The partial mitigation is procedural and already a calibration: re-derive every number before quoting it |

**Recommendation:** build the first three as gates before Phase 3, since Phase 3 is the highest-risk
task in the release and is where a silently-unapplied mutant costs the most. They are small — a
build step, a tracked script with a make target, and four synthetic fixtures.

## Adversarial cross-model review as a standing option — FLAGGED, NOT ADOPTED

**HUM LEAD, 2026-09-03, recorded verbatim because it is a live option rather than a decision:**

> *"The other option is I have work constantly reviewed adversarial by different models like Codex
> and make determinations on which model should take over. A2DH is model agnostic, but constantly
> switching from you <-> codex would be another layer of cost and context management. It might be
> worth the investment."*

**Status: flagged, no action taken, at the HUM LEAD's instruction.**

What the evidence here says about it, for whenever it is decided:

- **The find-rate is the argument for it.** Four fresh adversarial reviews across two fixes returned
  ADDITIONAL FIXES REQUIRED four times out of four, and one finding was a defect the fix under
  review had itself introduced. Every gate — build, vet, race, allocation pins, P10, 119 mutants —
  was green while those defects stood.
- **The reviewers were the same model as the author**, and still found them, because what makes a
  reviewer effective here is being FRESH (not carrying the author's misunderstanding) and being
  required to RUN something. A different model would add independence of training as well as
  independence of context, which is strictly more — the open question is how much more, and that is
  measurable: run one round both ways on the same finding set and compare.
- **The cost the HUM LEAD names is real and is the deciding factor**, not the capability: handing
  over context between models is where the framework has the least support today. A2DH being
  model-agnostic makes the review portable; it does not make the CONTEXT portable, and the record —
  build log, rulings, mutants, follow-ups — is what would have to carry it. That the record was
  found contradicting the code twice this release is a caution about relying on it for handover.
- **A cheaper middle option exists:** cross-model review at PHASE EXITS only (where red-team already
  runs and the context is already written down for the report), rather than per fix. That keeps the
  handover cost bounded to moments where the record must be complete anyway.
