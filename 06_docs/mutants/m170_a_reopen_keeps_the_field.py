import pathlib
# D-154: a map reopened before its release draws no Propagation field.
# Killed by TestAReopenedMapDrawsNoPropagationField.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\tmm = mm.stopPropagation(e).setProp(e)'
assert old in s, 'm170'
p.write_text(s.replace(old, '\tmm = mm.stopPropagation(e)'))
