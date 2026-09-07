# UAT — P4, and the feature (multi-voice-support 0.14.0)

**This is the real one.** P0–P3 were agent-verifiable only (`agent-uat-p1.md`, RN-4): the cast existed but
you had to hand-edit a file to reach it. From P4 it is yours — you open Setup and pick a voice, like a
listener.

**What you are judging, and what you are not.** The *sound* of every preset, the *wording* of every script,
and the Setup labels are **yours** (`tones.md` §1, MVS-D-21, the mock). If a tone is wrong, the wording is
clumsy, or a row reads badly, that is a ruling, not a bug — say so and I will change it. What is a bug is
anything that does not do what it says.

Roughly 30–40 minutes. Steps 1–4 are the feature; 5–7 are the things most likely to be wrong.

---

## 0. Build, on your real config

```sh
cd ~/Desktop/PERSONAL_PROJECTS/watchpost && make build && ./dist/watchpost
```

Your own config this time — the point is the journey a listener takes. Nothing here can lose a setting: a
save preserves keys it does not know, and you can revert any choice from the same window.

---

## 1. The cast — the thing the release is for

Press **`V`**. Setup opens **at the correspondents**, not at a chooser (the `[V]` chooser is gone).

- `space` on **Correspondent Cast**, then `↓` to **Alerts / Takeovers** and `←→` to pick a voice.
- `p` previews the focused one. On a voice you do not have, the **first** `p` offers the download and the
  **second** does it — it must never pull 63 MB because you pressed a key to hear something.
- `↓` to **All Reports** and pick a *different* voice.
- `enter` on the last row of the group saves.

Tune the radio (`space` on a location) and **listen to a full cycle**.

**Pass:** the alerts read in one voice and the reports in another, and at each boundary the incoming
correspondent introduces themselves — *"This is Rishi, taking over for Samantha."* No gap, no splice.

**Judge:** does the hand-over sound natural, or intrusive at every boundary? Is the wording right? Both are
yours (`scripts/handover/line.txt`).

---

## 2. The tones — the part most likely to be wrong

Tab to **ALERTS - TONE**. Six classes; `space` ticks one, and ticking implies Mute.

The presets sound different from each other for the first time, and **every one of them is your pass**:

| Class | Sound |
|---|---|
| Significant Quakes & Disasters · Warnings | dual-tone, 853 + 960 Hz, 2 s (the EAS-style attention signal) |
| Watches | 1050 Hz, 2 s |
| Advisories | the classic tone, unchanged from 0.13.0 |
| Special Statements | soft chime, 880 + 1760 Hz, decaying |
| Maritime (tropical / winter storms) | low sweep, 330 → 520 Hz, twice |

**Judge:** is each one right for what it precedes? Is the dual-tone too much for a routine warning? Is the
soft chime audible enough over a ducked broadcast? Are the lengths right — a warning's words land about
1.2 s later than an advisory's, by design.

**Also check:** `[M]` mutes **tones only** now. Press it and trigger a takeover: you should hear no tone and
**still hear the words**. That is the accepted change from 0.13.0 (MVS-D-34) and it is the one a listener
upgrading will notice.

---

## 3. Setup at both sizes

At **133×44** the window is two columns, as the mock draws it. Resize to **80×24**: it stacks, it scrolls,
and the chip row stays **pinned** at the bottom.

**Pass:** whatever row you focus, you can see it *and the lines under it* — its hint, its value, its note.
Walk `↓` from the first row to the last at 80×24 and watch nothing disappear.

**Judge:** the DATA rows now read `Default location: …` and `NASA FIRMS key: …` (the mock's words, OP-2).
The mock's two misspellings ship **corrected** — "Maritime Report", "Fire/Hotspots". Say if you wanted them
as drawn.

---

## 4. The Radio panel, three ways

Resize wide (≥146 cols), medium (≥84), narrow (<84).

**Pass:** wide has the station name and a three-row visualizer; medium has one; narrow has none — and the
`v` control **disappears with it**, because there is nothing to toggle. `[T]` is gone everywhere; each width
has a standard height.

---

## 5. `[S]` — who is reading, and why

Press **`S`**. Three new blocks: **CAST** (role · speaks · via · why), **TONES**, **CONFIG**.

**Pass:** every role has a row, the inherited ones say so, and any voice that lost says *why* — "unknown on
this host", "not installed". CONFIG names anything in your config this build could not use.

Then, from a shell:

```sh
./dist/watchpost report --report-only --verbose "<your location>" | sed -n '/^CAST/,$p'
```

**Pass:** the same answers, no dashboard needed.

---

## 6. The maritime report — coastal only

Tune a **coastal** location and listen past the forecast.

**Pass:** the sea is read between the forecast and the fire report — coastal-waters forecast, the nearest
buoy, the tide in force, the next high and low, the current. An **inland** location reads none of it (not
"no data available" — the sea is not applicable there).

**Judge, and expect to have opinions:**
- **Length.** This is the longest section in a cycle. A coastal location with fire and seismic can run past
  four minutes before the sign-off. If it drags, the two levers are the number of forecast periods (3 today)
  and the section's own cap (12 segments), in that order.
- **The words** are yours (MVS-D-21): "above the low-water mark", a 12-hour clock, wind in your units,
  currents always in knots.
- **Tide times read in the LOCATION's zone, not yours.** Correct — a tide happens where the water is — but
  surprising on a watchlist spanning coasts. Tell me if it should follow you instead.

---

## 7. Nothing you had is worse

- A config with no cast sounds **exactly** as 0.13.0 did, apart from the class tones.
- `[V]` still opens something useful; `t`, `M`, `w`, `S`, `?` unchanged.
- If you rebound `T` in `[keys]`, the app still starts — the binding is ignored with a note in `[S]`'s
  CONFIG block rather than refusing to launch.

---

## Recording it

Pass/fail per step, and separately any **rulings** — a tone, a wording, a label, the report's length. I
expect rulings here; that is what this step is for, and every one of them is cheap to change now.

**Still outstanding after this:** M3's two listening trials (`gates.md` §2 → `07-readiness/validate/m3.md`)
and the Linux protocol (`linux-validation-protocol.md`), both yours, both before SHIP.
