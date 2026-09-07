# Alert tones — the ratified presets (brief A-7 · R-12 · MVS-D-9 / MVS-D-11)

Eight candidates were synthesized on 2026-08-29 with the app's own tone parameters (`domains/radio/synth/tone.go:14-23`:
22 050 Hz mono, amplitude 0.45, 8 ms envelope) and auditioned by the HUM LEAD. Five were kept and assigned;
three were dropped ("too happy / like a phone ring"). The generator is reproducible (§3) so PLAN can port each
preset into `tone.go` as parameters, not samples.

## 1. The presets and their classes

| Preset | Class it precedes | Style it follows | Parameters |
|---|---|---|---|
| **Dual-tone** | **Significant Quakes & Disasters** (earthquakes, tsunamis) *and* **Warnings** (Tornado / Severe Thunderstorm / Flash Flood …) — two classes, separately mutable (MVS-D-28), one preset | the Emergency Alert System attention signal | 853 Hz + 960 Hz summed (÷2), sustained 2.0 s |
| **1050 Hz** | **Watches** | NOAA Weather Radio's warning alarm tone (the *style* is borrowed; the HUM LEAD assigned it to Watches deliberately — MVS-D-11) | 1050 Hz sustained 2.0 s |
| **Classic** | **Advisories** | today's Watchpost tone (unchanged) | 1000 Hz, three 200 ms pulses, 100 ms gaps |
| **Soft chime** | **Special Weather Statements** | a public-address chime | 880 Hz + 1760 Hz, 1.2 s, amplitude 0.40, exponential decay τ = 0.35 s |
| **Low sweep** | **Tropical / Winter Storm — named maritime storms** (the Setup label "Maritime") | a horn-like signal | linear sweep 330 → 520 Hz over 0.7 s, 0.2 s gap, repeated once |

Dropped: rising two-tone (660→880 Hz × 3), descending two-tone (880→660 Hz × 2), five-note quake chime.

Every preset keeps the 2 s pre-narration tail that `AlertTone` builds into its buffer (`tone.go:56`). The
**signal lengths differ** — Classic 0.8 s · Soft chime 1.2 s · Low sweep 1.6 s · Dual-tone and 1050 Hz 2.0 s — and
`s.hold(s.attention())` (`app/ticker.go:259`) holds for the whole buffer (`app/radio.go:379`), so a Warning
takeover's first words land ~1.2 s later than today's, by design (red-team D-8). NFR-2 therefore measures
time-to-**tone-start**, and records the per-class offset as deliberate. Each class can be **muted** in Setup (MVS-D-26 — the share switch of MVS-D-9 is dropped; every class always
sounds its own preset); `[M]` mutes them all, tones only; the sound of every preset remains the HUM LEAD's pass.

## 2. Classification (the rule the classifier implements — OQ-16 resolved: storm wins, MVS-D-15)

The classifier takes the **product string** (it must serve both the breaking path, which holds
`globalfeed.Event`, and the severe read, which holds `tty.SevereRow` — `voice-architecture.md` A-3.2):

| Class | Products |
|---|---|
| disaster | significant earthquakes (the ticker's quake class), tsunami products — the Setup label "Significant Quakes & Disasters" |
| storm | tropical cyclones (NHC), Hurricane / Tropical Storm Warnings and Watches, Winter Storm Warnings and Watches, Blizzard Warnings — *Blizzard is RAT-3, pending E-7* *(the ruling's wording: "Tropical / Winter Storm — named maritime storms"; storm wins over warning/watch — MVS-D-15; the Setup label "Maritime")* |
| warning | any other product ending in "Warning"; **and any product string no rule matches** (the loud default) |
| watch | any other product ending in "Watch" |
| advisory | products ending in "Advisory" (Wind Advisory, Winter Weather Advisory, Heat Advisory, …) |
| statement | Special Weather Statement (and other "Statement" products) |

## 3. Reproducing the candidates

```python
RATE, AMP, ENV = 22050, 0.45, 0.008
def tone(freqs, secs, amp=AMP, decay=None):   # summed sines ÷ len(freqs), 8 ms linear envelope, optional exp decay
def sweep(f0, f1, secs):                        # linear chirp, same envelope
dualtone  = tone([853, 960], 2.0)
wat1050   = tone([1050], 2.0)
classic   = tone([1000], .2) + gap(.1) + tone([1000], .2) + gap(.1) + tone([1000], .2)
softchime = tone([880, 1760], 1.2, amp=.4, decay=.35)
lowsweep  = sweep(330, 520, .7) + gap(.2) + sweep(330, 520, .7)
```
(the full script that produced the auditioned WAVs is in the session scratchpad; the parameters above are the
contract.)
