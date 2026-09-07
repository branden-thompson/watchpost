# Release checklist — 0.14.0 multi-voice-support

The FULL GIT shape (`project-watchpost-git-protocol`): one `feature/multi-voice-support` branch for the release;
local `main` is the dev trunk (never pushed); `main-publish` mirrors `origin/main`; the release is one
`commit-tree` squash of the feature tree with parent `main-publish`, pushed as `release/v0.14.0`, PR'd to `main`
with the canonical template, squash-merged, tagged on the merged commit; the feature branch is deleted after.

## Before the release commit

- [x] **P1–P4 gates green; `p10-p1..4.json` present in `07-readiness/`.** The `tree_hash` ROWS this line
  asked for do not exist and should not: `gates.md`'s batch record deleted that column deliberately —
  *"a hand-copied hash beside it was a second place to be wrong, and was: the P1 row recorded a value
  that is not a git object and does not match `p10-p1.json`"*. The JSON carries the hash; citing it is
  the point. All four files are present, each with its `tree_hash`, `tools` and findings (8 · 9 · 11 ·
  11), and **every finding is covered by a ledger row for its file or its package** — checked by
  matching each finding's path against `.a2dh-p10-exemptions.yml`. (That check is by path, not by
  rule-id pair, so it establishes coverage rather than exact correspondence.)
- [x] **Every P10 row THIS RELEASE added or changed is ratified — HUM LEAD, 2026-09-06 ("Ratified").**
  Each carries an explicit ratification with a date in the ledger, mirrored into `director-build-log.md`
  because the ledger is gitignored (PL-16): P1's four (2026-08-30), `platform/category` (2026-09-01),
  the `everyTick` row (2026-09-03, which REPLACED two rather than adding a third), `app/inject_release.go`
  (2026-09-05) and R-5's (2026-09-06). **Audited by enumeration, not by reading the record's own summary** —
  and that found the build log claiming the whole ledger was ratified when it had checked fifteen rows.
  **23 rows still name a gate that passed releases ago; they are pre-existing, carried as F-45, and
  deliberately NOT folded into this ratification** — presenting rows the HUM LEAD never saw by absorbing
  them into their answer would be self-approval by proxy.
- [x] **`CHANGELOG.md` `[0.14.0]` complete**: Added · Changed (incl. the `[M]` line) · Fixed. The three
  late REVIEW fixes were missing and are now entered — the silent voice preview (F-41), the Details
  title after a lookup (F-42), and the Watchlist tab that could not say why it was empty. All three
  came from UAT, not from the suite, which is the note worth keeping about this section.
- [x] **README complete, prose and captures.** All sixteen images are 0.14.0, captured on `v17e6a23`, and none teaches a retired control. Original note: Verified present: the
  Correspondents group (`Settings' *Watchpost Radio — Correspondents*`), the tone mutes (the `Alerts —
  Tone` group, the `M` row, and `[radio.tones]`), the marine report, all three config key groups, the
  scripts paragraph naming `handover/` and `marine-report/`, and the MVS-D-20 comment note. Fixed while
  checking: three places still called the window **Setup** after 0.14.0 renamed it Settings, the report
  was still introduced as **maritime** after MARITIME became MARINE, and the Radio section's broadcast
  order **omitted the marine report entirely** — the order sentence named fire and seismic and skipped
  the segment `compose.go` puts before them. Still owed: `docs/img/maritime.png`, plus re-captures of
  `setup.png` and `radio.gif`. (`docs/img/voices.png` and `radio-min.png` are gone from both the README
  and the directory. The line's `maritime-report/` is the script directory's pre-rename name; it is
  `marine-report/` on disk, and the README names it correctly.) **The captures are taken at PR time —
  HUM LEAD ruling, 2026-09-06:** *"Screenshots will happen once we're ready to cut the PR."* The prose
  half of this row is complete and does not block REVIEW exit.
