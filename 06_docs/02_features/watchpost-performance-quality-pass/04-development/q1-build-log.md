# Q1 build log — Defect, hygiene, cache retention, network resilience (`v0.9.5`)

Written for someone who has never seen the code (plan §0.5). Q1 is the first batch that changes
behaviour: one functional defect fixed, two growth terms closed, and the retry path made polite under
failure. Every change is test-first; the fault pins from plan §2.3 are named in §3.

| Field | Value |
|---|---|
| Batch | Q1 (plan §3) · branch `feature/watchpost-performance-quality-pass` · commits `bdf65c7`, `2f3e919` (+ the gate commit) |
| Ships | `v0.9.5` with Q0's tooling |
| Gate | §7 — CI: fault + sweep tests; local: `make p10` snapshot, 1 h soak, relay proof on both platforms, fault run ≤ 3× |

## 1. What changed, and why (junior-first)

1. **weatherUSA is reachable again** (`domains/radio/stream/directory.go`). DISCOVER LR-1: the relay's
   directory only speaks a TLS key exchange Go removed in 1.22, so for every Go build since, the
   directory failed silently (4 retries every 5 min) and its ~120 mounts were never offered. Its mounts
   were always `http://…/NWR/*.mp3`, so the fix is the honest one: both weatherUSA constants are
   `http://` (relay policy, said in the CHANGELOG). Two guards make plain HTTP safe: a mount is accepted
   only when the document's advertised host is the relay's host (port ignored — Icecast says `:8000`), and
   every client refuses redirects that leave the origin (§3 below).
2. **A down directory is said, once** (`stream.Status`, `radioDeck.noteDirectories`). `MountsWithStatus`
   reports each relay's health; the deck turns the first failure into a `radio_unavailable` warning in
   `[S]` (the enum already had the code — no schema change) and re-arms after recovery. A failing
   directory is asked again only after `directoryTTL` (5 min), whichever of Tune / advance / SetMode
   resolves — the ToS cadence holds on failure too (PR-9).
3. **Redirect policy** (`httpx.SameOriginRedirect`): three hops, same scheme and host, on the data client
   and the stream client. A refused redirect is our decision — it never memoises the host (IS-3).
3a. **The tune list spans every candidate station** (`tuneList`, `followMount`). Found by the live
   proof: weatherUSA lists 118 sources and some answer 404 (listed, not connected). Before Q1 such a
   transmitter was simply not offered and the resolver's next station played; with weatherUSA back it
   could be "relayed" by a dead mount alone and the engine — which walks one station's mounts — read
   FAILED. Now the engine walks all candidates' mounts in resolver order and the deck re-labels when a
   later station's mount plays (R6: the relay path may not regress).
4. **The disk tier is bounded** (`platform/httpx/cache.go`, plan §2.2). Four rules, each with its test:
   the persistence floor (caller TTL > 5 min or `Persist()` — only the relay directory), one retention
   rule (expired-with-validators kept 24 h as an LRU citizen; never served as fresh), the stale-memory
   disk skip, and the allow-list sweep (own names only, regular files only, `watchpost` path guard,
   `max(Expires, mtime) + 24 h`, 256 MB cap oldest-first, ≤ 10k listed / ≤ 1k removed per pass, at writer
   start and daily). Validators are stored now (two bounded fields) so Q5 has bodies to renew;
   `renew()` refreshes an mtime for it. The negative cache is capped at 1,024. `DiskWrites` is counted.
5. **One retry layer and a guarded failure memo** (`httpx/memo.go`, plan §2.3). `MaxRetries` zero value
   means none (the −1 encoding is refused); the dashboard clients use 1, `report` 3. The memo arms on a
   transport error at once, on 5xx only after three consecutive failures on two distinct URLs, never on
   4xx or a cancelled context; it is **consulted on the normal lane only** — the priority lane always
   attempts and clears it on any 2xx. `Retry-After` (integer or date) is clamped to 5 min into the memo
   and imposes a ≤ 30 s pacing hold on both lanes; no sleep inside `do()`. A retry that finds the host
   memoised ends the call with the original cause (the F1 message survives).
6. **Ledger and one-liners.** `timeType()` and `defaultTheme()` replaced two package variables (their
   P10-06 entries deleted); the ten dead directory-level kit entries are gone (SC-1); the remaining 66
   kit entries carry the template *frozen because / real items / patch / removable when* with the Q1
   finding count per file. Non-kit entries: **53** — the ratified target.
