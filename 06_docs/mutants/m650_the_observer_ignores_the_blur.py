import pathlib
# 0.19.3 D-367: the Observer takes the terminal's blur.
# Killed by TestTheCursorWearsTheColourItCovers.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = '\t\t_, d.blurred = msg.(tea.BlurMsg)\n'
assert s.count(old) == 1, 'm650'
p.write_text(s.replace(old, '\t\t_, _ = msg.(tea.BlurMsg)\n', 1))
