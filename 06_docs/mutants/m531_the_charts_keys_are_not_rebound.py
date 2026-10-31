import pathlib
# 0.19.0 D-256 (FR-1.14): a [keys] entry for the chart's targets is not taken.
# Killed by TestTheChartsTargetKeysAreRebindable.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tif _, ok := defaults[act]; ok {\n\t\t\town[act] = b'
assert s.count(old) == 1, 'm531'
p.write_text(s.replace(old, '\t\tif _, ok := defaults[act]; ok {\n\t\t\t_ = b', 1))
