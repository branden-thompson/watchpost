# P1 build log — domain: widened feeds, bounded alerts, `domains/severe`

**Batch:** P1 (Tasks 1.1–1.10) · **Date:** 2026-08-28 · **Gate:** `make verify` ALL GATES GREEN · `make p10` 0 live, 0 unmatched

## Pre-code gate (before-you-write-code)

READY — plan approved (7f4fd7e); FULL GIT/TDD/DOCS in effect; `go build/vet/test ./...` green at HEAD; zero
new deps; no FFI; spike: go-studs `DataTable` renders six columns with bracketed spread headers at exactly
103 cells with `…` truncation and the positional gutter rule (spike test run, then deleted).

## What landed (TDD: RED by compile → GREEN)

| Task | Files | Notes |
|---|---|---|
| 1.1 | `detail.go` | `QuakeDetail`/`TropicalDetail`/`SevereDetail`; `clampProse` 4 000, `clampNonNeg`, `clampSlice` 50, `clampFloat` |
| 1.2 | `event.go` | `Name`, per-class pointers, `Title()`, name-aware `Sentence()` (no article when named) |
| 1.7 | `memo.go` | `sourceMemo`: skip decode on cache hit or equal sha256; errors never memoised; callers get their own slice; a successful empty parse memoises `[]` not nil |
| 1.3–1.5 | `usgs.go`, `nhc.go`, `nws.go`, `supersede.go` | `parse(body)` + memo-calling `Fetch`; render list decoded; PAGER enum; storm `Name`; `nwsProps` + `severeDetailOf`; parameters allowlist; guarded `Supersedes`/`SupersededBy` keyed on `SenderName` |
| 1.6 | `platform/snapshot/types.go`, `domains/weather/nws/alerts.go` | `Alert.SenderName`; every CAP field bounded on the location path; `declset.txt` re-captured (intentional addition) |
| 1.8 | `domains/severe/severe.go` | `NormalizeID` (OID grammar + the one legitimate prefix), `Classify` (`TabNone`), `Guard`, `Union` split into `addFeed`/`addLocations`/`locationRow` (P10-04), `Sort`/`Sorted`, `Cap` (positive-cap invariant), `ByTab` |
| 1.9 | `domains/severe/record.go` | `RecordOf` — the one class switch; `capExtras` shared; `shortDur`; `Plain` on every field; product-name precondition |
| 1.10 | `cov_test.go` | **COV 9/9 quake · 6/6 storm · 9/9 warning** from the committed probe fixtures |

## Deviations from the plan (all recorded, none silent)

1. **`sourceMemo` field/method name clash** (`events`) — Go forbids it; the plan's code (and two review
   rounds) missed it. Field renamed `last`. A compile-level defect the plan carried.
2. **The 0.12.0 superseded fixture** (`fetch_test.go`) predates the sender/product/newer guard; it now
   carries `senderName` + `sent` — the legitimate-update case still supersedes; the rogue case is
   `TestSupersedesOnlySameSenderProductNewer`.
3. **USGS depth** — the probe sample's first event is a landslide at depth 0; the render-list test asserts
   depth across the set, not per event (a surface event legitimately has none).
4. **P10 gate tooling** — the installed `a2dh` (Jul 26 build) lacks `p10`; the framework repo's Aug 18 build
   has it and the Makefile's `A2DH=` override runs it. Recorded for the readiness checklist.
5. **P10 findings on the new code, fixed at source:** `parse` 66 → split (`severeDetailOf`); `Union`
   complexity 22 → three helpers; `clampInt(lo=0)` → `clampNonNeg`; invariant density — real checks added
   where wrongness harms (`Union` one-row-per-key, `Cap` positive cap, `RecordOf` precondition, memo never
   memoises nil) and the two pure packages ledgered under the HUM-LEAD-approved package pattern
   (`.a2dh-p10-exemptions.yml`, "ratify at the 0.13.0 BUILD gate").
6. **`make schema`** regenerated the published report schema for `Alert.sender_name` (the `-race` gate's
   `TestPublishedSchemaMatchesGenerator` caught it).
7. **`go mod tidy`** removed a stale `x/tools` sum line the tidy gate had been reporting.

## Measurements

- `go test ./domains/globalfeed -run TestRenderListCoverage`: 100 % per class.
- `make verify`: fmt · tidy · vet · lint-imports · race · alloc-budget · … ALL GATES GREEN (2026-08-28).
