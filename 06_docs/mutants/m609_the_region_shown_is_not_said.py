import pathlib
# 0.19.3 AX-6 (A-107 (8)): the region row says which region the map is bound to.
# Killed by TestEveryMapControlSaysItsState.
p = pathlib.Path('modes/tty/map_scrub.go'); s = p.read_text()
old = 'r.Name == d.mapPane.region.Name {'
assert s.count(old) == 1, 'm609'
p.write_text(s.replace(old, 'r.Name != d.mapPane.region.Name {', 1))
