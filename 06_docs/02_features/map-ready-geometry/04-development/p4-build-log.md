---
title: "p4 — the seam"
date: 2026-09-21
phase: BUILD
sev: SEV-0
release: 0.17.0
---

# p4 — the seam

| Task | The test written first | Note |
|---|---|---|
| **4.1** an alert's area is resolved, and the store is wired | an alert that carries its own polygon keeps it and nothing is fetched for it; a zone-only alert gets its zones' outlines; an alert whose zone cannot be got comes back with nothing, and that is not an error | Exercised with **no map at all**, which is the point: nothing draws in this release |

## Why the resolving lives in `app/`

The zone store is a domain. Whatever draws will be under `platform/`. **Neither
`modes/` nor `platform/` may import a domain**, and `app/` is the only place
that may name both - so the domain fetches, `app` wires, and whatever draws is
handed plain shapes from `platform/geo` that it can name (MG-3). That is the
same shape the spike used, arrived at from the rule rather than from the spike.

## What the resolver does, and what it refuses to decide

Every zone any alert names is asked for **once**, because the same zone is
commonly named by several alerts and by several locations' copies of one alert.
An alert that brought its own polygon is not looked up at all.

An alert whose zones could not be got comes back empty. **That is not an error
here**: whether a partly-known area should be drawn is a question for something
with a view to answer it against, and this has none. That was MG-10 before the
second red-team moved it; this is the shape it moved into.

## Gates

`gofmt`, `go vet`, `lint-imports`, `alloc-budget` and the full test tree pass.
`app`'s declaration set was re-captured for `resolveAlertAreas`.
