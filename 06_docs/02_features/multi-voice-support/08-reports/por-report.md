# Plan of Record — multi-voice-support (0.14.0)

**Feature:** `feature/multi-voice-support` · **Phase:** PLAN exit · **Level 1 · SEV-0 · HUMAN LEAD**
**Prepared for:** the HUM LEAD's GO/NO-GO · **Date:** 2026-08-30
**Artefacts of record:** `01-objectives/objectives.md` · `03-architecture-design/plan.md` ·
`04-development/{implementation-plan,p1-foundation,p2-seams,p3-maritime-tones,p4-ui}.md` ·
`07-readiness/{gates,perf-protocol,linux-validation-protocol,release-checklist}.md`

---

## 1. Executive summary

Watchpost Radio has one voice for everything. After 0.14.0 it has a **cast**: the listener can keep one voice, or
give the alerts one correspondent and the reports another, with optional voices for the local weather, the sea,
the fires and the quakes. Correspondents hand over to each other by name. Every alert opens with a **tone that
says what kind of alert it is** — five sounds across six classes — and any class can be muted. Coastal listeners
gain a **maritime report** (the coastal forecast, the buoy, the tides and currents). The `[V]` and `[T]` controls
retire; `V` opens Setup at the new Correspondents group, and `[S]` explains who speaks for whom and why.

**Shape:** five batches — P0 measure · P1 foundation · P2 seams · P3 maritime and tones · P4 the screen — each
UAT-able on its own, each ending at the same gate.

**Confidence.** Four red-team rounds ran: forty lens-passes, 507 findings, all dispositioned. The last two rounds
**compiled and ran** the design in throwaway worktrees: the role package, the concurrency cap and the whole
broadcast seam pass their tests under `-race`, including the rule that protects the radio (R6). What the same
passes found in the *written* code — undefined symbols, wrong arities, tests contradicting their own code — is
why the plan now carries contracts and no implementation code; that work is preserved separately for BUILD.

**Asked of you:** eleven rulings (§8) and five mock questions (§9). Nothing else blocks BUILD.

---

## 2. Context — the problem and why now

The locked problem (brief v1.1.0, Candidate A): *a listener who is not looking at the screen cannot tell, by ear,
that what has just started speaking is an alert rather than one of the routine reports.* Today every word comes
in one voice; the only cue is the words themselves.

Two mechanisms answer it, and they answer different halves. The **tones** are on for everyone from the first
launch and say *which kind* of alert is coming. The **cast** — a different voice for alerts than for reports —
is what makes the distinction audible in the first two seconds without the listener configuring anything about
tones at all, but it is opt-in by ruling (MVS-D-8). Which of the two the release should claim as "the answer" is
E-3, below; the objectives currently claim neither, pending your ruling.

Also in scope by ruling: the maritime report (MVS-D-4/14), the Setup redesign that the retirement of `[V]`
implies (MVS-D-3/23), and per-class tone mutes (MVS-D-26/28).

---

## 3. The selected architecture

Five seams, each with one owner:

