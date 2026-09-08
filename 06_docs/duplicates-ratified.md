# Ratified duplicates — metric D

**Metric D counts operations implemented more than once with no written, ratified reason. Target 0.**

**SCOPE: PRODUCTION CODE. Stated, because it was a default before it was a decision** (red team,
2026-09-08 — the gate was reporting "0 unexplained" while never having looked at a test file).

**HUM LEAD ruling 2026-09-08:** *"(a) is the metric that matters for success; (b) is an internal
quality metric that we should work down, but that's a secondary or tertiary metric since it pays us
in productivity later, but doesn't necessarily affect the end-user experience / quality of the
application."*

So there are two numbers, and only the first is metric D:

| | Scope | Today | Standing |
|---|---|---|---|
| **Metric D — success** | production code | **0 unexplained** | target 0, gated in `verify` and CI |
| Test duplication — internal | `-tests` | **17 unexplained** across 18 groups | worked down over time; NOT gated, NOT a release criterion |

`go run ./tools/dupes -tests` is the second number whenever someone wants it. Its largest group is
five copies of `TestDeclarationSetUnchanged`, which is per-package by necessity — a reminder that the
test number needs judgement before it needs work.

A row here is an exemption, and it works exactly like a P10 exemption: **the reason is RATIFIED by the
HUM LEAD, never self-issued.** That is the metric's own hardening in the project brief — otherwise D
could be satisfied by writing a justification for every duplicate.

The gate is `make dupes` (`tools/dupes`). A duplicate group not listed here fails it. A row without
the word `RATIFIED` does not count, on the same rule the P10 ledger gate enforces.

**Fingerprints are structural**: two functions share one when their bodies have the same AST shape
with identifiers and literals ignored, so a renamed copy fingerprints identically. A fingerprint
changes when either body changes — which is intended. An edited exemption has to be re-ratified.

| Fingerprint | Sites | Reason |
|---|---|---|
| `fbf77287aa96` | `Opts.Distance` · `Opts.TideHeight` | **RATIFIED by the HUM LEAD 2026-09-08.** Same skeleton — nil check, unit branch, two `Sprintf` — over two genuinely independent column contracts. They differ in unit system, precision, sentinel (`""` vs `"n/a"`) and field width, and the widths are pinned by UAT 61 and UAT 62 so a negative low never shifts the column. Collapsing would parameterise five things to save six lines and would couple two specifications that have no reason to move together. |

## Collapsed rather than ratified (2026-09-08)

The detector's first run found ten groups. Nine were collapsed; only the row above earned an
exemption.

| Was | Now |
|---|---|
| `app.spokenList` · `synth.joinAnd` | `plaintext.SpokenList` — byte-identical, both feeding SPOKEN output; a comma-policy change in one would have made two reports read differently |
| `severe.ByTab` · `tty.bucketSevere` | `platform/bucket.ByIndex` — `modes/` may not import `domains/`, so the owner had to be under `platform/` |
| `nhc.Fetch` · `usgs.Fetch` · **`nws.Fetch`** | `globalfeed.jsonFeed.fetch` — the third copy differed by one statement and the detector MISSED it |
| `cycleRelayLang` · `cycleRelayDwell` | `cycleIn` — the wrap arithmetic `((at+step)%n+n)%n` now exists once |
| `relayApplyCmd` · `relayLangApplyCmd` · `radiusApplyCmd` | `applyIfChanged` — three of the five `*ApplyCmd`; `cast` and `ui` return messages and were left alone |
| `memoCounts` · `modalMemoCounts` | embedded `memoStats.counts()` — a third memo now inherits it |
| `tick` · `vizTick` | `tickEvery` — TWO CLOCKS ARE CORRECT (300 ms shimmer, 50 ms bars); only how one is BUILT was collapsed |
| `modalAlertTone` · `alertModalBG` | `alertPair` — the fg and bg now cannot disagree about which alert is a warning |
| `Slot.CountsAgainstMax` · `Slot.textAtStandby` · **`Slot.String`** | `Slot.row()` — a third copy again, differing only in the field returned |

**Two of the nine were three-copy groups the detector reported as two.** A structural fingerprint
misses a near-duplicate that differs by one statement, so **the count is a floor, not a total.**
