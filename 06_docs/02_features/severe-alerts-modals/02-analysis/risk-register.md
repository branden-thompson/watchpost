# Risk register — severe-alerts-modals (0.13.0), status at REVIEW (2026-08-29)

The objectives' §6 table is the register of record; this is its status at the REVIEW exit, one line per risk,
with the evidence that holds it. RESOLVED = the mitigation shipped and is pinned; HELD = mitigated by design and
pinned; OPEN = carried with an owner.

| Risk | Status | Evidence |
|---|---|---|
| RS-1 Widening `Event` ripples | RESOLVED | seen-store ids only; bounds per field (`clampField`, `clampID`); memory gauged in `[S]` |
| RS-2 Two colour schemes collide | RESOLVED | modal tokens separate; independence guard with planted control |
| RS-3 Scope creep | RESOLVED | render list frozen (FR-5); non-goals held (R5-A) |
| RS-4 `ctrl+s` eaten outside the app | HELD | `w` primary; `make pty-severe`; the ixon limitation documented (README) |
| RS-5 Per-tick rebuild × six tables | RESOLVED | modal memo; budgets pinned (`bench_test.go`) |
| RS-6 Source mismatch / dishonest counts | RESOLVED | union + normalised id + "showing N of M" (pinned at REVIEW) |
| RS-7 Duplicate rows from id forms | RESOLVED | `NormalizeID`; the 200-rune id bound on both paths (R5-B-05) |
| RS-8 Re-parse churn | HELD | parse memo keyed on the httpx cache fact; `BenchmarkSevereDeckTrigger`; the 1-h soak owed at VALIDATE |
| RS-9 Hidden second modal state | RESOLVED | `severeDetail` reset by `close()` (R5-A-04); exclusivity test |
| RS-10 Location-path superseded alerts shown twice | HELD | NFR-12 guard in the index; the `[A]` path follow-up (R5-A-08) |
| RS-11 Fixed tints on a light terminal | SUPERSEDED / OPEN | blend 1.0 by HUM LEAD; light terminals fail AA broadly (R5-C-03 — ruling) |
| RS-12 Accessibility regression | HELD | `--ascii` forms for the new marks; class in text with colour off; AA gates over every theme for the painted pairs; the remaining dark pairs listed (R5-C-04) |
| RS-13 Empty / low-signal modal | RESOLVED | empty state per tab (FR-14) |
| RS-14 Narrow-terminal truncation | RESOLVED | 130-col ceiling; the column ladder; geometry sweeps 20–240 × 20–100 clean (R5-C) |
| RS-15 Metric baselines wrong | RESOLVED | corrected at DISCOVER |
| RS-16 Warning suppression via `references` | HELD | same sender + product + newer (NFR-12), both paths pinned |
| RS-17 Feed prose reaching the terminal | RESOLVED | every seam through `plaintext` (R5-C-05); the escape corpus swept clean (R5-C) |
| NEW — a shared snapshot between the deck and the tty | RESOLVED | the deck's own copy (R5-B-01), race probe lifted as a test |
| NEW — the radio path changed (NFR-7 superseded) | OPEN | R6 relay + audio smoke BLOCKING at VALIDATE, both platforms |
