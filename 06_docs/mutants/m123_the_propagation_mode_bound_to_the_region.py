import pathlib
# W2.2 (FR-1.2, D-21): the Propagation mode has no region's bound.
# Killed by TestEachModeKeepsItsOwnBound.
p = pathlib.Path("modes/tty/map_pane.go"); s = p.read_text()
old = "\tcase modePropagation:\n\t}\n\tvar err error"
assert old in s, "m123"
p.write_text(s.replace(old, "\tcase modePropagation:\n\t\tb = tuimaps.Bound{MinZoom: regionFitZoom(r, d.mapBodySize()), W: r.W, S: r.S, E: r.E, N: r.N}\n\t}\n\tvar err error"))
