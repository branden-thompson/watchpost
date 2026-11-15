---
title: "0.19.6 PLAN — design CK: the console, the kit and the test-support package (step 1 of modes/tty's split)"
date: 2026-11-15
phase: PLAN
agent: CK
status: "FILED — checked by the lead against e57afeaf; its choices ruled D-21 to D-26 and A-19 to A-27"
---

# Design CK: `modes/tty` step 1 — the kit, the console, and the memo guard's test support

> **Filed from PLAN's design agent CK.** Its tables are in `design-data/CK/`. `<scratch>` names the
> session's working directory, which is not committed: the computed graph there is committed in batch 0 as
> its own module (A-12). Where a ruling chose differently from this design, the ruling stands.

## Verdict

Step 1 is a move with one small prep: no interface, no cycle, no new message, no new allocation.

- **The kit** is TS-4's closure, 41 declarations in 12 root files, reproduced. It goes to
  `modes/tty/internal/frame` (name an A-row) with **25 exported names and 5 methods**. TS's 42nd
  declaration, `indentBody`, is the console's alone (`broadcaster_detail.go:160` is its only use), so
  it moves with the console and not into the kit.
- **The two snapshot messages stay at the root** (recommended, P3). Today they are what forces the
  kit out of `internal/`. The Router hands the console the snapshot through two methods, and `app`
  does not change. The console's six messages move with it to `modes/tty/console`, which stays
  importable because `app` sends them (C1).
- **The console's API:** 40 exported declarations: 9 types, 3 constants, 3 functions and 25
  methods. `Opening` (7 fields) and `Mirror` (5 fields) are among them.
- **The mirror adds nothing.** `SetMirror` takes a pointer receiver. Measured with `-gcflags=-m`
  and `-S` on overlay builds, it is inlined, adds no escape and no stack copy, and leaves
  `Router.Update`'s frame unchanged. The value-typed `Mirrored` also allocates nothing, but adds two
  472-byte copies and 48 B of stack to every update.
- **The test-support package** is `modes/tty/internal/ttytest`. Both the root's and the console's
  tests may import it. **The unchanged engine breaks more P10 rules than D-15 ruled:** besides
  P10-09 (`unsafe`, `reflect`), it breaks P10-01, because `walkAt` (`:167`) and `bump` (`:282`,
  `:285`) recurse. That needs a ruling.
- **Tests under D-14:** 127 crossing references at test-function level, plus 61 test-helper
  crossings. **20 references in 16 test functions in 10 files cannot be served without a test-only
  export.** That is the ruling D-14 owes. If the clock design K1 is refused, it rises to 23.
- **The router trace's new form** is keyed by message index, sets both clocks by pinning the
  Observer's before `NewRouter`, and describes each command by its shape.
- **Mutants:** of the 80 anchored in `broadcaster*.go`, 77 change path and 3 stay at the root.
  Across the corpus, 26 anchors change text: 12 name a kit symbol, 14 a console one. None names
  `tty.`.

## What I computed, and how

Everything is in `design-data/CK/`. Every script runs offline, and none touches the repository.

