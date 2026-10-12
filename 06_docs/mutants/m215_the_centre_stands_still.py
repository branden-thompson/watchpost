import pathlib
# 0.19.0 FR-1.11: the centre is answered again as the map pans.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = '\tstep := func(v float64) float64 { return math.Round(v/propCentreStep) * propCentreStep }\n'
assert old in s, 'm215'
p.write_text(s.replace(old, '\tstep := func(v float64) float64 { return 0 * v }\n'))
