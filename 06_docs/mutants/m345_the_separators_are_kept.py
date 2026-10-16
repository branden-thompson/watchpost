import pathlib
# 0.19.0 FR-3.2: the windows' separators are said as pauses.
# Killed by TestSpokenWritesTheFiguresOut.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t{regexp.MustCompile(` · `), ", "},\n'
assert old in s, 'm345'
p.write_text(s.replace(old, ''))
