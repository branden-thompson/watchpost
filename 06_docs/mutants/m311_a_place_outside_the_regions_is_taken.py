import pathlib
# 0.19.0 FR-4.9: a typed place in no region the weather covers is refused.
# Killed by TestCoordinatesAndLocatorsNeverGoOnline.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\tif _, in := geo.RegionOf(at.Lat, at.Lon); !in && !anywhere {\n'
assert old in s, 'm311'
p.write_text(s.replace(old, '\tif _, in := geo.RegionOf(at.Lat, at.Lon); false && !in && !anywhere {\n'))
