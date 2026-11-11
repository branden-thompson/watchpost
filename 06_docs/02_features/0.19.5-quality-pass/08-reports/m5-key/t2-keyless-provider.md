# Answer key, T2: add a keyless provider (pollen index)

> **0.19.5 M5 answer key (D-24), hidden from the blind reviewer.** Built on a scratch copy of the
> tree at batch 41 (`e25a9ea9`) by doing the task and leaving out one site at a time. Batch 42 acted on
> it: the doc findings below are fixed, and the guards it added are named in the build log
> (`TestEveryFetchKindHasADomain`, `TestAllAsksEveryFamily`, `TestEveryChipWordIsRead`,
> `TestReduceMotionsChipsNameIt`). The site lists stand as the key; a row's "NO GUARD" is as measured
> at batch 41. Scratch paths are written as `<scratch>/`.


Scratch copy: `q195/m5key/t2-provider` (base commit `7ce73e2 base`, branch `main`, the change left uncommitted).
Provider built: `domains/pollen`. It is keyless (`pollen.example.org/v1/index?lat=&lon=`, JSON
`{index, level, dominant, observed_at}`), has a new fetch kind `KindPollen` and a new top-level
snapshot block `Location.Pollen *PollenState`. The reading shows in three places: `--json` (through the
snapshot and schema), the plain report line `pollen: index 7.3, level high, mostly tree`, and a
POLLEN row in Location Details.

How I checked the guards: I removed each site from the finished change on its own, ran the named
guard, and restored the file by copy. The driver is `q195/t2-scripts/omit.py` with `exps.py`, and the
raw results are in `q195/t2-scripts/omit.log`. "Repo guard" means a test or gate that existed before
the change. "New-test guard" means a test written as part of this change because the docs ask for
one (extending.md:53, step 5). A blind attempt may not write that test, so the repo-guard column is
the one to grade against. **NO GUARD** means I removed the site, ran every package that could notice
(whole, `-count=1`), and nothing failed.

## 1. Every site, and 2. its guard (each one checked by removing it)

