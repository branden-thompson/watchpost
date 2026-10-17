import pathlib
# 0.19.0 AX-10: the world finder is never told to try 'City, ST'.
# Killed by TestTheWorldsFailuresSayItsOwnFix.
p = pathlib.Path('domains/locations/openmeteo/geocode.go'); s = p.read_text()
old = '\t\tif world {\n\t\t\treturn snapshot.LocationRef{}, errors.New("no place found by that name'
assert old in s, 'm419'
p.write_text(s.replace(old, '\t\tif world && false {\n\t\t\treturn snapshot.LocationRef{}, errors.New("no place found by that name'))
