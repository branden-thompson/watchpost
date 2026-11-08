# Extending watchpost

> **Status:** everything named here exists and the steps are verified against the shipped code
> (Documented-Commands-Execute rule). `TestExtendingNamesRealSymbols` (`cmd/watchpost`) resolves every
> symbol and file path on this page against the tree, and fails on one that names nothing. There is no view registry. The two top-level views, the
> Observer and the Broadcaster console, sit under a thin **Router** (`modes/tty/router.go`): the
> Router is the Bubble Tea model handed to the program, and it fans messages to the two surfaces
> (FR-1.1, FR-1.2).

watchpost is organized so you can find a feature by its name (`domains/…`) and extend it in that
feature's folder plus the wiring each walkthrough below lists, site by site, with the check that
fails when a site is missed, or a plain "nothing fails" where none does. A test named here is in the
package of the step's file unless the step names its file; `go test ./<package> -run '^TestName$' -v`
runs it alone. Two invariants keep everything honest — the import
linter in `make verify` enforces the first; `term.Merge` plus its tests enforce the second:

1. **Data flows one way:** domains write `platform/snapshot`; `modes/` (all rendering: TTY and
   `report`) read ONLY the snapshot — never a domain. Anything the dashboard needs from a domain
   arrives as an app-provided hook in `tty.Config` (radio, voices, hydrate, spectrum).
2. **Keybindings are data** (D-15): every scope's commands are named once, in the keymap of the
   scope that owns them —
   `defaultKeyMap()` in `modes/tty/dashboard.go` for the Observer, `broadcasterKeyMap()` in
   `modes/tty/router.go` for the console, `defaultMapKeyMap()` in `modes/tty/map_pane.go` for the
   map while it is open, and `defaultChartKeyMap()` in `modes/tty/map_prop_chart.go` for the bands
   chart's window (an open window owns its keys, D-61); users override any of them via `[keys]` in
   config.toml by the action's name; `?` is reserved for help, which renders from the merged map so
   it is always truthful. The keys inside a window over the map - the arrows, space, enter and esc of
   the Overlays menu and the find, frequency and acknowledgement windows, and the bands chart
   window's esc, PgUp and PgDn - are fixed, not keymap actions (D-292).

---

## Walkthrough 1 — add a UV-index line to the location detail modal

Goal: show `Harmonized.UVIndex` in the detail view (`enter`).

| Step | File | What you do |
|---|---|---|
| 1 | `platform/snapshot/types.go` | Nothing — `Conditions.UVIndex *float64` already exists ([architecture §10.1](../06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md)). A nil pointer means "provider has no value" and renders `n/a`. |
| 2 | `platform/render/units.go` | Only if no formatter fits: add `UVIndex(v *float64) string` next to `Distance`/`Knots`/`TideHeight` — `platform/render` is the ONLY package that may import go-studs, and every formatter has an `n/a` path and a width-bound test. |
| 3 | `modes/tty/detail.go` | In `detailLines` (the modal body builder), add the row where the mock puts it, using the formatter from step 2. Rows are label/value pairs aligned by the modal's shared inset; follow the neighbours. |
| 4 | Tests | Extend the detail-modal test with a sentinel UV value (e.g. `7.3`); the report-mode parity test then asserts the same value appears in `--json` (M5). |

You never touch providers, the scheduler, or `modes/report`.

## Walkthrough 2 — add a keyed provider (the shipped pattern: NASA FIRMS)

Goal: a new source that needs a user key. `domains/fire/firms` is the worked example (B5).

