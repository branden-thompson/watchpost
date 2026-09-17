# 0.16.0 — handoff

**Status: BUILD, not exited. Do not ship.** Seven blind reviewers returned NOT CLEAR on 2026-09-16.
The product is in good shape; the layer built to *prove* the product is not, and it stopped converging.

**Read this before touching the gate layer.** The "What did not work" section is the reason this
handoff exists, and it is more useful than the findings list.

---

## 1. Where the release actually stands

| | |
|---|---|
| Branch | `feature/0.16.0-broadcaster-ui` |
| Pushed | `origin/main` @ `fd761ab` (0.15.0) and `origin/feature/0.16.0-broadcaster-ui` @ `66dc88d` — **both diverged from local HEAD; neither carries the rewritten history** |
| `make verify` | **ALL GATES GREEN on `7c916c2` (2026-09-17, 16 min, GNU Make 4.4.1)** — 24 gates including the 379-anchor mutant check. The four verifies before it each stopped at a gate only a full run reaches (`tidy`, `race`, `lint`, `p10`), and every one of those notifications said exit 0 while the log said `Error 1` |
| `go test ./...` | green |
| `make p10` | 0 live, 0 unmatched, 0 unratified, 152 ratified rows |
| Mutation sweep | **NOT RUN.** Deliberately deferred — see §4 |
| Open findings | `06_docs/follow-ups.md` — rounds one through eight on the gate layer are disposed there; F-156 (port the 23 `scripts/` sources to Go) is the standing backlog; F-115…F-151 from the earlier rounds where not closed since |

**The product reviewed clean.** Code Quality found zero unused parameters, zero dead functions and
zero live P10 findings across 62,419 added lines. Business Quality probed FR-3.3's own surface and
found no display field that can diverge from the schedule. One real product defect was found and
fixed: FR-8.10, below.

### Git history was rewritten once, deliberately

Three `authoring` tool binaries (14.1 MB) had been committed and merely untracked. On the HUM LEAD's
ruling — *published* history is never rewritten, unpublished history is cleaned — `94b654b^..HEAD`
was rewritten with `git filter-branch --index-filter` to drop them. Verified afterwards: **zero
objects over 400 KB in the whole release range**, and `git diff pre-blob-rewrite HEAD` is empty, so
every tree is byte-identical.

**To finish the job when you are satisfied** (the blobs still exist in the local object database,
held alive by the safety net; they cannot reach a remote because only the rewritten branch is
pushable):

```
git tag -d pre-blob-rewrite
git update-ref -d refs/original/refs/heads/feature/0.16.0-broadcaster-ui
git reflog expire --expire=now --all && git gc --prune=now --aggressive
```

---

## 2. What was fixed this session

**Product (1).** `FR-8.10` — `app/radio.go` discarded the read-path alerts fetch error, so a 502 left
`loc.Alerts` empty, the composer emitted nothing, and the listener heard a report that sounded
complete and named no hazard. An empty alert list and an unreachable alert feed were the same value
and opposite facts. Now `Reports.HazardsUnavailable` carries the failure and the report says so
aloud, in a **literal string rather than a script lookup** — a phrase the script library does not
carry is silently not spoken, which is the failure class being fixed.

**Identity (1 class, 22 files).** `/Users/<account>/…` in 12 files and agent-harness scratchpad paths
in 10, on both pushed branches. Fixed forward per ruling: ten dead artefacts deleted, one spike
script defaulted, two records scrubbed of the account name while keeping their findings.
`TestThePublishedTreeNamesNoPersonOrMachine` now asks the whole index; the pre-existing rule was
scoped to a single file and printed "no machine paths" while the class was live in 21 others.

**Gates (7).** `p10` was red and on none of the three lists; `dupes-selftest` and `lint-injector`'s
control ran nowhere; an alloc pin was excluded by its own `$` anchor; `required-gates.txt` was
missing three gates `verify` runs; the empty-branch hole in the report-kind guard; the tracked-binary
gate.

**Everything above was verified by constructing the defect and watching the gate fail.** That
discipline held throughout and is the reason the findings below are trustworthy.

---

## 3. What did NOT work — read this first

The HUM LEAD's observation, and it matches the evidence: *this pattern exists beyond one model or
harness.* What follows is the specific shape it took here, recorded so the next session does not
repeat it.

### 3.1 The defect lives inside its own guard

**Five instances in one day, every one found by a reviewer rather than by a gate:**

1. The test whose purpose is stopping two lists disagreeing carried a hardcoded second copy of one of
   its own lists (`ciOnlyOK`).
2. The gate written to catch *"controls that exist and are not run"* did not check that a control
   runs.
3. Its fix scoped *which targets* are read and still counted a **commented-out** line as a control.
4. The guard written to close the empty-branch hole **has** the empty-branch hole — a one-character
   `:=` shadow passes it.
5. The reason-floor test, written to close the exemption tables' last hole, is itself a hand-written
   enumeration whose own subject check cannot fire.

This is not carelessness about a detail. It is a systematic blind spot: **the author who has just
understood a defect class well enough to build a detector for it is, in that moment, the person least
able to see that member of the class in their own code.** The understanding and the blindness have
the same source.

### 3.2 Remediation rounds did not converge

Three consecutive rounds, each finding defects in the previous round's fixes, and the third round
made something **worse**: the build-path "fix" added a conjunct that narrowed the scan and reinstated
a hole the pre-fix version had caught (F-128).

**Convergence is the signal to watch.** Round 1 found gates that did not exist. Round 2 found gates
that could not fail. Round 3 found that the fixes for round 2 could not fail. The defect count per
round did not drop. **A remediation loop that is not converging should be stopped, not continued** —
that is the ruling that ended this session, and it was correct.

