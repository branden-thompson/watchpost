import pathlib
# 0.19.0 R-2.5 (D-73): a disturbed band has its own glyph.
# Killed by TestADisturbedBandIsDrawnAndNamed.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tif s.Status == ionomaps.Disturbed {\n'
assert old in s, 'm221'
p.write_text(s.replace(old, '\t\tif s.Status == ionomaps.Disturbed && false {\n'))
