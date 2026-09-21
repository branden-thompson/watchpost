# 0.18.0 — handoff

**Read this before touching anything.** 0.17.0 made hazard data map-ready and deliberately drew
nothing. 0.18.0 draws. Everything below is what a session starting cold needs and cannot derive
from the code.

---

## 1. Where things actually stand

| | Watchpost | go-tuiMaps |
|---|---|---|
| Released | **v0.17.0**, merged `944df72` | **v0.1.0**, merged `bbc039a8` |
| Repository | `github.com/branden-thompson/watchpost` | `github.com/branden-thompson/go-tuimaps` (public) |
| Published tree | `1e73f65` | `ace896b3` |
| Local checkout | `<workspace>/watchpost` | `<workspace>/go-tuimaps` (sibling checkouts) |

**The library is published and resolvable.** `go get github.com/branden-thompson/go-tuimaps@v0.1.0`
works from `proxy.golang.org` — verified from a module outside both trees, using a host point type
carrying JSON tags and named ring types. **The first thing 0.18.0 should do is replace any local
path reference with that require**, because the local-path era is what hid a defect for a day (§4.2).

**Local `main` carries the full development history; `origin/main` carries one squashed commit per
release.** The two lines diverge by design and never fast-forward into each other. What must agree
is their **trees**, not their graphs — check that, not the shape.

---

## 2. The artifacts, all of them

**Diagrams** — rebuilt at ship, every block parsing:

- `06_docs/architecture-atlas.html` — 49 diagrams, this repo
- `06_docs/architecture-atlas.html` in go-tuiMaps — 41 diagrams

Both are generated, never hand-edited: `go run ./tools/atlas` from either repo root. Open the file
directly in a browser; it needs no server. Hosted copies exist for sharing, and their addresses are
deliberately not written down here — a link that decays is worse than the command that rebuilds the
page from source.

**The atlas is built from the mermaid blocks**, so a correction written only in the prose beside a
diagram never reaches the picture. That mistake has now been made in both projects.

**Pull requests** — the reasoning behind each release, in the form it was argued:

- Watchpost 0.17.0 — <https://github.com/branden-thompson/watchpost/pull/21>
- go-tuiMaps v0.1.0 — <https://github.com/branden-thompson/go-tuimaps/pull/1>

**In-repo documents that a cold session should read, in this order:**

1. `06_docs/02_features/map-ready-geometry/03-architecture-design/as-built.md` — what was built and
   how it diverged from the plan, including the finding the first review got wrong
2. `06_docs/02_features/map-ready-geometry/06-key_learnings/learnings.md` — four lessons, each
   earned by a defect that reached committed code
3. `06_docs/02_features/map-ready-geometry/05-debugging/defects-found-while-building.md` — six
   defects and, for each, the **instrument** that found it
4. `06_docs/02_features/map-ready-geometry/02-analysis/data-shape.md` — every live measurement
5. `06_docs/follow-ups.md` — **the record.** A follow-up living anywhere else is forgotten
6. `06_docs/red-team-brief.md` — the dispatch template; do not compose one from memory
7. `06_docs/remediation-review-loop.md` — how to fix a finding without opening two more

**In go-tuiMaps:** `06_docs/02_features/go-tuimaps/07-readiness/integration-review.md` §9 is the
cross-project review — the two defects that only appeared when the library was pointed at a real
host, and the measurements that overturned a design decision.

---

## 3. What 0.18.0 inherits, concretely

### 3.1 The data shape is proven, not assumed

`resolveAlertAreas` (`app/mapgeometry.go`) returns `map[string]geo.Area`. An `Area` is the ground
**and which parts of it are unknown**. It has been run end to end against the live service; it has
**no production caller** — that is 0.18.0's first wiring job, and the release said so deliberately.

What the library needs, and where Watchpost already has it:

| `tuimaps` wants | Watchpost has | |
|---|---|---|
| `Overlay.Valid` / `Keeps` | `Alert.Sent`, `Effective`, `Expires`, `Ends` | ✓ |
| `Feature.Role` token | `Alert.Severity` → `AlertExtreme…AlertUnknown` | ✓ exact 1:1 |
| `Feature.Label` | `Alert.Event` | ✓ |
| `Overlay.Credit` | `Alert.SenderName` | ✓ |
| One `Feature` per area | `geo.Shape` is `[]Polygon` | ✓ |
| A point that says where it is | `geo.Point.LonLat()` | ✓ |

**One Feature per area, never one Feature per hazard.** A feature's first ring is its outline and
every ring after it is a hole. Convert with `tuimaps.Rings(polygon)` per polygon.

### 3.2 The completeness decision is deferred to you, on purpose

