> **Reconstructed 2026-09-13**, from the commits, the tests and a re-run of the corpus.  See
> `p4-build-log.md` for why the record ran a month behind the code.

# P5 and P6 — the bed, and the settings that make it testable

## P5 — the bed

**Plan:** *"FR-4, D-24 — transitions wired, duck-per-medium, the paused main track."*

| Ruling | What landed |
|---|---|
| D-76/D-77 | the cut-over is the MONITOR'S, and the bed's fence is its own |
| D-78 | the bed's controls — **and the event that had never fired**.  `shift+←/→` steps the relay, so the bare arrows stay the card's |
| D-90 | **the operator's relay choice STICKS**, and the frequent publisher stops winning |
| D-117 | the bed **resolves its own relays**, and is refused when none stream |

**D-117 is where the bed became true rather than plausible.**  HUM LEAD, 2026-09-12: *"Relay bed
doesn't work … If none exist in that area we should probably tell the broadcaster there is no valid
relays for their area and disable the BED option so the Operator cannot choose something that will
broadcast dead air."*

The station's derived area is now cross-checked against the weatherradio.us / wxradio feeds, the
answer is **remembered rather than re-derived** (`refreshBedRelays` on a move, not on every ask), and
`bedAvailable()` gates the control.  A station with no relay in reach **cannot select the bed at all**,
which is the difference between an empty list and dead air.

**One correction is recorded here because it was wrong in the tree for a day:** `bedRelays`' own doc
comment still claimed the list was *"DERIVED ON EVERY ASK … from the embedded table"*, which D-117 had
made false.  A comment that describes a retired design is worse than none — it is a claim the next
reader has no reason to doubt.

## P6 — settings

**Plan:** *"FR-6, FR-8 including FR-8.9's national-scope answer, FR-9, FR-10.  BLOCKED on the cadence
measurement."*

| Ruling | What landed |
|---|---|
| D-92 | a settings row belongs to a SURFACE — the console's rows are not Observer's |
| D-115 | **the station's transmitter and service radius reach Settings (F-87)** |

**D-115 exists because the HUM LEAD could not test the thing without it.**  2026-09-12: *"Currently I
cannot change these settings [without] direct code changes — we need them exposed so I can also UAT
the re-derivation logic."*

The transmitter functions like Observer's Default location; the radius like the alerts-radius filter
**minus the "all alerts" option**, since a station with no service area is not a station.  Both sit
under the DATA group.

**And the borrowed location is now SAID OUT LOUD.**  A console with no transmitter of its own follows
Observer's default (D-72's `followsDefault`), which is correct and was invisible.  HUM LEAD:
*"Borrowing Observer's location when user hasn't set the Broadcaster Location is fine — as long as we
inform the user in some way."*  The hint reads `Broadcasting location - Enter City, ST or Zip`.

## What P6 dropped, and why that was the cheaper answer

**CORRESPONDENT was cut in favour of `C U R R E N T L Y` / `CONDITIONS  NOW`.**  HUM LEAD, 2026-09-12:
*"Useful information and doesn't require the correspondents wiring work."*

The column had no data behind it and no pipeline to get any.  Replacing it with conditions the app
already fetches turned an empty column into a decision-support one at the cost of a struct field —
and the running order now shows the operator what the weather is doing at each beat.
