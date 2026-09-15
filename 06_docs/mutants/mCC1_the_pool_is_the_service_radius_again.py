import pathlib
# D-130. The location check stops at the POOL, so a real place inside the service
# radius is refused for being the 26th. The pool is capped at 25
# (locations.PoolCap) and is a DELIBERATE subset of the fence — "Rainbow, CA" is
# 14.7 miles from Oceanside and answers "not found" (HUM LEAD, UAT 2026-09-14).
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = """		if r == nil {
			return snapshot.LocationRef{}, false, false
		}"""
new = """		if r != nil {
			return snapshot.LocationRef{}, false, false
		}"""
assert old in s, "mCC1"
p.write_text(s.replace(old, new, 1))
