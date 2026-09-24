# M1 and M1b's recorded scenarios

Nine scenarios captured from `api.weather.gov` on 2026-09-24 (the time is in `manifest.json`), each a
folder: `alerts.json` (the alerts as served), `zones/` (the zone shapes each zone-based alert names)
and `scenario.json` (the place, the sizes it is shown at, any failure or withheld zone, and the answer
key). Email addresses in the NWS sender headers are replaced with `sender@nws.invalid`; everything else
is as served.

| Scenario | Place | What it is |
|---|---|---|
| 01 | Oak Ridge, TN | Inside a Flash Flood Warning's polygon — covers |
| 02 | Charleston, WV | 9.9 km outside a Flash Flood Warning — stops short |
| 03 | Harlan, KY | Inside one Flash Flood Warning, 7 km from two more |
| 04 | Roswell, NM | Covered by a zone-based Flood Watch; a Flood Warning 34 km away — to one side |
| 05 | Fort Davis, TX | A Flood Watch over five zones with Davis Mountains (TXZ277) withheld — partial |
| 06 | Wilmington, DE | A Coastal Flood Warning one county over — to one side |
| 07 | Buxton, NC | Covered by Hatteras Island's Coastal Flood Warning; a Gale Warning 4 km offshore — marine |
| 08 | Hilo, HI | A Hurricane Watch over the Big Island interior and waters — Hawaii's region |
| 09 | Oak Ridge, TN | Scenario 01 with the network down — failure |

**The answer key** is computed from the recorded geometry — covers: the place is inside the area;
stops short: outside, the nearest edge within 15 km; lies to one side: farther — and **the HUM LEAD
confirms it before scoring** (M1's protocol, `problem-statement.md`). M1b is scored first, or on a
disjoint half, and the choice is recorded with the result.
