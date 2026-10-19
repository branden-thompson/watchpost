import pathlib
# 0.19.0 D-239: Offline is said as NOAA not reached, though NOAA may have answered.
# Killed by TestEverySnapshotsStateIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\twords = append(words, "nothing came back from NOAA: built from what is held") // D-239: never more than happened\n'
assert s.count(old) == 1, 'm511'
p.write_text(s.replace(old, '\t\twords = append(words, "NOAA not reached: the last field")\n', 1))
