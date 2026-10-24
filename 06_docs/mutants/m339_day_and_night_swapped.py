import pathlib
# 0.19.0 FR-3.1 (AX-E): day or night at the origin.
# Killed by TestTheDayAndItsTurnAreSaid.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif day.Day {\n'
assert old in s, 'm339'
p.write_text(s.replace(old, '\tif !day.Day {\n'))
