import pathlib
# 0.19.0 FR-1.10 (0.18.0 D-124): a failed find says why, with the way on.
# Killed by TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\t\td.mapPane.propFindErr = v.err.Error()\n'
assert old in s, 'm391'
p.write_text(s.replace(old, '\t\t_ = v.err\n'))