| # | File | Symbol | Change | Repo guard when omitted (gist) | New-test guard |
|---|---|---|---|---|---|
| 1 | `domains/pollen/pollen.go` (new) | `Provider`, `New`, `ID`="pollen", `Domains`=["pollen"], `Fetch`, `Attribution`, `Host`, `indexTTL` | The provider. It uses `snapshot.FetchEach`, `client.GetText(…, httpx.TTL(30m))`, and `Forget` on a bad body. The level and dominant words come from a closed set, and the index is limited to 0..12 | Compiler: `app/pollen.go` cannot build | — |
| 2 | `domains/pollen/pollen_test.go` + `domains/pollen/testdata/index-oceanside.json` (new) | `TestFetchReadsTheRecordedAnswer` and 4 more | The httptest fixture test. It also checks the request shape | **NO GUARD**: with both removed, `./domains/pollen` reports "no test files" and `./cmd/watchpost` passes | (this is the test) |
| 3 | `platform/snapshot/types.go` | `KindPollen` const | New fetch kind after `KindSeismic` | Compiler (`pollen.go:73`, `sched.go:312`) | — |
| 4 | `platform/snapshot/types.go` | `PollenState` type | `AsOf`, `Index *float64`, `Level`, `Dominant`, `Source` | Compiler | — |
| 5 | `platform/snapshot/types.go` | `Location.Pollen` | `*PollenState` with `json:"pollen"` | Compiler (`assembler.go` Snapshot) | — |
| 6 | `platform/snapshot/types.go` | `PartialData.Pollen` | `*PollenState` | Compiler (`assembler.go` mergeLocationLocked) | — |
| 7 | `platform/snapshot/assembler.go` | `Assembler.pollen` field | `map[LocationKey]*PollenState` | Compiler | — |
| 8 | `platform/snapshot/assembler.go` | `newAssembler` | `pollen: map[…]{}` init | **NO GUARD** (snapshot, app and schema pass). It panics at run time on the first pollen fragment: "assignment to entry in nil map" | `TestAssemblerMergesPollenAndIsolates` panics |
| 9 | `platform/snapshot/assembler.go` | `mergeLocationLocked` | `if pd.Pollen != nil { a.pollen[k] = clonePollen(pd.Pollen) }` | **NO GUARD** (snapshot, app and schema pass). Pollen never reaches the snapshot | `TestAssemblerMergesPollenAndIsolates`: "the pollen fragment must reach the published location: <nil>" |
| 10 | `platform/snapshot/assembler.go` | `SetLocations` | `delete(a.pollen, k)` | **NO GUARD** (snapshot and app pass). A removed and re-added place inherits stale pollen, and the map grows | same test: "a location removed and added again starts with no pollen state" |
| 11 | `platform/snapshot/assembler.go` | `Snapshot` | `loc.Pollen = clonePollen(ps)` | **NO GUARD** (snapshot, app, schema and report pass) | same test: "…must reach the published location: <nil>" |
| 12 | `platform/snapshot/merge_pollen.go` (new) | `clonePollen` | Deep copy on the way in and out. If it is skipped (aliasing at Apply or at publish), the result is: | **NO GUARD** for aliasing (snapshot passes without the new test) | same test: "must alias neither the fragment nor a published value: 9.9" (publish) / ": 1" (apply) |
| 13 | `platform/snapshot/merge_pollen_test.go` (new) | `TestAssemblerMergesPollenAndIsolates` | The assembler merge test (extending.md:53, "for a new kind, the assembler merge test") | **NO GUARD** (removed, snapshot passes) | (this is the test) |
| 14 | `platform/snapshot/testdata/declset.txt` | golden | +`const KindPollen`, `func clonePollen`, `type PollenState` (`-update-declset`) | `TestDeclarationSetUnchanged` (platform/snapshot): "top-level declarations changed… added: [const KindPollen …]" | — |
| 15 | `platform/sched/sched.go` | `domainFor` | `case snapshot.KindPollen: return "pollen"` | **NO GUARD** (sched, pollen and app pass). `sched.Serves` returns false, so neither the tiers nor the one-shot report ever call `Fetch`. Pollen is silently never fetched | none (the provider test calls `Fetch` directly) |
| 16 | `app/pollen.go` (new) | `pollenProviders(client)` | The family's one constructor, shared by the dashboard and the report | Compiler (`app.go:82`, `dashboard.go:69`) | — |
| 17 | `app/doc.go` | the package map | Add `pollen.go` to the providers-by-family row | `TestEveryFileIsInItsPackagesMap` (cmd/watchpost): "app/pollen.go is in no row of app/doc.go" | — |
| 18 | `app/app.go` | `reportFetch`, providers | `providers = append(providers, pollenProviders(client)...)` | **NO GUARD** (app and cmd/watchpost pass). `watchpost report` and `--json` carry no pollen | — |
| 19 | `app/app.go` | `reportFetch`, `kinds` list | add `snapshot.KindPollen` | **NO GUARD** (app and cmd/watchpost pass). The provider is registered but never asked: its status reads "ok" with no data | — |
| 20 | `app/dashboard.go` | `livePipelines.pollen` field | `[]snapshot.Provider` | Compiler | — |
| 21 | `app/dashboard.go` | `newLivePipelines` | `pollen: pollenProviders(client)` | **NO GUARD** (app and cmd/watchpost pass). The live dashboard has no pollen | — |
| 22 | `app/dashboard.go` | `livePipelines.providers()` | `return append(out, lp.pollen...)` | **NO GUARD** (app passes, even `TestEveryProviderHasAnEndpoint`, because that test builds its provider set through `providers()`) | — |
| 23 | `app/dashboard.go` | `newAssembler` switch, plus the `domains/pollen` import | `case "pollen": asm.SetAttribution(…, "secondary", pollen.Attribution)` | **NO GUARD** (app and cmd/watchpost pass). The report prints `provider pollen: ok ()`. The shipped keyless example `usgs` has no case either | — |
| 24 | `app/pipelines.go` | `priorityTiers` | `{Kind: KindPollen, Every: 30m}` | `TestCadenceTableIsTheDoc`: "a cadence changed — … re-run with -update-cadences" | — |
| 25 | `app/pipelines.go` | `recentTiers` | `{Kind: KindPollen, Every: 1h}` | `TestCadenceTableIsTheDoc` (same gist) | — |
| 26 | `app/cadence_test.go` | `kindName` map | `snapshot.KindPollen: "pollen"` | `TestCadenceTableIsTheDoc` (against the committed golden). If the golden is recaptured without it, the table carries a blank tier name and nothing fails | — |
| 27 | `app/testdata/cadences.md` | golden | Two pollen rows (`go test ./app -update-cadences`) | `TestCadenceTableIsTheDoc` | — |
| 28 | `app/credits.go` | `creditGroups` | A POLLENWATCH group: `{Badge: "POLLEN", What: "Pollen Index", Host: "pollen.example.org"}` | `TestEveryStatusSourceIsGroupedAndCredited` (app): "provider pollen: {Feature:1 Badge:POLLEN}; want a feature and a credited badge" | — |
| 29 | `app/endpoints.go` | `providerEndpoints` | `"pollen": {"pollen.example.org"}` | `TestEveryProviderHasAnEndpoint`: "provider \"pollen\" has no endpoint". This fires **only because of row 31** | — |
| 30 | `app/endpoints.go` | `providerSources` | `"pollen": {Feature: tty.FeatureWeather, Badge: "POLLEN"}` | `TestEveryStatusSourceIsGroupedAndCredited`: "provider pollen: {Feature:0 Badge:}; want a feature and a credited badge" | — |
| 31 | `app/stats_test.go` | `TestEveryProviderHasAnEndpoint`'s hand-built `livePipelines{…}` | add `pollen: pollenProviders(c)` | **NO GUARD**. Removed, app passes. **Removed together with row 29, app still passes** (and so does the providerSources check, which walks `providerEndpoints`). The test's comment says "the ids come from the PRODUCTION construction", but it builds the struct by hand, so a new family field is invisible to it | — |
| 32 | `app/testdata/declset.txt` | golden | `+func pollenProviders` | `TestDeclarationSetUnchanged` (app) | — |
| 33 | `modes/tty/map_parts.go` | `chipGrounds` | `"POLLEN": render.MapChipPollenBG` | `TestEverySourceIsCreditedOnce` (app): "\"Pollen Index\" under POLLENWATCH has no chip of its own (D-235)" | — |
| 34 | `platform/render/theme.go` | `MapChipPollenBG` token const | `"map.chip.pollen.bg"` | Compiler | — |
| 35 | `platform/render/theme.go` | `defaultTheme()` | `MapChipPollenBG: "48;2;163;163;10"` | `TestEveryChipIsItsOwnAndReadsInEveryTheme`: "<theme>: map.chip.pollen.bg's words read 0.00:1 on \"\"" | — |
| 36 | `platform/render/themes.go` | the Monochrome overrides | `MapChipPollenBG: "48;2;170;170;170"` (a grey that is not used elsewhere) | `TestMonochromeHasNoColourInIt`: "map.chip.pollen.bg = … is a coloured GROUND" | — |
| 37 | `platform/render/themes.go` | `lightOverrides()` | `MapChipPollenBG: "48;2;236;252;203"` | `TestTheLightThemePaintsNoDarkGround`: "map.chip.pollen.bg … is a dark ground (luminance 0.34)" | — |
| 38 | `platform/render/contrast.go` | `ChipGrounds()` | add `MapChipPollenBG` | **NO GUARD** (render and modes/tty pass). The chip is then never checked for uniqueness or for AA in any theme | — |
| 39 | `platform/render/aa_completeness_test.go` | `unmeasurable` | `MapChipPollenBG: "a chip ground…"` | `TestEveryTokenIsMeasuredOrExcused`: "map.chip.pollen.bg is in neither — carry it, or write down why nothing can" | — |
| 40 | `platform/render/testdata/declset.txt` | golden | `+const MapChipPollenBG` | `TestDeclarationSetUnchanged` (render) | — |
| 41 | `modes/report/report.go` | `RenderPlain`, plus the new `pollenLines` and `word` | The pollen line, printed when `loc.Pollen != nil` | `TestPlainRenderContainsCoreData` ("plain render missing \"pollen: index 7.3…\"") and `TestPlainReportGoldenAt80Columns`. Both fire **only because of row 42** | — |
| 42 | `modes/report/report_test.go` | `fixtureSnapshot`, `TestPlainRenderContainsCoreData` | Sentinel `Pollen{Index: 7.3, …}` and the wanted line. M5 reverse parity then holds 7.3 to a JSON leaf | `TestPlainReportGoldenAt80Columns` drift + `TestPlainRenderContainsCoreData`. Without both this row and row 41, nothing in the repo checks the report line | — |
| 43 | `modes/report/testdata/plain-80.golden` | golden | +`  pollen: index 7.3, level high, mostly tree` (`go test ./modes/report -update-golden`) | `TestPlainReportGoldenAt80Columns`: "plain report drifted from the golden" | — |
| 44 | `modes/tty/detail_pollen.go` (new) | `pollenRows` | The POLLEN row (`detailRow`/`gridRow`, "n/a" for a nil index) | Compiler (`detail.go:62`) | — |
| 45 | `modes/tty/detail.go` | `detailLines` | `if loc.Pollen != nil { …pollenRows… }` after `seismicRows` | **NO GUARD** in the repo (modes/tty passes without the new test). Details never show pollen | `TestDetailPollenRowShowsTheIndex`, `TestDetailPollenWithoutAnIndexReadsNA` |
| 46 | `modes/tty/detail_pollen_test.go` (new) | 3 tests | Detail-row tests | **NO GUARD** (removed, modes/tty passes) | (this is the test) |
| 47 | `pkg/schema/watchpost-report.v1.1.0-rc.schema.json` | the published schema | Regenerated (+67 lines). The documented route is `make schema` = `go run ./cmd/watchpost schema > …`. I produced the same bytes from `schema.Generate()` in a throwaway test, so I never ran the CLI | `TestPublishedSchemaMatchesGenerator`: "pkg/schema/*.schema.json is stale — run `make schema`" | — |