| Step | File | What you do |
|---|---|---|
| 1 | `domains/<domain>/<name>/<name>.go` | Implement `snapshot.Provider`: `ID()`, `Domains()` (`["fire"]`, `["weather"]`, …) and `Fetch(ctx, req)` returning a `Fragment` with `PerLocation`. Use `platform/httpx` for every request — a 32-hex path segment or a `key`/`api_key` query value is redacted from every error and log line; cache with `httpx.TTL(d)` where the server sends no cache headers. |
| 2 | Key wiring | Take the key in the constructor **and** behind a lock with `SetKey` / `Enabled()` (`firms.go`): the Setup window keys a provider while the dashboard runs. Validate the key's shape before it is stored or used (`firms.CheckKey`) so the redaction never depends on a well-formed paste, and scrub it from transport errors yourself (`redactKey`). Users store it in the Setup window (`s`) or `[providers.<name>]\nkey = "…"` in config.toml (0600); `app` reads `cfg.Providers["<name>"].Key`. Keys never appear in output: the no-secret golden asserts it. |
| 3 | `app/<family>.go` + `app/dashboard.go` | Build the family in one place (`fireProviders`, `seismicProviders`), then give it a field in `providerFamilies` and a line in `newProviderFamilies` and in `providerFamilies.all` (`app/dashboard.go`). That one constructor is what the live pipelines **and** the one-shot report (`app/app.go:reportFetch`) fetch from, so a provider there is fetched by both; `TestAllAsksEveryFamily` fails a family `all` leaves out. Give `providerEndpoints` (`app/endpoints.go`) the host it talks to and `providerSources` its feature and badge, for the Status window: `TestEveryProviderHasAnEndpoint` and `TestEveryStatusSourceIsGroupedAndCredited` fail a provider with none. Give it a credit in `creditGroups` (`app/credits.go`), which About draws as `CreditLine` rows; its badge needs a chip (below). Give `newAssembler` its attribution case (`app/dashboard.go`) so the report's provider line names the source; **nothing fails without it**, the line just reads `ok ()`. Register a keyed provider **always**, and mark it `off` while unkeyed with `Assembler.SetInactive` (`livePipelines.markFIRMS`) so the API status never says "ok" for a feed that contributes nothing. |
| 4 | Scheduler | Cadence is per-**kind**: an existing kind (weather, obs, marine) rides its tier for free. A **new** kind needs: a `snapshot.FetchKind` (`platform/snapshot/types.go`); its domain in `platform/sched/sched.go:domainFor`, which `sched.Serves` reads to decide whether a tier asks a provider at all (`TestEveryFetchKindHasADomain` fails a kind with neither a case nor a reason in its `onDemand`); a tier line in `priorityTiers` and one in `recentTiers` (`app/pipelines.go`, the one owner of both lists); its name in `kindName` (`app/cadence_test.go`) and the cadence golden re-captured (`go test ./app -args -update-cadences`), which `TestCadenceTableIsTheDoc` holds; and `Domains()` on the provider. **The one-shot report asks its own list of kinds**, `kinds` in `app/app.go:reportFetch`: add the kind there too, or `watchpost report` never asks for it, and **nothing fails**. B5 added `KindFire` at 10 / 15 minutes. |
| 4a | A new block in the snapshot | A kind whose data is a new block (`fire`, `seismic`) also needs, in `platform/snapshot`: the state type and its field on `Location` and on `PartialData` (`types.go`); the assembler's map, made in `newAssembler`, stored in `mergeLocationLocked`, deleted in `SetLocations` and deep-copied out in `Snapshot` (`assembler.go`), with the copy in its own `merge_<kind>.go` (`merge_seismic.go` is the model). **Only the merge test in step 5 catches a missed one of those**: a map never made panics on the first fragment, and a store or copy left out publishes nothing. Then the published schema, regenerated with `make schema` and held by `TestPublishedSchemaMatchesGenerator` (`pkg/schema`), and the declaration-set goldens (`go test ./platform/snapshot ./app -args -update-declset`). |
| 4b | Where it shows | The plain report's line in `modes/report/report.go:RenderPlain`, with a sentinel value in `report_test.go`'s fixture and the golden re-captured (`go test ./modes/report -args -update-golden`); a Location Details section in `modes/tty/detail_<kind>.go`, wired into `detailLines` (`modes/tty/detail.go`). **Nothing fails if `detailLines` never calls it** except the section's own test. A credit badge's chip: `chipGrounds` (`modes/tty/map_parts.go`), whose `TestEverySourceIsCreditedOnce` fails a badge with none; a `MapChip…BG` token in `platform/render/theme.go`, with its colour in `defaultTheme` (the same file), in `lightOverrides` and in the `"Monochrome"` entry of `builtinOverrides` (both `themes.go`), each held by a theme test; and the token in `ChipGrounds` (`platform/render/contrast.go`), where **nothing fails if it is missing** but the chip is then never checked for contrast or uniqueness. |
| 5 | Tests | A fixture-backed `Fetch` test (httptest, recorded body) that also asserts the request shape; a key test (stored trimmed, malformed refused without echo); for a new block, the assembler merge test, which is the only guard on step 4a's assembler sites; and a test of the detail section. |

Geometry note: a hazard's shape lives in `platform/geo` (`Point`, `Ring`, `Polygon`, `Shape`, `Area`) because both a
domain and whatever draws must name it, and neither `modes/` nor `platform/` may import a domain.
**A `Shape` is areas, and an area is an outline then its holes** — the grouping is load-bearing,
because whatever draws reads an area's first ring as its outline and every ring after it as a hole,
so two areas flattened together make the second a hole in the first. An `Area` adds whether all of
the hazard's ground is known: a shape built from two of nine zones and one built from all nine are
otherwise the same value.
Read a shape from the wire with `platform/geo.ReadGeometry` — **never** by decoding `coordinates`
into an `any`, which recurses once per array level with no ceiling. Zone outlines come from
`domains/weather/nws/zones`; it deliberately keeps no cache of its own, because `platform/httpx`
already honours the lifetime a server declares and revalidates rather than discards.

