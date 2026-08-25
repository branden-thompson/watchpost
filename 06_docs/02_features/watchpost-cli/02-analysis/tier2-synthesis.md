# DISCOVER Tier 2 — Cross-Cutting Synthesis

Date: 2026-08-23 · Inputs: `research/AI-3`, `AI-9`, `AI-10`, `AI-11` (+ Tier 1) · Phase: DISCOVER (SEV-0, HUMAN LEAD)

## A. Composition effects

1. **Fire panel = three sources, one pattern (AI-3 + AI-1 + AI-2).** HMS hotspots (keyless, slow) + FIRMS (keyed, fast) + WFIGS incidents (keyless, names/containment) + NWS Red Flag alerts compose into one "fire" block. This is the same *keyless-default / keyed-upgrade* shape as weather (NWS+Open-Meteo default, Pirate/OWM upgrade) — the provider interface from Tier 1 #14 covers fire unchanged.
2. **The JSON contract absorbs every Tier-1/2 schema (AI-10 + AI-1/2/3/4).** `harmonized / by_provider / diffs` per location plus `alerts[]` (CAP names), `fire{hotspots[]}`, `radio{}` — every data domain discovered has a slot. The `Snapshot` struct is the single architectural pivot: providers write it, TTY reads it, JSON marshals it, goldens pin it (M5).
3. **Glyph hazards meet the upstream constraint (AI-9 + AI-6).** Emoji are excluded from aligned layouts (runewidth under-reports VS16/pictographs); blocks/arrows/box-drawing are ambiguous-width under CJK locales. Therefore every new go-studs chart component needs an ASCII fallback mode *in the component* (upstreamable, generic), driven by one `--ascii` / `TERM=dumb` / EAW switch — not per-view hacks.
4. **Frame policy + M9 + go-studs tick (AI-9 + AI-6 + M9).** bubbletea v1 short-circuits unchanged frames; one 100 ms tick while animating (AI-9 §3: 10 fps ≈ ≤1 % CPU; 20 fps ≈ 2 %), 0 fps idle ≈ 0 %. go-studs' own `AnimationController` (50 ms goroutine ticker) must NOT be used — drive `NextFrame()` from the single `tea.Tick`. This is now a hard BUILD rule.
5. **Evac-orders path is already in NWS (AI-3 + AI-1).** NWS CAP relays `Evacuation Immediate` events; since the alert pipeline exists in v0.1, R-9b (evac, v1.x) becomes an *event-type filter + UI treatment*, not a new provider. Cross-check against AI-1 event list at PLAN.

## B. Convergence

6. **"Keyless default, keyed upgrade" is now the universal provider posture** — weather (AI-2), geocoding (AI-8), fire (AI-3), radio (AI-4: public mounts, user-supplied URL override). `watchpost setup` has one consistent story.
7. **Two independent agents (AI-9, AI-10) converge on "stdout mode = width once, no SGR, auto-detect when piped, hint on stderr."** Matches OQ-12 and the go-studs `GetTerminalSize()` finding.
8. **Public-domain/open data everywhere except the radio relays and go-studs.** NWS, HMS, WFIGS, FIRMS (attribution), GeoNames/Open-Meteo (CC-BY) are all shippable. The only licensing exposure is RS-12 (volunteer streams) and RS-1 (go-studs).

## C. Contradictions

9. **RS-1 is worse than the brief assumed (AI-11 vs OQ-4).** OQ-4 ("start local, decide later") assumed the decision could be deferred to PLAN exit. Facts: repo is in a private enterprise GitHub behind an IP allow-list; README says "Internal the employer library / contact the the upstream maintainers" while LICENSE says "Copyright Branden Thompson"; all commits from `@private.com`; extracted from the reference CLI; a multi-author the employer fork exists with no LICENSE. And `go install` ignores `replace`, so any public watchpost built on (a)/(b)/(d) is **not installable by anyone else**. **Resolution proposed:** keep OQ-4's local `replace` for BUILD, but (i) keep the watchpost repo private until the four AI-11 questions are answered, and (ii) add a PLAN-exit gate "go-studs distribution mechanism decided" that blocks SHIP. → **OQ-17**
10. **HMS as keyless default vs. "alerts are especially important in fire regions" (R-9 intent).** HMS is hours-delayed; a user in a fire zone without a FIRMS key gets worse hotspot latency than the app's alert latency target. Not a contradiction with any ruling, but a UX obligation: `watchpost setup` should prompt for a free FIRMS key when a configured location is in a `fireWeatherZone` with active Red Flag history. → BUILD requirement; ruled as **OQ-19** (§E). 
11. **AI-10 exit code 3 (`--fail-on-stale`) vs. T-C "simple stdout mode".** Minor scope creep. Recommend ship 0/1/2 in v0.1; defer 3 and `--watch` JSON-Lines to v0.2 unless HUM LEAD wants agent streaming in v0.1. → **OQ-18**

