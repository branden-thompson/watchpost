import pathlib
# 0.19.5 Q5.3 (DR-6, A-4, RR-7): a window runs from the onset when the onset follows the issue time.
# Killed by TestTheAlertWindowIsSpokenAsTheRecordShowsIt.
p = pathlib.Path('domains/severe/record.go'); s = p.read_text()
old = '\twindow := until.Sub(from)\n'
assert s.count(old) == 1, 'm762'
p.write_text(s.replace(old, '\twindow := until.Sub(at)\n', 1))
