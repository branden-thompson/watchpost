import pathlib
# D-129. The console's `[l]` stops being scoped to the station pool, so a place
# the transmitter cannot reach answers as a valid location. The operator types
# "Lone Pine, CA" — hundreds of miles outside the service radius — gets no
# refusal, and enter opens its details as if the station could broadcast about
# it (HUM LEAD, UAT 2026-09-14).
#
# Re-pointed 2026-09-14 (D-130): the hook is LocateInRadius now — the pool was
# never the right test.
#
# THE SURFACE IS THE WHOLE RULE. Observer may look anywhere and must keep doing
# so; this mutation makes the console behave like Observer, which is exactly the
# defect and not a simplification.
p = pathlib.Path("modes/tty/modal_location.go"); s = p.read_text()
old = '	return d.addMode == "lookup" && d.surface == SurfaceBroadcaster && d.cfg.LocateInRadius != nil'
new = '	return false'
assert old in s, "mCB1"
p.write_text(s.replace(old, new, 1))