## D. Risk signal status (delta from Tier 1)

| RS | Status |
|---|---|
| **RS-1 go-studs IP/access** | **ESCALATED → High (blocking SHIP, not BUILD).** Facts in AI-11 §1–4. |
| RS-3 feed availability | Fire: resolved (HMS/WFIGS keyless). Radio: unchanged (Medium, best-effort). |
| RS-6 TTY/JSON drift | Closed by design — `Snapshot` + bidirectional golden + reflection test (AI-10 §3). |
| RS-8 perf | Mitigated — frame policy (AI-9 §3) gives arithmetic inside M9; spike still recommended. |
| RS-10 alert trust | Unchanged — stale warnings now first-class in JSON `warnings[]` (AI-10). |
| **RS-14 (new) glyph-width drift** | Medium — runewidth vs terminal disagreement on emoji; CJK ambiguous width. Mitigation #3. |
| **RS-15 (new) HMS latency in fire zones** | Low — mitigation #10. |

## E. Open question status

| OQ | Status | Ruling (HUM LEAD, 2026-08-23, verbatim) |
|---|---|---|
| OQ-13 (OQ-3 reinterpretation) | **RULED** | "Acknowledge" — watchpost adopts current bubbletea directly; go-studs unchanged for v0.1. |
| OQ-14 (contact wxradio operator) | **RULED** | "Defer" — no operator contact for now; revisit at SHIP readiness. |
| OQ-15 (attribution) | **RULED** | "Accept in About frame/modal, not footer" — attribution (Open-Meteo, GeoNames, NASA FIRMS, NOAA/NWS) lives in the About view and `--about`; footer stays clean. |
| OQ-16 (cities15000 vs 5000) | **RULED** | "Yes + online fallback" — embed cities15000 + Open-Meteo online fallback. |
| OQ-17 | **RULED** | "Yes - keep repo private" — repo private until go-studs distribution decided; SHIP gate added. |
| OQ-18 | **RULED** | "Defer to 0.2" — v0.1 stdout = `--json`/`--report-only`, exit 0/1/2. |
| OQ-19 | **RULED** | "Yes" — HMS keyless default + WFIGS + optional FIRMS key prompted for fire-zone locations. |
| AI-11 Q1–Q4 | **ANSWERED/DEFERRED** | "I am the author of go-studs, and even have the first version in my branen-thompson repo. We can defer/evaluate if we should simply leverage bubbletea, or fork to create components that can be leverage later. As long as we architect things to make that changeover/pivot easy, it becomes a secondary concern." → **D-9 architectural requirement:** the component/render layer must sit behind an internal abstraction so the go-studs dependency can be swapped (fork under branden-thompson, bubbletea-native, or upstream) without touching views. RS-1 downgraded to **Medium-deferred**, resolved-by-architecture + SHIP gate. |

## F. Implications for DISCOVER exit / PLAN

12. **DISCOVER research is complete.** All 11 AI items have evidence-backed reports; no further waves needed. Remaining DISCOVER work is HUM LEAD rulings (OQ-13..19 + AI-11 Q1–Q4), red-team, and the Discovery Report.
13. **PLAN inputs are concrete enough for FULL PLAN depth:** provider interface (4 domains), `Snapshot` contract, scheduler tiers, cache subsystem, audio pipeline, geocoder, view registry, frame policy, glyph/ASCII policy, breakpoints (40/60/80/120), config (0600, XDG), exit codes, schema versioning.
14. **Two pre-BUILD spikes recommended:** (S1) oto+go-mp3 radio playback CPU/RSS; (S2) embedded GeoNames index memory footprint with a compact encoding — both convert estimates into M8/M9 evidence before architecture locks.
