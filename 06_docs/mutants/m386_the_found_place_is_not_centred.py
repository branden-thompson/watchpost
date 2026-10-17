import pathlib
# 0.19.0 FR-1.10 (D-193): the place found is the map's centre, so the centre answers for it.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\t\td.mapPane.call("Recentre", func() { _ = m.Recentre(tuimaps.LonLat{Lon: ref.Lon, Lat: ref.Lat}) })\n'
assert old in s, 'm386'
p.write_text(s.replace(old, '\t\t_ = m\n'))
