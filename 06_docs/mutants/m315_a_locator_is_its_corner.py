import pathlib
# 0.19.0 FR-1.10: a locator is the centre of its square.
# Killed by TestALocatorIsTheCentreOfItsSquare.
p = pathlib.Path('platform/geo/maidenhead.go'); s = p.read_text()
old = '\treturn LatLon{Lat: lat + h/2, Lon: lon + w/2}, true\n'
assert old in s, 'm315'
p.write_text(s.replace(old, '\treturn LatLon{Lat: lat, Lon: lon}, true\n'))
