import pathlib
# `evictLocked` on an empty map takes the ZERO VALUE of K as its victim and
# deletes an absent key — a silent no-op, so an empty memo would "evict" for
# ever and the caller's bound would never be reached.
#
# IT SURVIVES BY DESIGN, and honestly so: `delete` on a key the map does not hold
# is a no-op in Go, so removing this guard changes no observable behaviour today.
# Every real call site reaches evictLocked only when len(m.items) >= max and max
# is at least 1, so the map is never empty when it runs.
#
# THE GUARD IS FOR A FUTURE CALLER, not for this one, which is exactly what makes
# it a tripwire rather than a defect: it fails the day someone calls evictLocked
# from a path that does not already know the map is occupied.
p = pathlib.Path("platform/bodymemo/bodymemo.go"); s = p.read_text()
old = """	if len(m.items) == 0 {
		return
	}
"""
assert old in s, "mBM2"
p.write_text(s.replace(old, "", 1))
