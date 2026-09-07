import pathlib
# The acquire side of mE6. `hold` dips the bed and claims it in TWO critical
# sections instead of one, so the lock is free between them — and `hold` runs on
# the executor's goroutine while `takeBack` runs on the arbiter's. A take-back
# landing in that window sees held=false over a ducked bed, lifts it, and `held`
# is then set over a broadcast already back at full volume: the rail reads its
# whole drain against an undipped bed, which is the inverse of MVS-D-67.
#
# This is the shape the code actually had, found by a fresh review after the
# release side was fixed and pinned (F-D5).
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = """	m.mu.Lock()
	defer m.mu.Unlock()
	m.dip()
	m.held = true
}"""
new = """	m.giveWay()
	m.mu.Lock()
	m.held = true
	m.mu.Unlock()
}"""
assert old in s, "mE7"
p.write_text(s.replace(old, new, 1))
