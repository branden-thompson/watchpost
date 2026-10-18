# The M1 and M4 instrument: design, and the forks it needs ruled

Up: [plan, W10.3](../04-development/implementation-plan.md) · [problem statement, M1 and M4](../01-objectives/problem-statement.md) · [dry run](dry-run.md) · [rulings](../02-analysis/rulings.md)

**Where this stands (2026-10-18).** D-208 puts the M1/M4 scorer's baseline inside BUILD: the
climatology's agreement with WSPR, the shipped build's, and a column for the background alone (B-F7).
This page is the instrument's design. **Its four forks are ruled, each as recommended: A2 (D-218), B1
(D-219), C1 (D-220), D1 (D-221).** The instrument is built to them next; nothing has been fetched yet.

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

## Built after the rulings

A tool under `tools/` (Go, as the shell ledger asks): it reads the spots, draws the set by B, answers it
three ways through go-ionomaps by replaying the day's recorded inputs, reads each answer by C, scores by
A, and writes the per-band table with its inputs' checksums. The run's figures come from the committed
tool at the committed version (E-19).
