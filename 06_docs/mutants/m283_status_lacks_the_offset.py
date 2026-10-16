import pathlib
# 0.19.0 FR-5.4: the Status window says the live offset beside its typical.
# Killed by TestStatusShowsTheLiveAndTypicalOffset.
p = pathlib.Path('modes/tty/status_sources.go'); s = p.read_text()
old = '\t\twords = append(words, propOffsetWords(s.Offset)) // FR-5.4'
assert old in s, 'm283'
p.write_text(s.replace(old, '\t\t_ = s // FR-5.4'))