Sites that are optional or a judgement call, not done in this copy. None of them has a guard:
- `06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md:397-419` (§11.1) says
  "edit there [`app/testdata/cadences.md`], then copy". Nothing checks the copy, and it is **already stale**: it has no
  fire or seismic rows, and its kinds list (`:416-419`) stops at `KindMarineObs`. **NO GUARD.**
- `docs/architecture.md:23` (the domains node in the diagram) would gain `pollen`. Nothing requires it. If it
  is edited, the atlas must be regenerated (`tools/atlas` `TestTheAtlasMatchesAFreshRender`). **NO GUARD.**
- `snapshot.SchemaVersion` (`types.go:22`) stays `1.1.0-rc`. extending.md:67-69 says "a new top-level
  block (like `fire`) is a schema change", but neither it nor any test says whether to bump the version.
  Seismic added `seismic` under the same `-rc`. Not bumped here. **NO GUARD.**
- `pkg/schema` `TestGeneratedSchemaValidatesRealEnvelope` builds its envelope from nws and coops only, so nothing
  validates a real pollen block against the schema. Shape errors are caught only through generation.
- Registries a keyless station provider does **not** join, checked against the code: the FR-3.8 closed list
  (`TestEveryRegisteredSourceIsOnTheClosedList`) covers only map sources; `hostSources` is only for hosts
  no provider claims; User-Agent arrives with the shared client (`app/app.go:UserAgent`,
  `dataClientConfig`); the httpx cache has no host registry (TTL per request, a disk floor above 5 min,
  sweep by file name); there is no config or Setting (keyless); there is no `SetInactive` (that is for keyed
  providers only); there is no `app/stats.go` gauge (only for providers that keep a memo); `modes/tty/doc.go`
  already covers `detail*`.
