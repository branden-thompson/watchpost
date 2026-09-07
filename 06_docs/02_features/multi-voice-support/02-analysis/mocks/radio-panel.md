# Radio panel mock — HUM LEAD, 2026-08-29 (MVS-D-23; reproduce exactly, colour is the HUM LEAD's pass)

Retires **both** the `[V] Voice` and `[T] Size` controls: every breakpoint has a **standard vertical size**, so
the size toggle goes with the voice chooser. Three breakpoints as drawn (146 · 84 · 66 columns outer).

## Wide (≈146 columns) — header · 4 inner rows (visualizer, toggleable, + marquee) · controls

```
┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│    WATCHPOST WEATHER RADIO • ♪ EVENT · Special Weather Statement · Palomar Mountain, CA       VOL  - ███████████░░░░░░░░░ +   55   ▶ PLAYING   │
│   │                                                                                                                                         │  │
│   │                                                       VISUALIZER AREA (toggleable)                                                      │  │
│   │                                                                                                                                         │  │
│   │                                                             < marquee area >                                                            │  │
│     space  Play   r  Repeat: Off   m  Mode: Synth   v  Viz: Off                                                                                │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Medium (≈84 columns) — header · 2 inner rows · controls; the product as its code (SWS)

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│   ♪ EVENT · SWS · Palomar Mountain, CA    VOL  - █████░░░░░ +   55   ▶ PLAYING   │
│   │                     VISUALIZER AREA (toggleable)                         │   │
│   │                           < marquee area >                               |   │
│     space  Play   r  Repeat: Off   m  Mode: Synth  v  Viz: Off                   │
└──────────────────────────────────────────────────────────────────────────────────┘
```

## Narrow (≈66 columns) — header · marquee only · keys only; the place as its ZIP (corrected by the HUM LEAD 2026-08-29: no `[ v ]` — there is no visualizer to toggle — and no `[ T ]`)

```
┌────────────────────────────────────────────────────────────────┐
│   ♪ EVENT · SWS · 92057     VOL  - ███░░░ +   55   ▶ PLAYING   │
│   │                    < marquee area >                    |   │
│   [ space ] [ r ] [ m ]                                        │
└────────────────────────────────────────────────────────────────┘
```

## Reading of the mock (for PLAN P4; open points below)

- **Header row** = station name (wide only: `WATCHPOST WEATHER RADIO •`) · the on-air detail (`♪ EVENT · <product> ·
  <place>`, the product as its code at medium and narrow, the place as its ZIP at narrow) · a **volume bar**
  (`VOL  - <bar> +   <level>`; bar 20 · 10 · 6 cells) · the state (`▶ PLAYING`).
- **Inner box** (`│ … │` rows): the visualizer area (toggleable, 3 rows wide / 1 row medium / none narrow) and
  the marquee row.
- **Controls row**: bare keys with labels (`space  Play   r  Repeat: Off   m  Mode: Synth   v  Viz: Off`) at wide
  and medium; **keys only, bracketed** at narrow (`[ space ] [ r ] [ m ]` — no viz toggle without a visualizer).
- **Vertical size is fixed per breakpoint:** 8 rows wide (6 content + 2 border) · 6 medium · 5 narrow.
- **The `v` control exists iff the breakpoint has a visualizer area** (HUM LEAD 2026-08-29: "smallest size won't show viz
  so the [v] control disappears; once it gets wider it can reappear") — the control row is derived from the
  breakpoint, never from a separate setting.

## Open points for the HUM LEAD (asked 2026-08-29)

1. ~~The narrow row's `[ T ]`~~ — **corrected by the HUM LEAD: `[ space ] [ r ] [ m ]` only.**
2. **FR-10 (the on-air correspondent)** — **ruled 2026-08-29: dropped from the panel entirely for now** (MVS-D-24).
   The speaking correspondent is known from the spoken identity lines and the `[S]` cast table; the panel shows
   no voice name.
3. The volume control — **the `-`/`+` keys, exactly as today** (MVS-D-24); the header bar is their display.
