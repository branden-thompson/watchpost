import pathlib
# 0.19.0 D-198: each chart row starts with its shade.
# Killed by TestEachRowShowsItsNearestStation.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = 'propShade(a.Nearest, o.ASCII) + " " + render.PadTo'
assert old in s, 'm277'
p.write_text(s.replace(old, '"  " + render.PadTo'))
