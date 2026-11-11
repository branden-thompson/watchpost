import pathlib
# 0.19.5 Q3.3 (PF-3): at the narrow breakpoint the full player is the two-row one, and the plan counts it so.
# Killed by TestTheRadiosPlannedHeightIsItsRows.
p = pathlib.Path('modes/tty/radio_panel.go'); s = p.read_text()
old = '\tif !compact && !p.twoRow {\n\t\tn = 2'
assert s.count(old) == 1, 'm735'
p.write_text(s.replace(old, '\tif !compact {\n\t\tn = 2', 1))
