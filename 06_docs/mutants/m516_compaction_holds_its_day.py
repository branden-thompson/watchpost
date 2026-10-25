import pathlib
# 0.19.0 D-242: compaction and roll-up hold the day they rewrite, pushing out the days a map replays.
# Killed by TestADayIsCompactedWithoutLosingARecord.
p = pathlib.Path('platform/history/cache.go'); s = p.read_text()
old = '\treturn s.readDay(f, d, day, entries)\n}\n\n// heldDayOf'
assert s.count(old) == 1, 'm516'
p.write_text(s.replace(old, '\theld := heldDay{files: now, entries: s.readDay(f, d, day, entries)}\n\ts.days.Put(f.day, held)\n\treturn held.entries\n}\n\n// heldDayOf', 1))
