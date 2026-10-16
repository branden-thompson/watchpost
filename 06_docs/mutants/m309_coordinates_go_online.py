import pathlib
# 0.19.0 FR-4.9: coordinates are read locally, never sent online.
# Killed by TestEveryResolverParsesCoordinatesLocally.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\tif p, ok := geo.ParseCoordinates(q); ok {\n'
assert old in s, 'm309'
p.write_text(s.replace(old, '\tif p, ok := geo.ParseCoordinates(q); false && ok {\n'))