- The client is `HTTPSOnly`, so an `http://` production base would be refused at run time, and no test
  holds a station provider's default base to https (the map hosts are held to it in `maplayers_test.go`).

**Count:** 47 sites (40 production or registry sites, plus tests, fixtures and goldens), and 4 optional doc
or judgement sites. Some rows gather several adjacent edits in one file.

**NO GUARD rows in the repo's own tests (16):** 2, 8, 9, 10, 11, 12, 13, 15, 18, 19, 21, 22, 23, 31, 38, 45,
and 46 is the same kind as 2 and 13. The worst are **15** (`domainFor`, which means pollen is never fetched
anywhere), **18/19** (the one-shot report path in `app/app.go`, which extending.md never names), **21/22**
(the live pipeline), and **31** (the endpoint guard goes blind unless its test fixture is edited by hand).
The tests the docs ask for (step 5) catch 8-12 and 45. Nothing catches 15, 18, 19, 21, 22, 23, 31 or 38.

**Lightweight gates on the finished change:** `make lint-imports`: OK. `make dupes`: 2 groups, both
ratified, 0 NOT. `make wires`: 95 members, 0 NOT (`FetchKind` has no `num…` sentinel, so `wires` does
not see `KindPollen`). `go run ./tools/authoring`: no findings. `gofmt -l`: clean. `go vet` on the touched packages: clean.

