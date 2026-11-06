import pathlib
# 0.19.5 Q3.6 (PF-11): the one-layout overlay still lays Diagnostics' confirmation
# over the window.
# Killed by TestTheConsolesWindowIsTheSameLaidOnce.
p = pathlib.Path('modes/tty/debug.go'); s = p.read_text()
old = '\tif box != "" {\n\t\tbase = render.Overlay(base, box, termWidth)\n'
assert s.count(old) == 1, 'm728'
p.write_text(s.replace(old, '\tif box != "" && false {\n\t\tbase = render.Overlay(base, box, termWidth)\n', 1))
