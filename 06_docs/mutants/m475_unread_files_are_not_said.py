import pathlib
# 0.19.0 BUILD red team round 2 (FR-5.2, CN2-5): NOAA's files that could not be read are said.
# Killed by TestEverySnapshotsStateIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif len(s.Inputs.Unread) > 0 {'
assert old in s, 'm475'
p.write_text(s.replace(old, '\tif false {', 1))
