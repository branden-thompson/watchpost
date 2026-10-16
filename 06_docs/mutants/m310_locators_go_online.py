import pathlib
# 0.19.0 FR-4.9, FR-1.10: a locator is read locally, never sent online.
# Killed by TestCoordinatesAndLocatorsNeverGoOnline.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\t} else if p, ok := geo.ParseLocator(q); ok {\n'
assert old in s, 'm310'
p.write_text(s.replace(old, '\t} else if p, ok := geo.ParseLocator(q); false && ok {\n'))
