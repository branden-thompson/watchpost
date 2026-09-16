import pathlib
# The bound OQ-9 states in bodymemo's own package doc — "It is bounded. At most
# max entries, least-recently-used out" — is no longer checked, so the day the
# eviction condition is wrong the memo grows without limit and every observable
# goes on reading correct: a hit still returns what a parse would, and `parses`
# still counts only misses.
p = pathlib.Path("platform/bodymemo/bodymemo.go"); s = p.read_text()
old = """	if err := invariant.Check(len(m.items) <= m.max, "the memo holds at most max entries"); err != nil {
		return val, err
	}
"""
assert old in s, "mBM1"
p.write_text(s.replace(old, "", 1))
