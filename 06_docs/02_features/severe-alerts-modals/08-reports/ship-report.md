# SHIP Report — Severe Weather / Disaster Events Modals (Watchpost 0.13.0)

| Field | Value |
|---|---|
| Phase | SHIP (VALIDATE → **SHIP** → REFLECT) · SEV-0 · HUM LEAD |
| Entry | VALIDATE signed off 2026-08-29 ("GO 4 SHIP"); release checklist: A1–A5, B1–B6, B8, B10, C1–C5 ☑; B7 Linux half, B7 takeover check and B9 carried as **accepted risks** (below) |
| What ships | `github.com/branden-thompson/watchpost` — public, MIT · release `v0.13.0` · five platform binaries + `checksums.txt` · install line unchanged (`curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh \| sh`) |
| Verdict | see §6 (filled at the end of the phase) |

## 1. What a user gets

The **Severe Weather / Disaster Events window** (`w`): every active warning, watch, advisory, special
statement, significant quake and tropical cyclone — the union of the national feeds and the tracked
locations' alerts, de-duplicated — browsed by category with the full record of each one press away,
and read aloud on the radio with `space`. The ticker's breaking line now names storms. The
dashboard's severe strip is a one-row box per category; the header and column titles are painted
bands; the Setup window groups its questions. Thirteen themes, including the first light one
(**Watchpost Light**), every painted pair lifted to WCAG AA. A `--ascii` form for every glyph. Feeds
decode entry by entry, so one malformed record no longer empties a category. The full list is
`CHANGELOG.md` §0.13.0.

Locked problem statement (project brief, ratified 2026-08-28): *a user who has just been told a
severe weather or disaster event is active cannot see which events are active, or read the full
details of any one of them, without leaving the app or recalling and searching a location by hand.*
Metrics at ship — **M1 K2D** 4 actions, no typing: `w` · `→` · `↓` · `enter` (the VALIDATE journey's
"w opens the severe window" → "right moves to Watches" → "enter opens a record" steps, scripted PTY,
22/22) · **M2 COV** 100 % of the frozen render list (`TestRenderListCoverage`,
`TestUSGSDecodesTheRenderList`, `TestNWSDecodesTheRenderListAndAllowlistsParameters` over
`domains/globalfeed/testdata/`) · **M3 NAR** 2/2 (`go test ./app -run Narration`, green 2026-08-29) ·
**M4 R6·PERF** macOS relay + audio smoke passed 2026-08-29; frame allocations within the pinned
budgets; the 1-hour soak flat (`07-readiness/validate/soak-1h.csv`).

## 2. Pre-ship checklist (Step 1)

