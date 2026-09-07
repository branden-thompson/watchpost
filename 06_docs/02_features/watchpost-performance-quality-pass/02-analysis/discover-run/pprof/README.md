# The raw profiles are not kept here

The 0.13.0 performance pass captured thirty-two pprof profiles in this directory. They were removed
from the repository in the 0.14.0 red team.

**Why.** Every one of them embedded absolute build paths from the machine that captured them —
`/Users/<user>/…` — and this directory is on the published branch, so the repository was publishing a
developer's home directory layout to anyone who unzipped a profile.

**What is lost, and what is not.** A profile is a measurement of one binary on one machine at one
moment. Nobody else can re-run against it, so it was never reproducible evidence; it was a working
file that happened to be committed. What it *showed* is written up and stays:

- `02-analysis/` — the findings and the numbers taken from those runs.
- `06-key_learnings/reading-profiles-and-soak-logs.md` — how to capture and read a profile, so the
  next pass can produce its own.

**If you need profiles again**, capture them (`WATCHPOST_DEBUG_PPROF`, per the key-learnings note)
and keep them out of the tree, or strip the paths before committing anything binary.
`docs/accepted-costs.md` records the measurements that mattered.

Note that removing them here does not remove them from git history. The published branch is rebuilt
rather than merged, so a future publish will not carry them; a history rewrite would be needed to
reach the old objects, which is out of proportion to a directory path.
