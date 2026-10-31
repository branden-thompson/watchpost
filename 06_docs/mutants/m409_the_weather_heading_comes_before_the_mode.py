import pathlib
# 0.19.0 D-210: the mode and its answers before the place's weather and the night.
# Killed by TestTheCentreAnswerIsSaid.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = '\t\treturn append(d.propWords(loc.Lat, loc.Lon), spokenAll(out)...)'
assert old in s, 'm409'
p.write_text(s.replace(old, '\t\treturn append(spokenAll(out), d.propWords(loc.Lat, loc.Lon)...)'))