1. **A role registry (`domains/radio/cast`)** — a small, dependency-free package that knows the roles, their
   inheritance (a role with no voice uses its parent's), and how to resolve a role to a voice **on this host**,
   explaining why the requested voice lost when it did. It knows nothing about audio, config or the screen.
2. **A cap on concurrent synthesis (`synth.Limiter`)** — today nothing bounds how many `say`/`piper` processes
   run at once; on Linux each is a ~63 MB model load. One decorator wraps every `Say` with N ordinary slots plus
   two reserved for the Station Director, so the line a listener is waiting for never queues behind the
   broadcast's read-ahead.
3. **Roles on segments, voices in the cache key (`synth.Source`)** — each segment carries its role; the Source
   resolves it once at render and caches the audio under the voice that spoke it. A change of correspondent
   between segments is announced with a scripted line **rendered ahead of the air**, so the goroutine feeding the
   speaker never waits on a synthesiser.
4. **The Station Director** — today's narration arbiter, renamed, now forwarding the job's role (whose voice) and
   the alert's class (which tone). The tone is generated from constants, so it sounds immediately even when the
   voice behind it is still downloading.
5. **The screen** — Setup grows two groups per the HUM LEAD's mock (tones, correspondents); the Radio panel takes
   one of three fixed layouts by width; `[S]` and `watchpost report --verbose` both answer "who speaks for whom,
   and why".

The config carries a typed pair per role (`macos`, `piper`) so one file works on both platforms, and 0.14.0 is
the first release whose save **preserves keys it does not understand** — the hole that makes an older binary
drop the new tables.

Diagrams and the component/import rules are in `plan.md` §2–§3.

---

## 4. Alternatives considered

| Axis | Chosen | Rejected, and why |
|---|---|---|
| Where resolution lives | a pure `cast` package over a host interface | in the deck — untestable without audio, and the rule would have three copies |
| The Piper backend | a process per utterance under the cap | a resident warm process — unmeasured, unrunnable on the developer's machine, and its seam already exists (**E-1**) |
| The cap | one decorator around `Voice.Say` | a semaphore at each call site — every future caller must remember it |
| The hand-over | rendered ahead, cached per pair | rendered on the writer — measured at 2–20 s of silence per boundary on Linux |
| The tones | five parameterised presets | embedded WAVs — unreviewable, unreproducible |
| The marine zone | the nearest zone by geometry, with a no-geometry fallback | geometry only, or fallback only (**E-8**) |
| Setup | groups inside the existing window | a separate sub-window — a second place to learn |

---

## 5. The implementation plan

| Batch | What lands | Proven so far |
|---|---|---|
| **P0** | the two "before" numbers: the tone path's code-path timing on `v0.13.0`, and today's Setup allocation | method fixed; the 0.13.0 twin is named |
| **P1** | the gate's own scope helper; the cap; `Compose(Reports{})`; the `cast` package; the config tables, the unknown-key preservation and its fixtures; the deck as a host | `cast` **3/3 green**; Limiter **green at `-race -count=3`**; config **green on go-toml v2.4.3** |
| **P2** | roles on segments; the voice-keyed byte-bounded cache; the hand-over rendered ahead; the Station Director; find-only alert path with the tone from constants; the ticker and read tones | the assembled Source **17/19** and **12/14 green under `-race`**, incl. R6's writer property; `make pty-severe` green |
| **P3** | the maritime report and its scripts; the coastal forecast; the five presets | the marine wording matches its scripts exactly; preset envelopes within tolerance |
| **P4** | Setup's two groups; the Radio panel per breakpoint; the retirements; `[S]` and `report --verbose`; docs, goldens, the journey | the row table and both groups match the mock character-for-character (hand-computed; not yet compiled) |

Every batch ends at `07-readiness/gates.md` §1 and is UAT-able alone. Each task is stated as **file · symbol ·
contract · test intent · verify command**; the code is written at BUILD against the compiler.

---

## 6. Risks and how they are held

| Risk | Hold |
|---|---|
| **R6 — the broadcast stalls** (the radio's sacred rule) | the hand-over renders ahead; a background change never renders on the writer; a test asserts *zero* synthesiser calls on the writer goroutine and a paced reader sees no gap beyond the output path's ≈0.7 s |
| **Unbounded synthesis** (today's worst case is six processes) | the Limiter, landing in P1 before any role work |
| **A voice that will not speak** | resolution always falls back up the tree and says so in `[S]`; the one legitimately silent case (Linux, nothing installed) is named and validated |
| **The alert tone waiting on a download** | the tone is constants; the alert path never installs synchronously; a failed install backs off |
| **Config data loss on save** | the unknown-key preservation, with fixtures — including the escaped-key case that proved the dependency bump necessary |
| **Hostile text from the config or the network** | one plain-text seam per surface; the maritime section is bounded; zone ids are validated at the I/O edge |
| **Memory** | the audio cache is bounded in bytes (40 MB) with the numbers written into the soak's pass rule |

---

## 7. Decisions of record

Rulings **MVS-D-1…28** stand as recorded in the brief. Decisions the red-team rounds produced, which are the
agent's and are recorded as such: **D-R2-1…5** (the reserved slot for every Director job; the trust window
deleted; the hand-over rendered ahead; the install backoff; the dependency spike), **D-R3-1…3** (two reserved
slots; no writer render for a background change; the two-part M4 instrument) and **D-R4-1…3** (the two failure
rules; `Invalidate` lands at the next segment *rendered*; the go-toml bump is required). Two of those amended
ratified material and are therefore in your list below (**E-9**).

---

## 8. Rulings requested

| # | Question | Recommendation |
|---|---|---|
| **E-1** | Confirm the **resident-Piper cut** from 0.14.0. AX-1 was ratified as "build the seam for all three"; the seam is the existing `Voice` interface, so nothing is lost. The measurement stays on your UAT list as 0.15.0 input. | **Confirm** |
| **E-2** | **Scope.** (a) whole; (b) minus the maritime report; (c) minus the per-class mute's UI; (d) minus the three-breakpoint Radio panel. A "tones + pickers, UI later" split cannot ship — the pickers *are* the Setup work, and MVS-D-3 couples `[V]`'s retirement to the panel. | **(a)** — each batch is UAT-able alone, so (b)/(c)/(d) remain available at the P2 or P3 gate |
| **E-3** | **What the release claims.** (a) keep Candidate A and the "default-on answer" wording; (b) keep Candidate A, drop that wording — the tones are the fresh-install *cue*, the cast is the answer; (c) re-lock as delight. | **(b)** now; revisit with M3's result |
| **E-4** | A **default alert voice on a fresh install** (free on macOS; +63 MB background on Linux) versus MVS-D-8's one voice. | **Keep MVS-D-8** for 0.14.0 |
| **E-5** | The Setup label **"Maritime"** for the storm class — it covers a Kansas blizzard. | "Tropical / Winter Storms", or keep the mock's word |
| **E-6** | **`[M]` compatibility.** A listener who muted alerts in 0.13.0 will hear the words again. (a) accept with a CHANGELOG line; (b) map the old setting to "mute tones and words" for one release; (c) a one-time in-app note. | **(a)** |
| **E-7** | Three items marked "not objected — proceeding" need **explicit rulings** at SEV-0: Blizzard Warnings in the storm class; the maritime wording as written; the three-period cap on the coastal forecast. | ratify all three |
| **E-8** | **The marine zone.** (A) nearest-by-geometry (a new NWS endpoint, capped and budgeted) or (B) the first nearshore block, no geometry. MVS-D-14 said the forecast came "for free"; (A) is not free. | **(B)** for 0.14.0; (A) returns if UAT reads the wrong stretch |
| **E-9** | **Ratify (or send back) the amendments the red-team made to ratified material:** M3 becomes two trials with a comparative rule (AM-17); M2's "≤ 5 actions" becomes the measured **11 keypresses**, pinned as a regression guard (AM-18); M5 admits the one legitimately silent row (AM-19); M4 becomes a code-path pin plus an absolute live budget (AM-21); AX-4 gains a **second** reserved slot. | **Ratify all five** as MVS-D-30…34 — or rule M2 back to a target, in which case OP-4's Save chord is the lever |
| **E-10** | **Unattended downloads.** A hand-edited config can start background installs on launch (catalogue-pinned, but unasked). (a) as planned; (b) a per-session cap, then "install on next Save"; (c) only on a Setup save. | **(b)** |
| **E-11** | **NFR-7's lint claim.** It promises `golangci-lint` and `staticcheck` "at every exit"; neither has ever run in `make verify` or CI, and the command is red today on one pre-existing style finding outside this feature. (a) amend NFR-7 to the gates that exist; (b) adopt the linter now (one line to fix, plus mirroring the `third_party/` exclusion). | **(a)**, with "add a `make lint` target" staying on the carried follow-ups |

---

## 9. Open points on the mock (Setup)

**OP-1** the `←→ Voice` chip · **OP-2** the re-worded DATA rows · **OP-3** the "nothing ticked = every class"
support line · **OP-4** the chip wording, the EVENTS key rule, and whether Save gets a `ctrl+s` chord (the only
lever on M2's number) · **OP-5** the chips pinned as a footer rather than scrolling with the body.
Details in `p4-ui.md`.

---

## 10. Critical analysis — what this plan is weakest at

- **The listening test is one person.** M3 is now two trials with a control build and a blinding rule, but the
  panel is the HUM LEAD. It is evidence, not proof, and E-3's ruling should not wait on it alone.
- **Linux is validated by one operator on one box.** Every Piper-shaped behaviour — installs, RSS, the first
  cycle's timing — rests on `07-readiness/linux-validation-protocol.md` being worked through by hand.
- **P4 has not been compiled.** The Setup rows match their builders on paper; the batch is the one place where
  BUILD should expect surprises, and its gate is the whole tree's tests plus three new goldens.
- **The maritime report is the largest addition with the least user evidence.** E-2 (b) and E-8 exist precisely
  so it can be trimmed without disturbing the rest.
- **Four red-team rounds produced diminishing design findings.** Rounds 1–2 changed the architecture; rounds 3–4
  mostly found defects in written code that no longer exists. The lesson is recorded as `AP-PLANCODE-01`.

---

## 11. Sources

The brief (v1.3.0, rulings MVS-D-1…28, amendments AM-1…21) · the objectives (FR-1…14, NFR-1…8) · the six analyses
and the two mocks · the DISCOVER report and its red-team · the two plan reviews · the red-team ledger
(`red-team-plan.md` §1–§14, with round 2's per-ID file) · the readiness protocols · `prior-art/README.md` for
what was compiled and what it proved.

---

## 12. Next steps on your GO

1. Rule E-1…E-11 and OP-1…OP-5; I fold the answers into the objectives, the plan and the batch files, and record
   them as MVS-D-29 onward.
2. **P0** — take the two "before" numbers.
3. **P1** — the gate helper first (it fails today on any docs-only commit), then the cap, the `cast` package and
   the config, with the go-toml bump and its licence regeneration.
4. Gate, build log, your UAT, then P2.

BUILD writes the code against the compiler, using `prior-art/` only as a comparator once its own version exists.
