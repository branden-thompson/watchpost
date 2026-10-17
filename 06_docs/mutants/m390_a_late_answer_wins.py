import pathlib
# 0.19.0 FR-1.10: an earlier find's answer arriving late is dropped.
# Killed by TestFocusReturnsAfterAFind.
p = pathlib.Path('modes/tty/map_prop_find.go'); s = p.read_text()
old = '\tif v.seq != d.mapPane.propFindSeq {\n'
assert old in s, 'm390'
p.write_text(s.replace(old, '\tif v.seq != v.seq {\n'))
