import pathlib
# 0.19.0 FR-1.11: a centre past the near-vertical radius is answered by its path.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\tif c.km <= key.radiusKm {\n'
assert old in s, 'm216'
p.write_text(s.replace(old, '\tif c.km >= 0 {\n'))
