import pathlib
# 0.19.0 FR-5.4: the diagnostics record each live offset.
# Killed by TestStatusShowsTheLiveAndTypicalOffset.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\t\td.problem(propOffsetWords(snap.Offset))'
assert old in s, 'm284'
p.write_text(s.replace(old, '\t\t\t_ = snap.Offset'))
