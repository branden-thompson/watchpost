import pathlib
# 0.19.0 FR-5.1: the climatology alone is said, not an age.
# Killed by TestTheAgeIsTheOldestInputs.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\treturn "climatology", 0, false\n'
assert old in s, 'm327'
p.write_text(s.replace(old, '\t\treturn "data just in", 0, false\n'))
