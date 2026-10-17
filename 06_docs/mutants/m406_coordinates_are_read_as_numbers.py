import pathlib
# 0.19.0 AX-2: a typed place's coordinates are said as degrees.
# Killed by TestSpokenWritesTheFiguresOut.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\treturn hemisphere(p[1], "north", "south") + " " + hemisphere(p[2], "east", "west")\n'
assert old in s, 'm406'
p.write_text(s.replace(old, '\t\treturn m + p[0][:0]\n'))
