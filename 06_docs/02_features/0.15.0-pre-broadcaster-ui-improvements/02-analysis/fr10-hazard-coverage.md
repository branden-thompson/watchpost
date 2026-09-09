# FR-10 — hazard coverage, and the boundary (bounded investigation, 2026-09-08)

**The sighting.** Brengel Fire, Vista CA, morning of 2026-09-06. An evacuation order was issued and
the hotspot appeared in the details for **neither** Oceanside 92057 nor Vista — the two nearest
tracked locations.

**Scope discipline (HUM LEAD, 2026-09-06, unchanged): the FEEDS were examined, not the reader.** This
is a written disposition, not an open hunt.

## FR-10.1 — the FIRE. Why no hotspot.

**Candidates eliminated, with evidence:**

- **A confidence filter — NO.** `fire.Rules.MinConfidence` is `nominal`, and `confidenceRank` returns
  −1 for an unrecognised value, so a raw feed string *would* be dropped silently. But `firms.confidence`
  maps VIIRS `l`/`n`/`h` and MODIS 0–100 onto the shared scale and **falls back to `nominal` for
  anything it does not recognise** — kept, not dropped. HMS hardcodes `analyst` (rank 3). Neither
  feed can lose a detection here.
- **A radius — NO.** `RadiusKm` defaults to 25 km and the UAT ran a 16-mile (≈25.7 km) ring. Vista and
  Oceanside are a few km apart; a fire forcing evacuations in Vista is inside both rings.
- **An FRP floor — UNLIKELY.** `MinFRPMW` is 5, and `Keep` explicitly **keeps a detection whose FRP is
  unknown** (`frpMW == nil`). It could drop a very weak detection, but not one large enough to force
  evacuations.

**The cause that survives: THE SOURCES CANNOT SEE A NEW FIRE QUICKLY, BY CONSTRUCTION.**

| Source | Satellites | What that means for a morning ignition |
|---|---|---|
| **FIRMS** (`firms.sources()`) | `VIIRS_NOAA20_NRT`, `VIIRS_NOAA21_NRT` — **polar-orbiting only** | Each satellite passes roughly twice a day (~01:30 / ~13:30 local). A fire that starts after the night pass is **invisible until the afternoon pass** — a gap of hours. |
| **HMS** | GOES-East/West, NOAA-20/21, Suomi NPP, MODIS | The archive refreshes every 10 minutes, but its contents are **analyst-curated**. The curation is the latency. |
| **WFIGS** (`services3.arcgis.com`) | — | Named *incidents*, not detections. A fire enters it when an agency files it, not when it ignites. |

**There is no geostationary near-real-time detection path.** GOES sees the continent every few
minutes and reaches this app only through HMS, behind analyst curation. So the honest answer to
FR-10.1 is: **not a defect, a coverage boundary** — the product cannot see a new fire until a polar
satellite passes over it or an analyst draws it, and the window for a morning ignition can be most of
a day.

**A keyless install is narrower still.** FIRMS needs a key; without one the hotspot feed is HMS alone,
and `livePipelines.markFIRMS` correctly reports the provider `off` rather than `ok`. The listener is
told the provider is off; they are **not** told what that costs them.

## FR-10.2 — the ORDER. Why no evacuation alert.

**Every alert this app reads comes from three endpoints:** `api.weather.gov/alerts/active`,
`earthquake.usgs.gov/...`, `nhc.noaa.gov`. That is the whole list.

`Evacuation Immediate` is an NWS CAP event, and C-2 made it lead the read — **necessary and not
sufficient**, exactly as the requirement predicted. A fire evacuation ordered by **CAL FIRE, a county
OES or a sheriff** is not an NWS product and **will never appear on `api.weather.gov`**. No amount of
work inside the NWS reader changes that.

**This is the same question raised at the fire UAT** — the CAL FIRE three-tier scheme
(AWARE / WARNING / IMMEDIATE, "Ready / Set / Go"). The answer is now definite: **it is not in our
feeds, and it needs a new source** (CAL FIRE's own incident/evacuation API, or a county OES feed, or
an aggregator). That is a 0.16.0-scale provider addition under the FR-7.1 precedent, not a fix here.

## FR-10.3 — the boundary is stated

Deliverable regardless of the other two, and the reason the item was rolled into 0.15.0: **a boundary
nobody states is a boundary nobody can account for.** What must be said, and to whom:

- **To the OPERATOR** — the coverage the install actually has, including what an unkeyed FIRMS costs.
- **To the LISTENER — ALREADY SHIPPED, and better than a disclaimer.** Every report head carries the
  notice, and the fire and seismic heads additionally **name the providers the report derived from**,
  which is a per-read coverage statement rather than a blanket caveat:

  > *"This is the Watchpost Fire and Hotspot report for {{.Location}}. This report is derived from data
  > from {{.Sources}}. Data for this report may be delayed or incomplete, and is not intended for life
  > safety use."*

  Verified across every report kind: `weather-radio`, `fire-report`, `seismic-report`, `event-report`,
  `global`, `transition/masthead` and `marine-report` all carry it; `test-alert` says it is a test.
  *(An earlier pass here reported `marine-report` as missing it. That was a grep for `life safety`
  against a script that writes `life-safety`, hyphenated — a false absence, and the third from one of
  my own instruments in this release.)*

- **The BREAKING takeover carries no notice, and that is RULED, not an omission.** HUM LEAD
  2026-09-08: *"Leave it. Takeovers are brief — name the location, and direct the user or listener to
  a detailed report or official website."* The takeover RELAYS an official warning rather than
  interpreting data, and `breaking/burst-closing` already does the directing — *"For more details on
  any of these alerts, press W in Watchpost"* — with the destination named as the one
  edition-specific value, so a Broadcaster station names its website instead. Hedging a live warning
  is the one place a caveat would cost more than it buys.

**Nothing in this document is a code change.** FR-10.1 and FR-10.2 both conclude "cannot be fixed
here", which is the outcome the requirement explicitly allowed for, and FR-10.3 was already shipped.
**What is owed downstream is a SOURCE, not a fix:** CAL FIRE / county OES evacuation orders, as a
0.16.0 provider under the FR-7.1 precedent.