- [x] **`docs/where-things-happen.md` rows present** — the Director (the Watchlist row: `advanceBed` /
  `onEnded`), the resolver (the one-owner row), the hand-over (`renderAhead` / `announce` / `takeOver`),
  Settings (three rows: the picker, the layout, the finish) and the `[S]` table (`StatusTable`).
  **`docs/extending.md` Walkthrough 3** exists, and every symbol it names still resolves — `defaultKeyMap`,
  `toggleRadio`, `radioControlLines`, `radioDeck` (still declared in `app/radio.go`), `RadioStatusMsg`,
  `withCmd`, `takeCmd` — checked rather than assumed, because a walkthrough naming a dead symbol is worse
  than none (AP-HIST-01).
- [x] **NFR-8 assessed — it cannot read zero, and should not.** The grep (`[V]`, `[T]`, `radio-size`,
  `radio-min`, `radioMin`, `modalVoice`, `VoiceNoteMsg`, `Severe Alerts`, `voices.png`) returns 23
  outside `06_docs`, and every one is accounted for: **4** are `Memo[T]`, a generic type parameter
  the pattern catches by accident; **1** is `platform/lineup/script.go`'s reference to the
  *Broadcaster mock's* `[T]` panel, a different `[T]` entirely; **9** are comments and tests that
  exist to RECORD the retirement (`"[T] Size retired with the breakpoints"`,
  `"radio-size": retired at 0.14.0`) — a test that pins a thing is gone must name it; **1** is
  `dashboard.go`'s current and correct note that `[V]` opens Settings at the correspondents; and
  **8** are `VoiceNoteMsg`, which was on the list because it was expected to die with the chooser
  and instead was repurposed — it carries the deck's preview progress to the Settings cast rows
  (F-41). `modalVoice` and `voices.png` are genuinely gone, which is what the requirement was for.
- [x] **The three Setup goldens reviewed against the mocks — DELEGATED to the agent by the HUM LEAD,
  2026-09-06 ("Approved for you to do this"); the record is `07-readiness/goldens-vs-mocks.md`.**
  Layout, glyphs, labels and controls; colour excluded by the mock's own instruction (MVS-D-10). Seven
  divergences traced to later rulings and listed so they are not re-opened, including the Maritime /
  Marine pair that looks wrong and is MVS-D-33. Both collapse rules and the scroll rail confirmed.
  **Two findings, neither cosmetic: F-46** (the mock's *All Reports* picker is missing and the role is
  config-only — it needs a ruling) and **F-47** (`--ascii` survivors in the radio panel). Neither was
  visible from the goldens passing: they pin what IS drawn, and this review asked what SHOULD be drawn
  and is not.
- [x] **`07-readiness/validate/m3.md` recorded** (`a3ef5f2`, "the ear test, accepted on UAT evidence").
  Accepted on the HUM LEAD's own listening across UAT sessions rather than a staged A/B: *"I hear the
  difference when a burst contains warnings vs. only contains watches … from a UAT perspective I am
  satisfied with that functionality for this version."*
- [→] **MOVED TO VALIDATE (post-release) — HUM LEAD ruling, 2026-09-06.** `linux-validation-protocol.md`
  and `perf-protocol.md` §3–5 (per-Piper RSS, soak phase B, disk) need the CUT RELEASE installed on the
  Arch box, so they cannot precede the release commit: *"Arch linux box always happens POST release —
  because I need the cut release to install and test on the box."* §0–2 (time-to-tone-start on macOS,
  the Setup allocation pin) are filled and stay here. **This is a sequencing ruling, not a waiver: the
  rows are still owed, and 0.15.0's resident-Piper decision (MVS-D-17/OQ-18) has no input until §3 is
  measured.**
