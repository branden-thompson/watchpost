import pathlib
# 0.19.0 D-242: recording caches the day's documents for readers when it reads a day.
# Killed by TestRecordingHoldsNoDocument.
p = pathlib.Path('platform/history/history.go'); s = p.read_text()
old = '\t\tx.hours = marksOf(s.readDay(f, d, day, entries))\n'
assert s.count(old) == 1, 'm514'
p.write_text(s.replace(old, '\t\tx.hours = marksOf(s.heldDayOf(f, d, day).entries)\n', 1))