### 3.3 Exemption tables are the attack surface, and they accumulate

Eight exemption tables now exist across three files. The junior reviewer named the consequence
exactly:

> *"Every table is a one-line escape, and the only thing standing behind it is a reviewer noticing a
> plausible sentence. The gates that would be hard to silence — the derived ones — are the ones with
> no table."*

Each table was individually justified. Together they are an accumulation, not a design, and they grew
fastest during the rounds where the most gates were added quickest.

### 3.4 "Verified" meant "verified against the spelling I thought of"

`F-117` was closed **three times**. Each time a real attack was constructed and really did fail the
gate. Each time a cheaper spelling of the same attack was never tried:

- Round 1 tested step-level `if:` on the line after `- run:` → caught.
- Round 2 added job-level `if:` → caught.
- Round 3 (a junior, in minutes) tried `- if: false` — the key as the step's **first** entry, where
  the dash sits between the indent and the key — → **survived**, with all three lists agreeing.

Two documents still say "Both attacks verified". **A verification is only as wide as the adversary's
imagination, and the author is the worst available adversary.**

### 3.5 What DID work, and should be kept

- **Blind agents with a written brief.** Seven reviewers, none with session context. Convergence
  across independent reviewers measured the repository; **disagreement between them measured what the
  repository fails to RULE**, which a single reviewer cannot produce at all.
- **Making them argue.** Three juniors reviewed independently, then were given each other's reports.
  One proposed raising a ratified cap — the only change in the round that would have broken a written
  ruling — and another caught it by reading four lines further into the same file.
- **"Answer every question by name, including the ones you found nothing for."** A silently skipped
  question is indistinguishable from a passing one.
- **Constructing the defect rather than trusting a comment.** Every reviewer who did this found
  something; the findings that came from reading alone were the weakest.
- **Derived gates over listed ones.** Every gate that enumerates from the artefact — the git index,
  the Makefile, the AST — held up under attack. Every gate with a hand-written list was defeated.

---

## 4. Recommended next steps, in order

1. ~~**Do not add another gate.** Consolidate~~ **DONE 2026-09-16** — one model, one registry, two deletions (`1ed3a93`); then, after a blind adversary defeated the parsed model six ways, **the Makefile half was made to EXECUTE** (`fa022cb`): make and sh are the oracle, 68 specimens run on every invocation, and the attack lists were written before both. The CI half is an honestly-labelled ratchet. Then three more blind rounds each found the same defect — text standing in for execution — one layer down, until the stubs were made to RECORD what make ran (`8a5c121`): nothing in the executed half reads a recipe, and its first run against the shipped tree found a discard the regex could not see (O10). A sixth round found the scratch tree itself was the last lie — empty, where the repository is not — so the oracle now runs in a clone of the tree with stubs by PATH alone; a seventh shaped that clone as CI has it (detached, no tags, no ignored files) and tightened the instrument's joints. Round seven was the first with no single cause behind its findings. Round eight's count rose, and the look found the threat model: the oracle enforces DRIFT and declares EVASION (`gate-attack-list.md`, round eight); the instrument moved to `tools/gateoracle/` as a Go package with no shell in it, under two standing rules — everything is Go unless absolutely necessary, and test code is a product — both mechanised (`AP-SHELL-01`, the shell ledger). Round nine, the first drift-briefed round, found the fourth verdict — a tool's ABSENCE — and closed it; Criticals across nine rounds: 6 → 4 → 3 → 6 → 6 → 5 → 2 → 4 → 1. GNU Make ≥ 3.82 is required (F-153 closed). The original item, for the record — consolidate: one `assertRowsResolve(t, name, table, resolve)` helper
   collapses six near-identical staleness loops, and makes "eight tables" a design rather than an
   accumulation. Consider deleting `modes/tty/declset_test.go` (F-122) and the `declaredLocals`
   helper (36 lines that close one token and miss the shadow).
2. **Fix F-125…F-129 as one batch** — they are five holes in three gates, and fixing them separately
   is what produced them. **Have someone else construct the attacks**, or write the attack list
   *before* the fix.
3. **Fix the false published numbers**: F-143 (accepted-costs, wrong by 35×), F-136
   (lint-authoring's scope banner), F-145 (exposure statement), F-148 (traceability completeness),
   and the F-117 row that says CLOSED and OPEN in one line.
4. **Then the reader-facing gaps**: F-146 (README has no Broadcaster), F-147, F-144, F-123
   (`where-things-happen.md` has no Broadcaster rows).
5. **Then the mutation sweep** — 379 mutants, ~3 hours — for the build report's figures. **It was
   deliberately not run**: a sweep against a surface that is about to change produces numbers you
   throw away. Run it when the tree is stable.
6. **Then re-present BUILD exit** with corrected figures.

`06_docs/tooling-backlog.md` holds the ranked mechanisable rules **and the do-not-build list**. Read
the second half first.

---

## 5. Standing rules that were tested this session and held

- A reason is **RATIFIED, never self-issued** — four P10 exemptions went to the HUM LEAD rather than
  being padded to satisfy an invariant-density ratio the reviewer had independently called
  "gameable… a metric that rewards ritual is worse than none".
- **Read the LOG, never the notification** — three `make verify` runs reported exit 0 in the task
  notification and `EXIT=2` in the log.
- **Leave it better than you found it** — a check is scoped to the artefact, never to the session.
- **Measure the surface before building a ratchet.** Two FP estimates that were made from reading did
  not survive being counted.
