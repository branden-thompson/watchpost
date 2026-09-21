# Defects found while building, and how each was actually caught

Six real faults, recorded with the *instrument* that found each — because the pattern across them
is more useful than any one of them.

| # | The fault | How it was caught | Would a review have caught it? |
|---|---|---|---|
| 1 | **The GeoJSON reader was off by one.** A position closes *at* `positionsAt`, not one below it | A test asserting vertex *counts* | No — it parsed without error, and a test asserting "no error" would have passed |
| 2 | **A denial of service in our own parser** (RT-1). Two million empty rings: 117 MB held, zero vertices reported | Running the attack | **No.** Depth and size were bounded and the code read as complete. Nothing counted *quantity* |
| 3 | **Seeding was never called** (RT-2). Approved, built, tested, wired to nothing | Counting non-test callers | No — it passed every test it had |
| 4 | **A nil client would panic the seeding goroutine**, ending the program | A test that happened to pass a provider with no client | Possibly, as a style note; not as a crash |
| 5 | **`make schema` hard-coded the version** the test derives, so the first bump broke it | The first bump | Unlikely |
| 6 | **Rings were flattened out of their areas** (RT-8). A thirty-two-island zone would draw as one islet with holes | Reading the *consumer's* contract, then measuring a real zone | No — three reviews of the producer missed it |

## What the column on the right is saying

**Five of six were found by running or counting something. One was found by reading — and what was
read was the other side of the interface, not this code.**

Faults 2, 3 and 6 are the same shape: the code was correct against the model its author held, and
the model was wrong. No amount of re-reading finds those, because re-reading re-derives the model.
This is the whole case for the blind reviewer, and for the rule that a reviewer must run something.

Fault 6 adds the sharper version: **the defect was not in this code at all.** Every line of the
reader was right about GeoJSON. It was wrong about what the thing downstream would do with the
result, and that could only be found by opening the downstream file. "Have you read what reads
this?" is now a row in `07-readiness/build-exit-record.md`.

## The two that were not in the code

Worth keeping because neither is a Go defect and both cost real time.

- **The shell-ledger gate refused a Python script** under `/scripts/` — correctly: this repo's rule
  is that everything is Go unless a ruling says otherwise. The atlas builder was rewritten in Go
  under `tools/` rather than an exemption being sought. A gate that is argued with instead of
  obeyed stops being a gate.
- **Five architecture diagrams had not rendered for three releases** — unquoted parentheses in
  flowchart labels, semicolons inside sequence messages. Silent failures: the document existed and
  looked complete. Carried as `F-170`.
