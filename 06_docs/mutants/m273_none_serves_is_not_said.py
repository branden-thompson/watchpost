import pathlib
# 0.19.0 D-197: where no band fits, none serves.
# Killed by TestTheBestFrequencyIsNamed.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '\t\treturn "none serves"\n'
assert old in s, 'm273'
p.write_text(s.replace(old, '\t\treturn ""\n'))
