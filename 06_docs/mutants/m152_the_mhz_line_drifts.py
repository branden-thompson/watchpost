import pathlib
# D-160: each band's MHz stands under it.
# Killed by TestThePropagationLegendIsBandOverMHz.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\tedges.WriteString(render.PadTo(b.mhz, each))'
assert old in s, 'm152'
p.write_text(s.replace(old, '\t\tedges.WriteString(render.PadTo(b.mhz, each+1))'))
