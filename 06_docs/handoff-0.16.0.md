# 0.16.0 — handoff

**Status: BUILD EXITED at `557a40f` (2026-09-17, HUM LEAD: "Recommendation approved; Go 4 Review").
IN REVIEW — remediations R1–R7 landed; REVIEW exit not yet presented. Not shipped.** The four-axis
REVIEW red team returned *do not ship* on 557a40f; every Critical was either fixed (the release path,
the traceability method, the fail-open verifiers, the F-150 fault class — twice, under two blind
reviewers) or is a HUM CALL recorded below. The figures in §1 are re-derived at the commit named there;
a figure with no commit beside it is stale and should be read as such.

**Read this before touching the gate layer.** The "What did not work" section is the reason this
handoff exists, and it is more useful than the findings list.

---

## 1. Where the release actually stands

| | |
|---|---|
| Branch | `feature/0.16.0-broadcaster-ui`; the REVIEW-exit code commit is `73bce2b` (2026-09-18), followed only by records |
| Pushed | `origin/main` @ `fd761ab` (0.15.0). The feature branch on origin is stale and will be deleted at SHIP; **the release is a squash-merge release branch** (HUM LEAD ruling, 2026-09-17), one commit onto `main`, so nothing in this branch's history — including the 15 MB blob at `6b1b621` — reaches the remote |
| `make verify` | **ALL GATES GREEN on `73bce2b`** (`dist/verify-review-73bce2b.log`, 2026-09-18); `p10` moved OUT of verify to `make quality` (phase-exit) at R1, because the release workflow runs verify and has no `a2dh` |
| `make quality` (P10) | 0 live, 0 unmatched, 0 unratified at `f7fe5fa` (153 ratified rows, `06_docs/p10-ledger.md`) |
| Mutation sweep | **REVIEW exit, on `73bce2b`: 380 mutants — 376 CAUGHT, 4 SURVIVED by design (`m16`, `m43`, `mBM1`, `mSC3`), 0 NO EVIDENCE**; `mBM2` is caught now. A first REVIEW sweep on `0963b36` was stopped at 183/380 when the last rulings arrived, because the figure has to come from the final commit. Blind spot: a `-race`-only detector reads as SURVIVED |
| Requirements | 60 in `requirements.md`: 31 traced by ID, 13 by a named test without the ID, 4 OPEN with follow-up rows (FR-3.6 F-161, FR-8.3 F-157, FR-8.8 F-162, FR-9.2 F-160), the rest closed/partial/superseded in the red team's own rows — derived by roster test NAME scoped to this release (`build-report.md`, ruling 9) |
| Open findings | `06_docs/follow-ups.md`: F-156 (port the shell) is the standing backlog; F-157…F-163 are this REVIEW's rows; three HUM CALLS are open (rulings 10, 11, 12 below) |

**The product reviewed clean.** Code Quality found zero unused parameters, zero dead functions and
zero live P10 findings across the release diff (68,796 added lines `fd761ab..73bce2b`, 47,611 outside `06_docs/`). Business Quality probed FR-3.3's own surface and
found no display field that can diverge from the schedule. One real product defect was found and
fixed: FR-8.10, below.

### Git history was rewritten once, deliberately

Three `authoring` tool binaries (14.1 MB) had been committed and merely untracked. On the HUM LEAD's
ruling — *published* history is never rewritten, unpublished history is cleaned — `94b654b^..HEAD`
was rewritten with `git filter-branch --index-filter` to drop them. Verified afterwards: **zero
objects over 400 KB in the range as rewritten** (the 15 MB blob at `6b1b621` came LATER, and the
squash-merge ruling is what keeps it off the remote), and `git diff pre-blob-rewrite HEAD` is empty, so
every tree is byte-identical.

**Moot under the release ruling (2026-09-17):** the release is a squash-merge branch — one commit
onto `main` — so neither the rewritten range nor the 15 MB blob at `6b1b621` (found by the hygiene
reviewer at REVIEW) can reach the remote, and the feature branch is deleted on origin at SHIP. The
local cleanup below is optional housekeeping, not a release step:

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

The gate-layer history that used to sit here (nine adversarial rounds on the oracle, the consolidation,
the RECORD/DRIFT/EVASION threat model) is in `06_docs/gate-attack-list.md` and `quality-observations.md`;
this list is what is left to do.

1. ~~HUM CALLS still open from REVIEW~~ **RULED 2026-09-17 ("recommendations approved") and done:**
   ruling 10 — the radio diagnostic names places by label, never by coordinate, and FR-9.4's gate
   covers it (`TestTheRadioDiagnosticNamesPlacesNotCoordinates`); ruling 12 — the README's Broadcaster
   section carries its key table; F-163 — a `Run: 0` escalation is shown by its reason alone, and a
   decline does not break a run.
2. **REVIEW exit**: `make verify` and the mutation sweep on the final REVIEW commit, the figures in §1
   replaced from those runs, the exposure statement re-run at that commit, then present.
3. **Before the release branch is cut — a final local build and a regression pass of everything
   signed off before BUILD exit** (HUM LEAD, 2026-09-17: REVIEW made "lots of code changes to the
   product", which is fine, and the signed-off functionality must be shown not to have regressed
   before `release/v0.16.0` exists). Build from the REVIEW-exit commit with the documented
   commands, run the UAT checklists that were signed off, and record the result beside the exit.
   **Done 2026-09-18 (HUM LEAD): the Observer pass — good; the Broadcaster pass — good, except the
   thirty-minute rotation (`p3-uat.md` case 10), which is not signed off and is carried into VALIDATE
   with its state to be ruled.** REVIEW APPROVED; GO 4 VALIDATE.
4. **Before SHIP, two rulings**: `p3-uat.md` case 10 (not run, or not held), and F-164 (a
   fall-through to synth under the operator's cut-over — release, hold, or re-tune; the code releases).
5. **SHIP**: the squash-merge release branch (one commit onto `main`), the tag, the push, the feature
   branch deleted on origin — every step outward-facing and on the HUM LEAD's word.
6. **0.16.5 (the dedicated quality pass)** opens with the rows tagged for it in `follow-ups.md`:
   F-156 (shell → Go), F-157 (tier three), F-159 (carry the fault on `Publish`), F-160…F-162 (the
   open requirements), the tenth oracle round, and F-137's undetected history comments.

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
