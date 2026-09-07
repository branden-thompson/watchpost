# P1 — the foundation: the cap, the cast package, the config (multi-voice-support, 0.14.0)

```
Goal:         The seams every later batch stands on — a bound on concurrent synthesis, the role registry and
              its resolution, and the config that carries a cast — with today's behaviour as the zero value.
Architecture: plan.md §2.1–2.2; data-shape.md §1–4
Branch:       feature/multi-voice-support
Gate:         07-readiness/gates.md §1 (the one owner of the commands)
```

Tasks are **task shape**: file · symbol · contract · test intent · verify. Code is written at BUILD against the
compiler (`AP-PLANCODE-01`, `06-key_learnings/retro-notes.md` RN-1). The PLAN-phase sketches — including the
blocks the round-4 lenses compiled and ran — are kept as prior art in `prior-art/` for comparison **after** you
have written yours; `prior-art/README.md` records what was proven green and the nine defects not to copy.

## File map

```
MODIFY: scripts/quality/p10-unmatched.sh (+ _test.sh)  — Task 1.0: Go-free-diff dormancy; rename-aware symbols
CREATE: domains/radio/synth/limit.go (+ test)          — Limiter, Limited, WithPriority, the per-Say bound
MODIFY: domains/radio/synth/compose.go (+ call sites)  — Reports struct; Compose(…, Reports)
CREATE: domains/radio/cast/{cast,resolve,validate,tone}.go (+ cast_test.go)
MODIFY: platform/config/config.go                      — RoleVoice, Voices, Tones, Radio, Validate, Save
CREATE: platform/config/keep.go                        — unknownKeys, keepUnknown, unknownRadioKeys
CREATE: platform/config/testdata/*.toml                — 11 fixtures (§1.10)
CREATE: app/cast.go (+ cast_test.go)                   — castConfig, the deck as cast.Host, runtimeGOOS seam
MODIFY: app/radio.go, app/voices.go (+ radio_test.go)  — the deck's limiter, Discovered/Installed, listVoices
MODIFY: go.mod, go.sum, THIRD_PARTY_LICENSES.md        — the go-toml bump (Task 1.9; now required, see below)
```

---

### Task 1.0 — the P10 gate's scope helper: dormancy for a Go-free diff, rename-aware symbols

**Why first:** every later gate runs `make p10`, whose second step is `scripts/quality/p10-unmatched.sh`. Two
defects there fail the gate for reasons unrelated to the code under review.

**Contract 1 — a Go-free diff means dormant, not in-scope.** The helper scopes rows to the Go files in the diff
(P10 governs compiled code only; the tool agrees — a docs-only diff yields no findings). Its fallback currently
conflates *no base resolved* with *base resolved, no Go file changed*; both leave the file list empty and the
empty case is treated as "everything in scope". A **resolved** base with no Go files changed makes every ledger
row dormant (exit 0); only an unresolvable base keeps the pre-FULL-GIT behaviour.

**Contract 2 — a renamed file's rows are scoped against both paths.** Git reports `narrate.go → director.go` as
a rename and shows only changed hunks, so P2's re-keyed rows match nothing while their new path is in the diff.
For a rename target, the symbol test runs against the diff of **both** paths (`-M`, `-U0`); a symbol whose hunks
are unchanged is dormant.

**Constraint:** the script is `#!/bin/sh` and runs on macOS (bash 3.2) and Linux (dash) — POSIX only: no
associative arrays, no `[[`, no `local`, no process substitution.

**Tests:** a fixture git repo, three cases — docs-only commit → all dormant, exit 0; rename with the symbol
unchanged → dormant, exit 0; rename with the symbol changed and nothing matched → unmatched, exit 1.

**Verify:** the three fixture cases; `A2DH=<framework build> make p10` on a docs-only branch → exit 0.

---

### Task 1.1 — `synth.Limiter`: a bounded, prioritised `Say` (FR-12)

**Files:** `domains/radio/synth/limit.go`, `limit_test.go`.

