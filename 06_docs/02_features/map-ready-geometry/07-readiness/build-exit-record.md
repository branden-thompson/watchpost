# BUILD exit — map-ready geometry (0.17.0)

**This record is retroactive, and says so rather than reading as contemporaneous.** RT-10 of the
BUILD-exit red team was that no pre-code READY verdict existed for this feature: the PLAN was
red-teamed twice and approved, and the build then began without anything recording that the gate
had been passed. This page is written after the fact from the branch's own evidence — commits,
build logs and gate runs — and every row below cites what it is read from. A retroactive record is
worth having; a retroactive record dressed as a contemporaneous one is not.

## What the pre-code gate would have asked, and what the evidence says

| Question | Verdict | Read from |
|---|---|---|
| Is there a ratified plan? | **Yes.** Red-teamed twice; the second round deleted more than it added | `93ec92b`, `03-architecture-design/plan.md` |
| Does the plan name its tests before its code? | **Yes**, per phase p1–p4 | `plan.md` task table |
| Are the interfaces it crosses agreed? | **Partly — and this is where RT-8 got in.** The plan fixed `geo.Shape` as the seam without reading what the map library does with one | `plan.md`; `internal/overlay/store.go:40` in go-tuiMaps |
| Is the data measured, not assumed? | **Yes**, unusually well: live NWS measurements throughout | `02-analysis/data-shape.md` |
| Are the bounds derived? | **No.** `MaxVertices` was chosen. Closed at BUILD exit round 2 (RT-6) | `platform/geo/geojson.go:11` |
| Is there a rollback? | **Not applicable** — nothing draws yet; the release is additive to the snapshot | `as-built.md` |

## The one that mattered

The gate's third row is not a formality. **The plan settled the shape of the seam without reading
the contract of the only thing that will ever consume it.** `overlay.Feature` documents its rings
as "a polygon's outer ring and its holes" — first ring the outline, every ring after it a hole —
and `geo.Shape` was defined as a flat list of rings. A zone of thirty-two separate islands would
have drawn as one islet with thirty-one holes in it.

Nothing in the plan, either red-team round on the plan, or the BUILD-exit review's first pass
caught it, because all of them read the *producer*. It was found by reading the *consumer* and then
measuring a real zone. The pre-code gate is where that question belongs, and its absence is not
why it was missed — but a gate that asks "have you read what reads this?" would have.

## Verdict, as of the second BUILD-exit round

**Ready to exit BUILD**, subject to the blind red-team round this record precedes. Ten findings in
round one; eight now closed in code, two carried as `F-171` and `F-172` in `06_docs/follow-ups.md`
with the evidence that decides them.

The carried items are bounded and named. Neither blocks 0.18.0: one is a sampling question about
two constants set well above the measured tail - `MaxVertices` at about three
times it, `MaxRings` at thirty, and the other is a fault no producer
we read has ever shown.
