import pathlib
# 0.19.0 FR-4.9: a coordinate is on the globe (geo.OnEarth, since 0.19.5 DR-9).
# Killed by TestNaNCoordinatesAreRefused.
p = pathlib.Path('platform/geo/coords.go'); s = p.read_text()
old = '\tif !OnEarth(v[0], v[1]) {\n'
assert s.count(old) == 1, 'm313'
p.write_text(s.replace(old, '\tif !OnEarth(v[0]/10, v[1]) {\n', 1))
