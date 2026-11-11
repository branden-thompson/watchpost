import pathlib
# 0.19.5 Q5.6 (DR-9): the edges are on the Earth.
# Killed by TestOnEarthRefusesWhatIsNoPlace.
p = pathlib.Path('platform/geo/coords.go'); s = p.read_text()
old = '\treturn lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180\n'
assert s.count(old) == 1, 'm771'
p.write_text(s.replace(old, '\treturn lat > -90 && lat < 90 && lon >= -180 && lon <= 180\n', 1))
