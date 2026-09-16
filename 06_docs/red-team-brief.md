# The red-team dispatch brief

**A blind reviewer finds what the brief asks about, and nothing else.** This page is the brief.
Copy the template below, fill the five bracketed fields, and send it verbatim. Do not compose one
from memory.

## Why this page exists

li-A2DH's red-team skill says every dispatch prompt "MUST satisfy the Subagent Prompt Discipline"
and ships no template that does. Composing one from memory drops whatever is not top of mind, and
what is top of mind is whatever was just worked on — which is the one area a blind reviewer adds
least to.

**This release is the worked example.** Two agents were dispatched with an ad-libbed brief and
neither reported a single `AP-HIST-01`, an anti-pattern the code-quality axis names explicitly.
There were 194 of them in the tree. The reviewers were not wrong; they were not asked.

A second reason it is here rather than only upstream: **a brief in the repository does not need the
harness.** A contributor with no li-A2DH install can still run the review the project expects, and
the axis content is reproduced below rather than linked for that reason.

## The template

> **Scope.** Review `[the surface: a branch, a diff range, a directory, a list of files]`. Read
> whatever else you need to judge it; report only on the scope.
>
> **What you are.** `[the axis perspective, verbatim from the axis section below]`
>
> **What you do not know.** You have not seen this work before and that is deliberate. Do not ask
> for context — if something is unclear, that is itself a finding. `[Any context the agent genuinely
> cannot derive: the domain, the release's purpose, a ruling that settles a question they would
> otherwise raise. Keep it to what they cannot read off the tree.]`
>
> **What NOT to tell you.** `[Nothing, unless a current disposition would anchor them. If this
> review exists to check a conclusion, say the question and never the answer.]`
>
> **Work read-only.** Do not edit the tree. `[If the reviewer needs to run something, name a
> scratchpad path.]` A sweep or a gate run may be in flight; `make tree-free` says whether it is.
>
> **Answer every question in the axis below, in order, by name.** A question you found nothing for
> gets one line saying so. An axis question you silently skip is indistinguishable from an axis
> question that passed, and the two are not the same result.
>
> **Every finding cites `file:line`.** No finding without one. State the finding directly, then:
> Severity (Critical / Important / Minor), Evidence (`file:line`), Simplify/Delete? (Y/N),
> recommended Action.
>
> **Do not pad.** If an area is clean, omit it — but say so against the question that covers it
> (see above); those are different obligations.
>
> **Give an opinion.** End with: ship / do not ship, and the one finding you would fix first.
>
> **Budget: `[n]` words.**

## The axes

Reproduced from li-A2DH `02_skills/critical-analysis/red-team/axes/`. Four axes, always on.
Dispatch one agent per axis, or run them in order in one context.

### Code Quality

**Perspective.** Distinguished Engineer. You optimise for the next reader and the next change, and
you are hostile to code that exists without justification.

1. **Understandability** — can a new reader follow this without the author present?
2. **Maintainability** — what breaks the next time requirements change here?
3. **Complexity** — is there a simpler construction that meets the same requirement?
4. **Necessity** — could this function / branch / abstraction / parameter be DELETED and still meet
   requirements? **This is the primary lens** — name every removable element explicitly.
5. **Name semanticism** — do names mean what they say, or mislead? (`SN-01`)
6. **Code documentation** — are non-obvious decisions explained where they live, not in a distant
   doc (`SN-02`)? Do comments describe current state, not history (`AP-HIST-01`)?
7. **Testability** — can this be tested without elaborate scaffolding? Is it actually tested?
8. **P10 conformance** — does changed Go hold the Power-of-Ten rules (`P10-*`)? Cite the IDs.
9. **Evidence soundness (verifiers fail closed)** — does any test or script report success on a
   branch where a claimed proof point was skipped, not located, or left unconfirmed? A verifier that
   cannot verify must FAIL, not pass.

**Questions 5, 6 and 9 are mechanically checkable here.** `make lint-authoring` decides `AP-HIST-01`,
`AP-DEAD-01` and `SN-02` over the whole tree, and `06_docs/code-standards.md` writes the rules out.
A reviewer who reports one of those is reporting a lint failure, which means the lint is off — say
so. **The rest of each rule is still theirs**: the tool cannot see a comment that is merely WRONG,
which is the larger class.

### Project Hygiene

**Perspective.** Staff engineer inheriting the repository on Monday with no handover.

1. Is anything in the tree that should not be — a binary, a secret, a scratch file, a dead branch?
2. Does the build work from a clean clone, with the documented commands?
3. Are the gates the project claims to run actually run, on every path that claims to run them?
4. Is anything named as owed — a TODO, a follow-up row, a ratified exemption — that has quietly
   stopped being true?
5. Does the record say what happened, or what somebody intended to happen?

### Docs Quality

**Perspective.** The reader the document was written for, who has not read the code.

1. Does each document answer the question its title asks?
2. Does anything in it contradict the code, or another document?
3. Is a claim made that nothing verifies, in a document that reads as verified?
4. Is the audience order right — designers, then PMs, then engineers?
5. Is a number published without its blind spots beside it (INST-5)?

### Business Quality

**Perspective.** The person who has to defend this release to someone who did not build it.

1. Does the work meet the requirement that was written down, or the one that was remembered?
2. What does the user lose if this ships as it stands?
3. What was cut, and is the cut recorded where the next reader will find it?
4. Is there a cheaper way to get the same outcome for the user?

## Standing rules a reviewer should be told

- **A finding is not excused by predating the change.** The question is whether the repository
  contains the defect, never who wrote it or when. A scope narrowed after seeing a count, which
  reduces the count, is a defect until proven otherwise.
- **A reason is RATIFIED, never self-issued.** An exemption a reviewer thinks is warranted is a
  recommendation to the HUM LEAD, not a disposition.
- **UX, read-order and layout are the HUM LEAD's.** Present the fork with evidence; do not rule.

## Upstream

This page is an upstream candidate for li-A2DH — `axes/dispatch-brief.md` beside the axis files, so
the skill's own "MUST satisfy the Subagent Prompt Discipline" has something to point at. Recorded in
`06_docs/quality-observations.md`.
