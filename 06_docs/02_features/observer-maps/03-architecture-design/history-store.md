# The local history store: design (W18.3; D-166 to D-169)

Status: **DESIGN, for the HUM LEAD.** It is not built until the open questions (section 10) are ruled. This page gives API shape only.

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

**Root.** `$XDG_DATA_HOME/watchpost/weather/history`, by default `~/.local/share/watchpost/weather/history`.

- A relative `XDG_DATA_HOME` is refused, as `config.Path` refuses one.
- `DefaultRoot()` returns `""` when there is no home directory, and the store then does nothing.

**Paths.**

- Hourly datasets: `<root>/<dataset>/v<version>/<source>/<place>/<YYYY-MM-DD>/<HH>.json`.
- Daily datasets: `.../<YYYY>/<MM-DD>.json`.
- One directory per day means pruning a day is removing a directory.

**Format.** One versioned JSON object per record, with a header followed by the fields.

- It can be inspected by hand.
- `null` carries NaN.
- The schema check is explicit.
- It needs no new dependency.

**Size.** About 2.5 KB a record, uncompressed.

- The lower 48 has 9 boxes, so a day of hourly records is about 23 KB an hour, 48 hours about 1.1 MB, and 30 days about 16 MB.

**Permissions.** Directories are 0700 and files 0600.

## 4. Several instances, one store

**Writes.**

- A write goes to a `.tmp-` file created in the target directory.
- The file is fsynced, closed and chmodded, then renamed over the target. This is `httpx.cache.writeEntry`'s pattern.
- A reader therefore sees a whole old record or a whole new one, never part of either.

**Two writers on one record.** The last rename wins.

- A writer that finds a record with an equal or newer `IssuedAt` skips its own write.
- The race between two writers is harmless, because either record is correct data.

**Readers** skip any name beginning with `.`.

**Pruning.** `ENOENT` is success everywhere, because another instance may prune at the same time.

**No lock files.** Rename is atomic, and every other operation can be repeated safely.

**Crashed writers.** Temp files older than 10 minutes are swept by the prune.

## 5. Retention

**Per dataset.** Each dataset sets its own `Retain`.

- The fallback datasets keep 72 hours: the 48 hours needed, plus margin.
- A trend dataset chooses its own.

**Global budget.** 64 MB by default. Past the budget, the oldest day directories go first, across all datasets.

**When pruning runs.** `Prune(now)` runs when the store opens and hourly from the recorder, and any instance may run it.

- Each pass is bounded: at most 256 directories visited and 64 removed.
- Whatever is left waits for the next pass.
- A version directory that is no longer registered ages out and is never read.

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
- The recorder runs hourly at about minute 5, on the app's `everyTick`, even while the map is closed.
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

## 10. Open questions for the HUM LEAD

1. **Directory.** Should the store be in the XDG data directory, as above, or in `./watchpost/data/weather/history` as the ruling suggested, or should it be a Setting?
2. **Trends.** How long should trend datasets keep by default? 30 days, for example?
3. **Budget.** Is 64 MB right?
4. **Broadcaster alone.** Is the station's region recorded when only Broadcaster is running? That costs up to 9 NDFD requests an hour.
5. **Labelling.** Does a replayed grid say it is recorded, for example "recorded 14:00" in its note or credit?
