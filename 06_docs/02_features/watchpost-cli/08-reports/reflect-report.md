# REFLECT Report — Watchpost CLI 0.9 (After Action Report)

| Field | Value |
|---|---|
| Phase | REFLECT (li-A2DH: SHIP → **REFLECT**) · SEV-0 · HUM LEAD |
| Covers | DISCOVER → PLAN → BUILD (B0–B5) → REVIEW → VALIDATE → SHIP, 2026-08-23 → 2026-08-25 |
| Outcome | `watchpost` 0.9.4 public on GitHub, MIT; 123 UAT sessions; ~190 commits on the working branch; 9 phase reports, 3 red-team reports, 128 red-team findings dispositioned |

## What was set out to do, and what happened

The plan of record (PLAN, D-19/D-20) was a terminal weather station driven UX-first from mocks, with
gates scaled to SEV-0 and a Linux install that works from one line. That shipped, plus two things the
plan had for 1.0 that HUM LEAD pulled forward at the BUILD exit: fire data (HMS/WFIGS/FIRMS) and the
Fire and Hotspot report on air. The record departs from the plan in three places, all recorded in
architecture §11.9: setup moved from a wizard into the dashboard (UAT 100), the fire tiers run at one
cadence instead of two, and the keyless global (non-US) provider stays a 1.0 item.

## What went well

- **UAT-first with exact mocks.** 123 sessions, each a table of "what was asked → what shipped →
  what pins it". Fit-and-finish at the end (masthead, empty states, the ◆ marks block, the Setup
  form) went in hours because the render path was pinned and the mocks were literal.
- **Red-team at every exit, refute-before-report.** Three rounds, 128 findings, 1 Critical (the HMS
  parse repeated once per RECENT scheduler — 4.5 GB per tick — found by measurement, not by reading).
  The lenses that measured (parse cost, 50-scheduler launch shape, FIRMS 400 probed live) found the
  real defects; the lenses that read found the doc drift.
- **Fix-forward after publish.** Four point releases in a day from the Linux run, each green first
  time, each one paragraph in the CHANGELOG — because the pipeline, the gates and the record already
  existed.
- **Modularity rules held.** Extract at the second caller; single-owner knobs (`fireRules`,
  `Config.FireBoldMW`); controls live where they act. The P10 gate stayed at 0 live findings across
  the whole exit with exemptions recorded, not waved.

## What went badly, and what changed

- **The release pipeline had never run on a tag.** Three CI-only failures before the first publish
  (learning §7). Changed: race-aware timing constants; `lint-watermark` tolerates a detached checkout;
  `GOOS=linux go vet` before tagging.
- **Identity drift cost an hour.** A managed `~/.ssh/config` wiped a block; the personal key dropped
  out of the agent; pushes went out as the wrong account until pinned (learning §8). Changed: the
  repo pins its key with the ambient config ignored; identity is verified before every outward step;
  all of it is in the project memory so it is not rediscovered.
- **Two docs promised more than the code did** at REVIEW (a `y`/`n` Setup step that no longer
  existed; a schema whose `$id` was dead) — caught by the docs and contract lenses, not by the author.
  Changed: `make schema` + a byte-compare test; the docs-vs-code lens is a standing REVIEW step.
- **Validation found only "second-machine" bugs** — the voice list, the silent wait, the glyphs
  (learning §9). Nothing was wrong on the build machine; everything was wrong for a stranger. The
  Linux protocol is now written for the current build and Arch is named in it.

## Debt carried (named, with triggers)

| Item | Trigger |
|---|---|
| Schema `1.0.0-rc` → `1.0.0` | HUM LEAD ratification; additive changes only until then |
| `--ascii` / `--no-animation` flags (render path exists, no CLI flag) | First report of tofu from a real terminal |
| Keyless global (non-US) provider | 1.0 plan |
| `report --every` (screen-reader live surface) | First 1.0 item |
| Fire spread direction | A feed that gives two observations of one fire |
| User theme files validated for raw-SGR shape | First user theme file that renders wrong |
| THIRD_PARTY_LICENSES over-includes test-only modules | Cosmetic; regenerate from `go list -deps` when convenient |

## Numbers

Warm launch → full view 550 ms; cold 1.1 s. Radio soak: RSS 142–221 MB oscillating over two hours,
no trend; CPU 1–8 %; ~30 threads. HMS archive: 1.46 MB, 27.5k points, one parse per change
(~120 ms). Release assets: five binaries, 15–17 MB each. Tests: `go test -race ./...` green; live
`--json` validates against the published schema.

## Source Documents

Every phase report under `08-reports/`, the UAT log (`04-development/b3-uat-log.md`), the infra and
debugging ledgers, `06-key_learnings/b3-ux-backwards.md` §1–§10, `07-readiness/`.
