import pathlib
# 0.19.0 D-195: a swap with the map open puts it on the new UI mode's place.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '\tif mm.mapPane.m == nil || mm.mapPane.roleDrawn == e.Surface {\n'
assert old in s, 'm261'
p.write_text(s.replace(old, '\tif true {\n'))