Schema note: `by_provider` is additive — a new provider id needs no schema version bump (schema
`snapshot.SchemaVersion` policy, [architecture §10.3](../06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md)). A new top-level block (like `fire`) is a schema change and
lands with its `modes/report` lines and parity fixtures. While `SchemaVersion` is a release candidate
(`-rc`), a new block rides it, as `seismic` did: `make schema` rewrites the current `-rc` file. Whether
a block moves the version is the HUM LEAD's call; ask.

A station provider joins no host list but `providerEndpoints`: FR-3.8's closed list is the map's
sources (Walkthrough 5). The model for step 5's merge test is `TestAssemblerMergesSeismicAndIsolates`
(`platform/snapshot/snapshot_test.go`).

**Second worked example — a keyless hazard with its own detail section (`seismic`):** the same
steps, minus the key (step 2), with two of step 3 and 4b not modelled: seismic has no attribution case
in `newAssembler` and no line in `RenderPlain`, so for those copy fire's. Its shape adds two ideas worth copying. (a) *Shared canonicalised
requests* when a domain is regional: `domains/seismic/usgs` snaps each location's wide query to a fixed
grid so nearby locations share one request (the FIRMS-tile precedent), with a bounded, gauged parse memo
(`boxmemo.go`) — see `docs/where-things-happen.md` "A seismic request is made". (b) *A one-provider
domain* merges without fire's cross-provider fold: the assembler keeps the latest state per location and
deep-copies it into the snapshot (`platform/snapshot/merge_seismic.go`). The section itself is
`modes/tty/detail_seismic.go` (a `fireRows` sibling wired into `detailLines`), with a new
`render.SeismicMark` token and colour-off + `--ascii` tests.

## Walkthrough 3 — add a radio control

Controls live where they act: a key in `defaultKeyMap()`, a case in `toggleRadio`, a chip in
`radioControlLines`, a method on the `tty.Radio` interface, and its implementation on
`app/radio.go`'s `radioDeck`. The Dashboard calls it off the update loop, as a command
(`Dashboard.withCmd`, collected by `Dashboard.takeCmd`), and the deck reports
back through `RadioStatusMsg`. `[m] Mode` (UAT 97) is the smallest complete example.

## Walkthrough 4 — add a report kind

The growth axis this codebase was designed around, and the one a newcomer is most likely to be handed.
`platform/report/report.go`'s package comment is the authority; this is the route to it.

**The registry's cost is one row. The FEATURE's cost is not**: `TestEveryReportKindReachesTheComposer` fails a kind that composes to nothing. This table is what a new kind
costs.

| # | Where | What |
|---|---|---|
| 1 | `platform/report/report.go` | A `Kind` constant before `numKinds`, and a row in `all()`. Everything derived — the request window's rows, `Set.Describe()`, the running-order label — picks it up with no further edit. **This is the part that really is one row.** |
| 2 | `app/radio.go` (`segments`) | A branch under `if want.Has(report.X)` that assigns a value **and passes it to `Compose`**. Naming the kind is not enough: `TestEveryKindsBranchReachesTheComposer` fails a branch whose value never arrives. |
| 3 | `domains/radio/synth/compose.go` | A field on `synth.Reports`, and a tagging block if the kind is spoken. |
| 4 | `domains/radio/synth/<kind>.go` | The segments themselves, plus a source name and counts helper in `manifest.go`. |
| 5 | `domains/radio/script/scripts/<kind>/` | The script templates, found by folder and file name. Each part is spoken only where Go code asks for it (Walkthrough 7), and `TestBuiltinScriptsFollowTheConvention` reads the tree: a new folder joins its `Reports()` list. **A part no code asks for is simply not spoken**, and nothing fails. |
| 6 | `domains/radio/cast/cast.go` | A `Role` and a row in `Assignable()`, or the kind reads in an inherited voice. |
| 7 | `modes/tty/setup_cast.go` | `castRowOrder()` and `castLabel` are **hand-written** and derive from nothing. Miss this and the operator cannot cast the correspondent — silently. |
| 8 | A data source, if none exists | Walkthrough 2 in full: provider, `snapshot.FetchKind`, cadence tier, assembler merge, `credits.go`, and — for a new top-level block — a schema change with `modes/report` parity fixtures. |

