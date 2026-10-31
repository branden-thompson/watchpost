import pathlib
# 0.19.0 D-195: the Broadcaster's map draws its service area.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = '{place: mapOnStation, zone: true,'
assert old in s, 'm257'
p.write_text(s.replace(old, '{place: mapOnStation,'))
