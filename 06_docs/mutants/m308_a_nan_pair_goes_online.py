import pathlib
# 0.19.0 FR-4.9: a pair of numbers off the globe is refused offline.
# Killed by TestEveryResolverParsesCoordinatesLocally.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\t} else if numberPair(q) {\n'
assert old in s, 'm308'
p.write_text(s.replace(old, '\t} else if false && numberPair(q) {\n'))
