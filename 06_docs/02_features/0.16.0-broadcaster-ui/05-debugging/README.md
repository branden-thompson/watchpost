---
title: "0.16.0 — debugging"
status: "Populated at BUILD exit, 2026-09-15.  SEV-0 requires ALL folders; this one was absent while its material sat elsewhere."
---

# Debugging — 0.16.0 Broadcaster UI

**This folder was missing and the material was not.**  Found by red team at BUILD exit: SEV-0 requires
every folder, `05-debugging/` had none, and the debugging record for this release had been filed under
`04-development/` and in the project-wide ledgers instead.  Rather than move documents that other
files already cite by path, this is the index to where each piece lives.

## The investigations, and where they are written up

| What was investigated | Where it is | What it concluded |
| --- | --- | --- |
| **The P3 flip** — the audio merge that was reverted the same day | `04-development/p3-flip-postmortem.md` (201 lines) | Four blockers, every one already answered in 0.14.0's documents.  *"The record was complete; it was not read."* |
| **A gate that failed once and cannot be explained** | `06_docs/follow-ups.md` F-101 | Open.  Unreproduced; the mechanism was never identified. |
| **A `t.TempDir()` cleanup that failed once** | `06_docs/follow-ups.md` F-102 | Open, re-verified 2026-09-15: 245 clean runs, no sighting, mechanism still unknown. |
| **"Data requests feel a little slow"** | `06_docs/perf-measurement.md` | The card build is not the cause — 449 ms cold, 1–2 ms warm, not parallelisable.  Three candidates carried to 0.16.5. |
| **The location lookup's real cost** | `06_docs/perf-measurement.md` | The offline index is 1–15 µs; the geocoder is ~200 ms warm, 939 ms cold.  Rainbow, CA is in neither embedded table — the network is the only authority for it. |
| **Corpus sweep: two survivors** | `07-readiness/mutant-verdicts.log`, disposition at `07-readiness/gates.md` | Both equivalent mutants, HUM-LEAD-kept rather than retired. |
| **Every defect found in UAT** (D-121 … D-138) | `04-development/p7-build-log.md` | Each carries its reproduction, its cause and what now measures it. |
| **BUILD-exit red team** (2 blind agents, A2DH axes) | `08-reports/red-team-build.md` | 2 critical, 7 important, 12 minor.  Dispositions in the build report. |

## The debugging lessons that outlived their bug

These are in `06_docs/quality-observations.md` because they are about METHOD rather than about this
release, and the A2DH skill extraction reads that file:

- **An assertion that is true for the wrong reason** — five instances in this release, three of them
  found by red team after I had already written the shape up.
- **A gate that is only clean because a different gate ran.**
- **A dry run that mutates** (`make -n` executes recipes containing `$(MAKE)`).
- **INVALID is not SURVIVED** — a mutation that does not compile is not evidence.
