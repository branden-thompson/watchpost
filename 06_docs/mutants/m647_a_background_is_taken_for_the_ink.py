import pathlib
# 0.19.3 D-366: a background's values set the paper, never the ink.
# Killed by TestInkAtReadsTheCellsForeground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\t\tpen.Paper, pen.Papered = p.extended(pen.Paper, pen.Papered)\n'
assert s.count(old) == 1, 'm647'
p.write_text(s.replace(old, '\t\t\tpen.Ink, pen.Inked = p.extended(pen.Ink, pen.Inked)\n', 1))
