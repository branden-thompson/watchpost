import pathlib
# D-154: a map reopened before its release draws no Propagation field.
# Killed by TestAReopenedMapDrawsNoPropagationField.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\td = d.stopPropagation().setProp()'
assert old in s, 'm170'
p.write_text(s.replace(old, '\td = d.stopPropagation()'))
