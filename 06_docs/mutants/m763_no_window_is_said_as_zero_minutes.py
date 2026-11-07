import pathlib
# 0.19.5 Q5.3 (DR-6, A-4, RR-7): an alert with no window has none said.
# Killed by TestTheAlertWindowIsSpokenAsTheRecordShowsIt.
p = pathlib.Path('app/severe_read.go'); s = p.read_text()
old = '\tif window <= 0 {\n\t\treturn ""\n\t}\n'
assert s.count(old) == 1, 'm763'
p.write_text(s.replace(old, '\tif window < 0 {\n\t\treturn ""\n\t}\n', 1))
