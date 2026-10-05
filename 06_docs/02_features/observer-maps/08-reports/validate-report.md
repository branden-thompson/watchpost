---
title: "0.18.0 — VALIDATE report"
date: 2026-10-05
phase: VALIDATE
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-05 (D-282); SHIP proceeds, the tag cleared"
---

# VALIDATE report — 0.18.0

VALIDATE runs every tier the release claims, on the code REVIEW approved (D-279), and closes what REVIEW
left to it: M5 by its protocol, M6's thresholds (D-251), the release notes against what ships.

## The tiers

| Tier | Result |
|---|---|
| The full gate, local (`make verify`) | green on the commit this report is made with; every required gate (`06_docs/required-gates.txt`) |
| The hosted full gate, macOS and ubuntu | green on batch 138 with its CI fixes (run 37348280922); this report's commit is pushed and its run read before SHIP |
| The scripted-PTY journey on the real binary (D-271) | green on both platforms in the hosted gate: `g`, `A`, a pan, a zoom, a region, `O`, a clean `q`; no network, a sandbox on macOS |
| M5 - time to picture (D-46, D-251, D-280) | **met**, in process: 20 cold opens at 149x38 on recorded responses, p50 40.0 ms, p90 67.3 ms, max 244.9 ms, against 3.5 s |
| M6 - loop smoothness (D-53, D-281) | **met** against the HUM LEAD's thresholds: tick lateness p99 1.2 ms (≤ 100 ms), intervals the 1 s step bar the library's 2 s last-frame hold, key to frame p90 5.4 ms (≤ 100 ms) |
| M1, M1b | UAT stands as their evidence (D-252); the answer-key test reads the description's words exactly, every scenario through the real parts (REVIEW QA-2, QA-7) |
| M2 - M4 | their instruments green in the gate (`TestNoFrameIsWiderThanTheRegion`; `TestTheNewestFramesAgeIsAlwaysSaid` on the screen; `TestAPartialAreaIsDrawnAsFoundAndSaid`) |
| P10 | clean: `make p10`, 0 live, 0 unmatched, 0 unratified |
| The release notes | the CHANGELOG's 0.18.0 entry checked against batches 137 and 138: the loading words, the Overlays menu as text, the history's datasets, MAP STATUS's disclosure, Clear map data, the hardened clients, the debug server; the cuts and D-250's known gap |

Measurements and their commands: `07-readiness/m5-m6-measurement.md`.

## Blind spots, beside the numbers

**M5's number is not a listener's first open.** It measures the drawing and the app's feed on recorded
answers, in process. The network, the first-ever zone fetch, the process and the terminal are outside it.

- The built binary cannot complete a frame offline: no recorded tiles for the view, and go-tuiMaps fetches its own over https.
- The live first-ever cold open measured 7.0-9.4 s (n = 5, W14).

**M6 is one machine's**, an Apple M5 Pro under the load of the session's other work. Its loop numbers come from an in-process run with stand-in feeds and one snapshot a second; the binary's key-to-frame alone is from the built program.

**The journey's Linux path** runs without the macOS sandbox: the local proxy is its only guard against the network.

## What SHIP has left

- `release/0.18.0` squash-merged to `main` through a PR, closing #22, on a full green run.
- The CHANGELOG's 0.18.0 entry dated, the tag `v0.18.0` on the merge.
- Local `main` reset to `origin/main` (D-259).

## The ruling asked

Approve VALIDATE exit, so 0.18.0 proceeds to SHIP.
