import pathlib
# 0.19.0 FR-1.18: NOAA's outlook of a storm to come is named.
# Killed by TestARaisedScaleIsNamed.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = '\t\tif !raised(day.G) {\n'
assert old in s, 'm354'
p.write_text(s.replace(old, '\t\tif true {\n'))
