import pathlib
# 0.19.3 D-367: the terminal is asked for focus reports.
# Killed by TestTheRouterAsksForFocusReports.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = '\tv.ReportFocus = true '
assert s.count(old) == 1, 'm652'
p.write_text(s.replace(old, '\tv.ReportFocus = false ', 1))
