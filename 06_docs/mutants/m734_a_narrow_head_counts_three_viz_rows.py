import pathlib
# 0.19.5 Q3.3 (PF-3): a full player whose head takes its short form draws one visualiser row, and the plan counts one.
# Killed by TestTheRadiosPlannedHeightIsItsRows.
p = pathlib.Path('modes/tty/radio_panel.go'); s = p.read_text()
old = '\tcase compact || p.twoRow || p.narrow:\n'
assert s.count(old) == 1, 'm734'
p.write_text(s.replace(old, '\tcase compact || p.twoRow:\n', 1))
