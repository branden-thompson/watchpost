import pathlib
# 0.19.0 D-242: a write leaves the day's documents held for readers, though they are no longer its files.
# Killed by TestRecordingHoldsNoDocument.
p = pathlib.Path('platform/history/cache.go'); s = p.read_text()
old = '\ts.days.Forget(f.day)\n\tif had < -1 {'
assert s.count(old) == 1, 'm513'
p.write_text(s.replace(old, '\tif had < -1 {', 1))
