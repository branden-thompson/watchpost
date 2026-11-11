import pathlib
# 0.19.5 Q5.3 (DR-6, A-4, RR-7): a window rounds to the minute, so thirty seconds is said as a minute.
# Killed by TestTheAlertWindowIsSpokenAsTheRecordShowsIt.
p = pathlib.Path('domains/severe/record.go'); s = p.read_text()
old = '\td = d.Round(time.Minute)\n\tif d >= 24*time.Hour {\n'
assert s.count(old) == 1, 'm761'
p.write_text(s.replace(old, '\td = d.Truncate(time.Minute)\n\tif d >= 24*time.Hour {\n', 1))