`Area.Missing` names the parts that could not be fetched. **The library has no channel for
partial data** — `Answer` carries `NoData` and `Stale`, nothing for "incomplete". So the decision
is the drawing layer's, and it is a real one: an alert is matched to a place by the **full** zone
list but drawn from the parts that resolved, so a map can shade up to a county line and stop. A
reader whose county failed reads that as *not me*.

### 3.3 Layering that constrains where the drawing can live

`scripts/lint-imports.sh`: **neither `modes/` nor `platform/` may import `domains/*`**, and
`platform/` is a leaf. `app/` is the only place that may name both sides. That is why `geo.Area`
lives in `platform/geo` — so whatever draws, under `modes/`, can name it.

---

## 4. What did not work, and will happen again if unread

### 4.1 A test of a thing is not a test of its wiring

**Four defences in one release were written, tested, and connected to nothing** — seeding, the cap
on held shapes, the counters, and a panic guard placed in a goroutine that could not catch the
panic. Every one had passing tests. All four were committed in the round whose stated lesson was
that exact defect.

Unit tests cannot see this: they call the method themselves, so the method works and the program
does not. The gate now is `TestEveryLivePipelinesMethodIsReachedFromProductionCode`
(`app/reachable_test.go`) — weak by construction, and it fires the moment a call site is deleted.

**Applies to 0.18.0 directly:** the drawing path will be wired from the composition root the same
way, and the same mistake is available.

### 4.2 Read what reads your output

`geo.Shape` was designed, planned, red-teamed twice and built as a flat ring list. The only thing
that would ever consume it documents its first ring as an outline and **every ring after it as a
hole**. A thirty-two-island zone would have drawn as one islet with thirty-one holes.

Every review that missed it read the *producer*. It was found by opening the consumer's source.
**Owning both sides makes it easier to check and less likely that anyone does.**

### 4.3 A mutation that "passes" because the build broke is not a passing mutation

Three times, deleting a fix left an unused variable or import, so the mutation failed to **build**
— which looks like a caught mutation and means **nothing tested the behaviour**. Check that the
mutant compiles and the *test* fails.

And once, a test passed for the wrong reason: its fixture made the good path and the bad path both
error, so it could not tell them apart. **Assert the fixture is valid before asserting behaviour.**

### 4.4 Measure before believing a bound, and state the bias

`MaxVertices` was chosen, then derived from eighty live zones weighted to the worst shapes: median
252, p95 7,477, max 15,194. A random sample would have found the median and taught nothing —
**a cap is a statement about the tail**. The bias belongs beside the number (`INST-5`).

### 4.5 Running a suite twice is a cheap flake detector

The release's last defect was found because CI builds every commit twice: one run passed, the
duplicate failed, on a byte-identical tree. A mutation harness was misreading a crash as a
survival about one Linux run in two. **If F-173 narrows the trigger, replace that detection
deliberately** — it was only ever an accident.

---

## 5. Standing rules that held, and cost something to keep

- **A gate is obeyed or ruled on, never argued with.** A shell-ledger gate refused a Python script;
  it was rewritten in Go rather than exempted. A `dupes` gate refused a self-issued reason; it went
  to the HUM LEAD (D-127). A PR hook refused a branch name; the fix was the repo's missing
  `origin`, not a bypass.
- **Never merge without a full green run** — HUM LEAD, at 0.17.0, after a Linux flake.
- **Red-team personas run BLIND.** A mask worn by the author inherits the author's blind spots.
  Every finding that mattered in 0.17.0 came from a reviewer with no context, and from *running*
  something.
- **Rulings one at a time**, with evidence, options, a recommendation and the strongest
  counter-argument. Recorded verbatim.
- **Silence is not consent** (D-39).
- **No AI attribution, ever**, in commits, PRs or tracked files. Personal git identity only.

---

## 6. Where to start

1. `git log --oneline -20` on `main`, then read `as-built.md` and `learnings.md` (§2)
2. Replace the local-path reference to the map library with
   `require github.com/branden-thompson/go-tuimaps v0.1.0`
3. Wire `resolveAlertAreas` into the snapshot or the view — it has no production caller, and
   `TestEveryLivePipelinesMethodIsReachedFromProductionCode` is the guard that will notice
4. Decide the completeness question (§3.2) with the drawing in front of you, not before
5. `make verify` before anything is committed; both CI platforms green before anything is merged

**Owed to the HUM LEAD and not done:** the screen-reader half of WP-14.17, and the first-host spike
review at WP-14.19. The spike branch `spike/tuimaps-first-host` is kept as reference until 0.18.0
lands, then cleaned up.
