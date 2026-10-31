import pathlib
# 0.19.0 FR-5.2 (R3L-7): a burst's refused count is noted again for every snapshot of the hour, crowding the diagnostics' ring.
# Killed by TestEveryRefusedReadingIsNoted.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\tif snap.Inputs.Implausible > 0 && snap.Inputs.Implausible != was.Implausible {\n'
assert s.count(old) == 1, 'm564'
p.write_text(s.replace(old, '\t\tif snap.Inputs.Implausible > 0 {\n', 1))
