---
title: "0.16.0 Broadcaster UI — DEBRIEF (After Action Report)"
date: 2026-09-18
phase: REFLECT
sev: SEV-0
authority: HUM LEAD
status: "Shipped — v0.16.0 tagged on 237e2a5, release run 35398502049 green in 37 m 42 s, 8 assets published; origin carries main alone"
---

# 0.16.0 Broadcaster UI — DEBRIEF

**Problem (locked at DISCOVER, 2026-09-09):** *A person broadcasting hyper-local weather over a
two-way radio channel cannot see what is about to be transmitted, cannot change its order before it
goes out, and cannot confirm at a glance whether they are currently on the air — so they operate the
station blind, and their listeners hear the wrong thing at the wrong time.*

**Shipped 2026-09-18.** Solved on all three clauses: the operator sees the running order, changes it
through the Director that owns it, and reads the station's state in words the audio engine reports.
The two safety-class fixes of the release — the fault band and the cut-over — are the third clause
enforced against the code rather than the mock.

---

## 1. What was delivered

**The Broadcaster console**, a second full-screen surface beside the Observer: a station banner
(STOPPED, STANDBY, ON AIR, in words as well as colour, with the sentence that ON AIR means audio is
leaving this program and nothing more), an UP NEXT card beside an alert-takeover slot, a scheduled
line-up of sixteen cards the operator can open, move and drop, a location pool sized by the service
radius with its reach stated in the footer, a relay bed the main track can be cut over to, a Line-Up
Request window, and the station's own settings — the transmitter and the service radius, split from
the listener's default location on a ruling. Reached with `ctrl+b`, refused on the way back while ON
AIR, and its keys are data like the Observer's.

**The schedule's write side**, which did not exist. Promote, demote, drop and request go through the
Director; a fault on a live station reaches the operator in the station's own band after a run of
three, a place that could not be read sits out a cool-off through every door back in, and the
operator's cut-over survives the relay stepping. The read seam grades a dead voice as a fault and a
deliberate stop as routed, and the mutants that pin the difference are in the corpus.

**A privacy boundary stated exactly and held at the writer.** The tower's position stays in
`config.toml`, goes to the National Weather Service in the same forecast request every watched place
makes, and reaches the opt-in radio diagnostic only as a label and an opaque id — every coordinate
pair rewritten by the one function that writes the file, gated by a test that reads the file back.
The README and the settings form say the same sentence.

**The gate layer as a product.** The gate oracle became a Go package with no shell in it, running
`make` in a clone of the tree as CI has it, with stubs that record what ran and four verdicts a gate
can earn; nine blind adversarial rounds converged on a written threat model (drift enforced, evasion
declared). Two standing rules were mechanised rather than remembered: everything in Go unless
absolutely necessary, and test code is a product. The traceability table is derived by roster test
name scoped to the release, after two reviewers found the ID grep colliding across releases.

## 2. What went well

- **Independent review as a phase gate, counted.** Every exit had blind reviewers in their own clones,
  and every safety fix went to a fresh reviewer and was iterated to LGTM. The count is the argument:
  the two Criticals that reached VALIDATE — a workflow cap under the measured job, and a struct
  replacement wiping the operator's cut-over — were found by reviewers reading the tree from outside,
  not by the author's plants; and two of the five remediation rounds found the author's own fix
  wanting (a fault band with two owners; a redaction covering one line of seven).
- **The record said what happened.** Every figure carries its commit; the sweep log opens with its
  hash; the docs-quality reviewer's "the release-facing record contradicts itself on every headline
  number" was true at REVIEW's first exit and false at its second, and the difference is in the diff.
- **Rulings were asked for, not taken.** Twelve asks at REVIEW with evidence and a recommendation
  each; UX and semantics forks (the mute on air, one place at three miles, the fall-through under a
  cut-over) went to the HUM LEAD and came back as D-numbers. The release posture — never recommend
  accepting a known defect — held: nothing shipped with a known defect the record does not name.
- **The CI rounds were findings, as last release.** Three rounds on PR #20: a guard that read the
  clock twice, then green, then the cap raised to twice the measured job before the tag rather than
  after a tag with no release behind it.
- **The HUM LEAD's regression pass before the branch was cut**, on their own build, both surfaces and
  the thirty-minute rotation. It became a standing step and it earned its place: the captures that
  now show the console in the README came from it.

## 3. What to carry forward

- **A claim about what a file carries is held at the one function that writes the file**, and gated
  by a test that reads the file back with the sensitive value in play. A test of the line-builder
  proves the line-builder. (`quality-observations.md`, "A privacy claim is held at the writer".)
- **A wholesale write is a bundle of resets.** Narrowing it needs a reader of every field it used to
  reset, not only the one the finding was about; the fix that traded a lost cut-over for a false stall
  every rotation is the worked example. ("A line doing two jobs, fixed for one of them".)
- **When a state has a SET and a CLEAR, they travel one seam.** Two owners is a finding before any
  sequence is constructed. ("The second owner of a message is where the message is lost".)
