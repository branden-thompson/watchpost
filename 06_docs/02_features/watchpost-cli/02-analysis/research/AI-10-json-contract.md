# AI-10: JSON Contract (`--json` / `--report-only`)

Builds on AI-2 §3 (SI-internal normalized schema + `source{}` block) and AI-1 (alert fields, QC semantics). Requirement T-C, metric M5 (100% data parity).

## 1. Prior Art

| Source | Lesson for Watchpost |
|---|---|
| **CAP 1.2** (OASIS) [1] | Alert = `alert{identifier,sender,sent,status,msgType,references}` + `info[]{category,event,urgency,severity,certainty,onset,expires,headline,description,instruction,area[]}`. Enums are closed, capitalized (`Extreme/Severe/Moderate/Minor/Unknown`). Timestamps are ISO-8601 **with offset, `Z` forbidden**. NWS fields in AI-1 are CAP verbatim; keep the names. |
| **NWS API** [2] | GeoJSON + JSON-LD `@context`; every measurement is `QuantitativeValue{value, unitCode:"wmoUnit:degC", qualityControl}`; `null` value = not observed. Self-describing but verbose and triple-nested. |
| **Open-Meteo** [3] | Flat parallel arrays + companion `hourly_units{}`; ISO-8601 local or unix. Compact, jq-hostile for per-row access, no provenance. |
| **Pirate Weather / Dark Sky** [4] | `currently/minutely/hourly/daily/alerts/flags`; `flags.sources[]`, `flags.units`, `flags.version`; sentinel `-999` for "no expiry" (anti-pattern: use `null`). |
| **WMO/OGC** | SI unit names via WMO unit codes (`degC`, `m_s-1`, `hPa`) [2]; OGC API Features uses `links[]` and explicit `numberReturned` counts. |
| **`gh --json`** [5] | Field selection + built-in `--jq`/`--template`; pretty-print only on TTY. Caller-chosen fields = stable scripts. |
| **`kubectl -o json`** | `apiVersion`/`kind` envelope = self-identifying version; `items[]` list wrapper; never omits declared fields within a version. |
| **clig.dev** [6] | `--json` for structure, `--plain` for one-record-per-line, TTY detection disables colour, non-zero exit on failure. |
| **JSON Lines / `jc`** | One object per line for streams; `jc` standardises `null` for missing and `_unix`/`_iso` duplicate timestamp fields. |

Conventions extracted: explicit `schema_version` (kubectl, Pirate `flags.version`); timestamps RFC3339 UTC plus IANA `tz` per location (CAP/NWS keep offsets, but UTC + tz is friendlier to diffing); SI internally with one `units{}` block (Open-Meteo style) rather than per-field suffixes or NWS triple nesting; **`null` for "provider has no value", omit nothing declared** (NWS, `jc`); enums lower-case, closed, documented in schema.

## 2. Proposed v1 Contract

Envelope: `schema_version`, `generated_at`, `watchpost_version`, `request{command,args,units_requested,providers_requested}`, `units{}`, `providers[]{id,role,status,fetched_at,attribution}`, `locations[]`, `warnings[]{code,message,location,provider}`.

Per location: `label`, `zip`, `lat`, `lon`, `tz`, `harmonized{current,hourly[],daily[]}`, `alerts[]` (CAP names, NWS-only in v1), `fire{hotspots[]}`, `radio{station,stream_url,status}`, `by_provider{<id>:{current,hourly[],daily[]}}`, `diffs[]`.

Provenance: **per block**, not per field. `harmonized.current.source` names the winning provider and `fill_from` lists providers that filled nulls (NWS tie-break ruling). Per-field provenance is derivable from `by_provider` and would double the payload.

