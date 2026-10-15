import pathlib
# 0.19.0 D-194: the Map window is 85% of the terminal's width.
# Killed by TestTheMapWindowLeavesTheDashboardInSight.
p = pathlib.Path('modes/tty/view.go'); s = p.read_text()
old = 'const mapWindowPct = 85'
assert old in s, 'm248'
p.write_text(s.replace(old, 'const mapWindowPct = 80'))