- **A fixture that reads the real clock is a flake with a period**, and CI is where the run count is.
  Pin the clock; prove the guard sees a moving one. ("A guard that reads the clock twice".)
- **A plant is constructed outside the instrument's allow-list**, or it measures the allow-list.
  ("A plant that the instrument is designed to ignore is not a plant".)
- **A verdict is read from the test's own line, never a pipeline's exit; a plant's revert never
  shares a command with an uncommitted fix.** Both bit this release once. ("A pipeline's exit code".)
- **Instrumentation is bounded by ship intent.** Nine oracle rounds converged; the tenth is 0.16.5's.
  The HUM LEAD's sentence — "a good investment, but we do want to get the latest version out the door
  at some point" — is the rule.

## 4. Process review

**What the phases cost.** DISCOVER and PLAN closed 2026-09-09; BUILD ran to 2026-09-17 with the
gate-layer investment inside it (nine oracle rounds, the Go conversion, the standing rules); REVIEW,
VALIDATE and SHIP took 2026-09-17 and 2026-09-18. The product itself reviewed clean at BUILD exit;
the two days after were the instruments and the record catching up to it, then the reviewers
catching what the instruments could not.

**What the reviewers found that the author did not**, by phase: REVIEW — the release path (`p10` in
verify with no harness on the runner), the traceability method, four verifiers that failed open, the
fault class that could never fire on a live station, the privacy sentence; REVIEW remediation — the
two-owner band, the ghost `Finished`, the off-air count, the one-of-seven redaction, two red tests in
a commit; VALIDATE — the workflow cap, the bed replacement, the "never sent anywhere" overclaim, the
missing console picture; VALIDATE remediation — the false stall, the `Live:false` fork.

**What the author got wrong in process**, named because the next reader should expect them: a commit
on a pipeline's exit code with a red package inside it; a plant's revert in the same command as an
uncommitted fix; a REVIEW exit presented before `a2dh validate` was 100 % green (the P10 run record
was stale; caught by the next run, and the exit sequence now runs it first); a sweep started, then
stopped at 183/380 when rulings changed the code — the right call, made 2.3 hours late.

**Upstream candidates for li-A2DH**, each with its worked example in `quality-observations.md`: the
dispatch brief as a template beside the axis files; the remediation-review loop (fresh reviewer per
safety fix, one finding at a time, iterate to LGTM) as a phase rule; INST-5 "a number's blind spot
beside it" as a report calibration; the phase-exit `a2dh validate` before red-team, which this
release proved by skipping once.

## 5. Follow-ups carried

Into **0.16.5, the dedicated quality pass** (`06_docs/follow-ups.md`): **F-156** the shell port to
`tools/`; **F-157** tier three never reaching a slot (the hyper-local station's pool); **F-158** the
P3(d) ruling and the pressed cut-back under a reading hazard; **F-159** the fault run's two authors
and the free-goroutine race (carry the fault on `Publish`); **F-160…F-162** the OPEN requirements
(FR-9.2, FR-3.6, FR-8.8); **F-137** the history comments the phrase list cannot see; the tenth oracle
round. Ruled and closed at exit: **F-163** (a `Run: 0` escalation shown by its reason; a decline does
not break a run) and **F-164** (D-159: a fall-through to synth releases the cut-over). Owed to the
record: the HUM LEAD's internal-name scrub result on the release checklist.

## 6. By the numbers

| | |
|---|---|
| Timeline | intake 2026-09-09 → shipped 2026-09-18 |
| Commits on the branch | 521, squash-merged as one |
| Diff against 0.15.0 | 727 files, +70,370 / −2,699 lines |
| Mutant corpus | 172 → 380; exit sweep 375 caught, 5 survived by design, 0 no evidence |
| Test files in the tree | 368 |
| Requirements | 60: 31 traced by ID, 13 by a named test, 4 OPEN with rows |
| Rulings cited in this release's record | 45 distinct D-1xx numbers, D-100 to D-159 |
| Blind reviewers | BUILD 7 (three rounds) + 9 oracle rounds; REVIEW 8 + 3 remediation over 6 passes; VALIDATE 2 + 1 over 2 passes |
| CI rounds on the PR | 3, each a finding |
| Release job | 37 m 42 s under a 90-minute cap; trend 13.0 → 13.6 → 13.7 → 16.1 → 37.7 |
| Published artifact | checksum OK, `0.16.0`, 0 build paths in 110,902 strings |

## Source documents

`01-objectives/project-brief.md` (the body of issue #10), `00-REQUIRED-READING.md`,
`07-readiness/gates.md`, `07-readiness/release-checklist.md`, `07-readiness/exposure-statement.md`,
`08-reports/build-report.md`, `review-report.md`, `red-team-review.md`, `validate-report.md`,
`red-team-validate.md`, `06_docs/gate-attack-list.md`, `06_docs/quality-observations.md`,
`06_docs/follow-ups.md`, `06_docs/handoff-0.16.0.md`.
