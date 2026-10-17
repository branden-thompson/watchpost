import pathlib
# 0.19.0 FR-3.2: every frequency's figure carries its unit.
# Killed by TestSpokenWritesTheFiguresOut.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tout := p[1] + " " + p[2] + " megahertz"\n'
assert old in s, 'm344'
p.write_text(s.replace(old, '\t\tout := p[1] + " " + p[2]\n'))