7. **Record.** `caching.md` rewritten (the floor, the rule, the skip, the sweep, the derived write
   rate); CHANGELOG `[Unreleased]` gains Fixed/Changed; `WATCHPOST_DEBUG_PPROF_ADDR` lets a second
   instrumented instance pick its port (the Q0 24 h soak holds 6060).

## 2. Files touched

| Area | Files |
|---|---|
| httpx | `cache.go` (rewritten around the four rules), `memo.go` (new), `httpx.go` (`Persist`, `MaxRetries`, `do` → `attemptOnce`, `CheckRedirect`), `stats.go` (`FastFail`), tests `cache_q1_test.go`, `memo_test.go` (new) |
| stream / player | `directory.go` (constants, `Status`, `MountsWithStatus`, host pin, failure memo), `resolve.go` (`ResolveWithStatus`), `icy.go` (redirect policy); tests |
| app | `radio.go` (`warn`, `noteDirectories`), `dashboard.go` (`MaxRetries`, `attachDiagnostics`, `wireDeckWarnings`, `debugAddr`), `app.go` (report `MaxRetries` 3), `stats.go` (`httpx.disk.writes`), `live_relay_test.go` (new, `WATCHPOST_LIVE=1`) |
| one-liners | `pkg/schema/schema.go` `timeType()`, `platform/render/theme.go` `defaultTheme()`; the two mock header constants moved into `render_test.go` |
| docs | `caching.md` (watchpost-cli record), `CHANGELOG.md`, `scripts/quality/soak.sh` (address), this log |

**Docs touched**: `caching.md:26` "Best effort" and `:58` "Do not add a memo" rewritten; CHANGELOG
`[Unreleased]` Fixed/Changed; `architecture.md:316` (obs `max-age`) is Q2's; README diagnostics
paragraph gains `WATCHPOST_DEBUG_PPROF_ADDR` at the gate commit.

## 3. Tests first (the pins)

| Test | Pins |
|---|---|
| `httpx.TestSingleFiveHundredHealsWithinFifteenSeconds` | one URL's 5xx never memoises the host; the next call heals |
| `httpx.TestOneStationFiveHundredNeverDelaysAlerts` | ≥ 3 consecutive 5xx on ≥ 2 URLs arm; the normal lane fast-fails without touching the network; the priority lane is attempted and clears the memo; fast-fails counted |
| `httpx.TestThreeDistinctRecentFailuresThenAlertsIsAttempted` | the alerts tick after RECENT failures is tried |
| `httpx.TestOneTransportErrorDelaysNoPriorityRow` | a transport error arms; the priority lane never consults the memo |
| `httpx.TestFirstViewUnchangedUnderOneURLFiveHundred` | one URL's failures do not memoise |
| `httpx.TestMemoNeverArmsOnFourXXOrCancel`, `…RefusesToArmBeyondSixteenHosts` | arming rules and the bound |
| `httpx.TestRetryAfterIsClampedAndHoldsBothLanes` | integer/date parsing, the 5-min clamp, the ≤ 30 s hold on both lanes |
| `httpx.TestRedirectPolicy` | same-origin followed; cross-host refused; > 3 hops refused |
| `httpx.TestZeroRetriesIsExpressible` (updated) | 0 = one attempt; −1 refused |
| `httpx.TestSweepIsAnAllowList` | README, subdir, symlink-with-a-cache-name, fresh, within-grace, renewed-by-mtime, young tmp and wrong-shape names survive; beyond-grace, legacy `.json`, old tmp and header-less files go |
| `httpx.TestSweepRefusesADirectoryWithoutWatchpostElement`, `…EnforcesTheDirectoryCapOldestFirst`, `…RunsAtWriterStartAndDaily` | the guard, the cap, the schedule |
| `httpx.TestPersistenceFloorAndPersistOption`, `…StaleMemoryEntrySkipsTheDiskRead`, `…ExpiredEntriesWithValidatorsAreRetained`, `…ValidatorsAreStoredBounded`, `…RenewedEntryOutlivesAnUntouchedOne`, `…NegativeCacheIsCapped` | the floor, the skip, the rule, the fields, the renewal, the cap |
| `stream.TestWeatherUSAIsPlainHTTPEndToEnd`, `…MountsAcceptedOnlyOnTheDirectoryHost`, `…FailingDirectoryIsAskedAtMostOncePerTTL` | the constants, the host pin (port-agnostic, canonical host accepted), the 5-min failure memo with statuses |
| `player.TestStreamClientRefusesCrossHostRedirect` | the stream client's policy |
| `app.TestDirectoryOutageWarnsOncePerOutage` | one `radio_unavailable` per outage, re-armed on recovery, nil-safe |
| `app.TestTuneListSpansCandidatesAndLabelFollowsTheMount` | mounts flatten in station order; the label stays until another station's mount plays, then follows it |
| `app.TestLiveRelayMountsPlayOnBothRelays` (`WATCHPOST_LIVE=1`) | real directories, one mount per relay opened and 64 KB read — the "plays" gate on each platform |