**Contract:** one owner of "how many renders at once" — a decorator `Limited(v Voice, l *Limiter) Voice` whose
`Say` is admitted by the limiter and bounded in time; `Name`/`Rate` pass through. **N ordinary + 2 reserved** (the second slot amends ratified AX-4 and is itself ratified as **MVS-D-43**)
slots (plan §1.3): a context carrying `WithPriority` — set by the Station Director for *every* job it runs — may
take a reserved slot; two, because a takeover is admitted while the read it suspended may still hold one for an
in-flight render (D-R3-1). The bound covers **the wait for a slot as well as the render** (`DefaultBound = 90 s`),
so a Say queued behind wedged renders cannot wait N × bound. Nil voice or nil limiter **fails closed**: the
returned Voice's `Say` reports the wiring error; never an uncapped voice.

**Test intent:** N + 1 ordinary renders queue while the reserved slots stay free; a priority context enters when
the ordinary pool is full; the bound fires while a slot is held (signal entry on a channel — no timing polls);
nil wiring returns a Voice whose `Say` errors.

**Verify:** `go test ./domains/radio/synth -run 'Limited|Priority' -race -count=3`

---

### Task 1.2 — `Compose(…, Reports{})` (RS-12)

**Files:** `domains/radio/synth/compose.go`; call sites in `synth_test.go`, `seismic_test.go`, `soak_test.go`,
`app/radio.go`.

**Contract:** replace the eight positional parameters with one `Reports{Fire, Seismic, Maritime}` struct. The
**zero value must produce today's broadcast** — that is FR-2's anchor for every later batch.

**Test intent:** a zero-value `Reports` yields the 0.13.0 segment list (keys, order, pauses).

**Verify:** `go test ./domains/radio/synth -run Compose -count=1`

---

### Task 1.3 — `cast`: the role registry (FR-1; data-shape §1)

**Files:** `domains/radio/cast/cast.go`, `cast_test.go`.

**Contract:** the role tree `All → {Alerts → {Breaking, SevereRead}, Standard → {Weather, Maritime, Fire,
Seismic, Station}}` with `Parent`, `Key` (the config word), `String` (the label) and `Assignable()`. `Config{Root,
Pairs, CastMode, Tones}`; `Pair{MacOS, Piper}`. **Out-of-range values read empty, never panic** — the enums are
plain ints and cross package boundaries. The package imports nothing above `platform/invariant`.

**Test intent:** every role's key/label/parent; `Assignable()`'s membership and order; out-of-range reads.

**Verify:** `go test ./domains/radio/cast -count=1`

---

### Task 1.4 — `cast.Resolve`: the walk and the fallback matrix (FR-7; data-shape §3–4)

**Files:** `domains/radio/cast/resolve.go`, `cast_test.go`.

**Contract:** `Host{Platform, Discovered, Installed}` — `Discovered()` is a **closed allowlist at every moment**:
the curated macOS list until `say -v ?` has answered, the intersection after (D-R2-2 deleted the "trust while
discovering" window). `Resolve(role, cfg, host) Resolution{Role, Requested, Spoken, Link, Reason}` walks role →
parent → root → platform default and returns the first voice that exists here, plus **why** the requested one
lost. `Spoken` is the only string that ever reaches a voice or `[S]`. `WantsInstall(host)` names the Piper key
worth fetching in the background. The walk is **counter-bounded** (`maxDepth = 3`) with an invariant that it
reached `All` — a tree idiom, not an assumption; `Resolve` sits at the P10-04 ceiling (15 decisions), so anything
added goes in a helper.

**Test intent:** the M5 matrix as a table — assigned/installed · assigned-not-installed (Piper) · assigned-unknown
(macOS) · other-OS field only · the sentinel · inherited · root unresolvable → platform default · Linux with
nothing installed (the one legitimately silent row, AM-19). Every row asserts `Spoken`, `Link` and `Reason`.

**Verify:** `go test ./domains/radio/cast -run 'Resolve|Matrix|Range' -count=1`

---

### Task 1.5 — `cast.Validate` (FR-8, RS-2)

**Files:** `domains/radio/cast/validate.go`.

**Contract:** the **one** validator of the cast's semantics: assigned names that cannot speak here (a warning
naming where they fall back), the cast mode, and the closed tone-class set. Config validates table *shapes*
only. Unknown class keys are reported **once with a count**, not one problem per entry (the array is
file-controlled). `Validate` calls into the host, so its callers must not hold the deck's lock (Task 1.12).

**Test intent:** each problem's key and message; a clean config yields none; a hostile/unknown key set yields one
counted problem.

**Verify:** `go test ./domains/radio/cast -run Validate -count=1`

---

