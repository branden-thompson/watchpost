# The local history store: design (W18.3; D-166 to D-169)

Status: **GO given (HUM LEAD, 2026-09-30); ruled D-166 to D-180. The store's core is built (batch 74, `platform/history`); the recorder, the replay, the [ Data ] tab and the shared quota come after.** API shape only here.

## Why

On 2026-09-30, Open-Meteo refused watchpost for the rest of the day (HTTP 429). Every Open-Meteo layer then drew nothing (D-165). The HUM LEAD ruled on what to do about that:

- **D-166.** watchpost records NDFD's hours in the background, whether or not Open-Meteo is answering. A later loop replays its past hours from that record when Open-Meteo fails. The record lives on disk, outside the binary. It is shared by every running instance and survives a restart.
- **D-167 and D-168.** Open-Meteo's valid UV hours and rain days are recorded too.
- **D-169.** The store is general: "architected to support other features in the future ... trend visualizations".

## 1. Placement

**`platform/history`** is a new package. It imports only the standard library, `platform/geo` and `platform/invariant`, and knows nothing of temperature, NDFD or Open-Meteo.

**`app/history.go`** holds the app's adapter:

- It turns a `temperature.Lattice` and one hour's values into a record, and back again.
- It runs the background recorder.
- It answers replay requests.

**`domains/temperature`** gains one method, `NDFD.Hour(ctx, Lattice, now) (Series, error)`. It asks for the current hour alone, using the same URL as the second request of `Fetch`, so the two share one HTTP cache entry.

## 2. Data model

**Every kind of source (D-179).** The store is not the weather's alone: a dataset is whatever a source says, by time.

**`Dataset{Name, Title, Description string; Version int; Step time.Duration; Fields []Field; Hours, Days time.Duration; MaxBytes int64}`**

- `Step` is the width of one record's bucket: a minute to a day, an hour by default - NDFD's hour, an ionosonde's MUF every 5 or 15 minutes, a tide gauge's 6.
- `Hours` is how long its records are kept; `Days` how long its days, rolled up, are kept past that (zero: none).
- `MaxBytes` is the most it holds on disk, every version counted (0.19.0 D-143; zero: no bound). Past it, its oldest files go first, before its retention ends (section 5).
- `Title` and `Description` are for a reader that did not register it (D-180).

**`Field{Name, Label, Unit string; Decimals int}`** - `temp`, "Temperature", °C, 1; `muf`, "Maximum usable frequency", MHz, 2. Decimals 0 to 9.

**`Key{Source, Place string}`** - `{ndfd, us-a}`, `{giro, pa836}`, `{station, ksea}`: each 1-64 bytes of `[a-z0-9._-]`, never `.` or `..`; any other key is refused.

**`Shape{Box geo.Box; Cols, Rows int}`** - a lattice, or with one of each a point.

**`Record{Key; At, IssuedAt time.Time; Shape; Values map[string][]float64; Doc json.RawMessage}`** - one bucket, in either or both of two payloads:

- `Values`: each numeric field over the shape, rows from the north, west to east within a row; NaN for missing, `null` on disk, never zero. Values need a shape.
- `Doc`: a JSON document of at most 1 MiB whose form is the dataset's own, versioned with it - for what is not a number over a shape: alerts, quakes, fire perimeters.
- Pictures are not recorded: the HTTP cache and the map library keep those (a later payload kind would carry its own budget).

**`Day{Key; Date; Shape; Hours int; Min, Max, Mean map[string][]float64}`** - a day rolled up: each numeric field's minimum, maximum and mean at each point, over its recorded buckets. Documents are kept, not rolled up.

## 3. On disk

**Root (D-170).** `$XDG_DATA_HOME/watchpost/weather/history`, by default `~/.local/share/watchpost/weather/history`; a relative `XDG_DATA_HOME` is refused. `DefaultRoot()` is `""` with no home directory, and the store then does nothing.

**A series' files** (`<root>/<dataset>/v<version>/<source>/<place>/`):

