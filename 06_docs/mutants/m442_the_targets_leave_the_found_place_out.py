import pathlib
# 0.19.0 D-212: the chart's targets are the continents, then the place found.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\treturn append(append([]propanswers.Target(nil), propanswers.Continents...), k.found)\n'
assert old in s, 'm442'
p.write_text(s.replace(old, '\treturn propanswers.Continents\n'))
