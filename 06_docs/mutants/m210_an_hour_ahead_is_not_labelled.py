import pathlib
# 0.19.0 FR-1.13, D-109: an hour ahead is labelled typical under the map.
# Killed by TestAnHourAheadSaysTypical.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tif origin.ok && d.mapPane.propAhead > 0 {'
assert old in s, 'm210'
p.write_text(s.replace(old, '\tif origin.ok && d.mapPane.propAhead > 99 {'))
