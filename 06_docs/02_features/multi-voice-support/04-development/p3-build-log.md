# P3 build log — the maritime report and the tones (multi-voice-support, 0.14.0)

```
Batch:   P3 — the maritime report, the coastal-waters forecast, the five presets
Date:    2026-08-30
Branch:  feature/multi-voice-support
Gate:    07-readiness/gates.md §1
CLI:     a2dh v1.17.0 (commit 7960f0e)
```

**Agent-verifiable alone; still not the listener's judgement** (RN-4). A coastal location now reads the maritime
report between the forecast and the fire report, and each alert class sounds its own tone. Whether those tones
are *right* is the HUM LEAD's ear at P4 UAT — the sound of every preset remains their pass.

## Tasks

| Task | What landed | State |
|---|---|---|
| 3.1 | the marine words lifted to `platform/render`; the Details view unchanged | ✅ |
| 3.2 | `MarineReport`, `MarineSegments`, the spoken number forms | ✅ |
| 3.3 | the eleven `maritime-report/*.txt` script parts | ✅ |
| 3.4 | `Assembler.MarineFor` (sharing `harmonizeMarine`), `marineFor`, `Reports.Maritime`, the order | ✅ |
| 3.5 | `CoastalForecast` — **(B) only, MVS-D-38** | ✅ |
| 3.6 | dual-tone, 1050 Hz, soft chime, low sweep | ✅ |
| 3.7 | this gate | ✅ |

## The forecast is (B), and (A) is genuinely absent

MVS-D-38 ruled the nearest-zone geometry out of 0.14.0, so **no `MarineZoneFor`, no centroid memo, no fan-out
cap and no separate call budget were written.** `nearshoreBlock` takes the office synopsis plus the first
nearshore block, and `coastal.go`'s header says where the seam would go if (A) ever returns. Building (A)'s code
"ready for later" would have been the anti-pattern, not diligence.

Three periods (`SpokenPeriodsCap`, RAT-6/MVS-D-37), and the cut happens on the **raw** text before normalisation
rewrites the period tags. `.SYNOPSIS...` is excluded from the count: it looks exactly like a period tag and is
not one, and counting it would spend a third of the budget on something that is not a forecast.

## Two things the work found

**1. The screen's formatters are wrong for a voice.** Reusing `render.Opts` gave `" 3.0 ft"`, `"74ºF"` and
`" 1.4 kt"` — right in a column, unspeakable in a sentence ("eff tee", and a degree symbol is anyone's guess).
The report now has its own spoken forms (`heightWords`, `tempWords`, `windWords`, `knotWords`), and the
distinction is written down: **RS-11's "one owner" is about the WORDS** — is this sea rough, is the tide rising —
which do come from `platform/render`. A number's *register* is each surface's own business. The parity test
(`app`, which may import both) pins the words.

**2. A real concurrency defect, surfaced as a flaky test.** `TestRecastHandsOverMidSegmentAtTheSameSpot` failed
once in six runs. The cause was in P2's `renderAhead`: it read the hard generation *after* rendering, so a recast
landing between resolving the voice and reading the counter stamped a segment voiced by the **outgoing**
correspondent with the **incoming** generation. The writer would then see it as current, play it as rendered, and
the listener would hear the old voice with no hand-over immediately after saving. The window is microseconds
wide — it would have reached the field as a rare, unreproducible report. Fixed by reading the generation
*before* resolving.

## Gate

| Gate | Result |
|---|---|
| Unit + race | `go test ./... -race` green |
| Verify | **ALL GATES GREEN** |
| Harness | `a2dh validate` 17/17 |
| Alloc pins | green |
| P10 | **0 live, 0 unmatched** (111 dormant) · `07-readiness/p10-p3.json` · **no new exemptions** |
| Declsets | `app`, `modes/tty`, `platform/render`, `platform/snapshot` re-captured |
| PTY | `make pty-severe` green |
| Goldens | `cycle.golden` re-captured with the maritime section |

One live finding was **fixed rather than exempted** (a loop that should have been an append-spread).

## Deviations from the plan

1. **`MarineZoneFor` and `nws/marinezone.go` do not exist.** Task 3.5 described them as (A)'s implementation;
   MVS-D-38 ruled (A) out. Writing them anyway would have been speculative code for a ruling that went the other
   way.
2. **`UGCCodes` was not needed.** It existed to feed (A)'s zone list. `FilterUGC` (which already exists) handles
   the one case (B) needs — a listener whose own marine zone happens to appear in the product.
3. **The spoken number forms are new, not lifted.** See above: the plan assumed the screen's helpers would serve
   the voice, and they cannot.
4. **`MarineFor` shares `harmonizeMarine`** rather than a newly extracted `mergeMarine`: the publisher's helper
   was already the right body, so the second caller uses it as-is instead of extracting a third name for it.

## Notes carried to P4

1. **The tones now differ audibly by class**, so P4's UAT is the first that can judge them by ear. Every preset's
   sound is the HUM LEAD's pass (tones.md §1) and is expected to be revisited.
2. **The maritime report is the longest section in a cycle** (~10 sentences before the forecast). `maxMaritimePieces`
   bounds it at 12 segments; if UAT finds a coastal cycle too long, that constant and `SpokenPeriodsCap` are the
   two levers, in that order.
3. **The report reads the location's zone for tide times** (MVS-D-21), not the listener's. On a watchlist
   spanning coasts that is the correct but surprising behaviour, and worth stating in the UAT sheet.
