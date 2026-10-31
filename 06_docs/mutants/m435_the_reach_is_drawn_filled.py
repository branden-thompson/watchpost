import pathlib
# 0.19.0 go-tuiMaps rc.2: the reach is asked for as its edge alone (Type.Edge).
# Killed by TestYourFrequencyDrawsItsReach.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = 'tuimaps.Type{Breaks: []float64{0.5}, Edge: true}'
assert old in s, 'm435'
p.write_text(s.replace(old, 'tuimaps.Type{Breaks: []float64{0.5}}'))
