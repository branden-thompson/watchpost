# Summary

Tool calls are approximate. About 8 calls of shared reading (the README's Start here, architecture.md,
where-things-happen.md, extending.md) are counted separately.

- T1, Forecast days Setting: 17 sites (two "no change" checks among them). W6 covers the Settings half
  well. The Overlays-window half and the open-from-Setting rule had no walkthrough. ~17 calls.
- T2, UV INDEX head: 1 site, reached by a guess. No doc names the map's legend row. ~3 calls.
- T3, DATA PULL drops today's date: 2 sites changed (function and its comment), plus 2 checked as no
  change and a CHANGELOG line. W8 routed straight there. ~8 calls.
- T4, "yesterday at": 3 sites (function, its comment, the Composer header's caveat) plus CHANGELOG. W7
  step 2 routed there, two calls up from the site. ~6 calls.
- IDs across all tasks: ~3 calls.

Helped most: `docs/extending.md` (W6 and W8 especially), `docs/where-things-happen.md` (the event
table), and the code's own comments (the `fcLow` field, `spokenWhen`, `cardPulledShort`).

Helped least: `docs/architecture.md` (shape only, no sites for these tasks), `06_docs/ids.md` (most
bare D-n IDs have three or four release rows, and S-7, SAM-D-26, PD-3, W10.10 and U2-18 are missing),
and where-things-happen.md's "A time is written or spoken" row, which sent T3 and T4 to `clock.go`
when neither site is there.
