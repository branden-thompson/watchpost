import pathlib
# 0.19.0 FR-5.2 (R2C-10): readings whose M(3000)F2 alone was refused are never noted in the diagnostics.
# Killed by TestEveryRefusedReadingIsNoted.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\tif snap.Inputs.ImplausibleM > 0 && snap.Inputs.ImplausibleM != was.ImplausibleM {\n'
assert s.count(old) == 1, 'm561'
p.write_text(s.replace(old, '\t\tif snap.Inputs.ImplausibleM > 1<<30 {\n', 1))
