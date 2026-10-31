import pathlib
# 0.19.0 D-192: an absorbed band is drawn ○.
# Killed by TestAClosedBandIsDrawnByItsKind.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\t\t\tg = render.Tint(d.mark(propAbsorbedGlyph)'
assert old in s, 'm226'
p.write_text(s.replace(old, '\t\t\tg = render.Tint(d.mark(propClosedGlyph)'))
