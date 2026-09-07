# The remediation review loop

A process for fixing red-team findings on **safety-critical code**, written down here because it was
derived the hard way: two rounds of remediation on 0.14.0 produced roughly thirty new defects, and
every one of them passed the gates.

**Candidate for A2DH.** This belongs in the framework as a skill beside `red-team` — provisionally
`critical-analysis/remediation-loop` — gated on a project being marked safety-critical (or SEV-0/1).
It is recorded here first because this is where the evidence is.

## The problem it solves

Red-team finds a defect. The author fixes it, writes a test, the gates pass, and the finding is
marked Fixed. Nothing in that sequence checks whether the fix **works**, because the person who
misunderstood the defect well enough to ship it is the same person judging whether it is now
understood.

Measured on watchpost 0.14.0, that produced six repeatable failure modes:

| Failure mode | What it looks like |
|---|---|
| **Fixed the finding, not the defect** | The symptom named in the report stops reproducing; the defect does not |
| **Fixed one of N paths** | Two surfaces share a rule; only the one in the report is corrected |
| **Test written to match the fix** | The test constructs the input the fix expects, which the pipeline cannot produce |
| **Fixed the artifact, left the references** | The thing is renamed or deleted; everything pointing at it still points |
| **Asserted what the code does not do** | The comment or commit message states the intended invariant, not the implemented one |
| **New defect created by the fix** | The remediation opens a hole the original did not have |

Gates cannot see any of these. They are all *semantic* — the code compiles, the tests pass, the
allocation pins hold, and the product is still broken.

## The loop

**One finding at a time.** Never batch: a batched remediation hides which fix the reviewer is
rejecting, and the failure modes above are per-fix.

1. **Author writes the failing test FIRST, through the pipeline's real entry point** — not through
   the fixed function's own signature. Run it. *Watch it fail.* A test that has never failed proves
   nothing about the defect.
2. **Author fixes.** Re-run: the new test passes.
3. **Author re-runs the ORIGINAL finding's test, if the previous round wrote one.** If it now fails,
   that test was probably never valid — audit it rather than bending the fix to satisfy it.
4. **A fresh adversarial reviewer** — a Distinguished Engineer wearing the axis or persona the
   finding came from — receives the original finding, the diff, and the author's reasoning, and
   answers four questions, independently, with evidence:
   - **Does this actually work?** Verify it by running something, not by reading the diff.
   - **Does it introduce a new problem?**
   - **Would a junior developer understand what is happening here?**
   - **Can I independently verify the author's claims?** Every claim in the commit message and the
     comments is in scope.
5. **The reviewer returns LGTM or ADDITIONAL FIXES REQUIRED**, with evidence either way. "Looks
   fine" is not a verdict; a reviewer who ran nothing has not reviewed.
6. **Iterate 2–5 until LGTM.** Then, and only then, the next finding.

## What the author owes the reviewer first

Measured over the first two remediations: the reviewers' rounds went almost entirely to checks the
author could have run. Each of these moves work out of a 40-minute review round and into a
two-minute check.

- **Mutate before handing over.** Delete each line of the fix and run the suite. If it stays green,
  the new test does not pin the new code, and the reviewer's round will be spent discovering that.
  Three of ten findings in one round were exactly this.
- **Enumerate the consumers before fixing.** For a data-flow defect, list every reader of the value
  and state in the commit which ones the fix covers. *Fixed one of N paths* produced the blocking
  finding in both rounds: a fix correct at the reported surface, with the same defect intact one
  layer up.
- **No claim without a command.** Every behavioural sentence in a commit message or comment needs a
  run behind it, or it does not go in. Claim-checking is the single largest consumer of reviewer
  effort, and four of nine checkable claims in one round were false.
- **Hand the reviewer the reproduction.** A review that starts from a runnable case comes back
  faster and with better evidence than one that starts from a diff.
- **Re-run the PREVIOUS review's mutants too, verbatim.** Author-invented mutants only probe what the
  author already suspects. The ones a reviewer invented are, by construction, the ones that slipped
  past — carry them forward as a regression set for the review itself.
- **Assert the fixture is valid before asserting the behaviour.** Where a fixture passes through a
  validator — an id grammar, an active window, a classifier — check it is *accepted* first. Three
  vacuous tests here were invalid inputs rather than wrong assertions (an alert id the CAP grammar
  rejects, an alert expiring exactly at `now`), and each passed while proving nothing. This is
  *test written to match the fix* wearing a different hat, and it is the most repeated mistake in
  the record.

## Corrections to this round's record

Claims made in commit messages that the reviews disproved. They are recorded here because a wrong
claim in a commit body is not self-correcting — the next reader takes it as evidence.

| Commit | The claim | What is true |
|---|---|---|
| `dde48d4` | "All six mutants of this change now fail a test." | M4 (`keepWorst` deleted from `capPerLane`) survives — the fixture puts the severe event at index 0, so insertion order saves it whatever the sort does. M1 as written does not compile. |
| `dde48d4` | The lane floor closes the starvation. | It closes the **count** bound only. `capBurst` re-sorts by severity, putting the quiet lanes at the tail, which is where `breakingCap`'s 30-second cut lands. Reproduced: 6 s narration, 5 warnings per cycle, 8 cycles — 40 reads, the hurricane never read. |
| `d01cfaf` | "All three of those mutants fail it." | Only the combined mutant failed. The two id guards were mutually redundant, so neither was observable end-to-end alone — and one of them was dead. Closed in `de33575`+: the redundant guard is deleted and the live one is pinned directly, where it can fail. |
| `07f762e` | "Four mutants of this change now fail a test, including one that disables parking altogether." | Two of the four survived; one did not compile. Nothing pinned parking at all. The commit is reverted (F-16). |

**All four came from the same broken instrument**, since replaced (`06_docs/mutants/run.sh`): the sweep
decided a mutant was caught by grepping `go test` output for `FAIL`, and a mutation that breaks the
build prints `FAIL … [build failed]`. Roughly ten measurements were false, and four reached commit
messages as evidence. A measurement that cannot fail honestly is worse than no measurement: it
converts an unpinned line into a pinned one in the record while changing nothing in the code.

## Rules that make it work

- **The reviewer is fresh.** It has not seen the author's earlier attempts, so it cannot inherit the
  misunderstanding that produced them.
- **The reviewer must run something.** Reading a diff reproduces the author's blind spot; executing
  the code does not.
- **The author's comments and commit message are review surface.** Half the failure modes above are
  claims, not code.
- **The reviewer may reject the test as well as the fix.** A passing test built on an impossible
  input is worse than no test: it converts an open defect into a closed one.
- **LGTM is the reviewer's to give.** The author does not self-certify and does not overrule; a
  disagreement that survives two rounds goes to the human.

## Cost, honestly

It roughly doubles the work per finding and it is worth it on hazard paths. On watchpost the
alternative was measured: without it, a remediation round closed 17 findings and opened about 30.
Scale it to the blast radius — a SEV-0 alert path earns the full loop; a doc typo does not.
