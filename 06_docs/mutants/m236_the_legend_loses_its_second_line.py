import pathlib
# 0.19.0 D-192: the legend names above the upper limit and no data.
# Killed by TestAClosedBandIsDrawnByItsKind.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\tstring(propClosedGlyph)+" above the upper limit   blank: no data   figures in MHz",'
assert old in s, 'm236'
p.write_text(s.replace(old, '\t\t"",'))
