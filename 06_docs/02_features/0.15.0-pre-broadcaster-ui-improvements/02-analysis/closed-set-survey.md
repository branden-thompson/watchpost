---
title: "0.15.0 — Closed-set survey (FR-2.1a)"
date: 2026-09-07
phase: BUILD · B1 opening task
sev: SEV-0
status: "Complete. 4 arms need work, not 49."
---

# Closed-set survey

**FR-2.1a: size the population before generalising a mechanism over it.**  This exists because the
0.14.2 lane guard was generalised from too few instances and shipped asserting four of seven lanes.

## The exclusion rule, stated before counting

**Non-test · non-`third_party/` · non-`tools/`.**  Roots: `app`, `modes`, `domains`, `platform`,
`cmd`, `pkg`.

The rule is not bookkeeping.  Without it the arm count is **50**; with it, **49** — and including the
vendored go-studs patch stack would move it by ~30%.  PLAN revision 1 asserted "27 sets and 50
default arms" with no boundary, which is a number nobody can reproduce.

## Population

**27 closed sets · 49 `default:` arms across 34 files.**

## Classification

| Bucket | Count | Meaning |
|---|---|---|
| **1 — closed-set consumer, needs a check** | **4** | Switches on a closed-set value, defaults, and the default is not a stated decision |
| **2 — stated decision, no check owed** | **3** | Switches on a closed-set value, and the default's rationale is written where the decision lives |
| **3 — open set, a default is correct** | **42** | 5 non-blocking `select`; 25 bare boolean `switch {}`; 12 switches over strings from providers, keyboard input, `reflect.Kind`, or type switches |

**The headline: the population that FR-2 must treat is FOUR, not forty-nine.**  Thirty-eight of the
forty-nine are `switch {}` on boolean conditions or `select` for non-blocking reads — they have no
member set to be complete over.  The plan sized B1 **L** on a count of fifty; the work is much
smaller, and the survey is what shows it.

## Bucket 1 — the four

| Site | Switch | Default | Why it needs a check |
|---|---|---|---|
| `app/inject_debug.go:133` | `switch key` (scenario) | injects `"Tornado Warning"` | **Already known — this is FR-4.1.** `"emergency"` and `default` are byte-identical, so the scenario named for the emergency path does not exercise it |
| `modes/tty/view.go:134` | `switch d.modal` | `raw = d.helpLines(...)` | **A modal with no case silently renders HELP text.**  Broadcaster adds windows; each new one renders help until someone notices |
| `domains/globalfeed/event.go:73` | `switch e.Class` | `return "declared"` | The spoken verb.  `ClassSevereWx` reaches it correctly today, but a fourth class would be "declared" by accident rather than by decision |
| `domains/radio/cast/resolve.go:70` | `switch l` (`Link`) | `return "none"` | A `Link` with no case reads as unlinked |

## Bucket 2 — stated decisions, and one is a model

| Site | Why it is already right |
|---|---|
| `app/ticker.go:595` | The comment **is** the ruling: *"EXPLICIT, with no default.  A row whose tab has no lane must be dropped, not guessed at: the old form defaulted to Advisory, so a new lane-eligible tab would have been labelled 'Advisory' on the band and nothing would have said otherwise."*  It names the trade, the prior defect, and the residual risk.  **This is what FR-2.3 asks for, already in the tree** |
| `domains/weather/nws/provider.go:88` | Returns `fmt.Errorf("nws does not serve fetch kind %d", req.Kind)` — the unknown member is named in the error rather than absorbed |
| `modes/tty/setup.go:412` | Falls through to a structured `kind`-based switch rather than to a value; the fallback is itself typed |

## What this changes in the plan

1. **B1 shrinks.**  Four arms, one of which (`inject_debug.go`) is already scheduled as FR-4.1 in B3.
   The survey's own cost was the reading pass, and it is now spent.
2. **`view.go:134` is a new finding** and belongs to the Broadcaster-coupling argument: it is the one
   bucket-1 arm whose failure mode grows with every window 0.16.0 adds.
3. **FR-2.3 has a worked example in-tree.**  `ticker.go:595` is the standard to write the other three
   against — a comment that names the trade, the prior defect, and what is still at risk.
4. **Bucket 3's size is the real result.**  Thirty-eight arms have no member set at all.  A mechanism
   generalised over "all default arms" would have been aimed at a population that mostly does not
   exist — the same error, one level up, as generalising the lane guard from one instance.

## Reproducing this

```
non-test, non-third_party/, non-tools/ under app modes domains platform cmd pkg
  type X int|uint8|string with `X = iota`        → 27
  ^\t+default:                                    → 49
  of those, preceded by `select`                  → 5
  of those, preceded by bare `switch {`           → 25
```
