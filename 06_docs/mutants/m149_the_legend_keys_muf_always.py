import pathlib
# 0.19.0 FR-1.5: the legend row keys the layer drawn.
# Killed by TestBothPropagationLayersDrawWithTheirUnits.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\treturn d.presetRow("fof2", "foF2, MHz │ ", "LOWER ", " HIGHER", width)'
assert old in s, 'm149'
p.write_text(s.replace(old, '\t\treturn d.presetRow("muf", "MUF(3000), MHz │ ", "LOWER ", " HIGHER", width)'))
