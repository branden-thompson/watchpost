import pathlib
# 0.19.0 BUILD red team round 2 (FR-5.2, BL2-3): a snapshot GIRO left paused says so.
# Killed by TestEverySnapshotsStateIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase ionomaps.GIROPaused:\n'
assert old in s, 'm473'
p.write_text(s.replace(old, '\tcase ionomaps.Reason(250):\n', 1))
