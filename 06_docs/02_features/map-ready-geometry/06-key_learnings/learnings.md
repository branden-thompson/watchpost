# What this release taught

Four things, each earned by a defect that reached committed code.

## 1. A test of a thing is not a test of its wiring

Seeding was measured, argued, ruled on, built and tested — and nothing called it. It passed every
test it had and did nothing at all. The tests were of the *function*; what was missing was a test
that the function is **reached** from the path a user takes.

**Applies to:** anything approved as a behaviour rather than an API. The test to write is the one
that fails when the call is deleted, not when the body is.

## 2. Bounds come in three kinds, and we had two

The parser bounded **depth** (against stack overflow) and **size** (against a huge document). A
document of two million empty rings passed both and held 117 MB while reporting zero vertices. The
third kind is **quantity**, and nothing counted it.

**Applies to:** every reader of untrusted input. Depth, size, quantity — and the emptiness case at
each level, because "zero of these" is how you get past a counter that only counts non-zero things.
Adding the area level to the reader re-opened this exact door one level up, and it had to be shut
again by hand.

## 3. Read what reads your output

`geo.Shape` was designed, planned, red-teamed twice and built as a flat list of rings. The one
thing that will ever consume it documents the first ring as an outline and **every ring after it as
a hole**. A thirty-two-island zone would have drawn as one islet with thirty-one holes punched in
it, and the first of those islets covers about one per cent of the zone.

Every review that missed this read the producer. It was found by opening the consumer's source and
then measuring a real fixture.

**Applies to:** any seam between two components, especially two we own. Owning both sides makes it
*easier* to check and *less likely* that anyone does.

## 4. The sample that sets a bound should be drawn from the tail

`MaxVertices` was first chosen — "50,000 sounds generous against the 12,000 we saw". Replacing it
with a derived figure meant sampling eighty zones deliberately weighted to Alaska and the marine
zones, where the island chains are. That found a 15,194-position zone (forty-one areas) and a zone of 122 **rings**,
both above anything in the fixtures.

That sentence was wrong when first written - it said "a 122-area one" - which is
worth leaving on the record: the count of rings and the count of areas are
exactly the distinction lesson 3 is about, and it was got backwards in the
document teaching it.

A random sample would have found the median (252) and told us almost nothing, because **a cap is a
statement about the tail**. The bias has to be stated alongside the number (`INST-5`), and it is.

## The thread running through all four

Three of these four were found by **running or counting**, and the fourth by reading *the other
side of an interface*. None was found by re-reading the code under review. Re-reading re-derives
the model that produced the defect — which is the argument for the blind reviewer, and for the rule
that a reviewer who ran nothing has not reviewed.
