import pathlib
# 0.19.0 BUILD red team round 2 (D-214, BL2-3): no stations by the listener's choice is said as a choice.
# Killed by TestEverySnapshotsStateIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase s.Inputs.NoStations:\n'
assert old in s, 'm474'
p.write_text(s.replace(old, '\tcase false:\n', 1))
