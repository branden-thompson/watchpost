import pathlib
# 0.19.0 D-195: the service radius is in miles.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/map_zone.go'); s = p.read_text()
old = '* units.KmPerMile\n'
assert old in s, 'm259'
p.write_text(s.replace(old, '* units.KmPerMile / units.KmPerMile\n'))