| File | What it is |
|---|---|
| `declpos/` | a stdlib-only Go tool that prints every top-level declaration of a directory with its line range. Its outputs are `decls-tty.tsv` (non-test) and `decls-tty-tests.tsv` |
| (reproduction) | TS's closure, from `<scratch>/ts`: `python3 -c "from closure import closure; from mapA1 import KIT; print(closure(set(KIT)))"` gives 41 declarations, 0 blocked edges |
| `tests_ck.py` | the test files under the console+kit map (TS's A2 assignment), whole-file: 41 straddle, 145 references outside the file's destination |
| `tests_fn.py` | the same at **test-function** granularity, iterated to a fixed point because helpers follow their users; `fn-refs.tsv` |
| `sites_ck.py` → `fn-sites.txt` | the line of every crossing reference |
| `classify_ck.py`, `rollup_ck.py` | the D-14 class of every reference: **`tests-CK-refs.tsv`** (188 rows) and **`tests-CK.tsv`** (62 files) |
| `mutants_ck.py`, `anchors_ck.py` → **`mutants-CK.tsv`** | each `modes/tty` mutant's target, the declaration its anchor sits in, where that lands, and whether its text names a renamed identifier (string literals only, through Python's tokenizer) |
| **`kit-CK.tsv`** | the kit's 41 declarations and 4 methods, by `file:line`, with the exported name, the kit file and the users |
| **`console-api-CK.tsv`** | the console's 40 exported declarations, with the signature, today's name and the Router line |
| `probe/` | overlay builds of `modes/tty` (`go build -overlay`, the tree untouched): v1, a value `Mirrored`; v2, a pointer `SetMirror`; `-gcflags=-m` for each (`m-v1.txt`, `m-v2.txt`), and `Router.Update`'s `-S` for each and for the baseline (`upd-base.s`, `upd-v1.s`, `upd-v2.s`) |
| `cmdname/` | a scratch program against the cached `charm.land/bubbletea/v2@v2.0.3` that prints the runtime names of `Tick`, `Every`, `Batch` and `Sequence` commands (the trace's stable description) |
| `m-base.txt` | `go build -gcflags=-m ./modes/tty` at the baseline |

Sizes come from `<scratch>/ts/sizes`: `Broadcaster` is 472 B with 29 fields, `Router` 7,136 B,
`Dashboard` 6,584 B, `memoStats` 24 B, `mastheadSlot` 192 B and `cursorSlot` 88 B. The toolchain
is go1.27.4. I ran no `go test`, no make and no benchmark.

---

## CK-1: the kit

### Its declarations (`kit-CK.tsv`)

The closure is 41 declarations. Seeded with TS's 20 (`<scratch>/ts/mapA1.py:3`), it stops at no
`Dashboard`, `Router` or `Broadcaster` symbol: 0 blocked edges. It also needs the four methods on
its own types. By source file:

| Source file | Declarations (`file:line`) → exported name, or kept unexported |
|---|---|
| `dashboard.go` | `SnapshotMsg` `:27`, `RecentSnapshotMsg` `:32` (both stay at the root under P3, below); `tickMsg` `:1053` → `TickMsg`; `tickEvery` `:1059` → `TickEvery`; `tick` `:1063` → `Tick`; `armShimmer` `:1117` → `ArmShimmer` |
| `masthead.go` (moves whole) | `mastheadTitle` `:32` → `MastheadTitle`; `mastheadStamp` `:49` → `MastheadStamp`; `mastheadKey` `:74` (unexported); `mastheadKeyOf` `:87` → `MastheadKeyOf`; `mastheadSlot` `:102` → `MastheadSlot`, with `get` `:109` → `Get` |
| `body.go` | `staleAfter` `:95` (unexported); `agoWords` `:107` → `AgoWords`; `apiSummaryOf` `:129` → `APISummaryOf`; `providerCounts` `:144` and `countProviders` `:148` (unexported); `weatherRow` `:400` → `WeatherRow`; `mmdd` `:444` (unexported); `trend` `:453` → `Trend` |
| `cursor.go` | `sides` `:28` → `Sides`; `focusShare` `:32` (unexported); `cursorLook` `:36`, with `cursor` `:85` (unexported); `cursorSlot` `:47` → `CursorSlot`, with `finish` `:63` → `Finish`; `focusCell` `:98` and `lineAt` `:118` (unexported); `focusMark` `:135` → `FocusMark`; `sidesOnly` `:146` (unexported); `surfaceMark` `:171` → `SurfaceMark` |
| `memo.go` | `memoStats` `:66` → `MemoStats`, with `counts` `:73` → `Counts`; `rowLoading` `:214` → `RowLoading` |
| `ticker.go` | `testEventMark` `:148` → `TestEventMark`; `centerText` `:221` → `CenterText`; `clipToWidth` `:247` (unexported) |
| `detail_fire.go` | `fireCount` `:244` and `fireHot` `:255` (unexported); `plural` `:273` → `Plural`; `fireBoldDefaultMW` `:285` → `FireBoldDefaultMW` |
| `alerts.go` | `dataAsOf` `:175` → `DataAsOf` |
| `status.go` | `providersOf` `:776` → `ProvidersOf` |
| `detail_seismic.go` | `seismicRowLevel` `:114` (unexported) |
| `radio_panel.go` | `levelControl` `:89` → `LevelControl` |

**Correction to TS-4:** `indentBody` (`broadcaster_detail.go:338`) is called only by
`Broadcaster.detailBody` (`broadcaster_detail.go:160`), so it moves with the console. TS's
`mapA2.py:6` placed it with the Observer's window code (`OBSWIN`), and `runA2t.py:3` then moved it
into the kit.

**The exported surface** (under P3): **25 names** and **5 methods**.
- The names: `AgoWords`, `APISummaryOf`, `ArmShimmer`, `CenterText`, `CursorSlot`, `DataAsOf`,
  `FireBoldDefaultMW`, `FocusMark`, `LevelControl`, `MastheadKeyOf`, `MastheadSlot`,
  `MastheadStamp`, `MastheadTitle`, `MemoStats`, `Plural`, `ProvidersOf`, `RowLoading`, `Sides`,
  `SurfaceMark`, `TestEventMark`, `Tick`, `TickEvery`, `TickMsg`, `Trend`, `WeatherRow`.
- The methods: `CursorSlot.Finish`, `MastheadSlot.Get`, `MemoStats.Counts`, `MemoStats.Hit`,
  `MemoStats.Miss`.

Each name has a non-test user outside the kit (`kit-CK.tsv`, last column), so M5's zero holds. 14
declarations stay unexported. `mastheadKey` is one of them: it is returned by `MastheadKeyOf` and
taken by `MastheadSlot.Get`, and no non-test file outside names it. Exporting it would make it a
name whose only outside user is a test (CK-5).

**`MemoStats`.** Today the console's memo locks and counts through the embedded struct's
unexported fields: `m.mu.Lock()`, `m.hits++` and `m.misses++` (`broadcaster_memo.go:139-147`). The
root's `bodyMemo` and `modalMemo` do the same (`memo.go:155-163`, `:505-513`). Recommended (M2),
in the prep commit:
- `type MemoStats struct { sync.Mutex; hits, misses int }`, which keeps the size at 24 B;
- `func (m *MemoStats) Hit()`, `func (m *MemoStats) Miss()`, and
  `func (m *MemoStats) Counts() (hits, misses int)`.

The alternative (M1) exports the fields `Mu`, `Hits` and `Misses`. That renames nothing else, but
it puts a mutex in an exported field. No mutant anchors the counter lines; I grepped
`06_docs/mutants` for `hits++`, `misses++` and `mu.Lock` under `modes/tty`, and none does.

**The kit's files**, by responsibility:
- `masthead.go`: the masthead, its key and slot, with `staleAfter`, `agoWords`, `apiSummaryOf`,
  `providerCounts`, `countProviders`, `dataAsOf` and `providersOf`;
- `row.go`: the location row and what it draws (`weatherRow`, `mmdd`, `trend`, `rowLoading`,
  `fireCount`, `fireHot`, `fireBoldDefaultMW`, `seismicRowLevel`, `levelControl`);
- `cursor.go`: the cursor;
- `tick.go`: the shimmer tick;
- `memo.go`: the memo counters;
- `text.go`: `centerText`, `clipToWidth`, `testEventMark` and `plural`;
- a `doc.go` with its file map and its declared inputs (M3).

Each partial file is cut into its kit file in place, in the prep commit. The move commit then
`git mv`s whole files, so R3's move-only rule holds.

### Placement under C-5: the two snapshot messages

`app` sends them: `app/pipelines.go:193` (`tty.SnapshotMsg`) and `app/pipelines.go:333`
(`tty.RecentSnapshotMsg`). Its tests use `app/m5_measure_test.go:139` and `app/mapfeed_test.go:386`.
Nothing else outside `modes/tty` names them (grep over `app`, `cmd`, `modes`, `platform`,
`domains` and `pkg`). Inside, the root reads them in `Dashboard.route`, `applySnapshot` and
`applyRecent`, and `programScoped` names them (`router.go:350-353`). The console reads them in
`Broadcaster.Update` (`broadcaster.go:502-513`).

| Option | What it does | Principle 2 | C-4 / C-5 |
|---|---|---|---|
| P1 | the kit is not internal (`modes/tty/frame`); `app` imports `frame.SnapshotMsg` | all 25 kit names become importable by the whole module. TG-6: `unused` stops seeing them | C-5 is met by the letter, because `app` does import it; 4 `app` sites change qualifier |
| P2 | the two messages get their own small importable package; the kit is internal | a two-type package is a layer package (GP-5), and A-14 gives it no natural name | 4 `app` sites change; one more package |
| **P3 (recommended)** | **the messages stay at the root.** The Router hands the console the snapshot through `WithSnapshot` and `WithRecent` (CK-2), at the same point and in the same order as today's `r.broadcaster.Update(msg)` (`router.go:568-574`) | the console's inputs are all explicit. What the Router tells it is a method (`Opening`, `SetMirror`, the snapshot); what `app` tells it is its own messages. The kit is internal | `app` is untouched, so no alias is needed. The kit goes under `internal/` |
| P4 | the messages move into the console | the Observer's main data message would live in the console's package: a misleading name (JN-5's class) | 4 `app` sites change |
| P5 | `platform/snapshot` hosts them | puts a UI message type in `platform` and changes a package outside scope | rejected |

P3's cost is a body change in the prep commit:
- `Broadcaster.Update` loses two arms (`broadcaster.go:502-513`);
- the console gains two methods;
- `Router.scopedMessage`'s `programScoped` branch picks the method by type for the two messages;
- about 10 setup lines in console-bound tests change from `Update(SnapshotMsg{…})` to the method
  (`broadcaster_loaded_test.go` 1, `broadcaster_memo_completeness_test.go` 2,
  `broadcaster_pooldata_test.go` 5, `broadcaster_pool_test.go` 2; class S).

R8 holds, since a method call replaces a message the Router already delivers directly. The change
is behaviour-equal by construction: same guard (`Snap != nil`), same command (`armTick(nil)`), same
order after `r.observer.update(msg)` and the listener-mute mirror.

**The strongest counter-argument:** D-59's symmetry, "the snapshot reaches the console too",
becomes asymmetric. The Observer gets the message and the console gets a call. And the prep commit
is not a pure rename.

### The console's messages that `app` uses

`cross-package-edges.tsv` lists 17 names. Six are types: `LineupMsg`, `StationMsg`,
`StationFaultMsg`, `BedMsg`, `BedRelaysMsg`, `StationAreaMsg`. One is a constant: `MainTrackSlots`.
Ten are fields: `Lineup`, `Power`, `Run`, `Reason`, `Relay`, `Carrying`, `Count`, `Transmitter`,
`RadiusMi`, `Pool`.

The non-test sites:
- `app/executors.go:289`, `:290`, `:291`, `:308`, `:468`;
- `app/bedrelay.go:125`, `:126`, `:180`;
- `app/pool.go:241`;
- `app/dashboard.go:416`, which fills `tty.Config.StationArea`, a field of type `StationAreaMsg`
  (`modes/tty/dashboard.go:224`);
- `app/schedule.go:142`.

The tests: `app/schedule_test.go` (11 sites), `app/pool_test.go` (4), `app/executors_test.go` (8),
`app/bedrelay_test.go` (4), `app/pool_publish_test.go` (4).

| Option | Cost |
|---|---|
| **C1 (recommended)** | **the messages move with the console to `modes/tty/console`, which stays importable** (C-5: `app` imports it). `consoleScoped` becomes the console's own predicate (CK-2). `app`'s 11 non-test sites and its test sites change qualifier in the move commit, with no alias. This is the package map's path (§2) and TS-3's shape |
| C2 | the messages stay at the root, the Router translates them into six console methods, and the console can be `modes/tty/internal/console` | the console's own inputs would be declared in another package. The Router grows a six-arm translation, and `Broadcaster.Update`'s four mutants re-anchor |

**The strongest counter-argument to C1:** `app` gains a second `modes/tty` package to import, and
the console's whole exported API becomes visible to `app`, which needs none of its methods.

**Confidence:** high on the facts and sites. The choice is the lead's (Choices 1 and 2).

---

## CK-2: the console's API (`console-api-CK.tsv`)

All in `modes/tty/console`. Types with their field lists, and signatures only.

**Construction-time values: `type Opening struct`.** Each field comes from today's `NewRouter`
(`router.go:292-315`).

| Field | Type | Source today |
|---|---|---|
| `ASCII` | `bool` | `o.cfg.ASCII` (`router.go:297`) |
| `Version` | `string` | `o.cfg.Version` (`:301`) |
| `FireBoldMW` | `float64` | `o.fireBoldMW()` (`:306`) |
| `Units` | `render.Units` | `o.units` (`:309`) |
| `Clock` | `render.Clock` | `o.clockFmt` (`:309`) |
| `Area` | `StationAreaMsg` | `o.cfg.StationArea` (`:312`). `New` sets the area generation to 1, as `b.areaGen+1` on a fresh console does today |
| `Now` | `func() time.Time` | new (K1, CK-6). `NewRouter` passes the Dashboard's own clock, `o.now` (`dashboard.go:817`; `time.Now` from `NewDashboard`, `dashboard.go:1001`) |

`func New(o Opening) Broadcaster` replaces `NewBroadcaster` (`broadcaster.go:278`) plus the seven
writes. A zero `Opening` gives today's `NewBroadcaster()` value with the area generation at 1, which
no test reads as a value. The Router is `New`'s only non-test caller.

**Per-update values: `type Mirror struct`** (48 B: three 8-byte words, a padded bool, a string).

| Field | Type | Source today |
|---|---|---|
| `Gain` | `int` | `out.observer.radioVolume` (`router.go:452`) |
| `Units` | `render.Units` | `out.observer.units` (`:453`) |
| `Clock` | `render.Clock` | `out.observer.clockFmt` (`:453`) |
| `ReduceMotion` | `bool` | `out.observer.reduceMotion`. When true, the console sets its shimmer phase to 0 (`:454-456`) |
| `StatusNote` | `string` | `out.refusal`, after the refusal is cleared or kept (`:457-461`) |

`func (b *Broadcaster) SetMirror(m Mirror)` is called once, where `:461` stands. Nothing between
`:452` and `:461` reads gain, units, clock or phase: `stationIsLive` reads the power (`:1033`), and
the bed check reads `bed.Carrying` (`:458`). So writing all five at `:461` is behaviour-equal.

**The listener mute is not in `Mirror`** (TS-3 put it there). Today it is written only before the
console handles a scoped or program message (`router.go:556`, `:571`), not on every update. Moving
it into the mirror would refresh it on every message. That is invisible in every state I found,
because `muteNotice` draws only on air (`broadcaster.go:399`), and the console cannot be on air
while the Observer is active (`toggleStation` acts only on the console, `router.go:933-935`; `canSwap`
refuses to leave a live station, `:905-925`). But that is an argument, not
construction. So the mute gets its own call at the same two places:
`func (b *Broadcaster) SetListenerMuted(muted bool)`.

**Methods the Router calls** (TS's 15, all value receivers, each renamed in the prep commit):

| Exported | Signature | Today | Router use |
|---|---|---|---|
| `Update` | `(b Broadcaster) Update(msg tea.Msg) (Broadcaster, tea.Cmd)` | same (`broadcaster.go:498`) | `router.go:484`, `:557`, `:572` |
| `View` | `(b Broadcaster) View() tea.View` | same (`:572`) | `:507` |
| `CardDetail` | `(b Broadcaster) CardDetail(handle int) (string, func(render.Opts) (string, []string), bool)` | `cardDetail` (`broadcaster_detail.go:55`) | `:1062`, `:1217`, `:1239` |
| `AlertDetail` | `(b Broadcaster) AlertDetail() (string, func(render.Opts) (string, []string), bool)` | `alertDetail` (`:82`) | `:1059`, `:1252` |
| `CardWindowGroundFor` | `(b Broadcaster) CardWindowGroundFor(handle int) string` | `cardWindowGroundFor` (`:104`) | `:1063`, `:1222`, `:1244` |
| `AlertWindowGround` | `(b Broadcaster) AlertWindowGround() string` | `alertWindowGround` (`:117`) | `:1060`, `:1253` |
| `IndexForSlot` | `(b Broadcaster) IndexForSlot(slot int) int` | `indexForSlot` (`broadcaster_slots.go:48`) | `:1139` |
| `LiveOffset` | `(b Broadcaster) LiveOffset() int` | `liveOffset` (`broadcaster_slots.go:70`) | `:447` |
| `ScrollQueue` | `(b Broadcaster) ScrollQueue(by int) Broadcaster` | `scrollQueue` (`broadcaster.go:824`) | `:718` |
| `Cursor` | `(b Broadcaster) Cursor(content string) (x, y int, ok bool)` | `consoleCursor` (`cursor.go:336`). Its parameter `frame` would shadow the kit's package name and is renamed | `:512` |
| `HeldCursor` | `(b Broadcaster) HeldCursor() *frame.CursorSlot` | `heldCursor` (`cursor.go:342`) | `:516` |
| `BedAvailable` | `(b Broadcaster) BedAvailable() bool` | `bedAvailable` (`broadcaster.go:1166`) | `:709` |
| `MainTrack` | `(b Broadcaster) MainTrack() []lineup.Card` | `mainTrack` (`broadcaster_air.go:129`) | `:1182` |
| `PoolSelection` | `(b Broadcaster) PoolSelection() int` | `poolSelection` (`broadcaster.go:858`) | `:1205` |
| `LineupSelection` | `(b Broadcaster) LineupSelection() int` | `lineupSelection` (`broadcaster.go:847`) | `:1213` |

**Fields the Router reads**, which become getters, each with a production caller:
- `Width() int` (`router.go:514`);
- `Blurred() bool` (`:516`);
- `Power() lineup.Power` (`:1033`, `stationIsLive`; it reads the Director's power and is never a
  setter, per TS-10);
- `BedCarrying() bool` (`:458`, `:982`, `:1003`);
- `Pool() []snapshot.LocationRef` (`:1206`).

**`Owns`, for today's `consoleScoped`:** `func Scoped(msg tea.Msg) bool`, which reports exactly
`LineupMsg`, `StationMsg`, `StationAreaMsg` and `BedMsg` (`router.go:366-372`), called at
`router.go:554`. I recommend the name `Scoped` over TS's `Owns`. The console owns six message
types, and `StationFaultMsg` and `BedRelaysMsg` are not in the set. `Owns(StationFaultMsg{})`
returning false would be a misleading name (JN-5's class). That they reach the console only while it
is the active surface is today's behaviour, recorded here and not changed (R1).

**The card hand-over:**
- `CardDetail`, `AlertDetail`, `CardWindowGroundFor` and `AlertWindowGround` (above), with
  `MainTrack`, `IndexForSlot` and `LiveOffset`;
- `func Manageable(c lineup.Card) bool` (today's `manageable`, `broadcaster_detail.go:172`;
  `router.go:1096`);
- `const ScheduledFrom = 2` (today's `bcScheduledFrom`, `broadcaster_lineup.go:212`). The root uses
  it at `router.go:1123`, `:1217` and `:1222`, at `request.go:104`, and in the card window's prompts
  (`broadcaster_manage.go`).

**Other names:**
- `const NoTransmitter`, today's `bcNoTransmitter` (`broadcaster_header.go:141`), read by the map at
  `map_prop_origin.go:51`;
- `MainTrackSlots`, unchanged;
- `Init` (`router.go:320`, through `surfaceView`);
- the six message types;
- `WithSnapshot(s *snapshot.Snapshot) (Broadcaster, tea.Cmd)` and
  `WithRecent(s *snapshot.Snapshot) (Broadcaster, tea.Cmd)`, under P3.

**Total exported: 40 declarations:**
- 9 types: `Broadcaster`, `Opening`, `Mirror`, and the six messages;
- 3 constants;
- 3 functions: `New`, `Scoped`, `Manageable`;
- 25 methods.

`Opening`'s 7 and `Mirror`'s 5 fields make 52 names. The messages' 10 fields are exported already.
Every one has a non-test caller in the root or in `app`. No setter exists for power or the air
(TS-10). R8 holds, because every new method replaces a direct field write or an `Update` the Router
already makes.

**What stays at the root**, though its file is named for the console:
- all of `broadcaster_manage.go`: `cardAction`, `movePrompt`, `dropPrompt`, `moveTarget`
  (`:44-100`);
- the `Dashboard` half of `broadcaster_detail.go`: `showCard` (`:285`), `sameCard`, `cardLines`
  (`:309`), `cardTitleOf` (`:318`).

These are the Observer's card window (TS-2). The prep commit cuts them into one root file named for
the card window (an A-row, for example `card_window.go`), so no `broadcaster_*` file is left at the
root.

**Confidence:** high. Every call site was read. The names are proposals.

---

## CK-3: the performance of the console seam

**Today.** `Router.Update` writes six console fields in place on the local `out`
(`router.go:452-461`). At the baseline, `-gcflags=-m` shows the one model box, `router.go:463:9:
out escapes to heap` (the `tea.Model` return, PF-1), and `Init`'s two (`:320:50`, the
`surfaceView` interface). `-S` gives `Router.Update` 832 B of code with `locals=0x6fa8`
(28,584 B).

**Measured on overlay builds** (`design-data/CK/probe/`; the repository untouched):

| Variant | Inlined | Escapes added | Allocation | Stack copy per update | `Router.Update` code / locals |
|---|---|---|---|---|---|
| baseline | — | — | — | — | 832 B / 0x6fa8 |
| v1: value `Mirrored(Mirror) Broadcaster` | yes | none (still only `out escapes to heap`) | none | **two 472-byte copies**, `out.broadcaster` to a temporary and back (the `MOVD $7, R24` loops of 7 × 64 + 24 B in `upd-v1.s`) | 1,008 B / 0x6fd8 (+48 B, the `Mirror` temporary) |
| **v2: pointer `SetMirror(Mirror)`** | yes (`can inline (*Broadcaster).probeMirror`; `b does not escape`) | none | none | **none**: the same stores as today | 896 B / 0x6fa8 (unchanged) |

**Design (recommended): v2.** It adds no allocation, no escape, no copy and no stack. Measured on
the code, not argued:
- the console beat pin (377 allocations, 122,387 B) and the idle beat pin (168, 75,988 B) run
  `Router.Update` once a message and cannot move;
- `SetListenerMuted` has the same shape;
- `Broadcaster` stays 472 B, because `Mirror` is a parameter type and is never stored;
- `Router` stays 7,136 B.

The precedent for a pointer-receiver mutator on a value model is the Dashboard's ticker methods
(`ticker.go:309`, `:400`, `:421`, `:479`).

**The strongest counter-argument:** the console then mixes receiver kinds, against Code Review
Comments. GP-9 keeps value receivers, and v1 is the house shape (`withSize`, `broadcaster.go:1643`).
v1's cost is CPU only: 944 B of stack traffic an update, against a `Router.Update` that already
copies the 7,136-byte Router seven times (seven `MOVD $111` copy loops in the baseline's `-S`). The pins would not see it, and
benchstat might not.

**The rest of the seam:**
- `Router.View` keeps drawing through concrete types: `r.broadcaster.View()` (`router.go:507`), then
  `Cursor`, `HeldCursor().Finish`, `Width()` and `Blurred()`, all on the concrete field. The
  interface stays refused (`router.go:502-505`), and `surfaceView` (`:844-852`) stays for `Init`
  alone (`:857-862`).
- The getters return one field each, so they inline like `bedAvailable` and `liveOffset` do today
  (`m-base.txt`).
- Inlining crosses packages through export data (PB-3). My probe was in one package, so the
  cross-package inlining of `SetMirror` is **UNVERIFIED** until the batch's `-m` diff. That diff
  must show `inlining call to (*Broadcaster).SetMirror` at the `router.go` call and no model escape
  beyond PB-2's three lines.
- `Opening` is about 192 B and is built once in `NewRouter`, off the frame and beat paths.
  `WithSnapshot` and `WithRecent` copy the console as `Update` does today. The message is already
  boxed, so a type switch moving from the console to the Router adds nothing.
- `Opening.Now` (K1) makes the console's clock an indirect call in production where today `nil`
  falls to a direct `time.Now()` (`broadcaster.go:281-286`). That is one call per masthead build,
  which the masthead memo spends anyway. R9 should see it: a func field the console already has,
  read on the frame path, which production would now set. The Dashboard does exactly this today
  (`dashboard.go:1001`, `body.go:99-102`).

**Confidence:** high on v1 and v2 in package. Medium on cross-package inlining until measured.

---

## CK-4: the memo guard's test-support package (D-15)

**Path: `modes/tty/internal/ttytest`** (name an A-row, CK-10). Go's rule lets a path containing
`internal` be imported only from the tree rooted at `internal`'s parent, here `modes/tty`. So both
`modes/tty`'s tests and `modes/tty/console`'s tests may import it, and so may the kit's and, later,
the map package's if it sits under `modes/tty/`. `app` and `cmd` may not.

**It must import neither `modes/tty` nor any package under it.** An in-package console test imports
`ttytest`. If `ttytest` imported the console, that would be an import cycle in the console's test
build.

**The engine as exported API** (`memo_completeness_test.go:124-289`, bodies unchanged):
- `func PerturbEach[T any](t *testing.T, base T, excuse map[string]string, writer func(string, *T), emit func(string, T))`
  is today's `perturbEach` (`:124`), renamed only.
- Unexported, unchanged: `apply` (`:138`), `walk` (`:150`), `walkAt` (`:160`), `walkNested`
  (`:193`), `fieldAtPath` (`:235`), `bump` (`:242`).
- **These stay at the root, because they are the Observer's model:** `perturbed` (`:78`),
  `perturbations` (`:89`), `dashboardWriter` (`:109`), `nestedExcuse` (`:218`), `modalName`,
  `fixtureFor`, and `TestTheNestedWalkExcusesNothingQuietly` (`:491`).

**Also in `ttytest`: the pure helpers that more than one package's tests need after the move**
(`tests-CK-refs.tsv`, the `pure test helper` rows):
- `StripANSI` (today's `stripANSITest`, `dashboard_test.go:195`), used by 21 console-bound files;
- `FirstDiffLine` (`golden_test.go:57`);
- `Card` (`broadcaster_lanes_test.go:31`, pure `platform/lineup`);
- `MastheadSnap` (`broadcaster_header_test.go:148`);
- `LoadedPoolSize` (`broadcaster_loaded_test.go:22`);
- `Main`, the `TestMain` helper (CK-7).

Fixtures that build a `Broadcaster` value (`bcWith`, `loadedConsoleAt`, `bcStandby`,
`broadcasterWithOneCard`) cannot go into `ttytest` without the cycle. They split: the pure data
(snapshots, pools, line-ups) goes into `ttytest`, and each package keeps the few lines that send it
to its own model. Choice 8 has the go-studs and build-tag exceptions.

**The check that no non-test file imports it.**
- **Where:** `scripts/lint-imports.sh`, beside its go-studs rule (`check_studs`, which has the same
  shape: "no other package's non-test file imports it").
- **What it scans:** every `.go` file in the module except `_test.go`, `third_party/`, `.claude/`
  and `dist/`, for the import path `github.com/branden-thompson/watchpost/modes/tty/internal/ttytest`.
- **How it fails:** it prints each file that imports the path and exits 1. Its control is a fixed
  floor recorded at the batch: the real tree must show at least the known test importers,
  `memo_completeness_test.go` and the console's guard, so a renamed path cannot pass vacuously
  (CQ F-18).
- **The planted specimen:** `--self-test` writes `modes/tty/console/bad.go` importing the package
  into a temporary tree and must see it refused. It writes `ok_test.go`, importing the same path,
  and must see it allowed. The specimen is seen failing before the rule lands (R2).

**P10.** `ttytest` is non-test code, so P10 judges all of it (`06_docs/code-standards.md:323`).
Row text, in the ledger's form (`06_docs/p10-ledger.md:215-220`: File, Symbol, Ratified, Reason),
**for the HUM LEAD to ratify; not self-issued**:

> | `modes/tty/internal/ttytest/perturb.go` | `package` | *(date of ratification)* | `unsafe` and `reflect` are the guard's mechanism. The F-30 frozen-frame guard (`PerturbEach`) derives every field of a surface model by `reflect`, because a hand-kept list of fields is the failure the guard exists to catch. It writes the unexported ones through `reflect.NewAt(…, unsafe.Pointer(v.UnsafeAddr()))` (`fieldAtPath`, `walkNested`), because a model's fields are unexported and the copy it perturbs is addressable. Test-time only: no non-test file imports the package (`make lint-imports`, with a planted specimen), so the binary never holds it. Moved unchanged from `modes/tty/memo_completeness_test.go:124-289` (0.19.6 D-15). Removing either import removes the guard, and with it the coverage of both surfaces' memo keys. RATIFIED by the HUM LEAD *(date, verbatim)*. |

**Not covered by D-15 (Choice 7):**
- **P10-01, recursion.** `walkAt` calls itself at `memo_completeness_test.go:167` (an embedded
  struct). `bump` calls itself at `:282` and `:285` (a pointer's pointee). The checker finds
  recursion by bare name (`code-standards.md:329`, the P10-01 row), so the unchanged engine raises
  two P10-01 findings. The precedent is the structural, type-bounded recursion of
  `pkg/schema/schema.go:typeSchemaDepth` (ledger P10-01 row). Both are bounded by the type's
  nesting depth. The alternative, bounded loops, changes the body, which D-15 ruled out.
- **P10-05, density.** `ttytest` falls under `modes/tty`'s density row by prefix
  (`scripts/quality/p10-unmatched.sh:113-115`; `p10-ledger.md:129`, floor 0.42). The engine carries
  no `invariant.Check`, so it falls below that floor. D-10 brings it to the HUM LEAD when found and
  ratifies its own floor at BUILD exit.

**Where the guards land, and what each calls:**
- **The Observer's guard** stays at `modes/tty/memo_completeness_test.go`
  (`TestTheMemoKeyCoversEverythingTheFrameShows`, `:37`). It calls `ttytest.PerturbEach` through
  `perturbations`. `body_completeness_test.go:49` and `tape_memo_test.go:90` keep calling
  `perturbations`.
- **The console's guard** moves whole to `modes/tty/console/broadcaster_memo_completeness_test.go`,
  with `consoleWriter`, `consoleExcuse` and `sameSpan`. It calls `ttytest.PerturbEach`, and calls
  `buildSpans` and `consoleKeyFor` in-package (`:85-93`). `projection_cache_test.go:81` moves with
  it.
- **The masthead guard** (`masthead_memo_test.go:53`) walks both surfaces in one test, so it splits
  by surface:
  - **root:** the Observer half calls `perturbations`, `d.header` and `frame.MastheadKeyOf`. It
    holds the key without naming the kit's unexported type, comparing with `==`, which reads the
    same value;
  - **console:** the console half calls `ttytest.PerturbEach` with `consoleExcuse` and
    `consoleWriter`, `b.header` and `frame.MastheadKeyOf`, in-package;
  - **the named inputs** (`mastheadInputs`, `:21-48`) have no clean home. One input reads the
    kit-internal `staleAfter` (`:24`). Another calls go-studs' colour switch, which
    `check_studs` refuses in a non-test `ttytest` file (`scripts/lint-imports.sh`, `check_studs`).
    Recommended: a kit-level test in `frame`'s own tests applies all six inputs to `MastheadTitle`,
    `MastheadStamp` and `MastheadKeyOf` for both editions, where `staleAfter` and go-studs are in
    scope; each surface half keeps its model walk. It is one of the 20 references in CK-5 and in
    Choice 6, because the six inputs' subject moves from each surface's header to the kit's
    masthead.

**Confidence:** high on the engine, the path rule and the check. High that P10-01 fires (bare-name
rule). Medium on the masthead split, which is a ruling.

---

## CK-5: tests under D-14 (`tests-CK.tsv`, `tests-CK-refs.tsv`)

**How it was counted.** Each test-file declaration lands where it must compile:
- the root, if it reaches any root symbol (a `Router`, a `Dashboard`, a root helper);
- otherwise the console, if it reaches a console symbol;
- otherwise the kit.

Helpers follow their users to a fixed point. Each unexported reference from a declaration to
another package is one row, classed by its line (`fn-sites.txt`).

TS's 40 files and 114 references (Map I) used "home = the package a file reaches most". That
undercounts files such as `router_test.go` (home "console", while its real home is the root) and
`broadcaster_uat_test.go`. Whole files, landed by my rule, give 41 files and 145 references
(`tests_ck.py`). At function level:

- **Where the files land:** 24 move whole to the console; 12 split between the console and the
  root; 1 splits between the console and the kit (`clip_ansi_test.go`); 2 split between the kit and
  the root, one of them only nominally (below); 23 stay at the root, giving 62 files with any
  crossing.
  - `dashboard_test.go`'s `runCmd` is counted as kit-bound only because it names `tickMsg`; it stays
    at the root with its users.
  - `masthead_memo_test.go`'s named inputs are kit-bound (CK-4).
- **127 unexported crossing references:**
  - **S**, setup only: 73;
  - **A**, the same value through production API: 23;
  - **M**, moves with the code: 12, the masthead guard's console half;
  - **X**, cannot be served without a test-only export: **19**.
- **61 helper rows:** 14 console fixtures that root tests call, rebuilt through `console.New` and
  messages (S); 1 `consoleWriter` (M); 45 pure helpers to `ttytest` (M); and **1 X, the masthead
  guard's named inputs** (CK-4).

**Class X, exact: 20 references, 16 test functions, 10 files.** These are the ruling D-14 owes. "V"
means the value is drawn in the console's frame, so a production observable exists, but it is not
the same value.

| File | Test | Reference (site) | Why |
|---|---|---|---|
| `router_test.go` | `TestSizeReachesTheINACTIVESurface` | `.height` (`:67`, `:70`) | no production reader. V: `clamp` sizes the frame to it. `.width` on the same line is A, `Width()` |
| `router_test.go` | `TestBackgroundColourReachesTheINACTIVESurface` | `.darkBG` (`:77`) | V: `View().BackgroundColor` is `render.WindowBG(darkBG)` (`broadcaster.go:579`) |
| `router_test.go` | `TestTheConsoleInheritsTheASCIIDecision` | `.ascii` (`:95`) | no production reader |
| `router_bedplay_test.go` | `TestSpacePlaysTheBedRelayAndNeverOverTheProgramme` | `.statusNote` (`:35`, `:40`) | V: the station row. Reading the Router's own `refusal` would stop pinning the mirror step (`router.go:461`) |
| `router_bedplay_test.go` | `TestStandbyClearsTheBedRefusal` | `.statusNote` (`:68`) | same |
| `router_diagnostics_test.go` | `TestGainPressedOnTheConsoleMovesObserversOwnLevel` | `.gain` (`:270`, `:272`, `:280`, `:282`) | V: the gain control |
| `router_diagnostics_test.go` | `TestGainDownReachesTheSameLevel` | `.gain` (`:299`, `:300`) | V: the gain control |
| `router_diagnostics_test.go` | `TestTheWindowsStackOnTheFrameRatherThanBesideIt` | `.height` (`:124`, `:126`) | V |
| `router_diagnostics_test.go` | `TestEveryAdvertisedControlIsBound` | `.controlRow`, `.opts` (`:316`) | console-internal. V: the control row is in the frame |
| `broadcaster_fault_test.go` | `TestAMutedListenerIsShownWhileTheStationIsOnAir` | `.listenerMuted` read (`:59`) | V: the band says so on air. The writes at `:48` and `:52` are S, through `SetListenerMuted` |
| `motion_test.go` | `TestTheConsoleFollowsReduceMotion` | `.frame` (`:159`) | V: the shimmer phase |
| `broadcaster_detail_test.go` | `TestTheAlertBoxsWayInOpensThePriorityCard` | `cardTitle`, `.detailBody` (`:296`) | the expected window is computed from console internals (mutant mAU2's guard) |
| `broadcaster_tone_test.go` | `TestTheAlertWindowWearsTheCardsGround` | `cardTone` (`:199`) | console-internal |
| `broadcaster_pooldata_test.go` | `TestTheConsoleReadsTheOperatorsFireThreshold` | `.poolRows`, `.locIndex` (`:277`); `fireHot` (`:286`) | console-internal (V: pool rows) and kit-internal |
| `detail_fire_test.go` | `TestTheFireDetailListsEveryNamedIncident` | `fireCount` (`:138`) | kit-internal |
| `masthead_memo_test.go` | the named inputs (`:21-48`) | `staleAfter` (`:24`) | kit-internal. See CK-4 |

That is 11 V references and 9 others. If K1 (CK-6) is refused, three more join:
`broadcaster_detail_test.go:160`, `console_units_test.go:60` and `router_trace_test.go:55`, which
set the console's clock. That makes **23 in 19 functions in 12 files**.

**Every router test that reads or writes a console field**, a row each in `tests-CK-refs.tsv`:

| Site | Field | Class |
|---|---|---|
| `router_test.go:67` | `.width` | A: `Width()` |
| `router_test.go:67`, `:70` | `.height` | X |
| `router_test.go:77` | `.darkBG` | X |
| `router_test.go:95` | `.ascii` | X |
| `router_test.go:109` | `.power` | A: `Power()` |
| `router_bedplay_test.go:35`, `:40`, `:68` | `.statusNote` | X |
| `router_diagnostics_test.go:30`, `:244` | `.width`, `.height` | S: `Update(tea.WindowSizeMsg)` |
| `router_diagnostics_test.go:124` | `.height` | X |
| `router_diagnostics_test.go:198`, `:206` | `.width` | A: `Width()` |
| `router_diagnostics_test.go:270`–`:300` | `.gain` | X |
| `router_diagnostics_test.go:315` | `.width`, `.height`, `.ascii` | S |
| `router_diagnostics_test.go:316` | `.controlRow`, `.opts` | X |
| `router_refusal_test.go:31` | `.width`, `.height` | S |
| `router_refusal_test.go:88` | `.power` write | S: `Update(StationMsg)` |
| `router_trace_test.go:55` | `.now` | S under K1 |
| `request_slot_test.go:62`, `:187` | `.power` writes | S |
| `scroll_keys_test.go:16`–`:22` | `.selected` | A: `LineupSelection()`. The same value while the pointer is in the running order, which the test's 0, 1, 0 are (`broadcaster.go:847-852`) |
| `motion_test.go:159` | `.frame` | X |
| `cursor_test.go:139` | `.opts().Glyphs()` | A: `render.Opts{ASCII}.Glyphs`, the ASCII choice the test made |
| `console_units_test.go:60` | `.now` | S under K1 |
| `broadcaster_fault_test.go:59` | `.listenerMuted` | X |
| `broadcaster_scroll_test.go:173`, `:194`, `:213`, `:256`, `:270` | `.selected` writes | S: `ScrollQueue` |
| `broadcaster_scroll_test.go:184` | `.cardDetail` | A: `CardDetail` |
| `broadcaster_manage_test.go:23` | `.selected` | S |
| `broadcaster_manage_test.go:118` | `.liveOffset()` | A: `LiveOffset` |
| `broadcaster_manage_test.go:241` | `.lineup` | S: the test keeps the line-up it sent |

**Setup changes that the pins depend on**, flagged because a rebuilt fixture is what a pin measures:
- `beat_alloc_test.go`'s `beats` builds the console-beat pin's console through `loadedConsole`, a
  console test helper. It is rebuilt through `console.New` and messages (S), and the pin
  comparator must read 377 and 122,387 B unchanged.
- `TestConsoleFrameAllocBudget` (`broadcaster_alloc_test.go`) and `BenchmarkConsoleFrame*`
  (`bench_test.go`) move to the console. Their package path changes, so the pin comparator and
  benchstat must map the old key to the new one (PB, TG).

**Confidence:** high on the counts, which are derived. Medium on the S/A boundary for single
lines, and the table names every one.

---

## CK-6: the router trace

**Today.**
- It prints `%T%v` (`router_trace_test.go:62`), and its golden has 900 lines, 38 of them
  `tty.tickMsg{}`.
- It sets `d.now` (`:53`, a root field) and `r.broadcaster.now` (`:55`, a console field only tests
  write: `broadcaster.go:139-143`; `broadcaster.go:282` is its only reader).
- It records `cmd != nil`.

**The new form**, written in batch 0, recorded on code identical to `e57afeaf`, byte-identical
after every move:

1. **Key:** `<seed>.<step> m<index> surface=<n> cmd=<description> frame=<sha256>`.
   - `<index>` is the message's position in `traceMessages` (`:18-34`), two digits. Nothing prints
     a Go type.
   - Batch 0 widens the alphabet once, before recording (RX-8, A-12). After that it is never
     reordered or extended; adding a message changes every draw. An entry's spelling may change
     with a move (`tickMsg{}` becomes `frame.TickMsg{}`) and its index does not.
   - The "GO" normalisation of About stays (`:50`, `:63`).
2. **Clock, through production API:** `NewRouter`. The trace pins the Dashboard's clock (`:53`) and
   builds the Router with `NewRouter(d)`. Under K1, `NewRouter` hands that clock to the console
   (`Opening.Now`, CK-2), so `:55` is deleted. Both surfaces read `at`, as now.
   - On `e57afeaf` no production API sets the console's clock. The batch-0 recording therefore keeps
     `:55`.
   - The console batch's prep commit lands K1 and deletes `:55`, a setup change, and the golden
     must not move. That proves K1 changed nothing the trace sees.
   - **If K1 is refused**, no production API exists. `:55` is then a reference that cannot be
     served without a test-only export (D-14's ruling, counted in CK-5), and the trace cannot cross
     the console's boundary.
3. **A stable command description**, derived without running any command but Bubble Tea's own pure
   wrapper. The name is `runtime.FuncForPC` of the command, verified on `bubbletea/v2@v2.0.3`:
   - `-` for nil;
   - `tick` for `charm.land/bubbletea/v2.Tick.func1`;
   - `every` for `charm.land/bubbletea/v2.Every.func1`;
   - for `charm.land/bubbletea/v2.compactCmds[...].func1`, which `Batch` and `Sequence` share,
     the trace calls the closure. It only returns its slice, as `tea.BatchMsg` or `tea.sequenceMsg`.
     The trace then writes `batch(d1,d2,…)` or `seq(…)`, recursing to a fixed bound;
   - `fn` for any other function.

   A watchpost closure is not named, because its runtime name carries the package path and the
   receiver. The kit move changes the first. 2a re-homes 381 methods' receivers, which changes the
   second. Every watchpost tick goes through `tea.Tick` (`tickEvery`, `dashboard.go:1059-1061`), so
   ticks read `tick` before and after any move.

**Confidence:** high on the form and the names (printed by `design-data/CK/cmdname`). The
widened alphabet is batch 0's, and the map's messages after 2b are MS's (Touches).

---

## CK-7: `TestMain` invariants that travel

**Today:** `modes/tty/main_test.go:20-22`. `TestMain` sets `time.Local = time.UTC` (`:21`) before
any test, then exits with `m.Run()`'s code (`:22`).

`TestNoTestSetsTheZoneMidRun` (`:30`) holds the rule by reading `os.ReadDir(".")` (`:34`). It goes
quiet for tests that leave `.`, which is TS-12's class.

`modes/tty` does not set the config or state directories. That is `app`'s invariant
(`app/platform_sim_test.go:26-49`, AP's lane).

**The helper:** `func Main(m *testing.M)` in `ttytest`. It pins `time.Local` to UTC, runs the
tests, and exits with their code, today's three lines, unchanged in effect. Every package under
`modes/tty/` that has tests declares `func TestMain(m *testing.M)`, whose only statement calls
`ttytest.Main(m)`:
- the root;
- `console`;
- `frame`, which gains tests (`clip_ansi_test.go`'s `TestClippingCountsCellsAndNotEscapeBytes` and
  the masthead inputs);
- `ttytest` itself, if it has tests.

**The check:** a test in `cmd/watchpost`, beside `gates_test.go`, named for example
`TestEveryPackageUnderTheTwoTreesCarriesItsParentsTestMain`.
- It walks `modes/tty/` and `app/` for directories with `_test.go` files and parses them
  (`go/parser`).
- For a package under `modes/tty/`, it requires a `TestMain` whose body calls `ttytest.Main`. For
  `app/`, it requires AP's helper.
- It fails by naming each package that lacks one.
- Its floor is fixed at batch 0: at least 2 packages, `modes/tty` and `app`.
- Its planted specimen is a `testdata` tree holding one package with a test and no `TestMain`,
  which the walker must report.

`TestNoTestSetsTheZoneMidRun` walks the `modes/tty` tree, not `.`. It skips `ttytest`'s
non-test files, since it reads `_test.go` files only, and has a fixed file-count floor and a planted
control.

**Confidence:** high.

---

## CK-8: mutants (`mutants-CK.tsv`)

**The 80 that anchor in `broadcaster*.go`:**
- **76 move to the console** with their declarations: a path re-point in the move commit, of which
  66 keep their text.
- **1 moves with its test:** `mAN2`, whose target is `broadcaster_alloc_test.go` in
  `TestConsoleFrameAllocBudget`. Its path changes with the test.
- **3 stay at the root**, in the Observer's card window:
  - `mW6` (`Dashboard.showCard`, `broadcaster_detail.go`) changes path in the prep commit, when
    that half is cut into the root's card-window file;
  - `m695` (`Dashboard.movePrompt`) and `mAL2` (`Dashboard.moveTarget`, `broadcaster_manage.go`)
    change path only if the card-window rename A-row is taken.

**Outside `broadcaster*.go`:** 12 kit anchors move with the kit (path):
- 7 in `cursor.go`: `m635`, `m636`, `m643`, `m644`, `m645`, `m649`, `m654`;
- 4 in `masthead.go`: `m744`–`m747`;
- 1 in `body.go`: `mAG1` (`weatherRow`).

**Paths changed in step 1: 90**, or 92 with the rename. That is 77 to the console, 12 to the kit,
and 1 (or 3) at the root.

**Anchors whose text changes: 26** (string literals only, read with Python's tokenizer; three name
hits were false and are excluded: `m168`, `m717`, `mRQ1`).
- **12 name a kit symbol renamed on export:**
  - `m293` (`tickEvery`), `m425` (`plural`), `m636` (`sides`), `m744` (`agoWords`);
  - `mAD1`, `mAD2` (`testEventMark`);
  - `mAD4`, `mCE1`, `mCK2` (`centerText`);
  - `mAF1` (`rowLoading`, `weatherRow`), `mAN1` (`weatherRow`), `mAQ2` (`fireBoldDefaultMW`).
- **14 name a console symbol** renamed, or a Router field read that becomes a getter or the mirror:
  - `m241` (`bcNoTransmitter`), `mAL2` (`bcScheduledFrom`);
  - `m641`, `m642`, `m664`, `mAB4`, `mAG2`, `mAM1`, `mAU1`, `mAV3`, `mRQ2`, `mW2` (`router.go`);
  - `mAK3` (`bedAvailable`'s declaration), `mP6` (`indexForSlot`).
- **By commit:** 12 change in the prep commit alone, a rename inside one package. **14 also gain a
  package qualifier in the move commit**: the 8 console anchors that name a kit symbol, `m293`,
  `m425`, `m241`, `mAL2`, `mAB4`, and `m664` (`console.Mirror`). R3's move-commit list allows
  "mutants re-pointed by path". These 14 need "and the same qualifier edits the code receives",
  proved by A-11's old-and-new anchor table with qualifiers normalised. That is a ruling or an A-row
  for the lead (Choice 10).
- **None names `tty.`** that step 1 changes. Only `m187` (`app/endpoints.go`,
  `tty.FeatureAdverse`) and `mJ4` (`app/severe_read.go`, `tty.SevereMsg`) contain `tty.`, and both
  are root names. Under P3 no `app` anchor names a moved message; under C1 none names a console
  message.

**The P10 rows this batch re-keys** (D-10's table):
- `modes/tty/broadcaster_rail.go | package` (`p10-ledger.md:190`) goes to
  `modes/tty/console/broadcaster_rail.go`;
- the eight `router.go` and `view.go` P10-01 rows (`:49-56`) keep their file. Their proof says the
  only `surfaceView` implementers are `Broadcaster` and `Dashboard`, so it must be re-proved with
  the console in another package (D-10's re-proved set).

**Confidence:** high.

---

## CK-9: M3, the console's reach into the Observer's model

**Today: 13 symbols** (JN-4), from `file-edges.tsv`, broadcaster files to `dashboard.go`:
- `broadcaster.go` reads `.Snap`, `RecentSnapshotMsg`, `SnapshotMsg` and `tickMsg`;
- `broadcaster_detail.go` reads `Dashboard`, `.cardGen`, `.cardGround`, `.cardID` and `.cardRows`;
- `broadcaster_manage.go` reads `.cardErr` and `.cardMoveTo`;
- `broadcaster_slots.go` reads `armShimmer` and `tick`.

Seven of the 13 are fields.

**After the move: 0.** The console cannot import `modes/tty`, because the root imports the console.
Of the 13:
- 7 were never the console's: the card window's `Dashboard` code that stays at the root (CK-2);
- `tickMsg`, `tick` and `armShimmer` are the kit's;
- the snapshot arrives as a parameter (P3).

The console reads:
- the kit's exported API: 24 symbols today, the 28 of TS-3 less the two messages and their `.Snap`;
- `Opening` (7 values, at construction);
- `Mirror` (5, per update);
- `SetListenerMuted`;
- `WithSnapshot` and `WithRecent`.

The console's `doc.go` lists these as its declared inputs (M3). The committed graph tool should
show console→root = 0, and the compiler holds it.

**The kit's users after the move:** the root, with 29 symbols (TS's 33, less the two messages and
their fields), in `body.go`, `cursor.go`, `dashboard.go`, `detail.go`, `detail_fire.go`, `memo.go`,
`radio_panel.go`, `router.go`, `severe.go`, `status.go`, `status_sources.go`, `ticker.go`,
`view.go`, and the map files `map_boxes.go`, `map_prefs.go`, `map_prop_ahead.go` and
`map_prop_update.go`; and the console, with 24. `app` and `cmd` do not use it.

**Confidence:** high.

---

## CK-10: names (A-14)

| Package | Candidates | Recommendation |
|---|---|---|
| kit | **`frame`** (TS); `masthead`; `surfaces` | **`modes/tty/internal/frame`**: "the parts of a frame both surfaces draw". "A frame is built" is the record's word (`docs/where-things-happen.md:37`), it shadows no imported standard-library package, and it is not a keyword. `masthead` is too narrow, since the kit also holds the cursor, the tick and the row. `surfaces` is a plural package name and sits beside the type `Surface`. **Counter-argument:** "frame" is overloaded. It also means the shimmer phase (`Broadcaster.frame`, `render.Opts.Frame`) and the whole screen. One parameter would shadow the package, `consoleCursor(frame string)` (`cursor.go:336`), renamed in the prep commit (grep: no other function both declares `frame` and calls a kit symbol) |
| console | `console` | **confirmed: `modes/tty/console`.** The vocabulary row is "the Broadcaster, the console" (`docs/where-things-happen.md:129`), and the code says "the operator console's model" (`broadcaster.go:126`). That row's pointer, `broadcaster.go:NewBroadcaster`, becomes `console/broadcaster.go:New` in the batch |
| test support | **`ttytest`**; `memoguard`; `testsupport` | **`modes/tty/internal/ttytest`**: Go's convention for a package's test helpers (`httptest`, `fstest`), and it holds more than the memo guard (`Main`, the pure helpers). `memoguard` names one of its jobs. `testsupport` is a mechanism name, against A-14 |

Each gets a vocabulary row in `docs/where-things-happen.md`.

---

## Choices for the lead

1. **The snapshot messages (CK-1):** P1, P2, **P3 (recommended)**, P4 or P5. **Counter-argument:**
   D-59's symmetry, and a prep commit that is not a pure rename.
2. **The console's messages (CK-1):** **C1, they move with the console, which stays importable
   (recommended)**, or C2, they stay at the root and the console goes internal. **Counter-argument
   to C1:** `app` imports a second `modes/tty` package and sees the console's API.
3. **The mirror's form (CK-3):** **a pointer `SetMirror`, measured with no copy and no added stack
   (recommended)**, or a value `Mirrored`, no allocation but two 472-byte copies an update. The
   listener mute goes **outside `Mirror`, at its two places (recommended)**, or inside it.
   **Counter-argument:** mixed receiver kinds (GP-9), and TS's narrower shape.
4. **The console's clock (CK-6):** **K1, `Opening.Now` inherited from the Dashboard in `NewRouter`
   (recommended)**, or K2, no production API, three references for the D-14 ruling, and the trace
   cannot cross the boundary. **Counter-argument:** K1 sets a func field in production that today
   only tests set, so it is a test seam served through production code. R9 should rule on it.
5. **`MemoStats` (CK-1):** **M2, an embedded mutex with `Hit`, `Miss` and `Counts`
   (recommended)**, or M1, exported fields. **Counter-argument:** M2 edits six counter lines in
   three memos in the prep commit.
6. **D-14's ruling (CK-5): 20 references in 16 test functions in 10 files** (23 in 19 in 12 without
   K1).
   - (a) For the 11 V references, read the value from the console's drawn frame through `View()`.
     The value differs, so it is a ruling.
   - (b) For the rest, restate each expected value through production API where one exists, or keep
     the test at a weaker subject.
   - (c) None may use a test-only export.

   Recommended: (a) for V, and for the masthead inputs the kit-level test (CK-4). The other 8
   (`cardTitle`/`detailBody`, `cardTone`, `poolRows`/`locIndex`/`fireHot`, `fireCount`, `.ascii`)
   come back singly with their killers. **Counter-argument:** a frame-read assertion is weaker
   where no mutant sits (D-14's own counter).
7. **`ttytest` and P10 (CK-4):** D-15 ratified P10-09 only. The unchanged engine also raises P10-01
   (`walkAt` `:167`; `bump` `:282`, `:285`) and P10-05 (density under `modes/tty`'s 0.42 floor).
   Options: (a) ratify two P10-01 rows, type-depth-bounded recursion with `pkg/schema`'s precedent
   **(recommended)**; (b) rewrite as bounded loops, against D-15's "unchanged in body". The density
   floor goes to BUILD exit (D-10). **Counter-argument to (a):** the first `unsafe` exemption arrives
   with two recursion exemptions beside it.
8. **What else goes into `ttytest` (CK-4):**
   - The race flag `raceEnabled` would need two `//go:build` files in non-test code, which P10-08
     lists and which need two ratified rows. **Recommended:** each package keeps its own two-line
     `_test.go` pair, a copy of a build-tag fact that cannot drift.
   - The golden helper's `-update-golden` flag would be a package variable (P10-06). **Recommended:**
     `CheckGolden` takes the update choice as a parameter, and each package's tests declare the
     flag.
   - **Counter-argument:** RX-5's "a copied helper is not allowed".
9. **The card window's root file (CK-2):** cut the `Dashboard` half of `broadcaster_detail.go` and
   all of `broadcaster_manage.go` into one root file named for the card window (an A-row), moving 3
   mutants' paths, or leave `broadcaster_manage.go` at the root under a console name (TS-2).
10. **R3 and the anchors (CK-8):** 14 anchors gain a package qualifier in the move commit. Either
    R3's list reads "mutants re-pointed by path, and the qualifiers the code receives", held by
    A-11's table, or those 14 are re-anchored on qualifier-free text in the prep commit.
11. **Names (CK-10):** `frame`, `console` and `ttytest`, as A-rows.

## Touches other agents

- **AP:**
  - Under C1, `app` imports `modes/tty/console` for six messages and `MainTrackSlots`, at the 11
    non-test sites in CK-1, as qualifier edits in the move commit.
  - Under P3, `app/pipelines.go:193` and `:333` do not change.
  - `app`'s `TestMain` helper (the XDG directories and the GOOS seam, H-6) is AP's. CK-7's check
    spans both trees.
- **MS:**
  - `map_prop_origin.go:51` reads `console.NoTransmitter`. After 2b the map package imports the
    console for one constant, unless MS moves it to the kit.
  - The map files use six kit names (`Plural`, `TickEvery`, `Trend` and others), so the map package
    must sit under `modes/tty/` to import `internal/frame`.
  - The trace's alphabet must stay constructible from the root after 2b, so no unexported map
    message can be in it. The trace names no closure, because 2a re-homes receivers.
  - TS-9's depth rule applies to `ttytest`'s unchanged walker, which goes one level down
    (`memo_completeness_test.go:179-189`).
- **TG / batch 0:**
  - `TestEveryWindowReplyIsRoutedBackToTheWindow` excuses `"tickMsg"` by name
    (`router_lookup_test.go:134-137`). After the kit move that type leaves `.` and becomes
    `frame.TickMsg`, and the excuse goes stale without failing. The tree-wide version (TG-6) must
    fail on a stale excuse and must judge the console's six exported messages.
  - The pin comparator and benchstat keys move with `TestConsoleFrameAllocBudget` and
    `BenchmarkConsoleFrame*` (CK-5).
- **KB:** M3's console figure is 0 reads of the model after the move (CK-9). The M4 tasks are not
  touched.

## What I could not check

- Cross-package inlining of `SetMirror`, `SetListenerMuted` and the getters (UNVERIFIED). The probe
  was in one package, and the batch's `-m` diff settles it.
- That each S row's rebuilt fixture gives byte-identical state, `beats` and the console's goldens
  above all. The pin comparator, the goldens and the scoped sweep settle it at the batch.
- Whether the P10 checker reads generic functions' recursion, here `walkAt[T]`, by bare name. The
  rule says it does (`code-standards.md:329`). A-7's pilot would show it.
