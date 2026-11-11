# The shape of watchpost

This page is the program as it is built: what feeds it, what holds the data, what draws it, and the
one rule that keeps those apart. The design history behind each part is in its release's documents
under `06_docs/02_features/`. This page and the as-built maps are what the architecture atlas
(`06_docs/architecture-atlas.html`) renders.

## The program

```mermaid
flowchart TB
  subgraph ext["External sources: each is credited in About (app/credits.go)"]
    NWS["api.weather.gov<br/>observations · forecasts · alerts · zones · NDFD grids"]
    NOAA["NOAA<br/>CO-OPS tides · NDBC buoys · HMS fire · NHC cyclones · MRMS radar"]
    FIRE["NASA FIRMS (keyed) · NIFC WFIGS"]
    MORE["USGS quakes · EPA AirNow and UV · IEM radar<br/>Open-Meteo (supplemental)"]
    ION["GIRO ionosondes · NOAA GloTEC and SWPC"]
    ICE["NOAA Weather Radio relays (Icecast)"]
    OFM["OpenFreeMap basemap tiles"]
  end

  subgraph dom["domains/ — one folder per source family; none imports another's"]
    D1["weather/nws · marine/coops · marine/ndbc · fire/hms · fire/wfigs · fire/firms<br/>seismic/usgs · airquality · uv · temperature · radar · locations"]
    GF["globalfeed<br/>the world's largest active events, for the ticker"]
    PROP["propagation<br/>the one go-ionomaps object"]
    RAD["radio/stream · player · synth · cast · script · pronounce"]
  end

  subgraph plat["platform/ — shared, and imports no domain"]
    SCHED["sched<br/>tiered scheduler"]
    HTTPX["httpx<br/>cache · lanes · failure memo · redaction"]
    SNAP[("snapshot<br/>THE contract: immutable, published")]
    HIST[("history<br/>the record of what the sources said")]
    LINE["lineup<br/>the Director's schedule, as pure values"]
    REND["render<br/>the one importer of go-studs"]
  end

  subgraph modes["modes/ — read the snapshot, never a domain"]
    TTY["tty<br/>the Router: Observer and the Broadcaster console<br/>(Bubble Tea v2)"]
    REP["report<br/>--json · --report-only"]
  end

  APP["app/ — the composition root<br/>pipelines · the radio deck · the arbiter · MasterControl · the schedule's executors"]
  MAPS["go-tuimaps<br/>the map library"]
  IONO["go-ionomaps<br/>the propagation library"]
  AUDIO["oto (audio out) · macOS say or Piper (voices)"]

  NWS & NOAA & FIRE & MORE --> D1
  NWS & NOAA & MORE --> GF
  ION --> IONO --> PROP
  ICE --> RAD
  D1 -->|"Fetch, through httpx"| HTTPX
  SCHED -->|"tiers ask the location providers"| D1
  D1 -->|"a Fragment per location"| SNAP
  APP -.->|"wires"| dom & plat & modes
  APP -->|"records readings"| HIST
  APP -->|"executes its effects"| LINE
  SNAP -->|"read only"| TTY & REP
  TTY --> REND
  TTY -->|"draws in Update"| MAPS -->|"tiles"| OFM
  RAD --> AUDIO
```

## The rules the shape keeps

- **`modes/` and `platform/` import no domain** (`make lint-imports`). A domain writes the snapshot,
  and everything the surfaces need from a domain arrives as a hook in `tty.Config`, wired by `app`.
- **`platform/render` is the one package that imports go-studs** (0.19.5 D-10, `make lint-imports`).
  A surface that needs a table or a row asks for a render function.
- **One deck, one air.** Both surfaces read through the same radio deck in `app`. The Director
  (`platform/lineup`) orders what is read, and MasterControl gates the deck from the air. The
  Broadcaster always has the air while it is on screen. Its five roles, the Card Producer, the Card
  Composer, the Director, the Reader and MasterControl, are defined in `docs/where-things-happen.md`'s
  vocabulary. The as-built picture is
  `06_docs/02_features/0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md`.
- **The snapshot is immutable once published.** A frame reads one snapshot, and a newer one replaces
  it whole.

## A map of the packages

The main packages. `platform/` has 24; the rest are small helpers, each with a package comment
(`go doc ./platform/<name>`).

| Where | What lives there |
|---|---|
| `cmd/watchpost` | the command line: flags, the one-shot reports, and the dashboard's start |
| `app/` | the composition root: the PRIORITY and RECENT pipelines (`app/pipelines.go`), the ticker (`app/ticker.go`), the radio deck (`app/radio.go`), the arbiter (`app/arbiter.go`), MasterControl (`app/mastercontrol.go`), the schedule and its executors (`app/schedule.go`, `app/executors.go`), the map's builders and feeds (`app/maps.go`, `app/maplayers.go`), the history recorder (`app/history.go`) |
| `domains/` | one folder per source family, each a `snapshot.Provider` or a source a pipeline asks; see the diagram |
| `platform/snapshot` | the data contract every reader consumes, and the assembler that merges the fragments |
| `platform/report` | what a location report can be made of: the report kinds, each one row in `all()`, and the only place a kind is described |
| `platform/sched`, `platform/httpx` | when a provider is asked, and how a request is made, cached and redacted |
| `platform/history` | the local record of readings, hourly and rolled up, for trends |
| `platform/lineup` | the Director's schedule: cards, tracks, and the events that move them |
| `platform/render` | every formatter, table and glyph set; the go-studs seam |
| `platform/config` | `config.toml`: load, validate, and the one serialized writer |
| `platform/geo`, `platform/propanswers` | shapes and bearings; the Propagation mode's answers |
| `modes/tty` | the live program: the Router, Observer's dashboard and windows, the Broadcaster console, the map pane |
| `modes/report` | `--json` and `--report-only` |
| `pkg/schema` | the published JSON Schema of the report |
| `tools/` | the gates' own instruments (authoring, dupes, wires, atlas, ids and the rest) |

**Comments cite the original design,
[`06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md`](../06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md), as "watchpost-cli architecture §n".**
It was approved before the first build, and its sections are the contracts the oldest packages were written
to: the snapshot (§2, §10.1), the scheduler and the client (§3), rendering (§4), the radio (§5),
the terminal and the CLI (§6), the schema policy (§10.3) and config (§10.11). Its diagrams are that
release's design; this page is the program as it stands.

Where a particular thing happens, step by step, is `docs/where-things-happen.md`. How to add a
detail row, a provider, a radio control, a report kind, a map layer, a Setting, a spoken phrase or a line on a Broadcaster card is
`docs/extending.md`.