| File | Holds | Written |
|---|---|---|
| `manifest.json.gz` (per dataset version) | the dataset's description (D-180) | at open, when absent or changed |
| `<YYYY-MM>/<DD>/<HHMM>.json.gz` | one bucket's record | by `Put`, a file of its own |
| `<YYYY-MM>/<DD>.json.gz` | a finished day's records, compacted | by the prune pass, once the day is done |
| `rollup/<YYYY-MM>-p<N>.json.gz` | a month's days rolled up, in parts | by the prune pass, past the hours' retention |
| `rollup/<YYYY>.json.gz` | a year's days rolled up, as 0.18.0 wrote them | read, never written; pruned past the days' retention |

- A year of a series is 365 day files once compacted, not thousands; a trend's range read opens one file a day.
- **A roll-up part never passes the read cap (0.19.0 #27, FR-6.5).** A part is held to a budget of half the read cap, 16 MiB decompressed. A day goes to the part already holding its date, else the month's last part while it stays within the budget, else a new part. A part that exists but cannot be read is never written over: the day waits, its hours kept. A month's parts are listed from the directory, so a part removed hides none after it. A 31-day month of a 2° global grid with two fields is about 16.7 MB, one part (`TestNinetyDaysOfAGlobalGridStayReadable`, `make property`).
- **Values are held as one array a field**, a missing one NaN, not one allocation a value (FR-6.5). On disk they are written exactly as before, a missing one `null`. Every read merges a day's file with its buckets not yet compacted, the newer issue where both hold a bucket.
- Compressed JSON, versioned: `zcat` shows any of it; `null` carries NaN; no new dependency.
- **Two tiers.** Within the hourly retention every bucket is kept; beyond it each day is rolled up and its records removed. Fallback datasets keep records only; trend datasets roll up (D-176: NDFD's hours roll up into trends).
- **Sizes - per region, not per place (D-171).** The lower 48's nine boxes, each an 80-point lattice, cover every place inside them:

| Lower 48, all nine boxes | Compressed |
|---|---|
| An hour, one box | about 0.5 KB |
| 30 days of hours | about 3.2 MB |
| A year of hours (opted into) | about 39 MB |
| A year of daily roll-ups | about 5 MB |

- **A place's own series** - a point, five fields, 24 hours a day - is about 70 KB a year; the watchlist and recent places, 60 or so, about 4 MB a year (a target use, D-171).
- Directories 0700, files 0600.

## 4. Several instances, one store

**No writer reads, merges and replaces a record file.** The first draft did - a day's hours in one file, each writer adding its hour - and under contention it lost records: twelve instances writing twelve hours of one day kept three. So:

- **Each bucket is its own file.** `Put` writes it by temp file (in its own directory), fsync and rename; no writer of another bucket touches it. Two writers of one bucket write the same data; a newer issue replaces an older, never the other way.
- **A finished day is compacted under a claim.** The day's `.compact-<DD>` claim is created exclusively; the holder writes the day's file from everything it reads, reads it back, and removes each bucket file the day's file holds at an issue at least as new - a bucket written meanwhile is left for the next pass.
- **A month's roll-up part is written under the series' roll-up claim** (`rollup/.claim-<YYYY-MM>`), the one file still read, merged and replaced - one writer at a time; a day's records go only once its part is read back holding it.
- **A bucket is fetched by one instance.** `Claim` creates `.claim-<DD>T<HHMM>` exclusively; a bucket already recorded is claimed by none.
- **A claim holds its time**, on the store's clock. One older than 10 minutes with nothing recorded is stale - its claimant crashed or closed - and is taken over.
- **Readers** skip every name beginning with `.`; every removal tolerates what another instance removed first; temp files older than 10 minutes are swept.
- **No lock files, no waiting.** Tested: two stores on one directory; twelve concurrent writers of one day; two concurrent compactions with a late bucket written during them (ten runs under the race detector, none lost).

## 4b. Several instances, beyond the store

Several watchpost instances on one machine - a Broadcaster and an Observer, say - share more than the store's files. Each of these is settled so N instances cost what one does, and none depends on another staying up.

**The Open-Meteo quota is the machine's, not a process's.** The quota gate (batch 71) holds a spent host in memory, so each instance finds the refusal for itself - one refused request each - and probes on its own clock. Instead:

- The gate keeps its hold in a small shared file, `$XDG_STATE_HOME/watchpost/quota.json` (by default `~/.local/state/watchpost/quota.json`): each spent host, its period, its reset and its next probe - written by temp file and rename, read before each ask (memoised for a few seconds).
- Any instance that is refused writes it; every instance honours it, so one refusal holds them all.
- **One probe for all:** the instance whose probe is due takes it by moving the next probe time forward in the file first (write, re-read, proceed only if its own write stands); an answered probe clears the host for everyone.
- The file unreadable or absent: each instance falls back to its own memory.
- **The land Open-Meteo Marine answers nothing for is the machine's too (D-218):** kept in `marine-land.json` beside `quota.json`, by lattice (its box and shape), each write merged with what the file holds so no instance's learning is lost; land learned more than 90 days ago is asked again; "Clear map data" removes it.
- **What is paid for is served while held (D-217):** a held host's answer already in the client's cache is served, never refused; only an answer from the network frees the host, so a probe is never spent on a cached body.

**The recorder records each hour once.** At :05 every running instance would fetch the same hour.

- An instance claims a series' hour by creating `.claim-<HH>` beside its day file with `O_CREATE|O_EXCL` - atomic on every local filesystem - and only the claimant fetches and writes.
- A claim older than 10 minutes with no record written is stale: another instance may remove it and claim again (the claimant crashed or was closed).
- Instances start the recorder at a random offset in the first minutes of the hour, so claims rarely contend.
- The fetch itself still goes through the shared HTTP cache: an hour the map already fetched costs nothing.

**Pruning and roll-ups** run in whichever instance's hourly pass comes first; both are idempotent, bounded, and tolerate files removed under them (section 4). A roll-up is written before the hours it summarises are removed, so a reader never finds a day in neither tier.

**Settings.** The retention and the [ Data ] tab's choices live in `config.toml`, which every instance reads; a change made in one reaches the others at their next config read, and pruning always uses the retention read at that pass, never a remembered one. An instance never prunes past a longer retention another has just chosen.

**Tests.** Two gates on one state file: a refusal in one holds the other, one probe between them, an answer frees both. Two recorders on one store: each hour fetched once; a stale claim taken over. A roll-up racing a reader: the day read whole from one tier or the other.

## 5. Retention

**Per dataset (D-171).** The fallback datasets keep 72 hours (the 48 needed, and margin). Trend datasets keep 30 days of hours by default, rolled up beyond that.

**The [ Data ] Settings tab** lets the listener opt into longer hourly and trend retention, and shows the store's size in its notices, at the foot of the tab (D-237). The byte budget follows what is chosen.

**A byte bound per dataset (0.19.0 D-143).** Each dataset also declares the most it holds on disk, every version counted. 0.18.0's datasets are bounded far past what the longest retention holds at their measured rates: 4 GiB for NDFD's current hour and AirNow's national file, 512 MiB for the rest. When a dataset passes its bound, the prune pass removes its oldest files first: roll-ups, then days with their buckets, never the current day. The tab then names it: "Kept shorter than chosen, at its size limit, oldest first: ...". A changed retention clears that until the bound acts again.

**When pruning runs.** `Prune(now)` runs when the store opens and hourly from the recorder; any instance may run it.

- Each pass is bounded: at most 256 directories visited and 64 removed; whatever is left waits for the next pass.
- A version directory no longer registered ages out and is never read.

## 6. API

**Writing**

- `Open(root string, now func() time.Time, sets ...Dataset) *Store` - never fails; an unusable root is a store whose writes fail and reads find nothing. Writes each dataset's manifest.
- `(*Store) Put(dataset string, r Record) bool` - records one bucket.
- `(*Store) Claim(dataset string, k Key, at time.Time) bool` - this instance's to fetch.
- `(*Store) RollUpAndPrune()` - compacts finished days, rolls up and removes what is past retention or past a dataset's byte bound, sweeps; bounded (256 directories visited, 64 files removed a pass); any instance may run it.

**Reading**

- `(*Store) Get(dataset string, k Key, at time.Time) (Record, bool)` - one bucket.
- `(*Store) Range(dataset string, k Key, from, to time.Time, max int) []Record` - a span, oldest first.
- `(*Store) Latest(dataset string, k Key, at time.Time, maxAge time.Duration) (Record, bool)` - the newest at or before `at` (replay).
- `(*Store) Days(dataset string, k Key, from, to time.Time, max int) []Day` - rolled-up days (trends).
- `(*Store) Bounded(dataset string) bool` - whether its byte bound has cut it short of its retention (D-143).

**Browsing - for a reader that did not write it (D-180, the Analyst mode to come)**

- `(*Store) Catalog() []Dataset` - every dataset version on disk, from the manifests, whoever wrote them.
- `(*Store) Series(dataset string, max int) []Key` - a dataset's series.
- `(*Store) Extent(dataset string, k Key) (first, last time.Time, ok bool)` - a series' first and last day.

**The diagnostics** - `(*Store) Stats() Stats`: puts, failures, skipped, corrupt, version mismatches, pruned, rolled up (D-124).

## 7. The first datasets

**`ndfd-hourly` v1**

- Fields: `temp` and `feels` in °C, `wind` and `gust` in km/h, `wind_from` in degrees.
- The recorder runs hourly at about minute 5, on the app's `everyTick`, **whenever any watchpost instance runs, in any mode - Broadcaster alone, the map never opened (D-172)**.
- It covers every field box of the station's region, and the region the map last asked for. In the lower 48 that is all 9 boxes, so any view can replay.
- A box another instance has already recorded this hour is skipped.
- Cost: at most 9 NDFD requests an hour for the lower 48, about 216 a day, and nothing extra when the map already fetched the same hour. NDFD needs no key and has no quota.

**`openmeteo-uv` v1**, with a daily `uv_max`.

- It is written by `withUV` whenever Open-Meteo answers.
- An hour with every value missing is not written.

**`epa-uv-cities` v1** (W20, D-224)

- Field: `uv`, the UV index. A series a city - `epa/<city>-<state>` - its shape the city's point, its
  name, state and zone in each record's document, so a reader that did not write it knows the city.
- It is written whenever EPA answers for a city, once a city a day: every hour of EPA's forecast for the
  city's day.
- It is read to draw every city in view with a reading for its own day, beside the cities asked
  (D-225); the series are listed at first use and each hour, so another instance's cities are known.

**One writer a kind of data (D-234).** The recorder above - the historian, running whenever any instance
runs, the map used or not - is the one writer of the map's sources; the dashboard's scheduler is the one
writer of the station's feeds (below). Radar is never stored. Open-Meteo is not kept beyond the replay
datasets D-167 and D-168 ruled (`openmeteo-uv`, `openmeteo-rain-days`).

**The historian's, once an hour, each claimed so one instance fetches it:**

- **`ndfd-rain-days` v1**: NDFD's rain (liquid-equivalent, mm) and snow (cm), each day it gives a box of
  the recorded regions, on the lattice the map asks totals on; NDFD serves no history of its forecasts.
- **`airnow-hourly` v1**: AirNow's national file as the hour's one record - each reporting area's name,
  state, point, AQI and forecasts for today and tomorrow, a missing reading absent (JSON says no NaN). The
  map's air layer draws the hour's record where there is one, and fetches the file only where there is
  not. One record, not a series an area: a series an area measured 11.6 s of disk an hour (D-234).
- **`usgs-quakes` v1**: the USGS `1.0_day` feed's quakes, each record an hour's by origin time, rewritten
  as the feed's list for that hour grows - a late report joins its hour. Each quake: its id, magnitude,
  scale, place, depth, origin time, epicentre, tsunami flag.

**The station's feeds (W22.2, D-230, D-231)**, written by `stationRecorder` from each fetch the dashboard's
scheduler applies (`sched.Config.OnFragment`) - no fetch of their own, each kept by the source it is of:

- **`nws-observations` v1**: each location's latest NWS observation (`nws/<location>`), an hour a record -
  temperature, feels-like, dew point, humidity, pressure, wind, its direction, gusts, the hour's
  precipitation, visibility; a reading NWS did not give is missing. NWS keeps about a week (D-230).
- **`nws-alerts` v1**: each alert its own series (`nws/a<digest of its id>`), its document as sent, at the
  hour it was sent.
- **`ndbc-buoys` v1**: each buoy's readings (`ndbc/<buoy>`) - waves, swell, wind-waves, wind, gusts,
  water temperature - an hour a record.
- **`coops-tides` v1**: each tide station's observed level above MLLW (`coops/<station>`), at the fetch's
  hour, the level carrying no time of its own.
- **`hms-hotspots` v1**, **`firms-hotspots` v1**: each feed's detections near a location
  (`hms/<location>`, `firms/<location>`), the list at the feed's time.
- **`wfigs-incidents` v1**: each incident its own series (`wfigs/i<digest of its name, state, discovery>`),
  its acres and containment an hour a record.

**Every dataset keeps what the Data tab says (D-175, D-231).** The two presets apply to all alike: the
values an hour a record for the hourly detail's window, then rolled up into days for the trends'; a
document, which rolls up to nothing, is kept through both windows together. Raising either preset first
asks (D-231): "KEEP MORE HISTORY?" with what it will take on disk - each dataset's growth since the store's
oldest day (`Since`, `BytesOf`) times the days added, a value dataset's trends day at a twenty-fourth of
its hours, a document dataset's whole - and that what was not recorded cannot be fetched back. Enter keeps
the longer window, esc puts the shorter back; a shorter window asks nothing.

**`openmeteo-rain-days` v1**

- Fields: `peak` in mm/h, `rain` in mm, `snow` in cm, one record per target date.
- It is written by `withRainDays` whenever Open-Meteo answers.

**Replay.** When a live source fails:

- The adapter reads `Latest` for the anchor hour, or `Get` for each day.
- Radar mode's past hours come from `Range`.
- Hourly replay reaches back at most 48 hours.

## 8. Failure

**Disk.** A disk that is missing or read-only makes `Put` return false.

**Bad records** read as absent. That covers:

- a corrupt file;
- a wrong shape;
- a version mismatch;
- a directory for a future version.

**Visibility.** None of this is ever shown to the listener (D-124). It counts in `Stats`, which reaches the debug dump and the diagnostics.

**Drawing.** A replayed grid is drawn exactly as a live one would be. Where there is nothing to replay, that box is not drawn.

**Said by one chip (D-173, D-174).** While a layer draws anything from the history, one RECORDED chip - `  RECORDED  ` on its own ground, no brackets, as every map chip - is appended after its source's chip: `RADAR  O-METEO  RECORDED ` drawn as two grounds side by side. One style for every source, no per-source logic.

## 9. Tests

**Store behaviour.**

- Two stores opened on one directory see each other's writes.
- Many goroutines `Put` the same bucket; the result always parses, and the newest `IssuedAt` wins.
- A leftover temp file is invisible to readers and is swept.
- `Range` returns records in order and stops at `max`.
- `Latest` honours `maxAge`.
- Retention removes by age, the budget removes the oldest first, and a prune pass stops at its bound.
- A prune while another store reads raises no error.

**Bad input.** Each of these is ignored and counted:

- truncated JSON;
- a wrong schema;
- a wrong shape;
- values of the wrong length.

**Edge cases.**

- NaN round-trips through `null`.
- An empty root gives a store that does nothing.

**App.** A fake NDFD or Open-Meteo refuses, and the grid is drawn from a store seeded beforehand.

**P10.**

- Every directory walk and every look-back is bounded.
- There is no recursion, only fixed-depth loops.
- `invariant.Check` guards shape lengths and keys.
- The recorder is the one unbounded loop, and it runs through `everyTick`.

## 10. Ruled

1. **Directory:** the XDG data directory (D-170).
2. **Retention:** 72 hours for the fallback, 30 days for trends, by default; longer opted into on a [ Data ] Settings tab; the store shaped for fidelity and speed (D-171).
3. **Budget:** follows the retention chosen; its size shown on the Data tab (D-171).
4. **Recording:** always, whenever any instance runs (D-172).
5. **Replay:** one appended RECORDED chip, drawn as every map chip - its ground and label, no brackets (D-173, D-174) - on a muted violet no source uses (D-178).
6. **The [ Data ] tab:** presets - hourly 72 h / 7 d / 30 d / 1 y; trends 30 d / 90 d / 1 y / 5 y - and the store's size (D-175); Clear history behind ARE YOU SURE (D-177).
7. **Trends from the fallback:** NDFD's hours roll up past 72 h and are kept for the trend retention (D-176).
8. **Every source, any step, numbers or documents (D-179); readable by a reader that did not write it - the Analyst mode to come (D-180).**

**A place's own series** - the watchlist, recent places - is a target use. Its readings could be sampled from the region's recorded lattices (no requests, but interpolated) or fetched per place (faithful, up to one request a place an hour); the feature that uses it chooses.