```json
{
  "schema_version": "1.0.0",
  "generated_at": "2026-08-23T18:04:11Z",
  "watchpost_version": "0.1.0",
  "request": {"command": "report", "args": ["20001"], "units_requested": "si", "providers_requested": ["nws","open-meteo"]},
  "units": {"temp": "degC", "wind": "m_s-1", "pressure": "hPa", "precip": "mm", "distance": "km", "visibility": "m"},
  "providers": [
    {"id": "nws", "role": "reference", "status": "ok", "fetched_at": "2026-08-23T18:04:09Z", "attribution": "NOAA/NWS"},
    {"id": "open-meteo", "role": "secondary", "status": "ok", "fetched_at": "2026-08-23T18:04:10Z", "attribution": "Open-Meteo.com (CC-BY 4.0)"}
  ],
  "locations": [{
    "label": "Washington, DC", "zip": "20001", "lat": 38.9072, "lon": -77.0369, "tz": "America/New_York",
    "harmonized": {
      "current": {
        "observed_at": "2026-08-23T17:52:00Z", "temp": 29.4, "feels": 33.1, "dewpoint": 22.0,
        "humidity_pct": 64, "pressure": 1014.2, "wind": 3.6, "wind_dir_deg": 180, "wind_gust": null,
        "precip_1h": 0.0, "cloud_pct": 40, "condition_code": "partly_cloudy", "is_day": true,
        "visibility": 16093, "uv_index": 7,
        "source": {"provider": "nws", "model_or_station": "KDCA", "distance_km": 4.8, "issued_at": "2026-08-23T17:52:00Z",
                   "fill_from": {"uv_index": "open-meteo"}}
      },
      "hourly": [{"time": "2026-08-23T19:00:00Z", "temp": 30.1, "precip_prob_pct": 20, "condition_code": "partly_cloudy", "source": {"provider": "nws"}}],
      "daily": [{"date": "2026-08-23", "temp_max": 31.0, "temp_min": 22.0, "precip_prob_pct": 30, "sunrise": "2026-08-23T10:27:00Z", "sunset": "2026-08-24T00:51:00Z", "source": {"provider": "nws"}}]
    },
    "alerts": [{
      "id": "urn:oid:2.49.0.1.840.0.abc", "event": "Heat Advisory", "severity": "moderate", "urgency": "expected", "certainty": "likely",
      "message_type": "alert", "sent": "2026-08-23T14:05:00Z", "effective": "2026-08-23T14:05:00Z", "onset": "2026-08-23T16:00:00Z",
      "expires": "2026-08-24T00:00:00Z", "ends": "2026-08-24T00:00:00Z", "references": [], "affected_zones": ["DCZ001"],
      "area_desc": "District of Columbia", "headline": "Heat Advisory until 8 PM EDT", "description": "...", "instruction": "...",
      "source": {"provider": "nws"}
    }],
    "fire": {"hotspots": [{"lat": 38.95, "lon": -77.10, "detected_at": "2026-08-23T06:30:00Z", "confidence": "nominal", "frp_mw": 12.3, "distance_km": 7.9, "source": {"provider": "firms"}}]},
    "radio": {"station": "KHB36 NOAA Weather Radio", "stream_url": "https://...", "status": "playing"},
    "by_provider": {
      "nws":        {"current": {"observed_at": "2026-08-23T17:52:00Z", "temp": 29.4, "uv_index": null, "condition_code": "partly_cloudy"}},
      "open-meteo": {"current": {"observed_at": "2026-08-23T18:00:00Z", "temp": 30.2, "uv_index": 7, "condition_code": "partly_cloudy"}}
    },
    "diffs": [{"field": "current.temp", "values": {"nws": 29.4, "open-meteo": 30.2}, "delta": 0.8, "threshold": 1.5, "flagged": false}]
  }],
  "warnings": [{"code": "obs_stale", "message": "KDCA observation is 72 min old", "location": "20001", "provider": "nws"}]
}
```

(Hourly/daily truncated to one row; `by_provider` shows a subset.) Side-by-side view reads `by_provider`; diff view reads `diffs[]`; harmonized view reads `harmonized`.

## 3. Parity Mechanism (M5)

- One Go type, `Snapshot`, is the *only* input to both `View()` (Bubble Tea) and `json.Marshal`. Providers write into `Snapshot`; the view never calls providers or caches.
- Renderers receive `Snapshot` by value and may not hold any other data source; a lint rule (`go vet` custom analyzer or a `forbidigo` pattern) blocks imports of `provider/*` from `ui/*`.
- **Golden parity test**: load a fixture `Snapshot`, render TTY at width 80 and 200 with ANSI stripped, marshal JSON, then walk every leaf of the JSON and assert its *rendered form* (via the same `format.Temp()` etc. helpers the view uses) appears in the TTY text. Inverse check: regex-extract every number/time token from the TTY text and assert each maps to some JSON leaf. Both directions fail on drift.
- **Schema-completeness test**: reflect over `Snapshot`, assert every exported field has a `json` tag and a matching property in `schema.json` (or an explicit `json:"-"` with a comment `// tty-only`).
- Legitimately TTY-only: column widths, glyphs/icons, colour, sparkline characters, scroll position, selected tab, elapsed-since-refresh countdown. These live in a separate `ViewState` struct that wraps `Snapshot`, so they can never leak into the contract. `condition_code` (enum) is data; the glyph chosen for it is not.