## 3. Doc accuracy (for each step used)

| Doc:line | Routes right? | Finding |
|---|---|---|
| README.md:623-634 "Start here" | Yes | Routes to the four docs; every one exists. |
| docs/extending.md:10-12 | **Wrong** | "extend it by touching **one folder plus one wiring line**". A keyless provider with its own kind took 47 sites in 26 files. |
| docs/extending.md:14-16 | Yes | One-way data flow; `make lint-imports` holds it (code-standards.md:45, architecture.md:66). |
| docs/extending.md:49 (step 1) | Yes | `snapshot.Provider` (ID/Domains/Fetch), `platform/httpx`, `httpx.TTL(d)` all exist as described. |
| docs/extending.md:51 (step 3) | Partly | Right: `fireProviders` pattern, `livePipelines.fire`/`marine`/`providers()`, `newAssembler` attribution case, `creditGroups`, `providerEndpoints`, `TestEveryProviderHasAnEndpoint`. **Missing:** (a) `app/app.go:reportFetch`, both its provider list and its separate `kinds` slice; leaving them out drops the provider from `watchpost report`/`--json` with no guard. (b) The `livePipelines` struct field and its `newLivePipelines` initialiser. (c) `providerSources` in the same `app/endpoints.go`, required by `TestEveryStatusSourceIsGroupedAndCredited`. (d) The credit's Badge must be a known chip: `chipGrounds` in `modes/tty/map_parts.go` (`TestEverySourceIsCreditedOnce`), which needs a new theme token in `platform/render` (`theme.go` const + `defaultTheme`, `themes.go` Monochrome + `lightOverrides`, `contrast.go:ChipGrounds`, `aa_completeness_test.go:unmeasurable`). The colour rule at extending.md:206-217 describes tokens in general and names neither chips nor `ChipGrounds` nor Monochrome. (e) `TestEveryProviderHasAnEndpoint` builds `livePipelines` by hand (`app/stats_test.go:33-38`), so a new family needs a line there or the "fails a provider with none" claim is false for it. **Misleading:** the "attribution case" is described as a must, but the shipped keyless example (`usgs`) has none and nothing checks it. |
| docs/extending.md:52 (step 4) | Partly | Right: `snapshot.FetchKind`, `priorityTiers`/`recentTiers` in `app/pipelines.go`, `Domains()`. **Missing:** `platform/sched/sched.go:domainFor`, the kind→domain switch that `sched.Serves` reads. Without it the new kind is never fetched, and no test notices. Also missing: `kindName` in `app/cadence_test.go` and the `app/testdata/cadences.md` golden (the flag is named at :189, but the step does not say the golden moves). |
| docs/extending.md:53 (step 5) | Yes | The tests it asks for are the only guard on rows 8-12 and 45. |
| docs/extending.md:67-69 (schema note) | Partly | Says a new block "is a schema change" and lands with `modes/report` lines and parity fixtures. It names no file or command for the schema: `pkg/schema/watchpost-report.v<ver>.schema.json`, `make schema`, `TestPublishedSchemaMatchesGenerator`. It does not say whether `SchemaVersion` bumps. It names none of the snapshot sites (`Location` field, `PartialData` field, the `Assembler` map plus `newAssembler`, `mergeLocationLocked`, `SetLocations`, `Snapshot`). |
| docs/extending.md:71-79 (seismic, keyless) | Yes, but thin | "The same five steps, minus the key". It names `merge_seismic.go` and `detail_seismic.go`/`detailLines` correctly, and adds no further sites. |
| docs/extending.md:185-189 (golden flags) | Yes | `-update-golden` (modes/report), `-update-declset` (app, platform/snapshot, platform/render), `-update-cadences` (app) are all correct and all needed. It omits `make schema`. |
| docs/where-things-happen.md:30 | Yes | The one place that hints `app/endpoints.go` also names "what each provider and host serves and its badge" (that is `providerSources`). |
| docs/where-things-happen.md:40 | Partly | `runTier → Provider.Fetch → Assembler.Apply` is right, but `sched.Serves`/`domainFor` decides whether a tier calls a provider at all, and the page does not mention it. |
| docs/where-things-happen.md:48 | **Slightly wrong** | "Apply stores `a.seismic[k]`". In the code, `Apply` delegates to `mergeLocationLocked` (`assembler.go:254-290`), which stores it. `Snapshot` deep-copies through `cloneSeismic` (correct). |
| docs/where-things-happen.md:63 | Yes | `app/app.go:ReportOnceWithStats` is the route to `reportFetch`, the one-shot path extending.md leaves out. |
| docs/architecture.md:12, :66, :83 | Yes | Credits in `app/credits.go`; `make lint-imports`; domains as providers. |
| 06_docs/code-standards.md:39-46, :287-296 | Yes | `make lint-authoring`, `make dupes`, `make lint-imports`, `make wires`, `go test -run Extending ./cmd/watchpost` all exist and ran as described. |
| 06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md:397-399 | **Stale** | "edit there, then copy". The table under it lacks fire and seismic (`:401-414`), and the kinds list (`:416-419`) lacks `KindFire` as a live tier, `KindSeismic` and every later kind. Nothing checks the copy. |

