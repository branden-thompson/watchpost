# Quality gates — multi-voice-support (0.14.0, SEV-0)

The gates every batch and the release pass, in the order they run; one row is filled per batch at its exit
(`04-development/p{n}-build-log.md` holds the detail). **This table owns the commands; the batch checklists copy
them.** `make p10` needs the framework build of the CLI (`A2DH=<framework build>`; the installed CLI has no
`p10`) — the path lives in the developer's shell, never in the tree.

## 1. Per-batch gates (P1–P4)

| Gate | Command | Pass rule |
|---|---|---|
| Unit + race | `go test ./... -race -count=2 -timeout 600s` (the batch's packages at minimum; P4 runs the whole tree; the timeout turns a hang into a failure) | green; no `-race` report |
| Verify | `make verify` (fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls) | green — `lint-imports`: no `modes/` → `domains/` import; `lint-watermark`: no attribution trailers in `main..HEAD` |
| Harness | `a2dh validate` | every check passing or skipped-with-declaration (NFR-7). The harness has **18** checks; a bare "17/17" predates the eighteenth |
| Alloc pins | `make alloc-budget` (CI runs it too, `.github/workflows/ci.yml:33`) | every pin holds |
| Lint | **Ruled (MVS-D-45, E-11): the lint gate is what `make verify` already runs** — `fmt vet tidy vuln race lint-imports lint-watermark gate-controls`. `golangci-lint`/`staticcheck` are **not** gates: neither has ever run here, and the former is red on one pre-existing `QF1001` outside this feature (plus `third_party/`, which every other gate excludes and its config does not). Adding a `make lint` target is a carried follow-up, not 0.14.0 work. | `make verify` green |
| P10 | `A2DH=<framework build> make p10` → `cp dist/p10.json 06_docs/02_features/multi-voice-support/07-readiness/p10-p{n}.json` — run **after** staging everything but the record and the build log, so the `tree_hash` is the commit's | 0 unmatched findings; ledger rows added only from the run, each presented for HUM LEAD ratification in the build log; the run's `tree_hash` and the CLI's version + commit recorded below beside the commit (the ledger's symbol keys depend on the CLI's key scheme — one build for the whole release) |
| Declsets | `go test <pkgs> -run DeclarationSet -update-declset` — the packages whose exported top-level set moves: app P1–P4 · render P3 · snapshot P3 · nws P3 · tty P4 (`platform/term` and struct-field additions move no declset) | re-captured at the gate, reviewed in the diff, never mid-batch |
| PTY | `make pty-severe` (P2–P4); `make journey` (P4 — a fresh HOME, the real binary, real feeds; it was a command to remember until 2026-09-06, and a gate nothing runs is not a gate)
| Docs | `go test ./cmd/... -run WhereThingsHappen`; the NFR-8 grep (P4) | green; zero hits |
| Goldens | `go test ./modes/tty -run Golden` (P4; the frame, severe and the three Setup goldens re-recorded once with `-update-golden`, reviewed row by row against the mocks); `go test ./domains/radio/synth -run Golden` (P2's composed-cycle golden, the same `-update-golden` flag) | green |
| Build | `make build` → `./dist/watchpost` | the batch's UAT-able behaviour observed by the HUM LEAD |

### Batch record

*The p10 run's own record is `p10-p{n}.json` — it carries the tree hash, the tool set and every
finding. A hand-copied hash beside it was a second place to be wrong, and was: the P1 row recorded a
value that is not a git object and does not match `p10-p1.json`. The column is gone; cite the JSON.*

| Batch | Commit | CLI (`a2dh version`) | Ledger rows ratified | Build log |
|---|---|---|---|---|
| P1 | `8964e6f` | v1.17.0 (7960f0e) | 4 added, **ratified 2026-08-30** (2× P10-06 immutable tables, 1× P10-05 package density, 1× P10-06 `runtimeGOOS` seam) | `04-development/p1-build-log.md` |
| P2 | `6565bb6` | v1.17.0 (7960f0e) | 2 re-keyed (narrate.go → director.go), 1 deleted as dead | `04-development/p2-build-log.md` |
| P3 | `c8851a4` | v1.17.0 (7960f0e) | 0 added | `04-development/p3-build-log.md` |
| P4 | `e31f8a5` | v1.17.0 (7960f0e) | **0 added** — six live findings all fixed | `04-development/p4-build-log.md` |

| P5 (UAT + perf + red team) | this batch | v1.17.0 (framework build) | **1 added** — `app/release.go:start`, the allowlisted poller shape; presented for ratification | `04-development/p5-build-log.md` |

## 2. Release gates (before SHIP)

| Gate | Evidence | Owner |
|---|---|---|
| M1 ROLE — every role reads in its resolved voice | `TestBroadcastSectionsSpeakInTheirRolesVoices` (P2 2.8) + `TestDirectorForwardsTheRoleToRenderAndTheClassToTone` (2.6) | agent |
| M2 A2V — the keypresses from the dashboard to "role X speaks in voice Y" | `modes/tty:TestM2TheKeypressesToAssignACorrespondent` — **six** (V, down, space, down, right, esc), against the ≤ 11 pin (AM-18, ratified **MVS-D-40**) and the project brief's target of 5. **The 11 was never measured**: the journey script counted its own `send` calls and asserted 11 ≤ 11, and the path it sent never reached a save at all — Enter ADVANCES from a picker row; esc is what writes. Measured in the Update loop rather than a PTY, which cannot tell a closed window from a late redraw (F-6) | test |
| M3 EAR (AM-17, ratified **MVS-D-39**) | **two trials**, each recorded in `07-readiness/validate/m3.md` with who built the play order: (1) fresh install — *name the class from the tone*, 0.14.0 only, pass ≥ 9/10 (this measures R-12's class tones; 0.13.0 has one tone and cannot be compared); (2) configured cast — *alert vs report with the tone stripped* (audio starts after the tone, the brief's clause), **A/B against the 0.13.0 binary**, order shuffled by a second person or a script the listener does not build, ten per arm; pass = 0.14.0 ≥ 9 **and** 0.13.0 ≤ 7; anything between → E-3 rules on the problem statement's standing (n = 10 cannot separate 9 from 8) | HUM LEAD |
| M4 R6 · PERF — the tone path's regression pin (the benchmark, both trees, ≤ +50 %) and the live budget (`breaking:` → `tone:` ≤ 250 ms, root installed and fresh Linux HOME); app RSS ≤ 126 MB no trend; summed `piper` RSS ≤ 0.13.0's × 1.10 and ≤ (N + 2) × the §3 number; the writer never starves (`TestWriterNeverStarvesAcrossHandOvers`, `TestInvalidateNeverRendersOnTheWriter`); no per-tick misses — all per `perf-protocol.md` §1/§4, the one owner | `perf-protocol.md` §0–4; the 1-h soak on a coastal cycle (P3 3.7) both platforms | agent (macOS) · HUM LEAD (Linux) |
| M5 FALL — every fallback row of the matrix explicit, never silent except the one legitimately silent row (AM-19, ratified **MVS-D-41**) | `TestResolveFallbackMatrix` (P1 1.4); `watchpost report --verbose` on a host with nothing installed (**MVS-D-47**: the `[S]` cast table was retired at UAT; this is the surface that carries those answers) | agent |
| NFR-8 | the grep is zero over code, help, README, docs | agent |
| Linux | `07-readiness/linux-validation-protocol.md` all rows | HUM LEAD |
| Release checklist | `07-readiness/release-checklist.md` all boxes | agent + HUM LEAD |
