import pathlib
# 0.19.0 AX-1: the app's namer keeps its last point; a frame asks no scan.
# Killed by TestAPointIsNamedAnywhere.
p = pathlib.Path('app/mapnames.go'); s = p.read_text()
old = '\t\tif asked && lat == lastLat && lon == lastLon {\n'
assert old in s, 'm404'
p.write_text(s.replace(old, '\t\tif asked && lat == lastLat && lon == lastLon && false {\n'))
