import pathlib
# 0.19.3 D-366: a reset or 39 clears the ink.
# Killed by TestInkAtReadsTheCellsForeground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\t\tink, inked = color.RGBA{}, false\n'
assert s.count(old) == 1, 'm646'
p.write_text(s.replace(old, '\t\t\t// passed over\n', 1))
