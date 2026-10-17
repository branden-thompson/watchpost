import pathlib
# 0.19.0 L-F3: a snapshot rebuilt at the same moment is answered again.
# Killed by TestARebuiltSnapshotIsAnsweredAgain.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\td.mapPane.propBuilt++ // a snapshot rebuilt'
assert old in s, 'm430'
p.write_text(s.replace(old, '\t\t_ = 0 // a snapshot rebuilt'))
