---
title: "p3 — the zone store"
date: 2026-09-21
phase: BUILD
sev: SEV-0
release: 0.17.0
---

# p3 — the zone store

**This phase is much smaller than it was planned to be**, because two of its
rulings turned out to be already built.

## Two rulings that needed no code

MG-11 said honour the service's own freshness. MG-13 said a stale shape is
refetched when the network allows and never discarded. Both are already in
`platform/httpx`:

- `serverTTL` reads `Cache-Control: max-age`, falling back to `Expires`
  (`platform/httpx/cache.go:614-637`);
- an expired entry that carries validators lives a further 24 hours so a
  conditional request can renew it (`cache.go:104`, `staleGrace`);
- the disk tier means a relaunch is warm.

So this package caches nothing of its own beyond the **parsed** shapes. Parsing
twelve thousand positions again for every frame would be the waste; fetching
them again would not, because the client already does not.

**The lesson is the red-team's own question** - *what can be deleted* - arriving
one phase later than the review. A store was planned with a disk cache, a
freshness rule and a revalidation path, and the right amount of that to write
was none.

## What was built

| Task | The test written first | Note |
|---|---|---|
| **3.1** a zone, fetched once, kept whole | its name is kept beside its shape; asked for twice it is fetched once; a zone that does not exist is an error, not a panic | The name rides along because it is in the same answer and a description will want it (MG-9) |
| **3.3** a set of ids at once | the same id named twice costs one request; the answer says which ids were got **and which were not** | MG-10 as re-ruled: this layer has no view to decide a gap against, so it reports and the caller decides |
| **3.4** the watched places' zones, seeded | seeding holds them, and a zone already seeded is not fetched again; **a cold start with no network is not an error** | On demand alone is coldest during severe weather, which is the one time the map matters (MG-7) |

## Checked against the service, not only against a fake

The tests above run against a local server. The store was also run against
`api.weather.gov` once, by hand:

| id | name | rings | positions |
|---|---|---|---|
| TXZ119 | Dallas | 1 | 80 |
| INZ018 | Allen | 1 | 74 |
| **AKZ320** | **Glacier Bay** | **32** | **12,004** |

Three zones in **944 ms**, including the 1.4 MB one, with a fourth id reported
missing rather than failing the batch. The counts match what an independent
tool made of the same documents in DISCOVER.

## Gates

`gofmt`, `go vet`, `lint-imports`, `alloc-budget` and the full test tree pass.
The new package carries no declaration set of its own.
