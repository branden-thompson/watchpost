import pathlib
# 0.19.3 D-366: a background's values are passed over, never taken for the ink.
# Killed by TestInkAtReadsTheCellsForeground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\t\tp.extended(ink, inked)\n'
assert s.count(old) == 1, 'm647'
p.write_text(s.replace(old, '\t\t\tink, inked = p.extended(ink, inked)\n', 1))
