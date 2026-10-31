import pathlib
# 0.19.0 AX-2: the figure after "under" is a frequency, given its unit.
# Killed by TestSpokenWritesTheFiguresOut.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '|best about|under|up to) '
assert old in s, 'm405'
p.write_text(s.replace(old, '|best about|up to) '))
