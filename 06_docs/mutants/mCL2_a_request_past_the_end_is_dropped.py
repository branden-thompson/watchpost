import pathlib
# D-149. A request at a position past the last visible card is refused instead of
# landing at the end — which is the Line-Up Request window's OWN DEFAULT (slot
# 15) against an ordinary three-card running order. The window has already closed
# on `valid()`, so the operator is shown a scheduled request the schedule never
# took: FR-3.3's named trap.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = "	if n := l.visible(t); to > n {\n		to = n\n	}\n"
assert old in s, "mCL2"
p.write_text(s.replace(old, "", 1))
