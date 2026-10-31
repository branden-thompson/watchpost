import pathlib
# 0.19.0 I-2: a position in a form not read here is refused, never sent online.
# Killed by TestAPositionInAnotherFormNeverGoesOnline.
p = pathlib.Path('domains/locations/resolver.go'); s = p.read_text()
old = '\t} else if positionShaped(q) {\n'
assert old in s, 'm432'
p.write_text(s.replace(old, '\t} else if positionShaped(q) && false {\n'))
