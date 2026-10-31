import pathlib
# 0.19.0 FR-1.13: a shown hour ahead's label carries its typical error.
# Killed by TestTheTypicalErrorStatesItsBasis.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t} else if e, ok := d.propTypicalError(lat, lon); ok {\n'
assert old in s, 'm363'
p.write_text(s.replace(old, '\t} else if e, ok := d.propTypicalError(lat, lon); false && ok {\n'))