- [→] **`LinkedIn` returns ZERO across the tracked tree** (checked 2026-09-06 with the name supplied). `li-A2DH` appears in 7 files, all of them **already public from earlier releases**, so the release exposed nothing new; whether to scrub them is its own decision. Original line: (`bare a2dh` only as the gate command) — `: "${A2DH_HOME_NAME:?set in your shell}" "${EMPLOYER_NAME:?set in your shell}"; git grep -n -i -e "$A2DH_HOME_NAME" -e "$EMPLOYER_NAME" -- ':!third_party'` → zero (the guard refuses to run with either name unset — an empty `-e` would match every line; the two names live in the developer's shell, never in the tree); `.a2dh.yml` and the P10 ledger untracked
- [x] `git worktree list` shows the main tree only — the stale agent worktree was released 2026-09-06.
- [x] **`THIRD_PARTY_LICENSES.md` current — no dependency has moved since it was generated.** `go.mod`
  and `THIRD_PARTY_LICENSES.md` were last changed in the *same* commit (`ab96c48`, 2026-08-30), so the
  file cannot be behind the module list.
- [→] **VERIFIED AT THE PR, not here — HUM LEAD ruling, 2026-09-06:** *"We'll verify we're using the
  right github login through the PR process."* The identity check moves to the release section below,
  where the first outward action actually happens.

## The release

- [x] **`main-publish` is current** — fetched 2026-09-06; `main-publish` and `origin/main` are both `e74e4fd`, so no pull was needed. The feature branch is 771 commits ahead.
- [x] `git branch release/v0.14.0 $(git commit-tree feature/multi-voice-support^{tree} -p main-publish -m "0.14.0: multi-voice support …")`
  — **mechanics proven by dry run (2026-09-06)**: the candidate commit's tree is byte-identical to the
  feature tip with `main-publish` as its only parent and an empty diff against the tip. Not cut yet, and
  deliberately: the tree must already contain the README captures, so cutting before them would only be
  re-cut afterwards.
- [x] **`gh api user` = `branden-thompson`**, under `GH_CONFIG_DIR=~/.config/gh-personal` — checked 2026-09-06, before any outward action.
- [x] The README captures taken and committed — plus `relay-fault.png`, a headline feature that had no image at all, and `themes.png` replaced by `themes.gif` because the caption always said "applied live".
- [x] **`a2dh pr-template check` PASSES** on `07-readiness/pr-body.md` — the filled body, written against the
  canonical A2DH template (the repo's own `.github/PULL_REQUEST_TEMPLATE.md` is thinner than the contract).
  **The checker was itself controlled before its tick was trusted**: removing a section raises
  `R3-section-present`, and a metrics table carrying only the placeholder row raises `R3-metrics-row`.
  The body carries no attribution and no internal names — the template's guidance comments, which do name
  one, are deleted as the template instructs, so nothing internal reaches this public repo.
- [x] **PR #5**, squash-merged as `1abf27d`; **PR #6** (one line, the changelog date) merged as `64ae770`.
- [x] **CI green on both legs — after FIVE rounds, which is the release's finding.** The branch was local-only
  and the last CI of any kind was 0.13.0, eight days earlier, so PR #5 was the first time 0.14.0's code had ever
  run on Linux. It panicked. Four rounds fixed where the pointer was nil rather than why a unit test was
  downloading 63 MB; the fifth fixed an assertion I had preserved without asking what it was for.
- [x] **`v0.14.0` annotated on `64ae770`, pushed 2026-09-07.** Run `34116932007` green in 13m42s; it re-ran the
  full `make verify` against the tag before publishing, so the artefacts came from a tree that passed on the
  runner. **8 assets**, both Linux binaries among them.
- [x] Local `main` carries the feature tip, tree byte-identical to `origin/main`; `main-publish` mirrors `64ae770`.
- [ ] Delete `origin/release/v0.14.0`, `origin/release/v0.14.0-date` and the local feature branch (all merged)

## After

- [ ] **VALIDATE on the Arch box, against the installed release** — `linux-validation-protocol.md` all rows;
  `perf-protocol.md` §3 (per-Piper RSS), §4 (soak, phase B) and §5 (disk). Feeds OQ-18 / MVS-D-17.
- [x] **DEBRIEF written** — `08-reports/debrief.md`; awaiting HUM LEAD approval
- [x] **Carried items recorded in `project-watchpost-follow-ups`** (position updated to SHIPPED, with the Linux-first finding and the unbuilt producer/consumer check carried explicitly): resident Piper (0.15.0, after the RSS measurement), the Source rate decorator, the `Roles()` table, the 12/24-h · TZ · FRS-transmit preferences (MVS-D-21 backlog)
