---
title: "0.15.0 — Pre-Broadcaster UI Improvements — VALIDATE REPORT"
date: 2026-09-09
phase: VALIDATE
level: LEVEL-1
sev: SEV-0
authority: COLLABORATE
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "Awaiting HUM LEAD approval to exit VALIDATE and enter SHIP"
---

# 0.15.0 — Pre-Broadcaster UI Improvements — VALIDATE REPORT

## Bottom Line Up Front

**No blocking findings. The first clean red-team round of the release, and the install was verified
end to end for the first time.**

`scripts/install.sh` was run against a server on the same code path as a GitHub download: the asset
names it builds reconcile exactly with what `release-matrix` writes and `release.yml` publishes, the
checksum verifies, the tamper control fires, and the installed binary starts and reports its version.
**Nothing in this release had exercised that path**, and a mismatch would have shipped a broken
one-line install behind the README's first command.

**Recommendation: exit VALIDATE.** `make verify` ALL GATES GREEN at `8b9db31`; HUM LEAD UAT passed on
all four exercised surfaces.

## 1. Exit gates

| Gate | Evidence |
|---|---|
| `deployment_verified` | `make build` 6.2 s; `install.sh` end to end; asset names reconciled across three files; checksum + tamper control both exercised |
| `no_regressions` | `make verify` green; two blind red-team agents; HUM LEAD UAT |
| `readme_content_audited` | Full content audit — three claims corrected (§3) |
| `stakeholder_acceptance` | HUM LEAD UAT 2026-09-09 (§4) |
| `critical_analysis_complete` | Two agents on the A2DH axes, no blockers (§2) |
| `report_published` | This document |

## 2. Critical analysis

**Agent 1 — docs-quality, hunting one shape: claims of protections the code does not implement.**
That shape had bitten twice this release (the voice cap; D-9's "shell does not decide"), so it was
swept deliberately rather than waited for. **Twelve claims chased into their implementing lines** —
the voice mutex and its sha256, the redirect hop cap and same-origin rule, the FIRMS key's single
host, the emergency-order exemption, `RecentCap`, the credits width. All backed, most also pinned by a
test. **No findings**, with the table to audit the sweep.

**Agent 2 — project-hygiene + safety-critical.** No blocking findings. Four low, three fixed:

- `setup.go`'s comment claimed `setupFinishCmd` "owns the radius". It does not — `radiusApplyCmd`
  does, listed on the next line — and twenty lines below sits the warning that a drift between them
  makes the model and the file disagree. **A comment misattributing ownership is the seed of the
  drift it warns about.**
- A digit typed into a **full** radius buffer flipped the radio to "Within" without changing the
  number — a press that appears to choose and does not. Introduced by this release.
- Running without a TTY printed `bubbletea: error opening TTY: … /dev/tty: device not configured` — a
  dependency the listener did not choose and no step they can take. It now names the cause and two
  ways forward, one of which was verified to exist before being suggested.

## 3. README content audit

Reading the document **whole** found three things that six targeted greps this session did not. Each
grep found what it went looking for; none asked whether everything there still describes the app.

1. **A safety cap that does not exist** — *"fetches at most two voices in the background per session"*.
   Installs serialise on a mutex; there is no count. Corrected to what the code does; whether the cap
   should exist is F-60, because what happens when a read needs a voice beyond it is listener-facing.
2. **The severe categories were wrong twice and disagreed with each other** — prose said six, a
   caption said seven, the truth is **eight**. The prose omitted **Emergency Orders**, the category
   that carries evacuation orders, which FR-10 had established in this same release is the most
   safety-relevant thing that window shows.
3. **An overstated accessibility claim** — *"every colour pair in every theme is checked"*. Eleven
   tokens are excused with a stated mechanism (F-57).

## 4. UAT

HUM LEAD, 2026-09-09, on `v0.14.2-113-g8b9db31+debug`:

| Surface | Result |
|---|---|
| Fire read — quiet area | good |
| Fire read — one incident, no hotspot | good |
| Fire read — busy location | good |
| Settings — changes preserved on `esc` | good |
| Location Details — tables | good |
| Radio | good |

The three fire variants each exercise a different branch: quiet confirms both nothings are spoken per
ring; one-incident confirms the singular AND that one empty half does not suppress the other; busy
confirms the count line and the list still read together after the header was removed.

## 5. Carried into SHIP

| Item | Why it is not a blocker |
|---|---|
| **F-61** | The fire read has no age bound, so an hour-old incident list is spoken in the present tense. Every report head disclaims DELAY, and that covers stale data — it does not cover a positive assertion of ABSENCE, which is why the never-answered case WAS a blocker and this is not. The bound is a listener-facing ruling and likely differs per feed |
| **F-62** | The radius field renders "Within [0] mi" while 0 means All locations. Errs toward MORE alerts, never fewer |
| **F-60** | Whether a per-session voice-install cap should exist |
| **F-58** | Lookup stalls on a freshly seeded install; `make journey` red at 26/28 by ruling |
| **F-55** | Below the minimum terminal size the frame overflows silently; the contract lands with Broadcaster UI |
| Demo location | 67 fixture files; recorded API responses. The shipped binary is clean |

## 6. The release's defect curve

| Round | Reviewer | Findings | Blockers |
|---|---|---|---|
| BUILD exit 1 | the author | 5 | 0 |
| BUILD exit 2 | 3 blind agents | ~20 | **3** |
| BUILD exit 3 | 3 blind agents, scoped | 15 | 0 (**2 regressions, both the author's**) |
| REVIEW | 2 blind agents | 6 | **1** |
| VALIDATE | 2 blind agents | 4 | **0** |

The curve inverted: rounds that found pre-existing defects gave way to a round that found only the
author's own remediation, then to a clean sweep. **That is only legible because every round was
counted**, and it is the strongest argument in this release for independent review as a phase gate
rather than a courtesy.

## VALIDATE exit — recommendation

**Exit VALIDATE, enter SHIP.** Deployment verified, no regressions, README audited, stakeholder
accepted, critical analysis complete, and every carried item named with the reason it is carried.
