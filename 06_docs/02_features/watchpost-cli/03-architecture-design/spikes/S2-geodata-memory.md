# Spike S2 — Geocoding Dataset Memory Cost (gates PLAN exit)

**Date:** 2026-08-23 · **Machine:** macOS arm64 (Darwin 25.6.0), Go 1.24.4 · **Metric under test:** M8 (≤40MB total RSS at 10 locations) · **Red-team flag:** P-1 (budget may not close)

## 1. Dataset facts

| Dataset | Download | Uncompressed | Rows | Trimmed TSV | Trimmed + gzip -9 |
|---|---|---|---|---|---|
| cities15000.zip | 3,307,857 B | 8,409,959 B | 34,106 | 2,239,173 B | 768,967 B |
| US.zip (postal) | 634,334 B | 2,668,861 B | 41,490 | 1,473,115 B | 539,436 B |
| **Total** | 3.76MB | 10.6MB | 75,596 | 3.54MB | **1.25MB** |

Trim commands (GeoNames column layouts):

```
awk -F'\t' 'BEGIN{OFS="\t"}{print $2,$3,$11,$9,$5,$6,$15,$18}' cities15000.txt > cities_trim.tsv
awk -F'\t' 'BEGIN{OFS="\t"}{print $2,$3,$5,$10,$11}' US.txt > zips_trim.tsv
gzip -9 -k cities_trim.tsv zips_trim.tsv
```

## 2. Method

One Go binary (`go:embed` of both .gz files), run once per mode as a separate process. Each run: `runtime.ReadMemStats` HeapAlloc after double-GC before and after load; `debug.FreeOSMemory()`; RSS via `ps -o rss=` on own PID and `Getrusage` MaxRSS. Prefix lookups: 1,000 random 3–6-char prefixes drawn (seed 42) from real asciinames, case-insensitive, results sorted by population descending. Sanity check: `"spring"` → 22 hits, top `Springs` (pop 186,394) in both b and c. Binary delta: identical source rebuilt with zero-byte embedded files. Numbers are medians of 3–4 runs (RSS spread ≈ ±1.5MB).

## 3. Results

| Representation | Live heap delta | Process RSS after load | Prefix lookup (mean/1000) |
|---|---|---|---|
| Empty baseline (Go runtime only) | — | **5.2MB** | — |
| a. NAIVE `[]struct` w/ strings | 10.29MB | **29–32MB** | not measured |
| b. COMPACT TSV bytes + sorted `[]uint32` offsets, lazy parse | 4.64MB | **16.5–18.1MB** | **6.7–7.3µs** ✅ |
| c. GZIP-resident, decompress-scan per query | 0.74MB | **6.6MB** (16.5MB after queries) | **12.3–12.7ms** ❌ |

Load time: naive 35ms, compact 73ms, gzip 0.1ms. Binary size: 4,149,906 B with data vs 2,828,946 B without → **+1.32MB** (≈ the gzip payload, byte-for-byte).

## 4. Verdict — does M8 close?

**Yes, with representation (b).** Arithmetic on this machine:

- Baseline Go runtime RSS: **5.2MB** (the red-team's 15–25MB assumption was ~3–5x pessimistic here)
- Compact geodata cost: 18.1MB − 5.2MB ≈ **12.9MB RSS** (live heap only 4.6MB; the rest is macOS lazily reclaiming decompress/sort churn — MADV_FREE pages stay counted until memory pressure)
- **Total: ~18MB → ~22MB headroom** under the 40MB ceiling for 10 locations of watch state, TUI, and HTTP buffers.

Even NAIVE (~30–32MB) technically fits but leaves only ~8–10MB headroom — reject. GZIP-resident is cheapest (6.6MB) but fails the <10ms type-ahead budget at 12.3ms/query (and burns CPU per keystroke).

## 5. Recommendation

**Ship (b) COMPACT**: `go:embed` the two gzipped trimmed TSVs (+1.32MB binary), decompress once at startup into a single backing `[]byte` per dataset, build `[]uint32` line-offset indexes (cities sorted by lowercased asciiname), binary-search prefixes, parse rows lazily on access. 7µs lookups give 1000x margin on type-ahead. Optional refinement: pre-sort the TSV at build time and stream-decompress into a preallocated buffer to cut the ~8MB of load-time churn RSS.

## 6. Caveats

- RSS figures are macOS-specific; Linux (MADV_DONTNEED behavior differs) and Windows will differ. Baseline 5.2MB is this spike's tiny binary — the real TUI app (bubbletea etc.) will have a larger runtime baseline; re-verify M8 in VAL.
- Watch-state cost at 10 locations is not measured here; only ~22MB headroom is claimed, not proven end-to-end.
- MaxRSS is peak, not steady-state; steady-state live data is the HeapAlloc column (4.6MB for compact — close to the plan's 7–10MB estimate, which was roughly right for NAIVE at 10.3MB).
- Lookup latency measured with warm cache, single-threaded, exact-prefix hits only; fuzzy matching would cost more.
- Dataset snapshot of 2026-08-23; GeoNames row counts drift over time.

**Artifacts:** trimmed/gzipped TSVs + measurement source retained durably at `06_docs/02_features/watchpost-cli/04-development/spikes/` (`_src/s2-main.go.txt`, cities_trim.tsv.gz, zips_trim.tsv.gz); raw downloads deleted.
