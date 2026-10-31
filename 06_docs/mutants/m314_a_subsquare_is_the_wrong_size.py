import pathlib
# 0.19.0 FR-1.10: a subsquare is a 24th of a square.
# Killed by TestALocatorIsTheCentreOfItsSquare.
p = pathlib.Path('platform/geo/maidenhead.go'); s = p.read_text()
old = '\t\tw, h = w/24, h/24\n'
assert old in s, 'm314'
p.write_text(s.replace(old, '\t\tw, h = w/12, h/12\n'))
