import pathlib
# 0.19.0 R-2.11: the centre's answer says its nearest station.
# Killed by TestAnUnscoredNearestStationIsSaid.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\treturn append(out, d.propNearestWords(c.nearest))\n'
assert old in s, 'm224'
p.write_text(s.replace(old, '\treturn out\n'))