## 4. Streaming / Agent Mode

- `watchpost report <loc> --json --watch` emits **JSON Lines**: one object per line, each with `event` ∈ `snapshot | alert_added | alert_updated | alert_cancelled | provider_error | heartbeat`, plus `schema_version` and `at`. `snapshot` carries the full envelope (simplest for agents; diffs are derivable). Pretty-print only when stdout is a TTY, like `gh` [5].
- Exit codes: `0` ok; `1` error (no usable data, bad args); `2` partial (any provider `status != ok` or any `warnings[]`); `3` stale when `--fail-on-stale=<dur>` is set and the reference observation is older than `dur`. In `--watch`, exit codes apply at termination (SIGINT → 0).
- `--report-only`: plain text, no ANSI, width computed once from `$COLUMNS`/`--width` (default 80), one record per line where possible (clig `--plain` [6]). Rendered from the same `Snapshot` via the same formatters, so it inherits M5 coverage.
- Auto-detect: when stdout is not a TTY and neither flag is given, default to `--report-only` and print a hint on stderr.

## 5. Versioning & Compatibility

- Schema semver independent of binary: `schema_version: "1.0.0"`. Within major 1: additive only (new optional fields, new enum values *only* in fields documented as open enums, e.g. `warnings[].code`; closed enums such as `severity` require a major bump). Renames/removals/type changes → `2.0.0`, and the binary supports `--schema-version 1` for one major release.
- Deprecation: `"deprecated": true` annotation in `schema.json` plus a `warnings[]` entry with `code:"deprecated_field"` when the field is emitted.
- Publish `schema/watchpost-report.v1.schema.json` (draft 2020-12, `$id` URL, `additionalProperties:false` at the envelope so drift is caught) in the repo, embedded via `embed` and printed by `watchpost schema [--version 1]`. Include `"$schema"` reference URL in the output only under `--json-include-schema` to keep the payload small.
- CI: `santhosh-tekuri/jsonschema` v6 [7] — supports 2020-12 with full test-suite compliance and hierarchical errors; `xeipuuv/gojsonschema` is unmaintained and stops at draft-7. Tests validate every golden fixture and every `--watch` event line against the schema; a second test asserts the schema itself has not changed without a `schema_version` bump (hash check).

## 6. Opinion

**Shape**: nested per-location, per-block provenance. Flat Open-Meteo arrays optimise bandwidth, not agents; nested objects with `harmonized` / `by_provider` / `diffs` map 1:1 to the three user views, which makes M5 checkable per view. Keep CAP field names for alerts so agents can cross-reference NWS directly.

**Parity test**: the bidirectional golden test in §3 plus the reflection schema-completeness test. It is cheap (one fixture, two renders) and catches the two real failure modes: a field added to the view but not the struct, and a field added to the struct but not the schema.

**Strongest counter-argument**: per-block `source` loses information when harmonization fills individual nulls from a secondary provider — an agent cannot tell which field came from Open-Meteo without diffing `by_provider`. The `fill_from{}` map in `source` addresses the common case, but if harmonization ever blends (averages) values, per-field provenance becomes mandatory and the contract would need a 2.0. Mitigation: ruling forbids blending (NWS wins outright, secondaries fill only nulls), so record that constraint in the schema description.

## Sources

1. https://docs.oasis-open.org/emergency/cap/v1.2/CAP-v1.2-os.html
2. https://api.weather.gov/openapi.json (QuantitativeValue, Alert schemas; see AI-1)
3. https://open-meteo.com/en/docs
4. https://docs.pirateweather.net/en/latest/API/
5. https://cli.github.com/manual/gh_help_formatting
6. https://clig.dev/
7. https://github.com/santhosh-tekuri/jsonschema
8. https://kubernetes.io/docs/reference/using-api/api-concepts/ (apiVersion/kind envelope)
9. https://jsonlines.org/
10. AI-2 §3 normalized schema; AI-1 alert/QC fields (this repo)