**The two guards, and what they do not cover.** `TestEveryReportKindReachesTheComposer` and
`TestEveryKindsBranchReachesTheComposer` both derive their expectations from the registry, so neither
can be satisfied by editing a test. Neither can see a hook that is permanently `nil`: it assigns in the
source and returns a zero value at run time. **The last step is a fixture proving data arrives**, and no
gate performs it for you.

**Three things that are ratified, so do not "fix" them:**

- **The card manifest shows four sources** (`bcReadLines`, `modes/tty/broadcaster_slots.go`). The card's
  height is fixed so cards below it do not move; the fifth source appears in the detail window, which
  has a scroll rail. See `modes/tty/broadcaster_detail.go` — and note its test fixture is already named
  for an air-quality report.
- **A phrase is found by name**, not by a registry: Go code asks for a report's part, and the library
  reads `scripts/<report>/<part>.txt`. So a new part is spoken only once Go code asks for it
  (Walkthrough 7).
- **`synth.Reports` carries typed fields, not a table keyed by `Kind`.** A table erases the types that
  make the branches readable (`platform/report/report.go`).

**One thing that is not ruled:** nothing constrains `Spec.Label`'s width. `REPORT TYPE` is 20 cells at
its declared width and `FullLabel` is already 21; `render.PadTo` pads and never truncates, so a long
label shifts its neighbour rather than clipping. Ask before choosing one.

## Walkthrough 5 — add a map layer

The map's layers live in one registry, `app/maplayers.go`. A layer registers itself from its own file,
and Settings' MAP - LAYERS rows, the Overlays menu and the cost estimate all walk the registry, so a new
layer edits none of them. The earthquakes (`app/mapquakes.go`) are the smallest complete example.

| Step | File | What you do |
|---|---|---|
| 1 | `modes/tty/map_temp.go` (beside the other keys) | A key for the layer, e.g. `const LightningLayer = "lightning"`. Every overlay the layer makes has an id that starts with the key and a slash (`lightning/…`): the window switches a layer's overlays by the part before the slash. |
| 2 | `app/map<layer>.go` | In the file's `init`, `registerMapLayer(mapLayer{key, label, on, cost, chips})`: the words Settings shows, whether it is on by default, its cost function, and the badge's source names when they never change. A key registered twice panics. The new row appears in Settings and in the Overlays menu's *Data Points* group with no further edit. |
| 3 | the same file | The cost function: what one refresh would fetch if nothing were held, as bytes and requests. `refreshCost` sums it over the layers switched on; above 3,000,000 bytes or 25 requests, the window and Settings warn. |
| 4 | `modes/tty/map_prefs.go`, `app/maplayers.go`, `app/mapfeed.go` | The data. A flag on `MapAsk`, set from `layerOn` in `mapAsk`, so it is fetched only while on. A field on `mapInputs`, filled in `fetchInputs` (one goroutine an input, all joined before the feed reads any). Its overlays added in `mapFeedWith` (`addNow` for one drawn as now). Fetch in `app/` or a domain, never in `modes/`, and over fixed boxes or national files, never the view, unless the Status window's disclosure says so. A gridded field rides `mapTemperature` instead. |
| 5 | `app/map<layer>.go`, `app/maps.go` | Its hosts: a `<layer>Hosts()` returning `[]tty.MapSource`, appended in `mapSourceList`, with its badge in `mapBadges` (`app/endpoints.go`), so the Status window lists them under MAP DRAWING DATA, or COMMON DATA where the station uses the host too. A new host must be on FR-3.8's closed list. |
| 6 | `app/credits.go` | Its credit in `creditGroups`, with what the source's terms ask. |
| 7 | Tests | Add the key to `TestAFakeLayerPlugsInWithoutEditingTheOthers`' list and the host to `TestEveryRegisteredSourceIsOnTheClosedList`' closed list (both in `app/maplayers_test.go`), so a layer or host cannot arrive unnoticed. Then a cost test, and a feed test over a recorded answer showing the overlays arrive with the layer's key. |

**Fetch only while on.** The estimate counts only the layers switched on, so a layer that fetches while
off spends what the warning never shows.

## Walkthrough 6 — add a Setting

**Settings, not flags.** A choice a listener makes while watchpost runs is a Setting: a row in the
Settings window, saved to `config.toml`. A command-line flag belongs to a one-shot command (`watchpost
report`) and never to the live program.