## 4. Before / after

| Quantity | Before (DISCOVER) | After Q1 | Source |
|---|---|---|---|
| weatherUSA directory | 4 failed TLS handshakes / 5 min; 0 mounts offered | read over HTTP; ~120 mounts offered; a down relay says so once in `[S]` | LR-1; live proof §8 |
| disk cache at launch | 1,429 files / 115 MB (593 legacy orphans) | **798 files / 37.7 MB** after the start sweep | soak row 1 |
| disk writes | ~45k/day (L4-F2) | derived ≈ 23k/day (floor: alerts 0, obs 0, FIRMS 12.8k, ≥ 30-min group 10k) — measured in §8 | `httpx.disk.writes` |
| outage attempts | ~23,000/h estimated (L2-F1) | one retry + memo: a down host costs the normal lane one probe per 20 s, the priority lane its cadence — measured in §8 | fault run |
| P10 | 123 findings / ledger 131 (55 non-kit) | 121 findings / 0 live / ledger 119 (**53 non-kit**, 66 kit itemised) / 0 unmatched | `07-readiness/p10-q1.json` |

## 5. Bounds stated (§0.8)

| Structure | Bound | Pinned by |
|---|---|---|
| failure memo | 16 hosts, refuse to arm beyond; TTL 20 s; Retry-After ≤ 5 min; hold ≤ 30 s | `TestMemoRefusesToArmBeyondSixteenHosts`, `TestRetryAfterIsClampedAndHoldsBothLanes` |
| negative cache | 1,024 entries, soonest-expiring dropped | `TestNegativeCacheIsCapped` |
| disk tier | 256 MB cap; ≤ 10k listed / ≤ 1k removed per pass; 24 h grace | `TestSweepEnforcesTheDirectoryCapOldestFirst`, constants |
| validators | 2 fields × 512 B, control bytes refused | `TestValidatorsAreStoredBounded` |
| directory failure memo | one entry per relay (2); 5 min | `TestFailingDirectoryIsAskedAtMostOncePerTTL` |

## 6. Decisions and non-decisions

- **`MaxRetries` semantics changed**: the zero value now means *no retries* and −1 is refused (PA-7's
  trap). Every test client that said `-1` now says `0`; the dashboard says `1`, `report` says `3`.
- **The station chain already continued on any error** (`nws.fetchObs` loops with `continue`) — no
  change, PR-2's concern was about the v1 plan text.
- **A refused redirect never arms the memo and is never retried** — it is our policy, not the host.
- **The canonical relay host is accepted alongside the fetch base** so the trimmed live fixtures keep
  working under the pin; production is unaffected (base host == canonical host).
- **`p10-unmatched.sh` now matches like the tool** (directory prefix only for P10-05) — that is what
  exposed the ten dead kit entries; platform-tagged files are checked on their own OS.
- **Q5b's shared-cache question stays open**; the two clients still sweep their own writes into one
  directory — the sweep is idempotent and cheap, so racing daily sweeps cost nothing.

## 7. Gate

| Check | Result |
|---|---|
| `make verify` (fmt, vet, tidy, vuln, race, lint-imports, lint-watermark, gate-controls) | ALL GATES GREEN |
| `make alloc-budget` (CI, non-race) | green (Q1 touches no render path) |
| fault tests · sweep tests (CI) | green — the six pins of §2.3 and the sweep's allow-list, guard, cap and schedule |
| `make p10` (local) | 0 live · 0 unmatched · non-kit **53** (= the ratified target) · 66 kit entries itemised · snapshot `07-readiness/p10-q1.json` |
| `a2dh validate` | 18/18 |
| 1 h soak: orphans 0 after the start sweep; files/bytes flat | **yes** — 1,429 files / 115 MB → 798 / 37.7 MB at the start sweep; 799 / 36.1 MB an hour later (§8) |
| 1 h soak: disk-write rate vs the derived rate | **404/h measured vs ≈ 950/h derived** — below, not within ± 20 %; see §8 for why the derivation over-stated FIRMS for this configuration; the policy holds (writes down 78 % from 45k/day) |
| Nearest Relay plays a weatherUSA mount **and** a wxradio mount | **macOS: yes** — `WATCHPOST_LIVE=1 go test ./app -run LiveRelay`: wxradio KEC80 (80 kbps) and weatherUSA KZZ69 (32 kbps, plain HTTP) each open through the player and yield 64 KB · **Arch: HUM LEAD runs the same command** (the Linux half of the gate) |
| Synth pty smoke (R6) | first data → PLAYING in 4 s → Repeat: Watchlist (2-min run at volume 5 on the Q1 binary) |
| counters under a fault run ≤ 3× healthy | **not run live** — a 5xx cannot be injected into TLS'd NWS traffic without a MITM; the bound is proven at the client (`memo_test.go`) and analytically: a full NWS outage costs the normal lane one probe per 20 s (180/h) and the priority lane its cadence with one retry (≤ 2 × ~350/h) ≈ 900/h ≈ **1.3× the healthy 691/h**. The live fault run moves to Q5's gate, where the provider bases already point at `httptest` servers. |

