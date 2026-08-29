# Readiness — gate ledger (0.13.0 severe-alerts-modals)

Every command below was executed on 2026-08-29 at the REVIEW exit (branch `feature/severe-alerts-modals`, at `df59528` —
after the round-5 remediation, the rulings and the Light theme correction); the result column is what it printed. Rewritten at every exit (R4-C-02). **The
allocation budgets have one owner: `modes/tty/bench_test.go`** — every other number in the documents is
superseded by it.

| Gate | Command | Result |
|---|---|---|
| Harness validate (100 % required at every exit) | `a2dh validate` | 18/18 |
| Format · vet · tidy · vuln · race · imports · watermark · controls | `make verify` | `verify: ALL GATES GREEN` |
| P10 rules (the installed `a2dh` predates `p10`; point `A2DH` at a current build; the config and the ledger are local by HUM LEAD ruling, C-07) | `A2DH=<path-to-current-a2dh> make p10` | `p10: 0 live, 0 unmatched` — 17 findings in the diff against `merge-base:main`, all exempted; the report is committed as `07-readiness/p10-build.json` |
| P10 ledger hygiene | `scripts/quality/p10-unmatched.sh dist/p10.json` | every in-scope ledger entry matched a finding (104 dormant rows outside the diff) |
| Cross-platform vet | `GOOS=linux go vet ./...` | clean |
| Allocation budgets (no race) | `go test ./modes/tty -run AllocBudget -v` | frame 133×44 hit 655 / miss 7 102 · 133×70 705 / 13 859 · 200×60 713 / 17 325 · **80×24 964 / 3 238** (pins 962·1.05 / 3 598·1.05) · severe window hit 2 031 / miss 7 637 (pins 2 521 / 7 939) — all within pins |
| COV metric (FR-5: the frozen render list) | `go test ./domains/globalfeed -run TestRenderListCoverage -v` | quake 11/11 · storm 13/13 · warning 11/11 (the `MoveDirDeg` predicate corrected, R5-A-03) |
| Statement coverage (`-cover`) | `go test -cover ./domains/globalfeed ./domains/severe ./app ./modes/tty ./platform/render` | globalfeed 92.5 % · severe 92.0 % · app 53.0 % · tty 91.6 % · render 85.9 % |
| Race, twice, on the touched packages | `go test -race -count=2 ./app ./domains/... ./modes/tty ./platform/...` | green (the R5-B-01 race probe is now a test) |
| Schema | `make schema` then `git diff --exit-code pkg/schema` | regenerated at P1 (`sender_name`); clean since |
| PTY machine-verify | `make pty-severe` | `pty-severe: ok (w opens, right moves to Watches, enter/esc/esc, w re-opens, esc, ctrl+s opens, esc, q quits)` |
| Live feeds smoke (network) | `WATCHPOST_LIVE=1 go test ./domains/globalfeed -run TestLiveGlobalFeeds -v` | green at the round-3 exit; re-run owed at VALIDATE (the parsers changed: `clampID`) |
| Goldens (byte pins with width invariants) | `go test ./modes/tty -run 'Golden'` | seven severe goldens and three frame goldens; re-recorded this phase for the centred WX STN / ZIP cells and the column-title row |
| Declaration sets | `go test ./... -run DeclarationSet` | re-captured where intentional (`app`, `domains/radio/{player,script,pronounce,synth}`, `domains/severe`, `domains/globalfeed`, `domains/weather/nws`, `modes/tty`, `platform/{render,snapshot,plaintext}`) |
| Radio in the diff (NFR-7 superseded at UAT) | `git diff v0.12.0 --stat -- domains/radio` | 50 files changed, +1 244 / −171 — the engine (held/aside/audition lines, the tap tee), the composer (scripts), the pronunciation tables, the diagnostic |

## Owed — with an owner and a phase

| Gate | Why not here | Owner · when |
|---|---|---|
| **R6 relay + audio smoke — BLOCKING** | The radio engine, composer and voice path changed at UAT and REVIEW: a real relay and a real synth cycle to the sign-off, a `[space]` read paused by a takeover and resumed, an audition during a read, `[+]`/`[-]` during a read, quit during a read (`linux-validation-protocol.md` §2) | HUM LEAD at VALIDATE, macOS and Linux |
| 1-hour PPROF soak (RS-8, plan 4.5) | done at VALIDATE 2026-08-29: no trend (`validate/soak-1h.csv`) | ☑ |
| Linux halves of `make verify` (race) and `make pty-severe` | macOS machine | `linux-validation-protocol.md` §1 / Linux CI with the release PR |
| Live feeds smoke | done at VALIDATE 2026-08-29 (192/192 NWS features decode) | ☑ |
| README screenshots | HUM LEAD captured 2026-08-29 (14 frames; two animations built with ffmpeg) | ☑ done |

## Bounds worth restating

- `seen.json` (the ticker's announced-id store): written `0700`/`0600`, re-tightened on every save, capped at 20 000 ids.
- Severe index: 500 rows after the sort; every text field bounded at the parser (120 runes short fields, 200 for ids,
  4 000 prose, 50-entry lists); every string that reaches a frame passes `platform/plaintext` (labels once at the
  assembler; provider prose at its seam); an override script part ≤ 64 KB.
- The engine's preview ceiling is 12 000 × 50 ms of PLAY (a held line spends none of it); a render is bounded at 2 min;
  a `[space]` read is cancelled and waited for at shutdown (bounded 2 s).
- The modal memo is one slot; its key is 60-odd bytes plus a `[32]byte` stats fingerprint only while `[S]` is open.

## P10 ledger — the scope rule, and the rows to ratify (HUM LEAD)

With FULL GIT (local `main` = 0.12.0), the P10 scan covers the release diff; the ledger's rows for code outside the
diff are **dormant** (104), not dead — `scripts/quality/p10-unmatched.sh` says so and still fails on a dead in-scope
row. The config and the ledger stay local (HUM LEAD ruling, C-07).

**Ratified by HUM LEAD 2026-08-29** (the four refreshed reasons applied to the local ledger). The rows — every one of the 17 exempted findings (the reasons are in the local ledger; four predate the code
they absorb and are refreshed at ratification — R4-C-06; the `platform/plaintext` row was added at REVIEW):

- `app/dashboard.go`
- `app/narrate.go:168`
- `app/narrate.go:230`
- `app/pipelines.go:188`
- `app/pipelines.go:321`
- `app/pipelines.go:335`
- `domains/globalfeed/detail.go`
- `domains/radio/player/engine.go`
- `domains/radio/pronounce/pronounce.go`
- `domains/radio/script/script.go`
- `domains/radio/synth/compose.go`
- `domains/severe/codes.go`
- `domains/weather/nws/alerts.go`
- `modes/tty/alerts.go`
- `platform/plaintext/plaintext.go`
- `platform/render/contrast.go`
- `platform/snapshot/assembler.go`