| Check | Result |
|---|---|
| Release checklist — every ☐ dispositioned | A5 closed at ship (CHANGELOG dated 2026-08-29). Open ☐ = accepted risks, §4 |
| Identity | `gh api user` → `branden-thompson`; `ssh-add -l` lists the personal key; the repo pins `core.sshCommand` to it. No employer account or key involved |
| Public-tree scrub | `git diff main-publish feature/severe-alerts-modals`: no employer or internal-product names, no machine-local paths, no addresses. The bare gate-command name appears only where it *is* the gate command (ruling 2026-08-29). `.a2dh.yml` and the P10 ledger are untracked (`.gitignore: .a2dh*`) |
| Gates on the release tree | `make verify` ALL GATES GREEN · `make p10` 0 live / 0 unmatched (`07-readiness/p10-build.json`) · `a2dh validate` 18/18 · `make pty-severe` green · race suite 40 packages ok (`07-readiness/validate/after-soak.txt`) |
| Lint (the PR template's check) | `golangci-lint` + `staticcheck` found five findings on new code — an unused `fgMix` helper, an unchecked `os.Unsetenv` in a test, three De Morgan hints — fixed at SHIP, gates re-run. Left as is: one De Morgan hint in `domains/radio/stream/directory.go` (the relay path is not touched at a release, R6) and two upstream nits in the vendored kit. Note: the linters rewrite `go.sum` (they resolve with `-mod=mod`); `git checkout go.sum` before `make verify`, whose tidy gate is strict |
| Rollback plan | exists and **tested** — §3 |
| Communication | CHANGELOG §0.13.0 is the release note; `release.yml` publishes the GitHub release with generated notes (`generate_release_notes`) and the five binaries, `checksums.txt`, LICENSE and THIRD_PARTY_LICENSES.md |

## 3. Rollback plan (tested 2026-08-29)

Releases are immutable tags; `main` is fix-forward. Rolling a user back is one line:

```sh
WATCHPOST_VERSION=v0.12.0 sh -c "$(curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh)"
```

Tested before the tag: `WATCHPOST_VERSION=v0.12.0 WATCHPOST_INSTALL_DIR=<scratch> sh scripts/install.sh`
installed a binary that reports `watchpost version 0.12.0` (SHA-256 verified by the installer).

If `v0.13.0` must be withdrawn after publishing: mark the GitHub release *pre-release* (the installer's
"latest" then resolves to `v0.12.0` again), fix forward on `main`, and cut `v0.13.1`. A published tag is
never moved (the 0.12.0 exception — a red first run that had published no artifact — does not apply
once assets exist). Config is compatible both ways across 0.12 ↔ 0.13 (`platform/config/config.go` is unchanged since
`v0.12.0`; the severe index is in-memory; the ticker cache and seen-store schemas are unchanged), so a
rolled-back binary reads the same files.

## 4. Accepted risks (HUM LEAD, 2026-08-29)

| # | Risk | Mitigation |
|---|---|---|
| B7-L | Linux half of the R6 relay + audio smoke not yet run (needs an audio device on the Arch box) | Post-release per `07-readiness/linux-validation-protocol.md`; the radio pipeline is unchanged in 0.13.0 except the arbiter, which the race suite and PTY smokes cover on Linux CI |
| B7-T | The breaking-takeover pause/resume of an event read not yet exercised live (quiet day) | Manual check later 2026-08-29 when events are more numerous; `TestEventReadIsSuspendedByABreakingTakeover` pins the behaviour |
| B9 | Linux `make pty-severe` not run locally | The release PR's Linux CI runs the race suite and the PTY smokes |
| PERF | Frame render 133×44 +38 % time vs v0.12.0 (facelifts: painted bands, AA lift, box strip) | Root-caused and accepted at VALIDATE; allocations within budget; soak flat |

## 5. Ship mechanics (Step 2–3, FULL GIT)

1. `release/v0.13.0` = one squash commit of the `feature/severe-alerts-modals` tree with parent
   `main-publish` (`git commit-tree … -p main-publish`) — the same shape as every release since 0.9.0:
   the working branch's history stays on this machine; GitHub `main` is the chain of release commits.
   The release tree is byte-identical to the VALIDATEd feature tree plus the CHANGELOG date, the A5
   row and this report.
2. PR `release/v0.13.0` → `main` with the harness PR template (`a2dh pr-template check` before
   `gh pr create`); Linux CI (`ci.yml`: race + PTY) must be green.
3. Merge the PR (rebase merge — the PR is one commit on `main`'s tip, so `main` stays a chain of
   release commits), then tag `v0.13.0` on the merged commit and push the tag; `release.yml` (verify → matrix → installer smoke test → publish) publishes the five
   binaries and `checksums.txt`. The version is the tag (`make` stamps `git describe`), so the "bump
   commit" and the release commit are one and the same — nothing to tag before it.
4. Verify the artifact, not the echo: install from the published release and read `watchpost --version`.
5. Delete `feature/severe-alerts-modals` (FULL GIT: one branch per release); `main` (dev trunk)
   fast-forwards to the release tree's commit for 0.14.0 to branch from.

## 6. Outcome

*(filled at phase exit)*

## Source documents

`08-reports/validate-report.md` · `08-reports/review-report.md` · `08-reports/build-report.md` ·
`08-reports/project-brief.md` (locked statement, metrics) · `07-readiness/release-checklist.md` ·
`07-readiness/gates.md` · `CHANGELOG.md`.
