import pathlib
# 0.19.0 FR-5.2 (R2C-10, R2L-5): readings refused whole as implausible are never noted in the diagnostics.
# Killed by TestEveryRefusedReadingIsNoted.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\tif snap.Inputs.Implausible > 0 && snap.Inputs.Implausible != was.Implausible {\n'
assert s.count(old) == 1, 'm560'
p.write_text(s.replace(old, '\t\tif snap.Inputs.Implausible > 1<<30 {\n', 1))
