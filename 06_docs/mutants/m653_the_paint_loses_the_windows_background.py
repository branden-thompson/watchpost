import pathlib
# 0.19.3 D-367: the painted cell's background is set back after it.
# Killed by TestPaintCellSetsOnlyThatCellsBackground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\tback = paperSGR(c.Paper)\n'
assert s.count(old) == 1, 'm653'
p.write_text(s.replace(old, '\t\tback = "\\x1b[49m"\n', 1))
