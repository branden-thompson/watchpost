import pathlib
# The bound OQ-9 states in bodymemo's own package doc — "It is bounded. At most
# max entries, least-recently-used out" — is no longer checked, so the day the
# eviction condition is wrong the memo grows without limit and every observable
# goes on reading correct: a hit still returns what a parse would, and `parses`
# still counts only misses.
#
# IT SURVIVES BY DESIGN — the D-42 tripwire shape, and it was CHECKED rather than
# assumed. The invariant it deletes only fires when eviction is ALREADY wrong, so
# while eviction is correct the memo never exceeds its cap and removing the check
# changes nothing any test can see. Three tests drive the bound itself
# (TestTheMemoNeverHoldsMoreThanItsCap and its neighbours) and all three stay
# green under this mutation, correctly.
#
# THAT IS THE POINT OF KEEPING IT. The guard is for the day the eviction
# CONDITION is edited — `!ok && len(m.items) >= m.max` is one `&&` away from
# never evicting — and on that day this mutant stops surviving and the invariant
# starts earning its place.
#
# IT OVERRIDES THE CHECK RATHER THAN DELETING IT, and that is not tidiness. This
# is now the file's ONLY use of `invariant` — the other one was a no-op removed at
# D-160 — so cutting the block outright orphans the import and the mutant fails to
# COMPILE, which `mutant-check` reads as a broken mutant rather than as evidence.
# An INVALID mutation measures nothing; the `true` keeps the call and takes away
# what it decides.
p = pathlib.Path("platform/bodymemo/bodymemo.go"); s = p.read_text()
old = """	if err := invariant.Check(len(m.items) <= m.max, "the memo holds at most max entries"); err != nil {"""
new = """	if err := invariant.Check(true, "the memo holds at most max entries"); err != nil {"""
assert old in s, "mBM1"
p.write_text(s.replace(old, new, 1))
