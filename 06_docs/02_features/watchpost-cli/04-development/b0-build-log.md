# B0 Build Log — Skeleton, Gates, Platform Foundations

| Field | Value |
|---|---|
| Milestone | B0 (architecture §8) · BUILD phase · SEV-0 |
| Date | 2026-08-23 |
| Commits | `eb0aca3` skeleton+gates · `0d389f3` platform pkgs · `67ceb61` Go 1.27 toolchain · `62e1905` extending.md · (this commit) red-team remediations |

## BEFORE-YOU-WRITE-CODE gate (persisted verdict — run at BUILD entry)

```
BEFORE-YOU-WRITE-CODE GATE — milestone B0
  [✓] Intent & plan understood — B0 = skeleton + gates + platform/{config,term,httpx} per architecture §8/§10.10
  [✓] Directives satisfied — FULL GIT (feature branch), FULL TDD (tests first per package), FULL DOCS (package contracts + this log)
  [✓] Dependencies pinned & compiling — per-milestone exact pins (go-toml v2.2.4, x/term v0.45.0/x/sys v0.47.0 post-Go-upgrade); module + 5-platform matrix build green
  [✓] Library safety — B0 stdlib+pure-Go only; oto concurrency stress deferred to B4 entry gate (recorded); charm deps at B3 (G-5 source-verified)
  [✓] Abstraction spikes — S1/S2/G5 done pre-BUILD; bubbles/help adapter spike scheduled at B3 entry
  VERDICT: READY
```

## Delivered

go.mod (Go 1.25 floor, `toolchain go1.27.0`) · Option-C tree (tracked via .gitkeep) · Makefile verify (fmt/vet/race/import-lint/watermark-lint + positive controls) · 5-platform release matrix + checksums · `scripts/release/install.sh` skeleton (T-M, explicitly not-live) · `platform/invariant` (vendored verbatim; `Checkf` retained as part of the vendored API surface) · `platform/config` · `platform/term` · `platform/httpx` · `cmd/watchpost` placeholder with tested `run()` seam · `docs/extending.md` · P10: 0 live findings, 2 HUM-LEAD-approved density exemptions · govulncheck: 0 vulns (system Go upgraded 1.24.4 → 1.27.0, official pkg, checksum-verified).

## Red-team B0 exit — disposition ledger

Round: code axis (solo) + sectioned hygiene/docs/business/infosec/perf/junior-dev (a11y scope-scanned NOT TRIGGERED — no UI in B0; re-scope at B1a/B3). Verdicts: Code CONDITIONAL PASS · Hygiene CONDITIONAL PASS · Docs CONDITIONAL PASS · Business PASS-with-edits · InfoSec PASS · Perf PASS · Junior-dev CONDITIONAL PASS. Watermark sweeps clean (gate + independent grep).

| Finding | Sev | Disposition |
|---|---|---|
| F1 transport errors swallowed; redactErr dead; security test vacuous (probed) | Critical | **Fixed**: doAttempt returns the cause; exhaustion path wraps `redactErr(lastErr)`; test now asserts the transport cause survives redacted |
| F2 ticker goroutine leak per Client (probed ×50) | Important | **Fixed** by deletion: lazy next-token pacing under a mutex — no goroutine exists |
| F3 MaxRetries 0→3 trap | Important | **Fixed**: `-1` = no retries, documented; `TestZeroRetriesIsExpressible` |
| F4 backoff shift overflow → rand panic | Minor | **Fixed**: shift and total capped (30s) |
| F5 Save persists FirstRun → first-run detection breaks; dead Load check | Important | **Fixed**: Save refuses FirstRun (tested); dead check deleted; B2 rule recorded: branch on `len(Locations)` |
| F6 no fsync; Windows rename/perms semantics | Minor | **Fixed** (tmp.Sync); Windows coverage → VALIDATE list |
| F7 impossible-state invariant in resolveWidth | Minor | **Fixed** by deletion |
| F8 lint evasions (raw-string imports, transitive deps; scan/live exclusion divergence) | Minor | Watermark unified + post-merge fallback **fixed**; `go list -deps` import check → VALIDATE list |
| F9 toolchain directive missing; §10.11-vs-§8 snapshot-types conflict | Minor | **Fixed** (`toolchain go1.27.0`); resolution recorded here: §8 governs — snapshot types land at B1a |
| F10 fake-400 invariant; Checkf unused | Minor | Invariant **deleted**; Checkf **retained** as vendored-API surface (recorded rationale) |
| H-1/B-1/J-1 untracked skeleton + missing release stub | Important | **Fixed**: .gitkeep-tracked tree + install.sh skeleton |
| H-2 spike .log files gitignored | Important | **Fixed**: renamed .log.txt, tracked |
| H-3/D-4 READY verdict + build tracking write-only | Important | **Fixed**: this log |
| H-4 watermark main..HEAD empty post-merge | Minor | **Fixed**: -n 20 fallback |
| D-1 false "linter enforces both" claim | Important | **Fixed**: reworded |
| D-2/D-3 banner coverage; tier wording | Minor | **Fixed** |
| B-2 term exemption reason mislabeled "stateful" | Minor | **Fixed**: "validating path (pure but assertion-bearing)" |
| S-1 FIRMS path-segment key unredacted (future B5) | Important | **Fixed now**: 32-hex path segments masked + tripwire test |
| S-2 xargs quote handling | Minor | **Fixed**: `-z`/`-0` |
| P-1 dripper leak (theoretical, single shared client per §3) | Minor | Superseded by F2 fix |
| P-2 RatePerSec > 1e9 ticker panic | Minor | **Fixed**: rate capped ≤1000 (and no ticker exists) |
| `[keys]` config carries no Help text; Merge replaces whole Bindings | — | **Transferred → B3** wiring item (user-override merge must preserve Help) |
| term "TTY never calls Width()" unenforced | — | **Transferred → B3** check |
| Zero-key Actions pass Merge silently | — | **Transferred → B3** policy decision |

**Declined:** none. **VALIDATE list additions:** Windows rename/perms; `go list -deps` import-direction check; fsync crash-window behavior on non-APFS filesystems.

## Exit state

`make verify` ALL GREEN · `go test ./... -race` 4/4 ok · p10 0 live · govulncheck 0 · validate 18/18 · release matrix OK.
