---
title: "0.16.0 P3 — UAT: the audio merge, shared with nothing"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "READY TO RUN.  This UAT covers P3 AND NOTHING ELSE, deliberately."
---

# P3's UAT

**It is shared with nothing, on the 0.14.0 precedent for the takeover swap:** *"it replaces the working
takeover path, so it carries its own red team, and it must not share a UAT with anything else or a
regression cannot be attributed."*  **P3 replaces how every ordinary broadcast reaches the speaker.**
If it is run alongside anything else and something sounds wrong, there is no way to say which change did
it.

**Run it on a build of this branch, with a real relay in reach and a real one out of reach.**

```sh
make build
WATCHPOST_DEBUG_RADIO=1 ./dist/watchpost
```

**~~`WATCHPOST_MAINTRACK=dark`~~ IS GONE.**  The staged switch existed only between the producer landing
and the flip, and it was deleted with the direct path — a switch that outlived the merge would be a
second way for the station to behave, which is the thing being removed.  What was the dark run's
instrument stayed: `needs-read` lines are still written to the same log, and they are still how a read
is paired with the decision that caused it.

## What is actually being tested

**Two owners of the audio device became one.**  Everything below is a way of asking whether the one
that remains behaves like the one that left.

| # | Do this | It passes if | It fails if |
|---|---|---|---|
| **1** | Tune a location with **no NWR relay in reach**.  Let the report run to its sign-off | The report plays exactly as it did in 0.15.0: the same words, the same voices, the marquee moving line by line paced to the speech, the player row reading `Watchpost Synth (<voice>)` | Any of those is missing, static, or differently voiced |
| **2** | Watch the **detail line** as it starts | It says why: *"… — not relayed"*, or *"… — no NWR relay in reach"*, with the nearest live station when there is one | It is blank, or carries the reason from the PREVIOUS read |
| **3** | Let the report **end on its own** | The rotation moves to the next location by itself | It stops there.  **This is the one to watch**: the report's end is what advances the rotation, and it now travels a different route |
| **4** | Set **repeat-one** and let a report end | It reads again — and the observation is **re-fetched**, not a replay.  Check a value that changes (the time in the sign-off) | The second read is identical to the first, word for word |
| **5** | While a report is reading, let a **severe alert arrive** (or force one) | The report **PAUSES**, the alert reads in full, and the report **resumes from where it stopped** | The report keeps playing under the alert, or restarts, or never comes back |
| **6** | While a **relay** is playing, let an alert arrive | The relay **DIPS** and plays on underneath.  This is the regression half: it must be unchanged | It pauses |
| **7** | Press **`[M]`** while a report is reading | **The broadcast keeps playing.**  `[M]` means "do not read me hazards", and it has never stopped the radio | The report stops.  **This is the merge turning the mute key into a stop button**, and it is the failure mode P3(d) most nearly shipped |
| **8** | Press `[M]`, then let an alert arrive | The alert is **held**, not read — and it is offered again when unmuted | It reads anyway, or is consumed silently and never sounded |
| **9** | Press **stop** mid-report | Everything goes quiet at once, and nothing starts itself again | Audio continues, or the rotation restarts on its own a few minutes later |
| **10** | Press **stop**, then start again | It reads again from the current location | It stays silent |
| **11** | **Tune to a live relay, then kill the network** so the relay fails while playing | The station falls back to a synthesised read, and the detail line says *"relay unavailable — …"* | It goes silent, or reads without saying why, or reads TWICE (two voices at once is the defect this whole batch exists to remove) |
| **12** | Tune to a relay that **connects and then goes silent** | Same as 11, with *"the relay was silent"* | Same failures |
| **13** | Remove a location from the watchlist **while its report is queued** | Nothing plays for it and the station moves on | It reads a location that is no longer watched, or the station stops |
| **14** | Run for **thirty minutes** through a full rotation | Every location is read once per turn, in order, and none is read twice in a row | A location repeats, or one is skipped, or the rotation stalls |

## The log, and how to read it

Three kinds of line share `~/Library/Caches/watchpost/debug/radio.log`, timestamped:

| Line | What it means |
|---|---|
| `needs-read fresh=<bool> ref=<lat,lon> why=<reason>` | the deck noticed nothing is carrying this location.  **`fresh=false` is a decision, not an error** — the need arrived after the listener stopped or moved on, and was dropped deliberately |
| `<state> mount=… title=…` | the engine's own transitions |
| `segment key=… spoken=…` | which segment the read reached, and for how long |
| `schedule:declined:…` | a card the schedule refused, with the reason in words |

**Pair them.**  Every `needs-read … fresh=true` should be followed by segments for that location.  A
`needs-read` with no segments after it is case 3 or 13; segments with no `needs-read` in front of them
would be an audio path nobody asked for, which is the defect.

## What this UAT does NOT cover

- **The operator console.**  P4 and P4.5 are the surface; this is the audio underneath it.
- **Anything about the alert rail's own content** — the burst ordering, the divert count, the ladder.
  Those shipped in 0.14.0 and 0.15.0 and are not touched here; cases 5, 6 and 8 test only that the
  merge did not disturb them.
- **Cold-start voice installation.**  Case 1 on a machine with no voice will download one; that is the
  first-run install ruling and is not a P3 behaviour.