### Task 1.6 — `cast.Classify`, `ToneName`, `Muted` (FR-11; tones.md §2; MVS-D-15)

**Files:** `domains/radio/cast/tone.go`.

**Contract:** `Classify(product) Class` by the rule of record (tones.md §2): storm > statement > advisory >
watch > **warning as the loud default for anything unmatched**. `ToneName(class)` names the preset (disaster and
warning share the dual-tone); out of range returns the loudest, never a panic. `Muted(class, tones)`: mute mode
**and** (the set contains the class **or the set is empty**) — "Mute:" with nothing ticked means every class, so
what the screen says is what the ear gets. `ClassKeys()` is the one owner of the config words.

**Test intent:** a classifier table over real product strings including the precedence cases; `Muted`'s
empty-set rule; out-of-range `ToneName`.

**Verify:** `go test ./domains/radio/cast -run 'Classify|Tone|Muted' -count=1`

---

### Task 1.7 — withdrawn

The import pin re-implemented `make lint-imports`. Deleted at the round-1 review.

---

### Task 1.8 — config: the `[radio]` tables (FR-8; data-shape §2)

**Files:** `platform/config/config.go`.

**Contract:** `RoleVoice{MacOS, Piper}` × nine roles under `[radio.voices.<role>]`; `[radio] cast` (`""` single |
`"cast"`); `[radio.tones] mode/muted`. Each OS writes only its own half of a pair, so a config written on one
platform loads on the other with unresolvable roles falling back (FR-7). `Radio.Validate()` runs at load and
checks **table shapes only** (the class keys are `cast.Validate`'s — Task 1.5). A 0.13.0 `ticker_muted = true`
with no `[radio.tones]` loads as mute mode with an **empty** set (= every class); `Save` mirrors
`ticker_muted` from the mode — **`Save` is the one owner of that mirror**, no caller writes it.

**Test intent:** round-trip both directions; an assigned pair is a table and an empty pair is omitted; the
0.13.0 mute mapping; `Validate` names the key it rejects.

**Verify:** `go test ./platform/config -run 'Radio|Validate' -count=1`

---

### Task 1.9 — config: `Save` preserves keys it does not know (NFR-5)

**Files:** `platform/config/keep.go`; `Save` in `config.go`; `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES.md`.

**Contract:** today's load → edit → save through the typed struct discards unknown keys — the hole that makes a
0.13.0 binary drop the role tables. The strict decoder (`DisallowUnknownFields` → `StrictMissingError`) names
every unknown key; those paths, **kept as segments and never joined-then-re-split** (so a quoted top-level
`"radio.mode"` can never alias the nested key), are copied from the old file into the marshalled one. Paths are
walked **parents first**, so an unknown table is copied once and its children read as taken. A known scalar is
never turned into a table; a path through an array-of-tables (`[[locations]]`, `[[recent]]`) is **not preserved**
— a recorded limitation in NFR-5. Re-marshal only when something was kept, so an ordinary save stays
byte-stable. The strict decode is a **diagnostic**: it may never fail a `Load`, and its `recover` is scoped to
the decode alone (a bug in the caller's loop must not read as "no unknown keys").

**Dependency decision (was a spike; the spike ran):** on the pinned go-toml **v2.2.4 the strict decode panics**
for a quoted key carrying an escape (`subslice address … is before data address`), the recover masks it, and
unknown keys are then silently dropped — NFR-5 fails quietly. On **v2.4.3 the whole config suite passes**.
Therefore the bump is **required**, not conditional: `go get github.com/pelletier/go-toml/v2@v2.4.3`, then
`scripts/third-party-licenses.sh`, then `make verify`'s tidy/vuln. Keep the scoped recover regardless.

**Test intent:** a future key survives a save; a cleared known key does not resurrect; an untouched file is
byte-stable; an unparsable old file is not merged; the escaped-quoted-key file loads without a panic and its
keys survive; `[[locations]]` unknowns are dropped (the recorded limitation).

**Verify:** `go test ./platform/config -race -count=1`

---

### Task 1.10 — cross-version fixtures (NFR-5, FR-8)

**Files:** `platform/config/testdata/*.toml` — `0.13.0-only`, `0.14.0-both-os`, `0.14.0-macos-only`,
`0.14.0-piper-only`, `hostile-name`, `quoted-escape-key`, `unknown-key`, `wrong-type`, `bad-tone`.

**Contract:** each fixture pins one load rule: the 0.13.0 file inherits everything (FR-2); both-OS round-trips;
each single-OS file falls back on the other platform; the hostile file's names never reach a frame or argv; the
escaped-quoted-key file neither panics nor aliases a known key; a wrong type is reported as corrupt naming the
key; a bad tone value is named. **Every fixture is referenced by a case** — a fixture no test names is dead
weight (the two merge-edge files proposed in round 3 are dropped unless a case uses them).

**Verify:** `go test ./platform/config -run Fixture -count=1`

---

### Task 1.11 — `[S]` lists unknown `[radio.*]` keys once (NFR-5)

**Files:** `platform/config/keep.go`, `config.go`.

**Contract:** `Config.Unknown` carries the `[radio.*]` keys this build does not know, as **display strings** —
dotted and through `plaintext.Line`, because a quoted TOML key may carry escapes — capped at 32. Never
persisted. This is the one place the rendering happens.

**Test intent:** the hostile fixture's key reaches `Unknown` plain and truncated; a non-`radio` unknown key does
not appear.

**Verify:** `go test ./platform/config -count=1`

---

### Task 1.12 — `app`: config → `cast.Config`, and the deck as `cast.Host`

**Files:** `app/cast.go`, `app/radio.go`, `app/voices.go`, `app/cast_test.go`.

**Contract:** `castConfig(cfg)` maps the typed config to `cast.Config` (one mapper). The deck implements
`cast.Host`: `Platform` through a **production seam** `runtimeGOOS` (a package-level var in `app/cast.go` that
tests may assign — not a test-file function); `Discovered()` answers the curated list until `say -v ?` has been
read, then the intersection; `Installed(key)` is a pure `os.Stat`, never an install. Discovery runs under a
context with a 30-second ceiling — one package-level `discoverMacVoices(ctx)` shared by the deck and (P4) the
report, not a deck method. **Lock discipline, stated once and binding on every later task:** `Discovered` and
`Installed` take the deck's mutex, therefore **`cast.Resolve` and `cast.Validate` are never called while it is
held** — snapshot under the lock, resolve outside, store under the lock. P1 also lands a no-op `castChanged()`
so the batch builds alone; P2 gives it its body.

**Test intent:** the mapping round-trips; the host answers before and after discovery; a `setCast`-shaped call
with an assigned pair completes under a timeout (the deadlock guard) — the test must **fail**, not hang.

**Verify:** `go test ./app -run 'Cast|Host' -race -count=1 -timeout 60s`

---

### Task 1.13 — the deck owns one `Limiter`; every voice it hands out is `Limited` (FR-12)

**Files:** `app/radio.go`, `app/voices.go`, `app/radio_test.go`.

**Contract:** one limiter per process, on the deck: `NewLimiter(renderSlots(), 2)` — 3 ordinary on macOS, 2 on
Linux/Windows (a Piper render is a ~63 MB model load), 2 reserved for the Director. Every voice the deck builds
or previews passes through it; nothing else in `app` calls `Say` directly.

**Test intent:** with one ordinary slot held, the deck's next `Say` reports the limiter's bound (signal the hold
on a channel; no sleeps).

**Verify:** `go test ./app -run 'Limited|Voice' -race -count=1`

---

### Task 1.14 — P1 gate (the batch-exit checklist)

Run `07-readiness/gates.md` §1 in order, then:

- **Order matters:** update the declsets *before* the test line, and stage everything but the p10 record and the
  build log *before* `make p10`, so the recorded `tree_hash` is the commit's.
- **P10 ledger:** rows are added **only from `dist/p10.json`** — never predicted ahead of the tool. Expect a
  `domains/radio/cast` package density row (a pure package) and reason refreshes on packages the diff
  re-activates. Record the CLI's version and commit in the gates.md batch record: the ledger's symbol keys
  depend on the CLI's key scheme, so one build serves the whole release.
- **Build log** `p1-build-log.md`: the go-toml bump and its licence regeneration, the ledger decisions, deviations
  from this plan, the gate table. No attribution trailers.

**UAT-able alone:** a 0.13.0 config still sounds exactly as 0.13.0; a hand-written `[radio.voices.*]` pair is
accepted, validated and reported in `[S]`, though nothing reads it until P2.
