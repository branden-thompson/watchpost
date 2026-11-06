import pathlib
# 0.19.5 D-26 (DEF-7): tab leaves the reports from any row.
# Killed by TestTheRequestWindowsKeysDoWhatTheyClaim.
p = pathlib.Path('modes/tty/request.go'); s = p.read_text()
old = '\tcase requestReports:\n\t\tst.field = requestPosition\n'
assert s.count(old) == 1, 'm658'
p.write_text(s.replace(old, '\tcase requestReports:\n\t\tst.field = requestReports\n', 1))
