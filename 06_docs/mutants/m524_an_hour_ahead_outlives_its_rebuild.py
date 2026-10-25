import pathlib
# 0.19.0 RC-4 (L-F3): an hour ahead asked before a rebuild is kept for the rebuilt snapshot.
# Killed by TestAnHourAheadOfARebuiltSnapshotIsDropped.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = ' || v.made != d.propMadeOf(*d.mapPane.propSnap) {'
assert s.count(old) == 1, 'm524'
p.write_text(s.replace(old, ' || v.made.computed != d.mapPane.propSnap.Computed {', 1))