A Setting is read from the config file at launch, carried into the model, drawn as a row, moved by its
arrows, and written back when the window closes. Two complete examples are worth reading side by side:
- **Reduce motion**, an on/off Setting of the Watchpost UI tab. An on/off Setting is a two-value
  picker: `kind: rowPicker` with `picker: true`, and a cycle of its own. **Not `rowToggle`**: that
  kind's arrows and space go to `toggleClass`, which only the tone rows answer, so a new row of that
  kind ignores space. Search for `ReduceMotion`, `reduceMotion` and `rowReduceMotion`.
- **The map's quake feed**, a map picker. Search for `MapQuakeFeed`, `mapQuakeFeed` and `rowMapQuakes`.

The last column says what fails if the step is skipped. Each step was proven by adding a throwaway
Setting of each kind on a scratch copy and leaving out one step at a time (0.19.5 batch 36).

| Step | File | What you do | Skipped, this fails |
|---|---|---|---|
| 1 | `platform/config/config.go` | The file's key: a field with its `toml:"…,omitempty"` tag, and a comment saying what its words mean and which is the default. | the compiler, at step 3 |
| 2 | `modes/tty/dashboard.go` | The field on `Config` (what the model is built with) and on `UIPrefs` (what Settings saves). | the compiler |
| 3 | `app/dashboard.go`, `app/themes.go` | The app hands the file's value to `Config` when it builds the dashboard (`livePipelines.ttyConfig`), and writes `UIPrefs`' value back to the file in the save (`setUIHook`). | `TestASavedSettingIsTheSettingAfterARestart` |
| 4 | `modes/tty/dashboard_choices.go` | The model's field in its group's struct (`mapChoices`, `uiChoices` or `radioChoices`), and one line in that group's builder (`mapChoicesOf` and the others) reading it from `Config`. A word the file does not offer reads as the default. Embedded, the field reads as `d.<name>` everywhere. | the builder line: `TestASettingIsSavedAsItIsWhateverSavesIt`, once step 10's fixture names the field; `TestEveryConfigFieldReachesTheModel` and `TestThePrefsSavedAreTheSettingsInEffect` hold goldens, so they fail only when the golden was recorded with the line present |
| 5 | `modes/tty/setup_ui.go` | One line in `uiForSave`, the value the model draws by written as the file's word; and the value mirrored into `d.cfg` in `applyUISaved`, beside its neighbours. | `uiForSave`: `TestASettingIsSavedAsItIsWhateverSavesIt`. The mirror: nothing, because nothing reads `d.cfg` for a Setting after launch; keep it with its neighbours |
| 6 | `modes/tty/setup_rows.go` | A row id in the `setupRowID` list and its row in `setupTable`: its group, the surface it belongs to, its kind, and `picker: true` for a picker. **The list's order is the focus order**, so a row inserted mid-list renumbers every row after it, and the goldens that name rows by number move. `setupTable` is at 59 of P10-04's 60 code lines: `06_docs/code-standards.md`'s specimen names the routes past it. | `TestEverySettingsRowIsRuledForItsSurface`. `picker: true`: `TestEveryChipWordIsRead`, once step 7's words are in |
| 7 | `modes/tty/setup_layout.go` | The picker's words in `pickerChipWords`, what its arrows change ("Motion", "Quakes"). The chip row reads them only on a row marked `picker: true`. | `TestEveryPickerNamesWhatItsArrowsChange`: without them the chips would offer a voice |
| 8 | the group's drawing | Draw the row. A UI Setting is a line in `uiLines` (`modes/tty/setup_ui.go`), beside Reduce motion, with `at` set when it has the focus, and `uiLabelW` widened if its label is longer than its neighbours'; a map Setting is a `mapRow` in `modes/tty/map_settings.go`, in the order the tab shows it. | the line: `TestEveryArrowRowChangesSomething`, `TestSetupGoldenEveryTab`. `at`: `TestSettingsFocusLineFollowsTheBalancedLayout`. `uiLabelW`: `TestAGroupsPickersLineUp` |
| 9 | the arrows | A UI Setting gets its own `cycle…` beside the state it moves (Reduce motion's is `cycleReduceMotion`, in `modes/tty/motion.go`) and a case in `cycleFocused` (`modes/tty/setup.go`) beside Reduce motion's, before the generic cases, because the order is the dispatch; the case returns `.flashed(forward)`, which lights the pressed arrow. A map Setting gets a case in `mapPrefArrow` (`modes/tty/map_prefs.go`) calling its own `cycle…`. **The `cycle…` ends with `d.uiTouched()`**: it marks the window for saving on close, and redraws it. | the case: `TestEveryArrowRowChangesSomething`. `uiTouched`: `TestAnArrowOnASavedSettingMarksItForSaving`. `.flashed` and the place in the order: only `TestEveryArrowRowsOutcomeIsPinned`'s golden, so read its diff before re-recording it |
| 10 | Tests | Add the Setting, set away from its default, to `everySettingChanged` (`modes/tty/config_reach_test.go`). Each `settings_kept_test.go` fixture takes two lines: in `modes/tty`, `keptPrefs` and the `Config` literal of `TestASettingIsSavedAsItIsWhateverSavesIt`; in `app`, the `want` and the `back` map of `TestASavedSettingIsTheSettingAfterARestart`. Then re-record the goldens a new UI row moves: `model-from-config`, `prefs-saved`, `setup-arrows`, `setup-133x44-ui` and `router-trace` (`go test ./modes/tty -args -update-golden`), reading each diff first (below). | `everySettingChanged`: the two config goldens only. The fixtures: the two settings-kept tests, which name the field |

`p` previews a voice on a voice row only (0.19.5 D-33), so a new picker needs nothing to stay silent.

Three things the table does not cover, because the Setting decides whether they apply:
- **Its surface is ruled, not chosen.** Observer, the Broadcaster, both with one value, or both with
  their own: `TestEverySettingsRowIsRuledForItsSurface` fails a row nobody ruled, and the HUM LEAD
  rules it. Ask.
- **A Setting both surfaces read** reaches the console through `Router.Update`
  (`modes/tty/router.go`), which carries Observer's value onto the Broadcaster as it does Reduce
  motion's.
- **A Setting a memoised region is drawn from** belongs in that memo's key, or the memo keeps
  serving the old frame: the masthead's is `mastheadKey` (`modes/tty/masthead.go`).

## Walkthrough 7 — add a spoken phrase

**Re-wording a line the station already says** is a text edit: change its
`domains/radio/script/scripts/<report>/<part>.txt`, or override it under the config dir's `scripts/`.
A test that pins the old wording fails, as `TestComposeReadsLikeNWR` pins the broadcast's sign-off;
`domains/radio/synth/testdata/cycle.golden` holds each segment's key, role and pause, not its words.

**A new line is a new part, and a part is spoken only when Go code asks for it by name.** A `.txt`
alone is never read, and every test still passes.

| Step | File | What you do |
|---|---|---|
| 1 | `domains/radio/script/scripts/<report>/<part>.txt` | The phrase. Its first line is a `{{/* <report>/<part>: … */}}` comment naming its data; nothing checks the comment. It must not render empty or end in a newline, which `TestBuiltinScriptsFollowTheConvention` holds. A data field needs its entry in that test's `data` map, and **the map is every script's fields together**: a name another script already uses passes even when this caller never supplies it, and the line is then dropped when it runs. |
| 2 | The code that speaks it | `Composer.Compose` (`domains/radio/synth/compose.go`) for the broadcast cycle, `eventScript` (`app/severe_read.go`) for an event read, `breakingLine` (`app/burst_words.go`) for a takeover. Ask the library for the part (`Composer.say`), make a `Segment` with its `Key` and its cast `Role` (an untagged segment is read by the root voice), and leave it out when the text renders empty, which an override of only a comment does. |
| 3 | Its place | A segment in a different voice from the one before it is introduced by a hand-over line. The sign-off is the one `SelfIntro` segment: a Station line placed before it hands over into the station, and the sign-off then introduces the same speaker again. A closing line goes after the sign-off. The broadcast cycle's `Compose` also composes each Broadcaster card's script (`composeFor`, `app/schedule.go`), so a line at the cycle's end ends every card too. |
| 4 | Tests and goldens | Re-capture `cycle.golden` (`go test ./domains/radio/synth -args -update-golden`). Update the tests that pin the segment list: `TestComposeWithZeroReportsIsTodaysBroadcast`, `TestComposeReadsLikeNWR`, `TestReportsAreSeparatedByAir`. Add the part to `TestTheAppsPartsExist`'s list (`domains/radio/script/script_test.go`), which is hand-written, so **nothing fails without it**. Write a test of the new segment's place, role and empty override. |
| 5 | `README.md`, "What the correspondent says" | The list of parts. It is hand-written, and nothing reads it. |

**Spoken words.** A quantity is worded by `domains/radio/synth/spoken_units.go`, the one owner of
spoken units, in the listener's units. A place name goes through `ExpandStates`, as `Compose` does for
the location. An abbreviation needs nothing in the script: `Pronounce` runs on every rendered
segment, reading `domains/radio/pronounce/rules/`, and a new spelling is a row in `domains/radio/pronounce/rules/words.txt`.
The library's own `Say` only strips terminal escapes from what it renders.

## Walkthrough 8 — add a line to a Broadcaster card

The console's cards are drawn by `modes/tty`, from the `lineup.Card` the Director holds, so a line that
shows something the card already carries changes nothing in `app` or `platform`. The UP NEXT card is the
one with lines: LIVE NOW is a row of the air box, not a card.

| Step | File | What you do |
|---|---|---|
| 1 | `modes/tty/broadcaster_slots.go` | The UP NEXT card's body is `readBody`, called from `upNextBox` (`modes/tty/broadcaster_upnext.go`). Add the row to its `rows` literal, after the row it follows, led by `bcCardInset`. **The card's height is fixed** (`bcReadCardRows`): `readBody` pads its rows to it, and a new line takes one of the blank rows the padding left. Raising `bcReadCardRows` instead makes every read card taller and moves the console's minimum height. |
| 2 | `modes/tty/broadcaster_manifest.go` | The line's words, in a helper beside `cardPulledShort`. Use the words the console already uses for the same thing: `reportTypeOf` names a card's reports as the running order does, and an empty set, which is every card the Director makes, is the whole report. `report.Set.Describe` alone says nothing for an empty set. |
| 3 | its width | Give the helper `lane.inner()-len(bcCardInset)`, the card's cell less its inset (`cardLane`, `modes/tty/broadcaster.go`). The console's floor is 100 columns (`bcMinCols`), where the UP NEXT cell is 36 cells. `render.TruncateCells` cuts with no ellipsis, so a line that can be cut adds the glyph set's own, `o.Glyphs().Ellipsis`: a literal "…" breaks `--ascii`. |
| 4 | Goldens | The console's frames are `TestConsoleGoldens` (`modes/tty/golden_surfaces_test.go`): 12 goldens, three sizes by two states by colour on and off. Re-record them (`go test ./modes/tty -args -update-golden`) and read each diff first: the new line, and nothing else moving. |
| 5 | Tests | Write the line's own tests in `modes/tty/broadcaster_upnext_test.go`: its words for an empty set, a subset and the full set; its place among the rows; the card's height unchanged; the ellipsis under `--ascii`; and the longest value it can hold at the floor width. **Without them only the goldens fail**, and they fail only if they were recorded with the line right. |
| 6 | The frame's budget | The console's frame allocations are pinned (`bcFrameAllocs`, `modes/tty/broadcaster_alloc_test.go`) and checked by `make alloc-budget`, which `-race` skips. A line that allocates more than the budget's headroom fails it there. |

What the console must not do is in `06_docs/02_features/0.16.0-broadcaster-ui/00-REQUIRED-READING.md`,
and its five roles are in `docs/where-things-happen.md`'s vocabulary. Whether a new line is how the
console draws a card or part of what the card says, which the Card Composer in `app` decides, is not
ruled for a general case: ask before adding one the Composer might own.

## When a golden or a declset fails

A golden holds a rendered frame, a trace or a table byte for byte. A declset holds a package's
declarations by file. Either fails when the thing it holds changes, which is right when the change was
meant, and the failure prints the first line that differs. To accept a change:

- **Read the diff first.** A golden that moved where the change did not mean it to is the defect the
  golden exists to catch.
- **Re-capture with the package's flag, after the package path**: `go test ./modes/tty -update-golden`.
  Before the path, `go test` reads the path as the flag's value and tests the directory it is run
  from, which fails with "no Go files". The flags are `-update-golden`
  (`modes/tty`, `modes/report`, `domains/radio/synth`), `-update-declset` (`app`,
  `domains/weather/nws`, `platform/history`, `platform/render`, `platform/snapshot`), `-update-plan`
  (`platform/lineup`) and `-update-cadences` (`app`).
- **Say why** in the batch's build log: which goldens moved, and that each move is the change.

## Rules you inherit for free

- **A network call is not automatically a provider, and Walkthrough 2 is not the only shape.** A
  `snapshot.Provider` is **location-scoped hazard data that merges into the snapshot** — the interface
  says so: `ID()`, `Domains() []string`, `Fetch(ctx, req) (Fragment, error)`. If what you are adding
  has no domain, no location and no fragment, it is not a provider, and putting it behind the seam
  means inventing all three, which makes the seam mean *less*. The one named exception today is
  `app/release.go`: it asks GitHub whether a newer release exists, **once, at startup**, and is opt-in
  (`update_check`). It appears in `[S]`'s host table only because it goes through `httpx`. Before
  adding another non-provider fetch, say in [architecture §1.1](../06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md) why it is not hazard data,
  state its bound, and make that bound reachable by a test. See its §1.1 and §1.2 (which root a new
  file belongs under).


- **Every colour is a theme token.** Views call `render.Tint(text, render.Tok(render.SomeToken))`;
  the tables' own palette is three tokens (`table.header`, `table.muted`, `table.name`) the
  seam applies through go-studs' `HeaderColor` / `CellStyles` with the kit's automatic styling
  switched off (`NoAutoStyle`), so `NO_COLOR` is honoured by one gate and a user theme file can
  restyle the whole frame. A new colour is a token declared in `platform/render/theme.go` with a value
  in `defaultTheme()`, and **four** places may need to know it: `builtinOverrides()` and
  `lightOverrides()` in `themes.go` (a theme that says nothing inherits the default — that is fine and
  usually right), `quattroOverrides()` in `quattro.go` (seven of the thirteen themes are generated
  there, and a miss surfaces as an AA failure with nothing pointing at the file), and `aaPairs()` in
  `contrast.go`, which is what actually holds a colour to WCAG AA. Register a FOREGROUND there against
  every ground it is painted on — the window, the modal tile, a band — and the lift does the rest; a
  background token belongs in one of the ground lists instead.
- **A new window is one `modal` constant** plus a case in `modalView` (what it draws), `modalWidth`
  (how wide) and, if it scrolls, `modalLines`; a key that opens it goes through `toggleModal`.
  Exclusivity is the type's: the rendered-frame test (`modal_test.go`) adds the window to its
  table and proves nothing stacks.
- **go-studs is patched, never edited.** An approved change to the kit is a
  `third_party/go-studs/patches/NNN-name.patch` with a row in `LOCAL_CHANGES.md`;
  `scripts/sync-go-studs.sh` re-applies the stack on every sync and refuses a drifted upstream.
  **We do not fork or reimplement a dependency to win performance** — a hand-rolled replacement for
  something lipgloss or go-studs already does has to be re-checked and re-invested on every upstream
  release, which is the cost the dependency was taken on to avoid. Patch narrowly, send it upstream,
  or accept the cost and record it.
- **Some slow things are slow on purpose.** `docs/accepted-costs.md` is the register of costs that
  were measured, priced and chosen — usually to protect what the listener sees. Read it before
  optimising the render path, the scheduler or the HTTP client. Every entry states the trigger that
  would re-open it; that trigger, measured, is the only route back, and the sites themselves carry
  `ACCEPTED COST` comments pointing at it.

- **`make verify` runs every gate in `06_docs/required-gates.txt` except three, and the three it
  skips matter.** `release-matrix` and `install-test` are CI-only — they build five platforms and
  install what was built, which `verify` runs before they exist. `p10` is a phase-exit gate, run by
  `make quality` at BUILD and REVIEW exit rather than on every verify. A green local `verify` does
  not cover them; a green CI run and a green `quality` do. Read `required-gates.txt` rather than any
  count written in prose (this page names none): the Makefile,
  `ci.yml` and that file are checked against each other by `cmd/watchpost/gates_test.go`, and a gate
  on none of them is caught by `TestEveryGateShapedTargetIsListedOrExempt`.
- **The gate oracle needs GNU Make 3.82 or newer** — the make CI runs. macOS ships 3.81 (2006), which
  has no `.SHELLFLAGS`, so a Makefile silenced that way would pass here and fail on CI. `make verify`
  refuses to run the oracle under 3.81 and says so by name. On macOS: `brew install make`, then put
  `/opt/homebrew/opt/make/libexec/gnubin` at the front of `PATH` so `make` resolves to 4.x.
- `vuln` is `govulncheck`, which downloads the tool on each run, and `lint` fetches the pinned
  golangci-lint the first time — the two steps that need the network. `make tree-free` says whether a gate run is already in flight; do not edit the tree while
  one is.
- New render primitives are built **on demand** — add them when a view needs them, never
  speculatively ([architecture §10.10](../06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md), item 5).
- Every package carries a package comment stating its contract; follow the pattern in
  `platform/config`, `platform/term`, `platform/httpx`.
- **Recording alert fixtures:** `go run ./tools/alertrec -zones CAZ554,CAC073 -every 20s -out feed.jsonl -for 10m`
  polls the live NWS feed into a replay fixture; drop it in `domains/alerts/testdata/replays/` and
  the M2/M3 harness replays it (`domains/alerts/replay_test.go`). The committed `basic.jsonl` is
  synthetic.
- **Refreshing the transmitter table:** `curl -o CCL.js https://weather.gov/source/nwr/JS/CCL.js && go run ./tools/nwrtable -in CCL.js` regenerates
  `domains/radio/stream/transmitters.csv` from the NWS `CCL.js`.
- Guard stateful paths with `platform/invariant`; pure total functions may be
  left without checks when a test pins them.
