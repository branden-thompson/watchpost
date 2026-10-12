import pathlib
# 0.19.0 FR-1.13: a new snapshot drops the hours ahead of the old.
# Killed by TestANewSnapshotMissesTheAnswerMemo.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = "\t\td.mapPane.propAheadHours, d.mapPane.propAheadAsked = nil, 0 // the hours ahead are the old snapshot's\n"
assert old in s, 'm213'
p.write_text(s.replace(old, '\t\t_ = 0\n'))
