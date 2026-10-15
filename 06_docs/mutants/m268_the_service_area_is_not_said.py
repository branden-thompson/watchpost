import pathlib
# 0.19.0 FR-2.4, D-195: the words say the service area.
# Killed by TestTheServiceAreaIsSaid.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\tif w := d.zoneWords(); w != "" {\n'
assert old in s, 'm268'
p.write_text(s.replace(old, '\tif w := d.zoneWords(); false && w != "" {\n'))
