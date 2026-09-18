import pathlib
# The pipeline is reconciled BEFORE the station moves, so a restationed console is
# published a new pool and fetches the old one — it names a region whose weather
# nothing is asking for (D-112).
p = pathlib.Path("app/dashboard.go"); s = p.read_text()
old = """	if lp.recent != nil {
		lp.recent.update(withPool(recent, lp.poolRefs))
	}
	return nil"""
assert old in s, "mAF3"
new = """	return nil"""
s = s.replace(old, new, 1)
old2 = """	lp.watchRefs = append([]snapshot.LocationRef(nil), watch...) // re-home the ticker's tie to the new watchlist (under lp.mu, already held)"""
assert old2 in s, "mAF3b"
new2 = """	if lp.recent != nil {
		lp.recent.update(withPool(recent, lp.poolRefs))
	}
""" + old2
p.write_text(s.replace(old2, new2, 1))