## 4. IDs met, resolved through `06_docs/ids.md`

The check is `q195/t2-scripts/ids.py`. It looks for every `PREFIX-n` token in README:623-641,
docs/extending.md, docs/where-things-happen.md, docs/architecture.md and 06_docs/code-standards.md,
and matches each against the first cell of an ids.md row.

**Resolve:** D-10, D-15, D-30, D-56, D-61, D-124, D-160, D-292, 0.19.5 D-4/D-10/D-11/D-12/D-13/D-27/
D-30/D-33/D-34, 0.19.5 A-2, 0.19.0 D-142, DC-14, JM-3, DR-21, F-21, FR-1.1, FR-1.2, FR-3.8, FR-9, FR-9.4.

**Do not resolve through ids.md:**
- `0.15.0 FR-7.1` (where-things-happen.md:68): ids.md has FR-7.1 only for **0.16.0** (a breakpoint
  vocabulary row). The 0.15.0 FR-7.1 (update check) exists only in `0.15.0-…/08-reports/plan-report.md:323`
  as a range "FR-7.1 … FR-7.5", which is not a defining row.
- `F-111` (extending.md:94): no row. It is mentioned in `0.16.0-broadcaster-ui/08-reports/red-team-build.md:87` and `p10-ledger.md:143`.
- `MVS-D-3` (where-things-happen.md:54), `MVS-D-15` (:60), `MVS-D-80` (:147; code-standards.md:225):
  ids.md has no MVS-prefixed rows. The only multi-voice "D-3"/"D-15" rows are red-team findings
  (`red-team-discover.md`), not the MVS rulings. code-standards.md:225 calls `MVS-D-80` "qualified", but
  no index row resolves it.
- `R3-A-01` (where-things-happen.md:34): no row. It is defined in `severe-alerts-modals/08-reports/red-team-build.md:21`.
- `AP-HIST-01`, `AP-CITE-01`, `AP-CAPS-01`, `AP-DEAD-01`, `AP-OK-01`, `AP-SHELL-01`, `SN-02`, `SN-03`,
  `P10-00`…`P10-10`: not in ids.md, and that is by design. They are defined in 06_docs/code-standards.md itself
  (where-things-happen.md:159 says so for P10).
- Not `PREFIX-n` IDs, so never indexed: `UAT 97` (extending.md:87), `B0`/`B5` (where-things-happen.md:155; extending.md:45),
  `M2`/`M3`/`M5` (extending.md:39, :256; **M is not in where-things-happen's prefix table at :152-160 either**),
  `Q6.4`/`Q7.4` (code-standards.md:344, :224), and `§n` section references into the watchpost-cli architecture.

## Final run with the complete change in place

All touched packages were run whole with `go test -count=1` after every omission had been restored:
`domains/pollen`, `platform/snapshot`, `platform/sched`, `platform/render`, `modes/report`, `pkg/schema`,
`cmd/watchpost`, `modes/tty` and `app`. **All 9 passed** (app 137.7 s, cmd/watchpost 75.1 s, modes/tty 61.5 s).
The log is `q195/t2-scripts/whole2.log`. `go build ./...` is clean.
