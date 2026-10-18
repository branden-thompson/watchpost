# The M1 and M4 instrument: design, and the forks it needs ruled

Up: [plan, W10.3](../04-development/implementation-plan.md) · [problem statement, M1 and M4](../01-objectives/problem-statement.md) · [dry run](dry-run.md) · [rulings](../02-analysis/rulings.md)

**Where this stands (2026-10-18).** D-208 puts the M1/M4 scorer's baseline inside BUILD: the
climatology's agreement with WSPR, the shipped build's, and a column for the background alone (B-F7).
This page is the instrument's design and, below, what was built. **Its four forks are ruled, each as
recommended: A2 (D-218), B1 (D-219), C1 (D-220), D1 (D-221).** The instrument is built to them
(`tools/reachscore`, batch 45, A-74); its dry run on PLAN's week comes next.

## What is already fixed

| | | Ruled by |
|---|---|---|
| The reference circuit | SSB voice, 100 W, simple antennas, **+13 dB SNR in 2.5 kHz** (ITU-R F.339-8, J3E just usable, stable) | D-73, D-91 |
| WSPR as the observation | a spot counts toward "would carry voice" when its SNR, normalised to 100 W from its reported power (`snr + (50 - power_dBm)`), is at least +13 dB; WSPR's SNR is in 2500 Hz and WSPR is constant-envelope | D-73, D-91 |
| The window | the answer for the UTC hour, matched within ±1 h | M1, M4 |
| The set | at least 200 scenarios per metric, both outcomes, at least 3 days, reported by band | D-99 |
| The target | the climatology's agreement plus 5 points; if the climatology is above 90 %, "no worse than climatology"; a miss comes to the HUM LEAD at VALIDATE | D-99, D-118 |
| M1's scenarios | the tower, a place in its served area, the hour, the band | M1 |
| M4's scenarios | the listener's place, a target place, the hour, the band | M4 |

## The columns

Each scenario is answered three ways, on the same set:

1. **The climatology alone** (PyIRI's, as go-ionomaps carries it): the baseline the target is set against.
   It needs no live input, so any day can be answered.
2. **The background alone**: NOAA's GloTEC and the climatology, no stations assimilated (B-F7, D-214's
   "None"). It needs the day's GloTEC grids.
3. **The shipped build**: GloTEC, the climatology and GIRO's stations, as watchpost answers. It needs the
   day's grids and the stations' readings.

Each is the chart's answer for the band at the hour, read as open or closed by fork C.

## The forks

### A. What evidences "closed"

M1 and M4 need both outcomes, and WSPR reports only what was heard.

| | Option | For | Against |
|---|---|---|---|
| A1 | **A spot heard below +13 dB normalised** is "would not carry voice": closed for the reference circuit | it is a measurement on that very path and hour; it is exactly the threshold the circuit sets | its negatives cluster near the threshold, where every model is least certain |
| A2 | A1, **and a pair monitored but not heard**: the transmitter spotted by others on the band in the window, the receiver reporting others on the band in the window, and no spot between them | negatives far from the threshold; closer to "the band was closed" | an unheard pair can be the antennas, local noise or the schedule, not the ionosphere |
| A3 | A2's monitored pairs alone, A1's spots counted as heard | | counts a path that carried a signal as closed, which is not what the circuit asks |

**Recommended: A2, the two kinds of negative reported in separate columns as well as together**, so the
agreement is not won or lost by which kind dominates. **Strongest counter-argument:** a monitored pair's
silence is weaker evidence than a measured weak spot, and folding it in may flatter whichever model
calls more paths closed.

### B. How scenarios are drawn

| | Option | For | Against |
|---|---|---|---|
| B1 | **Stratified by band and UTC hour, balanced between outcomes**, drawn at random within each stratum from the days' pairs | a constant answer scores 50 % (the anti-solution checks, `uat-protocol.md:34`); every band and hour is represented | the balance is not the real base rate, so agreement here is not "how often watchpost is right" |
| B2 | At random from all pairs, the base rate as it falls | agreement reads as accuracy in use | a model that always says "closed" at night can score high on base rate alone |

**Recommended: B1, with the base rate reported beside it.** M1's tower is a WSPR transmitter in the
mainland US and its served area the receivers within the service radius (D-195); M4's pairs are any US
listener and the six continents' reference points' regions (FR-1.9), so M4 tests the paths the chart
answers. **Strongest counter-argument:** WSPR's stations are not where listeners and towers are, so the
set measures WSPR's geography.

### C. How a chart answer is read as open or closed

| | Option |
|---|---|
| C1 | **Open or disturbed is "reaches"; absorbed, too weak and above the upper limit are "does not"**, as the words and `Usable` already read them (D-73: disturbed is degraded, not closed) |
| C2 | Disturbed counted as closed |

**Recommended: C1**, the reading the product already shows. **Strongest counter-argument:** a band
disturbed by D-RAP may well fail +13 dB, and C1 counts it as reaching.

### D. Which days

