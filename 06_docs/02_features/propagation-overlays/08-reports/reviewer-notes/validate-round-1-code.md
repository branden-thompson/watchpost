# VALIDATE exit red team (D-338): the code reviewer's report, verbatim

Brief: kept with the evidence outside the tree (`red-team-validate-r1/prompt-code.txt`). Dispositions: `../red-team-validate.md`.

# VALIDATE exit red team — CODE reviewer (VC)

**Scope reviewed:** watchpost `306fe5ce..3df12963` (tree at ecfd616c, clean) and go-ionomaps `d71cf55..37b7d42` (tree at 01f5dbb, a README-only commit past it; the rc.17 tag resolves to 37b7d42 and the module cache matches it file-for-file). All work read-only; mutants and probes via `go test -overlay` from `<scratch>/rt-v1c`.

## 1. Correct

**httpx (batch 77) — matches D-324/A-101.** `noteFailure` excuses only `ctx.Err()`, so a client timeout now arms the memo 20 s (`memo.go:145`). Each host takes a share (cap/2: 8/16, 4/8, 4/8) before the lane slot, released on success and on a wait that ends (`httpx.go:444-461`); 16 hosts own a share, the 17th+ share `OtherHost`. No deadlock: a slot holder already holds its share and waits on nothing else.

**Chart (batch 78)** — clean; `propAgreementLine` drawn under the shade legend, fits the prose width.

**Refresh/STALE (batch 79)** — Hourly is the default; unknown/"hourly" file values read as Hourly (Propagation is new in 0.19.0, so no shipped user loses a setting). `staleAfter` = 1 h + wait: 2 h / 1 h 10 min / 1 h. Clean apart from VC-3.

**First run (batch 80)** — `Mutate` clears FirstRun (`config.go:527`); `Default()` has no Locations, so the first file written holds no fake place. `applyCommitted` closes only the Setup window. The `armPropChart` settle gate is sound (the chart window holds the keys). One gap:

**VC-1 — first-run esc with a mistyped FIRMS key loses the chosen location** (`modes/tty/setup.go:249-253`, `setup.go:563`, `dashboard.go:1860`). On a first run esc passes the typed key-row text to `setupSave` (unlike normal esc, which discards typed fields). `applySetup` rejects a key that is not 32 hex (`firms.CheckKey`); the error path reopens Setup with `d.setup` already reset at the esc. Probe (overlay test), confirmed: pick Oceanside, enter (focus → key row), type "abc123", esc → Setup reopens "setup failed: …32 hex…", `ref` nil, typed key gone; a second esc → `def` nil, location dropped — the D-336 defect exactly. Recoverable (fix key → enter → "choose your default location first" → re-pick). Severity: **Minor** (needs first run + mistyped key + esc; the listener is told truthfully, but the pick vanishes from the reopened window). Action: on the first-run esc, send the key only if `CheckKey` passes, or keep the pick for the error reopen; add a test for this path.

**go-ionomaps rc.17** — clean. `ClimatologyShare`: 1 inside, linear to 0 over 4°, antimeridian and NaN handled; the blend applies only where GloTEC gives a value; the live offset counts only stations at share 0 (`update.go:523`); the no-readings correction is scaled by GloTEC's share (`update.go:561`). Station coordinates are geographic degrees, so the box test is right. Darwin and Hawaii are outside; Okinawa and Yamagawa sit in the edge band (blended, left out of the offset) — acceptable.

**VC-2 — requests already waiting on a hung host still try it once each** (`httpx.go:730`, `734`, `745`). The memo is read before `acquire` and again only on retries; requests queued on the share before the memo armed each make their first attempt and wait the full timeout. Probe: 40 concurrent asks of a hung host, 300 ms timeout → all 40 reached it in 5 waves of 8, none failed fast. In D-324's scenario (46 waits, 30 s timeout) that is ~3 min of FIRMS holding half the normal lane and the fire rows waiting it — other hosts unaffected (batch 77's measured 20 s). Severity: **Minor** (stated limit; D-324's "avoided as one that refuses" holds for new requests, not those already queued). Action: re-read the memo after `acquire` (the existing check).

**VC-3 — Hourly's STALE bound leaves ~5 min margin.** The live log (`stale-hourly-watch-2026-10-29.log`) shows the oldest input at 55 min when a refresh lands; go-ionomaps keeps readings up to 1 h old (`hybrid.For`). Just before the next refresh: age = (oldest reading's age at build) + 60 min + fetch time against a bound of 120 min — a reading ~59 min old at build shows STALE briefly before the next refresh. **Minor.** Action: run D-332's pre-SHIP live check on the shipped 2 h bound (the existing log is from the old 1 h bound) and record it.

**VC-5 — the interactive-lane share (4 for api.weather.gov) is below the zone fetcher's 6** (`httpx.go:101`, `domains/weather/nws/zones/zones.go:39`): an uncached map open runs at 2/3 of its former zone throughput. Zones are disk-cached, so rarely met. **Minor** (stated limit, A-101 as ruled). Action: none beyond noting it.

## 2. Concurrency and state

`-race` clean: watchpost `platform/httpx`, `platform/config`, `modes/tty` (225 s); go-ionomaps `.` (241 s), `internal/background`, `internal/assimilate`. **watchpost `./app` under `-race` had not completed when handing back (log empty) — no result claimed; should be re-run.** No deadlock or lost share; share map bounded at 17. Config writes correct apart from VC-1's error path (after the second esc there the reopened window had empty state; the defaults it wrote — dwell 5m, lang en — were harmless on a first run).

## 3. Tests

All mutants killed by their named tests: watchpost m582–m596 (each applied on a copy and run through overlay) and go-ionomaps m148–m154 (applied from `mutants.json`). `GOARCH=amd64`: watchpost `platform/httpx`, `platform/config` and the new/changed `modes/tty` tests pass; go-ionomaps `.`, `internal/background`, `internal/assimilate` pass. Gap: no test for VC-1's error path.

## 4. Measured claims

A-101's half lane (8/4/4) and 20 s memo (`memoTransportTTL`) match the code. D-334's box (0–30°N, 130°E–180°) and 4° edge match.

**VC-4 (Minor):** the fresh-days figures (Pacific 1.23 against 1.22, all 0.80) are from gm3 at 37b7d42 (`gm3/fresh-shipped-37b7d42.md`). The 10-minute-day (1.39, 0.91) and PLAN-week (1.28, 0.84) figures come only from `pacific-probe-2026-10-29_test.go.txt`, whose box was hard-edged (no 4° blend); go-ionomaps D-124 (`rulings.md:145`) presents all three as the build's. Action: re-score those two days with `gm3-37b7d42`, or label them as the probe's.

## Verdict

No Important. VC-1 is the one to fix — cheap, and D-336's own path. **exit VALIDATE**, with VC-1..VC-5 as Minors (and the `app` race run to complete).
