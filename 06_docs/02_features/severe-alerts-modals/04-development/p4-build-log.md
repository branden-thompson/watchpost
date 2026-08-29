# P4 build log — verification, hardening, docs

**Batch:** P4 (Tasks 4.1–4.5) · **Date:** 2026-08-28 · **Gate:** `make verify` ALL GATES GREEN · `make p10` 0 live,
0 unmatched · `make pty-severe` ok

| Task | Files | Notes |
|---|---|---|
| 4.1 | `modes/tty/alerts.go` | `render.Plain` on **every** [A] field (severity · urgency · certainty · area · instruction — the event and description already had it); the vestigial `i` parameter dropped from `alertRecordLines` |
| 4.2 | `app/severe_test.go` | `TestSevereEscapesNeverReachTheFrameEndToEnd`: a feed event with an OSC-52 clipboard write and an SGR in every field, through the deck → record → message → browse and detail frames; the ESC and BEL bytes never arrive (the OSC's text may survive as plain characters — that is the boundary's contract) |
| 4.3 | `scripts/quality/severe-modal.expect`, `Makefile` | `make pty-severe` builds and drives the real binary on a pty: `w` opens, `→` moves to Watches, `esc` closes, `ctrl+s` opens, `q` quits. Ran green here (`pty-severe: ok`) — no network needed, the window opens on its empty state |
| 4.4 | `README.md`, `CHANGELOG.md` | the `w` / `ctrl+s` key row; the 0.13.0 section (Added / Changed) |
| 4.5 | `07-readiness/gates.md` | the gate ledger with the commands that produced each result |

## Notes

- The 0.12.0 ticker narration test (`app/ticker_test.go`) carries the new tail; the burst closing line
  is a new sentence, not a change to an existing one.
- `make pty-severe` runs the user's real config; on a machine whose first run opens Setup, `w` is inert
  by design and the script reports `FAIL: w did not open the window` — run Setup once first.

## BUILD-exit addendum (red-team round 3)

4.1 was incomplete as first logged: the `[A]` module body, the compact line and both titles rendered
`Headline`/`Severity`/`Event` raw (R3-C-01) — now through the boundary with `TestAlertModuleAndCompactLineStripEscapes`.
4.3's journey now drills `enter → esc → esc → w` and the script is executable; the tmux/`ixon` note is in the
README. 4.5's ledger was rewritten with the gates it lacked (C-04). The COV test's denominator was reconciled
with the amended render list (C-05: 11/11 · 13/13 · 11/11).