**Deviation for HUM LEAD.** Two gate lines are not met as literally written: the write rate is *below*
the derived band rather than inside it, and the fault run is analytical rather than live. Neither is a
regression; both are stated here so the gate is a decision, not a wave.

## 8. The 1-hour run and the relay proof

Q1 binary (`2f3e919`), idle dashboard (6 favourites + 50 recent, radio off), `WATCHPOST_DEBUG_PPROF=1`
on port 6061, 2026-08-26 16:32–17:34 UTC, PID 18454, 120 samples (`02-analysis/q1-soak-1h.csv`).

| Series | Q1 hour | Q0 hour (same machine, day before) |
|---|---|---|
| post-GC heap, 5-min minima | 37.9 · 34.8 · 37.6 · 35.4 · 37.8 · 36.9 · 39.6 · 35.3 · 35.5 · 36.2 · 39.2 · 39.2 MB — **flat** (34.8–39.6) | 33.8–38.7 |
| raw post-GC samples | 34.8–129.9 MB (the HMS parse transient lands in one sample) | 33.8–66.8 |
| footprint / RSS at the end | 108 MB / 273 MB (RSS inflated by the parse transient; footprint is the number) | 99 / 142 |
| threads | 24 (25 for three samples) | 24 → 26 |
| goroutines / fds | 274 mode (274–309) / 15–20 | 274 / 14–17 |
| disk cache | **798 → 799 files, 36.0 → 36.1 MB** (one date-keyed URL in the hour; the daily sweep takes it) | 1,428 → 1,429 files, 114.9 MB |
| disk writes | **429 in 64 min = 404/h** | 26 at launch, then unmeasured |
| requests (healthy hour) | NWS 691 attempts / 689 net / 528 cache / 24.6 MB, all h2, 5 TLS handshakes · CO-OPS 179 / 158 · NDBC 178 / 178 / 9.8 MB · FIRMS 45 / 539 cache · HMS 8 / 17.1 MB · WFIGS 5 · **0 fast-fails, 0 negative hits** | — |
| publishes | priority 270 (10 folded) · recent 451 (1,682 folded) | recent 392 |
| gauges | `httpx.mem.entries` 360 / 11.1 MB · `nws.gridinfo` 56 · `coops.stations` 7,951 · `hms.memo.points` 40,754 · `tz.cache` 8 · `assembler.*.warnings` 0 | as Q0 |

**Why the write rate is below the derived figure.** The derivation took L4-F2's per-kind split of the
45k/day baseline (HUM LEAD's process: 10 favourites, a busier RECENT set) and kept FIRMS at 12.8k/day.
On this machine's configuration FIRMS made 45 network fetches in the hour (the rest were cache hits
across the two pipelines' shared URLs), so its persisted share is ~1k/day, not 12.8k. The ≥ 30-min
group (points, forecasts, gridpoints, stations, tides) accounts for the rest: ≈ 400/h ≈ 9.7k/day —
exactly the derivation's 10k/day for that group. The floor policy is doing what it says; the
derivation's FIRMS term was a configuration-dependent over-estimate. Recommendation: the criterion
reads "≤ derived" from here (the derived figure is the ceiling the floor guarantees), and Q5's FIRMS
tiles retire the term.

**The relay proof** (`app/live_relay_test.go`, `WATCHPOST_LIVE=1`): both directories read (wxradio
had refused one connection minutes earlier — a transient the memo would have handled); of weatherUSA's
118 listed sources the first tried (KZZ41) answered 404 (listed, not connected) and the second (KZZ69)
played — the observation behind §1 item 3a. Both mounts went through the stream client with the
same-origin redirect policy in place.
