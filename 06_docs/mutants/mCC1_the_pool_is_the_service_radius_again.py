import pathlib
# D-130. The location check stops at the POOL, so a real place inside the service
# radius is refused for being the 26th. The pool is capped at 25
# (locations.PoolCap) and is a DELIBERATE subset of the fence — "Rainbow, CA" is
# 14.7 miles from Oceanside and answers "not found" (HUM LEAD, UAT 2026-09-14).
p = pathlib.Path("app/pool.go"); s = p.read_text()
# Re-pointed 2026-09-16 (D-151): the hook returns a fourth value — whether the
# question could be PUT at all — so the no-resolver arm carries it. The mutation
# is unchanged: inverting the guard stops the lookup ever reaching the resolver,
# which is the pool-only behaviour this rule forbids.
old = """		if r == nil {
			return snapshot.LocationRef{}, false, false, false // no resolver: the question cannot be put
		}"""
new = """		if r != nil {
			return snapshot.LocationRef{}, false, false, false // no resolver: the question cannot be put
		}"""
assert old in s, "mCC1"
p.write_text(s.replace(old, new, 1))
