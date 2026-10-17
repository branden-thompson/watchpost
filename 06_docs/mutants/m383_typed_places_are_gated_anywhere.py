import pathlib
# 0.19.0 FR-1.10: the Propagation mode reads coordinates and locators anywhere on the globe.
# Killed by TestALocalMatchSendsNothing.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\tif ref, typed, err := typedPlace(q, true); typed {\n'
assert old in s, 'm383'
p.write_text(s.replace(old, '\tif ref, typed, err := typedPlace(q, false); typed {\n'))
