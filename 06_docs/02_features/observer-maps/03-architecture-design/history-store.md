# The local history store: design (W18.3; D-166 to D-169)

Status: **DESIGN, ruled (D-170 to D-173), for the HUM LEAD's review before it is built.** This page gives API shape only.

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

**`Dataset{Name string; Version int; Cadence Cadence; Fields []Field; Retain time.Duration}`**

- A dataset is registered when the store opens.
- `Cadence` is `Hourly` (a UTC hour) or `Daily` (a UTC date).

**`Field{Name, Unit string; Decimals int}`**, for example `temp` in °C with 1 decimal, or `wind_from` in degrees with 0.

**`Key{Source, Place string}`**, for example `{ndfd, us-a}`, or `{station, KSEA}` for a later trend.

- Each segment matches `[a-z0-9._-]` and is at most 64 bytes.
- A key outside that is refused.

**`Shape{Box geo.Box; Cols, Rows int}`**

- A lattice has many columns and rows.
- A point has one of each: `Cols` and `Rows` are both 1, and `Box` is the point.

**`Record{Dataset string; Version int; Key Key; Bucket, IssuedAt time.Time; Shape Shape; Values map[string][]float64}`**

- Each field's values run in rows from the north, west to east within a row, as `Lattice` orders them.
- A missing value is NaN in memory and `null` on disk, never zero.
- A record whose shape differs from the reader's expected shape reads as absent. That keeps old records from being drawn after the geometry of `fieldBoxes` or `LatticeFor` changes.

## 3. On disk

**Root (D-170).** `$XDG_DATA_HOME/watchpost/weather/history`, by default `~/.local/share/watchpost/weather/history`.

- A relative `XDG_DATA_HOME` is refused, as `config.Path` refuses one.
- `DefaultRoot()` returns `""` when there is no home directory, and the store then does nothing.

**A file a day per series, compressed (D-171: fidelity and speed together).**

- Hourly datasets: `<root>/<dataset>/v<version>/<source>/<place>/<YYYY-MM>/<DD>.json.gz` - the day's hourly records in one versioned JSON document, rewritten whole (temp file and rename) each hour it gains one.
- Daily roll-ups: `.../rollup/<YYYY>.json.gz`, a year's days in one file.
- A year of a series is 365 files, not 8,760, and a trend's range read opens one file a day.
- `zcat` still shows any of it, `null` carries NaN, and there is no new dependency.

**Two tiers.**

- Within the hourly retention, every hour is kept.
- Beyond it, each day is rolled up - each field's minimum, maximum and mean at each point - and the day's hourly file is removed.
- Fallback datasets keep hours only; trend datasets roll up.

**Sizes - per region, not per place (D-171).** The lower 48's nine boxes, each an 80-point lattice, cover every place inside them.

| Lower 48, all nine boxes | Compressed |
|---|---|
| An hour, one box | about 0.5 KB |
| 30 days of hours | about 3.2 MB |
| A year of hours (opted into) | about 39 MB |
| A year of daily roll-ups | about 5 MB |

**A place's own series** - a point, five fields, 24 hours a day - is about 70 KB a year. The watchlist and recent places, 60 or so, come to about 4 MB a year: a target use the design holds to (D-171).

**Permissions.** Directories 0700, files 0600.

## 4. Several instances, one store

**Writes.**

- A write goes to a `.tmp-` file created in the target directory.
- The file is fsynced, closed and chmodded, then renamed over the target. This is `httpx.cache.writeEntry`'s pattern.
- A reader therefore sees a whole old record or a whole new one, never part of either.

**Two writers on one record.** The last rename wins.

- A writer that finds a record with an equal or newer `IssuedAt` skips its own write.
- The race between two writers is harmless, because either record is correct data.
- **A day file is read, merged and replaced**, so two instances adding *different* hours of one series at once could each drop the other's (the last rename wins). A writer re-reads the file after its rename and, if an hour it merged is missing, merges and writes again - at most three times, then counted in `Stats` and left to the next hour. Instances record on the same clock, so this is rare, and the loss is bounded to one hour of one series.

**Readers** skip any name beginning with `.`.

**Pruning.** `ENOENT` is success everywhere, because another instance may prune at the same time.

**No lock files.** Rename is atomic, and every other operation can be repeated safely.

**Crashed writers.** Temp files older than 10 minutes are swept by the prune.

## 4b. Several instances, beyond the store

Several watchpost instances on one machine - a Broadcaster and an Observer, say - share more than the store's files. Each of these is settled so N instances cost what one does, and none depends on another staying up.

**The Open-Meteo quota is the machine's, not a process's.** The quota gate (batch 71) holds a spent host in memory, so each instance finds the refusal for itself - one refused request each - and probes on its own clock. Instead:

- The gate keeps its hold in a small shared file, `$XDG_STATE_HOME/watchpost/quota.json` (by default `~/.local/state/watchpost/quota.json`): each spent host, its period, its reset and its next probe - written by temp file and rename, read before each ask (memoised for a few seconds).
- Any instance that is refused writes it; every instance honours it, so one refusal holds them all.
- **One probe for all:** the instance whose probe is due takes it by moving the next probe time forward in the file first (write, re-read, proceed only if its own write stands); an answered probe clears the host for everyone.
- The file unreadable or absent: each instance falls back to its own memory - as batch 71 is now.

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

**The [ Data ] Settings tab** lets the listener opt into longer hourly and trend retention, and shows the store's size. The byte budget follows what is chosen, not a fixed cap.

**When pruning runs.** `Prune(now)` runs when the store opens and hourly from the recorder; any instance may run it.

- Each pass is bounded: at most 256 directories visited and 64 removed; whatever is left waits for the next pass.
- A version directory no longer registered ages out and is never read.

## 6. API

- **`Open(root string, sets ...Dataset) *Store`** never fails. A root that cannot be used gives a store that does nothing.
- **`(*Store) Put(r Record) bool`** records one bucket.
- **`(*Store) Get(dataset string, k Key, bucket time.Time) (Record, bool)`** reads one bucket.
- **`(*Store) Range(dataset string, k Key, from, to time.Time, max int) []Record`** reads a span, oldest first, at most `max` records. This is what a trend reads.
- **`(*Store) Latest(dataset string, k Key, at time.Time, maxAge time.Duration) (Record, bool)`** returns the newest record at or before `at`, no older than `maxAge`. It looks back at most `maxAge` divided by the cadence. This is what replay reads.
- **`(*Store) Keys(dataset string, max int) []Key`** lists what has been recorded.
- **`(*Store) Stats() Stats`** returns `Stats{Puts, PutFailures, Skipped, Corrupt, VersionMismatch, Pruned, Bytes}`.

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
5. **Replay:** one appended RECORDED chip, drawn as every map chip - its ground and label, no brackets (D-173, D-174).

**A place's own series** - the watchlist, recent places - is a target use. Its readings could be sampled from the region's recorded lattices (no requests, but interpolated) or fetched per place (faithful, up to one request a place an hour); the feature that uses it chooses.
