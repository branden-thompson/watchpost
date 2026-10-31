import pathlib
# 0.19.0 L-F3: a snapshot rebuilt at the same moment is answered again.
# Killed by TestARebuildAndBackIsAnsweredAgain (R2C-1: TestARebuiltSnapshotIsAnsweredAgain flips the options, which tell the snapshots apart without the build).
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\t\td.mapPane.propBuilt++ '
assert old in s, 'm430'
p.write_text(s.replace(old, '\t\t\t_ = 0 '))
