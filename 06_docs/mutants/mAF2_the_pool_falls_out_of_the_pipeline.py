import pathlib
# The recent pipeline is reconciled with the LISTENER's list alone, so the first
# commit — a lookup, a favourite, any watchlist edit — sees twenty-five pool
# locations that are no longer wanted, stops every one of their schedulers and
# drops their data. The pool table goes back to shimmering and stays there (D-112).
p = pathlib.Path("app/dashboard.go"); s = p.read_text()
old = """		lp.recent.update(withPool(recent, lp.poolRefs))"""
assert old in s, "mAF2"
p.write_text(s.replace(old, """		lp.recent.update(recent)""", 1))
