# Summary

The harness refused this file to the reviewer; the lead agent wrote it from the reviewer's final reply,
its words kept. Tool calls are approximate, and about 13 calls of shared reading are counted apart.

- T1, Forecast days Setting: 17 sites, one optional (README's config block) and one more if the row is
  ruled shared (`Router.Update`). Walkthrough 6 covers the Settings half step by step; the Overlays menu
  (`O`) needs five edits in `modes/tty/map_boxes.go` that no document routes to. ~22 calls.
- T2, UV INDEX head: 1 site (`modes/tty/map_temp.go:805`, `tempLegendRow`) and the CHANGELOG. ~6 calls.
- T3, DATA PULL today: 2 sites (`cardPulledShort` and its comment) and the CHANGELOG. No wrong routes.
  ~5 calls.
- T4, "yesterday at": 3 sites (`spokenWhen`, its comment, the `app/compose_takeover.go` header) and the
  CHANGELOG. No wrong routes. ~4 calls.

Helped most: `docs/extending.md`, `docs/where-things-happen.md`, `06_docs/ids.md`. Helped least:
`docs/architecture.md`, `06_docs/code-standards.md`, `map_boxes.go`'s header.

One grep over `modes/tty/*.go`, to find where four cycle functions live, broke the no-search rule;
`T1.md` says so.
