import pathlib
# 0.19.5 Q5.2 (DR-5): the severe window's rows and records share one clock,
# which names the zone the time is in.
# Killed by TestTheSevereRowsTimesReadInThePlacesZone.
p = pathlib.Path('domains/severe/record.go'); s = p.read_text()
old = '\treturn t.In(in).Format("01/02 15:04 MST")\n'
assert s.count(old) == 1, 'm759'
p.write_text(s.replace(old, '\treturn t.In(in).Format("01/02 15:04")\n', 1))
