import pathlib
# The card window never re-hands its renderer, so a card that re-hydrates under the
# operator leaves the window showing its first frame — a stale DATA PULLED stamp
# and a stale manifest, on the one surface whose whole job is to be trusted before
# something goes on the air. This is F-30's freeze with a read in it: RefreshAfter
# is half of StaleAfter, so the window is open across a refresh routinely.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	if r.observer.modal != modalCard {
		return r
	}"""
new = """	if r.observer.modal != modalCard || true {
		return r
	}"""
assert old in s, "mW3"
p.write_text(s.replace(old, new, 1))
