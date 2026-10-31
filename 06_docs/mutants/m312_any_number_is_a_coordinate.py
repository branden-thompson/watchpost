import pathlib
# 0.19.0 FR-4.9: exponents and hex are no coordinate.
# Killed by TestNaNCoordinatesAreRefused.
p = pathlib.Path('platform/geo/coords.go'); s = p.read_text()
old = '\t\tif !coordinatePart.MatchString(f) {\n'
assert old in s, 'm312'
p.write_text(s.replace(old, '\t\tif false {\n'))
