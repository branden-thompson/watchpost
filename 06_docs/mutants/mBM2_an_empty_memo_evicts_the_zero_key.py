import pathlib
# `evictLocked` on an empty map takes the ZERO VALUE of K as its victim and
# deletes an absent key — a silent no-op, so an empty memo would "evict" for
# ever and the caller's bound would never be reached.
p = pathlib.Path("platform/bodymemo/bodymemo.go"); s = p.read_text()
old = """	if len(m.items) == 0 {
		return
	}
"""
assert old in s, "mBM2"
p.write_text(s.replace(old, "", 1))
