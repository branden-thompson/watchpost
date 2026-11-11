import pathlib
# 0.19.5 Q3.4 (PF-4): the one-pass finisher pads each row to the terminal's width.
# Killed by TestTheConsolesFinisherIsByteIdentical.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = '\t\tfor range max(0, b.width-render.Width(l)) {'
assert s.count(old) == 1, 'm727'
p.write_text(s.replace(old, '\t\tfor range max(0, b.width-1-render.Width(l)) {', 1))
