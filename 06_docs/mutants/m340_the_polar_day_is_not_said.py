import pathlib
# 0.19.0 FR-3.1: where the sun neither rises nor sets, that is said.
# Killed by TestTheDayAndItsTurnAreSaid.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\treturn words + ", and the sun neither rises nor sets there today."\n'
assert old in s, 'm340'
p.write_text(s.replace(old, '\t\treturn words + "."\n'))
