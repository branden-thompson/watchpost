import pathlib
# 0.19.3 D-366: a wide glyph's second cell is its own.
# Killed by TestInkAtReadsTheCellsForeground.
p = pathlib.Path('platform/render/ink.go'); s = p.read_text()
old = '\t\tif cell < at+w {\n'
assert s.count(old) == 1, 'm648'
p.write_text(s.replace(old, '\t\tif cell == at {\n', 1))