| | Option | For | Against |
|---|---|---|---|
| D1 | **The instrument built and its dry run on PLAN's week now** (its GloTEC and GIRO data are kept with the PLAN evidence, outside the tree; WSPR spots for those days from wspr.live); **the graded measurement on the fresh days beside REVIEW**, with G10.4's fetch (D-132, D-208) | the baseline inside BUILD as D-208 asks; the grade on days nothing was tuned on | two runs |
| D2 | Everything on PLAN's week | one run, now | PLAN's week chose the hybrid and set the targets (D-132's reason): the grade could not fail |
| D3 | Everything on fresh days, beside REVIEW | the cleanest grade | no baseline inside BUILD, against D-208 |

**Recommended: D1.** **Strongest counter-argument:** a baseline on PLAN's week may not be the baseline
on the fresh days, so the "+5 points" could be judged against the wrong climatology figure; D1 answers
that by reporting the climatology on both.

## What the instrument would send

wspr.live, as PLAN did (`dry-run.md`, "Requests made"): **aggregate and filtered SQL**, one query a day
and band at most, the spots' fields only (time, band, the two locators, power, SNR), for the US mainland
and the continents' regions; nothing else is fetched for the dry run. The spots, like PLAN's, stay
outside the tree with the PLAN evidence.

## As built: `tools/reachscore` (batch 45, A-74)

A Go tool with two commands. `spots` sends wspr.live the day's one query and keeps the answer outside the
tree. `score` reads the spots, draws the set by B1, answers each scenario three ways through go-ionomaps
by replaying the days' recorded files, reads each answer by C1, scores by A2, and prints the table by
band. It keeps the scenarios and the inputs' checksums outside the tree as well. The table names the
tool's commit and says whether the tree was clean; figures are cited only from a clean, committed tree
(E-19).

```mermaid
flowchart LR
  W[wspr.live: one aggregate query a day] -->|spots kept outside the tree| D[draw B1: band x UTC hour, open = closed]
  P[PLAN's recorded files: GloTEC grids, GIRO replies, solar file] --> R[replay at the clock: nothing dated after it]
  R --> C1[go-ionomaps: no grid, no stations = climatology]
  R --> C2[go-ionomaps: grids, no stations = background]
  R --> C3[go-ionomaps: grids and stations = shipped build]
  D --> A[each scenario answered by propanswers.WorkChart, read by C1]
  C1 --> A
  C2 --> A
  C3 --> A
  A --> T[table by band: agreement, base rate, each kind of closed, the target]
```

**What it sends.** wspr.live is sent one query a day: the spots heard by receivers in the mainland US,
on the ten bands go-ionomaps answers, from transmitters in the US or within the region of a continent's
reference point. The query aggregates them per pair of locators, band and hour, keeping the count and
the strongest SNR normalised to 100 W. **Locators are fetched, never callsigns**, which is all "the spots'
fields" above names. Each day's window runs from 23:00 the day before to 23:00, covering every scored
hour's ±1 h. The courtesy matches D-39's for GIRO: one query at a time, 15 s apart, and after a refusal
60 s doubling to 15 minutes before the day is given up. A day already kept is never asked for again.
The request log keeps the rate headers alone (D-83).

**The replay.** One library per column runs through the days hour by hour, as a host left running would,
making the stations' burst when it is due. The replay answers from the recorded files at the library's
clock:
- GloTEC's index lists only the grids valid by then.
- The solar file holds only the days before.
- A station's reply is cut to the window asked and the clock, in the columns go-ionomaps asks for, with
  the line naming the original requester left out.

It never serves anything dated after the clock. Any address it does not know fails the run. It also
checks each column was fed as the column says: no grid in the climatology's, and no station's reading
outside the shipped build's.

**The scored hours.** Every third UTC hour, with the clock at 35 minutes past: PLAN's week kept a grid
every three hours, at 5 past (`dry_fetch.py`). That makes each scored hour's grid the newest one, and
well within go-ionomaps' hour.

**The instrument's settings (A-74), which shape the set and are put to the HUM LEAD with the dry run's
figures:**

| Setting | Value | Why |
|---|---|---|
| The ground wave's reach | pairs under 50 km are not drawn | PLAN's own cut (`dry-run.md`, "ground wave removed") |
| M1's served area | receivers 50 to 161 km from the tower, answered for a 161 km service radius | 161 km is the largest service radius ruled (100 mi); the default 25 mi (40 km) lies inside the ground wave's reach |
| M4's targets | transmitters within 1000 km of a continent's reference point | the regions the chart's six targets stand for (FR-1.9) |
| M4's listener | answered with the 400 km near-vertical radius | the Setting's default |
| The draw | at most 10 of each outcome per stratum, seed 19 | B1 needs tens per stratum; the seed fixes the set |

**What the dry run cannot say.** PLAN's week kept no D-RAP table and no scales file, so every column
lacks a disturbance: "disturbed" is never said, and C1's reading of it is not exercised. The solar file
PLAN kept begins on 2026-09-06, so the early days' F10.7 means average fewer than 30 days. A locator
names a square rather than a station: two stations in one square count as one, and a 4-character
locator is a square about 110 by 170 km in the US. "Monitored" means monitored by the US receivers the query
keeps.
