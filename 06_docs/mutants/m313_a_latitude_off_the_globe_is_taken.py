import pathlib
# 0.19.0 FR-4.9: a coordinate is on the globe.
# Killed by TestNaNCoordinatesAreRefused.
p = pathlib.Path('platform/geo/coords.go'); s = p.read_text()
old = '\tif v[0] < -90 || v[0] > 90 ||'
assert old in s, 'm313'
p.write_text(s.replace(old, '\tif v[0] < -900 || v[0] > 900 ||'))
