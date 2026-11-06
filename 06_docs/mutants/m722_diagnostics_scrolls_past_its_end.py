import pathlib
# 0.19.5 Q2.13 (RN minor): Diagnostics stops scrolling at its last line.
# Killed by TestDiagnosticsScrollsItsProse.
p = pathlib.Path('modes/tty/debug.go'); s = p.read_text()
old = '\t\t\td.modalScroll = min(d.modalScroll+1, d.footerModalScrollMax(d.opts()))\n'
assert s.count(old) == 1, 'm722'
p.write_text(s.replace(old, '\t\t\td.modalScroll = d.modalScroll + 1\n', 1))
